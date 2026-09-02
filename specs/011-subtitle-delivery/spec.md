---
feature: 011-subtitle-delivery
title: Subtitle delivery
status: Implemented
created: 2026-08-29
updated: 2026-08-31
implemented: 2026-08-31
amended: 2026-08-30 at the tasks gate — §3.4 and AC-5 said the master playlist's *variant line* gains the subtitle group, which was true while the master answered exactly one variant and stopped being true on the day 008's own T15 gave an HDR stream copy a standard-range entrance beside it; the reference gives the group to every variant it writes, so the criterion is written against every variant line and an entrance with no subtitles is the failure it now catches. And §3.2's text/image split reads a codec spelling the file itself does not report: four subtitle codecs are renamed when a file is inspected, and against the unrenamed spellings the rule inverts on the two commonest image formats in a real library; and 2026-08-30 by T2 — the inversion is the **DVD and digital-broadcast bitmap** names alone, the servable-alone flag inverts with the split rather than following it (`PGSSUB` is servable where `DVDSUB` is not), and both facts are stated on every stream of every kind, `false` on everything that is not a subtitle; and 2026-08-30 by T5 — AC-10's *"the concatenation of every window of a track is the whole track"* is false of a cue that starts exactly on a window boundary, which two consecutive windows both answer because their shared boundary position is inclusive at each end: read off the reference first, then **measured** on it, in the constructed form and through the reference's own generated playlist alike. And §3.5 now says, also measured, that a converted document carries a region declaration and a placement setting on every cue's timing line, which is where a player puts the text and which a cue-by-cue check cannot see; and 2026-08-30 by T7 — §3.5 gains the eleven format spellings with the label and the byte order mark of each read off a run, of which `subrip` and `webvtt` answer a **body** under `application/octet-stream` where both documents had predicted a refusal, the four deprecated query parameters that override the address, the query start position that beats the one in the path, and the same-format short circuit measured with a window on it — which contradicts AC-10 and is recorded beside it rather than folded into it, because plan §6.8 makes that amendment the user's; and §3.7 gains a row nothing had measured, an item that **exists** and holds nothing servable, which answers `500` where an identifier naming nothing answers `400` and which is what an implementation has to know before it can answer either; and 2026-08-30 by T7, at the user's decision — **AC-9 and AC-10 both state what was measured, in this change rather than at a later gate**, because documentation moves with the code in the same commit and T8 through T11 are written by people reading them. AC-10 gains the same-format short circuit: a windowed fetch whose requested format is the one the track is already in answers the **whole track**, unwindowed and unrebased, on both fetch routes and under either timestamp switch — read off the reference first, then measured on it, which is why the clause carries a `[probe:]` and not the `[source:]` it was found with. AC-9 is narrowed rather than widened: the timings match the source's exactly where the container begins at zero and are offset by the container's own start time where it does not, which is **parity stated precisely** — a reference server on the same extraction build answers the same offset, because it asks for no timestamp preservation either — and the criterion is checked against the offset read off the container rather than against a literal, so it holds on both builds. Plan §6.8 had left both to a later gate; the deferral was declined; and 2026-08-30 by T8, at the user's decision — **§3.7's playlist column is corrected in three cells, in the change that measured them.** The table was drafted from a run that asked the fetch route almost every question, and its shape invites a reader to assume that where both routes answer they answer alike; on these three they do not. The malformed-identifier refusal names **`itemId`** on the playlist and `routeItemId` on the fetch routes, because each names its own path segment — measured, both bodies in one run, where the row had generalised the fetch route's answer to a route that had never been asked. An item that **exists and is not a video** — a series, an audio track — is the playlist's problem-details `404` where the fetch routes answer `500` for that same identifier, because the playlist's own lookup asks for a video and not for an item: that cell was a **dash**, and the dash is what let an implementation reuse the fetch routes' lookup and answer `500` to an item this route refuses one question earlier. And a source with **no runtime** had no row at all and now has one, marked ⚠️ **read rather than measured**: the reference refuses it on its own argument check at the same status and the same bytes as a zero window length, and it could not be measured because every media source of every video item in the measured library states a runtime and the route asks for a video before it reads one — a fact about libraries, since a runtime is written by the scan that creates an item and the state cannot be produced from outside a server. The run reports that miss on every run. Plan §6.8 had made two of the three the user's; the deferral was declined, for the reason AC-9 and AC-10 were taken at T7 — documentation moves with the code in the same commit, and T9 through T12 are written by people reading this table; and 2026-08-30 by T9 — **§3.3 gains a rule that inverts this feature's own assumption about who its subtitle half touches, and two criteria are narrowed against it.** The delivery method of the *selected* track is a direct-play condition on the reference: a track the client could only be shown by burning in refuses direct play with `SubtitleCodecNotSupported`, so the same file, the same client and the same profile answer a direct play with no track named and a transcode with one — measured on both sides, an external text profile keeping direct play for a `subrip` track and losing it for the image track in the same file, and an index naming no stream costing nothing. AC-15's *"nothing changes what a direct-played file answers"* was therefore false of exactly the request this feature exists to serve, and it now says so. AC-2's *"and a delivery address naming it"* was false too, in two directions: the index and the method are both dropped where the delivery method is *external* and the index is dropped where it is `-1`, while a request asking for burn-in puts the index back and appends its own flag to the end of the address. §3.2 gains the start position that address carries — zero on every playlist answer, the negotiation's own seek on a progressive one — and the flag stated beside every external address. And §3.3 records what plan §6.8 left owed: the five members of the delivery-method vocabulary bind case-insensitively and by ordinal on a request body, while a word that is no member is a `400`, which is the shape 012 measured for a query value; and 2026-08-30 by T10 — **§3.2's `NAME` box gains the assembly, measured, and the one word in it that the plan had spelled from the wrong place.** The reference's own assembly was read at the accepted gate and never reproduced; it now is, on 909 of 909 subtitle streams of a real library, each rebuilt exactly from its own properties with every branch reached. What that reproduction found is the marker for a stream that states no language: the assembly falls back to `Und` only when the localised string behind it is empty, and no served stream is ever in that state — all five localised strings arrive filled on 910 of 910 subtitle streams — so `Und` is a string no reference of any configuration writes and the invariant form's marker is `Undefined`, which is what an English-configured host writes. Plan §6.4 had listed `Und` among the words it called parity; and 2026-08-30 by T11 — **§3.4's own sentence about what announces a track was wrong about the half a client controls.** *"The delivery address naming the manifest method beside a stream index"* was true of every address a negotiation writes and false of the route it addresses: the method alone announces every text subtitle stream, with no index, with `-1` and with an index naming no stream alike, and the index decides only which entry carries the default attribute. That is the difference between a negotiated address and the one the video client rewrites by hand, which is the request this feature exists to serve, so a server requiring both would have announced nothing to it. AC-5 is written against the method alone. Three more, measured in the same run: the vocabulary of that parameter binds in any case and by ordinal and **refuses nothing** — a word that is no member is a `200` announcing nothing where the same word on a request body is a `400`, so AC-6 gains it and the drop method as two further classes that must change the manifest not at all; a track sitting beside the media is announced like one inside the container, at the wire index §3.6 gives it; and the `Unknown` language fallback, which the accepted gate could only read because every text track of the source it picked stated a language, is measured
depends_on: [003, 005, 008]
---

# 011 — Subtitle delivery

> **This document describes WHAT and WHY only.** No technology names, no storage decisions.

## 1. Purpose

Put a subtitle on the screen: announce the text subtitle tracks a file carries, let a client choose
one, and serve its text.

**Client behaviour unlocked:** subtitles. Today a user who turns them on sees nothing, on every
playback path that is not the file being read byte for byte off disk.

**This is a v1 promise being kept, not v1 growing.** The
[roadmap's own exclusion row](../../docs/roadmap.md#out-of-scope-and-why) excludes *subtitle
burn-in* and says, in the same sentence, that **v1 delivers subtitle files**. Nothing was ever
written that delivers one. The promise fell between two features rather than being descoped by
either: 008 §2 excludes *"subtitle extraction, conversion and delivery as a separate route"* — a
correct decision for a feature about deciding a play method and moving bytes — and the feature
order ran 001 to 010 with no row to catch what 008 put down. So the scope question this feature
answers is not *should v1 do this*, which the roadmap answered before 001 was written. It is
*which feature owns it*, and until this document the answer was none.

## 2. Scope

**In scope**

- The subtitle properties a media source's streams carry: which streams are **text** subtitles,
  whether a stream can be served on its own, how it would be delivered and from where.
- Choosing a delivery method for a chosen track from what the client's profile says it accepts,
  and honouring the track the client asked for — at negotiation and on a delivery request.
- Announcing text subtitle tracks in the **HLS master playlist**, when the client's profile asks
  for them.
- **Three** routes, not two. `GetSubtitlePlaylist` is the address a manifest entry names;
  `GetSubtitle` is the cues, whole or windowed by position parameters; and `GetSubtitleWithTicks`
  is the same answer with the start position in the path instead of the query — which is in scope
  because it is the route a *negotiation's own* `DeliveryUrl` names, so a client following the
  address it was handed lands there and not on the other one
  `[probe: tools/probe_subtitle_negotiation.py, Jellyfin 10.11.11, 2026-08-29]`. The opening draft
  of this section said two, and folded the windowed form into `GetSubtitle` as though it were the
  same operation; §7's first dependency is corrected with it.
- Extracting a text subtitle track from its container and converting it to the format the client
  asked for.
- **Subtitle files sitting beside the media**: seeing them at all, counting them, offering them as
  streams, and serving them.

**Out of scope**

- **Subtitle burn-in.** Unchanged, and for the reason the roadmap gives: a text-rendering stack and
  a second filter path. This feature is the other half of that row, not a reopening of it.
- **Image subtitles as text.** A bitmap track cannot become a cue list without optical character
  recognition. It is announced as a stream like any other and never announced in a manifest;
  §3.4.
- **Subtitle search and download from providers**, and everything under
  `/Providers/Subtitles/…`. The roadmap lists it under *later, unscheduled*.
- **Uploading or deleting a subtitle** — `POST` and `DELETE /Videos/{itemId}/Subtitles/{index}`.
  No analysed client calls either, and v1 writes nothing into a library root (004 §2).
- **Per-user subtitle preferences** — a default language, a "forced only" mode, a remembered
  choice. 008 §3.1 records `DefaultSubtitleStreamIndex` as exactly this and v1 does not keep it;
  a client sends the index it wants on each request. **Measuring §3.3 made this exclusion
  expensive, and it is kept anyway**: the reference's default track is a function of two *user*
  settings this feature does not store, so with none stored there is nothing to compute and the
  answer is no default at all — which is exactly what the reference answers for a user whose
  subtitle mode is `None`. What is lost is that a *new* user's mode is `Default` rather than
  `None` `[probe: tools/probe_subtitle_negotiation.py, Jellyfin 10.11.11, 2026-08-29]`, so a
  stock reference proposes a track where Atrium proposes none. Recorded as an accepted gap
  ([behaviours §5](../../docs/compatibility/behaviours.md#5-accepted-gaps-in-v1)) rather than
  designed around, because the alternative is two user settings, five modes and a language
  preference list — a per-user feature, which is what §2 excludes.
- **Every other finding the two client traces record**, from a source with no stored inspection to
  UDP discovery. §2.1 says where each goes, and why none of them is decided by what decides a
  subtitle.
- Trickplay, and live streams.

### 2.1 Why this is one feature, and why the gaps beside it are not in it

**Both client documents hand this scoping decision here, and neither takes it.** The trace of the
video client closes with *"they are not 008 defects, and they are not amendments to it — 008 closes
on its accepted scope; every finding above was measured against code that does what its spec, plan
and tasks say it does"*, and routes its findings to *"the next one after 010"*
([client-atrium-tvos §6](../../docs/compatibility/client-atrium-tvos.md#6-where-these-findings-go));
the music client's trace says the same in its own words and calls them *"input to the feature that
comes after 010"*
([client-embeat-mobile §7](../../docs/compatibility/client-embeat-mobile.md#7-where-these-findings-go)).
That feature is this one, so the first thing to say is that **an earlier draft of this section was
wrong**: it recorded four of these as *amendments owed to 008, 007 and 002*, and they are not.
008's code does what 008's documents say. A specification that is silent about a case is not a
specification that is wrong about it, and calling a silence an amendment blames a document for a
question nobody put to it.

**But "input to the feature after 010" is a destination, not a scope**, and both documents show
it: each closes with a *grouping table* whose rows are visibly different shapes of work — one a
009 scope decision, one an amendment to 001, one a sentence of prose in behaviours, one a test
that *"can be written today"*. Two documents' worth of findings do not become one feature by
sharing a date. So the scoping is the work this section does, and the test it uses is the
[roadmap's *"008 is one feature, not two"*](../../docs/roadmap.md#feature-order) paragraph read in
the other direction.

That paragraph keeps transcoding inside 008 because it is **the third branch of one decision**, and
splitting it *"would put one decision ladder in two specifications and guarantee they drift"*.
Applied to a feature being *assembled* rather than split, it says: a finding belongs here only if
what decides it is what decides subtitles. Bundling by *when a gap was found* rather than by *what
decides it* turns a specification into a changelog, and produces the same drift from the other
direction — several mechanisms, each described in two documents.

**On that test, exactly two findings are this feature**, and the video client's own grouping table
had already put them together and sized them: *"§4.2 + §4.3 — subtitle delivery, end to end. The
largest of them, and §4.3 is a line inside it rather than work of its own."* §4.3's own closing
sentence is blunter still — *"whoever does §4.2 owns this line"* — and it is AC-4 here.

**Everything else is handed on**, to the shape the client documents themselves measured it as:

| Finding | Shape of the work, as its own document sizes it | Not here because |
|---|---|---|
| [tvOS §4.1](../../docs/compatibility/client-atrium-tvos.md#41-a-source-with-no-stored-inspection-is-the-clients-documented-dead-end) + [music §5.1](../../docs/compatibility/client-embeat-mobile.md#51-a-source-with-no-stored-inspection-loses-the-music-clients-whole-negotiation) — a source with no stored inspection advertises direct play and offers no address | One branch, one criterion, one accepted-gap entry. *"Shared with the tvOS client's §4.1 and worth doing once"* | It is a decision about what the **negotiation ladder** answers with nothing to negotiate against. One skipped branch, two clients, two failures that look nothing alike — and none of them is a subtitle |
| [tvOS §4.6](../../docs/compatibility/client-atrium-tvos.md#46-two-spellings-of-hls-and-only-one-of-them-selects-hls) — `"Hls"` does not select HLS where `"hls"` does | *"One probe each, then one parameter each. Both are cheap and neither is safe to guess"* | It is the **delivery address's** vocabulary, and it is unanswerable before the probe that says whether the reference is case-insensitive there |
| [tvOS §4.4](../../docs/compatibility/client-atrium-tvos.md#44-get-sessions-takes-no-deviceid-and-the-client-sends-one) — the session list takes no `deviceId` | Same row of the same table | It is the **session list's** request shape. The client already matches on what is playing, so this is a degradation whose worst case is an administrator reading another device's row |
| [tvOS §4.5](../../docs/compatibility/client-atrium-tvos.md#45-the-fmp4-init-segment-restarts-the-encoder-which-is-the-defect-the-client-pre-warms-to-dodge) — the initialisation segment restarts production | *"A [behaviours §3.0](../../docs/compatibility/behaviours.md#30-how-the-decision-is-made) decision, taken on a probe, before any code"* | It is a **defect decision**, not a requirement. The restart is faithful reproduction, the client pre-warms to dodge it `[client-contract: 2026-08-29, §3]`, and the Jellyfin-side claim behind it is a third-party lead this repository has not measured. A feature is the wrong container for an argument |
| [music §5.3](../../docs/compatibility/client-embeat-mobile.md#53-a-piped-mp3-carries-no-xing-frame-which-is-not-the-blank-one-the-client-measured), [§5.4](../../docs/compatibility/client-embeat-mobile.md#54-every-universal-request-re-encodes-for-a-different-reason-than-the-reference-does), [§6.1](../../docs/compatibility/client-embeat-mobile.md#61-an-honest-content-length-on-a-capped-transcode), [§6.2](../../docs/compatibility/client-embeat-mobile.md#62-keying-a-transcode-on-a-client-supplied-playsessionid) | *"One question about where a progressive re-encode is produced, asked three ways. Settle it once"* | It is one question about **where a progressive re-encode is produced**. Two of its four parts were recorded as Principle I improvements rather than parity — measured at this gate, and **one of the two was mis-framed**: keying on the client's play session is what the reference already does (§7.1). Both stay owned there rather than here |
| [music §5.8](../../docs/compatibility/client-embeat-mobile.md#58-the-album-play-queue-is-correctly-ordered-by-accident) — the album play queue is ordered by accident | *"One test, and it can be written today. The only item here that needs no decision"* | See below |
| [tvOS §4.7](../../docs/compatibility/client-atrium-tvos.md#47-udp-discovery-is-out-of-v1-and-the-client-needs-it) — UDP discovery | *"An amendment to 001, or its own small feature. Not a route"* | It is not an endpoint and not a subtitle |

**The ordering finding is 005's, and its own document says the cheap answer is a test.** The music
client requests its album queue as `SortBy=ParentIndexNumber,IndexNumber,SortName` and its year
sort as `ProductionYear,PremiereDate,SortName`; three of those keys are outside v1's eight-member
vocabulary and an unrecognised token is **dropped, not refused**
([behaviours §1.12](../../docs/compatibility/behaviours.md#112-an-unrecognised-query-value-is-ignored-not-rejected),
005 §3.3), so two thirds of the first request evaporates and the album queue is ordered by
`SortName` alone. It comes out **right**, because an audio sort name is a zero-padded disc and
track ahead of the raw name
([behaviours §2.6](../../docs/compatibility/behaviours.md#26-sortname-has-two-derivations-and-three-types-use-the-second)),
so sorting by it sorts by disc then track then name — the order the dropped keys asked for, reached
from the other end. **The finding is that nothing states the dependency**, and the answer is one
test asserting a multi-disc album's order under that exact `sortBy` string, which belongs beside
[005 §3.4](../005-item-query-api/spec.md). Adding the three keys is the expensive answer and
probably the wrong one — a key no reference server orders by is a delta — and what decides it is a
debt already recorded:
[behaviours §2.5](../../docs/compatibility/behaviours.md#25-sortby-vocabulary)'s eight members rest
on a `[prior-probe: Jellyfin 10.11.11, 2026-06-13]` nobody has re-run, while the reference's own
enumeration has thirty members `[source: Jellyfin.Data/Enums/ItemSortBy.cs @ v10.11.11]`. A client
sending three keys outside the eight is the first evidence the debt is worth paying.

**What the handed-on findings are not is homeless.** Four of them are one probe away from being
specifiable and none of them can be specified before its probe, which is a feature whose first act
is a measurement session — and by the convention that numbers are assigned in the order features
are *started* ([specs/README.md](../README.md)), it takes its own number on the day that session
runs. Recording them here with their sizing, rather than absorbing them, is what keeps that
possible.

### 2.2 What is left, and why it is one thing

Subtitles are one mechanism seen from five places, which is the positive half of the same argument.
A subtitle a client can watch requires, in order: a stream the source says is text; a delivery
method negotiated from the profile; a line in the manifest; an address that line points at; and
bytes at that address. **Remove any one and the other four deliver nothing** — which is precisely
how v1 arrived at zero subtitles from exclusions that were each defensible alone
([client-atrium-tvos §4.2](../../docs/compatibility/client-atrium-tvos.md#42-v1-has-no-way-to-deliver-a-subtitle-and-this-client-has-one-way-to-receive-one)).
The video client's own trace reaches the same conclusion from its side, listing the work in
dependency order — *"emit `IsTextSubtitleStream`; bind `EnableSubtitlesInManifest`; extract and
serve WebVTT; announce the tracks"* — and closing it with *"the first two are cheap and buy nothing
alone"*. That sentence is what makes this one feature rather than four cheap tickets. It is one
decision ladder, which is the test the roadmap paragraph sets.

## 3. Behaviour

### 3.1 What a client sees today

Stated as the starting point, because every criterion below is a change to one of these. Each is
observable from a running server on `main` at 2026-08-29, and each is traced against merged code by
[client-atrium-tvos §4.2 and §4.3](../../docs/compatibility/client-atrium-tvos.md#42-v1-has-no-way-to-deliver-a-subtitle-and-this-client-has-one-way-to-receive-one),
which is where the file-level evidence for this table lives:

| What a client does | What it gets today |
|---|---|
| Reads an item's `MediaStreams` or a source's | Subtitle streams, with codec, language and the default and forced flags — and **no property saying which of them are text**, and none saying how one would be delivered |
| Posts a profile declaring how it accepts subtitles | Accepted, and the subtitle part of it discarded on arrival |
| Posts `SubtitleStreamIndex` with a negotiation | Accepted, and never read |
| Fetches the HLS master playlist | One variant line and no media tags of any kind |
| Fetches any subtitle address | Nothing. No route in the surface accepts one |
| Puts an `.srt` beside a film and rescans | Nothing. No scan sees the file, so nothing counts it |
| Sends a subtitle track selection on a delivery request | Ignored. The delivery routes read an audio track selection and no subtitle one — *"today this costs nothing"*, because there is no subtitle on the HLS path to select, and *"it stops costing nothing the moment §4.2 is closed"* |

The result is the table that document already draws: subtitles work on an on-device remux, because
the tracks are inside the file the client is reading byte for byte, and **nowhere else** — not over
server HLS, remuxed or transcoded, and not from a file beside the media by any path.

### 3.2 Which streams are subtitles, and which of those are text

**Two of the four properties 008 §3.1 named as owed are not this feature's to add.** Whether a
stream is a **text** subtitle and whether it can be served on its own are facts about the file,
and the reference carries both on every read: a bare listing row, a bare item, and a negotiation
with no profile all state them on every subtitle stream. What appears only on a **negotiated**
source is *how* the track would be delivered and *where from* — and a fifth property nobody had
named, the per-stream **score** of §3.3, which appears only for the streams the user's own
preferences selected `[probe: tools/probe_subtitle_negotiation.py, Jellyfin 10.11.11,
2026-08-29]`. That is OQ-2, and it moves work out of this feature rather than into it: the two
file facts belong wherever a stream is emitted, and only the two negotiation answers are decided
here.

**"Wherever a stream is emitted" is literal: both are stated on every stream of every kind**, and
are `false` on everything that is not a subtitle — video, audio and the cover-art streams a
library has — rather than being left off a stream they cannot describe. Measured on 1 968 streams,
947 of them subtitles: both properties on all 1 968; the score, the delivery method, the delivery
address and the external-address flag on none of them; and the file path on the fourteen streams
that came from a file beside the media and no others `[probe:
tools/probe_sidecar_subtitles.py, Jellyfin 10.11.11, 2026-08-30]`.

**The delivery method is resolved for every subtitle stream, not for the selected one.** A source
with six subtitle streams answers six methods, whatever the request selected. The address is
narrower: `DeliveryUrl` is emitted only for the streams whose method is *external*, because that
is the only method whose answer is a URL the client fetches itself, and it is the one property
that brings a companion — the flag saying the address is not a remote one is stated beside it and
on no other stream. Same probe. **That address carries a start position, and it is not always
zero**: it is zero on every answer whose delivery is a playlist, which forces it so because a
playlist preserves timings, and it is the negotiation's own seek where the answer is a progressive
production — a body asking to start ten minutes in is answered an address ten minutes in
`[probe: tools/probe_subtitle_negotiation.py, Jellyfin 10.11.11, 2026-08-30]`.

The text/image split decides everything downstream. A text track is a cue list and can be
converted, served alone and announced; an image track is a sequence of bitmaps and can be none of
those without burn-in, which is out (§2). The reference filters the manifest on exactly this
property `[source: Jellyfin.Api/Helpers/DynamicHlsHelper.cs:192-195 @ v10.11.11]`, and the split
is decided by the **codec spelling** rather than by anything measured in the file: everything
counts as text except a codec containing `pgs`, `dvdsub` or `dvbsub`, or spelled exactly `sup` or
`sub` — with one exception written into the rule itself, because a codec containing `microdvd` is
text however it is spelled: that text format and an image one share the `.sub` extension
`[source: MediaBrowser.Model/Entities/MediaStream.cs:751-761 @ v10.11.11]`. So the split is a
lookup, not an inspection, and a stream with no codec at all is text only when it came from a
file beside the media `[source: MediaBrowser.Model/Entities/MediaStream.cs:639-654 @ v10.11.11]`.

**The spelling the lookup reads is not the one the file itself reports**, and this was
corrected at the tasks gate on 2026-08-30 because the difference is the whole of the rule for two
of the four image formats. Four subtitle codecs are **renamed when a file is inspected** — the
two digital-broadcast spellings, the DVD one and the Blu-ray one become `DVBSUB`, `DVBTXT`,
`DVDSUB` and `PGSSUB` — and every later reader, the split included, sees the renamed form; it is
also the spelling a subtitle stream's codec carries on the wire `[source:
MediaBrowser.MediaEncoding/Probing/ProbeResultNormalizer.cs:632-652, 765-768 @ v10.11.11]`,
`[probe: tools/probe_sidecar_subtitles.py, Jellyfin 10.11.11, 2026-08-30]`. AC-1's "differ in the
first of them" is a claim about the renamed spelling.

**Two of the four renames change an answer and two do not**, measured while implementing the rule
on 2026-08-30, and which two matters because it decides what can prove the rename at all. The
Blu-ray name already contains `pgs` before it is renamed and the broadcast *teletext* one is a
text format under either spelling; it is the **DVD** and **digital-broadcast bitmap** names that
invert, because neither contains `dvdsub` or `dvbsub` until it has been renamed. So a server that
kept the file's own spelling would announce every DVD subtitle track in a library as text —
twenty of them on the reference library measured, against 338 Blu-ray tracks that would have
survived the mistake. Same probe.

**The servable-alone flag inverts with it, and it is not "not an image".** A Blu-ray bitmap track
*can* be served on its own and a DVD bitmap track cannot, so the two commonest image formats
answer that property differently: `PGSSUB` is `true` and `DVDSUB` is `false`, measured on every
subtitle stream of a real library. Same probe.

**The delivery-method property is an answer to a negotiation, not a fact about a file.** The same
track answers differently for two clients, and differently for the same client direct-playing and
transcoding — the reference resolves it separately on the direct-play branch and the transcode
branch of its own ladder
`[source: MediaBrowser.Model/Dlna/StreamBuilder.cs:771-773, 806-807 @ v10.11.11]`.

> ⚠️ **The names a manifest needs are names 008 decided not to emit, and the measurement made the
> bill exact.** The reference labels each announced track with the stream's `DisplayTitle`
> `[source: Jellyfin.Api/Helpers/DynamicHlsHelper.cs:604, 608 @ v10.11.11]`, and that string is
> assembled from up to six pieces joined by ` - `: the track's own title if it has one, then the
> language's **name**, a hearing-impaired word, a default word, a forced word, the codec upper
> cased, and an external word — each of the five words omitted when its flag is false, and each
> attribute dropped when the title already contains it as a substring
> `[source: MediaBrowser.Model/Entities/MediaStream.cs:380-465 @ v10.11.11]`. Measured on a
> Spanish-configured server the announced names read
> `Castellano [Forzados Planos] - Español - Predeterminado - SUBRIP` and
> `Ingles SDH [Completos Planos] - Inglés - Discapacidad Auditiva - SUBRIP`
> `[probe: tools/probe_subtitle_manifest.py, Jellyfin 10.11.11, 2026-08-29]`.
>
> **The assembly is now measured rather than read**, on 909 of 909 subtitle streams of a real
> library, each rebuilt exactly from its own properties: the order, the separator, the codec's
> casing and the title's substring suppression, with every branch reached — a stream with no
> language, a hearing-impaired one, one that came from a file beside the media, and a title that
> swallows an attribute `[probe: tools/probe_stream_display_title.py, Jellyfin 10.11.11,
> 2026-08-30]`.
>
> **And the word for a stream that states no language is `Undefined`.** It is the fifth of the
> localised strings and the one most easily got wrong, because the assembly carries a shorter
> spelling — `Und` — that is written only when the localised string is empty, and no served
> stream is ever in that state: all five arrive filled, on 910 of 910 subtitle streams. So `Und`
> is a string no reference of any configuration writes, and the invariant form's marker is
> `Undefined`. Same probe.
>
> **What that costs is two localisations, not one, and neither is the table v1 already has.** The
> five flag words come from the server's own translation table in its configured interface
> culture; the language name comes from the *platform's* culture data in that same culture — not
> from the culture table `/Localization/Cultures` serves, whose display names are English
> (`Spanish; Castilian` where the announced name says `Español`). So reaching the reference's
> exact string means shipping a translation table and a localised language-name table, in one
> configurable culture, for a string that appears in one attribute of one manifest.
>
> **The decision this feature takes: write the name in the invariant form and record the
> divergence.** A manifest cannot leave `NAME` absent — it is a required attribute — so "omit it,
> as 008 does" is not available here, and that is what made this the one place 011 cannot inherit
> 008's accepted decision unchanged. The invariant form is the same assembly with the English
> words and an English language name, which is what the reference itself writes on an
> English-configured host. **Which** English name is a plan question rather than a spec one, and
> it is not free either: the two tables available disagree — the reference's own culture list
> says `Spanish; Castilian` where the platform data it actually reads says `Spanish` — so a plan
> that reaches for the table v1 already has produces a third string rather than the reference's.
> It is observably different from a Spanish-configured reference on
> every announced track, which is precisely the objection 008 raised — and it is answered here by
> what the attribute is *for*: `NAME` is a label a person reads in a track picker, no client
> branches on it, and the attributes clients do branch on (`LANGUAGE`, `FORCED`, `DEFAULT`,
> `URI`) are byte-identical. **OQ-4, resolved.** The gap stays recorded in
> [behaviours §5](../../docs/compatibility/behaviours.md#5-accepted-gaps-in-v1) with the rest of
> the localised properties, and closing it here would close it there.

### 3.3 Choosing a track and a delivery method

Two inputs and one answer, and the measurement moved both inputs.

**The track a client names is read only when the request also names the media source.** A
negotiation carrying `SubtitleStreamIndex` and nothing else is answered as though it carried
nothing: the index is dropped in silence, and the answer is the default. Add the matching
`MediaSourceId` and the same index is honoured, appears as the source's stated default, and is
written into the delivery address `[probe: tools/probe_subtitle_negotiation.py, Jellyfin
10.11.11, 2026-08-29]`. **This settles the third-party lead the opening draft flagged**: the claim
was that the reference ignores the indices posted with a negotiation and builds its address from
the source's defaults. Measured, it is half right, and the half that is true is the half that
matters to the client — a client that posts an index without a source id gets a delivery address
naming a different track, which is exactly the failure the video client rewrites the address by
hand to avoid `[client-contract: 2026-08-29, §3]`. An index naming no stream at all is not an
error: it is accepted, restated and written into the address.

**With no track named, the rule is not a ranking.** The reference holds a *score* on each subtitle
stream, and the opening draft of this section said it takes the highest and consults the profile
only to break a tie. It never takes the highest. The score is read to find out whether **more than
one** stream shares the top of it; when exactly one does, the score is discarded and the answer is
the source's own stated default, computed by a different rule that never looks at a score
`[source: MediaBrowser.Model/Dlna/StreamBuilder.cs:546-584 @ v10.11.11]`. Measured on the wire:
a user preferring Spanish is answered stream 2 while streams 0 and 1 sit above it on score, and
the same request answers stream 0 the moment the profile is changed
`[probe: tools/probe_subtitle_negotiation.py, Jellyfin 10.11.11, 2026-08-29]`. So the profile
does not pick the default and it does not merely break a tie — **when there is a tie it decides
outright, and when there is not it decides nothing at all.**

**The score is reproducible, and what it is computed from is out of scope.** It is six decisions
read as six digits: the position of the stream's language in the *user's* subtitle language
preferences, then forced, default, servable-alone, text, external, each contributing 2 or 1
`[source: Emby.Server.Implementations/Library/MediaStreamSelector.cs:181-192 @ v10.11.11]`,
reproduced from each stream's own properties and compared against the emitted value on every
scored stream. Which streams are scored at all is decided by the user's subtitle **mode**, and a
mode of `None` scores nothing and states no default. Both settings are per-user preferences §2
excludes, which is why the answer this feature owes for an unnamed track is *no default* — the
reference's own answer for that mode. **OQ-12, resolved, and it is a scope answer rather than an
algorithm.**

**The index is honoured on a delivery request as well**, because a client changes the subtitle
track mid-playback by re-requesting delivery rather than by re-negotiating: the video client
rewrites both track indices in the address it was handed `[client-contract: 2026-08-29, §3]`.
That override already works for audio and is dropped for subtitles
([client-atrium-tvos §4.3](../../docs/compatibility/client-atrium-tvos.md#43-the-clients-track-override-works-for-audio-and-is-dropped-for-subtitles)),
and §3.4's measurement promotes that line from *"a line inside §4.2"* to the **only** thing that
makes a manifest announce anything. AC-4.

**Naming a track can cost a source its direct play, and neither this section nor the plan said
so.** The delivery method of the *selected* track is resolved a second time, against the file
exactly as it stands, and the answer is a direct-play condition like a codec or a container: a
track the client will take as a separate file, embedded in the container, or dropped leaves the
answer alone, and a track it could only be shown by burning in refuses direct play, with
`SubtitleCodecNotSupported` beside whatever else was refused. So the same file, the same client
and the same profile answer a direct play with no track named and a transcode with one. Measured
on both sides of the discrimination: an external text profile keeps direct play for the `subrip`
track of a file and loses it for the image track in that same file, a profile that declares
nothing loses it for either, and an index naming **no** stream costs nothing at all — there is no
track to resolve a method for `[probe: tools/probe_subtitle_negotiation.py, Jellyfin 10.11.11,
2026-08-30]`. AC-15 is narrowed against this, and the ordering it implies is not circular: the
refusal reads the method the file would get *at direct play*, where the manifest method is not
available, and the answer the client is finally given reads it again against whatever the
negotiation landed on.

**The address names the track only where the client is not fetching it for itself.** A negotiation
that selected a track writes the index and the method into the delivery address, with two
subtractions and one override: the index and the method are both dropped where the method is
*external* — the client was already handed that track's own address, so there is nothing for the
delivery route to do — and the index is dropped where it is `-1`, which names no track; and a body
asking for burn-in puts the index back beside an external method, because the delivery route needs
one to honour, while leaving the method dropped. That body also appends its own flag to the end of
the address, after everything else it carries. All four measured in one run
`[probe: tools/probe_subtitle_negotiation.py, Jellyfin 10.11.11, 2026-08-30]`; AC-2 states the
narrowed form rather than the unconditional one it was drafted with.

**And the vocabulary is read loosely and refused strictly.** A profile entry naming its delivery
method in any case — `hls` where the model spells it `Hls` — is answered identically to the
declared spelling, as is the member's ordinal; a word that is no member at all is a `400`. Four
classes, one run, on a request body: the same shape 012 measured for a value in a query string,
which is where this question came from. Same probe.

**The method.** The client's profile declares, per subtitle format, how it will take that format:
embedded in the container, as a separate file, as a separate stream in the manifest, burned in, or
dropped `[source: MediaBrowser.Model/Dlna/SubtitleDeliveryMethod.cs @ v10.11.11]`. The manifest
method applies only when the play method is transcode
`[source: MediaBrowser.Model/Dlna/StreamBuilder.cs:1549 @ v10.11.11]`, which is the mechanical
reason a direct-played file needs nothing from this feature and an HLS-delivered one needs all of
it — and it is measurable from outside: a profile declaring the manifest method is answered *burn
in* on a source it will direct-play, and the manifest method on the same source when the container
is rejected. Same probe.

**Burn-in is the reference's fallback, and it is an answer it gives rather than one it avoids.**
The opening draft called it *"the only place in this feature where a divergence is certain"* and
said the reference's own answer was unavailable *"because it never has to give one"*. It gives one
constantly: every subtitle stream that no declared profile fits is answered `Encode`, per stream,
at negotiation — an image track under any text profile, and *every* track for a profile that
declares no subtitle handling at all `[probe: tools/probe_subtitle_negotiation.py, Jellyfin
10.11.11, 2026-08-29]`. So the branch is not rare and it is not hypothetical; it is the default
answer for most of a real track list.

**The decision this feature takes: say the same word.** v1 answers `Encode` exactly where the
reference does, and never burns anything in. The property is a *statement about what would happen*
on a track the client has not selected, and a client that selects such a track and starts playback
gets a stream with no subtitle painted into it — which is the same thing it gets today, and the
same thing [behaviours §5](../../docs/compatibility/behaviours.md#5-accepted-gaps-in-v1)'s burn-in
row has said since 008. The two alternatives the opening draft listed are both worse and both are
deltas: dropping the track changes a property on every source that has an image subtitle, and
offering it as a separate file promises a fetch that cannot succeed (§3.7 measures what that
fetch does — it attempts the extraction for twenty seconds and then refuses). **OQ-5, resolved**,
and the divergence it was certain would be needed turns out to be parity plus a gap already
recorded.

### 3.4 The manifest

**There is exactly one lever, and it is not the profile.** The opening draft read the reference's
condition as an *or* — a manifest delivery method **or** the client's profile asking for subtitles
in the manifest — and asked, as OQ-1, what the direct-play case answers. The second half of that
*or* is unreachable on this route: the master playlist does not accept the manifest flag as a
parameter at all. It is a parameter of the live-stream playlist, which is out of scope
`[probe: tools/probe_subtitle_manifest.py, Jellyfin 10.11.11, 2026-08-29]`.

**And the reference's own negotiation writes it anyway.** A profile whose transcoding entry sets
the manifest flag is answered a delivery address carrying `EnableSubtitlesInManifest=True`, and
following that address changes nothing: the route it addresses cannot read the parameter it was
given. Same probe. So a client that asks for subtitles in the manifest the way the reference's own
model says to ask gets a manifest with no subtitles in it.

**What announces a track is the delivery address naming the manifest method, and that is the
whole of the condition.** This paragraph said *"beside a stream index"* until T11 measured it, and
the index turns out to decide nothing about whether anything is announced: an address naming the
manifest method and **no index at all** announces every text subtitle stream of the source, and so
does one naming `-1` or an index that names no stream. What the index decides is which entry
carries the default attribute — and nothing at all when it matches no announced stream `[probe:
manual requests via tools/_probe.py, Jellyfin 10.11.11, 2026-08-30]`.

**That correction matters to the client this feature exists for**, and it is why the sentence was
worth measuring rather than reproducing: the video client changes track by *rewriting the address
it was handed* ([§4.3](../../docs/compatibility/client-atrium-tvos.md#43-the-clients-track-override-works-for-audio-and-is-dropped-for-subtitles)),
so a server that required both parameters would announce nothing to a client that sent one. What
is true of the *negotiation* is what the paragraph had generalised: it writes the method only
where a track was selected, so no address it produces carries one without the other — which
returns §3.3's selection rule to the centre of a **negotiated** flow and to nothing else. AC-5 is
written against the address rather than the profile for the reason above, and against the method
rather than the pair for this one.

**The vocabulary of that parameter is read loosely and refuses nothing.** `hls`, `HLS` and `hLs`
announce exactly what `Hls` announces, and so do the member's ordinal `3`, a signed `+3` and a
padded ` 3 `; a word that is no member, an ordinal that names no member, an empty value, `3.0`,
`--3` and a non-ASCII digit each answer a `200` announcing nothing. Same run. **The last class is
the finding**: the same word on a request body is a `400` (§3.3), so one vocabulary is refused two
ways depending on where it arrives — and beside it, in the same address, an index that is not a
number *is* refused. §3.7's shapes are unchanged by this; what changes is that no shape at all is
reached for an unreadable method.

**A comma-separated value is one value, and its parts are combined rather than the first one
winning.** That is measurable rather than academic, because two members combine into a third:
naming the embedded and external methods together announces every text track — the combination is
the manifest method's own position in the vocabulary — while naming the external one twice
announces none, and one unreadable part makes the whole value unreadable. Measured on all four,
same run. It is reproduced, because a server that read only the first name would answer the
opposite on both of the first two.

When the address names the manifest method, the master playlist gains **one media entry per text
subtitle stream** and **every** variant line gains their group
`[source: Jellyfin.Api/Helpers/DynamicHlsHelper.cs:192-210, 338-350, 596-632 @ v10.11.11]`.
Measured verbatim, an entry is:

```
#EXT-X-MEDIA:TYPE=SUBTITLES,GROUP-ID="subs",NAME="…",DEFAULT=NO,FORCED=YES,AUTOSELECT=YES,URI="…",LANGUAGE="spa"
```

with the attributes in that order, the group literally `subs`, `AUTOSELECT=YES` on every entry,
`DEFAULT=YES` on the selected stream and `NO` on the rest, `FORCED` from the stream's own flag,
and the language falling back to a literal `Unknown` rather than being omitted. A variant line
gains `,SUBTITLES="subs"` **last**, after the frame rate. Same probe — with the `Unknown` fallback
read rather than measured there, because every text track of the source it picked stated a
language; it is **measured** now, on the one track of the reference library that states none
`[probe: manual requests via tools/_probe.py, Jellyfin 10.11.11, 2026-08-30]`.

**A track sitting beside the media is announced exactly like one inside the container**, at the
wire index §3.6 gives it — ahead of the container's own — and with the external word in its name.
Measured on a source whose three announced entries are all files beside it. Same run. So the
renumbering §3.6 describes reaches the manifest, which is the one place a wrong number would send
a client to another track rather than to nothing.

**Every variant line gains it, not one**, and that sentence was corrected at the tasks gate on
2026-08-30 because the shape it describes changed underneath it: an HDR source whose video is
stream-copied is offered a standard-range entrance beside the copy, so the master carries more
than one variant. The reference gives the same group to every entrance it appends — the copy, the
codec entrances, the level rewrite and the adaptive-bitrate variants alike `[source:
Jellyfin.Api/Helpers/DynamicHlsHelper.cs:213-315, 325-345 @ v10.11.11]` — and it has to: the
entrance exists so that a client which cannot render the copy has somewhere to go, and an
entrance with no subtitle group is that client losing subtitles for the reason it was offered the
entrance. AC-5 is written against every variant line for the same reason. **Measured again at T11,
through the probe rather than by hand** — three variants against an HDR film whose video is
copied, all three ending in the group `[probe: tools/probe_subtitle_manifest.py, Jellyfin 10.11.11,
2026-08-30]`; the gate had to measure it by hand because that probe read only the *first* variant
line, which is the reason the shape went unnoticed for a day.

**The address in a media entry is a playlist, not a file**, it is relative to the master
playlist's own directory, and it carries two things the opening draft did not know about: a
**hard-coded** window length of thirty seconds — not the segment length of the stream — and **the
caller's own access token**, written into the query string. The token is load-bearing rather than
decorative: the playlist route it addresses requires a caller (§3.5), and a player following a
`URI` out of a manifest sends no headers of its own. Same probe.

**Image subtitle streams are never announced**, because the filter is on the stream kind — but the
filter is *not* on the selection, and the difference is observable: selecting an **image** track
for manifest delivery still announces every text track, with `DEFAULT=NO` on all of them, because
no announced stream matches the selected index. Same probe. AC-7 holds and is narrower than it
reads.

**A burned-in selection suppresses the group entirely**, and so does every other method: external,
burn-in, and an index with no method at all each answer a manifest with no media entries. Stated
so that the absence of the branch is a recorded consequence rather than an oversight.

**When the address names no subtitle method, the master playlist is unchanged.** This is the
criterion that keeps 008's accepted answer intact: a client that says nothing about subtitles must
receive byte-identically what it receives today (AC-6). Measured against the same negotiation's
own address, four ways.

### 3.5 Fetching a subtitle

Three addresses, and the manifest reaches all three.

**The per-track playlist** — `GetSubtitlePlaylist` — lists the windows of one track of one media
source, at a window length the caller states, covering the source's runtime. Measured, it is a
complete VOD playlist: `#EXTM3U`, a target duration, version 3, media sequence 0,
`#EXT-X-PLAYLIST-TYPE:VOD`, one `#EXTINF` and one address per window, `#EXT-X-ENDLIST`
`[probe: tools/probe_subtitle_delivery.py, Jellyfin 10.11.11, 2026-08-29]`.

**Its entries name a lower-case `stream.vtt`**, always in that one format whatever the track holds,
always with both timestamp switches set, and always with the caller's token appended — where the
reference's own declaration spells the route `Stream.{format}` with a capital. Both spellings
answer, identically. A route that is served only under the declared spelling would break every
client that follows a playlist as written, which is what AC-8's traversal exists to catch.

**The track itself** — `GetSubtitle`, and `GetSubtitleWithTicks` for the form with the start
position in the path — answers the track converted into the format named in the address, whole or
windowed. Measured: the cue-list formats all answer, a format the server cannot produce answers a
refusal (§3.7), and two spellings answer something other than a subtitle file — the cue list as a
JSON object of `TrackEvents` with tick positions, under `json` and under its alias `js`. Same
probe.

**The two timestamp switches are not decoration, and one of them changes the bytes.** Without the
copy switch, a window's cues are rebased on the window — a cue 36.1 s into the file comes back at
6.1 s in a window starting at 30 s. With it, the cue keeps the time it has in the file. The time
map switch prepends a mapping line into the header **and drops the byte order mark** the plain
answer starts with, because the body is rebuilt to insert it. The playlist sets both. **OQ-11,
resolved.** Same probe.

**The mapping line is inserted wherever the format's own name appears, not only at the top.** The
rewrite is a plain replacement over the finished document, so a cue whose text contains the word
gets a mapping line of its own — and the switch is read against one spelling of the format and not
against the alias beside it `[source: Jellyfin.Api/Controllers/SubtitleController.cs:250-262 @
v10.11.11]`.

**The document a client is given is not the minimal one, and the difference is on screen.** A
converted subtitle in the format the manifest names carries a **region declaration** in its
header — `Region: id:subtitle width:80% lines:3 regionanchor:50%,100% viewportanchor:50%,90%`,
after a blank line — and a placement setting on **every** cue's timing line, measured whole:
`00:00:35.099 --> 00:00:37.185 region:subtitle line:90%`
`[probe: tools/probe_subtitle_delivery.py, Jellyfin 10.11.11, 2026-08-30]`. That is where a
player puts the text. A header of the format name alone, with bare timing lines, is well formed,
holds the same cues, and positions them somewhere else — so [§6](#6-conformance) carries this as
the second thing asserted as bytes. Two smaller shapes of the same kind are **read and not
measured**: a cue whose end does not follow its start is pushed out by one millisecond by that
writer alone, and the cue-list format renumbers its cues from one, discarding whatever the source
called them `[source: MediaBrowser.MediaEncoding/Subtitles/VttWriter.cs:34-38, SrtWriter.cs:32 @
v10.11.11]`. Neither is reachable from a playlist a client follows, and each is one row of the
same battery away. **One of the two is now measured**: a window that starts past the first cue,
asked for under the spelling that renders rather than the one the short circuit answers, comes
back numbered from `1` where the same window's cue-list answer calls the first cue `131`
`[probe: tools/probe_subtitle_delivery.py, Jellyfin 10.11.11, 2026-08-30]`. The millisecond is
**not**, and the run says why: 5 983 cues across twelve tracks read from files beside the media,
and not one of them states an end that does not follow its start. It stays a reading until a
library carries such a file, which is a fact about the library and not about the server.

**Every spelling the writers admit answers, and two of them answer under a label nothing chose.**
The eleven a client can put in the address were measured in one run, and three of them had never
been asked for `[probe: tools/probe_subtitle_delivery.py, Jellyfin 10.11.11, 2026-08-30]`:

| Spelling | Answer | Content type |
|---|---|---|
| `vtt`, `srt`, `ass`, `ssa`, `json`, `js` | `200` | `text/vtt`, `application/x-subrip`, `text/x-ssa` twice, `application/json` twice |
| `ttml` | `200` | `application/ttml+xml` |
| `subrip`, `webvtt` | **`200`** | `application/octet-stream` |
| `sub`, `xyz` | `400`, `text/plain` | — |

`subrip` and `webvtt` are the two spellings the reference can write and has **no media type
for**, and the answer is not the refusal that suggests: the document is rendered and the response
falls back to the type an unrecognised container gets rather than refusing. So a format with no
media type is still a body — and it is a *different* body from the one its canonical spelling
answers, because `Stream.subrip` converts where `Stream.srt` short-circuits (below). The byte
order mark is on every converted document but the cue-list one, `ttml` included.

**The address is not the last word: four deprecated query parameters override it.** `itemId`,
`mediaSourceId`, `index` and `format` are declared obsolete and still bound, and each beats the
path segment beside it — `Stream.vtt?format=srt` answers SubRip under `application/x-subrip`, and
`Stream.vtt?index=` naming no stream answers that index's refusal. On the route with the start
position in its path, a `StartPositionTicks` in the **query** wins as well: the same address
carrying both answers the track from its first cue. Same probe.

**Asking for the format the track is already in answers the whole track, window and all.** The
answer is decided before anything is parsed, so both timestamp switches and both position
parameters are ignored: `Stream.srt` on a SubRip track answers the identical 84 858 bytes with a
window on it and without one, on both fetch routes `[source:
MediaBrowser.MediaEncoding/Subtitles/SubtitleEncoder.cs:144-155 @ v10.11.11]`, same probe. It is
unreachable from a playlist — every entry names `stream.vtt`, and no track a client reaches is
already WebVTT — and one hand-made request away.

> ⚠️ **This contradicted AC-10 as written, and AC-10 now carries it.** *"A windowed fetch answers
> the cues of that window and no others"* is true of every window a client reaches by following an
> address and was false of that one request, so the criterion states the exception itself rather
> than pointing at prose that states it — the same shape as the clause it already carries for the
> boundary repeat, and with the same `[probe:]` citation, because this was measured before it was
> written. [Plan §6.8](plan.md#68-what-no-probe-here-has-measured-and-what-stays-owed) had left
> the amendment to be taken at a later gate; it was taken here instead, because documentation
> moves with the code in the same commit and four tasks after this one are written by people
> reading AC-10.

> ⚠️ **The last window's duration is written in the server's locale.** A partial window comes back
> as `#EXTINF:7,851,` on a Spanish-configured server, which an HLS parser reads as a duration of
> `7`. Recorded as a reference defect and diverged from at
> [behaviours §3.12](../../docs/compatibility/behaviours.md#312-a-subtitle-playlists-window-durations-are-written-in-the-servers-locale--class-b-diverged):
> Atrium writes a decimal point, always, because it has no server locale to reproduce the defect
> from and inventing one in order to write a number wrongly is not replication.

> **Two rows of one client document disagree about when a whole-file fetch fires, and no probe
> here can settle it.** The answer table of
> [client-atrium-tvos §2](../../docs/compatibility/client-atrium-tvos.md#2-the-answer) records the
> client requesting `…/Subtitles/{index}/Stream.vtt` ***when the manifest carries none***, while
> [§4.2](../../docs/compatibility/client-atrium-tvos.md#42-v1-has-no-way-to-deliver-a-subtitle-and-this-client-has-one-way-to-receive-one)
> says that path is wired for the other server flavour and *"the client will not compensate"*. It
> is a claim about the client, not about the reference, so it is a question for the trace's author
> and **it stays open**. Nothing in this document depends on the answer: the route is in either
> way, because the manifest addresses it (§3.4), every criterion below is written against the
> manifest's traversal rather than against a bare fetch, and what the answer changes is the
> *order* the work is worth doing in — which belongs in the plan.

**These two routes do not share an authentication rule, and the measurement matches the
declaration for once.** The playlist route refuses a caller with no token and a caller with an
unknown token alike, with an empty `401` and no body; both fetch routes answer `200` with the cues
to a caller with no credential at all. Both accept the token in the query string, which is how the
addresses in §3.4 and in the playlist work at all. This is the same split
[behaviours §2.10](../../docs/compatibility/behaviours.md#210-the-image-and-delivery-routes-accept-a-token-and-require-none)
describes for the delivery routes, extended to two more. **OQ-6, resolved.** Same probe.

**Neither route answers `Accept-Ranges`**, and both state a `Content-Length`. That is the same
shape 008 T14 measured on the two HLS playlists, and it is worth stating here because *"every
delivery route whose body has a known size answers `Accept-Ranges: bytes`"* was false then and is
false on two more routes now.

**Conversion is part of delivery, not a separate capability.** A client asks for a format by
naming it in the address; the stored track is in whatever format its container holds. If v1 served
only the formats it already has, a client asking for the one format it can render would be refused
by a server that holds the same cues in a different spelling.

### 3.6 Subtitles beside the media

A subtitle file next to the media is invisible to v1 today: nothing in a scan looks at it, so
nothing counts it, offers it, or serves it. 008 §3.1 already records one half of the consequence —
`HasSubtitles` counts only what is inside the container, so *"a film with an external `.srt` and no
embedded track answers nothing where the reference answers `true`"*, measured true
`[probe: tools/probe_sidecar_subtitles.py, Jellyfin 10.11.11, 2026-08-29]`. That gap closes here,
and the rest of it with it.

**The rule is a stem match and a right-to-left read, and it was reproduced rather than described.**
A file counts when its name without its extension begins with the media file's name without *its*
extension, and then either stops or continues with a dot; the extension must be one of nine the
reference admits, two of which name image formats. What follows the stem is read one dot-delimited
token at a time from the right, each token claimed by the first vocabulary that recognises it —
a default word, then a forced word, then a language, then a hearing-impaired word — and every
token nothing claims is prepended to the stream's **title**. The reproduction was checked against
six items in directories holding up to 259 files each, and every discovered file, its language,
its flags and its title came out identical. **OQ-7, resolved.** Same probe.

**Two consequences no rule about names states, and the second is the expensive one.**

1. `HasSubtitles` counts the discovered files, which is the 008 gap closing.
2. **The discovered streams are numbered first.** An item whose subtitles are all files answers
   them at indices 0, 1, 2, and the container's own video and audio streams begin at 3. So putting
   a file beside a film **renumbers every stream it has** — and a stream index is what a delivery
   address carries. AC-12 is written for this: removing the file must renumber them back.

**One place the reference looks that no probe here can reach**: the item's own internal metadata
directory, where the reference puts a subtitle it downloaded or extracted `[source:
MediaBrowser.Providers/MediaInfo/MediaInfoResolver.cs:216-226 @ v10.11.11]`. No route exposes it, so
its contribution to a source's stream list is a bound rather than a measurement — and it is a bound
this feature can live inside, because v1 neither downloads nor stores extracted subtitles (§2).

### 3.7 Error paths

Measured per route, because 008 found delivery-route refusals splitting across three shapes by
*where* the refusal happens, and these two routes split the same way
`[probe: tools/probe_subtitle_delivery.py, Jellyfin 10.11.11, 2026-08-29]`. **OQ-8, resolved.**

| Condition | Playlist route | Fetch route |
|---|---|---|
| Caller has no token, or an unknown one | `401`, no body | `200` — the cues (§3.5) |
| Item id is well formed and names nothing | `404`, problem details | `400`, `text/plain` `Error processing request.` |
| Item id names an item **with nothing servable** — a series, an audio track | `404`, problem details — the playlist asks for a **video**, so an item that is not one is refused exactly as one that does not exist | `500`, `text/plain` |
| Item id is the all-zero identifier | `400`, `text/plain` | `400`, `text/plain` |
| Item id is not an identifier at all | `400`, problem details naming **`itemId`** | `400`, problem details naming **`routeItemId`** |
| Media source names nothing on the item | `500`, `text/plain` | `500`, `text/plain` |
| Media source states **no runtime** — ⚠️ **read, not measured** | `400`, `text/plain` `[source: Jellyfin.Api/Controllers/SubtitleController.cs:356-363 @ v10.11.11]` | — |
| Index names no stream | **`200`** — a full playlist | `500`, `text/plain` |
| Index names a video or audio stream | **`200`** — a full playlist | `500`, `text/plain` |
| Index is negative | **`200`** — a full playlist | `500`, `text/plain` |
| Index names an **image** subtitle, text format asked for | — | `400`, `text/plain`, after ~20 s of extraction. **Atrium: the same status and the same bytes, before any process starts** |
| Format in the address cannot be produced | — | `400`, `text/plain` |
| Window length absent | `400`, problem details naming `segmentLength` | — |
| Window length is zero | `400`, `text/plain` | — |
| Window length is not a number | `400`, problem details naming `segmentLength` | — |
| Window whose end precedes its start | — | `200`, a body with no cues |

**Three cells of the playlist column were wrong or empty, and all three were corrected at T8.**
The table was drafted from a run that asked the *fetch* route almost every question, and its shape
invites a reader to assume that where the two routes both answer, they answer the same thing. On
these three they do not:

- **The two routes name different parameters for the same malformed identifier.** The playlist's
  problem details name `itemId` and the fetch routes' name `routeItemId`, because each names its
  own path segment and the two spell it differently — read out of the two bodies in one run
  `[probe: tools/probe_subtitle_delivery.py, Jellyfin 10.11.11, 2026-08-30]`. The row said
  `routeItemId` for both, which was the fetch route's answer generalised to a route that had never
  been asked.
- **The playlist route asks for a *video*, not for an item**, so an item that exists, is visible
  and is not one — a series, an audio track — is refused exactly as an identifier naming nothing
  is: the problem-details `404`, where the fetch routes answer `500` for that same identifier.
  Measured on a series identifier beside the fetch route's `500` for it in one run, same probe.
  That cell was a dash, and the dash is what let an implementation reuse the fetch routes' lookup
  and answer `500` to an item this route refuses one question earlier.
- **A source with no runtime had no row at all, and now has one marked as read.** The reference
  refuses it on its own argument check, which is the same status and the same bytes as a window
  length of zero. It is a **reading and not a measurement**, and the reason is a fact about
  libraries rather than a gap in the probe: every media source of every video item in the measured
  library — all 2 480 of them — states a runtime, and the route asks for a video *before* it reads
  one, so nothing of any other type reaches that check. A runtime is written by the scan that
  creates an item, so the state cannot be produced from outside a server at all. The run reports
  the miss every time rather than inferring the row, and it closes the day a library carries such
  a file.

**The playlist route never reads the index it is given.** It is a declared parameter that the
reference's own source marks as unused, and the consequence is the row above: a playlist for a
stream that does not exist is a `200` listing a hundred addresses, every one of which answers
`500`. This is the sharpest reason AC-8 is written as a traversal rather than as a string
comparison — a manifest and a playlist can both be well formed and lead nowhere, and only
following them says so.

**Two shapes, split by where the refusal happens.** A refusal the framework raises before the
route runs is problem details naming the parameter it could not bind; a refusal the route raises
is `text/plain` `Error processing request.` with no `Content-Length`. Both are shapes
[behaviours §1.11](../../docs/compatibility/behaviours.md#111-there-are-four-error-shapes-not-one) already
names — with one extension: §1.11 records the `text/plain` shape for a controller refusing at
`4xx`, and these two routes reach the identical 25-byte body at **`500`** as well, on every
condition that reaches a lookup and finds nothing.

**And the split between those two statuses on the fetch route is *whether the item is there*, not
what was wrong with the request.** An identifier nothing holds is a `400`; an item that exists and
has nothing to convert — a series, an audio track, a source no `mediaSourceId` matches, an index
naming no subtitle — is a `500`, measured on all four `[probe:
tools/probe_subtitle_delivery.py, Jellyfin 10.11.11, 2026-08-30]`. That is the reverse of the pair
008 measured on its four delivery routes, where the *item* is the `404` and the **source** is the
`400`, so a route that resolved the two the same way would answer the wrong status to one of them.
The third row of the table is this feature's: it was measured because an implementation cannot
answer the first row without knowing which side of it an existing-but-empty item falls on. **And
that row is where the two columns part company**: the same series identifier is the fetch routes'
`500` and the playlist's `404`, because the playlist asks a question the fetch routes never ask —
*is this a video* — before it asks for parts at all.

## 4. Data the feature owns

| State | Observable as | Lifetime |
|---|---|---|
| Which streams of a source are text subtitles | The stream properties of §3.2 | For as long as the file is inspected |
| Subtitle files found beside the media | Additional external subtitle streams on the source, and `HasSubtitles` | Until a rescan finds them gone |
| The converted text of a track | Only as the body of a fetch | Not observable as state; a repeat fetch answers the same cues |

**No per-user state.** A subtitle choice is carried on the request that needs it and is not
remembered, which is the same line 008 draws for `DefaultSubtitleStreamIndex` and §2 restates.

## 5. Acceptance criteria

1. Every subtitle stream of an item and of a media source carries the text flag and the
   servable-alone flag — on a listing row and on a bare item, not only on a negotiation — and a
   text track and an image track differ in the first of them. A **negotiated** source carries, in
   addition, a delivery method on every subtitle stream and a delivery address on every stream
   whose method is external.
2. A negotiation carrying a subtitle index **and the matching media source** answers a source
   whose stated default subtitle track is the one named, and — where the track's own delivery
   method is not *external* and the index is not `-1` — a delivery address naming it; the
   same negotiation without the media source answers as though no index had been sent. A
   negotiation carrying neither answers **no default subtitle track**, which is §3.3's measured
   answer for a server that keeps no per-user subtitle preference.

   The two subtractions are the reference's own and were measured at T9, which is why this
   criterion no longer says the address names the track unconditionally: a client that will fetch
   the file for itself was already handed that file's address, and `-1` names no track at all. A
   request that asks for burn-in puts the index back and leaves the method out, and the address
   then ends with that request's own flag `[probe: tools/probe_subtitle_negotiation.py, Jellyfin
   10.11.11, 2026-08-30]`.
3. A profile that declares no subtitle handling negotiates exactly as it does today except that
   every subtitle stream now states a delivery method of `Encode` — the answer §3.3 measured, and
   the one property that changes.
4. A **delivery** request carrying a subtitle index is served with that track — the criterion
   [client-atrium-tvos §4.3](../../docs/compatibility/client-atrium-tvos.md#43-the-clients-track-override-works-for-audio-and-is-dropped-for-subtitles)
   asks for, in its subtitle half. *(Its audio half was owed to 008 and **008 T14 paid it**: the
   parameter had only ever been asserted as a string in a negotiated address, and the assertion
   that it changes the audio that comes back is now
   `tests/conformance/test_progressive_delivery.py`'s. Without both halves the video client's
   "change the track" path breaks against Atrium and no test here fails, so this criterion is the
   remaining one.)*
5. A master playlist request whose **address** names the manifest delivery method carries one
   media entry per text subtitle stream, and **every** variant line it answers ends in the group
   name — including the standard-range entrance a high-dynamic-range stream copy is offered
   beside the copy. A subtitle stream index in the same address decides which entry carries the
   default attribute and nothing else: an address naming none, naming `-1`, or naming a stream
   that does not exist announces the same entries with no default among them.

   Written against the address rather than against the profile because the profile flag is
   unreachable on this route (§3.4), which is what OQ-1 measured; written against *every* variant
   because the criterion was drafted when the master answered exactly one and no longer does; and
   written against the **method alone** because T11 measured that the index is no part of the
   condition — the criterion said *"beside a subtitle stream index"*, which was true of every
   address a negotiation writes and false of the route, and the difference is exactly the address
   a client rewrites by hand (§3.4).
6. A master playlist for any request that does **not** name the manifest delivery method is
   **byte-identical** to the one the same request answers today — including one that names the
   manifest flag, one that names an index with no method, and one that names the external or
   burn-in method. Two more classes joined those four at T11, measured beside them: a method that
   is **no member of the vocabulary at all**, which announces nothing without refusing anything
   (§3.4), and the drop method.
7. An image subtitle stream never appears in a master playlist, whatever the address asks for —
   including when the selected index **is** an image stream, which still announces every text
   stream with the default attribute set to `NO` on all of them.
8. The address of every media entry in a master playlist is fetched successfully by following it
   as written, and every entry of the playlist it answers is fetched successfully the same way —
   **as written** including its lower-case spelling of the fetch route, which is not the spelling
   the route is declared under.
   *(Written as a traversal rather than as a string comparison: the failure this feature exists to
   prevent is an announcement that leads nowhere — §3.4 and §3.7.)*
9. A whole-file fetch of a text track answers its cues, in the requested format, with timings
   that match the source's — **exactly where the container begins at zero, and offset by the
   container's own start time where it does not**. That offset is not this server's and not a
   tolerance: an extracted track is expressed on a timeline beginning at the container's start
   time, which is the earliest of all its streams', so one frame of audio encoder priming read as
   a *negative* start time carries every cue of the subtitle track beside it forward by exactly
   that much — 21 ms on one extraction tool's build and nothing on another, for the same bytes
   `[probe: two builds of the extraction tool over the generated fixtures, 2026-08-30]`. **A
   reference server on the same build answers the same offset**, because its own extraction asks
   for no timestamp preservation either `[source:
   MediaBrowser.MediaEncoding/Subtitles/SubtitleEncoder.cs:629-646 @ v10.11.11]`, so this
   criterion states parity precisely rather than recording a divergence. It is checked against the
   offset read **off the container being extracted** rather than against a literal, which is what
   makes it exact on either build and still failing on a dropped cue, a mangled timing or the
   wrong track. Narrowed at T7, which owns this criterion's test, from the finding T6 measured.
10. A windowed fetch answers the cues of that window and no others — **except where the
    requested format is the one the track is already in, which answers the whole track,
    unwindowed and unrebased**; with the copy switch their timings are the source's and without it
    they are the window's; and the concatenation of every window of a track is the whole track,
    **plus one repeat of every cue that starts exactly on a window boundary**.

    The exception is the reference's own short circuit: it hands back the readable file before it
    parses anything, so the window and both timestamp switches are ignored `[source:
    MediaBrowser.MediaEncoding/Subtitles/SubtitleEncoder.cs:144-155 @ v10.11.11]` — measured on
    both fetch routes, with and without the copy switch, and the same body byte for byte as the
    unwindowed request (§3.5)
    `[probe: tools/probe_subtitle_delivery.py, Jellyfin 10.11.11, 2026-08-30]`. It is unreachable
    from a playlist, whose every entry names one format no extracted track is already in, and one
    hand-made request away. Added at T7, which measured it rather than reading it.

    The repeat is measured, both ways round: a cue at 37.802 s is answered by the window
    ending there **and** by the window starting there, and the same cue one millisecond off the
    boundary is answered by the earlier window alone — and the reference's *own* playlist reaches
    it, a cue at 3 282 s falling on the grid at a window length of 6 s and coming back from both
    of the two entries that share that position, followed as written
    `[probe: tools/probe_subtitle_delivery.py, Jellyfin 10.11.11, 2026-08-30]`. The reason it
    happens is that consecutive windows are handed the *same* boundary position — one window's
    end is the next one's start — and both ends of the selection are inclusive `[source:
    MediaBrowser.MediaEncoding/Subtitles/SubtitleEncoder.cs:100-112,
    Jellyfin.Api/Controllers/SubtitleController.cs:394-405 @ v10.11.11]`.
11. A subtitle file placed beside a media file and then scanned becomes an external subtitle
    stream on that item's source, is counted by `HasSubtitles`, and is fetchable through the same
    routes as an embedded one — with its language, its flags and its title read from its name by
    §3.6's rule.
12. Removing that file and rescanning removes the stream, **renumbers the remaining streams back**
    to the indices they had before it appeared, and affects neither the item nor its user data.
13. Each row of §3.7 answers the status and body measured for it, per route — including the two
    rows where the playlist route answers `200` for a stream that does not exist.
14. A subtitle fetched twice answers the same bytes.
15. Nothing in this feature changes what a **direct-played** file answers: the negotiation, the
    source list and the delivery of a file the client reads byte for byte are unchanged, except
    for the stream properties AC-1 and AC-3 add — **and except where the request names a subtitle
    track the client cannot take**, which is not a direct play any more and never was on the
    reference either.

    Narrowed at T9, which measured it. The *selected* track's delivery method is a direct-play
    condition: a track the client will take as a separate file, embedded, or dropped changes
    nothing, and a track it could only be shown by burning in refuses direct play with
    `SubtitleCodecNotSupported` (§3.3). So the unchanged half is asserted on a file with no
    subtitle stream at all and the narrowing is asserted on both sides of the discrimination —
    the same file and the same profile, one index kept and one lost
    `[probe: tools/probe_subtitle_negotiation.py, Jellyfin 10.11.11, 2026-08-30]`.
16. A subtitle window's declared duration is written with a decimal point whatever the host is
    configured for — the divergence
    [behaviours §3.12](../../docs/compatibility/behaviours.md#312-a-subtitle-playlists-window-durations-are-written-in-the-servers-locale--class-b-diverged)
    argues.

**Two things the music client asks for are deliberately not criteria here**, and are measurements
instead: an honest `Content-Length` on a capped transcode, and keying a transcode on a
client-supplied play-session identifier (OQ-9, OQ-10; §2.1 hands them to the question they are
three-quarters of). Both are now measured, and **one of them was mis-framed** — see §7. A criterion
asserting either would make this feature fail for being *more correct* than the thing it
reproduces, or would smuggle another feature's work in under this one's name.

## 6. Conformance

| Endpoint | Level | How it is proven |
|---|---|---|
| The subtitle stream properties on an item and a source | **L3** | Golden per stream kind, plus differential — the properties appear on every source that has a subtitle, so an error is everywhere |
| `POST /Items/{itemId}/PlaybackInfo`, subtitle half | **L3** | Golden per profile class — declared method × text/image × named index or not, with and without the media source id — plus differential |
| `GET /Videos/{itemId}/master.m3u8`, subtitle half | **L3** | Golden manifest per address class, including the six that must leave it unchanged (AC-6, four at the gate and two measured at T11), plus differential |
| `GET …/Subtitles/{index}/subtitles.m3u8` | **L2** | Playlist shape, window coverage, the invariant duration of AC-16, and the traversal of AC-8 |
| `GET …/Subtitles/{index}/Stream.{format}` and its ticks-in-path form | **L2** | Cue-level assertions against a fixture of known cues, whole and windowed, both timestamp switches, both spellings of the path, plus determinism (AC-14) |
| Subtitle files beside the media | **L2** | Fixture mutated between assertions (AC-11, AC-12), including the renumbering |
| Error paths | **L2** | Table-driven per route over §3.7 |

**Converted text is asserted cue by cue, not byte by byte against the reference.** Two converters
given the same cue list agree on the cues and disagree on whitespace, ordering of optional
attributes, and how they round a timestamp — differences no player sees. What is asserted is the
property a client depends on: *the cues, their text and their timings are the source's*. A byte
comparison here would be asserting that Atrium ships the reference's converter, which is not a
compatibility claim — the same argument 008 §6 makes for transcoded bytes.

**A converted document's own framing is the exception inside that exception, and it is asserted as
bytes.** The header a format declares, the placement setting on each cue's timing line, the
millisecond a zero-length cue is pushed out by and the byte order mark five of the six writable
formats begin with are not properties of the cue list and are not whitespace either — they decide
where a player draws the text and what the time map switch has to drop (§3.5). The first two and
the mark are measured on the wire — the mark on every spelling, `ttml` included
`[probe: tools/probe_subtitle_delivery.py, Jellyfin 10.11.11, 2026-08-30]`; **the renumbering is
measured too**, and the millisecond alone is still read, for the reason §3.5 gives. Those are
pinned literally; everything between them stays a cue comparison.

**The manifest is the exception, and it is asserted as bytes.** Everything in a media entry except
`NAME` is mechanical, and `NAME` is the one attribute this feature knowingly diverges on (§3.2), so
the differential compares the entries with that attribute masked and compares `NAME` against the
invariant form.

**Fixtures are synthetic**, extending 008's: the existing generated media gain an embedded text
subtitle track of known cues, an embedded image subtitle track, and a sidecar file beside one of
them. No copyrighted media, and the cue list is small enough to assert in full.

## 7. Open questions, and what measuring them did

**All twelve were measured on 2026-08-29**, by five probes written for this gate. Ten of them
block the plan and are now answered; two of them block nothing here and are recorded as
measurements. **Four of the twelve did not survive their own probes**, and one of the four was
wrong in a way that changes what this feature is for.

| # | Answer | Held? | Measured by |
|---|---|---|---|
| OQ-1 | **No.** The question assumed the manifest flag was one of two conditions; the master playlist route does not accept it at all, on any play method. It is a parameter of the live-stream playlist. The one lever is the delivery address naming the manifest method — beside a stream index in the row as it was accepted, which T11 measured to be the method alone — and the reference's own negotiation writes the unreadable flag into that address anyway (§3.4) | ✗ **died** | `tools/probe_subtitle_manifest.py` |
| OQ-2 | Two of the four properties are on every bare read; the delivery method and its address appear only on a negotiated source, the method on **every** subtitle stream and the address only on the external ones. A fifth property, the score, appears there too (§3.2) | ✗ **narrowed** | `tools/probe_subtitle_negotiation.py` |
| OQ-3 | Eight attributes in a fixed order, the group literally `subs`, `AUTOSELECT=YES` always, the language falling back to a literal `Unknown`, a **hard-coded** thirty-second window length and **the caller's own token** in the address; **every** variant line gains the group last (§3.4 — "the variant line" as the row was accepted, corrected at the tasks gate and reproduced at T11) | ✓ | `tools/probe_subtitle_manifest.py` |
| OQ-4 | The name is the localised display title, assembled from up to six pieces out of **two** localisations — the server's translation table for the flag words and the platform's culture data for the language name, both in the server's interface culture. **Decided: the invariant form**, with the divergence recorded, because a manifest cannot omit the attribute and no client branches on it (§3.2) | ✓ + decided | `tools/probe_subtitle_manifest.py` |
| OQ-5 | **The reference does answer.** Burn-in is not a branch it declines to reach — it is the per-stream answer for every track no declared profile fits, which is every image track under a text profile and every track for a profile that declares nothing. **Decided: say the same word and burn nothing in**, which is parity plus a gap already recorded, not the certain divergence the draft predicted (§3.3) | ✗ **died** | `tools/probe_subtitle_negotiation.py` |
| OQ-6 | The declaration and the wire agree for once: the playlist refuses no-token and unknown-token alike with an empty `401`; both fetch routes answer `200` to a caller with no credential; all three accept the token in the query string, which is how the emitted addresses work (§3.5) | ✓ | `tools/probe_subtitle_delivery.py` |
| OQ-7 | A stem match, a dot delimiter, nine extensions, and a right-to-left read in which each token is claimed by the first vocabulary that recognises it and the rest becomes the title. Reproduced and checked against six items in directories of up to 259 files. Two consequences the question did not ask about: `HasSubtitles` counts them, and the discovered streams are numbered **ahead of** the container's own (§3.6) | ✓ + more | `tools/probe_sidecar_subtitles.py` |
| OQ-8 | Fourteen rows, two shapes, and the sharpest one is not an error at all: the playlist route never reads the stream index, so a playlist for a stream that does not exist is a `200` whose every entry is a `500` (§3.7) | ✓ + more | `tools/probe_subtitle_delivery.py` |
| OQ-9 | Measured, framing intact — see below | ✓ | `tools/probe_progressive_production.py` |
| OQ-10 | Measured, **framing inverted** — see below | ✗ **died** | `tools/probe_progressive_production.py` |
| OQ-11 | Without the copy switch a window is rebased on itself; with it the cues keep the file's timings. The time-map switch prepends a mapping line **and drops the byte order mark**, because the body is rebuilt to insert it. The playlist sets both (§3.5) | ✓ | `tools/probe_subtitle_delivery.py` |
| OQ-12 | **The highest score is never taken.** The score is read only to find out whether more than one stream shares the top of it; with a single stream there the score is discarded and the source's own default answers, computed by a rule that never looks at a score. With a tie the client's profile decides outright. And the whole computation is a function of two per-user settings §2 excludes, so v1's answer for an unnamed track is *no default* — the reference's own answer for a user whose subtitle mode is `None` (§3.3) | ✗ **died** | `tools/probe_subtitle_negotiation.py` |

### 7.1 The two measurements that are not this feature's to act on

Both were recorded so they would not be lost and so that nobody would read them as failures. Both
are now measured, and **the second was mis-framed in the same way a neighbouring finding was**: the
music client's gapless finding was recorded as an improvement and re-measured as a parity gap, and
OQ-10 turns out to be the same shape. `[probe: tools/probe_progressive_production.py, Jellyfin
10.11.11, 2026-08-29]`

**OQ-9 — an honest `Content-Length` on a capped transcode. Framing intact: it is an improvement.**
A lossless source asked for at a bitrate cap answers chunked with no `Content-Length`, and it still
does on a repeat of a request whose bytes are already produced and sitting in a file — the same
answer, byte for byte, delivered eight times faster. So there is no state in which the reference
states a length for this, and an honest one would be a Principle I improvement exactly as
[client-embeat-mobile §6.1](../../docs/compatibility/client-embeat-mobile.md#61-an-honest-content-length-on-a-capped-transcode)
says. One correction to the record: the answer carries `Accept-Ranges: none` — the header is
present and says no, rather than absent. Still owned by the *"where a progressive re-encode is
produced"* question of §2.1, and still not a criterion here.

**OQ-10 — keying a transcode on a client-supplied play session. Framing inverted: on three routes
of four it is parity, and on the fourth it is a defect of the reference.** The question was
recorded as *"declaring the parameter is a delta — the reference has not got it there"*. The
reference has got it there: `playSessionId` is a declared parameter of the two audio delivery
routes and of the video one, and the reference **already keys the produced file on it**, together
with the media path, the user agent and the device
`[source: Jellyfin.Api/Helpers/StreamingHelpers.cs:374-383 @ v10.11.11]`. Measured: the same
request repeated with the same play session is answered from the existing file, and with a
different one it is produced again.

What has no such parameter is `/Audio/{itemId}/universal` — the one route the music client uses —
and it does not merely lack it: it mints a **fresh** play session per request, so every reconnect
and every retry re-encodes from the beginning. Measured on the same track: two `/universal`
requests took the same time as each other, where the two keyed ones did not.

So the ask splits, and neither half is the one that was written down. Declaring the parameter on
the routes that already have it is **parity**, and Atrium not keying on it would be a gap.
Declaring it on `/universal` is the delta — and the thing worth arguing there is not a new
parameter but the reference's own choice to discard the session, which is a
[behaviours §3.0](../../docs/compatibility/behaviours.md#30-how-the-decision-is-made) defect
decision on the route rather than a feature request. **Neither belongs here**, and both belong to
the same question §2.1 hands on, now with the measurement it needed.

### 7.2 The one question no probe here can answer

Whether the video client fetches a whole-file subtitle when the manifest carries none, or will not
compensate at all. Its own trace says both in different sections, it is a claim about the client
rather than about the reference, and **it stays open, marked as needing the trace's author**. §3.5
records why nothing in this document depends on the answer: the route is in either way, every
criterion is written against the manifest's traversal, and what the answer changes is the order the
work is worth doing in — which belongs in the plan.

### 7.3 The three dependencies outside this document

1. **The surface grew by three rows, not two** — `GetSubtitlePlaylist`, `GetSubtitle` and
   `GetSubtitleWithTicks`, the last of them because it is the route a negotiation's own
   `DeliveryUrl` names. Added to [surface.yaml](../../docs/compatibility/surface.yaml) and to
   [api-surface-v1.md §8.1](../../docs/compatibility/api-surface-v1.md) at this gate, validated
   against the reference's own document, and the surface is 58 endpoints rather than 55.
2. **The accepted-gap row is corrected twice.** The subtitle row of
   [behaviours §5](../../docs/compatibility/behaviours.md#5-accepted-gaps-in-v1) said subtitles
   were *"delivered as files"*; they were not delivered at all. **008 T14 made that correction**,
   in the change that marked 008 `Implemented` — and this gate corrected it again, because the
   ordered list it then gave as the closing mechanism (*"emit `IsTextSubtitleStream`, bind
   `EnableSubtitlesInManifest`, extract and serve, announce"*) is two-fifths wrong: the property
   is already emitted by every read, and the manifest flag is not a parameter the route accepts.
   The `HasSubtitles` row and the localised-properties row move with it, and one new gap is
   added — no per-user subtitle preference, so no default track is proposed (§2).
3. **The two client traces are a floor, not a ceiling**, and they say so: absence from one means
   *not measured*, never *not needed*. The video client's went stale in a day. Nothing in CI
   notices when they do.

## 8. References

- [docs/roadmap.md](../../docs/roadmap.md#out-of-scope-and-why) — *"v1 delivers subtitle files"*,
  the promise this feature keeps, and the feature-order table corrected in the same change
- [docs/compatibility/client-atrium-tvos.md](../../docs/compatibility/client-atrium-tvos.md) — the
  video client traced against merged code: §4.2 and §4.3 are this feature, §6 is the scoping
  handover §2.1 answers, and §4.1 and §4.4 through §4.7 are what §2.1 hands on
- [docs/compatibility/client-embeat-mobile.md](../../docs/compatibility/client-embeat-mobile.md) —
  the music client's counterpart: §5.8 is the ordering finding owed to 005, and §6.1 and §6.2 are
  OQ-9 and OQ-10 with the status this document keeps them at
- [008 §2, §3.1, §3.3, §3.5, §3.7](../008-playback-negotiation-and-delivery/spec.md) — the
  exclusion this feature reverses, the three stream families it emits, and the delivery rules it
  inherits unchanged
- [005 §3.4](../005-item-query-api/spec.md) — the sort vocabulary §2.1 hands back
- [003 §3.2, §3.7](../003-library-configuration-and-scanning/spec.md) — what a scan looks at, and
  the name cleaning a sidecar rule would have to match
- [docs/compatibility/behaviours.md §1.11, §2.10, §3.0, §3.3](../../docs/compatibility/behaviours.md)
  — the error shapes, the token rule on delivery routes, the defect procedure, and the sizing rule
- `[spec: GetSubtitle, GetSubtitleWithTicks, GetSubtitlePlaylist, GetMasterHlsVideoPlaylist, GetPostedPlaybackInfo, MediaStream, MediaSourceInfo, DeviceProfile, SubtitleProfile, SubtitleDeliveryMethod, TranscodingProfile]`
- **The five probes this gate wrote**, which are where every measurement above comes from:
  `tools/probe_subtitle_negotiation.py` (OQ-2, OQ-5, OQ-12 and the media-source rule),
  `tools/probe_subtitle_manifest.py` (OQ-1, OQ-3, OQ-4),
  `tools/probe_subtitle_delivery.py` (OQ-6, OQ-8, OQ-11 and the locale defect),
  `tools/probe_sidecar_subtitles.py` (OQ-7), and
  `tools/probe_progressive_production.py` (OQ-9, OQ-10)
- The reference's own paths, read at the opening and **now measured against**:
  `[source: Jellyfin.Api/Helpers/DynamicHlsHelper.cs:192-210, 338-350, 596-632 @ v10.11.11]`,
  `[source: Jellyfin.Api/Controllers/SubtitleController.cs:207-345, 389-392 @ v10.11.11]`,
  `[source: Jellyfin.Api/Controllers/DynamicHlsController.cs:165-276, 408-470 @ v10.11.11]`,
  `[source: MediaBrowser.Model/Dlna/StreamBuilder.cs:546-584, 654, 771-773, 806-807, 1442-1570 @ v10.11.11]`,
  `[source: Emby.Server.Implementations/Library/MediaStreamSelector.cs:30-192 @ v10.11.11]`,
  `[source: Emby.Server.Implementations/Library/MediaSourceManager.cs:395-424 @ v10.11.11]`,
  `[source: MediaBrowser.Model/Entities/MediaStream.cs:253-465, 639-761 @ v10.11.11]`,
  `[source: MediaBrowser.Providers/MediaInfo/MediaInfoResolver.cs:85-253, Emby.Naming/ExternalFiles/ExternalPathParser.cs, Emby.Naming/Common/NamingOptions.cs:163-318 @ v10.11.11]`,
  `[source: Jellyfin.Api/Helpers/StreamingHelpers.cs:374-383, 515-560 @ v10.11.11]`,
  `[source: MediaBrowser.Model/Dlna/SubtitleDeliveryMethod.cs, MediaBrowser.Model/Dlna/SubtitleProfile.cs @ v10.11.11]`,
  `[source: MediaBrowser.Model/Dlna/TranscodingProfile.cs:119, MediaBrowser.Model/Dlna/StreamInfo.cs:117, 1067-1070
  @ v10.11.11]`, `[source: MediaBrowser.Providers/MediaInfo/FFProbeVideoInfo.cs:275 @ v10.11.11]`
