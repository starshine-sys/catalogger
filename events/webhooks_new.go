package events

import (
	"context"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/mediocregopher/radix/v4"
)

func whKeyNew(id discord.ChannelID) string {
	return "wh:" + id.String()
}

// SetWebhookNew ...
func (bot *Bot) SetWebhookNew(channelID discord.ChannelID, w *Webhook) error {
	return bot.Redis.Do(context.Background(), radix.Cmd(nil, "SET", whKeyNew(channelID), w.ID.String()+":"+w.Token, "EX", webhookCacheExpiry))
}

// GetWebhookNew gets a cached webhook from redis
func (bot *Bot) GetWebhookNew(id discord.ChannelID) (*Webhook, error) {
	return bot.fetchCachedKey(whKeyNew(id))
}

// ResetCacheNew ...
func (bot *Bot) ResetCacheNew(id discord.GuildID, channels ...discord.ChannelID) {
	var keys []string

	for _, ch := range channels {
		keys = append(keys, whKeyNew(ch))
	}

	bot.Sugar.Debugf("Deleting cache entries for %v", id)

	err := bot.Redis.Do(context.Background(), radix.Cmd(nil, "DEL", keys...))
	if err != nil {
		bot.Sugar.Errorf("Error resetting webhook cache: %v", err)
	}
}

func (bot *Bot) webhookCacheNew(guildID discord.GuildID, ch discord.ChannelID) (*discord.Webhook, error) {
	// try getting the cached webhook
	var wh *discord.Webhook

	w, err := bot.GetWebhookNew(ch)
	if err != nil {
		bot.Sugar.Debugf("Couldn't find webhook for %v in cache, falling back to fetching webhook", ch)

		if err != ErrNotExists && err != ErrInvalid {
			bot.Sugar.Errorf("Error fetching webhook: %v", err)
		}

		wh, err = bot.getWebhook(ch, bot.Router.Bot.Username)
		if err != nil {
			return nil, err
		}

		bot.SetWebhookNew(ch, &Webhook{
			ID:    wh.ID,
			Token: wh.Token,
		})
	} else {
		wh = &discord.Webhook{
			ID:        w.ID,
			Token:     w.Token,
			ChannelID: ch,
		}
	}

	wh.GuildID = guildID
	return wh, nil
}
