// AC-13 at the level the criterion is written at: the sort key the **store
// ends up holding** for the fixture's awkward names, and the order the store
// therefore answers in.
//
// # Why this file exists, and what it is the fix for
//
// 003 T20's closing audit hunted one shape by name — *a criterion about what
// the store ends up holding, proven about the function that computes it* — and
// found it here. AC-13 reads *"sort ordering matches the table in §3.7 for the
// fixture's awkward names"*. Before this file it was discharged entirely in
// `internal/library/sortkey_test.go`, over `SortKeyFor` as a pure function,
// and in `internal/store/sqlite/items_test.go`, over a four-item corpus whose
// expected keys that test computes **by calling `library.SortKeyFor` itself**.
// Nothing anywhere connected the two: no assertion said that a scan of the
// fixture puts §3.7's bytes in the `sort_key` column.
//
// Three lines in [scan.Scanner.Scan], between the resolver and the batch —
//
//	for i := range plan.Items {
//		plan.Items[i].SortKey = plan.Items[i].Name
//	}
//
// — store every item's *name* as its sort key, which is behaviours §2.6's named
// temptation and reorders every album, every episode and every list a client
// will ever ask for, and leave **the entire repository green**
// `[measurement: 003 T20, mutation of internal/scan.Scanner.Scan, Go 1.27.1,
// 2026-09-05]`. That is 001's and 002's closing findings for the third time,
// in the form a feature with no routes takes.
//
// # What makes this an assertion rather than a second spelling of the derivation
//
// The expected keys are **written down**, not worked out. Computing them with
// `library.SortKeyFor` would be the store test's own weakness one layer up: the
// mutation above leaves that function untouched, so anything that asks it
// agrees with the build that is wrong.
// [TestTheExpectedSortKeysAreLiteralsAndNotADerivation] holds that property
// rather than leaving it to how this file happens to be written, which is 003
// T1's guard over `expected.go` applied to the same hazard.
//
// The keys are compared between delimiters, because `rock  roll`'s double space
// and `s w a t `'s trailing one are the contract and neither is visible in a
// diff (003 T4's rule).
//
// # What this does not prove
//
// That the order a **client** receives is this one. That is 003 plan §8.3's
// second row and it is 005's: one `/Items` listing over the fixture, compared
// byte for byte. A green run here is evidence about the column and about the
// read that orders on it, and about nothing downstream of either.
package app

import (
	"bytes"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/vdatanet/atrium-go/internal/ports"
)

// TestTheSortKeyTheStoreEndsUpHoldingIsTheOneTheTableDerives is AC-13 over a
// real scan into a real store.
//
// It is the assertion the audit found missing, and it is written so that the
// mutation that found it — storing the name — fails on **every** row rather
// than on one, which is what the second control below requires of the table.
func TestTheSortKeyTheStoreEndsUpHoldingIsTheOneTheTableDerives(t *testing.T) {
	t.Parallel()

	data, libraries := theWholeFixture(t, t.TempDir())
	if summaries := scanSummaries(t, data); len(summaries) != len(libraries) {
		t.Fatalf("the scan reported %d libraries where %d were declared", len(summaries), len(libraries))
	}

	wanted := map[string][]sortKeyRow{"Movies": theSortKeysOfTheMoviesLibrary()}
	for name, rows := range theSortKeysOfTheOverridingTypes() {
		wanted[name] = rows
	}

	checked := 0
	for _, lib := range libraries {
		rows, ok := wanted[lib.Name]
		if !ok {
			continue
		}
		held := sortKeysByPlace(t, storedItems(t, data, lib.ID))
		for _, row := range rows {
			got, found := held[row.where]
			if !found {
				t.Errorf("%s: no item at %q, so the row saying %s was not checked at all",
					lib.Name, row.where, row.shows)
				continue
			}
			if got.name != row.name {
				t.Errorf("%s: the item at %q is named %s, want %s — this table names the item as "+
					"well as its key, so a name that moved fails here rather than silently "+
					"changing what is being asserted",
					lib.Name, row.where, quoted(got.name), quoted(row.name))
			}
			if got.key != row.key {
				t.Errorf("%s: the stored sort_key of %q is %s, want %s (%s)",
					lib.Name, row.where, quoted(got.key), quoted(row.key), row.shows)
			}
			checked++
		}
	}

	// The control, in the direction the criterion does not name: a run that
	// found nothing satisfies every assertion above.
	if want := len(theSortKeysOfTheMoviesLibrary()) + 15; checked != want {
		t.Fatalf("checked %d rows, want %d — the table and the tree have gone out of step and "+
			"the rows that went missing were not asserted", checked, want)
	}
}

// TestEveryExpectedSortKeyDiffersFromTheNameItIsDerivedFrom is the control that
// makes the table above able to fail on the build the audit found.
//
// The mutation stores each item's **name** in the `sort_key` column. A table
// containing a row whose key happens to equal its name would pass that row, so
// the table is required to contain none: every row of it fails on that build,
// and a row added later that does not is refused here rather than weakening the
// assertion silently.
func TestEveryExpectedSortKeyDiffersFromTheNameItIsDerivedFrom(t *testing.T) {
	t.Parallel()

	rows := theSortKeysOfTheMoviesLibrary()
	for _, byLibrary := range theSortKeysOfTheOverridingTypes() {
		rows = append(rows, byLibrary...)
	}
	if len(rows) == 0 {
		t.Fatal("there are no rows, so this control has nothing to control")
	}
	for _, row := range rows {
		if row.key == row.name {
			t.Errorf("the row at %q expects the key %s, which is the item's own name — a build "+
				"that stored the name as the key passes that row, and this table exists because "+
				"such a build is green everywhere else", row.where, quoted(row.key))
		}
	}
}

// TestTheStoreAnswersTheFixtureInTheOrderItsSortKeysName is AC-13's *ordering*
// half, over the whole `movies` library.
//
// The store's read orders on `sort_key` and compares it as bytes (T12). What
// this adds is the composition: the order a scan of a real tree leaves behind
// is the order §3.7's table names, and it is **not** the order the names name.
// The second half is the control — over a corpus whose two orders agreed, an
// ordering assertion asserts nothing (003 T12's finding, one layer up).
func TestTheStoreAnswersTheFixtureInTheOrderItsSortKeysName(t *testing.T) {
	t.Parallel()

	data, libraries := theWholeFixture(t, t.TempDir())
	scanSummaries(t, data)

	rows := theSortKeysOfTheMoviesLibrary()
	byName := make([]string, len(rows))
	for i, row := range rows {
		byName[i] = row.name
	}
	slices.Sort(byName)
	if wantedOrder := namesInRowOrder(rows); slices.Equal(byName, wantedOrder) {
		t.Fatal("this library's items sort the same way by name and by key, so nothing below " +
			"distinguishes a store that ordered on `name` from one that ordered on `sort_key`")
	}

	var movies string
	for _, lib := range libraries {
		if lib.Name == "Movies" {
			movies = lib.ID
		}
	}
	if movies == "" {
		t.Fatal("the fixture declared no library called Movies")
	}

	items := storedItems(t, data, movies)
	got := make([]string, 0, len(items))
	for _, item := range items {
		got = append(got, item.Name)
	}
	if want := namesInRowOrder(rows); !slices.Equal(got, want) {
		t.Errorf("the store answers\n  %s\nwant\n  %s", strings.Join(got, " | "), strings.Join(want, " | "))
	}
}

// TestTheExpectedSortKeysAreLiteralsAndNotADerivation holds the property the
// file's own comment claims, rather than leaving it to how the file happens to
// be written.
//
// The build this file exists to catch leaves `library.SortKeyFor` correct and
// stores something else, so a table that asked that function what to expect
// would agree with the wrong build about every row. 003 T1 wrote the same guard
// over `expected.go` for the same reason.
func TestTheExpectedSortKeysAreLiteralsAndNotADerivation(t *testing.T) {
	t.Parallel()

	const literal = "library_sortkey_table_test.go"

	source, err := os.ReadFile(literal)
	if err != nil {
		t.Fatalf("reading %s: %v", literal, err)
	}
	for _, derivation := range []string{"SortKeyFor", "SortKeyBase", "sortPadWidth", "sortArticles"} {
		if bytes.Contains(source, []byte(derivation)) {
			t.Errorf("%s mentions %q, so the expected keys are worked out by the derivation this "+
				"file is checking. The mutation it exists to catch leaves that derivation "+
				"correct.", literal, derivation)
		}
	}
}

// --- helpers -------------------------------------------------------------------

// storedSortKey is what one stored item contributes to the comparison.
type storedSortKey struct {
	name string
	key  string
}

// sortKeysByPlace keys the stored items the way the table does.
func sortKeysByPlace(t *testing.T, items []ports.ScannedItem) map[string]storedSortKey {
	t.Helper()

	held := make(map[string]storedSortKey, len(items))
	for _, item := range items {
		where := item.Path
		if where == "" {
			where = fmt.Sprintf("%s: %s", item.Type, item.Name)
		}
		if _, clash := held[where]; clash {
			t.Fatalf("two items are both at %q, so this comparison pairs them arbitrarily", where)
		}
		held[where] = storedSortKey{name: item.Name, key: item.SortKey}
	}
	return held
}

// namesInRowOrder is the table's own order, which is the order its keys name.
func namesInRowOrder(rows []sortKeyRow) []string {
	names := make([]string, len(rows))
	for i, row := range rows {
		names[i] = row.name
	}
	return names
}

// quoted puts a value between delimiters, because a trailing space and a double
// space are part of this contract and neither survives review otherwise (003
// T4).
func quoted(value string) string { return "|" + value + "|" }
