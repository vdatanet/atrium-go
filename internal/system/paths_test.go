package system_test

import (
	"path/filepath"
	"testing"

	"github.com/vdatanet/atrium-go/internal/system"
)

// The layout, written down a second time on purpose.
//
// A test that recomputed these with filepath.Join would agree with the code
// whatever the code said, which is no assertion at all. Spelled out, changing
// where an installation keeps its metadata is a deliberate act with a failing
// test attached rather than a rename nobody reviews — and these seven strings
// are on the wire (spec 3.2), so they are contract and not housekeeping.
func TestPathsForDerivesTheLayoutFromTheDataDirectory(t *testing.T) {
	t.Parallel()

	got := system.PathsFor(filepath.FromSlash("/var/lib/atrium"))

	for _, row := range []struct{ what, got, want string }{
		{"ProgramData", got.ProgramData, "/var/lib/atrium"},
		{"Web", got.Web, "/var/lib/atrium/web"},
		{"ItemsByName", got.ItemsByName, "/var/lib/atrium/metadata"},
		{"Cache", got.Cache, "/var/lib/atrium/cache"},
		{"Log", got.Log, "/var/lib/atrium/log"},
		{"InternalMetadata", got.InternalMetadata, "/var/lib/atrium/metadata"},
		{"TranscodingTemp", got.TranscodingTemp, "/var/lib/atrium/cache/transcodes"},
	} {
		if want := filepath.FromSlash(row.want); row.got != want {
			t.Errorf("%s: got %q, want %q", row.what, row.got, want)
		}
	}
}

// Two fields, one value. The reference fills both ItemsByNamePath and
// InternalMetadataPath from InternalMetadataPath
// [source: Emby.Server.Implementations/SystemManager.cs:71-72 @ v10.11.11], so
// a response in which they differ is a response the reference never sends.
func TestItemsByNameIsTheInternalMetadataDirectory(t *testing.T) {
	t.Parallel()

	paths := system.PathsFor(filepath.FromSlash("/srv/atrium"))
	if paths.ItemsByName != paths.InternalMetadata {
		t.Errorf("ItemsByName %q and InternalMetadata %q are two values, and the reference sends one",
			paths.ItemsByName, paths.InternalMetadata)
	}
}

// The data directory is reported exactly as it was given, not resolved.
//
// What the response says, what --data-dir said and what the "starting" log line
// printed are then the same string, which is worth more to somebody reading all
// three than an absolute path they would have to match up by hand.
func TestPathsForDoesNotResolveTheDataDirectory(t *testing.T) {
	t.Parallel()

	relative := filepath.FromSlash("./data")
	if got := system.PathsFor(relative).ProgramData; got != relative {
		t.Errorf("ProgramData: got %q, want the configured %q back unchanged", got, relative)
	}
}
