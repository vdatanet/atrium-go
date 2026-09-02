package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/vdatanet/atrium-go/internal/ports"
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
