package bot

import (
	"os"
	"os/signal"

	"emperror.dev/errors"
	"github.com/getsentry/sentry-go"
	"github.com/starshine-sys/catalogger/v2/bot"
	"github.com/starshine-sys/catalogger/v2/commands/config"
	metacommands "github.com/starshine-sys/catalogger/v2/commands/meta"
	"github.com/starshine-sys/catalogger/v2/common"
	"github.com/starshine-sys/catalogger/v2/common/log"
	"github.com/starshine-sys/catalogger/v2/logging/cache"
	"github.com/starshine-sys/catalogger/v2/logging/channels"
	"github.com/starshine-sys/catalogger/v2/logging/invites"
	"github.com/starshine-sys/catalogger/v2/logging/messages"
	"github.com/starshine-sys/catalogger/v2/logging/meta"
	"github.com/starshine-sys/catalogger/v2/logging/roles"
	"github.com/urfave/cli/v2"
)

var Command = &cli.Command{
	Name:   "bot",
	Usage:  "Run the bot",
	Action: run,
}

func run(c *cli.Context) error {
	conf, err := bot.ReadConfig("config.toml")
	if err != nil {
		return errors.Wrap(err, "reading config")
	}

	// set up sentry
	if conf.Auth.Sentry != "" {
		log.Debug("setting up sentry")
		err := sentry.Init(sentry.ClientOptions{
			Dsn:     conf.Auth.Sentry,
			Release: common.Version(),
		})
		if err != nil {
			log.Fatalf("setting up sentry: %v", err)
		}

		log.Debug("set up sentry")
	} else {
		log.Debugf("sentry DSN was not provided, not setting it up")
	}

	b, err := bot.New(conf)
	if err != nil {
		return errors.Wrap(err, "creating bot")
	}

	// set up modules (cache, logging, commands)
	cache.Setup(b)    // non-logging cache handlers
	roles.Setup(b)    // role logging
	messages.Setup(b) // message logging
	meta.Setup(b)     // meta logging (guilds, ready)
	invites.Setup(b)  // invite logging
	channels.Setup(b) // channel logging

	config.Setup(b)       // config commands
	metacommands.Setup(b) // meta commands

	// actually run bot!
	ctx, cancel := signal.NotifyContext(c.Context, os.Interrupt, os.Kill)
	defer cancel()

	err = b.Open(ctx)
	if err != nil {
		return errors.Wrap(err, "opening gateway connection")
	}

	defer func() {
		err = b.Router.ShardManager.Close()
		if err != nil {
			log.Errorf("closing gateway connection: %v", err)
		}
	}()

	<-ctx.Done()
	return nil
}
