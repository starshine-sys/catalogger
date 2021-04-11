package commands

import (
	"fmt"
	"strings"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/starshine-sys/bcr"
)

func (bot *Bot) channels(ctx *bcr.Context) (err error) {
	var setChannels []string
	var unsetEvents []string

	ch, err := bot.DB.Channels(ctx.Message.GuildID)
	if err != nil {
		_, err = ctx.Sendf("Error: %v", err)
		return
	}

	for k, v := range ch {
		if v.IsValid() {
			setChannels = append(setChannels, fmt.Sprintf("- `%v`: %v", k, v.Mention()))
		} else {
			unsetEvents = append(unsetEvents, "`"+k+"`")
		}
	}

	e := &discord.Embed{
		Title:       "Events",
		Description: strings.Join(setChannels, "\n"),
		Color:       bcr.ColourPurple,
	}

	if len(unsetEvents) > 0 {
		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  "Other events",
			Value: fmt.Sprintf("These events are currently not being logged:\n%v", strings.Join(unsetEvents, ", ")),
		})
	}

	_, err = ctx.Send("", e)
	return
}
