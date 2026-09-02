---
feature: 009-playlists
title: Playlists
status: Implemented
created: 2026-08-26
updated: 2026-09-01
amended: 2026-08-31 at the plan gate — §3.7 and a new AC-19 state the *bytes* of the refusal a caller gets for naming another user, which the spec gate had measured and recorded only as a status. The reference answers the 25-byte `text/plain` body every controller-level refusal carries; this server answered an empty `403` for that whole class, on the argument that it is decided where the empty `401` is, with a `⚠️` in the code saying the shape was unmeasured because no non-administrator account existed to produce one. The visibility probe made one. It is a wire difference on a route 005 already ships, so the correction is taken where the refusal is decided rather than on 009's own two routes — decided by the user at the plan gate; and 2026-08-31 by T1 — §3.5 stated the move's arithmetic for a caller who sees the whole playlist and left the other caller to §3.7. It now says both: the index is judged against the list that reader was given and the entry lands at `newIndex` **of it**, where the reference is off by one on every downward move and will reorder an entry that reader was never shown. Neither difference is reachable against a reference server — what it hides is hidden by a parental-rating check — so both belong to §3.7's divergence, and AC-17 gains the clause. Plus the provenance: all thirty (source, `newIndex`) pairs are measured where one was; and 2026-08-31 by T2 — a `403` is **two** shapes and §3.7 had described one. The content type was never measured on either: the probe cited for both printed forty bytes of body and no headers, which cannot separate an empty body from a body-less refusal. Measured, the controller's refusal is `text/plain` with no `charset` and §3.8's elevated-controller refusal carries no content type and no body at all — so §3.7 and §3.8 were never in conflict, AC-18 and AC-19 assert different bytes on purpose, and only the first shape is the one the shared handler answers. Plus two raise sites that shared that handler and are neither measurement: a live token whose account was disabled (002 OQ-5, still open) and one user reading another, which the reference does not refuse at all and 2026-08-31 by T3 — §3.2 said `MediaType` is *inferred at creation* and left open whether it then follows the contents, which is the difference between a value a row carries and a value anybody derives. Measured, it is fixed at creation and never revised: a playlist created empty answers `Audio` after a film is added, one created from a film answers `Video` after a track is, and the body's own `MediaType` outranks both. §3.2 says so, and gains a third refusal — an unrecognised `MediaType` is a `400` in the validation shape, not a dropped token. §4 gains the consequence nobody had asked about: `mediaTypes=` filters playlists by the stored value, so the parameter is the one place a per-row media type meets a listing that answers it from the kind, and that is left undecided rather than improvised and 2026-08-31 by T5 — §3.7's table said which callers may edit and never what a refusal to edit looks like, and AC-13 and AC-14 said "refused" and "may not" with no status at all. Measured, every "no" in that column is `403` with **no body and no content type** — the body-less shape, on a permission test the playlist controller makes itself, which means behaviours §1.11's split between a controller and a policy is really the split between a refusal *thrown* and a refusal *returned*, and both happen in the same action. Plus the two classes nobody had produced: a `CanEdit: false` share is stored by the create body and is a reader who is refused the move, and a public playlist's reader is refused identically and 2026-08-31 by T7 - §3.1 said a playlist cannot hold one item twice and §3.4 said de-duplication is two stages, both measured on a playlist holding one entry. Measured on playlists that already held several: the entry already there keeps its position and the first occurrence of a repeat is the one that survives - and the stage that compares against the existing entries reads an id cache that is empty until an entry has been resolved, so 6 of 8 identical requests added an item the playlist already held. The duplicate that produces is unaddressable - two rows, one `PlaylistItemId`, `Move` reordering the first and `Remove` deleting both - so §3.1, §3.4 and AC-5 now state Atrium's rule with its argument, and the feature ships a fourth divergence (behaviours §3.18) and 2026-08-31 by T8 — §3.2's error table called its validation `400` one refusal keyed on the property. It is **three**, at three keys, told apart by what about the body was wrong: a `Name` that is absent is keyed `$` — the deserialiser refusing the document before any property is validated — a `Name` that is present and **null** is keyed `Name`, and a malformed identifier in `Ids` or `UserId` is keyed with the empty string. None of the three names the action parameter behaviours §1.11 said every body refusal names; that row belongs to a **required** body, and this route's is optional. The query form is real and is implemented — `?name=` with no body at all creates a playlist and a query value beats the body's — but a body that fails to deserialise is refused before the query is read, so the two sources merge after binding rather than instead of it. Two requests the reference cannot serve it answers anyway: no name in either source is a **`500`**, and a `UserId` naming nobody is a `200` creating a playlist no rule in §3.7 can reach; both are refused here, behaviours §3.19. And one value is refused two ways on one route — `MediaType: Nonsense` in the body is T3's validation `400`, `?mediaType=Nonsense` in the query is dropped and the playlist created and 2026-09-01 by T10 — §3.4's *"every kind of container"* named five kinds and the rule is wider than any list: a plain folder, **the library root itself** and **another playlist** expand too, recursively, and the expansion lands where the container was named in the batch rather than at the end. The orders are two — a folder's own for a folder, and album artist, album, sort name over the *credited* tracks for an artist — and creation expands as well, which moves the media type it infers: a series in `Ids` creates a `Video` playlist where the series' own media type is `Unknown` and the fallback is `Audio`, so §3.2's *"the media type of the first resolvable id"* is the media type of what that id expanded to. Plus the identifier neither document had separated: an id of **all zeros** is refused with the bare-text `400` by the add route and by creation in the position where an ordinary unknown id is skipped, because the reference rejects an empty identifier in its lookup rather than failing to find it — and it is *not* a refusal on the removal, which looks nothing up. §3.5 gains the removal's four measured classes and the third request behaviours §3.19 now carries: a malformed playlist id is the binder's `400` on the add and a **`500`** on the removal, one path, two bindings. AC-7 states the width and AC-13 the condition it was missing — the administrator's `403` is reachable only on a playlist that administrator can see, and is a `404` otherwise and 2026-09-01 by T11 — §3.5 described the move as one arithmetic and one identifier, and the route has **three path segments that bind three different ways**. The playlist id is parsed, so plain, dashed and braced all address it; the entry id is not parsed at all but compared as text against the plain 32-character spelling, so an upper-case entry id moves the entry and a **dashed one moves nothing** — which makes this the one route in the feature that must not canonicalise the identifier it is given, and which is why an entry id that is not an identifier at all is a silent `204` rather than a refusal; and the index is a number in the path, so a `newIndex` that is not one is the validation `400` keyed `newIndex`, which is parity and needs no code. §3.5's boundary table gains those rows and two more it never had: a **malformed playlist id is a `500`** here as it is on the removal and not the binder's `400` as it is on the addition — behaviours §3.19's fourth request — and a caller who may not edit is refused `403` **even when the index is one the reference crashes on**, so the refusals are ordered `404`, `403`, then the index, then the entry, and the `400` this feature makes its own is reachable only by a caller who may edit and 2026-09-01 by T12 — §3.6 said `404` for an unknown **or invisible** item, and for a playlist the second half is false: `DELETE /Items/{itemId}` applies no visibility test to a playlist at all, so a caller who is answered `404` by the read route and by `GET /Items/{itemId}` is answered `401` here and learns the playlist exists — measured, replicated, and the reason §3.7's administrator row is the one cell in that table whose “yes” does not depend on seeing the playlist. Media is filtered the other way and keeps the `404`. Two identifiers the section had not named answer before either: a malformed one is the binder's validation `400` keyed `itemId`, and an all-zeros one is the same bare-text `400` §3.4's write routes make. And v1's rule is stated as what it is — **only a playlist deletes** — where it had been written as *“removes no file from disk”*, which would have permitted deleting a genre the next scan rebuilds. AC-12 and AC-13 gain those clauses, and behaviours §4.3 gains the second cell where the divergence is observable: for a caller with no deletion permission the reference **refuses** too, with the same 21 bytes, so Atrium's `403` differs there from a refusal rather than from a deletion and 2026-09-01 by T13 — §3.8 said the route *"reads `Name`, on a `Playlist`, for an administrator"*, and every clause of that sentence but the last was too small. The body is a **whole item** and three of its properties are required — `Genres`, `Tags` and `ProviderIds`, absent or `null`, each the 25-byte `400`, measured by dropping all thirty-nine of a read's properties one at a time — so the client's round trip is load-bearing and a body carrying only a `Name` is refused by a stock reference server. The route **applies seven fields beside `Name`** — `Overview`, `ForcedSortName`, `OfficialRating`, `CustomRating`, `ProductionYear`, `Genres` and `Tags` — which Atrium does not, and that narrowing is now a recorded gap rather than a sentence claiming parity. A body with **no** `Name` at all is a `204` that *erases* the name, refused here as behaviours §3.21, the feature's sixth divergence. And the refusal is ordered ahead of the binder: a non-administrator meets the empty `403` for a path segment that is not an identifier, where an administrator meets the validation `400` — which is what an elevated *controller* means on the wire, and is why the check cannot live inside the route. AC-18 gains the ordering and the body; §3.8 gains the identifier table, measured on this method rather than inherited from the `DELETE` that shares its path and 2026-09-01 by T14 — AC-15 asserted that a request naming the playlist's owner in `userId` is part of the `404` it describes. It is not: every route in this project that takes that parameter refuses a non-administrator with AC-19's 25-byte `403`, before any playlist is looked up. One request cannot have two shapes, and the two criteria beside it assert the one the wire has, so the clause is corrected rather than the code. Plus what the closing review found the criteria owed and the documents did not say: AC-20's rescan had no test of any kind — on the one item a rescan cannot rebuild — AC-5's *"on both the creation and the addition paths"* had only ever asked the addition, and AC-13's *"the same three routes"* had asked one of the three
depends_on: [005]
---

# 009 — Playlists

> **This document describes WHAT and WHY only.** No technology names, no storage decisions.

## 1. Purpose

Let a user create an ordered list of items, read it, add to it, remove from it and reorder it.

Playlists are the only place in v1 where a client **writes structure** to the server. Everything
else it writes is per-user state.

**Client behaviour unlocked:** building and editing a playlist from the app.

## 2. Scope

**In scope**

- `POST /Playlists`, `GET /Playlists/{playlistId}/Items`,
  `POST` and `DELETE /Playlists/{playlistId}/Items`,
  `POST /Playlists/{playlistId}/Items/{itemId}/Move/{newIndex}`,
  `DELETE /Items/{itemId}` for deleting a playlist.
- `POST /Items/{itemId}` for renaming a playlist — §3.8. Added at the spec-review gate on
  2026-08-31, because it has a named consumer: it is how the music client renames a playlist
  `[client-contract: 2026-08-29, §10]`, and it was the one operation that client calls which no
  feature owned.
- Playlist entries, their identity, and why that identity is not what it looks like.
- Ownership and visibility, including the two disclosures this feature diverges on.

**Out of scope**

- `POST /Playlists/{playlistId}` (`UpdatePlaylist`) and `/Playlists/{playlistId}/Users` (sharing).
  No analysed client calls either.

  > This exclusion cost something, and the cost is written down rather than discovered later.
  > `UpdatePlaylist` is the **only** route that renames a playlist for an owner who is not an
  > administrator: the route the client actually calls refuses that owner with `403`, and this one
  > answers `204` for the same user on the same playlist.
  > `[probe: tools/probe_playlist_rename.py, Jellyfin 10.11.11, 2026-08-31]` Principle VI decides
  > it anyway — an endpoint with no named consumer is not added — and the consequence is §3.8's
  > last paragraph.

- Smart or rule-based playlists.
- Reading or writing playlist files on disk. The reference keeps a playlist as a directory, and
  what of that reaches the wire is §3.2's last paragraph and §4.
- Deleting **media** through `DELETE /Items/{itemId}`. §3.6.

## 3. Behaviour

### 3.1 An entry's identifier is the item's identifier

The distinction this feature was written around does not exist, and finding that out is what the
spec-review gate was for.

**`PlaylistItemId` equals the item's own `Id`, on every row.** Measured on the wire, and the
reference's reason is visible in its source: the field the response reads from is a *cache of the
resolved item's id*, filled in the first time the entry is looked up.
`[probe: tools/probe_playlist_move.py, Jellyfin 10.11.11, 2026-08-31]`
`[source: MediaBrowser.Controller/Entities/BaseItem.cs:1797-1802 @ v10.11.11]`

> **This section said something different until it was measured, twice.** It first claimed that a
> track could appear several times, each occurrence a distinct entry. The 2026-08-26 probe
> established that **the reference de-duplicates** (§3.4), and the section was corrected to keep
> entry identity as a real thing addressed by `Move` and `Remove`. The 2026-08-31 probe took the
> remaining half: the identifier those routes address is the item's, so there was never a second
> identifier to be distinct from.
>
> The two findings are one fact seen from two sides. An entry that cannot be named apart from its
> item cannot appear twice in a list addressed by name — so de-duplication is not a policy the
> reference chose, it is the only shape its identifiers allow.

**Atrium reproduces this**, and it is not a free choice: a `PlaylistItemId` that differed from `Id`
would be a value no reference server sends, which is Principle I's first forbidden thing. The
consequence for the plan is that an entry is addressed by the item it references, and a playlist
therefore cannot hold one item twice.

**And the field belongs to one route, not to the item.** Re-measured at T9 against the route that
emits it: equal to `Id` on every row, immediately after `Id` in the wire order, and **absent from
an `/Items` row carrying the same track** — the one property separating a playlist row from a bare
list row (§3.3). So it is a property of a row *in a playlist*, which is why no other response
carries it. `[probe: tools/probe_playlist_read.py, Jellyfin 10.11.11, 2026-09-01]`

> **The reference's can, and that was measured at T7.** Its de-duplication is a lookup in the same
> id cache this section is about, and the cache is empty until an entry has been resolved — so
> **6 of 8 identical add requests put the same item in the playlist twice**
> `[probe: tools/probe_playlist_writes.py, Jellyfin 10.11.11, 2026-08-31]`. What that produces is
> exactly what this section predicts: two rows carrying **one** `PlaylistItemId`, which `Move`
> reorders by moving the first and `Remove` deletes both of at once. Atrium never produces it
> (§3.4, [behaviours §3.18](../../docs/compatibility/behaviours.md)), and the sentence above is
> Atrium's rule rather than a description of the reference.

> **What this costs a client.** The music client's contract asks for a `PlaylistItemId` "distinct
> from the track id" so that it can address duplicates `[client-contract: 2026-08-29, §10]`. Half
> of that is satisfiable and half is not: the field is there on every row, and it is not distinct.
> A reference server does occasionally produce a duplicate — and it is *unaddressable*, by the same
> field, which makes the contract's ask unsatisfiable rather than merely unnecessary.

### 3.2 `POST /Playlists` — `CreatePlaylist`

**Consumers:** music-client.

**Body:** `Name`, `Ids` (initial items, may be empty), `UserId`, `MediaType`, `IsPublic`, `Users`.
`[spec: CreatePlaylistDto]` The same four of them may be sent as query parameters, which take
precedence, and the body itself is optional `[spec: CreatePlaylist]`.

`Users` is the one to notice: it carries the shares that §3.7's third class of writer is made of,
so the sharing model is reachable from this body even though the sharing routes are out of scope.

**Response — 200:** `{"Id": "<32 hex>"}` — the new playlist's item id. `[spec: PlaylistCreationResult]`

A playlist is a first-class item of `Type: Playlist`, so it appears in `/Items` queries filtered by
that type, carries `UserData`, and can be a favourite.

**Errors, as measured** `[probe: tools/probe_playlist_creation.py, Jellyfin 10.11.11, 2026-08-31]`:

| Request | Answer |
|---|---|
| No `Name` property at all | `400`, and the body is the validation shape of [behaviours §1.11](../../docs/compatibility/behaviours.md#111-there-are-four-error-shapes-not-one), keyed on **`$`** — the deserialiser refusing the document, naming the type it was building and the property it did not find |
| `Name` present and **`null`** | `400` as well, and a **different body**: keyed on `Name`, reading `The Name field is required.` The document deserialised and the property's own validator refused the value |
| `Name` empty, or only spaces | **`200`** — the playlist is created, and carries that name |
| No `Name` in the body **and none in the query** | **`500`** in the bare-text shape. Atrium answers `400` in those same bytes and creates nothing — [behaviours §3.19](../../docs/compatibility/behaviours.md) |
| `UserId` naming a user that does not exist | **`200`** — a playlist owned by nobody, which no rule in §3.7 can then reach. Atrium answers `404`, which is the rule §3.7 already applies to that parameter |
| `UserId` naming **another user**, from a non-administrator | `403`, `text/plain`, the 25 bytes — the same helper the add route uses, measured here too `[probe: tools/probe_playlist_visibility.py, Jellyfin 10.11.11, 2026-08-31]` |
| An id in `Ids` that does not exist, with **no** `MediaType` and no resolvable id before it | **`400`**, and the body is the bare-text shape, not the validation one |
| An id in `Ids` that does not exist, after a resolvable id, or with `MediaType` given | `200`; the unknown id is skipped |
| An id in `Ids` of **all zeros**, anywhere in the list | **`400`**, the bare-text shape, and nothing is created — including after a resolvable id, where an ordinary unknown id is skipped. §3.4 carries the rule and where it comes from `[probe: tools/probe_playlist_add_remove.py, Jellyfin 10.11.11, 2026-09-01]` |
| A `MediaType` the reference does not know — `Nonsense` | **`400`**, and the body is the validation shape, keyed on `$` rather than on a parameter name: the value is refused where the body is deserialised, not by a check inside the route `[probe: tools/probe_playlist_media_type.py, Jellyfin 10.11.11, 2026-08-31]` |
| A value in `Ids` or `UserId` that is not an identifier | `400`, the validation shape keyed on the **empty string** with `The supplied value is invalid.` — the binder's refusal, which is a third key on the same route |
| Unauthenticated | `401` |

> **Two of those rows were the opposite way round in this document until they were measured.** It
> claimed `400` for a missing *or empty* name and an unconditional "unknown ids are skipped". The
> empty name creates a playlist, and the skipping is conditional: with no `MediaType`, the
> reference walks the id list to infer one and refuses on the first id that does not resolve,
> stopping as soon as one does. So whether a stale id is fatal depends on where it sits in the list
> and on whether the client named a media type — which the music client does only sometimes.
> `[source: Emby.Server.Implementations/Playlists/PlaylistManager.cs:88-121 @ v10.11.11]`
>
> The two `400`s are also two *shapes* on one route, which is the pattern behaviours §1.11 already
> records for three other routes: the refusal a framework produces and the refusal the handler
> produces do not look alike.
>
> **There are four of them, not two, and T8 is what counted them.** The table above had one
> validation row where the wire has three, told apart by *what* about the body was wrong: a
> required property absent is keyed `$`, the same property present and null is keyed by its own
> name, a value in no vocabulary is keyed `$` again, and a malformed identifier is keyed with the
> empty string. Beside the bare-text row that is four bodies from three layers on one route, and
> none of them names the action parameter that behaviours §1.11 said every body refusal names —
> that row belongs to a **required** body, and this one is optional.
> `[probe: tools/probe_playlist_creation.py, Jellyfin 10.11.11, 2026-08-31]`

**The four properties may be sent as query parameters, and that is not a formality.** `?name=` with
no body at all creates a playlist, and a query value beats the body's on the same property —
measured, not read off the schema. A body that fails to deserialise is refused **before** the query
is consulted, so naming `name` in the query does not rescue a body without `Name`: the two are one
route's two sources for one value, resolved after the body has bound and not instead of it.
`[probe: tools/probe_playlist_creation.py, Jellyfin 10.11.11, 2026-08-31]`

**An unrecognised `MediaType` is refused in the body and dropped in the query.** The same token,
the same route, two answers: `{"MediaType": "Nonsense"}` is the validation `400` above, and
`?mediaType=Nonsense` is ignored and the playlist is created with the inferred value — which is
[behaviours §1.12](../../docs/compatibility/behaviours.md)'s token rule and its first appearance
beside the refusal it is usually contrasted with.
`[probe: tools/probe_playlist_creation.py, Jellyfin 10.11.11, 2026-08-31]`

**What creation decides for the client.** `MediaType` is inferred when it is not given — `Audio`
for a playlist created empty, and the media type of the first resolvable id otherwise. Two
playlists may carry the same `Name`; they are two items.
`[probe: tools/probe_playlist_creation.py, Jellyfin 10.11.11, 2026-08-31]`

> **"The first resolvable id" is not the id's own media type when that id is a container**, which
> T10 measured because creation expands as well (§3.4). A **series** in `Ids` creates a playlist of
> its episodes answering `Video`, where the series' own media type is `Unknown` and the fallback
> for a playlist that settles nothing is `Audio` — so the value comes from what the ids **expanded
> to**. Four containers answer from their kind before their contents are consulted, and they are
> the reason a media type can be decided by a container that expands to nothing: a music album,
> artist or genre answers `Audio` and a `Genre` answers `Video`
> `[source: Emby.Server.Implementations/Playlists/PlaylistManager.cs:95-114 @ v10.11.11]`. A
> container that expands to nothing and answers from neither decides nothing, and the walk moves
> on: an empty folder alone creates an `Audio` playlist and the same folder followed by a film
> creates a `Video` one.
> `[probe: tools/probe_playlist_expansion.py, Jellyfin 10.11.11, 2026-09-01]`

**And creation decides it once.** The value is a fact about *that playlist*, not about playlists
and not about its current contents: a playlist created empty answers `Audio` for ever, including
after a film is added to it, and one created from a film answers `Video` after a track is added.
A `MediaType` in the body outranks the contents outright — a playlist created from a film with
`MediaType: Audio` answers `Audio`. So the value is written at creation and never revised, which
is what makes it something a playlist *carries* rather than something anybody derives on the way
out. `[probe: tools/probe_playlist_media_type.py, Jellyfin 10.11.11, 2026-08-31]`

**The reference builds a playlist as a directory**, and three of its fields say so: `Path` is a
filesystem path under the server's data directory, and `DateCreated` and `DateModified` come from
that directory. The parent is a playlists folder that **does not appear in `/UserViews`**, so this
feature adds no view to a 005 response. §4 says what Atrium does with the three fields.
`[probe: tools/probe_playlist_creation.py, Jellyfin 10.11.11, 2026-08-31]`

### 3.3 `GET /Playlists/{playlistId}/Items` — `GetPlaylistItems`

**Consumers:** music-client, video-client.

Returns the standard list envelope (005 §3.1), with each item carrying its `PlaylistItemId` (§3.1).

**Order is the playlist's order**, and it is the point of the endpoint. **No sort parameter is
accepted**: the route takes `userId`, `startIndex`, `limit`, `fields`, `enableImages`,
`enableUserData`, `imageTypeLimit` and `enableImageTypes`, and nothing else. `[spec: GetPlaylistItems]`

> **This section claimed a `sortBy` that does not exist**, and said it applied "as a view over that
> order". Honouring a parameter the reference ignores is the kind of extension Principle I forbids
> most plainly — a client could discover it — so the correction removes a capability rather than
> adding one.

**The row is a bare list row and one property, and the property sits immediately after `Id`.** The
width is measured rather than assumed, because 005 §3.2 found there is no single item
representation: subtracting the two property sets over the same items gives **`PlaylistItemId` and
nothing else** in one direction and nothing at all in the other, and an `/Items` row carrying the
same track does not have the property at all. So this is the list-row width plus one name, and the
name is on no other route.
`[probe: tools/probe_playlist_read.py, Jellyfin 10.11.11, 2026-09-01]`

The envelope is 005 §3.1's three keys, and **`TotalRecordCount` is taken after filtering and before
paging** — `startIndex=1&limit=2` on a five-entry playlist answers two rows and a count of five,
and `startIndex=99` answers no rows and the same five. That order is the only one that lets a
client page. `fields`, `enableUserData` and `enableImages` are all live on this route.
`[probe: tools/probe_playlist_read.py, Jellyfin 10.11.11, 2026-09-01]`

**`404` for an unknown playlist, and for one the reader may not see — and it is not the shape every
other `404` in this API is.** The body is the message as a **JSON-encoded bare string**,
`"Playlist not found"` under `application/json; charset=utf-8`, 20 bytes: the shape
[behaviours §1.11](../../docs/compatibility/behaviours.md#111-there-are-four-error-shapes-not-one)
records for a controller that refuses *with a message*, measured here on a second route and a
second feature. One body covers three different requests — an identifier that addresses nothing,
an identifier that addresses a real item which is **not** a playlist, and a playlist this reader
may not see — which is what makes a private playlist undisclosable. A `GET /Items/{itemId}` on that
same playlist answers problem details, so the two routes disagree on purpose.
`[probe: tools/probe_playlist_read.py, Jellyfin 10.11.11, 2026-09-01]`
`[probe: tools/probe_playlist_visibility.py, Jellyfin 10.11.11, 2026-09-01]`

> **This section said `404` and stopped there**, and a status is not a shape. Every other `404`
> this project raises from a handler is problem details, so the obvious implementation would have
> shipped a body no reference server sends on the feature's first read route — the same class of
> miss 006 T3 and 008 T6 each found once.

**A fourth request is a different status entirely.** A `playlistId` that is not an identifier is
the model binder's validation `400`, keyed on the parameter and quoting the value back —
`{"playlistId": ["The value 'not-an-identifier' is not valid."]}` — and never reaches the route.
That is the *path* parameter's sentence, which is not the one a malformed identifier inside
§3.2's body gets. `[probe: tools/probe_playlist_read.py, Jellyfin 10.11.11, 2026-09-01]`

The `403` the route declares is reachable only for a playlist stored outside the server's own
playlists folder, because everywhere else the visibility test in front of it has already answered
`404`. `[source: Jellyfin.Api/Controllers/PlaylistsController.cs:520-531 @ v10.11.11]`

**`userId` names the reader, and Atrium diverges on who may name it.** §3.7. A request naming
**nobody** is not that case: it answers `200` for the caller's own view, which is the default
`userId` has everywhere `[probe: tools/probe_playlist_read.py, Jellyfin 10.11.11, 2026-09-01]`.

### 3.4 `POST /Playlists/{playlistId}/Items` — `AddItemToPlaylist`

**Consumers:** music-client.

Appends items, identified by media item id, to the end. `204`.

**Duplicates are silently dropped.** Adding an item already in the playlist adds nothing, and a
single request naming the same item twice adds it once. The reference de-duplicates in two stages —
against the existing entries, then within the incoming batch. `[source: Emby.Server.Implementations/Playlists/PlaylistManager.cs:222-225 @ v10.11.11]`

Measured on both paths, because they are separate code paths: creating a playlist with the same id
twice yields one entry, and adding an id already present yields zero new entries. `[probe: tools/probe_playlist_move.py, Jellyfin 10.11.11, 2026-08-31]`

**Where a repeat lands is measured too, and it is nowhere: the entry already there keeps its
position.** Re-adding the first of three entries leaves the order untouched, and `Ids` naming A B A
on creation creates A B — the first occurrence, not the last. A one-entry playlist cannot tell
"dropped" from "removed and appended", which is the only shape the paragraph above was measured on.
`[probe: tools/probe_playlist_writes.py, Jellyfin 10.11.11, 2026-08-31]`

**But only the second of the two stages is reliable, and this is 009's fourth divergence.** The
stage that compares against the *existing* entries reads a cache filled the first time an entry is
resolved, and an entry whose cache is cold is invisible to it: **6 of 8 identical requests** added
an item the playlist already held, on the same server, seconds apart. The stage that de-duplicates
*within* the batch never missed. `[probe: tools/probe_playlist_writes.py, Jellyfin 10.11.11,
2026-08-31]`

**Atrium de-duplicates always**, which is a divergence only from the failing side of a coin flip:
the argument, and why a client cannot have built on the duplicate it produces, is in
[behaviours §3.18](../../docs/compatibility/behaviours.md). §3.1 is why de-duplication is not a
policy choice here in the first place.

**Adding a container adds its children, and every kind of container.** Measured: an album adds its
tracks **in the album's own order**, an artist adds their tracks, a series and a season add their
episodes, and a collection adds its films. The container itself never becomes an entry.
`[probe: tools/probe_playlist_expansion.py, Jellyfin 10.11.11, 2026-08-31]`

> **"Every kind" is wider than the five this paragraph names, and three more were measured at
> T10.** A **plain folder** expands, **the library root itself** expands — twenty-one entries from
> a view listing three children, because the expansion is recursive — and **another playlist**
> expands to its own entries. Anything that holds something is a container; the five kinds above
> are examples of the rule and not the rule. A container that holds nothing adds nothing and says
> so with the same `204`.
> `[probe: tools/probe_playlist_expansion.py, Jellyfin 10.11.11, 2026-09-01]`
>
> **And the expansion happens where the container was named.** A request naming a film, an album
> and a second film lands the album's tracks *between* the two films, so a client that builds a
> playlist in one request gets the order it asked for. No single-id request can tell that from an
> expansion appended at the end, which is why it was measured separately.
>
> **The orders are two, not one.** A folder answers in the folder's own order — the order
> `/Items?parentId=` gives, which is what makes AC-7's "the album's own order" true. An **artist**
> answers a different query, ordered by album artist, then album, then sort name, over the tracks
> that artist is *credited* on: forty-two rows where a walk down the item tree gives forty.

`404` for an unknown playlist, in §3.3's twenty bytes — an identifier that addresses nothing and a
real item that is not a playlist are one body here too, so no write route discloses a playlist a
caller may not see. A **malformed** playlist id is the validation `400`, as on the read.
`[probe: tools/probe_playlist_add_remove.py, Jellyfin 10.11.11, 2026-09-01]`

Unknown item ids are skipped — unconditionally here, unlike §3.2's creation path: measured in
first, last and middle position, and the batch's other ids are added. A **malformed** id in `ids`
is dropped in silence, where the same value in §3.2's *body* is a validation `400` — a query list
binds token by token and a body property binds as a whole.

> **One identifier is refused rather than skipped, and it is the one a client sends by accident.**
> An id of all zeros — `Guid.Empty`, what a default-initialised field serialises to — answers
> `400` in the bare-text shape and adds **nothing**, beside a resolvable id included; and on §3.2's
> creation path it is refused even in the position where an ordinary unknown id is skipped. The
> reference rejects an empty identifier in its item lookup rather than failing to find it
> `[source: Emby.Server.Implementations/Library/LibraryManager.cs:1357-1362 @ v10.11.11]`, so it is
> one rule on both routes rather than a property of either
> `[probe: tools/probe_playlist_add_remove.py, Jellyfin 10.11.11, 2026-09-01]`. It is **not** a
> refusal on the removal below, which looks nothing up.

A request naming no ids at all is `204` and changes nothing.

**The caller who may not edit is refused with `403`, no body and no content type** — §3.7's
*May edit* column, and the shape §3.8 measures rather than the 25 bytes of AC-19. It is reachable
only for a playlist the caller can **see**: an administrator who is none of §3.7's three classes is
answered `404` first, because the lookup in front of the editing test filters by owner, share and
`IsPublic` with no administrator branch
`[source: Emby.Server.Implementations/Playlists/PlaylistManager.cs:62-78 @ v10.11.11]`.

### 3.5 Removing and reordering

**`DELETE /Playlists/{playlistId}/Items`** removes entries by entry id, several at once, named in
`entryIds`. `204`. Removing an entry id that is not in the playlist is `204`, not an error —
clients retry, and a retry after a successful removal must not fail. `[spec: RemoveItemFromPlaylist]`

**Measured over every class of identifier, and all of them are that same `204`**: an id that
addresses nothing, a malformed one, an id of all zeros — which the add route refuses — and no
`entryIds` parameter at all. The route looks nothing up, which is why the one refusal §3.4 has
cannot happen here. The surviving entries keep their order.
`[probe: tools/probe_playlist_add_remove.py, Jellyfin 10.11.11, 2026-09-01]`

**This route takes no `userId`**, where the add route beside it does `[spec: RemoveItemFromPlaylist]`
— it reads the caller's own identity, and declaring the parameter here would be a lever no
reference server has.

**And a malformed *playlist* id answers differently on the two routes of one path.** `POST` is the
model binder's validation `400`; `DELETE` is a **`500`** in the bare-text shape, because that
action takes the segment as text and parses it itself
`[source: Jellyfin.Api/Controllers/PlaylistsController.cs:447-456 @ v10.11.11]`. Atrium answers the
validation `400` on both — [behaviours §3.19](../../docs/compatibility/behaviours.md), the same
argument as the two refusals already recorded there
`[probe: tools/probe_playlist_add_remove.py, Jellyfin 10.11.11, 2026-09-01]`. The **move** below is
a third: it parses the segment itself as this route does, so it answers the same `500`, and Atrium
answers the same validation `400`
`[probe: tools/probe_playlist_move.py, Jellyfin 10.11.11, 2026-09-01]`.

**`POST /Playlists/{playlistId}/Items/{itemId}/Move/{newIndex}`** moves one entry to an absolute
index. `204`.

**Indices are zero-based and name the entry's position in the list _after_ the move.** The entry is
removed first, then inserted so that it ends up at exactly `newIndex` in the resulting list.

Measured with the only case that distinguishes the two readings — a downward move, index 0 to
index 3 on `[A B C D E]`:

| Reading | Result |
|---|---|
| **Post-removal / final index** — what the reference does | `B C D A E` |
| Pre-removal — insert before whatever was at index 3 | `B C A D E` |

`[probe: tools/probe_playlist_move.py, Jellyfin 10.11.11, 2026-08-26, re-measured 2026-08-31]`

Upward moves are identical under both readings, which is why a client author can build against the
wrong one and not notice until a user drags something down. This specification asserted the wrong
reading until it was measured; the probe existed precisely because getting it wrong is invisible
until it is expensive.

**The boundaries, measured on a five-entry playlist**
`[probe: tools/probe_playlist_move.py, Jellyfin 10.11.11, 2026-08-31]`:

| Case | The reference answers | Atrium answers |
|---|---|---|
| `newIndex` equal to the entry count | `204`, the entry goes last | The same |
| `newIndex` greater than the entry count | **`500`**, and nothing moves | `400`, and nothing moves — [behaviours §3.15](../../docs/compatibility/behaviours.md) |
| `newIndex` negative | **`204`**, and the entry moves to index 1 | `400`, and nothing moves — the same entry |
| Moving to its current index | `204`, nothing changes | The same |
| Entry id not in the playlist, index in range | **`204`**, nothing changes | The same |
| Entry id not in the playlist, index out of range | **`500`** | `400` — the index is judged before the entry is looked up, in both servers |
| Entry id that is not an identifier at all, or is all zeros | **`204`** in range, **`500`** out of it | The same two answers — it is an entry the playlist does not hold, and nothing more |
| A caller who may not edit, with the index out of range | **`403`**, body-less | The same — the caller is judged before the index |
| A malformed playlist id | **`500`** | `400` in the validation shape — [behaviours §3.19](../../docs/compatibility/behaviours.md) |
| `newIndex` that is not a number | `400`, keyed `newIndex` | The same |

> **This table had four rows and one of them was right.** It claimed a clamp for anything past the
> end, `400` for a negative index and `404` for an absent entry; only *moving to its current index*
> survived. What is really there is a clamp exactly one position wide, a negative index that moves
> the entry rather than refusing it, and an absent entry that is a silent success — and, on the two
> rows nobody had thought to write, an unhandled failure, which is the divergence recorded beside
> them.

**The index is judged against what the caller can see**, and so is the position the entry lands at.
For an owner the two lists are one and the table above is the whole story. For a reader who is
shown fewer entries (§3.7) the reference is off by one on every downward move — it takes the
neighbour's position in the stored order *before* removing the entry — and it moves entries that
reader was never shown. Atrium does neither: the entry ends up at `newIndex` of the list that
reader was given, and an entry omitted from it is answered as an entry that is not in the playlist
— `204`, nothing changes. Neither difference is reachable against a reference server, because the
entries it hides are hidden by a parental-rating check and never by library access
`[source: Emby.Server.Implementations/Playlists/PlaylistManager.cs:289-345 @ v10.11.11]`; both
belong to the divergence §3.7 already carries.

**The three path segments accept three different things, and one identifier has two spellings that
are not interchangeable.** The playlist id is parsed, so `Move` addresses the same playlist whether
it is written plain or dashed. The **entry** id is not parsed at all: it is compared as
text against the plain 32-character spelling of each entry, so an upper-case entry id moves the
entry and a **dashed or braced one moves nothing** — measured on all four spellings, and the reason
an entry id that is not an identifier at all is a silent `204` rather than a refusal
`[probe: tools/probe_playlist_move.py, Jellyfin 10.11.11, 2026-09-01]`
`[source: Emby.Server.Implementations/Playlists/PlaylistManager.cs:308-323 @ v10.11.11]`. Atrium
reproduces both, which means this is the one route in the feature that must **not** canonicalise
the identifier it is given.

The index is a number in the path, so a `newIndex` that is not one never reaches the arithmetic:
it is the validation `400` keyed `newIndex`, in the same problem-details shape §3.3's malformed
playlist id answers, and it is parity.

**The refusals happen in the reference's own order, and it is not the arithmetic's.** A playlist
this caller cannot see is `404` in §3.3's twenty bytes; a caller who may read it and may not edit
it is §3.7's body-less `403` — *even when the index is one the reference crashes on*, measured, so
the `400` this feature makes its own is reachable only by a caller who may edit
`[probe: tools/probe_playlist_shares.py, Jellyfin 10.11.11, 2026-09-01]`. Only then is the index
judged, and only then the entry looked up.

Entry identifiers survive a move: the row keeps its `PlaylistItemId` — which, by §3.1, it could
hardly fail to do.

**Every pair is measured, not modelled.** All thirty of source × `newIndex` 0–5 on `[A B C D E]`,
not the one pair that distinguishes the two readings: the post-removal reading reproduces the
reference on every one of them, and the one-position clamp is a property of every source rather
than of the one the boundary table asked about.
`[probe: tools/probe_playlist_move.py, Jellyfin 10.11.11, 2026-08-31]`

### 3.6 Deleting a playlist

Through `DELETE /Items/{itemId}`, which is why that route is in v1 at all.

**This route is dangerous, and its danger is asymmetric.** For a playlist it removes server-side
state the user created. For a media item it can delete the user's file.

v1's rule: **`DELETE /Items/{itemId}` succeeds only for a playlist.** Deleting a movie, an episode
or a track answers `403`, regardless of the user's `EnableContentDeletion` policy — and so does
everything else this route can name, including the rows no file backs. A genre or an artist is
rebuilt by the next scan, so deleting one would be a deletion that does not stick, which is the
plausible-looking stub Principle VI forbids rather than the harmless case it looks like.

> This is a **deliberate divergence**, and unlike the others in this project it is not argued from
> "no client can tell" — a client *can* tell, by deleting a movie and finding it still there.
>
> It is argued from consequence. v1 has no undo, no trash, and no confirmation flow of its own; it
> would be trusting a client's confirmation dialog with an irreversible operation on files the user
> may not have backed up. The cost of the divergence is a client's delete button failing on media
> and working on playlists, and the gate measured that cost at zero for both analysed clients:
> `DeleteItem` is named by one of them and only ever for playlists.
>
> **It is observable in a second place, measured at T12 and narrower than it looks.** Where the
> caller is *not* entitled to delete content, the reference refuses too — `401` with the same 21
> bytes it refuses a playlist with — so Atrium's `403` differs from a refusal rather than from a
> deletion for that caller `[probe: tools/probe_item_deletion.py, Jellyfin 10.11.11, 2026-09-01]`.
> One status for every media request is deliberate: v1 does not enforce `EnableContentDeletion` on
> any route, so splitting the answer by it would make a refusal's *shape* depend on a permission
> nothing else here honours.
>
> Media deletion is revisited when there is a trash with a retention window to put things in.
> Recorded in [behaviours §4.3](../../docs/compatibility/behaviours.md#43-delete-itemsitemid-refuses-to-delete-media).

**A caller who may not delete the playlist is refused with `401`, not `403`** — measured on the
owner-and-administrator rule below, with the body `"Unauthorized access"`, 21 bytes, under
`application/json; charset=utf-8`.
`[probe: tools/probe_playlist_visibility.py, Jellyfin 10.11.11, 2026-08-31]`
`[probe: tools/probe_item_deletion.py, Jellyfin 10.11.11, 2026-09-01]`
`[source: Jellyfin.Api/Controllers/LibraryController.cs:374-383 @ v10.11.11]` This document said
`403` until the gate measured it. A deletion that succeeds is `204` with no body and no content
type.

**And that refusal is what a caller who may not even *read* the playlist gets.** This route applies
**no visibility test to a playlist**: a caller answered `404` by the read route and by
`GET /Items/{itemId}` for a private playlist is answered `401` here, and so learns that it exists.
So on this one route a `404` means *no such item* and nothing else — for a playlist. Media is the
other way round, and measured beside it: an item in a library the caller cannot open is `404`
before any permission is consulted. `404` is therefore an unknown identifier, or an item this
caller could not have seen in the first place
`[probe: tools/probe_item_deletion.py, Jellyfin 10.11.11, 2026-09-01]`.

**Two more identifiers, and neither is a `404`.** One that is not an identifier at all is the
binder's validation `400` naming `itemId`; one of **all zeros** is the fixed 25-byte `text/plain`
`400`, the same refusal §3.4's write routes make, because the reference rejects an empty identifier
in the item lookup this route shares with them
`[probe: tools/probe_item_deletion.py, Jellyfin 10.11.11, 2026-09-01]`.

Deletion is the **one** operation an administrator may perform on a playlist they do not own — and
unlike every refusal in §3.7, it is **not** conditional on their being able to see it: the
administrator who is answered `404` by every other route on a private playlist deletes it here.
`[source: MediaBrowser.Controller/Playlists/Playlist.cs:261-264 @ v10.11.11]`
`[probe: tools/probe_item_deletion.py, Jellyfin 10.11.11, 2026-09-01]`

### 3.7 Ownership and visibility

A playlist has an owner, and three classes of caller can reach it:

| Class | May read | May edit | May delete |
|---|---|---|---|
| The owner | yes | yes | yes |
| A user the playlist is shared with, `CanEdit` | yes | yes | no |
| A user the playlist is shared with, without `CanEdit` | yes | no | no |
| Anybody, when `IsPublic` | yes | no | no |
| An administrator who is none of the above | no | **no** | **yes** |

Shares are set through the create body's `Users` (§3.2), which is what puts them in scope. Measured:
a shared reader with `CanEdit` moved an entry and the owner saw the new order.
`[probe: tools/probe_playlist_visibility.py, Jellyfin 10.11.11, 2026-08-31]`

**Every "no" in the *May edit* column is a `403` with no content type and no body**, and the two
rows that had never been produced are now measured: a share **without** `CanEdit` is stored by the
create body and is exactly a reader who is refused the move, and a public playlist's reader is
refused the same way `[probe: tools/probe_playlist_shares.py, Jellyfin 10.11.11, 2026-08-31]`. It
is the **body-less** shape of the two — §3.8's, not the 25 bytes AC-19 asserts — because a refusal
returned as a result carries nothing while a refusal thrown as an exception is rendered by the
error middleware `[source: Jellyfin.Api/Controllers/PlaylistsController.cs:421-427 @ v10.11.11]`
`[source: Jellyfin.Api/Helpers/RequestHelpers.cs:77-81 @ v10.11.11]`. So the split
[behaviours §1.11](../../docs/compatibility/behaviours.md#111-there-are-four-error-shapes-not-one)
draws between a controller and a policy is really between an exception and a returned result, and
both happen inside the same action.

> **The administrator row is the one this document had wrong.** It said administrators may "modify
> or delete". Every editing route tests owner-or-share and has no administrator branch; deletion is
> the only route that has one.
> `[source: Jellyfin.Api/Controllers/PlaylistsController.cs:132-134, 422-424, 461-463 @ v10.11.11]`
>
> **And that row's "yes" is unconditional, where every "no" beside it is not.** The editing routes
> reach their refusal only on a playlist the caller can see — a private one is `404` at the lookup
> in front of it — but the deletion route has no such lookup, so an administrator deletes a private
> playlist they are answered `404` for everywhere else
> `[probe: tools/probe_item_deletion.py, Jellyfin 10.11.11, 2026-09-01]`. The three "may read"
> cells are about the *read* routes, and §3.6 is not one of them.

A non-public playlist does not appear in another user's `/Items`, and a direct fetch answers `404`.
`[probe: tools/probe_playlist_visibility.py, Jellyfin 10.11.11, 2026-08-31]`

**Two disclosures, and Atrium diverges on both.**

**The reader can be named.** `GET /Playlists/{playlistId}/Items` takes the identity it checks
permissions against from the `userId` query parameter, with no test that the caller is entitled to
name it — so a non-administrator reads any private playlist by naming its owner. Measured: `200`,
with the entries, to a restricted user who is answered `404` by every other route on that playlist.
The **same parameter on the same controller's add route answers `403`**, through the check this one
does not make. `[probe: tools/probe_playlist_visibility.py, Jellyfin 10.11.11, 2026-08-31]`

**Atrium honours `userId` for administrators and refuses it otherwise**, which is the rule the
reference applies on its own write routes. [behaviours §3.16](../../docs/compatibility/behaviours.md)
carries the argument; the short form is that a reference that refuses on one route what it permits
on another cannot have taught a client to depend on the permissive one.

**The refusal is the reference's own, bytes included.** It is `403` carrying the 25-byte
`text/plain` body `Error processing request.`, with no `charset` on the content type — the shape
[behaviours §1.11](../../docs/compatibility/behaviours.md#111-there-are-four-error-shapes-not-one)
gives a controller that refuses a request itself, measured here for a permission refusal for the
first time `[probe: tools/probe_playlist_visibility.py, Jellyfin 10.11.11, 2026-08-31]`. This
server answered an **empty** `403` for every refusal of that class until 009 — a body no reference
server sends, on a route 005 already ships — and 009 corrects it where it is decided rather than on
its own two routes alone.

**But `403` is two shapes, and only this one is a controller's.** The refusal §3.8 measures — the
elevated rename controller turning away a non-administrator — carries **no content type and no body
at all**, because an authorization policy answers it before any controller runs. Both are `403` and
the bytes are the whole difference. So the correction above belongs to the refusals a route decides
for itself, and a route refused by policy keeps the empty shape
`[probe: tools/probe_playlist_visibility.py, Jellyfin 10.11.11, 2026-08-31]`.

> **The content type was never measured until this was implemented.** The probe cited above had
> printed forty bytes of each refusal's body and none of its headers, which cannot tell an empty
> body from a body-less refusal, and cannot see the `charset` that separates this shape from every
> JSON one. Both cells are measured now, and they are what turned one shape into two.

**An entry the reader has no access to is shown anyway.** The reference filters a playlist's
entries through a parental-rating and tag check, which has nothing to do with which libraries a
user may open. Measured: a reader restricted to one library, who can list **zero** items of another
and is answered `404` fetching one directly, is handed those items as playlist rows, counted in
`TotalRecordCount`. `[probe: tools/probe_playlist_visibility.py, Jellyfin 10.11.11, 2026-08-31]`
`[source: MediaBrowser.Controller/Entities/BaseItem.cs:1736-1741 @ v10.11.11]`

**Atrium omits entries the reader cannot reach**, and the remaining entries keep their order and
their entry ids. [behaviours §3.17](../../docs/compatibility/behaviours.md) carries the argument.
The count follows the omission: `TotalRecordCount` is what the reader may see, which is what the
reference does too, on the criterion it happens to be filtering.

> **This document already asserted the omission**, as though it were the reference's behaviour, and
> the gate found it was Atrium's alone. What was a description is now a divergence with an
> argument, which is the difference between a specification and a hope.

### 3.8 `POST /Items/{itemId}` — `UpdateItem`, for renaming a playlist

**Consumers:** music-client.

The client renames a playlist by fetching the item, changing `Name`, and posting the item back
`[client-contract: 2026-08-29, §10]`. `204` on success.

**The route is administrator-only**, and that is the whole of what this section had to measure. The
reference declares the controller elevated, so:

| Caller | Answer |
|---|---|
| An administrator, on any playlist | `204`, and the name changes |
| The playlist's own owner, not an administrator | **`403`**, with an empty body **and no content type** — an authorization policy's refusal, not a controller's, which is why it is not §3.7's 25 bytes |

`[probe: tools/probe_playlist_rename.py, Jellyfin 10.11.11, 2026-08-31]`
`[source: Jellyfin.Api/Controllers/ItemUpdateController.cs @ v10.11.11]`

**And the refusal comes before everything else**, which is what an elevated *controller* means on
the wire: a caller who is not an administrator is answered that same empty `403` for an identifier
that names nothing and for a path segment that is not an identifier at all, where an administrator
sending those two requests is answered `404` and the binder's `400`
`[probe: tools/probe_playlist_rename.py, Jellyfin 10.11.11, 2026-09-01]`. Nothing about an item
reaches a caller who may not touch it, not even whether it exists.

**The directory does not move.** An administrator's rename changes the item's `Name` and leaves the
path it was created under, so the two disagree from the first rename onwards
`[probe: tools/probe_playlist_rename.py, Jellyfin 10.11.11, 2026-08-31]` — which is one more
reason §4 keeps `Path` out of what Atrium promises.

**The body is the whole item, and three of its properties are load-bearing.** The client fetches
the item and posts it back, so what arrives carries every property the read emitted — thirty-nine
on a playlist. Dropping each of them in turn, the reference refuses exactly three: `Genres`, `Tags`
and `ProviderIds`, absent or `null`, each a `400` in the 25-byte controller shape; a body of those
three and a `Name` is accepted, and the other thirty-five change nothing by their absence
`[probe: tools/probe_playlist_rename.py, Jellyfin 10.11.11, 2026-09-01]`. So a client that posted
`{"Name": …}` alone would be refused by a stock reference server, and Atrium refuses it identically.

**It is not a rename.** The same measurement applied `Overview`, `ForcedSortName`, `OfficialRating`,
`CustomRating`, `ProductionYear`, `Genres` and `Tags` from the posted body, while `Path` and
`IsFolder` were computed and ignored. **Atrium applies `Name` and nothing else**, which is a
narrowing recorded as a gap in [behaviours §5](../../docs/compatibility/behaviours.md) rather than
described as parity: v1 has a consumer for none of the other seven and nowhere to put them that the
next scan would not overwrite (004 T10).

**A body with no `Name` is a `204` that erases the name**, on the reference — absent or `null`,
the playlist comes back with no `Name` property at all. Atrium refuses that request with the same
`400` and the same 25 bytes an incomplete body gets, so the status is the whole difference; it is
009's sixth divergence, [behaviours §3.21](../../docs/compatibility/behaviours.md), and it stands
beside §3.19's four for the same reason. An **empty or blank** name, by contrast, is applied as
sent on both servers, which is §3.2's finding on the creation route.

| The identifier in the path | Answer |
|---|---|
| Plain, dashed, braced or upper-case | Addresses the playlist — the segment is parsed |
| Not an identifier | The validation `400`, keyed `itemId` |
| All zeros | The bare-text `400`, as on every route that resolves an identifier |
| Well-formed and unknown | `404`, problem details — **before** the body is looked at |

Measured on this method rather than inherited from the `DELETE` that shares the path
`[probe: tools/probe_playlist_rename.py, Jellyfin 10.11.11, 2026-09-01]`.

`404` for an unknown item.

> **v1 routes this operation for playlists.** Its reference counterpart edits every field of every
> item type, and v1 has no consumer for any of that (Principle VI): the music client posts it at
> playlists and nothing else does. An item type nobody names is not routed here rather than being
> answered with a lie.
>
> **What the client still cannot do.** A non-administrator using that client cannot rename their own
> playlist — not against Atrium, and not against a stock reference server either, which is where
> the refusal comes from. The route that would let them is `UpdatePlaylist`, excluded by §2 for
> having no named consumer. This is a gap in the client's own feature, reproduced exactly, and it
> is recorded in [behaviours §5](../../docs/compatibility/behaviours.md) rather than fixed here.

## 4. Data the feature owns

| State | Observable as | Lifetime |
|---|---|---|
| Playlists | `Type: Playlist` items in `/Items` | Until deleted |
| Entries and their order | `GET /Playlists/{id}/Items` | Until modified |
| Entry identity | `PlaylistItemId`, equal to the item's `Id` (§3.1) | For the life of the entry |
| Ownership, shares and visibility | Whether another user sees the playlist, and what they may do | With the playlist |

Playlists are the **only** structural state in v1 that does not come from the filesystem, and
therefore the only thing in the server's store that cannot be rebuilt by a rescan. That makes them
the only thing whose loss is unrecoverable, and the plan has to treat them accordingly.

> **A playlist's media type is a fact about the row, and one query parameter reads it.** Every
> other item's media type follows from its kind — a film is `Video`, a track is `Audio`, a
> container is `Unknown` — and `mediaTypes=` is answered by turning the asked-for value back into
> the kinds that answer it. A playlist breaks that: two playlists of the same kind answer
> differently, and the reference filters them by the row it stored. Measured, `mediaTypes=Audio`
> over playlists returns the audio one and not the video one, and `mediaTypes=Video` the reverse
> `[probe: tools/probe_playlist_media_type.py, Jellyfin 10.11.11, 2026-08-31]`. Answering that
> parameter from the type alone would claim every playlist for `Audio` and none for `Video`.
> **Decided at T6, and it is the clause rather than the gap.** The listing answers `mediaTypes=`
> from the stored value for a playlist and from the kind for everything else. It was left open on
> the argument that no analysed client sends the pair — but the same read decides the `MediaType`
> a listed playlist *reports*, and a listing that filtered one way while reporting the other would
> disagree with itself on the wire, which no client has to send anything unusual to see.
>
> **And the stored value has a third answer.** Measured on a stock reference: over eight
> playlists, `mediaTypes=Audio` returns five, `mediaTypes=Video` two and `mediaTypes=Unknown`
> **one** `[probe: tools/probe_playlist_media_type.py, Jellyfin 10.11.11, 2026-08-31]`. Creation
> cannot produce that value — an id list that resolves to nothing falls back to `Audio`
> `[source: Emby.Server.Implementations/Playlists/PlaylistManager.cs:124-126 @ v10.11.11]` — but a
> playlist the reference builds from a directory is given no media type at all
> `[source: Emby.Server.Implementations/Library/Resolvers/PlaylistResolver.cs:40-45 @ v10.11.11]`,
> and its own file cannot restore one, because an unknown media type is the single value the
> saver omits
> `[source: MediaBrowser.LocalMetadata/Savers/PlaylistXmlSaver.cs:52-55 @ v10.11.11]`. Atrium
> builds no playlist from a directory, so the value it stores is one of the other two — but the
> parameter is answered by comparing what the playlist *holds*, not by naming the two values it is
> expected to hold.

> **The reference's playlists do come from the filesystem**, and three fields carry it to the wire:
> `Path`, `DateCreated` and `DateModified` (§3.2). Atrium has no directory to report, and inventing
> a path that no file backs would be a worse answer than the honest one — so a playlist item here
> carries no `Path`, and its two dates are the ones its own store records. This is the one place
> where 009 cannot be byte-identical, it is visible only to a client that asks for `Path` by name,
> and it is recorded in [behaviours §5](../../docs/compatibility/behaviours.md).

## 5. Acceptance criteria

1. `POST /Playlists` returns an id; the playlist then appears in `/Items?IncludeItemTypes=Playlist`.
2. Creating with an empty `Name` **succeeds**, and the playlist carries the empty name; creating
   with no `Name` property at all answers `400` in the validation shape.
3. Creating with an unknown id **after** a resolvable one succeeds and skips it; creating with an
   unknown id **first** and no `MediaType` answers `400`; supplying `MediaType` makes it succeed.
4. Every item from `GET /Playlists/{id}/Items` carries a `PlaylistItemId`, and it is equal to that
   item's `Id`.
5. Adding an item already in the playlist adds nothing and moves nothing, and one request naming it
   twice adds it once — on both the creation and the addition paths, and **every time**, where the
   reference manages it only when its own id cache is warm (§3.4).
6. Removing by entry id removes exactly that row; removing an entry id that is not present
   answers `204`.
7. Adding an album adds its tracks in the album's own order, and the album itself is not an entry;
   adding a series adds its episodes; and so does **every** other container — a season, an artist,
   a plain folder, a library root and another playlist — with the expansion landing where the
   container was named in the batch rather than after it. A container holding nothing adds nothing
   and still answers `204`.
8. `GET /Playlists/{id}/Items` accepts no sort parameter, and the order it returns is the
   playlist's.
9. Moving an entry from index 0 to index 3 on a five-entry playlist produces `B C D A E` — the
   entry ends up **at** `newIndex` in the resulting list — and every entry keeps its
   `PlaylistItemId`.
10. `newIndex` equal to the entry count puts the entry last; `newIndex` past the count answers
    `400` and moves nothing; a negative `newIndex` answers `400` and moves nothing.
11. Moving an entry id that is not in the playlist answers `204` and changes nothing when the
    index is in range, and `400` when it is not — the index is judged before the entry is looked
    up.
12. `DELETE /Items/{id}` on a playlist deletes it with `204` and no body; on a movie it answers
    `403` and the file remains on disk, and so does anything else that is not a playlist; on a
    playlist the caller may not delete — including one they may not read — it answers `401` with
    the 21-byte body `"Unauthorized access"`. An unknown identifier, and an item this caller could
    not see anyway, answer `404`.
13. An administrator who does not own a playlist may delete it and may not edit it: `Move`,
    `Add` and `Remove` are refused with `403`, **no body and no content type** — on a playlist that
    administrator can *see*. On one they cannot, the same three routes answer `404` before the
    editing test is reached, which is the same lookup §3.3's `404` comes from. The deletion carries
    no such condition: on that same invisible playlist it answers `204` (§3.6).
14. A user shared with `CanEdit` through the create body may reorder the playlist; one shared
    without it is refused in that same shape, and so is a public playlist's reader.
15. A non-public playlist is invisible to another non-administrator user in `/Items` and answers
    `404` on direct fetch. Naming its owner in `userId` does not reach it either — but that
    request is **refused**, with AC-19's `403`, before any playlist is looked up, where the
    reference answers `200` with the entries (§3.7). *This criterion said the named-owner request
    was part of the `404` until T14: one request cannot have two shapes, and the two criteria
    beside it — AC-16 and AC-19 — assert the one the wire has.*
16. `userId` naming another user is honoured for an administrator and refused otherwise.
17. A playlist containing an item the reader cannot reach returns the remaining entries, in order,
    with unchanged entry ids, and a `TotalRecordCount` that counts only those — and that reader's
    `Move` indexes the list they were given: the entry lands at `newIndex` of it, and naming the
    omitted entry answers `204` and changes nothing.
18. An administrator renames a playlist through `POST /Items/{id}`; its non-administrator owner is
    answered `403` **with no body and no content type**, which is the policy shape of §3.7 and not
    the sentence AC-19 asserts — and that owner meets the same `403` for an identifier naming
    nothing and for one that is not an identifier at all, where the administrator meets `404` and
    the validation `400`. The body is the reference's: one omitting `Genres`, `Tags` or
    `ProviderIds` is the 25-byte `400`, and one omitting `Name` is that same `400` where the
    reference answers `204` and erases the name.
19. A refusal for a caller naming another user carries `403` with the reference's 25-byte
    `text/plain` body, content type included and `charset` absent — on this feature's routes and on
    the 005 route that shares the rule. The two `403`s of AC-18 and AC-19 are different bytes on
    purpose.
20. Playlist state survives a full library rescan.

## 6. Conformance

| Endpoint | Level | How it is proven |
|---|---|---|
| `POST /Playlists` | **L2** | Golden response; creation then query; the two `400` shapes and the empty name (AC-2, AC-3) |
| `GET /Playlists/{id}/Items` | **L2** | Golden response including `PlaylistItemId`; order assertions; the `userId` divergence (AC-15, AC-16) |
| `POST`/`DELETE .../Items` | **L2** | Duplicate and idempotency cases (AC-5, AC-6); container expansion (AC-7) |
| `.../Move/{newIndex}` | **L2** | Table-driven over source × target, including the boundaries (AC-9 to AC-11) |
| `DELETE /Items/{itemId}` | **L2** | Playlist path plus the media refusal, with an on-disk assertion (AC-12) |
| `POST /Items/{itemId}` | **L2** | Rename by an administrator, refusal for a non-administrator owner, the four identifier classes, the three properties the body may not omit and the one it may not (AC-18) |
| The shared refusal | **L2** | The `403` body, asserted as bytes **and content type** on a 009 route and on `/Items?userId=` (AC-19) — and beside it AC-18's, which is the other shape |

The move test is table-driven because off-by-one errors in reordering pass every hand-written case
and fail the one nobody wrote. Every (source, target) pair on a five-entry playlist is 25 rows and
catches all of them. **Two tables, not one:** the reference's move arithmetic indexes the entries
the caller can see while inserting into the entries the playlist has, so a playlist with one
unreachable entry is a second matrix — and under §3.7's divergence, the one Atrium has to get right
is the filtered view its own readers get.

## 7. Open questions

None. All six were answered at the spec-review gate on 2026-08-31.

### Resolved

| # | Question | Answer | Resolved by |
|---|---|---|---|
| OQ-1 | Does the reference interpret `newIndex` pre- or post-removal? | **Post-removal.** The entry ends up at `newIndex` in the resulting list. §3.5 and AC-9 said the opposite and are corrected | `tools/probe_playlist_move.py`, 2026-08-26, re-measured 2026-08-31 |
| OQ-2 | Does the reference de-duplicate on add? | **Yes, and on create too** — two stages. §3.1 and §3.4 are corrected, and §3.1 now carries the reason: an entry has no identifier of its own to be duplicated under | `tools/probe_playlist_move.py`, 2026-08-26 |
| OQ-3 | Does adding a container expand it, or add the container itself? | **It expands**, for every container kind — album, artist, series, season and collection — and an album arrives in the album's own order. AC-7 holds | `tools/probe_playlist_expansion.py`, 2026-08-31 |
| OQ-4 | What the reference does with entries the reader cannot see | **Nothing.** The filter it applies is a parental-rating check, so entries from libraries the reader cannot open are returned and counted. §3.7's omission rule is now Atrium's divergence rather than a description | `tools/probe_playlist_visibility.py`, 2026-08-31 |
| OQ-5 | Whether any client relies on `DELETE /Items/{itemId}` deleting media | **No.** The surface names one consumer for the route, its contract uses it only for playlists, and the other client's operation table does not carry it. The §3.6 divergence costs both analysed clients nothing | The two client contracts and `surface.yaml`, 2026-08-31 |
| OQ-6 | Whether `newIndex` beyond the end clamps or errors, and what a negative index does | **Both, and neither as written**: a clamp one position wide, then `500`; a negative index moves the entry to index 1. §3.5 carries the measured table | `tools/probe_playlist_move.py`, 2026-08-31 |

### Answered without having been asked

| Finding | Where it landed |
|---|---|
| `PlaylistItemId` is the item's own `Id` | §3.1, AC-4 — and it is the finding the feature was least prepared for |
| `GET /Playlists/{id}/Items` accepts no sort parameter | §3.3, AC-8 |
| `userId` names the reader, unchecked | §3.7, behaviours §3.16, AC-15 and AC-16 |
| An empty `Name` creates a playlist | §3.2, AC-2 |
| An unknown id is fatal to creation under one condition | §3.2, AC-3 |
| Refusing a deletion is `401`, not `403` | §3.6, AC-12 |
| Administrators may delete a playlist and may not edit one | §3.7, AC-13 |
| Shares are reachable from the create body | §3.2, §3.7, AC-14 |
| The rename route is administrator-only | §3.8, AC-18 |
| A playlist is a directory, and a rename does not move it | §3.2, §3.8, §4 |
| The playlists folder is not a view | §3.2 — 009 adds nothing to a 005 response |
| A playlist's `MediaType` is fixed at creation and never follows its contents | §3.2, §4 — it is a value the row carries, not one anybody derives |
| An unrecognised `MediaType` is a `400`, not a dropped token | §3.2's error table — the third refusal on that route |
| `mediaTypes=` filters playlists by the row, not by the type | §4 — undecided, and the only place 009 meets a 005 parameter it cannot answer from a kind |
| De-duplication against the existing entries fires about a third of the time, so a playlist **can** hold one item twice | §3.1, §3.4, AC-5, behaviours §3.18 — measured at T7, and the duplicate it makes is unaddressable |

## 8. References

- [docs/compatibility/api-surface-v1.md §6](../../docs/compatibility/api-surface-v1.md#6-playlists)
- [specs/005 §3.1](../005-item-query-api/spec.md) — the list envelope
- [docs/compatibility/client-embeat-mobile.md §3](../../docs/compatibility/client-embeat-mobile.md) — the named consumer for the rename
- [docs/compatibility/behaviours.md](../../docs/compatibility/behaviours.md) §2.7, §2.8, §2.26, §3.15, §3.16, §3.17, §3.18, §4.3, §5
- `[spec: CreatePlaylist, CreatePlaylistDto, PlaylistCreationResult, GetPlaylistItems, AddItemToPlaylist, RemoveItemFromPlaylist, MoveItem, DeleteItem, UpdateItem]`
