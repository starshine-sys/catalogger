package main

import (
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
)

// userCache is a cached user's Client and User objects
type userCache struct {
	*api.Client
	User *discord.User
}

// getUser gets a user from their cookie's value
// only half of the cookie (in base 64) is used
func (s *server) getUser(cookie string) (*userCache, bool) {
	v, err := s.UserCache.Get(cookie[:80])
	if err == nil {
		c, ok := v.(*userCache)
		if ok {
			return c, true
		}
	}
	return nil, false
}

// setUser sets the user's client and user object in the cache
func (s *server) setUser(cookie string, c *userCache) {
	s.UserCache.Set(cookie[:80], c)
}
