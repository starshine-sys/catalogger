package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/utils/wsutil"
	"github.com/getsentry/sentry-go"
	_ "github.com/joho/godotenv/autoload"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/bot"
	"github.com/starshine-sys/catalogger/commands"
	"github.com/starshine-sys/catalogger/db"
	"github.com/starshine-sys/catalogger/events"
	"github.com/starshine-sys/catalogger/logsetup"
	"github.com/starshine-sys/catalogger/web/server"
)

func main() {
	zap, err := logsetup.SetupLogging()
	if err != nil {
		panic(err)
	}
	sugar := zap.Sugar()

	wsutil.WSDebug = sugar.Named("ws").Debug
	wsutil.WSError = func(err error) {
		sugar.Named("ws").Error(err)
	}

	// set up logger for this section
	log := sugar.Named("init")

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
		log.Fatalf("Error creating bot: %v", err)
	}
	r.EmbedColor = bcr.ColourPurple

	// sentry, if enabled
	var hub *sentry.Hub
	if os.Getenv("SENTRY_URL") != "" {
		err = sentry.Init(sentry.ClientOptions{
			Dsn: os.Getenv("SENTRY_URL"),
		})
		if err != nil {
			log.Fatalf("Error initialising Sentry: %v", err)
		}
		hub = sentry.CurrentHub()
	}

	// create a database connection
	db, err := db.New(os.Getenv("DATABASE_URL"), sugar, hub)
	if err != nil {
		log.Fatalf("Error opening database connection: %v", err)
	}
	log.Infof("Opened database connection.")

	// add message create + interaction create handler
	b, err := bot.New(os.Getenv("REDIS"), r, db, sugar)
	if err != nil {
		log.Fatal("Error connecting to Redis: %v", err)
	}

	// actually load events + commands
	commands.Init(b, sugar)

	cacheFunc, countFunc, guildPermFunc, joinedFunc := events.Init(b, sugar)
	server.NewServer(r, db, cacheFunc, countFunc, guildPermFunc, joinedFunc)

	// connect to discord
	if err := r.ShardManager.Open(context.Background()); err != nil {
		log.Fatal("Failed to connect:", err)
	}

	// Defer this to make sure that things are always cleanly shutdown even in the event of a crash
	defer func() {
		db.Pool.Close()
		log.Info("Closed database connection.")
		r.ShardManager.Close()
		log.Info("Disconnected from Discord.")
	}()

	log.Info("Connected to Discord. Press Ctrl-C or send an interrupt signal to stop.")

	s, _ := r.StateFromGuildID(0)
	botUser, _ := s.Me()
	log.Infof("User: %v#%v (%v)", botUser.Username, botUser.Discriminator, botUser.ID)
	r.Bot = botUser
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
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	log.Infof("Interrupt signal received. Shutting down...")
}
