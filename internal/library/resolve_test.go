package library

import (
	"errors"
	"reflect"
	"testing"

	"github.com/vdatanet/atrium-go/internal/libraryfixture"
	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/units"
)

// These are `library`-level assertions, which is the weakest of 003 plan §8.1's
// three levels and the only one available to this task: `Resolve` is a pure
// function and there is no route, no store and no wire representation between
// it and anything a client would see.
//
// What that costs is written on each test that needs it, and the largest is at
// the foot of this file.

// aMoviesLibrary is the library every test here resolves against. Its identity
// is a literal because it is an **input** to every identifier below, and one
// generated per run would make a failure message that changes every time.
func aMoviesLibrary() ports.Library {
	return ports.Library{
		ID:             "0123456789abcdef0123456789abcdef",
		Name:           "Movies",
		NameFolded:     "movies",
		CollectionType: string(Movies),
	}
}

// aReading turns a list of paths into one root's reading, with a size and a
// modification time that differ per file so that nothing downstream can pass by
// treating them as interchangeable.
func aReading(root int, paths ...string) Reading {
	entries := make([]Entry, len(paths))
	for i, p := range paths {
		entries[i] = Entry{
			Path:       p,
			Size:       int64(1_000_000 + i),
			ModifiedAt: units.TimeFromTicks(units.Ticks(638_000_000_000_000_000 + int64(i))),
		}
	}
	return Reading{Root: root, Entries: entries}
}

// resolveMovieFiles resolves one root's reading and returns only the films,
// which is every item but the library's own row.
func resolveMovieFiles(t *testing.T, paths ...string) []ports.ScannedItem {
	t.Helper()
	plan, err := Resolve(aMoviesLibrary(), []Reading{aReading(0, paths...)})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	return filmsOf(t, plan)
}

func filmsOf(t *testing.T, plan Plan) []ports.ScannedItem {
	t.Helper()
	var films []ports.ScannedItem
	roots := 0
	for _, item := range plan.Items {
		switch Kind(item.Type) {
		case KindMovie:
			films = append(films, item)
		case KindCollectionFolder:
			roots++
		default:
			t.Fatalf("a movies library resolved an item of type %q", item.Type)
		}
	}
	if roots != 1 {
		t.Fatalf("the plan carries %d library rows, want exactly 1", roots)
	}
	return films
}

func namesOf(items []ports.ScannedItem) []string {
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = item.Name
	}
	return out
}

// referenceNameOfTheStackedFilm is what the reference's own reading of this
// repository's fixture tree calls the multi-part film
// `[probe: tools/probe_reference_scan.py, Jellyfin 10.11.11, 2026-09-02]`.
//
// It is a literal here rather than a read of
// `docs/compatibility/reference-fixture-reading.json`, because what this file
// asserts is a **declared inequality** and a declaration that reads itself out
// of the thing it is declared against cannot be one.
const referenceNameOfTheStackedFilm = "The Long Film (1998)"

// TestAMultiPartFilmIsOneItemWhoseNameCameFromItsDirectory is AC-4's first
// half over the fixture's own tree: two parts, one item, the parts in ordinal
// order, and the item named after the directory rather than after a file.
//
// The count is asserted first, because two items each carrying one part
// satisfies every per-field assertion anybody would write about a part.
//
// **What this cannot assert is where the name came from.** Over this tree the
// directory rule and the year rule each repair a name taken from the first
// part, so the mutation that takes it passes everything here. The test below
// makes that assertion on a tree where nothing can.
func TestAMultiPartFilmIsOneItemWhoseNameCameFromItsDirectory(t *testing.T) {
	const (
		partOne = "The Long Film (1998)/The Long Film (1998) - part1.mkv"
		partTwo = "The Long Film (1998)/The Long Film (1998) - part2.mkv"
	)

	films := resolveMovieFiles(t, partOne, partTwo)

	if len(films) != 1 {
		t.Fatalf("resolved %d films from two parts, want 1: %q", len(films), namesOf(films))
	}
	film := films[0]

	if film.Name == "The Long Film (1998) - part1" {
		t.Fatal("the item was named after its first part, so the stack was built and then discarded")
	}
	if got, want := delimiter+film.Name+delimiter, delimiter+"The Long Film"+delimiter; got != want {
		t.Errorf("name = %s, want %s", got, want)
	}
	if film.ProductionYear == nil || *film.ProductionYear != 1998 {
		t.Errorf("production year = %v, want 1998", film.ProductionYear)
	}

	if len(film.Files) != 2 {
		t.Fatalf("the item carries %d files, want 2", len(film.Files))
	}
	if film.Files[0].Path != partOne || film.Files[1].Path != partTwo {
		t.Errorf("parts in %q, %q, want part1 then part2", film.Files[0].Path, film.Files[1].Path)
	}
	if film.Files[0].Ordinal != 1 || film.Files[1].Ordinal != 2 {
		t.Errorf("part ordinals %d, %d, want 1, 2", film.Files[0].Ordinal, film.Files[1].Ordinal)
	}
	if film.Path != partOne {
		t.Errorf("the item's path is %q, want the first part's — it is what the identifier derives from", film.Path)
	}
}

// TestAStacksNameComesFromTheStackAndNotFromItsFirstPart is the assertion the
// test above **cannot** make, and finding that out is this task's own closing
// finding.
//
// 003's task list asks for the name of `The Long Film (1998)/… - part1.mkv` as
// the thing that catches a build which stacked the parts and then took the
// first file's stem. Over that tree it catches nothing, because two other rules
// repair the name before anybody looks at it: the directory holds exactly one
// film and names it, and — even with that rule removed — the year extraction
// discards everything after `(1998)`, `- part1` included. The mutation that
// takes the first part's stem passes every assertion above.
//
// So the assertion is made where nothing can repair it: two parts directly
// under the library root, with the year out of the way. The name must be the
// **stack's** title, and a build that reaches for `Files[0]` says
// `The Long Film - part1`.
func TestAStacksNameComesFromTheStackAndNotFromItsFirstPart(t *testing.T) {
	films := resolveMovieFiles(t, "The Long Film - part1.mkv", "The Long Film - part2.mkv")

	if len(films) != 1 {
		t.Fatalf("resolved %d films, want 1: %q", len(films), namesOf(films))
	}
	if films[0].Name == "The Long Film - part1" {
		t.Fatal("the item was named after its first part; no directory and no year stood in the way this time")
	}
	if got, want := delimiter+films[0].Name+delimiter, delimiter+"The Long Film"+delimiter; got != want {
		t.Errorf("name = %s, want %s", got, want)
	}
}

// TestTheReferenceNamesTheSameStackedFilmAfterTheSameDirectory is the declared
// inequality beside the assertion above.
//
// Both servers name that item after its **directory** and neither after a
// file, which is the agreement worth having. They differ by exactly one thing:
// the reference keeps the directory's name whole and Atrium takes the year out
// of it, because 003 §3.3 says a film's name is its title with the year
// extracted. That is one of the forty-seven differences 003 declares over its
// own fixture (plan §8.2), and it is two rows there rather than one — this film
// and the folder-per-film `The Matrix (1999)`.
//
// Asserting it as an equation between the two names, rather than as a comment,
// is what makes it fail if either half moves.
func TestTheReferenceNamesTheSameStackedFilmAfterTheSameDirectory(t *testing.T) {
	films := resolveMovieFiles(t,
		"The Long Film (1998)/The Long Film (1998) - part1.mkv",
		"The Long Film (1998)/The Long Film (1998) - part2.mkv",
	)
	if len(films) != 1 {
		t.Fatalf("resolved %d films, want 1", len(films))
	}

	if films[0].Name == referenceNameOfTheStackedFilm {
		t.Fatalf("the name now equals the reference's %q; the declared difference has gone away, "+
			"which fails as loudly as an undeclared one (plan §8.2)", referenceNameOfTheStackedFilm)
	}

	name, year := cleanVideoName(referenceNameOfTheStackedFilm)
	if name != films[0].Name {
		t.Errorf("the reference's name cleaned is %q and Atrium's is %q; the two servers are no longer "+
			"naming this film after the same string", name, films[0].Name)
	}
	if year == nil || films[0].ProductionYear == nil || *year != *films[0].ProductionYear {
		t.Errorf("the year the reference's name carries is %v and Atrium's is %v", year, films[0].ProductionYear)
	}
}

// TestABareTrailingLetterIsTwoFilmsAndACdLetterIsOne is U-43, asserted as the
// divergence it is registered as.
//
// The two shapes are one line apart on purpose. `- a`/`- b` are **two** works
// and `- cda`/`- cdb` are **one**, and 003 §3.3's withdrawn parenthetical said
// the first pair was one item too. That reading is the only one in this feature
// that **loses** an item: two films merged, the second gone, which is worse
// than the doubling §3.3 spends its warning on.
//
// Neither shape has been sent to a running reference — it is a source reading
// `[source: Emby.Naming/Common/NamingOptions.cs:141-145 @ v10.11.11]` and
// U-43 is open for exactly that. **This test is written to go red the day
// somebody measures it and finds otherwise**, which is the notification the
// register is worth having.
func TestABareTrailingLetterIsTwoFilmsAndACdLetterIsOne(t *testing.T) {
	films := resolveMovieFiles(t,
		"The Film - a.mkv",
		"The Film - b.mkv",
		"The Other Film - cda.mkv",
		"The Other Film - cdb.mkv",
	)

	if len(films) != 3 {
		t.Fatalf("resolved %d films, want 3 — two from the bare letters and one from the `cd` pair: %q",
			len(films), namesOf(films))
	}

	if got, want := namesOf(films), []string{"The Film - a", "The Film - b", "The Other Film"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("names %q, want %q", got, want)
	}

	bareA, bareB, stacked := films[0], films[1], films[2]
	if len(bareA.Files) != 1 || len(bareB.Files) != 1 {
		t.Errorf("the bare-letter films carry %d and %d files, want one each — U-43 has moved",
			len(bareA.Files), len(bareB.Files))
	}
	if bareA.ID == bareB.ID {
		t.Error("the two bare-letter films share an identifier, so they are one item however they are counted")
	}
	if len(stacked.Files) != 2 {
		t.Errorf("the `cd` pair carries %d files, want 2 — U-43 has moved the other way", len(stacked.Files))
	}
	if stacked.Files[0].Ordinal != 1 || stacked.Files[1].Ordinal != 2 {
		t.Errorf("`cda` and `cdb` are parts %d and %d, want 1 and 2",
			stacked.Files[0].Ordinal, stacked.Files[1].Ordinal)
	}
}

// TestADirectoryHoldingTwoTitlesNamesNeither is 003 §3.3's *"only part of the
// rule a single path cannot decide"*, and its companion is the test below it:
// the two trees differ by exactly one file.
func TestADirectoryHoldingTwoTitlesNamesNeither(t *testing.T) {
	films := resolveMovieFiles(t,
		"Some Collection/The First Film (1990).mkv",
		"Some Collection/The Second Film (1991).mkv",
	)

	if got, want := namesOf(films), []string{"The First Film", "The Second Film"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("names %q, want %q — a directory holding two titles is a category and names neither", got, want)
	}
	for _, film := range films {
		if film.Name == "Some Collection" {
			t.Errorf("%q took the category's name", film.Path)
		}
	}
}

// TestADirectoryHoldingOneFilmNamesIt is the same tree with one file removed,
// and it is where the measurement lives: across 1,557 films the directory's
// cleaned name matched what the reference resolved 1,087 times against the
// file's 457 `[read: Jellyfin 10.11.11, 2026-08-27]`.
//
// The filename here disagrees with the directory on purpose. A build that took
// the filename would pass every count and every identifier in this file.
func TestADirectoryHoldingOneFilmNamesIt(t *testing.T) {
	films := resolveMovieFiles(t, "The Real Title (1990)/mangled.by.a.download.tool.mkv")

	if len(films) != 1 {
		t.Fatalf("resolved %d films, want 1", len(films))
	}
	if got, want := films[0].Name, "The Real Title"; got != want {
		t.Errorf("name = %q, want %q — the directory names a film that sits in its own directory", got, want)
	}
	if films[0].ProductionYear == nil || *films[0].ProductionYear != 1990 {
		t.Errorf("production year = %v, want 1990 — the year comes out of the directory with the name",
			films[0].ProductionYear)
	}
}

// TestALibraryRootNeverNamesAFilm is the third of the three, and it is the one
// a reading of the rule as *"the containing directory"* gets wrong.
//
// A library holding exactly one film directly under its root has a directory
// holding exactly one film. The root is the library and not a folder-per-film,
// 003 plan §4.2 gives the library's own row no path for the same reason, and a
// build without this exclusion names that film after whatever an operator
// called the mount point.
func TestALibraryRootNeverNamesAFilm(t *testing.T) {
	films := resolveMovieFiles(t, "The Only Film (1990).mkv")

	if len(films) != 1 {
		t.Fatalf("resolved %d films, want 1", len(films))
	}
	if got, want := films[0].Name, "The Only Film"; got != want {
		t.Errorf("name = %q, want %q", got, want)
	}
}

// TestAStackNeedsTwoPartsAndOneMarkerWord walks the three rules that keep a
// stack from swallowing a film it should not.
func TestAStackNeedsTwoPartsAndOneMarkerWord(t *testing.T) {
	t.Run("a stack of one is not a stack", func(t *testing.T) {
		films := resolveMovieFiles(t, "The Film - cd1.mkv")
		if len(films) != 1 {
			t.Fatalf("resolved %d films, want 1", len(films))
		}
		if len(films[0].Files) != 1 {
			t.Fatalf("the item carries %d files, want 1", len(films[0].Files))
		}
		if got := films[0].Files[0].Ordinal; got != 0 {
			t.Errorf("the one file's ordinal is %d, want 0 — it is not a part of anything", got)
		}
		if got, want := delimiter+films[0].Name+delimiter, delimiter+"The Film -"+delimiter; got != want {
			t.Errorf("name = %s, want %s — a lone `cd1` is release-tag noise and the cut lands on the separator "+
				"before it, which is the reference's own shape", got, want)
		}
	})

	t.Run("two marker words do not stack together", func(t *testing.T) {
		films := resolveMovieFiles(t, "The Film - cd1.mkv", "The Film - part2.mkv")
		if len(films) != 2 {
			t.Fatalf("resolved %d films, want 2 — `cd` and `part` are different stacks", len(films))
		}
	})

	t.Run("a numeric marker does not join an alphabetic stack", func(t *testing.T) {
		films := resolveMovieFiles(t, "The Film - cd1.mkv", "The Film - cdb.mkv")
		if len(films) != 2 {
			t.Fatalf("resolved %d films, want 2 — one stack is numeric and the other is not: %q",
				len(films), namesOf(films))
		}
	})

	t.Run("a repeated part number joins nothing", func(t *testing.T) {
		caseSensitive := aMoviesLibrary()
		caseSensitive.CaseSensitive = true

		plan, err := Resolve(caseSensitive, []Reading{aReading(0,
			"The Film - CD1.mkv",
			"The Film - cd1.mkv",
		)})
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		films := filmsOf(t, plan)
		if len(films) != 2 {
			t.Fatalf("resolved %d films, want 2 — a stack with two first parts has lost a file, not gained one",
				len(films))
		}
	})

	t.Run("parts are ordered by their number and not by their path", func(t *testing.T) {
		films := resolveMovieFiles(t, "The Film - part10.mkv", "The Film - part2.mkv")
		if len(films) != 1 {
			t.Fatalf("resolved %d films, want 1", len(films))
		}
		files := films[0].Files
		if len(files) != 2 {
			t.Fatalf("the item carries %d files, want 2", len(files))
		}
		// `part10` sorts before `part2` by bytes, so a build that ordered the
		// parts by path gets this pair backwards and every pair below ten
		// right — which is why the pair is written with a ten in it.
		if files[0].Path != "The Film - part2.mkv" || files[1].Path != "The Film - part10.mkv" {
			t.Errorf("parts in %q, %q, want part2 then part10", files[0].Path, files[1].Path)
		}
		if files[0].Ordinal != 1 || files[1].Ordinal != 2 {
			t.Errorf("ordinals %d, %d, want 1, 2 — the ordinal is the position, not the number written on the file",
				files[0].Ordinal, files[1].Ordinal)
		}
	})

	t.Run("two directories do not share a stack", func(t *testing.T) {
		films := resolveMovieFiles(t,
			"A/Feature - cd1.mkv", "A/Feature - cd2.mkv",
			"B/Feature - cd1.mkv", "B/Feature - cd2.mkv",
		)
		if len(films) != 2 {
			t.Fatalf("resolved %d films, want 2 — one per directory", len(films))
		}
		for _, film := range films {
			if len(film.Files) != 2 {
				t.Errorf("%q carries %d files, want 2", film.Path, len(film.Files))
			}
		}
	})
}

// TestTwoReadingsBuiltInOppositeOrdersProduceTheIdenticalPlan is Principle VII
// at the layer where insertion order can still get in, and it is the one thing
// a per-path resolver could not have.
//
// Both the order of the roots and the order of the entries inside each root are
// reversed, because a resolver that sorted one and not the other passes half of
// this. The comparison is on the whole plan — identifiers, names, parents,
// **part order** and every file's size — rather than on a list of names, since
// a name list is stable under exactly the reordering that would break a part.
func TestTwoReadingsBuiltInOppositeOrdersProduceTheIdenticalPlan(t *testing.T) {
	first := []Reading{
		aReading(0,
			"The Long Film (1998)/The Long Film (1998) - part1.mkv",
			"The Long Film (1998)/The Long Film (1998) - part2.mkv",
			"The Long Film (1998)/The Long Film (1998) - part3.mkv",
			"Some Collection/The First Film (1990).mkv",
			"Some Collection/The Second Film (1991).mkv",
			// The trio that makes the *entry* order matter rather than only
			// the output order. Sorted, `cd1` arrives first and establishes a
			// numeric stack that `cda` cannot join, which is two items; with
			// `cda` first the stack is alphabetic, both numbered parts stand
			// alone and it is three. Nothing else in this tree can tell the
			// two readings apart.
			"Mixed/Feature - cd1.mkv",
			"Mixed/Feature - cd2.mkv",
			"Mixed/Feature - cda.mkv",
		),
		aReading(1, "Another Root/The Third Film (1992).mkv", "Loose (1993).mkv"),
	}

	second := reverseReadings(first)

	forward, err := Resolve(aMoviesLibrary(), first)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	backward, err := Resolve(aMoviesLibrary(), second)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if !reflect.DeepEqual(forward, backward) {
		t.Fatalf("the two plans differ:\n forward = %+v\nbackward = %+v", forward, backward)
	}

	if got := len(filmsOf(t, forward)); got != 7 {
		t.Fatalf("the tree resolved to %d films, want 7; the mixed-marker trio is what makes the "+
			"entry order observable and a change to it would make this test agree with itself for "+
			"the wrong reason", got)
	}

	films := filmsOf(t, forward)
	var stacked *ports.ScannedItem
	for i := range films {
		if len(films[i].Files) == 3 {
			stacked = &films[i]
		}
	}
	if stacked == nil {
		t.Fatal("no three-part film in the plan, so the part order this test is about was never built")
	}
	for i, file := range stacked.Files {
		if file.Ordinal != i+1 {
			t.Errorf("part %d has ordinal %d, want %d", i, file.Ordinal, i+1)
		}
	}
}

// TestTheReadingIsSortedBeforeAnythingLooksAtItAndTheCallersSliceIsNot is the
// same property as the test above, one layer down, and it is here because the
// plan-level assertion cannot see it: the items are sorted again on the way
// out, so a resolver that never sorted its input at all still produces one
// stable answer for most trees.
//
// Two things are asserted apart. The readings and their entries come back
// **sorted**, which is 003 plan §5's *"the reading is sorted on the path before
// anything looks at it"*. And the caller's own slices come back **untouched**,
// because the walk that built them may still be holding one and a function that
// reorders its argument is a function nobody can call twice.
func TestTheReadingIsSortedBeforeAnythingLooksAtItAndTheCallersSliceIsNot(t *testing.T) {
	original := []Reading{
		aReading(2, "c.mkv", "a.mkv"),
		aReading(0, "z.mkv", "b.mkv"),
	}
	before := reflect.DeepEqual(original, cloneReadings(original))
	if !before {
		t.Fatal("cloneReadings does not clone, so the assertion below proves nothing")
	}
	untouched := cloneReadings(original)

	sorted := sortReadings(original)

	if len(sorted) != 2 || sorted[0].Root != 0 || sorted[1].Root != 2 {
		t.Fatalf("roots came back in order %d, %d, want 0, 2", sorted[0].Root, sorted[1].Root)
	}
	if got := []string{sorted[0].Entries[0].Path, sorted[0].Entries[1].Path}; !reflect.DeepEqual(got, []string{"b.mkv", "z.mkv"}) {
		t.Errorf("root 0's entries came back %q, want b then z", got)
	}
	if got := []string{sorted[1].Entries[0].Path, sorted[1].Entries[1].Path}; !reflect.DeepEqual(got, []string{"a.mkv", "c.mkv"}) {
		t.Errorf("root 2's entries came back %q, want a then c", got)
	}
	if !reflect.DeepEqual(original, untouched) {
		t.Errorf("the caller's readings were reordered underneath it:\n got = %+v\nwant = %+v", original, untouched)
	}
}

// cloneReadings copies a reading list deeply enough for the comparison above.
func cloneReadings(readings []Reading) []Reading {
	out := make([]Reading, len(readings))
	for i, reading := range readings {
		entries := make([]Entry, len(reading.Entries))
		copy(entries, reading.Entries)
		out[i] = Reading{Root: reading.Root, Entries: entries}
	}
	return out
}

// reverseReadings reverses the roots and every root's entries.
func reverseReadings(readings []Reading) []Reading {
	out := make([]Reading, 0, len(readings))
	for i := len(readings) - 1; i >= 0; i-- {
		entries := make([]Entry, 0, len(readings[i].Entries))
		for j := len(readings[i].Entries) - 1; j >= 0; j-- {
			entries = append(entries, readings[i].Entries[j])
		}
		out = append(out, Reading{Root: readings[i].Root, Entries: entries})
	}
	return out
}

// TestTheLibrarysOwnRowIsDerivedFromItsIdentityAndCarriesNoPath asserts the one
// item every plan has whether or not the tree holds a file.
func TestTheLibrarysOwnRowIsDerivedFromItsIdentityAndCarriesNoPath(t *testing.T) {
	lib := aMoviesLibrary()
	plan, err := Resolve(lib, nil)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(plan.Items) != 1 {
		t.Fatalf("an empty library resolved to %d items, want 1", len(plan.Items))
	}

	row := plan.Items[0]
	if Kind(row.Type) != KindCollectionFolder {
		t.Errorf("type = %q, want %q", row.Type, KindCollectionFolder)
	}
	if row.Name != lib.Name {
		t.Errorf("name = %q, want the library's own %q", row.Name, lib.Name)
	}
	if row.Path != "" || row.ParentID != "" {
		t.Errorf("path %q and parent %q, want both empty: a library may have several roots and is under nothing",
			row.Path, row.ParentID)
	}
	if want := DeriveID(lib.ID, KindCollectionFolder, lib.ID); row.ID != want {
		t.Errorf("identifier = %q, want %q", row.ID, want)
	}

	moved := lib
	moved.Roots = []string{"/somewhere/else"}
	movedPlan, err := Resolve(moved, nil)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if movedPlan.Items[0].ID != row.ID {
		t.Error("the library's own identifier moved when its roots did")
	}
}

// TestEveryFilmIsAChildOfTheLibrarysOwnRow is the parent-child half of AC-1,
// as far as this feature can take it.
func TestEveryFilmIsAChildOfTheLibrarysOwnRow(t *testing.T) {
	plan, err := Resolve(aMoviesLibrary(), []Reading{aReading(0,
		"The Only Film (1990).mkv",
		"A Folder (1991)/inside.mkv",
	)})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	root := plan.Items[0]
	for _, film := range filmsOf(t, plan) {
		if film.ParentID != root.ID {
			t.Errorf("%q has parent %q, want the library's row %q", film.Path, film.ParentID, root.ID)
		}
	}
}

// TestAPathThatWillNotNormaliseFailsTheWholeLibrary is 003 plan §7's row, and
// the reason it is here rather than at the walk is the one the identity task's
// handoff spelled out: `/Movies/The Matrix (1999).mkv` is a **candidate** by
// every path rule in §3.2 — nothing about it is hidden, an extra or a refused
// extension — and `Normalise` refuses it. A resolver that consulted only the
// skip vocabulary would write an item under a root it does not have.
func TestAPathThatWillNotNormaliseFailsTheWholeLibrary(t *testing.T) {
	const absolute = "/Movies/The Matrix (1999).mkv"

	if ok, skip := Movies.Candidate(absolute); !ok {
		t.Fatalf("Candidate(%q) refused it as %v; this test is no longer about the case it is named for",
			absolute, skip)
	}

	plan, err := Resolve(aMoviesLibrary(), []Reading{aReading(0, "The Only Film (1990).mkv", absolute)})
	if !errors.Is(err, ErrPathAbsolute) {
		t.Fatalf("error = %v, want %v", err, ErrPathAbsolute)
	}
	if !reflect.DeepEqual(plan, Plan{}) {
		t.Errorf("a refused library returned %d items; there is no partial plan", len(plan.Items))
	}

	var pathErr *PathError
	if !errors.As(err, &pathErr) || pathErr.Path != absolute {
		t.Errorf("the error does not name the path it refused: %v", err)
	}
}

// TestAPathThatIsNotACandidateIsANoteAndNotAnItem asserts the other channel.
// A skip refuses one file and is not a fault; a key that will not normalise
// fails the library. The two are different types on purpose.
func TestAPathThatIsNotACandidateIsANoteAndNotAnItem(t *testing.T) {
	plan, err := Resolve(aMoviesLibrary(), []Reading{aReading(0,
		"The Only Film (1990)/poster.jpg",
		"The Only Film (1990)/The Only Film (1990).mkv",
	)})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	films := filmsOf(t, plan)
	if len(films) != 1 {
		t.Fatalf("resolved %d films, want 1", len(films))
	}
	if len(plan.Skipped) != 1 || plan.Skipped[0].Path != "The Only Film (1990)/poster.jpg" {
		t.Fatalf("skipped notes = %+v, want the one poster", plan.Skipped)
	}
	if got, want := plan.Skipped[0].Reason, SkipExtension.String(); got != want {
		t.Errorf("reason = %q, want %q", got, want)
	}
	if len(plan.Unplaceable) != 0 {
		t.Errorf("unplaceable = %+v, want none: nothing under `movies` produces one", plan.Unplaceable)
	}

	// And the poster did not count towards the directory's one film, which is
	// the failure that would rename it.
	if films[0].Name != "The Only Film" {
		t.Errorf("name = %q, want %q — the refused file was counted as a second title", films[0].Name, "The Only Film")
	}
}

// TestEveryCollectionTypeResolvesAFileOfItsOwn replaces the refusal that stood
// here while `music` had no resolver, and it is a **reachable** assertion where
// that one was an unreachable branch.
//
// The refusal existed for Principle VI's *"no plausible-looking stub"* at the
// one place in this feature where a stub would be silent and destructive: an
// empty plan is the answer *"this library holds nothing"*, and the caller that
// believes it removes every item under it. All three resolvers are written
// now, so the error nobody can reach is itself the stub — one layer up — and
// it is gone.
//
// What is left is the property it was protecting, asserted where a fourth
// collection type would actually meet it: every type
// [AllCollectionTypes] names resolves a file of its own admitted extension
// into an item beyond the library's own row. A type added to that list with no
// arm in [Resolve]'s switch fails here rather than quietly emptying a library.
func TestEveryCollectionTypeResolvesAFileOfItsOwn(t *testing.T) {
	for _, collection := range AllCollectionTypes() {
		extensions := collection.Extensions()
		if len(extensions) == 0 {
			t.Fatalf("%s admits no extension, so this test cannot hand it a file", collection)
		}

		lib := aMoviesLibrary()
		lib.CollectionType = string(collection)

		plan, err := Resolve(lib, []Reading{aReading(0, "A Name/Another Name/A File"+extensions[0])})
		if err != nil {
			t.Errorf("%s: Resolve: %v", collection, err)
			continue
		}
		if len(plan.Items) < 2 {
			t.Errorf("%s: resolved %d items — the library's own row and nothing else, which is the answer "+
				"\"this library holds nothing\" and the caller that believes it removes everything",
				collection, len(plan.Items))
		}
	}
}

func TestACollectionTypeThatIsNotOneOfTheThreeIsRefused(t *testing.T) {
	lib := aMoviesLibrary()
	lib.CollectionType = "books"

	if _, err := Resolve(lib, nil); !errors.Is(err, ErrCollectionTypeUnknown) {
		t.Errorf("error = %v, want %v", err, ErrCollectionTypeUnknown)
	}
}

// TestTheFixturesFilmsResolveToTheExpectedRows compares this resolver against
// the item set 003 plan §8.5 keeps as a **literal**, for the four Movies rows
// whose names are decisions the specification does not state.
//
// The two sides were written independently — the expectation with the fixture
// tree, this resolver afterwards — and the reading below is a literal too,
// because deriving it from `libraryfixture.Libraries()` would make this a test
// of arithmetic. What it is *not* is a scan: the ten files the fixture holds
// that no row names are absent here because a **walk** drops them, and the walk
// is not this task's. So this asserts what the resolver does with the files a
// walk would hand it, and nothing about which files that is.
func TestTheFixturesFilmsResolveToTheExpectedRows(t *testing.T) {
	// Every candidate file the fixture's `Movies` library holds, in the order
	// a sorted walk yields them. The zero-byte film and everything under an
	// exclusion rule is deliberately absent; see this test's own comment.
	reading := aReading(0,
		"  Padded   (1999).mkv",
		"10 Things I Hate About You (1999).mkv",
		"100% Wolf (2020).mkv",
		"2 Fast 2 Furious (2003).mkv",
		"A Bridge Too Far (1977).mkv",
		"A Broadcast Capture (2011).ts",
		"A Newer Transfer (2015).mp4",
		"Amélie (2001).mkv",
		"An Old Transfer (1985).avi",
		"Don't Look Up (2021).mkv",
		"Rock & Roll (1978).mkv",
		"S.W.A.T. (2003).mkv",
		"The Long Film (1998)/The Long Film (1998) - part1.mkv",
		"The Long Film (1998)/The Long Film (1998) - part2.mkv",
		"The Matrix (1999)/The Matrix (1999).mkv",
		"Wall-E (2008).mkv",
		"iRobot (2004).mkv",
	)

	lib := aMoviesLibrary()
	plan, err := Resolve(lib, []Reading{reading})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	var want []libraryfixture.ExpectedItem
	for _, row := range libraryfixture.ExpectedItems() {
		if row.Library == "Movies" {
			want = append(want, row)
		}
	}
	if len(plan.Items) != len(want) {
		t.Fatalf("resolved %d items, want %d: %q", len(plan.Items), len(want), namesOf(plan.Items))
	}

	for i, row := range want {
		got := plan.Items[i]
		if got.Type != row.Type {
			t.Errorf("item %d (%q): type %q, want %q", i, row.Name, got.Type, row.Type)
		}
		if delimiter+got.Name+delimiter != delimiter+row.Name+delimiter {
			t.Errorf("item %d: name |%s|, want |%s|", i, got.Name, row.Name)
		}
		if got.Path != row.Path {
			t.Errorf("item %d (%q): path %q, want %q", i, row.Name, got.Path, row.Path)
		}
		if got.Unplaceable != row.Unplaceable {
			t.Errorf("item %d (%q): unplaceable %v, want %v", i, row.Name, got.Unplaceable, row.Unplaceable)
		}

		wantParent := ""
		if row.Parent == libraryfixture.LibraryRoot {
			wantParent = plan.Items[0].ID
		}
		if got.ParentID != wantParent {
			t.Errorf("item %d (%q): parent %q, want %q", i, row.Name, got.ParentID, wantParent)
		}
	}
}

// TestAnItemsPathIsTheOneTheWalkReadAndNotTheFoldedKey is the distinction the
// identifier and the path are on either side of, and it is a correction to
// `ports.ScannedItem.Path`'s own first wording.
//
// The **key** an identifier derives from is case-folded in a case-insensitive
// library. The **path** is not: it is what an operator sees, what a media
// source will point at, and what something will eventually open. A stored path
// that had been lowercased cannot be opened on a case-sensitive filesystem at
// all, and nothing in this feature would notice, because every test that
// resolves a lower-case tree agrees with both.
func TestAnItemsPathIsTheOneTheWalkReadAndNotTheFoldedKey(t *testing.T) {
	const onDisk = "Mixed Case Folder (1990)/Mixed Case File.mkv"

	insensitive := aMoviesLibrary()
	films := resolveMovieFiles(t, onDisk)
	if len(films) != 1 {
		t.Fatalf("resolved %d films, want 1", len(films))
	}
	if films[0].Path != onDisk {
		t.Errorf("path = %q, want %q exactly as the reading carried it", films[0].Path, onDisk)
	}
	if films[0].Files[0].Path != onDisk {
		t.Errorf("the file's path = %q, want %q", films[0].Files[0].Path, onDisk)
	}

	key, err := Normalise(onDisk, insensitive.CaseSensitive)
	if err != nil {
		t.Fatalf("Normalise: %v", err)
	}
	if key == onDisk {
		t.Fatal("the normalised key equals the path, so this test cannot tell the two apart")
	}
	if want := DeriveID(insensitive.ID, KindMovie, key); films[0].ID != want {
		t.Errorf("identifier = %q, want the one derived from the folded key %q", films[0].ID, want)
	}
}
