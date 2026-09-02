# The reference target

**Last verified: 2026-09-01, against Jellyfin 10.11.11 source and the 10.11.11 OpenAPI document.**

This document answers one question precisely: *when we say Atrium is compatible with Jellyfin,
compatible with what, exactly?*

## 1. The pinned version

**Atrium targets the Jellyfin `10.11.x` API.** Concretely:

| | Value |
|---|---|
| API contract | Jellyfin `10.11.11` OpenAPI document |
| Behavioural reference | Jellyfin `10.11.11` source and a running instance |
| Version Atrium reports | `10.11.11` — see §4 |
| Reference instance image | `jellyfin/jellyfin@sha256:aefb67e6a7ff1debdd154a78a7bbb780fd0c873d8639210a7f6a2016ad2b35db` — the published Jellyfin `10.11.11` image, **pinned by digest** and never by tag ([ADR-0007](../decisions/0007-a-container-runtime-for-the-reference-instance.md)). Written into this row on 2026-09-02 by the task that landed the single-use instance, which is the first run that had one to print. It is the **multi-architecture index** digest rather than one platform's, so a contributor on arm64 and a maintainer on amd64 pin the same line. `tools/_reference.py` holds the same value and `tests/conformance/test_differential.py` fails when the two drift apart |

The reasoning for pinning, and for pinning to this particular line rather than `master`, is in
[ADR-0004](../decisions/0004-pin-to-jellyfin-10-11.md).

> **The two pins are now one version, and the move that made them one is recorded here**
> (2026-09-01). `surface.yaml` pinned the `10.11.10` document while the reachable reference server
> is `10.11.11`; the gap was recorded on 2026-08-26 as undecided, on the grounds that moving the
> pin is a version move whose step 2 needs the differential harness feature 010 delivers. Two
> measurements taken on 2026-09-01 settled it the other way.
>
> **The first: the `10.11.10` document is unobtainable, and its one committed artefact was never
> stock.** `docs/compatibility/property-names.json` held 1043 names, and nineteen of them —
> `added`, `deleted`, `episodes`, `ids`, `imdb`, `movies`, `not_found`, `number`, `people`,
> `season`, `seasons`, `shows`, `slug`, `tmdb`, `trakt`, `tvdb`, `tvrage`, `updated`, `year` —
> appear nowhere in the `10.11.11` document at any depth, and nowhere in Jellyfin's source at
> `v10.11.11` either. They are the Trakt.tv API's vocabulary: an `ids` object of
> `trakt`/`slug`/`imdb`/`tmdb`/`tvdb`/`tvrage`, and a sync response of
> `added`/`deleted`/`updated`/`not_found` each holding
> `movies`/`shows`/`seasons`/`episodes`/`people`.
>
> **A Jellyfin's OpenAPI document is the core API plus whatever plugins are installed.** That is
> measured, not inferred: the reference server has six plugins, and two of its 316 paths —
> `/TMDbBoxSets/Refresh` and `/Tmdb/ClientConfiguration` — come from them
> `[probe: /Plugins and /api-docs/openapi.json, Jellyfin 10.11.11, 2026-09-01]`. None of the six is
> Trakt. So the index was an extraction of **one server's** `10.11.10` document, taken while that
> server had a plugin this one does not — and no server this project can reach serves that
> document. The freshness check could not pass anywhere, and had never once run.
>
> **The second: step 2 of the procedure had no input.**
> [conformance.md](conformance.md#when-the-reference-version-moves) says *"run the full
> differential harness against the **new server**"*, and there is no new server. Every one of this
> repository's 515 provenance tags reads `Jellyfin 10.11.11` and every one of its 340 source
> citations reads `@ v10.11.11`; not one names `10.11.10`. The running reference has been
> `10.11.11` for the whole project and the behavioural row of the table above always said so. What
> moved was the contract row alone — from a document nobody has to the document describing the
> server every probe already measured. Step 2 exists to catch behavioural differences a *server*
> change introduces; a document-only move introduces none, and conformance.md now says so in its
> own words.
>
> **What the move cost, measured before it was made:** all 461 aliases this project serialises are
> declared by the `10.11.11` document, so the sweep passes unchanged. The index went from 1043
> names to 1026 — losing exactly the nineteen, gaining `GenreItems` and `LockedFields`. The alias
> sweep's `MEASURED_BEYOND_THE_PINNED_DOCUMENT` exception, which carried `GenreItems` because the
> old pin lacked it, is empty and deleted. Steps 1 and 3 were run: the surface validator passes on
> all 59 endpoints against the `10.11.11` document — its first ever run against a document — and
> the claims this repository draws from the document were re-measured one by one, which moved two
> of their numbers (§2 below, and
> [behaviours §1.1](behaviours.md#11-property-casing-is-pascalcase)).
>
> **What the move did not fix, and cannot:** CI still has no document and must not have one, so the
> freshness step still skips. What replaces it is an assertion that needs no document — **no name
> in the index contains an underscore.** Jellyfin serialises PascalCase, and camelCase in its
> package and error schemas; of the 1026 names in the `10.11.11` document, none has one.
> `not_found` sat in the index from its first commit and nothing could see it: the checks that run
> without a document — sorted, unique, self-counting, pinning the same version `surface.yaml` does
> — are all true of a polluted index. `tests/conformance/test_aliases.py`.
>
> **[ADR-0004](../decisions/0004-pin-to-jellyfin-10-11.md) is not amended**, and its table still
> reads `10.11.10`. Its decision is *"pin to `10.11.x`"*, it names moving the pin as a deliberate
> act delegated to conformance.md, and a record is immutable once accepted
> ([decisions/README.md](../decisions/README.md)). The live values are the table above.

`master` (the 12.0.0 line) is explicitly **not** the target. It moves, it has already changed
behaviours that clients depend on, and no client ships against it.

## 2. Sources of truth, in precedence order

When two sources disagree, the higher one wins.

1. **A running Jellyfin 10.11.x** — probed by a script in `tools/`, with the result recorded.
   This is the only source that reflects what clients actually receive.
2. **The Jellyfin source at tag `v10.11.11`** — for behaviour that is hard to probe (error paths,
   ordering rules, identifier derivation).
3. **The OpenAPI document for 10.11.11** — for the shape of requests and responses, parameter
   names and enum vocabularies. It is the document a running reference serves, which means it is
   also the core API *plus that server's plugins*; §1 records what that cost once.

The OpenAPI document is last on purpose. It is generated from the C# controllers and is
**demonstrably not a complete description of behaviour**: it declares response headers with
`allowEmptyValue`, which is invalid for a Header object and makes strict parsers reject the whole
document; it declares all but three of its JSON responses three times with `profile="CamelCase"`
and `profile="PascalCase"` variants, **against the same schema, while two of the three serialise
differently**; and it declares `required` and `additionalProperties: false` on schemas that the
server does not actually honour.

The middle one is worth dwelling on, because this repository fell for it. Three content types
against one schema read as three names for one behaviour, and the specification said so. The
CamelCase variant really does emit camelCase — measured, in
[behaviours §1.13](behaviours.md#113-the-camelcase-profile-really-is-camelcase) — and no reading of
the document could have told anyone that. The document describes *shapes*, and a serialisation is
not a shape.
`[spec: directly observable in the 10.11.11 document]`

### Prior measurements, and the debt they carry

Some claims in this repository were measured against a real Jellyfin **before this repository
existed**, during the author's earlier client work. They are cited as
`[prior-probe: Jellyfin <version>, <date>]`.

They are real observations of a real server, and they are the reason the compatibility documents
start out substantive rather than speculative. But nobody can re-run them from here, which makes
each one a **standing debt**: it is discharged by writing the probe script under `tools/` that
reproduces the measurement, at which point the citation becomes a plain `[probe: …]`.

| Claim | Cited at | Discharged by | Status |
|---|---|---|---|
| ~~The four accepted authentication mechanisms~~ | 2026-06-13 | `tools/probe_auth_mechanisms.py` (feature 002) | ✅ **discharged 2026-08-26**, under a name this row did not carry, which is why it read *"not written"* for three weeks. And it moved the claim: there are **five** mechanisms, not four ([behaviours §2.4](behaviours.md#24-there-are-five-authentication-mechanisms-and-one-of-them-wins)); the fifth entered the probe on 2026-08-28 and all five are re-measured on every run |
| Item ids are 32 lowercase hex, **stable across rescans** | 2026-06-13 | `tools/probe_item_identity.py` (feature 003) — half of it | **Open, and half paid.** The form and the derivation are measured: 448 of 448 live ids reproduce from the item's own `Path` `[probe: tools/probe_item_identity.py, Jellyfin 10.11.11, 2026-08-27]`, and a value equal to that construction is 32 lowercase hex by construction. **Stability across rescans is not.** The probe reads one moment and never sees a second scan, and a rescan is a **write** — so it is the single-use reference instance's to answer, beside the scan [010 T10](../../specs/010-conformance-harness/tasks.md) already performs. [behaviours §1.4](behaviours.md#14-item-identifiers-are-32-lowercase-hex-characters) keeps the `prior-probe` for that half alone |
| ~~`UserData` is returned without `Fields`~~ | 2026-06-13 | `tools/probe_item_shapes.py` (feature 005) | ✅ **discharged 2026-08-27**, under another name again. `UserData` is present on the bare list row of all nine content types and of `/UserViews` — 12 of 12 items each, with no `Fields` and no `EnableUserData` — and its keys include `Key` and `ItemId` `[probe: tools/probe_item_shapes.py, Jellyfin 10.11.11, 2026-08-27]`. It also narrowed the claim: a by-name row from `/Genres` carries **no** `UserData` at all, where the same genre through `/Items?ids=` does `[probe: manual requests via tools/_probe.py, Jellyfin 10.11.11, 2026-08-28]` |
| ~~Item-level `Container` is a demuxer list~~ | 2026-06-13 | `tools/probe_media_container.py` (feature 008) | ✅ **discharged 2026-08-29** — and the claim did not survive as written: the item level is a list for the mp4 family and a single word for everything else, and the single form on a listing is the **file's own extension** rather than anything a profile resolved ([behaviours §1.6](behaviours.md#16-container-at-item-level-is-a-list-for-some-formats-and-the-single-form-is-per-response)). The 2026-06-13 reading is kept rather than deleted: it was taken on an mp4 and is true of one — it generalised wrongly rather than failing to reproduce |
| ~~`StartIndex` present in list envelopes~~ | 2026-06-13 | `tools/probe_query_envelope.py` (feature 005) | ✅ **discharged 2026-08-26** |
| ~~`/Users/Public` may return `[]`~~ | 2026-06-13 | `tools/probe_public_users.py` (feature 010) | ✅ **discharged 2026-09-02**, on an instance this project stood up and destroyed — hiding every account on an operator's server was never a measurement anybody could take. The claim holds and it **understated the case**: `IsHidden` is true on the administrator the wizard makes and on every account `POST /Users/New` creates `[source: Jellyfin.Data/UserEntityExtensions.cs:174 @ v10.11.11]`, so a server nobody has configured already answers `200 []`. The flag was measured in both directions — two un-hidden accounts, one hidden, none — and read with **no credential**, because two of the route's four filters read the caller ([behaviours §2.2](behaviours.md#22-userspublic-can-legitimately-be-empty)) |
| ~~The `SortBy` vocabulary~~ | 2026-06-13 | `tools/probe_sort_vocabulary` (feature 005) | ✅ **discharged 2026-09-02** — and the claim did not survive as written. The doubt was right: the set is a **floor read as a vocabulary**. The reference's enumeration names thirty and **twenty-one tokens outside the recorded eight order rows**, including all three a shipping music client sends `[probe: tools/probe_sort_vocabulary, Jellyfin 10.11.11, 2026-09-02]`. The control token answered identically to `Default`, which is the method checking itself; `Random` was the one member whose identical requests differ; `SeriesDatePlayed` orders too, at 63 seconds a request where every other token answers in under one. [behaviours §2.5](behaviours.md#25-sortby-vocabulary) carries the finding and hands the decision to [005](../../specs/005-item-query-api/spec.md), because which of the thirty v1 serves is a Principle I question now rather than an assumption |
| ~~Dates carry seven fractional digits~~ | 2026-06-19 | `tools/probe_wire_format` (feature 001) | ✅ **discharged 2026-09-02** — and it narrowed the claim. **Seven is the maximum and the usual case, not a constant**: 346 of 352 date values carry seven digits, and `LastPlayedDate` and `LastActivityDate` were observed with six and three `[probe: tools/probe_wire_format, Jellyfin 10.11.11, 2026-09-02]`. The mechanism is not established and [behaviours §1.2](behaviours.md#12-dates-carry-up-to-seven-fractional-digits) says so rather than guessing — trailing-zero trimming explains every short value and is contradicted by `.0000000` being written in full elsewhere. The run returned two things nobody asked it for: three date fields do **not** end in `Date`, so [conformance.md](conformance.md)'s unit sweep as written checks six of nine; and a casing sweep that treats dictionary keys as property names reports 688 of 899 keys as failures |
| ~~`/Sessions/Playing/Progress` needs no `MediaSourceId`~~ | 2026-06-13 | `tools/probe_playstate.py` (feature 007) | ✅ **discharged 2026-08-26** |
| ~~PCM/WAV transcoding returns 500, and `/universal` returns headerless PCM~~ | 2026-08-03 | `tools/probe_universal_audio.py` (feature 008) | ✅ **discharged 2026-08-29** — and it moved both claims: the 500 has two causes rather than one, and the headerless body comes from the *transcoding* container rather than from `Container` |
| ~~`LocalAddress` gets an HTTPS override~~ | 2026-08-14 | `tools/probe_local_address.py` (feature 010) | ✅ **discharged 2026-09-02** — and it reproduced exactly: the same route over the same plain-HTTP request answers `http://<address>:8096` before a certificate and `https://<address>:8920` after one, the scheme **and** the port. It needed the instance for two reasons rather than one: installing a certificate is a write to a configuration, and the certificate is read at **startup**, so the run also has to restart the server it configured ([behaviours §2.3](behaviours.md#23-localaddress-is-one-string-and-may-be-https), and §4.2's argument rests on it) |
| ~~`TotalRecordCount` is 0 without `limit`~~ | 2026-08-05 | `tools/probe_by_name_counts.py` (feature 005) | ✅ **discharged 2026-08-28** |
| ~~The `/System/Info/Public` payload: seven fields, their order and shapes~~ | 2026-06-13 | `tools/probe_public_info.py` (feature 001) | ✅ **discharged 2026-08-28** — the 2026-08-28 audit (M8) found this claim carried no register row at all |
| ~~`AccessToken` is 32 lowercase hex~~ | 2026-06-13 | `tools/probe_auth_mechanisms.py` (feature 002) | ✅ **discharged 2026-08-28** — same audit finding: no row until the discharge |
| ~~`ImageTags` is a map and `BackdropImageTags` a list~~ | 2026-06-13 | `tools/probe_image_tags.py` (feature 006) | ✅ **discharged 2026-08-28** — same audit finding: no row until the discharge |

**These probes are Go, where the register named Python.** The addresses in this table were the
exporting project's; `tools/` here holds Go programs
([architecture §3](../architecture.md#3-repository-layout)), run as `go run ./tools/<name>`. The
finding is what a row records and the address is not — and a probe that measures the reference is
this project's to write when the register says nobody has written it.

**Written is not discharged.** A script that exists but has never been pointed at a server has
proved nothing; the citation changes from `prior-probe` to `probe` only when it has been run and
its finding recorded.

**And discharged under another name is still discharged**, which is the half this register was
missing. **Four** of its rows named a script nobody ever wrote while the question was already
being answered — whole or in part — by a probe written for some other feature, and a row that says *"not written"* about a
measurement somebody has taken is worse than no row at all: it hides work, and it makes the debt
look bigger than it is. **The row now names the script that actually answered it**, and the test
below refuses a struck row that names a file which is not there.

**Twelve down, three to go**, reconciled on 2026-09-02 at [010 T1](../../specs/010-conformance-harness/tasks.md) and moved the same day by [010 T13](../../specs/010-conformance-harness/tasks.md), which paid the two rows that needed a server this project may configure.
The first two were re-measured on 2026-08-26 against a live 10.11.11 and both held: `StartIndex` is
present on every envelope, and `/Sessions/Playing/Progress` is accepted without a `MediaSourceId`.
Every discharged citation is now a plain `probe:` and its row is struck from this register.

**Each of the three that remain says why it is still open**, because AC-9 of
[010](../../specs/010-conformance-harness/spec.md) asks for a probe script *or a recorded reason
there cannot be one*, and a bare *"not written"* is neither. **The two that were blocked on a
configuration are paid**: `/Users/Public` returning `[]` and the `LocalAddress` HTTPS override were
both measured on 2026-09-02 against a single-use instance
([ADR-0007](../decisions/0007-a-container-runtime-for-the-reference-instance.md)), which is what
that instance exists for. Of the three left, one is blocked on something other than an author — the
item-identity row needs a library scanned **twice** — and the other two need somebody to write ten
lines of `urllib`.

`tests/unit/test_probe_convention.py` asserts the properties this table has to keep: a struck
row names a script that exists under `tools/`, an open row names one that exists or carries its
reason, the sentence above is recomputed from the rows rather than believed, and every dated
`prior-probe` citation in the repository belongs to a row — which is the 2026-08-28 audit's M8
finding, where three claims cited a prior measurement this register had never recorded.

**Every run so far has returned more than the claim it was sent to check** — three envelope
shapes the original measurement had never covered, a six-branch completion rule where the
documentation had two thresholds, and, at the PCM/WAV row, a symptom with two causes where one
was recorded and a symptom recorded against a parameter that does not produce it. The three rows
struck on 2026-09-02 say the same thing again, and all three had been sitting in the register as
*"not written"* while their answers were already recorded elsewhere: **four** authentication
mechanisms turned out to be five, an item-level `Container` is a list for one family of formats and
a single word for the rest, and `UserData` is on every item **except** a by-name row from
`/Genres`. That is the argument for discharging the rest rather than trusting them.

A claim that fails to reproduce when its probe is finally written is not quietly dropped: it goes
into [behaviours.md](behaviours.md) as a behaviour that *changed*, with both dates.

### Obtaining the reference documents

The OpenAPI document is **not vendored** into this repository — it is generated from GPL-licensed
source, and vendoring it would drag a licensing question into a repository that does not need one
(see [ADR-0005](../decisions/0005-licence.md)). Fetch it instead:

```bash
python3 tools/fetch_reference_spec.py http://<your-jellyfin>:8096 --out reference/openapi.json
```

`reference/` is git-ignored. A local checkout of the Jellyfin source at `v10.11.11` is the second
input; the probe scripts need neither.

## 3. What "compatible" means, in four levels

Parity is not one thing. Each endpoint in
[api-surface-v1.md](api-surface-v1.md) is assigned a level:

| Level | Meaning | How it is proven |
|---|---|---|
| **L0 — Routed** | The path exists and returns a plausible status code. | Route test |
| **L1 — Shape** | The response has the right fields, casing, types and units. | Golden-response test |
| **L2 — Semantic** | The response has the right *values* for a known library state. | Fixture library test |
| **L3 — Differential** | The response is byte-comparable to a real Jellyfin's, modulo a documented allowlist of legitimately-varying fields. | Differential harness |

**v1 requires L2 for every endpoint in the surface, and L3 for the endpoints on the playback and
authentication paths** — the two places where a client's behaviour actually diverges when the
server is wrong.

Full method in [conformance.md](conformance.md).

## 4. Server identity: what Atrium tells clients it is

This is the one place where Principle I (zero delta) and Principle X (honest about lineage) pull
against each other, so it is settled here rather than left to the implementation.

`GET /System/Info/Public` returns, among other fields:

```json
{
  "ServerName": "atrium",
  "Version": "10.11.11",
  "ProductName": "Jellyfin Server",
  "OperatingSystem": "",
  "Id": "<32 hex chars>",
  "LocalAddress": "http://host:8096",
  "StartupWizardCompleted": true
}
```
`[probe: tools/probe_public_info.py, Jellyfin 10.11.11, 2026-08-28]`

**`ProductName` must be `"Jellyfin Server"` and `Version` must be a real 10.11.x version.** This
is not cosmetic: `ProductName` is the documented discriminator that multi-server clients use to
decide whether they are talking to Emby or Jellyfin, and the version string drives client-side
capability gating. A client that reads `"Atrium"` there takes an unknown-server path, and
Principle I is broken at the very first request.

Honesty is preserved where it costs nothing and where humans, not clients, are reading:

- The `ServerName` field is the operator's chosen name and defaults to `atrium`.
- The HTTP `Server` response header identifies Atrium and its own version.
- The README, the project page and every log line say plainly what this is.

**Decision:** identify as Jellyfin on the fields clients parse; identify as Atrium everywhere a
human looks. This is recorded as a deliberate, permanent exception in
[behaviours.md](behaviours.md).

## 5. What is *not* a target

- **Emby.** Emby's API is the ancestor of Jellyfin's and diverges in real ways: numeric item ids
  instead of GUIDs, `LocalAddresses[]` instead of `LocalAddress`, user-scoped write routes,
  `/universal.mp3`. Atrium implements the Jellyfin dialect only. Multi-server clients already carry
  an Emby driver; Atrium falls on the Jellyfin side of that split, which is exactly what makes its
  delta zero.
- **The Jellyfin web UI.** Serving it would pull in `DisplayPreferences`, `Branding`,
  `Configuration`, `QuickConnect`, `Localization` and a static asset pipeline — a large surface
  whose only consumer is a UI this project is not building. Revisit as a v2 goal.
- **Plugins.** Jellyfin's plugin API is a .NET assembly-loading contract. There is no Python
  equivalent and no reason to invent one.
- **`master`/12.0.0.** See §1.
