# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repository is

**Specifications and measurements for Atrium — a from-scratch implementation of the Jellyfin
`10.11.x` HTTP API. There is no code here yet, and that is deliberate.**

The 37 files were exported from a *different* repository (`vdatanet/atrium-media-server`, commit
`531c172`, 2026-09-02) that implemented these same specifications in Python. 513 files were
withheld — the implementation, its tests, its plans, its task lists, its build config, and the ADRs
covering stack, store and password hashing. See [PROVENANCE.md](PROVENANCE.md) for the full
withholding table and the reasoning.

This repository (`atrium-go`) is the **receiving project**. It writes the Go implementation from
the WHAT, without inheriting the HOW.

There is no build system, no test runner, no linter and no `git` repository here yet — nothing has
been decided. Do not invent commands; there are none to run. Do not scaffold a Go module, choose a
router, or pick a store unless asked: those are ADR-shaped decisions (see *Decisions this project
still owes*).

## The experiment, and the contamination boundary

**This repository is an experiment: can one set of specifications produce two indistinguishable
servers on two different platforms?** The Python implementation exists and passed its own
conformance gates. This is the second, independent run — Go, from zero. If both servers answer a
real Jellyfin identically, the specifications did their job; wherever they diverge, the
specification was underdetermined and the difference is the finding.

The experiment is only meaningful if the second implementation never sees the first. **A
transliteration would answer nothing.**

### The sibling checkouts on this machine

`../atrium-media-server`, `../jellyfin`, `../jellyfin-dev-data`, `../embeat` (the music client) and
`../jellyfin-apple-tv` (the video client) all sit beside this repository.

**Never read anything withheld from `../atrium-media-server`.** The boundary is not "source code" —
it is the withheld/exported split PROVENANCE.md already drew and hashed. `plan.md`, `tasks.md`,
tests, audits, `architecture.md` and `roadmap.md` contain no Python and contaminate just as much:
they are the HOW and the STEPS, withheld precisely so this project derives its own.

The 37 exported files in this repository **are** the complete non-contaminating surface. There is
nothing to gain by looking further, and the experiment's validity to lose.

What may be read, and why:

| Source | Allowed | Why |
|---|---|---|
| `../jellyfin` source | **Yes**, as a behavioural reference — cite `file:line @ v10.11.11` | Principles II and IV: read *what it does*, never translate it |
| `../embeat`, `../jellyfin-apple-tv` | **Yes** | Real consumers; the v1 surface was derived from them, and `client-*.md` here are the existing analyses |
| `../atrium-media-server` exported files | **Done and closed** — the two stale ones were refreshed on 2026-09-02 (see below). Nothing further. | Same documents, newer bytes — not a boundary crossing |
| `../atrium-media-server` anything else | **No** | Withheld deliberately |

### The two stale files were refreshed — 2026-09-02

`specs/010-conformance-harness/spec.md` and `specs/README.md` carry source commit `681b083`; the
other 35 files carry the export ref `531c172`. The refresh is recorded in PROVENANCE.md.

It brought one change, D-7: **010 is `Implemented` at the source**, and AC-2 no longer claims both
servers produce the same library — measured, they differ in forty-seven declared places over the
six fixture libraries, every one owned by 003 or 004. AC-2 now states the comparison that actually
runs: the reference's reading recorded, Atrium's scan compared against it, an undeclared difference
failing **and a declared one that has gone away failing too**.

That last clause is the shape this project's checks take, and it is worth copying: a conformance
assertion is a declared inequality, not an equality. This is the single most useful thing 010 hands
the Go implementation.

**The sibling repositories are closed from here on.** This was the one outstanding read, and it is
done.

### How the result gets measured

The instrument is already specified: **feature 010, the differential harness**, plus the 69 probes
that stayed in the Python repository *"pointed at the new server over HTTP"*. They are run as black
boxes against this server — executed, never read.

That yields the experiment's actual answer:

1. Each server against the same digest-pinned Jellyfin → one differential report each.
2. **Diff the two reports.** Identical reports mean the specifications determined the behaviour.
   Every asymmetry is a place where they did not.
3. The deliberate divergences in `behaviours.md` should reproduce in Go *from the argument alone*.
   One that does not reproduce is a divergence that was really an implementation accident.

### Where contamination is most likely to have already happened

The **157 leak lines** are the highest-value part of this experiment, not merely an untidiness.
Each is a spot where the WHAT may have absorbed the Python HOW — a behaviour stated in terms only
that stack makes natural. Where Go cannot satisfy such a line without transliterating, the
specification, not the implementation, is what needs amending.

## Working agreements

- **Never commit to `main`.** Every change goes on a branch, opens a pull request, and reaches
  `main` by merge. The root commit is the only exception, because a pull request needs a base.
- **English in everything that lands here** — identifiers, comments, documentation, commit
  messages, branch names, pull request titles and bodies. This is Principle IX, and it holds even
  when the conversation that produced the change was in another language.
- **Reference material is never vendored.** Jellyfin source and the pinned OpenAPI document are
  fetched into the git-ignored `reference/` directory at development time (ADR-0005).
- The repository is **public** from its first commit, so Principle X applies to what it says about
  Jellyfin: an independent implementation, unaffiliated and not endorsed.

## The binding documents

Read in this order. The first two outrank everything else in the repository.

| File | What it is |
|---|---|
| [docs/constitution.md](docs/constitution.md) | Ten principles that do not bend. A conflict is resolved in favour of this file, or the file is amended first. |
| [specs/README.md](specs/README.md) | The Spec-Driven Development workflow, the directory convention, and the status ladder. |
| [docs/compatibility/behaviours.md](docs/compatibility/behaviours.md) | 3,215 lines: every measured Jellyfin behaviour, every replicated defect, every deliberate divergence, every accepted gap. The single most important reference when implementing. |
| [docs/compatibility/api-surface-v1.md](docs/compatibility/api-surface-v1.md) | The 59 endpoints of v1 and the real clients that call each one. |
| [docs/compatibility/conformance.md](docs/compatibility/conformance.md) | The L0–L3 proof levels and the machinery for each. |
| [docs/compatibility/reference-target.md](docs/compatibility/reference-target.md) | Exactly which Jellyfin "compatible" means, plus the register of prior measurements that still owe a probe. |
| [docs/glossary.md](docs/glossary.md) | Jellyfin's vocabulary and this project's own (delta, non-improvement, provenance). |
| [docs/decisions/](docs/decisions/) | ADRs. One per file, immutable once accepted — a wrong one is *superseded*, never edited. |

### The principles that bite most often

- **I — Zero delta.** No endpoint, field, name, casing, type or unit that Jellyfin does not have.
  Not even behind a flag. A better idea gets written into `behaviours.md` §6 as a *non-improvement*
  and then not done.
- **II — Behaviour is measured, not assumed.** Every compatibility claim carries provenance:
  `[probe: tools/x.py, Jellyfin 10.11.11, DATE]`, `[source: file:line @ v10.11.11]`, or
  `[spec: operationId]`. "Jellyfin probably…" is forbidden. An unverified claim is marked
  `⚠️ UNVERIFIED` and blocks the spec from leaving draft.
- **III — Spec before implementation**, and documentation moves *in the same commit* as code.
- **IV — No forked code.** Jellyfin source is read for *what it does*, never translated. This is
  also the licence argument (GPL-3.0-or-later, [ADR-0005](docs/decisions/0005-licence.md)).
- **V — Bug-for-bug where clients depend on it.** The decision runs through the class A/B/C
  procedure in [behaviours §3.0](docs/compatibility/behaviours.md), not through intuition. Default
  is *replicate*; diverging needs a written argument.
- **VI — Implement what is actually called.** Jellyfin has 322 paths; v1 has 59. No endpoint
  without a named consumer, and no plausible-looking stub — an unimplemented endpoint answers what
  Jellyfin answers when a feature is absent, or is not routed at all.
- **VII — Determinism.** Identifiers derive from stable inputs, never insertion order, timestamps
  or randomness. A rescan must not invalidate client caches, favourites and resume positions.
- **VIII — Every behaviour ships with a conformance check at the HTTP boundary**, asserting on
  bytes. Casing, `null`-vs-absent and numeric type are invisible once parsed.
- **IX — English everywhere**, including commit messages and TODOs.

## The workflow

```
spec.md (WHAT/WHY, no technology) → plan.md (HOW) → tasks.md (STEPS) → code
         ↑___________ what implementation taught goes back into the spec, same commit ___________|
```

Each arrow is a **review gate**. A plan is not written against a draft spec; tasks are not written
against a draft plan; code is not written against draft tasks.

Statuses: `Draft` → `In review` → `Accepted` → `Implemented` (or `Superseded by NNN`). They live in
YAML front matter on every artefact, alongside `feature`, `created`, `updated`, `depends_on`, and an
`amended:` line recording what each later feature changed and why.

Templates for all three artefacts are in [specs/templates/](specs/templates/) — including the
tasks template's *definition of done*, which is the checklist a feature closes against.

### The status trap

**Every `status:` in `specs/` is a statement about the exporting Python project, not about this
one.** Eleven specs say `Implemented`; nothing is implemented here. PROVENANCE.md §"What the
receiving project must decide first" lists all twelve. The export deliberately did not rewrite
them — deciding what they mean now is this project's call, and it has not been made.

Treat `Implemented` as *"the WHAT is settled and was proven once, elsewhere"* — which is exactly
what makes these specs worth having — and never as *"this repository serves that route"*.

### Only `spec.md` survived

The 12 feature directories contain a `spec.md` and nothing else. `plan.md` and `tasks.md` were
withheld on purpose: *"HOW — inheriting it makes the second implementation a transliteration"*.
Writing them is the work, and they must be written in Go's terms, not Python's.

## Decisions this project still owes

Withheld because they belong to the receiving project:

- ~~**ADR-0006** password hashing. The number is reserved; `docs/decisions/README.md` still indexes
  it. Kept in the index because ADRs are numbered immutably and a gap would be a lie about the
  history.~~ **Taken on 2026-09-03**: [ADR-0006](docs/decisions/0006-password-hashing.md) decides
  Argon2id, its parameters, and the verification path that keeps an unknown username and a wrong
  password indistinguishable in time as well as in bytes. The index row already named the algorithm
  — it is exported bytes — and was deliberately not read as evidence; the second derivation reached
  it anyway, which is the experiment's own logic applied to an ADR.
- `tools/README.md`.

**~~Five~~ six of these have been taken, five on 2026-09-02 and ADR-0006 on 2026-09-03.** [ADR-0002](docs/decisions/0002-go-and-the-runtime-stack.md)
decides the runtime stack — Go, chi over `net/http`, `encoding/json` behind one wire package, and
no cgo — after measuring the three things that decide it rather than assuming them.
[docs/architecture.md](docs/architecture.md) is the project-level shape every `plan.md` inherits
from; **three** documents cite it (`specs/README.md`, the plan template, and ADR-0007 twice by
anchor), and this list said nine, which is `docs/roadmap.md`'s count one row above it in
PROVENANCE's table. [ADR-0003](docs/decisions/0003-sqlite-as-the-store.md) decides the store — embedded SQLite, pure
Go, hand-written SQL, split into a derived half a rescan rebuilds and a precious half that is
migrated. [docs/roadmap.md](docs/roadmap.md), [docs/README.md](docs/README.md) and
[AGENTS.md](AGENTS.md) are written too.

**Dangling links are intentional.** PROVENANCE.md §"Links with nothing to point at" enumerates
them: *"retargeting one is an edit to a specification, and that is this project's decision rather
than the export's."* Do not silently repoint or delete them.

## Leaks: 157 lines that name the Python implementation

The exported documents contain 157 lines naming a technology or pointing at a file that stayed
behind — every one enumerated with its line number in PROVENANCE.md §Leaks. Concentrated in
`behaviours.md` (27), the two client analyses (81), and `specs/README.md` (11).

When you meet `compat/responses.py`, `Starlette`, `pytest -m "not ffmpeg"` or `tests/unit/…` in
these documents: **the measured behaviour beside it is real and load-bearing; the file path is
not.** Read the finding, ignore the address. A citation of a *probe* under `tools/` is not a leak —
the probes measure the reference, they stay in the source repository, and this project points them
at its own server over HTTP rather than rewriting them.

## Conformance levels

| Level | Question | Needs a real Jellyfin? |
|---|---|---|
| **L0 — Routed** | Does the path exist and answer a sane status? | No |
| **L1 — Shape** | Are fields, casing, types and units right? | No |
| **L2 — Semantic** | Are the *values* right for a known library? | No |
| **L3 — Differential** | Is it byte-comparable to what Jellyfin actually sends? | Yes |

v1 requires **L2 everywhere**, **L3 on the authentication and playback paths**. `surface.yaml`
carries the declared level per endpoint: 1 × L1, 50 × L2, 8 × L3.

Two cross-cutting L1 sweeps apply to every route at once: every response field name is PascalCase,
every `*Ticks` field is an integer, every `*Date` field serialises with seven fractional digits and
a `Z`.

L3 needs both servers over one fixture, which is why [ADR-0007](docs/decisions/0007-a-container-runtime-for-the-reference-instance.md)
stands up a **single-use reference instance** from a digest-pinned image and destroys it with
everything it wrote — never an operator's own server (009's probes left 28 playlists on one).

## The machine-readable compatibility artefacts

These are the cross-checked heart of the repository. Each is paired with a prose table, and a test
compares the two row for row so they cannot drift apart — **edit both halves or neither.**

| File | Holds | Paired prose |
|---|---|---|
| [surface.yaml](docs/compatibility/surface.yaml) | The 59 v1 endpoints: `operation`, `consumers`, owning `feature`, required `level`. The router must expose exactly the implemented rows and nothing else (Principle VI, enforced in CI). | `api-surface-v1.md` |
| [allowlist.yaml](docs/compatibility/allowlist.yaml) | What may differ between the two servers, and why. Every entry is **scoped** to an endpoint + JSON Pointer (+ request case), and cites either a `behaviours §N` or one of four derivation classes — `derived-identifier`, `wall-clock`, `content-hash`, `installation-path`. An entry with neither fails the load. | `conformance.md` L3, `010 spec §3.3` |
| [request-cases.yaml](docs/compatibility/request-cases.yaml) | What a differential run actually sends, per endpoint **and per identity**. 12 of 23 reads answer differently to a restricted non-administrator, so a case naming one seat proves nothing. | `010 spec §3.2, §3.9` |
| [named-comparisons.yaml](docs/compatibility/named-comparisons.yaml) | The 20 differences a sweep cannot raise on its own. An unrun row keeps a run from being called clean. | `010 spec §3.10` |
| [property-names.json](docs/compatibility/property-names.json) | 1,026 property names extracted from the pinned OpenAPI document. Generated; never edit by hand. | — |
| [reference-fixture-reading.json](docs/compatibility/reference-fixture-reading.json) | The reference's recorded reading of the fixture tree. Compared against Atrium's own scan **as a declared inequality**: ~~47~~ **32** known differences, where an undeclared one fails *and a declared one that has gone away fails too*. **Struck 2026-09-05 at 003's closing audit.** Forty-seven is 010's D-7, counted against the *exporting* implementation over all **six** fixture libraries; this project builds **four** of them (`Films` and `Tunes` are 008's) and its scan declares thirty-two, with eight more predicted for the two unbuilt libraries and **seven not derivable from the recorded reading and this project's specifications at all** — the reading records four fields per item, so a difference in any other field is invisible however real it is. The seven are the experiment's own finding and must not be closed by inventing rows: see 003 tasks.md's *What this feature owes the next ones*. | — |

## Working in here

- **The exported bytes are committed history from another repository.** Amending a spec is a normal
  and expected act — every implemented feature amended its own spec repeatedly, and the `amended:`
  front-matter line is where that is recorded — but do it deliberately and say what forced it.
- **`spec.md` may not name a technology.** No Go package, no library, no table, no function. The
  moment a spec names one it has started deciding *how*, and the review that was supposed to be
  about *what* never happens. Everything technical goes in `plan.md`.
- **A claim without provenance is not a claim.** If you cannot cite a probe, a source line at
  `v10.11.11`, or the pinned OpenAPI document, mark it `⚠️ UNVERIFIED` rather than asserting it.
- **The reference is what a running Jellyfin does** — not its docs, not its OpenAPI schema. Where
  the three disagree, the running server wins and the disagreement gets recorded.
- **Read `behaviours.md` before specifying or implementing anything.** Its six sections are wire
  format, semantics, defects, deliberate exceptions, accepted gaps in v1, and non-improvements.
  Most questions that feel novel already have a measured answer in there.
- **The closing task of a feature is where the real findings come from.** Every implemented feature
  found, in its own final task, an acceptance criterion with no test or a test proving less than
  its name. Budget for that rather than treating the last task as a formality.
