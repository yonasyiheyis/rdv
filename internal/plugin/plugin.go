package plugin

import (
	"sync"

	"github.com/spf13/cobra"
)

// Plugin defines the minimal contract every plugin must satisfy.
type Plugin interface {
	// Name returns something unique like "aws" or "db".
	Name() string
	// Register attaches the plugin's sub‑commands to the given root cmd.
	Register(root *cobra.Command)
}

var (
	mu       sync.RWMutex
	registry = make(map[string]Plugin)
)

// Register stores p so LoadAll can later wire it into Cobra.
// It panics if two plugins share the same name.
func Register(p Plugin) {
	mu.Lock()
	defer mu.Unlock()

	if _, exists := registry[p.Name()]; exists {
		panic("plugin '" + p.Name() + "' already registered")
	}
	registry[p.Name()] = p
}

// LoadAll iterates over registered plugins and calls their Register func.
func LoadAll(root *cobra.Command) {
	mu.RLock()
	defer mu.RUnlock()

	for _, p := range registry {
		p.Register(root)
	}
}
