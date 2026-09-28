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

func NewMCPServer(c *Catalog, log *slog.Logger) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: serverName, Version: Version}, nil)
	for _, tool := range c.Tools() {
		route, _ := c.Lookup(tool.Name)
		r := route
		s.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			start := time.Now()
			log.Info("tool call started", "tool", req.Params.Name, "backend", r.Backend.Name())
			result, err := r.Backend.Call(ctx, r.Tool, req.Params.Arguments)
			if err != nil {
				log.Error(
					"tool call failed", "tool",
					req.Params.Name, "backend", r.Backend.Name(), "duration", time.Since(start), "error", err,
				)
				return nil, err
			}
			log.Info(
				"tool call completed", "tool",
				req.Params.Name, "backend", r.Backend.Name(), "duration", time.Since(start),
			)
			return result, nil
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
