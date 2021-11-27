package db

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/starshine-sys/catalogger/common"
)

// Events is a list of all available events
var Events []string = []string{
	"GUILD_UPDATE",
	"GUILD_EMOJIS_UPDATE",

	"GUILD_ROLE_CREATE",
	"GUILD_ROLE_UPDATE",
	"GUILD_ROLE_DELETE",

	"CHANNEL_CREATE",
	"CHANNEL_UPDATE",
	"CHANNEL_DELETE",

	"GUILD_MEMBER_ADD",
	"GUILD_MEMBER_UPDATE",
	"GUILD_MEMBER_NICK_UPDATE",
	"GUILD_KEY_ROLE_UPDATE",
	"GUILD_MEMBER_REMOVE",

	"GUILD_BAN_ADD",
	"GUILD_BAN_REMOVE",
	"GUILD_MEMBER_KICK",

	"INVITE_CREATE",
	"INVITE_DELETE",

	"MESSAGE_UPDATE",
	"MESSAGE_DELETE",
	"MESSAGE_DELETE_BULK",
}

// EventDescs is a description of events used in command options
var EventDescs map[string]string = map[string]string{
	"GUILD_UPDATE":        "Guild Update: changes to the server",
	"GUILD_EMOJIS_UPDATE": "Guild Emojis Update: changes to custom emotes",

	"GUILD_ROLE_CREATE": "Guild Role Create: roles being created",
	"GUILD_ROLE_UPDATE": "Guild Role Update: changes to roles",
	"GUILD_ROLE_DELETE": "Guild Role Delete: roles being deleted",

	"CHANNEL_CREATE": "Channel Create: channels being created",
	"CHANNEL_UPDATE": "Channel Update: changes to channels",
	"CHANNEL_DELETE": "Channel Delete: channels being deleted",

	"GUILD_MEMBER_ADD":         "Guild Member Add: members joining",
	"GUILD_MEMBER_UPDATE":      "Guild Member Update: roles being added/removed from members",
	"GUILD_MEMBER_NICK_UPDATE": "Guild Member Nick Update: avatar and nickname changes",
	"GUILD_KEY_ROLE_UPDATE":    "Key Role Update: key roles added/removed to users",
	"GUILD_MEMBER_REMOVE":      "Guild Member Remove: members leaving",
	"GUILD_MEMBER_KICK":        "Guild Member Kick: members being kicked",

	"GUILD_BAN_ADD":    "Guild Ban Add: members being banned",
	"GUILD_BAN_REMOVE": "Guild Ban Remove: members being unbanned",

	"INVITE_CREATE": "Invite Create: invites being created",
	"INVITE_DELETE": "Invite Delete: invites being deleted",

	"MESSAGE_UPDATE":      "Message Update: edited messages",
	"MESSAGE_DELETE":      "Message Delete: deleted messages",
	"MESSAGE_DELETE_BULK": "Message Delete Bulk: bulk deleted messages",
}

// EventMap is a map of event names → log channels
type EventMap map[string]discord.ChannelID

// DefaultEventMap is an event map with all events set to 0
var DefaultEventMap EventMap = EventMap{
	"GUILD_UPDATE":        0,
	"GUILD_EMOJIS_UPDATE": 0,

	"GUILD_ROLE_CREATE": 0,
	"GUILD_ROLE_UPDATE": 0,
	"GUILD_ROLE_DELETE": 0,

	"CHANNEL_CREATE": 0,
	"CHANNEL_UPDATE": 0,
	"CHANNEL_DELETE": 0,

	"GUILD_MEMBER_ADD":         0,
	"GUILD_MEMBER_UPDATE":      0,
	"GUILD_MEMBER_NICK_UPDATE": 0,
	"GUILD_KEY_ROLE_UPDATE":    0,
	"GUILD_MEMBER_REMOVE":      0,

	"GUILD_BAN_ADD":     0,
	"GUILD_BAN_REMOVE":  0,
	"GUILD_MEMBER_KICK": 0,

	"INVITE_CREATE": 0,
	"INVITE_DELETE": 0,

	"MESSAGE_UPDATE":      0,
	"MESSAGE_DELETE":      0,
	"MESSAGE_DELETE_BULK": 0,
}

// Channels gets the server's event:channel map
func (db *DB) Channels(id discord.GuildID) (ch EventMap, err error) {
	sql, args, err := sq.Select("channels").From("guilds").Where(squirrel.Eq{"id": id}).ToSql()
	if err != nil {
		return nil, err
	}

	err = db.QueryRow(context.Background(), sql, args...).Scan(&ch)
	return
}

// SetChannels ...
func (db *DB) SetChannels(id discord.GuildID, ch EventMap) (err error) {
	sql, args, err := sq.Update("guilds").Set("channels", ch).Where(squirrel.Eq{"id": id}).ToSql()
	if err != nil {
		return err
	}

	_, err = db.Exec(context.Background(), sql, args...)
	return
}

// RedirectMap is a map of origin channels → redirect channels.
// Key is string type so it can get encoded into JSON to store in the database
type RedirectMap = map[string]discord.ChannelID

// DefaultRedirectMap is an empty redirect map
var DefaultRedirectMap = RedirectMap{}

// Redirects gets the server's channel:channel map
func (db *DB) Redirects(id discord.GuildID) (m RedirectMap, err error) {
	sql, args, err := sq.Select("redirects").From("guilds").Where(squirrel.Eq{"id": id}).ToSql()
	if err != nil {
		return nil, err
	}

	err = db.QueryRow(context.Background(), sql, args...).Scan(&m)
	return
}

// SetRedirects sets the server's channel:channel map
func (db *DB) SetRedirects(id discord.GuildID, m RedirectMap) (err error) {
	sql, args, err := sq.Update("guilds").Set("redirects", m).Where(squirrel.Eq{"id": id}).ToSql()
	if err != nil {
		return err
	}

	_, err = db.Exec(context.Background(), sql, args...)
	return
}

func (db *DB) IsBlacklisted(guildID discord.GuildID, channelID discord.ChannelID) (blacklisted bool) {
	err := db.QueryRow(context.Background(), "select exists(select id from guilds where $1 = any(ignored_channels) and id = $2)", channelID, guildID).Scan(&blacklisted)
	if err != nil {
		common.Log.Errorf("Error checking if channel is blacklisted: %v", err)
	}
	return blacklisted
}
