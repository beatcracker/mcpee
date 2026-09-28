package mcpee

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestRoutingForwardsArgumentsAndResult(t *testing.T) {
	wantResult := &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}
	b := &fakeBackend{
		name:   "backend_a",
		tools:  []*mcp.Tool{{Name: "tool_a", InputSchema: map[string]any{"type": "object"}}},
		result: wantResult,
	}
	c, err := ProjectCatalog([]Backend{b})
	if err != nil {
		t.Fatal(err)
	}
	s := NewMCPServer(c, slog.New(slog.NewTextHandler(io.Discard, nil)))
	st, ct := mcp.NewInMemoryTransports()
	ss, err := s.Connect(context.Background(), st, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ss.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	cs, err := client.Connect(context.Background(), ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()
	got, err := cs.CallTool(
		context.Background(),
		&mcp.CallToolParams{Name: "backend_a__tool_a", Arguments: map[string]any{"value": "hello"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if b.called != "tool_a" {
		t.Fatalf("called %q", b.called)
	}
	raw, ok := b.args.(json.RawMessage)
	if !ok {
		t.Fatalf("args type %T", b.args)
	}
	var gotArgs map[string]any
	if err := json.Unmarshal(raw, &gotArgs); err != nil {
		t.Fatal(err)
	}
	if gotArgs["value"] != "hello" {
		t.Fatalf("args %s", raw)
	}
	if len(got.Content) != 1 {
		t.Fatalf("result %#v", got)
	}
}

func TestCancellationPropagates(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	b := &fakeBackend{name: "x"}
	if _, err := b.Call(ctx, "tool", nil); err != nil {
		t.Fatal(err)
	}
	if ctx.Err() != context.Canceled {
		t.Fatal("context not cancelled")
	}
}
