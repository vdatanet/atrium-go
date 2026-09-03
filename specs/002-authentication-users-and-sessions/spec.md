---
feature: 002-authentication-users-and-sessions
title: Authentication, users and sessions
status: Implemented
created: 2026-08-26
updated: 2026-09-03
accepted: 2026-08-26
amended: 2026-08-26 by the T1 probe - sections 3.1, 3.2, 3.3, 3.5, AC-2, AC-3 and the open questions; by T7 - sections 2, 3.1, 3.2, AC-3 and section 6; by T11 - sections 3.3, 3.4, 3.5 and AC-6; by T12 - section 3.8; by T18 - AC-3 and AC-10. 2026-08-28 by the L2 probe fold - section 3.8: an unknown capabilities property is dropped from the session's echo, not kept. 2026-09-01 by tools/probe_user_read.py - section 3.7, AC-7 and the section 6 matrix: GET /Users/{userId} refuses no authenticated caller, the 403 it stated with no provenance is withdrawn, and the two identifiers that name nobody are a 404 and a 400 rather than that same refusal; and 2026-09-01 at the closing audit - OQ-5 is narrowed to the three refusals still unmeasurable without costing somebody's account a lockout counter. It also held the `403` for insufficient permission and the shape of both `403`s, on the premise that the only account available to measure with is an administrator; three probes now create throwaway non-administrators, and 009 T2 measured both shapes, so that half was a debt the table was still reporting after it had been paid. 2026-09-03 at 001's closing audit (001 T21) - section 5 gains AC-14, the half of 001 AC-5 that needs a credential only this feature can issue, together with the two assertions 001 parked at a lower level for the same reason. 2026-09-03 by this feature's own plan, in four places: section 3.9 and a section 4 row are new, because nothing in v1 completes setup and AC-14 would otherwise be met by a request carrying no token at all; section 3.5 records that two of the twenty-eight unenforced policy flags are enforced by the reference at authentication itself, so "they all gate features v1 does not have" holds for twenty-six; section 5's two carried conditions are corrected, because neither "when this feature can complete setup over HTTP" nor "when the rename endpoint lands" can ever be met - no v1 row does either; and OQ-6 records what the reference's source says about the -1 sentinel while staying open, because the running server has not been asked. 2026-09-03 again, in the change that follows that plan - section 3.8 declares `GET /Sessions`' three request parameters, section 5 gains AC-15 and records that AC-4's second half finally has a request, and section 6's row for the route names the matrix that proves it. What forced it: behaviours 2.25 has carried the measurement since 2026-08-29 and says in terms that the three are "specified in 002, in the change that adds them", the video client sends `deviceId` today, and a task list cannot be written across the gap. The refusal is the load-bearing half - naming another user's device is an empty `200` while naming another user in `controllableByUserId` is a `403`, and a route declaring neither answers both with a `200` the client reads as success. And one claim did not survive being written down: the *order* of `deviceId` and the visibility rule is NOT observable, because predicates over one list commute - what a client can see is the combination with `controllableByUserId`, which is what AC-15 asserts - and behaviours 2.25's "Atrium does: none of it" is corrected in the same change. Two cases the measurement does not reach are marked UNVERIFIED and registered as U-17 and U-18 rather than answered
depends_on: [001]
---

# 002 — Authentication, users and sessions

> **This document describes WHAT and WHY only.** No technology names, no storage decisions.

## 1. Purpose

Turn a username and a password into a token, recognise that token on every later request, and know
which device is holding it.

Everything after this feature is user-scoped: an item's `UserData`, a library's visibility, a resume
position. None of it can be specified until "who is asking" has an answer.

**Client behaviour unlocked:** a login screen that works, and a session that survives the app being
closed.

## 2. Scope

**In scope**

- `POST /Users/AuthenticateByName`, `GET /Users/Public`, `GET /Users/Me`, `GET /Users/{userId}`,
  `POST /Users/Configuration`, `POST /Sessions/Capabilities/Full`, `GET /Sessions`.
- The five token-presentation mechanisms, on every authenticated route in the project.
- The client-identification header, and the device identity it establishes.
- User accounts, their configuration and the parts of their policy v1 honours.
- Session lifecycle: creation, activity tracking, expiry.

**Out of scope**

- Creating, editing and deleting users over HTTP (`POST /Users/New`, `/Users/Password`,
  `/Users/{userId}/Policy`). v1 manages accounts through configuration, not an admin API.
- Quick Connect, forgotten-password flows, external authentication providers.
- Remote control of one session by another; `GET /Sessions` reports, it does not command.
- API keys not belonging to a user.

## 3. Behaviour

### 3.1 How a client presents a token

**Five** mechanisms, all accepted, on **every** authenticated route:

| Mechanism | Form |
|---|---|
| Header | `Authorization: MediaBrowser Token="{token}"` |
| Header | `X-Emby-Authorization: MediaBrowser Token="{token}"` |
| Header | `X-Emby-Token: {token}` |
| Query | `?ApiKey={token}` |
| Query | `?api_key={token}` |

Listed in the order the reference resolves them. **The second was missing from this specification
until it was measured** `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]` — the reference reads both header names with the same
grammar of §3.2, and a token in either authenticates. It is the historical Emby form, so a server
implementing only the other four would refuse clients that have worked against the reference for
years. Either header may carry the client's identification and the token together, which is what
most clients send.

`[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-26]`

All five are required, not a choice. Clients use headers for API calls and the query forms for URLs
handed to media players and image loaders, which do not set headers. Supporting only the headers
breaks playback and artwork while leaving browsing intact — a failure that looks like a bug in the
client.

**When a request carries two that disagree, the one that wins is measured, not chosen:**

    Authorization  >  X-Emby-Authorization  >  X-Emby-Token  >  ?ApiKey= / ?api_key=

measured pair by pair, in both directions each time.
`[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]`

**The image and delivery route classes accept all four and require none.** On the reference an
image and a static stream answer a request carrying no token at all, so on those two classes the
mechanisms are accepted rather than demanded. That is a measurement about the reference, and what
Atrium does about it belongs to the features that own those routes — see
[behaviours §2.10](../../docs/compatibility/behaviours.md#210-the-image-and-delivery-routes-accept-a-token-and-require-none).

**Rejection is not uniform, and the split is measured.** A request to an authenticated route with
no token or an unknown token is `401` with an empty body. A valid token whose user lacks the
required permission is `403`. **A disabled account is `403`**, not the `401` this specification
first assumed. `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-26]` The distinction is what clients branch on — they re-authenticate on
`401` and show an error on `403` — so a disabled account answered `401` loops a client through a
login it can never complete, with the user's password correct every time.

### 3.2 How a client identifies itself

`X-Emby-Authorization: MediaBrowser Client="…", Device="…", DeviceId="…", Version="…"`

**Mandatory on `POST /Users/AuthenticateByName`, and there only.** A header carrying no
`DeviceId` is served normally on every other route — measured `200`, not a refusal. `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]`
The `Emby` in the name is historical.

| Component | Meaning |
|---|---|
| `Client` | Application name, e.g. `Jellyfin Android` |
| `Device` | Human-readable device name shown in session lists |
| `DeviceId` | Stable per-installation identifier. **This is what identifies a session** |
| `Version` | Client version string |

**The header must carry a scheme word**, and it is `MediaBrowser` or `Emby`, matched
case-insensitively. Without one — or with any other word — nothing is read out of the header at
all. `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]`

Parsing is lenient in some of the ways clients are sloppy and **strict in two the reference is
strict about**, and the difference was measured rather than assumed:

| Variation | Accepted |
|---|---|
| Components in any order | yes |
| Values quoted, or bare | yes |
| No space after a comma, or a space before one | yes |
| Extra spaces after the scheme, or a trailing comma | yes |
| Unknown components | yes, ignored |
| **Whitespace around the `=`** | **no** |
| **A lowercase component name** | **no** |

An earlier version of this section claimed whitespace around `=` was accepted. It is not, and
matching the reference matters more than being kind: no working client sends that form, and
accepting it would let a client be built against Atrium that fails against the reference
([behaviours §6](../../docs/compatibility/behaviours.md#6-non-improvements)).

**The header is accepted anywhere and authenticates nobody.** Alongside a token the request
succeeds; carrying only this header and no token, an authenticated route answers `401` with the
same empty body as a request carrying nothing at all. `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-26]` It identifies a client; it
does not admit one. Atrium accepts it on any route and uses `DeviceId` to attribute the session
when present.

### 3.3 `POST /Users/AuthenticateByName` — `AuthenticateUserByName`

**Consumers:** music-client, video-client.

**Request**

| Part | Name | Required | Notes |
|---|---|---|---|
| header | `X-Emby-Authorization` | yes | §3.2 |
| body | `Username` | yes | Matched case-insensitively |
| body | `Pw` | yes | May be empty when the account has no password |

**Response — 200**

```json
{
  "User": { },
  "SessionInfo": { },
  "AccessToken": "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6",
  "ServerId": "3f9c1a7e5b2d4e8091a6c3f70d5e2b14"
}
```

| Field | Notes |
|---|---|
| `User` | The full user object of §3.5 |
| `SessionInfo` | The session created by this authentication, §3.8. `LastPlaybackCheckIn` is `0001-01-01T00:00:00.0000000Z` for one that has never played anything — .NET's minimum date, not null and not absent `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]` |
| `AccessToken` | 32 lowercase hex characters `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]` |
| `ServerId` | The server identity from 001 §3.1 |

**Errors**

| Condition | Status |
|---|---|
| Unknown username | `401` `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-26]` |
| Wrong password on an enabled account | `401` — v1's own decision; the reference's answer is unmeasured and held by §7 OQ-5 |
| **Disabled account** | **`403`** — whether the password is right or wrong `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-26]` |
| Locked out by failed attempts | `403` — v1's own decision; the reference's answer is unmeasured and held by §7 OQ-5 |
| Missing or unparseable `X-Emby-Authorization` | `400` `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-26]` |
| Malformed body | `400` |

**Every refusal measured above carries the same body**: 25 bytes of `text/plain`, with no charset
parameter, reading `Error processing request.` This is a third refusal shape, distinct from the
empty `401` an authenticated route sends and from the structured problem document the framework's
own validation sends. `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-26]` A malformed body is the exception and keeps that structured
shape, because it is rejected before the handler runs. The measured refusals are asserted by
comparing bytes, since they share one body and only the status tells them apart —
[behaviours §1.11](../../docs/compatibility/behaviours.md#111-there-are-four-error-shapes-not-one).

The split is load-bearing for clients: `401` means "your credentials are wrong, ask again", `403`
means "this account cannot log in, stop asking", and any other `4xx` means "something else went
wrong, do not tell the user their password is bad". Answering `401` where the reference answers
`403` produces a login loop the user cannot escape by typing the correct password.

**Behaviour beyond the response**

1. A session is created for `(user, DeviceId, Client)`. Authenticating again from the **same
   `DeviceId`** replaces that session rather than accumulating one per login.
2. Failed attempts are counted per user. After the configured threshold the account locks and
   further attempts fail regardless of credentials. A success resets the count. **The status a
   locked-out account answers with is not measured** — measuring it means locking a real account
   on a real server, which no probe here will do to somebody's installation (§7 OQ-5). Until it
   is, Atrium answers `403`, on the argument that a locked account is in the same state a disabled
   one is: further attempts cannot succeed, and telling a client to keep asking is the failure
   mode `403` exists to prevent.
3. Passwords are never recoverable from the server's stored state, and never appear in logs — not
   at any log level, not in a request trace, not in an error message.

### 3.4 `GET /Users/Public` — `GetPublicUsers`

**Consumers:** video-client. Unauthenticated.

Returns the users the server shows on login screens: an array of **complete** §3.5 objects,
`Configuration` and `Policy` included, byte-identical to what an authenticated caller receives.
`[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]`

**This section previously asserted the opposite** — that the two were omitted, "because this is
pre-authentication and it must not disclose what a user is allowed to do". The reasoning was sound
and the premise was measured to be false: the reference sends every user's full policy and
configuration to anyone who can reach the port. Atrium replicates it, and the argument, including
the case for diverging, is in
[behaviours §3.5](../../docs/compatibility/behaviours.md#35-userspublic-discloses-every-users-policy-to-anyone--class-b-replicated).

**An empty array is a valid `200`.** Users flagged hidden from login screens are excluded, and an
installation where every user is hidden legitimately returns `[]`.
`[prior-probe: Jellyfin 10.11.11, 2026-06-13]`

### 3.5 The user object

Returned by §3.3, §3.4, §3.6 and §3.7.

| Field | Type | Notes |
|---|---|---|
| `Name` | string | Sent first |
| `ServerId` | string | **Before `Id`** — the reference's order, not this table's former one `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]` |
| `Id` | string | 32 hex |
| `ServerName` | string | |
| `PrimaryImageTag` | string | Present only when the user has an avatar |
| `PrimaryImageAspectRatio` | number | Same condition |
| `HasPassword` | boolean | |
| `HasConfiguredPassword` | boolean | |
| `HasConfiguredEasyPassword` | boolean | Always `false`; v1 has no PIN concept |
| `EnableAutoLogin` | boolean | |
| `LastLoginDate` | date | Absent until first login |
| `LastActivityDate` | date | |
| `Configuration` | object | §3.6. **Sent by `/Users/Public` too** |
| `Policy` | object | **Sent by `/Users/Public` too** |

`ServerName` and `PrimaryImageAspectRatio` are declared and **absent from every measured
response**, because they are null and nulls are omitted globally
([behaviours §1.7](../../docs/compatibility/behaviours.md)). Their position in the order is
therefore unverified: nothing can measure where a property that is never sent would sit.

`[spec: UserDto]`

**Policy in v1.** The reference sends **42** policy properties. `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]` v1 stores and returns the
whole set so clients see the shape they expect, but **honours fourteen of them** — eleven since it
was written, and the three playback permissions since transcoding entered v1, which is the last
row of this table and the amendment below it:

| Flag | Effect |
|---|---|
| `IsAdministrator` | Reserved; v1 has no admin surface to gate |
| `IsDisabled` | Authentication answers `403` `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-26]` |
| `IsHidden` | Excluded from `/Users/Public` |
| `EnableAllFolders`, `EnabledFolders` | Which libraries the user sees |
| `EnableMediaPlayback` | Whether delivery routes answer |
| `EnableContentDeletion`, `EnableContentDeletionFromFolders` | Whether deletion is permitted |
| `LoginAttemptsBeforeLockout`, `InvalidLoginAttemptCount` | §3.3. The reference sends **-1** for the first, which is a sentinel and not a count `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]` — see §7 OQ-6 |
| `MaxActiveSessions` | Cap on concurrent sessions; `0` means unlimited, and `0` is what the reference sends `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]` |
| `EnableVideoPlaybackTranscoding`, `EnableAudioPlaybackTranscoding`, `EnablePlaybackRemuxing` | Whether the negotiation may answer this user with a transcode or a remux ([008 §3.3](../008-playback-negotiation-and-delivery/spec.md#33-the-decision)) `[spec: UserPolicy]` |

**The other 28 are stored and echoed unchanged.** **This is a known, bounded gap**, not an oversight: a
flag returned but unenforced is a delta a client could observe by testing the restriction. It is
accepted for v1 because the unenforced flags all gate features v1 does not have (Live TV, sync,
remote control) — enforcing "you may not sync" on a server that never syncs is not observable. Any
flag whose feature arrives must be enforced in the same change.

> **Amended 2026-09-03, while writing this feature's plan: two of the 28 are not unobservable, and
> the sentence above is not true of them.** `EnableRemoteAccess` and `AccessSchedules` are enforced
> by the reference **at authentication itself** — a user without remote access authenticating from
> outside the local network, and a user outside their permitted schedule, are each refused there
> `[source: Jellyfin.Server.Implementations/Users/UserManager.cs:595-611 @ v10.11.11]`. The feature
> they gate is not Live TV or sync; it is logging in, which v1 has. So v1 admits two logins the
> reference refuses, and *"all gate features v1 does not have"* holds for 26 of the 28. The
> reference's own refusals here are unmeasured — each needs an account restricted in that way and a
> caller outside the local network — so this is recorded rather than acted on, and it is
> [U-15](../../docs/compatibility/reference-target.md) in the register of measurements this project
> owes. ~~[behaviours §5](../../docs/compatibility/behaviours.md)'s accepted-gaps row states the same
> thing about the same 28 and is owed the same correction by whoever owns it.~~ **That correction
> landed on 2026-09-03**, in the change that wrote the login path — the code that decides not to
> enforce them — so the two documents now say twenty-six in the same words.

**The three transcoding flags moved into the enforced set on 2026-08-27**, when transcoding entered
v1 ([roadmap](../../docs/roadmap.md#in-scope)). That is this rule working as written rather than an
edit to it: the feature arrived, so the flags that restrict it stopped being unobservable, and a
user whose policy forbids transcoding is told the source is not playable instead of being handed
one.

### 3.6 `POST /Users/Configuration` — `UpdateUserConfiguration`

**Consumers:** video-client.

Replaces the authenticated user's configuration: audio and subtitle language preferences, subtitle
mode, whether missing episodes are displayed, view ordering and exclusions, next-episode autoplay.
`[spec: UserConfiguration]` The reference sends **16** properties in all, the rest being cast
receiver, local-password, remembered track selections, played-item hiding, collection view and
folder grouping. `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]`

`204` on success. `401` unauthenticated. Unknown properties are ignored, not rejected.

**v1 stores and returns every property faithfully but acts on only those that change something v1
does:** language preferences (which influence default track selection in 008) and
`DisplayMissingEpisodes` (which changes what 005 returns). The same bounded-gap reasoning as §3.5
applies.

### 3.7 `GET /Users/Me` and `GET /Users/{userId}`

**Consumers:** video-client (`Me`), music-client (`{userId}`).

Both return the §3.5 object in full, including configuration and policy.

**`GET /Users/{userId}` refuses no authenticated caller.** Any caller carrying a usable token is
answered `200` with the named user's whole §3.5 object — a non-administrator naming another
non-administrator, a restricted non-administrator naming an **administrator**, an administrator
naming anybody, and a user naming themselves are one answer, and the bytes do not depend on who
asked: the administrator's object as read by a restricted stranger is byte-identical to that
administrator's own reading of it.
`[probe: tools/probe_user_read.py, Jellyfin 10.11.11, 2026-09-01]`

| The request | The answer |
|---|---|
| Any authenticated caller, any existing `userId` | `200`, the whole §3.5 object, `Configuration` and `Policy` included |
| A `userId` that is well formed and belongs to nobody | `404`, the message as a JSON-encoded bare string: `"User not found"`, 16 bytes, `application/json; charset=utf-8`. **The same body to an administrator and to a non-administrator** |
| A `userId` that is not an identifier at all | `400`, the model binder's validation body, keyed on the parameter's own spelling: `{"userId": ["The value 'not-an-identifier' is not valid."]}` |
| No credential at all | `401`, empty, as everywhere else |

`[probe: tools/probe_user_read.py, Jellyfin 10.11.11, 2026-09-01]`

> **This section asserted a `403` for a non-administrator reading anybody else, from the day 002
> was written until 2026-09-01, with no provenance.** 009 T2 measured one cell of it on 2026-08-31
> and found `200`; the whole matrix above was measured on 2026-09-01 and found no refusal anywhere
> in it. Atrium **replicates**, and the decision is
> [behaviours §3.22](../../docs/compatibility/behaviours.md#322-any-authenticated-caller-reads-any-user-whole--class-b-replicated):
> it is the disclosure §3.4 already replicates on `/Users/Public`, reached by a second road, and
> keeping the refusal on one road while disclosing on the other is the inconsistency rather than
> the protection. Principle I outranks the improvement — a client that reads another user against
> the reference must not meet a `403` here.
>
> The `404` and the `400` in the table are the second half of the finding, and neither was in the
> question that started this: the refusal Atrium sent for an identifier nobody has was the same
> `403`, so that the two could not be told apart. The reference tells them apart, and its `404` is
> the **fourth** error shape rather than the problem details every other handler-raised `404` in
> this project answers
> ([behaviours §1.11](../../docs/compatibility/behaviours.md#111-there-are-four-error-shapes-not-one)).

### 3.8 Sessions

A session is a `(user, device, client)` triple, created at authentication and identified by the
`DeviceId` of §3.2.

**`POST /Sessions/Capabilities/Full` — `PostFullCapabilities`.** Included by design; clients post
their playable media types, supported commands and device profile after logging in. v1 stores the
declaration and reflects it in §3.9. It does not act on `SupportedCommands`, because v1 has no
remote control — but a client that posts capabilities and then sees them missing from `/Sessions`
has observed a difference, which is why storing it is not optional.

**`GET /Sessions` — `GetSessions`.** Returns the sessions the caller may see: their own always, and
all sessions for an administrator. Each carries the identity fields (`Id`, `UserId`, `UserName`,
`Client`, `DeviceId`, `DeviceName`, `ApplicationVersion`, `RemoteEndPoint`), activity timestamps,
the declared capabilities, and — while something is playing — `NowPlayingItem` and `PlayState`,
which feature 007 populates. `[spec: SessionInfoDto]`

**`POST /Sessions/Capabilities/Full` answers `204` with no body, and replaces rather than merges.**
An unknown property is accepted — the `204` — and **dropped from the session's `Capabilities`**;
Atrium keeps it, which is a recorded divergence and not parity
([behaviours §5.9](../../docs/compatibility/behaviours.md#59-an-unknown-capabilities-property-survives-into-sessions-here-and-not-there)).
*(This section said "the reference keeps it too" until the 2026-08-28 run read the echo back:
the 2026-08-26 measurement saw only the `204`.)*
`[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]`

`SupportsMediaControl` and `SupportsRemoteControl` are `false` in v1. This was argued as honest
rather than a gap — a client that saw `true` would offer a remote-control UI that does nothing —
and it is now **measured to be no divergence at all**: the reference reports `false` at the top
level for a session that posted `SupportsMediaControl: true`, while echoing that `true` back inside
`Capabilities`. `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]` The declaration is the client's; the flag is the server's
judgement about it. `PlayableMediaTypes` and `SupportedCommands` *are* hoisted from the declaration
verbatim.

**`GET /Sessions` accepts three parameters, and they are not three filters.** The reference declares
`controllableByUserId`, `deviceId` and `activeWithinSeconds`
`[source: Jellyfin.Api/Controllers/SessionController.cs:52-59 @ v10.11.11]` and applies them in a
fixed order whose observable half is narrower than it looks — below
`[probe: tools/probe_session_filters.py, Jellyfin 10.11.11, 2026-08-29]`. The names match without
regard to case, as every parameter name does
([behaviours §1.15](../../docs/compatibility/behaviours.md#115-query-parameter-names-match-case-insensitively));
the spellings above are the declared ones.

| Parameter | What it does to the result |
|---|---|
| `deviceId` | Narrows the **whole** list **first**, before the rule about whose sessions the caller may see, keeping the sessions whose `DeviceId` matches — compared **without regard to case**. A value nothing matches is `200` with `[]` |
| `activeWithinSeconds` | Applied **last**, after visibility, and only when greater than zero: it keeps the sessions whose `LastActivityDate` is no older than that many seconds |
| `controllableByUserId` | **Not a filter.** It replaces the caller's own visibility rule with a different one — the sessions the named user is allowed to remote-control — and refuses some callers outright, below |

**What the order buys, and what it does not.** behaviours §2.25 records the order the reference
applies these in, and it is worth being exact about which part of it a client can see. `deviceId`
and the visibility rule are both predicates over one list, so they **commute**: a caller who is not
an administrator naming somebody else's device is answered `200` with `[]` whichever ran first, and
`activeWithinSeconds` running last is the same kind of no-difference. So the sequence is not by
itself the observable thing, and a criterion that claimed to tell the two orders apart on `deviceId`
alone would be asserting something no request can distinguish.

**What is observable is that the two parameters answer differently.** Naming another user's
**device** is an empty `200`; naming another **user** in `controllableByUserId` is a `403`. One
route, two parameters that name somebody else's property, two answers — and the empty list is not a
redaction of the refusal, it is the ordinary result of a filter that matched a row the caller may
not see. The sequence does become visible in one place, which is the combination: `deviceId`
narrows the list **before** `controllableByUserId` replaces the visibility rule, so the two apply
together rather than the second discarding the first.

**Absent, empty, and values that are none of the above.**
[behaviours §1.12](../../docs/compatibility/behaviours.md#112-an-unrecognised-query-value-is-ignored-not-rejected)'s
ignore-don't-reject rule is the rule here, and its line is token-versus-type:

| The request carries | The answer |
|---|---|
| None of the three | Every session the caller may see, as above |
| `deviceId=` — empty | Ignored: the parameter is treated as absent, not as a device nothing is named after `[probe: tools/probe_session_filters.py, Jellyfin 10.11.11, 2026-08-29]` |
| `activeWithinSeconds=0`, or any negative value | Ignored: the unfiltered list, not a refusal `[probe: tools/probe_session_filters.py, Jellyfin 10.11.11, 2026-08-29]` |
| `controllableByUserId` naming the caller | Accepted from any caller, administrator or not |
| `controllableByUserId` naming anybody else, from a caller who is **not** an administrator | `403`, the controller's own refusal shape — `text/plain` with no charset and the same 25 bytes as §3.3's refusals. The status and the media type are measured `[probe: tools/probe_session_filters.py, Jellyfin 10.11.11, 2026-08-29]`; the bytes are [behaviours §1.11](../../docs/compatibility/behaviours.md#111-there-are-four-error-shapes-not-one)'s measured rule applied to a refusal the reference raises the same way, and they are **owed a request of their own** (register U-18) |
| A parameter this route does not declare | Ignored, as everywhere |
| `activeWithinSeconds` that is not an integer, or `controllableByUserId` that is not an identifier | ⚠️ **UNVERIFIED.** v1 answers `400` with the validation body keyed on the parameter's own spelling — the shape §3.7 measured for `userId` — because §1.12 forgives an unrecognised *token* and refuses a value that cannot parse as its declared *type*. Neither has been sent on this route: register U-17 |
| `controllableByUserId=` — empty | ⚠️ **UNVERIFIED.** v1 treats it as absent, which is the direction the measured `deviceId` row takes: register U-17 |

**What v1 answers for `controllableByUserId`, and why it is not a gap.** v1 accepts the parameter,
sends the `403` above, and otherwise answers `200` with `[]` — for every caller, including an
administrator. That is not a filter left unimplemented: the first clause of the rule is that a
session is remote-controllable only while a live control channel is attached to it
`[source: MediaBrowser.Controller/Session/SessionInfo.cs:246-266 @ v10.11.11]`, v1 has no such
channel, and this section already reports `SupportsRemoteControl: false` on every session — which
is measured to be what the reference reports for a request-response client too
([behaviours §2.14](../../docs/compatibility/behaviours.md#214-a-clients-declared-capabilities-and-the-servers-flags-are-different-values)).
The rule's three later clauses — the named user's shared-device setting, the caller's
remote-control permission, and per-device access — therefore decide nothing observable here, and
this specification does not state them. They belong to whichever feature first attaches a control
channel, and until one does they cannot be told apart from the empty list.

**Why all three are declared when one consumer sends one.**

- **`deviceId` has a consumer today.** The video client verifies that a stream really is being
  copied rather than transcoded by asking `GET /Sessions?deviceId=…` and reading the transcoding
  fields off the session whose `NowPlayingItem.Id` matches
  ([client-atrium-tvos §4.4](../../docs/compatibility/client-atrium-tvos.md#44-get-sessions-takes-no-deviceid-and-the-client-sends-one)).
  A route that drops the parameter answers a wider list, which §1.12 makes look correct, and the
  client then matches on the item alone — so an administrator on a busy server can read **another
  device's** session as its own. That analysis called it *"a degradation, not a break"* against a
  server whose route declared nothing; declaring the parameter removes the degradation instead of
  documenting it.
- **`activeWithinSeconds` has no observed consumer**, and it is one comparison against a field this
  section already carries. Not answering it is a delta with nothing on the other side of the
  scale: the same request returns fewer rows there and more here.
- **`controllableByUserId` earns its place on the `403` rather than on the list.** The refusal is
  observable whatever v1 does with the list, and a route that ignored the parameter would answer
  `200` — with the caller's own sessions — where the reference refuses. A client that branches on
  that refusal would take the success path.

> **Added 2026-09-03, forced by this feature's own plan.** This section declared no parameter on
> this route from the day 002 was written, and
> [behaviours §2.25](../../docs/compatibility/behaviours.md#225-get-sessions-three-filters-are-two-filters-and-a-visibility-rule)
> has carried the measurement since 2026-08-29 while saying in terms that the three are *"specified
> in 002, in the change that adds them"*. The plan reached its own `/Sessions` section, found that
> it could not plan a handler for parameters no specification declares without writing WHAT into a
> HOW, and named this as the one thing the specification owed **before** the task list.
> ~~*"Atrium does: none of it — v1 declares no parameter on this route"*~~ was a true sentence about
> this document and is no longer one; behaviours §2.25 is corrected in the same change. What makes
> it more than tidiness is the refusal: a client asking about somebody else's session is answered an
> empty list on one parameter and a `403` on another, and a route that declared neither answers
> `200` with the caller's own sessions to both — a success where the reference refuses.
>
> **Writing it down cost one claim.** The sentence this section reached for first was that the
> *order* is observable, which is how §2.25's own wording reads. It is not: `deviceId` and the
> visibility rule are predicates over one list and predicates commute, so the empty `200` arrives
> whichever ran first. The order is real and it is in the reference's source; what a client can see
> of it is the combination with `controllableByUserId` and nothing else, and AC-15 asserts that
> rather than a sequence no request distinguishes.

**Lifecycle**

| Event | Effect |
|---|---|
| Authentication from a known `DeviceId` | Existing session is replaced, its token invalidated |
| Any authenticated request | `LastActivityDate` advances |
| Playback reporting (007) | `LastPlaybackCheckIn` advances |
| Inactivity beyond the configured window | Session and token expire; next request is `401` |
| `MaxActiveSessions` exceeded | Oldest session is evicted |

> ⚠️ **OQ-2.** The reference's inactivity window and whether it is observable to a client. If a
> client caches a token for weeks and Atrium expires it sooner, users see spurious logouts. Until
> measured, v1 does not expire tokens on inactivity at all — the safe direction, since a token that
> outlives the reference's is invisible to a client, whereas one that dies sooner is not.

### 3.9 An installation becomes set up when it is given its first account

**Setup completion is observable and nothing in v1 makes it happen**, which is why it is stated here
rather than left implicit. [001 §3.1](../001-server-identity-and-discovery/spec.md#31-get-systeminfopublic--getpublicsysteminfo)
reports `StartupWizardCompleted` on the response every client enters the API through, and
[001 §3.2](../001-server-identity-and-discovery/spec.md#32-get-systeminfo--getsysteminfo) permits
`GET /System/Info` to **any** caller while setup is outstanding, whatever the request carried. So on
an installation that can never finish setting up, that route admits every request for ever, and the
field is `false` for ever.

**An installation is set up from the moment it holds one account**, recorded once, at the instant the
first account is created; creating a second does not move it, and removing every account does not
undo it. The reference completes setup through an operation this surface does not include and no
named consumer calls `[source: Jellyfin.Api/Controllers/StartupController.cs:41-46 @ v10.11.11]`, so
v1 reaches the same observable state by the only event it has.

Two things this fixes, and both are why it is a behaviour rather than an implementation note:

- **AC-14 would otherwise be met by a request carrying nothing.** A token this feature issues is
  admitted to `GET /System/Info` — and so is no token at all, while setup is outstanding. The
  criterion is about the credential, so the state it is asserted in has to be one where the
  credential is what decides.
- **A reference stood up for a differential run has completed its own setup**, and answers `true`.
  An Atrium that can only answer `false` is a difference on an L3 response, on a boolean, that no
  feature before this one could have closed.

*(Added 2026-09-03, while writing this feature's plan.)*

## 4. Data the feature owns

| State | Observable as | Lifetime |
|---|---|---|
| User accounts | `/Users/*` responses | Until removed by an operator |
| Setup completion | `StartupWizardCompleted` in 001 §3.1 and §3.2, and whether `GET /System/Info` admits an unauthenticated caller | Recorded once, at the first account (§3.9) |
| Password verifier | `HasPassword`, and whether authentication succeeds | With the account |
| Failed-attempt counter | `InvalidLoginAttemptCount`; lockout behaviour | Reset on success |
| Access tokens | Whether a request is `401` | Until replaced, evicted or revoked |
| Sessions | `/Sessions` responses | See §3.8 |
| Per-user configuration | `Configuration` in §3.5 | Until replaced |

## 5. Acceptance criteria

1. Authenticating with valid credentials answers `200` with a 32-hex `AccessToken`, a full user
   object and a session.
2. An unknown username answers `401`; a **disabled account answers `403`**; a missing
   `X-Emby-Authorization` answers `400`. All three carry the reference's `text/plain` body, byte
   for byte, and are asserted as bytes rather than as status codes.
3. All **five** token mechanisms of §3.1 authenticate the same request identically on an API
   route, and a request carrying two resolves in the measured order:
   `Authorization` > `X-Emby-Authorization` > `X-Emby-Token` > query. On the image and delivery
   route classes all five are **accepted and none is required**, which is what the reference does —
   the criterion there is that presenting a token is never itself a reason to refuse.
4. No token on an authenticated route is `401`; a valid token lacking permission is `403`.
5. Re-authenticating from the same `DeviceId` replaces the session and invalidates the prior token.
6. `/Users/Public` answers without a token, excludes hidden users, and answers `200` with `[]`
   when all users are hidden. It carries the **whole** user object — `Configuration` and `Policy`
   included — because the reference does; this criterion asserted the opposite until it was
   measured.
7. `GET /Users/{userId}` answers **`200` with the named user's whole object to every authenticated
   caller** — a non-administrator naming another non-administrator, a non-administrator naming an
   administrator, an administrator naming anybody, and a user naming themselves alike — with
   `Configuration` and `Policy` included and the same bytes whoever asked. An identifier no account
   has is `404` carrying `"User not found"`; a malformed one is the validation `400` keyed on
   `userId`; no credential is the empty `401`. **This criterion asserted a `403` for a
   non-administrator until the route was measured on 2026-09-01**, and it was the last thing in
   this document still describing a refusal §3.7 had already withdrawn.
8. `POST /Users/Configuration` round-trips every property, including ones v1 does not act on.
9. Capabilities posted to `/Sessions/Capabilities/Full` appear in the caller's `/Sessions` entry.
10. After `LoginAttemptsBeforeLockout` failures the account answers **`403`** even with correct
    credentials, and one success afterwards resets the counter. `403` rather than `401` for the
    same reason a disabled account does — the reference's own answer here is unmeasured (§7, OQ-5),
    and this criterion said `401` until §3.3 was corrected and the two drifted apart.
11. A password never appears in any log record at any level, and never in an error body.
12. `GET /Users/Me` returns the caller's §3.5 object in full, configuration and policy included.
    *(Added at the 2026-08-28 audit — M30: the route was implemented and golden-tested with no
    criterion naming it.)*
13. The session lifecycle is observable: exceeding `MaxActiveSessions` evicts the least recently
    used session and its token; an authenticated request advances `LastActivityDate`, written at
    the next flush; and `POST /Sessions/Capabilities/Full` answers `204` with no body and
    **replaces** the previous set rather than merging into it (§3.8). *(Added at the same
    audit — M29.)*
14. **`GET /System/Info` answers `200` to a request carrying a token this feature issued**, with
    the body [001 §3.2](../001-server-identity-and-discovery/spec.md#32-get-systeminfo--getsysteminfo)
    describes. *(Carried from [001 AC-5](../001-server-identity-and-discovery/spec.md#5-acceptance-criteria)
    at 001's closing audit, 2026-09-03 — see the amendment below.)*
15. **`GET /Sessions` applies the three parameters of §3.8 in the measured order.** `deviceId`
    narrows the whole list before the visibility rule and matches without regard to case, so a
    caller who is not an administrator naming another user's device is answered `200` with `[]`;
    `activeWithinSeconds` is applied last and only when greater than zero, and `0` or a negative
    value answers the unfiltered list rather than refusing; `controllableByUserId` naming anybody
    but the caller, from a caller who is not an administrator, is `403` carrying the same 25 bytes
    as AC-2's refusals, and is otherwise `200` with `[]`. Statuses and bodies are asserted as bytes
    at the wire. What the criterion asserts about the order is what §3.8 says is observable — that
    the two parameters naming somebody else's property answer differently, and that `deviceId`
    still narrows a request that also carries `controllableByUserId` — and **not** that `deviceId`
    runs before the visibility rule, which no request can tell apart. *(Added 2026-09-03, with
    §3.8's parameter declaration.)*

**AC-4's second half has a request now, and it is that one.** *"A valid token lacking permission is
`403`"* named no route in this feature: none of the seven gates on a permission, §3.7 refuses
nobody, and 001's `/System/Info` admits any authenticated caller once setup is complete.
`GET /Sessions` naming another user in `controllableByUserId` is the one request in 002 where a
valid token is refused **for who its holder is**, and it is at the wire rather than at the domain.
The check is for the administrator role rather than for one of §3.5's twenty-eight policy flags,
which is what the criterion's word *"permission"* means on this route.

**Amended 2026-09-03. AC-14 is 001's, and it is here because 001 cannot reach it.**
[001 AC-5](../001-server-identity-and-discovery/spec.md#5-acceptance-criteria) is three claims —
`401` without a token, `200` with a valid one, and a body that is a superset of
`/System/Info/Public` agreeing on every shared field. 001 proves the first and the third. The
second needs a **valid credential**, and a credential is something this feature issues: 001 serves
no route that authenticates anybody, so the strongest thing it can say is that its handler asks its
authentication port and obeys the answer, which it asserts under that name
(`TestSystemInfoAdmitsWhatTheAuthenticatorAdmits`) rather than under AC-5's.

It is written here as a criterion rather than left as a note because the alternative was a test in
001 that admitted a **fictional** token — which would have proved the wiring, been named for AC-5,
and quietly closed a criterion nothing had met. Two things discharge it, and both belong to this
feature:

- 001's `Authenticator` port is filled — it is `httpapi.Authenticator`, `Authenticate(request)
  (Access, error)`, and it takes the whole request because §3.1's five mechanisms are three header
  names and two query names with a measured precedence;
- the request goes to the running binary in `conformance/`, carrying a token this feature's own
  authentication route returned, and the response is the superset body.

**Two smaller carried notes ride with it**, both recorded in 001's `tasks.md` and repeated here so
that this feature meets them as work rather than as a surprise. 001's `401` on `/System/Info` is
asserted at the HTTP boundary in `internal/httpapi` rather than in `conformance/`, because the
refusal needs an installation whose setup is **complete** and 001 serves no route that completes
one — ~~when this feature can complete setup over HTTP, that assertion moves to `conformance/`,
where it belongs~~ **when an installation whose setup is complete can be stood up, that assertion
moves to `conformance/`, where it belongs**. And 001's `/System/Ping` discrimination — that the body
is the product name and not the operator's friendly name — is proven at the wire against a server
whose friendly name is still the default, because 001 has no route that renames one; ~~when the
rename endpoint lands, send it in `TestPingAnswersTheProductNameAndNotThisServersFriendlyName` and
drop the caveat~~ **the caveat drops when an operator can name a server and a fixture can therefore
rename one; it does not drop when "the rename endpoint lands", because no such endpoint is in this
surface.**

> **Both conditions amended 2026-09-03, while writing this feature's plan, because neither could
> be met as written.** §3.9 completes setup and does not do it over HTTP, so the first note's
> precondition never arrives while the thing it was asking for — an installation in that state,
> reachable from a black-box test — does. And the reference renames a server at an operation this
> surface does not include `[source: Jellyfin.Api/Controllers/StartupController.cs:74-78 @ v10.11.11]`,
> so *"when the rename endpoint lands"* is a condition **no v1 feature can satisfy**, and a note
> waiting on it would outlive the project. What can satisfy it is the friendly name becoming
> something an operator sets; that is 001's datum and remains 001's to decide, and the caveat stands
> until somebody does.

## 6. Conformance

| Endpoint | Level | How it is proven |
|---|---|---|
| `POST /Users/AuthenticateByName` | **L3** | Golden response plus differential. Everything downstream depends on this being byte-right. **L2 is met; the differential half needs the harness [010](../010-conformance-harness/spec.md) delivers and a reachable reference server, so the gap is recorded rather than counted as met** |
| `GET /Users/Public` | **L2** | Golden response; fixture with a hidden user |
| `GET /Users/Me`, `GET /Users/{userId}` | **L2** | Golden response, plus the caller matrix — every pair of caller and subject, the identifier nobody has, the malformed one and the absent credential. It was a *permission* matrix until 2026-09-01; there are no permissions on this route to tabulate, so what it proves is that no caller is refused and no body is redacted |
| `POST /Users/Configuration` | **L2** | Round-trip test |
| `GET /Sessions` | **L2** | Fixture with two sessions on two devices, **plus §3.8's parameter matrix**: `deviceId` in another case, empty, and matching nothing; the order, from a caller who is not an administrator, against a device that is not theirs; `activeWithinSeconds` at `0`, at a negative value and at one that excludes a row; and `controllableByUserId` naming the caller, naming somebody else, and named by an administrator |
| `POST /Sessions/Capabilities/Full` | **L1** | Shape only; its effect is asserted through `/Sessions` |
| The five token mechanisms | **L2** | Table-driven across three route classes, including the precedence pairs and the grammar table (AC-3) |

## 7. Open questions

| # | Question | Blocks | Resolved by |
|---|---|---|---|
| OQ-2 | The reference's token inactivity window, and whether it is observable | The expiry row of §3.8. v1 defaults to no inactivity expiry | A probe holding a token idle |
| OQ-4 | Are `HasConfiguredPassword` and `HasConfiguredEasyPassword` read by any client? | Nothing; honest values are sent | Differential harness (010) |
| OQ-6 | What `LoginAttemptsBeforeLockout = -1` means. It is what the reference sends, so it is what most accounts carry, and it is a sentinel rather than a threshold | §3.3's lockout rule, which reads it as a count | A probe against a throwaway account, alongside OQ-5 |
| OQ-5 | The refusals a probe will not send at a real installation: an enabled account given a **wrong password**, an account **locked out** by failed attempts, and a **live token whose user was disabled** after it was issued | The two rows §3.3 states as v1's own decision, and the `403` v1 answers a locked-out account with | `tools/probe_auth_mechanisms.py` against a **throwaway enabled, non-administrator** account somebody is willing to lock |

> **OQ-6 gained an answer from the reference's own source on 2026-09-03, and stays open anyway.**
> `LoginAttemptsBeforeLockout` is a three-way switch and not a count: `-1` means *never lock*, `0`
> means *three*, and any other value is the number
> `[source: Jellyfin.Server.Implementations/Users/UserManager.cs:816-821 @ v10.11.11]`. Since `-1` is
> what the reference sends for a default account, **the default account never locks out at all**.
> The row is not moved to *Resolved*: [AGENTS.md §1.3](../../AGENTS.md) makes the running server the
> tie-breaker and none has been asked, and the probe that answers OQ-5 answers this in the same run.
> It is recorded here because a reader who took OQ-6 as *"nobody knows"* would compare a failure
> count against `-1` and lock every account either never or immediately.

> **Narrowed on 2026-09-01.** This row also held a **`403` for insufficient permission** and the **shape** of both `403`s, on the premise that *"the account available to measure with is an administrator, and an administrator lacks no permission"*. That premise stopped being true: three probes now create and delete throwaway non-administrators, and the shape is measured rather than analogised. A controller's own refusal is `text/plain` with no charset and 25 bytes; an authorization **policy**'s refusal has no body and no content type `[probe: tools/probe_playlist_visibility.py, Jellyfin 10.11.11, 2026-08-31]`. So the sentence *"which v1 sends empty by analogy with the measured empty `401`"* described code that no longer exists — `ForbiddenError` has answered the 25 bytes since 009 T2, and the empty shape is a second class. What remains open is the three refusals above, which still cost somebody's account a lockout counter to measure; [behaviours §1.11](../../docs/compatibility/behaviours.md#111-there-are-four-error-shapes-not-one) carries the shapes that were settled.

**Why OQ-5 is not simply measured.** Each of them needs a real account to fail against, and
failing against one moves a lockout counter that no probe can reset — on somebody's own server, for
an account somebody uses. The probe measures the refusals that cost nothing (an unknown username
cannot be locked out, and an account already disabled cannot be locked further) and declines the
rest by design rather than by omission.

### Resolved

| # | Question | Answer | Resolved by |
|---|---|---|---|
| OQ-1 | Is `X-Emby-Authorization` accepted outside authentication, and does it change behaviour alongside a token? | **Accepted anywhere, authenticates nobody.** Alongside a token the request succeeds; alone it is the same empty `401` as no header at all. §3.2 rewritten | `tools/probe_auth_mechanisms.py`, 2026-08-26 |
| OQ-3 | Does the reference answer `401` or `403` for a disabled user? §3.3 assumed `401` | **Contradicted: `403`**, and distinguishable from the `401` an unknown username gets — the opposite of the "indistinguishable on purpose" this specification asserted. §3.1, §3.3, §3.5 and AC-2 corrected, and the reasoning is in [behaviours §2.11](../../docs/compatibility/behaviours.md#211-a-disabled-account-is-refused-with-403-not-401) | `tools/probe_auth_mechanisms.py`, 2026-08-26 |
| — | Which mechanism wins when a request carries two that disagree? Not asked; the plan called the order arbitrary | **`Authorization` > `X-Emby-Token` > query.** [plan §6.1](plan.md#61-token-extraction) had fixed the opposite order | `tools/probe_auth_mechanisms.py`, 2026-08-26 |
| — | Do the image and delivery route classes require a token? Assumed yes by AC-3 | **No — both answer `200` with no token at all.** AC-3 corrected; what Atrium does about it is deferred to 006 and 008, [behaviours §2.10](../../docs/compatibility/behaviours.md#210-the-image-and-delivery-routes-accept-a-token-and-require-none) | `tools/probe_auth_mechanisms.py`, 2026-08-26 |

## 8. References

- [docs/compatibility/api-surface-v1.md §3](../../docs/compatibility/api-surface-v1.md#3-authentication-users-and-sessions)
- [docs/compatibility/behaviours.md §2.2, §2.4](../../docs/compatibility/behaviours.md)
- `[spec: AuthenticateUserByName, AuthenticationResult, UserDto, UserConfiguration, UserPolicy, SessionInfoDto, ClientCapabilitiesDto]`
