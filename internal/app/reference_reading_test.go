package app

// The declared differences between the reference's reading of the fixture tree
// and Atrium's own scan of it — 003 tasks' T17.
//
// # What this is
//
// [docs/compatibility/reference-fixture-reading.json] is the reference's own
// reading of this repository's fixture tree, recorded once by a probe against a
// single-use instance
// `[probe: tools/probe_reference_scan.py, Jellyfin 10.11.11, 2026-09-02]`.
// This file scans the same tree with this server and compares the two.
//
// **The comparison is not an equality**, and that is the single most useful
// thing 010 hands this feature (003 plan §8.2, [AGENTS.md] §3): a conformance
// assertion is a *declared inequality*, where an undeclared difference fails
// **and a declared one that has gone away fails too**. An equality would be
// false, and a comparison with no declaration at all would be useless.
//
// # Why the declaration is Go and not a seventh file under docs/compatibility/
//
// 003 tasks' T17 decides it and 003 plan §8.2 records it. A new machine-readable
// artefact there owes a prose twin and a row-for-row test comparing the two, and
// this declaration has no twin to pair with: the prose that explains a row is
// *this project's own specification section*, which the row cites in [Reason].
// [docs/compatibility/conformance.md]'s L3 section already describes the
// declaration as living "in that module with its reason", so a Go table beside
// the comparison is the recorded shape rather than a new one.
//
// # What is compared, and what deliberately is not
//
// **Type, name and path. Never an identifier.** behaviours §1.4 establishes that
// the two servers derive identifiers differently by design, so comparing them
// would declare 74 differences that say nothing. The recorded reading carries
// none, and [TestTheComparisonSurvivesEveryIdentifierMoving] asserts the
// stronger half: two installations of the same tree, whose library identifiers
// and therefore whose every item identifier differ, produce the *same*
// difference set.
//
// # The count, which is thirty-two here and forty-seven in five documents
//
// 010's D-7 measured **forty-seven** declared differences "over the six
// libraries the fixture composes"
// `[probe: tools/probe_reference_scan.py, Jellyfin 10.11.11, 2026-09-02]`, and
// that number is what [CLAUDE.md], specs/README.md, 010's spec, 003's plan and
// (after this change) [docs/compatibility/conformance.md] all carry. **This
// declaration holds thirty-two, and the difference is not a disagreement about
// the reference.** The arithmetic, because a number nobody can reproduce is a
// number nobody can check:
//
//	the reading's six libraries                                74 items
//	  Movies, Shows, Music, Empty — built by internal/libraryfixture
//	  Films, Tunes  — 008's media world, which ffmpeg encodes and a build
//	                  tag separates (architecture §8), named in
//	                  libraryfixture.NotBuiltHere and not built here
//
//	differences over the four libraries this feature builds        32   declared below
//	differences over Films and Tunes, if 008's world were built     8   predicted, unreachable
//	                                                              ---
//	                                                               40
//	010's D-7 count                                                47
//	                                                              ---
//	unaccounted for                                                 7
//
// The eight predicted are Films' and Tunes' own root rows and the six names
// this project's rules would derive differently there — five folder-per-film
// titles that keep their year in the reading, and the album the reading calls
// `The Album` under a directory called `Untitled Folder`. They are predictions
// and not declarations, because **this feature cannot build the tree they are
// over**: 008 owns it, and a row that no run can reach is not a declared
// inequality, it is a comment.
//
// The remaining **seven are not derivable from the reading and this project's
// specifications**, which is the only material T17 is allowed (003 plan §8.2:
// "It has the reading. It does not have the declaration"). The forty-seven was
// counted against the *other* implementation over both fixture worlds; this one
// derived thirty-two over the four worlds it has. **Manufacturing seven rows to
// reach a number would be exactly the failure the count assertion exists to
// prevent, one direction round.** The gap is a finding of the experiment rather
// than a defect: it is a place where two implementations of one specification
// differ, and it is recorded here, in 003 plan §8.2 and in
// [docs/compatibility/conformance.md]'s correction.
//
// # The mutations, because a check that has never failed has proved nothing
//
// The two clauses that make this a declared inequality are each proven from the
// *declaration* side by a test — [TestAnUndeclaredDifferenceFails] removes every
// row in turn, [TestADeclaredDifferenceThatHasGoneAwayFails] invents two. Two
// real behavioural mutations were run as well, end to end through the
// subcommand and the store, because a comparison can be right about a table it
// is handed and wrong about the server
// `[measurement: 003 T17, 2 mutations, 2026-09-05]`:
//
//	a track's name keeps its leading number      10 declared differences gone away
//	a film's name keeps its year                  2 gone away, 1 changed shape,
//	                                             13 undeclared
//
// The first is the shape 004's landing will have, and the second is the shape a
// naming rule quietly regressing has. Both were reverted.
//
// # Twenty-three of the thirty-two are 004's, and 004 does not exist
//
// They are declared now because the comparison cannot run without a reason for
// every difference. The consequence is written down rather than discovered:
// **a declared difference that has gone away fails too, so the day 004's
// metadata resolution renames one of these items, the row declaring it goes
// red.** That is the rule working, not a defect. What 004 owes is to *edit*
// those rows — the [declaredDifference] literal in this file — rather than
// delete them, and to keep [declaredDifferenceTotal] equal to the table's own
// length, since a row removed to make a run go green is exactly what the count
// assertion exists to catch.
//
// [AGENTS.md]: ../../AGENTS.md
// [CLAUDE.md]: ../../CLAUDE.md
// [docs/compatibility/conformance.md]: ../../docs/compatibility/conformance.md
// [docs/compatibility/reference-fixture-reading.json]: ../../docs/compatibility/reference-fixture-reading.json

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/vdatanet/atrium-go/internal/libraryfixture"
)

// repositoryRoot is where this package sits, so a test can name the reference's
// recorded reading from the package directory the test runs in. The same
// constant, for the same reason, is in internal/libraryfixture,
// internal/scan and internal/surface.
const repositoryRoot = "../.."

// declaredDifferenceTotal is the number of rows [declaredDifferences] holds,
// written here so that the length is asserted against something rather than
// against itself.
//
// **A row deleted to make a run go green is a failing count and not a quieter
// suite.** That is the whole reason this constant exists: every other assertion
// in this file compares the declaration against the two readings, and a
// declaration with a row taken out agrees with two readings that also lost the
// difference. Only the count notices.
const declaredDifferenceTotal = 32

// --- The vocabulary -----------------------------------------------------------

// differenceKind is what kind of difference a row records. A pair that differs
// in both type and name is a typeDiffers row: both sides are written out, so
// the name is not hidden by the classification.
type differenceKind string

const (
	typeDiffers        differenceKind = "the type differs"
	nameDiffers        differenceKind = "the name differs"
	onlyInTheReference differenceKind = "only the reference has it"
	onlyInAtrium       differenceKind = "only Atrium has it"
)

// theLibrarysOwnRow is the Where of a library's own item — the row that has no
// path because a library may be configured with several roots (003 §4.2).
const theLibrarysOwnRow = "(the library's own row)"

// declaredDifference is one difference between the two readings, with the
// reason it exists and the feature that owns whether it stays.
//
// **A declaration is a reason and an owning feature, not a licence** (003 plan
// §8.2). Both halves are required and [validateDeclarations] refuses a row
// missing either — which is [docs/compatibility/allowlist.yaml]'s own rule
// ("an entry with neither fails the load") applied to a different file for the
// same reason.
//
// [docs/compatibility/allowlist.yaml]: ../../docs/compatibility/allowlist.yaml
type declaredDifference struct {
	// Library is the fixture library, which must be one this package builds.
	Library string

	// Where names the item: its root-relative path, [theLibrarysOwnRow], or
	// "(no path) Type: Name" for an item the reference gives no path at all.
	Where string

	Kind differenceKind

	// Reference and Atrium are "Type: Name" as each side reads it, and empty
	// where that side has no such item. They are part of the declaration
	// rather than of the observation so that a row goes red when either side
	// moves — which is what makes 004's landing turn these rows red rather
	// than silently redefine them.
	Reference string
	Atrium    string

	// Reason is the specification section that decides **Atrium's** side of
	// this difference. It is a section of a document in this repository, so a
	// wrong reason is a wrong citation somebody can check.
	Reason string

	// Owner is the feature that owns whether this difference stays.
	Owner string

	// Because is the one sentence a reader needs to agree or disagree.
	Because string
}

// ownableFeatures are the features a row may name as its owner. It is a closed
// list on purpose: an owner nobody recognises is a row nobody will revisit.
var ownableFeatures = map[string]string{
	"003": "library configuration and scanning",
	"004": "metadata resolution",
}

// --- The declaration ----------------------------------------------------------

// declaredDifferences is the table.
//
// The shape of it, which is what 004 will edit:
//
//	                                              rows  owner
//	Movies  the library's own row                    1    003
//	        a film's name loses its year             2    004
//	        a path-derived name is trimmed           1    004
//	        a zero-byte file is not a media file     1    003
//	Shows   the library's own row                    1    003
//	        a season no candidate file reaches       1    003
//	        a season the reference invents           1    003
//	        a series named from something else       1    004
//	        an episode is named after its title      7    004
//	Music   the library's own row                    1    003
//	        a disc directory is not an item          2    003
//	        an album's name loses its year           2    004
//	        a track is named after its title        10    004
//	Empty   the library's own row                    1    003
//	                                               ---
//	                                                32
func declaredDifferences() []declaredDifference {
	return []declaredDifference{
		// ---- Movies ---------------------------------------------------------
		{
			Library: "Movies", Where: theLibrarysOwnRow, Kind: typeDiffers,
			Reference: "Folder: Movies", Atrium: "CollectionFolder: Movies",
			Reason: "003 §3.1", Owner: "003",
			Because: "spec §3.1: \"each library becomes a CollectionFolder item\". The reference's own row for a library is a Folder, and 010's amendment names this shape for every library",
		},
		{
			Library: "Movies", Where: "  Padded   (1999).mkv", Kind: nameDiffers,
			Reference: "Movie:   Padded", Atrium: "Movie: Padded",
			Reason: "003 §3.3, §3.5", Owner: "004",
			Because: "§3.5 states the rule in terms — \"the name derived from a path is trimmed\" — where the reference keeps the leading whitespace its filename carries",
		},
		{
			Library: "Movies", Where: "An Incomplete Copy (2000).mkv", Kind: onlyInTheReference,
			Reference: "Movie: An Incomplete Copy", Atrium: "",
			Reason: "003 §3.2", Owner: "003",
			Because: "§3.2's zero-byte rule working as written: an incomplete copy is not a media file here, and the reference makes an item of it whose probe finds no streams",
		},
		{
			Library: "Movies", Where: "The Long Film (1998)/The Long Film (1998) - part1.mkv", Kind: nameDiffers,
			Reference: "Movie: The Long Film (1998)", Atrium: "Movie: The Long Film",
			Reason: "003 §3.3", Owner: "004",
			Because: "§3.3 extracts the year out of a title into ProductionYear; the reference names a folder-per-film item after the directory whole, year included",
		},
		{
			Library: "Movies", Where: "The Matrix (1999)/The Matrix (1999).mkv", Kind: nameDiffers,
			Reference: "Movie: The Matrix (1999)", Atrium: "Movie: The Matrix",
			Reason: "003 §3.3", Owner: "004",
			Because: "the same rule as the multi-part film above, over a directory holding one file",
		},

		// ---- Shows ----------------------------------------------------------
		{
			Library: "Shows", Where: theLibrarysOwnRow, Kind: typeDiffers,
			Reference: "Folder: Shows", Atrium: "CollectionFolder: Shows",
			Reason: "003 §3.1", Owner: "003",
			Because: "the library's own row, as for every library",
		},
		{
			Library: "Shows", Where: "The Series", Kind: nameDiffers,
			Reference: "Series: tvshow", Atrium: "Series: The Series",
			Reason: "003 §3.4", Owner: "004",
			Because: "§3.4 names a series after the directory that holds it. No path-derived rule produces `tvshow` from a directory called `The Series`, and the tree carries no sidecar to explain it, so the name came from somewhere 003 does not read — which is 004's",
		},
		{
			Library: "Shows", Where: "The Series/Season 03", Kind: onlyInTheReference,
			Reference: "Season: Season 3", Atrium: "",
			Reason: "003 §3.4", Owner: "003",
			Because: "the walk collects candidate *files* and this directory holds none, so nothing reaches it and no season is made of it. The reference makes a Season of the directory itself",
		},
		{
			Library: "Shows", Where: "(no path) Season: Season Unknown", Kind: onlyInTheReference,
			Reference: "Season: Season Unknown", Atrium: "",
			Reason: "003 §3.4, §3.8", Owner: "003",
			Because: "the reference invents a pathless season for the episode whose filename says no number; §3.8 counts that file as unplaceable instead and leaves it under the season directory that holds it",
		},
		{
			Library: "Shows", Where: "24/Season 01/24 - S01E01 - 12-00 AM.mkv", Kind: nameDiffers,
			Reference: "Episode: 24 - S01E01 - 12-00 AM", Atrium: "Episode: 12-00 AM",
			Reason: "003 §3.4", Owner: "004",
			Because: "an episode is named after what follows its numbering; the reference names it after its whole cleaned filename",
		},
		{
			Library: "Shows", Where: "The Series/Season 01/The Series - S01E01 - Pilot.mkv", Kind: nameDiffers,
			Reference: "Episode: The Series - S01E01 - Pilot", Atrium: "Episode: Pilot",
			Reason: "003 §3.4", Owner: "004",
			Because: "the same rule",
		},
		{
			Library: "Shows", Where: "The Series/Season 01/The Series - S01E02-E03 - Two Parter.mkv", Kind: nameDiffers,
			Reference: "Episode: The Series - S01E02-E03 - Two Parter", Atrium: "Episode: Two Parter",
			Reason: "003 §3.4", Owner: "004",
			Because: "the same rule, over a file spanning two episode numbers",
		},
		{
			Library: "Shows", Where: "The Series/Season 01/The Series - S01E04 - Old Transfer.avi", Kind: nameDiffers,
			Reference: "Episode: The Series - S01E04 - Old Transfer", Atrium: "Episode: Old Transfer",
			Reason: "003 §3.4", Owner: "004",
			Because: "the same rule",
		},
		{
			Library: "Shows", Where: "The Series/Season 02/The Series - S02E99 - Beyond Any Real Count.mp4", Kind: nameDiffers,
			Reference: "Episode: The Series - S02E99 - Beyond Any Real Count", Atrium: "Episode: Beyond Any Real Count",
			Reason: "003 §3.4", Owner: "004",
			Because: "the same rule, over an episode number beyond any real count",
		},
		{
			Library: "Shows", Where: "The Series/Specials/The Series - S00E01 - A Special.mkv", Kind: nameDiffers,
			Reference: "Episode: The Series - S00E01 - A Special", Atrium: "Episode: A Special",
			Reason: "003 §3.4", Owner: "004",
			Because: "the same rule, over season zero",
		},
		{
			Library: "Shows", Where: "The Series/The Series - S02E01 - No Season Directory.mkv", Kind: nameDiffers,
			Reference: "Episode: The Series - S02E01 - No Season Directory", Atrium: "Episode: No Season Directory",
			Reason: "003 §3.4", Owner: "004",
			Because: "the same rule, over an episode whose season is inferred from its own filename",
		},

		// ---- Music ----------------------------------------------------------
		{
			Library: "Music", Where: theLibrarysOwnRow, Kind: typeDiffers,
			Reference: "Folder: Music", Atrium: "CollectionFolder: Music",
			Reason: "003 §3.1", Owner: "003",
			Because: "the library's own row, as for every library",
		},
		{
			Library: "Music", Where: "The Artist/Double Album/CD1", Kind: onlyInTheReference,
			Reference: "Folder: CD1", Atrium: "",
			Reason: "003 §3.5", Owner: "003",
			Because: "§3.5's disc directory names a disc number and never an item: one album across two discs is one album (AC-8). The reference makes a Folder of each disc directory as well",
		},
		{
			Library: "Music", Where: "The Artist/Double Album/CD2", Kind: onlyInTheReference,
			Reference: "Folder: CD2", Atrium: "",
			Reason: "003 §3.5", Owner: "003",
			Because: "the second disc of the same album, for the same reason",
		},
		{
			Library: "Music", Where: "The Artist/First Album (2001)", Kind: nameDiffers,
			Reference: "MusicAlbum: First Album (2001)", Atrium: "MusicAlbum: First Album",
			Reason: "003 §3.5", Owner: "004",
			Because: "an album's directory name loses its year to ProductionYear and nothing else; the reference cleans an album folder's name not at all",
		},
		{
			Library: "Music", Where: "Various Artists/A Compilation (1999)", Kind: nameDiffers,
			Reference: "MusicAlbum: A Compilation (1999)", Atrium: "MusicAlbum: A Compilation",
			Reason: "003 §3.5", Owner: "004",
			Because: "the same rule, over the compilation",
		},
		{
			Library: "Music", Where: "The Artist/Double Album/CD1/01 - First Disc.flac", Kind: nameDiffers,
			Reference: "Audio: 01 - First Disc", Atrium: "Audio: First Disc",
			Reason: "003 §3.5", Owner: "004",
			Because: "§3.5's path fallback takes the leading track number off a track's name. That is Atrium's declared divergence — behaviours §2.16, OQ-8 — and the reference keeps the whole stem",
		},
		{
			Library: "Music", Where: "The Artist/Double Album/CD2/01 - Second Disc.flac", Kind: nameDiffers,
			Reference: "Audio: 01 - Second Disc", Atrium: "Audio: Second Disc",
			Reason: "003 §3.5", Owner: "004",
			Because: "the same rule",
		},
		{
			Library: "Music", Where: "The Artist/First Album (2001)/01 - Opening.flac", Kind: nameDiffers,
			Reference: "Audio: 01 - Opening", Atrium: "Audio: Opening",
			Reason: "003 §3.5", Owner: "004",
			Because: "the same rule",
		},
		{
			Library: "Music", Where: "The Artist/First Album (2001)/02 - Second.flac", Kind: nameDiffers,
			Reference: "Audio: 02 - Second", Atrium: "Audio: Second",
			Reason: "003 §3.5", Owner: "004",
			Because: "the same rule",
		},
		{
			Library: "Music", Where: "The Artist/Second Album/01 - In Another Container.m4a", Kind: nameDiffers,
			Reference: "Audio: 01 - In Another Container", Atrium: "Audio: In Another Container",
			Reason: "003 §3.5", Owner: "004",
			Because: "the same rule",
		},
		{
			Library: "Music", Where: "The Artist/Second Album/02 - And Another.dsf", Kind: nameDiffers,
			Reference: "Audio: 02 - And Another", Atrium: "Audio: And Another",
			Reason: "003 §3.5", Owner: "004",
			Because: "the same rule",
		},
		{
			Library: "Music", Where: "The Artist/spandau_ballet-through_the_barricades/01 - Tagged Differently.flac", Kind: nameDiffers,
			Reference: "Audio: 01 - Tagged Differently", Atrium: "Audio: Tagged Differently",
			Reason: "003 §3.5", Owner: "004",
			Because: "the same rule. This is the file §3.5's measurement is about, and neither server can read a tag out of filler bytes, so both fall back to the path",
		},
		{
			Library: "Music", Where: "Various Artists/A Compilation (1999)/01 - By One Artist.flac", Kind: nameDiffers,
			Reference: "Audio: 01 - By One Artist", Atrium: "Audio: By One Artist",
			Reason: "003 §3.5", Owner: "004",
			Because: "the same rule",
		},
		{
			Library: "Music", Where: "Various Artists/A Compilation (1999)/02 - By Another.flac", Kind: nameDiffers,
			Reference: "Audio: 02 - By Another", Atrium: "Audio: By Another",
			Reason: "003 §3.5", Owner: "004",
			Because: "the same rule",
		},
		{
			Library: "Music", Where: "Various Artists/A Compilation (1999)/03 - By A Third.flac", Kind: nameDiffers,
			Reference: "Audio: 03 - By A Third", Atrium: "Audio: By A Third",
			Reason: "003 §3.5", Owner: "004",
			Because: "the same rule",
		},

		// ---- Empty ----------------------------------------------------------
		{
			Library: "Empty", Where: theLibrarysOwnRow, Kind: onlyInAtrium,
			Reference: "", Atrium: "CollectionFolder: Empty",
			Reason: "003 §3.1", Owner: "003",
			Because: "§3.1 makes every library a CollectionFolder whether or not anything is under it. A library with no items is nothing at all to the reference, which is one of the four shapes 010's own amendment names",
		},
	}
}

// --- The two readings ---------------------------------------------------------

// readingItem is one item of either reading, reduced to the three things the
// comparison is over.
type readingItem struct {
	Type string
	Name string

	// Path is root-relative and slash-separated, and empty where the item has
	// none: the library's own row on both sides, and the season the reference
	// invents for an episode it cannot number.
	Path string
}

func (i readingItem) describe() string { return i.Type + ": " + i.Name }

// referenceReading is docs/compatibility/reference-fixture-reading.json,
// parsed.
//
// Only the members the comparison reads are declared. The file also carries the
// probe's citation, the image digest, the counts and the fetcher names it
// switched off, and a struct that reproduced them would be a transcription of
// exactly the kind this file exists to avoid.
type referenceReading struct {
	Libraries []referenceLibrary `json:"libraries"`
}

type referenceLibrary struct {
	Name      string `json:"name"`
	ItemCount int    `json:"item_count"`
	Items     []struct {
		Type string  `json:"type"`
		Name string  `json:"name"`
		Path *string `json:"path"`
	} `json:"items"`
}

// theReferencesReading reads and parses the recorded reading.
func theReferencesReading(t *testing.T) map[string][]readingItem {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash(libraryfixture.ReferenceReading)))
	if err != nil {
		t.Fatalf("reading %s: %v", libraryfixture.ReferenceReading, err)
	}

	var parsed referenceReading
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("parsing %s: %v", libraryfixture.ReferenceReading, err)
	}
	if len(parsed.Libraries) == 0 {
		t.Fatalf("%s parsed to no libraries at all", libraryfixture.ReferenceReading)
	}

	byLibrary := make(map[string][]readingItem, len(parsed.Libraries))
	for _, lib := range parsed.Libraries {
		if len(lib.Items) != lib.ItemCount {
			t.Fatalf("%s says %s holds %d items and lists %d",
				libraryfixture.ReferenceReading, lib.Name, lib.ItemCount, len(lib.Items))
		}
		items := make([]readingItem, 0, len(lib.Items))
		for _, item := range lib.Items {
			path := ""
			if item.Path != nil {
				path = *item.Path
			}
			items = append(items, readingItem{Type: item.Type, Name: item.Name, Path: path})
		}
		byLibrary[lib.Name] = items
	}
	return byLibrary
}

// atriumsReading scans the whole fixture through the subcommand and reads back
// what the store ended up holding, by library name.
//
// It goes through `atrium library add` and `atrium library scan` rather than
// calling a resolver, because the comparison is against a *server's* reading of
// a tree and a function has no store to end up holding anything.
func atriumsReading(t *testing.T) map[string][]readingItem {
	t.Helper()

	data, libraries := theWholeFixture(t, t.TempDir())
	if summaries := scanSummaries(t, data); len(summaries) != len(libraries) {
		t.Fatalf("the scan reported %d libraries where %d were declared", len(summaries), len(libraries))
	}

	byLibrary := make(map[string][]readingItem, len(libraries))
	for _, lib := range libraries {
		items := make([]readingItem, 0, 32)
		for _, item := range storedItems(t, data, lib.ID) {
			items = append(items, readingItem{Type: item.Type, Name: item.Name, Path: item.Path})
		}
		byLibrary[lib.Name] = items
	}
	return byLibrary
}

// --- The comparison -----------------------------------------------------------

// observedDifference is one difference the comparison actually found. It
// carries exactly the fields a declaration is matched on, plus both sides.
type observedDifference struct {
	Library   string
	Where     string
	Kind      differenceKind
	Reference string
	Atrium    string
}

func (d observedDifference) String() string {
	return fmt.Sprintf("%s / %s — %s (reference %q, Atrium %q)",
		d.Library, d.Where, d.Kind, d.Reference, d.Atrium)
}

// where names an item for the comparison: its path, or a spelling for the two
// shapes that have none.
func where(item readingItem, isTheLibrarysOwnRow bool) string {
	switch {
	case isTheLibrarysOwnRow:
		return theLibrarysOwnRow
	case item.Path != "":
		return item.Path
	default:
		return "(no path) " + item.describe()
	}
}

// theOwnRowOf finds a library's own row on one side and returns it with the
// rest.
//
// The two sides spell it differently — that difference is the point — so it is
// found by *role* and not by type: the one item of the library that has no path
// and is not something under it. The reference's Shows library has a second
// pathless row, the season it invents for an unnumbered episode, and that one is
// not a library's own row: it is told apart by carrying a type that is not one
// of the two container spellings.
func theOwnRowOf(t *testing.T, side, library string, items []readingItem) (readingItem, bool, []readingItem) {
	t.Helper()

	var own readingItem
	found := false
	rest := make([]readingItem, 0, len(items))
	for _, item := range items {
		if item.Path == "" && (item.Type == "Folder" || item.Type == "CollectionFolder") {
			if found {
				t.Fatalf("%s's %s reading has two pathless container rows, so which is the "+
					"library's own row is a guess: %q and %q", side, library, own.describe(), item.describe())
			}
			own, found = item, true
			continue
		}
		rest = append(rest, item)
	}
	return own, found, rest
}

// compareReadings is the whole comparison, as a pure function over the two
// readings of one library.
//
// It is a function and not a sequence of assertions so that the two clauses
// that matter can be *proven*: [TestAnUndeclaredDifferenceFails] and
// [TestADeclaredDifferenceThatHasGoneAwayFails] both need a comparison whose
// answer can be inspected rather than one that ends a test.
func compareReadings(t *testing.T, library string, reference, atrium []readingItem) []observedDifference {
	t.Helper()

	referenceOwn, referenceHasOwn, referenceRest := theOwnRowOf(t, "the reference's", library, reference)
	atriumOwn, atriumHasOwn, atriumRest := theOwnRowOf(t, "Atrium's", library, atrium)

	var found []observedDifference
	switch {
	case referenceHasOwn && atriumHasOwn:
		if kind, differs := howTheyDiffer(referenceOwn, atriumOwn); differs {
			found = append(found, observedDifference{
				Library: library, Where: theLibrarysOwnRow, Kind: kind,
				Reference: referenceOwn.describe(), Atrium: atriumOwn.describe(),
			})
		}
	case referenceHasOwn:
		found = append(found, observedDifference{
			Library: library, Where: theLibrarysOwnRow, Kind: onlyInTheReference,
			Reference: referenceOwn.describe(),
		})
	case atriumHasOwn:
		found = append(found, observedDifference{
			Library: library, Where: theLibrarysOwnRow, Kind: onlyInAtrium,
			Atrium: atriumOwn.describe(),
		})
	}

	referenceByWhere := indexed(t, "the reference's", library, referenceRest)
	atriumByWhere := indexed(t, "Atrium's", library, atriumRest)

	everyWhere := make([]string, 0, len(referenceByWhere)+len(atriumByWhere))
	for key := range referenceByWhere {
		everyWhere = append(everyWhere, key)
	}
	for key := range atriumByWhere {
		if _, both := referenceByWhere[key]; !both {
			everyWhere = append(everyWhere, key)
		}
	}
	slices.Sort(everyWhere)

	for _, key := range everyWhere {
		referenceItem, inReference := referenceByWhere[key]
		atriumItem, inAtrium := atriumByWhere[key]
		switch {
		case inReference && inAtrium:
			if kind, differs := howTheyDiffer(referenceItem, atriumItem); differs {
				found = append(found, observedDifference{
					Library: library, Where: key, Kind: kind,
					Reference: referenceItem.describe(), Atrium: atriumItem.describe(),
				})
			}
		case inReference:
			found = append(found, observedDifference{
				Library: library, Where: key, Kind: onlyInTheReference,
				Reference: referenceItem.describe(),
			})
		default:
			found = append(found, observedDifference{
				Library: library, Where: key, Kind: onlyInAtrium,
				Atrium: atriumItem.describe(),
			})
		}
	}
	return found
}

// howTheyDiffer answers whether two matched items differ, and on which of the
// two compared fields. A pair differing on both is a typeDiffers, and both
// sides are written out either way, so nothing is hidden by the classification.
func howTheyDiffer(reference, atrium readingItem) (differenceKind, bool) {
	switch {
	case reference.Type != atrium.Type:
		return typeDiffers, true
	case reference.Name != atrium.Name:
		return nameDiffers, true
	default:
		return "", false
	}
}

// indexed keys a side's items by their Where, failing loudly on a collision —
// two items one key would silently swallow.
func indexed(t *testing.T, side, library string, items []readingItem) map[string]readingItem {
	t.Helper()

	byWhere := make(map[string]readingItem, len(items))
	for _, item := range items {
		key := where(item, false)
		if existing, taken := byWhere[key]; taken {
			t.Fatalf("%s %s reading names %q twice — %q and %q — so the comparison would "+
				"silently drop one", side, library, key, existing.describe(), item.describe())
		}
		byWhere[key] = item
	}
	return byWhere
}

// theWholeComparison runs [compareReadings] over every library of the reading,
// and fails if a library is neither built here nor declared as not built.
//
// **A check that silently skipped a third of the libraries would read exactly
// like a check over all six**, which is why libraryfixture.NotBuiltHere exists
// and why this asks it rather than a list of its own.
func theWholeComparison(t *testing.T, reference, atrium map[string][]readingItem) []observedDifference {
	t.Helper()

	names := make([]string, 0, len(reference))
	for name := range reference {
		names = append(names, name)
	}
	slices.Sort(names)

	var found []observedDifference
	for _, name := range names {
		if reason, notBuilt := libraryfixture.NotBuiltHere[name]; notBuilt {
			if _, built := atrium[name]; built {
				t.Fatalf("%s is in libraryfixture.NotBuiltHere (%s) and Atrium scanned it anyway",
					name, reason)
			}
			continue
		}
		if _, built := atrium[name]; !built {
			t.Fatalf("%s is a library of %s that Atrium did not scan and that "+
				"libraryfixture.NotBuiltHere does not account for",
				name, libraryfixture.ReferenceReading)
		}
		found = append(found, compareReadings(t, name, reference[name], atrium[name])...)
	}

	for name := range atrium {
		if _, inTheReading := reference[name]; !inTheReading {
			t.Fatalf("Atrium scanned a library %q that %s does not name at all",
				name, libraryfixture.ReferenceReading)
		}
	}
	return found
}

// --- Declaration against observation ------------------------------------------

// key is what a declaration and an observation are matched on.
type key struct {
	Library string
	Where   string
	Kind    differenceKind
}

func (d declaredDifference) key() string {
	return fmt.Sprintf("%s / %s / %s", d.Library, d.Where, d.Kind)
}

func (d observedDifference) key() string {
	return fmt.Sprintf("%s / %s / %s", d.Library, d.Where, d.Kind)
}

// problemsWith is the declared inequality itself: what the comparison reports
// when the declaration and the two readings disagree.
//
// Three kinds of problem, and **the second is the one an equality cannot
// state**:
//
//  1. a difference nothing declares;
//  2. a declared difference that has gone away;
//  3. a declared difference whose two sides are no longer what the row says.
func problemsWith(declared []declaredDifference, observed []observedDifference) []string {
	byKey := make(map[string]declaredDifference, len(declared))
	for _, row := range declared {
		byKey[row.key()] = row
	}
	seen := make(map[string]bool, len(observed))

	var problems []string
	for _, difference := range observed {
		row, isDeclared := byKey[difference.key()]
		if !isDeclared {
			problems = append(problems, fmt.Sprintf(
				"undeclared difference: %s. Declare it with the specification section that "+
					"decides Atrium's side and the feature that owns it, or fix the difference",
				difference))
			continue
		}
		seen[difference.key()] = true
		if row.Reference != difference.Reference || row.Atrium != difference.Atrium {
			problems = append(problems, fmt.Sprintf(
				"declared difference %s no longer reads the way the row says: the row has "+
					"reference %q / Atrium %q and the comparison found reference %q / Atrium %q",
				row.key(), row.Reference, row.Atrium, difference.Reference, difference.Atrium))
		}
	}

	for _, row := range declared {
		if seen[row.key()] {
			continue
		}
		problems = append(problems, fmt.Sprintf(
			"declared difference that has gone away: %s (%s, owned by %s). A declaration is a "+
				"record of a decision and not a licence, so a difference that stopped appearing "+
				"fails rather than passing quietly — delete the row deliberately, and say what "+
				"closed it",
			row.key(), row.Reason, row.Owner))
	}
	slices.Sort(problems)
	return problems
}

// validateDeclarations is the table's own load, and it is
// docs/compatibility/allowlist.yaml's rule applied to a different file for the
// same reason: an entry that names neither a reason nor an owner fails to load.
func validateDeclarations(declared []declaredDifference) []string {
	var problems []string
	seen := map[string]bool{}
	built := map[string]bool{}
	for _, fixture := range libraryfixture.Libraries() {
		built[fixture.Name] = true
	}

	for _, row := range declared {
		switch {
		case row.Library == "":
			problems = append(problems, fmt.Sprintf("a row names no library: %+v", row))
		case !built[row.Library]:
			problems = append(problems, fmt.Sprintf(
				"%s names the library %q, which internal/libraryfixture does not build",
				row.key(), row.Library))
		}
		if row.Where == "" {
			problems = append(problems, fmt.Sprintf("a row of %s names no item at all", row.Library))
		}
		switch row.Kind {
		case typeDiffers, nameDiffers, onlyInTheReference, onlyInAtrium:
		default:
			problems = append(problems, fmt.Sprintf("%s names no kind of difference this comparison makes", row.key()))
		}
		if !strings.Contains(row.Reason, "§") {
			problems = append(problems, fmt.Sprintf(
				"%s names no reason as a specification section, and a difference with no reason "+
					"is an excuse rather than a decision", row.key()))
		}
		if _, known := ownableFeatures[row.Owner]; !known {
			problems = append(problems, fmt.Sprintf(
				"%s names no owning feature this project recognises (it says %q), and a "+
					"difference nobody owns is one nobody will revisit", row.key(), row.Owner))
		}
		if row.Because == "" {
			problems = append(problems, fmt.Sprintf("%s says nothing a reviewer could disagree with", row.key()))
		}
		switch row.Kind {
		case onlyInTheReference:
			if row.Reference == "" || row.Atrium != "" {
				problems = append(problems, fmt.Sprintf("%s is only-in-the-reference and does not read like it", row.key()))
			}
		case onlyInAtrium:
			if row.Atrium == "" || row.Reference != "" {
				problems = append(problems, fmt.Sprintf("%s is only-in-Atrium and does not read like it", row.key()))
			}
		default:
			if row.Reference == "" || row.Atrium == "" {
				problems = append(problems, fmt.Sprintf("%s compares two items and writes only one of them out", row.key()))
			}
		}
		if seen[row.key()] {
			problems = append(problems, fmt.Sprintf("%s is declared twice", row.key()))
		}
		seen[row.key()] = true
	}
	slices.Sort(problems)
	return problems
}

// --- The comparison itself ----------------------------------------------------

// TestTheTwoReadingsDifferInExactlyTheDeclaredPlaces is the run.
//
// It is AC-1's comparison against the reference's recorded reading (003 plan
// §8.2, §8.4 row 1) and 010's AC-2 as that criterion now reads: *"the
// reference's reading is recorded, Atrium's scan of the same tree is compared
// against it in the default job, every difference is declared with its reason
// and its owning feature, an undeclared difference fails, and a declared
// difference that has gone away fails too"*.
//
// **No Jellyfin anywhere** (AGENTS.md §1.6). The reference's half was recorded
// once, by a probe, against a single-use instance that no longer exists.
func TestTheTwoReadingsDifferInExactlyTheDeclaredPlaces(t *testing.T) {
	t.Parallel()

	declared := declaredDifferences()
	if problems := validateDeclarations(declared); len(problems) > 0 {
		t.Fatalf("the declaration does not load:\n  %s", strings.Join(problems, "\n  "))
	}

	observed := theWholeComparison(t, theReferencesReading(t), atriumsReading(t))

	// The control, in the direction the criterion does not name: a comparison
	// that found nothing would satisfy "no undeclared difference" over a build
	// that scanned nothing at all.
	if len(observed) == 0 {
		t.Fatal("the comparison found no difference of any kind, which over this tree means it " +
			"compared nothing — the two readings differ in declared places and an empty answer is " +
			"a broken instrument rather than a clean run")
	}

	if problems := problemsWith(declared, observed); len(problems) > 0 {
		t.Fatalf("the two readings do not differ in exactly the declared places:\n  %s",
			strings.Join(problems, "\n  "))
	}
	if len(observed) != len(declared) {
		t.Fatalf("the comparison found %d differences and the declaration holds %d",
			len(observed), len(declared))
	}
}

// TestAnUndeclaredDifferenceFails removes one declaration and watches the
// comparison go red.
//
// **A check that has never failed has proved nothing.** This is that proof for
// the first half of the declared inequality, and it is run over *every* row
// rather than over one, because a comparison keyed on the wrong field would
// still catch the row somebody happened to pick.
func TestAnUndeclaredDifferenceFails(t *testing.T) {
	t.Parallel()

	declared := declaredDifferences()
	observed := theWholeComparison(t, theReferencesReading(t), atriumsReading(t))

	for _, row := range declared {
		t.Run(row.key(), func(t *testing.T) {
			short := make([]declaredDifference, 0, len(declared)-1)
			for _, other := range declared {
				if other.key() != row.key() {
					short = append(short, other)
				}
			}

			problems := problemsWith(short, observed)
			if len(problems) != 1 {
				t.Fatalf("removing the declaration of %s produced %d problems and not one: %v",
					row.key(), len(problems), problems)
			}
			if !strings.Contains(problems[0], "undeclared difference") ||
				!strings.Contains(problems[0], row.Where) {
				t.Fatalf("removing the declaration of %s did not report that difference as "+
					"undeclared; it reported %q", row.key(), problems[0])
			}
		})
	}
}

// TestADeclaredDifferenceThatHasGoneAwayFails declares a difference the two
// readings do not have, and watches the comparison go red.
//
// **This is the assertion an equality cannot make**, and it is what makes the
// table a record of decisions rather than a list of excuses: a rule that
// quietly stops working takes a declared difference away with it, and a
// comparison that only refused undeclared differences would go *greener* as
// that happened.
//
// Two shapes, because they fail through different code: a row for an item that
// exists and does not differ, and a row for an item neither reading has at all.
func TestADeclaredDifferenceThatHasGoneAwayFails(t *testing.T) {
	t.Parallel()

	observed := theWholeComparison(t, theReferencesReading(t), atriumsReading(t))

	invented := []declaredDifference{
		{
			// The two readings agree on this film exactly, which is why it is
			// the one to declare: the row is well-formed, cites a real
			// section, names an owner, and is still false.
			Library: "Movies", Where: "10 Things I Hate About You (1999).mkv", Kind: nameDiffers,
			Reference: "Movie: 10 Things I Hate About You", Atrium: "Movie: 10 Things I Hate About You",
			Reason: "003 §3.3", Owner: "003",
			Because: "invented by TestADeclaredDifferenceThatHasGoneAwayFails",
		},
		{
			Library: "Movies", Where: "A Film Neither Server Has Ever Seen (1970).mkv", Kind: onlyInTheReference,
			Reference: "Movie: A Film Neither Server Has Ever Seen", Atrium: "",
			Reason: "003 §3.2", Owner: "003",
			Because: "invented by TestADeclaredDifferenceThatHasGoneAwayFails",
		},
	}

	for _, extra := range invented {
		t.Run(extra.key(), func(t *testing.T) {
			// The row is a well-formed declaration: what fails it is the
			// comparison and not the load, which is the distinction this test
			// is about.
			if problems := validateDeclarations(append(declaredDifferences(), extra)); len(problems) > 0 {
				t.Fatalf("the invented row does not even load, so this test would fail for the "+
					"wrong reason: %v", problems)
			}

			problems := problemsWith(append(declaredDifferences(), extra), observed)
			if len(problems) != 1 {
				t.Fatalf("declaring a difference the two readings do not have produced %d "+
					"problems and not one: %v", len(problems), problems)
			}
			if !strings.Contains(problems[0], "gone away") || !strings.Contains(problems[0], extra.Where) {
				t.Fatalf("declaring %s did not report it as a difference that has gone away; "+
					"it reported %q", extra.key(), problems[0])
			}
		})
	}
}

// TestADeclaredDifferenceThatChangedItsShapeFails is the third problem
// [problemsWith] can report, and the one 004 will meet first: a difference that
// is still there and no longer reads the way the row says.
//
// It matters because the row records both sides. Without that, a build that
// renamed `Pilot` to something else would keep a name difference at the same
// path, match the same declaration, and pass.
func TestADeclaredDifferenceThatChangedItsShapeFails(t *testing.T) {
	t.Parallel()

	observed := theWholeComparison(t, theReferencesReading(t), atriumsReading(t))

	moved := make([]declaredDifference, 0, len(declaredDifferences()))
	target := ""
	for _, row := range declaredDifferences() {
		if row.Kind == nameDiffers && target == "" {
			target = row.key()
			row.Atrium = "Episode: Something Else Entirely"
		}
		moved = append(moved, row)
	}
	if target == "" {
		t.Fatal("the declaration holds no name difference, so this test has nothing to move")
	}

	problems := problemsWith(moved, observed)
	if len(problems) != 1 || !strings.Contains(problems[0], "no longer reads the way the row says") {
		t.Fatalf("moving %s's Atrium side produced %v", target, problems)
	}
}

// --- The table's own load -----------------------------------------------------

// TestEveryRowNamesASpecificationSectionAndAnOwningFeature is
// docs/compatibility/allowlist.yaml's rule applied to a different file for the
// same reason, and it is asserted in both directions: the real table loads, and
// a row missing either half does not.
func TestEveryRowNamesASpecificationSectionAndAnOwningFeature(t *testing.T) {
	t.Parallel()

	if problems := validateDeclarations(declaredDifferences()); len(problems) > 0 {
		t.Fatalf("the declaration does not load:\n  %s", strings.Join(problems, "\n  "))
	}

	sound := declaredDifference{
		Library: "Movies", Where: "Wall-E (2008).mkv", Kind: nameDiffers,
		Reference: "Movie: Wall-E", Atrium: "Movie: WALL-E",
		Reason: "003 §3.3", Owner: "004", Because: "a well-formed row, for this test to spoil",
	}
	if problems := validateDeclarations([]declaredDifference{sound}); len(problems) > 0 {
		t.Fatalf("the row this test spoils does not load to begin with: %v", problems)
	}

	for _, spoiled := range []struct {
		what string
		row  declaredDifference
		says string
	}{
		{"no reason at all", func() declaredDifference { r := sound; r.Reason = ""; return r }(), "no reason as a specification section"},
		{"a reason that is not a section", func() declaredDifference { r := sound; r.Reason = "it seemed sensible"; return r }(), "no reason as a specification section"},
		{"no owner at all", func() declaredDifference { r := sound; r.Owner = ""; return r }(), "no owning feature"},
		{"an owner nobody recognises", func() declaredDifference { r := sound; r.Owner = "someone"; return r }(), "no owning feature"},
		{"neither", func() declaredDifference { r := sound; r.Reason, r.Owner = "", ""; return r }(), "no reason as a specification section"},
	} {
		t.Run(spoiled.what, func(t *testing.T) {
			problems := validateDeclarations([]declaredDifference{spoiled.row})
			if len(problems) == 0 {
				t.Fatalf("a row naming %s loaded", spoiled.what)
			}
			if !slices.ContainsFunc(problems, func(p string) bool { return strings.Contains(p, spoiled.says) }) {
				t.Fatalf("a row naming %s was refused for the wrong reason: %v", spoiled.what, problems)
			}
		})
	}
}

// TestTheTotalIsReadFromTheDeclarationsOwnLength is the assertion that a row
// deleted to make a run go green is a failing count rather than a quieter
// suite.
//
// It also states the arithmetic against 010's forty-seven, because a count this
// project cannot reproduce is a count nobody can check — the file's own comment
// carries the working.
func TestTheTotalIsReadFromTheDeclarationsOwnLength(t *testing.T) {
	t.Parallel()

	declared := declaredDifferences()
	if len(declared) != declaredDifferenceTotal {
		t.Fatalf("the declaration holds %d rows and declaredDifferenceTotal says %d. If a row "+
			"was removed, say what closed the difference; if one was added, say what opened it",
			len(declared), declaredDifferenceTotal)
	}

	byLibrary := map[string]int{}
	byOwner := map[string]int{}
	for _, row := range declared {
		byLibrary[row.Library]++
		byOwner[row.Owner]++
	}
	for library, want := range map[string]int{"Movies": 5, "Shows": 11, "Music": 15, "Empty": 1} {
		if byLibrary[library] != want {
			t.Errorf("%s holds %d declared differences and the table's own shape says %d",
				library, byLibrary[library], want)
		}
	}
	// 003 tasks' T17 and 003 plan §8.2: the rows 004's landing turns red are
	// counted here so that moving one between owners is a visible change.
	for owner, want := range map[string]int{"003": 9, "004": 23} {
		if byOwner[owner] != want {
			t.Errorf("%s owns %d declared differences and the table's own shape says %d",
				owner, byOwner[owner], want)
		}
	}
}

// TestTheFourShapes010NamesAreEachPresentWithTheRightOwner checks the
// declaration against the only prior description of it this project has:
// 010's own AC-2 amendment, which names four shapes and their owners
// `[probe: tools/probe_reference_scan.py, Jellyfin 10.11.11, 2026-09-02]`.
//
// Two of the four are counted rather than merely found, because "the zero-byte
// film is declared" is satisfied by a table holding one row and nothing else.
func TestTheFourShapes010NamesAreEachPresentWithTheRightOwner(t *testing.T) {
	t.Parallel()

	declared := declaredDifferences()
	find := func(library, whereAt string, kind differenceKind) (declaredDifference, bool) {
		for _, row := range declared {
			if row.Library == library && row.Where == whereAt && row.Kind == kind {
				return row, true
			}
		}
		return declaredDifference{}, false
	}

	// 1. The zero-byte film, owned by 003.
	zeroByte, found := find("Movies", "An Incomplete Copy (2000).mkv", onlyInTheReference)
	if !found {
		t.Fatal("the zero-byte film is not declared")
	}
	if zeroByte.Owner != "003" || !strings.Contains(zeroByte.Reason, "§3.2") {
		t.Errorf("the zero-byte film is declared as %s / %s, where 010 names it 003 §3.2",
			zeroByte.Owner, zeroByte.Reason)
	}

	// 2. The differently-named files, owned by 004. 010 counts twenty-five over
	// six libraries; twenty of those are over the four this feature builds, and
	// the file's own comment carries the arithmetic for the other five.
	names := 0
	for _, row := range declared {
		if row.Kind == nameDiffers && row.Owner == "004" && strings.Contains(row.Where, ".") {
			names++
		}
	}
	if names != 20 {
		t.Errorf("%d files are declared as differently named and owned by 004, where the four "+
			"libraries this feature builds hold 20 of 010's twenty-five", names)
	}

	// 3. The empty library, owned by 003.
	empty, found := find("Empty", theLibrarysOwnRow, onlyInAtrium)
	if !found {
		t.Fatal("the empty library — nothing at all to the reference — is not declared")
	}
	if empty.Owner != "003" || !strings.Contains(empty.Reason, "§3.1") {
		t.Errorf("the empty library is declared as %s / %s, where 010 names it 003 §3.1",
			empty.Owner, empty.Reason)
	}

	// 4. Every library's own root row. "Every" is the assertion: one row per
	// library the reading names and this feature builds, and no library without
	// one.
	roots := 0
	for _, row := range declared {
		if row.Where != theLibrarysOwnRow {
			continue
		}
		roots++
		if row.Owner != "003" || !strings.Contains(row.Reason, "§3.1") {
			t.Errorf("%s's own row is declared as %s / %s, where 010 names it 003 §3.1",
				row.Library, row.Owner, row.Reason)
		}
	}
	if want := len(libraryfixture.Libraries()); roots != want {
		t.Errorf("%d libraries have their own row declared as a difference, and this feature "+
			"builds %d — 010 names the shape for *every* library", roots, want)
	}
	for _, fixture := range libraryfixture.Libraries() {
		if _, found := find(fixture.Name, theLibrarysOwnRow, typeDiffers); found {
			continue
		}
		if _, found := find(fixture.Name, theLibrarysOwnRow, onlyInAtrium); !found {
			t.Errorf("%s's own row is not declared as a difference of either kind", fixture.Name)
		}
	}
}

// --- What the comparison is over ----------------------------------------------

// TestTheComparisonSurvivesEveryIdentifierMoving is the "type, name and path,
// never an identifier" clause, asserted rather than commented.
//
// behaviours §1.4 establishes that the two servers derive identifiers
// differently by design, so a comparison over them would declare 74 differences
// that say nothing. The recorded reading carries none at all, which makes the
// clause easy to satisfy by accident — so this asserts the property that would
// still hold if it did: **two installations of the same tree, whose library
// identifiers are freshly allocated and whose item identifiers therefore all
// differ, produce the same difference set.**
//
// The control is the first assertion: if the two installations' identifiers
// turned out to be the same strings, the test proves nothing and says so.
func TestTheComparisonSurvivesEveryIdentifierMoving(t *testing.T) {
	t.Parallel()

	first, firstIDs := aReadingAndItsIdentifiers(t)
	second, secondIDs := aReadingAndItsIdentifiers(t)

	shared := 0
	for id := range firstIDs {
		if secondIDs[id] {
			shared++
		}
	}
	if shared != 0 || len(firstIDs) == 0 {
		t.Fatalf("the two installations share %d of %d identifiers, so this test cannot tell a "+
			"comparison that reads them from one that does not", shared, len(firstIDs))
	}

	reference := theReferencesReading(t)
	if a, b := theWholeComparison(t, reference, first), theWholeComparison(t, reference, second); !slices.Equal(a, b) {
		t.Fatalf("two installations of one tree produced different difference sets:\n%v\n%v", a, b)
	}
}

// aReadingAndItsIdentifiers is [atriumsReading] with the identifiers it
// deliberately does not compare, so that a test can assert they moved.
func aReadingAndItsIdentifiers(t *testing.T) (map[string][]readingItem, map[string]bool) {
	t.Helper()

	data, libraries := theWholeFixture(t, t.TempDir())
	scanSummaries(t, data)

	byLibrary := make(map[string][]readingItem, len(libraries))
	identifiers := map[string]bool{}
	for _, lib := range libraries {
		items := make([]readingItem, 0, 32)
		for _, item := range storedItems(t, data, lib.ID) {
			items = append(items, readingItem{Type: item.Type, Name: item.Name, Path: item.Path})
			identifiers[item.ID] = true
		}
		byLibrary[lib.Name] = items
	}
	return byLibrary, identifiers
}

// TestNoTwoItemsOfEitherReadingDifferOnlyByCase is
// [U-44](../../docs/compatibility/reference-target.md), and it is here because
// **003 tasks' T17 asked for the case-insensitive pair to be among the
// forty-seven and it cannot be.**
//
// conformance §L2 lists "a name that differs only by case" among the cases the
// fixture covers, and U-44's own row says the resulting difference "is one of
// the forty-seven differences 003 declares over its own fixture". T1 measured
// the recorded reading and found no such pair; this asserts it of both readings
// at once, which is the half T1 could not reach.
//
// **Building the pair would not fix it.** Atrium would then hold an item the
// recorded reading has no row for — a difference in the wrong direction, added
// to make a number come out. U-44 stays what its register row otherwise says it
// is: a claim one scan against a single-use reference settles, and this feature
// leaves it unmeasured rather than manufacturing a thirty-third difference.
func TestNoTwoItemsOfEitherReadingDifferOnlyByCase(t *testing.T) {
	t.Parallel()

	for side, reading := range map[string]map[string][]readingItem{
		"the reference's": theReferencesReading(t),
		"Atrium's":        atriumsReading(t),
	} {
		for library, items := range reading {
			seen := map[string]readingItem{}
			for _, item := range items {
				folded := strings.ToLower(item.Name)
				previous, taken := seen[folded]
				if taken && previous.Name != item.Name {
					t.Errorf("%s %s reading holds %q and %q, which differ only by case — so "+
						"U-44 is observable over this fixture after all, and the declaration "+
						"and this test's own reasoning both need revisiting",
						side, library, previous.Name, item.Name)
				}
				seen[folded] = item
			}
		}
	}
}
