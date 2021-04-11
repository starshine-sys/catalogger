package events

import (
	"fmt"

	"github.com/diamondburned/arikawa/v2/api"
	"github.com/diamondburned/arikawa/v2/api/webhook"
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/starshine-sys/bcr"
)

func (bot *Bot) pkMessageDelete(m *gateway.MessageDeleteEvent) {
	if !m.GuildID.IsValid() {
		return
	}

	ch, err := bot.DB.Channels(m.GuildID)
	if err != nil {
		bot.Sugar.Errorf("Error getting server channels: %v", err)
		return
	}
	if !ch["MESSAGE_DELETE"].IsValid() {
		return
	}

	// try getting the message
	msg, err := bot.DB.GetProxied(m.ID)
	if err != nil {
		return
	}

	// try getting the cached webhook
	var wh *discord.Webhook

	w, err := bot.GetWebhooks("msg", m.GuildID)
	if err != nil {
		wh, err = bot.getWebhook(ch["MESSAGE_DELETE"], bot.Router.Bot.Username)
		if err != nil {
			bot.Sugar.Errorf("Error getting webhook: %v", err)
			return
		}

		bot.SetWebhooks("msg", m.GuildID, &Webhook{
			ID:    wh.ID,
			Token: wh.Token,
		})
	} else {
		wh = &discord.Webhook{
			ID:    w.ID,
			Token: w.Token,
		}
	}

	mention := msg.UserID.Mention()
	var author *discord.EmbedAuthor
	u, err := bot.State.User(msg.UserID)
	if err == nil {
		mention = fmt.Sprintf("%v\n%v#%v\nID: %v", u.Mention(), u.Username, u.Discriminator, u.ID)
		author = &discord.EmbedAuthor{
			Icon: u.AvatarURL(),
			Name: u.Username + u.Discriminator,
		}
	}

	e := discord.Embed{
		Author:      author,
		Title:       fmt.Sprintf("Message by \"%v\" deleted", msg.Username),
		Description: msg.Content,
		Color:       bcr.ColourRed,
		Fields: []discord.EmbedField{
			{
				Name:  "​",
				Value: "​",
			},
			{
				Name:   "Channel",
				Value:  fmt.Sprintf("%v\nID: %v", msg.ChannelID.Mention(), msg.ChannelID),
				Inline: true,
			},
			{
				Name:   "Linked Discord account",
				Value:  mention,
				Inline: true,
			},
			{
				Name:  "​",
				Value: "​",
			},
			{
				Name:   "System ID",
				Value:  msg.System,
				Inline: true,
			},
			{
				Name:   "Member ID",
				Value:  msg.Member,
				Inline: true,
			},
		},
		Footer: &discord.EmbedFooter{
			Text: fmt.Sprintf("ID: %v", msg.MsgID),
		},
		Timestamp: discord.NewTimestamp(msg.MsgID.Time()),
	}

	_, err = webhook.New(wh.ID, wh.Token).ExecuteAndWait(webhook.ExecuteData{
		AvatarURL: bot.Router.Bot.AvatarURL(),
		Embeds:    []discord.Embed{e},
	})
	if err == nil {
		bot.DB.DeleteProxied(msg.MsgID)
	}
}

func (bot *Bot) getWebhook(id discord.ChannelID, name string) (*discord.Webhook, error) {
	ws, err := bot.State.ChannelWebhooks(id)
	if err == nil {
		for _, w := range ws {
			if w.Name == name {
				return &w, nil
			}
		}
	} else {
		return nil, err
	}

	w, err := bot.State.CreateWebhook(id, api.CreateWebhookData{
		Name: name,
	})
	return w, err
}
