package events

import (
	"context"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/starshine-sys/catalogger/common"
)

func (bot *Bot) isIgnored(g discord.GuildID, ch discord.ChannelID, u discord.UserID) bool {
	channel, err := bot.RootChannel(g, ch)
	if err != nil {
		common.Log.Errorf("error getting root channel for channel %v: %v", ch, err)

		return false
	}

	if bot.DB.IsBlacklisted(g, channel.ID) {
		return true
	}

	return bot.isUserIgnored(g, u)
}

func (bot *Bot) isUserIgnored(g discord.GuildID, u discord.UserID) bool {
	var userIgnored bool
	err := bot.DB.QueryRow(context.Background(), "select exists(select id from guilds where $1 = any(ignored_users) and id = $2)", u, g).Scan(&userIgnored)
	if err != nil {
		common.Log.Errorf("Error checking if user is ignored: %v", err)
		return false
	}
	return userIgnored
}
