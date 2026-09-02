# Roadmap

**Status: written 2026-09-02, for a repository with no code in it yet.**

This document says **where v1 stops, why it stops there, and in what order the twelve features get
built.** It is the answer to *"should Atrium do this?"*, which is a different question from
*"what does Jellyfin do?"* — that one is
[behaviours.md](compatibility/behaviours.md) — and from *"which endpoints are v1?"*, which is
[api-surface-v1.md](compatibility/api-surface-v1.md).

**Two halves of this document have different standing, and it matters.**

The **scope** is not new. Every row of §"Out of scope, and why" is a commitment the exported
documents already cite and lean on: 011 was opened because of one of them, the video client's
analysis turns another into arithmetic, and 006 defers a whole feature to a third. Those rows are
restated here — in the receiving project's own words, because the source's roadmap was withheld —
and a row that contradicted a document citing it would be a bug in this file, not a scope change.

The **order** is genuinely this project's, and it is where this document differs from the first
run. Nothing about build order crosses the experiment's boundary: the question being asked is
whether one set of specifications produces two indistinguishable servers, not whether two teams
schedule the same way.

---

## In scope

v1 is **59 of Jellyfin's 322 paths**, chosen by named consumer rather than by importance
([api-surface-v1.md](compatibility/api-surface-v1.md)), and served to
[L2 everywhere, L3 on the authentication and playback paths](compatibility/conformance.md).

| Area | v1 does |
|---|---|
| Identity and discovery | The public and authenticated system info, ping, public users, cultures |
| Authentication and sessions | All five credential mechanisms, both spellings of the client header, session tracking, capabilities |
| Library and metadata | On-demand and scheduled scanning, incremental and deterministic; `.nfo` sidecars; tag-derived music metadata |
| Query | The whole read surface — filtering, sorting, paging, `Fields`, search, by-name endpoints, resume, next up |
| Images | Serving, resizing, format negotiation, on-disk cache |
| User data | Favourites, played state, playback reporting |
| Playlists | Create, read, add, remove, move, rename, delete |
| Playback | **Direct play, remuxing, and software transcoding**, progressive and over HLS |
| Subtitles | Delivery of tracks that already exist — sidecar and embedded, converted to the format the client asked for |

### Why the playback ladder goes as far as transcoding, and no further

**Direct play, then remux, then transcode.** Direct play costs nothing. Remuxing copies the
elementary streams into a different container — no decode, no encode, near-zero CPU, and an output
whose size is computable — and it covers the large majority of real playback, because most
incompatibilities are container mismatches rather than codec ones. Transcoding costs a decode and an
encode per frame, so it is last, reached only when the first two have failed.

**Transcoding is in v1 because the alternative answer at that point is "cannot play this".** A file
the user owns and cannot watch is the one failure with no cosmetic version: everything else v1
leaves out degrades into something a client can live with, and this degrades into a black screen.

> **Transcoding entered v1 on 2026-08-27, and it was not there before.** It is recorded as a dated
> change rather than presented as the original plan, because a document that quietly absorbs its own
> amendments cannot be audited. The change had a visible consequence the same day:
> [002 §3.5](../specs/002-authentication-users-and-sessions/spec.md) moved the three transcoding
> permission flags out of the accepted-unenforced set and into the enforced one — *"the feature
> arrived, so the flags that restrict it stopped being unobservable"*.

---

## Out of scope, and why

**This table owns the *reasons*.** [api-surface §10](compatibility/api-surface-v1.md#10-deliberately-excluded-from-v1)
owns the endpoint-level view of the same exclusions, and the two are a pair: a row that moves here
moves there in the same change.

| Excluded | Why |
|---|---|
| **Subtitle burn-in** | A text-rendering stack — fonts, ASS positioning, shaping — and a second filter path. **v1 delivers subtitle files**; it does not paint them into frames. |
| **Hardware-accelerated transcoding** | **v1 encodes on the CPU — slower, but portable and testable on any machine that can run the test suite.** VAAPI, QSV, NVENC and VideoToolbox are a per-machine surface, not an endpoint, and each needs hardware CI cannot have. |
| **Trickplay and chapter-image generation** | Decoding a video at intervals, as a background sweep over the whole library. v1 *routes* chapter images and can never put one on disk, which is an accepted gap rather than a broken route ([behaviours §5.8](compatibility/behaviours.md)). |
| **The Emby dialect** | Emby's pre-flattening routes — `/Users/{userId}/Items/{itemId}` and its relatives — which 10.11 itself no longer serves. Serving them would be a delta in the one direction Principle I forbids: an endpoint the reference does not have. |
| **Live TV, DVR, channels, tuners** | A separate product domain with its own hardware surface. |
| **SyncPlay, and the WebSocket `/socket`** | Push notification of library changes and a shared session model. v1's clients poll, and both analysed clients work that way today. |
| **Plugins, packages, repositories** | Assembly loading into a running server. There is no analogue here worth inventing, and the surface it would open is larger than v1 entire. |
| **DLNA server and profiles** | Outside the client-facing contract. |
| **Backup, scheduled tasks, activity log** | An operations surface, not a client surface. |
| **Quick Connect** | Convenience authentication that adds a second auth state machine beside the one 002 specifies. |
| **Subtitle provider search and download** | A network dependency on third-party services, and a moderation surface. Delivery of tracks that exist is in; acquiring new ones is not. |
| **Books, photos, home videos** | Outside the stated media scope. |
| **The Jellyfin web UI** | Not a target ([reference-target §5](compatibility/reference-target.md#5-what-is-not-a-target)). Atrium is a server. |

**Two of these rows are load-bearing beyond their own scope**, which is why they are worth reading
even by someone who agrees with them:

- **The burn-in row is a promise as much as an exclusion.** It excludes burn-in and says in the same
  sentence that v1 delivers subtitle files — and for nine features nothing did. The promise fell
  between two specifications rather than being descoped by either, and
  [011](../specs/011-subtitle-delivery/spec.md) exists to keep it. What survives is narrower and
  still real: a track whose negotiated delivery method is `Encode` is announced as burned in and is
  not ([behaviours §5](compatibility/behaviours.md)).
- **The CPU row decides what a small server can produce**, not just what this project builds. Where
  a negotiation answers with a re-encode, that re-encode is a CPU one, and for an HDR film on a
  low-powered host that is not merely slow but infeasible — which the video client's analysis works
  through in arithmetic ([client-atrium-tvos](compatibility/client-atrium-tvos.md)).

**Growing this surface is a decision taken here, not one an implementer takes opportunistically.**
An endpoint arrives with a named consumer, a row in `surface.yaml`, an owning feature and a declared
conformance level, or it does not arrive.

---

## Feature order

### What the dependencies allow

```
001 identity ──┐
               ├── 002 auth ──┐
003 scanning ──┴── 004 metadata ──┴── 005 query ──┬── 006 images
                                                  ├── 007 user data
                                                  ├── 009 playlists
                                                  └── 008 playback ── 011 subtitles
                                                                   └── 012 negotiation inputs
```

Taken from each spec's own `depends_on`. It is a partial order, not a sequence: 003 owes nothing to
001, and 006, 007 and 009 owe nothing to each other.

### The order this project builds in

| # | Feature | Endpoints | Why here |
|---|---|---|---|
| 1 | **001** Server identity and discovery | 4 | The smallest surface that exercises **every whole-project constraint at once** — see below. |
| 2 | **002** Authentication, users and sessions | 7 | Nothing else can be reached by a real client without it, and it is the first L3 path. |
| 3 | **003** Library configuration and scanning | 0 | No HTTP surface of its own. Everything downstream needs a library to answer about. |
| 4 | **004** Metadata resolution | 1 | What a scan found becomes items with fields. |
| 5 | **005** Item query API | 17 | The workhorse, and the largest single feature. A client can browse. |
| 6 | **006** Images | 2 | The first point at which a client *looks* correct rather than merely working. |
| 7 | **007** User data and playstate | 7 | Favourites, played state, and the reporting routes playback will use. |
| 8 | **009** Playlists | 7 | Depends only on 005, and closes the read-and-write surface before the hard feature starts. |
| 9 | **008** Playback negotiation and delivery | 11 | The largest risk and the deepest feature: the first that opens a file and the first that runs a subprocess. |
| 10 | **011** Subtitle delivery | 3 | Shares 008's delivery machinery and the manifest it emits. Built beside it, not nine features later. |
| 11 | **012** Negotiation inputs | 0 | Refines what 008 decides with, once 008 exists to be refined. |

**45 of the 59 endpoints are routed before 008 begins**, and the remaining 14 are the delivery
surface. That is the point of putting 009 at position 8: the first serious differential sweep can
cover three quarters of the surface before the feature most likely to invalidate an assumption
starts.

### 001 carries the edge, and that is deliberate

001 is four endpoints and looks like a warm-up. It is not. It is the smallest feature that needs
**every** cross-cutting piece [architecture §4](architecture.md#4-the-compatibility-boundary)
names — a JSON body with PascalCase fields and the escape table, a date, an absent property, an
unauthenticated route, a `404`, a `405` with its `Allow`, the response-time header, the content type
and the canonicalising middleware.

So 001's plan builds the edge, and 001's task list will look disproportionate to its four routes.
That is recorded here so it is reviewed as the roadmap's intent rather than as a feature that grew.
The alternative — a spec-less foundation phase — is not available: Principle III says code follows a
spec, and there is no spec for "the edge".

### 008 is one feature, not two

Transcoding is not a separate feature from playback negotiation. It is **the third branch of one
decision** — the ladder above, where the same negotiation answers direct play, remux or re-encode
depending on what the profile accepts — and splitting it would put one decision ladder in two
specifications and guarantee they drift.

The paragraph is quoted more often **read backwards** than forwards, and that is its more useful
direction. 011 and 012 both used it as the test for what belongs *in* a feature being assembled: a
finding belongs together with the others only if **what decides it is what decides them**. Bundling
by *when* a gap was found rather than by *what settles it* is how a feature becomes a list.

### What changed from the first run's order, and why

Two things, and both are order rather than scope.

**010 is not built here.** The conformance harness is specified in
[010](../specs/010-conformance-harness/spec.md) and already exists — in the source repository, as a
program that takes `--atrium <url>`. It is pointed at this server over HTTP and run as a black box,
never read. That is not a corner cut: **one instrument measuring two subjects is a better
experiment than two instruments**, because an asymmetry in the reports is then a property of the
servers and not of the harnesses. What this project does owe is the *consuming* half — L0, L1 and
L2 live in `conformance/` here ([architecture §8](architecture.md#8-testing-and-conformance)), and
the checked-in artefacts 010 produced are read here: `allowlist.yaml`, `request-cases.yaml`,
`named-comparisons.yaml`, and the reference's recorded fixture reading.

**011 and 012 move up beside 008.** The first run ordered 001 to 010 and then found 011, which is
how the subtitle promise came to be owned by nobody for nine features. That finding is in hand
before this run starts, so the two features that depend on 008 are built next to it. This costs
nothing and removes the one gap the first run is known to have left.

### Milestones, in terms of what actually works

| After | A user can |
|---|---|
| 002 | Reach a login screen and sign in. Nothing else. |
| 005 | Browse a real library in a real client. |
| 006 | Browse one that looks right. |
| 009 | Do everything except play something. |
| 011 | Play, with subtitles — the full v1 product. |

---

## Later, unscheduled

Not rejected — **not scheduled**, and deliberately without dates, because a date on this list would
be invented rather than estimated.

| Candidate | What it waits for |
|---|---|
| Trickplay and chapter-image generation | A background job framework over the whole library, which v1 has no other use for. |
| Hardware-accelerated transcoding | A per-machine capability surface, and a way to test it without the hardware in CI. |
| Subtitle burn-in | A text-rendering stack, and the second filter path it implies. |
| The WebSocket `/socket`, and SyncPlay on top of it | A push session model. Its absence is visible today only as clients polling. |
| Live TV and DVR | A separate product domain, and a second reference to measure against. |
| Quick Connect | A second authentication state machine, which is cheap only once the first is proven. |
| A second reference version | [reference-target](compatibility/reference-target.md) pins 10.11.11 by digest, and moving the pin has a procedure. Supporting two at once is a different project. |

Each of these would arrive the same way anything arrives: a specification first, a named consumer
for every endpoint, and a row in `surface.yaml` before a route exists.

---

## How this document changes

**A feature that changes scope updates this file in the same change**, the way documentation moves
with code under Principle III. Both features that have done so are visible above: 011 corrected the
feature-order table when it took a number the order had no row for, and 012 updated it again.

A change to §"Out of scope, and why" is also a change to
[api-surface §10](compatibility/api-surface-v1.md#10-deliberately-excluded-from-v1). The two are
prose halves of one decision, and a change to one of them alone is a drift, not an edit.
