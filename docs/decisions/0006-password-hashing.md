# ADR-0006 — Password hashing

**Status:** Accepted · **Date:** 2026-09-03

## Context

This number was reserved rather than exported, for the same reason as
[ADR-0002](0002-go-and-the-runtime-stack.md) and [ADR-0003](0003-sqlite-as-the-store.md): it is a
decision the receiving project takes for itself ([PROVENANCE.md](../../PROVENANCE.md)). It is the
last of the three, and the index row has pointed at nothing since the export — the single entry
under `docs/decisions/0006-password-hashing.md` in PROVENANCE's *"Links with nothing to point at"*.

**The index row is not the decision, and was not read as one.** `docs/decisions/README.md` has
carried *"Password hashing — Argon2id | Accepted"* since the export: exported bytes from a project
that took this decision once already, in another language, on another stack. Treating it as the
answer would be the transliteration this repository exists to avoid, on the one artefact where it
would be easiest to get away with. So the candidates were weighed and measured here, and the row
was checked afterwards rather than followed. It survives, which is a result about the
specifications rather than a coincidence — see *Consequences*.

### What makes this decision unusually free

**No password hash ever reaches the wire, and no client can observe the choice.**
[002 §2](../../specs/002-authentication-users-and-sessions/spec.md#2-scope) puts creating, editing
and deleting users over HTTP out of scope — `POST /Users/New`, `/Users/Password` and
`/Users/{userId}/Policy` are not in v1, and v1 manages accounts through configuration. What a
client sees of a credential is a boolean, `HasPassword`, and whether authentication succeeds
([002 §4](../../specs/002-authentication-users-and-sessions/spec.md#4-data-the-feature-owns)).

That is worth stating against the strongest check the project has rather than the weakest.
`POST /Users/AuthenticateByName` is one of the eight rows [surface.yaml](../compatibility/surface.yaml)
declares at **L3** — byte-comparable to what the reference actually sends — and not one byte a
differential run compares there is a function of how the password was stored.

So **Principle I does not bind this record.** Zero delta is a constraint on the bytes a client
receives, and there are none here. This is the rare project-level choice that is an engineering
question and not a compatibility one, and it should be said plainly rather than discovered later by
somebody looking for the compatibility argument that does not exist.

### What does bind it

**ADR-0002 chose `CGO_ENABLED=0`**, and that constraint is load-bearing rather than incidental.
Every candidate below had to be available in pure Go, which rules out binding `libargon2`,
`libsodium` or any of the C implementations that would otherwise be the obvious way to get a
well-audited primitive. It also rules out the shape where hashing quietly becomes the one thing
that ends static builds and cross-compilation, which is exactly how a `CGO_ENABLED=0` decision
usually dies.

**ADR-0002 also declined to pre-approve dependencies**: *"A further dependency is argued where it
is needed, in the plan that needs it, not pre-approved here."* One candidate needs no dependency at
all — `crypto/pbkdf2` has been in the standard library since Go 1.24, so it is reachable at this
module's Go 1.25 floor — and the other three all live in `golang.org/x/crypto`. That is a real
difference between them and it is priced below.

**ADR-0003 decided where the record lives**: the **precious half** of the store, the half that is
migrated forward and never rebuilt, because a credential is not reconstructible from anything on
disk.

**And 002 constrains the verification path more than the algorithm.**
[§3.3](../../specs/002-authentication-users-and-sessions/spec.md#33-post-usersauthenticatebyname--authenticateuserbyname)
answers an **unknown username** and a **wrong password on an enabled account** with the *same*
status and the *same* 25 bytes, while a **disabled account** is `403` whichever password it was
sent
([behaviours §2.11](../compatibility/behaviours.md#211-a-disabled-account-is-refused-with-403-not-401)).
The reference therefore discloses that an account exists and is disabled, and discloses nothing
about the password. Atrium replicates that — and a hashing scheme that lets a stopwatch separate
the two `401`s would hand back the disclosure the identical bytes were there to withhold.

### What was measured, on 2026-09-03

Four candidates, each in the pure-Go implementation this project would actually use, on Go 1.27.0
`darwin/arm64`, 10 cores. Median of nine derivations of a 28-byte password with a 16-byte salt.
The benchmark is `tools/bench_password_hashing` and it is kept, for the reason given under
*Decision*.

| Candidate | Working memory | Median | Dependency |
|---|---|---|---|
| Argon2id `m=19 MiB, t=2, p=1` | 19 MiB | 20.7 ms | `golang.org/x/crypto` |
| Argon2id `m=46 MiB, t=1, p=1` | 46 MiB | 22.0 ms | `golang.org/x/crypto` |
| Argon2id `m=64 MiB, t=2, p=2` | 64 MiB | 34.2 ms | `golang.org/x/crypto` |
| **Argon2id `m=64 MiB, t=3, p=2`** | **64 MiB** | **52.4 ms** | `golang.org/x/crypto` |
| Argon2id `m=256 MiB, t=3, p=2` | 256 MiB | 213.1 ms | `golang.org/x/crypto` |
| scrypt `N=2^15, r=8, p=1` | 32 MiB | 37.2 ms | `golang.org/x/crypto` |
| scrypt `N=2^16, r=8, p=1` | 64 MiB | 74.5 ms | `golang.org/x/crypto` |
| bcrypt `cost=10` | 4 KiB | 45.1 ms | `golang.org/x/crypto` |
| bcrypt `cost=12` | 4 KiB | 179.6 ms | `golang.org/x/crypto` |
| PBKDF2-SHA512, 210,000 iterations | negligible | 43.0 ms | **none** — `crypto/pbkdf2` |
| PBKDF2-SHA512, 600,000 iterations | negligible | 122.7 ms | **none** — `crypto/pbkdf2` |

`[measurement: golang.org/x/crypto v0.55.0 and crypto/pbkdf2, Go 1.27.0, 2026-09-03]`

**Read the table by column, not by row.** Every candidate can be tuned to any wall-clock the server
is willing to spend; what separates them is what the attacker has to buy to match it. The memory
column is that number, and it spans four orders of magnitude at a defender cost that varies by less
than three.

The reference's own scheme is in that table on purpose. It is **PBKDF2-SHA512, 210,000 iterations,
a 16-byte salt and a 64-byte output**
`[source: MediaBrowser.Model/Cryptography/Constants.cs:11-21 @ v10.11.11]`
`[source: Emby.Server.Implementations/Cryptography/CryptographyProvider.cs:16-35 @ v10.11.11]`,
and it costs this machine 43.0 ms — within 20% of the row chosen below, for a per-guess memory
footprint of nothing.

Two further things were measured rather than assumed.

**A memory-hard hash on an unauthenticated route is a memory lever, and the scaling is exactly
linear.**

```
 4 simultaneous Argon2id m=64 MiB t=3 p=2:  102.0 ms wall, peak live heap  256 MiB
 8 simultaneous Argon2id m=64 MiB t=3 p=2:  186.9 ms wall, peak live heap  512 MiB
16 simultaneous Argon2id m=64 MiB t=3 p=2:  450.5 ms wall, peak live heap 1024 MiB
```

`[measurement: golang.org/x/crypto/argon2 v0.55.0, Go 1.27.0, 2026-09-03]`

There is no sub-linearity to hope for: `n` derivations hold `n × m` bytes live. On the deployment
shape this project targets — one process sharing a host with SQLite and with ffmpeg
([architecture §5](../architecture.md#5-deployment-shape), [§7](../architecture.md#7-external-processes))
— that is the number that decides the parameters, and it is why the decision below is a pair
*(parameters, concurrency ceiling)* rather than a parameter set.

**Go's bcrypt refuses a long passphrase; it does not truncate it.**

```
bcrypt: a 72-byte passphrase is accepted
bcrypt: a 73-byte passphrase is refused: bcrypt: password length exceeds 72 bytes
```

`[measurement: golang.org/x/crypto/bcrypt v0.55.0, Go 1.27.0, 2026-09-03]`

This was measured because the usual objection to bcrypt is *silent truncation* — two different
passphrases sharing a hash. The Go implementation does something else, and it is worse for this
project rather than better: a user whose passphrase is 73 bytes cannot have an account at all,
and the failure surfaces at provisioning time with an error no operator expects.

## Decision

### Argon2id, `m = 64 MiB`, `t = 3`, `p = 2`, 16-byte salt, 32-byte key

Argon2id because at an equal 40–55 ms of this server's time it forces the largest attacker
footprint of the four — 64 MiB per guess against bcrypt's 4 KiB and PBKDF2's nothing — and because
it is the only candidate whose **time and memory raise independently**. That second property is not
a nicety here: the measurement above says memory is the constrained axis on a small host, so the
knob that must exist is the one that makes a verification *slower without making it bigger*.
`t` is that knob, and scrypt does not have it.

The parameters are one step above the OWASP-minimum row in the table rather than the maximum this
machine can carry, because the ceiling below is what the deployment shape has to afford, and
`m=256 MiB` puts that ceiling at a gigabyte.

**They are constants in the code, not a setting.** [architecture §9](../architecture.md#9-configuration-identity-and-logging)
keeps process settings few and says they are not a feature; an operator who lowers these silently
weakens every credential written afterwards, and the record carries its own parameters anyway
(below), so raising them needs no setting either.

### At most four verifications in flight, and the rest wait rather than being refused

The ceiling is 4, which the measurement prices at **256 MiB transient and about 77 verifications a
second** — far above any real login rate for a media server whose sessions survive the app closing.

**Requests over the ceiling queue; they are never refused.** This is the one place where the
absence of a compatibility constraint on the *algorithm* does not extend to the *behaviour around
it*: a `503` on `POST /Users/AuthenticateByName` is a status the reference does not send there, and
Principle I does bind that. Latency is not a wire delta and a new status code is, so a flood makes
Atrium slow rather than makes Atrium wrong. The queue is bounded by the server's own connection
handling and by nothing this record adds.

### The record is a self-describing string, and the parameters travel with it

```
$argon2id$v=19$m=65536,t=3,p=2$<base64 salt>$<base64 key>
```

Standard PHC encoding, unpadded base64, one column in the precious half of the store. **The
parameters are read out of the record at verification time, not out of the constants**, which is
the whole mechanism by which they can be raised later: an existing credential written at
`t=3` keeps verifying after the constant becomes `t=4`, with no migration and no schema change —
a parameter raise touches [ADR-0003](0003-sqlite-as-the-store.md)'s migrated half without being a
migration.

The reference stores its own hashes exactly this way, in its own dialect — `$PBKDF2-SHA512$iterations=210000$<hex salt>$<hex hash>` —
and uses it to keep verifying legacy `PBKDF2`-over-SHA1 records written by older versions
`[source: MediaBrowser.Model/Cryptography/PasswordHash.cs:169-209 @ v10.11.11]`
`[source: Emby.Server.Implementations/Cryptography/CryptographyProvider.cs:38-63 @ v10.11.11]`.
That is a mechanism read at the source and adopted for what it does, not a format copied: the
encoding here is PHC's, which every other Argon2 implementation reads, so a credential this project
wrote is not a credential only this project can read.

### A raise takes effect on the next successful login, and only then

When a verification succeeds against a record whose parameters are below the current constants, the
password is re-derived with the current ones and the record replaced. **A successful login is the
only moment the plaintext exists**, so it is the only moment a rehash is possible; there is no
background job that could do this, and pretending otherwise would be the kind of promise a
`tasks.md` discovers is impossible.

The consequence is stated rather than hidden: **a user who never logs in keeps the old parameters
for ever.** That is not a regression against any alternative — the alternative to rehash-on-login
is not rehashing at all — and it means a raise is a slow migration measured in logins, not a
deployment step.

### The verification path runs the same work whether or not the account exists

Four rules, and each answers a specific disclosure:

1. **A username that matches no account is verified against a decoy record anyway**, and the result
   discarded. Without it the unknown-username `401` returns in microseconds and the
   wrong-password `401` returns in 52 ms, and two refusals that
   [002 §3.3](../../specs/002-authentication-users-and-sessions/spec.md#33-post-usersauthenticatebyname--authenticateuserbyname)
   made byte-identical become distinguishable with a stopwatch.
2. **The decoy is derived at startup from a random secret using the current constants**, never a
   literal in the source. A decoy pinned to old parameters would become its own oracle the moment
   the constants were raised: the account that does not exist would answer faster than one whose
   record has been rehashed.
3. **The derived key is compared with `crypto/subtle.ConstantTimeCompare`.** The reference compares
   with `SequenceEqual`, which short-circuits
   `[source: Emby.Server.Implementations/Cryptography/CryptographyProvider.cs:42 @ v10.11.11]`.
   Diverging here is free — the comparison is not a byte on the wire — and it is recorded so that
   nobody later reads the reference's code and assumes this project matched it.
4. **An account with no password is not equalised, deliberately.** `Pw` may be empty when the
   account has no password, and skipping the derivation there is observable in time — but
   `HasPassword` is already sent for every user on the unauthenticated `GET /Users/Public`, in a
   complete user object
   ([002 §3.4](../../specs/002-authentication-users-and-sessions/spec.md#34-get-userspublic--getpublicusers),
   [behaviours §3.5](../compatibility/behaviours.md)). Equalising a fact the reference publishes on
   an open route would be work that protects nothing.

The refusals that come *before* any derivation — a disabled account, a locked-out account — are
already disclosed by their status being `403` rather than `401`, so no equalisation is owed there
either. What must never differ is `401` from `401`.

### The plaintext is a distinct type that cannot be logged

[002 AC-11](../../specs/002-authentication-users-and-sessions/spec.md#5-acceptance-criteria)
requires that a password never appears in any log record at any level and never in an error body.
A `string` in a struct satisfies that only until somebody logs the struct, so the plaintext is
carried in its own type whose `slog.LogValue` and `String` return a redaction, and it is never
stored in a field of anything that is logged whole.

### `golang.org/x/crypto`, pinned at v0.55.0

This is the dependency ADR-0002 asked to be argued rather than assumed, and it is the third in the
module after chi and the SQLite driver. It is pure Go, it carries no transitive dependency this
module does not already have, and it is where three of the four candidates live — so choosing any
of them but PBKDF2 adds it.

**v0.55.0 and not the current v0.56.0**, because v0.56.0 declares `go 1.26.0` and would drag this
module's floor up with it. ADR-0002 put the floor at 1.25 and said it *"moves in its own commit,
with a reason"*; a floor that moves as a side effect of adding a dependency is exactly the shape
that rule exists to prevent.
`[measurement: golang.org/x/crypto module graph, Go 1.27.0, 2026-09-03]`

### The benchmark is kept

`tools/bench_password_hashing` stays in the repository. ADR-0002 and ADR-0003 measured and did not
keep their measurement code, and the difference is that those records chose *a component*, once,
while this one chooses *a number that is a property of the hardware*. Raising the parameters is a
foreseeable future change, and the right way to take it is to run the benchmark on the host in
question rather than to guess again from a table dated 2026-09-03. It contacts nothing, it is
imported by nothing, and it is the only executable artefact this record adds.

## Consequences

- **A login costs about 52 ms of CPU and 64 MiB of transient memory on this machine**, and the
  server must be able to spend that up to four times at once. On the smallest host this project
  targets, 256 MiB is a real reservation sitting beside SQLite's page cache and whatever ffmpeg is
  doing.
- **The login route is a memory lever an unauthenticated caller can pull**, and rule 1 above makes
  it worse rather than better: the decoy verification means an attacker does not even need a valid
  username to make the server allocate. The ceiling of four is the whole mitigation, and it is why
  the ceiling is part of the decision rather than a tuning detail.
- **Credentials are the precious half.** [ADR-0003](0003-sqlite-as-the-store.md)'s derived half can
  be dropped and rescanned; this column cannot be reconstructed from anything, which is what makes
  it worth 64 MiB a guess in the first place.
- **A parameter raise is a code change plus a slow migration measured in logins.** Nothing breaks
  at the moment of the raise, which is the property the self-describing record buys, and nothing
  finishes at that moment either.
- **This record constrains provisioning without specifying it.** However
  [002](../../specs/002-authentication-users-and-sessions/spec.md) decides an account gets a
  password — v1 manages accounts through configuration, not an admin API — what lands in the data
  directory is a PHC record, and a plaintext password is never written anywhere that survives the
  request that carried it.
- **Go cannot scrub the plaintext, and no claim is made that it can.** A garbage-collected runtime
  copies and the copies are not reachable to zero. The mitigations are the ones above: the
  plaintext lives for one request, in a type that will not print itself.
- **The dangling link closes and PROVENANCE is left alone.** `docs/decisions/0006-password-hashing.md`
  was the last of the three withheld decisions in PROVENANCE's *"Links with nothing to point at"*
  table; that table still describes the exported bytes, because it is a description and not a
  to-do list.
- **The second derivation reached the same algorithm as the first, and that is a finding.** The
  index row said Argon2id before this record existed; the argument above reached Argon2id from
  ADR-0002's `CGO_ENABLED=0`, a measured table and a deployment shape, without reading it as
  evidence. This is the experiment's own logic applied to an ADR — an independent second derivation
  either reproduces the first or it does not — and reproducing it says the constraints were
  determining rather than that the choice was arbitrary. **What did not come from the row are the
  parameters, the concurrency ceiling, the storage format, the rehash rule and the four rules of
  the verification path**, none of which the index row states and every one of which is a decision
  this record takes.

### What is not verified, and is owed

- **The measurements are one machine and one architecture.** Ten Apple-silicon cores is a
  comparison between candidates, which is what it was for. A small ARM host — the shape a media
  server most often runs on — will be several times slower at these parameters, and 52 ms could
  become a quarter of a second there. The number that would reopen the parameters is the wall-clock
  of one verification on that host, and nobody has taken it. ⚠️ UNVERIFIED
- **The timing equalisation is specified here and asserted nowhere.** No test yet shows the
  unknown-username and wrong-password paths landing within a measurable margin of each other, and a
  rule of this kind that is not tested is a rule that regresses the first time somebody adds an
  early return. Writing that test belongs to
  [002](../../specs/002-authentication-users-and-sessions/spec.md), and it is the one check this
  record's argument stands or falls on. ⚠️ UNVERIFIED
- **`golang.org/x/crypto/argon2` has not been verified by this project.** Argon2id's first pass is
  data-independent by construction, which is the property the *algorithm* offers against a
  cache-timing attacker; that says nothing about this implementation's own behaviour, and this
  project has audited none of it. ⚠️ UNVERIFIED
- **The reference's answer to a wrong password is still unmeasured.** It is
  [002 OQ-5](../../specs/002-authentication-users-and-sessions/spec.md#7-open-questions), held open
  because measuring it costs somebody's account a lockout counter. It does not bear on the
  algorithm, but it does bear on what the equalisation is equalising: v1 answers `401`, by its own
  decision, and if the reference turns out to answer something else then the two refusals this
  record works to make indistinguishable were never the right pair. ⚠️ UNVERIFIED

## Alternatives rejected

**PBKDF2-SHA512 at 210,000 iterations — what the reference itself stores.** This was the closest
call, and it has two arguments the winner does not. It is the only candidate that adds **no
dependency at all**: `crypto/pbkdf2` is in the standard library at this module's Go 1.25 floor, and
a project with two dependencies should notice when a third is avoidable. And it is what a Jellyfin
installation's own database holds
`[source: Emby.Server.Implementations/Cryptography/CryptographyProvider.cs:16 @ v10.11.11]`, so
matching it would be the only thing that could ever make an import path possible.

It loses on the column the table exists to show. At 43.0 ms it buys a per-guess memory footprint of
nothing, and SHA-512 is precisely the shape purpose-built cracking hardware is best at, so an
attacker holding the store file parallelises to the width of whatever they can rent. Raising it to
600,000 iterations costs this server 122.7 ms and moves that not at all — the ratio between
defender and attacker is what a memory-hard function changes, and iteration counts do not. As for
the import: **v1 has no import path, and the store's shape is not a compatibility surface** — no
hash crosses the wire, so matching the reference here would be paying a permanent cost in offline
resistance for a feature nobody has specified. **What would flip it** is v1 growing a Jellyfin
import, which would need this record superseded rather than amended.

**scrypt at `N=2^15, r=8, p=1`.** Genuinely close, and cheaper than the chosen row at 37.2 ms for
32 MiB. It is memory-hard, it is older and more widely deployed than Argon2, and it lives in the
same module so it costs the same dependency. It loses on tunability: `N` is the only knob with real
effect and doubling it doubles memory and time together, so on a host where 64 MiB is already the
ceiling, scrypt cannot be made stronger at all. Argon2id's `t` can. On a decision whose whole
future is *the parameters get raised*, a candidate that cannot be raised on the unconstrained axis
is the wrong one.

**bcrypt at cost 10 or 12.** The most deployed of the four, no memory question to answer, and 45.1
ms at cost 10 puts it in the same bracket as everything else. It is rejected on a measured
correctness problem rather than on cost: **Go's implementation refuses a 73-byte password
outright**, so a user with a long passphrase cannot have an account. The standard workaround — hash
the password to a fixed length first, then bcrypt that — is a scheme this project would then own,
document and get right, including the base64 step that exists to avoid feeding bcrypt a NUL byte.
That is a bespoke construction adopted to rescue a candidate that was not winning anyway. Its 4 KiB
working set is also two orders of magnitude below the memory-hard candidates.

**Argon2id at `m=256 MiB`.** 213.1 ms and, at the same ceiling of four, a **1 GiB** reservation —
measured, and linear. That is not a defensible thing to ask of a host that is also running ffmpeg,
and the extra strength is small next to the concurrency ceiling it would force down to one.

**Making the parameters configurable.** Superficially the answer to the previous paragraph: let an
operator on a large host raise them and an operator on a small one lower them. It is rejected
because the failure is silent and asymmetric — nobody notices that every credential written after
the day they lowered it is weaker — and because
[architecture §9](../architecture.md#9-configuration-identity-and-logging) keeps settings few and
not a feature. The record carrying its own parameters already gives an operator the safe half of
this, which is that a raise costs nothing and breaks nothing.

**A pepper — an HMAC key kept outside the database.** A real improvement against the exact threat
this record is defending: the store file leaking. It is rejected on
[architecture §5](../architecture.md#5-deployment-shape), which is *one process, one data
directory* — the key would live beside the database it protects, so the two leak together and the
pepper defends only the case where somebody reads one file and not its neighbour. It also needs a
rotation story, and rotating a pepper requires every plaintext, which nobody has. If a deployment
shape ever appears with a real secret store in it, this is the thing to revisit.

**Deferring the choice into 002's `plan.md`.** The most tempting, because 002 is the only feature
that will touch a password and the plan is where technology goes. It is rejected on shape: this is
a project-level, cross-feature commitment with alternatives worth writing down, which is the
definition of an ADR here, and [AGENTS.md §3](../../AGENTS.md) says a plan **inherits** project-level
choices rather than making them. The practical half is sharper still — 002's plan cannot cite an
inherited decision that does not exist, and the index row promising one has pointed at nothing
since the export.
