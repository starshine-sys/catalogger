package events

import (
	"fmt"

	"github.com/diamondburned/arikawa/v2/api/webhook"
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/starshine-sys/bcr"
)

func (bot *Bot) channelDelete(ev *gateway.ChannelDeleteEvent) {
	bot.ChannelsMu.Lock()
	delete(bot.Channels, ev.ID)
	bot.ChannelsMu.Unlock()

	ch, err := bot.DB.Channels(ev.GuildID)
	if err != nil {
		bot.Sugar.Errorf("Error getting server channels: %v", err)
		return
	}
	if !ch["CHANNEL_DELETE"].IsValid() {
		return
	}

	wh, err := bot.webhookCache("channel_delete", ev.GuildID, ch["CHANNEL_DELETE"])
	if err != nil {
		bot.Sugar.Errorf("Error getting webhook: %v", err)
		return
	}

	desc := fmt.Sprintf("**Name:** #%v", ev.Name)
	if cat, err := bot.State.Channel(ev.CategoryID); err == nil {
		desc += fmt.Sprintf("\n**Category:** %v", cat.Name)
	} else {
		desc += "\n**Category:** None"
	}

	e := discord.Embed{
		Title:       "Channel deleted",
		Description: desc,

		Color: bcr.ColourRed,
		Footer: &discord.EmbedFooter{
			Text: "ID: " + ev.ID.String(),
		},
		Timestamp: discord.NowTimestamp(),
	}

	if ev.Topic != "" {
		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  "Description",
			Value: ev.Topic,
		})
	}

	webhook.New(wh.ID, wh.Token).Execute(webhook.ExecuteData{
		AvatarURL: bot.Router.Bot.AvatarURL(),
		Embeds:    []discord.Embed{e},
	})
}
