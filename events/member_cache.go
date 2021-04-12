package events

import (
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
)

type memberCacheKey struct {
	GuildID discord.GuildID
	UserID  discord.UserID
}

func (bot *Bot) requestGuildMembers(g *gateway.GuildCreateEvent) {
	bot.State.Gateway.RequestGuildMembers(gateway.RequestGuildMembersData{
		GuildID: []discord.GuildID{g.ID},
		Limit:   0,
	})
}

func (bot *Bot) guildMemberChunk(g *gateway.GuildMembersChunkEvent) {
	bot.MembersMu.Lock()
	defer bot.MembersMu.Unlock()
	for _, m := range g.Members {
		bot.Members[memberCacheKey{
			GuildID: g.GuildID,
			UserID:  m.User.ID,
		}] = m
	}
}
