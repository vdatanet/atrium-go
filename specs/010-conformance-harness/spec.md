---
feature: 010-conformance-harness
title: Conformance harness
status: Accepted
created: 2026-08-26
updated: 2026-09-02
amended: 2026-09-01 at the measurement gate — the spec was written before the thing it compares existed, and four probes moved it. §7's four open questions are answered and closed. **OQ-1's premise survives and its remedy does not:** identifiers cannot join two servers, but neither can a path — the reference sends none on a default list row at all, so a run that joins on one is comparing requests no client sends; asking for it still leaves a virtual season, a remote channel and every by-name row with nothing to join on, and the paths those rows do carry name the reference installation's own data directory. **OQ-3 is answered yes for reads and no for the feature:** every read of the surface but three answers byte-identical bodies on repeated identical requests, and the only header values that move are the response time and the clock — but a recording answers only the requests it recorded, which is the class of defect L3 exists to find, so a recorded session is a regression net and never the gate. **OQ-4's two non-deterministic endpoints are three, and the third is not an endpoint:** `/UserViews` answers a fresh random `ChildCount` between 1 and 9 on every request, because the reference declines to count a top-level folder and substitutes a number so clients will not think it empty. **The allowlist §3.3 describes cannot express any of them:** it is a list of fields compared by shape, and what these need is a whole array excused, which is a second mechanism and now a stated one. Plus the three things the gate found that nobody asked about: `/Items/{itemId}/Similar` is not a ranking at all but a fresh draw whose successive answers share **nothing**, and on a **movie** seed it answers `limit + 4` rows where the pinned document calls `limit` a maximum — both new divergences against an implemented feature, and both owner decisions; and a differential that authenticates once measures half the table — **12 of 23 reads of the surface answer differently to a restricted non-administrator**, two of them not as refusals but as shorter lists. §3.9 and §3.10 are new sections, §5 gains five criteria, and the fixture-differential AC-2 is recorded as unreachable from the only reference this project can reach. **Amended again the same day by the three decisions that gate left to their owners.** G-1 and G-2 are decided and recorded as divergences, with their arguments in behaviours §3.23 and §3.24 and their statements in 005 §3.7 and AC-12; and **OQ-5 is answered, so AC-2 is no longer blocked** — a run that needs the fixture on both servers stands up a single-use reference instance of the pinned version over it and destroys it, including on failure, which also takes the writing measurements off an operator's server where 009's runs left 28 playlists behind. §3.1's blocked wording, §3.5's cleanup claim, AC-2, §4's artefact table and §7 are rewritten accordingly, and the clause above recording AC-2 as unreachable is what they supersede. **Amended a third time on 2026-09-01, deliberately, by the decision the implementation plan reserved as D-3.** **AC-6 was unsatisfiable by the allowlist this document ships**: it failed the run on any entry with no behaviours.md reference, and three of the eight field rows and one of the three array rows named one. Seven named a *reason* instead — a scan's wall-clock time, a content hash, a different mount point — and those are not divergences anybody argued for; they are facts about two separate installations. **AC-6 now takes either**: a behaviours section where the difference was chosen, or one of four declared derivation classes — `derived-identifier`, `wall-clock`, `content-hash`, `installation-path` — where it was not, with a fifth class reviewable and never a substitute for an argument somebody owes. **The one row that was neither was written rather than reclassified:** the reference's random `ChildCount` on a library view had no behaviours entry at all, and it is now [behaviours §3.25](../../docs/compatibility/behaviours.md) — class B, diverged, because a number redrawn on every request is one no client can have compensated for. §3.3's two tables gain the `because` column, every row of both now carries one, and §3.3 records what the same measurement decided about the mechanism: **an entry is scoped to an endpoint and a path, never to a field name**, since `ChildCount` elsewhere is a real subtree aggregate on both servers and excusing the name would excuse the value the lower levels check. **Amended a fourth time on 2026-09-02, by the decision the task list reserved as D-6.** **§3.10's sixteen named comparisons are twenty.** Reviewing the six inherited lists against the compatibility documents found four readings with a written home and no owner, each askable only against a library this project may change, and each therefore unblocked by the same single-use instance OQ-5 settled: **behaviours §5.2** — a container that has lost every file, which carries the **only surviving `⚠️ UNVERIFIED`** in the compatibility documents and whose own text names a disposable library as the remedy; **behaviours §5.6** — a replaced poster a default rescan does not notice; **005 §7 OQ-7** — whether Next Up excludes a pristine specials season, which no measured library could ask; and **007's paused-session ticker freeze**, cited from the reference's source and never seen on the wire. Three need the library changed between two scans and the fourth needs ten minutes of deliberate silence, so none of them is a sweep finding and all four are what §3.10 is for. §3.10 gains the four rows with their provenance and the reason each needs the instance, **AC-16 now names the count — twenty, where it counted sixteen** — and §7 records D-6 as taken. **Behaviours §5.2's `⚠️ UNVERIFIED` marker stays**: `behaviours.md` is not a specification and may carry one, and a §3.10 row is an owner and a method rather than the reading that discharges it. **Amended a fifth time on 2026-09-02, by the task that wrote §3.3's two tables into the file they describe (T3).** Four rows did not survive the writing. `DateLastSaved` is **not a property of an item body at all** — it is an `ItemFields` token, and the pinned document's `BaseItemDto` does not carry it `[spec: BaseItemDto, ItemFields]` — so a row of the wall-clock table excused a field neither server can send, and it is withdrawn. The identifier row's `…` is **spelled out**, because a file cannot hold an ellipsis, and spelling it out named five identifiers the table had never listed — `Key` and `ItemId` inside `UserData`, `ParentThumbItemId`, `ParentBackdropItemId` and `PlaylistItemId`. The **`Server` header gains a row**: it is a divergence behaviours §4.1 already argues and the allowlist excused nowhere, and it differs on every response of every case. And **two rows are conditioned on the request rather than on the route** — `TotalRecordCount` without a limit, and a listing ordered at random — which an entry keyed on an endpoint and a pointer cannot state, so both now name a **request-case id**, which is a seventh field in the file and a narrowing one: an entry with no condition says `*`. §3.3's two tables and [conformance.md](../../docs/compatibility/conformance.md#l3--differential)'s rendering of the same list carry all four changes, and `tests/unit/test_allowlist.py` now compares both prose copies against the file row for row so the three cannot drift again. **No acceptance criterion changes**: AC-6 still takes a behaviours section or one of the same four derivation classes.
depends_on: [001, 002, 003, 004, 005, 006, 007, 008, 009, 011]
---

# 010 — Conformance harness

> **This document describes WHAT and WHY only.** No technology names, no storage decisions.
>
> This feature has no HTTP surface. Its "clients" are the project's own contributors, and its
> observable outputs are reports and CI results. It is specified rather than left as scaffolding
> because Principle VIII makes it the thing that decides when any other feature is done.

## 1. Purpose

Prove that Atrium behaves like Jellyfin, and — more valuable — **find the places where it does
not, that nobody thought to ask about.**

Levels L0 through L2 test Atrium against what this project *believes* about Jellyfin. They cannot
catch a belief that is wrong. **L3, the differential layer, is the only thing in the project that
can**, and delivering it is what this feature is for.

**Unlocked:** the ability to say "compatible" and mean something measurable.

**And one thing this feature must not pretend.** A differential is a sweep, and a sweep finds only
what a request under comparison exposes. Six implemented features have each handed this one a list
of differences a sweep will **not** raise — because they need a caller the run does not have, a
library the reference cannot be given, a library changed between two scans, a deliberate wait, or a
comparison of something that is not in a body. Those
are §3.10, they are named rather than discovered, and a harness that reports a clean run without
them is reporting the absence of the questions it did not ask.

## 2. Scope

**In scope**

- The differential harness: same request to both servers, structural comparison, triaged report.
- The **identities** a run authenticates as, and what the second one buys (§3.9).
- The **named comparisons** (§3.10): the differences the sweep cannot raise, run deliberately.
- The allowlist of legitimately-varying fields *and arrays*, and the discipline around changing it.
- The fixture library as a build artifact reproducible on both servers.
- The probe-script convention, and discharging the prior-measurement debts.
- The ignored-parameter report from 005 §3.3.
- The version-bump procedure of `conformance.md`, as an executable process.
- CI wiring: what runs always, what runs on demand.

**Out of scope**

- L0 and L1 machinery, delivered by 001 — this feature consumes them.
- Performance benchmarking. Correctness only.
- Testing Jellyfin. When a difference is Jellyfin's defect, it is recorded, not fixed here.
- **Deciding what Atrium does about a difference this feature finds.** The harness triages
  (§3.4); the answer belongs to the feature that owns the endpoint, through
  [behaviours §3.0](../../docs/compatibility/behaviours.md#30-how-the-decision-is-made). The gate
  that accepted this document found two such differences and left both to 005 (§7).

## 3. Behaviour

### 3.1 The fixture library

One directory tree, checked in as **metadata only**, from which both servers build a library.

| Content | Form |
|---|---|
| Directory structure and filenames | Checked in |
| `.nfo` sidecars | Checked in |
| Artwork | Tiny generated images, built at build time |
| Media files | Synthetic — seconds of colour bars and a tone, muxed at build time |

**No copyrighted media, ever.** Not a clip, not a sample, not "just for testing".

Building it is deterministic: the same generator inputs produce byte-identical files, so both
servers probe the same thing and a difference in output is a difference in the server.

**Two of these worlds already exist**, built by the features that needed them rather than by this
one — a paths-and-filler world for scanning and a real-media world for delivery, described in
[conformance.md](../../docs/compatibility/conformance.md#l2--semantic). What this feature adds is
not a third world. It is the part neither of them has: **the same tree on the other server.**

The fixture deliberately contains the cases that break naive implementations — the ones enumerated
across 003 §5, 005 §5 and 009 §5: multi-part films, `S01E02-E03`, specials, a series named `24`,
multi-disc albums, compilations, tags that contradict paths, non-ASCII names, names differing only
by case, and a playlist with duplicate entries. Five more are owed by features that landed after
this document was written, each because the difference it exposes is unreachable without it: a
**multi-part film** (008 — one media source per part here, one source and a part count there), a
film with a **subtitle file beside it** and one with an **image subtitle track** (011), a subtitle
file in a **legacy encoding** (behaviours §5.11), and a **playlist holding items from two
libraries** (§3.10).

**AC-2 needs a reference this project controls, so a run stands one up.** *(Decided 2026-09-01.
This paragraph recorded AC-2 as blocked; it is not.)* The only Jellyfin this repository could
measure when the document was accepted is an operator's own server, holding an operator's own
library, on another machine, and both halves of "point both servers at the same built fixture" are
refused by that: the fixture tree is not on that machine, and adding a library to it would be
writing to data this project does not own.

**So a run does not borrow a reference. It creates one, uses it, and destroys it.** A run that
needs the fixture on both sides brings up a reference server of the pinned version, gives it the
fixture tree of this section as its **only** library, waits for its scan to finish, issues the
comparison, and then destroys the instance and everything it wrote — including on failure, and
including whatever the run itself created inside it. The instance is **single use**: nothing
carries from one run to the next, which is what makes the comparison reproducible and what keeps
the fixture the only library either server has ever seen. The requirement is what this section
states — an instance of the pinned version, configurable, disposable, and holding nothing but the
fixture. *How* one is stood up is 010's plan, and this document does not decide it.

**The second reason is that it takes the writing measurements off an operator's server**, and it
is not the smaller one. §3.5 requires a probe that writes to remove what it made, including on
failure; on 2026-09-01 the operator's server still held **28 playlists left behind by 009's probe
runs**, each carrying the name those probes create them under. Against a disposable instance a
leftover is harmless, because the instance does not survive the run either way — which is the
difference between a cleanup that must be perfect and one that only has to be tidy.

Everything in §3.9, §3.10 and §7 was measured **before** such an instance existed, against the real
library, which is why each of those is a *named* measurement and not a sweep.

### 3.2 The differential run

```
for each identity in the identities of §3.9:
    for each endpoint in surface.yaml:
        for each request case defined for it:
            response_a = atrium(request)
            response_b = jellyfin(request)
            compare(response_a, response_b)
```

**Comparison is structural, in three passes**, because the three failures are not equally serious:

| Pass | Finds | Severity |
|---|---|---|
| **Key sets** | A field one server sends and the other does not | Highest — a missing field is invisible to L1 and L2, which only check what we knew to check |
| **Types** | Same key, different JSON type | High — `"5"` versus `5` breaks decoders |
| **Values** | Same key and type, different content | Triaged; many are legitimate |

**Key-set differences are the entire point of the exercise.** They are the only class of defect
that this project structurally cannot find any other way, and the report ranks them first.

Headers are compared too, on the delivery routes, where `Content-Length`, `Accept-Ranges`,
`Content-Range` and `Content-Type` are the contract.

**Three things the passes above cannot do on their own, all measured at this document's gate.**

1. **A row has to be found before it can be compared, and nothing on the wire identifies it.**
   Identifiers cannot: the two servers derive them differently and matching the reference's bytes
   is explicitly not a goal ([behaviours §1.4](../../docs/compatibility/behaviours.md)). The path
   is the only value two servers scanning one tree could agree on, and it is **absent from every
   default list row** — 0 of 1000 — so a run that joins on it has changed the request it is
   comparing. Asked for by name it still covers only what is file-backed: a virtual season and a
   remote channel carry none, and every by-name row carries one naming *the reference
   installation's own data directory*, which no fixture reproduces. `(Type, Name)` is not a key
   either — 976 distinct of 1000 rows on a real library.
   `[probe: tools/probe_differential_join.py, Jellyfin 10.11.11, 2026-09-01]`
   **So arrays are compared as ordered sequences and rows by position**, which makes the ordering
   part of the contract under test rather than a convenience.
2. **Position is only a comparison where the ordering is total, and the reference's is not.** Its
   ties are engine-resolved and its artist sorts lose and duplicate rows across pages, which is a
   recorded divergence ([behaviours §3.6](../../docs/compatibility/behaviours.md)). A positional
   comparison of any response ordered by a key with ties reports differences that are Atrium doing
   what §3.6 says it does. Those responses are compared **as multisets of rows**, and the ordering
   difference is the allowlist row rather than the finding.
3. **Three read shapes are draws rather than readings**, and no comparison of their rows means
   anything (§3.3, and OQ-4 in §7).

### 3.3 The allowlist

Fields compared by **shape** rather than by value, each with a written reason.

| Field | Why it may differ | Because |
|---|---|---|
| `Id`, `ItemId`, `Key`, `ServerId`, `ParentId`, `SeriesId`, `SeasonId`, `AlbumId`, `ParentThumbItemId`, `ParentBackdropItemId`, `PlaylistItemId`, `ThumbImageItemId`, `BackdropImageItemId`, `UserId`, `DeviceId` | Derivation differs by design (behaviours §1.4) | `derived-identifier` |
| `DateCreated`, `DateLastMediaAdded`, `LastActivityDate` | Scan wall-clock time | `wall-clock` |
| `Etag`, `ETag`, `ImageTags.*` | Content hashes over differently-derived inputs | `content-hash` |
| `PlaySessionId`, `AccessToken` | Generated once per session and per token, by each server for itself | `derived-identifier` |
| `Path` | Different mount points, and on the by-name rows a different installation's data directory (§3.2) | `installation-path` |
| `LocalAddress` | Deliberate divergence | behaviours §4.2 |
| `TotalRecordCount` on the by-name endpoints, on a request that carries no limit (`by-name-without-limit`) | Deliberate divergence | behaviours §3.1 |
| `X-Response-Time-ms` and `Date`, the response clock | Move on every response, measured on 19 of 19 read cases (§7 OQ-3) | behaviours §1.9 |
| `Server` | Deliberate divergence — this server says what it really is | behaviours §4.1 |
| `ChildCount` on a library view | **The reference's value is a fresh random integer** (§7 OQ-4) | behaviours §3.25 |

**A field is not the only unit a difference comes in**, and the gate found three that this table
cannot express. Where the reference's *whole answer* is a draw, no field of it is comparable and
excusing them one by one would excuse the response. Those get a **second kind of entry — an
excused array** — which states the endpoint, the request shape that triggers it, and what is still
compared when the rows are not:

| Array | Why it may differ | What is still compared | Because |
|---|---|---|---|
| The rows of `/Items/{itemId}/Similar` | A fresh draw per request; four identical requests shared **no** item (§7 OQ-4) | Key sets and types of each row, the envelope's own properties, and the row **count** | behaviours §3.23 |
| The rows of any listing ordered at random (`listing-ordered-at-random`) | The same, by the caller's own request | The same | behaviours §3.6 |
| The rows of a listing ordered by a key with ties (`listing-ordered-by-a-key-with-ties`) | The reference's ordering is not total | Everything, as a multiset rather than a sequence | behaviours §3.6 |

**Every entry says why, and there are exactly two kinds of why.** *(Decided 2026-09-01, with the
`because` column above.)* Where the difference is one a server **chose** — this project diverging,
or the reference doing something it decided to do — the entry names the behaviours.md section that
carries the argument, and adding the entry means having written that argument. Where the difference
is one **neither server chose** — it follows from the two being separate installations of separate
software, scanned at different moments, over different mount points — the entry names one of four
**derivation classes**:

| Class | What it says |
|---|---|
| `derived-identifier` | The two servers derive this identifier differently by design, and neither derivation is wrong (behaviours §1.4) |
| `wall-clock` | The value is the moment the scan happened, and the two scans happened at different moments |
| `content-hash` | The value is a hash over inputs that are themselves derived differently |
| `installation-path` | The value names where this installation keeps its files |

**A fifth class is not added without review**, and neither is a class used where a behaviours entry
is what is actually owed. The distinction is the whole discipline: a derivation class is a fact
about how two installations differ, and it can never be the excuse for a value one of the two
decided. Where the honest answer was a behaviours entry that did not exist, the entry was written —
the reference's random `ChildCount` is now behaviours §3.25, which is what closed the last row of
this table.

**Adding an entry of either kind is a contract decision, not a way to make a red run green.** It
happens in review, the reason goes in the table, and an entry justified by "we do it differently"
with neither a behaviours.md section nor a declared derivation class behind it is rejected (AC-6).

**And an entry is scoped to an endpoint and to a path within the body, never to a field name.**
`ChildCount` is the case that decides it: the reference's number is excused on a library view, and
the same property elsewhere is a real count of a container's children on both servers — so an entry
that excused the *name* would excuse, on every container, exactly the value the lower levels exist to
check (behaviours §3.25).

**Amended 2026-09-02, when the two tables above were written into the file they describe** and
four of their rows did not survive the writing. *(010 T3.)* **`DateLastSaved` is gone**: it is an
`ItemFields` token and not a property of an item body — the pinned document's `BaseItemDto` has
153 properties and that is not one of them `[spec: BaseItemDto, ItemFields]` — so the row excused a
field no response on either server can carry. **The identifier row's `…` is spelled out**, because
an ellipsis is not an entry, and spelling it out named five identifiers the table had never listed:
`Key` and `ItemId` inside `UserData`, `ParentThumbItemId` and `ParentBackdropItemId` on the rows
that inherit an image, and `PlaylistItemId`, which 009's gate measured to be the item's own `Id`.
**The `Server` header is a row now**, where it was a divergence argued in behaviours §4.1 and
excused nowhere: `Atrium/<version>` against the reference's `Kestrel`
`[probe: tools/probe_routing.py, Jellyfin 10.11.11, 2026-08-28]` differs on **every** response, so
until the entry existed every case in the sweep reported the same header. And **two rows are
conditioned on the request rather than on the route** — `TotalRecordCount` names the by-name call
that carries no limit, and the random listing names the caller's own `SortBy` — which is a
condition an entry keyed on an endpoint and a pointer cannot state. They carry a **request-case
id** instead, named in the table above, and an entry with no condition says `*`.

The allowlist is also **a metric**: it should shrink over time as derivations converge, and a run
that grows it is worth a second look. The excused arrays are the exception that proves it — none of
the three can ever be removed, because none of them is Atrium's to converge with.

### 3.4 The report

One document per run, and it is the deliverable — not a pass/fail line.

```
Differential run — Atrium <sha> vs Jellyfin 10.11.11 — <date>

  identities             <n>
  endpoints compared     59
  request cases          <n>
  identical              <n>
  allowlisted            <n>
  DIFFERENCES            <n>

  Missing keys        (n)    <-- read these first
  Extra keys          (n)
  Type mismatches     (n)
  Value mismatches    (n)

  Named comparisons   (n of the §3.10 list run, n outstanding)
```

Each difference carries the endpoint, the request case, **the identity that sent it**, the JSON
path, both values, and — where it matches a known entry — a link to the behaviours.md section that
explains it.

**Every difference is triaged into one of four outcomes**, and none of them is "ignore":

| Outcome | Meaning |
|---|---|
| **Fix** | Atrium is wrong. A defect, against the owning feature |
| **Replicate** | Jellyfin's behaviour is odd; we match it. A behaviours.md entry |
| **Diverge** | We deliberately differ. A behaviours.md entry with the argument, and an allowlist row |
| **Defer** | Out of v1 scope. Recorded with the feature that will resolve it |

An untriaged difference blocks the run from being called clean. **So does an unrun named
comparison**: the §3.10 list is part of the report's coverage line, not a footnote to it.

### 3.5 Probe scripts, and the debt

Every probe under `tools/` follows one convention:

1. Takes a server URL and credentials.
2. Makes the minimum requests needed to answer **one** question.
3. Prints its finding **and its own citation**, in the form the documentation uses:
   `[probe: tools/probe_x.py, Jellyfin 10.11.11, 2026-08-26]`.
4. Exits non-zero if the finding contradicts what the documentation currently claims.

That last point turns the probes into a regression suite for the project's *beliefs*, not just its
code. When a server upgrade changes a behaviour, the probe says so instead of the documentation
quietly becoming false.

**A probe that writes creates what it needs and removes it, including on failure**, and never
touches anything the operator owns. Three of them now build a throwaway account, because the
question they answer is invisible from an administrator's seat (§3.9).

**That is the requirement, and on 2026-09-01 it was checked against the server and did not hold**:
009's runs had left 28 playlists behind, where `tools/README.md` records of every writing probe
that it deletes what it made including on failure. The requirement stands — a probe that leaks is a
probe with a defect — but it is no longer the only thing standing between a run and an operator's
data, because the fixture runs of §3.1 write to an instance that is destroyed afterwards regardless.

**This feature discharges the prior-measurement debts** registered in
[reference-target.md](../../docs/compatibility/reference-target.md): each becomes a probe script,
and its citation changes from `prior-probe` to `probe`. A claim that fails to reproduce is not
deleted — it becomes a behaviours.md entry recording that the behaviour *changed*, with both dates.

### 3.6 The ignored-parameter report

005 §3.3 accepts a bounded delta: Tier 3 query parameters are ignored rather than rejected, and
counted.

This feature turns that counter into a report — parameter, endpoint, count, and the client that
sent it — so the accepted delta has a closing mechanism instead of being a permanent excuse. Any
parameter that appears gets promoted to Tier 2 or explicitly declined in writing.

### 3.7 What runs when

| Job | Runs | Requires |
|---|---|---|
| Surface validation | Every change | Nothing beyond the repository |
| L0 route registration | Every change | Atrium |
| L1 sweeps and golden responses | Every change | Atrium |
| L2 fixture tests | Every change | Atrium and a built fixture |
| **L3 differential** | On demand, and on every reference-version bump | A real Jellyfin |
| **The §3.10 named comparisons** | On demand, and on every reference-version bump | A real Jellyfin, and for most of them a second identity |
| Probe scripts | On demand, and on every bump | A real Jellyfin |
| Live-provider tests (004) | On demand only | Network and credentials |

**L3 is opt-in and never gates the default CI job.** It needs a second server, so making it
mandatory would make the build depend on infrastructure a contributor may not have. It is *the*
gate on a version bump, where it matters most.

**No test that runs by default touches the network.** Principle VII.

### 3.8 The version-bump procedure

[conformance.md](../../docs/compatibility/conformance.md#when-the-reference-version-moves) defines
it; this feature makes it executable as a single command that runs the steps in order and refuses
to continue past a failure:

1. Fetch the new OpenAPI document; validate the surface. Disappeared paths are breaking changes.
2. Run the full differential, and the §3.10 named comparisons. Triage every new difference into
   behaviours.md.
3. Re-run every probe. Update the `Last verified` line of every document they support.
4. Only then change the pinned version.

**A bump that skips step 2 has not been done, it has been declared.** The command enforces the
order so that the shortcut is not available.

### 3.9 The identities a run authenticates as

**A run that authenticates once measures one row of a two-row table, and its report says nothing
about the other.** Every probe written before this document was accepted authenticated as an
administrator, and an administrator lacks no permission — so no measurement in this repository had
ever been taken from a seat that could be refused something.

Measured, on the same twenty-three reads of the surface issued twice — once as the administrator,
once as a throwaway non-administrator restricted to one library — **twelve answer differently**:

| The answer changes as | Endpoints |
|---|---|
| **A refusal** | an item in a library the reader cannot open; a playlist read that does not name its owner |
| **A narrower object** | the caller's own user record — twelve properties to an administrator, ten to everybody else |
| **A shorter list** | the views; the item listing and its total; resume; next up; the artists; the genres; the music genres; the years; the sessions |

`[probe: tools/probe_restricted_surface.py, Jellyfin 10.11.11, 2026-09-01]`

**The second row is the dangerous one and the third is the expensive one.** A refusal is a status
code and any comparison sees it. A *shorter list* is a `200` that differs only in how many rows it
holds, which is exactly the shape §3.10's two playlist comparisons take — and exactly what a run
looking for missing keys will pass over.

So: **a run states its identities, and a run with one identity is reported as covering one.** The
minimum is two — an administrator and a restricted non-administrator whose library access is
narrower than the library — and the restricted seat is **created and destroyed by the run**, never
borrowed from the operator.

### 3.10 What a differential cannot find on its own

Six implemented features each ended with a list of what it leaves the next ones, and 010 collects
most of them, and so do two compatibility entries and two open questions those lists point at.
**The twenty rows below are the ones a sweep will not raise**, each for one of five
reasons — no caller, no library, no second scan, no elapsed time, nothing in a body — and each is
run deliberately with its own written comparison:

| The difference | Why the sweep misses it | What the named comparison is |
|---|---|---|
| A playlist read that **names its own reader** (behaviours §3.16) | It needs a caller the run does not have | Issue the read naming the owner **as a restricted non-administrator**: entries there, a refusal here |
| **Entries a reader cannot reach** (behaviours §3.17) | Over a stock library the two servers *agree*: what the reference hides is hidden by a rating check, never by library access | A reader restricted to one library, a playlist holding items from two. **The row count is the whole signal** |
| The **de-duplication that misses** (behaviours §3.18) | The reference disagrees with *itself* — 6 of 8 identical requests behaved differently | Run the add twice; a disagreement is the entry, not a flake to retry |
| A progressive re-encode's **missing self-description** (behaviours §3.3) | §6 declines to byte-compare produced media | Parse the first frames of both servers' progressive output and compare the header frame's presence, not the bytes after it |
| **Burn-in** (behaviours §5, subtitle row) | The same: the difference is in the pixels | Negotiate a track that resolves to an encode on both, produce a few seconds, look for the cues in the frames |
| The manifest's announced **track name** (011 §6) | It differs by the reference host's culture, not by the server | Compare the announced entries with the name masked, and the name against the invariant form |
| A subtitle playlist's **decimal point** (behaviours §3.12) | A parser normalises it away | Byte-compare the two playlists |
| The **delivery-time policy refusal** (behaviours §2.21) | It needs an account with a denied playback permission | The §3.9 restricted seat, with that permission denied: bytes there, a refusal here |
| An image track's `400` arriving **without twenty seconds of waiting** (011) | Nothing in a body carries elapsed time | Measure the latency of both, not the payload |
| A **multi-part film**'s media sources (008) | No reachable library has one | Needs the fixture of §3.1 |
| A media source stating **no runtime**, and a zero-length cue's millisecond (011) | No reference library can be put into either state from outside | Reported as a **miss** by its own probe on every run, rather than inferred |
| A subtitle file in a **legacy encoding** (behaviours §5.11) | An all-UTF-8 library never raises it | Needs the fixture of §3.1 |
| **EXIF orientation on resize** (006) | No remote request reaches the edge | Needs a planted file in a controlled library |
| An **empty library**'s played state (behaviours §5.7) | The only shape where the question is askable | A server with an empty library — a differential's job rather than a probe's |
| A container that has **lost every file** (behaviours §5.2) | The library has to change *underneath a rescan*, which no read of a stock server performs — and the entry carries the only surviving `⚠️ UNVERIFIED` in the compatibility documents, unmeasured because emptying a directory out of somebody else's library is not a measurement anybody may take | Scan the fixture on both, delete one series' episodes, rescan both, and compare whether the container is still offered. **Needs the instance of §3.1**: a scan is a write, and that entry's own text names the remedy — *a disposable library on a server somebody owns* |
| A **replaced poster** a default rescan does not notice (behaviours §5.6) | The difference lies between two scans rather than inside one answer, so no single request raises it | Scan, replace the artwork beside an untouched film, rescan at default depth on both, and compare the image tag and the bytes it identifies. **Needs the instance of §3.1**: the entry is unmeasured precisely because deciding it means writing into a library and rescanning it |
| Whether Next Up excludes a **pristine specials season** (005 §7 OQ-7) | No reachable library holds a series whose only unplayed episodes are season 0's; the one measured on 2026-08-28 had none, and the probe says so in its own output `[probe: tools/probe_next_up.py, Jellyfin 10.11.11, 2026-08-28]` | Build that series into the fixture, read the route on both, compare. **Needs the instance of §3.1**: the library has to be made rather than found |
| The **paused-session ticker freeze** (007's list) | Nothing in a body carries ten minutes of silence, and the claim is a source reading and not a measurement `[source: MediaBrowser.Controller/Session/SessionInfo.cs:23, 373-451 @ v10.11.11]` | Report a paused session to both, stay silent past the reap threshold, and compare the position each one commits. **Needs the instance of §3.1**: it is a write held open for ten minutes, which is the one thing an operator's server must not be asked for |
| The **`"$"` message** in a body-binding refusal (behaviours §1.11) | Nothing; it *will* be found — on the first malformed body | Listed so it is recognised rather than triaged twice |
| A **body with no content type** on five routes (009) | Measured on one of the five | The other four, one request each |

**The last two rows are here to be recognised, not discovered.** The rest are the feature's real
inheritance: a harness that reports a clean run without them has proved that the questions it asked
have the same answers, which is a smaller claim than it sounds.

**Four of the twenty rows were added on 2026-09-02, by the decision the task list reserved as D-6**
(§7). They were found where the six inherited lists meet the compatibility documents: behaviours
§5.2 and §5.6, 005's OQ-7 and 007's paused-session ticker are each written down somewhere, each
askable only against a library this project may change, and each was outside this table until the
single-use instance of §3.1 existed to ask them. **They are named comparisons and not sweep
findings for the reason every other row here is one**: three of them need the library changed
between two scans and the fourth needs a deliberate silence, and no request compares either. The
alternative — carrying them beside the table as outstanding readings excluded from AC-16 — would
have let a run report *"sixteen of sixteen"* while four questions with a written home went on being
nobody's.

**Behaviours §5.2 keeps its `⚠️ UNVERIFIED` marker.** A row here is an owner and a method, not a
measurement: the reading that would discharge the marker does not exist until the comparison has
run. What changes on 2026-09-02 is that it is owned, and by what.

## 4. Data the feature owns

| Artefact | Where | Lifetime |
|---|---|---|
| Fixture library sources | Repository | Permanent |
| Built fixture media | Build output | Disposable, regenerable |
| Golden responses | Repository | Changed only by review |
| The allowlist, of both kinds | Repository | Changed only by review |
| The named-comparison list | Repository, and this document's §3.10 | Changed only by review |
| Differential reports | Git-ignored output | Per run |
| Ignored-parameter reports | Git-ignored output | Per run |
| Throwaway accounts a run creates | The reference server, briefly | Destroyed by the run that made them, including on failure |
| The reference instance a fixture run stands up | Alongside the run | Single use: destroyed by the run that made it, including on failure (§3.1) |

## 5. Acceptance criteria

1. The fixture library builds deterministically: two builds produce byte-identical media files.
2. Both servers, pointed at the same built fixture, produce libraries with the same item count and
   the same structure. *(**Unblocked 2026-09-01**: the run stands up a single-use reference
   instance of the pinned version over the fixture and destroys it — §3.1. This criterion was
   recorded as blocked when the spec was accepted, on the reading that the only reachable reference
   was an operator's own server.)*
3. The differential covers every endpoint in `surface.yaml`, with at least one request case each,
   and reports its coverage.
4. A deliberately introduced defect — a renamed field, a changed type, an omitted field — is caught,
   and classified into the right pass.
5. The report ranks missing keys first.
6. **Every allowlist entry declares why the difference is excused, and an entry that does not fails
   the run.** An entry that excuses a **divergence** — something one of the two servers chose —
   names the behaviours.md section that carries the argument. An entry that excuses a
   **derivation** — something neither server chose, which follows from the two being separate
   installations — names one of four declared classes: `derived-identifier`, `wall-clock`,
   `content-hash`, `installation-path`. An entry naming neither, or naming a fifth class, fails the
   run. *(Refined 2026-09-01 — see the amendment note.)*
7. Every probe prints a citation in the documented form and exits non-zero when its finding
   contradicts the documentation.
8. A probe whose finding contradicts the documentation produces a message naming the document and
   section to update.
9. Every prior-measurement debt in reference-target.md has a probe script, or a recorded reason it
   cannot have one.
10. The ignored-parameter report lists parameter, endpoint, count and client.
11. The default CI job passes with no Jellyfin available and no network access.
12. The version-bump command refuses to advance past a failed step.
13. No fixture file is a copyrighted work.
14. **A run states the identities it authenticated as, and its coverage line is per identity.** A
    single-identity run is reported as covering one identity, not as covering the surface.
15. **The restricted identity is created and destroyed by the run**, and a run that finds one
    already present refuses to start rather than reusing it.
16. **Every one of §3.10's twenty named comparisons is either run or reported outstanding**, by
    name, in the same report. An outstanding one blocks the run from being called clean (§3.4).
    *(The count moved from sixteen to twenty on 2026-09-02 — D-6, §7.)*
17. **An excused array is excused as an array**: its rows are not value-compared, and its key sets,
    types, envelope and row count still are.
18. **A response whose ordering is not total is compared as a multiset**, and a difference in row
    order alone is not reported as a difference.

## 6. Conformance

This feature is the thing that proves conformance, so it is proven by **mutation**: the harness is
correct if it catches defects that are injected on purpose.

| Property | How it is proven |
|---|---|
| Catches missing keys | Inject a removed field; assert it is reported in the key-set pass |
| Catches type changes | Inject an integer-as-string; assert the type pass |
| Catches value changes | Inject a changed title; assert the value pass |
| Does not cry wolf | Run against itself; assert zero differences outside the allowlist |
| Allowlist discipline | Add an unbacked entry; assert the run fails (AC-6) |
| Probe self-checking | Contradict a probe's expectation; assert non-zero exit (AC-7) |
| Excuses an array without excusing its shape | Remove a key from a row of an excused array; assert it is still reported (AC-17) |
| Reports what it did not ask | Run with one identity and an unrun named comparison; assert the run is not clean (AC-14, AC-16) |

**"Does not cry wolf" is not a formality.** A harness with false positives gets ignored within a
week, and an ignored harness is worse than none — it provides the feeling of coverage without the
coverage.

**It is also the reason the two mutation rows above were added.** The three excused arrays and the
random `ChildCount` are false positives on every single run, on endpoints a client uses constantly;
a harness that reports them is one nobody reads by the second week.

## 7. Open questions

All four questions this document was written with are answered. None survived unchanged.

### Resolved

| # | Question | Answer |
|---|---|---|
| OQ-1 | How to make both servers derive the same library from one fixture, given different identifier derivations | **The premise holds and the remedy does not.** Comparison by path was the proposed way out, and the path is not on the wire: 0 of 1000 default list rows carry one, so a run that joins on it is comparing a request no client sends. Asked for by name it covers the file-backed items only — a virtual season and a remote channel carry none — and every by-name row's path names *the reference installation's own data directory*. `(Type, Name)` is not a key either: 976 distinct of 1000. **Rows are therefore compared by position** (§3.2), which promotes the ordering into the contract and pulls in behaviours §3.6. `[probe: tools/probe_differential_join.py, Jellyfin 10.11.11, 2026-09-01]` |
| OQ-2 | How many request cases per endpoint are enough | **Not a question about the reference, and its measured input is 764.** That is the number of query parameters the pinned document declares across the 59 endpoints of the surface `[spec: the query parameters declared for the 59 operations of surface.yaml]`, so "one case per parameter class" is a three-figure number and "at least one each" (AC-3) is 59. What settles the floor is not a count but a demonstration: the two differences this gate found on an implemented endpoint are both invisible to a bare request — one appears only when `limit` is sent, and only for one seed type. **One case per endpoint is measurably not enough**; the growth rule of AC-3 stands, seeded by the cases the analysed clients actually send. |
| OQ-3 | Can the differential run against a recorded session instead of a live one? | **Yes for the bodies, no for the feature.** 16 of 19 reads of the surface answered byte-identical bodies on three identical requests, and the only header values that moved are the response time and the clock — so a recording is faithful enough to replay. But a recording answers only the requests it recorded, and the defect class L3 exists to find is *the field nobody thought to ask for*; a request nobody thought to record cannot find it. It also cannot carry a write: the playlist routes change what the next read reports, and one of them disagrees with itself (behaviours §3.18). **A recorded session is a regression net for CI, never the gate on a version bump.** `[probe: tools/probe_reference_determinism.py, Jellyfin 10.11.11, 2026-09-01]` |
| OQ-4 | What to do when Jellyfin's own response is non-deterministic (`Random`, `Similar`) | **Compare by shape — and the two endpoints are three.** `/UserViews` answers a fresh random `ChildCount` between 1 and 9 on every request, on every view, with the row order and every other property unchanged: the reference declines to count a top-level folder and substitutes a number so that clients "won't think the folders are empty" `[source: Emby.Server.Implementations/Dto/DtoService.cs:516-526 @ v10.11.11]`, reached because that route asks for every field `[source: Jellyfin.Api/Controllers/UserViewsController.cs:89 @ v10.11.11]`. And the mechanism §3.3 proposed does not fit: excusing a *field* cannot excuse `Similar`, whose whole array is redrawn — four identical requests returned 48 distinct items with **none** in common. §3.3 now carries two kinds of entry. `[probe: tools/probe_reference_determinism.py and tools/probe_similar_ranking.py, Jellyfin 10.11.11, 2026-09-01]` |

### Raised by the gate, owned elsewhere, and decided

Two differences against an **implemented** feature, found by hand while answering OQ-4. Neither was
this feature's to decide (§2): both are 005's, through
[behaviours §3.0](../../docs/compatibility/behaviours.md#30-how-the-decision-is-made). **Both were
decided on 2026-09-01**, and the column below records the decision rather than the debt. They are
two decisions and not one because §3.0.2 forbids deciding once for a whole endpoint.

| # | What was measured | The decision, taken 2026-09-01 |
|---|---|---|
| G-1 | **`/Items/{itemId}/Similar` is not a ranking.** The reference filters on the seed's own genres and tags and orders the result at random `[source: Jellyfin.Api/Controllers/LibraryController.cs:790-801 @ v10.11.11]`; four identical requests shared no item. 005 §3.7 chose determinism deliberately and argued it costs nothing under Principle I — that argument is now standing on a measurement rather than on "not obviously so" | **Diverge, and record it.** Atrium keeps its deterministic scoring, as [behaviours §3.23](../../docs/compatibility/behaviours.md) — class B, through §3.0's first escape hatch: a draw that never repeats gives a client nothing to compensate for, so no compensation breaks when the answer holds still. The entry does **not** argue that the difference is invisible; it argues that nothing can be built on it, and that replicating the draw would leave this the one endpoint the harness cannot compare at all, which is OQ-4's problem above. 005 §3.7 and AC-12 carry it |
| G-2 | **`limit` is not a maximum on a movie seed.** `limit=N` answers **N + 4** rows — measured at 1, 5 and 20, on two seeds — where a series, an album and an artist seed answer exactly N. The reference adds four to any limited query that groups by metadata key `[source: Jellyfin.Server.Implementations/Item/BaseItemRepository.cs:1427-1429 @ v10.11.11]` and this route sets that flag for the movie case alone `[source: Jellyfin.Api/Controllers/LibraryController.cs:795 @ v10.11.11]`; nothing de-duplicates afterwards. The reference's `TotalRecordCount` is the number of rows it returned; Atrium's is the size of the pool before the limit | **Diverge: exactly `limit` rows, on every seed type, with `TotalRecordCount` the pre-limit pool size.** [behaviours §3.24](../../docs/compatibility/behaviours.md) — class B, and taken through §3.0 explicitly: the request succeeds with *more* than was asked, so the default is replicate, and what moves it is that there is nothing consistent to have compensated for. The count difference **is** observable, and the entry says so rather than claiming otherwise; the argument is that the four extra rows are neither stable across the route — three of the four seed types honour `limit` — nor explicable from the wire, so a client cannot have built on a rule the reference does not follow. 005 §3.7 and AC-12 carry it. `[probe: tools/probe_similar_ranking.py, Jellyfin 10.11.11, 2026-09-01]` |

### Also resolved

| # | Question | Answer |
|---|---|---|
| OQ-5 | Whether a reference instance this project may configure and discard can exist | **Yes, and a run makes its own.** Decided 2026-09-01: a run that needs the fixture on both servers stands up a single-use reference instance of the pinned version, gives it the fixture tree as its only library, compares, and destroys it and everything it wrote — including on failure. AC-2 is unblocked and §3.1 states the mechanism; what the instance runs on belongs to this feature's plan. The second reason is that it takes the writing measurements off an operator's server, where 009's runs left 28 playlists behind on 2026-09-01 despite §3.5's cleanup requirement. Every §3.10 row that needs a planted file, a multi-part film, a legacy-encoded subtitle or an empty library is unblocked with it |

### Raised by the task list, and decided

The task list found four readings that the six inherited lists and the compatibility documents owe
and that §3.10 did not carry. Widening §3.10 widens AC-16, which is an edit to an accepted
document, so the list reserved the call rather than taking it — the same shape as D-3 — and
recommended taking it.

| # | The call | The decision, taken 2026-09-02 |
|---|---|---|
| D-6 | Whether behaviours §5.2, behaviours §5.6, 005 OQ-7 and 007's paused-session ticker join §3.10 as named comparisons, widening AC-16 from sixteen | **Yes. §3.10 is twenty rows and AC-16 counts twenty.** All four are differences a sweep cannot raise, which is what §3.10 is for, and all four need exactly the single-use instance OQ-5 unblocked: three want the library changed between two scans, the fourth wants ten minutes of deliberate silence against a paused session. Behaviours §5.2 in particular is the **only surviving `⚠️ UNVERIFIED`** in the compatibility documents, and its own text names a disposable library as the remedy. The alternative was to carry them beside the register as outstanding readings excluded from the count, which is a run reporting *"sixteen of sixteen"* while four questions with a written home go on being nobody's. §3.10 gains the four with their provenance, AC-16 names the count, and this frontmatter carries the amendment |

**Nothing in this document is now waiting on a decision.** The four questions it was written with
are answered above, the two differences it raised against 005 are decided, OQ-5 is resolved, and
D-6 is taken.

## 8. References

- [docs/compatibility/conformance.md](../../docs/compatibility/conformance.md) — the four levels
- [docs/compatibility/reference-target.md](../../docs/compatibility/reference-target.md) — the prior-measurement debts
- [docs/constitution.md](../../docs/constitution.md) — Principles II, VII and VIII
- [specs/005 §3.3](../005-item-query-api/spec.md) — the ignored-parameter delta this feature closes
- The six lists this feature collects: [005](../005-item-query-api/tasks.md#what-this-feature-owes-the-next-ones),
  [006](../006-images/tasks.md#what-this-feature-owes-the-next-ones),
  [007](../007-user-data-and-playstate/tasks.md#what-this-feature-owes-the-next-ones),
  [008](../008-playback-negotiation-and-delivery/tasks.md#what-this-feature-owes-the-next-ones),
  [009](../009-playlists/tasks.md#what-this-feature-owes-the-next-ones) and
  [011](../011-subtitle-delivery/tasks.md#what-this-feature-owes-the-next-ones)
