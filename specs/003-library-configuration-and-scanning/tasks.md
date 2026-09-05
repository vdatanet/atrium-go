---
feature: 003-library-configuration-and-scanning
title: Library configuration and scanning — tasks
status: In review
created: 2026-09-05
updated: 2026-09-05
plan_status_required: Accepted
---

# 003 — Tasks

Ordered. Each task is a reviewable change on its own, and states how you know it worked.

No task may say "implement the feature". If one does, it needs breaking down.

**On the gate.** [plan.md](plan.md) was `In review` and said in terms that it *"becomes `Accepted`
when that review returns and a task list is asked for, which is what `tasks.md`'s own
`plan_status_required` gates"*. One has been asked for, so the plan moves to `Accepted` in this
change and this list is `In review` behind it — the same transition 001 and 002 both recorded, one
artefact further down. Writing the list amended the plan in four places and the specification in
two; [What the gate changed](#what-the-gate-changed) says what forced each, and the loop in
[specs/README.md](../README.md) closing is why that does not reopen the gate behind it.

**The shape of this list follows the plan's shape, and the plan's shape is not a list of
endpoints — because there are none.** [plan §1](plan.md#1-approach) says 003 is *"no endpoints and
the whole library, and the absence of endpoints is the organising problem rather than a saving"*,
and then names the two consequences that decide everything: **Principle VIII has no boundary to
assert at**, and **the scan is the most destructive code in the project with nothing above it to
notice a mistake**. So T1 builds the tree everything else is table-driven over; T2–T7 are
`internal/library`, which is a function over paths and names and where most of spec §3 lives;
T8–T9 are the walk and the reconciliation; T10–T12 are the store, including the one change 001 said
would never be needed; T13–T16 are the scan and the criteria that need two of them; T17 is the
forty-seven; T18 is what a package that speaks HTTP can prove about a feature with no routes;
T19–T20 close the documents.

**Three levels, and every task says which one its assertions sit at**, because [plan §8.1](plan.md#81-what-replaces-the-http-boundary-and-how-much-weaker-it-is)
makes that the whole honesty of this feature's proof:

| Level | Means | What it cannot see |
|---|---|---|
| `library` / `scan` | A Go test beside the package. A function's test **at the layer of the function** — not *"one layer in"*, because these functions are the only producers | Anything about what was stored, or about what a client would receive |
| `app` | Through the subcommand, over a real temporary data directory and a real tree | Anything on the wire |
| `conformance` | Through the built binary, as an operator could | Anything about an item's *shape*, because 003 produces no wire representation at all |

**One ordering decision is taken here deliberately, and it is 002 T17's shape.** The moment
`derived/library.sql` exists, `internal/store/sqlite/migrate_test.go`'s assertion that the derived
half is at a literal `0` is false, and so are two more of 001's tests and three of the runner's own
comments. There is no intermediate state in which the schema exists and the build is green. **So the
derived schema, the generation constant, the runner change and 001's four affected assertions land
in one change (T11)**, because every task here has to be mergeable on its own and the alternative is
a pull request that cannot go green. The cost is that T12's store methods are written against a
schema whose only caller is a test until T13, and T12 says so.

**And one deferral is placed rather than mentioned.** [plan §8.3](plan.md#83-what-only-becomes-observable-at-005-and-what-005-must-not-accept-as-proven)
lists six claims this feature decides, produces no output it can assert against a client's view, and
which become wrong answers on somebody else's route. Each one has a task that establishes the half
that *can* be established here, and each of those tasks carries a **What this does not prove** line
naming what is left. None of them is closed by this feature and no definition-of-done line may say
otherwise:

| §8.3's claim | Established here by | Left to |
|---|---|---|
| The derived identifier's **bytes** | T3 (the derivation), T15 (the stored string, across three scans) | 005 — `Id` and `ParentId` on a body |
| The sort key's **bytes**, and every list's order | T4 (the key as a string), T12 (`BINARY`, as a stored column) | 005 — that `ORDER BY` uses that column |
| Parent-child structure | T6, T7 (placement), T13 (the `parent_id` the store ends up holding) | 005 — `/Items?parentId=` |
| `IndexNumber`, `ParentIndexNumber`, `IndexNumberEnd`, `ProductionYear` | T5, T6, T7 (the values), T12 (the columns) | 005 — their **type** on the wire |
| A multi-part film being one item with two sources | T5 (one candidate, two parts), T12 (one `items` row, two `item_files`) | 008 — `MediaSources` |
| A container with nothing under it not being offered | **Nothing.** T9 establishes only that the container is *kept* | 005, entirely — [behaviours §5.2](../../docs/compatibility/behaviours.md#52-a-container-that-has-lost-every-file-is-not-removed) |

## Legend

`[ ]` not started · `[~]` in progress · `[x]` done · `[!]` blocked (say by what)

---

## T1 — The fixture tree: one declaration, two ways to reach it, and the reading it is checked against

- [x] **Changes:** `internal/libraryfixture` — the declaration of
  [conformance §L2](../../docs/compatibility/conformance.md#l2--semantic)'s scanning world as a Go
  value, and a builder that writes it into a directory. `tools/build_library_fixture` — the same
  declaration as a program, so `conformance/` can have the tree without importing anything of ours
  ([plan §3](plan.md#3-modules), [§8.5](plan.md#85-the-fixture-and-why-it-is-generated-rather-than-checked-in)).
  The expected item set lives in a **third** file, as a literal.
- **It is first because everything after it is table-driven over it**, and because the tree is the
  one input this feature shares with the reference: [reference-fixture-reading.json](../../docs/compatibility/reference-fixture-reading.json)
  was taken by mounting *this* tree into a container, so a fixture that drifts from it makes T17's
  comparison meaningless while leaving it green — which plan §8.5 calls the worst available outcome.
- **Three libraries, not six.** `Movies`, `Shows` and `Music` are this feature's; `Empty` is a
  library with nothing in it, which [behaviours §5.7](../../docs/compatibility/behaviours.md#57-an-empty-library-reads-unplayed-where-the-references-source-reads-it-as-played)
  needs and which this feature must be able to *configure* even though it has nothing to scan; the
  `Films` and `Tunes` trees are the media world 008 encodes with ffmpeg and are behind a build tag
  ([architecture §8](../../docs/architecture.md#8-testing-and-conformance)). The check against the
  reading is **scoped to the four this feature builds and says so**, because a check that silently
  skipped a third of the libraries would read exactly like a check over all six.
- **Depends on:** —
- **Verified by:** every path `reference-fixture-reading.json` names under those libraries exists in
  a freshly built tree, with the expected paths **read out of the JSON** rather than transcribed —
  transcribing them is how the two stop being the same tree, and the transcription would still pass;
  the zero-byte film measures **zero bytes** and the `.ignore` marker is **empty**, asserted as
  lengths, because each of those two files is a *rule* the tree exists to exercise and a builder that
  wrote one byte into either would disable a rule silently and cost nothing visible; two builds into
  two directories produce the same relative paths and the same sizes and **different modification
  times**, which is the property a committed tree cannot have and is plan §8.5's first reason for
  generating it; `Empty` is built as a directory that exists and holds nothing, not as a path that is
  missing, because the second is AC-12's failure and not an empty library; and the expected item set
  is asserted to be reachable **without calling the builder** — the test that carries it imports the
  literal, so a change to the tree is a change to two files and a reviewer sees both. Plan §8.5's
  *"must not assert a count it computed from the same declaration"* is the whole point: a count
  derived from the builder is a test of arithmetic.
- **Spec reference:** §6; plan §3, §8.5.

## T2 — `internal/library`: the collection types, the extension lists, and the promotion that must not happen

- [x] **Changes:** `internal/library` — `CollectionType` over spec §3.1's three names, `Admits` over
  §3.2's measured lists, and the path-shaped exclusions that can be decided from one path: a
  component beginning with `.`, the trailer/sample/extra suffixes, and the extras folder names
  ([plan §6.1](plan.md#61-the-walk-and-what-it-refuses-to-look-at)). The lists are constants per
  collection type and there is nothing to configure them with
  ([plan §4.3](plan.md#43-which-of-these-is-derived-and-the-two-that-are-not-obvious)).
- **Depends on:** —
- **Verified by:** each of the three types admits exactly its measured list and nothing else
  `[probe: tools/probe_library_extensions.py, Jellyfin 10.11.11, 2026-08-27]`; an `.mp3` and an
  `.mka` under a `movies` or `tvshows` library are admitted by **no type at all**, asserted as *no
  item of any kind* rather than as *"movies refuses `.mp3`"* — [behaviours §2.15](../../docs/compatibility/behaviours.md#215-an-audio-file-under-a-video-root-is-not-an-item)'s
  finding is about **promotion**, and a test written the second way passes on a build that quietly
  files theme music beside a film as `Audio`, which is the exact bug the measurement exists to
  prevent; **`Specials` is not an extras name**, asserted here as well as at T6 because spec §3.4
  says a scanner that grouped it with `Extras` and `Featurettes` *"would drop every special episode
  in every series while producing a scan that looks entirely correct"* — the failure is invisible in
  the summary, so it needs an assertion at the predicate; and an extras **folder** excludes its
  contents while an extras **suffix** excludes one file, asserted apart, because a build that
  conflated them is right about the fixture and wrong about a `Featurettes` directory holding
  something that is not suffixed.
- **Spec reference:** §3.1, §3.2, §3.4; plan §6.1, §4.3.

## T3 — Path normalisation and `DeriveID`, which the whole feature rests on

- [x] **Changes:** `internal/library` — `Normalise(path, caseSensitive)` and
  `DeriveID(libraryID, kind, key)` of [plan §6.3](plan.md#63-identity-and-the-normalisation-the-whole-feature-rests-on):
  the first 16 bytes of SHA-256 over the library identifier, a NUL, the item kind, a NUL, and the
  normalised key, rendered as 32 lowercase hexadecimal characters.
- **Depends on:** —
- **Verified by:** a decomposed and a precomposed spelling of one name derive **one** identifier, and
  a name differing only in capitalisation derives one identifier when the library is case-insensitive
  and **two** when it is not — which is the whole of what `case_sensitive` decides and is why §3.6
  freezes it; the two normalisation steps are asserted **in the stated order** over an input where
  the order is observable, and the test names the character it relies on rather than asserting an
  order no input can distinguish — case folding is not closed over normalisation forms, and a test
  that could not find such an input would be asserting nothing while looking thorough; an
  **absolute** key and one that **climbs above its root** are errors and not normalisations, and the
  error is asserted to be distinguishable from *"this file was skipped"*, because T13 has to fail a
  whole library on it; `DeriveID("ab", kind, "c")` and `DeriveID("a", kind, "bc")` differ, which is
  the collision a concatenation without separators produces and which 002 T9 already had to assert
  once for a session identifier; a `Series` keyed on `Shows/The Series` and a `Season` keyed on the
  same string differ, because the kind is in the key precisely so a directory and the item it backs
  cannot collide; two libraries over the *same* tree derive different identifiers and one library
  whose **root path** moved derives the same ones, which is [plan §6.3](plan.md#63-identity-and-the-normalisation-the-whole-feature-rests-on)'s
  *"the library identifier is in the key, and the root path is not"* asserted as the pair of
  inequalities it is; and the output is 32 lowercase hexadecimal characters, compared against a
  golden so that the derivation is pinned rather than merely shaped
  ([behaviours §1.4](../../docs/compatibility/behaviours.md#14-item-identifiers-are-32-lowercase-hex-characters)).
- **What this does not prove:** that the string a client receives is the string that was stored.
  Nothing in this feature can. §8.3 row 1, discharged at 005 by one `/Items` listing.
- **Spec reference:** §3.6, AC-10; plan §6.3.

## T4 — The two sort-name derivations, and the single entry point that makes the wrong one unreachable

- [x] **Changes:** `internal/library` — `SortKeyBase(string)` over spec §3.7.1's six ordered steps
  with the measured defaults, and `SortKeyFor(*ports.ScannedItem)` which switches on the type and
  applies §3.7.2 to `Audio`, `Episode` and `Season` and §3.7.3 to anything carrying an explicit sort
  title ([plan §6.6](plan.md#66-sort-keys)). The implementation walks runes and appends;
  `strings.Fields`, `strings.TrimSpace` and `strings.Join` are each a way to lose the contract by
  accident.
- **Depends on:** —
- **Verified by:** spec §3.7.1's fourteen-row table verbatim, with the expected values written so
  that **a trailing space is visible in review** — `s w a t ` and `rock  roll` are the contract, a
  trailing space inside a Go string literal is invisible in a diff and `gofmt` will not save it, so
  the table is written with an explicit delimiter and the comparison is on the whole string; the
  three overriding types with their real asymmetry — `Audio` pads disc and track to **4**, `Episode`
  pads its season to **3** and its episode to **4**, `Season` is four digits and nothing else
  `[source: MediaBrowser.Controller/Entities/Audio/Audio.cs:94-98, MediaBrowser.Controller/Entities/TV/Episode.cs:238-242, MediaBrowser.Controller/Entities/TV/Season.cs:149-152 @ v10.11.11]`;
  a missing number contributing **no segment at all**, asserted as the absence of a `0000 - ` run
  rather than as some other string, because a run of zeros is what the obvious implementation
  produces and it sorts everything unnumbered ahead of everything numbered; `2 Fast 2 Furious`
  sorting before `10 Things` **by bytes**, with the pad width moved to 9 and to 11 in a mutation that
  must turn it red — the width is part of the contract and a comment saying so is not a check.
  *(Amended when the test was written: the ordering **does not reverse** at 9 or at 11, because
  padding to any width of at least two already makes the shorter run compare low, so the ordering
  assertion this line asks for is satisfied by three different contracts. The bytes are what pin the
  width. The test asserts the ordering at all three widths, the bytes at all three, and the reversal
  at width one — no padding — which is the only mutation of the width the ordering alone can see;
  plan §6.6 records it.)* And
  `SortKeyFor` over an `Audio` producing a key that ends in the **raw** `The Song`, asserted as the
  raw name and not as the absence of an article, because [behaviours §2.6](../../docs/compatibility/behaviours.md#26-sortname-has-two-derivations-and-three-types-use-the-second)'s
  named temptation is a codebase with *one* sort-name function and its symptom is every album in the
  library reordered. AC-15: an explicit sort title replaces the derivation for **every** type
  including the three that override, and is lowercased and digit-padded but **not**
  article-stripped — asserted with a title beginning `The `, since that is the only clause that
  distinguishes §3.7.3 from §3.7.1. OQ-7's tail — fold, a short table of the obvious Latin readings,
  then drop — is asserted as **stable** (the same input twice gives the same answer) and not as
  correct, because nothing has measured it and the register already holds it open.
- **What this does not prove:** that any list a client receives is ordered by this key. §8.3 row 2;
  T12 carries the collation half and 005 carries the `ORDER BY`.
- **Spec reference:** §3.7, AC-13, AC-15; plan §6.6.

## T5 — `Resolve` for films: the marker vocabulary, and the directory that names the film

- [x] **Changes:** `internal/library` — the classify and group passes of
  [plan §6.2](plan.md#62-resolution-and-the-three-shapes-that-need-siblings) for `movies`: title and
  year extraction, release-tag removal, multi-part stacking, and the rule that a directory holding
  exactly one video candidate names it. `Resolve` takes every root's reading at once, sorts it, and
  sorts every map on the way out (plan §5).
- **Depends on:** T1, T2, T3, T4
- **Verified by:** `The Long Film (1998)/… - part1.mkv` and `… - part2.mkv` resolve to **one** item
  with two parts in ordinal order, and the item's **name** is asserted to be `The Long Film (1998)`
  and not `The Long Film (1998) - part1` — the name is the assertion that catches a build which
  stacked the parts and then took the first file's name, and the reference's own reading of this tree
  names it the same `[probe: tools/probe_reference_scan.py, Jellyfin 10.11.11, 2026-09-02]`;
  `The Film - a.mkv` and `The Film - b.mkv` are **two** items while `The Film - cda.mkv` and
  `The Film - cdb.mkv` are **one**, which is [U-43](../../docs/compatibility/reference-target.md)
  asserted as a divergence from the parenthetical spec §3.3 withdrew — it is the one reading in this
  feature that **loses** an item, so the test exists to go red the day somebody sends both shapes to
  a running reference; a directory holding two different titles names **neither**, asserted by both
  candidates keeping their filename-derived names, because §3.3 says that is the only part of the
  rule a single path cannot decide and the group pass is where it becomes decidable; the year is
  taken from the bracketed and the trailing forms and refused outside 1900–2099; the release-tag
  removal is asserted **behaviourally over a corpus of names**, never by transcribing an expression
  (Principle IV), and the corpus is written down rather than generated; and two readings of one tree
  whose directory entries were created in opposite orders produce the **identical** plan, including
  part order — Principle VII at the layer where insertion order can still get in, and the one thing a
  per-path resolver could not have.
  *(Amended when the task was written, in two places, and the first is this feature's third
  criterion proven a level too low.* **(1) The name of the fixture's own multi-part film catches
  nothing.** *Over `The Long Film (1998)/… - part1.mkv` two rules independently repair a name taken
  from the first file: the directory holds exactly one film and names it, and — with that rule
  removed — the year extraction discards everything after `(1998)`, `- part1` included. The
  mutation that names the item after its first part passes every assertion this tree can carry. So
  the assertion is made where nothing can repair it, on two parts directly under a root with no
  year in the name, and the fixture's tree keeps the count, the part order and the path.
  [Plan §8.5](plan.md#85-the-fixture-and-why-it-is-generated-rather-than-checked-in) records it.*
  **(2) The reference does not name it `The Long Film (1998)` for the same reason Atrium names it
  `The Long Film`.** *Both servers name that item after its **directory** and neither after a file,
  which is the agreement this line is really about; they differ by the year, because §3.3 takes it
  out of a name and the reference keeps the directory whole. That is a **declared** difference
  already counted in [plan §8.2](plan.md#82-the-declared-inequality-and-the-forty-seven) — two rows,
  this film and `The Matrix (1999)` — so the test asserts the reference's own name run through
  Atrium's year rule equals Atrium's, which fails if either half moves.)*
- **What this does not prove:** that a client sees one item with two media sources. §8.3 row 5;
  T12 stores the two rows and 008 answers `MediaSources`.
- **Spec reference:** §3.3, AC-4; plan §6.2; U-43.

## T6 — `Resolve` for series, seasons and episodes

- [x] **Changes:** `internal/library` — the three levels of spec §3.4, with the number patterns
  matched against the **filename first and the parent directory second**, seasons inferred where no
  directory exists, `Specials` as season zero, multi-episode files as one candidate spanning two
  numbers, and extras ignored rather than attached.
- **Depends on:** T2, T3, T4, T5
- **Verified by:** `S01E02-E03` resolves to **one** item with `IndexNumber` 2 and `IndexNumberEnd`
  3 — the *count* asserted first, because two items each carrying one number satisfies every
  per-field assertion anybody would write; `Specials` resolves to season 0 **and** `Extras` and
  `Featurettes` beside it resolve to no season at all, which is the companion spec §3.4 asks for
  because the failure produces a scan that looks entirely correct; the series `24` keeps its title,
  acquires no episode number, and its episode's numbers come from the **filename** — the mutation
  that must fail it is matching the directory first, and the fixture's
  `24/Season 01/24 - S01E01 - 12-00 AM.mkv` is built to catch exactly that; a season **inferred**
  from `The Series - S02E01 - No Season Directory.mkv` where no directory exists, and its identifier
  derived from the series' identity plus the number rather than from a path it does not have (§3.6);
  `The Daily Show - 2024-01-31` resolving by date; `S02E99` resolving without complaint, because an
  episode number beyond any real count is not an error and real libraries hold them; and
  `blob.mkv` in a season directory producing an item marked **unplaceable** rather than being
  skipped — spec §3.8 counts the two apart precisely because *"an operator told that both were
  skipped would go looking for something that is not missing"*, and this is the fixture's one file
  that exercises the distinction.
- **Amended at T6, and it is the third time in this feature.** *(1) The fixture's
  `24/Season 01/24 - S01E01 - 12-00 AM.mkv` does **not** catch a resolver matching the directory
  first.* Two rules repair it: the containing directory is `Season 01` and says season 1 as loudly
  as the filename does, and the flat `24/24 - S01E01 - …` shape has no directory *below the series*
  to match first, because a series directory is never also a season directory. The order is
  asserted over a `Season 05` directory holding a file whose name says `S01E01`, where the two
  sources disagree — and that tree kills a second mutation too, a season taking a path from a
  directory whose number is not its own. What the fixture path does catch is §3.4's other half,
  that a series' own title is not read as a number.
  *(2) One assertion this line does not ask for was needed by the same file:* a spaced ` - `
  followed by digits is not an episode range. Read loosely, `24 - S01E01 - 12-00 AM` becomes
  **episodes 1 to 12** — still one item, still season 1, still episode 1, so AC-5's own test, AC-7's
  own test and the fixture comparison all stay green.
  [Plan §6.2 and §8.5](plan.md#62-resolution-and-the-three-shapes-that-need-siblings) record both.
- **What this does not prove:** that a client asking for a season's children gets them. §8.3 row 3.
- **Spec reference:** §3.4, AC-5, AC-6, AC-7; plan §6.2.

## T7 — `Resolve` for music, and the seam 004 fills

- [x] **Changes:** `internal/library` — album and artist grouping for `music`, the `TagSource`
  interface consulted **once per file before grouping** (plan §6.2), and the null implementation v1
  ships. The path fallback for the title, the track number and the disc number, with the tie-break
  spec §3.5 states.
- **Depends on:** T2, T3, T4
- **Verified by:** `The Artist/Double Album/CD1` and `CD2` resolve to **one** album whose tracks
  carry `ParentIndexNumber` 1 and 2 — one album asserted before any disc number, because two albums
  each with the right disc number is the failure and it passes a per-track assertion;
  `Various Artists/A Compilation (1999)` resolves to **one** album, with the attribution coming from
  the directory under the null tag source and 004's tests carrying the tag-driven half — stated here
  so that a green suite is not read as evidence for spec §3.5's precedence rule, which this feature
  cannot exercise at all; the fallback's tie-break asserted as the **narrowing** it is —
  `24K Magic.flac` is a song called `24K Magic` with no track number, `01 - Opening.flac` is track 1
  called `Opening`, and a file named after a hash keeps its whole name — because a name Atrium
  declines to take a number out of is a name it agrees with the reference about; and the fallback
  asserted as a **divergence** rather than as agreement, since the reference takes none of the three
  from a filename ([behaviours §2.16](../../docs/compatibility/behaviours.md#216-a-music-tracks-number-comes-from-tags-never-from-its-filename),
  spec's OQ-8) — so the day OQ-8 is answered, a failing test is the notification and not a
  rediscovery.
- **Amended at T7, and one of the two is a change to a specification rather than to a plan.**
  *(1) The fixture's `Various Artists/A Compilation (1999)` cannot fail the failure AC-9 is named
  for, and no tree in this feature can.* Under the null tag source there is no track artist to
  differ, so three files in one directory answer one album under **every** grouping rule a build
  could have — by directory, by album name, by album artist — and there is no mutation of the
  resolver that makes that tree answer three. This is not T5's and T6's finding a fourth time: those
  two were repaired by a better tree, and this one cannot be, because the distinction AC-9 is about
  does not exist until something says the artists differ. The assertion is made through the
  `TagSource` seam with a stub, which proves the resolver's grouping key and **nothing** about a real
  tagged library, and the fixture's own test says so in terms.
  *(2) Spec §3.6's identity table contradicted §3.5 about an album, and writing the code had to pick
  one.* The table put `MusicAlbum` under *"the library root plus the normalised name"*, which makes
  two artists' `Greatest Hits` **one item** — one row, one parent, half an album's tracks under an
  artist that did not record them. §3.5 already says *"an album's identity comes from its album
  artist"*, so the row is amended to read as the `Season` row above it does. The fixture's five album
  names are distinct, so the merge would have shipped green.
  [Plan §5, §6.2 and §8.5](plan.md#62-resolution-and-the-three-shapes-that-need-siblings) record the
  six decisions writing the resolver took, and the refusal deleted along with the last unwritten
  resolver.
- **What this does not prove:** that a client asking for an album's tracks gets them, or that a
  track's numbers reach one as integers. §8.3 rows 3 and 4; T12 stores the columns and 005 answers
  them.
- **Spec reference:** §3.5, AC-8, AC-9, OQ-8; plan §6.2.

## T8 — The walk: what it refuses to look at, and the ancestor search that stops at the root

- [x] **Changes:** `internal/scan` — one walk per root over `fs.FS` from `os.DirFS(root)`, applying
  T2's predicates per file and per directory, the zero-byte rule, and the `.ignore` marker: **empty
  or whitespace-only excludes the subtree; the search runs from a file's directory up to the library
  root and no further; a non-empty marker excludes nothing** (plan §6.1,
  [U-42](../../docs/compatibility/reference-target.md)). The reading is sorted on the path before
  anything looks at it.
- **Depends on:** T1, T2
- **Verified by:** an empty marker excludes the directory holding it and everything beneath; **the
  same marker one directory up** excludes the subtree below it, which is the ancestor search and
  which a per-directory implementation passes the first assertion without having; a marker planted
  **above the library root** excludes nothing, built by rooting the library inside a temporary
  directory that carries one — a deliberate divergence from the reference, which walks to the
  filesystem root, and asserted as a divergence so U-42's measurement lands on a failing test; a
  **non-empty** marker excludes nothing, which is the accepted shortfall and is asserted rather than
  left as a comment; a hidden **directory** is not descended into and a hidden **file** is skipped,
  asserted apart, because a walk that skipped the file and descended anyway gives the same answer
  until something inside is not hidden; a zero-byte file yields no candidate — one of the
  forty-seven, since the reference makes an item of it (T17); and two walks over trees whose entries
  were created in opposite orders yield the **identical** `Reading`, which is the determinism
  Principle VII wants asserted at the layer where a filesystem's own ordering could still reach the
  answer.
- **On the row that moved.** Spec §3.2 lists *"files being written, detected by size change between
  two passes"* among the ignore rules, and it is a property of a **pair of scans** rather than of a
  file. It is decided at T9 with both readings in hand, and plan §6.1 says why what v1 does is
  narrower and in the direction that costs an operator nothing.
- **Amended at T8, and the first of the two is this task's own finding.**
  *(1) The determinism clause cannot be varied the way this line asks for it to be.* *"Two walks over
  trees whose entries were created in opposite orders"* varies nothing at all: `os.DirFS` implements
  `fs.ReadDirFS`, its `ReadDir` sorts, and `fs.WalkDir` therefore reads one tree in the same order
  whichever way round it was built. The assertion would be satisfied by the standard library and
  would survive the removal of every line this feature owns — which is **T6's finding a second time,
  in a different package**, and T6's handoff warned this task about exactly it. The order is varied
  where it can reach the walk instead: an `fs.FS` whose `ReadDir` answers backwards, which
  `fs.ReadDir` hands straight through without sorting. Both are asserted; the second is the one that
  kills the mutation.
  *(2) A directory entry that is not a regular file is a rule the walk has and neither the
  specification nor plan §6.1 stated.* A symbolic link is the case: the reference follows one, so
  refusing a linked film would show **fewer** items than the reference, which is the unsafe direction
  for a scanner. The walk stats through the link; a link to a directory, to a device and to
  **nothing** are refused, and the last is refused rather than raised because a dangling link — or a
  file moved between the directory read and the stat — must not cost an operator every item in the
  library. Plan §6.1 gains the row and the argument, and `library.Skip` gains three reasons
  (`SkipIgnoreMarker`, `SkipZeroBytes`, `SkipNotARegularFile`) as T2's handoff said it should.
- **Spec reference:** §3.2; plan §6.1; U-42.

## T9 — `Reconcile`: the pure function where every removal in this project is decided

- [x] **Changes:** `internal/scan` — `Reconcile(previous, desired, full)` over two item sets, taking
  no store, no filesystem and no clock, returning the batch to write, the identifiers to remove and
  the counts of spec §3.8 ([plan §6.4](plan.md#64-change-detection)).
- **Depends on:** T3, T8
- **Verified by:** the four rows of §6.4's table; **size and modification time varied
  independently**, because a build reading only one of the two passes any test that varies both and
  the two failures it hides are different (a re-encode that keeps the length, and a restore that
  keeps the time); a stored modification time that is not a whole multiple of a tick reporting **no
  change** on a second reading of the same file, which is the mistake that reports every file
  changed on the first rescan of every installation on a filesystem whose resolution is not a tick's;
  `full` changing **only** whether an unchanged signal is believed, asserted by the two runs
  agreeing on every other row, since a full re-examination that also changed a removal decision would
  make spec §3.8's *"the default is the fast one, the full one is always available"* untrue in the
  dangerous direction; a recomputed identifier that disagrees with the stored one producing an
  **error** and never a rewritten row — the silent discard Principle VII exists to prevent, asserted
  as an error rather than as the absence of a rewrite, because the absence is also what a build that
  ignored the disagreement produces; and the removal pass marking **file-backed** items removed while
  leaving the containers above them, which is [behaviours §5.2](../../docs/compatibility/behaviours.md#52-a-container-that-has-lost-every-file-is-not-removed)
  and is asserted as a series row that survives its last episode's deletion.
- **Amended at T9, in three places, and the third is this task's own finding.**
  *(1) The signature §5 declared could not carry the answer this task's own line requires.*
  `(Changes, []ports.ScannedItem, []string)` has no return value for an error, and a disagreeing
  identifier is one; `Changes.Removed` and the third return value were the same list under two
  names; and the two lists that keep this feature's most dangerous assertions from being assertions
  about an **absence** were not there at all. It is `(Reconciliation, error)` now, where `Unchanged`
  says a row was believed rather than merely not written and `Retained` names the container the
  removal pass looked at and declined to remove.
  *(2) Two decisions plan §6.4's table does not state, and writing the function had to take both.*
  A **record** that moved is an update even where no file signal could ever move — which is the only
  way a container is ever rewritten at all, and without it a renamed library would hold its old name
  for the life of the installation — and a **full** re-examination applies to an item that has a
  file, so that the thorough option is never also a different set of deletions. §6.4 has both, and
  it says how a desired item and a previous row are paired: by identifier, with the path carrying
  the identifier comparison, because pairing by path alone turns a case-changed filename into a
  delete and an add and pairing by identifier alone cannot see the disagreement at all.
  *(3) Twenty-one mutations were run; twenty fail a named test and the survivor is the tick
  clause's own honest limit.* Comparing two `units.Time` values with `==` instead of
  `units.Time.Equal` survives the whole suite, and it is not a gap in the tests: `units.At` strips the
  monotonic reading and sets the location to UTC, so **two Times naming one instant have one
  representation** and the two operators cannot be told apart by any value this project can build.
  The comparison stays `Equal` — the equivalence is a property of the constructor rather than of the
  comparison — and the measurement is written into the function's own comment so the next person
  does not spend the afternoon reproducing it. The two mutations that *did* survive the first pass
  were both the multi-part film: nothing varied a **part's path** or the **number of parts**, and
  both are changes to an item that neither half of the signal can see.
- **And T8's row lands here, in the shows-more direction, with one correction on the way.** Spec
  §3.2's *"files being written, detected by size change between two passes"* is a property of a pair
  of scans, and this is the only place in the feature that holds both. **It lands as an update and
  never as a refusal**: the item carries the new size and nothing withdraws it while the copy runs,
  which is [plan §6.1](plan.md#61-the-walk-and-what-it-refuses-to-look-at)'s *"narrower in the
  direction that costs an operator nothing"* stated as what the code does. The correction: §6.1 said
  a file whose size **and** modification time both moved is re-read, where spec §3.8's table and plan
  §6.4's both say **or** — and under a conjunction neither of this task's two named failures is
  visible at all, so the clause requiring the two be varied independently is unsatisfiable. §6.1 is
  amended; nothing was built on the wrong reading.
- **What this does not prove:** that a container with nothing under it is not offered to a client.
  Nothing here can — §8.3's sixth row is 005's entirely, and this task establishes only the half
  that makes it 005's problem: the row is still there.
- **Spec reference:** §3.2, §3.8, AC-14; plan §6.1, §6.4, §6.5.

## T10 — The two ports, and the precious migration this feature owns

- [x] **Changes:** `internal/ports` — `LibraryStore`, `ItemStore`, and the four record types, which
  live here rather than in the domain for the reason 002 T4 decided once already: a port method
  returning a domain type inverts [architecture §2](../../docs/architecture.md#2-layers-and-the-direction-of-dependency)'s
  arrow. `internal/store/sqlite` — `0003_libraries.sql` in the **precious** lineage, creating
  `libraries` and `library_roots` ([plan §4.1](plan.md#41-the-precious-half--migration-0003_librariessql)).
- **Depends on:** —
- **Verified by:** the migration applies on an empty data directory and the **precious** version
  advances by one while the derived one does not move — 002 T1's assertion, and load-bearing twice
  over here because this is the one feature that writes into both halves; `name_folded`'s uniqueness
  refuses two libraries whose names differ only in case, so the subcommand's assumption is the
  database's rule rather than a convention; `collection_type`'s `CHECK` refuses a fourth value;
  `library_roots` reads back **in ordinal order** after a `ReplaceRoots` that reorders them, because
  the order decides nothing and a list that moves is a list nothing can be compared against; and
  **there is no method that writes `collection_type` or `case_sensitive` after creation**, asserted
  over the interface rather than over the SQL — spec §3.6 refuses the change, and the way it is
  refused is that there is nothing to call. A test that asserted an error would be asserting a
  refusal this design does not implement.
- **Amended at T10, in three places, and the third is 002's own amendment repeated one number
  along.**
  *(1) Two things `plan §4.1`'s table did not state and a schema cannot leave open.*
  `library_roots.library_id` gets **`ON DELETE CASCADE`** — foreign keys are on (ADR-0003's writer
  DSN), so without it `RemoveLibrary` is *refused* outright and the verb never works; making it the
  database's rule rather than the method's discipline is `name_folded`'s own argument one row above
  it. And `case_sensitive` deliberately carries **no `DEFAULT`**: a default is a second place the
  value is decided, where spec §3.6 makes it a property an operator states when the library is
  declared.
  *(2) `ScanBatch` was named in a signature and never declared*, so the task that owes *the four
  record types* had to say what is in one. `{LibraryID, Items, ClaimedBy, At}`, and `ClaimedBy` is
  the field that is a decision: §6.9 renews the claim inside the batch's transaction, and a renewal
  that did not name the claimant lets a scanner whose claim has gone stale and been taken renew one
  it no longer holds — two scanners writing one library, each believing it is alone. `ItemStore` is
  declared and nothing implements it yet; T12 is where `var _ ports.ItemStore` appears.
  *(3) `0003_libraries.sql` turned three of 002 T1's own literals red, and they are the literals
  002 T1 wrote while correcting 001's.* Its amendment note reads *"the assertion read want [1] and
  was a literal that every later migration invalidates"*, and the body it replaced them with was
  `len(precious) != 2`, `[]int{2}` and `want 2`. The rule never mentioned a total: it is *this
  file, in this half, advancing this half by one*. It is now `filedUnderThePreciousLineage(t,
  filename)` in `migrate_test.go`, given a migration's file name, and both features' tests call it.
  **The general shape, and it is the third feature to meet it: a correction that restates a literal
  is not a correction.**
- **Twenty-four mutations were run; twenty-three fail a named test and the survivor is a limit the
  code predicted before it was measured.** Removing `ORDER BY ordinal` from the *single-library*
  roots read leaves the whole suite green: the filter is on `library_id`, which SQLite answers
  through the primary-key index on `(library_id, ordinal)`, so the rows arrive in ordinal order
  whether or not anything asked. That is why `Libraries` reads the **whole** roots table in one
  query — a full-table read is a row-order scan, so the ordinal clause is observable there — and
  `TestLibraryRootsAreOrderedByTheirOrdinalAndNotByWhereTheRowsSit` rewrites the rows into an order
  that disagrees with their ordinals and asserts the scramble took *before* it asserts the answer.
- **What this does not prove:** that the precious half survives the derived half being dropped.
  Nothing here can drop anything — `RebuildDerived` is declared and unimplemented — so this task
  establishes only that a library outlives a restart. ADR-0003's central claim is T11's.
- **Spec reference:** §3.6, §4; plan §4.1, §5.

## T11 — The derived half stops being a lineage, and 001's runner changes after all

- [x] **Changes:** `internal/store/sqlite` — `derived/library.sql` holding the whole current derived
  schema (`items`, `item_files`, `scan_state` of [plan §4.2](plan.md#42-the-derived-half--schema-generation-1-derivedlibrarysql)),
  the constant `derivedGeneration = 1` paired with the schema file's SHA-256, `RebuildDerived`, and
  `Open` comparing the recorded derived version against the constant after the precious lineage is
  applied and **dropping and recreating on any difference in either direction**
  ([plan §6.8](plan.md#68-the-derived-halfs-generation-and-the-rescan-that-replaces-002s-refusal)).
  The runner stops applying a lineage to the derived half at all.
- **This is [ADR-0003](../../docs/decisions/0003-sqlite-as-the-store.md)'s gap closed, and the record
  is not edited.** The ADR says *"a derived-version mismatch at startup is a rescan rather than an
  error"*; 001's T3 implemented a refusal instead, with a comment saying the branch *"needs a
  scanner to rescan with, which is 003's"* and that *"the refusal is owed a replacement there, not
  here"*, and 002's T3 inherited it and recorded the same. This is the replacement. The ADR is
  accepted and immutable ([AGENTS.md §4](../../AGENTS.md)) and it is not wrong — it is a record of a
  decision as taken, including what it owed — so **say so in the commit and in plan §6.8** rather
  than leaving a reader to think the gap is open, which is how 002 T6 handled ADR-0006's own
  *"asserted nowhere"* line.
- **It is the deviation [plan §2](plan.md#2-inherited-decisions) records, and it costs four of 001's
  assertions rather than the one plan §6.8 named.** Writing this list found the other three; plan
  §6.8 is amended in the same change with all four, because a plan that names one leaves the other
  three to be discovered as a red build by somebody who will assume they broke something:
  - `TestTheHalvesCarrySeparateSchemaVersions` asserts the derived half is a literal `0`, *"neither
    001 nor 002 scans anything"*. It becomes `derivedGeneration` — which is what
    [002's handover asked for by name](../002-authentication-users-and-sessions/tasks.md#what-this-feature-owes-the-next-ones),
    and the whole point of its having been written as a literal rather than as a count.
  - `TestLoadLineageReadsAHalfWithNothingInIt` asserts `loadLineage(migrationFiles, Derived)` is
    empty. It stays true and **stops meaning anything**, so it is replaced by an assertion that no
    `migrations/derived` directory exists at all: after this change a file filed there is applied by
    nothing, and a migration nobody runs is worse than a missing one.
  - `TestTheRunnerAppliesOnlyWhatIsPending` drives the runner over `Derived` **as a lineage**, to
    prove the runner takes a half. That is no longer a call anything makes, so the test moves to a
    synthetic lineage on the other half and keeps proving the same thing.
  - `migrate`'s newer-than-known refusal, and the doc comments on `Half` and `Derived`, each say in
    terms that the derived lineage is empty and that the refusal is owed a replacement at 003. All
    three become false in this change and all three are rewritten with it.
- **Depends on:** T10
- **Verified by:** a database recorded at generation **2** opened by a build declaring 1 drops and
  recreates, and one recorded at **0** does the same — both directions asserted, because a
  one-directional branch answers a downgrade and not an upgrade and that is plan §6.8's first
  rejected shape; the **precious** half is untouched across a rebuild, asserted by an account created
  before it authenticating after it, which is ADR-0003's central claim and the one thing a wrong drop
  destroys for good; editing `derived/library.sql` without moving the constant **fails**, with a
  message naming both values — 002 T1's *"the constraint is redundant today on purpose: it is the one
  thing in the schema that notices"* applied to a constant; every object the schema declares is
  dropped, asserted by reading the derived objects out of `sqlite_master` before and after rather
  than by dropping a list somebody typed, because a table added to the schema and forgotten in the
  drop list survives a rebuild carrying its old columns and nothing else in the suite would see it;
  a rebuild leaves every library's `scan_state` row **absent**, which is the same state a library
  that has never been scanned is in and is why plan §4.3 puts it in this half; and `foreign_keys(1)`
  stays on across the drop with `libraries` holding rows — the assertion that fails the day somebody
  writes `REFERENCES libraries(id)` into the derived schema, which is
  [architecture §6](../../docs/architecture.md#6-state-and-the-store-boundary)'s rule at the one
  place it actually bites.
- **Done at T11, and it cost *five* assertions rather than the four this entry lists.** The fifth is
  `TestAFirstStartCreatesTheTwoLibraryTables`, asserting `SchemaVersion(Derived) == 0` — written by
  **T10**, one task before this one, and named only in T10's handover. A list of what a change will
  break is written against the tree as it was when the list was written, and T10 landed in between.
  The five did **not** become five spellings of `derivedGeneration`: T10's own finding is that *a
  correction that restates a literal is not a correction*, so the four call sites that meant *"a
  start leaves the derived half where this build's schema puts it"* share
  `theDerivedHalfIsAtItsGeneration`, and a sixth caller costs a line rather than a number.
- **And one clause did not fail the way it was named for, which is the finding.** *"`foreign_keys(1)`
  stays on across the drop"* was written as though a derived table referencing `libraries(id)` would
  make the drop refuse. Measured, it does not: `DROP TABLE` performs an implicit `DELETE` of the
  **child** rows and deleting a child violates nothing, so such a schema drops and recreates in
  silence `[measurement: modernc.org/sqlite v1.58.0, Go 1.27.1, 2026-09-05]`. The assertion that does
  bite reads the constraint — every foreign key the derived schema declares must target a derived
  object — with *still on* and *still enforcing* beside it so that a rebuild reaching for
  `PRAGMA foreign_keys = OFF` fails. plan §6.8 carries both the measurement and the replacement.
- **Seventeen mutations were run; fifteen fail a named test and both survivors are declared
  controls.** A whitespace-only edit to `derived/library.sql` with the digest updated survives, which
  is the generation being a number the build states rather than a fingerprint (§6.8's second rejected
  shape). And dropping in declaration order rather than in reverse survives, which is the same
  measurement as above seen from the other side: the cascade answers either order.
- **Spec reference:** §4; ADR-0003; plan §4.2, §4.3, §6.8, §2.

## T12 — The SQLite half of the two derived ports

- [x] **Changes:** `internal/store/sqlite` — the readers and writers behind `ItemStore`:
  `ItemsForLibrary`, `ApplyScanBatch` as one transaction that also renews the claim, `RemoveItems`,
  `ClaimScan` and `ReleaseScan` ([plan §5](plan.md#5-contracts), [§6.9](plan.md#69-two-scanners-batching-and-what-a-scan-does-while-the-server-serves)).
- **These methods have no caller but a test until T13**, which is the cost of the ordering decision
  above and is stated rather than discovered.
- **Depends on:** T11
- **Verified by:** `ApplyScanBatch` is **one** transaction — an injected failure part way through
  leaves neither the items nor the renewed claim, so a scan cannot record progress it did not make
  and the claim cannot outlive the batch that was supposed to prove the scanner is alive; a
  multi-part film round-trips as **one** `items` row and **two** `item_files` rows in ordinal order,
  read back rather than compared against what was written; `RemoveItems` cascades `item_files` and
  leaves another library's rows alone, which is the over-broad `DELETE` 002 T4 already had to guard
  against once and which here costs a whole library instead of a token; `ClaimScan` returns **false**
  and not an error for a live claim and **true** for one older than `staleAfter`, with the previous
  claimant readable so the log line can name it; `ItemsForLibrary` orders on a stated key, asserted
  by two reads agreeing element for element, because `Reconcile` compares sets and a store that
  returned them in storage order would make a scan's answer depend on insertion order one layer below
  where Principle VII is usually enforced; and `sort_key` compares as **bytes** under `BINARY`,
  asserted with two names that `NOCASE` orders differently from a byte comparison — ADR-0003 names
  that collation mistake and nothing else in this feature can see it.
- **Amended at T12, in three places, and the first is what the clause about the previous claimant
  forced.**
  *(1) `ClaimScan` returns the claimant it displaced or lost to.* ~~`(bool, error)`~~ became
  `(bool, string, error)`. Plan §7 asks for two messages naming a claimant — *"the second reports
  'already being scanned'"* and *"broken and taken, with a log line naming the previous claimant"* —
  and the declaration had no way to produce either: after the call the row names the winner, and a
  caller that read it first would be naming a claimant it had not necessarily displaced. Same shape
  as T9's amendment of `Reconcile` and T4's addition of `SortTitle`.
  *(2) §6.9's "one conditional statement" became one transaction*, for the same reason: an upsert's
  `RETURNING` answers the row as it now stands. The transaction takes the write lock at `BEGIN`
  (`_txlock=immediate`) on the handle capped at one connection, so the atomicity is the statement's
  and the only thing that changed is that the old value survives long enough to be returned.
  *(3) Two rows were added to plan §7*, both of them refusals this task had to decide rather than
  transcribe: a batch naming one item twice, which is **T3's finding turned into a decision** —
  NFC's singleton mappings put two files under one identifier, and the alternative to failing the
  library's scan is a last write winning silently — and a removal naming an identifier no row holds,
  which is 001's rows-affected rule where a `DELETE` that quietly matched fewer rows leaves the next
  scan computing the same removal for ever.
- **Twenty-five mutations were run; twenty-three fail a named test and both survivors are declared.**
  Removing `ORDER BY item_id, ordinal` from the **file** read survives, because the primary key on
  `(item_id, ordinal)` answers that shape whichever way the rows sit — T10's finding one table along,
  and the reason the *item* read is ordered on `sort_key` rather than on the primary key, where the
  same mutation fails. And moving the claim's renewal from the first statement of the batch to the
  last survives: both orders are inside one transaction and no failure this store can reach tells
  them apart, so the renewal being first is a property of the test rather than of the behaviour and
  says so in the code. Two mutations plan §8.3 names by name were run deliberately: `ORDER BY name`
  instead of `sort_key` (which needed `The Abyss` in the corpus, because the article this project
  strips is the only thing making the two orders differ) and `COLLATE NOCASE`.
- **What this does not prove:** that `ORDER BY` in any query a client reaches uses this column, or
  that the four numeric columns travel as integers. §8.3 rows 2 and 4, both 005's. Both rows now say
  what T12 added underneath them, so that neither reads as discharged.
- **Spec reference:** §4; plan §4.2, §5, §6.9, §7, §8.3; ADR-0003.

## T13 — The scan: the three guards, the batches, and the summary that counts two things apart

- [x] **Changes:** `internal/scan` and `internal/app` — the act, assembled: claim the library, walk
  every root, run the guards on the **reading**, `Resolve`, `Reconcile`, write in batched
  transactions, remove in the final one, release the claim and write the summary
  ([plan §6.5](plan.md#65-the-guard-against-a-mass-delete), §6.9).
- **Depends on:** T5, T6, T7, T9, T12
- **Verified by:** **AC-12 through the subcommand over a real data directory** — the item count and
  three named identifiers before and after a scan whose root cannot be read, and the test shown able
  to fail by moving the reconciliation ahead of the guard, because plan §7 says in terms that *"a
  test that asserts only the error is met by a build that errors after removing"*; both shapes of an
  unreadable root are covered — a path that does not exist, and a directory whose permissions were
  removed — and the second **skips itself** when the directory turns out to be readable anyway, since
  `os.Chmod` does not make a directory unreadable for `root` and a test that silently passed under a
  root user would be a green proving nothing (plan §3); **guard 2**, the amendment this list forced
  into the specification: a root that reads as holding no candidate file where the previous scan
  recorded at least one refuses and **names the root**, and the same scan with the operator's
  explicit permission proceeds and removes — both halves, because a test asserting only the refusal
  passes on a build whose override does nothing (AC-16); **guard 3**: a library with two roots whose
  **second** walk fails removes nothing at all, which is the state a per-root reconciliation gets
  wrong and a whole-library one cannot; a scan interrupted between batches has added some items and
  removed **none**, which plan §6.9 calls the only partial state this feature can leave behind; the
  claim renewed **inside** the batch transaction, asserted by a failed batch leaving the claim at its
  previous instant; and the summary counting **skipped** files and **unplaceable** items as two
  numbers that both differ from zero over the fixture — spec §3.8 separates them, and a build that
  added them together passes every test in which one of the two happens to be zero.
- **And the two seam assertions [plan §8.1](plan.md#81-what-replaces-the-http-boundary-and-how-much-weaker-it-is)
  asks for**, both through the subcommand and not through the function, because this is 003's
  analogue of the wiring 001's audit found twice: a build whose `Resolve` is right and whose
  `ApplyScanBatch` writes the wrong `parent_id` goes red here, and one whose `Reconcile` is right and
  whose removal is applied to the wrong library goes red here.
- **Amended at T13, in five places, and the first two are the same finding from opposite sides.**
  *(1) The claim is taken **after** the reading, not before it.* [plan §6.9](plan.md#69-two-scanners-batching-and-what-a-scan-does-while-the-server-serves)
  said when a claim is renewed and never when it is taken, and its own defence of `staleAfter` — the
  claim is renewed on every committed batch, so the value *"only has to exceed the time between two
  batches"* — is false under the other answer: nothing renews a claim during a walk, so a claim taken
  first would have to outlive the walk of the largest library an operator has. The visible half is
  that a guard's refusal now leaves **no claim at all**, which is what lets an operator fix a mount
  and scan again immediately instead of waiting one out.
  *(2) §6.5's third guard stops claiming "one transaction".* [plan §5](plan.md#5-contracts) declares
  `ApplyScanBatch`, `RemoveItems` and `ReleaseScan` as three methods, and three methods are three
  transactions. What the guard is actually made of is the **ordering** — every failure this feature
  can reach happens before `RemoveItems` is called — and the one state the single transaction would
  not have had is written down rather than left to be found.
  *(3) §6.5's second guard counts **files**, not items.* The section says *"where the previous scan
  recorded at least one"* and leaves what is counted open. A library's own `CollectionFolder` row
  backs no file, so an item count would make a library an operator emptied on purpose — having said
  `--allow-empty-root` once — refuse every scan of that root from then on, with the override as the
  only way to scan a legitimately empty library.
  *(4) §5 declares the scanner.* The listing had `Changes` and no producer of one, so `New`,
  `Scan`, `Options` and the three refusals were this task's to choose. `Options` is a record because
  a third flag added as a third parameter is a call site every caller has to be found at, and every
  refusal names the library **and the root's ordinal and path**, because §7's audience is somebody
  with a shell looking at four libraries.
  *(5) §6.7 gains `--format json` and `--log-level` on `scan`, at T13 rather than T14.* Three of the
  criteria above are about what the store holds after an operator ran a scan, so the verb has to
  exist for them to be assertable at all — and the summary's two counts have to be read out of a
  document, because a test that parses the human table starts constraining prose. `--log-level` is
  001's own flag reused, and it is what makes §3.8's *"files skipped **with the reason**"* reachable:
  the counts are the summary and the per-path reasons are the progress, at debug, since a document
  holding every skipped path of a large library is not a summary. `library.FoldName` is added for
  `--name`, because §3.6's rule that two names differing only in case are one name is the domain's
  and a second lowercaser is a second answer.
- **Twenty-six mutations were run; twenty-five fail a named test and the one survivor is a
  measurement.** Adding `library.Plan.Skipped` to the walk's skip count — the exact error T5's and
  T8's handovers both warn about — **cannot be caught by any test over a real tree**, because the
  walk hands the resolver only paths it has already accepted and that list is therefore always
  empty. A corpus that could tell the two apart would have to reach `Resolve` with no walk in front
  of it, which no scan does. It is the walk's count anyway because the rule is *"report what the walk
  refused"*, and `Changes.Skipped`'s doc comment carries the measurement.
- **And the reusable finding is about the two seams, not about the guards.** A seam test needs a
  corpus in which the wrong answer is not also the right one, and each seam needed a different thing
  for that. The parent seam is asserted over the fixture's **whole** declared parent-child structure
  because *"every parent becomes the library's own row"* is invisible wherever a parent happens to be
  the library's own row — which is most of a `movies` tree. The removal seam needs **two** libraries
  that both hold items, because with one library a removal landing on "the wrong library" lands on
  the right one; and it compares the other library's identifiers one for one rather than its count,
  because a count passes on a build that removed one row and added another. This is T12's *"a corpus
  that cannot distinguish two orderings is an ordering test that asserts nothing"* for the third time
  in this feature.
- **What this does not prove:** that a client asking for a container's children is answered with the
  items whose `parent_id` this task asserted. [plan §8.3](plan.md#83-what-only-becomes-observable-at-005-and-what-005-must-not-accept-as-proven)
  row 3, and it is 005's. Nothing here says whether a container with nothing under it is *offered*
  either — row 6, still established by nobody.
- **Spec reference:** §3.8, AC-12, AC-16; plan §6.5, §6.9, §7, §8.1.

## T14 — `atrium library`, which is the operator's only interface and therefore a contract

- [x] **Changes:** `internal/app` — `RunLibrary` and the six verbs of
  [plan §6.7](plan.md#67-configuring-a-library-given-there-is-no-route-to-do-it-with), with
  `--format json` on `list` and `scan`; `cmd/atrium` — one more arm on the dispatch it already has,
  and nothing else.
- **Depends on:** T13
- **Verified by:** `add` then `list` reads the library back with its roots in the order given;
  `add` refuses a fourth collection type **listing the three**, and refuses a folded name that exists
  **naming the library that holds it**, because an operator with a shell is this feature's whole
  failure audience (plan §7); **no verb offers `--collection-type` or `--case-sensitive` on anything
  but `add`**, asserted over the parsed flag sets rather than by reading the source — 002 T7 asserted
  the absence of `--password` the same way, and here the absence is how spec §3.6's refusal is
  implemented rather than a precaution beside it; `remove` followed by `add` with the same name and
  the same roots yields a **different** library identifier and therefore different item identifiers,
  asserted as the inequality it is, because that is §3.6's sharpest consequence and a test asserting
  that `remove` works cannot see it; `rename` and `roots` leave every identifier unchanged, which is
  the same criterion from the free side; `scan --format json` emits spec §3.8's summary as a document
  a test parses while the human table is parsed **nowhere**, since a test that parses prose starts
  constraining prose; and `atrium --data-dir …` with no subcommand still serves while `atrium user
  add` still works — the regression a second arm on a first-argument dispatch is most likely to
  introduce, which 002 T7 paid for once and which gets cheaper to re-assert than to rediscover.
- **Amended at T14, in five places. The first is a debt this list left, and it is the largest.**
  *(1) The scheduled scan and the start-time rescan are this task's, and they were nobody's.*
  [plan §3](plan.md#3-modules) gives `internal/app` *"the `library` subcommand (§6.7), **the
  scheduled scan**, and the **start-time rebuild-and-rescan of §6.8**"*; §6.9 specifies
  `--scan-interval` and §6.8 says a generation bump *"enqueues a full scan of every library"*, which
  `store.DerivedRebuilt()` exists to signal and which T13 found is **logged and acted on by
  nobody**. No task in this list owned either, and the reason is worth more than the correction:
  **this list was written from spec §3 and §5, and the schedule appears in the specification only in
  §2's scope note** — beside filesystem watching, in the sentence that puts filesystem watching out
  of scope — while the rescan appears only in ADR-0003. A behaviour named in a scope note is a
  behaviour no task list derives. They are taken here rather than deferred because T14 is the last
  task in this feature that adds production code (T15–T18 are tests, T19–T20 documents), so a
  deferral would have been the second time nobody owned them. **Verified by:** two installations
  started from the same tree at the same moment, one with a schedule and one with
  `--scan-interval 0`, where the scheduled one having scanned is what times the assertion that the
  other has not — a *"the items turn up eventually"* assertion passes on any build where something
  else scanned; and, for the rescan, a film deleted from the tree between a scan and a start whose
  generation was moved, so that a build ignoring the rebuild holds no items, a build that never
  rebuilt holds the departed one, and only the owed **full** scan produces the three that are there.
  `--scan-interval` is 12 hours by default, which is the reference's own interval for the same task
  `[source: Emby.Server.Implementations/ScheduledTasks/Tasks/RefreshMediaLibraryTask.cs:47-54 @ v10.11.11]`.
  *(2) `library.NewID` allocates a library's identity, and `internal/scan` gains one context check.*
  §3.6 requires the identity to be allocated and nothing said what allocates it: it is 16 bytes of
  `crypto/rand`, the installation identity's shape. The context check is one per root in
  `Scanner.read`, because a scheduled scan is cancelled by the server's shutdown and `Walk` takes an
  `fs.FS`, which carries no context — so a stop bounds the *next* root rather than the current one.
  *(3) The fold T10 could not decide is decided.* `library.FoldName` normalises to NFC **before** it
  lowercases, which is `Normalise`'s own order, because §3.6 says normalised means the same thing for
  a path and for a name. Without it `Amélie` is declarable twice and one of the two is unaddressable.
  *(4) `remove` removes the items before the library.* Nothing in the schema does —
  `items.library_id` is a string and not a foreign key — so the obvious implementation leaks every
  row the library ever scanned. The `scan_state` row does outlive it, and plan §6.7 now says so
  rather than leaving it to be found.
  *(5) The dispatch regression is asserted in `conformance/`, which is where `cmd/atrium` is
  observable at all.* One test runs `library add`, then `library list`, then `user add` on the same
  installation, then starts the server with no subcommand — the three arms in the order that makes a
  build whose new arm consumed the argument vector fail on the line after. Nothing else in this task
  touches `conformance/`; 003 T18 still owns that package's own question.
- **Seventeen mutations were run; sixteen fail a named test and the one survivor is a measurement.**
  Removing the wait [Run] performs for the scanning goroutine before its deferred close survives
  every test in the package: no test stops a server while a scan of its own is still in flight, and
  building one would need a tree large enough that a walk outlasts a cancellation, which is a race
  dressed as a fixture. What the ordering prevents is a `database is closed` at the end of a scan on
  a real installation; it is recorded in `scheduledScans.wait`'s doc comment, in the shape T12 and
  T13 both used.
- **What this does not prove:** that a client is ever told a library exists. 003 registers no route,
  so `list` reading a library back is a store round trip and the `CollectionFolder` a client would
  receive is [plan §8.3](plan.md#83-what-only-becomes-observable-at-005-and-what-005-must-not-accept-as-proven)'s
  first row, still 005's.
- **Spec reference:** §3.6, §3.8; plan §6.7, §6.8, §6.9, §7.

## T15 — AC-2, AC-3 and AC-10: the three criteria that are about scanning more than once

- [x] **Changes:** tests in `internal/app` over a real data directory and a built tree, and whatever
  they find. No new production behaviour is expected; if one is needed, it is a finding. **None was
  needed**: all three criteria hold on the build T14 left, and what the task found is two corrections
  to its own *Verified by* line, struck below.
- **Depends on:** T14
- **Verified by:** **AC-2** — scan, scan again: byte-identical identifiers and a summary reporting
  zero added, zero updated and zero removed. **AC-3** — scan, rebuild the derived half through T11,
  scan again, compare identifiers: this is AC-2's criterion **with the store's memory removed**, and
  the mutation that separates them is ~~a derivation that reuses a previous row's identifier when it
  finds one, which passes AC-2 and every unit test in T3 and fails only here~~ **that reuse over an
  identifier that is *allocated* rather than derived**. **AC-10** — scan, move the whole tree to a
  second temporary directory, `library roots` it, scan again: **every** identifier unchanged, and the
  mutation is putting the root's path into the key, which passes AC-2, AC-3 and T3's table and fails
  only this — **and only in its narrow form**. The three are one task because they are one property
  measured three ways and because writing them apart is how one of them ends up asserting a subset of
  another; they are asserted at `app` and not at `library` because the criterion is about what the
  store ends up holding and a function cannot be asked that.
- **Amended 2026-09-05, at this task, in the two places struck above. Both were found by running the
  mutations rather than by writing them down**, and both are the same shape: a mutation named in the
  abstract turned out to be one this repository cannot fail. Five were run, one at a time, each
  against all three criteria, against T14's moved-root test and against the whole of
  `internal/library` `[measurement: 003 T15, 5 mutations, 2026-09-05]`.
  - **Reuse on its own is a no-op and nothing in this project can see it.** Against a correct
    derivation the adopted string and the derived string are the same string, so that build is green
    everywhere — AC-3 included. What AC-3 catches is the reuse **hiding an identifier that is not
    derived at all**: allocated for an item with no previous row, adopted for one that has a row. It
    is stable across a rescan, stable across a remount, right in every table in `internal/library`,
    and different on every installation — which is exactly the *"derived from the item's stable
    identity, never allocated"* of §3.6, and it is red only under AC-3.
  - **The root path in the key has a broad form and a narrow one, and only the narrow one is AC-10's
    own.** The broad form moves every `Movie`, `Episode` and `Audio` identifier and T14's moved-root
    test is red on it too. The narrow form puts the root's path in the key of the two kinds whose
    §3.6 row literally says *"the library root plus the normalised name"* — `Series` and
    `MusicArtist` — and it is green on T14's test, green on `internal/library`'s own moved-root test,
    and red only here. **This is the corpus finding, and it is why AC-10 moves the whole fixture and
    not a tree of films**: both moved-root assertions that existed before this task ask about one row
    of §3.6's five — T14's about a `Movie`, `internal/library`'s about the `CollectionFolder` of an
    empty library. Fifteenth time in this feature that an assertion could not produce the failure it
    was named for.
- **What this does not prove:** that the identifier a client receives is the one that was stored.
  §8.3 row 1, and it is the cheapest debt in the project to discharge — one `/Items` listing at 005
  covers it together with rows 2, 3 and 4. **Left visible rather than discharged**: nothing here
  reads a body, and no test in this change pretends to.
- **Spec reference:** AC-2, AC-3, AC-10; plan §8.4.

## T16 — AC-11 and AC-14: what changes on disk, and the user data that outlives an item

- [x] **Changes:** tests in `internal/app`, plus the store method the middle clause needs — ~~plus~~
  **and the precious table that method writes into**, which this entry did not say and which the
  clause cannot do without. `0004_item_user_data.sql` holds the two nouns of §3.8's own sentence,
  *favourites* and *resume position*, and nothing of 007's other four; [plan §4.1](plan.md#41-the-precious-half--migration-0003_librariessql)
  carries the argument for its being 003's and the rule that 007 **extends** it rather than
  replacing it. The two methods are on the store and deliberately not on a `ports` interface: 003
  declares no domain that reads or writes user data, so a port method here would be a contract with
  no caller above it and would fix the shape of a method 007 has to design.
- **Depends on:** T14
- **Verified by:** **AC-14**, as four mutations of the fixture between two scans, one per row of
  §3.8's change table that the criterion names: a modified file is re-inspected and keeps its
  identity and its user data, an appearing file is added with its ancestors, a renamed file is a
  delete **plus** an add, and a deleted file is removed — with **size and modification time varied
  independently**, because a build reading only one of the two passes a test that varies both.
  **AC-11**, in three clauses of which the middle one is the criterion: delete a file, scan, the item
  is gone; a row written into a **precious** table keyed on that identifier **before** the deletion
  survives the scan; restore the file, scan, and the identifier returns so the association is live
  again. Until 007 exists the precious row is written by the test through a store method rather than
  by a feature, and that is stated rather than skipped — the alternative is a criterion with no test
  until somebody else's feature lands, which is exactly the shape both closing audits have caught.
  **And the risk plan §6.5 names is asserted from the other side**: there is no retention rule and no
  orphan sweep in this feature, so a later feature that "tidies up" user data whose item is gone
  breaks AC-11 and **nothing else in the suite would notice** — this test is the thing that notices,
  and its comment says so.
- **Seven mutations were run, six fail a named test, and the survivor is a mutation that was wrong
  as written rather than a gap** `[measurement: 003 T16, 7 mutations, 2026-09-05]`. Each was applied
  to a scratch copy one at a time and run against all five tests and against the whole of
  `go test ./...`. The table is in the header of `internal/app/library_change_test.go`.
  - **The sweep is caught here and nowhere else, and that is now a measurement rather than an
    argument.** Adding *"delete every `item_user_data` row naming a removed identifier"* to
    `RemoveItems` fails `TestADeletedFilesUserDataOutlivesItAndTheAssociationComesBack` and **not
    one other test in the repository**. It is the only row of the mutation table with nothing in its
    last column, which is what makes this criterion's test the thing plan §6.5's closing risk was
    missing. §6.5's prediction is struck in place with the measurement beside it.
  - **The rename mutation was wrong as written, in T15's exact shape, and running it is what said
    so.** *"Adopt a previous row whose file signal matches"* adopts **nothing**: plan §6.4's signal
    compares the file's **path** as well as its size and its time — a multi-part film's parts are
    one item's files (§3.3) — and a renamed file's path is the one thing that moved. That build is
    green everywhere including here. The build the criterion can fail adopts over the size and the
    modification time **alone**, reports one update where §3.8 requires one removal and one
    addition, and is red on this task's rename test and on T9's own
    `TestARenameIsARemovalAndAnAdditionAndNotAnIdentifierMismatch`. **Sixteenth time in this feature
    that an assertion, or the mutation named beside one, could not produce the failure it was named
    for**, and the second time the *mutation* rather than the assertion was the thing at fault.
  - **The two halves of the signal are separated by exactly the two mutations plan §8.4 names**, and
    each test begins by failing if the half it is holding still did not stand still — `os.Chtimes`
    typed one line up is a test that quietly varies both.
- **What this does not prove:** that any client is ever told about a favourite or a resume position.
  003 registers no route and produces no `UserData` object; what is asserted is the **row**, and
  every rule about what these values mean, when they change and how they aggregate is 007's. The
  test says so in its own closing section rather than leaving a green here to be read as more.
- **Spec reference:** §3.8, AC-11, AC-14; plan §4.1, §4.3, §6.5, §8.4.

## T17 — The forty-seven declared differences, which this project holds the reading of and not the reasons

- [x] **Changes:** the declaration of the forty-seven, one row each; the comparison of
  [reference-fixture-reading.json](../../docs/compatibility/reference-fixture-reading.json) against a
  real scan of the built tree, run in the default job with no Jellyfin anywhere
  ([AGENTS.md §1.6](../../AGENTS.md)); and the correction of one count in
  [conformance.md](../../docs/compatibility/conformance.md).
- **This is the largest single task in the feature and [plan §8.2](plan.md#82-the-declared-inequality-and-the-forty-seven)
  sizes it as one.** This project has the reading. It does **not** have the declaration — the module
  that held it stayed in the source repository — so the forty-seven reasons are written here from the
  reading, from the specifications that cause each one, and from nothing else.
- **Where the declaration lives, decided here because plan §8.2 does not say and plan §8.2 is amended
  with it.** It is a Go table beside the comparison in `internal/app`, not a seventh machine-readable
  file under `docs/compatibility/`. Two reasons: a new artefact there owes a prose twin and a
  row-for-row test ([docs/README.md](../../docs/README.md#paired-files-edit-both-halves-or-neither)),
  and this declaration has no twin to pair with — the prose that explains a row is **this project's
  own specification section**, which the row cites; and
  [conformance.md](../../docs/compatibility/conformance.md#l3--differential) already describes the
  declaration as living *"in that module with its reason"*, so writing it there is the recorded shape
  rather than a new one.
- **Twenty-five of the forty-seven are 004's, and 004 does not exist.** They are declared now,
  because the comparison cannot run without them, and the consequence is written down rather than
  discovered: *a declared difference that has gone away fails too*, so **004's landing turns those
  rows red by design**. That is the rule working, not a defect, and the handover names the file and
  the row shape 004 must edit.
- **The count is forty-seven, and this task takes the correction.** conformance.md states it twice
  and the two disagree: forty-seven in its L2 section, **twenty-six** in its L3 section. Forty-seven
  is 010's D-7, taken 2026-09-02, and is the number [CLAUDE.md](../../CLAUDE.md), specs/README.md and
  010's own spec all carry. **It is 003's to take rather than 010's** on 002 T22's test — *this
  feature's own work is what makes the number checkable*, and an implementer who took twenty-six
  would declare twenty-one differences too few and fail a run for the wrong reason, which is a cost
  paid by whoever reads that sentence next. Struck in place with the date and the reason
  ([AGENTS.md §4](../../AGENTS.md)), never rewritten. **The pairing is not disturbed**: conformance.md
  L3 is one third of a three-way pairing with [allowlist.yaml](../../docs/compatibility/allowlist.yaml)
  and 010 §3.3, the stale sentence describes the **fixture-reading** comparison and not an allowlist
  row, and no row of that file is added, removed or re-scoped — so the row-for-row comparison sees
  the same rows before and after. That is the same argument 002 T22 made before striking a row of
  `request-cases.yaml`.
- **Depends on:** T13, T15
- **Verified by:** the comparison is over **type, name and path** and not over identifiers, because
  behaviours §1.4 already establishes those differ by design and comparing them would declare 74
  differences that say nothing; an **undeclared** difference fails, asserted by removing one
  declaration and watching it go red; a **declared difference that has gone away** fails, asserted by
  declaring one the two readings do not have — this is the assertion an equality cannot make and the
  one that makes the table a record of decisions rather than a list of excuses; every row names a
  reason as a **specification section** and an **owning feature**, and a row naming neither fails the
  table's own load, which is `allowlist.yaml`'s rule applied to a different file for the same reason;
  the four shapes 010's own amendment names are each present with the right owner — the zero-byte
  film (003 §3.2), the twenty-five differently-named files (004), the empty library (003 §3.1) and
  every library's root row (003 §3.1, `CollectionFolder` against the reference's `Folder`); ~~the
  case-insensitive pair of files is among them ([U-44](../../docs/compatibility/reference-target.md));~~
  and the **total is asserted as ~~forty-seven~~ the declaration's own length**, so a row
  deleted to make a run go green is a failing count rather than a quieter suite.
- **Two clauses struck 2026-09-05, by running them.** Neither is a clause the task declined; each is
  one the measurement refused.
  - **The case-insensitive pair cannot be among them.** T1 already found the tree holds no
    case-only-differing name and plan §8.5 records why building one would be drift; T17 confirms it
    over *both* readings and asserts the absence instead. Manufacturing the pair would give Atrium an
    item the recorded reading has no row for — a difference in the wrong direction, added to make a
    list come out. [U-44](../../docs/compatibility/reference-target.md)'s own claim that the
    difference *"is one of the forty-seven"* is struck there too, and the row stays owed a probe.
  - **The total is thirty-two and not forty-seven.** Forty-seven is 010's D-7, counted over the six
    libraries the fixture composes; **two of those six are 008's media world and no run in the
    default job can build them**. Thirty-two are declared over the four this feature builds, eight
    more are predicted over `Films` and `Tunes`, and the remaining **seven are not derivable from
    the recorded reading and this project's specifications** — which is all this project has, since
    the module holding the original declaration stayed in the source repository. The mechanism the
    clause is for is unchanged and is what the assertion keeps: the total is read from the
    declaration's own length. **Inventing seven rows to reach a number is exactly what that
    assertion exists to prevent, one direction round.** Plan §8.2 carries the arithmetic, and
    conformance.md records it beside the corrected forty-seven.
- **Spec reference:** §3.1, §3.2, §3.6, AC-1; plan §8.2; [010 AC-2](../010-conformance-harness/spec.md#5-acceptance-criteria).

## T18 — What `conformance/` can prove about a feature with no routes, and what it cannot

- [x] **Changes:** `conformance/` — the four assertions of plan §8.1's table, over the built binary
  and a tree written by `go run ./tools/build_library_fixture -into <dir>` as a **subprocess**: a
  subprocess is not an import, and `tools/check_conformance_imports` reads `go list -deps` rather
  than a process tree (plan §3).
- **Depends on:** T14, T17
- **Verified by:** `library add`, `list` and `remove` end to end through the binary, including that
  the flag set offers no way to change a frozen column — the operator's interface asserted where an
  operator stands; a scan of the built tree reporting the counts of spec §3.8; a **second** scan of
  an unchanged tree reporting no changes, which is AC-2's second half made a fact about the binary
  rather than about a function; `go run ./tools/check_conformance_imports` still passing, which is
  what makes *"the fixture reached `conformance/` without an import"* a check instead of an
  intention; and the **L0 registration check staying green with no new rows** — both halves still
  finding exactly the eleven rows of 001 and 002 and nothing else, which is how a feature proves it
  added no route and is the reason plan §10 refuses to add one to make a test possible.
- **And the limit is asserted rather than assumed.** Not one of those assertions can see an item's
  shape: no field name to be PascalCase, no `null` to be absent, no integer that could have been a
  string, no key order. **The instrument is not weaker than the HTTP boundary at catching those; it
  is inapplicable, because the things Principle VIII exists to catch do not yet exist.** What it is
  weaker at is everything §8.3 lists, and this task's own comment says so, so that a green
  `conformance/` package is not read here as evidence for anything on that list.
- **Two things running the assertions found, 2026-09-05** `[measurement: 003 T18, 18 mutations,
  2026-09-05]`. Neither is a clause this task declined; each is one the measurement corrected.
  - **"Staying green" was not an assertion, and plan §8.1's own word for both halves —
    *unchanged* — is struck there.** Both halves of the L0 check derive what they expect from
    `surface.yaml`, so a route registered **together with a row in that document** satisfies both by
    construction: measured, the two derived checks stay green on a build that registers a
    `POST /Library/Refresh` and declares it. So each half gains a **literal** of the eleven rows,
    which is the thing a reviewer who means to add a route has to edit. Three other checks fail on
    that build for reasons of their own, and not one of them is a statement about this feature having
    added a route.
  - **The second scan's control was relative where it had to be absolute.** Comparing the second
    scan's file counts against the **first scan's** is satisfied by a build reporting nothing
    examined for every scan — both readings zero, both equal — and it was the only survivor of the
    first mutation run. The comparison is now against the fixture's declared counts. Fourth time in
    this feature that an assertion about an absence needed a control, and second time that a control
    named in prose turned out to have nothing to fail.
- **Spec reference:** §6; plan §8.1, §8.3, §10.

## T19 — The cross-document debts: which of them are this feature's

- [x] **Changes:** the cross-document debts the plan and this list recorded and did not fix, each
  **decided** rather than listed.
  - **Taken at T17, and recorded here as taken:** conformance.md's stale *"twenty-six places"*. The
    argument for its being 003's is in T17; it lands there rather than here because Principle III
    moves documentation in the same change as the thing that forced it, and the thing that forced it
    is a declaration whose length is forty-seven.
  - **Taken here.** [behaviours §5.2](../../docs/compatibility/behaviours.md#52-a-container-that-has-lost-every-file-is-not-removed)
    and [§5.6](../../docs/compatibility/behaviours.md#56-a-default-rescan-does-not-notice-a-replaced-poster)
    cite `003 plan §6.4` and `§6.5`, which were two of PROVENANCE's *"links with nothing to point
    at"* and now resolve — §6.5 by anchor and §6.4 by file. Check both, and check that §6.5 really
    carries **three** guards, because the sentence citing it counts three. **Nothing is repointed and
    nothing is deleted** ([AGENTS.md §4](../../AGENTS.md)); the sentences beside those citations were
    written about the exporting project's plan at those numbers, and what stands under them here is
    this project's.
  - **Taken here.** [specs/README.md](../README.md) and
    [012 §7.1](../012-negotiation-inputs/spec.md#71-what-measuring-corrected-in-this-document) both
    cite `003 tasks.md#what-the-gate-changed`, which this file makes resolve. What stands under that
    anchor is [this project's own gate record](#what-the-gate-changed), with the same caution: the
    two sentences citing it describe the *exporting* project's gate, and the section they now reach
    is this one's.
  - **Not taken, because it does not exist.** This feature owes **no** row in
    [allowlist.yaml](../../docs/compatibility/allowlist.yaml). An entry there is scoped to an
    endpoint and a JSON Pointer, 003 has no endpoint, and every difference it produces is declared in
    T17's table instead. It is written down because 002's plan overcounted its own allowlist gaps by
    looking for endpoint names, and the opposite error is available here: a reader counting what 003
    owes that file would find three plausible-looking holes and all three belong to 005.
  - **Not taken, with the reason, and handed on.** [U-42, U-43 and U-44](../../docs/compatibility/reference-target.md)
    each need a running reference, which this run does not have. What this task does is check that
    each register row names a **request** and that this feature asserts the divergence as a
    divergence — so the day the probe runs, a failing test is the notification. U-43 is the one to
    run first, and the register already says why.
  - Write **[What this feature owes the next ones](#what-this-feature-owes-the-next-ones)** with
    these, with §8.3's six rows and the request that discharges each, and with anything T1–T18 found,
    in the shape 002 used.
- **Depends on:** T17, T18
- **Verified by:** every internal link in the edited documents resolves to a file and an anchor that
  exists; the corrected count is **struck in place** rather than replaced, so the file still records
  what it believed; `allowlist.yaml`, `request-cases.yaml` and `named-comparisons.yaml` all still
  load and still carry the same number of rows as before the change, which is the mechanical half of
  *"the pairing is not disturbed"*; and each debt not taken names its owner and the measurement or
  the request that owner will need, because a debt handed on without the thing that settles it is a
  debt nobody can close.
- **Amended 2026-09-05, on doing it. The first two are what the checks found; the rest are what this
  line did not name.**
  - **All four citations resolve, and what had to be checked was the shape of what they now reach.**
    [plan §6.4](plan.md#64-change-detection) is *Change detection* and
    [§6.5](plan.md#65-the-guard-against-a-mass-delete) is *The guard against a mass delete*, and
    §6.5 carries **three numbered guards** — which is what behaviours §5.2's *"none of the three
    guards"* counts, so the sentence beside that citation is true of this project's plan as well as
    of the one it was written about. Both `003 tasks.md#what-the-gate-changed` citations reach
    [this list's own gate record](#what-the-gate-changed). Measured with a link check over the five
    citing documents and both of this feature's own: the only unresolved links in any of them are
    **PROVENANCE's other dangling targets** — 006's, 008's, 009's and 011's plans, five task lists
    that do not exist, `tools/README.md#probes` and `pyproject.toml` — and **not one of them is
    003's**. Nothing was repointed and nothing deleted ([AGENTS.md §4](../../AGENTS.md)).
  - **The three artefacts load and are untouched**: `allowlist.yaml` 85 entries,
    `request-cases.yaml` 86 cases, `named-comparisons.yaml` 20 comparisons, the same counts before
    and after `[measurement: 003 T19, 2026-09-05]`. This change edits none of the three, which is
    the strongest form of *"the pairing is not disturbed"* available.
  - **The three plausible-looking allowlist holes, named — because *"a reader would find three"* is
    not a warning until it says which three.** They are `/Items/-/Name`, `/Items/-/SortName`
    together with the order of `/Items`, and `/TotalRecordCount` over the fixture library. Each
    looks like 003's, because 003 decides an item's name, its sort key and which items exist at all.
    **All three are 005's, and the reason is the pointer rather than the cause**: every one of them
    names a body on `GET /Items`, which is [surface.yaml](../../docs/compatibility/surface.yaml)'s
    row for **005** and this project's only `L3` listing. And the first is not an allowlist entry in
    anybody's hands: a name difference is one a server *chose*, and that file refuses a derivation
    class as the excuse for a chosen value — it would need a `behaviours §N`, which is the hole 001
    filled with §4.5 and 002 checked for and did not have. **003 has neither, because it has no
    endpoint**, and the twenty-three name differences are declared in T17's table instead, where the
    reason is a specification section and the owner is named.
  - **The register rows name a *scan of a tree* and not a request, and that is the register being
    right rather than a row being short.** [reference-target.md](../../docs/compatibility/reference-target.md)
    says so in terms — U-42 to U-44 are *"the first rows about something other than a response"*,
    and *"the run that settles them is not a request but a scan of a tree"* — and each names the
    tree: one library holding an empty marker, a marker one directory up and a marker with one
    pattern (U-42); two files differing by ` - a`/` - b` and two more by ` - cda`/` - cdb` (U-43);
    two files differing only in case, on a server whose setting has not been touched (U-44). **The
    clause of this task's own line that says *"names a request"* is the wrong noun for a feature
    with no routes, and the register is what it should have said.**
  - **Two of the three land on a failing test the day they are measured and the third cannot, which
    is the finding this check was worth running for.** U-42 is asserted as a divergence by
    `TestAMarkerAboveTheLibraryRootExcludesNothing` and `TestANonEmptyMarkerExcludesNothing`, each
    with the control T8 wrote for it; U-43 by `TestABareTrailingLetterIsTwoFilmsAndACdLetterIsOne`,
    whose doc comment says in terms that it is written to go red the day somebody measures it.
    **U-44 has only its Atrium half** — `TestCapitalisationIsTheWholeOfWhatCaseSensitiveDecides`
    asserts that two capitalisations of one name derive one identifier in the default library — and
    there is no reference half for it to disagree with, because T1 and T17 both found the fixture
    holds no case-only-differing pair and that building one would be drift. So a probe that settles
    U-44 lands on a register row and a specification section and on **no failing test**, where the
    other two land on a red build. That asymmetry is now a paragraph in the register, beside the one
    that already says U-43 is the row to measure first.
  - **The stale count in this list's own definition of done is struck**, which is T17's correction
    reaching the last sentence that still carried the old number. **T17's heading is deliberately
    left alone**: [plan.md](plan.md) cites `#t17--the-forty-seven-declared-differences-…` twice, and
    renaming a section to correct one word inside it would break two links to fix none.
  - **T16 left one question to this task and it is answered.** Spec §3.8's *"directory emptied"* row
    has no acceptance criterion, and T16 asked whether that is a gap in §5 or a deferral. **It is
    the deferral, and it is already placed**: it is [plan §8.3](plan.md#83-what-only-becomes-observable-at-005-and-what-005-must-not-accept-as-proven)'s
    sixth row, and this list's own table says it is established here by nothing. A criterion of this
    feature's could only assert that the container is **retained**, which is T9's and is asserted;
    what §3.8's row is about is whether a user ever **sees** it, and this feature has no observer to
    answer that. Writing an AC-17 for it would be a criterion about an absence with nobody to
    notice — 001's and 002's audits' own shape, one turn worse. It stays 005's, item 9 below.
- **Spec reference:** plan §8.2, §11; [docs/README.md §Paired files](../../docs/README.md#paired-files-edit-both-halves-or-neither).

## T20 — The closing audit

- [ ] **Changes:** whatever this task finds. It is not a formality:
  [AGENTS.md §5](../../AGENTS.md) records that every implemented feature in the exporting project
  found, in its own final task, **an acceptance criterion with no test or a test proving less than
  its name** — and both features this repository has implemented found one each, twice over the same
  shape.
- **Depends on:** all of the above
- **Verified by:** five passes, each recorded with what it found **or that it found nothing** —
  - **(a) Every one of the sixteen acceptance criteria mapped to a named test that fails when the
    *behaviour* is broken, verified by mutating the production code rather than by reading.** A
    mutation that merely deletes a function is not on the list, because a test that fails only when
    code is missing is a test of the build. **Three shapes are hunted by name.** The first two are
    what 001 and 002 each found: a criterion about a **request** proven about the **mechanism** that
    serves it (001's F-1, 002's F-1 — a `slog` line between the body and the redacting type printed
    a password with the whole suite green), and a criterion about *"the same bytes"* proven about an
    **echo** rather than about a response (001's F-2). **003 has no request, so the shape it takes
    here is named rather than borrowed: a criterion about what the store ends up holding, proven
    about the function that computes it.** The likeliest instances are guessed here so the guess can
    be wrong the way 002's was, which is the useful part: **AC-13**, whose fourteen-row table is the
    most beautiful pure-function test in the feature and which says nothing about any stored key;
    **AC-1**, discharged twice at two levels and at neither a client; and **AC-4**, whose *"one item
    with two media sources"* is one `items` row and two `item_files` rows here and is
    `MediaSources` at 008.
  - **And a fourth pattern, from [002's T22](../002-authentication-users-and-sessions/tasks.md#t22--the-cross-document-debts-which-of-them-are-this-features):
    a correction that narrows a claim instead of testing the narrower one is how a claim outlives its
    refutation.** This feature narrowed four claims, and each is checked for a test **of the narrower
    claim** rather than for the words having got smaller: spec §3.2's `.ignore` rule narrowed to the
    empty marker and to the search bounded at the root (T8); spec §3.3's part-marker parenthetical
    narrowed to the source's vocabulary (T5); spec §3.5's name fallback narrowed by the tie-break
    that reads an ambiguous name as saying less (T7); and plan §6.1's *"files being written"*
    narrowed from a property of a file to a property of a pair of scans (T9). A narrowing with no
    test is the claim still standing.
  - **(b) Every paragraph of spec §3 either tested or listed as untested with a reason.** *Tested*
    means at least one named test fails when the paragraph's behaviour is broken.
  - **(c) The levels stated rather than ticked.** Spec §6 declares **L2** for five behaviours and
    names no endpoint, and L0 and L1 do not apply because there is no route. What must be said
    plainly is the other half: **§8.3's six rows reach no level here at all**, and no task and no
    definition-of-done line may claim otherwise. This is 001's and 002's *"a route reaches a level
    only as far as its states are reachable"*, applied to a feature whose states are not reachable
    from a client at all.
  - **(d) The register.** Anything this feature asserts and has never measured goes to
    [reference-target.md](../../docs/compatibility/reference-target.md) beside U-42 to U-44, rather
    than into a plan paragraph — that register exists because four of 001's tasks wrote *"this
    belongs in the register"* and nobody owned the document.
  - **(e) What implementation taught, written back.** Into `spec.md` in **this same change**
    (Principle III), and any newly *measured* reference behaviour into `behaviours.md` with
    provenance. **The first half of that is expected to be empty and the reason is worth stating:**
    no probe can run in this repository today, so every reference claim this feature acquires is a
    source reading, and a source reading is not a measurement ([AGENTS.md §1.3](../../AGENTS.md)).
- **Spec reference:** all of §5; §6; AGENTS.md §5.

---

## What the gate changed

*The anchor two documents already cite. [specs/README.md](../README.md) and
[012 §7.1](../012-negotiation-inputs/spec.md#71-what-measuring-corrected-in-this-document) both point
here, and both sentences were written about the **exporting** project's task list at this path — the
same situation [plan.md](plan.md)'s own preamble handled for behaviours' three citations of its §6.4
and §6.5. What stands here is this project's own record. The gate proper is
the review that moves this list to `Accepted`, and what it finds lands here; what follows is what
**writing** the list already changed, and each entry says what forced it.*

**Two amendments to `spec.md`, and they are one finding.** Writing a task per criterion is what made
it visible: [plan §6.5](plan.md#65-the-guard-against-a-mass-delete)'s **second guard has no
acceptance criterion**, and it is the guard that catches the failure §3.8 calls *"the single most
destructive thing a scanner can do"* in the form that failure actually arrives in. §3.8's rule is
conditioned on a root that *"cannot be read at all"*, and a share that fails to mount is usually
perfectly readable — the mount point is an empty directory, the walk finds nothing, and the scan
computes the deletion of the whole library. A list written from §5 alone would have tested the
destructive case that is easy to construct and not the one that happens. So:

1. **§3.8 states the second guard**, as behaviour rather than as a mechanism: a root that reads as
   holding no candidate file, where the previous scan of that library recorded at least one, is
   treated as unavailable rather than as emptied; the scan refuses, names the root and changes
   nothing; and an operator can say explicitly that they meant it. Zero is the threshold because it
   needs no number. The flag that carries the operator's permission stays in the plan, where a
   mechanism belongs.
2. **§5 gains AC-16**, in both halves — the refusal, *and* the override proceeding — because a test
   asserting only the refusal passes on a build whose override does nothing.

**Four amendments to `plan.md`, each forced by a task having to name something the plan left open.**

1. **§6.5's *"it is the only one of the three the specification states"* is corrected to two.** It
   was true when written and the amendment above is what falsified it.
2. **§6.8 names four of 001's assertions, not one.** The plan named `migrate_test.go`'s literal `0`.
   Reading the runner for T11 found three more that stop being true in the same change:
   `TestLoadLineageReadsAHalfWithNothingInIt`, which stays true and stops meaning anything;
   `TestTheRunnerAppliesOnlyWhatIsPending`, which proves the runner takes a half **by migrating the
   derived one**, a call that stops existing; and the doc comments on `Half`, `Derived` and
   `migrate`'s refusal, all three of which say the derived lineage is empty and that the refusal is
   owed a replacement at 003. A plan that names one leaves the other three to be met as a red build
   by somebody who will assume they caused it.
3. **§8.2 says where the forty-seven declarations live, and that twenty-five of them are 004's.**
   The plan sized the task and did not place the file; T17 places it, with the argument against a
   seventh paired artefact. And the consequence of 004 owning twenty-five rows of a table 003 writes
   is stated: *a declared difference that has gone away fails*, so 004's landing turns those rows red
   **by design**, which is a fact a reader should meet in the plan rather than in CI.
4. **§8.4 gains AC-16's row and its count becomes sixteen.**

**Nothing else was amended, and two things were deliberately left alone.** [ADR-0003](../../docs/decisions/0003-sqlite-as-the-store.md)
is not edited although T11 closes the gap its *"a derived-version mismatch at startup is a rescan
rather than an error"* left open: an accepted ADR is immutable and this one is not wrong — it is a
record of a decision as taken, and 002 T6 handled ADR-0006's own *"asserted nowhere"* line the same
way. And spec §3.6 is not amended for [U-44](../../docs/compatibility/reference-target.md): it
already states Atrium's default as a decision and says the reference's default is unmeasured, which
is still true, and OQ-9 is where the source reading belongs.

---

## What this feature owes the next ones

*Written at T19, from what T1–T18 **found** rather than from what the list predicted they would.
Fourteen items. Each names what is owed, **who owns it**, and the measurement or the request that
owner will need — because a debt handed on without the thing that settles it is a debt nobody can
close. Where a task refuted a row of the draft this section replaces, the refuted words are struck
and kept: [002's T22](../002-authentication-users-and-sessions/tasks.md#t22--the-cross-document-debts-which-of-them-are-this-features)
found that a correction which narrows a claim instead of testing the narrower one is how a claim
outlives its refutation, and a row rewritten silently is that failure one document earlier.*

1. **The six claims of [plan §8.3](plan.md#83-what-only-becomes-observable-at-005-and-what-005-must-not-accept-as-proven)
   — owner: 005, except the fifth, which is 008's.** The table at the top of this list says which
   task establishes the lower half of each. Two rules go with them, and both are addressed to
   whoever writes 005's plan. **A green suite here is not evidence for any of the six**; each needs
   an assertion at the HTTP boundary, and 001's audit says how to check that the assertion is real —
   break the wiring, order by `name` instead of `sort_key`, write the wrong column, and watch a test
   go red. **The request that discharges each is named, because "005 will cover it" is not a debt
   anybody can close:**
   - **Rows 1 to 4 are one request.** A single `/Items` listing over the fixture library, compared
     byte for byte, covers the identifier, the order, the parent and the four numbers at once. It is
     the cheapest debt in the project to discharge and the easiest to leave open, because every one
     of those values will *look* right in a body somebody eyeballs.
   - **Row 5 is 008's**, and its request is a `POST /Items/{itemId}/PlaybackInfo` — or any body
     carrying `MediaSources` — for the fixture's multi-part film, asserting **two** sources whose
     paths are the two parts and whose order is the part ordinals T12 round-trips as `1` and `2`.
   - **Row 6 is 005's entirely**, and its request is a `/Items` listing taken after a series'
     episodes have all been removed and a scan has run: T9 proves the container is **retained**, and
     what nobody has established is that the listing declines to offer it. That is
     [behaviours §5.2](../../docs/compatibility/behaviours.md#52-a-container-that-has-lost-every-file-is-not-removed)'s
     closing half, and item 9 below is the rest of it.

   **The deferral is now stated in six places** — the package comments of `identity_test.go`,
   `sortkey_test.go`, `shows_test.go` and `music_test.go`, the header of `internal/app/
   library_test.go`, and the header of `conformance/library_configuration_test.go` — with the
   phrasing 005 should not soften: the process boundary is **not weaker** than the HTTP boundary at
   catching a PascalCase field name, a `null` that should be absent, a stringified integer or a key
   order. It is **inapplicable**, because 003 produces no wire representation at all. A weaker
   instrument leaves a residual risk to argue about; an inapplicable one has nothing yet to apply.

2. **~~Twenty-five of T17's forty-seven~~ Twenty-three of T17's thirty-two declared differences are
   004's, and 004's landing is supposed to break them — owner: 004.** (Corrected at T17, which wrote
   the declaration; plan §8.2 carries the arithmetic between its thirty-two and 010's forty-seven.)
   They are every difference about *which name an item carries* — twenty files the reference names
   differently, plus two album directories and the series the reading calls `tvshow` — and 003
   declares them because the comparison cannot run without a reason for every difference. The rule is
   that a **declared difference that has gone away fails too**, so the day 004's metadata resolution
   changes an item's name, the row declaring that difference goes red. That is the mechanism working.
   **The file is `internal/app/reference_reading_test.go` and the thing to edit is the
   `declaredDifference` literal in `declaredDifferences`**: a row is `Library`, `Where` (the item's
   root-relative path, or `theLibrarysOwnRow`, or `"(no path) Type: Name"`), `Kind`, the `Reference`
   and `Atrium` sides as `"Type: Name"`, a `Reason` naming a specification section, an `Owner` and a
   `Because`. Both sides are **part of the declaration**, so a name that moves on either fails a
   third way — reported as *"no longer reads the way the row says"* — rather than silently matching
   the same row. Change the `Atrium` side to the name 004 now produces, or change `Kind` and drop the
   row's two sides if the difference closes entirely, and keep `declaredDifferenceTotal` equal to the
   table's own length: a row removed to make a run go green is exactly what the count assertion
   exists to catch. **One of the twenty-three has no established cause** — no path-derived rule makes
   `tvshow` out of a directory called `The Series`, the tree carries no sidecar, and planting one to
   make the name come out would be drift. Its `Because` says so, and if 004 finds the real cause that
   is the row whose reason to correct.

3. **Seven of the differences 010 counted are not derivable from anything this project holds, and
   that gap is a live question — owners: 010 for the reading, 004 for not closing it the wrong way.**
   010's D-7 counted **forty-seven** against the *other* implementation over **six** fixture
   libraries. T17 derives **thirty-two** over the four `internal/libraryfixture` builds; a further
   **eight** are predicted over `Films` and `Tunes`, which are 008's ffmpeg world and which no run in
   the default job can build; **seven are left over**. The reason they cannot be derived is
   measurable rather than mysterious: [reference-fixture-reading.json](../../docs/compatibility/reference-fixture-reading.json)
   records exactly four fields per item — `type`, `name`, `file` and `path` — so a difference in any
   *other* field is invisible to the comparison however real it is. T7 already found one of that
   shape and could not declare it: **Atrium puts a `ProductionYear` on an album where the reference
   does not, and the reading has no year field to hold either side.**
   - **What 010 needs to do:** re-record the reading against a single-use instance
     ([ADR-0007](../../docs/decisions/0007-a-container-runtime-for-the-reference-instance.md)) with
     **more per item than four fields** — production year, index numbers and sort name are the ones
     003 can already predict a difference in — and re-derive. Rows that appear only once the reading
     carries them are the likeliest shape of the seven. What remains after that is the experiment's
     own answer rather than a bookkeeping error: a difference the Python implementation produced and
     the Go one does not is exactly what diffing the two reports is for.
   - **What 004 must not do:** invent rows to reach forty-seven. A difference 004 produces that the
     reading cannot see is a **real** difference with nothing to declare it against, and it belongs
     in 004's own plan and in a request to 010, not in the table. The total is asserted from the
     declaration's own length precisely so that reaching a number is not a way to make a run go
     green — and this is that assertion working in the direction nobody expects.

4. **`items.name` has one writer today and 004 adds a second — owner: 004's plan.** A refresh
   overwriting what a scan resolved, or the reverse, is out of this feature's hands and is named in
   [plan §9](plan.md#9-risks). [behaviours §5](../../docs/compatibility/behaviours.md#5-accepted-gaps-in-v1)'s
   rename row already records the same fight from the editing side, so the argument exists and what
   is missing is the decision about which writer wins.

5. **The `TagSource` seam is declared here and implemented by 004 — owner: 004.** It is
   `ResolveWithTags(lib, readings, TagSource)` in `internal/library/resolve.go`, and `Resolve` is
   that function with `NoTags{}`; the source is consulted **once per file, before grouping**, because
   the album artist decides which album a track belongs to and grouping cannot be redone afterwards
   (plan §6.2). **There is no error return, deliberately**: a file whose tags cannot be read is
   neither a skipped file nor an unplaceable item — spec §3.8 counts two things and no third — it is
   an item resolved from its path, and reporting the failure belongs to the feature that owns the
   reader. Three things 004 inherits with it. **`SortTitle` is deliberately not in `Tags`**: T4 put it
   on `ports.ScannedItem` as the input to spec §3.7.3, which applies to all eight kinds, where
   `TagSource` answers a music file's five, so setting one means mutating items after `Resolve`
   returns and re-deriving `SortKeyFor`, or a second pass — neither is written. **The precedence is
   three ranks and not two** — a tag outranks a directory, a directory outranks an inference, and
   §3.5's *"only if"* is what makes track artists fill a hole rather than overrule; the wider reading
   turns an ordinary album with a guest on every track into a compilation. And **spec §3.5's measured
   precedence rule — 413 tracks whose album name cannot have come from the path — is exercised by
   nothing in 003**: the stub source proves the grouping key and the precedence order and nothing
   about a real reader over a real library, and both the package comment and plan §8.5 say so.

6. **Two identifier collisions are unreachable today and 004 makes one of them reachable — owner:
   004, and the store already decided what happens.** `ApplyScanBatch` refuses a batch naming one
   item twice with `ErrRepeatedIdentifier` (T12), which is T3's finding turned into a decision: NFC
   has singleton mappings, so `K.mkv` (Kelvin sign) and `K.mkv` (plain) are two files on disk and one
   key even in a case-sensitive library, and nothing in `internal/library` can notice because the
   derivation is a pure function of the key. **The second collision is 004's own**: the compilation
   rule runs after the grouping and never regroups, so an album attributed `Various Artists` from its
   track artists and one whose directory already said `Various Artists`, sharing an album name, are
   one identifier — unreachable under `NoTags` and reachable the day a real reader lands. Do not
   close either by deduplicating a batch: the pair is two things in the library and the operator has
   to see it. *(A related fact worth having before somebody writes the wrong test: a library's
   identifier is 16 bytes of `crypto/rand`, so **nothing about item identifiers is reproducible
   across two installations** and any assertion of the shape 002 has for accounts — "two data
   directories provisioned the same way hold the same identifiers" — is false for libraries and
   always will be. Its absence is not a gap.)*

7. **U-42, U-43 and U-44 each need a running reference — owner: whoever next has one.** All three are
   source readings this feature implemented differently or narrowly; the register names the tree that
   settles each, and **the run is a scan and not a request**, which the register says in terms. **U-43
   first**, and not because it is the likeliest: it is the only one of the three whose wrong reading
   *loses an item*. **Checked at T19, and the check found an asymmetry the register now carries:**
   U-42 and U-43 are asserted here **as divergences**, each with a control, so a probe that
   contradicts either arrives as a red build; **U-44 has only its Atrium half**, because the fixture
   holds no case-only-differing pair and T1, T17 and conformance.md's L2 list all record that
   building one would be drift. So U-44's measurement lands on a register row and on no failing test,
   and the run that settles it needs a tree of its own.

8. **007 inherits AC-11's middle clause, and the table it needs already exists — owner: 007.** *"User
   data outlives items"* is a property of [ADR-0003](../../docs/decisions/0003-sqlite-as-the-store.md)'s
   split rather than of this feature's removal pass: no precious row references a derived row id, so a
   removed item's user data keeps naming a string that will exist again. **`item_user_data` is
   `0004_item_user_data.sql` in the precious lineage** ([plan §4.1](plan.md#41-the-precious-half--migration-0003_librariessql))
   — `user_id` a real foreign key, `item_id` a **string** and deliberately not one, `is_favourite`
   and `playback_position_ticks`, keyed `(user_id, item_id)`. **Extend it; do not replace it**, or
   AC-11's assertion is left watching rows nobody writes. The two store methods are
   `sqlite.Store.SetItemUserData` and `ItemUserData` and are deliberately **not** on a `ports`
   interface, because 003 declares no domain that reads or writes user data; declaring the port is
   007's. **And there must never be an `ON DELETE CASCADE` from `items`, nor an orphan sweep**: that
   absence is measured rather than argued — adding *"delete every `item_user_data` row naming a
   removed identifier"* to `RemoveItems` fails **one** test in the repository and no other
   `[measurement: 003 T16, 7 mutations, 2026-09-05]`. Note also that `atrium library remove` is a
   second producer of removals and takes every item of a library at once; whatever notices a sweep
   appearing should notice that too.

9. **005 owns the observable half of [behaviours §5.2](../../docs/compatibility/behaviours.md#52-a-container-that-has-lost-every-file-is-not-removed),
   and 003 proves only that the container is still there — owner: 005.** A series whose episodes all
   vanished keeps its row deliberately; what makes that invisible to a user is `/Items` declining to
   offer a container with nothing under it. The two servers agree about the row
   `[probe: tools/differential.py --named container-that-lost-every-file, Jellyfin 10.11.11,
   2026-09-02]`; nobody has measured the offering. **T19's decision on the question T16 raised: spec
   §3.8's *"directory emptied"* row having no acceptance criterion is the deferral and not a gap in
   §5.** A criterion of this feature's could only assert the retention, which T9 asserts already; the
   row is about whether a user sees the container, and this feature has no observer.

10. **The three sort-name lists and the pad width are part of the contract, and the day one becomes
    editable is a derived-generation bump nothing will remind anybody of — owner: whichever feature
    makes them editable.** `sortArticles`, `sortRemovedCharacters`, `sortReplacedCharacters` and
    `sortPadWidth = 10` are constants in `internal/library/sortkey.go`, and changing any of them
    invalidates **every stored sort key in the database** ([plan §4.3](plan.md#43-which-of-these-is-derived-and-the-two-that-are-not-obvious)).
    **The pair that protects the derived schema does not protect the derivation**, and this is the
    trap: `derivedSchemaDigest` is a hash of `derived/library.sql`, so
    `TestTheDerivedSchemaAndItsGenerationMoveTogether` fails on any edit to *that file* — and the
    sort-name constants are not in it. Change an article, update the byte-exact tables that go red in
    `internal/library`, and **nothing anywhere requires `derivedGeneration` to be bumped**: half a
    library keeps keys derived under the old rule, every item written afterwards uses the new one,
    the two orders interleave permanently and no test fails. Measured rather than reasoned —
    `sortPadWidth` moved from 10 to 11 fails **three tests, all of them `internal/library`'s byte
    tables**, and not one assertion in `internal/store/sqlite` or `internal/app` and nothing naming
    the generation `[measurement: 003 T19, 1 mutation, 2026-09-05]`. Whoever makes one editable owes the bump
    ([plan §6.8](plan.md#68-the-derived-halfs-generation-and-the-rescan-that-replaces-002s-refusal))
    and a check that notices it was not made.

11. **The L0 check has a third blind spot, and it is not 003's to fix — owner: the argument is 001's,
    the maintenance is 004's and 005's.** [001 plan §8.5](../001-server-identity-and-discovery/plan.md#85-routes-against-surfaceyaml)
    designs the check as two views covering each other's blind spots, and
    [conformance.md's L0 section](../../docs/compatibility/conformance.md#l0--routed) says the same.
    **Both views derive what they expect from `surface.yaml`, so a route registered together with a
    row in that document satisfies both by construction** — measured, not argued: registering a
    `POST /Library/Refresh` and declaring it in both copies of the surface document leaves
    `TestTheServerIsReachableOnExactlyTheImplementedRowsOfTheSurfaceDocument` and
    `TestTheRouterServesExactlyTheImplementedRowsOfTheSurfaceDocument` **green**
    `[measurement: 003 T18, 18 mutations, 2026-09-05]`. Neither view's blind-spot column names that
    case, and 003's own definition-of-done line *"proven by both halves staying green"* would have
    been a claim about nothing. What T18 put in its place is a **literal** of the eleven rows 001 and
    002 registered, in `conformance/library_configuration_test.go` and
    `internal/httpapi/registration_test.go`, with a comparison beside each that reports both
    directions. **004 and 005 add their rows to both literals as they land**, and a reviewer who means
    to add a route edits them deliberately. Whether the check itself should gain a third view is 001's
    document to answer, and this feature does not answer it in someone else's plan.

12. **`atrium library scan` is the only way to ask this server for a scan, and that is a measured
    consequence rather than an omission — owner: the feature that measures a client calling the
    reference's route.** `POST /Library/Refresh` is not in [surface.yaml](../../docs/compatibility/surface.yaml)
    because Principle VI keeps an endpoint out until a client is measured calling it. behaviours §5.2
    already records what it costs — *"Atrium cannot be asked for a second scan over the wire at
    all"* — so 010's report names that comparison outstanding on every run.

13. **The fixture is a declaration, and the rule for adding to it is narrower than it looks — owner:
    anyone extending `internal/libraryfixture`, which is 004, 008 and 010 in turn.** **A file added
    must be one *both* servers drop, with the citation of the rule that drops it beside it in `Why`**
    ([plan §8.5](plan.md#85-the-fixture-and-why-it-is-generated-rather-than-checked-in)). A file only
    Atrium drops is a difference [reference-fixture-reading.json](../../docs/compatibility/reference-fixture-reading.json)
    has no row for; a file neither drops is an item it has no row for. Either moves T17's count, and
    **neither is visible in a green run**. `libraryfixture.NotBuiltHere` names `Films` and `Tunes`
    with the reason and is the list to extend rather than duplicate; a library appearing in the
    reading that is in neither it nor `Libraries()` already fails a test. **008 owns `Films` and
    `Tunes`**, and the eight predicted differences of item 3 are theirs to confirm or refute.

14. **A server scans on its own now, so a test that asserts what a store holds over a long-running
    server must say `--scan-interval 0` — owner: anyone writing a `conformance/` test over an
    installation that holds a library, which is 004 and 005 next.** T14 wired the schedule and the
    start-time rescan that plan §3, §6.8 and §6.9 specified and no task owned; the default interval
    is twelve hours and `startServer` in `conformance/` passes none, so nothing is affected in
    practice today. A test that provisions a library, starts a server and then asserts *"the store
    still holds exactly what I put there"* is one small interval away from being flaky for a reason
    that looks like anything but a schedule. **The ownership shape is the finding rather than the
    schedule**: the task list was written from spec §3 and §5, and the schedule appears in the
    specification only in §2's scope note — so anything stated outside §3 and §5 is invisible to a
    list derived from them. AC-16 was the same shape. It is worth one pass over the rest of §2 and §4
    before the next list is written from §3 and §5 alone.

---

## Definition of done

The feature is done when **all** of these hold:

- [ ] Every acceptance criterion in `spec.md` §5 — **sixteen, since this change added AC-16** — has a
  passing test, mapped in T20's pass (a) and each mapping verified by **breaking the behaviour**
  rather than by reading.
- [ ] Every endpoint reaches the conformance level declared in `spec.md` §6. **This feature has no
  endpoints**, so the line is met by there being nothing to meet it with — and that is stated rather
  than ticked quietly, because the same sentence in a feature with routes means something this one
  cannot claim. What replaces it is spec §6's five L2 behaviours over the fixture, plus T17's
  declared inequality against the reference's own reading, which is the strongest check this feature
  has and is not an L-level at all.
- [ ] `docs/compatibility/surface.yaml` lists every route added, and no route exists outside it —
  met here by **adding none**, proven by both halves of the L0 check staying green over the same
  eleven rows (T18). A route added to make a test possible would be a delta added to make a test
  possible (plan §10).
- [ ] Anything learned during implementation is back in `spec.md`, in this same change.
- [ ] Any new measured Jellyfin behaviour is in `docs/compatibility/behaviours.md` with provenance,
  **and anything this feature asserts and has not measured is in `reference-target.md`'s register**
  rather than in a plan paragraph. The first half is expected to be empty: no reference instance is
  reachable in this run, so every reference claim 003 acquires is a source reading.
- [ ] The debt 002 recorded here is discharged and says so: the derived half's schema version stops
  being a literal `0` **deliberately** (T11), which is what 002's handover asked for by name, and
  ADR-0003's *"a derived-version mismatch at startup is a rescan rather than an error"* is
  implemented rather than refused.
- [ ] The ~~forty-seven~~ **thirty-two** differences are declared, the count is asserted from the
  declaration's own length, and both failure directions are shown to fail (T17). **Struck
  2026-09-05 at T19**, which found this the last sentence in the feature still carrying the old
  number: T17 derived thirty-two over the four libraries `internal/libraryfixture` builds, eight
  more are predicted over the two 008 builds, and seven are not derivable from the recorded reading
  and this project's specifications at all — a gap this line must not be read as closing, since
  inventing seven rows to reach forty-seven is precisely what asserting the count from the
  declaration's own length exists to prevent. T17's *heading* keeps the old number on purpose:
  [plan.md](plan.md) cites its anchor twice.
- [ ] `spec.md`, `plan.md` and `tasks.md` are all marked `Implemented`.

**One line of this list cannot hold as written, and saying so is the job rather than a failure of
it.** *"Every acceptance criterion has a passing test"* is true and is less than it sounds, because
[§8.3](plan.md#83-what-only-becomes-observable-at-005-and-what-005-must-not-accept-as-proven)'s six
claims are not acceptance criteria of this feature and never were: they are decisions 003 takes whose
first observable consequence is on somebody else's route. A suite that is green on all sixteen
criteria is entirely consistent with a client receiving the wrong identifier, the wrong order, the
wrong parent and the wrong numeric type — because nothing in this feature has a client. **That is
this feature's largest single weakness and it is structural rather than an oversight**; the list
above is where it is written down, and 005's plan inherits it.
