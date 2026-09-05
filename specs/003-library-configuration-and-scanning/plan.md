---
feature: 003-library-configuration-and-scanning
title: Library configuration and scanning — implementation plan
status: Accepted
created: 2026-09-05
updated: 2026-09-05
spec_status_required: Accepted
amended: 2026-09-05 by the change that wrote tasks.md, in four places - section 6.5's count of how many guards the specification states, section 6.8's count of 001's assertions the runner change costs, section 8.2's home for the forty-seven declarations and who owns twenty-five of them, and section 8.4's criterion table, which gains AC-16; 2026-09-05 at T1, section 8.5, with the rule for what the fixture may hold beyond the paths the reference's reading names and why the case-only-differing name conformance L2 lists is not one of them; 2026-09-05 at T2, section 6.1, because the extras row's "per file and per directory" was two decisions the row did not take - the folder name matches the immediate containing directory and no ancestor, so the walk descends into an extras directory rather than pruning it, and the rules apply under movies and tvshows and not under music, which is the reference's media-type gate expressed as the one term this feature has; 2026-09-05 at T3, section 6.3, with the four decisions writing Normalise and DeriveID took that the section had left open - the NFC implementation and the dependency ADR-0002 defers to the plan that needs it, the fold being Unicode's simple lowercase rather than this package's ASCII one or a full case fold, what the separator step reduces beyond the separator character, and an interior parent element being a normalisation where one that leaves the root is the error - plus the singleton mapping that makes a Kelvin sign and a K one key in a case-sensitive library; 2026-09-05 at T4, sections 5 and 6.6, in three places - the record listing gains SortTitle, the field section 8.4 called a seam and the listing had none of, and the library package's comment stops claiming it names no port when the same block declares two signatures over one; section 6.6 gains the Season missing-number case and the two of six steps an explicit sort title actually uses, both source readings the code had to answer and the specification did not state; and section 6.6 records that the pad-width check the plan asked for does not pin the pad width, because the ordering it names survives 9, 10 and 11 alike; 2026-09-05 at T5, sections 6.2 and 8.5, with the six decisions writing the film resolver took that section 6.2 constrained and did not take - the year rule read at the pinned tag and the order it runs in, the release-tag vocabulary transcribed as data where the expressions are not transcribed at all, a library root never naming a film, the four rules a stack needs, and an item's path being the one the walk read rather than the folded key its identifier came from - plus the refusal a collection type with no resolver answers instead of an empty plan; and section 8.5 records that the fixture's own multi-part film cannot assert where its name came from, because the directory rule and the year rule each repair a name taken from the first part; 2026-09-05 at T6, sections 6.2 and 8.5, with the seven decisions writing the series, season and episode resolver took that section 6.2 constrained and did not take - where each of the three levels comes from and what a candidate under a bare root does for a series name, the season number's three sources in order, a series directory never also being a season directory, a season being named from its number rather than from its directory and taking a path only from a directory whose number is its own, the numbering family being the specification's with the reference's remaining expressions declared absent, the two cases a multi-episode range needs and the guard on its ending number, and Specials being season zero where the reference's own parser accepts Extras as well; and section 8.5 records that the fixture's 24 cannot assert that the filename is matched before the directory, because the season directory agrees with the filename and the flat shape has no season directory at all; 2026-09-05 at T7, sections 5, 6.2 and 8.5 - section 5 gains the TagSource seam and stops claiming the library package declares no interface, section 6.2 gains the six decisions writing the music resolver took (the three levels and a disc directory not being one of them, the disc vocabulary being the reference's album-stacking list and not the film one, the reference's further multi-disc conditions declared absent, an album's name losing its year and nothing else, the compilation rule implemented as its "only if", and a track no directory places not being unplaceable) and records that the refusal a collection type with no resolver answered is itself a stub now that all three are written, and section 8.5 records that the fixture's compilation cannot fail AC-9's own failure and that no tree in this feature can, because the distinction does not exist until something says the artists differ; 2026-09-05 at T8, section 6.1, with the two decisions writing the walk took that the section's table did not - an entry that is not a regular file once followed, where a link to a file is the file it points at and a link to nothing is refused rather than raised because failing a library's whole scan over a dangling link is worse than skipping it, and the .ignore search implemented as a pruned subtree with the library root as its inclusive end - and with the record that the determinism clause the task list asked for cannot be varied by a tree's creation order at all, because os.DirFS sorts every directory it reads; 2026-09-05 at T9, sections 5, 6.4 and 6.5 - section 5's Reconcile becomes a record and an error, because a disagreeing identifier is an error section 6.4 already required and the listing had no return value for, because Removed and the third return value were one list under two names, and because Unchanged and Retained are what keep this feature's two most dangerous claims from being claims about an absence; section 5 also gains library.SortItems, exported so that the batch a store is handed is in the one order every item set in this feature is in; section 6.4 gains the two decisions writing the function took that its table did not state - a record that moved is an update even where no file signal could ever move, which is the only way a container is ever rewritten, and a full re-examination applying to an item that has a file, so that the thorough option is never also a different set of deletions - and says how a desired item and a previous row are paired, by identifier with the path carrying the comparison, because pairing by either one alone loses a case-changed name or cannot see the disagreement at all; section 6.5 records that the removal pass names the containers it kept; and section 6.1's last row is corrected from a file whose size AND modification time moved to one whose size OR modification time moved, which is what spec 3.8's table and section 6.4's both say and which a conjunction made unsatisfiable, and gains the half the row left implicit - it lands as an update and never as a refusal; 2026-09-05 at T10, sections 4.1 and 5 - section 4.1's library_roots row gains ON DELETE CASCADE, because foreign keys are on and a delete of a library holding roots is refused without it, and the section records that case_sensitive deliberately carries no DEFAULT; section 5 declares ScanBatch, the fourth record type the listing named in a signature and never defined, and argues the ClaimedBy field, because a renewal that did not name the claimant would let a scanner whose claim had gone stale and been taken renew a claim it no longer holds; 2026-09-05 at T12, sections 5, 6.9, 7 and 8.3 - ClaimScan returns the claimant it displaced or lost to beside the boolean, because section 7 asks for two messages naming a claimant and neither name survives the call that overwrote it; section 6.9's one conditional statement becomes one transaction for the same reason, since an upsert's RETURNING answers the row as it now stands, with the write lock taken at BEGIN so the atomicity is unchanged; section 6.9 also records that a claim stamped after the instant offered is a clock that moved backwards and is treated as live, because breaking a claim on a clock adjustment is two scanners writing one library; section 7 gains two rows for refusals this task had to decide rather than transcribe, a batch naming one item twice and a removal naming an identifier no row holds; and section 8.3's second and fourth rows say what the store now asserts underneath them so that neither reads as discharged; 2026-09-05 at T13, sections 5, 6.5, 6.7, 6.9 and 8.1 - section 5 declares the scanner the listing had a summary for and no producer of, with Options as a record rather than two booleans and every refusal naming the library, the root's ordinal and its configured path; section 6.5's second guard counts files and not items, because a library's own row backs no file and an item count would make a library an operator deliberately emptied refuse every scan of that root for ever; section 6.5's third guard stops claiming one transaction, because section 5 declares the removal and the release as two methods and what the guard is made of is the ordering rather than the atomicity; section 6.9 records that the claim is taken after the reading, which the section's own argument for staleAfter forces since nothing renews a claim during a walk, and names the two consequences and the refusal that leaves no claim behind; section 6.7 records that scan, --format json and --log-level land at T13 rather than T14 because three of T13's criteria are about what the store holds after an operator ran a scan, that --name matches on the domain's fold, and that a name no library has is a refusal and not an empty run; and section 8.1 records what a seam test needs to discriminate at all and which two of T13's clauses sit at the scan level because a subcommand building its own store leaves nowhere to stand between two of its transactions; 2026-09-05 at T14, sections 6.7, 6.9 and 7 - section 6.7 gains the five decisions writing the other five verbs took (what allocates a library's identity and why Principle VII does not govern it, what folds a library's name, which T10 left open and which is Normalise's last two steps in Normalise's order, what remove does to the items nothing in the schema removes and the one row that outlives a library, a root being checked and made absolute where it is typed, and the shape --format json reports on list); section 6.9's claim that a cancelled scan releases its claim is struck, because a claim is released by a scan reaching its own end and a cancellation is the exit that does not reach one, and the section gains the twelve-hour default read at the pinned tag, the cancellation bounding the next root rather than the current one, and the record that the schedule and the start-time rescan were owned by no task because the schedule appears in the specification only in section 2's scope note; and section 7 gains three rows - a verb naming a library that does not exist, a refused scan interval, and a scheduled scan that fails; 2026-09-05 at T15, section 8.4, in the two mutations rows 3 and 10 named - a derivation that reuses a stored identifier is on its own a no-op no test in this repository fails, so what AC-3 catches is that reuse over an identifier that is allocated rather than derived; and the root path in the key has a broad form T14's moved-root test already catches and a narrow one, section 6.3's own misreading of the library root plus the normalised name, which only a corpus holding a Series and a MusicArtist can see - which is why AC-10 moves the whole fixture and asserts that the corpus holds every kind before it asserts anything else; 2026-09-05 at T16, sections 4.1, 4.3, 6.5 and 8.4 - section 4.1 gains a second precious migration, item_user_data, because AC-11's middle clause is unassertable without a real precious row and 007 does not exist, and it holds the two nouns of spec 3.8's own sentence and nothing of 007's other four; section 4.3's table gains its row; section 6.5's closing prediction that nothing in this feature would fail is struck, because one assertion does now and adding an orphan sweep to RemoveItems fails that test and not one other test in the repository; and section 8.4 records that row 11's clause needed a table as well as a method and that row 14's rename mutation has a form that adopts nothing, because the file signal includes the path and a renamed file's path is the one thing that moved; 2026-09-05 at T17, sections 8.2, 8.5, 9 and 11 - section 8.2 gains the declaration it sized and the count it derived, which is thirty-two over the four libraries this feature builds where 010's D-7 counted forty-seven over six, with eight of the remaining fifteen predicted over 008's media world and seven not derivable from the recorded reading and this project's specifications at all, and with the two mutations that prove the declared inequality can fail in both directions; section 8.5 records that T17's case-insensitive pair cannot be among the declared differences and that the clause is struck rather than satisfied by building the pair; section 9's case-sensitivity risk row loses its claim that the difference is one of the declared ones; and section 11's conformance.md correction is taken here rather than left owed, because this feature's own work is what makes the number checkable
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
| `library_id` | TEXT NOT NULL REFERENCES `libraries(id)` ON DELETE CASCADE | ~~No note~~ — the cascade was added at **T10**, which wrote `RemoveLibrary`. Foreign keys are on (ADR-0003's writer DSN), so a delete of a library holding roots is *refused* without it and the verb never works. Making it the database's rule rather than the method's discipline is `name_folded`'s own argument one row up |
| `ordinal` | INTEGER NOT NULL | The order the operator gave them, which decides nothing but keeps a list stable (architecture §2 forbids an order derived from anything else) |
| `path` | TEXT NOT NULL | Absolute, as configured |

*(Amended at T10, which wrote the migration and the six methods over it. Two things the table
above did not say and writing it had to decide. **The cascade**, in the `library_id` row. And
**`case_sensitive` gets no `DEFAULT`**: a default would be a second place the value is decided,
where spec §3.6 makes it a property an operator states when the library is declared — the column is
`NOT NULL` and the caller has to mean it. Neither is a deviation from §4.1; both are places §4.1 was
silent and a schema cannot be.)*

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

**Amended at T16: this feature ships a second precious migration, `0004_item_user_data.sql`, and
that is a decision this section did not take.** §6.5's closing paragraph says user data costs the
scanner nothing and then names the risk it leaves — *"a later feature that 'tidies up' user data
whose item is gone would break AC-11 and nothing in this feature would fail"*. AC-11's middle clause
is the criterion for exactly that, and it is unassertable without a real precious row: 007 owns user
data and 007 does not exist, so without a table here the criterion has **no test at all** until
somebody else's feature lands, which is the shape both closing audits caught.

**`item_user_data`** — one row per account per item, `PRIMARY KEY (user_id, item_id)`.

| Column | Type | Note |
|---|---|---|
| `user_id` | TEXT NOT NULL REFERENCES `users(id)` | A real foreign key: both tables are precious, and architecture §6 forbids a reference *from* the precious half *into* the derived one, not one within a half |
| `item_id` | TEXT NOT NULL | The item's **derived** identifier, as a string and deliberately **not** a foreign key. A constraint here would refuse §6.8's rebuild, and a cascade beside it would delete the user's favourite with the item — which is the thing spec §3.8 forbids. A row naming an identifier no `items` row holds is spec §3.8's *"in case it returns"* and not an orphan |
| `is_favourite` | INTEGER NOT NULL | Spec §3.8's *"favourites"* |
| `playback_position_ticks` | INTEGER NOT NULL | Spec §3.8's *"resume position"*. Ticks |

**It holds the two nouns of spec §3.8's own sentence and nothing else.** 007 §4 owns four more
properties — played, play count, last played date, and a session's live playstate — and none of them
is named by 003, so declaring them here would be this feature deciding another feature's storage,
which is the opposite mistake to the one above. **007 extends this table rather than replacing it**,
by a further precious migration; a parallel table would leave the assertion that guards all of this
watching the wrong rows. The two store methods are on the store and deliberately **not** on a
`ports` interface: 003 declares no domain that reads or writes user data, so a port method here
would be a contract with no caller above it and would fix the shape of a method 007 has to design.

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
| `item_user_data` | **Precious** *(added at T16)* | It is the only copy of what a user did, and spec §3.8 requires it outlive the item. Filed derived it would be dropped by the very rebuild it has to survive — see §4.1 |
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
    SortTitle               string   // §6.6's input, and the only field that is not a column
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

ScanBatch struct {               // one committed step of a scan, §6.9
    LibraryID string
    Items     []ScannedItem       // additions and updates; removals are never batched
    ClaimedBy string              // the claim the transaction renews is renewed as this claimant
    At        units.Time
}

ItemStore interface {
    ItemsForLibrary(ctx, libraryID string) ([]ScannedItem, error)
    ApplyScanBatch(ctx, batch ScanBatch) error       // one transaction, §6.9
    RemoveItems(ctx, ids []string) error
    ClaimScan(ctx, libraryID, by string, at units.Time, staleAfter units.Ticks) (bool, string, error)
    ReleaseScan(ctx, libraryID string, at units.Time, summary []byte, full bool) error
    RebuildDerived(ctx) error                        // §6.8; the drop and recreate
}
```

The four record types live in `internal/ports`, which is **T4's decision in 002 applied rather than
retaken**: a port method returning `library.Item` would make the bottom of architecture §2's diagram
import a domain package. The cost that decision carried there — the policy crossing as bytes — has
no analogue here, because a `ScannedItem` is already flat values and a `units.Time`.

**`SortTitle` is the one field with no column behind it, and T4 added it rather than found it.**
§8.4's row for AC-15 says the explicit sort title *"is supplied through the same seam 004 will
fill"*, and this listing had no seam to supply it through: `sort_key` is what §4.2 stores, and the
title is the **input** to the derivation that produces it (spec §3.7.3). A derivation with no way of
being reached is not a derivation, so the field is declared here, beside the record it belongs to,
rather than at 004 — and it is empty for everything 003 produces on its own.

**`ScanBatch` is the fourth record type and the listing above named it without declaring it.**
`ApplyScanBatch(ctx, batch ScanBatch) error` was the only mention of it, so T10 — which owes *the
four record types* — had to say what is in one. It is `{LibraryID, Items, ClaimedBy, At}`, and the
one field that is a decision rather than a transcription is **`ClaimedBy`**: §6.9 renews the claim
inside the batch's transaction, and a renewal that did not name the claimant would let a scanner
whose claim had already gone stale and been taken renew a claim it no longer holds — which is two
scanners writing one library, each believing it is alone. `At` is the instant the renewal records
and comes from `ports.Clock`, for architecture §2's reason. *(Added at T10.)*

**`ClaimScan` returns a boolean rather than an error for a library already being scanned.** Two
scanners over one store is a state this feature creates on purpose (§6.7: an operator may run
`atrium library scan` against a data directory a server is serving from), and *"somebody else is
scanning"* is an outcome the caller reports, not a fault. `staleAfter` is what breaks a claim left
by a process that died; §6.9 argues the value.

**And it returns the claimant it displaced or lost to beside the boolean.** ~~`(bool, error)`~~
became `(bool, string, error)` at T12, and what forced it is §7: two of that table's rows ask for a
message naming a claimant — *"the second reports 'already being scanned'"* and *"broken and taken,
with a log line naming the previous claimant"* — and neither name is recoverable after the call,
because the row now names the winner. A caller that read the row first would be naming a claimant it
had not necessarily displaced, and would be reading it outside the transaction that displaced it.
The string is empty when there was nobody to displace, which is the first scan of a library and
every scan after a rebuild. *(Amended at T12, which wrote the implementation. Same shape as T9's
amendment of `Reconcile` and T4's addition of `SortTitle`: a listing that had no way to produce
something the plan already required.)*

```
// internal/library — no os and no net/http. It imports internal/ports for the
// four record types above and for nothing else: the direction is downwards. It
// declares exactly one interface of its own, TagSource, which 004 implements
// and §6.2 argues.

CollectionType string                      // Movies | Shows | Music, from the three spec §3.1 names
func (CollectionType) Admits(ext string) bool

Normalise(path string, caseSensitive bool) (string, error)   // §6.3; refuses absolute and climbing
DeriveID(libraryID string, kind Kind, key string) string      // §6.3

Reading struct { Root int; Entries []Entry }   // what a walk saw, sorted
Entry  struct { Path string; Size int64; ModifiedAt units.Time }

Tags struct { AlbumArtist, Artist, Album, Title string; Track, Disc *int }
TagSource interface { TagsFor(root int, path string) Tags }   // §6.2; 004 implements it
NoTags struct{}                                               // the one v1 ships; answers nothing

Resolve(lib Library, readings []Reading) (Plan, error)        // §6.2; pure, and NoTags
ResolveWithTags(lib Library, readings []Reading, TagSource) (Plan, error)
Plan   struct { Items []ports.ScannedItem; Unplaceable, Skipped []Note }
SortItems(items []ports.ScannedItem)          // root ordinal, then path, then identifier

SortKeyBase(name string) string                               // §6.6, spec §3.7.1
SortKeyFor(item *ports.ScannedItem) string                    // §6.6, spec §3.7.2 and §3.7.3
```

**`TagSource` is the one interface this package declares, and it is declared here rather than at 004
for the reason `SortTitle` is.** §6.2 requires the source to be consulted once per file *before*
grouping, because the album artist decides which album a track belongs to; a seam that 004 has to
reach past would mean 004 rewriting the grouping rather than supplying it. `Resolve` is
`ResolveWithTags` with the `NoTags` source, so the fallback path and the tag-driven path are the
**same code with one different collaborator** rather than two behaviours with one of them tested.
It is asked only for a `music` library: §3.3's and §3.4's names come from the path, and what 004
replaces on those it replaces through `ScannedItem.SortTitle` and its own metadata pass.

*(Added at T7, which wrote the music resolver. The listing above had no seam at all, and the comment
heading it said the package declared no interface — a claim the block would have contradicted the
moment §6.2's requirement was implemented. Same shape as T4's addition of `SortTitle`.)*

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

Walk(fsys fs.FS, rootOrdinal int, collection library.CollectionType) (Result, error)   // §6.1
Result struct { Reading library.Reading; Skipped []library.Note }

Reconciliation struct {
    Write                     []ports.ScannedItem   // added and updated, ancestors before what hangs from them
    Remove                    []string              // file-backed rows only; see §6.5's closing paragraph
    Added, Updated, Unchanged []string              // a partition of the desired set
    Retained                  []string              // the containers the removal pass declined to remove
}

IdentifierMismatchError struct { Root int; Path, Stored, Derived string }   // §6.4

Reconcile(previous, desired []ports.ScannedItem, full bool) (Reconciliation, error)

Changes struct { Added, Updated, Removed []string; Examined, Skipped, Unplaceable int }
                                          // spec §3.8's summary, assembled by the scan of §6.9:
                                          // three of the six fields are a Reconciliation's and the
                                          // other three are the walk's and the resolver's counts
func (Changes) Document() ([]byte, error) // what ReleaseScan stores and --format json prints

New(Config) (*Scanner, error)             // Items, Clock, ClaimedBy, StaleAfter, BatchSize,
                                          // Logger, Tags — the last four have defaults
func (*Scanner) Scan(ctx, ports.Library, Options) (Changes, error)   // §6.5's order, §6.9's batching
Options struct { Full, AllowEmptyRoot bool }

AlreadyScanningError struct { LibraryID, LibraryName, ClaimedBy string }   // §7, ErrAlreadyScanning
UnavailableRootError struct { LibraryID, LibraryName string; Root int; Path string; Err error }
EmptyRootError       struct { LibraryID, LibraryName string; Root int; Path string; PreviousFiles int }
```

*(The block below `Changes` was added at T13, which wrote the scan. §5 declared the summary and
never the thing that produces it, so `Scan`'s signature and the three refusals were this task's to
choose. Two of them are decisions rather than transcriptions.* **`Options` is a record and not two
booleans in the signature**, *because §6.7's subcommand grows verbs and a third flag added as a
third parameter is a call site every caller has to be found and edited at. And* **every error names
the library**, *which no signature can require and which §7's whole audience — somebody with a
shell, looking at four libraries — depends on: `UnavailableRootError` and `EmptyRootError` also name
the root's ordinal and its configured path, because an operator with three roots configured should
not have to count.)*

**`Reconcile` is a pure function over two item sets and is where every removal in this project is
decided.** It takes no store, no filesystem and no clock. `full` is spec §3.8's re-examination,
which changes only whether an unchanged signal is believed. Its output is the batch to write and the
identifiers to remove, and the guards of §6.5 run on the reading **before** it is called, so a
partial reading never reaches it.

*(Amended at T9, which wrote it, and the amendment is one shape rather than three.* ~~`(Changes,
[]ports.ScannedItem, []string)`~~ *became a record and an error.* **A disagreeing identifier is an
error**, which §6.4 already required and which the listing had no return value for; a fourth return
value beside three would have been the point at which a caller stops reading them. `Removed` and the
third return value were the same list under two names, so the record holds it once. And two lists
the listing did not have are what keep this feature's two most dangerous assertions from being
assertions about an **absence**: `Unchanged` says a row was believed rather than merely not written,
and `Retained` names the container the removal pass looked at and declined to remove — where
"missing from `Remove`" is also what a build that removed nothing at all produces. `Changes` stays
where it was, as §3.8's summary, and the comment beside it says who fills which half. *`SortItems`
is exported in the same change, and for the reason the listing above it exists at all: the batch a
store is handed is in the order every other item set in this feature is in, and an ordering rule
spelled in two packages is two rules that will disagree.)*

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
| An entry that is not a regular file once followed | The walk, per entry | Added at T8; see below |
| A file whose size changed between two passes | §6.4, not here | It needs two readings, and the walk has one |

**Amended 2026-09-05 at T2: the extras row said *"per file and per directory"* and that is two
decisions, not one.** Writing the predicates forced both, and each is the reference's own shape read
at the pinned tag rather than a choice this project had:

- **The folder name is matched against the *immediate* containing directory and no ancestor.** The
  reference compares its token against `Path.GetFileName(Path.GetDirectoryName(path))`
  `[source: Emby.Naming/Video/ExtraRuleResolver.cs:35,51 @ v10.11.11]`, so a file one level below an
  extras folder — `Extras/Making Of/clip.mkv` — is an item there. Walking every component instead
  would be an item this server lacks and the reference has, over a shape no measurement covers and
  no allowlist declares. The reference also refuses the rule when the directory *is* the library
  root `[source: Emby.Naming/Video/ExtraRuleResolver.cs:52 @ v10.11.11]`; here that costs no code,
  because a root-relative path for a file directly under the root has no containing component. The
  walk therefore **descends into an extras directory** and refuses its files, rather than pruning
  the subtree — which is the one place the row's *"per directory"* would have been read the other
  way.
- **The rules apply under `movies` and `tvshows` and not under `music`, and that is a media-type
  gate rather than a preference.** Every rule v1 implements is declared `MediaType.Video`, and the
  reference skips a rule whose media type the file does not have
  `[source: Emby.Naming/Video/ExtraRuleResolver.cs:40-44 @ v10.11.11]`. Every extension `movies` and
  `tvshows` admit is in the reference's `VideoFileExtensions`; every extension `music` admits is in
  its `AudioFileExtensions` and in neither the other
  `[source: Emby.Naming/Common/NamingOptions.cs:24-80,213-295 @ v10.11.11]`. So the collection type
  decides exactly what the media type decides, and the one term this feature has expresses the whole
  gate. The consequence is that the reference's two `MediaType.Audio` tokens — `theme` as a whole
  filename and `theme-music` as a directory name — are **not** implemented: under `movies` and
  `tvshows` they would change nothing, because §3.2's lists admit no audio extension and refuse such
  a file first; under `music` implementing them would be behaviour this project owns outright on a
  source reading with no measurement behind it. It is an accepted shortfall in the direction that
  shows more rather than less, and the fixture carries no file it can reach.

All of that is a **source reading and not a measurement** — no probe in this project has sent a file
named for an extra — so what the tests assert about extras is what the reference is written to do.

**The last row is the one that moves.** Spec §3.2 lists *"files being written, detected by size
change between two passes"* among the ignore rules, which reads as a property of a file. It is a
property of a **pair of scans**, so it is decided in §6.4 where both readings exist: a file whose
recorded size ~~and~~ **or** modification time moved since the last scan is re-read as an update, and
a file examined twice **inside one scan** is not something a single walk performs.

*(Amended at T9, which landed the row.* **`and` was wrong and `or` is what the specification says**
*— spec §3.8's table reads "File modified (size or time of change)" and §6.4's own table reads "Size
or modification time moved", so this sentence was the only place in the plan asking for both. Under
`and`, a re-encode that keeps the length and a restore that keeps the time are each invisible, which
is precisely the pair T9's task line requires be varied independently; the clause is unsatisfiable
under a conjunction. Nothing was built on the wrong reading — the function reads §6.4's table — and
the correction is recorded rather than tidied away.)*

**And the row lands as an update, never as a refusal**, which is the half *"decided in §6.4"* left
implicit. A file whose size moved between two readings becomes an item carrying the **new** size;
nothing anywhere skips it. That is the narrower behaviour above stated as what the code does: an
item that appears and then vanishes again while a copy runs is worse for an operator than an item
whose size is briefly wrong, and the second corrects itself on the next scan with no rule to write. What v1 does is
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

**Amended 2026-09-05 at T8, with the two things writing the walk had to decide that the table
above did not.**

- **A directory entry that is not a regular file**, which spec §3.2 does not mention and a walk
  cannot avoid taking a position on. A symbolic link is the case that matters: the reference reads a
  library through a filesystem that follows one, so a linked film is an item there, and refusing one
  here would show **fewer** items than the reference — the unsafe direction for a scanner. So the
  walk stats through the link rather than reading the directory entry, and a link to a file is the
  file it points at, size included. What is still not a regular file afterwards — a link to a
  directory, a device node, a socket — is refused, and so is a link pointing at **nothing**, which is
  refused rather than raised: a dangling link, or a file moved between the directory read and the
  stat, is a race a walk of a live tree really has, and failing a library's whole scan over one would
  mean a download completing during a scan costs the operator every item in that library. A `.ignore`
  that is a directory rather than a file excludes nothing and is not an error either, for the same
  reason in the same direction.
- **The `.ignore` search is implemented as a pruned subtree rather than as a search up the
  ancestors**, and the two are the same answer: a file is excluded exactly when some directory
  between it and the root carries an excluding marker, which is exactly when the walk stopped before
  reaching it. The bound at the library root then costs no code at all, because `fs.FS` from
  `os.DirFS(root)` has nothing above the root to look at. The marker at the root **does** apply,
  which is the inclusive end of *"up to the library root"* and is asserted beside the divergence so
  that a build with the boundary off by one fails exactly one of the pair.

**And one thing the plan asked for that a test cannot get from where the plan asked for it.** T8's
determinism clause is *"two walks over trees whose entries were created in opposite orders"*, and
**creation order cannot vary anything a walk of a real tree sees**: `os.DirFS` implements
`fs.ReadDirFS`, its `ReadDir` sorts, and `fs.WalkDir` therefore reads one tree in the same order
whichever way round it was built. Asserting it that way would be satisfied by the standard library
and would survive the removal of every line this feature owns — 003 T6's finding a second time, in a
different package. The order is varied where it can reach the walk instead: an `fs.FS` whose
`ReadDir` answers backwards, which `fs.ReadDir` hands straight through without sorting. The walk's
own sort is what the assertion then pins, and removing it turns the test red.

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

**Amended 2026-09-05 at T5, which wrote the film resolver.** Six decisions the specification
constrains and does not take, each forced by writing the code and each recorded here rather than in
`spec.md`, because every one of them is a *how*:

- **The title and the year are one rule and the year runs first.** Reading the reference at the
  pinned tag, the year is taken out and **everything after it is discarded with it**, the **last**
  year in the name wins, the year must be preceded by a separator and by at least one character of
  title, it must not touch another digit, and it must not be a date's leading year
  `[source: Emby.Naming/Common/NamingOptions.cs:147-151, Emby.Naming/Video/VideoResolver.cs:88-96 @ v10.11.11]`.
  Two of those are worth a sentence each. *Everything after it* is what makes
  `The.Film.2019.1080p.BluRay.x264` a 2019 film called `The.Film`. *The last one* is what makes
  `Blade Runner 2049 (2017)` a 2017 film called `Blade Runner 2049` rather than a 2049 film called
  `Blade Runner`. And the date guard is what keeps spec §3.4's `The Daily Show - 2024-01-31` from
  acquiring a production year.
- **The order matters and is asserted as a pair.** Tags before the year answers `Some Film` with
  **no year** for `Some Film DVDRip 1999`, because the year is behind the tag. `The.Film.2019.…`
  cannot see that, and it was the corpus row this project first wrote for it.
- **The release-tag vocabulary is transcribed as data; the expressions are not transcribed at all**
  `[source: Emby.Naming/Common/NamingOptions.cs:153 @ v10.11.11]`. That is the same treatment
  §6.1's extras folder names and extras suffixes already get, and Principle IV's line is between a
  list of facts and a program. Of the reference's six clean-string expressions this implements the
  three spec §3.3 names — a delimited tag, a trailing release-group bracket, a leading one — and
  the other three have owners: an episode range and a trailing number are §3.4's, and the extras
  suffixes are §3.2's and shipped at T2. A token containing a separator (`blu-ray`, `read.nfo`) is
  matched **literally** where the expression's `.` matches any character, which refuses a name the
  reference would have cleaned rather than cleaning one it would have left alone.
- **A library root never names a film.** Spec §3.3's rule is *"where a film sits in its own
  directory"*, and a library holding exactly one film directly under its root satisfies a naive
  reading of it. The root is the library — §4.2 gives the library's own row no path because a
  library may have several roots — so the rule is asked of the **immediate containing directory**
  and never of the root, which is the same shape §6.1's extras rule already takes. The year comes
  out of the directory's name with the title when the directory names the film.
- **Stacking needs three guards and a floor**, all the reference's
  `[source: Emby.Naming/Video/StackResolver.cs:76-127 @ v10.11.11]`: files stack only within one
  directory; the **first** file establishes the marker word and whether the stack is numeric or
  alphabetic, and a later file disagreeing with either stands alone; a repeated part number joins
  nothing; and **a stack of one is not a stack**. The floor is the one that is easy to leave out and
  it changes a name: a lone `The Film - cd1.mkv` keeps its whole stem, which the release-tag rule
  then cleans, because `cd[1-9]` is in that vocabulary and `part1` is not.
- **An item's `Path` is the path the walk read and not the key its identifier came from.** They
  differ exactly when the library is case-insensitive, and a stored path that had been folded cannot
  be opened on a case-sensitive filesystem. `ports.ScannedItem.Path`'s own comment said the wrong
  one of the two and is corrected in the same change.

**Amended 2026-09-05 at T6, which wrote the series, season and episode resolver.** Seven decisions
the specification constrains and does not take, every one of them a *how* and therefore recorded
here:

- **The three levels come from three different places, and only the middle one is negotiable.** A
  series is the **first path component** and is named by that directory, as a film in its own folder
  is (§3.3); an episode is a candidate file; and a season is whichever of three sources answers
  first. A candidate directly under a library root has no series directory, so its series name is
  the reference's own `seriesname` capture — everything in the stem before the numbering, trimmed of
  whitespace, `_`, `.` and `-`
  `[source: Emby.Naming/Common/NamingOptions.cs:324, Emby.Naming/TV/EpisodePathParser.cs:85-88 @ v10.11.11]`.
  Where even that is empty the item is **unplaceable** and hangs from the library's own row.
- **The season number's three sources, in order**, which is spec §3.4's *"resolved by position, not
  by preference"* made concrete: the filename's own numbering; failing that the containing
  directory's name, but only a directory **below** the series; failing both, season 1 when the name
  gave an episode number or a date and there was no season directory to ask
  `[source: Emby.Server.Implementations/Library/Resolvers/TV/EpisodeResolver.cs:78-82 @ v10.11.11]`.
  Failing all three there is no season and the episode's parent is its series.
- **A series directory is never also a season directory**, and this is where §3.4's `24` really
  bites. `24` cleans to a name that is nothing but digits, which is exactly the shape the reference
  reads as a numeric season folder
  `[source: Emby.Naming/TV/SeasonPathParser.cs:88-92 @ v10.11.11]`; the reference never asks,
  because its season parser only runs on a directory whose parent is a `Series`
  `[…/SeasonResolver.cs:45 @ v10.11.11]`, and neither does this.
- **A season's name comes from its number and never from its directory**: `Season 01` on disk is an
  item called `Season 1`, and zero is called `Specials`
  `[source: Emby.Server.Implementations/Library/Resolvers/TV/SeasonResolver.cs:85-91 @ v10.11.11]`.
  That is what lets an inferred season and a directory-backed one read identically in a client. A
  season's **path** is filled only by a directory whose parsed number is the number the season
  ended up with — a `Season 05` folder holding a file that says `S01E01` gives season 1 nothing.
- **The numbering family is the specification's and not the reference's whole list**, and the
  shortfall is declared rather than discovered: `S01E02` and its separators, `1x02`, `E02`/`EP02`
  and date-based naming are implemented; the reference's optimistic bare-number expressions,
  `Episode 16` standing alone and the part and chapter forms are not, and an episode carries no
  production year where the reference takes one out of the filename. Every one of them places
  **fewer** episodes here than there, and §3.8 counts an unplaceable item apart from a skipped file
  precisely so an operator can see which. None is exercised by the fixture tree.
- **A multi-episode range needs the reference's two cases and not "a hyphen or a letter".** A bare
  hyphen may stand alone; a spaced ` - ` requires the letter
  `[source: Emby.Naming/Common/NamingOptions.cs:754-765 @ v10.11.11]`. The loose reading makes
  `24 - S01E01 - 12-00 AM` **episodes 1 to 12** — one item still, season 1 still, episode 1 still.
  And the ending number is refused when a digit, a `p` or an `i` follows it, which is the
  reference's own guard against reading `s09e14-1080p` as a hundred episodes
  `[source: Emby.Naming/TV/EpisodePathParser.cs:154-165 @ v10.11.11]`.
- **`Specials` is season zero and `Extras` is not**, where the reference's season parser accepts
  both `[source: Emby.Naming/TV/SeasonPathParser.cs:81-86 @ v10.11.11]`. Under §3.2 an `Extras`
  directory yields no candidate, so nothing is ever placed in a season made of one, and a season
  nothing is placed in is never created — the same rule that leaves the fixture's empty
  `The Series/Season 03` with no item. Implementing the second alias could therefore change exactly
  one thing: it could let an `Extras` directory beside a `Specials` one supply season zero's
  **path**. So the alias implemented is the one §3.4 states.

**And the finding, which is T5's for the third time in this feature.** The task list says the
fixture's `24/Season 01/24 - S01E01 - 12-00 AM.mkv` is *"built to catch exactly that"* — a resolver
matching the directory before the filename. **It is not**, and two rules repair the mutation: the
directory is `Season 01` and says 1 as loudly as the filename does, and in the `24` tree that has no
season directory there is no directory below the series to match first at all. Only a tree where the
two sources **disagree** catches the order, so the assertion is made over a `Season 05` directory
holding a file whose name says `S01E01`. What the fixture path does catch is the other half of
§3.4's sentence — that a series' own title is not read as a number — which is a different mutation
with a different killing test. §8.5 records it beside T5's own.

**And one thing `Resolve` refused rather than answered, until T7.** ~~A collection type whose
resolver is not written yet is an **error**, never an empty plan.~~ The reasoning stands and the
branch is gone: an empty plan is the answer *"this library holds nothing"*, and §6.5's guards run on
the reading rather than on the plan, so a caller would reconcile it against what the library holds
now and take every item away — Principle VI's *"no plausible-looking stub"*, at the one place in
this feature where the stub would be silent and destructive. **All three resolvers are written now,
so the refusal is a branch nobody can reach, which is the same stub one layer up**, and T7 deleted
it along with `ErrCollectionTypeNotResolved`. What replaces it is an assertion that can fail: every
type `AllCollectionTypes` names resolves a file of its own admitted extension into an item beyond
the library's own row, so a fourth type added with no arm in the switch fails a test rather than
quietly emptying a library.

**Amended 2026-09-05 at T7, which wrote the music resolver.** Six decisions the specification
constrains and does not take, plus one it contradicted itself about, every one of them a *how* and
therefore recorded here:

- **The three levels come from the directories, and a disc directory is not one of them.** The album
  is the file's containing directory, or the one **above** it where the containing directory names a
  disc; the artist is the directory above the album; and a candidate directly under a library root
  has neither. A folder holding audio directly under a music root is therefore an **album** and not
  an artist, which is the reference's own shape — its album resolver checks for audio files before it
  descends into anything
  `[source: Emby.Server.Implementations/Library/Resolvers/Audio/MusicAlbumResolver.cs:128-139 @ v10.11.11]`.
- **The disc vocabulary is the reference's `AlbumStackingPrefixes` and is not the film-stacking one**
  `[source: Emby.Naming/Common/NamingOptions.cs:183-193, Emby.Naming/Audio/AlbumParser.cs:34-68 @ v10.11.11]`.
  The two differ in **both** directions: `dvd` and `pt` stack a film and name no disc, and
  `digital media`, `vol`, `volume` and `act` name a disc and stack no film. One shared list would be
  wrong for both. The rule around the vocabulary is transcribed as the rule it states — separators
  collapsed to one space, then a prefix, then a number — and every prefix is tried, because
  `Volume 3` matches `vol` first and the `ume 3` left behind is not a number.
  What the reference does with the answer and what Atrium does with it are **different things**:
  there it decides only whether a folder is a disc folder of a multi-disc album, and the number
  itself comes from the tag or the container `[…/AudioFileProber.cs:182,312 @ v10.11.11]`. Here it is
  both, and that second half is the declared divergence below.
- **The reference's further conditions on a multi-disc album are not implemented, and the shortfall
  shows fewer items.** There, a parent is a multi-disc album only when it holds no audio of its own
  and **every** music-holding subfolder is a disc folder; a stray file beside two disc folders makes
  the disc folders plain `Folder` items with the tracks under them. Here the disc rule is a property
  of the directory's own name, so that tree stays one album. Fewer items here than there, which is
  the direction §6.1's and §6.2's other narrowings already run in, and nothing in the fixture
  exercises it.
- **An album's name loses its year and loses nothing else.** Spec §3.5's table has a row for
  `Album (2001)` resolving to *"Album with a year"*, so the year comes out and becomes
  `ProductionYear`. The **release-tag vocabulary does not apply**: it is `names.go`'s, read out of
  the reference's *video* cleaner, and the reference takes nothing at all out of an album folder's
  name — its reading of this repository's fixture calls the album `First Album (2001)`
  `[probe: tools/probe_reference_scan.py, Jellyfin 10.11.11, 2026-09-02]`. Running an album's name
  through the film cleaner would widen a difference §3.5 asks for into one it does not. An artist's
  name is not cleaned at all; both are trimmed, which is §3.5's own word.
- **§3.5's compilation rule is implemented as its *"only if"* and not as its sentence.** *"Where no
  album artist is present, the album is attributed to `Various Artists` only if the track artists
  actually differ."* The track artists therefore **fill a hole and never overrule**: a tag outranks a
  directory (§3.5), a directory outranks an inference, and only an album that neither attributed
  consults them. The wider reading turns an ordinary album with a guest on every track into a
  compilation. Under the `NoTags` source there are no track artists at all, so the rule cannot fire
  and `Various Artists` is attributed only where a directory names it — which is what the fixture's
  `Various Artists/A Compilation (1999)` gives.
- **A track that no directory places is not unplaceable.** A candidate directly under a music root
  has no album and no artist and hangs from the library's own row, and it is **not** counted with
  §3.8's unplaceable items. The distinction is the one §3.8 is written for: an episode with no
  episode number is a file whose *name* failed to say something it had to say, where a track's name
  never had that job. So `music`, like `movies`, produces no `Plan.Unplaceable` note at all.
- **And the one the specification contradicted itself about: an album's identity.** §3.6's table put
  `MusicAlbum` with `Series` and `MusicArtist` under *"the library root plus the normalised name"*,
  which makes two artists' `Greatest Hits` **one item** — one row, one parent, half an album's tracks
  under an artist that did not record them. §3.5 settles it in terms — *"an album's identity comes
  from its album artist"* — so the table is amended and the album's key is its artist's identity plus
  its normalised name, exactly as the `Season` row above it already reads. This is a change to a
  **WHAT**, so it is in `spec.md` with its own `amended:` line and not only here.

And one consequence of the compilation rule running *after* the grouping, recorded rather than
engineered around: an album it attributes was grouped under the **empty** artist, so two groups can
derive one album identifier — one attributed `Various Artists` from its track artists and one whose
directory already said `Various Artists`, sharing an album name. That is the same shape §6.3 records
for two canonically equal filenames: a repeated identifier inside one batch, which `ApplyScanBatch`
has to decide the meaning of (T10) and which nothing in `internal/library` can see. Under `NoTags`
it cannot happen at all, because the track artists are what fill the hole and there are none, so it
is 004's to watch rather than this feature's to guard.

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

**Amended 2026-09-05 at T3, because writing the function took four decisions this row had left
open.**

- **NFC comes from `golang.org/x/text/unicode/norm`, and that is a dependency this plan argues.**
  [ADR-0002](../../docs/decisions/0002-go-and-the-runtime-stack.md#standard-library-first-and-chi-is-the-only-dependency-this-record-adds)
  says *"a further dependency is argued where it is needed, in the plan that needs it"*, and this is
  the place: the standard library has no normaliser, the tables are Unicode's own and not something
  to hand-roll, the package is pure Go and adds nothing to a `CGO_ENABLED=0` build, and it is the
  same repository `golang.org/x/crypto` already comes from.
- **The fold is `strings.ToLower` and not this package's `foldASCIICase`, which is T2's finding
  applied in the opposite direction.** The ASCII fold exists because the reference compares
  *extensions* ordinally; here `AMÉLIE` and `amélie` are one directory, and an ASCII fold would give
  them two identifiers — the exact loss spec §3.6's case rule exists to prevent. It is not a *full*
  case fold either: full folding maps `ß` to `ss`, and `Straße` and `Strasse` are two directories.
- **The first step reduces more than the separator character.** Runs of separators collapse, `.`
  elements disappear and a trailing separator is dropped, because each is a second spelling of one
  path and would otherwise be a second identifier. The absolute test is hand-rolled rather than
  `filepath.IsAbs`, which answers differently depending on the platform the binary was built for —
  `C:\Movies` must be absolute on every one of them.
- **`a/../b` is `b`, and only a `..` that leaves the root is the error.** That keeps
  `ErrPathClimbsAboveRoot`'s name true rather than approximate; the reduction is lexical, and the
  walk produces neither element anyway.

One consequence is worth writing down before a scan surprises somebody: **NFC is a canonical
equivalence with singleton mappings**, so U+212A KELVIN SIGN becomes `K` and a file named with one
is the same key as a file named with a plain `K` — *even in a case-sensitive library*. That is
correct, and a filesystem can still hold both files, in which case two files derive one identifier.
Nothing in §6.3 can notice it; the scan's own handling of a repeated identifier (§6.9's batch) is
where it would surface.

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

**Five things this table does not say and the code has to**, and two of the five were added at
T9 by writing the function.

**A record that moved is an update even when the file signal did not.** *(Added at T9, which wrote
the function; the section had three.)* This is a case the table does not cover rather than an
optimisation of one it does. A container has no file at all, so a series that was renamed, a season
whose parent moved, or the library's own row after `atrium library rename` has **no signal that
could ever move** — a build comparing only §6.4's signal would hold the old name for the life of the
installation and nothing would report it. It reaches one level down too: an album's parent is
derived from its album artist across all of its tracks (spec §3.5), so adding one track can change a
*sibling* album's record while every byte of its own files stands still. Both halves are therefore
compared, the record and the files, and the record comparison excludes exactly one field —
`ScannedItem.SortTitle`, the one with no column behind it (§5), which a row read back always carries
empty and which would otherwise report every item updated on every scan the moment 004 supplies one.

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

*(T9 had to say how the row is found, because the sentence above names one key and the decision needs
two.* **A desired item and a previous row are paired by identifier; the path is what the comparison
above is made over.** *)* Pairing by path alone loses a file whose *name* changed case in a folding
library — same identifier, different path, and a per-path pairing removes it and adds it back,
costing the user nothing visible and the store a delete and an insert of the same row. Pairing by
identifier alone can never see the disagreement at all: an identifier that changed is
indistinguishable from a rename, which spec §3.8 requires be a delete plus an add. So both indexes
exist, the pairing is by identifier, and the check is *"this path's stored identifier and this path's
derived identifier"* — which a rename cannot trigger, because a renamed file's previous row is at a
path the reading no longer holds.

**A full re-examination applies to an item that has a file.** *(Added at T9.)* Spec §3.8's
re-examination *"ignores the signal and looks at every file"*, and an item with no file has no signal
to disbelieve: forcing every artist, series and season to be rewritten would make a full scan report
a library's whole spine as updated on every run, for nothing. What `full` changes is therefore
exactly one decision on exactly the rows that have one, and every other row — added, updated,
removed, retained — is what the default scan would have said. Spec §3.8's *"the default is the fast
one, the full one is always available"* is untrue in the dangerous direction the moment the thorough
option is also a different set of deletions.

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

   **Amended at T13, which implemented it: the threshold is counted in *files* and not in items.**
   The section above says *"where the previous scan recorded at least one"* and leaves what is
   counted open, and the two answers are not the same. A library's own `CollectionFolder` row backs
   no file and neither does an inferred container, so a library an operator emptied on purpose —
   having said `--allow-empty-root` once and had every item under the root removed — still holds a
   row afterwards. Counting items would read that row as *"the last scan recorded one"* and refuse
   every scan of that root from then on, with the override as the only way to scan a library that is
   legitimately empty. Counting files makes the guard's question the same question the reading
   answers: *this root yields no candidate file and the last scan of it recorded some.*
3. **A removal is computed from a complete reading of every root of a library and applied ~~in one
   transaction with the additions~~ after every batch of them.** So a scan that was cancelled, that
   hit a failed batch, or whose second root failed while the first succeeded removes nothing at all.
   This is why `Reconcile` takes whole sets rather than streaming: a streaming reconciliation cannot
   tell *"not seen yet"* from *"not there"*, and the difference between them is a user's library.

   **Amended at T13: one transaction is not available and §5 is why.** The contract declares
   `ApplyScanBatch`, `RemoveItems` and `ReleaseScan` as three methods, so the removal is its own
   transaction and the release is another, in that order. What the guard is actually made of is the
   **ordering** rather than the atomicity: every failure this feature can reach happens before
   `RemoveItems` is called, so nothing partial is ever removed. What the sequence costs is one state
   the single transaction would not have had — a scan killed between the removal and the release has
   removed what it computed and left a claim to go stale, which the next scan corrects and which is
   why §6.9's staleness rule is load-bearing rather than a nicety. Making it one transaction would
   mean a port method that took the removals, the summary and the release together, and that method
   is the over-broad one `RemoveItems`' own comment argues against.

**What none of the three watches is one level down, and that is the whole of
[behaviours §5.2](../../docs/compatibility/behaviours.md#52-a-container-that-has-lost-every-file-is-not-removed).**
A *directory* that mounts empty inside a healthy root has no guard: the root is readable, the
library is not empty, and the reading is complete. So a series whose episodes all vanished is
correctly read as a series with no episodes — and the removal pass therefore marks **file-backed
items** removed and leaves the containers above them alone. *(T9: it says which ones it left, in
`Reconciliation.Retained`. A container missing from the removals is also what a build that removed
nothing at all produces, so the kept row is named rather than inferred from an absence.)* Removing the container would be spec
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

**Amended at T16, which asserted that risk from the other side.** ~~Nothing in this feature would
fail~~ — one thing does now, and it is the only thing that does. `internal/app`'s AC-11 test writes
a row into `item_user_data` (§4.1) through a store method, deletes the file, scans, and requires the
row to still be there; adding an orphan sweep to `RemoveItems` fails **that test and not one other
test in the repository** `[measurement: 003 T16, 7 mutations, 2026-09-05]`. The sentence above is
therefore still true of the *scanner* — no code here retains or sweeps anything — and no longer true
of the *suite*, which is the point: a later feature that wants a retention rule has to argue with
spec §3.8 in a failing test rather than discover the rule in an operator's library. The paragraph is
struck rather than rewritten because what it predicted is exactly what the measurement confirms.

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
A missing number contributes **no segment at all** rather than a run of zeros — which falls out of
the construction rather than needing a case, because the separator belongs to the segment: the
reference formats the number and its `" - "` with one format string. **Season's missing-number case
is the one that does not fall out**, since its whole key is the prefix; the source answers it with
the raw name `[source: MediaBrowser.Controller/Entities/TV/Season.cs:151 @ v10.11.11]`, spec §3.7.2
is amended to say so, and T4 implements it. It is not hypothetical: §3.4 infers seasons, and the
reference's recorded reading of this project's fixture tree contains a `Season Unknown`.

**§3.7.3 applies two of the six steps and not four**, which the specification's sentence implies and
the source settles: a forced sort name is digit-padded and diacritic-folded and then lowercased, and
is not trimmed, not article-stripped, and has neither character list applied to it
`[source: MediaBrowser.Controller/Entities/BaseItem.cs:535-536 @ v10.11.11]`. §3.7.3 is amended with
it. That is why the two derivations are written as two functions over shared steps rather than as
one function with a flag: the difference between them is not one clause.

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

**T4 wrote that test and found it does not pin the width.** `2 Fast` sorts before `10 Things` at pad
width 9, at 10 and at 11 alike — padding to any width of at least two already makes the shorter run
compare low — so the ordering assertion the paragraph above asks for is satisfied by three different
contracts. What pins the width is the byte-exact table, and the test now carries the demonstration
rather than the claim: it asserts the ordering at all three widths, asserts that the **bytes** differ
at all three, and asserts that at width one — no padding — the pair reverses, which is the only
mutation of the width the ordering alone can see. Same shape as 002 T22: a check that names the
right contract can still be about something weaker than its name.

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
                      [--format table|json] [--log-level LEVEL]
```

**`scan` and its two extra flags landed at T13 rather than at T14, and the reason is a criterion.**
T13's assertions are AC-12, AC-16 and the two seams of §8.1, and every one of them is about *what
the store holds after an operator ran a scan* — so the verb has to exist for them to be assertable
at all, and `--format json` has to exist for the summary's two counts to be read out of a document
rather than out of prose. `--log-level` is 001's own flag and its own environment fallback, reused
rather than reinvented, and it is what makes spec §3.8's *"files skipped **with the reason**"*
reachable: the counts are the summary and the per-path reasons are the progress, at debug. A
document holding every skipped path of a large library is not a summary. **`--name` matches on the
domain's fold** (`library.FoldName`, added at T13 for this), not on bytes, because §3.6 makes two
library names differing only in case one name and the store's unique index is enforcing that fold.
A name no library has is a **refusal** and not an empty run: an operator who mistyped and was told
*"0 libraries scanned"* would read it as *"nothing changed"*, which is the sentence a successful
scan of an unchanged tree produces. The other five verbs are T14's.

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

**Amended at T14, which wrote the other five verbs. Five decisions this section constrained and did
not take.**

- **What allocates a library's identity.** Spec §3.6 says it is *"allocated when it is declared and
  kept, rather than derived from its name or its roots"* and nothing said what allocates it. It is
  sixteen bytes from `crypto/rand`, rendered as thirty-two lowercase hexadecimal characters — the
  installation identity's own shape one package along, and for its reason: two libraries declared in
  the same second must not share an identity. **This is the one identifier in the feature that
  Principle VII's "never randomness" does not govern**, and the distinction is that this value is
  computed *once* and read back, where an item's is recomputed by every scan. Deriving it from the
  name would make `remove` followed by `add` a no-op, which is the opposite of what §3.6 makes it.
- **What folds a library's name**, which T10 recorded as undecided when it wrote `RenameLibrary` to
  take the fold as a parameter. It is `Normalise`'s last two steps in `Normalise`'s order — NFC, then
  simple lowercase — because §3.6 says *"normalised means the same thing for a path and for a name"*.
  A fold that only lowercased would let `Amélie` be declared twice, once precomposed and once
  decomposed: two libraries an operator cannot tell apart, one of which `--name` could never address.
  The store's uniqueness is over the stored bytes, so the column refuses exactly the pairs this fold
  folds together and no others.
- **What `remove` does to what was scanned.** It removes the items first and the library second.
  Nothing in the schema does this — `items.library_id` is a string and deliberately not a foreign
  key, because the derived half may not reference the precious one — so a `remove` that deleted only
  the library row would leave every row that library ever scanned behind for ever: unreachable,
  because every reader arrives through a library that no longer exists, and unrecoverable, because
  the next `add` allocates a different identity. **What does survive is the library's `scan_state`
  row**, and that is stated rather than hidden: no port method deletes one, adding one is an
  amendment to `ports.ItemStore`, and the row can never be read again because nothing will ever hold
  that identifier again.
- **A root is checked and made absolute where it is typed.** §7's first row says `add` refuses a root
  that does not exist or is not a directory, and `roots` refuses one the same way: refusing at
  declaration turns a mistyped mount into a message an operator reads while they are still looking at
  the command. The path is made absolute for `--data-dir`'s reason — the server that reads this
  installation was started from somewhere else — and symbolic links are **not** resolved, because an
  operator who configured a link configured the link.
- **`--format json` on `list`**, which this section already asked for, reports `id`, `name`,
  `collectionType`, `caseSensitive` and `roots` **in configured order**, with `[]` and never `null`
  for an empty list. The order is the assertion rather than a detail: `ScannedItem.RootOrdinal`
  indexes that list, so two roots that came back swapped would leave every item's recorded ordinal
  naming the root its path is *not* relative to.

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

**Discharged at T11 (2026-09-05).** `internal/store/sqlite/derived/library.sql` holds the whole
schema, `derivedGeneration = 1` is paired with its SHA-256, `Open` compares the recorded generation
against the constant after the precious lineage is applied and rebuilds on any difference in either
direction, and `ports.ItemStore.RebuildDerived` is the same act as something a caller can perform.
ADR-0003 keeps its wording and gains nothing: the record was never wrong, and what it owed is now
paid.

*(Amended at T11, in five places, and four of them are corrections to what the paragraphs above
predicted.)*

**(1) It cost *five* of 001's and 002's assertions, not four.** The four named above were found by
reading the runner; the fifth — `TestAFirstStartCreatesTheTwoLibraryTables`, asserting
`SchemaVersion(Derived) == 0` — was written by **T10 itself**, one task before this one, and was
named only in T10's handover. It is the same shape as the other two version literals and it is worth
one sentence of its own: *a list of what a change will break is written against the tree as it was
when the list was written*, and T10 landed between.

**(2) The five became one helper and not five spellings of `derivedGeneration`.** T10's own finding
was that *a correction that restates a literal is not a correction* — 002 T1 rewrote 001's `want [1]`
as `want 2`, which 003's third migration turned red on the day it landed — so the four call sites
that meant *"a start leaves the derived half where this build's schema puts it"* now call
`theDerivedHalfIsAtItsGeneration`, and a sixth caller costs one line rather than one more number.

**(3) The drop list is read out of the schema file, and that is what the "every object is dropped"
clause needed.** A list typed beside the schema is the failure the clause names — a table added to
`library.sql` and forgotten in the list survives a rebuild carrying its old columns — so
`derivedObjects` parses the `CREATE` statements and the drop runs in reverse of them. The test
refuses to compute the object set the same way: it reads `sqlite_master` and subtracts what the
precious lineage creates, so a parse that missed a table would not also miss it in the assertion.

**(4) `foreign_keys(1)` across the drop does not fail the way this section assumed, and the
assertion had to be built rather than found.** Measured: a derived table declaring
`REFERENCES libraries(id)`, holding rows, in a database whose `libraries` holds rows, with
`PRAGMA foreign_keys` reading 1, **drops without complaint** — `DROP TABLE` performs an implicit
`DELETE` of the *child* rows, and deleting a child violates nothing
`[measurement: modernc.org/sqlite v1.58.0, Go 1.27.1, 2026-09-05]`. So *"the rebuild refuses"* is not a
thing that happens, and a test that only rebuilt and expected an error would have been a green
proving the rule unbreakable when it is not. What bites is reading the constraint: every foreign key
the derived schema declares is listed by the engine, and every target of one must itself be a derived
object. Two clauses beside it keep that honest — foreign keys are still **on** after a rebuild, and
still **enforcing** over the freshly created tables — so a rebuild reaching for
`PRAGMA foreign_keys = OFF` around its drop fails, which is the shape that would have made the check
meaningless.

**(5) The derived schema declares no index, and the drop already handles one.** §4.2 names three
tables and no index; adding one speculatively would be a query-shape decision taken a task before the
queries (T12) exist. `derivedObjects` recognises `CREATE INDEX`, `VIEW` and `TRIGGER` all the same,
tested over a synthetic schema, so the first index added is dropped without anyone remembering to
arrange it.

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

**A scan claims its library — and it takes the claim *after* the reading, which this paragraph's own
argument for `staleAfter` is what forces.** *(Placed at T13, which assembled the scan. The section
said when a claim is renewed and never when it is taken, and the two available answers are not
interchangeable.)* The claim is renewed on every committed batch and by nothing else, so nothing
renews it during a walk; a claim taken before the reading would therefore have to outlive the walk
of the largest library an operator has, which is exactly the guess the paragraph below refuses to
make. Taking it after the reading and before the first write keeps `staleAfter` a number about a
batch. Two consequences, both stated rather than discovered: two scanners may walk one library at
once and only one of them writes, which costs a walk and nothing else; and a reconciliation computed
against a reading another scanner has since changed is **refused rather than applied**, because
every identifier in `Remove` came from this store and one that now matches no row fails
`RemoveItems`' rows-affected check. It also means a refusal by either guard leaves no claim behind
at all, which is what lets an operator fix a mount and scan again immediately rather than waiting
out a claim their own failed scan left.

`ClaimScan` reads the `scan_state` row and writes
`(claimed_at, claimed_by)` on it, and reports whether it won. ~~In one conditional statement~~ — the
read and the write are **one transaction** instead, and T12 amended this when it wrote them: an
upsert's `RETURNING` answers the row as it now stands, so the previous claimant the two §7 rows
above want named has already been overwritten by the time it could be read. The transaction takes
SQLite's write lock at `BEGIN` (`_txlock=immediate`, ADR-0003's writer DSN) rather than upgrading
part way, and runs on the writing handle, which is capped at one connection — so the atomicity is
the statement's and the only thing that changed is that the old value survives long enough to be
returned. A claim stamped *after* the instant offered is a clock that moved backwards and is treated
as **live**: breaking a claim on the strength of a clock adjustment is the one outcome this method
must not produce. A claim older than `staleAfter` is broken
and taken, because a process killed mid-scan leaves one behind and the alternative is a library
nothing will ever scan again. `staleAfter` is **not** a guess about how long a scan takes: the claim
is **renewed on every committed batch**, so the value only has to exceed the time between two
batches. That is what makes it defensible rather than a number nobody could argue for later, and it
is why the renewal is part of the batch's transaction rather than a timer beside it.

**Batches are sized in items, not in time**, and each is one transaction that writes the additions
and updates it holds, renews the claim, and commits. Removals are **not** batched: they are computed
once from the complete reading and applied ~~in the final transaction (§6.5's third guard), which is
also the transaction that releases the claim and writes the summary~~ **after the last batch, in
their own transaction, with the release after that** — §6.5's third guard as amended at T13, where
the reason the contract cannot make it one transaction is written down. So a scan that dies half way
has added and updated some items and removed none, which is a state the next scan corrects, and it
is the *only* partial state this feature can leave behind. **The batch size is
`scan.DefaultBatchSize`, five hundred items**, which is two orders of magnitude inside the
measurement below rather than a round number.

**Readers are not blocked.** ADR-0003 measured it: 57,664 reads completed during one write
transaction inserting 30,000 rows, worst read latency 393 µs
`[measurement: modernc.org/sqlite v1.58.0, Go 1.27.0, 2026-09-02]`. The batching is what keeps that
measurement relevant rather than a hope about a transaction held open for a whole tree.

**The schedule.** Spec §2 puts filesystem watching out of scope and says v1 rescans *on demand and
on a schedule*. The schedule is a process setting under
[architecture §9](../../docs/architecture.md#9-configuration-identity-and-logging) — `--scan-interval`,
with an environment fallback, a default measured in hours, and `0` to disable — and every scheduled
scan is **incremental**. It is bound to the server's own lifetime and cancelled by shutdown, and a
scan cancelled that way ~~releases its claim on the way out~~ **leaves its claim to go stale**, which
is what makes the graceful-shutdown rule of architecture §5 load-bearing for this feature as well as
for 008's children.

**Amended at T14, which wrote it, in three places.**

- **The default is twelve hours, and it is the reference's own number for the same task**
  `[source: Emby.Server.Implementations/ScheduledTasks/Tasks/RefreshMediaLibraryTask.cs:47-54 @ v10.11.11]`.
  *"A default measured in hours"* left the hour to be invented; an operator who has run the
  reference has already decided what a library scan costs them. `0` is a **setting** and not an
  absence — it says this server never scans on its own, which an operator scanning from a cron entry
  means — and a negative duration is neither that nor a schedule, so it is refused at parsing.
- **A cancelled scan does not release its claim, and the strike above is what writing it found.** A
  claim is released by the scanner reaching the end of its own `Scan`, and a cancellation is
  precisely the exit that does not reach one. What a cancelled scan leaves is what §7's *"a batch
  fails to commit"* row already describes — a claim left to go stale, bounded by `staleAfter` — and
  the alternative would write a summary document describing a scan that did not happen. The clause
  was not a small error: it is the only sentence in this plan that promised something the design
  cannot do, and a reader would have looked for the release rather than for the staleness.
- **A cancellation bounds the *next* root and not the current one.** `Walk` takes an `fs.FS`, and
  `io/fs` offers no context, so what a stop can interrupt is the loop over a library's roots.
  `Scanner.read` therefore consults the context once per root, and a stop waits out one root's walk
  and no longer. Nothing is written either way: every failure at that point is guard 1's, and guard
  3 is the ordering that keeps a removal behind it.
- **And this section, with §3 and §6.8, was owned by no task at all.** The scheduled scan and the
  start-time rescan a rebuilt derived half owes are named in §3's module table, specified here and
  in §6.8, and appear in no task of the list written from spec §3 and §5 — because the schedule
  appears in the *specification* only in §2's scope note, beside filesystem watching, and the rescan
  only in ADR-0003. Both are taken at T14, which amended the task list to say so. **A behaviour named
  only in a scope note is a behaviour no task list derives**, and this is the second time this
  feature has found a rule with no criterion under it (AC-16 was the first).

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
| A batch names one item twice | `ApplyScanBatch`, before it opens its transaction | The library's scan fails, naming the identifier and one of the two paths. §6.3's canonical equality is how two files reach one identifier | A bug report, or a file renamed on disk |
| A removal names an identifier no row holds | `ApplyScanBatch`'s sibling `RemoveItems`, on rows affected | The whole removal is refused and nothing is removed | Next scan. The identifiers came from this store, so one that matches nothing is a caller holding another library's reading |
| Two scanners on one library | `ClaimScan` returns false | The second reports *"already being scanned"* and exits non-zero. Not a fault | Wait, or `--full` later |
| A claim left by a dead process | `ClaimScan`, on age | Broken and taken, with a log line naming the previous claimant | — |
| The derived generation differs from the build's | `Open` | Drop, recreate, and enqueue a full scan of every library (§6.8) | Automatic |
| The derived schema cannot be recreated | `Open` | **Refuse to start**, naming the file. The precious half is untouched | Operator removes the database, or restores it |
| A precious migration fails | `Open` | Refuse to start — 001's rule, unchanged | — |
| A recomputed identifier disagrees with the stored one | §6.4 | The library's scan fails, naming both. **Never a rewrite** | A bug report; a rewrite would be the silent discard Principle VII forbids |
| `library add` names a folded name that exists | The unique index | Refuse, and say which library holds it | Choose a name |
| `library add` names a collection type that is not one of three | Flag parsing | Refuse, listing the three | — |
| An attempt to change a frozen column | `rename`/`roots` do not offer it; there is no verb | No verb exists to refuse (spec §3.6) | Remove and add, knowing what it costs |
| A verb names a library that does not exist | The lookup on the folded name | Refuse, naming what was typed. **Never a run that changed nothing** — an operator who mistyped `remove` and saw nothing said would believe the library was gone | Check `library list` |
| `--scan-interval` is negative, or is not a duration | Flag parsing | Refuse, naming the flag. `0` is a setting and means *never scan on my own* | — |
| A scheduled scan fails | The scheduler | **Logged and not returned**: there is no operator reading an exit status, the next tick is the retry, and one library's missing mount must not stop the others. A failure while the server is stopping is logged at debug, because a stop is not a fault | Next tick |

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

*(Done at T13, and the shape is worth recording because the second seam needed something the first
did not.* **A seam test needs a corpus in which the wrong answer is not also the right one.** *The
parent seam is asserted over the fixture's whole declared parent-child structure rather than over
one chain, because the mutation that matters — every parent becomes the library's own row — is
invisible wherever a parent happens to **be** the library's own row, which is most of a `movies`
tree. The removal seam needs **two** libraries that both hold items, because with one library the
removal landing on "the wrong library" lands on the right one; the assertion is that the other
library's identifiers are unchanged one for one, not that its count is, because a count passes on a
build that removed one row and added another. Both mutations were run and both go red
`[measurement: 003 T13, 26 mutations, 1 declared survivor, 2026-09-05]`.)*

**Two of T13's clauses are asserted at the `scan` level and not through the subcommand, and that is
a limit rather than a preference.** The batching and *"nothing renews a claim outside a batch"* are
properties of what the scan **asks** a store to do, and a scan driven through the subcommand builds
its own SQLite store with nowhere to stand between two of its transactions. So they are asserted
against a recording store that models exactly the two guarantees T12 already measured of the real
one — a batch renews the claim and writes its items, or neither — and what that buys is the half no
store can hold: a build that renewed the claim beside each batch leaves it at the *failed* batch's
instant rather than at the last committed one, and is red. The real store, the real tree and the
real refusal are still exercised together at `app` level by the one tree `internal/library` does not
object to and the store does: two filenames that differ in bytes and derive one identifier (T3's NFC
singleton finding), where the scan fails, names the library, and removes nothing because the removal
is after every batch.

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

**Amended 2026-09-05 at T17, by writing the declaration this section sized — and the count it
derived is thirty-two, not forty-seven.**

The declaration is `declaredDifferences` in `internal/app/reference_reading_test.go`, beside the
comparison, exactly as this section decided. It runs in the default job with no Jellyfin anywhere
([AGENTS.md §1.6](../../AGENTS.md)), over type, name and path and never over an identifier, and the
two clauses that make it a declared inequality are each proven by a mutation rather than asserted:
removing any one of the thirty-two rows reports that difference as undeclared, and declaring one the
two readings do not have reports it as gone away. Two real behavioural mutations were run as well,
end to end through the subcommand and the store
`[measurement: 003 T17, 2 mutations, 2026-09-05]`: a track keeping its leading number takes **ten**
declared differences away and the run goes red; a film keeping its year takes two away, changes the
shape of a third, and adds **thirteen** undeclared ones.

**The count, and it is a finding rather than a miscount.** The arithmetic is in the file's own
comment and in [conformance.md](../../docs/compatibility/conformance.md#l3--differential):

| | rows |
|---|---|
| Declared over `Movies`, `Shows`, `Music` and `Empty` — the four libraries `internal/libraryfixture` builds | **32** |
| Predicted over `Films` and `Tunes` if 008's media world were built: their two root rows, five folder-per-film titles that keep their year in the reading, and the album the reading calls `The Album` under a directory called `Untitled Folder` | 8 |
| Not derivable from the recorded reading and this project's specifications | 7 |
| [010's D-7](../010-conformance-harness/spec.md) count over all six libraries | 47 |

The eight are predictions and **not declarations**: a row no run can reach is a comment, not a
declared inequality, and this feature cannot build the tree they are over. The seven are the
interesting number. This section says in terms that the project *"has the reading"* and *"does not
have the declaration"*, and the reading plus this project's own specifications produce thirty-two —
so the forty-seven, counted against the **other** implementation over both fixture worlds, is a
place where two implementations of one specification differ over one tree. **Manufacturing seven
rows to reach it would be the failure the count assertion exists to prevent, one direction round**,
which is the same argument §8.5 made about not building U-44's pair. The total is asserted from the
declaration's own length, so a row deleted to make a run go green is a failing count.

**Twenty-three of the thirty-two are 004's** — every difference that is about *which name an item
carries*, since a title is metadata's product and §3.3, §3.4 and §3.5 supply the path-derived
fallback. Twenty of those are 010's *"twenty-five files named differently"*; the other three are the
two album directories and the series the reading calls `tvshow`. The other nine are 003's and are
about which items exist and what type they are: four libraries' own rows, the zero-byte film, the
season no candidate file reaches, the season the reference invents for an unnumbered episode, and
the two disc directories. **004 edits those twenty-three rows rather than deleting them.**

**And the count this section left for another document is taken here.** §11's last paragraph named
[conformance.md](../../docs/compatibility/conformance.md)'s two disagreeing counts and declined to
fix them because *"this plan is not the change that touches it"*. T17 **is** that change — this
feature's own work is what makes the number checkable — so the stale twenty-six is struck in place
with the date and the reason, the half about a declared difference that has gone away is added
because the sentence predates 010's amended AC-2, and the derived thirty-two is recorded beside
both. **No row of [allowlist.yaml](../../docs/compatibility/allowlist.yaml) is added, removed or
re-scoped**: the amended paragraph describes the fixture-reading comparison and not an allowlist
entry, so the three-way pairing with that file and 010 §3.3 sees the same rows before and after.
That is 002 T22's test, checked before it was relied on.

### 8.3 What only becomes observable at 005, and what 005 must not accept as proven

**This is the deferral, and it is stated as plainly as 001 stated its two L3 rows and 002 stated its
one.** The following are decided by this feature, produce no output this feature can assert against
a client's view, and become wrong answers on somebody else's route:

| Claim | Where it surfaces | What is proven here instead |
|---|---|---|
| The derived identifier's **bytes** | `Id` on every item, `ParentId`, and every user-data key from 007 | The derivation is a function with a table-driven test, and the **stored** string is asserted in `internal/store/sqlite`. Nothing checks that the string a client receives is the string that was stored |
| The sort key's **bytes**, and therefore every list's order | The order of every `/Items` response | The key is asserted as a string, including the double space and the trailing space, and the **store's own** read is asserted to order on that column and to compare it as bytes — two names `NOCASE` would order the other way round (T12). Nothing checks that any `ORDER BY` a client can reach is that one |
| Parent-child structure | `ParentId`, and `/Items?parentId=` | The `parent_id` column. Nothing checks that a client asking for a season's children gets them |
| `IndexNumber`, `ParentIndexNumber`, `IndexNumberEnd`, `ProductionYear` | The item body | The columns, round-tripped with **absent told apart from zero** (T12), which is season 0 against a season with no number. Nothing checks their **type** on the wire, which is precisely the class behaviours §1.1 to §1.7 exist for |
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

**Amended 2026-09-05 at T15, which wrote rows 2, 3 and 10 and measured what separates them.** The
three are one property measured three ways, and this table named a separating mutation for two of
them; running all five found that neither name was quite what it described
`[measurement: 003 T15, 5 mutations, 2026-09-05]`.

- **Row 3's *"a derivation that accidentally reads a stored row"* is not, on its own, a build any
  test in this repository fails.** Against a correct derivation the stored string and the derived
  string are the same string, so a scan that adopted the stored one is a no-op. What AC-3 catches is
  that adoption **over an identifier that is allocated rather than derived** — §3.6's *"never
  allocated"*, which is the shape a naïve implementation reaches for first: allocate for a new item,
  reuse for an existing one. That build is stable across a rescan and across a remount, right in
  every table in `internal/library`, and different on every installation. Removing the store's
  memory is the only thing that asks it.
- **Row 10's mutation has a broad form and a narrow one, and the corpus decides which of them the
  criterion can see.** The broad form — the root's path in every file-backed key — moves every
  `Movie` identifier, and T14's `TestRenameAndRootsLeaveEveryIdentifierUnchanged` over a tree of
  films is red on it. The narrow form is §6.3's own paragraph made real: *"the library root plus the
  normalised name"* read as the root's **path**, in the `Series` and `MusicArtist` keys alone. It is
  green on T14's test, green on `internal/library`'s own moved-root assertion — which moves the roots
  of an **empty** library and therefore asks about one `CollectionFolder` row — and red only under
  AC-10. So AC-10 moves the **whole fixture**, all four libraries and all eight kinds, and asserts
  that the corpus holds every kind before it asserts anything else. **Both of the moved-root
  assertions that existed before T15 ask about one row of §3.6's five.**

**Amended 2026-09-05 at T16, which wrote rows 11 and 14 and ran the mutations both name.** Two
things, and the second is the same shape T15 recorded one row above
`[measurement: 003 T16, 7 mutations, 2026-09-05]`.

- **Row 11's *"until 007 exists the precious row is written by the test through a store method
  rather than by a feature"* is discharged, and it needed a table as well as a method.** The table
  is `0004_item_user_data.sql` and §4.1 carries the argument for its being 003's. The clause is
  worth more than its wording suggested: the row it writes is the only row in this repository that
  a later orphan sweep would touch, so the test that asserts it survives is the only assertion
  §6.5's closing risk has. That measurement is in §6.5.
- **Row 14's rename mutation has a form that adopts nothing and a form the criterion can fail.**
  *"A renamed file is a delete plus an add"* is failed by a build that recognises the file at its
  new path and carries the old identifier over — but *"adopt a previous row whose **file signal**
  matches"*, written with the comparison this plan already has, adopts nothing at all: §6.4's signal
  includes the file's **path**, and a renamed file's path is the one thing that moved. The mutation
  the criterion fails is adoption over the size and the modification time **alone**, and running it
  is what told the two apart. The general shape is T15's, again: a mutation named in prose is a
  hypothesis.

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

**Amended 2026-09-05, at T1, in the one place building the tree forced a decision this section did
not take: what the declaration may hold *beyond* the paths the reading names.**

The check above runs one way — every path the reading names exists in the tree — and says nothing
about the other direction, which is exactly where drift gets in. **The rule T1 takes: a file the
declaration adds beyond the reading must be one that *both* servers drop, and it carries the
citation of the rule that drops it there.** Nine files are added on those terms — the second part of
the multi-part film, which both fold into the one item the reading names; an `.mp3` under a `movies`
root and an `.mka` under a `tvshows` one
`[probe: tools/probe_library_extensions.py, Jellyfin 10.11.11, 2026-08-27]`; a hidden directory and
a hidden file, which are §3.2's dot rule here and `**/.*` there
`[source: Emby.Server.Implementations/Library/IgnorePatterns.cs:89 @ v10.11.11]`; an empty `.ignore`
marker and the film beneath it
`[source: Emby.Server.Implementations/Library/DotIgnoreIgnoreRule.cs:41-68 @ v10.11.11]`; and the
legacy-encoded subtitle and the EXIF-carrying image
[conformance §L2](../../docs/compatibility/conformance.md#l2--semantic) keeps in this world for
behaviours §5.11 and 006, which no collection type admits. The reason the rule is that narrow: **a
file only Atrium drops is a difference the reading has no row for, and a file neither drops is an
item it has no row for.** Either one moves a count this project asserts ~~at forty-seven~~ **from
§8.2's declaration's own length**, and neither is visible in a green run. (Struck 2026-09-05 at T17,
which wrote the declaration and derived thirty-two; the rule this sentence states is unchanged and
the number it names was 010's.)

**Which settles one case against [conformance §L2](../../docs/compatibility/conformance.md#l2--semantic)'s
own list, and the reason is worth more than the file.** That list includes *"a name that differs
only by case"*, and the recorded reading holds no such pair: over the fifty-eight items it names in
`Movies`, `Shows` and `Music`, no two differ only in capitalisation. So either the tree the reading
was taken over did not carry the pair, or the reference folded it into one item — and
[U-44](../../docs/compatibility/reference-target.md) predicts the opposite of the second, unmeasured.
**T1 therefore does not build one.** Adding it would give Atrium an item the reading has no row for,
which is a difference in the wrong direction added to make a list come out, and it would buy an
assertion about a rule no measurement covers. U-44 stays what its own register row says it is: a
claim one scan against a single-use reference settles.

**T17 closed both halves of that, 2026-09-05.** Its Verified-by line asked for *"the case-insensitive
pair of files … among them"* and the pair cannot be among them, so the clause is struck in the task
list with the reason rather than satisfied by building the pair. What replaced it is an assertion
of the absence over **both** readings at once, which is the half T1 could not reach — T1 measured the
recorded reading; the test measures Atrium's own scan beside it. [U-44](../../docs/compatibility/reference-target.md)'s
own row claimed the difference was *"one of the forty-seven differences 003 declares over its own
fixture"*, and that clause is struck there too.

**And one thing found while reading the reading, which is [T17](tasks.md#t17--the-forty-seven-declared-differences-which-this-project-holds-the-reading-of-and-not-the-reasons)'s
rather than this section's.** The reference's reading names the series item backed by
`Shows/The Series` **`tvshow`** — a name no path-derived rule produces from that directory, and one
this feature will not reproduce. The tree carries no `.nfo` sidecar to explain it, and inventing one
to make the name come out would be the drift this amendment exists to refuse.

**Amended 2026-09-05, at T5: the fixture's multi-part film cannot assert where its name came
from, and the task list asked it to.** [T5](tasks.md#t5--resolve-for-films-the-marker-vocabulary-and-the-directory-that-names-the-film)'s
Verified-by line makes the name of `The Long Film (1998)/… - part1.mkv` *"the assertion that catches
a build which stacked the parts and then took the first file's name"*. Over that tree it catches
nothing, and it takes two independent repairs to see why: the directory holds exactly one film and
names it, so the file's stem is never consulted; and with that rule removed the **year** rule
discards everything after `(1998)`, `- part1` included. The mutation that names the item after
`Files[0]` passes every assertion the fixture's shape can carry.

So the assertion is made on a tree the fixture does not hold and does not need to: two parts
directly under a library root, with no year in the name, where `The Long Film - part1` is the only
thing a build reaching for the first file can say. **The general shape is 001's, 002's and T4's
again — when a document says an assertion exists to catch a particular failure, produce that failure
and check the assertion actually moves** — and it is the third time in this feature that the answer
was *"it does not, and something else was quietly repairing it"*.

**Amended 2026-09-05, at T6: the fixture's `24` cannot assert that the filename is matched before
the directory, and the task list asked it to.**
[T6](tasks.md#t6--resolve-for-series-seasons-and-episodes)'s Verified-by line makes
`24/Season 01/24 - S01E01 - 12-00 AM.mkv` the file *"built to catch exactly that"*. Over that path
the two orders agree: the containing directory is `Season 01` and says season 1 as loudly as the
filename does. And in the other `24` shape — a flat `24/24 - S01E01 - …` with no season directory —
there is no directory *below the series* to match first at all, because a series directory is never
also a season directory. Two independent rules, again, and the swapped order passes everything.

So the order is asserted over a tree where the two sources **disagree**: a `Season 05` directory
holding a file whose name says `S01E01`. Position decides, the filename wins, and season 1
consequently has **no path**, which kills a second mutation — a season taking a directory whose
number is not its own. What the fixture path does catch is §3.4's other half, that a series' own
title is not read as a number, and that has its own test and its own mutation. Recorded here beside
T5's because it is the same shape a third time and the pattern is now the feature's most reliable
source of findings: **a fixture built to demonstrate a rule is not the same thing as a tree that can
refute it.**

**Amended 2026-09-05, at T7: the fixture's compilation cannot fail AC-9's own failure, and this one
is not a fixable tree.** AC-9 is *"a compilation with a different artist per track resolves to one
album"* and the failure it guards against is one album **per track**. Under the `NoTags` source
there is no track artist to differ, so `Various Artists/A Compilation (1999)` is three files in one
directory and *every* grouping rule a build could have — by directory, by album name, by album
artist — answers one album. There is no mutation of the resolver that makes that tree answer three.
Unlike T5's and T6's findings this cannot be repaired by choosing a better tree: **the distinction
AC-9 is about does not exist until something says the artists differ, and nothing in this feature
reads a tag.** So the criterion's own assertion is made through the `TagSource` seam with a stub —
which proves the resolver's grouping key and proves nothing about a real tagged library — and the
fixture's own test says in terms that a green run over that tree is evidence of the tree. Both
statements are on the tests rather than in a comment somewhere, because §3.5's precedence half is
**004's** and a green suite here must not be read as evidence for it.

## 9. Risks

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Every claim in §8.3 is proven a layer below where it is written | **Certain** | Four kinds of wrong answer that only 005 can see, on the busiest route in the surface | §8.3 lists them as debts with the request that discharges them, and 005's plan inherits the list. This is the feature's largest single weakness and it is structural, not an oversight |
| The forty-seven declared differences are written from the reading rather than inherited | High | A miscount fails 010's run for the wrong reason; a difference declared with the wrong reason hides a real bug | §8.2. Each declaration names a specification section, so a wrong reason is a wrong citation somebody can check |
| A version bump leaves libraries incomplete while the background rescan runs | Medium | A client lists a partial library and caches it | §6.8. Nothing is invalidated — identifiers are derived — and the alternative is refusing to serve at all. Named so it is a decision rather than a surprise |
| Two scanners over one store file | Medium | Contention, or two scans of one library | §6.9's claim, renewed per batch; ADR-0003's WAL and busy timeout absorb the rest |
| The `.ignore` rule is narrower than the reference's in two ways | Medium | On a tree with a non-empty marker, items here that the reference hides | §6.1, [U-42](../../docs/compatibility/reference-target.md). One `.ignore` file settles it |
| Spec §3.3's `-a`/`-b` stacking form does not exist at the reference | Medium | Implementing the spec's reading merges two films into one and loses one of them | §6.2 implements the source's vocabulary; [U-43](../../docs/compatibility/reference-target.md), and the spec's parenthetical is corrected |
| Case-insensitive identifiers by default differ from the reference's default | Medium | Two files differing only in case are one item here and two there — an item-count difference on a real library | §6.3, [U-44](../../docs/compatibility/reference-target.md) ~~, and it is one of the forty-seven §8.2 declares~~. **Struck 2026-09-05 at T17**: the fixture holds no case-only-differing pair, so it is one of none — U-44 is unobservable over this tree in either direction and stays owed a probe |
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

**Taken 2026-09-05 at T17, and §8.2 records it.** The correction is this feature's after all: T17 is
the change that writes the declaration the stale sentence describes, and *this feature's own work is
what makes the number checkable*. Twenty-six is struck in place with the date and the reason; the
pairing with [allowlist.yaml](../../docs/compatibility/allowlist.yaml) is undisturbed because no row
of that file is added, removed or re-scoped. The derived count is **thirty-two** and is recorded
beside the corrected forty-seven rather than replacing it — §8.2 carries the arithmetic and the
seven rows this project cannot derive.
