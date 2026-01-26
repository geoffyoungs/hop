package backend

import (
	"errors"
	"sync"
)

var (
	ErrBackendNotFound = errors.New("backend not found")
	ErrNotSupported    = errors.New("operation not supported by this backend")
)

var (
	registry = make(map[string]Backend)
	mu       sync.RWMutex
)

// Register adds a backend to the registry
func Register(b Backend) {
	mu.Lock()
	defer mu.Unlock()
	registry[b.Name()] = b
}

// Get retrieves a backend by name
func Get(name string) (Backend, error) {
	mu.RLock()
	defer mu.RUnlock()
	b, ok := registry[name]
	if !ok {
		return nil, ErrBackendNotFound
	}
	return b, nil
}

// All returns all registered backends
func All() map[string]Backend {
	mu.RLock()
	defer mu.RUnlock()
	result := make(map[string]Backend, len(registry))
	for k, v := range registry {
		result[k] = v
	}
	return result
}
