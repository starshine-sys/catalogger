package commands

import (
	"strings"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/db"
)

func (bot *Bot) events(ctx bcr.Contexter) (err error) {
	var s []string

	for _, e := range db.Events {
		s = append(s, "- `"+e+"`")
	}

	e := discord.Embed{
		Title:       "Available events",
		Description: "The following events are available:\n" + strings.Join(s, "\n"),
		Color:       bcr.ColourPurple,
	}

	return ctx.SendEphemeral("", e)
}
