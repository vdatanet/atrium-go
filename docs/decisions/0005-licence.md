# ADR-0005 — Licence

**Status:** Accepted · **Date:** 2026-08-26 · **Decided:** 2026-08-26

## Context

Atrium is to be open source. The licence is not a formality here, because of how the project is
built: Jellyfin's source is read as a behavioural reference (Principle II), and Jellyfin is
**GPL-2.0-or-later**.

Two things have to be kept apart, because conflating them produces bad advice in both directions.

**What is not at risk.** An API — the set of paths, field names, status codes and units — is an
interface, and reimplementing an interface independently is not creating a derivative work of the
implementation behind it. Observing that `RunTimeTicks` is an integer in units of 100 ns, and
writing your own code that emits that, does not make your code a derivative of Jellyfin's.

**What is at risk.** Reading GPL source and *translating* it — carrying over an algorithm's
structure, its identifier names, its file organisation — does create a derivative work, whatever
language the result is in. This is exactly the line Principle IV draws, and the reason it is a
principle rather than a style note.

So the licence question is really a question about how much confidence the project wants that the
line was never crossed.

## Decision

**GPL-3.0-or-later**, with two supporting practices:

1. **Principle IV enforced in review.** Every change that was informed by reading Jellyfin's source
   cites what it read and what it took — behaviour, not code. Reviewers check that.
2. **No vendored Jellyfin artefacts.** Not source, not assets, and not the generated OpenAPI
   document. Reference material is fetched into a git-ignored `reference/` directory at
   development time.

## Consequences

- **The lineage question stops being interesting.** With a compatible copyleft licence, the worst
  case of an accidental structural similarity is a licence-compliance situation the project is
  already in, not a violation.
- Contributions stay open, which fits a project whose stated purpose is didactic.
- **Cost:** GPL is a constraint on downstream users, including the author. A closed-source product
  cannot link Atrium's code. For a server that clients reach over HTTP this matters less than it
  first appears — the API boundary is not a linking boundary — but it is a real limit and should be
  chosen with eyes open, not by default.
- Copyright headers and a `LICENSE` file land in the first commit, not retrofitted.

## Alternatives

**MIT or Apache-2.0.** Maximally permissive, and defensible on the pure-interface argument above.
The cost is that it depends entirely on Principle IV having been followed perfectly, with no
fallback if it was not — and Principle IV depends on human discipline while reading a codebase in
another language. Apache-2.0 at least adds an explicit patent grant, which MIT lacks. Choose this
if the project may later feed permissively-licensed work, and accept a stricter review burden.

**AGPL-3.0.** The copyleft that actually bites for a network service: it closes the "run a modified
version as a service without publishing" gap, which is precisely the shape of a media server. The
strongest choice if the goal is that every deployed fork stays open. The cost is that it deters
some contributors and most commercial adoption.

**GPL-2.0-only**, matching Jellyfin exactly. Maximum compatibility with the reference project, at
the price of an older licence with no patent grant and no clear position on network use.

## How it was decided

The question put was: is the priority that Atrium's code stays open wherever it goes (→ AGPL-3.0),
that it is safe by construction given how it was written (→ GPL-3.0), or that it can be reused as
widely as possible (→ Apache-2.0)?

**Answer: safe by construction. GPL-3.0-or-later.**

The reasoning that follows from that answer, and that this record fixes:

- The project's method is to read GPL-2.0-or-later source as a behavioural reference. A compatible
  copyleft licence means the worst case of an accidental structural similarity is a compliance
  situation the project is already inside, not a violation. That is worth more here than the reach a
  permissive licence would buy.
- AGPL was not chosen because Atrium's whole premise is that it is *replaceable* — a client cannot
  tell it from Jellyfin, so anyone unhappy with the terms runs Jellyfin instead. AGPL's network
  clause would buy little and deter contributors on a project whose purpose is didactic.
- GPL-3.0 rather than GPL-2.0-only: a patent grant, an explicit position on tivoisation, and the
  `-or-later` suffix so the project is not pinned to a licence version forever. GPL-3.0 is one-way
  compatible with GPL-2.0-or-later material, which is the direction that matters here.

**Consequences now in force:**

- `LICENSE` at the repository root holds the verbatim GPL-3.0 text.
- Every source file carries an SPDX identifier — `# SPDX-License-Identifier: GPL-3.0-or-later` —
  from its first commit, not retrofitted.
- `pyproject.toml` declares `license = "GPL-3.0-or-later"`.
- Contributions are accepted under the same terms. No CLA: the licence is the agreement.
- Principle IV still binds. The licence is a backstop for a mistake, **not permission to
  translate Jellyfin's code**. A GPL-compatible licence does not make copying acceptable practice,
  and a change that carried over implementation would still be rejected in review.
