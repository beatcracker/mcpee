package mcpee

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type fakeClientSession struct {
	mu        sync.Mutex
	calls     int
	result    *mcp.CallToolResult
	err       error
	done      chan struct{}
	closeOnce sync.Once
}

func newFakeClientSession(result *mcp.CallToolResult, err error) *fakeClientSession {
	return &fakeClientSession{
		result: result,
		err:    err,
		done:   make(chan struct{}),
	}
}

func (s *fakeClientSession) CallTool(
	context.Context,
	*mcp.CallToolParams,
) (*mcp.CallToolResult, error) {
	s.mu.Lock()
	s.calls++
	result := s.result
	err := s.err
	s.mu.Unlock()

	if errors.Is(err, mcp.ErrConnectionClosed) {
		s.terminate()
	}
	return result, err
}

func (s *fakeClientSession) Close() error {
	s.terminate()
	return nil
}

func (s *fakeClientSession) Wait() error {
	<-s.done
	return nil
}

func (s *fakeClientSession) terminate() {
	s.closeOnce.Do(func() {
		close(s.done)
	})
}

func (s *fakeClientSession) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func testTool(name string) *mcp.Tool {
	return &mcp.Tool{
		Name:        name,
		InputSchema: map[string]any{"type": "object"},
	}
}

func TestStdioBackendRecoversOnNextCallWithoutReplay(t *testing.T) {
	tool := testTool("run")
	transportErr := errors.New("test transport failure")
	first := newFakeClientSession(nil, transportErr)
	wantResult := &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}
	second := newFakeClientSession(wantResult, nil)

	var connects atomic.Int32
	b := &stdioBackend{
		name:    "exec",
		cfg:     BackendConfig{Name: "exec", Command: "unused"},
		session: watchSession(first),
		catalog: []*mcp.Tool{tool},
		connect: func(context.Context, BackendConfig) (clientSession, []*mcp.Tool, error) {
			connects.Add(1)
			return second, []*mcp.Tool{tool}, nil
		},
	}

	var reconciles atomic.Int32
	b.setCatalogReconciler(func([]*mcp.Tool) error {
		reconciles.Add(1)
		return nil
	})

	if _, err := b.Call(context.Background(), "run", nil); !errors.Is(err, transportErr) {
		t.Fatalf("first call error = %v, want %v", err, transportErr)
	}
	if first.callCount() != 1 {
		t.Fatalf("first session calls = %d, want 1", first.callCount())
	}
	if connects.Load() != 0 {
		t.Fatalf("reconnects during failed call = %d, want 0", connects.Load())
	}

	got, err := b.Call(context.Background(), "run", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != wantResult {
		t.Fatalf("result = %#v, want %#v", got, wantResult)
	}
	if first.callCount() != 1 {
		t.Fatalf("failed invocation replayed: first session calls = %d", first.callCount())
	}
	if second.callCount() != 1 {
		t.Fatalf("replacement session calls = %d, want 1", second.callCount())
	}
	if connects.Load() != 1 {
		t.Fatalf("reconnects = %d, want 1", connects.Load())
	}
	if reconciles.Load() != 1 {
		t.Fatalf("reconciles = %d, want 1", reconciles.Load())
	}
}

func TestStdioBackendReusesHealthySession(t *testing.T) {
	tool := testTool("run")
	session := newFakeClientSession(&mcp.CallToolResult{}, nil)

	var connects atomic.Int32
	b := &stdioBackend{
		name:    "exec",
		session: watchSession(session),
		catalog: []*mcp.Tool{tool},
		connect: func(context.Context, BackendConfig) (clientSession, []*mcp.Tool, error) {
			connects.Add(1)
			return nil, nil, errors.New("unexpected reconnect")
		},
	}

	for range 2 {
		if _, err := b.Call(context.Background(), "run", nil); err != nil {
			t.Fatal(err)
		}
	}
	if session.callCount() != 2 {
		t.Fatalf("calls = %d, want 2", session.callCount())
	}
	if connects.Load() != 0 {
		t.Fatalf("reconnects = %d, want 0", connects.Load())
	}
}

func TestStdioBackendSerializesConcurrentRecovery(t *testing.T) {
	tool := testTool("run")
	session := newFakeClientSession(&mcp.CallToolResult{}, nil)

	started := make(chan struct{})
	release := make(chan struct{})
	var startOnce sync.Once
	var connects atomic.Int32

	b := &stdioBackend{
		name:    "exec",
		cfg:     BackendConfig{Name: "exec", Command: "unused"},
		catalog: []*mcp.Tool{tool},
		connect: func(context.Context, BackendConfig) (clientSession, []*mcp.Tool, error) {
			connects.Add(1)
			startOnce.Do(func() { close(started) })
			<-release
			return session, []*mcp.Tool{tool}, nil
		},
	}

	const callers = 10
	errs := make(chan error, callers)
	for range callers {
		go func() {
			_, err := b.Call(context.Background(), "run", nil)
			errs <- err
		}()
	}

	<-started
	close(release)

	for range callers {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
	if connects.Load() != 1 {
		t.Fatalf("reconnects = %d, want 1", connects.Load())
	}
	if session.callCount() != callers {
		t.Fatalf("calls = %d, want %d", session.callCount(), callers)
	}
}

func TestStdioBackendClosePreventsRecovery(t *testing.T) {
	var connects atomic.Int32
	b := &stdioBackend{
		name: "exec",
		cfg:  BackendConfig{Name: "exec", Command: "unused"},
		connect: func(context.Context, BackendConfig) (clientSession, []*mcp.Tool, error) {
			connects.Add(1)
			return newFakeClientSession(&mcp.CallToolResult{}, nil), []*mcp.Tool{testTool("run")}, nil
		},
	}

	if err := b.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Call(context.Background(), "run", nil); err == nil {
		t.Fatal("expected call after close to fail")
	}
	if connects.Load() != 0 {
		t.Fatalf("reconnects after close = %d, want 0", connects.Load())
	}
}

func TestStdioBackendRemovedToolIsNotCalledAfterRecovery(t *testing.T) {
	oldTool := testTool("old")
	replacement := newFakeClientSession(&mcp.CallToolResult{}, nil)

	b := &stdioBackend{
		name:    "exec",
		cfg:     BackendConfig{Name: "exec", Command: "unused"},
		catalog: []*mcp.Tool{oldTool},
		connect: func(context.Context, BackendConfig) (clientSession, []*mcp.Tool, error) {
			return replacement, nil, nil
		},
	}
	b.setCatalogReconciler(func([]*mcp.Tool) error { return nil })

	if _, err := b.Call(context.Background(), "old", nil); err == nil {
		t.Fatal("expected removed tool to fail")
	}
	if replacement.callCount() != 0 {
		t.Fatalf("removed tool reached replacement backend %d times", replacement.callCount())
	}
}

func TestStdioBackendProtocolErrorKeepsSession(t *testing.T) {
	tool := testTool("run")
	protocolErr := &jsonrpc.Error{
		Code:    jsonrpc.CodeInvalidParams,
		Message: "bad request",
	}
	session := newFakeClientSession(nil, protocolErr)

	var connects atomic.Int32
	b := &stdioBackend{
		name:    "exec",
		session: watchSession(session),
		catalog: []*mcp.Tool{tool},
		connect: func(context.Context, BackendConfig) (clientSession, []*mcp.Tool, error) {
			connects.Add(1)
			return nil, nil, errors.New("unexpected reconnect")
		},
	}

	for range 2 {
		if _, err := b.Call(context.Background(), "run", nil); !errors.Is(err, protocolErr) {
			t.Fatalf("call error = %v, want protocol error", err)
		}
	}
	if session.callCount() != 2 {
		t.Fatalf("calls = %d, want 2", session.callCount())
	}
	if connects.Load() != 0 {
		t.Fatalf("reconnects = %d, want 0", connects.Load())
	}
}
