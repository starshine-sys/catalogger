package main

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/getsentry/sentry-go"
	_ "github.com/joho/godotenv/autoload"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/commands"
	"github.com/starshine-sys/catalogger/db"
	"github.com/starshine-sys/catalogger/events"
	"github.com/starshine-sys/catalogger/web/server"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	debug, _ := strconv.ParseBool(os.Getenv("DEBUG_LOGGING"))

	// set up a logger
	zcfg := zap.NewProductionConfig()
	zcfg.Encoding = "console"
	zcfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	zcfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	zcfg.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	zcfg.EncoderConfig.EncodeDuration = zapcore.StringDurationEncoder

	if debug {
		zcfg.Level.SetLevel(zapcore.DebugLevel)
	} else {
		zcfg.Level.SetLevel(zapcore.InfoLevel)
	}

	zap, err := zcfg.Build(zap.AddStacktrace(zapcore.ErrorLevel))
	if err != nil {
		panic(err)
	}
	sugar := zap.Sugar()

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
		sugar.Fatalf("Error creating bot: %v", err)
	}
	r.EmbedColor = bcr.ColourPurple

	// add message create handler
	r.AddHandler(r.MessageCreate)

	// sentry, if enabled
	var hub *sentry.Hub
	if os.Getenv("SENTRY_URL") != "" {
		err = sentry.Init(sentry.ClientOptions{
			Dsn: os.Getenv("SENTRY_URL"),
		})
		if err != nil {
			sugar.Fatalf("Error initialising Sentry: %v", err)
		}
		hub = sentry.CurrentHub()
	}

	// create a database connection
	db, err := db.New(os.Getenv("DATABASE_URL"), sugar, hub)
	if err != nil {
		sugar.Fatalf("Error opening database connection: %v", err)
	}
	sugar.Infof("Opened database connection.")

	// actually load events + commands
	commands.Init(r, db, sugar)

	cacheFunc, countFunc, guildPermFunc, joinedFunc := events.Init(r, db, sugar)
	server.NewServer(r, db, cacheFunc, countFunc, guildPermFunc, joinedFunc)

	// connect to discord
	if err := r.ShardManager.Open(context.Background()); err != nil {
		sugar.Fatal("Failed to connect:", err)
	}

	// Defer this to make sure that things are always cleanly shutdown even in the event of a crash
	defer func() {
		db.Pool.Close()
		sugar.Info("Closed database connection.")
		r.ShardManager.Close()
		sugar.Info("Disconnected from Discord.")
	}()

	sugar.Info("Connected to Discord. Press Ctrl-C or send an interrupt signal to stop.")

	s, _ := r.StateFromGuildID(0)
	botUser, _ := s.Me()
	sugar.Infof("User: %v#%v (%v)", botUser.Username, botUser.Discriminator, botUser.ID)
	r.Bot = botUser
	// normally creating a Context would do this, but as we set the user above, that doesn't work
	r.Prefixes = append(r.Prefixes, "<@"+r.Bot.ID.String()+">", "<@!"+r.Bot.ID.String()+">")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	sugar.Infof("Interrupt signal received. Shutting down...")
}
