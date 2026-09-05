package libraryfixture

// This is the third file, and it is separate on purpose.
//
// 003 plan §8.5: the fixture "must not be scanned by a test that then asserts
// a count it computed from the same declaration. Two of spec §5's criteria are
// counts, and a count derived from the builder is a test of nothing." So the
// set of items a scan of this tree must produce is written out here as a
// literal, reachable without calling the builder — a change to the tree is
// then a change to two files and a reviewer sees both.
//
// # What is asserted here and what is not
//
// A row carries the four things AC-1 is about and 003's comparison against the
// reference's reading is over: the library, the item's type, its name and its
// path, plus the parent that makes "the expected parent-child structure" a
// statement rather than a hope. The numbers — index, parent index, index end,
// production year — are not here: they are asserted by T5, T6 and T7 at the
// resolver, in tables of their own, and a second copy of them in this file
// would be a second thing to keep in step for no assertion that is not already
// made.
//
// # Where these names come from, since some of them are decisions
//
// Every row is derived from 003's specification, never from the reference's
// reading of the same tree — the two disagree in declared places and T17 is
// where that disagreement is written down. Four derivations are worth stating
// because a reader will otherwise assume they were copied from somewhere:
//
//   - A film's name is its title with the year taken out (§3.3), so the
//     folder-per-film rows are "The Matrix" and "The Long Film" and not the
//     directory names the reference kept whole.
//
//   - A name derived from a path is trimmed (§3.5's own words), so the padded
//     film is "Padded".
//
//   - An episode's name is what follows the numbering in the filename; where
//     nothing follows it, the whole stem stays. That is what leaves the daily
//     show's episode named after its own file and every other episode named
//     after its title. §3.4 does not state this rule, T6 owns it, and this is
//     the file where changing it is visible.
//
//   - A track's name is what the filename says once the leading track number
//     is taken off it (§3.5's fallback, with the tie-break that reads an
//     ambiguous name as saying less). That is a *divergence*: the reference
//     names an untagged file after its whole name, leading digits included
//     (behaviours §2.16, OQ-8), so these ten rows differ from the reading by
//     design.
//
// # What the tree holds that no row here names
//
// Ten files. The zero-byte incomplete copy and everything under an exclusion
// rule (§3.2), plus the second part of the multi-part film, which is a second
// file of the one item "The Long Film" rather than an item of its own. Each of
// them carries a Why in the declaration saying which rule drops it.

// LibraryRoot is the Parent of an item that sits directly under a library's
// own row, and it is not a path: the library's row has no path of its own,
// because a library may be configured with several roots (003 §4.2).
const LibraryRoot = "."

// ExpectedItem is one row of the set a scan of the built tree must produce.
type ExpectedItem struct {
	Library string

	// Type is one of 003 plan §4.2's eight: CollectionFolder, Movie, Series,
	// Season, Episode, MusicArtist, MusicAlbum, Audio.
	Type string

	Name string

	// Path is relative to the library root, slash-separated. It is empty for
	// the library's own row, and for that row only: every container in this
	// tree is backed by a directory.
	Path string

	// Parent is the parent's Path, LibraryRoot for an item directly under the
	// library's own row, and empty for that row itself.
	Parent string

	// Unplaceable marks the one item whose name says too little to place it.
	// 003 §3.8 counts those apart from skipped files, "because an operator
	// told that both were skipped would go looking for something that is not
	// missing".
	Unplaceable bool
}

// ExpectedItems is the set, in a stable order. It is a literal and it calls
// nothing.
func ExpectedItems() []ExpectedItem {
	return []ExpectedItem{
		// ---- Movies: 003 §3.3 ------------------------------------------------
		{Library: "Movies", Type: "CollectionFolder", Name: "Movies"},
		{Library: "Movies", Type: "Movie", Name: "Padded", Path: "  Padded   (1999).mkv", Parent: LibraryRoot},
		{Library: "Movies", Type: "Movie", Name: "10 Things I Hate About You", Path: "10 Things I Hate About You (1999).mkv", Parent: LibraryRoot},
		{Library: "Movies", Type: "Movie", Name: "100% Wolf", Path: "100% Wolf (2020).mkv", Parent: LibraryRoot},
		{Library: "Movies", Type: "Movie", Name: "2 Fast 2 Furious", Path: "2 Fast 2 Furious (2003).mkv", Parent: LibraryRoot},
		{Library: "Movies", Type: "Movie", Name: "A Bridge Too Far", Path: "A Bridge Too Far (1977).mkv", Parent: LibraryRoot},
		{Library: "Movies", Type: "Movie", Name: "A Broadcast Capture", Path: "A Broadcast Capture (2011).ts", Parent: LibraryRoot},
		{Library: "Movies", Type: "Movie", Name: "A Newer Transfer", Path: "A Newer Transfer (2015).mp4", Parent: LibraryRoot},
		{Library: "Movies", Type: "Movie", Name: "Amélie", Path: "Amélie (2001).mkv", Parent: LibraryRoot},
		{Library: "Movies", Type: "Movie", Name: "An Old Transfer", Path: "An Old Transfer (1985).avi", Parent: LibraryRoot},
		{Library: "Movies", Type: "Movie", Name: "Don't Look Up", Path: "Don't Look Up (2021).mkv", Parent: LibraryRoot},
		{Library: "Movies", Type: "Movie", Name: "Rock & Roll", Path: "Rock & Roll (1978).mkv", Parent: LibraryRoot},
		{Library: "Movies", Type: "Movie", Name: "S.W.A.T.", Path: "S.W.A.T. (2003).mkv", Parent: LibraryRoot},
		// One item, two parts. Its path is the first part's, which is the path
		// its identity derives from (§3.6), and its name comes from the
		// directory because the directory holds exactly one film (§3.3).
		{Library: "Movies", Type: "Movie", Name: "The Long Film", Path: "The Long Film (1998)/The Long Film (1998) - part1.mkv", Parent: LibraryRoot},
		{Library: "Movies", Type: "Movie", Name: "The Matrix", Path: "The Matrix (1999)/The Matrix (1999).mkv", Parent: LibraryRoot},
		{Library: "Movies", Type: "Movie", Name: "Wall-E", Path: "Wall-E (2008).mkv", Parent: LibraryRoot},
		{Library: "Movies", Type: "Movie", Name: "iRobot", Path: "iRobot (2004).mkv", Parent: LibraryRoot},

		// ---- Shows: 003 §3.4 -------------------------------------------------
		{Library: "Shows", Type: "CollectionFolder", Name: "Shows"},
		{Library: "Shows", Type: "Series", Name: "24", Path: "24", Parent: LibraryRoot},
		{Library: "Shows", Type: "Series", Name: "The Daily Show", Path: "The Daily Show", Parent: LibraryRoot},
		{Library: "Shows", Type: "Series", Name: "The Series", Path: "The Series", Parent: LibraryRoot},
		{Library: "Shows", Type: "Season", Name: "Season 1", Path: "24/Season 01", Parent: "24"},
		{Library: "Shows", Type: "Season", Name: "Season 2024", Path: "The Daily Show/Season 2024", Parent: "The Daily Show"},
		{Library: "Shows", Type: "Season", Name: "Season 1", Path: "The Series/Season 01", Parent: "The Series"},
		// The Series/Season 02 and the season inferred from
		// "The Series - S02E01 - No Season Directory.mkv" are one item, not
		// two: a season's identity is its series' identity plus its number
		// (§3.6), so an inferred season and a directory naming the same number
		// derive the same identifier.
		{Library: "Shows", Type: "Season", Name: "Season 2", Path: "The Series/Season 02", Parent: "The Series"},
		{Library: "Shows", Type: "Season", Name: "Specials", Path: "The Series/Specials", Parent: "The Series"},
		// "The Series/Season 03" holds no episode, so the walk — which collects
		// candidate *files* — never reaches it and no item is made of it. The
		// reference makes a Season of that directory, which is a difference T17
		// declares rather than one this file hides.
		{Library: "Shows", Type: "Episode", Name: "12-00 AM", Path: "24/Season 01/24 - S01E01 - 12-00 AM.mkv", Parent: "24/Season 01"},
		{Library: "Shows", Type: "Episode", Name: "The Daily Show - 2024-01-31", Path: "The Daily Show/Season 2024/The Daily Show - 2024-01-31.mkv", Parent: "The Daily Show/Season 2024"},
		{Library: "Shows", Type: "Episode", Name: "A Special", Path: "The Series/Specials/The Series - S00E01 - A Special.mkv", Parent: "The Series/Specials"},
		{Library: "Shows", Type: "Episode", Name: "Pilot", Path: "The Series/Season 01/The Series - S01E01 - Pilot.mkv", Parent: "The Series/Season 01"},
		{Library: "Shows", Type: "Episode", Name: "Two Parter", Path: "The Series/Season 01/The Series - S01E02-E03 - Two Parter.mkv", Parent: "The Series/Season 01"},
		{Library: "Shows", Type: "Episode", Name: "Old Transfer", Path: "The Series/Season 01/The Series - S01E04 - Old Transfer.avi", Parent: "The Series/Season 01"},
		{Library: "Shows", Type: "Episode", Name: "blob", Path: "The Series/Season 01/blob.mkv", Parent: "The Series/Season 01", Unplaceable: true},
		{Library: "Shows", Type: "Episode", Name: "No Season Directory", Path: "The Series/The Series - S02E01 - No Season Directory.mkv", Parent: "The Series/Season 02"},
		{Library: "Shows", Type: "Episode", Name: "Beyond Any Real Count", Path: "The Series/Season 02/The Series - S02E99 - Beyond Any Real Count.mp4", Parent: "The Series/Season 02"},

		// ---- Music: 003 §3.5 -------------------------------------------------
		{Library: "Music", Type: "CollectionFolder", Name: "Music"},
		{Library: "Music", Type: "MusicArtist", Name: "The Artist", Path: "The Artist", Parent: LibraryRoot},
		{Library: "Music", Type: "MusicArtist", Name: "Various Artists", Path: "Various Artists", Parent: LibraryRoot},
		// One album across two discs, so CD1 and CD2 back no item of their own
		// (AC-8). The reference makes a Folder of each, which is T17's to
		// declare.
		{Library: "Music", Type: "MusicAlbum", Name: "Double Album", Path: "The Artist/Double Album", Parent: "The Artist"},
		{Library: "Music", Type: "MusicAlbum", Name: "First Album", Path: "The Artist/First Album (2001)", Parent: "The Artist"},
		{Library: "Music", Type: "MusicAlbum", Name: "Second Album", Path: "The Artist/Second Album", Parent: "The Artist"},
		{Library: "Music", Type: "MusicAlbum", Name: "spandau_ballet-through_the_barricades", Path: "The Artist/spandau_ballet-through_the_barricades", Parent: "The Artist"},
		{Library: "Music", Type: "MusicAlbum", Name: "A Compilation", Path: "Various Artists/A Compilation (1999)", Parent: "Various Artists"},
		{Library: "Music", Type: "Audio", Name: "First Disc", Path: "The Artist/Double Album/CD1/01 - First Disc.flac", Parent: "The Artist/Double Album"},
		{Library: "Music", Type: "Audio", Name: "Second Disc", Path: "The Artist/Double Album/CD2/01 - Second Disc.flac", Parent: "The Artist/Double Album"},
		{Library: "Music", Type: "Audio", Name: "Opening", Path: "The Artist/First Album (2001)/01 - Opening.flac", Parent: "The Artist/First Album (2001)"},
		{Library: "Music", Type: "Audio", Name: "Second", Path: "The Artist/First Album (2001)/02 - Second.flac", Parent: "The Artist/First Album (2001)"},
		{Library: "Music", Type: "Audio", Name: "In Another Container", Path: "The Artist/Second Album/01 - In Another Container.m4a", Parent: "The Artist/Second Album"},
		{Library: "Music", Type: "Audio", Name: "And Another", Path: "The Artist/Second Album/02 - And Another.dsf", Parent: "The Artist/Second Album"},
		{Library: "Music", Type: "Audio", Name: "Tagged Differently", Path: "The Artist/spandau_ballet-through_the_barricades/01 - Tagged Differently.flac", Parent: "The Artist/spandau_ballet-through_the_barricades"},
		{Library: "Music", Type: "Audio", Name: "By One Artist", Path: "Various Artists/A Compilation (1999)/01 - By One Artist.flac", Parent: "Various Artists/A Compilation (1999)"},
		{Library: "Music", Type: "Audio", Name: "By Another", Path: "Various Artists/A Compilation (1999)/02 - By Another.flac", Parent: "Various Artists/A Compilation (1999)"},
		{Library: "Music", Type: "Audio", Name: "By A Third", Path: "Various Artists/A Compilation (1999)/03 - By A Third.flac", Parent: "Various Artists/A Compilation (1999)"},

		// ---- Empty: a library that reads clean and holds nothing (§3.1) ------
		{Library: "Empty", Type: "CollectionFolder", Name: "Empty"},
	}
}
