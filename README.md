# Atrium

An independent implementation of the **Jellyfin API**, written in Go from specifications alone.

> **Status: there is no code here yet, and that is deliberate.**
>
> This repository currently holds twelve specifications, a constitution, seven architecture
> decision records and a set of measured compatibility documents. Nothing is implemented. No
> compatibility claim on this page has been measured against this server, because this server does
> not exist yet.

## What this is

Atrium speaks Jellyfin's API rather than one of its own. A client that works against Jellyfin
should work against Atrium **with no branch, no capability probe and no configuration** — and if it
needs to know which server it is talking to, the project has failed, however good the reason was.
That is [Principle I](docs/constitution.md), and it outranks everything else here, including
correctness.

The API is not extended, improved, or offered with a better alternative alongside it. Ideas that
would genuinely be nicer done differently are written down as
[deliberate non-improvements](docs/compatibility/behaviours.md) and then not done.

## The experiment

**These specifications have already produced a working server once, in Python.** This repository is
the second, independent run of the same documents — in Go, from zero.

The question it asks: *can one set of specifications produce two indistinguishable servers on two
different platforms?*

For that to mean anything, the second implementation must never see the first. So the
specifications came across and **the implementation did not**: 513 files were withheld, including
not just the source and its tests, but every `plan.md` and `tasks.md` — the HOW and the STEPS.
Those contain no Python at all and would contaminate the experiment just as thoroughly, because
inheriting them turns a second implementation into a transliteration. What was exported, what was
withheld, and why, is recorded in [PROVENANCE.md](PROVENANCE.md).

The result will not be measured as an equality. The Python implementation does not produce a
library identical to Jellyfin's either — pointed at the same fixture, the two differ in
forty-seven declared places. What gets compared is the **set of declared differences**: same
divergences, same reasons, same owners means the specifications determined the behaviour. Every
asymmetry is a place where they did not, and that is the finding.

## What "compatible" means here

Precisely, because the claim is not testable otherwise:

| | |
|---|---|
| API contract | Jellyfin `10.11.11` OpenAPI document |
| Behavioural reference | Jellyfin `10.11.11` source, and a running instance |
| Surface | **59 endpoints** of Jellyfin's 322 paths |

The 59 were not chosen by reading documentation and picking what looked important. They were
extracted from the source of two production clients — a multiplatform music client and a tvOS video
client — by taking what those clients actually call. An endpoint with no named consumer has to
justify itself on other grounds. See [api-surface-v1.md](docs/compatibility/api-surface-v1.md).

**The reference is not the documentation and not the schema — it is what a running Jellyfin
actually does.** Every compatibility claim in this repository traces to a probe run against a real
server, a cited source line at a version tag, or the pinned OpenAPI document. A claim without one
is marked `⚠️ UNVERIFIED` and blocks its specification from leaving draft. There is exactly one
such marker left in the compatibility documents, and it has an owner.

Behaviours are proven at four levels, from *the route exists* to *the bytes match a real
Jellyfin's, modulo a documented allowlist*. Defined in
[conformance.md](docs/compatibility/conformance.md).

## Jellyfin's bugs are part of the interface

Some Jellyfin behaviours are defects, and clients have already built workarounds for them. A server
that quietly does the right thing breaks those workarounds — so being correct is not automatically
safe.

Each one is decided by a written procedure that turns on a single question: *can a client have
built something that being correct would break?* The default is to **replicate the defect**, and
diverging requires an argument that no client can observe the difference. Every decision is
recorded with what Jellyfin does, whether a known client depends on it, and what Atrium does. That
register is [behaviours.md](docs/compatibility/behaviours.md), and it is the most useful document
here.

## How it is built

Specification-driven, in a strict order, with a review gate at each arrow:

```
spec.md  (WHAT and WHY — no technology)
   ↓
plan.md  (HOW — architecture, stack, data model)
   ↓
tasks.md (verifiable steps, each with its acceptance check)
   ↓
code
```

The loop closes: what implementation teaches goes back into the specification **in the same
commit**, not in a follow-up. The workflow is [specs/README.md](specs/README.md); the principles
that do not bend are the [constitution](docs/constitution.md).

## Where to start reading

| | |
|---|---|
| [docs/constitution.md](docs/constitution.md) | Ten principles, each stating what it forbids |
| [specs/](specs/) | Twelve feature specifications — WHAT and WHY, no technology |
| [docs/compatibility/behaviours.md](docs/compatibility/behaviours.md) | Every measured behaviour, replicated defect and deliberate divergence |
| [docs/decisions/](docs/decisions/) | Architecture decision records |
| [PROVENANCE.md](PROVENANCE.md) | Where these documents came from and what did not come with them |

**A caution for readers.** Every `status:` line in `specs/` describes the *exporting* project.
Eleven say `Implemented`. Nothing is implemented here.

## Not yet decided

The runtime stack, the store and the password hashing scheme are open. Their record numbers —
`0002`, `0003` and `0006` — are reserved and deliberately absent: they were withheld as decisions
this project takes for itself, and no implementation plan can be written until they are taken.

## Relationship to Jellyfin

Atrium is an **independent implementation of Jellyfin's API. It is not affiliated with, endorsed by,
or derived from the Jellyfin project**, and it is not a fork, a plugin or a successor.

Jellyfin's source is read the way one reads a specification — to learn what it does — and never as
a source of code to translate. No Jellyfin source, assets or generated documents are vendored here.
The distinction the project holds to is between **interface** and **implementation**: the interface
is reimplemented, the implementation never carried over. Jellyfin is a trademark of its owners and
is used here only to describe what this software is compatible with.

## Licence

[GPL-3.0-or-later](LICENSE). The choice is reasoned in
[ADR-0005](docs/decisions/0005-licence.md): reimplementing an interface does not create a
derivative work, but reading GPL source is close enough to the line that a compatible copyleft
licence makes the question uninteresting.
