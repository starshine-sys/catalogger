package db

import (
	"context"

	"emperror.dev/errors"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/georgysavva/scany/pgxscan"
)

// LogChannels is the map of log channels stored per server
type LogChannels struct {
	GuildUpdate           discord.ChannelID `json:"GUILD_UPDATE"`
	GuildEmojisUpdate     discord.ChannelID `json:"GUILD_EMOJIS_UPDATE"`
	GuildRoleCreate       discord.ChannelID `json:"GUILD_ROLE_CREATE"`
	GuildRoleUpdate       discord.ChannelID `json:"GUILD_ROLE_UPDATE"`
	GuildRoleDelete       discord.ChannelID `json:"GUILD_ROLE_DELETE"`
	ChannelCreate         discord.ChannelID `json:"CHANNEL_CREATE"`
	ChannelUpdate         discord.ChannelID `json:"CHANNEL_UPDATE"`
	ChannelDelete         discord.ChannelID `json:"CHANNEL_DELETE"`
	GuildMemberAdd        discord.ChannelID `json:"GUILD_MEMBER_ADD"`
	GuildMemberUpdate     discord.ChannelID `json:"GUILD_MEMBER_UPDATE"`
	GuildKeyRoleUpdate    discord.ChannelID `json:"GUILD_KEY_ROLE_UPDATE"`
	GuildMemberNickUpdate discord.ChannelID `json:"GUILD_MEMBER_NICK_UPDATE"`
	GuildMemberRemove     discord.ChannelID `json:"GUILD_MEMBER_REMOVE"`
	GuildMemberKick       discord.ChannelID `json:"GUILD_MEMBER_KICK"`
	GuildBanAdd           discord.ChannelID `json:"GUILD_BAN_ADD"`
	GuildBanRemove        discord.ChannelID `json:"GUILD_BAN_REMOVE"`
	InviteCreate          discord.ChannelID `json:"INVITE_CREATE"`
	InviteDelete          discord.ChannelID `json:"INVITE_DELETE"`
	MessageUpdate         discord.ChannelID `json:"MESSAGE_UPDATE"`
	MessageDelete         discord.ChannelID `json:"MESSAGE_DELETE"`
	MessageDeleteBulk     discord.ChannelID `json:"MESSAGE_DELETE_BULK"`
}

// For returns the channel ID for the given event.
// TODO: add all events
func (lc LogChannels) For(evName string) discord.ChannelID {
	switch evName {
	case "GuildRoleCreateEvent":
		return lc.GuildRoleCreate
	}

	return discord.NullChannelID
}

func (db *DB) Channels(guildID discord.GuildID) (lc LogChannels, err error) {
	sql, args, err := sq.Select("channels").From("guilds").Where("id = ?", guildID).ToSql()
	if err != nil {
		return lc, errors.Wrap(err, "building sql")
	}

	err = pgxscan.Get(context.Background(), db, &lc, sql, args...)
	if err != nil {
		return lc, errors.Wrap(err, "getting channels")
	}
	return lc, nil
}
