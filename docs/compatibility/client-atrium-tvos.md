# One client's requirements, traced against v1

**Last verified: 2026-08-29**, against the client's conformance document as received from its
author on 2026-08-29, and this repository at `95a6b67` — 008 T1 through T12 merged. **§§3.2 and 3.3
rest on something else**: what that author said about the client in conversation on the same date,
which is evidence of a different kind and carries a mark of its own
([§1](#1-how-to-read-the-evidence-here)).

[api-surface-v1.md](api-surface-v1.md) is written from the server's side: *these are the endpoints
v1 serves, and here is who asked for them*. This document asks the same question from the other
side, for one real client — **what must Atrium do so that this client cannot tell the
difference?** — which is Principle I with the consumer named. Its sibling
[client-embeat-mobile.md](client-embeat-mobile.md) does the same for the music client, and the two
disagree about almost everything structural; §2 of that document is the summary of how.

The client is the one `surface.yaml` calls **video-client**: a tvOS application for movies, series
and music. [api-surface-v1.md §1](api-surface-v1.md#1-how-this-set-was-derived) describes the two
analysed clients by role rather than by name, because their internals are not this repository's to
publish; this one is named here because its author published a conformance document *for* this
repository, written in English and meant to be quoted here. The tag stays `video-client`, so
nothing machine-readable moves.

Much of the v1 surface was derived from this client, so the endpoint half of the answer is
uninteresting and was always going to be: **every operation it calls is already in the 55.** The
useful half is behavioural, and on 2026-08-28 it was four gaps in specifications. It is now
**eight findings against merged code**, which is a different kind of document: not *what should
008 say*, but *what does this server do to this client today*.

## 1. How to read the evidence here

Two provenance marks are used in addition to the ones in [../README.md](../README.md#conventions):

| Mark | Meaning |
|---|---|
| `[client-contract: 2026-08-29, §3]` | That section of the client's own conformance document, of that date |
| `[client-author: 2026-08-29]` | Something the client's author said about the client they wrote, in conversation with this repository on that date, and written in no edition of the contract |

**`client-contract` ranks with `prior-probe`, with one difference that matters.** A `prior-probe` was a
measurement *of the reference* made by this project and carried forward; these are claims made by a
third party about their own software — and, in several places, about Jellyfin. Claims of the first
kind are authoritative for what *the client* does, because its author is the one who can know.
Claims of the second kind are **leads for probes, never measured behaviours** (Principle II), and
this document marks each one as such.

**`client-author` ranks below `client-contract`, and not because the source is weaker.** It is the
same author, about the same software, and on the question it is used for — *why* the client is
built the way it is — that author is again the only one who can know. What it lacks is a document.
A `client-contract` row can be walked to a numbered section that exists, which whoever holds both
documents can read; a `client-author` row ends at the sentence carrying it. So the mark is used for
**intent and decisions** — what a path is for, what was considered and set aside — and never for
behaviour: anything about what the client *does* on the wire keeps `client-contract` or waits for a
probe. Nothing about **Jellyfin** may carry it at all, because a third-party claim about the
reference is only a lead when it is written down (Principle II) and less than one when it is
remembered. And it carries no `§N`, because there is no section to name — inventing one would make
it look checkable.

**Two editions, and the date in the mark is doing real work.** The contract has been received
twice: 2026-08-28, and a 2026-08-29 edition derived from the client at a later commit. The second
is **narrower, not a retraction**: it declares its scope as *video playback*, and discovery,
authentication and browsing appear in it only where playback depends on them. So the thirty
operations of §3, the seven hand-built URLs of §4, the exclusions of §7 and the UDP discovery of §1
are not restated there — and the rows below that rest on them keep `[client-contract: 2026-08-28,
§N]`, with its older date, rather than being silently re-dated to the newer document. Nothing was
withdrawn; a narrower document simply cannot renew what it does not mention.

What the newer edition adds is the part this repository could not have written for itself: the
client's playback paths, and **what each field of a `PlaybackInfo` response decides**. That is
where every new finding in §4 comes from.

**No line of the client's source is cited here, and that is deliberate.** The contract traces each
of its own rows to a `file:line` of the client; this document traces to the contract instead. A
path in a repository the reader of this one cannot open is neither verifiable by them nor ours to
publish ([AGENTS.md](../../AGENTS.md), *never cite a path outside this repository*) — the same
policy that keeps [api-surface-v1.md §1](api-surface-v1.md#1-how-this-set-was-derived) describing
its two clients by role. Whoever holds both documents can walk the last step; whoever holds only
this one loses nothing they could have checked anyway.

Everything asserted about *Atrium* below is checkable from this repository, and cites a
specification section, a document line or a source line.

## 2. The answer

| Contract section | What v1 does | Verdict |
|---|---|---|
| §0 `Authorization: MediaBrowser Client=…, DeviceId=…`, token appended | Either header spelling is read for the client identification ([`compat/auth.py:146-152`](../../src/atrium/compat/auth.py)) and for the token ([`:155-177`](../../src/atrium/compat/auth.py)) | ✅ |
| §1 The eleven-field `PlaybackInfo` body | Every one of them is bound ([`api/media_info.py:190-205`](../../src/atrium/api/media_info.py)); `EnableTranscoding: false` decides nothing, which is the reference's own behaviour and not a gap ([008 gate](../../specs/008-playback-negotiation-and-delivery/spec.md)) | ✅ |
| §1 `MediaSources[0]` exists and has `Id` | Emitted for every part of every item, inspection or no inspection ([`media/info.py:410-438`](../../src/atrium/media/info.py)) | ✅ |
| §1 `SupportsDirectPlay`/`SupportsDirectStream` reflect reality | Only where a stored inspection exists — see [§4.1](#41-a-source-with-no-stored-inspection-is-the-clients-documented-dead-end) | 🔴 [§4.1](#41-a-source-with-no-stored-inspection-is-the-clients-documented-dead-end) |
| §1 `TranscodingUrl` present whenever direct is refused | The reference's own condition, transcribed ([`api/media_info.py:381-388`](../../src/atrium/api/media_info.py)) — except in the case above, where nothing runs at all | ✅ / 🔴 |
| §1 Two round trips: the second sets both switches false and takes HLS | [008 §3.3](../../specs/008-playback-negotiation-and-delivery/spec.md): a step removed by the request is not silently substituted — the ladder falls through to transcode, with a `TranscodingUrl` | ✅ |
| §1 `Size` is the byte length of the file being served | Read from the stored part, so it survives a missing inspection ([`media/info.py:427`](../../src/atrium/media/info.py)) — one of the three obligations of [§3.2](#32-on-device-remux-is-a-placement-of-work-not-only-a-way-round-a-defect) | ✅ |
| §1 `MediaStreams[].IsTextSubtitleStream` | Emitted on every stream since 011 T2, beside `SupportsExternalStream`, read off the codec spelling the reference renames at inspection ([`media/info.py`](../../src/atrium/media/info.py), [`media/probe.py`](../../src/atrium/media/probe.py)) | ✅ |
| §1 `DeviceProfile.TranscodingProfiles[].EnableSubtitlesInManifest: true` | Bound since 011 T9 ([`api/media_info.py`](../../src/atrium/api/media_info.py)) and **read by nothing, which is parity**: the reference writes it into the delivery address and the route that address names does not accept it either (011 §3.4, measured) | ✅, and the flag is not what makes subtitles appear — see [§4.2](#42-v1-has-no-way-to-deliver-a-subtitle-and-this-client-has-one-way-to-receive-one) |
| §1 `DeviceProfile.TranscodingProfiles[].Protocol` selects HLS | Compared case-sensitively against `"hls"` ([`media/urls.py:202`](../../src/atrium/media/urls.py), [`:236`](../../src/atrium/media/urls.py)) where `/universal` normalises ([`api/universal_audio.py:267`](../../src/atrium/api/universal_audio.py)) | 🟠 [§4.6](#46-two-spellings-of-hls-and-only-one-of-them-selects-hls) |
| §2 `Range` must answer `206`, never `200` | [`compat/ranges.py:87-140`](../../src/atrium/compat/ranges.py): a well-formed `bytes=lo-hi` inside the file is `PARTIAL_CONTENT`, always — one of the three obligations of [§3.2](#32-on-device-remux-is-a-placement-of-work-not-only-a-way-round-a-defect) | ✅ |
| §2 `static=true` is the original container bytes | [behaviours §2.20](behaviours.md#220-statictrue-serves-the-original-bytes-the-urls-container-is-only-a-label), implemented at 008 T6 — one of the three obligations of [§3.2](#32-on-device-remux-is-a-placement-of-work-not-only-a-way-round-a-defect) | ✅ |
| §3 The master carries `VIDEO-RANGE`, `CODECS`, `FRAME-RATE` | [`media/hls.py:357-364`](../../src/atrium/media/hls.py) writes all three, on every variant | ✅ |
| §3 The master announces subtitle tracks | Since 011 T11: one `#EXT-X-MEDIA:TYPE=SUBTITLES` per text subtitle stream, and **every** variant line ends in the group — the standard-range entrance beside an HDR copy included ([`media/hls.py`](../../src/atrium/media/hls.py)) — when the delivery address names the manifest method | ✅ |
| §3 `…/Subtitles/{index}/Stream.vtt` when the manifest carries none | Three rows of [`surface.yaml`](surface.yaml) since 011's spec gate, served since 011 T7 and T8 ([`api/subtitles.py`](../../src/atrium/api/subtitles.py)) — whole, windowed, and with the start position in the path | ✅ |
| §3 `AudioStreamIndex`/`SubtitleStreamIndex` overridden on the stream URL | The audio half is a delivery parameter and is honoured ([`api/delivery.py`](../../src/atrium/api/delivery.py)); the subtitle half is one too since 011 T11, and the master playlist reads it beside `SubtitleMethod` | ✅ [§4.3](#43-the-clients-track-override-works-for-audio-and-is-dropped-for-subtitles) |
| §3 `GET /Sessions?deviceId=…` for copy verification | The route takes no `deviceId` ([`api/sessions.py:287-293`](../../src/atrium/api/sessions.py)) | 🟠 [§4.4](#44-get-sessions-takes-no-deviceid-and-the-client-sends-one) |
| §3 `DELETE /Videos/ActiveEncodings` on every re-negotiation | 008 T12: keyed on the play session, and it stops the encoder rather than answering `204` and lying | ✅ |
| §3 Workaround 1 — the fMP4 init segment starts a second transcode | Reproduced ([`media/sessions.py:499-506`](../../src/atrium/media/sessions.py)) | 🔴 [§4.5](#45-the-fmp4-init-segment-restarts-the-encoder-which-is-the-defect-the-client-pre-warms-to-dodge) |
| §3 Workaround 2 — the session key contains the `User-Agent` | **Not** reproduced: device, play session and path ([`media/sessions.py:151-164`](../../src/atrium/media/sessions.py)) | ✅, and the workaround is unnecessary here |
| §4 Direct play of a track at `/Audio/{id}/stream?static=true` | Surface row, `video-client` among its consumers since [§5.2](#52-getaudiostream-is-tagged-with-one-consumer-and-has-two) | ✅ |
| §5 The four reports, `PlayMethod` included | 007, implemented — and `PlayMethod` is stored, never used to infer a path | ✅ |
| §1 (2026-08-28) UDP discovery on 7359 | Out of v1, by an accepted decision ([001 §2](../../specs/001-server-identity-and-discovery/spec.md)) | 🔴 [§4.7](#47-udp-discovery-is-out-of-v1-and-the-client-needs-it) |
| §2 (2026-08-28) Authentication without `X-Emby-Authorization` | Accepted: `AuthenticateByName` reads either header name | ✅, and [§5.1](#51-x-emby-authorization-is-not-the-only-spelling-authenticatebyname-accepts) corrects a document |
| §2 (2026-08-28) `401`/`403` mean "not authorised", and nothing else | A malformed client header is `400`, a disabled account `403`, an absent token `401` ([behaviours §2.11](behaviours.md#211-a-disabled-account-is-refused-with-403-not-401)) | ✅ |
| §3 (2026-08-28) The thirty operations | All thirty are in the 55 — see [§3](#3-the-thirty-operations-and-the-seven-urls) | ✅ |
| §4 (2026-08-28) Chapter images for the scrubbing UI | Served, never generated — now recorded as [behaviours §5.8](behaviours.md#58-a-chapter-image-can-never-be-served-in-v1) | 🟠 [§4.8](#48-chapter-images-are-served-never-generated) |
| §7 (2026-08-28) What the client does *not* need | Agrees with v1's exclusions, item for item — see [§3.1](#31-the-exclusions-agree) | ✅ |

**None of the eight findings is a missing route**, and only one of them (§4.2) needs one. That was
true on 2026-08-28 and it survived the re-verification, which is the single most useful thing this
document says to whoever scopes the work: the surface is not the problem.

## 3. The thirty operations, and the seven URLs

**All thirty operations of the contract's §3 are in [`surface.yaml`](surface.yaml)**, and the
cross-reference already exists in machine-readable form: `consumers: [video-client]`. Established
on 2026-08-28 and not restated by the 2026-08-29 edition, whose scope excludes browsing
`[client-contract: 2026-08-28, §3]`.

| Contract §3 group | Operations | Owning feature | Status today |
|---|---|---|---|
| Identity and configuration | 7 | 001, 002, 004 | Implemented |
| Library | 14 | 005, 009 | Implemented — `GET /Playlists/{playlistId}/Items` was the last of the fourteen, at 009 T9 on 2026-09-01 |
| User data | 4 | 007 | Implemented |
| Playback | 5 | 007, 008 | Implemented — `PlaybackInfo` at 008 T5, `DELETE /Videos/ActiveEncodings` at T12 |

The contract's §4 lists seven URLs the client builds by hand rather than through its generated
client. **Five** are surface rows now that 011 serves the subtitle one; the other two are the
interesting ones:

| Hand-built URL | v1 |
|---|---|
| `/Items/{id}/Images/{kind}` | `GetItemImage` (006, implemented) |
| `/Items/{id}/Images/Chapter/{index}` | `GetItemImageByIndex` (006, implemented) — but see [§4.8](#48-chapter-images-are-served-never-generated) |
| `/Videos/{id}/stream?static=true` | `GetVideoStream` (008, implemented at T6) |
| `/Audio/{id}/stream?static=true` | `GetAudioStream` (008, implemented at T6) — its consumer list was one name short until this document, [§5.2](#52-getaudiostream-is-tagged-with-one-consumer-and-has-two) |
| ~~`/Users/{id}/Images/Primary`~~ | **Not in v1, and must not be.** The contract marks it a defect in the client — the route does not exist in 10.11 — and asks that it not be served. It is not, and neither is its replacement `GET /UserImage`, which no analysed client calls |
| `/Videos/{id}/{sourceId}/Subtitles/{index}/Stream.vtt` | `GetSubtitle` (011, implemented at T7) — a surface row since 011's spec gate, and served as written, including the lower-case `stream.vtt` a playlist entry spells it with. This row read *"Not in v1"* until 011 was implemented on 2026-08-31; see [§4.2](#42-v1-has-no-way-to-deliver-a-subtitle-and-this-client-has-one-way-to-receive-one) |
| `/Videos/{id}/Trickplay/{width}/tiles.m3u8` | Not in v1, and the client has parked the feature `[client-contract: 2026-08-28]`. Agreed on both sides |

**Query parameter names and casing are part of this contract**, because these URLs arrive as the
client wrote them: `quality` is always `90`; the trickplay URL uses `MediaSourceId`/`ApiKey` while
the stream URLs use `mediaSourceId`/`api_key` `[client-contract: 2026-08-28, §4]`. Atrium already
matches parameter names case-insensitively ([behaviours §1.15](behaviours.md#115-query-parameter-names-match-case-insensitively)),
so the mixed casing costs nothing.

### 3.1 The exclusions agree

Stated because a matching exclusion is worth as much as a matching feature, and because each of
these would otherwise look like something v1 owes somebody:

| Feature | The client | v1 |
|---|---|---|
| Quick Connect | Never calls it | Excluded ([api-surface-v1.md §10](api-surface-v1.md#10-deliberately-excluded-from-v1)) |
| Live TV / DVR | Out of its roadmap | Excluded |
| The `/emby` path prefix | Not used, measured against Emby | Not served |
| Trickplay | Asked for, but parked | Generation excluded |
| Emby's pre-flattening routes | Only used when the server fails the `jellyfin` test | Not served — [roadmap](../roadmap.md), "Emby dialect" |

**And one exclusion the newer edition adds, which is worth more than any of them.** Progressive
direct play of *video* is dead on the client side by design: whenever the server says a video
source can direct-play, the client immediately re-negotiates with both switches false and takes
the HLS answer, because the progressive path starves on that hardware at any bitrate
`[client-contract: 2026-08-29, §1]`. **For video there are two paths, not three.** Nothing in v1
changes, and no effort spent making progressive video smooth is spent on this client. Music still
uses it. **Which two they are, and why the surviving pair is not one real path and one fallback, is
[§3.2](#32-on-device-remux-is-a-placement-of-work-not-only-a-way-round-a-defect).**

### 3.2 On-Device Remux is a placement of work, not only a way round a defect

The two are **the client fetching the original file and packaging it on the device**, and **the
server producing HLS**. The client's own name for the first is **On-Device Remux**
`[client-author: 2026-08-29]`, and the name is the argument. Reading it as a workaround — *the
reference servers package badly, so the client stopped asking them to* — is true, and is the
smaller half. The other half is that **packaging is work, and the work has to be paid for
somewhere**: a host too small to package a film can be relieved of packaging by a device that is
not, and the author's own example of such a host is a Raspberry Pi `[client-author: 2026-08-29]`.
On that deployment the switch is not a measure taken while waiting for somebody to fix a server. It
is where the work belongs.

Three consequences for this repository:

**1. The setting is a permanent choice, not a temporary one.** Nothing Atrium can ship makes it
obsolete, because a server that packages correctly does not move the CPU cost back off the device —
it removes only the *other* reason for the switch. A server that treats the on-device path as
something its users will outgrow will under-invest in it for ever.

**2. Static delivery is therefore a first-class delivery mode for this client**, and the whole
server side of it is three obligations — all three already ✅ in [§2](#2-the-answer) above:
`static=true` answers with the original container's bytes whatever the URL's extension claims
([behaviours §2.20](behaviours.md#220-statictrue-serves-the-original-bytes-the-urls-container-is-only-a-label)),
every well-formed `Range` inside the file answers `206` and never `200`, and the `Size` the
negotiation advertised is the byte length of what the route actually serves. What changes here is
not those three verdicts but their weight. They are the cheap ✅s of a table whose interesting rows
are 🔴s, and on a small host they are the rows that decide whether anything plays at all. The third
of them had never been *tested* until [008 T14](../../specs/008-playback-negotiation-and-delivery/tasks.md),
which found that the advertised size and the served bytes had never met in any test — *"a client
reads that field as the length of what it is about to fetch and bounds every range request with
it"*.

**3. On a small enough host it is not the better path but the only one**, and
[roadmap.md](../roadmap.md#out-of-scope-and-why) is what sharpens that from a preference into
arithmetic. Hardware-accelerated transcoding is out of v1 on purpose — *"v1 encodes on the CPU —
slower, but portable and testable on any machine that can run the test suite"* — which is the right
call for the project, and is also the sentence that decides what a small server can produce.
Wherever the negotiation's answer is a re-encode rather than a stream copy, that re-encode is a CPU
one, and for an HDR film on a low-powered host it is not merely slow but infeasible
`[client-author: 2026-08-29]`. What is left is the path that costs the host nothing but bytes, so
**on that deployment the static route carries more of v1 than the produced one does** — the inverse
of where 008's effort went.

**That third one is an argument, not a measurement, and is marked so deliberately.** No probe in
this repository has run an encode on such a host, and the number that would settle "infeasible" — a
real-time factor for one CPU-encoded ladder on the hardware in question — is in none of them. It
rests on the roadmap's own exclusion plus the client author's experience of living with it, which is
exactly what `client-author` is for and exactly what it may not be promoted past (Principle II).

**One cost of this path that no server can remove.** The client's on-device demuxer reads **SubRip
and nothing else** `[client-author: 2026-08-29]`. A film whose subtitles are anything else — ASS, a
PGS or DVD bitmap, a sidecar in another format — has no subtitle the device can render on this
path, and the only way to see one is to turn the switch off and take the server's HLS. **That is
precisely the path a small host cannot pay for**, so the two costs compose: the deployment that
most needs the on-device path is the one an unsupported subtitle format takes it away from.
[011](../../specs/011-subtitle-delivery/spec.md) approaches the same user-visible problem from the
server's side — a track the device cannot read, converted before it is delivered — and for
image-based tracks the reference's own answer is burn-in, which v1 excludes
([roadmap](../roadmap.md#out-of-scope-and-why)). Where those two meet, v1 reaches that subtitle on
neither side of the switch.

**An observation of this repository's own, and a lead for
[010](../../specs/010-conformance-harness/spec.md) rather than a finding here.** That demuxer is an
executable statement of what a server must produce for Apple's native player: it takes bytes and
either plays them or does not, with no JSON to compare. 010's differential layer compares Atrium
against a reference carrying the very defects that made the demuxer worth writing
([§4.5](#45-the-fmp4-init-segment-restarts-the-encoder-which-is-the-defect-the-client-pre-warms-to-dodge)
is one of them), so a request both servers answer identically can still be two servers being wrong
together — the one failure a structural comparison is unable to see. An oracle that answers *"does
this play"* rather than *"do these bodies match"* would see it. **This is not a claim by the
client's author and not a plan**: no contract and no conversation contains it, the client is not
this repository's to run, its source is cited nowhere here, and nothing about how such an oracle
would be built has been designed. It is written down so that whoever scopes 010 has it.

### 3.3 The switch is global, and server identity would be the wrong key for it

**On-Device Remux is one setting for the whole application, and it stays one**
`[client-author: 2026-08-29]`. A per-server switch was considered and set aside, and the reason is
worth recording because "make it per-server" reads like an obvious improvement: a normal user has
exactly one server, so per-server and global are the same switch for everybody except the person
writing a second server to point the client at.

**Server identity would be the wrong key even where it is available.** What the switch is about is
whether *this master, for this file*, is serviceable — a property of a version and of a file, not
of a name. A reference server whose defects are fixed should have the switch off; an Atrium that
ships a broken master should have it on; keying on who served the bytes gets both of those
backwards.

**And this client could not use that key anyway, which is Principle I working rather than a gap in
it.** Atrium answers `ProductName` as exactly `Jellyfin Server`
([001 §3.1](../../specs/001-server-identity-and-discovery/spec.md)), because a client that has to
know which server it is talking to in order to behave correctly is the failure that principle
names. The client's own `jellyfin` test — the one that decides whether Emby's pre-flattening routes
are used ([§3.1](#31-the-exclusions-agree)) — therefore puts Atrium in the Jellyfin bucket, which is
correct and is the entire point. A switch keyed on it would have a constant for a key.

**What all this costs the server is one line: both paths must work, always.** The switch's position
never reaches the server — a `static=true` request and a master playlist request are all Atrium
sees, and neither says which setting produced it — so it is a choice Atrium cannot detect, cannot
steer, and cannot excuse one path from by being good at the other. And because the setting is
global rather than per-file, a user who turns it on for the one film that stutters has turned it on
for the whole library: the static route has to be right for **every** container a library admits,
which is why the sweep behind
[behaviours §2.20](behaviours.md#220-statictrue-serves-the-original-bytes-the-urls-container-is-only-a-label)
was worth its cost at 008 T6.

## 4. The eight findings

Four of these were the gaps of the 2026-08-28 trace, re-verified against merged code. Four are new,
and all four come from the newer edition's per-path detail.

### 4.1 A source with no stored inspection is the client's documented dead end

`PlaybackInfo` walks the item's sources beside whatever inspection is stored for each, and a source
whose inspection is `None` is **skipped entirely** — the ladder never runs and the annotation never
happens ([`api/media_info.py:479-483`](../../src/atrium/api/media_info.py)). What the client
receives is the intrinsic shape from [`media/info.py:410-438`](../../src/atrium/media/info.py):
`Id`, `Container` inferred from the path, `Size` from the stored part — and `SupportsDirectPlay:
true`, because that is the model's default
([`media/info.py:176`](../../src/atrium/media/info.py)) and nothing overwrote it. No
`TranscodingUrl`, because `_annotate` is where one would have been written.

Read that against the contract's own hard requirements
`[client-contract: 2026-08-29, §1]`: `TranscodingUrl` must be present whenever direct is refused,
or *"`isTranscoding` is false **and** there is no HLS → dead end"*. This server produces exactly
the shape the client names, from the opposite direction — direct play is *advertised* rather than
refused, so the client takes it, and then the second round trip sets both switches false, is
skipped again for the same reason, and comes back with the same answer and still no URL.

**What the user sees:** a title that never starts, with no on-screen reason. The rejection lines of
the contract's §6 are all source-side gates; none of them fires here, because the client got an
answer it believes.

**When it happens:** whenever a file is in the library and nothing has opened it — a scan that ran
before 008's inspection existed, a file added since the last scan, a probe that failed. The comment
at the `continue` says *"a rescan is what fixes it"*, which is true and is not something a client
can ask for.

This is the same root cause as
[client-embeat-mobile.md §5.1](client-embeat-mobile.md#51-a-source-with-no-stored-inspection-loses-the-music-clients-whole-negotiation),
with a different symptom: the music client keeps playing and silently loses hi-res detection and
its cast-to-a-capped-renderer path, because it never asked a question it could be dead-ended on.
**One skipped branch, two clients, two failures that look nothing alike.**

### 4.2 v1 has no way to deliver a subtitle, and this client has one way to receive one

**Closed on 2026-08-30 by [011](../../specs/011-subtitle-delivery/) T11**, and the four facts
below are kept as they were written because what closed each of them is the useful part. Today the
master playlist this client is handed carries one `#EXT-X-MEDIA:TYPE=SUBTITLES` per text subtitle
track and every variant line ends in the group; the address of each announcement is a per-track
playlist, and every window of that playlist is a WebVTT fetch. **The one thing that did not happen
is the thing this section predicted would**: the client's `DeviceProfile` flag is not the lever and
cannot be — the reference's own master playlist route does not accept it either (011 §3.4), so
what announces a track is `SubtitleMethod=Hls` in the delivery address. That is the *other* half of
this client's own behaviour, §4.3, which this section had sized as a line inside it.

This was the gap with consequences, and the 2026-08-28 trace called it correctly. What had changed
by 2026-08-30 was that it was four facts about merged code rather than three about
specifications — and the fourth of them was closed first, which is why it reads the other way
round:

- `GetSubtitle` is not among the 55, and [008 §2](../../specs/008-playback-negotiation-and-delivery/spec.md)
  excludes *"subtitle extraction, conversion and delivery as a separate route"*. `Stream.vtt` is
  not a row of [`surface.yaml`](surface.yaml), and L0 forbids serving a route that is not listed.
  **Closed by 011's own spec gate, which added three rows rather than two**, and served at 011 T7
  and T8: the whole-track fetch, its windowed form, and the ticks-in-path form a negotiation's own
  `DeliveryUrl` names;
- **the master playlist announces no subtitle track.** It wrote `#EXTM3U` and one
  `#EXT-X-STREAM-INF` per variant — one, or two where an HDR source is stream-copied and the
  second is its standard-range entrance ([008 §3.7](../../specs/008-playback-negotiation-and-delivery/spec.md#37-video-delivery),
  corrected 2026-08-30) — and nothing else. **Closed at 011 T11**
  ([`media/hls.py`](../../src/atrium/media/hls.py)'s `master_playlist`): the block is written
  before the first variant and **every** variant line ends in `,SUBTITLES="subs"`, the entrance
  included — an entrance with no subtitle group would be this client losing them for the very
  reason the entrance exists;
- **`EnableSubtitlesInManifest` is not a field of the profile model.**
  [`api/media_info.py`](../../src/atrium/api/media_info.py) declared eleven properties of a
  `TranscodingProfile` and that was not one of them, so `extra="ignore"`
  ([`compat/model.py:67`](../../src/atrium/compat/model.py)) dropped it on arrival. The client
  sends it `true` on every transcoding profile. **Bound at 011 T9 and read by nothing, which is
  parity**: the reference writes it into the delivery address it hands back and the route that
  address names cannot read it either (011 §3.4, measured), so a client asking for subtitles the
  way the reference's own model says to ask gets a manifest with none. This client's flag is
  therefore not what makes its subtitles appear;
- `IsTextSubtitleStream` **is emitted since 011 T2 (2026-08-30)**, on every stream and beside
  `SupportsExternalStream`, so the client's own input to *which* subtitle indexes it would put in
  the manifest query is now there. It was the first of the four pieces of work below and it is
  done; the three that reach this client are not.

The client's side of it `[client-contract: 2026-08-29, §1, §3]`: for a server it has identified as
Jellyfin it expects `EXT-X-MEDIA:TYPE=SUBTITLES` in the master, requested through the
`DeviceProfile`, and it rewrites the master before AVPlayer sees it but **does not add anything the
server left out**. It has a whole-file WebVTT fallback and that path is wired for the other
flavour — so a Jellyfin-identifying server that serves HLS without subtitle tracks in the manifest
shows no subtitles at all, and the client will not compensate.

**The blast radius is smaller than it first looks, and the shape of it decides the fix:**

| Playback path | Subtitles, before 011 | Since 011 T11 (2026-08-30) |
|---|---|---|
| On-device remux, embedded tracks | Fine **where the track is SubRip**, which is all the client's own demuxer reads `[client-author: 2026-08-29]`: the bytes are inside the file it is reading, and anything else among them is not rendered ([§3.2](#32-on-device-remux-is-a-placement-of-work-not-only-a-way-round-a-defect)) | Unchanged |
| Anything delivered over server HLS (remux or transcode) | None | Announced, per text track, **when the delivery address names the manifest method** — which is §4.3's line, not this section's |
| External sidecar files (`.srt` beside the media), any path | None, and none reachable | Discovered, numbered ahead of the container's own tracks, announced and served like an embedded one |

Which means the obvious fix was the wrong one: **adding `GetSubtitle` as a 56th endpoint would not
have helped this client**, because on the Jellyfin path it never asks. The only lever that reaches
it is the manifest, and the manifest cost the WebVTT extraction 008 excluded. Four pieces of work,
in dependency order: emit `IsTextSubtitleStream` (**011 T2**); bind `EnableSubtitlesInManifest`
(**011 T9** — bound, and read by nothing, which is what the reference does); extract and serve
WebVTT (**011 T5 to T8**); announce the tracks (**011 T11**). All four are done, and the second
turned out to buy nothing not because it is cheap but because the parameter decides nothing on
either server.

**One correction to this repository was owed here, and it was owed on 2026-08-28 too**: the
subtitle row of [behaviours §5](behaviours.md#5-accepted-gaps-in-v1) said subtitles are *"delivered
as files"*, and in v1 as implemented they are not delivered at all. **Made at 008 T14**, in the
change that marked the feature `Implemented` — a feature closing over that row would have been
exactly the claim the acceptance map exists to catch. The row now carries the three cases of the
table above, and names [011](../../specs/011-subtitle-delivery/) as the mechanism that closes it.

### 4.3 The client's track override works for audio, and is dropped for subtitles

**Closed on 2026-08-30 by [011](../../specs/011-subtitle-delivery/) T11.** The heading is kept
because other documents link to it and because what it names is still the finding; the subtitle
half is no longer dropped. Read the two paragraphs after the audio one for what changed.

The 2026-08-28 trace recorded this as one gap — *"the track indices in a `TranscodingUrl`'s query
are unspecified"* — and asked for one acceptance criterion covering both. Implemented, it split in
half.

`AudioStreamIndex` **is** a delivery parameter: bound at
[`api/delivery.py:166`](../../src/atrium/api/delivery.py), read at
[`:212`](../../src/atrium/api/delivery.py), and honoured at
[`:625`](../../src/atrium/api/delivery.py), where `_audio_stream` picks the stream whose index the
client named and falls back to the first only when there is no match. That is the client's
workaround working exactly as it needs to.

`SubtitleStreamIndex` **was not a delivery parameter at all**, and is one since 011 T11
(2026-08-30). It used to appear on the `PlaybackInfo` body
([`api/media_info.py`](../../src/atrium/api/media_info.py)) and on the playstate reports and
nowhere in [`api/delivery.py`](../../src/atrium/api/delivery.py), so a delivery request carrying it
was silently dropped — the reference's documented treatment of an unrecognised query value
([behaviours §1.12](behaviours.md#112-an-unrecognised-query-value-is-ignored-not-rejected)), and
therefore invisible.

It is now bound in [`api/delivery.py`](../../src/atrium/api/delivery.py)'s shared video parameter
set beside `SubtitleMethod`, and the master playlist reads both: the method decides whether tracks
are announced at all, and the index decides which announcement carries `DEFAULT=YES`. So the
client's rewrite of the address it was handed selects the track it names, which is this criterion
(011 AC-4) end to end — the manifest entry, the per-track playlist it addresses and the cues that
playlist's windows answer.

**This line said it would stop costing nothing the moment §4.2 closed, and that it would not
announce itself when it did.** Both were right, and the measurement sharpened the first: it is not
merely that the index would be ignored — the reference's master playlist route does not need it at
all, so an implementation reading §4.2's own summary would have required the pair and announced
**nothing** to a client that sent only the method. The lever is the method; the index is this
line. Whoever did §4.2 owned this line, and 011 T11 did both.

The contract's own claim about the reference — that it builds `TranscodingUrl` from the source's
*default* tracks and ignores the indexes posted in `PlaybackInfo`, which is why the client rewrites
them on the URL `[client-contract: 2026-08-29, §3]` — remains a third-party claim about Jellyfin
and is **a lead, not a measured behaviour**. `tools/probe_transcode_decision.py`, which [008
OQ-8](../../specs/008-playback-negotiation-and-delivery/spec.md) already names, is where it gets
settled. Atrium reads the body's `AudioStreamIndex` when the body names the source
([`api/media_info.py:312`](../../src/atrium/api/media_info.py)), so it may well not have the
behaviour the workaround exists for — which is safe either way, because the client overrides with
the same values.

### 4.4 `GET /Sessions` takes no `deviceId`, and the client sends one

The client verifies that a stream copy really is a copy by asking `GET /Sessions?deviceId=…` and
reading `TranscodingInfo.IsVideoDirect`, `.VideoCodec` and `.TranscodeReasons` off the session
whose `NowPlayingItem.Id` matches `[client-contract: 2026-08-29, §3]`.

[`api/sessions.py:287-293`](../../src/atrium/api/sessions.py) declares no `deviceId`, so the
parameter is dropped and the caller gets the unfiltered set it is allowed to see: its own sessions,
or every session if it is an administrator.

**This is a degradation, not a break, and the contract says why:** *"Returning nothing is safe: the
caller only acts on an explicit 'I am not copying'."* The client matches on `NowPlayingItem.Id`
anyway, so a wider list still finds the right row — on a single-user server it is the same row. It
gets worse exactly where the list gets long: an administrator on a busy server matching by item id
alone can find *another device's* session playing the same film, and read that session's
`TranscodingInfo` as its own.

Two things this is not. It is not a missing route — the row is in `surface.yaml`, tagged
`video-client`, implemented at 002. And it is not a delta to fix casually: whether the reference
filters on that parameter, and how it behaves when the caller is not an administrator, is
unmeasured here. A probe answers it in one request.

### 4.5 The fMP4 init segment restarts the encoder, which is the defect the client pre-warms to dodge

The contract names two Jellyfin behaviours under the heading **"do not reproduce them"**. This
server reproduces the first and avoids the second.

The first: requesting the `EXT-X-MAP` initialisation segment while the file is absent starts a
transcode from segment 0 unconditionally, and the positioned segment that follows kills it and
restarts with `-ss` — **two transcodes per resumed playback**. The client pre-warms the session to
dodge it `[client-contract: 2026-08-29, §3]`.

Atrium's restart rule is [`media/sessions.py:499-506`](../../src/atrium/media/sessions.py), whose
docstring is *"The reference's five branches, in its order"*, and whose first branch is
`index == INITIALISATION_INDEX` → restart, with no condition attached. The playlist puts that
segment in front of every fMP4 variant
([`media/hls.py:266-268`](../../src/atrium/media/hls.py)), so every resumed fMP4 playback pays for
it: the init request starts production from zero, and the positioned segment that follows exceeds
the gap allowance and restarts it.

**This is faithful reproduction, and it is the expensive kind.** [behaviours
§3.0](behaviours.md#30-how-the-decision-is-made) is the procedure for deciding it, and the input it
needs is not in this document: the contract's claim is third-party and cites Jellyfin's own source
at a line this repository has not read, so it is a **lead for a probe**. What is measurable from
here is the cost, and it is the client's own number — one wasted transcode start per resumed
playback, on the path the client takes for everything.

Worth noting for whoever picks it up: the client's pre-warm is a *workaround*, so a server that
did not have the defect would still see the pre-warm request. The fix is not "make the pre-warm
unnecessary"; it is "make the init segment not restart a running encoder that is already producing
the right thing".

### 4.6 Two spellings of `hls`, and only one of them selects HLS

The client's `DeviceProfile` declares every transcoding profile as HLS
`[client-contract: 2026-08-29, §1]`. Which spelling of the protocol string arrives on the wire is
not stated in the contract, and it decides the whole answer.

[`media/urls.py:202`](../../src/atrium/media/urls.py) and
[`:236`](../../src/atrium/media/urls.py) compare `decision.sub_protocol` to the constant `"hls"`
with `==`, and `sub_protocol` is the client's own
`TranscodingProfile.Protocol` carried through unchanged
([`media/decision.py:1001`](../../src/atrium/media/decision.py) ←
[`api/media_info.py:145`](../../src/atrium/api/media_info.py), a bare `str`). A profile spelled
`"Hls"` or `"HLS"` therefore fails both comparisons, and the `TranscodingUrl` comes back as a
*progressive* `/videos/{id}/stream.{container}` — a valid URL to a shape this client has designed
itself out of (§3.1), with no master and no variants.

`/universal` does not have this problem, and the contrast is the finding:
[`api/universal_audio.py:267`](../../src/atrium/api/universal_audio.py) normalises with
`.strip().lower()` before comparing, under a docstring that calls it *"measured, not lenience"*. So
two routes in this repository read the same value with two rules.

**Whether the reference is case-insensitive here is unmeasured**, and it is the question that
decides whether this is a gap or a robustness nicety: if it binds the property to an enum, .NET's
JSON enum binding matches names case-insensitively and a `"Hls"` profile works there and not here,
which is a delta in the one direction Principle I has no tolerance for. One probe posting a
`PlaybackInfo` body with `"Protocol": "Hls"` settles it.

### 4.7 UDP discovery is out of v1, and the client needs it

[001 §2](../../specs/001-server-identity-and-discovery/spec.md) puts it in as many words: *"UDP
autodiscovery on the local network — not in v1; clients take an address."* The client listens for a
broadcast datagram on port **7359**, answers the payload `who is JellyfinServer?` with a unicast
JSON carrying `Id`, `Name` and `Address`, and connects to that `Address` verbatim
`[client-contract: 2026-08-28, §1]`. It gives the network two seconds.

**What the user sees:** the "find my server" screen finds nothing, for ever. Typing the address
works. This is the one requirement in the contract that v1 excludes rather than merely leaves
unwritten, and it is excluded by a specification that is already **Implemented** — so closing it is
an amendment to 001 or a feature of its own.

Two things it is *not*. It is not an endpoint, so [`surface.yaml`](surface.yaml) and the L0 sweep
are untouched by the decision either way — Principle VI is about routes, and this is a datagram. And
it is not a place where a well-meant improvement is available: the contract is explicit that the
second probe, `who is EmbyServer?`, **is not ours to answer**, because a server that answers it is
claiming to be something it is not.

Unchanged since 2026-08-28, and not restated by the newer edition, whose scope is playback.

### 4.8 Chapter images are served, never generated

[006 §3.5](../../specs/006-images/spec.md) is explicit and reasoned: v1 serves chapter images that
exist on disk, does not extract them, and answers `404` per chapter for the ones that do not exist.
The client requests `/Items/{id}/Images/Chapter/{index}` for its scrubbing UI, independently of
Trickplay `[client-contract: 2026-08-28, §4]`.

**What the user sees:** a scrubbing bar with no thumbnails, on a library where Jellyfin would show
them, since Jellyfin extracts them on a background sweep.

**This one has moved since 2026-08-28**, and in the right direction: it is now recorded as
[behaviours §5.8](behaviours.md#58-a-chapter-image-can-never-be-served-in-v1), with a measurement
(1,311 of 1,354 reference entries carry a tag), a named closing mechanism and a tripwire test that
notices if any v1 code path ever starts writing chapter rows. The 2026-08-28 trace asked for this
gap to be recorded so that *"missing" is not later confused with "broken"*; the 2026-08-28 audit
recorded it. Nothing here is owed.

## 5. Two corrections this trace forced on our own documents

**Both were applied on 2026-08-28**, in the change that first carried this document. Neither was a
code change: both were places where a document in this repository said something that this client's
existence contradicts, while the server did the right thing already. Kept here because the argument
is what makes the consumer list trustworthy, not the edit.

### 5.1 `X-Emby-Authorization` is not the only spelling `AuthenticateByName` accepts

[api-surface-v1.md §3](api-surface-v1.md#3-authentication-users-and-sessions) said the route
*"requires the `X-Emby-Authorization` header"*, and repeated it below the table as **mandatory**.
The client sends that header **never, on any request, including sign-in** — the device-identifying
components travel in `Authorization: MediaBrowser Client="…", Device="…", DeviceId="…", Version="…"`
and nowhere else `[client-contract: 2026-08-29, §0]`, which the newer edition restates as the one
precondition on every request. A server built from that sentence refuses this client at the login
screen, and the refusal reads to a user like a wrong password.

Atrium was fine: [`compat/auth.py:146-152`](../../src/atrium/compat/auth.py) reads either header
name, and [behaviours §2.4](behaviours.md#24-there-are-five-authentication-mechanisms-and-one-of-them-wins)
already establishes that the reference reads both with the same grammar. What was wrong was the
prose, now corrected to *a client-identification header in either spelling, carrying a `DeviceId`*.

**One thing was left alone deliberately, and the music client has now made it interesting.** Two
error strings in [`compat/auth.py:137`](../../src/atrium/compat/auth.py) and
[`:141`](../../src/atrium/compat/auth.py) name only the Emby spelling — a `400` whose message points
at a header this client was never going to send. It is still there. It is *correct* for the music
client, which sends exactly that header and only on that route
([client-embeat-mobile.md §4](client-embeat-mobile.md#4-the-answer)), and misleading for this one.
Which makes it a wording change for whoever next opens 002, not a documentation edit, and now with
two clients' evidence rather than one's.

**And there is a measurement hiding in this.** Whether the *reference* accepts an
`Authorization`-only sign-in was never probed: `tools/probe_auth_mechanisms.py` set
`X-Emby-Authorization` and only that header on this route, every time it called it, so the question
was never put. **It was turned into a probe on 2026-08-28**: the probe now signs in with the
components in `Authorization` and the reference answers `200`
`[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]`, so the corrected
paragraph rests on a measurement of this repository's own rather than on this client's word. It is
the one row of this document that has stopped being third-party evidence.

### 5.2 `GetAudioStream` is tagged with one consumer and has two

[`surface.yaml`](surface.yaml) recorded `consumers: [music-client]` for
`GET /Audio/{itemId}/stream`. The tvOS client builds that URL by hand for music playback
`[client-contract: 2026-08-28, §4]`, so the row now carries `video-client` as well, in both the
YAML and [§8 of the prose table](api-surface-v1.md#8-playback-negotiation-and-delivery).

The contract's own *rationale* — that `/Videos/…` answers `404` for a track — did not survive
measurement: the video route serves the track's bytes whole, under `Content-Type:
video/quicktime` `[probe: tools/probe_video_stream_for_a_track.py, Jellyfin 10.11.11,
2026-08-28]`. The client's choice stands anyway, on the correct content type rather than on a
refusal — and the consumer fact this section exists for is unchanged. The 2026-08-29 edition
repeats the `404` rationale `[client-contract: 2026-08-29, §4]`, so the correction has not reached
the client's author; it is theirs to make or not, and it changes nothing on either side.

Counting the same way in the other direction: 33 rows of `surface.yaml` carried `video-client`
before that change and the client touches **34**. The count still agrees on 2026-08-29.

## 6. Where these findings go

**They are not 008 defects, and they are not amendments to it.** 008 closes on its accepted scope;
every finding above was measured against code that does what its spec, plan and tasks say it does.
Six of the eight are gaps between *that scope* and *this client*, which is a different question and
belongs to a different feature — the next one after 010. §4.7 belongs to 001 or to a feature of its
own, and §4.8 is closed.

Grouped as a scope would take them:

| Finding | Shape of the work |
|---|---|
| [§4.1](#41-a-source-with-no-stored-inspection-is-the-clients-documented-dead-end) | Decide what an un-inspected source advertises. One branch, one acceptance criterion, and a behaviours entry it does not yet have |
| [§4.2](#42-v1-has-no-way-to-deliver-a-subtitle-and-this-client-has-one-way-to-receive-one) + [§4.3](#43-the-clients-track-override-works-for-audio-and-is-dropped-for-subtitles) | Subtitle delivery, end to end. The largest of them, and §4.3 is a line inside it rather than work of its own |
| [§4.4](#44-get-sessions-takes-no-deviceid-and-the-client-sends-one), [§4.6](#46-two-spellings-of-hls-and-only-one-of-them-selects-hls) | One probe each, then one parameter each. Both are cheap and neither is safe to guess |
| [§4.5](#45-the-fmp4-init-segment-restarts-the-encoder-which-is-the-defect-the-client-pre-warms-to-dodge) | A [behaviours §3.0](behaviours.md#30-how-the-decision-is-made) decision, taken on a probe, before any code |
| [§4.7](#47-udp-discovery-is-out-of-v1-and-the-client-needs-it) | An amendment to 001, or its own small feature. Not a route |

The two behaviours entries this document owed — §4.2's correction to the
[§5](behaviours.md#5-accepted-gaps-in-v1) subtitle row, and §4.1's missing entry — were **written at
008 T14**: both describe what 008 ships, so both belonged to the change that closed it, and neither
waits on the work above.

## 7. What this document does not do

**It does not grow the surface.** No requirement here promotes an endpoint into v1: the thirty are
already in, the three unserved hand-built URLs are one client defect, one agreed exclusion and one
open decision, and the open one (§4.2) is not answered by adding a route.

**It does not become a second endpoint table.** [`surface.yaml`](surface.yaml) is the surface, and
`consumers: [video-client]` is where this client is already named; §3 above deliberately rolls up
rather than restating rows that would then drift.

**It is a floor, not a ceiling.** The contract says so of itself: absence from it means *not
measured*, never *not needed*. It describes the client at one commit on 2026-08-29, from its own
source — not from Jellyfin's documentation and not from a differential run. When the client
changes, this document is stale and nothing in CI will notice. It went stale once already, in a
day. The two `client-author` sections go stale differently and no less quietly: a decision does not
drift, it is reversed, and a reversed one leaves no trace on the wire either.

The mechanism that turns any of this into something measured is the same one the constitution
already names: the differential harness of
[010](../../specs/010-conformance-harness/spec.md), run request by request against both servers.
What this document contributes to that is a much smaller suite than 322 paths would suggest — **the
thirty-four rows above, and the eight behaviours of §4** — and the observation that the eight are
exactly the places where a server can pass every JSON comparison and still leave the user staring at
a video with no subtitles, a title that never starts, or a list of servers that stays empty.
