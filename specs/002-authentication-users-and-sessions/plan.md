---
feature: 002-authentication-users-and-sessions
title: Authentication, users and sessions — implementation plan
status: Accepted
created: 2026-09-03
updated: 2026-09-03
spec_status_required: Accepted
---

# 002 — Implementation plan

> **This document describes HOW.** It may not restate WHAT: the spec is the authority on behaviour,
> and a plan that repeats it will disagree with it eventually.

**On the gate.** The template asks for a spec at `Accepted` or better. 002's spec says
`Implemented`, which is [a statement about the exporting project](../../PROVENANCE.md) — *the WHAT
is settled and was proven once, elsewhere* — and 001's plan recorded that reading for the first
time. It is taken again here, with one addition this feature is the first to need: **writing this
plan amended the spec**, in four places, and that does not reopen the gate. The loop in
[specs/README.md](../README.md) closes deliberately — what planning teaches goes back into the
spec, in the same change — and the amendments are listed in §11 with what forced each one.

~~This plan is `In review`. It becomes `Accepted` when that review returns and a task list is asked
for, which is what `tasks.md`'s own `plan_status_required` gates.~~ **Moved to `Accepted` on
2026-09-03, in the change that wrote [tasks.md](tasks.md)** — the review returned and the task list
was asked for, which is exactly the transition that sentence described and what the list's own
`plan_status_required` needs. 001's plan recorded the same move in the same place.

**On two anchors this file has to honour.** Two documents already cite sections of a `plan.md` at
this path that did not exist: [behaviours §2.4](../../docs/compatibility/behaviours.md#24-there-are-five-authentication-mechanisms-and-one-of-them-wins)
and [spec §7](spec.md#7-open-questions) cite *"002 plan §6.1"* at the anchor `#61-token-extraction`,
and [behaviours §2.13](../../docs/compatibility/behaviours.md) cites *"002 plan §6.3"* at
`#63-the-x-emby-authorization-grammar`. They are two of PROVENANCE's *"links with nothing to point
at"*, and writing this file makes them resolve. **So §6.1 is token extraction and §6.3 is the
header grammar, deliberately**, because the alternative is a citation that lands on the wrong
subject — which is worse than one that lands nowhere. Nothing is repointed and nothing is deleted
([AGENTS.md §4](../../AGENTS.md)). One warning for a reader who follows them: the *sentences* beside
those citations describe the **exporting** project's plan at those numbers — §6.1 there *"fixed the
opposite order and called it arbitrary"* — and this §6.1 fixes the measured order instead. The
number is honoured; the claim about what is written under it is not this project's.

## 1. Approach

**002 is seven endpoints, no new edge, and one new value: the thing that turns a request into a
caller.**

001 built the whole pipeline and this feature inherits every stage of it — the readiness gate, the
response-time stamp, the `Server` header, path and query canonicalisation, routing, the three empty
refusal shapes, `internal/wire`'s two naming policies and its profile negotiation, `internal/surface`
and the L0 check in two views, and both L1 sweeps. **None of that is re-planned here.** What 001
deliberately left is a seam: [`httpapi.Authenticator`](../001-server-identity-and-discovery/plan.md#610-authentication-is-a-port-and-001-fills-it-with-nothing),
written at T18 *for this feature to fill*, taking the whole `*http.Request` because a credential is
three header names and two query names with a measured precedence between them.

The organising decision follows from where that seam is: **authentication is asked for by the route,
not imposed by a stage.** A stage that authenticated every request would be wrong twice over —
[behaviours §2.10](../../docs/compatibility/behaviours.md#210-the-image-and-delivery-routes-accept-a-token-and-require-none)
measures **six of the nine** delivery and subtitle actions requiring no token at all, and the image
routes requiring none either, so a refusing stage would refuse requests the reference serves; and a
`Caller` read out of a request context is the zero
value in any handler tested without the stage installed, silently, which is the failure 001 already
refused twice (`wire.Profile` at §6.3 there, and its own audit's *"a criterion about a request is not
met by a test about the mechanism"*). One `Authenticator` value serves every route, so *"all five
mechanisms on every authenticated route"* is a property of there being one reader rather than a
promise fifty-nine handlers keep.

The second organising decision is the one that makes this feature bigger than seven handlers.
**This is the first feature that cannot be tested without standing up an installation.** Every
criterion in spec §5 needs an account, most need two of different kinds, and
[`conformance/`](../../docs/architecture.md#3-repository-layout) may not import `internal/` — so
without a way to create accounts from **outside** the process, every one of those criteria would be
proven one layer in, which is precisely what 001's closing audit caught itself doing. Provisioning
is therefore part of this feature and is planned in §6.9, not left to whoever writes the tasks.

**Four readings changed this plan before it was written**, all at the pinned tag, and each is argued
where it lands:

1. **Nothing in v1 completes setup**, so `/System/Info`'s first-time-setup exemption admits *every*
   request for ever and AC-14 — *"`200` to a request carrying a token this feature issued"* — would
   be met by a request carrying no token at all. §6.8, and the spec gains a §3.9 for it.
2. **The reference refuses a login that would exceed `MaxActiveSessions`; it does not evict.**
   `[source: Emby.Server.Implementations/Session/SessionManager.cs:1623-1629 @ v10.11.11]` — where
   spec §3.8 and AC-13 evict the least recently used session. Implemented as the spec is written,
   recorded as [U-13](../../docs/compatibility/reference-target.md). §6.7.
3. **`LoginAttemptsBeforeLockout` is a three-way switch, not a count** — `-1` means *never lock*,
   `0` means *three*, anything else is the number
   `[source: Jellyfin.Server.Implementations/Users/UserManager.cs:816-821 @ v10.11.11]`. That is
   spec §7's OQ-6 answered by reading rather than by asking; the row stays open, because the running
   server is the tie-breaker and there is none here. §6.7.
4. **The 25-byte `text/plain` refusal is an exception filter, not a controller**, and its status is a
   function of which exception was thrown: `ArgumentException` → `400`, `AuthenticationException` →
   `401`, `SecurityException` → `403`, everything else → `500`
   `[source: Jellyfin.Api/Middleware/ExceptionMiddleware.cs:92-99,123-134 @ v10.11.11]`. Three of the
   four refusals spec §3.3 measures fall out of that one mapping, which is why §7 of this plan is a
   table of *which failure raises which* rather than a list of statuses per handler.

## 2. Inherited decisions

| Decision | Source |
|---|---|
| Go, `chi` over `net/http`, `encoding/json` behind `internal/wire`, no cgo; optional fields are pointers and `omitempty` on a non-pointer is banned | [ADR-0002](../../docs/decisions/0002-go-and-the-runtime-stack.md) |
| Embedded SQLite, pure Go, hand-written SQL, one file, two migration lineages; **no reference points from the precious half into the derived half** | [ADR-0003](../../docs/decisions/0003-sqlite-as-the-store.md), [architecture §6](../../docs/architecture.md#6-state-and-the-store-boundary) |
| Argon2id `m=64 MiB, t=3, p=2`, a PHC record carrying its own parameters, rehash on the next successful login, a startup-derived decoy, a ceiling of four verifications in flight where the rest **queue**, `golang.org/x/crypto` pinned at v0.55.0, and a plaintext type that cannot be logged | [ADR-0006](../../docs/decisions/0006-password-hashing.md) |
| Four layers, one direction; the domain imports no HTTP; ports import nothing of ours but the unit types | [architecture §2](../../docs/architecture.md#2-layers-and-the-direction-of-dependency) |
| `internal/` for everything but `cmd` and `conformance`; **`conformance/` imports nothing of ours**, enforced by `tools/check_conformance_imports` | [architecture §3](../../docs/architecture.md#3-repository-layout) |
| Where each wire fact lives, and that middleware order is contract | [architecture §4](../../docs/architecture.md#4-the-compatibility-boundary) |
| One process, one data directory; graceful shutdown is load-bearing | [architecture §5](../../docs/architecture.md#5-deployment-shape) |
| The session model is a **project-level** component, because 008's transcode registry is keyed by it | [architecture §7](../../docs/architecture.md#7-external-processes) |
| Process settings are few and are not a feature | [architecture §9](../../docs/architecture.md#9-configuration-identity-and-logging) |
| The pipeline and its order; the refusal helpers; the `Authenticator` seam and its rule that an `Access` the handler does not recognise is an error rather than a fall-through | [001 plan §6.7, §6.10](../001-server-identity-and-discovery/plan.md#67-pipeline-order) |
| A run needs three seats — `administrator`, `restricted`, `playback-denied` | [request-cases.yaml](../../docs/compatibility/request-cases.yaml) |

**Deviations:** none. One change looks like one and is not: `httpapi.Authenticator`'s return type
widens from `Access` to a struct carrying the caller beside it. 001's §6.10 wrote that interface
*for this feature to fill*, reserved its third `Access` value for this feature to add *"with the
shape"*, and put the domain half — *"a token to a session, a session to a user, a user to a
permission"* — explicitly in 002's hands. §5 argues the shape; the invariant 001 relied on is kept
by construction, and named there so it cannot be lost.

## 3. Modules

| Module | Change | Responsibility |
|---|---|---|
| `internal/users` | new | The domain of accounts: the user, its policy, its configuration, credential verification, the failed-attempt counter and the lockout rule. Imports no HTTP. |
| `internal/sessions` | new | The domain of sessions and tokens: identity derivation, activity, the declared capabilities, what a caller may see of somebody else's session, and — since spec §3.8 declared them — the route's three request parameters **and the order they apply in**. |
| `internal/ports` | extended | `Clock` (declared by 001's plan §5, never written; this feature is its first caller), and the two store interfaces the packages above declare. |
| `internal/store/sqlite` | extended | One migration in the **precious** lineage, and the readers and writers behind the two new ports. |
| `internal/httpapi` | extended | The credential reader and the header grammar (§6.1, §6.3); the `Authenticator` implementation; the seven handlers; the three refusal shapes 001 could not reach. |
| `internal/app` | extended | Wiring, plus the provisioning subcommand (§6.9). |
| `cmd/atrium` | extended | One dispatch on the first argument, and nothing else. |
| `conformance/` | extended | The seven rows' L1 and L2 evidence, over an installation the fixture provisions before the server starts. |

**Amended 2026-09-03, by spec §3.8's parameter declaration.** `internal/sessions`' row said *"what a
caller may see of somebody else's session"*, which was the whole of what the route decided when the
route took no parameters. It now decides three things in a fixed order, and the order is observable
(§6.10), so the row names it: a package whose responsibility is stated as *visibility* invites a
handler to do the filtering itself and get the order wrong.

**Why `users` and `sessions` are two packages.** A session outlives the credential that opened it
and is read by features that never look at an account:
[architecture §7](../../docs/architecture.md#7-external-processes) makes the transcode registry
*"touched by the session model 002 owns and by the delivery routes 008 owns"*, and 007 writes
`NowPlayingItem` and `PlayState` into a session row without touching a user. A single package would
make every one of those importers reach the credential verifier to hold a session id. The dependency
runs one way — `sessions` knows a user by identifier and never the reverse — which is
[architecture §6](../../docs/architecture.md#6-state-and-the-store-boundary)'s rule about identifier
references applied inside the domain rather than only across the store boundary.

**Why the credential reader is in the edge and not in the domain.** It reads three header names and
two query names off an `*http.Request`; that is HTTP, and
[architecture §2](../../docs/architecture.md#2-layers-and-the-direction-of-dependency) keeps HTTP out
of the domain. It follows 001's stage shape — a value built once, whose exported core is a **pure
function over strings** so that the grammar table of spec §3.2 is a table-driven test and not a
request per row. What crosses into the domain is a token string and nothing else.

**Why provisioning is `internal/app` and not `cmd/atrium`.**
[architecture §3](../../docs/architecture.md#3-repository-layout) says `cmd/atrium` may hold no
branch a test would want to reach, and provisioning is entirely branches worth reaching. This is
001 T1's amendment applied a second time rather than rediscovered: `cmd/atrium` dispatches, and
`internal/app` holds the command.

**What this feature does *not* add, deliberately.** No configuration for `LocalAddress`'s three
tiers, although 001 T16 addressed that note to *"002/003, or whichever feature first adds
configuration"*: the tiers are the installation's addressing and belong with whoever configures
libraries and publication, not with whoever creates accounts. The note stays open, and it is not
paid here by an unrelated flag.

## 4. Data model

All of it is the **precious** half. One migration, `0002_users_and_sessions.sql`, forward-only and
numbered contiguously in the lineage 001's runner already applies (001 plan §4). Nothing in this
feature is derived: a rescan rebuilds a library, and it must not log anybody out.

### The tables

**`users`** — one row per account.

| Column | Type | Note |
|---|---|---|
| `id` | TEXT PRIMARY KEY | 32 lowercase hex (behaviours §1.4's shape), derived — §6.5 |
| `username` | TEXT NOT NULL | As the operator spelled it; this is what `Name` returns |
| `username_folded` | TEXT NOT NULL UNIQUE | **A query-pattern column.** §3.3 matches a username case-insensitively, and this is the only column an authentication reads to find a row. It is stored rather than computed per query so that the uniqueness the login depends on is the database's rule and not a convention |
| `policy_document` | TEXT NOT NULL | §6.6 |
| `configuration_document` | TEXT NOT NULL | §6.6 |
| `invalid_login_attempt_count` | INTEGER NOT NULL | State, not policy — see below |
| `last_login_at` | INTEGER NULL | Ticks. NULL is what makes `LastLoginDate` **absent** until first login (§3.5) |
| `last_activity_at` | INTEGER NULL | Ticks |

**`user_credentials`** — zero or one row per account.

| Column | Type | Note |
|---|---|---|
| `user_id` | TEXT PRIMARY KEY REFERENCES `users(id)` | |
| `phc` | TEXT NOT NULL | ADR-0006's record: `$argon2id$v=19$m=65536,t=3,p=2$<salt>$<key>` |
| `written_at` | INTEGER NOT NULL | Ticks. What a rehash moves, and the only way to see one happened |

It is a table of its own rather than two columns on `users` for one reason worth stating: **every
read of a user object is a read of `users`, and none of them wants the verifier in memory.** `Pw`
and the record it is checked against exist on exactly one code path, and a separate table makes that
a property of the SQL rather than of everybody's discipline. `HasPassword` and
`HasConfiguredPassword` are both *"a row exists"*; `HasConfiguredEasyPassword` is the constant
`false` (§3.5), because v1 has no second credential to be configured.

**`sessions`** — one row per `(Client, DeviceId)`, with the user as a column. §6.5 argues that key.

| Column | Type | Note |
|---|---|---|
| `id` | TEXT PRIMARY KEY | 32 lowercase hex, derived from `(Client, DeviceId)` — §6.5 |
| `user_id` | TEXT NOT NULL REFERENCES `users(id)` | |
| `client`, `device_id`, `device_name`, `application_version` | TEXT NOT NULL | The four components of §3.2, echoed back by `/Sessions` |
| `remote_endpoint` | TEXT NOT NULL | |
| `capabilities_document` | TEXT NULL | The declaration `POST /Sessions/Capabilities/Full` posted, whole. NULL until one is posted |
| `created_at`, `last_activity_at` | INTEGER NOT NULL | Ticks |
| `last_playback_check_in_at` | INTEGER NOT NULL | Ticks. **The zero tick is a value here, not a missing one**: §3.3 measures `0001-01-01T00:00:00.0000000Z` for a session that has never played anything |

**`access_tokens`** — one row per live token.

| Column | Type | Note |
|---|---|---|
| `token_digest` | TEXT PRIMARY KEY | SHA-256 of the token, lowercase hex — below |
| `user_id` | TEXT NOT NULL REFERENCES `users(id)` | |
| `session_id` | TEXT NOT NULL REFERENCES `sessions(id)` | |
| `device_id` | TEXT NOT NULL | **A query-pattern column**, duplicating the session's. §6.5's replacement rule is keyed on `(user, device)` and the session's key is `(client, device)`, so the two cannot be reached through one another |
| `created_at` | INTEGER NOT NULL | Ticks |

**The token is stored as a digest and not as itself.** ADR-0006's threat model is the store file
leaking, and a leaked table of live bearer tokens is that leak with the hashing skipped. The digest
is **unsalted SHA-256, deliberately**: the input is 128 bits of uniform randomness (§6.5), so a salt
would defend against precomputation over a space nobody can precompute, while costing the primary-key
lookup that makes a per-request check one indexed read. This is invisible on the wire — no response
carries a stored token — so it is an engineering choice in ADR-0006's own sense and not a delta.

**Amended 2026-09-03, by T1, which wrote the migration.** The tables above list columns, and two
things a schema also carries were left implicit. Both are now in
`0002_users_and_sessions.sql` and both are tested:

- **`sessions` carries `UNIQUE (client, device_id)`.** *"One row per `(Client, DeviceId)`"* was
  stated here in prose and enforced only by the primary key, because §6.5 derives `id` from exactly
  that pair. That is true until the derivation changes, and the day it does the symptom is two live
  sessions for one client on one device and an authentication that replaces neither. The constraint
  is redundant today on purpose: it is the one thing in the schema that notices.
- **`access_tokens` carries an index on `(user_id, device_id)`.** §6.5's replacement rule is the
  only query against that table that does not go by its primary key, and without the index it is a
  scan of every live token on the installation.

### Two rules that decide what a later reader will otherwise get wrong

**The two document columns are serialised models, not schemas.** `policy_document` and
`configuration_document` hold the JSON of a Go struct whose **declaration order is the wire order**,
exactly as `httpapi.PublicSystemInfo` does — the user object travels inside the body of
`POST /Users/AuthenticateByName`, which is the one **L3** row this feature owns, and key order is
contract at L3. Adding a property is therefore a code change and **not** a migration.

The consequence has to be written down because Go's default is wrong here: **a document written by
an older build must decode onto the reference's defaults, never onto the zero struct.** The
reference's `UserPolicy` constructor sets `EnableMediaPlayback`, `EnableAllFolders`,
`EnableAllChannels`, `EnableRemoteAccess`, the three playback flags and eight more to `true`, and
`LoginAttemptsBeforeLockout` to `-1`
`[source: MediaBrowser.Model/Users/UserPolicy.cs:16-68 @ v10.11.11]`. Decoding onto `UserPolicy{}`
would answer `false` and `0` for every property an old document does not carry — and `0` is the
sentinel that means *lock after three attempts* (§6.7). Every decode starts from
`users.DefaultPolicy()` and unmarshals over it.

**The failed-attempt counter is state and lives on `users`, even though it is reported inside
`Policy`.** §3.5 lists `InvalidLoginAttemptCount` among the policy properties, and it moves on every
failed login; keeping it inside the document would rewrite the whole policy on each failure and make
a counter a schema change away from a permission. It is **overlaid into the policy object when the
user object is built** (§6.6), which is the one place the wire's shape and the store's shape are
allowed to disagree.

### Which of these is precious, and in what sense

[ADR-0003](../../docs/decisions/0003-sqlite-as-the-store.md) splits the store by what a rescan
rebuilds, and **the split is binary while the value of these four tables is not.** Written out, so
that a later feature does not put a session in the derived half by analogy:

| Table | Half | Why |
|---|---|---|
| `users`, `user_credentials` | Precious, in the strong sense | Reconstructible from nothing. A credential is the record ADR-0003 calls out by name |
| `sessions`, `access_tokens` | Precious, in the weak sense | **Not** rebuildable by a rescan, so they cannot be derived — a library scan that logged every client out would be the worst kind of surprise. But they are reconstructible *by the user*, at the cost of one login, which no other precious row is |

The distinction is not decorative. It is the rule for a repair: the derived half may be dropped and
rescanned, `users` and `user_credentials` may never be dropped at all, and `sessions`/`access_tokens`
are the one part of the precious half a repair may **truncate** — losing exactly one login per
client and no data. Nothing in this feature does that today; it is written here so that the day
something needs to, it is not a judgement call.

## 5. Contracts

```
// internal/ports — declared by the domain, implemented by the store
Clock interface { Now() units.Time }

UserStore interface {
    UserByFoldedName(ctx, folded string) (User, bool, error)
    UserByID(ctx, id string) (User, bool, error)
    Users(ctx) ([]User, error)
    Credential(ctx, userID string) (Credential, bool, error)
    ReplaceCredential(ctx, userID string, phc string, at units.Time) error
    ReplaceConfiguration(ctx, userID string, document []byte) error
    RecordLoginOutcome(ctx, userID string, outcome LoginOutcome, at units.Time) error
    TouchActivity(ctx, userID string, at units.Time) error
}

SessionStore interface {
    OpenSession(ctx, Session, tokenDigest string) error   // one statement, §6.5
    SessionByTokenDigest(ctx, digest string) (Session, string, bool, error)
    Sessions(ctx) ([]Session, error)
    ReplaceCapabilities(ctx, sessionID string, document []byte) error
    TouchSession(ctx, sessionID string, at units.Time) error
    RevokeTokensFor(ctx, userID, deviceID string) error
}
```

`RecordLoginOutcome` is one method rather than *"increment the counter"*, *"reset the counter"*,
*"set the disabled flag"* and *"stamp the login date"*, because §6.7's rule is a single transition
and four callers would be four chances to perform three quarters of it.

**Amended 2026-09-03, by T4, which wrote both ports.** ~~The block above writes both interfaces in
terms of `User`, `Credential`, `Session` and `LoginOutcome` without saying where those four types
live.~~ They live in **`internal/ports`**, and the task list said in terms that this was the plan's
open question and T4's to close. What forced the answer:

- **The alternative inverts the diagram.** A method returning `users.User` makes `ports` import a
  domain package, and [architecture §2](../../docs/architecture.md#2-layers-and-the-direction-of-dependency)
  puts `ports` at the bottom: it may import *"nothing of ours"* but the unit types, which its
  2026-09-03 amendment admits as the one exception because they are a leaf that imports nothing
  itself. `internal/users` is not a leaf. The rule that row is really stating — a port imports
  nothing that can reach a request, a status code or a store — is the same rule, and this is not a
  deviation from it, so no ADR is owed.
- **It would have been two domain packages, not one.** §3 splits `users` from `sessions`
  deliberately, and a single `ports` file naming both would re-join in the lowest layer exactly what
  the domain took care to separate — while making `sessions` reachable from every importer of the
  account store.
- **The cost is real and is the honest shape.** These are store records, so the policy and the
  configuration cross the boundary as **the bytes of their stored documents** rather than as
  `users.Policy` and `users.Configuration`. That follows §4 and §6.6 rather than working around
  them: a stored document decodes onto the reference's defaults and never onto Go's zero value, and
  `InvalidLoginAttemptCount` is overlaid from its own column afterwards. A port that handed the
  domain an already-decoded policy would have performed both rules inside the store, where the
  reason for them is invisible.
- **The store may still import `internal/users`, and does, in exactly one place.** The store is
  outward of `ports`, not inward of it, so the arrow is undisturbed. `RecordLoginOutcome`'s lockout
  is the caller: T1 shipped **no `is_disabled` column** — §4 keeps the flag in `policy_document` —
  so setting it is a read, a `DecodePolicy`, a change and an encode, inside the same transaction as
  the counter.

Three consequences of writing it that this section did not say and now does:

- **`LoginOutcome`'s zero value is not an outcome.** It is `LoginFailed`, `LoginSucceeded`,
  `LoginLockedOut` and an invalid zero that is refused. There is no safe default here — defaulting
  to a failure counts an attempt nobody made, defaulting to a success clears a lockout — which is
  the *opposite* decision from `Authentication{}` two blocks below, and for the opposite reason:
  there, the zero value is the safe answer and exists so that an unwired authenticator admits
  nobody. **Which failure reaches the threshold stays the domain's**, because
  `LoginAttemptsBeforeLockout` is §6.7's three-way sentinel; the store applies the transition it is
  handed.
- **`SessionByTokenDigest`'s second return value is the *token's* user, not the session's.** §6.5
  already argues that the two differ — a session is keyed on `(Client, DeviceId)` and names whoever
  authenticated there last, a token on `(user, device)` — so two people sharing one client on one
  device hold two live tokens against one session row. A caller resolved from the session's user
  would be whoever logged in most recently, on somebody else's account, with no error anywhere.
  That is why this method returns two values, and this is where it is written down.
- **Neither interface creates an account, and T4 did not invent one.** Provisioning is §6.9's and
  T7's, so the port method that writes a new `users` row is T7's to add and to amend this section
  with. Until it lands, the only thing that builds a `users` row is the test helper T1 wrote.

```
// internal/httpapi — the port 001 declared, filled
Access int  // AccessUnauthenticated | AccessGranted | AccessForbidden

type Caller struct {
    UserID    string
    SessionID string
    Policy    users.Policy
}

type Authentication struct {
    Access Access
    Caller *Caller   // nil unless Access is AccessGranted
}

Authenticator interface { Authenticate(*http.Request) (Authentication, error) }
```

**Amended 2026-09-03, by spec §3.8's parameter declaration.** `Visible` took ~~`(all []Session,
caller Caller)`~~ until this change, because the route took no parameters. It now takes the bound
`Selection` and the clock, and it stays **one function** rather than a filter the handler composes
with a visibility rule. That is the whole lesson of spec §3.8: the three apply in an order a client
can observe — `deviceId` before visibility, `activeWithinSeconds` after it — and two exported
functions are two chances to compose them the wrong way round, in a package whose test would still
pass because each half is right. `ActiveWithinSeconds` is an `int` and not a pointer: spec §3.8
makes `0` and absent the same request, so the zero value already means what the wire means.

**The invariant 001 built on is kept, and it is the reason the struct is shaped this way.** 001's
§6.10 relies on `AccessUnauthenticated` being the zero value, so that a nil authenticator — and any
future failure to wire one — admits nobody. `Authentication{}` still means *unauthenticated with no
caller*, so widening the return type moves nothing: the safe direction is still what you get for
free. `AccessForbidden` is the third value 001 reserved for this feature; §7 gives it the shape 001
said it had no measurement for, which [behaviours §1.11](../../docs/compatibility/behaviours.md#111-there-are-four-error-shapes-not-one)
has since measured.

**Why the caller travels with the access rather than through a second method.** A second interface
answering *"who is this"* would read the same five mechanisms a second time, and two reads of one
credential can disagree — a token that expired between them, or a second reader that never learned
about `X-Emby-Authorization`. One question, one answer.

**Why `Policy` is on the `Caller` and not fetched by the handler.** `GET /Sessions` branches on
`IsAdministrator` and the delivery routes 008 owns branch on `EnableMediaPlayback`; a handler that
had to fetch a policy would be a handler that could forget to. The value is a copy, and the domain
type — not a map — so a flag that does not exist does not compile.

```
// internal/users — no net/http in any signature
DefaultPolicy() Policy                       // the reference's constructor, not Go's zero value
Verify(record string, pw Plaintext) (ok bool, needsRehash bool, err error)
Derive(pw Plaintext) (record string, err error)

// internal/sessions
DeriveID(client, deviceID string) string     // §6.5

type Selection struct {                      // spec §3.8's three parameters, already bound
    DeviceID            string               // "" when absent or empty
    ControllableByUser  string               // "" when absent or empty
    ActiveWithinSeconds int                  // 0 when absent, and 0 means "do not apply"
}

Visible(all []Session, caller Caller, sel Selection, now units.Time) []Session
```

`Plaintext` is ADR-0006's type whose `String` and `slog.LogValue` return a redaction. It is declared
in `internal/users` because the domain is where it is verified, and **nothing outside that package
constructs one from anything but a request body or the provisioning command's stdin**.

## 6. Algorithms

### 6.1 Token extraction

Five mechanisms, one reader, run in the measured order and stopping at the first that yields a
non-empty token `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]`:

```
Authorization  >  X-Emby-Authorization  >  X-Emby-Token  >  ?ApiKey=  /  ?api_key=
```

1. `Authorization` and `X-Emby-Authorization` are read with **the same grammar** (§6.3) and yield
   the value of a `Token` component.
2. `X-Emby-Token` is the whole field value, trimmed of surrounding whitespace and nothing else.
3. The two query names are read **from the raw query string**, matched case-insensitively, first
   occurrence wins.

Three decisions inside those three lines.

**A header that is present but yields nothing does not stop the search.** A request carrying
`Authorization: Bearer x` and `?api_key=<good>` authenticates: the first header contributes no
`Token` component at all (§6.3 reads nothing out of a header whose scheme word is neither
`MediaBrowser` nor `Emby`), so it is not *"a mechanism that disagreed"*, it is a mechanism that was
not used. The measured precedence is between mechanisms that **each produced a token**, which is
what the probe sent pair by pair.

**The query names are read by this reader and not by query canonicalisation.**
[001 plan §6.2](../001-server-identity-and-discovery/plan.md#62-query-key-canonicalisation-behaviours-115)
keys a declared spelling by **route**, on the argument that the spelling that matters is *"the one
this server's own handler binds"*. No handler binds a credential: it is a property of the request,
accepted on every authenticated route in the project, including routes that are not this feature's.
Declaring `ApiKey` and `api_key` on all fifty-nine rows to make one stage cover them would be a
declaration nobody reads. So `httpapi.V1QuerySpellings()` stays empty until a feature declares a
real parameter, and this reader does its own case-insensitive lookup over the raw query — which is
also the only way it keeps working on a request whose route was never matched.

**`ApiKey` and `api_key` are two names, not one name in two cases.** They do not fold together, so
no precedence between them is needed and none is measured: [behaviours §2.4](../../docs/compatibility/behaviours.md#24-there-are-five-authentication-mechanisms-and-one-of-them-wins)
records that *"the two query spellings were never set against each other"*. If both are present and disagree, this reader
takes the first in the raw query string — stated here so it is a decision rather than a property of
a map, and named in §9 as the one place a client could observe an order nobody measured.

### 6.2 Which routes require a token, and which merely accept one

Requiring is per **route**, and the route asks. The table this feature ships:

| Route | Token |
|---|---|
| `POST /Users/AuthenticateByName` | Not read as a credential at all; the body is the credential. The client-identification header is **mandatory** here and nowhere else (§3.2) |
| `GET /Users/Public` | Not required, not read |
| `GET /Users/Me`, `GET /Users/{userId}`, `POST /Users/Configuration`, `GET /Sessions`, `POST /Sessions/Capabilities/Full` | Required |

A handler that requires one calls the authenticator, and answers `401` in
[behaviours §1.11](../../docs/compatibility/behaviours.md#111-there-are-four-error-shapes-not-one)'s
empty shape — `httpapi.WriteUnauthorized`, which 001 already wrote — for
`AccessUnauthenticated`, and the empty `403` of §7 for `AccessForbidden`. An `Access` it does not
recognise is a `500`, which is 001's rule at §6.10 and the test that fails the day a fourth value
is added without teaching the switch.

**`/Users/Public` reads no credential even though one may be present**, which matters because the
route is measured to answer the *same* bytes to an authenticated and an unauthenticated caller
([behaviours §3.5](../../docs/compatibility/behaviours.md#35-userspublic-discloses-every-users-policy-to-anyone--class-b-replicated)).
Making the handler ignore the credential is how that becomes a property rather than a coincidence:
there is no branch to get wrong.

### 6.3 The `X-Emby-Authorization` grammar

One parser, used for both header names, returning the four components of §3.2 and any `Token`.
It is a pure function from a string to a value, and it **never fails**: it returns what it could
read, and *"nothing at all"* is a legitimate answer.

```
scheme-word  1*(component)
component    name "=" ( quoted-string | bare )  [ "," ]
```

- The scheme word is compared **case-insensitively** against `MediaBrowser` and `Emby`. Any other
  word — or none — and **nothing is read out of the header**, not even components that would have
  parsed.
- Component names are matched **case-sensitively**, and a lowercase name is not a component.
- Whitespace **around the `=`** ends the parse of that component: it is not a component.
- Values may be quoted or bare; extra spaces after the scheme, a missing space after a comma, a
  space before one, a trailing comma and unknown components are all accepted, the last ignored.

`[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]`

**The two strict rules are the interesting half and the parser must not be kind about them.**
Accepting whitespace around `=` or a lowercase name would let a client be built against Atrium that
fails against the reference — [behaviours §6](../../docs/compatibility/behaviours.md)'s
non-improvement, and the reason §3.2 records that an earlier version of the specification had it
wrong.

**A missing `DeviceId` is fatal on exactly one route and on no header.** The parser does not raise:
a request to any other route carrying a header with no `DeviceId` is served normally, measured `200`
`[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]`. It is
`POST /Users/AuthenticateByName` that refuses, `400`, because that route needs the components to
open a session — which is the reference's own shape, where the four arguments are checked at the
session manager rather than at the parser
`[source: Emby.Server.Implementations/Session/SessionManager.cs:1589-1592 @ v10.11.11]`. Putting the
refusal in the parser is the one mistake that would refuse requests the reference serves, on every
route at once; [behaviours §2.13](../../docs/compatibility/behaviours.md) records it as *"the one
fatal case"*, and it is fatal to a route, not to a parse.

### 6.4 Verifying a password

ADR-0006 decides the algorithm, the parameters, the record, the ceiling and the four rules of the
verification path. What this plan adds is where each one lives, because they are easy to implement
in a place that undoes them.

1. **The ceiling is a buffered channel of four tokens in `internal/users`**, acquired around the
   derivation and nothing else. It is in the domain rather than at the handler because ADR-0006
   prices it as a **memory** reservation — four × 64 MiB live — and a limiter at the handler would
   not bound the provisioning command, which derives with the same parameters. Acquisition
   **blocks**: a `503` on this route is a status the reference does not send there, so a flood makes
   Atrium slow rather than wrong.
2. **The decoy is derived once, at start, from 32 random bytes, with the current constants**, and
   held in the `users` package. It is derived at start rather than lazily so that the first
   unknown-username request is not the one that pays for it — which would be an oracle in the first
   second of every process. ADR-0006's rule 2 is the reason it is not a literal.
3. **The verification runs before the outcome is known and after nothing.** The order at the login
   path is: find the account by folded name; if there is none, verify the decoy and discard the
   result; if the account is disabled or locked, refuse **without** verifying — those refusals are
   `403` and already disclose themselves by status, which is ADR-0006's own reasoning; otherwise
   verify.
4. **The comparison is `crypto/subtle.ConstantTimeCompare`**, which `golang.org/x/crypto/argon2`
   does not do for you: it returns a key, and comparing two keys is the caller's step.
5. **A record whose parameters are below the current constants is re-derived on success**, inside
   the same request, before the response is written. Not after: a background rewrite would be a
   write nothing waits for and nothing observes, and ADR-0006 is explicit that the successful login
   is the only moment the plaintext exists.

**The reference equalises nothing, and that is recorded rather than matched.** An unknown username
is refused before any derivation runs
`[source: Jellyfin.Server.Implementations/Users/UserManager.cs:575-582 @ v10.11.11]`, so the
reference *is* distinguishable with a stopwatch where Atrium is not. It is a divergence in the sense
that the two servers behave differently and it is not one in the sense Principle I means: no byte
moves, and the differential compares bytes.

**Amended 2026-09-03, by T5, which wrote the path.** Five things this section fixed the order of
without saying what the order *returns*, and each had to be decided to write it:

- **The path answers two refusals, not four.** `ErrCredentialsRefused` and `ErrAccountDisabled`,
  neither naming a status — the domain does not know what one is, and §7's table is the mapping.
  Collapsing "no such account" and "wrong password" into one value is not tidiness: it makes it
  impossible for a later handler to answer them differently, which is the distinction the decoy
  spends 52 ms a request to hide. **There is no third sentinel for a lockout**, because §6.7's
  lockout *is* the disabled flag being set: a locked-out account is disabled on every later attempt,
  and a distinct value would be a state nothing here can reach.
- **The attempt that reaches the threshold is still a credential refusal.** It is a wrong password;
  the lockout is what it *writes*, not what it answers. The refusal as *disabled* starts on the next
  attempt. That is the reference's own order — the disabled check runs before the failure is counted
  `[source: Jellyfin.Server.Implementations/Users/UserManager.cs:585-593,636-641 @ v10.11.11]` — and
  it is why §6.7's two rows are one state on the second try.
- **The threshold is compared against the counter *after* this attempt.** The reference increments
  and then compares `>=` `[source: Jellyfin.Server.Implementations/Users/UserManager.cs:636-641 @
  v10.11.11]`, so a threshold of two locks on the second failure and not the third. The count comes
  from the column and never from the stored document (§4, §6.6); reading it off the document would
  compare a fresh attempt against whatever number the document was written with, and would never
  lock.
- **A stored record this package could not have written is a fault, not a wrong password.** `Verify`
  reports it as an error, and the path passes it out rather than folding it into the refusal:
  answering `401` would tell a client to discard a credential that was fine over a corrupt row, and
  counting it as a failed attempt would lock an account out over one. It joins §7's
  *store-unreadable* row at `500`. A failed rehash write and a failed outcome write are the same
  class and answer the same way — a login whose transition could not be recorded has not happened.
- **The path answers the account as the store holds it afterwards**, by reading it back rather than
  by patching the copy in hand. The response carries `InvalidLoginAttemptCount` and `LastLoginDate`
  and both have just moved, and a patched copy would be a second spelling of the transition §5
  deliberately made one store method. **Whether the reference's `AuthenticationResult.User` reflects
  the login it just performed is unmeasured** — its update bypasses the entity it then serialises
  `[source: Jellyfin.Server.Implementations/Users/UserManager.cs:614-627 @ v10.11.11]`, which reads
  as though it answers the *previous* `LastLoginDate`, and on a first login that would be no member
  at all. It is one request to settle and it is the register's, at T23.

**And the fold is this package's, spelled down.** `users.Fold` reduces a presented username to the
`username_folded` column §4 declares; provisioning fills that column with the same function, and it
has to be the same one on both sides or an account becomes unauthenticatable with no error anywhere.
It lower-cases where the reference upper-cases
`[source: Jellyfin.Server.Implementations/Users/UserManager.cs:155-166 @ v10.11.11]`. The direction
is not observable — the fold is a store key and `Username` is what reaches the wire — but *which
names collide* can differ between the two on a handful of characters. v1 has no rename and
provisioning refuses a name whose fold is taken, so nothing can produce the disagreement today.

### 6.5 What an authentication writes: two identities and one replacement

`POST /Users/AuthenticateByName`, after §6.4 succeeds, performs one transaction:

1. **The token** is 16 bytes from `crypto/rand`, rendered as 32 lowercase hex — the shape §3.3
   measures, and the same 128 bits the reference mints
   `[source: src/Jellyfin.Database/Jellyfin.Database.Implementations/Entities/Security/Device.cs:29 @ v10.11.11]`.
   It is the one identifier in this project that must **not** be derived: Principle VII wants
   identifiers stable across a rescan so that caches and favourites survive, and a bearer credential
   that is a function of anything an attacker knows is not a credential.
2. **The session identifier is derived**, from the client name and the device identifier, as the
   first 16 bytes of SHA-256 over `Client`, a NUL, and `DeviceId`, in 32 lowercase hex. Principle VII
   and behaviours §1.4's shape.
3. **Every token for `(user, DeviceId)` is revoked**, which is §3.3's *"authenticating again from the
   same `DeviceId` replaces that session"*, and is the reference's own order — it logs out the
   existing devices for that user and device before it creates the new one
   `[source: Emby.Server.Implementations/Session/SessionManager.cs:1653-1681 @ v10.11.11]`.
4. **The session row is inserted or updated** at the derived identifier, with this user, this
   device name, this version and this remote endpoint.

**The session identifier is derived and is deliberately *not* the reference's derivation.** The
reference computes `MD5(Client + DeviceId)`
`[source: Emby.Server.Implementations/Session/SessionManager.cs:486-487,554 @ v10.11.11]`, and
reproducing it byte for byte is both possible and free — which is exactly why it has to be argued
rather than done. Three reasons not to:

- [`allowlist.yaml`](../../docs/compatibility/allowlist.yaml) already declares
  `POST /Users/AuthenticateByName` `/SessionInfo/Id` as a `derived-identifier` difference, and
  [AGENTS.md §3](../../AGENTS.md) makes a conformance assertion a **declared inequality**, where *a
  declared difference that has gone away fails too*. Agreeing with the reference here would put a
  three-way paired artefact — `allowlist.yaml`, `conformance.md` L3 and 010 §3.3 — into the change,
  and 001 already refused to write one third of that triple.
- [architecture §6](../../docs/architecture.md#6-state-and-the-store-boundary) settles the general
  question: *"reproducing the reference's exact identifier bytes is not a goal and never was;
  reproducing its stability is."*
- The reference's derivation has a collision its concatenation cannot see — `("ab", "c")` and
  `("a", "bc")` are one session — and copying a defect that no client can observe buys nothing under
  Principle V, whose class A/B/C procedure needs a client that depends on it.

**The session's key is `(Client, DeviceId)` and the user is a column, which is not quite the triple
§3.8 describes.** The reference keys a session on the client name and the device alone, and updates
its `UserId` when somebody else logs in there; it keys a *token* on `(user, device)`. So two users
sharing one device and one client have two tokens and **one** session row on the reference, and the
row names whoever authenticated last. Spec §3.8's *"a `(user, device, client)` triple"* reads as
what a session **contains**; §3.8's next sentence — *"identified by the `DeviceId`"* — is what it is
**keyed** on, and a derivation that included the user would answer two identifiers where the
reference answers one. The unmeasured half is what `/Sessions` then shows, and it is recorded as
[U-16](../../docs/compatibility/reference-target.md).

### 6.6 Building the user object

One function builds the §3.5 object, and every route that returns one calls it — `/Users/Public`,
`/Users/Me`, `/Users/{userId}` and the `User` member of the authentication result. This is 001's
*"the superset is structural"* argument at §6.9 applied to a body four routes share: the cheapest way
to guarantee that `/Users/Public` is byte-identical to the authenticated reading (§3.4, measured) is
for there to be one filler.

- **`Policy` is a Go struct in the reference's declaration order**, decoded from
  `policy_document` over `DefaultPolicy()`, with `InvalidLoginAttemptCount` overlaid from the column
  (§4). The reference declares **44** properties and sends **42**
  `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]`; the two that do not
  travel are `MaxParentalRating` and `MaxParentalSubRating`, nullable integers that are null on a
  default account and therefore omitted by
  [behaviours §1.7](../../docs/compatibility/behaviours.md)'s global rule
  `[source: MediaBrowser.Model/Users/UserPolicy.cs:112-114 @ v10.11.11]`. 44 − 2 = 42 is the
  arithmetic that makes the measured count a check on the model rather than a number to hit, and
  both are `*int` under ADR-0002.
- **`Configuration` is the same shape** over the 16 properties `[spec: UserConfiguration]`, with the
  same defaults rule. §3.6 says an unknown property is **ignored, not rejected**, so the decode drops
  it and AC-8's round-trip is over the declared set. That is the opposite of what the session's
  capabilities do (§6.10), and the two are opposite because the reference is:
  [behaviours §5.9](../../docs/compatibility/behaviours.md#59-an-unknown-capabilities-property-survives-into-sessions-here-and-not-there)
  is a recorded divergence on capabilities and there is no such divergence here.
- **`PrimaryImageTag` and `PrimaryImageAspectRatio` are always absent**, because v1 provides no way
  to give a user an avatar. §3.5 conditions both on the user having one; an installation where none
  does is the only installation this binary can be started as, and 006 owns the day that changes.
- **`ServerName` and `PrimaryImageAspectRatio` are null and therefore absent**, so their position in
  the order is unverifiable — §3.5 says so, and the model carries the note rather than an assertion.

**Amended 2026-09-03, by T2, which wrote the two models.** *"`Configuration` is the same shape"* is
true of the rule and not of the nullability, and two decisions this section did not take had to be
taken to write the structs:

- **Three of the sixteen configuration properties are nullable at the reference and are plain
  strings here.** `AudioLanguagePreference`, `SubtitleLanguagePreference` and `CastReceiverId` are
  `string?`, and the reference fills them per account — coercing the subtitle preference to the
  empty string, leaving the audio preference as the account holds it, and answering
  `CastReceiverId` with the first cast receiver application the installation has
  `[source: Jellyfin.Server.Implementations/Users/UserManager.cs:426-447 @ v10.11.11]`. Under
  §1.7 a null is omitted, so a fresh account read from the source alone would send **15**, and §3.6
  measured **16**. The measurement wins ([AGENTS.md §1.3](../../AGENTS.md)) and the three are
  therefore non-optional here, which makes sixteen travel. **What the reference puts in them is
  ⚠️ UNVERIFIED**, and `CastReceiverId` is the one this server cannot match on value: 001 answers
  `/System/Info` with an empty `CastReceiverApplications` because Atrium ships no cast receiver, so
  the only honest value is the empty string. This is the register's, at T23, not this section's to
  resolve.
- **`AccessSchedules` has an unspecified element type**, which is 001's rule for
  `CompletedInstallations` applied unchanged: v1 gives an operator no way to create one, so
  declaring the reference's element members would be a schema for a value nothing here can produce.
  The feature that ever fills the array declares the type it fills it with. §3.5's amendment already
  records that the reference enforces access schedules at authentication and that v1 does not.

**And the two documents are encoded by the standard encoder rather than by the serialiser.**
[architecture §2](../../docs/architecture.md#2-layers-and-the-direction-of-dependency) puts the
serialiser in the Edge and the account domain in the Domain, so the domain may not import it — and a
stored document is not a response: it is negotiated with nobody, escaped for nobody's decoder and
carries no content type. What the two encodings share is the **struct**, which is where the property
that matters lives: one declaration fixes both the stored order and the key order of the L3 body.

### 6.7 Disabled, locked out, and at the session ceiling

The three refusals that are not about the password, in the order the login path tests them:

| Test | Answer | Provenance |
|---|---|---|
| The account is disabled | `403`, 25 bytes | Measured `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-26]`, [behaviours §2.11](../../docs/compatibility/behaviours.md#211-a-disabled-account-is-refused-with-403-not-401) |
| The account is locked out | `403`, 25 bytes | v1's own decision (§3.3), and the source says why it is the right guess: lockout **sets the disabled flag** `[source: Jellyfin.Server.Implementations/Users/UserManager.cs:634-641 @ v10.11.11]`, so on the reference a locked-out account *is* a disabled one on every later attempt. Still open as OQ-5 |
| The user is at `MaxActiveSessions` | The spec's answer: the least recently used session is evicted and the login succeeds | **Contradicted by the reference's source**, which refuses instead — see below |

**Lockout counting.** `LoginAttemptsBeforeLockout` is not a count. The reference maps it
`-1 → never lock`, `0 → three`, anything else → itself
`[source: Jellyfin.Server.Implementations/Users/UserManager.cs:816-821 @ v10.11.11]`, and `-1` is
what it sends for a default account
`[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]` — so **the default account
never locks**. A build that read the property as a threshold would lock every account after minus-one
failures, which is either never or immediately depending on the comparison, and would be discovered
by a user rather than by a test. The counter resets to zero on a success and increments on a failure;
reaching the threshold sets `IsDisabled` in the policy document, which is why the lockout is
permanent until an operator clears it and why the second attempt afterwards is the *disabled* row
above rather than a lockout row.

**`MaxActiveSessions`: the spec evicts and the reference refuses.** Spec §3.8's lifecycle table and
AC-13 say the oldest session is evicted; the reference counts the user's sessions and throws
`SecurityException("User is at their maximum number of sessions.")`
`[source: Emby.Server.Implementations/Session/SessionManager.cs:1623-1629 @ v10.11.11]`, which the
exception filter turns into the `403` and the 25 bytes. The two answers differ on exactly one
request — the login that would be one too many — and they differ in the direction a client notices:
eviction logs another device out, refusal keeps it. **The specification is implemented as written**,
because [AGENTS.md §1.3](../../AGENTS.md) makes the running server the tie-breaker, there is none
here, and source evidence does not discharge a specification. It is recorded as
[U-13](../../docs/compatibility/reference-target.md#claims-this-project-asserts-and-has-never-measured--added-2026-09-03),
and it is one request to settle: two logins on a user whose `MaxActiveSessions` is 1. This is the
same shape as 001's §3.5 and §3.4 findings and it is recorded for the same reason.

`0` means unlimited, which the reference spells as the guard `maxActiveSessions >= 1` and sends as
the default `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]`.

### 6.8 What makes an installation set up, and why this feature has to decide it

**Nothing in v1 completes setup, and until something does, `/System/Info` admits everybody.** 001's
handler exempts the route while setup is outstanding — the reference's own policy, which succeeds on
`!IsStartupWizardCompleted` before it looks at a role
`[source: Jellyfin.Api/Auth/FirstTimeSetupPolicy/FirstTimeSetupHandler.cs:29-31 @ v10.11.11]` — and
001 T18 measured the consequence from the inside: dropping the exemption failed twelve tests,
*"because setup outstanding is the only state a v1 installation can be in"*. The reference completes
it at `POST /Startup/Complete`
`[source: Jellyfin.Api/Controllers/StartupController.cs:41-46 @ v10.11.11]`, which is **not** one of
[surface.yaml](../../docs/compatibility/surface.yaml)'s fifty-nine rows and never will be under
Principle VI, because no named consumer calls it.

Two things follow, and they are why this is a plan section and a spec amendment rather than a note.

**AC-14 is unprovable without it.** *"`GET /System/Info` answers `200` to a request carrying a token
this feature issued"* is met, on an installation whose setup is outstanding, by a request carrying
**no token at all** — the exemption admits it. A test written against such a server would be green,
named for AC-14, and would prove nothing: 001's closing audit found exactly that shape twice and
called it the general lesson for 002.

**And `StartupWizardCompleted` can never be `true`**, on the response every client enters the API
through — an **L3** row — while a reference stood up for a differential run has completed its own
wizard ([ADR-0007](../../docs/decisions/0007-a-container-runtime-for-the-reference-instance.md)
configures the instance *"over the reference's first-time-setup operations"*). That is an undeclared
difference on `/System/Info/Public` waiting for 010, on a boolean, and 001 could not have closed it:
it serves no route and no command that completes anything.

**So: an installation becomes set up when it is provisioned with its first account**, recorded once
by the instant, through the `MarkSetupComplete` port 001 wrote for exactly this and left with no
caller. It is idempotent at the caller: the provisioning command reads `Installation()` first and
calls it only when `SetupCompleted` is false, because the recorded instant is *when setup was first
completed* and a second write would be a lie about a fact no response currently exposes as anything
but a boolean. This is the first caller of `ports.Clock`, which 001's plan §5 declared and 001 never
wrote.

`spec.md` gains §3.9 for this, because it is observable and therefore WHAT. §11 records it.

### 6.9 Provisioning, and the three seats a run needs

Spec §2 puts creating, editing and deleting users **over HTTP** out of scope and says v1 manages
accounts *"through configuration"*. The reading taken here, stated because the word is broad: it
forbids an **admin API**, and leaves the mechanism to this plan. The mechanism is a **subcommand of
the same binary**:

```
atrium user add    --data-dir DIR --name NAME [--administrator] [--disabled] [--hidden]
                   [--no-password] [--max-active-sessions N] [--login-attempts-before-lockout N]
                   [--enable-media-playback=false]
atrium user list   --data-dir DIR
atrium user set-password --data-dir DIR --name NAME
```

with the password read from **standard input**, never from a flag. An argument vector is readable by
every process on the host — `ps` on macOS, `/proc/<pid>/cmdline` on Linux — so a `--password` flag
would put the one value ADR-0006 works to keep ephemeral into a place that outlives the command.
Running with no subcommand serves, exactly as today, so nothing about `atrium --data-dir …` changes.

Four things this shape buys, each of which decided it:

- **`conformance/` can use it.** That package may import nothing of ours and already builds and runs
  the binary (001 T16); a subcommand is a black box it can run before `startServer`. Without it,
  every criterion needing an account would be asserted one layer in — the failure 001's audit named.
- **A plaintext password never reaches disk.** A configuration file listing accounts and passwords
  would be a permanent secret in the data directory, and the obvious next step — committing it — is
  what [AGENTS.md §1.7](../../AGENTS.md) exists to catch.
- **The store stays the single source of truth.** A file that had to round-trip 44 policy properties
  faithfully would be a second schema, hand-edited, next to the one the wire is built from.
- **The three seats a differential run needs are producible.**
  [request-cases.yaml](../../docs/compatibility/request-cases.yaml) names `administrator`,
  `restricted` and `playback-denied`, and a run that cannot stand up the second answers twelve of
  twenty-three reads with the wrong seat. `--administrator`, the default (a plain account) and
  `--enable-media-playback=false` are those three.

**The user identifier is derived, not random**: 32 lowercase hex from the first 16 bytes of SHA-256
over the folded username. Principle VII, and it buys something concrete — an installation
provisioned twice with the same names has the same identifiers, so a golden that names a user is not
a golden that names one particular run. The cost is stated: **renaming an account would change its
identifier**, which is behaviours §1.4's library-root trap in miniature. v1 has no rename, so the
cost is not payable today, and the feature that adds one inherits this line and should read it
before it does.

### 6.10 Sessions: activity, capabilities, and what `/Sessions` answers

- **`LastActivityDate` advances on every authenticated request**, written from the authenticator,
  once, after the token resolves. It is written **at most once per session per second** — the value
  is a date on the wire and a busy client would otherwise turn every request into a write. That is a
  choice about frequency and not about the value.
- **`POST /Sessions/Capabilities/Full` replaces, never merges**, answers `204` with no body, and
  **stores the document the client posted whole** — which is where an unknown property survives into
  `/Sessions`, the recorded divergence of
  [behaviours §5.9](../../docs/compatibility/behaviours.md#59-an-unknown-capabilities-property-survives-into-sessions-here-and-not-there).
  Storing the raw document is what makes that divergence the *stated* one rather than an accident,
  and it is why this column is a document and the policy column's decode-and-drop rule does not
  apply to it.
- **`SupportsMediaControl` and `SupportsRemoteControl` are the server's judgement and are `false`**,
  while the client's own declaration is echoed inside `Capabilities` unchanged, including a `true`
  the server disagrees with. That is measured, not a gap
  `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]`. `PlayableMediaTypes` and
  `SupportedCommands` **are** hoisted from the declaration verbatim, so the hoist is a three-item
  list and not a copy of the object.
- **Visibility is a domain function over the whole list**: the caller's own sessions always, every
  session for an administrator. It takes the list and the caller rather than querying per caller, so
  the rule is a pure function with a table-driven test. ~~That is the whole of what this route
  decides.~~ **It decides three things now, in the order below** — the function's shape is §5's.

**This route is owed three parameters that this feature has not specified, and the task list must
not start before they are.** [behaviours §2.25](../../docs/compatibility/behaviours.md#225-get-sessions-three-filters-are-two-filters-and-a-visibility-rule)
carries `controllableByUserId`, `deviceId` and `activeWithinSeconds`, measured at 012's gate
`[probe: tools/probe_session_filters.py, Jellyfin 10.11.11, 2026-08-29]`, with the note that they are
*"specified in 002, in the change that adds them"* — and the video client **sends `deviceId` today**.
The measurement is in and the specification is not: §3.8 declares no parameter, so this plan cannot
plan a handler for them without writing WHAT into a HOW. It is named here, in §9, and in the report,
as the one thing 002's spec still owes before its task list is written. The half that makes it more
than tidiness is the security sentence: `deviceId` narrows the list **before** the visibility rule,
so a non-administrator naming somebody else's device gets an empty `200`, while
`controllableByUserId` naming anybody else is a `403`.

**Resolved 2026-09-03, in the change this plan asked for — and here is what the resolution costs
this plan.** [Spec §3.8](spec.md#38-sessions) now declares all three, and behaviours §2.25's
*"Atrium does: none of it"* is corrected with it. What that turns into here:

- **One ordered function, not a filter and a rule.** `sessions.Visible(all, caller, sel, now)` (§5)
  applies `sel.DeviceID` to the whole list, *then* the visibility rule, *then*
  `sel.ActiveWithinSeconds` when it is greater than zero. **The order is written this way because
  the reference's is, and not because a test can catch it the wrong way round** — spec §3.8 records
  what writing the criterion taught: `deviceId` and the visibility rule are predicates over one list
  and predicates commute, so no request tells the two sequences apart. What the test can assert is
  the combination — a request carrying `deviceId` *and* `controllableByUserId` still narrows on the
  device — and it is written under that name rather than under "the order", so nobody later reads a
  green test as proof of something it never checked.
- **The `403` is the handler's and the predicate is one line.** `controllableByUserId` naming
  anybody but the caller, from a caller who is not an administrator, is refused before the domain is
  asked at all — the reference raises it in the controller, ahead of the session manager
  `[source: Jellyfin.Api/Controllers/SessionController.cs:60-61 @ v10.11.11]`, and here it is
  §7's `WriteControllerRefusal` for the same reason 001 gives about shapes: a refusal decided in the
  domain would have to travel back out as a status the domain does not own. The predicate — *is this
  identifier the caller's own, or is the caller an administrator* — reads `Caller.Policy`, which §5
  already puts on the caller for exactly this branch.
- **`controllableByUserId` filters nothing here, and the plan does not pretend otherwise.** Spec
  §3.8 argues that the list is always empty because v1 attaches no control channel, so `Visible`
  returns early for a non-empty `sel.ControllableByUser` rather than implementing three clauses that
  cannot be observed. **The early return is where the surprise lives** — it is the one branch in
  this feature whose correctness is an argument rather than a comparison — so it carries the
  argument in a comment and the test asserts the *reason*: a fixture session that declares
  `SupportsMediaControl: true` is still absent from the answer, because the declaration is the
  client's and the flag is the server's (spec §3.8).
- **These are the first route-keyed entries `httpapi.QuerySpellings` gets.** 001 plan §6.2 shipped
  the canonicalisation stage with an **empty** map and said *"its first source arrives with the
  first parameter"*; this is that arrival, and it is three names on one route. §6.1's two query
  token names are not entries — they are read on every route rather than declared by one — so
  nothing was there to fold before this.
- **U-12 bites here first.** The register's row about a query pair carrying a semicolon says it is
  *"owed by the first feature that reads a query **value** rather than folding a name"*. §6.1
  already reads two, but a discarded `ApiKey` pair fails closed as a `401`; a discarded `deviceId`
  fails **open**, as a wider list the client cannot tell from a correct one. The row is not amended
  — it anticipated this — but this is the request that makes it worth a probe.

## 7. Failure handling

Every refusal in this feature, with the shape and where it comes from. The first three shapes are
001's; the last three are the ones 001 could not reach and this feature writes.

| Failure | Detection | Response | Recovery |
|---|---|---|---|
| No credential on a route that requires one | Authenticator | `401`, empty, `Content-Length: 0`, no `Content-Type`, no `WWW-Authenticate` — `WriteUnauthorized` | Client re-authenticates |
| An unknown or revoked token | Authenticator | The same `401`. **Indistinguishable from no credential, which is measured** | — |
| A live token whose user was disabled after it was issued | Authenticator | `403`, **empty, no content type** — the *policy* refusal shape of [behaviours §1.11](../../docs/compatibility/behaviours.md#111-there-are-four-error-shapes-not-one), measured at 009 T2, because this refusal happens before any handler runs. **Unmeasured on this path**: it is OQ-5's third clause | Operator re-enables |
| Unknown username, or a wrong password on an enabled account | Login path | `401`, `text/plain` with **no charset**, the 25 bytes `Error processing request.` — asserted as bytes | — |
| A disabled account, a locked-out account | Login path | `403`, the same 25 bytes. The status is the whole difference | — |
| A missing or unparseable `X-Emby-Authorization` on the login route | §6.3, then the route's own check | `400`, the same 25 bytes | Client sends the header |
| A body that is not JSON, or a required member missing, on the login route | Decode | `400`, RFC 9457 problem details, `application/json; charset=utf-8` — **not** `application/problem+json` — with `errors` keyed `"$"` and the action parameter's own name, which is `request` `[source: Jellyfin.Api/Controllers/UserController.cs:211 @ v10.11.11]`. This is behaviours §1.11's measured **rule** applied to this route, not a measurement of this route | — |
| `GET /Users/{userId}` naming nobody | Lookup | `404`, the JSON-encoded bare string `"User not found"`, 16 bytes, `application/json; charset=utf-8` — the same body whoever asked | — |
| `GET /Users/{userId}` with an identifier that is not one | Bind | `400`, the validation body keyed on the parameter's **declared** spelling: `{"userId": ["The value 'x' is not valid."]}` | — |
| `GET /Sessions` naming another user in `controllableByUserId`, from a caller who is not an administrator | Handler, §6.10, before the domain is asked | `403`, `text/plain` with no charset, the same 25 bytes — `WriteControllerRefusal`. The status and the media type are measured `[probe: tools/probe_session_filters.py, Jellyfin 10.11.11, 2026-08-29]`; **the bytes are §1.11's rule applied**, and are register U-18 | Caller names themselves, or an administrator asks |
| `GET /Sessions` with an `activeWithinSeconds` that is not an integer, or a `controllableByUserId` that is not an identifier | Bind | `400`, the same validation body as the row above, keyed on the parameter's own spelling. **⚠️ UNVERIFIED — register U-17**: spec §3.8 marks it, and it is the reading that §1.12 forgives a *token* and refuses a *type* | — |
| The store is unreadable while authenticating | Port error | `500`, empty — **never `401`**. 001's rule, and the reason is that a client told `401` discards a credential that was fine | — |
| Four verifications already in flight | §6.4 | **Waits.** Never a status | — |

**Three refusal writers join 001's four**, in the same file and for the same reason it gives —
a shape is defined as much by what it does not carry, and an absence cannot be restored by a later
stage: `WriteControllerRefusal(w, status)` for the 25 bytes, `WriteJSONMessage(w, status, message)`
for the bare string, and `WriteForbidden(w)` for the empty policy `403`. The problem-details writer
is a fourth and is a model rather than a shape, because it carries a `traceId`
([behaviours §1.11](../../docs/compatibility/behaviours.md#111-there-are-four-error-shapes-not-one)).

**A password never reaches a log record.** ADR-0006's type does half of it — `String` and
`slog.LogValue` redact — and the other half is a rule this plan states because the type cannot
enforce it: the request body is decoded into a struct that is **never logged whole**, and the
refusal bodies above are constants, so no error path can interpolate one. AC-11 is asserted by a
test that logs to a buffer and searches it for the password, over every refusal path.

## 8. Testing strategy

Each criterion becomes a named test at the level spec §6 declares, **against the running binary in
`conformance/` unless it cannot be, with the reason written down where it cannot**. That rule is
001's closing audit stated as a policy rather than rediscovered: *a criterion written about a request
is not met by a test about the mechanism that serves it, however good that test is.*

| AC | Where | How |
|---|---|---|
| 1 | L2, `conformance/` | Provision, authenticate, assert `200`, a 32-hex `AccessToken` by shape, and the `User`/`SessionInfo` members present. A golden holds the body with the two derived members stated (§8.2) |
| 2 | L1/L2 golden, `conformance/` | Three requests, three statuses, **one** golden body compared by all three — which is what makes "the same 25 bytes" an assertion rather than three assertions written alike. `Content-Type: text/plain` with **no** charset asserted as a field value |
| 3 | L2, `conformance/` | Table-driven: five mechanisms × two route classes (a required route, `/Users/Public`), plus the four precedence pairs in both directions, plus the grammar table of §6.3. The image and delivery classes are 006's and 008's and are named as not-this-feature's |
| 4 | L2, `conformance/` | The `401` half over the wire. ~~**The second half — "a valid token lacking permission is `403`" — has no request in this feature**~~ **Amended 2026-09-03: it has one, and it is AC-15's — `GET /Sessions?controllableByUserId=` naming somebody else, from a caller who is not an administrator, is a `403` at the wire. Both halves are `conformance/` now, and the paragraph below survives as the reason the *domain* assertion stays.** The reasoning it was recorded with, and which still holds of the other six routes: none of them gates on a permission, `/Users/{userId}` refuses nobody, and `/System/Info` admits any authenticated caller once setup is complete — its policy requires no administrator `[source: Jellyfin.Server/Extensions/ApiServiceCollectionExtensions.cs:78 @ v10.11.11]` and succeeds for any caller in the user role `[source: Jellyfin.Api/Auth/FirstTimeSetupPolicy/FirstTimeSetupHandler.cs:46-50 @ v10.11.11]`. It is proven at the domain over `AccessForbidden` — which stays, because the authenticator's own `403` for a token whose user was disabled has no wire request either way — and was recorded as a criterion **partly out of reach**, ~~not ticked~~ **which it no longer is** |
| 5 | L2, `conformance/` | Authenticate twice from one `DeviceId`; the first token is then `401` and `/Sessions` shows one row |
| 6 | L2, `conformance/` | Two fixtures: one with a hidden user (excluded, others whole) and one where every user is hidden (`[]`). Byte-compared against the authenticated reading of the same users |
| 7 | L2, `conformance/` | The caller matrix: every pair of seat and subject, the identifier nobody has, the malformed one, and no credential. **The administrator's object as read by a restricted stranger is compared byte for byte with the administrator's own reading** — the measurement §3.7 records is an equality, so the test is one |
| 8 | L2, `conformance/` | Post all 16 properties, read them back through `/Users/Me`; then post an unknown one and assert it is dropped and the declared ones survive |
| 9 | L1/L2, `conformance/` | Post capabilities, read the session; the hoist of `PlayableMediaTypes` and `SupportedCommands`; `SupportsMediaControl` `false` beside a declared `true` |
| 10 | L2, `conformance/` | A fixture account with `--login-attempts-before-lockout 2`: two failures, then the correct password answering `403`; and a second account where one success between failures resets the count |
| 11 | Unit, `internal/httpapi` and `internal/users` | Every refusal path with a logger writing to a buffer; the buffer must not contain the password. Plus a compile-time assertion that the plaintext type implements `slog.LogValuer` |
| 12 | L2, `conformance/` | `/Users/Me` whole, configuration and policy included |
| 13 | L2, `conformance/` | The `204` with no body; replacement rather than merge; `LastActivityDate` advancing across two requests; and the `MaxActiveSessions` row **as the specification writes it**, with §6.7's contradiction named in the test's own comment |
| 14 | L2, `conformance/` | §8.2 |
| 15 | L2, `conformance/` | Six requests on one fixture: `deviceId` in another case and matching a row; `deviceId=` empty; a `deviceId` nothing matches; `activeWithinSeconds` at `0`, at `-5` and at a value that excludes a row; `controllableByUserId` naming the caller; and naming somebody else from a restricted seat, whose `403` is byte-compared with AC-2's golden. **The combination is its own case** — `deviceId` and `controllableByUserId` together — because that is the only part of the order a request can see (§6.10) |

**Fixtures.** One provisioning helper in `conformance/`, calling the subcommand, producing: an
administrator with a password; a restricted non-administrator with a password; a hidden user; a
disabled user; an account with no password; and an account with a low lockout threshold. A second,
one-line fixture in which every user is hidden. They are built by the same command an operator runs,
which is the point: the fixture is not a back door into the store.

### 8.1 The timing equalisation, which is the one check ADR-0006's argument stands on

ADR-0006 records that *"the timing equalisation is specified here and asserted nowhere"* and hands
the test to this feature. It is two assertions and neither is sufficient alone:

- **The mechanism.** `internal/users`, over a counter the verifier increments: an authentication
  against a username that matches no account performs **exactly one** derivation, with the **current
  constants**, and discards it. The second clause is ADR-0006's rule 2 — a decoy pinned to old
  parameters becomes its own oracle the moment the constants are raised — and it is asserted by
  reading the decoy record's own PHC parameters and comparing them to the constants, which is a
  test that fails on the raise rather than on the mistake.
- **The wall clock.** `internal/users`, over the real verifier: nine unknown-username and nine
  wrong-password authentications, medians compared, and the assertion is that they differ by **less
  than a quarter of one derivation**. Nine is ADR-0006's own sample size for the same measurement.
  The margin is deliberately loose: what this test exists to catch is a **missing** derivation,
  which is a whole 52 ms gap on the machine that measured it, and a tight margin would make a
  scheduling hiccup look like a regression. A test that is flaky is a test somebody deletes.

**And the ADR is not edited when this lands, which is worth stating because the obvious tidy-up is
forbidden.** ADR-0006's *"the timing equalisation is specified here and asserted nowhere"* becomes
untrue at [T6](tasks.md), and [AGENTS.md §4](../../AGENTS.md) makes an accepted ADR immutable — a
wrong one is superseded, never edited. This one is not wrong: it records a decision as taken,
including what it owed at the time, and it already names 002 as where the debt is paid. The
discharge is recorded in the task list and here, not by striking a line in the record.
*(Added 2026-09-03, while writing the task list.)*

Both must be proven able to fail: an early return on the unknown-username path fails the second and
the counter assertion fails the first. **Neither is in `conformance/`**, and the reason is worth
stating rather than apologising for: at the wire both paths carry the same 52 ms of Argon2 plus the
network, so an HTTP-level timing test is the same assertion with more noise and eighteen times the
pipeline. What `conformance/` proves instead is the half that *is* a wire fact — that the two
refusals are byte-identical (AC-2) — and the two halves together are the disclosure ADR-0006
describes.

**The debt is paid, and here is what it cost.** T6 landed both halves in `internal/users`:
`TestAnUnknownUsernameDerivesOnceWithTheCurrentConstants` is the mechanism and
`TestTheTwoRefusalsCannotBeToldApartWithAStopwatch` is the wall clock. Measured on the machine that
wrote them, the two medians over nine samples each are **50.554 ms** for an unknown username and
**50.818 ms** for a wrong password — a difference of **264 µs** against a margin of **12.639 ms**,
so the check passes with roughly fifty times the headroom it needs
`[measurement: internal/users, Go 1.27.0 darwin/arm64, 2026-09-03]`. **ADR-0006 is not edited**: its
*"asserted nowhere"* line is now untrue, and [AGENTS.md §4](../../AGENTS.md) makes an accepted ADR
immutable — this paragraph and [T6](tasks.md) are where the discharge is recorded, so no reader of
the record is left thinking the debt is open.

Both halves were shown to fail, and the two mutations do not partition the way this section assumed.
Deleting the decoy verification collapses the unknown-username median to **375 ns** against
**49.760 ms** for a wrong password, failing the wall clock and the mechanism's count together. A
decoy derived at the previous constants (`m=19 MiB, t=2, p=1`, the OWASP-minimum row of ADR-0006's
table) fails the mechanism's parameter assertion on all three axes **and** the wall clock, at
**16.911 ms** against **50.061 ms** `[measurement: internal/users, Go 1.27.0 darwin/arm64,
2026-09-03]`. That second result is worth keeping: a stale decoy is not a *missing* derivation but a
*cheaper* one, so the wall clock sees it only while the parameter gap stays large, and the
deterministic half is what would catch a raise from `t=3` to `t=4`. Neither is sufficient alone, for
a sharper reason than this section originally gave.

**Neither half is behind a build tag or a `testing.Short` guard, and that is a decision rather than
an omission.** The twenty derivations cost **0.94 s**, taking `internal/users` from 3.710 s to
4.654 s `[measurement: go test -count=1 ./internal/users, 2026-09-03]` — a quarter more, on a
package that already spends most of its time on Argon2id. CI runs `go test ./...` with no flags, so
a `-short` skip would change nothing there while creating a second mode in which ADR-0006's only
evidence silently does not run, and nothing else in this repository uses `testing.Short`. **A check
this record's argument stands on is the wrong place to introduce an off switch**, and the loose
margin is what makes it safe to run unconditionally on shared hardware: the gap it must survive is
scheduling noise, and the gap it must catch is a whole derivation.
*(Added 2026-09-03, at T6.)*

### 8.2 What this feature owes 001, and how each half is discharged

001's closing audit put **AC-14** in this specification and rode two smaller notes on it.

- **AC-14 itself.** `conformance/` provisions an administrator, authenticates over HTTP, and sends
  the returned token to `GET /System/Info`, asserting `200` and the superset body. **It is only a
  proof because of §6.8**: on an installation whose setup is outstanding the same request succeeds
  carrying nothing, so the test provisions first — which completes setup — and its companion asserts
  that the *same* request **without** a token is now `401`. Two requests, one server, and the
  criterion is about the token rather than about the exemption.
- **001's `401` on `/System/Info` moves to `conformance/`.** 001 asserted it in `internal/httpapi`
  because it needs an installation whose setup is **complete** and 001 could not make one. The note
  says *"when this feature can complete setup over HTTP"*; 002 completes setup and not over HTTP,
  and the condition that actually mattered — that such an installation can be **stood up** — is met.
  The assertion moves; the `internal/httpapi` test stays where it is, because it is the only one that
  can see `Content-Length: 0` and the two absent headers over a real connection alongside a stubbed
  store, and losing it would trade a stronger check for a tidier directory. §11 amends the spec's
  wording.
- **`TestPingAnswersTheProductNameAndNotThisServersFriendlyName` keeps its caveat, and the reason
  changes.** The note says the caveat drops *"when the rename endpoint lands"*. It never lands: the
  reference renames at `POST /Startup/Configuration`
  `[source: Jellyfin.Api/Controllers/StartupController.cs:74-78 @ v10.11.11]`, which is not one of
  surface.yaml's fifty-nine rows and has no named consumer, so no v1 feature can discharge that
  precondition. What could discharge it is the friendly name becoming operator configuration — one
  more subcommand over the `SetServerName` port 001 wrote and nobody calls — and **this plan does not
  take it**: the name is 001's datum, adding a configuration surface for it is not this feature's
  decision to make on the way past, and the test already discriminates on a fresh installation whose
  name is `atrium`. §11 corrects the spec's wording so the note names a condition that can be met.

### 8.3 L3 is deferred, and this is the feature where that costs the most

`POST /Users/AuthenticateByName` is the first **L3** row this project implements. L3 needs a real
reference; [AGENTS.md §1.6](../../AGENTS.md) forbids any CI job from contacting or starting one, and
[ADR-0007](../../docs/decisions/0007-a-container-runtime-for-the-reference-instance.md) stands up a
single-use instance outside CI. So what this feature can prove is **L2 and byte-level goldens**, and
the differential half is a recorded gap — which is what spec §6's own row already says, and 001 did
the same for its two L3 rows.

What this plan can do is make the run meet declared differences rather than surprises.
[allowlist.yaml](../../docs/compatibility/allowlist.yaml) already carries **ten** rows on
`POST /Users/AuthenticateByName` — eight `derived-identifier`, covering both `Id`s, both
`ServerId`s, the `DeviceId`, the `SessionInfo` identifier and the `AccessToken`, and two
`wall-clock`, for `/User/LastActivityDate` and `/SessionInfo/LastActivityDate` — and **two** on
`GET /Sessions`, for `/-/UserId` and `/-/DeviceId`. That is the L3 route covered. Three gaps are
named here for 010 rather than written, because that file is one third of a three-way pairing and
001 already refused to write one third of it:

- **`/Users/Public`, `/Users/Me` and `/Users/{userId}` have no rows at all.** All three carry the
  same `User` object, whose `Id` and `ServerId` are derived and whose `LastLoginDate` and
  `LastActivityDate` are wall clocks. They are L2, so no run compares them until 010 chooses to —
  and when it does they are six undeclared differences on three routes, of shapes the authentication
  result's own rows already spell.
- **`LastLoginDate` has no row anywhere in the file**, on any route. `LastActivityDate` has two.
  The authentication result is the first body in the project to carry either, and the pair travels
  together in every user object.
- **`GET /Sessions` has no row for `/-/Id`**, which is the derived session identifier §6.5 argues,
  while the same value has one inside the authentication result.

**Neither is written here, and 001 gave the reason:** writing one third of a paired triple is how a
paired set drifts.

### 8.4 The L0 friction is real and the task order has to plan for it

001 T20 derived *"implemented"* rather than declaring it: **a feature the server serves any row of
must serve every row of it**, in both halves of the L0 check. The day this feature registers its
first route, both halves start requiring **all seven** of 002's `surface.yaml` rows, and go red until
the last one lands. That is the intended friction. The task list therefore either lands the seven
registrations in one change over handlers that already work, or accepts a red L0 in between — and
the choice is the task list's to make deliberately, named here so it is not discovered.

**Taken 2026-09-03, in [tasks.md](tasks.md): the seven registrations land in one change, at T17.**
The second option is not available in practice, because every task here opens a pull request that
has to go green on its own, and a red L0 is a red build. What it costs is written into the list
rather than hidden: T12–T16's handlers are asserted at the HTTP boundary in `internal/httpapi`
before any of them is reachable over the wire, and T18–T21 are where the criteria become assertions
about requests — which is the distinction 001's closing audit turned into this plan's §8 rule.

Adding a field to `httpapi.Handlers` also fails
`TestTheRegistrationCheckIsRunWithEveryHandlerAServerCanBeBuiltWith` until `everyHandler` fills it.
That is deliberate too, and it is one line.

## 9. Risks

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| `MaxActiveSessions` is implemented as the spec writes it and the reference refuses instead | **Certain**, if the source reading is right | One `403` a client sees as a success, and another device logged out | U-13, and one request settles it. §6.7 |
| ~~`GET /Sessions` ships with no parameters while the video client sends `deviceId` today~~ **Retired 2026-09-03: spec §3.8 declares all three** | ~~High~~ | A client's filter silently ignored, which behaviours §1.12 makes correct-looking | **Taken, not mitigated.** §6.10 carries what the declaration costs this plan; what remains is U-17 and U-18, two register rows and four requests |
| `POST /Users/Configuration` takes a `userId` the specification does not mention `[source: Jellyfin.Api/Controllers/UserController.cs:488-511 @ v10.11.11]` | High | An administrator naming another user updates **their own** configuration here and that user's there — a silent wrong write, not an error | Implemented as specified (the caller's own); recorded as U-14; one request settles both the parameter and its `403` |
| Two policy flags gate **authentication** in the reference and are in v1's unenforced 28 | Medium | A user restricted to the local network, or to a schedule, logs in from anywhere | Recorded as U-15 and as a spec amendment (§11); enforcing needs a refusal shape nothing has measured |
| Timing equalisation is asserted with a wall clock in CI | Medium | A flaky test somebody deletes, taking ADR-0006's only check with it | §8.1's margin is a quarter of a derivation and the sample is nine; the mechanism half is deterministic and catches the same regression |
| The user identifier is derived from the folded username | Low | A future rename would change every reference to that user | Stated in §6.9; v1 has no rename, and the feature that adds one inherits the line |
| The session identifier deliberately differs from the reference's | Low | An allowlist row that stays rather than shrinks | Argued in §6.5 against the alternative, which is a three-way paired edit |
| Four Argon2id verifications hold 256 MiB live | Medium | On the smallest host, beside SQLite and ffmpeg | ADR-0006 priced it and chose the ceiling; nothing here raises it |

## 10. Alternatives considered

**Authenticate in a pipeline stage and carry the caller in the request context.** The obvious shape,
and it loses three ways: six of the nine delivery and subtitle actions require no token, nor do the
image routes
([behaviours §2.10](../../docs/compatibility/behaviours.md#210-the-image-and-delivery-routes-accept-a-token-and-require-none)),
so the stage would have to know the per-route rule anyway; a `Caller` read out of a context is the
zero value in any handler tested without the stage, silently; and 001 rejected the identical shape
for profile negotiation, with the identical argument, at its §6.3.

**Provision accounts from a configuration file in the data directory.** Closest to the specification's
word *"configuration"*, and declarative. Rejected on ADR-0006: a file listing accounts and passwords
is a permanent plaintext secret in the data directory, and the version of it that avoids that — the
operator pastes a PHC record — is a worse interface than a subcommand. It would also be a second
schema for 44 policy properties, hand-edited, beside the one the wire is built from.

**Match the reference's session identifier, `MD5(Client + DeviceId)`.** Free, and it would make one
allowlisted difference disappear — which is the reason not to: a declared difference that has gone
away is a failure under [AGENTS.md §3](../../AGENTS.md), so the change is not one derivation but a
three-way paired edit across `allowlist.yaml`, `conformance.md` and 010's spec. Argued in §6.5, with
architecture §6's *"reproducing its stability, not its bytes"* as the general answer.

**Store the access token as it is issued, as the reference does.** One fewer hash per request and a
table anybody with the file can replay. §4 takes the digest; the cost is one SHA-256 per
authenticated request and the benefit is that ADR-0006's threat model covers tokens as well as
passwords.

**One `internal/auth` package holding accounts, sessions and the credential reader.** Fewer packages,
and it puts an `*http.Request` in the same package as the lockout rule.
[architecture §2](../../docs/architecture.md#2-layers-and-the-direction-of-dependency) forbids the
combination, and §3 above gives the positive reason: 007 and 008 reach for a session and must not
reach through a verifier to get one.

**Refuse over the verification ceiling with a `503`.** Simpler than queueing and self-documenting
under load. ADR-0006 rejected it and this plan inherits the rejection: latency is not a wire delta
and a status the reference never sends there is.

**Derive the user identifier randomly.** It would survive a rename. It would also make an
installation's identifiers a property of the run, so no golden could name a user without recording
one particular provisioning — the same trap 001 met with `installation-id` and solved by stating the
input rather than softening the comparison.

## 11. What this change amended in `spec.md`, and what forced each one

~~Four edits, all in this change~~ **Four edits in the change that wrote this plan, and a fifth in the
change that followed it** — all dated in the specification's front-matter `amended:` line.

1. **§3.9 is new — an installation becomes set up when it is provisioned.** Forced by AC-14: on an
   installation whose setup is outstanding, `/System/Info` admits a request carrying no token, so
   the criterion 001 handed here would be met by a request that proves nothing. It is observable —
   `StartupWizardCompleted` on the response every client enters the API through — so it is WHAT and
   belongs in the specification rather than in §6.8. A row is added to §4 for the same reason.
2. **§3.5 gains a note: two of the twenty-eight unenforced flags are not unobservable.**
   `EnableRemoteAccess` and `AccessSchedules` are enforced by the reference **at authentication**
   `[source: Jellyfin.Server.Implementations/Users/UserManager.cs:595-611 @ v10.11.11]`, so the
   sentence *"the unenforced flags all gate features v1 does not have"* is not true of them: v1 has
   the enforcement point, it is the login. Forced by reading the login path in order to write §6.7.
   [behaviours §5](../../docs/compatibility/behaviours.md)'s accepted-gaps row says the same thing
   and is **owed the same correction**, which is not taken here because that document is not this
   feature's.
3. **§5's two carried notes name conditions that can be met.** *"When this feature can complete
   setup over HTTP"* — it never will; nothing in v1's fifty-nine rows completes setup, and the
   condition that mattered is that such an installation can be stood up. *"When the rename endpoint
   lands"* — it never lands either; the reference renames at `POST /Startup/Configuration`, which is
   not a v1 row. Forced by §8.2, where both notes had to be discharged or explained.
4. **§7's OQ-6 records what the source says and stays open.** `LoginAttemptsBeforeLockout` is a
   three-way switch — `-1` never locks, `0` means three — read at the pinned tag. The row stays open
   because the running server is the tie-breaker and there is none here, but an implementer reading
   OQ-6 as *"nobody knows"* would write a threshold comparison against `-1`.

**And four rows were added to [reference-target.md](../../docs/compatibility/reference-target.md)'s
register**, U-13 to U-16: the `MaxActiveSessions` contradiction, the unspecified `userId` on
`POST /Users/Configuration`, the two authentication-gating policy flags, and what `/Sessions` shows
when two users share one device and client.

**Amended 2026-09-03, by the change that declares `GET /Sessions`' three request parameters — a
fifth spec edit, and the plan edits it forced.** §6.10 named the declaration as the one thing 002's
specification owed before its task list. It is taken, in its own change, and it is recorded here
beside the four above rather than only in the document it changed.

5. **Spec §3.8 declares `controllableByUserId`, `deviceId` and `activeWithinSeconds`, §5 gains
   AC-15, and §6's row for the route names the matrix.** Forced by
   [behaviours §2.25](../../docs/compatibility/behaviours.md#225-get-sessions-three-filters-are-two-filters-and-a-visibility-rule),
   which has carried the measurement since 2026-08-29 and says the three are *"specified in 002, in
   the change that adds them"*, and by the video client, which sends `deviceId` today. What a route
   accepts and what it does with it is WHAT, and §6.10 could not plan a handler across the gap.
   behaviours §2.25's *"Atrium does: none of it"* is corrected in the same change, and two cases the
   measurement does not reach are marked `⚠️ UNVERIFIED` in the specification and registered as
   **U-17** and **U-18** rather than answered.

**And it cost one claim, which is the finding.** The natural reading of §2.25 — and this plan's own
first wording — is that the *order* is observable. It is not: `deviceId` and the visibility rule are
predicates over one list, and predicates commute, so no request tells the two sequences apart. What
a client can see is that the two parameters naming somebody else's property answer differently — an
empty `200` and a `403` — and that `deviceId` still narrows a request that also carries
`controllableByUserId`. AC-15 and §8's row for it assert that and say so, because a criterion named
for the order would have passed while proving something else.

**Five sections of this plan moved with it**: §3's `internal/sessions` row, §5's `Visible` signature
and the new `Selection`, §6.10, §7's two new refusal rows, and §8's rows for AC-4 and AC-15. **§9's
`GET /Sessions` risk is retired** — it was the risk that this change would not be made, and AC-4's
second half stopped being partly out of reach on the same request.

**And two sections moved in the change that wrote [tasks.md](tasks.md), 2026-09-03. Neither is a
spec amendment**: writing the ordered steps taught nothing about WHAT, which is the outcome a plan
this heavily amended should hope for.

- **§8.4's open choice is closed.** It named the L0 friction and left *"either the seven
  registrations in one change or a red L0 in between"* to the task list. The list takes the single
  change, because a red build is not a state a task can merge from, and the paragraph now records
  which way it went and what it costs. A plan that leaves a choice open and never learns the answer
  is a plan whose reader has to go and find the code.
- **§8.1 states that ADR-0006 is not edited when the equalisation check lands.** The record's
  *"asserted nowhere"* line stops being true at T6 and stays in the record, because
  [AGENTS.md §4](../../AGENTS.md) makes an accepted ADR immutable. Without the sentence, the obvious
  and forbidden tidy-up is the first thing a reader of §8.1 would reach for.

**One thing this plan leaves open on purpose, and the task list says so rather than guessing.** §5
writes `UserStore` and `SessionStore` in terms of `User`, `Credential`, `Session` and
`LoginOutcome` without saying whether those types belong to `internal/ports` or to the domain
packages that declare the interfaces. The two are not equivalent —
[architecture §2](../../docs/architecture.md#2-layers-and-the-direction-of-dependency) lets `ports`
import nothing of ours but `internal/units`, so a method returning `users.User` inverts the arrow —
and T4 takes the decision and amends §5 with what forced it.
