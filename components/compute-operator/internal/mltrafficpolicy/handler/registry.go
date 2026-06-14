package handler

import (
	"fmt"
	"sync"

	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// registry stores compile-time Handler factories keyed by (backend, engine).
// Each handler package calls Register from init() to make itself discoverable.
var (
	registryMu sync.RWMutex
	factories  = map[Key]Factory{}
)

// Register adds a factory. Panics on duplicate keys.
func Register(key Key, f Factory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := factories[key]; exists {
		panic(fmt.Sprintf("mltrafficpolicy handler: duplicate registration for %s", key))
	}
	factories[key] = f
}

// Keys returns every registered (backend, engine) tuple.
func Keys() []Key {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]Key, 0, len(factories))
	for k := range factories {
		out = append(out, k)
	}
	return out
}

// Build instantiates every registered factory against the manager.
func Build(mgr manager.Manager) (map[Key]Handler, []Handler, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	byKey := make(map[Key]Handler, len(factories))
	all := make([]Handler, 0, len(factories))
	for k, f := range factories {
		h, err := f(mgr)
		if err != nil {
			return nil, nil, fmt.Errorf("handler %s: %w", k, err)
		}
		byKey[k] = h
		all = append(all, h)
	}
	return byKey, all, nil
}
