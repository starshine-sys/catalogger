package events

import (
	"context"
	"errors"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/v3/api/webhook"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/session/shard"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/bot"
	"github.com/starshine-sys/catalogger/common"
	"github.com/starshine-sys/catalogger/events/eventcollector"
	"github.com/starshine-sys/catalogger/events/handler"
)

// delete messages after this many days have passed
const deleteAfterDays = 15

// Bot ...
type Bot struct {
	*bot.Bot

	SentryEnricher *handler.SentryHandler

	ProxiedTriggers   map[discord.MessageID]struct{}
	ProxiedTriggersMu sync.Mutex

	HandledMessages   map[discord.MessageID]struct{}
	HandledMessagesMu sync.Mutex

	UnauthorizedErrorsSent map[discord.ChannelID]time.Time
	UnauthorizedErrorsMu   sync.Mutex

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

	guildCount, roleCount, channelCount, msgCount int64

	guildsToChunk, guildsToFetchInvites map[discord.GuildID]struct{}
	chunkMu                             sync.RWMutex
	doneChunking                        bool

	// bot stats
	client     *http.Client
	topGGToken string

	messageRetentionDays int
}

// Init ...
func Init(bot *bot.Bot) (clearCacheFunc func(discord.GuildID, ...discord.ChannelID), memberFunc func() int64, guildPermFunc func(discord.GuildID, discord.UserID) (discord.Guild, discord.Permissions, error), joinedFunc func(discord.GuildID) bool) {
	joinLeaveLog, _ := discord.ParseSnowflake(os.Getenv("JOIN_LEAVE_LOG"))

	b := &Bot{
		Bot:            bot,
		Start:          time.Now().UTC(),
		SentryEnricher: eventcollector.New(),

		ProxiedTriggers:        make(map[discord.MessageID]struct{}),
		HandledMessages:        make(map[discord.MessageID]struct{}),
		UnauthorizedErrorsSent: make(map[discord.ChannelID]time.Time),

		Channels:             make(map[discord.ChannelID]discord.Channel),
		Roles:                make(map[discord.RoleID]discord.Role),
		Guilds:               make(map[discord.GuildID]discord.Guild),
		Queues:               make(map[discord.WebhookID]*Queue),
		WebhookClients:       make(map[discord.WebhookID]*webhook.Client),
		guildsToChunk:        make(map[discord.GuildID]struct{}),
		guildsToFetchInvites: make(map[discord.GuildID]struct{}),

		BotJoinLeaveLog: discord.ChannelID(joinLeaveLog),

		client:     &http.Client{},
		topGGToken: os.Getenv("TOPGG_TOKEN"),
	}

	i, err := strconv.Atoi(os.Getenv("MESSAGE_RETENTION_DAYS"))
	if err != nil {
		b.messageRetentionDays = 15
	} else {
		if i <= 0 {
			common.Log.Warnf("MESSAGE_RETENTION_DAYS must be a positive number, got %d; falling back to 15 days", i)
			b.messageRetentionDays = 15
		} else if i > 30 {
			common.Log.Fatalf("MESSAGE_RETENTION_DAYS cannot be over 30, got %d", i)
		} else {
			b.messageRetentionDays = i
		}
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
	evh.AddHandler(b.memberKick)

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
		return -1
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
		ct, err := bot.DB.Exec(context.Background(), "delete from messages where msg_id < $1", discord.NewSnowflake(time.Now().UTC().Add(time.Duration(bot.messageRetentionDays)*-24*time.Hour)))
		if err != nil {
			time.Sleep(1 * time.Minute)
			continue
		}

		if ct.RowsAffected() == 0 {
			common.Log.Debugf("Deleted 0 messages older than %v days.", deleteAfterDays)
		} else {
			common.Log.Infof("Deleted %v messages older than %v days.", ct.RowsAffected(), deleteAfterDays)
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

	m, err := bot.Member(guildID, userID)
	if err != nil {
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
		perms = perms.Add(discord.PermissionAll)
	}

	return g, perms, nil
}

// handleError handles any errors in event handlers
func (bot *Bot) handleError(ev reflect.Value, err error) {
	evName := ev.Elem().Type().Name()

	common.Log.Errorf("Error in %v: %v", evName, err)

	if bot.DB.Hub == nil {
		return
	}

	hub := bot.DB.Hub.Clone()

	bot.SentryEnricher.Handle(hub, ev.Interface())

	hub.CaptureException(err)
}
