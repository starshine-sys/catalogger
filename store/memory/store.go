// Package memory provides an in-memory store.
package memory

import (
	"sync"

	"github.com/diamondburned/arikawa/v3/discord"
)

type Store struct {
	guilds   map[discord.GuildID]*discord.Guild
	guildsMu sync.RWMutex

	channels      map[discord.ChannelID]*discord.Channel
	guildChannels map[discord.GuildID][]discord.ChannelID
	channelsMu    sync.RWMutex

	roles      map[discord.RoleID]*discord.Role
	guildRoles map[discord.GuildID][]discord.RoleID
	rolesMu    sync.RWMutex
}

func New() *Store {
	return &Store{
		guilds:        make(map[discord.GuildID]*discord.Guild),
		channels:      make(map[discord.ChannelID]*discord.Channel),
		guildChannels: make(map[discord.GuildID][]discord.ChannelID),
		roles:         make(map[discord.RoleID]*discord.Role),
		guildRoles:    make(map[discord.GuildID][]discord.RoleID),
	}
}

// remove removes the given value in slice.
func remove[T comparable](slice []T, val T) []T {
	for i := range slice {
		if slice[i] == val {
			if i == 0 {
				slice = slice[1:]
			} else if i == len(slice)-1 {
				slice = slice[:len(slice)-1]
			} else {
				slice = append(slice[:i], slice[i+1:]...)
			}
			break
		}
	}
	return slice
}

// contains returns true if slice contains val.
func contains[T comparable](slice []T, val T) bool {
	for i := range slice {
		if slice[i] == val {
			return true
		}
	}
	return false
}
