// Package registry provides a simple thread-safe tool registry for Atlas.
package registry

import (
	"sort"
	"sync"

	"github.com/Yashh56/atlas/internal/tools"
)

// Registry holds named Tool implementations and allows callers to look them up
// by name.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]tools.Tool
}

// NewRegistry returns an empty, ready-to-use Registry.
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]tools.Tool)}
}

// Register adds t to the registry under its Name(). If a tool with the same
// name already exists it is silently replaced.
func (r *Registry) Register(t tools.Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.Name()] = t
}

// Get returns the tool registered under name and whether it was found.
func (r *Registry) Get(name string) (tools.Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// List returns a sorted slice of all registered tool names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
