package main

import (
	"context"

	"github.com/diamondburned/arikawa/v3/api"
)

type contextKey int

var contextKeyDiscord contextKey = 1

// DiscordAPIFromSession ...
func DiscordAPIFromSession(ctx context.Context) (client *api.Client) {
	if val := ctx.Value(contextKeyDiscord); val != nil {
		if cast, ok := val.(*api.Client); ok {
			return cast
		}
	}
	return nil
}
