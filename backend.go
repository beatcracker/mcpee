package mcpee

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Backend interface {
	Name() string
	Catalog() []*mcp.Tool
	Call(context.Context, string, any) (*mcp.CallToolResult, error)
	Close() error
}

type clientSession interface {
	CallTool(context.Context, *mcp.CallToolParams) (*mcp.CallToolResult, error)
	Close() error
	Wait() error
}

type backendSession struct {
	client clientSession
	done   chan struct{}
}

func watchSession(client clientSession) *backendSession {
	s := &backendSession{client: client, done: make(chan struct{})}
	go func() {
		_ = client.Wait()
		close(s.done)
	}()
	return s
}

func (s *backendSession) alive() bool {
	select {
	case <-s.done:
		return false
	default:
		return true
	}
}

type backendConnector func(context.Context, BackendConfig) (clientSession, []*mcp.Tool, error)

type stdioBackend struct {
	name string
	cfg  BackendConfig

	mu        sync.Mutex
	session   *backendSession
	catalog   []*mcp.Tool
	connect   backendConnector
	reconcile func([]*mcp.Tool) error

	closing atomic.Bool
}

func (b *stdioBackend) Name() string { return b.name }

func (b *stdioBackend) Catalog() []*mcp.Tool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]*mcp.Tool(nil), b.catalog...)
}

func (b *stdioBackend) Call(
	ctx context.Context,
	tool string,
	args any,
) (*mcp.CallToolResult, error) {
	session, err := b.sessionForTool(ctx, tool)
	if err != nil {
		return nil, err
	}

	result, err := session.client.CallTool(ctx, &mcp.CallToolParams{Name: tool, Arguments: args})
	if sessionUnusable(ctx, err) {
		b.invalidate(session)
	}
	return result, err
}

func sessionUnusable(ctx context.Context, err error) bool {
	if err == nil {
		return false
	}
	if ctx.Err() != nil ||
		errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var rpcErr *jsonrpc.Error
	return !errors.As(err, &rpcErr)
}

func (b *stdioBackend) sessionForTool(ctx context.Context, tool string) (*backendSession, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closing.Load() {
		return nil, fmt.Errorf("backend %q is closed", b.name)
	}

	if b.session == nil || !b.session.alive() {
		b.session = nil

		connect := b.connect
		if connect == nil {
			connect = connectStdioSession
		}
		client, tools, err := connect(ctx, b.cfg)
		if err != nil {
			return nil, err
		}
		candidate := watchSession(client)

		closeCandidate := func() {
			_ = candidate.client.Close()
		}

		if b.closing.Load() {
			closeCandidate()
			return nil, fmt.Errorf("backend %q is closed", b.name)
		}
		if b.reconcile != nil {
			if err := b.reconcile(tools); err != nil {
				closeCandidate()
				return nil, fmt.Errorf("reconcile backend %q tools: %w", b.name, err)
			}
		}
		if b.closing.Load() {
			closeCandidate()
			return nil, fmt.Errorf("backend %q is closed", b.name)
		}
		if !candidate.alive() {
			closeCandidate()
			return nil, fmt.Errorf("connect backend %q: %w", b.name, mcp.ErrConnectionClosed)
		}

		b.catalog = append([]*mcp.Tool(nil), tools...)
		b.session = candidate
	}

	if !hasTool(b.catalog, tool) {
		return nil, unknownToolError(b.name + separator + tool)
	}
	return b.session, nil
}

func hasTool(tools []*mcp.Tool, name string) bool {
	for _, tool := range tools {
		if tool != nil && tool.Name == name {
			return true
		}
	}
	return false
}

func (b *stdioBackend) invalidate(session *backendSession) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.session == session {
		b.session = nil
	}
}

func (b *stdioBackend) setCatalogReconciler(reconcile func([]*mcp.Tool) error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.reconcile = reconcile
}

func (b *stdioBackend) Close() error {
	b.closing.Store(true)

	b.mu.Lock()
	session := b.session
	b.session = nil
	b.mu.Unlock()

	if session == nil {
		return nil
	}
	return session.client.Close()
}

func ConnectStdioBackend(ctx context.Context, cfg BackendConfig) (Backend, error) {
	client, tools, err := connectStdioSession(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &stdioBackend{
		name:    cfg.Name,
		cfg:     cfg,
		session: watchSession(client),
		catalog: append([]*mcp.Tool(nil), tools...),
		connect: connectStdioSession,
	}, nil
}

func backendCommand(cfg BackendConfig) (*exec.Cmd, error) {
	cwd, err := effectiveCwd(cfg.Cwd)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(cfg.Command, cfg.Args...)
	cmd.Dir = cwd
	cmd.Stderr = os.Stderr
	if cfg.InheritEnv {
		cmd.Env = append([]string{}, os.Environ()...)
	} else {
		cmd.Env = []string{}
	}

	keys := make([]string, 0, len(cfg.Env))
	for k := range cfg.Env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		cmd.Env = append(cmd.Env, k+"="+cfg.Env[k])
	}
	return cmd, nil
}

func (cfg BackendConfig) EffectiveMaxFrameBytes() int {
	if cfg.MaxFrameSize == 0 {
		return mcp.DefaultMaxLineLength
	}
	return cfg.MaxFrameSize
}

func connectStdioSession(ctx context.Context, cfg BackendConfig) (clientSession, []*mcp.Tool, error) {
	cmd, err := backendCommand(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("prepare backend %q: %w", cfg.Name, err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: serverName, Version: Version}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{
		Command:       cmd,
		MaxLineLength: cfg.MaxFrameSize,
	}, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("connect backend %q: %w", cfg.Name, err)
	}
	res, err := session.ListTools(ctx, nil)
	if err != nil {
		_ = session.Close()
		return nil, nil, fmt.Errorf("list tools for backend %q: %w", cfg.Name, err)
	}
	return session, res.Tools, nil
}
