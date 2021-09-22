package bot

import (
	"context"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/getsentry/sentry-go"
	"github.com/mediocregopher/radix/v4"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/bcr/bot"
	"github.com/starshine-sys/catalogger/db"
	"go.uber.org/zap"
)

// Bot ...
type Bot struct {
	*bot.Bot

	DB    *db.DB
	Redis radix.Client
	Sugar *zap.SugaredLogger
}

// New ...
func New(redisURL string, r *bcr.Router, db *db.DB, log *zap.SugaredLogger) (b *Bot, err error) {
	b = &Bot{
		Bot:   bot.NewWithRouter(r),
		DB:    db,
		Sugar: log,
	}

	b.Redis, err = (&radix.PoolConfig{}).New(context.Background(), "tcp", redisURL)
	if err != nil {
		return nil, err
	}
	log.Info("Connected to Redis")

	b.Router.AddHandler(b.messageCreate)
	b.Router.AddHandler(b.interactionCreate)

	return b, nil
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
		bot.Sugar.Errorf("Error getting context: %v", err)
		return
	}

	defer func() {
		r := recover()
		if r != nil {
			bot.Sugar.Errorf("Caught panic in channel ID %v (user %v, guild %v): %v", m.ChannelID, m.Author.ID, m.GuildID, r)
			bot.Sugar.Infof("Panic message content:\n```\n%v\n```", m.Content)

			if ctx == nil {
				sentry.CurrentHub().Recover(r)
				return
			}

			id := sentry.CurrentHub().Recover(r)
			bot.DB.ReportEmbed(ctx, id)
			return
		}
	}()

	err = bot.Router.Execute(ctx)
	if err != nil {
		bot.Sugar.Errorf("Error executing command: %v", err)
		return
	}
}

func (bot *Bot) interactionCreate(ic *gateway.InteractionCreateEvent) {
	if ic.Type != discord.CommandInteraction {
		return
	}

	ctx, err := bot.Router.NewSlashContext(ic)
	if err != nil {
		bot.Sugar.Errorf("Couldn't create slash context: %v", err)
		return
	}

	defer func() {
		r := recover()
		if r != nil {
			bot.Sugar.Errorf("Caught panic in channel ID %v (user %v, guild %v): %v", ic.ChannelID, ctx.Author.ID, ctx.Channel.GuildID, r)
			bot.Sugar.Infof("Command: %v", ctx.CommandName)

			if ctx == nil {
				sentry.CurrentHub().Recover(r)
				return
			}

			id := sentry.CurrentHub().Recover(r)
			bot.DB.ReportEmbed(ctx, id)
			return
		}
	}()

	err = bot.Router.ExecuteSlash(ctx)
	if err != nil {
		bot.Sugar.Errorf("Couldn't create slash context: %v", err)
	}
}
