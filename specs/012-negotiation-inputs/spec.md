---
feature: 012-negotiation-inputs
title: Negotiation inputs
status: Accepted
created: 2026-08-29
updated: 2026-08-30
amended: 2026-08-30 by **011 T9** — unusually, an amendment made from another feature's task, and recorded as such because the measurement was taken there and the answer belongs here. OQ-4 answered the case-insensitive binding for the **protocol** value alone; measuring the same question for a delivery method's vocabulary showed the leniency is the *binder's* and not that enumeration's, because a direct-play entry typed `video` rather than `Video` binds and is answered a direct play. OQ-4 is therefore widened rather than corrected — the original answer was narrow, not wrong — and now names the four further enumerated values this body carries, each of them refused here with `400` where the reference answers `200`. Nothing else in this document moves, and no code changes with it: 011 T9 fixed only the vocabulary 011 added, and the general fix stays 012's to make
depends_on: [003, 008]
---

# 012 — Negotiation inputs

> **This document describes WHAT and WHY only.** No technology names, no storage decisions.

## 1. Purpose

Make the negotiation answer only what it can deliver. `PlaybackInfo` has two inputs — what the
server knows about the file, and what the client said about itself — and on each of them v1 has a
branch the reference has not got: an input it cannot read is stepped over rather than **resolved**,
and the answer that comes back names a capability with no address behind it.

**Every claim in this document was measured at 012's own gate on 2026-08-29**, by four probes
against Jellyfin 10.11.11, and the measurements moved it: the reference does not merely decide what
an un-inspected source advertises, it **opens the file inside the request and keeps what it learns**
— so the feature's subject is reproducing an inspection, not choosing an advertisement. What did not
move is that both of v1's branches produce a `200` a client believes and cannot act on.

**Client behaviour unlocked:** a title that starts. Today a client that is handed one of these
answers either dead-ends with nothing to fetch, or takes an address of a shape it did not ask for.

**This is not new scope.** Both cases are inside 008's accepted surface, on 008's own route, and
neither is an 008 defect: 008's code does what 008's documents say, and a specification silent
about a case is not one that is wrong about it
([client-atrium-tvos §6](../../docs/compatibility/client-atrium-tvos.md#6-where-these-findings-go)).
What is new is that the silence has been traced from the client's side, by two independent client
authors, and that the reference has been read at both points and does something different at both.

## 2. Scope

**In scope**

- What the negotiation answers for a media source the server has **never opened** — including
  whether it opens it then and there, what it answers if opening fails, and what the same item's
  library listing says about it.
- What the negotiation does with a **delivery protocol it does not recognise** in the client's
  profile: a spelling it did not expect, and a value that names no protocol at all.
- The consistency rule underneath both: **every capability the answer advertises has an address
  the client can fetch**, and every address it hands out is of the shape the answer says it is.

**Out of scope**

- **Everything the negotiation already decides.** The ladder, its order, its ceilings, its reasons
  and its refusals are [008 §3.3](../008-playback-negotiation-and-delivery/spec.md#33-the-decision)
  and are not reopened. This feature is about the two inputs reaching that ladder, not about the
  ladder.
- **Delivery.** No route serves anything new here; the four `stream` routes, `/universal`, the
  playlists and the segments are 008's and unchanged.
- **Subtitles**, which are [011](../011-subtitle-delivery/spec.md)'s, including the subtitle
  properties a negotiated source carries.
- **Narrowing the session list.** The reference declares three query parameters on `GET /Sessions`
  that v1 declares none of; §2.1 argues that they belong to the feature that owns the route, which
  is 002 and not this one. Measured here (OQ-7) because the measurement session was here; recorded
  in [behaviours §2.25](../../docs/compatibility/behaviours.md) and specified there.
- **The initialisation segment that restarts production.** A defect decision under the written
  procedure, not a requirement — §2.1, OQ-8. Measured here, decided in
  [behaviours §3.14](../../docs/compatibility/behaviours.md), and the decision is *replicate*.
- **Per-user preferences of any kind**, and anything that would make an answer depend on state v1
  does not keep.
- **Growing the surface.** No endpoint enters v1 here and no row of
  [surface.yaml](../../docs/compatibility/surface.yaml) changes: every behaviour in this document
  is a change to what an already-listed route answers.

### 2.1 Why this is one feature, and why two of the four findings are not in it

**This feature was assembled from a list, which is the failure mode it has to argue its way out
of.** [011 §2.1](../011-subtitle-delivery/spec.md) took two findings out of eleven that two client
traces had recorded, handed the rest on at the size their own documents measured them, and named
four of those as *"one probe away from being specifiable and none specifiable before its probe …
a feature whose first act is a measurement session, taking its number on the day that session
runs"*. This is that number and this is that day. But 011's own warning applies to its own
handover: **four findings do not become one feature by having been handed on together**, any more
than eleven became one by sharing a date.

The test is the [roadmap's *"008 is one feature, not two"*](../../docs/roadmap.md#feature-order)
paragraph read backwards, exactly as 011 read it: transcoding stays inside 008 because it is *the
third branch of one decision*, and splitting it *"would put one decision ladder in two
specifications and guarantee they drift"*. Applied to an assembly rather than a split: **a finding
belongs here only if what decides it is what decides the others.**

Answering that honestly costs this feature half its list.

| Finding | What decides it | Verdict |
|---|---|---|
| [tvOS §4.1](../../docs/compatibility/client-atrium-tvos.md#41-a-source-with-no-stored-inspection-is-the-clients-documented-dead-end) + [music §5.1](../../docs/compatibility/client-embeat-mobile.md#51-a-source-with-no-stored-inspection-loses-the-music-clients-whole-negotiation) — a source with no stored inspection | What the negotiation answers when the **file side** of its input is unreadable | **Here** |
| [tvOS §4.6](../../docs/compatibility/client-atrium-tvos.md#46-two-spellings-of-hls-and-only-one-of-them-selects-hls) — `"Hls"` does not select HLS where `"hls"` does | What the negotiation answers when the **client side** of its input is unreadable | **Here** |
| [tvOS §4.4](../../docs/compatibility/client-atrium-tvos.md#44-get-sessions-takes-no-deviceid-and-the-client-sends-one) — the session list takes no `deviceId` | Which sessions a caller may see, and how that set may be narrowed | **002's** |
| [tvOS §4.5](../../docs/compatibility/client-atrium-tvos.md#45-the-fmp4-init-segment-restarts-the-encoder-which-is-the-defect-the-client-pre-warms-to-dodge) — the initialisation segment restarts production | Whether a reproduced defect stays reproduced | **A defect decision, not a feature** |

**The two that stay are one decision seen from its two sides.** `PlaybackInfo` takes exactly two
inputs — a media source and a device profile
([008 §3.3](../008-playback-negotiation-and-delivery/spec.md#33-the-decision) opens with that
sentence) — and v1 has a lenient branch on each. A source with no stored inspection is stepped
over, so it keeps flags that were never decided and gets no address. A profile whose protocol is
spelled in a case the comparison does not match falls through to the other branch, so it gets an
address of the wrong shape. **Both produce a `200` a client believes and cannot act on**, and both
exist because v1 chose to carry on where the reference does not: the reference **opens the file**
`[source: Emby.Server.Implementations/Library/MediaSourceManager.cs:170-189 @ v10.11.11]` and
**binds the protocol to a two-member enumeration**
`[source: MediaBrowser.Model/Dlna/TranscodingProfile.cs:77 @ v10.11.11]`,
`[source: Jellyfin.Data/Enums/MediaStreamProtocol.cs @ v10.11.11]`. Neither of those is a
tolerance. **Measured, neither is quite a two-answer question either** — the file read has a
failure case with an answer of its own, and the enumeration has a default, two ordinals and a
refusal `[probe: tools/probe_uninspected_source.py, Jellyfin 10.11.11, 2026-08-29]`,
`[probe: tools/probe_playback_info.py, Jellyfin 10.11.11, 2026-08-29]` — and that does not weaken
the grouping, it sharpens it: on both inputs the reference *resolves* the input before the ladder
sees it, where v1 carries on with it unresolved. Closing them is one rule — *the negotiation
answers nothing it cannot deliver* — and §2.2 is that rule stated positively.

**The session list is 002's, and the opening reading of this feature had the wrong owner.** It was
put here as *"a route feature 007 owns"*; it is not. `GetSessions` carries `feature: "002"` in
[surface.yaml](../../docs/compatibility/surface.yaml), is specified in
[002 §3.8](../002-authentication-users-and-sessions/spec.md#38-sessions) with its own acceptance
criterion and its own conformance row, and the video client's trace says so in as many words —
*"it is not a missing route — the row is in `surface.yaml`, tagged `video-client`, implemented at
002"*. 007 added what a session **row carries** while it is playing
([007 §3.6](../007-user-data-and-playstate/spec.md)); it did not touch what a session **request may
ask for**. And the sentence the parameter narrows is 002's own — *"the sessions the caller may see:
their own always, and all of them for an administrator"* — so the interesting half of the question
is 002's too: whether the reference applies the filter before or after that visibility rule, which
is a question about who may see whose device and not a question about playback. It is also **three
parameters, not one**: the reference declares `controllableByUserId`, `deviceId` and
`activeWithinSeconds` `[source: Jellyfin.Api/Controllers/SessionController.cs:52-59 @ v10.11.11]`,
and the third of them interacts with the session reap
[007 §3.8](../007-user-data-and-playstate/spec.md) rather than with anything here. Measured at this
gate (OQ-7) because a measurement session is cheap to extend and expensive to convene; specified in
002, in the change that adds them. **The measurement is in**, and the security half of it is not a
convenience at all: `deviceId` is applied to the whole session list *before* the visibility rule,
so a non-administrator naming somebody else's device gets an empty `200` rather than a refusal —
while `controllableByUserId` naming anybody but themselves is refused outright with `403`
`[probe: tools/probe_session_filters.py, Jellyfin 10.11.11, 2026-08-29]`. Two parameters that read
as siblings answer a trespasser two different ways, which is exactly the kind of sentence that
belongs in the specification that owns the visibility rule.

**The initialisation segment is a decision, and a feature is the wrong container for one.** The
restart is faithful reproduction — the reference's first branch is *"starting transcoding because
fmp4 init file is being requested"*, taken before it has looked at what is running
`[source: Jellyfin.Api/Controllers/DynamicHlsController.cs:1501-1505 @ v10.11.11]` — and the video
client pre-warms its session to dodge it `[client-contract: 2026-08-29, §3]`. So the question is
not *what should this server do*, it is *should this server stop doing what the reference does*,
which is [behaviours §3.0](../../docs/compatibility/behaviours.md#30-how-the-decision-is-made) and
belongs in the behaviours document with an argument attached. Its input is a cost measurement,
which this gate took (OQ-8), and its output is a `§3` entry, which this gate wrote:
[behaviours §3.14](../../docs/compatibility/behaviours.md). **The reading it inherited was
incomplete** — the restart branch is third, not first, and two file-existence checks stand in front
of it — so the measured cost for a player that reads a playlist and follows it is nothing at all,
and the procedure settles on **replicate**. Making
it an acceptance criterion here would smuggle a Principle V decision in as a requirement — the
mistake 011 refused to make twice, with its own two Principle I questions.

**Neither handover loses the work**, which is the whole point of writing the sizing down rather
than absorbing it: 011 handed the ordering finding to 005 with the test that answers it named, and
both of these are handed on with the probe that answers them named, in §7.

### 2.2 What is left, and why it is one thing

**One rule, stated positively: a negotiation answer is actionable.** Every capability it
advertises has an address behind it, and every address it hands out is of the shape the answer
says it is. The two findings are the two ways v1 breaks that rule, and they break it in the same
direction — by carrying on with an input the ladder cannot use, instead of resolving it or
refusing.

Read from the client's side, the two failures look nothing alike, and that is the argument rather
than an objection to it. The video client refuses direct play, looks for the address it was
promised, finds none and stops
([client-atrium-tvos §4.1](../../docs/compatibility/client-atrium-tvos.md#41-a-source-with-no-stored-inspection-is-the-clients-documented-dead-end));
the music client never asks, plays the bytes and quietly loses four features it computes off the
streams
([client-embeat-mobile §5.1](../../docs/compatibility/client-embeat-mobile.md#51-a-source-with-no-stored-inspection-loses-the-music-clients-whole-negotiation));
and the video client with a differently-cased profile takes an address that works, for a shape it
has designed itself out of
([client-atrium-tvos §4.6](../../docs/compatibility/client-atrium-tvos.md#46-two-spellings-of-hls-and-only-one-of-them-selects-hls)).
**Three symptoms, two lenient branches, one rule.** A feature defined by the rule catches all
three; a feature defined by any one symptom catches one and leaves the others to be found by the
next client author.

**The measurement moved one of the three symptoms and did not move the rule.** The music client's
losses turn out to be parity on the read side — a stock reference answers a listing the same way
(§3.1, OQ-3) — so what closes them is not a change to the listing but the negotiation's *write*:
the reference keeps what an on-demand inspection learns, and every later listing of that item
carries it (OQ-9). That is a better argument for one feature than the one this section opened with,
because it is the same mechanism rather than the same shape: **one write, seen by two readers.**
And the rule needs one word of care that the measurement supplied — the reference guarantees that
an advertised capability *has* an address, not that the address *answers* (§3.2).

## 3. Behaviour

### 3.1 What a client sees today, and what the reference answers

Stated as the starting point, because every criterion below is a change to one of these. The left
column is observable from a running server on `main` at 2026-08-29, and the file-level evidence for
it lives in the two client traces rather than here
([client-atrium-tvos §4.1 and §4.6](../../docs/compatibility/client-atrium-tvos.md#4-the-eight-findings),
[client-embeat-mobile §5.1](../../docs/compatibility/client-embeat-mobile.md#51-a-source-with-no-stored-inspection-loses-the-music-clients-whole-negotiation)).
The right column is this gate's, and **one of its rows is the reason the feature is half the size
it looked**.

| What a client does | What Atrium answers today | What the reference answers |
|---|---|---|
| Negotiates a video item nothing has opened | A source carrying `Id`, a `Container` inferred from its path and `Size`; `RunTimeTicks` absent, `Bitrate` absent, `MediaStreams` empty; the three capability flags all **`true`**; and **no `TranscodingUrl`** | It opens the file. If it can be read, a fully annotated source and a working address; if it cannot, the same empty annotation as Atrium's but with the flags **decided** — `false`/`false`/`true` against a profile that plays neither — and an address |
| Negotiates the same item again with direct play and direct stream both switched off | The same answer, unchanged, with the same three flags and still no address | A different answer, as for any other source: the switches reach the ladder because nothing was skipped |
| Reads that item's media source from a library listing | The same empty shape, on every route that offers a source | **The same empty shape, on every route that offers a source** — flags `true`, no address, no streams. Parity, exactly |
| Reads that listing *after* something negotiated the item | The same empty shape, for ever, until a rescan | The streams, the runtime, the bitrate and a corrected `Size`: what the negotiation learned was written down |
| Negotiates an **audio** item nothing has opened, with a profile | The same empty source, `200` | `400`, `text/plain` — the whole request is refused, because the ladder's first act for an audio item is to choose an audio stream and there is none |
| Posts a profile whose transcoding entries name the protocol as `Hls` or `HLS` | A `TranscodingUrl` to the **progressive** delivery route, and `TranscodingSubProtocol` echoing the client's own spelling back — an answer that names one shape and addresses another | An HLS address, and `TranscodingSubProtocol: "hls"` — the enumeration's spelling, not the client's |
| Posts a profile whose transcoding entries name a protocol that is neither spelling | The same progressive answer. Nothing is refused and nothing is reported | It depends on the value, and on three different things: `400` problem details for a word it cannot bind, the declared default for an empty string or a missing property, and the **ordinal** for a number |

`[probe: tools/probe_uninspected_source.py, Jellyfin 10.11.11, 2026-08-29]`,
`[probe: tools/probe_playback_info.py, Jellyfin 10.11.11, 2026-08-29]`

**Row three is the one that walks a claim back.** Both client traces call this *"one root cause,
two clients"*, and the music client's half was reported as a gap. It is not: on a listing the
reference answers exactly what Atrium answers, so the four things that client computes off the
streams — high-resolution detection, casting to a rate-capped renderer, the gapless trim and the
truncation guard — are lost against a stock reference too. **Row four is what saves the sentence,
and it saves it through a different mechanism than the traces proposed**: the reference's listing
recovers, not because the listing path probes but because *some other request* did and the result
was kept. On a reference server anything that negotiates that item heals every later listing of it;
on Atrium nothing does.

**Rows one and two are still a gap, and still the client-visible one.** The first says the failure
is not self-correcting: a client that comes back with the switches set to say *"I cannot direct-play
this"* is answered identically, because the branch that would have read the switches is the branch
that was skipped.

### 3.2 A media source the server has never opened

**The reference does not answer this question while the file is readable, because it does not let
itself be asked it.** On a negotiation it walks the item's sources and, when the first of them
carries no video stream for a video item or no audio stream for an audio item, it refreshes the
item with probing enabled and re-reads the sources before any profile is applied
`[source: Emby.Server.Implementations/Library/MediaSourceManager.cs:170-189 @ v10.11.11]`,
`[source: Jellyfin.Api/Helpers/MediaInfoHelper.cs:87-110 @ v10.11.11]`. Measured on the wire, that
is not a code path that might not run: **it runs, it completes inside the request, and the client
gets the whole annotation back**.

The fixture that proves it has to be built by subtraction, because a library cannot supply one: the
scan that creates an item is the scan that probes it, so a readable file is never in this state.
A library of deliberately unreadable files, one of which becomes readable behind the server's back,
is what the measurement needs `[probe: tools/probe_uninspected_source.py, Jellyfin 10.11.11,
2026-08-29]`.

| What was asked | What the reference answered |
|---|---|
| Listing an item whose file could not be read | `Container` from the path, `Size`, no `RunTimeTicks`, no `Bitrate`, `MediaStreams: []`, the three flags `true`, no address — **identical on `/Items`, `/Items/{itemId}` and `/Items/Latest`**, and identical to Atrium's answer |
| Negotiating that item with a profile that plays neither its container nor its codec | `200`; the same empty annotation, and flags **decided**: `SupportsDirectPlay: false`, `SupportsDirectStream: false`, `SupportsTranscoding: true`, with a `TranscodingUrl` |
| Negotiating an item whose bytes became valid after the scan | `200`, fully annotated — two streams, a runtime, a bitrate and a `Size` corrected from 4 096 to the file's real length — in **0.20 s**, against 0.01 s for an item already annotated |
| Listing that same item immediately afterwards | Everything the negotiation learned. **The inspection is kept**, not spent on one answer |
| Negotiating the unreadable item again, three times | 0.18 s, 0.19 s, 0.20 s. A file that can never be resolved pays the probe on **every** negotiation, for ever |
| Negotiating an **audio** item with no audio stream, with a profile | `400`, `text/plain`. Without a profile, `200` and the empty source |
| Negotiating an item whose file was deleted after the scan | `200`, fully annotated from what the scan stored, with a working-looking address. Nothing notices |

**Two of those rows retire a claim this feature was built on.** The listing row makes the music
client's four losses **parity**: a stock reference answers a listing the same way Atrium does, so
nothing is being lost against the reference by reading one. And the row beneath it is why the
feature is nevertheless one feature and not half of one — the reference's listing is not empty
*for long*, because any negotiation of that item writes the inspection down and every later listing
carries it. The traces' *"one root cause, two clients"* survives, with a different cure than either
trace proposed: what closes the music client's symptom is not a change to the listing path but the
negotiation's write.

**And the reference's own answer for an unreadable file is not one this feature can copy whole.**
The address it hands out for a source with no `RunTimeTicks` names `live.m3u8` rather than
`main.m3u8` — a source with no runtime is addressed as an infinite stream — and that playlist
answers **`500`**. So the reference does satisfy *"every capability has an address"* and does not
satisfy *"every address can be fetched"*, and the two are not the same rule. That is a defect of
the reference in a path v1 does not yet have, recorded and deferred rather than argued here
([behaviours §3.13](../../docs/compatibility/behaviours.md)).

**What v1 does**, now that the measurements are in, and the candidates were not equivalent:

| Candidate | What a client sees | What it costs | Verdict |
|---|---|---|---|
| Open the file during the negotiation, then answer normally | The reference's answer | One inspection inside a request that has a client waiting: **0.2 s measured**, paid once for a file that can be read and on every negotiation for one that cannot | **Chosen** |
| Answer the source with every capability flag `false` | The refusal [008 §3.3](../008-playback-negotiation-and-delivery/spec.md#33-the-decision) rung 4 already specifies | A source the server could have played, refused because nobody looked — and a delta from a reference that would have played it | Rejected |
| Keep today's answer and record the gap | Unchanged | The two symptoms of §3.1, unchanged | Rejected |

The first is the only candidate that is *parity*, and the measurement removed the reason it was not
chosen at the opening: its cost is a fifth of a second, it is bounded by one file read, and it is
what makes the music client's half close as well as the video client's.

**A truncated file is not one of these.** The first kibibyte of a 145 KiB Matroska probes cleanly
and answers a full annotation with a `Size` of 1 024 — the header is at the front, so truncation
does not make a file un-inspectable. What does: a zero-length file, and bytes that are not the
container the extension claims. §3.4's rows are written from what was measured rather than from
what sounded like the same class of accident.

**When it happens** is not exotic. A file whose inspection failed; a file the server could not read
at the moment the scan reached it; a file on a mount that was down; and — the widest of them — a
server on which nothing can inspect at all, where every item in every library is in this state at
once. The closing mechanism the accepted-gap record named was *"a rescan"*
([behaviours §5](../../docs/compatibility/behaviours.md#5-accepted-gaps-in-v1)), which is true and
is not something a client can ask for; that row's wording moves with this feature, because the
reference's closing mechanism turns out to be the negotiation itself.

**Whatever is decided, the flags stop being defaults.** The three capability flags are an *answer*
in the reference — computed per negotiation from the profile, with `SupportsDirectStream` mirroring
`SupportsDirectPlay` ([behaviours §2.22](../../docs/compatibility/behaviours.md#222-supportsdirectstream-mirrors-supportsdirectplay))
and `SupportsTranscoding` following the profile rather than the outcome
([008 T14](../008-playback-negotiation-and-delivery/tasks.md)) — and the measurement above shows
them decided even for a source with nothing in it. Today, on this path, all three are whatever they
were initialised to and nothing decided them.

### 3.3 A delivery protocol the negotiation does not recognise

A client's profile says, per transcoding entry, how it wants a produced stream delivered. Two
spellings are meaningful to the reference and they are **lower-case by declaration**: the
enumeration's members are `http` and `hls`, spelled that way deliberately and carrying a comment
saying so `[source: Jellyfin.Data/Enums/MediaStreamProtocol.cs @ v10.11.11]`. The property is
bound to that enumeration rather than to free text, with `http` as its default
`[source: MediaBrowser.Model/Dlna/TranscodingProfile.cs:77 @ v10.11.11]`.

**Eighteen spellings were posted to the reference on one item and one profile**, and they answer
four different ways rather than the two the opening reading predicted
`[probe: tools/probe_playback_info.py, Jellyfin 10.11.11, 2026-08-29]`:

| What the profile said | What came back |
|---|---|
| `hls`, `Hls`, `HLS`, `hLs` | An HLS address, and `TranscodingSubProtocol: "hls"` |
| `http`, `Http`, `HTTP` | A progressive address, and `TranscodingSubProtocol: "http"` |
| the property absent, `null`, or `""` | The declared default: a progressive address, `"http"` |
| `1` and `"1"`; `0` and `"0"` | The **ordinal**: HLS for one, progressive for zero |
| `2` and `"2"` — an ordinal no member has | `200`, a progressive address, and `TranscodingSubProtocol: 2` — **a number in a field the enumeration spells as a word** |
| `dash`, `" "`, `true` | `400`, RFC 9457 problem details, `errors` keyed on `$.DeviceProfile.TranscodingProfiles[0].Protocol` |

**Three findings follow, and one of them was not asked for.**

1. **A differently-cased spelling binds, and this server's comparison does not.** `Hls` and `HLS`
   select HLS there and progressive here — a delta in the direction Principle I has least tolerance
   for, because the client is *correct* and this server is the one that misreads it.
2. **A value that binds to nothing refuses the whole body**, which is the opposite of what this
   server answers and of the lenience
   [behaviours §1.12](../../docs/compatibility/behaviours.md#112-an-unrecognised-query-value-is-ignored-not-rejected)
   records for a **query** value — and the same shape 008 §3.2 measured for an unrecognised token
   inside this body. But *"a value that is neither spelling"* is not one class: an empty string and
   an absent property take the default, and numbers bind by ordinal. A criterion written over
   *"an empty string, an unknown word and a numeric value"* would have asserted one answer where
   the reference gives three.
3. **The answer echoes the enumeration's spelling, not the profile's.** A profile saying `Hls` is
   answered `"hls"`. So this server's contradiction is **two** defects rather than one: it takes
   the wrong branch, *and* it writes the client's own spelling into `TranscodingSubProtocol` beside
   an address that does not match it. The reference's only self-contradiction of that shape is the
   out-of-range ordinal, where it answers a progressive address beside a sub-protocol of `2`.

`/universal` does not have the problem, and the contrast is what makes it a defect rather than a
reading: the audio route normalises the same value before comparing it, under a note calling that
*"measured, not lenience"*
([client-atrium-tvos §4.6](../../docs/compatibility/client-atrium-tvos.md#46-two-spellings-of-hls-and-only-one-of-them-selects-hls)).
Two routes of this server read one value by two rules, and the measurement says which of them is
parity: the audio one.

### 3.4 Error paths

Every row here was a `200` before this feature and every one of them is now measured, so the table
states answers rather than questions. **No row invents a refusal**: each is what the reference
answered to the same request `[probe: tools/probe_uninspected_source.py, Jellyfin 10.11.11,
2026-08-29]`, `[probe: tools/probe_playback_info.py, Jellyfin 10.11.11, 2026-08-29]`.

| Condition | What the reference answers | What v1 answers today |
|---|---|---|
| Negotiating a **video** item whose only file cannot be opened | `200`, an un-annotated source with the flags decided and a `TranscodingUrl` — whose HLS playlist then answers `500`, because a source with no runtime is addressed as a live stream | `200`, every capability advertised, no address |
| Negotiating an **audio** item whose only file cannot be opened, with a profile | `400`, `text/plain` — the whole request, not the source | `200`, every capability advertised, no address |
| The same audio item with **no** profile in the body | `200`, the un-annotated source | The same |
| Negotiating an item whose file has gone from disk since the scan | `200`, fully annotated from what the scan stored, with an address. Nothing looks at the file | The same, and for the same reason |
| A profile naming a delivery protocol that binds to no member — a word, whitespace, a boolean | `400`, problem details naming `$.DeviceProfile.TranscodingProfiles[0].Protocol` | `200`, a progressive address |
| A profile naming a delivery protocol in an unexpected case | `200` and an HLS address | `200`, a progressive address and a contradicting `TranscodingSubProtocol` |
| A profile whose protocol is an empty string, absent, or an ordinal | `200`, the default or the ordinal's member — never a refusal | `200`, a progressive address |
| Every existing refusal on this route | Unchanged — [008 §3.2](../008-playback-negotiation-and-delivery/spec.md#32-post-itemsitemidplaybackinfo--getpostedplaybackinfo)'s table stands | Unchanged |

**The `400` on the audio path is the row worth reading twice**, because it is the one place where
this feature adds a refusal — and it adds it by *reproduction*, not by invention. The reference's
ladder chooses an audio stream before it does anything else for an audio item, and an item with no
audio stream has none to choose; the refusal is what falling off that step looks like from outside.
[behaviours §3.0.2](../../docs/compatibility/behaviours.md#302-what-is-never-acceptable) forbids
inventing a third behaviour, and none of these rows does: every one of them was answered by a real
server to a request a probe in this repository sends.

## 4. Data the feature owns

| State | Observable as | Lifetime |
|---|---|---|
| Whether a file has been opened and what it holds | The stream properties, runtime and bitrate of a media source, on a listing and on a negotiation | Until the file changes |
| The result of an inspection taken **during a negotiation** | Whether the **next listing** of the same item carries what that negotiation learned | The same lifetime: it is the same state, written from a second place |

**No new state.** The first row is 003's and 008's, already owned and already observable; this
feature changes *when* it is written, not what it is. The second row is not a second store either,
and that is OQ-9's answer rather than an assumption: the reference's on-demand inspection is
**kept**, so the next listing of a healed item carries its streams, its runtime, its bitrate and a
corrected `Size` `[probe: tools/probe_uninspected_source.py, Jellyfin 10.11.11, 2026-08-29]`. It is
the reason this feature has one symptom to close rather than two — the write, not the read, is what
both clients are downstream of.

## 5. Acceptance criteria

**Written against measurements, not around them.** The provisional set this document opened with
held for seven of its eight criteria; the eighth said something the reference does not do, and the
fifth named one answer where the reference gives three. Both are corrected below rather than
softened.

1. A negotiation for an item nothing has opened answers a source whose three capability flags were
   **decided** — by the profile and the ladder, exactly as they are for every other source — rather
   than left at whatever they were initialised to. For a source that could not be read, against a
   profile that plays neither its container nor its codec, that is `SupportsDirectPlay: false`,
   `SupportsDirectStream: false`, `SupportsTranscoding: true`.
2. A negotiation for an item whose file **can** be read answers it fully annotated — streams,
   runtime, bitrate, a `Size` taken from the file — whether or not anything had opened it before,
   and the request that did so is the request that answers. This is the criterion
   [client-atrium-tvos §4.1](../../docs/compatibility/client-atrium-tvos.md#41-a-source-with-no-stored-inspection-is-the-clients-documented-dead-end)
   asks for, in the form the measurement showed the reference to satisfy.
3. What that negotiation learned is **written down**: the next listing of the same item carries it,
   with no scan in between. This is the criterion the music client's four losses turn on, and §3.1
   shows it is the only one of them that is not already parity.
4. A negotiation whose answer advertises a capability the client cannot exercise directly carries
   an address for it — **including** for a source that could not be read, which is what the
   reference does. Whether that address can then be fetched is
   [008 §3.4](../008-playback-negotiation-and-delivery/spec.md)'s delivery path and a recorded
   reference defect ([behaviours §3.13](../../docs/compatibility/behaviours.md)); this criterion is
   about the negotiation answering with an address, not about the address succeeding.
5. Repeating that negotiation with direct play and direct stream switched off answers something
   **different** from the first answer — the property §3.1's second row shows is missing today, and
   the one a client relies on when it comes back for a second opinion.
6. A negotiation for an **audio** item with no audio stream, carrying a device profile, refuses the
   whole request with `400`; the same request with no profile answers `200` and the un-annotated
   source.
7. A profile that names the delivery protocol in any case answers the same address; and the
   answer's stated sub-protocol and the shape of the address it hands over agree, with the
   sub-protocol spelled the way the reference spells it rather than the way the profile did.
8. A profile whose protocol names no member answers by class, not by one rule: `400` with problem
   details naming the property's JSON path for a value that cannot bind at all; the declared
   default for an absent property, a null and an empty string; and the ordinal's member for `0` and
   `1`.
9. Nothing in this feature changes what a negotiation answers for an item that **has** been
   opened, for any profile: the ladder, the reasons, the addresses and the flags of
   [008 §3.3](../008-playback-negotiation-and-delivery/spec.md#33-the-decision) are unchanged, and
   the existing conformance suite is the proof.
10. Nothing in this feature changes what a **listing** answers for any item, opened or not. The
    listing's answer for a never-opened source is already the reference's answer, and the flags it
    carries there stay `true`: they are not a negotiation and nothing decides them.

**AC-10 is the criterion this gate added, and it is a prohibition rather than a requirement.** The
draft's eighth criterion asked the listing and the negotiation to *agree*; measured, they do not
agree on the reference either, and making them agree here would be a delta invented to satisfy a
symmetry nobody has.

## 6. Conformance

| Endpoint | Level | How it is proven |
|---|---|---|
| `POST /Items/{itemId}/PlaybackInfo`, the never-opened source | **L3** | Golden per candidate class — a source that can be read and one that cannot, each with and without a profile and with the two switches on and off, plus the audio `400` — and differential against an item the reference has likewise never opened |
| `POST /Items/{itemId}/PlaybackInfo`, the profile's delivery protocol | **L3** | Golden per spelling class, and the classes are §3.3's four: the canonical pair, three altered cases, the three that take the default, the two ordinals and the out-of-range one, and the three that refuse. Plus differential, which is the only thing that can settle a refusal against a fall-through |
| Media sources on the listing routes | **L2** | Fixture with one file the world has deliberately not opened, asserted on every route that offers a source — and asserted to be **unchanged** by a negotiation of a different item |
| The inspection an on-demand probe writes | **L2** | The same fixture, read twice: a listing, a negotiation, the same listing again |
| Error paths | **L2** | Table-driven over §3.4, per case |

**L3 on both negotiation rows, and the reason is the same as 008's.** v1 requires L3 for the
playback paths ([conformance §1](../../docs/compatibility/conformance.md)), and these two are the
places where this server and the reference can agree on every field of a response and still send
a client somewhere different — which is exactly what a differential is for and what a golden test
cannot catch on its own.

**The fixture is the interesting part, and it is a subtraction — which this gate had to build
before it could measure anything.** 008's generated media are inspected by the world that builds
them; this feature needs media that are **not**, and a library cannot supply them, because the scan
that creates an item is the scan that probes it. What produces the state is a file the inspection
*fails* on: a zero-length one, or bytes that are not the container their extension claims. A file
truncated to its first kibibyte is **not** one of them — it probes cleanly. `tools/probe_uninspected_source.py`
builds that fixture, and the conformance world can build the same one the same way.
[007's task gate](../007-user-data-and-playstate/tasks.md#what-the-gate-changed) and
[006's](../006-images/tasks.md#what-the-gate-changed) both found a criterion with no world to
prove it in; this one was noticed at the spec and answered at the gate.

## 7. Open questions and what measuring them did

**All nine were measured on 2026-08-29**, at this feature's own gate, by four probes: two written
for it (`tools/probe_uninspected_source.py`, `tools/probe_session_filters.py`) and two extended
(`tools/probe_playback_info.py`, `tools/probe_transcode_session.py`). Every source line this
document cites was read at `v10.11.11`; every wire claim now carries a probe. A reading predicts,
it does not measure (Principle II), and this gate is the third in a row where that distinction paid.

| # | Question | What it turned out to be |
|---|---|---|
| OQ-1 | Does a negotiation for an item the reference has never probed come back **fully annotated**, and how long does a client wait? | **Yes, inside the request.** A file that became readable behind the server's back is fully annotated by the first negotiation that asks for it — two streams, a runtime, a bitrate and a corrected `Size` — in **0.20 s**, against 0.01 s for an item already annotated. A file that can never be read pays the same probe on **every** negotiation, measured at 0.18–0.20 s three times running. The feasibility answer is that the cost is a fifth of a second and it is bounded by one file read |
| OQ-2 | What does the reference answer when the on-demand probe **fails**? | **It depends on the media type, which nothing had predicted.** A video item answers `200` with the flags decided and an address. An **audio** item carrying a device profile answers **`400`, `text/plain`** — the whole request refused, because the ladder picks an audio stream before anything else and there is none to pick. With no profile the same audio item answers `200`. And a file *deleted* after the scan is not this case at all: the stored streams are still there, so nothing re-reads the file and the answer is a normal, fully annotated `200` |
| OQ-3 | What does a **listing** answer for the same never-probed item? | **Exactly what Atrium answers** — `Container` from the path, `Size`, no runtime, no bitrate, no streams, three flags `true`, no address — identically on `/Items`, `/Items/{itemId}` and `/Items/Latest`. **The music client's four losses are parity**, and the claim that they were a gap is withdrawn. What is not parity is §3.1's fourth row: the reference's listing stops being empty as soon as anything negotiates the item |
| OQ-4 | Does `Hls` or `HLS` select HLS on the reference? | **Yes**, and so does `hLs`. The binding is case-insensitive, and this server's comparison is not. The delta is in the direction Principle I has least tolerance for. **And the leniency is a property of the binder rather than of that one enumeration**, widened here on 2026-08-30 from a measurement taken while 011 was implementing its own subtitle vocabulary: a direct-play entry whose *type* is written `video` rather than `Video` binds too, and is answered a direct play. So the answer reaches **every** enumerated value this body carries, which is four more than this row named: the `Type` of a direct-play or transcoding entry, the `Type` of a codec profile, and a profile condition's `Condition` and its `Property`. Each of the four is refused here with `400` where the reference answers `200` `[probe: tools/probe_subtitle_negotiation.py, Jellyfin 10.11.11, 2026-08-30]`. Closing it is one change to the shared request-model behaviour rather than five, which is what makes the width worth stating before anybody writes the narrow fix |
| OQ-5 | What does the reference answer for a protocol value that names neither member? | **Three different things.** A word, whitespace or a boolean refuses the whole body with `400` problem details naming `$.DeviceProfile.TranscodingProfiles[0].Protocol`. An empty string, a `null` and an absent property take the declared default. A **number binds by ordinal** — and `2`, which no member has, is accepted and echoed back as the number `2` in a field the enumeration spells as a word. The draft's two candidates were both right and neither was complete |
| OQ-6 | Does the answer echo the profile's spelling or the enumeration's? | **The enumeration's.** `Hls` in, `"hls"` out. So this server's contradiction is two defects rather than one: the wrong branch, and the client's own spelling written into `TranscodingSubProtocol` beside an address that does not match it |
| OQ-7 | *(002's)* What do `GET /Sessions`' three parameters narrow, and in what order? | **Three different kinds of narrowing.** `deviceId` matches case-insensitively, is applied **before** the visibility rule and is ignored when empty. `activeWithinSeconds` is applied last and is ignored at zero and below. `controllableByUserId` is not a filter but a second visibility rule: it keeps only sessions with a live control channel — which no request-response client ever has — and answers **`403`** when a non-administrator names anybody but themselves, where naming another user's *device* answers an empty `200`. Recorded in [behaviours §2.25](../../docs/compatibility/behaviours.md); specified in 002 |
| OQ-8 | *(a defect decision)* What does the initialisation-segment restart cost? | **Nothing, in either order a client uses.** The restart branch is third, not first: two file-existence checks stand in front of it, and an fMP4 transcode writes the initialisation segment before it writes any segment. So a session that has produced anything already has the file the branch tests for — measured at 0.03 s against 0.69 s for the same request on an empty directory, with the segments already produced still answering afterwards. The branch is reached only where there is nothing to discard. Decided under [behaviours §3.0](../../docs/compatibility/behaviours.md#30-how-the-decision-is-made) and recorded at [§3.14](../../docs/compatibility/behaviours.md): **replicate** |
| OQ-9 | Does the reference **keep** what an on-demand probe learns? | **Yes.** The next listing carries the streams, the runtime, the bitrate and the corrected `Size`, with no scan in between. It is what makes §2.2's "one rule, two faces" true — through the write rather than through the read, which is not the mechanism either client trace proposed |

### 7.1 What measuring corrected in this document

Recorded because the corrections came from probes, which is the method
[003's gate](../003-library-configuration-and-scanning/tasks.md#what-the-gate-changed) named and
every gate since has paid for.

1. **The reference has an un-inspected source after all — an un-*inspectable* one.** The opening
   reading said the state does not exist there, because the negotiation probes on demand. Half
   right: a *readable* file never stays un-inspected, and an unreadable one is in that state for
   ever and is answered anyway. So the feature's subject is both things at once — reproduce the
   inspection, *and* decide what the answer looks like when it fails — and the second half is where
   the `400` and the `500` live.
2. **The music client's half is parity, and the claim that it was a gap is withdrawn.** A listing
   on a stock reference answers the same empty source Atrium answers. This walks back a finding
   already reported, which is what a measurement is for.
3. **"One root cause, two clients" survives, with the wrong cure named.** Both traces put the
   music client's fix on the listing path. It is on the negotiation's *write*: the reference keeps
   what it learns, so anything that negotiates an item heals every later listing of it.
4. **A truncated file is not an un-inspectable one.** The first kibibyte of a Matroska probes
   cleanly and answers a full annotation with a `Size` of 1 024. §3.4's row for it was written from
   a plausible-sounding class rather than from a measured one, and is gone.
5. **The protocol question had two candidates and needed four.** An empty string is not an unknown
   word, and a number is neither.
6. **The initialisation-segment reading was incomplete, and the contract's framing with it.** The
   branch that restarts is guarded by a file-existence test the same session has already satisfied.
7. **The session list is 002's route, not 007's, and it is three parameters rather than one.** Both
   corrections were made by opening files at the spec's own opening; the measurement added the half
   that matters, which is the `403`.

### 7.2 The dependencies outside this document

1. **No surface change.** Nothing enters [surface.yaml](../../docs/compatibility/surface.yaml) and
   nothing leaves it; no measurement here needed a route this project does not already have. Every
   behaviour in this document is a change to what an already-listed route answers.
2. **One accepted-gap row is rewritten, not merely pointed at.** The never-opened-source row of
   [behaviours §5](../../docs/compatibility/behaviours.md#5-accepted-gaps-in-v1) named its closing
   mechanism as *"a decision about what an un-inspected source should advertise"*. That was the
   wrong shape: the reference resolves the state rather than describing it, so the mechanism is an
   inspection taken during the negotiation and written down. The row moves with this gate.
3. **Two records are owed to documents this feature does not own, and both are paid**: 002 gains
   the three session parameters and the `403` (OQ-7, recorded in behaviours §2.25 until 002's own
   change lands), and the behaviours document gains the initialisation-segment decision (OQ-8,
   §3.14) and three entries nobody asked for — §2.23, the two read paths and the write between
   them; §2.24, the protocol enumeration; and §3.13, the live
   playlist an un-inspectable source is addressed to.
4. **The two client traces are a floor, not a ceiling**, and they say so: absence from one means
   *not measured*, never *not needed*. This gate withdrew one of their findings and kept the other
   three. Nothing in CI notices when a trace goes stale.

## 8. References

- [docs/compatibility/client-atrium-tvos.md](../../docs/compatibility/client-atrium-tvos.md) — the
  video client traced against merged code: §4.1 and §4.6 are this feature, §4.4 is handed to 002
  and §4.5 is handed to the behaviours document, and §6 is the grouping table §2.1 re-reads
- [docs/compatibility/client-embeat-mobile.md](../../docs/compatibility/client-embeat-mobile.md) —
  the music client's counterpart: §5.1 is the second face of §3.2, and its four losses are what
  OQ-3 and OQ-9 decide the status of
- [011 §2.1](../011-subtitle-delivery/spec.md) — the handover this feature is the other side of,
  and the scoping test it applies to itself
- [008 §3.1, §3.2, §3.3](../008-playback-negotiation-and-delivery/spec.md) — the media source
  shape, the negotiation this feature is inside, and the ladder it does not reopen
- [002 §3.8](../002-authentication-users-and-sessions/spec.md#38-sessions) — the session list and the
  visibility rule OQ-7 is a narrowing of
- [003 §3.2, §3.8](../003-library-configuration-and-scanning/spec.md) — what a scan opens, and the
  change detection that decides when a file is opened again
- [docs/compatibility/behaviours.md §1.12, §2.21, §2.22, §3.0, §5](../../docs/compatibility/behaviours.md)
  — the lenient-value rule this feature has to distinguish itself from, the negotiation-inert
  permissions, the mirrored flag, the defect procedure, and the accepted gap that opens §3.2
- [docs/roadmap.md](../../docs/roadmap.md#feature-order) — the *"008 is one feature, not two"*
  paragraph, read backwards, and the feature-order table updated in the same change
- `[spec: GetPostedPlaybackInfo, GetSessions, MediaSourceInfo, MediaStream, DeviceProfile,
  TranscodingProfile, PlaybackInfoDto, PlaybackInfoResponse]`
- The four probes this document's claims rest on, all run against Jellyfin 10.11.11 on
  2026-08-29: `tools/probe_uninspected_source.py` (OQ-1, OQ-2, OQ-3, OQ-9),
  `tools/probe_playback_info.py` (OQ-4, OQ-5, OQ-6), `tools/probe_session_filters.py` (OQ-7),
  `tools/probe_transcode_session.py` (OQ-8)
- The reference's own paths, read at the opening and **since measured**:
  `[source: Emby.Server.Implementations/Library/MediaSourceManager.cs:170-215, 348 @ v10.11.11]`,
  `[source: Jellyfin.Api/Helpers/MediaInfoHelper.cs:87-117, 251-268 @ v10.11.11]`,
  `[source: Jellyfin.Api/Controllers/MediaInfoController.cs:119-215 @ v10.11.11]`,
  `[source: Emby.Server.Implementations/Dto/DtoService.cs:261 @ v10.11.11]`,
  `[source: MediaBrowser.Model/Dlna/TranscodingProfile.cs:77 @ v10.11.11]`,
  `[source: Jellyfin.Data/Enums/MediaStreamProtocol.cs @ v10.11.11]`,
  `[source: Jellyfin.Api/Controllers/SessionController.cs:44-70 @ v10.11.11]`,
  `[source: Jellyfin.Api/Controllers/DynamicHlsController.cs:1480-1530 @ v10.11.11]`
