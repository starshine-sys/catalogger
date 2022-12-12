package common

import "sync"

// entry is used to make the implementations of Set slightly nicer.
type entry struct{}

// Set is a concurrent set. It is not ordered.
type Set[T comparable] struct {
	m  map[T]entry
	mu sync.RWMutex
}

// NewSet returns a new Set.
func NewSet[T comparable](initial ...T) *Set[T] {
	set := &Set[T]{
		m: make(map[T]entry),
	}

	for i := range initial {
		set.m[initial[i]] = entry{}
	}

	return set
}

// Add adds a value to the set. It returns true if the value doesn't already exist, false otherwise.
func (s *Set[T]) Add(v T) bool {
	s.mu.Lock()
	_, exists := s.m[v]
	s.m[v] = entry{}
	s.mu.Unlock()

	return !exists
}

// Append adds a slice of values to the set.
func (s *Set[T]) Append(values ...T) {
	s.mu.Lock()
	for _, v := range values {
		s.m[v] = entry{}
	}
	s.mu.Unlock()
}

// Remove removes a value from the set. It returns true if the value existed in the set, false otherwise.
func (s *Set[T]) Remove(v T) (exists bool) {
	s.mu.Lock()
	_, exists = s.m[v]
	delete(s.m, v)
	s.mu.Unlock()
	return exists
}

// Exists returns true if v exists in the set, false otherwise.
func (s *Set[T]) Exists(v T) (exists bool) {
	s.mu.RLock()
	_, exists = s.m[v]
	s.mu.RUnlock()
	return exists
}

// Values returns all values in the set. The return slice is unordered.
func (s *Set[T]) Values() []T {
	s.mu.RLock()
	values := make([]T, 0, len(s.m))
	for k := range s.m {
		values = append(values, k)
	}
	s.mu.RUnlock()
	return values
}

// Length returns the length of the set.
func (s *Set[T]) Length() int {
	s.mu.RLock()
	length := len(s.m)
	s.mu.RUnlock()
	return length
}
