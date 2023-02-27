package bot

import (
	"context"
	"fmt"
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
	"github.com/starshine-sys/pkgo/v2"
)

const Intents = gateway.IntentGuildModeration |
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
	PK     *pkgo.Session

	user   discord.User
	Config Config

	Cabinet store.Cabinet

	queues   map[discord.WebhookID]*queue
	queuesMu sync.Mutex

	webhookClients   map[discord.WebhookID]*webhook.Client
	webhookClientsMu sync.Mutex

	users   map[discord.UserID]*discord.User
	usersMu sync.Mutex
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
		PK:     pkgo.New(""),

		queues:         map[discord.WebhookID]*queue{},
		webhookClients: map[discord.WebhookID]*webhook.Client{},
		users:          map[discord.UserID]*discord.User{},
	}

	if c.Bot.NoAutoMigrate {
		log.Warnf("Not running migrations automatically. Please run `catalogger migrate` before starting the bot.")
	}

	// setup database
	bot.DB, err = db.New(c.Auth.Postgres, c.Auth.Redis, c.Bot.AESKey, !c.Bot.NoAutoMigrate)
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

	// add interaction handler
	bot.AddHandler(bot.interactionCreate)

	// set up pkgo user agent
	// TODO: get version from git tags after full release
	bot.PK.UserAgent = fmt.Sprintf("Catalogger/v2 (+https://github.com/starshine-sys/catalogger; %v)", bot.Config.Bot.Owner)

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

func (bot *Bot) Me() discord.User { return bot.user }

// ready sets the bot user for webhook purposes
func (bot *Bot) ready(ev *gateway.ReadyEvent) {
	if ev.Shard == nil || ev.Shard.ShardID() != 0 {
		return
	}
	bot.user = ev.User
}
