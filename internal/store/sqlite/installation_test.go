package sqlite

import (
	"context"
	"strings"
	"testing"

	"github.com/vdatanet/atrium-go/internal/units"
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
// T3 wrote the column here in SQL, because the method that writes it takes a
// date and the date type was T4's. T4 has it, so the write goes through the
// method — and the SQL that was the test is now the assertion, because what the
// column holds is a unit and a unit is not visible through a boolean.
func TestSetupCompletedIsTheColumnBeingSet(t *testing.T) {
	store := openForTest(t)
	ctx := context.Background()

	// 2025-06-19T00:00:00Z. Spelled as a date rather than as the tick count it
	// becomes, so that the arithmetic under test is not also the arithmetic the
	// expectation is built from.
	at, err := units.ParseTime("2025-06-19T00:00:00.0000000Z")
	if err != nil {
		t.Fatalf("ParseTime returned %v", err)
	}

	if err := store.MarkSetupComplete(ctx, at); err != nil {
		t.Fatalf("MarkSetupComplete returned %v", err)
	}

	installation, err := store.Installation(ctx)
	if err != nil {
		t.Fatalf("Installation returned %v", err)
	}
	if !installation.SetupCompleted {
		t.Error("SetupCompleted is false with the column set, want true")
	}

	// The column is ticks since 0001-01-01T00:00:00Z. That instant is
	// 1750291200 seconds after the Unix epoch, which is itself 62135596800
	// seconds after year one, so the tick count is (1750291200 + 62135596800)
	// times ten million. A store that wrote Unix seconds, Unix ticks or
	// nanoseconds would still make SetupCompleted true, which is exactly why the
	// boolean cannot be the whole test.
	var stored int64
	if err := store.reader.QueryRow(
		`SELECT setup_completed_at FROM installation WHERE id = 1`).Scan(&stored); err != nil {
		t.Fatalf("reading the column back: %v", err)
	}
	if want := int64((1750291200 + 62135596800) * 10_000_000); stored != want {
		t.Errorf("setup_completed_at is %d, want %d ticks since 0001-01-01T00:00:00Z", stored, want)
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
