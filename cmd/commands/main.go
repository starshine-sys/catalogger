package commands

import (
	"fmt"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/starshine-sys/catalogger/v2/common"
	"github.com/urfave/cli/v2"
)

var Command = &cli.Command{
	Name:   "commands",
	Usage:  "Synchronize slash commands",
	Action: run,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "token",
			Usage:    "The bot's token",
			EnvVars:  []string{"TOKEN"},
			Required: true,
		},
		&cli.Uint64Flag{
			Name:     "app-id",
			Usage:    "The bot's application ID",
			EnvVars:  []string{"APP_ID"},
			Required: true,
		},
		&cli.BoolFlag{
			Name:  "global",
			Usage: "Synchronize slash commands globally (mutually exclusive with --guild)",
		},
		&cli.Uint64Flag{
			Name:  "guild",
			Usage: "Synchronize slash commands to a specific guild",
		},
	},
}

func run(c *cli.Context) error {
	global := c.Bool("global")
	guild := c.Uint64("guild")
	if global && guild != 0 {
		return cli.Exit("`global` and `guild` are mutually exclusive", 1)
	}

	if global {
		return globalCommands(c)
	}

	if guild != 0 {
		return guildCommands(c)
	}

	return cli.Exit("Neither `global` nor `guild` were set", 1)
}

func globalCommands(c *cli.Context) error {
	appID := c.Uint64("app-id")
	token := c.String("token")

	client := api.NewClient("Bot " + token)

	_, err := client.BulkOverwriteCommands(discord.AppID(appID), common.Commands)
	if err != nil {
		fmt.Println("Error overwriting commands:", err)
		return err
	}

	fmt.Println("Wrote global commands!")
	return nil
}

func guildCommands(c *cli.Context) error {
	appID := c.Uint64("app-id")
	token := c.String("token")
	guildID := c.Uint64("guild")

	client := api.NewClient("Bot " + token)

	_, err := client.BulkOverwriteGuildCommands(discord.AppID(appID), discord.GuildID(guildID), common.Commands)
	if err != nil {
		fmt.Println("Error overwriting commands:", err)
		return nil
	}

	fmt.Printf("Wrote guild commands in %v!\n", guildID)
	return nil
}
