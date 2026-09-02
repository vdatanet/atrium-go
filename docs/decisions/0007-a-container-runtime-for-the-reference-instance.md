# ADR-0007 — A container runtime for the reference instance

**Status:** Accepted · **Date:** 2026-09-01

## Context

[010](../../specs/010-conformance-harness/spec.md) is the layer that can find what this project has
got wrong about the reference, and its strongest criterion — AC-2, *both servers pointed at the
same built fixture produce the same library* — needs a Jellyfin this project may configure.

The only Jellyfin this repository could reach until 2026-09-01 is an operator's own server, holding
an operator's own library, on another machine. Both halves of "point both servers at the same
fixture" are refused by that: the fixture tree is not on that machine, and adding a library to it
would be writing to data this project does not own. The evidence that this is not a theoretical
concern is on that server — 28 playlists left behind by 009's probe runs on 2026-09-01, under a
cleanup requirement 010 §3.5 states and `tools/README.md` repeats.

010's spec answered the question it could answer: OQ-5 decided that a run which needs the fixture on
both sides **stands up a single-use reference instance of the pinned version, uses it, and destroys
it with everything it wrote**, and it deliberately left *how* to the plan. This record takes that
half.

The choice is not "should the project depend on containers" in the abstract. It is: **what stands up
a configured Jellyfin of a pinned version, on a contributor's machine, without a human**, given that
the alternative already exists and is what this feature is trying to leave — asking each contributor
to install and configure a Jellyfin by hand, which produces a different reference on every machine
and no way to know that it did.

## Decision

**A container runtime — Docker or Podman — invoked as a subprocess through its command line.**

**Through the CLI, not through a Python SDK.** `tools/` is standard library only on a Python 3.9
floor ([architecture §3](../architecture.md#3-repository-layout), and 010's D-2, which kept it), so
the harness may not import a client library. It runs the runtime the way it runs `ffmpeg`: a
subprocess, arguments it composes, output it parses.

**A development-time dependency, and nothing else.** Nothing a user installs is affected: the
deployment shape stays one process and no second service
([architecture §5](../architecture.md#5-deployment-shape)). The runtime is needed by a tool, on the
machine of somebody running that tool.

**Never in CI.** No job may contact a Jellyfin server, and none may start one. That rule predates
this record ([AGENTS.md](../../AGENTS.md), `tests/conftest.py`'s socket guard) and this decision
does not weaken it: the consequence — *the strongest check in the project is the one that is never
automatic* — is stated in 010's plan §6.11 rather than worked around here.

**The image is pinned by digest**, not by tag, and the digest is recorded in
[reference-target §1](../compatibility/reference-target.md#1-the-pinned-version) beside the two
version rows it already pins. A tag moves; a digest is the version this project measured. The run
prints the digest in the report header beside the Atrium sha, so a difference that reproduces on one
machine only can be told from a difference that is real.

**The instance is single use.** It is created by the run, given the fixture tree read-only as its
only library, configured over the reference's own first-time-setup operations
`[spec: UpdateInitialConfiguration, UpdateStartupUser, CompleteWizard, AddVirtualFolder]`, and
destroyed on the success path and the exception path alike, together with everything it wrote. A run
that finds the wreckage of a killed one destroys it first.

**And it degrades rather than fails.** `--reference-url` accepts an instance somebody else stood up
— an operator's server, or a container a contributor started by hand — and a machine with no runtime
at all still runs the sweep against a reachable server and everything in the default CI job. What it
does **not** do is report those runs as complete: every case and every named comparison that
declared `needs: fixture` is reported **outstanding with the reason**, and the run is not clean. The
dependency buys coverage; its absence costs coverage and says so.

## Consequences

- **A second thing to install**, on the machines that want the fixture rows. It is the common
  development tool on all three platforms this project is developed on, and the rows that need it
  are the rows that were unreachable before it.
- **The mount is load-bearing, and it is why this is a bind mount rather than a copy.**
  `tests/fixtures/library/generate.py` stamps every file with one fixed modification time so that
  the same tree is the same tree to a change signal. A copy that did not preserve times would put a
  difference into `DateCreated` on every item — a field the allowlist excuses, which is worse than a
  visible failure, because the noise would be invisible.
- **The fixture root is mounted read-only**, so neither server can change the thing both are
  measured against.
- **Two prior-measurement debts become answerable** that an operator's server cannot answer at all:
  `/Users/Public` returning `[]` needs every user hidden, and the `LocalAddress` HTTPS override needs
  a server configured for HTTPS (behaviours §2.2, §2.3). Both are writes to a configuration, which is
  what a disposable instance is for.
- **A contributor's machine can now hold a Jellyfin,** which is exactly what the no-network rule in
  the suite exists to keep out of the default run. Nothing changes there: the harness lives in
  `tools/`, the suite fails any test that opens a TCP connection, and the mutation proofs of 010 §6
  run on checked-in pairs.
- **Running the reference's published server is reading the reference, not forking it** (Principle
  IV). Nothing of it is copied into this repository; it is started, asked questions, and destroyed.
- **The unattended configuration sequence is read from a document, not measured**, and the task that
  writes it verifies it against the first instance. If a credential is required earlier than the
  declared policies read, the sequence gains a step — the wizard creates the administrator either
  way.

## Alternatives rejected

**A Python SDK for the runtime.** It would make the lifecycle code shorter and it costs the rule
that makes `tools/` runnable before an environment exists — standard library only, Python 3.9. The
subprocess boundary is also the one that survives a change of runtime: Docker and Podman accept the
same arguments for what this needs, and an SDK does not.

**Ask each contributor to install and configure a Jellyfin by hand.** This is the state the feature
exists to leave. It produces a different reference on every machine — a different version, a
different library, a different set of plugins, which is not hypothetical: the reference server's
OpenAPI document carries two paths that come from its plugins
`[probe: /Plugins and /api-docs/openapi.json, Jellyfin 10.11.11, 2026-09-01]` — and it gives a run no
way to know which of those it measured.

**A virtual machine, or a full orchestration tool.** More to install, slower to start and stop, and
nothing here needs more than one container with one mount and one port. The instance lives for the
length of one comparison.

**Keep using the operator's server, with a fixture library added beside the real one.** Rejected on
2026-09-01 with OQ-5 and rejected again here: it means writing to data this project does not own, and
the writing probes had already left 28 playlists behind.

**Keep an instance alive between runs, to save the scan.** A surviving instance accumulates what each
run wrote, so the second run measures a library the first one changed. The property that makes the
fixture comparison mean anything is that the fixture is the only library either server has ever seen.

**Run the instance in CI, so AC-2 gates every change.** Rejected, and not on cost. A gate whose
result depends on pulling somebody else's image is not a gate, and *"no CI job contacts a Jellyfin
server"* is a rule this project enforces in `tests/conftest.py` rather than promises. The mechanism
that makes the run happen at the moment it matters most is `tools/bump_reference_version.py`, which
refuses to advance a version bump past a step that did not run.
