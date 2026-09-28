package mcpee

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Backend interface {
	Name() string
	Catalog() []*mcp.Tool
	Call(context.Context, string, any) (*mcp.CallToolResult, error)
	Close() error
}

type stdioBackend struct {
	name    string
	session *mcp.ClientSession
	catalog []*mcp.Tool
}

func (b *stdioBackend) Name() string         { return b.name }
func (b *stdioBackend) Catalog() []*mcp.Tool { return b.catalog }
func (b *stdioBackend) Call(ctx context.Context, tool string, args any) (*mcp.CallToolResult, error) {
	return b.session.CallTool(ctx, &mcp.CallToolParams{Name: tool, Arguments: args})
}
func (b *stdioBackend) Close() error { return b.session.Close() }

func ConnectStdioBackend(ctx context.Context, cfg BackendConfig) (Backend, error) {
	cmd := exec.Command(cfg.Command, cfg.Args...)
	cmd.Stderr = os.Stderr
	cmd.Env = append([]string{}, os.Environ()...)
	keys := make([]string, 0, len(cfg.Env))
	for k := range cfg.Env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		cmd.Env = append(cmd.Env, k+"="+cfg.Env[k])
	}

	client := mcp.NewClient(&mcp.Implementation{Name: serverName, Version: Version}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		return nil, fmt.Errorf("connect backend %q: %w", cfg.Name, err)
	}
	res, err := session.ListTools(ctx, nil)
	if err != nil {
		_ = session.Close()
		return nil, fmt.Errorf("list tools for backend %q: %w", cfg.Name, err)
	}
	return &stdioBackend{name: cfg.Name, session: session, catalog: res.Tools}, nil
}
