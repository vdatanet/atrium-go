package surface

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// repositoryRoot is where this package sits, so that the test can name the
// canonical document from the package directory the test runs in.
const repositoryRoot = "../.."

// TestTheEmbeddedCopyIsTheDocument is the whole cost of embedding a copy, paid
// in one test.
//
// docs/compatibility/surface.yaml is the canonical half of a paired artefact
// (docs/README.md) and stays where it is. go:embed cannot reach outside its own
// package directory, and ADR-0002 wants one static binary that finds its route
// table wherever it was started from — so the file is copied here and the copy
// is asserted to be the document, byte for byte. There is no direction in which
// the two may differ, so a comparison rather than a merge is the right check.
//
// This test is internal because it compares the embedded bytes themselves and
// not something parsed from them: a comparison of two loaded tables would pass
// on a copy that differed in a comment, and a comment in this document is where
// its provenance is written.
func TestTheEmbeddedCopyIsTheDocument(t *testing.T) {
	canonical, err := os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash(EmbeddedFile)))
	if err != nil {
		t.Fatalf("reading %s: %v", EmbeddedFile, err)
	}
	if !bytes.Equal(document, canonical) {
		t.Errorf("the embedded internal/surface/surface.yaml is not %s.\n"+
			"Fix it with:\n\n    cp %s internal/surface/surface.yaml\n",
			EmbeddedFile, EmbeddedFile)
	}
}

// TestTheEmbeddedDocumentLoads makes V1's panic a failure of the build rather
// than of a run: the embedded table is parsed at every `go test`, so a document
// the loader refuses is caught here and never in a server that has already
// started answering.
func TestTheEmbeddedDocumentLoads(t *testing.T) {
	if _, err := Load(document); err != nil {
		t.Fatalf("the embedded %s does not load: %v", EmbeddedFile, err)
	}
	if V1().Len() == 0 {
		t.Error("V1() is empty")
	}
}
