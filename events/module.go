package events

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/ReneKroon/ttlcache/v2"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway/shard"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/utils/handler"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/db"
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

	Guilds   map[discord.GuildID]discord.Guild
	GuildsMu sync.Mutex

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

	RedirectCache *ttlcache.Cache

	BotJoinLeaveLog discord.ChannelID

	Start time.Time
}

// Init ...
func Init(r *bcr.Router, db *db.DB, s *zap.SugaredLogger) (clearCacheFunc func(discord.GuildID, ...discord.ChannelID), memberFunc func() int64, guildPermFunc func(discord.GuildID, discord.UserID) (discord.Guild, discord.Permissions, error)) {
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
		Guilds:   map[discord.GuildID]discord.Guild{},

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

		RedirectCache: ttlcache.NewCache(),

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
	b.RedirectCache.SetTTL(10 * time.Minute)

	// add member cache handlers
	b.AddHandler(b.requestGuildMembers)
	b.AddHandler(b.guildMemberChunk)

	// add join/leave log handlers
	b.Router.ShardManager.ForEach(func(s shard.Shard) {
		state := s.(*state.State)

		state.PreHandler = handler.New()
		state.PreHandler.Synchronous = true
		state.PreHandler.AddHandler(b.guildDelete)
		state.AddHandler(b.guildCreate)
	})

	// add guild create handler
	b.AddHandler(b.DB.CreateServerIfNotExists)

	// add pluralkit message create/delete handlers
	b.AddHandler(b.pkMessageCreate)
	b.AddHandler(b.pkMessageCreateFallback)
	b.AddHandler(b.pkMessageDelete)

	// add message create/update/delete handlers
	b.AddHandler(b.messageCreate)
	b.AddHandler(b.messageUpdate)
	b.AddHandler(b.messageDelete)
	b.AddHandler(b.bulkMessageDelete)

	// add guild member handlers
	b.AddHandler(b.guildMemberAdd)
	b.AddHandler(b.guildMemberUpdate)
	b.AddHandler(b.guildMemberRemove)

	// add invite handlers
	b.AddHandler(b.invitesReady)
	b.AddHandler(b.inviteCreate)
	b.AddHandler(b.inviteDelete)

	// add invite create/delete handlers
	b.AddHandler(b.inviteCreateEvent)
	b.AddHandler(b.inviteDeleteEvent)

	// add ban handlers
	b.AddHandler(b.guildBanAdd)
	b.AddHandler(b.guildBanRemove)

	// add channel handlers
	b.AddHandler(b.channelCreate)
	b.AddHandler(b.channelUpdate)
	b.AddHandler(b.channelDelete)

	// add role handlers
	b.AddHandler(b.guildRoleCreate)
	b.AddHandler(b.guildRoleUpdate)
	b.AddHandler(b.guildRoleDelete)

	// add guild handlers
	b.AddHandler(b.guildUpdate)

	// add webhook update handler
	b.AddHandler(b.webhooksUpdate)

	// add clear cache command
	b.AddCommand(&bcr.Command{
		Name:    "clear-cache",
		Aliases: []string{"clearcache"},
		Summary: "Clear this server's webhook cache.",

		Permissions: discord.PermissionManageGuild,
		Command: func(ctx *bcr.Context) (err error) {
			channels, err := ctx.State.Channels(ctx.Message.GuildID)
			if err != nil {
				return b.DB.ReportCtx(ctx, err)
			}
			ch := []discord.ChannelID{}
			for _, c := range channels {
				ch = append(ch, c.ID)
			}

			b.ResetCache(ctx.Message.GuildID, ch...)
			_, err = ctx.Send("Reset the webhook cache for this server.")
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

	clearCacheFunc = b.ResetCache
	memberFunc = func() int64 {
		b.MembersMu.Lock()
		n := int64(len(b.Members))
		b.MembersMu.Unlock()
		return n
	}
	guildPermFunc = b.guildPerms

	return clearCacheFunc, memberFunc, guildPermFunc
}

// State gets a state.State for the guild
func (bot *Bot) State(id discord.GuildID) *state.State {
	s, _ := bot.StateFromGuildID(id)
	return s
}

func (bot *Bot) cleanMessages() {
	for {
		c, err := bot.DB.Pool.Exec(context.Background(), "delete from pk_messages where msg_id < $1", discord.NewSnowflake(time.Now().UTC().Add(-720*time.Hour)))
		if err != nil {
			time.Sleep(1 * time.Minute)
			continue
		}

		if n := c.RowsAffected(); n == 0 {
			bot.Sugar.Debugf("Deleted 0 PK messages older than 30 days.")
		} else {
			bot.Sugar.Infof("Deleted %v PK messages older than 30 days.", n)
		}

		c, err = bot.DB.Pool.Exec(context.Background(), "delete from messages where msg_id < $1", discord.NewSnowflake(time.Now().UTC().Add(-720*time.Hour)))
		if err != nil {
			time.Sleep(1 * time.Minute)
			continue
		}

		if n := c.RowsAffected(); n == 0 {
			bot.Sugar.Debugf("Deleted 0 normal messages older than 30 days.")
		} else {
			bot.Sugar.Infof("Deleted %v normal messages older than 30 days.", n)
		}

		time.Sleep(1 * time.Minute)
	}
}

func (bot *Bot) guildPerms(guildID discord.GuildID, userID discord.UserID) (g discord.Guild, perms discord.Permissions, err error) {
	bot.GuildsMu.Lock()
	g, ok := bot.Guilds[guildID]
	bot.GuildsMu.Unlock()
	if !ok {
		return g, 0, errors.New("guild not found")
	}

	s, _ := bot.StateFromGuildID(guildID)
	g.Roles, err = s.Roles(guildID)
	if err != nil {
		return g, 0, err
	}

	bot.MembersMu.Lock()
	m, ok := bot.Members[memberCacheKey{guildID, userID}]
	bot.MembersMu.Unlock()
	if !ok {
		return g, 0, errors.New("member not found")
	}

	if g.OwnerID == userID {
		return g, discord.PermissionAll, nil
	}

	for _, r := range g.Roles {
		for _, id := range m.RoleIDs {
			if r.ID == id {
				if r.Permissions.Has(discord.PermissionAdministrator) {
					return g, discord.PermissionAll, nil
				}

				perms |= r.Permissions
				break
			}
		}
	}

	return g, perms, nil
}
