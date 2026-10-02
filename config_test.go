package mcpee

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func writeTestConfig(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "c.yaml")
	if err := os.WriteFile(p, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadConfigDefaults(t *testing.T) {
	p := writeTestConfig(t, "backends:\n  - name: test\n    command: test-backend\n")

	c, err := LoadConfig(p)
	if err != nil {
		t.Fatal(err)
	}

	if c.Listen != "127.0.0.1:8080" {
		t.Fatalf("listen %q", c.Listen)
	}
	if len(c.Backends) != 1 {
		t.Fatalf("backends = %d, want 1", len(c.Backends))
	}

	b := c.Backends[0]
	wantCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if b.Cwd != wantCwd {
		t.Fatalf("cwd = %q, want %q", b.Cwd, wantCwd)
	}
	if !b.InheritEnv {
		t.Fatal("inherit_env = false, want default true")
	}
	if b.MaxFrameSize != 0 {
		t.Fatalf("max_frame_size = %d, want 0", b.MaxFrameSize)
	}
	if got := b.EffectiveMaxFrameBytes(); got != mcp.DefaultMaxLineLength {
		t.Fatalf("effective max frame bytes = %d, want %d", got, mcp.DefaultMaxLineLength)
	}
}

func TestLoadConfigBackendOptions(t *testing.T) {
	cwd := t.TempDir()
	yamlCwd := strings.ReplaceAll(cwd, "'", "''")
	p := writeTestConfig(t, fmt.Sprintf(`backends:
  - name: test
    command: test-backend
    cwd: '%s'
    inherit_env: false
    max_frame_size: 32MiB
    env:
      FOO: bar
`, yamlCwd))

	c, err := LoadConfig(p)
	if err != nil {
		t.Fatal(err)
	}
	b := c.Backends[0]

	wantCwd, err := filepath.Abs(cwd)
	if err != nil {
		t.Fatal(err)
	}
	if b.Cwd != wantCwd {
		t.Fatalf("cwd = %q, want %q", b.Cwd, wantCwd)
	}
	if b.InheritEnv {
		t.Fatal("inherit_env = true, want false")
	}
	if b.MaxFrameSize != 32*1024*1024 {
		t.Fatalf("max_frame_size = %d, want %d", b.MaxFrameSize, 32*1024*1024)
	}
	if b.Env["FOO"] != "bar" {
		t.Fatalf("env FOO = %q, want bar", b.Env["FOO"])
	}
}

func TestLoadConfigFrameSizes(t *testing.T) {
	tests := []struct {
		value string
		want  int
	}{
		{"33554432", 33554432},
		{"32MB", 32000000},
		{"32MiB", 33554432},
		{"-3KiB", -3072},
		{"0", 0},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			p := writeTestConfig(t, "backends:\n  - name: test\n    command: test-backend\n    max_frame_size: "+tt.value+"\n")
			c, err := LoadConfig(p)
			if err != nil {
				t.Fatal(err)
			}
			if got := c.Backends[0].MaxFrameSize; got != tt.want {
				t.Fatalf("max_frame_size = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestLoadConfigRejectsFrameSizeOverflow(t *testing.T) {
	p := writeTestConfig(t, "backends:\n  - name: test\n    command: test-backend\n    max_frame_size: 9223372036854775808B\n")
	if _, err := LoadConfig(p); err == nil {
		t.Fatal("expected max_frame_size overflow to fail")
	}
}

func TestEffectiveMaxFrameBytesPreservesConfiguredValue(t *testing.T) {
	cfg := BackendConfig{MaxFrameSize: -3072}
	if got := cfg.EffectiveMaxFrameBytes(); got != -3072 {
		t.Fatalf("effective max frame bytes = %d, want -3072", got)
	}
}
