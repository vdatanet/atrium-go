# Proving parity

Principle VIII: a behaviour is done when a test asserts it **at the HTTP boundary**. This document
defines the four levels referenced throughout the specifications, and the machinery for each.

## The four levels

| Level | Question it answers | Needs a real Jellyfin? |
|---|---|---|
| **L0 — Routed** | Does the path exist and answer a sane status? | No |
| **L1 — Shape** | Are the fields, casing, types and units right? | No |
| **L2 — Semantic** | Are the *values* right for a known library? | No |
| **L3 — Differential** | Is it the same as what Jellyfin actually sends? | Yes |

v1 requires **L2 for every endpoint**, and **L3 for the authentication and playback paths** — the
two places where a wrong server makes a client misbehave rather than merely look wrong.

## L0 — Routed

The cheapest and most mechanical check, and it is generated, not hand-written:
`docs/compatibility/surface.yaml` lists every endpoint in v1, and the test suite asserts that each
one is registered and that no route exists outside the list.

The second half matters as much as the first — it is the automated enforcement of Principle VI. An
endpoint that appears in the router without appearing in the surface file fails CI.

Both halves are checked against **two views** of what the application serves — the OpenAPI document
the framework generates, and the route table the factory builds from its own list of routers — and
against each other. Each view has a blind spot the other covers: a route hidden from the document,
and a router included without being listed. See
[001 plan §8.5](../../specs/001-server-identity-and-discovery/plan.md#85-routes-against-surfaceyaml).

L0 also covers **what counts as the same path**. The reference matches case-insensitively and
accepts one trailing slash ([§1.14](behaviours.md#114-paths-match-case-insensitively-and-tolerate-one-trailing-slash)),
so a client that lowercases its URLs must not meet a `404` here.

## L1 — Shape

Golden-response tests. A request is issued against a fixture library, and the **raw response bytes**
are compared to a checked-in file under `tests/golden/`.

Two rules make these tests worth having:

1. **Compare bytes, not parsed objects.** Casing, `null`-versus-absent and integer-versus-string
   are all part of the contract and all invisible after parsing. A test that does
   `assert body["ItemId"] == ...` cannot fail on any of them.
2. **Golden files are reviewed, never blindly regenerated.** `--update-golden` exists, but a diff in
   a golden file is a contract change and gets read like one in review.

Two cross-cutting L1 checks apply to every route at once, because Principle I says a single lapse
breaks everything:

- **Casing sweep** — walk every registered response model and fail on any field name that is not
  PascalCase. Python's ecosystem defaults the wrong way; this makes forgetting impossible rather
  than unlikely.
- **Unit sweep** — every field whose name ends in `Ticks` must be an integer, and every field whose
  name ends in `Date` must serialise with seven fractional digits and a `Z`.

  > ⚠️ **Both halves of that sentence were measured on 2026-09-02 and both need qualifying**
  > `[probe: tools/probe_wire_format, Jellyfin 10.11.11, 2026-09-02]`.
  >
  > **The suffix does not reach every date.** Of nine date-valued fields observed, three do not end
  > in `Date` — `DateCreated`, `DateLastMediaAdded` and `LastPlaybackCheckIn` — so a sweep keyed on
  > the suffix alone checks six of nine. The rule the sweep needs is *a value that is a date*, and
  > the field name is a heuristic for finding one.
  >
  > **Seven digits is not universal in the reference**, which is [behaviours §1.2](behaviours.md#12-dates-carry-up-to-seven-fractional-digits)'s
  > own correction: `LastPlayedDate` and `LastActivityDate` were observed with three and six. The
  > sweep is still right as a rule for **Atrium** — it is what Atrium emits — but it is a statement
  > about this server, not a shared fact, and an L3 comparison of those two fields will differ.
  >
  > **And the casing sweep has a trap beside it.** A dictionary's *keys* are data, not schema:
  > `ImageBlurHashes` is keyed by image tag, and a sweep that treats every JSON object key as a
  > property name reported **688 of 899 keys** as casing failures in the same run. The sweep must
  > walk the registered response models, not an arbitrary decoded body.

## L2 — Semantic

A fixture library with known content, checked into the repository as **metadata only**: directory
trees and `.nfo` sidecars, written from a declared manifest at test time. No copyrighted media,
ever.

**There are two fixture worlds, because they answer different questions.** The scanning library is
paths and filler bytes: 003 never opens a file, so a decodable one would have bought it nothing and
cost it a build tool. Feature 008 *does* open them — every delivery route reads bytes and every
decision reads a codec — so it brings a second world, tiny synthetic media that ffmpeg really
encodes into each container, codec and sample rate the delivery tests need, generated into a cached
directory and scanned by the real pipeline (`tests/fixtures/media.py`). Tests that reach the
binaries carry the `ffmpeg` marker; `pytest -m "not ffmpeg"` staying green is the check that they
all do.

**Both worlds go across to a reference instance, and that was measured rather than assumed.** 010's
plan defaulted to porting the structural cases into the media world and giving a reference server
one library; the reference in fact makes items out of the paths-and-filler tree as well — it
resolves an item from a path and probes it afterwards, and a probe that fails leaves an item with no
streams rather than no item — so both trees are handed across as libraries of their own, beside one
library with nothing in it at all
`[probe: tools/probe_reference_scan.py, Jellyfin 10.11.11, 2026-09-02]`.
`tests/fixtures/reference_tree.py` is what composes the three, and
`docs/compatibility/reference-fixture-reading.json` is the reference's recorded reading of them,
compared against Atrium's own scan by `tests/library/test_reference_reading.py` with no Jellyfin
anywhere. The comparison is **not an equality**: forty-seven declared differences, where one that is
not declared fails and a declared one that has gone away fails too.

The fixture library covers, deliberately, the cases that break naive scanners:

- A film in a folder, a film as a bare file, and a film with a year in the title.
- A series with specials (season 0), a multi-episode file, and an absolute-numbered season.
- An album split across discs, a compilation with per-track album artists, and a track whose
  embedded tags disagree with its filename.
- Non-ASCII names, names with brackets and dots, ~~and a name that differs only by case~~.
- A directory excluded by an **empty** `.ignore` marker, which is the only kind that excludes
  anything, and a zero-byte file that is an incomplete copy rather than an exclusion.
- A subtitle file in a **legacy single-byte encoding**, which is the one input
  [behaviours §5.11](behaviours.md#511-a-subtitle-file-in-a-legacy-encoding-is-decoded-by-a-rule-and-not-by-a-detector)
  has, and an image carrying an **EXIF orientation** planted beside a film, which is the one input
  the resize edge 006 owes has. Both exist because no remote request reaches either.

**Amended 2026-09-05, by 003's T1 and closed by its T17: the case-only-differing name is struck,
because the tree the recorded reading was taken over does not hold one.** Over the fifty-eight items
[reference-fixture-reading.json](reference-fixture-reading.json) names in `Movies`, `Shows` and
`Music`, **no two differ only in capitalisation**, and a test in `internal/app` now asserts that of
both readings at once. Planting the pair would give Atrium an item the recorded reading has no row
for — a difference in the wrong direction, added to make a list come out — so
[U-44](reference-target.md) stays what its own register row says it is: a claim one scan against a
single-use reference settles, and this project leaves it unmeasured rather than manufacturing a
difference for it.

L2 tests then assert real values: that the album with two discs yields one album and N tracks with
the right `ParentIndexNumber`, that resume position round-trips through ticks, that sort order
matches Jellyfin's normalisation.

## L3 — Differential

The strongest check, and the only one that can catch what we did not think to ask.

**Method:** issue the same request to Atrium and to a real Jellyfin, both pointed at libraries
built from the same fixture tree, and compare the JSON structurally.

```bash
python3 tools/differential.py \
    --atrium   http://localhost:8096 \
    --jellyfin http://your-jellyfin:8096 \
    --surface  docs/compatibility/surface.yaml \
    --report   reference/differential-report.md
```

Six more flags sit beside those: `--identity` (repeatable; the default is the administrator and a
restricted non-administrator, which is [010 §3.9](../../specs/010-conformance-harness/spec.md)'s
minimum), `--fixture` (ask the half that needs a reference instance over this repository's own
fixture), `--fixture-root` (the tree that instance is given as its only library), `--named`
(attempt only these named comparisons, by id — the rest are still *reported*, outstanding),
`--reference-url` (an instance somebody else stood up, instead of one this run makes) and
`--ignored-parameters` (Atrium's data directory, from which the run also writes
`reference/ignored-parameters-<date>.md` — [010 §3.6](../../specs/010-conformance-harness/spec.md)'s
parameter, endpoint, count and client). That last one reads **the tally Atrium wrote when it last
stopped**, which is the only moment the count is complete, and never this run's own sweep: the same
fact that makes it a file in the data directory and not an endpoint Jellyfin does not have. Credentials
come from the
same git-ignored `.env` the probes read: `JELLYFIN_URL`, `JELLYFIN_USERNAME`, `JELLYFIN_PASSWORD`
or `JELLYFIN_TOKEN` for the reference, and `ATRIUM_USERNAME`, `ATRIUM_PASSWORD` or `ATRIUM_TOKEN`
for the server under test.

**A row of [surface.yaml](surface.yaml) declaring `level: L3` is a claim only this program can
pay for, and the report says which seats paid it.** Eight rows declare it, and until 2026-09-02
nothing had read the column at all: `tools/extract_v1_surface.py` checks that the value is one of
`L0..L3` — the vocabulary and not the claim — and the route tests read `feature` and `consumers`.
The report now prints the declared level beside every endpoint, with `**partly**` and the seat names
where a run compared it from **some** of its identities: on a surface where 12 of 23 reads answer
differently to a restricted non-administrator, an endpoint compared from the administrator's seat
alone was compared from the one seat that can be refused nothing, and a `yes` there would be a level
claimed rather than reached.

**The two servers are told apart by the `Server` header and never by `ProductName`.** Atrium
answers `ProductName: "Jellyfin Server"` on purpose
([§4.1](behaviours.md#41-atrium-identifies-as-jellyfin-on-the-fields-clients-parse)), so the
obvious check admits a run pointed at two Atriums — a comparison of this project with itself,
reporting parity it never measured. `Server` is `Atrium/<version>` here against the reference's
`Kestrel` `[probe: tools/probe_routing.py, Jellyfin 10.11.11, 2026-08-28]`, and a pair that fails
that test is refused before anything is compared.

**With `--fixture`, the run stands up its own reference and destroys it.** One container of the
pinned Jellyfin image — pinned by **digest**, [reference-target §1](reference-target.md#1-the-pinned-version)
— over the fixture tree mounted read-only, configured through the reference's own first-time-setup
operations with no human, waited on until its library scan reports itself finished, and destroyed
with everything it wrote on the success path and the exception path alike
([ADR-0007](../decisions/0007-a-container-runtime-for-the-reference-instance.md)). It needs a
container runtime, and **no CI job has one, because no CI job may contact or start a Jellyfin**. A
machine without one loses nothing it had: the sweep still runs, and every case and named row that
needed an instance is reported *outstanding with the reason* rather than skipped. The report header
carries the instance's address and the image digest beside the Atrium sha, so a difference that
reproduces on one machine only can be told from a difference that is real.
`tools/reference_instance.py` stands the same instance up and **leaves it running**, for looking at
a difference by hand.

**Exit codes: `0` clean, `1` not clean, `2` the run could not start.** *Not clean* is the ordinary
answer and not a failure: it means the run has an untriaged difference, a declared case it could
not issue, or a named comparison it did not run. The report is the deliverable
([010 §3.4](../../specs/010-conformance-harness/spec.md#34-the-report)), and it opens with what a
reader may and may not conclude from it — the seats it authenticated as, the cases it did not
issue with the reason for each, and the named comparisons still outstanding. **Outstanding is not
green**: a run that swept every endpoint and skipped nine named comparisons has proved that the
questions it asked have the same answers, which is a smaller claim than it sounds.

**What it compares:** the set of keys at every level, the type of every value, and the value itself
for everything not on the allowlist.

**Which row is which row, and it is not the identifier.** The two servers derive identifiers
differently on purpose ([§1.4](behaviours.md#14-item-identifiers-are-32-lowercase-hex-characters)),
and the obvious way out — comparing by path — is not available either: the reference sends no
`Path` on a default list row at all, and asking for one changes the request under comparison and
still leaves a virtual season, a remote channel and every by-name row with nothing to join on.
`(Type, Name)` is not unique. So rows are compared **by position**, which makes the ordering part of
the contract — and where the reference's own ordering is not total
([§3.6](behaviours.md#36-ties-are-engine-resolved-and-paging-the-artist-sorts-loses-rows--class-b-diverged))
the comparison is of multisets rather than sequences.
`[probe: tools/probe_differential_join.py, Jellyfin 10.11.11, 2026-09-01]`

**A run states the identities it authenticated as.** Every probe written before 2026-09-01
authenticated as an administrator, and an administrator lacks no permission — measured, **12 of 23
reads of the surface answer differently to a restricted non-administrator**, and two of those differ
as *shorter lists* rather than as refusals, which no status comparison would see. A single-identity
run is reported as covering one identity.
`[probe: tools/probe_restricted_surface.py, Jellyfin 10.11.11, 2026-09-01]`

**And a sweep is not the whole method.** The differences a sweep cannot raise — because they need a
caller the run does not have, a library the reference cannot be given, a library **changed between
two scans**, a deliberate **wait**, or a comparison of something that is not in a body — are
enumerated as **named comparisons** in
[010 §3.10](../../specs/010-conformance-harness/spec.md), and an unrun one keeps a run from being
called clean. There are **twenty** of them: sixteen when 010's spec was accepted, and four more on
2026-09-02, when the four readings the compatibility documents and the inherited lists owed —
[behaviours §5.2](behaviours.md) and [§5.6](behaviours.md), 005's OQ-7 and 007's paused-session
ticker — were given this table as their owner rather than left as debts nobody was measuring.

**The twenty are checked in as [named-comparisons.yaml](named-comparisons.yaml)**, one row each,
and `tests/unit/test_allowlist.py` compares the file against 010 §3.10 row for row. Every row
carries a **`needs`**: the seat, the fixture, the rescan, the wait, the latency, the bytes or the
second run without which it cannot be asked at all. That is what lets a report say *"four
outstanding, and three of them because no fixture instance was available"* rather than *"four
outstanding"* — and what stops a row that was never askable from being quietly dropped instead of
counted as a miss.

**Since 2026-09-02 every row also names its `runner`** — the callable in `tools/differential.py`
that makes the comparison, in the six shapes [010 plan §6.4](../../specs/010-conformance-harness/plan.md)
describes. **Naming one is not running one.** A row whose `needs` this run cannot meet is
outstanding with the reason; a runner that raises leaves its row outstanding with the exception and
the run carries on to the other nineteen; and a row that *ran* and measured something the entry it
cites does not predict is an untriaged difference, which keeps the run from being called clean
exactly as a sweep finding does. Two rows are **not** runnable as comparisons at all today and say
so every run: *the library changed underneath a rescan* needs a second scan on both servers, and
Atrium has no library-refresh route — `POST /Library/Refresh` is the reference's and is not in
[surface.yaml](surface.yaml), because Principle VI keeps an endpoint out until a client is measured
calling it. Their runners take the **reference** half and report outstanding carrying it.

**What the sweep sends is checked in too**, as [request-cases.yaml](request-cases.yaml): per
endpoint, a name, a query, a body, a content type, the **anchors** that fill its path parameters
and the identities it is meaningful for. AC-3's floor is one case per endpoint — 59 — and 010's own
gate measured that floor to be *not enough*: both differences it found on `/Items/{itemId}/Similar`
are invisible to a bare request. **The eight `level: L3` rows of [surface.yaml](surface.yaml) are
seeded first**, because that column is a required conformance level and nothing has ever checked
that one is reached — `tools/extract_v1_surface.py` validates only that the value is one of
`L0..L3`, and every implemented feature's definition of done has deferred the differential half
here. Each of the eight carries more than one case and names more than one identity.

**And what the reference makes of the fixture is checked in as well**, as
[reference-fixture-reading.json](reference-fixture-reading.json): every item of every library the
reference builds from this repository's own tree, as a type, a name and the file behind it, with
the probe's citation and the image digest inside the file. It exists because AC-2 — *both servers,
pointed at the same built fixture, produce libraries with the same item count and the same
structure* — needs two servers to be **taken** and only one to be **checked**, so the reading is
recorded once by `tools/probe_reference_scan.py` against a single-use instance and compared against
Atrium's own scan by `tests/library/test_reference_reading.py`, in the default job, with no Jellyfin
anywhere. **The comparison is not an equality**: the two servers disagree over that tree in
~~twenty-six~~ **forty-seven** places, each declared in that module with its reason, and a difference
that is not declared fails **and a declared one that has gone away fails too**. Re-running the probe
is what moves the record; editing the table is what moves what is expected of the comparison, and
doing the second to make the first go away is the one thing it is for preventing.

**Amended 2026-09-05, by 003's T17 — the task that writes the declaration this paragraph describes.**
This document stated the count **twice and disagreed with itself**: forty-seven in [§L2](#l2--semantic)
above and twenty-six here. Forty-seven is the later number — it is [010's D-7](../../specs/010-conformance-harness/spec.md#7-open-questions),
taken 2026-09-02, and it is what [CLAUDE.md](../../CLAUDE.md), [specs/README.md](../../specs/README.md),
010's own specification and 003's plan all carry. The stale number is struck rather than rewritten,
because **an implementer who took twenty-six as the target would declare twenty-one differences too
few and fail the run for the wrong reason**, and the record of that is worth more than a tidy
sentence. The half about a declared difference that has gone away is added in the same edit: it is
010's amended AC-2, and the sentence predates it. **No row of [allowlist.yaml](allowlist.yaml) moves**
— this paragraph describes the *fixture-reading* comparison and not an allowlist entry, so the
three-way pairing with that file and 010 §3.3 sees the same rows before and after. That is 002 T22's
argument, checked before it was relied on.

**And what 003 derived, which is not forty-seven.** 003's own declaration holds **thirty-two** rows,
over the four libraries `internal/libraryfixture` builds; a further eight are predicted over `Films`
and `Tunes`, which are 008's media world and which no run in the default job can build. The
remaining **seven are not derivable from the recorded reading and this project's specifications**,
which is all 003 has — the module holding the original declaration stayed in the source repository.
The count is asserted from the declaration's own length in `internal/app`, so the gap is a stated
finding rather than a row somebody invented to reach a number.

**The reading is a reading of the tree only because the instance is configured for it.** A library
added with nothing but its path fetches metadata from the internet, and over this tree that supplied
nine of the fifty-nine names — `Highlander: Reunion` for an episode of a series that does not exist
— which would have put a third party's database into the comparison, moving without either server
moving. `LibraryOptions.EnableInternetProviders` does not stop it: it is declared, it is stored, it
reads back, and nothing in the reference consults it
`[source: MediaBrowser.Model/Configuration/LibraryOptions.cs:64 @ v10.11.11]`. The per-type fetcher
list does, because it is an allowlist
`[source: MediaBrowser.Controller/BaseItemManager/BaseItemManager.cs:42 @ v10.11.11]`.
`[probe: tools/probe_reference_scan.py, Jellyfin 10.11.11, 2026-09-02]`

**An anchor is never an identifier**, because the two servers derive those differently by design
([§1.4](behaviours.md#14-item-identifiers-are-32-lowercase-hex-characters)). It is a declared
listing case and a row position, resolved against each server immediately before the case runs — or
a JSON Pointer into an earlier case's response, for the things no listing carries, or a literal for
a path parameter that names a format rather than an item. **An anchor over a listing the allowlist
marks `drawn` or `unordered` is refused**: an anchor is only as sound as the ordering it indexes,
and over a listing with none it names an arbitrary row.

**The allowlist** — fields that legitimately differ and are therefore compared by *shape* rather
than by value — is checked in beside the tool, and **every entry declares one of exactly two kinds
of reason**: the behaviours.md section that argues a difference a server *chose*, or one of four
**derivation classes** for a difference neither server chose — `derived-identifier`, `wall-clock`,
`content-hash`, `installation-path`. An entry with neither fails the run
([010 AC-6](../../specs/010-conformance-harness/spec.md#5-acceptance-criteria), refined 2026-09-01).

The list itself is checked in as [allowlist.yaml](allowlist.yaml), and this table is a rendering of
it. `tests/unit/test_allowlist.py` compares both — this one and
[010 §3.3](../../specs/010-conformance-harness/spec.md#33-the-allowlist)'s — against the file row
for row, so the two prose copies cannot drift apart from each other or from what the harness reads.

| Field | Why it may differ | Because |
|---|---|---|
| `Id`, `ItemId`, `Key`, `ServerId`, `ParentId`, `SeriesId`, `SeasonId`, `AlbumId`, `ParentThumbItemId`, `ParentBackdropItemId`, `PlaylistItemId`, `ThumbImageItemId`, `BackdropImageItemId`, `UserId`, `DeviceId` | Derivation differs by design (behaviours §1.4) | `derived-identifier` |
| `DateCreated`, `DateLastMediaAdded`, `LastActivityDate` | Scan wall-clock time | `wall-clock` |
| `Etag`, `ETag`, `ImageTags.*` | Content hashes over differently-derived inputs | `content-hash` |
| `PlaySessionId`, `AccessToken` | Generated once per session and per token, by each server for itself | `derived-identifier` |
| `Path` | Different mount points, and on the by-name rows a different installation's data directory | `installation-path` |
| `LocalAddress` | Deliberate divergence | [behaviours §4.2](behaviours.md#42-localaddress-does-not-get-an-https-override) |
| `TotalRecordCount` on the by-name endpoints, on a request that carries no limit (`by-name-without-limit`) | Deliberate divergence | [behaviours §3.1](behaviours.md#31-totalrecordcount-is-0-on-by-name-endpoints-without-limit--class-b) |
| `X-Response-Time-ms` and `Date`, the response clock | Move on every response | [behaviours §1.9](behaviours.md#19-every-response-carries-x-response-time-ms) |
| `Server` | Deliberate divergence — this server says what it really is | [behaviours §4.1](behaviours.md#41-atrium-identifies-as-jellyfin-on-the-fields-clients-parse) |
| `ChildCount` on a library view | The reference's value is a fresh random integer between 1 and 9 `[source: Emby.Server.Implementations/Dto/DtoService.cs:516-526 @ v10.11.11]` | [behaviours §3.25](behaviours.md#325-childcount-on-a-library-view-is-a-fresh-random-integer--class-b-diverged) |

**`DateLastSaved` was in this table until 2026-09-02 and is not a property of an item body at
all** — it is an `ItemFields` token, and the pinned document's `BaseItemDto` does not carry it
`[spec: BaseItemDto, ItemFields]`. It was withdrawn when the table was written into the file, along
with the identifier row's `…`, which a file cannot hold.

**An entry is scoped to an endpoint and to a path inside the body, never to a bare field name.**
`ChildCount` is why: the reference's number is excused on a library view, while the same property on
a series, a season or a multi-disc album is a real count of the container's children on both servers.
A row keyed on the name alone would excuse the value the L2 tests exist to check.

**A field is not the only unit a difference comes in.** Where the reference's whole answer is a
draw, no field of it is comparable: `/Items/{itemId}/Similar` and any listing ordered at random are
excused **as arrays** — their rows are not value-compared, while their key sets, types, envelope and
row count still are. Four identical `Similar` requests returned 48 distinct items with none in
common. `[probe: tools/probe_similar_ranking.py, Jellyfin 10.11.11, 2026-09-01]` The same
rule applies to these, and the two that depend on what the caller asked for carry a request-case
id — `listing-ordered-at-random` and `listing-ordered-by-a-key-with-ties` — because an entry keyed
on an endpoint and a pointer cannot say *"when the order was drawn"*. The `Similar` array's reason
is
[behaviours §3.23](behaviours.md#323-similar-is-a-random-draw-not-a-ranking--class-b-diverged), and
a listing whose order is drawn or whose ties are engine-resolved is
[behaviours §3.6](behaviours.md#36-ties-are-engine-resolved-and-paging-the-artist-sorts-loses-rows--class-b-diverged).

**An excused array's row count is still compared, and on `Similar` it differs every time.** The
reference answers `limit + 4` rows on a movie seed where Atrium answers exactly `limit`
`[probe: tools/probe_similar_ranking.py, Jellyfin 10.11.11, 2026-09-01]`, which is the divergence
[behaviours §3.24](behaviours.md#324-similar-answers-limit--4-on-a-movie-seed--class-b-diverged)
argues. So every run of that endpoint reports the count, permanently, and that is the intended
answer rather than a gap: the count is the only quantity of a drawn array a run can still check, so
an entry excusing it would leave the endpoint with nothing measured at all, and a report line saying
the known divergence is still exactly the known divergence is what notices the day it stops being
`+ 4`. What the count must not do is stop the rows being walked for their shape — that split is
[010 plan §6.2](../../specs/010-conformance-harness/plan.md).

**Adding to the allowlist is a contract decision**, not a way to make a red test green. It happens
in review, with the reason written in the table.

The differential harness is **opt-in**: it needs a second server, so it is a program run by hand
and never a test, and no CI job runs it. It runs on demand, and after every bump of the pinned
reference version — which is the moment its findings are worth most.

*This paragraph named an `ATRIUM_`-prefixed environment variable as the switch until 2026-09-02,
and no code read it and no test skipped on it.* The mechanism was real — the differential is
opt-in and nothing in the default job touches the network — but the name was a claim about an
implementation that did not exist. The harness reads `JELLYFIN_URL` from `.env`, which is what
`tools/_probe.py` and all 53 probes have used since 002, and the sentence is corrected here rather
than a second name introduced.

## When the reference version moves

Bumping the pinned Jellyfin version is a deliberate act with a fixed procedure:

1. Fetch the new OpenAPI document; run the surface validator. Any path or method that disappeared
   is a breaking change to record.
2. Run the full differential harness against the new server. Every new difference is triaged into
   [behaviours.md](behaviours.md) — replicate, diverge with an argument, or defer.
3. Re-run every probe script under `tools/`, and update the `Last verified` line of every document
   whose claims they support.
4. Only then change the version in [reference-target.md](reference-target.md).

A version bump that skips step 2 has not been done; it has been declared.

**Since 2026-09-02 the four steps are a program**: `tools/bump_reference_version.py` runs them in
this order and stops at the first failure, with every later step reported *not run* rather than
skipped quietly. It adds no mechanism — each step is a tool that already existed — and it removes
the shortcut, which is the whole point. Two things it does that this prose could not: it decides
the paragraph below by **measuring** the running reference's version rather than by taking
somebody's word, and it validates step 1 against a **copy** of `surface.yaml` with the pin moved,
because the validator's own version gate would otherwise fail step 1 on every bump before it could
report a disappeared path.

### The two rows move separately

[reference-target §1](reference-target.md#1-the-pinned-version) pins two things, and the procedure
above is written for moving the **behavioural** one — a *new server*, whose differences step 2
exists to find.

**When only the contract row moves — same server, a different document of it — step 2 has no
input.** Nothing behavioural changed, so there is no new difference for a differential harness to
triage; running it would compare a server against itself. The steps that do apply are 1, 3 and 4,
and they apply in full: the surface validator against the new document, every claim the repository
draws from a document re-measured against it, and only then the version changed.

This is not a licence to skip step 2 by declaring a move "document-only". The test is whether the
running reference server changed. If it did, step 2 is mandatory and no argument substitutes for
it. The one move made under this paragraph — `10.11.10` → `10.11.11` on 2026-09-01 — was made
against a server that had already been `10.11.11` for the whole project, with every one of the
repository's 515 provenance tags naming it and none naming `10.11.10`.
