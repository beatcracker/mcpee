package mcpee

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const separator = "__"

var validToolName = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)

type Route struct {
	Backend   Backend
	Tool      string
	Projected *mcp.Tool
}

type Catalog struct {
	mu       sync.RWMutex
	backends []Backend
	routes   map[string]Route
	tools    []*mcp.Tool
}

type CatalogChange struct {
	Removed  []string
	Upserted []*mcp.Tool
}

func ProjectCatalog(backends []Backend) (*Catalog, error) {
	c := &Catalog{
		backends: append([]Backend(nil), backends...),
		routes:   make(map[string]Route),
	}
	for _, b := range backends {
		routes, err := projectBackendTools(b, b.Catalog())
		if err != nil {
			return nil, err
		}
		for _, route := range routes {
			name := route.Projected.Name
			if _, ok := c.routes[name]; ok {
				return nil, fmt.Errorf("projected tool collision %q", name)
			}
			c.routes[name] = route
			c.tools = append(c.tools, route.Projected)
		}
	}
	return c, nil
}

func projectBackendTools(b Backend, tools []*mcp.Tool) ([]Route, error) {
	if !validToolName.MatchString(b.Name()) {
		return nil, fmt.Errorf("invalid backend name %q", b.Name())
	}

	routes := make([]Route, 0, len(tools))
	seen := make(map[string]bool, len(tools))
	for _, t := range tools {
		if t == nil || !validToolName.MatchString(t.Name) {
			return nil, fmt.Errorf("backend %q has invalid tool name %q", b.Name(), toolName(t))
		}
		name := b.Name() + separator + t.Name
		if !validToolName.MatchString(name) {
			return nil, fmt.Errorf("invalid projected tool name %q", name)
		}
		if seen[name] {
			return nil, fmt.Errorf("projected tool collision %q", name)
		}
		seen[name] = true

		copy := *t
		copy.Name = name
		projected := &copy
		routes = append(routes, Route{
			Backend:   b,
			Tool:      t.Name,
			Projected: projected,
		})
	}
	return routes, nil
}

func (c *Catalog) ReconcileBackend(b Backend, tools []*mcp.Tool) (CatalogChange, error) {
	projected, err := projectBackendTools(b, tools)
	if err != nil {
		return CatalogChange{}, err
	}

	next := make(map[string]Route, len(projected))
	for _, route := range projected {
		next[route.Projected.Name] = route
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for name := range next {
		if current, ok := c.routes[name]; ok && current.Backend.Name() != b.Name() {
			return CatalogChange{}, fmt.Errorf("projected tool collision %q", name)
		}
	}

	current := make(map[string]Route)
	for name, route := range c.routes {
		if route.Backend.Name() == b.Name() {
			current[name] = route
		}
	}

	var change CatalogChange
	for name := range current {
		if _, ok := next[name]; !ok {
			change.Removed = append(change.Removed, name)
		}
	}
	for name, route := range next {
		old, ok := current[name]
		if !ok || !reflect.DeepEqual(old.Projected, route.Projected) {
			change.Upserted = append(change.Upserted, route.Projected)
		}
	}

	for name := range current {
		delete(c.routes, name)
	}
	for name, route := range next {
		c.routes[name] = route
	}

	names := make([]string, 0, len(c.routes))
	for name := range c.routes {
		names = append(names, name)
	}
	sort.Strings(names)
	c.tools = c.tools[:0]
	for _, name := range names {
		c.tools = append(c.tools, c.routes[name].Projected)
	}

	sort.Strings(change.Removed)
	sort.Slice(change.Upserted, func(i, j int) bool {
		return change.Upserted[i].Name < change.Upserted[j].Name
	})
	return change, nil
}

func toolName(t *mcp.Tool) string {
	if t == nil {
		return ""
	}
	return t.Name
}

func unknownToolError(name string) error {
	return &jsonrpc.Error{
		Code:    jsonrpc.CodeInvalidParams,
		Message: fmt.Sprintf("unknown tool %q", name),
	}
}

func (c *Catalog) Tools() []*mcp.Tool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]*mcp.Tool(nil), c.tools...)
}

func (c *Catalog) Backends() []Backend {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]Backend(nil), c.backends...)
}

func (c *Catalog) Lookup(name string) (Route, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	route, ok := c.routes[name]
	return route, ok
}
