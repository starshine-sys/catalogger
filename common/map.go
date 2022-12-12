package common

import "sync"

// Map is a concurrent map. It wraps the standard library's map with a mutex for concurrent access.
type Map[K comparable, V any] struct {
	m  map[K]V
	mu sync.RWMutex
}

// NewMap returns a new Map.
func NewMap[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{
		m: make(map[K]V),
	}
}

// Set sets the key k to the value v.
func (m *Map[K, V]) Set(k K, v V) {
	m.mu.Lock()
	m.m[k] = v
	m.mu.Unlock()
}

// Get gets the value at key k, or the zero value if not set.
func (m *Map[K, V]) Get(k K) (v V, ok bool) {
	m.mu.RLock()
	v, ok = m.m[k]
	m.mu.RUnlock()
	return v, ok
}

// Remove removes a key from the map. It returns true if the key existed in the map, false otherwise.
func (m *Map[K, V]) Remove(k K) (exists bool) {
	m.mu.Lock()
	_, exists = m.m[k]
	delete(m.m, k)
	m.mu.Unlock()
	return exists
}

// Exists returns true if key exists in the map, false otherwise.
func (m *Map[K, V]) Exists(k K) (exists bool) {
	m.mu.RLock()
	_, exists = m.m[k]
	m.mu.RUnlock()
	return exists
}

// Length returns the size of m.
func (m *Map[K, V]) Length() int {
	m.mu.RLock()
	c := len(m.m)
	m.mu.RUnlock()
	return c
}

// Values returns all values in m. The returned slice is unordered.
func (m *Map[K, V]) Values() []V {
	values := make([]V, 0, len(m.m))
	for _, v := range m.m {
		values = append(values, v)
	}
	return values
}

// ReadFunc runs fn with the mutex locked for reading.
// The raw map is passed to fn.
func (m *Map[K, V]) ReadFunc(fn func(map[K]V)) {
	m.mu.RLock()
	fn(m.m)
	m.mu.RUnlock()
}

// WriteFunc runs fn with the mutex locked for writing.
// The raw map is passed to fn.
func (m *Map[K, V]) WriteFunc(fn func(map[K]V)) {
	m.mu.Lock()
	fn(m.m)
	m.mu.Unlock()
}
