package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/units"
)

// Installation reads the single row of the installation table.
//
// It reads through the reading pool: this is on the path of every
// /System/Info/Public, which is the response a multi-server client issues first
// and the one it decides on, so it must not queue behind whatever is writing.
func (s *Store) Installation(ctx context.Context) (ports.Installation, error) {
	var (
		name        string
		completedAt sql.Null[int64]
	)
	err := s.reader.QueryRowContext(ctx,
		`SELECT server_name, setup_completed_at FROM installation WHERE id = 1`,
	).Scan(&name, &completedAt)
	if errors.Is(err, sql.ErrNoRows) {
		// The migration seeds the row, so its absence is not "not configured
		// yet" — it is a database somebody edited. Saying so is worth more
		// than a zero value that would be served as a real answer.
		return ports.Installation{}, fmt.Errorf("%s: the installation row is missing", s.path)
	}
	if err != nil {
		return ports.Installation{}, fmt.Errorf("%s: reading the installation: %w", s.path, err)
	}

	// StartupWizardCompleted is this column being non-NULL (plan 4). The
	// instant itself does not cross the boundary, because no response carries
	// it and nothing decides on it.
	return ports.Installation{Name: name, SetupCompleted: completedAt.Valid}, nil
}

// SetServerName replaces the friendly name reported as ServerName.
//
// The name is not validated here. What an operator may call their server is a
// question about the wire — the reference's own limits, and what a client does
// with a name containing a newline — and answering it in the store would put a
// measured behaviour in the layer furthest from where it is measured.
func (s *Store) SetServerName(ctx context.Context, name string) error {
	result, err := s.writer.ExecContext(ctx,
		`UPDATE installation SET server_name = ? WHERE id = 1`, name)
	if err != nil {
		return fmt.Errorf("%s: setting the server name: %w", s.path, err)
	}

	// An UPDATE that matched nothing succeeds, which would make renaming a
	// server that has no row look like renaming it.
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: setting the server name: %w", s.path, err)
	}
	if affected != 1 {
		return fmt.Errorf("%s: setting the server name changed %d rows, want 1", s.path, affected)
	}
	return nil
}

// MarkSetupComplete records the instant initial configuration finished.
//
// The column holds .NET's DateTime.Ticks — 100-nanosecond intervals since
// 0001-01-01T00:00:00Z — which is what units.Time.Ticks returns. The unit was
// already the migration's ("in ticks", 0001_installation.sql); the origin is
// what this method pins, and it is pinned to .NET's because behaviours 1.3
// makes the tick .NET's tick and every date the wire carries is a .NET
// DateTime. A column holding ticks since some other epoch would be a second
// unit wearing the first one's name.
//
// It does not refuse a second call. Whether setup may be completed twice is a
// question about the wizard, which 002 owns, and a store that answered it here
// would be deciding it for a caller that has not been written.
func (s *Store) MarkSetupComplete(ctx context.Context, at units.Time) error {
	result, err := s.writer.ExecContext(ctx,
		`UPDATE installation SET setup_completed_at = ? WHERE id = 1`, int64(at.Ticks()))
	if err != nil {
		return fmt.Errorf("%s: recording that setup completed: %w", s.path, err)
	}

	// The same guard SetServerName carries, for the same reason: an UPDATE that
	// matched nothing succeeds, and a setup that completed against no row would
	// look exactly like one that completed.
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: recording that setup completed: %w", s.path, err)
	}
	if affected != 1 {
		return fmt.Errorf("%s: recording that setup completed changed %d rows, want 1", s.path, affected)
	}
	return nil
}
