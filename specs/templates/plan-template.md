---
feature: NNN-kebab-case-name
title: <Human-readable title> — implementation plan
status: Draft
created: YYYY-MM-DD
updated: YYYY-MM-DD
spec_status_required: Accepted
---

# NNN — Implementation plan

> **This document describes HOW.** It may not restate WHAT: the spec is the authority on behaviour,
> and a plan that repeats it will disagree with it eventually.

## 1. Approach

The shape of the solution in a few paragraphs, and the one or two decisions that were not obvious.

## 2. Inherited decisions

Project-level choices this plan takes as given, from
[../../docs/architecture.md](../../docs/architecture.md) and the
[ADRs](../../docs/decisions/). List only what this feature actually leans on.

| Decision | Source |
|---|---|

**Deviations:** none, or one row per deviation with a link to its own ADR. A deviation without an
ADR is not a deviation, it is an inconsistency.

## 3. Modules

| Module | Change | Responsibility |
|---|---|---|

New module boundaries, and why they fall where they do.

## 4. Data model

Tables, columns, indexes, constraints. Migrations, and whether they are reversible.

Say which columns exist to serve a *query pattern* rather than a fact — those are the ones a later
reader will otherwise try to normalise away.

## 5. Contracts

Interfaces between this feature and the rest of the system: the signatures, the invariants, and what
callers may assume.

## 6. Algorithms

Anything a reader would otherwise have to reverse-engineer from code: derivations, ordering rules,
normalisation, precedence between sources.

## 7. Failure handling

| Failure | Detection | Response | Recovery |
|---|---|---|---|

## 8. Testing strategy

How each acceptance criterion in the spec becomes a test, and where each conformance level is
implemented. Name the fixtures the feature needs.

## 9. Risks

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|

## 10. Alternatives considered

What else was on the table and why it lost. A plan with no alternatives is a plan nobody stress-tested.
