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

- [ ] **Changes:** `internal/surface` — load `docs/compatibility/surface.yaml`; expose the canonical
  spelling per path, the methods registered on each path, and the owning feature and level. Refuse
  to load a row with an unknown level or a duplicate method-and-path.
- **Depends on:** —
- **Verified by:** the table loads all 59 rows; 001's four are present with their operations; a
  fixture with a duplicate row and one with `level: L9` are both rejected.
- **Spec reference:** §3.6; Principle VI.

## T9 — Path canonicalisation

- [ ] **Changes:** `internal/httpapi` — a middleware that folds a request path's **literal**
  segments and rewrites them to the route's own spelling, passes path **parameters** through byte
  for byte, trims one trailing slash, and answers two or more with an empty `404`. No redirect is
  ever issued.
- **Depends on:** T8
- **Verified by:** a table test over §3.6's table: canonical, any casing, one trailing slash — same
  route and same bytes; two slashes `404`; and a parameter whose casing survives.
- **Spec reference:** §3.6, AC-11; behaviours §1.14.

## T10 — Query key canonicalisation

- [ ] **Changes:** the same treatment for query parameter **names** against each route's declared
  spellings. **Values are never touched**, and an unrecognised key is left in place rather than
  dropped.
- **Depends on:** T8
- **Verified by:** `Limit`, `limit` and `LIMIT` reach the handler as the declared spelling; a value
  differing only in case is unchanged; an unknown key survives to be counted later.
- **Spec reference:** behaviours §1.15, §1.12.

## T11 — The refusal shapes, and `Allow` computed from the table

- [ ] **Changes:** the router's `NotFound` and `MethodNotAllowed` replaced — `404` and `405` with an
  empty body and **no content type** — and `Allow` built from `internal/surface`: every method that
  **path** has, sorted alphabetically. An unauthenticated refusal answers `401` with an empty body,
  `Content-Length: 0` and **no** `WWW-Authenticate`.
- **Depends on:** T8, T9
- **Verified by:** the measured case, which the router gets wrong on its own — `PUT`, `HEAD` and
  `OPTIONS` on `/System/Ping` must each answer `405` with `Allow: GET, POST`, where chi answers one
  arbitrary method
  `[measurement: github.com/go-chi/chi/v5 v5.3.2, Go 1.27.0, 2026-09-02]`. Plus an empty `404` with
  no content type, and the `401` shape.
- **Spec reference:** §3.6, AC-11; behaviours §1.11.

## T12 — `X-Response-Time-ms` and `Server`

- [ ] **Changes:** two middlewares — the response time in fractional milliseconds on **every**
  response, and `Server: Atrium/<version>` from the build stamp.
- **Depends on:** T1
- **Verified by:** both headers present on a `200`, on the empty `404`, on the `405` and on the
  `503` — the refusals are where a header added by the wrong layer goes missing.
- **Spec reference:** behaviours §1.9, §4.1.

## T13 — The readiness gate and the `503`

- [ ] **Changes:** the outermost stage. Until the server reports itself ready, **every** route
  answers `503` with `Retry-After` in full integer seconds, a `Message` header, and a `text/html`
  body — never JSON. The same response serves a deliberate withdrawal, with a different message and
  a longer hint, without stopping the process.
- **Depends on:** T1
- **Verified by:** a server held pre-ready answers `503` on all four routes **and on a path that
  matches no route**; `Retry-After` parses as an integer; the body's content type is `text/html`.
- **Spec reference:** §3.5, AC-12.

## T14 — Assemble the pipeline, and assert its order

- [ ] **Changes:** wire the stages in plan §6.7's order and nowhere else.
- **Depends on:** T9, T10, T11, T12, T13
- **Verified by:** three assertions that only the order can satisfy — a `503` from the gate still
  carries the response-time stamp and `Server`; a `404` from canonicalisation carries them too; and
  the gate answers before routing, so an unknown path answers `503` and not `404` while starting.
- **Spec reference:** plan §6.7; architecture §4.

## T15 — `LocalAddress`, as a pure function over `RequestFacts`

- [ ] **Changes:** `internal/system` — the three tiers of §3.4 over the domain's own
  `RequestFacts`. Tier 1 accepts the **subnet-scoped** published-URL form and picks the most
  specific matching prefix; tier 2 omits the port when it is the scheme's default; tier 3 returns
  the scheme and port the server is actually reachable on.
- **Depends on:** T3
- **Verified by:** a table test with synthesised requester addresses covering AC-7, AC-8 and AC-13,
  and a case where a certificate is configured and tier 3 still answers the reachable scheme — the
  deliberate divergence, asserted rather than assumed.
- **Spec reference:** §3.4, AC-7, AC-8, AC-13; behaviours §2.3, §4.2.

## T16 — `GET /System/Info/Public`

- [ ] **Changes:** the handler and its response model — exactly the seven fields of §3.1, with
  `ProductName` the literal `Jellyfin Server`, `OperatingSystem` the empty string, and `Version` the
  pinned reference version.
- **Depends on:** T7, T14, T15
- **Verified by:** a byte-compared golden on an empty installation, plus per-field assertions so a
  golden diff names which field moved. Answers before any user exists and before any library is
  configured.
- **Spec reference:** §3.1, AC-1, AC-2, AC-3.

## T17 — `GET /System/Ping` and `POST /System/Ping`

- [ ] **Changes:** both methods returning the bare JSON string `"Jellyfin Server"` — the **product
  name**, not the operator's friendly name.
- **Depends on:** T14
- **Verified by:** exact bytes on both methods. A test that would pass if the handler returned
  `ServerName` is not a test of this; the fixture sets a friendly name that differs.
- **Spec reference:** §3.3, AC-6.

## T18 — `GET /System/Info`

- [ ] **Changes:** the authenticated superset of §3.2, with the plan's stated values for the flags
  and paths. Token validation is taken through a port that **002 fills**; until then the only
  reachable states are *setup incomplete*, which the reference permits without a credential, and
  *any token*, which is invalid.
- **Depends on:** T16
- **Verified by:** the superset assertion — every field shared with `/System/Info/Public` agrees —
  and the `401` shape without a token once setup is complete.
- **Spec reference:** §3.2, AC-5.
- **Carried:** AC-5's *"and `200` with a valid one"* half cannot be proven here, because no
  credential exists until 002. It is recorded as a **criterion carried into 002** rather than
  marked met, and 002's task list closes it. See T21.

## T19 — The two cross-cutting sweeps

- [ ] **Changes:** `conformance` — the casing sweep over every registered response type, and the
  unit sweep. The unit sweep recognises **a date by its value, not by its name**, since
  `DateCreated` does not end in `Date`.
- **Depends on:** T16, T17, T18
- **Verified by:** both sweeps pass; and each one is proven to be able to fail — a deliberately
  camelCase field and a deliberately three-digit date, in test-only models, must each be caught.
  **A sweep that has never failed has proved nothing.**
- **Spec reference:** §6; [conformance L1](../../docs/compatibility/conformance.md#l1--shape).

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
