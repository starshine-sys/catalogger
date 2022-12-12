package importdb

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

var Command = &cli.Command{
	Name:   "importdb",
	Usage:  "Import messages and guild settings from an old database version",
	Action: run,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "from-db",
			Usage:    "URL of old database",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "from-key",
			Usage:    "Encryption key of old database",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "to-db",
			Usage:    "URL of new database",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "to-key",
			Usage:    "Encryption key of new database",
			Required: true,
		},
	},
}

const confirmText = "yes, I know what I'm doing!"

func run(c *cli.Context) error {
	fmt.Println("WARNING! This will delete any data already stored in the target database!")
	fmt.Printf("If you want to continue, please type \"%s\"\n", confirmText)

	return nil
}
