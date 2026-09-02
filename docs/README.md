# The documentation

Everything in this repository that is not a specification. Four kinds of document, and they do not
outrank each other the same way:

| | |
|---|---|
| [constitution.md](constitution.md) | **Ten principles that do not bend.** A conflict is resolved in favour of this file, or the file is amended first. |
| [decisions/](decisions/) | **ADRs.** One decision per file, numbered, immutable once accepted. A wrong one is *superseded*, never edited. |
| [architecture.md](architecture.md), [roadmap.md](roadmap.md) | **Project-level shape and order.** What every `plan.md` inherits, and what gets built when. |
| [compatibility/](compatibility/) | **Measured facts about Jellyfin.** Not opinions, and not decisions — the reference's observed behaviour, with provenance on every claim. |
| [glossary.md](glossary.md) | Jellyfin's vocabulary, and this project's own. |

## Read in this order

1. [constitution.md](constitution.md) — ten principles. Nothing below makes sense without I, II and
   VIII.
2. [compatibility/behaviours.md](compatibility/behaviours.md) — 3,215 lines of measured behaviour.
   **Read it before specifying or implementing anything.** Most questions that feel novel already
   have a measured answer in it.
3. [compatibility/api-surface-v1.md](compatibility/api-surface-v1.md) — the 59 endpoints, and the
   real clients that call each one.
4. [architecture.md](architecture.md) and [roadmap.md](roadmap.md) — the shape, and the order.
5. [compatibility/conformance.md](compatibility/conformance.md) — how a claim is proved: L0 to L3.

## Conventions

### Provenance marks

Principle II: **behaviour is measured, not assumed.** Every compatibility claim carries one of
these, and *"Jellyfin probably…"* is not among them.

| Mark | Means | Strength |
|---|---|---|
| `[probe: tools/probe_x.py, Jellyfin 10.11.11, 2026-08-28]` | A request was issued against a running reference and the answer was recorded. | **Strongest.** The reference is what a running Jellyfin does. |
| `[source: Path/To/File.cs:234 @ v10.11.11]` | A line of the reference's source, at the pinned version, read for *what it does*. | Strong for mechanism, weaker for behaviour — source says what should happen, a probe says what did. |
| `[spec: operationId]` | The pinned OpenAPI document declares it. | Weakest of the three. The document and the server disagree in places, and the server wins. |
| `[prior-probe: Jellyfin 10.11.11, 2026-06-13]` | A measurement this project made earlier and carried forward, without the probe surviving as a file. | Ranks below `probe`: real, but not re-runnable. |
| `[measurement: net/http.ServeMux, Go 1.27.0, 2026-09-02]` | A measurement of **this project's own tools**, not of the reference. | Used where a stack decision turns on what a library actually does. Never a claim about Jellyfin. |
| `⚠️ UNVERIFIED` | Nobody has measured this. | **Blocks a spec from leaving draft.** Marking it is the correct move; asserting it is not. |

Two further marks — `client-contract` and `client-author` — are declared locally by
[client-atrium-tvos.md §1](compatibility/client-atrium-tvos.md), because they are claims by a third
party about their own software rather than measurements of the reference.

**Where the three sources disagree, the running server wins**, and the disagreement gets recorded
rather than resolved silently.

### Paired files: edit both halves or neither

Several machine-readable artefacts have a prose twin, and a test compares them row for row so they
cannot drift:

| Machine-readable | Prose |
|---|---|
| [surface.yaml](compatibility/surface.yaml) | [api-surface-v1.md](compatibility/api-surface-v1.md) |
| [allowlist.yaml](compatibility/allowlist.yaml) | [conformance.md](compatibility/conformance.md) L3, and 010 §3.3 |
| [request-cases.yaml](compatibility/request-cases.yaml) | 010 §3.2, §3.9 |
| [named-comparisons.yaml](compatibility/named-comparisons.yaml) | 010 §3.10 |

[property-names.json](compatibility/property-names.json) and
[reference-fixture-reading.json](compatibility/reference-fixture-reading.json) are **generated**.
Never edit them by hand.

The same rule holds for two prose documents that split one decision:
[roadmap §"Out of scope, and why"](roadmap.md#out-of-scope-and-why) and
[api-surface §10](compatibility/api-surface-v1.md#10-deliberately-excluded-from-v1).

### Amendment, and why documents here carry their own history

A corrected claim is **struck through in place with the date and the reason**, not quietly replaced.
`behaviours.md` and `api-surface-v1.md` are full of these, and they are the most useful paragraphs
in either file: they record what this project believed, what measured it wrong, and what the wrong
belief would have cost. A document that absorbs its own amendments cannot be audited.

### Dangling links are intentional

Some links here point at files that do not exist. They were withheld at the export and
[PROVENANCE.md](../PROVENANCE.md) enumerates them: *"retargeting one is an edit to a specification,
and that is this project's decision rather than the export's."* Repointing one is a deliberate act
that says so — as [architecture §3](architecture.md#3-repository-layout) does for the rule ADR-0007
cites from it.

## Still owed

| | |
|---|---|
| **ADR-0003** — the store | Reserved and unwritten. [architecture §6](architecture.md#6-state-and-the-boundary-adr-0003-still-owes) gives it a boundary, not an answer. |
| **ADR-0006** — password hashing | Reserved and unwritten. |
| `tools/README.md` | Waiting for there to be a `tools/`. |
