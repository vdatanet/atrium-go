# Project architecture

**Status: the shape, before there is code. Written 2026-09-02.**

This is the document every `plan.md` inherits from. The plan template's §2 asks for *"project-level
choices this plan takes as given"* and points here and at the
[ADRs](decisions/); [specs/README.md](../specs/README.md) says the same, and adds the rule that
matters: **a plan restates a choice only where it deviates, and a deviation needs its own ADR.**

So this document exists to be decided once instead of twelve times. It is not a specification — it
states no observable behaviour — and it is not an ADR, because it answers no single question. Where
something here needed a decision with alternatives, it is an ADR and this document cites it.

**It was owed from the first commit.** The source project's architecture document was withheld on
purpose ([PROVENANCE.md](../PROVENANCE.md)): it is the largest single piece of the *HOW*, and it is
the piece whose inheritance would most quietly turn a second implementation into a transliteration.
Three documents have linked to it and got nothing since the export.

---

## 1. The one constraint that shapes everything

**Principle VIII: a behaviour is done when a test asserts it at the HTTP boundary, on bytes.**

Every wire-format fact in [behaviours §1](compatibility/behaviours.md) is invisible the moment a
body is parsed — casing, `null`-versus-absent, integer-versus-string, the escape table, the number
of fractional digits on a date. A parsed-object assertion cannot fail on any of them.

That turns the whole architecture into one question: **where may a Jellyfin-shaped fact live?** The
answer this document gives is *in exactly one place per fact, and that place is a choke point a
route author cannot route around.* The alternative — each handler getting it right — is not a
weaker version of the same thing. [§1.7](compatibility/behaviours.md#17-a-null-property-is-absent-everywhere-by-one-setting)
already says why: a per-route flag *"is one someone eventually forgets, and the one they forget is
the one a client sees a stray `null` on"*.

Three families of fact are whole-project constraints rather than per-endpoint ones:

| Family | Sections | Consequence here |
|---|---|---|
| Serialisation | §1.1 PascalCase, §1.7 null-versus-absent, §1.16 the escape table | One package writes every response body. There is no second way to produce JSON. |
| Units | §1.2 seven-digit dates, §1.3 ticks | The tick and date types are the only ones the rest of the code may hold. §1.3 puts ticks in *storage*, not only on the wire, "so no conversion can be forgotten at a boundary". |
| Refusal | §1.11's seven shapes, §1.14 and §1.15's matching | Routing, canonicalisation and refusal belong to the edge and are written once, not per handler. |

---

## 2. Layers, and the direction of dependency

Four layers, and dependencies point one way only.

| Layer | Holds | May import |
|---|---|---|
| **Entry** | Flag parsing, configuration, wiring, start and stop | everything |
| **Edge** | The router, the middleware, the 59 handlers, the serialiser | Domain, Ports |
| **Domain** | The measured semantics — sorting, resume branches, negotiation, playlist rules | Ports, and the unit types |
| **Ports** | Interfaces the domain declares: the store, media inspection, the clock | nothing of ours |

**The domain imports no HTTP.** A rule about sort order or a resume threshold is a fact about
Jellyfin's semantics, not about a request, and the measured ones in
[behaviours §2](compatibility/behaviours.md) read as ordinary functions over ordinary values. What
they must *not* do is reach for a request, a header or a status code, because then the only way to
test a six-branch resume rule is to issue a request.

**The unit types are a leaf, imported by both.** They are not part of the serialiser even though
they serialise specially, because §1.3 says a duration is a tick everywhere — in the domain and in
the store — and a domain that imported the serialiser to hold a duration would have inverted the
whole diagram to get a number.

**Ports are declared by the domain and implemented outward.** This is what lets
[ADR-0003](decisions/) be argued after features are planned rather than before: a plan writes
against the interface it needs, and the eventual store decision implements interfaces that already
exist instead of rewriting the code that would have named a database.

### Three rules that are specific to this language, and each of them bites once

- **No map iteration may reach a response body.** Go randomises map iteration order deliberately.
  Principle VII forbids order that derives from anything but stable input, and L3 compares list rows
  **by position** ([conformance L3](compatibility/conformance.md)), which promotes ordering into the
  contract. Anything ordered is sorted explicitly, on a stated key. A response that is right on
  Tuesday and differs on Wednesday is the worst failure mode this project has, because the
  differential will report it as a real difference.
- **`omitempty` on a non-pointer is banned.** It omits `0`, `false` and `""`, which is not §1.7's
  rule but a silently different one. Optional fields are pointers, project-wide
  ([ADR-0002](decisions/0002-go-and-the-runtime-stack.md)).
- **The wall clock is a port.** `wall-clock` is one of the allowlist's four derivation classes, so
  the differences a clock creates are already enumerated; a clock the tests replace is what keeps a
  golden body stable between two runs.

---

## 3. Repository layout

```
atrium-go/
├── cmd/atrium/            the binary: flags, configuration, wiring, start and stop
├── internal/
│   ├── wire/              every response body is written here, and nowhere else
│   ├── units/             ticks and dates — the leaf both the domain and the wire hold
│   ├── httpapi/           router, middleware, refusal shapes, the 59 handlers
│   ├── <domain packages>  one per owning feature of surface.yaml
│   ├── ports/             the interfaces the domain declares
│   └── media/             the ffmpeg and ffprobe boundary
├── conformance/           HTTP-boundary tests, and the goldens they compare
│   └── testdata/golden/
├── tools/                 the checks CI runs that are not `go test`
├── docs/                  this document, the constitution, the ADRs, the compatibility set
├── specs/                 the twelve features
└── reference/             git-ignored: Jellyfin source, the pinned OpenAPI document, reports
```

**Everything but `cmd` and `conformance` is under `internal/`.** This project's only public surface
is an HTTP API it does not own. Nothing here is a library, nothing here should be imported by
anything, and `internal/` says so to the compiler instead of to a reader.

**`cmd/atrium` is wiring and nothing else.** No behaviour, no decision, no branch that a test would
want to reach. If something there is worth testing, it is in the wrong place.

**`conformance/` speaks HTTP and imports nothing of ours.** It starts the server and issues
requests, which is the whole of Principle VIII: a test that can reach inside can assert on a value
the wire never carried, and will keep passing when the wire stops carrying it. **Go does not enforce
this** — `internal/` restricts imports across module paths, not within a module, so the compiler
will happily let a conformance test reach into `internal/`. It is enforced instead by a CI check
over `go list -deps`, and stated here so that the check is understood as load-bearing rather than
pedantic.

**`tools/` holds Go programs**, run as `go run ./tools/<name>`. A Go repository already carries a
toolchain, and adding a second language for tooling would be a dependency bought with nothing.

**The probes and the differential harness are not here, and are not copied here.** They stayed in
the source repository, and this project points them at its own server over HTTP —
`tools/differential.py --atrium <url>` already takes one. They are run as black boxes, never read.
That is not a restriction this project works around; it is what makes the run's answer mean
anything.

**`reference/` is git-ignored**, because reference material is fetched at development time and never
vendored ([ADR-0005](decisions/0005-licence.md)). So is `.env`, which is where the harness reads
`ATRIUM_USERNAME`, `ATRIUM_PASSWORD` or `ATRIUM_TOKEN` and the reference's own credentials
([conformance L3](compatibility/conformance.md)).

### What ADR-0007's citation of this section meant, and what it means now

[ADR-0007](decisions/0007-a-container-runtime-for-the-reference-instance.md) cites *"architecture
§3"* for a rule that does not survive the crossing: that `tools/` is standard library only, on a
Python 3.9 floor. The section number is honoured — this is still where the layout and the `tools/`
rule live — and the rule itself is now the Go one above.

The part of ADR-0007 that depended on it **does** survive, and is the reason the rule existed at
all: the harness composes its subprocess arguments itself and talks to a container runtime through
its command line rather than through an SDK, so that it runs before an environment has been built.
Nothing above weakens that.

---

## 4. The compatibility boundary

One table, because this is the part a plan most often needs to look up rather than re-derive.

| Fact | Lives in | Never in |
|---|---|---|
| PascalCase field names (§1.1) | struct tags, swept by reflection over every registered response type | a handler |
| Null-versus-absent, and its two exceptions (§1.7) | pointer-typed fields; `ChannelId` and `/UserViews`' `ParentId` named in one place | a per-route flag |
| The escape table (§1.16) | the one pass in `wire`, with the encoder's own HTML escaping switched off | anywhere else |
| Seven fractional digits and a `Z` (§1.2), ticks (§1.3) | the unit types in `units` | an ad-hoc format string |
| Case-insensitive paths, one trailing slash (§1.14) | the canonicalising middleware | the router's own behaviour |
| Case-insensitive query names (§1.15) | the same middleware | a handler reading two spellings |
| The seven refusal shapes (§1.11) | the router's `NotFound` and `MethodNotAllowed`, and one refusal helper | a handler inventing a body |
| `X-Response-Time-ms` on every response (§1.9) | middleware | anywhere else |
| `application/json; charset=utf-8` (§1.10) | the response writer that produced the body | middleware bolted on afterwards |
| `Server: Atrium/<version>` (§4.1) | middleware, from a build-stamped version | a constant edited by hand |

**Middleware order is part of the contract, not a detail.** Canonicalisation runs before routing,
because it rewrites the path the router will match. The response-time stamp must wrap everything it
claims to be timing. The refusal shapes must survive every layer above them — a middleware that
helpfully adds a `Content-Type` to an empty `401` breaks §1.11 for the whole surface at once.

**The canonicaliser is load-bearing for a real client, and that is measured.** chi answers `404` to
`/Videos/i1/ms1/Subtitles/2/stream.vtt`
`[measurement: github.com/go-chi/chi/v5 v5.3.2, Go 1.27.0, 2026-09-02]`, and lower-case
`stream.vtt` is exactly the spelling **the reference's own subtitle playlist emits**, against a
declaration that spells the route `Stream.{format}`
([api-surface §8.1](compatibility/api-surface-v1.md#81-subtitle-delivery)). Without this middleware,
a client that follows the address it was handed gets a `404` here and cues there. It is therefore
the first thing built and the first thing tested, before any route uses it.

**Path parameters and query values are data and are never rewritten** — only the segments a route
declares literally, and only the *names* of query parameters. §1.14 says why: lowercasing an
identifier is invisible until something case-sensitive reads one.

---

## 5. Deployment shape

**One process. No second service. One data directory.** A user installs a binary and points it at a
directory; there is no broker, no worker, no sidecar and no database server to run.

- **A single static binary.** `CGO_ENABLED=0`, so it cross-compiles to every target from any host
  with no system libraries ([ADR-0002](decisions/0002-go-and-the-runtime-stack.md)).
- **`ffmpeg` and `ffprobe` beside it**, found on `PATH`. They are the only external executables, and
  008 already assumed them: every delivery route reads bytes and every negotiation reads a codec.
- **One data directory**, given by flag or environment. It holds the store, the resized-image cache,
  transcoding scratch space, and the ignored-parameter tally. It is the *installation path* the
  allowlist's fourth derivation class already excuses differences in.

**What in it is disposable, and what is not.** The image cache and the scratch space may be deleted
at any moment with no change to any response body ([006](../specs/006-images/spec.md) states this as
an acceptance criterion). The store is not. A plan that adds a third kind of state says which of the
two it is.

**Graceful shutdown is load-bearing, in two independent ways**, which is why it is stated here and
not left to whoever writes `main`:

- **Every child process must die with the parent.** 008 puts it plainly: answering a stop while
  leaving work running *"accumulates processes until the machine dies"*. Children are started
  context-bound and in their own process group, and the group is killed on the way out.
- **The ignored-parameter tally is written when the server stops.** The differential reads *"the
  tally Atrium wrote when it last stopped, which is the only moment the count is complete"*
  ([conformance L3](compatibility/conformance.md)) — a file in the data directory, deliberately not
  an endpoint Jellyfin does not have. A process killed without flushing it silently degrades a
  harness run on another machine.

**The version is build-stamped, and it is not cosmetic.** A differential run **refuses to start**
unless the two servers differ on the `Server` header, because `ProductName` is `Jellyfin Server`
here on purpose (§4.1) and the obvious check would otherwise admit a run comparing this project
with itself. `Server: Atrium/<version>` is therefore the one thing that tells the two apart, and a
binary that cannot state its version cannot be measured.

---

## 6. State, and the store boundary

**The store is a port.** The domain declares the interfaces it needs; an implementation satisfies
them. Above that line there is no SQL, no query builder, no transaction object and no vocabulary
belonging to any particular store.

[ADR-0003](decisions/0003-sqlite-as-the-store.md) implements those interfaces with an embedded
SQLite database, and the boundary stays: no SQL, no query builder and no transaction object appears
above it. **The store is also split in two** — a *derived* half rebuilt by a rescan and a *precious*
half that is the only copy — and the rule that makes the split work belongs here rather than in the
record, because it is a constraint on every feature: **no reference points from the precious half
into the derived half.** User data names an item by its derived identifier string, so rebuilding the
whole library leaves every favourite pointing at the right item.

**Identity is derived, never stored as a sequence.** Principle VII, and
[§1.4](compatibility/behaviours.md#14-item-identifiers-are-32-lowercase-hex-characters): a rescan
must not invalidate a client's caches, favourites or resume positions. Atrium keys on the path
**relative to its library root** ([003 §3.6](../specs/003-library-configuration-and-scanning/spec.md)),
so moving a library root costs nothing — where the reference, keying on the absolute path, silently
discards every client's state for that library. Reproducing the reference's exact identifier bytes
is not a goal and never was; reproducing its *stability* is.

**Ticks are the stored unit**, not a serialisation of one (§1.3). The conversion from what a prober
reports happens once, at ingestion, and rounds rather than truncates.

---

## 7. External processes

One package owns every subprocess, and nothing else in the tree calls `os/exec`.

**Every child is context-bound.** A cancelled request, a timeout and a shutdown are the same
mechanism, so there is one way for work to stop and one place it is implemented.

**A transcode registry is a project-level component, not 008's private detail.** `DELETE
/Videos/ActiveEncodings` has to find and terminate work that a *different* request started, keyed by
the session the negotiation opened — so the registry is touched by the session model 002 owns and by
the delivery routes 008 owns, and a component two features share is one this document names rather
than one either of them invents. 008's acceptance criterion is observational: the process is gone
and the scratch space with it.

**No cgo, and WebP is encoded by ffmpeg** ([ADR-0002](decisions/0002-go-and-the-runtime-stack.md)).
The reason it costs nothing in conformance is worth repeating where a plan will look for it: image
bytes are excused by the allowlist under `content-hash`, and 006's goldens assert headers and
dimensions rather than pixels, on the stated grounds that encoder output is not stable across
library versions.

---

## 8. Testing and conformance

The four levels are defined in [conformance.md](compatibility/conformance.md). This section says
only where each one lives.

| Level | Where | Runs in CI |
|---|---|---|
| **L0 — Routed** | `conformance/`, over `surface.yaml`: exactly these routes and no others | yes |
| **L1 — Shape** | `conformance/`, goldens in `testdata/golden/`, plus the two reflection sweeps | yes |
| **L2 — Semantic** | `conformance/`, over the fixture libraries | yes |
| **L3 — Differential** | not `go test` at all — the harness, against a real Jellyfin | **never** |

**Unit tests live beside their package**, as Go expects. What may not live beside a package is an
assertion about a wire fact: those go through the HTTP boundary or they are not evidence.

**Goldens are reviewed, never blindly regenerated.** An update flag exists; a diff in a golden file
is a contract change and is read like one in review.

**Two fixture worlds, because they answer different questions** — a scanning tree of paths and
filler bytes, and a small media tree that ffmpeg really encodes into each container and codec the
delivery tests need. Tests that reach the binaries are separated by a build tag, and the suite
staying green without it is the check that they all carry it.

**No test opens a network connection, and no CI job contacts or starts a Jellyfin.** ADR-0007 states
the rule and notes that the source project enforced it rather than promising it; this project owes
the same enforcement, and its home is `conformance/`'s own harness. The consequence is stated
plainly there and repeated here because it is uncomfortable and true: **the strongest check in the
project is the one that is never automatic.**

**The route registration check is the automated half of Principle VI.** It reads `surface.yaml` and
asserts the server exposes exactly those rows — an endpoint served and not listed fails it, and so
does one listed and not served, once its feature claims to be implemented.

---

## 9. Configuration, identity and logging

**Process settings come from flags, with an environment fallback**: the bind address, the data
directory, the log level. There are few of them and they are not a feature.

**Library configuration lives in the data directory** and its shape is
[003](../specs/003-library-configuration-and-scanning/spec.md)'s to specify, not this document's.

**Logging is `log/slog` to standard error, structured.** The response-time *header* is not a log
line: §1.9 measured that the reference's two configuration flags gate a slow-response log line and
**not** the header, which is unconditional. Getting that backwards would put a difference on every
response in the project.

**The server identifies as two things at once, deliberately.** `ProductName` is `Jellyfin Server`,
because clients branch on it and Principle I outranks the discomfort
([§4.1](compatibility/behaviours.md)). `Server` is `Atrium/<version>`. The first is what makes
clients work; the second is what makes measurement possible.

---

## 10. What this document does not decide

- **The store** — [ADR-0003](decisions/), reserved and unwritten. §6 gives it a boundary, not an
  answer.
- **Password hashing** — [ADR-0006](decisions/), reserved and unwritten.
- **The package split inside the domain.** §3 names the directories; which feature gets which
  package, and where a boundary falls between two of them, is a `plan.md`'s §3 and is argued with
  the feature in front of it.
- **The order features are built in.** That is `docs/roadmap.md`, which is still owed.
- **Anything about the surface.** 59 endpoints, chosen by named consumers, is Principle VI and
  [api-surface-v1.md](compatibility/api-surface-v1.md). Growing it is a roadmap decision.

And the standing rule this document is the other half of: **a plan takes what is here as given, and
deviates only with its own ADR.** A deviation without one is not a deviation, it is an
inconsistency.
