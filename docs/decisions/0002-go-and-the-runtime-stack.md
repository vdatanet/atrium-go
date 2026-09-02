# ADR-0002 — Go, and the runtime stack

**Status:** Accepted · **Date:** 2026-09-02

## Context

This number was reserved rather than exported. ADR-0002 at the source named Python and the stack
around it; it was withheld because the runtime is the largest single piece of the *HOW*, and
inheriting it would have made this repository a transliteration instead of a second independent
run ([PROVENANCE.md](../../PROVENANCE.md)). The index has carried the Python row and a link to
nothing ever since, and that row is itself one of the 157 recorded leak lines
(`docs/decisions/README.md:13`).

**The language is not what this record decides.** Go is the premise of the experiment: two
implementations of one set of specifications, on two platforms, where the second never reads the
first ([CLAUDE.md](../../CLAUDE.md)). A record that re-argued the language would be re-deciding the
question the repository exists to answer.

What this record decides is everything the language leaves open — **the HTTP layer, the
serialisation, and the dependency floor** — and it is not a matter of taste. Four documents
constrain it before any preference gets a vote:

- **[surface.yaml](../compatibility/surface.yaml)** — 59 routes, four of which put a wildcard
  *inside* a path segment.
- **[behaviours §1](../compatibility/behaviours.md)** — PascalCase, seven-digit dates, ticks,
  null-versus-absent, and an escape table that is not any JSON library's default.
- **[behaviours §1.11](../compatibility/behaviours.md#111-there-are-four-error-shapes-not-one)** —
  seven refusal shapes, several of which are a framework default turned off rather than on.
- **[Principle VIII](../constitution.md)** — conformance asserts on **bytes**. Every one of the
  above is invisible the moment a body is parsed.

**So the stack was measured before it was chosen**, on this project's own machine, with Go 1.27.0
`darwin/arm64`, on 2026-09-02. Three measurements decided three of the four questions.

### The standard library's router cannot express four of the 59 routes

`net/http.ServeMux` requires a wildcard to be a whole path segment, and rejects the pattern at
registration:

```
GET /Audio/{itemId}/stream.{container}                    panic: at offset 20: bad wildcard segment (must start with '{')
GET /Videos/{a}/{b}/Subtitles/{i}/Stream.{fmt}            panic: at offset 34: bad wildcard segment (must start with '{')
GET /Videos/{i}/hls1/{playlistId}/{segmentId}.{container} panic: bad wildcard name "segmentId}.{container"
GET /Audio/{itemId}/{file}                                ok
```

`[measurement: net/http.ServeMux, Go 1.27.0, 2026-09-02]`

The four are `GetAudioStreamByContainer`, `GetVideoStreamByContainer`, `GetHlsVideoSegment` and the
two subtitle fetch routes of [§8.1](../compatibility/api-surface-v1.md#81-subtitle-delivery). The
last line is the workaround the standard library leaves: capture the segment whole and split it in
the handler.

### The standard library gets `Allow` right and the refusal body wrong

```
PUT /UserFavoriteItems/{itemId}  ->  405  Allow: "DELETE, POST"
                                     Content-Type: "text/plain; charset=utf-8"
                                     body: "Method Not Allowed\n"
```

`[measurement: net/http.ServeMux, Go 1.27.0, 2026-09-02]`

The header is **exactly** what §1.11 measured against the reference — alphabetical, and on the one
pair where alphabetical and registration order differ. The body is not: the reference answers `405`
with an **empty body and no `Content-Type`**. So the refusal shapes have to be taken over from the
router whatever the router is.

Path handling is wrong in three directions at once, which is the same trap §1.14 records against
Starlette rather than a smaller version of it:

```
/System/Info/Public     -> 200   /System/Info/Public/   -> 404   (reference: 200)
/system/info/public     -> 404                                   (reference: 200)
/System/Info/Public//   -> 307 -> /System/Info/Public/            (reference: 404)
```

`[measurement: net/http.ServeMux, Go 1.27.0, 2026-09-02]`

### `encoding/json` gets the structure right and the escaping wrong

```json
{"Name":"28 años después <b> & 'x' + `y`","ChannelId":null,"IndexNumber":0,"RunTimeTicks":12345,"IsFavorite":false}
```

`[measurement: encoding/json, Go 1.27.0, 2026-09-02]`

Read against §1.7 and §1.16, that one line is four findings:

- **Field order is declaration order.** Deterministic, and ours to choose per struct — which is
  what a byte comparison needs and what a map-based encoder could not give.
- **Pointer fields are `WhenWritingNull` exactly.** A nil pointer with `omitempty` is absent, a
  pointer to zero emits `0`, and a nil pointer *without* `omitempty` emits `null` — so §1.7's
  general rule and its two named exceptions (`ChannelId` everywhere, `ParentId` on `/UserViews`)
  are the same mechanism rather than a special case bolted on.
- **The escape hex is lower case**, where the reference writes `\u003C`.
- **Non-ASCII is passed through, and four of the seven ASCII characters are too.** `ñ`, `é`, `'`,
  `` ` `` and `+` all came out literal; §1.16 requires all five escaped.

### The newest standard-library encoder cannot express it either

Go 1.27 ships `encoding/json/v2` and `encoding/json/jsontext` in the standard library, importable
with no `GOEXPERIMENT` set. It was checked rather than assumed, because a v2 encoder that could be
told which characters to escape would remove the need for a pass of our own.

It cannot be. `jsontext`'s entire escaping surface is two options — `EscapeForHTML` and
`EscapeForJS` — and with **both** enabled the output is byte-identical to v1's:

```
v1                     : "\" \u0026 ' + \u003c \u003e ` ñ é / = : ! * ( ) - _"
v2 default             : "\" & ' + < > ` ñ é / = : ! * ( ) - _"
v2 EscapeForHTML+ForJS : "\" \u0026 ' + \u003c \u003e ` ñ é / = : ! * ( ) - _"
```

`[measurement: encoding/json, encoding/json/v2, encoding/json/jsontext, Go 1.27.0, 2026-09-02]`

Against §1.16 that is still missing four of the seven ASCII characters, every non-ASCII character,
and the case of the hex. **So the pass below is not a preference between two ways of escaping; it is
the only way to emit these bytes from Go**, and that is a stronger reason than the one this record
was first written with.

## Decision

### Go, with the language floor at 1.25

`go.mod` declares `go 1.25`. Nothing in this record needs anything newer, and a floor two releases
below the toolchain that took the measurements above keeps a contributor on a distribution Go from
being excluded. The floor moves in its own commit, with a reason, the way the reference pin does.

### `net/http` for the server, `github.com/go-chi/chi/v5` for routing

chi serves all four of the routes the standard library rejects, verbatim as `surface.yaml` spells
them, with the parameters split correctly:

```
/Audio/abc/stream.mp3                   -> 200  itemId="abc" container="mp3"
/Videos/i1/ms1/Subtitles/2/Stream.vtt   -> 200  a="i1" b="ms1" i="2" format="vtt"
/Videos/i1/hls1/pl1/7.mp4               -> 200  seg="7" container="mp4"
```

`[measurement: github.com/go-chi/chi/v5 v5.3.2, Go 1.27.0, 2026-09-02]`

chi is chosen for three properties and not for its feature list. It is **`http.Handler` all the way
down**, so nothing it does is un-opt-out-able; its **`NotFound` and `MethodNotAllowed` handlers are
settable**, which is where §1.11's empty-bodied `404` and `405` go; and it brings **no transitive
dependencies**.

### Path and query canonicalisation is ours, not the router's

A middleware rewrites a request's path to the matched route's own spelling before routing, and its
query keys to the route's declared spellings before binding — §1.14 and §1.15, exactly as those
entries describe, with **path parameters and query values never touched** because they are data.

This is not a tidiness layer, and the measurement says why:

```
/Videos/i1/ms1/Subtitles/2/stream.vtt   -> 404
```

`[measurement: github.com/go-chi/chi/v5 v5.3.2, Go 1.27.0, 2026-09-02]`

Lower-case `stream.vtt` is not a hypothetical client being careless. It is **the spelling the
reference's own subtitle playlist emits**, against a declaration that spells it `Stream.{format}`
([§8.1](../compatibility/api-surface-v1.md#81-subtitle-delivery)). Without this middleware a client
that follows the address it was handed gets a `404` from us and cues from the reference.

### `encoding/json`, pointer-typed optional fields, and one project-owned wire package

The encoder stays; what it gets wrong is taken over in **one place**, a `wire` package that owns
serialisation for the whole project:

- **HTML escaping is switched off in the encoder**, so exactly one piece of code escapes anything
  and it is ours.
- **The escape pass writes §1.16's table** — every non-ASCII character and the seven ASCII ones, as
  `\uXXXX` with upper-case hex. It **counts backslash parity** rather than searching for `\u`, for
  the reason §1.16 states: a *value* that genuinely contains the six characters `\u00e9` must
  survive as those six while the encoder's own escapes are rewritten.
- **The tick type and the date type live here**, so §1.2's seven fractional digits and §1.3's
  100-nanosecond unit cannot be forgotten at a boundary.
- **Optional fields are pointers**, project-wide. `omitempty` on a non-pointer is banned, because
  it omits `0`, `false` and `""` — which is not §1.7's rule but a silently different one.

The two cross-cutting L1 sweeps read struct tags by reflection over the registered response types:
every field name PascalCase, every `*Ticks` an integer, every `*Date` seven digits and a `Z`.

### No cgo. `CGO_ENABLED=0`, and WebP is encoded by ffmpeg

The enabling fact is a conformance one, not a preference: **image bytes are not a comparison
surface.** The allowlist excuses them under `content-hash`, and 006's own goldens assert *headers
and dimensions, not pixels*, on the stated grounds that encoder output is not stable across library
versions ([006 §Conformance](../../specs/006-images/spec.md)). So the encoder is an operational
choice with no compatibility consequence — which is exactly the kind of choice that should be made
on deployment cost.

ffmpeg is already a hard dependency of 008: every delivery route reads bytes and every negotiation
reads a codec. Encoding WebP through a binary the server already requires adds nothing to install,
and the resized-image cache of [006 §3.3](../../specs/006-images/spec.md) absorbs the per-request
process cost, since a cache hit runs nothing at all.

### Standard library first, and chi is the only dependency this record adds

`log/slog` for logging, `os/exec` for ffmpeg and ffprobe, `testing` for tests — with **no assertion
library**, because a golden test compares bytes and `testing` plus a byte comparison is the whole
of that need. A further dependency is argued where it is needed, in the plan that needs it, not
pre-approved here.

## Consequences

- **One static binary.** `CGO_ENABLED=0` cross-compiles to every platform the project targets from
  any of them, with no system libraries on the build host or in the image. ffmpeg and ffprobe are
  external binaries beside it, which is the shape 008 already assumed.
- **`wire` is a choke point, and that is its purpose.** §1.1, §1.2, §1.3, §1.7 and §1.16 are five
  whole-project constraints that a route author must not be able to forget one of. The Python side
  reached the same conclusion by a different road — §1.7 records a per-route flag as "one someone
  eventually forgets, and the one they forget is the one a client sees a stray `null` on".
- **Every body is written twice**, once by the encoder and once by the escape pass. The cost is a
  copy per response; the alternative is an encoder we maintain. Goldens are what would catch the
  pass getting it wrong, and they compare bytes.
- **The refusal shapes are handler code, not framework configuration.** chi's `NotFound` and
  `MethodNotAllowed` are set to §1.11's empty-bodied forms, and the `405` must additionally carry
  the alphabetical `Allow`.
- **The canonicalising middleware gets its own tests before any route does.** It is the one piece
  measured here to be load-bearing for a *real client following a real address*, and it is the one
  piece no route's own tests would exercise.
- **This record decides nothing about the store, the password hashing, or the architecture.**
  ADR-0003 and ADR-0006 stay reserved and their index rows stay dangling;
  [docs/architecture.md](../architecture.md) — which every `plan.md` inherits from — is still owed.
  Three documents cite it: `specs/README.md`, the plan template, and ADR-0007 twice, by anchor.
  (CLAUDE.md says nine; nine is `docs/roadmap.md`'s count in PROVENANCE's table, and this record
  repeated the error before checking.)
- **One leak line closes and PROVENANCE is left alone.** `docs/decisions/README.md:13` named Python;
  it now names this project's own decision. PROVENANCE's leak table still records what the export
  contained, because it is a description of the exported bytes and not a to-do list.

### What is not verified, and is owed

Principle II applies to this record as much as to a behaviour, and three claims here are reasoned
rather than measured:

- **chi's `Allow` ordering on a `405` is not measured.** The standard library's is, and is
  alphabetical; chi's was not put under the `PUT /UserFavoriteItems/{itemId}` case. Until it is, the
  ordering is the middleware's to guarantee rather than the router's. ⚠️ UNVERIFIED
- **ffmpeg's WebP quality scale is not mapped** to the `quality` 0–100 of
  [006 §3.2](../../specs/006-images/spec.md). ⚠️ UNVERIFIED
- **EXIF orientation is not confirmed** to be applied on the ffmpeg path. The fixture library plants
  an image carrying an orientation tag precisely because no remote request reaches that edge
  ([conformance L2](../compatibility/conformance.md)), so the check exists and has not been run.
  ⚠️ UNVERIFIED

## Alternatives rejected

**The standard library's `ServeMux` alone, for zero dependencies.** It cannot express four of the 59
routes — measured above, at registration, as a panic. The workaround is real: capture
`/Audio/{itemId}/{file}` and split on the dot. What it costs is that the router stops being the
place the surface is declared, and Principle VI's route-registration check — *the server exposes
exactly `surface.yaml`'s rows and no others* — has to reconstruct four rows from handler code to
know what is served. A dependency that keeps the surface declarative in the one file CI reads is
worth more than a zero in `go.mod`.

**`gorilla/mux`.** It also matches inside a segment, by regexp. It matches more slowly, it has been
through an archival and a revival, and it buys nothing over chi for this surface.

**A full framework — echo, gin or fiber.** Rejected on §1.11 rather than on weight. These bring
their own JSON serialisation, their own binding and their own error rendering, and all seven of the
reference's refusal shapes are a fight with one of them: an empty `401` with `Content-Length: 0` and
no `Content-Type`, a `404` with no body at all, problem details under
`application/json; charset=utf-8` rather than `application/problem+json`, a 25-byte `text/plain`
body with no charset, and a JSON-encoded bare string. A framework that makes the common case easy
makes every one of those a special case. Fiber is not `net/http` at all, which forfeits the standard
library's `Range` and streaming handling that every delivery route in 008 needs.

**Our own streaming JSON encoder.** One pass instead of two, and total control at the point of
writing. It is also a serialiser this project maintains forever, with reflection or code generation
underneath it either way, to save a copy per response on a server whose expensive work is reading
media files and running ffmpeg. If a profile ever shows the copy mattering, this is the thing to
revisit — with the profile in hand.

**Generated marshallers (easyjson and the like).** Fast, and they would let the escape table be
written straight into the generated code. They add a build tool and a directory of generated files
to a repository whose stated practice is that goldens are *reviewed, never blindly regenerated*.
The tension is not fatal, but it is not paid for by a bottleneck anyone has measured.

**cgo with libvips or libwebp.** The best image quality and the best throughput, and full lossy
WebP. It ends static builds and cross-compilation, and puts a system library on every contributor's
machine and in every container image — for an output that the allowlist already excuses and that
006's goldens deliberately do not assert. Paying a permanent deployment cost for bytes nothing
compares is the wrong trade.

**A pure-Go WebP encoder.** It would keep everything in one binary with no external process. The
available ones are lossless-oriented, and a lossless WebP of a film poster is larger than the JPEG
it replaced — which inverts the point of §3.3's negotiation, where a browser offering `image/webp`
is offered WebP to save bytes.

**Deciding the store here as well.** The store is ADR-0003's, and it is a discussion of its own —
schema, migrations, and what concurrency a scan writing while requests read actually needs. Folding
it in would have produced one record that argued two things and reviewed neither.
