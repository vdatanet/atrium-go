# A second client's requirements, traced against v1

**Last verified: 2026-08-29**, against the client's conformance document as received from its
author on 2026-08-29, and this repository at `95a6b67` — 008 T1 through T12 merged.

This is the companion to [client-atrium-tvos.md](client-atrium-tvos.md), asking the same question
of the second real client: **what must Atrium do so that this client cannot tell the difference?**
The client is the one [`surface.yaml`](surface.yaml) calls **music-client** — a Kotlin
Multiplatform music application shipping on iOS and Android, sharing one server driver between
them. The tag stays `music-client`, so nothing machine-readable moves.

It is named here for the same reason its sibling is: its author published a conformance document
*for* this repository, written in English and meant to be quoted here.
[api-surface-v1.md §1](api-surface-v1.md#1-how-this-set-was-derived) still describes both clients
by role rather than by name, because their internals are not this repository's to publish.

**The interesting half is not the endpoints.** Twenty-seven rows of the surface already carry this
client's tag and its operations were part of how the 55 were chosen. One operation is outside them
(§3), and everything else worth writing down is behavioural — which for this client means something
different than it does for the other one, for the reason in §2.

## 1. How to read the evidence here

The provenance mark is the `client-contract` one that
[client-atrium-tvos.md §1](client-atrium-tvos.md#1-how-to-read-the-evidence-here) introduced, and it
means the same thing. That document now defines a second mark, for what its client's author has
said in conversation rather than published; this document has no rows from that source and does not
use it:

| Mark | Meaning |
|---|---|
| `[client-contract: 2026-08-29, §4]` | That section of the client's own conformance document, of that date |

**It ranks with `prior-probe`.** Claims about *the client's own software* are authoritative,
because its author is the one who can know. Claims about *Jellyfin* — and this contract makes many
more of them than the tvOS one does, several with dates and tool names attached — are **leads for
probes, never measured behaviours** (Principle II). Every one of them is marked as such below. Two
of them agree with measurements this repository already made independently, which is worth saying
where it happens and is not a licence to promote the rest.

**No line of the client's source is cited here, and that is deliberate.** The contract traces every
one of its own rows to a `file:line` of the client; this document traces to the contract instead. A
path in a repository the reader of this one cannot open is neither verifiable by them nor ours to
publish ([AGENTS.md](../../AGENTS.md), *never cite a path outside this repository*).

Everything asserted about *Atrium* below is checkable from this repository, and cites a
specification section, a document line or a source line.

## 2. There is no negotiation, and that changes what the server owes

Every assumption this repository has made about playback came from a client that calls
`POST /Items/{itemId}/PlaybackInfo`. 008's spec, its ladder, its `TranscodingUrl`, its
`SupportsDirectPlay` — all of it is a conversation, and the whole design rests on there being a
moment when the server gets to tell a client what it may have.

**This client never calls `PlaybackInfo` at all.** Not once, on any path. Its URL builder is
synchronous and pure: it takes the track already in the queue plus the user's quality preference
and returns a URL, with no suspend, no I/O and no round trip `[client-contract: 2026-08-29, §1]`.

Four consequences, and they are the reason this document exists:

1. **The item listing *is* the negotiation.** Everything a `PlaybackInfo` response would have
   carried has to arrive earlier, inside the library item, and it only arrives if the server
   honours `Fields=MediaSources`. Every endpoint in this client that can produce a playable track
   asks for it — six of them.
2. **There is no second chance.** A field missing from a list row is missing for the whole
   playback. The client does not discover it and re-ask; it degrades, silently, in a way named in
   §5.1.
3. **`SupportsDirectPlay`, `SupportsDirectStream` and `TranscodingUrl` are not in the picture.**
   The client decides its path from the *user's quality setting*, not from anything the server
   says. A server cannot steer this client at all — it can only answer the URL the client already
   decided to build.
4. **Its `PlaySessionId` is deterministic and client-minted** — one value per device and track,
   the same one it reports to `/Sessions/Playing`. Where the other client takes the server's
   session id out of a negotiation, this one hands the server an id and expects it to be used.

Read against [008 §3](../../specs/008-playback-negotiation-and-delivery/spec.md), which is a
specification of a negotiation, the effect is that **the parts of 008 this client exercises are the
delivery routes and `Fields=MediaSources`, and nothing in between.** Two of the five findings in §5
are places where that showed.

## 3. The twenty-seven operations, and the one that is not in the 55

**Twenty-seven rows of [`surface.yaml`](surface.yaml) carry `consumers: [music-client]`**, and the
cross-reference is already machine-readable. Rolled up rather than restated, so it cannot drift
from the YAML:

| Contract group | Operations | Owning feature | Status today |
|---|---|---|---|
| Identity and session (§2) | 4 | 001, 002 | Implemented |
| Library and browsing (§10) | 10 | 005 | Implemented |
| Playlists (§10) | 6 | 009 | Implemented on 2026-09-01 — see below for the one that refuses this client's own users |
| User data and reporting (§8) | 7 | 007 | Implemented |
| Artwork (§9) | 1 | 006 | Implemented |
| Delivery (§3, §4, §6, §7) | 3 | 008 | Implemented — the static pair at T6, `/universal` at T8 |

**One operation the client calls was not in the 55, and now it is a row.** `POST /Items/{itemId}` —
the reference's `UpdateItem` — is how this client renames a playlist `[client-contract: 2026-08-29,
§10]`. Every other playlist-editing operation it uses was already there: `POST /Playlists`,
`POST`/`DELETE /Playlists/{id}/Items`, `POST /Playlists/{id}/Items/{itemId}/Move/{newIndex}` and
`DELETE /Items/{itemId}`, all owned by 009.

**Decided at 009's spec review on 2026-08-31, and it entered the surface** — this document was the
named consumer [Principle VI](../constitution.md) asks for. It is routed for playlists, and the
gate measured the thing that decides what the row is worth: **the reference declares that
controller administrator-only, so this client's rename answers `403` for every user who is not
one — the playlist's own owner included**
`[probe: tools/probe_playlist_rename.py, Jellyfin 10.11.11, 2026-08-31]`. The route that renames
for an ordinary owner is `POST /Playlists/{playlistId}`, which this client does not call and which
Principle VI therefore keeps out.

So the rename works here exactly as far as it works against a stock reference server, and no
further. That is parity, and it is a gap in the client's own feature rather than in this one —
recorded in [behaviours §5](behaviours.md) with the mechanism that would close it.

**Shipped on 2026-09-01, and the row above is what shipped**: the rename is administrator-only
here because it is administrator-only there. 009 T13 measured two more things about it that this
client's own round trip walks into — the reference refuses a body that omits `Genres`, `Tags` or
`ProviderIds`, so `{"Name": …}` alone is not a request a stock server serves; and a body with no
`Name` at all answers `204` there and **erases** the playlist's name, which this server refuses
([behaviours §3.21](behaviours.md)). Fetch-change-post, which is what this client does, is
therefore the shape that works on both.

**The rest of the playlist group asks for something the reference does not have.** Every row of
`GET /Playlists/{playlistId}/Items` must carry a `PlaylistItemId` distinct from the track id, or
the client cannot address duplicates for removal and reordering `[client-contract: 2026-08-29,
§10]`. Half of that is satisfiable: the property is on every row. The other half is not —
**`PlaylistItemId` is the item's own `Id`**, measured on the wire
`[probe: tools/probe_playlist_move.py, Jellyfin 10.11.11, 2026-08-31]` — and the duplicates it
exists to address cannot occur, because the reference de-duplicates
([behaviours §2.7](behaviours.md), [§2.26](behaviours.md)). 009 §3 asserted the distinctness too,
in four places including an acceptance criterion, and its spec review corrected all four. Nothing
is owed to this client that a reference server delivers.

### 3.1 The one operation it deliberately avoids

`GET /Search/Hints`, which the client will not call because the two reference servers answer it in
divergent shapes; it searches through `GET /Items?SearchTerm=` instead `[client-contract:
2026-08-29, §10]`. `Search/Hints` is in `surface.yaml` with `consumers: []`, so nothing moves — but
it is now a *measured* empty list rather than an unexamined one. The divergence the client names is
between the two reference servers, which is not what
[behaviours §1.8](behaviours.md#18-get-itemslatest-returns-a-bare-array) measured; that entry found
`/Search/Hints` answering a shape of its own *within* Jellyfin, a fourth envelope beside three
others. Same family of problem, independently arrived at, and between them a good argument for the
route staying at zero consumers.

## 4. The answer

| Contract section | What v1 does | Verdict |
|---|---|---|
| §0 `X-Emby-Token` on every authenticated request | The second of the three shapes `extract_token` resolves, in the reference's own order ([`compat/auth.py:155-177`](../../src/atrium/compat/auth.py)) | ✅ |
| §0 `X-Emby-Authorization` on `AuthenticateByName` only, never again | That route requires a client-identification header carrying a `DeviceId` and no other route does ([`compat/auth.py:127-143`](../../src/atrium/compat/auth.py), [behaviours §2.13](behaviours.md#213-deviceid-is-mandatory-on-one-route-not-on-the-header)) | ✅ |
| §0 Stream, image and renderer URLs authenticate by `api_key` with no headers at all | The third shape, and the delivery and image routes accept a token and require none ([behaviours §2.10](behaviours.md#210-the-image-and-delivery-routes-accept-a-token-and-require-none)) | ✅ |
| §0 PascalCase in both directions, unknown keys ignored | [`compat/model.py:63-68`](../../src/atrium/compat/model.py): alias generator, `populate_by_name`, `serialize_by_alias`, `extra="ignore"` | ✅ |
| §0 Query parameter names cased inconsistently — six routes, three spellings of the same idea | Rewritten to each route's declared spelling before the framework binds them, key only, value untouched ([`compat/query_params.py`](../../src/atrium/compat/query_params.py), [behaviours §1.15](behaviours.md#115-query-parameter-names-match-case-insensitively)) | ✅ |
| §0 Any status outside 2xx/401/403 is retried three times | Nothing to implement, but see [§5.6](#56-every-error-this-server-returns-costs-three-requests-not-one) | 🟠 |
| §1 `Fields=MediaSources` on six listing routes | Served, and built from the stored inspections ([`api/item_dto.py:502-507`](../../src/atrium/api/item_dto.py)) | ✅ |
| §1 `MediaSources[0].Id`, `.Container`, `.Bitrate`, `RunTimeTicks` | All four emitted ([`media/info.py:410-438`](../../src/atrium/media/info.py)) — three of them only where an inspection exists | ✅ / 🔴 [§5.1](#51-a-source-with-no-stored-inspection-loses-the-music-clients-whole-negotiation) |
| §1 `MediaStreams[Audio].Codec`/`SampleRate`/`BitDepth`/`Channels` | All four emitted ([`media/info.py:371-375`](../../src/atrium/media/info.py)) — and this is the load-bearing one, since `SampleRate` decides the cast path | ✅ / 🔴 [§5.1](#51-a-source-with-no-stored-inspection-loses-the-music-clients-whole-negotiation) |
| §1 `Name` is required — a track without one fails the whole page | 005 emits it on every item; nothing in v1 can produce a nameless one | ✅ |
| §2 `ProductName` contains `jellyfin`, or the client uses its **Emby** driver | `REFERENCE_PRODUCT_NAME = "Jellyfin Server"` ([`src/atrium/__init__.py:26`](../../src/atrium/__init__.py)), emitted at [`api/system.py:112`](../../src/atrium/api/system.py), argued in [behaviours §4.1](behaviours.md#41-atrium-identifies-as-jellyfin-on-the-fields-clients-parse) | ✅ |
| §2 `/System/Info/Public`'s `Id` equals `AuthenticateByName`'s `ServerId` | One value: `state.server_id` at [`api/system.py:115`](../../src/atrium/api/system.py) and at [`api/users.py:154`](../../src/atrium/api/users.py) | ✅ |
| §2 `AuthenticateByName` body is `{"Username", "Pw"}`, answer carries `User.Id`/`AccessToken`/`ServerId` | [`api/users.py:147-155`](../../src/atrium/api/users.py) | ✅ |
| §3 `/Audio/{id}/stream?static=true` is the original container bytes | [behaviours §2.20](behaviours.md#220-statictrue-serves-the-original-bytes-the-urls-container-is-only-a-label), implemented at 008 T6 — extension and all, which is what §7 needs | ✅ |
| §3 `Range` honoured, and `Content-Length` load-bearing on this path | [`compat/ranges.py:87-140`](../../src/atrium/compat/ranges.py) and the four measured headers of [`api/delivery.py:405-419`](../../src/atrium/api/delivery.py) | ✅ |
| §3 Two concurrent streams per device, one idle for minutes | Nothing in the server closes an idle response, but see [§5.7](#57-a-suspended-preload-is-an-idle-connection-a-deployment-can-cut) | 🟠 |
| §4 Progressive MP3 on `/universal`, first byte within 20 s | Served at 008 T8, streamed as produced ([`api/delivery.py:812-846`](../../src/atrium/api/delivery.py)) | ✅ |
| §4 A complete Xing/LAME header, or the gapless trim is guessed | **No Xing frame at all** — see [§5.3](#53-a-piped-mp3-carries-no-xing-frame-which-is-not-the-blank-one-the-client-measured) | 🔴 [§5.3](#53-a-piped-mp3-carries-no-xing-frame-which-is-not-the-blank-one-the-client-measured) |
| §4 `Range` honoured on `/universal` for reconnects | Chunked answers set `Accept-Ranges: none` and read no range at all ([`api/delivery.py:845`](../../src/atrium/api/delivery.py), [behaviours §3.3](behaviours.md#33-progressive-transcoding-responses-carry-no-content-length-or-accept-ranges--class-c)) — parity with the reference | ✅ parity |
| §4 `PlaySessionId` accepted on `/universal`, transcode keyed on it | Not declared ([`api/universal_audio.py:271-295`](../../src/atrium/api/universal_audio.py)) — see [§5.4](#54-every-universal-request-re-encodes-for-a-different-reason-than-the-reference-does) and [§6.2](#62-keying-a-transcode-on-a-client-supplied-playsessionid) | 🟠 |
| §4 `RunTimeTicks` must be right, or an unsized stream stalls at the end | 005 emits it; §5.1 is where it goes missing | ✅ / 🔴 |
| §5 HLS on `/universal` (`TranscodingProtocol=hls`) | Served, and normalised case-insensitively ([`api/universal_audio.py:267`](../../src/atrium/api/universal_audio.py)). Dormant on the client side since 2026-08-03 | ✅ |
| §6 The two measured download defects — an unreadable first MP4, and `m4a` withholding every byte | Neither is reproducible here: `m4a` muxes to `ipod`, which is in `NEEDS_FRAGMENTING`, so a piped answer carries `frag_keyframe+empty_moov+default_base_moof` ([`media/ffmpeg.py:210-213`](../../src/atrium/media/ffmpeg.py)) and is readable and flowing from its first bytes | ✅ |
| §7 `LocalAddress` is plain `http://` | True at defaults ([`net/address.py:87-93`](../../src/atrium/net/address.py), [behaviours §4.2](behaviours.md#42-localaddress-does-not-get-an-https-override)) — and see [§5.5](#55-localaddress-is-plain-http-at-defaults-and-an-operator-can-take-that-away) | ✅ / 🟠 |
| §7 The capped renderer stream: `AudioSampleRate`, `MediaSourceId` on `/Audio/{id}/stream.{ext}` | Both bound ([`api/delivery.py:229-243`](../../src/atrium/api/delivery.py)); the `wav` form is sized and range-capable (008 T9), the `flac` form the Jellyfin driver actually asks for is not — [§5.2](#52-the-capped-renderer-stream-is-sized-for-wav-and-the-client-asks-for-flac) | 🔴 [§5.2](#52-the-capped-renderer-stream-is-sized-for-wav-and-the-client-asks-for-flac) |
| §7 An honest `Content-Length` on the capped stream deletes client code | A Principle I question, not a failure — [§6.1](#61-an-honest-content-length-on-a-capped-transcode) | ❓ |
| §8 `/Sessions/Playing`, `/Progress`, `/Stopped` with `ItemId`, `PlaySessionId`, `PositionTicks` | 007, implemented; ticks are 100 ns everywhere | ✅ |
| §8 No `GET /Sessions`, no `DELETE` to stop an encoding — a server that keeps ffmpeg alive accumulates jobs | It does not: a client that disconnects makes the framework close the body generator, and the `finally` stops the encoder ([`api/delivery.py:849-863`](../../src/atrium/api/delivery.py), 008 AC-26) | ✅ |
| §9 The album-cover pointer: `AlbumId` + `AlbumPrimaryImageTag` on a track | Emitted for `Audio` items ([`api/item_dto.py:174-175`](../../src/atrium/api/item_dto.py), [`:584-585`](../../src/atrium/api/item_dto.py)) — the client's shape 3, which it calls Jellyfin's form | ✅ |
| §9 `maxWidth`/`maxHeight`/`fillWidth`/`fillHeight`, token in the query | 006, implemented, and the fill pair crops rather than boxes | ✅ |
| §10 `TotalRecordCount: 0` on by-name endpoints without a `Limit` | [behaviours §3.1](behaviours.md#31-totalrecordcount-is-0-on-by-name-endpoints-without-limit--class-b) **diverges and returns the true count**; the client maps `Items` and never reads the field on those routes | ✅ |
| §10 The album detail is the play queue: `SortBy=ParentIndexNumber,IndexNumber,SortName` | Two of the three keys are dropped — [§5.8](#58-the-album-play-queue-is-correctly-ordered-by-accident) | 🟠 [§5.8](#58-the-album-play-queue-is-correctly-ordered-by-accident) |
| §10 `SortBy=Year` is sent as `ProductionYear,PremiereDate,SortName` | `ProductionYear` is not in the vocabulary either — same finding | 🟠 [§5.8](#58-the-album-play-queue-is-correctly-ordered-by-accident) |

**The identity block at the top of that table is the one to read first.** Four rows decide whether
this client speaks to Atrium as a Jellyfin server or as an Emby one, and the discriminator is a
single string: `ProductName` **not** containing `jellyfin` routes the client onto its Emby driver,
which uses different paths for the single item, the favourite, the played flag, the progressive
transcode and the image box `[client-contract: 2026-08-29, §2]`. Those Emby routes are not in the
55 and must not be, so a server that got `ProductName` wrong would not degrade — it would fail
every one of those five operations with a `404`. [behaviours §4.1](behaviours.md#41-atrium-identifies-as-jellyfin-on-the-fields-clients-parse)
already argues the identity exception on general grounds; this is the concrete consumer, and it is
a harder dependency than the argument there assumes.

## 5. The findings

### 5.1 A source with no stored inspection loses the music client's whole negotiation

`PlaybackInfo` skips a source whose inspection is missing, and so does every list row: a
`MediaSourceInfo` built with no inspection behind it keeps `Id`, `Container` inferred from the
path, `Size` and `Name`, and carries `RunTimeTicks: null`, `Bitrate: null` and **`MediaStreams:
[]`** ([`media/info.py:410-438`](../../src/atrium/media/info.py)). For `PlaybackInfo` the whole
annotation is skipped ([`api/media_info.py:479-483`](../../src/atrium/api/media_info.py)).

For the tvOS client that is a dead end and the playback never starts
([client-atrium-tvos.md §4.1](client-atrium-tvos.md#41-a-source-with-no-stored-inspection-is-the-clients-documented-dead-end)).
For this client the track **plays fine**, because direct play needs nothing but the item id — and
four things quietly stop working:

| Lost | Because |
|---|---|
| The hi-res badge | It is read off `MediaStreams[Audio].BitDepth`/`SampleRate` |
| Casting to a rate-capped renderer | The proxy route fires only when `SampleRate > ceiling`, and a null sample rate is not greater than anything. The track is sent to the Sonos at its own rate, which is the case §7 exists to avoid |
| The exact gapless trim | Computed from `RunTimeTicks` |
| The truncation guard on every capped stream | It compares decoded audio against `RunTimeTicks` and stalls when it falls 2 s short. With no duration there is nothing to compare, so a truncated transcode ends the track early instead |

**Nothing tells anybody.** The client's own failure-mode warning applies exactly — *"'it played
fine' with no log open proves very little"* — and here it would be true in the strongest sense: the
audio is correct, and four features are absent.

**One root cause, two clients, two failures that look nothing alike.** That is the argument for
fixing it in the source builder rather than in `PlaybackInfo`, where the tvOS symptom happens to
live.

It also needed a [behaviours](behaviours.md) entry it did not have — §5, *accepted gaps*, is where
a bounded shortfall with a named closing mechanism goes, and the mechanism is "a rescan", which is
not something a client can ask for. **Written at 008 T14**, sharing one row with the tvOS client's
§4.1, because it is one branch and the two symptoms are the same skip.

### 5.2 The capped renderer stream is sized for `wav`, and the client asks for FLAC

When a renderer's sample-rate ceiling is below the track's, this client bypasses its quality tier
entirely and asks for a rate-capped lossless stream: `/Audio/{id}/stream.{wav|flac}` with
`AudioCodec`, `AudioSampleRate=48000`, `MediaSourceId` and `PlaySessionId`. Its **Emby** driver
asks for PCM/`wav`; its **Jellyfin** driver asks for FLAC `[client-contract: 2026-08-29, §7]`. So
against Atrium it is always the FLAC branch.

That branch is chunked. The sized path is taken only for a remux or for a container that
`needs_seeking` ([`api/delivery.py:531-546`](../../src/atrium/api/delivery.py)), and
`NEEDS_SEEKING` has exactly one row: `frozenset({"wav"})`
([`media/ffmpeg.py:195`](../../src/atrium/media/ffmpeg.py)). A rate cap forces a re-encode, so a
capped FLAC is neither — it goes to [`_chunked`](../../src/atrium/api/delivery.py), which answers
`200` with `Accept-Ranges: none`, no `Content-Length` and no range handling at all
([`api/delivery.py:812-846`](../../src/atrium/api/delivery.py)).

**This is parity, deliberately.** [behaviours §3.3](behaviours.md#33-progressive-transcoding-responses-carry-no-content-length-or-accept-ranges--class-c)
states the rule and its one exception in as many words: *send the size when it is known*, plus
*produce somewhere seekable when the body would otherwise lie about it* — and **"the one place
Atrium does not diverge is a progressive re-encode whose final length is unknown until the last
frame."** A capped FLAC is exactly that. So the client's `BufferedPassthrough` — fetch the whole
capped stream to a temp file and count it, 321 MB and 7.1 s for a 25-minute track — stays alive
against this server, and that is the specified outcome rather than an oversight. The ask to change
it is [§6.1](#61-an-honest-content-length-on-a-capped-transcode).

**One thing about it is not parity, and it is not the header.** The reference produces progressive
output to a file and streams the file as it grows ([`media/ffmpeg.py:31-34`](../../src/atrium/media/ffmpeg.py));
Atrium produces to a pipe. A FLAC written to a pipe cannot have its `STREAMINFO` block completed,
so it declares `total_samples = 0` and an all-zero MD5, and `ffprobe` reports its duration as `N/A`
— against `3.000000` for the identical encode written to a file (measured locally with ffmpeg's
`flac` muxer, 2026-08-29). The body is playable and the client counts bytes rather than reading
`STREAMINFO`, so nothing breaks today; it is recorded because it is the same root cause as
[§5.3](#53-a-piped-mp3-carries-no-xing-frame-which-is-not-the-blank-one-the-client-measured), where
it does break something, and because a renderer pointed straight at this route would be reading a
container that declares no duration.

### 5.3 A piped MP3 carries no Xing frame, which is not the blank one the client measured

This is the finding worth the most, because it is the one place where Atrium is **further from the
reference than the parity rule intended**, and the parity rule is why.

Every capped quality tier on this client resolves to progressive MP3 on `/universal`
`[client-contract: 2026-08-29, §4]`. Lossy encoders pad both ends, and the trim that removes the
padding is read out of the MP3's Xing/LAME header frame. The client's table has four branches:

| The stream says | What the client does |
|---|---|
| A **complete** Xing/LAME header | Exact trim: `delay`, `padding`, `frames × framesPerPacket` |
| A **blank** header from a `LAME`/`Lavc` encoder | Assumes 576 samples of priming, computes the length from the library duration |
| A blank header from any other encoder name | Trims nothing — an audible microcut at every track change |
| An `m4a` packet table | Exact |

Its measurement of both reference servers is the second row: *"the live transcode writes the Xing
header but leaves it blank — `frames=0 delay=0 padding=0` — because the values aren't known until
encoding finishes."* That is a third-party claim about Jellyfin and therefore a **lead, not a
measured behaviour**; it is also exactly what a file destination streamed while still being written
would produce, which is what the reference does.

**Atrium's answer is a fifth case the table does not have: there is no header frame.** Measured
locally on 2026-08-29, encoding the same three seconds through `libmp3lame` twice — once with
`-f mp3 pipe:` and once to a file:

- the file destination writes an `Info` frame at byte 65, carrying a frame count and the encoder
  string `Lavc63.1`;
- the pipe destination writes no `Xing` and no `Info` frame anywhere in the body — the first audio
  frame follows the ID3 tag directly, and the two files differ by 417 bytes, which is one frame at
  128 kbps.

ffmpeg reserves that frame and seeks back to fill it in at the end; to a pipe it cannot, so it does
not write one. Atrium pipes every progressive re-encode
([`api/delivery.py:812-846`](../../src/atrium/api/delivery.py)) and `mp3` is in neither
`NEEDS_SEEKING` nor `NEEDS_FRAGMENTING` ([`media/ffmpeg.py:195`](../../src/atrium/media/ffmpeg.py),
[`:167`](../../src/atrium/media/ffmpeg.py)), so nothing catches it.

**What the user hears:** a microcut at every track change, because no branch of that table fires —
plus the loss of the MP3 seek index, which the client already builds for itself.

**Why `NEEDS_SEEKING` correctly did not catch this.** Its rule is *a body that would lie about its
own length* — a piped `wav` writes `ffffffff` where the size goes
([`media/ffmpeg.py:188-195`](../../src/atrium/media/ffmpeg.py)). A piped MP3 does not lie. It omits
a frame the reference's body contains, which is a different property, and the set was written
before there was a client whose gapless engine reads it.

**This is a parity gap, not an improvement.** That distinction is what separates it from §6: the
reference's chunked MP3 carries a Xing frame and Atrium's does not, so the fix moves *towards* the
reference and needs no Principle I argument. It also does not need a `Content-Length`: producing to
somewhere seekable and streaming the result gives the frame, whatever the headers then say. The
question a probe has to settle first is the client's claim itself — whether the reference's
progressive MP3 really does carry a blank Xing frame — because it decides whether the target is
"blank, like the reference" or "complete, which is better than the reference and therefore §6's
kind of question".

### 5.4 Every `/universal` request re-encodes, for a different reason than the reference does

The client sends `PlaySessionId=<deviceId>_<itemId>` on `/universal`, deterministic and stable
across every request for that track, so that a server can tie the transcode job to the session. Its
measurement of the reference is that Jellyfin's `/universal` does not declare the parameter, mints a
fresh GUID per call, and — because that id is part of the transcode file's key — **re-encodes the
whole track on every request**: three identical `/universal` requests, three ffmpeg invocations
`[client-contract: 2026-08-29, §4]`. Third-party, and a lead rather than a measured behaviour, but
one with an upstream issue number attached.

[`api/universal_audio.py:271-295`](../../src/atrium/api/universal_audio.py) declares twenty
parameters and `playSessionId` is not among them, so the client's value is dropped — and, because
the ignored-parameter recorder counts every undeclared parameter a real client sends
([`compat/query_params.py`](../../src/atrium/compat/query_params.py)), it is dropped *visibly*,
which is the one good thing here.

**The outcome matches the reference's and the cause does not**, and the difference matters to
whoever fixes it. There is no GUID: the chunked branch has **no cache at all**. Every request runs
`ledger.start` and streams a fresh pipe. Meanwhile the *sized* branch caches better than a session
id ever could — [`_to_scratch`](../../src/atrium/api/delivery.py) names its output after the
command and the file's change signal, so identical requests from different devices and different
sessions collide on one produced file, and a `Range` is served from what the first request made.

So the machinery the client is asking for already exists here, keyed on something stronger than
what it offers, and it is switched on for one branch and off for the other. What that means for the
ask is [§6.2](#62-keying-a-transcode-on-a-client-supplied-playsessionid).

Two costs this carries today, both from the contract: a reconnect after a dropped connection
re-encodes from zero rather than resuming, and the download layer's *"re-request once"* retry —
which exists for a defect Atrium does not have (§4, `m4a`) — costs a full re-encode if it ever
fires for another reason.

### 5.5 `LocalAddress` is plain HTTP at defaults, and an operator can take that away

The client hands a DLNA renderer a URL repointed at the server's LAN address, read from the
authenticated `/System/Info`. Embedded renderers with no usable TLS stack accept an HTTPS URI,
report `PLAYING` and never advance, so the client **refuses to repoint at all** when `LocalAddress`
is `https://` — measured against a Jellyfin with `EnableHttps` on, which answers
`https://<ip>:8920` `[client-contract: 2026-08-29, §7]`. The renderer then gets the public URL,
which is the case the whole feature exists to avoid.

Atrium's answer is `http://` and the reasoning is already written down:
[`net/address.py:87-93`](../../src/atrium/net/address.py) builds the tier-3 address with a literal
`http` under a comment saying the scheme comes from what this server actually serves, and
[behaviours §4.2](behaviours.md#42-localaddress-does-not-get-an-https-override) argues the
divergence on the grounds that v1 terminates no TLS and holds no certificate, so the state in which
the reference rewrites the scheme cannot be configured here at all.

**That argument is true of tier 3 and not of tier 1.** An operator who sets `PublishedUrl` gets it
back verbatim ([`net/address.py:80-81`](../../src/atrium/net/address.py)), and an operator behind a
reverse proxy will set it to the `https://` URL their users type — which is correct for every other
consumer of that field and reintroduces this client's measured refusal exactly. Tier 2,
`use_request_host`, has the same shape.

**This is a documentation gap rather than a code one**, and probably an operator-guidance one: the
`PublishedUrl` setting's own comment ([`config/settings.py:44`](../../src/atrium/config/settings.py))
says what it is for and not what it costs. behaviours §4.2's argument needs the sentence *"except
where an operator publishes an HTTPS URL, and then the reference's behaviour and ours coincide
again"* — which is not written there because a change is in flight against that document.

### 5.6 Every error this server returns costs three requests, not one

The client's HTTP layer maps 401 to `Unauthorised`, 403 to `Forbidden`, and **everything else — 4xx
included — to `ServerFault`, retried three times with 100/200 ms backoff** `[client-contract:
2026-08-29, §0]`. A `400` costs three requests; so does a `404`.

Nothing about that is a requirement on the server, and there is nothing to implement. It is
recorded because it changes the cost of two things this repository has decided deliberately:

- [behaviours §3.9](behaviours.md#39-an-unparseable-mediasourceid-is-a-500-where-a-well-formed-one-is-a-400--class-a-diverged)
  and the four error shapes of [§1.11](behaviours.md#111-there-are-four-error-shapes-not-one) —
  every refusal this server is careful to spell correctly is a refusal this client will ask for
  three times;
- the `404`-per-chapter and absent-image answers of 006, and any route this client reaches with a
  parameter v1 ignores.

The load implication is small and the diagnostic one is not: a log showing three identical `400`s
is one client request, not a client retry loop worth investigating.

### 5.7 A suspended preload is an idle connection a deployment can cut

The client opens the next track while the current one is still playing, and applies backpressure by
**suspending the data task** above an 8 MB look-ahead, resuming below 3 MB. A preloaded track can
therefore hold an open, completely idle TCP connection for the entire length of the current track;
the client raises its own timeout to 3600 s for exactly this and reads a server-side close as a
truncation `[client-contract: 2026-08-29, §3]`.

So a server serving this client should expect **two concurrent open streams per device, one of them
idle for minutes**, and the contract names the risk directly: default request and idle timeouts in
the usual ASGI servers will cut it.

Nothing in `src/` closes an idle response, so the application is fine. What is not written anywhere
in this repository is the deployment requirement, and [008
§4](../../specs/008-playback-negotiation-and-delivery/spec.md)'s operational notes are the natural
home for it. Recorded here as owed prose, with a named consumer.

### 5.8 The album play queue is correctly ordered by accident

The client's album detail *is* its play queue, requested with
`SortBy=ParentIndexNumber,IndexNumber,SortName` `[client-contract: 2026-08-29, §10]`.

[`domain/queries.py:45-52`](../../src/atrium/domain/queries.py) is the whole `sortBy` vocabulary
and it has eight members: `SortName`, `DateCreated`, `PremiereDate`, `PlayCount`, `DatePlayed`,
`Random`, `AlbumArtist`, `Artist`. `ParentIndexNumber` and `IndexNumber` are not among them, and an
unrecognised token is **dropped, never rejected**
([behaviours §1.12](behaviours.md#112-an-unrecognised-query-value-is-ignored-not-rejected)). So two
thirds of that request evaporates and the order is `SortName` alone.

**Which is the right order, for a reason that has nothing to do with the request.** A track's sort
name is not derived the way an album's is: for `Audio`, `Episode` and `Season` the reference builds
a zero-padded numeric prefix followed by the raw name, and for `Audio` that prefix is **disc and
track, each padded to four** ([behaviours §2.6](behaviours.md#26-sortname-has-two-derivations-and-three-types-use-the-second),
lines 634-637). Sorting by `SortName` alone therefore sorts by disc, then track, then name — which
is what `ParentIndexNumber,IndexNumber,SortName` asked for, arrived at from the other end.

The same paragraph covers the client's other sort: `SortBy=Year` is sent as
`ProductionYear,PremiereDate,SortName`, and `ProductionYear` is not in the vocabulary either, so
that request is really `PremiereDate,SortName`. The client already measured that this is where both
reference servers get it wrong — *"'newest' in a library with 2026 albums came back 2023"* — which
is a lead worth a probe of its own, since a `PremiereDate`-only ordering of albums that mostly lack
one is the same shape of wrong here.

**The finding is not the ordering. It is that nothing states the dependency.** No test in this
repository asserts that an album's tracks come back in disc-and-track order, and the two facts that
make it true — the audio sort-name derivation, and `sortBy` dropping unknown tokens — live in
different features and neither one mentions the other. A change to either breaks the play queue of
a shipping client, and the suite stays green.

**The cheapest fix is a test, not a feature.** One case, asserting the order of a multi-disc album's
tracks under this client's exact `sortBy` string, converts an accident into a stated behaviour.

Adding the three keys to the vocabulary is the *expensive* answer, and it is probably the wrong
one. The enum's own docstring makes the argument: a ninth member *"would be a key no reference
server orders by, which is a delta in the one direction Principle I has no tolerance for."* Whether
the reference really refuses `ParentIndexNumber`, `IndexNumber` and `ProductionYear` there is the
question that decides it, and the eight members rest on a
`[prior-probe: Jellyfin 10.11.11, 2026-06-13]` that this repository has not yet re-run — a debt
[behaviours §2.5](behaviours.md#25-sortby-vocabulary) already carries. The client sending three
keys outside the eight is the first evidence that the debt is worth paying.

## 6. Two open questions, and neither is a failure

The contract makes two asks that are **improvements over both reference servers, not parity**. They
are Principle I questions, and this section exists so that they are not read as requirements the
server has failed. The decision is to **measure first**: neither is scoped, and both need a probe of
the reference before an argument about them means anything.

### 6.1 An honest `Content-Length` on a capped transcode

**The ask.** *"No transcoding route on either server returns `Content-Length`, and a renderer
refuses a body of unknown size"* — which is the entire reason the client ships a local HTTP proxy on
both platforms. A server that sends an honest length and honours `Range` on the capped stream lets
a renderer be pointed straight at it, and the proxy, the buffering, the 300 MB temporaries and a
whole per-server delivery strategy become dead code `[client-contract: 2026-08-29, §7]`.

**Why it is a Principle I question and not a bug.** [behaviours §3.3](behaviours.md#33-progressive-transcoding-responses-carry-no-content-length-or-accept-ranges--class-c)
already decided the general rule and already diverges from the reference wherever the size is
knowable before the first byte. This ask is to extend that divergence to a case where the size is
*not* knowable before the first byte — you get it by producing the whole file first — which is a new
kind of divergence rather than more of the accepted one, and it changes latency as well as headers.

**What is cheap about it, and what is not.** The machinery exists:
[`_to_scratch`](../../src/atrium/api/delivery.py) already produces to a named, deterministic scratch
file and serves ranges from it, and the branch is chosen by one frozenset
([`media/ffmpeg.py:195`](../../src/atrium/media/ffmpeg.py)). Adding a container to that set is a
one-row change. What is not cheap is the consequence: the first byte then arrives after the *whole*
track is encoded, against this client's 20 s open budget, on a route whose whole point is to start
fast. The client's own warning is the sharpest input available — *"an estimated length is worse than
none"*, measured at 50% long for lossless — so half-measures are ruled out before they are proposed.

**Its own warning also bounds it.** Emby's `EstimateContentLength` is the failed version of this;
the working version is the one where the number is real.

### 6.2 Keying a transcode on a client-supplied `PlaySessionId`

**The ask.** Accept a client `PlaySessionId` on `/universal` and key the transcode cache on it, so
every re-request, reconnect and retry becomes a cache hit `[client-contract: 2026-08-29, §4]`.

**Two separable halves, and they have different answers.** Declaring the parameter is a **delta**:
the reference does not declare it there `[spec: GetUniversalAudioStream]` — nineteen parameters,
`playSessionId` not among them — and adding a query parameter Jellyfin has not got is the
thing Principle I forbids most plainly — even a benign one, even one a client already sends.
Caching the chunked branch is **not obviously a delta at all**: a client cannot observe a response
being faster, and §5.4 shows this repository already caches the sized branch on a key that is
strictly better than a session id, because it collides across devices and sessions rather than
within one.

So the shape of the answer is probably *neither what was asked for nor a refusal*: extend the
content-addressed cache to the chunked branch, and the parameter never needs declaring. That
disposes of the delta and gets the client the behaviour it wants. It is written here as an open
question rather than as a plan because it is entangled with §6.1 — both are about producing a
progressive re-encode somewhere other than a pipe — and because the reference's own behaviour here
is a third-party claim that no probe in this repository has checked.

## 7. Where these findings go

**None of this is an 008 defect.** 008 closes on its accepted scope, and every finding above was
measured against code that does what its spec, plan and tasks say it does. §5.3 is the one that
comes closest to being a defect — it is a divergence from the reference — and it is a divergence
008 could not have anticipated, because the property it turns on is inside a body rather than in a
header, and no client this repository had analysed read it.

They are input to the feature that comes after 010, alongside the eight in
[client-atrium-tvos.md §6](client-atrium-tvos.md#6-where-these-findings-go). Grouped as a scope
would take them:

| Finding | Shape of the work |
|---|---|
| [§5.1](#51-a-source-with-no-stored-inspection-loses-the-music-clients-whole-negotiation) | Shared with the tvOS client's §4.1 and worth doing once. One branch, and a behaviours §5 entry |
| [§5.3](#53-a-piped-mp3-carries-no-xing-frame-which-is-not-the-blank-one-the-client-measured) | A probe of the reference's progressive MP3 first; then a destination change, not a header change |
| [§5.4](#54-every-universal-request-re-encodes-for-a-different-reason-than-the-reference-does) + [§6.1](#61-an-honest-content-length-on-a-capped-transcode) + [§6.2](#62-keying-a-transcode-on-a-client-supplied-playsessionid) | One question about where a progressive re-encode is produced, asked three ways. Settle it once |
| [§5.8](#58-the-album-play-queue-is-correctly-ordered-by-accident) | One test, and it can be written today. The only item here that needs no decision |
| [§5.5](#55-localaddress-is-plain-http-at-defaults-and-an-operator-can-take-that-away), [§5.7](#57-a-suspended-preload-is-an-idle-connection-a-deployment-can-cut) | Prose: a sentence in behaviours §4.2, a paragraph of deployment guidance |
| [§3](#3-the-twenty-seven-operations-and-the-one-that-is-not-in-the-55) | A 009 scope decision, **taken** when 009's spec was accepted on 2026-08-31: `POST /Items/{itemId}` entered the surface with this document as its named consumer, and shipped at 009 T13 |

Three [behaviours](behaviours.md) entries were owed here and **two are now written**, both at 008
T14 because both describe what 008 ships: §5.1's accepted-gap entry (§5), and §5.2's and §5.3's
shared note that the pipe destination costs a container its own self-description, which landed in
§3.3 as a divergence rather than as a footnote. **§5.5's sentence on §4.2's argument stays owed** —
it is about `LocalAddress` at tier 1 and tier 2, which is 001's field and a deployment-guidance
change, so it belongs neither to 008 nor to 011.

## 8. What this document does not do

**It does not grow the surface.** One operation is outside the 55 (§3) and it is handed to 009's
spec review rather than promoted here. Nothing else this client calls is missing.

**It does not become a second endpoint table.** [`surface.yaml`](surface.yaml) is the surface, and
`consumers: [music-client]` is where this client is already named; §3 rolls up rather than restating
rows that would then drift.

**It does not promote a third party's measurements.** This contract is unusually rich in them —
dated, tool-named, with upstream issue numbers — and every one of them is still a lead. Two of them
happen to agree with probes this repository ran independently (the by-name `TotalRecordCount`, and
the PCM/WAV routes of [behaviours §3.2](behaviours.md#32-pcmwav-output--one-bug-two-symptoms-two-classes)),
which is corroboration and not promotion.

**It is a floor, not a ceiling.** It describes the client at one commit on 2026-08-29, from its own
source. When the client changes, this document is stale and nothing in CI will notice.

The mechanism that turns any of this into something measured is the differential harness of
[010](../../specs/010-conformance-harness/spec.md). What this document contributes to it is
twenty-seven rows and eight behaviours — and the observation that on a client with no negotiation,
**the differential's item-listing comparison is also its playback comparison**, because for this
client those are the same request.
