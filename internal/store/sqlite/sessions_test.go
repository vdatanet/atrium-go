package sqlite

import (
	"bytes"
	"context"
	"testing"

	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/units"
)

// The two identifiers the session tests share, beside T1's testSessionID. They
// are the shape 002 plan 6.5 derives rather than convenient short strings.
const (
	otherUserID       = "8fa14cdd754f91cc6554c9e71929cce7"
	otherSessionID    = "b4b147bc522828731f1a016bfa72c073"
	aTokenDigest      = "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"
	anotherTokenDiges = "6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b"
)

// aSession builds a session with the fixed dates the tests compare against.
func aSession(id, userID, client, deviceID string) ports.Session {
	return ports.Session{
		ID:                    id,
		UserID:                userID,
		Client:                client,
		DeviceID:              deviceID,
		DeviceName:            "a device",
		ApplicationVersion:    "1.0.0",
		RemoteEndpoint:        "127.0.0.1",
		CreatedAt:             aLoginInstant,
		LastActivityAt:        aLoginInstant,
		LastPlaybackCheckInAt: units.TimeFromTicks(0),
	}
}

// TestOpenSessionIsOneStatement is T4's second clause: a failure part-way
// through leaves neither the session row nor the token digest.
//
// The failure is injected through the data rather than through a seam in the
// code, and the schema is what makes that a genuine mid-way failure. A token
// names its session by a foreign key, so the session row *has* to be written
// first — no implementation of this method can put the token first — and a
// token digest is the access_tokens primary key, so a digest that already
// exists makes the second write fail after the first has succeeded. The
// collision is not the scenario being defended against (a digest is SHA-256 of
// 128 bits of randomness); it is the cheapest way to fail the second statement
// of two whose order the database already pins.
//
// What a non-transactional build leaves behind is a session nothing can reach,
// and — the day the two writes are ever reordered — a token whose session is
// missing, which is a credential resolving to a caller with no client, no
// device and no activity to stamp.
func TestOpenSessionIsOneStatement(t *testing.T) {
	store := openForTest(t)
	ctx := context.Background()

	if err := insertUser(t, store.writer, testUserID, "Alice", "alice"); err != nil {
		t.Fatalf("inserting the account returned %v", err)
	}

	live := aSession(testSessionID, testUserID, "music-client", "device-1")
	if err := store.OpenSession(ctx, live, aTokenDigest); err != nil {
		t.Fatalf("opening the first session returned %v", err)
	}

	// A different session, on a different client and device, whose token
	// collides with the one already stored.
	doomed := aSession(otherSessionID, testUserID, "video-client", "device-2")
	if err := store.OpenSession(ctx, doomed, aTokenDigest); err == nil {
		t.Fatal("opening a session whose token digest already exists succeeded, want a refusal")
	}

	open, err := store.Sessions(ctx)
	if err != nil {
		t.Fatalf("Sessions returned %v", err)
	}
	if len(open) != 1 {
		t.Fatalf("there are %d sessions after the failed open, want 1: the session row survived a "+
			"write that did not complete", len(open))
	}
	if open[0].ID != live.ID {
		t.Errorf("the surviving session is %s, want %s", open[0].ID, live.ID)
	}

	// And nothing about the token moved either: the digest still resolves to
	// the session it was issued for, so the failed call wrote nothing at all.
	session, tokenUser, found, err := store.SessionByTokenDigest(ctx, aTokenDigest)
	if err != nil || !found {
		t.Fatalf("SessionByTokenDigest returned (%t, %v), want the first session", found, err)
	}
	if session.ID != live.ID || tokenUser != testUserID {
		t.Errorf("the digest resolves to session %s held by %s, want %s held by %s",
			session.ID, tokenUser, live.ID, testUserID)
	}
}

// TestOpeningTheSameClientAndDeviceReplacesTheSessionRatherThanAccumulating is
// spec 3.3's "authenticating again from the same DeviceId replaces that session
// rather than accumulating one per login", at the store.
//
// The identifier is derived from (Client, DeviceId), so the second login
// arrives at the same row; what the update carries is 002 plan 6.5 step 4's
// list, and what it deliberately leaves alone is asserted too.
func TestOpeningTheSameClientAndDeviceReplacesTheSessionRatherThanAccumulating(t *testing.T) {
	store := openForTest(t)
	ctx := context.Background()

	if err := insertUser(t, store.writer, testUserID, "Alice", "alice"); err != nil {
		t.Fatalf("inserting the account returned %v", err)
	}
	if err := insertUser(t, store.writer, otherUserID, "Bob", "bob"); err != nil {
		t.Fatalf("inserting the second account returned %v", err)
	}

	first := aSession(testSessionID, testUserID, "music-client", "device-1")
	if err := store.OpenSession(ctx, first, aTokenDigest); err != nil {
		t.Fatalf("opening the session returned %v", err)
	}

	// A client posts its capabilities, so that the second login can be seen
	// not to discard them.
	declaration := []byte(`{"PlayableMediaTypes":["Audio"]}`)
	if err := store.ReplaceCapabilities(ctx, testSessionID, declaration); err != nil {
		t.Fatalf("ReplaceCapabilities returned %v", err)
	}

	// Somebody else logs in on the same client and the same device, later.
	later := units.TimeFromTicks(aLoginInstant.Ticks() + units.TicksPerSecond)
	second := aSession(testSessionID, otherUserID, "music-client", "device-1")
	second.DeviceName = "a renamed device"
	second.ApplicationVersion = "2.0.0"
	second.RemoteEndpoint = "192.0.2.7"
	second.CreatedAt = later
	second.LastActivityAt = later
	if err := store.OpenSession(ctx, second, anotherTokenDiges); err != nil {
		t.Fatalf("opening the session again returned %v", err)
	}

	open, err := store.Sessions(ctx)
	if err != nil {
		t.Fatalf("Sessions returned %v", err)
	}
	if len(open) != 1 {
		t.Fatalf("there are %d sessions after two logins from one client and device, want 1", len(open))
	}
	got := open[0]

	// What the replacement carries.
	if got.UserID != otherUserID {
		t.Errorf("the session names %s, want %s: the row names whoever authenticated last "+
			"(002 plan 6.5)", got.UserID, otherUserID)
	}
	if got.DeviceName != "a renamed device" || got.ApplicationVersion != "2.0.0" || got.RemoteEndpoint != "192.0.2.7" {
		t.Errorf("the session is %q/%q/%q after the second login, want the second login's values",
			got.DeviceName, got.ApplicationVersion, got.RemoteEndpoint)
	}
	if !got.LastActivityAt.Equal(later) {
		t.Errorf("last_activity_at is %s, want %s: a login is itself activity", got.LastActivityAt, later)
	}

	// And what it does not. The session is the same session, so its creation
	// date does not move and the declaration a client posted is not discarded
	// by somebody else's login.
	if !got.CreatedAt.Equal(aLoginInstant) {
		t.Errorf("created_at is %s after the second login, want %s: the row is the same session",
			got.CreatedAt, aLoginInstant)
	}
	if !bytes.Equal(got.CapabilitiesDocument, declaration) {
		t.Errorf("the capabilities document is %s after the second login, want %s",
			got.CapabilitiesDocument, declaration)
	}
}

// TestATokenResolvesToItsOwnUserAndNotTheSessionsIsWhySessionByTokenDigestAnswersTwo
// is the reason this method returns a user beside the session.
//
// 002 plan 6.5: a session is keyed on (Client, DeviceId) and names whoever
// authenticated there last, while a token is keyed on (user, device). Two
// people sharing one client on one device therefore hold two live tokens
// against one session row — and a caller resolved from the session's user would
// be whoever logged in most recently, on somebody else's account, with no error
// anywhere.
func TestATokenResolvesToItsOwnUserAndNotTheSessionsIsWhySessionByTokenDigestAnswersTwo(t *testing.T) {
	store := openForTest(t)
	ctx := context.Background()

	if err := insertUser(t, store.writer, testUserID, "Alice", "alice"); err != nil {
		t.Fatalf("inserting the first account returned %v", err)
	}
	if err := insertUser(t, store.writer, otherUserID, "Bob", "bob"); err != nil {
		t.Fatalf("inserting the second account returned %v", err)
	}

	// Alice logs in, then Bob logs in on the same client and device. Alice's
	// token is not revoked here, because RevokeTokensFor is keyed on
	// (user, device) and Bob's login revokes Bob's tokens.
	if err := store.OpenSession(ctx, aSession(testSessionID, testUserID, "music-client", "device-1"), aTokenDigest); err != nil {
		t.Fatalf("opening Alice's session returned %v", err)
	}
	if err := store.OpenSession(ctx, aSession(testSessionID, otherUserID, "music-client", "device-1"), anotherTokenDiges); err != nil {
		t.Fatalf("opening Bob's session returned %v", err)
	}

	session, tokenUser, found, err := store.SessionByTokenDigest(ctx, aTokenDigest)
	if err != nil || !found {
		t.Fatalf("SessionByTokenDigest returned (%t, %v), want Alice's token", found, err)
	}
	if tokenUser != testUserID {
		t.Errorf("Alice's token resolves to user %s, want %s", tokenUser, testUserID)
	}
	if session.UserID != otherUserID {
		t.Errorf("the session names %s, want %s — and the two differing is the whole reason this "+
			"method answers both", session.UserID, otherUserID)
	}

	// An unknown token is an absence and not a failure: 002 plan 7 makes it
	// indistinguishable from no credential at all, and a store error taking
	// that path would tell a client its perfectly good credential was wrong.
	if _, _, found, err := store.SessionByTokenDigest(ctx, "not a digest anybody minted"); err != nil || found {
		t.Errorf("an unknown digest returned (%t, %v), want (false, nil)", found, err)
	}
}

// TestRevokingTokensLeavesTheSameUsersTokenOnAnotherDeviceLive is T4's third
// clause, and the pair in the WHERE clause is what it is about.
//
// This is the clause spec 3.3's "authenticating again from the same DeviceId
// replaces that session" turns on. An over-broad DELETE ... WHERE user_id = ?
// removes it invisibly: nothing in any response reports it, the client that was
// logged out simply gets a 401 hours later and re-authenticates, and a test
// asserting only that the replaced token is gone passes on that build.
//
// The other half of the pair is asserted too — somebody else's token on the
// same device survives — because a DELETE by device alone is the mirror mistake
// and logs out every user of a shared device.
func TestRevokingTokensLeavesTheSameUsersTokenOnAnotherDeviceLive(t *testing.T) {
	store := openForTest(t)
	ctx := context.Background()

	if err := insertUser(t, store.writer, testUserID, "Alice", "alice"); err != nil {
		t.Fatalf("inserting the first account returned %v", err)
	}
	if err := insertUser(t, store.writer, otherUserID, "Bob", "bob"); err != nil {
		t.Fatalf("inserting the second account returned %v", err)
	}

	const (
		aliceOnDeviceOne = aTokenDigest
		aliceOnDeviceTwo = anotherTokenDiges
		bobOnDeviceOne   = "d4735e3a265e16eee03f59718b9b5d03019c07d8b6c51f90da3a666eec13ab35"
	)

	// Alice on two devices, and Bob on the first of them. All three tokens are
	// live.
	if err := store.OpenSession(ctx, aSession(testSessionID, testUserID, "music-client", "device-1"), aliceOnDeviceOne); err != nil {
		t.Fatalf("opening Alice's first session returned %v", err)
	}
	if err := store.OpenSession(ctx, aSession(otherSessionID, testUserID, "music-client", "device-2"), aliceOnDeviceTwo); err != nil {
		t.Fatalf("opening Alice's second session returned %v", err)
	}
	if err := store.OpenSession(ctx, aSession(testSessionID, otherUserID, "music-client", "device-1"), bobOnDeviceOne); err != nil {
		t.Fatalf("opening Bob's session returned %v", err)
	}

	if err := store.RevokeTokensFor(ctx, testUserID, "device-1"); err != nil {
		t.Fatalf("RevokeTokensFor returned %v", err)
	}

	if _, _, found, err := store.SessionByTokenDigest(ctx, aliceOnDeviceOne); err != nil || found {
		t.Errorf("Alice's token on device-1 returned (%t, %v) after revocation, want it gone", found, err)
	}
	if _, user, found, err := store.SessionByTokenDigest(ctx, aliceOnDeviceTwo); err != nil || !found || user != testUserID {
		t.Errorf("Alice's token on device-2 returned (%q, %t, %v), want it still live: a revocation "+
			"by user alone would have logged her out of every device", user, found, err)
	}
	if _, user, found, err := store.SessionByTokenDigest(ctx, bobOnDeviceOne); err != nil || !found || user != otherUserID {
		t.Errorf("Bob's token on device-1 returned (%q, %t, %v), want it still live: a revocation "+
			"by device alone would have logged out every user of a shared device", user, found, err)
	}

	// Revoking where there is nothing to revoke is not a failure. The first
	// login from a device revokes nothing, and a caller that had to treat that
	// as an error would refuse every first login.
	if err := store.RevokeTokensFor(ctx, testUserID, "a device nobody has used"); err != nil {
		t.Errorf("revoking tokens that do not exist returned %v, want no error", err)
	}
}

// TestACapabilitiesDeclarationIsStoredWholeAndActivityIsStamped covers the two
// session writers that are not the open.
//
// The document is stored raw and unread, which is what makes behaviours 5.9's
// divergence — an unknown capabilities property surviving into /Sessions — the
// stated one rather than an accident. So the property this asserts is
// byte-equality, and an unknown member is in the document precisely to make
// that mean something.
func TestACapabilitiesDeclarationIsStoredWholeAndActivityIsStamped(t *testing.T) {
	store := openForTest(t)
	ctx := context.Background()

	if err := insertUser(t, store.writer, testUserID, "Alice", "alice"); err != nil {
		t.Fatalf("inserting the account returned %v", err)
	}
	if err := store.OpenSession(ctx, aSession(testSessionID, testUserID, "music-client", "device-1"), aTokenDigest); err != nil {
		t.Fatalf("opening the session returned %v", err)
	}

	// A fresh session has posted nothing, which is an absence and not an empty
	// declaration.
	open, err := store.Sessions(ctx)
	if err != nil {
		t.Fatalf("Sessions returned %v", err)
	}
	if open[0].CapabilitiesDocument != nil {
		t.Errorf("a session that has posted nothing carries %s, want no document at all",
			open[0].CapabilitiesDocument)
	}
	if spelling := open[0].LastPlaybackCheckInAt.String(); spelling != "0001-01-01T00:00:00.0000000Z" {
		t.Errorf("LastPlaybackCheckIn on a session that never played is %q, want %q (spec 3.3)",
			spelling, "0001-01-01T00:00:00.0000000Z")
	}

	declaration := []byte(`{"PlayableMediaTypes":["Audio"],"SomethingThisServerNeverHeardOf":true}`)
	if err := store.ReplaceCapabilities(ctx, testSessionID, declaration); err != nil {
		t.Fatalf("ReplaceCapabilities returned %v", err)
	}
	later := units.TimeFromTicks(aLoginInstant.Ticks() + units.TicksPerSecond)
	if err := store.TouchSession(ctx, testSessionID, later); err != nil {
		t.Fatalf("TouchSession returned %v", err)
	}

	open, err = store.Sessions(ctx)
	if err != nil {
		t.Fatalf("Sessions returned %v", err)
	}
	if !bytes.Equal(open[0].CapabilitiesDocument, declaration) {
		t.Errorf("the stored declaration is %s, want %s byte for byte",
			open[0].CapabilitiesDocument, declaration)
	}
	if !open[0].LastActivityAt.Equal(later) {
		t.Errorf("last_activity_at is %s, want %s", open[0].LastActivityAt, later)
	}

	// The same guard every other write carries: a session nobody has is an
	// error, not an UPDATE that quietly matched nothing.
	if err := store.ReplaceCapabilities(ctx, otherSessionID, declaration); err == nil {
		t.Error("capabilities stored against a session that does not exist succeeded, want an error")
	}
	if err := store.TouchSession(ctx, otherSessionID, later); err == nil {
		t.Error("activity stamped against a session that does not exist succeeded, want an error")
	}
}

// TestSessionsAreReturnedInAStatedOrder is architecture 2's ordering rule for
// the other list.
func TestSessionsAreReturnedInAStatedOrder(t *testing.T) {
	store := openForTest(t)
	ctx := context.Background()

	if err := insertUser(t, store.writer, testUserID, "Alice", "alice"); err != nil {
		t.Fatalf("inserting the account returned %v", err)
	}

	newer := aSession(testSessionID, testUserID, "video-client", "device-1")
	newer.CreatedAt = units.TimeFromTicks(aLoginInstant.Ticks() + units.TicksPerSecond)
	older := aSession(otherSessionID, testUserID, "music-client", "device-1")

	// Inserted newest first, so storage order is not the answer.
	if err := store.OpenSession(ctx, newer, aTokenDigest); err != nil {
		t.Fatalf("opening the newer session returned %v", err)
	}
	if err := store.OpenSession(ctx, older, anotherTokenDiges); err != nil {
		t.Fatalf("opening the older session returned %v", err)
	}

	open, err := store.Sessions(ctx)
	if err != nil {
		t.Fatalf("Sessions returned %v", err)
	}
	if len(open) != 2 {
		t.Fatalf("there are %d sessions, want 2", len(open))
	}
	if open[0].ID != older.ID || open[1].ID != newer.ID {
		t.Errorf("Sessions returned %s then %s, want the older one first", open[0].ID, open[1].ID)
	}
}
