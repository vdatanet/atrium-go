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
wants. Its `Allow` header is wrong, and not only in ordering. ~~On a path carrying `GET` and `POST`
it names **one** method, and which one varies with the request:~~

```
HEAD    /System/Ping -> 405  Allow: POST      PUT    /System/Ping -> 405  Allow: POST
OPTIONS /System/Ping -> 405  Allow: GET       DELETE /System/Ping -> 405  Allow: GET
```

`[measurement: github.com/go-chi/chi/v5 v5.3.2, Go 1.27.0, 2026-09-02]`

**Amended 2026-09-03, at T11 — the reading above is wrong, and the conclusion it supports is not.**
Re-measured over a raw connection, with every `Allow` field line kept rather than one of them, on
the same chi and the same Go:

```
PUT /System/Ping -> 405  Allow: GET\r\nAllow: POST      FOO /System/Ping -> 405  (no Allow at all)
```

`[measurement: github.com/go-chi/chi/v5 v5.3.2, Go 1.27.0, 2026-09-03]`

chi names **both** methods, in **two field lines**, and there are three faults rather than one:

1. **Two field lines where the reference sends one comma-joined line.** §1.11 measured
   `Allow: DELETE, POST`. HTTP combines the two spellings into the same field value, which is why
   the difference is invisible to a client and visible to L3, and it is why the original reading —
   a reader that took one field line of two — saw a single method.
2. **The order is not stable.** chi builds the list by ranging over a Go map, so it is
   map-iteration order: over 200 identical requests it was `GET` then `POST` 171 times and `POST`
   then `GET` 29. §1.11 measured alphabetical, and Principle VII wants an order derived from a
   stable input rather than from a hash seed.
3. **A method token chi does not know reaches the `405` branch carrying no methods**, so the header
   goes missing entirely — and, before that, an unroutable *path* reaches the `405` branch too
   rather than the `404` one, because chi checks the method against its own nine before it routes.

§3.6 requires every method **the path** has. So `Allow` is computed from the route table and set by
this project, and ADR-0002's first `⚠️ UNVERIFIED` is discharged as a finding rather than a
confirmation. The same run settles the other half of §3.6 in our favour: `HEAD` and `OPTIONS` are
already `405` with an empty body, so *"nothing is automatic"* costs nothing to hold.

**The correction is kept rather than swallowed because of what it cost nothing to catch and would
have cost everything to miss.** A measurement of a header taken through a reader that returns one
field line per name cannot see a repeated header, and the same instrument reads Jellyfin. Nothing
in this feature depends on it — the reference's `Allow` was measured as one line by a probe this
project did not write — but a repeated header anywhere in the v1 surface would be invisible to a
run made the same way, and that belongs in 010's territory rather than in a footnote here.

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

**Amended 2026-09-03, at T8.** This table said `internal/surface` is "loaded from `surface.yaml`"
and left open the two things loading it actually turns on.

- **How the file reaches the binary: it is embedded, from a derived copy beside the package.**
  The document stays under `docs/`, where its prose twin is and where
  [docs/README.md](../../docs/README.md#paired-files-edit-both-halves-or-neither) governs it. But
  `go:embed` cannot name a path outside its own package directory, and reading
  `docs/compatibility/surface.yaml` off disk at run time would make the working directory part of
  the deployment — which contradicts ADR-0002's one static binary as directly as a system library
  would. So `internal/surface/surface.yaml` is a copy, and a test compares it to the document byte
  for byte. The alternative, generating a Go source file from the document, was not taken: it moves
  the refusals this task exists to ship out of the package and into a generator, and the task's
  wording puts them in the loader.
- **No YAML dependency.** ADR-0002 puts the standard library first and argues a further dependency
  "where it is needed, in the plan that needs it"; this plan does not name one, and the document
  does not need one. It is generated, and every line in it is a comment, a top-level key, a
  `key: value` or a flow sequence of scalars. The reader written here is strict where a general
  parser is lenient — an unknown key, a missing key, a repeated key, an unexpected indent and a
  value it cannot read are each an error naming the line — which is the property the surface file
  wants: a row that does not say what level it must reach is a row nobody decided about, not a row
  with a default.

Two refusals beyond the task's own pair fell out of writing it and are cheap enough to keep: an
`operation` declared on two rows, and two paths that differ only in casing. The second is the one
worth naming, because canonicalisation (T9) folds a request's literal segments case-insensitively
and would have no rule for choosing between two spellings that fold together.

**Amended 2026-09-03, at T18.** `internal/system`'s row says *"server identity, friendly name,
setup state, and the three-tier `LocalAddress` choice"* and now also holds **the installation's
path layout** — the seven directories spec §3.2 reports, derived from the data directory (§6.9).
It belongs in the domain for the same reason the identity does: it is a fact about *this
installation*, it reaches for no HTTP, and putting it at the handler would make the layout
something each response decided for itself.

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

**Amended 2026-09-03, at T6.** `NamingCamel` is declared, and with it the rule the argument was
being kept honest for. The same argument then applies once more to the *unknown* value: a `Naming`
this package does not recognise is an **error**, not a fall-through to PascalCase. A caller that
asked for camelCase and was answered in PascalCase is §1.13's failure mode exactly — an empty
object out of the client's decoder — and a silent fall-through is that failure with no way to see
it.

**Amended 2026-09-03, at T7.** `Write`'s last argument is a **`Profile`**, not a `Naming`:

```
// internal/wire
Profile int  // ProfilePlain | ProfilePascal | ProfileCamel
func Negotiate(accept string) Profile
func Write(w http.ResponseWriter, status int, v any, p Profile) error
```

Three declared content types over two naming policies (spec §3.0.2) is the whole reason. A body
written under `NamingPascal` cannot say which of `application/json` and
`application/json; profile="PascalCase"` asked for it, so the argument that was carrying the naming
policy could not also carry the echo, and §1.10's rule — *the content type belongs to the thing
that produced the body* — forbids the obvious repair of stamping the header in a middleware
afterwards. The negotiation therefore hands `Write` its single winner whole, and the winner's two
halves are read from one table.

`Naming` survives as the policy `marshal` dispatches on, and nothing outside the package produces
one any more. The T6 amendment's rule holds at both levels: an unknown `Profile` and an unknown
`Naming` are each an error rather than a fall-through to PascalCase.

**Amended 2026-09-03, at T18.** This section's list of contracts is complete for the domain and
silent about the one an authenticated route needs. The first authenticated route added
`httpapi.Authenticator`, and it is deliberately **not** in `internal/ports` beside the store —
§6.10 argues where it lives, why it takes an `*http.Request` where everything else in this plan
takes `RequestFacts`, and why the value that means *forbidden* is not declared yet.

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

**Amended 2026-09-03, at T9.** Three steps above, and three things they left open.

- **A segment is not always either literal or a parameter.** Step 2 reads as though the two kinds
  alternate one per segment, and five rows of `surface.yaml` say otherwise:
  `/Audio/{itemId}/stream.{container}` spells `stream.` itself and takes the rest from the client,
  and `/Videos/{itemId}/hls1/{playlistId}/{segmentId}.{container}` puts two parameters in one
  segment with a literal dot between them. The fold is therefore per **run** within a segment
  rather than per segment: `/audio/AbC/STREAM.MP4` canonicalises to `/Audio/AbC/stream.MP4`, the
  literal respelled and both parameters untouched. A parameter run ends at the leftmost occurrence
  of the next literal, from one byte in — it must match at least one byte, or `/videos/x/hls1/p/.ts`
  would be a segment with no name.
- **A literal path is looked up before a parametrised one.** `/items/filters` is `/Items/Filters`
  rather than `/Items/{itemId}` holding an item called `Filters`. Nothing in the table is ambiguous
  under that rule today, and among parametrised paths the document's own order decides
  (Principle VII); it is written down so that a row added later is a decision rather than something
  that falls out of a map.
- **The stage is a value with a `Wrap` method, and it is built once.** `NewPathFolder(table)`
  returns an error rather than panicking, because a table that cannot be folded is a failure to
  start (§7) and the entry layer is where a failure to start is reported. A method value is a
  `func(http.Handler) http.Handler`, so `folder.Wrap` is what a router's `Use` wants with no
  adapter. **This is the shape T10–T13 inherit**, and `internal/httpapi`'s package documentation
  states it.

The path folded is the request's **escaped** path, and the rewrite is published as both `Path` and
`RawPath` the way `net/http` keeps them. Folding the decoded path instead would segment on a `%2F`
a client percent-encoded precisely so that it would not be a separator. A percent-encoded *literal*
segment therefore does not fold — `/%53ystem/Info/Public` is not `/System/Info/Public` here — and
what the reference does with one has not been measured; it is a probe somebody owes, not a
behaviour this plan is claiming.

### 6.2 Query key canonicalisation (behaviours §1.15)

The same shape, on names only. Each route declares its parameter spellings; an incoming key that
folds to one of them is rewritten to the declared spelling. **Values are never touched**, and an
unrecognised key is left alone so [§1.12](../../docs/compatibility/behaviours.md)'s ignore-don't-
reject rule still sees it — and so the ignored-parameter tally has something to count.

**Amended 2026-09-03, at T10.** *"Each route declares its parameter spellings"* named no place for
a declaration to live, and there was none. `surface.yaml` carries a path, a method, an operation,
its consumers, the owning feature and the required level, and no parameters at all — while 001's
own four routes take no query parameter, so this feature cannot supply an example either. Three
ways were open and this is the decision taken, rather than an omission left standing.

- **The declarations live in Go, beside the routes that have them.** `httpapi.QuerySpellings` is a
  map from route to declared names, and `httpapi.V1QuerySpellings()` is the set the server runs on.
  It is **empty**, and the fold is a no-op on every request this server can answer today.
- **`surface.yaml` is not extended, and the cost is only half the argument.** It is a paired
  artefact ([docs/README.md](../../docs/README.md#paired-files-edit-both-halves-or-neither)): its
  prose twin, the derived copy beside `internal/surface` and the strict loader would all have to
  move in the same commit, to write an empty list on 59 rows — and the document's own header says
  it is generated against the pinned OpenAPI document by a tool that stayed in the source
  repository, so a column added here by hand is a column the next generation would not know to
  keep. The other half is what kind of fact a spelling is. §1.15 measured that the pinned document
  spells every parameter camelCase and the reference's own clients send PascalCase, **and both
  work** — so there is no single spelling the surface file could state. What this stage needs is
  the one spelling *this server's own handler* binds, which is the handler's to declare and not the
  surface's to record.
- **The mechanism ships now and its first source arrives with the first parameter.** §6.7 makes the
  order of the stages contract, so a stage inserted later is a change to that contract rather than
  a row in a map. If the list ever grows unwieldy — 005's item query alone has dozens — moving it
  into `surface.yaml` stays available, and moving a list that exists is a smaller decision than
  inventing a column for an empty one.
- **A declaration is keyed by route, not by path**, which is this section's own word. The reference
  binds parameters per action, and a path served by two methods may bind different ones on each, so
  a path-keyed declaration would rewrite a name on a method that never declared it — and §1.12 says
  an unrecognised name is left alone. A request never carries the pattern that matched it, so the
  stage borrows §6.1's fold to name the row a request belongs to before the router does.
- **A declaration the route table has no row for is a refusal**, at construction, the way §6.1's
  fold refuses a table it cannot describe. A name declared against a route that was renamed folds
  nothing and says nothing, and the failure it hides is the one this stage exists to prevent.

The fold runs on the query string's **own bytes**, undecoded, for the reason §6.1 gives for the
path. A percent-encoded *name* therefore does not fold — `%4Cimit` is not `limit` here — and what
the reference does with one has not been measured; it is a probe somebody owes, of exactly the same
shape as the one §6.1 records.

[architecture §4](../../docs/architecture.md#4-the-compatibility-boundary) puts case-insensitive
query names in *"the same middleware"* as case-insensitive paths. The point of that row is where the
behaviour may **not** be — *"a handler reading two spellings"* — and §6.7 fixes the pipeline with
the two as adjacent stages, which is what shipped: two stages sharing one fold.

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

**Amended 2026-09-03, at T6**, with where that point is in Go.

`encoding/json` has no seam at which a property name is still a property name: its encoder emits
struct fields and map keys through the same call, and a `MarshalJSON` hook sees a whole value
rather than one name. Re-implementing the encoder to get that seam would put this project's models
at the mercy of a second reading of tag rules, embedding and `omitempty`.

So the rename **walks the encoded document beside the value it was encoded from**. The bytes decide
the structure — they are the encoder's own, and every number, string and escape is copied out of
them unchanged — and the value answers the one question the bytes cannot: whether this object's
keys came from a struct's fields or from a map's keys. That keeps `encoding/json` as the single
authority on what a body contains, and confines this package to what it is named after.

Two consequences worth stating rather than discovering:

- **A value that writes its own JSON is not renamed**, because its names are not fields by the time
  they exist. This is the reference's behaviour too — its naming policy is applied by the object
  converter from a type's property metadata, and a custom converter that writes members never
  consults it `[source: src/Jellyfin.Extensions/Json/JsonDefaults.cs:34-45,55-58 @ v10.11.11]`. A
  model that needs both policies must therefore not write its own members. `units.Time` is safe: it
  writes a string.
- **A shape the walk cannot account for is a refusal, not a copy.** Half a body converted is the
  same wrong answer as none of it, and it would be invisible; `Write` has already promised to send
  nothing when it returns an error, so the caller can still refuse.

**Amended 2026-09-03, at T7**, with three things the four rules did not settle and one that the
measurement behind them does not reach.

**Where it lives.** `wire.Negotiate(accept string) Profile`, beside the writer, because §3's module
table already puts *"the content type that names the one it used"* there and the parse has no other
consumer. It takes the header value rather than a request: where a request carries more than one
`Accept` field line, the caller joins them with a comma first (RFC 9110 §5.3 makes that the same
header), and that is T14's to do.

**A wildcard range is a candidate, and it names no profile.** `*/*` and `application/*` rank as
ordinary ranges answered with the plain type, so `*/*, application/json; profile="CamelCase"`
answers **plain** on the tie. The reference does not discard a wildcard before ranking: it sets
`RespectBrowserAcceptHeader`, under the comment *"Allow requester to change between camelCase and
PascalCase"* `[source: Jellyfin.Server/Extensions/ApiServiceCollectionExtensions.cs:125-126 @ v10.11.11]`.

**Parameters written after `q` do not select a profile.** They are accept-extensions and belong to
the negotiation rather than to the media type (RFC 9110 §12.5.1), so
`application/json;q=0.9;profile="CamelCase"` asks for plain JSON. ⚠️ UNVERIFIED: no probe has sent
one, and this is the RFC's reading rather than a measurement.

**And the rule that a `charset` "falls back to the plain type" has two readings, which no
measurement separates.** Either the range becomes a candidate *for the plain type* and keeps its
place in the ranking, or it matches nothing and the next range is tried. Every measured case is a
single-range `Accept`, where both readings answer plain and agree. They differ on one shape:

```
Accept: application/json; profile="CamelCase"; charset=utf-8, application/json; profile="CamelCase"
```

— plain under the first reading, camelCase under the second. This implementation takes the **first**,
because it is what this plan and spec §3.0.2 say in as many words. It is one request to settle, and
it is owed a probe.

**Amended 2026-09-03, at T14 — the join has a home, and it is not a stage.**

`httpapi.NegotiateProfile(r)` is `wire.Negotiate(strings.Join(r.Header.Values("Accept"), ","))`, and
a handler calls it. `Header.Get` would have answered only the *first* field line, so a client
sending `Accept: text/plain` and `Accept: application/json; profile="CamelCase"` on two lines would
be answered in PascalCase — [§1.13](../../docs/compatibility/behaviours.md)'s failure mode exactly,
an empty object out of the client's decoder.

It is deliberately **not** a middleware carrying a `wire.Profile` in the request context, which is
what "the negotiated profile travels to the handler" invites. Negotiation writes nothing, refuses
nothing and answers nothing, so it is not one of §6.7's stages — it is the *handler → wire* step
that order already ends with. And a `Profile` read out of a context is `ProfilePlain` in any handler
tested without the stage installed: silently, with a correct-looking body and a content type that
lies about what was asked for, which is the failure AC-9 exists to catch. Taking the request means
the answer cannot be wrong for want of a stage. §6.7's order is therefore unchanged by this task.

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

**Amended 2026-09-03, at T11.** Two things the rule did not say, both of which the router forces a
decision on.

A request never carries the pattern that matched it, so the lookup is `/Items/abc` →
`/Items/{itemId}` → the table, using canonicalisation's own `pattern` (§6.1). Reading the request
path against the table directly works for every path 001 serves and for none that takes a
parameter.

And **a path the table has no row for is a `404`, whatever the method**. chi checks the request's
method against the nine it knows *before* it routes, so `FOO /Nowhere` arrives at the
method-not-allowed branch rather than at the not-found one. §3.6 keys its `404` on the path — *"a
path matching no route"* — and says nothing about the method, so the path decides; a `405` there
would have to carry an `Allow` naming methods that do not exist. This is a reading of §3.6 rather
than a measurement: what the reference answers to an unknown method **token** has not been probed,
and it is one request to settle.

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

**Amended 2026-09-03, at T15.** Four things the three lines above did not settle, and one of them
is a contradiction with the reference's own source that is recorded rather than resolved.

**~~one trailing `/` removed~~ — every leading and trailing `/` is removed.** The reference spells
it `PublishedServerUrl.Trim('/')`
`[source: Emby.Server.Implementations/ApplicationHost.cs:877 @ v10.11.11]`, and .NET's `Trim(char)`
takes every one of them from both ends, not one from the end. §3.4 says *"any trailing `/`
removed"*, which the source matches and this line did not; a published URL configured with two
trailing slashes would have come back carrying one. The scoped form reaches the same place by a
different route — a value that starts with `http` is returned by `GetLocalApiUrl` after
`TrimEnd('/')` `[source: Emby.Server.Implementations/ApplicationHost.cs:932-935 @ v10.11.11]` —
so both forms are trimmed alike.

**The order of tiers 1 and 2 contradicts the source, and the specification is implemented as
written.** `LocalAddress` is served by the `HttpRequest` overload of `GetSmartApiUrl`
`[source: Emby.Server.Implementations/SystemManager.cs:77, 120 @ v10.11.11]`, and that overload
tests `EnablePublishedServerUriByRequest` **first**, reaching `PublishedServerUrl` only when it is
off `[source: Emby.Server.Implementations/ApplicationHost.cs:885-901 @ v10.11.11]`. §2.3's
corrected table numbers the branches the other way round, and §3.4 follows it. The two readings
disagree on exactly one installation — a published URL set **and** derivation switched on — and on
nothing else. AGENTS.md §1.3 makes the running server the authority and there is none here, so this
is **not** closed by editing the specification on source evidence: it is **one request to settle**,
against a reference configured with both, and failing that it surfaces in 010's differential run as
a single undeclared difference on `/System/Info/Public`. This is the same shape as §6.8, and it is
recorded for the same reason.

**The certificate is an input this function takes and deliberately does not read.** §4.2's argument
is that v1 has no certificate configuration, so the state in which the reference rewrites the scheme
*cannot be configured on Atrium at all* — which would leave the divergence with nothing to assert.
A divergence with no input can only be assumed. `AddressConfig` therefore carries
`CertificateConfigured` and `HTTPSPort`, the two inputs the reference consults, and no branch reads
either; T15's check sets them, asserts the answer does not move on any tier, and fails the day one
does. This is not a v1 configuration surface — nothing at the entry layer sets them — and Principle I
is untouched, because neither reaches the wire.

**Two answers the tiers do not cover, decided here.** `BoundAddresses` is in **preference order**
and tier 3 takes the **first** entry whose subnet contains the requester, which is what the
reference does with its interfaces
`[source: src/Jellyfin.Networking/Manager/NetworkManager.cs:891-905 @ v10.11.11]`; the ordering
itself is the caller's to establish, so no order derives from a map (Principle VII). And an
installation with **no** published URL, **no** derivation and **no** bound address answers from the
request anyway, because an empty `LocalAddress` is a field every client reads and none can use.

### 6.7 Pipeline order

Order is contract, so it is declared once and tested:

```
readiness gate → response-time stamp → Server header → path canonicalisation
  → query canonicalisation → routing → refusal shapes → handler → wire
```

The readiness gate is outermost because §3.5 exempts nothing, the stamp must wrap what it claims to
time, and canonicalisation must precede routing because it rewrites what the router matches.

**Amended 2026-09-03, at T12 — this order and T14's own acceptance cannot both hold, and which of
them moves is T14's decision rather than T12's.**

A middleware that answers without calling the next handler is never reached by anything below it.
The order above puts the readiness gate *above* the response-time stamp, so a `503` from the gate
carries neither `X-Response-Time-ms` nor `Server` — while [T14](tasks.md)'s *Verified by* line asks
for exactly the opposite: *"a `503` from the gate still carries the response-time stamp and
`Server`"*. `TestAStageOutsideTheStampIsNotStamped` asserts the constraint rather than describing
it, so whichever way it is resolved, it is resolved deliberately.

There are two ways out and **the reference has already taken the first**: its response-time
middleware is registered near the outside of the main pipeline and its startup gate well inside it
`[source: Jellyfin.Server/Startup.cs:163,217 @ v10.11.11]`, so the `503` that pipeline answers while
the server is loading is stamped. The second is a gate that writes both headers itself, which is the
per-handler duplication §1 exists to prevent — and it would have to be repeated by every later stage
that refuses. Neither costs anything against §3.5: a gate one stage further in still exempts no
route, because nothing between the two stages routes.

T12 ships two stages that satisfy either reading: both are ordinary `Wrap` middlewares, and the
stamp is a `ResponseWriter` decorator rather than a handler, so anything *inside* it is stamped
whatever wrote the response.

**Amended 2026-09-03, at T13 — resolved, the first way out. The order is:**

```
response-time stamp → Server header → readiness gate → path canonicalisation
  → query canonicalisation → routing → refusal shapes → handler → wire
```

~~The readiness gate is outermost~~ — it is third, immediately inside the two stages that stamp
every response and immediately outside everything that could route. Three reasons, in the order
they bind:

1. **The reference resolves it this way already.** Its response-time middleware sits near the
   outside of the main pipeline and its startup gate well inside it
   `[source: Jellyfin.Server/Startup.cs:163,217 @ v10.11.11]`, so the `503` that pipeline answers
   while loading is stamped. Principle I settles a choice this project would otherwise be making
   for itself.
2. **The alternative is the duplication §1 exists to prevent.** A gate that wrote
   `X-Response-Time-ms` and `Server` itself would be the second place each of those values is
   spelled, and every later stage that refuses without calling the next handler — 002's `401` is
   the next one — would have to do the same.
3. **It costs §3.5 nothing.** Nothing between the stamp and the gate reads the path, matches a
   route or refuses; the two stages above set a header each and call the next handler
   unconditionally. So *"nothing is exempt"* is exactly as true one stage further in, and the gate
   still answers before routing, which is what makes an unknown path a `503` rather than a `404`
   while the server is starting. T13 asserts that row by name.

T13 was the one to decide it rather than T14, because T14 only assembles what the stages already
are: the constraint is a property of the gate, and T13 is where the gate is written. T14's
*Verified by* line stands unchanged and is achievable against this order — the amendment moves the
plan to meet the acceptance, not the other way round.

### 6.8 The readiness gate (§3.5, AC-12)

The gate is shut from the moment it is built and opened once, by the entry layer, after the start
has finished. A gate that had to be told to refuse would serve for the length of whatever went
wrong before the first `MarkReady`, which is the window §3.5 describes.

Three values go on the wire, rendered when the state changes rather than per request:

| Part | While starting | On a deliberate withdrawal |
|---|---|---|
| `Retry-After` | `5` | The operator's hint, rounded **up** to a whole second |
| `Message` | `Jellyfin Server is loading. Please try again shortly.` | The operator's reason |
| Body | The message, in a minimal `text/html` document | The reason, in the same document |

The starting message is the reference's own localised string
`[source: Jellyfin.Api/Middleware/ServerStartupMessageMiddleware.cs:45-48 @ v10.11.11]`
`[source: Emby.Server.Implementations/Localization/Core/en-US.json:79 @ v10.11.11]`, and five
seconds is the reference's own hint
`[source: Jellyfin.Server/ServerSetupApp/SetupServer.cs:143 @ v10.11.11]`. The rounding is **up**
because a hint that under-states when to come back invites the retry storm the header exists to
prevent, and a hint of zero or less is refused rather than rounded, because a caller asking for no
delay has not thought about the header.

An operator's reason is validated before it is stored, not when it is written: it becomes a header
field value, and one carrying `CR` or `LF` would end the `Message` field line and let the rest be
read as further headers — response splitting out of a configuration string. Go drops such a header
silently at write time, which is both the wrong place and the wrong answer. The same reason is
HTML-escaped in the body and left as written in the header, because only the body is parsed as
HTML.

#### §3.5's *"nothing is exempt"* is contradicted by the reference's own source, and this plan owes a probe

**This is the largest open question 001 leaves behind, and it is recorded here rather than acted
on.** §3.5 and AC-12 cite the **pinned document**: every operation declares a `503` carrying
`Retry-After`, `Message` and a `text/html` body, 389 of 389, because an OpenAPI operation filter
attaches that declaration to all of them at once
`[source: Jellyfin.Server/Filters/RetryOnTemporarilyUnavailableFilter.cs:7-51 @ v10.11.11]`. What
the reference *runs* is not that, in three measured-by-reading places:

1. The `503` answered before the application exists comes from a **separate setup web server**, not
   from the main pipeline `[source: Jellyfin.Server/ServerSetupApp/SetupServer.cs:177-259 @
   v10.11.11]`. That server registers no response-time middleware, so its `503` very likely carries
   no `X-Response-Time-ms` at all — which, if true, is a difference on every `503` this project
   sends.
2. That setup server answers a **real** `/System/Info/Public` body, with `StartupWizardCompleted`
   false, rather than a `503` `[source: .../SetupServer.cs:204-237 @ v10.11.11]`. One route is
   exempt there, and it is the first request every client makes.
3. The **main** pipeline's gate exempts `/system/ping` case-insensitively and sends **neither**
   `Retry-After` nor `Message` — only the status, `text/html` and the localised string
   `[source: Jellyfin.Api/Middleware/ServerStartupMessageMiddleware.cs:38-48 @ v10.11.11]`. So the
   two headers are only ever sent together by the setup server, which exempts two paths of its own.

[AGENTS.md §1.3](../../AGENTS.md) ranks the three sources: where a probe, a source line and the
pinned document disagree, **the running server wins**. There is no running reference here and no CI
job may start one ([AGENTS.md §1.6](../../AGENTS.md)), so the disagreement **cannot be settled in
this repository**. §3.5 and AC-12 are therefore implemented as written — every route, no exemption,
both headers, a `text/html` body — because the spec is the authority on WHAT and source evidence
alone does not discharge it. **`spec.md` is deliberately not amended.**

What settles it: **one probe against a reference caught while it is starting**, issuing the four
v1 routes and one unrouted path and recording status, `Retry-After`, `Message`, `Content-Type` and
`X-Response-Time-ms` on each. That is a hard probe to run — the window is short — and it belongs in
[reference-target.md](../../docs/compatibility/reference-target.md)'s register of measurements that
still owe one. Failing that, **010's differential run is where it surfaces**, as up to five
undeclared differences on the starting server. Whichever way it resolves, the change is to `spec.md`
§3.5 and to `allowlist.yaml`, not to this stage's shape.

Two smaller items are owed by the same probe. The reference's setup server zero-pads `Retry-After`
to three digits — `005` for its five-second hint `[source: .../SetupServer.cs:242 @ v10.11.11]` —
which parses as the same integer and is **not** what the pinned document declares; this project
sends `5`. And the `text/html` body's bytes are **⚠️ UNVERIFIED**: the main pipeline writes the bare
message, the setup server renders a page out of the startup log, so there is no single body to copy
and §3.5 asks only for the media type.

### 6.9 What `/System/Info` answers with (§3.2)

*Added 2026-09-03, at T18. This plan named no value for any of spec §3.2's twenty additional
fields, and T18's own wording — "with the plan's stated values for the flags and paths" — pointed
at a section that did not exist. Here it is, with what each answer turns on.*

**The superset is structural.** The public model is **embedded** in the authenticated one and
filled in by the one function that fills the public response, so the seven shared members are one
value each rather than two values that agree. AC-5's *"agreeing on every shared field"* is then a
property of the code rather than a promise a test has to keep checking — which it also does, over
the wire, member by member as raw JSON.

**The flags are all `false`, and two of them differ from the reference on purpose.** Atrium has no
self-update, no restart, no browser to launch and no filesystem watcher, so `HasPendingRestart`,
`IsShuttingDown`, `CanSelfRestart`, `CanLaunchWebBrowser`, `HasUpdateAvailable` and
`SupportsLibraryMonitor` are false. The reference answers `SupportsLibraryMonitor = true`
unconditionally `[source: Emby.Server.Implementations/SystemManager.cs:79 @ v10.11.11]` and
`CanSelfRestart = true` by a default it marks obsolete
`[source: MediaBrowser.Model/System/SystemInfo.cs:67-69 @ v10.11.11]`. Both differences are the
spec's own answer and both are honest: a client that started a watch-dependent flow on a `true`
would wait for a notification that never came.

**`WebSocketPortNumber` is the port this process is actually listening on**, which forced a change
at the entry layer. The handlers are built before the pipeline, the pipeline before the server, and
the server is what binds — so at construction the port is not known, and under `--bind-address :0`
it is not yet chosen. It therefore arrives as a **function**, which the entry layer fills in after
it binds and before it opens the readiness gate. The alternative — parsing the port out of the
configured bind address — answers `0` for every server started on port 0 while it is happily
serving requests on a real one.

**The seven paths are derived from the one path an operator configures.** 001's only path
configuration is `--data-dir`, and the layout is:

| Field | Value |
|---|---|
| `ProgramDataPath` | the data directory, exactly as `--data-dir` gave it |
| `CachePath` | `<data>/cache` |
| `LogPath` | `<data>/log` |
| `InternalMetadataPath` | `<data>/metadata` |
| `ItemsByNamePath` | **the same value as `InternalMetadataPath`** |
| `TranscodingTempPath` | `<data>/cache/transcodes` |
| `WebPath` | `<data>/web` |

Three things about that table. The names follow the reference's own defaults — `metadata`
`[source: Emby.Server.Implementations/ServerApplicationPaths.cs:36 @ v10.11.11]`,
`cache/transcodes`
`[source: MediaBrowser.Common/Configuration/EncodingConfigurationExtensions.cs:35 @ v10.11.11]` —
not for compatibility, since no client reads these and every one of them differs per installation
in a differential run, but because an operator who has run the reference already knows where to
look. `ItemsByNamePath` is *not* a directory of its own: the reference fills both fields from
`InternalMetadataPath` `[source: Emby.Server.Implementations/SystemManager.cs:71-72 @ v10.11.11]`,
so two fields carrying one value is a fact about the response. And **none of these directories is
created**: 001 creates the data directory and nothing inside it (§4, at T3), so a path field is an
address rather than a promise that something is at it, and the feature that first needs one creates
it there.

The data directory is reported **as it was given**, not resolved to an absolute path. What the
response says, what `--data-dir` said and what the `starting` log line printed are then one string.

**The four remaining strings are empty, and that is spec §3.2's rule rather than a shrug.**
`OperatingSystemDisplayName`, `SystemArchitecture` and `EncoderLocation` are each marked obsolete
in the reference and never assigned, so the value it sends is a stale constant — `""`, `"X64"` and
`"System"` respectively `[source: MediaBrowser.Model/System/SystemInfo.cs:28-29,137-143 @
v10.11.11]`. §3.2 asks for *"real values where meaningful, empty string otherwise"*, and a field
whose own declaration says it is not set has no meaningful value to give: copying `"X64"` would
assert something false about this host, and reporting the host's real architecture would put a
string on the wire that no reference server sends, on a field no reference server fills. Empty
asserts nothing. §8 records the two differences that follow.

**`PackageName` is not sent at all**, and that is the one place this section departs from a plain
reading of §3.2's table. See the amendment in `spec.md` §3.2: §3.0.3 subordinates the general rule
to a per-field verification wherever one exists, and behaviours §1.7 is that verification — the
reference declares this property on this response and does not send it
`[probe: tools/probe_public_info.py, Jellyfin 10.11.11, 2026-08-28]`.

**`CompletedInstallations` and `CastReceiverApplications` are empty arrays, not nulls**, and their
element type is deliberately left unspecified. The reference's are models of a package manager and
a cast receiver that Atrium does not have; declaring their members here would be a schema for a
value nothing can produce. The feature that ever fills one declares what it fills it with.

### 6.10 Authentication is a port, and 001 fills it with nothing

*Added 2026-09-03, at T18.*

`/System/Info` is the first authenticated route, and **002 owns authentication**: five mechanisms
with a measured precedence between them, tokens that name a session, users that hold permissions
(002 spec §3.1). None of that is written here. What is written is the seam:

```
// internal/httpapi
Access int  // AccessUnauthenticated | AccessGranted
Authenticator interface { Authenticate(*http.Request) (Access, error) }
```

Three decisions in six lines.

**It takes the request, and therefore lives in the edge package rather than in `internal/ports`.**
Everything else the domain needs of a request reaches it as `RequestFacts` (§5), and a credential
cannot: it is three header names and two query names, each with its own grammar and a measured
order between them when two disagree. Reducing that to a value *before* this interface means
implementing 002's reader, which is the one thing this task must not do. The domain half — a token
to a session, a session to a user, a user to a permission — is 002's to declare in `ports`, and it
is not this interface.

**`AccessUnauthenticated` is the zero value, and 001 passes no `Authenticator` at all.** A server
that has issued no token recognises no credential, so *every* credential is unrecognised — which
is the true answer rather than a stub, and it is what makes the `401` of §7 reachable and testable
now.

**The third value is missing on purpose.** The reference answers a valid credential without the
route's permission with `403`, and clients branch on the difference. 001 routes nothing that can
reach it and behaviours §1.11 gives that shape a body this feature has no measurement for, so
declaring the constant would be a plausible-looking stub in an enumeration (Principle VI). An
`Access` the handler does not recognise is an **error**, never a fall-through — `internal/wire`'s
rule for an unknown `Profile`, for the same reason: the two directions a fall-through could take
are *admit everybody* and *refuse everybody*, and both are silent. That is the test which fails the
day 002 adds the value without teaching the handler about it.

**The first-time-setup exemption stays at the handler, not in the port.** §3.2 permits the route
while setup is outstanding, and the reference's authorisation handler succeeds on
`!IsStartupWizardCompleted` before it looks at a role
`[source: Jellyfin.Api/Auth/FirstTimeSetupPolicy/FirstTimeSetupHandler.cs:29-31 @ v10.11.11]`; an
unrecognised token does not change that, because its authentication handler answers *no result*
rather than a failure and leaves authorisation to decide
`[source: Jellyfin.Api/Auth/CustomAuthenticationHandler.cs:48-51,79-83 @ v10.11.11]`. That is a
fact about **this route's policy**, so it belongs with the route. It also fixes the order the
handler asks its two questions in: the setup state comes from the store, so the installation is
read before admission is decided, and the same read supplies `StartupWizardCompleted` afterwards.

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
| 5 | L2 | `401` without a token; with one, every shared field equal to the public body — ~~one row~~ **two levels; see the T18 amendment below** |
| 6 | L1 golden | Exact bytes `"Jellyfin Server"`, both methods |
| 7, 8, 13 | L2 | Table-driven over the three tiers with synthesised `RequestFacts` — a pure-function test, which is what §5's seam buys |
| 9 | L1 | Plain and `PascalCase` byte-identical; `CamelCase` same values, camelCase names at depth, dictionary keys untouched, content type echoing the match |
| 10 | L1 sweep | ~~Reflection over every registered response type: every field name PascalCase~~ **two halves — reflection over every registered response type, and a walk of every response's bytes; see the T19 amendment below** |
| 11 | L0 | Every route of `surface.yaml` registered and nothing outside it; casing and one trailing slash; two slashes `404`; empty `404` and `405`; `Allow` complete and alphabetical |
| 12 | L2 | Requests issued against a server held pre-ready: `503`, integer `Retry-After`, `Message`, `text/html` |

**The unit sweep ships here too** — every `*Ticks` an integer, every `*Date` seven digits and a `Z`
— even though 001 sends neither, because it is the first feature with a response model to sweep.
Its definition takes the correction the probe found: **a date is recognised by its value, not by its
name**, since `DateCreated` does not end in `Date`
([conformance L1](../../docs/compatibility/conformance.md#l1--shape)).

**Fixtures:** none. 001 answers before any user exists and before any library is configured, which
is precisely what makes it testable with nothing on disk but a data directory.

**Amended 2026-09-03, at T16.** This section named the levels and never said what a conformance test
*is* here, and the first one had to decide it.

- **`conformance/` starts the binary, not a server object.**
  [architecture §3](../../docs/architecture.md#3-repository-layout) forbids this package from
  importing anything under `internal/`, which rules out `app.NewServer` along with everything else —
  so the harness builds `./cmd/atrium` once in `TestMain` and runs one process per test on
  `127.0.0.1:0`, reading the bound address back out of the server's own log line, which is the only
  place a caller that did not choose the port can learn it. It reaches nothing outside this machine
  (AGENTS.md §1.6). The cost is a `go build` per package run; what it buys is that everything a
  conformance test knows is something a client could have known.
- **A golden holds the exact bytes, and stating an input is how one is made possible.** Two of
  §3.1's seven fields derive from the run rather than from the specification — `Id` is random on a
  first start, `LocalAddress` is built from the request — and normalising either away would give up
  the byte comparison that is the whole point (Principle VIII). So the identity file is **written
  before the server starts** and the `Host` header is **stated in the request**, and the recorded
  body is spec §3.1's own. The per-field assertions run on a fresh installation instead, where `Id`
  is asserted by shape. Every later golden in this repository has the same problem to solve and can
  copy the shape: hold the derived input still, do not soften the comparison.
- **The update flag rewrites and then fails.**
  [architecture §8](../../docs/architecture.md#8-testing-and-conformance) says a golden is reviewed
  rather than regenerated, and a flag that left the run green would make it a record of whatever the
  code last did. `-update-golden` writes the file and fails the test with the new bytes in the
  message; the suite is re-run without it once the diff has been read.
- **Key order is part of the response and is not measured.** L3 compares bytes. The order
  implemented is §3.1's, which is also the reference model's declaration order
  `[source: MediaBrowser.Model/System/PublicSystemInfo.cs:14-53 @ v10.11.11]` — but no probe in this
  repository records the key order of any body, and this repository's two sample bodies for this
  route disagree about it. Marked `⚠️ UNVERIFIED` at the model. **One request settles it**, and
  failing that it is an undeclared difference in 010's run. Same shape as §6.6 and §6.8.
- **`LocalAddress` reaches only its fallback in a v1 deployment, and that is not a defect of §6.6.**
  001 declares no configuration surface for any of the three tiers — no published URL, no derive
  flag, no bound-address list — so the entry layer passes the zero `AddressConfig` and every
  installation this binary can be started as answers from the request itself, which §6.6 already
  states as the deliberate answer for an installation with none of the three. The domain function is
  whole and tested over all three tiers (T15); what is missing is a caller, and the feature that adds
  the configuration is where it arrives. Worth naming because a reader of §6.6 would otherwise
  expect tier 3 to be reachable today: it is not.

**Amended 2026-09-03, at T18.** AC-5's row above is one line and it is three claims, two of which
this repository cannot make from `conformance/`.

- **The superset half is over the wire, against the running binary.** Both routes are issued to one
  server with one `Host`, and the two bodies are compared **member by member as raw JSON** — the
  only level at which a shared field that had changed type, `true` becoming `"true"`, is a
  difference at all.
- **The `401` half is not reachable from `conformance/`, and it is asserted at the HTTP boundary in
  `internal/httpapi` instead.** It needs an installation whose setup is *complete*, and 001 serves
  no route that completes setup — 002 owns it, and `conformance/` cannot reach into the store to
  say so (that is the rule that makes the package worth having). So the state is stated through the
  store at the handler level, and the request goes over a **real connection** rather than to a
  recorder, because three of the four things behaviours §1.11 measures about that shape —
  `Content-Length: 0`, no `Content-Type`, no `WWW-Authenticate` — are invisible to
  `httptest.ResponseRecorder` (T11's finding).
- **The remaining half, *"and `200` with a valid one"*, cannot be met here at all.** No credential
  exists until 002. It is recorded as a **criterion carried into 002** rather than marked met
  (`tasks.md` T18, T21). What *is* proven now is that the handler asks the port and obeys it, over
  a stub — which is a fact about the wiring and not about any token.

**There is no golden for `/System/Info`, and spec §6's row for it is amended to say so.** Seven of
its fields are the installation's own paths and one is the port the operating system chose, so a
recorded body would either match only on the machine that wrote it or have to be softened until it
stopped being a byte comparison — and softening is exactly what the T16 amendment above forbids.
What a golden buys is bought by two assertions instead: the **property names in order**, which is
the count, the order and the absence of `PackageName` in one line, and the per-field raw-JSON
values. Both are byte-level; neither can be held still as a file.

**Amended 2026-09-03, at T19. The two sweeps are split, and the split is not a compromise.**
[architecture §8](../../docs/architecture.md#8-testing-and-conformance) put both reflection sweeps
in `conformance/` and [§3](../../docs/architecture.md#3-repository-layout) forbids that package from
importing `internal/`; the import rule wins, and architecture §8 carries the amendment. What each
half is worth is the part to read:

- **The model sweep is in `internal/httpapi`**, over `reflect.Type`. It walks pointers, slices,
  dictionary *values* and embedded structs — `SystemInfo` embeds `PublicSystemInfo`, and a walk that
  did not recurse into an anonymous field would see one field named after a type and miss seven. It
  is the only half that sees a field no response has carried yet: `PackageName` is never sent, and
  its name is still contract.
- **The wire sweep is in `conformance/`**, over bytes. It is the half that reaches the corrected
  rule: **a date is recognised by its value**, since `DateCreated`, `DateLastMediaAdded` and
  `LastPlaybackCheckIn` do not end in `Date`. It is also the only half that sees inside an `any` —
  001's two empty arrays are `[]any`, and an interface has no fields to reflect over.
- **The registry is checked, not trusted.** The model sweep's list of operations is compared against
  the operations the router is actually built with, by walking it. A route added without a model
  fails, which is what stops the split from becoming a hole. The wire half's request list has no
  such check yet — T20 is the check that the router serves exactly the surface file's rows, and
  nothing ties the two lists together. Named in both files.
- **Both halves are proven able to fail**, over models declared in `_test.go` files, which is the
  strong form of "cannot leak into the served surface": the file is in no package the server is
  built from. The casing half additionally fires on a real response — the `profile="CamelCase"`
  body of `/System/Info/Public`, whose seven property names are camelCase *by contract* — which is
  why that profile is deliberately absent from the swept set.
- **The PascalCase rule is measured against the pinned document rather than guessed.** A rule
  spelled "a capital then lower-case letters, repeated" refuses `EnableIPv4`, `Video3DFormat` and
  `Hdr10PlusPresentFlag`, all of which the document contains, and would then be loosened by whoever
  met it. **23 of the 1026 names are not PascalCase** — five are RFC 7807's problem-details members
  and eighteen the plugin repository manifest's, none of them on a v1 route — so
  [behaviours §1.1](../../docs/compatibility/behaviours.md)'s *"every JSON property in PascalCase"*
  is 1003 of 1026 over the document. Recorded here rather than edited into the exported
  measurement; T21 or 010 decides whether it belongs there.

**Two differences from the reference are owed to 010, and neither is in `allowlist.yaml`.**
`SystemArchitecture` (`""` here, `"X64"` there) and `EncoderLocation` (`""` here, `"System"`
there) follow from §6.9's reading of spec §3.2, and an allowlist entry needs either a
`behaviours §N` or one of the four derivation classes — this has neither, so writing one would fail
the file's own load. They are named here so that 010 meets them as *declared and argued* rather than
as a surprise; whichever way that run resolves them, the change is to `behaviours.md` or to
`spec.md` §3.2, not to the handler. The seven paths are **not** in this class:
`request-cases.yaml` already calls them "triage rather than allowlist rows".

**Key order is unmeasured here too, and more weakly than on the public route.** §3.1's order is at
least the order a serialiser walking one type's properties would produce; `SystemInfo` *derives*
from `PublicSystemInfo`, and where a serialiser puts an inherited property relative to a declared
one is a property of that serialiser which no probe in this repository records. The order shipped
is the seven public fields followed by the reference model's own declaration order
`[source: MediaBrowser.Model/System/SystemInfo.cs:29-143 @ v10.11.11]`, marked `⚠️ UNVERIFIED` at
the model. The route is L2, so nothing here asserts it against the reference; one request settles
it, and failing that it is an undeclared difference in 010's run.

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
the standard library. Its `Allow` is wrong — ~~one arbitrary method, measured~~ **two field lines in
map-iteration order, and none at all for a method it does not know; re-measured at T11, §1** — and
correcting a header after the router has decided means reconstructing what the router knew.
Computing it from the table is less code and the table already exists.

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
