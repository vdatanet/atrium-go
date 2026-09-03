---
feature: 002-authentication-users-and-sessions
title: Authentication, users and sessions — tasks
status: Draft
created: 2026-09-03
updated: 2026-09-03
plan_status_required: Accepted
---

# 002 — Tasks

Ordered. Each task is a reviewable change on its own, and states how you know it worked.

No task may say "implement the feature". If one does, it needs breaking down.

**The shape of this list follows the plan's shape, and the plan's shape is not the endpoint list.**
[plan §1](plan.md#1-approach) says 002 is *"seven endpoints, no new edge, and one new value"* — and
then names the two things that make it much more than seven handlers: a domain that did not exist
(accounts, credentials, sessions, the store under all of it) and the fact that **this is the first
feature that cannot be tested without standing up an installation**. So T1–T11 build the domain, the
store, the credential reader and the refusal shapes; T12–T16 are the seven handlers, which are small
because everything under them is already asserted; T17 routes them; T18–T21 are the wire evidence
and the debt 001 left here; T22–T23 close the documents.

**One ordering decision is taken here deliberately, because [plan §8.4](plan.md#84-the-l0-friction-is-real-and-the-task-order-has-to-plan-for-it)
says it is this list's to take.** Both halves of the L0 check derive *implemented* rather than
reading a list — 001 T20's rule, *a feature the server serves any row of must serve every row of
it* — so the moment one 002 route is registered, both halves start demanding **all seven** and every
intermediate state is a red build. Plan §8.4 offers two ways out: land the seven registrations in
one change over handlers that already work, or accept a red L0 in between. **The registrations land
in one change (T17)**, because the alternative is a pull request that cannot go green, and every
task here has to be mergeable on its own. The cost is that T12–T16's handlers are asserted at the
HTTP boundary in `internal/httpapi` before they are reachable over the wire, and the wire evidence
follows in T18–T21 — which is why those five tasks say what they prove and T18–T21 say what they
still owe.

## Legend

`[ ]` not started · `[~]` in progress · `[x]` done · `[!]` blocked (say by what)

---

## T1 — `ports.Clock`, and the precious migration this feature owns

- [x] **Changes:** `internal/ports` — `Clock`, which [001 plan §5](../001-server-identity-and-discovery/plan.md)
  declared and 001 never wrote; this feature is its first caller (plan §6.8).
  `internal/store/sqlite` — `0002_users_and_sessions.sql` in the **precious** lineage, creating the
  four tables of [plan §4](plan.md#4-data-model) with their keys, their uniqueness and their foreign
  keys.
- **Depends on:** —
- **Verified by:** the migration applies on an empty data directory and the **precious** schema
  version advances by one while the derived one does not move — the assertion that fails if the file
  is filed under the wrong lineage, which is the one mistake here a rescan would punish and no test
  of the SQL would catch; a second start applies nothing; two accounts whose usernames differ only in
  case are refused by `username_folded`'s uniqueness, because that column exists precisely so the
  login's assumption is the database's rule rather than a convention; an `access_tokens` row naming a
  session that does not exist is refused by the foreign key; and `last_playback_check_in_at` is
  `NOT NULL` with the zero tick a writable value, since spec §3.3 measures
  `0001-01-01T00:00:00.0000000Z` for a session that has never played anything and a nullable column
  would answer an absence instead.
- **Spec reference:** §4; [ADR-0003](../../docs/decisions/0003-sqlite-as-the-store.md); plan §4.

## T2 — The policy and the configuration, and the rule that a stored document decodes onto the reference's defaults

- [x] **Changes:** `internal/users` — `Policy` in the reference's declaration order, `Configuration`
  over spec §3.6's sixteen properties, and the constructors both decode over
  (`DefaultPolicy`, and the same rule for configuration). Declaration order is wire order, so these
  are Go structs and adding a property is a code change and not a migration (plan §4).
- **Depends on:** —
- **Verified by:** a policy document holding **one** property decodes with `EnableMediaPlayback`
  true and `LoginAttemptsBeforeLockout` `-1` — the test that fails when the decode starts from
  `Policy{}`, which is Go's default and is silently wrong in the direction that locks every account
  or none (plan §4); the encoded policy carries **44** members and the body written through
  `internal/wire` carries **42**, the absent two being `MaxParentalRating` and `MaxParentalSubRating`
  under [behaviours §1.7](../../docs/compatibility/behaviours.md)'s global null rule — `44 − 2 = 42`
  is the arithmetic that makes the measured count a check on the model rather than a number typed in
  to match `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]`
  `[source: MediaBrowser.Model/Users/UserPolicy.cs:16-68,112-114 @ v10.11.11]`; the order is
  asserted as the **byte order of the encoded document**, not as a set, because L3 compares bytes and
  a set comparison passes on a reordered model; and an unknown property in a stored configuration
  document is dropped while the declared ones survive, which is spec §3.6's *ignored, not rejected*
  and is the opposite of what the session's capabilities do (plan §6.6, §6.10).
- **Spec reference:** §3.5, §3.6; plan §4, §6.6.

## T3 — Argon2id: the record, the ceiling, the decoy, and a plaintext that cannot be logged

- [x] **Changes:** `internal/users` — `Derive`, `Verify`, the `Plaintext` type,
  [ADR-0006](../../docs/decisions/0006-password-hashing.md)'s PHC record, the buffered ceiling of
  four, and the decoy derived once at start from 32 random bytes with the current constants.
- **Depends on:** —
- **Verified by:** `Derive` then `Verify` round-trips and a wrong password does not verify; a record
  derived below the current constants reports `needsRehash` and one at them does not; **the decoy's
  own PHC parameters equal the current constants**, which is ADR-0006's rule 2 written as a test
  that fails on the day somebody raises the constants without the decoy following — a decoy pinned
  to old parameters is its own oracle, and nothing else in the suite would notice; five concurrent
  derivations never have more than four in flight and **none is refused**, because ADR-0006 chose
  queueing over a `503` and a limiter that refused would be a status the reference does not send
  there; the comparison is `crypto/subtle.ConstantTimeCompare`, asserted by a record whose stored
  key differs from the derived one in its **last** byte failing exactly as one differing in its
  first does; and `Plaintext` formatted with `%v`, `%s` and through `slog` yields the redaction and
  never the password, beside a compile-time assertion that it implements `slog.LogValuer`.
- **Spec reference:** §3.3; ADR-0006; plan §6.4.

## T4 — The two store ports, and their SQLite half

- [x] **Changes:** `internal/ports` — `UserStore` and `SessionStore` as [plan §5](plan.md#5-contracts)
  writes them. `internal/store/sqlite` — the readers and writers behind both.
- **Note the plan left one thing open and this task takes it.** §5 writes both interfaces in terms
  of `User`, `Credential`, `Session` and `LoginOutcome` without saying where those types live, and
  the two candidates are not equivalent: a port method returning `users.User` would make `ports`
  import a domain package, which inverts
  [architecture §2](../../docs/architecture.md#2-layers-and-the-direction-of-dependency)'s arrow —
  `ports` may import nothing of ours but `internal/units`. Decide it, and **amend plan §5 in the
  same change** with what forced it, the way 001 T3 and T4 amended theirs.
- **Depends on:** T1, T2
- **Verified by:** `RecordLoginOutcome` is **one** transition, asserted over three cases read back
  through `UserByID` — a failure increments the counter and moves nothing else; a success resets it
  to zero and stamps `last_login_at`; reaching the threshold sets `IsDisabled` **in the stored policy
  document**. It is one method for the reason plan §5 gives, and the test that makes that worth
  anything is the third: a build performing three quarters of the transition passes any test written
  per field. `OpenSession` is one statement — an injected failure mid-way leaves neither the session
  row nor the token digest, so no token can exist without its session. `RevokeTokensFor(user,
  device)` removes exactly that pair's tokens and leaves the **same user's token on another device**
  live, which is the clause spec §3.3's *"replaces that session"* turns on and the one an
  over-broad `DELETE ... WHERE user_id = ?` would break invisibly.
- **Spec reference:** §3.3, §3.8, §4; plan §4, §5.

## T5 — The login path: the order the refusals are tested in, and the three-way lockout switch

- [x] **Changes:** `internal/users` — the login path of [plan §6.4](plan.md#64-verifying-a-password)
  and [§6.7](plan.md#67-disabled-locked-out-and-at-the-session-ceiling): find by folded name; verify
  the decoy and discard when there is no account; refuse a disabled or locked account **without**
  verifying; otherwise verify, and re-derive on success inside the same call.
  `LoginAttemptsBeforeLockout` is read as the reference's three-way switch — `-1` never locks, `0`
  means three, anything else is itself
  `[source: Jellyfin.Server.Implementations/Users/UserManager.cs:816-821 @ v10.11.11]`.
- **This task also corrects [behaviours §5](../../docs/compatibility/behaviours.md)'s accepted-gaps
  row**, which still reads *"All of them gate features v1 lacks"* about spec §3.5's twenty-eight
  stored-and-unenforced policy flags. Spec §3.5 was amended on 2026-09-03 and says the row is owed
  the same correction: `EnableRemoteAccess` and `AccessSchedules` are enforced by the reference **at
  authentication**
  `[source: Jellyfin.Server.Implementations/Users/UserManager.cs:595-611 @ v10.11.11]`, which v1 has,
  so the sentence holds for **26** of the 28. It lands here rather than in a documentation task
  because this is the code that decides not to enforce them, and Principle III moves the
  documentation in the same commit. Amend in place, dated, citing
  [U-15](../../docs/compatibility/reference-target.md).
- **Depends on:** T3, T4
- **Verified by:** a username matching no account performs **exactly one** derivation and refuses,
  counted over T3's verifier; **a disabled account and a locked-out account refuse with no
  derivation at all**, asserted over the same counter — which turns ADR-0006's *"no equalisation is
  owed there either"* into a property of the code rather than a sentence in a record; an account
  whose threshold is `-1` does not lock after fifty failures and one whose threshold is `0` locks
  after **three**, which are the two readings a plain threshold comparison gets exactly backwards;
  an account whose threshold is `2` locks after two failures and a success between them resets the
  count; and the attempt **after** a lockout is refused as *disabled*, because the lockout set the
  flag — so the two rows of plan §6.7 are one state on the second try and a test that expected a
  distinct "locked" answer would be asserting something this design does not produce.
- **Spec reference:** §3.3, §3.5, §7 (OQ-5, OQ-6); plan §6.4, §6.7.

## T6 — The timing equalisation, which is the one check ADR-0006's argument stands on

- [x] **Changes:** two tests in `internal/users` and no production code.
  [ADR-0006](../../docs/decisions/0006-password-hashing.md) records that *"the timing equalisation is
  specified here and asserted nowhere"* and hands the check to this feature by name, calling it *"the
  one check this record's argument stands or falls on"*. [plan §8.1](plan.md#81-the-timing-equalisation-which-is-the-one-check-adr-0006s-argument-stands-on)
  says what the two halves are and why neither is sufficient alone.
- **The ADR is not edited, deliberately.** Its *"asserted nowhere"* line becomes untrue in this
  change, and [AGENTS.md §4](../../AGENTS.md) makes an accepted ADR immutable — a wrong one is
  superseded, never edited. This one is not wrong: it is a record of a decision as taken, including
  what it owed at the time, and it already names 002 as where the debt is paid. **Say so in the
  commit and in plan §8.1** rather than leaving a reader to think the debt is open.
- **Depends on:** T5
- **Verified by:** the **mechanism** half — an authentication against a username that matches no
  account performs exactly one derivation with the **current** constants, read off the decoy record's
  own PHC parameters; the **wall-clock** half — nine unknown-username and nine wrong-password
  authentications, medians compared, differing by less than a quarter of one derivation. Nine is
  ADR-0006's own sample size and the margin is loose on purpose: what this catches is a **missing**
  derivation, which is a whole derivation's gap, and a tight margin turns a scheduling hiccup into a
  red build somebody deletes. **Both halves must be shown to fail**: an early return on the
  unknown-username path fails the wall clock, and a decoy derived at the previous constants fails the
  parameter assertion. A timing test that has never failed has proved nothing, and this one is the
  whole of ADR-0006's evidence.
- **Spec reference:** ADR-0006 §"What is not verified, and is owed"; plan §8.1.

## T7 — `atrium user add`, and the account that completes setup

- [x] **Changes:** `internal/app` — the three subcommands of [plan §6.9](plan.md#69-provisioning-and-the-three-seats-a-run-needs),
  with the password read from **standard input** and never from a flag; `cmd/atrium` — one dispatch
  on the first argument and nothing else (plan §3). The user identifier is derived from the folded
  username. The first account records setup completion once, through `MarkSetupComplete` and
  `ports.Clock`, idempotent at the caller (plan §6.8).
- **It is this early because `conformance/` imports nothing of ours.** Without a black-box way to
  make an account, every criterion in spec §5 would be proven one layer in — which is exactly the
  shape 001's closing audit caught itself in twice. It also produces the three seats
  [request-cases.yaml](../../docs/compatibility/request-cases.yaml) names.
- **Plan §5 had no `CreateUser` and this task adds it**, which T4 recorded as out of its scope:
  nothing in either store interface made a `users` row, and the only thing that built one was a test
  helper. The method is added and **plan §5 is amended in the same change**, the way T4 amended it.
  Plan §6.8 and §6.9 gain amendments too, for the two things writing the command measured.
- **Depends on:** T4, T5
- **Verified by:** `user add` on an empty data directory creates the account and `Installation()`
  then reports `SetupCompleted` true; a **second** `user add` does not move the recorded instant,
  read back in SQL — the idempotence is at the caller, so a test that only checked the boolean would
  pass on a build that rewrote the instant every time; the parsed flag set contains **no**
  `--password`, asserted over the flags themselves rather than by reading the source, because an
  argument vector is readable by every process on the host and that is the whole reason for stdin;
  provisioning two directories with the same names yields **byte-identical** identifiers, which is
  Principle VII and what lets a golden name a user without recording one particular run;
  `--administrator`, the bare default and `--enable-media-playback=false` produce the three seats,
  asserted by reading the stored policy back rather than by the command exiting zero;
  `atrium --data-dir …` with **no** subcommand still serves, which is the regression a first-argument
  dispatch is most likely to introduce; and — in `conformance/`, against the running binary — a
  server started on a **provisioned** directory answers `StartupWizardCompleted: true` on
  `GET /System/Info/Public` where one started on an unprovisioned directory answers `false`. That
  last row is a wire assertion with no 002 route registered, and it is what makes T21 possible at
  all.
- **Spec reference:** §3.9, §4; plan §6.8, §6.9.

## T8 — The credential reader and the `X-Emby-Authorization` grammar

- [x] **Changes:** `internal/httpapi` — one reader over the five mechanisms of spec §3.1 in the
  measured order ([plan §6.1](plan.md#61-token-extraction)), and one parser for the grammar of spec
  §3.2, used for **both** header names, which never fails and returns what it could read
  ([plan §6.3](plan.md#63-the-x-emby-authorization-grammar)). Both cores are pure functions over
  strings, so the grammar table is a table test and not a request per row.
- **Depends on:** —
- **Verified by:** spec §3.2's grammar table row by row, with the two **strict** rows asserted as
  absences rather than as different values — whitespace around the `=` and a lowercase component
  name each yield *no component*, and a parser that was kind about either would let a client be
  built against Atrium that fails against the reference
  ([behaviours §6](../../docs/compatibility/behaviours.md)); a scheme word that is neither
  `MediaBrowser` nor `Emby` yields **nothing at all, even when every component would have parsed**,
  which is the case a lenient parser passes and the reference refuses; the four precedence pairs of
  spec §3.1 in **both** directions; `Authorization: Bearer x` beside `?api_key=<good>`
  authenticating — plan §6.1's *"a header that is present but yields nothing does not stop the
  search"*, and the row a `if hasHeader { return }` implementation fails; `ApiKey`, `api_key` and
  `APIKEY` all read off the **raw** query with the first occurrence winning; and a request whose
  path matches no route still yielding its token, since this reader deliberately does not go through
  query canonicalisation (plan §6.1) and a reader that did would stop working on exactly the
  requests it is most needed for.
- **Spec reference:** §3.1, §3.2; plan §6.1, §6.3.

## T9 — `internal/sessions`: the derived identity, and one ordered `Visible`

- [x] **Changes:** `internal/sessions` — `DeriveID(client, deviceID)` of [plan §6.5](plan.md#65-what-an-authentication-writes-two-identities-and-one-replacement),
  the `Selection` of plan §5, and `Visible(all, caller, sel, now)` as **one** function applying
  `deviceId`, then the visibility rule, then `activeWithinSeconds`.
- **Depends on:** T4
- **Verified by:** `DeriveID` is 32 lowercase hex, stable across processes, and gives `("ab", "c")`
  and `("a", "bc")` **different** identifiers — the reference's own concatenation collides there and
  plan §6.5 argues deliberately against copying it, so the divergence is asserted rather than
  assumed. Then `Visible`, table-driven over spec §3.8: `deviceId` matched **without regard to
  case**; `deviceId=` empty treated as *absent* rather than as a device nothing is named after;
  `activeWithinSeconds` at `0` and at `-5` answering the **unfiltered** list and at a positive value
  excluding a row whose `LastActivityDate` is older; an administrator seeing every session and
  everybody else only their own; a non-administrator naming **another user's device** answered `[]`;
  a non-empty `ControllableByUser` answering `[]` **even for a session that declares
  `SupportsMediaControl: true`** — which asserts the *reason* spec §3.8 gives (the declaration is the
  client's, the flag is the server's) rather than the emptiness, and is the one branch in this
  feature whose correctness is an argument rather than a comparison; and the **combination**,
  `deviceId` together with `controllableByUserId`, as its own case, because plan §6.10 makes that the
  only part of the order a request can see. **One thing is deliberately not asserted and the test
  file must say so**: that `deviceId` runs before the visibility rule. Predicates over one list
  commute, no request tells the two sequences apart (spec §3.8), and a case named for the order would
  pass while proving something else — which is the claim writing AC-15 already cost this feature
  once.
- **Amended 2026-09-03, by the task itself, in two places and both in [plan.md](plan.md).** §5 owed
  a decision this line reads straight past: `Visible`'s `Caller` and `Authentication`'s `Caller`
  cannot be one type, because `internal/sessions` may import neither the edge that declares the
  second nor the `internal/users` its `Policy` comes from. It is `sessions.Caller{UserID,
  IsAdministrator}` and the edge reduces its own to it in one line at the call site; §5 carries the
  argument. And §6.10's *"a request carrying `deviceId` and `controllableByUserId` still narrows on
  the device"* is **not observable either** — the early return answers `[]` whatever `deviceId`
  narrowed to, so the two sequences agree there as well. That is the same claim spec §3.8 already
  lost one paragraph earlier, written a second time in the plan and caught only because the case was
  made to fail. The case stays, under the same name, for the two wrong builds it does catch: one
  that ignores the parameter and one that lets it widen the list back.
- **Spec reference:** §3.8, AC-15; plan §5, §6.5, §6.10.

## T10 — The `Authenticator`, filled and widened

- [x] **Changes:** `internal/httpapi` — `Access` gains `AccessForbidden`, the third value 001
  reserved for this feature *"with the shape"*; `Authentication` and `Caller` of plan §5;
  `Authenticate` reads T8's credential, digests it, resolves a session and a user, and advances
  `LastActivityDate` at most once per session per second (plan §6.10).
- **Depends on:** T4, T8, T9
- **Verified by:** `Authentication{}` still means *unauthenticated with no caller*, asserted as the
  zero value — [001 plan §6.10](../001-server-identity-and-discovery/plan.md) relies on it so that a
  nil authenticator and any future failure to wire one admit **nobody**, and widening the return type
  is only safe because that stays true; a request with no credential and one with an unknown token
  are indistinguishable both at this port and at the wire, which is measured and is what plan §7's
  first two rows say; a live token whose user has since been disabled answers `AccessForbidden` with
  a **nil** caller, so a handler that read the caller anyway would not compile into a wrong answer;
  a store error is an error and never `AccessUnauthenticated`, asserted through 001's `/System/Info`
  handler answering `500` rather than `401` — a client told `401` discards a credential that was
  fine; and two authenticated requests inside one second write `LastActivityDate` once while two a
  second apart write twice, read back from the store, since that is a decision about frequency and
  a test asserting *"it advanced"* would pass on a build that wrote on every request.
- **Amended 2026-09-03, by the task itself, in two places and both in [plan.md](plan.md).** §5 owed
  a function §4 had written as a property of a column: the digest has **two** callers — the login
  that mints a token and this port, which resolves one — and a digest spelled twice is two digests
  whose disagreement makes every credential this server issues fail to authenticate with no error
  anywhere. It is `sessions.TokenDigest`, in the domain rather than at the edge, and it is not
  truncated the way `DeriveID` is. And §6.10's activity bullet named a rule for a date without
  saying **which** — the one written here is the *session's*, while `ports.UserStore.TouchActivity`
  still has no caller and spec §3.5's `LastActivityDate` on the **user** object would therefore be
  absent where the reference sends one. That rule is not this task's to invent: the reference
  throttles a user at sixty seconds and a token at three minutes, neither measured here, and
  whichever task first serves the user object owes it.
- **Spec reference:** §3.1, §3.8; plan §5, §6.10; [001 plan §6.10](../001-server-identity-and-discovery/plan.md).

## T11 — The three refusal shapes 001 could not reach

- [x] **Changes:** `internal/httpapi/refusal.go` — `WriteControllerRefusal(w, status)` for the 25
  bytes, `WriteJSONMessage(w, status, message)` for the JSON-encoded bare string, and
  `WriteForbidden(w)` for the empty policy `403`. Beside 001's four, in the same file and for the
  reason it gives: a shape is defined as much by what it does **not** carry, and an absence cannot be
  restored by a later stage.
- **Depends on:** —
- **Verified by:** over a **real connection** rather than a recorder — 001 T11 measured that three of
  the four things [behaviours §1.11](../../docs/compatibility/behaviours.md#111-there-are-four-error-shapes-not-one)
  states about a refusal shape are invisible to `httptest.ResponseRecorder` — that
  `WriteControllerRefusal` sends exactly the 25 bytes `Error processing request.`, `Content-Type:
  text/plain` with **no charset parameter**, and `Content-Length: 25`; that `WriteJSONMessage` sends
  the 16 bytes `"User not found"` under `application/json; charset=utf-8`; that `WriteForbidden`
  sends no body and **no content type at all**, which is the policy refusal measured at 009 T2 and is
  a different shape from the controller's `403` on the same status; and that the three differ from
  one another and from 001's four in at least one header each, so a copy-paste that gave one
  another's headers fails here rather than in a differential run a year later.
- **Spec reference:** §3.3, §3.7, §3.8; plan §7; behaviours §1.11.

## T12 — `POST /Users/AuthenticateByName`

- [x] **Changes:** `internal/httpapi` — the handler, its request model bound under the reference's own
  parameter name `[source: Jellyfin.Api/Controllers/UserController.cs:211 @ v10.11.11]`, the
  transaction of plan §6.5 (mint the token, derive the session identifier, revoke the pair's tokens,
  insert or update the session row) and the `AuthenticationResult` body of spec §3.3.
- **Depends on:** T5, T9, T10, T11
- **Verified by:** at the HTTP boundary in `internal/httpapi` — `200` carrying a 32-lowercase-hex
  `AccessToken` and a `SessionInfo.LastPlaybackCheckIn` of exactly
  `0001-01-01T00:00:00.0000000Z`, asserted as **that value** rather than as a present field, because
  it is a value and not an absence (spec §3.3) and 001 T4 left the zero `units.Time` able to be
  mistaken for one; a username matched **case-insensitively**, through the folded column rather than
  by lowering in the handler; the four measured refusals of spec §3.3 asserted as **bytes** against
  **one** golden body compared by all of them — which is what makes *"they carry the same 25 bytes"*
  a single assertion instead of four written alike, and what fails when one of them drifts; a body
  that is not JSON keeping the **problem-details** shape keyed `"$"` and the action parameter's own
  name, which is the one refusal on this route that is not the 25 bytes; authenticating twice from
  one `DeviceId` leaving **one** session row with the first token unusable; and a
  `X-Emby-Authorization` carrying no `DeviceId` refused `400` **here** while the identical header is
  served `200` on another route — behaviours §2.13's *"fatal to a route, not to a parse"*, and the
  mistake plan §6.3 says would refuse requests the reference serves on every route at once.
- **Spec reference:** §3.2, §3.3, AC-1, AC-2, AC-5; plan §6.5, §7.
- **Amended 2026-09-03, on landing. What this row covers and what it does not.** This is the
  project's **first L3 row**, and [ADR-0007](../../docs/decisions/0007-a-container-runtime-for-the-reference-instance.md)
  makes a differential run need a reference instance that is never automatic
  ([AGENTS.md §1.6](../../AGENTS.md)). **No reference instance is available in this run**, so what
  is met here is **L2 plus byte-level goldens** — the four measured refusals compared against one
  recorded body, the `LastPlaybackCheckIn` value compared as encoded bytes — and the differential
  half is a **recorded gap that closes the first time [010](../010-conformance-harness/spec.md)
  runs**, which is what spec §6's row already says. It is not L3 and this task does not claim it.
  Three further things landed here that the row's own wording did not name, each recorded in
  `plan.md` with the source that decided it: the four client components are checked rather than
  `DeviceId` alone (plan §6.5), the user object's single filler arrived with this route because it
  is the first to return one (plan §6.6), and the session model declares fifteen of the reference's
  twenty-eight members (plan §6.10). And `MaxActiveSessions` is **not** enforced here — spec §3.8's
  eviction needs a store method plan §5 does not declare, and AC-13's wire evidence at T20 will meet
  that rather than a passing test.

## T13 — One function builds the user object, and `GET /Users/Public`

- [x] **Changes:** `internal/httpapi` — the single filler of [plan §6.6](plan.md#66-building-the-user-object)
  that every route returning a user object calls, and the `/Users/Public` handler, which reads **no**
  credential even when one is present (plan §6.2).
- **Depends on:** T2, T4, T10
- **Verified by:** `/Users/Public` byte-compared with the same users read through an **authenticated**
  route — spec §3.4 measured that equality, so the test is an equality and not two independent shape
  checks; a hidden user excluded and the others whole with `Configuration` and `Policy` included; an
  installation where every user is hidden answering `200` with `[]`; `LastLoginDate` **absent** before
  a first login and present after, which is the `NULL`-versus-zero distinction a non-pointer field
  would silently answer `0001-01-01T00:00:00.0000000Z` for and which is the opposite of T12's
  `LastPlaybackCheckIn`; `PrimaryImageTag` and `PrimaryImageAspectRatio` absent, because v1 gives a
  user no avatar; and `ServerId` written **before** `Id`, asserted as key order in the encoded bytes
  — spec §3.5 records that this table had them the other way round until it was measured, and a
  member-by-member assertion cannot see it.
- **Spec reference:** §3.4, §3.5, AC-6; plan §6.6.
- **Amended 2026-09-03, on landing. The filler was already here, and four things the row did not
  name were decided.** [plan §6.6](plan.md#66-building-the-user-object)'s single filler landed with
  T12, because `POST /Users/AuthenticateByName` is the first route that returns a user object; this
  task extended nothing of it and added no second one, which is the section's whole point. What is
  new is the handler and every assertion the *Verified by* line names, none of which existed. The
  four decisions are recorded in [plan §6.2](plan.md#62-which-routes-require-a-token-and-which-merely-accept-one)'s
  amendment with the source that forced each: the answer's **order** is the store's and the
  reference's is by the unfolded username; the reference excludes **disabled** accounts too, where
  §3.4 names only the hidden — implemented as written, with the difference asserted as a test the
  way T15 asserts U-14; the reference answers **every** account before setup completes, which
  §6.8 makes unreachable here; and the reference narrows the list by device access and by
  `EnableRemoteAccess` for a caller outside the local network, which is the second place that flag
  is enforced and which qualifies §3.4's measured equality to a local caller. **The register at T23
  is owed a row for all four.** One further finding is in the handoff: the credential-independence
  assertion needs a **hidden** account in its fixture, because an installation whose only account is
  visible answers the same bytes however the handler treats a token.

## T14 — `GET /Users/Me` and `GET /Users/{userId}`, and the caller matrix

- [x] **Changes:** `internal/httpapi` — both handlers over T13's filler, with the `404` and the `400`
  of spec §3.7 written through T11's two new shapes.
- **Depends on:** T10, T11, T13
- **Verified by:** the whole matrix of spec §3.7 — every pair of seat and subject, including a
  restricted non-administrator naming an **administrator** — with the administrator's object as read
  by that stranger **byte-compared** with the administrator's own reading of it. The byte comparison
  is the assertion and a *"no refusal"* check is not: this route answered a `403` in this project's
  own specification from the day it was written until 2026-09-01, and the successor mistake is a
  handler that answers `200` with a redacted body, which every status assertion passes. Plus: an
  identifier that is well formed and belongs to nobody answering the 16-byte `404` with **the same
  body to an administrator and to a non-administrator**; an identifier that is not one answering the
  validation `400` keyed on `userId`'s own spelling; and no credential answering the empty `401` —
  three different shapes on one route, which is why they are asserted as bytes and not as statuses.
- **Spec reference:** §3.5, §3.7, AC-7, AC-12; plan §6.6, §7.
- **Amended 2026-09-03, on landing. The route has four answers where the *Verified by* line names
  three, and the fourth is the one worth reading.** A `userId` of **all zeros** is well formed and
  belongs to nobody, so the row above would make it the 16-byte `404`. It is not: the reference's
  account lookup refuses an empty identifier before it queries anything
  `[source: Jellyfin.Server.Implementations/Users/UserManager.cs:123-133 @ v10.11.11]`, the
  exception is mapped to `400` under `text/plain`
  `[source: Jellyfin.Api/Middleware/ExceptionMiddleware.cs:92-99,123-136 @ v10.11.11]`, and the same
  request **measured** on another route that resolves an identifier answered exactly the 25 bytes
  ([009 §3.8](../009-playlists/spec.md)'s identifier table, 2026-09-01). It is implemented rather
  than recorded, because the alternative was answering `404` to a request the reference refuses; it
  is compared against T12's golden, so five responses now stand on one file; and
  [spec §3.7](spec.md#37-get-usersme-and-get-usersuserid)'s table is owed the row, which is T22's
  document rather than this task's. Three further decisions are in
  [plan §7](plan.md#7-failure-handling)'s amendment with the source that forced each: the credential
  is read **before** the segment is bound, which is one observable request and a reading rather than
  a measurement of this route; what counts as an identifier here is narrower than the reference's
  binder and the dashed spelling's refusal is asserted as a divergence in T13's and T15's shape; and
  the `Access`-to-response mapping moved out of `SystemHandler.admits` into one function both use,
  which is T8's *a rule enforced twice is a rule no mutation of either half can reach* applied before
  it could happen a second time. **The register at T23 is owed a row for the order and one for the
  identifier grammar.** One finding is in the handoff and is the reason the central assertion has a
  fixture rather than two accounts: T13's lesson that *the same bytes to everybody* proves nothing
  over data with only one possible answer applies here too, so the administrator carries flags a
  redacting handler would plausibly withhold and a separate test asserts the three seats read as
  three **different** objects — without it, every equality in the file is satisfied by a handler that
  answers one account to everybody.

## T15 — `POST /Users/Configuration`

- [ ] **Changes:** `internal/httpapi` — the handler, replacing the **authenticated caller's**
  configuration and answering `204` with no body.
- **Depends on:** T10, T13
- **Verified by:** all sixteen properties posted and read back through `/Users/Me` unchanged; an
  unknown property accepted and **dropped**, with the declared ones surviving, which is spec §3.6 and
  is deliberately the opposite of what T16's capabilities do; `204` with no body and no content type;
  `401` without a token; and the case [U-14](../../docs/compatibility/reference-target.md) names — a
  request carrying a `userId` that names **somebody else** updates the caller's own configuration and
  leaves the named user's untouched, asserted on both accounts. That is the specification as written
  and it contradicts the reference's source, which updates the named user
  `[source: Jellyfin.Api/Controllers/UserController.cs:488-511 @ v10.11.11]`; writing it as a test
  rather than as a comment is what makes the day the probe lands a **failing test naming the
  behaviour that moved** instead of a rediscovery.
- **Spec reference:** §3.6, AC-8; plan §9; U-14.

## T16 — `POST /Sessions/Capabilities/Full` and `GET /Sessions`

- [ ] **Changes:** `internal/httpapi` — the capabilities handler, storing the posted document
  **whole**; the `/Sessions` handler over T9's `Visible`, with `controllableByUserId`'s `403` decided
  in the handler before the domain is asked (plan §6.10); and the first route-keyed entries
  `httpapi.QuerySpellings` has ever had — three names on one route, which is the arrival
  [001 plan §6.2](../001-server-identity-and-discovery/plan.md#62-query-key-canonicalisation-behaviours-115)
  shipped an empty map waiting for.
- **Depends on:** T9, T10, T11
- **Verified by:** `204` with no body, and **replacement rather than merge** — a second post carrying
  fewer properties leaves none of the first behind, which a merge implementation passes every
  round-trip test for; an unknown property **surviving** into `/Sessions`, asserted as
  [behaviours §5.9](../../docs/compatibility/behaviours.md#59-an-unknown-capabilities-property-survives-into-sessions-here-and-not-there)'s
  **recorded divergence**, so the test fails if it silently becomes parity — a declared difference
  that has gone away is a failure under [AGENTS.md §3](../../AGENTS.md), and this is the one place
  002 owns one; `PlayableMediaTypes` and `SupportedCommands` hoisted verbatim while
  `SupportsMediaControl` answers **`false` beside a declared `true`**, which is measured and is the
  case that separates the client's declaration from the server's judgement. Then spec §3.8's
  parameter matrix, over a fixture with two sessions on two devices: `deviceId` in another case,
  empty, and matching nothing; `activeWithinSeconds` at `0`, at `-5` and at a value that excludes a
  row; `controllableByUserId` naming the caller, naming somebody else **from a restricted seat** —
  whose `403` is byte-compared with T12's golden, since this is the same 25 bytes and register
  [U-18](../../docs/compatibility/reference-target.md) is the reason to assert them rather than
  assume them — and named by an administrator; and the combination of `deviceId` with
  `controllableByUserId`. **The `403` is the load-bearing half of this matrix**, and it is also
  AC-4's second half: it is the one request in 002 where a valid token is refused for who its holder
  is, and a route declaring neither parameter would answer both cases `200` with the caller's own
  sessions — a success where the reference refuses.
- **Spec reference:** §3.8, AC-4, AC-9, AC-13, AC-15; plan §6.10, §7.

## T17 — Register the seven rows, in one change

- [ ] **Changes:** `internal/httpapi` — `Handlers` gains its fields for this feature, `Routes`
  registers all **seven** of 002's `surface.yaml` rows, `everyHandler` fills the new fields (one
  line, and deliberately a failing check until it does — plan §8.4), and `conformance/`'s wire sweep
  gains the seven requests.
- **Why the seven arrive together:** 001 T20 derived *implemented* rather than listing it — a feature
  the server serves any row of must serve every row of it — so the first registration makes all seven
  required in both halves of the L0 check at once. Plan §8.4 leaves the choice here; this list takes
  the single change, because the alternative is a pull request that cannot be green.
- **Depends on:** T12, T13, T14, T15, T16
- **Verified by:** both halves of the L0 check pass with **eleven** rows served rather than four — the
  `chi.Walk` half in `internal/httpapi` and the one-real-request-per-row half in `conformance/` — and
  a route registered without a `surface.yaml` row still fails, and a row of an implemented feature
  that is not served still fails; the model sweep and the wire sweep pass over the new response types
  **and are each shown to fail** on a deliberately camelCase field and a deliberately three-digit
  date planted in a new model, because a sweep that has never failed over the types it has just been
  handed has proved nothing about them; and `Allow` on `/Sessions` and on `/Users/Configuration` is
  computed from the table rather than from the router, which is the value chi gets wrong in three
  ways (001 T11).
- **Spec reference:** §6; Principle VI; plan §8.4.

## T18 — `conformance/`: an installation the fixture provisions, and the credential criteria

- [ ] **Changes:** `conformance/` — one provisioning helper calling T7's subcommand before
  `startServer`, producing plan §8's fixture: an administrator with a password, a restricted
  non-administrator with a password, a hidden user, a disabled user, an account with no password and
  an account with a low lockout threshold; plus a second one-line fixture in which **every** user is
  hidden. Then AC-1, AC-2, AC-3 and AC-5 asserted against the running binary.
- **The fixture is built by the command an operator runs**, which is the point: it is not a back door
  into the store, and `conformance/` could not open one if it wanted to.
- **Depends on:** T7, T17
- **Verified by:** AC-1 — authenticate, `200`, a 32-hex `AccessToken` by shape, `User` and
  `SessionInfo` present, with a golden holding the body and its two derived members **stated** rather
  than normalised away (001 T16's rule: a byte-compared golden needs the response to stop deriving
  from the run, and the honest fix is to state the input). AC-2 — three requests, three statuses, one
  golden body compared by all three, and `Content-Type: text/plain` asserted **as a field value** so
  the absent charset parameter is part of the assertion. AC-3 — table-driven, five mechanisms across
  a required route and `/Users/Public`, plus the four precedence pairs in both directions and the
  grammar table over the wire; the image and delivery classes are 006's and 008's and are **named in
  the test as not this feature's** rather than silently skipped. AC-5 — authenticate twice from one
  `DeviceId`, the first token then answering `401` and `/Sessions` showing one row.
- **Spec reference:** AC-1, AC-2, AC-3, AC-5; §6; plan §8.

## T19 — `conformance/`: the user routes over the wire

- [ ] **Changes:** `conformance/` — AC-6, AC-7, AC-8 and AC-12 against the running binary.
- **Depends on:** T18
- **Verified by:** AC-6 — the hidden-user fixture and the all-hidden fixture, with `/Users/Public`
  **byte-compared against the authenticated reading of the same users**; AC-7 — the caller matrix at
  the wire, every seat against every subject, the identifier nobody has, the malformed one and the
  absent credential, with the byte comparison of one administrator's object read by two different
  seats; AC-8 — post all sixteen properties, read them back through `/Users/Me`, then post an unknown
  one and assert it is dropped and the declared ones survive; AC-12 — `/Users/Me` whole,
  configuration and policy included. Each of these exists at the wire *as well as* at the handler on
  purpose: 001's closing audit found two criteria proven about a mechanism rather than about a
  request, and the general lesson it wrote for this feature is that **a criterion written about a
  request is not met by a test about the mechanism that serves it, however good that test is**.
- **Spec reference:** AC-6, AC-7, AC-8, AC-12; §6; plan §8.

## T20 — `conformance/`: sessions, the lockout, and the parameter matrix over the wire

- [ ] **Changes:** `conformance/` — AC-9, AC-10, AC-13 and AC-15 against the running binary.
- **Depends on:** T18
- **Verified by:** AC-9 — post capabilities, read the session, the hoist and the `false` beside a
  declared `true`; AC-10 — a fixture account provisioned with `--login-attempts-before-lockout 2`,
  two failures and then the **correct** password answering `403`, and a second account where one
  success between failures resets the count; AC-13 — the `204` with no body, replacement rather than
  merge, `LastActivityDate` advancing across two requests, and the `MaxActiveSessions` row **as the
  specification writes it**, with plan §6.7's contradiction named in the test's own comment and
  [U-13](../../docs/compatibility/reference-target.md) cited, so that the request which settles it
  arrives at a test that already says what it expects; AC-15 — the six requests of plan §8's row plus
  the combination, with the `403` byte-compared against AC-2's golden. **The combination is its own
  case and is named for what it proves** — that `deviceId` still narrows a request that also carries
  `controllableByUserId` — and **not** for the order, which no request distinguishes (spec §3.8).
- **Spec reference:** AC-9, AC-10, AC-13, AC-15; §6; plan §6.7, §8.

## T21 — AC-14, and the two notes 001 parked on it

- [ ] **Changes:** `conformance/` — provision an administrator, authenticate over HTTP, send the
  returned token to `GET /System/Info` and assert `200` with the superset body; and 001's `401` on
  the same route moved here from `internal/httpapi`, which is possible for the first time because T7
  gives this package an installation whose setup is **complete**.
- **This is the debt 001's closing audit recorded in 002's specification, and this task is where it
  is discharged.** [001 AC-5](../001-server-identity-and-discovery/spec.md#5-acceptance-criteria) is
  three claims; 001 proved two, and the third needs a credential only this feature can issue. It
  became [002 AC-14](spec.md#5-acceptance-criteria) rather than a note, for the reason 001 gave: *a
  criterion carried in a sentence is one nobody closes.*
- **Depends on:** T7, T17, T18
- **Verified by:** two requests against **one** server — the token answering `200` with the superset
  body, and the **identical** request without a token answering `401`. The second request is what
  makes the first a proof: on an installation whose setup is outstanding, `/System/Info` admits a
  request carrying nothing at all
  `[source: Jellyfin.Api/Auth/FirstTimeSetupPolicy/FirstTimeSetupHandler.cs:29-31 @ v10.11.11]`, so a
  test written against such a server would be green, named for AC-14 and proving nothing. **Show it
  can fail**: run the same pair against an *unprovisioned* installation, where the token-less request
  answers `200` and the companion assertion must go red. That run is the evidence, and it belongs in
  the pull request body. The `internal/httpapi` `401` test **stays where it is** — it is the only one
  that can see `Content-Length: 0` and the two absent headers over a real connection beside a stubbed
  store, and losing it would trade a stronger check for a tidier directory (plan §8.2).
  `TestPingAnswersTheProductNameAndNotThisServersFriendlyName` **keeps its caveat**, and the
  correction 001's note needs is recorded rather than the caveat dropped: the reference renames a
  server at an operation this surface does not include
  `[source: Jellyfin.Api/Controllers/StartupController.cs:74-78 @ v10.11.11]`, so *"when the rename
  endpoint lands"* is a condition no v1 feature can satisfy; what can satisfy it is the friendly name
  becoming operator configuration, that is 001's datum, and this feature deliberately does not add a
  configuration surface for it on the way past.
- **Spec reference:** AC-14 and §5's two carried notes; plan §8.2; [001 T18, T21](../001-server-identity-and-discovery/tasks.md).

## T22 — The cross-document debts: which of them are this feature's

- [ ] **Changes:** the paired-file and cross-document debts the plan and the spec amendment reported
  and did not fix, each **decided** rather than listed.
  - **Taken here.** [request-cases.yaml](../../docs/compatibility/request-cases.yaml)'s
    `video-client-device-id` row says its parameter is *"a parameter neither server's route
    declares"*. That is now false on **both** halves: the reference declares it
    `[source: Jellyfin.Api/Controllers/SessionController.cs:52-59 @ v10.11.11]`, and Atrium declares
    it at spec §3.8 as of 2026-09-03. Strike it in place with the date and the reason
    ([AGENTS.md §4](../../AGENTS.md)), and state the consequence for
    [010 §3.6](../010-conformance-harness/spec.md) in the same edit: the row stops being an
    *ignored-parameter reading* on this server, so a run expecting to count `deviceId` there will
    count nothing. **The pairing is not disturbed** — the file's prose twin is 010 §3.2 and §3.9,
    neither of which says anything per-case, and no row is added or removed, so the row-for-row
    comparison is unchanged. This is 002's to close because 002's own declaration is what falsified
    it.
  - **Not taken, with the reason, and handed on.** [allowlist.yaml](../../docs/compatibility/allowlist.yaml)'s
    three gaps — no rows at all on `/Users/Public`, `/Users/Me` and `/Users/{userId}`, no
    `LastLoginDate` row anywhere in the file, and no `/-/Id` row on `GET /Sessions` — are named in
    [plan §8.3](plan.md#83-l3-is-deferred-and-this-is-the-feature-where-that-costs-the-most) and stay
    named. That file is one third of a three-way pairing with `conformance.md` L3 and 010 §3.3, and
    001 already refused to write one third of a triple. What 001 *did* do in its place is the model
    to follow: it wrote [behaviours §4.5](../../docs/compatibility/behaviours.md), the thing an
    allowlist entry would have to **cite**, because an entry citing nothing fails the file's own
    load. Check that the same hole does not exist here — the shapes in question are
    `derived-identifier` and `wall-clock`, both already declared classes — and if a gap needs a
    behaviours section, **that** is 002's to write and the allowlist rows are still 010's.
  - **Not taken, with the reason, and handed on.** A request case for `controllableByUserId` — the
    one request in this feature where a valid token is refused for who its holder is — does not
    exist in `request-cases.yaml`, so a differential run never sends 002's only permission refusal.
    What a run sends is 010's decision; the measurement that justifies the case is this feature's and
    is handed on below.
  - Write **[What this feature owes the next ones](#what-this-feature-owes-the-next-ones)** with
    these and anything T12–T21 found, in the shape 008, 009 and 011 use.
- **Depends on:** T16, T21
- **Verified by:** every internal link in the edited documents resolves to a file and an anchor that
  exists; the corrected row is struck in place rather than replaced, so the file still records what
  it believed; `request-cases.yaml` still loads and still carries the same number of rows as before
  the change, which is the mechanical half of *"the pairing is not disturbed"*; and each debt not
  taken names its owner and the measurement that owner will need.
- **Spec reference:** §3.8; plan §8.3, §11; [docs/README.md §Paired files](../../docs/README.md#paired-files-edit-both-halves-or-neither).

## T23 — The closing audit

- [ ] **Changes:** whatever this task finds. It is not a formality:
  [AGENTS.md §5](../../AGENTS.md) records that every implemented feature in the exporting project
  found, in its own final task, **an acceptance criterion with no test or a test proving less than
  its name** — and 001, the only feature this repository has implemented, found two.
- **Depends on:** all of the above
- **Verified by:** five passes, each recorded with what it found **or that it found nothing** —
  - **(a) Every one of the fifteen acceptance criteria mapped to a named test that fails when the
    *behaviour* is broken, verified by mutating the production code rather than by reading.** A
    mutation that merely deletes a function is not on the list, because a test that fails only when
    code is missing is a test of the build. Two shapes are hunted for by name, because 001 found
    exactly these: a criterion about a **request** proven about the **mechanism** that serves it
    (001's F-1, AC-9 proven over a model declared in a `_test.go` file), and a criterion about *"the
    same bytes"* proven about an **echo** rather than a response (001's F-2). In this feature the
    likeliest instances are AC-3, whose five mechanisms are a pure function with a beautiful table
    test, and AC-11, which is a property of logging and not of any route.
  - **(b) Every paragraph of spec §3 either tested or listed as untested with a reason.** *Tested*
    means at least one named test fails when the paragraph's behaviour is broken.
  - **(c) The L3 row stated rather than ticked.** `POST /Users/AuthenticateByName` is this project's
    first `level: L3` row. No reference instance is available in this run, and
    [AGENTS.md §1.6](../../AGENTS.md) forbids CI from contacting or starting one, so what is met is
    **L2 and byte-level goldens** and the differential half is a recorded gap that closes the first
    time [010](../010-conformance-harness/spec.md) runs — which is spec §6's own row and is how 001
    discharged its two L3 rows. Name what already covers it: allowlist.yaml carries **ten** rows on
    this endpoint and **two** on `GET /Sessions`. **No task and no definition-of-done line may claim
    L3 is reached.**
  - **(d) The register.** Anything this feature asserts and has never measured goes to
    [reference-target.md](../../docs/compatibility/reference-target.md) beside U-13 to U-18 rather
    than into a plan paragraph — that register exists because four of 001's tasks wrote *"this
    belongs in the register"* and nobody owned the document.
  - **(e) What implementation taught, written back.** Into `spec.md` in **this same change**
    (Principle III), and any newly *measured* reference behaviour into `behaviours.md` with
    provenance. A source reading is not a measurement: where the reference's source contradicts this
    specification, [AGENTS.md §1.3](../../AGENTS.md) makes the running server the tie-breaker, and
    with none here the specification is implemented as written and the contradiction is recorded —
    which 001 did twice and which this feature already owes at U-13 and U-14.
- **Spec reference:** all of §5; §6; AGENTS.md §5.

---

## What this feature owes the next ones

*Written at T22. Until then this section is empty on purpose — a handover written before the work is
a guess.*

---

## Definition of done

The feature is done when **all** of these hold:

- [ ] Every acceptance criterion in `spec.md` §5 has a passing test, mapped in T23's pass (a) and
  each mapping verified by **breaking the behaviour** rather than by reading.
- [ ] Every endpoint reaches the conformance level declared in `spec.md` §6 — **except the
  differential half of `POST /Users/AuthenticateByName`'s L3, which is deferred on the
  specification's own terms and closes the first time 010 runs.** It is named here rather than
  ticked, the way 001 named its own two.
- [ ] `docs/compatibility/surface.yaml` lists every route added, and no route exists outside it —
  proven by both halves of the L0 check, over eleven rows rather than four (T17).
- [ ] Anything learned during implementation is back in `spec.md`, in this same change.
- [ ] Any new measured Jellyfin behaviour is in `docs/compatibility/behaviours.md` with provenance,
  and anything this feature asserts and has **not** measured is in `reference-target.md`'s register
  rather than in a plan paragraph.
- [ ] The debt 001 recorded here is discharged and says so: AC-14 at T21, together with the two
  assertions 001 parked at a lower level for the same reason.
- [ ] ADR-0006's timing equalisation — *"specified here and asserted nowhere"*, and the one check
  that record's argument stands on — is asserted at T6, in both halves, each shown able to fail.
- [ ] `spec.md`, `plan.md` and `tasks.md` are all marked `Implemented`.
