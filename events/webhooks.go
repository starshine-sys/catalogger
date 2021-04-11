package events

import (
	"errors"
	"strings"

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
	case "msg":
		bot.MsgWebhookCache.Set(id.String(), w)
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
	case "msg":
		v, err = bot.MsgWebhookCache.Get(id.String())
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
	bot.MsgWebhookCache.Remove(id.String())
}
