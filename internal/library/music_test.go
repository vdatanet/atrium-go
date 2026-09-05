package library

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/vdatanet/atrium-go/internal/libraryfixture"
	"github.com/vdatanet/atrium-go/internal/ports"
)

// These are `library`-level assertions, the weakest of 003 plan §8.1's three
// levels and the only one this task can reach: [Resolve] is a pure function
// with no route, no store and no wire representation between it and anything a
// client sees.
//
// **Two things in this file are not evidence for what they look like evidence
// for, and both are said here rather than left to be assumed.**
//
// First, 003 plan §8.3 rows 3 and 4. Everything below about an album's tracks
// is about a `ParentID` field and a pair of `*int` fields in a value. That the
// parent reaches a `parent_id` column is T12's, that `/Items?parentId=`
// answers with it is 005's, and that `IndexNumber` and `ParentIndexNumber`
// reach a client **as integers** is 005's too. A green run here is evidence
// for none of them.
//
// Second, and it is the one this task was told to state plainly: **004 carries
// the tag-driven half of 003 §3.5.** This feature ships no metadata reader,
// the fixture's music is generated silence carrying no tag at all, and the
// [TagSource] the tests below hand to [ResolveWithTags] is a stub this file
// wrote. What that stub can prove is that *this resolver* consults the seam
// once per file before grouping and prefers what it says to what the path
// says. What it cannot prove is 003 §3.5's precedence rule — whether a real
// reader over a real library answers those strings — because nothing here
// reads a tag. A green suite is not evidence for that rule and must not be
// read as any.

// aMusicLibrary is the library every test here resolves against. Its identity
// is a literal because it is an **input** to every identifier below.
func aMusicLibrary() ports.Library {
	return ports.Library{
		ID:             "00112233445566778899aabbccddeeff",
		Name:           "Music",
		NameFolded:     "music",
		CollectionType: string(Music),
	}
}

// resolveMusicFiles resolves one root's reading of a `music` library with the
// [NoTags] source v1 ships.
func resolveMusicFiles(t *testing.T, paths ...string) Plan {
	t.Helper()
	plan, err := Resolve(aMusicLibrary(), []Reading{aReading(0, paths...)})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	return plan
}

// stubTags is a [TagSource] that answers from a table and records what it was
// asked. It is this file's own invention: see the package comment above on
// what it can and cannot prove.
type stubTags struct {
	byPath map[string]Tags
	asked  []string
}

func (s *stubTags) TagsFor(_ int, path string) Tags {
	s.asked = append(s.asked, path)
	return s.byPath[path]
}

func intPtr(n int) *int { return &n }

// ---------------------------------------------------------------------------
// AC-8 — a two-disc album
// ---------------------------------------------------------------------------

// TestATwoDiscAlbumIsOneAlbumWhoseTracksCarryTheirDiscNumbers is AC-8.
//
// **The count is asserted first and it is the whole point.** Two albums, one
// per disc directory, each holding a track carrying the right disc number,
// satisfies every per-track assertion anybody would write — and it is the
// failure spec §3.5's table names, *"One album; tracks carry a disc number"*.
// A build that lifts the disc number out of the directory without lifting the
// album with it passes every `ParentIndexNumber` check below and answers a
// client with two albums called `CD1` and `CD2`.
//
// The second disc's track is numbered **5** rather than 1, which is T6's
// handoff finding applied: the fixture's own pair is `CD1/01 - …` and
// `CD2/01 - …`, where the disc directory and the filename's leading number
// agree about the first disc, and a build that read the disc number off the
// filename gets half of it right. Here the two sources disagree on both discs.
func TestATwoDiscAlbumIsOneAlbumWhoseTracksCarryTheirDiscNumbers(t *testing.T) {
	plan := resolveMusicFiles(t,
		"The Artist/Double Album/CD1/03 - First Disc.flac",
		"The Artist/Double Album/CD2/05 - Second Disc.flac",
	)

	albums := itemsOfKind(plan, KindMusicAlbum)
	if len(albums) != 1 {
		t.Fatalf("resolved %d albums from two disc directories, want 1: %q", len(albums), namesOf(albums))
	}
	album := albums[0]
	if got, want := delimiter+album.Name+delimiter, delimiter+"Double Album"+delimiter; got != want {
		t.Errorf("album name = %s, want %s — the disc directory named the album", got, want)
	}
	if album.Path != "The Artist/Double Album" {
		t.Errorf("album path = %q, want the album's directory and not a disc's", album.Path)
	}

	tracks := itemsOfKind(plan, KindAudio)
	if len(tracks) != 2 {
		t.Fatalf("resolved %d tracks, want 2", len(tracks))
	}
	for _, track := range tracks {
		if track.ParentID != album.ID {
			t.Errorf("%q hangs from %q, want the one album %q", track.Path, track.ParentID, album.ID)
		}
	}

	for _, row := range []struct {
		path        string
		disc, track int
	}{
		{"The Artist/Double Album/CD1/03 - First Disc.flac", 1, 3},
		{"The Artist/Double Album/CD2/05 - Second Disc.flac", 2, 5},
	} {
		track := itemAtPath(t, plan, row.path)
		if track.ParentIndexNumber == nil || *track.ParentIndexNumber != row.disc {
			t.Errorf("%q: ParentIndexNumber = %s, want %d", row.path, number(track.ParentIndexNumber), row.disc)
		}
		if track.IndexNumber == nil || *track.IndexNumber != row.track {
			t.Errorf("%q: IndexNumber = %s, want %d — the disc came from the directory and the track from "+
				"the filename, and this pair disagrees so that neither can stand in for the other",
				row.path, number(track.IndexNumber), row.track)
		}
	}
}

// TestADiscDirectoryIsRecognisedByItsOwnVocabulary walks the reference's
// `AlbumStackingPrefixes` and the shapes around them
// `[source: Emby.Naming/Common/NamingOptions.cs:183-193,
// Emby.Naming/Audio/AlbumParser.cs:34-68 @ v10.11.11]`.
//
// The `volume` row is the one that needs every prefix tried rather than the
// first that matches: `Volume 3` begins with `vol`, and the `ume 3` left
// behind is not a number.
func TestADiscDirectoryIsRecognisedByItsOwnVocabulary(t *testing.T) {
	for _, row := range []struct {
		name   string
		number int
		ok     bool
	}{
		{name: "CD1", number: 1, ok: true},
		{name: "cd 2", number: 2, ok: true},
		{name: "Disc 3", number: 3, ok: true},
		{name: "Disk.4", number: 4, ok: true},
		{name: "(Part 5)", number: 5, ok: true},
		{name: "Vol 6", number: 6, ok: true},
		{name: "Volume 7", number: 7, ok: true},
		{name: "Act 8", number: 8, ok: true},
		{name: "Digital Media 9", number: 9, ok: true},
		{name: "CD10", number: 10, ok: true},

		// `dvd` and `pt` stack a *film* and name no disc: the two
		// vocabularies differ in both directions and a shared list would be
		// wrong for both.
		{name: "DVD 1"},
		{name: "pt1"},

		{name: "Bonus"},
		{name: "CD"},
		{name: "CD One"},
		{name: "The CD 1"},
	} {
		got, ok := parseDiscDirectoryName(row.name)
		if ok != row.ok {
			t.Errorf("parseDiscDirectoryName(%q) = (%d, %v), want ok=%v", row.name, got, ok, row.ok)
			continue
		}
		if ok && got != row.number {
			t.Errorf("parseDiscDirectoryName(%q) = %d, want %d", row.name, got, row.number)
		}
	}
}

// TestADiscDirectoryAtTheTopOfALibraryIsAnAlbumAndNotADisc is the guard that
// keeps the rule above from eating a library.
//
// A disc directory is one **below** the album's, so a folder called `CD1`
// directly under a library root is an album in its own right — which is the
// reference's shape too, since a folder holding audio is an album there
// `[source: Emby.Server.Implementations/Library/Resolvers/Audio/MusicAlbumResolver.cs:128-139 @ v10.11.11]`.
// Without the guard the album is lifted to the library root, which has no name
// to give it.
func TestADiscDirectoryAtTheTopOfALibraryIsAnAlbumAndNotADisc(t *testing.T) {
	plan := resolveMusicFiles(t, "CD1/01 - A Track.flac")

	albums := itemsOfKind(plan, KindMusicAlbum)
	if len(albums) != 1 {
		t.Fatalf("resolved %d albums, want 1: %q", len(albums), namesOf(albums))
	}
	if got, want := albums[0].Name, "CD1"; got != want {
		t.Errorf("album name = %q, want %q", got, want)
	}
	if albums[0].ParentID != plan.Items[0].ID {
		t.Error("the album hangs from something other than the library's own row")
	}
	if track := itemAtPath(t, plan, "CD1/01 - A Track.flac"); track.ParentIndexNumber != nil {
		t.Errorf("the track carries disc %s; the directory it sits in is its album, not its disc",
			number(track.ParentIndexNumber))
	}
}

// ---------------------------------------------------------------------------
// AC-9 — a compilation
// ---------------------------------------------------------------------------

// TestACompilationIsOneAlbumAttributedByItsDirectory is AC-9 over the
// fixture's own tree — and **the failure AC-9 is named for cannot be produced
// over that tree**, which is the finding this test carries rather than hides.
//
// AC-9 is *"a compilation with a different artist per track resolves to one
// album"*, and the failure it guards against is one album per track. Under the
// [NoTags] source there is no track artist to differ, so all three files sit in
// one directory and **every** grouping rule a build could have — by directory,
// by album name, by album artist — gives one album. There is no mutation of
// this resolver that makes this tree answer three, so a green run here proves
// the tree and not the rule.
//
// What it does assert is the half that is real without tags: the attribution
// comes from the **directory** (003 plan §6.2, *"with no tag source,
// `Various Artists` is attributed only where the directory names it"*), and the
// three tracks share one album and one artist.
//
// The assertion AC-9 is actually about is the test below this one, which needs
// tags for the artists to differ at all.
func TestACompilationIsOneAlbumAttributedByItsDirectory(t *testing.T) {
	plan := resolveMusicFiles(t,
		"Various Artists/A Compilation (1999)/01 - By One Artist.flac",
		"Various Artists/A Compilation (1999)/02 - By Another.flac",
		"Various Artists/A Compilation (1999)/03 - By A Third.flac",
	)

	albums := itemsOfKind(plan, KindMusicAlbum)
	if len(albums) != 1 {
		t.Fatalf("resolved %d albums, want 1: %q", len(albums), namesOf(albums))
	}
	artists := itemsOfKind(plan, KindMusicArtist)
	if len(artists) != 1 {
		t.Fatalf("resolved %d artists, want 1: %q", len(artists), namesOf(artists))
	}

	if got, want := delimiter+artists[0].Name+delimiter, delimiter+"Various Artists"+delimiter; got != want {
		t.Errorf("artist name = %s, want %s — under the null tag source the directory is what attributes "+
			"the album, and nothing else could have", got, want)
	}
	if got, want := delimiter+albums[0].Name+delimiter, delimiter+"A Compilation"+delimiter; got != want {
		t.Errorf("album name = %s, want %s", got, want)
	}
	if albums[0].ParentID != artists[0].ID {
		t.Error("the album does not hang from the artist the directory named")
	}

	tracks := itemsOfKind(plan, KindAudio)
	if len(tracks) != 3 {
		t.Fatalf("resolved %d tracks, want 3", len(tracks))
	}
	for _, track := range tracks {
		if track.ParentID != albums[0].ID {
			t.Errorf("%q hangs from %q, want the one album", track.Path, track.ParentID)
		}
	}
}

// TestAnAlbumIsGroupedByItsAlbumArtistAndNotByItsTrackArtists is AC-9's actual
// failure, and it needs the seam because the artists have to differ before the
// distinction exists.
//
// 003 §3.5: *"an album's identity comes from its album artist, so a
// compilation with a different artist on every track is one album, not many"*.
// The measurement behind it is 33 albums of 468 holding tracks by more than one
// artist under a single album artist, the largest 60 tracks by 40 artists
// `[probe: tools/probe_music_precedence.py, Jellyfin 10.11.11, 2026-08-27]`.
//
// A build that grouped on the **track** artist answers three albums here. Note
// what this is and is not evidence for: it is evidence about this resolver's
// grouping key, and it is no evidence at all about what a real reader would
// have found in a real file — see the package comment.
func TestAnAlbumIsGroupedByItsAlbumArtistAndNotByItsTrackArtists(t *testing.T) {
	paths := []string{
		"Compilations/A Compilation/01 - One.flac",
		"Compilations/A Compilation/02 - Two.flac",
		"Compilations/A Compilation/03 - Three.flac",
	}
	tags := &stubTags{byPath: map[string]Tags{
		paths[0]: {AlbumArtist: "Various Artists", Artist: "One Singer", Album: "A Compilation"},
		paths[1]: {AlbumArtist: "Various Artists", Artist: "Another Singer", Album: "A Compilation"},
		paths[2]: {AlbumArtist: "Various Artists", Artist: "A Third Singer", Album: "A Compilation"},
	}}

	plan, err := ResolveWithTags(aMusicLibrary(), []Reading{aReading(0, paths...)}, tags)
	if err != nil {
		t.Fatalf("ResolveWithTags: %v", err)
	}

	albums := itemsOfKind(plan, KindMusicAlbum)
	if len(albums) != 1 {
		t.Fatalf("resolved %d albums from three differently-attributed tracks, want 1: %q",
			len(albums), namesOf(albums))
	}
	artists := itemsOfKind(plan, KindMusicArtist)
	if len(artists) != 1 || artists[0].Name != "Various Artists" {
		t.Fatalf("resolved artists %q, want the one album artist", namesOf(artists))
	}
	if albums[0].ParentID != artists[0].ID {
		t.Error("the album does not hang from its album artist")
	}
}

// TestTrackArtistsDecideAnAttributionOnlyWhereNothingElseSuppliedOne is 003
// §3.5's *"only if"*, and it is the clause rather than the sentence.
//
// *"Where no album artist is present, the album is attributed to
// `Various Artists` **only if** the track artists actually differ."* Three
// cases, and the third is the narrowing this project takes: the track artists
// fill a hole and never overrule something that already answered. A tag
// outranks a directory (§3.5), a directory outranks an inference, and a build
// that let differing track artists rename an album whose directory attributed
// it turns an ordinary album with a guest on every track into a compilation.
func TestTrackArtistsDecideAnAttributionOnlyWhereNothingElseSuppliedOne(t *testing.T) {
	t.Run("differing track artists and nothing else attribute the album", func(t *testing.T) {
		paths := []string{"A Compilation/01 - One.flac", "A Compilation/02 - Two.flac"}
		tags := &stubTags{byPath: map[string]Tags{
			paths[0]: {Artist: "One Singer"},
			paths[1]: {Artist: "Another Singer"},
		}}

		plan, err := ResolveWithTags(aMusicLibrary(), []Reading{aReading(0, paths...)}, tags)
		if err != nil {
			t.Fatalf("ResolveWithTags: %v", err)
		}
		artists := itemsOfKind(plan, KindMusicArtist)
		if len(artists) != 1 || artists[0].Name != variousArtists {
			t.Fatalf("artists %q, want the one %q", namesOf(artists), variousArtists)
		}
		if albums := itemsOfKind(plan, KindMusicAlbum); len(albums) != 1 {
			t.Fatalf("resolved %d albums, want 1", len(albums))
		}
	})

	t.Run("one track artist throughout attributes the album to that artist", func(t *testing.T) {
		paths := []string{"An Album/01 - One.flac", "An Album/02 - Two.flac"}
		tags := &stubTags{byPath: map[string]Tags{
			paths[0]: {Artist: "One Singer"},
			paths[1]: {Artist: "One Singer"},
		}}

		plan, err := ResolveWithTags(aMusicLibrary(), []Reading{aReading(0, paths...)}, tags)
		if err != nil {
			t.Fatalf("ResolveWithTags: %v", err)
		}
		artists := itemsOfKind(plan, KindMusicArtist)
		if len(artists) != 1 || artists[0].Name != "One Singer" {
			t.Fatalf("artists %q, want the one shared track artist", namesOf(artists))
		}
	})

	t.Run("a directory that already attributed the album is not overruled", func(t *testing.T) {
		paths := []string{
			"The Artist/An Album/01 - One.flac",
			"The Artist/An Album/02 - Two.flac",
		}
		tags := &stubTags{byPath: map[string]Tags{
			paths[0]: {Artist: "The Artist"},
			paths[1]: {Artist: "The Artist feat. A Guest"},
		}}

		plan, err := ResolveWithTags(aMusicLibrary(), []Reading{aReading(0, paths...)}, tags)
		if err != nil {
			t.Fatalf("ResolveWithTags: %v", err)
		}
		artists := itemsOfKind(plan, KindMusicArtist)
		if len(artists) != 1 || artists[0].Name != "The Artist" {
			t.Fatalf("artists %q, want the directory's %q — differing track artists fill a hole and "+
				"never overrule an attribution that already exists", namesOf(artists), "The Artist")
		}
	})
}

// ---------------------------------------------------------------------------
// The seam 004 fills
// ---------------------------------------------------------------------------

// TestTheTagSourceIsAskedOncePerFileAndBeforeGrouping is 003 plan §6.2's
// ordering clause, asserted as two separate things because a build can have
// either without the other.
//
// **Once per file**: the recorded calls are exactly the candidate paths, each
// once. A source asked twice is a file opened twice per scan, and a source
// asked once per *album* cannot answer the question that decides which album a
// track belongs to.
//
// **Before grouping**: the second file's album tag disagrees with its
// directory, and the tag is what decides which album it lands in. A resolver
// that grouped on the path and then laid the tags over the result gives one
// album with two tracks and passes any assertion made about a field.
func TestTheTagSourceIsAskedOncePerFileAndBeforeGrouping(t *testing.T) {
	paths := []string{
		"The Artist/One Directory/01 - One.flac",
		"The Artist/One Directory/02 - Two.flac",
	}
	tags := &stubTags{byPath: map[string]Tags{
		paths[1]: {Album: "A Different Album"},
	}}

	plan, err := ResolveWithTags(aMusicLibrary(), []Reading{aReading(0, paths...)}, tags)
	if err != nil {
		t.Fatalf("ResolveWithTags: %v", err)
	}

	if !reflect.DeepEqual(tags.asked, paths) {
		t.Errorf("the source was asked for %q, want each candidate exactly once in path order: %q",
			tags.asked, paths)
	}

	albums := itemsOfKind(plan, KindMusicAlbum)
	if len(albums) != 2 {
		t.Fatalf("resolved %d albums, want 2 — the tag decided which album the second file belongs to, "+
			"and one directory holds them both: %q", len(albums), namesOf(albums))
	}
	if got := namesOf(albums); !reflect.DeepEqual(got, []string{"A Different Album", "One Directory"}) &&
		!reflect.DeepEqual(got, []string{"One Directory", "A Different Album"}) {
		t.Errorf("album names %q, want the directory's and the tag's", got)
	}
}

// TestATagOutranksThePathForEveryFieldItAnswers is 003 §3.5's inversion —
// *"embedded tags outrank the path"* — asserted field by field, because a
// build can prefer the tag for one and the path for another and nothing about
// the shape of the code says which.
//
// The path here says everything and every tag contradicts it, so a field that
// came from the path is visible as itself rather than as an absence.
func TestATagOutranksThePathForEveryFieldItAnswers(t *testing.T) {
	const path = "The Path Artist/The Path Album (1999)/CD2/07 - The Path Title.flac"
	tags := &stubTags{byPath: map[string]Tags{
		path: {
			AlbumArtist: "The Tag Artist",
			Album:       "The Tag Album",
			Title:       "The Tag Title",
			Track:       intPtr(3),
			Disc:        intPtr(4),
		},
	}}

	plan, err := ResolveWithTags(aMusicLibrary(), []Reading{aReading(0, path)}, tags)
	if err != nil {
		t.Fatalf("ResolveWithTags: %v", err)
	}

	artists := itemsOfKind(plan, KindMusicArtist)
	if len(artists) != 1 || artists[0].Name != "The Tag Artist" {
		t.Errorf("artists %q, want the tag's", namesOf(artists))
	}
	albums := itemsOfKind(plan, KindMusicAlbum)
	if len(albums) != 1 || albums[0].Name != "The Tag Album" {
		t.Fatalf("albums %q, want the tag's", namesOf(albums))
	}
	if albums[0].ProductionYear != nil {
		t.Errorf("the tag-named album carries production year %d; the year came out of a directory the "+
			"tag replaced, and 004 owns whether a date tag supplies one", *albums[0].ProductionYear)
	}

	track := itemAtPath(t, plan, path)
	if track.Name != "The Tag Title" {
		t.Errorf("track name = %q, want the tag's", track.Name)
	}
	if track.IndexNumber == nil || *track.IndexNumber != 3 {
		t.Errorf("IndexNumber = %s, want the tag's 3 and not the filename's 7", number(track.IndexNumber))
	}
	if track.ParentIndexNumber == nil || *track.ParentIndexNumber != 4 {
		t.Errorf("ParentIndexNumber = %s, want the tag's 4 and not the directory's 2",
			number(track.ParentIndexNumber))
	}
}

// ---------------------------------------------------------------------------
// The fallback, and the divergence it is
// ---------------------------------------------------------------------------

// TestALeadingNumberIsATrackNumberOnlyWhenASeparatorFollowsIt is 003 §3.5's
// tie-break, asserted as the **narrowing** it is.
//
// The direction is the reasoning: the reference takes a track number off no
// filename at all, so every stem Atrium declines to find a number in is a stem
// it **agrees** with the reference about, and an ambiguous shape is therefore
// read as saying *less*. Each row below says which of the two it is, and the
// `agrees` column is not decoration — it is the property that makes the
// tie-break the safe direction rather than merely a choice.
func TestALeadingNumberIsATrackNumberOnlyWhenASeparatorFollowsIt(t *testing.T) {
	for _, row := range []struct {
		stem   string
		name   string
		track  *int
		agrees bool
	}{
		{stem: "01 - Opening", name: "Opening", track: intPtr(1)},
		{stem: "02 Second", name: "Second", track: intPtr(2)},
		{stem: "03.Third", name: "Third", track: intPtr(3)},
		{stem: "04_Fourth", name: "Fourth", track: intPtr(4)},

		// The three the tie-break exists for. Each keeps its whole stem, which
		// is exactly the name the reference gives it.
		{stem: "24K Magic", name: "24K Magic", agrees: true},
		{stem: "9f86d081884c7d659a2feaa0c55ad015", name: "9f86d081884c7d659a2feaa0c55ad015", agrees: true},
		{stem: "1917", name: "1917", agrees: true},
		{stem: "07 - ", name: "07 -", agrees: false},
	} {
		name, track := parseTrackName(row.stem)

		if delimiter+name+delimiter != delimiter+row.name+delimiter {
			t.Errorf("parseTrackName(%q) named it |%s|, want |%s|", row.stem, name, row.name)
		}
		switch {
		case row.track == nil && track != nil:
			t.Errorf("parseTrackName(%q) found track %d; an ambiguous name is read as saying less", row.stem, *track)
		case row.track != nil && (track == nil || *track != *row.track):
			t.Errorf("parseTrackName(%q) found track %s, want %d", row.stem, number(track), *row.track)
		}

		if row.agrees && name != row.stem {
			t.Errorf("parseTrackName(%q) changed a name it declines to find a number in, so it no longer "+
				"agrees with the reference about it", row.stem)
		}
	}
}

// TestAnAlbumsNameLosesItsYearAndNothingElse is the second half of a decision
// whose first half is asserted everywhere and whose second half was asserted
// nowhere until a mutation said so.
//
// The year comes out, because 003 §3.5's table has a row for
// `Music/Artist/Album (2001)/…` resolving to *"Album with a year"*. The film
// **release tags do not**: `names.go`'s vocabulary is read out of the
// reference's *video* cleaner, and the reference takes nothing at all out of an
// album folder's name. Running an album's name through it would widen a
// difference §3.5 asks for by exactly one word into a difference it does not —
// and every album directory in the fixture tree survives that cleaner
// unchanged, so the mutation is invisible in a green run over it.
func TestAnAlbumsNameLosesItsYearAndNothingElse(t *testing.T) {
	for _, row := range []struct {
		directory string
		name      string
		year      *int
	}{
		{directory: "First Album (2001)", name: "First Album", year: intPtr(2001)},
		{directory: "Double Album", name: "Double Album"},
		{directory: "  Padded Album  ", name: "Padded Album"},

		// Each of these is a token the film cleaner cuts a title at. An album
		// is not a film and keeps them.
		{directory: "An Album - DVDRip", name: "An Album - DVDRip"},
		{directory: "Live 1080p", name: "Live 1080p"},
		{directory: "Sessions [Remastered]", name: "Sessions [Remastered]"},
		{directory: "spandau_ballet-through_the_barricades", name: "spandau_ballet-through_the_barricades"},
	} {
		name, year := albumNameFromDirectory(row.directory)

		if delimiter+name+delimiter != delimiter+row.name+delimiter {
			t.Errorf("albumNameFromDirectory(%q) named it |%s|, want |%s|", row.directory, name, row.name)
		}
		switch {
		case row.year == nil && year != nil:
			t.Errorf("albumNameFromDirectory(%q) found year %d, want none", row.directory, *year)
		case row.year != nil && (year == nil || *year != *row.year):
			t.Errorf("albumNameFromDirectory(%q) found year %s, want %d", row.directory, number(year), *row.year)
		}
	}
}

// TestAnArtistAndAnAlbumAreKeyedThroughNormalise is T3's finding applied to the
// two names this resolver derives an identifier from.
//
// `DeriveID` does not normalise and cannot: whether case is folded is a
// property of the **library** and only the caller holds it. A resolver that
// passed a raw name would derive, in the default case-insensitive library, an
// identifier that nothing will ever derive again the day an operator fixes the
// capitalisation of a directory — and no test in `identity.go` could see it,
// because the function has no way to tell.
func TestAnArtistAndAnAlbumAreKeyedThroughNormalise(t *testing.T) {
	t.Run("a case-insensitive library sees one artist and one album", func(t *testing.T) {
		plan := resolveMusicFiles(t,
			"The Artist/An Album/01 - One.flac",
			"THE ARTIST/AN ALBUM/02 - Two.flac",
		)
		if got := itemsOfKind(plan, KindMusicArtist); len(got) != 1 {
			t.Errorf("resolved %d artists, want 1 — case is folded by default (003 §3.6): %q",
				len(got), namesOf(got))
		}
		if got := itemsOfKind(plan, KindMusicAlbum); len(got) != 1 {
			t.Errorf("resolved %d albums, want 1: %q", len(got), namesOf(got))
		}
	})

	t.Run("a case-sensitive library sees two of each", func(t *testing.T) {
		lib := aMusicLibrary()
		lib.CaseSensitive = true

		plan, err := Resolve(lib, []Reading{aReading(0,
			"The Artist/An Album/01 - One.flac",
			"THE ARTIST/AN ALBUM/02 - Two.flac",
		)})
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if got := itemsOfKind(plan, KindMusicArtist); len(got) != 2 {
			t.Errorf("resolved %d artists, want 2 — the library was declared case-sensitive: %q",
				len(got), namesOf(got))
		}
		if got := itemsOfKind(plan, KindMusicAlbum); len(got) != 2 {
			t.Errorf("resolved %d albums, want 2: %q", len(got), namesOf(got))
		}
	})
}

// referenceNamesInTheFixturesMusicLibrary are what the reference's own reading
// of this repository's fixture tree calls three of its music items
// `[probe: tools/probe_reference_scan.py, Jellyfin 10.11.11, 2026-09-02]`.
//
// They are literals here rather than a read of
// `docs/compatibility/reference-fixture-reading.json`, for the reason
// `resolve_test.go` gives: what the test below asserts is a **declared
// inequality**, and a declaration that reads itself out of the thing it is
// declared against cannot be one.
const (
	referenceNameOfATrack                = "01 - Opening"
	referenceNameOfAnAlbum               = "First Album (2001)"
	referenceTypeOfADiscFolder           = "Folder"
	referenceNameOfADiscFolder           = "CD1"
	referencePathOfADiscFolder           = "The Artist/Double Album/CD1"
	referenceReadsNoTrackNumberFromAName = "the reference takes an Audio's number from the tag or the container and " +
		"from nowhere else (behaviours §2.16)"
)

// TestTheReferenceTakesNoneOfTheThreeFromAPath is 003 §3.5's fallback asserted
// as the **divergence** it is, and it is written so that the day OQ-8 is
// answered a failing test is the notification rather than a rediscovery.
//
// The reference takes a track number, a disc number and a title from the
// embedded tag or from the number the container carries, and from **nowhere
// else**; a file whose tags supply no title keeps its whole filename stem,
// leading digits included
// `[source: MediaBrowser.Providers/MediaInfo/AudioFileProber.cs:181-182,312 @ v10.11.11]`
// `[source: Emby.Server.Implementations/Library/ResolverHelper.cs:96 @ v10.11.11]`.
// Atrium reads all three off the path when nothing else supplies them
// ([behaviours §2.16], 003 §3.5), and **OQ-8 holds the decision open**: the
// evidence that would settle it is how much real music carries no readable tag,
// which needs a library this suite does not have.
//
// So every assertion here has two halves. Atrium's answer must **differ** from
// the reference's, and it must differ by exactly the rule 003 states — the
// reference's own name run through Atrium's own derivation must equal Atrium's
// answer. Either half moving fails it: the day the fallback is removed the
// first half goes red, and the day the rule changes shape the second does.
func TestTheReferenceTakesNoneOfTheThreeFromAPath(t *testing.T) {
	plan := resolveMusicFiles(t,
		"The Artist/First Album (2001)/01 - Opening.flac",
		"The Artist/Double Album/CD1/01 - First Disc.flac",
	)

	t.Run("a track's name and its number", func(t *testing.T) {
		track := itemAtPath(t, plan, "The Artist/First Album (2001)/01 - Opening.flac")

		if track.Name == referenceNameOfATrack {
			t.Fatalf("the track is now named %q, which is what the reference names it; the declared "+
				"difference has gone away, and that fails as loudly as an undeclared one (plan §8.2)",
				referenceNameOfATrack)
		}
		cleaned, trackNumber := parseTrackName(referenceNameOfATrack)
		if cleaned != track.Name {
			t.Errorf("the reference's name through Atrium's own rule is %q and Atrium's is %q; the two "+
				"servers are no longer naming this track from the same string", cleaned, track.Name)
		}
		if trackNumber == nil || track.IndexNumber == nil || *trackNumber != *track.IndexNumber {
			t.Errorf("track number from the reference's own name = %s, Atrium's = %s",
				number(trackNumber), number(track.IndexNumber))
		}
		if track.IndexNumber == nil {
			t.Errorf("Atrium found no track number, so there is nothing to diverge about: %s",
				referenceReadsNoTrackNumberFromAName)
		}
	})

	t.Run("an album's name and its year", func(t *testing.T) {
		album := itemAtPath(t, plan, "The Artist/First Album (2001)")

		if album.Name == referenceNameOfAnAlbum {
			t.Fatalf("the album is now named %q, which is the reference's own name; the declared "+
				"difference has gone away", referenceNameOfAnAlbum)
		}
		cleaned, year := albumNameFromDirectory(referenceNameOfAnAlbum)
		if cleaned != album.Name {
			t.Errorf("the reference's name through Atrium's own rule is %q and Atrium's is %q", cleaned, album.Name)
		}
		if year == nil || album.ProductionYear == nil || *year != *album.ProductionYear {
			t.Errorf("the year the reference's name carries is %s and Atrium's is %s",
				number(year), number(album.ProductionYear))
		}
	})

	t.Run("a disc directory is an item there and none here", func(t *testing.T) {
		for _, item := range plan.Items {
			if item.Path == referencePathOfADiscFolder {
				t.Fatalf("Atrium made a %s called %q of the disc directory; the reference makes a %q "+
					"called %q of it and Atrium makes nothing, which is a declared difference and not "+
					"a gap to close", item.Type, item.Name, referenceTypeOfADiscFolder, referenceNameOfADiscFolder)
			}
		}

		// And the tracks under it are in the album rather than orphaned by the
		// directory having no item.
		track := itemAtPath(t, plan, "The Artist/Double Album/CD1/01 - First Disc.flac")
		album := itemAtPath(t, plan, "The Artist/Double Album")
		if track.ParentID != album.ID {
			t.Errorf("the disc's track hangs from %q, want the album %q", track.ParentID, album.ID)
		}
	})
}

// ---------------------------------------------------------------------------
// Identity and structure
// ---------------------------------------------------------------------------

// TestAnAlbumsIdentityIsItsArtistsIdentityPlusItsName is the amendment this
// task made to spec §3.6, asserted at the one place a build gets it wrong
// silently.
//
// §3.6's table read on its own puts `MusicAlbum` with `Series` and
// `MusicArtist` under *"the library root plus the normalised name"*, and
// implemented that way **two artists' `Greatest Hits` are one item**: one album
// row, one parent, and half of its tracks appearing under an artist that did
// not record them. §3.5 is the section that settles it — *"an album's identity
// comes from its album artist"* — and the table now says so, exactly as the
// `Season` row already says a season's identity is its series' plus its number.
//
// Nothing in the fixture tree exercises this: its five album names are
// distinct, so the merge is invisible in a green run over it.
func TestAnAlbumsIdentityIsItsArtistsIdentityPlusItsName(t *testing.T) {
	plan := resolveMusicFiles(t,
		"One Artist/Greatest Hits/01 - One.flac",
		"Another Artist/Greatest Hits/01 - Two.flac",
	)

	albums := itemsOfKind(plan, KindMusicAlbum)
	if len(albums) != 2 {
		t.Fatalf("resolved %d albums called `Greatest Hits`, want 2 — one per artist: %q",
			len(albums), namesOf(albums))
	}
	if albums[0].ID == albums[1].ID {
		t.Fatal("the two albums share an identifier, so they are one item however they are counted")
	}

	artists := itemsOfKind(plan, KindMusicArtist)
	if len(artists) != 2 {
		t.Fatalf("resolved %d artists, want 2: %q", len(artists), namesOf(artists))
	}
	parents := map[string]bool{albums[0].ParentID: true, albums[1].ParentID: true}
	if len(parents) != 2 {
		t.Error("both albums hang from the same artist")
	}

	// And the identity is derived rather than allocated: the same name under
	// the same artist derives the same identifier whatever else moved.
	moved := resolveMusicFiles(t,
		"One Artist/Greatest Hits/01 - One.flac",
		"Another Artist/Greatest Hits/01 - Two.flac",
		"A Third Artist/Something Else/01 - Three.flac",
	)
	for _, album := range itemsOfKind(moved, KindMusicAlbum) {
		if album.Path != albums[0].Path {
			continue
		}
		if album.ID != albums[0].ID {
			t.Errorf("the album at %q changed identifier when an unrelated artist appeared", album.Path)
		}
	}
}

// TestAnArtistsIdentityIsItsNameAndNothingElse is §3.6's *"the library root
// plus the normalised name"* for the one type whose row this task did **not**
// amend, and it is here because a mutation said nothing was asserting it.
//
// An artist that is keyed on anything an album contributes — the album's name,
// its directory, the count of them — has an identifier that moves when a
// record is added to the library or taken out of it. That silently discards
// every favourite and every resume position under that artist (§3.6:
// *"Stability is the whole requirement"*), and it is invisible in any test that
// only counts artists or reads their names.
//
// So the assertion is that the identifier does not move, asserted across a
// library that gained an album and one that lost one, and pinned against the
// derivation as well — because a build whose artist identifiers are stable and
// derived from the wrong string passes the first half.
func TestAnArtistsIdentityIsItsNameAndNothingElse(t *testing.T) {
	one := resolveMusicFiles(t, "The Artist/An Album/01 - One.flac")
	two := resolveMusicFiles(t,
		"The Artist/An Album/01 - One.flac",
		"The Artist/A Different Album/01 - Two.flac",
	)

	first := itemsOfKind(one, KindMusicArtist)
	second := itemsOfKind(two, KindMusicArtist)
	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("resolved %d and %d artists, want 1 each", len(first), len(second))
	}
	if first[0].ID != second[0].ID {
		t.Errorf("the artist's identifier moved from %q to %q when a second album appeared beside it; "+
			"every favourite and every resume position under it is discarded by that",
			first[0].ID, second[0].ID)
	}

	key, err := Normalise("The Artist", aMusicLibrary().CaseSensitive)
	if err != nil {
		t.Fatalf("Normalise: %v", err)
	}
	if want := DeriveID(aMusicLibrary().ID, KindMusicArtist, key); first[0].ID != want {
		t.Errorf("identifier = %q, want %q — the library's identity plus the normalised name (§3.6)",
			first[0].ID, want)
	}
}

// TestATrackWithNoAlbumHangsFromTheLibraryAndIsNotUnplaceable is the case 003
// §3.5's table does not have a row for, decided here and declared rather than
// left to a reader.
//
// A candidate directly under a library root has no album directory and no
// artist directory. It is still a complete item — a track's name never has to
// say anything for the track to exist — so it hangs from the library's own row
// and is **not** unplaceable. That is the opposite answer to `tvshows`, where a
// file whose name says which episode it is not is exactly what
// [ReasonNoEpisodeNumber] counts, and the two differ because there the name had
// a job to do and here it does not.
func TestATrackWithNoAlbumHangsFromTheLibraryAndIsNotUnplaceable(t *testing.T) {
	plan := resolveMusicFiles(t, "01 - Loose.flac")

	if len(plan.Unplaceable) != 0 {
		t.Errorf("unplaceable = %+v, want none: nothing under `music` produces one", plan.Unplaceable)
	}
	if got := itemsOfKind(plan, KindMusicAlbum); len(got) != 0 {
		t.Errorf("resolved %d albums from a file with no directory, want none: %q", len(got), namesOf(got))
	}
	if got := itemsOfKind(plan, KindMusicArtist); len(got) != 0 {
		t.Errorf("resolved %d artists, want none — a library root is not an artist: %q", len(got), namesOf(got))
	}

	track := itemAtPath(t, plan, "01 - Loose.flac")
	if track.ParentID != plan.Items[0].ID {
		t.Errorf("the track hangs from %q, want the library's own row %q", track.ParentID, plan.Items[0].ID)
	}
	if track.Unplaceable {
		t.Error("the track is marked unplaceable; its name said everything a track's name has to say")
	}
	if track.IndexNumber == nil || *track.IndexNumber != 1 {
		t.Errorf("IndexNumber = %s, want 1 — the fallback runs wherever the file sits", number(track.IndexNumber))
	}
}

// TestAFolderHoldingAudioDirectlyUnderARootIsAnAlbum is the middle case
// between the two above, and it is the reference's own shape: a folder holding
// audio is an album there
// `[source: Emby.Server.Implementations/Library/Resolvers/Audio/MusicAlbumResolver.cs:128-139 @ v10.11.11]`,
// so a library one directory deep resolves to albums with no artist rather than
// to artists with no album.
func TestAFolderHoldingAudioDirectlyUnderARootIsAnAlbum(t *testing.T) {
	plan := resolveMusicFiles(t, "Some Folder/01 - One.flac")

	albums := itemsOfKind(plan, KindMusicAlbum)
	if len(albums) != 1 || albums[0].Name != "Some Folder" {
		t.Fatalf("albums %q, want the one folder", namesOf(albums))
	}
	if got := itemsOfKind(plan, KindMusicArtist); len(got) != 0 {
		t.Errorf("resolved %d artists, want none: %q", len(got), namesOf(got))
	}
	if albums[0].ParentID != plan.Items[0].ID {
		t.Error("the album does not hang from the library's own row")
	}
}

// TestAnAudiosSortKeyIsDiscThenTrackThenName is 003 §3.7.2 at the one place
// this resolver can get it wrong invisibly: the numbers have to be on the item
// **before** the key is derived.
//
// behaviours §2.6 names the failure — a track keyed by the base derivation
// sorts `The Song` under `s`, which reorders every album in the library and
// which no client can recognise as wrong. A resolver that keyed the item and
// then filled in its numbers leaves every track keyed as though it had none,
// and every count and every name in this file stays green.
func TestAnAudiosSortKeyIsDiscThenTrackThenName(t *testing.T) {
	plan := resolveMusicFiles(t, "The Artist/Double Album/CD2/05 - The Song.flac")

	track := itemAtPath(t, plan, "The Artist/Double Album/CD2/05 - The Song.flac")
	if got, want := track.SortKey, SortKeyFor(&track); got != want {
		t.Fatalf("sort key = %q, want %q", got, want)
	}
	if !strings.HasPrefix(track.SortKey, "0002") {
		t.Errorf("sort key = %q, want it to begin with the disc number — the numbers were not on the item "+
			"when the key was derived", track.SortKey)
	}
	if track.SortKey == SortKeyBase(track.Name) {
		t.Errorf("sort key = %q, which is the base derivation: the track carries no numbering in its key "+
			"and sorts by its title alone (behaviours §2.6)", track.SortKey)
	}
}

// ---------------------------------------------------------------------------
// The fixture
// ---------------------------------------------------------------------------

// TestTheFixturesMusicResolvesToTheExpectedRows compares this resolver against
// the item set 003 plan §8.5 keeps as a **literal**.
//
// It is compared as a **set** and not positionally, for the reason
// `shows_test.go` gives: the literal is grouped by type and [Resolve] orders by
// path. The reading is a literal too — deriving it from
// `libraryfixture.Libraries()` would make this a test of arithmetic — and it is
// not a scan: the fixture's `Music` library holds only candidates, but which
// files a walk hands over is T8's and not this task's.
func TestTheFixturesMusicResolvesToTheExpectedRows(t *testing.T) {
	plan := resolveMusicFiles(t,
		"The Artist/Double Album/CD1/01 - First Disc.flac",
		"The Artist/Double Album/CD2/01 - Second Disc.flac",
		"The Artist/First Album (2001)/01 - Opening.flac",
		"The Artist/First Album (2001)/02 - Second.flac",
		"The Artist/Second Album/01 - In Another Container.m4a",
		"The Artist/Second Album/02 - And Another.dsf",
		"The Artist/spandau_ballet-through_the_barricades/01 - Tagged Differently.flac",
		"Various Artists/A Compilation (1999)/01 - By One Artist.flac",
		"Various Artists/A Compilation (1999)/02 - By Another.flac",
		"Various Artists/A Compilation (1999)/03 - By A Third.flac",
	)

	var want []libraryfixture.ExpectedItem
	for _, row := range libraryfixture.ExpectedItems() {
		if row.Library == "Music" {
			want = append(want, row)
		}
	}
	if len(plan.Items) != len(want) {
		t.Fatalf("resolved %d items, want %d:\n got  %q\n want %q",
			len(plan.Items), len(want), describeItems(plan.Items), describeExpected(want))
	}

	got := describeItems(plan.Items)
	sort.Strings(got)
	expected := describeExpected(want)
	sort.Strings(expected)
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("row %d:\n got  %s\n want %s", i, got[i], expected[i])
		}
	}

	// The parent of every expected row is named by its **path**, which is what
	// makes "the expected parent-child structure" an assertion about the tree
	// rather than about a string.
	byPath := map[string]ports.ScannedItem{}
	for _, item := range plan.Items {
		if Kind(item.Type) != KindCollectionFolder {
			byPath[item.Path] = item
		}
	}
	for _, row := range want {
		if row.Parent == "" {
			continue
		}
		item, ok := byPath[row.Path]
		if !ok {
			t.Errorf("nothing resolved to %q", row.Path)
			continue
		}
		wantParent := plan.Items[0].ID
		if row.Parent != libraryfixture.LibraryRoot {
			parent, ok := byPath[row.Parent]
			if !ok {
				t.Errorf("%q expects parent %q and nothing resolved to that path", row.Path, row.Parent)
				continue
			}
			wantParent = parent.ID
		}
		if item.ParentID != wantParent {
			t.Errorf("%q: parent %q, want %q (%q)", row.Path, item.ParentID, wantParent, row.Parent)
		}
	}
}

// itemAtPath is the one item a plan holds at a path, and it fails the test
// rather than returning a zero value nothing would notice.
func itemAtPath(t *testing.T, plan Plan, path string) ports.ScannedItem {
	t.Helper()
	var found []ports.ScannedItem
	for _, item := range plan.Items {
		if item.Path == path {
			found = append(found, item)
		}
	}
	if len(found) != 1 {
		t.Fatalf("the plan holds %d items at %q, want exactly 1", len(found), path)
	}
	return found[0]
}
