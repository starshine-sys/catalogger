package bot

import (
	"os"
	"os/signal"

	"emperror.dev/errors"
	"github.com/starshine-sys/catalogger/v2/bot"
	"github.com/starshine-sys/catalogger/v2/common/log"
	"github.com/starshine-sys/catalogger/v2/logging/cache"
	"github.com/urfave/cli/v2"
)

var Command = &cli.Command{
	Name:   "bot",
	Usage:  "Run the bot",
	Action: run,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "token",
			Usage:   "Bot token",
			Aliases: []string{"t"},
		},
	},
}

func run(c *cli.Context) error {
	conf, err := bot.ReadConfig("config.toml")
	if err != nil {
		return errors.Wrap(err, "reading config")
	}

	b, err := bot.New(conf)
	if err != nil {
		return errors.Wrap(err, "creating bot")
	}

	// set up modules (cache, logging, commands)
	cache.Setup(b)

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
