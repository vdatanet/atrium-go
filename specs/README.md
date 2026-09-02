# Specifications

This project practises **Spec-Driven Development**. This file defines the workflow, the directory
convention and the rules that keep the three artefacts from collapsing into one document with three
headings.

## The loop

```
   ┌──────────┐      ┌──────────┐      ┌───────────┐      ┌──────┐
   │ spec.md  │─────▶│ plan.md  │─────▶│ tasks.md  │─────▶│ code │
   │ WHAT/WHY │      │   HOW    │      │   STEPS   │      │      │
   └──────────┘      └──────────┘      └───────────┘      └───┬──┘
        ▲                                                     │
        └─────────────────────────────────────────────────────┘
            what implementation taught us goes back in the spec,
                          in the same change
```

Each arrow is a **review gate**. A plan is not written against a draft spec; tasks are not written
against a draft plan; code is not written against draft tasks. Principle III forbids
short-circuiting, and the reason is not ceremony: the value of SDD is entirely in the moments where
writing the spec makes you notice that you did not know what you wanted.

The loop closes. Implementation always teaches something the spec did not say — that goes back into
`spec.md` **in the same commit as the code**, not in a follow-up.

## The three artefacts, and what makes them different

### `spec.md` — WHAT and WHY

Observable behaviour. What a client sends, what it gets back, what changes on the server, what
happens on each error path.

**Test for a good spec:** two competent engineers could implement it in two different languages and
their servers would be indistinguishable to a client.

**Forbidden in `spec.md`:**
- Any technology name — Python, FastAPI, SQLite, a table name, a module name, a function name.
- "We will store…" — storage is a plan concern. Say what is *observable*, not where it lives.
- Any claim about Jellyfin without provenance (Principle II).

### `plan.md` — HOW

Architecture, data model, module boundaries, libraries, algorithms, migrations, failure handling.
This is where technology names finally appear.

**Test for a good plan:** an implementer never has to invent a design decision. If they do, the
decision belonged in the plan.

Project-level choices are inherited from [../docs/architecture.md](../docs/architecture.md) and the
[ADRs](../docs/decisions/); a plan restates them only where it deviates, and a deviation needs its
own ADR.

### `tasks.md` — verifiable steps

An ordered list. Each task states **what changes** and **how you know it worked** — a specific test
or command, not "verify it works".

**Test for a good task list:** each task is a reviewable change on its own, and the list has no step
that says "and then implement the feature".

## Directory convention

```
specs/
├── README.md                    this file
├── templates/
│   ├── spec-template.md
│   ├── plan-template.md
│   └── tasks-template.md
└── NNN-kebab-case-name/
    ├── spec.md
    ├── plan.md
    ├── tasks.md
    └── notes/                   optional: probe output, measurements, dead ends
```

Numbers are assigned in the order features are *started* and never reused. A gap in the sequence is
information — it means a feature was abandoned, and the directory says why.

## Status

Every artefact carries a status line in its front matter:

| Status | Meaning |
|---|---|
| `Draft` | Being written. Nothing downstream may start |
| `In review` | Complete, awaiting a gate |
| `Accepted` | Gate passed. The next artefact may start |
| `Implemented` | Code merged, conformance level reached |
| `Superseded by NNN` | Replaced. Kept, not deleted |

## Rules that are easy to break

**No technology in `spec.md`.** The most common failure, and the one that destroys the method: once
a spec names a library, it has started deciding *how*, and the review that was supposed to be about
*what* never happens.

**Every Jellyfin claim carries provenance.** `[probe: …]`, `[source: file:line @ tag]` or
`[spec: operationId]`. An unverified claim is marked `⚠️ UNVERIFIED` and keeps the spec in draft
(Principle II).

**Error paths are specified, not implied.** Absent item, wrong token, malformed profile, unreadable
file — each with its status code and its body. Clients branch on these more than on success.

**Every spec declares its conformance level** per endpoint, using L0–L3 from
[../docs/compatibility/conformance.md](../docs/compatibility/conformance.md). A spec that does not
say how it will be proven is not finished.

## Current specifications

| # | Feature | spec | plan | tasks |
|---|---|---|---|---|
| [001](001-server-identity-and-discovery/) | Server identity and discovery | **Implemented** | **Implemented** | **Implemented** |
| [002](002-authentication-users-and-sessions/) | Authentication, users and sessions | **Implemented** | **Implemented** | **Implemented** |
| [003](003-library-configuration-and-scanning/) | Library configuration and scanning | **Implemented** | **Implemented** | **Implemented** |
| [004](004-metadata-resolution/) | Metadata resolution | **Implemented** | **Implemented** | **Implemented** |
| [005](005-item-query-api/) | Item query API | **Implemented** | **Implemented** | **Implemented** |
| [006](006-images/) | Images | **Implemented** | **Implemented** | **Implemented** |
| [007](007-user-data-and-playstate/) | User data and playstate | **Implemented** | **Implemented** | **Implemented** |
| [008](008-playback-negotiation-and-delivery/) | Playback negotiation and delivery | **Implemented** | **Implemented** | **Implemented** |
| [009](009-playlists/) | Playlists | **Implemented** | **Implemented** | **Implemented** |
| [010](010-conformance-harness/) | Conformance harness | **Implemented** | **Implemented** | **Implemented** — fifteen of fifteen tasks, [D-7 taken 2026-09-02](010-conformance-harness/tasks.md) |
| [011](011-subtitle-delivery/) | Subtitle delivery | **Implemented** | **Implemented** | **Implemented** |
| [012](012-negotiation-inputs/) | Negotiation inputs | **Accepted** | — | — |

**001 through 011 are implemented** — 008 on 2026-08-29 across fourteen tasks,
011 on 2026-08-31 across twelve, 009 on 2026-09-01 across fourteen, and 010 on 2026-09-02 across
fifteen, which leaves **012 the only feature that is `Accepted` and not built**. **008's** spec and
plan were accepted the same day, at a review that wrote and ran the five probes its open questions had
been citing prospectively — all twelve OQs answered, five claims overturned (the policy story, the
body's `EnableTranscoding` switch, `static=true` as an error, `enableRedirection`'s `302`, and the
HLS half of the §3.5 divergence, which measured as parity), and two defects found that nobody was
looking for (behaviours §3.7 and §3.8: the Opus rate ladder applied to every codec, and the
codec-less empty `200`). **Every one of the fourteen tasks then found something further**, which is
why the spec carries thirteen amendments and not one: the sharpest are the four `stream` routes
requiring no token where the task list said the opposite (T6), a `PlaySessionId`-keyed stop whose
mandatory `deviceId` decides nothing (T12), and a `SegmentKeepSeconds` that is a distance behind
the client rather than a file age (T13). **009 was accepted at its own measurement gate on 2026-08-31**, three days after 012's: six open
questions answered by five probes — one extended, four written — and **thirteen claims that did not
survive them**, the sharpest being that `PlaylistItemId` is the item's own `Id`, which is the
distinction the whole feature had been written around. **It shipped on 2026-09-01**, across
fourteen tasks, every one of which found something the documents had wrong — which is why its spec
carries twelve amendments and its plan nine. **010's spec was accepted at its own measurement gate on 2026-09-01**, the day 009 shipped
and the last of its dependencies became implemented. Four probes answered its four open questions
and none survived unchanged: the path it proposed to join two servers on is **absent from every
default list row**, a recorded session replays faithfully and still cannot be the gate, and its two
non-deterministic endpoints are three — `/UserViews` answers a fresh random `ChildCount` on every
request. It also found two differences nobody was looking for, both against **implemented** 005 and
**both decided on 2026-09-01**: `/Items/{itemId}/Similar` is a random draw rather than a ranking,
and its `limit` answers `limit + 4` on a movie seed — Atrium diverges on each, argued in
[behaviours §3.23 and §3.24](../docs/compatibility/behaviours.md). The question that gate left
open went with them: a run that needs the fixture on both servers stands up a **single-use
reference instance** over it and destroys it, so AC-2 is unblocked and nothing in the document is
waiting on a decision. Its new §3.9 and §3.10 are the two halves the
document had no room for — the identities a run needs, measured at 12 of 23 reads answering
differently to a restricted non-administrator, and the comparisons a sweep cannot raise, collected
from [what 008 owes it](008-playback-negotiation-and-delivery/tasks.md#what-this-feature-owes-the-next-ones),
[what 011 does](011-subtitle-delivery/tasks.md#what-this-feature-owes-the-next-ones) and
[what 009 does](009-playlists/tasks.md#what-this-feature-owes-the-next-ones). **Its plan was
drafted on 2026-09-01** and is what the next gate reads. The bulk of it is the thing that does not
exist yet — the single-use reference instance, stood up from a pinned image over the repository's
own fixture, configured over the reference's first-time-setup operations with no human, and
destroyed with everything it wrote — and it reserved **five decisions** for their owner rather than
taking them, **all five taken on 2026-09-01** with every recommendation accepted: a container
runtime for the instance ([ADR-0007](../docs/decisions/0007-a-container-runtime-for-the-reference-instance.md)),
`tools/` and the 3.9 floor unchanged, AC-6 refined and 010's accepted spec amended for it with the
missing `ChildCount` entry written ([behaviours §3.25](../docs/compatibility/behaviours.md)), the
fixture world defaulted with its measurement stated as waiting on that instance, and the
ignored-parameter report's fourth column taken as a file in the data directory rather than a route. Writing it against the files rather
than against the documents moved four claims: `tools/differential.py` is a command line
`conformance.md` publishes and a program nobody has written, the `ATRIUM_JELLYFIN_URL` that
document names as the harness's switch appears nowhere in the repository, the prior-measurement
register is stale in four rows — three of its eight open debts have in fact been paid under another
script's name — and AC-6, applied literally, failed the very allowlist the spec ships — which is what D-3 then refined it out of. **011's spec was accepted on the same day as 008's**, at its
own measurement gate, its plan and task list on 2026-08-30, and the twelve tasks ran from
2026-08-30 to 2026-08-31; **012 was opened on 2026-08-29 and accepted at its own gate the same
day**. **010 landed on 2026-09-02**: its spec, plan and task list were all accepted by that date and
its **fifteen tasks all ran the same day** — the harness, the allowlist, the three registers, the
identities, the single-use reference instance, the fixture composed into six libraries, the twenty
named comparisons, the probe convention and the version-bump command. **Its closing task found the
class 008 T14, 011 T12 and 009 T14 each found, and it refused to flip a status line over it**:
**AC-11 had no test at all** — plan §8 mapped it to *"CI, unchanged"*, a claim about a workflow file
and a fixture with nothing asserting either, in the one feature whose whole value is a second
server; **AC-7 had a half with no test**, the citation a probe prints, which is what turns a finding
into provenance; and **AC-2 said something its own measurement contradicts**. AC-2 claimed the two
servers *"produce libraries with the same item count and the same structure"*, and the recorded
comparison declares **forty-seven differences** over the six fixture libraries — every one of them
003's or 004's, which [spec §2](010-conformance-harness/spec.md) puts outside this feature. Amending
it was **D-7**, reserved for its owner rather than improvised and **taken on 2026-09-02**: the
criterion now states the comparison that exists and runs — the reference's reading recorded,
Atrium's scan compared against it in the default job, every difference declared with its reason and
its owning feature, an undeclared difference failing and a declared one that has gone away failing
too. **`Implemented` there means fifteen of fifteen tasks and eighteen of eighteen criteria, and
nothing wider**: six of the twenty named comparisons are outstanding with their owners and no
`level: L3` row has yet been shown to reach L3, both on 010's own owes list. The lowest-numbered
feature that is not implemented is **012**. Code does not start until a list is `Accepted`
(Principle III). Both
those lists were inputs to the plan. Each of 009's own two gates found something, which
is the rate this project expects: the plan gate found §8 budgeting a fixture task for a second
non-administrator restricted to one library, which `tests/fixtures/query.py` has seeded since 005 —
five criteria called unreachable, none of which were. The tasks gate found the plan pricing *"a
playlist is a row in `items`"* at one migration when it is one migration, three maps and a clause —
and the clause is the one without which `/Items?includeItemTypes=Playlist` answers every user's
private playlists to everybody.

**009's closing task found the class it exists to catch, and one of the three was a criterion with
no test at all.** AC-20 — *"playlist state survives a full library rescan"* — had never been
asserted, on the one item in the store a rescan cannot rebuild; two independent clauses keep a
playlist out of the scan's removal pass and neither had been written down as load-bearing. Two more
proved less than their names: AC-5's *"on both the creation and the addition paths"* had only ever
asked the addition, and AC-13's *"the same three routes answer `404`"* had asked one of three — the
move, the route whose refusals are ordered, being the one it mattered on. And AC-15 asserted that a
request naming a playlist's owner in `userId` is part of the `404` it describes, where every route
in the project that takes that parameter refuses it with the 25-byte `403` that AC-16 and AC-19
assert; the criterion was corrected rather than the code.

**008's own closing task found the class it exists to catch.** The acceptance map is where a
criterion and the test that proves it are put on one line, and doing that showed two criteria whose
tests contradict them — `SupportsTranscoding` derived from the negotiated answer rather than from
the profile, and *"every delivery route whose body has a known size answers `Accept-Ranges: bytes`"*,
which the two playlist routes disprove on both servers — and two more mapped to tests that proved
less than their names: nothing had ever compared the `Size` a negotiation advertises with the bytes
the delivery route serves, and `audioStreamIndex` was asserted as a string in a URL and never as a
property of the audio that came back. The definition of done's *"no other response differs
observably from the measured reference"* was also false: a progressive re-encode produced to a pipe
carries no MP3 header frame and no completed FLAC `STREAMINFO`, which is a **fourth** delivery
divergence and the only one in the feature pointing away from the reference.

**011 was opened on 2026-08-29 for a promise, not for a new idea.** An audit of the two
first-party clients found requirements nothing owns, and the sharpest is one the
[roadmap](../docs/roadmap.md#out-of-scope-and-why) has made since before 001: its exclusion row
excludes subtitle *burn-in* and says in the same sentence that **v1 delivers subtitle files**.
Nothing does. 008 §2 excluded subtitle extraction, conversion and delivery — correctly, for a
feature about deciding a play method — and the feature order ran 001 to 010 with no row to catch
what 008 put down, so the promise fell between two features rather than being descoped by either.
[011's spec](011-subtitle-delivery/spec.md) opened as a draft with **twelve open questions and no
measurements of its own**, which is 008's shape before its gate — and it went the same way.

**Its gate ran on 2026-08-29 and four of the twelve did not survive.** Five probes were written
for it (`probe_subtitle_negotiation`, `probe_subtitle_manifest`, `probe_subtitle_delivery`,
`probe_sidecar_subtitles`, `probe_progressive_production`) and the sharpest finding changes what
the feature is for: **the master playlist does not accept the manifest flag at all**, so the
condition the spec had read as *"a manifest delivery method **or** the profile asking"* has only
one reachable half — and the reference's own negotiation writes the unreadable flag into the
address it hands the client. What announces a subtitle is the delivery **address** naming the
manifest method, which is the client-side override the trace had sized as *"a line inside"* the
main work. *(The gate's wording said "beside a stream index"; 011 T11 measured that the method
alone announces every text track and the index decides only which one is the default — which is
what makes an override a client performs by hand work at all.)* Beside it: burn-in is not a branch
the reference avoids but the answer it gives on every track no profile fits; the default track is
**never** the highest-scoring stream, because the score is only ever read to detect a tie; a
posted subtitle index is dropped in silence unless the request also names the media source; and
the surface grew by **three** rows rather than two, because a negotiation's own `DeliveryUrl`
names a third operation. Two defects nobody asked about: a subtitle playlist's window durations are written in the *server's locale*
(behaviours §3.12), and the playlist route never reads the stream index it is given, so a playlist
for a stream that does not exist is a `200` whose every entry is a `500`. And the two Principle I
questions the spec parked were measured rather than argued: an honest `Content-Length` is an
improvement as recorded, while **keying a transcode on the client's play session is parity** —
the reference already does it on the three routes that declare the parameter, and `/universal`,
the one the music client uses, mints a fresh session per request instead.

**Its plan was accepted on 2026-08-30 and its task list at a gate on the same day, and that gate
found the class this project keeps meeting from a new direction: a document that was true when it
was written.** Plan §6.5 put the manifest's subtitle group on *"the variant line"*, which was
right while the master playlist answered exactly one — and 008's own T15 had given an HDR stream
copy a standard-range entrance beside it hours earlier, so an entrance offering no subtitles would
have shipped to precisely the client the entrance exists for. Three more: the text/image split
reads a **renamed** codec spelling, and against the name a file actually reports the rule inverts
on every DVD and broadcast subtitle track there is — a property 008 already emits, invisible until
now because no fixture had a subtitle stream; the embedded **image** subtitle track the fixture
needs cannot be encoded by ffmpeg at all, so it is written as a bitstream by hand; and the sidecar
naming rule's *"eight regional rows"* are nine, two of which are not regional. **Twelve tasks**,
ordered so that the two stream numberings land before any address carries one.

**It shipped on 2026-08-31, and its own closing task found the sharpest thing in the feature.**
Twelve tasks, every one of them finding something a document had wrong, which is why the spec
carries nine amendments and not one: the manifest lever is the delivery method **alone** and the
stream index decides only which entry is the default (T11); a windowed fetch of the format a track
is already in answers the whole track (T7); a cue sitting exactly on a window boundary is delivered
**twice** (T5); the marker for a stream with no language is `Undefined` and not the `Und` the
assembly falls back to (T10); and naming a subtitle track the client cannot take costs the source
its **direct play**, which made AC-15 false of exactly the request the feature exists to serve
(T9). Then T12, writing the acceptance map, found that the plan's own risk register had predicted
the feature's one real defect and prescribed a mitigation that could not catch it: `-map` was given
the **wire** stream index where ffmpeg counts the demuxer's, so every remux, transcode and HLS
segment of a film with a subtitle file beside it mapped one stream too far — a remux of the one
such fixture answers `200` carrying no video stream at all. Nothing saw it because every
produced-bytes test in the repository ran over a source with no external stream, T1 having put the
sidecar beside a film 008 asserts nothing about for exactly that reason. Four criteria were also
mapped to tests that proved less than their names, and the definition of done's *two* divergences
are three: burn-in is the answer the reference gives for every track no profile fits, and v1 says
the word and produces the frames without the cues.

**Its scope argument is the roadmap's *"008 is one feature, not two"* read backwards.** The two
client traces
([tvOS](../docs/compatibility/client-atrium-tvos.md#6-where-these-findings-go),
[music](../docs/compatibility/client-embeat-mobile.md#7-where-these-findings-go)) both close by
routing every finding to *"the feature that comes after 010"* — which is this one — while insisting,
correctly, that **none of them is an 008 defect or an amendment to it**: 008's code does what 008's
documents say, and a specification silent about a case is not one that is wrong about it. *An
earlier draft of 011 §2.1 called four of them amendments owed to 008, 007 and 002, and the client
documents landing on `main` overturned it.* But a destination is not a scope: each trace then
decomposes its findings into a grouping table whose rows are visibly different shapes — a 009 scope
decision, an amendment to 001, a sentence of prose, a test that *"can be written today"*. Eleven
findings do not become one feature by sharing a date. 011 therefore takes the two the video client's
own table had already grouped and sized — *"§4.2 + §4.3 — subtitle delivery, end to end. The largest
of them"* — and hands the rest on at the size their own documents measured: a source with no
inspection that still advertises direct play, `"Hls"` not selecting HLS where `"hls"` does, a
session list that takes no `deviceId`, an initialisation segment that restarts production, and the
one question about where a progressive re-encode is produced that the music client asks three ways.
Four of those are one probe away from being specifiable and none can be specified before its probe,
so they are a feature whose first act is a measurement session, taking its number on the day that
session runs. **The ordering finding is 005's**, and its own document says the cheap answer is a
test rather than three new sort keys: the album play queue is correctly ordered only as a side
effect of [behaviours §2.6](../docs/compatibility/behaviours.md#26-sortname-has-two-derivations-and-three-types-use-the-second)'s
sort-name derivation, and nothing states the dependency.

**012 took that number on 2026-08-29, and its first act was to apply 011's own test to 011's own
handover.** The four findings 011 handed on arrived grouped by the fact that they were handed on
together, which is the failure mode 011 §2.1 named — *"eleven findings do not become one feature by
sharing a date"* — and the roadmap's *"008 is one feature, not two"* test, applied to them, keeps
**two**. `PlaybackInfo` has exactly two inputs, and v1 has a lenient branch on each: a source with
no stored inspection is stepped over, and a profile whose protocol is spelled in an unexpected case
falls through. Both answer `200` with something a client cannot act on — a capability with no
address, and an address of a shape the same answer says it is not — so they are one rule seen from
its two sides, and [012](012-negotiation-inputs/spec.md) is that rule. **The other two are handed
on again**: the session list's missing parameters belong to the feature that owns the route, and the
initialisation segment that restarts production is a
[behaviours §3.0](../docs/compatibility/behaviours.md#30-how-the-decision-is-made) defect decision
rather than a requirement — both measured at 012's gate, neither specified there, which is 011's
own OQ-9/OQ-10 device.

**Opening it corrected the handover in five places, all of them by opening a file.** The session
list is **002's** route, not 007's — `feature: "002"` in `surface.yaml`, specified in 002 §3.8, and
the video client's trace says so in as many words — and it is **three** parameters rather than one.
The reference **has no un-inspected source to describe**: its negotiation refreshes the item with
probing when the first source carries no stream of the item's own kind
`[source: Emby.Server.Implementations/Library/MediaSourceManager.cs:170-189 @ v10.11.11]`, so the
decision may be *reproduce the on-demand inspection* rather than *decide what an un-inspected
source advertises*, which is a different decision with a different cost. Its **listing** path does
not probe `[source: Emby.Server.Implementations/Dto/DtoService.cs:261 @ v10.11.11]`, which puts
*"one root cause, two clients"* in doubt on the cure even where it holds on the cause — the music
client never negotiates, so the reference's on-demand probe never fires for it. And the
initialisation-segment claim is **no longer third-party**: the line the client's contract cited has
now been read, and the restart is the first branch, taken before the reference looks at what is
running `[source: Jellyfin.Api/Controllers/DynamicHlsController.cs:1501-1505 @ v10.11.11]`. Nine
open questions, each naming the probe that answers it, and **not one measurement**: like 011's, its
next gate was a measurement session.

**That session ran on 2026-08-29 and 012's spec is accepted.** Four probes — two written for it,
two extended — answered all nine, and the measurements were harder on the doubts than on the
claims. **The reference does have an un-inspected source to describe, but only an un-*inspectable*
one**: a readable file is annotated inside the negotiation that asks for it, in 0.20 s, and what it
learns is **kept**, so the next listing carries it. **The music client's half is parity and the
claim that it was a gap is withdrawn** — a stock reference answers a listing the same empty source
Atrium does — and *"one root cause, two clients"* survives through the negotiation's **write**
rather than through anything on the listing path, which is not the cure either trace proposed.
An **audio** item with no audio stream refuses the whole request with `400` where a video item
answers `200` with an address, and the address a video item is given resolves to a live playlist
that answers `500`. The protocol question had two candidates and needed four: altered cases bind,
ordinals bind, an empty string takes the default, and only a word that binds to nothing refuses.
And the initialisation-segment restart is guarded by a file check the same session has already
satisfied, so it costs nothing in either order a client uses — decided *replicate* under behaviours
§3.0, at §3.14.

**007's thirteen tasks found something in seven of them, and two were features that did not
exist.** The sharpest is T11's: **the container `PlayedPercentage` had never been implemented.**
`PlayedPercentage` was position-over-runtime for every item — the *leaf* reading — so AC-20's
first half ("a bare container row carries no percentage") passed because there was no percentage
to gate. The second half was unreachable. T8's is the same class seen from the wire: **this
project's first typed request body answered `{"item_id": …}`**, snake_case, because the framework
keys validation errors on the model's Python field — behaviours §1.1's exact failure, in a body
nothing had ever sent, since 002's only body is read with `request.json()` and never bound. The
routes now name their body parameters after the reference's and the handler files a body failure
under `""` or `"$"` beside `The <parameter> field is required.`, measured byte for byte.

**Three findings came from measuring rather than reasoning.** T1's probe run found that
`NowPlayingItem`'s **width is the item's, not the shape's** — two movies measured 41 and 40
properties, the difference being a null `IsHD` — which is a false positive waiting for 010's
differential if it compares counts. T9 measured the property *list* rather than the count and
replaced the plan's design with it: the shape is a **subtraction**, a full item body minus a named
fifteen, so 005's existing `omit` mechanism expresses it exactly and `MediaSources` is already
excluded for the day 008 emits it. And T2, implementing the six-branch rule, found that
**row 4's second clause decides nothing** under the reference's own thresholds: "within one second
of the end" implies "past 90%" for anything longer than ten seconds, and anything shorter is
completed by the runtime floor — the spec's paragraph explaining why the clause was *not*
redundant had the arithmetic backwards.

**And two were about this repository rather than about Jellyfin**: `last_playback_check_in` had
**no writer at all** — 002 created the column, reflected it back and never moved it, so a session
that had played something reported `0001-01-01` for ever (T7) — and three of T8's route tests were
passing for the *fixture's* reasons rather than the route's, because the seeded films carry a
resume position and "nothing was written" was reading numbers the world had put there.

**OQ-7 was resolved with an answer the question did not anticipate** (T11): for four of the five
container types it cannot be asked at all, because an empty `Series`, `Season`, `MusicArtist` or
`MusicAlbum` does not earn its place and is not offered. The one exemption is a library folder,
where Atrium reads `Played: false` and the reference's source reads vacuously played —
[behaviours §5.7](../docs/compatibility/behaviours.md), owed to 010.

**007's task-list gate changed four things**, on 2026-08-28, and the first is the class 006's
gate taught, back for the very next feature. **The seeded world has exactly one runtime** —
`tests/fixtures/query.py` gives one to a single film and to nothing else — so §3.7's rule, which
is a function of runtime, had one item to run on at route level, and **row 5, the short-item
branch OQ-6 opened and measured, had no world at all**. **Nothing has ever written
`last_playback_check_in`**: the plan hands the column to 002's activity flusher, whose `touch`
writes `last_activity_date` alone, so "the flusher's existing pass writes both columns" is a
change to make rather than a property to lean on. **OQ-7 belongs to this list**, not to 010 —
the fixture library can build an empty container, so the empty-subtree answer is a decision to
take here, and today it lives in a docstring, which is exactly how 006 found an exception
withdrawn three features earlier. And **AC-16 needs no new test**: 003's own AC-11 already
plants a favourite and a resume position, deletes the file, rescans and restores it, from the
other side of the same guarantee. Each is recorded in
[007's tasks](007-user-data-and-playstate/tasks.md#what-the-gate-changed).

**007's plan gate measured before accepting, and the sharpest answer was an absence**, on
2026-08-28. The gate ran plan §6.8's four catalogued batteries as hand requests against the
live reference: a playing session's `NowPlayingItem` — a `BaseItemDto` width nothing had ever
captured — carries 41 properties and **no `UserData`**, sits between `DeviceName` and
`DeviceId`, and includes nine media-derived properties v1 cannot yet emit, now a named gap in
the spec rather than a silent one; `PlayState` is **replaced whole by each report** — a
progress omitting `CanSeek` reads back `false` — where the draft plan had left merge-or-replace
to the implementer; the error shapes all landed on behaviours §1.11's existing taxonomy
(problem-details `404`, validation `400`, the `text/plain` controller refusal for a negative
position, the empty `401`); a `Start` carrying 30% leaves the stored position at 0; and a
movie's `UserData.Key` measured as the item's own GUID **in dashed form** beside the 32-hex
`ItemId` — one object spelling one identity two ways. Spec §3.6 gained the playing-session
block and AC-21/AC-22; nothing measured contradicted the plan's structure, and the plan moved
to `Accepted` the same day.

**007's plan stores nothing new**, on 2026-08-28: `item_user_data` has been complete since 003
— the deliberately absent foreign key *is* the survival guarantee — so the plan is five
decisions about writers. The measured semantics become pure functions in `domain/playstate.py`;
live playback state stays in memory with the reference's ticking position computed at read time
rather than by a per-second timer; the cascade is a write-time sweep over the leaves through
005's visibility scope; the mark responses are built by the same hydration path as list rows;
and the controller split mirrors the reference's. Plan §6.8 catalogues what no probe has
measured — sharpest, the playing session's `NowPlayingItem`: a `BaseItemDto` width nothing has
captured, which is 005 T1's lesson pointed at `/Sessions` — for the gate to answer before
accepting.

**007's spec review measured first and corrected four accepted-draft claims**, on 2026-08-28.
Reading the reference's source predicted all four and the extended `tools/probe_playstate.py`
confirmed each on the wire: a bare `POST /UserPlayedItems` is **`max(count, 1)`** — only the
`datePlayed` form increments, so AC-3's "increments" was wrong; **nothing guards against an
older position** — a progress at 40% then 20% reads back 20%, reversing AC-10, because a
deliberate seek backwards arrives as exactly that report; **a play is counted at `Start`**,
which also sets `Played` to *false* on a previously played item, while a positionless stop
counts a second time; and **the six-branch rule runs on every position-bearing report**, so a
progress past the ceiling marks played mid-playback. The `--reap` battery answered OQ-4 with
more than the question asked: the session cleared after 8.6 minutes of silence and the stored
position was **48.5%, not the reported 40%** — a per-session one-second ticker extrapolates the
unpaused position in real time and the reap commits the extrapolated value, so AC-15's "last
position intact" came back 8.6 minutes richer (spec §3.8). Also pinned: strict boundaries at
tick precision, the cascade that writes leaves and never the container's own row, favourites
that do not cascade, the field-gated container `PlayedPercentage`, and OQ-1's survey — no
analysed client reads `UserData.Key`. One question opened: OQ-7, the empty container the
source reads as vacuously played where 005 shipped unplayed.

**006's thirteen tasks found something in nine of them, and three were in documents that had
already been accepted.** The sharpest is T12's: **the image tag could never change.**
`Field.IMAGES` merged under the rule that keeps whatever an item already has unless the refresh
mode is `Replace` — and v1 has no refresh route through which anything could ask for `Replace` —
so an item that had ever been given artwork could never be given different artwork, at any scan
depth. AC-2's second half was unreachable and client-side cache invalidation with it: a tag that
cannot change is a poster that can never be corrected. The field is `REDERIVED` now (004's plan
§6.1 carries the amendment), and the residual limitation is recorded rather than hidden —
a *default* scan reads an item's artwork only when its media file changed, so a replaced poster
needs a deep scan
([behaviours §5.6](../docs/compatibility/behaviours.md#56-a-default-rescan-does-not-notice-a-replaced-poster)).

**T6 deleted a universal the spec had stated three times**: "never upscale" is a property of
*which parameter was sent*, not of the server. `maxWidth`, `maxHeight` and the fill pair are
capped at the source; `width` and `height` are honoured past it — `width=4000` of a 2000×3000
source measured **4000×6000**. Implemented literally, Atrium would have answered a *smaller*
image than a client asked for by name, on the one path whose entire meaning is "this size".

**T1's probe found what its own output had been printing since the spec review**: a forgiven
parameter is not a dropped one. `maxWidth=-100` answers `200` at the source's dimensions and
**three times its bytes** — the reference re-encodes — while a bare `quality` does *not* transform
at all, where the plan had made it a reason to re-encode every poster for the clients that append
one out of habit ([behaviours §1.17](../docs/compatibility/behaviours.md)).

And three that are about this repository rather than about Jellyfin: **behaviours §4.4 had been
withdrawn for three features without anybody saying so** — 005 T4 reversed it and 006's task list
still cited it as standing (T3); **T5's hostile-path test passed with the containment check
deleted**, because `../../../../etc/passwd` from a `tmp_path` root reaches nothing, so the
refusal it asserted was the wrong refusal; and **T8's AC-8 failed against the obvious
implementation**, because deciding the transform from the file's dimensions rather than the
row's makes the cache key move whenever the file does.

**006's task-list gate changed four things**, on 2026-08-28 — two of them the exact classes
earlier gates taught, back for the very next feature. [Spec §6](006-images/spec.md#6-conformance)'s
"Indexed form" conformance row had no task holding its **positive** case — every index test in
the draft was an error test, and nothing asserted that `/Backdrop/1` returns backdrop 1; AC-14's
discriminating fixture **does not exist** — no seeded episode carries artwork of its own, because
005 never needed one, so "inheritance does not gate on the child's own images" was a criterion
with no world to prove it in, 005's fixture lesson one feature later; the draft cited an
all-routes PascalCase canonicalisation test **that does not exist in that shape** — found by
opening `tests/unit/test_compat_query_params.py`, not by re-reading the list, which is 003's
method paying again; and AC-12's "over the mechanism list itself" now names the importable
enumeration, so "not a copy" is an import rather than an aspiration. Each is recorded in
[006's tasks](006-images/tasks.md#what-the-gate-changed).

**006's plan gate measured before accepting, and the measurements changed the accepted spec
twice**, on 2026-08-28. The plan's §6.8 had catalogued the edges no probe had covered; a
manual-request sweep answered them, and two answers contradicted accepted documents. **AC-6 was
corrected** — `fillWidth`/`fillHeight` do not crop: they scale to cover and keep the overflow,
300×600 asked of a 2000×3000 source returning 400×600. The earlier probe had measured "exactly
the box" on a source that was itself square, where covering and cropping are indistinguishable —
the second acceptance criterion this project has reversed by measurement, 005 AC-11's class.
**AC-15 was added** — a transformed response negotiates `Accept: image/webp` under a
`Vary: Accept` sent on every image response. The plan's own §10 had rejected content negotiation
as a delta, and the measurement reversed the rejection: the earlier probe's offer rode a request
nothing transformed, which negotiates nothing, so every browser-based client was quietly owed
WebP posters no document promised. The sweep also found a **fourth error shape**
([behaviours §1.11](../docs/compatibility/behaviours.md#111-there-are-four-error-shapes-not-one)):
the route's absent-image `404`s answer a JSON-encoded string naming the item — on a tokenless
route — while its unknown-item `404` stays problem details, one route splitting its two lookups
across two shapes. The rest confirmed the plan as drafted: invalid tokens change nothing,
`format=Banana` drops, `Svg` short-circuits to the verbatim path, both-axes resizing distorts,
the `304` is the `200`'s header set minus `Content-Length`, and `?imageIndex` selects the
backdrop it names. Every answer is folded into [the spec](006-images/spec.md),
[the plan](006-images/plan.md) and behaviours §1.11, with EXIF orientation the one edge a remote
request cannot reach.

**Writing 006's plan changed one row of the accepted spec**, on 2026-08-28. §3.2's error table
read "`imageType` outside §3.2's set → `400`", and the measurement it cites distinguishes the
reference's thirteen-member **vocabulary** from an item's holdings: a string outside the
vocabulary is `400`, while `Box` — a member outside §3.2's eight — measured `404`. Implemented
literally, the eight-member reading would have manufactured a `400` where the reference answers
`404`, on the first request any probing client sends for a type this server does not carry. The
row now names the vocabulary. The plan's §6.8 lists the six edges no probe has measured — an
invalid token on the tokenless route, the format-vocabulary collision, both-axes resizing, the
error bodies, the `304`'s headers, the query-spelling index — each owed a measurement task before
its code lands.

**006's spec gate ran probes first, and both halves moved the document**, on 2026-08-28. The
review found the exact class 005's gate named — two documents disagreeing with a measurement
between them: §3.2 required authentication while 002 AC-3, the criterion it claimed to share,
records the measured opposite. The decision behaviours §2.10 had deferred to 006 is now taken
the way every prior collision resolved — a token accepted, none required, an item id a
capability. Two new probes answered five of the six open questions the same day, and the
sharpest finding was one no reading could reach: the reference sends **no `ETag` and no
`Accept-Ranges` on an image response** — `Last-Modified`/`If-Modified-Since` is the validator
pair it actually serves, so §3.4 and AC-9 now assert the pair that exists. The rest of the
measurement: a stale `tag` serves the current image byte-identically (AC-10 is a reproduction),
an unparseable dimension is `400` — the one measured error path that is **not** lenient, where
behaviours §1.12 would have predicted forgiveness — an explicit `format=Jpg` on a transparent
logo is honoured and discards the alpha, and chapters advertise `ImageTag` per `Chapters` entry.
OQ-4 stays open for 010's differential harness. The record is
[006's spec](006-images/spec.md) itself: every answer went back in with its citation, in the
same change.

**005's seventeen tasks kept the measured-first habit paying**, and the pattern sharpened: this
time the documents lost *acceptance criteria*, not only claims. AC-11 was **reversed** — season
0 sorts first on the measured wire, not last as the spec argued clients expect — AC-13 was
restated because a consequence already recorded in behaviours §5.3 makes the drafted containment
structurally unsatisfiable in Atrium, and AC-14 required populating `MatchedTerm`, a field
seventeen measured hints never carried. T1 had already split "one item representation" into
three route-dependent widths, T11's measurement overturned the Latest grouping rule (a group of
one surfaces as the item, not its container), and T13's probe confirmed the NextUp chain with
the one discriminating case reading could not settle. Each Done note in
[005's tasks](005-item-query-api/tasks.md) records one.

**005's task-list gate changed five things**, on 2026-08-27 — the two previous gates' class,
promises with no task holding them, plus a new one: two accepted documents disagreeing with
nothing measured between them. AC-1's "every list endpoint" was held one endpoint at a time,
with no test saying *every*; the spec and the plan disagree about whether search hints match the
sort name, which is now measured rather than arbitrated; a new probe had no row in
`tools/README.md` — the exact omission 004's gate caught, back for the very next script; the
filter summary's computation appeared in no accepted document; and the plan's own fixture
paragraph seeds one series where its own test table proves NextUp on three watched ones. Each is
recorded in [005's tasks](005-item-query-api/tasks.md#what-the-gate-changed).

**004's sixteen tasks contradicted eleven things the accepted documents asserted**, which is the
highest rate any feature has managed and the reason its ordering put measurement first. The three
that changed the most code: the reference **splits a genre on a slash** where plan §6.2 said it
does not and cited the parser that does; the **path-derived name is merged last, not third**,
without which AC-1 — "a film with a full `.nfo` resolves entirely from it" — is unreachable; and
the culture table is **not** the ISO 639-2 registry plan §6.9 named but a 192-row list only the
reference has. Each Done note in [004's tasks](004-metadata-resolution/tasks.md) records one.

**Its task-list gate had already changed three things** of the class 003's gate taught — promises
with no task holding them: AC-1 was only proven in a world with no remote code, the plan's opt-in
live test had no task, and a new tool had no row in `tools/README.md`. All three were delivered,
and the first turned out to be the most valuable thing on the list.

**002 measured more than it implemented.** Its eighteen tasks contradicted four things the accepted
specification asserted — a fifth authentication mechanism the surface had never listed, a disabled
account refused with `403` rather than the `401` the spec argued for on purpose, a client-header
grammar stricter than "lenient" in two ways, and `/Users/Public` disclosing every user's policy to
an unauthenticated caller. Each of them was one request away, and none was reachable by reading.

**All four probes have been run**, on 2026-08-26 against a live Jellyfin 10.11.11. Three confirmed
the documentation and one contradicted it:

| Question | Outcome |
|---|---|
| 005 OQ-6 — list envelope shapes | Confirmed, **plus three shapes** the original measurement never covered |
| 003 OQ-3 — sort-name derivation | Confirmed 15/15, **plus a second rule**: three item types bypass it entirely |
| 007 OQ-2 — completion thresholds | Answered: 90% / 5% / 300s, **and six branches** where the spec had two |
| 009 OQ-1 — `Move` semantics | **Contradicted.** The spec had the reading backwards; §3.5 and AC-8 corrected |

**Five more were run at the 009 spec gate**, on 2026-08-31 — `probe_playlist_move.py` extended and
`probe_playlist_creation.py`, `probe_playlist_expansion.py`, `probe_playlist_visibility.py` and
`probe_playlist_rename.py` written — and it is the gate at which the most claims died at once:

| Question | Outcome |
|---|---|
| 009 OQ-3 — container expansion | Confirmed, **and wider**: album, artist, series, season and collection all expand, and an album keeps its own order |
| 009 OQ-4 — entries the reader cannot see | **Contradicted.** Nothing is filtered for library access; the omission §3.7 described was Atrium's alone and is now a divergence (behaviours §3.17) |
| 009 OQ-5 — client reliance on media deletion | Answered on the documents: none. The §3.6 divergence costs both analysed clients nothing |
| 009 OQ-6 — the `Move` boundaries | **Contradicted, every row**: a clamp one position wide, then `500`; a negative index that moves rather than refuses; an absent entry that is a silent `204` (behaviours §3.15) |
| Not asked — entry identity | **`PlaylistItemId` is the item's `Id`**, on the wire. §3.1, AC-5, the surface's own note and a client contract sentence all said otherwise (behaviours §2.26) |
| Not asked — who may read | A private playlist is readable by **anyone who names its owner** in `userId`, on the one route that does not check (behaviours §3.16) |
| Not asked — who may rename | The route the music client renames with is administrator-only, so a playlist's own owner is refused `403` (009 §3.8) |

**Four were written and run at the 010 spec gate**, on 2026-09-01 —
`probe_similar_ranking.py`, `probe_differential_join.py`, `probe_reference_determinism.py` and
`probe_restricted_surface.py`, the last of them the first probe in this repository to measure from
a seat that can be refused something:

| Question | Outcome |
|---|---|
| 010 OQ-1 — joining two servers' items | **The remedy died, not the premise.** Comparison by path was the way out and the path is not on the wire: 0 of 1000 default list rows carry one, and asked for by name it still leaves a virtual season, a remote channel and every by-name row unjoinable. `(Type, Name)` is 976 distinct of 1000 |
| 010 OQ-2 — how many request cases | **Not a question about the reference.** Its measured input is 764 declared query parameters over the 59 endpoints; what settles the floor is that both differences this gate found are invisible to a bare request |
| 010 OQ-3 — a recorded session | **Yes for the bodies, no for the feature.** 16 of 19 reads are byte-stable and only the response time and the clock move — but a recording answers only what it recorded, which is the class L3 exists to find |
| 010 OQ-4 — non-deterministic responses | **Two endpoints are three**, and the third is `/UserViews`: its `ChildCount` is a fresh random integer between 1 and 9 on every request. And a field-level allowlist cannot express any of them |
| Not asked — `Similar` | **It is not a ranking.** A random draw over items sharing the seed's genres and tags; four identical requests shared **no** item |
| Not asked — `Similar`'s `limit` | **`limit + 4` on a movie seed**, exactly, at 1, 5 and 20 — where a series, an album and an artist answer exactly the limit |
| Not asked — the identity a run uses | **12 of 23 reads answer differently to a restricted non-administrator**, and two of them differ as shorter lists rather than as refusals |

**Two more were written and run at the 004/005 spec gate**, on 2026-08-27, and the pattern held:

| Question | Outcome |
|---|---|
| 004 OQ-3 — genre re-normalisation | Confirmed: 97 of 97 by-name ids reproduce from the case-folded name, so §3.7 rule 1 is a reproduction, not a divergence — **and the merge was caught live**, two spellings on items collapsing into one row (behaviours §2.18) |
| 005 OQ-3 — sort tie-breaking | Answered: the reference appends almost nothing, **and its own artist-sort paging drops and duplicates rows** — the defect 005 §3.4 rule 2 now diverges from on the record (behaviours §3.6) |

The same gate found by hand-measurement that the accepted 005 spec's error path for enum values
was **wrong** — an unrecognised token is ignored, not `400` (behaviours §1.12) — and that query
parameter **names** match case-insensitively, which no route had needed before 005
(behaviours §1.15).

**003's task list changed at its gate**, on 2026-08-27, and the two changes that mattered were
tasks that were *not in it*: nothing measured the two open questions the specification names probes
for, and nothing extended the acceptance map — which `test_every_implemented_feature_has_a_map`
would have failed on the day 003 was marked `Implemented`. Reading a list tells you whether its
steps are right, not which step is missing; both were found by checking the list against files in
this repository. Four smaller corrections are recorded in
[003's tasks](003-library-configuration-and-scanning/tasks.md#what-the-gate-changed).

Every one returned more than it was sent to check. That is the argument for running a probe before
writing a plan rather than after. Running them is the cheapest work available and it
changes what the specs say, so it belongs before the plans, not after. See
[tools/README.md](../tools/README.md#probes).

The order and rationale are in [../docs/roadmap.md](../docs/roadmap.md#feature-order).
