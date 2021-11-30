package main

import (
	"os"

	"github.com/starshine-sys/catalogger/cmd/catalogger/bot"
	"github.com/starshine-sys/catalogger/cmd/catalogger/web"
	"github.com/starshine-sys/catalogger/common"
	"github.com/urfave/cli/v2"
)

var app = &cli.App{
	Name:    "Catalogger",
	Usage:   "Logging bot for Discord",
	Version: common.Version,

	Commands: []*cli.Command{
		bot.Command,
		web.Command,
	},
}

func main() {
	err := app.Run(os.Args)
	if err != nil {
		common.Log.Fatal(err)
	}
}
