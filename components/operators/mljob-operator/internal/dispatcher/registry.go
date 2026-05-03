package dispatcher

import (
	"fmt"
	"sync"

	"axisml.io/operators/mljob/internal/handler"
)

// Registry holds the (backend, engine) → Handler routing table. Handler
// packages register themselves through their package init() to keep the
// dispatcher decoupled from concrete backends. Lookups are read-heavy so
// a sync.RWMutex is sufficient.
type Registry struct {
	mu       sync.RWMutex
	handlers map[handler.Key]handler.Handler
}

// NewRegistry constructs an empty registry. The operator binary's
// main.go decides which handlers to install.
func NewRegistry() *Registry {
	return &Registry{handlers: map[handler.Key]handler.Handler{}}
}

// Register associates a handler with its declared (backend, engine).
// Duplicate registrations panic — they indicate a packaging bug.
func (r *Registry) Register(h handler.Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := h.Key()
	if _, exists := r.handlers[k]; exists {
		panic(fmt.Sprintf("dispatcher: duplicate handler for backend=%s engine=%s", k.Backend, k.Engine))
	}
	r.handlers[k] = h
}

// Lookup returns the handler for (backend, engine) or false when no
// handler claims the tuple. Dispatcher writes status.phase=Failed for
// the unregistered case (design §5).
func (r *Registry) Lookup(backend, engine string) (handler.Handler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handlers[handler.Key{Backend: backend, Engine: engine}]
	return h, ok
}

// All returns a snapshot slice of every registered handler. Used by
// SetupWithManager to wire watch targets.
func (r *Registry) All() []handler.Handler {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]handler.Handler, 0, len(r.handlers))
	for _, h := range r.handlers {
		out = append(out, h)
	}
	return out
}
