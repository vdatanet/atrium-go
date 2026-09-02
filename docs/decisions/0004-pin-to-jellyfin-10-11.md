# ADR-0004 — Pin the reference to Jellyfin 10.11

**Status:** Accepted · **Date:** 2026-08-26

## Context

"Compatible with Jellyfin" is not a testable statement until a version is named. Jellyfin's API has
changed in observable ways between lines — `10.11` removed the user-scoped item routes
(`/Users/{userId}/Items/{itemId}`) that Emby still serves and that older clients still call, and
`master` (the 12.0.0 line) carries further changes.

Available anchors at the time of writing:
- **10.11.x** — current stable. What clients ship against today.
- **master / 12.0.0** — moving, unreleased, no client targets it.
- **10.10.x** — previous stable, already superseded.

## Decision

**Pin to `10.11.x`.**

| | Value |
|---|---|
| API contract | `10.11.10` OpenAPI document |
| Behavioural reference | `10.11.11` source and a running instance |
| Version Atrium reports | `10.11.11` |

Consequences of the pin are worked out in
[compatibility/reference-target.md](../compatibility/reference-target.md).

## Consequences

- Atrium serves the **new** routes (`/Items/{itemId}`, `/UserFavoriteItems/{itemId}`) and **not**
  the user-scoped legacy forms. A client written for Emby's dialect will not work, and that is
  correct: Atrium is Jellyfin-shaped, and multi-server clients already branch there.
- The pin is a *floor for correctness*, not a promise of eternity. Moving it is a deliberate act
  with a fixed procedure — fetch, validate, run the differential harness, triage every new
  difference, re-run the probes, then update the version. Defined in
  [conformance.md](../compatibility/conformance.md#when-the-reference-version-moves).
- Behaviours fixed upstream after 10.11 but not backported stay in
  [behaviours.md](../compatibility/behaviours.md) with their upstream reference, so it is visible
  which of our decisions are dated rather than principled.
- The reported version string is part of the contract, not decoration: clients gate capabilities on
  it. It moves only when the pin moves.

## Alternatives rejected

**Track `master`.** The reference would move under us, and no client ships against it. Chasing an
unreleased target would spend the project's effort on differences no user will ever see.

**Support a version range.** Multiplies every conformance test by the number of versions, for a
compatibility guarantee nobody asked for. One version, moved deliberately.

**No pin — "compatible with Jellyfin" generally.** Untestable, and therefore a claim rather than a
property. Principle II forbids it.
