package events

import (
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/catalogger/db"
)

type memberCacheKey struct {
	GuildID discord.GuildID
	UserID  discord.UserID
}

func (bot *Bot) requestGuildMembers(g *gateway.GuildCreateEvent) {
	bot.ChannelsMu.Lock()
	for _, ch := range g.Channels {
		bot.Channels[ch.ID] = ch
	}
	bot.ChannelsMu.Unlock()

	bot.RolesMu.Lock()
	for _, r := range g.Roles {
		bot.Roles[r.ID] = r
	}
	bot.RolesMu.Unlock()

	err := bot.State(g.ID).Gateway.RequestGuildMembers(gateway.RequestGuildMembersData{
		GuildIDs: []discord.GuildID{g.ID},
		Limit:    0,
	})
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   "GuildCreateEvent",
			Command: "Request guild members",
			GuildID: g.ID,
		}, err)

		time.AfterFunc(time.Minute, func() {
			bot.requestGuildMembers(g)
		})
	}
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
