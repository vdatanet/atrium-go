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

### Claims this project asserts and has never measured — added 2026-09-03

The register above is about measurements somebody **took** and nobody here can re-run. This one is
the opposite debt and it is the larger of the two: places where an implementation had to answer a
question the reference has not been asked, wrote down the answer it took, and said so.

They are collected here because they were scattered — each one was recorded where it arose, in a
`plan.md` section or beside a constant in the code, and four separate tasks of feature 001 wrote
*"this belongs in the register of measurements that still owe a probe"* without anyone owning the
document. Collected, they are one afternoon's work against a running reference and a container
this project already knows how to stand up
([ADR-0007](../decisions/0007-a-container-runtime-for-the-reference-instance.md)).

**The rows are not `prior-probe` citations and this table is not the one above.** Nothing here was
ever measured; each row is an `⚠️ UNVERIFIED` inference or a specification this project implemented
**as written** while its own source reading pointed the other way. Where the second is true the row
says so, because AGENTS.md §1.3 makes the running server the tie-breaker and there is no running
server here — *the specification was not amended on source evidence, and that is the correct move,
not an oversight.*

| # | What is unmeasured | What this project does today | Recorded at | One request? |
|---|---|---|---|---|
| U-1 | **Whether a starting reference exempts anything from its `503`.** Three source readings contradict [001 §3.5](../../specs/001-server-identity-and-discovery/spec.md)'s *"nothing is exempt"*: the starting `503` comes from a **separate setup web server** which registers no response-time middleware `[source: Jellyfin.Server/ServerSetupApp/SetupServer.cs:177-259 @ v10.11.11]`, that server answers a **real** `/System/Info/Public` rather than a `503` `[source: .../SetupServer.cs:204-237 @ v10.11.11]`, and the **main** pipeline's gate exempts `/system/ping` and sends **neither** `Retry-After` nor `Message` `[source: Jellyfin.Api/Middleware/ServerStartupMessageMiddleware.cs:38-48 @ v10.11.11]` | Implements §3.5 as written: every route, no exemption, both headers, a `text/html` body | 001 plan §6.8 | Four routes plus one unrouted path against a *starting* reference, recording status, `Retry-After`, `Message`, `Content-Type` and `X-Response-Time-ms`. Hard only because it has to catch a server mid-start |
| U-2 | **The order of `LocalAddress` tiers 1 and 2.** The overload that serves this response tests `EnablePublishedServerUriByRequest` **before** `PublishedServerUrl` `[source: Emby.Server.Implementations/SystemManager.cs:77,120; ApplicationHost.cs:885-901 @ v10.11.11]`; [behaviours §2.3](behaviours.md)'s corrected table and [001 §3.4](../../specs/001-server-identity-and-discovery/spec.md) number them the other way | Implements the spec's order — published URL first | 001 plan §6.6 | Yes. One installation with **both** configured is the only case the two readings differ on |
| U-3 | **The key order of a JSON body.** L3 compares bytes, so key order is contract. No probe in this repository records the key order of any response, and this repository's two sample bodies for `/System/Info/Public` disagree (§4 opens with `ServerName`, 001 §3.1 with `LocalAddress`) | Ships the reference model's declaration order `[source: MediaBrowser.Model/System/PublicSystemInfo.cs:14-53 @ v10.11.11]` | 001 plan §8, marked `⚠️ UNVERIFIED` at the model | Yes, and it is the response every client enters the API through. **Weaker still on `/System/Info`**, whose model *derives* from the public one: where a serialiser puts an inherited property is a property of that serialiser and nothing here records it |
| U-4 | **Whether `Retry-After` is zero-padded.** The setup server formats it `"000"` — `005` for a five-second hint `[source: Jellyfin.Server/ServerSetupApp/SetupServer.cs:242 @ v10.11.11]`; the pinned document declares an integer | Sends `5` | 001 plan §6.8 | Rides U-1 |
| U-5 | **The bytes of the `text/html` `503` body.** There is no single reference body to copy: a bare string in one place, a page rendered from the startup log in the other | Asserts the media type and nothing about the bytes | 001 plan §6.8 | Rides U-1 |
| U-6 | **What a `charset` beside a `profile` does to *ranking*.** [§1.13](behaviours.md#113-the-camelcase-profile-really-is-camelcase) measured the fallback on a **single-range** `Accept` only. Two readings fit: the range becomes a plain candidate and keeps its rank, or it matches nothing and the next range is tried | Takes the first reading, because 001 §3.0.2 says "falls back to the plain type" in as many words | 001 plan §6.3 | Yes, and exactly one request tells them apart: `Accept: application/json; profile="CamelCase"; charset=utf-8, application/json; profile="CamelCase"` |
| U-7 | **What a percent-encoded *literal* path segment matches.** Canonicalisation runs on the **escaped** path, because folding the decoded one would segment on a `%2F` a client encoded precisely so it would not be a separator — so `/%53ystem/Info/Public` does not fold here | Does not fold it | 001 plan §6.1 | Yes |
| U-8 | **What a percent-encoded query *name* matches.** Same shape as U-7, on the query string's own bytes: `%4Cimit` is not `limit` here | Does not fold it | 001 plan §6.2 | Yes |
| U-9 | **What an unknown method token on an unrouted path answers.** 001 §3.6 keys its `404` on the path, so this server answers `404` whatever the method; that is a *reading* of §3.6 rather than a measurement | `404` | 001 plan §6.5 | Yes |
| U-10 | **Whether the reference writes a character above U+FFFF as a surrogate pair**, and **whether it escapes a control character at all.** [§1.16](behaviours.md) was measured only on characters fitting one UTF-16 code unit | Writes `\uXXXX\uXXXX`, and spells a control character `\uXXXX` upper-cased for consistency with §1.16 | 001 plan §6.4, marked in `internal/wire` | Yes — an item or playlist name carrying an emoji settles both |
| U-11 | **The shape of a `500`.** [§1.11](behaviours.md) has no `500` row, so nothing is known about whether the reference sends a body with one | Status, empty body, nothing else | 001 plan §7 | Yes |
| U-12 | **Whether the reference binds a query pair carrying a semicolon.** Go discards one — `url.ParseQuery` answers `invalid semicolon separator in query` for `a;b=1&c=2` and `r.URL.Query()` swallows the error `[measurement: net/url, Go 1.27.0, 2026-09-03]` — and ASP.NET Core has no such rule, so a request the reference binds may reach a handler here with the parameter missing | Neither creates nor hides it: query canonicalisation splits on `&` alone and leaves the fragment in place | 001 plan §6.2 | Yes, and it is owed by the first feature that reads a query **value** rather than folding a name |
| U-13 | **Whether exceeding `MaxActiveSessions` evicts a session or refuses the login.** [002 §3.8](../../specs/002-authentication-users-and-sessions/spec.md#38-sessions) and its AC-13 evict the least recently used session; the reference counts the user's sessions and throws `SecurityException("User is at their maximum number of sessions.")` `[source: Emby.Server.Implementations/Session/SessionManager.cs:1623-1629 @ v10.11.11]`, which its exception filter turns into the `403` and the 25 bytes `[source: Jellyfin.Api/Middleware/ExceptionMiddleware.cs:92-99,123-134 @ v10.11.11]` **And there is a second half, added 2026-09-03 at 002 T20 in the change that implemented the ceiling:** the reference's check runs **before** it touches the session list and counts the session a re-authentication is about to replace `[source: Emby.Server.Implementations/Session/SessionManager.cs:1623-1629 @ v10.11.11]`, so an account whose `MaxActiveSessions` is 1 **cannot log in again from the device it is already on** there. Here it can, because plan §6.5 replaces that session rather than adding one and `sessions.Evictions` excludes it from the count | Implements the specification as written: the oldest session is evicted and the login succeeds | 002 plan §6.7 | Yes, and one probe answers both halves: send the second login twice, once from a new device and once from the old one |
| U-14 | **What `POST /Users/Configuration` does with a `userId` the specification does not mention.** The reference declares the parameter, updates the named user, answers `404` for an identifier nobody has and `403` — carrying its own message rather than the 25 bytes — to a caller who may not update that user `[source: Jellyfin.Api/Controllers/UserController.cs:488-511 @ v10.11.11]`. [002 §3.6](../../specs/002-authentication-users-and-sessions/spec.md#36-post-usersconfiguration--updateuserconfiguration) says only "the authenticated user's configuration" **`POST /Sessions/Capabilities/Full`'s `id` is the same act on a second route** (added 2026-09-03 at 002 T16): the reference declares it and this server ignores it, so a client naming another session updates its own here. | Ignores the parameter, per [§1.12](behaviours.md#112-an-unrecognised-query-value-is-ignored-not-rejected), and updates the caller's own configuration — so an administrator naming somebody else writes to the wrong account rather than erroring. The same on the capabilities route, asserted at `TestPostingCapabilitiesNamingAnotherSessionUpdatesTheCallersOwn` | 002 plan §9 | Yes, and it needs two seats: one request settles the parameter, its `404` and the shape of its `403` |
| U-15 | **Whether `EnableRemoteAccess` and `AccessSchedules` really refuse a login.** Both are in 002 §3.5's twenty-eight *stored and unenforced* flags, on the argument that every one of them gates a feature v1 lacks; the reference enforces both **at authentication** `[source: Jellyfin.Server.Implementations/Users/UserManager.cs:595-611 @ v10.11.11]`, which v1 has | Admits both logins; the two flags are stored and echoed like the other twenty-six | 002 plan §9, 002 §3.5 | Yes, and it costs no lockout counter: a restricted account and a caller outside the local network |
| U-16 | **What `GET /Sessions` shows when two users share one device and one client.** The reference keys a session on `(Client, DeviceId)` alone and updates its `UserId` `[source: Emby.Server.Implementations/Session/SessionManager.cs:486-487,554 @ v10.11.11]`, while keying a *token* on `(user, device)` — so the pair holds two tokens and one session row, naming whoever authenticated last. [002 §3.8](../../specs/002-authentication-users-and-sessions/spec.md#38-sessions) calls a session a `(user, device, client)` triple, which reads as its contents rather than its key | Follows the reference's key: one session row, two tokens | 002 plan §6.5 | Yes — two logins from one device and client, as two users |
| U-17 | **What `GET /Sessions` answers for a value that cannot parse as its parameter's type, and for an empty one.** [002 §3.8](../../specs/002-authentication-users-and-sessions/spec.md#38-sessions) declares `controllableByUserId` as an identifier and `activeWithinSeconds` as an integer `[source: Jellyfin.Api/Controllers/SessionController.cs:52-59 @ v10.11.11]`; [§2.25](behaviours.md#225-get-sessions-three-filters-are-two-filters-and-a-visibility-rule) measured neither a malformed value nor an empty one for either, and [§1.12](behaviours.md#112-an-unrecognised-query-value-is-ignored-not-rejected)'s line is token-versus-type rather than lenient everywhere | Answers `400` with the validation body keyed on the parameter's own spelling — the shape 002 §3.7 measured for `userId` — and treats an empty value as absent, which is the direction the *measured* `deviceId` row takes | 002 §3.8, marked `⚠️ UNVERIFIED` there | Yes, and it costs nothing: four requests from one seat, no writes |
| U-18 | **Whether this route's `403` carries the 25 bytes.** §2.25 measured the status and `text/plain` for a caller who is not an administrator naming somebody else in `controllableByUserId`; the *bytes* follow from the reference raising it as a controller exception `[source: Jellyfin.Api/Helpers/RequestHelpers.cs:79-82 @ v10.11.11]` that its exception filter turns into `Error processing request.` `[source: Jellyfin.Api/Middleware/ExceptionMiddleware.cs:92-99,123-134 @ v10.11.11]` — which is [§1.11](behaviours.md#111-there-are-four-error-shapes-not-one)'s measured **rule** applied to this route rather than a measurement of it | Sends the 25 bytes, byte-identical to the login refusals | 002 §3.8, and AC-15 asserts them | Rides U-17, from a restricted seat |
| U-19 | **Whether an `Authorization` header that is present but unreadable stops the search for a credential.** The reference falls back to `X-Emby-Authorization` only when the first field is **absent**, not when it is present and yields nothing `[source: Jellyfin.Server.Implementations/Security/AuthorizationContext.cs:231-238 @ v10.11.11]`, so `Authorization: Bearer x` beside `X-Emby-Authorization: MediaBrowser Token="<good>"` is `401` there and `200` here. [002 plan §6.1](../../specs/002-authentication-users-and-sessions/plan.md#61-token-extraction) states the rule in general terms, which is a generalisation of measured pairs that each carried a readable scheme word. **Of the six precedence pairs and the four unreadable-first-header rows the reader is asserted over, this is the only one that is a candidate difference**: `X-Emby-Token` and the two query names are read from their own fields on both servers `[source: .../AuthorizationContext.cs:98-111 @ v10.11.11]` | Reads the second header and answers `200` | 002 plan §6.1, ⚠️ UNVERIFIED at `presentedToken` and `presentedClientIdentification` | Yes, and it is the highest-value row here: one request, on an authenticated path, on the header every client sends |
| U-20 | **Which of `ApiKey` and `api_key` wins when a request carries both and they disagree.** The reference reads `ApiKey` and consults `api_key` only when that is empty `[source: .../AuthorizationContext.cs:103-111 @ v10.11.11]` — a precedence between *names*. [§2.4](behaviours.md#24-there-are-five-authentication-mechanisms-and-one-of-them-wins) records that the two spellings were never set against each other | Positional: whichever comes first in the raw query string, so `?api_key=a&ApiKey=b` is `a` here and `b` there. Decided deliberately at plan §6.1 rather than left to a map | 002 plan §6.1 and §9 | Yes |
| U-21 | **`X-MediaBrowser-Token`, a sixth field the reference reads** between `X-Emby-Token` and the query names `[source: .../AuthorizationContext.cs:98-101 @ v10.11.11]`. [002 §3.1](../../specs/002-authentication-users-and-sessions/spec.md#31-how-a-client-presents-a-token) measured and declares five | Reads five. A request carrying only that header authenticates there and not here — Principle VI: a sixth mechanism nothing has probed and no client analysis names is an endpoint-shaped stub in header form | 002 plan §6.1 | Yes |
| U-22 | **Whether the `X-Emby-Authorization` grammar's strictness around the `=` is a rule or a consequence.** [002 §3.2](../../specs/002-authentication-users-and-sessions/spec.md#32-how-a-client-identifies-itself) states it as a symmetric rule, measured as a refusal `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]`. The reference has no such rule: it trims the *name* before the `=`, does not trim the *value* after it, then strips quote characters off the value's ends `[source: .../AuthorizationContext.cs:276-317 @ v10.11.11]` — so `Token = "x"` yields a `Token` whose value is `` `"x`` (which is why the probe saw `401`) while `Token ="x"` appears to yield a clean `x`. **The probe measured the refusal, not the mechanism** | Implements the symmetric rule spec §3.2 and plan §6.3 both write | 002 plan §6.3, and `ParseClientIdentification`'s doc comment | Yes, and the request is not the one already sent: a header carrying `Client ="x"`, whose `Client` this server drops and the reference appears to keep |
| U-23 | **Whether `AuthenticationResult.User` reflects the login it just performed.** The reference's update bypasses the entity it then serialises `[source: Jellyfin.Server.Implementations/Users/UserManager.cs:614-627 @ v10.11.11]`, which reads as though it answers the **previous** `LastLoginDate` — and on a first login that is no member at all, where this server sends one | Answers the account as the store holds it *after* the transition: `InvalidLoginAttemptCount` already `0`, `LastLoginDate` already stamped | 002 plan §6.5 | Yes, and it is on this project's one L3 body: provision an account, authenticate twice, read `User.LastLoginDate` out of the **first** response |
| U-24 | **Three `SessionInfo` members the reference sends on a session that has never played anything.** It constructs `PlayState` eagerly and initialises both queues empty `[source: MediaBrowser.Controller/Session/SessionInfo.cs:44-48 @ v10.11.11]`; [002 §3.8](../../specs/002-authentication-users-and-sessions/spec.md#38-sessions) conditions `PlayState` on something playing | Sends spec §3.8's fifteen members and not those three. `Capabilities` is the body's **first** member only because `PlayState` and `AdditionalUsers` are absent, which is why the order assertion transcribes the whole list | 002 plan §6.10 | Yes, and it is on the one L3 body. 007 inherits it with `PlayState` |
| U-25 | **`EnableAutoLogin`.** The reference fills it per account; v1 stores no such column | `false` for every account | 002 plan §6.6 | Rides any user-object request |
| U-26 | **How `RemoteEndPoint` is spelled.** This server sends the host half of `RemoteAddr`, parsed and unmapped so that a dual-stack listener's `::ffff:192.168.1.44` is the IPv4 address an operator recognises; the reference has its own `GetNormalizedRemoteIP` and nothing here has compared the two | The normalisation `requestFacts` already applies everywhere else | 002 plan §6.5 | Yes, and it rides any authentication |
| U-27 | **The login route's refusal order.** A request carrying both an unusable client header **and** a wrong password is `400` here and never `401`, read from the reference checking its four arguments at the session manager before it looks a user up `[source: Emby.Server.Implementations/Session/SessionManager.cs:1589-1592 @ v10.11.11]`. That pair has never been sent to a running reference | `400`, asserted at `TestTheClientComponentsAreCheckedBeforeTheCredential` and its companion for the body | 002 plan §6.3 | Yes |
| U-28 | **`CastReceiverId`, on both sides of the round trip.** Reading, the reference answers it with the first cast receiver application the installation declares `[source: Jellyfin.Server.Implementations/Users/UserManager.cs:426-447 @ v10.11.11]`; writing, it keeps a posted value only when the installation declares an application with that identifier `[source: .../UserManager.cs:760-799,785-789 @ v10.11.11]`. 001 answers `/System/Info` with an empty `CastReceiverApplications`, so Atrium declares none | Reads `""`, and stores whatever was posted. Asserted as a divergence at `TestACastReceiverIdTheInstallationDoesNotDeclareIsStoredHere` | 002 plan §6.6 and §6.2 | Yes, and it is a third request on `POST /Users/Configuration` beside U-14's two |
| U-29 | **`AuthenticationProviderId` and `PasswordResetProviderId`.** Two of the 42 policy members that travel, carrying the reference's .NET type names (`Jellyfin.Server.Implementations.Users.Default{Authentication,PasswordReset}Provider`), read from source and never measured | Sends the two type names verbatim | 002 plan §6.6 | Rides any user-object request |
| U-30 | **Whether a default account really sends sixteen configuration properties.** [002 §3.6](../../specs/002-authentication-users-and-sessions/spec.md#36-post-usersconfiguration--updateuserconfiguration) measures **16** `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]`. The reference's own source coerces only the *subtitle* preference to `string.Empty` `[source: Jellyfin.Server.Implementations/Users/UserManager.cs:426-447 @ v10.11.11]` while `AudioLanguagePreference` is null on a fresh account `[source: src/Jellyfin.Database/Jellyfin.Database.Implementations/Entities/User.cs:113 @ v10.11.11]`, and [§1.7](behaviours.md) omits a null — so read from source alone a fresh account sends **15**, and this server has an extra member on the wire | Follows the measurement, which is what [AGENTS.md §1.3](../../AGENTS.md) asks for: all three are declared plain strings and sixteen travel | 002 §3.6's 2026-09-03 note, 002 plan §6.6 | Yes, and it is the same request that settles U-28 |
| U-31 | **The order `GET /Users/Public` answers in.** The reference orders by the *unfolded* username, culture-aware `[source: Jellyfin.Api/Controllers/UserController.cs:653-655 @ v10.11.11]` | Orders by the folded username then the identifier, because [architecture §2](../architecture.md) requires determinism — not because anything measured says the reference orders them this way | 002 plan §6.2 | Yes — two accounts whose folded and unfolded orders differ |
| U-32 | **Three narrowings `GET /Users/Public` applies at the reference that [002 §3.4](../../specs/002-authentication-users-and-sessions/spec.md#34-get-userspublic--getpublicusers) does not name.** Disabled accounts are excluded as well as hidden ones `[source: Jellyfin.Api/Controllers/UserController.cs:117,625-633 @ v10.11.11]`; before the startup wizard completes it answers **every** account `[:111-115]`; and after it completes it narrows by the authenticated principal's device access and by `EnableRemoteAccess` for a caller outside the local network `[:635-651]` — the **second** place that flag decides something, U-15 being the first, and the reason §3.4's measured byte-equality is really *"to a local caller on a permitted device"* | Implements §3.4 as written: hidden only. The disabled divergence is asserted at `TestADisabledUserIsListedHereAndTheReferencesSourceExcludesIt`; the pre-setup listing is unreachable here, because provisioning the first account completes setup (§3.9) | 002 plan §6.2 | Yes |
| U-33 | **`GET /Users/{userId}`'s refusal order.** A request carrying no credential **and** a malformed `userId` is the empty `401` here, read across from 009 §3.8's measurement that the reference's authorization filter runs ahead of its model binder and never sent on this route | `401`, asserted at `TestTheCredentialIsReadBeforeTheIdentifierIsBound` | 002 plan §7 | Yes |
| U-34 | **The identifier grammar.** This server refuses the dashed and braced spellings 009 §3.8 measured the reference's binder accepting, which is a delta in [§3.0.3](behaviours.md)'s dangerous direction: a request that succeeds everywhere else meets a `400` here | 32 hex characters in either case, folded to lower, and nothing else. Asserted as a divergence at `TestADashedIdentifierIsRefusedHereAndTheReferencesBinderParsesIt`, so the day it is measured is a failing test rather than a rediscovery | 002 plan §7 | Yes |
| U-35 | **The reference's two activity throttles, and the member they decide.** It stamps a *user* only after sixty seconds `[source: Emby.Server.Implementations/Session/SessionManager.cs:265-271 @ v10.11.11]` and a *token* only after three minutes `[source: Jellyfin.Server.Implementations/Security/AuthorizationContext.cs:180-184 @ v10.11.11]`, while keeping the session's own date exact — it holds sessions in memory where this server holds them in a table. **The consequence is sharper than the throttle:** nothing in 002 calls `ports.UserStore.TouchActivity`, so `LastActivityDate` is **absent** from every user object this server sends where the reference sends one. That is an absent member rather than a differing value, and [allowlist.yaml](allowlist.yaml)'s `wall-clock` entry for that pointer excuses a value | Stamps the session at most once per second and the account never | 002 plan §6.10, 002 §3.5's table note | Yes, and it rides any user-object request |
| U-36 | **Whether the empty policy `403` declares `Content-Length: 0`.** [§1.11](behaviours.md#111-there-are-four-error-shapes-not-one)'s row is *no content type, no body*; a silent measurement is not a measured absence | Declares `0`, which is 001's uniform decision across its refusal shapes and keeps one spelling of an empty refusal in one file. Omitting it produces no absence on the wire anyway — `net/http` adds a length to a body-less response — so only a `HEAD` could differ, and no policy-refused route answers one | 002 plan §7, `WriteForbidden`'s doc comment | Rides any policy `403` |
| U-37 | **Whether the reference converts a stored capabilities document's keys under the camelCase profile.** `internal/wire` renames by walking the document beside the value it was encoded from, and a document kept as raw bytes leaves that walk — so the posted keys travel unchanged under **both** profiles. It is a second face of [§5.9](behaviours.md#59-an-unknown-capabilities-property-survives-into-sessions-here-and-not-there) rather than a new divergence: a server that keeps bytes it never parsed cannot rename what it did not read | Echoes the posted keys verbatim, and the wire sweep reports them rather than being exempted from them | 002 plan §6.10 | Yes — post a camelCase declaration, then read `GET /Sessions` under the camelCase profile |
| U-38 | **`activeWithinSeconds` above `Int32`.** The reference declares it `int?` `[source: Jellyfin.Api/Controllers/SessionController.cs:58 @ v10.11.11]`, so `4611686018427387904` is a binder `400` there and the unfiltered list here | Binds at Go's `int`, which is the safe direction — no request that succeeds there meets a refusal here — and narrowing would make the domain's saturating window unreachable from the wire. Asserted as a divergence at `TestAnActiveWithinSecondsAboveInt32IsAcceptedHereAndTheReferencesBinderRefusesIt` | 002 plan §6.10 | Rides U-17 |
| U-39 | **What `/Users/Configuration` advertises in `Allow`, and what a `GET` of it answers.** chi's own `MethodNotAllowed` names **both** `POST` and `GET` for that path `[measurement: github.com/go-chi/chi/v5 v5.3.2, Go 1.27.0, 2026-09-03]` — the `GET` coming from `/Users/{userId}`, a different row of a different path that matches the same request path — and the reference raises its `405` from a candidate set built the same way, so `GET, POST` is the likelier answer there. The second half is not a `405` on either server: chi searches the static subtree, finds no handler for the method and continues into the parameter node, so the identifier is the literal string `Configuration` | Answers `Allow: POST`, from the refusal table, because 002 §3.6 asks for the methods **that path** has; and a `GET` of it is the empty `401`, because the credential is read before the identifier is bound. Both are asserted, with the reasoning, in `conformance/method_refusals_test.go` | 002 T17 | Yes, and one pair of requests settles both halves |
| U-40 | **Whether a re-authentication clears a client's posted capabilities.** 002 plan §6.5 leaves `capabilities_document`, `created_at` and `last_playback_check_in_at` alone when it updates a session row, and nothing records what the reference does with the first of the three | Keeps them: a client that logs in again still sees what it declared | 002 plan §6.5 | Yes — post capabilities, authenticate again from the same device, read `GET /Sessions` |
| U-41 | **Whether any client branches on `GET /System/Info`'s `WebSocketPortNumber`** — which is [001's OQ-1](../../specs/001-server-identity-and-discovery/spec.md#7-open-questions) and is why this is a row rather than an edit. The route has **no** [allowlist.yaml](allowlist.yaml) entry at all. Seven of its twenty-six members are `installation-path`, which is a declared class and therefore work rather than a question; the port the operating system chose is none of the four classes, so an entry for it would fail that file's own load and it needs either a fifth class — *"not added without review"* — or a `behaviours §N` of the kind [§4.5](behaviours.md#45-systeminfo-answers-four-fields-with-what-is-true-here-not-with-the-references-constants) already is | Reports the port this process is listening on. [request-cases.yaml](request-cases.yaml) already describes the seven paths in prose as *"triage rather than allowlist rows"* and does not mention the eighth member | 001 §3.2; found at 002 T21 and left as a note at T22, because it is 001's route, 001's argument and 010's file | Yes, and it is the same run that writes the rows |
| U-42 | **The `.ignore` rule's real shape, which is three rules where [003 §3.2](../../specs/003-library-configuration-and-scanning/spec.md#32-what-is-considered-a-media-file) named one.** The reference searches for the marker from a file's own directory **upwards to the filesystem root**; an empty or whitespace-only marker excludes everything beneath the directory holding it; and a **non-empty** one is a set of `.gitignore`-style patterns of which only the matches are excluded, with the fallback that a marker whose every pattern fails to parse excludes everything `[source: Emby.Server.Implementations/Library/DotIgnoreIgnoreRule.cs:18-30,41-68,95-131 @ v10.11.11]`. No probe in this repository has sent a `.ignore` file of any kind | Excludes on an empty or whitespace-only marker found at or **under the library root**, never above it, and treats a non-empty marker as excluding nothing. Both narrowings show more than the reference rather than less, which is [§3.0.3](behaviours.md#303-the-shape-of-a-safe-divergence)'s safe direction for a scanner | 003 plan §6.1, and 003 §3.2's 2026-09-05 amendment | Yes, and one library settles all three: an empty marker, a marker one directory up, and a marker holding one pattern |
| U-43 | **The multi-part marker vocabulary.** [003 §3.3](../../specs/003-library-configuration-and-scanning/spec.md#33-movies) wrote it as `part1`/`pt1`/`cd1`/`disc1` *"and the `-a`/`-b` form"*; the reference's stacking rules take `cd`, `dvd`, `part`, `pt`, `disc` or `disk`, followed by digits **or by a single letter `a`–`d` after that same word** `[source: Emby.Naming/Common/NamingOptions.cs:141-145 @ v10.11.11]` — so a bare trailing letter stacks nothing there. Read from source; neither shape has been sent to a running reference | Implements the source's vocabulary. The alternative was the one reading in this feature that **loses** an item: two films merged into one, the second gone | 003 plan §6.2, and 003 §3.3's 2026-09-05 amendment | Yes — two files differing only by ` - a` and ` - b`, and two more differing by ` - cda` and ` - cdb` |
| U-44 | **`EnableCaseSensitiveItemIds`'s default, and that it is the server's rather than a library's.** It is declared `= true` `[source: MediaBrowser.Model/Configuration/ServerConfiguration.cs:89 @ v10.11.11]` and read as a server-wide setting `[source: Emby.Server.Implementations/Library/LibraryManager.cs:650 @ v10.11.11]`. This is [003's OQ-9](../../specs/003-library-configuration-and-scanning/spec.md#7-open-questions), which asked exactly this and had nothing to answer it with; the one server ever measured has the flag **set**, so it cannot answer either. A source reading is not a measurement | Defaults to case-**insensitive**, per library, frozen at creation — 003 §3.6's own decision, which never claimed to match a default. The identifiers differ from the reference's either way ([§1.4](behaviours.md#14-item-identifiers-are-32-lowercase-hex-characters)), but the **item count** does not: two files differing only in case are one item here and two there, and it is one of the forty-seven differences 003 declares over its own fixture | 003 plan §6.3, and 003 OQ-9's 2026-09-05 note | Yes — two files differing only in case, in one library, on a server whose setting has not been touched |

**A warning for [010](../../specs/010-conformance-harness/spec.md) that is not a row, because it is
about the instrument rather than about the reference.** A differential run reads headers with some
library, and **a header reader that returns one field line per name cannot see a repeated header**.
That is not hypothetical: this project's own recorded measurement of chi's `Allow` said it named
one arbitrary method, and re-measuring found it names **both, in two `Allow` field lines**, in Go
map-iteration order — `GET, POST` 171 times and `POST, GET` 29 over 200 identical requests
`[measurement: github.com/go-chi/chi/v5 v5.3.2, Go 1.27.0, 2026-09-03]`. The first reading is
exactly what an instrument of that kind sees of two field lines. The same kind of instrument reads
the reference. **A repeated header anywhere in the v1 surface would be invisible to a differential
run made that way**, and the check is one line: read field *lines*, not a joined value, and assert
their count.

Nothing in this table blocks feature 001, and none of it is a defect in the implementation: every
row is a decision taken deliberately, argued where it was taken, and reachable from here. What each
row buys is that 010's differential run meets it as a **declared** difference rather than as a
surprise — which is the shape [AGENTS.md §3](../../AGENTS.md) asks every conformance assertion to
take.

**U-17 and U-18 were added the same day, in the change that declared `GET /Sessions`' three request
parameters**, and they are the smallest rows here: two of them are settled by four requests that write
nothing and lock nobody out. They are also the first rows a *specification* points at — the `⚠️ UNVERIFIED`
marks live in 002 §3.8 rather than in a plan or beside a constant, because what is unmeasured is
an answer the route has to give rather than a mechanism somebody chose. [The constitution](../constitution.md)
makes such a mark block a specification from leaving draft; 002's `status:` is [a statement about the
exporting project](../../PROVENANCE.md) and not a gate this project has passed, so what the marks do
here is what they are for — they block, and the block is recorded rather than argued away.

**U-13 to U-16 were added on 2026-09-03, while 002's plan was written, and they are the same four
shapes one feature further on.** The sentence above holds for them too — none blocks 002 — with one
qualification worth making rather than leaving to be noticed: **U-14 is the first row where what
this project does today is not a different answer but a different *act*.** A `userId` this
specification does not mention is ignored under [§1.12](behaviours.md#112-an-unrecognised-query-value-is-ignored-not-rejected),
so an administrator asking to change somebody else's configuration changes their own — a silent
write to the wrong account rather than a status a client can branch on. It is still recorded rather
than acted on, for the reason every row here is, but it is the row to measure first.

**U-19 to U-41 were added on 2026-09-03, at 002's closing audit, and they are why this register
exists.** Thirteen of 002's twenty-two implementing tasks ended a handover note with some form of
*"this belongs in the register"*; every one of those claims had been written down where it arose —
in a `plan.md` section, beside a constant, or in a doc comment — and none of them was anywhere a
differential run would look. Collected, they are one run against a reference and rather less than an
afternoon: **eight of the twenty-three ride a request another row already sends**, and only four
need a fixture of their own.

Three things about this batch are worth stating rather than leaving to be counted.

**U-19 is the row to measure first**, and it displaces U-14 from that position. It is a difference on
an **authenticated** path, reachable by a request every client is capable of sending, and it exists
because a plan generalised from measured pairs: [002 plan §6.1](../../specs/002-authentication-users-and-sessions/plan.md#61-token-extraction)'s
*"a header that is present but yields nothing does not stop the search"* is true of every pair the
probe sent and unproven of the pair it did not. 002 T18 narrowed it as far as reading can: rewriting
the reader to stop at a present `Authorization` leaves all six precedence pairs green and fails only
four rows, of which this is the only candidate difference, because the other three mechanisms are
read from their own fields on both servers.

**Six of these rows are asserted in the code as divergences rather than left in a comment**, which
is a habit this feature settled into and is worth keeping: U-28, U-32, U-34, U-37, U-38 and U-15
each have a named test asserting *what this server does*, with the reference's source citation
beside it, so the day the probe runs the answer arrives as a failing test naming the decision rather
than as a rediscovery. **U-15's assertion was written at this audit and not before**, and the reason
is the general one the audit found: [002 §3.5](../../specs/002-authentication-users-and-sessions/spec.md#35-the-user-object)'s
*"all twenty-eight gate features v1 does not have"* was corrected by being **narrowed** to
twenty-six, and a claim corrected by narrowing has not been corrected until the narrower claim is
tested.

**U-24 and U-35 are absent members rather than differing values, and an allowlist row does not
excuse an absence.** Both are on bodies this feature owns, one of them the project's only `L3` body.
A `wall-clock` entry for `/LastActivityDate` says the value may differ; it does not say the member
may be missing.

**U-42 to U-44 were added on 2026-09-05, while 003's plan was written, and they are the first rows
about something other than a response.** 003 serves no route, so none of the three is a body a
differential run compares — each is a rule that decides **what a library contains**, and the run
that settles them is not a request but a scan of a tree over the single-use instance
[ADR-0007](../decisions/0007-a-container-runtime-for-the-reference-instance.md) already stands up
and destroys. Two things about them are worth stating rather than leaving to be counted.

**All three were found by reading source in order to write a plan, and each contradicted a sentence
of an `Accepted` specification.** That is the shape 002's batch had — a plan reading the reference to
implement a spec and finding the spec narrower than the code it describes — and the response is the
same: the specification is amended to say what it means, the reading is registered, and nothing
about what Atrium does is moved on source evidence alone
([AGENTS.md §1.3](../../AGENTS.md)). Two of 003's amendments are on the same date as this batch and
name these rows.

**U-43 is the row to measure first of the three, and the reason is not that it is the likeliest.**
It is the only one whose wrong reading **loses an item**: a bare trailing letter read as a part
marker merges two films into one and makes the second disappear, where U-42 and U-44 each show a
user *more* than the reference does. A difference that hides a work is worse than one that reveals a
theme tune.

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
