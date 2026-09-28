package mcpee

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigDefaults(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.yaml")
	if err := os.WriteFile(p, []byte("backends:\n  - name: test\n    command: test-backend\n"), 0600); err != nil {
		t.Fatal(err)
	}

	c, err := LoadConfig(p)
	if err != nil {
		t.Fatal(err)
	}

	if c.Listen != "127.0.0.1:8080" {
		t.Fatalf("listen %q", c.Listen)
	}
}
