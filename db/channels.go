package db

import (
	"context"

	"emperror.dev/errors"
	"github.com/diamondburned/arikawa/v3/discord"
)

// LogChannels is the map of log channels stored per server
type LogChannels struct {
	GuildUpdate             discord.ChannelID `json:"GUILD_UPDATE"`
	GuildEmojisUpdate       discord.ChannelID `json:"GUILD_EMOJIS_UPDATE"`
	GuildRoleCreate         discord.ChannelID `json:"GUILD_ROLE_CREATE"`
	GuildRoleUpdate         discord.ChannelID `json:"GUILD_ROLE_UPDATE"`
	GuildRoleDelete         discord.ChannelID `json:"GUILD_ROLE_DELETE"`
	ChannelCreate           discord.ChannelID `json:"CHANNEL_CREATE"`
	ChannelUpdate           discord.ChannelID `json:"CHANNEL_UPDATE"`
	ChannelDelete           discord.ChannelID `json:"CHANNEL_DELETE"`
	GuildMemberAdd          discord.ChannelID `json:"GUILD_MEMBER_ADD"`
	GuildMemberUpdate       discord.ChannelID `json:"GUILD_MEMBER_UPDATE"`
	GuildKeyRoleUpdate      discord.ChannelID `json:"GUILD_KEY_ROLE_UPDATE"`
	GuildMemberNickUpdate   discord.ChannelID `json:"GUILD_MEMBER_NICK_UPDATE"`
	GuildMemberAvatarUpdate discord.ChannelID `json:"GUILD_MEMBER_AVATAR_UPDATE"`
	GuildMemberRemove       discord.ChannelID `json:"GUILD_MEMBER_REMOVE"`
	GuildMemberKick         discord.ChannelID `json:"GUILD_MEMBER_KICK"`
	GuildBanAdd             discord.ChannelID `json:"GUILD_BAN_ADD"`
	GuildBanRemove          discord.ChannelID `json:"GUILD_BAN_REMOVE"`
	InviteCreate            discord.ChannelID `json:"INVITE_CREATE"`
	InviteDelete            discord.ChannelID `json:"INVITE_DELETE"`
	MessageUpdate           discord.ChannelID `json:"MESSAGE_UPDATE"`
	MessageDelete           discord.ChannelID `json:"MESSAGE_DELETE"`
	MessageDeleteBulk       discord.ChannelID `json:"MESSAGE_DELETE_BULK"`
}

type Redirects map[string]discord.ChannelID

type Ignores struct {
	GlobalChannels []discord.ChannelID         `json:"global_channels"`
	GlobalUsers    []discord.UserID            `json:"global_users"`
	PerChannel     map[string][]discord.UserID `json:"per_channel"`
}

type Channels struct {
	Channels  LogChannels
	Redirects Redirects
	Ignores   Ignores
}

// For returns the channel ID for the given event.
// TODO: add all events
func (lc LogChannels) For(evName string) discord.ChannelID {
	switch evName {
	case "GuildRoleCreateEvent":
		return lc.GuildRoleCreate
	case "GuildRoleUpdateEvent":
		return lc.GuildRoleUpdate
	case "MessageDeleteEvent":
		return lc.MessageDelete
	}

	return discord.NullChannelID
}

func (db *DB) Channels(guildID discord.GuildID) (chs Channels, err error) {
	sql, args, err := sq.Select("channels", "redirects", "ignores").From("guilds").Where("id = ?", guildID).ToSql()
	if err != nil {
		return chs, errors.Wrap(err, "building sql")
	}

	err = db.QueryRow(context.Background(), sql, args...).Scan(&chs.Channels, &chs.Redirects, &chs.Ignores)
	if err != nil {
		return chs, errors.Wrap(err, "getting channels")
	}
	return chs, nil
}

func (db *DB) SetChannels(guildID discord.GuildID, chs Channels) error {
	sql, args, err := sq.Update("guilds").
		Set("channels", chs.Channels).
		Set("redirects", chs.Redirects).
		Set("ignores", chs.Ignores).
		Where("id = ?", guildID).
		ToSql()
	if err != nil {
		return errors.Wrap(err, "building sql")
	}

	_, err = db.Exec(context.Background(), sql, args...)
	if err != nil {
		return errors.Wrap(err, "executing sql")
	}
	return nil
}
