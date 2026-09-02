# Glossary

The Jellyfin API is descended from Emby's, and carries vocabulary that is not obvious from the
names. Defining it once here keeps the specifications short and stops two documents meaning
different things by the same word.

## Items and the library

**BaseItem** — the single model behind everything in a library: a movie, a season, a track, an
album, a playlist, even a library root. Its concrete kind is the `Type` field. Much of the API's
shape follows from this one design choice: `/Items` can return heterogeneous results because
everything is a BaseItem.

**Item type** — the value of `Type`. The ones v1 produces: `Movie`, `Series`, `Season`, `Episode`,
`MusicArtist`, `MusicAlbum`, `Audio`, `Playlist`, `CollectionFolder`, `UserView`, `Folder`.

**Library / media folder** — a configured root directory with a collection type (`movies`,
`tvshows`, `music`). What the operator sets up.

**UserView** — a library *as a particular user sees it*, after that user's visibility policy is
applied. `/UserViews` returns these, not the raw libraries. A user with no access to a library does
not see its view.

**By-name item** — an item that exists because metadata mentions it, not because a file does:
`MusicArtist`, `Genre`, `Studio`, `Person`. These are served by their own endpoints (`/Artists`,
`/Genres`, `/MusicGenres`) and share one code path upstream — which is why they share one defect
(see [behaviours §3.1](compatibility/behaviours.md#31-totalrecordcount-is-0-on-by-name-endpoints-without-limit--class-b)).

**Fields** — a query parameter naming optional properties to include in a response, e.g.
`Fields=MediaSources,Overview`. Absent fields are omitted, not null. Note that `UserData` is *not*
one of these: it comes back always.

**Ticks** — the unit of every duration and position: 100 nanoseconds, 10,000,000 per second. A
.NET inheritance, and non-negotiable.

## Users and access

**UserData** — the per-user, per-item state: `IsFavorite`, `Played`, `PlayCount`,
`PlaybackPositionTicks`, and Jellyfin's extra `Key` and `ItemId`. Returned inline on every item.

**Policy** — a user's permissions and restrictions: administrator, which libraries are visible,
whether they are hidden from login screens, playback and deletion rights.

**AccessToken** — the 32-hex-character credential returned by `AuthenticateByName`, presented
afterwards by header or query parameter.

**Session** — a (user, device, client) triple the server tracks. Created at authentication,
identified by the `DeviceId` from `X-Emby-Authorization`, and the thing playback reporting attaches
to.

## Playback

**MediaSource** — one playable representation of an item: a path, a container, a bitrate, a list of
`MediaStream`s, and the `SupportsDirectPlay` / `SupportsDirectStream` / `SupportsTranscoding` flags.
An item can have several.

**MediaStream** — one track inside a media source: video, audio, subtitle or embedded image, with
its codec, language, channel layout and index.

**DeviceProfile** — the client's declaration of what it can play: containers, codecs, resolution and
bitrate ceilings, plus conditions. Posted to `PlaybackInfo`; the server's answer is a decision made
against it.

**PlaybackInfo** — the negotiation: client posts item id and profile, server answers with media
sources annotated with what the client should do.

**Direct play** — the client reads the original file byte-for-byte. Server does no processing beyond
serving `Range` requests. The fast path, and the one v1 optimises for.

**Direct stream** — the container is rewritten but the audio and video **elementary streams are
copied**, not re-encoded. Cheap: no decode, no encode. Also called **remuxing**, and the ceiling of
v1's playback scope.

**Transcoding** — decoding and re-encoding to a different codec, bitrate or resolution. Expensive,
CPU- or GPU-bound. **Out of v1.**

**PlaySessionId** — identifies one playback attempt across `PlaybackInfo`, the delivery request and
the three reporting calls, so the server can associate them and clean up.

## Wire format

**PascalCase** — the casing of every JSON property. See
[behaviours §1.1](compatibility/behaviours.md#11-property-casing-is-pascalcase).

**GUID / item id** — 32 lowercase hex characters, no dashes.

**QueryResult** — the envelope of every list response: `{"Items": [...], "TotalRecordCount": n,
"StartIndex": i}`.

**ImageTag** — a content hash identifying a specific version of an item's image, used for cache
invalidation. Advertised in `ImageTags`.

**`X-Emby-Authorization`** — the header carrying client identity on authentication:
`MediaBrowser Client="…", Device="…", DeviceId="…", Version="…"`. The `Emby` in the name is
historical; the reference reads this header and `Authorization` with the same grammar and accepts
either spelling, and the one route that demands client identity requires one of the two
([behaviours §2.4](compatibility/behaviours.md#24-there-are-five-authentication-mechanisms-and-one-of-them-wins)).

## Project vocabulary

**Delta** — any observable difference between Atrium and Jellyfin that would make a client behave
differently. The project's central metric, and the target is zero (Principle I).

**Non-improvement** — a change that would be genuinely better but creates a delta, and is therefore
recorded and not done. The list is in
[behaviours §6](compatibility/behaviours.md#6-non-improvements).

**Provenance** — the citation attached to every compatibility claim: a probe, a source line, or the
pinned spec. A claim without one is `⚠️ UNVERIFIED` (Principle II).

**Conformance level (L0–L3)** — how strongly a behaviour is proven. Defined in
[conformance.md](compatibility/conformance.md).
