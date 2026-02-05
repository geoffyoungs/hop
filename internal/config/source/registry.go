package source

import (
	"errors"
	"sync"
)

var (
	// ErrSourceNotFound is returned when a source is not found in the registry
	ErrSourceNotFound = errors.New("source not found")
)

var (
	registry = make(map[string]HostSource)
	mu       sync.RWMutex
)

// Register adds a source to the registry.
// Typically called from init() functions in source implementation files.
func Register(s HostSource) {
	mu.Lock()
	defer mu.Unlock()
	registry[s.Name()] = s
}

// Get retrieves a source by name
func Get(name string) (HostSource, error) {
	mu.RLock()
	defer mu.RUnlock()
	s, ok := registry[name]
	if !ok {
		return nil, ErrSourceNotFound
	}
	return s, nil
}

// All returns all registered sources
func All() map[string]HostSource {
	mu.RLock()
	defer mu.RUnlock()
	result := make(map[string]HostSource, len(registry))
	for k, v := range registry {
		result[k] = v
	}
	return result
}

// Names returns the names of all registered sources
func Names() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

// FindSourceForFile returns the first source that can handle the given file
func FindSourceForFile(path string) (HostSource, bool) {
	mu.RLock()
	defer mu.RUnlock()
	for _, s := range registry {
		if s.CanLoad(path) {
			return s, true
		}
	}
	return nil, false
}
