// Package app is the entry layer: this process's configuration, its logger, and
// the lifecycle of its HTTP server.
//
// It exists because architecture 3 allows cmd/atrium to be wiring and nothing
// else — "if something there is worth testing, it is in the wrong place" — and
// a server that starts, drains and stops is worth testing. cmd/atrium is
// therefore a main that calls Run, and everything Run does is here.
//
// architecture 2 puts flag parsing, configuration, wiring, start and stop in
// the entry layer, which may import everything. Nothing may import it.
package app

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"path/filepath"
	"strings"
	"time"
)

// Config is every process-level setting Atrium takes. architecture 9: there are
// few of them, they come from flags with an environment fallback, and they are
// not a feature.
type Config struct {
	// BindAddress is the host and port the HTTP server listens on.
	BindAddress string

	// DataDirectory is the one directory this installation owns, held as an
	// absolute path. architecture 5: it holds the store, the resized-image
	// cache, transcoding scratch space and the ignored-parameter tally.
	DataDirectory string

	// LogLevel is the lowest level the process logs.
	LogLevel slog.Level

	// ShutdownTimeout bounds the drain: how long a stop waits for requests
	// that are already in flight before it gives up on them. It is not a flag
	// because no operator has needed to change it; it is a field so that a
	// test does not have to wait out the default.
	ShutdownTimeout time.Duration
}

// The defaults. The bind address is the port the reference listens on, which is
// also the port this repository's own documents assume Atrium answers on:
// conformance.md's differential invocation passes --atrium http://localhost:8096
// and behaviours 2.3 measured the reference at :8096. A client configured for
// one and pointed at the other should not have to be told about a port.
const (
	DefaultBindAddress     = ":8096"
	DefaultLogLevel        = slog.LevelInfo
	DefaultShutdownTimeout = 15 * time.Second
)

// The environment variables each flag falls back to. The name of each is its
// flag's name, upper-cased with ATRIUM_ in front, so that neither has to be
// looked up once the other is known.
const (
	EnvBindAddress   = "ATRIUM_BIND_ADDRESS"
	EnvDataDirectory = "ATRIUM_DATA_DIR"
	EnvLogLevel      = "ATRIUM_LOG_LEVEL"
)

const (
	flagBindAddress   = "bind-address"
	flagDataDirectory = "data-dir"
	flagLogLevel      = "log-level"
)

// ParseConfig reads the configuration from args, which excludes the program
// name. A flag that was not given falls back to its environment variable, and
// then to its default; a flag that was given wins over both, because an
// argument is the more deliberate of the two.
//
// Usage and error text go to output. A request for help returns flag.ErrHelp
// with the usage already written, which is the standard library's contract and
// is left alone so that --help behaves the way every other Go program does.
func ParseConfig(args []string, getenv func(string) string, output io.Writer) (Config, error) {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}

	fs := flag.NewFlagSet("atrium", flag.ContinueOnError)
	fs.SetOutput(output)

	bindAddress := fs.String(flagBindAddress, DefaultBindAddress,
		"host and port to listen on; an empty host means every interface (env "+EnvBindAddress+")")
	dataDirectory := fs.String(flagDataDirectory, "",
		"directory this installation's data lives in; required (env "+EnvDataDirectory+")")
	logLevel := fs.String(flagLogLevel, levelName(DefaultLogLevel),
		"lowest level to log: "+strings.Join(levelNames(), ", ")+" (env "+EnvLogLevel+")")

	fs.Usage = func() {
		fmt.Fprint(output, "atrium — an independent implementation of the Jellyfin API.\n\n")
		fmt.Fprint(output, "Usage:\n  atrium --"+flagDataDirectory+" <directory> [flags]\n\nFlags:\n")
		fs.PrintDefaults()
		fmt.Fprint(output, "\nEvery flag falls back to the environment variable named in its\n"+
			"description, and a flag given on the command line wins over one.\n")
	}

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}
	if fs.NArg() > 0 {
		fs.Usage()
		return Config{}, fmt.Errorf("unexpected argument %q: atrium takes flags only", fs.Arg(0))
	}

	given := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { given[f.Name] = true })
	fallBackToEnv(bindAddress, given[flagBindAddress], getenv(EnvBindAddress))
	fallBackToEnv(dataDirectory, given[flagDataDirectory], getenv(EnvDataDirectory))
	fallBackToEnv(logLevel, given[flagLogLevel], getenv(EnvLogLevel))

	cfg := Config{ShutdownTimeout: DefaultShutdownTimeout}

	var err error
	if cfg.BindAddress, err = checkBindAddress(*bindAddress); err != nil {
		return Config{}, err
	}
	if cfg.DataDirectory, err = checkDataDirectory(*dataDirectory); err != nil {
		return Config{}, err
	}
	if cfg.LogLevel, err = parseLevel(*logLevel); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// fallBackToEnv overwrites a flag's value with the environment's when the flag
// was not given and the environment has something to say. An empty variable is
// read as unset: an empty bind address, data directory or level is not a
// setting anybody means.
func fallBackToEnv(value *string, given bool, fromEnv string) {
	if !given && fromEnv != "" {
		*value = fromEnv
	}
}

// checkBindAddress rejects an address net.Listen would reject later, so that
// the complaint names the flag rather than arriving from the network stack.
func checkBindAddress(address string) (string, error) {
	if _, _, err := net.SplitHostPort(address); err != nil {
		return "", fmt.Errorf("--%s %q is not a host and port (env %s): %w",
			flagBindAddress, address, EnvBindAddress, err)
	}
	return address, nil
}

// checkDataDirectory requires the directory and makes it absolute.
//
// There is deliberately no default. architecture 5 has the process own one
// directory holding the store, the image cache and the scratch space, and a
// default would put all three somewhere the operator did not name — under
// whatever the working directory happened to be, which for a service is
// wherever it was started from. The directory is not created here: nothing in
// this task writes to it.
func checkDataDirectory(directory string) (string, error) {
	if strings.TrimSpace(directory) == "" {
		return "", fmt.Errorf("a data directory is required: pass --%s or set %s",
			flagDataDirectory, EnvDataDirectory)
	}
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return "", fmt.Errorf("--%s %q: %w", flagDataDirectory, directory, err)
	}
	return absolute, nil
}

// levels is the whole vocabulary of --log-level, in the order the usage text
// lists them. slog's own parser is not used: it accepts offsets such as
// "INFO+2" and would make a typo mean something.
var levels = []struct {
	name  string
	level slog.Level
}{
	{"debug", slog.LevelDebug},
	{"info", slog.LevelInfo},
	{"warn", slog.LevelWarn},
	{"error", slog.LevelError},
}

func parseLevel(name string) (slog.Level, error) {
	folded := strings.ToLower(strings.TrimSpace(name))
	for _, l := range levels {
		if folded == l.name {
			return l.level, nil
		}
	}
	return 0, fmt.Errorf("--%s %q is not one of %s (env %s)",
		flagLogLevel, name, strings.Join(levelNames(), ", "), EnvLogLevel)
}

func levelName(level slog.Level) string {
	for _, l := range levels {
		if l.level == level {
			return l.name
		}
	}
	return level.String()
}

func levelNames() []string {
	names := make([]string, 0, len(levels))
	for _, l := range levels {
		names = append(names, l.name)
	}
	return names
}
