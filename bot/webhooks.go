package bot

import (
	"context"
	"strings"

	"emperror.dev/errors"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/api/webhook"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/mediocregopher/radix/v4"
	"github.com/starshine-sys/catalogger/v2/common/log"
)

// webhookClient returns a client for the given webhook.
// If no client is cached, it creates a new one.
func (bot *Bot) webhookClient(wh *discord.Webhook) *webhook.Client {
	bot.webhookClientsMu.Lock()
	defer bot.webhookClientsMu.Unlock()

	c, ok := bot.webhookClients[wh.ID]
	if !ok {
		c = webhook.FromAPI(wh.ID, wh.Token, bot.Router.Rest)
		bot.webhookClients[wh.ID] = c
	}

	return c
}

const (
	errWebhookNotFound = errors.Sentinel("webhook not found")
	errWebhookInvalid  = errors.Sentinel("webhook invalid")
)

func webhookKey(id discord.ChannelID) string { return "webhook:" + id.String() }

func (bot *Bot) fetchCachedWebhookKey(channelID discord.ChannelID) (id discord.WebhookID, token string, err error) {
	var s string
	err = bot.DB.Redis.Do(context.Background(), radix.Cmd(&s, "GET", webhookKey(channelID)))
	if err != nil {
		return
	}
	if s == "" {
		return id, token, errWebhookNotFound
	}

	w := strings.SplitN(s, ":", 2)
	if len(w) != 2 {
		err = bot.DB.Redis.Do(context.Background(), radix.Cmd(nil, "DEL", webhookKey(channelID)))
		return id, token, errors.Append(err, errWebhookInvalid)
	}

	whID, err := discord.ParseSnowflake(w[0])
	if err != nil {
		err = bot.DB.Redis.Do(context.Background(), radix.Cmd(nil, "DEL", webhookKey(channelID)))
		return id, token, errors.Append(err, errWebhookInvalid)
	}

	return discord.WebhookID(whID), w[1], nil
}

func (bot *Bot) storeWebhook(channelID discord.ChannelID, wh *discord.Webhook) (err error) {
	stored := wh.ID.String() + ":" + wh.Token

	err = bot.DB.Redis.Do(context.Background(), radix.Cmd(nil, "SET", webhookKey(channelID), stored))
	if err != nil {
		return errors.Wrap(err, "storing webhook")
	}
	return nil
}

func (bot *Bot) getWebhook(channelID discord.ChannelID) (*discord.Webhook, error) {
	// check if we've got a cached webhook
	id, token, err := bot.fetchCachedWebhookKey(channelID)
	if err == nil {
		return &discord.Webhook{
			ID:        id,
			Token:     token,
			ChannelID: channelID,
		}, nil
	}

	log.Debugf("no cached webhook found for %v, falling back to API", channelID)

	// no cached webhook, fetch webhooks instead
	whs, err := bot.Router.Rest.ChannelWebhooks(channelID)
	if err != nil {
		return nil, errors.Wrap(err, "getting webhooks")
	}

	// return first webhook where user ID == our ID
	for _, wh := range whs {
		if wh.User != nil && wh.User.ID == bot.user.ID {
			err = bot.storeWebhook(channelID, &wh)
			if err != nil {
				log.Errorf("storing webhook %v in %v: %v", wh.ID, channelID, err)
			}
			return &wh, nil
		}
	}

	log.Debugf("no webhooks found for %v, creating webhook", channelID)

	// create new webhook
	wh, err := bot.Router.Rest.CreateWebhook(channelID, api.CreateWebhookData{
		Name: bot.user.Username,
	})
	if err != nil {
		return nil, errors.Wrap(err, "creating new webhook")
	}

	err = bot.storeWebhook(channelID, wh)
	if err != nil {
		log.Errorf("storing webhook %v in %v: %v", wh.ID, channelID, err)
	}

	return wh, nil
}
