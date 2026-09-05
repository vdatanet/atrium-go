package app

// The scans nobody asked for, asserted through [Run] over a real installation.
//
// # Why these run the server rather than the function beside them
//
// 001's closing audit found twice that *a criterion written about a request is
// not met by a test about the mechanism that serves it*, and 003 T13 found the
// same shape a third time at the seam between a scanner and a store. A test of
// `startScheduledScans` would pass on a build where [Run] never calls it, and on
// one that passes it a `ScanInterval` nothing parsed — which is the whole of
// what this change adds. So the assertions below start the server the way the
// binary starts it and read the **store** afterwards.

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/vdatanet/atrium-go/internal/library"
	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/store/sqlite"

	_ "modernc.org/sqlite"
)

// --- The setting --------------------------------------------------------------

// --scan-interval is a process setting of architecture §9's kind: a flag, an
// environment fallback, and a default that is not a feature.
//
// `0` is a **setting** rather than an absence — an operator who scans from a
// cron entry means it — and a negative duration is neither that nor a schedule,
// so it is refused where it was typed.
func TestTheScanIntervalIsASettingAndANegativeOneIsRefused(t *testing.T) {
	t.Parallel()

	data := t.TempDir()

	for _, testCase := range []struct {
		name    string
		args    []string
		env     map[string]string
		want    time.Duration
		refused bool
	}{
		{name: "the default is the reference's own", args: nil, want: DefaultScanInterval},
		{name: "a flag", args: []string{"--" + flagScanInterval, "90m"}, want: 90 * time.Minute},
		{name: "zero never rescans", args: []string{"--" + flagScanInterval, "0"}, want: 0},
		{
			name: "the environment, when the flag was not given",
			env:  map[string]string{EnvScanInterval: "45s"},
			want: 45 * time.Second,
		},
		{
			name: "and the flag wins over it",
			args: []string{"--" + flagScanInterval, "1h"},
			env:  map[string]string{EnvScanInterval: "45s"},
			want: time.Hour,
		},
		{name: "a negative duration", args: []string{"--" + flagScanInterval, "-1h"}, refused: true},
		{name: "a number with no unit", args: []string{"--" + flagScanInterval, "12"}, refused: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			args := append([]string{"--" + flagDataDirectory, data}, testCase.args...)
			cfg, err := ParseConfig(args, func(name string) string { return testCase.env[name] },
				&strings.Builder{})

			if testCase.refused {
				if err == nil {
					t.Fatalf("%v was accepted as %s", testCase.args, cfg.ScanInterval)
				}
				if !strings.Contains(err.Error(), flagScanInterval) {
					t.Errorf("the refusal does not name the flag: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseConfig(%v): %v", args, err)
			}
			if cfg.ScanInterval != testCase.want {
				t.Errorf("ScanInterval = %s, want %s", cfg.ScanInterval, testCase.want)
			}
		})
	}
}

// --- The schedule -------------------------------------------------------------

// **A running server rescans its libraries with nobody asking it to**, which is
// what spec §2 means by *"v1 rescans on demand and on a schedule"* and which
// nothing in this project did until this change.
//
// The control is the whole test. A server that never scanned on its own would
// pass an assertion of the form *"the items are there eventually"* if anything
// else in the installation had scanned — so two installations are started from
// the same tree at the same moment, one with a schedule and one with
// `--scan-interval 0`, and the scheduled one having finished is what times the
// assertion that the unscheduled one is still empty.
func TestAServerWithAScheduleRescansAndOneWithoutDoesNot(t *testing.T) {
	t.Parallel()

	films := []string{"The Matrix (1999)", "A Bridge Too Far (1977)", "Wall-E (2008)"}
	trees := t.TempDir()

	scheduled := t.TempDir()
	scheduledRoot := aTreeOfFilms(t, trees, "scheduled", films...)
	scheduledLibrary := addLibrary(t, scheduled, "Films", string(library.Movies), scheduledRoot)

	unscheduled := t.TempDir()
	unscheduledRoot := aTreeOfFilms(t, trees, "unscheduled", films...)
	unscheduledLibrary := addLibrary(t, unscheduled, "Films", string(library.Movies), unscheduledRoot)

	startRunning(t, scheduled, "--"+flagScanInterval, "50ms")
	startRunning(t, unscheduled, "--"+flagScanInterval, "0")

	waitForItems(t, scheduled, scheduledLibrary.ID, 4)

	if items := storedItems(t, unscheduled, unscheduledLibrary.ID); len(items) != 0 {
		t.Errorf("the server started with --%s 0 scanned anyway, and holds %d items: %s",
			flagScanInterval, len(items), describe(items))
	}
}

// **A start that had to rebuild the derived half rescans every library**, which
// is ADR-0003's *"a derived-version mismatch at startup is a rescan rather than
// an error"* finally kept whole.
//
// 003 T11 landed the rebuild and `store.DerivedRebuilt()`; until this change
// nothing acted on it, so an installation whose build had bumped the generation
// came up serving an empty library and stayed that way until somebody noticed.
//
// # The corpus is chosen so that doing nothing cannot pass
//
// A film is deleted from the tree between the first scan and the start. A build
// that ignored the rebuild leaves the store holding **no** items — the rebuild
// dropped them — and a build that rescanned incrementally over a store that
// remembers nothing would still have to derive the same identifiers for the two
// films that remain. So the assertion is three-sided: the item that has gone is
// absent, the two that remain are back under the identifiers they had, and the
// schedule is switched off so that nothing but the owed rescan could have done
// it.
func TestAStartThatRebuiltTheDerivedHalfRescansEveryLibrary(t *testing.T) {
	t.Parallel()

	data := t.TempDir()
	root := aTreeOfFilms(t, t.TempDir(), "films",
		"The Matrix (1999)", "A Bridge Too Far (1977)", "Wall-E (2008)")
	declared := addLibrary(t, data, "Films", string(library.Movies), root)

	mustScan(t, data, "--"+flagName, "Films")
	before := storedItems(t, data, declared.ID)
	if len(before) != 4 {
		t.Fatalf("the first scan stored %d items, want 4: %s", len(before), describe(before))
	}
	departed := identifierAt(t, before, "Wall-E (2008).mkv")
	if err := os.Remove(filepath.Join(root, "Wall-E (2008).mkv")); err != nil {
		t.Fatalf("removing a film: %v", err)
	}
	remaining := []string{
		identifierAt(t, before, "The Matrix (1999).mkv"),
		identifierAt(t, before, "A Bridge Too Far (1977).mkv"),
	}

	aDatabaseWrittenByAnotherBuild(t, data)

	startRunning(t, data, "--"+flagScanInterval, "0")
	after := waitForItems(t, data, declared.ID, 3)

	identifiers := make([]string, 0, len(after))
	for _, item := range after {
		identifiers = append(identifiers, item.ID)
	}
	if slices.Contains(identifiers, departed) {
		t.Errorf("the item %s came back although its file is gone: the rescan a rebuild owes is "+
			"a full one over what is on the disk now", departed)
	}
	for _, id := range remaining {
		if !slices.Contains(identifiers, id) {
			t.Errorf("the item %s did not come back after the derived half was rebuilt: %s",
				id, describe(after))
		}
	}
}

// aDatabaseWrittenByAnotherBuild moves the recorded derived generation away from
// the one this build declares, which is the state a schema change leaves behind
// on every installation that had the previous build.
//
// It is raw SQL and not a method on purpose: no port offers a way to write a
// schema version, because nothing in the program should have one. What is
// simulated here is another *build*, and the only honest way to produce that
// from inside this one is to write the number a different build would have left.
func aDatabaseWrittenByAnotherBuild(t *testing.T, dataDirectory string) {
	t.Helper()

	db, err := sql.Open("sqlite", "file:"+filepath.Join(dataDirectory, sqlite.DatabaseFile))
	if err != nil {
		t.Fatalf("opening the database directly: %v", err)
	}
	defer db.Close()

	result, err := db.Exec(`UPDATE schema_version SET version = version + 1 WHERE half = ?`,
		string(sqlite.Derived))
	if err != nil {
		t.Fatalf("moving the derived generation: %v", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("moving the derived generation: %v", err)
	}
	if affected != 1 {
		t.Fatalf("moving the derived generation changed %d rows, want 1: the test would then "+
			"assert a rebuild that never happened", affected)
	}
}

// --- Running the server, and reading what it wrote ----------------------------

// startRunning starts the binary's own [Run] on a data directory and stops it
// when the test ends.
//
// It waits for the listening line before returning, which is 001's own way of
// knowing a start finished: the readiness gate opens on the line before it, and
// the scanning this file is about is started on the line after.
func startRunning(t *testing.T, dataDirectory string, args ...string) {
	t.Helper()

	log := &synchronisedBuffer{}
	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan error, 1)

	go func() {
		stopped <- Run(ctx, append([]string{
			"--" + flagDataDirectory, dataDirectory,
			"--" + flagBindAddress, "127.0.0.1:0",
		}, args...), noEnvironment, log)
	}()

	t.Cleanup(func() {
		cancel()
		select {
		case err := <-stopped:
			if err != nil {
				t.Errorf("Run returned %v, want nil\n%s", err, log.String())
			}
		case <-time.After(30 * time.Second):
			// A stop that never comes is this change's own risk: [Run] waits
			// for the scanning goroutine before it returns, so a scan that
			// ignored the cancelled context would hang here rather than fail
			// somewhere unrelated.
			t.Errorf("Run did not return within 30s of the stop\n%s", log.String())
		}
	})

	waitForListeningAddress(t, log)
}

// waitForItems polls the store until the library holds exactly count items.
//
// The store and not the log: a scanner that reported what it had not written is
// the seam 003 plan §8.1 names, and this file is about wiring.
func waitForItems(t *testing.T, dataDirectory, libraryID string, count int) []ports.ScannedItem {
	t.Helper()

	deadline := time.Now().Add(30 * time.Second)
	for {
		items := storedItems(t, dataDirectory, libraryID)
		if len(items) == count {
			return items
		}
		if time.Now().After(deadline) {
			t.Fatalf("the library holds %d items after 30s, want %d: %s",
				len(items), count, describe(items))
		}
		time.Sleep(20 * time.Millisecond)
	}
}
