package app

import (
	"bytes"
	"errors"
	"flag"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseConfigTakesItsDefaults(t *testing.T) {
	cfg, err := ParseConfig([]string{"--data-dir", t.TempDir()}, noEnvironment, io.Discard)
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if cfg.BindAddress != DefaultBindAddress {
		t.Errorf("BindAddress = %q, want %q", cfg.BindAddress, DefaultBindAddress)
	}
	if cfg.LogLevel != DefaultLogLevel {
		t.Errorf("LogLevel = %v, want %v", cfg.LogLevel, DefaultLogLevel)
	}
	if cfg.ShutdownTimeout != DefaultShutdownTimeout {
		t.Errorf("ShutdownTimeout = %v, want %v", cfg.ShutdownTimeout, DefaultShutdownTimeout)
	}
}

// TestParseConfigFallsBackToTheEnvironment is the fallback architecture 9 asks
// for, and the precedence between the two halves.
func TestParseConfigFallsBackToTheEnvironment(t *testing.T) {
	environment := map[string]string{
		EnvBindAddress:   "127.0.0.1:9000",
		EnvDataDirectory: "/var/lib/from-the-environment",
		EnvLogLevel:      "debug",
	}
	getenv := func(name string) string { return environment[name] }

	cfg, err := ParseConfig(nil, getenv, io.Discard)
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if cfg.BindAddress != "127.0.0.1:9000" {
		t.Errorf("BindAddress = %q, want the environment's", cfg.BindAddress)
	}
	if cfg.DataDirectory != "/var/lib/from-the-environment" {
		t.Errorf("DataDirectory = %q, want the environment's", cfg.DataDirectory)
	}
	if cfg.LogLevel != slog.LevelDebug {
		t.Errorf("LogLevel = %v, want %v", cfg.LogLevel, slog.LevelDebug)
	}

	given, err := ParseConfig([]string{
		"--bind-address", "127.0.0.1:9100",
		"--data-dir", "/var/lib/from-the-flag",
		"--log-level", "warn",
	}, getenv, io.Discard)
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if given.BindAddress != "127.0.0.1:9100" {
		t.Errorf("BindAddress = %q, want the flag's", given.BindAddress)
	}
	if given.DataDirectory != "/var/lib/from-the-flag" {
		t.Errorf("DataDirectory = %q, want the flag's", given.DataDirectory)
	}
	if given.LogLevel != slog.LevelWarn {
		t.Errorf("LogLevel = %v, want the flag's", given.LogLevel)
	}
}

// TestParseConfigMakesTheDataDirectoryAbsolute keeps one installation path in
// the logs and, later, in the allowlist's installation-path class, rather than
// one that means something different to every process that reads it.
func TestParseConfigMakesTheDataDirectoryAbsolute(t *testing.T) {
	cfg, err := ParseConfig([]string{"--data-dir", "atrium-data"}, noEnvironment, io.Discard)
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if !filepath.IsAbs(cfg.DataDirectory) {
		t.Errorf("DataDirectory = %q, want an absolute path", cfg.DataDirectory)
	}
	if filepath.Base(cfg.DataDirectory) != "atrium-data" {
		t.Errorf("DataDirectory = %q, want it to still name the directory that was asked for", cfg.DataDirectory)
	}
}

func TestParseConfigRefusals(t *testing.T) {
	cases := []struct {
		name        string
		args        []string
		wantMention []string
	}{
		{
			name:        "no data directory anywhere",
			args:        nil,
			wantMention: []string{"--data-dir", EnvDataDirectory},
		},
		{
			name:        "a bind address with no port",
			args:        []string{"--data-dir", "/tmp/atrium", "--bind-address", "localhost"},
			wantMention: []string{"--bind-address", "localhost", EnvBindAddress},
		},
		{
			name:        "a level that is not one",
			args:        []string{"--data-dir", "/tmp/atrium", "--log-level", "verbose"},
			wantMention: []string{"--log-level", "verbose", "debug, info, warn, error"},
		},
		{
			name:        "an argument that is not a flag",
			args:        []string{"--data-dir", "/tmp/atrium", "serve"},
			wantMention: []string{"serve"},
		},
		{
			name:        "a flag nothing declares",
			args:        []string{"--data-dir", "/tmp/atrium", "--port", "8096"},
			wantMention: []string{"port"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var output bytes.Buffer
			_, err := ParseConfig(c.args, noEnvironment, &output)
			if err == nil {
				t.Fatal("ParseConfig accepted it")
			}
			said := err.Error() + output.String()
			for _, mention := range c.wantMention {
				if !strings.Contains(said, mention) {
					t.Errorf("the refusal does not mention %q; it said: %s", mention, said)
				}
			}
		})
	}
}

// TestParseConfigHelpNamesEveryFlagAndItsVariable is the "--help prints the
// flags" half of this task's check, at the only level that can assert on what
// it prints.
func TestParseConfigHelpNamesEveryFlagAndItsVariable(t *testing.T) {
	var output bytes.Buffer
	_, err := ParseConfig([]string{"--help"}, noEnvironment, &output)
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("ParseConfig returned %v, want %v", err, flag.ErrHelp)
	}

	printed := output.String()
	for _, want := range []string{
		"-bind-address", EnvBindAddress,
		"-data-dir", EnvDataDirectory,
		"-log-level", EnvLogLevel,
		"debug, info, warn, error",
		DefaultBindAddress,
	} {
		if !strings.Contains(printed, want) {
			t.Errorf("--help does not mention %q; it printed:\n%s", want, printed)
		}
	}
}

func noEnvironment(string) string { return "" }
