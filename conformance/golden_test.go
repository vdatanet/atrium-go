package conformance_test

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

// updateGolden rewrites the golden files from the responses this run received.
//
// architecture 8: goldens are reviewed, never blindly regenerated, and a diff
// in one is a contract change read like one. So this flag does two things and
// the second is the important one — it writes the file, and then it **fails the
// test**. No single run can both rewrite a golden and report green, which means
// the new bytes have to be looked at and the suite re-run without the flag
// before anything is called passing.
//
// That is the difference between an update flag and a "make the test pass"
// button. The button is what turns a golden into a record of whatever the code
// last did, which is a test that cannot fail.
var updateGolden = flag.Bool("update-golden", false,
	"rewrite the golden files from this run's responses, then fail so the diff is reviewed")

// goldenDirectory is where the recorded responses live.
const goldenDirectory = "testdata/golden"

// assertGolden compares one response body against its recorded bytes.
//
// The comparison is on bytes and nothing else. A golden compared after parsing
// would agree about a body whose property names had changed case, whose numbers
// had become strings, or whose keys had been reordered — and all three are
// differences a client sees (Principle VIII, conformance L3).
//
// The recorded file holds exactly what the wire carried, with no trailing
// newline, because internal/wire sends none. An editor that adds one to the
// file is a failing test rather than a silently forgiving one, which is the
// right way round for a file whose whole job is to be exact.
func assertGolden(t *testing.T, name string, got []byte) {
	t.Helper()

	path := filepath.Join(goldenDirectory, name)

	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("making the golden directory: %v", err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
		t.Fatalf("%s was rewritten from this run.\n"+
			"A golden is a record of what a client receives, so read the diff as a contract change, "+
			"then re-run without -update-golden.\n%s", path, got)
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v\n"+
			"If this response is new, create the file with -update-golden and review what it recorded.",
			path, err)
	}

	if bytes.Equal(got, want) {
		return
	}

	t.Errorf("the response does not match %s.\n got %s\nwant %s\n%s",
		path, got, want, firstDifference(got, want))
}

// firstDifference names the byte the two bodies first disagree on, because a
// two-hundred-byte JSON object printed twice is not a diff a reader can see.
func firstDifference(got, want []byte) string {
	limit := min(len(got), len(want))
	for i := range limit {
		if got[i] != want[i] {
			return "They first differ at byte " + itoa(i) + ": got " +
				quoteByte(got[i]) + ", want " + quoteByte(want[i]) + "."
		}
	}
	switch {
	case len(got) > len(want):
		return "They agree for " + itoa(limit) + " bytes and the response then carries " +
			itoa(len(got)-limit) + " more: " + string(got[limit:])
	case len(want) > len(got):
		return "They agree for " + itoa(limit) + " bytes and the response then stops; the golden carries " +
			itoa(len(want)-limit) + " more: " + string(want[limit:])
	default:
		return ""
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

func quoteByte(b byte) string {
	if b >= 0x20 && b < 0x7f {
		return "'" + string(rune(b)) + "'"
	}
	const hex = "0123456789abcdef"
	return "0x" + string([]byte{hex[b>>4], hex[b&0x0f]})
}
