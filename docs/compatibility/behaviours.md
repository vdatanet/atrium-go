# Measured behaviours

**Last verified: 2026-08-28, against Jellyfin 10.11.11.**

This is the registry required by Principle V. Every entry has the same three fields:

- **Jellyfin does** — the observed behaviour, with provenance.
- **Depends on it** — whether a known client relies on it.
- **Atrium does** — our decision, and why if it differs.

The default answer in the third field is *the same thing*. Divergence needs an argument.

Entries are grouped by how deep they cut. §1 is the wire format — get one of these wrong and
nothing works. §2 is semantics. §3 is defects. §4 is the deliberate exceptions, §5 the gaps v1 accepts, §6 the good ideas we refuse.

---

## 1. Wire format

### 1.1 Property casing is PascalCase

**Jellyfin does:** serialises every JSON property in PascalCase — `ItemId`, `RunTimeTicks`,
`TotalRecordCount`, `IsFavorite`. It additionally advertises `application/json; profile="CamelCase"`
and `application/json; profile="PascalCase"` content types alongside plain `application/json`,
all pointing at the same schema — on 367 of the document's 370 JSON responses, two of which add
`text/css` beside them. The other three declare `application/json` alone:
`GetNamedConfiguration`, `GetSchedulesDirectCountries`, and a plugin's `TmdbClientConfiguration`.
*"Every"* is therefore very nearly true and not quite, which is worth knowing before writing a
client that counts on it. `[spec: every JSON response in the 10.11.11 document, counted 2026-09-01]`

The same schema, and **not** the same serialisation: the CamelCase profile really does emit
camelCase. That is §1.13, and this entry is about what a client gets when it asks for nothing in
particular — which is PascalCase.

**Depends on it:** everything. A camelCase body is not a lesser response, it is an empty object to
a client's decoder.

**Atrium does:** the same. This is a whole-project constraint, not a per-endpoint one: the
serialisation layer emits PascalCase by default and it must be impossible for a route author to
forget. Requests are parsed case-insensitively, because Jellyfin's model binder is.

> ⚠️ This is the single most likely source of a silent, total incompatibility, because Python's
> ecosystem defaults to snake_case everywhere. It gets a conformance test that walks every
> registered response model and fails on any non-PascalCase field name.

**Amended 2026-09-03, at 001's closing audit. *"Every"* is 1003 of 1026, and the exceptions are in
two places.** Writing the sweep meant writing the rule down, and a rule has to be checked against
the document rather than against intuition. **23 of the 1026 property names the pinned document
declares are not PascalCase**: five are RFC 7807's problem-details members (`type`, `title`,
`status`, `detail`, `instance`) and eighteen belong to the plugin repository manifest. **None is on
a v1 route**, so nothing in this project's surface changes and the *"Atrium does: the same"* above
stands as written.
`[spec: docs/compatibility/property-names.json, extracted from the pinned 10.11.11 document,
counted 2026-09-03]`

The finding that matters for anyone implementing the sweep is the shape of the rule rather than the
count. **A rule spelled "a capital, then lower-case letters, repeated" is wrong**: the document
contains `EnableIPv4`, `Video3DFormat` and `Hdr10PlusPresentFlag`, all of which it refuses. A sweep
written that way fires on real names, and whoever meets it loosens it — which is how a check that
was meant to catch a camelCase leak stops catching anything. The rule this project runs is measured
against all 1026 rather than argued: **the first character is upper-case, and every character after
it is a letter or a digit** — which admits the three names above and refuses an underscore, a
hyphen and a leading lower-case letter alike.

### 1.2 Dates carry up to seven fractional digits

**Jellyfin does:** emits .NET round-trip ISO-8601, e.g. `2025-06-19T00:00:00.0000000Z` — seven
fractional digits, which is more than the three that most ISO-8601 parsers accept. A strict parser
rejects it outright. `[prior-probe: Jellyfin 10.11.11, 2026-06-19]`

**Depends on it:** clients have already built tolerance for this (the tvOS client ships a custom
transcoder for exactly this reason), so emitting three digits would not break them. But emitting
*seven* is what the differential harness compares against.

**Atrium does:** emits seven fractional digits and a `Z` suffix. Accepts anything ISO-8601 on input,
with or without a timezone; a missing timezone is read as UTC.

> ⚠️ **Seven is the maximum and the usual case, not a constant — measured 2026-09-02**, when this
> claim's probe was finally written and the `prior-probe` discharged. Over 352 date values on eleven
> routes, **346 carry seven fractional digits and six do not**
> `[probe: tools/probe_wire_format, Jellyfin 10.11.11, 2026-09-02]`:
>
> | Field | Observed |
> |---|---|
> | `LastPlayedDate` | 7 digits x104, **6 digits x3**, **3 digits x1** — `2026-08-13T14:01:13.061Z` |
> | `LastActivityDate` | 7 digits x12, **6 digits x2** — `2026-09-02T17:58:58.632188Z` |
> | `DateCreated`, `DateLastMediaAdded`, `EndDate`, `LastLoginDate`, `LastPausedDate`, `LastPlaybackCheckIn`, `PremiereDate` | 7 digits, always |
>
> **The mechanism is not established here**, and this is recorded as a measurement without one — the
> weaker kind of entry, marked as such, the way [§1.7](#17-a-null-property-is-absent-everywhere-by-one-setting)'s
> exception is. The obvious explanation does not survive its own evidence: every short value is one
> whose remaining digits would be trailing zeros (`.061` is `.0610000`, `.055673` is `.0556730`), which
> looks like trailing-zero trimming — but `.0000000` is written **in full** on seven other fields, and
> a rule that trims trailing zeros would have emitted no fraction at all there.
>
> **What this changes for Atrium: nothing, and that is the point.** Emitting seven digits always is
> still right — it is what 346 of 352 values do, it is what a client parses either way, and inventing
> the short form would mean reproducing a mechanism nobody has identified. What changes is the
> **conformance** claim: a golden that asserts seven digits on `LastPlayedDate` or `LastActivityDate`
> is asserting something the reference does not always do, and a differential comparing those two
> fields will see a difference that is real and is the reference's.

> **`0001-01-01T00:00:00.0000000Z` is how an unset date arrives**, on `DateCreated`,
> `DateLastMediaAdded` and `LastPlaybackCheckIn` in the same run. It is .NET's `DateTime.MinValue`,
> and it is emitted rather than omitted — so §1.7's "a null property is absent" does not cover it:
> the value is not null, it is the zero of its type.

### 1.3 Durations and positions are .NET ticks

**Jellyfin does:** expresses `RunTimeTicks`, `PositionTicks`, `PlaybackPositionTicks` and
`StartPositionTicks` in **ticks of 100 nanoseconds** — 10,000,000 ticks per second. The unit is
.NET's `TimeSpan` tick: ffprobe's seconds become `TimeSpan.FromSeconds(…).Ticks` at ingestion
`[source: MediaBrowser.MediaEncoding/Probing/ProbeResultNormalizer.cs:234 @ v10.11.11]`.

**Depends on it:** every progress bar and resume position.

**Atrium does:** the same. Internally, durations are stored in ticks, not seconds, so no conversion
can be forgotten at a boundary. Where a source (ffprobe) reports seconds as a float, the conversion
happens once, at ingestion, and rounds rather than truncates.

### 1.4 Item identifiers are 32 lowercase hex characters

**Jellyfin does:** serialises GUIDs in .NET's `"N"` format — 32 hex characters, no dashes, e.g.
`0d41983a5d18d53282f56e7460e2c2cd`. Ids are stable across rescans.
`[prior-probe: Jellyfin 10.11.11, 2026-06-13]`

They are derived deterministically:

```
guid = DotNetGuidFromBytes( MD5( UTF16LE( type.FullName + key ) ) )
```

where `key` is the item's path, lowercased unless `EnableCaseSensitiveItemIds` is set, and
`DotNetGuidFromBytes` applies .NET's mixed-endian layout (first four bytes little-endian, then two,
then two, then the remaining eight in order).
`[source: Emby.Server.Implementations/Library/LibraryManager.cs:636 @ v10.11.11;
MediaBrowser.Common/Extensions/BaseExtensions.cs:30 @ v10.11.11]`

**Confirmed from outside, which a source citation cannot do on its own.** Recomputing that
expression from each item's own reported `Path` reproduced the `Id` the server returned for
**448 of 448** items across `Movie`, `Episode`, `Audio`, `Series` and `MusicAlbum`, and 447 of the
447 whose path contains an uppercase character reproduced from the path **verbatim** — which is
what `EnableCaseSensitiveItemIds`, set on that server, means.
`[probe: tools/probe_item_identity.py, Jellyfin 10.11.11, 2026-08-27]`

Two things follow, and the second is the one 003 is built around:

- **Containers are path-keyed too.** A `Series` and a `MusicAlbum` derive from their *directory*,
  not from their name, so they move with everything else.
- **The key is the path and nothing else** — not the library, not a stored row. So moving a library
  root **changes every identifier under it**, and every client's favourites and resume positions
  for everything in that library are silently discarded. The only symptom is a user saying their
  library looks wrong.

**The reference's *default* for `EnableCaseSensitiveItemIds` is not measured here**, and no claim
in this repository that names it carries provenance. The server used above has the flag set, so it
cannot answer the question; a server with it unset would. The probe says so in its own output
rather than letting the measured value stand in for a default.

**Depends on it:** clients key their caches, favourites and resume positions on these strings. A
client's stored state survives a server rescan *because* the ids are derived, not sequential.

**Atrium does:** the same **shape** and the same **stability guarantee**, using a deterministic
derivation from the item's stable identity (Principle VII).

Reproducing Jellyfin's *exact* bytes for the same file is **not a goal** — it would require
matching a C# type's `FullName`, which is an implementation detail of a codebase we do not fork
(Principle IV). Atrium's derivation is its own, documented in the library specification. Any
client that assumes a particular id for a particular file is already broken against Jellyfin, which
changes ids when `EnableCaseSensitiveItemIds` flips.

**Atrium keys on the path *relative to its library root*** ([003 §3.6](../../specs/003-library-configuration-and-scanning/spec.md#36-identity)),
so the root move above costs nothing —
`tests/library/test_root_move.py` moves a scanned library to another mount, reconfigures the root
and rescans, and asserts every identifier unchanged and no user data orphaned. That is a
divergence from the reference in **behaviour nobody can observe**: the ids differ from the
reference's either way, so there is no client that could tell.

### 1.5 List responses carry `StartIndex`

**Jellyfin does:** returns `{"Items": [...], "TotalRecordCount": n, "StartIndex": i}`. Emby omits
`StartIndex`. Confirmed across all ten envelope-returning endpoints of the v1 surface.
`[probe: tools/probe_query_envelope.py, Jellyfin 10.11.11, 2026-08-26]`

**Depends on it:** no known client reads it (pagination is driven by the request), but its absence
is a visible difference.

**Atrium does:** includes it.

### 1.6 `Container` at item level is a list for some formats, and the single form is per response

**Jellyfin does:** report one normalised container string at item level, which names a single
container for some formats and lists several for others. A `.mp4` and a `.m4a` both answer
`"mov,mp4,m4a,3gp,3g2,mj2"`; a `.mkv` answers `"mkv"`, an `.avi` `"avi"`, a `.flac` `"flac"`
`[probe: tools/probe_media_container.py, Jellyfin 10.11.11, 2026-08-29]`. It is **not** ffprobe's
`format_name` verbatim: `matroska` is renamed and `webm` dropped where the streams disqualify it
`[source: MediaBrowser.MediaEncoding/Probing/ProbeResultNormalizer.cs:124,270-315 @ v10.11.11]`.

The single container on a `MediaSource` is derived from that string per response, and the two
routes derive it differently. On a listing **no profile is involved**: it is the file's own
extension where the list contains it — so the same list answers `mp4` for a `.mp4` and `m4a` for a
`.m4a` — and the list's first member where it does not
`[source: Emby.Server.Implementations/Dto/DtoService.cs:316-352 @ v10.11.11]`. In a negotiation it
is the first member the `DeviceProfile` accepts, and a **profile-less** `PlaybackInfo` passes the
list through untouched — the same `.m4a` answers `m4a` on `/Items` and the full list on
`PlaybackInfo` `[probe: tools/probe_media_container.py, Jellyfin 10.11.11, 2026-08-29]`, `[probe:
tools/probe_playback_info.py, Jellyfin 10.11.11, 2026-08-28]`.

Two details of the listing branch, read at 008 T3 from the same source line: the membership test is
**case-insensitive** and the value kept is the *path's* own spelling, so a file named `.MP4`
answers `MP4`; and a container string that already names one format is passed through rather than
resolved again, which is why a `.mkv` answers `mkv` at both levels.

*The 2026-06-13 measurement recorded here — "the item level is ffprobe's format-name list, the
real container is on the `MediaSource`" `[prior-probe: Jellyfin 10.11.11, 2026-06-13]` — was made
on an mp4 and is true of one. It is kept rather than deleted: it did not fail to reproduce, it
generalised wrongly.*

**Depends on it:** clients that pick a player by container have already learned to read the
`MediaSource`, and a client reading the item-level field of an mp4 expects the list form.

**Atrium does:** the same, from one stored string. Inspection stores the normalised container and
what the demuxer said before it, and each single form is derived where the response is built
([008 §3.1](../../specs/008-playback-negotiation-and-delivery/spec.md#31-media-sources), plan §4).

### 1.7 A null property is absent, everywhere, by one setting

**Jellyfin does:** omit any property whose value is null. Not per-property and not a judgement —
its whole JSON pipeline is configured with
`DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull`. [source: src/Jellyfin.Extensions/Json/JsonDefaults.cs:33, Jellyfin.Server/Extensions/ApiServiceCollectionExtensions.cs:148 @ v10.11.11]

Measured too: `/System/Info` declares `PackageName` in its schema and does not send it. `[probe: tools/probe_public_info.py, Jellyfin 10.11.11, 2026-08-28]`

**Except that two properties survive it.** `ChannelId` arrives as an explicit `null` on **every
item of every type**, in list rows and in single-item bodies alike — 208 observations across nine
content types — and `ParentId` arrives as an explicit `null` on the `/UserViews` rows that have no
parent. `[probe: tools/probe_item_shapes.py, Jellyfin 10.11.11, 2026-08-27]`

```json
"OfficialRating": "ES-18",
"ChannelId": null,
"CommunityRating": 6.7,
```

**What overrides the setting has not been established here.** The configuration cited above is not
in dispute; something defeats it for these two properties, and identifying it needs a reading of
the reference's source that had not been taken as of 2026-08-28 — the machine that ran the probe
had no reference checkout. The exception is therefore recorded as a measurement
with no mechanism, which is weaker than the rest of this entry and is marked as such.

**Depends on it:** decoders differ. A generated Swift client distinguishes "absent" from "null"
only when the schema is nullable; a hand-written Kotlin one usually does not — and the exception
is where that bites hardest, because a client which draws the distinction sees `ChannelId: null`
on every row of every list, the highest-traffic response in the API.

**Atrium does:** the same, in the base model rather than per route — with `ChannelId` emitted
explicitly as `null` on every item, and `ParentId` likewise where it has no parent. A
`response_model_exclude_none` flag on every route is one someone eventually forgets, and the one
they forget is the one a client sees a stray `null` on; the two exceptions are named in one place
for the same reason.

> **This entry previously read "per-property and not consistent", marked ⚠️ UNVERIFIED**, and
> planned to let the differential harness enumerate it. It is one line of configuration. The
> assumption was more complicated than the truth, which is worth remembering the next time
> something looks like it needs a harness to work out.
>
> **And then it was measured, on 2026-08-27, and the exception it had ruled out turned out to
> exist after all** — narrower than the original "per-property and not consistent", but real, and
> found only because a probe counted nulls it was not looking for. The line of configuration was
> the right answer to "how is this done" and the wrong answer to "does it hold everywhere".

### 1.8 `GET /Items/Latest` returns a bare array

**Jellyfin does:** answers with a JSON array of items, **not** the `{Items, TotalRecordCount,
StartIndex}` envelope every other list endpoint uses. `/Items/Filters` is a third shape again
(`{Genres, Tags, OfficialRatings, Years}`), and `/Search/Hints` a fourth
(`{SearchHints, TotalRecordCount}`). `[probe: tools/probe_query_envelope.py, Jellyfin 10.11.11, 2026-08-26]`

**Depends on it:** completely. A client decoding a bare array as an envelope gets nothing at all —
not a degraded result, an empty one. This asymmetry is the reason the probe that measured it was
worth writing before any code.

**Atrium does:** the same four shapes, per endpoint, never normalised into one.

### 1.16 Every non-ASCII character in a body is escaped, and so are seven ASCII ones

**Jellyfin does:** serialise with ASP.NET Core's HTML-safe `JavaScriptEncoder`, which writes every
non-ASCII character and seven ASCII ones as `\uXXXX` with **uppercase** hex. `28 años después`
goes out as `28 a\u00F1os despu\u00E9s`; `Abraham's Boys` as `Abraham\u0027s Boys`.
`[probe: tools/probe_query_envelope.py, Jellyfin 10.11.11, 2026-08-28]`

| Escaped | Left literal |
|---|---|
| every non-ASCII character | `/` `=` `:` space `!` `*` `(` `)` `-` `_` |
| `"` → `\u0022`, `&` → `\u0026`, `'` → `\u0027`, `+` → `\u002B` | |
| `<` → `\u003C`, `>` → `\u003E`, `` ` `` → `\u0060` | |

Note `"` : JSON's own escape is `\"`, and the reference does not use it.

**How it was measured is the interesting part.** Item names prove only what the library happens to
contain — a Spanish film catalogue gives `\u00F1` and `\u0027` and says nothing about `<` or a
backtick. The exact set came from **echoing arbitrary characters back through a validation error**:
`?limit=a<b>c\`d'e&f+g"h/i=j:k` answers `400` with the value quoted in `errors`, which is the one
route that puts client-supplied text into a response body.

**Depends on it:** no client can tell. A JSON parser decodes `\u00F1` and `ñ` to the same string,
so nothing branches on this and Principle I does not require it.

**Atrium does:** the same, in `compat/responses.py` — **which reversed §4.4**, an exception taken
one feature earlier on the argument that the upper-casing could not be done safely. It can: the
rewrite counts backslash parity rather than searching for `\u`. §4.4 is marked withdrawn and kept
as the record. Not for Principle I but for **Principle VIII**:
the goldens compare bytes, and a library with accented titles would otherwise differ from the
reference on nearly every response while being correct in every field. One override in the response
class is cheaper than an asterisk on every golden and a permanent exception in the differential.

> **The one hard case is a literal.** A *value* containing the six characters `\u00e9` must survive
> as those six characters while the encoder's own escapes are uppercased, so the rewrite counts
> backslash parity rather than searching for `\u`. `json.dumps` has already doubled every literal
> backslash by then, which is what makes the parity exact.

---

## 2. Semantics

### 2.1 `UserData` is always present

**Jellyfin does:** returns `UserData` on every item without `Fields=UserData` or
`EnableUserData=true`, and includes `Key` and `ItemId` inside it (Emby does not).
`[probe: tools/probe_playstate.py, Jellyfin 10.11.11, 2026-08-28]`

**Depends on it:** the played ticks and progress bars a client draws on every list — the object
arrives unasked, so clients never learned to ask for it. `Key` and `ItemId` inside it have no
reader in either analysed client; the pair is a dialect marker, present to be present
(007 §3.2's survey).

**Atrium does:** the same.

### 2.2 `/Users/Public` can legitimately be empty

**Jellyfin does:** honours each user's "hidden from login screens" policy flag and returns `200` with
`[]` when every user is hidden. `[probe: tools/probe_public_users.py, Jellyfin 10.11.11, 2026-09-02]`

**And `[]` is what a server answers before anybody has configured anything.** Measured on
2026-09-02, when the claim was finally reproduced against an instance this project could hide
accounts on: `IsHidden` is **true** on the administrator the setup wizard creates and on every
account `POST /Users/New` creates — the permission is added set
`[source: Jellyfin.Data/UserEntityExtensions.cs:174 @ v10.11.11]` — so a freshly installed
reference answers `200 []` with two accounts on it. Un-hidden, the same two accounts are two rows;
hidden again, one row and then none. The flag drives the route in both directions, which is the
half this entry always claimed; what is new is that the empty answer is the **default state** and
not a hardened one. The route is read with **no credential**, which is who calls it: two of its
four filters read the caller — the device comes from the token and the network from the remote
address `[source: Jellyfin.Api/Controllers/UserController.cs:635-651 @ v10.11.11]` — so an
administrator's token turns the question into one about that administrator's device.

**Depends on it:** clients must fall back to a username field. This is client behaviour, not server
behaviour, but a server that "helpfully" returned hidden users would leak them. The 2026-09-02
reading raises the stakes: a client that treats `[]` as *"this server has no users"* breaks against
**every** reference nobody has configured, not against a rare hardened one.

**Atrium does:** the same, including the policy flag.

> **The default differs, and it is 002's to decide.** Atrium's user record defaults
> `is_hidden` to `false` (`src/atrium/domain/user.py`, and the `0` server default of the
> `is_hidden` column), where the reference adds the permission set. So a first account on each
> server answers a different `/Users/Public` — one row here, none there — which is a difference a
> login screen can see. 010 triages and does not decide (
> [010 §2](../../specs/010-conformance-harness/spec.md#2-scope)); the endpoint is
> [002 §3.4](../../specs/002-authentication-users-and-sessions/spec.md)'s, and this is the
> procedure of §3.0 waiting for it. Nothing about the flag's *effect* differs: both servers
> exclude a hidden user and answer `200` with `[]` when every account is hidden.

### 2.3 `LocalAddress` is one string, and may be HTTPS

**Jellyfin does:** returns a single `LocalAddress` string (Emby returns `LocalAddresses[]`), chosen
by matching the requester's network — so over a VPN it returns the VPN-side address. When a
certificate is configured it advertises the **HTTPS scheme and port**, regardless of the scheme the
request came in on. `[probe: tools/probe_local_address.py, Jellyfin 10.11.11, 2026-09-02]`

**Reproduced exactly on 2026-09-02**, on an instance this project stood up, gave a self-signed
certificate and restarted: the same route over the same plain-HTTP request answers
`http://<address>:8096` before the certificate and `https://<address>:8920` after it — the scheme
**and** the port, on `/System/Info/Public` and on `/System/Info` alike. The condition is
`EnableHttps` **and** a certificate that actually loaded
`[source: Emby.Server.Implementations/ApplicationHost.cs:267 @ v10.11.11]`, and it is read while
the host is being built `[source: Emby.Server.Implementations/ApplicationHost.cs:457-458 @
v10.11.11]` — a configuration change validates the new path and asks for a restart
`[source: Emby.Server.Implementations/ApplicationHost.cs:761-764, 779-797 @ v10.11.11]`, so a
server configured in place goes on advertising the scheme it started with until it comes back.

> ⚠️ **The override is the last of four branches, and this entry described it as the rule —
> corrected 2026-09-02.** An operator's server with `EnableHttps` on and a valid public certificate
> answers `LocalAddress: http://192.168.1.39:7096` on `/System/Info/Public`, plain HTTP, against a
> request that arrived over TLS `[probe: manual request against an operator's server, Jellyfin
> 10.11.11, 2026-09-02]`. The rule as written says that cannot happen. Read at the pinned tag, four
> things can produce this value and only the last consults `ListenWithHttps`:
>
> | # | Condition | Produces | `ListenWithHttps`? |
> |---|---|---|---|
> | 1 | `PublishedServerUrl` is set | that value, verbatim | **No** |
> | 2 | `EnablePublishedServerUriByRequest` is on | the **request's own** scheme and host | **No** |
> | 3 | `PublishedServerUriBySubnet` has an entry matching **the caller's** subnet | that URI, verbatim | **No** |
> | 4 | otherwise | the bind address, with the scheme defaulted | **Yes** |
>
> `[source: Emby.Server.Implementations/ApplicationHost.cs:871-901, 929-947 @ v10.11.11;
> src/Jellyfin.Networking/Manager/NetworkManager.cs:830-855, 995-1031 @ v10.11.11]`
>
> **Two separate short-circuits, and the second is the subtle one.** `GetLocalApiUrl` defaults the
> scheme with `scheme ??= ListenWithHttps ? Uri.UriSchemeHttps : Uri.UriSchemeHttp`, so a scheme
> **passed in** is never overridden — that is branch 2. But branch 3 never reaches that line at all:
> `MatchesPublishedServerUrl` returns a **whole URI**, and `GetLocalApiUrl` opens with
> *"If the smartAPI doesn't start with http then treat it as a host or ip"* and returns it
> untouched. A configured override is not merged with the server's scheme; it replaces the answer.
>
> **Branch 3 is per-caller, which makes `LocalAddress` a function of who is asking.** Overrides are
> split into internal and external sets and filtered to those whose subnet contains the caller,
> most-specific prefix first. The observed server carries `192.168.1.0/24=http://192.168.1.39:7096`,
> and the request came from that subnet — **so this is branch 3, established rather than inferred**,
> and the same server answers something else to a caller from outside it.
>
> **The 2026-09-02 reproduction is not withdrawn.** It measured branch 4, on an instance this project
> stood up with none of the other three configured, and for that branch it holds exactly. What a
> default instance cannot show is that it is one branch of four.
>
> **And the footgun in the next paragraph is not hypothetical.** The operator of that server set the
> override *because DLNA streaming did not work*: Jellyfin handed a renderer an address it could not
> use, where Emby had handed over a usable one `[client-author: 2026-09-02]`. The subnet override is
> the workaround that cost them the debugging time this entry mentions — which is direct evidence for
> [§4.2](#42-localaddress-does-not-get-an-https-override), where Atrium returns the scheme it is
> actually reachable on and no operator has to discover this setting to make a renderer work.

**Depends on it:** clients that hand this address to a device without a TLS stack (a DLNA renderer)
break when it is HTTPS. This is a genuine footgun that has cost real debugging time.

**Atrium does:** returns a single string and selects by requester network, matching Jellyfin. It
does **not** replicate the HTTPS override: `LocalAddress` reflects the scheme the server is
actually reachable on for that network. ⚠️ **This is a deliberate divergence** — see §4.2.

### 2.4 There are five authentication mechanisms, and one of them wins

**Jellyfin does:** accept **five**.
[api-surface-v1.md §3](api-surface-v1.md#3-authentication-users-and-sessions) listed four until
2026-08-28, when this section's count was carried into it. The fifth is
`X-Emby-Authorization` carrying a `Token=` component: the reference reads that header and
`Authorization` with the same grammar, and a token in either authenticates. `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]` It is
the historical Emby form and it is what a great many clients send, so a server implementing only
the documented four would refuse clients that have worked against Jellyfin for years.

**Four of the five are measured on three route classes** — an authenticated API route, an image
route and a delivery route — and all four authenticate on each.
`[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-26]` On the image and streaming
routes the query forms are the only practical option, because those URLs are handed to players and
image loaders that set no headers.

**The fifth is measured on the API route, and on the other two classes it cannot be.**
`mechanisms()` sent four until 2026-08-28 and sends all five since, to every route class. The run
that followed settles the class that requires a token: on `GET /Users/Me`, which refuses a request
carrying none with `401`, **all five answer `200`**
`[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]`.

On the image and delivery routes that same run answers `200` for all five **and for no token at
all**, so the fifth's `200` there distinguishes nothing (§2.10). That is not a shortcoming of the
probe: it is the most a route requiring nothing can be asked. "All five work everywhere" is
therefore measured where it can be, and untestable where it cannot, for as long as those two
classes require no token.

When a request carries two that disagree, the order is **not** arbitrary. Measured pair by pair,
in both directions each time:

    Authorization  >  X-Emby-Authorization  >  X-Emby-Token  >  ?ApiKey= / ?api_key=

The two query spellings were never set against each other, and `Authorization` against a query
parameter is inferred from the chain rather than measured.

**Depends on it:** a client holding a stale token in one place and a fresh one in another, which
sounds contrived until you notice that clients set a header once when the connection is built and
assemble streaming and image URLs from a template. Resolving in a different order turns a working
request into a `401` for exactly those clients.

**Atrium does:** the same order. [002 plan §6.1](../../specs/002-authentication-users-and-sessions/plan.md#61-token-extraction)
fixed the opposite one and called it arbitrary, on the argument that it only had to be
deterministic. It had to be deterministic **and** the reference's.

### 2.12 The client header's grammar is stricter than "lenient" suggests

**Jellyfin does:** require a scheme word, and it is `MediaBrowser` or `Emby`, compared
case-insensitively. Anything else — `Bearer`, a made-up word, or no scheme at all — and nothing is
read out of the header. Within it, one row per variation, all measured: `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]`

| Variation | Reference |
|---|---|
| Values quoted, or bare | accepted |
| No space after a comma, or a space *before* one | accepted |
| Extra spaces after the scheme | accepted |
| Components in any order | accepted |
| An unknown component alongside | accepted |
| A trailing comma | accepted |
| **Whitespace around the `=`** | **`401`** |
| **A lowercase component name** (`token=`) | **`401`** |

**Depends on it:** the two refusals are the interesting half. No working client can be sending
either form today, because the reference refuses both.

**Atrium does:** the same, including the two refusals. Being kinder costs nothing today and lets
somebody build a client against Atrium that fails against Jellyfin, which is the direction that
matters — see §6.

### 2.14 A client's declared capabilities and the server's flags are different values

**Jellyfin does:** echo `SupportsMediaControl: true` back inside `Capabilities` for a session that
posted it, while reporting `SupportsMediaControl: false` at the **top level** of the same session.
`[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]` `SupportsRemoteControl` is `false` there too. `PlayableMediaTypes` and
`SupportedCommands`, by contrast, **are** hoisted from the declaration to the top level verbatim.

`POST /Sessions/Capabilities/Full` answers `204` with no body and **replaces** rather than merges —
the route is named `Full` and it behaves like it.

**Depends on it:** a client that reads the top-level flag to decide whether another session can be
controlled. Hoisting the declaration into it would tell every such client that a remote-control UI
will work, for every session whose client declared it, whether or not anything is listening.

**Atrium does:** the same. The declaration is the client's claim; the flag is the server's judgement
about it, and v1 answers `false` because it has no remote control — which
[002 §3.8](../../specs/002-authentication-users-and-sessions/spec.md#38-sessions) argued was honest
rather than a gap, and which is now measured to be **not a divergence at all**.

### 2.15 An audio file under a video root is not an item

**Jellyfin does:** admit a file only when its extension is on the list for its library's collection
type, and it does not fall back to another type's list. Under the `movies` and `tvshows` roots of a
library holding 8,288 items, 89 `.mp3` files and 3 `.mka` files produced **no item of any type** —
not a `Movie`, not an `Episode`, and not an `Audio` either. The same server admits `.flac`, `.m4a`
and `.dsf` under its `music` root, so the extensions are recognised; the collection type is what
refuses them. `[probe: tools/probe_library_extensions.py, Jellyfin 10.11.11, 2026-08-27]`

The measured admitted sets are `movies` `.mkv` `.mp4` `.avi` `.ts`; `tvshows` `.mkv` `.avi` `.mp4`;
`music` `.flac` `.m4a` `.dsf`. They are a **lower bound** — what one real library contained — not
the reference's configured lists, which the API does not expose.

**Depends on it:** yes, by absence. A film library that admitted every audio extension would return
theme music, commentary tracks and stray downloads as items the reference does not have, and the
user would see a library that disagrees with the one their other client shows. Absence is as
observable as presence.

**Atrium does:** the same. Extension lists are per collection type, and a file whose extension is
not on its own type's list is ignored silently ([003 §3.2](../../specs/003-library-configuration-and-scanning/spec.md#32-what-is-considered-a-media-file))
— never promoted to another type because some other list would accept it.

### 2.16 A music track's number comes from tags, never from its filename

**Jellyfin does:** take an `Audio` item's `IndexNumber` and `ParentIndexNumber` from the file's
embedded tag, or failing that from the number the container carries, and from nothing else. No
resolver and no provider reads a leading number off the filename.
`[source: MediaBrowser.Providers/MediaInfo/AudioFileProber.cs:181 @ v10.11.11]`
`[source: MediaBrowser.MediaEncoding/Probing/ProbeResultNormalizer.cs:1369 @ v10.11.11]`
A file whose tags supply no title keeps the name every item starts with — its **whole filename
stem**, leading digits included.
`[source: Emby.Server.Implementations/Library/ResolverHelper.cs:96 @ v10.11.11]`

So an untagged `01 - The Track.flac` resolves on the reference to an item named `01 - The Track`
with no track number at all, rather than to `The Track` as track one.

**Its own library cannot show this**, and that is worth stating because the numbers elsewhere in
this document came from it. All 5,814 of its tracks carry a title tag, so the fallback never fires
there, and not one of the 5,814 filenames begins with a digit glued straight to a letter.
`[read: Jellyfin 10.11.11, 2026-08-27, /Items?IncludeItemTypes=Audio&Fields=Path]` The 77.9%
path-versus-reference agreement on track numbers recorded at 003 T13 measures how often a **tag**
happened to match the filename beside it, not a path fallback the reference has.

**Depends on it:** unknown, and narrowly. Only a library holding untagged music can tell the two
apart; for a tagged file both servers answer from the tag and agree.

**Atrium does:** read a track number, a disc number and a title off the filename when its metadata
source says nothing — which for an untagged file is a value the reference does not have.
[003 §3.5](../../specs/003-library-configuration-and-scanning/spec.md#35-music) states the
divergence and **OQ-8 holds the decision open**: the evidence that would settle it is how much real
music carries no readable tag. Half of that moved on 2026-08-27 — 004 T7 built the tag reader, so
the fraction is measurable against any real library; what it still needs is a library that is not
this suite's generated silence (003 §7 OQ-8). Until that measurement is taken, the divergence
stands the way §5's gaps do: observable only in a library holding untagged music, with OQ-8's
measurement as the mechanism that closes it.

What is already settled is the direction of the tie-break inside that fallback. Since every stem
Atrium declines to find a number in is a stem it agrees with the reference about, an ambiguous shape
parses *less*: a digit is a track number only when a separator follows it, so `24K Magic.flac` is a
song called `24K Magic` and not track 24 of `K Magic`. `tests/corpus/naming.yaml` pins both that and
what it costs.
### 2.17 No item and no media source carries a modification time

**Jellyfin does:** expose `DateCreated` on an item and `Size` on a media source, and **no
modification time anywhere**. 120 `Movie`, `Episode` and `Audio` items requested with
`Fields=MediaSources,Path,Etag,DateCreated,DateLastMediaAdded` carried no property whose name
contains "modif" on the item or on the source; the only time-shaped properties were `DateCreated`,
`PremiereDate` and `RunTimeTicks`. `[probe: tools/probe_item_shapes.py, Jellyfin 10.11.11, 2026-08-28]` The
pinned document agrees: `DateModified` exists on `FontFile` and `LogFile` and on nothing else, and
the only other property whose name contains "modif" is `ChapterInfo.ImageDateModified` — neither an
item nor a media source. `[spec: components.schemas in the 10.11.11 document, re-measured 2026-09-01]`

**Depends on it:** no client can, because there is nothing there to depend on. That is the whole
value of the entry — it says which half of a scanner's change-detection signal is observable and
which is private. `Size` is observable, so an item whose file was replaced by one of a different
length **must** end up with the new number; a modification time is not, so which signal a server
uses to decide *whether to look* creates no delta whatever it is.

**Atrium does:** use `(size, mtime_ns)` as the signal and re-read the size of every file it
examines ([003 §3.8](../../specs/003-library-configuration-and-scanning/spec.md#38-scanning-and-change-detection),
[plan §6.4](../../specs/003-library-configuration-and-scanning/plan.md#64-change-detection)). The
modification time is stored and never serialised. A full re-examination that ignores the signal is
available to an operator, and it is unobservable for the same reason: it changes what the server
looks at, not what it answers.

### 2.18 Two spellings of one genre are one item

**Jellyfin does:** fold case when a name becomes a by-name item, so `Electronic` and `electronic`
on two files produce **one** genre row, not two. Measured from the outside: 97 of 97 live genre
and music-genre ids reproduce from the case-folded name, every one of them only because of the
fold; the measured library carries both spellings above on its items and holds exactly one row for
each; and no two by-name rows differ only by case.
`[probe: tools/probe_by_name_normalisation.py, Jellyfin 10.11.11, 2026-08-27]`

The mechanism is the id itself: the by-name key is lowercased before hashing whenever
`EnableNormalizedItemByNameIds` is set, and it defaults to set
`[source: Emby.Server.Implementations/Library/LibraryManager.cs:636-658,1095-1100 @ v10.11.11]`
`[source: MediaBrowser.Model/Configuration/ServerConfiguration.cs:72 @ v10.11.11]`. Two limits of
the fold, both part of the behaviour: it is **case only** — spellings differing in diacritics stay
separate items — and it also spans the characters a filename cannot carry, which the key builder
replaces with spaces before hashing
`[source: MediaBrowser.Controller/Entities/Genre.cs:79-92 @ v10.11.11]`. Which spelling a merged
row *displays* is the one that created it, because an existing row is reused rather than renamed
`[source: Emby.Server.Implementations/Library/LibraryManager.cs:1052-1075 @ v10.11.11]` — not
observable read-only, so the probe reports it as source-backed rather than measured.

**Depends on it:** yes, quietly. A client filtering by `genreIds` sends one id and expects the
items of every spelling behind it; a user scrolling `/Genres` sees a list without near-duplicates.

**Atrium does:** the same — one item per case-folded name, first spelling seen as the display
name, diacritics preserved and distinct ([004 §3.7](../../specs/004-metadata-resolution/spec.md#37-people-genres-and-studios)).
The ids themselves differ by derivation, as everywhere (§1.4).

### 2.19 A play is counted at start, and reports resolve last-writer-wins

**Jellyfin does:** increments `PlayCount` and sets `LastPlayedDate` when the **start** report
arrives, not when playback ends — and sets `Played` to *false*, so starting a previously played
item un-marks it until it completes again. A stop carrying a position adds no further count; a
stop carrying **none** counts a second time, so a start-to-finish viewing whose stop omits the
position measures `PlayCount: 2`. The bare mark route is different again: `POST
/UserPlayedItems` without `datePlayed` is `max(count, 1)` — marking twice stays at one — and
only the `datePlayed` form increments. And nothing anywhere compares a report's position against
the stored one: a progress at 40% followed by one at 20% reads back 20%, because a deliberate
seek backwards arrives as exactly that report.
`[probe: tools/probe_playstate.py, Jellyfin 10.11.11, 2026-08-28]`
`[source: Emby.Server.Implementations/Session/SessionManager.cs:814-832, 1115-1145 @ v10.11.11]`
`[source: MediaBrowser.Controller/Entities/BaseItem.cs:1893-1927 @ v10.11.11]`

**Depends on it:** any client or script comparing play counts across servers, and every viewer
who rewinds — a server that "protected" the stored position from older reports would pin them at
their furthest point.

**Atrium does:** the same. [007 §3.4 and §3.6](../../specs/007-user-data-and-playstate/spec.md)
carry the full effect tables; the draft of both had the intuitive rules — increment on mark,
never rewind — and the measurement reversed each.

### 2.20 `static=true` serves the original bytes; the URL's container is only a label

**Jellyfin does:** answers any `static=true` delivery request with the untouched source bytes,
whatever container the path names: `stream.mp3?static=true` on a FLAC track is `200` with FLAC
bytes behind `Content-Type: audio/mpeg`, and `stream.mkv?static=true` on an mp4 film is the mp4
bytes as `video/x-matroska` — byte-identical to the unsuffixed static route, ranges included.
No error, no remux, no re-encode. Swept at 008 T6 across **every** container a library admits, on
video and on audio, and the bytes were identical for all thirty-eight
`[probe: tools/probe_range_matrix.py, Jellyfin 10.11.11, 2026-08-28, 2026-08-29]`.

Three further parts of the same rule, measured in that sweep. The `container` **query** parameter
is the same lever as the path suffix, and answers the same label. A container the table has no row
for is not an error either: the label falls back to the file's own extension, so
`stream.banana?static=true` on an mp4 is `video/mp4`. And `stream.wav?static=true` is a `200` with
the source bytes — the one shape of the PCM/WAV defect of §3.2 that is *not* broken, because static
never starts an encoder.

**Depends on it:** a downloading client that names a wrong container still receives, correctly,
the original file — sniffing tools open it fine and only the label lies. A server that refused
instead would break that download outright.

**Atrium does:** the same, implemented at 008 T6. Static means the source bytes, absolutely; the
suffix picks the `Content-Type` and nothing else, from a table measured row by row rather than
transcribed from the reference's own — which matters twice, because copying it would be copying
code (Principle IV) and because several rows are not guessable: `.opus` is `audio/ogg`, `.alac` is
`audio/mp4`, `.mts` is `model/vnd.mts`. The
[008 draft](../../specs/008-playback-negotiation-and-delivery/spec.md#35-delivery-the-rules-that-apply-to-every-route)
said a mismatch would be an error, and the measurement replaced it.

### 2.21 Playback policy permissions are negotiation-inert

**Jellyfin does:** consult `EnableMediaPlayback` on **no** playback route — its only readers are
the item DTO's `PlayAccess` property and the remote-control `Play` command `[source:
MediaBrowser.Controller/Entities/BaseItem.cs:1057,
Emby.Server.Implementations/Session/SessionManager.cs:1321 @ v10.11.11]` — and treat the three
processing permissions as one gate: for a video item, `SupportsTranscoding` drops to `false`
only when `EnableVideoPlaybackTranscoding`, `EnableAudioPlaybackTranscoding` **and**
`EnablePlaybackRemuxing` are all denied; any single denial changes nothing at negotiation
`[probe: tools/probe_playback_info.py, Jellyfin 10.11.11, 2026-08-28; source:
Jellyfin.Api/Helpers/MediaInfoHelper.cs:278-293 @ v10.11.11]`. At delivery it reads two of the
three, **per stream**: a user denied video transcoding has the video stream force-copied
"regardless of whether it will be compatible or not", and one denied audio transcoding has the
audio stream force-copied the same way `[source:
MediaBrowser.Controller/MediaEncoding/EncodingHelper.cs:7136-7166 @ v10.11.11]`.

Two limits on that, both of which decide what a faithful reimplementation may do (008 T13). The
force-copy is reached **only from a video request** `[source:
Jellyfin.Api/Helpers/StreamingHelpers.cs:198 @ v10.11.11]`, so an audio-only delivery consults
neither permission and re-encodes for a denied account exactly as for a permitted one. And the
reference's own delivery-time *refusal* — "User does not have access to video transcoding",
raised when a video job's output codec is not a copy `[source:
MediaBrowser.MediaEncoding/Transcoding/TranscodeManager.cs:385-393 @ v10.11.11]` — **cannot
fire**, because the same permission on the same user has already rewritten that codec to `copy`
two calls earlier. The refusal is unreachable code; the force-copy is the behaviour.

**Depends on it:** an operator who denies one permission and observes that clients still play; a
client that never learned to handle a policy `403` from these routes, because none exists.

**Atrium does:** the same negotiation semantics — the all-three gate, flags rather than errors,
no invented `403`.

> **Measured on 2026-09-02, and the first half of that sentence is false on the wire.** A seat with
> `EnableVideoPlaybackTranscoding`, `EnableAudioPlaybackTranscoding` and `EnablePlaybackRemuxing`
> all denied — read back denied on both servers — negotiates a video item the profile cannot direct
> play and is answered **`SupportsTranscoding: true` here and `false` there**
> `[probe: tools/differential.py --named delivery-time-policy-refusal, Jellyfin 10.11.11,
> 2026-09-02]`. The all-three gate is described in this entry and is not applied by the
> negotiation, so the flag Atrium sends is the model's default rather than the answer this entry
> claims. It is the first difference 010's harness has found in **Atrium** rather than in a
> document, and it is **008's** to decide: the harness triages and the feature that owns the
> endpoint answers ([010 §2](../../specs/010-conformance-harness/spec.md#2-scope)). The delivery
> half of the row is untouched by it — both servers answered the same `200` for a request that
> would have to re-encode, so the force-copy edge this entry describes was not reached by the
> fixture film either.

**The one edge not replicated** is delivery-time force-copy into an output that
violates the negotiated profile: Atrium refuses the step instead, and no client can depend on
receiving a broken stream ([008 §3.3](../../specs/008-playback-negotiation-and-delivery/spec.md#33-the-decision)).
The refusal is scoped to where the reference would have copied — the two streams of a video
delivery, each against its own permission — so an audio delivery and a plan that copies the
stream a denial names are both served exactly as a permitted account's are, and
`EnablePlaybackRemuxing` changes nothing at delivery on either server. It is answered with the
`500` that route already gives anything it cannot produce, not with a new status: a policy
`4xx` on a playback route is the fiction 008's gate removed.

### 2.22 `SupportsDirectStream` mirrors `SupportsDirectPlay`

**Jellyfin does:** disables its direct-stream option on every negotiation — the source comment
says "direct-stream http streaming is currently broken" — so the flag never answers
independently: it is `true` exactly when `SupportsDirectPlay` is
`[probe: tools/probe_playback_info.py, Jellyfin 10.11.11, 2026-08-28; source:
Jellyfin.Api/Helpers/MediaInfoHelper.cs:251-268 @ v10.11.11]`. A remux answer is expressed as a
`TranscodingUrl` with the streams copied at delivery, not as the direct-stream flag.

**Depends on it:** a client that branches on `SupportsDirectStream` is, on this version,
branching on direct play; one that expected an independent remux flag would never see it.

**Atrium does:** the same mirror. Resurrecting the distinction would be a flag no reference
answer sets, which is a delta a differential would flag on the first negotiation.

### 2.23 A negotiation opens the file; a listing does not, and what it learns is kept

**Jellyfin does:** two different things with the same media source, depending on which route is
asked. A listing reads the stored sources and stops there
`[source: Emby.Server.Implementations/Dto/DtoService.cs:261 @ v10.11.11]`, so an item whose
inspection never succeeded is answered with a `Container` inferred from its path, a `Size`, no
`RunTimeTicks`, no `Bitrate`, `MediaStreams: []` and the three capability flags all `true` — the
same shape on `/Items`, `/Items/{itemId}` and `/Items/Latest`. A **negotiation** for the same item
refreshes it with probing enabled before any profile is applied, whenever the first source carries
no stream of the item's own kind
`[source: Emby.Server.Implementations/Library/MediaSourceManager.cs:170-189 @ v10.11.11]`, and that
refresh completes inside the request: a file that became readable after the scan comes back fully
annotated — streams, runtime, bitrate and a corrected `Size` — in 0.20 s, against 0.01 s for an
item already annotated. **The result is written down**: the next listing of that item carries all
of it, with no scan in between. A file that can never be read pays the probe on every negotiation,
measured at 0.18–0.20 s three runs running
`[probe: tools/probe_uninspected_source.py, Jellyfin 10.11.11, 2026-08-29]`.

Two consequences that are easy to state backwards. A file **truncated** to its first kibibyte is
not in this state at all — a Matroska header is at the front, so it probes cleanly and answers a
full annotation with a `Size` of 1 024. And a file **deleted** after the scan is not either: the
stored streams are still there, so the refresh is never triggered and the negotiation answers as
though the file were present, address and all.

**Depends on it:** the video client, which refuses direct play and then looks for the address the
answer promised — on the reference there is always one, because the annotation happened. The music
client depends on the *listing* half, and gets the same empty source from a stock reference that it
gets here: that half is parity.

**Atrium does:** [012](../../specs/012-negotiation-inputs/spec.md) reproduces both halves — the
negotiation opens the file and writes what it finds, the listing does not — which is the whole of
that feature's §3.2. Until it lands, the shortfall is [§5](#5-accepted-gaps-in-v1)'s row.

### 2.24 A profile's delivery protocol is an enumeration, in every sense

**Jellyfin does:** bind `TranscodingProfile.Protocol` to a two-member enumeration whose members are
lower-case by declaration `[source: Jellyfin.Data/Enums/MediaStreamProtocol.cs @ v10.11.11]`,
`[source: MediaBrowser.Model/Dlna/TranscodingProfile.cs:77 @ v10.11.11]`, and read it the way a
.NET enumeration is read. Eighteen spellings posted to one item on one profile answer four ways
`[probe: tools/probe_playback_info.py, Jellyfin 10.11.11, 2026-08-29]`:

| What the profile said | What came back |
|---|---|
| `hls`, `Hls`, `HLS`, `hLs` | An HLS address, `TranscodingSubProtocol: "hls"` |
| `http`, `Http`, `HTTP` | A progressive address, `TranscodingSubProtocol: "http"` |
| absent, `null`, `""` | The declared default — progressive, `"http"` |
| `0`/`"0"`, `1`/`"1"` | The ordinal's member: progressive, HLS |
| `2`/`"2"` — an ordinal no member has | `200`, a progressive address, and `TranscodingSubProtocol: 2`, a **number** in a field the enumeration spells as a word |
| `dash`, `" "`, `true` | `400`, RFC 9457 problem details, `errors` keyed on `$.DeviceProfile.TranscodingProfiles[0].Protocol` |

The echo is the enumeration's spelling, never the profile's: `Hls` in, `"hls"` out.

**Depends on it:** any client whose profile spells the protocol the way its own language does. A
client sending `Hls` is *correct* against this reference and gets HLS; a server that compares the
string case-sensitively sends it a progressive address instead, which is the one direction
Principle I has least tolerance for.

**Atrium does:** [012 §3.3](../../specs/012-negotiation-inputs/spec.md) reproduces the whole table,
including the refusal — which is the same shape §1.1's binding already produces for this body, and
the opposite of §1.12's rule for a **query** value. The out-of-range ordinal is reproduced too: it
is a `200` a client can act on, so it is class B and there is nothing to gain by tidying it.

### 2.25 `GET /Sessions`' three filters are two filters and a visibility rule

**Jellyfin does:** declare `controllableByUserId`, `deviceId` and `activeWithinSeconds`
`[source: Jellyfin.Api/Controllers/SessionController.cs:52-59 @ v10.11.11]` and apply them in an
order that is observable `[probe: tools/probe_session_filters.py, Jellyfin 10.11.11, 2026-08-29]`:

- `deviceId` narrows the **whole** session list first, before the rule about whose sessions the
  caller may see, and matches case-insensitively; an empty value is ignored. A non-administrator
  naming another user's device therefore gets an empty `200` — the filter matched and the
  visibility rule then removed the row.
- `activeWithinSeconds` is applied **last**, and only when greater than zero: `0` and `-5` answer
  the unfiltered list, which is [§1.12](#112-an-unrecognised-query-value-is-ignored-not-rejected)'s
  family rather than a refusal.
- `controllableByUserId` is not a filter at all. It replaces the caller's own visibility rule with
  a different one — sessions that are remote-controllable, subject to the named user's shared-device
  setting, the caller's remote-control permission and per-device access. A session is
  remote-controllable only while a live control channel is attached to it
  `[source: MediaBrowser.Controller/Session/SessionInfo.cs:246-266 @ v10.11.11]`, so declaring
  `SupportsMediaControl` is necessary and not sufficient, and a request-response client is never
  one. And a **non-administrator naming anybody but themselves is refused `403`**, `text/plain` —
  where naming another user's *device* was an empty `200`.

**Depends on it:** the video client sends `deviceId` on this route today. Nothing observed sends
the other two.

**Atrium does:** ~~none of it — v1 declares no parameter on this route.~~ **declare all three, at
[002 §3.8](../../specs/002-authentication-users-and-sessions/spec.md#38-sessions), in the change
this entry asked for — 2026-09-03.** `deviceId` and `activeWithinSeconds` are reproduced whole,
including the order and the forgiveness of an empty value and a non-positive one.
`controllableByUserId` is accepted and its `403` reproduced, while the list it answers is **always
empty here**: the rule's first clause is a live control channel, v1 has none, and it reports
`SupportsRemoteControl: false` on every session — which §2.14 measures to be what the reference
reports for a request-response client too. So the three later clauses decide nothing observable
here and 002 §3.8 deliberately does not state them. Recorded here at 012's
measurement gate (012 OQ-7) because the route is
[002 §3.8](../../specs/002-authentication-users-and-sessions/spec.md#38-sessions)'s and the `403`
is a sentence about who may see whose device, which is 002's to specify.

### 2.13 `DeviceId` is mandatory on one route, not on the header

**Jellyfin does:** answer `200` on an ordinary authenticated route for a client header carrying no
`DeviceId` at all, and `400` for one on `POST /Users/AuthenticateByName`. `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]`

**Depends on it:** clients that set the header once and reuse it, minus the component, on routes
where nothing needs a session.

**Atrium does:** the same. [002 plan §6.3](../../specs/002-authentication-users-and-sessions/plan.md#63-the-x-emby-authorization-grammar)
called it "the one fatal case", which is true of one route and not of the parser: a parser that
raised would refuse requests the reference serves.

### 2.5 `SortBy` vocabulary

**Jellyfin does:** supports `SortName`, `DateCreated`, `PremiereDate`, `PlayCount`, `DatePlayed`,
`Random`, `AlbumArtist`, `Artist` — a superset of Emby's.
`[prior-probe: Jellyfin 10.11.11, 2026-06-13]`

**Depends on it:** every sort menu a client offers — the tokens are what it may send, and a token
outside the set is ignored rather than refused (005 §3.3), so a missing member would not error; it
would return a default order under the name of the one the client asked for.

**Atrium does:** the same set. How `SortName` itself is derived is §2.6.

> ⚠️ **The set is not the vocabulary — measured 2026-09-02**, when this claim's probe was written
> and the `prior-probe` discharged. The reference's own enumeration names **thirty**
> `[source: Jellyfin.Data/Enums/ItemSortBy.cs @ v10.11.11]`, and asked one at a time against a
> 9,333-item library, **twenty-one tokens outside the recorded eight order rows**
> `[probe: tools/probe_sort_vocabulary, Jellyfin 10.11.11, 2026-09-02]`:
>
> `AiredEpisodeOrder`, `Album`, `OfficialRating`, `StartDate`, `Name`, `Runtime`,
> `CommunityRating`, `ProductionYear`, `CriticRating`, `IsFolder`, `IsUnplayed`, `IsPlayed`,
> `SeriesSortName`, `VideoBitRate`, `AirTime`, `Studio`, `IsFavoriteOrLiked`,
> `DateLastContentAdded`, `SeriesDatePlayed`, `ParentIndexNumber`, `IndexNumber`.
>
> **The method, because the verdict depends on it.** Each token was asked ascending and descending
> and the returned identifier sequences compared; a token that orders answers two different
> sequences, and a token that is ignored answers the default order both times. The control is a
> token no enumeration contains, and it is the method's own check that it answered **identically to
> `Default`** — the one member of the thirty that is *supposed* to mean "no ordering".
> `Random` was the one member whose two identical requests differed, which is §3.6's seedless
> per-row shuffle seen from another angle. One member, `SeriesDatePlayed`, orders rows but takes
> **63 seconds** a request on this library where every other token answers in under a second — the
> one place this run measured a cost rather than a behaviour, and it is why the probe's timeout is
> generous.
>
> **This is a floor that was read as a set.** The eight were what a 2026-06-13 session happened to
> ask for, and nothing in that measurement claimed closure — the register said so, which is why the
> row stayed open.
>
> **What it changes is not this entry but 005's.** Principle I is zero delta: a client sending
> `SortBy=ParentIndexNumber` gets one order from the reference and, from a server implementing eight
> tokens, a different one. **All three tokens a shipping music client sends outside the eight —
> `ParentIndexNumber`, `IndexNumber` and `ProductionYear` — order rows here**, so this is not a
> hypothetical client. [client-embeat-mobile §5.8](client-embeat-mobile.md#58-the-album-play-queue-is-correctly-ordered-by-accident)
> concluded that album ordering survives *by accident* because two of the three keys evaporate; the
> reference does not drop them, so against the reference the same request is ordered **on purpose**,
> and a server that drops them is the one introducing the accident. Which of the thirty v1
> implements is [005](../../specs/005-item-query-api/spec.md)'s decision and is now a decision it
> knows it is taking.

### 2.6 `SortName` has two derivations, and three types use the second

**Jellyfin does:** derives a sort name in one of two entirely different ways depending on the item
type. `[probe: tools/probe_sort_names.py, Jellyfin 10.11.11, 2026-08-26]`

For movies, series, albums, artists and playlists: trim and lowercase, strip configured articles
at the start, in the middle and at the end, remove one configured character set, replace another
with spaces, **left-pad every digit run to ten characters**, then fold diacritics. Nothing trims or
collapses the whitespace this leaves behind, so `Rock & Roll` becomes `rock  roll` with two spaces
and `S.W.A.T.` becomes `s w a t ` with a trailing one.

For `Audio`, `Episode` and `Season`: a zero-padded numeric prefix followed by the **raw** name,
with none of the above applied. Audio pads disc and track to four; Episode pads season to
**three** and episode to **four**; Season is the number alone.
`[source: MediaBrowser.Controller/Entities/Audio/Audio.cs:94-98, MediaBrowser.Controller/Entities/TV/Episode.cs:238-242, MediaBrowser.Controller/Entities/TV/Season.cs:149-152 @ v10.11.11]`

**Depends on it:** every ordered list a client draws. This is not a field a client reads and
compares — it is the order items arrive in, which no client can correct and most will not even
recognise as wrong.

**Atrium does:** both, exactly, including the whitespace artefacts. Full specification in
[003 §3.7](../../specs/003-library-configuration-and-scanning/spec.md).

**One part of step 6 is not measured and is not claimed.** The step says "transliterate anything
still outside ASCII", and the only case in the measured set is `Amélie` — whose `é` decomposes, so
folding alone reaches it. A character with no ASCII decomposition (`ø`, `ß`, a non-Latin script)
was never sent. Atrium folds, applies a short table of the obvious Latin readings, and drops what
remains; that last part is a decision, not a reproduction, and it is 003 OQ-7.

Two temptations to name, because both are what a careful implementer would otherwise do:

1. **Tidying the whitespace.** Collapsing `rock  roll` to `rock roll` changes the ordering of every
   name containing a removed character, quietly, and only for some names.
2. **Using one sort-name function for everything.** Applying the base rule to audio makes `The
   Song` sort under `s` instead of `T` and reorders every album in the library.

### 2.7 Playlists hold each item at most once

**Jellyfin does:** de-duplicates in two stages — dropping items already in the playlist, then
dropping repeats within the incoming batch. Measured on both code paths: creating a playlist with
the same id twice yields one entry, and adding an id already present yields zero new entries.
`[probe: tools/probe_playlist_move.py, Jellyfin 10.11.11, 2026-08-26]`
`[source: Emby.Server.Implementations/Playlists/PlaylistManager.cs:222-225 @ v10.11.11]`

**Depends on it:** a client adding a track already present sees the count stay the same. It is not
told anything — the request succeeds.

**Atrium does:** the same. The argument for allowing duplicates is real — a set list may want the
same track twice — and it loses to Principle I.

**Entry identity is where this comes from**, which this entry had backwards until 2026-08-31. It
read *"a row is still addressed by its own `PlaylistItemId`, not by the item's `Id`"*. The two are
the same value (§2.26), so an entry cannot be named apart from its item — and a list addressed by
name cannot hold one twice. The de-duplication is not a policy beside the identifiers; it is what
the identifiers leave possible.

### 2.8 `Move`'s `newIndex` is the entry's position after the move

**Jellyfin does:** removes the entry, then inserts it so it ends up at exactly `newIndex` in the
resulting list. On `[A B C D E]`, moving index 0 to index 3 gives `B C D A E`, not `B C A D E`.
`[probe: tools/probe_playlist_move.py, Jellyfin 10.11.11, 2026-08-26, re-measured 2026-08-31]`
The boundaries around it are a defect rather than a semantic, and they are §3.15.

**Depends on it:** every drag-and-drop reorder. Upward moves are identical under either reading,
so a client built against the wrong one works until a user drags something **down**.

**Atrium does:** the same. This project's specification asserted the opposite until it was
measured — which is the whole reason the probe was written before the code.

### 2.26 A playlist entry's identifier is the item's identifier

**Jellyfin does:** answer every row of `GET /Playlists/{playlistId}/Items` with a
`PlaylistItemId` equal to that row's own `Id`. The property is filled from a field that caches the
resolved item's id the first time the entry is looked up, so there is no second identifier
anywhere in the exchange. `[probe: tools/probe_playlist_move.py, Jellyfin 10.11.11, 2026-08-31]`
`[source: MediaBrowser.Controller/Entities/BaseItem.cs:1797-1802 @ v10.11.11]`

**Depends on it:** the two routes that address a row — `Move` and `Remove` — take that value, and
a client that held an entry id and an item id as different things would still work, because it
would be holding the same string twice. What no client can do is address two occurrences of one
track, and §2.7 is why there are never two.

**Atrium does:** the same, and it is not a free choice — a `PlaylistItemId` differing from `Id`
would be a value no reference server sends. Specified in
[009 §3.1](../../specs/009-playlists/spec.md).

> **This is the finding the 009 spec review was least prepared for.** The specification was built
> around the opposite claim, in four places including an acceptance criterion, and the music
> client's contract asks for a `PlaylistItemId` *"distinct from the track id"*. Both were reasoning
> from what the two names suggest. Nothing had compared the two values.

### 2.9 A reported position resolves through six branches, not two thresholds

**Jellyfin does:** decides between *discard the position*, *mark played* and *keep it resumable*
through an ordered rule — **and runs it on every report that carries a position, progress as
much as stop**: a progress at 95% marks the item played mid-playback. A stop with no position
counts as played to the end; an unknown runtime counts as played; below 5% of runtime the
position is discarded; above 90%, or within one second of the end, it is played; an item whose
**runtime** is under 300 seconds is played rather than resumable; otherwise the position is kept.
The percentage comparisons are strict at both ends, pinned at tick precision.
`[probe: tools/probe_playstate.py, Jellyfin 10.11.11, 2026-08-28]`
`[source: Emby.Server.Implementations/Library/UserDataManager.cs:296-370 @ v10.11.11]`

**Depends on it:** what appears in "continue watching", which is the most-used row in most clients.

**Atrium does:** the same, with the same defaults. Full rule in
[007 §3.7](../../specs/007-user-data-and-playstate/spec.md).

The branch most easily missed is the 300-second one: it is a floor on the **item's runtime**, not
on the position. A short clip stopped in the middle is *played*, not resumable. Reading it as a
position floor produces a server that keeps resume points for every short item.
*(This heading said "a stop report" until 007's review measured the rule firing on progress.)*

### 2.10 The image and delivery routes accept a token and require none

**Jellyfin does:** answer `GET /Items/{id}/Images/Primary` and
`GET /Videos/{id}/stream?static=true` with `200` to a request carrying **no token at all**. All
five mechanisms are accepted there — four measured on 2026-08-26 `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-26]`
and the fifth on 2026-08-28, once `mechanisms()` sent it `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]` —
and **not one of them is required**, which makes the count the least interesting half of this
section. A route that answers `200` to a request carrying nothing accepts every mechanism
trivially and proves nothing about any of them; what it proves is the sentence in the heading.
An **invalid** token changes nothing either: an unknown 32-hex token and a malformed one, sent
through the header, the query and the `MediaBrowser` scheme, each answer the identical `200` —
the route does not validate what it does not require
`[probe: manual requests via tools/_probe.py, Jellyfin 10.11.11, 2026-08-28]`.

**Depends on it:** yes, and in the shape that is hardest to see from inside a client. A bare URL
handed to an image loader or an external player is exactly what these routes are for, and a client
that has never sent a token on them is a client a server can break by starting to want one.

**Atrium does:** for the image routes, the same — decided at 006's spec review, 2026-08-28: a
token is accepted, none is required, and there is no per-user visibility branch
([006 §3.2](../../specs/006-images/spec.md#32-get-itemsitemidimagesimagetype--getitemimage)).

**And for the delivery routes, the same, decided at 008 T6 on 2026-08-29.** The deferral above ran
out at the task that landed the routes, and the measurement it was waiting for is now this feature's
own: no token, an unknown token and `?api_key=` answer identically on all four `stream` routes
`[probe: tools/probe_range_matrix.py, Jellyfin 10.11.11, 2026-08-29]`. The split is **per action**,
not per feature — `/Audio/{itemId}/universal` answers `401` to the first two from the same probe run,
because it is the one delivery action carrying an authorization attribute `[source:
Jellyfin.Api/Controllers/AudioController.cs:89, Jellyfin.Api/Controllers/VideosController.cs:312,
Jellyfin.Api/Controllers/UniversalAudioController.cs:94 @ v10.11.11]`. So Atrium requires a token on
`/universal` and on none of the other four, which is the reference's own line and not a simplification
of it.

The argument is the paragraph above, at its strongest here: a bare URL handed to an **external
player** is what these routes are for, and the divergence would be invisible in every test written
against a client that also holds a token. 008's own task list had it the other way round — "a
tokenless request refuses" — while [002 §3.1](../../specs/002-authentication-users-and-sessions/spec.md#31-how-a-client-presents-a-token),
accepted three days earlier, already said none is required on this class. The measurement settled it
in 002's favour.

**Two more delivery actions require one, found at 008 T10**: `GET /Videos/{id}/master.m3u8` and
`GET /Videos/{id}/main.m3u8` answer the empty `401` to a request carrying nothing, because the
reference's whole HLS controller carries the authorization attribute where its stream actions carry
none `[source: Jellyfin.Api/Controllers/DynamicHlsController.cs:39-41 @ v10.11.11]`, `[probe:
tools/probe_hls.py, Jellyfin 10.11.11, 2026-08-29]`. That does not soften the heading: the routes a
bare URL is handed to are the ones that require nothing, and a playlist is followed by a player that
already holds a token. It does mean the split is genuinely **per action** rather than "delivery
requires none".

**Two more require none, added at 011 T7**, and they are the two a subtitle playlist's own entries
point at: `GET /Videos/{id}/{sourceId}/Subtitles/{index}/Stream.{format}` and its ticks-in-path
form answer `200` with the cues to a request carrying no token, an unknown token or `?ApiKey=`
alike, measured in the same run in which the subtitle *playlist* beside them refuses the first two
with the empty `401` `[probe: tools/probe_subtitle_delivery.py, Jellyfin 10.11.11, 2026-08-29]`.
The reference's own controller carries the authorization attribute on the playlist action alone
`[source: Jellyfin.Api/Controllers/SubtitleController.cs:208, 338-345 @ v10.11.11]`. Atrium does
the same, so the count is now **six of nine require nothing** and three do —
`/Audio/{id}/universal` and the two HLS playlists — with the subtitle playlist joining the three
when 011 T8 lands it.

What 002 records is the consequence, so that whoever takes it takes it knowingly: on the
reference **an item id is a capability**, and any divergence 006 or 008 chooses is one a client can
observe. Both features have now taken it and neither diverged, so the consequence stands as the
reference's: anyone holding an item's identifier can read its bytes, and no per-user visibility
branch runs on the way.

### 2.11 A disabled account is refused with `403`, not `401`

**Jellyfin does:** refuse a disabled account with `403` and an unknown username with `401`. The
bodies are identical — `text/plain`, 25 bytes, `Error processing request.` — so the **status** is
the whole of the difference. The disabled account answers `403` whether the password it was sent is
right or wrong, so the refusal discloses that the account exists and is disabled, and discloses
nothing about the password. `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-26]`

**Depends on it:** every client, in the direction that matters most. Clients re-authenticate on
`401` and show an error on `403`. A server answering `401` for a disabled account puts a client in
a login loop — prompt, correct password, `401`, prompt again — with the user's credentials correct
every single time.

**Atrium does:** the same. **This overturns a decision that was deliberate rather than an
oversight.** [002 §3.3](../../specs/002-authentication-users-and-sessions/spec.md#33-post-usersauthenticatebyname--authenticateuserbyname)
specified `401` for a disabled account and called it indistinguishable *on purpose*, on the
argument that disclosing account state is an enumeration risk. The risk was real and it does not
disappear; what changed is that the reference discloses it anyway, so refusing to would be a delta
that protects nobody — the reference is still there to be asked. The cost is bounded and now
written down: an unauthenticated caller can tell a disabled account from a name that was never
registered. It cannot tell a right password from a wrong one, which is the disclosure that would
matter more.

> **Three refusals remain unmeasured, and the probe declines to measure them.** An enabled account
> sent a wrong password, an account locked out by failed attempts, and a live token whose user was
> disabled after it was issued. Each needs a real account to fail against, and failing against one
> moves a lockout counter no probe can reset. They are
> [002 OQ-5](../../specs/002-authentication-users-and-sessions/spec.md#7-open-questions), to be
> measured against a throwaway enabled account, and until then the first is an assumption rather
> than a measurement.

### 1.9 Every response carries `X-Response-Time-ms`

**Jellyfin does:** stamps every response with the time it took, in fractional milliseconds —
`X-Response-Time-ms: 2.1329`. Its middleware is registered unconditionally; the two configuration
flags beside it gate a slow-response *log line*, not the header. `[probe: tools/probe_routing.py, Jellyfin 10.11.11, 2026-08-28]` `[source: Jellyfin.Api/Middleware/ResponseTimeMiddleware.cs:17, Jellyfin.Server/Startup.cs:163 @ v10.11.11]`

**Depends on it:** no known client. It is a diagnostic.

**Atrium does:** the same. Omitting it would be a difference on **every** response in the project —
55 rows of noise in the first differential run — for a middleware that costs fifteen lines.

> **This project did not know the header existed.** Neither specification mentioned it, and no
> amount of reading either codebase would have surfaced it: it took issuing one real request and
> reading what came back. It is the smallest useful argument for the differential harness that
> feature 010 delivers.

### 1.10 JSON responses carry `charset=utf-8`

**Jellyfin does:** sends `Content-Type: application/json; charset=utf-8`, as ASP.NET Core's JSON
formatter does. `[probe: tools/probe_routing.py, Jellyfin 10.11.11, 2026-08-28]`

**Depends on it:** unlikely — a client parses JSON as UTF-8 regardless. But it is on every response.

**Atrium does:** the same, through a response class rather than a middleware, so the content type
belongs to the thing that produced the body. Starlette appends `charset=utf-8` only to `text/*`
media types, so its `JSONResponse` would send a bare `application/json`.

### 1.11 There are four error shapes, not one

**Jellyfin does:** answer a refusal in one of several forms, decided by **where** the refusal
happened. `[probe: tools/probe_routing.py, Jellyfin 10.11.11, 2026-08-28]` `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]` `[probe: tools/probe_query_envelope.py, Jellyfin 10.11.11, 2026-08-28]`

| Refusal | Shape |
|---|---|
| Unauthenticated request | `401`, **empty body**, `Content-Length: 0`, no `Content-Type`, **no `WWW-Authenticate`** |
| Path matching no route | `404`, **empty body**, no `Content-Type` |
| A method the path does not have | `405`, **empty body**, no `Content-Type`, and `Allow` naming every method that path has `[probe: tools/probe_routing.py, Jellyfin 10.11.11, 2026-08-26]`. The order is alphabetical on the one measured pair where alphabetical and registration order differ: `PUT /UserFavoriteItems/{itemId}` answers `Allow: DELETE, POST` `[probe: tools/probe_routing.py, Jellyfin 10.11.11, 2026-08-28]` — the order `compat/errors.py`'s sort produces |
| An item a handler could not find | `404`, **RFC 9457 problem details** as JSON |
| A malformed value the model binder rejected | `400`, **RFC 9457 problem details** with an `errors` map |
| A controller that refused the request itself | `4xx`, **`text/plain` with no `charset`**, and the fixed 25-byte body `Error processing request.` `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-26]`. **Measured for a policy refusal too**, at 009's spec gate: a non-administrator naming another user in `userId` is answered `403` with those same 25 bytes `[probe: tools/probe_playlist_visibility.py, Jellyfin 10.11.11, 2026-08-31]` |
| A controller that refused with its own message | `404`, the message as a **JSON-encoded bare string** — `"<item name> does not have an image of type Box"` — `application/json; charset=utf-8` `[probe: manual requests via tools/_probe.py, Jellyfin 10.11.11, 2026-08-28]`. **Measured on a second route and a second feature at 009 T9**: `GET /Playlists/{playlistId}/Items` answers the fixed 20-byte `"Playlist not found"` for an id that addresses nothing, for a real item that is not a playlist, and for a playlist the reader may not see `[probe: tools/probe_playlist_read.py, Jellyfin 10.11.11, 2026-09-01]`. **And it is not only a
`404` shape**: `DELETE /Items/{itemId}` refuses a caller who may not delete the item with `401`
carrying the 21-byte `"Unauthorized access"` under the same content type, so one route answers
`401` in two shapes — this one when a credential arrived, the empty one above when none did
`[probe: tools/probe_item_deletion.py, Jellyfin 10.11.11, 2026-09-01]` |

**The fourth shape is not the exception it looked like at 006, and the split is per route rather
than per feature.** The image route splits its two `404`s by *which lookup failed* — an item that
does not exist gets problem details, an item lacking the image gets the string. The playlist read
route has no such split: every way it cannot hand over a playlist is the one string, and the
problem-details `404` for that same playlist is a **different route**, `GET /Items/{itemId}`. So a
route's `404` shape is a fact about the route, and reading it off another route is how a feature
ships a body no reference server sends. A **malformed** identifier in the path never reaches either
shape: it is the validation `400` below, keyed on the parameter and quoting the value back —
`{"playlistId": ["The value 'not-an-identifier' is not valid."]}`, which is the *path* parameter's
sentence and not the body's `The supplied value is invalid.`
`[probe: tools/probe_playlist_read.py, Jellyfin 10.11.11, 2026-09-01]`

```json
{"type": "https://tools.ietf.org/html/rfc9110#section-15.5.5",
 "title": "Not Found", "status": 404, "traceId": "00-b1be…-8a91…-00"}

{"type": "https://tools.ietf.org/html/rfc9110#section-15.5.1",
 "title": "One or more validation errors occurred.", "status": 400,
 "errors": {"itemId": ["The value 'not-a-guid' is not valid."]},
 "traceId": "00-0138…-3158…-00"}
```

**Two details of the problem-details shape were measured on 2026-08-27** and neither is what the
natural implementation produces `[probe: tools/probe_query_envelope.py, Jellyfin 10.11.11, 2026-08-28]`:

- **The content type is `application/json; charset=utf-8`**, not `application/problem+json`. Both
  ASP.NET Core and every Python framework default to the second for a problem-details body, so
  matching here means overriding a default rather than accepting one.
- **The `errors` key is the parameter's *declared* spelling, not the client's.** `Limit=abc`
  against a route whose parameter is `limit` answers `"errors": {"limit": […]}`. That is the same
  canonicalisation §1.15 describes, visible from the other side — and it means a server that
  echoed the client's spelling would differ on exactly the requests a PascalCase client sends.

**A refusal of the *body* names two keys, and one of them is per route.** The `errors` map above
holds a parameter name because a parameter was what failed. When the **body** fails, the map holds
the binder's own key — `"$"` with the JSON parser's message and byte position when the text is not
JSON, the **empty string** with `The supplied value is invalid.` when it parses and a value inside
it does not bind — **beside the name of the action parameter the route declares**, saying that
field is required. Measured across 007's three reporting routes, where that name is
`playbackStartInfo`, `playbackProgressInfo` and `playbackStopInfo` respectively: one failure,
three spellings, none of them anything the client sent
`[probe: tools/probe_playstate.py, Jellyfin 10.11.11, 2026-08-28]`.

```json
{"type": "https://tools.ietf.org/html/rfc9110#section-15.5.1",
 "title": "One or more validation errors occurred.", "status": 400,
 "errors": {"": ["The supplied value is invalid."],
            "playbackProgressInfo": ["The playbackProgressInfo field is required."]},
 "traceId": "00-9c83…-a105…-00"}
```

**The action-parameter key is measured on a second required body at 009 T13**, and the name is
the reference's own action parameter again: `POST /Items/{itemId}` answers a body that is not JSON
with `"$"` **and** `{"request": ["The request field is required."]}`
`[probe: tools/probe_playlist_rename.py, Jellyfin 10.11.11, 2026-09-01]`. Two features, two
required bodies, and neither spelling is anything a client sent.

**And there is a fifth shape this project does not send, measured in the same run.** A request with
**no body and no `Content-Type` at all** on a route whose body is required is answered `415`,
`Unsupported Media Type`, in the problem-details shape — not the `400` above. Atrium answers the
validation `400` there, on this route and on the four older ones with a required body, which is the
gap [§5](#5-accepted-gaps-in-v1) records rather than a shape any route here reproduces.

**The second key is not universal, and 009 T8 measured what decides it: whether the body is
*required*.** `POST /Playlists` declares an optional body — its four properties may be sent as
query parameters instead — and every one of its refusals names **one** key and no second one.
Measured on four of them in a single run, which also split the first key in two
`[probe: tools/probe_playlist_creation.py, Jellyfin 10.11.11, 2026-08-31]`:

| What was wrong with the body | Key | Message |
|---|---|---|
| A required property is **absent** | `"$"` | `JSON deserialization for type '<the reference's type>' was missing required properties including: 'Name'.` |
| That property is present and **null** | the property — `"Name"` | `The Name field is required.` |
| A value matches no member of a declared vocabulary | `"$"` | `The JSON value could not be converted to <the enum>. Path: $ \| LineNumber: 0 \| BytePositionInLine: 10.` |
| A value did not bind — a malformed identifier | `""` | `The supplied value is invalid.` |

Three of those are new. The split between rows 1 and 2 is the deserialiser refusing the document
against the property validator refusing a value, and the *same property* produces both; row 3 is
`"$"` rather than `""` because an enum converter throws while the text is still being read, which
is what puts it on the deserialiser's side of the same line.

**And row 3's byte position is reproducible where §1.11's parser message is not.** It is the offset
**inside the quoted token** — a one-character value answers `3` where an eight-character one
answers `10` — not an offset into the request, so Atrium sends the sentence byte for byte. Rows 1
and 3 name a type of the reference's own; those two strings are wire facts, declared on the model
that reproduces them (`compat/model.py`'s `WIRE_TYPE` and `WIRE_ENUM_TYPES`), and nothing about how
the reference computes them is carried over. What is still a divergence is the message under `"$"`
when the text is **not JSON at all**, for the reason the paragraph below gives.

**One of Atrium's refusals had the wrong body, and it took three features to find out.**
`compat/errors.ForbiddenError` answers an **empty** `403`, on the argument that it is decided in the
same place as the empty `401` beside it — and its own docstring carried a `⚠️` saying the shape was
unmeasured, because the only account available to measure with was an administrator, and an
administrator lacks no permission. 009's visibility probe creates a throwaway non-administrator and
so could ask: the reference answers the 25-byte body above. Every route that refuses this way is
affected, and one of them shipped in 005 — `/Items?userId=<somebody else>` — so this is a wire
difference on an implemented feature rather than a question about a future one. **Decided at 009's
plan gate: fixed there** ([009 plan §2](../../specs/009-playlists/plan.md)), because 009 needs the
same refusal on two of its own routes and a third wrong copy is worse than one changed handler. The
three *authentication* refusals of
[002 OQ-5](../../specs/002-authentication-users-and-sessions/spec.md#7-open-questions) are a
different class and stay open: each needs a real account to fail against.

**A `403` is two shapes, and one changed handler cannot be all of them — measured at 009 T2.** The
paragraph above says *"every route that refuses this way"* as though the reference refused one way.
It refuses two, and the split is this section's own rule applied to a single status:

| Refusal | Shape | Measured on |
|---|---|---|
| A controller tests the request itself | `403`, `text/plain` with no charset, the 25 bytes | `POST /Playlists/{id}/Items?userId=<somebody else>` as a non-administrator |
| An authorization **policy** refuses before the controller runs | `403`, **no content type, no body** | `POST /Items/{id}` — the elevated controller the music client renames through |

`[probe: tools/probe_playlist_visibility.py, Jellyfin 10.11.11, 2026-08-31]`. The content type is
newly measured on both: the earlier reading of that probe printed forty bytes of body and no header
at all, which cannot tell an empty body from a body-less refusal. So 009 §3.7's sentence and §3.8's
table were never in conflict — they describe the two shapes — and a route refused by policy keeps
the empty one. `compat/errors.ForbiddenError` is the first shape only.

**The line between the two is not the layer — it is how the refusal is expressed, and the second
shape reaches inside a controller too (009 T5).** The table above says *"a controller tests the
request itself"* against *"an authorization policy refuses"*, and a third refusal does not fit
either description: a **shared reader without `CanEdit`** who moves an entry is refused by the
playlist controller's own permission test, and the answer is `403` with **no content type and no
body** — the second shape `[probe: tools/probe_playlist_shares.py, Jellyfin 10.11.11, 2026-08-31]`.
The same is true of a public playlist's reader, who may read it and may not reorder it. The
discriminator is the mechanism: a refusal **thrown** as an exception is rendered by the error
middleware and carries the 25 bytes
`[source: Jellyfin.Api/Helpers/RequestHelpers.cs:77-81 @ v10.11.11]`, while a refusal **returned**
as a result carries nothing at all
`[source: Jellyfin.Api/Controllers/PlaylistsController.cs:421-427 @ v10.11.11]`. Both live in the
same action.

| Refusal | Shape | Measured on |
|---|---|---|
| A permission test **inside** a 009 route: `may_edit` says no | `403`, **no content type, no body** | `POST /Playlists/{id}/Items/{entry}/Move/{n}` as a shared reader without `CanEdit`, and as a public playlist's reader |

So every `may_edit` refusal this feature ships is the body-less shape, not `ForbiddenError`'s
sentence — the same shape 009 §3.8's rename needs, and the reason its second exception class has a
second caller.

**One of that class's raise sites in Atrium is neither shape's measurement**, and there were two.
`api/deps.py` refuses a live token whose account was disabled afterwards — 002 OQ-5's third row,
still open, and it moved from one analogy to another when the handler changed. The second was
`api/users.py` refusing one user reading another, and **the reference does not refuse that at
all**: every authenticated caller is answered `200` with the named user's whole object,
`Configuration` and `Policy` included, in bytes that do not depend on who asked — measured on the
route's whole matrix on 2026-09-01
`[probe: tools/probe_user_read.py, Jellyfin 10.11.11, 2026-09-01]`, after 009 T2 had measured one
cell of it `[probe: tools/probe_playlist_visibility.py, Jellyfin 10.11.11, 2026-08-31]`.
**Decided on 2026-09-01, outside 009 because the route belongs to 002: Atrium replicates**
([§3.22](#322-any-authenticated-caller-reads-any-user-whole--class-b-replicated)), so that raise
site is gone rather than reshaped, and 002 §3.7's provenance-free `403` is withdrawn.

**That route's `404` is the fourth shape, and it is the third route to send one.** An identifier no
account has answers the JSON-encoded bare string `"User not found"`, 16 bytes, under
`application/json; charset=utf-8` — the same body to an administrator and to a non-administrator —
where a **malformed** one is the validation `400` below, keyed on `userId`
`[probe: tools/probe_user_read.py, Jellyfin 10.11.11, 2026-09-01]`. Three features have now looked
at a `404` they assumed was problem details and found this shape instead, which is the argument for
looking on the fourth rather than reading it off a neighbour.

**Atrium gets the shape for free and the keys only deliberately, and it pays for them.** A path
or query parameter's refusal already matched — `compat/errors.validation_errors` keys on the
declared name, which is the bullet above. A body's did not, and worse than expected: the framework
here keys on the **model's Python field**, so the first typed request body in this project answered
`{"item_id": …}`, snake_case, on the wire (§1.1's exact failure).
007 T8 reproduces the measured pair instead: the route names its body parameter after the
reference's (`playbackStartInfo`, `playbackProgressInfo`, `playbackStopInfo`), and the handler
files body failures under `""` or `"$"` beside `The <parameter> field is required.`

One half is a **recorded divergence**: the `"$"` entry's *message* is this parser's, where the
reference's is .NET's `'n' is an invalid start of a property name. … BytePositionInLine: 1.`
Reproducing that sentence would mean writing a JSON parser to fail like another one; the key and
the status match. That no client branches on the text is **assumed**, not surveyed — §3.0.1
tie-break 1's default would presume a compensation exists, set aside here only because the text
cannot be matched at any price short of that parser. The differential harness (010) is what would
surface a client that reads it.

**That divergence is narrower than it read, and 009 T8 is what narrowed it.** It applies to the
`"$"` entry whose message is a *parser's* — the text that is not JSON — and not to the two `"$"`
messages of the table above, which are a deserialiser's and a converter's: both are templates over
values this server already has, so both are reproduced exactly. What cannot be matched is a byte
offset into a document that failed to parse; what can be is a sentence naming a type and a position
inside a token.

The split is not arbitrary: the empty ones are produced before the framework's controller pipeline
runs, the JSON ones by that pipeline, and the last two by a controller inside it — the fixed
25-byte body where it refused abstractly, the quoted string where it wrote a message.

**The third shape answers a `404` as well, and that was measured at 008 T6** on 2026-08-29. An
identifier no library holds answers `404`, `text/plain` with no charset and the fixed 25 bytes on
all four `stream` routes, while the *same* identifier on `GET /Items/{itemId}/PlaybackInfo` answers
problem details `[probe: tools/probe_range_matrix.py, Jellyfin 10.11.11, 2026-08-29]`. One feature,
one identifier, two bodies. The delivery controller throws its own exception before the framework's
not-found result is ever reached `[source: Jellyfin.Api/Helpers/StreamingHelpers.cs:111 @
v10.11.11]`, which is the same "decided by where it happened" rule the table states — worth writing
down because the third shape had been met only at `4xx`s that were not `404`s, and "an item that
could not be found is problem details" reads like a rule until this pair breaks it.

**And at `400` and `500` as well, measured at 008 T7.** The delivery routes answer *every* refusal
they decide themselves in this one shape: a `mediaSourceId` naming no source is a `400` and an
unparseable one a `500` (§3.9), while a container no muxer writes — `stream.banana`,
`?container=banana`, or `stream.mp3` on a film — is a `500` carrying `Accept-Ranges: none`, because
the produced path writes that header before it asks the encoder for anything
`[probe: tools/probe_progressive_delivery.py, Jellyfin 10.11.11, 2026-08-29]`. Four statuses, one
body: on these routes the status *is* the whole difference, which is what the golden responses
compare bytes for.

**The `errors` map's *message* is per annotation, and one more of them is reproduced exactly.** A
value failing a declared **pattern** answers `The field container must match the regular expression
'^[a-zA-Z0-9\-\._,|]{0,40}$'.` — the expression itself, not the value the client sent, with the
apostrophes escaped as `\u0027` like every other quotable character (§1.16). Measured on
`/Videos/{id}/stream.a%20b` and on a forty-one-character container, and reproduced here rather than
recorded as a divergence because it is a template rather than a parser's output
`[probe: tools/probe_range_matrix.py, Jellyfin 10.11.11, 2026-08-29]`. The refusal also happens
**before** the item is looked up: an unknown item behind an illegal container answers this `400`.

**The fourth shape was measured at 006's plan gate**, on 2026-08-28, and it is the shape of the
image route's own `404`s: an item that exists but lacks the asked-for image type, an
`imageIndex` past the last backdrop, and a chapter with no thumbnail all answer the bare JSON
string above — while an *unknown item* on the same route answers the problem-details shape, two
rows up. One route, two `404` bodies, split by which of the two lookups failed. The string
carries the item's **display name**, and the route requires no token (§2.10) — so the name
travels to any caller holding the id, which is the id-as-capability consequence §2.10 records,
visible from another angle.

**Measured again at 006 T3, byte for byte** `[probe: manual requests via tools/_probe.py,
Jellyfin 10.11.11, 2026-08-28]`. `GET /Items/{id}/Images/Box` on an item called `#1 to Infinity`
answers 51 bytes — `"#1 to Infinity does not have an image of type Box"`, the quotes included —
and `Backdrop/99` and `Chapter/0` answer the same sentence naming **the type, never the index**.
The name is escaped like every other body (§1.16): `DW Español` goes out as `DW Espa\u00F1ol`.
One edge is not reproduced: the **all-zeros identifier** is `Guid.Empty` on the reference, which
resolves to the user's root folder and answers this route the *third* shape — `400`, `text/plain`,
the fixed 25 bytes — where any other unowned identifier answers problem details. Atrium has no
root-folder item at all, so the id is simply unknown and answers the `404`; no client sends the
empty GUID to an image route.

**The absent `charset` on the third shape is the reference's, and it is easy to lose.** JSON
responses carry `charset=utf-8` (§1.10) and this one does not; web frameworks append it to any
`text/*` type unless told otherwise, so the natural way to write this handler produces
`text/plain; charset=utf-8` and a difference on every refusal in feature 002.

**The same status carries different bytes depending on which layer refused.** An unauthenticated
`GET /Users/Me` is the empty `401` in the first row; a `401` from `POST /Users/AuthenticateByName`
for an unknown username is 25 bytes of `text/plain`. The `400` for a missing
`X-Emby-Authorization` and the `403` for a disabled account (§2.11) are that same third shape. A
golden response that compares bytes catches this; a test that asserts a status code does not, and
002 has four acceptance criteria that would otherwise have been written against the status alone.

**Depends on it:** a client branching on a body it expects to be JSON. FastAPI's own
`HTTPException` sends `{"detail": "…"}`, which is neither shape.

**Atrium does:** all four, per refusal. `traceId` is a W3C trace-context identifier and is
per-request by definition, so it is compared by shape rather than by value.

> **The empty shapes were documented here and not implemented, for three tasks.** Until 001 had
> routes there was no path to get wrong, so nothing noticed that an unmatched path answered
> `{"detail": "Not Found"}` — the exact body two paragraphs above call neither shape. The
> route-registration work (001 T17) was the first thing to ask the question, and it asked it by
> issuing the request rather than by reading this file. A documented behaviour with no test is a
> plan, not a behaviour.

> **The absent `WWW-Authenticate` is worth keeping absent.** RFC 7235 says a 401 SHOULD carry one.
> Adding `Basic` would make a browser open a credentials dialog on routes no browser was meant to
> drive — so here, matching the reference is also the safer behaviour.

### 1.12 An unrecognised query value is ignored, not rejected

**Jellyfin does:** answer `200` with a full, unfiltered result for `/Genres?SortBy=NotASortOption`.
`[probe: tools/probe_query_envelope.py, Jellyfin 10.11.11, 2026-08-28]` The same holds across `/Items`'
enum-valued parameters — an unrecognised token in `includeItemTypes`, `sortBy`, `fields` or
`filters` drops that filter and the request succeeds — while a value that cannot parse as its
declared *type* (`limit=abc`, a malformed id) is a `400` in §1.11's problem-details shape. The
line is token-versus-type, not parameter-versus-parameter.
`[probe: tools/probe_query_envelope.py, Jellyfin 10.11.11, 2026-08-28]`

**Depends on it:** yes, and this is the measurement behind a decision already taken.
[005 §3.3](../../specs/005-item-query-api/spec.md) accepts a bounded delta — Tier 3 query
parameters are ignored rather than rejected — on the argument that rejecting turns a partial answer
into no answer *and is itself a delta*. That argument was reasoned; this is the evidence.

**Atrium does:** the same, and counts what it ignored (010 §3.6).

**And the rule does not extend to a request *body*.** An unrecognised enum token inside a posted
JSON document is a `400` in §1.11's problem-details shape, not a dropped value: a
`PlaybackInfo` body whose codec profile states `"Property": "NotAThing"` is refused, keyed by the
JSON path of the offending value `[probe: manual requests via tools/_probe.py, Jellyfin 10.11.11,
2026-08-29]`. The leniency above belongs to the *query* binder, and a server that generalised it
to bodies would accept a profile the reference rejects — which is the failure direction that
matters, since the client would then be served an answer negotiated against a profile it does not
have. Atrium declares the profile vocabulary as enums so the refusal is the framework's own; the
error map's keys are its `""`/parameter-name shape (§1.11) rather than the reference's JSON paths,
which name .NET types no client can act on.

### 1.17 A forgiven dimension re-encodes; a bare `quality` does not

**Jellyfin does:** answer an image request that changes nothing with the source file's **own
bytes** — and a request carrying a *non-positive* dimension with a re-encode at the source's own
size. On an 800×800 JPEG poster of 84,351 bytes
`[probe: tools/probe_image_formats.py, Jellyfin 10.11.11, 2026-08-28]`:

| Request | Status | Delivered | Bytes |
|---|---|---|---|
| no parameters | `200` | 800×800 JPEG | 84,351 — the file |
| `maxWidth=800`, `maxWidth=3200` | `200` | 800×800 JPEG | 84,351 — byte-identical to the file |
| `quality=90`, nothing resized | `200` | 800×800 JPEG | 84,351 — byte-identical |
| `format=Svg&maxWidth=400` | `200` | 800×800 JPEG | 84,351 — byte-identical, the resize ignored |
| `maxWidth=-100`, `maxWidth=0`, `fillWidth=-5` | `200` | 800×800 JPEG | **282,225 — re-encoded** |

Two things follow, and neither is reachable by reading §1.12. A parameter that is *forgiven*
(§1.12's lenient shape, and spec [006 §3.2](../../specs/006-images/spec.md)'s `maxWidth=-100`
row) is not the same as a parameter that was *never sent*: the forgiven one still puts the request
on the encoder's path, at the reference's own default quality — which here is three times the
size of the file it re-encoded. And `quality` alone does **not**: it moves the byte count only
when something else already triggered a transform (`quality=10` at `maxWidth=400` measured
4,039 bytes against 119,366 unqualified, same probe).

**How it was found:** by subtracting two numbers that had been printed side by side since the
OQ-5 measurement on 2026-08-28. The probe reported `maxWidth=-100 → 200 … 282225B` four lines
below `source, no parameters → 200 … 84351B`; both say `800x800`, both say `image/jpeg`, and
nothing compared them. The probe now compares payloads to the source's bytes rather than
eyeballing a size.

**Depends on it:** no client can. The delivered image has the same status, dimensions, format and
`Content-Type` either way; only `Content-Length` and the pixels' compression differ.

**Atrium does:** serve the source's bytes on **all five** rows, the non-positive one included.
The bare `quality` and `format=Svg` rows are reproductions. The last row is a deliberate
divergence, and the argument is that there is no third option: two encoders never agree on bytes,
so Atrium cannot reproduce 282,225 bytes by re-encoding either — it can only spend CPU to deliver
a *different* wrong number and a generation of quality loss. Serving the file matches on
everything a client reads and on the one thing it renders. Recorded rather than assumed, per
§3.0.3's shape of a safe divergence: same status, same dimensions, same format, no field added or
removed. [006 plan §6.3](../../specs/006-images/plan.md#63-the-transform-decision) steps 1 and 5
are written from this measurement.

### 1.13 The `CamelCase` profile really is camelCase

**Jellyfin does:** answer `Accept: application/json; profile="CamelCase"` with **camelCase property
names**, and echo the matched profile in the response's content type. The other two declared types
answer in PascalCase. `[probe: tools/probe_content_type_profiles.py, Jellyfin 10.11.11, 2026-08-26]`

| Request `Accept` | Property names | Response `Content-Type` |
|---|---|---|
| `application/json` | PascalCase | `application/json; charset=utf-8` |
| `application/json; profile="PascalCase"` | PascalCase | `application/json; profile="PascalCase"; charset=utf-8` |
| `application/json; profile="CamelCase"` | **camelCase** | `application/json; profile="CamelCase"; charset=utf-8` |

Three declared content types, two behaviours. The profile selects an output formatter with a
different naming policy, and the registration comment says so in as many words — *"Allow requester
to change between camelCase and PascalCase"*.
`[source: Jellyfin.Server/Extensions/ApiServiceCollectionExtensions.cs:126-129,
Jellyfin.Api/Formatters/CamelCaseJsonProfileFormatter.cs:15-18,
src/Jellyfin.Extensions/Json/JsonDefaults.cs:21,55-58 @ v10.11.11]`

Four measured details that a reimplementation needs and that reading the document does not give:

- **The match is on the media type's parameter, leniently.** `profile=CamelCase` unquoted and
  `profile="camelcase"` both match; a **charset parameter alongside it does not** — that request
  falls back to plain `application/json`. An unknown profile falls back too.
- **Ranking is ordinary content negotiation.** `application/json, application/json;
  profile="CamelCase"` selects the first; with `q=` values the higher one wins.
- **Names are converted at every depth**, and **dictionary keys are not converted at all** —
  `ProviderIds`, `ImageTags` and `ImageBlurHashes` keep their keys, because the reference sets
  `PropertyNamingPolicy` and never sets `DictionaryKeyPolicy`.
- **The conversion is .NET's, not "lower the first letter".** A leading run of capitals lowers all
  but the last of them. Over the ~~1043~~ **1026** names of the pinned document the two rules
  disagree exactly
  **once** — `UICulture` becomes `uiCulture`, and lowering the first letter would give `uICulture` —
  and that one name is the one that was measured. The other name with a leading run, `ETag`,
  becomes `eTag` under both rules, which is why the difference is so easy to miss: a spot check
  almost certainly lands on a name where the wrong rule is right.

> **Amended 2026-09-03, at 001's closing audit.** The count above read **1043** and the pin moved
> under it. `docs/compatibility/property-names.json` declares and holds **1026**, and
> [reference-target.md §1](reference-target.md#1-the-pinned-version) records why on 2026-09-01: the
> 1043 was an extraction of one server's `10.11.10` document taken while that server carried a
> Trakt plugin this project's reference does not, and the move to the `10.11.11` document dropped
> nineteen names and gained two. That entry counted the cost of the move and this sentence was not
> among the things it recounted. **The conclusion is unaffected** — `UICulture` is the single
> disagreement under either index, and this project's own test asserts it over the 1026
> `[spec: docs/compatibility/property-names.json, extracted from the pinned 10.11.11 document]`.
> Corrected rather than deleted, because a number that moved silently is exactly the kind of thing
> the amendment paragraphs exist to make visible.

**Depends on it:** neither analysed client sends the profile, and both were checked rather than
assumed. music-client decodes with a PascalCase naming strategy of its own and sets no `Accept`
profile. video-client generates its API client from the OpenAPI document after a build step that
**deletes the `profile=` content types** — they produced unusable generated method names — so its
generated code cannot ask for one. That build step's comment gives the same reason this repository
did: all three point at the same schema. Both projects read the schema and inferred the behaviour.

But a client that did send it and got PascalCase would not get a degraded response; it would get an
**empty object** out of its decoder, which is the failure mode of §1.1 exactly.

**Atrium does:** the same, all of it — the lenient match, the charset that stops it, the ranking,
the echo, and the conversion at every depth with dictionary keys untouched. The conversion is
applied by the response models themselves rather than to the finished body, because that is the
only place a property name can still be told apart from a dictionary key; a conversion applied to
the bytes would rename `ProviderIds`' keys and be wrong the first time feature 005 returns one.

The rule was verified rather than reasoned: **293 property names were read from nine endpoints
under both profiles, and all 281 conversions agreed** with the implementation — the eight names
left alone being exactly the dictionary keys.
`[probe: tools/probe_content_type_profiles.py, Jellyfin 10.11.11, 2026-08-26]`

> **This is what reading a schema instead of a server costs.** The claim it replaces —
> "answers all three identically" — carried a `[spec: …]` citation, was true of the *schema*, and
> was wrong about the server. It also had a passing conformance test, which asserted that Atrium's
> three answers agree with each other. They did. Nobody had asked the reference.

### 1.14 Paths match case-insensitively, and tolerate one trailing slash

**Jellyfin does:** route `/system/info/public`, `/SYSTEM/INFO/PUBLIC` and `/System/info/Public` to
the same handler as `/System/Info/Public`, and answer `/System/Info/Public/` with the same body.
`/System/Info/Public//` is a `404`. ASP.NET Core's routing compares path segments without regard to
case; the trailing slash is one empty segment, and two are one too many.
`[probe: tools/probe_routing.py, Jellyfin 10.11.11, 2026-08-26]`

**Depends on it:** any client that lowercases a URL before sending it, and there is a long tail of
reasons one might — a hand-written path literal typed in the wrong case, a URL normaliser, a proxy
that canonicalises, a configuration file someone edited. None of them are exotic, and a client
built against the reference has never had a reason to notice it was relying on this.

**Atrium does:** the same, by rewriting a request's path to the route's own spelling before routing
it. Only the segments a route declares literally are respelled; a path **parameter** is data and
reaches the handler exactly as it arrived, because lowercasing an identifier is invisible until
something case-sensitive reads one.

> **The framework's default here is not neutral, it is a third behaviour.** Starlette answers an
> unmatched trailing slash with a `307` redirect — a round trip the reference does not make — and
> answers the *doubled* slash with a `307` to a URL that works, where the reference refuses. So
> "leave the default alone" would not have meant "differ in one small way"; it would have meant
> differing in two directions at once.

### 1.15 Query parameter names match case-insensitively

**Jellyfin does:** treat `Limit=1`, `limit=1` and `LIMIT=1` as the same parameter, and a
lowercased `sortby=PremiereDate&sortorder=Descending` reorders `/Items` exactly as the PascalCase
spelling does. ASP.NET Core's query binding compares parameter names without regard to case, the
same way §1.14's routing compares path segments.
`[probe: tools/probe_query_envelope.py, Jellyfin 10.11.11, 2026-08-28]`

**Depends on it:** the pinned document spells every parameter camelCase (`startIndex`), the
reference's own clients send PascalCase (`StartIndex`), and both work — so *every* client depends
on at least one half of this, and which half is a per-client accident.

**Atrium does:** the same, by canonicalising a request's query keys to the route's own declared
spellings before the framework binds them — the query-string counterpart of the §1.14 path
rewrite, and like it, values are data and are never touched. The framework default is a silent
third behaviour: an unrecognised spelling would not be rejected but *ignored*, which for
`StartIndex` against a camelCase route means every page is page one.

---

## 3. Defects

Principle V: the default is to replicate. Each entry states what Atrium does and why — and this
section opens with the procedure that produces those answers, because "replicate unless you have a
good argument" is not a procedure, it is a preference with a disclaimer.

### 3.0 How the decision is made

One question decides it: **can a client have built something that being correct would break?**
Principle I is about what a client can observe, not about what is correct, and this is the form
that question takes for a defect.

The table below is a fast heuristic for answering it. **Direct evidence overrides the heuristic**
— see §3.0.1.

| Class | Shape of the defect | What clients do about it | Default |
|---|---|---|---|
| **A** | **Fails loudly** — 5xx, reset connection, unparseable body | They cannot build *on* it, only *around* it: a fallback, or an error to the user | **Diverge — be correct** |
| **B** | **Succeeds wrongly** — 200 with a wrong value, wrong unit, wrong content | They build *compensating* code that assumes the wrong value | **Replicate** |
| **C** | **Omits something** — absent field, absent header | They treat it as absent, which is what it is | **Supply it**, after checking |

The reasoning behind class A: nobody can depend on a failure in a way that a *success* breaks. A
client meeting a 500 either has a fallback — which stays unused, harmlessly — or shows an error,
which it now does not have to. A client using the failure as a capability probe gets a better
answer from a working response than from a broken one.

The reasoning behind class B is the mirror image, and it is the one that catches people out.
**Fixing a wrong-but-usable response invalidates the workaround built on top of it.** A client that
compensates for a wrong value by correcting it produces a doubly-corrected value once the server is
fixed. Being right is not automatically safe.

Class B has two escape hatches, and between them they are where most real cases land.

**If every plausible compensation is defect-tolerant, diverge.** A client that ignores a field it
knows to be unreliable is unaffected when the field becomes reliable. A client that *sniffs* rather
than assumes is unaffected either way.

**If no compensation is possible at all, diverge.** A class-B defect that a client cannot work
around behaves like class A in the only respect that matters: replicating it protects nobody,
because there is nobody to protect. §3.4 is exactly this case, and it is not rare — a defect that
forces a client into a corner is precisely the sort that gets noticed and reported.

### 3.0.0 Replication is not free, and for this project it is not the lazy option

For a fork, replicating a defect means leaving code alone. **For Atrium it means writing extra code
to reproduce a mistake**, and then a test to prove the mistake is faithfully reproduced, and then a
comment explaining to the next reader why it is there.

That cost is real and it belongs in the decision. It does not override Principle I — a client that
would break still outranks the tidiness of our source — but where the evidence is balanced, the
side that requires deliberately writing bug code needs the stronger argument, not the weaker one.

### 3.0.1 The tie-breaks

When the class does not settle it, in this order:

1. **Does a known client compensate, and does the compensation break?** Answerable for the clients
   this project can read, and for the rest it is what the differential harness is for. Absent
   evidence, assume a compensation exists.
2. **What is upstream's position?** Three answers, and they are not the same:

   | Upstream state | What it tells us | Weight |
   |---|---|---|
   | **Fixed** | The behaviour is a transient, not a contract. Replicating means matching something on its way out and undoing it at the next pin bump — and any client depending on it is already broken against the next release | Toward **diverge** |
   | **Judged and deliberately kept** | The behaviour is durable, and someone who knows the codebase decided it is right or that changing it costs too much | Toward **replicate** |
   | **Not judged** | Nothing at all | **None** |

   **"Not judged" includes a closed pull request.** A PR can be closed for scope, for process, for
   maintainer bandwidth, or because nobody got to it — none of which is a ruling on the behaviour.
   Reading a rejection as a verdict is the easiest mistake in this whole procedure, and §3.4 is a
   case where it would have produced exactly the wrong answer.

   [ADR-0004](../decisions/0004-pin-to-jellyfin-10-11.md) pins the contract to `10.11.x`; upstream
   head does not define the contract, but it does define the **direction**.

   ⚠️ **This tie-break must not become "wait for upstream".** This project cannot reliably move
   upstream state — see §3.4 — so a defect that is fixable in principle may be unfixable in
   practice, and a decision deferred on that basis is a decision never made.
3. **Is the defect in a code path v1 has at all?** If not, the decision is recorded and deferred
   rather than argued now. A decision made about code nobody will write for a year is a decision
   made with the least information it will ever have.

### 3.0.2 What is never acceptable

- **Inventing a third behaviour.** Where the reference 500s and correctness says 200, the choice is
  those two. A tidy `400` is worse than both: it is a delta from the reference *and* from correct.
- **Fixing a defect because it is obviously wrong.** Obviousness is not evidence. Principle V.
- **Replicating a defect without recording it**, so that a later contributor "fixes" it.
- **Deciding once for a whole endpoint.** Defects are per behaviour, and one endpoint can carry two
  that need opposite treatment — see §3.2, which is exactly that.
- **Treating a closed pull request as a ruling.** See tie-break 2.

### 3.0.3 The shape of a safe divergence

Where a divergence is unavoidable, its **shape** decides how much risk it carries. Best first:

1. **Gated on an explicit client declaration.** The new behaviour happens only for clients that
   said something specific about themselves. A client that never makes that declaration cannot
   observe any difference — the divergence is invisible to everyone who did not ask for it. §3.4 is
   this shape, and it is why that divergence is cheap.
2. **Strictly more correct on a path that previously failed.** Nothing to break, because nothing
   worked. §3.2 symptom 1.
3. **Strictly more information in a place clients read optionally.** A header that was absent, a
   size that was unknown. §3.3.
4. **Changing a value clients already read.** The dangerous shape. Needs evidence about
   compensations, not reasoning about correctness. §3.2 symptom 2.

Prefer 1. Where a defect can be diverged from *conditionally*, do that rather than diverging
unconditionally, even when the unconditional version looks cleaner.

### 3.1 `TotalRecordCount` is 0 on by-name endpoints without `limit` — class B

**Jellyfin does:** `/Artists`, `/Artists/AlbumArtists`, `/Genres`, `/MusicGenres` and `/Studios`
share the `GetItemValues` path, which **disables counting when the request has no `limit`**:

```
/Artists?UserId=…             -> TotalRecordCount=0    Items=684
/Artists?UserId=…&limit=1000  -> TotalRecordCount=684  Items=684
```

`[probe: tools/probe_by_name_counts.py, Jellyfin 10.11.11, 2026-08-28; upstream
jellyfin/jellyfin#17541]` — first seen on `master` on 2026-08-05, and the probe run settles that
the pinned line has it too, on all five shared-path endpoints.

`/Years` has its own face of the same defect, measured on the pinned line: without a `limit` it
answers a count that is neither zero nor the row count — `TotalRecordCount: 9754` beside 97 rows
on the measured library — so the field is unreliable across the whole family, each route in its
own way. `[probe: tools/probe_by_name_counts.py, Jellyfin 10.11.11, 2026-08-28]`

**Depends on it:** no. Known clients map `Items` and ignore `TotalRecordCount` on these routes —
precisely because it is unreliable. A client that *paginated* on it would be broken today.

Class B, but through the escape hatch of §3.0: the only compensation anyone builds for an
unreliable field is to ignore it, and ignoring a field is defect-tolerant — it does not stop
working when the field starts being right.

**Atrium does:** **diverge — always return the true count.** The argument required by Principle V:
no client can observe the difference in a way that changes its behaviour, because a correct count
is what a client that reads the field already expects, and the clients that ignore the field are
unaffected. The upstream fix is approved, so replicating the defect would mean deliberately
matching a behaviour that is on its way out.

### 3.2 PCM/WAV output — one bug, two symptoms, two classes

The worked example for §3.0, and it is a good one because the naive answer — "be correct" — is
right for one symptom and wrong for the other.

**The cause is a single block** of `EncodingHelper.GetProgressiveAudioFullCommandLine`, which for
any `pcm_*` encoder forced the raw muxer `-f s16le` and fed `-ar` from `AudioBitRate` — the wrong
field, and an optional one `[source:
MediaBrowser.Controller/MediaEncoding/EncodingHelper.cs:7773-7776 @ v10.11.11]`, `[probe:
tools/probe_universal_audio.py, Jellyfin 10.11.11, 2026-08-29]`. The fix is
`jellyfin/jellyfin#17537`, merged to master on 2026-08-05 and in no 10.11.x — so §3.0.1's
tie-break 2 reads **fixed upstream** for both symptoms below.

Two symptoms come out of it. **Both were carried as `[prior-probe: 2026-08-03]` until 008 T9 wrote
the battery, and writing it moved both of them**: symptom 1 turned out to have two causes, and
symptom 2 turned out to name a parameter that does not produce it. Neither symptom belongs to one
route family — the split below is by *whether the request carried an `AudioBitRate`*, not by
which route was called.

#### Symptom 1 — a PCM request with nothing to put in `-ar` returns 500

Two ways in, one status, measured on the same server in one run `[probe:
tools/probe_universal_audio.py, Jellyfin 10.11.11, 2026-08-29]`:

* **`GET /Audio/{id}/stream.wav` naming no codec at all.** The codec is inferred from the part of
  the path after its last dot, and the inference table has no `wav` row, so it answers `wav` —
  which `GetAudioEncoder` passes straight to `-acodec` because it is a well-formed container name
  `[source: MediaBrowser.Controller/MediaEncoding/EncodingHelper.cs:667-684, 746-806 @ v10.11.11]`.
  There is no encoder called `wav`. This request never reaches the PCM block at all, and the
  entry's first wording missed it.
* **Any `pcm_*` codec sent without an `AudioBitRate`.** `-ar` is built from an absent field, the
  argument degrades to a bare `-ar`, and ffmpeg aborts before its first frame. Reached identically
  by `stream.wav?audioCodec=pcm_s16le`, by `stream?container=wav&audioCodec=pcm_s16le`, and by
  `/universal` with a `wav` transcoding container.

**Class A.** A client cannot build on a 500. Whatever it does today — fall back to FLAC, show an
error — keeps working when the request succeeds instead.

**Atrium: diverge. Serve valid WAV**, with a RIFF header, a real `Content-Length` and `Range`
support.

#### Symptom 2 — the same request *with* a bitrate returns headerless PCM

`200`, `Content-Type: audio/wav`, and a body with no `RIFF` header, because the raw muxer was
applied regardless of the container the client asked for. The bitrate is what separates this from
symptom 1: it gives `-ar` something to consume, so the command is well formed and the encoder
runs. Measured on `GET /Audio/{id}/stream.wav?audioCodec=pcm_s16le&audioBitRate=128000` and on
`GET /Audio/{id}/universal` with `transcodingContainer=wav` and the same pair — **the transcoding
container, not `Container`**: `Container=wav` is `/universal`'s direct-play list, and a source it
does not cover transcodes to the default target, answering `audio/mpeg`. This entry said
`Container=wav` until the battery was written `[probe: tools/probe_universal_audio.py, Jellyfin
10.11.11, 2026-08-29]`.

**Class B, and the trap is real.** A client compensating for this must **synthesise a RIFF header
and prepend it** — it has no other way to make the stream playable. Send a correct header to that
client and it produces **two**, which is corrupt audio. "Being correct" breaks it.

So the class default says replicate. The tie-breaks say otherwise, and both are needed:

1. **Does a known compensation break?** A workaround cannot assume which server version it is
   talking to, so a competent one sniffs for the `RIFF` magic bytes before prepending. A sniffing
   compensation is defect-tolerant and survives the fix. One that prepends blindly does not — and
   that client is **already broken against upstream head**, where the fix has landed.
2. **Is it fixed upstream?** Yes, in `master`. Replicating a headerless `audio/wav` would mean
   deliberately emitting a malformed body to match a version that no longer emits it, and undoing
   that at the next pin bump.

**Atrium: diverge. Serve a real RIFF header** — and record that this one carries a risk symptom 1
does not: a client that blindly prepends a header receives corrupt audio. If the differential
harness ever finds such a client, this decision is revisited, not defended.

#### The divergence cannot be served chunked, and that decides how it is implemented

A WAV file states its own length twice — the `RIFF` header's size and the `data` chunk's — and a
muxer writing to a pipe can never go back to fill either in: ffmpeg writes `ffffffff` into both
and exits `0`. Two invocations of the same conversion, one to a file and one to a pipe, differ in
exactly those eight bytes and in nothing else (measured, 2026-08-29). So "serve valid WAV with a
real `Content-Length`" is not a header the delivery code adds on top of the chunked answer of
§3.3 — it is a different production shape: the output is produced to scratch and then served
whole, which is where the length and the `Range` come from. `media/ffmpeg.py` refuses to build the
piped invocation rather than leaving that to a caller to remember, and the refusal is asserted in
`tests/unit/test_media_ffmpeg.py` beside the measurement in
`tests/conformance/test_wav_delivery.py`.

**One more row the reference has not got.** Its codec inference answers an unlisted container with
the container's own name, so nothing anywhere maps `wav` to a PCM encoder — which is symptom 1's
first cause. Atrium supplies that row, and it has to: ffmpeg's wav muxer *accepts* a FLAC stream
under a codec tag and writes a genuine `RIFF` header over it, so a bare `stream.wav` that fell
back to the source's codec the way a bare `stream.mkv` does would pass every "is it RIFF" check
and play nowhere (measured, 2026-08-29).

#### Status in v1

**Both paths are served in v1.** Producing PCM requires re-encoding, and transcoding entered v1 on
2026-08-27 ([008 §2](../../specs/008-playback-negotiation-and-delivery/spec.md)), so the deferral
that used to close this section no longer applies: Atrium answers both routes with valid WAV — a
real RIFF header, a real length, `Range` support — as decided above, and
[008 AC-20](../../specs/008-playback-negotiation-and-delivery/spec.md#5-acceptance-criteria) is
where it is asserted. The reasoning was written while it was fresh and was waiting for the code;
this is the case it was written for. Implemented at 008 T9, which is also where both
`[prior-probe:]` citations above became `[probe:]` ones — this section now carries none.

### 3.3 Progressive transcoding responses carry no `Content-Length` or `Accept-Ranges` — class C

**Jellyfin does:** streams **progressive** transcoded and remuxed output chunked, with no size
and no range support: `GetTranscodedFile` forces `Accept-Ranges: none` and sets no
`Content-Length` — the transcode is written out as ffmpeg produces it
`[source: Jellyfin.Api/Helpers/FileStreamResponseHelpers.cs:123-135 @ v10.11.11]`. Measured on
`/Audio/{id}/stream.mp3`, on a resampling `stream.flac`, and on a stream-copy
`/Videos/{id}/stream.mp4` — chunked, `Accept-Ranges: none`, no length, even where the remux's
size is knowable. **Its HLS segments are the opposite**, and this entry's first wording missed
that: a finished segment answers `Content-Length`, `Accept-Ranges: bytes` and byte-identical
retries, and the playlists carry lengths too
`[probe: tools/probe_hls.py, Jellyfin 10.11.11, 2026-08-28]`.

**Depends on it:** negatively — DLNA renderers refuse a stream with no size, which is why clients
that cast run a local sizing proxy.

**A `Range` on a progressive answer is not merely unhonoured, it is unread.** Measured at 008 T7
on a remux: `bytes=100-199`, a suffix range, a single byte and an unreadable `bytes=abc-def` each
answer the identical `200` with no `Content-Range` and the body from its first byte
`[probe: tools/probe_progressive_delivery.py, Jellyfin 10.11.11, 2026-08-29]`. That is the reading
[008 plan §6.8](../../specs/008-playback-negotiation-and-delivery/plan.md#68-measured-at-the-gate-and-what-stays-owed)
left owed: the *sized* case has five shapes with two answers, and the chunked case has one answer
for every shape there is.

**Atrium does:** HLS exactly as the reference — sized, range-capable segments are parity now,
not a divergence — and **diverges on the progressive routes wherever the size is knowable**:
remuxed output whose size is computable or which is written somewhere seekable sends
`Content-Length` and honours `Range`. Same reasoning as §3.2 — a client cannot branch on a
response being more correct. Implemented at 008 T7: a remux is produced to scratch under a name
derived from the command and the file's change signal, so the size is known before the first byte
leaves and a `Range` is served from what the first request produced. The divergence is exactly a
size and a range unit — the **progressive** answer carries no `Last-Modified`, because the
reference sends none there and bytes that did not exist a second ago have no modification time
worth inventing.

**A segment does carry one**, which is the other half of the same sentence and had to be measured
rather than inferred from it: a finished segment answers `Content-Length`, `Content-Type`,
`Accept-Ranges: bytes` and `Last-Modified`, and no `ETag` — the static answer's four headers
exactly, because the reference serves the finished file the way it serves any file
`[probe: tools/probe_transcode_session.py, Jellyfin 10.11.11, 2026-08-29]`. Atrium sends the same
four (008 T11).

**The one place Atrium does not diverge** is a progressive re-encode whose final length is unknown
until the last frame. That answers chunked, exactly as the reference does, because the alternative
is inventing a number, and a wrong `Content-Length` truncates playback.

**Except where the container will not have it**, found at 008 T9 and worth stating here because
it makes "sized" a property of the *output* rather than of the decision: a `wav` answer is a
re-encode and is sized all the same, because a WAV header states its own length and a piped one
would state `ffffffff`. There is no chunked form of that container to be faithful to. So the rule
is *send the size when it is known*, plus *produce somewhere seekable when the body would
otherwise lie about it* — see §3.2.

**And the pipe costs a container its own self-description, which is this feature's one divergence
pointing the wrong way.** Everything above is Atrium being *more* correct than the reference; this
is the opposite, and it was found by an audit of a first-party client rather than by a probe. The
reference produces progressive output to a **file** — `state.OutputFilePath`, served by a
`ProgressiveFileStream` over that path as it grows `[source:
Jellyfin.Api/Helpers/FileStreamResponseHelpers.cs:133,165 @ v10.11.11]` — and Atrium produces to a
pipe for everything outside `NEEDS_SEEKING`. ffmpeg reserves the frame a container describes itself
in and seeks back to fill it at the end; to a pipe it cannot, so it writes none:

| Container | To a file | To a pipe |
|---|---|---|
| `mp3` | A reserved frame right after the ID3 tag, carrying an `Info` tag at byte 65, a frame count, and the encoder string `Lavc` | **No `Xing` and no `Info` frame anywhere** — the first audio frame follows the ID3 tag directly, and the two bodies differ by exactly one frame (417 bytes at 128 kbps) |
| `flac` | `STREAMINFO` with the sample count and the stream's MD5; `ffprobe` reads `3.000000` | `total_samples = 0` and an all-zero MD5; `ffprobe` reads the duration as `N/A` |

*(Both measured locally with ffmpeg 9.0.1, 2026-08-29, encoding the same three seconds twice — this
is a property of the muxer and of where Atrium points it, not a claim about Jellyfin.)*

**Depends on it:** yes, and concretely. A gapless music client reads the MP3 header frame to trim
the encoder's padding, and its table has four branches — a complete header, a blank one from a
recognised encoder, a blank one from an unrecognised encoder, and an `m4a` packet table. *No header
frame at all* is a fifth case it does not have, so no branch fires and the user hears a microcut at
every track change
([client-embeat-mobile §5.3](client-embeat-mobile.md#53-a-piped-mp3-carries-no-xing-frame-which-is-not-the-blank-one-the-client-measured),
traced from that client's own source on 2026-08-29).

**Atrium does:** nothing yet, and this row is the record that it is a **parity gap and not an
improvement** — the fix moves towards the reference and needs no Principle I argument, unlike the
sizing divergence above. `NEEDS_SEEKING` correctly did not catch it: its rule is *a body that would
lie about its own length*, and a piped MP3 does not lie, it omits. **Closing mechanism:** produce
the progressive re-encode somewhere seekable and stream the result, which is the same lever §3.2
already pulls for `wav` — and which is entangled with two asks that are *not* parity (an honest
`Content-Length` on a capped stream, and caching a chunked transcode), so it is one decision rather
than three. The claim that the reference's own progressive MP3 carries a *blank* Xing frame is a
third-party measurement and a **lead**: it decides whether the target is "blank, like the reference"
or better than it, and no probe here has checked it.

### 3.4 HDR10+ metadata stripped from clients that asked for it — class B, no compensation

The second worked example, and it is the one where the heuristic gives the wrong answer and the
evidence rescues it.

**Jellyfin does:** during a **stream copy**, plans removal of the HDR10+ SEI from any
`DOVIWithHDR10Plus` video stream as soon as the client's requested range types contain `DOVI` —
without considering that the client may have declared `DOVIWithHDR10Plus` itself. The removal is
carried out with `-bsf:v hevc_mp4toannexb,hevc_metadata=remove_hdr10plus=1`, and the rewritten
bitstream **breaks HLS fMP4 playback on AVPlayer**.
`[source: MediaBrowser.Controller/MediaEncoding/EncodingHelper.cs @ v10.11.11; the removal logic
is unchanged since jellyfin/jellyfin#13277 and still unchanged as of master post-#17571; related
upstream issue jellyfin/jellyfin#16687]`

**Class B by shape** — a `200` and a stream, not a failure.

**But no compensation exists.** No device profile combination avoids the strip while keeping Dolby
Vision remuxing, because the `dvh1` tagging in `DynamicHlsController` requires `DOVI` in the range
type. The client is cornered: declare `DOVI` and lose HDR10+ along with playback, or do not declare
it and lose Dolby Vision remuxing entirely. **Escape hatch 2 of §3.0 applies** — a class-B defect
nobody can work around protects nobody when replicated.

**Upstream position: not judged.** A fix was proposed and the pull request closed under the
project's LLM/AI development policy. The automated quality gate passed; no reviewer assessed the
behaviour. That is a process outcome, not a ruling, and tie-break 2 therefore contributes
**nothing** — reading it as "upstream considered this and declined" would invert the decision on
evidence that does not exist.

> This is the concrete case behind the warning in §3.0.1: **this project cannot reliably move
> upstream state.** Contributions authored with AI assistance are not accepted upstream, and every
> commit in this repository carries a `Co-Authored-By` trailer. A defect that is fixable in
> principle is therefore not fixable in practice by this route, and waiting for upstream is not a
> plan. The way to obtain a judgement is an **issue describing the behaviour**, hand-authored,
> proposing no code — a different artefact, subject to a different policy, and the one that gets
> the defect assessed on its merits.

**Atrium does: diverge, in the safest available shape.** Honour an explicit `DOVIWithHDR10Plus`
declaration: keep the metadata for clients that made it, strip it for clients that did not, exactly
as the reference already treats the neighbouring `DOVIWithELHDR10Plus` coexistence case. This is
shape 1 of §3.0.3 — **a client that never declares `DOVIWithHDR10Plus` cannot observe any
difference at all**, which is what makes the divergence cheap rather than merely justified.

**In scope for v1**, and it always was — stream copy is remuxing, which no scope change ever
excluded ([008 §3.3](../../specs/008-playback-negotiation-and-delivery/spec.md)). And §3.0.0 applies with
force here: Atrium never had this defect, so replicating it would mean writing a bitstream filter
whose only job is to remove something the client said it wanted.

---

### 3.5 `/Users/Public` discloses every user's policy to anyone — class B, replicated

**Jellyfin does:** answer `GET /Users/Public` with the **whole user object** — `Configuration` and
`Policy` included — to a caller carrying no token at all, byte-identical to the authenticated
response. `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]` All 42 policy properties and all 16 configuration properties, for every
user not marked hidden, to anybody who can reach the port.

**Depends on it:** unknown, and that is the crux. A login screen needs `Name`, `Id`,
`PrimaryImageTag` and `HasPassword`. Nothing in the surface's named consumers is known to read
`Policy` from this route — and §3.0.1 tie-break 1 says that absent evidence, assume a compensation
exists. A client reading `Policy.IsAdministrator` here to decide what to show before login is not
far-fetched.

**Atrium does:** the same, and this is the entry that most deserves re-reading.

The class is **B** — it succeeds, with more than it should. §3.0's question is whether a client can
have built something that being correct would break, and here it plainly can: omitting two
properties is exactly the shape that breaks a decoder expecting them.

Against that: this is a **disclosure**, not a wrong number, and the argument for diverging is not
that the reference is untidy but that replicating it publishes information about every account on
the server. §3.0.2 forbids fixing a defect *because it is obviously wrong*, and "it discloses too
much" is close enough to obviousness to be worth naming as the temptation it is.

So the default holds — Principle V, replicate — and the divergence stays available and written down
rather than argued again from scratch. If it is ever taken, its shape is a middling one on §3.0.3's
list: strictly *less* information, on a route no known consumer reads those properties from, which
is the least dangerous kind of change to make and still not free.

**And it would have to be taken twice.** The same disclosure is reachable through
`GET /Users/{userId}`, measured on 2026-09-01 and replicated there for these reasons plus one this
route does not have — refusing there means refusing a request that succeeds against every reference
server. That entry is
[§3.22](#322-any-authenticated-caller-reads-any-user-whole--class-b-replicated), and the two are
one decision on two roads: a divergence on either alone leaves the object published by the other.

> **This overturned an acceptance criterion, not a detail.** 002's AC-6 asserted that
> `/Users/Public` **omits** `Configuration` and `Policy`, and its §3.4 gave the reason: "this is
> pre-authentication, and it must not disclose what a user is allowed to do." The reasoning was
> sound and the premise was measured to be false. Both are corrected, and the criterion now asserts
> what the reference does.

### 3.6 Ties are engine-resolved, and paging the artist sorts loses rows — class B, diverged

**Jellyfin does:** append almost nothing after the ordering a client asked for. `Name` is chained
when the first ordering is `SortName` or `Default`; after any other ordering — a date, a play
count, an artist — **no further key is ever added**, not even the id
`[source: Jellyfin.Server.Implementations/Item/BaseItemRepository.cs:1592-1652 @ v10.11.11]`. What
that costs was measured per `SortBy` over 485-row windows, three ways — request twice, page in 97s,
analyse the tie runs:

- On the movie sorts (`SortName`, `DateCreated`, `PremiereDate`, `PlayCount`, `DatePlayed`) the
  order is repeatable, pages reassemble the one-shot list exactly, and ties happen to arrive in
  ascending id order — the engine's habit, guaranteed by nothing in the source.
- On the artist sorts (`AlbumArtist`, `Artist`, whose key lives in a joined table) the same
  request repeats identically, **but the concatenation of its pages is not the one-shot list**:
  ties resolve differently at different offsets, so a client paging a large audio library sees
  some items twice and never sees others.
- `Random` is a fresh shuffle on every request — two identical 97-row requests shared 4 items —
  matching the seedless per-row random in the source
  `[source: Jellyfin.Server.Implementations/Item/OrderMapper.cs:33 @ v10.11.11]`.

`[probe: tools/probe_sort_stability.py, Jellyfin 10.11.11, 2026-08-27]`

**Depends on it:** no, and this one cannot be depended on. An order that differs between a paged
and an unpaged read of the same data is not a value a client can build compensating code against —
the only compensations are to fetch everything in one request or to tolerate duplicates and gaps,
and both are defect-tolerant: neither stops working when the order becomes total. That is §3.0's
first escape hatch, the same one §3.1 went through.

**Atrium does: diverge — every ordering is total.** The requested keys, then `Name` where the
reference chains it, then the id as the final key, so paging visits every item exactly once for
every `SortBy` ([005 §3.4](../../specs/005-item-query-api/spec.md#34-sorting)). Within any tie the
result is *an* order the reference could have produced — on the movie sorts it is the very order
the measured server does produce — so no response is distinguishable from a plausible reference
response; what changes is only that the order holds still across pages. No upstream issue is known
for this; nothing here waits on one.

### 3.7 A sample-rate ceiling is answered from the Opus ladder — class B, diverged

**Jellyfin does:** applies the sample-rate ladder Opus needs — `≤8000 → 8000, ≤12000 → 12000,
≤16000 → 16000, ≤24000 → 24000, else 48000` — to **every** audio re-encode, not only Opus: the
condition around the ladder is inverted. A `/universal` request with `maxAudioSampleRate=22050`
is answered at 24 000 Hz, **above the ceiling the client stated**; a `maxAudioSampleRate=44100`
against a 96 kHz source would land at 48 000 the same way.
`[probe: tools/probe_universal_audio.py, Jellyfin 10.11.11, 2026-08-28]` The restructure that
scopes the ladder to Opus is merged upstream in the same change as §3.2's PCM fix, and is in no
10.11.x.

**Depends on it:** no client benefits from receiving a higher rate than its declared maximum — a
ceiling is declared because something downstream cannot go above it, and the failure lands at the
client's decoder, far from the cause. A compensating client would have to resample locally, and
that compensation survives the fix untouched.

**Atrium does: diverge — honour the ceiling exactly.** The output sample rate is the stated
ceiling when the source exceeds it, the source's own rate otherwise. Same reasoning as §3.2:
fixed upstream, and no blind compensation exists that a correct answer would break.
[008 §3.6 and AC-19](../../specs/008-playback-negotiation-and-delivery/spec.md) carry it.

### 3.8 `/universal` without `audioCodec` answers an empty 200 — class A, diverged

**Jellyfin does:** answer a `/universal` request whose `transcodingProtocol` is `http` and which
names no `audioCodec` with `200`, `Content-Length: 0` and an empty body — every retry identical,
whether the request named a `transcodingContainer` of `flac` or none at all, the latter behind
`Content-Type: audio/mpeg` `[probe: tools/probe_universal_audio.py, Jellyfin 10.11.11,
2026-08-29]`.

**The mechanism is not the one this entry first recorded**, and the difference decides what a
correct answer looks like. The transcoding profile is *not* codec-less: the controller builds it
with `audioCodec ?? "mp3"`, so the negotiation has a codec and resolves the container perfectly
well — `audio/mpeg` above is that default arriving. What has no codec is the **streaming request**
the controller then builds, which passes the raw parameter through; and a streaming request with
no codec infers one from the part of the request path after its last dot `[source:
Jellyfin.Api/Helpers/StreamingHelpers.cs:71-75 @ v10.11.11]`. On `/Audio/{id}/stream.mp3` that is
`mp3`; on `/Audio/{id}/universal` there is no dot at all, and the helper's answer to a missing
separator is *the whole string* `[source: src/Jellyfin.Extensions/StringExtensions.cs RightPart @
v10.11.11]`.

**And the last step of that mechanism is one further sentence, added at 008 T9**, because the
entry's "with or without a `transcodingContainer`" turned out to be false for a container it had
not been asked about. The path does **not** become the encoder name: `GetAudioEncoder` guards its
input with the container-validation pattern and substitutes `aac` for anything that fails it,
which a path full of slashes does `[source:
MediaBrowser.Controller/MediaEncoding/EncodingHelper.cs:41,746-752 @ v10.11.11]`. So the request
is `-acodec aac` into whatever the transcoding container is, and the empty body is that encoder
meeting a muxer that cannot carry it. `flac` and `mp3` cannot; **`wav` can**, and a `/universal`
request naming a `wav` transcoding container and no codec answers a real `RIFF….WAVE` — measured
in the same run. The observable is therefore container-dependent, and this entry's claim holds for
the containers it was measured on rather than for every one.

**Depends on it:** nothing can be built on an empty body behind a `200` — a player fed zero
bytes errors on its own side. Class A by the same logic as §3.2's symptom 1: whatever a client
does today, it keeps working when the request succeeds instead.

**Atrium does: diverge — answer the request.** When the client names no codec, the transcoding
container's own codec is the target, derived by **the reference's own inference table** given the
container instead of a dotless path — so `mp3` in and `mp3` out on both servers, and the
divergence is confined to the request that named a transcoding container and no codec, which is
the request the reference answers with nothing. Recorded in
[008 §3.6](../../specs/008-playback-negotiation-and-delivery/spec.md) and implemented at 008 T8.
For a `wav` container that table has no row and 008 T9 supplied one (§3.2), so the request the
reference answers with AAC inside a RIFF header is answered here with PCM inside it — a body on
both servers, and the codec the container exists for.

### 3.9 An unparseable `mediaSourceId` is a 500 where a well-formed one is a 400 — class A, diverged

**Jellyfin does:** answer a delivery request whose `mediaSourceId` names no source of the item in
one of **two** ways, split by whether the string happens to parse as a GUID. The resolution
compares the parameter against each of the item's sources and then, only when none matched,
*parses* it to ask whether it is the item's own identifier
`[source: Jellyfin.Api/Helpers/StreamingHelpers.cs:136-140 @ v10.11.11]`. A well-formed
identifier that matches nothing reaches the parse, survives it, leaves the media source null and
is refused as an argument failure: `400`, `text/plain`, the fixed 25 bytes. One that is not an
identifier at all throws `FormatException` out of the parse, and `FormatException` is not a type
the error middleware maps — so it falls to the default and answers `500` in the same shape
`[source: Jellyfin.Api/Middleware/ExceptionMiddleware.cs GetStatusCode @ v10.11.11]`. Measured on
both halves of the same route, `static=true` and produced, all four identical
`[probe: tools/probe_progressive_delivery.py, Jellyfin 10.11.11, 2026-08-29]`. Still present on
upstream `master` at 2026-08-07, so §3.0.1 tie-break 2 reads **not judged** and weighs nothing.

**Depends on it:** nothing that a `400` breaks. A client only ever sends a `mediaSourceId` it was
handed by a negotiation, and those are always well-formed; the `500` is reachable by a
hand-written or corrupted URL. A client meeting it either shows an error — which it still does —
or retries, because `5xx` is the retriable class, and a retry of a request that can never succeed
is not a behaviour worth preserving.

**Atrium does: diverge — the `400`, for both.** Class A's default, and the two arguments that
carry it past §3.0.2's ban on inventing a third behaviour:

- **The `400` is not invented.** Both values mean the same thing — *this names no source of this
  item* — and the `400` is the answer the reference itself gives that sentence one value away, in
  the same error shape, on the same parameter, on the same route. What is dropped is a split the
  reference draws by accident of ordering, not a behaviour it chose.
- **And the reference gives both values the `400` on the sibling route.** `GET
  /Audio/{itemId}/universal` answers `400`, `text/plain`, the same 25 bytes, to a well-formed
  identifier naming no source **and** to `banana` — measured at 008 T8, on the same server and the
  same item as the pair above `[probe: tools/probe_universal_audio.py, Jellyfin 10.11.11,
  2026-08-29]`. It resolves the source through the negotiation helper rather than the streaming
  one, so nothing parses the string in order to throw. Atrium's single answer is therefore not
  merely *derived* from the reference's — it is one of the two answers the reference already gives
  the same parameter, chosen over the one that only an accident produces.
- **Replicating it costs code that exists only to fail worse.** §3.0.0: the natural implementation
  compares the identifier against the item's sources and refuses when none matches, which is one
  refusal. Reproducing the reference would mean adding a parse whose sole purpose is to throw, and
  where the evidence is balanced the side that requires deliberately writing bug code needs the
  stronger argument.

The divergence takes §3.0.3's second shape — strictly more correct on a path that previously
failed — and is asserted as one parametrised row in
`tests/conformance/test_progressive_delivery.py` and again in
`tests/conformance/test_universal_audio.py`, so the two values are visibly one answer.
Recorded in [008 §3.5](../../specs/008-playback-negotiation-and-delivery/spec.md#35-delivery-the-rules-that-apply-to-every-route).

### 3.10 A segment's declared duration is not the duration it holds — class B, diverged

**Jellyfin does:** state two different segment lengths, one to the playlist and one to the
encoder, and never reconcile them. The media playlist scales the requested length up so a whole
number of frames fits — `ceil(3000 × ceil(rate) ÷ rate)`, the arithmetic §3.7 of
[008's spec](../../specs/008-playback-negotiation-and-delivery/spec.md#37-video-delivery) records
— and writes `#EXTINF:3.004000` on every line `[source:
Jellyfin.Api/Controllers/DynamicHlsController.cs:1425-1432 @ v10.11.11]`. The encoder is then
told the **unscaled integer**: `-hls_time 3` and forced keyframes at
`expr:gte(t,n_forced*3)` `[source: Jellyfin.Api/Controllers/DynamicHlsController.cs:1667-1680,
MediaBrowser.Controller/MediaEncoding/EncodingHelper.cs:1948-1962 @ v10.11.11]`. So each segment
declares four milliseconds more than it holds, and a 2h22 film's playlist claims eleven seconds
of media that is not in it.

**Depends on it:** nothing that could. Every HLS player reconciles `#EXTINF` against what the
segment's own timestamps say — segments of unequal length are ordinary, and the last one always
differs — so a client compensating for the gap is compensating in a way that is unaffected when
the gap closes. That is class B's first escape hatch. The stronger point is that there is nothing
to compare: the bytes are ours in one server and theirs in the other, and
[008 §6](../../specs/008-playback-negotiation-and-delivery/spec.md#6-conformance) already declines
to byte-compare produced output, because two encoders given one instruction never agree.

**Atrium does: diverge — the encoder is told the planned cadence**, so the playlist's promise and
the produced media agree. The playlist itself is unchanged: the scaled number is reproduced
exactly, so two servers answering the same request write the same `#EXTINF` lines and the
difference lives entirely inside media nobody compares. Implemented at 008 T11 in
`media/ffmpeg.segment_command` and asserted on a delivered segment's real duration in
`tests/conformance/test_hls_segments.py`.

The divergence takes §3.0.3's fourth shape read from the safe side — a value clients already read
is *unchanged*, and what moves is the thing it was always meant to describe. Recorded in
[008 §3.7](../../specs/008-playback-negotiation-and-delivery/spec.md#37-video-delivery) rule 2,
which asked for it before it was known that the reference did not do it.

### 3.11 A stopped transcode leaves its `TranscodingInfo` on the session — class B, diverged

**Jellyfin does:** keep reporting a transcode that has ended. `TranscodingInfo` hangs off the
*device's* session and is overwritten on every progress tick `[source:
Emby.Server.Implementations/Session/SessionManager.cs:1866-1875 @ v10.11.11]`; the job's own exit
handler reports one last time with every number null, which leaves the object in place with two
fewer properties rather than removing it `[source:
MediaBrowser.MediaEncoding/Transcoding/TranscodeManager.cs:333-368, 640-644 @ v10.11.11]`. Only a
playback report saying the item is no longer being transcoded clears it. Measured: a session
whose job was killed by `DELETE /Videos/ActiveEncodings` — and one whose job died on the kill
timer — still carried a `TranscodingInfo` of eleven keys, `Framerate` and `CompletionPercentage`
gone and the codecs, the container, the size and the reasons still there `[probe:
tools/probe_transcode_session.py, Jellyfin 10.11.11, 2026-08-29]`. 008's own spec had said the
opposite until this was measured.

**Depends on it:** nobody who is not already tolerant of it, which is class B's first escape
hatch. There are two compensations a client can build on a report that outlives its job, and
neither breaks when the report goes away with the job. One is to believe `TranscodingInfo` only
while it carries a moving `CompletionPercentage` — and the reference drops that property at
exactly the moment the object goes stale, so such a client already reads the stale object as "not
transcoding", which is what Atrium's absence says. The other is to read the object's presence as
"this device is transcoding", which is a belief Atrium makes true and the reference makes false
for as long as playback goes unreported. The remaining consumer is a person looking at an
administration dashboard, and it shows them a job that has stopped.

**Atrium does: diverge — the report lives exactly as long as the work.** `/Sessions` carries
`TranscodingInfo` while the transcode manager owns a live session for that device and omits it
once the session is stopped, reaped or shut down. The manager *is* the set of live transcodes
here, and there is no second place to keep a copy that nothing would ever clear: reproducing the
reference's staleness would mean holding a per-device record whose only removal path is a
playback report belonging to another feature, and a record like that goes stale in the other
direction — a client that stops playing without calling the stop route would keep a
`TranscodingInfo` for as long as the process lives rather than for as long as the reference keeps
one.

**And the shape is one the reference sends.** Atrium's `TranscodingInfo` carries eleven of the
thirteen declared properties; the missing pair is `Framerate` and `CompletionPercentage`, which
come from parsing the encoder's progress output. Atrium runs its encoders at `-loglevel error`
and reads their diagnostics only to say why one failed, so it has no honest number to put there —
and inventing one from the segments produced so far would report the *last request's* position
rather than the encoder's. Both are nullable upstream and both are genuinely absent there twice:
before ffmpeg has reported anything, and after the job stops.

Implemented at 008 T12 in `media/sessions.TranscodingReport` and `api/sessions.TranscodingInfo`,
asserted over the wire in `tests/unit/test_transcode_lifecycle.py`. The divergence takes
§3.0.3's fourth and most dangerous shape — a property clients already read changes, from present
to absent — so it is carried by the evidence above rather than by the argument that it is more
correct: every property a client reads *while the work is real* is unchanged, and the two the
reference itself withdraws when the work ends are the two a compensation would key on.

### 3.12 A subtitle playlist's window durations are written in the server's locale — class B, diverged

**Jellyfin does:** format the `#EXTINF` duration of a subtitle window with the *server's* culture
rather than the invariant one. The playlist is assembled by appending a `double` straight to the
builder, with no format provider `[source:
Jellyfin.Api/Controllers/SubtitleController.cs:389-392 @ v10.11.11]`, so the decimal separator is
whichever one the host is configured for. Every full window is a whole number of seconds and hides
it; the last one is a remainder, and on a Spanish-configured server it comes out as
`#EXTINF:7,851,` where an HLS parser reads a duration of `7` and a title of `851,`
`[probe: tools/probe_subtitle_delivery.py, Jellyfin 10.11.11, 2026-08-29]`. The same reference
answers a different playlist on a differently configured host, for the same request.

**Depends on it:** nothing that could. A client cannot know the operator's locale, so it cannot
compensate — which is class B's second escape hatch, and the same reasoning §3.4 takes. The
window a wrong duration describes is the last one of the track, and the error is under a second on
a track that is already ending; no player builds on that.

**Atrium does: diverge — the duration is written with a decimal point, always.** This is not a
choice to be more correct at a client's expense: it is the only reproducible answer available.
Atrium has no server interface culture to reproduce the defect *from*, so replicating it would
mean inventing a locale setting in order to write a number wrongly. The invariant form is also
what the reference itself writes on an English-configured host, which is the majority case and
the one every published Jellyfin playlist example shows.

Recorded at 011's spec review on 2026-08-29 and **shipped at 011 T8 on 2026-08-30**, which is the
change that served the route. A window's duration is rendered from an exact tick count rather than
through a float, and a whole window is still written `30` and not `30.0` — so the divergence is
visible on the last window of a track and nowhere else. Two tests hold it: one sets a
comma-decimal locale and renders the reference's own measured runtime, whose last window reads
`7,851` there and `7.851` here, and one asserts the whole playlist a route answers over HTTP.
Neither depends on the host having that locale, which Principle VII forbids; setting one only
makes them harder to pass.

### 3.13 An un-inspectable source is advertised, and the address it is given answers `500` — class A, deferred

**Jellyfin does:** answer a negotiation for a video item whose file cannot be read with `200`, a
source carrying no streams, no runtime and no bitrate, the three capability flags **decided**
against the profile — `SupportsDirectPlay: false`, `SupportsDirectStream: false`,
`SupportsTranscoding: true` for a profile that plays neither the container nor the codec — and a
`TranscodingUrl`. Following that address gives a master playlist that answers `200` and names
**`live.m3u8`** rather than `main.m3u8`, because a source with no `RunTimeTicks` is addressed as an
infinite stream; and `live.m3u8` answers **`500`, `text/plain`**
`[probe: tools/probe_uninspected_source.py, Jellyfin 10.11.11, 2026-08-29]`.

So the guarantee the reference actually offers is *an advertised capability has an address*, not
*an advertised capability has an address that answers*. The two are a paragraph apart and only one
of them is true.

**Depends on it:** nothing that could build on it — the client is handed an address and meets a
`500` at the end of it, which is class A's shape: a fallback that stays unused, or an error the
user sees. The video client's documented behaviour on this path is to stop, which it does either
way.

**Atrium does: not decided here, and the deferral is the point.** v1 does not emit an address on
this path at all today, so the defect is in a code path this project does not yet have — which is
[§3.0.1](#301-the-tie-breaks) tie-break 3 exactly: *a decision made about code nobody will write
for a while is a decision made with the least information it will ever have.* The candidates are
two and they are both defensible: reproduce the `200`-with-an-address and let the delivery route
fail as the reference's does, or decide that a source nothing could read advertises no capability
at all. Choosing now would also be choosing before
[012](../../specs/012-negotiation-inputs/spec.md)'s delivery half exists to be measured against.
Recorded at 012's measurement gate on 2026-08-29, and owed to the task that implements 012 §3.2.

### 3.14 The fMP4 initialisation segment restarts production — class B, replicated

**Jellyfin does:** treat a request for segment `-1` as a reason to start transcoding, before it has
looked at what is already running — *"Starting transcoding because fmp4 init file is being
requested"*
`[source: Jellyfin.Api/Controllers/DynamicHlsController.cs:1501-1505 @ v10.11.11]`. Read on its own
that is a defect with a real cost, and the video client pre-warms its session to dodge it.

**Measured, the branch is third rather than first, and it is not reached by either order a client
uses.** Two file-existence checks stand in front of it, one of them inside the transcode lock, and
both return the segment without starting anything
`[source: Jellyfin.Api/Controllers/DynamicHlsController.cs:1481-1496 @ v10.11.11]`. An fMP4
transcode writes the initialisation segment *before* it writes any segment, so a session that has
produced anything already has the file the branch tests for. Asking for the map after three
segments answered in **0.03 s**, against **0.69 s** for the same request on a directory with
nothing in it, and the segments already produced still answered immediately afterwards —
nothing was discarded. A second request for an initialisation segment already on disk restarts
nothing at all `[probe: tools/probe_transcode_session.py, Jellyfin 10.11.11, 2026-08-29]`.

**Depends on it:** a client that pre-warms is unaffected either way, which is the compensation
being defect-tolerant. Nothing else can observe a restart that discards nothing.

**Atrium does: replicate — which here means write the branch and stop worrying about it.** Under
[§3.0](#30-how-the-decision-is-made) the defect is class B (a `200` produced by more work than
necessary), and its escape hatches do not open: the compensation is tolerant, and there is a
measured cost of zero to remove. [§3.0.0](#300-replication-is-not-free-and-for-this-project-it-is-not-the-lazy-option)
then decides it, because diverging would mean writing *extra* code — a fourth condition on a branch
that already behaves correctly — to avoid a cost nobody pays. The ordering is what makes this safe
and it is worth writing down: **the file-existence checks must come before the segment-id branch**,
because a server that tested the segment id first would restart a producing session on every map
request and the measurement above would invert.

Decided at 012's measurement gate on 2026-08-29 (012 OQ-8), under the procedure in §3.0.

### 3.15 `Move`'s index is unguarded in both directions — class A, diverged

**Jellyfin does:** answer a `Move` whose `newIndex` is past the entry count with **`500`** and the
bare text `Error processing request.`, and a `Move` whose `newIndex` is **negative** with `204` and
a move the caller did not ask for. On a five-entry playlist, index 5 puts the entry last; index 6
is the `500`; index -1 answers `204` and leaves `B A C D E`. An entry id that is not in the
playlist is a silent `204` when the index is in range, and the **same `500`** when it is not —
because the index arithmetic runs before the entry is looked up.
`[probe: tools/probe_playlist_move.py, Jellyfin 10.11.11, 2026-08-31]`
`[source: Emby.Server.Implementations/Playlists/PlaylistManager.cs:307-345 @ v10.11.11]`

**And the entry segment is never parsed**, which 009 T11 measured while implementing the route: an
entry id that is malformed, all zeros or written in the *dashed* spelling of a real entry's id is
matched against nothing, so all three take the same two answers as an absent one — `204` in range,
the `500` out of it. An **upper-case** one matches, because the comparison folds case and nothing
else `[probe: tools/probe_playlist_move.py, Jellyfin 10.11.11, 2026-09-01]`
`[source: Emby.Server.Implementations/Playlists/PlaylistManager.cs:308-323 @ v10.11.11]`.

**Depends on it:** nothing can be built on either. A `500` carries no information a client can act
on, and a negative index that silently reorders is a result no client asked for — a drag-and-drop
that computes -1 for a row dropped above the first one gets a move to position **1**, which reads
as a bug in the client. Class A by §3.9's reasoning, which is the same shape: an unhandled
exception on a malformed input, where the well-formed input one step away is a clean refusal.

**Atrium does: diverge — refuse both with `400`, and move nothing.** The clamp at exactly the entry
count is parity and is kept; past it, and below zero, the request is refused rather than crashed or
guessed. The absent-entry `204` is parity and is kept as well, and so is the order the reference
judges in — the index first, so an entry that is not in the playlist is refused rather than ignored
when the index is out of range too. The three spellings that match nothing are parity too, and they
are the reason this route does **not** canonicalise the identifier it is handed: normalising a
dashed entry id would reorder a playlist no reference server reorders, which the caller sees in the
order that comes back. The only observable difference is a `400` where the reference produces a
`500` or an unrequested move, and it is reachable only by a caller who may edit — the permission
test runs first, measured with an index the reference crashes on
`[probe: tools/probe_playlist_shares.py, Jellyfin 10.11.11, 2026-09-01]`. Specified in
[009 §3.5](../../specs/009-playlists/spec.md).

---

### 3.16 A playlist read names its own reader — class A, diverged

**Jellyfin does:** take the identity it checks a playlist's permissions against from the `userId`
query parameter of `GET /Playlists/{playlistId}/Items`, with no test that the caller may name it —
`var callingUserId = userId ?? User.GetUserId();`, and every permission test below it runs against
the named user `[source: Jellyfin.Api/Controllers/PlaylistsController.cs:520-531 @ v10.11.11]`.
Measured: a restricted non-administrator, answered `404` by `GET /Items/{id}` and by the same
playlist route without the parameter, is answered **`200` with the entries** when the request names
the owner. `[probe: tools/probe_playlist_visibility.py, Jellyfin 10.11.11, 2026-08-31]`

**The same controller refuses the same parameter one route away.** `CreatePlaylist` and
`AddItemToPlaylist` pass it through a helper that throws for a non-administrator naming another
user — measured as `403` in the same run `[source: Jellyfin.Api/Helpers/RequestHelpers.cs:67-85 @
v10.11.11]`.

**Depends on it:** an administrator reading another user's playlist, which the helper permits on
the write routes and which this route permits for everybody. §3.0.1's tie-break 1 — absent
evidence, assume a compensation exists — is what kept
[§3.5 `/Users/Public`](#35-userspublic-discloses-every-users-policy-to-anyone--class-b-replicated)
replicated, and it does not reach here: a client cannot have built a workflow on a permission the
same server refuses on the routes beside it, because the workflow would fail the moment it wrote
anything.

**Atrium does: diverge — honour `userId` for an administrator, refuse it otherwise**, which is the
reference's own rule on its own write routes rather than a rule of ours. The divergence is strictly
*less* disclosed, which §3.0.3 names as the safest shape, and it is confined to a request a
non-administrator has no legitimate reason to send. Specified in
[009 §3.7](../../specs/009-playlists/spec.md).

**Shipped at 009 T9, and it is one call rather than a rule of this route's own.** The read route
puts `userId` through `effective_user` — 005's helper, unchanged, the same one `POST /Playlists`
uses — so the refusal is the controller's 25-byte `text/plain` measured beside it and there is no
second copy of the rule to drift. Re-measured on the implementation date and unchanged: the
reference still answers `200` with the entries `[probe: tools/probe_playlist_visibility.py,
Jellyfin 10.11.11, 2026-09-01]`.

**The divergence has a second half, and it is parity.** An administrator naming a user is honoured
here, and what they get is **that user's view** rather than an administrator's: the visibility
clause has no administrator branch (009 plan §6.5), so an administrator who is none of §3.7's three
classes is answered `404` for a private playlist, `userId` or not.

---

### 3.17 A playlist's entries ignore the reader's library access — class B, diverged

**Jellyfin does:** filter a playlist's entries through a visibility call that is a parental-rating
and tag check, and nothing else `[source: MediaBrowser.Controller/Entities/BaseItem.cs:1736-1741 @
v10.11.11]`. Library access is enforced by the item *queries*, which this route does not use — so a
reader restricted to one library, who can list **zero** items of another and is answered `404`
fetching one directly, is handed those items as playlist rows, with their names and ids, counted in
`TotalRecordCount`. `[probe: tools/probe_playlist_visibility.py, Jellyfin 10.11.11, 2026-08-31]`

**Depends on it:** a client rendering a shared playlist sees rows it cannot play — every one of
them answers `404` on fetch and on delivery. Nothing is built on that; what a client would have to
have built on is the *count*, and a count that includes unplayable rows is not something a client
can use either.

**Atrium does: diverge — omit entries the reader cannot reach**, keeping the order and the entry
ids of the rest, and counting only what it returns. The difference a client can observe is fewer
rows and a smaller total for the same shared playlist, which is the direction §3.0.3 calls the
least dangerous, and it is the behaviour
[009 §3.7](../../specs/009-playlists/spec.md) had been describing as the reference's own since the
day it was written. Specified there, as a divergence now that the gate has measured it.

**`Move` follows the omission, and the reference has no answer here to reproduce.** The index a
reader gives is judged against the list they were given, and the entry lands at that index **of
it** — where the reference bounds the index against the entries it shows and then computes the
landing position in the stored order *before* removing the entry, so a downward move lands one
position short of where the caller asked, and an entry the caller was never shown is reordered
rather than left alone `[source: Emby.Server.Implementations/Playlists/PlaylistManager.cs:289-345 @
v10.11.11]`. Neither can fire against a reference server for the reason above: what it hides is
hidden by the parental-rating check, never by library access. Atrium answers an omitted entry the
way it answers one that is not in the playlist at all — `204`, nothing changes (§3.15's parity row).

### 3.18 A playlist's de-duplication misses about a third of the time — class B, diverged

**Jellyfin does:** de-duplicate an add in two stages — against the entries the playlist already
holds, then within the incoming batch — and only the second stage is reliable. The first compares
the incoming items against `playlist.LinkedChildren.Select(c => c.ItemId)`
`[source: Emby.Server.Implementations/Playlists/PlaylistManager.cs:221-224 @ v10.11.11]`, and that
field is **not** the entry's item: it is a cache, filled the first time the entry is resolved
`[source: MediaBrowser.Controller/Entities/BaseItem.cs:1773-1805 @ v10.11.11]`, and the writer that
creates entries never fills it
`[source: MediaBrowser.Controller/Entities/LinkedChild.cs:26-40 @ v10.11.11]`. An entry whose cache
is cold is invisible to the filter, so the item goes in a second time. Measured: **6 of 8 identical
requests** — same server, same playlist shape, seconds apart — added an item the playlist already
held, and the exploratory battery that found it duplicated on 5 of 16
`[probe: tools/probe_playlist_writes.py, Jellyfin 10.11.11, 2026-08-31]`.

The duplicate it produces is **unaddressable**, which is
[009 §3.1](../../specs/009-playlists/spec.md) seen from the other side: both rows carry one
`PlaylistItemId` — the item's own `Id` — so `Move` on that id reorders the first copy and `Remove`
on it deletes both at once. Measured, on a playlist the race had just duplicated.

**Depends on it:** nothing can. The behaviour is a coin flip on identical requests, so a client
cannot predict it, cannot ask for it, and cannot compensate for it — the only compensation
available is de-duplicating client-side, which is unaffected by a server that never duplicates.
Nor can a user clean up afterwards: the two operations that address an entry treat the two copies
as one. This is §3.0's second escape hatch — a class-B defect no client can work around behaves
like class A in the only respect that matters — and Principle VII settles what is left, because
replicating a race means a server whose answer to one request differs between two runs.

**Atrium does: diverge — de-duplicate always.** The entry table's key is the playlist and the item,
so a repeat is dropped on the creation path and on the addition path alike, the entry already
present keeps its position, and the first occurrence within a batch is the one that lands — all
three of which are the reference's own behaviour on the runs where its cache is warm. What a client
can observe is a playlist that never grows a row it cannot then delete. Specified in
[009 §3.1 and §3.4](../../specs/009-playlists/spec.md), AC-5.

---

### 3.19 The playlist write routes crash on requests they cannot serve — class A, diverged

**Jellyfin does:** answer `POST /Playlists` carrying **neither a `Name` in its body nor one in its
query** with **`500`**, `text/plain`, the fixed 25 bytes. The four properties may be sent either
way — the query wins, and `?name=` with no body at all creates a playlist — so "no name" is a
property of the merged pair rather than of either source, and it is the one combination the route
does not survive `[probe: tools/probe_playlist_creation.py, Jellyfin 10.11.11, 2026-08-31]`.

The same run measured a second unserveable request answered rather than refused: a body whose
`UserId` names **nobody at all** answers `200` and creates a playlist owned by a user that does not
exist. Nothing then reaches it — every rule in [009 §3.7](../../specs/009-playlists/spec.md) is a
comparison against an owner or a share.

**And a third, on a sibling route, found at 009 T10.** A **malformed** playlist identifier in the
path of `DELETE /Playlists/{playlistId}/Items` is a `500` in the same 25 bytes, where the **same
malformation on the same path** through `POST` is the model binder's validation `400`. The two
actions bind that segment differently — the add declares an identifier and the removal declares a
string it parses inside the method — so one refusal is the framework's and the other is an
unhandled parse `[probe: tools/probe_playlist_add_remove.py, Jellyfin 10.11.11, 2026-09-01]`
`[source: Jellyfin.Api/Controllers/PlaylistsController.cs:447-456 @ v10.11.11]`.

**And a fourth, on the third route of the same controller, found at 009 T11.** `POST
/Playlists/{playlistId}/Items/{itemId}/Move/{newIndex}` parses that segment inside the action as
the removal does, so a malformed playlist id is the same `500` in the same 25 bytes — the two
segments of one path bind differently from each other on this route as well, since the *entry* id
is never parsed at all and refuses nothing
`[probe: tools/probe_playlist_move.py, Jellyfin 10.11.11, 2026-09-01]`
`[source: Jellyfin.Api/Controllers/PlaylistsController.cs:409-431 @ v10.11.11]`.

**Depends on it:** nothing can be built on any of the four. A `500` carries no information a
client can act on, and a playlist owned by nobody is unreadable, uneditable and undeletable by
every caller — a client that made one could not tell its user what happened to it. Class A by
§3.9's reasoning, and the same shape as
[§3.15](#315-moves-index-is-unguarded-in-both-directions--class-a-diverged) one route away: an
unhandled failure on a malformed input where the well-formed input beside it is a clean answer.

**Atrium does: diverge — refuse all four, and create nothing.** A request naming no name anywhere
is `400` in the *same* `text/plain` shape the reference's `500` carries, so the status is the whole
difference; a `userId` naming nobody is the problem-details `404` that `effective_user` already
answers on every 005 route, which is where the rule lives rather than being written a second time
here; and a malformed playlist id is the validation `400` on **all three** write routes, because
each declares that segment as an identifier and one shape for one path is the smaller surprise. The query
form itself is **parity** and is implemented: refusing a request the reference serves would be the
larger divergence. Specified in [009 §3.2](../../specs/009-playlists/spec.md) and
[§3.5](../../specs/009-playlists/spec.md).

### 3.20 A deletion discloses a playlist the reader is refused — class B, replicated

**Jellyfin does:** answer `DELETE /Items/{itemId}` for a playlist the caller may neither read nor
delete with `401` and the body `"Unauthorized access"`, where every other route on that same
playlist answers `404` — the read route's twenty bytes, or `GET /Items/{itemId}`'s problem details.
The item lookup this route shares applies the collection-folder test that hides **media** and no
test at all that hides a playlist, so a caller holding an identifier learns from the difference
whether a playlist is there `[probe: tools/probe_item_deletion.py, Jellyfin 10.11.11, 2026-09-01]`
`[source: Jellyfin.Api/Controllers/LibraryController.cs:374-383 @ v10.11.11]`.

**Depends on it:** a client cannot have built anything on the disclosure itself, but it can have
built something on the `401`: a delete button that refuses a playlist somebody else owns must show
its user *refused*, and a `404` invites "it is already gone" instead. That is the compensation
§3.0.1 tie-break 1 asks about, and it is the reason this is class B rather than a free correction.

**Atrium does:** the same. The disclosure is bounded — it reveals nothing but the existence of an
identifier the caller must already hold, and identifiers here are minted rather than guessable
(Principle VII's determinism is over inputs, not over anything a stranger has) — while the
alternative changes a refusal a client can act on into one it cannot. §3.5's reasoning applies
almost word for word: "it discloses too much" is the temptation §3.0.2 forbids acting on by itself.
Specified in [009 §3.6](../../specs/009-playlists/spec.md).

### 3.21 The rename applies a partial body destructively — class A, diverged

**Jellyfin does:** treat `POST /Items/{itemId}` as an update of the whole item, and answer two
kinds of incomplete body in two ways, neither of them survivable
`[probe: tools/probe_playlist_rename.py, Jellyfin 10.11.11, 2026-09-01]`:

- **A body that omits `Genres`, `Tags` or `ProviderIds`, or sends one of the three as `null`**, is
  refused `400` with the controller's 25 bytes. Those three are the only required ones: dropping
  each of the thirty-nine properties a read of a playlist hands back, one at a time, refuses on
  exactly those three, and a body of just them and a `Name` is accepted. **And the refusal is not
  clean.** After a body with no `Genres`, the playlist's own `GET /Items/{itemId}` answers `400`
  from then on while the item still counts in a listing — the update applied part of the body
  before it threw. A later complete body repairs it.
- **A body carrying no `Name`, or a `Name` that is `null`**, is answered `204` and **erases the
  name**. The playlist comes back with no `Name` property at all, from a route whose only purpose
  in this feature is to set one.

**Depends on it:** nothing. The first is a refusal, so no client can have built on the success it
never got; the second destroys the one property the request exists to write, which no client can
have wanted. Class A by §3.9's reasoning, and the same shape as
[§3.19](#319-the-playlist-write-routes-crash-on-requests-they-cannot-serve--class-a-diverged) one
method away: an unserveable request answered destructively, where the well-formed request beside it
is a clean `204`.

**Atrium does: the first is parity, the second is refused.** A body missing any of the three is the
same `400` in the same 25 bytes — the client's round trip is load-bearing on both servers, and a
client posting `{"Name": …}` alone is refused by a stock reference too — and the playlist stays
readable afterwards, because nothing is written until the whole body has been checked. A body with
no `Name` is that same `400` rather than a `204` that erases the name: the bytes are identical and
the status is the whole of the difference, which is the argument §3.19 makes for the four requests
beside it. Specified in [009 §3.8](../../specs/009-playlists/spec.md).

### 3.22 Any authenticated caller reads any user whole — class B, replicated

**Jellyfin does:** answer `GET /Users/{userId}` with the **whole user object** — `Configuration`
and `Policy` included — to any caller holding a usable token, whoever they are and whoever they
name. Measured across the route's whole matrix rather than one pair of it: an ordinary
non-administrator naming another non-administrator, a *restricted* non-administrator naming an
**administrator**, an administrator naming anybody, and a user naming themselves are one `200` —
and the administrator's object as read by that restricted stranger is **byte-identical** to the
administrator's own reading of it, so there is no per-caller redaction anywhere in the route
`[probe: tools/probe_user_read.py, Jellyfin 10.11.11, 2026-09-01]`. 009 T2 measured the
administrator cell alone on 2026-08-31, which is where the question came from
`[probe: tools/probe_playlist_visibility.py, Jellyfin 10.11.11, 2026-08-31]`.

**This is [§3.5](#35-userspublic-discloses-every-users-policy-to-anyone--class-b-replicated)
reached by a second road**, and the same 42 policy properties travel down it. The two differ in
who can walk them: `/Users/Public` needs no token and lists the users itself, this route needs a
token and needs the caller to name an identifier. That is a smaller audience for the same
disclosure, not a different one.

**Depends on it:** unknown, and the shape of the dependency is a client that reads another user at
all — the music client is this route's named consumer. What a client cannot have built on is the
refusal, because no reference server sends it. That asymmetry is the whole decision: replicating
risks a disclosure clients may not need, and diverging **breaks a request that succeeds against
every Jellyfin there is**.

**Atrium does:** the same. It answered `403` to a non-administrator naming anybody else until
2026-09-01, and that refusal was never the reference's — [002 §3.7](../../specs/002-authentication-users-and-sessions/spec.md)
had stated it with no provenance since the specification was written, which is exactly the failure
mode Principle II names.

The class is **B**: the request succeeds, with more than it should. §3.0's question is whether a
client can have built something that being correct would break, and here it plainly can — a `403`
where the reference sends a body is not "less information", it is a failed request, and §3.0.3
puts *refusing what the reference answers* at the dangerous end of its list. The disclosure
argument that §3.5 weighs and rejects is the same argument here, one road over, and taking it here
alone would leave Atrium refusing on one road while publishing the identical object on the other:
the inconsistency, not the protection. So the default holds — Principle V, replicate — and if the
divergence is ever taken it is taken on **both** roads, in one change, with §3.5's entry rewritten
beside this one.

**Two refusals came back with the measurement, and they are the reference's own.** An identifier
that is well formed and belongs to nobody is `404` with the JSON-encoded bare string
`"User not found"` — the fourth error shape of §1.11, a third route for it — and the *same* body
to an administrator and to a non-administrator, so the reference does not conceal which
identifiers exist either. An identifier that is not an identifier is the validation `400`, keyed
on `userId` and quoting the value back. Atrium had answered both with the same `403` as everything
else, deliberately, so that a caller who may not look could not tell them apart; there is nobody
who may not look. Specified in
[002 §3.7](../../specs/002-authentication-users-and-sessions/spec.md).

### 3.23 `Similar` is a random draw, not a ranking — class B, diverged

**Jellyfin does:** answer `GET /Items/{itemId}/Similar` with items that share the seed's own
genres and tags, **in no order at all**. There is no scoring pass anywhere in the route: it builds
a query from the seed's genre and tag names and asks for the result ordered at random
`[source: Jellyfin.Api/Controllers/LibraryController.cs:790-801 @ v10.11.11]`. Measured, four
identical requests against one seed returned **48 distinct items with none in common** — not one
result set reshuffled, but a fresh draw over the matching pool each time
`[probe: tools/probe_similar_ranking.py, Jellyfin 10.11.11, 2026-09-01]`.

So the array is not a ranking whose leading rows mean more than its trailing ones. The word
*similar* is carried entirely by the filter, and everything after the filter is chance.

**Depends on it:** no, and this one cannot be depended on. There is no order to compensate for: a
response that never repeats itself offers a client nothing to assume, so the only two things a
client can do with the array are show it as it arrives and re-sort it itself — and both are
untouched by the order becoming stable. That is §3.0's first escape hatch, the same one
[§3.1](#31-totalrecordcount-is-0-on-by-name-endpoints-without-limit--class-b) and
[§3.6](#36-ties-are-engine-resolved-and-paging-the-artist-sorts-loses-rows--class-b-diverged)
went through. What a client *can* observe is that a second request returns the same rows where the
reference returns different ones; that is a difference in variety rather than in correctness, and
the route's one named consumer — the video client — states nothing about either in its traced
contract.

**Atrium does: diverge — score deterministically**, so one seed over one library produces the same
rows in the same order every time. v1 scores on shared genres, people and studios
([005 §3.7 and AC-12](../../specs/005-item-query-api/spec.md)).

The argument required by Principle V is **not** that the difference is invisible — a client issuing
the request twice can see it. It is that nothing can be built on the difference: a random draw is
not a value, and a client treating a stable answer as one the reference could have given it by
chance is right, because it is. Every response Atrium sends here is a response the reference could
have drawn.

The second half of the argument belongs to this repository rather than to any client.
**Replicating the randomness would make this endpoint undiffable by the very harness that exists to
check it.** A route whose successive answers share no rows cannot be compared field by field
against another server, or against itself — so reproducing the draw faithfully would buy nothing a
client can use and cost the only mechanism able to find what this project has got wrong here. That
is [010 OQ-4](../../specs/010-conformance-harness/spec.md#7-open-questions), which had to answer
this endpoint with a whole array excused rather than a field, and
[010 §3.3](../../specs/010-conformance-harness/spec.md#33-the-allowlist) carries the mechanism.
§3.0.0 applies with unusual force besides: replicating means writing a shuffle on purpose, and then
a test asserting that it shuffles.

Class **B** — the request succeeds, carrying something wrong in the only sense that matters here —
decided through the escape hatch rather than against it. No upstream issue is known, so §3.0.1's
tie-break 2 reads *not judged* and weighs nothing.

`limit` on the same route is a **separate** decision with its own class and its own argument
(§3.24), because §3.0.2 forbids deciding once for a whole endpoint.

### 3.24 `Similar` answers `limit + 4` on a movie seed — class B, diverged

**Jellyfin does:** treat `limit` on `GET /Items/{itemId}/Similar` as a maximum for three seed types
and not for the fourth. Measured at `limit=1`, `limit=5` and `limit=20`, on two seeds each: a
**movie** seed answers **exactly N + 4** rows every time, where a series, an album and an artist
seed answer exactly N `[probe: tools/probe_similar_ranking.py, Jellyfin 10.11.11, 2026-09-01]`.

The four are not a rounding or a tie: a limited query that groups by metadata key has four added to
its limit before it runs `[source: Jellyfin.Server.Implementations/Item/BaseItemRepository.cs:1427-1429
@ v10.11.11]`, and this route sets that grouping flag for the movie case alone
`[source: Jellyfin.Api/Controllers/LibraryController.cs:795 @ v10.11.11]`. Nothing removes the
surplus afterwards, so it reaches the wire.

`TotalRecordCount` travels with it, and the two servers fill the field from different things. The
reference's is **the number of rows it just returned** — N + 4 on the movie seed, N elsewhere —
where Atrium's is the size of the matching pool before the limit, which is what
[005 AC-5](../../specs/005-item-query-api/spec.md) requires of every list endpoint in the project
and what [§3.1](#31-totalrecordcount-is-0-on-by-name-endpoints-without-limit--class-b) already
decided for the by-name family.

**Depends on it:** no compensation is possible that a correct answer would break, and the reference
itself is the evidence. **The difference is observable** — a client that counts what it asked for
sees five rows where it asked for one — so this entry does not get the argument that nobody can see
it. What it gets instead is that there is nothing consistent to have compensated for:

- The surplus is **not stable across the route**. Three of the four seed types honour `limit`
  exactly, so a client that hard-coded "ask for four fewer" would under-fill every series, album and
  artist shelf it drew. A compensation that is wrong on three quarters of the same endpoint is not a
  compensation anyone shipped.
- The surplus is **not explicable from the wire**. Nothing in the response says why the extra rows
  are there or which four they are; they are indistinguishable from the ones asked for.
- The remaining compensations are **defect-tolerant**: a client that displays what arrives keeps
  working when N arrives, and a client that truncates to N locally finds the truncation a no-op.
  §3.0's first escape hatch, again.

Taken through §3.0 explicitly, the class is **B** and not A or C: the request succeeds — no 5xx, no
unparseable body — and it succeeds with *more* than was asked for rather than with something
omitted. So the default is *replicate*, and the escape hatch above is what moves it. §3.0.3 is
honest about the risk that carries: changing a value clients already read is shape 4, the dangerous
one, which needs evidence about compensations rather than reasoning about correctness — and the
evidence is the three seed types the reference already answers correctly, on the same route, in the
same version. A client cannot have built on a rule the reference does not follow.

§3.0.0 is the last weight: replicating means writing code that adds four arbitrary rows to a movie
seed's answer and only a movie seed's, plus a test asserting the four are there. Where the evidence
is balanced the side that has to write bug code needs the stronger argument, and here it does not
have one. No upstream issue is known — tie-break 2 reads *not judged*.

**Atrium does: diverge — answer exactly `limit` rows on every seed type**, and report
`TotalRecordCount` as the pre-limit pool size on all of them, as
[005 AC-5](../../specs/005-item-query-api/spec.md) already required. Replicating the count instead
would mean sending a `TotalRecordCount` equal to the page — a total that is not a total — on one
route while every other list endpoint in the project sends the real one, which is §3.1's decision
reversed for no reader's benefit.

### 3.25 `ChildCount` on a library view is a fresh random integer — class B, diverged

**Jellyfin does:** answer `GET /UserViews` with a `ChildCount` between 1 and 9 that is **redrawn on
every request**, on every view, with the row order and every other property of the response
unchanged `[probe: tools/probe_reference_determinism.py, Jellyfin 10.11.11, 2026-09-01]`. It is not
a stale count and not a cache: the reference declines to compute a top-level folder's count at all
and substitutes a number in its place, so that a client does not read the library as empty
`[source: Emby.Server.Implementations/Dto/DtoService.cs:516-526 @ v10.11.11]`. `/UserViews` reaches
it because that route asks for every field
`[source: Jellyfin.Api/Controllers/UserViewsController.cs:89 @ v10.11.11]`.

This is the third of the three non-deterministic answers 010's gate found, and the only one that is
a single field: the other two are whole arrays (§3.23, and the caller's own random ordering).

**Depends on it:** no, and nothing can. The value is a different number each time it is asked, so
there is no value to compensate for — a client that reads it either shows a number that changes
under it or ignores it, and both are untouched by the number becoming true and stable. §3.0's first
escape hatch, the same one §3.1, §3.6 and §3.23 went through. What a client can observe is that two
requests agree here and disagree there, which is a difference in variety and not in correctness. The
one thing the reference's number is for — *not looking empty* — is served better by a count that is
right, and an empty library is the one case where the reference's substitute is a lie a client can
act on: it offers navigation into a view with nothing in it.

**Atrium does: diverge — emit the real count.** A `/UserViews` row carries the same computed subtree
aggregate every other container carries: `ChildCount` is one of the gated fields
[005 §3.2](../../specs/005-item-query-api/spec.md) answers with a count of the container's children,
and this route asks for it unasked as part of its measured width. No code
exists here to make it random and none will be written: §3.0.0's cost lands with unusual weight when
replicating means writing a random number generator into a count and then a test asserting it varies.

Class **B** — the request succeeds and carries something wrong — decided through the escape hatch
rather than against it. No upstream issue is known, so §3.0.1's tie-break 2 reads *not judged*.

**This entry is also why the conformance allowlist is scoped by endpoint and JSON path**, and it is
worth stating here because the flat version is the one a later reader will reach for. The difference
that needs excusing is *the reference's number on a `/UserViews` row*. An entry that excused the
**name** `ChildCount` would excuse Atrium's real aggregate everywhere it appears — on the series,
the season, the two-disc album — which is precisely the value L2 exists to check
([010 §3.3](../../specs/010-conformance-harness/spec.md#33-the-allowlist), and its plan's §6.3).

---

## 4. Deliberate exceptions

Every one of them is listed here so it is never mistaken for an oversight — including §4.4, which
stays after its 2026-08-28 withdrawal because the record of an exception outlives the exception.

### 4.1 Atrium identifies as Jellyfin on the fields clients parse

`ProductName: "Jellyfin Server"` and a real `10.11.x` version string. Full reasoning in
[reference-target.md §4](reference-target.md#4-server-identity-what-atrium-tells-clients-it-is).
Humans see "Atrium" in the `Server` header, the `ServerName` field, the logs and the project page.

**The `Server` header is a measured divergence, not a hypothetical one.** The reference sends
`Server: Kestrel`. `[probe: tools/probe_routing.py, Jellyfin 10.11.11, 2026-08-28]` Atrium sends `Server: Atrium/<version>`.

A client cannot usefully branch on it — `Kestrel` identifies a .NET web server, not Jellyfin, and
the discriminator multi-server clients actually read is `ProductName`. So this is the one header
where the honest answer costs nothing, and it is where a person looking at a `curl` dump, a proxy
log or a bug report finds out what they are really talking to.

### 4.2 `LocalAddress` does not get an HTTPS override

See §2.3. Jellyfin's behaviour here is not a contract clients rely on; it is a source of breakage
they work around. Atrium reports the scheme it is actually reachable on.

The argument that no client can observe the difference: the override fires only **when a
certificate is configured** (§2.3's measured condition — reproduced on 2026-09-02, where the
override needed a certificate that loaded *and* a restart before it fired), and v1 terminates no
TLS and has no certificate configuration — the state in which the reference rewrites the scheme
cannot be configured on Atrium at all. A v1 deployment is reachable over HTTPS only through something else's
TLS, and a reference server in that same position holds no certificate of its own either, so its
override does not fire and the two answers coincide. A client could only see the divergence on a
configuration v1 does not have.

### 4.3 `DELETE /Items/{itemId}` refuses to delete media

**Jellyfin does:** deletes the item and its files, gated by the user's `EnableContentDeletion`
permission — or, failing that, the per-folder
`EnableContentDeletionFromFolders` list
`[source: MediaBrowser.Controller/Entities/BaseItem.cs:829-844 @ v10.11.11]`.

**Depends on it:** a client's delete button. This divergence **is** observable — a user deletes a
film and finds it still there.

**Atrium does:** permits deletion only for a **playlist**. Everything else this route can name
answers `403` — movies, episodes and tracks, and the by-name rows a deletion would take no file
from either. A genre or an artist is rebuilt by the next scan, so deleting one would be a deletion
that does not stick, which is Principle VI's plausible-looking stub; the reference refuses those
too, because its own precondition is that the item have a file at all
`[source: MediaBrowser.Controller/Entities/BaseItem.cs:820-828 @ v10.11.11]`.

**`403` here is an invented status, deliberately, and the parity path beside it is `401`.** The
reference has no refusal to copy for the case this divergence exists for — for an entitled user it
deletes the file — so the status is ours to choose, and `403` is what an authenticated caller
forbidden by policy is owed. The refusal the reference *does* make, for a caller who may not delete
that **playlist**, is `401` with the body `"Unauthorized access"`
`[probe: tools/probe_playlist_visibility.py, Jellyfin 10.11.11, 2026-08-31]`, and Atrium reproduces
that one exactly. Two refusals, two statuses, and only one of them is a divergence.

**Where the divergence is observable is narrower than it looks, and measured at 009 T12.** A caller
who is *not* entitled to delete content is refused by the reference as well — `401`, the same 21
bytes — so for that caller Atrium's `403` differs from a refusal rather than from a deletion
`[probe: tools/probe_item_deletion.py, Jellyfin 10.11.11, 2026-09-01]`. One status for every media
request is deliberate rather than an oversight: v1 enforces `EnableContentDeletion` nowhere, and
splitting a refusal's shape by a permission nothing else here honours would put a policy on the
wire that this server does not otherwise have.

Unlike the other divergences in this document, this one is not argued from "no client can tell". It
is argued from consequence. v1 has no trash, no undo and no confirmation flow of its own, so
honouring the route means trusting a client's dialog with an irreversible operation on files the
user may not have backed up. The cost of diverging is a delete button that fails on media. The cost
of not diverging is a bug in a new server destroying somebody's library. Revisited when there is a
trash with a retention window to delete into. Specified in
[009 §3.6](../../specs/009-playlists/spec.md).

### 4.4 Non-ASCII characters are sent as themselves, not as `\uXXXX` — **withdrawn 2026-08-28**

> **This exception no longer exists.** It was taken at 004 T15 and **reversed at 005 T4**, which
> implemented the escaping in `compat/responses.py` and recorded it as §1.16. The reversal never
> came back here, so for three features this section said Atrium sends the character while the
> code sent `\u00E7` — and 006's task list cited *this* section as the standing rule for an item
> name in an error body, which is how it was noticed (006 T3).
>
> The two paragraphs below are kept as the record of the decision and of what changed it: the
> objection was that upper-casing an escape's hex *after the fact* cannot be done safely, because
> a string legitimately containing `\u00e7` is indistinguishable from an escape. §1.16 answers it
> — the rewrite counts **backslash parity** rather than searching for `\u`, and `json.dumps` has
> already doubled every literal backslash by then, which makes the distinction exact. An objection
> answered by a mechanism is not a standing exception.
>
> Measured again on 2026-08-28, on the one body that carries an *item's own name*: an item called
> `DW Español` comes back from the image route as `"DW Espa\u00F1ol does not have an image of
> type Box"` `[probe: manual requests via tools/_probe.py, Jellyfin 10.11.11, 2026-08-28]`.
> Atrium's is byte-identical.


**Jellyfin does:** escape every non-ASCII character in a JSON body. `Occitan (post 1500);
Provençal` goes out as `Occitan (post 1500); Proven\u00E7al`, with the hex in **upper case** — the
default of .NET's JSON encoder, applied to every response rather than to this endpoint
`[probe: tools/generate_cultures.py, Jellyfin 10.11.11, 2026-08-27]`. Found by byte-comparing
Atrium's `GET /Localization/Cultures` against the reference's, which is the first response in the
project to contain a non-ASCII character at all.

**Depends on it:** nothing can. `"ç"` and `"\u00E7"` are **the same JSON string** — every
conformant parser produces the identical value, and the difference survives only in bytes nobody
reads. It is visible in `Content-Length`, which no client checks against an expectation.

**Atrium does:** send the character. Matching would mean re-encoding every response body with
`ensure_ascii` and then upper-casing the hex of each escape, and that second step is the problem:
a string that legitimately contains a backslash followed by `u00e7` is indistinguishable from an
escape after the fact, so the substitution is **unsafe in general**. Trading a real corruption
risk on unusual data for a byte difference no client can observe is the wrong trade, and it is the
shape §3.0.3 describes: the divergence is invisible through any parser, and reproducing it costs
correctness elsewhere.

Worth revisiting only if the differential harness (010) finds a client that reads raw bytes — and
if one exists, the fix belongs in `compat/responses.py` for every endpoint at once, not in the
feature that happened to notice.

### 4.5 `/System/Info` answers four fields with what is true here, not with the reference's constants

**Added 2026-09-03, at 001's closing audit.** Four members of `/System/Info` differ from the
reference in **value**, and they are here because they had nowhere else to be: an
[`allowlist.yaml`](allowlist.yaml) entry must cite a `behaviours §N` or one of the four derivation
classes, these have no derivation class, and the specification that decides them
([001 §3.2](../../specs/001-server-identity-and-discovery/spec.md)) is not a section of this
document. So the file's own load would have refused an entry and
[010](../../specs/010-conformance-harness/spec.md) would have met four undeclared differences on
the second response of the API. **This section is the `§N` those entries cite.**

They fall into two pairs with two different arguments, and both are Principle V's *shape of a safe
divergence* (§3.0.3) rather than Principle I: the **fields are all present, spelled and typed
exactly as the reference declares them** — Principle I is a statement about the surface, and the
surface is unchanged.

**The first pair: the reference's value is true of nothing.**

| Field | Reference | Atrium |
|---|---|---|
| `SystemArchitecture` | `"X64"` | `""` |
| `EncoderLocation` | `"System"` | `""` |

Both are **defaults under an obsolete marker on a property the server never assigns** —
`SystemArchitecture` is marked *"no longer set"*
`[source: MediaBrowser.Model/System/SystemInfo.cs:141-143 @ v10.11.11]` and `EncoderLocation`
*"isn't set correctly anymore"* `[source: MediaBrowser.Model/System/SystemInfo.cs:137-139 @
v10.11.11]`. So the reference's own declaration says the value means nothing, and a reference server
answers `X64` on an ARM machine.

Copying the constant would put a false claim about this host on the wire; reporting this host's real
architecture would put a string there that **no reference server sends**, on a field whose
declaration says it is not set. 001 §3.2's rule — real values where meaningful, empty string
otherwise — picks the third answer, and it is the same answer the reference gives for
`OperatingSystem` (§1.1's sibling case, and the one 001 §3.1 already fixes at `""`).

**The second pair: Atrium honestly cannot.**

| Field | Reference | Atrium | Why |
|---|---|---|---|
| `SupportsLibraryMonitor` | `true`, unconditionally `[source: Emby.Server.Implementations/SystemManager.cs:79 @ v10.11.11]` | `false` | v1 watches no filesystem |
| `CanSelfRestart` | `true`, marked obsolete saying so `[source: MediaBrowser.Model/System/SystemInfo.cs:67-69 @ v10.11.11]` | `false` | v1 cannot restart itself |

Answering `true` would be the failure §3.0.3 warns about in its other direction: a client that
started a watch-dependent flow on the strength of a `true` waits for a notification that never
arrives, and one that offers a restart button gets a route that is not in v1 at all (Principle VI).
A capability flag is the one kind of field where the honest answer is also the compatible one,
because its whole purpose is to let a client not try.

**Depends on it:** neither analysed client reads any of the four; 001's OQ-1 is the open question
about the second pair and it blocks nothing. **What this section costs:** four rows of
`allowlist.yaml`, scoped to `GET /System/Info` and one JSON Pointer each, citing `behaviours §4.5`.
Whichever way 010's run resolves them, the change is to this section or to 001 §3.2 — not to the
handler.

---

## 5. Accepted gaps in v1

Deltas that are **not** deliberate choices about what is right, but bounded shortfalls of a first
version. Each is listed with the mechanism that closes it, because a gap without one is just an
undocumented bug.

| Gap | What a client sees | Closing mechanism |
|---|---|---|
| **Tier 3 query parameters ignored** ([005 §3.3](../../specs/005-item-query-api/spec.md)) | A filter that does not narrow — more items than asked for | The ignored-parameter report ([010 §3.6](../../specs/010-conformance-harness/spec.md)); anything real clients send gets promoted |
| **Item fields outside the observed union omitted** ([005 §3.2](../../specs/005-item-query-api/spec.md)) | A field absent that the reference sends | The differential's key-set pass, which exists mainly for this |
| **Policy flags stored but unenforced** ([002 §3.5](../../specs/002-authentication-users-and-sessions/spec.md)) | A restriction that does not restrict | All of them gate features v1 lacks; each is enforced in the change that adds its feature |
| **Image decoration parameters ignored** ([006 §3.2](../../specs/006-images/spec.md)) | `percentPlayed`, `blur`, `foregroundLayer` have no effect | Implement if the differential shows a client sending them |
| **No subtitle burn-in** ([011 §2](../../specs/011-subtitle-delivery/spec.md#2-scope)) | A track whose negotiated delivery method is `Encode` is announced as burned in and is not: the reference paints those cues into the frames it produces and Atrium produces the same frames without them. It is not a rare branch — `Encode` is the reference's per-stream answer for **every** track no declared subtitle profile fits, which is every image track under a text profile and every track for a profile that declares nothing `[probe: tools/probe_subtitle_negotiation.py, Jellyfin 10.11.11, 2026-08-29]` — and naming such a track costs the source its direct play on both servers, so the client is pushed onto the transcode that then carries nothing. A client whose profile names a subtitle format it can take never reaches it | A text-rendering stack — fonts, ASS positioning, shaping — and a second filter path, which is the exclusion row the [roadmap](../roadmap.md#out-of-scope-and-why) has carried since before 001. *This row is what was left of "no subtitle delivery at all" when [011](../../specs/011-subtitle-delivery/spec.md) closed the rest of it on 2026-08-31: the manifest announces every text track, the sidecars are found and served, and `Encode` is the one answer this server says and does not do. 011 T12 narrowed the row rather than deleting it, because 011 §2 and §7's OQ-5 both call this gap "already recorded" and deleting it would have made two accepted documents false* |
| **A media source with no stored inspection is skipped** ([008 §3.1](../../specs/008-playback-negotiation-and-delivery/spec.md#31-media-sources)) | On a **listing**, nothing: the source keeps `Id`, a `Container` inferred from its path and `Size` and carries `RunTimeTicks: null`, `Bitrate: null` and `MediaStreams: []` — and so does the reference's `[probe: tools/probe_uninspected_source.py, Jellyfin 10.11.11, 2026-08-29]`. On `PlaybackInfo` the whole annotation is skipped, so it answers the model's default `SupportsDirectPlay: true` **with no `TranscodingUrl`**, where the reference opens the file and answers it annotated — or, when the file cannot be read, answers the same empty source with the flags *decided* and an address. It happens whenever a file is in the library and nothing has opened it: a scan from before 008, a file added since, a probe that failed | Not a rescan, and not a decision about what to advertise — **the negotiation itself**, which is what the reference does and what [012 §3.2](../../specs/012-negotiation-inputs/spec.md) specifies: open the file inside the request and write down what it says. *This row read "the real mechanism is a decision about what an un-inspected source should advertise" until 012's measurement gate measured the reference resolving the state rather than describing it. It also read as though the listing were part of the shortfall; it is parity, and the music client's four losses with it* |
| **No per-user subtitle preference, so no default subtitle track is proposed** ([011 §2, §3.3](../../specs/011-subtitle-delivery/spec.md)) | A negotiation that names no subtitle index answers `DefaultSubtitleStreamIndex` absent, where a stock reference proposes a track. It is the reference's own answer for a user whose subtitle mode is `None` — but a *new* reference user's mode is `Default`, not `None` `[probe: tools/probe_subtitle_negotiation.py, Jellyfin 10.11.11, 2026-08-29]` | The two user settings the choice is a function of: a subtitle mode with five values and a language preference list. Both are a per-user feature, which is what 011 §2 excludes; a client that names the track it wants is unaffected, and both analysed clients name it |
| **`Path`-derived identifiers differ from the reference's** ([§1.4](#14-item-identifiers-are-32-lowercase-hex-characters)) | Nothing — ids are opaque | Not a gap to close; a deliberate design choice |
| **A container that has lost every file is still returned** ([003 §3.8](../../specs/003-library-configuration-and-scanning/spec.md#38-scanning-and-change-detection)) | An empty series or album in a library, with nothing under it — **and a stock reference returns one too**, measured on 2026-09-02 `[probe: tools/differential.py --named container-that-lost-every-file, Jellyfin 10.11.11, 2026-09-02]`, which is what this row had assumed the other way round since it was written | A query-time filter in 005: a container with no visible children is not offered. See §5.2 — which is now a **measured parity** rather than a gap, and whether the filter is still worth its predicate is 005's call |
| **No loudness scan** ([004 §3.3](../../specs/004-metadata-resolution/spec.md#33-embedded-tags)) | On a server whose operator enabled the reference's opt-in scan, `NormalizationGain` absent where it would have a computed value. Tag-carried gains are unaffected | 008, which brings the decoder the scan needs. See §5.4 |
| **A stream carries no `DisplayTitle` and no `Localized*` names** ([008 §3.1](../../specs/008-playback-negotiation-and-delivery/spec.md#31-media-sources)) | A track picker with nothing to label its rows: the reference sends one localised string per stream — `Español - MP3 - Stereo - Predeterminado` on a Spanish server — and Atrium sends none. **[011 §3.4](../../specs/011-subtitle-delivery/spec.md) is where this stops being only a label**: a manifest entry's `NAME` is required, and the reference fills it from exactly this string | The localisation the strings are assembled from — which is two sources, not one: the flag words come from the server's own translation table and the language name from the platform's culture data, in the server's configured interface culture `[probe: tools/probe_subtitle_manifest.py, Jellyfin 10.11.11, 2026-08-29]`. An English-only approximation would differ from the reference on **every** track rather than be absent on it, which is the worse of the two. **011 T10 makes that asymmetric on purpose, and this row is where it is recorded**: the property stays absent — a JSON property can be — while a manifest's `NAME`, which cannot be, is written in the invariant form. The four flag words are the reference's English ones exactly; the fifth localised string, the marker for a stream with no language, is `Undefined`, the translation table's row, and **not** the `Und` the assembly falls back to when that string is empty, which no served stream ever is — all five arrive filled on 910 of 910 subtitle streams `[probe: tools/probe_stream_display_title.py, Jellyfin 10.11.11, 2026-08-30]`. The language name is the one piece that stays divergent, and closing this row makes `NAME` exact in the same change |
| **A subtitle file's encoding is decided by a rule, not by a detector** ([011 §3.5, §3.6](../../specs/011-subtitle-delivery/spec.md)) | On a file that is UTF-8 or carries a byte order mark — the overwhelming majority — nothing. On a subtitle file in a legacy single-byte encoding outside `cp1252`, **different cue text**: the words a player draws are not the words in the file, and where a byte is one `cp1252` does not define, the fetch answers the `500` a failed extraction does | A detector behind the one function that makes the choice, with its runtime dependency argued in an ADR on the day a real library needs it. See §5.11. **Measured on the wire on 2026-09-02**: a `cp1251` sidecar served as WebVTT comes back `Çäðàâñòâóéòå` here and `Здравствуйте` there — same status, same route, different letters `[probe: tools/differential.py --named legacy-encoded-subtitle-file, Jellyfin 10.11.11, 2026-09-02]` |
| **A stream carries no `IsAVC`, `TimeBase` or `NalLengthSize`** ([008 §3.1](../../specs/008-playback-negotiation-and-delivery/spec.md#31-media-sources)) | Three properties absent on every stream | Columns migration 0006 does not have; they arrive with the migration that adds them, and nothing in v1 reads them |
| **Dolby Vision and HDR10+ are not classified** ([008 §3.1, §3.7](../../specs/008-playback-negotiation-and-delivery/spec.md#31-media-sources)) | A Dolby Vision or HDR10+ film's master playlist carries no `SUPPLEMENTAL-CODECS`, and its copied segments are tagged `hvc1` where the reference tags `dvh1` — because the file is inspected here as the HDR10 file its colour metadata claims `[probe: tools/probe_transcode_decision.py, Jellyfin 10.11.11, 2026-08-29]` | Reading the Dolby Vision configuration record and the HDR10+ marker out of a stream's side data, which is where that signal lives. See §5.10 |
| **A playlist item carries no `Path`** ([009 §4](../../specs/009-playlists/spec.md)) | A `Playlist` item fetched with `fields=Path` has none, and its two date fields are its store's rather than a directory's. The reference builds a playlist as a directory under its data path and reports it `[probe: tools/probe_playlist_creation.py, Jellyfin 10.11.11, 2026-08-31]` | Nothing to close: v1's playlists are not files, and reporting a path no file backs would be the worse answer. Visible only to a client that asks for `Path` by name, and neither analysed client does |
| **A non-administrator cannot rename their own playlist** ([009 §3.8](../../specs/009-playlists/spec.md)) | The music client's rename button answers `403` for every user who is not an administrator — which is what a stock reference server answers too, because the route that client calls is declared elevated `[probe: tools/probe_playlist_rename.py, Jellyfin 10.11.11, 2026-08-31]`. This is a reproduced gap, not an introduced one: the reference's own working rename for an owner is `POST /Playlists/{playlistId}`, which no analysed client calls | `UpdatePlaylist` entering the surface, on the day a client is measured calling it. Principle VI keeps it out until then, and 009 §2 records what the exclusion costs |
| **The rename applies `Name` and nothing else, and refuses every item that is not a playlist** ([009 §3.8](../../specs/009-playlists/spec.md)) | A client that edits a playlist's overview, rating, year, genres or tags through `POST /Items/{itemId}` finds them unchanged, and the same request against a film, an episode, a track or a by-name row is `403` where the reference applies the body. Measured: a whole posted body changes `Overview`, `ForcedSortName`, `OfficialRating`, `CustomRating`, `ProductionYear`, `Genres` and `Tags` on the reference, while `Path` and `IsFolder` are computed and ignored on both `[probe: tools/probe_playlist_rename.py, Jellyfin 10.11.11, 2026-09-01]`. Neither analysed client posts anything but a changed `Name` | Item metadata editing entering the surface with a named consumer — which is more than a route: 004 T10 measured the scan and the refresh already fighting over `Item.name`, so an edited field needs somewhere to live that the next scan does not overwrite. Until then a refusal is the honest answer and a partial apply would be Principle VI's plausible-looking stub |
| **A required body that is missing entirely is `400` and not `415`** ([§1.11](#111-there-are-four-error-shapes-not-one)) | A request carrying no body and no `Content-Type` on one of the five routes whose body is required — 007's three reporting routes, 008's `PlaybackInfo` and 009's rename — answers the validation `400` where the reference answers `415 Unsupported Media Type` in the problem-details shape `[probe: tools/probe_playlist_rename.py, Jellyfin 10.11.11, 2026-09-01]`. No analysed client sends a body-less request to a route that requires one | A content-type gate in `compat/`, ahead of the body binding, and one measurement per required-body route rather than an extrapolation from this one: the shape is measured on the rename alone, and the four older routes were never asked. **[010](../../specs/010-conformance-harness/spec.md) is the owner of the measuring half** — a differential replaying a body-less request against both servers is what asks the other four, and it is the only thing that can — and the gate itself belongs to whoever owns `compat/model.py`, which every request model in the project inherits, rather than to any one feature. 009 T14 decided that rather than closing it inside a task about the acceptance map |
| **A multi-part film answers one media source per part** ([008 §3.1](../../specs/008-playback-negotiation-and-delivery/spec.md#31-media-sources)) | Two sources on one item, where the reference answers one source, a `PartCount` and a separate route for the rest | Not a gap to close on its own: it follows from 003 §3.3 modelling the parts as one item's sources, and closing it means changing that model or adding `GET /Videos/{id}/AdditionalParts` to the surface |

The difference between this section and §4 is intent. §4 says *we thought about it and chose
differently*. This section says *we have not done it yet, and here is how we will know when it
matters*.

---

### 5.1 `SupportedCommands` is not validated against the reference's enum

The reference binds that field to an enum and answers `400` with RFC 9457 problem details for a
value it does not know — the whole body is rejected, and the `errors` map names **both** the
offending element's JSON path and the field — `$[0]` and `capabilities` — for a bad value in the
first position. *(This entry said the map reported `capabilities` alone until the 2026-08-28 run
read the body back.)* `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]`

Atrium accepts it. v1 acts on **none** of those commands, so reproducing a thirty-value enum in
order to refuse values no working client sends is cost without a client that benefits — and the
divergence is class A in §3.0's terms: the reference fails loudly, so nothing can have been built
on the failure in a way that a success breaks.

**This is the opposite call from the one in §2.12**, where Atrium matches the reference's strictness
about whitespace in the client header, and the difference is worth stating so the two do not read as
inconsistent. There, matching cost one character in a regular expression and the field is an
authentication boundary. Here it costs an enum that has to be kept in step with the reference
forever, on a field nothing reads. The principle is the same in both: spend the cost where a client
could be harmed.

### 5.2 A container that has lost every file is not removed

**Jellyfin does: keep the row, exactly as Atrium does — measured on 2026-09-02, and the belief
this entry carried was wrong.** A series of six episodes emptied of every file and rescanned is
**still fetchable**, with zero episodes under it and **one row in a `Series` listing**
`[probe: tools/differential.py --named container-that-lost-every-file, Jellyfin 10.11.11,
2026-09-02]`. The reading was taken against the **single-use reference instance**
[010 §3.1](../../specs/010-conformance-harness/spec.md) stands up over this repository's own
fixture and destroys — the *"disposable library on a server somebody owns"* this entry had been
asking for since it was written — with the episodes deleted underneath it, the library rescanned on
the server's own answer, and the files put back.

*This entry read* ⚠️ **UNVERIFIED** *until that run.* It said the reference was **believed** to
remove such a container on the next scan; it does not. The marker is gone because the reading was
taken, and what it removes is not a doubt about Atrium but a difference nobody had: **the two
servers agree here.**

**The comparison is still outstanding as a comparison**, and that is a separate sentence.
`POST /Library/Refresh` is the reference's route and is not in
[surface.yaml](api-surface-v1.md) — Principle VI keeps an endpoint out until a client is measured
calling it — so Atrium cannot be asked for a second scan over the wire at all, and 010's report
names the row outstanding with that reason on every run. What is measured above is the half that
was unmeasurable before, and it is the half this entry was about.

The reading does not change the decision below. Atrium's reason for keeping the row is an argument
about **its own** guards, and it would have stood whichever way the reference went.

**Depends on it:** a user sees an empty series in their library instead of not seeing it. Visible,
and mildly annoying; nothing breaks and no state is lost.

**Atrium does:** keep the row. 003's scan marks every *file-backed* item removed and leaves the
containers above them alone, deliberately: removing a container is the same judgement 003 §3.8
refuses to make about a root — *this directory is empty, so it is gone for good* — made one level
down, where none of the three guards of [003 plan §6.5](../../specs/003-library-configuration-and-scanning/plan.md#65-the-guard-against-a-mass-delete)
is watching. A share that mounts half-empty takes its roots' guards with it; a *directory* that
mounts empty has nothing.

**The observable half belongs to 005**, which decides what `/Items` returns: a container with no
visible children is not offered, and the row's continued existence is then invisible. That is the
closing mechanism, and it is cheaper and safer than the alternative — a filter costs one predicate
and is wrong for one query, while a removal is written down and is wrong until somebody notices.

**Found at T20 and decided at T21.** 003 §3.8's table said "directory emptied → remove the container
item" from the day the specification was written, no acceptance criterion covered it, and nothing
implemented it. The three ways out were to implement it late in a feature whose removal semantics
were settled at T17, to delete the row and pretend it had never claimed anything, or to say plainly
what happens and who closes it. This is the third.

### 5.3 An artist in two music libraries is two rows

**Jellyfin does:** hold one server-wide item per artist name — artists are by-name items, like
genres, with ids derived from the name alone
`[source: Emby.Server.Implementations/Library/LibraryManager.cs:1030-1075 @ v10.11.11]`
`[source: Emby.Server.Implementations/ServerApplicationPaths.cs:59 @ v10.11.11]` — so the same
artist appearing in two music libraries is one row whose discography spans both.

**Depends on it:** only a setup with **two or more music libraries** sharing an artist can
observe the difference, and what it observes is cosmetic: the artist appears once per library in
`/Artists`, each entry listing that library's albums. Nothing breaks and no state is lost; it
looks wrong rather than behaving wrongly.

**Atrium does:** keep 003's per-library artist identity — `(type, library, folded name)` — which
was settled, implemented and accepted before this consequence had a surface to show on. Migrating
to a server-wide rule in 004 would rewrite identifiers 003 already derived, which is the one
operation this project treats as radioactive (003 plan §1). Recorded as an accepted gap rather
than fixed quietly or shipped silently: the closing mechanism, if multi-music-library setups turn
out to matter, is a deliberate identity migration with its own feature, informed by the
differential harness (010). Single-music-library servers — the shape every measured setup has —
cannot observe it at all.

**A second consequence, found by 004 T9 and larger than the first.** Because an artist is a *tree*
item here rather than a by-name one, it is the **scanner** that creates it — one per **album
artist**, from the album's own tags. A track's *performers* are frequently other people, and 004
records all of them (AC-6). So a performer who is nobody's album artist has a **name on the track
and no artist item behind it**: a client renders the name and cannot follow it, and `/Artists`
does not list them. In the reference, where artists are by-name items created on demand, every
performer has a row.

The three ways out were weighed at T9 and two are worse than the gap. Creating the missing item
from the refresh puts a tree item outside the scan that builds the tree, and the next scan — which
reconciles what it resolved against what exists — would mark it removed: a row that appears and
disappears every other scan. Dropping the credit loses the performer's name, which is the thing
AC-6 exists to keep. So the name is stored and the link is nullable
(`item_artists.artist_item_id`, revision 0004), which is that sentence in the schema. The closing
mechanism is the same deliberate identity migration as above; until then a client sees a complete
list of who played on a track and a shorter list of artists to browse.

### 5.4 No loudness scan, so a track without the tag has no gain

**Jellyfin does:** serve `NormalizationGain` on an item from **two** sources, in a fixed
precedence. A measured loudness value wins when one exists, converted to a gain against a −18 LUFS
reference; otherwise the value read from the file's tags is used
`[source: Emby.Server.Implementations/Dto/DtoService.cs:1000-1007 @ v10.11.11]`. The tag path reads
exactly one tag — the track gain, with a trailing unit suffix stripped
`[source: MediaBrowser.Providers/MediaInfo/AudioFileProber.cs:362-375 @ v10.11.11]`. The measured
path is a scheduled task that decodes every audio file, and it runs only for libraries whose
options enable it
`[source: Emby.Server.Implementations/ScheduledTasks/Tasks/AudioNormalizationTask.cs:82,101,173 @ v10.11.11]`
`[source: MediaBrowser.Model/Configuration/LibraryOptions.cs:49 @ v10.11.11]`. When neither source
has a value the property is absent, like every other null (§1.7).

**Depends on it:** no observed client. Neither of the two clients of
[api-surface-v1 §1](api-surface-v1.md#1-how-this-set-was-derived) reads the property — one has no
code path for it and puts volume levelling beyond its current version, the other carries it only
because its API layer is generated from the reference's document (survey by role, 2026-08-27;
[004 OQ-5](../../specs/004-metadata-resolution/spec.md#7-open-questions)). A client that *did* read
it would be a music client applying a per-track volume adjustment.

**Atrium does:** the tag half only. A track whose file carries the track-gain tag gets the same
number the reference would report from the same tag; a track without it reports nothing, where a
reference **with the scan enabled** would report a computed value. The scan is out of v1's reach
for a concrete reason rather than a shrug: it decodes every audio file end to end, which needs the
transcoding dependency 008 owns and which no part of 004 otherwise requires — putting it in the
scan path would make a first scan of a large music library cost hours of CPU for a field nothing
reads. The closing mechanism is 008: once a decoder is a dependency the server already has, the
task is a bounded addition, and the precedence above is the whole of the behaviour to reproduce.
The gap is invisible on the default configuration, because the option is off unless an operator
turns it on.

### 5.7 An empty library reads unplayed, where the reference's source reads it as played

**Jellyfin does:** decide a folder's played state by asking whether anything beneath it is
unplayed, which for a folder with no children at all is vacuously **true** - zero unplayed of zero
`[source: MediaBrowser.Controller/Entities/Folder.cs:1798-1840 @ v10.11.11]`. **On the wire it
answers `Played: false`, and the source reading does not reach the view at all** - measured on
2026-09-02 against a library with nothing in it, which no reachable server had:
`GET /UserViews` gives the empty library `Played: false` on both servers, and the one difference is
beside it - **`UnplayedItemCount` is `0` here and absent there**
`[probe: tools/differential.py --named empty-library-played-state, Jellyfin 10.11.11, 2026-09-02]`.
It was unmeasured until then because creating an empty library means writing into somebody's
server, which the probes deliberately never do.

**Atrium does:** answer `Played: false` with `UnplayedItemCount: 0`. The rollup reads
`total > 0 and played >= total`, so nothing under it means nothing watched.

**Depends on it:** no client branches on it, and a *user* sees it as a tick on an empty section -
which is the argument for the divergence rather than against it. A library that reads "watched"
before anything has been added to it is a poster with a tick on it and nothing behind the tick.
**The measurement makes the flag itself parity** and leaves `UnplayedItemCount` as the only
observable difference on an empty view; what to do about that number belongs to 007, which owns
the rollup, and not to the harness that found it ([010 §2](../../specs/010-conformance-harness/spec.md#2-scope)).

**Why the question is this narrow:** it can only be asked of a library. A `Series`, `Season`,
`MusicArtist` or `MusicAlbum` with nothing visible beneath it is not offered at all
(section 5.2's closing half), so an empty one of those has no row for a client to read a flag off.
`CollectionFolder` is the one exemption - an empty library stays in a sidebar - and it is
therefore the only shape where "vacuously played" is observable at all. **010 owns the
measurement**: a differential against a server with an empty library is the one way to settle it
without writing into a real one (007 spec section 3.5, OQ-7).

### 5.6 A default rescan does not notice a replaced poster

**Jellyfin does: notice it, at default depth — measured on 2026-09-02.** Replacing `poster.jpg`
beside a film whose own media file was not touched and rescanning the library **changed the item's
`Primary` image tag and the bytes it identifies**, on a stock reference at its default scan depth
`[probe: tools/differential.py --named replaced-poster-default-rescan, Jellyfin 10.11.11,
2026-09-02]`. It was *"unmeasured from here"* until then — deciding it meant writing into
somebody's library and rescanning it, which the probes deliberately never do — and the reading was
taken against the single-use reference instance of
[010 §3.1](../../specs/010-conformance-harness/spec.md) rather than against anybody's library, as
the **named comparison** D-6 added to §3.10.

So the gap below is real and is now measured on both halves rather than on one: the reference
re-reads the directory on a scan that finds no changed media file, and Atrium does not.

**Atrium does:** re-derive an item's artwork from its directory on every refresh that *reads* the
directory — and a default scan reads it only when the item's **media file** changed, because
003's change-detection signal is that file's size and modification time
([003 plan §6.4](../../specs/003-library-configuration-and-scanning/plan.md)). Replacing
`poster.jpg` beside an untouched film therefore changes nothing until a **deep** scan runs, and a
client keeps the old poster until then — correctly, because its `tag` is still the tag of the
image the server is still serving.

**Depends on it:** nothing can depend on it, but a *user* meets it: they replace a poster, rescan,
and see no change. The workaround is the documented one — a deep scan is 003's escape hatch for
exactly this class of change, "a library whose tags were rewritten in place by a tool that
preserved both size and time".

**Why it is a gap and not a fix:** widening the signal means stat-ing every candidate artwork name
of every item on every scan — dozens of names per item, which is the cost 003's `_by_name` listing
was written to avoid. That is a measurement somebody should take before paying it.

> **What was a bug and is fixed:** until 006 T12 the tag could not change *at all*, at any scan
> depth. `Field.IMAGES` merged under the rule that keeps whatever the item already has unless the
> mode is `Replace`, so an item that had ever been given artwork could never be given different
> artwork — and v1 has no refresh route through which anybody could ask for `Replace`. 006 AC-2's
> second half was unreachable, and the whole of client-side cache invalidation with it. The field
> is `REDERIVED` now: it has exactly one source, and "keep what we have" was protecting a stale
> index of a directory from the directory.

### 5.5 No BlurHash is computed, so `ImageBlurHashes` is always empty

**Jellyfin does:** send `ImageBlurHashes` on **every item of every response**, unasked — the same
shape as `ImageTags`, one BlurHash string per image tag, `{}` for an item with no images
`[probe: tools/probe_item_shapes.py, Jellyfin 10.11.11, 2026-08-27]`. The hash is computed when an
image is processed and stored beside the tag; a client renders it as a blurred placeholder while
the real image loads.

**Depends on it:** any client that renders placeholders reads it — which is why the property is in
005 §3.2's always-present set at all. A client finding the map empty renders no placeholder and
then the real image, so the failure mode is cosmetic latency, never a wrong image.

**Atrium does:** send the empty object, always. The property is present — its absence would be a
delta on every row of every list — and it is empty because Atrium computes no BlurHash and
inventing one would be a lie a client renders. Computing one for real belongs to the association
path 004 owns (the bytes are already open in Pillow when dimensions are read), needs a stored
column, and no consumer has asked yet; 010's differential will report the gap on every item, which
is the mechanism that decides when it closes. Recorded at 005 T9, where the emitter was written.

### 5.8 A chapter image can never be served in v1

**Jellyfin does:** generate chapter thumbnails in a background job and serve them at
`GET .../Images/Chapter/{index}`, advertising each with an `ImageTag` on the gated `Chapters`
field — 1,311 of 1,354 measured entries carried one
`[probe: tools/probe_image_tags.py, Jellyfin 10.11.11, 2026-08-28]`. A chapter whose image was
never generated answers the absent-image `404`.

**Depends on it:** the scrubbing UI of a video client. A client finding no tag draws no thumbnail
and scrubs blind at that chapter — the same failure mode a reference server that has not finished
generating them shows anyway.

**Atrium does:** serve the route and the `404`, always. 006 wired the route and the tag emission,
but the generation job — trickplay and chapter images — is excluded from v1 in its own right
([roadmap](../roadmap.md)), 004 extracts nothing, and no other v1 code path writes a chapter row;
`tests/conformance/test_image_routes.py`'s tripwire (`test_no_v1_writer_can_create_a_chapter_row`)
is what notices when that stops being true. So every chapter request in v1 is the absent-image
`404`, indistinguishable from a reference that has not generated them yet. **Closing mechanism:**
the feature that first writes chapter rows — trickplay and chapter-image generation, unscheduled —
at which point 006 §3.5's route serves them and this entry is withdrawn. Recorded at the
2026-08-28 audit (H2), which found §3.5 asserting the serving half with nothing able to exhibit
it — the exact shape of §5.2's history.

### 5.9 An unknown capabilities property survives into `/Sessions` here, and not there

**Jellyfin does:** accept a `POST /Sessions/Capabilities/Full` body carrying a property outside
its schema — the `204` of [002 §3.8](../../specs/002-authentication-users-and-sessions/spec.md) —
and **drop that property**: the session's `Capabilities` in `GET /Sessions` echoes the declared
fields and not the stranger. `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]`
*(The 2026-08-26 hand-measurement saw the `204` and recorded "kept"; reading the echo back on
2026-08-28 is what corrected it — the leniency is at the door, not in the echo.)*

**Depends on it:** nothing can. A client that posted an unknown property and read it back finds it
gone on the reference, so no working client is built on the echo.

**Atrium does:** keep it — the declaration is stored as the document the client posted and echoed
whole, so a property from a newer client than the schema survives the round trip. The divergence
is visible only to a client doing exactly what no client of the reference can usefully do.
**Closing mechanism:** filter the stored declaration to the reference's `ClientCapabilitiesDto`
members if the differential harness (010) ever shows a client observing the difference.

### 5.10 Dolby Vision and HDR10+ are not classified, so two emissions never fire

**Jellyfin does:** describe a stream-copied video's dynamic metadata twice over, in two places, and
both turn on a range *flavour* rather than on high dynamic range itself
`[probe: tools/probe_transcode_decision.py, Jellyfin 10.11.11, 2026-08-29]`:

| Where | What it writes | When |
|---|---|---|
| The master playlist's copy variant | `SUPPLEMENTAL-CODECS="dvh1.08.06/db1p"` | The source's range type is one of the Dolby Vision spellings, and the metadata is not being stripped |
| The same attribute | `SUPPLEMENTAL-CODECS="hvc1.2.4.L150.B0/cdm4"` — the variant's own codec string, then the marker | The source's range type is HDR10+ |
| The produced segment's sample entry | `dvh1` instead of `hvc1` | A Dolby Vision copy **whose client declared `DOVI`** in its range types; a client that did not is muxed `hvc1` `[source: Jellyfin.Api/Controllers/DynamicHlsController.cs:1838-1866 @ v10.11.11]` |

Measured on one Dolby Vision film in one run: `SUPPLEMENTAL-CODECS="dvh1.08.06/db1p"` on the copy
variant and on neither entrance, present whether or not the client declared Dolby Vision, and
`dvh1` read back out of the fMP4 initialisation segment by `ffprobe`. A plain HDR10 source beside
it carried neither.

**Depends on it:** a Dolby Vision player reads both. Neither is load-bearing for playback of the
HDR10 base layer, which is what a player that ignores them gets.

**Atrium does: neither, and the cause is one layer down.** [008 §3.1](../../specs/008-playback-negotiation-and-delivery/spec.md#31-media-sources)'s inspection derives a stream's
range from its colour transfer characteristics, which yields exactly three answers — standard
range, HDR10 and HLG — and the eight Dolby Vision spellings plus HDR10+ are not among them,
because that signal is not in a stream's colour metadata. So on **every source this server can
describe**, the reference sends no `SUPPLEMENTAL-CODECS` and tags `hvc1` either: the emission is
absent here and would be absent there, which is why writing it would be writing a branch nothing
can reach. What is really divergent is narrower and upstream of the playlist: a Dolby Vision file
is *inspected* here as the HDR10 file its colour metadata claims, and the two attributes are the
first place a client can see that.

**Closing mechanism:** teach the inspection to read the Dolby Vision configuration record and the
HDR10+ marker — a stream's side data rather than its colour metadata — which adds the range-type
members and the profile and level the attribute is assembled from. The two emissions then follow
from data that exists, and so does [§3.4](#34-hdr10-metadata-stripped-from-clients-that-asked-for-it--class-b-no-compensation)'s
divergence, which is written as a decision and is today equally unreachable for the same reason.

### 5.11 A subtitle file in a legacy encoding is decoded by a rule and not by a detector

**Jellyfin does:** run a **statistical detector** over the bytes of a subtitle file beside the
media before reading a cue out of it — a port of Mozilla's universal charset detector, carried as
a dependency `[source:
MediaBrowser.MediaEncoding/MediaBrowser.MediaEncoding.csproj:30 @ v10.11.11]` — and read the file
back through whatever encoding it names, in both of the two places that need one: the read that
produces the cues `[source:
MediaBrowser.MediaEncoding/Subtitles/SubtitleEncoder.cs:169-193 @ v10.11.11]`, and the name handed
to ffmpeg as `-sub_charenc` when a format its own parsers do not cover is converted `[source:
SubtitleEncoder.cs:965-995 @ v10.11.11]`. A detector of that kind names any of some thirty
encodings from the byte statistics alone, so a Cyrillic, Greek or Japanese subtitle file written
before UTF-8 was universal is read correctly without anything being declared anywhere.

**Depends on it:** every client that displays a subtitle. This is not a property a client branches
on — it is the **cue text itself**, and a player draws exactly the characters the server decoded.
That is why this entry is here and not in [§3](#3-defects): a §3 divergence
has to carry the argument that no client can observe the difference, and a client observes this one
directly, as the wrong letters on the screen.

**Atrium does:** decide by a rule with three steps and no statistics
(`media/extract.py`'s `_detect`): a **byte order mark** decides outright — the four a text file
can begin with, each read through the codec that consumes the mark rather than leaving it in the
text — then **strict UTF-8**, then a single declared fallback of `cp1252`.

* **Where the two agree, which is nearly everywhere.** Every subtitle file that is UTF-8, with or
  without a mark, and every UTF-16 or UTF-32 file that carries one, decodes to the same text on
  both servers. That is the overwhelming majority of what a library holds, all of what any fixture
  in this repository holds, and everything either analysed client has been observed to fetch.
* **Where they diverge.** A file in a legacy single-byte encoding that is *not* `cp1252` —
  `cp1251`, `iso-8859-7`, Shift-JIS and their neighbours — is decoded here as `cp1252` and comes
  back as mojibake, where the reference names the encoding and comes back correct. A file holding a
  byte `cp1252` leaves undefined refuses instead: the fetch answers the `500` a failed extraction
  answers, which is [011 plan §7](../../specs/011-subtitle-delivery/plan.md)'s own row for text
  nothing can decode.
* **Why the fallback is `cp1252` and not `latin-1`.** `latin-1` decodes every byte sequence there
  is, so it would turn *every* unreadable file into plausible-looking nonsense and make the refusal
  above unreachable. `cp1252` covers the same Western European files a real library holds and still
  has undefined bytes, so "this is not text in any encoding this server can read" stays a thing the
  server can say.

**Closing mechanism:** a detector behind `_detect`, which is one function with one call site
on each of the two paths — the same two the reference detects on — and a **runtime dependency
argued in an [ADR](../decisions/) at that point rather than now**.
[`pyproject.toml`](../../pyproject.toml)'s own rule is that the dependency set says what the code
needs rather than what the roadmap intends, and nothing in v1 has yet met a file it decodes
wrongly; the day a real library produces one is the day the argument has an input it does not
have today. Decided at 011 T6, where the choice was raised as a gate question and answered as a
documented gap rather than as a dependency.

## 6. Non-improvements

Principle I requires that good ideas which would create a delta get written down and then not done.
This list exists so they stop being re-proposed.

| Idea | Why not |
|---|---|
| An aggregate endpoint returning an artist's full discography in one call | Solves a real N+1, but no client would call it. It is a new endpoint: delta. A well-indexed `/Items` query solves the same problem within the contract |
| snake_case JSON behind a content-negotiation header | Every client would still use PascalCase; the alternative would rot untested |
| A capability-advertisement endpoint so clients can use Atrium's better paths | The definition of a delta. If a client has to ask what server it is talking to, the project has failed |
| Richer error bodies than Jellyfin's | Clients parse status codes; a different body shape is a difference they can observe |
| Numeric ids because they are easier to debug | Breaks §1.4 and every client's id parsing |
| Accepting whitespace around `=` in the client header, since the intent is obvious | The reference answers `401` (§2.12). No working client sends it, so tolerating it protects nobody — and it lets a client be developed against Atrium and then fail against Jellyfin |
