package bot

import (
	"context"
	"sync"

	"emperror.dev/errors"
	"github.com/diamondburned/arikawa/v3/api/webhook"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/session/shard"
	"github.com/diamondburned/arikawa/v3/state"
	arikawastore "github.com/diamondburned/arikawa/v3/state/store"
	"github.com/diamondburned/arikawa/v3/utils/ws"
	"github.com/starshine-sys/bcr/v2"
	"github.com/starshine-sys/catalogger/v2/common/log"
	"github.com/starshine-sys/catalogger/v2/db"
	"github.com/starshine-sys/catalogger/v2/store"
	"github.com/starshine-sys/catalogger/v2/store/memory"
	"github.com/starshine-sys/catalogger/v2/store/redis"
)

const Intents = gateway.IntentGuildBans |
	gateway.IntentGuildEmojis |
	gateway.IntentGuildIntegrations |
	gateway.IntentGuildInvites |
	gateway.IntentGuildMembers |
	gateway.IntentGuildMessages |
	gateway.IntentGuildWebhooks |
	gateway.IntentGuilds

type Bot struct {
	Router *bcr.Router
	DB     *db.DB

	user   discord.User
	Config Config

	Cabinet store.Cabinet

	queues   map[discord.WebhookID]*queue
	queuesMu sync.Mutex

	webhookClients   map[discord.WebhookID]*webhook.Client
	webhookClientsMu sync.Mutex
}

// New creates a new Bot.
func New(c Config) (*Bot, error) {
	// set up debug logging
	ws.WSDebug = log.Debug
	ws.WSError = func(err error) {
		log.SugaredLogger.Error("ws error: ", err)
	}

	// set up the shard manager, including intents and stores
	mgr, err := shard.NewManager("Bot "+c.Auth.Discord, state.NewShardFunc(func(m *shard.Manager, s *state.State) {
		s.AddIntents(Intents)

		// clear all stores that we manage ourselves, as well as ones we don't use (message/presence store)
		s.Cabinet.ChannelStore = arikawastore.Noop
		s.Cabinet.GuildStore = arikawastore.Noop
		s.Cabinet.MemberStore = arikawastore.Noop
		s.Cabinet.MessageStore = arikawastore.Noop
		s.Cabinet.PresenceStore = arikawastore.Noop
		s.Cabinet.RoleStore = arikawastore.Noop
	}))
	if err != nil {
		return nil, errors.Wrap(err, "creating shard manager")
	}

	// set up interaction router + bot
	bot := &Bot{
		Config: c,
		Router: bcr.NewFromShardManager("Bot "+c.Auth.Discord, mgr),
	}

	// setup database
	bot.DB, err = db.New(c.Auth.Postgres, c.Auth.Redis)
	if err != nil {
		return nil, errors.Wrap(err, "creating database")
	}

	// create stores
	memoryStore := memory.New()
	redisStore, err := redis.New(c.Auth.Redis)
	if err != nil {
		return nil, errors.Wrap(err, "creating redis store")
	}

	// add self user cache handler
	mgr.Shard(0).(*state.State).AddHandler(bot.ready)

	// create cabinet
	// TODO: make redis optional
	bot.Cabinet = store.Cabinet{
		MemberStore:  redisStore,
		ChannelStore: memoryStore,
		GuildStore:   memoryStore,
		RoleStore:    memoryStore,
	}

	return bot, nil
}

func (bot *Bot) Open(ctx context.Context) error {
	log.Debug("opening gateway connection")

	return bot.Router.ShardManager.Open(ctx)
}

func (bot *Bot) Close() error {
	return bot.Router.ShardManager.Close()
}

// AddHandler adds handlers to all states.
func (bot *Bot) AddHandler(i ...any) {
	bot.Router.ShardManager.ForEach(func(shard shard.Shard) {
		s := shard.(*state.State)
		for _, hn := range i {
			s.AddHandler(hn)
		}
	})
}

func (bot *Bot) StateFromGuildID(guildID discord.GuildID) (s *state.State, id int) {
	shard, id := bot.Router.ShardManager.FromGuildID(guildID)
	return shard.(*state.State), id
}

// ready sets the bot user for webhook purposes
func (bot *Bot) ready(ev *gateway.ReadyEvent) {
	if ev.Shard == nil || ev.Shard.ShardID() != 0 {
		return
	}
	bot.user = ev.User
}
