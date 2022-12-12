package main

import (
	"fmt"
	"os"

	"github.com/starshine-sys/catalogger/v2/cmd/bot"
	"github.com/starshine-sys/catalogger/v2/cmd/importdb"
	"github.com/starshine-sys/catalogger/v2/cmd/migrate"
	"github.com/starshine-sys/catalogger/v2/common"
	"github.com/urfave/cli/v2"
)

var app = &cli.App{
	Name:    "catalogger",
	Usage:   "Discord logging bot",
	Version: common.Version(),

	Commands: []*cli.Command{
		bot.Command,
		migrate.Command,
		importdb.Command,
	},
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Println("error in command:", err)
		os.Exit(1)
	}
}
