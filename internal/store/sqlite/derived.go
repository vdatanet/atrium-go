package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"fmt"
	"regexp"
)

// derivedSchema is the whole current shape of the derived half.
//
// One file and not a lineage. ADR-0003 says the derived half is dropped and
// rescanned on a schema change and that no migration is ever written for it, so
// there is nothing here to apply the tail of: what a build knows is one schema
// and one number, and a database that disagrees with either is rebuilt rather
// than migrated (003 plan §6.8).
//
//go:embed derived/library.sql
var derivedSchema string

// derivedGeneration is the schema this build declares.
//
// It is an integer the build states rather than a fingerprint of the file,
// which is the third of the three shapes 003 plan §6.8 weighed. A fingerprint
// rebuilds on a whitespace change and on a comment, so a reader of the diff
// cannot tell whether a change was meant to cost every installation a full
// rescan; a number in the source says so in the diff itself.
//
// **Bump it whenever derived/library.sql changes.** Nothing else has to be
// written, and forgetting it is what derivedSchemaDigest below turns into a
// failing build rather than into a schema that silently disagrees with the
// queries above it.
const derivedGeneration = 1

// derivedSchemaDigest is the SHA-256 of derived/library.sql as
// derivedGeneration 1 shipped it.
//
// The pair is the one thing in this half that notices — 002 T1's "the
// constraint is redundant today on purpose" applied to a constant instead of to
// a UNIQUE. A generation nobody bumps is a schema change that ships as a silent
// corruption: every installation keeps the old tables and every query this
// package writes is compiled against the new ones. The test that compares these
// two fails on any edit to the file, including a comment, and the fix is
// always the same two lines.
const derivedSchemaDigest = "394ccf47ebc1670b3e49d999b54a0d5156a325f63042bd72223762a04d0fe848"

// derivedObject is one thing derived/library.sql creates, as the drop that
// precedes a rebuild has to name it.
type derivedObject struct {
	kind string // TABLE, INDEX, VIEW or TRIGGER, as DROP spells it
	name string
}

// createStatement matches the head of every object declaration in the derived
// schema.
//
// The drop list is read out of the schema rather than typed beside it, and that
// is the whole design of this function. A table added to the schema and
// forgotten in a hand-written drop list would survive a rebuild carrying its
// old columns — a database claiming this generation's shape while holding the
// last one's — and nothing that queries it would notice until a column had
// moved.
var createStatement = regexp.MustCompile(`(?im)^\s*CREATE\s+(TABLE|INDEX|VIEW|TRIGGER)\s+(?:IF\s+NOT\s+EXISTS\s+)?([A-Za-z_][A-Za-z0-9_]*)`)

// derivedObjects reads every object schema declares, in declaration order.
//
// It refuses an empty result rather than returning one. A schema that parsed to
// nothing would make the drop a no-op and the rebuild an append, and every
// assertion about "the derived half is empty afterwards" would pass over it for
// the wrong reason.
func derivedObjects(schema string) ([]derivedObject, error) {
	matches := createStatement.FindAllStringSubmatch(schema, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("the derived schema declares no object: nothing would be dropped and a rebuild would be an append")
	}
	objects := make([]derivedObject, 0, len(matches))
	for _, match := range matches {
		objects = append(objects, derivedObject{kind: match[1], name: match[2]})
	}
	return objects, nil
}

// rebuildDerived drops every object derived/library.sql declares, creates the
// schema again, and records the generation — in one transaction.
//
// One transaction because the three cannot be separated. A drop that committed
// without the create leaves a database with no derived half and a version
// saying it has one; a create that committed without the version leaves the
// next start rebuilding a schema that is already correct, which is a full
// rescan of every library for nothing. SQLite makes DDL transactional, so this
// costs nothing to insist on.
//
// **Foreign keys stay on across it.** Turning them off around the drop is the
// tempting shape and it is the one that hides the mistake this half is not
// allowed to make: with them off, a derived table referencing a precious one
// drops and recreates without complaint, and architecture §6's rule becomes a
// comment. PRAGMA foreign_keys is a no-op inside a transaction anyway, so
// disabling it here would have to be done outside the transaction that does the
// work — which is the shape a reviewer should refuse.
//
// The objects are dropped in reverse declaration order, children before
// parents. With foreign keys on, DROP TABLE performs an implicit DELETE, and
// dropping a parent first would make the rebuild depend on the cascades rather
// than on the order.
func rebuildDerived(ctx context.Context, db *sql.DB) error {
	objects, err := derivedObjects(derivedSchema)
	if err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("rebuilding the derived half: %w", err)
	}
	defer tx.Rollback()

	for i := len(objects) - 1; i >= 0; i-- {
		object := objects[i]
		// The name comes out of an embedded file this repository ships and is
		// matched against [A-Za-z_][A-Za-z0-9_]*, so there is nothing here for
		// a placeholder to carry — and DROP takes no parameters anyway.
		if _, err := tx.ExecContext(ctx, `DROP `+object.kind+` IF EXISTS `+object.name); err != nil {
			return fmt.Errorf("rebuilding the derived half: dropping %s %s: %w", object.kind, object.name, err)
		}
	}
	if _, err := tx.ExecContext(ctx, derivedSchema); err != nil {
		return fmt.Errorf("rebuilding the derived half: creating the schema: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE schema_version SET version = ? WHERE half = ?`, derivedGeneration, string(Derived)); err != nil {
		return fmt.Errorf("rebuilding the derived half: recording generation %d: %w", derivedGeneration, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("rebuilding the derived half: %w", err)
	}
	return nil
}

// ensureDerivedGeneration brings the derived half to the generation this build
// declares, and reports whether it had to rebuild it.
//
// **Any difference, in either direction.** A database recorded above this
// build's generation is a downgrade and one below it is an upgrade, and both
// answers are the same one: there is nothing in the derived half that a rescan
// cannot compute again, so the version it was written at tells a build nothing
// it needs. Branching only on "newer than known" — the shape the precious
// runner has to take, because a precious downgrade has no answer at all — would
// answer the downgrade and leave the upgrade to be discovered as queries
// against columns that have moved.
//
// This runs at start and is synchronous, because a store handed to a caller
// with a schema from another generation has exactly one correct use and nothing
// enforces it. **The scan that refills it is not**: a synchronous full scan of
// every library would turn a generation bump into a start that takes minutes
// and a readiness gate that stays shut for all of them, so the entry layer
// enqueues that after the server begins serving (003 plan §6.8).
func ensureDerivedGeneration(ctx context.Context, db *sql.DB) (bool, error) {
	recorded, err := schemaVersion(ctx, db, Derived)
	if err != nil {
		return false, err
	}
	if recorded == derivedGeneration {
		return false, nil
	}
	if err := rebuildDerived(ctx, db); err != nil {
		return false, err
	}
	return true, nil
}

// RebuildDerived drops every object of the derived schema and creates it again,
// leaving the precious half untouched.
//
// It is ports.ItemStore's method and ADR-0003's central claim as something a
// caller can perform: corruption in the derived half, a bug in a scanner or a
// schema bump are all answered by dropping and rescanning, with no
// user-visible loss. Afterwards no library has a scan_state row, which is the
// same state a library that has never been scanned is in.
func (s *Store) RebuildDerived(ctx context.Context) error {
	return rebuildDerived(ctx, s.writer)
}

// DerivedRebuilt reports whether this Open found the derived half at another
// generation and rebuilt it.
//
// It is on the store rather than logged from inside it for AppliedMigrations'
// reason — the store has no logger and should not grow one — and it is the
// signal the entry layer needs rather than a nicety: every library holds no
// items after a rebuild, and something has to know that a full scan of all of
// them is owed.
func (s *Store) DerivedRebuilt() bool { return s.derivedRebuilt }

// digestOf is what the schema/generation pairing compares. It is here rather
// than in the test so that the value in the failure message and the value in
// the constant are produced by one function.
func digestOf(schema string) string {
	sum := sha256.Sum256([]byte(schema))
	return hex.EncodeToString(sum[:])
}
