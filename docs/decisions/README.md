# Architecture decision records

One decision per file, numbered, dated, and immutable once accepted. A decision that turns out to
be wrong is not edited — it is **superseded** by a new record that says so, and the old one gets a
`Superseded by` line. The history of what we believed and why is part of the documentation.

Format: context → decision → consequences → alternatives rejected. The alternatives section is not
optional; a decision without visible alternatives is a preference.

| # | Title | Status |
|---|---|---|
| [0001](0001-implement-the-jellyfin-api-not-a-new-one.md) | Implement the Jellyfin API, not a new one | Accepted |
| [0002](0002-go-and-the-runtime-stack.md) | Go, and the runtime stack | Accepted |
| [0003](0003-sqlite-as-the-store.md) | SQLite as the store | Accepted |
| [0004](0004-pin-to-jellyfin-10-11.md) | Pin the reference to Jellyfin 10.11 | Accepted |
| [0005](0005-licence.md) | Licence — GPL-3.0-or-later | Accepted |
| [0006](0006-password-hashing.md) | Password hashing — Argon2id | Accepted |
| [0007](0007-a-container-runtime-for-the-reference-instance.md) | A container runtime for the reference instance | Accepted |

**0006's row is unchanged, and that is a finding rather than an absence of work — 2026-09-03.** The
row is exported bytes: it named Argon2id before this project had decided anything, because the
exporting project decided it once already. [0006](0006-password-hashing.md) was derived here
independently — from ADR-0002's `CGO_ENABLED=0`, a measured table of four candidates on this
machine, and the deployment shape — without reading the row as evidence, and it reached the same
algorithm. So the row stands, now pointing at a record rather than at nothing; what the row never
said, and what this project decided for itself, is the parameters, the concurrency ceiling, the
storage format, the rehash rule and the verification path.
