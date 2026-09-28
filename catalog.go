package mcpee

import (
	"fmt"
	"regexp"

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
	routes map[string]Route
	tools  []*mcp.Tool
}

func ProjectCatalog(backends []Backend) (*Catalog, error) {
	c := &Catalog{routes: make(map[string]Route)}
	for _, b := range backends {
		if !validToolName.MatchString(b.Name()) {
			return nil, fmt.Errorf("invalid backend name %q", b.Name())
		}
		for _, t := range b.Catalog() {
			if t == nil || !validToolName.MatchString(t.Name) {
				return nil, fmt.Errorf("backend %q has invalid tool name %q", b.Name(), toolName(t))
			}
			name := b.Name() + separator + t.Name
			if !validToolName.MatchString(name) {
				return nil, fmt.Errorf("invalid projected tool name %q", name)
			}
			if _, ok := c.routes[name]; ok {
				return nil, fmt.Errorf("projected tool collision %q", name)
			}
			copy := *t
			copy.Name = name
			p := &copy
			c.routes[name] = Route{Backend: b, Tool: t.Name, Projected: p}
			c.tools = append(c.tools, p)
		}
	}
	return c, nil
}
func toolName(t *mcp.Tool) string {
	if t == nil {
		return ""
	}
	return t.Name
}
func (c *Catalog) Tools() []*mcp.Tool               { return c.tools }
func (c *Catalog) Lookup(name string) (Route, bool) { r, ok := c.routes[name]; return r, ok }
