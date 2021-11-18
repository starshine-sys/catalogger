package events

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/v3/api/webhook"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/gateway/shard"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/bot"
	"github.com/starshine-sys/catalogger/events/eventcollector"
	"github.com/starshine-sys/catalogger/events/handler"
	"go.uber.org/zap"
)

// delete messages after this many days have passed
const deleteAfterDays = 15

// Bot ...
type Bot struct {
	*bot.Bot
	Sugar *zap.SugaredLogger

	SentryEnricher *handler.SentryHandler

	ProxiedTriggers   map[discord.MessageID]struct{}
	ProxiedTriggersMu sync.Mutex

	HandledMessages   map[discord.MessageID]struct{}
	HandledMessagesMu sync.Mutex

	Invites  map[discord.GuildID][]discord.Invite
	InviteMu sync.Mutex

	Members   map[memberCacheKey]discord.Member
	MembersMu sync.RWMutex

	Channels   map[discord.ChannelID]discord.Channel
	ChannelsMu sync.RWMutex

	Roles   map[discord.RoleID]discord.Role
	RolesMu sync.Mutex

	Guilds   map[discord.GuildID]discord.Guild
	GuildsMu sync.RWMutex

	BotJoinLeaveLog discord.ChannelID

	Start time.Time

	Queues  map[discord.WebhookID]*Queue
	QueueMu sync.Mutex

	WebhookClients map[discord.WebhookID]*webhook.Client
	WebhooksMu     sync.Mutex

	EventHandler *handler.Handler

	guildCount, memberCount, roleCount, channelCount, msgCount int64

	guildsToChunk, guildsToFetchInvites map[discord.GuildID]struct{}
	chunkMu                             sync.RWMutex
	doneChunking                        bool
}

// Init ...
func Init(bot *bot.Bot, log *zap.SugaredLogger) (clearCacheFunc func(discord.GuildID, ...discord.ChannelID), memberFunc func() int64, guildPermFunc func(discord.GuildID, discord.UserID) (discord.Guild, discord.Permissions, error), joinedFunc func(discord.GuildID) bool) {
	joinLeaveLog, _ := discord.ParseSnowflake(os.Getenv("JOIN_LEAVE_LOG"))

	b := &Bot{
		Bot:            bot,
		Sugar:          log.Named("event"),
		Start:          time.Now().UTC(),
		SentryEnricher: eventcollector.New(),

		ProxiedTriggers: make(map[discord.MessageID]struct{}),
		HandledMessages: make(map[discord.MessageID]struct{}),

		Invites:              make(map[discord.GuildID][]discord.Invite),
		Members:              make(map[memberCacheKey]discord.Member),
		Channels:             make(map[discord.ChannelID]discord.Channel),
		Roles:                make(map[discord.RoleID]discord.Role),
		Guilds:               make(map[discord.GuildID]discord.Guild),
		Queues:               make(map[discord.WebhookID]*Queue),
		WebhookClients:       make(map[discord.WebhookID]*webhook.Client),
		guildsToChunk:        make(map[discord.GuildID]struct{}),
		guildsToFetchInvites: make(map[discord.GuildID]struct{}),

		BotJoinLeaveLog: discord.ChannelID(joinLeaveLog),
	}

	// either add counts to metrics collector, or spawn loop to collect stats every minute
	if b.DB.Stats != nil {
		b.DB.Stats.Counts = b.counts
	} else {
		go b.countsLoop()
	}

	// add default handlers
	// these don't actually log anything and just request/update info
	b.Router.AddHandler(b.requestGuildMembers)
	b.Router.AddHandler(b.guildMemberChunk)
	b.Router.AddHandler(b.chunkGuildDelete)
	b.Router.AddHandler(b.DB.CreateServerIfNotExists)
	b.Router.AddHandler(b.inviteCreate)
	b.Router.AddHandler(b.inviteDelete)
	b.Router.AddHandler(b.webhooksUpdate)

	evh := handler.New()
	b.EventHandler = evh
	evh.HandleResponse = b.handleResponse
	evh.HandleError = b.handleError

	b.Router.AddHandler(evh.Call)

	// add join/leave log handlers
	evh.AddHandler(b.guildCreate)
	evh.AddHandler(b.guildDelete)

	// add message create/update/delete handlers
	evh.AddHandler(b.messageCreate)
	evh.AddHandler(b.messageUpdate)
	evh.AddHandler(b.messageDelete)
	evh.AddHandler(b.bulkMessageDelete)

	// add guild member handlers
	evh.AddHandler(b.guildMemberAdd)
	evh.AddHandler(b.guildMemberUpdate)
	evh.AddHandler(b.keyroleUpdate)
	evh.AddHandler(b.guildMemberRemove)

	// add invite create/delete handlers
	evh.AddHandler(b.inviteCreateEvent)
	evh.AddHandler(b.inviteDeleteEvent)

	// add ban handlers
	evh.AddHandler(b.guildBanAdd)
	evh.AddHandler(b.guildBanRemove)

	// add channel handlers
	evh.AddHandler(b.channelCreate)
	evh.AddHandler(b.channelUpdate)
	evh.AddHandler(b.channelDelete)

	// add role handlers
	evh.AddHandler(b.guildRoleCreate)
	evh.AddHandler(b.guildRoleUpdate)
	evh.AddHandler(b.guildRoleDelete)

	// add guild handlers
	evh.AddHandler(b.guildUpdate)
	evh.AddHandler(b.emojiUpdate)

	// add clear cache command
	b.Router.AddCommand(&bcr.Command{
		Name:    "clear-cache",
		Aliases: []string{"clearcache"},
		Summary: "Clear this server's webhook cache.",

		Permissions: discord.PermissionManageGuild,
		Options:     &[]discord.CommandOption{},
		SlashCommand: func(ctx bcr.Contexter) (err error) {
			channels, err := ctx.Session().Channels(ctx.GetGuild().ID)
			if err != nil {
				return b.DB.ReportCtx(ctx, err)
			}
			ch := []discord.ChannelID{}
			for _, c := range channels {
				ch = append(ch, c.ID)
			}

			b.ResetCache(ctx.GetGuild().ID, ch...)
			return ctx.SendX("Reset the webhook cache for this server.")
		},
	})

	b.Router.AddCommand(&bcr.Command{
		Name:    "stats",
		Aliases: []string{"ping"},
		Summary: "Show the bot's latency and other stats.",

		SlashCommand: b.ping,
	})

	// mmmm spaghetti
	// [this isn't *that* spaghetti but i wish we'd added a better way to do this]
	for _, g := range bot.Router.SlashGroups {
		if g.Name == "help" {
			g.Add(&bcr.Command{
				Name:         "stats",
				Summary:      "Show the bot's latency and other stats.",
				SlashCommand: b.ping,
			})
		}
	}

	go b.cleanMessages()

	b.Router.ShardManager.ForEach(func(s shard.Shard) {
		state := s.(*state.State)

		var o sync.Once
		state.AddHandler(func(*gateway.ReadyEvent) {
			o.Do(func() {
				go b.updateStatusLoop(state)
				go b.chunkGuilds()
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
		ct, err := bot.DB.Exec(context.Background(), "delete from messages where msg_id < $1", discord.NewSnowflake(time.Now().UTC().Add(15*-24*time.Hour)))
		if err != nil {
			time.Sleep(1 * time.Minute)
			continue
		}

		if ct.RowsAffected() == 0 {
			bot.Sugar.Debugf("Deleted 0 messages older than %v days.", deleteAfterDays)
		} else {
			bot.Sugar.Infof("Deleted %v messages older than %v days.", ct.RowsAffected(), deleteAfterDays)
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

	for _, role := range g.Roles {
		for _, id := range m.RoleIDs {
			if id == role.ID {
				perms = perms.Add(role.Permissions)
			}
		}
	}

	if perms.Has(discord.PermissionAdministrator) {
		perms.Add(discord.PermissionAll)
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

		status := discord.IdleStatus
		if bot.doneChunking {
			status = discord.OnlineStatus
		} else {
			bot.Sugar.Infof("Not done chunking, setting idle status")
		}

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
					Status: status,
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

// handleError handles any errors in event handlers
func (bot *Bot) handleError(ev reflect.Value, err error) {
	evName := ev.Elem().Type().Name()

	bot.Sugar.Errorf("Error in %v: %v", evName, err)

	if bot.DB.Hub == nil {
		return
	}

	hub := bot.DB.Hub.Clone()

	bot.SentryEnricher.Handle(hub, ev.Interface())

	hub.CaptureException(err)
}
