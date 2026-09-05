package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/vdatanet/atrium-go/internal/units"
)

// ItemUserData is the per-user, per-item state 003 §3.8 requires outlive an
// item: *"a file that disappears and comes back … must not cost the user their
// favourites and resume position"*.
//
// It is the two nouns of that sentence and nothing more. 007 owns this data and
// four more properties beside it (played, play count, last played date, and a
// session's live playstate); none of those is named by 003, so none of them is
// declared here. See `migrations/precious/0004_item_user_data.sql` for why the
// table is nevertheless 003's.
type ItemUserData struct {
	// IsFavourite is 003 §3.8's *"favourites"*.
	IsFavourite bool

	// PlaybackPositionTicks is 003 §3.8's *"resume position"*.
	PlaybackPositionTicks units.Ticks
}

// SetItemUserData writes one account's state for one item, replacing any it
// already had.
//
// **It is deliberately not on a `ports` interface.** 003 declares no domain
// that reads or writes user data — plan §6.5 is explicit that this feature has
// *"no retention rule to write and no orphan to sweep"* — so a port method here
// would be a contract with no caller above it, and it would fix the shape of a
// method 007 has to design. What this pair is for is the assertion AC-11's
// middle clause makes: that a precious row keyed on an item's identifier is
// untouched by the scan that removes the item.
//
// itemID is not checked against `items`, and that absence is the behaviour.
// A row naming an identifier no item currently has is 003 §3.8's *"in case it
// returns"*, which is the state the whole criterion is about.
func (s *Store) SetItemUserData(ctx context.Context, userID, itemID string, data ItemUserData) error {
	_, err := s.writer.ExecContext(ctx,
		`INSERT INTO item_user_data (user_id, item_id, is_favourite, playback_position_ticks)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(user_id, item_id) DO UPDATE SET
		     is_favourite = excluded.is_favourite,
		     playback_position_ticks = excluded.playback_position_ticks`,
		userID, itemID, boolToInteger(data.IsFavourite), int64(data.PlaybackPositionTicks))
	if err != nil {
		// The account's own row is what the foreign key protects, so a user
		// identifier nobody has fails here rather than writing state nothing
		// can reach. The *item* identifier is unconstrained on purpose.
		return fmt.Errorf("%s: writing item user data: %w", s.path, err)
	}
	return nil
}

// ItemUserData reads one account's state for one item, and reports whether
// there is any.
//
// Absent and zero are different answers here for the reason they are different
// everywhere else in this store: an item nobody has marked and an item somebody
// un-marked are the same values and not the same fact, and AC-11 asserts that a
// row **is there**, which a zero-valued struct cannot say.
func (s *Store) ItemUserData(ctx context.Context, userID, itemID string) (ItemUserData, bool, error) {
	var (
		data        ItemUserData
		isFavourite int64
		position    int64
	)
	err := s.reader.QueryRowContext(ctx,
		`SELECT is_favourite, playback_position_ticks FROM item_user_data
		 WHERE user_id = ? AND item_id = ?`, userID, itemID).Scan(&isFavourite, &position)
	if errors.Is(err, sql.ErrNoRows) {
		return ItemUserData{}, false, nil
	}
	if err != nil {
		return ItemUserData{}, false, fmt.Errorf("%s: reading item user data: %w", s.path, err)
	}
	data.IsFavourite = isFavourite != 0
	data.PlaybackPositionTicks = units.Ticks(position)
	return data, true, nil
}

// boolToInteger is SQLite's spelling of a boolean, which is an INTEGER under
// STRICT and has no other representation.
func boolToInteger(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
