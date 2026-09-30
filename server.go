package mcpee

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const Version = "0.1.0"
const serverName = "mcpee"

type catalogReconciler interface {
	setCatalogReconciler(func([]*mcp.Tool) error)
}

func NewMCPServer(c *Catalog, log *slog.Logger) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: serverName, Version: Version}, nil)

	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		route, ok := c.Lookup(req.Params.Name)
		if !ok {
			return nil, unknownToolError(req.Params.Name)
		}

		start := time.Now()
		log.Info("tool call started", "tool", req.Params.Name, "backend", route.Backend.Name())
		result, err := route.Backend.Call(ctx, route.Tool, req.Params.Arguments)
		if err != nil {
			log.Error(
				"tool call failed", "tool",
				req.Params.Name, "backend", route.Backend.Name(), "duration", time.Since(start), "error", err,
			)
			return nil, err
		}
		log.Info(
			"tool call completed", "tool",
			req.Params.Name, "backend", route.Backend.Name(), "duration", time.Since(start),
		)
		return result, nil
	}

	for _, tool := range c.Tools() {
		s.AddTool(tool, handler)
	}

	for _, backend := range c.Backends() {
		recoverable, ok := backend.(catalogReconciler)
		if !ok {
			continue
		}
		b := backend
		recoverable.setCatalogReconciler(func(tools []*mcp.Tool) error {
			change, err := c.ReconcileBackend(b, tools)
			if err != nil {
				return err
			}
			if len(change.Removed) > 0 {
				s.RemoveTools(change.Removed...)
			}
			for _, tool := range change.Upserted {
				s.AddTool(tool, handler)
			}
			return nil
		})
	}

	return s
}

func HTTPHandler(s *mcp.Server) http.Handler {
	mux := http.NewServeMux()
	mux.Handle(
		"/mcp",
		mcp.NewStreamableHTTPHandler(
			func(*http.Request) *mcp.Server { return s },
			&mcp.StreamableHTTPOptions{Stateless: true},
		),
	)
	return mux
}

func CloseBackends(backends []Backend) error {
	var first error
	for i := len(backends) - 1; i >= 0; i-- {
		if err := backends[i].Close(); err != nil && first == nil {
			first = fmt.Errorf("close backend %q: %w", backends[i].Name(), err)
		}
	}
	return first
}
