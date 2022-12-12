package events

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/starshine-sys/catalogger/common"
)

func (bot *Bot) isIgnored(guildID discord.GuildID, channelID discord.ChannelID, userID discord.UserID, appID discord.AppID) bool {
	channel, err := bot.RootChannel(guildID, channelID)
	if err != nil {
		common.Log.Errorf("error getting root channel for channel %v: %v", channelID, err)

		return false
	}

	if bot.DB.IsBlacklisted(guildID, channel.ID) {
		return true
	}

	return bot.isUserIgnored(guildID, userID, appID)
}

var sq = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

func (bot *Bot) isUserIgnored(guildID discord.GuildID, userID discord.UserID, appID discord.AppID) bool {
	if !appID.IsValid() {
		appID = discord.AppID(userID)
	}

	var userIgnored bool
	err := bot.DB.QueryRow(context.Background(),
		"select exists(select id from guilds where ($1 = any(ignored_users) or $2 = any(ignored_users)) and id = $3)",
		userID, appID, guildID,
	).Scan(&userIgnored)
	if err != nil {
		common.Log.Errorf("Error checking if user is ignored: %v", err)
		return false
	}
	return userIgnored
}
