package storageregistry

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/storage"
)

// Factory builds a storage adapter from provider-specific config.
type Factory func(raw json.RawMessage) (storage.StorageAdapter, error)

type Registry struct {
	mu        sync.RWMutex
	factories map[string]Factory
}

func New() *Registry {
	return &Registry{factories: make(map[string]Factory)}
}

func (r *Registry) Register(name string, factory Factory) error {
	normalized := strings.TrimSpace(strings.ToLower(name))
	if normalized == "" {
		return fmt.Errorf("register storage adapter: name must not be empty")
	}
	if factory == nil {
		return fmt.Errorf("register storage adapter %q: factory must not be nil", normalized)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[normalized]; exists {
		return fmt.Errorf("register storage adapter %q: already registered", normalized)
	}

	r.factories[normalized] = factory
	return nil
}

func (r *Registry) MustRegister(name string, factory Factory) {
	if err := r.Register(name, factory); err != nil {
		panic(err)
	}
}

func (r *Registry) New(name string, raw json.RawMessage) (storage.StorageAdapter, error) {
	normalized := strings.TrimSpace(strings.ToLower(name))
	if normalized == "" {
		return nil, fmt.Errorf("new storage adapter: name must not be empty")
	}

	r.mu.RLock()
	factory, exists := r.factories[normalized]
	r.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("new storage adapter %q: unknown adapter", normalized)
	}

	adapter, err := factory(raw)
	if err != nil {
		return nil, fmt.Errorf("new storage adapter %q: %w", normalized, err)
	}

	return adapter, nil
}

func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
