---
feature: 006-images
title: Images
status: Implemented
created: 2026-08-26
updated: 2026-08-28
accepted: 2026-08-28
implemented: 2026-08-28
amended: 2026-08-28 by T12 — AC-2 says when a rescan reads artwork at all; and 2026-08-28 by T9 — §3.4's `Last-Modified` row records the one divergence on a transformed response; and 2026-08-28 by T6 — §3.3's never-upscale rule scoped to the box parameters, the exact `width`/`height` path measured upscaling; and 2026-08-28 by T3 — §3.2's error table: the two `404` bodies named, the index message measured, the empty-GUID edge recorded; and 2026-08-28 by T2 — §3.1's `ParentBackdropItemId` gap closed, and the pairing measured; and 2026-08-28 by T1's probe — §3.2's forgiven-value row, §3.2's parameter table (which still said `fillWidth` crops, three amendments after §3.3 stopped saying it), and the two §3.3 citations the committed script now reproduces; and 2026-08-28 at the spec review — §3.1, §3.2, §3.4, §3.5, AC-12, AC-14, OQ-5, OQ-6; and by the two probes the same day — §3.2 response and errors, §3.3, §3.4 validators, §3.5 discovery, AC-9, OQ-1/2/3/5/6 answered; and 2026-08-28 by the plan — §3.2's error row now names the thirteen-member vocabulary the probe distinguishes, `Box` measured `404`; and 2026-08-28 at the plan gate — §3.3 fill covers rather than crops (AC-6 corrected) and negotiates Accept (AC-15 added), §3.2 response constants and invalid tokens, §3.4 Vary, AC-12, OQ-3's missing cell
depends_on: [002, 004, 005]
---

# 006 — Images

> **This document describes WHAT and WHY only.** No technology names, no storage decisions.

## 1. Purpose

Serve the artwork that 004 located, at the size the client asked for, with cache behaviour that
makes a grid of two hundred posters load once rather than every time.

**Client behaviour unlocked:** a library that looks like anything at all. Posters are most of what
a user sees.

## 2. Scope

**In scope**

- `GET /Items/{itemId}/Images/{imageType}` and its indexed form.
- Which image an item advertises, and how a client knows.
- Resizing, format selection and quality.
- Cache behaviour: tags, conditional requests, and the on-disk cache.
- Delivery of existing chapter images.

**Out of scope**

- Uploading, deleting or reordering images over HTTP.
- Generating images: trickplay tiles, chapter extraction, splashscreens.
- User avatars as a separate route.
- `/Artists/{name}/Images/...` and other by-name image routes — no analysed client uses them, and
  by-name items carry their images through their item id.
- Locating artwork on disk or fetching it from a provider — that is 004.

## 3. Behaviour

### 3.1 How a client discovers an image

Items carry `ImageTags`, a map of image type to tag:

```json
"ImageTags": { "Primary": "ceec6133f1be9a00bb8b07cc59a26c99" },
"BackdropImageTags": [ "b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7" ]
```

`[probe: tools/probe_image_tags.py, Jellyfin 10.11.11, 2026-08-28]`

**Backdrops are a list**, not a map, because an item can have several and they are addressed by
index.

**A tag is a content hash.** It must change when the image changes, and **only** then. This is the
entire cache-invalidation mechanism: clients embed the tag in the URL, so a stable tag means an
image cached forever and a churning tag means every poster re-fetched on every scan. A tag derived
from a timestamp or a row id would look correct and quietly destroy client caching.

An item with no image of a type simply omits it. `ImageTags` is `{}` rather than absent when there
are none, so a client can distinguish "no images" from "field not requested".

Items also carry **inherited** tags so a list can render without a second request per item:
`AlbumPrimaryImageTag`, `SeriesPrimaryImageTag`, `SeriesThumbImageTag`,
`ParentThumbImageTag` with `ParentThumbItemId`, and `ParentBackdropImageTags` with
`ParentBackdropItemId`. An episode without its own artwork is rendered with its season's or
series' — and the client needs both the tag and the id of the item that owns it, which is why
every inherited tag travels with an owning id: `AlbumId`, `SeriesId`, or the explicit
`Parent…ItemId` beside it. On the measured wire `ParentBackdropItemId` pairs with
`ParentBackdropImageTags` row for row
`[probe: tools/probe_item_shapes.py, Jellyfin 10.11.11, 2026-08-27]`, and they are present
exactly together: of 200 sampled episodes, 197 carried both and none carried either alone, each
naming the ancestor whose own `BackdropImageTags` the row repeats
`[probe: manual requests via tools/_probe.py, Jellyfin 10.11.11, 2026-08-28]`. 005 §3.2's matrix
listed only the tags — the gap this feature was to reconcile before AC-14 was provable for
backdrops, **closed at T2**, in 005's own matrix as well as here.

A `ParentLogoImageTag`/`ParentLogoItemId` pair is also on the wire and stays out here
for the same reason it stayed out of 005: no analysed client reads it (Principle VI,
[005 notes](../005-item-query-api/notes/item-shapes.md)). `SeriesThumbImageTag` is carried
**unconfirmed**, exactly as 005 §3.2 records it — never observed across twelve episodes, and
"gated" and "no sampled series had a Thumb" are indistinguishable from outside.

### 3.2 `GET /Items/{itemId}/Images/{imageType}` — `GetItemImage`

**Consumers:** music-client, video-client.

`imageType` is one of `Primary`, `Backdrop`, `Thumb`, `Logo`, `Banner`, `Art`, `Disc`, `Chapter`
— the eight of the reference's thirteen `ImageType` members an item here can ever hold
`[spec: ImageType]`. The indexed form `.../{imageIndex}` addresses one of several.

**Request parameters** v1 honours:

| Parameter | Effect |
|---|---|
| `maxWidth`, `maxHeight` | Fit inside the box, preserving aspect ratio |
| `width`, `height` | Exact dimension |
| `fillWidth`, `fillHeight` | Cover the box, aspect intact, keeping the overflow — §3.3, and **not** a crop |
| `quality` | Compression quality, 0–100 |
| `format` | Requested output format |
| `tag` | The expected content tag; §3.4 |

Declared upstream but **not implemented in v1**: `percentPlayed`, `unplayedCount`, `blur`,
`backgroundColor`, `foregroundLayer`. These composite decorations onto the image server-side. No
analysed client uses them, and a client that sends one receives the undecorated image — a visible
difference, recorded here rather than discovered later.

**A token is accepted here, and none is required.** The reference answers this route `200` with no
token at all, all of 002 §3.1's mechanisms accepted and not one of them demanded
`[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-26]` — the measurement 002
recorded and deferred to this feature
([behaviours §2.10](../../docs/compatibility/behaviours.md#210-the-image-and-delivery-routes-accept-a-token-and-require-none)).
An **invalid** token is not a refusal either: unknown and malformed tokens, sent through the
header, the query and the `MediaBrowser` scheme, each answer the identical `200`
`[probe: manual requests via tools/_probe.py, Jellyfin 10.11.11, 2026-08-28]`.
Atrium does the same, and takes the recorded consequence knowingly: **an item id is a capability**
on this route. The alternative breaks the clients the route exists for — these URLs are handed to
image loaders and external players that set no headers and not always a query either; a server
that starts wanting a token here leaves browsing working and every poster broken. In practice a
well-behaved client still appends `?api_key=` (002 §3.1). There is no per-user visibility branch:
a tokenless request carries no user to filter for, and filtering only the requests that do carry
one would make presenting a token a reason to refuse, which 002 AC-3 forbids.

**Response — 200:** the image bytes, with the real `Content-Type` and a `Content-Length`. The
draft promised `Accept-Ranges: bytes` too; the measured reference sends no such header on an
image response, and a poster is not a seek target
`[probe: tools/probe_image_tags.py, Jellyfin 10.11.11, 2026-08-28]`. Three constant headers ride
every image response, `304`s included: `Content-Disposition: attachment`,
`transferMode.dlna.org: Interactive` and `realTimeInfo.dlna.org: DLNA.ORG_TLAG=*` — plus the
`Vary: Accept` that §3.3's negotiation implies
`[probe: manual requests via tools/_probe.py, Jellyfin 10.11.11, 2026-08-28]`, all under §3.4's
deployment caveat.

**Errors**

| Condition | Status |
|---|---|
| Unknown item | `404` — and nothing else: §3.2's authentication rule leaves no "may not see" branch on this route. Byte-identical to `/Items/{itemId}`'s own refusal for the same id, measured; the **all-zeros identifier** is the one exception and is not reproduced — it is the reference's empty GUID, resolves to a user root folder v1 does not have, and answers `400` in the controller's plain-text shape instead `[probe: manual requests via tools/_probe.py, Jellyfin 10.11.11, 2026-08-28]` |
| Item exists but has no image of that type | `404`, and a **different body**: the message shape of [behaviours §1.11](../../docs/compatibility/behaviours.md#111-there-are-four-error-shapes-not-one), naming the item and the type |
| `imageIndex` out of range | The same `404` and the same message, which names **the type, not the index** — `Backdrop/99` answers "…does not have an image of type Backdrop" `[probe: manual requests via tools/_probe.py, Jellyfin 10.11.11, 2026-08-28]` |
| Unparseable dimension or quality | `400` — measured, against the lenient pattern of behaviours §1.12; a parseable but absurd value (`maxWidth=-100`) is forgiven with `200` instead — and *forgiven* is not *ignored*: the reference re-encodes at the source's own size rather than serving the file, which v1 does not reproduce ([behaviours §1.17](../../docs/compatibility/behaviours.md#117-a-forgiven-dimension-re-encodes-a-bare-quality-does-not)) `[probe: tools/probe_image_formats.py, Jellyfin 10.11.11, 2026-08-28]` |
| `imageType` outside the reference's thirteen-member vocabulary `[spec: ImageType]` | `400` (same probe); a vocabulary member the item merely lacks is the `404` above — `Box`, a member outside §3.2's eight that no item here can ever hold, measured `404` |

### 3.3 Resizing

**Never upscale — asking for a *box*.** A request for a 600px box from a 400px source returns
400px. Upscaling costs CPU and bytes to deliver a blurrier image than the original. The reference
agrees: 3200px asked of an 800px source returns the source
`[probe: tools/probe_image_formats.py, Jellyfin 10.11.11, 2026-08-28]`, and a `fillWidth`/
`fillHeight` box the source cannot cover returns it unchanged too (same probe).

**Asking for a *dimension* is different, and the reference honours it past the source.**
`width`/`height` are exact, up as well as down: `width=4000` of a 2000×3000 source returns
**4000×6000**, `width=2500&height=1000` returns exactly that, and `width=4000&maxWidth=1000`
returns 1000×1500 — the exact size, fitted afterwards by the box parameter
`[probe: tools/probe_image_formats.py, Jellyfin 10.11.11, 2026-08-28]`. So the rule is a
property of *which parameter was sent*, not of the server: `maxWidth`, `maxHeight`, `fillWidth`
and `fillHeight` mean **at most**, and `width`/`height` mean **exactly**. AC-5 is the first
sentence and stays as it is; a client asking for a dimension gets the dimension it named.

**Aspect ratio is preserved** on every path but one: `width` and `height` sent **together** are
honoured exactly even against the source's ratio — 300×300 asked of a 2000×3000 source is a
distorted 300×300. `fillWidth`/`fillHeight` do not crop, whatever this section's draft said: they
scale to **cover** the box with the aspect intact and the overflow kept — 300×600 asked of the
same source returns 400×600, not a 300×600 crop — so the delivered size equals the box only when
the ratios already match, and a box the source cannot cover without upscaling delivers the source
unchanged `[probe: tools/probe_image_formats.py, Jellyfin 10.11.11, 2026-08-28]`. The
earlier probe had measured "exactly the box" on a source that was itself square, where covering
and cropping are indistinguishable — the script now finds a source whose sides differ before it
asks, and reports itself unexercised on a library that has none.

**Format selection**, in order: an explicit `format` if supported; otherwise, **when a transform
runs and the `Accept` header offers `image/webp`, WebP** — the negotiation every browser
triggers, `Vary: Accept` on the response, invisible to the earlier probe because it made the
offer only on a request nothing transformed, and a request served verbatim negotiates nothing;
otherwise the source format — a JPEG poster resizes to JPEG, a PNG logo to PNG, and
`format=Png|Jpg|Webp` are each honoured
`[probe: tools/probe_image_formats.py, Jellyfin 10.11.11, 2026-08-28]`. An explicit `format`
beats the `Accept` offer, and `image/avif` is not negotiated
`[probe: tools/probe_image_formats.py, Jellyfin 10.11.11, 2026-08-28]`. A source with no
transparency may additionally be served as JPEG when that is materially smaller. Transparency is
never discarded **implicitly** — a resized logo keeps its alpha, and a logo silently served as
JPEG would acquire a white box, immediately visible on any dark client theme. An **explicit**
`format=Jpg` wins over that rule, because it does on the measured reference — the transparent
logo comes back opaque (same probe) — and refusing what the client asked for by name would be
the real divergence.

**Resized results are cached on disk**, keyed by item, image type, index, tag and the full
parameter set. The cache is disposable: deleting it costs CPU, never correctness. It lives outside
the library root (004 §2).

### 3.4 Caching and conditional requests

The contract with the client:

| Mechanism | Behaviour |
|---|---|
| `Last-Modified` / `If-Modified-Since` | The validator pair the reference actually serves: every image response carries `Last-Modified`, and `If-Modified-Since` at that date answers `304` with no body `[probe: tools/probe_image_tags.py, Jellyfin 10.11.11, 2026-08-28]`. **The value on a *transformed* response is the one thing here v1 does not reproduce**: the reference sends the variant's own creation time — measured one second *after* that response's own `Date` — and Atrium sends the carrier's, on every path. Both are valid validators for the same entity; Atrium's survives a cache wipe, where the reference's forces a client to re-download every poster `[probe: manual requests via tools/_probe.py, Jellyfin 10.11.11, 2026-08-28]` |
| `Cache-Control` | `public` without a `tag` on the URL; `public, max-age=31536000` with one (same probe) — only the tag makes the URL immutable |
| `ETag` / `If-None-Match` | **Not sent by the measured reference.** The draft promised an etag on every response; no analysed client needs one where the tag-in-URL mechanism exists, so v1 mirrors the reference rather than inventing a second validator (same probe) |
| `Vary: Accept` | On every image response, `304` included — the consequence of §3.3's WebP negotiation `[probe: manual requests via tools/_probe.py, Jellyfin 10.11.11, 2026-08-28]` |

One caveat the probe prints itself: the deployment measured answers through a caching
intermediary (an `Age` header), so the header rows describe that deployment; the
status-and-bytes findings are unaffected.

**The `tag` parameter makes a URL immutable.** A request carrying the tag it expects can be cached
indefinitely, because the URL changes when the image does. A request without one is still correct
but cannot be cached as aggressively.

**A stale `tag` is not an error.** A client asking for an image by a tag the item no longer has
receives the *current* image with a `200` — not a `404`. The client is behind, not wrong, and
failing here empties a user's grid during a refresh. Measured: the reference answers a stale tag
`200`, byte-identical to the untagged request
`[probe: tools/probe_image_tags.py, Jellyfin 10.11.11, 2026-08-28]` — AC-10 is a reproduction,
not a divergence.

### 3.5 Chapter images

`imageType` of `Chapter` with an index addresses a chapter thumbnail, for the scrubbing UI of a
video client.

v1 **routes** chapter thumbnails, and nothing in v1 can yet put one on disk — so today every
chapter request answers the absent-image `404`, an accepted gap recorded in
[behaviours §5.8](../../docs/compatibility/behaviours.md#58-a-chapter-image-can-never-be-served-in-v1).
It does not **extract** them: generating them
means decoding a video at intervals and running a background job over the whole library, and that
job — trickplay and chapter-image generation — is out of v1 in its own right
([roadmap](../../docs/roadmap.md#later-unscheduled),
[api-surface §10](../../docs/compatibility/api-surface-v1.md#10-deliberately-excluded-from-v1)).
The arrival of transcoding does not change
this; a decode on demand for one client is not a sweep over every item. An item whose chapters have
no images answers `404` per chapter, which is what a client already handles for a server that has
not finished generating them.

A client learns that a chapter *has* an image from the `Chapters` field (gated, 005 §3.2): each
entry carries an `ImageTag` when its image exists — 1,311 of 1,354 measured entries did — and
`GET .../Images/Chapter/{index}` serves the corresponding thumbnail
`[probe: tools/probe_image_tags.py, Jellyfin 10.11.11, 2026-08-28]`. An entry without the tag is
a chapter whose image was never generated, and answers `404` as above.

## 4. Data the feature owns

| State | Observable as | Lifetime |
|---|---|---|
| Image tags | `ImageTags` and friends in 005's responses | Until the underlying image changes |
| Resized-image cache | Response latency only | Disposable at any moment |

The images themselves belong to the user, on disk, and are owned by 004.

## 5. Acceptance criteria

1. An item with a poster advertises a `Primary` tag; one without advertises `ImageTags: {}`.
2. The tag is unchanged across a rescan when the image file is unchanged, and changes when the file
   changes — *when the scan reads the directory at all*: a default scan re-examines an item only
   when its **media** file changed, so replacing a poster beside an untouched film is picked up by
   a deep scan and not before
   ([behaviours §5.6](../../docs/compatibility/behaviours.md#56-a-default-rescan-does-not-notice-a-replaced-poster)).
3. `GET .../Images/Primary` returns the bytes with the right `Content-Type` and a `Content-Length`.
4. `maxWidth=300` on a 1000px-wide source returns a 300px-wide image with the aspect ratio
   preserved.
5. `maxWidth=2000` on a 400px-wide source returns 400px — no upscaling.
6. `fillWidth`/`fillHeight` scale to cover the box with the aspect ratio intact and the overflow
   kept — 300×600 of a 2000×3000 source is 400×600, never a crop — and a box the source cannot
   cover without upscaling returns the source unchanged.
7. An image with transparency is never served in a format that discards it.
8. A second identical request is served from cache and is byte-identical to the first.
9. `If-Modified-Since` at the `Last-Modified` the server sent answers `304` with an empty body —
   the validator pair measured on the reference, which sends no image etag (§3.4).
10. A request carrying a **stale** `tag` answers `200` with the current image, not `404`.
11. An unknown item, an item with no such image, and an out-of-range index all answer `404`; an
    unparseable dimension or quality, and an `imageType` outside the reference's thirteen-member
    vocabulary, are the validation `400` — a vocabulary member the item merely lacks is the
    `404` above. *(The `400` rows were folded in at the 2026-08-28 audit — M44: their tests were
    already filed under this criterion, asserting more than it said.)*
12. A request with no token answers `200`, and every token mechanism — `?api_key=` included, an
    unknown or malformed token included — is accepted without changing the answer (shared with
    002 AC-3).
13. Deleting the entire resize cache changes no response body.
14. An episode carries its series' `Primary` tag **and** the id of the item that owns it, whether
    or not the episode has artwork of its own — inheritance is unconditional on the wire, and
    falling back to it is the client's decision, not the server's.
15. A resized response is WebP when the request's `Accept` offers `image/webp`, and carries
    `Vary: Accept`; an explicit `format` overrides the offer, and a request served verbatim
    ignores it.
16. Asking for a **dimension** is exact, up as well as down: `width` past the source upscales —
    `width=4000` of a 2000×3000 source is 4000×6000 — and `width` with `height` together are
    honoured even against the source's ratio, a deliberate 300×300 distortion included (§3.3).
    *(Added at the 2026-08-28 audit — M42: the spec itself conceded AC-5 stopped at the box
    parameters.)*
17. Three constant headers ride every image response — `Content-Disposition: attachment`,
    `transferMode.dlna.org: Interactive`, `realTimeInfo.dlna.org: DLNA.ORG_TLAG=*` — and
    `Cache-Control` is `public` bare on an untagged URL, `public, max-age=31536000` with a tag
    (§3.2, §3.4). *(Added at the same audit — M43.)*

## 6. Conformance

| Endpoint / behaviour | Level | How it is proven |
|---|---|---|
| `GET /Items/{itemId}/Images/{imageType}` | **L2** | Golden headers and byte-length assertions against fixture images |
| Indexed form | **L2** | Fixture with three backdrops |
| Resize matrix | **L2** | Table-driven over the parameter combinations of §3.2, the `Accept` offer included |
| Cache and conditional requests | **L2** | Request pairs asserting `304` and cache-hit identity |
| Tag stability | **L2** | Rescan comparison (AC-2) |

Golden files here assert **headers and dimensions**, not image bytes: encoder output is not stable
across library versions, and a test that breaks when an encoder is upgraded teaches nothing.

## 7. Open questions

Five of the six were measured at the spec review, on 2026-08-28, and the answers are folded into
the sections they blocked:

| # | Question | Answer | Measured by |
|---|---|---|---|
| OQ-1 | How the reference derives its image tags — content hash or something weaker | **Weaker**: 0 of 12 tags reproduce as MD5 of the image bytes in either GUID spelling, and every tag is stable across requests. Blocks nothing — change-when-changed is all the contract needs, and a content hash delivers it | `tools/probe_image_tags.py`, 2026-08-28 |
| OQ-2 | Does the reference `404` or serve the current image for a stale `tag`? §3.4 assumes serve | **Serves the current image**, `200` and byte-identical to the untagged request — AC-10 is a reproduction | `tools/probe_image_tags.py`, 2026-08-28 |
| OQ-3 | The reference's format-selection rule, especially transparency handling | **Source format survives a resize**; `format=Png\|Jpg\|Webp` each honoured — `Jpg` on a transparent logo included, which comes back opaque; alpha survives every implicit path. §3.3 rewritten from this — and the plan gate added the cell this probe was blind to: a **resized** response negotiates `Accept: image/webp` (§3.3, AC-15) | `tools/probe_image_formats.py`, 2026-08-28; manual requests, 2026-08-28 |
| OQ-4 | Does any client send `percentPlayed`, `blur` or `foregroundLayer`? | Open — the declared gap in §3.2 stands until the differential harness says otherwise | Differential harness (010) |
| OQ-5 | Whether an unparseable dimension is refused with `400` or ignored, and what an `imageType` outside §3.2's set answers | **`400` for both** — the one measured error path that is not lenient; a parseable but absurd `maxWidth=-100` is forgiven with `200` | `tools/probe_image_formats.py`, 2026-08-28 |
| OQ-6 | How chapters advertise their images — an image tag per `Chapters` entry, or something else | **`ImageTag` per `Chapters` entry** — 1,311 of 1,354 measured entries carried one — served by the indexed `Chapter` route | `tools/probe_image_tags.py`, 2026-08-28 |

## 8. References

- [docs/compatibility/api-surface-v1.md §7](../../docs/compatibility/api-surface-v1.md#7-images)
- [behaviours §2.10](../../docs/compatibility/behaviours.md#210-the-image-and-delivery-routes-accept-a-token-and-require-none) — the measurement behind §3.2's authentication rule
- [specs/004 §3.4](../004-metadata-resolution/spec.md) — which file becomes which image
- [specs/005 §3.2](../005-item-query-api/spec.md) — where tags are advertised
- [specs/005 notes/item-shapes.md](../005-item-query-api/notes/item-shapes.md) — the measured inherited-tag pairs of §3.1
- `[spec: GetItemImage, GetItemImageByIndex, ImageType]`
