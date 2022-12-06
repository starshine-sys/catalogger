package common

import (
	"context"

	"github.com/diamondburned/arikawa/v3/state"
)

// WaitFor is a wrapper around s.WaitFor that checks for a specific type
// and accepts a filter function based on that, avoiding type casting in the filter function.
func WaitFor[T any](ctx context.Context, s *state.State, filter func(t T) bool) (t T, ok bool) {
	v := s.WaitFor(ctx, func(i interface{}) bool {
		if t, ok := i.(T); ok {
			return filter(t)
		}
		return false
	})

	if v == nil {
		return *new(T), false
	}

	t, ok = v.(T)
	if !ok {
		return *new(T), false
	}

	return t, true
}
