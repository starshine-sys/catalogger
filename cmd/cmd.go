package cmd

import (
	"os"

	"github.com/starshine-sys/catalogger/v2/cmd/bot"
	"github.com/starshine-sys/catalogger/v2/common"
	"github.com/urfave/cli/v2"
)

var app = &cli.App{
	Name:    "Catalogger",
	Usage:   "Discord logging bot",
	Version: common.Version(),

	Commands: []*cli.Command{
		bot.Command,
	},
}

func Run() error {
	return app.Run(os.Args)
}
