---
feature: 001-server-identity-and-discovery
title: Server identity and discovery
status: Implemented
created: 2026-08-26
updated: 2026-09-03
accepted: 2026-08-26
amended: 2026-09-03 by T18 - section 3.2, on PackageName being absent rather than empty, and section 5, on AC-5's third half being carried into 002; by T18 - section 6, on why /System/Info has no golden. 2026-09-03 at the closing audit (T21) - sections 3.2, 3.4, 3.5 and 6: two places where this document is contradicted by the reference's own source and is deliberately NOT amended on that evidence, one refusal 001 can reach no state to produce, and what AC-9 and AC-11 were and were not proven against
depends_on: []
---

# 001 — Server identity and discovery

> **This document describes WHAT and WHY only.** No technology names, no storage decisions.

## 1. Purpose

Every client's first request is "is there a server here, and what is it?". This feature answers it.

It is first not because it is large — it is the smallest feature in v1 — but because it is the
first thing that can be wrong, and because it forces the wire-format decisions (property casing,
version string, identifier shape, the meaning of `ProductName`) that everything else will encode.
A client that mis-identifies the server here takes an unknown-server path and never reaches the
rest of the API.

**Client behaviour unlocked:** entering a server address in a client and getting "connected to
<name>" instead of "cannot find a server".

## 2. Scope

**In scope**

- `GET /System/Info/Public` — unauthenticated identity.
- `GET /System/Info` — authenticated, fuller identity.
- `GET /System/Ping` and `POST /System/Ping` — liveness.
- The rules for choosing the advertised address.
- The project-wide wire-format rules that these responses are the first to exercise.

**Out of scope**

- `GET /Users/Public` — the login-screen user list, which belongs with authentication (002).
- `GET /Localization/Cultures` — belongs with item metadata (004).
- `/System/Info/Storage`, `/System/Restart`, `/System/Shutdown`, `/System/Logs`,
  `/System/Endpoint` — administrative surface, not in v1.
- UDP autodiscovery on the local network — not in v1; clients take an address.

## 3. Behaviour

### 3.0 Rules that apply to every response in this specification

These are stated once here and inherited by every later specification. They come from
[behaviours.md §1](../../docs/compatibility/behaviours.md#1-wire-format).

1. **Property names are PascalCase.** Non-negotiable, project-wide.
2. **Content type is `application/json`.** The server additionally accepts requests specifying
   `application/json; profile="PascalCase"` or `profile="CamelCase"`. Those are three names for
   **two** behaviours, not three names for one: the plain type and the PascalCase profile answer
   identically, the CamelCase profile answers with camelCase property names **at every depth**, and
   the response's content type echoes the profile that matched, before the charset.
   `[probe: tools/probe_content_type_profiles.py, Jellyfin 10.11.11, 2026-08-26]`

   Three details decide whether a reimplementation is right, and none are in the reference's
   published contract:

   - the profile is matched leniently on the media type's parameter — unquoted and in any casing —
     but **a charset parameter beside it stops the match**, and an unknown profile falls back;
   - ranking is ordinary content negotiation: equal quality keeps the client's order;
   - **dictionary keys are never converted.** Only property names are, which is why the conversion
     happens where a field is still a field.

   The conversion itself is the reference's own naming policy rather than "lower the first letter":
   a leading run of capitals lowers all but the last of them, so `UICulture` becomes `uiCulture`.
3. **Absent optional values are omitted or null exactly as the reference server does**, verified per
   field rather than by rule.
4. **Identifiers are 32 lowercase hexadecimal characters**, no dashes.

### 3.1 `GET /System/Info/Public` — `GetPublicSystemInfo`

**Consumers:** music-client, video-client. The single most important response in the API: it is what
a multi-server client uses to decide which dialect it is speaking.

**Request:** no authentication, no parameters. Must answer before any user exists and before any
library is configured.

**Response — 200**

```json
{
  "LocalAddress": "http://192.168.1.20:8096",
  "ServerName": "atrium",
  "Version": "10.11.11",
  "ProductName": "Jellyfin Server",
  "OperatingSystem": "",
  "Id": "3f9c1a7e5b2d4e8091a6c3f70d5e2b14",
  "StartupWizardCompleted": true
}
```

| Field | Type | Value |
|---|---|---|
| `LocalAddress` | string | The address this server advertises for the requester's network. §3.4 |
| `ServerName` | string | Operator-chosen friendly name. Default `atrium` |
| `Version` | string | `10.11.11` — the pinned reference version, not Atrium's own |
| `ProductName` | string | **Exactly `Jellyfin Server`** |
| `OperatingSystem` | string | **Always the empty string** |
| `Id` | string | Stable 32-hex server identity, generated once and persisted |
| `StartupWizardCompleted` | boolean | Whether initial setup is finished |

**Compatibility notes**

- `ProductName` is the documented discriminator multi-server clients use to tell Emby from
  Jellyfin. It must be the literal `Jellyfin Server`. The reasoning, and why this does not conflict
  with being honest about what Atrium is, is in
  [reference-target.md §4](../../docs/compatibility/reference-target.md#4-server-identity-what-atrium-tells-clients-it-is).
  `[probe: tools/probe_public_info.py, Jellyfin 10.11.11, 2026-08-28]`
- `OperatingSystem` is empty because the reference implementation marks the field obsolete and
  never assigns it, leaving its default empty-string value.
  `[source: MediaBrowser.Model/System/PublicSystemInfo.cs:37-38 @ v10.11.11]`
- The reference builds this response from exactly six assignments and never sets
  `OperatingSystem`. Atrium sends the same seven fields and no others.
  `[source: Emby.Server.Implementations/SystemManager.cs:112-125 @ v10.11.11]`
- `Id` must survive restarts and database rebuilds. A server whose identity changes makes every
  client treat it as a new server and re-authenticate.

**Errors**

| Condition | Status |
|---|---|
| Server still starting | `503`; see §3.5 |

### 3.2 `GET /System/Info` — `GetSystemInfo`

**Consumers:** music-client.

**Request:** authenticated. The reference also permits this during first-time setup, before any
user exists.
`[source: Jellyfin.Api/Controllers/SystemController.cs:67-71 @ v10.11.11]`

**Response — 200:** a superset of §3.1. In addition to the seven public fields, the reference
returns paths, capability flags and update state.
`[spec: SystemInfo]`

Atrium returns the superset with these values:

| Field group | Atrium's answer |
|---|---|
| `HasPendingRestart`, `IsShuttingDown`, `CanSelfRestart`, `CanLaunchWebBrowser`, `HasUpdateAvailable` | `false` — Atrium has no self-update or restart capability |
| `SupportsLibraryMonitor` | `false` in v1 — filesystem watching is not implemented |
| `WebSocketPortNumber` | The server's own port; v1 serves no WebSocket, but the field is a number clients read unconditionally |
| `ProgramDataPath`, `CachePath`, `LogPath`, `InternalMetadataPath`, `ItemsByNamePath`, `TranscodingTempPath`, `WebPath` | Real paths from the running configuration |
| `CompletedInstallations`, `CastReceiverApplications` | Empty arrays |
| `PackageName`, `OperatingSystemDisplayName`, `SystemArchitecture`, `EncoderLocation` | Real values where meaningful, empty string otherwise |

> ⚠️ **Open question OQ-1.** Whether any client branches on `SupportsLibraryMonitor` or
> `WebSocketPortNumber`. Neither analysed client reads them. Answering it needs the differential
> harness (010) or a survey of other clients. It does not block this feature: the fields are sent
> with honest values either way.

**Amended 2026-09-03.** The table's last row asks for *"real values where meaningful, empty string
otherwise"*, and for one of its four fields that is the wrong answer: **`PackageName` is not sent
at all.** §3.0.3 says absent optional values are omitted *"exactly as the reference server does,
verified per field rather than by rule"*, and for this field the per-field verification exists —
the reference declares `PackageName` on this response and does not send it
`[probe: tools/probe_public_info.py, Jellyfin 10.11.11, 2026-08-28]`, recorded in
[behaviours §1.7](../../docs/compatibility/behaviours.md#17-a-null-property-is-absent-everywhere-by-one-setting)
as the worked example of the omit-when-null rule. An empty string here would be a property no
reference server carries, which is a delta by Principle I on the count. The row stands for the
other three, which the reference does send and which are all empty.

**Errors**

| Condition | Status |
|---|---|
| No or invalid token, and setup already complete | `401` |
| Valid token without permission | `403` |

**Amended 2026-09-03 at the closing audit. The `403` row is unreachable in this feature and is
untested here on purpose.** 001 issues no credential, so there is no request that can be *valid and
insufficient*: the two states it can reach are setup outstanding, which is admitted whatever the
request carried, and setup complete with no credential, which is the `401`. The row is not softened
or deleted — it is what the reference does — and the implementation refuses to guess at it: its
admission port declares **two** values, granted and unauthenticated, and a third that arrived
without the refusal being taught would be answered `500` loudly rather than admitted or refused
silently. [behaviours §1.11](../../docs/compatibility/behaviours.md) also gives a `403` a **body**
this project has not measured on this route, which is the second reason not to invent one.
[002](../002-authentication-users-and-sessions/spec.md) is where a token exists and where this row
is met.

### 3.3 `GET /System/Ping`, `POST /System/Ping` — `GetPingSystem`, `PostPingSystem`

**Consumers:** none of the two analysed clients; included by design (health checks, reverse proxies,
and clients that probe before a full request).

**Request:** no authentication, no parameters, both methods.

**Response — 200:** a bare JSON string — **the product name, not the friendly server name**:

```json
"Jellyfin Server"
```

**Compatibility note.** The reference's documentation comment says "the server name", but the code
returns the application's product name.
`[source: Jellyfin.Api/Controllers/SystemController.cs:102-106 @ v10.11.11, returning
_appHost.Name; ApplicationHost.cs:260 defines Name => ApplicationProductName]`
Following the comment instead of the code would return the operator's chosen name and produce a
delta. **The code is the specification.**

### 3.4 Choosing `LocalAddress`

The reference resolves this in three tiers, and the tiers are observable, so Atrium reproduces
them. `[source: Emby.Server.Implementations/ApplicationHost.cs:871-949 @ v10.11.11]`

1. **A configured published URL wins**, returned verbatim with any trailing `/` removed. This is
   what an operator behind a reverse proxy sets, and it must not be second-guessed.
2. **Otherwise, if configured to derive the address from the request**, build it from the request's
   own host and scheme, omitting the port when it is the default for that scheme (80/http,
   443/https).
3. **Otherwise, match the requester's address against the server's bound addresses** and return the
   one on the same network, with the port it is bound to. A request arriving over a VPN gets the
   VPN-side address.

**Deliberate divergence.** In tier 3 the reference overrides the scheme to HTTPS, with the HTTPS
port, whenever a certificate is configured — regardless of the scheme the request arrived on. This
has a measured cost: clients that hand this address to a device with no TLS stack (a DLNA renderer)
break. `[probe: tools/probe_local_address.py, Jellyfin 10.11.11, 2026-09-02]`

**Atrium reports the scheme and port the server is actually reachable on for that network.** Logged
as a deliberate divergence in
[behaviours.md §4.2](../../docs/compatibility/behaviours.md#42-localaddress-does-not-get-an-https-override).

**Amended 2026-09-03 at the closing audit. The order of tiers 1 and 2 is contradicted by the
reference's own source, and this document is deliberately not changed on that evidence.** The
overload that serves this response tests *derive from the request* **before** *published URL*
`[source: Emby.Server.Implementations/SystemManager.cs:77,120;
ApplicationHost.cs:885-901 @ v10.11.11]`, which is the opposite of the numbering above and of
[behaviours §2.3](../../docs/compatibility/behaviours.md)'s corrected table. The two readings differ
on **exactly one installation**: one with both configured. AGENTS.md §1.3 makes a running server
the tie-breaker over source, and no running reference is reachable from here — so the
implementation follows this document as written, and the question is registered as **U-2** in
[reference-target.md](../../docs/compatibility/reference-target.md#claims-this-project-asserts-and-has-never-measured--added-2026-09-03).
One request settles it; failing that it is one undeclared difference in 010's run, and whichever
way it falls, what changes is this paragraph rather than the implementation's shape.

### 3.5 What the server says while it is starting

Not an error path of one endpoint but a property of the whole server: **every operation the
reference declares declares a `503`**, so nothing is exempt — not even a liveness probe. Re-measured
against the pinned document on 2026-09-01: 389 of 389, every one carrying exactly `Retry-After` and
`Message` and a `text/html` body. The count is a property of *that server* rather than of Jellyfin —
an installed plugin adds operations, which is how the 395 recorded on 2026-08-26 was reached — so
the claim is the coverage, not the number.
`[spec: every operation's 503 response in the pinned 10.11.11 document]`

**Response — 503**

| Part | Value |
|---|---|
| `Retry-After` | Full seconds, as an integer. Not an HTTP-date |
| `Message` | A short plain-text reason |
| Body | `text/html` — **not** JSON |

`Retry-After` is what separates "starting" from "broken" for a client. Without it a `503` is
indistinguishable from a server that is down, and a client that cannot tell will either give up or
hammer.

The same response serves a deliberate withdrawal from service — a long rebuild, say — with a
different message and a longer hint, without stopping the process.

> The `503` responses are also where the reference's OpenAPI document declares `allowEmptyValue`
> on header objects, which is invalid there and makes strict parsers reject the whole document. See
> [reference-target.md §2](../../docs/compatibility/reference-target.md#2-sources-of-truth-in-precedence-order).

**Amended 2026-09-03 at the closing audit. *"Nothing is exempt"* is cited from the pinned document
and is contradicted by three readings of the reference's source, and this section is deliberately
implemented as written rather than corrected.** The three, all at `v10.11.11`:

1. the starting `503` does not come from the main pipeline at all. It comes from a **separate setup
   web server** that runs before the application object exists, whose whole pipeline is `/health`,
   forwarded headers, `/startup/logger`, `/System/Info/Public` and a catch-all `503`
   `[source: Jellyfin.Server/ServerSetupApp/SetupServer.cs:177-259 @ v10.11.11]`. That server
   registers **no** response-time middleware;
2. on that server, `/System/Info/Public` is **not** a `503` once the host object exists — it
   answers a real public body with `StartupWizardCompleted = false`
   `[source: .../SetupServer.cs:204-237 @ v10.11.11]`;
3. the **main** pipeline's own gate exempts `/system/ping`, case-insensitively, and sends
   **neither** `Retry-After` nor `Message` — only the status, `text/html` and a localised string
   `[source: Jellyfin.Api/Middleware/ServerStartupMessageMiddleware.cs:38-48 @ v10.11.11]`.

None of that is a measurement of a running server, which is what would settle it (AGENTS.md §1.3),
and there is no running reference here. So the section stands, the implementation exempts nothing
and sends both headers, and the contradiction is registered as **U-1** in
[reference-target.md](../../docs/compatibility/reference-target.md#claims-this-project-asserts-and-has-never-measured--added-2026-09-03)
together with the two smaller unknowns it carries: whether `Retry-After` is zero-padded (the setup
server writes `005`; the pinned document declares an integer; this server sends `5`) and what the
`text/html` body's bytes are — there being no single reference body to copy, a bare string in one
place and a rendered startup-log page in the other, so the media type is all this document asserts.
**Editing §3.5 on source evidence would have been the wrong move**, and is recorded here as one not
taken.

### 3.6 How a request is matched to a route

Every path in this repository is written in one canonical spelling, and that is not the only
spelling that reaches the route. Like §3.5, this is a property of the whole server rather than of
one endpoint, and it is stated here because 001 is the feature that first has routes to match.
`[probe: tools/probe_routing.py, Jellyfin 10.11.11, 2026-08-26]`

| A client sends | Answer |
|---|---|
| The canonical spelling | The route |
| Any other casing — `/system/info/public`, `/SYSTEM/INFO/PUBLIC` | The same route, the same bytes |
| One trailing slash | The same route, the same bytes. **Not** a redirect |
| Two trailing slashes | `404` |
| A path matching no route | `404`, empty body, no content type |
| A method the path does not have | `405`, empty body, no content type, and an `Allow` header |

**`Allow` lists every method the path has**, which is not the same as every method the matched
route has: a path served by two routes — `GET` and `POST` on the same path — advertises both.

**Nothing is automatic.** `HEAD` on a path that has only `GET` is a `405`, and so is `OPTIONS`.
Answering either would be a difference on every route at once, which is the kind of default worth
turning off deliberately rather than inheriting.

Path **parameters** are values, not spellings: only the segments a route declares literally are
matched case-insensitively, and whatever occupies a parameter reaches the handler exactly as the
client wrote it.

## 4. Data the feature owns

Observable, and surviving restart:

| State | Observable as | Lifetime |
|---|---|---|
| Server identity | `Id` in §3.1 and §3.2 | Generated once on first start; never changes |
| Friendly name | `ServerName` | Operator-configurable |
| Setup completion | `StartupWizardCompleted` | Set once initial configuration is done |

Everything else in these responses is derived at request time.

## 5. Acceptance criteria

1. `GET /System/Info/Public` answers `200` with exactly the seven fields of §3.1, in PascalCase,
   with no user configured and no library present.
2. `ProductName` is exactly `Jellyfin Server` and `OperatingSystem` is exactly `""`.
3. `Version` matches the pinned reference version.
4. `Id` is 32 lowercase hex characters and is **identical across a restart and across a rebuild of
   the store from empty**.
5. `GET /System/Info` answers `401` without a token and `200` with a valid one, and its body is a
   superset of `/System/Info/Public` agreeing on every shared field.
6. `GET /System/Ping` and `POST /System/Ping` both answer `200` with the JSON string
   `"Jellyfin Server"`.
7. A configured published URL is returned verbatim in `LocalAddress`, trailing slash removed.
8. With no published URL, two requests from two different networks receive two different
   `LocalAddress` values, each on the requester's network.
9. Requests sent with `Accept: application/json` and with `application/json; profile="PascalCase"`
   receive byte-identical bodies. A request sent with `profile="CamelCase"` receives the same
   values under camelCase property names, and every response says in its own content type which of
   the two it used.
10. No response in this feature contains a property name that is not PascalCase.
11. A path spelled in any casing, with or without one trailing slash, reaches the same route and
    returns the same bytes; two trailing slashes do not. A path matching no route and a method a
    path does not have are both refused with an empty body, and the second carries an `Allow`
    header naming **every** method that path has.
12. While the server is starting, **every** route answers `503` with `Retry-After` in full
    integer seconds, a `Message` header saying why, and a `text/html` body — never JSON (§3.5).
    *(Added at the 2026-08-28 audit — M27: the whole of §3.5 was implemented and tested with no
    criterion covering it.)*
13. With no published URL and the server configured to derive the address from the request, the
    request's own host and scheme come back in `LocalAddress`, with the port omitted when it is
    the scheme's default (§3.4 tier 2). *(Added at the same audit — M28.)*

**Amended 2026-09-03. AC-5 is met in two of its three halves, and the third is carried.** The
superset assertion and the `401` are proven; *"and `200` with a valid one"* is not, because a valid
credential is something [002](../002-authentication-users-and-sessions/spec.md) issues and nothing
here can. It is recorded as a **criterion carried into 002** rather than quietly widened into
something a stub could satisfy — a test that admitted a fictional token would prove the handler
asks its port, which is a true and much smaller claim, and it is asserted under that name instead.

**Amended 2026-09-03 at the closing audit: the carried half now has somewhere to be met.** It is
written into 002 as **[AC-14](../002-authentication-users-and-sessions/spec.md#5-acceptance-criteria)**,
with the two conditions that discharge it — 001's authentication port filled, and the request
issued to the running binary carrying a token 002's own route returned. A criterion carried in a
sentence is a criterion nobody closes; a criterion carried into another specification's numbered
list is one somebody has to.

**Amended 2026-09-03 at the closing audit: AC-9 and AC-11 were proven at a level below the one they
are written at, and now are not.** Both read as claims about *requests to this feature's
endpoints*, and both were proven over stand-ins — AC-9 over a model declared in a test file, AC-11
over a router whose handler echoes the route it was reached through. Each was verified by mutation
rather than by reading: a handler writing a constant content profile, and a pipeline stage that
recognises the doubled slash and then folds nothing, each left every test at the HTTP boundary
green. Both now have tests against the running binary, and §6 records what they are.

## 6. Conformance

| Endpoint | Level | How it is proven |
|---|---|---|
| `GET /System/Info/Public` | **L3** | Golden response, plus differential against a real reference server. It is the first request every client makes; a difference here costs everything downstream |
| `GET /System/Info` | **L2** | ~~Golden response and~~ the superset assertion of AC-5, the property names in order, and each value asserted on its own — see the amendment below |
| `GET /System/Ping` (both methods) | **L2** | Exact-body test |
| `LocalAddress` selection | **L2** | Table-driven test over the three tiers with synthesised requester addresses |
| Path matching and refusals (§3.6) | **L0** | Every route registered against `surface.yaml` and none outside it; the accepted spellings; the empty `404` and `405` |
| The three content types (§3.0.2) | **L1** | Byte-identical plain and `PascalCase` bodies, camelCase names carrying the same raw values, and the echo — against both `/System/Info` routes as well as over the serialiser |

The two cross-cutting sweeps described in
[conformance.md](../../docs/compatibility/conformance.md#l1--shape) — PascalCase over every
response model, and units over every `*Ticks` and `*Date` field — are **delivered by this feature**,
because it is the first one with a response model to sweep.

**Amended 2026-09-03. `/System/Info` has no golden, and the reason is in the response.** Seven of
its fields are the installation's own paths and one is the port the operating system chose, so a
recorded body would either match only on the machine that recorded it or have to be softened until
it stopped comparing bytes — and a softened golden is worth less than none, because it looks like a
byte comparison. The two things a golden buys are bought separately and both are byte-level: the
**property names in the order they arrive**, which is the field count, the key order and the
absence of `PackageName` in one assertion, and the **per-field raw JSON values**, which tell an
empty array from a null one and a boolean from the string spelling of one. The superset assertion
of AC-5 compares the two bodies member by member as raw JSON for the same reason.

**Amended 2026-09-03 at the closing audit. Three of this feature's behaviours are asserted below
the HTTP boundary, and each is there for a reason that is a property of 001 rather than a shortcut.**
Naming them is the point: a reader who assumes the whole of §3 is proven against the running binary
would be wrong in exactly these three places, and each closes in a later feature.

- **§3.4's three tiers, and AC-7, AC-8 and AC-13 with them.** 001 gives an operator no way to
  configure a published URL, a derive-from-request flag or a bound-address list, so every
  installation this binary can be started as reaches tier 3's fallback and answers from the
  request. The choice is a pure function of the request's facts and the configuration, tested over
  all three tiers and over the deliberate divergence; what is missing is a **caller**, and it
  arrives with the first feature that adds configuration.
- **§3.5 and AC-12.** The binary opens its readiness gate before it serves, and 001 routes nothing
  that withdraws it, so no request a client can make ever sees a `503`. Every route and one
  unrouted path are asserted against the assembled pipeline over a real connection instead, held
  pre-ready — which is the same instrument minus the process boundary.
- **§3.2's `401`.** It needs an installation whose setup is *complete*, and 001 serves no route
  that completes one. Asserted at the handler over a real connection, because three of the four
  things this refusal is measured by are invisible to an in-memory recorder. **It moves to the
  HTTP-boundary suite the day 002 can finish setup over HTTP.**

**L3 for `/System/Info/Public` is not met, and is deferred rather than skipped.** Its second half —
the same request issued to Atrium and to a real reference server, compared field by field — needs
the differential harness that [feature 010](../010-conformance-harness/spec.md) delivers, and that
harness needs a reachable Jellyfin, which no gate in this repository is allowed to depend on. What
is met now is **L2**: a byte-compared golden response, every field's value asserted, and the
measurements behind them recorded with provenance. The gap closes the first time 010 runs, and the
`LocalAddress` row of its allowlist is already written.

## 7. Open questions

| # | Question | Blocks | Resolved by |
|---|---|---|---|
| OQ-1 | Does any real client branch on `SupportsLibraryMonitor` or `WebSocketPortNumber`? | Nothing. Honest values are sent either way | Differential harness (010), or surveying additional clients |
| OQ-3 | Is `StartupWizardCompleted` meaningful for Atrium, which has no wizard? | Nothing; `true` after first configuration | A decision in 002, where user creation happens |
| OQ-4 | Whether a running reference actually emits both headers, or only declares them | Nothing; both are sent | `tools/probe_startup.py`, which has to catch a server mid-start |

### Resolved

| # | Question | Answer | Resolved by |
|---|---|---|---|
| OQ-2 | Does the reference emit `503` with `Retry-After` while starting? | **Yes, and it is server-wide** — all 395 operations declare it, with `Retry-After` **and** a `Message` header, and a `text/html` body. §3.5 records it; §3.1's one-line error row was incomplete | The pinned document, 2026-08-26 |

## 8. References

- [docs/compatibility/reference-target.md](../../docs/compatibility/reference-target.md) — the pin, and server identity
- [docs/compatibility/behaviours.md](../../docs/compatibility/behaviours.md) — §1 wire format, §2.3 and §4.2 `LocalAddress`
- [docs/compatibility/api-surface-v1.md §2](../../docs/compatibility/api-surface-v1.md#2-identity-and-discovery)
- Jellyfin v10.11.11: `Jellyfin.Api/Controllers/SystemController.cs`,
  `Emby.Server.Implementations/SystemManager.cs`,
  `Emby.Server.Implementations/ApplicationHost.cs`,
  `MediaBrowser.Model/System/PublicSystemInfo.cs`
