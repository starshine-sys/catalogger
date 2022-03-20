package bot

import (
	"context"
	"sync"

	"emperror.dev/errors"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/session/shard"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/state/store"
	"github.com/diamondburned/arikawa/v3/utils/httputil/httpdriver"
	"github.com/getsentry/sentry-go"
	"github.com/mediocregopher/radix/v4"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/bcr/bot"
	"github.com/starshine-sys/catalogger/common"
	"github.com/starshine-sys/catalogger/db"
	"github.com/starshine-sys/catalogger/db/stats"
	mstore "github.com/starshine-sys/catalogger/store"
	"github.com/starshine-sys/catalogger/store/redisstore"
	"gitlab.com/1f320/x/concurrent"
)

// Bot ...
type Bot struct {
	*bot.Bot

	DB          *db.DB
	Redis       radix.Client
	MemberStore mstore.Store

	Channels *concurrent.Map[discord.ChannelID, discord.Channel]
	Roles    *concurrent.Map[discord.RoleID, discord.Role]
	Guilds   *concurrent.Map[discord.GuildID, discord.Guild]

	userCache map[discord.UserID]discord.User
	userMu    sync.Mutex
}

// New ...
func New(redisURL string, r *bcr.Router, db *db.DB) (b *Bot, err error) {
	b = &Bot{
		Bot:       bot.NewWithRouter(r),
		DB:        db,
		userCache: map[discord.UserID]discord.User{},
		Channels:  concurrent.NewMap[discord.ChannelID, discord.Channel](),
		Roles:     concurrent.NewMap[discord.RoleID, discord.Role](),
		Guilds:    concurrent.NewMap[discord.GuildID, discord.Guild](),
	}

	b.Redis, err = (&radix.PoolConfig{}).New(context.Background(), "tcp", redisURL)
	if err != nil {
		return nil, err
	}
	common.Log.Info("Connected to Redis")

	mstore, err := redisstore.NewStore(redisURL)
	if err != nil {
		return nil, errors.Wrap(err, "creating member store")
	}
	b.MemberStore = mstore

	b.Router.AddHandler(b.messageCreate)
	b.Router.AddHandler(b.interactionCreate)
	b.Router.AddHandler(b.handleEventForCache)

	r.ShardManager.ForEach(func(s shard.Shard) {
		state := s.(*state.State)

		// log requests and their response codes
		state.Client.Client.OnResponse = append(state.Client.Client.OnResponse, b.onResponse)

		// these are never referenced in code and otherwise take up memory (not a whole lot, but hey)
		// for now, bcr.Context still uses GuildStore, ChannelStore, and RoleStore
		// TODO: replace r.NewContext with a custom method that uses the bot's own cache, will require a major refactor (as it's currently in events.Bot)
		state.Cabinet.MessageStore = store.Noop
		state.Cabinet.EmojiStore = store.Noop
		state.Cabinet.MemberStore = store.Noop
		state.Cabinet.VoiceStateStore = store.Noop
		state.Cabinet.PresenceStore = store.Noop
	})

	return b, nil
}

func (bot *Bot) onResponse(req httpdriver.Request, resp httpdriver.Response) error {
	method := ""

	v, ok := req.(*httpdriver.DefaultRequest)
	if ok {
		method = v.Method
		if method == "" {
			method = "GET"
		}
	}

	if resp == nil {
		return nil
	}

	if _, ok := resp.(*httpdriver.DefaultResponse); !ok {
		return nil
	}

	common.Log.Debugf("%v %v => %v", method, stats.LoggingName(req.GetPath()), resp.GetStatus())

	go bot.DB.Stats.IncRequests(method, req.GetPath(), resp.GetStatus())

	return nil
}

// ForEach runs the given function on each shard
func (bot *Bot) ForEach(fn func(s *state.State)) {
	bot.Router.ShardManager.ForEach(func(s shard.Shard) {
		state := s.(*state.State)
		fn(state)
	})
}

// MultiDo executes the given Actions in order, and returns the error of the first to return a non-nil error.
func (bot *Bot) MultiDo(ctx context.Context, actions ...radix.Action) error {
	for _, action := range actions {
		err := bot.Redis.Do(ctx, action)
		if err != nil {
			return err
		}
	}
	return nil
}

func (bot *Bot) messageCreate(m *gateway.MessageCreateEvent) {
	// if the author is a bot, return
	if m.Author.Bot {
		return
	}

	// if the message does not start with any of the bot's prefixes (including mentions), return
	if !bot.Router.MatchPrefix(m.Message) {
		return
	}

	// get the context
	ctx, err := bot.Router.NewContext(m)
	if err != nil {
		common.Log.Errorf("Error getting context: %v", err)
		return
	}

	defer func() {
		r := recover()
		if r != nil {
			common.Log.Errorf("Caught panic in channel ID %v (user %v, guild %v): %v", m.ChannelID, m.Author.ID, m.GuildID, r)
			common.Log.Infof("Panic message content:\n```\n%v\n```", m.Content)

			if ctx == nil {
				sentry.CurrentHub().Recover(r)
				return
			}

			id := sentry.CurrentHub().Recover(r)
			if err = bot.DB.ReportEmbed(ctx, id); err != nil {
				common.Log.Errorf("Error sending error message: %v", err)
			}
			return
		}
	}()

	err = bot.Router.Execute(ctx)
	if err != nil {
		common.Log.Errorf("Error executing command: %v", err)
		return
	}

	bot.DB.Stats.IncCommand()
}

func (bot *Bot) interactionCreate(ic *gateway.InteractionCreateEvent) {
	if ic.Data.InteractionType() != discord.CommandInteractionType {
		return
	}

	ctx, err := bot.Router.NewSlashContext(ic)
	if err != nil {
		common.Log.Errorf("Couldn't create slash context: %v", err)
		return
	}

	defer func() {
		r := recover()
		if r != nil {
			common.Log.Errorf("Caught panic in channel ID %v (user %v, guild %v): %v", ic.ChannelID, ctx.Author.ID, ctx.Channel.GuildID, r)
			common.Log.Infof("Command: %v", ctx.CommandName)

			if ctx == nil {
				sentry.CurrentHub().Recover(r)
				return
			}

			id := sentry.CurrentHub().Recover(r)
			if err = bot.DB.ReportEmbed(ctx, id); err != nil {
				common.Log.Errorf("Error sending error message: %v", err)
			}
			return
		}
	}()

	err = bot.Router.ExecuteSlash(ctx)
	if err != nil {
		common.Log.Errorf("Error executing slash command: %v", err)
	}

	bot.DB.Stats.IncCommand()
}
