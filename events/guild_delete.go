package events

import (
	"fmt"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/common"
	"github.com/starshine-sys/catalogger/events/handler"
)

func (bot *Bot) guildDelete(ev *gateway.GuildDeleteEvent) (resp *handler.Response, err error) {
	if ev.Unavailable {
		return
	}

	g, ok := bot.Guilds.Get(ev.ID)
	if !ok {
		return bot.guildDeleteNoState(ev)
	}
	bot.Guilds.Remove(ev.ID)

	e := discord.Embed{
		Title: "Left server",
		Color: bcr.ColourPurple,
		Thumbnail: &discord.EmbedThumbnail{
			URL: g.IconURL(),
		},

		Description: fmt.Sprintf("Left server **%v**", g.Name),
		Footer: &discord.EmbedFooter{
			Text: fmt.Sprintf("ID: %v", g.ID),
		},
		Timestamp: discord.NowTimestamp(),
	}

	return &handler.Response{
		ChannelID: bot.BotJoinLeaveLog,
		Embeds:    []discord.Embed{e},
	}, nil
}

// this is run if the left guild isn't found in the state
// which gives us almost no info, only the ID
func (bot *Bot) guildDeleteNoState(g *gateway.GuildDeleteEvent) (resp *handler.Response, err error) {
	common.Log.Infof("Left server %v.", g.ID)

	if !bot.BotJoinLeaveLog.IsValid() {
		return
	}

	return &handler.Response{
		ChannelID: bot.BotJoinLeaveLog,
		Embeds: []discord.Embed{{
			Title:       "Left server",
			Description: fmt.Sprintf("Left server **%v**", g.ID),

			Footer: &discord.EmbedFooter{
				Text: fmt.Sprintf("ID: %v", g.ID),
			},
			Timestamp: discord.NowTimestamp(),
		}},
	}, nil
}
