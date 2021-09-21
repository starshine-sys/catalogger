package events

import (
	"fmt"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/bcr"
)

func (bot *Bot) guildDelete(g *gateway.GuildDeleteEvent) {
	if g.Unavailable {
		return
	}

	guild, err := bot.State(g.ID).Guild(g.ID)
	if err != nil {
		// didn't find the guild, so just run this normally
		bot.guildDeleteNoState(g)
		return
	}

	e := discord.Embed{
		Title: "Left server",
		Color: bcr.ColourPurple,
		Thumbnail: &discord.EmbedThumbnail{
			URL: guild.IconURL(),
		},

		Description: fmt.Sprintf("Left server **%v**", guild.Name),
		Footer: &discord.EmbedFooter{
			Text: fmt.Sprintf("ID: %v", guild.ID),
		},
		Timestamp: discord.NowTimestamp(),
	}

	_, err = bot.State(g.ID).SendEmbeds(bot.BotJoinLeaveLog, e)
	if err != nil {
		bot.Sugar.Errorf("Error sending leave log message: %v", err)
	}
	return
}

// this is run if the left guild isn't found in the state
// which gives us almost no info, only the ID
func (bot *Bot) guildDeleteNoState(g *gateway.GuildDeleteEvent) {
	bot.Sugar.Infof("Left server %v.", g.ID)

	if !bot.BotJoinLeaveLog.IsValid() {
		return
	}

	_, err := bot.State(g.ID).SendEmbeds(bot.BotJoinLeaveLog, discord.Embed{
		Title:       "Left server",
		Description: fmt.Sprintf("Left server **%v**", g.ID),

		Footer: &discord.EmbedFooter{
			Text: fmt.Sprintf("ID: %v", g.ID),
		},
		Timestamp: discord.NowTimestamp(),
	})
	if err != nil {
		bot.Sugar.Errorf("Error sending leave log message: %v", err)
	}
	return
}
