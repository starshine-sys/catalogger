package common

import (
	"context"
)

type StateWaiter interface {
	WaitFor(context.Context, func(any) bool) any
}

// WaitFor is a wrapper around s.WaitFor that checks for a specific type
// and accepts a filter function based on that, avoiding type casting in the filter function.
func WaitFor[T any](ctx context.Context, s StateWaiter, filter func(t T) bool) (t T, ok bool) {
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
