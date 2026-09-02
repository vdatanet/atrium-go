// Package sqlite is the store: an embedded SQLite database in the data
// directory, opened once at start and closed once at stop.
//
// It implements the interfaces internal/ports declares and adds nothing above
// that line. No SQL, no transaction and no vocabulary belonging to a database
// leaves this package (architecture 6), which is what keeps ADR-0003 a decision
// the rest of the tree could survive changing.
//
// ADR-0003 decides everything technical here: embedded SQLite, a pure-Go driver
// so CGO_ENABLED=0 holds, hand-written SQL over database/sql, and the split
// into a derived half a rescan rebuilds and a precious half that is the only
// copy. Half is where that split is spelled out.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	// The pure-Go driver of ADR-0003, registered as "sqlite". It is the whole
	// reason this package exists rather than a thinner one: it costs about
	// 2.3x on the hot read path against the cgo driver, and buys a static
	// binary and a cross-compilable build.
	_ "modernc.org/sqlite"

	"github.com/vdatanet/atrium-go/internal/ports"
)

// Store is the open database: one handle that writes and a pool that reads.
//
// The two are separate handles rather than one pool because they are opened
// differently and used differently. SQLite takes one writer at a time, so the
// writing handle is capped at a single connection and serialises callers in Go
// rather than discovering the same limit as a busy error; readers are
// concurrent, which is what write-ahead logging buys and what ADR-0003
// measured — 57,664 reads completed during one write transaction inserting
// 30,000 rows, worst read latency 393 microseconds
// [measurement: modernc.org/sqlite v1.58.0, Go 1.27.0, 2026-09-02].
type Store struct {
	path    string
	writer  *sql.DB
	reader  *sql.DB
	applied map[Half][]int
}

// Store satisfies what the domain asked for. The assertion is here rather than
// in a test because it is the only thing that makes the interface load-bearing:
// nothing in 001 has a handler yet, so without it the port and the store could
// drift apart until the first one is written.
var _ ports.InstallationStore = (*Store)(nil)

// DatabaseFile is the name, within the data directory, of the database.
//
// One file, not one per half. The halves are two migration lineages and two
// rebuild policies, not two databases: ADR-0003 wants backup to be one file,
// and a precious row naming a derived item by its identifier is an ordinary
// join that would otherwise need the two halves attached to each other. Their
// separation is enforced by the rule that no reference points from the precious
// half into the derived one (architecture 6), which a second file would not
// enforce either.
//
// Write-ahead logging adds "-wal" and "-shm" beside it while the database is
// open; SQLite removes both when the last connection closes.
const DatabaseFile = "atrium.db"

// maxIdleReaders is how many reading connections are kept open between queries.
//
// An idle SQLite connection is a file descriptor and a page cache, not a
// session on a server somewhere, so keeping a few costs little and reopening
// one per request costs a file open and a set of pragmas. The number of
// concurrent readers is not capped: ADR-0003's concurrency measurement was one
// reader loop and is marked unverified there, and a cap chosen without a
// measurement is a number nobody could defend later.
const maxIdleReaders = 8

// Open opens the database in dataDirectory, applies every pending migration of
// both halves, and returns the store.
//
// The migrations run here rather than behind a separate call because a store
// handed to a caller before its schema exists has exactly one correct use, and
// nothing enforces it. plan 4 says migrations are applied at start; this is
// that start, and plan 7 makes a failure here a refusal to start.
//
// It does not create dataDirectory. That is the entry layer's, which creates it
// before it reads the installation identity — the identity file is read first
// and would otherwise fail before this could.
func Open(ctx context.Context, dataDirectory string) (*Store, error) {
	path := filepath.Join(dataDirectory, DatabaseFile)

	// Checked before the driver is asked, so that a data directory that is not
	// there is reported as that rather than as SQLite's "unable to open
	// database file", which is the same message for a missing directory, a
	// permission problem and a corrupt header.
	if info, err := os.Stat(dataDirectory); err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	} else if !info.IsDir() {
		return nil, fmt.Errorf("opening %s: %s is not a directory", path, dataDirectory)
	}

	writer, err := sql.Open(driverName, writerDSN(path))
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	// One connection, because SQLite takes one writer. Callers then queue in
	// database/sql instead of colliding in the engine, and the busy timeout is
	// left for the writers this process does not own — a backup tool, or an
	// operator with a shell.
	writer.SetMaxOpenConns(1)
	writer.SetMaxIdleConns(1)

	if err := writer.PingContext(ctx); err != nil {
		writer.Close()
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}

	store := &Store{path: path, writer: writer, applied: map[Half][]int{}}

	if err := bootstrap(ctx, writer); err != nil {
		writer.Close()
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	for _, half := range halves {
		lineage, err := loadLineage(migrationFiles, half)
		if err != nil {
			writer.Close()
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		applied, err := migrate(ctx, writer, half, lineage)
		store.applied[half] = applied
		if err != nil {
			writer.Close()
			return nil, fmt.Errorf("%s: %w", path, err)
		}
	}

	// After the migrations, so that the file the readers open is one with a
	// schema in it.
	reader, err := sql.Open(driverName, readerDSN(path))
	if err != nil {
		writer.Close()
		return nil, fmt.Errorf("opening %s for reading: %w", path, err)
	}
	reader.SetMaxIdleConns(maxIdleReaders)
	if err := reader.PingContext(ctx); err != nil {
		reader.Close()
		writer.Close()
		return nil, fmt.Errorf("opening %s for reading: %w", path, err)
	}
	store.reader = reader

	return store, nil
}

// Close releases both handles. Both are closed even when the first fails, and
// both failures are reported: a handle left open holds the write-ahead log open
// with it, and a Close that gave up halfway would leave one behind on exactly
// the path where somebody is already reading an error message.
func (s *Store) Close() error {
	var readerErr error
	if s.reader != nil {
		readerErr = s.reader.Close()
	}
	var writerErr error
	if s.writer != nil {
		writerErr = s.writer.Close()
	}
	if err := errors.Join(readerErr, writerErr); err != nil {
		return fmt.Errorf("closing %s: %w", s.path, err)
	}
	return nil
}

// Path is where the database is, for a log line or an error message that has to
// tell an operator which file to look at.
func (s *Store) Path() string { return s.path }

// SchemaVersion reports how many migrations of half this database has had.
func (s *Store) SchemaVersion(ctx context.Context, half Half) (int, error) {
	return schemaVersion(ctx, s.writer, half)
}

// AppliedMigrations returns the versions of half that this Open applied, which
// is empty on every start after the first.
//
// It is on the store rather than logged from inside it because the store has no
// logger and should not grow one: what a start did is the entry layer's to
// report, and a test's to assert.
func (s *Store) AppliedMigrations(half Half) []int {
	return s.applied[half]
}

// driverName is the name modernc.org/sqlite registers itself under. It is not
// "sqlite3", which is the cgo driver's, and getting it wrong is a panic at
// sql.Open rather than a compile error.
const driverName = "sqlite"

// writerDSN is the connection string for the writing handle.
//
// Every pragma ADR-0003 names is set here rather than executed after connecting,
// because database/sql may open a new connection at any time and one configured
// by a statement somebody ran once is configured only until then.
//
//   - journal_mode(WAL): a reader is not blocked while a scan writes.
//   - synchronous(NORMAL): the write-ahead log is not fsynced on every commit.
//     With WAL this risks the last transactions of a power cut, not the
//     database, and a scan commits in batches precisely so it can resume.
//   - foreign_keys(1): SQLite leaves them off by default, so a schema that
//     declares one and does not set this has a comment where it wanted a
//     constraint.
//   - busy_timeout: wait rather than fail when another process holds the write
//     lock — a backup, or an operator with a shell.
//   - _txlock=immediate: a write transaction takes the write lock when it
//     begins instead of upgrading part-way through, which is where SQLite's
//     unupgradeable deadlock lives.
func writerDSN(path string) string {
	return "file:" + path +
		"?_txlock=immediate" +
		"&_pragma=journal_mode(WAL)" +
		"&_pragma=synchronous(NORMAL)" +
		"&_pragma=foreign_keys(1)" +
		"&_pragma=busy_timeout(" + busyTimeoutMilliseconds + ")"
}

// readerDSN is the connection string for the reading pool.
//
// query_only(1) is the difference that matters: it makes the engine refuse a
// write on these connections, so "one writer handle and a pool of readers"
// is a property the database enforces rather than a convention this package
// asks callers to keep.
func readerDSN(path string) string {
	return "file:" + path +
		"?_pragma=query_only(1)" +
		"&_pragma=journal_mode(WAL)" +
		"&_pragma=foreign_keys(1)" +
		"&_pragma=busy_timeout(" + busyTimeoutMilliseconds + ")"
}

// busyTimeoutMilliseconds is how long a connection waits for a lock another
// process holds. Five seconds is long enough to outlast a checkpoint or a
// backup and short enough that a request blocked on one still answers.
const busyTimeoutMilliseconds = "5000"
