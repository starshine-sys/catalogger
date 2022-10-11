// Package cache contains handlers that are *only* used for caching.
// These should not call bot.Send at any point,
// and should not cache anything that may also be responded to (such as role updates).
package cache

import (
	"sync"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/session/shard"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/starshine-sys/catalogger/v2/bot"
)

type Bot struct {
	*bot.Bot

	guildsMu             sync.RWMutex
	guildsToFetchMembers map[int]map[discord.GuildID]struct{}
	guildsToFetchInvites map[int]map[discord.GuildID]struct{}
}

func Setup(root *bot.Bot) {
	bot := &Bot{
		Bot:                  root,
		guildsToFetchMembers: make(map[int]map[discord.GuildID]struct{}),
		guildsToFetchInvites: make(map[int]map[discord.GuildID]struct{}),
	}

	bot.AddHandler(
		// Cache guild (separate from logging as *all* guilds should be cached, not just new guilds)
		// Also add the guild to the fetch queue if needed
		bot.guildCreate,
		// Cache guild members when they're received
		bot.guildMembersChunk,
	)

	// set up fetch loop
	bot.Router.ShardManager.ForEach(func(shard shard.Shard) {
		s := shard.(*state.State)

		var o sync.Once
		s.AddHandler(func(*gateway.ReadyEvent) {
			o.Do(func() {
				go bot.fetchLoop(s)
			})
		})
	})
}
