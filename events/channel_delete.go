package events

import (
	"fmt"

	"github.com/diamondburned/arikawa/v3/api/webhook"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/db"
)

func (bot *Bot) channelDelete(ev *gateway.ChannelDeleteEvent) {
	bot.ChannelsMu.Lock()
	delete(bot.Channels, ev.ID)
	bot.ChannelsMu.Unlock()

	ch, err := bot.DB.Channels(ev.GuildID)
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   keys.ChannelDelete,
			GuildID: ev.GuildID,
		}, err)
		return
	}
	if !ch[keys.ChannelDelete].IsValid() {
		return
	}

	wh, err := bot.webhookCache(keys.ChannelDelete, ev.GuildID, ch[keys.ChannelDelete])
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   keys.ChannelDelete,
			GuildID: ev.GuildID,
		}, err)
		return
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

	err = webhook.FromAPI(wh.ID, wh.Token, bot.State(ev.GuildID).Client).Execute(webhook.ExecuteData{
		AvatarURL: bot.Router.Bot.AvatarURL(),
		Embeds:    []discord.Embed{e},
	})
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   keys.ChannelDelete,
			GuildID: ev.GuildID,
		}, err)
		return
	}
}
