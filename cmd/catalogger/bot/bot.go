package bot

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"emperror.dev/errors"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/utils/ws"
	"github.com/getsentry/sentry-go"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/bot"
	"github.com/starshine-sys/catalogger/commands"
	"github.com/starshine-sys/catalogger/common"
	"github.com/starshine-sys/catalogger/db"
	"github.com/starshine-sys/catalogger/events"
	"github.com/starshine-sys/catalogger/web/server"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

var Command = &cli.Command{
	Name:   "bot",
	Usage:  "Run the bot",
	Action: run,
}

func run(*cli.Context) (err error) {
	desugared := common.Log.Desugar()

	ws.WSDebug = common.Log.Named("ws").Debug
	ws.WSError = func(err error) {
		desugared.WithOptions(zap.AddCallerSkip(1)).Sugar().Error(err)
	}

	// set up logger for this section
	log := common.Log.Named("init")

	intents := gateway.IntentGuilds | gateway.IntentGuildMembers |
		gateway.IntentGuildBans | gateway.IntentGuildEmojis |
		gateway.IntentGuildIntegrations | gateway.IntentGuildWebhooks |
		gateway.IntentGuildInvites | gateway.IntentDirectMessageReactions |
		gateway.IntentGuildMessages | gateway.IntentGuildMessageReactions |
		gateway.IntentDirectMessages

	sf, _ := discord.ParseSnowflake(os.Getenv("OWNER"))

	// create a new router
	r, err := bcr.NewWithIntents(
		os.Getenv("TOKEN"),
		[]discord.UserID{discord.UserID(sf)},
		strings.Split(os.Getenv("PREFIXES"), ","),
		intents,
	)
	if err != nil {
		return errors.Wrap(err, "creating router")
	}
	r.EmbedColor = bcr.ColourPurple

	// sentry, if enabled
	var hub *sentry.Hub
	if os.Getenv("SENTRY_URL") != "" {
		err = sentry.Init(sentry.ClientOptions{
			Dsn: os.Getenv("SENTRY_URL"),
		})
		if err != nil {
			return errors.Wrap(err, "initing Sentry")
		}
		hub = sentry.CurrentHub()
	}

	// create a database connection
	db, err := db.New(os.Getenv("DATABASE_URL"), hub)
	if err != nil {
		return errors.Wrap(err, "opening database connection")
	}
	log.Infof("Opened database connection.")

	if db.Stats != nil {
		r.AddHandler(db.Stats.EventHandler)
	}

	// add message create + interaction create handler
	b, err := bot.New(os.Getenv("REDIS"), r, db)
	if err != nil {
		return errors.Wrap(err, "connecting to Redis")
	}

	// actually load events + commands
	commands.Init(b)

	cacheFunc, countFunc, guildPermFunc, joinedFunc := events.Init(b)
	server.NewServer(r, db, cacheFunc, countFunc, guildPermFunc, joinedFunc)

	// get current user
	s, _ := r.StateFromGuildID(0)
	botUser, err := s.Me()
	if err != nil {
		return errors.Wrap(err, "fetching bot user")
	}
	r.Bot = botUser

	// connect to discord
	if err := r.ShardManager.Open(context.Background()); err != nil {
		return errors.Wrap(err, "connect to Discord")
	}

	// Defer this to make sure that things are always cleanly shutdown even in the event of a crash
	defer func() {
		// set a status message
		// we're not actually properly closing the gateway so it'll stay for a few minutes
		// who needs a clean disconnection anyway :~]
		b.ForEach(func(s *state.State) {
			_ = s.Gateway().Send(context.Background(), &gateway.UpdatePresenceCommand{
				Status: discord.DoNotDisturbStatus,
				Activities: []discord.Activity{{
					Name: "Restarting, please wait...",
					Type: discord.GameActivity,
				}},
			})
		})

		db.Pool.Close()
		log.Info("Closed database connection.")
	}()

	log.Info("Connected to Discord. Press Ctrl-C or send an interrupt signal to stop.")

	log.Infof("User: %v (%v)", botUser.Tag(), botUser.ID)
	// normally creating a Context would do this, but as we set the user above, that doesn't work
	r.Prefixes = append(r.Prefixes, "<@"+r.Bot.ID.String()+">", "<@!"+r.Bot.ID.String()+">")

	// sync slash commands *if needed*
	sync := !strings.EqualFold(os.Getenv("SYNC_COMMANDS"), "false")
	guildID, _ := discord.ParseSnowflake(os.Getenv("COMMANDS_GUILD_ID"))
	if sync {
		if guildID == 0 {
			err = r.SyncCommands()
		} else {
			err = r.SyncCommands(discord.GuildID(guildID))
		}
		if err != nil {
			log.Errorf("Error syncing slash commands: %v", err)
		} else {
			s := "Synced slash commands"
			if guildID.IsValid() {
				s += " in " + fmt.Sprint(guildID)
			}
			log.Infof(s)
		}
	} else {
		log.Infof("Note: not syncing slash commands. Set SYNC_COMMANDS to true to sync commands")
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	log.Infof("Interrupt signal received. Shutting down...")
	return nil
}
