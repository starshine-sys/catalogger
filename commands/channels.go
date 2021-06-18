package commands

import (
	"context"
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
		return bot.DB.ReportCtx(ctx, err)
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

	var ignored []uint64
	err = bot.DB.Pool.QueryRow(context.Background(), "select ignored_channels from guilds where id = $1", ctx.Message.GuildID).Scan(&ignored)
	if err == nil && len(ignored) > 0 {
		f := discord.EmbedField{Name: "Ignored channels"}
		for i, ch := range ignored {
			if len(f.Value) > 900 {
				f.Value += fmt.Sprintf("\n```Too many to list (showing %v/%v)```", i, len(ignored))
				break
			}
			f.Value += discord.ChannelID(ch).Mention()
			if i != len(ignored)-1 {
				f.Value += ", "
			}
		}
		e.Fields = append(e.Fields, f)
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
