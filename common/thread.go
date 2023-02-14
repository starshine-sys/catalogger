package common

import "github.com/diamondburned/arikawa/v3/discord"

func IsThread(ch discord.Channel) bool {
	return ch.Type == discord.GuildNewsThread || ch.Type == discord.GuildPrivateThread || ch.Type == discord.GuildPublicThread
}
