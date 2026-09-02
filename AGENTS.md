# Working in this repository

**Rules for anyone working here — human or agent.** They are short because each one has cost
something. Where a rule looks arbitrary, the reason beside it is the record of what went wrong
without it.

[CLAUDE.md](CLAUDE.md) orients: what this repository is, what to read, and where the traps are.
**This file is normative.** Where the two overlap, the rule is here and the explanation is there.

---

## 1. The six that are absolute

### 1.1 Never read anything withheld from `../atrium-media-server`

This repository is the second, independent implementation of specifications a Python server already
implemented. **The experiment is only meaningful if the second implementation never sees the
first**, and a transliteration would answer nothing.

The boundary is not *"source code"*. It is the withheld/exported split
[PROVENANCE.md](PROVENANCE.md) already drew and hashed. `plan.md`, `tasks.md`, tests, audits,
`architecture.md` and `roadmap.md` contain no Python and contaminate just as much: they are the HOW
and the STEPS.

The 37 exported files here **are** the complete non-contaminating surface. There is nothing to gain
by looking further and the experiment's validity to lose.

What may be read: Jellyfin's source, as a behavioural reference, cited at a version tag; the two
client analyses already in `docs/compatibility/`. The sibling checkouts are otherwise closed.

### 1.2 Never cite a path outside this repository

A path in a repository the reader of this one cannot open is **neither verifiable by them nor ours
to publish**.

The test is whether a reader can open it, not whether you can:

| Citation | Allowed | Why |
|---|---|---|
| `[source: Emby.Server.Implementations/Library/LibraryManager.cs:636 @ v10.11.11]` | **Yes** | Public, and pinned to a version. Anyone can open exactly what you read. |
| A `file:line` in a first-party client | **No** | Private. Cite the client's own conformance document instead, as both analyses in `docs/compatibility/` do. |
| A path in `../atrium-media-server` | **No** | Withheld, and citing one means you opened it. |
| A path under `tools/` in the source repository | **In the exported documents only** | 157 leak lines already carry them. Read the finding, ignore the address; do not add more. |

The same policy is why [api-surface-v1.md §1](docs/compatibility/api-surface-v1.md#1-how-this-set-was-derived)
describes its two clients **by role** rather than by name. Whoever holds both documents can walk the
last step; whoever holds only this one loses nothing they could have checked anyway.

### 1.3 A claim without provenance is not a claim

Principle II. Every compatibility claim carries a probe, a source line at `v10.11.11`, or the pinned
OpenAPI document — the six marks are declared and ranked in
[docs/README.md §Conventions](docs/README.md#conventions).

*"Jellyfin probably…"* is forbidden. If you cannot cite one, write `⚠️ UNVERIFIED` and let it block
the specification from leaving draft. **Marking it is the correct move; asserting it is not.**

Where a probe, a source line and the OpenAPI document disagree, **the running server wins**, and the
disagreement is recorded rather than resolved silently.

### 1.4 Never commit to `main`

Every change goes on a branch, opens a pull request, and reaches `main` by merge. The root commit
was the only exception, because a pull request needs a base.

### 1.5 English in everything that lands here

Principle IX. Identifiers, comments, documentation, commit messages, branch names, pull request
titles and bodies. **This holds even when the conversation that produced the change was in another
language** — which is the case more often than not.

### 1.6 No CI job contacts a Jellyfin server, and none may start one

This predates [ADR-0007](docs/decisions/0007-a-container-runtime-for-the-reference-instance.md),
which relies on it: the differential run needs a real reference, and it is therefore **never**
automatic. The consequence is uncomfortable and true — *the strongest check in the project is the
one that is never automatic* — and it is stated rather than worked around.

Enforced, not promised: the test harness fails any test that opens a network connection
([architecture §8](docs/architecture.md#8-testing-and-conformance)).

---

## 2. Before you write a specification

**Read [behaviours.md](docs/compatibility/behaviours.md) first.** Most questions that feel novel
already have a measured answer in its six sections. This is the single highest-value habit in the
repository and the easiest to skip.

**`spec.md` may not name a technology.** No package, no library, no table, no function. The moment a
spec names one it has started deciding *how*, and the review that was supposed to be about *what*
never happens. Everything technical goes in `plan.md`.

**Run the probe before writing the plan, not after.** Every probe run at a spec gate returned more
than it was sent to check, and several killed claims that had already been written down. Running one
is the cheapest work available and it changes what the spec says.

**Test for a good spec:** two competent engineers could implement it in two different languages and
their servers would be indistinguishable to a client. That is not a metaphor here — it is the
experiment.

---

## 3. Before you write code

**Project-level choices are inherited, not re-decided.** [docs/architecture.md](docs/architecture.md)
and the [ADRs](docs/decisions/) are what a plan takes as given. A plan restates one only where it
deviates, and **a deviation without its own ADR is not a deviation, it is an inconsistency.**

**Documentation moves in the same commit as code** (Principle III). Not in a follow-up.

**Every behaviour ships with a conformance check at the HTTP boundary, asserting on bytes**
(Principle VIII). Casing, `null`-versus-absent and numeric type are all invisible once a body is
parsed, so a test that asserts on a parsed object cannot fail on any of them.

**A conformance assertion is a declared inequality, not an equality.** The two servers differ from
the reference in declared places, and the check that matters fails on an **undeclared** difference
*and on a declared one that has gone away*. An equality assertion would be either false or useless.

**No endpoint without a named consumer** (Principle VI), and no plausible-looking stub. An
unimplemented endpoint answers what Jellyfin answers when a feature is absent, or is not routed at
all.

---

## 4. Changing a document

**Amend in place, with the date and the reason.** A corrected claim is struck through and kept, not
quietly replaced. The amendment paragraphs in `behaviours.md` and `api-surface-v1.md` are the most
useful text in either file: they record what this project believed, what proved it wrong, and what
the wrong belief would have cost. **A document that absorbs its own amendments cannot be audited.**

**Paired files move together.** Several machine-readable artefacts have a prose twin and a test that
compares them row for row; the pairs are listed in
[docs/README.md](docs/README.md#paired-files-edit-both-halves-or-neither). Edit both halves or
neither.

**ADRs are immutable once accepted.** A wrong one is *superseded* by a new record that says so, and
the old one gets a `Superseded by` line. Never edited.

**Dangling links are intentional.** [PROVENANCE.md](PROVENANCE.md) enumerates them. Retargeting one
is a decision this project takes deliberately and says it is taking — as
[architecture §3](docs/architecture.md#3-repository-layout) does for the rule ADR-0007 cites from
it. Do not silently repoint or delete one.

**Do not edit the exported bytes to tidy them.** They are committed history from another repository.
Amending a specification is normal and expected — say what forced it, in the `amended:` front-matter
line.

---

## 5. Two traps that have caught people already

**Every `status:` in `specs/` describes the *exporting* project.** Eleven say `Implemented`; nothing
is implemented here. Read `Implemented` as *"the WHAT is settled and was proven once, elsewhere"* —
which is exactly what makes these specifications worth having — and never as *"this repository
serves that route"*.

**The closing task of a feature is where the real findings come from.** Every implemented feature
found, in its own final task, an acceptance criterion with no test or a test proving less than its
name. Budget for that rather than treating the last task as a formality.

---

## 6. Commits and pull requests

- **A branch per change**, a pull request per branch, English throughout.
- **Say what forced the change**, not only what changed. A commit message that lists files is a
  `diff` with extra steps.
- **Quote the measurement.** Where a change turns on something measured, put the measurement in the
  message. It is what lets a reviewer disagree with the conclusion rather than only with the
  wording.
- **The repository is public from its first commit.** Principle X applies to anything it says about
  Jellyfin: an independent implementation, unaffiliated and not endorsed.
