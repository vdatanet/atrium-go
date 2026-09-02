# 005 T1 — what the reference actually sends, per item type

`[probe: tools/probe_item_shapes.py, Jellyfin 10.11.11, 2026-08-27]`

The question was one sentence: **which properties does the reference emit per item type, bare and
when asked?** Spec §3.2 sorted roughly seventy of them into three tiers, assembled from what two
clients *read* rather than from a measurement of what the reference *sends*, and T9 turns that
table into code. A wrong row there is a field silently absent, or silently present, on every
response of an item type.

Nine content types were reached through `/Items` (`Movie`, `Series`, `Season`, `Episode`,
`MusicArtist`, `MusicAlbum`, `Audio`, `Playlist`, `Folder`), up to twelve items each, plus the six
rows `/UserViews` returns. Each was fetched four ways: a bare list row, a list row with **all 49
members of the server's own `ItemFields` enum** requested, and the same item from
`/Items/{itemId}` both ways.

**Six findings. Five change spec §3.2; one changes `docs/compatibility/behaviours.md`, and it is
the one that reaches furthest outside this feature.**

## 1. There is no single item representation. There are three shapes

Spec §3.2 opened with *"One representation for every item type, discriminated by `Type`"*. That
is not what the server does. The same item, requested three ways, comes back three sizes:

| Shape | Movie | Series | Episode | MusicArtist | Audio |
|---|---|---|---|---|---|
| `/Items` list row, bare | 21 | 21 | 32 | 13 | 31 |
| `/Items/{itemId}`, bare, no `Fields` | 56 | 52 | 65 | 52 | 56 |

`/UserViews` is the third shape: **40 properties unasked**, on rows whose `Type` is
`CollectionFolder` or `UserView` — wider than any bare list row and close to a full body.

**`/Items/{itemId}` ignores the tiering entirely.** A bare full body already carries `Overview`,
`Genres`, `GenreItems`, `People`, `Studios`, `Path`, `ProviderIds`, `SortName`,
`PrimaryImageAspectRatio`, `MediaSources`, `MediaStreams`, `Chapters`, `Etag`, `Tags`, `Taglines`,
`ExternalUrls`, `DateCreated` and more — up to **39 properties a bare list row does not have**, for
`MusicArtist`. `Fields` widens the list row; it has nothing left to add to the full body.

The consequence for T9 is structural, not cosmetic: the registry is a rule about `/Items` list
rows. The item route needs its own rule, and it is *emit everything*.

## 2. Three properties are always present and were in no tier

`ChannelId`, `ImageBlurHashes` and `LocationType` arrive on **every sampled item of every content
type**, bare, with nothing requested. §3.2's always-present table had nine rows; the measurement
says twelve.

`ImageBlurHashes` is the expensive one to have missed: it mirrors `ImageTags` with a BlurHash per
image id, and a client that renders placeholders reads it.

## 3. Seven of the twelve "Common" names are gated on a list row

§3.2's *"Present when the item type has them"* Common group listed twelve. Seven of them are
**never** on a bare list row of any content type, and arrive only when asked for:

`SortName`, `Overview`, `Genres`, `GenreItems`, `Studios`, `People`, `PrimaryImageAspectRatio`.

The five that really are per-type: `ProductionYear`, `PremiereDate`, `RunTimeTicks`,
`OfficialRating`, `CommunityRating`.

Six of the seven are `ItemFields` tokens, which is what made the row wrong: **a token is gated by
definition.** A name that appears in the enum cannot also be unconditional, and §3.2 had six of
them in a tier that says it is.

`PrimaryImageAspectRatio` is the one with a dependency behind it — 004 owes it to this feature
through `item_images` width and height, and §3.2 said a list row carries it. It does not.

## 4. `ChildCount` arrives unasked on `Playlist`

Claimed gated; present on 2 of 2 sampled playlists without being requested. The sample is two
rows, which is what the library has, so this is recorded as measured-but-thin.

## 5. `/UserViews` carries sixteen names that `/Items` gates

Unasked, on all six rows: `ChildCount`, `DateCreated`, `DateLastMediaAdded`, `Etag`,
`ExternalUrls`, `GenreItems`, `Genres`, `ParentId`, `Path`, `People`, `PrimaryImageAspectRatio`,
`ProviderIds`, `SortName`, `Studios`, `Taglines`, `Tags`.

This is why the first run of this probe reported the wrong thing. Folding `/UserViews` in with the
content types let one fat row promote a gated name to "per-type" for every type at once, and
`Genres`, `Studios`, `People`, `SortName` and `PrimaryImageAspectRatio` came out looking per-type
when they are gated everywhere `/Items` is involved. **The tiers are a property of the route, not
of the item type**, and the probe now classifies over `/Items` alone and reports `/UserViews`
separately.

`/Items?IncludeItemTypes=CollectionFolder` answers **zero rows**. `/UserViews` is the only way to
reach a library root, so nothing about these rows could have been measured through `/Items`.

## 6. A null property is *not* absent everywhere — `ChannelId` is always `null`

This one is not about 005.

[behaviours §1.7](../../../docs/compatibility/behaviours.md#17-a-null-property-is-absent-everywhere-by-one-setting)
says the reference omits *any* property whose value is null — *"Not per-property and not a
judgement"* — and cites the one line of JSON configuration that does it.

Measured: **`ChannelId` arrived as an explicit `null` 208 times**, on every item of every type, in
list rows and full bodies alike. `ParentId` arrived as an explicit `null` on 2 of the 6
`/UserViews` rows.

```json
"OfficialRating": "ES-18",
"ChannelId": null,
"CommunityRating": 6.7,
```

The entry is not deleted and the configuration it cites is not disputed — something overrides it
for at least these two properties. **What that something is has not been established from here**:
this repository had no `reference/` checkout when this was written on 2026-08-27, so the cause is
unmeasured and the
entry now records the exception with its measurement instead of a mechanism.

It matters well beyond 005. §1.7 says Atrium omits nulls *in the base model, rather than per
route*, precisely so nobody has to remember. If `ChannelId` must be `null` on every item, that is
a per-property exception on the single highest-traffic response in the API, and a client
distinguishing absent from null sees it on every row of every list.

## 7. `SeriesThumbImageTag` was not observed at all

It is in §3.2's Episode group and it appeared in no body of any kind — bare, asked for, or full —
across twelve episodes. A null property is omitted, so "gated" and "none of these twelve series
has a Thumb" look identical from outside. It stays in the spec, marked unconfirmed. Deleting a row
on one library's evidence is the mistake this probe exists to avoid making in the other direction.

## What was deliberately not added

The reference sends far more than §3.2 lists, and the measurement was not treated as a shopping
list. `SeasonName`, `ParentBackdropItemId`, `ParentLogoItemId`, `ParentLogoImageTag`, `Container`,
`VideoType`, `HasLyrics`, `NormalizationGain`, `CriticRating` and others are all emitted on list
rows and none of them entered the spec: §3.2's set is *the union of what the two analysed clients
read*, and Principle VI applies to fields as it does to endpoints. What the measurement corrects
is which **tier** a listed field is in, not how many fields are listed.

The three that did enter — `ChannelId`, `LocationType`, `ImageBlurHashes` — are there for a
different reason: they are emitted **unconditionally**, so omitting them is a delta on every item
of every response, not a narrower answer.

## The measurement

Bare presence per type, as present/sampled. `-` is absent from every sampled body. The tier column
is classified over the nine `/Items` content types; `UserViews` is shown but not counted.

```
  bare presence per type, as present/sampled. `-` is absent from every sampled body.

  property                  tier            Movie       Series       Season      Episode  MusicArtist   MusicAlbum        Audio     Playlist       Folder    UserViews
  --------------------------------------------------------------------------------------------------------------------------------------------------------------------
  BackdropImageTags         always          12/12        12/12        12/12        12/12        12/12        12/12        12/12          2/2        12/12          6/6
  ChannelId                 always          12/12        12/12        12/12        12/12        12/12        12/12        12/12          2/2        12/12          6/6
  Id                        always          12/12        12/12        12/12        12/12        12/12        12/12        12/12          2/2        12/12          6/6
  ImageBlurHashes           always          12/12        12/12        12/12        12/12        12/12        12/12        12/12          2/2        12/12          6/6
  ImageTags                 always          12/12        12/12        12/12        12/12        12/12        12/12        12/12          2/2        12/12          6/6
  IsFolder                  always          12/12        12/12        12/12        12/12        12/12        12/12        12/12          2/2        12/12          6/6
  LocationType              always          12/12        12/12        12/12        12/12        12/12        12/12        12/12          2/2        12/12          6/6
  MediaType                 always          12/12        12/12        12/12        12/12        12/12        12/12        12/12          2/2        12/12          6/6
  Name                      always          12/12        12/12        12/12        12/12        12/12        12/12        12/12          2/2        12/12          6/6
  ServerId                  always          12/12        12/12        12/12        12/12        12/12        12/12        12/12          2/2        12/12          6/6
  Type                      always          12/12        12/12        12/12        12/12        12/12        12/12        12/12          2/2        12/12          6/6
  UserData                  always          12/12        12/12        12/12        12/12        12/12        12/12        12/12          2/2        12/12          6/6
  AirDays                   per-type            -        12/12            -            -            -            -            -            -            -            -
  Album                     per-type            -            -            -            -            -            -        12/12            -            -            -
  AlbumArtist               per-type            -            -            -            -            -        12/12        12/12            -            -            -
  AlbumArtists              per-type            -            -            -            -            -        12/12        12/12            -            -            -
  AlbumId                   per-type            -            -            -            -            -            -        12/12            -            -            -
  AlbumPrimaryImageTag      per-type            -            -            -            -            -            -        12/12            -            -            -
  ArtistItems               per-type            -            -            -            -            -        12/12        12/12            -            -            -
  Artists                   per-type            -            -            -            -            -        12/12        12/12            -            -            -
  ChildCount                per-type            -            -            -            -            -            -            -          2/2            -          6/6
  CommunityRating           per-type        12/12        12/12            -        10/12            -            -            -            -            -            -
  Container                 per-type        12/12            -            -        12/12            -            -        12/12            -            -            -
  CriticRating              per-type        11/12            -            -            -            -            -            -            -            -            -
  DisplayOrder              per-type            -        12/12            -            -            -            -            -            -            -            -
  EndDate                   per-type            -        12/12            -            -            -            -            -            -            -            -
  HasLyrics                 per-type            -            -            -            -            -            -        12/12            -            -            -
  HasSubtitles              per-type        11/12            -            -         6/12            -            -            -            -            -            -
  IndexNumber               per-type            -            -        12/12        12/12            -            -        12/12            -            -            -
  NormalizationGain         per-type            -            -            -            -            -            -         6/12            -            -            -
  OfficialRating            per-type        10/12        12/12            -            -            -            -            -            -            -            -
  ParentBackdropImageTags   per-type            -            -        11/12        12/12            -        10/12        12/12            -         9/12            -
  ParentBackdropItemId      per-type            -            -        11/12        12/12            -        10/12        12/12            -         9/12            -
  ParentIndexNumber         per-type            -            -            -        12/12            -            -        12/12            -            -            -
  ParentLogoImageTag        per-type            -            -         9/12        10/12            -        10/12        12/12            -         9/12            -
  ParentLogoItemId          per-type            -            -         9/12        10/12            -        10/12        12/12            -         9/12            -
  ParentThumbImageTag       per-type            -            -        10/12        12/12            -            -            -            -            -            -
  ParentThumbItemId         per-type            -            -        10/12        12/12            -            -            -            -            -            -
  PremiereDate              per-type        12/12        12/12        12/12        12/12            -        12/12        12/12            -            -            -
  ProductionYear            per-type        12/12        12/12        12/12        12/12            -        12/12        12/12            -            -            -
  RunTimeTicks              per-type        12/12        12/12            -        12/12        12/12        12/12        12/12          2/2            -            -
  SeasonId                  per-type            -            -            -        12/12            -            -            -            -            -            -
  SeasonName                per-type            -            -            -        12/12            -            -            -            -            -            -
  SeriesId                  per-type            -            -        12/12        12/12            -            -            -            -            -            -
  SeriesName                per-type            -            -        12/12        12/12            -            -            -            -            -            -
  SeriesPrimaryImageTag     per-type            -            -        12/12        12/12            -            -            -            -            -            -
  Status                    per-type            -        12/12            -            -            -            -            -            -            -            -
  VideoType                 per-type        12/12            -            -        12/12            -            -            -            -            -            -
  AlbumCount                gated               -            -            -            -            -            -            -            -            -            -
  ArtistCount               gated               -            -            -            -            -            -            -            -            -            -
  CanDelete                 gated               -            -            -            -            -            -            -            -            -          6/6
  CanDownload               gated               -            -            -            -            -            -            -            -            -          6/6
  Chapters                  gated               -            -            -            -            -            -            -            -            -            -
  CumulativeRunTimeTicks    gated               -            -            -            -            -            -            -            -            -            -
  DateCreated               gated               -            -            -            -            -            -            -            -            -          6/6
  DateLastMediaAdded        gated               -            -            -            -            -            -            -            -            -          6/6
  DisplayPreferencesId      gated               -            -            -            -            -            -            -            -            -          6/6
  EnableMediaSourceDisplay  gated               -            -            -            -            -            -            -            -            -          6/6
  EpisodeCount              gated               -            -            -            -            -            -            -            -            -            -
  Etag                      gated               -            -            -            -            -            -            -            -            -          6/6
  ExternalUrls              gated               -            -            -            -            -            -            -            -            -          6/6
  ForcedSortName            gated               -            -            -            -            -            -            -            -            -          1/6
  GenreItems                gated               -            -            -            -            -            -            -            -            -          6/6
  Genres                    gated               -            -            -            -            -            -            -            -            -          6/6
  Height                    gated               -            -            -            -            -            -            -            -            -            -
  IsHD                      gated               -            -            -            -            -            -            -            -            -            -
  LocalTrailerCount         gated               -            -            -            -            -            -            -            -            -          6/6
  LockData                  gated               -            -            -            -            -            -            -            -            -          6/6
  LockedFields              gated               -            -            -            -            -            -            -            -            -          6/6
  MediaSources              gated               -            -            -            -            -            -            -            -            -            -
  MediaStreams              gated               -            -            -            -            -            -            -            -            -            -
  MovieCount                gated               -            -            -            -            -            -            -            -            -            -
  MusicVideoCount           gated               -            -            -            -            -            -            -            -            -            -
  OriginalTitle             gated               -            -            -            -            -            -            -            -            -            -
  Overview                  gated               -            -            -            -            -            -            -            -            -            -
  ParentId                  gated               -            -            -            -            -            -            -            -            -          6/6
  Path                      gated               -            -            -            -            -            -            -            -            -          6/6
  People                    gated               -            -            -            -            -            -            -            -            -          6/6
  PlayAccess                gated               -            -            -            -            -            -            -            -            -          6/6
  PrimaryImageAspectRatio   gated               -            -            -            -            -            -            -            -            -          6/6
  ProductionLocations       gated               -            -            -            -            -            -            -            -            -            -
  ProgramCount              gated               -            -            -            -            -            -            -            -            -            -
  ProviderIds               gated               -            -            -            -            -            -            -            -            -          6/6
  RecursiveItemCount        gated               -            -            -            -            -            -            -            -            -            -
  RemoteTrailers            gated               -            -            -            -            -            -            -            -            -          6/6
  SeriesCount               gated               -            -            -            -            -            -            -            -            -            -
  SeriesStudio              gated               -            -            -            -            -            -            -            -            -            -
  SongCount                 gated               -            -            -            -            -            -            -            -            -            -
  SortName                  gated               -            -            -            -            -            -            -            -            -          6/6
  SpecialFeatureCount       gated               -            -            -            -            -            -            -            -            -          6/6
  Studios                   gated               -            -            -            -            -            -            -            -            -          6/6
  Taglines                  gated               -            -            -            -            -            -            -            -            -          6/6
  Tags                      gated               -            -            -            -            -            -            -            -            -          6/6
  TrailerCount              gated               -            -            -            -            -            -            -            -            -            -
  Trickplay                 gated               -            -            -            -            -            -            -            -            -            -
  Width                     gated               -            -            -            -            -            -            -            -            -            -
```

```
  ItemFields enum              49 members, all asked for; 0 of spec 3.2's gated names are not members
  Movie                        12 sampled; bare list row 21 properties, bare /Items/{itemId} 56; the full route adds unasked CanDelete, CanDownload, Chapters, DateCreated, DisplayPreferencesId, EnableMediaSourceDisplay, Etag, ExternalUrls, ForcedSortName, GenreItems, Genres, Height, IsHD, LocalTrailerCount, LockData, LockedFields, MediaSources, MediaStreams, OriginalTitle, Overview, ParentId, Path, People, PlayAccess, PrimaryImageAspectRatio, ProductionLocations, ProviderIds, RemoteTrailers, SortName, SpecialFeatureCount, Studios, Taglines, Tags, Trickplay, Width
  Series                       12 sampled; bare list row 21 properties, bare /Items/{itemId} 52; the full route adds unasked CanDelete, CanDownload, ChildCount, CumulativeRunTimeTicks, DateCreated, DateLastMediaAdded, DisplayPreferencesId, EnableMediaSourceDisplay, Etag, ExternalUrls, ForcedSortName, GenreItems, Genres, LocalTrailerCount, LockData, LockedFields, OriginalTitle, Overview, ParentId, Path, People, PlayAccess, PrimaryImageAspectRatio, ProviderIds, RecursiveItemCount, RemoteTrailers, SortName, SpecialFeatureCount, Studios, Taglines, Tags
  Season                       12 sampled; bare list row 24 properties, bare /Items/{itemId} 51; the full route adds unasked CanDelete, CanDownload, ChildCount, DateCreated, DateLastMediaAdded, DisplayPreferencesId, EnableMediaSourceDisplay, Etag, ExternalUrls, GenreItems, Genres, LocalTrailerCount, LockData, LockedFields, ParentId, People, PlayAccess, PrimaryImageAspectRatio, ProviderIds, RecursiveItemCount, RemoteTrailers, SeriesStudio, SortName, SpecialFeatureCount, Studios, Taglines, Tags
  Episode                      12 sampled; bare list row 32 properties, bare /Items/{itemId} 65; the full route adds unasked CanDelete, CanDownload, Chapters, DateCreated, DisplayPreferencesId, EnableMediaSourceDisplay, Etag, ExternalUrls, ForcedSortName, GenreItems, Genres, Height, IsHD, LocalTrailerCount, LockData, LockedFields, MediaSources, MediaStreams, Overview, ParentId, Path, People, PlayAccess, PrimaryImageAspectRatio, ProviderIds, RemoteTrailers, SeriesStudio, SortName, SpecialFeatureCount, Studios, Taglines, Tags, Trickplay, Width; list-only HasSubtitles
  MusicArtist                  12 sampled; bare list row 13 properties, bare /Items/{itemId} 52; the full route adds unasked AlbumCount, ArtistCount, CanDelete, CanDownload, ChildCount, CumulativeRunTimeTicks, DateCreated, DateLastMediaAdded, DisplayPreferencesId, EnableMediaSourceDisplay, EpisodeCount, Etag, ExternalUrls, ForcedSortName, GenreItems, Genres, LocalTrailerCount, LockData, LockedFields, MovieCount, MusicVideoCount, Overview, ParentId, Path, People, PlayAccess, PrimaryImageAspectRatio, ProgramCount, ProviderIds, RecursiveItemCount, RemoteTrailers, SeriesCount, SongCount, SortName, SpecialFeatureCount, Studios, Taglines, Tags, TrailerCount
  MusicAlbum                   12 sampled; bare list row 23 properties, bare /Items/{itemId} 53; the full route adds unasked CanDelete, CanDownload, ChildCount, CumulativeRunTimeTicks, DateCreated, DateLastMediaAdded, DisplayPreferencesId, EnableMediaSourceDisplay, Etag, ExternalUrls, ForcedSortName, GenreItems, Genres, LocalTrailerCount, LockData, LockedFields, Overview, ParentId, Path, People, PlayAccess, PrimaryImageAspectRatio, ProviderIds, RecursiveItemCount, RemoteTrailers, SortName, SpecialFeatureCount, Studios, Taglines, Tags
  Audio                        12 sampled; bare list row 31 properties, bare /Items/{itemId} 56; the full route adds unasked CanDelete, CanDownload, DateCreated, DisplayPreferencesId, EnableMediaSourceDisplay, Etag, ExternalUrls, GenreItems, Genres, LocalTrailerCount, LockData, LockedFields, MediaSources, MediaStreams, ParentId, Path, People, PlayAccess, PrimaryImageAspectRatio, ProviderIds, RemoteTrailers, SortName, SpecialFeatureCount, Studios, Taglines, Tags; list-only NormalizationGain
  Playlist                     2 sampled; bare list row 14 properties, bare /Items/{itemId} 41; the full route adds unasked CanDelete, CanDownload, CumulativeRunTimeTicks, DateCreated, DateLastMediaAdded, DisplayPreferencesId, EnableMediaSourceDisplay, Etag, ExternalUrls, GenreItems, Genres, LocalTrailerCount, LockData, LockedFields, ParentId, Path, People, PlayAccess, PrimaryImageAspectRatio, ProviderIds, RecursiveItemCount, RemoteTrailers, SortName, SpecialFeatureCount, Studios, Taglines, Tags
  Folder                       12 sampled; bare list row 16 properties, bare /Items/{itemId} 38; the full route adds unasked CanDelete, CanDownload, ChildCount, DateCreated, DateLastMediaAdded, DisplayPreferencesId, EnableMediaSourceDisplay, Etag, ExternalUrls, GenreItems, Genres, LocalTrailerCount, LockData, LockedFields, ParentId, Path, People, PlayAccess, ProviderIds, RecursiveItemCount, RemoteTrailers, SortName, SpecialFeatureCount, Studios, Taglines, Tags; list-only ParentBackdropImageTags, ParentBackdropItemId, ParentLogoImageTag, ParentLogoItemId
  UserViews                    6 sampled; bare list row 40 properties, bare /Items/{itemId} 39; list-only ForcedSortName
  explicit nulls               ChannelId x208, ParentId x2
  UserData keys                IsFavorite, ItemId, Key, LastPlayedDate, PlayCount, PlaybackPositionTicks, Played, PlayedPercentage, UnplayedItemCount
  ImageTags when empty         `{}` on Folder
  PrimaryImageAspectRatio      unasked tier: gated
  UserViews rows report Type   CollectionFolder, UserView
  UserViews shape              40 properties unasked, including ChildCount, DateCreated, DateLastMediaAdded, Etag, ExternalUrls, GenreItems, Genres, ParentId, Path, People, PrimaryImageAspectRatio, ProviderIds, SortName, Studios, Taglines, Tags
```

## What was not measured

- **No type was unreachable.** All nine content types and `/UserViews` returned rows, so nothing
  in §3.2 is left resting on a guess — except the sample sizes: `Playlist` had 2 rows and
  `/UserViews` 6, against 12 for everything else.
- **`UserView` versus `CollectionFolder` as separate rows.** `/UserViews` returns both `Type`
  values and this probe measured them as one set. Splitting them belongs to T11, whose route it
  is.
- **`GenreItems` is emitted by the server but is not among the 1043 property names extracted from
  the pinned 10.11.10 document** (`docs/compatibility/property-names.json`). Not pursued here:
  it is either a difference between 10.11.10 and 10.11.11 or a limit of the extractor's schema
  walk, and either way it belongs to whoever next touches that generator.

---

## Addendum — measured at T9, 2026-08-28

`[probe: manual requests via tools/_probe.py, Jellyfin 10.11.11, 2026-08-28]`

T9 turned the tables above into the registry, and four questions the first run had not asked
needed answers before the emitters could be written.

**A by-name row has no `IsFolder`, anywhere.** A `Genre` from `/Genres` (10 properties bare),
from `/Items?ids=` (11 — `UserData` joins), and in full (45): `IsFolder` appears in none of
them, while `LocationType` is `FileSystem` and `MediaType` is `Unknown`. The same holds for
`MusicGenre` and `Year`. §3.2's always-present table now records the hole.

**`UserData` is per-route on the by-name endpoints, per-item everywhere else.** A genre row from
`/Genres` carries **no** `UserData`; the same genre through `/Items?ids=` carries it, as does its
full body, and an artist row from `/Artists` carries it too. So the omission belongs to
`/Genres`-family routes, not to the by-name types — T14's to measure per route (`/Years`
unmeasured) and reproduce.

**A container's `UserData` is a rollup.** A bare `Series` list row: `UnplayedItemCount` beside
the stored keys; a `Season` with everything watched: `UnplayedItemCount: 0, Played: true` with
`PlayCount: 0`. The played flag is a statement about the subtree, not the stored row, and the
count rides every bare container row.

**`ExternalUrls` patterns, and the `Etag` shape.** A movie: `IMDb → https://www.imdb.com/title/{id}`,
`TMDB → https://www.themoviedb.org/movie/{id}`; a series: `.../tv/{id}`; an artist:
`MusicBrainz Artist → https://musicbrainz.org/artist/{id}`; an album: `MusicBrainz Album →
/release/{id}`, `MusicBrainz Album Artist → /artist/{id}`, `MusicBrainz Release Group →
/release-group/{id}` (and `TheAudioDb Album`, whose ids 004 never writes); a track with no ids:
`[]`, the empty list, not an absent property. `Etag` is 32 lowercase hex on every body sampled.

**And one document gap found by declaring a field:** `GenreItems` is emitted by the server and
declared by the `10.11.11` document, but the pinned `10.11.10` document — and therefore
`property-names.json` — does not contain it (`LockedFields` likewise). Recorded in
[reference-target §1](../../../docs/compatibility/reference-target.md#1-the-pinned-version); the
alias sweep carries it as a measured exception.
