package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"git.sr.ht/~starshine-sys/logger/commands"
	"git.sr.ht/~starshine-sys/logger/db"
	"git.sr.ht/~starshine-sys/logger/events"
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	_ "github.com/joho/godotenv/autoload"
	"github.com/starshine-sys/bcr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	debug, _ := strconv.ParseBool(os.Getenv("DEBUG_LOGGING"))

	zapConf := zap.NewDevelopmentConfig()
	if debug {
		zapConf.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	} else {
		zapConf.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}

	zap, err := zapConf.Build()
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
		[]string{os.Getenv("PREFIX")},
		intents,
	)
	if err != nil {
		sugar.Fatalf("Error creating bot: %v", err)
	}
	r.EmbedColor = bcr.ColourPurple

	// add message create handler
	r.State.AddHandler(r.MessageCreate)

	// set status
	r.State.AddHandler(func(ev *gateway.ReadyEvent) {
		r.State.Gateway.UpdateStatus(gateway.UpdateStatusData{
			Activities: []discord.Activity{{
				Name: fmt.Sprintf("%vhelp", os.Getenv("PREFIX")),
			}},
		})
	})

	// create a database connection
	db, err := db.New(os.Getenv("DATABASE_URL"), sugar)
	if err != nil {
		sugar.Fatalf("Error opening database connection: %v", err)
	}
	sugar.Infof("Opened database connection.")

	// actually load events + commands
	commands.Init(r, db, sugar)
	events.Init(r, db, sugar)

	// connect to discord
	if err := r.State.Open(); err != nil {
		sugar.Fatal("Failed to connect:", err)
	}

	// Defer this to make sure that things are always cleanly shutdown even in the event of a crash
	defer func() {
		db.Pool.Close()
		sugar.Info("Closed database connection.")
		r.State.Close()
		r.State.Gateway.Close()
		sugar.Info("Disconnected from Discord.")
	}()

	sugar.Info("Connected to Discord. Press Ctrl-C or send an interrupt signal to stop.")

	botUser, _ := r.State.Me()
	sugar.Infof("User: %v#%v (%v)", botUser.Username, botUser.Discriminator, botUser.ID)
	r.Bot = botUser

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	sugar.Infof("Interrupt signal received. Shutting down...")
}
