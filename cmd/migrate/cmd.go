package migrate

import (
	"github.com/starshine-sys/catalogger/v2/bot"
	"github.com/starshine-sys/catalogger/v2/common/log"
	"github.com/starshine-sys/catalogger/v2/db"
	"github.com/urfave/cli/v2"
)

var Command = &cli.Command{
	Name:   "migrate",
	Usage:  "Run migrations manually",
	Action: run,
	Flags: []cli.Flag{&cli.BoolFlag{
		Name:    "force",
		Aliases: []string{"f"},
		Usage:   "Run migrations whether or not no_auto_migrate is set in the config.",
		Value:   false,
	}},
}

func run(c *cli.Context) error {
	conf, err := bot.ReadConfig("config.toml")
	if err != nil {
		log.Fatalf("Reading configuration: %v", err)
	}

	if conf.Auth.Postgres == "" {
		return cli.Exit("No database url set in config.toml.", 1)
	}

	if !conf.Bot.NoAutoMigrate && !c.Bool("force") {
		return cli.Exit("Migrations are run automatically, and the --force flag is not set.", 1)
	}

	err = db.RunMigrations(conf.Auth.Postgres)
	if err != nil {
		log.Fatalf("Running migrations: %v", err)
	}

	log.Info("Successfully ran migrations!")
	return nil
}
