---
feature: NNN-kebab-case-name
title: <Human-readable title>
status: Draft            # Draft | In review | Accepted | Implemented | Superseded by NNN
created: YYYY-MM-DD
updated: YYYY-MM-DD
depends_on: []           # feature numbers
---

# NNN — <Title>

> **This document describes WHAT and WHY only.** No technology names. No storage decisions. No
> module or function names. If you need to write one, it belongs in `plan.md`.

## 1. Purpose

Two or three sentences. What capability does this add, and which client behaviour does it unlock?
If you cannot name a client behaviour, question whether the feature belongs in v1 (Principle VI).

## 2. Scope

**In scope:**
-

**Out of scope:**
- <and where it is handled instead, if anywhere>

## 3. Behaviour

The substance. One subsection per endpoint or per observable behaviour.

### 3.1 `<METHOD> /<Path>` — `<operationId>`

**Consumers:** <which real clients call this, from api-surface-v1.md>

**Request**

| Part | Name | Required | Type | Notes |
|---|---|---|---|---|
| path / query / header / body | | | | |

**Response — 200**

```json
{ }
```

| Field | Type | Notes |
|---|---|---|

**Error responses**

| Condition | Status | Body |
|---|---|---|

**Compatibility notes**

Anything where Jellyfin's behaviour is surprising, with provenance:
`[probe: …]` / `[source: file:line @ tag]` / `[spec: operationId]`

## 4. Data the feature owns

*Observable* state only — what a client can see change, and what survives a restart. Not tables,
not schemas.

## 5. Acceptance criteria

Numbered, each independently checkable. Written so a reviewer can tell whether it passed without
reading the implementation.

1.

## 6. Conformance

| Endpoint | Level | How it is proven |
|---|---|---|
| | L0/L1/L2/L3 | |

Levels are defined in [../../docs/compatibility/conformance.md](../../docs/compatibility/conformance.md).

## 7. Open questions

Each with what it blocks and what would resolve it. An open question that blocks nothing is a note,
not a question.

| # | Question | Blocks | Resolved by |
|---|---|---|---|

## 8. References

- Provenance for every claim above.
