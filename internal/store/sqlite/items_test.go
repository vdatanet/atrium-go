package sqlite

// The derived half's two ports, at the store.
//
// # What a green run in this file is not evidence for
//
// 003 plan §8.3 rows 2 and 4 are open after every assertion here, and they are
// 005's. **Nothing in this file proves that any `ORDER BY` a client can reach
// uses `sort_key`**, and nothing proves that the four numeric columns travel as
// integers on the wire. What is proven is one level below both: that the column
// is compared as bytes when this package orders on it, and that the four
// numbers survive a round trip as numbers with absent and zero told apart.
//
// The rows are stated here as well as in the plan because this is the file
// somebody will point at when 005 asks whether the ordering is already covered.

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/vdatanet/atrium-go/internal/library"
	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/units"
)

// The instants these tests claim and renew at. They are far apart enough that a
// stale-claim boundary can sit between two of them and be named rather than
// computed.
var (
	aScanInstant  = units.TimeFromTicks(638_000_000_000_000_000)
	aTickLater    = units.TimeFromTicks(638_000_000_000_000_001)
	aMinuteLater  = units.TimeFromTicks(638_000_000_600_000_000)
	anHourLater   = units.TimeFromTicks(638_000_036_000_000_000)
	oneMinute     = units.Ticks(600_000_000)
	twoScanners   = [2]string{"scanner-a", "scanner-b"}
	noScanSummary []byte
)

// anItem builds an item of a type that keys through SortKeyBase, with the sort
// key derived rather than typed.
//
// The key is never a literal in this file. It is what `library.SortKeyFor`
// produces, because the column exists to hold that derivation and a test that
// wrote its own would be asserting the store against a second implementation of
// 003 §3.7.
//
// **That is right for this file and is exactly why it proves nothing about a
// scan.** A build whose scanner stores something other than the derivation
// leaves `SortKeyFor` correct, so every expected value here moves with it. 003
// T20's closing audit found that gap and closed it one layer up, in
// `internal/app/library_sortkey_table_test.go`, where the expected keys are
// literals and a guard fails if that file ever names this derivation.
func anItem(id, libraryID, kind, name, path string) ports.ScannedItem {
	item := ports.ScannedItem{
		ID:        id,
		LibraryID: libraryID,
		Type:      kind,
		Name:      name,
		Path:      path,
	}
	item.SortKey = library.SortKeyFor(&item)
	return item
}

// anEpisode builds an episode, whose sort key is 003 §3.7.2's numeric prefix
// followed by the **raw** name — which is the only reason this file can say
// anything about a collation at all. Everything keyed through `SortKeyBase` is
// lowercased on the way in, so `NOCASE` and `BINARY` would agree over it.
func anEpisode(id, libraryID, name, path string, season, number int) ports.ScannedItem {
	item := ports.ScannedItem{
		ID:                id,
		LibraryID:         libraryID,
		Type:              string(library.KindEpisode),
		Name:              name,
		Path:              path,
		ParentIndexNumber: &season,
		IndexNumber:       &number,
	}
	item.SortKey = library.SortKeyFor(&item)
	return item
}

// claimed takes the claim these tests write under, and fails the test rather
// than the batch when it could not.
func claimed(t *testing.T, store *Store, libraryID, by string) {
	t.Helper()

	won, previous, err := store.ClaimScan(context.Background(), libraryID, by, aScanInstant, oneMinute)
	if err != nil {
		t.Fatalf("ClaimScan returned %v", err)
	}
	if !won {
		t.Fatalf("ClaimScan lost to %q, want the claim: nothing else has taken it", previous)
	}
}

// applied writes a batch under a held claim and fails the test if it did not.
func applied(t *testing.T, store *Store, libraryID, by string, at units.Time, items ...ports.ScannedItem) {
	t.Helper()

	if err := store.ApplyScanBatch(context.Background(), ports.ScanBatch{
		LibraryID: libraryID, Items: items, ClaimedBy: by, At: at,
	}); err != nil {
		t.Fatalf("ApplyScanBatch returned %v", err)
	}
}

// identifiersOf names a set of items in the order they arrived, for a failure
// message that says which order went wrong rather than that one did.
func identifiersOf(items []ports.ScannedItem) []string {
	names := make([]string, len(items))
	for i, item := range items {
		names[i] = item.Name
	}
	return names
}

// claimRow reads the claim as it stands, which is what the transaction tests
// assert did or did not move.
func claimRow(t *testing.T, store *Store, libraryID string) (int64, string) {
	t.Helper()

	var (
		at sql.NullInt64
		by sql.NullString
	)
	err := store.reader.QueryRow(
		`SELECT claimed_at, claimed_by FROM scan_state WHERE library_id = ?`, libraryID).Scan(&at, &by)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ""
	}
	if err != nil {
		t.Fatalf("reading scan_state returned %v", err)
	}
	return at.Int64, by.String
}

// countIn is how many rows a table holds under a condition, for the assertions
// that are about an absence.
func countIn(t *testing.T, store *Store, query string, arguments ...any) int {
	t.Helper()

	var n int
	if err := store.reader.QueryRow(query, arguments...).Scan(&n); err != nil {
		t.Fatalf("counting rows returned %v", err)
	}
	return n
}

// TestApplyScanBatchIsOneTransaction is T12's first clause, and the failure is
// injected through the data rather than through a seam in the code — 002 T4's
// shape, and the schema is again what makes it a genuine mid-way failure.
//
// `item_files` is keyed on (item_id, ordinal), so an item carrying two files at
// the same ordinal fails on its **second** file: by then the batch has written
// a whole item, part of another, and — because the renewal is the transaction's
// first statement — a renewed claim. There is no implementation that can put
// those three anywhere else, so this is a failure part way through rather than
// one at the door.
//
// The control matters as much as the assertion. A batch that failed on its
// first statement would leave the same empty database, and the test would pass
// over a build with no transaction at all; so the first item of the doomed
// batch is written on its own afterwards and asserted to succeed, which is what
// says the rollback undid work that had already happened.
func TestApplyScanBatchIsOneTransaction(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)
	claimed(t, store, testFilmLibraryID, twoScanners[0])

	kept := anItem("00000000000000000000000000000001", testFilmLibraryID,
		string(library.KindMovie), "The Matrix", "The Matrix (1999).mkv")
	kept.Files = []ports.ScannedFile{{Ordinal: 0, Path: kept.Path, Size: 10, ModifiedAt: aScanInstant}}
	applied(t, store, testFilmLibraryID, twoScanners[0], aMinuteLater, kept)

	renewedAt, renewedBy := claimRow(t, store, testFilmLibraryID)
	if renewedAt != int64(aMinuteLater.Ticks()) || renewedBy != twoScanners[0] {
		t.Fatalf("after a batch the claim is (%d, %q), want (%d, %q): the renewal this test is "+
			"about to watch roll back does not happen",
			renewedAt, renewedBy, aMinuteLater.Ticks(), twoScanners[0])
	}

	// The first of the two is an ordinary item; the second carries two files at
	// ordinal 0 and is what the primary key refuses.
	first := anItem("00000000000000000000000000000002", testFilmLibraryID,
		string(library.KindMovie), "Amelie", "Amelie (2001).mkv")
	first.Files = []ports.ScannedFile{{Ordinal: 0, Path: first.Path, Size: 20, ModifiedAt: aScanInstant}}
	doomed := anItem("00000000000000000000000000000003", testFilmLibraryID,
		string(library.KindMovie), "The Long Film", "The Long Film (1998) - part1.mkv")
	doomed.Files = []ports.ScannedFile{
		{Ordinal: 0, Path: "The Long Film (1998) - part1.mkv", Size: 30, ModifiedAt: aScanInstant},
		{Ordinal: 0, Path: "The Long Film (1998) - part2.mkv", Size: 40, ModifiedAt: aScanInstant},
	}

	err := store.ApplyScanBatch(ctx, ports.ScanBatch{
		LibraryID: testFilmLibraryID,
		Items:     []ports.ScannedItem{first, doomed},
		ClaimedBy: twoScanners[0],
		At:        anHourLater,
	})
	if err == nil {
		t.Fatal("a batch whose second item repeats a file ordinal succeeded, want a refusal")
	}

	items, err := store.ItemsForLibrary(ctx, testFilmLibraryID)
	if err != nil {
		t.Fatalf("ItemsForLibrary returned %v", err)
	}
	if len(items) != 1 || items[0].ID != kept.ID {
		t.Errorf("the library holds %v after the failed batch, want only %q: a scan recorded "+
			"progress it did not make", identifiersOf(items), kept.Name)
	}

	// And the claim: the renewal is inside the same transaction, so a claim
	// stamped at the failed batch's instant is a scanner proving itself alive
	// by work that was thrown away.
	at, by := claimRow(t, store, testFilmLibraryID)
	if at != int64(aMinuteLater.Ticks()) || by != twoScanners[0] {
		t.Errorf("after the failed batch the claim is (%d, %q), want it left at (%d, %q): the "+
			"renewal outlived the batch that was supposed to prove the scanner alive",
			at, by, aMinuteLater.Ticks(), twoScanners[0])
	}

	// The control. Without it a build that refused the batch before writing
	// anything would pass every assertion above.
	applied(t, store, testFilmLibraryID, twoScanners[0], anHourLater, first)
	if n := countIn(t, store, `SELECT count(*) FROM items WHERE id = ?`, first.ID); n != 1 {
		t.Fatalf("the first item of the doomed batch cannot be written on its own (%d rows), so "+
			"the failure above was not part way through anything", n)
	}
}

// TestApplyScanBatchRefusesToRenewAClaimTheBatchNoLongerHolds is the other half
// of the renewal, and it is the reason `ClaimedBy` is a field on the batch.
//
// A scanner whose claim went stale and was taken must not renew it. Without the
// claimant in the WHERE clause it would, and the result is two scanners writing
// one library with each believing it is alone — which nothing else in this
// feature can detect, because both of them are writing valid items.
func TestApplyScanBatchRefusesToRenewAClaimTheBatchNoLongerHolds(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)

	claimed(t, store, testFilmLibraryID, twoScanners[0])
	won, previous, err := store.ClaimScan(ctx, testFilmLibraryID, twoScanners[1], anHourLater, oneMinute)
	if err != nil || !won {
		t.Fatalf("breaking the stale claim returned (%t, %q, %v), want it won", won, previous, err)
	}

	orphan := anItem("00000000000000000000000000000004", testFilmLibraryID,
		string(library.KindMovie), "The Matrix", "The Matrix (1999).mkv")
	if err := store.ApplyScanBatch(ctx, ports.ScanBatch{
		LibraryID: testFilmLibraryID,
		Items:     []ports.ScannedItem{orphan},
		ClaimedBy: twoScanners[0],
		At:        anHourLater,
	}); err == nil {
		t.Fatal("a batch from the displaced scanner succeeded, want a refusal")
	}

	if n := countIn(t, store, `SELECT count(*) FROM items`); n != 0 {
		t.Errorf("the displaced scanner wrote %d items, want 0", n)
	}
	if at, by := claimRow(t, store, testFilmLibraryID); by != twoScanners[1] || at != int64(anHourLater.Ticks()) {
		t.Errorf("the claim is (%d, %q), want the second scanner's (%d, %q)",
			at, by, anHourLater.Ticks(), twoScanners[1])
	}
}

// TestABatchNamingOneItemTwiceIsRefused is 003 T3's finding turned into the
// decision the store had to take.
//
// NFC has singleton mappings, so `K.mkv` written with U+212A KELVIN SIGN and
// `K.mkv` written with a plain `K` are two files on disk and one derived
// identifier — even in a case-sensitive library, and with nothing in
// `internal/library` able to notice. The pair therefore reaches a batch, and
// the alternative to this refusal is an upsert whose last write wins: a library
// holding one of the two files, silently, and a different one of them depending
// on which the walk read second.
func TestABatchNamingOneItemTwiceIsRefused(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)
	claimed(t, store, testFilmLibraryID, twoScanners[0])

	const shared = "00000000000000000000000000000005"
	kelvin := anItem(shared, testFilmLibraryID, string(library.KindMovie), "K", "KK.mkv")
	plain := anItem(shared, testFilmLibraryID, string(library.KindMovie), "K", "KK.mkv")

	err := store.ApplyScanBatch(ctx, ports.ScanBatch{
		LibraryID: testFilmLibraryID,
		Items:     []ports.ScannedItem{kelvin, plain},
		ClaimedBy: twoScanners[0],
		At:        aMinuteLater,
	})
	if !errors.Is(err, ErrRepeatedIdentifier) {
		t.Fatalf("a batch naming one identifier twice returned %v, want ErrRepeatedIdentifier", err)
	}
	if n := countIn(t, store, `SELECT count(*) FROM items`); n != 0 {
		t.Errorf("the refused batch wrote %d items, want 0", n)
	}
	if at, _ := claimRow(t, store, testFilmLibraryID); at != int64(aScanInstant.Ticks()) {
		t.Errorf("the refused batch renewed the claim to %d, want it left at %d",
			at, aScanInstant.Ticks())
	}
}

// TestAMultiPartFilmRoundTripsAsOneItemWithItsPartsInOrdinalOrder is T12's
// second clause, and it is read back rather than compared against what was
// written.
//
// The two sides of the round trip come from different places on purpose. What
// goes in is `library.Resolve`'s own answer over the two part files — so the
// case is the resolver's multi-part film and not a record this test invented —
// and what comes out is asserted against the paths, in ordinal order, plus the
// two row counts read straight out of the tables. Comparing the read against
// the written record would pass over a store that held the item in a single
// column and reconstructed it.
//
// The file rows are rewritten into the reverse of their ordinals first. That
// scramble is asserted to have taken, for `TestLibraryRootsAreOrdered…`'s
// reason — but see filesForLibrary's comment: on this table the primary key
// answers the ordering whether or not the query asks, so this establishes the
// property without being able to fail when the clause is removed.
//
// # What 003 T20's closing audit changed here, and why the order of the
// assertions is now load-bearing
//
// The two row counts used to sit **after** the scramble, which empties
// `item_files` and re-inserts both parts from the resolver's answer in memory.
// So they counted this test's own two inserts and said nothing about what
// `ApplyScanBatch` had written: a store that kept only the first file of an
// item was green here `[measurement: 003 T20, mutation of
// internal/store/sqlite.applyItem, Go 1.27.1, 2026-09-05]`, and the mutation
// was caught only incidentally, by a transaction test one file along and by
// three of AC-14's in `internal/app`. AC-4's *"one item with two media
// sources"* is one `items` row and two `item_files` rows (003 plan §8.3), so
// the counts are asserted **before** anything rewrites them, and the read
// assertions keep the scramble they need.
func TestAMultiPartFilmRoundTripsAsOneItemWithItsPartsInOrdinalOrder(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)

	const (
		partOne = "The Long Film (1998)/The Long Film (1998) - part1.mkv"
		partTwo = "The Long Film (1998)/The Long Film (1998) - part2.mkv"
	)
	films := aLibrary(testFilmLibraryID, "Films", "films")
	plan, err := library.Resolve(films, []library.Reading{{Root: 0, Entries: []library.Entry{
		{Path: partOne, Size: 111, ModifiedAt: aScanInstant},
		{Path: partTwo, Size: 222, ModifiedAt: aMinuteLater},
	}}})
	if err != nil {
		t.Fatalf("Resolve returned %v", err)
	}

	claimed(t, store, testFilmLibraryID, twoScanners[0])
	applied(t, store, testFilmLibraryID, twoScanners[0], aMinuteLater, plan.Items...)

	// One item row for the film and two file rows beneath it, asserted on what
	// the batch wrote and before anything here rewrites them. AC-4 is this
	// pair of numbers and nothing else at this layer.
	if n := countIn(t, store, `SELECT count(*) FROM items WHERE type = ?`, string(library.KindMovie)); n != 1 {
		t.Fatalf("the two parts became %d Movie rows, want 1", n)
	}
	if n := countIn(t, store, `SELECT count(*) FROM item_files`); n != 2 {
		t.Fatalf("the batch wrote %d file rows for the film, want 2 — a store that keeps one "+
			"file per item answers 008's MediaSources with half a film", n)
	}
	if written := filePathsInRowOrder(t, store); len(written) != 2 ||
		!slices.Contains(written, partOne) || !slices.Contains(written, partTwo) {
		t.Fatalf("the batch wrote the file rows %v, want both parts", written)
	}

	// The two parts, written in the order their ordinals do not name.
	if _, err := store.writer.ExecContext(ctx, `DELETE FROM item_files`); err != nil {
		t.Fatalf("emptying item_files returned %v", err)
	}
	film := itemOfType(t, plan.Items, string(library.KindMovie))
	for ordinal := len(film.Files) - 1; ordinal >= 0; ordinal-- {
		file := film.Files[ordinal]
		if _, err := store.writer.ExecContext(ctx,
			`INSERT INTO item_files (item_id, ordinal, path, size, modified_at) VALUES (?, ?, ?, ?, ?)`,
			film.ID, int64(file.Ordinal), file.Path, file.Size, int64(file.ModifiedAt.Ticks()),
		); err != nil {
			t.Fatalf("rewriting item_files returned %v", err)
		}
	}
	if rowOrder := filePathsInRowOrder(t, store); len(rowOrder) != 2 || rowOrder[0] != partTwo {
		t.Fatalf("a read with no ORDER BY answers %v, want the second part first: the scramble "+
			"did not take", rowOrder)
	}

	items, err := store.ItemsForLibrary(ctx, testFilmLibraryID)
	if err != nil {
		t.Fatalf("ItemsForLibrary returned %v", err)
	}
	read := itemOfType(t, items, string(library.KindMovie))
	if len(read.Files) != 2 {
		t.Fatalf("the film reads back with %d files, want 2", len(read.Files))
	}
	// The ordinals are the part markers' own numbers rather than slice
	// positions — `part1` and `part2` — which is 003 T5's decision and is read
	// out of the resolver here rather than restated.
	for i, want := range []ports.ScannedFile{
		{Ordinal: 1, Path: partOne, Size: 111, ModifiedAt: aScanInstant},
		{Ordinal: 2, Path: partTwo, Size: 222, ModifiedAt: aMinuteLater},
	} {
		got := read.Files[i]
		if got.Ordinal != want.Ordinal || got.Path != want.Path || got.Size != want.Size ||
			!got.ModifiedAt.Equal(want.ModifiedAt) {
			t.Errorf("file %d reads back as %+v, want %+v — the parts are what 008 answers "+
				"MediaSources from, in this order", i, got, want)
		}
	}
}

// itemOfType finds the one item of a type, and fails when there is not exactly
// one.
func itemOfType(t *testing.T, items []ports.ScannedItem, kind string) ports.ScannedItem {
	t.Helper()

	var found []ports.ScannedItem
	for _, item := range items {
		if item.Type == kind {
			found = append(found, item)
		}
	}
	if len(found) != 1 {
		t.Fatalf("there are %d items of type %s, want 1", len(found), kind)
	}
	return found[0]
}

// filePathsInRowOrder reads item_files the way a query with no ORDER BY does.
func filePathsInRowOrder(t *testing.T, store *Store) []string {
	t.Helper()

	rows, err := store.reader.Query(`SELECT path FROM item_files`)
	if err != nil {
		t.Fatalf("reading item_files returned %v", err)
	}
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			t.Fatalf("reading item_files returned %v", err)
		}
		paths = append(paths, path)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("reading item_files returned %v", err)
	}
	return paths
}

// TestItemsForLibraryOrdersOnTheSortKeyAndNotOnWhereTheRowsSit is T12's
// ordering clause, and it is written so that it can fail.
//
// 003 T10's finding is that a store's ordering clause is only observable where
// the query plan does not already agree with it, and T11's is that
// `derived/library.sql` declares **no index** — so `items` is answered by a
// scan in row order, and a set written in an order that disagrees with the sort
// key distinguishes an ordered read from an unordered one. The scramble is
// asserted before it is used, because a test that scrambled and did not check
// would pass for the wrong reason the day an index appears.
//
// The two reads are what the task asks for and they are not the whole
// assertion: two reads of a storage-ordered store also agree with each other.
// What makes the agreement mean something is the expected order beside it.
func TestItemsForLibraryOrdersOnTheSortKeyAndNotOnWhereTheRowsSit(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)
	claimed(t, store, testFilmLibraryID, twoScanners[0])

	// Written in the reverse of the order their keys name — and `The Abyss` is
	// in the set for a second reason. 003 §3.7.1 removes the leading article,
	// so its key is `abyss` and it sorts **first**, where its name sorts after
	// `Amelie`. Without it, `ORDER BY name` answers this corpus in exactly the
	// order `ORDER BY sort_key` does, and the mutation plan §8.3 names by name
	// — *"order by `name` instead of `sort_key`"* — survives the whole suite.
	written := []ports.ScannedItem{
		anItem("00000000000000000000000000000011", testFilmLibraryID,
			string(library.KindMovie), "Zodiac", "Zodiac (2007).mkv"),
		anItem("00000000000000000000000000000012", testFilmLibraryID,
			string(library.KindMovie), "The Matrix", "The Matrix (1999).mkv"),
		anItem("00000000000000000000000000000013", testFilmLibraryID,
			string(library.KindMovie), "Amelie", "Amelie (2001).mkv"),
		anItem("00000000000000000000000000000014", testFilmLibraryID,
			string(library.KindMovie), "The Abyss", "The Abyss (1989).mkv"),
	}
	applied(t, store, testFilmLibraryID, twoScanners[0], aMinuteLater, written...)

	unordered := itemNamesInRowOrder(t, store)
	if strings.Join(unordered, ",") != "Zodiac,The Matrix,Amelie,The Abyss" {
		t.Fatalf("a read with no ORDER BY answers %v, want the order this test wrote. The "+
			"scramble did not take, so nothing below distinguishes an ordered read from an "+
			"unordered one", unordered)
	}

	first, err := store.ItemsForLibrary(ctx, testFilmLibraryID)
	if err != nil {
		t.Fatalf("ItemsForLibrary returned %v", err)
	}
	second, err := store.ItemsForLibrary(ctx, testFilmLibraryID)
	if err != nil {
		t.Fatalf("the second ItemsForLibrary returned %v", err)
	}

	want := []string{"The Abyss", "Amelie", "The Matrix", "Zodiac"}
	if got := identifiersOf(first); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("ItemsForLibrary answers %v, want %v — the order is the sort key's, not the "+
			"order the rows happen to sit in", got, want)
	}
	if len(first) != len(second) {
		t.Fatalf("two reads answered %d and %d items", len(first), len(second))
	}
	for i := range first {
		if first[i].ID != second[i].ID {
			t.Errorf("element %d is %s in one read and %s in the next: a scan's answer would "+
				"depend on which read it made", i, first[i].ID, second[i].ID)
		}
	}
}

// itemNamesInRowOrder reads items the way a query with no ORDER BY does.
func itemNamesInRowOrder(t *testing.T, store *Store) []string {
	t.Helper()

	rows, err := store.reader.Query(`SELECT name FROM items`)
	if err != nil {
		t.Fatalf("reading items returned %v", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("reading items returned %v", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("reading items returned %v", err)
	}
	return names
}

// TestTheStoredSortKeyComparesAsBytesAndNotUnderNOCASE is the collation clause,
// and ADR-0003 is the only document in this feature that can see it.
//
// The pair has to be one whose two orderings differ, and finding one takes
// knowing where a capital letter survives: `SortKeyBase` lowercases at step 1,
// so everything keyed through it agrees under either collation. 003 §3.7.2's
// three overriding types append the **raw** name, so two episodes at the same
// numbers are where the bytes are visible — `Z` is 0x5A and `e` is 0x65, so
// byte order puts `Zero Day` first and `NOCASE` puts the lower-case title
// first.
//
// The control is the second read. It orders the same rows `COLLATE NOCASE` and
// requires the two answers to **differ**, so the pair is proved to discriminate
// rather than assumed to: a test whose two names ordered the same way under
// both collations would assert nothing and look identical.
func TestTheStoredSortKeyComparesAsBytesAndNotUnderNOCASE(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)
	claimed(t, store, testShowLibraryID, twoScanners[0])

	const lowercaseTitle = "eps1.0_hellofriend.mov"
	applied(t, store, testShowLibraryID, twoScanners[0], aMinuteLater,
		anEpisode("00000000000000000000000000000021", testShowLibraryID,
			lowercaseTitle, "Mr Robot/Season 01/Mr Robot - S01E01.mkv", 1, 1),
		anEpisode("00000000000000000000000000000022", testShowLibraryID,
			"Zero Day", "Zero Hour/Season 01/Zero Hour - S01E01.mkv", 1, 1),
	)

	items, err := store.ItemsForLibrary(ctx, testShowLibraryID)
	if err != nil {
		t.Fatalf("ItemsForLibrary returned %v", err)
	}
	byBytes := identifiersOf(items)
	if strings.Join(byBytes, ",") != "Zero Day,"+lowercaseTitle {
		t.Errorf("ItemsForLibrary answers %v, want Zero Day first. The column is compared with "+
			"the default BINARY collation; NOCASE is ASCII-only and would order two names by a "+
			"rule this project cannot explain (ADR-0003)", byBytes)
	}

	underNOCASE := itemNamesOrderedBy(t, store, `sort_key COLLATE NOCASE, id`)
	if strings.Join(underNOCASE, ",") == strings.Join(byBytes, ",") {
		t.Fatalf("the two names order the same way under BINARY and under NOCASE (%v), so this "+
			"test asserts nothing about a collation", byBytes)
	}
}

// itemNamesOrderedBy reads the item names under an ORDER BY of the test's
// choosing, which is how the collation control asks the same rows a different
// question.
func itemNamesOrderedBy(t *testing.T, store *Store, orderBy string) []string {
	t.Helper()

	rows, err := store.reader.Query(`SELECT name FROM items ORDER BY ` + orderBy)
	if err != nil {
		t.Fatalf("reading items returned %v", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("reading items returned %v", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("reading items returned %v", err)
	}
	return names
}

// TestRemoveItemsTakesTheFilesWithThemAndLeavesAnotherLibraryAlone is T12's
// third clause.
//
// The cascade is the schema's rather than a second DELETE this method could
// forget, and the other library is the half that matters: a removal written
// `WHERE library_id = ?` — the shape somebody reaches for when the identifiers
// look inconvenient — is the over-broad DELETE 002 T4 guarded against once, and
// here it costs an entire library's items rather than a token.
func TestRemoveItemsTakesTheFilesWithThemAndLeavesAnotherLibraryAlone(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)

	doomed := withOneFile(anItem("00000000000000000000000000000031", testFilmLibraryID,
		string(library.KindMovie), "Zodiac", "Zodiac (2007).mkv"))
	kept := withOneFile(anItem("00000000000000000000000000000032", testFilmLibraryID,
		string(library.KindMovie), "Amelie", "Amelie (2001).mkv"))
	elsewhere := withOneFile(anItem("00000000000000000000000000000033", testShowLibraryID,
		string(library.KindEpisode), "Pilot", "The Series/Season 01/S01E01.mkv"))

	claimed(t, store, testFilmLibraryID, twoScanners[0])
	applied(t, store, testFilmLibraryID, twoScanners[0], aMinuteLater, doomed, kept)
	claimed(t, store, testShowLibraryID, twoScanners[1])
	applied(t, store, testShowLibraryID, twoScanners[1], aMinuteLater, elsewhere)

	if err := store.RemoveItems(ctx, []string{doomed.ID}); err != nil {
		t.Fatalf("RemoveItems returned %v", err)
	}

	if n := countIn(t, store, `SELECT count(*) FROM item_files WHERE item_id = ?`, doomed.ID); n != 0 {
		t.Errorf("the removed item left %d file rows behind, want 0 — the cascade the schema "+
			"declares is what takes them", n)
	}
	for _, survivor := range []ports.ScannedItem{kept, elsewhere} {
		if n := countIn(t, store, `SELECT count(*) FROM items WHERE id = ?`, survivor.ID); n != 1 {
			t.Errorf("%q is gone after removing another item, want it left alone", survivor.Name)
		}
		if n := countIn(t, store, `SELECT count(*) FROM item_files WHERE item_id = ?`, survivor.ID); n != 1 {
			t.Errorf("%q lost its file rows after removing another item, want 1", survivor.Name)
		}
	}
}

// withOneFile gives an item the single file most items have.
func withOneFile(item ports.ScannedItem) ports.ScannedItem {
	item.Files = []ports.ScannedFile{
		{Ordinal: 0, Path: item.Path, Size: int64(len(item.Path)), ModifiedAt: aScanInstant},
	}
	return item
}

// TestRemovingAnIdentifierThatIsNotThereIsRefusedAndRemovesNothing is 001's
// rows-affected rule at the place it bites hardest.
//
// Every identifier a removal names came out of this store, so one that matches
// no row means the caller is holding some other library's reading — and a
// DELETE that quietly matched fewer rows leaves the next scan computing the
// same removal again, for ever, with nothing reporting it. The refusal takes
// the whole call with it, which is what the surviving row asserts.
func TestRemovingAnIdentifierThatIsNotThereIsRefusedAndRemovesNothing(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)

	real := withOneFile(anItem("00000000000000000000000000000041", testFilmLibraryID,
		string(library.KindMovie), "Zodiac", "Zodiac (2007).mkv"))
	claimed(t, store, testFilmLibraryID, twoScanners[0])
	applied(t, store, testFilmLibraryID, twoScanners[0], aMinuteLater, real)

	err := store.RemoveItems(ctx, []string{real.ID, "0000000000000000000000000000ffff"})
	if err == nil {
		t.Fatal("removing an identifier that is not there succeeded, want a refusal")
	}
	if n := countIn(t, store, `SELECT count(*) FROM items WHERE id = ?`, real.ID); n != 1 {
		t.Errorf("the refused removal took %q with it, want the call to be one transaction", real.Name)
	}
}

// TestClaimScanReportsALiveClaimAsFalseAndNamesWhoHoldsIt is T12's fourth
// clause, first half.
//
// It is false rather than an error because two scanners over one store is a
// state this feature creates on purpose (003 plan §6.7): an operator may run a
// scan against a data directory a server is already serving from, and
// *"somebody else is scanning"* is something the subcommand prints rather than
// a fault it reports. The claimant travels beside it because plan §7's row for
// this failure says the message names one.
func TestClaimScanReportsALiveClaimAsFalseAndNamesWhoHoldsIt(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)
	claimed(t, store, testFilmLibraryID, twoScanners[0])

	// One tick short of stale, which is the boundary the other half of this
	// pair sits on.
	won, previous, err := store.ClaimScan(ctx, testFilmLibraryID, twoScanners[1],
		units.TimeFromTicks(aScanInstant.Ticks()+oneMinute-1), oneMinute)
	if err != nil {
		t.Fatalf("ClaimScan returned %v", err)
	}
	if won {
		t.Error("ClaimScan took a claim one tick short of stale, want it left alone")
	}
	if previous != twoScanners[0] {
		t.Errorf("ClaimScan names %q as holding the claim, want %q — a refusal that cannot say "+
			"who holds it is a message an operator cannot act on", previous, twoScanners[0])
	}
	if at, by := claimRow(t, store, testFilmLibraryID); by != twoScanners[0] || at != int64(aScanInstant.Ticks()) {
		t.Errorf("the claim is (%d, %q) after the losing call, want it untouched at (%d, %q)",
			at, by, aScanInstant.Ticks(), twoScanners[0])
	}
}

// TestClaimScanBreaksAClaimOlderThanStaleAfterAndNamesThePreviousClaimant is
// the second half, and the pair is asserted at the boundary rather than a long
// way either side of it.
//
// A claim is broken because a process killed mid-scan leaves one behind and the
// alternative is a library nothing will ever scan again. `staleAfter` is not a
// guess about how long a scan takes: the claim is renewed on every committed
// batch, so the value only has to exceed the time between two of them
// (003 plan §6.9).
func TestClaimScanBreaksAClaimOlderThanStaleAfterAndNamesThePreviousClaimant(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)
	claimed(t, store, testFilmLibraryID, twoScanners[0])

	// Exactly staleAfter old: the age is not *less than* staleAfter, so it
	// goes. One tick younger is the losing case above, so the two together say
	// where the line is rather than that there is one somewhere.
	at := units.TimeFromTicks(aScanInstant.Ticks() + oneMinute)
	won, previous, err := store.ClaimScan(ctx, testFilmLibraryID, twoScanners[1], at, oneMinute)
	if err != nil {
		t.Fatalf("ClaimScan returned %v", err)
	}
	if !won {
		t.Fatal("ClaimScan left a claim exactly staleAfter old, want it broken and taken")
	}
	if previous != twoScanners[0] {
		t.Errorf("ClaimScan names %q as the previous claimant, want %q — plan §7 wants the log "+
			"line to say whose claim was broken, and the row now names the winner",
			previous, twoScanners[0])
	}
	if got, by := claimRow(t, store, testFilmLibraryID); by != twoScanners[1] || got != int64(at.Ticks()) {
		t.Errorf("the claim is (%d, %q), want the breaker's (%d, %q)", got, by, at.Ticks(), twoScanners[1])
	}
}

// TestAClaimIsNotBrokenByAClockThatMovedBackwards is the one outcome this
// method must never produce.
//
// A claim stamped *after* the instant the caller offers is a clock adjustment
// and not an abandoned scan, and treating the negative age as "older than
// anything" would break a live claim and put two scanners on one library. The
// signed arithmetic is what makes it fall the right way, and this is what says
// so.
func TestAClaimIsNotBrokenByAClockThatMovedBackwards(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)
	claimed(t, store, testFilmLibraryID, twoScanners[0])

	earlier := units.TimeFromTicks(aScanInstant.Ticks() - 10*oneMinute)
	won, previous, err := store.ClaimScan(ctx, testFilmLibraryID, twoScanners[1], earlier, oneMinute)
	if err != nil {
		t.Fatalf("ClaimScan returned %v", err)
	}
	if won {
		t.Error("a clock that moved backwards broke a live claim, want it left alone: that is " +
			"two scanners writing one library")
	}
	if previous != twoScanners[0] {
		t.Errorf("ClaimScan names %q, want %q", previous, twoScanners[0])
	}
}

// TestClaimingALibraryThatHasNeverBeenScannedCreatesItsRow is why ClaimScan is
// an insert and not an UPDATE.
//
// A library with no `scan_state` row has never been scanned, and that is the
// same state a rebuild leaves every library in (003 plan §4.3) — so this is not
// a rare first-start branch, it is the branch every library takes after a
// generation bump. An `UPDATE ... WHERE library_id = ?` matching no row
// succeeds and claims nothing, which is 003 T10's `ReplaceRoots` trap one table
// along.
func TestClaimingALibraryThatHasNeverBeenScannedCreatesItsRow(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)

	if n := countIn(t, store, `SELECT count(*) FROM scan_state`); n != 0 {
		t.Fatalf("a fresh database holds %d scan_state rows, want 0", n)
	}

	won, previous, err := store.ClaimScan(ctx, testFilmLibraryID, twoScanners[0], aScanInstant, oneMinute)
	if err != nil {
		t.Fatalf("ClaimScan returned %v", err)
	}
	if !won || previous != "" {
		t.Errorf("the first claim of a library returned (%t, %q), want (true, \"\")", won, previous)
	}
	if at, by := claimRow(t, store, testFilmLibraryID); by != twoScanners[0] || at != int64(aScanInstant.Ticks()) {
		t.Errorf("the claim is (%d, %q), want (%d, %q) — an UPDATE would have claimed nothing "+
			"and said it had", at, by, aScanInstant.Ticks(), twoScanners[0])
	}
}

// TestReleaseScanDropsTheClaimAndRecordsWhatTheScanDid is the last transaction
// of a scan, seen from the store.
//
// A released claim has to be *absent* rather than expired: the next scan of the
// library must win immediately, whatever `staleAfter` is, and that is asserted
// here by claiming again at the same instant the release happened.
func TestReleaseScanDropsTheClaimAndRecordsWhatTheScanDid(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)
	claimed(t, store, testFilmLibraryID, twoScanners[0])

	summary := []byte(`{"Added":1,"Updated":0,"Removed":0}`)
	if err := store.ReleaseScan(ctx, testFilmLibraryID, aMinuteLater, summary, true); err != nil {
		t.Fatalf("ReleaseScan returned %v", err)
	}

	var (
		lastAt   sql.NullInt64
		lastFull sql.NullInt64
		document sql.NullString
	)
	if err := store.reader.QueryRow(
		`SELECT last_scan_at, last_scan_full, summary_document FROM scan_state WHERE library_id = ?`,
		testFilmLibraryID).Scan(&lastAt, &lastFull, &document); err != nil {
		t.Fatalf("reading scan_state returned %v", err)
	}
	if lastAt.Int64 != int64(aMinuteLater.Ticks()) || lastFull.Int64 != 1 {
		t.Errorf("the release recorded (%d, full=%d), want (%d, full=1)",
			lastAt.Int64, lastFull.Int64, aMinuteLater.Ticks())
	}
	if document.String != string(summary) {
		t.Errorf("the summary reads back as %q, want %q", document.String, summary)
	}
	if _, by := claimRow(t, store, testFilmLibraryID); by != "" {
		t.Errorf("the claim still names %q after a release, want nobody", by)
	}

	won, previous, err := store.ClaimScan(ctx, testFilmLibraryID, twoScanners[1], aMinuteLater, oneMinute)
	if err != nil || !won {
		t.Fatalf("claiming a released library returned (%t, %q, %v), want it won immediately",
			won, previous, err)
	}
	if previous != "" {
		t.Errorf("claiming a released library names %q as displaced, want nobody: the claim was "+
			"given up rather than broken", previous)
	}
}

// TestReleasingAScanNobodyClaimedIsRefused is the other half of 001's
// rows-affected rule here. ClaimScan created the row, so a release that matched
// none is a caller that has lost track of which library it was scanning.
func TestReleasingAScanNobodyClaimedIsRefused(t *testing.T) {
	store := openForTest(t)

	if err := store.ReleaseScan(context.Background(), testFilmLibraryID,
		aMinuteLater, noScanSummary, false); err == nil {
		t.Fatal("releasing a scan of a library nothing claimed succeeded, want a refusal")
	}
}

// TestEveryColumnOfAnItemSurvivesTheRoundTrip is the shape assertion under all
// of the above, and the two rows are chosen so that absent and zero are told
// apart.
//
// The numbers are pointers because season **0** is `Specials` (003 §3.4) and a
// season with no number at all is not it, so the minimal row asserts nil where
// the full one asserts a pointer to zero. `root_ordinal` is the field that is
// deliberately not nullable-by-emptiness: an inferred season has no path and
// still belongs to a root, and one dropped along with the path would come back
// as 0 and make every scan report the season updated.
//
// `SortTitle` is asserted **empty** on the way out, and that is load-bearing
// rather than an omission: it is the one field with no column (003 plan §5), so
// 003 T9's comparison excludes it, and a store that invented a value for it
// would make every item differ from itself the day 004 supplies one.
func TestEveryColumnOfAnItemSurvivesTheRoundTrip(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)
	claimed(t, store, testShowLibraryID, twoScanners[0])

	premiere := units.TimeFromTicks(638_100_000_000_000_000)
	full := ports.ScannedItem{
		ID:                "00000000000000000000000000000051",
		LibraryID:         testShowLibraryID,
		ParentID:          "00000000000000000000000000000052",
		Type:              string(library.KindEpisode),
		Name:              "Pilot",
		SortTitle:         "a sort title no column holds",
		Path:              "The Series/Season 01/The Series - S01E01 - Pilot.mkv",
		RootOrdinal:       2,
		IndexNumber:       intPointerFor(1),
		ParentIndexNumber: intPointerFor(0),
		IndexNumberEnd:    intPointerFor(2),
		ProductionYear:    intPointerFor(2024),
		PremiereDate:      &premiere,
		Unplaceable:       true,
		Files: []ports.ScannedFile{
			{Ordinal: 0, Path: "The Series/Season 01/The Series - S01E01 - Pilot.mkv",
				Size: 4096, ModifiedAt: aTickLater},
		},
	}
	full.SortKey = library.SortKeyFor(&full)

	// A path-less container that still names a root, which is the pair the
	// nullability of `path` and the non-nullability of `root_ordinal` is about.
	minimal := ports.ScannedItem{
		ID:          "00000000000000000000000000000052",
		LibraryID:   testShowLibraryID,
		Type:        string(library.KindSeason),
		Name:        "Season Unknown",
		RootOrdinal: 2,
	}
	minimal.SortKey = library.SortKeyFor(&minimal)

	applied(t, store, testShowLibraryID, twoScanners[0], aMinuteLater, minimal, full)

	items, err := store.ItemsForLibrary(ctx, testShowLibraryID)
	if err != nil {
		t.Fatalf("ItemsForLibrary returned %v", err)
	}
	read := map[string]ports.ScannedItem{}
	for _, item := range items {
		read[item.ID] = item
	}

	gotFull, ok := read[full.ID]
	if !ok {
		t.Fatalf("the episode is missing from the %d items read back", len(items))
	}
	// SortTitle has no column, so what came out cannot carry the one that went
	// in — and must not carry an invented one either.
	wantFull := full
	wantFull.SortTitle = ""
	assertSameItem(t, "the episode", gotFull, wantFull)

	gotMinimal, ok := read[minimal.ID]
	if !ok {
		t.Fatal("the season is missing from the items read back")
	}
	assertSameItem(t, "the season", gotMinimal, minimal)

	if gotMinimal.IndexNumber != nil {
		t.Errorf("a season with no number reads back as %d, want nil: absent and zero are "+
			"different answers, and season 0 is Specials", *gotMinimal.IndexNumber)
	}
	if gotFull.ParentIndexNumber == nil || *gotFull.ParentIndexNumber != 0 {
		t.Errorf("a season number of 0 reads back as %v, want a pointer to 0", gotFull.ParentIndexNumber)
	}
	// The stored representation, not only what comes back through the record.
	// An empty string and NULL both read back as "" — so a store that wrote the
	// empty string would pass every assertion above and leave `path IS NULL`,
	// which is how 003 plan §4.2 says "an inferred container has no directory",
	// answering nothing.
	if n := countIn(t, store,
		`SELECT count(*) FROM items WHERE id = ? AND path IS NULL AND parent_id IS NULL`,
		minimal.ID); n != 1 {
		t.Errorf("the path-less season is not stored with a NULL path and a NULL parent "+
			"(%d rows match), want it to be: the empty string is a parent whose identifier "+
			"is empty, which is a different question", n)
	}
	if n := countIn(t, store,
		`SELECT count(*) FROM items WHERE id = ? AND path IS NOT NULL AND parent_id IS NOT NULL`,
		full.ID); n != 1 {
		t.Errorf("the episode is stored with a NULL path or a NULL parent (%d rows match the "+
			"opposite), want both present — otherwise the assertion above holds because "+
			"everything is NULL", n)
	}
	if gotMinimal.Path != "" || gotMinimal.RootOrdinal != 2 {
		t.Errorf("the path-less season reads back with path %q and root %d, want \"\" and 2 — a "+
			"root ordinal dropped with the path makes every scan report the season updated",
			gotMinimal.Path, gotMinimal.RootOrdinal)
	}
}

// intPointerFor is the pointer the record's optional numbers carry. It is named
// for what it is rather than shortened, because a `p(0)` in a table of numbers
// is the shape that makes nil and zero look alike in review.
func intPointerFor(n int) *int { return &n }

// assertSameItem compares two items field by field, which is what a round trip
// is about: a comparison that skipped a field would pass over a store that
// never wrote its column.
func assertSameItem(t *testing.T, through string, got, want ports.ScannedItem) {
	t.Helper()

	if got.ID != want.ID {
		t.Errorf("%s: ID is %q, want %q", through, got.ID, want.ID)
	}
	if got.LibraryID != want.LibraryID {
		t.Errorf("%s: LibraryID is %q, want %q", through, got.LibraryID, want.LibraryID)
	}
	if got.ParentID != want.ParentID {
		t.Errorf("%s: ParentID is %q, want %q", through, got.ParentID, want.ParentID)
	}
	if got.Type != want.Type {
		t.Errorf("%s: Type is %q, want %q", through, got.Type, want.Type)
	}
	if got.Name != want.Name {
		t.Errorf("%s: Name is %q, want %q", through, got.Name, want.Name)
	}
	if got.SortKey != want.SortKey {
		t.Errorf("%s: SortKey is %q, want %q", through, got.SortKey, want.SortKey)
	}
	if got.SortTitle != want.SortTitle {
		t.Errorf("%s: SortTitle is %q, want %q — it is the one field with no column",
			through, got.SortTitle, want.SortTitle)
	}
	if got.Path != want.Path {
		t.Errorf("%s: Path is %q, want %q", through, got.Path, want.Path)
	}
	if got.RootOrdinal != want.RootOrdinal {
		t.Errorf("%s: RootOrdinal is %d, want %d", through, got.RootOrdinal, want.RootOrdinal)
	}
	assertSameOptionalInt(t, through, "IndexNumber", got.IndexNumber, want.IndexNumber)
	assertSameOptionalInt(t, through, "ParentIndexNumber", got.ParentIndexNumber, want.ParentIndexNumber)
	assertSameOptionalInt(t, through, "IndexNumberEnd", got.IndexNumberEnd, want.IndexNumberEnd)
	assertSameOptionalInt(t, through, "ProductionYear", got.ProductionYear, want.ProductionYear)
	switch {
	case (got.PremiereDate == nil) != (want.PremiereDate == nil):
		t.Errorf("%s: PremiereDate is %v, want %v", through, got.PremiereDate, want.PremiereDate)
	case got.PremiereDate != nil && !got.PremiereDate.Equal(*want.PremiereDate):
		t.Errorf("%s: PremiereDate is %s, want %s", through, got.PremiereDate, want.PremiereDate)
	}
	if got.Unplaceable != want.Unplaceable {
		t.Errorf("%s: Unplaceable is %v, want %v — 003 §3.8 counts it apart from a skip",
			through, got.Unplaceable, want.Unplaceable)
	}
	if len(got.Files) != len(want.Files) {
		t.Fatalf("%s: %d files, want %d", through, len(got.Files), len(want.Files))
	}
	for i := range got.Files {
		if got.Files[i].Ordinal != want.Files[i].Ordinal ||
			got.Files[i].Path != want.Files[i].Path ||
			got.Files[i].Size != want.Files[i].Size ||
			!got.Files[i].ModifiedAt.Equal(want.Files[i].ModifiedAt) {
			t.Errorf("%s: file %d is %+v, want %+v", through, i, got.Files[i], want.Files[i])
		}
	}
}

// assertSameOptionalInt compares two optional numbers, telling absent from zero
// in the message as well as in the comparison.
func assertSameOptionalInt(t *testing.T, through, field string, got, want *int) {
	t.Helper()

	switch {
	case got == nil && want == nil:
	case got == nil || want == nil:
		t.Errorf("%s: %s is %s, want %s", through, field, describeOptionalInt(got), describeOptionalInt(want))
	case *got != *want:
		t.Errorf("%s: %s is %d, want %d", through, field, *got, *want)
	}
}

func describeOptionalInt(value *int) string {
	if value == nil {
		return "absent"
	}
	return "present"
}

// TestAnUpdateReplacesTheFilesRatherThanAccumulatingThem is what makes the
// upsert an update rather than an append.
//
// A two-part film whose second part was deleted keeps its identifier — the
// identifier is derived from the first part's path — and loses a file. An
// upsert on `item_files` would leave the old part behind, and 008 would answer
// a `MediaSources` naming a file that is not there.
func TestAnUpdateReplacesTheFilesRatherThanAccumulatingThem(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)
	claimed(t, store, testFilmLibraryID, twoScanners[0])

	film := anItem("00000000000000000000000000000061", testFilmLibraryID,
		string(library.KindMovie), "The Long Film", "The Long Film (1998) - part1.mkv")
	film.Files = []ports.ScannedFile{
		{Ordinal: 0, Path: "The Long Film (1998) - part1.mkv", Size: 10, ModifiedAt: aScanInstant},
		{Ordinal: 1, Path: "The Long Film (1998) - part2.mkv", Size: 20, ModifiedAt: aScanInstant},
	}
	applied(t, store, testFilmLibraryID, twoScanners[0], aMinuteLater, film)

	film.Files = film.Files[:1]
	applied(t, store, testFilmLibraryID, twoScanners[0], anHourLater, film)

	items, err := store.ItemsForLibrary(ctx, testFilmLibraryID)
	if err != nil {
		t.Fatalf("ItemsForLibrary returned %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("the library holds %d items after the update, want 1", len(items))
	}
	if len(items[0].Files) != 1 || items[0].Files[0].Ordinal != 0 {
		t.Errorf("the film reads back with %d files, want 1: an upsert leaves the withdrawn part "+
			"behind as a media source pointing at a file that is not there", len(items[0].Files))
	}
}

// TestItemsForLibraryAnswersOnlyThatLibrary is the read's half of the pair
// TestRemoveItems… asserts for the write.
//
// The filter is the only thing separating two libraries in this table — there
// is no foreign key, because the derived half holds no reference into the
// precious one (architecture §6) — so a read that lost its WHERE would hand a
// scan of one library the items of another and reconcile them into removals.
func TestItemsForLibraryAnswersOnlyThatLibrary(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)

	claimed(t, store, testFilmLibraryID, twoScanners[0])
	applied(t, store, testFilmLibraryID, twoScanners[0], aMinuteLater,
		withOneFile(anItem("00000000000000000000000000000071", testFilmLibraryID,
			string(library.KindMovie), "Zodiac", "Zodiac (2007).mkv")))
	claimed(t, store, testMusicLibraryID, twoScanners[1])
	applied(t, store, testMusicLibraryID, twoScanners[1], aMinuteLater,
		withOneFile(anItem("00000000000000000000000000000072", testMusicLibraryID,
			string(library.KindMusicAlbum), "First Album", "An Artist/First Album (2001)")))

	items, err := store.ItemsForLibrary(ctx, testFilmLibraryID)
	if err != nil {
		t.Fatalf("ItemsForLibrary returned %v", err)
	}
	if len(items) != 1 || items[0].LibraryID != testFilmLibraryID {
		t.Fatalf("the film library reads back %v, want only its own item", identifiersOf(items))
	}
	if len(items[0].Files) != 1 {
		t.Errorf("the item reads back with %d files, want 1", len(items[0].Files))
	}
}
