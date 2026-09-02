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
| [0003](0003-sqlite-as-the-default-store.md) | SQLite as the default store | Accepted |
| [0004](0004-pin-to-jellyfin-10-11.md) | Pin the reference to Jellyfin 10.11 | Accepted |
| [0005](0005-licence.md) | Licence — GPL-3.0-or-later | Accepted |
| [0006](0006-password-hashing.md) | Password hashing — Argon2id | Accepted |
| [0007](0007-a-container-runtime-for-the-reference-instance.md) | A container runtime for the reference instance | Accepted |
