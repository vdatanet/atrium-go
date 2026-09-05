---
feature: 003-library-configuration-and-scanning
title: Library configuration and scanning
status: Implemented
created: 2026-08-26
updated: 2026-09-05
accepted: 2026-08-26
amended: 2026-08-27 by T1 - sections 3.2, 3.5 and the open questions; by T4 - section 3.7; by T5 - section 3.6; by T7 - sections 3.1 and 3.6 and OQ-2; by T11 - section 3.3 and OQ-4; by T12 - section 3.4; by T18 - section 3.8; by T19 - section 3.6 and OQ-2's limit; by T20 - sections 3.8 and 7; by 004's T7 - OQ-8. 2026-09-05 by the change that wrote plan.md - section 3.2's `.ignore` rule, section 3.3's multi-part marker vocabulary and OQ-9, each forced by a source reading at the pinned tag that the plan had to decide against; registered as U-42 to U-44 rather than treated as measurements. 2026-09-05 again, by the change that wrote tasks.md, in two places that are one finding - section 3.8 states the second guard against a mass delete (a root that reads as holding no candidate file where the last scan saw one is unavailable, not emptied, and an operator can say they meant it) and section 5 gains AC-16 for it in both halves. What forced it: writing a task per criterion made visible that the guard which catches the way an unmounted share actually arrives - readable, and empty - was in the plan and in no criterion, so a list written from section 5 alone would have tested the destructive case that is easy to construct and not the one that happens. 2026-09-05 again, by 003 T4 - section 3.7.2 gains the missing-number case for a `Season`, whose whole sort name is the prefix, and section 3.7.3 gains what its one sentence leaves out. Both are source readings at the pinned tag rather than measurements, and both were forced by having to write the derivation: the code has to answer a case the section does not state, and the honest place to answer it is the section. 2026-09-05 again, by 003 T7 - section 3.6's identity table splits `MusicAlbum` off the row it shared with `Series` and `MusicArtist`, because "the library root plus the normalised name" makes two artists' identically-named albums one item, and section 3.5 already says an album's identity comes from its album artist. Forced by writing the music resolver: the row and section 3.5 contradicted each other and the code had to pick one, and the fixture cannot tell the two apart because its five album names are distinct
implemented: 2026-08-27
depends_on: []
---

# 003 — Library configuration and scanning

> **This document describes WHAT and WHY only.** No technology names, no storage decisions.

## 1. Purpose

Turn directories of files into a hierarchy of items with stable identifiers.

This feature has **no HTTP surface of its own**, which is why it can be built in parallel with 001
and 002. Everything it produces becomes observable through feature 005, and every mistake it makes
becomes a wrong answer there.

**Client behaviour unlocked:** nothing directly — and everything indirectly. A client cannot show a
library that was never scanned.

## 2. Scope

**In scope**

- Library roots: what an operator configures, and what a collection type means.
- File discovery: which files are considered, which are ignored, and why.
- Resolution: turning paths into item types and parent-child structure for movies, series and
  music.
- Naming: extracting titles, years, season and episode numbers, disc and track numbers from paths.
- Identity: deriving identifiers that survive a rescan.
- Change detection: what an incremental scan must notice.
- Sort names: the normalisation that decides ordering everywhere else.

**Out of scope**

- Fetching metadata from anywhere, local or remote — that is 004. This feature produces *structure*
  and whatever the path itself tells us, nothing more.
- Inspecting media with a prober for codecs, duration and resolution — that is 008, and it is
  triggered by this feature but specified there.
- Filesystem watching. v1 rescans on demand and on a schedule.
- Books, photos, home videos, mixed-content folders.

## 3. Behaviour

### 3.1 Libraries

An operator configures one or more libraries. Each has a name, one or more root paths, and a
**collection type** from: `movies`, `tvshows`, `music`.

The collection type is not a hint. It selects which resolution rules apply, and a file under a
`music` root is never resolved as a movie no matter what it is called. Mixed-content roots are not
supported in v1: this is a real limitation of the reference too, and an operator with mixed content
is expected to split the root.

Each library becomes a `CollectionFolder` item, and each user sees it as a `UserView` (005) if
policy permits (002 §3.5).

### 3.2 What is considered a media file

A file is a candidate when its extension is on the type list for its library's collection type.
Everything else is ignored silently — a library root contains artwork, subtitles, `.nfo` sidecars
and operating-system detritus, and none of that is an error.

Ignored regardless of extension:

| Rule | Reason |
|---|---|
| Any path component beginning with `.` | Hidden files and macOS resource forks |
| A directory carrying an **empty** `.ignore` marker, or one anywhere between it and the library root | The operator's explicit exclusion; see below |
| Files matching the trailer, sample and extra suffixes | They are extras, not the work; §3.4 |
| Zero-byte files | Incomplete copies |
| Files being written, detected by size change between two passes | Downloads in progress |

**Amended 2026-09-05, while the implementation plan was written: the `.ignore` row named one of
three rules and read as though it were all of them.** At the reference the marker is searched for
from a file's own directory **upwards, to the filesystem root**; an **empty** or whitespace-only
marker excludes everything beneath the directory holding it; and a **non-empty** one is a set of
`.gitignore`-style patterns of which only the matching paths are excluded, with the fallback that a
marker whose every pattern fails to parse excludes everything
`[source: Emby.Server.Implementations/Library/DotIgnoreIgnoreRule.cs:18-30,41-68,95-131 @ v10.11.11]`.
This is a source reading and no probe here has sent a `.ignore` file of any kind, so it is
[U-42](../../docs/compatibility/reference-target.md) in the register rather than a measurement.

**What v1 does, stated as the rule it is.** An empty or whitespace-only marker excludes the
directory holding it and everything beneath it, and the search runs from a file's directory up to
**the library root and no further** — a marker above the root belongs to somebody else's directory
tree, and honouring one there would let a stray file in a home directory empty every library beneath
it. A **non-empty** marker excludes nothing. Both narrowings show *more* than the reference rather
than less, which is the safe direction for a scanner, and the second is an accepted shortfall: honouring
patterns is behaviour this project would own outright, for a shape no measurement shows anybody
using, and getting it subtly wrong hides files an operator expects to see.

**The extensions a collection type admits**, measured against a library of 8,288 items:

| Collection type | Admitted |
|---|---|
| `movies` | `.mkv` `.mp4` `.avi` `.ts` |
| `tvshows` | `.mkv` `.avi` `.mp4` |
| `music` | `.flac` `.m4a` `.dsf` |

`[probe: tools/probe_library_extensions.py, Jellyfin 10.11.11, 2026-08-27]`

This is a **measured lower bound, not the reference's configured list.** It is what one real
library contained; an extension nobody has a file of was not measured, and its absence here is not
evidence of refusal. v1 keeps the conservative union for anything outside the table, and the
measurement is repeatable, so a library holding a new extension reports it rather than being
guessed at.

**The lists do not fall back to one another, and that is the observable part.** Under `movies` and
`tvshows` roots the same measurement found 89 `.mp3` files and 3 `.mka` files, and **not one of
them became an item of any type** — not a film, not an episode, and not a track either, though the
same server admits three audio extensions under its `music` root. A file whose extension is not on
its own collection type's list is ignored; it is never promoted to another type because some other
list would take it. Theme music sitting beside a film is the ordinary case, and it is not an item.
`[probe: tools/probe_library_extensions.py, Jellyfin 10.11.11, 2026-08-27]`,
[behaviours §2.15](../../docs/compatibility/behaviours.md#215-an-audio-file-under-a-video-root-is-not-an-item)

### 3.3 Movies

| Layout | Resolves to |
|---|---|
| `Movies/The Film (1999).mkv` | One `Movie` |
| `Movies/The Film (1999)/The Film (1999).mkv` | One `Movie`, folder-per-film |
| `Movies/The Film (1999)/The Film (1999) - part1.mkv`, `- part2.mkv` | **One** `Movie` with two parts, not two movies |

**Title and year** are extracted from the name by stripping a trailing or bracketed four-digit year
in the 1900–2099 range, then removing release-tag noise — resolution, source, codec, audio format,
language, release-group brackets. Both are documented regex sets upstream.
`[source: Emby.Naming/Common/NamingOptions.cs:147-161 @ v10.11.11]`

Atrium reimplements the *rules*, not the expressions (Principle IV). The acceptance test is
behavioural: given a corpus of real-world names, the same title and year come out.

**Where a film sits in its own directory, the directory names it.** Measured across 1,557 films:
the directory's cleaned name matched what the reference resolved **1,087 times against the file's
457**. The reason is mechanical — the tools that fetch films mangle filenames and leave directories
alone. 135 of those films had a filename with **no spaces at all** while its directory had them,
and others were truncated mid-word or suffixed with the name of the site that served them.
`[read: Jellyfin 10.11.11, 2026-08-27]`

A directory holding **several different titles** is a category rather than a film, and names none
of them. That is the only part of the rule a single path cannot decide.

**Multi-part files** are grouped into one item when they differ only by a part marker. The parts
become one item's media sources, in order. Getting this wrong doubles a user's library, which is the
most visible possible scanning bug.

**Amended 2026-09-05, while the implementation plan was written.** ~~The marker was written as
`part1`/`pt1`/`cd1`/`disc1`, *"and the `-a`/`-b` form"*.~~ The second half of that is not a form the
reference has. A marker is one of **`cd`, `dvd`, `part`, `pt`, `disc` or `disk`**, separated from the
title by a space, underscore, dot or hyphen or by a closing bracket, and followed either by **digits**
or by a **single letter `a`–`d` after that same word**
`[source: Emby.Naming/Common/NamingOptions.cs:141-145 @ v10.11.11]`. A bare trailing letter is not a
part marker: `The Film - a.mkv` and `The Film - b.mkv` are two works. The correction matters more
than the two words it changes, and in the opposite direction to the warning above — reading a bare
letter as a part marker merges two films into **one** and makes the second disappear. Source read,
never probed: [U-43](../../docs/compatibility/reference-target.md).

### 3.4 Series, seasons and episodes

Three levels, and the middle one is often not a directory.

| Layout | Resolves to |
|---|---|
| `Shows/The Series/Season 01/The Series - S01E02 - Title.mkv` | `Series` → `Season 1` → `Episode 2` |
| `Shows/The Series/The Series - S01E02.mkv` | Same, with the season **inferred** from the episode |
| `Shows/The Series/Specials/…` | `Season 0` — `Specials` is an alias for season zero `[source: Emby.Naming/TV/SeasonPathParser.cs:82 @ v10.11.11]` |
| `…S01E02-E03…` | **One** episode item spanning two numbers, not two items |

**Season and episode numbers** come from a family of patterns: `S01E02` and its separators,
`1x02`, `E02`/`EP02`, and date-based naming (`2024-01-31`) for daily shows. The reference carries
about thirty expressions plus eight for multi-episode files.
`[source: Emby.Naming/Common/NamingOptions.cs:320-360, 754-763 @ v10.11.11]`

**Ambiguity is resolved by position**, not by preference: the pattern is matched against the
filename first, then against the parent directory. A series called `24` must not have its title
read as an episode number, and this is exactly where naive scanners fail.

**Extras** — trailers, featurettes, deleted scenes, interviews, behind-the-scenes — are recognised
by suffix and by containing-folder name, and do not become episodes.
`[source: Emby.Naming/Common/NamingOptions.cs:160, 697 @ v10.11.11]`

**v1 ignores them rather than attaching them**, and this paragraph previously said the opposite of
§3.2 — that they are "attached to their parent". Two things settle it. This feature produces
structure and nothing else, and an extra is not structure: it has its own title, its own artwork
and its own duration, which are 004's and 006's to fetch and 008's to measure. And there is nowhere
to attach one: an item's files are the parts of the work itself, so filing a trailer among them
would make it play as part of the film.

An operator loses nothing they can currently see — v1 has no surface that shows extras — and a
later feature that adds one starts from a rule that says what happens rather than from two
paragraphs that disagree.

**`Specials` is not an extras directory.** It is an alias for season zero, it sits beside `Extras`
and `Featurettes` in real libraries, and a scanner that grouped it with them would drop every
special episode in every series while producing a scan that looks entirely correct.

**A missing season directory is normal.** So is a season directory with no episodes, and an episode
whose number exceeds any real count. None of these is an error; all of them appear in real
libraries.

### 3.5 Music

| Layout | Resolves to |
|---|---|
| `Music/Artist/Album/01 Track.flac` | `MusicArtist` → `MusicAlbum` → `Audio` |
| `Music/Artist/Album/CD1/01 Track.flac` | One album; tracks carry a disc number |
| `Music/Artist/Album (2001)/…` | Album with a year |

Music inverts the priority of the other two types: **embedded tags outrank the path**. A file's
`albumartist`, `album`, `title`, `track` and `disc` tags are authoritative where present, and the
directory layout is the fallback. This is not a preference — a well-tagged library with a flat
directory structure must produce the right albums, and a compilation must not become one album per
track.

**Measured**, across 5,814 tracks: 413 of them (7.1%) carry an album name bearing no resemblance to
the directory holding the file, which is a name that cannot have come from the path. A further 129
resolved names keep **leading or trailing whitespace** — `Through the Barricades ` under a
directory called `Spandau_Ballet-Through_the_Barricades` — and a path cannot produce those,
because the name derived from a path is trimmed and a directory cannot end in a space. The tag is
copied verbatim, artefacts and all.
`[probe: tools/probe_music_precedence.py, Jellyfin 10.11.11, 2026-08-27]`

**What that measurement does not cover:** every album on the measured library lived in exactly one
directory, so the strongest form of the rule — a *flat* directory of well-tagged files resolving
into the right albums — is not proven, only its precedence is. A directory name merely decorated
with a year is counted separately and is not evidence of tag precedence: stripping a year from a
directory is path cleaning, and folding the two together would roughly double the number while
proving something weaker.

Reading those tags is feature 004. This feature produces the structure and *asks* 004 for the
identity; the ordering of that conversation is a plan concern.

**Album artist versus track artist** is the distinction that makes compilations work: an album's
identity comes from its album artist, so a compilation with a different artist on every track is
one album, not many. Measured: of 468 albums, 59 hold tracks by more than one artist and 33 of
those carry a single album artist throughout — each resolving to **one** album. The largest holds
60 tracks by 40 distinct artists under one album artist.
`[probe: tools/probe_music_precedence.py, Jellyfin 10.11.11, 2026-08-27]`

Where no album artist is present, the album is attributed to `Various Artists` only if the track
artists actually differ.

**The fallback is narrower than "the directory layout" suggests.** It holds for the album and the
artist, which the directories name, and **not** for the track number, the disc number or the title:
the reference takes a track and a disc number from the embedded tag, or failing that from the
number the container carries, and from nowhere else, and it names a file whose tags supply no title
after that file's whole name, leading digits included.
`[source: MediaBrowser.Providers/MediaInfo/AudioFileProber.cs:181 @ v10.11.11]`
`[source: MediaBrowser.MediaEncoding/Probing/ProbeResultNormalizer.cs:1369 @ v10.11.11]`
`[source: Emby.Server.Implementations/Library/ResolverHelper.cs:96 @ v10.11.11]`
So an untagged `01 - The Track.flac` is an item named `01 - The Track` with no track number, rather
than `The Track` as track one. The measured library cannot show this from the outside — all 5,814
of its tracks carry a title tag, so the fallback never runs there
`[read: Jellyfin 10.11.11, 2026-08-27, /Items?IncludeItemTypes=Audio&Fields=Path]` — and the 77.9%
agreement recorded for track numbers is how often a tag matched the name beside it, not evidence of
a fallback the reference has.

Atrium reads all three from the name anyway when nothing else supplies them, which is a difference
a library of untagged music can see. **OQ-8** holds that open. What follows from the measurement
either way is the tie-break *within* the fallback: a name Atrium declines to take a number out of
is a name it agrees with the reference about, so an ambiguous one is read as saying **less**. A
leading number is a track number only when something separates it from what follows, so
`24K Magic.flac` is a song called `24K Magic` and not track 24 of `K Magic`, and a file named after
a hash keeps its name whole. Recorded in
[behaviours §2.16](../../docs/compatibility/behaviours.md#216-a-music-tracks-number-comes-from-tags-never-from-its-filename).

### 3.6 Identity

Every item gets an identifier that is **32 lowercase hexadecimal characters** and **stable across
rescans** — including a rescan into an empty database.

Stability is the whole requirement. Clients key their caches, favourites and resume positions on
these strings; an identifier that changes when a library is rescanned silently discards a user's
state. Identifiers are therefore **derived from the item's stable identity**, never allocated.

The stable identity is:

| Item type | Derived from |
|---|---|
| Movie, Episode, Audio | The item's path, relative to its library root |
| Season | Its series' identity plus its season number |
| MusicAlbum | Its album artist's identity plus the normalised album name |
| Series, MusicArtist | The library root plus the normalised name |
| CollectionFolder | The library's configured identity |

*(The `MusicAlbum` row was ~~`Series, MusicAlbum, MusicArtist` — the library root plus the
normalised name~~ until 2026-09-05, when writing the music resolver made it a merge rather than a
derivation. Two artists each holding a `Greatest Hits` produce one album row under that reading:
one identifier, one parent, and half of an album's tracks appearing under an artist that did not
record them. §3.5 is the section that settles it and already says so — **"an album's identity comes
from its album artist"** — so the row now reads the way the `Season` row above it already does, as
a container's identity plus the one thing that distinguishes it inside that container. Nothing in
this project's fixture tree exercises the merge, because its five album names are distinct, which
is why the row could have shipped wrong and green.)*

**Normalised** means the same thing for a path and for a name, and it is three steps: separators
reduced to one form, the text reduced to one Unicode form, and case folded — the last only when the
library is not case-sensitive. Each exists because the *same* file or name would otherwise produce
two identifiers: a walker on one platform yields one separator and on another the other; one
filesystem hands back a decomposed accent where another gives the precomposed character; and a
directory renamed only in its capitalisation is not a different directory. A path that is absolute,
or that climbs above its root, is refused rather than normalised — either one means the caller has
a path that is not relative to the root it believes it is.

**Reproducing the reference's exact identifiers for the same file is not a goal**, and the reasoning
is in [behaviours §1.4](../../docs/compatibility/behaviours.md#14-item-identifiers-are-32-lowercase-hex-characters).

**Relative to the root, not absolute.** An operator who moves a library from `/mnt/a` to `/mnt/b`,
or who runs the same library from a container with a different mount point, must not lose every
identifier. The reference has this problem — every one of 448 measured identifiers is reproducible
from the file's absolute path alone, containers included
`[probe: tools/probe_item_identity.py, Jellyfin 10.11.11, 2026-08-27]` — and Atrium does not have
to inherit it.

**Case sensitivity is a property of a library, not of the server, and it is fixed when the library
is created.** Paths are compared without regard to case by default. That is Atrium's decision, and
it is stated as one: **what the reference defaults to is not something this repository has
measured**, and OQ-9 records it as the open question it is rather than as a claim wearing a hedge.
What *is* measured is that the reference has such a setting and that it decides the key — the
server measured for identity has it switched on, and reproduces its identifiers from the path
verbatim `[probe: tools/probe_item_identity.py, Jellyfin 10.11.11, 2026-08-27]`.
An operator who needs two files differing only in case to be two items says so when the
library is declared. The setting is recorded with that library and **an attempt to change it
afterwards is refused, not accepted with a warning** — changing it rewrites every identifier under
it, and nothing stores the old ones to undo with. Making it a server-wide switch would mean one
flip rewrote every identifier in every library at once.

**The same applies to a library's own identity, and it is easier to trigger by accident.** A
library's identity is allocated when it is declared and kept, rather than derived from its name or
its roots — so renaming a library, or moving its roots to another mount, costs nothing. The
consequence worth stating plainly: *deleting* a library and declaring another one with the same
name and the same roots is **not** the same library, and every item under it gets a new
identifier. Editing a library is free; recreating one is not.

**A library's collection type is fixed at creation too**, and refused afterwards for the same
reason: it selects which resolution rules apply, so changing it re-resolves every file under a
different set of rules and gives every item a new type and a new identifier.

### 3.7 Sort names

Every item has a sort name that decides its position in every list feature 005 returns. Getting
this wrong is visible on every screen a client draws, which is why it was measured before anything
was built. `[probe: tools/probe_sort_names.py, Jellyfin 10.11.11, 2026-08-26]`

**There is not one rule. There are two**, and the second is not a refinement of the first.

#### 3.7.1 The base derivation

Used by movies, series, albums, artists and playlists. Applied in this order, and the order
matters:

| # | Step |
|---|---|
| 1 | Trim surrounding whitespace, then lowercase |
| 2 | Remove each configured article: from the **start** when followed by a space, from **anywhere** when surrounded by spaces, from the **end** when preceded by a space |
| 3 | Remove each configured character outright |
| 4 | Replace each configured character with a space |
| 5 | Left-pad **every** run of digits with zeros to a fixed width |
| 6 | Fold diacritics; transliterate anything still outside ASCII |

Defaults for the three configured lists — articles `the a an`, removed characters `, & - { } '`,
replaced characters `. + %`, pad width 10 — reproduce every measured case:

| `Name` | `SortName` | What it shows |
|---|---|---|
| `The Matrix` | `matrix` | article at the start |
| `Matrix The` | `matrix` | **and at the end** |
| `Once The Time` | `once time` | **and in the middle** |
| `A Bridge` | `bridge` | single-letter article |
| `Amélie` | `amelie` | diacritics folded |
| `iRobot` | `irobot` | case normalised |
| `2 Fast 2 Furious` | `0000000002 fast 0000000002 furious` | **every** digit run, not just the leading one |
| `10 Things` | `0000000010 things` | which is what makes 2 sort before 10 |
| `Wall-E` | `walle` | character removed |
| `Rock & Roll` | `rock  roll` | **two spaces** — nothing collapses them |
| `Don't Look Up` | `dont look up` | apostrophe removed |
| `S.W.A.T.` | `s w a t ` | **trailing space** — nothing trims it |
| `100% Wolf` | `0000000100  wolf` | replacement and padding together |
| `  Padded  ` | `padded` | trimmed at step 1, before anything else |

**The whitespace artefacts are part of the contract.** `rock  roll` keeps its double space and
`s w a t ` its trailing one because steps 3 to 5 neither trim nor collapse. An implementation that
tidied them would sort differently from the reference — quietly, and only for names containing
those characters. This is the kind of detail that is invisible in a specification written from
intuition and obvious in one written from a measurement.

**Numeric ordering is not numeric comparison.** It is lexical comparison over zero-padded digit
runs, which is why the pad width is part of the contract: a different width produces a different
ordering between names whose digit runs differ in length.

> ⚠️ **OQ-7.** Step 6 says "transliterate anything still outside ASCII", and the only case measured
> was `Amélie` — which needs no transliteration, because `é` decomposes and folding alone reaches
> it. What the reference does with a character that has **no** ASCII decomposition (`ø`, `ß`, `æ`,
> or a name in a non-Latin script) is unmeasured. v1 folds, then applies a short table of the
> obvious Latin readings, then drops what is left; dropping is at least stable, which a partial
> guess would not be.

The three lists are server configuration rather than protocol. Atrium exposes them with the same
defaults and honours them the same way.

#### 3.7.2 The three types that replace it

`Audio`, `Episode` and `Season` **do not use §3.7.1 at all**. Each builds a numeric prefix and
appends the **raw** name — no lowercasing, no article removal, no diacritic folding, no digit
padding.

| Type | Sort name | Example |
|---|---|---|
| `Audio` | disc padded to 4, `" - "`, track padded to 4, `" - "`, name | `0001 - 0003 - The Song` |
| `Episode` | season padded to **3**, `" - "`, episode padded to **4**, `" - "`, name | `001 - 0002 - Pilot` |
| `Season` | season padded to 4, and **nothing else** | `0004` |

`[source: MediaBrowser.Controller/Entities/Audio/Audio.cs:94-98,
MediaBrowser.Controller/Entities/TV/Episode.cs:238-242,
MediaBrowser.Controller/Entities/TV/Season.cs:149-152 @ v10.11.11]`

The asymmetry is real and not a transcription error: an episode's **season** is three digits while
its **episode number** is four. A missing number contributes no prefix segment at all rather than a
run of zeros.

**A `Season` with no number keeps its raw name**, because a season's whole sort name is the prefix
and dropping the segment would leave nothing. This is not a measurement — nothing sent the reference
an unnumbered season — but the source is unambiguous
`[source: MediaBrowser.Controller/Entities/TV/Season.cs:151 @ v10.11.11]`, and the case arises here:
§3.4 infers a season where no directory exists, and the reference's recorded reading of the fixture
tree contains a `Season Unknown`. The alternative reading — the empty string — sorts every
unnumbered season ahead of every numbered one in every series at once.

**Consequence worth stating plainly:** a track called `The Song` sorts under `T`, not under `S`.
Applying §3.7.1 to audio — which is the natural thing to do when a codebase has one sort-name
function — reorders every album in the library.

#### 3.7.3 Explicit sort titles

Metadata (004) may carry an explicit sort title. It replaces the derivation entirely, for every
type, and is lowercased and digit-padded but not article-stripped.

**That sentence is exhaustive rather than illustrative**, and the source agrees with it: an explicit
sort title is digit-padded and diacritic-folded and then lowercased, and it is **not** trimmed, not
article-stripped, and neither configured character list is applied to it
`[source: MediaBrowser.Controller/Entities/BaseItem.cs:535-536 @ v10.11.11]`. So of §3.7.1's six
steps it uses two, not five. A title an operator or a tag wrote is taken as written, which is what
writing one is for.

This is a source reading and not a measurement, and it neighbours OQ-6 rather than closing it: the
library measured contained almost no item carrying an explicit sort title, so nothing here was
observed running. OQ-6 asks whether §3.7.2's formulas survive such a title; the source says they do
not, for any type, which is what this section states — and the row stays open, because a running
server is the tie-breaker.

### 3.8 Scanning and change detection

A scan is **incremental by default** and **deterministic**: the same tree scanned twice produces
the same items, the same identifiers and the same ordering.

| Change on disk | The scan must |
|---|---|
| New file | Add the item, creating ancestors as needed |
| File modified (size or time of change) | Re-inspect and update, **preserving identity and user data** |
| File deleted | Remove the item, **preserving user data** in case it returns |
| File renamed | Treated as delete plus add — identity is path-derived, so it changes |
| Directory emptied | Remove every item that was in it. **The container itself stays**, and stops being returned because nothing is under it — see below |

**User data outlives items.** A file that disappears and comes back — a re-download, a remount, a
temporarily unavailable network share — must not cost the user their favourites and resume
position. This is why user data is keyed by identity and retained after the item is gone.

**An unavailable root is not an empty root.** If a library root cannot be read at all, the scan for
that library fails loudly and changes nothing. Treating an unmounted share as "every item was
deleted" is the single most destructive thing a scanner can do.

**Amended 2026-09-05, while the task list was written: that rule guards the failure it names and not
the form the failure usually arrives in.** A share that fails to mount is very often perfectly
*readable* — the mount point is an ordinary empty directory — so *"cannot be read at all"* is a
condition an unmounted share does not meet, the walk finds nothing, and the scan computes the
deletion of everything the library holds. So: **a library root that reads as holding no candidate
file, where the previous scan of that library recorded at least one, is treated as unavailable
rather than as emptied.** The scan refuses, names the root, and changes nothing. Zero is the
threshold because it needs no number — it is the mount failure's signature, where *"fewer than
before"* is a judgement nobody could defend — and a root an operator really emptied is a deliberate
act, so **an operator can say explicitly that they meant it** and the scan then proceeds and removes.
What forced the amendment: writing the task list found that this rule had **no acceptance
criterion**, so a list written from §5 alone would have tested the destructive case that is easy to
construct and not the one that happens. §5 gains AC-16.

**A container that has lost every one of its files keeps its own record**, and this is a change from
what this section first said. A series whose episodes are all deleted still has a series; what it
does not have is anything to show. Removing the container instead would mean deciding, at scan
time, that a directory somebody emptied this afternoon is gone for good — the same judgement the
paragraph above refuses to make about a root, made one level down and with no guard watching it.
So the record stays and **a container with nothing visible under it is not offered to a client**,
which is the observable half and which the query behaviour owns rather than the scan. Recorded as a
bounded gap in [behaviours §5](../../docs/compatibility/behaviours.md#5-accepted-gaps-in-v1) until
that query behaviour exists.

**A scan is a guess about what is worth looking at.** Deciding a file has not changed is cheap
and occasionally wrong: a file can be replaced by one of the same length whose recorded time of
change was put back with it, which is what an ordinary copy or restore does. An operator can
therefore ask for a **full re-examination** that ignores the signal and looks at every file. The
default is the fast one, the full one is always available, and neither is described as the other.

A scan reports progress and a summary: items added, updated, removed, files examined, files
skipped with the reason, and files that **were** scanned but whose names said too little to place
the item they produced. The last two are counted apart. A file that was skipped is not in the
library and a file that was noticed is, so an operator told that both were "skipped" would go
looking for something that is not missing.

## 4. Data the feature owns

| State | Observable as (via 005) | Lifetime |
|---|---|---|
| Library configuration | `UserViews`, `CollectionFolder` items | Until the operator changes it |
| The item hierarchy | Every `/Items` response | Until the underlying files change |
| Item identity | The `Id` of every item | Permanent for a given path and library |
| Sort names | The ordering of every list | Recomputed on rescan |
| Scan state | Scan progress and summary | Per scan |

## 5. Acceptance criteria

1. The fixture library scans to exactly the expected set of items, with the expected parent-child
   structure, for all three collection types.
2. Scanning twice produces byte-identical item identifiers, and the second scan reports no changes.
3. Scanning into an empty database produces the same identifiers as the first scan did.
4. A multi-part film resolves to **one** item with two media sources, not two items.
5. `S01E02-E03` resolves to **one** episode item spanning both numbers.
6. A `Specials` folder resolves to season 0.
7. A series named with digits (`24`) keeps its title and does not acquire an episode number.
8. A two-disc album resolves to one album whose tracks carry the right disc numbers.
9. A compilation with a different artist per track resolves to **one** album.
10. Moving the library root to a different path leaves every identifier unchanged.
11. Deleting a file removes the item but leaves its user data recoverable; restoring the file
    restores the association.
12. A root that cannot be read fails the scan for that library and removes nothing.
13. Sort ordering matches the table in §3.7 for the fixture's awkward names.
14. An incremental rescan notices exactly what changed: a modified file — size or time of
    change — is re-inspected and keeps its identity and its user data; a file that appears is
    added; a renamed file is a delete plus an add, because identity is path-derived (§3.8).
    *(Added at the 2026-08-28 audit — M31: of §3.8's change table only the deletion row had a
    criterion.)*
15. An explicit sort title from metadata replaces the §3.7 derivation entirely, for every type —
    the overriding three included — and is lowercased and digit-padded but not article-stripped
    (§3.7.3). *(Added at the same audit — M32.)*
16. A library root that reads as holding **no candidate file**, where the previous scan of that
    library recorded at least one, refuses the scan and removes nothing — **and** the same scan, with
    the operator's explicit permission to proceed over an empty root, removes every item that was
    under it (§3.8). *(Added 2026-09-05, while the task list was written. §3.8's unavailable-root
    rule guards a root that **cannot be read**, and a share that mounts as an empty directory can be
    read perfectly well; the rule that catches that form had no criterion at all. Both halves are
    named because a test asserting only the refusal passes on a build whose override does nothing.)*

## 6. Conformance

This feature has no endpoints, so L0/L1 do not apply. It is proven at **L2** against the fixture
library, and its results reach L3 through feature 005.

| Behaviour | Level | How it is proven |
|---|---|---|
| Resolution for all three types | **L2** | Fixture library with the awkward cases of §5 |
| Identity stability | **L2** | Scan, rescan, and scan-into-empty comparison (AC-2, AC-3) |
| Sort normalisation | **L2** | Table-driven, plus L3 through `/Items` ordering |
| Change detection | **L2** | Fixture mutated between scans |
| Destructive-failure safety | **L2** | Scan with a root made unreadable (AC-12) |

The fixture library is checked in as **metadata only** — directory trees, sidecars, and synthetic
media generated at build time. No copyrighted media, ever.

## 7. Open questions

| # | Question | Blocks | Resolved by |
|---|---|---|---|
| OQ-6 | Whether the §3.7.2 formulas hold for items carrying an explicit sort title, and how many real items do | §3.7.3 | The override rows of `tools/probe_sort_names.py`, read against a larger library |
| OQ-7 | What the reference does with a character that has no ASCII decomposition — `ø`, `ß`, a non-Latin script | The ordering of those names, and nothing else | Crafted names in `tools/probe_sort_names.py`; the measured set contains none |
| OQ-8 | Whether a track number, disc number and title should be read from a file's name at all, given the reference reads none of the three from there (§3.5) | Only untagged music: for a tagged file both answer from the tag | How much real music carries no readable tag. **Half of this moved on 2026-08-27**: 004 T7 built the reader, so the question is now measurable against a real library and is no longer waiting on code. What it still needs is a *library* — this suite's music is silence we generated, so measuring the untagged fraction here would measure the fixture. A probe over the reference server's own music library answers it |
| OQ-9 | What the reference **defaults** `EnableCaseSensitiveItemIds` to | Nothing. §3.6 states Atrium's own default and does not claim to match one | `tools/probe_item_identity.py` against a server that has not changed the setting; the one measured has it **set**. **The source says `true`, and the row stays open — 2026-09-05**: it is declared `= true` and is a property of the **server**, not of a library `[source: MediaBrowser.Model/Configuration/ServerConfiguration.cs:89 @ v10.11.11]`, consulted at `[source: Emby.Server.Implementations/Library/LibraryManager.cs:650 @ v10.11.11]`. A source reading is not a measurement and the running server is the tie-breaker, so this stays open; what it changes is that §3.6's case-insensitive default is a **known** difference rather than a possible one — two files differing only in case are one item here and two there, which is an item count and therefore observable. [U-44](../../docs/compatibility/reference-target.md) |

**None of them blocks this feature, and each is open for its own reason.** Each needs a *measurement this
repository cannot take today*, not a decision somebody has been avoiding. OQ-6 needs a library
containing items with explicit sort titles, and the one measured has almost none; OQ-7 needs names
carrying characters the measured set does not contain. Both change the **ordering** of names that
are already scanned, found and playable — so a wrong answer is a list in a slightly wrong order,
not a missing item or a lost identifier. Closing either by guessing would replace "unmeasured" with
"asserted", which is the failure the provenance rule exists to prevent. Whoever next points a probe
at a larger library answers them.

### Resolved

| # | Question | Answer | Resolved by |
|---|---|---|---|
| OQ-3 | The reference's sort-name normalisation | **Six ordered steps, three configurable lists, pad width 10 — and three item types that bypass all of it.** §3.7 now states both rules; 15 of 15 crafted cases matched | `tools/probe_sort_names.py`, 2026-08-26 |
| OQ-1 | The exact extension lists the reference honours | **Measured, and the lists do not fall back to one another**: `movies` `.mkv` `.mp4` `.avi` `.ts`; `tvshows` `.mkv` `.avi` `.mp4`; `music` `.flac` `.m4a` `.dsf`. 89 `.mp3` and 3 `.mka` files under video roots produced **no item of any type**. A lower bound, not the configured list — §3.2 says which part is measured and which is not | `tools/probe_library_extensions.py`, 2026-08-27 |
| OQ-2 | Case sensitivity of path normalisation for identity | **A per-library fact, not a server decision**, defaulting to case-insensitive — and **frozen once the library exists**, with the change refused rather than warned about. The question asked whether Atrium should treat case as a global decision or a per-library fact, and that is answered. What the *reference* defaults to was never part of it and is now OQ-9, measured by nothing. §3.6 states it, and the same paragraph now records the two neighbouring traps: a library's identity is allocated rather than derived, so recreating a library is not editing one, and its collection type is frozen for the same reason | A decision plus a recorded per-library setting, 2026-08-27 |
| OQ-4 | Does the reference merge a folder-per-film layout when the folder and file names disagree? | **The interesting half of the question does not arise.** Across 1,480 one-film directories, a directory and a file naming two genuinely *different* works did not occur once; what occurs is the directory naming the same work more cleanly, and there the directory wins — 1,087 matches against the file's 457. §3.3 records the rule and the reason | Read against a live library, 2026-08-27 |
| OQ-5 | What the reference does with a file whose embedded tags contradict its path | **The tag wins, verbatim.** 413 of 5,814 tracks carry an album name with no resemblance to their directory, and 129 keep whitespace a path cannot produce. 33 compilations resolve to one album each. Not covered: a genuinely flat directory, which the measured library had none of | `tools/probe_music_precedence.py`, 2026-08-27 |

## 8. References

- [docs/compatibility/behaviours.md §1.4](../../docs/compatibility/behaviours.md#14-item-identifiers-are-32-lowercase-hex-characters)
- [docs/glossary.md](../../docs/glossary.md) — item types and by-name items
- Jellyfin v10.11.11: `Emby.Naming/Common/NamingOptions.cs`, `Emby.Naming/TV/SeasonPathParser.cs`,
  `Emby.Naming/Video/`, `Emby.Naming/Audio/` — read for **rules**, never copied
