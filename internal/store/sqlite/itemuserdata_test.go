package sqlite

import (
	"context"
	"testing"

	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/units"
)

// The store's own half of 003 AC-11. The criterion itself is asserted through
// the subcommand, over a real scan of a real tree, in `internal/app` — a store
// method cannot be asked whether a *scan* left a row alone. What is asserted
// here is what that test rests on: the table is precious, the round trip is
// whole, a row may name an item the store does not hold, and a rebuild of the
// derived half does not reach it.

// anAccountFor writes one account, because item user data is keyed on a user
// and the foreign key into `users` is real.
func anAccountFor(t *testing.T, store *Store, id, username string) string {
	t.Helper()
	if err := store.CreateUser(context.Background(), ports.User{
		ID: id, Username: username, UsernameFolded: username,
		PolicyDocument: []byte("{}"), ConfigurationDocument: []byte("{}"),
	}); err != nil {
		t.Fatalf("creating the account %q: %v", username, err)
	}
	return id
}

// TestTheItemUserDataMigrationIsFiledUnderThePreciousLineage is the clause the
// whole of AC-11 rests on, and it is the same clause 002's and 003's earlier
// migrations each carry.
//
// Filed under the derived directory this migration would create exactly the
// same table and every other assertion in this file would pass — and a rescan
// would then be entitled to drop it, which is 003 §3.8's *"user data outlives
// items"* deleted by the thing it exists to survive.
func TestTheItemUserDataMigrationIsFiledUnderThePreciousLineage(t *testing.T) {
	filedUnderThePreciousLineage(t, "0004_item_user_data.sql")
}

// TestAFirstStartCreatesTheItemUserDataTable is the same clause seen from a
// start rather than from the runner.
func TestAFirstStartCreatesTheItemUserDataTable(t *testing.T) {
	store := openForTest(t)

	var name string
	if err := store.reader.QueryRow(
		`SELECT name FROM sqlite_schema WHERE type = 'table' AND name = ?`, "item_user_data",
	).Scan(&name); err != nil {
		t.Errorf("the item_user_data table is not there after a first start: %v", err)
	}

	theDerivedHalfIsAtItsGeneration(t, store, "a first start")
}

// TestItemUserDataRoundTripsAndReplaces is the round trip and the upsert in one
// test, because the second is the first performed twice.
//
// The two values are deliberately both non-default in the first write and both
// different in the second: a favourite that stayed `true` and a position that
// stayed `0` is a test that passes on a build writing one column and ignoring
// the other.
func TestItemUserDataRoundTripsAndReplaces(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)
	user := anAccountFor(t, store, "0000000000000000000000000000ada0", "ada")

	const item = "1111111111111111111111111111aaaa"

	if _, found, err := store.ItemUserData(ctx, user, item); err != nil || found {
		t.Fatalf("ItemUserData found %v (err %v) before anything was written", found, err)
	}

	first := ItemUserData{IsFavourite: true, PlaybackPositionTicks: units.Ticks(123_456_789)}
	if err := store.SetItemUserData(ctx, user, item, first); err != nil {
		t.Fatalf("SetItemUserData returned %v", err)
	}
	if held, found, err := store.ItemUserData(ctx, user, item); err != nil || !found || held != first {
		t.Errorf("ItemUserData returned %+v, %v, %v; want %+v, true, nil", held, found, err, first)
	}

	second := ItemUserData{IsFavourite: false, PlaybackPositionTicks: units.Ticks(42)}
	if err := store.SetItemUserData(ctx, user, item, second); err != nil {
		t.Fatalf("SetItemUserData a second time returned %v", err)
	}
	if held, found, err := store.ItemUserData(ctx, user, item); err != nil || !found || held != second {
		t.Errorf("ItemUserData returned %+v, %v, %v after a replacement; want %+v, true, nil",
			held, found, err, second)
	}
}

// TestItemUserDataMayNameAnItemTheStoreDoesNotHold is the absence that makes
// AC-11 possible, asserted rather than assumed.
//
// 003 plan §6.5: no precious row references the derived half by row id, so a
// removed item's user data *"keeps naming a string that will exist again when
// the file returns"*. A foreign key into `items` — or the `ON DELETE CASCADE`
// that usually comes with one — would make that impossible, and it would make
// the derived half's rebuild refuse as well.
//
// The write below names an identifier no scan ever produced, and the read finds
// it. It is also asserted from the other side: the foreign key into `users`
// **is** real, so an account nobody has is refused. Without that half this test
// would pass on a build with no constraints at all.
func TestItemUserDataMayNameAnItemTheStoreDoesNotHold(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)
	user := anAccountFor(t, store, "0000000000000000000000000000ada0", "ada")

	const noSuchItem = "deadbeefdeadbeefdeadbeefdeadbeef"
	held := ItemUserData{IsFavourite: true, PlaybackPositionTicks: units.Ticks(7)}
	if err := store.SetItemUserData(ctx, user, noSuchItem, held); err != nil {
		t.Fatalf("writing user data for an item the store does not hold returned %v, and that "+
			"state is 003 §3.8's \"in case it returns\"", err)
	}
	if got, found, err := store.ItemUserData(ctx, user, noSuchItem); err != nil || !found || got != held {
		t.Errorf("ItemUserData returned %+v, %v, %v; want %+v, true, nil", got, found, err, held)
	}

	if err := store.SetItemUserData(ctx, "0000000000000000000000000000beef", noSuchItem, held); err == nil {
		t.Error("writing user data for an account nobody has was accepted, so the foreign key " +
			"into users is not there and the clause above proves nothing")
	}
}

// TestARebuildOfTheDerivedHalfLeavesItemUserDataAlone is ADR-0003's central
// claim aimed at this table.
//
// A rebuild is the act 003 plan §6.8 performs at start whenever the derived
// generation moves, and it drops everything the derived schema declares. A
// table filed on the wrong side of that line loses every row the moment
// somebody ships a new schema — silently, on an operator's installation, with
// no test between the change and the loss.
func TestARebuildOfTheDerivedHalfLeavesItemUserDataAlone(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)
	user := anAccountFor(t, store, "0000000000000000000000000000ada0", "ada")

	const item = "1111111111111111111111111111aaaa"
	held := ItemUserData{IsFavourite: true, PlaybackPositionTicks: units.Ticks(999)}
	if err := store.SetItemUserData(ctx, user, item, held); err != nil {
		t.Fatalf("SetItemUserData returned %v", err)
	}

	if err := store.RebuildDerived(ctx); err != nil {
		t.Fatalf("RebuildDerived returned %v", err)
	}

	got, found, err := store.ItemUserData(ctx, user, item)
	if err != nil || !found {
		t.Fatalf("the user data is gone after the derived half was rebuilt: found %v, err %v",
			found, err)
	}
	if got != held {
		t.Errorf("the user data reads back %+v after a rebuild, want %+v", got, held)
	}
}
