// The expected sort keys of 003 §3.7, written down.
//
// This file is a **literal and never a derivation**, and
// `TestTheExpectedSortKeysAreLiteralsAndNotADerivation` in
// `library_sortkey_test.go` fails if it ever mentions the derivation it is the
// expected value of. The reason is the build the audit found: three lines in
// the scanner store each item's name as its sort key while leaving the
// derivation itself correct, so a table that asked the derivation what to
// expect would agree with the wrong build about every row.
//
// It is separated from the assertions for exactly 003 T1's reason — the guard
// over `expected.go` cannot live in the file it guards, because the guard has
// to name what it forbids.
package app

// sortKeyRow is one expected key: where the item is, and the bytes the column
// must hold for it.
//
// where is the item's root-relative path, or `"<Type>: <Name>"` for the
// container rows that have none — the same spelling
// `reference_reading_test.go` uses, for the same reason: a comparison keyed on
// the path alone silently pairs two path-less rows.
type sortKeyRow struct {
	where string
	name  string
	key   string
	shows string
}

// theSortKeysOfTheMoviesLibrary is every item of the fixture's `movies`
// library, in the order their keys name.
//
// It is the whole library rather than a selection, because the ordering
// assertion below needs a total order and a selection would assert a
// subsequence. Ten of spec §3.7.1's fourteen measured rows are in here; the
// four that are not (`Matrix The`, `Once The Time`, `A Bridge` alone and
// `10 Things` alone) are names no directory tree carries, and they stay where
// they are asserted, in `internal/library`.
func theSortKeysOfTheMoviesLibrary() []sortKeyRow {
	return []sortKeyRow{
		{"2 Fast 2 Furious (2003).mkv", "2 Fast 2 Furious",
			"0000000002 fast 0000000002 furious", "every digit run padded, not just the leading one"},
		{"10 Things I Hate About You (1999).mkv", "10 Things I Hate About You",
			"0000000010 things i hate about you", "which is what makes 2 sort before 10"},
		{"100% Wolf (2020).mkv", "100% Wolf",
			"0000000100  wolf", "replacement and padding together, and the double space they leave"},
		{"Amélie (2001).mkv", "Amélie",
			"amelie", "diacritics folded"},
		{"A Bridge Too Far (1977).mkv", "A Bridge Too Far",
			"bridge too far", "single-letter article at the start"},
		{"A Broadcast Capture (2011).ts", "A Broadcast Capture",
			"broadcast capture", "the same article, on the library's one `.ts`"},
		{"Don't Look Up (2021).mkv", "Don't Look Up",
			"dont look up", "apostrophe removed"},
		{"iRobot (2004).mkv", "iRobot",
			"irobot", "case normalised"},
		{"The Long Film (1998)/The Long Film (1998) - part1.mkv", "The Long Film",
			"long film", "a multi-part film is keyed once, on the name its directory gave it"},
		{"The Matrix (1999)/The Matrix (1999).mkv", "The Matrix",
			"matrix", "article at the start"},
		{"CollectionFolder: Movies", "Movies",
			"movies", "the library's own row is keyed like everything else"},
		{"A Newer Transfer (2015).mp4", "A Newer Transfer",
			"newer transfer", "article at the start"},
		{"An Old Transfer (1985).avi", "An Old Transfer",
			"old transfer", "a two-letter article"},
		{"  Padded   (1999).mkv", "Padded",
			"padded", "trimmed at step 1, before anything else"},
		{"Rock & Roll (1978).mkv", "Rock & Roll",
			"rock  roll", "TWO SPACES — nothing collapses them"},
		{"S.W.A.T. (2003).mkv", "S.W.A.T.",
			"s w a t ", "a TRAILING SPACE — nothing trims it"},
		{"Wall-E (2008).mkv", "Wall-E",
			"walle", "character removed"},
	}
}

// theSortKeysOfTheOverridingTypes is spec §3.7.2's three types, over the
// fixture's own tree.
//
// Each row is chosen for something §3.7.1 would get wrong: the raw name that
// survives un-lowercased and un-padded, the asymmetric widths, the season-zero
// that is a present zero rather than an absent number, and the missing number
// that contributes no segment at all.
func theSortKeysOfTheOverridingTypes() map[string][]sortKeyRow {
	return map[string][]sortKeyRow{
		"Shows": {
			{"The Series/Specials/The Series - S00E01 - A Special.mkv", "A Special",
				"000 - 0001 - A Special", "season 0 is a present zero and not an absent number"},
			{"The Series/Specials", "Specials",
				"0000", "a Season is four digits and nothing else"},
			{"The Series/Season 01/The Series - S01E02-E03 - Two Parter.mkv", "Two Parter",
				"001 - 0002 - Two Parter", "season padded to THREE and episode to FOUR"},
			{"The Series/Season 02/The Series - S02E99 - Beyond Any Real Count.mp4", "Beyond Any Real Count",
				"002 - 0099 - Beyond Any Real Count", "the raw name, neither lowercased nor folded"},
			{"The Series/Season 01/blob.mkv", "blob",
				"001 - blob", "a missing number contributes no segment at all"},
			{"The Daily Show/Season 2024", "Season 2024",
				"2024", "a four-digit season number is not padded further"},
			{"The Daily Show/Season 2024/The Daily Show - 2024-01-31.mkv", "The Daily Show - 2024-01-31",
				"2024 - The Daily Show - 2024-01-31", "the raw name keeps its article and its digit runs"},
			{"The Daily Show", "The Daily Show",
				"daily show", "a Series is §3.7.1's derivation and not §3.7.2's"},
			{"24", "24",
				"0000000024", "a series whose name is digits is padded, and is not an episode number"},
		},
		"Music": {
			{"The Artist/Double Album/CD1/01 - First Disc.flac", "First Disc",
				"0001 - 0001 - First Disc", "disc padded to 4, track padded to 4, then the RAW name"},
			{"The Artist/Double Album/CD2/01 - Second Disc.flac", "Second Disc",
				"0002 - 0001 - Second Disc", "the second disc's tracks sort after the first's"},
			{"The Artist/First Album (2001)/01 - Opening.flac", "Opening",
				"0001 - Opening", "no disc number contributes no segment"},
			{"The Artist", "The Artist",
				"artist", "a MusicArtist is §3.7.1's derivation"},
			{"Various Artists/A Compilation (1999)", "A Compilation",
				"compilation", "a MusicAlbum is too, article and all"},
			{"The Artist/spandau_ballet-through_the_barricades", "spandau_ballet-through_the_barricades",
				"spandau_balletthrough_the_barricades", "the hyphen goes and the underscores stay"},
		},
	}
}
