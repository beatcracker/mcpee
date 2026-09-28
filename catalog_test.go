package mcpee

import (
	"context"
	"errors"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type fakeBackend struct {
	name   string
	tools  []*mcp.Tool
	called string
	args   any
	result *mcp.CallToolResult
	err    error
}

func (b *fakeBackend) Name() string         { return b.name }
func (b *fakeBackend) Catalog() []*mcp.Tool { return b.tools }
func (b *fakeBackend) Call(ctx context.Context, tool string, args any) (*mcp.CallToolResult, error) {
	b.called = tool
	b.args = args
	if b.err != nil {
		return nil, b.err
	}
	return b.result, nil
}
func (b *fakeBackend) Close() error { return nil }

func TestProjectCatalog(t *testing.T) {
	backendA := &fakeBackend{
		name: "backend_a",
		tools: []*mcp.Tool{
			{
				Name:        "tool_a",
				Description: "tool a",
				InputSchema: map[string]any{"type": "object"},
			},
		},
	}
	backendB := &fakeBackend{
		name: "backend_b",
		tools: []*mcp.Tool{
			{
				Name:        "tool_b",
				Description: "tool b",
				InputSchema: map[string]any{"type": "object"},
			},
			{
				Name:        "tool_c",
				Description: "tool c",
				InputSchema: map[string]any{"type": "object"},
			},
		},
	}
	c, err := ProjectCatalog([]Backend{backendA, backendB})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"backend_a__tool_a", "backend_b__tool_b", "backend_b__tool_c"}
	for i, n := range want {
		if got := c.Tools()[i].Name; got != n {
			t.Fatalf("tool %d: got %q want %q", i, got, n)
		}
	}
	r, ok := c.Lookup("backend_b__tool_b")
	if !ok || r.Backend != backendB || r.Tool != "tool_b" {
		t.Fatalf("bad route: %#v %v", r, ok)
	}
	if backendA.tools[0].Name != "tool_a" {
		t.Fatal("projection mutated backend catalog")
	}
	if c.Tools()[0].Description != "tool a" {
		t.Fatal("projection lost tool metadata")
	}
}

func TestProjectCatalogRejectsCollisionAndInvalidName(t *testing.T) {
	a := &fakeBackend{
		name:  "x",
		tools: []*mcp.Tool{{Name: "a", InputSchema: map[string]any{"type": "object"}}},
	}
	b := &fakeBackend{
		name:  "x",
		tools: []*mcp.Tool{{Name: "a", InputSchema: map[string]any{"type": "object"}}},
	}
	if _, err := ProjectCatalog([]Backend{a, b}); err == nil {
		t.Fatal("expected collision")
	}
	bad := &fakeBackend{
		name:  "bad name",
		tools: []*mcp.Tool{{Name: "a", InputSchema: map[string]any{"type": "object"}}},
	}
	if _, err := ProjectCatalog([]Backend{bad}); err == nil {
		t.Fatal("expected invalid name")
	}
}

func TestBackendErrorIdentity(t *testing.T) {
	sentinel := errors.New("backend exploded")
	b := &fakeBackend{name: "x", tools: nil, err: sentinel}
	_, err := b.Call(context.Background(), "tool", map[string]any{"x": 1})
	if !errors.Is(err, sentinel) {
		t.Fatalf("got %v", err)
	}
}
