package library

import (
	"strings"

	"github.com/vdatanet/atrium-go/internal/ports"
)

// This file is 003 §3.5's three levels — an artist, an album, and a track —
// and the seam feature 004 fills.
//
// # Music inverts the priority the other two types have
//
// For a film and for an episode the path is the only source there is. For a
// track the **embedded tags outrank the path** (003 §3.5), and the path is the
// fallback. Reading tags is 004's; this feature produces the path's answer for
// every file and asks a [TagSource] first, so that the day the reader exists
// nothing about the grouping has to be rewritten.
//
// # The fallback is narrower than "the directory layout" suggests, and part of
// it is a declared divergence
//
// The path answers for the **album** and the **artist**, which the directories
// name. It does **not** answer, at the reference, for the track number, the
// disc number or the title: those come from the embedded tag or from the
// number the container carries and from nowhere else, and a file whose tags
// supply no title is named after its whole filename stem, leading digits
// included
// `[source: MediaBrowser.Providers/MediaInfo/AudioFileProber.cs:181-182,312 @ v10.11.11]`
// `[source: MediaBrowser.MediaEncoding/Probing/ProbeResultNormalizer.cs:1369 @ v10.11.11]`
// `[source: Emby.Server.Implementations/Library/ResolverHelper.cs:96 @ v10.11.11]`.
//
// Atrium reads all three off the path anyway when nothing else supplies them,
// which is a difference a library of untagged music can see
// ([behaviours §2.16], 003 §3.5, and **OQ-8 holds the decision open**). The
// tests that assert it are written as the **divergence** it is rather than as
// agreement, so that the day OQ-8 is answered a failing test is the
// notification and not a rediscovery.
//
// What is already settled is the tie-break inside that fallback, and it is a
// **narrowing**: every stem Atrium declines to find a number in is a stem it
// agrees with the reference about, so an ambiguous shape parses as saying
// *less*. See [parseTrackName].
//
// [behaviours §2.16]: ../../docs/compatibility/behaviours.md#216-a-music-tracks-number-comes-from-tags-never-from-its-filename

// Tags are what a metadata reader found inside one file: the five 003 §3.5
// names as authoritative, plus the track artist its compilation rule needs.
//
// Every field is empty or nil when the file said nothing, and **empty means
// absent**: a track whose title tag is the empty string has no title tag, and
// there is no case in which an item wants to be called nothing.
type Tags struct {
	// AlbumArtist decides which album a track belongs to, which is what makes
	// a compilation with a different artist on every track **one** album
	// rather than one per track (003 §3.5).
	AlbumArtist string

	// Artist is the track's own artist. It is not an album's identity, and it
	// is here for the one rule that consults it: where nothing else supplies
	// an album artist, an album whose track artists **differ** is attributed
	// to `Various Artists`. See [resolveMusic].
	Artist string

	// Album is the album's name.
	Album string

	// Title is the track's name.
	Title string

	// Track and Disc are the numbers. They are pointers for the reason
	// `ports.ScannedItem`'s are: a file that carries no track number and one
	// that carries track 0 are different answers.
	Track *int
	Disc  *int
}

// TagSource is asked what one candidate file's embedded tags say.
//
// # Why it is an interface here and implemented elsewhere
//
// 003 §3.5 makes tags outrank the path and says in terms that reading them is
// **004's**, and 003 plan §6.2 settles the ordering of that conversation: the
// source is consulted **once per file, before grouping**, because the album
// artist decides which album a track belongs to and grouping cannot be redone
// afterwards without making the answer depend on the order the entries arrived
// in (Principle VII).
//
// So this is the seam, and it is shaped for the implementation that fills it
// rather than for the one v1 ships. A source is built **for one library**, so
// it holds that library's roots and needs only the ordinal and the
// root-relative path to open a file; root indexes `ports.Library.Roots`.
//
// # Why it cannot fail
//
// There is no error return, and that is a decision rather than an omission. A
// file whose tags cannot be read is neither a skipped file nor an unplaceable
// item (003 §3.8 counts those two and no third thing): it is an item resolved
// from its path, which is exactly what every file in v1 is. A reader that
// could not open a file answers the zero [Tags] and reports the failure
// through whatever it logs with — a question that belongs to the feature that
// owns the reader.
type TagSource interface {
	TagsFor(root int, path string) Tags
}

// NoTags is the [TagSource] v1 ships: it reads nothing and answers nothing.
//
// It is a type rather than a nil check so that the fallback path and the
// tag-driven path are the **same** code with one different collaborator. A
// resolver with an `if tags == nil` branch has two behaviours and tests only
// one of them.
type NoTags struct{}

// TagsFor answers the zero [Tags] for every file.
func (NoTags) TagsFor(int, string) Tags { return Tags{} }

// variousArtists is what an album is attributed to when nothing supplies an
// album artist and its tracks name more than one (003 §3.5).
//
// It is 003 §3.5's own spelling and **not** a reference constant: the string
// appears nowhere in the reference's source at the pinned tag, so this is
// Atrium's specification being implemented rather than a measurement being
// reproduced. It is a literal here for the reason the season-zero name is one:
// there is nothing in v1 to configure it with.
const variousArtists = "Various Artists"

// ---------------------------------------------------------------------------
// What a name says
// ---------------------------------------------------------------------------

// trackNumberSeparators is the run that may stand between a leading track
// number and the title behind it.
//
// It is deliberately **not** [partSeparators], [tagSeparators] or
// [numberSeparators]. Those belong to the film and episode rules and are read
// from the reference's own expressions; this one has no reference behind it at
// all, because the reference takes no track number off a filename
// ([behaviours §2.16]). Writing one set in terms of another would make a change
// to either silently change the other.
const trackNumberSeparators = " _.-"

// trackNameTrimSeparators are taken off both ends of the title left behind
// when a track number is removed.
const trackNameTrimSeparators = " \t_.-"

// parseTrackName splits a filename's stem into the title and the track number
// 003 §3.5's fallback reads out of it.
//
// # The tie-break, and the direction it runs in
//
// A leading number is a track number **only when something separates it from
// what follows**. That is 003 §3.5's own rule and the reasoning is worth
// keeping beside the code: every stem Atrium declines to find a number in is a
// stem it **agrees with the reference about**, because the reference finds a
// number in none of them. So an ambiguous shape is read as saying *less*, and
// the three shapes that matter are:
//
//   - `24K Magic` is a song called `24K Magic`, not track 24 of `K Magic`;
//   - `01 - Opening` is track 1 called `Opening`;
//   - a file named after a hash keeps its whole name, whether or not the hash
//     begins with digits.
//
// A stem that is nothing but digits keeps its name too: a number with no title
// behind it is a filename, not a numbering.
func parseTrackName(stem string) (string, *int) {
	if number, after, ok := readNumber(stem, 0); ok {
		if after < len(stem) && strings.ContainsRune(trackNumberSeparators, rune(stem[after])) {
			if title := strings.Trim(stem[after:], trackNameTrimSeparators); title != "" {
				return title, &number
			}
		}
	}
	return strings.TrimSpace(stem), nil
}

// discPrefixes is the vocabulary a disc directory may spell itself with,
// transcribed from the reference's `AlbumStackingPrefixes`
// `[source: Emby.Naming/Common/NamingOptions.cs:183-193 @ v10.11.11]`.
//
// It is **data**, transcribed for the same reason the release tags, the extras
// folder names and the season words are: a vocabulary is a list of facts about
// what the reference recognises, where an expression is a program
// (Principle IV).
//
// It is **not** `partTypes`, which is the film-stacking vocabulary, and the
// two differ in both directions: `dvd` and `pt` stack a film and not a disc,
// and `digital media`, `vol`, `volume` and `act` name a disc and not a film
// part. A single shared list would be wrong for both.
var discPrefixes = []string{"cd", "digital media", "disc", "disk", "vol", "volume", "part", "act"}

// discSeparators is the class the reference collapses to a single space before
// it looks for a prefix `[source: Emby.Naming/Audio/AlbumParser.cs:26,48 @ v10.11.11]`.
const discSeparators = "-.() \t\n\r\v\f"

// parseDiscDirectoryName reads a disc number out of one directory's name.
//
// The rule is the reference's `AlbumParser.IsMultiPart`, in its own order
// `[source: Emby.Naming/Audio/AlbumParser.cs:34-68 @ v10.11.11]`: every run of
// `-`, `.`, `(`, `)` or whitespace becomes a single space, the leading
// whitespace is removed, and the name must then begin with one of
// [discPrefixes] — compared without regard to case — followed by a number.
// Every prefix is tried, because `volume 3` matches `vol` first and the `ume 3`
// left behind is not a number.
//
// **What the reference does with it and what Atrium does with it are different
// things.** There, the answer decides only whether a folder is a *disc folder*
// of a multi-disc album; the disc number itself comes from the tag or the
// container and never from a directory
// `[source: MediaBrowser.Providers/MediaInfo/AudioFileProber.cs:182,312 @ v10.11.11]`.
// Here it is both, and the second half is the declared divergence
// ([behaviours §2.16], OQ-8).
func parseDiscDirectoryName(name string) (int, bool) {
	cleaned := strings.TrimLeft(collapseRuns(name, discSeparators, ' '), " ")
	for _, prefix := range discPrefixes {
		if len(cleaned) < len(prefix) || foldASCIICase(cleaned[:len(prefix)]) != prefix {
			continue
		}
		rest := strings.Trim(cleaned[len(prefix):], " ")
		if end := strings.IndexByte(rest, ' '); end >= 0 {
			rest = rest[:end]
		}
		if number, after, ok := readNumber(rest, 0); ok && after == len(rest) {
			return number, true
		}
	}
	return 0, false
}

// collapseRuns replaces every run of characters in set with one instance of
// replacement.
func collapseRuns(s, set string, replacement byte) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if !strings.ContainsRune(set, rune(s[i])) {
			b.WriteByte(s[i])
			i++
			continue
		}
		b.WriteByte(replacement)
		for i < len(s) && strings.ContainsRune(set, rune(s[i])) {
			i++
		}
	}
	return b.String()
}

// albumNameFromDirectory is a directory's name with the year taken out of it,
// and **nothing else taken out of it**.
//
// Two decisions are in that sentence and both are declared rather than
// inherited:
//
//   - The year comes out, because 003 §3.5's table has a row for
//     `Music/Artist/Album (2001)/…` resolving to *"Album with a year"*. The
//     reference takes nothing out of an album folder's name — its reading of
//     this repository's fixture calls the album `First Album (2001)`
//     `[probe: tools/probe_reference_scan.py, Jellyfin 10.11.11, 2026-09-02]`
//     — so this is a declared difference and it is two rows of 003 plan §8.2's
//     forty-seven.
//   - The **release tags do not**. `names.go`'s vocabulary is a film's, read
//     out of the reference's video cleaner, and running an album's name
//     through it would widen a difference 003 §3.5 asks for by exactly one
//     word into a difference it does not. A trailing whitespace run is
//     removed, which is §3.5's own *"the name derived from a path is trimmed"*.
func albumNameFromDirectory(name string) (string, *int) {
	title, year := takeYear(name)
	title = strings.TrimSpace(title)
	if title == "" {
		return strings.TrimSpace(name), nil
	}
	return title, year
}

// ---------------------------------------------------------------------------
// The three passes
// ---------------------------------------------------------------------------

// musicCandidate is one file, after the classify pass read everything its own
// path and its own tags yield.
type musicCandidate struct {
	root  int
	entry Entry

	// albumDir is the directory the album is named after, and it is the file's
	// containing directory **unless that directory is a disc directory**, in
	// which case it is the one above. It is empty for a file directly under a
	// library root.
	albumDir string

	// artistDir is the directory above albumDir, and is empty when the album's
	// own directory is the first path component — which is the reference's
	// shape too, since a folder holding audio directly under a music root is
	// an album there `[source: Emby.Server.Implementations/Library/Resolvers/Audio/MusicAlbumResolver.cs:128-139 @ v10.11.11]`.
	artistDir string

	albumName  string
	albumYear  *int
	artistName string

	// trackArtist is the track's own artist, and is only ever consulted for an
	// album that nothing else attributes.
	trackArtist string

	title string
	track *int
	disc  *int
}

// classifyMusic is pass 1 for one entry: everything the path yields, with the
// tags laid over it.
//
// **The tags are asked here, once, before any grouping decision** (003 plan
// §6.2). Every field below takes the tag where there is one and the path where
// there is not, which is 003 §3.5's precedence in the one place it can be
// stated once.
func classifyMusic(root int, entry Entry, tags Tags) musicCandidate {
	candidate := musicCandidate{root: root, entry: entry, trackArtist: tags.Artist}

	if dir := dirName(entry.Path); dir != "." {
		candidate.albumDir = dir
		if above := dirName(dir); above != "." {
			if number, ok := parseDiscDirectoryName(baseName(dir)); ok {
				candidate.albumDir = above
				candidate.disc = &number
			}
		}
		if above := dirName(candidate.albumDir); above != "." {
			candidate.artistDir = above
		}
	}

	candidate.title, candidate.track = parseTrackName(stemOfPath(entry.Path))
	if tags.Title != "" {
		candidate.title = tags.Title
	}
	if tags.Track != nil {
		candidate.track = tags.Track
	}
	if tags.Disc != nil {
		candidate.disc = tags.Disc
	}

	switch {
	case tags.Album != "":
		// A tag names the album and carries no directory to take a year out
		// of, so a tagged album has no production year here. 004 owns whether
		// a date tag supplies one.
		candidate.albumName = tags.Album
	case candidate.albumDir != "":
		candidate.albumName, candidate.albumYear = albumNameFromDirectory(baseName(candidate.albumDir))
	}

	switch {
	case tags.AlbumArtist != "":
		candidate.artistName = tags.AlbumArtist
	case candidate.artistDir != "":
		candidate.artistName = strings.TrimSpace(baseName(candidate.artistDir))
	}
	return candidate
}

// albumGroup is one album and the tracks that resolved into it.
type albumGroup struct {
	// artistName is the album artist as the classify pass answered it, and is
	// empty when neither a tag nor a directory supplied one. [resolveMusic]
	// may fill it from the track artists afterwards, and that is the only
	// place it changes.
	artistName string

	name string
	year *int
	path string
	root int

	trackArtists []string
	tracks       []musicCandidate
}

// artistGroup is one album artist.
type artistGroup struct {
	id   string
	name string
	path string
	root int
}

// resolveMusic is 003 plan §6.2's three passes for a `music` library.
//
// # Why the middle pass needs siblings
//
// Because an album's identity comes from its **album artist across all of its
// tracks** (003 §3.5, plan §5). A compilation with a different artist on every
// track is one album, and a resolver answering a path at a time would have to
// discover that by rewriting what it had already returned.
//
// # The one rule that consults the track artists
//
// 003 §3.5: *"Where no album artist is present, the album is attributed to
// `Various Artists` only if the track artists actually differ."* The **only
// if** is what this implements: the track artists decide nothing where a tag
// or a directory already supplied an album artist, and they are consulted only
// to fill a hole neither of those filled. That is the narrower of the two
// readings, and it is the one that keeps §3.5's own precedence intact — a tag
// outranks a directory, and a directory outranks an inference.
//
// Under [NoTags] there are no track artists at all, so this rule cannot fire
// and `Various Artists` is attributed only where a directory names it. That is
// what the fixture's `Various Artists/A Compilation (1999)` gives, and it is
// why a green run over this feature's own tree is **not** evidence for 003
// §3.5's precedence rule.
//
// # Determinism
//
// No map is iterated to produce output. Albums are appended to a slice in the
// sorted entry order and read back out of a map by key, artists are appended
// in album order, and the items are sorted once on the way out of [Resolve].
func resolveMusic(lib ports.Library, readings []Reading, tags TagSource, parentID string) ([]ports.ScannedItem, error) {
	var (
		albums     = map[string]*albumGroup{}
		albumOrder []*albumGroup
		loose      []musicCandidate
	)

	for _, reading := range readings {
		for _, entry := range reading.Entries {
			candidate := classifyMusic(reading.Root, entry, tags.TagsFor(reading.Root, entry.Path))
			if candidate.albumName == "" {
				loose = append(loose, candidate)
				continue
			}

			artistKey, err := Normalise(candidate.artistName, lib.CaseSensitive)
			if err != nil {
				return nil, err
			}
			albumKey, err := Normalise(candidate.albumName, lib.CaseSensitive)
			if err != nil {
				return nil, err
			}

			key := joinKey(artistKey, albumKey)
			group, seen := albums[key]
			if !seen {
				group = &albumGroup{
					artistName: candidate.artistName,
					name:       candidate.albumName,
					year:       candidate.albumYear,
					path:       candidate.albumDir,
					root:       candidate.root,
				}
				albums[key] = group
				albumOrder = append(albumOrder, group)
			}
			group.tracks = append(group.tracks, candidate)
			if candidate.trackArtist != "" {
				group.trackArtists = append(group.trackArtists, candidate.trackArtist)
			}
		}
	}

	// The compilation rule runs after the grouping and never regroups. An
	// album it attributes was grouped under the empty artist, so **two groups
	// can end up deriving one album identifier**: one attributed
	// `Various Artists` from its track artists, and one whose directory
	// already said `Various Artists`, sharing an album name. It is the same
	// shape as the canonically-equal filenames §6.3 records — a repeated
	// identifier in one batch, which `ApplyScanBatch` has to decide the
	// meaning of (T10) and which nothing in `internal/library` can notice.
	// Under [NoTags] it cannot happen at all, because the track artists are
	// what fill the hole and there are none; it is 004's to watch.
	for _, group := range albumOrder {
		if group.artistName == "" {
			group.artistName = attributionFromTrackArtists(group.trackArtists)
		}
	}

	artists := map[string]*artistGroup{}
	var artistOrder []*artistGroup
	for _, group := range albumOrder {
		if group.artistName == "" {
			continue
		}
		key, err := Normalise(group.artistName, lib.CaseSensitive)
		if err != nil {
			return nil, err
		}
		if _, seen := artists[key]; !seen {
			artist := &artistGroup{
				id:   DeriveID(lib.ID, KindMusicArtist, key),
				name: group.artistName,
				root: group.root,
			}
			if group.tracks[0].artistDir != "" {
				artist.path = group.tracks[0].artistDir
			}
			artists[key] = artist
			artistOrder = append(artistOrder, artist)
		}
	}

	items := make([]ports.ScannedItem, 0, len(albumOrder)+len(artistOrder)+len(loose))
	for _, artist := range artistOrder {
		items = append(items, artist.item(lib, parentID))
	}

	for _, group := range albumOrder {
		// An album's identity is its **album artist's identity plus its
		// normalised name** (003 §3.6, amended at this task). Read as "the
		// library root plus the normalised name", which is what the row said
		// before, two artists each holding a `Greatest Hits` are one item:
		// one row, one parent, and half an album's tracks under an artist
		// that did not record them. §3.5 settles it — "an album's identity
		// comes from its album artist" — so this reads as the `Season` row
		// does, a container's identity plus what distinguishes it inside.
		albumParent, artistID := parentID, ""
		if group.artistName != "" {
			key, err := Normalise(group.artistName, lib.CaseSensitive)
			if err != nil {
				return nil, err
			}
			artistID = artists[key].id
			albumParent = artistID
		}

		albumKey, err := Normalise(group.name, lib.CaseSensitive)
		if err != nil {
			return nil, err
		}
		albumID := DeriveID(lib.ID, KindMusicAlbum, joinKey(artistID, albumKey))
		items = append(items, group.item(lib, albumID, albumParent))

		for _, track := range group.tracks {
			item, err := trackItem(lib, track, albumID)
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
	}

	for _, track := range loose {
		item, err := trackItem(lib, track, parentID)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

// attributionFromTrackArtists is 003 §3.5's compilation rule, and it runs only
// where nothing else attributed the album.
//
// One artist throughout is that artist; more than one is [variousArtists]; and
// none at all leaves the album unattributed, hanging from the library's own
// row rather than from an artist invented for it.
func attributionFromTrackArtists(names []string) string {
	first := ""
	for _, name := range names {
		switch {
		case first == "":
			first = name
		case name != first:
			return variousArtists
		}
	}
	return first
}

func (a *artistGroup) item(lib ports.Library, parentID string) ports.ScannedItem {
	item := ports.ScannedItem{
		ID:          a.id,
		LibraryID:   lib.ID,
		ParentID:    parentID,
		Type:        string(KindMusicArtist),
		Name:        a.name,
		Path:        a.path,
		RootOrdinal: a.root,
	}
	item.SortKey = SortKeyFor(&item)
	return item
}

func (g *albumGroup) item(lib ports.Library, id, parentID string) ports.ScannedItem {
	item := ports.ScannedItem{
		ID:             id,
		LibraryID:      lib.ID,
		ParentID:       parentID,
		Type:           string(KindMusicAlbum),
		Name:           g.name,
		Path:           g.path,
		RootOrdinal:    g.root,
		ProductionYear: g.year,
	}
	item.SortKey = SortKeyFor(&item)
	return item
}

// trackItem builds one `Audio` row.
//
// The numbers are set **before** the sort key is derived, and that ordering is
// load-bearing rather than tidy: an `Audio`'s key is its disc, then its track,
// then its name (003 §3.7.2), so a resolver that keyed the item and then
// filled in its numbers would leave every track in the library keyed as though
// it had none.
func trackItem(lib ports.Library, candidate musicCandidate, parentID string) (ports.ScannedItem, error) {
	key, err := Normalise(candidate.entry.Path, lib.CaseSensitive)
	if err != nil {
		return ports.ScannedItem{}, err
	}

	item := ports.ScannedItem{
		ID:                DeriveID(lib.ID, KindAudio, key),
		LibraryID:         lib.ID,
		ParentID:          parentID,
		Type:              string(KindAudio),
		Name:              candidate.title,
		Path:              candidate.entry.Path,
		RootOrdinal:       candidate.root,
		IndexNumber:       candidate.track,
		ParentIndexNumber: candidate.disc,
		Files: []ports.ScannedFile{{
			Ordinal:    0,
			Path:       candidate.entry.Path,
			Size:       candidate.entry.Size,
			ModifiedAt: candidate.entry.ModifiedAt,
		}},
	}
	item.SortKey = SortKeyFor(&item)
	return item, nil
}
