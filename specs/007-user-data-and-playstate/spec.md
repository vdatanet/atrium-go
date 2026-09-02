---
feature: 007-user-data-and-playstate
title: User data and playstate
status: Implemented
created: 2026-08-26
updated: 2026-08-28
accepted: 2026-08-28
implemented: 2026-08-28
amended: 2026-08-28 at the spec review — the reference's source predicted and the extended probe confirmed four corrections: §3.4 (a bare mark is `max(count, 1)`; only `datePlayed` increments), §3.5 (a container's `PlayedPercentage` is field-gated), §3.6 (rule 2 reversed — reports resolve last-writer-wins, and a play is counted at Start), §3.7 (the rule runs on every position-bearing report, not only stops); §3.8 added and AC-15 corrected by the `--reap` measurement — the reap commits a position extrapolated through the silence; OQ-1 and OQ-3 through OQ-6 answered; and 2026-08-28 at the plan gate, which measured plan §6.8's catalogue — §3.2's second Key calibration (a movie's is its own GUID, dashed), §3.3/§3.4's refusal shapes, §3.6's playing-session block (the NowPlayingItem slot and width, PlayState replaced whole) and error floor (AC-21, AC-22 added), and a Start's position measured unwritten
depends_on: [002, 005]
---

# 007 — User data and playstate

> **This document describes WHAT and WHY only.** No technology names, no storage decisions.

## 1. Purpose

Remember, per user and per item: whether it is a favourite, whether it has been played, how many
times, and where the user stopped.

**Client behaviour unlocked:** a heart that stays filled, a "continue watching" row that is
correct, and an episode that resumes at the right second on a different device from the one that
paused it.

## 2. Scope

**In scope**

- `POST` and `DELETE /UserFavoriteItems/{itemId}`, `POST` and `DELETE /UserPlayedItems/{itemId}`.
- `POST /Sessions/Playing`, `/Sessions/Playing/Progress`, `/Sessions/Playing/Stopped`.
- The `UserData` object 005 returns on every item.
- Aggregation: how a season's played state follows from its episodes.
- The completion threshold that turns "stopped near the end" into "played".

**Out of scope**

- `POST`/`DELETE /UserItems/{itemId}/Rating` and `/UserItems/{itemId}/UserData` — no analysed client
  uses them.
- Remote control: v1 records what a session reports, it does not command sessions.
- Watch history as a queryable log. v1 keeps aggregate state, not a timeline.
- SyncPlay.

## 3. Behaviour

### 3.1 The `UserData` object

Returned inline on **every** item in every 005 response, with no `Fields` or `EnableUserData`
needed.

| Field | Type | Notes |
|---|---|---|
| `PlaybackPositionTicks` | integer | Ticks; `0` when not started |
| `PlayCount` | integer | Moved by playback and by `datePlayed`, not by a bare mark; §3.4, §3.6 |
| `IsFavorite` | boolean | |
| `Played` | boolean | |
| `PlayedPercentage` | number | Leaves: position over runtime, present when a position exists. Containers: fraction of children played, present **only when the request asks for `Fields=RecursiveItemCount`** — a bare container row carries `UnplayedItemCount` and `Played` and no percentage `[probe: tools/probe_playstate.py, Jellyfin 10.11.11, 2026-08-28]` |
| `UnplayedItemCount` | integer | Containers only |
| `LastPlayedDate` | date | Set when playback **starts** (§3.6), and by the mark route; absent until then |
| `Key` | string | Jellyfin-specific; §3.2 |
| `ItemId` | string | Jellyfin-specific |

`[probe: tools/probe_playstate.py, Jellyfin 10.11.11, 2026-08-28]` — `Key` and `ItemId` inside
`UserData` are present in the Jellyfin dialect and absent in Emby's, so a client using them to
tell the servers apart would see the wrong answer if Atrium omitted them. On the wire the pair
closes the object — a measured mark response reads `UnplayedItemCount`,
`PlaybackPositionTicks`, `PlayCount`, `IsFavorite`, `Played`, `Key`, `ItemId`, which is the
declared order 005's emitter already pins with the null-suppressed fields absent.

**Every field is per user.** Two users watching the same file share nothing.

### 3.2 `Key`

A stable string identifying the item *for user-data purposes*, distinct from its `Id`.

It exists because user data should survive things an item id does not. In the reference it is
derived from the item's provider identity or its name, so a re-identified or moved item can keep
its state.

**v1 derives `Key` from the item's stable identity** (003 §3.6), which gives the same survival
property for the cases v1 can produce. It is opaque to clients — they store it and send it back —
so its derivation is not observable.

**OQ-1 is resolved: neither analysed client reads it.** jellyfin-apple-tv maps `Played`,
`IsFavorite`, `PlayedPercentage`, `PlaybackPositionTicks` and `UnplayedItemCount` into its domain
type and drops the rest; embeat's `UserDataDto` does not even declare `Key` or `ItemId`, so its
decoder discards them on arrival `[survey: jellyfin-apple-tv Domain/Mapping.swift, embeat
MediaBrowserDto.kt, 2026-08-28]`. The fields exist to be *present* — a dialect marker (§3.1) —
not to be parsed, and the derivation stays v1's own. For calibration, the reference's own values
are visibly not item ids — a season measured `Key: "309992001"`, a provider-derived string,
beside a 32-hex `ItemId` `[probe: tools/probe_playstate.py, Jellyfin 10.11.11, 2026-08-28]` —
so clients demonstrably tolerate `Key` values of any shape. A movie and an artist measured the
other direction at the plan gate: their `Key` **is** the item's own GUID, in dashed form beside
the 32-hex `ItemId` — one object spelling one identity two ways
`[probe: tools/probe_playstate.py, Jellyfin 10.11.11, 2026-08-28]` (the by-name battery, T1).

### 3.3 Favourites

`POST /UserFavoriteItems/{itemId}` marks; `DELETE` unmarks. Both return the updated `UserData`
object.

**These are the Jellyfin routes.** Emby's user-scoped `/Users/{userId}/FavoriteItems/{itemId}` was
removed in 10.11 and Atrium does not serve it ([ADR-0004](../../docs/decisions/0004-pin-to-jellyfin-10-11.md)).

**Idempotent, in both directions.** Marking twice is `200` both times and leaves one favourite;
unmarking twice is `200` both times too — unmarking what was never marked is not an error
`[probe: tools/probe_playstate.py, Jellyfin 10.11.11, 2026-08-28]`. Clients retry.

**The refusals, measured at the plan gate** — all four mark routes share them: an unknown or
invisible item is `404` RFC 9457 problem details; a path `itemId` that is not a GUID at all is
`400` validation problem details naming the parameter; no token is the empty-body `401`. Each
is a shape behaviours §1.11 already catalogues — nothing new, and nothing bespoke
`[probe: tools/probe_playstate.py, Jellyfin 10.11.11, 2026-08-28]` (the refusal battery, T1).

Any item type may be a favourite — an artist, an album, a series, a single track. **A container's
favourite does not cascade**: favouriting a season leaves every episode unfavourited, measured —
the flag is stored on the container's own row, unlike the played mark (§3.4)
`[probe: tools/probe_playstate.py, Jellyfin 10.11.11, 2026-08-28]`.

### 3.4 Played state

`POST /UserPlayedItems/{itemId}` marks played; `DELETE` marks unplayed. Both return the updated
`UserData`. The `POST` takes one optional query parameter, `datePlayed`, and it changes more than
the date.

| Action | Effect |
|---|---|
| Mark played, no `datePlayed` | `Played` true, **`PlayCount` becomes `max(count, 1)` — marking twice leaves it at one** — `LastPlayedDate` set if absent and kept if present, **`PlaybackPositionTicks` reset to 0** |
| Mark played with `datePlayed` | As above, except `PlayCount` **increments** and `LastPlayedDate` becomes the given date |
| Mark unplayed | `Played` false, `PlayCount` 0, position 0, `LastPlayedDate` cleared |

`[probe: tools/probe_playstate.py, Jellyfin 10.11.11, 2026-08-28]`
`[source: MediaBrowser.Controller/Entities/BaseItem.cs:1893-1951 @ v10.11.11]`

*This section said "`PlayCount` incremented" until the review measured it. The count belongs to
playback (§3.6): the mark route only guarantees it is non-zero, and the `datePlayed` form exists
for imports — a scrobble backfill that increments once per record.*

The position reset matters: an item that is played *and* has a resume position appears in "continue
watching" forever. That state is nonetheless reachable — the reference's own progress path
produces it (§3.7) — so the reset is a property of the mark route, not a server-wide invariant.

**Marking a container marks its leaves, and only its leaves.** Marking a season played marks its
episodes; marking an album played marks its tracks. The container's **own stored row is never
written** — after marking a season played its episodes each carry `PlayCount: 1`, while the
season's row still reads `PlayCount: 0` with no `LastPlayedDate`, its `Played: true` being
derived from the children (§3.5). Measured, and the sweep in the reference's source covers every
non-container descendant, so a series mark reaches episodes through their seasons
`[probe: tools/probe_playstate.py, Jellyfin 10.11.11, 2026-08-28]`
`[source: MediaBrowser.Controller/Entities/Folder.cs:1730-1786 @ v10.11.11]`. A client offering
"mark season watched" expects exactly this. Unmarking a container sweeps the same set back.

### 3.5 Aggregation

A container's played state is derived from its children, never stored independently:

| Container field | Derived as | When present |
|---|---|---|
| `UnplayedItemCount` | Count of unplayed leaf descendants | Every container row |
| `Played` | True when every leaf descendant is played | Every container row |
| `PlayedPercentage` | Fraction of leaf descendants played | **Only when the request carries `Fields=RecursiveItemCount`** |

The field-gating is measured, not guessed: a season with nine of ten episodes played answers
`Played: false, UnplayedItemCount: 1` and **no** `PlayedPercentage` on a bare row, and
`PlayedPercentage: 90` when the request asks for `RecursiveItemCount`
`[probe: tools/probe_playstate.py, Jellyfin 10.11.11, 2026-08-28]`. The favourite flag, the
count and the position are **not** aggregates — they are the container's own row (§3.3, §3.4).

"Leaf descendants" means the recursive non-container children — the reference's aggregate skips
folders and virtual items, a distinction v1 cannot yet exhibit because it has no virtual items
`[source: MediaBrowser.Controller/Entities/Folder.cs:1798-1840 @ v10.11.11]`.

Derived, not cached-and-updated. Cached aggregates drift — a rescan adds an episode, a file is
removed, a user marks a child directly — and a drifted count is visible on every series poster.
Where derivation is too slow, the plan may cache it, but the specified behaviour is that a client
can never observe a stale aggregate.

**OQ-7 is resolved, and the answer is that the question mostly cannot be asked.** A `Series`,
`Season`, `MusicArtist` or `MusicAlbum` with nothing visible beneath it is **not offered at all**
— it does not earn its place ([behaviours §5.2](../../docs/compatibility/behaviours.md)), so
there is no row for a client to read a flag off and the vacuous-played question never reaches the
wire. The one container that *is* exempt is a library folder, because an empty library must stay
in a sidebar, and that is where the answer lives: an empty library reads `Played: false` with
`UnplayedItemCount: 0` here, where the reference's source reads a childless folder as vacuously
played. Recorded as a divergence
([behaviours §5.7](../../docs/compatibility/behaviours.md)) and owed to 010's differential,
which is the only thing that can measure it without creating a library on somebody's server.

### 3.6 Playback reporting

Three endpoints, all answering **`204`**, all used by both analysed clients.

| Endpoint | When | Body |
|---|---|---|
| `POST /Sessions/Playing` | Playback starts | `ItemId`, `PlaySessionId`, `MediaSourceId`, `PlayMethod`, `CanSeek`, `AudioStreamIndex`, `SubtitleStreamIndex`, `PositionTicks` |
| `POST /Sessions/Playing/Progress` | Periodically, and on pause, seek and track change | Same, plus `IsPaused`, `IsMuted`, `VolumeLevel` |
| `POST /Sessions/Playing/Stopped` | Playback ends | `ItemId`, `PlaySessionId`, `PositionTicks`, `Failed` |

`[spec: PlaybackStartInfo, PlaybackProgressInfo, PlaybackStopInfo]`

**`MediaSourceId` is not required on `Progress`.** Emby requires it; Jellyfin does not, and a server
that rejected reports without it would silently lose the resume positions of any client written
against the Jellyfin dialect. `[probe: tools/probe_playstate.py, Jellyfin 10.11.11, 2026-08-28]`

**A report binds to the caller's session, not to a session it names.** Whatever a body carries,
the reference resolves the session from the authenticated device and overwrites any claimed
identifier; `PlaySessionId` names the *playback*, not the session, and none of the three requires
it. `[source: Jellyfin.Api/Controllers/PlaystateController.cs:199-260 @ v10.11.11]`

**Effects** — measured, and two of them are not what the draft assumed
`[probe: tools/probe_playstate.py, Jellyfin 10.11.11, 2026-08-28]`
`[source: Emby.Server.Implementations/Session/SessionManager.cs:814-832, 938-968, 1115-1145 @ v10.11.11]`:

| Report | Effect on `UserData` | Effect on the session |
|---|---|---|
| Start | **`PlayCount` increments and `LastPlayedDate` is set here**, at start — not at the end. `Played` becomes **false**: starting a previously played item un-marks it until it completes again. Position untouched — a Start carrying `PositionTicks` at 30% measured the stored position still 0 | `NowPlayingItem` and `PlayState` appear (002 §3.8); `LastPlaybackCheckIn` advances |
| Progress | The reported position resolves **through §3.7's rule** — a progress past the ceiling marks the item played mid-playback. No count change | `PlayState` updated; `LastPlaybackCheckIn` advances |
| Stopped, with a position | The position resolves through §3.7's rule. **No count change** — the play was counted at start | `NowPlayingItem` cleared |
| Stopped, no position | Played to the end: `Played` true, position 0 — and the count increments **a second time**, so a natural start-to-finish viewing without a final position measures `PlayCount: 2` | `NowPlayingItem` cleared |

**What a playing session shows** — measured at the plan gate on a live playback and
reproduced by the probe's playing-session battery at T1 `[probe: tools/probe_playstate.py, Jellyfin 10.11.11, 2026-08-28]`:

- **`NowPlayingItem` takes one slot: between `DeviceName` and `DeviceId`**, the session
  object's 002-measured order otherwise unchanged.
- **`PlayState` is the last report, whole — replaced, never merged.** After a Start carrying
  `CanSeek: true` and `VolumeLevel: 80`, a progress omitting both read back `CanSeek: false`
  and no `VolumeLevel` at all. The measured playing set, in order: `PositionTicks`, `CanSeek`,
  `IsPaused`, `IsMuted`, `VolumeLevel`, `AudioStreamIndex`, `SubtitleStreamIndex`,
  `MediaSourceId`, `PlayMethod`, `RepeatMode`, `PlaybackOrder` — the nullable ones present
  exactly when the last report carried them, and `PositionTicks` advancing between reports
  (§3.8's ticker).
- **`NowPlayingItem` is an item without `UserData`** — the one measured item shape that omits
  §3.1's object entirely — **and without fourteen other properties a full item body carries**.
  Measured against the same playback: the item's 41 properties are exactly a full
  `/Items/{itemId}` body's 56 minus `Etag`, `CanDelete`, `CanDownload`, `SortName`,
  `ForcedSortName`, `MediaSources`, `ProductionLocations`, `PlayAccess`, `RemoteTrailers`,
  `People`, `UserData`, `DisplayPreferencesId`, `Tags`, `LockedFields` and `LockData`. It is a
  **subtraction from the full body**, not a selection of its own, and nothing in it is absent from
  that body `[probe: manual requests via tools/_probe.py, Jellyfin 10.11.11, 2026-08-28]`. The media-derived subset (`MediaStreams`, `Chapters`, `Width`,
  `Height`, `HasSubtitles`, `IsHD`, `VideoType`, `Trickplay`, `Container`) and `CriticRating`
  are outside what v1 can yet say, stay absent until the feature that owns them, and are a
  recorded gap the differential will show — 006's `Chapter` shape, not a silent one.
- **Its width is the item's, not the shape's.** Two movies measured **41 and 40 properties**,
  the difference being `IsHD` — null on one of them, and nulls are omitted globally (§3.1's
  emitter, behaviours §1.9). "Forty-one properties" was one item's width read as a constant; a
  differential that compares property *counts* between two servers would report a difference
  that is only the item talking `[probe: tools/probe_playstate.py, Jellyfin 10.11.11, 2026-08-28]`.

**The reports' error floor, measured** — leniency starts after the body binds: a body that is
not JSON, or an `ItemId` that is not a GUID at all, is `400` validation problem details; a
`Stopped` carrying a **negative** position is the one refusal past binding — `400`,
`text/plain`, the fixed `Error processing request.` body (behaviours §1.11's controller-refusal
shape). Rule 1's `204` is for a well-formed id that names nothing.

**And the binder's refusal names two things, one of them per route:** the failure itself — `$`
carrying the parser's message and byte position when the text is not JSON, the **empty string**
carrying `The supplied value is invalid.` when it parses but a value does not bind — beside the
name of the body the route declares, which is `playbackStartInfo`, `playbackProgressInfo` or
`playbackStopInfo`, saying that field is required. So one failure spells its `errors` keys three
ways across the three routes `[probe: tools/probe_playstate.py, Jellyfin 10.11.11, 2026-08-28]` (behaviours §1.11). *This is the half of the plan
gate's "the extended handler reproduces it for free" that is not free: a path parameter's
refusal already matches, a body's does not.*

**Robustness rules, because these arrive over unreliable networks from clients that crash:**

1. **Unknown `ItemId` is `204`, not an error.** All three endpoints, measured `204` for an id
   that names nothing. A report for an item removed mid-playback is not worth failing; the
   client cannot act on the failure anyway.
2. **Reports resolve last-writer-wins.** A progress report carrying a position older than the
   stored one **rewinds it** — 40% then 20% reads back 20%, measured. *The draft specified the
   opposite: "a progress report older than the stored position does not rewind it". No such
   guard exists in the reference,* and it could not: a deliberate seek backwards arrives as
   exactly this report, and a server that ignored it would pin every rewinding viewer at their
   furthest point. Ordering is the client's job; the server's is to record what it is told.
3. **A missing `Stopped` must not lose the position.** Clients are killed by the operating
   system mid-playback. The last `Progress` stands on its own — measured: a progress lands with
   no `Start` before it and no `Stopped` after — and a session that stops reporting is reaped
   with the viewer's place preserved, advanced through the silence rather than frozen at the
   last report (§3.8).
4. **`Failed: true` records nothing — and undoes nothing.** The failed stop itself writes no
   position and no count, measured. But the play was counted at *start* (row 1), so a failed
   playback still leaves `PlayCount` incremented and `LastPlayedDate` set. The draft's "a
   playback that never started is not progress" was half right: the stop is inert, the start
   already happened.

### 3.7 What a reported position does to the stored one

Not two thresholds — **one ordered rule with six branches**, measured and then read from the
reference's source to learn its shape. `[probe: tools/probe_playstate.py, Jellyfin 10.11.11, 2026-08-26]` `[source: Emby.Server.Implementations/Library/UserDataManager.cs:296-370 @ v10.11.11]`

**The rule runs on every report that carries a position — `Progress` as much as `Stopped`.**
Measured: a progress report at 95% marks the item played and clears the position mid-playback,
exactly as a stop there would `[probe: tools/probe_playstate.py, Jellyfin 10.11.11, 2026-08-28]`.
*This section's title said "a stop report" until the review measured that; the one branch that
remains stop-only is row 1 — a `Progress` with no position simply leaves the stored one alone,
while a `Stopped` with none means played-to-the-end.* Only rows 2, 4 and 5 touch `Played`; the
mid-range branch (row 6)
does not, which is how a position reported *after* completion coexists with `Played: true` —
measured, and §3.4's "continue watching forever" state is therefore reachable on the reference
through its own progress path.

Given a report carrying position `P` against an item of runtime `R`:

| # | Condition | Outcome |
|---|---|---|
| 1 | No position reported at all | `P` becomes `R` — a stop with no position means *played to the end* |
| 2 | `R` unknown | Marked **played**, no position kept |
| 3 | `P/R < MinResumePct` | Position **discarded**. Not played |
| 4 | `P/R > MaxResumePct`, **or** `P` within one second of `R` | Position discarded, marked **played** |
| 5 | Otherwise, but `R` shorter than `MinResumeDurationSeconds` | Position discarded, marked **played** |
| 6 | Otherwise | Position **kept**. The item is resumable |

Defaults, and Atrium adopts them: `MinResumePct` **5**, `MaxResumePct` **90**,
`MinResumeDurationSeconds` **300**. Measured against a live reference at 90.2% and 5.1% on a
100-minute item by bisection at 0.5% precision, then pinned **exactly**: a stop at the first
tick whose percentage reaches 5 keeps its position and one tick below discards it; a stop at the
last tick not past 90 keeps its position and one tick above marks played
`[probe: tools/probe_playstate.py, Jellyfin 10.11.11, 2026-08-28]`.

Four details that a rule written from intuition gets wrong, and each is observable:

**The comparisons are strict.** Exactly at the floor the position is kept; exactly at the ceiling
it is kept. Only *below* the floor and *above* the ceiling do the branches fire. Measured at
tick precision, above.

**Row 5 is about the item's runtime, not the position.** A clip shorter than five minutes that is
stopped in the middle is marked **played**, not resumable — the reference decides that something
that short has no meaningful resume point. This is a different rule from row 3, which is a floor on
*progress*, and conflating the two produces a server that keeps resume positions for every short
item. Measured, resolving OQ-6: a 215-second track stopped at 50% reads back `Played: true`
with no position `[probe: tools/probe_playstate.py, Jellyfin 10.11.11, 2026-08-28]`.

**Row 4's second clause decides nothing under these thresholds, and the argument once given for
it pointed the other way.** "For a long item, 90% can still be minutes from the end" is true, and
it means a position within one second of the end is *far above* 90% — the first clause has already
fired. Below ten seconds of runtime the first clause can lose, and row 5 marks the item played
anyway. Checked exhaustively over runtimes from one second to two hours at every boundary
position: **no report can tell the two rules apart** (`test_the_within_one_second_clause_changes_no_answer_under_these_thresholds`).
It is reproduced because the reference has it and because a deployment that lowered
`MinResumeDurationSeconds` would give it something to decide — not because a client can observe
it. *This paragraph claimed the opposite until T2 implemented the rule.*

**Row 1 matters more than it looks.** Clients send a stop with no position when playback ends
naturally. Treating that as position zero would leave every finished item unplayed and at the
start.

Books and audiobooks follow a different, minute-based rule in the reference — an absolute floor in
minutes and a *remaining time* ceiling. Out of v1's media scope, recorded so that a later feature
does not apply the percentage rule to them by default.

An item marked played this way is excluded from `/UserItems/Resume` (005 §3.7).

### 3.8 Reaping a silent session — and the position that kept moving

A session that reports a start and then goes silent must not hold `NowPlayingItem` forever, and
must not lose the viewer's place. The reference sweeps every five minutes for playing sessions
whose last check-in is more than five minutes old and stops each as though it had sent a
`Stopped` carrying the session's live position — which then resolves through §3.7 like any other
`[source: Emby.Server.Implementations/Session/SessionManager.cs:612-680 @ v10.11.11]`. Measured:
`NowPlayingItem` cleared after **8.6 minutes** of silence
`[probe: tools/probe_playstate.py --reap, Jellyfin 10.11.11, 2026-08-28]`.

**The committed position is not the last reported one.** The measurement asked for confirmation
that the last `Progress` survives and got a better answer: 40% reported, 8.6 minutes of silence,
**48.5% stored** — the 40% plus the silence, almost to the second. The reference runs a
one-second ticker per unpaused session that extrapolates the position in real time, the live
`PlayState.PositionTicks` a `/Sessions` reader watches advance between reports; it never
advances a paused session and never past the runtime, and the reap commits the extrapolated
value `[source: MediaBrowser.Controller/Session/SessionInfo.cs:23, 373-451 @ v10.11.11]`.

Both halves are observable — the resume point lands minutes past the last report, and `/Sessions`
shows a position that moves between reports — so **v1 adopts the whole shape**: the same five-minute
sweep and threshold, a position read as *last reported plus unpaused wall-clock since, capped at
the runtime*, and the reaped stop resolving through §3.7.

The paused-session variant — the reference can also *command* a paused session to stop after a
configurable threshold, disabled by default — is remote control, out of v1's scope (§2).

## 4. Data the feature owns

| State | Observable as | Lifetime |
|---|---|---|
| Per-user, per-item favourite state | `UserData.IsFavorite` | Until changed |
| Per-user played state and count | `UserData.Played`, `PlayCount`, `LastPlayedDate` | Until changed |
| Per-user resume position | `UserData.PlaybackPositionTicks` | Until played or reset |
| Live playback state per session | `NowPlayingItem`, `PlayState` in `/Sessions` | While playing, until stopped or reaped (§3.8) |

**User data outlives items** (003 §3.8). A file that disappears and returns keeps its state,
because the alternative is a user losing their library's history to a temporarily unmounted share.

## 5. Acceptance criteria

1. `UserData` is present on every item in every 005 response, with `Key` and `ItemId` inside it.
2. Marking a favourite twice answers `200` twice and leaves one favourite; unmarking twice
   answers `200` twice and leaves none.
3. Marking played resets `PlaybackPositionTicks` to 0 and sets `PlayCount` to at least one —
   **marking twice leaves the count at one**, and only a mark carrying `datePlayed` increments
   it. *(This criterion said "increments" until the review measured the bare mark.)*
4. Marking unplayed clears played, count, position and `LastPlayedDate`.
5. Marking a season played marks every episode played, and the season's own row is never
   written — its `PlayCount` stays 0 while every episode's reads 1.
6. A season's `UnplayedItemCount` matches its unplayed episodes after: marking one episode,
   rescanning with a new episode added, and removing an episode.
7. Two users' state on the same item is fully independent.
8. All three reporting endpoints answer `204`.
9. `Progress` without `MediaSourceId` is accepted, and a `Progress` with no `Start` before it
   still lands. `[probe: tools/probe_playstate.py, Jellyfin 10.11.11, 2026-08-28]`
10. Reports resolve last-writer-wins: a progress at 40% followed by one at 20% reads back 20%.
    *(Reversed at the review — the draft required the rewind **not** to happen, and the
    reference has no such guard.)*
11. A report for an unknown item answers `204` — all three endpoints.
12. Each branch of §3.7 is exercised: no position; unknown runtime; below the floor; above the
    ceiling; within one second of the end; a short item stopped mid-way; and the resumable case.
13. The floor and ceiling comparisons are strict at tick precision — the first tick reaching 5%
    keeps its position, one below discards; the last tick not past 90% keeps, one above plays.
14. A `Failed: true` stop writes nothing itself — position and count unchanged by the stop —
    while the start that preceded it keeps its effects (AC-17).
15. A session silent past the threshold is reaped: its `NowPlayingItem` clears with no report
    arriving, and the stored position is the last reported one **advanced by the unpaused
    silence**, capped at the runtime (§3.8). *(The draft said "the last position intact"; the
    measurement came back 8.6 minutes richer.)*
16. Deleting an item's file and rescanning preserves its user data; restoring the file restores the
    association.
17. A `Start` report increments `PlayCount`, sets `LastPlayedDate`, and sets `Played` to
    **false** — starting a previously played item un-marks it until it completes again.
18. A `Stopped` with a position does not change `PlayCount`; a `Stopped` without one increments
    it a second time.
19. A `Progress` past the ceiling marks the item played mid-playback — §3.7's rule is not
    stop-only.
20. A container's `PlayedPercentage` appears only when the request carries
    `Fields=RecursiveItemCount`; a bare container row carries `UnplayedItemCount` and `Played`
    and no percentage.
21. A report whose body does not bind — non-JSON, or a non-GUID `ItemId` — answers `400`
    validation problem details; a `Stopped` with a negative position answers the `text/plain`
    `400`; a mark for an unknown item answers the problem-details `404` and for a non-GUID
    path the validation `400`.
22. During playback the session entry carries `NowPlayingItem` between `DeviceName` and
    `DeviceId`, without `UserData` inside it, and `PlayState` mirrors exactly the last
    report's fields — a progress omitting `CanSeek` reads back `CanSeek: false`.
23. A container's favourite does not cascade: favouriting a season leaves every episode
    unfavourited — the flag lands on the container's own row, unlike the played mark of AC-5
    (§3.3). *(Added at the 2026-08-28 audit — M45: measured, implemented and tested with no
    criterion pinning the asymmetry.)*
24. A report binds to the caller's session, never to a session it names: whatever
    `PlaySessionId` or session identifier a body carries, the playback lands on the
    authenticated device's `/Sessions` entry and nowhere else (§3.6). *(Added at the same
    audit — M46, with the discriminating test it lacked.)*

## 6. Conformance

| Endpoint | Level | How it is proven |
|---|---|---|
| Favourite mark/unmark | **L2** | Round-trip plus idempotency both ways (AC-2) |
| Played mark/unmark | **L2** | Including the count that does not move (AC-3), the position reset and the cascade (AC-5) |
| The three reporting endpoints | **L2** | Status, effect on `UserData` (AC-17, AC-18), effect on `/Sessions` (AC-22), the error floor (AC-21), and the reap (AC-15) |
| `UserData` shape | **L3** | Golden plus differential — it appears on every item, so an error is everywhere |
| Aggregation | **L2** | Fixture mutated between assertions (AC-6), the field-gated percentage (AC-20) |
| Threshold behaviour | **L2** | Table-driven over position × runtime, on both report types (AC-19) |

## 7. Open questions

None. OQ-7, the last one open, was resolved at T11.

### Resolved

| # | Question | Answer | Resolved by |
|---|---|---|---|
| OQ-1 | Does any client parse `UserData.Key`, or is it opaque? | **Opaque — neither analysed client reads it.** One drops it in mapping, the other never declares it. §3.2's derivation stays free | Survey of jellyfin-apple-tv and embeat, 2026-08-28 |
| OQ-2 | The reference's completion percentage and minimum-position thresholds | **90% ceiling, 5% floor, 300s minimum item runtime — and the rule has six branches, not two.** §3.7 rewritten; the boundaries pinned strict at tick precision on 2026-08-28 | `tools/probe_playstate.py`, 2026-08-26 and 2026-08-28 |
| OQ-3 | Does the reference cascade a container mark to children, or only report aggregates? | **It cascades to the leaves — ten of ten episodes — and never writes the container's own row.** AC-5 is parity, not a divergence | `tools/probe_playstate.py`, 2026-08-28 |
| OQ-4 | How long the reference waits before reaping a silent session | **A five-minute sweep for sessions silent past five minutes — measured at 8.6 — and it commits the position extrapolated through the silence, not the last reported one.** §3.8, AC-15 corrected | `tools/probe_playstate.py --reap` and source, 2026-08-28 |
| OQ-5 | Does the reference count a play at start, at stop, or at the threshold? | **At start** — and a second time on a positionless stop. §3.6's effects table | `tools/probe_playstate.py`, 2026-08-28 |
| OQ-6 | Whether row 5 fires as the source reads — a short item stopped mid-way marked played | **It fires**: a 215-second track stopped at 50% is played with no position | `tools/probe_playstate.py`, 2026-08-28 |
| OQ-7 | Is an empty container vacuously played, where 005 shipped unplayed? | **Unaskable for four of the five container types** — an empty one is not offered at all — and `Played: false` for the fifth, an empty library. §3.5, behaviours §5.7 | 007 T11, 2026-08-28 |

## 8. References

- [docs/compatibility/api-surface-v1.md §5, §9](../../docs/compatibility/api-surface-v1.md#5-user-data)
- [docs/compatibility/behaviours.md §2.1, §2.9](../../docs/compatibility/behaviours.md#21-userdata-is-always-present)
- [specs/003 §3.8](../003-library-configuration-and-scanning/spec.md) — user data outliving items
- `[spec: MarkFavoriteItem, UnmarkFavoriteItem, MarkPlayedItem, MarkUnplayedItem, ReportPlaybackStart, ReportPlaybackProgress, ReportPlaybackStopped, UserItemDataDto]`
- The reference's own resolution paths, read at the review and then measured:
  `[source: Emby.Server.Implementations/Library/UserDataManager.cs:296-370 @ v10.11.11]`,
  `[source: Emby.Server.Implementations/Session/SessionManager.cs:612-680, 746-832, 938-968, 1012-1145 @ v10.11.11]`,
  `[source: MediaBrowser.Controller/Entities/BaseItem.cs:1893-1951 @ v10.11.11]`,
  `[source: MediaBrowser.Controller/Entities/Folder.cs:1730-1840 @ v10.11.11]`,
  `[source: MediaBrowser.Controller/Session/SessionInfo.cs:23, 373-451 @ v10.11.11]`
