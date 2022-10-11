package cache

import (
	"context"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/catalogger/v2/common/log"
)

func (bot *Bot) guildCreate(ev *gateway.GuildCreateEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := bot.Cabinet.GuildSet(ctx, ev.Guild)
	if err != nil {
		log.Errorf("setting guild %v: %v", err)
		return
	}

	isCached, err := bot.Cabinet.IsGuildCached(ctx, ev.ID)
	if err != nil {
		log.Errorf("checking if guild %v is cached: %v", ev.ID, err)
		return
	}

	if isCached {
		return
	}

	_, shardID := bot.Router.ShardManager.FromGuildID(ev.ID)

	bot.guildsMu.Lock()
	defer bot.guildsMu.Unlock()

	bot.addToMemberFetchQueue(shardID, ev.ID)
	bot.addToInviteFetchQueue(shardID, ev.ID)
}

func (bot *Bot) addToMemberFetchQueue(shardID int, guildID discord.GuildID) {
	if bot.guildsToFetchMembers[shardID] == nil {
		bot.guildsToFetchMembers[shardID] = make(map[discord.GuildID]struct{})
	}
	bot.guildsToFetchMembers[shardID][guildID] = struct{}{}
}

func (bot *Bot) addToInviteFetchQueue(shardID int, guildID discord.GuildID) {
	if bot.guildsToFetchInvites[shardID] == nil {
		bot.guildsToFetchInvites[shardID] = make(map[discord.GuildID]struct{})
	}
	bot.guildsToFetchInvites[shardID][guildID] = struct{}{}
}
