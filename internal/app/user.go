package app

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/store/sqlite"
	"github.com/vdatanet/atrium-go/internal/users"
)

// UserCommand is the first argument that selects the account subcommands.
//
// It is a constant rather than a literal in cmd/atrium because the dispatch
// there is one branch and nothing else (002 plan 3), and a word spelled in two
// places is a word that can be spelled two ways.
const UserCommand = "user"

// The three subcommands of 002 plan 6.9.
const (
	userAdd         = "add"
	userList        = "list"
	userSetPassword = "set-password"
)

// Why accounts are managed by a subcommand of this binary and not over HTTP:
// 002 spec 2 puts creating, editing and deleting users out of scope for v1 and
// says accounts are managed "through configuration", and 002 plan 6.9 reads
// that as forbidding an admin API rather than as naming a mechanism. The
// mechanism is here, and four things decided it — the one that decided it most
// is that conformance/ may import nothing of ours, so without a black box that
// makes an account, every criterion in spec 5 would be proven one layer in.
//
// # The password is read from standard input and there is no flag for it
//
// An argument vector is readable by every process on the host — ps on macOS,
// /proc/<pid>/cmdline on Linux — so a --password flag would put the one value
// ADR-0006 works to keep ephemeral into a place that outlives the command, and
// into the operator's shell history besides. That absence is asserted over the
// flag set itself rather than by reading this comment, because a comment is not
// a check.

// RunUser runs `atrium user <subcommand>`.
//
// It is here and not in cmd/atrium because architecture 3 allows cmd to hold no
// branch a test would want to reach, and provisioning is entirely branches
// worth reaching. cmd/atrium dispatches on the first argument; everything below
// is testable from a test binary (002 plan 3, which is 001 T1's amendment
// applied a second time).
//
// stdin carries the password, stdout carries anything an operator asked to see,
// and stderr carries the usage text and nothing else. The three are parameters
// so that a test provides all three.
func RunUser(ctx context.Context, args []string, getenv func(string) string,
	stdin io.Reader, stdout, stderr io.Writer) error {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	if len(args) == 0 {
		writeUserUsage(stderr)
		return fmt.Errorf("%s: a subcommand is required", UserCommand)
	}

	switch args[0] {
	case userAdd:
		return runUserAdd(ctx, args[1:], getenv, stdin, stdout, stderr)
	case userList:
		return runUserList(ctx, args[1:], getenv, stdout, stderr)
	case userSetPassword:
		return runUserSetPassword(ctx, args[1:], getenv, stdin, stdout, stderr)
	default:
		writeUserUsage(stderr)
		return fmt.Errorf("%s: %q is not a subcommand", UserCommand, args[0])
	}
}

func writeUserUsage(output io.Writer) {
	fmt.Fprint(output, "atrium "+UserCommand+" — manage this installation's accounts.\n\n"+
		"Usage:\n"+
		"  atrium "+UserCommand+" "+userAdd+"          --"+flagDataDirectory+" <directory> --"+flagName+" <name> [flags]\n"+
		"  atrium "+UserCommand+" "+userList+"         --"+flagDataDirectory+" <directory>\n"+
		"  atrium "+UserCommand+" "+userSetPassword+" --"+flagDataDirectory+" <directory> --"+flagName+" <name>\n\n"+
		"The password is read from standard input. There is no flag for it: an\n"+
		"argument vector is readable by every process on the host.\n")
}

// The flags the subcommands take. Every one of them is a name spelled once.
const (
	flagName                       = "name"
	flagAdministrator              = "administrator"
	flagDisabled                   = "disabled"
	flagHidden                     = "hidden"
	flagNoPassword                 = "no-password"
	flagMaxActiveSessions          = "max-active-sessions"
	flagLoginAttemptsBeforeLockout = "login-attempts-before-lockout"
	flagEnableMediaPlayback        = "enable-media-playback"
)

// addOptions is everything `user add` was told.
//
// The policy is a whole users.Policy rather than a handful of booleans, and it
// starts as DefaultPolicy() — so every flag's default below is literally the
// reference's own default for that property, and `atrium user add --name x`
// produces exactly the policy the reference gives a fresh account.
type addOptions struct {
	dataDirectory string
	name          string
	noPassword    bool
	policy        users.Policy
}

// newUserAddFlags builds `user add`'s flag set.
//
// It is a function of its own, returning the set unparsed, because 002 T7's
// definition of done asserts over the *flags themselves* that none of them
// takes a password. A test that read this file for the absence of a string
// would be a test of the source and not of the program.
func newUserAddFlags(output io.Writer) (*flag.FlagSet, *addOptions) {
	options := &addOptions{policy: users.DefaultPolicy()}

	fs := flag.NewFlagSet("atrium "+UserCommand+" "+userAdd, flag.ContinueOnError)
	fs.SetOutput(output)

	fs.StringVar(&options.dataDirectory, flagDataDirectory, "",
		"directory this installation's data lives in; required (env "+EnvDataDirectory+")")
	fs.StringVar(&options.name, flagName, "",
		"the account's name, as clients will display it; required")
	fs.BoolVar(&options.noPassword, flagNoPassword, false,
		"create the account without a password instead of reading one from standard input")

	fs.BoolVar(&options.policy.IsAdministrator, flagAdministrator, options.policy.IsAdministrator,
		"grant the account administrator rights")
	fs.BoolVar(&options.policy.IsDisabled, flagDisabled, options.policy.IsDisabled,
		"create the account disabled, so authenticating as it answers 403")
	// The default is *true*, and that is the reference's and not a slip: a
	// freshly created account is granted PermissionKind.IsHidden true
	// [source: Jellyfin.Data/UserEntityExtensions.cs:173 @ v10.11.11], so a
	// new account does not appear on a login screen until somebody unhides it.
	// --hidden=false is therefore the interesting spelling, which is the
	// opposite of what the flag's name suggests, and is why it is said here.
	fs.BoolVar(&options.policy.IsHidden, flagHidden, options.policy.IsHidden,
		"keep the account off login screens by excluding it from /Users/Public")
	fs.BoolVar(&options.policy.EnableMediaPlayback, flagEnableMediaPlayback, options.policy.EnableMediaPlayback,
		"let the account play media")

	fs.IntVar(&options.policy.MaxActiveSessions, flagMaxActiveSessions, options.policy.MaxActiveSessions,
		"cap on concurrent sessions; 0 means no cap")
	fs.IntVar(&options.policy.LoginAttemptsBeforeLockout, flagLoginAttemptsBeforeLockout,
		options.policy.LoginAttemptsBeforeLockout,
		"failed attempts before the account is locked out; -1 never locks and 0 means three")

	fs.Usage = func() {
		fmt.Fprint(output, "atrium "+UserCommand+" "+userAdd+" — create an account.\n\n"+
			"The password is read from standard input; pass --"+flagNoPassword+
			" for an account without one.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	return fs, options
}

// runUserAdd creates one account, and completes setup if this is the first.
func runUserAdd(ctx context.Context, args []string, getenv func(string) string,
	stdin io.Reader, stdout, stderr io.Writer) error {
	fs, options := newUserAddFlags(stderr)
	if err := parseSubcommand(fs, args); err != nil {
		return err
	}

	dataDirectory, err := resolveDataDirectory(options.dataDirectory, getenv)
	if err != nil {
		return err
	}
	name, err := checkName(options.name)
	if err != nil {
		return err
	}

	// Before the data directory is created and before the store is opened, so
	// that a mistyped invocation does not leave an installation behind.
	password := users.NewPlaintext("")
	if !options.noPassword {
		if password, err = readPassword(stdin); err != nil {
			return err
		}
	}

	store, err := openStoreAt(ctx, dataDirectory)
	if err != nil {
		return err
	}
	defer store.Close()

	// Looked up first for the message. The unique index on username_folded is
	// what actually forbids the second row (002 plan 4) — this is the
	// difference between "an account named Ada already exists" and a
	// constraint violation, and it is not the check.
	folded := users.Fold(name)
	if _, found, err := store.UserByFoldedName(ctx, folded); err != nil {
		return err
	} else if found {
		return fmt.Errorf("%s %s: an account whose name folds to %q already exists",
			UserCommand, userAdd, folded)
	}

	user, err := newAccount(name, folded, options.policy)
	if err != nil {
		return err
	}
	if err := store.CreateUser(ctx, user); err != nil {
		return err
	}

	if !options.noPassword {
		record, err := users.Derive(password)
		if err != nil {
			return err
		}
		if err := store.ReplaceCredential(ctx, user.ID, record, SystemClock().Now()); err != nil {
			return err
		}
	}

	if err := completeSetup(ctx, store, SystemClock()); err != nil {
		return err
	}

	fmt.Fprintf(stdout, "%s\t%s\n", user.ID, user.Username)
	return nil
}

// newAccount builds the record CreateUser writes.
//
// The identifier is derived from the folded name and never invented here
// (Principle VII, 002 plan 6.9), the configuration is the reference's default,
// and the two dates are nil: a new account has never logged in and has never
// been seen, and NULL is what makes LastLoginDate absent rather than the
// minimum date (spec 3.5).
func newAccount(name, folded string, policy users.Policy) (ports.User, error) {
	policyDocument, err := policy.Document()
	if err != nil {
		return ports.User{}, err
	}
	configurationDocument, err := users.DefaultConfiguration().Document()
	if err != nil {
		return ports.User{}, err
	}
	return ports.User{
		ID:                       users.DeriveID(name),
		Username:                 name,
		UsernameFolded:           folded,
		PolicyDocument:           policyDocument,
		ConfigurationDocument:    configurationDocument,
		InvalidLoginAttemptCount: 0,
		LastLoginAt:              nil,
		LastActivityAt:           nil,
	}, nil
}

// completeSetup records that initial configuration finished, if it has not been
// recorded already.
//
// This is 002 plan 6.8: nothing else in v1 completes setup, the reference's own
// operation for it is not on this surface and never will be under Principle VI,
// and until something completes it StartupWizardCompleted is false for ever and
// GET /System/Info admits every request for ever.
//
// **The idempotence is here, at the caller, and not in the store.** The store
// deliberately does not refuse a second MarkSetupComplete — "whether setup may
// be completed twice is a question about the wizard, which 002 owns" — so this
// reads the installation first and writes only when there is nothing recorded.
// The instant means *when setup was first completed*, and a second write would
// be a lie about it. That the second `user add` leaves the recorded instant
// alone is asserted by reading the column back, because a check of the boolean
// alone passes on a build that rewrites the instant on every call.
func completeSetup(ctx context.Context, store ports.InstallationStore, clock ports.Clock) error {
	installation, err := store.Installation(ctx)
	if err != nil {
		return err
	}
	if installation.SetupCompleted {
		return nil
	}
	return store.MarkSetupComplete(ctx, clock.Now())
}

// runUserList prints what accounts this installation holds.
func runUserList(ctx context.Context, args []string, getenv func(string) string,
	stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("atrium "+UserCommand+" "+userList, flag.ContinueOnError)
	fs.SetOutput(stderr)
	dataDirectory := fs.String(flagDataDirectory, "",
		"directory this installation's data lives in; required (env "+EnvDataDirectory+")")
	if err := parseSubcommand(fs, args); err != nil {
		return err
	}

	directory, err := resolveDataDirectory(*dataDirectory, getenv)
	if err != nil {
		return err
	}
	store, err := openStoreAt(ctx, directory)
	if err != nil {
		return err
	}
	defer store.Close()

	accounts, err := store.Users(ctx)
	if err != nil {
		return err
	}

	// Tab-separated with a header, which is what an operator can read and what
	// a shell can cut. The order is the store's, which is stated and stable
	// (username_folded, then id).
	table := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(table, "ID\tNAME\tADMINISTRATOR\tDISABLED\tHIDDEN\tPASSWORD")
	for _, account := range accounts {
		policy, err := users.PolicyOf(account)
		if err != nil {
			return err
		}
		_, hasPassword, err := store.Credential(ctx, account.ID)
		if err != nil {
			return err
		}
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\n",
			account.ID, account.Username,
			yesNo(policy.IsAdministrator), yesNo(policy.IsDisabled),
			yesNo(policy.IsHidden), yesNo(hasPassword))
	}
	return table.Flush()
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

// runUserSetPassword replaces one account's password record.
func runUserSetPassword(ctx context.Context, args []string, getenv func(string) string,
	stdin io.Reader, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("atrium "+UserCommand+" "+userSetPassword, flag.ContinueOnError)
	fs.SetOutput(stderr)
	dataDirectory := fs.String(flagDataDirectory, "",
		"directory this installation's data lives in; required (env "+EnvDataDirectory+")")
	name := fs.String(flagName, "", "the account to give a new password to; required")
	if err := parseSubcommand(fs, args); err != nil {
		return err
	}

	directory, err := resolveDataDirectory(*dataDirectory, getenv)
	if err != nil {
		return err
	}
	wanted, err := checkName(*name)
	if err != nil {
		return err
	}
	password, err := readPassword(stdin)
	if err != nil {
		return err
	}

	store, err := openStoreAt(ctx, directory)
	if err != nil {
		return err
	}
	defer store.Close()

	account, found, err := store.UserByFoldedName(ctx, users.Fold(wanted))
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("%s %s: there is no account named %q", UserCommand, userSetPassword, wanted)
	}

	record, err := users.Derive(password)
	if err != nil {
		return err
	}
	if err := store.ReplaceCredential(ctx, account.ID, record, SystemClock().Now()); err != nil {
		return err
	}

	fmt.Fprintf(stdout, "%s\t%s\n", account.ID, account.Username)
	return nil
}

// parseSubcommand parses a subcommand's flags and refuses a positional
// argument, which is 001's rule for the server's own flags applied here: a word
// that is not a flag is a mistyped flag or a misplaced value, and either is
// worth a message rather than being ignored.
func parseSubcommand(fs *flag.FlagSet, args []string) error {
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		fs.Usage()
		return fmt.Errorf("%s: unexpected argument %q: this subcommand takes flags only",
			fs.Name(), fs.Arg(0))
	}
	return nil
}

// resolveDataDirectory applies the same rule ParseConfig applies: the flag
// beats the environment, there is no default, and the answer is absolute.
//
// It is the same rule and the same environment variable deliberately. An
// operator who sets ATRIUM_DATA_DIR to run the server and then has to pass
// --data-dir to create an account is an operator who will one day create an
// account in the wrong installation.
func resolveDataDirectory(given string, getenv func(string) string) (string, error) {
	fallBackToEnv(&given, given != "", getenv(EnvDataDirectory))
	return checkDataDirectory(given)
}

// checkName refuses an empty account name.
//
// It refuses only what makes the rest impossible. What else the reference
// forbids in a username is measured nowhere in this repository — ⚠️ UNVERIFIED
// — and inventing a rule here would be a restriction no client could have
// predicted and this project could not have cited.
func checkName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", fmt.Errorf("%s: an account name is required: pass --%s", UserCommand, flagName)
	}
	return trimmed, nil
}

// readPassword reads the password from standard input.
//
// One trailing line ending is removed and nothing else, because that is what a
// terminal and a pipe both add and neither means: `printf 'secret' | atrium`
// and an operator typing `secret` and pressing return have to produce the same
// record. Trimming *every* trailing newline would silently accept two different
// inputs as one password.
//
// The value becomes a users.Plaintext immediately and is never held as a
// string: that type's String, GoString and slog.LogValue all answer a
// redaction, so a password cannot reach a log record through an error message
// somebody adds later (ADR-0006).
func readPassword(stdin io.Reader) (users.Plaintext, error) {
	raw, err := io.ReadAll(stdin)
	if err != nil {
		return users.Plaintext{}, fmt.Errorf("%s: reading the password from standard input: %w",
			UserCommand, err)
	}
	line := strings.TrimSuffix(strings.TrimSuffix(string(raw), "\n"), "\r")
	password := users.NewPlaintext(line)
	if password.IsEmpty() {
		return users.Plaintext{}, fmt.Errorf(
			"%s: no password on standard input: pipe one in, or pass --%s for an account without one",
			UserCommand, flagNoPassword)
	}
	return password, nil
}

// openStoreAt creates the data directory if it is not there and opens the
// store, applying every pending migration — the same two steps, in the same
// order and through the same functions, that a start performs.
//
// A provisioning command that opened the store a different way would be a
// second definition of what an installation is, and the first divergence
// between the two would be discovered by an operator.
func openStoreAt(ctx context.Context, dataDirectory string) (*sqlite.Store, error) {
	if err := EnsureDataDirectory(dataDirectory); err != nil {
		return nil, err
	}
	// context.WithoutCancel for the reason Run gives: a start is not something
	// this process abandons half-way, and a migration interrupted by a signal
	// is worse than one that finishes.
	return sqlite.Open(context.WithoutCancel(ctx), dataDirectory)
}
