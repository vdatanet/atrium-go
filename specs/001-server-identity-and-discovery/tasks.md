---
feature: 001-server-identity-and-discovery
title: Server identity and discovery — tasks
status: Draft
created: 2026-09-02
updated: 2026-09-03
plan_status_required: Accepted
---

# 001 — Tasks

Ordered. Each task is a reviewable change on its own, and states how you know it worked.

No task may say "implement the feature". If one does, it needs breaking down.

**The shape of this list follows the plan's shape.** T1–T14 build the edge — the pipeline every
later feature inherits — and T15–T18 are the four endpoints, which are small because the edge
carries them. That is the roadmap's *"001 carries the edge"* made into an order, and it is why this
list is long for four routes.

## Legend

`[ ]` not started · `[~]` in progress · `[x]` done · `[!]` blocked (say by what)

---

## T1 — A binary that starts, serves nothing, and stops cleanly

- [x] **Changes:** `cmd/atrium` — flags for bind address, data directory and log level, with
  environment fallback; `log/slog` to standard error; an `http.Server` on the bind address; a
  build-stamped version; graceful shutdown on `SIGINT`/`SIGTERM` that drains connections and waits
  for the process group.
- **Amended 2026-09-03, on doing it:** all of that but `main` landed in **`internal/app`**, with the
  version stamp in **`internal/build`**. `cmd/atrium` may hold no branch a test would want to reach
  ([architecture §3](../../docs/architecture.md#3-repository-layout)), and the check below starts a
  server in process and signals it, which needs a package a test can call. `plan.md` §3 carries the
  same amendment.
- **Depends on:** —
- **Verified by:** a test that starts the server on a random port, issues one request, sends the
  shutdown signal and asserts `Shutdown` returns before a deadline with no goroutine left running.
  `go run ./cmd/atrium --help` prints the flags.
- **Spec reference:** §3.5 (the lifecycle the readiness gate hangs on), architecture §5.

## T2 — The installation identity, as a file beside the store

- [x] **Changes:** `internal/system` — read `installation-id` from the data directory; create it
  with `O_EXCL` from 16 cryptographically random bytes rendered as 32 lowercase hex when absent;
  **refuse to start** on a file that is unreadable or not 32 lowercase hex.
- **Amended 2026-09-03, on doing it:** two things the wording did not survive contact with.
  `O_EXCL` on the destination leaves the file existing and empty until the write lands, and a
  concurrent start that reads in that window refuses to boot on an identity it watched being
  written — measured on round five of sixty-four rounds of eight starts. The line is written to a
  temporary file and published with a **hard link**, which refuses an existing name the way
  `O_EXCL` does and never exposes a half-written file; `plan.md` §4 carries the same amendment.
  And the store rebuild below is performed as *"remove everything in the data directory except the
  identity file"* rather than by name: the store's file names are T3's to choose, and a test that
  named one would pass for the wrong reason on the day it changed.
- **Depends on:** T1
- **Verified by:** four tests — a fresh directory produces a 32-hex id; a second start returns the
  same one; **deleting the database and starting again returns the same one** (AC-4's second
  clause); a file containing `nonsense` refuses the start with an error naming the file.
- **Spec reference:** §4, AC-4.

## T3 — The store: a migration runner and the installation row

- [x] **Changes:** `internal/store/sqlite` — open with WAL, `synchronous=NORMAL`, foreign keys on
  and a busy timeout; one writer handle and a reader pool; a forward-only numbered migration runner
  with a schema-version table per half; `0001_installation.sql` creating the single-row table of
  plan §4 with its `CHECK (id = 1)`.
- **Amended 2026-09-03, on doing it:** three decisions the wording left open, all recorded in
  `plan.md` §4. The database is **one file** for both halves, `atrium.db`, with `schema_version`
  holding a row per half. **The entry layer creates the data directory** — T2 left this to T3, and
  the store cannot be where it happens, because the identity file is read first and would fail
  first; it creates the final component only, so a mistyped `--data-dir` is a refusal naming the
  missing parent rather than an empty installation that looks like a fresh one. And the reader pool
  is opened `query_only`, which makes *"one writer handle and a pool of readers"* something the
  engine refuses to break rather than a convention this package asks callers to keep.
  `internal/ports` carries `InstallationStore` with two of plan §5's three methods:
  `MarkSetupComplete` takes a date, dates are ticks, and the tick type is T4's — the column it
  writes exists and the method lands with the type it needs.
- **Depends on:** T1
- **Verified by:** a migration on an empty directory creates the row with `server_name = 'atrium'`;
  a second start applies nothing; an attempted second row is refused by the constraint; the
  precious and derived schema versions are recorded separately.
- **Spec reference:** §4; [ADR-0003](../../docs/decisions/0003-sqlite-as-the-store.md).

## T4 — `internal/units`: the tick and the date

- [x] **Changes:** `internal/units` — a tick type marshalling as a JSON integer, and a date type
  marshalling as seven fractional digits and a `Z`. Parsing accepts anything ISO-8601, with or
  without a timezone; a missing timezone reads as UTC.
- **Amended 2026-09-03, on doing it:** three things, two of them measurements.
  **The date is rounded to a whole tick and held in UTC at construction**, not at write time —
  Go's own formatting truncates, so a value half a tick short of a second would be written as the
  second before it, and a value carrying precision the wire cannot express compares unequal to the
  bytes it serialises as. **A comma as the decimal separator is accepted**: it was written into the
  rejection table and the table disagreed, because ISO-8601 permits it and Go's fractional layout
  element already reads one `[measurement: Go 1.27.0, 2026-09-03]`. And **`IsTime` is one parse and
  no re-format check**: the check was written first, on the assumption that `time.Parse` rolls an
  out-of-range day forward the way `time.Date` does, and removing it made no test fail — Go refuses
  `2025-02-30T00:00:00.0000000Z` itself `[measurement: Go 1.27.0, 2026-09-03]`. A guard no case can
  reach has proved nothing.
  **`MarkSetupComplete` lands here**, as T3 said it would: `internal/ports` gains the third method
  of plan §5 taking a `units.Time`, and the store writes the column as .NET's `DateTime.Ticks`.
  `plan.md` §4 carries that origin, which the migration had left unsaid.
- **Depends on:** —
- **Verified by:** a table test over the round trip, including a value whose fraction is zero
  (`.0000000Z`, not an omitted fraction) and one with sub-tick input that rounds rather than
  truncates.
- **Spec reference:** behaviours §1.2, §1.3. 001 sends neither; plan §3's module table says why they
  are here — the unit sweep of T19 needs a type to recognise, and §8 restates it.

## T5 — `internal/wire`: the encoder and the escape pass

- [x] **Changes:** `internal/wire` — `Write(w, status, v, naming)`; the encoder's HTML escaping
  switched off; one pass applying behaviours §1.16's table, counting backslash parity; the content
  type set by the writer that produced the body, with `charset=utf-8`.
- **Amended 2026-09-03, on doing it:** the pass tracks whether it is inside a string, because
  §1.16 escapes `"` as `\u0022` and a rewrite that treated every quote alike would escape the
  document's own delimiters. And a character above U+FFFF is written as a surrogate pair, which is
  an inference from the reference's stack rather than a measurement and is marked
  `⚠️ UNVERIFIED` — §1.16 was measured on characters that all fit one UTF-16 code unit.
  `plan.md` §6.4 carries both. `NamingCamel` is deliberately **not** declared: T6 declares it with
  the policy behind it, and a constant that names a policy the package has not got would write
  PascalCase in silence. `plan.md` §5 carries that.
- **Depends on:** T4
- **Verified by:** a table test asserting **bytes**: every non-ASCII character and the seven ASCII
  ones as upper-case escapes, and the ten characters left literal. One case is the hard one — a
  string value that genuinely contains the six characters of an escape sequence must survive while
  the encoder's own escapes are rewritten.
- **Spec reference:** §3.0.1, §3.0.3.

## T6 — `wire`: the camelCase naming policy

- [x] **Changes:** two naming policies chosen at write time. The camelCase one lowers a leading run
  of capitals all but the last, applies to **property names at every depth**, and never to
  **dictionary keys**.
- **Amended 2026-09-03, on doing it:** the conversion is a **walk of the encoded document beside
  the value it was encoded from**. `encoding/json` offers no seam where a property name is still a
  property name, and re-implementing the encoder to make one would put tag rules, embedding and
  `omitempty` at the mercy of a second reading — so the bytes decide the structure and the value
  answers only the question they cannot, whether an object's keys came from a struct or from a map.
  `plan.md` §6.3 carries the technique and the two things it forces: **a value that writes its own
  JSON is not renamed** (which is the reference's behaviour as well
  `[source: src/Jellyfin.Extensions/Json/JsonDefaults.cs:34-45,55-58 @ v10.11.11]`), and **a shape
  the walk cannot account for is a refusal rather than a copy**, because half a converted body is
  the same wrong answer as none of it and nobody would see it. An unrecognised `Naming` is an error
  for the same reason; `plan.md` §5 carries that.
- **Depends on:** T5
- **Verified by:** a table test — `Id` → `id`, `UICulture` → `uiCulture`, a nested object's names
  converted, and a map-valued field whose keys come through untouched.
- **Spec reference:** §3.0.2.

## T7 — Content negotiation for the profile

- [x] **Changes:** parse `Accept` into ranges with `q`; match `application/json` and compare
  `profile` case-insensitively and unquoted; a `charset` parameter beside `profile` stops the
  profile match; an unknown profile falls back to plain; rank by `q`, ties keep the client's order.
  The winner sets both the naming policy and the echoed content type, profile before charset.
- **Amended 2026-09-03, on doing it:** *"the winner sets both"* could not be said in the signature
  the winner arrived through. Three declared content types over two naming policies means a body
  written under `NamingPascal` cannot say which of the two PascalCase types asked for it, so
  `Write`'s last argument became a **`Profile`** — the negotiation's whole answer — rather than a
  `Naming`. A middleware stamping the header afterwards was the alternative and is the one
  behaviours §1.10 rules out. `plan.md` §5 carries it.
  Three cases the four rules leave open were decided and written into `plan.md` §6.3: a **wildcard
  range is a candidate that names no profile** (the reference does not discard one before ranking
  `[source: Jellyfin.Server/Extensions/ApiServiceCollectionExtensions.cs:125-126 @ v10.11.11]`);
  a parameter written **after `q`** is an accept-extension and selects nothing (RFC 9110 §12.5.1,
  ⚠️ UNVERIFIED); and *"a charset falls back to the plain type"* has **two readings that no
  measurement separates**, which differ only when a charset-bearing range precedes a bare profile
  range in the same header. The plan's literal reading was taken and the probe is owed.
- **Depends on:** T6
- **Verified by:** a table test over the four rules, plus AC-9's three requests — plain and
  `PascalCase` byte-identical, `CamelCase` the same values under converted names, each response
  naming the one it used.
- **Spec reference:** §3.0.2, AC-9.

## T8 — `internal/surface`: the route table

- [x] **Changes:** `internal/surface` — load `docs/compatibility/surface.yaml`; expose the canonical
  spelling per path, the methods registered on each path, and the owning feature and level. Refuse
  to load a row with an unknown level or a duplicate method-and-path.
- **Amended 2026-09-03, on doing it:** two things the wording left open, both recorded in
  `plan.md` §3. The file **reaches the binary embedded, from a derived copy beside the package**,
  with a test asserting it is the document byte for byte: `go:embed` cannot name a path outside its
  own package directory, and reading the document off disk at run time would make the working
  directory part of the deployment, against ADR-0002's one static binary. And the document is read
  **without a YAML dependency**, because ADR-0002 argues one "in the plan that needs it" and this
  plan names none — the reader is strict where a general parser is lenient, so a key nobody
  consumes, a missing key and an unexpected indent are refusals naming a line rather than a table
  that loads with a hole in it. `Methods` sorts, because that ordering is T11's `Allow` header and
  one place that sorts is one place to be wrong.
- **Depends on:** —
- **Verified by:** the table loads all 59 rows; 001's four are present with their operations; a
  fixture with a duplicate row and one with `level: L9` are both rejected.
- **Spec reference:** §3.6; Principle VI.

## T9 — Path canonicalisation

- [x] **Changes:** `internal/httpapi` — a middleware that folds a request path's **literal**
  segments and rewrites them to the route's own spelling, passes path **parameters** through byte
  for byte, trims one trailing slash, and answers two or more with an empty `404`. No redirect is
  ever issued.
- **Amended 2026-09-03, on doing it:** *"literal segments"* is one word too coarse. Five paths in
  the table put a literal and a parameter **inside one segment** —
  `/Audio/{itemId}/stream.{container}` and `/Videos/{itemId}/hls1/{playlistId}/{segmentId}.{container}`
  among them — so the fold is per run within a segment, not per segment, and `/audio/AbC/STREAM.MP4`
  answers `/Audio/AbC/stream.MP4`. `plan.md` §6.1 carries the same amendment, together with the
  two other rules writing it settled: a literal path is looked up before a parametrised one, and a
  parameter run must match at least one byte.
- **Depends on:** T8
- **Verified by:** a table test over §3.6's table: canonical, any casing, one trailing slash — same
  route and same bytes; two slashes `404`; and a parameter whose casing survives.
- **Spec reference:** §3.6, AC-11; behaviours §1.14.

## T10 — Query key canonicalisation

- [x] **Changes:** the same treatment for query parameter **names** against each route's declared
  spellings. **Values are never touched**, and an unrecognised key is left in place rather than
  dropped.
- **Amended 2026-09-03, on doing it:** *"each route's declared spellings"* had nowhere to live, and
  the decision is recorded rather than left open. `surface.yaml` carries no parameters and 001's own
  four routes take none, so the declarations are **Go, beside the routes** — `QuerySpellings` keyed
  by route, and `V1QuerySpellings()`, which is **empty**. Extending the paired artefact was rejected
  on two grounds: the prose twin, the derived copy and the loader would all move to write an empty
  list on 59 rows, and — the load-bearing half — §1.15 measured that the pinned document spells
  every parameter camelCase while the reference's own clients send PascalCase *and both work*, so
  there is no single spelling a surface file could state. The one this stage needs is the one this
  server's handler binds. `plan.md` §6.2 carries the same amendment, with the two rules writing it
  settled: a declaration is keyed by **route** rather than by path, and a declaration the table has
  no row for is a refusal at construction.
- **Depends on:** T8
- **Verified by:** `Limit`, `limit` and `LIMIT` reach the handler as the declared spelling; a value
  differing only in case is unchanged; an unknown key survives to be counted later.
- **Spec reference:** behaviours §1.15, §1.12.

## T11 — The refusal shapes, and `Allow` computed from the table

- [x] **Changes:** the router's `NotFound` and `MethodNotAllowed` replaced — `404` and `405` with an
  empty body and **no content type** — and `Allow` built from `internal/surface`: every method that
  **path** has, sorted alphabetically. An unauthenticated refusal answers `401` with an empty body,
  `Content-Length: 0` and **no** `WWW-Authenticate`.
- **Amended 2026-09-03, on doing it:** the measurement below is wrong and is corrected in
  `plan.md` §1, which keeps both readings. chi names **both** methods, in **two `Allow` field
  lines**, in **map-iteration order** — `GET, POST` 171 times and `POST, GET` 29 over 200 identical
  requests — and sends **no `Allow` at all** for a method token it does not know
  `[measurement: github.com/go-chi/chi/v5 v5.3.2, Go 1.27.0, 2026-09-03]`. The original reading is
  what a header reader that returns one field line per name sees of two, which is worth more than
  the correction: the same kind of instrument reads the reference. The conclusion is unchanged and
  the assertion below is unchanged, because §1.11 measured one comma-joined field line in
  alphabetical order and none of chi's three faults produces one. Two things the wording left open
  are decided in `plan.md` §6.5: the lookup goes through canonicalisation's `pattern`, because a
  request never carries the pattern that matched it; and a path the table has no row for is a `404`
  whatever the method, because chi checks the method before it routes and §3.6 keys its `404` on
  the path.
- **Depends on:** T8, T9
- **Verified by:** the measured case, which the router gets wrong on its own — `PUT`, `HEAD` and
  `OPTIONS` on `/System/Ping` must each answer `405` with `Allow: GET, POST`, where chi answers one
  arbitrary method
  `[measurement: github.com/go-chi/chi/v5 v5.3.2, Go 1.27.0, 2026-09-02]`. Plus an empty `404` with
  no content type, and the `401` shape.
- **Spec reference:** §3.6, AC-11; behaviours §1.11.

## T12 — `X-Response-Time-ms` and `Server`

- [x] **Changes:** two middlewares — the response time in fractional milliseconds on **every**
  response, and `Server: Atrium/<version>` from the build stamp.
- **Amended 2026-09-03, on doing it.** Three things the wording did not settle.
  **The `503` row of the *Verified by* line below could not be reached**, because the gate that
  answers one is T13's, and it is worth more than a missing row: `plan.md` §6.7 puts that gate
  *outside* these two stages, where a refusal carries neither header, while T14's own acceptance
  asks that the gate's `503` carry both. `plan.md` §6.7 now records the contradiction and the
  reference's own answer to it — its response-time middleware is registered near the outside of the
  main pipeline and its startup gate well inside it
  `[source: Jellyfin.Server/Startup.cs:163,217 @ v10.11.11]` — and leaves the choice to T13/T14.
  What ships here is a test for the constraint and a stand-in row named as one: a `503` written by a
  stage *inside* the stamp carries both headers.
  **The `Date` header T1 left open is answered and owes nothing.** Go's `net/http` sends one on
  every response; so does the reference, measured moving on 19 of 19 read cases
  `[probe: tools/probe_reference_determinism.py, Jellyfin 10.11.11, 2026-09-01]` and excused by name
  in `allowlist.yaml` beside the response time itself. No divergence is owed.
  **The value's shape is the reference's own quantisation, not a format choice.** The reference
  formats a .NET `TimeSpan`'s total milliseconds with the invariant culture
  `[source: Jellyfin.Api/Middleware/ResponseTimeMiddleware.cs:61 @ v10.11.11]`, and a `TimeSpan`
  counts the same 100-nanosecond tick `internal/units` counts (behaviours §1.3) — so at most four
  decimal places, which is what §1.9's measured `2.1329` shows, and the conversion is integer
  arithmetic on `units.Ticks` rather than a format string. That a whole millisecond is sent with no
  decimal part at all follows from .NET's shortest-round-trip formatting and is **⚠️ UNVERIFIED**;
  it is marked in the code, and it costs nothing because the header is excused on every endpoint.
- **Depends on:** T1
- **Verified by:** both headers present on a `200`, on the empty `404`, on the `405` and on the
  `503` — the refusals are where a header added by the wrong layer goes missing.
- **Spec reference:** behaviours §1.9, §4.1.

## T13 — The readiness gate and the `503`

- [x] **Changes:** ~~the outermost stage~~ **the third stage.** Until the server reports itself
  ready, **every** route answers `503` with `Retry-After` in full integer seconds, a `Message`
  header, and a `text/html` body — never JSON. The same response serves a deliberate withdrawal,
  with a different message and a longer hint, without stopping the process.
- **Amended 2026-09-03, on doing it.** Two things, and neither is a detail.
  **The contradiction T12 left is resolved and `plan.md` §6.7 carries the decision**: the gate is
  third, immediately inside the response-time stamp and `Server` and immediately outside anything
  that could route. A middleware that answers without calling the next handler is never reached by
  anything below it, so "outermost" and T14's *"a `503` from the gate still carries the
  response-time stamp and `Server`"* could not both hold. The reference has already taken this way
  out — response-time middleware near the outside of the main pipeline, startup gate well inside it
  `[source: Jellyfin.Server/Startup.cs:163,217 @ v10.11.11]` — the alternative duplicates two
  header values into every stage that refuses, and §3.5 costs nothing either way because nothing
  between the stamp and the gate reads a path. **The plan moved to meet T14's acceptance, not the
  other way round.**
  **§3.5's *"nothing is exempt"* is contradicted by the reference's own source, and it is recorded
  rather than acted on.** The starting `503` comes from a separate setup server with no
  response-time middleware; that setup server answers a real `/System/Info/Public`; and the main
  pipeline's gate exempts `/system/ping` and sends neither header
  `[source: Jellyfin.Server/ServerSetupApp/SetupServer.cs:177-259,204-237 @ v10.11.11]`
  `[source: Jellyfin.Api/Middleware/ServerStartupMessageMiddleware.cs:38-48 @ v10.11.11]`. §3.5 and
  AC-12 cite the **pinned document**, [AGENTS.md §1.3](../../AGENTS.md) says the running server
  wins, and there is no running reference here — so the disagreement cannot be settled and the
  specification, as the authority on WHAT, is implemented as written. `plan.md` §6.8 states what
  discharges it: one probe against a starting reference, failing which 010's differential run.
  **`spec.md` is deliberately unamended.**
  Two values came from the reference rather than from a choice made here: the starting message is
  its own localised string, and the five-second hint is its own
  `[source: Emby.Server.Implementations/Localization/Core/en-US.json:79 @ v10.11.11]`
  `[source: Jellyfin.Server/ServerSetupApp/SetupServer.cs:143 @ v10.11.11]`.
- **Depends on:** T1
- **Verified by:** a server held pre-ready answers `503` on all four routes **and on a path that
  matches no route**; `Retry-After` parses as an integer; the body's content type is `text/html`.
- **Spec reference:** §3.5, AC-12.

## T14 — Assemble the pipeline, and assert its order

- [x] **Changes:** wire the stages in plan §6.7's order and nowhere else.
- **Amended 2026-09-03, on doing it.** Three things the wording did not settle.
  **The order assembled is T13's, not the struck-through one**, and the three assertions hold
  against it: `httpapi.NewPipeline` is the one place the chain exists, and each assertion was run
  against a chain with exactly one stage moved to prove it can fail.
  **The `Accept` join plan §6.3 left to this task is `httpapi.NegotiateProfile(r)`, a function a
  handler calls rather than a stage.** Negotiation writes nothing and refuses nothing, so it is the
  *handler → wire* step §6.7 already ends with, and a `Profile` carried in a request context would
  be `ProfilePlain` in any handler tested without the stage — silently. §6.7's order is unchanged;
  §6.3 carries the argument.
  **Two of the five stages cannot be seen by any request this server serves, and one still cannot.**
  Removing query canonicalisation from the chain broke no test in the repository, because
  `V1QuerySpellings()` is empty — none of 001's four routes takes a query parameter (T10). A test
  that supplies its own declaration now covers it. The one still unasserted is the *relative* order
  of the two canonicalisers: the query folder builds its own path folder internally, so swapping
  them changes no response. And the `404` row of the *Verified by* line does not, on its own, prove
  canonicalisation ran — the router answers `404` with both headers too. It is the fold reaching a
  handler (`GET /system/ping/` → `200`) that proves it, and that row is asserted beside it.
- **Depends on:** T9, T10, T11, T12, T13
- **Verified by:** three assertions that only the order can satisfy — a `503` from the gate still
  carries the response-time stamp and `Server`; a `404` from canonicalisation carries them too; and
  the gate answers before routing, so an unknown path answers `503` and not `404` while starting.
- **Spec reference:** plan §6.7; architecture §4.

## T15 — `LocalAddress`, as a pure function over `RequestFacts`

- [x] **Changes:** `internal/system` — the three tiers of §3.4 over the domain's own
  `RequestFacts`. Tier 1 accepts the **subnet-scoped** published-URL form and picks the most
  specific matching prefix; tier 2 omits the port when it is the scheme's default; tier 3 returns
  the scheme and port the server is actually reachable on.
- **Amended 2026-09-03, on doing it.** Three things the wording did not settle, all recorded in
  `plan.md` §6.6.
  **The trailing slash is ~~one~~ every one, at both ends.** The reference spells it
  `PublishedServerUrl.Trim('/')`
  `[source: Emby.Server.Implementations/ApplicationHost.cs:877 @ v10.11.11]`, which is what §3.4's
  *"any trailing `/` removed"* says and what plan §6.6's *"one"* did not. A published URL
  configured with two trailing slashes would have come back carrying one.
  **The order of tiers 1 and 2 contradicts the reference's own source, and the spec was
  implemented as written.** `LocalAddress` is served by the `HttpRequest` overload
  `[source: Emby.Server.Implementations/SystemManager.cs:77, 120 @ v10.11.11]`, which tests
  `EnablePublishedServerUriByRequest` **before** the published URL
  `[source: Emby.Server.Implementations/ApplicationHost.cs:885-901 @ v10.11.11]`. The two readings
  differ on exactly one installation — both configured — and there is no running reference here to
  settle it (AGENTS.md §1.3), so it is recorded as one request owed rather than closed by editing
  the spec. Same shape as §3.5 at T13.
  **The certificate had to become an input for the divergence to be assertable.** §4.2 argues that
  v1 cannot configure a certificate at all, which would leave the *Verified by* line's last clause
  with nothing to set. `AddressConfig` carries `CertificateConfigured` and `HTTPSPort`, no branch
  reads either, and the check sets them and asserts the answer does not move **on every tier** —
  a divergence with no input can only be assumed.
- **Depends on:** T3
- **Verified by:** a table test with synthesised requester addresses covering AC-7, AC-8 and AC-13,
  and a case where a certificate is configured and tier 3 still answers the reachable scheme — the
  deliberate divergence, asserted rather than assumed.
- **Spec reference:** §3.4, AC-7, AC-8, AC-13; behaviours §2.3, §4.2.

## T16 — `GET /System/Info/Public`

- [x] **Changes:** the handler and its response model — exactly the seven fields of §3.1, with
  `ProductName` the literal `Jellyfin Server`, `OperatingSystem` the empty string, and `Version` the
  pinned reference version.
- **Amended 2026-09-03, on doing it:** four things this wording did not settle, and the first is the
  one that decides how every golden in this repository works.
  **A byte-compared golden needs the response to stop deriving from the run.** Two of the seven
  fields do: `Id` is 16 cryptographically random bytes on a first start, and `LocalAddress` is built
  from the request, whose port the operating system chose. Neither can be normalised away without
  giving up the byte comparison, so both are **stated** instead — the test writes the identity file
  before the server starts and sends a fixed `Host` header, and the recorded body is the one spec
  §3.1 prints. The per-field assertions run on a *genuinely* fresh installation, where `Id` is
  asserted by shape.
  **`conformance/` cannot import `internal/`, so it cannot build a server — it starts the binary.**
  `go build` once in `TestMain`, then one process per test on `127.0.0.1:0`, with the bound address
  read back out of the server's own log because a caller that did not choose the port has nowhere
  else to learn it. That is the shape T19 and T20 inherit, and it is the import rule
  ([architecture §3](../../docs/architecture.md#3-repository-layout)) doing its job rather than
  getting in the way: everything these tests know is something a client could have known.
  **The update flag rewrites the golden and then fails.**
  [architecture §8](../../docs/architecture.md#8-testing-and-conformance) says goldens are reviewed
  and never blindly regenerated, so no single run may both rewrite one and report green —
  `-update-golden` writes the file and fails the test, and the suite has to be re-run without it.
  An update flag that left the run green is a button that turns a golden into a record of whatever
  the code last did.
  **Key order is contract and it is not measured.** L3 compares bytes, so the order of the seven
  keys is part of the response. The reference's model declares them in exactly §3.1's order
  `[source: MediaBrowser.Model/System/PublicSystemInfo.cs:14-53 @ v10.11.11]` and that is what is
  implemented — but no probe in this repository records the key order of a body, and the two sample
  bodies here **disagree** (§3.1 opens with `LocalAddress`, `reference-target.md` §4 with
  `ServerName`). It is one request to settle, marked `⚠️ UNVERIFIED` in the model, and until it is
  settled it is a difference 010's run would raise.
- **Depends on:** T7, T14, T15
- **Verified by:** a byte-compared golden on an empty installation, plus per-field assertions so a
  golden diff names which field moved. Answers before any user exists and before any library is
  configured.
- **Spec reference:** §3.1, AC-1, AC-2, AC-3.

## T17 — `GET /System/Ping` and `POST /System/Ping`

- [x] **Changes:** both methods returning the bare JSON string `"Jellyfin Server"` — the **product
  name**, not the operator's friendly name.
- **Amended 2026-09-03, on doing it:** two things this wording did not settle, and the second is
  the one a later task has to finish.
  **A test that sends two methods does not prove it sent two methods.** Every assertion here runs
  over `GET` and `POST`, and a harness that ignored the method it was handed and issued `GET` every
  time passed all of them — measured, by making it do exactly that: the whole `conformance` package
  stayed green. Nothing that answers `200` to both methods can tell them apart. What tells them
  apart is a request that must answer *differently* by method, so `PUT /System/Ping` → `405` is
  asserted at the wire beside them, and it fails when the harness is broken that way. T11 owns the
  `Allow` computation and is not re-run; this row records only that the value a client receives did
  not move when two real handlers arrived on one path.
  **The fixture cannot set a friendly name here, and the *Verified by* line above asks it to.**
  001 gives an operator no way to rename an installation — `SetServerName` exists on the port and
  nothing calls it, because the rename endpoint belongs to 002 — so the only friendly name this
  binary can be started with is plan §4's default, `atrium`. The discrimination is therefore proven
  where a fixture *can* choose one: the handler-level test sets a deliberately unlike name through
  the store, and the conformance test reads `ServerName` off the same running server, refuses to
  proceed if it has become the product name, and then asserts the ping bytes. Both guards were run
  against a mutation that makes the two names equal, and both fire. **→ 002: when the rename
  endpoint lands, send it here and drop the caveat.**
- **Depends on:** T14
- **Verified by:** exact bytes on both methods. A test that would pass if the handler returned
  `ServerName` is not a test of this; the fixture sets a friendly name that differs.
- **Spec reference:** §3.3, AC-6.

## T18 — `GET /System/Info`

- [x] **Changes:** the authenticated superset of §3.2, with the plan's stated values for the flags
  and paths. Token validation is taken through a port that **002 fills**; until then the only
  reachable states are *setup incomplete*, which the reference permits without a credential, and
  *any token*, which is invalid.
- **Amended 2026-09-03, on doing it:** four things the wording did not survive contact with.
  **The plan stated no paths** — it says nothing about `ProgramDataPath` and its six neighbours —
  so the layout was decided here and `plan.md` §6.9 now carries it, derived from the one path an
  operator configures. **`PackageName` is not sent at all**: §3.0.3 defers to a per-field
  verification where one exists, behaviours §1.7 is that verification, and `spec.md` §3.2 carries
  the amendment. **The superset is structural** — the public model is embedded in the
  authenticated one and filled in by the same function, so the seven shared values are one value
  each rather than two that agree. And **`WebSocketPortNumber` forced a change to the entry
  layer**: the handler is built before the listener exists, so the port arrives as a function the
  entry layer fills in after it binds, which is the only way `--bind-address :0` answers anything
  but a lie.
- **Depends on:** T16
- **Verified by:** the superset assertion — every field shared with `/System/Info/Public` agrees —
  and the `401` shape without a token once setup is complete.
- **Amended 2026-09-03, on doing it:** the two halves are proven at two levels, and they have to
  be. The superset is asserted over the wire against the running binary
  (`conformance/system_info_test.go`), member by member as raw JSON. The `401` is **not** reachable
  there: it needs an installation whose setup is complete, and 001 serves no route that completes
  setup — 002 owns it. So it is asserted at the HTTP boundary in `internal/httpapi`, over a real
  connection rather than a recorder, because three of the four things behaviours §1.11 measures
  about that shape are invisible to a recorder (T11's finding). **There is no golden for this
  route**, and that is a consequence of the response rather than a shortcut: seven of its fields
  are the installation's own paths and one is the port the operating system chose. `plan.md` §8 and
  `spec.md` §6 both carry the amendment.
- **Spec reference:** §3.2, AC-5.
- **Carried:** AC-5's *"and `200` with a valid one"* half cannot be proven here, because no
  credential exists until 002. It is recorded as a **criterion carried into 002** rather than
  marked met, and 002's task list closes it. See T21.

## T19 — The two cross-cutting sweeps

- [x] **Changes:** ~~`conformance`~~ **`conformance` *and* `internal/httpapi`, split — see the
  amendment** — the casing sweep over every registered response type, and the
  unit sweep. The unit sweep recognises **a date by its value, not by its name**, since
  `DateCreated` does not end in `Date`.
- **Depends on:** T16, T17, T18
- **Verified by:** both sweeps pass; and each one is proven to be able to fail — a deliberately
  camelCase field and a deliberately three-digit date, in test-only models, must each be caught.
  **A sweep that has never failed has proved nothing.**
- **Spec reference:** §6; [conformance L1](../../docs/compatibility/conformance.md#l1--shape).
- **Amended 2026-09-03, at T19. The sweeps are split by what each half can see**, because
  `conformance/` may not import `internal/` and a reflection sweep over Go types therefore cannot
  live there. The **model sweep** is in `internal/httpapi` and walks `reflect.Type`; the **wire
  sweep** is in `conformance/` and walks response bytes, which is where the "by value" date rule
  belongs. `docs/architecture.md` §8 carries the amendment for the placement it stated, and
  `plan.md` §8 the reasoning. The registry the model sweep walks is checked against the operations
  the router is really built with, so the split cannot silently leave a model unswept; the wire
  half's request list has no such check until T20, and that is said in the file.
- **Carried:** nothing. Both halves fail on demand — see the mutation runs in this task's pull
  request.

## T20 — The L0 registration check

- [ ] **Changes:** `conformance` — assert the router exposes **exactly** the `surface.yaml` rows
  whose owning feature is implemented, and nothing outside `surface.yaml` at all.
- **Depends on:** T8, T16, T17, T18
- **Verified by:** the four 001 rows are served; a route registered without a row fails the test;
  a row whose feature is implemented and which is not served fails it too.
- **Spec reference:** §3.6, AC-11; Principle VI.

## T21 — The closing audit

- [ ] **Changes:** whatever this task finds. It is not a formality: every implemented feature in the
  exporting project found, in its own final task, **an acceptance criterion with no test or a test
  proving less than its name**.
- **Depends on:** all of the above
- **Verified by:** four passes, each recorded with what it found or that it found nothing —
  (a) every one of the thirteen acceptance criteria mapped to a named test that fails when the
  behaviour is broken, not merely when the code is absent; (b) every paragraph of spec §3 either
  tested or listed as untested with a reason; (c) AC-5's carried half recorded in 002's spec rather
  than left implicit; (d) anything implementation taught written back into `spec.md` **in this same
  change**, and any newly measured reference behaviour into `behaviours.md` with provenance.
- **Spec reference:** all of §5; AGENTS.md §5.

---

## Definition of done

The feature is done when **all** of these hold:

- [ ] Every acceptance criterion in `spec.md` §5 has a passing test — **except AC-5's second half,
  which T18 records as carried into 002 with the reason, rather than as met.**
- [ ] Every endpoint reaches the conformance level declared in `spec.md` §6. `/System/Info/Public`
  reaches **L2**; its L3 half is deferred on the spec's own terms and closes the first time 010 runs.
- [ ] `docs/compatibility/surface.yaml` lists every route added, and no route exists outside it.
- [ ] Anything learned during implementation is back in `spec.md`, in this same change.
- [ ] Any new measured Jellyfin behaviour is in `docs/compatibility/behaviours.md` with provenance.
- [ ] `spec.md`, `plan.md` and `tasks.md` are all marked `Implemented`.
