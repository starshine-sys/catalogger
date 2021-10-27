package events

import (
	"fmt"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/events/handler"
)

func (bot *Bot) channelDelete(ev *gateway.ChannelDeleteEvent) (resp *handler.Response, err error) {
	bot.ChannelsMu.Lock()
	delete(bot.Channels, ev.ID)
	bot.ChannelsMu.Unlock()

	ch, err := bot.DB.Channels(ev.GuildID)
	if err != nil {
		return
	}
	if !ch[keys.ChannelDelete].IsValid() {
		return
	}

	resp = &handler.Response{
		ChannelID: ch[keys.ChannelDelete],
		GuildID:   ev.GuildID,
	}

	desc := fmt.Sprintf("**Name:** #%v", ev.Name)
	if cat, err := bot.State(ev.GuildID).Channel(ev.ParentID); err == nil {
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

	resp.Embeds = append(resp.Embeds, e)
	return
}
