package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCLI(t *testing.T) {
	exe := filepath.Join(t.TempDir(), "mcpee")
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}

	build := exec.Command("go", "build", "-o", exe, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build mcpee: %v\n%s", err, out)
	}

	run := func(args ...string) (string, string, int) {
		t.Helper()
		cmd := exec.Command(exe, args...)
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		if err == nil {
			return stdout.String(), stderr.String(), 0
		}
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("run %v: %v", args, err)
		}
		return stdout.String(), stderr.String(), exitErr.ExitCode()
	}

	validConfig := filepath.Join(t.TempDir(), "valid.yaml")
	if err := os.WriteFile(validConfig, []byte("backends:\n  - name: test\n    command: definitely-not-a-real-backend\n"), 0600); err != nil {
		t.Fatal(err)
	}
	invalidConfig := filepath.Join(t.TempDir(), "invalid.yaml")
	if err := os.WriteFile(invalidConfig, []byte("backends: []\n"), 0600); err != nil {
		t.Fatal(err)
	}

	t.Run("config test aliases", func(t *testing.T) {
		for _, args := range [][]string{
			{"-t", "-c", validConfig},
			{"--test-config", "--config", validConfig},
		} {
			stdout, stderr, code := run(args...)
			if code != 0 {
				t.Fatalf("%v exit = %d, stderr = %q", args, code, stderr)
			}
			if stdout != "config OK\n" {
				t.Fatalf("%v stdout = %q", args, stdout)
			}
			if stderr != "" {
				t.Fatalf("%v stderr = %q", args, stderr)
			}
		}
	})

	t.Run("invalid config", func(t *testing.T) {
		_, stderr, code := run("-t", "-c", invalidConfig)
		if code != 1 {
			t.Fatalf("exit = %d, want 1", code)
		}
		if !strings.Contains(stderr, "load config failed") {
			t.Fatalf("stderr = %q", stderr)
		}
	})

	t.Run("version aliases", func(t *testing.T) {
		short, shortErr, shortCode := run("-v")
		long, longErr, longCode := run("--version")
		if shortCode != 0 || longCode != 0 {
			t.Fatalf("exit codes = %d, %d", shortCode, longCode)
		}
		if short != long || short != "mcpee wip\n" {
			t.Fatalf("version output = %q, %q", short, long)
		}
		if shortErr != "" || longErr != "" {
			t.Fatalf("version stderr = %q, %q", shortErr, longErr)
		}
	})

	t.Run("help aliases", func(t *testing.T) {
		_, short, shortCode := run("-h")
		_, long, longCode := run("--help")
		if shortCode != 0 || longCode != 0 {
			t.Fatalf("exit codes = %d, %d", shortCode, longCode)
		}
		if short != long {
			t.Fatalf("help differs:\n-h:\n%s\n--help:\n%s", short, long)
		}
		if !strings.Contains(short, "Usage: mcpee [options]") {
			t.Fatalf("help = %q", short)
		}
	})

	t.Run("positional argument", func(t *testing.T) {
		_, stderr, code := run("wat")
		if code != 2 {
			t.Fatalf("exit = %d, want 2", code)
		}
		if !strings.Contains(stderr, `unexpected argument "wat"`) {
			t.Fatalf("stderr = %q", stderr)
		}
	})
}
