# ADR-0003 — SQLite as the store

**Status:** Accepted · **Date:** 2026-09-02

## Context

This number was reserved rather than exported, for the same reason as
[ADR-0002](0002-go-and-the-runtime-stack.md): the store is the second-largest single piece of the
*HOW*, and inheriting it would decide this implementation's shape before it had argued for one
([PROVENANCE.md](../../PROVENANCE.md)).

Three things constrain the choice before any preference gets a vote.

**The query surface is relational.** [005](../../specs/005-item-query-api/spec.md) is seventeen
endpoints of filtering, sorting, paging and aggregation. `TotalRecordCount` is the count *before*
paging, so every list is two queries. The by-name endpoints group and count. And
[behaviours §3.6](../compatibility/behaviours.md#36-ties-are-engine-resolved-and-paging-the-artist-sorts-loses-rows--class-b-diverged)
is a **divergence Atrium already owes**: the reference's ordering is resolved by its engine and is
not total, so paging a large audio library there shows some items twice and never shows others.
Atrium's ordering must be total — the requested keys, then `Name`, then the id — which is a
requirement on whatever does the sorting.

**The deployment shape is already decided.** One process, no second service, one data directory
([architecture §5](../architecture.md#5-deployment-shape)). Whatever this is, it is embedded.

**cgo is not available.** ADR-0002 chose `CGO_ENABLED=0` for static builds and cross-compilation,
and the usual Go SQLite driver needs cgo. So the question is not only *SQLite or not* but *is there
a pure-Go SQLite that can carry this*.

### The observation that halves the problem

**Item identity is derived from the path, not stored as a sequence.** Principle VII, measured in
[§1.4](../compatibility/behaviours.md#14-item-identifiers-are-32-lowercase-hex-characters), and
keyed by this project on the path *relative to its library root*
([003 §3.6](../../specs/003-library-configuration-and-scanning/spec.md)).

So the scanned library is **reconstructible**: delete every item row, rescan, and the same files
produce the same identifiers. What is *not* reconstructible is what a user did — favourites, played
state, resume positions, accounts, tokens, playlists. Half of this database is a cache and the
other half is the only copy, and they are not the same kind of thing.

### What was measured, on 2026-09-02

Three drivers, one schema, the same queries: 40,000 items, WAL, one connection, on Go 1.27.0
`darwin/arm64`.

| Driver | Ingest 40k | List + count | Search `LIKE` | By-name | Binary | Cold build |
|---|---|---|---|---|---|---|
| `modernc.org/sqlite` (pure Go) | 418 ms | **2.93 ms** | 12.8 ms | 8.6 ms | 9.5 MiB | 7 s |
| `ncruces/go-sqlite3` (wasm) | 400 ms | 3.32 ms | 12.9 ms | 9.4 ms | 14.2 MiB | 10 s |
| `mattn/go-sqlite3` (cgo) | 291 ms | 1.27 ms | 6.1 ms | 4.1 ms | 6.7 MiB | — |

`[measurement: three SQLite drivers, Go 1.27.0, 2026-09-02]`

**This table is where ADR-0002's price is paid, and it is the honest number for it: about 2.3× on
the hot read path.** In absolute terms a page of 100 items with its `TotalRecordCount` in front
costs 2.93 ms over a 40,000-item library, and a client issues a handful of those per screen. The
search row is a `LIKE '%…%'` that no index rescues on any of the three; that is a query shape, not a
driver.

Two further questions were measured rather than assumed:

- **FTS5 is present in the pure-Go build, and matches non-ASCII.** `CREATE VIRTUAL TABLE … USING
  fts5` succeeds and `MATCH 'años'` returns the row. `[measurement: modernc.org/sqlite v1.58.0, Go 1.27.0, 2026-09-02]`
- **A reader is not blocked while a scan writes.** With WAL and separate handles, **57,664 reads
  completed** during one write transaction inserting 30,000 rows, worst read latency **393 µs**.
  `[measurement: modernc.org/sqlite v1.58.0, Go 1.27.0, 2026-09-02]`

## Decision

### SQLite, embedded, one file in the data directory

Opened with **WAL**, `synchronous=NORMAL`, foreign keys on, and a busy timeout. **One writer handle
and a pool of readers** — which is what the measurement above says is safe, and what makes an
incremental scan compatible with serving requests.

**A scan writes in batched transactions**, not one transaction for the whole tree. The measurement
shows a reader survives a long write; it does not show that a write held open for the length of a
library scan is a good idea, and a batch that commits is also a scan that can resume.

Choosing what the reference chose has a second benefit worth naming: where the two servers differ
on ordering or on a count, the difference is **ours** rather than an artefact of two different query
engines. That is one fewer explanation to rule out in a differential report.

### `modernc.org/sqlite`, pure Go

Smaller binary, faster cold build, and the faster of the two pure-Go drivers on the path that
matters. `CGO_ENABLED=0` holds.

### Hand-written SQL over `database/sql`. No ORM, no code generation

The queries in 005 **are** the feature. They must be readable in the form they execute, because the
thing being reviewed is which index the planner will use and whether the ordering is total. A layer
that composes SQL out of Go is a layer between the reviewer and the contract.

The cost is scanning rows by hand, which is repetitive and unclever, and that is an acceptable price
for the queries staying legible.

### Sort keys are computed at write time and compared as bytes

**Ordering may never depend on the engine's collation.** SQLite's `NOCASE` is ASCII-only, which
would silently mis-order every non-ASCII title in a library — and §3.6 is precisely a divergence
about ordering, so an ordering this project cannot fully explain is the last thing it needs.

So [§2.6](../compatibility/behaviours.md)'s `SortName` derivation runs at **write** time into a
stored sort-key column, and every `ORDER BY` compares that column with the default `BINARY`
collation, ending in the id. The order is then a property of what this project wrote, not of what
the engine happened to do.

### The store is split into a derived half and a precious half

| Half | Holds | On a schema change | Backed up |
|---|---|---|---|
| **Derived** | The scanned library: items, media streams, images, the by-name aggregates | **Dropped and rescanned.** No migration is written. | Not worth it |
| **Precious** | Users, credentials, tokens and sessions, favourites, played state, resume positions, playlists, library configuration | **Migrated**, forward-only, never destructively | Yes — it is the only copy |

**The two halves carry separate schema versions**, and a derived-version mismatch at startup is a
rescan rather than an error.

**No foreign key points from the precious half into the derived half.** User data references an item
by its **derived identifier string**, not by a row id — which is exactly what Principle VII bought:
rebuild the whole derived half and every favourite still points at the right item, because the
identifier is a function of the path and not of an insertion order. This is the constraint that
makes the split work at all, and it is why it is stated as a rule rather than left to a schema
review.

### Time is stored in the wire's unit

Ticks are `INTEGER`, and so are dates — a count of 100-nanosecond intervals, because
[§1.2](../compatibility/behaviours.md#12-dates-carry-up-to-seven-fractional-digits)'s seven
fractional digits **are** tick resolution. Storing a date as text or as milliseconds would round
away the seventh digit somewhere between the disk and the wire, and §1.3 already says the conversion
happens once, at ingestion.

## Consequences

- **Every list costs two queries**, because `TotalRecordCount` is the count before paging. That is
  the contract, not an inefficiency to optimise away, and the measurement above times both together
  for that reason.
- **Search is the slow shape and has a known answer.** `LIKE '%…%'` scans on every driver. FTS5 is
  present and works on non-ASCII, so if `SearchTerm` becomes a bottleneck the remedy exists — but it
  is [005](../../specs/005-item-query-api/spec.md)'s plan to reach for, with a measurement, not
  something this record pre-decides.
- **A rescan is now a legitimate repair.** Corruption in the derived half, a schema bump, a bug in a
  scanner — all answered by dropping and rescanning, with no user-visible loss. That is a genuinely
  cheaper operational story than one where every table is precious.
- **The store stays a port** ([architecture §6](../architecture.md#6-state-and-the-store-boundary)).
  This record implements the interfaces the domain declares; it does not remove them, and no SQL
  appears above that line.
- **Backup is one file**, and the precious half is what a user would lose. A backup command is not
  in v1 and is not blocked by anything here.
- **The 2.3× is a standing debt, and this is what would reopen it.** If a real library at a real
  size makes the list path visible to a user, the options in order are: fix the query or the index;
  move `SearchTerm` to FTS5; and only then reconsider cgo — which would supersede ADR-0002 and cost
  static builds. A profile comes first, not a preference.

### What is not verified, and is owed

- **The measurements are one machine, one schema, one shape of library.** 40,000 rows of synthetic
  data on an Apple-silicon laptop is a comparison between drivers, which is what it was for. It is
  **not** a claim about how this server behaves on a real library on a small host. ⚠️ UNVERIFIED
- **Nothing here has been measured under a concurrent scan and a real request mix.** The reader test
  held one writer and one reader loop. ⚠️ UNVERIFIED
- **`modernc.org/sqlite` is a transpilation of SQLite's C sources, not SQLite.** It passes SQLite's
  own test suite upstream, and this project has verified none of that itself. The first behaviour
  that differs from the reference in a way that traces to the engine is the thing to watch for.
  ⚠️ UNVERIFIED

## Alternatives rejected

**`ncruces/go-sqlite3`, real SQLite compiled to WebAssembly.** This was the closest call, and its
argument is a good one: it runs SQLite's actual sources under `wazero`, so its semantics track
upstream *by construction* rather than by a transpiler being correct — which is not nothing in a
project that compares bytes and has just written an ⚠️ UNVERIFIED about exactly that. It loses on
measured cost: 4.7 MiB more binary in a project whose deployment story is one static file, a slower
cold build, and 13% on the hot path. **What would flip it** is the third unverified item above
becoming a real bug rather than a caution.

**`mattn/go-sqlite3` with cgo.** 2.3× faster and the smallest binary. It supersedes ADR-0002's
`CGO_ENABLED=0` and takes static builds and cross-compilation with it, and it puts a C toolchain on
every contributor's machine — to make a 2.93 ms query into a 1.27 ms one, on a server whose
expensive work is reading media files and running ffmpeg.

**An embedded key-value store — bbolt.** Pure Go with no C and no WebAssembly anywhere, ACID, and
faster than any of the above for reading by key. It is rejected on 005: every filter, every ordering
and every aggregation becomes an index this project maintains by hand, and the ordering has to be
total across all of them. That is writing a query engine in order to avoid depending on one, and the
result would be less reviewable than SQL, not more.

**Keeping the library in memory and persisting only user data.** Genuinely tempting, because the
derived/precious split above is most of the way there and the identity derivation makes the library
reconstructible. It fails on two counts: a library big enough to be interesting is a library whose
index does not want to be resident, and **every restart becomes a full rescan** — where 003's whole
point is that a scan is incremental, which needs the previous scan's state on disk to compare
against.

**An external SQL server.** More capable than anything here needs. It contradicts
[architecture §5](../architecture.md#5-deployment-shape) — one process, no second service — and
turns installing Atrium into administering two things. It would have to supersede that decision
rather than sit beside it.

**An ORM (GORM, ent) or generated queries (sqlc).** An ORM puts a query builder between the reviewer
and the SQL, in a project where row order is contract and the planner's choice of index is the thing
under review. `sqlc` is the better of the two — SQL stays the source of truth and the scanning
boilerplate goes away — and it is rejected for the same reason ADR-0002 rejected generated
marshallers: it adds a build tool and a directory of generated files to pay for boilerplate nobody
has measured as a problem. If hand-scanning rows becomes the thing slowing changes down, `sqlc` is
the first thing to revisit.

**One migration lineage for the whole database.** Simpler to reason about, and it means writing a
migration for tables a rescan would have rebuilt in a minute — and, worse, it invites a migration
that touches user data in order to fix a library table. Splitting them makes the dangerous half
small, explicit, and the only half anybody has to be careful with.
