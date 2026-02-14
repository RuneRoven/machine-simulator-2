package machine

import (
	"fmt"
	"sort"
	"sync"

	"machine-simulator-2/internal/config"
)

// MachineFactory is a constructor function for a machine type.
type MachineFactory func(cfg config.MachineInstanceConfig) Machine

var (
	registryMu sync.RWMutex
	registry   = map[string]MachineFactory{}
)

// Register adds a machine type to the global registry.
// Called from init() in each machine type package.
func Register(typeName string, factory MachineFactory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := registry[typeName]; exists {
		panic(fmt.Sprintf("machine type %q already registered", typeName))
	}
	registry[typeName] = factory
}

// Create instantiates a machine from the registry by type name.
func Create(typeName string, cfg config.MachineInstanceConfig) (Machine, error) {
	registryMu.RLock()
	factory, ok := registry[typeName]
	registryMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown machine type: %q (registered: %v)", typeName, ListTypes())
	}
	return factory(cfg), nil
}

// ListTypes returns all registered machine type names, sorted.
func ListTypes() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	types := make([]string, 0, len(registry))
	for name := range registry {
		types = append(types, name)
	}
	sort.Strings(types)
	return types
}

// IsRegistered checks if a machine type name is in the registry.
func IsRegistered(typeName string) bool {
	registryMu.RLock()
	defer registryMu.RUnlock()
	_, ok := registry[typeName]
	return ok
}
