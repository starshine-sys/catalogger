package db

import (
	"context"

	"github.com/diamondburned/arikawa/v3/discord"
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
	"GUILD_MEMBER_REMOVE",

	"GUILD_BAN_ADD",
	"GUILD_BAN_REMOVE",

	"INVITE_CREATE",
	"INVITE_DELETE",

	"MESSAGE_UPDATE",
	"MESSAGE_DELETE",
	"MESSAGE_DELETE_BULK",
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
	"GUILD_MEMBER_REMOVE":      0,

	"GUILD_BAN_ADD":    0,
	"GUILD_BAN_REMOVE": 0,

	"INVITE_CREATE": 0,
	"INVITE_DELETE": 0,

	"MESSAGE_UPDATE":      0,
	"MESSAGE_DELETE":      0,
	"MESSAGE_DELETE_BULK": 0,
}

// Channels gets the server's event:channel map
func (db *DB) Channels(id discord.GuildID) (ch EventMap, err error) {
	err = db.Pool.QueryRow(context.Background(), "select channels from guilds where id = $1", id).Scan(&ch)
	return
}

// SetChannels ...
func (db *DB) SetChannels(id discord.GuildID, ch EventMap) (err error) {
	_, err = db.Pool.Exec(context.Background(), "update guilds set channels = $1 where id = $2", ch, id)
	return
}

// RedirectMap is a map of origin channels → redirect channels.
// Key is string type so it can get encoded into JSON to store in the database
type RedirectMap = map[string]discord.ChannelID

// DefaultRedirectMap is an empty redirect map
var DefaultRedirectMap = RedirectMap{}

// Redirects gets the server's channel:channel map
func (db *DB) Redirects(id discord.GuildID) (m RedirectMap, err error) {
	err = db.Pool.QueryRow(context.Background(), "select redirects from guilds where id = $1", id).Scan(&m)
	return
}

// SetRedirects sets the server's channel:channel map
func (db *DB) SetRedirects(id discord.GuildID, m RedirectMap) (err error) {
	_, err = db.Pool.Exec(context.Background(), "update guilds set redirects = $1 where id = $2", m, id)
	return
}
