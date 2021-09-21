package events

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ReneKroon/ttlcache/v2"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/gateway/shard"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/utils/handler"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/bot"
)

// delete messages after this many days have passed
const deleteAfterDays = 15

// Bot ...
type Bot struct {
	*bot.Bot

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

	Queues  map[discord.ChannelID]*Queue
	QueueMu sync.Mutex
}

// Init ...
func Init(bot *bot.Bot) (clearCacheFunc func(discord.GuildID, ...discord.ChannelID), memberFunc func() int64, guildPermFunc func(discord.GuildID, discord.UserID) (discord.Guild, discord.Permissions, error), joinedFunc func(discord.GuildID) bool) {
	joinLeaveLog, _ := discord.ParseSnowflake(os.Getenv("JOIN_LEAVE_LOG"))

	b := &Bot{
		Bot:   bot,
		Start: time.Now().UTC(),

		ProxiedTriggers: map[discord.MessageID]struct{}{},
		BotMessages:     map[discord.MessageID]struct{}{},

		Invites:  map[discord.GuildID][]discord.Invite{},
		Members:  map[memberCacheKey]discord.Member{},
		Channels: map[discord.ChannelID]discord.Channel{},
		Roles:    map[discord.RoleID]discord.Role{},
		Guilds:   map[discord.GuildID]discord.Guild{},
		Queues:   map[discord.ChannelID]*Queue{},

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
	b.Router.AddHandler(b.requestGuildMembers)
	b.Router.AddHandler(b.guildMemberChunk)

	// add join/leave log handlers
	b.Router.ShardManager.ForEach(func(s shard.Shard) {
		state := s.(*state.State)

		state.PreHandler = handler.New()
		state.PreHandler.Synchronous = true
		state.PreHandler.AddHandler(b.guildDelete)
		state.AddHandler(b.guildCreate)
	})

	// add guild create handler
	b.Router.AddHandler(b.DB.CreateServerIfNotExists)

	// add pluralkit message create handlers
	b.Router.AddHandler(b.pkMessageCreate)
	b.Router.AddHandler(b.pkMessageCreateFallback)

	// add message create/update/delete handlers
	b.Router.AddHandler(b.messageCreate)
	b.Router.AddHandler(b.messageUpdate)
	b.Router.AddHandler(b.messageDelete)
	b.Router.AddHandler(b.bulkMessageDelete)

	// add guild member handlers
	b.Router.AddHandler(b.guildMemberAdd)
	b.Router.AddHandler(b.guildMemberUpdate)
	b.Router.AddHandler(b.guildMemberRemove)

	// add invite handlers
	b.Router.AddHandler(b.invitesReady)
	b.Router.AddHandler(b.inviteCreate)
	b.Router.AddHandler(b.inviteDelete)

	// add invite create/delete handlers
	b.Router.AddHandler(b.inviteCreateEvent)
	b.Router.AddHandler(b.inviteDeleteEvent)

	// add ban handlers
	b.Router.AddHandler(b.guildBanAdd)
	b.Router.AddHandler(b.guildBanRemove)

	// add channel handlers
	b.Router.AddHandler(b.channelCreate)
	b.Router.AddHandler(b.channelUpdate)
	b.Router.AddHandler(b.channelDelete)

	// add role handlers
	b.Router.AddHandler(b.guildRoleCreate)
	b.Router.AddHandler(b.guildRoleUpdate)
	b.Router.AddHandler(b.guildRoleDelete)

	// add guild handlers
	b.Router.AddHandler(b.guildUpdate)

	// add webhook update handler
	b.Router.AddHandler(b.webhooksUpdate)

	// add clear cache command
	b.Router.AddCommand(&bcr.Command{
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

	b.Router.AddCommand(&bcr.Command{
		Name:    "stats",
		Aliases: []string{"ping"},
		Summary: "Show the bot's latency and other stats.",

		Command: b.ping,
	})

	go b.cleanMessages()

	b.Router.ShardManager.ForEach(func(s shard.Shard) {
		state := s.(*state.State)

		var o sync.Once
		state.AddHandler(func(*gateway.ReadyEvent) {
			o.Do(func() {
				go b.updateStatusLoop(state)
			})
		})
	})

	clearCacheFunc = b.ResetCache
	memberFunc = func() int64 {
		b.MembersMu.Lock()
		n := int64(len(b.Members))
		b.MembersMu.Unlock()
		return n
	}
	guildPermFunc = b.guildPerms
	joinedFunc = func(id discord.GuildID) bool {
		b.GuildsMu.Lock()
		_, ok := b.Guilds[id]
		b.GuildsMu.Unlock()
		return ok
	}

	return clearCacheFunc, memberFunc, guildPermFunc, joinedFunc
}

// State gets a state.State for the guild
func (bot *Bot) State(id discord.GuildID) *state.State {
	s, _ := bot.Router.StateFromGuildID(id)
	return s
}

func (bot *Bot) cleanMessages() {
	for {
		ct, err := bot.DB.Pool.Exec(context.Background(), "delete from messages where msg_id < $1", discord.NewSnowflake(time.Now().UTC().Add(15*-24*time.Hour)))
		if err != nil {
			time.Sleep(1 * time.Minute)
			continue
		}

		if n := ct.RowsAffected(); n == 0 {
			bot.Sugar.Debugf("Deleted 0 normal messages older than %v days.", deleteAfterDays)
		} else {
			bot.Sugar.Infof("Deleted %v normal messages older than %v days.", n, deleteAfterDays)
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

	s, _ := bot.Router.StateFromGuildID(guildID)
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
			}
		}
	}

	return g, perms, nil
}

func (bot *Bot) updateStatusLoop(s *state.State) {
	time.Sleep(5 * time.Second)

	for {
		guildCount := 0
		bot.Router.ShardManager.ForEach(func(s shard.Shard) {
			state := s.(*state.State)

			guilds, _ := state.GuildStore.Guilds()
			guildCount += len(guilds)
		})

		shardNumber := 0
		bot.Router.ShardManager.ForEach(func(s shard.Shard) {
			state := s.(*state.State)

			str := fmt.Sprintf("%vhelp", strings.Split(os.Getenv("PREFIXES"), ",")[0])
			if guildCount != 0 {
				str += fmt.Sprintf(" | in %v servers", guildCount)
			}

			i := shardNumber
			shardNumber++

			go func() {
				i := i
				bot.Sugar.Infof("Setting status for shard #%v", i)
				s := str
				if bot.Router.ShardManager.NumShards() > 1 {
					s = fmt.Sprintf("%v | shard #%v", s, i)
				}

				err := state.UpdateStatus(gateway.UpdateStatusData{
					Status: discord.OnlineStatus,
					Activities: []discord.Activity{{
						Name: s,
						Type: discord.GameActivity,
					}},
				})
				if err != nil {
					bot.Sugar.Errorf("Error setting status for shard #%v: %v", i, err)
				}
			}()
		})

		time.Sleep(10 * time.Minute)
	}
}
