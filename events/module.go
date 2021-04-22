package events

import (
	"context"
	"os"
	"sync"
	"time"

	"git.sr.ht/~starshine-sys/logger/db"
	"github.com/ReneKroon/ttlcache/v2"
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/utils/handler"
	"github.com/starshine-sys/bcr"
	"go.uber.org/zap"
)

// Bot ...
type Bot struct {
	*bcr.Router

	DB    *db.DB
	Sugar *zap.SugaredLogger

	ProxiedTriggers   map[discord.MessageID]struct{}
	ProxiedTriggersMu sync.Mutex

	BotMessages   map[discord.MessageID]struct{}
	BotMessagesMu sync.Mutex

	Invites  map[discord.GuildID][]discord.Invite
	InviteMu sync.Mutex

	Members   map[memberCacheKey]discord.Member
	MembersMu sync.Mutex

	Channels   map[discord.ChannelID]discord.Channel
	ChannelsMu sync.Mutex

	Roles   map[discord.RoleID]discord.Role
	RolesMu sync.Mutex

	MessageDeleteCache     *ttlcache.Cache
	MessageUpdateCache     *ttlcache.Cache
	MessageDeleteBulkCache *ttlcache.Cache

	InviteCreateCache *ttlcache.Cache
	InviteDeleteCache *ttlcache.Cache

	GuildBanAddCache    *ttlcache.Cache
	GuildBanRemoveCache *ttlcache.Cache

	GuildMemberAddCache    *ttlcache.Cache
	GuildMemberRemoveCache *ttlcache.Cache

	GuildMemberUpdateCache     *ttlcache.Cache
	GuildMemberNickUpdateCache *ttlcache.Cache

	ChannelCreateCache *ttlcache.Cache
	ChannelUpdateCache *ttlcache.Cache
	ChannelDeleteCache *ttlcache.Cache

	GuildUpdateCache       *ttlcache.Cache
	GuildEmojisUpdateCache *ttlcache.Cache
	GuildRoleCreateCache   *ttlcache.Cache
	GuildRoleDeleteCache   *ttlcache.Cache
	GuildRoleUpdateCache   *ttlcache.Cache

	BotJoinLeaveLog discord.ChannelID

	Start time.Time
}

// Init ...
func Init(r *bcr.Router, db *db.DB, s *zap.SugaredLogger) {
	joinLeaveLog, _ := discord.ParseSnowflake(os.Getenv("JOIN_LEAVE_LOG"))

	b := &Bot{
		Router: r,
		DB:     db,
		Sugar:  s,
		Start:  time.Now().UTC(),

		ProxiedTriggers: map[discord.MessageID]struct{}{},
		BotMessages:     map[discord.MessageID]struct{}{},

		Invites:  map[discord.GuildID][]discord.Invite{},
		Members:  map[memberCacheKey]discord.Member{},
		Channels: map[discord.ChannelID]discord.Channel{},
		Roles:    map[discord.RoleID]discord.Role{},

		MessageDeleteCache: ttlcache.NewCache(),
		MessageUpdateCache: ttlcache.NewCache(),

		GuildMemberAddCache: ttlcache.NewCache(),
		InviteCreateCache:   ttlcache.NewCache(),
		InviteDeleteCache:   ttlcache.NewCache(),
		GuildBanAddCache:    ttlcache.NewCache(),
		GuildBanRemoveCache: ttlcache.NewCache(),

		GuildMemberNickUpdateCache: ttlcache.NewCache(),

		GuildMemberRemoveCache: ttlcache.NewCache(),
		GuildMemberUpdateCache: ttlcache.NewCache(),

		MessageDeleteBulkCache: ttlcache.NewCache(),

		ChannelCreateCache: ttlcache.NewCache(),
		ChannelUpdateCache: ttlcache.NewCache(),
		ChannelDeleteCache: ttlcache.NewCache(),

		GuildUpdateCache:       ttlcache.NewCache(),
		GuildEmojisUpdateCache: ttlcache.NewCache(),
		GuildRoleCreateCache:   ttlcache.NewCache(),
		GuildRoleDeleteCache:   ttlcache.NewCache(),
		GuildRoleUpdateCache:   ttlcache.NewCache(),

		BotJoinLeaveLog: discord.ChannelID(joinLeaveLog),
	}
	b.MessageDeleteCache.SetTTL(10 * time.Minute)
	b.MessageUpdateCache.SetTTL(10 * time.Minute)
	b.GuildMemberAddCache.SetTTL(10 * time.Minute)
	b.InviteCreateCache.SetTTL(10 * time.Minute)
	b.InviteDeleteCache.SetTTL(10 * time.Minute)
	b.GuildBanAddCache.SetTTL(10 * time.Minute)
	b.GuildBanRemoveCache.SetTTL(10 * time.Minute)
	b.GuildMemberRemoveCache.SetTTL(10 * time.Minute)
	b.GuildMemberUpdateCache.SetTTL(10 * time.Minute)
	b.GuildMemberNickUpdateCache.SetTTL(10 * time.Minute)
	b.MessageDeleteBulkCache.SetTTL(10 * time.Minute)
	b.ChannelCreateCache.SetTTL(10 * time.Minute)
	b.ChannelUpdateCache.SetTTL(10 * time.Minute)
	b.ChannelDeleteCache.SetTTL(10 * time.Minute)
	b.GuildUpdateCache.SetTTL(10 * time.Minute)
	b.GuildEmojisUpdateCache.SetTTL(10 * time.Minute)
	b.GuildRoleCreateCache.SetTTL(10 * time.Minute)
	b.GuildRoleDeleteCache.SetTTL(10 * time.Minute)
	b.GuildRoleUpdateCache.SetTTL(10 * time.Minute)

	// add member cache handlers
	b.Router.State.AddHandler(b.requestGuildMembers)
	b.Router.State.AddHandler(b.guildMemberChunk)

	// add join/leave log handlers
	b.Router.State.PreHandler = handler.New()
	b.Router.State.PreHandler.Synchronous = true
	b.Router.State.PreHandler.AddHandler(b.guildDelete)
	b.Router.State.AddHandler(b.guildCreate)

	// add guild create handler
	b.State.AddHandler(b.DB.CreateServerIfNotExists)

	// add pluralkit message create/delete handlers
	b.State.AddHandler(b.pkMessageCreate)
	b.State.AddHandler(b.pkMessageCreateFallback)
	b.State.AddHandler(b.pkMessageDelete)

	// add message create/update/delete handlers
	b.State.AddHandler(b.messageCreate)
	b.State.AddHandler(b.messageUpdate)
	b.State.AddHandler(b.messageDelete)
	b.State.AddHandler(b.bulkMessageDelete)

	// add guild member handlers
	b.State.AddHandler(b.guildMemberAdd)
	b.State.AddHandler(b.guildMemberUpdate)
	b.State.AddHandler(b.guildMemberRemove)

	// add invite handlers
	b.State.AddHandler(b.invitesReady)
	b.State.AddHandler(b.inviteCreate)
	b.State.AddHandler(b.inviteDelete)

	// add invite create/delete handlers
	b.State.AddHandler(b.inviteCreateEvent)
	b.State.AddHandler(b.inviteDeleteEvent)

	// add ban handlers
	b.State.AddHandler(b.guildBanAdd)
	b.State.AddHandler(b.guildBanRemove)

	// add channel handlers
	b.State.AddHandler(b.channelCreate)
	b.State.AddHandler(b.channelUpdate)
	b.State.AddHandler(b.channelDelete)

	// add role handlers
	b.State.AddHandler(b.guildRoleCreate)
	b.State.AddHandler(b.guildRoleUpdate)
	b.State.AddHandler(b.guildRoleDelete)

	// add clear cache command
	b.AddCommand(&bcr.Command{
		Name:    "clear-cache",
		Aliases: []string{"clearcache"},
		Summary: "Clear this server's webhook cache.",

		Permissions: discord.PermissionManageGuild,
		Command: func(ctx *bcr.Context) (err error) {
			b.ResetCache(ctx.Message.GuildID)
			_, err = ctx.Send("Reset the webhook cache for this server.", nil)
			return
		},
	})

	b.AddCommand(&bcr.Command{
		Name:    "stats",
		Aliases: []string{"ping"},
		Summary: "Show the bot's latency and other stats.",

		Command: b.ping,
	})

	go b.cleanMessages()
}

func (bot *Bot) cleanMessages() {
	for {
		c, err := bot.DB.Pool.Exec(context.Background(), "delete from pk_messages where msg_id < $1", discord.NewSnowflake(time.Now().UTC().Add(-720*time.Hour)))
		if err != nil {
			time.Sleep(1 * time.Minute)
			continue
		}

		bot.Sugar.Debugf("Deleted %v PK messages older than 30 days.", c.RowsAffected())

		c, err = bot.DB.Pool.Exec(context.Background(), "delete from messages where msg_id < $1", discord.NewSnowflake(time.Now().UTC().Add(-720*time.Hour)))
		if err != nil {
			time.Sleep(1 * time.Minute)
			continue
		}

		bot.Sugar.Debugf("Deleted %v normal messages older than 30 days.", c.RowsAffected())

		time.Sleep(1 * time.Minute)
	}
}
