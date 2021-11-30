package web

import (
	"github.com/starshine-sys/catalogger/web/frontend"
	"github.com/urfave/cli/v2"
)

var Command = &cli.Command{
	Name:   "web",
	Usage:  "Run the dashboard",
	Action: run,
}

func run(c *cli.Context) (err error) {
	return frontend.Main()
}
