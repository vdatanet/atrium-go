---
feature: 008-playback-negotiation-and-delivery
title: Playback negotiation and delivery
status: Implemented
created: 2026-08-26
updated: 2026-08-30
accepted: 2026-08-29
implemented: 2026-08-29
amended: 2026-08-29 by T3 — §3.1 gains the measured media-source field set (31 properties on every source, `VideoType` on a video one), the 32-bit number format its rates and level are written in, the three stream families v1 does not emit, and two divergences the first draft stated as parity: a multi-part film's parts are media sources here and separate items in the reference, and `HasSubtitles` counts only the streams inside the container; and 2026-08-29 at the spec review, which wrote the five probes the OQ table had been citing prospectively and ran all of them — all twelve open questions answered, and five claims did not survive: the policy story was fiction (no playback route consults `EnableMediaPlayback`, and a single denied permission moves nothing at negotiation — §3.2, §3.3, AC-31), `EnableTranscoding: false` in the request body is ignored (OQ-12), `static=true` on a mismatched container is not an error but the original bytes behind the wrong label (§3.5, AC-18), `enableRedirection` never redirects a local file (OQ-4, AC-21), and the reference's HLS segments already carry `Content-Length` — the §3.5 divergence shrank to the progressive routes. Plus one defect nobody was looking for: a sample-rate ceiling is answered from the Opus rate ladder and can be **exceeded** (§3.6, AC-19); and 2026-08-29 by T4 — §3.3's rule 1 loses "or empty": an absent profile means anything and an empty one permits nothing, which are opposite answers; the reasons list is measured to say why *direct play* failed rather than which rung was reached, to name `DirectPlayError` when nothing else explains the refusal, and to arrive in flag-value order; and a numeric ceiling is compared against the value the server holds rather than the shorter decimal it printed; and 2026-08-29 by T6 — §3.5's authentication sentence was the wrong way round: the four `stream` routes accept every mechanism and **require none**, where `/universal` alone requires one, so AC-32 records the decision 002 deferred here; the range table gains five measured rows including the one the RFC would have got backwards (an unreadable `Range` is the whole body, never a `416`); a delivery route's own refusal is the third error shape rather than problem details; the container is a label with a fallback and a spelling rule; and a static response carries exactly four headers and does no conditional handling; and 2026-08-29 by T8 — §3.6's codec-less hole is not a codec-less transcoding profile: the profile defaults to mp3 and negotiates, and it is the streaming request behind it that infers a codec from a request path with no extension, which is why the empty `200` arrives with a `transcodingContainer` as well as without one; AC-19's bit-depth clause is a direct-play refusal and not an output target, because neither server ever states a sample format; `transcodingProtocol` is unvalidated and case-insensitive, so a typed parameter would refuse requests the reference serves; `container` is split on commas before bars; and this route's three refusals are none of its siblings' — the item `404` is problem details where theirs is the third shape, and both `mediaSourceId` shapes answer one `400` where theirs split `400`/`500`; and 2026-08-29 by T9 — §3.6's PCM/WAV warning was wrong in both halves once its prior-probe was discharged: the `500` has two causes (a `wav` extension inferred as a codec, and a `pcm_*` codec with no `audioBitRate`), and the headerless body comes from the *transcoding* container, so AC-20's `Container=wav` named a request that answers mp3 on both servers. Neither symptom belongs to one route family — the split is whether an `audioBitRate` was sent — and AC-20 now names every WAV shape and the length and `Range` each of them carries; the codec-less hole is container-dependent, because the codec the reference ends up asking for is `aac` rather than the request path, and a `wav` transcoding container can carry it; and 2026-08-29 by T10 — §3.7's two cadence numbers were both attributed to the wrong thing: 3.004 s is a requested three seconds scaled by the rate the *container stores* (23.975988, where an exact 24000/1001 answers 3.003 s) and not "the cadence at 23.976 fps", and 6.0 s is the copy path's own default laid as an equal grid rather than "the source's own keyframes" — a copy follows real keyframes only for a container the operator has permitted on-demand extraction for, which ships as Matroska alone. §3.7 gains the playlist routes' four measured refusals and their header set, and AC-32 gains them: they are the second and third delivery routes that require a token, where it had named `/universal` alone; and 2026-08-29 by T11 — §3.7's two per-segment parameters are not symmetric: `runtimeTicks` decides the bytes and the index in the path only names the produced file; a segment answers the static header set including a `Last-Modified` and an honoured `Range`, `SegmentContainer=mp4` gives the playlist an `#EXT-X-MAP` naming a segment numbered -1, and the segment route's six refusals split across two shapes by where they happen; rule 2's second half is a divergence rather than a description, because the reference states the scaled cadence only to its playlist; and 2026-08-29 by T12 — §3.8's stop route was described with the wrong key and the wrong effect: `deviceId` is mandatory at the binder and then decides nothing, so a call carrying an unknown device still stops the session its `playSessionId` names, and the well-formed call does **not** remove the session's `TranscodingInfo` — the reference leaves the object in place, less its completion percentage and frame rate, until playback is reported stopped, which is the divergence behaviours §3.11 argues. §3.8 gains the measured kill timer (sixty seconds, and ten for a progressive stream, which nothing here is) and the report `/Sessions` carries while a transcode runs; and 2026-08-29 by T13 — §3.8's produced-segment window is a **distance behind the client**, not a file age: nothing is removed until the furthest-fetched position has passed `SegmentKeepSeconds`, and what goes then is every index below `(position − window) ÷ segment length` — measured with segment 29 gone and segment 33 kept on one 720-second window, forty-five seconds after both were produced. AC-29 said "older than the configured window" and the two rules disagree on exactly the case a paused client makes. §3.3's delivery-time policy rule is **per stream and video-only**: the reference force-copies each stream against its own permission, an audio-only delivery consults neither, `EnablePlaybackRemuxing` has no delivery-time reader at all, and the reference's own refusal beside the force-copy cannot fire because the same permission has already rewritten the codec to a copy; and 2026-08-29 by T14 — two acceptance criteria said something the tests that prove them contradict, which is what building the map surfaced: AC-6 derived `SupportsTranscoding` from the negotiated answer where §3.3 already had it right and T4 had measured it as a property of the profile, and AC-11's "every delivery route whose body has a known size" is false of the two playlist routes — measured, they carry a `Content-Length` and no range unit at all, on the reference and here, so §3.5's table gains the exception rather than Atrium gaining a header the reference does not send; and 2026-08-30 by T15 — §3.7's headline was wrong about a branch nothing had reached: OQ-7's "exactly one `#EXT-X-STREAM-INF`" was measured on the library's first film, which was standard range, so the SDR entrance beside a stream copy could not fire and its absence was recorded as the shape of the route. Measured on an HDR source, the master carries the copy plus one standard-range entrance per **permitted encoder** — h264 always, hevc and av1 only where `AllowHevcEncoding` or `AllowAv1Encoding` is on, and both ship off — every one at the copy's own `BANDWIDTH`, `RESOLUTION` and `FRAME-RATE`, so a client selects on colour range and on nothing else. §3.7 also gains the two Dolby Vision emissions and the reason neither is reproduced: `SUPPLEMENTAL-CODECS` and the `dvh1` sample-entry tag both turn on a range *flavour* §3.1's inspection cannot derive from colour metadata, so the reference sends neither for any source this feature can describe — behaviours §5.10
depends_on: [005, 007]
---

# 008 — Playback negotiation and delivery

> **This document describes WHAT and WHY only.** No technology names, no storage decisions.

## 1. Purpose

Decide how a given client should play a given file, tell it where to fetch the bytes, and then
serve those bytes correctly.

This is where a media server is judged. Every other feature can be slightly wrong and the user sees
a cosmetic defect; get this wrong and nothing plays.

**Client behaviour unlocked:** playback.

## 2. Scope

**In scope**

- `POST` and `GET /Items/{itemId}/PlaybackInfo`.
- Media inspection: what a file actually contains.
- Device-profile evaluation and the direct-play / remux / transcode decision.
- `GET /Audio/{itemId}/stream[.{container}]`, `GET /Audio/{itemId}/universal`.
- `GET /Videos/{itemId}/stream[.{container}]`.
- `GET /Videos/{itemId}/master.m3u8`, `/main.m3u8`, `/hls1/{playlistId}/{segmentId}.{container}`.
- `DELETE /Videos/ActiveEncodings`.
- **Software transcoding**: re-encoding video and/or audio when neither direct play nor remux
  satisfies the profile, within the profile's ceilings.
- Byte-range delivery, and the session lifecycle behind a remux or a transcode.

**Out of scope**

- **Hardware-accelerated encoding and decoding.** VAAPI, QSV, NVENC, VideoToolbox. Every frame in
  v1 goes through the CPU. This is a throughput decision, not a protocol one: the bytes a client
  receives are the same either way, which is why it can arrive later without any client noticing.
- **Subtitle burn-in.** Painting subtitles into frames needs a text-rendering stack and a second
  filter path. v1 delivers subtitle files; it does not draw them.
- Live streams, `/LiveStreams/Open`, `/LiveStreams/Close`.
- Subtitle extraction, conversion and delivery as a separate route.
- Trickplay.

> **Why the order is direct play, then remux, then transcode.** Direct play costs nothing. Remuxing
> copies the elementary streams into a different container: no decode, no encode, near-zero CPU, and
> an output whose size is computable — and it covers the large majority of real playback, because
> most incompatibilities are container mismatches, not codec ones. Transcoding costs a decode and an
> encode per frame, so it is *last*, reached only when the first two have failed. It is in v1
> because the alternative answer at that point is "cannot play this", and a file the user owns and
> cannot watch is the one failure that has no cosmetic version
> ([roadmap](../../docs/roadmap.md#in-scope)).

## 3. Behaviour

### 3.1 Media sources

Each playable item has one or more **media sources**. A single-file movie has one; a multi-part
film (003 §3.3) has one per part — and that last clause is a **divergence**, described below.

What a source says about the content — its codecs, streams, duration and dimensions — comes from
inspecting the actual file and never from its extension. The one field that is **not** a fact about
the content is `Container`, and the note below says what it is instead. A source carries:

| Group | Fields |
|---|---|
| Identity | `Id`, `Path`, `Protocol`, `Type`, `Name`, `ETag` |
| Container | `Container`, `Size`, `Bitrate`, `RunTimeTicks`, `Formats` |
| Streams | `MediaStreams`, `DefaultAudioStreamIndex`, `DefaultSubtitleStreamIndex` |
| Capability | `SupportsDirectPlay`, `SupportsDirectStream`, `SupportsTranscoding`, `SupportsProbing` |
| Transport | `IsRemote`, `IsInfiniteStream`, `ReadAtNativeFramerate`, `IgnoreDts`, `IgnoreIndex`, `GenPtsInput`, `RequiresOpening`, `RequiresClosing`, `RequiresLooping`, `RequiredHttpHeaders`, `HasSegments` |
| Video | `VideoType` — on a video source only |
| Attachments | `MediaAttachments` |
| Delivery | `TranscodingUrl`, `TranscodingContainer`, `TranscodingSubProtocol`, `UseMostCompatibleTranscodingProfile` |

`[spec: MediaSourceInfo]`

**Every one of those is sent unconditionally**, and that is the reason the last four groups are
listed rather than left out: 31 properties on every audio source and those plus `VideoType` on
every video one, identical across 180 sources of three item types
`[probe: tools/probe_media_source.py, Jellyfin 10.11.11, 2026-08-29]`. A property the reference
always sends is one a client can see missing — the argument 005 §3.2 made for `ChannelId` — so
Atrium sends all of them, at the values a local file measurably has. The three that never appeared
are the three that are not facts about a file: `TranscodingUrl` and `TranscodingContainer` are
answers to a negotiation, and `DefaultSubtitleStreamIndex` is a **per-user** selection driven by
that account's subtitle mode and remembered choices, which v1 does not record.

**Streams** carry codec, profile, level, bit depth, frame rate, resolution, colour and HDR
information, channel layout, sample rate, language, and the default/forced/external flags.
`[spec: MediaStream]`

> **A stream's frame rates and level are 32-bit numbers, and the wire shows it.** The reference
> writes the shortest decimal that reads back as the same single, so `24000/1001` arrives as
> `23.976025` rather than as the seventeen digits a double prints, and a whole rate arrives as
> `25` rather than `25.0` — a difference no parser sees and every byte comparison does. Same
> probe.
>
> **Three families of stream property are deliberately not emitted**, each for a different
> reason, and each is a bounded gap rather than a decision that they do not exist. `DisplayTitle`
> and the five `Localized*` properties are a *localised* rendering of a track — measured as
> `Español - MP3 - Stereo - Predeterminado` on a Spanish-configured server — and need the
> server's own localisation table; an English approximation would differ from the reference on
> every track rather than be absent on it. `IsAVC`, `TimeBase` and `NalLengthSize` are read from
> the demuxer and are not among the fields inspection records. `IsTextSubtitleStream`,
> `SupportsExternalStream`, `DeliveryMethod` and `DeliveryUrl` describe how a subtitle would be
> delivered, and v1 delivers none. **The first two are emitted since 011 T2 (2026-08-30)**, which
> found them to be facts about the file rather than about a delivery — stated by the reference on
> every stream of every kind, and read off a codec spelling 008 was storing under the file's own
> name rather than the reference's. The other two are still owed, to 011 T9.

> **A multi-part film's parts are media sources here and separate items in the reference.** The
> reference builds a source per *item* — itself plus its linked and local alternate versions —
> and a stacked film's later parts are none of those: they are counted in `PartCount` and fetched
> from `GET /Videos/{id}/AdditionalParts`, an endpoint outside v1's surface
> `[source: MediaBrowser.Controller/Entities/Video.cs:533-563, MediaBrowser.Controller/Entities/
> BaseItem.cs:1096,1120 @ v10.11.11]`. 003 §3.3 already merged the parts into one item with one
> source per part, so this is that model reaching the wire rather than a choice made here, and no
> library reachable from this repository has a multi-part film to measure the reference's answer
> on. A client sees one item that offers two sources instead of one item that offers one source
> and a part count; both are playable, and the second part is addressable either way.

> **`HasSubtitles` counts only the streams inside the container.** The reference sets it from
> everything the file's inspection produced, which for it includes the subtitle files sitting
> beside the media `[source: MediaBrowser.Providers/MediaInfo/FFProbeVideoInfo.cs:275 @
> v10.11.11]`. v1 inspects no sidecar, so a film with an external `.srt` and no embedded track
> answers nothing where the reference answers `true`. Bounded, and it closes when something reads
> sidecar subtitles.

> **Item-level `Container` is sometimes a demuxer list.** The reference reports one normalised
> container string at item level, and whether it names one container depends on the format: a
> Matroska file answers `"mkv"`, and every member of the mp4 family answers the whole six-name
> list `"mov,mp4,m4a,3gp,3g2,mj2"` `[probe: tools/probe_media_container.py, Jellyfin 10.11.11,
> 2026-08-29]`. Atrium reproduces the string. A client reading the item-level field of an mp4
> expects the list form, and "fixing" it would be a delta.
>
> **The single container a source reports is not a property of the file.** It is derived from
> that string per response, and the two routes derive it differently. On `/Items` **no profile is
> involved**: the single form is the file's own extension where the list contains it — the same
> six-name list answers `mp4` for a `.mp4` and `m4a` for a `.m4a` — and the list's first member
> where it does not `[source: Emby.Server.Implementations/Dto/DtoService.cs:316-352 @ v10.11.11]`.
> In a negotiation it is the first member the `DeviceProfile` accepts, and a **profile-less**
> negotiation leaves the list alone: the same `.m4a` that answers `m4a` on a listing answers the
> full list on `GET /Items/{itemId}/PlaybackInfo` `[probe: tools/probe_media_container.py,
> Jellyfin 10.11.11, 2026-08-29]`, `[probe: tools/probe_playback_info.py, Jellyfin 10.11.11,
> 2026-08-28]`. What inspecting a file can establish is the normalised string; each single form is
> the answer to a particular request and belongs to that response.

**Inspection is cached** and re-run only when the file changes. Probing an entire library on every
request is not viable, and probing on first playback makes the first play of every item slow.

**A file that has never been inspected still has a source.** The reference emits one for it, with
an empty stream list and the container taken from the file's extension `[source:
MediaBrowser.Controller/Entities/BaseItem.cs:1200-1207 @ v10.11.11]`, so an item whose inspection
failed or has not run yet is visible and unplayable rather than absent. A scan records the failure
the way it records a file whose name said too little (003 §3.8) and never drops the item over it.

**`ETag` is derived from the file's time of last change**, not from its bytes, so it moves when
the file is touched and not when the same content is written again — the same signal the cache
above turns on. Its exact derivation is recorded in [plan §6.1](plan.md#61-inspection-and-the-cache),
proven by recovering a real file's modification time from a tag the reference sent
`[probe: tools/probe_media_source.py, Jellyfin 10.11.11, 2026-08-29]`.

### 3.2 `POST /Items/{itemId}/PlaybackInfo` — `GetPostedPlaybackInfo`

**Consumers:** video-client. The negotiation entry point.

**Request body:** `UserId`, `MaxStreamingBitrate`, `StartTimeTicks`, `AudioStreamIndex`,
`SubtitleStreamIndex`, `MaxAudioChannels`, `MediaSourceId`, `DeviceProfile`, and the
`EnableDirectPlay` / `EnableDirectStream` / `EnableTranscoding` / `AllowVideoStreamCopy` /
`AllowAudioStreamCopy` switches. `[spec: PlaybackInfoDto]` **The whole body is optional** — a
request that carries none at all is answered rather than refused `[probe: manual requests via
tools/_probe.py, Jellyfin 10.11.11, 2026-08-29]` — while an unrecognised token *inside* one is a
`400`, which is the opposite of what an unrecognised query token does (§1.12 of the behaviours
document). `AudioStreamIndex` is applied only when `MediaSourceId` names the source it is about.

**Response — 200**

```json
{
  "MediaSources": [ ],
  "PlaySessionId": "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6"
}
```

Two properties, not three: an absent `ErrorCode` is **absent**, not `null`, under the global
null-suppression this server already reproduces. And the third shape has the opposite pair — an
empty source list answers `{"MediaSources": [], "ErrorCode": "NoCompatibleStream"}` with **no
`PlaySessionId`**, because one is issued only where there is something to play `[probe:
tools/probe_playback_info.py, Jellyfin 10.11.11, 2026-08-29]`.

Each returned media source is **annotated with the decision** for this client: the support flags
set to what this profile can actually do, and `TranscodingUrl` populated when the answer is
"fetch it from here instead of directly".

**A request with no `DeviceProfile` is not a request with no profile.** The negotiation falls back
to the profile the calling device stored through `POST /Sessions/Capabilities/Full`, so the same
bare request answers direct play before a client posts its capabilities and a `TranscodingUrl`
after `[probe: tools/probe_playback_info.py, Jellyfin 10.11.11, 2026-08-29; source:
Jellyfin.Api/Controllers/MediaInfoController.cs:137-147 @ v10.11.11]`. The `GET` below is
unaffected — it is profile-less by construction, measured on the same session. A client that
describes itself once and then negotiates with a bare body is what this exists for, and a server
without it hands that client a file it never said it could open.

**The annotation is per request, and the switches are not equals.** `EnableDirectPlay: false`
on a profile the source satisfies flips `SupportsDirectPlay` to `false` and produces a
`TranscodingUrl` — the flags describe *this* negotiation, not the source. `EnableTranscoding:
false` on a profile that forces a transcode changes **nothing**: the `TranscodingUrl` arrives
anyway. And `SupportsDirectStream` never answers independently — the reference disables its
direct-stream path outright ("direct-stream http streaming is currently broken"), so the flag
mirrors `SupportsDirectPlay` on every answer. `[probe: tools/probe_playback_info.py, Jellyfin
10.11.11, 2026-08-28; source: Jellyfin.Api/Helpers/MediaInfoHelper.cs:251-268 @ v10.11.11]`
A client that branches on `SupportsDirectStream` is branching on direct play, and Atrium keeps
the mirror rather than resurrecting a distinction no reference answer draws.

**`PlaySessionId` ties everything together** — this negotiation, the delivery request that follows,
and the three reports of 007. It is what makes `DELETE /Videos/ActiveEncodings` able to stop the
right thing.

**Errors**

| Condition | Status / body |
|---|---|
| Unknown or invisible item | `404`, problem details — byte-identical to `/Items/{itemId}`'s own refusal for the same identifier, on both routes `[probe: tools/probe_playback_info.py, Jellyfin 10.11.11, 2026-08-29]` |
| Unauthenticated | `401`, **empty**: no body, no `Content-Type`. Refused before the route runs, which is why it is not the problem-details shape above. Same probe |
| User lacks `EnableMediaPlayback` | `200`, **the negotiation unchanged** — see below |
| No source can be played by this profile | `200`, **not** a `4xx` — and **no** `ErrorCode`: the refusal is the source's own capability flags |

The first two rows carried no citation until the routes were implemented, and measuring them
turned up the one identifier that is neither: the **all-zeros** form is the reference's
`Guid.Empty` and never reaches a lookup at all, because a guard throws before it
`[source: Emby.Server.Implementations/Library/LibraryManager.cs:1359-1362 @ v10.11.11]` — so it
answers the controller's `400` in plain text where any other unowned identifier answers `404`.
That edge is already recorded as not reproduced (behaviours §1.11, and 006 §3.2's own table): this
server has no root-folder item, so the identifier is simply unknown here.

The last two rows are the important ones, and neither survived in its first wording.

The refusal row said an `ErrorCode` arrives, and none does. A profile that can play nothing gets
`200` with `SupportsDirectPlay`, `SupportsDirectStream` and `SupportsTranscoding` all `false`, no
`TranscodingUrl`, and no `ErrorCode` — measured in four request shapes, transcoding allowed and
forbidden `[probe: tools/probe_playback_refusal.py, Jellyfin 10.11.11, 2026-08-28]`. A `4xx`
would still be read as a transport failure; what a client actually branches on is the flags.

The `EnableMediaPlayback` row said `403`, and **no playback route consults the permission at
all**. A user whose policy denies playback negotiates exactly as anyone else — same flags, same
`TranscodingUrl`, no error `[probe: tools/probe_playback_info.py, Jellyfin 10.11.11,
2026-08-28]`. The flag's only consumers at 10.11.11 are the item DTO's `PlayAccess` property and
the remote-control `Play` command `[source: MediaBrowser.Controller/Entities/BaseItem.cs:1057,
Emby.Server.Implementations/Session/SessionManager.cs:1321 @ v10.11.11]`. A `403` here would
have been an invented refusal — a delta a policy-restricted client would observe.

**`ErrorCode` has one real value.** The schema's vocabulary is `NotAllowed`,
`NoCompatibleStream` and `RateLimitExceeded` `[spec: PlaybackInfoResponse]`, but the reference
has exactly one assignment site: `NoCompatibleStream`, set when the **media source list is
empty** `[source: Jellyfin.Api/Helpers/MediaInfoHelper.cs:123 @ v10.11.11]`. The other two are
dead members no response can carry. Atrium emits `NoCompatibleStream` in the same one place and
nothing else anywhere. What a v1 request reaches that place by is a **`MediaSourceId` naming no
part of the item**: the list is filtered to it, nothing survives, and the answer carries the code
and no `PlaySessionId` `[probe: tools/probe_playback_info.py, Jellyfin 10.11.11, 2026-08-29]`.

**`GET /Items/{itemId}/PlaybackInfo`** is the profile-less variant, included by design. Without a
profile there is nothing to negotiate against, so it returns the sources with their intrinsic
capabilities — all three flags `true`, no `TranscodingUrl` — and still issues a `PlaySessionId`
`[probe: tools/probe_playback_info.py, Jellyfin 10.11.11, 2026-08-28]`. It stays profile-less even
for a device whose stored capabilities carry one, which is what makes the fallback above the
`POST`'s alone — measured on the one session, answering both ways in the same minute `[probe:
tools/probe_playback_info.py, Jellyfin 10.11.11, 2026-08-29]`.

### 3.3 The decision

Given a media source and a device profile, exactly one outcome:

| Outcome | Meaning | Cost |
|---|---|---|
| **Direct play** | Client fetches the original file | Nothing |
| **Direct stream (remux)** | Container rewritten, streams copied | Near zero |
| **Transcode** | At least one stream re-encoded to something the profile accepts | A decode and an encode per frame |
| **Not playable** | No output this server can produce satisfies the profile | — |

**Evaluation order**, and it stops at the first success:

1. **Direct play** — the profile lists this container with these codecs, and every codec condition
   (profile, level, bit depth, channels, sample rate, bitrate, resolution) holds. Also requires the
   source bitrate to be within `MaxStreamingBitrate`.
2. **Remux** — a container the profile accepts exists into which these elementary streams can be
   copied unchanged. Codec conditions still have to hold: remuxing does not fix an unsupported
   codec, it only fixes the wrapper.
3. **Transcode** — some output this server can produce satisfies the profile: a container it
   accepts, holding codecs it accepts, within every ceiling it declared. Only the streams that fail
   a condition are re-encoded (§3.4).
4. **Not playable** — the profile accepts no container, or no codec, that v1 can produce. The
   answer is the capability flags all `false` (§3.2) — not an error, and not an `ErrorCode`.

**A numeric ceiling is compared against the number the server holds, not the number it printed.**
§3.1 records that a stream's frame rates and level reach a client as 32-bit numbers, written as
the shortest decimal that reads back as the same value. The condition is evaluated against the
value itself, which is not that decimal: a stream reported as `23.975988` is held as
`23.975988388…`, so a client that declares a frame-rate ceiling of exactly the rate it read off
the wire is answered with a **transcode**, and one that declares a hair more is answered with
direct play `[probe: tools/probe_decision_ladder.py, Jellyfin 10.11.11, 2026-08-29]`. Atrium
compares the same number, because a ladder that compared the printed one would direct-play a
file the reference re-encodes — visible to any client that echoes back what it was told.

**On the wire, remux and transcode are one shape.** A remux answer is a `TranscodingUrl` like
any transcode's, with the elementary streams copied at delivery; `SupportsDirectStream` stays
`false` because it mirrors direct play (§3.2). What separates the two outcomes is what the
session does per frame `[probe: tools/probe_transcode_decision.py, Jellyfin 10.11.11,
2026-08-28]`.

**And the reasons list does not separate them either.** `TranscodeReasons` says why **direct
play** was refused and nothing more: a profile that rejects a codec for direct play while its
own transcoding target accepts that codec answers `VideoCodecNotSupported` over a stream that is
then copied. Which rung was reached is decided against the *transcoding* profile, not against
the reasons `[source: MediaBrowser.Model/Dlna/StreamBuilder.cs GetVideoTranscodeProfile @
v10.11.11]`, `[probe: tools/probe_decision_ladder.py, Jellyfin 10.11.11, 2026-08-29]`. The
common case — a container-only rejection — does answer `ContainerNotSupported` alone, which is
what this paragraph used to state as the rule.

**A refusal with nothing to blame is `DirectPlayError`.** A profile that lists no direct-play
entry at all, and a request carrying `EnableDirectPlay: false` against a profile the source
satisfies, both answer with that single reason — a member of the vocabulary's "Errors" group
arriving on an ordinary, successful negotiation. **The reasons are listed in flag-value order**,
which is not the order the vocabulary is declared in: `VideoLevelNotSupported` precedes
`VideoRangeTypeNotSupported` on the wire and follows it in the declaration. Same probe.

**"Not playable" is now a much smaller set**, and it is worth being precise about what is left in
it: a profile listing only containers or codecs this server cannot produce, a source whose streams
cannot be decoded at all, a source that is not readable, and a user whose policy forbids the step
that would have answered.

**The user's policy barely gates the ladder, and the spec's first draft said otherwise.** The
measured rule at 10.11.11: a **single** denied permission changes nothing at negotiation. For a
video item, `SupportsTranscoding` goes `false` only when `EnableVideoPlaybackTranscoding`,
`EnableAudioPlaybackTranscoding` **and** `EnablePlaybackRemuxing` are all denied at once; for an
audio item, `EnableAudioPlaybackTranscoding` alone decides `[probe:
tools/probe_playback_info.py, Jellyfin 10.11.11, 2026-08-28; source:
Jellyfin.Api/Helpers/MediaInfoHelper.cs:278-293 @ v10.11.11]`. Even the all-denied answer is
flags, never an `ErrorCode`. At **delivery** the enforcement is stranger still, and it is
**per stream rather than all-or-nothing**: a user denied `EnableVideoPlaybackTranscoding` has
the video stream **force-copied "regardless of whether it will be compatible or not"**, and a
user denied `EnableAudioPlaybackTranscoding` has the audio stream force-copied the same way,
each against its own permission `[source:
MediaBrowser.Controller/MediaEncoding/EncodingHelper.cs:7136-7166 @ v10.11.11]` — an output
that can violate the profile it was negotiated for. Atrium replicates the negotiation rule
exactly (the all-three gate, flags not errors); at delivery it refuses those same two steps
rather than copying, and so never ships an output that violates the negotiated profile — a
force-copied incompatible stream fails at the client's decoder, and no client can *depend* on
receiving a broken stream. Of the request body's switches, `EnableDirectPlay` is honoured and
`EnableTranscoding` is ignored (§3.2); Atrium reproduces both.

**Two limits on that delivery-time reading, both read at the tag rather than assumed.** It
belongs to a **video** request and only to one: the force-copy is reached from the branch that
runs when a request carries video parameters `[source:
Jellyfin.Api/Helpers/StreamingHelpers.cs:198 @ v10.11.11]`, so an audio-only delivery consults
neither permission and re-encodes for an account denied audio transcoding exactly as it does
for a permitted one. And `EnablePlaybackRemuxing` has **no** delivery-time reader at all: a copy
is what a denied account is *given*, so there is nothing there to refuse. Atrium's refusal is
therefore scoped to where the reference would have force-copied — the two streams of a video
delivery — and the audio routes carry it nowhere.

**Three rules that prevent the classic failures:**

- **A profile that says nothing means "anything" — and a profile that says *nothing at all* is a
  different thing from an absent one.** An **absent** `DeviceProfile` is a client that has not
  told us, and the answer is direct play with every capability flag true. Reading absence as
  prohibition is how a server ends up refusing to play anything to a simple client. An **empty**
  profile is not absence: it is a client that listed no container, no codec and no target, and
  the answer is every flag false, no `TranscodingUrl` and no `ErrorCode` — the same refusal a
  profile that can play nothing gets, because it is one `[probe:
  tools/probe_decision_ladder.py, Jellyfin 10.11.11, 2026-08-29]`. This paragraph read "empty or
  absent" until the empty half was measured, and a server that answered direct play to it would
  hand a client bytes it had said nothing about being able to open. **"Absent" is decided by the
  route, not here**: a `POST` whose body names no profile is negotiated against the one its device
  stored, and only a device that stored none is a client that has not told us (§3.2).
- **Never claim a capability that is not there.** `SupportsTranscoding` is `true` on a source
  exactly when this server can produce, for *this* profile, a stream the profile accepts — and
  `false` otherwise, including when the profile's ceilings leave nothing producible. It is a claim
  about this negotiation, not a boast about the server. Advertising it and then failing at delivery
  time turns a clear "cannot play this" into a spinner that never resolves, which is strictly worse
  than the honest refusal v1 used to give.
- **Never remove what the client said it can handle.** A stream copy alters the container, not the
  content. Where a profile declares support for a metadata format — a dynamic HDR variant, a
  coexistence range type — that declaration is honoured, and nothing is filtered out of the
  bitstream on the client's behalf. Stripping metadata a client explicitly asked for is how the
  reference breaks Dolby Vision playback on one whole client platform
  ([behaviours §3.4](../../docs/compatibility/behaviours.md#34-hdr10-metadata-stripped-from-clients-that-asked-for-it--class-b-no-compensation)),
  and it is a defect Atrium would have to write on purpose in order to have.

### 3.4 Transcoding

Reached only when direct play and remux have both failed. Everything in this section is in service
of one idea: **do the least re-encoding that makes this source playable for this profile.**

**Only the streams that fail a condition are re-encoded.**

| Stream | v1 behaviour |
|---|---|
| Accepted by the profile, every condition holding | Copied, never re-encoded |
| Rejected: codec, profile, level or bit depth | Re-encoded to a codec the profile accepts |
| Rejected: resolution, frame rate or bitrate ceiling | Re-encoded down to the ceiling |
| Audio rejected: channel count or sample rate | Downmixed or resampled to the ceiling |

A file whose video the profile accepts and whose audio it does not is the common case — a modern
video track with a surround audio track a browser cannot decode — and it costs an audio encode, not
a video one. A server that re-encodes both because one was wrong burns two orders of magnitude more
CPU than the job needs.

**Ceilings are limits, not targets.** A profile permitting 1080p and 8 Mbps applied to a 720p
3 Mbps source produces 720p at approximately the source bitrate. Nothing is ever upscaled,
up-sampled or given more bits than it arrived with: that spends CPU to make the output larger and
no better.

**The output satisfies the profile it was built for.** This is the whole obligation of the feature.
An output that violates a condition the client declared is worse than "cannot play this", because
the refusal happens at the client's decoder, far from the cause, and looks like a broken file rather
than an unsupported one.

**Work starts where the client asked to start.** A request carrying a start position begins
production at that position; it does not produce from the beginning and discard. Measured: a
segment at ~90% of a 2h22 film arrives in 0.9 seconds, and the session's progress jumps to the
seek point — the transcoder is restarted at the requested position, and the same holds for a
segment requested far ahead of the produced window `[probe: tools/probe_transcode_session.py,
Jellyfin 10.11.11, 2026-08-28]`. It holds on the progressive routes too, where there is no playlist
to seek in: a start position ten minutes into a half-hour source answers its first bytes in the same
1.2 seconds a start position of zero does `[probe: tools/probe_progressive_delivery.py, Jellyfin
10.11.11, 2026-08-29]`. **A copied stream restarts at the last keyframe at or before the position**,
because a copy cannot begin mid-GOP; a re-encoded one starts at the position itself.

**A re-encode never produces more than the profile allows and never more than arrived.** Every
ceiling — resolution, bitrate, channel count, sample rate, bit depth — is stated to the encoder only
where it asks for *less* than the source has. That is not an optimisation: a limit equal to the
source is not an instruction, and issuing it as one asks encoders for things they do not have. A
lossless 96 kHz source re-encoded to a codec that stops at 48 kHz is the case that shows it — told
to produce 96 kHz because the client stated no ceiling, the encode fails and the request answers
nothing.

**Throttling is an operator setting, off as shipped.** The reference's `EnableThrottling`
defaults to `false` — an idle client leaves the encoder producing the whole file — and when an
operator enables it, production pauses once it leads the last downloaded position by
`max(ThrottleDelaySeconds, 60)` seconds, 180 by default, resuming when the gap closes `[source:
MediaBrowser.Model/Configuration/EncodingOptions.cs:23-24,
MediaBrowser.Controller/MediaEncoding/TranscodingThrottler.cs:118-171 @ v10.11.11]`. Measured on
a throttled server: production stalls ~180 seconds ahead of the one fetched segment and stays
there `[probe: tools/probe_transcode_session.py, Jellyfin 10.11.11, 2026-08-28]`. Atrium ships
the same two knobs with the same defaults and the same pause-at-gap behaviour — the draft's
"always throttled" would have been an observable difference in how much of a file exists after
an abandoned session, and the operator who wants the bound can turn it on in both servers.

**A pause is not a stop, and the difference is what a client sees next.** The paused work
resumes when the client asks for the next segment: the request moves the furthest-fetched
position, the gap closes, and production continues from where it left off rather than starting
again. A server that ended the work at the gap instead would answer every resumed playback with
a fresh encode of material it had already produced, which is the opposite of what the knob is
for.

**A transcode is bounded work with an owner**: it belongs to a playback session (§3.8), it stops
when the session stops, when the client disconnects, and when the server shuts down, and its output
lives in scratch space that dies with the session (§3.8).

**Determinism is per session.** For a remux, the byte-identity rule holds globally: the same source
and parameters give the same segments. For a re-encode the guarantee is narrower and is the one
players actually need — **within a session, a segment that has been produced is served identically
every time it is requested again**, so a retry after a network failure is the same bytes. Across
sessions, re-encoded bytes may differ; no client compares them.

**What the client is told about it.** `SupportsTranscoding`, `TranscodingUrl`,
`TranscodingContainer` and `TranscodingSubProtocol` on the negotiated source describe this answer
and nothing else: they are per source *and* per profile, decided by the negotiation that returned
them (§3.3).

> **Hardware acceleration is absent, and no client can tell.** The output of a CPU encode and a
> hardware encode are both just a stream that satisfies the profile. What differs is how many
> concurrent sessions a given machine survives — an operational property, not a protocol one. This
> is why it can arrive in a later version without touching this specification's observable
> behaviour, and it is the reason the exclusion is safe in a project whose first principle is that
> a client cannot tell.

### 3.5 Delivery: the rules that apply to every route

**Byte ranges are mandatory.**

| Requirement | Behaviour |
|---|---|
| `Accept-Ranges: bytes` | On every delivery response that carries a body of media bytes whose size is known. **Not on the two playlists**: they are sized and carry no range unit, measured on both `[probe: tools/probe_hls.py, Jellyfin 10.11.11, 2026-08-29]` |
| `Range: bytes=a-b` | `206` with a correct `Content-Range` and exactly the bytes asked for |
| A range naming the whole body, `bytes=0-{size-1}` | `206`, never a `200` |
| An open-ended range `bytes=a-` | `206`, from there to the last byte |
| A range overshooting the end | `206`, clamped to the last byte |
| Suffix range `bytes=-n` | `206` with the last `n` bytes, and the whole body where `n` exceeds it |
| Multiple ranges | The full body as `200` — the reference does not split |
| Reversed range `bytes=b-a` | The full body as `200`, not a `416` |
| An unreadable `Range` of any shape | The full body as `200`, not a `416` |
| Unsatisfiable range — past the end, or `bytes=-0` | `416` with `Content-Range: bytes */total` and `Content-Length: 0` |
| No `Range` | `200` with the full body |

The whole table is measured, not designed: the matrix runs against a direct-play
`/Videos/{itemId}/stream?static=true` — `bytes=100-199` answers `206` with
`Content-Range: bytes 100-199/{size}` and a `Content-Length` of exactly `100`; the suffix form
answers the last bytes; the multi-range, reversed and no-`Range` shapes all answer `200` with
the full body; one byte past the end is the `416` with `Content-Length: 0`.
`[probe: tools/probe_range_matrix.py, Jellyfin 10.11.11, 2026-08-29]`

**The unreadable row is the one that had to be measured**, because the careful reading of RFC 9110
gives the opposite answer. Five shapes of nonsense — a value with no unit at all, `bytes=`,
`bytes=-`, `bytes=abc-def` and `bytes=100-abc` — each answer `200` with the entire file, as does a
reversed range. A server that refused them with `416`, which is what the RFC invites, would refuse
requests the reference serves.

**A static response carries exactly four headers**, and the absences are as measured
as the presences: `Content-Length`, `Content-Type`, `Accept-Ranges: bytes` and `Last-Modified`,
with `Content-Range` added on a `206` and on the `416`. There is no `ETag`, no
`Content-Disposition`, no `Cache-Control`, and **no conditional handling at all** — a request whose
`If-Modified-Since` lies in the future is answered with the whole film, not a `304`.

**`Content-Length` is sent whenever the size is known** — always for direct play, for remuxed
output whenever it can be computed or the output is produced to a seekable location first, and for
every HLS segment, which is finished before it is served.

**For HLS this is parity, not a divergence.** The measurement narrowed the draft's claim: the
reference's finished segments already answer with `Content-Length`, `Accept-Ranges: bytes` and
byte-identical retries, and its playlists carry a length too `[probe: tools/probe_hls.py,
Jellyfin 10.11.11, 2026-08-28]`. What stays chunked and sizeless on the reference is the
**progressive** family — `/stream` remuxes and re-encodes, `/universal` over http — which
forces `Accept-Ranges: none` and no length even where the size is knowable, a remux to a
seekable location included ([behaviours §3.3](../../docs/compatibility/behaviours.md#33-progressive-transcoding-responses-carry-no-content-length-or-accept-ranges--class-c)).

**The one delivery in v1 that cannot carry a size** is a progressive (non-HLS) re-encode, where the
final length is not known until the last frame is produced. That response is chunked, exactly as the
reference's is. The rule is *send the size when it is known*, never *invent one*: a wrong
`Content-Length` truncates playback, which is a worse failure than the missing header this project
went out of its way to fix.

**On a chunked answer a `Range` header decides nothing, whatever shape it has.** The sized case has
five unreadable shapes and two answers; this one has a single answer for every shape there is —
`bytes=100-199`, a suffix, a single byte and `bytes=abc-def` all reply `200` with no `Content-Range`
and the body from its first byte `[probe: tools/probe_progressive_delivery.py, Jellyfin 10.11.11,
2026-08-29]`. A response that has said `Accept-Ranges: none` is answering a client that asked
anyway, and it answers by ignoring the question rather than by refusing it.

> **The deliberate divergence is now exactly one route family.** Where a progressive remux's
> size is knowable, Atrium sends it and honours `Range`; the reference answers chunked with
> `Accept-Ranges: none` `[source:
> Jellyfin.Api/Helpers/FileStreamResponseHelpers.cs:123-135 @ v10.11.11]`. That gap is why every
> client that casts to a DLNA renderer runs a local sizing proxy: a renderer will not touch a
> stream whose size it does not know. A client cannot branch on a response being *more* correct,
> so Principle I is not violated — and the HLS half of the draft's divergence dissolved into
> parity when it was measured.

**`static=true`** requests the original bytes with no processing, and the reference honours the
*bytes* absolutely — including past the URL. `stream.mp3?static=true` on a FLAC source answers
`200` with the untouched FLAC bytes behind a `Content-Type: audio/mpeg` label, and
`stream.mkv?static=true` on an mp4 film serves the mp4 bytes as `video/x-matroska`: the path's
container decides the label and nothing else, and no error, remux or re-encode ever happens on a
static request `[probe: tools/probe_range_matrix.py, Jellyfin 10.11.11, 2026-08-29]`
(behaviours §2.20). The draft said a mismatch would be an
error; it is not, and inventing one would break the client that names a wrong container while
downloading — it still receives, correctly, the original file. Atrium replicates: static always
serves the source bytes, whatever the path says.

**The container is the label and nothing more, in three parts.** It arrives either as the path's
suffix or as a `container` query parameter, and both answer identically. A container the server has
no label for is **not** an error: the label falls back to the file's own extension, so
`stream.banana?static=true` on an mp4 answers `video/mp4` over the same bytes. And a container
outside the reference's own spelling rule — anything but letters, digits and `-._,|`, or longer
than forty characters — is a validation `400` keyed on `container`, refused *before* the item is
looked up, so an unknown item behind an illegal container answers the `400` and not the `404`.
Every mapping is measured across the whole extension set a library admits, and several are not
guessable: `.opus` is `audio/ogg`, `.alac` is `audio/mp4`, `.mts` is `model/vnd.mts`.

**Authentication is accepted on every mechanism and required on none.** The four `stream` routes
answer identically to a request carrying no token at all, one carrying a token nothing issued, and
one carrying `?api_key=` — measured on all four, in the same run in which `/Audio/{itemId}/universal`
answered `401` to the first two `[probe: tools/probe_range_matrix.py, Jellyfin 10.11.11,
2026-08-29]`. The split is per action rather than per feature, and it is the reference's
`[source: Jellyfin.Api/Controllers/AudioController.cs:89,
Jellyfin.Api/Controllers/VideosController.cs:312,
Jellyfin.Api/Controllers/UniversalAudioController.cs:94 @ v10.11.11]`. This is what
[002 §3.1](../002-authentication-users-and-sessions/spec.md#31-how-a-client-presents-a-token) already
recorded and deferred, and 008 replicates it: these URLs are handed to external players and image
loaders that set no headers, so a server that began requiring a token would break the one thing
they exist for. The consequence — **an item id is a capability on these routes** — is
[behaviours §2.10](../../docs/compatibility/behaviours.md#210-the-image-and-delivery-routes-accept-a-token-and-require-none)'s,
taken knowingly and identically to 006's decision for the image routes.

**Which part of the item is `mediaSourceId`'s, and it is the same question on both halves.** Absent,
the request is about the first part, which is what the reference serves when the parameter is not
given. Naming a part serves that part's bytes. And a value naming *no* part of the item is refused
identically on a `static=true` request and on a produced one — which is what makes part selection a
property of the route rather than of the processing behind it. The reference splits that refusal in
two: a well-formed identifier that matches nothing is a `400` in the third error shape, and a value
that is not an identifier at all is a `500` in the same shape, because the fallback comparison
parses the string before comparing it `[probe: tools/probe_progressive_delivery.py, Jellyfin
10.11.11, 2026-08-29]`. **Atrium answers the `400` to both**: the two values mean the same thing,
and the `400` is the reference's own answer to that meaning one value away
([behaviours §3.9](../../docs/compatibility/behaviours.md#39-an-unparseable-mediasourceid-is-a-500-where-a-well-formed-one-is-a-400--class-a-diverged)).

**A produced request into a container the server cannot write is a `500`, and that is parity.**
Three shapes of one failure — a container no muxer exists for (`stream.banana`, `?container=banana`)
and one that cannot hold the streams it was handed (`stream.mp3` on a film) — each answer `500` in
the third error shape, with `Accept-Ranges: none` already on the response because the produced path
writes that header before it asks for anything. This is the failure-handling table's "the encoder
died" reached before the first byte rather than after it, and no decision is involved: a production
that cannot start is a `500` on both servers.

**What a produced request is muxed into, when the client names nothing.** The path's suffix or the
`container` parameter first; then the container the *requested codec* implies — `h264` means `ts`,
`hevc` and `av1` mean `mp4`, `vp8` and `vp9` mean `webm`, `aac` means `aac`, `mp3` means `mp3`; and
finally the **first member of the source's own stored container string**. That last step is a third
derivation of "the container" beside the two §3.1 records, and it is visible: a bare
`/Audio/{itemId}/stream` on an `.m4a` answers `Content-Type: video/quicktime`, because the stored
`mov,mp4,m4a,3gp,3g2,mj2` begins with `mov`, while the same request on an `.mkv` film answers
`video/x-matroska` `[probe: tools/probe_progressive_delivery.py, Jellyfin 10.11.11, 2026-08-29]`.

**A delivery route's own refusal is the third error shape, not problem details.** An identifier no
library holds answers `404` with `text/plain` — no charset — and the fixed 25-byte
`Error processing request.`, on all four `stream` routes; the same identifier on
`GET /Items/{itemId}/PlaybackInfo` answers the RFC 9457 body of §3.2's table
`[probe: tools/probe_range_matrix.py, Jellyfin 10.11.11, 2026-08-29]`. One feature, one
identifier, two bodies, split by which layer refused
([behaviours §1.11](../../docs/compatibility/behaviours.md#111-there-are-four-error-shapes-not-one)).

### 3.6 Audio delivery

| Route | Behaviour |
|---|---|
| `GET /Audio/{itemId}/stream` | The source, with `static=true` for direct play, remuxed to the requested container, or re-encoded when the requested container or codec cannot hold the source's streams |
| `GET /Audio/{itemId}/stream.{container}` | Same, container from the path |
| `GET /Audio/{itemId}/universal` | The server decides, from the client's stated constraints |

`/universal` accepts `container`, `audioCodec`, `maxAudioChannels`, `transcodingAudioChannels`,
`maxStreamingBitrate`, `audioBitRate`, `maxAudioSampleRate`, `maxAudioBitDepth`,
`transcodingContainer`, `transcodingProtocol`, `startTimeTicks`, `deviceId`, `userId`,
`mediaSourceId`, `enableRemoteMedia`, `enableAudioVbrEncoding`, `breakOnNonKeyFrames` and
`enableRedirection`. `[spec: GetUniversalAudioStream]`

**`container` is a list of lists.** It is split on commas first and each piece on `|` second, so
`opus,webm|opus,mp3,aac,m4a|aac,flac` states six containers, two of which restrict the codecs
allowed inside them `[source: Jellyfin.Api/Controllers/UniversalAudioController.cs:274-287 @
v10.11.11]`. **`transcodingProtocol` is not validated**: `hls` is matched case-insensitively and
anything unrecognised is answered as `http` rather than refused, so a request spelled `HLS`
reaches the playlists and one spelled `banana` reaches the progressive body — both `200`
`[probe: tools/probe_universal_audio.py, Jellyfin 10.11.11, 2026-08-29]`.

**A satisfied constraint set is direct play, and direct play here is the file**: `200`, a
`Content-Length` equal to the file, `Accept-Ranges: bytes`, a `Last-Modified`, and a mid-file
`Range` answered `206` with a correct `Content-Range` — the same four-header set §3.5 measures on
the `stream` pair, measured again on this route `[probe: tools/probe_universal_audio.py, Jellyfin
10.11.11, 2026-08-29]`.

**`/universal` re-encodes to meet a constraint the source violates** — a sample-rate ceiling, a
bit-depth ceiling, a channel ceiling, a codec the client cannot decode — **and the reference
then misses the target it was aiming at.** At 10.11.11 the output sample rate is not the ceiling
but the nearest step of the ladder Opus needs — `≤8000, ≤12000, ≤16000, ≤24000, else 48000` —
applied to **every** codec: a stated ceiling of 22 050 Hz is answered at 24 000 Hz, above what
the client declared `[probe: tools/probe_universal_audio.py, Jellyfin 10.11.11, 2026-08-28]`.
The restructure that scopes the ladder to Opus is merged upstream and in no 10.11.x — the same
family as behaviours §3.2's PCM fix. **Atrium honours the ceiling exactly**: a client that asked
for at most 22 050 Hz receives at most 22 050 Hz, because an output above a declared ceiling
fails at the client's decoder, far from the cause (recorded as the divergence in
[behaviours §3.7](../../docs/compatibility/behaviours.md#37-a-sample-rate-ceiling-is-answered-from-the-opus-ladder--class-b-diverged)).

**Of the three ceilings, two shape the output and the third only refuses a copy.** A sample-rate
ceiling and a channel ceiling become the produced stream's rate and channel count; a
`maxAudioBitDepth` below the source's stops the stream being copied and nothing downstream of
that ever states a sample format — the reference emits none, on any route `[source:
MediaBrowser.Controller/MediaEncoding/EncodingHelper.cs CanStreamCopyAudio @ v10.11.11]`. Atrium
does the same: the ceiling decides *that* a re-encode happens, and the target codec decides what
depth comes out. Stating one would be a third behaviour, and one that breaks the encoders whose
sample format is not a choice.

**A codec-less transcode request is the reference's other hole here.** `/universal` with a
`transcodingProtocol` of `http` and no `audioCodec` answers `200` with a `Content-Length: 0`
empty body — naming a `transcodingContainer` of `flac` and naming none at all `[probe:
tools/probe_universal_audio.py, Jellyfin 10.11.11, 2026-08-29]`. The transcoding profile is not
the codec-less part: it is built with `audioCodec ?? "mp3"` and negotiates fine. The **streaming
request** built after it carries the raw parameter, and a streaming request with no codec infers
one from the request path's extension — of which `/Audio/{itemId}/universal` has none, so the
whole path is what the inference is handed `[source:
Jellyfin.Api/Helpers/StreamingHelpers.cs:71-75 @ v10.11.11]`. **The encoder it ends up asking for
is `aac`**, not the path: the name is guarded by the reference's own container spelling pattern
and anything that fails it becomes `aac`, which a path full of separators does. So the empty body
is that encoder meeting a muxer that cannot carry it, and the hole is **container-dependent** — a
`wav` transcoding container, which can carry AAC, answers a real body. Nothing can be built on an
empty body behind a `200`; Atrium answers the request with a stream, giving that same inference
the transcoding container instead of a dotless path
([behaviours §3.8](../../docs/compatibility/behaviours.md#38-universal-without-audiocodec-answers-an-empty-200--class-a-diverged)).

**`enableRedirection` never redirects a local file, and the draft said otherwise.** The `302`
branch requires a source that is **remote** over HTTP, direct-playable, and a user with
`EnableRemoteMedia` — all at once `[source:
Jellyfin.Api/Controllers/UniversalAudioController.cs:175 @ v10.11.11]`; a library file is
protocol `File`, so a direct-play answer for anything a v1 library holds is proxied bytes with a
`200`, redirection enabled or not — measured `[probe: tools/probe_universal_audio.py, Jellyfin
10.11.11, 2026-08-28]`. Atrium accepts the parameter and, having no remote sources in v1, never
answers `302` — exactly the reachable subset of the reference's rule.

**This route's refusals are not its siblings'**, all three measured on one server in one run
`[probe: tools/probe_universal_audio.py, Jellyfin 10.11.11, 2026-08-29]`:

| Refusal | `/Audio/{itemId}/universal` | `/Audio/{itemId}/stream` |
|---|---|---|
| No credential | `401`, empty body (AC-32) | `200`, the bytes |
| An item nothing holds | `404`, **problem details**, byte-identical to `GET /Items/{itemId}`'s | `404`, `text/plain`, the fixed 25 bytes |
| A `mediaSourceId` naming no source | `400`, `text/plain`, the 25 bytes — for a well-formed identifier **and** for one that is not an identifier at all | `400` and **`500`** respectively (§3.5, behaviours §3.9) |

The item refusal differs because the universal controller resolves the item through the caller's
user and refuses with the framework's own not-found result, where the `stream` pair has no user
and throws out of its streaming helper `[source:
Jellyfin.Api/Controllers/UniversalAudioController.cs:124-128,
Jellyfin.Api/Helpers/StreamingHelpers.cs:111 @ v10.11.11]`. An **invisible** item is therefore
the same `404` as an unknown one here, and visible on this route at all only because this is the
one delivery route with a user to check against.

**The reference's PCM/WAV answers are broken at 10.11.11, and the break is split by whether the
request carried an `audioBitRate` rather than by which route was called** `[probe:
tools/probe_universal_audio.py, Jellyfin 10.11.11, 2026-08-29]`:

| Request | The reference answers |
|---|---|
| `stream.wav`, no codec named | `500` — the codec inferred from the extension is `wav`, and nothing encodes that |
| `stream.wav` or `stream?container=wav` with a `pcm_*` codec and **no** `audioBitRate` | `500` — the sample-rate argument is built from an absent field |
| the same **with** an `audioBitRate` | `200`, `audio/wav`, a body with **no RIFF header** |
| `/universal` whose **transcoding** container is `wav`, with a `pcm_*` codec and an `audioBitRate` | the same headerless body |
| `/universal` with `container=wav` and nothing else | mp3 — `container` is the direct-play list, not a target |

None of the broken answers carries a `Content-Length`, all answer `Accept-Ranges: none`, and every
`Range` on them is ignored (§3.5, behaviours §3.3). **Atrium answers each of the first four rows
with valid WAV** — a real RIFF header, a `Content-Length` equal to the body, `Range` honoured —
which requires producing the output somewhere seekable, because a WAV header written to a stream
that cannot be rewound states a length it does not know. The last row is parity: `container` is
what the client can play, not what it is asking to be sent, and both servers answer it with mp3.
Both divergences, and the risk carried by the second, are reasoned in
[behaviours §3.2](../../docs/compatibility/behaviours.md#32-pcmwav-output--one-bug-two-symptoms-two-classes);
the upstream fix is `jellyfin/jellyfin#17537`, merged to master and in no 10.11.x.

### 3.7 Video delivery

`GET /Videos/{itemId}/stream` and `/stream.{container}` behave as their audio equivalents.

**Remuxed and re-encoded video are both delivered over HLS**, through the same three routes. Which
of the two a client is receiving is a property of the negotiation, not of the URL — and a client
that only follows the playlist cannot tell, which is the point:

| Route | Returns |
|---|---|
| `/Videos/{itemId}/master.m3u8` | The master playlist: **one variant for this negotiation, and a standard-range entrance beside it where the video is copied and the source is high dynamic range** |
| `/Videos/{itemId}/main.m3u8` | The media playlist: the segment list |
| `/Videos/{itemId}/hls1/{playlistId}/{segmentId}.{container}` | One segment |

**The measured shape** `[probe: tools/probe_hls.py, tools/probe_transcode_decision.py, Jellyfin
10.11.11, 2026-08-28]`:

- The master playlist advertises **one** `#EXT-X-STREAM-INF` variant — never a ladder — whose
  `CODECS`, `RESOLUTION`, `FRAME-RATE` and `BANDWIDTH` describe the negotiated output, and whose
  URI is a relative `main.m3u8` carrying the entire query string forward. (The reference adds an
  `#EXT-X-IMAGE-STREAM-INF` trickplay entry when it has trickplay images; v1 has none, and a
  master playlist without one is a server with nothing to advertise there, not a different
  shape.)
- **A copied high-dynamic-range video is offered a standard-range entrance beside it**, and this
  is the one exception to the line above. Where the video is stream-copied *and* the source's own
  range is HDR, the reference appends a second `#EXT-X-STREAM-INF` describing the same picture
  re-encoded to h264 — `VIDEO-RANGE=SDR`, an `avc1.` codec string, the **same** `BANDWIDTH`,
  `AVERAGE-BANDWIDTH`, `RESOLUTION` and `FRAME-RATE` as the copy — whose URI is this
  negotiation's own address with the codec renamed and the copy switched off. Equal bandwidth is
  deliberate: with nothing to choose on rate, a client selects on colour range, which is the
  entire purpose of the entry. Two further entrances exist beside it, describing the same
  picture re-encoded to hevc or to av1, and each appears only where the copied codec is that
  codec **and the operator has permitted that encoder** (`AllowHevcEncoding`,
  `AllowAv1Encoding`) — permissions that ship off, so a server as shipped answers the copy and
  the h264 entrance alone. The measured master carried three because the measured operator had
  permitted hevc encoding
  `[probe: tools/probe_transcode_decision.py, Jellyfin 10.11.11, 2026-08-29]`.
- **A Dolby Vision or HDR10+ copy carries a `SUPPLEMENTAL-CODECS` attribute** on that first
  variant, and only on it: `dvh1.<profile>.<level>/db1p` for Dolby Vision over an HDR10 base
  layer, and the variant's own codec string followed by `/cdm4` for HDR10+. It is a property of
  the source's dynamic metadata rather than of the request — a client that declares no Dolby
  Vision support is sent the same attribute. The same distinction decides the four-character
  code inside the produced bytes: a Dolby Vision copy whose client *did* declare Dolby Vision is
  muxed with `dvh1`, and everything else with `hvc1`. **Neither is reproduced here**, because
  neither is reachable from what this feature can learn about a file: §3.1's inspection reads a
  stream's colour metadata, and the Dolby Vision and HDR10+ signals are elsewhere — so every
  source it can describe is one the reference sends no `SUPPLEMENTAL-CODECS` for either. The gap
  and what closes it are [behaviours §5.10](../../docs/compatibility/behaviours.md).
- The media playlist is `#EXT-X-PLAYLIST-TYPE:VOD`, `#EXT-X-VERSION:3`,
  `#EXT-X-MEDIA-SEQUENCE:0`, ends with `#EXT-X-ENDLIST`, and arrives **complete in a fraction of
  a second, before any segment exists** — 2 843 segments in 0.18 s. The boundaries are predicted
  from the source, not derived from produced output; this is what makes rule 1 below possible at
  all.
- Every `#EXTINF` line ends `, nodesc`, and every segment URI repeats the full query plus two
  per-segment parameters: `runtimeTicks` (the segment's cumulative start offset) and
  `actualSegmentLengthTicks` (its exact duration). **`runtimeTicks` is what decides the bytes**,
  and the index in the path decides only what the produced file is called: segment 0's own path
  asked for at the middle of a film answers the middle of the film, measured as two different
  digests from one path `[probe: tools/probe_transcode_session.py, Jellyfin 10.11.11,
  2026-08-29]`. The two agree for every URI a playlist writes, which is why the difference is
  invisible until something hand-writes one.
- **A segment is a finished file, answered as one**: `200`, a `Content-Length` equal to its body,
  its container's `Content-Type`, `Accept-Ranges: bytes`, a `Last-Modified`, and no `ETag` — the
  static answer's header set exactly, and a mid-segment `Range` answers `206` with a correct
  `Content-Range`. Neither of the two path parameters beside the index decides anything:
  `playlistId` is unused, and the path's own container is not what the segment is muxed into —
  `0.mp4` asked for while `SegmentContainer=ts` answers MPEG-TS bytes labelled `video/mp2t`.
- **`SegmentContainer=mp4` makes the playlist version 7** and gives it an `#EXT-X-MAP` line
  naming a segment numbered **-1**, with `runtimeTicks=0` and `actualSegmentLengthTicks=0`. That
  segment is the fMP4 initialisation header rather than a position in the film: it answers `200`,
  `video/mp4`, and begins `ftyp`.
- Both playlists carry a `Content-Length` and are labelled `application/vnd.apple.mpegurl`; the
  master carries `Expires: 0` and the media playlist carries none, and **neither carries
  `Accept-Ranges`** — which is where §3.5's range rule stops, read back off both header sets. `#EXT-X-TARGETDURATION` is the
  **longest** segment rounded up, not the requested length — a copy asked for at five seconds
  whose longest bucket is 5.045 s declares `6`
  `[probe: tools/probe_hls.py, Jellyfin 10.11.11, 2026-08-29]`.

**These two routes require a token where the four `stream` routes require none**, and their other
refusals are the `stream` pair's rather than `/universal`'s — all four measured on both playlist
routes in one run `[probe: tools/probe_hls.py, Jellyfin 10.11.11, 2026-08-29]`:

| Refusal | `master.m3u8` and `main.m3u8` |
|---|---|
| No credential | `401`, empty body |
| An item nothing holds | `404`, `text/plain`, the fixed 25 bytes |
| A `mediaSourceId` naming no source | `400`, `text/plain`, the same 25 bytes |
| No query string at all | **not a refusal**: a copy is planned at the copy default and a playlist is answered |

**The segment route takes the same three and adds two of its own**, measured in one run
`[probe: tools/probe_transcode_session.py, Jellyfin 10.11.11, 2026-08-29]`:

| Refusal | `hls1/{playlistId}/{segmentId}.{container}` |
|---|---|
| No credential | `401`, empty body |
| An item nothing holds | `404`, `text/plain`, the fixed 25 bytes |
| A `mediaSourceId` naming no source | `400`, `text/plain`, the same 25 bytes |
| A start position beside the segment's own | `400`, the same 25 bytes — a segment states where it begins, and two positions in one request have no meaning |
| No query string at all | **the framework's own**: `runtimeTicks` and `actualSegmentLengthTicks` are required, so this is problem details naming both — the one refusal on these three routes that is not the third shape, and the opposite of what the same treatment of `main.m3u8` answers |
| A `playlistId` nothing named | **not a refusal**: it decides nothing and the segment is served |

Rules:

1. **Segments are deterministic.** The same source with the same parameters yields the same segment
   boundaries every time, so a segment can be re-requested after a network failure and be the same
   bytes. A server that re-derives boundaries per session cannot serve a retry. For re-encoded
   output the byte-identity half of this holds **within the session** (§3.4); the boundaries hold
   always.
2. **Segment duration is uniform** except for the last, and the playlist's declared duration matches
   what is delivered. Players build their seek bar from this. **The reference does not do the
   second half**, which was found by stating the grid to an encoder rather than reading it: it
   scales the length for the playlist and states the unscaled integer to ffmpeg, so a segment
   declaring 3.004 s holds 3.000 s and a long film's playlist claims eleven seconds it does not
   have. This criterion stands as written and the divergence is recorded and argued in
   [behaviours §3.10](../../docs/compatibility/behaviours.md); the playlist itself is byte-identical
   on both servers, so what moves is inside media nobody compares. The number itself depends on the
   path: the same film measured 3.004 s per segment re-encoded and 6.0 s per segment
   stream-copied, each uniform within its session — and **neither number means what it looks
   like**, which is the reading this section owed and 008 T10 paid
   `[probe: tools/probe_hls.py, Jellyfin 10.11.11, 2026-08-29]`.

   The 3.004 s is a requested three seconds scaled up so a whole number of frames fits a segment:
   `ceil(3000 × ceil(rate) ÷ rate)` milliseconds, over the rate the *request* states and only
   where the video is re-encoded. The rate is the one the film's container **stores**, and this
   film stores 23.975988; a source at an exact 24000/1001 — 23.976025 — answers **3.003 s** from
   the same arithmetic. Measured at five requested lengths, one rate: 1 s → 1.002, 2 s → 2.003,
   3 s → 3.004, 5 s → 5.006, 10 s → 10.011.

   The 6.0 s is the **copy path's own default segment length**, laid as an equal grid over the
   runtime — not the source's keyframes. A copy's boundaries follow the file's real keyframes only
   where the operator has permitted on-demand keyframe extraction for that container, and the
   shipped permission names Matroska alone (`AllowOnDemandMetadataBasedKeyframeExtractionForExtensions`,
   `["mkv"]`). Asked for at five seconds, the measured mp4 answers ten segments of exactly 5.0 s
   and a Matroska file beside it answers 5.045, 5.0, 5.0 … — the bucketing, visible only where it
   is allowed. The unrequested default is three seconds re-encoded and six copied.
3. **A segment requested out of order is served.** Players seek; they do not walk the playlist.
4. **The playlist is complete and marked ended** for a finite source. A live-style rolling playlist
   would make the file appear unseekable.

### 3.8 Session lifecycle

Every remux and every transcode belongs to a **playback session**, keyed by `PlaySessionId`, with
an owning user and device. Nothing that costs CPU or disk exists outside one: that is what makes it
stoppable, countable and reapable.

| Event | Effect |
|---|---|
| Delivery request with a `PlaySessionId` | Session created or reused |
| `DELETE /Videos/ActiveEncodings` with the id | Session and its work stopped |
| Client stops requesting segments | Production pauses at the throttle margin, session reaped after an idle timeout |
| Client disconnects mid-response | Work stopped immediately |
| Server shutdown | All sessions stopped, scratch space cleared |

**`DELETE /Videos/ActiveEncodings` must actually stop something.** Clients call it when the user
stops. The reference is called by real clients for exactly this reason, and a server that answers
`204` while leaving work running accumulates processes until the machine dies. Answering `204`
without acting is worse than not implementing the route, because it looks correct.

**Its parameters are both mandatory, and only one of them decides anything.** `deviceId` and
`playSessionId` are each required — omitting either answers a validation `400` naming the missing
field, and omitting both names both — and the well-formed call answers `204` after which the
named session stops reporting progress `[spec: StopEncodingProcess; probe:
tools/probe_transcode_session.py, Jellyfin 10.11.11, 2026-08-29]`. It stops one session, the
named one, not everything the device owns. **The device is not part of the naming**: a call
carrying a device the server has never seen still stops the session its `playSessionId` names,
and a call carrying a `playSessionId` nothing issued leaves a live session running and still
answers `204`. The sentence read "the session's `TranscodingInfo` gone from `/Sessions`
immediately after" and named the device as half the key, until a battery measured both: what
goes is the completion percentage rather than the object (§3.8's last paragraph), and the device
is bound and then ignored.

**The session's own report outlives its work, and Atrium's does not.** While a transcode runs,
`/Sessions` carries a `TranscodingInfo` naming the output's codecs, container, size, channel
count and the reasons direct play was refused. When the work stops — by the route above, by the
kill timeout, or because the encoder died — the reference leaves that object in place, less its
completion percentage and its encoding frame rate, until a playback report says the item is no
longer being transcoded. Atrium carries the report for exactly as long as it owns the work and
omits it afterwards, which is the divergence argued in
[behaviours §3.11](../../docs/compatibility/behaviours.md), together with the two properties it
never sends because it does not read its encoders' progress.

**A session nobody asks about dies on a timer, and the timer is a minute.** The reference keeps
one timeout per job and chooses it by a single property — ten seconds for a progressive stream,
sixty for everything else `[source:
MediaBrowser.MediaEncoding/Transcoding/TranscodeManager.cs:153-160 @ v10.11.11]` — and every job
behind a segment request is one of the latter, because a progressive stream stops with the
response it belongs to. Measured end to end: a session whose client fetched one segment and then
went quiet stopped between 58 and 60 seconds later `[probe:
tools/probe_transcode_session.py, Jellyfin 10.11.11, 2026-08-29]`. Every request the session's
routes answer restarts that minute.

**Scratch space is reclaimed the way the reference reclaims it**: by session — on the stop
route, and when a session goes unpinged past its kill timeout, the partial output is deleted
with the job `[source:
MediaBrowser.MediaEncoding/Transcoding/TranscodeManager.cs:145-275 @ v10.11.11]` — and, behind
the operator's produced-segment knobs (`EnableSegmentDeletion`, off as shipped;
`SegmentKeepSeconds`, 720) `[source:
MediaBrowser.Model/Configuration/EncodingOptions.cs:25-26 @ v10.11.11]`, **by position behind
the client**. Atrium ships the same knobs with the same defaults. A remux that fills the disk
takes the server down with it, and a transcode fills it faster — which is why the
session-scoped reclamation is not optional even though the other is.

**And that second window is a distance, not an age**, which this section asserted the other way
round until it was measured. `SegmentKeepSeconds` is *seconds of media the client has already
fetched*: what is removed is every produced segment whose index falls below
`(furthest-fetched position − window) ÷ segment length`, and nothing at all is removed until
the client's furthest-fetched position has passed the window. Measured on a 720-second window
with a client whose furthest segment ended 811 seconds in: segment 29 was gone and segment 33
was not, forty-five seconds after both were produced, with nothing requested in between and the
session still running `[probe: tools/probe_transcode_session.py, Jellyfin 10.11.11, 2026-08-29;
source: MediaBrowser.Controller/MediaEncoding/TranscodingSegmentCleaner.cs:100-113 @
v10.11.11]`. The difference is not academic: read as an age, a session paused for longer than
the window loses the segments it is about to resume into, and the client's next request is a
re-encode of material it already had. The **furthest-fetched position never moves backwards** —
a client filling a gap behind itself leaves it where it was — which is what keeps a rewind from
re-arming the deletion.

## 4. Data the feature owns

| State | Observable as | Lifetime |
|---|---|---|
| Probe results per file | `MediaSources`, `MediaStreams` in 005 and here | Until the file changes |
| Playback sessions | Whether a delivery request succeeds; `/Sessions` via 007 | Until stopped or reaped |
| Remux scratch output | Response latency only | Disposable |
| Transcode scratch output | Response latency, and whether a re-requested segment is the same bytes | Disposable, bounded, session-scoped |

## 5. Acceptance criteria

1. `PlaybackInfo` with **no** profile answers direct play, not "not playable" — and
   `PlaybackInfo` with an **empty** profile answers the opposite, every flag false, because a
   profile that lists nothing permits nothing (§3.3). The criterion read "an empty profile" until
   both halves were measured.
2. `PlaybackInfo` with a profile accepting the source's container and codecs answers direct play.
3. A profile rejecting the container but accepting the codecs answers remux, with a
   `TranscodingUrl`.
4. A profile rejecting the codecs, but accepting at least one codec and container this server can
   produce, answers **transcode**, with a `TranscodingUrl` — not an error.
5. A profile accepting no container or codec this server can produce answers `200` with every
   capability flag `false`, no `TranscodingUrl`, and **no `ErrorCode`** — never a `4xx`. The one
   `ErrorCode` that exists, `NoCompatibleStream`, is emitted exactly when the media source list
   is empty (§3.2).
6. `SupportsTranscoding` is `true` on a source exactly when this server can produce, for *this*
   profile, a stream the profile accepts — and `false` otherwise, including on the ones refused
   with every flag down (§3.3). The criterion read "exactly on the sources whose negotiated answer
   is a stream this server can produce" until the flag was measured: it is not derived from the
   answer. One accepting profile with a transcoding entry and the same profile without one both
   answer **direct play**, and only the first says `true`.
7. A source whose video the profile accepts and whose audio it does not is delivered with its
   **video stream copied**: same codec, same resolution, same frame count as the source.
8. Delivered output satisfies **every** condition of the profile it was negotiated against —
   asserted table-driven over the profile classes, on the delivered bytes, not on the decision.
9. Nothing is upscaled or up-sampled: a 720p source under a 1080p ceiling is delivered at 720p.
10. A request carrying a start position begins production at that position: time to first byte for a
    seek near the end of a long source is of the same order as one near the beginning.
11. Every delivery route that serves a body of media bytes whose size is known answers
    `Accept-Ranges: bytes` — the static routes, a sized remux, `/universal`'s direct answer, a
    WAV answer and every HLS segment. **The two playlist routes are the measured exception**:
    they carry a `Content-Length` and no range unit at all, on the reference and here (§3.5,
    §3.7). The criterion read "every delivery route whose body has a known size" until the
    playlists' header sets were read back, and implemented literally it would have sent a header
    the reference does not send on the one delivery family a client parses as text.
12. `Range: bytes=100-199` answers `206` with a correct `Content-Range` and exactly 100 bytes.
13. An unsatisfiable range answers `416` with `Content-Range: bytes */total`.
14. Direct-play responses carry a `Content-Length` equal to the file size.
15. **Remuxed responses carry a `Content-Length` and honour `Range`** (the §3.5 divergence).
16. Every HLS segment carries a `Content-Length`, whether it was remuxed or re-encoded.
17. A progressive re-encode whose final size is unknown answers chunked, and never a
    `Content-Length` that is not the true length.
18. `static=true` always serves the untouched original bytes — never an error, a remux or a
    re-encode. A container-suffixed static URL that does not match the source changes only the
    `Content-Type` label, byte-for-byte identical body (§3.5).
19. `/universal` with a constraint that requires re-encoding — a sample-rate, bit-depth or channel
    ceiling below the source — answers a stream, and the two ceilings that name an output
    property **are met**: the sample rate is the stated ceiling, not the reference's Opus-ladder
    step above it, and the channel count is the stated ceiling clamped to the source (§3.6,
    behaviours §3.7). A bit-depth ceiling is the third trigger and not a third target — it
    refuses the copy, and neither server states a sample format for it (§3.6). The criterion read
    "meets the constraint" of all three until the reference's encoder arguments were read.
20. **Every produced WAV answer carries a valid RIFF header, a `Content-Length` equal to its body
    and an honoured `Range`**: `stream.wav`, `stream` with `container=wav`, and `/universal` whose
    **transcoding** container is `wav` — with a `pcm_*` codec and without one
    ([behaviours §3.2](../../docs/compatibility/behaviours.md#32-pcmwav-output--one-bug-two-symptoms-two-classes)).
    The criterion read "`/universal` with `Container=wav`" until the shapes were measured: that
    parameter is the direct-play list and answers mp3 on both servers, and the reference's
    headerless body comes from the transcoding container (§3.6).
21. `/universal` accepts `enableRedirection` and **never answers `302` for a local source**: the
    direct-play answer is the proxied bytes with a `200` (§3.6). The redirect branch exists only
    for remote HTTP sources, which v1 does not have.
22. The same **remuxed** HLS source requested twice yields identical segment boundaries and
    identical segment bytes.
23. Within one session, a **re-encoded** segment requested twice yields identical bytes.
24. An HLS segment requested out of order is served correctly.
25. `DELETE /Videos/ActiveEncodings` terminates the work, verified by the absence of the process and
    the reclamation of its scratch space.
26. A client disconnecting mid-remux or mid-transcode causes the work to stop within the timeout.
27. With throttling enabled in configuration, a client that fetches the first segments and then
    stops does not cause the whole source to be produced: production pauses at the configured
    gap, and the next segment request releases it. With it disabled — the shipped default,
    matching the reference — production continues to the end (§3.4). The gap is measured
    against the same furthest-fetched position AC-29 counts back from.
28. Item-level `Container` is the demuxer list; the media source's `Container` is the single
    resolved container.
29. Scratch space is reclaimed with its session: after a stop, a kill-timeout or a shutdown,
    nothing of the session's output remains; with segment deletion enabled, the produced
    segments lying more than the configured window **behind the client's furthest-fetched
    position** are removed while the session still runs, and a session whose client has not
    fetched past that window loses nothing however long it is left (§3.8). The criterion read
    "segments older than the configured window" until the window was measured: it is a distance
    in the film, and the two rules disagree on exactly the case a paused client produces.
30. A `PlaySessionId` from `PlaybackInfo` is accepted by the delivery route and by
    `ActiveEncodings`.
31. Policy shapes the negotiation exactly as measured (§3.3): a user denied **every**
    playback-processing permission negotiates `SupportsTranscoding: false` and no
    `TranscodingUrl`; a user denied only one of them negotiates exactly as a permitted user; and
    no policy shape produces an `ErrorCode` or a `4xx`. At delivery, a denied user's re-encode
    step is refused rather than force-copied into an output that violates the profile —
    **per stream and on the video routes**, which is where the reference reads the permissions:
    a plan that copies the stream a denial names is served exactly as a permitted account's is,
    and an audio-only delivery consults no permission at all (§3.3).
32. The four `stream` routes **accept every token mechanism and require none**: a request carrying
    nothing at all, one carrying a token nothing issued, and one carrying `?api_key=` answer
    identically, byte for byte. `/Audio/{itemId}/universal` and the **two playlist routes** are the
    three delivery routes that refuse without a token, and all three refuse with the empty `401`
    (§3.5, §3.6, §3.7, behaviours §2.10). The criterion was written the other way round until it
    was measured, and named only `/universal` until the playlists were measured too.

## 6. Conformance

| Endpoint | Level | How it is proven |
|---|---|---|
| `POST /Items/{itemId}/PlaybackInfo` | **L3** | Golden per profile class — including the classes that force a transcode — plus differential. The negotiation is where clients diverge |
| `GET /Audio/{itemId}/stream` | **L3** | Golden headers, byte-identity against the source, plus differential |
| `GET /Audio/{itemId}/universal` | **L3** | Golden per constraint class, plus differential |
| `GET /Videos/{itemId}/stream` | **L3** | Golden headers and range matrix, plus differential |
| Container-suffixed forms | **L2** | Same assertions, path-derived container |
| HLS routes | **L2** | Playlist shape, segment determinism (AC-22, AC-23), out-of-order fetch |
| Transcoded output | **L2** | The delivered bytes are inspected and asserted against the profile they were negotiated for (AC-8, AC-9). **Not** byte-compared with the reference |
| Throttling and scratch reclamation | **L2** | Produced output observed against what was fetched (AC-27, AC-29) |
| `DELETE /Videos/ActiveEncodings` | **L2** | Process and scratch-space observation (AC-25) |

**Transcoded bytes are not compared with the reference, and never will be.** Two encoders given the
same instruction produce different bytes, and the difference is not a defect. What is asserted is
the property a client depends on: *the output satisfies the profile*. Levels above that would be
asserting that Atrium ships the reference's encoder, which is not a compatibility claim.

**The range matrix is table-driven** over: no range, prefix, suffix, mid-file, single byte, exactly
the whole file, one past the end, and a reversed range. These are where range implementations
actually break, and each is one line of test.

Media fixtures are **synthetic, generated at build time** — a few seconds of colour bars and a tone,
muxed into each container the tests need. No copyrighted media, and the repository stays small. The
transcode tests need two more of them: a source whose codecs no common profile accepts, to force
step 3 of the decision, and a source with an accepted video track beside a rejected audio track, for
AC-7. **They stay seconds long**, because every transcode test now spends real CPU, and a suite that
takes minutes is a suite that stops being run.

## 7. Open questions

None open. OQ-5 moved to the 010 differential, where it always belonged; the other eleven were
answered at the spec review on 2026-08-28, by the five probes the table had been citing
prospectively — every one of which now exists and runs.

### Resolved

| # | Question | Answer | Resolved by |
|---|---|---|---|
| OQ-1 | The `ErrorCode` vocabulary and which code fires for each failure | **Three enum members, one assignment site.** `NoCompatibleStream` when the source list is empty; `NotAllowed` and `RateLimitExceeded` are dead vocabulary. A profile refusal is flags, not a code — AC-5 rewritten | `tools/probe_playback_info.py` and source, 2026-08-28 |
| OQ-2 | `SupportsDirectPlay` per source or per request? | **Per request** — `EnableDirectPlay: false` flips the flag on a source the profile satisfies. And `SupportsDirectStream` mirrors `SupportsDirectPlay` always: the reference's direct-stream path is disabled outright | `tools/probe_playback_info.py`, 2026-08-28 |
| OQ-3 | The HLS segment duration and boundary rule | **Predicted up front, uniform except the last, `, nodesc`, per-segment `runtimeTicks` and `actualSegmentLengthTicks`.** 3.004 s re-encoded at 23.976 fps, 6.0 s stream-copied, same film — the complete VOD playlist (2 843 segments) arrives in 0.18 s, before any segment exists | `tools/probe_hls.py`, 2026-08-28 |
| OQ-4 | Does `enableRedirection` answer `302`? | **Never for a local file.** The redirect requires a remote HTTP source and `EnableRemoteMedia`; a library file is proxied `200` bytes — AC-21 rewritten | `tools/probe_universal_audio.py` and source, 2026-08-28 |
| OQ-5 | Which `/universal` parameters clients actually send | Moved to the 010 differential — a question about clients, not about the reference | — |
| OQ-6 | `DELETE /Videos/ActiveEncodings`: session id or all of the caller's? | **One session, named by `playSessionId` alone.** Both parameters are mandatory — omitting either is a validation `400` naming it — and the device is then bound and ignored: a call carrying an unknown device still stops the named session. The well-formed call is `204` and the session stops reporting a completion percentage, but keeps the rest of its `TranscodingInfo` | `tools/probe_transcode_session.py`, 2026-08-29 |
| OQ-7 | One variant or several in the master playlist? | **One for a standard-range negotiation, and one standard-range entrance per enabled encoder beside a high-dynamic-range stream copy** — h264 always, hevc and av1 only where the operator has permitted those encoders, every one of them at the copy's own `BANDWIDTH` so a client selects on colour range. Plus a trickplay `#EXT-X-IMAGE-STREAM-INF` when the reference has trickplay images; v1 has none. *(Answered "exactly one" on 2026-08-28 from a run against the library's first film, which was standard range — so the branch could not fire and its absence was recorded as the shape of the route. Re-measured on an HDR source: three variants, at a server with `AllowHevcEncoding` on.)* | `tools/probe_transcode_decision.py`, 2026-08-29 |
| OQ-8 | What goes into `TranscodingContainer`, `TranscodingSubProtocol` and the `TranscodingUrl` | **`ts` / `hls`, and a relative `/videos/{dashed-id}/master.m3u8?&…`** with PascalCase parameters: `DeviceId`, `MediaSourceId`, `VideoCodec`, `AudioCodec`, stream indexes and bitrates, `SegmentContainer`, `PlaySessionId`, `ApiKey` (the caller's token), `Tag` (the source `ETag`), source-codec condition triplets (`hevc-level`, `hevc-profile`, `hevc-videobitdepth`) and `TranscodeReasons` | `tools/probe_transcode_decision.py`, 2026-08-28 |
| OQ-9 | Copy the compatible stream, or re-encode both? | **Copy.** The accepted video codec survives byte-inspected in the segment (`IsVideoDirect: true`), only the rejected audio is re-encoded — §3.4's table is parity | `tools/probe_transcode_decision.py`, 2026-08-28 |
| OQ-10 | The throttle margin, and what happens when fetching stops | **An operator setting, off as shipped.** `EnableThrottling` defaults `false`; enabled, production pauses `max(ThrottleDelaySeconds, 60)` s (default 180) ahead of the last download — measured stalled at the gap. §3.4 and AC-27 rewritten around the configuration | `tools/probe_transcode_session.py` and source, 2026-08-28 |
| OQ-11 | Start at the requested position, or produce from zero? | **Restart at the position.** A segment at ~90% of a 2 h 22 film arrives in 0.9 s and the session's progress jumps to the seek point | `tools/probe_transcode_session.py`, 2026-08-28 |
| OQ-12 | What does `EnableTranscoding: false` in the body answer? | **It is ignored.** The `TranscodingUrl` arrives anyway — there is no error path to name, and inventing one would be a delta | `tools/probe_playback_info.py`, 2026-08-28 |

**And two things nobody asked** fell out of the same session: the reference's `/universal`
answers a sample-rate ceiling from the Opus ladder — above the ceiling when it falls between
steps — and answers a codec-less http transcode request with an empty `200` (§3.6, behaviours
§3.7 and §3.8).

## 8. References

- [docs/compatibility/api-surface-v1.md §8](../../docs/compatibility/api-surface-v1.md#8-playback-negotiation-and-delivery)
- [docs/compatibility/behaviours.md §1.6, §3.2, §3.3](../../docs/compatibility/behaviours.md)
- [docs/glossary.md](../../docs/glossary.md) — direct play, direct stream, transcoding, PlaySessionId
- `[spec: GetPostedPlaybackInfo, GetPlaybackInfo, PlaybackInfoDto, PlaybackInfoResponse, MediaSourceInfo, MediaStream, DeviceProfile, GetAudioStream, GetUniversalAudioStream, GetVideoStream, GetMasterHlsVideoPlaylist, GetVariantHlsVideoPlaylist, GetHlsVideoSegment, StopEncodingProcess]`
