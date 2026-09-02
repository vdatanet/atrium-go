---
feature: 001-server-identity-and-discovery
title: Server identity and discovery — implementation plan
status: Accepted
created: 2026-09-02
updated: 2026-09-03
spec_status_required: Accepted
---

# 001 — Implementation plan

> **This document describes HOW.** It may not restate WHAT: the spec is the authority on behaviour,
> and a plan that repeats it will disagree with it eventually.

**On the gates.** This plan moved to `Accepted` when its review returned and the task list was
asked for; the task list's own `plan_status_required` is what that satisfies.

The template asks for a spec at `Accepted` or better. 001's spec says `Implemented`,
which is [a statement about the exporting project](../../PROVENANCE.md) — *the WHAT is settled and
was proven once, elsewhere*. That satisfies the gate and nothing else: no route is served here.
This is the first plan in the repository, so the reading is recorded rather than assumed.

## 1. Approach

**001 is four endpoints and the whole edge, and the second half is the work.**

Three of the spec's sections are not about these endpoints at all. §3.0 is the wire format every
later response inherits, §3.5 is a `503` that applies to *every* route, and §3.6 is how any path
matches any route. The [roadmap](../../docs/roadmap.md#feature-order) says this plainly — 001 carries
the edge because it is the smallest feature that needs all of it — so this plan is mostly a plan for
the request pipeline, and the four handlers are nearly trivial on top of it.

The organising decision follows from that: **everything server-wide is a stage in one pipeline,
declared once and in order**, and a handler cannot opt out of any of it. The alternative — each
handler doing its share — is the failure [§1.7](../../docs/compatibility/behaviours.md) already
described about a per-route flag somebody eventually forgets.

**Two things were measured before this plan was written, and both changed it.**

`chi` answers `405` with an empty body, which is what [§1.11](../../docs/compatibility/behaviours.md)
wants. Its `Allow` header is wrong, and not only in ordering. On a path carrying `GET` and `POST` it
names **one** method, and which one varies with the request:

```
HEAD    /System/Ping -> 405  Allow: POST      PUT    /System/Ping -> 405  Allow: POST
OPTIONS /System/Ping -> 405  Allow: GET       DELETE /System/Ping -> 405  Allow: GET
```

`[measurement: github.com/go-chi/chi/v5 v5.3.2, Go 1.27.0, 2026-09-02]`

§3.6 requires every method **the path** has. So `Allow` is computed from the route table and set by
this project, and ADR-0002's first `⚠️ UNVERIFIED` is discharged as a finding rather than a
confirmation. The same run settles the other half of §3.6 in our favour: `HEAD` and `OPTIONS` are
already `405` with an empty body, so *"nothing is automatic"* costs nothing to hold.

**And §3.0's content-type profiles make the serialiser bigger than
[ADR-0002](../../docs/decisions/0002-go-and-the-runtime-stack.md) assumed.** Three media types, two
behaviours, chosen per request, with camelCase applied *at every depth* and dictionary keys never
converted. A struct tag is one name; this needs two, decided at write time. That is §6.3, and it is
the one place this plan adds a mechanism the ADR did not anticipate.

## 2. Inherited decisions

| Decision | Source |
|---|---|
| Go, `chi` over `net/http`, `encoding/json` behind one serialisation package, no cgo | [ADR-0002](../../docs/decisions/0002-go-and-the-runtime-stack.md) |
| Optional fields are pointers; `omitempty` on a non-pointer is banned | ADR-0002 |
| Embedded SQLite, pure-Go driver, hand-written SQL; the store split into a derived half and a precious half | [ADR-0003](../../docs/decisions/0003-sqlite-as-the-store.md) |
| Four layers, one direction; the domain imports no HTTP; the store and the clock are ports | [architecture §2](../../docs/architecture.md#2-layers-and-the-direction-of-dependency) |
| No map iteration reaches a response body; the wall clock is a port | architecture §2 |
| `internal/` for everything but `cmd` and `conformance`; `conformance/` speaks HTTP only | [architecture §3](../../docs/architecture.md#3-repository-layout) |
| Where each wire fact lives, and that middleware order is contract | [architecture §4](../../docs/architecture.md#4-the-compatibility-boundary) |
| One process, one data directory, a build-stamped `Server: Atrium/<version>` | [architecture §5](../../docs/architecture.md#5-deployment-shape) |
| 001 is first and carries the edge | [roadmap](../../docs/roadmap.md#feature-order) |

**Deviations:** none.

## 3. Modules

| Module | Change | Responsibility |
|---|---|---|
| `cmd/atrium` | new | The `main` that calls `internal/app` and turns an error into an exit status. No behaviour, no branch a test would want to reach. |
| `internal/app` | new | The entry layer: flags and their environment fallback, the logger, and the HTTP server's lifecycle — bind, serve, drain, stop. |
| `internal/build` | new | The link-time version stamp, read by the entry layer for a startup line and by the edge for `Server: Atrium/<version>`. A leaf, because both hold it. |
| `internal/units` | new | The tick and the date types. 001 sends neither — it is here because 001 delivers the unit sweep (§6 of the spec) and the sweep needs a type to recognise. |
| `internal/wire` | new | Every response body is written here. Encoder, the §1.16 escape pass, the two naming policies, the content type that names the one it used. |
| `internal/surface` | new | The route table: method, path, operation, owning feature, level — loaded from `surface.yaml` and the single source for routing, for `Allow`, and for the L0 registration check. |
| `internal/httpapi` | new | The pipeline: readiness gate, response-time stamp, `Server`, path and query canonicalisation, routing, refusals. Plus the four handlers. |
| `internal/system` | new | The domain: server identity, friendly name, setup state, and the three-tier `LocalAddress` choice. Imports no HTTP. |
| `internal/ports` | new | `Clock`, and the narrow store interface `system` needs. |
| `internal/store/sqlite` | new | The precious half's first table and its migration runner. |
| `conformance` | new | L0, L1 and L2 over a server started in process. Imports nothing of ours. |

**Amended 2026-09-03, at T1.** This table gave `cmd/atrium` the start and the stop, and that
cannot be where they live: [architecture §3](../../docs/architecture.md#3-repository-layout) says
`cmd/atrium` is wiring and nothing else, *"if something there is worth testing, it is in the wrong
place"* — and a server that binds, drains and stops on a signal is worth testing. T1's check starts
one in process and signals it, which needs a package a test can call. So the entry layer is
`internal/app`, `cmd/atrium` is a `main` that calls it, and the build stamp is `internal/build`
because the edge needs it for `Server` (behaviours §4.1) and the entry layer needs it for a startup
line, so it may live in neither.

**Why `surface` is its own package and not a slice in `httpapi`.** Three unrelated things read the
same table — the router, the `Allow` computation and the L0 test that asserts the server exposes
exactly `surface.yaml` and nothing else. A table that lives inside the router cannot be used to
check the router.

## 4. Data model

**Precious half.** One table, and it is the only state 001 owns that a user would miss.

| Column | Type | Note |
|---|---|---|
| `server_name` | TEXT NOT NULL | Operator-chosen. Default `atrium` |
| `setup_completed_at` | INTEGER NULL | Ticks; `StartupWizardCompleted` is `setup_completed_at IS NOT NULL` |

Single-row, guarded by `CHECK (id = 1)` on an `id INTEGER PRIMARY KEY` — a configuration table with
two rows is a bug that reads as a mystery.

**Derived half:** nothing. 001 scans nothing.

**Migrations:** forward-only, numbered, applied at start. `0001_installation.sql` creates the above.

**Amended 2026-09-03, at T3.** Three things this section left open, and the answers T3 took:

- **One file, `atrium.db`, for both halves.** The halves are two migration lineages and two rebuild
  policies, not two databases. ADR-0003 wants a backup to be one file, and a precious row naming a
  derived item by its identifier is an ordinary join that two files would need `ATTACH` for. What
  keeps them apart is architecture §6's rule that no reference points from the precious half into
  the derived one, which a second file would not enforce either. The runner keeps its own state in
  a `schema_version` table with **one row per half**, so the derived half sits at `0` here while the
  precious half is at `1` — the state a single lineage could not represent.
- **The runner takes a half.** Both lineages are loaded and applied at every start, and the derived
  one is empty rather than absent. The first feature with a derived table adds a file; it does not
  also change the runner. ADR-0003's *"a derived-version mismatch at startup is a rescan rather than
  an error"* is **not** implemented: rescanning needs a scanner, which is 003's, so today both
  halves refuse a version higher than the build knows. That refusal is owed a replacement in 003.
- **The entry layer creates the data directory**, and creates its **final component only**. T2 left
  this open and it could not stay open: the store cannot create the directory it lives in, because
  the identity file is read first and would fail before the store was reached. `os.MkdirAll` was
  rejected — a server that invents every directory on a mistyped `--data-dir` answers a typo with an
  empty installation that looks exactly like a fresh one, while the operator's data sits untouched
  under the path they meant. One component under an existing parent makes a first start work and
  makes `--data-dir /var/lbi/atrium` fail while naming `/var/lbi`.

**Amended 2026-09-03, at T4.** `setup_completed_at` says *"ticks"* and did not say ticks *since
when*, which is half a unit. It is **.NET's `DateTime.Ticks`: 100-nanosecond intervals since
`0001-01-01T00:00:00Z`** — the same origin the wire's dates have, because
[§1.3](../../docs/compatibility/behaviours.md#13-durations-and-positions-are-net-ticks) makes the
tick .NET's tick and every date the wire carries is a .NET `DateTime`. A column counting the same
unit from a different origin would be a second unit wearing the first one's name, and the store test
asserts the integer rather than only the boolean derived from it, because every candidate origin
makes `StartupWizardCompleted` equally true.

`MarkSetupComplete` lands with T4 rather than T3, which is the whole reason T3 left it off §5's
interface: it takes a `units.Time`. That makes `internal/ports` import `internal/units`, and
[architecture §2](../../docs/architecture.md#2-layers-and-the-direction-of-dependency)'s table says
a port may import *"nothing of ours"*. The same section's prose is what settles it — the unit types
are *"a leaf, imported by both"*, because §1.3 puts ticks in storage as well as on the wire *"so no
conversion can be forgotten at a boundary"*. §5 above already wrote both `Clock` and
`MarkSetupComplete` in terms of `units.Time`, so this is the plan's own contract rather than a
deviation, and a port taking a bare integer would be precisely the forgotten conversion.

### The server identity is a file, not a row, and AC-4 is why

AC-4 asks that `Id` be *"identical across a restart **and across a rebuild of the store from
empty**"*. A row in the store cannot satisfy the second clause, and neither can a random value.

Two ways out, and the second is chosen:

- **Derive it** from something stable, the way item ids derive from paths (Principle VII). The only
  stable input available is the data directory's own path — which means **moving the data directory
  changes the server's identity**, and every client re-authenticates. That is
  [§1.4](../../docs/compatibility/behaviours.md#14-item-identifiers-are-32-lowercase-hex-characters)'s
  library-root trap, reintroduced at the level of the whole server.
- **Persist it beside the store, not in it.** A single-line file, `installation-id`, in the data
  directory: 32 lowercase hex from a cryptographically random 16 bytes, written once with
  `O_EXCL` so two starts cannot race, read thereafter. A store rebuild does not touch it; moving
  the data directory carries it along.

**Amended 2026-09-03, at T2.** `O_EXCL` alone does not buy what the paragraph above claims. Creating
the file exclusively and writing to it afterwards leaves a window in which the file exists and is
empty, and a second start that reads inside that window finds a malformed identity and — by §7,
correctly — refuses to boot. Sixty-four rounds of eight concurrent starts hit it on round five. The
guarantee is kept by writing the line to a temporary file in the same directory and publishing it
with a **hard link**, which fails with `EEXIST` exactly as `O_EXCL` does and never exposes a
half-written file. *"Two starts cannot race"* is unchanged; the mechanism that delivers it is not
`O_EXCL` on the destination.

The file is **precious** in ADR-0003's sense even though it is not in the database, and §7 says what
happens when it cannot be read.

## 5. Contracts

```
// ports
Clock interface { Now() units.Time }

// what internal/system needs of the store, declared by the domain
InstallationStore interface {
    Installation(ctx) (Installation, error)
    SetServerName(ctx, string) error
    MarkSetupComplete(ctx, units.Time) error
}

// internal/system — no net/http in any signature
Identity struct { ID string; Name string; SetupCompleted bool }
LocalAddress(req RequestFacts, cfg AddressConfig) string

// RequestFacts is the domain's own view of a request: remote address, host,
// scheme, port. The edge fills it in; the domain never sees *http.Request.
```

**`RequestFacts` is the seam that keeps architecture §2 true.** `LocalAddress` needs the requester's
address, the host and the scheme, and taking an `*http.Request` for them would put HTTP in the
domain and make the three-tier table testable only by issuing requests.

```
// internal/wire
Naming int  // NamingPascal | NamingCamel
func Write(w http.ResponseWriter, status int, v any, n Naming) error
```

The status and the value go in together because [§1.10](../../docs/compatibility/behaviours.md)'s
content type belongs to the thing that produced the body, not to a middleware bolted on after.

**Amended 2026-09-03, at T5.** `NamingCamel` is **not declared yet**; T6 declares it with the
policy behind it. An exported constant that a caller may pass and that silently writes PascalCase
is worse than one that does not compile: the whole point of taking `naming` at T5 is that the
policy is negotiated, and a value that names a policy the package has not got would make the one
mistake this argument exists to prevent invisible. The signature above is otherwise exactly as
written.

## 6. Algorithms

### 6.1 Path canonicalisation (§3.6, behaviours §1.14)

At start, fold every route's **literal** segments to lower case into one map from folded path shape
to the canonical spelling. Per request, before routing:

1. Reject a path with two or more trailing slashes — `404`, empty. One trailing slash is trimmed.
2. Fold the literal segments and look up the canonical spelling; where a route declares a
   parameter, the incoming segment passes through **byte for byte**.
3. Rewrite the request's path to the canonical spelling and route that.

The router therefore only ever sees canonical paths, which is also why chi's own case sensitivity
is never exercised. No redirect is ever issued: [§1.14](../../docs/compatibility/behaviours.md)
records that a `307` here would be a second divergence rather than a smaller one.

### 6.2 Query key canonicalisation (behaviours §1.15)

The same shape, on names only. Each route declares its parameter spellings; an incoming key that
folds to one of them is rewritten to the declared spelling. **Values are never touched**, and an
unrecognised key is left alone so [§1.12](../../docs/compatibility/behaviours.md)'s ignore-don't-
reject rule still sees it — and so the ignored-parameter tally has something to count.

### 6.3 Choosing a naming policy (§3.0.2)

Parse `Accept` into media ranges with their `q`. A range matches when its type is
`application/json`; its `profile` parameter is compared **case-insensitively and unquoted**, and a
range carrying a `charset` parameter beside `profile` **does not match the profile** — it falls back
to the plain type. Rank by `q` descending, and on a tie **keep the client's order**. An unknown
profile value falls back to plain.

The winner decides two things together: the naming policy, and the content type echoed back —
`application/json; profile="CamelCase"; charset=utf-8`, profile before charset.

**The camelCase conversion is the reference's own policy and not "lower the first letter":** a
leading run of capitals lowers all but the last, so `UICulture` becomes `uiCulture` and `Id`
becomes `id`. It applies to **property names at every depth** and never to **dictionary keys** —
which is why it happens at the point where the encoder still knows which is which, and not as a
pass over finished bytes.

### 6.4 The escape pass (behaviours §1.16)

The encoder's own HTML escaping is switched off, and one pass rewrites the body: every non-ASCII
character and the seven ASCII ones as `\uXXXX` with upper-case hex. It **counts backslash parity**
rather than searching for the escape prefix, so a value that genuinely contains those six
characters survives while the encoder's own escapes are rewritten.

**Amended 2026-09-03, at T5**, with two things the wording did not survive contact with.

**The pass has to know where a string starts and ends.** §1.16's table escapes `"` as
`\u0022` and notes that *"JSON's own escape is `\"`, and the reference does not use it"* — so a
rewrite that treated every quote alike would escape the document's own delimiters and emit
something that is not JSON. Tracking it is cheap and exact rather than a parse: the encoder never
writes a raw quote inside a string, so every raw quote is a delimiter and every `\"` is a
character of a value.

**A character above U+FFFF has no `\uXXXX` spelling, and this is `⚠️ UNVERIFIED`.** §1.16
was measured on accented Latin characters and seven printable ASCII ones, every one of which fits a
single UTF-16 code unit. The pass writes a surrogate pair, because that is what a UTF-16 encoder
emits and the only spelling that reads back as the same character — but it is an inference from the
reference's stack and not a measurement, and it owes a probe that puts one such character into a
body. Nothing in 001 can carry one; the first feature that puts a library's item names on the wire
can.

### 6.5 `Allow` (§3.6)

From the route table: every method registered on **that path**, sorted alphabetically, joined with
`", "`. Not chi's, for the reason in §1 — and it is a property of the path rather than of the route,
so `/System/Ping` answers `GET, POST` to a `PUT`, a `HEAD` and an `OPTIONS` alike.

### 6.6 `LocalAddress` (§3.4)

Three tiers, in order, in `internal/system`:

1. A configured published URL, returned verbatim with one trailing `/` removed. **The configuration
   accepts the subnet-scoped form too** — `192.168.1.0/24=http://host:port` — because the reference's
   is per-caller and the most specific matching prefix wins; this is the branch
   [§2.3](../../docs/compatibility/behaviours.md#23-localaddress-is-one-string-and-may-be-https)
   gained on 2026-09-02.
2. Derive from the request: its own host and scheme, omitting the port when it is that scheme's
   default.
3. Match the requester against the bound addresses and return the one on the same network, **with
   the scheme and port the server is actually reachable on** — the deliberate divergence of
   [§4.2](../../docs/compatibility/behaviours.md#42-localaddress-does-not-get-an-https-override).

### 6.7 Pipeline order

Order is contract, so it is declared once and tested:

```
readiness gate → response-time stamp → Server header → path canonicalisation
  → query canonicalisation → routing → refusal shapes → handler → wire
```

The readiness gate is outermost because §3.5 exempts nothing, the stamp must wrap what it claims to
time, and canonicalisation must precede routing because it rewrites what the router matches.

## 7. Failure handling

| Failure | Detection | Response | Recovery |
|---|---|---|---|
| Server still starting | Readiness flag not yet set | `503`, `Retry-After` in full integer seconds, `Message` header, **`text/html`** body — never JSON (§3.5, AC-12) | Clears when start completes |
| Deliberate withdrawal | Operator sets it | The same `503` with a different message and a longer hint; the process stays up | Operator clears it |
| Path matches no route | Canonicalisation lookup misses | `404`, empty body, **no content type** | — |
| Two or more trailing slashes | Canonicalisation | `404`, empty body | — |
| Method the path does not have | Route table | `405`, empty body, no content type, `Allow` per §6.5 | — |
| `/System/Info` with no or bad token | Auth check | `401`, empty body, `Content-Length: 0`, no `WWW-Authenticate` (§1.11) | — |
| `installation-id` unreadable or malformed | Read at start | **Refuse to start**, naming the file | Operator restores or removes it |
| `installation-id` absent | Read at start | Create it | — |
| Store unopenable or migration fails | Start | Refuse to start | — |
| Handler panics | Recovery middleware | `500`; the shape is **not** measured and is [owed](#9-risks) | — |

**Refusing to start on a bad identity file is deliberate.** Generating a fresh id instead would make
every client treat the server as new and re-authenticate — a silent, expensive failure where a
refusal to start is a loud, cheap one.

## 8. Testing strategy

Each acceptance criterion becomes a named test at the level §6 of the spec declares.

| AC | Where | How |
|---|---|---|
| 1, 2, 3 | L1 golden | Byte-compared golden of `/System/Info/Public` on an empty installation; field values asserted separately so a golden diff says which field moved |
| 4 | L2 | Start, read `Id`, stop, delete the database, start again, assert identical — the store rebuild AC-4 names |
| 5 | L2 | `401` without a token; with one, every shared field equal to the public body |
| 6 | L1 golden | Exact bytes `"Jellyfin Server"`, both methods |
| 7, 8, 13 | L2 | Table-driven over the three tiers with synthesised `RequestFacts` — a pure-function test, which is what §5's seam buys |
| 9 | L1 | Plain and `PascalCase` byte-identical; `CamelCase` same values, camelCase names at depth, dictionary keys untouched, content type echoing the match |
| 10 | L1 sweep | Reflection over every registered response type: every field name PascalCase |
| 11 | L0 | Every route of `surface.yaml` registered and nothing outside it; casing and one trailing slash; two slashes `404`; empty `404` and `405`; `Allow` complete and alphabetical |
| 12 | L2 | Requests issued against a server held pre-ready: `503`, integer `Retry-After`, `Message`, `text/html` |

**The unit sweep ships here too** — every `*Ticks` an integer, every `*Date` seven digits and a `Z`
— even though 001 sends neither, because it is the first feature with a response model to sweep.
Its definition takes the correction the probe found: **a date is recognised by its value, not by its
name**, since `DateCreated` does not end in `Date`
([conformance L1](../../docs/compatibility/conformance.md#l1--shape)).

**Fixtures:** none. 001 answers before any user exists and before any library is configured, which
is precisely what makes it testable with nothing on disk but a data directory.

### 8.5 Routes against `surface.yaml`

*The number is the one [conformance.md](../../docs/compatibility/conformance.md#l0--routed) already
cites, from before this plan existed; §8 has no other numbered subsection because nothing else is
cited by number.*

L0 asks that the router expose **exactly** the surface file's rows and nothing else, and
conformance.md asks for it to be checked against **two views** of what the application serves,
because *"each view has a blind spot the other covers"*.

**The two views it names do not both survive the crossing.** They were the OpenAPI document a
framework generates and the route table a factory builds — and this server generates no OpenAPI
document, so the blind spot *"a route hidden from the document"* cannot exist here because the
document does not. Half the argument goes with it.

**The other half survives, and Go offers a better second view than the one it replaces.**

| View | What it sees | The blind spot it covers |
|---|---|---|
| **Registration** — `chi.Walk` over the built router, which enumerates every method and pattern including the partial-segment ones `[measurement: github.com/go-chi/chi/v5 v5.3.2, Go 1.27.0, 2026-09-02]` | What the router was *told* | A route registered without a `surface.yaml` row, and a row nothing registers |
| **Reachability** — a real request issued to every row's path from `conformance`, asserting it is not a `404` | What a client actually *gets* | A route registered correctly and made unreachable by something above it: a canonicalisation bug, a middleware that swallows it, a gate that never opens |

The second is the stronger of the two and has no analogue in the arrangement conformance.md
describes: a document generated from the application agrees with the application by construction,
where a request that has been through the whole pipeline agrees with nothing unless the pipeline
works. It is also the only one of the two that would catch T9 or T14 being wrong.

**What neither view covers** is a route that is registered, reachable, and answers the wrong thing.
That is what the golden tests are for, and it is why T20 is not the last task.

**L3 is deferred, not skipped**, on the spec's own terms: what is met now is L2, and the gap closes
the first time 010 runs.

## 9. Risks

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| The `500` shape is unmeasured — no probe has made the reference throw | High | A body no reference server sends, on every unexpected failure | Send an empty `500` with no content type, the most conservative of §1.11's shapes, and record it as owed. A probe that makes the reference `500` is [§3.9](../../docs/compatibility/behaviours.md)'s territory |
| Profile negotiation is measured on one probe run and is intricate | Medium | A client asking for CamelCase gets PascalCase, silently | AC-9 covers the three cases; the ranking and the charset rule get their own table-driven test |
| `Retry-After` and `Message` are declared by the document, and OQ-4 asks whether a running server emits both | Medium | Sending a header the reference does not | Both are sent; OQ-4 stays open and is a differential row |
| The escape pass runs on every body | Low | A copy per response | Measured if it ever shows in a profile; ADR-0002 already names the alternative |
| The canonicalisation map is built from the route table at start | Low | A route added without a table row is unreachable | The L0 registration test fails on exactly that |

## 10. Alternatives considered

**Let chi answer `405` and `404` itself.** It already returns an empty body, so it is closer than
the standard library. Its `Allow` is wrong — one arbitrary method, measured — and correcting a
header after the router has decided means reconstructing what the router knew. Computing it from the
table is less code and the table already exists.

**Do the camelCase conversion as a pass over the encoded bytes**, the way the escape pass works.
Rejected on §3.0.2's own wording: **dictionary keys are never converted**, and after encoding a
property name and a dictionary key are the same thing. The conversion has to happen where a field is
still a field.

**Put the server identity in the store.** One fewer file, and it fails AC-4's second clause, which
is not a detail: it is the criterion that says a client's session survives an operator rebuilding a
corrupted database.

**Derive the identity from the data directory path.** Symmetric with item identity and Principle VII,
and it makes moving the data directory a silent re-authentication of every client — §1.4's
library-root trap at the level of the whole server.

**Build the pipeline as ordinary handler wrapping, without declaring the order.** It works and the
order becomes an emergent property of the wiring code. §6.7 exists because the order is contract:
a stamp inside the gate times the wrong thing, and a middleware that adds a content type after a
refusal breaks §1.11 for every route at once.

**Skip `internal/units` until a feature sends a tick.** It would be dead code for one feature. It is
here because the sweep that makes ticks impossible to get wrong ships with 001, and a sweep needs a
type to look for.
