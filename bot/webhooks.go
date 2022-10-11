package bot

import (
	"github.com/diamondburned/arikawa/v3/api/webhook"
	"github.com/diamondburned/arikawa/v3/discord"
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
