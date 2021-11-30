package frontend

import (
	"context"
)

type contextKey int

var (
	contextKeyDiscord contextKey = 1
)

// discordAPIFromSession ...
func discordAPIFromSession(ctx context.Context) (cache *userCache) {
	if val := ctx.Value(contextKeyDiscord); val != nil {
		if cast, ok := val.(*userCache); ok {
			return cast
		}
	}
	return nil
}
