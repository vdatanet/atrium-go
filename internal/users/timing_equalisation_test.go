package users_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/vdatanet/atrium-go/internal/users"
)

// This file is the one check ADR-0006's argument stands or falls on.
//
// The record says so itself, under "What is not verified, and is owed": *"the
// timing equalisation is specified here and asserted nowhere ... a rule of this
// kind that is not tested is a rule that regresses the first time somebody adds
// an early return"*, and it hands the test to this feature by name. Everything
// else in ADR-0006 — the parameters, the ceiling, the PHC record, the rehash —
// is asserted somewhere in this package already. The equalisation was not.
//
// **The ADR is not edited now that this exists.** AGENTS.md §4 makes an
// accepted ADR immutable: a wrong one is superseded, never edited, and this one
// is not wrong. It records a decision as taken, including what it owed at the
// time, and it already names 002 as where the debt is paid. The discharge is
// recorded in 002's task list and in plan §8.1, not by striking a line out of
// the record. If you came here to tidy that sentence, this paragraph is the
// answer.
//
// # Two halves, and neither is sufficient alone
//
// The mechanism half is deterministic and says *what the code does*: one
// derivation, at the current constants. It cannot see how long that derivation
// took, so on its own it would pass a build whose decoy was pinned to
// parameters ten times cheaper than the ones a real account's record carries.
//
// The wall-clock half is a measurement and says *what a stopwatch outside the
// process would see*. It is the only assertion in this repository that could
// notice an equalisation that is present in the code and absent in the timing,
// and it is necessarily noisier than everything around it.
//
// # Why the margin is a quarter of a derivation and not a tight one
//
// What this catches is a **missing** derivation, and a missing derivation is a
// whole derivation's gap — 52.4 ms on the machine ADR-0006 measured
// [measurement: golang.org/x/crypto v0.55.0, Go 1.27.0, 2026-09-03], and more
// on a shared CI runner. A margin tight enough to catch a 2 ms drift would turn
// every scheduling hiccup into a red build, and a flaky test is a test somebody
// eventually deletes — which would take ADR-0006's only evidence with it. A
// loose check that survives proves more than a tight one that gets removed.
// plan §9's risk table takes the same position.
//
// # What is *not* asserted here, and where it lives instead
//
// Neither half is in conformance/. At the wire both refusals carry the same
// derivation plus the network, so an HTTP-level timing test is this same
// assertion with more noise and eighteen times the pipeline. What conformance/
// proves instead is the half that is a wire fact — that the two refusals are
// byte-identical (AC-2) — and the two together are the disclosure ADR-0006
// describes: identical bytes, and identical time.

// timingSamples is how many authentications of each kind the wall-clock half
// times. Nine is ADR-0006's own sample size: every median in its table is over
// nine derivations.
const timingSamples = 9

// TestAnUnknownUsernameDerivesOnceWithTheCurrentConstants is the mechanism half
// of the equalisation.
//
// It is one test and not two because the claim is a conjunction, and each half
// of it is satisfied by a build the other rejects:
//
//   - **Exactly one derivation.** A path that skips it answers in microseconds
//     where a wrong password answers in 52 ms, which is ADR-0006 rule 1's whole
//     subject. A path that derives *twice* is as distinguishable, in the other
//     direction.
//   - **At the current constants.** ADR-0006 rule 2: a decoy pinned to old
//     parameters becomes its own oracle the moment the constants are raised —
//     the account that does not exist answers faster than one whose record has
//     been rehashed. The count cannot see this; the parameters can.
//
// The parameters are read off the decoy's own PHC record and compared against
// the constants, which makes this the assertion that fails on the day somebody
// raises the constants and leaves the decoy behind. It is deliberately a
// *relative* claim rather than a pinned number: the constants are allowed to
// move and the decoy must move with them.
//
// **What it does not prove.** Nothing here observes that the derivation the
// login path spent was against the decoy rather than against some other record
// at the same parameters; no Go test in this package can, and it would be a
// distinction without a difference for the disclosure ADR-0006 is closing,
// which is about cost and not about identity. The wall-clock half below is what
// holds the cost.
func TestAnUnknownUsernameDerivesOnceWithTheCurrentConstants(t *testing.T) {
	accounts := newFakeAccounts()
	login := users.NewLogin(accounts, fixedClock{})

	var err error
	spent := derivationsDuring(func() {
		_, err = login.Authenticate(context.Background(), "nobody at all", users.NewPlaintext(thePassword))
	})

	if !errors.Is(err, users.ErrCredentialsRefused) {
		t.Fatalf("authenticating a username matching no account returned %v, want ErrCredentialsRefused", err)
	}
	if spent != 1 {
		t.Errorf("a username matching no account spent %d derivations, want exactly 1 — "+
			"one is what makes this refusal cost what a wrong password costs (ADR-0006 rule 1)", spent)
	}

	// The second clause, and the reason this is not just T5's count again:
	// a build that derived once against a decoy written at weaker parameters
	// would satisfy the count above and hand back exactly the timing oracle
	// the decoy exists to close (ADR-0006 rule 2).
	assertParametersAreTheCurrentConstants(t, users.DecoyRecord(), "the decoy an unknown username is verified against")
}

// TestTheTwoRefusalsCannotBeToldApartWithAStopwatch is the wall-clock half.
//
// spec §3.3 answers an unknown username and a wrong password on an enabled
// account with the same status and the same 25 bytes. ADR-0006 exists so that
// they also take the same time: *"a hashing scheme that lets a stopwatch
// separate the two 401s would hand back the disclosure the identical bytes were
// there to withhold"*. This is the stopwatch.
//
// # How it is measured
//
// Nine of each, **interleaved**. Interleaving matters more than the count: a
// runner whose clock ramps up, a garbage collection, or a neighbour process
// starting halfway through would land entirely on the second group if the two
// were timed in blocks, and would show up as a difference that is not a
// difference. Alternating spreads any such drift across both medians.
//
// One authentication of each kind is run and discarded first. The first
// Argon2id derivation in a process pays for faulting in 64 MiB that no
// subsequent one pays for, and it would otherwise land in whichever group went
// first.
//
// The median is used rather than the mean because a single descheduled sample
// moves a mean of nine by a ninth of its own size and moves a median of nine
// not at all. That is the same reason ADR-0006's own table is medians.
//
// # What the margin is measured against
//
// A quarter of **one derivation as measured on the machine running this test**,
// and not a hard-coded 52.4 ms. A CI runner is several times slower than the
// machine ADR-0006 measured and a small ARM host slower still — the ADR says so
// itself, and holds the parameters open on exactly that ground — so a literal
// millisecond count here would be a number that is wrong everywhere but one
// desk. The smaller of the two medians is the estimate, which is the
// conservative choice: it is the tightest margin the two samples support.
//
// That estimate is not circular, because it fails in both directions. If the
// unknown-username path stopped deriving, its median would collapse to
// microseconds and take the margin with it, while the wrong-password median
// stayed at a full derivation — the gap would be a hundred times the margin
// rather than a quarter of it. The one build it could not catch is one where
// *neither* path derives, and the derivation count asserted below is what makes
// that impossible.
func TestTheTwoRefusalsCannotBeToldApartWithAStopwatch(t *testing.T) {
	accounts := newFakeAccounts()
	password := thePassword
	// LoginAttemptsBeforeLockout is -1 on DefaultPolicy, so the twenty failed
	// attempts below never lock this account. That is load-bearing rather than
	// convenient: a locked account is refused as *disabled*, with no derivation
	// at all, so a lockout partway through would silently turn the second group
	// into a measurement of nothing. The per-sample error assertion is what
	// makes that a failure rather than a quiet corruption.
	accounts.add(t, "u1", "Alice", users.DefaultPolicy(), &password)
	login := users.NewLogin(accounts, fixedClock{})

	ctx := context.Background()
	wrong := users.NewPlaintext("not the password")

	refuse := func(username string) time.Duration {
		started := time.Now()
		_, err := login.Authenticate(ctx, username, wrong)
		elapsed := time.Since(started)
		if !errors.Is(err, users.ErrCredentialsRefused) {
			t.Fatalf("authenticating %q with a wrong password returned %v, want ErrCredentialsRefused — "+
				"a refusal of another kind does not spend a derivation and would make these medians meaningless",
				username, err)
		}
		return elapsed
	}

	// The discarded pair. Its cost is the 64 MiB arena, not the algorithm.
	refuse("nobody at all")
	refuse("alice")

	unknownUsername := make([]time.Duration, 0, timingSamples)
	wrongPassword := make([]time.Duration, 0, timingSamples)
	spent := derivationsDuring(func() {
		for range timingSamples {
			unknownUsername = append(unknownUsername, refuse("nobody at all"))
			wrongPassword = append(wrongPassword, refuse("alice"))
		}
	})

	unknownMedian := median(unknownUsername)
	wrongMedian := median(wrongPassword)
	difference := unknownMedian - wrongMedian
	if difference < 0 {
		difference = -difference
	}
	margin := min(unknownMedian, wrongMedian) / 4

	t.Logf("unknown username: median %v over %d, samples %v", unknownMedian, timingSamples, unknownUsername)
	t.Logf("wrong password:   median %v over %d, samples %v", wrongMedian, timingSamples, wrongPassword)
	t.Logf("difference %v, margin %v (a quarter of one derivation as measured here)", difference, margin)

	if difference >= margin {
		t.Errorf("the two 401 refusals differ by %v, which is more than %v — a quarter of one "+
			"derivation as measured on this machine (unknown username %v, wrong password %v). "+
			"spec §3.3 makes these two answers byte-identical and ADR-0006 makes them cost the "+
			"same; a gap of a whole derivation means one of the two paths is no longer hashing, "+
			"and the disclosure the identical bytes withhold is back on the clock",
			difference, margin, unknownMedian, wrongMedian)
	}

	// The floor under the margin above, asserted last so that a build which
	// stopped deriving fails with the stopwatch's own numbers rather than with
	// a count that pre-empts them. Without it, a build where *neither* refusal
	// hashes anything would compare two microsecond medians and pass.
	if spent != 2*timingSamples {
		t.Errorf("%d authentications spent %d derivations, want %d — one each. "+
			"Two paths that both stopped hashing would agree with each other perfectly",
			2*timingSamples, spent, 2*timingSamples)
	}
}

// median answers the middle sample, which for an odd count is a value that was
// actually observed rather than an average of two that were not.
//
// It sorts a copy: the caller's slice is reported in the order it was measured
// when this test fails, and a sorted one would hide a drift across the run.
func median(samples []time.Duration) time.Duration {
	sorted := slices.Clone(samples)
	slices.Sort(sorted)
	return sorted[len(sorted)/2]
}
