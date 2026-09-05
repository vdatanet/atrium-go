---
feature: 003-library-configuration-and-scanning
title: Library configuration and scanning — implementation plan
status: Accepted
created: 2026-09-05
updated: 2026-09-05
spec_status_required: Accepted
amended: 2026-09-05 by the change that wrote tasks.md, in four places - section 6.5's count of how many guards the specification states, section 6.8's count of 001's assertions the runner change costs, section 8.2's home for the forty-seven declarations and who owns twenty-five of them, and section 8.4's criterion table, which gains AC-16
---

# 003 — Implementation plan

> **This document describes HOW.** It may not restate WHAT: the spec is the authority on behaviour,
> and a plan that repeats it will disagree with it eventually.

**On the gate.** The template asks for a spec at `Accepted` or better. 003's spec says
`Implemented`, which is [a statement about the exporting project](../../PROVENANCE.md) — *the WHAT
is settled and was proven once, elsewhere* — and 001's plan recorded that reading first, 002's
second. It is taken again here, with 002's addition: **writing this plan amended the spec**, in
three places, and that does not reopen the gate. The loop in [specs/README.md](../README.md) closes
deliberately, and §11 lists the three with what forced each.

~~This plan is `In review`. It becomes `Accepted` when that review returns and a task list is asked
for, which is what `tasks.md`'s own `plan_status_required` gates.~~ **Taken 2026-09-05**: a task
list was asked for, so this plan is `Accepted` and [tasks.md](tasks.md) is `In review` behind it.
Both earlier plans recorded the same transition in the same place. **Writing the task list amended
this plan in four places and the specification in two more**, and
[tasks.md's own gate record](tasks.md#what-the-gate-changed) says what forced each; the loop closing
is again why that does not reopen the gate behind it.

**On three anchors this file has to honour.** Three sentences in
[behaviours.md](../../docs/compatibility/behaviours.md) already cite sections of a `plan.md` at this
path that did not exist — two of PROVENANCE's *"links with nothing to point at"*, and writing this
file makes them resolve:

| Citation | Anchor | Where it is cited |
|---|---|---|
| *"003 plan §6.4"* | `#64-change-detection` | [behaviours §2.17](../../docs/compatibility/behaviours.md#217-no-item-and-no-media-source-carries-a-modification-time), on the `(size, modification time)` signal |
| *"003 plan §6.4"* | none | [behaviours §5.6](../../docs/compatibility/behaviours.md#56-a-default-rescan-does-not-notice-a-replaced-poster), on the poster a default rescan misses |
| *"003 plan §6.5"* | `#65-the-guard-against-a-mass-delete` | [behaviours §5.2](../../docs/compatibility/behaviours.md#52-a-container-that-has-lost-every-file-is-not-removed), on *"none of the three guards"* |

**So §6.4 is change detection and §6.5 is the guard against a mass delete, deliberately**, and §6.5
carries **three** guards because the sentence citing it counts three. Nothing is repointed and
nothing is deleted ([AGENTS.md §4](../../AGENTS.md)). The same warning 002's plan gave applies: the
*sentences* beside those citations were written about the exporting project's plan at those numbers,
and what is written under them here is this project's.

## 1. Approach

**003 is no endpoints and the whole library, and the absence of endpoints is the organising
problem rather than a saving.**

Everything this feature produces becomes observable through 005, and every mistake it makes becomes
a wrong answer there (spec §1). That has two consequences which shape the whole plan.

**The first is that Principle VIII has no boundary to assert at.** 001 and 002 both leaned on
`conformance/` starting the real binary and issuing requests, and 001's closing audit found *twice*
that a criterion proven one layer in proves less than its name. 003 cannot use that instrument:
[surface.yaml](../../docs/compatibility/surface.yaml) has zero rows for this feature, and
`conformance/` speaks HTTP. **Inventing a route to test would be Principle VI's plausible-looking
stub with a test attached.** §8 says what takes the instrument's place, how much weaker it is, and —
the part that matters — which of this feature's claims become observable only at 005. That is a real
deferral of proof and it is stated as plainly as [002 §8.3](../002-authentication-users-and-sessions/plan.md#83-l3-is-deferred-and-this-is-the-feature-where-that-costs-the-most)
stated its deferred L3.

**The second is that the scan is the most destructive code in the project and nothing above it will
notice a mistake.** A wrong identifier discards a user's favourites silently; a wrong removal takes
their library away; a wrong sort key reorders every screen. So the design decision that organises
everything below is this: **a scan is a pure function from a reading of a tree to a desired set of
items, and a separate reconciliation of that set against what the store holds.** Neither half
touches the other's inputs. The resolution half is a function over paths and needs no store, no
clock and no server; the reconciliation half is a function over two item sets and needs no
filesystem. Both are therefore table-driven over a fixture that is a *declaration* rather than a
recording, and the guards of §6.5 live in the seam between them, where a partial reading can be
refused before anything is written.

**The second decision follows from ADR-0003 and is the one this feature is first to need.** The
store is split into a derived half a rescan rebuilds and a precious half that is the only copy, and
**nothing has used the split yet**: 002 filed `0002_users_and_sessions.sql` in the precious lineage
and left the derived version a literal `0`, with a note that 003's first derived migration must
change that line deliberately. §4 takes the split, §6.8 replaces the refusal 002 left behind, and
the interesting part is not that a scan's output is derived — it is which parts of what this feature
stores are *not*.

**Three readings changed this plan before it was written**, all at the pinned tag, and each is
argued where it lands. None of the three is a measurement, so each is registered rather than allowed
to move a specification on its own ([AGENTS.md §1.3](../../AGENTS.md)):

1. **The reference's `.ignore` rule is three rules and spec §3.2 states one of them.** The marker is
   searched for **up the directory tree** rather than in the containing directory, an empty or
   whitespace-only one excludes everything beneath it, and a **non-empty** one is a set of
   `.gitignore`-style patterns of which only the matches are excluded
   `[source: Emby.Server.Implementations/Library/DotIgnoreIgnoreRule.cs:18-30,41-68,95-131 @ v10.11.11]`.
   §6.1, [U-42](../../docs/compatibility/reference-target.md), and the spec gains a note.
2. **The multi-part marker vocabulary is five words and two number forms, and the bare `-a`/`-b`
   form spec §3.3 names is not one of them.** The stacking rules take
   `cd`, `dvd`, `part`, `pt`, `disc` or `disk`, followed by digits or by a single letter `a`–`d`
   `[source: Emby.Naming/Common/NamingOptions.cs:141-145 @ v10.11.11]` — so `The Film - a.mkv` is
   not a part of anything there, and `The Film - cda.mkv` is. §6.2, [U-43](../../docs/compatibility/reference-target.md),
   and the spec's parenthetical is corrected.
3. **`EnableCaseSensitiveItemIds` defaults to `true` and is a property of the server, not of a
   library** `[source: MediaBrowser.Model/Configuration/ServerConfiguration.cs:89 @ v10.11.11]`,
   read at `[source: Emby.Server.Implementations/Library/LibraryManager.cs:650 @ v10.11.11]`. That
   is spec §7's OQ-9 answered by reading rather than by asking, and the row stays open because the
   running server is the tie-breaker and there is none here. It does not move §3.6's decision — the
   spec states Atrium's own default and never claimed to match one — but it turns *"unmeasured"*
   into *"a known difference in the direction we chose"*. §6.3, [U-44](../../docs/compatibility/reference-target.md).

## 2. Inherited decisions

| Decision | Source |
|---|---|
| Go, `chi` over `net/http`, `encoding/json` behind `internal/wire`, no cgo; optional fields are pointers and `omitempty` on a non-pointer is banned | [ADR-0002](../../docs/decisions/0002-go-and-the-runtime-stack.md) |
| Embedded SQLite, `modernc.org/sqlite`, hand-written SQL over `database/sql`, one file, one writer handle and a pool of readers, **a scan writes in batched transactions** | [ADR-0003](../../docs/decisions/0003-sqlite-as-the-store.md) |
| The store is split into a derived half that is dropped and rescanned and a precious half that is migrated; **the two carry separate schema versions**; **no reference points from the precious half into the derived half** | ADR-0003, [architecture §6](../../docs/architecture.md#6-state-and-the-store-boundary) |
| **Sort keys are computed at write time into a stored column and compared as bytes**, with `BINARY` collation, never by the engine's `NOCASE` | ADR-0003 |
| Ticks are `INTEGER`, and so are dates — the wire's unit is the stored unit | ADR-0003, [behaviours §1.3](../../docs/compatibility/behaviours.md#13-durations-and-positions-are-net-ticks) |
| Four layers, one direction; the domain imports no HTTP; ports import nothing of ours but the unit types | [architecture §2](../../docs/architecture.md#2-layers-and-the-direction-of-dependency) |
| No map iteration reaches a response body, and anything ordered is sorted on a stated key | architecture §2 |
| `internal/` for everything but `cmd` and `conformance`; **`conformance/` imports nothing of ours**, enforced by `tools/check_conformance_imports`; `tools/` holds Go programs | [architecture §3](../../docs/architecture.md#3-repository-layout) |
| One process, one data directory; the store is not disposable; graceful shutdown is load-bearing | [architecture §5](../../docs/architecture.md#5-deployment-shape) |
| Identity is derived, never a sequence; **Atrium keys on the path relative to its library root**, and reproducing the reference's identifier bytes is not a goal | architecture §6, [behaviours §1.4](../../docs/compatibility/behaviours.md#14-item-identifiers-are-32-lowercase-hex-characters) |
| Process settings are few, come from flags with an environment fallback, and are not a feature; **library configuration lives in the data directory and its shape is 003's** | [architecture §9](../../docs/architecture.md#9-configuration-identity-and-logging) |
| Two fixture worlds; tests that reach a real binary carry a build tag, and the suite staying green without it is the check that they all do | [architecture §8](../../docs/architecture.md#8-testing-and-conformance) |
| A subcommand of the same binary is how an operator creates state `conformance/` needs, and the password-shaped rule that goes with it | [002 plan §6.9](../002-authentication-users-and-sessions/plan.md#69-provisioning-and-the-three-seats-a-run-needs) |
| A feature the server serves any row of must serve every row of it — the rule both halves of the L0 check derive *implemented* from | [001 plan §8.5](../001-server-identity-and-discovery/plan.md#85-routes-against-surfaceyaml) |

**Deviations: one, and it is a deviation from a *plan* rather than from an ADR or from
architecture.md, so no new ADR is owed.** [001 plan §4](../001-server-identity-and-discovery/plan.md#4-data-model)
recorded, at its T3, that *"the first feature with a derived table adds a file; it does not also
change the runner"*. It does change the runner, and §6.8 argues why: the derived half's policy is
*drop and rebuild*, not *apply the tail*, and a runner that treats both halves as forward-only
lineages cannot express ADR-0003's central claim. The ADR is not deviated from — it is implemented
for the first time. The expectation that the runner would be untouched was 001's, reasonable when
written, and wrong; recording that is cheaper than quietly making it true.

## 3. Modules

| Module | Change | Responsibility |
|---|---|---|
| `internal/library` | new | The domain everything downstream reads: collection types and their extension lists, the resolution rules for the three types, naming, identity derivation and path normalisation, and the two sort-name derivations. A function over paths and names. Imports no HTTP, no store and no filesystem. |
| `internal/scan` | new | The act: the walk, the change-detection signal, the reconciliation of a reading against the previous scan, the three guards, the batching, and the progress and summary of spec §3.8. Imports `internal/library` and `internal/ports`. |
| `internal/ports` | extended | `LibraryStore` and `ItemStore` — the precious and the derived halves of what this feature keeps. |
| `internal/store/sqlite` | extended | One migration in the **precious** lineage, the **first derived schema** and the generation that governs it, the rebuild that replaces 002's refusal, and the readers and writers behind the two new ports. |
| `internal/app` | extended | The `library` subcommand (§6.7), the scheduled scan, and the start-time rebuild-and-rescan of §6.8. |
| `cmd/atrium` | extended | One more arm on the dispatch it already has. Nothing else. |
| `internal/libraryfixture` | new | The declaration of the fixture tree of [conformance §L2](../../docs/compatibility/conformance.md), as a Go value and a builder that writes it into a directory. §8.5. |
| `tools/build_library_fixture` | new | The same declaration as a program, so `conformance/` can have the tree without importing it. §8.5. |
| `conformance/` | extended | Exactly what a package that speaks HTTP can prove about a feature with no routes, which is less than it looks and is not nothing. §8.1. |

**`internal/httpapi` is untouched, and that is the sentence a reader should check first.** This
feature adds no handler, no refusal shape, no model and no route. Adding one would fail the L0
registration check against a `surface.yaml` that has no row for it, which is the check working.

**Why `library` and `scan` are two packages.** The split is the approach in §1 made structural.
`internal/library` is read by features that never scan anything — 004 merges metadata over the names
and numbers this feature derives, 005 orders every list by the sort key it computes, and both need
the *rules* without needing the walker — while `internal/scan` is called by exactly two callers, the
subcommand and the scheduler. A single package would make every importer of the sort-name derivation
reach a package that opens files, and would put the one function 005's ordering depends on behind the
one that can delete a library. The dependency runs one way and never back.

**Why there is no filesystem port, although architecture §2 makes the store, the prober and the
clock ports.** The walker takes an `fs.FS` and uses `io/fs`, which is the standard library's own
seam and imports nothing of ours. Three reasons it is not a port:

- **The fixture is a real tree anyway.** [conformance §L2](../../docs/compatibility/conformance.md)
  describes a world of *"paths and filler bytes"* whose whole value is that a scanner meets real
  directory entries, and the reference's recorded reading of it
  ([reference-fixture-reading.json](../../docs/compatibility/reference-fixture-reading.json)) was
  taken by mounting that tree into a container. A synthetic filesystem would be a second thing to
  keep in step with the tree the comparison is against.
- **The signal cannot be faked honestly.** §6.4's signal is the file's size and modification time,
  and [behaviours §2.17](../../docs/compatibility/behaviours.md#217-no-item-and-no-media-source-carries-a-modification-time)
  measures that neither is observable on the wire — so a fake filesystem would be inventing the one
  input whose correctness nothing downstream can check.
- **A port buys a test double, and the tests that want one want something else.** What the
  reconciliation tests need is two item sets, and those are values; what the walk tests need is a
  tree, and `t.TempDir()` is one.

The one thing this costs is named in §9: a test that wants an unreadable root has to make one, and
`os.Chmod` does not make a root unreadable for `root`. AC-12's test therefore uses a path that does
not exist as well as one whose permissions were removed, and skips the second when it turns out to
be readable anyway.

**Why the fixture's declaration is under `internal/` and its builder under `tools/`.** Two consumers
need the tree and one of them may import nothing of ours. `conformance/` runs
`go run ./tools/build_library_fixture -into <dir>` as a subprocess, exactly as it already runs
`go build` for the server binary — a subprocess is not an import, and
`tools/check_conformance_imports` reads `go list -deps` rather than a process tree. So there is one
declaration and two ways to reach it, which is the same trade
[001 plan §8.5](../001-server-identity-and-discovery/plan.md#85-routes-against-surfaceyaml) already
makes for `surface.yaml`: two readers of one document cannot agree by construction.

## 4. Data model

This is the first feature to write both halves of ADR-0003's split, and the boundary between them is
where the argument is. It is not *"a scan's output is derived and everything else is precious"*: two
things this feature stores are neither obviously.

### 4.1 The precious half — migration `0003_libraries.sql`

Forward-only, numbered contiguously in the lineage 001's runner already applies.

**`libraries`** — one row per configured library.

| Column | Type | Note |
|---|---|---|
| `id` | TEXT PRIMARY KEY | 32 lowercase hex, **allocated** and never derived — spec §3.6, and §6.3 below |
| `name` | TEXT NOT NULL | As the operator spelled it. Editable |
| `name_folded` | TEXT NOT NULL UNIQUE | **A query-pattern column**, and the same shape 002's `username_folded` is: the subcommand addresses a library by name, and uniqueness has to be the database's rule rather than a convention |
| `collection_type` | TEXT NOT NULL CHECK (…) | `movies`, `tvshows` or `music`. **Frozen after creation** (spec §3.6) |
| `case_sensitive` | INTEGER NOT NULL | Whether paths compare with regard to case when an identifier is derived. **Frozen after creation** (spec §3.6) |
| `created_at` | INTEGER NOT NULL | Ticks |

**`library_roots`** — one row per configured root, `PRIMARY KEY (library_id, ordinal)`.

| Column | Type | Note |
|---|---|---|
| `library_id` | TEXT NOT NULL REFERENCES `libraries(id)` | |
| `ordinal` | INTEGER NOT NULL | The order the operator gave them, which decides nothing but keeps a list stable (architecture §2 forbids an order derived from anything else) |
| `path` | TEXT NOT NULL | Absolute, as configured |

**All of it is precious, and the reason is `id` rather than the row.** Spec §3.6 makes a library's
identity **allocated** and kept, so that renaming a library or moving its roots costs nothing — and
says the consequence plainly: deleting a library and declaring another with the same name and roots
is not the same library. An allocated identifier is by definition not reconstructible from the
files, so `libraries` cannot live in a half that is dropped and rebuilt. Two columns beside it are
worse than the identifier if they are lost:

- **`case_sensitive` decides every identifier under the library** (§6.3). Losing it and defaulting
  would silently rewrite them all, which is the failure Principle VII exists to prevent, applied to
  a whole library at once.
- **`collection_type` decides which resolution rules apply**, so losing it changes every item's
  *type* as well as its identifier.

That is also why both are refused after creation rather than accepted with a warning: an edit that
rewrites every identifier under a library has no undo, because nothing stores the old ones.

### 4.2 The derived half — schema generation 1, `derived/library.sql`

**`items`** — one row per item, of every type, including the containers that back no file.

| Column | Type | Note |
|---|---|---|
| `id` | TEXT PRIMARY KEY | 32 lowercase hex, derived — §6.3 |
| `library_id` | TEXT NOT NULL | The library's identifier **as a string**, not a foreign key — see below |
| `parent_id` | TEXT NULL | The container's `id`. NULL for a library's own root row |
| `type` | TEXT NOT NULL | `CollectionFolder`, `Movie`, `Series`, `Season`, `Episode`, `MusicArtist`, `MusicAlbum`, `Audio` |
| `name` | TEXT NOT NULL | What the path or the tags said. 004 may replace it, and §9 records that fight |
| `sort_key` | TEXT NOT NULL | §6.6. **A query-pattern column in the strongest sense**: it exists so that `ORDER BY` never calls a function or a collation |
| `path` | TEXT NULL | Relative to the root, in the normalised form §6.3 derives the identifier from. NULL for an inferred container that has no directory |
| `root_ordinal` | INTEGER NULL | Which root the path is relative to |
| `index_number`, `parent_index_number`, `index_number_end` | INTEGER NULL | Episode, season, track and disc numbers, and the second number of a multi-episode file |
| `production_year` | INTEGER NULL | The year §3.3 strips out of a name |
| `premiere_date` | INTEGER NULL | Ticks, for the date-named episode of §3.4 |
| `unplaceable` | INTEGER NOT NULL | Whether the name said too little to place the item — the counter spec §3.8 requires be reported apart from a skip |

**`item_files`** — one row per file behind an item, `PRIMARY KEY (item_id, ordinal)`.

| Column | Type | Note |
|---|---|---|
| `item_id` | TEXT NOT NULL REFERENCES `items(id)` ON DELETE CASCADE | |
| `ordinal` | INTEGER NOT NULL | Part order for a multi-part film (spec §3.3), `0` for everything else |
| `path` | TEXT NOT NULL | Relative to the root |
| `size` | INTEGER NOT NULL | Bytes. **Observable**, because a media source carries `Size` ([behaviours §2.17](../../docs/compatibility/behaviours.md#217-no-item-and-no-media-source-carries-a-modification-time)) |
| `modified_at` | INTEGER NOT NULL | Ticks. **Not observable anywhere**, and stored only as half of §6.4's signal |

A table of its own rather than a column on `items` for one reason: a multi-part film is **one** item
with two sources in order (spec §3.3), and 008 will read exactly this table to answer
`MediaSources`. A path column on `items` with a second one beside it for part two is the shape that
makes the third part a migration.

**`scan_state`** — one row per library, the claim and the last summary. §6.7 and §6.9.

| Column | Type | Note |
|---|---|---|
| `library_id` | TEXT PRIMARY KEY | |
| `claimed_at`, `claimed_by` | INTEGER NULL, TEXT NULL | Which process is scanning, and since when. §6.9 |
| `last_scan_at`, `last_scan_full` | INTEGER NULL, INTEGER NULL | |
| `summary_document` | TEXT NULL | The counts of spec §3.8, as JSON |

### 4.3 Which of these is derived, and the two that are not obvious

| Table | Half | Why |
|---|---|---|
| `libraries`, `library_roots` | **Precious** | The identifier is allocated and the two frozen columns decide every identifier below them. Not reconstructible from anything on disk |
| `items` | Derived, in the plain sense | A rescan of the same tree produces the same rows, identifiers included. That is exactly what ADR-0003's *"the observation that halves the problem"* is about |
| `item_files.size`, `item_files.modified_at` | **Derived, and this is the interesting one** | They look like a cache of the filesystem and they are the *only* record of what the last scan saw. Dropping them is nevertheless free: a rescan re-reads both from the files, and a scan with no previous reading is a full scan, which is a correct answer rather than a degraded one |
| `scan_state` | **Derived, decided rather than obvious** | See below |

**`scan_state` is derived, and the argument is worth writing down because the other answer is
tempting.** *"When this library was last scanned"* reads like operator-facing history, and history
is precious. It is not, for one reason: a library that has never been scanned and a library whose
derived half was just dropped are **the same state**, and they have to be, because §6.8 answers a
generation mismatch by dropping the derived half and scanning every library from nothing. Putting
the last-scan instant in the precious half would leave a row saying *"scanned yesterday"* over an
empty item table, and every reader of that pair would have to decide which half to believe. One
half, one answer.

**And two things this feature deliberately does not store.**

- **The extension lists of spec §3.2 are code, not configuration.** They are a measured contract
  ([behaviours §2.15](../../docs/compatibility/behaviours.md#215-an-audio-file-under-a-video-root-is-not-an-item)),
  not an operator preference, and a per-library override would be a configuration surface with no
  named consumer (Principle VI). They are a constant per collection type in `internal/library`.
- **The three sort-name lists of spec §3.7 are constants too, with the measured defaults**, and the
  spec's *"Atrium exposes them with the same defaults and honours them the same way"* is read as
  *implements the same three configurable lists*, not as *routes a way to change them*: no v1
  endpoint reads or writes server configuration. **The rule that goes with the day one becomes
  editable is stated now, because it is not obvious**: changing an article, a character or the pad
  width invalidates **every stored sort key in the database**, so it is a derived-generation bump
  (§6.8) and not a setting. A build that let an operator edit the list without one would reorder
  half a library and leave the other half in the old order, permanently, with nothing failing.

**No foreign key points from `items` into `libraries`.** `library_id` is the identifier string, and
that is architecture §6's rule about the two halves applied where it actually bites: SQLite's
`foreign_keys(1)` is on, so a real constraint from a derived table to a precious one would refuse
the drop that §6.8 performs at start. The same rule is what makes 007's favourites and 009's
playlists survive a rescan — they will name an item by the derived identifier string — and this is
the first table on the derived side of it.

## 5. Contracts

```
// internal/ports — declared by the domain, implemented by the store

Library struct {                 // the precious record
    ID             string
    Name           string
    NameFolded     string
    CollectionType string
    CaseSensitive  bool
    Roots          []string      // in ordinal order
    CreatedAt      units.Time
}

LibraryStore interface {
    CreateLibrary(ctx, Library) error
    Libraries(ctx) ([]Library, error)         // ordered by NameFolded, then ID
    LibraryByFoldedName(ctx, folded string) (Library, bool, error)
    RenameLibrary(ctx, id, name, folded string) error
    ReplaceRoots(ctx, id string, roots []string) error
    RemoveLibrary(ctx, id string) error
}

ScannedItem struct {             // the derived record, and the unit both halves speak
    ID, LibraryID, ParentID string
    Type, Name, SortKey     string
    Path                    string
    RootOrdinal             int
    IndexNumber, ParentIndexNumber, IndexNumberEnd *int
    ProductionYear                                 *int
    PremiereDate                                   *units.Time
    Unplaceable                                    bool
    Files                                          []ScannedFile
}

ScannedFile struct {
    Ordinal    int
    Path       string
    Size       int64
    ModifiedAt units.Time
}

ItemStore interface {
    ItemsForLibrary(ctx, libraryID string) ([]ScannedItem, error)
    ApplyScanBatch(ctx, batch ScanBatch) error       // one transaction, §6.9
    RemoveItems(ctx, ids []string) error
    ClaimScan(ctx, libraryID, by string, at units.Time, staleAfter units.Ticks) (bool, error)
    ReleaseScan(ctx, libraryID string, at units.Time, summary []byte, full bool) error
    RebuildDerived(ctx) error                        // §6.8; the drop and recreate
}
```

The four record types live in `internal/ports`, which is **T4's decision in 002 applied rather than
retaken**: a port method returning `library.Item` would make the bottom of architecture §2's diagram
import a domain package. The cost that decision carried there — the policy crossing as bytes — has
no analogue here, because a `ScannedItem` is already flat values and a `units.Time`.

**`ClaimScan` returns a boolean rather than an error for a library already being scanned.** Two
scanners over one store is a state this feature creates on purpose (§6.7: an operator may run
`atrium library scan` against a data directory a server is serving from), and *"somebody else is
scanning"* is an outcome the caller reports, not a fault. `staleAfter` is what breaks a claim left
by a process that died; §6.9 argues the value.

```
// internal/library — no os, no net/http, no ports in any signature

CollectionType string                      // Movies | Shows | Music, from the three spec §3.1 names
func (CollectionType) Admits(ext string) bool

Normalise(path string, caseSensitive bool) (string, error)   // §6.3; refuses absolute and climbing
DeriveID(libraryID string, kind Kind, key string) string      // §6.3

Reading struct { Root int; Entries []Entry }   // what a walk saw, sorted
Entry  struct { Path string; Size int64; ModifiedAt units.Time }

Resolve(lib Library, readings []Reading) (Plan, error)        // §6.2; pure
Plan   struct { Items []ports.ScannedItem; Unplaceable, Skipped []Note }

SortKeyBase(name string) string                               // §6.6, spec §3.7.1
SortKeyFor(item *ports.ScannedItem)                           // §6.6, spec §3.7.2 and §3.7.3
```

**`Resolve` takes every root's reading at once and returns the whole library's items**, rather than
resolving a path at a time. That is not convenience: three of spec §3's rules cannot be decided from
one path. A directory holding several different titles is a category rather than a film (§3.3); a
multi-part film is one item only once its siblings have been seen (§3.3); and an album's identity
comes from the album artist across all of its tracks (§3.5). A per-path resolver would have to
discover each of those by mutating what it had already returned, which is the shape that makes a
scan's answer depend on the order the directory entries arrived in — and Principle VII forbids
exactly that.

**`Resolve` is deterministic and sorts its own inputs.** The walk yields entries in whatever order
the filesystem gives, so the reading is sorted on the path before anything looks at it, and every
map inside the resolver is emptied into a slice and sorted before it leaves. This is architecture
§2's *"no map iteration may reach a response body"* one layer earlier than a response: the identifier
is a function of the path and is safe, but the **parent-child assignment** of an inferred container
and the **part order** of a multi-part film are not, and both reach a body at 005.

```
// internal/scan

Changes struct { Added, Updated, Removed []string; Examined, Skipped, Unplaceable int }

Reconcile(previous, desired []ports.ScannedItem, full bool) (Changes, []ports.ScannedItem, []string)
```

**`Reconcile` is a pure function over two item sets and is where every removal in this project is
decided.** It takes no store, no filesystem and no clock. `full` is spec §3.8's re-examination,
which changes only whether an unchanged signal is believed. Its output is the batch to write and the
identifiers to remove, and the guards of §6.5 run on the reading **before** it is called, so a
partial reading never reaches it.

## 6. Algorithms

### 6.1 The walk, and what it refuses to look at

One walk per root, over `fs.FS` from `os.DirFS(root)`, depth-first, collecting an `Entry` for every
candidate file. Spec §3.2's exclusions, and where each is decided:

| Rule | Where | Note |
|---|---|---|
| Extension not on the collection type's list | The walk, per file | §3.2's measured lists, and **no fallback between types** ([behaviours §2.15](../../docs/compatibility/behaviours.md#215-an-audio-file-under-a-video-root-is-not-an-item)) |
| Any path component beginning with `.` | The walk, per component | A matching directory is not descended into, which is the difference between skipping a file and skipping a tree |
| A directory carrying a `.ignore` marker | The walk, per directory | See below |
| Trailer, sample and extra suffixes, and the extras folder names | The walk, per file and per directory | §3.4: v1 **ignores** them rather than attaching them |
| Zero-byte files | The walk, per file | And **the reference does not do this** — see below |
| A file whose size changed between two passes | §6.4, not here | It needs two readings, and the walk has one |

**The last row is the one that moves.** Spec §3.2 lists *"files being written, detected by size
change between two passes"* among the ignore rules, which reads as a property of a file. It is a
property of a **pair of scans**, so it is decided in §6.4 where both readings exist: a file whose
recorded size and modification time both moved since the last scan is re-read as an update, and a
file examined twice **inside one scan** is not something a single walk performs. What v1 does is
therefore narrower than the row suggests, and it is narrower in the direction that costs an operator
nothing: a half-copied file becomes an item with the wrong size, and the next scan corrects it,
where a scanner that walked every tree twice would double the cost of every scan to catch a file
that is being written at that instant.

**The zero-byte rule is a measured divergence, not a shared behaviour.** The reference makes an item
out of a zero-byte film; Atrium does not, and that is one of the forty-seven differences
[010's AC-2](../010-conformance-harness/spec.md) counts over this feature's own fixture
`[probe: tools/probe_reference_scan.py, Jellyfin 10.11.11, 2026-09-02]`. It is spec §3.2's rule
working as written, it is in [§3.0.3](../../docs/compatibility/behaviours.md#303-the-shape-of-a-safe-divergence)'s
safe direction only in the weak sense — an item is missing rather than extra — and §8.2 is where the
declaration lives so that the difference fails the day it *stops* being true.

**`.ignore`, which is three rules at the reference and one and a half here.** The reference searches
for the marker from the file's own directory **upwards, to the filesystem root**; an empty or
whitespace-only marker excludes everything beneath the directory holding it; and a non-empty one is
a set of `.gitignore`-style patterns of which only the matching paths are excluded — with the
fallback that a file whose every pattern fails to parse excludes everything
`[source: Emby.Server.Implementations/Library/DotIgnoreIgnoreRule.cs:18-30,41-68,95-131 @ v10.11.11]`.
v1 implements:

- **The empty marker**, which is the case spec §3.2 names and the case
  [conformance §L2](../../docs/compatibility/conformance.md) puts in the fixture — *"a directory
  excluded by an **empty** `.ignore` marker, which is the only kind that excludes anything"*.
- **The ancestor search, bounded at the library root.** A marker in a parent directory has to
  exclude its subtree or an operator who wrote one at the top of a season directory finds it
  excluded the top directory only. It is **not** searched above the root, deliberately: the
  reference walks to the filesystem root, so a stray `.ignore` in a home directory empties every
  library beneath it, and that is a foot-gun rather than a feature. Diverging costs a request
  nothing — nobody sends one — and it fails in the direction that shows more rather than less.
- **Not the patterns.** A non-empty `.ignore` excludes **nothing** here. Implementing
  `.gitignore` semantics means a matcher this project would own for a feature no measurement shows
  anybody using, and getting it subtly wrong excludes files an operator expects to see. The cost is
  stated: on a tree carrying a non-empty marker, Atrium has items the reference does not.

All three halves of that are a **source reading and not a measurement**, so the row is
[U-42](../../docs/compatibility/reference-target.md) and the spec gains a note rather than a new
rule.

### 6.2 Resolution, and the three shapes that need siblings

`Resolve` runs in three passes over the sorted reading, and the passes exist because of the three
rules that cannot be decided from one path (§5).

1. **Classify.** Each entry becomes a *candidate* with the fields its own path yields: for a video,
   a cleaned title and a year; for an episode, a season and one or two episode numbers, matched
   against the **filename first and then the parent directory** (spec §3.4 — a series called `24`
   must not have its title read as an episode number); for a track, a disc and a track number.
2. **Group.** Multi-part films are folded into one candidate with ordered parts; a multi-episode
   file keeps one candidate with two numbers; tracks are grouped into albums by album artist and
   album, and albums into artists (spec §3.5).
3. **Place.** Containers are created for the groups, seasons are inferred where no directory exists,
   and every candidate is given its parent.

Three decisions inside that, each of which the spec constrains and none of which it decides:

**The marker vocabulary for a multi-part film is the reference's, and it is wider than spec §3.3's
parenthetical.** The stacking rules take `cd`, `dvd`, `part`, `pt`, `disc` or `disk`, separated from
the title by a space, underscore, dot or hyphen or by a closing bracket, and followed either by
digits or by a **single letter `a`–`d` after the same word**
`[source: Emby.Naming/Common/NamingOptions.cs:141-145 @ v10.11.11]`. The spec's *"and the `-a`/`-b`
form"* describes a bare trailing letter, which the reference does **not** stack — `The Film - a.mkv`
and `The Film - b.mkv` are two films there. Implementing the spec's reading would merge two items the
reference keeps apart, which is the *"doubles a user's library"* failure §3.3 warns about, run
backwards and therefore worse: two films become one and one of them disappears. So the vocabulary is
the source's, the spec's parenthetical is corrected (§11), and the reading is registered as
[U-43](../../docs/compatibility/reference-target.md) because no probe here has sent either shape.

**A directory naming several different titles is a category, and the test for it is the group's
own.** Spec §3.3 makes the directory name a film when the directory holds one film, and says the
only part a single path cannot decide is a directory holding several. Pass 2 is where that becomes
decidable: after grouping, a directory holding exactly one video candidate takes its directory's
cleaned name (the measured 1,087 against 457 `[read: Jellyfin 10.11.11, 2026-08-27]`), and a
directory holding more than one leaves every candidate with its own filename-derived name. A
multi-part film is one candidate by then, so `The Long Film (1998)/…part1.mkv` and `…part2.mkv` take
the directory's name — which is what the reference's own reading of this repository's fixture shows,
naming that item `The Long Film (1998)` and not `The Long Film (1998) - part1`
`[probe: tools/probe_reference_scan.py, Jellyfin 10.11.11, 2026-09-02]`.

**Music asks 004 and does not read tags itself.** Spec §3.5 makes embedded tags outrank the path and
says in terms that reading them is 004's and that the ordering of the conversation is a plan
concern. The ordering: `Resolve` produces the path's answer for every track, and takes an optional
`TagSource` — an interface `internal/library` declares and 004 implements — consulted **once per
file, before grouping**, because the album artist decides which album a track belongs to and
grouping cannot be redone afterwards. v1 ships a `TagSource` that answers nothing, so this feature's
own tests exercise the path fallback and 004's exercise precedence. Two consequences:

- **The compilation rule needs the tags to have been asked.** With no tag source, `Various Artists`
  is attributed only where the directory names it, which is what the fixture's
  `Various Artists/A Compilation (1999)` gives. With one, spec §3.5's rule applies.
- **Spec §3.5's fallback for the track number, the disc number and the title is Atrium's own and is
  a known divergence** ([behaviours §2.16](../../docs/compatibility/behaviours.md#216-a-music-tracks-number-comes-from-tags-never-from-its-filename),
  spec's OQ-8). It is implemented as the spec writes it, including the tie-break that reads an
  ambiguous name as saying **less**: a leading digit is a track number only when a separator follows
  it, so `24K Magic.flac` is a song called `24K Magic`.

### 6.3 Identity, and the normalisation the whole feature rests on

`DeriveID(libraryID, kind, key)` is the first 16 bytes of SHA-256 over the library identifier, a NUL,
the item kind, a NUL, and the normalised key, rendered as 32 lowercase hexadecimal characters —
[behaviours §1.4](../../docs/compatibility/behaviours.md#14-item-identifiers-are-32-lowercase-hex-characters)'s
shape, and the same construction 002 uses for a session identifier
([002 plan §6.5](../002-authentication-users-and-sessions/plan.md)). The key per kind is spec §3.6's
table.

Four things this design decides, and each is a trap somebody has already fallen into.

**The library identifier is in the key, and the root path is not.** Spec §3.6 keys on *"the library
root plus the normalised name"* for a `Series`, `MusicAlbum` and `MusicArtist`, and on the path
relative to the root for the file-backed types. Reading *"the library root"* as the root's **path**
would reintroduce, one level down, the exact trap §3.6 spends four paragraphs escaping: an operator
who moves a library from `/mnt/a` to `/mnt/b` keeps every `Movie` identifier and loses every
`Series` one. It is read as the library's allocated identity instead, which is the value §3.6 makes
stable across a rename and a remount — and which is why §4 puts that identity in the precious half.
This is [001 plan §4](../001-server-identity-and-discovery/plan.md#4-data-model)'s argument against
deriving the *server's* identity from the data directory's path, at the scale it was originally made
at.

**The kind is in the key, so a directory and the item it backs cannot collide.** A `Series` at
`Shows/The Series` and a `Season` inferred with no directory of its own would otherwise be able to
derive the same string.

**Normalisation is three steps and refuses two inputs.** Separators to `/`, the text to Unicode NFC,
and case folded when the library is not case-sensitive — the spec's three, in that order, because
folding before normalising the form gives a different answer for a decomposed capital. A key that is
**absolute** or that **climbs above its root** is an error rather than a normalisation, and the error
travels: it fails the library's scan under §6.5's third guard rather than skipping one file, because
a caller holding a path it believes is relative and is not has computed the wrong root, not the
wrong file.

**Case-insensitive by default is a divergence from the reference's own default, and it is now known
to be one.** `EnableCaseSensitiveItemIds` defaults to `true`
`[source: MediaBrowser.Model/Configuration/ServerConfiguration.cs:89 @ v10.11.11]` and is consulted
as a server-wide setting `[source: Emby.Server.Implementations/Library/LibraryManager.cs:650 @ v10.11.11]`.
Spec §3.6 chose the opposite default deliberately and per library. Nothing about the *identifiers*
is observable — behaviours §1.4 already establishes that the two servers' bytes differ either way —
but **the item count is**: two files differing only in case are two items on a stock reference and
one here, and the fixture holds such a pair. That difference is one this project owns and declares
(§8.2), the spec's OQ-9 records the source reading and stays open, and the row is
[U-44](../../docs/compatibility/reference-target.md).

### 6.4 Change detection

The signal is `(size, modification time)` per file, read from the walk and compared against
`item_files`. It is the plan half of
[behaviours §2.17](../../docs/compatibility/behaviours.md#217-no-item-and-no-media-source-carries-a-modification-time),
whose measurement is the licence for it: **no item and no media source carries a modification time
on the wire**, so which signal a server uses to decide *whether to look* creates no delta whatever
it is. The size does travel — a media source carries `Size` — so an item whose file was replaced by
one of a different length must end up with the new number, and that is the assertion, not the
signal.

| Reading | Decision |
|---|---|
| A path with no previous row | Add. Ancestors are created as needed |
| Size or modification time moved | Update, **keeping the identifier**, because the identifier is a function of the path and the path did not move |
| Both unchanged | Believe it, unless the scan is a full re-examination |
| A previous row with no path in the reading | Remove, subject to §6.5 |

**Three things this table does not say and the code has to.**

**The stored modification time is a whole tick and the comparison is exact.** `units.Time` rounds to
a tick (100 nanoseconds) and a filesystem may report a coarser or finer resolution than that. So the
*stored* value is what the last scan read through the same conversion, and both sides of every
comparison have been through it — comparing a freshly-stated `time.Time` against a stored tick would
report a change on the first rescan of every file on a filesystem whose resolution is not a
multiple of a tick.

**An identifier is re-derived on every scan and compared, never assumed.** The row is found by path,
and the identifier is recomputed and checked against the stored one. They can only differ if the
derivation changed or the library's `case_sensitive` moved, both of which are supposed to be
impossible — which is exactly why it is worth one comparison to find out. A mismatch fails the
library's scan; it does not rewrite the row, because rewriting it is the silent discard Principle VII
exists to prevent.

**A default scan does not notice a poster.** Artwork beside a film is re-read only on a scan that
reads the directory, and a default scan reads it only when the item's own media file changed. That
is [behaviours §5.6](../../docs/compatibility/behaviours.md#56-a-default-rescan-does-not-notice-a-replaced-poster)'s
accepted gap, and it is **measured on both halves**: replacing `poster.jpg` beside an untouched film
and rescanning changes the reference's image tag and the bytes behind it
`[probe: tools/differential.py --named replaced-poster-default-rescan, Jellyfin 10.11.11, 2026-09-02]`.
The escape hatch is the full re-examination spec §3.8 requires, and widening the default signal means
stat-ing dozens of candidate artwork names per item per scan — a cost somebody should measure before
paying. This section is what that entry cites, and this paragraph is what it cites it for.

### 6.5 The guard against a mass delete

*"Treating an unmounted share as 'every item was deleted' is the single most destructive thing a
scanner can do"* (spec §3.8). Three guards, each answering a different way for that to happen, and
they run on the **reading**, before `Reconcile` is called.

1. **A root that cannot be read is a failed scan, not an empty one.** The root must resolve to a
   readable directory before the walk starts, and **any** error during the walk — a permission
   refused deep in the tree, an I/O error, a path that will not normalise — fails the scan for that
   library and writes nothing. This is spec §3.8's rule and it is ~~the only one of the three~~
   **one of the two** the specification states.
2. **A root that reads as holding no candidate file, where the previous scan recorded at least one,
   is treated as unavailable.** This is the guard the first one misses: a share that mounts as an
   empty directory is perfectly readable. Zero is the threshold *because* it needs no number — it is
   the mount failure's signature, where "fewer than before" is a judgement nobody could defend and
   a library an operator really emptied is a deliberate act. The scan refuses and says which root;
   `atrium library scan --allow-empty-root` is how an operator says they meant it.
   **Amended 2026-09-05, while the task list was written: this guard is spec §3.8's rule now, and
   AC-16 is its criterion.** It was a plan invention, which is what the corrected count above records
   — writing a task per criterion made visible that the guard catching the way an unmounted share
   actually arrives had no criterion at all, and a behaviour an operator can see is WHAT rather than
   HOW. Only the flag stays here.
3. **A removal is computed from a complete reading of every root of a library and applied in one
   transaction with the additions.** So a scan that was cancelled, that hit a failed batch, or whose
   second root failed while the first succeeded removes nothing at all. This is why `Reconcile` takes
   whole sets rather than streaming: a streaming reconciliation cannot tell *"not seen yet"* from
   *"not there"*, and the difference between them is a user's library.

**What none of the three watches is one level down, and that is the whole of
[behaviours §5.2](../../docs/compatibility/behaviours.md#52-a-container-that-has-lost-every-file-is-not-removed).**
A *directory* that mounts empty inside a healthy root has no guard: the root is readable, the
library is not empty, and the reading is complete. So a series whose episodes all vanished is
correctly read as a series with no episodes — and the removal pass therefore marks **file-backed
items** removed and leaves the containers above them alone. Removing the container would be spec
§3.8's *"this directory is empty, so it is gone for good"* judgement made where nothing is watching.
The two servers agree here, measured
`[probe: tools/differential.py --named container-that-lost-every-file, Jellyfin 10.11.11, 2026-09-02]`,
and the observable half — not offering a container with nothing under it — is 005's.

**User data costs this feature nothing, and that is a property of ADR-0003 rather than of the
removal pass.** Spec §3.8 requires that a file which disappears and comes back not cost a user their
favourites and resume position. Because no precious row references the derived half by row id
(§4.3), and because the identifier is a function of the path, a removed item's user data simply
keeps naming a string that will exist again when the file returns. **There is no retention rule to
write and no orphan to sweep**, and the risk is the opposite of the obvious one: a later feature
that "tidies up" user data whose item is gone would break AC-11 and nothing in this feature would
fail.

### 6.6 Sort keys

Two derivations, computed at write time into `items.sort_key`, compared with `BINARY` collation
(ADR-0003), never recomputed at read time.

**The base derivation** is spec §3.7.1's six ordered steps with the measured defaults, and the two
rules that make it right are both about **not** being tidy: nothing collapses the double space
`Rock & Roll` leaves behind and nothing trims the trailing space `S.W.A.T.` leaves, because steps 3
to 5 neither trim nor collapse. `strings.Fields`, `strings.TrimSpace` and `strings.Join` are each a
way to lose that by accident, so the implementation walks runes and appends.

**The three types that replace it** — `Audio`, `Episode`, `Season` — build a zero-padded numeric
prefix and append the **raw** name, and the asymmetry is real: an episode's season pads to three and
its episode number to four `[source: MediaBrowser.Controller/Entities/Audio/Audio.cs:94-98, MediaBrowser.Controller/Entities/TV/Episode.cs:238-242, MediaBrowser.Controller/Entities/TV/Season.cs:149-152 @ v10.11.11]`.
A missing number contributes **no segment at all** rather than a run of zeros.

**There is one exported entry point and it takes the item, not the name.** `SortKeyFor(*ScannedItem)`
switches on the type; `SortKeyBase(string)` is exported only because 004 needs it for an explicit
sort title (§3.7.3) and 005 needs it for a by-name row. This is
[behaviours §2.6](../../docs/compatibility/behaviours.md#26-sortname-has-two-derivations-and-three-types-use-the-second)'s
second named temptation made structurally hard: *"using one sort-name function for everything"*
makes a track called `The Song` sort under `s` and reorders every album in the library, and a caller
that has only the name in its hand cannot reach the type-aware function by accident.

**The pad width is part of the contract and the check for it is a test, not a comment.** Numeric
ordering here is lexical comparison over zero-padded digit runs, so a different width produces a
different ordering between names whose runs differ in length. The width is a named constant with the
measured value and a test that asserts the ordering of `2 Fast 2 Furious` against `10 Things`, which
is the pair the width exists for.

**OQ-7's tail is implemented as the spec writes it**: fold, then a short table of the obvious Latin
readings, then drop what remains. Dropping is stable; a partial guess is not. The row stays open
because it needs a name this repository has never measured.

### 6.7 Configuring a library, given there is no route to do it with

**The mechanism is a subcommand of the same binary**, which is
[002 plan §6.9](../002-authentication-users-and-sessions/plan.md#69-provisioning-and-the-three-seats-a-run-needs)'s
precedent taken deliberately rather than by analogy — and 002's argument for it was not convenience:
`conformance/` imports nothing of ours and needs a black-box way to create state.

```
atrium library add    --data-dir DIR --name NAME --type movies|tvshows|music
                      --root PATH [--root PATH …] [--case-sensitive]
atrium library list   --data-dir DIR
atrium library rename --data-dir DIR --name NAME --to NAME
atrium library roots  --data-dir DIR --name NAME --root PATH [--root PATH …]
atrium library remove --data-dir DIR --name NAME
atrium library scan   --data-dir DIR [--name NAME] [--full] [--allow-empty-root]
```

**What the shape buys**, each of which decided it:

- **`conformance/` can use it.** Adding a library and scanning it are the only two acts that put this
  feature's state into an installation, and both are now things a package that may import nothing of
  ours can perform. Without them every one of spec §5's criteria would be proven one layer in.
- **The store stays the single source of truth.** The alternative — a configuration file in the data
  directory, which is the reading architecture §9's *"library configuration lives in the data
  directory"* most obviously invites — would be a second schema for the frozen columns, hand-edited,
  beside the one the identifier derivation reads. Worse, it makes *"the collection type is frozen at
  creation"* a rule an operator breaks with a text editor and discovers by losing every identifier
  in the library.
- **Editing and recreating stay visibly different acts**, which is spec §3.6's sharpest consequence.
  `rename` and `roots` are free; `remove` followed by `add` is not the same library and every item
  under it gets a new identifier. Two verbs make that a decision an operator takes; one `set` verb
  that quietly recreated would make it an accident.

**What it costs**, stated rather than discovered:

- **A running server does not learn about a new library until it looks.** The subcommand writes to
  the same store file the server is reading, so `add` is visible to the next scheduled scan and to
  the next request that lists libraries, and not before. This is the honest cost of two processes
  over one file, and it is bounded by there being no cache: every reader reads the table.
- **`scan` from the command line and the server's scheduled scan can collide.** That is what
  `ClaimScan` is for (§6.9), and it is a refusal with a message rather than a race.
- **The commands are the operator's only interface, so their output is a contract of a sort.**
  `list` prints a table and `scan` prints spec §3.8's summary; both are read by `conformance/` (§8.1)
  and both are therefore asserted, machine-readably, with `--format json` on the two that report
  anything worth parsing. Parsing a human table in a test is how a test starts constraining prose.

**No password-shaped rule applies here**, and it is worth saying so: nothing this command takes is a
secret, so there is no stdin rule to inherit. What it *does* inherit is 002's placement — `RunLibrary`
lives in `internal/app` and `cmd/atrium` gains one arm on its dispatch, because
[architecture §3](../../docs/architecture.md#3-repository-layout) allows `cmd/atrium` no branch a
test would want to reach.

### 6.8 The derived half's generation, and the rescan that replaces 002's refusal

**The gap.** ADR-0003 says the two halves carry separate schema versions and that *"a derived-version
mismatch at startup is a rescan rather than an error"*. 001's T3 implemented both halves as
forward-only lineages, and its `migrate` refuses any recorded version above what the build knows —
with a comment saying, in terms, that the derived branch *"needs a scanner to rescan with, which is
003's"*, and that the refusal is owed a replacement there. 002's T3 inherited the refusal for both
halves and recorded the same thing. This is the replacement.

**The derived half is not a lineage and the runner has to stop treating it as one.** A forward-only
lineage cannot express *drop and rebuild*: it can only apply the steps after the recorded one, so a
schema edited in place is invisible to it and a database written by a newer build has nowhere to go
but a refusal. Three shapes were considered and the third is chosen:

| Shape | Why not |
|---|---|
| Keep the lineage; make the *newer-than-known* branch drop and rebuild | Answers a downgrade and not an upgrade. Editing `0001` in place still changes nothing, and adding `0002` writes a migration ADR-0003 says is never written |
| Record a **fingerprint** of the derived schema and rebuild whenever it moves | Rebuilds on a whitespace change and on a comment, so a reader cannot tell from the diff whether a change was meant to cost every installation a full rescan |
| **A generation integer the build declares**, and a rebuild on any difference in either direction | Chosen |

**So:** `internal/store/sqlite` declares `derivedGeneration = 1` beside a single embedded
`derived/library.sql` holding the whole current schema. At `Open`, after the precious lineage is
applied, the recorded derived version is compared with the constant. **Equal**: nothing happens.
**Different, either way**: every object the derived schema declares is dropped, the schema is
re-applied, the version is recorded, and every configured library is left with no items and no
`scan_state` row — which is the same state a library that has never been scanned is in, and is why
§4.3 puts `scan_state` in the derived half.

**What makes the bump deliberate rather than forgotten.** A generation nobody bumps is a schema
change that ships as a silent corruption, so the constant is paired with the SHA-256 of the schema
file and a test asserts the pair. Editing the schema without bumping the generation fails the build,
with a message naming the two values. This is 002 T1's shape — *"the constraint is redundant today
on purpose: it is the one thing in the schema that notices"* — applied to a constant instead of to a
`UNIQUE`.

**And 002's literal `0` moves to `1`, deliberately, which is what its handover asked for.**
`internal/store/sqlite/migrate_test.go` asserts the derived half is at `0` *"neither 001 nor 002
scans anything"*. It becomes an assertion that the derived half is at `derivedGeneration`, and the
task that writes the schema is the task that changes that line.

**Amended 2026-09-05, while the task list was written: that line is one of *four* assertions this
change costs, not one.** Reading the runner in order to write [T11](tasks.md#t11--the-derived-half-stops-being-a-lineage-and-001s-runner-changes-after-all)
found the other three, and naming one leaves the other three to be met as a red build by somebody who
will assume they caused it:

- `TestLoadLineageReadsAHalfWithNothingInIt` asserts that `loadLineage(migrationFiles, Derived)` is
  empty. It **stays true and stops meaning anything**, which is worse than failing: it is replaced by
  an assertion that no `migrations/derived` directory exists at all, so that a migration filed there
  — applied by nothing after this change — is a failing build rather than a file nobody reads.
- `TestTheRunnerAppliesOnlyWhatIsPending` proves the runner takes a half **by migrating the derived
  one**. That is not a call anything makes afterwards, so the test moves to a synthetic lineage and
  keeps proving the same thing about the runner.
- `migrate`'s newer-than-known refusal, and the doc comments on `Half` and `Derived`, each state that
  the derived lineage is empty and that the refusal *"is owed a replacement there, not here"*. All
  three become false in this change and all three are rewritten in it.

**And [ADR-0003](../../docs/decisions/0003-sqlite-as-the-store.md) is not edited, deliberately.** Its
*"a derived-version mismatch at startup is a rescan rather than an error"* stops being an open gap
here, and [AGENTS.md §4](../../AGENTS.md) makes an accepted record immutable — a wrong one is
superseded, never edited. This one is not wrong: it is a decision as taken, including what it owed at
the time, and it already names the scanner as what the branch was waiting for. Say so in the commit,
which is how 002 T6 handled ADR-0006's own *"asserted nowhere"* line.

**The rescan itself is at start and the scan is not.** Dropping is synchronous, inside `Open`, and it
must be: a store handed to a caller with a schema from another generation has exactly one correct
use and nothing enforces it. The **scan** that refills it is not, because a synchronous full scan of
every library would make a version bump into a start that takes minutes and a readiness gate
(001 §3.5) that stays shut for all of them. So the entry layer enqueues a full scan of every library
after the server begins serving, and ADR-0003's batched transactions are what make the intermediate
state coherent: a library mid-scan answers with what has been committed, which is a growing library
rather than a wrong one.

**The cost is named because a client sees it**: for the length of one rescan after a version bump,
a library is incomplete, and a client that caches a listing caches a partial one. Nothing is *wrong*
— identifiers are derived, so nothing a client already holds is invalidated — and the alternative,
refusing to serve until the scan finishes, trades a partial library for no library. §9 carries it.

### 6.9 Two scanners, batching, and what a scan does while the server serves

**A scan claims its library.** `ClaimScan` writes `(claimed_at, claimed_by)` on the `scan_state` row
in one conditional statement and reports whether it won. A claim older than `staleAfter` is broken
and taken, because a process killed mid-scan leaves one behind and the alternative is a library
nothing will ever scan again. `staleAfter` is **not** a guess about how long a scan takes: the claim
is **renewed on every committed batch**, so the value only has to exceed the time between two
batches. That is what makes it defensible rather than a number nobody could argue for later, and it
is why the renewal is part of the batch's transaction rather than a timer beside it.

**Batches are sized in items, not in time**, and each is one transaction that writes the additions
and updates it holds, renews the claim, and commits. Removals are **not** batched: they are computed
once from the complete reading and applied in the final transaction (§6.5's third guard), which is
also the transaction that releases the claim and writes the summary. So a scan that dies half way
has added and updated some items and removed none, which is a state the next scan corrects, and it
is the *only* partial state this feature can leave behind.

**Readers are not blocked.** ADR-0003 measured it: 57,664 reads completed during one write
transaction inserting 30,000 rows, worst read latency 393 µs
`[measurement: modernc.org/sqlite v1.58.0, Go 1.27.0, 2026-09-02]`. The batching is what keeps that
measurement relevant rather than a hope about a transaction held open for a whole tree.

**The schedule.** Spec §2 puts filesystem watching out of scope and says v1 rescans *on demand and
on a schedule*. The schedule is a process setting under
[architecture §9](../../docs/architecture.md#9-configuration-identity-and-logging) — `--scan-interval`,
with an environment fallback, a default measured in hours, and `0` to disable — and every scheduled
scan is **incremental**. It is bound to the server's own lifetime and cancelled by shutdown, and a
scan cancelled that way releases its claim on the way out, which is what makes the graceful-shutdown
rule of architecture §5 load-bearing for this feature as well as for 008's children.

## 7. Failure handling

There is no request to answer, so every row here ends in a log line, an exit status, or a refusal to
start. That is itself worth stating: **this feature's failure surface is an operator's, and the
audience for every message below is somebody with a shell.**

| Failure | Detection | Response | Recovery |
|---|---|---|---|
| A root does not exist, or is not a directory | `library add`, and again at every scan | `add` refuses with the path; a scan fails **that library** and changes nothing (§6.5, guard 1) | Operator fixes the mount |
| A root is a directory and cannot be read | The walk | The same: the library's scan fails, nothing is written | — |
| An error anywhere inside the walk | The walk | The library's scan fails, nothing is written. **Never a partial reconciliation** | — |
| A root reads as holding no candidate file where the last scan saw some | §6.5, guard 2 | The scan refuses and names the root | `--allow-empty-root`, when the operator meant it |
| A path that will not normalise — absolute, or climbing above its root | §6.3 | The library's scan fails. It is a caller error, not a bad file | A bug report; nothing an operator can fix |
| A file disappears between the walk and the batch | `ApplyScanBatch` | The item is written with the size the walk read; the next scan removes it | Next scan |
| A batch fails to commit | The store | The scan stops, the claim is left to go stale, nothing is removed | Next scan |
| Two scanners on one library | `ClaimScan` returns false | The second reports *"already being scanned"* and exits non-zero. Not a fault | Wait, or `--full` later |
| A claim left by a dead process | `ClaimScan`, on age | Broken and taken, with a log line naming the previous claimant | — |
| The derived generation differs from the build's | `Open` | Drop, recreate, and enqueue a full scan of every library (§6.8) | Automatic |
| The derived schema cannot be recreated | `Open` | **Refuse to start**, naming the file. The precious half is untouched | Operator removes the database, or restores it |
| A precious migration fails | `Open` | Refuse to start — 001's rule, unchanged | — |
| A recomputed identifier disagrees with the stored one | §6.4 | The library's scan fails, naming both. **Never a rewrite** | A bug report; a rewrite would be the silent discard Principle VII forbids |
| `library add` names a folded name that exists | The unique index | Refuse, and say which library holds it | Choose a name |
| `library add` names a collection type that is not one of three | Flag parsing | Refuse, listing the three | — |
| An attempt to change a frozen column | `rename`/`roots` do not offer it; there is no verb | No verb exists to refuse (spec §3.6) | Remove and add, knowing what it costs |

**The unreadable-root row is the one an acceptance criterion turns on**, and it is the row a test
most easily proves nothing about. AC-12 asks that a root which cannot be read fails the scan and
removes nothing; a test that asserts only the error is met by a build that errors *after* removing.
So the assertion is on the item count and on three specific identifiers before and after, and the
mutation that must fail it is a reconciliation moved ahead of the guard.

## 8. Testing strategy

### 8.1 What replaces the HTTP boundary, and how much weaker it is

**Principle VIII wants a conformance check at the HTTP boundary asserting on bytes. 003 has no
route, so there is no boundary and there are no bytes.** That is not a gap to be papered over with a
different kind of test called by the same name; it changes what can be proven and what cannot, and
this section says which is which.

**What `conformance/` can do here is small, real, and stated with its limit.** The package speaks
HTTP and imports nothing of ours, and this feature registers no route — so there is nothing for it
to request. What it *can* do is run the binary's own subcommands, which is a boundary of a different
kind: the **process** boundary, where everything the test knows is something an operator could have
known.

| Assertion | Instrument | What it proves |
|---|---|---|
| `library add`, `list`, `remove` behave, and the frozen columns have no verb | `conformance/`, running the binary | The operator's interface, end to end, including the flag set |
| A scan of the fixture tree reports the counts of spec §3.8 | `conformance/`, `scan --format json` | That the scan ran, and what it says it did |
| **A second scan of an unchanged tree reports no changes** (AC-2's second half) | `conformance/`, two scans | Determinism, at the only level this feature has one |
| The L0 check stays green with no new rows | `conformance/` and `internal/httpapi`, unchanged | That this feature added no route |

**And here is the limit, stated rather than left for a reader to work out.** Not one of those
assertions can see an item's *shape*. There is no field name to be PascalCase, no `null` to be
absent, no integer that could have been a string, no key order and no list order — because 003
produces no wire representation at all. **The instrument is not weaker than the HTTP boundary at
catching those; it is inapplicable, because the things Principle VIII exists to catch do not yet
exist.** What it *is* weaker at is everything else, and §8.3 lists exactly what.

**So the weight is carried by two instruments that are not `conformance/`, and both are honest about
where they sit.**

- **L2 over the fixture, beside the packages.** Spec §6 declares L2 for all five of this feature's
  behaviours and says in terms that L0 and L1 do not apply. The resolution, identity, sort-key and
  reconciliation rules are functions, and a function's test at the level of the function is not
  *"one layer in"* — it is the layer. What makes that claim honest rather than convenient is that
  the same functions are the only producers: there is no second path by which an item reaches the
  store, and §6.6's single entry point is designed so a caller cannot reach the wrong derivation.
- **The recorded reference reading, compared as a declared inequality.** §8.2. This is the strongest
  check this feature has and it is stronger than an L2 assertion in the one way that matters: it
  compares against what a **real Jellyfin** made of this repository's own tree, rather than against
  what this project expected.

**Where the 001 lesson bites, and what is done about it.** 001's closing audit found twice that *a
criterion written about a request is not met by a test about the mechanism that serves it, however
good that test is*, and the way to tell was to break the wiring. 003 has no request, but it has the
same class of wiring: the seam between `Resolve` and the store, and the seam between `Reconcile` and
the removal. **Both are asserted through the subcommand rather than through the function**, in
`internal/app`, over a real temporary data directory — so a build whose resolver is right and whose
`ApplyScanBatch` writes the wrong parent goes red. Every criterion's row in §8.4 says which of the
two levels its assertion sits at, and the ones that sit at the function level say why no higher one
exists.

### 8.2 The declared inequality, and the forty-seven

[reference-fixture-reading.json](../../docs/compatibility/reference-fixture-reading.json) is the
reference's own reading of this repository's fixture tree — 74 items over six libraries, each as a
type, a name and the file behind it, with the probe's citation and the image digest inside the file
`[probe: tools/probe_reference_scan.py, Jellyfin 10.11.11, 2026-09-02]`. It is generated, never
edited by hand ([docs/README.md](../../docs/README.md)).

**The comparison against it is not an equality**, and that is the single most useful thing 010 hands
this feature: a conformance assertion is a **declared inequality**, where an undeclared difference
fails *and a declared one that has gone away fails too* ([AGENTS.md §3](../../AGENTS.md)). The two
readings differ in **forty-seven declared places**, and
[010's AC-2](../010-conformance-harness/spec.md) records that **every one of the forty-seven belongs
to 003 or 004**.

**What this project has and what it does not.** It has the reading. It does **not** have the
declaration: the module that held it stayed in the source repository, so the forty-seven reasons are
written here from the reading, from the specifications that cause each one, and from nothing else.
That is work this plan sizes rather than inherits, and it is the largest single task in the feature.
Four shapes are already named by 010's own amendment and account for most of the count:

| Shape | Owner | Why |
|---|---|---|
| A zero-byte film that is an item there and not here | **003** | Spec §3.2's ignore rule, working as written (§6.1) |
| Twenty-five files named differently | **004** | The reference's own metadata resolution over the same names |
| An empty library that is nothing at all to the reference | **003** | A library with no items has no root row there and one here |
| Every library's own root row | **003** | The reference's is a `Folder`; spec §3.1 makes each library a `CollectionFolder` |

**Amended 2026-09-05, while the task list was written, in two places this section sized and did not
place.**

**Where the declaration lives.** It is a Go table beside the comparison in `internal/app`, and
**not** a seventh machine-readable file under `docs/compatibility/`. A new artefact there owes a
prose twin and a row-for-row test ([docs/README.md](../../docs/README.md#paired-files-edit-both-halves-or-neither)),
and this one has no twin to pair with: the prose that explains a row is *this project's own
specification section*, which the row cites. [conformance.md](../../docs/compatibility/conformance.md#l3--differential)
already describes the declaration as living *"in that module with its reason"*, so this is the
recorded shape rather than a new one. [T17](tasks.md#t17--the-forty-seven-declared-differences-which-this-project-holds-the-reading-of-and-not-the-reasons)
takes it.

**And twenty-five of the forty-seven are 004's, while 004 does not exist.** They are declared now
because the comparison cannot run without a reason for every difference — and the consequence is
worth meeting here rather than in CI: *a declared difference that has gone away fails too*, so **the
day 004's metadata resolution renames one of those items, the row declaring that difference goes
red.** That is the rule working rather than a defect. What 004 owes is to edit those rows rather than
delete them, and to keep the total asserted from the declaration's own length, since a row removed to
make a run go green is exactly what the count assertion exists to catch.

**Two rules for the comparison, and the second is the one that is easy to get wrong.** First: it
compares type, name and path — not identifiers, because behaviours §1.4 already establishes those
differ by design and comparing them would declare 74 differences that say nothing. Second: **a
declaration is a reason and an owning feature, not a licence.** A difference declared *"003 §3.2"*
that stops appearing fails the comparison, which is what makes the file a record of decisions rather
than a list of excuses — and it is exactly how a rule quietly stopping working gets found.

**And one thing found while reading for this section, which is not this plan's to fix.**
[conformance.md](../../docs/compatibility/conformance.md) states the count **twice** and the two
disagree: its L2 section says *"forty-seven declared differences"* and its L3 section says
*"twenty-six places"*. Forty-seven is the later number — it is 010's D-7, taken 2026-09-02, and it is
the number [CLAUDE.md](../../CLAUDE.md) carries. The stale one is recorded here rather than edited,
because that document is not this feature's and because an implementer who takes twenty-six as the
target would declare twenty-one differences too few and fail the run for the wrong reason. §11 names
it as owed.

### 8.3 What only becomes observable at 005, and what 005 must not accept as proven

**This is the deferral, and it is stated as plainly as 001 stated its two L3 rows and 002 stated its
one.** The following are decided by this feature, produce no output this feature can assert against
a client's view, and become wrong answers on somebody else's route:

| Claim | Where it surfaces | What is proven here instead |
|---|---|---|
| The derived identifier's **bytes** | `Id` on every item, `ParentId`, and every user-data key from 007 | The derivation is a function with a table-driven test, and the **stored** string is asserted in `internal/store/sqlite`. Nothing checks that the string a client receives is the string that was stored |
| The sort key's **bytes**, and therefore every list's order | The order of every `/Items` response | The key is asserted as a string, including the double space and the trailing space. Nothing checks that `ORDER BY` uses that column, or that it uses `BINARY` |
| Parent-child structure | `ParentId`, and `/Items?parentId=` | The `parent_id` column. Nothing checks that a client asking for a season's children gets them |
| `IndexNumber`, `ParentIndexNumber`, `IndexNumberEnd`, `ProductionYear` | The item body | The columns. Nothing checks their **type** on the wire, which is precisely the class behaviours §1.1 to §1.7 exist for |
| A multi-part film being one item with two sources | `MediaSources` at 008 | One `items` row and two `item_files` rows |
| A container with nothing under it not being offered | `/Items` at 005 | Nothing. It is [behaviours §5.2](../../docs/compatibility/behaviours.md#52-a-container-that-has-lost-every-file-is-not-removed)'s closing half and 005 owns it entirely |

**Two rules follow, and they are addressed to whoever writes 005's plan.**

1. **A green suite here is not evidence for any row above.** Each of them needs an assertion at the
   HTTP boundary in 005's own `conformance/` tests, and 001's audit says how to check that the
   assertion is real: break the wiring — write the wrong column, order by `name` instead of
   `sort_key` — and watch a test go red.
2. **The first four rows are one request each.** A single `/Items` listing over the fixture library,
   compared byte for byte, covers the identifier, the order, the parent and the four numbers at
   once. It is the cheapest debt in the project to discharge and the easiest to leave open, because
   every one of those values will *look* right in a body somebody eyeballs.

### 8.4 Each criterion, and the level its assertion sits at

Spec §5's ~~fifteen~~ **sixteen**. `library` and `scan` mean a Go test beside that package; `app`
means through the subcommand over a real data directory; `conformance` means through the binary.
*(AC-16 was added on 2026-09-05, while the task list was written; §6.5's amendment says what forced
it.)*

| AC | Where | How |
|---|---|---|
| 1 | `library` + `app` | The fixture tree resolves to the declared item set for all three collection types, compared as a whole set rather than by counting. §8.2's comparison is the same assertion against the reference's reading |
| 2 | `app` + `conformance` | Scan, scan again: identical identifiers, and the second scan's summary reports zero added, updated and removed. The `conformance` half is what makes "the second scan reports nothing" a fact about the binary |
| 3 | `app` | Scan, drop the derived half through §6.8's rebuild, scan again, compare identifiers. **This is AC-2's criterion with the store's memory removed**, and it is the one that catches a derivation that accidentally reads a stored row |
| 4 | `library` | `The Long Film (1998)` → one item, two `item_files` in part order. The **name** is asserted too, because §6.2's directory rule is what makes it `The Long Film (1998)` |
| 5 | `library` | `S01E02-E03` → one item with `IndexNumber` 2 and `IndexNumberEnd` 3 |
| 6 | `library` | `Specials` → season 0, **and** a companion asserting that `Extras` and `Featurettes` beside it are not seasons. The spec's warning is that grouping the three is a scan that looks entirely correct |
| 7 | `library` | The series `24` keeps its title and acquires no episode number, and its episode's numbers come from the filename rather than from the directory |
| 8 | `library` | `Double Album/CD1` and `CD2` → one album, tracks carrying disc 1 and disc 2 |
| 9 | `library` | `Various Artists/A Compilation (1999)` → one album. With the null tag source, the attribution comes from the directory; 004's tests carry the tag-driven half |
| 10 | `app` | Scan, move the whole tree to a second temporary directory, `library roots` it, scan again: **every identifier unchanged**. The criterion is the one §6.3's *"the library identifier is in the key, and the root path is not"* exists for |
| 11 | `app` | Delete a file, scan, assert the item gone; write a row into a precious table keyed on that identifier before the deletion and assert it survives; restore the file, scan, assert the identifier returns. **The middle clause is the one that would otherwise be missing**, and until 007 exists the precious row is written by the test through a store method rather than by a feature |
| 12 | `app` | §7's row: item count and three named identifiers before and after a scan whose root cannot be read. Proven able to fail by moving the reconciliation ahead of the guard |
| 13 | `library` | Spec §3.7's fourteen-row table, verbatim, including `rock  roll`'s two spaces and `s w a t `'s trailing one; plus §3.7.2's three types; plus the `2 Fast` versus `10 Things` ordering the pad width exists for |
| 14 | `app` | The four rows of §3.8's change table that AC-14 names, each as its own mutation of the fixture between two scans: a modified file keeps its identity, an appearing file is added, a renamed file is a delete plus an add. **Size and modification time are varied independently**, because a build reading only one of the two passes a test that varies both |
| 15 | `library` | An explicit sort title replaces the derivation for **every** type including the three that override, and is lowercased and digit-padded but not article-stripped. The value is supplied through the same seam 004 will fill |
| 16 | `app` | §6.5's second guard, in **both** halves: a root that reads as holding no candidate file where the last scan saw at least one refuses the scan and names the root, and the same scan with `--allow-empty-root` proceeds and removes. A test asserting only the refusal passes on a build whose override does nothing |

**Two criteria are asserted at a level below the one they are written at, and both say so here rather
than being discovered.** AC-1's *"exactly the expected set of items"* and AC-13's *"sort ordering"*
are both about what a client eventually sees, and neither can be asked of a client in this feature.
AC-1 is discharged twice — against this project's declaration and against the reference's recorded
reading — which is as close as it gets. AC-13 is discharged against the **stored key**, and the step
it cannot take is that the ordering a client receives is that key's; that step is in §8.3's table and
is 005's.

### 8.5 The fixture, and why it is generated rather than checked in

The tree is [conformance §L2](../../docs/compatibility/conformance.md)'s scanning world: paths and
filler bytes, no copyrighted media, ever (spec §6). It is **declared once** in
`internal/libraryfixture` and written into a directory by a builder, and `tools/build_library_fixture`
is the same declaration as a program so `conformance/` can have the tree without importing anything
(§3).

**Generated rather than committed, for three reasons that are each a file this repository cannot
hold.** A zero-byte file survives git; a **modification time** does not, and §6.4's whole signal is
one. A file that is being written cannot be a committed file at all. And the tree has to be
**mutated between two scans** for AC-11 and AC-14, which means every test needs its own copy anyway.

**The declaration is the reference's reading read backwards, and that is deliberate.**
`reference-fixture-reading.json` names every path the reference made an item out of, per library —
so the builder's declaration is checked against it: **every path the reading names must exist in the
built tree**. A fixture that drifted from the tree the reading was taken over would make §8.2's
comparison meaningless while leaving it green, which is the worst available outcome, and one test
closes it.

**Six libraries, and 003 needs three of them.** `Movies`, `Shows` and `Music` are this feature's;
`Films` and `Tunes` are the media world 008 encodes with ffmpeg and are behind a build tag
(architecture §8); `Empty` is a library with nothing in it, which
[behaviours §5.7](../../docs/compatibility/behaviours.md#57-an-empty-library-reads-unplayed-where-the-references-source-reads-it-as-played)
needs and this feature must be able to configure. **The suite staying green without the build tag is
the check that every ffmpeg-reaching test carries it**, and this feature is the first to need the
rule since architecture §8 wrote it.

**One thing the fixture must not do.** It must not be scanned by a test that then asserts a count it
computed from the same declaration. Two of spec §5's criteria are counts, and a count derived from
the builder is a test of nothing. The expected set is written out as a literal — the way §8.2's
comparison is — so that a change to the tree is a change to two files and a reviewer sees both.

## 9. Risks

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Every claim in §8.3 is proven a layer below where it is written | **Certain** | Four kinds of wrong answer that only 005 can see, on the busiest route in the surface | §8.3 lists them as debts with the request that discharges them, and 005's plan inherits the list. This is the feature's largest single weakness and it is structural, not an oversight |
| The forty-seven declared differences are written from the reading rather than inherited | High | A miscount fails 010's run for the wrong reason; a difference declared with the wrong reason hides a real bug | §8.2. Each declaration names a specification section, so a wrong reason is a wrong citation somebody can check |
| A version bump leaves libraries incomplete while the background rescan runs | Medium | A client lists a partial library and caches it | §6.8. Nothing is invalidated — identifiers are derived — and the alternative is refusing to serve at all. Named so it is a decision rather than a surprise |
| Two scanners over one store file | Medium | Contention, or two scans of one library | §6.9's claim, renewed per batch; ADR-0003's WAL and busy timeout absorb the rest |
| The `.ignore` rule is narrower than the reference's in two ways | Medium | On a tree with a non-empty marker, items here that the reference hides | §6.1, [U-42](../../docs/compatibility/reference-target.md). One `.ignore` file settles it |
| Spec §3.3's `-a`/`-b` stacking form does not exist at the reference | Medium | Implementing the spec's reading merges two films into one and loses one of them | §6.2 implements the source's vocabulary; [U-43](../../docs/compatibility/reference-target.md), and the spec's parenthetical is corrected |
| Case-insensitive identifiers by default differ from the reference's default | Medium | Two files differing only in case are one item here and two there — an item-count difference on a real library | §6.3, [U-44](../../docs/compatibility/reference-target.md), and it is one of the forty-seven §8.2 declares |
| 004 and the scan fight over `items.name` | Medium | A refresh overwrites what a scan resolved, or the reverse | Out of this feature's hands and named for 004's plan: the column has one writer today and 004 adds a second. [behaviours §5's rename row](../../docs/compatibility/behaviours.md#5-accepted-gaps-in-v1) already records the same fight from the editing side |
| A sort-configuration change silently invalidates every stored key | Low today | Half a library ordered one way and half the other, permanently | §4.3: the lists are constants in v1, and the day one becomes editable it is a generation bump |
| The derived generation is not bumped when the schema changes | Low | A database with an old schema and a current generation, failing at the first query | §6.8's paired constant and hash, asserted by a test that fails the build |

## 10. Alternatives considered

**Give 003 a route, so that Principle VIII has something to assert on.** The obvious escape from §1's
organising problem, and the reference even has one — `POST /Library/Refresh`. It is not in
[surface.yaml](../../docs/compatibility/surface.yaml) and cannot be: Principle VI keeps an endpoint
out until a named consumer is measured calling it, and neither analysed client does.
[behaviours §5.2](../../docs/compatibility/behaviours.md#52-a-container-that-has-lost-every-file-is-not-removed)
already records what that costs — *"Atrium cannot be asked for a second scan over the wire at all"* —
and pays it rather than routing a stub. A route added to make a test possible is a delta added to
make a test possible.

**Configure libraries from a file in the data directory.** Closest to architecture §9's *"library
configuration lives in the data directory"* read literally, and declarative. Rejected in §6.7: it is
a second schema for the two frozen columns, hand-edited, and it turns *"the collection type is frozen
at creation"* into a rule an operator breaks with a text editor and discovers by losing every
identifier under the library. The store is the data directory in the sense that matters.

**Make the derived half a forward-only lineage like the precious one.** It keeps 001's runner
untouched, which is what 001 expected. It cannot express *drop and rebuild*, so ADR-0003's central
claim would be implemented as *"write a migration for tables a rescan would have rebuilt in a
minute"* — the exact shape that record rejects in its own alternatives.

**Fingerprint the derived schema instead of numbering it.** Automatic, and it notices an edit nobody
bumped. Rejected in §6.8: it also rebuilds on a comment, so the diff no longer says whether a change
was meant to cost every installation a full rescan. The generation is paired with a hash and a test
instead, which gets the noticing without the ambiguity.

**Reconcile as a stream, item by item, as the walk yields them.** Constant memory, and it starts
writing sooner. It cannot tell *"not seen yet"* from *"not there"*, which means either no removals at
all or a removal pass that runs after the walk anyway — and the first thing a streaming design gives
up is §6.5's third guard, which is the one that makes a cancelled scan safe. A library's item set is
small enough to hold; ADR-0003 measured 40,000 rows.

**Reproduce the reference's identifier derivation**, `MD5(UTF16LE(type.FullName + path))` with .NET's
mixed-endian layout `[source: Emby.Server.Implementations/Library/LibraryManager.cs:636 @ v10.11.11]`.
Possible, and it would make one whole class of difference disappear. Rejected three times over
before this plan: it needs a C# type's `FullName`, which is an implementation detail of a codebase
this project does not fork (Principle IV); it keys on the **absolute** path, which is the trap
behaviours §1.4 measures and architecture §6 chose to escape; and a declared difference that has
gone away fails under [AGENTS.md §3](../../AGENTS.md), so the change would be a three-way paired edit
rather than a derivation. 002 refused the same temptation for a session identifier and gave the same
answer.

**Keep the scanned library in memory and persist only user data.** ADR-0003 already considered and
rejected it, and the reason is this feature's: *"every restart becomes a full rescan — where 003's
whole point is that a scan is incremental, which needs the previous scan's state on disk to compare
against"*. It is repeated here because §6.4's signal is exactly that state, and a reader of this plan
is the person most likely to think the derived half could be transient.

**One `internal/library` package holding the rules and the walk.** Fewer packages, and it puts
`os.DirFS` in the same package as the sort-name derivation 005 imports. §3 gives the positive reason:
005 and 004 need the rules and must not reach through a package that can delete a library to get
them.

## 11. What this change amended in `spec.md`, and what forced each one

Three edits, all in this change, all dated in the specification's front-matter `amended:` line. Each
is a **source reading** that contradicts or narrows a sentence the specification carries, and in each
case the specification is implemented as written where it says something, and extended where it says
nothing — the running server being the tie-breaker, and there being none here
([AGENTS.md §1.3](../../AGENTS.md)).

1. **§3.2's `.ignore` row gains what the marker actually is.** *"A directory containing a `.ignore`
   file"* is the empty-marker case; the reference searches ancestors and reads a non-empty marker as
   `.gitignore`-style patterns
   `[source: Emby.Server.Implementations/Library/DotIgnoreIgnoreRule.cs:18-30,41-68,95-131 @ v10.11.11]`.
   Forced by §6.1, which had to decide what the walk does with each of the three shapes and could not
   decide it from the row. The row now says what v1 excludes and what it does not.
2. **§3.3's multi-part parenthetical is corrected.** *"the `-a`/`-b` form"* is not a form the
   reference stacks: the letter follows the same part word as the digits do
   `[source: Emby.Naming/Common/NamingOptions.cs:141-145 @ v10.11.11]`. Forced by §6.2, and it is the
   only one of the three whose wrong reading would **lose an item**: two films merged into one.
3. **§7's OQ-9 records the source reading and stays open.** `EnableCaseSensitiveItemIds` defaults to
   `true` and is a server-wide setting
   `[source: MediaBrowser.Model/Configuration/ServerConfiguration.cs:89 @ v10.11.11;
   Emby.Server.Implementations/Library/LibraryManager.cs:650 @ v10.11.11]`. The row stays open
   because a running server is the tie-breaker; recording it matters because a reader taking OQ-9 as
   *"nobody knows"* would not realise that §3.6's default is a **known** difference and one of §8.2's
   forty-seven. This is 002's OQ-6 handled the same way.

**§3.6 is deliberately not amended.** It already states Atrium's default as a decision and says in
terms that *"what the reference defaults to is not something this repository has measured"*. That
sentence is still true — a source reading is not a measurement — and OQ-9 is where the reading
belongs.

**And three rows were added to [reference-target.md](../../docs/compatibility/reference-target.md)'s
register**, U-42 to U-44: the `.ignore` rule's real shape, the stacking vocabulary, and the
case-sensitivity default. All three are source readings this project has implemented differently or
narrowly, each recorded with the request that settles it.

**One thing this plan leaves for another document, with the reason.**
[conformance.md](../../docs/compatibility/conformance.md) states the count of declared differences
over the fixture twice and the two disagree — *"forty-seven"* in its L2 section and *"twenty-six"* in
its L3 section (§8.2). Forty-seven is 010's D-7, taken 2026-09-02. It is not corrected here because
that document is one half of a pair with [allowlist.yaml](../../docs/compatibility/allowlist.yaml)
and because this plan is not the change that touches it; it is named so the next editor of that file
does not have to rediscover which number is stale, and so that whoever writes this feature's task
list budgets for the declaration being forty-seven and not twenty-six.
