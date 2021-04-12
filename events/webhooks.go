package events

import (
	"errors"
	"strings"

	"github.com/diamondburned/arikawa/v2/api"
	"github.com/diamondburned/arikawa/v2/discord"
)

// ErrNotExists ...
var ErrNotExists = errors.New("webhooks not found in cache")

// Webhook ...
type Webhook struct {
	ID    discord.WebhookID
	Token string
}

// SetWebhooks ...
func (bot *Bot) SetWebhooks(t string, id discord.GuildID, w *Webhook) {
	switch strings.ToLower(t) {
	case "msg_delete":
		bot.MessageDeleteCache.Set(id.String(), w)
	case "msg_update":
		bot.MessageUpdateCache.Set(id.String(), w)
	case "join":
		bot.GuildMemberAddCache.Set(id.String(), w)
	case "invite-create":
		bot.InviteCreateCache.Set(id.String(), w)
	case "invite-delete":
		bot.InviteDeleteCache.Set(id.String(), w)
	default:
		return
	}
}

// GetWebhooks ...
func (bot *Bot) GetWebhooks(t string, id discord.GuildID) (*Webhook, error) {
	var (
		v   interface{}
		err error
	)

	switch strings.ToLower(t) {
	case "msg_delete":
		v, err = bot.MessageDeleteCache.Get(id.String())
	case "msg_update":
		v, err = bot.MessageUpdateCache.Get(id.String())
	case "join":
		v, err = bot.GuildMemberAddCache.Get(id.String())
	case "invite-create":
		v, err = bot.InviteCreateCache.Get(id.String())
	case "invite-delete":
		v, err = bot.InviteDeleteCache.Get(id.String())
	default:
		return nil, errors.New("invalid webhook type specified")
	}
	if err != nil {
		return nil, ErrNotExists
	}
	if _, ok := v.(*Webhook); !ok {
		return nil, errors.New("could not convert interface to Webhooks")
	}

	return v.(*Webhook), nil
}

// ResetCache ...
func (bot *Bot) ResetCache(id discord.GuildID) {
	bot.MessageDeleteCache.Remove(id.String())
	bot.MessageUpdateCache.Remove(id.String())
	bot.GuildMemberAddCache.Remove(id.String())
	bot.InviteCreateCache.Remove(id.String())
	bot.InviteDeleteCache.Remove(id.String())
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

func (bot *Bot) webhookCache(t string, guildID discord.GuildID, ch discord.ChannelID) (*discord.Webhook, error) {
	// try getting the cached webhook
	var wh *discord.Webhook

	w, err := bot.GetWebhooks(t, guildID)
	if err != nil {
		wh, err = bot.getWebhook(ch, bot.Router.Bot.Username)
		if err != nil {
			return nil, err
		}

		bot.SetWebhooks(t, guildID, &Webhook{
			ID:    wh.ID,
			Token: wh.Token,
		})
	} else {
		wh = &discord.Webhook{
			ID:    w.ID,
			Token: w.Token,
		}
	}

	return wh, nil
}
