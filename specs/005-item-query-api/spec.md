---
feature: 005-item-query-api
title: Item query API
status: Implemented
created: 2026-08-26
updated: 2026-09-01
accepted: 2026-08-27
implemented: 2026-08-28
amended: 2026-08-29 by 008 T3 — §3.2 gains `Container`, `HasSubtitles` and `VideoType` per type and `IsHD` gated, four properties T1 measured and declined under Principle VI and that 007's owed list and 008 AC-28 have since given readers; and 2026-08-28 by 006 T2 - section 3.2 gains `ParentBackdropItemId`, the pair 005 measured and emitted only half of; by T9 - section 3.2; by T10 - section 3.3; by T11 - section 3.7; by T12 - sections 3.8 and 5 (AC-11 reversed); by T13 - section 3.7 (NextUp measured); by T14 - sections 3.9 and 5 (AC-13 restated); by T15 - sections 3.10 and 5 (AC-14 restated); and 2026-09-01 at 010's measurement gate, by the two decisions that gate left to this feature - section 3.7's `Similar` hedge becomes a measurement and two recorded divergences (behaviours section 3.23 and section 3.24), AC-12 gains the exact-`limit` clause and names both, and OQ-5's `Similar` half moves to Resolved with both decisions made rather than owed
depends_on: [002, 004]
---

# 005 — Item query API

> **This document describes WHAT and WHY only.** No technology names, no storage decisions.

## 1. Purpose

Let a client browse and search the library: list items, filter and sort them, page through them,
and fetch one in full.

This is the largest feature in v1 — seventeen endpoints, and one of them (`/Items`) is the endpoint
every client uses for almost everything. Getting the response shape right here decides whether
anything a client displays is correct.

**Client behaviour unlocked:** every screen except playback.

## 2. Scope

**In scope**

- `GET /Items`, `GET /Items/{itemId}`, `GET /UserViews`.
- `GET /Items/Latest`, `GET /Items/Filters`, `GET /Items/{itemId}/Similar`,
  `GET /Items/{itemId}/InstantMix`, `GET /UserItems/Resume`.
- `GET /Shows/{seriesId}/Seasons`, `GET /Shows/{seriesId}/Episodes`, `GET /Shows/NextUp`.
- `GET /Artists`, `GET /Artists/AlbumArtists`, `GET /Genres`, `GET /MusicGenres`, `GET /Years`.
- `GET /Search/Hints`.
- The item representation, its optional fields, and the list envelope.

**Out of scope**

- Writing: no item creation, update or deletion here. Playlist mutation is 009; deletion of a
  playlist goes through 009's use of `DELETE /Items/{itemId}`.
- User-data mutation — 007.
- Image bytes — 006. This feature emits image *tags*; 006 serves what they identify.
- Media sources and streams beyond echoing what 008 produced.
- Collections/box sets, Live TV, channels, `/Studios`, `/Persons` as endpoints.

## 3. Behaviour

### 3.1 The list envelope

Every list endpoint except `/Items/Latest` and `/Items/Filters` returns:

```json
{ "Items": [ ], "TotalRecordCount": 0, "StartIndex": 0 }
```

`StartIndex` is present — the reference includes it where Emby does not.

**`GET /Items/Latest` returns a bare array**, not this envelope. A client decoding it as an
envelope gets nothing, so this asymmetry is load-bearing rather than cosmetic.

**Measured, not assumed.** All twelve list endpoints of the v1 surface were probed against a live
reference: ten return the envelope with `StartIndex` present, `/Items/Latest` returns a bare
array, `/Items/Filters` returns `{Genres, Tags, OfficialRatings, Years}`, and `/Search/Hints`
returns `{SearchHints, TotalRecordCount}` — four shapes, and Atrium reproduces each per endpoint
rather than normalising them. `[probe: tools/probe_query_envelope.py, Jellyfin 10.11.11, 2026-08-26]`

**Paging** is `StartIndex` and `Limit` on the request. `TotalRecordCount` is the count *before*
paging, so a client can size a scrollbar.

### 3.2 The item representation

**There is not one representation. There are three**, and which one a client gets depends on the
route it asked, not on the item type.
`[probe: tools/probe_item_shapes.py, Jellyfin 10.11.11, 2026-08-27]`

| Route | What comes back |
|---|---|
| A list row | The narrow shape below: always-present, plus what the type has, plus whatever `Fields` asked for |
| A single item by id | **Everything**, unasked. `Fields` has nothing left to add |
| The user's views | A third shape: as wide as a full body, unasked, for library roots |

Items are discriminated by `Type` (`Movie`, `Series`, `Season`, `Episode`, `MusicArtist`,
`MusicAlbum`, `Audio`, `Playlist`, `CollectionFolder`, `UserView`, `Folder`).

**Always present**, on every item, in every list:

| Field | Notes |
|---|---|
| `Id`, `ServerId`, `Name`, `Type` | |
| `MediaType` | `Video`, `Audio` or `Unknown` |
| `IsFolder` | **Absent on a by-name row** — a genre, a music genre or a year carries no `IsFolder`, list row and full body alike `[probe: manual requests via tools/_probe.py, Jellyfin 10.11.11, 2026-08-28]` |
| `LocationType` | `FileSystem` for everything v1 serves |
| `ChannelId` | **Always, and always `null`** — the one property that survives the null-omission of [behaviours §1.7](../../docs/compatibility/behaviours.md#17-a-null-property-is-absent-everywhere-by-one-setting) |
| `UserData` | **Always**, with no `Fields` or `EnableUserData` needed. Jellyfin's version carries `Key` and `ItemId` inside it `[probe: tools/probe_item_shapes.py, Jellyfin 10.11.11, 2026-08-27]`. On a container, `Played` and `UnplayedItemCount` describe the **subtree**: a series is played exactly when nothing beneath it is left unplayed, and the remainder rides every bare container row `[probe: manual requests via tools/_probe.py, Jellyfin 10.11.11, 2026-08-28]` |
| `ImageTags` | Empty object when the item has no images |
| `ImageBlurHashes` | `ImageTags`' shape again, a BlurHash per image id. A client rendering placeholders reads it. Atrium sends the empty object — an accepted gap, argued in [behaviours §5.5](../../docs/compatibility/behaviours.md#55-no-blurhash-is-computed-so-imageblurhashes-is-always-empty) |
| `BackdropImageTags` | |

**Present in a list row when the item type has them** — the measured matrix, one row per field.
*The draft grouped these into type families and the measurement disagreed row by row: a `Series`
list row carries no `ChildCount` (it is gated) and no `IndexNumber`, a `Season` carries the
series context, and an album carries the artist lists. Rewritten at T9 to say exactly what the
registry holds.* `[probe: tools/probe_item_shapes.py, Jellyfin 10.11.11, 2026-08-27]`

| Field | Types |
|---|---|
| `ProductionYear`, `PremiereDate` | `Movie`, `Series`, `Season`, `Episode`, `MusicAlbum`, `Audio` |
| `RunTimeTicks` | `Movie`, `Series`, `Episode`, `MusicArtist`, `MusicAlbum`, `Audio` |
| `OfficialRating` | `Movie`, `Series` |
| `CommunityRating` | `Movie`, `Series`, `Episode` |
| `IndexNumber` | `Season`, `Episode`, `Audio` (track) |
| `ParentIndexNumber` | `Episode`, `Audio` (disc) |
| `SeriesId`, `SeriesName`, `SeriesPrimaryImageTag` | `Season`, `Episode` |
| `SeasonId` | `Episode` |
| `SeriesThumbImageTag` | `Episode` — unconfirmed, see below |
| `ParentThumbItemId`, `ParentThumbImageTag` | `Season`, `Episode` |
| `ParentBackdropItemId`, `ParentBackdropImageTags` | `Season`, `Episode`, `MusicAlbum`, `Audio` |
| `Album`, `AlbumId`, `AlbumPrimaryImageTag` | `Audio` |
| `AlbumArtist`, `AlbumArtists`, `Artists`, `ArtistItems` | `MusicAlbum`, `Audio` |
| `CollectionType` | Library roots |
| `PlaylistItemId` | Playlist entries — see 009 |
| `Container` | `Movie`, `Episode`, `Audio` — added by 008 |
| `HasSubtitles`, `VideoType` | `Movie`, `Episode` — added by 008 |

**Only when a list row asks for them:** `MediaSources`, `MediaStreams`, `Path`, `Etag`,
`Chapters`, `DateCreated`, `DateLastMediaAdded`, `ProviderIds`, `Tags`, `Taglines`, `ExternalUrls`,
`OriginalTitle`, `ParentId`, `CumulativeRunTimeTicks`, `RecursiveItemCount`, `ChildCount`,
**`SortName`, `Overview`, `Genres`, `GenreItems`, `Studios`, `People`,
`PrimaryImageAspectRatio`**, `Width`, `Height`, `IsHD`. `[spec: ItemFields]`

> **Four properties arrived after this table was written**, and the note beside "where this field
> set comes from" is why they need a line rather than a silent edit. `Container`, `VideoType`,
> `HasSubtitles` and `IsHD` were all measured on the wire at T1 and all deliberately left out: no
> analysed client read them, and Principle VI applies to fields. What changed is that a client now
> does — [007's `NowPlayingItem` differential](../007-user-data-and-playstate/tasks.md#what-this-feature-owes-the-next-ones)
> names three of them among the nine media-derived properties a session entry is missing, and
> [008 AC-28](../008-playback-negotiation-and-delivery/spec.md) turns `Container` into the
> observable half of a rule about media sources. They are in the tiers the measurement put them in
> `[probe: tools/probe_item_shapes.py, Jellyfin 10.11.11, 2026-08-27]`: three per-type, one gated,
> and the two conditional ones (`HasSubtitles`, `IsHD`) emitted only where they are true, which is
> the reference's own shape `[source: Emby.Server.Implementations/Dto/DtoService.cs:1107-1110,1316-1323 @ v10.11.11]`.

> **`SeriesThumbImageTag` was not observed at all** — not bare, not asked for, not in a full body,
> across twelve episodes. Whether it is gated, or simply absent because none of those episodes'
> series carries a Thumb, cannot be told apart from outside: a null property is omitted. It stays
> in the table, unconfirmed, rather than being deleted on one library's evidence.

> **The seven in bold were in the per-type group until they were measured**, and six of them are
> requestable tokens — which is what made the row wrong, because a token is gated by definition. A
> list of movies carries no overview, no genres, no cast and no aspect ratio unless the client
> asks. `PrimaryImageAspectRatio` is the one with a dependency behind it: 004 supplies the width
> and height it is computed from, and a list row still does not carry it.
> `[probe: tools/probe_item_shapes.py, Jellyfin 10.11.11, 2026-08-27]`

**Where this field set comes from.** It is the union of what the two analysed clients actually read
from a response, not the reference's full representation, which has over 150 properties. A field no
client reads is not observable, and Principle VI applies to fields as it does to endpoints.
Emitting the full set would mean specifying and testing a hundred fields whose correctness nobody
can check.

**The single-item route is the exception to that**, and it is not a choice: the reference emits
everything there whatever the request says, so a narrow answer is observable as a missing field
rather than as a smaller one. What v1 emits from it is the union above, and the differential
harness is what closes the rest.

> ⚠️ **This is a known, bounded delta.** A different client reading a field Atrium omits sees it as
> absent. The mitigation is measurement, not guessing: the differential harness (010) reports every
> field the reference sends that Atrium does not, and each gets added or explicitly declined.

### 3.3 `GET /Items` — `GetItems`

**Consumers:** music-client, video-client. The endpoint everything else is built on.

The reference declares **86 query parameters**. `[spec: GetItems]` v1 implements them in two tiers,
and refuses to pretend about the third.

**Tier 1 — observed in use, required.**

| Parameter | Effect |
|---|---|
| `userId` | Whose user data and visibility apply |
| `parentId` | Children of one container |
| `recursive` | Descend the whole subtree rather than one level |
| `startIndex`, `limit` | Paging |
| `sortBy`, `sortOrder` | Ordering; §3.4 |
| `fields` | Optional fields from §3.2 |
| `includeItemTypes` | Restrict by `Type` |
| `searchTerm` | Name match |
| `ids` | Fetch a specific set |
| `albumArtistIds` | Music navigation |
| `filters` | `IsFavorite`, `IsPlayed`, `IsUnplayed`, `IsResumable` |
| `isPlayed` | Played state |

**Tier 2 — implemented by design**, because they are cheap, they are the obvious next thing a
client asks for, and a filter that silently does nothing is worse than one that does not exist:
`excludeItemTypes`, `excludeItemIds`, `mediaTypes`, `isFavorite`, `genres`, `genreIds`, `years`,
`studioIds`, `artistIds`, `albumIds`, `personIds`, `nameStartsWith`, `nameStartsWithOrGreater`,
`nameLessThan`, `minCommunityRating`, `enableUserData`, `enableImages`, `imageTypeLimit`,
`enableImageTypes`, `enableTotalRecordCount`.

**Tier 3 — not implemented in v1.** The remaining parameters, covering Live TV, 3D, parental
ratings, adjacency, resolution bounds, air dates and provider-id presence.

**What happens when a Tier 3 parameter arrives** is a real decision, not an oversight, and neither
available answer is free:

- *Ignore it silently* — the client receives more items than it asked for and shows the user wrong
  results, with nothing to indicate why.
- *Reject with `400`* — the reference answers `200`, so this is itself a delta, and it breaks a
  client that sends a parameter harmlessly.

**v1 ignores the parameter and records it.** Rejecting turns a partial answer into no answer, which
is worse for the client and further from the reference. Every ignored Tier 3 parameter is counted
and reported, so the set that real clients actually send becomes measurable — and anything that
shows up gets promoted to Tier 2. This is the one place in v1 where a delta is accepted knowingly,
with a mechanism attached for closing it.

**Errors:** `401` unauthenticated. `parentId` naming an unknown or invisible item is `404`. An
unrecognised **token** in an enum-valued parameter is ignored, never rejected — the filter simply
drops, measured across `includeItemTypes`, `sortBy`, `fields` and `filters`
(behaviours §1.12) `[probe: tools/probe_query_envelope.py, Jellyfin 10.11.11, 2026-08-28]`. A value that cannot
parse as its declared **type** — `limit=abc`, a malformed identifier — is `400`, in the
problem-details shape of behaviours §1.11 `[probe: tools/probe_query_envelope.py, Jellyfin 10.11.11, 2026-08-28]`.
This spec previously claimed the first case was a `400`; it is not, and treating it as one would
have refused requests the reference serves.

One consequence of the drop rule is worth stating, because getting it wrong inverts an answer
*(added by T10)*: a type token the reference's vocabulary **does** contain but this version cannot
produce — `Playlist` before 009, `BoxSet` — is a filter that **holds and matches nothing**, not an
unrecognised token to drop. Dropping it would answer the whole library to a client that asked for
playlists; the reference, which has such items, filters by them. `[spec: BaseItemKind]`

### 3.4 Sorting

`sortBy` accepts `SortName`, `DateCreated`, `PremiereDate`, `PlayCount`, `DatePlayed`, `Random`,
`AlbumArtist`, `Artist`, and combinations, with `sortOrder` of `Ascending` or `Descending`.
`[prior-probe: Jellyfin 10.11.11, 2026-06-13]`

Three rules make ordering reproducible:

1. **`SortName` uses the normalisation from 003 §3.7** — articles, diacritics, case, numeric
   prefixes. This is where ordering parity is won or lost, because it affects every list.
2. **Ordering is total.** After the requested keys — and after the name, which the reference
   chains behind a `SortName` ordering — ties break on the item id, so paging cannot show an item
   twice or skip one. **The reference's ordering is not total, and what that costs is measured**:
   its movie sorts arrive stable with ties in ascending id order, but under `AlbumArtist` and
   `Artist` the concatenation of a query's pages is **not** the one-shot list — a client paging a
   large audio library sees some items twice and never sees others
   `[probe: tools/probe_sort_stability.py, Jellyfin 10.11.11, 2026-08-27]`. Totality is therefore
   a deliberate divergence from a defect, argued in
   [behaviours §3.6](../../docs/compatibility/behaviours.md#36-ties-are-engine-resolved-and-paging-the-artist-sorts-loses-rows--class-b-diverged):
   within any tie Atrium's order is one the reference could have produced, and on the movie sorts
   it is the very order the measured server does produce.
3. **`Random` is seeded per request** and the seed is not exposed, so paging through a random
   ordering is not meaningful. Observably the reference's behaviour: a fresh shuffle on every
   request — two identical 97-row requests shared 4 items
   `[probe: tools/probe_sort_stability.py, Jellyfin 10.11.11, 2026-08-27]` — and clients use it
   for a single page.

Two further ordering behaviours, read from the source and reproduced:

- **An item with no `PremiereDate` sorts by January 1 of its `ProductionYear`** under
  `sortBy=PremiereDate`, rather than clumping with the dateless
  `[source: Jellyfin.Server.Implementations/Item/OrderMapper.cs:49 @ v10.11.11]`.
- **A request carrying `searchTerm` is ordered by match quality first** — exact match, then
  prefix at a word boundary, then prefix, then contains — ahead of whatever `sortBy` asked for
  `[source: Jellyfin.Server.Implementations/Item/BaseItemRepository.cs:1604-1611 @ v10.11.11]`
  `[source: Jellyfin.Server.Implementations/Item/OrderMapper.cs:76-93 @ v10.11.11]`.

### 3.5 `GET /Items/{itemId}` — `GetItem`

**Consumers:** music-client, video-client.

One item in full. **This is the Jellyfin route**; the Emby dialect's
`/Users/{userId}/Items/{itemId}` was removed in 10.11 and Atrium does not serve it
([ADR-0004](../../docs/decisions/0004-pin-to-jellyfin-10-11.md)).

`404` for unknown or invisible items — the same answer for both, so the endpoint does not disclose
the existence of items a user may not see. The `404` carries the problem-details shape of
behaviours §1.11, as does the `400` for an identifier that does not parse at all: which of the two
a caller gets depends only on whether the id is *shaped* like an id
`[probe: tools/probe_query_envelope.py, Jellyfin 10.11.11, 2026-08-28]`.

### 3.6 `GET /UserViews` — `GetUserViews`

The libraries as this user sees them, after policy (002 §3.5). Each carries `CollectionType`, which
tells the client which navigation to offer.

A user with no permitted libraries gets an empty envelope, not an error.

### 3.7 Discovery endpoints

| Endpoint | Returns | Rule that matters |
|---|---|---|
| `GET /Items/Latest` | **Bare array** of recently added items, grouped upward | Honours `LatestItemsExcludes` and `HidePlayedInLatest` from the stored configuration (002 §3.6); grouping rule below |
| `GET /UserItems/Resume` | Items with a resume position | Ordered most-recently-played first; excludes items played past the completion threshold (007 §3.7) |
| `GET /Shows/NextUp` | The next unwatched episode per series | One item per series, never several; "next" measured below |
| `GET /Items/{itemId}/Similar` | Related items | v1 scores on shared genres, people and studios. Deterministic |
| `GET /Items/{itemId}/InstantMix` | A radio-style queue from a seed | Deterministic for a given seed and library |
| `GET /Items/Filters` | `{Genres, Tags, OfficialRatings, Years}` for a parent | `[spec: QueryFiltersLegacy]` |

**NextUp's chain, measured** *(provenance added by T13 — the claim below predates its probe)*:
"next" is the first unplayed episode in `(season, episode)` order after the
**highest-numbered** played one — playing an early episode again moves nothing, which the probe
discriminated directly by marking E02 and then E01, in that order of time, and reading E03 back
both times. One row per series; the most recently played series first
`[probe: tools/probe_next_up.py, Jellyfin 10.11.11, 2026-08-28]`. The **specials exclusion is
measured since 2026-09-02** and §7 OQ-7 is closed: it was unmeasurable while no reachable library
held a series whose only unplayed episodes were season 0's — the probe said so in its own output —
so one was built. With every ordinary episode played and season 0 pristine, `GET /Shows/NextUp`
offers **nothing on either server**, which is this rule holding on both
`[probe: tools/differential.py --named next-up-pristine-specials-season, Jellyfin 10.11.11,
2026-09-02]`. It was a named comparison of
[010 §3.10](../010-conformance-harness/spec.md), run against the single-use reference instance
rather than against a library that happens to have one.

**The Latest grouping rule, measured** *(added by T11 — the plan's first wording said an episode
always surfaces as its series, and one response disproved it)*: recent items group under their
container — an episode under its **series**, a track under its album, a film under itself — and a
group surfaces as **the container only when it holds more than one recent item; a group of one
surfaces as the item itself**. One measured response carried a `Series` beside a lone `Episode`
and a lone `Audio` beside grouped `MusicAlbum`s, newest first, each group once
`[probe: manual requests via tools/_probe.py, Jellyfin 10.11.11, 2026-08-28]`. Both steering keys
were measured by name on a live configuration: `LatestItemsExcludes` (view identifiers; applied
to the unscoped request) and `HidePlayedInLatest`, `true` on a configuration never edited, which
keeps played items out unless the caller's own played filter asks for them.

**`Similar` is deterministic on purpose, and that is a recorded divergence** *(corrected at 010's
measurement gate, 2026-09-01 — this paragraph said "the reference's are not obviously so", which
was a hedge in a place where a measurement was available)*. **The reference does not rank `Similar`
at all**: it filters on the seed's own genres and tags and orders what matches at random, so four
identical requests returned 48 distinct items with **none** in common. v1 scores instead — shared
genres, people and studios — because a non-deterministic endpoint cannot be tested at L2 or
compared at L3 at all, which is 010's problem as much as this feature's. The argument is not that a
client cannot see the difference; it is that nothing can be built on a random draw, and every
answer Atrium sends here is one the reference could have drawn. Recorded, with the full argument,
in [behaviours §3.23](../../docs/compatibility/behaviours.md).

**`limit` is a maximum here, and that is the second divergence.** The reference answers `limit + 4`
rows on a **movie** seed — measured at 1, 5 and 20, on two seeds — where a series, an album and an
artist seed answer exactly `limit`; and it fills `TotalRecordCount` with the number of rows it
returned. Atrium answers exactly `limit` on every seed type, with `TotalRecordCount` the pre-limit
pool size that AC-5 requires of every list endpoint in this feature. Recorded in
[behaviours §3.24](../../docs/compatibility/behaviours.md).
`[probe: tools/probe_similar_ranking.py, Jellyfin 10.11.11, 2026-09-01]`

**`InstantMix` is deterministic for the same reason, and the reference's own behaviour is still
unmeasured** — §7 OQ-5 holds that half open, and no divergence is claimed for it until it is.

### 3.8 Series navigation

`GET /Shows/{seriesId}/Seasons` and `GET /Shows/{seriesId}/Episodes`, both returning the envelope.

Episodes accept a `seasonId` to scope them, and both honour `DisplayMissingEpisodes` from the
user's configuration.

**Season 0 is "Specials" and it sorts first, in plain index order** *(corrected by T12's
measurement — this section claimed the opposite, "every client expects it last", with nothing
behind it)*. A live series with a specials season answers `[Specials, Season 1]`
`[probe: manual requests via tools/_probe.py, Jellyfin 10.11.11, 2026-08-28]`, and Principle I
settles which order Atrium sends: the reference's. Where a client *shows* its user the specials
is the client's own re-sort, invisible to this API.

### 3.9 By-name endpoints

`GET /Artists`, `GET /Artists/AlbumArtists`, `GET /Genres`, `GET /MusicGenres`, `GET /Years`.

All return the envelope, and all take `parentId`, `userId`, `startIndex`, `limit`, `sortBy`,
`sortOrder`, `searchTerm`.

> **The `TotalRecordCount` divergence.** The reference disables counting on these endpoints when
> the request carries no `limit`, returning `TotalRecordCount: 0` beside a non-empty `Items`.
> `[probe: tools/probe_by_name_counts.py, Jellyfin 10.11.11, 2026-08-28; upstream jellyfin/jellyfin#17541]`
>
> **Atrium always returns the true count.** Argued and recorded in
> [behaviours §3.1](../../docs/compatibility/behaviours.md#31-totalrecordcount-is-0-on-by-name-endpoints-without-limit--class-b):
> no client can observe the difference in a way that changes its behaviour, and the upstream fix is
> approved.

`/Artists` versus `/Artists/AlbumArtists` is the distinction from 003 §3.5: every credited artist
versus only those credited on an album. A compilation-heavy library makes the difference large on
the reference, and a client offering "Artists" and "Album Artists" as separate views needs both
to be right.

> **In v1 the two routes coincide as row sets, and that is a recorded consequence, not a bug**
> *(added by T14)*. An artist here is a per-library item created per *album artist*
> ([behaviours §5.3](../../docs/compatibility/behaviours.md#53-an-artist-in-two-music-libraries-is-two-rows)),
> so an artist who only ever performs has a name on every track and **no row to list** — the
> reference, whose artists are by-name rows, lists them. The credit distinction still bites at
> item level, `artistIds` versus `albumArtistIds`, where T6 measured it. AC-13 states the v1
> truth below. Two more measured route quirks, reproduced: `/Genres` and `/MusicGenres` rows
> carry no `UserData`, and the two artist routes no `IsFolder`, where the same rows through
> `/Items` carry both `[probe: manual requests via tools/_probe.py, Jellyfin 10.11.11,
> 2026-08-28]`.

### 3.10 `GET /Search/Hints` — `GetSearchHints`

Included by design: the music-client searches through `/Items?searchTerm=`, but `/Search/Hints` is
what most clients use and its response shape is **not** the item envelope.

Returns `{"SearchHints": [...], "TotalRecordCount": n}`, where a hint is a flattened summary —
`ItemId`, `Id`, `Name`, `Type`, `MediaType`, the image tag pairs (primary, thumb and backdrop,
each resolved through the item's ancestors — a track's hint carries its album's cover), and
type-specific extras like `Series`, `Album`, `AlbumId`, `AlbumArtist`. **`Artists` travels on
every hint, empty list included, and `ChannelId` is the same explicit null every item carries.**
`[spec: SearchHint]` `[probe: manual requests via tools/_probe.py, Jellyfin 10.11.11, 2026-08-28]`

**Matching is case- and diacritic-insensitive, against the name** — *(corrected by T15: this
sentence said "against name and sort name", the plan said the folded name alone, and the
measurement picked the plan's side. The discriminating item existed on the live library — a
title whose padded sort form shares no substring with its folded name — and searched by that
sort fragment, the reference finds nothing.)* **`MatchedTerm` was never observed**: seventeen
measured hints across three terms arrived without it, so the claim that it "reports what matched
so a client can highlight it" described the schema, not the wire. It stays in the schema and
Atrium, like the reference, does not send it. `[probe: manual requests via tools/_probe.py,
Jellyfin 10.11.11, 2026-08-28]`

## 4. Data the feature owns

None. This feature is a read surface over what 003 and 004 produced, plus 007's user data. That is
the point of listing it: a query endpoint that owns state is a query endpoint with a cache bug.

## 5. Acceptance criteria

1. Every list endpoint except `/Items/Latest` and `/Items/Filters` returns the three-field envelope
   with `StartIndex` present; `/Items/Latest` returns a bare array.
2. `UserData` is present on every item without `Fields` or `EnableUserData`, and contains `Key` and
   `ItemId` — except the measured by-name quirks of criterion 17, whose rows carry none.
   *(Scoped at the 2026-08-28 audit — M36: as first written, this criterion contradicted the
   measured `/Genres` behaviour §3.9 records.)*
3. A field listed as `Fields`-gated in §3.2 is **absent** without the parameter and present with it.
4. Paging with `startIndex`/`limit` over a fixture of 100 items visits each item exactly once
   across all pages, for every supported `sortBy`.
5. `TotalRecordCount` is the pre-paging count on every list endpoint, **including by-name endpoints
   called without `limit`**.
6. `SortName` ordering matches 003 §3.7 for the fixture's awkward names.
7. `Random` returns a full set with no duplicates within one page.
8. `/Items/{itemId}` answers `404` identically for unknown and invisible items.
9. A user seeing no libraries gets an empty `/UserViews` envelope, not an error.
10. `/Shows/NextUp` returns at most one item per series — the first unplayed episode in
    `(season, episode)` order after the **highest-numbered** played one, so replaying an early
    episode moves nothing — and the most recently played series leads. *(Extended at the
    2026-08-28 audit — M39: the chain rule and the series ordering were measured, implemented
    and tested with only the one-per-series half stated here.)*
11. Season 0 sorts **first** — plain index order — in `/Shows/{seriesId}/Seasons`, as the
    reference sends it. *(Corrected by T12: the drafted criterion said "last" and the
    measurement said otherwise; see §3.8.)*
12. `Similar` and `InstantMix` return identical results for identical input on repeated calls,
    and `Similar` returns **exactly** `limit` rows for every seed type. *(Extended 2026-09-01 at
    010's gate: both halves were true and neither said that they are divergences. The reference
    draws `Similar` at random — [behaviours §3.23](../../docs/compatibility/behaviours.md) — and
    answers `limit + 4` on a movie seed —
    [behaviours §3.24](../../docs/compatibility/behaviours.md).)*
13. The album-credit set is a subset of the any-credit set, the two coincide as row sets for
    exactly [behaviours §5.3](../../docs/compatibility/behaviours.md#53-an-artist-in-two-music-libraries-is-two-rows)'s
    reason, and the credit distinction is observable at item level: `artistIds` finds the guest
    track and `albumArtistIds` does not. *(Restated by T14 — the draft asked the two routes to
    differ on the compilation, and under §5.3 there is no row that could show the difference.)*
14. `/Search/Hints` returns the hint shape, not the item shape — and, like the measured
    reference, sends no `MatchedTerm`, and matches case- and diacritic-insensitively against the
    **name**, never the sort name. *(Restated by T15: the draft required populating a field
    seventeen measured hints never carried; see §3.10. The matching clause was folded in at the
    2026-08-28 audit — M39.)*
15. A Tier 3 parameter is ignored, the response is `200`, and the parameter is recorded in the
    ignored-parameter report.
16. Every Tier 1 and Tier 2 parameter measurably narrows or reorders results on a fixture built to
    exercise it.

*(Criteria 17–22 were added at the 2026-08-28 audit — M35 to M41: each is a §3 behaviour that was
measured, implemented and tested with no criterion covering it, so the acceptance map could never
name its tests.)*

17. `/Genres` and `/MusicGenres` rows carry no `UserData`, and the two artist routes no
    `IsFolder`, where the same rows through `/Items` carry both — §3.9's measured route quirks,
    reproduced; year rows keep their `UserData`.
18. A bare `/Items/{itemId}` carries everything, unasked: `Fields` has nothing left to add
    (§3.2).
19. `/Items/Latest` groups: an episode under its series, a track under its album, a film under
    itself; a group surfaces as the container only when it holds more than one recent item, a
    group of one as the item itself; `LatestItemsExcludes` excludes a library, and
    `HidePlayedInLatest` steers played rows (§3.7).
20. `/UserItems/Resume` is ordered most recently played first, reports the position it resumes,
    pages, and is per user; an item played past the completion threshold does not appear,
    because 007 §3.7's rule cleared its position.
21. Request handling is §3.3's measured triad: an unrecognised token in an enum-valued parameter
    is dropped, never `400`; a value that cannot parse as its declared type is the
    problem-details `400`; a vocabulary type this version cannot produce is a filter that holds
    and matches nothing; and a `parentId` naming an unknown or invisible item is the same `404`
    as §3.6's.
22. An item with no `PremiereDate` sorts at January 1 of its `ProductionYear` under
    `sortBy=PremiereDate` rather than clumping with the dateless, and a request carrying
    `searchTerm` is ordered by match quality ahead of whatever `sortBy` asked for (§3.4).

## 6. Conformance

| Endpoint | Level | How it is proven |
|---|---|---|
| `GET /Items` | **L3** | Golden responses per parameter class, plus differential. The most-called endpoint in the API |
| `GET /Items/{itemId}` | **L3** | Golden per item type, plus differential |
| `GET /UserViews` | **L2** | Fixture with a restricted user |
| Discovery endpoints (§3.7) | **L2** | Golden responses on the fixture |
| Series navigation (§3.8) | **L2** | Fixture with specials and a missing season |
| By-name endpoints (§3.9) | **L2** | Including the no-`limit` count case (AC-5) |
| `GET /Search/Hints` | **L2** | Golden response; shape distinct from the envelope |

The **casing and unit sweeps** delivered by 001 cover every response model added here — which for
this feature means about seventy fields, the largest single surface in the project.

## 7. Open questions

| # | Question | Blocks | Resolved by |
|---|---|---|---|
| OQ-1 | Which fields the reference sends that §3.2 omits, and whether any client reads them | The bounded delta in §3.2 | Differential harness (010) — the single highest-value output it produces |
| OQ-2 | Which Tier 3 parameters real clients actually send | Promotion out of Tier 3 | The ignored-parameter report (AC-15) against real client traffic |
| OQ-5 | How the reference ranks `InstantMix` | Nothing; v1 diverges into determinism deliberately | Unmeasured. The `Similar` half of this question was answered at 010's gate and **its two decisions are made** — see the Resolved table below. `InstantMix` was not measured with it and no divergence is claimed for it until it is |
| OQ-7 | Whether the reference's Next Up really excludes a pristine specials season, as §3.8 states | §3.8's exclusion rule — it stands as v1's own reading until measured | **Owned since 2026-09-02**: a named comparison of [010 §3.10](../010-conformance-harness/spec.md), added by that list's D-6, run against the single-use reference instance 010 §3.1 stands up over the fixture. The question needs a library holding a series whose only unplayed episodes are season 0's, and the library measured on 2026-08-28 had none `[probe: tools/probe_next_up.py, Jellyfin 10.11.11, 2026-08-28]` — so it is built rather than found. **Answered on 2026-09-02, and §3.8's rule holds on both servers**: with every ordinary episode of a series marked played and season 0 left pristine, `GET /Shows/NextUp` offers **nothing at all** on the reference and nothing here — no special is proposed by either `[probe: tools/differential.py --named next-up-pristine-specials-season, Jellyfin 10.11.11, 2026-09-02]`. Closed |
### Resolved

| # | Question | Answer | Resolved by |
|---|---|---|---|
| OQ-5 (the `Similar` half) | How the reference ranks `Similar` | **It does not rank it at all, and `limit` is not a maximum on a movie seed.** The route filters on the seed's own genres and tags and orders what matches at random — four identical requests returned 48 distinct items with **none** in common — and a movie seed answers `limit + 4` rows where a series, an album and an artist answer exactly `limit`. Both readings were **decided on 2026-09-01** rather than left owed: each is a recorded divergence, argued in [behaviours §3.23 and §3.24](../../docs/compatibility/behaviours.md) and stated in §3.7 and AC-12. They were raised as G-1 and G-2 in [010 §7](../010-conformance-harness/spec.md), which records the same two decisions | `tools/probe_similar_ranking.py`, 2026-09-01 |
| OQ-6 | Whether `/Items/Latest` really returns a bare array on a live server, or the spec is wrong | **Yes — bare array, and three other endpoints have three further shapes.** §3.1 now records all four | `tools/probe_query_envelope.py`, 2026-08-26 |
| OQ-3 | The reference's tie-breaking key for each `sortBy` | **Almost none: `Name` is chained after `SortName` only, and nothing — not even the id — after anything else.** Measured, its movie sorts arrive stable with ties in id order, while `AlbumArtist`/`Artist` repeat identically yet their pages do not reassemble the one-shot list. §3.4 rule 2 is therefore a divergence from a defect, argued in behaviours §3.6 | `tools/probe_sort_stability.py`, 2026-08-27 |
| OQ-4 | The reference's completion threshold for `Resume` eligibility | **90% ceiling, 5% floor, 300-second minimum runtime — one ordered rule with six branches**, measured at 007 OQ-2 and specified in 007 §3.7, which §3.7's exclusion follows. The `probe_resume_threshold.py` this row used to name was never needed: `probe_playstate.py` answered it | `tools/probe_playstate.py`, 2026-08-26 |

## 8. References

- [docs/compatibility/api-surface-v1.md §4](../../docs/compatibility/api-surface-v1.md#4-library-navigation-and-queries)
- [docs/compatibility/behaviours.md §1.5, §2.1, §2.5, §3.1](../../docs/compatibility/behaviours.md)
- [specs/003 §3.7](../003-library-configuration-and-scanning/spec.md) — sort-name normalisation
- `[spec: GetItems, GetItem, GetUserViews, GetLatestMedia, GetResumeItems, GetNextUp, GetSeasons, GetEpisodes, GetArtists, GetAlbumArtists, GetGenres, GetMusicGenres, GetYears, GetSearchHints, GetQueryFiltersLegacy, GetSimilarItems, GetInstantMixFromItem, BaseItemDto, BaseItemDtoQueryResult, ItemFields, SearchHint]`
