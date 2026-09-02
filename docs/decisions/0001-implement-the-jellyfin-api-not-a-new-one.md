# ADR-0001 — Implement the Jellyfin API, not a new one

**Status:** Accepted · **Date:** 2026-08-26

## Context

Atrium is a media server built as a learning exercise, by an author who also maintains multi-server
clients driving Emby and Jellyfin. Three protocol strategies were available.

| Strategy | What it is | Cost to existing clients | Who benefits |
|---|---|---|---|
| **Own API** | A new protocol, plus a new client driver | Breaks the premise that Emby and Jellyfin speak one API: a third column in every verification matrix, one more common-vs-delta decision in every change | Only people who install this server |
| **Speak the existing API** | Our implementation, their protocol | **None.** No new driver, no new delta | Anyone with a client — and the server becomes a programmable test target |
| **Plugin inside their server** | Our endpoints, hosted by Jellyfin | One capability to detect, one fast path | People who install it into the server they already have |

The third was evaluated separately and rejected on grounds specific to the client product. The
choice here is between the first two.

## Decision

**Atrium implements the Jellyfin API.** It does not extend it, and does not offer an alternative
dialect alongside it. This is ratified as Principle I of the [constitution](../constitution.md),
which outranks every other consideration including correctness.

## Consequences

- Every Jellyfin client is a potential Atrium client with no work on the client side.
- The project gets a free, exhaustive test oracle: a real Jellyfin, which can be asked what the
  right answer is. This is what makes the differential harness possible.
- Design questions mostly stop being open. "What should this field be called?" has an answer we
  look up rather than debate. That is a **feature** for a learning project — it removes the freedom
  to design around difficulty, which is exactly where the learning is.
- Jellyfin's defects become our problem. Some get replicated deliberately
  ([Principle V](../constitution.md#v-bug-for-bug-where-clients-depend-on-it)).
- Growth is bounded by someone else's roadmap. Accepted.

## Alternatives rejected

**A clean, modern API of our own design.** More pleasant to write, and worthless: zero clients on
day one, and every client author who adopted it would carry a third protocol forever. The whole
value of this server is that it needs no adoption.

**Both — Jellyfin's API plus an "Atrium-native" one.** Superficially the best of both, actually the
worst: two contracts to keep correct, and the native one rots untested because every real client
uses the other. It also breaks Principle I the moment a client can detect which server it is on.
