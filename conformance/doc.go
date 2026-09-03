// Package conformance holds the checks that assert what this server puts on
// the wire, at the wire.
//
// # It imports nothing of ours, and that is the whole point
//
// architecture 3 puts L0, L1 and L2 here and forbids this directory from
// importing internal/. A test that can reach inside can assert on a value the
// wire never carried, and will keep passing on the day the wire stops carrying
// it — which is Principle VIII stated as a directory rule. Go does not enforce
// it (internal/ restricts imports across module paths, not within a module), so
// tools/check_conformance_imports does, over `go list -deps`, in CI.
//
// The rule has a consequence worth stating plainly, because it looks like an
// inconvenience and is actually the load-bearing part: this package cannot
// call NewServer, cannot build a pipeline and cannot construct a handler. It
// starts the **binary** — the same one an operator runs — on a loopback
// listener, and speaks HTTP to it. Everything these tests know about the server
// is something a client could have known.
//
// # Goldens
//
// architecture 8: goldens are reviewed, never blindly regenerated. The files
// under testdata/golden are the exact response bodies, byte for byte, with no
// trailing newline, because the wire has none. A diff in one of them is a
// change to what a client receives, and it is read in review like any other
// contract change.
//
// There is an update flag, and using it is deliberate rather than convenient:
// `go test ./conformance -update-golden` rewrites the files this run compared
// and then **fails**, so that no run both rewrites a golden and reports green.
// The diff has to be looked at and the suite re-run without the flag.
//
// # No test here contacts anything
//
// AGENTS.md 1.6: no test opens a network connection to anything outside this
// machine, and no CI job contacts or starts a Jellyfin. The servers these tests
// speak to are started by these tests, on 127.0.0.1, on a port the operating
// system chooses.
package conformance
