---
feature: 001-server-identity-and-discovery
title: Server identity and discovery — tasks
status: Implemented
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
  against a mutation that makes the two names equal, and both fire. ~~**→ 002: when the rename
  endpoint lands, send it here and drop the caveat.**~~ **Struck in place 2026-09-03, at 002's
  closing audit, because the condition can never be met.** The reference renames a server at
  `POST /Startup/Configuration`
  `[source: Jellyfin.Api/Controllers/StartupController.cs:74-78 @ v10.11.11]`, which is not one of
  `surface.yaml`'s fifty-nine rows — so *"the rename endpoint"* is not 002's and is not any v1
  feature's. What can discharge the caveat is the friendly name becoming **operator
  configuration**, over the `SetServerName` port this row already names as having no caller; 002
  deliberately did not add one, because it is 001's datum and not that feature's decision to take.
  [002 §5](../002-authentication-users-and-sessions/spec.md#5-acceptance-criteria) carries the
  correction in the form the note should have had. This record is struck rather than rewritten
  ([AGENTS.md §4](../../AGENTS.md)): what 001 believed is the useful half.
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

- [x] **Changes:** ~~`conformance`~~ **`conformance` *and* `internal/httpapi`, split — see the
  amendment** — assert the router exposes **exactly** the `surface.yaml` rows
  whose owning feature is implemented, and nothing outside `surface.yaml` at all.
- **Amended 2026-09-03, on doing it:** three things, and the first is the one the wording left
  entirely open.
  **"Implemented" is derived, not listed.** A list of implemented features written into either
  half would be right until 002 lands and then quietly wrong — and a stale list makes the check
  silent about exactly the rows it has stopped knowing about. Both halves apply one rule instead:
  **a feature the server serves any row of must serve every row of it.** A feature with no row
  served is one this build does not implement, which is a reading of the server rather than a
  claim about the roadmap. Today that rule reads `001` off the wire and off the router, and it
  will read `002` the day 002 lands without anybody editing a test. Its blind spot — a feature
  implemented and serving *none* of its rows — is named in `plan.md` §8.5 rather than left
  implicit.
  **The check is two checks in two directories**, because `conformance/` may not import
  `internal/surface` (T8's note, [architecture §3](../../docs/architecture.md#3-repository-layout)).
  Registration walks `Pipeline.Router()` from `internal/httpapi`; reachability issues a real
  request per row from `conformance/`, reading the row list out of
  `docs/compatibility/surface.yaml` with a strict reader of its own rather than a generated copy
  that would go stale silently. `plan.md` §8.5 and `docs/architecture.md` §8 both carry the
  reasoning.
  **The registration half had a hole of its own, and it is now closed by reflection.** `Routes`
  registers whatever the `Handlers` value it is given contains, and the test builds that value —
  so a feature adding a field and filling it in `cmd/atrium`'s wiring would register routes this
  check never walked. Every field of `Handlers` is therefore required to be set, which makes
  adding one fail the check until the check is taught about it.
- **Depends on:** T8, T16, T17, T18
- **Verified by:** the four 001 rows are served; a route registered without a row fails the test;
  a row whose feature is implemented and which is not served fails it too.
- **Also closed here:** T19's open gap. The wire sweep's request list was hand-written and tied to
  nothing; every row the running server answers must now appear in it, so a route added and not
  swept fails `conformance/`. `docs/architecture.md` §8 and `conformance/sweep_test.go` both said
  the gap was open and now say what closed it.
- **Spec reference:** §3.6, AC-11; Principle VI.

## T21 — The closing audit

- [x] **Changes:** whatever this task finds. It is not a formality: every implemented feature in the
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

### The record — 2026-09-03

**It found two, and they are the same finding twice.** AGENTS.md §5 says to budget for a criterion
with no test or a test proving less than its name. AC-9 and AC-11 each had a thorough, correct test
suite proving the *mechanism* and nothing proving the *criterion*, and in both cases the gap was
invisible to reading and obvious to a mutation. Both are closed in this change; the fixes are two
files in `conformance/` and no change to any behaviour.

#### Pass (a) — thirteen criteria, thirty-two mutations, two findings

Every row was verified by breaking the behaviour in the production code and watching the named
tests fail. **A mutation that merely deletes the code is not on this list**, because a test that
fails only when a function is missing is a test of the build.

| AC | The tests that carry it | Mutation, and what fell over |
|---|---|---|
| 1 | `conformance`: `TestPublicSystemInfoCarriesSevenFieldsInSpecOrder`, `TestPublicSystemInfoMatchesItsGolden`; `internal/httpapi`: `TestPublicInfoAnswersTheSevenFieldsOfSpecThreeOne` | An eighth field on the model → **8 tests**. Note the field-by-field test is **not** among them: it asserts seven values and never counts, which is why the count test exists beside it |
| 2 | `conformance`: `TestPublicSystemInfoAnswersFieldByFieldOnAnEmptyInstallation`, `TestPublicSystemInfoMatchesItsGolden`; `internal/httpapi`: `TestPublicInfoIdentifiesAsJellyfinAndNotAsAtrium` | `ProductName` → `"Jellyfin server"` (one letter) → **13**; `OperatingSystem` → `"Linux"` → **6** |
| 3 | the same three | `ReportedVersion` → `"10.11.10"` → **5** |
| 4 | `internal/system`: `TestInstallationIDIsTheSameOnASecondStart`, `TestInstallationIDSurvivesARebuildOfTheStore`, `TestInstallationIDIsCreatedOnAFreshDataDirectory`, `TestConcurrentStartsAgreeOnOneInstallationID`; `internal/app`: `TestRunReportsTheInstallationIDItStartedWith` | Regenerate on every start, keeping the file → **5**; generate the same value upper-case → **6** |
| 5 (`401`) | `internal/httpapi`: `TestSystemInfoRefusesWithoutACredentialOnceSetupIsComplete` | Never consult the admission port → **3** |
| 5 (superset) | `conformance`: `TestSystemInfoIsASupersetOfThePublicBody`; `internal/httpapi`: `TestSystemInfoAgreesWithThePublicBodyOnEverySharedField` | One shared field altered between the two bodies → **4** |
| 5 (`200` with a valid token) | **carried** — [002 AC-14](../002-authentication-users-and-sessions/spec.md#5-acceptance-criteria) | — see pass (c) |
| 6 | `conformance`: `TestPingMatchesItsGoldenOnBothMethods`, `TestAMethodPingDoesNotHaveIsStillRefused`; `internal/httpapi`: `TestPingAnswersTheProductNameOnBothMethods` | `202` instead of `200` → **3**; drop the `POST` registration → **10** |
| 7 | `internal/system`: `TestLocalAddressChoosesByTierAndByRequester`; `internal/httpapi`: `TestPublicInfoReportsTheConfiguredPublishedURL` | Keep the trailing slash → **2**; never consult tier 1 → **2** |
| 8 | `internal/system`: `TestTwoNetworksReceiveTwoDifferentAddresses`; `internal/httpapi`: `TestPublicInfoAnswersEachNetworkWithItsOwnAddress` | Tier 3 answers every requester the first bound address → **3** |
| 9 | `internal/wire`: `TestTheThreeDeclaredContentTypesAnswerAsTheReferenceDoes`, `TestNegotiateAnswersEveryProfileWithItsOwnContentType`, `TestNegotiateFollowsTheFourRules`; **new** in `conformance`: `TestThePlainTypeAndThePascalCaseProfileAnswerOneBodyUnderTwoContentTypes`, `TestTheCamelCaseProfileAnswersSpecThreeOnesValuesUnderConvertedNames`, `TestTheCamelCaseProfileConvertsTheSupersetsNamesAndNotItsValues` | **Finding F-1, below.** Three mutations after the fix: the PascalCase profile converting → **2**; the CamelCase profile not converting → **8**; the echo removed → **6**; and each `/System/Info*` handler ignoring the negotiation → **4** and **3** |
| 10 | `internal/httpapi`: `TestEveryResponseFieldNameIsPascalCase`; `conformance`: `TestEveryResponseSweepsClean` | A `json:"encoderVersion"` tag on a served model → **4**, one from each half of the sweep |
| 11 | `internal/httpapi`: `TestEveryAcceptedSpellingReachesTheSameRouteWithTheSameBytes`, `TestTwoOrMoreTrailingSlashesAreRefused`, `TestNoRedirectIsEverIssued`, `TestAnUnroutablePathIsAnEmpty404`, `TestTheMeasuredCaseChiGetsWrong`; `conformance`: `TestAMethodPingDoesNotHaveIsStillRefused`, `TestTheServerIsReachableOnExactlyTheImplementedRowsOfTheSurfaceDocument`; **new** in `conformance`: `TestEveryAcceptedSpellingOfAServedRouteAnswersTheSameBytes`, `TestATrailingSlashIsNotAnsweredWithARedirect`, `TestTwoTrailingSlashesOnAServedRouteAreRefused` | **Finding F-2, below.** Also: `Allow` naming only the first method → **7**; no `Allow` at all → **5**; a `Content-Type` on a refusal → **17**; two trailing slashes folded away → **5** |
| 12 | `internal/httpapi`: `TestEveryRouteAndEveryNonRouteAnswers503WhileStarting`, `TestTheGateAnswersBeforeTheStampSoA503CarriesBothHeaders`, `TestAHintIsRoundedUpToAWholeSecond`, `TestTheStartingBodyIsTheReferencesOwnMessage` | Exempt the liveness probe → **6**; JSON instead of `text/html` → **6**; a fractional `Retry-After` → **8**; no `Message` → **8** |
| 13 | `internal/system`: `TestLocalAddressChoosesByTierAndByRequester` | Never consult tier 2 → **1**; keep the scheme's default port → **1** |

**F-1 — AC-9 was proven about the serialiser and not about any endpoint.** The criterion says
requests to this feature's endpoints receive byte-identical bodies under two of the three content
types, camelCase names under the third, and an echo. `internal/wire` proves all of it — over a model
declared in a `_test.go` file, with a recorder. Making `/System/Info/Public` write with a constant
`wire.ProfilePlain` instead of the negotiated profile **left every test in `conformance/` green
except `TestTheCasingSweepFiresOnTheCamelCaseProfile`**, which is the casing sweep's own failure
proof, is named for AC-10's machinery, and asserts a *count of findings* rather than anything AC-9
says. Nothing at the boundary compared the two PascalCase answers with each other, and nothing
checked the echo on either `/System/Info` route. `conformance/profiles_test.go` is the fix, and the
same mutation now fails four tests.

**F-2 — AC-11's "the same bytes" was proven about an echo handler.** `internal/httpapi`'s spelling
tests run against a router whose handler writes back the route it was reached through, so the bytes
compared are the route's name; every assertion of that shape holds equally on a server whose real
handlers are wired to the wrong rows. A pipeline whose path folder recognises the doubled slash and
then **folds nothing** left the entire `conformance/` package green. `conformance/spellings_test.go`
is the fix — the four served rows in every accepted spelling, compared as responses — and the same
mutation now fails two of its tests.

**One instrument fault, found writing F-2's fix and worth carrying.** Go's `http.Client` follows a
`301` on its own, so a server answering every accepted spelling with a redirect to the canonical one
**passes a byte comparison of the followed response**. §3.6's *"Not a redirect"* needs a client that
does not follow, and it is a separate test for that reason. Measured: a fold stage rewritten to
issue `301` fails `TestATrailingSlashIsNotAnsweredWithARedirect` and passes
`TestEveryAcceptedSpellingOfAServedRouteAnswersTheSameBytes`.
`[measurement: mutation of internal/httpapi.PathFolder.Wrap, Go 1.27.0, 2026-09-03]`

**Three criteria are asserted below the HTTP boundary, each for a reason that is a property of
001.** They are named in `spec.md` §6 rather than only here. AC-7, AC-8 and AC-13 have no
configuration surface to reach from a client; AC-12 cannot be seen from a client at all, because
the binary opens its gate before it serves and 001 routes nothing that withdraws it; AC-5's `401`
needs a completed setup 001 cannot perform. The first closes with the first feature that adds
configuration, the third with 002.

#### Pass (b) — every paragraph of spec §3

Tested, unless the row says otherwise. *"Tested"* here means at least one named test fails when the
paragraph's behaviour is broken.

| §3 paragraph | Status |
|---|---|
| 3.0.1 PascalCase property names | AC-10, both halves of the sweep |
| 3.0.2 three names, two behaviours; the echo | AC-9, now at the boundary too |
| 3.0.2 lenient match; a charset beside the profile stops it; an unknown profile falls back; ranking keeps the client's order at equal quality | `TestNegotiateFollowsTheFourRules`. **The charset rule's *ranking* half is a reading, not a measurement** — registered as U-6 |
| 3.0.2 dictionary keys are never converted | `TestWriteConvertsPropertyNamesAndLeavesDictionaryKeys`, `TestADeclaredDictionarysKeysAreNotSweptAsPropertyNames` |
| 3.0.2 the conversion is .NET's, not "lower the first letter" | `TestCamelNameIsTheReferencePolicy`, `TestTheTwoRulesDisagreeOnExactlyOneName` — over the 1026 names of the pinned document |
| 3.0.3 absent optional values, verified per field | `TestSystemInfoDoesNotSendPackageName`, and the key-order tests, which is where an added `PackageName` would show |
| 3.0.4 identifiers are 32 lowercase hex | AC-4 |
| 3.1 request: no authentication, answers before any user or library | `TestPublicSystemInfoAnswersFieldByFieldOnAnEmptyInstallation`, on an installation the harness *asserts* was empty rather than assuming it |
| 3.1 the seven fields, their values and their order | AC-1, AC-2, AC-3. **Their order is not measured against the reference** — U-3 |
| 3.1 note: `Id` survives restarts and database rebuilds | Tested in `internal/system`. **Untested at the wire**: the harness runs one process per test and has no restart primitive. What the wire does prove is that the response reports what is on disk, because the golden's identity is written before the start |
| 3.1 errors: `503` while starting | AC-12 |
| 3.2 request: authenticated, and permitted during first-time setup | `TestSystemInfoIsServedDuringFirstTimeSetup`, and eight handler tests. **Mutation 8 of T18 is the thing to know**: setup outstanding is the only state a v1 installation can be in, so dropping the exemption fails twelve tests and the `401` tests are the only ones exercising the other branch |
| 3.2 the superset and its five value rows | AC-5, `TestSystemInfoAnswersTheFlagsAndArraysOfSpecThreeTwo`, `TestSystemInfoReportsThisInstallationsPaths`, `TestSystemInfoReportsThePortItIsListeningOn` |
| 3.2 OQ-1 | An open question, not a behaviour. Nothing to test |
| 3.2 errors: `401` | Tested at the handler over a real connection; moves to `conformance/` with 002 |
| 3.2 errors: `403` | **Untested, and unreachable.** 001 issues no credential, so no request can be valid *and* insufficient; the admission port declares two values on purpose and an unknown third is a loud `500`. §3.2 carries the amendment; 002 owns it |
| 3.3 both methods, a bare JSON string | AC-6 |
| 3.3 note: the product name, not the friendly name — "the code is the specification" | `TestPingIgnoresTheFriendlyNameTheSameHandlerReports` at the handler; at the wire the guard refuses to proceed if the two are equal. **The wire half is half-proven** — 001 has no route that renames a server — and ~~it is carried into 002 with the assertion to complete~~ **it stays 001's, struck 2026-09-03 at 002's closing audit**: no v1 row renames a server `[source: Jellyfin.Api/Controllers/StartupController.cs:74-78 @ v10.11.11]`, 002 deliberately did not add a configuration surface for the friendly name, and the caveat drops when a feature gives an operator one — over the `SetServerName` port, which still has no production caller |
| 3.4 the three tiers | Tested over all three, in `internal/system` and at the handler. **Untested at the wire, and unreachable there**: 001 ships no configuration surface. **The order of tiers 1 and 2 contradicts the reference's source** — U-2, and §3.4 carries the amendment |
| 3.4 the deliberate divergence (no HTTPS override) | `TestTierThreeIgnoresAConfiguredCertificate`, `TestACertificateChangesNothingOnAnyTier`. T15's mutation 12 — reintroducing the override — fails exactly those two and nothing else, which is what makes it asserted rather than assumed |
| 3.5 the whole section | AC-12, against the assembled pipeline. **Untested at the wire, and unreachable there.** *"Nothing is exempt"* is contradicted by three source readings — U-1 — and `Retry-After`'s padding and the body's bytes ride with it as U-4 and U-5. §3.5 carries the amendment and is **deliberately not corrected on source evidence** |
| 3.6 the six-row table | AC-11, now at the wire as well |
| 3.6 `Allow` names every method the *path* has | `TestAllowIsAPropertyOfThePathNotOfTheRoute`, `TestTheThreeMethodPathsAdvertiseAllThree`, `TestAMethodPingDoesNotHaveIsStillRefused` |
| 3.6 "Nothing is automatic" — `HEAD` and `OPTIONS` are `405` | `TestTheMeasuredCaseChiGetsWrong`, over a real connection, on six method tokens. **At the wire only `PUT` is sent**, and that request is there to prove the harness honours the method it is handed rather than to cover the token set |
| 3.6 path parameters are values, not spellings | `TestAPathParameterReachesTheHandlerByteForByte`, over stand-in routes. **Not testable at the wire in this feature: no route 001 serves has a parameter.** 002 is the first that can |
| 3.6 an unknown method token on an unrouted path | Answered `404`. **A reading of §3.6, not a measurement** — U-9 |

#### Pass (c) — AC-5's carried half

Written into 002 as **AC-14**, with the two conditions that discharge it: 001's authentication port
filled, and the request issued to the running binary carrying a token 002's own route returned. Two
smaller carried notes ride with it in the same amendment — the `401` assertion that moves to
`conformance/` when 002 can finish setup over HTTP, and the `/System/Ping` friendly-name
discrimination that completes ~~when a rename endpoint exists~~ **when an operator can name a
server**. 002's front matter records the amendment. *A criterion carried in a sentence is one nobody
closes.*

**Both conditions were struck in place on 2026-09-03, at 002's closing audit, because neither could
be met as written** — 002 completes setup and does not do it over HTTP, and no v1 row renames a
server `[source: Jellyfin.Api/Controllers/StartupController.cs:74-78 @ v10.11.11]`. The first is
discharged: 002 §3.9 makes an installation set up at its first account, so an installation in that
state can be stood up and the `401` assertion moved to `conformance/` with AC-14. The second is not,
and it is 001's to take whenever a feature gives an operator a way to set the friendly name. *A
condition no feature can satisfy is a debt that outlives the project, and it reads exactly like one
that is merely waiting.*

#### Pass (d) — what went back into the documents

- **`spec.md`** — §3.2's `403` is unreachable here and says so; §3.4 records that the reference's
  source orders tiers 1 and 2 the other way and that this document is **deliberately not amended on
  source evidence**; §3.5 records the same for *"nothing is exempt"*, with all three source
  readings; §5 records where AC-5's carried half now lives and that AC-9 and AC-11 were proven a
  level too low; §6 gains a row for the three content types and names the three behaviours asserted
  below the HTTP boundary.
- **`behaviours.md`** — **§4.5 is new**: the four value differences on `/System/Info`, with the
  reference's values, their obsolete markers and the argument. It exists because an
  `allowlist.yaml` entry must cite a `behaviours §N` or a derivation class, these have neither, and
  the file's own load would have refused an entry — so 010 would have met four **undeclared**
  differences on the second response of the API. §1.13's *"1043 names"* is corrected to **1026**:
  the pin moved on 2026-09-01 and this sentence was not among the things that recount recounted;
  the conclusion is unaffected. §1.1's *"every JSON property in PascalCase"* is amended to **1003
  of 1026**, with the eighteen manifest members and the five RFC 7807 ones, none on a v1 route, and
  with the rule-shape finding that matters more than the count.
- **`reference-target.md`** — a new register beside the prior-measurement one: **twelve claims this
  project asserts and has never measured**, U-1 to U-12, each with what is unmeasured, what this
  server does today, where it was recorded and whether one request settles it. Four tasks wrote
  *"this belongs in the register"* and nobody owned the document. It also carries T11's warning for
  010, which is about the **instrument** rather than the reference: a header reader that returns one
  field line per name cannot see a repeated header, and this project's own chi measurement was wrong
  in exactly that way before it was re-measured.
- **`docs/architecture.md` §2** — the layer table said a port imports *"nothing of ours"* and its own
  prose three sentences later said the unit types are *"a leaf, imported by both"*. T4 implemented
  the prose and flagged that an ADR might be the remedy. The audit agrees with the prose: the table
  is corrected in place, struck rather than replaced, and **no ADR is needed** — this is the
  document stating what it already meant, not a deviation from it.
- **Not changed, deliberately:** `allowlist.yaml`. It is one third of a three-way pairing compared
  row for row against `conformance.md` L3 and `010 spec §3.3`, and writing one third of a triple is
  how a paired set drifts. §4.5 is what those rows will cite; 010 writes them.

---

## Definition of done

The feature is done when **all** of these hold:

- [x] Every acceptance criterion in `spec.md` §5 has a passing test — **except AC-5's second half,
  which T18 records as carried into 002 with the reason, rather than as met.** Closed at T21 with
  the exception intact and now numbered: it is [002 AC-14](../002-authentication-users-and-sessions/spec.md#5-acceptance-criteria).
  Every other criterion is mapped to named tests in T21's pass (a), and each mapping is verified by
  **breaking the behaviour** rather than by reading — thirty-two mutations, two of which found a
  criterion proven a level below the one it is written at.
- [x] Every endpoint reaches the conformance level declared in `spec.md` §6. `/System/Info/Public`
  reaches **L2**; its L3 half is deferred on the spec's own terms and closes the first time 010 runs.
  **Three of §3's behaviours are asserted below the HTTP boundary** — §3.4's tiers, §3.5 and §3.2's
  `401` — each because of something 001 does not have rather than something it skipped, each named
  in `spec.md` §6 and in T21's pass (b), and each closing in a named later feature.
- [x] `docs/compatibility/surface.yaml` lists every route added, and no route exists outside it.
  Proven twice over and by two instruments that cannot agree by construction: a `chi.Walk` of the
  router the binary is built with, and one real request per row against the running binary (T20).
- [x] Anything learned during implementation is back in `spec.md`, in this same change — §3.2, §3.4,
  §3.5, §5 and §6, listed in T21's pass (d). **Two of those amendments record a contradiction and
  decline to resolve it**, which is the correct move under AGENTS.md §1.3 and is written down as one
  taken rather than left to look like an oversight.
- [x] Any new measured Jellyfin behaviour is in `docs/compatibility/behaviours.md` with provenance —
  §4.5 is new, §1.1 and §1.13 are amended. **And the honest half of this line: twelve claims this
  project asserts and has never measured are now enumerated as U-1 to U-12 in `reference-target.md`
  rather than scattered across a plan.** None of them is a measured behaviour, which is exactly why
  none of them belongs in `behaviours.md`.
- [x] `spec.md`, `plan.md` and `tasks.md` are all marked `Implemented`.

**One line of this list does not hold as written, and it is worth saying so plainly rather than
ticking it.** The second bullet reads *"every endpoint reaches the conformance level declared in
§6"*, and the level a route reaches is only as good as the level its **behaviours** are asserted at.
`/System/Info/Public` reaches L2 and `/System/Ping` reaches L2 in full. `/System/Info` reaches L2 for
everything a v1 installation can be in the state to answer — which is every field of §3.2, and not
its two refusals: the `401` is asserted one layer in and the `403` is unreachable. Neither is a
level this feature failed to reach; both are states this feature cannot enter. The distinction
matters because the row in §6 says *L2* and a reader deserves to know which requests that covers.
