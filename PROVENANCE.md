# Provenance

These documents were exported from another repository. They are specifications and
measurements; no implementation came with them, deliberately.

- source: https://github.com/vdatanet/atrium-media-server.git
- ref: `HEAD`
- commit: `531c1728fd07f90960f792217b993a7dad7e8f12`
- exported: 2026-09-02
- export digest: `sha256:997126f1a7c78268d490cbb7cd67122a4a7d9c5bc55e0f680ae0ba6c753205d8`
- files: 37 exported, 513 withheld

Licence: GPL-3.0-or-later, as it was at the source. `LICENSE` travelled with them.

**The exporting worktree had uncommitted changes to exported paths.** The bytes here
are the committed ones at the ref above, so those changes are *not* in this export:

- `specs/010-conformance-harness/spec.md`
- `specs/README.md`
## What was withheld, and why

| Reason | Files |
|---|---|
| Agent configuration pointing at this repository | 1 |
| An index of documents this export withholds | 1 |
| Audits of this implementation | 3 |
| Build configuration of the exporting stack | 3 |
| CI wired to the exporting stack | 1 |
| Credentials and endpoints of this installation | 1 |
| HOW -- inheriting it makes the second implementation a transliteration | 11 |
| Ignores paths the receiving project does not have | 1 |
| Probes and harness stay here and are pointed at the new server over HTTP | 69 |
| STEPS -- they belong to a plan the receiving project has not written | 11 |
| Stack, store and password hashing: decisions the receiving project takes for itself | 3 |
| Tests written against the implementation | 271 |
| The implementation itself | 133 |
| This implementation's front page | 1 |
| This implementation's order of work | 1 |
| This implementation's shape | 1 |
| Working instructions naming this repository's layout and commands | 1 |

## What the receiving project must decide first

These specifications carry a `status:` that is a statement about the *exporting*
project. Nothing here is implemented until this project implements it, and this
command does not edit an exported byte:

| File | `status:` at the source |
|---|---|
| `specs/001-server-identity-and-discovery/spec.md` | Implemented |
| `specs/002-authentication-users-and-sessions/spec.md` | Implemented |
| `specs/003-library-configuration-and-scanning/spec.md` | Implemented |
| `specs/004-metadata-resolution/spec.md` | Implemented |
| `specs/005-item-query-api/spec.md` | Implemented |
| `specs/006-images/spec.md` | Implemented |
| `specs/007-user-data-and-playstate/spec.md` | Implemented |
| `specs/008-playback-negotiation-and-delivery/spec.md` | Implemented |
| `specs/009-playlists/spec.md` | Implemented |
| `specs/010-conformance-harness/spec.md` | Accepted |
| `specs/011-subtitle-delivery/spec.md` | Implemented |
| `specs/012-negotiation-inputs/spec.md` | Accepted |

## Leaks

Lines in the exported documents that name a technology, or point at a file that stayed
behind. A probe citation is not among them: the probes measure the reference, they remain
in the source repository, and this project points them at its own server rather than
rewriting them. **157 lines**, by file:

| File | Lines |
|---|---|
| `docs/compatibility/allowlist.yaml` | 1 |
| `docs/compatibility/api-surface-v1.md` | 2 |
| `docs/compatibility/behaviours.md` | 27 |
| `docs/compatibility/client-atrium-tvos.md` | 40 |
| `docs/compatibility/client-embeat-mobile.md` | 41 |
| `docs/compatibility/conformance.md` | 9 |
| `docs/compatibility/named-comparisons.yaml` | 1 |
| `docs/compatibility/reference-fixture-reading.json` | 2 |
| `docs/compatibility/reference-target.md` | 4 |
| `docs/compatibility/request-cases.yaml` | 4 |
| `docs/constitution.md` | 3 |
| `docs/decisions/0005-licence.md` | 1 |
| `docs/decisions/0007-a-container-runtime-for-the-reference-instance.md` | 6 |
| `docs/decisions/README.md` | 2 |
| `specs/005-item-query-api/spec.md` | 1 |
| `specs/010-conformance-harness/spec.md` | 1 |
| `specs/011-subtitle-delivery/spec.md` | 1 |
| `specs/README.md` | 11 |

<details><summary>Every line</summary>

- `docs/compatibility/allowlist.yaml:4` (path) — # section 3.3, which carry the same list in prose. tests/unit/test_allowlist.py compares the two
- `docs/compatibility/api-surface-v1.md:90` (path) — > `compat/auth.py` has read it since; only this table had not caught up.
- `docs/compatibility/api-surface-v1.md:275` (technology) — | Plugins, packages, repositories | .NET assembly loading; no Python analogue worth inventing |
- `docs/compatibility/behaviours.md:42` (technology) — > ⚠️ This is the single most likely source of a silent, total incompatibility, because Python's
- `docs/compatibility/behaviours.md:125` (path) — `tests/library/test_root_move.py` moves a scanned library to another mount, reconfigures the root
- `docs/compatibility/behaviours.md:260` (path) — **Atrium does:** the same, in `compat/responses.py` — **which reversed §4.4**, an exception taken
- `docs/compatibility/behaviours.md:316` (path) — > `is_hidden` to `false` (`src/atrium/domain/user.py`, and the `0` server default of the
- `docs/compatibility/behaviours.md:497` (path) — song called `24K Magic` and not track 24 of `K Magic`. `tests/corpus/naming.yaml` pins both that and
- `docs/compatibility/behaviours.md:1024` (technology) — belongs to the thing that produced the body. Starlette appends `charset=utf-8` only to `text/*`
- `docs/compatibility/behaviours.md:1036` (path) — | A method the path does not have | `405`, **empty body**, no `Content-Type`, and `Allow` naming every method that path has `[probe: tools/probe_routing.py, Jel
- `docs/compatibility/behaviours.md:1072` (technology) — ASP.NET Core and every Python framework default to the second for a problem-details body, so
- `docs/compatibility/behaviours.md:1131` (path) — that reproduces them (`compat/model.py`'s `WIRE_TYPE` and `WIRE_ENUM_TYPES`), and nothing about how
- `docs/compatibility/behaviours.md:1187` (path) — `api/deps.py` refuses a live token whose account was disabled afterwards — 002 OQ-5's third row,
- `docs/compatibility/behaviours.md:1189` (path) — `api/users.py` refusing one user reading another, and **the reference does not refuse that at
- `docs/compatibility/behaviours.md:1210` (technology) — here keys on the **model's Python field**, so the first typed request body in this project answered
- `docs/compatibility/behaviours.md:1295` (technology) — **Depends on it:** a client branching on a body it expects to be JSON. FastAPI's own
- `docs/compatibility/behaviours.md:1462` (technology) — > **The framework's default here is not neutral, it is a third behaviour.** Starlette answers an
- `docs/compatibility/behaviours.md:1709` (path) — whole, which is where the length and the `Range` come from. `media/ffmpeg.py` refuses to build the
- `docs/compatibility/behaviours.md:1711` (path) — `tests/unit/test_media_ffmpeg.py` beside the measurement in
- `docs/compatibility/behaviours.md:1712` (path) — `tests/conformance/test_wav_delivery.py`.
- `docs/compatibility/behaviours.md:2059` (path) — `tests/conformance/test_progressive_delivery.py` and again in
- `docs/compatibility/behaviours.md:2060` (path) — `tests/conformance/test_universal_audio.py`, so the two values are visibly one answer.
- `docs/compatibility/behaviours.md:2090` (path) — `tests/conformance/test_hls_segments.py`.
- `docs/compatibility/behaviours.md:2141` (path) — asserted over the wire in `tests/unit/test_transcode_lifecycle.py`. The divergence takes
- `docs/compatibility/behaviours.md:2770` (path) — > implemented the escaping in `compat/responses.py` and recorded it as §1.16. The reversal never
- `docs/compatibility/behaviours.md:2808` (path) — if one exists, the fix belongs in `compat/responses.py` for every endpoint at once, not in the
- `docs/compatibility/behaviours.md:2838` (path) — | **A required body that is missing entirely is `400` and not `415`** ([§1.11](#111-there-are-four-error-shapes-not-one)) | A request carrying no body and no `C
- `docs/compatibility/behaviours.md:3091` (path) — `tests/conformance/test_image_routes.py`'s tripwire (`test_no_v1_writer_can_create_a_chapter_row`)
- `docs/compatibility/behaviours.md:3174` (path) — (`media/extract.py`'s `_detect`): a **byte order mark** decides outright — the four a text file
- `docs/compatibility/behaviours.md:3197` (path) — [`pyproject.toml`](../../pyproject.toml)'s own rule is that the dependency set says what the code
- `docs/compatibility/client-atrium-tvos.md:85` (path) — | §0 `Authorization: MediaBrowser Client=…, DeviceId=…`, token appended | Either header spelling is read for the client identification ([`compat/auth.py:146-152
- `docs/compatibility/client-atrium-tvos.md:86` (path) — | §1 The eleven-field `PlaybackInfo` body | Every one of them is bound ([`api/media_info.py:190-205`](../../src/atrium/api/media_info.py)); `EnableTranscoding: 
- `docs/compatibility/client-atrium-tvos.md:87` (path) — | §1 `MediaSources[0]` exists and has `Id` | Emitted for every part of every item, inspection or no inspection ([`media/info.py:410-438`](../../src/atrium/media
- `docs/compatibility/client-atrium-tvos.md:89` (path) — | §1 `TranscodingUrl` present whenever direct is refused | The reference's own condition, transcribed ([`api/media_info.py:381-388`](../../src/atrium/api/media_
- `docs/compatibility/client-atrium-tvos.md:91` (path) — | §1 `Size` is the byte length of the file being served | Read from the stored part, so it survives a missing inspection ([`media/info.py:427`](../../src/atrium
- `docs/compatibility/client-atrium-tvos.md:92` (path) — | §1 `MediaStreams[].IsTextSubtitleStream` | Emitted on every stream since 011 T2, beside `SupportsExternalStream`, read off the codec spelling the reference re
- `docs/compatibility/client-atrium-tvos.md:93` (path) — | §1 `DeviceProfile.TranscodingProfiles[].EnableSubtitlesInManifest: true` | Bound since 011 T9 ([`api/media_info.py`](../../src/atrium/api/media_info.py)) and 
- `docs/compatibility/client-atrium-tvos.md:94` (path) — | §1 `DeviceProfile.TranscodingProfiles[].Protocol` selects HLS | Compared case-sensitively against `"hls"` ([`media/urls.py:202`](../../src/atrium/media/urls.p
- `docs/compatibility/client-atrium-tvos.md:95` (path) — | §2 `Range` must answer `206`, never `200` | [`compat/ranges.py:87-140`](../../src/atrium/compat/ranges.py): a well-formed `bytes=lo-hi` inside the file is `PA
- `docs/compatibility/client-atrium-tvos.md:97` (path) — | §3 The master carries `VIDEO-RANGE`, `CODECS`, `FRAME-RATE` | [`media/hls.py:357-364`](../../src/atrium/media/hls.py) writes all three, on every variant | ✅ |
- `docs/compatibility/client-atrium-tvos.md:98` (path) — | §3 The master announces subtitle tracks | Since 011 T11: one `#EXT-X-MEDIA:TYPE=SUBTITLES` per text subtitle stream, and **every** variant line ends in the gr
- `docs/compatibility/client-atrium-tvos.md:99` (path) — | §3 `…/Subtitles/{index}/Stream.vtt` when the manifest carries none | Three rows of [`surface.yaml`](surface.yaml) since 011's spec gate, served since 011 T7 a
- `docs/compatibility/client-atrium-tvos.md:100` (path) — | §3 `AudioStreamIndex`/`SubtitleStreamIndex` overridden on the stream URL | The audio half is a delivery parameter and is honoured ([`api/delivery.py`](../../s
- `docs/compatibility/client-atrium-tvos.md:101` (path) — | §3 `GET /Sessions?deviceId=…` for copy verification | The route takes no `deviceId` ([`api/sessions.py:287-293`](../../src/atrium/api/sessions.py)) | 🟠 [§4.4]
- `docs/compatibility/client-atrium-tvos.md:103` (path) — | §3 Workaround 1 — the fMP4 init segment starts a second transcode | Reproduced ([`media/sessions.py:499-506`](../../src/atrium/media/sessions.py)) | 🔴 [§4.5](
- `docs/compatibility/client-atrium-tvos.md:104` (path) — | §3 Workaround 2 — the session key contains the `User-Agent` | **Not** reproduced: device, play session and path ([`media/sessions.py:151-164`](../../src/atriu
- `docs/compatibility/client-atrium-tvos.md:289` (path) — happens ([`api/media_info.py:479-483`](../../src/atrium/api/media_info.py)). What the client
- `docs/compatibility/client-atrium-tvos.md:290` (path) — receives is the intrinsic shape from [`media/info.py:410-438`](../../src/atrium/media/info.py):
- `docs/compatibility/client-atrium-tvos.md:293` (path) — ([`media/info.py:176`](../../src/atrium/media/info.py)) and nothing overwrote it. No
- `docs/compatibility/client-atrium-tvos.md:345` (path) — ([`media/hls.py`](../../src/atrium/media/hls.py)'s `master_playlist`): the block is written
- `docs/compatibility/client-atrium-tvos.md:350` (path) — [`api/media_info.py`](../../src/atrium/api/media_info.py) declared eleven properties of a
- `docs/compatibility/client-atrium-tvos.md:352` (path) — ([`compat/model.py:67`](../../src/atrium/compat/model.py)) dropped it on arrival. The client
- `docs/compatibility/client-atrium-tvos.md:405` (path) — [`api/delivery.py:166`](../../src/atrium/api/delivery.py), read at
- `docs/compatibility/client-atrium-tvos.md:406` (path) — [`:212`](../../src/atrium/api/delivery.py), and honoured at
- `docs/compatibility/client-atrium-tvos.md:407` (path) — [`:625`](../../src/atrium/api/delivery.py), where `_audio_stream` picks the stream whose index the
- `docs/compatibility/client-atrium-tvos.md:413` (path) — ([`api/media_info.py`](../../src/atrium/api/media_info.py)) and on the playstate reports and
- `docs/compatibility/client-atrium-tvos.md:414` (path) — nowhere in [`api/delivery.py`](../../src/atrium/api/delivery.py), so a delivery request carrying it
- `docs/compatibility/client-atrium-tvos.md:419` (path) — It is now bound in [`api/delivery.py`](../../src/atrium/api/delivery.py)'s shared video parameter
- `docs/compatibility/client-atrium-tvos.md:439` (path) — ([`api/media_info.py:312`](../../src/atrium/api/media_info.py)), so it may well not have the
- `docs/compatibility/client-atrium-tvos.md:449` (path) — [`api/sessions.py:287-293`](../../src/atrium/api/sessions.py) declares no `deviceId`, so the
- `docs/compatibility/client-atrium-tvos.md:475` (path) — Atrium's restart rule is [`media/sessions.py:499-506`](../../src/atrium/media/sessions.py), whose
- `docs/compatibility/client-atrium-tvos.md:479` (path) — ([`media/hls.py:266-268`](../../src/atrium/media/hls.py)), so every resumed fMP4 playback pays for
- `docs/compatibility/client-atrium-tvos.md:501` (path) — [`media/urls.py:202`](../../src/atrium/media/urls.py) and
- `docs/compatibility/client-atrium-tvos.md:502` (path) — [`:236`](../../src/atrium/media/urls.py) compare `decision.sub_protocol` to the constant `"hls"`
- `docs/compatibility/client-atrium-tvos.md:505` (path) — ([`media/decision.py:1001`](../../src/atrium/media/decision.py) ←
- `docs/compatibility/client-atrium-tvos.md:506` (path) — [`api/media_info.py:145`](../../src/atrium/api/media_info.py), a bare `str`). A profile spelled
- `docs/compatibility/client-atrium-tvos.md:512` (path) — [`api/universal_audio.py:267`](../../src/atrium/api/universal_audio.py) normalises with
- `docs/compatibility/client-atrium-tvos.md:577` (path) — Atrium was fine: [`compat/auth.py:146-152`](../../src/atrium/compat/auth.py) reads either header
- `docs/compatibility/client-atrium-tvos.md:583` (path) — error strings in [`compat/auth.py:137`](../../src/atrium/compat/auth.py) and
- `docs/compatibility/client-atrium-tvos.md:584` (path) — [`:141`](../../src/atrium/compat/auth.py) name only the Emby spelling — a `400` whose message points
- `docs/compatibility/client-embeat-mobile.md:151` (path) — | §0 `X-Emby-Token` on every authenticated request | The second of the three shapes `extract_token` resolves, in the reference's own order ([`compat/auth.py:155
- `docs/compatibility/client-embeat-mobile.md:152` (path) — | §0 `X-Emby-Authorization` on `AuthenticateByName` only, never again | That route requires a client-identification header carrying a `DeviceId` and no other ro
- `docs/compatibility/client-embeat-mobile.md:154` (path) — | §0 PascalCase in both directions, unknown keys ignored | [`compat/model.py:63-68`](../../src/atrium/compat/model.py): alias generator, `populate_by_name`, `se
- `docs/compatibility/client-embeat-mobile.md:155` (path) — | §0 Query parameter names cased inconsistently — six routes, three spellings of the same idea | Rewritten to each route's declared spelling before the framewor
- `docs/compatibility/client-embeat-mobile.md:157` (path) — | §1 `Fields=MediaSources` on six listing routes | Served, and built from the stored inspections ([`api/item_dto.py:502-507`](../../src/atrium/api/item_dto.py))
- `docs/compatibility/client-embeat-mobile.md:158` (path) — | §1 `MediaSources[0].Id`, `.Container`, `.Bitrate`, `RunTimeTicks` | All four emitted ([`media/info.py:410-438`](../../src/atrium/media/info.py)) — three of th
- `docs/compatibility/client-embeat-mobile.md:159` (path) — | §1 `MediaStreams[Audio].Codec`/`SampleRate`/`BitDepth`/`Channels` | All four emitted ([`media/info.py:371-375`](../../src/atrium/media/info.py)) — and this is
- `docs/compatibility/client-embeat-mobile.md:161` (path) — | §2 `ProductName` contains `jellyfin`, or the client uses its **Emby** driver | `REFERENCE_PRODUCT_NAME = "Jellyfin Server"` ([`src/atrium/__init__.py:26`](../
- `docs/compatibility/client-embeat-mobile.md:162` (path) — | §2 `/System/Info/Public`'s `Id` equals `AuthenticateByName`'s `ServerId` | One value: `state.server_id` at [`api/system.py:115`](../../src/atrium/api/system.p
- `docs/compatibility/client-embeat-mobile.md:163` (path) — | §2 `AuthenticateByName` body is `{"Username", "Pw"}`, answer carries `User.Id`/`AccessToken`/`ServerId` | [`api/users.py:147-155`](../../src/atrium/api/users.
- `docs/compatibility/client-embeat-mobile.md:165` (path) — | §3 `Range` honoured, and `Content-Length` load-bearing on this path | [`compat/ranges.py:87-140`](../../src/atrium/compat/ranges.py) and the four measured hea
- `docs/compatibility/client-embeat-mobile.md:167` (path) — | §4 Progressive MP3 on `/universal`, first byte within 20 s | Served at 008 T8, streamed as produced ([`api/delivery.py:812-846`](../../src/atrium/api/delivery
- `docs/compatibility/client-embeat-mobile.md:169` (path) — | §4 `Range` honoured on `/universal` for reconnects | Chunked answers set `Accept-Ranges: none` and read no range at all ([`api/delivery.py:845`](../../src/atr
- `docs/compatibility/client-embeat-mobile.md:170` (path) — | §4 `PlaySessionId` accepted on `/universal`, transcode keyed on it | Not declared ([`api/universal_audio.py:271-295`](../../src/atrium/api/universal_audio.py)
- `docs/compatibility/client-embeat-mobile.md:172` (path) — | §5 HLS on `/universal` (`TranscodingProtocol=hls`) | Served, and normalised case-insensitively ([`api/universal_audio.py:267`](../../src/atrium/api/universal_
- `docs/compatibility/client-embeat-mobile.md:173` (path) — | §6 The two measured download defects — an unreadable first MP4, and `m4a` withholding every byte | Neither is reproducible here: `m4a` muxes to `ipod`, which 
- `docs/compatibility/client-embeat-mobile.md:174` (path) — | §7 `LocalAddress` is plain `http://` | True at defaults ([`net/address.py:87-93`](../../src/atrium/net/address.py), [behaviours §4.2](behaviours.md#42-localad
- `docs/compatibility/client-embeat-mobile.md:175` (path) — | §7 The capped renderer stream: `AudioSampleRate`, `MediaSourceId` on `/Audio/{id}/stream.{ext}` | Both bound ([`api/delivery.py:229-243`](../../src/atrium/api
- `docs/compatibility/client-embeat-mobile.md:178` (path) — | §8 No `GET /Sessions`, no `DELETE` to stop an encoding — a server that keeps ffmpeg alive accumulates jobs | It does not: a client that disconnects makes the 
- `docs/compatibility/client-embeat-mobile.md:179` (path) — | §9 The album-cover pointer: `AlbumId` + `AlbumPrimaryImageTag` on a track | Emitted for `Audio` items ([`api/item_dto.py:174-175`](../../src/atrium/api/item_d
- `docs/compatibility/client-embeat-mobile.md:202` (path) — []`** ([`media/info.py:410-438`](../../src/atrium/media/info.py)). For `PlaybackInfo` the whole
- `docs/compatibility/client-embeat-mobile.md:203` (path) — annotation is skipped ([`api/media_info.py:479-483`](../../src/atrium/api/media_info.py)).
- `docs/compatibility/client-embeat-mobile.md:239` (path) — `needs_seeking` ([`api/delivery.py:531-546`](../../src/atrium/api/delivery.py)), and
- `docs/compatibility/client-embeat-mobile.md:241` (path) — ([`media/ffmpeg.py:195`](../../src/atrium/media/ffmpeg.py)). A rate cap forces a re-encode, so a
- `docs/compatibility/client-embeat-mobile.md:242` (path) — capped FLAC is neither — it goes to [`_chunked`](../../src/atrium/api/delivery.py), which answers
- `docs/compatibility/client-embeat-mobile.md:244` (path) — ([`api/delivery.py:812-846`](../../src/atrium/api/delivery.py)).
- `docs/compatibility/client-embeat-mobile.md:256` (path) — output to a file and streams the file as it grows ([`media/ffmpeg.py:31-34`](../../src/atrium/media/ffmpeg.py));
- `docs/compatibility/client-embeat-mobile.md:300` (path) — ([`api/delivery.py:812-846`](../../src/atrium/api/delivery.py)) and `mp3` is in neither
- `docs/compatibility/client-embeat-mobile.md:301` (path) — `NEEDS_SEEKING` nor `NEEDS_FRAGMENTING` ([`media/ffmpeg.py:195`](../../src/atrium/media/ffmpeg.py),
- `docs/compatibility/client-embeat-mobile.md:302` (path) — [`:167`](../../src/atrium/media/ffmpeg.py)), so nothing catches it.
- `docs/compatibility/client-embeat-mobile.md:309` (path) — ([`media/ffmpeg.py:188-195`](../../src/atrium/media/ffmpeg.py)). A piped MP3 does not lie. It omits
- `docs/compatibility/client-embeat-mobile.md:332` (path) — [`api/universal_audio.py:271-295`](../../src/atrium/api/universal_audio.py) declares twenty
- `docs/compatibility/client-embeat-mobile.md:335` (path) — ([`compat/query_params.py`](../../src/atrium/compat/query_params.py)), it is dropped *visibly*,
- `docs/compatibility/client-embeat-mobile.md:341` (path) — id ever could — [`_to_scratch`](../../src/atrium/api/delivery.py) names its output after the
- `docs/compatibility/client-embeat-mobile.md:364` (path) — [`net/address.py:87-93`](../../src/atrium/net/address.py) builds the tier-3 address with a literal
- `docs/compatibility/client-embeat-mobile.md:371` (path) — back verbatim ([`net/address.py:80-81`](../../src/atrium/net/address.py)), and an operator behind a
- `docs/compatibility/client-embeat-mobile.md:377` (path) — `PublishedUrl` setting's own comment ([`config/settings.py:44`](../../src/atrium/config/settings.py))
- `docs/compatibility/client-embeat-mobile.md:411` (technology) — the usual ASGI servers will cut it.
- `docs/compatibility/client-embeat-mobile.md:423` (path) — [`domain/queries.py:45-52`](../../src/atrium/domain/queries.py) is the whole `sortBy` vocabulary
- `docs/compatibility/client-embeat-mobile.md:484` (path) — [`_to_scratch`](../../src/atrium/api/delivery.py) already produces to a named, deterministic scratch
- `docs/compatibility/client-embeat-mobile.md:486` (path) — ([`media/ffmpeg.py:195`](../../src/atrium/media/ffmpeg.py)). Adding a container to that set is a
- `docs/compatibility/conformance.md:40` (path) — are compared to a checked-in file under `tests/golden/`.
- `docs/compatibility/conformance.md:54` (technology) — PascalCase. Python's ecosystem defaults the wrong way; this makes forgetting impossible rather
- `docs/compatibility/conformance.md:70` (path) — directory and scanned by the real pipeline (`tests/fixtures/media.py`). Tests that reach the
- `docs/compatibility/conformance.md:71` (technology) — binaries carry the `ffmpeg` marker; `pytest -m "not ffmpeg"` staying green is the check that they
- `docs/compatibility/conformance.md:81` (path) — `tests/fixtures/reference_tree.py` is what composes the three, and
- `docs/compatibility/conformance.md:83` (path) — compared against Atrium's own scan by `tests/library/test_reference_reading.py` with no Jellyfin
- `docs/compatibility/conformance.md:209` (path) — and `tests/unit/test_allowlist.py` compares the file against 010 §3.10 row for row. Every row
- `docs/compatibility/conformance.md:245` (path) — Atrium's own scan by `tests/library/test_reference_reading.py`, in the default job, with no Jellyfin
- `docs/compatibility/conformance.md:279` (path) — it. `tests/unit/test_allowlist.py` compares both — this one and
- `docs/compatibility/named-comparisons.yaml:5` (path) — # tests/unit/test_allowlist.py compares this file against that table row for row, so the two
- `docs/compatibility/reference-fixture-reading.json:2` (path) — "_": "The reference's own reading of this repository's fixture tree, written by the probe named below and compared against Atrium's scan by tests/library/test_r
- `docs/compatibility/reference-fixture-reading.json:10` (path) — "tree": "tests/fixtures/library (003's declared tree, paths and filler bytes)",
- `docs/compatibility/reference-target.md:17` (path) — | Reference instance image | `jellyfin/jellyfin@sha256:aefb67e6a7ff1debdd154a78a7bbb780fd0c873d8639210a7f6a2016ad2b35db` — the published Jellyfin `10.11.11` ima
- `docs/compatibility/reference-target.md:73` (path) — > — are all true of a polluted index. `tests/conformance/test_aliases.py`.
- `docs/compatibility/reference-target.md:166` (path) — `tests/unit/test_probe_convention.py` asserts the properties this table has to keep: a struck
- `docs/compatibility/reference-target.md:262` (technology) — - **Plugins.** Jellyfin's plugin API is a .NET assembly-loading contract. There is no Python
- `docs/compatibility/request-cases.yaml:5` (path) — # cover. tests/unit/test_allowlist.py reads the surface through the surface validator's own parser
- `docs/compatibility/request-cases.yaml:21` (path) — # tests/conformance/test_routes.py reads `feature` and `consumers` and never `level`. Meanwhile
- `docs/compatibility/request-cases.yaml:247` (path) — what_it_is_for: "The film the artwork cases anchor on: `The Planted Poster (2011)` of tests/fixtures/media.py, which carries a poster with an EXIF orientation a
- `docs/compatibility/request-cases.yaml:257` (path) — what_it_is_for: "The film the subtitle cases anchor on: `The Unconvertible (2009)` of tests/fixtures/media.py, which carries an `ass` track inside it and a subt
- `docs/constitution.md:80` (technology) — - Naming a library, framework, table or Python module inside `spec.md`. The specification
- `docs/constitution.md:97` (technology) — - Transliterating a Jellyfin C# method into Python.
- `docs/constitution.md:179` (technology) — - Asserting on a parsed Python object where the client sees bytes. Casing, `null`-vs-absent and
- `docs/decisions/0005-licence.md:90` (path) — - `pyproject.toml` declares `license = "GPL-3.0-or-later"`.
- `docs/decisions/0007-a-container-runtime-for-the-reference-instance.md:33` (technology) — **Through the CLI, not through a Python SDK.** `tools/` is standard library only on a Python 3.9
- `docs/decisions/0007-a-container-runtime-for-the-reference-instance.md:44` (path) — this record ([AGENTS.md](../../AGENTS.md), `tests/conftest.py`'s socket guard) and this decision
- `docs/decisions/0007-a-container-runtime-for-the-reference-instance.md:73` (path) — `tests/fixtures/library/generate.py` stamps every file with one fixed modification time so that
- `docs/decisions/0007-a-container-runtime-for-the-reference-instance.md:96` (technology) — **A Python SDK for the runtime.** It would make the lifecycle code shorter and it costs the rule
- `docs/decisions/0007-a-container-runtime-for-the-reference-instance.md:97` (technology) — that makes `tools/` runnable before an environment exists — standard library only, Python 3.9. The
- `docs/decisions/0007-a-container-runtime-for-the-reference-instance.md:122` (path) — server"* is a rule this project enforces in `tests/conftest.py` rather than promises. The mechanism
- `docs/decisions/README.md:13` (technology) — | [0002](0002-python-and-the-runtime-stack.md) | Python and the runtime stack | Accepted |
- `docs/decisions/README.md:14` (technology) — | [0003](0003-sqlite-as-the-default-store.md) | SQLite as the default store | Accepted |
- `specs/005-item-query-api/spec.md:527` (path) — | OQ-4 | The reference's completion threshold for `Resume` eligibility | **90% ceiling, 5% floor, 300-second minimum runtime — one ordered rule with six branche
- `specs/010-conformance-harness/spec.md:7` (path) — amended: 2026-09-01 at the measurement gate — the spec was written before the thing it compares existed, and four probes moved it. §7's four open questions are 
- `specs/011-subtitle-delivery/spec.md:833` (path) — `tests/conformance/test_progressive_delivery.py`'s. Without both halves the video client's
- `specs/README.md:39` (technology) — - Any technology name — Python, FastAPI, SQLite, a table name, a module name, a function name.
- `specs/README.md:197` (path) — non-administrator restricted to one library, which `tests/fixtures/query.py` has seeded since 005 —
- `specs/README.md:369` (technology) — keys validation errors on the model's Python field — behaviours §1.1's exact failure, in a body
- `specs/README.md:400` (path) — `tests/fixtures/query.py` gives one to a single film and to nothing else — so §3.7's rule, which
- `specs/README.md:430` (path) — decisions about writers. The measured semantics become pure functions in `domain/playstate.py`;
- `specs/README.md:497` (path) — opening `tests/unit/test_compat_query_params.py`, not by re-reading the list, which is 003's
- `specs/README.md:602` (path) — **Five more were run at the 009 spec gate**, on 2026-08-31 — `probe_playlist_move.py` extended and
- `specs/README.md:603` (path) — `probe_playlist_creation.py`, `probe_playlist_expansion.py`, `probe_playlist_visibility.py` and
- `specs/README.md:604` (path) — `probe_playlist_rename.py` written — and it is the gate at which the most claims died at once:
- `specs/README.md:617` (path) — `probe_similar_ranking.py`, `probe_differential_join.py`, `probe_reference_determinism.py` and
- `specs/README.md:618` (path) — `probe_restricted_surface.py`, the last of them the first probe in this repository to measure from

</details>

## Links with nothing to point at

Each was withheld above. A link is left as it was written: retargeting one is an edit to
a specification, and that is this project's decision rather than the export's.

| Target | Cited by |
|---|---|
| `AGENTS.md` | 3 file(s) |
| `docs/README.md` | 1 file(s) |
| `docs/architecture.md` | 3 file(s) |
| `docs/decisions` | 3 file(s) |
| `docs/decisions/0002-python-and-the-runtime-stack.md` | 1 file(s) |
| `docs/decisions/0003-sqlite-as-the-default-store.md` | 1 file(s) |
| `docs/decisions/0006-password-hashing.md` | 1 file(s) |
| `docs/roadmap.md` | 9 file(s) |
| `pyproject.toml` | 1 file(s) |
| `specs/001-server-identity-and-discovery` | 1 file(s) |
| `specs/001-server-identity-and-discovery/plan.md` | 1 file(s) |
| `specs/002-authentication-users-and-sessions` | 1 file(s) |
| `specs/002-authentication-users-and-sessions/plan.md` | 2 file(s) |
| `specs/003-library-configuration-and-scanning` | 1 file(s) |
| `specs/003-library-configuration-and-scanning/plan.md` | 1 file(s) |
| `specs/003-library-configuration-and-scanning/tasks.md` | 2 file(s) |
| `specs/004-metadata-resolution` | 1 file(s) |
| `specs/004-metadata-resolution/plan.md` | 1 file(s) |
| `specs/004-metadata-resolution/tasks.md` | 1 file(s) |
| `specs/005-item-query-api` | 1 file(s) |
| `specs/005-item-query-api/tasks.md` | 2 file(s) |
| `specs/006-images` | 1 file(s) |
| `specs/006-images/plan.md` | 2 file(s) |
| `specs/006-images/tasks.md` | 3 file(s) |
| `specs/007-user-data-and-playstate` | 1 file(s) |
| `specs/007-user-data-and-playstate/tasks.md` | 4 file(s) |
| `specs/008-playback-negotiation-and-delivery` | 1 file(s) |
| `specs/008-playback-negotiation-and-delivery/plan.md` | 2 file(s) |
| `specs/008-playback-negotiation-and-delivery/tasks.md` | 4 file(s) |
| `specs/009-playlists` | 1 file(s) |
| `specs/009-playlists/plan.md` | 1 file(s) |
| `specs/009-playlists/tasks.md` | 2 file(s) |
| `specs/010-conformance-harness` | 1 file(s) |
| `specs/010-conformance-harness/plan.md` | 1 file(s) |
| `specs/010-conformance-harness/tasks.md` | 2 file(s) |
| `specs/011-subtitle-delivery` | 2 file(s) |
| `specs/011-subtitle-delivery/plan.md` | 2 file(s) |
| `specs/011-subtitle-delivery/tasks.md` | 2 file(s) |
| `specs/012-negotiation-inputs` | 1 file(s) |
| `src/atrium/__init__.py` | 1 file(s) |
| `src/atrium/api/delivery.py` | 2 file(s) |
| `src/atrium/api/item_dto.py` | 1 file(s) |
| `src/atrium/api/media_info.py` | 2 file(s) |
| `src/atrium/api/sessions.py` | 1 file(s) |
| `src/atrium/api/subtitles.py` | 1 file(s) |
| `src/atrium/api/system.py` | 1 file(s) |
| `src/atrium/api/universal_audio.py` | 2 file(s) |
| `src/atrium/api/users.py` | 1 file(s) |
| `src/atrium/compat/auth.py` | 2 file(s) |
| `src/atrium/compat/model.py` | 2 file(s) |
| `src/atrium/compat/query_params.py` | 1 file(s) |
| `src/atrium/compat/ranges.py` | 2 file(s) |
| `src/atrium/config/settings.py` | 1 file(s) |
| `src/atrium/domain/queries.py` | 1 file(s) |
| `src/atrium/media/decision.py` | 1 file(s) |
| `src/atrium/media/ffmpeg.py` | 1 file(s) |
| `src/atrium/media/hls.py` | 1 file(s) |
| `src/atrium/media/info.py` | 2 file(s) |
| `src/atrium/media/probe.py` | 1 file(s) |
| `src/atrium/media/sessions.py` | 1 file(s) |
| `src/atrium/media/urls.py` | 1 file(s) |
| `src/atrium/net/address.py` | 1 file(s) |
| `tools/README.md` | 1 file(s) |
