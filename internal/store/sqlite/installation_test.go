package sqlite

import (
	"context"
	"strings"
	"testing"
)

// TestASecondInstallationRowIsRefused is T3's third clause and plan 4's
// CHECK (id = 1).
//
// A configuration table with two rows is a bug that reads as a mystery: every
// query returns whichever row the planner reached first, so the server answers
// with one name today and the other after an index is added. The constraint
// turns that into a failure at the moment the second row is written.
func TestASecondInstallationRowIsRefused(t *testing.T) {
	store := openForTest(t)

	for _, c := range []struct {
		name      string
		statement string
	}{
		{"a row with its own id", `INSERT INTO installation (id, server_name) VALUES (2, 'second')`},
		{"a row letting SQLite choose the id", `INSERT INTO installation (server_name) VALUES ('second')`},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := store.writer.Exec(c.statement)
			if err == nil {
				t.Fatal("the insert was accepted, want the CHECK to refuse it")
			}
			if !strings.Contains(err.Error(), "CHECK") {
				t.Errorf("the insert failed with %q, want the CHECK constraint", err)
			}
		})
	}

	var rows int
	if err := store.reader.QueryRow(`SELECT count(*) FROM installation`).Scan(&rows); err != nil {
		t.Fatalf("counting the rows: %v", err)
	}
	if rows != 1 {
		t.Errorf("the installation table holds %d rows, want exactly 1", rows)
	}
}

// TestInstallationReportsAFreshInstallation reads the seeded row back through
// the port the domain declared, which is the only shape a handler will see.
func TestInstallationReportsAFreshInstallation(t *testing.T) {
	store := openForTest(t)

	installation, err := store.Installation(context.Background())
	if err != nil {
		t.Fatalf("Installation returned %v", err)
	}
	if installation.Name != "atrium" {
		t.Errorf("Name is %q, want %q (spec 3.1)", installation.Name, "atrium")
	}
	if installation.SetupCompleted {
		t.Error("SetupCompleted is true on a fresh installation, want false")
	}
}

// TestSetupCompletedIsTheColumnBeingSet pins plan 4's derivation:
// StartupWizardCompleted is setup_completed_at IS NOT NULL, and the instant
// itself never crosses the port.
//
// The column is written here in SQL rather than through a method because the
// method that writes it takes a date, and the tick type it would take is T4's.
func TestSetupCompletedIsTheColumnBeingSet(t *testing.T) {
	store := openForTest(t)

	if _, err := store.writer.Exec(
		`UPDATE installation SET setup_completed_at = ? WHERE id = 1`, int64(638000000000000000)); err != nil {
		t.Fatalf("recording the completion: %v", err)
	}

	installation, err := store.Installation(context.Background())
	if err != nil {
		t.Fatalf("Installation returned %v", err)
	}
	if !installation.SetupCompleted {
		t.Error("SetupCompleted is false with the column set, want true")
	}
}

// TestSetServerNameReplacesTheName covers the other half of the port, and that
// the write is visible to the reading pool — two handles on one file, which is
// the arrangement that would hide a write in a transaction nobody committed.
func TestSetServerNameReplacesTheName(t *testing.T) {
	store := openForTest(t)
	ctx := context.Background()

	if err := store.SetServerName(ctx, "the front room"); err != nil {
		t.Fatalf("SetServerName returned %v", err)
	}

	installation, err := store.Installation(ctx)
	if err != nil {
		t.Fatalf("Installation returned %v", err)
	}
	if installation.Name != "the front room" {
		t.Errorf("Name is %q, want %q", installation.Name, "the front room")
	}
}

// TestAMissingInstallationRowIsSaidOutLoud. The migration seeds the row, so its
// absence is a database somebody edited rather than an installation that has
// not been configured — and an empty name served as a real answer would be a
// wrong ServerName on the response a multi-server client decides on.
func TestAMissingInstallationRowIsSaidOutLoud(t *testing.T) {
	store := openForTest(t)

	if _, err := store.writer.Exec(`DELETE FROM installation WHERE id = 1`); err != nil {
		t.Fatalf("deleting the row: %v", err)
	}

	if _, err := store.Installation(context.Background()); err == nil {
		t.Fatal("Installation returned a value with no row, want an error")
	}
	if err := store.SetServerName(context.Background(), "nobody"); err == nil {
		t.Fatal("SetServerName reported success with no row to change, want an error")
	}
}
