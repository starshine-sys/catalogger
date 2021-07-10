package events

import (
	"errors"
	"strings"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
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
	case "ban-add":
		bot.GuildBanAddCache.Set(id.String(), w)
	case "ban-remove":
		bot.GuildBanRemoveCache.Set(id.String(), w)
	case "leave":
		bot.GuildMemberRemoveCache.Set(id.String(), w)
	case "member-update":
		bot.GuildMemberUpdateCache.Set(id.String(), w)
	case "member-nick-update":
		bot.GuildMemberNickUpdateCache.Set(id.String(), w)
	case "channel_create":
		bot.ChannelCreateCache.Set(id.String(), w)
	case "channel_update":
		bot.ChannelUpdateCache.Set(id.String(), w)
	case "channel_delete":
		bot.ChannelDeleteCache.Set(id.String(), w)
	case "guild_update":
		bot.GuildUpdateCache.Set(id.String(), w)
	case "guild_emojis_update":
		bot.GuildEmojisUpdateCache.Set(id.String(), w)
	case "guild_role_create":
		bot.GuildRoleCreateCache.Set(id.String(), w)
	case "guild_role_update":
		bot.GuildRoleUpdateCache.Set(id.String(), w)
	case "guild_role_delete":
		bot.GuildRoleDeleteCache.Set(id.String(), w)
	case "message_delete_bulk":
		bot.MessageDeleteBulkCache.Set(id.String(), w)
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
	case "ban-add":
		v, err = bot.GuildBanAddCache.Get(id.String())
	case "ban-remove":
		v, err = bot.GuildBanRemoveCache.Get(id.String())
	case "leave":
		v, err = bot.GuildMemberRemoveCache.Get(id.String())
	case "member-update":
		v, err = bot.GuildMemberUpdateCache.Get(id.String())
	case "member-nick-update":
		v, err = bot.GuildMemberNickUpdateCache.Get(id.String())
	case "channel_create":
		v, err = bot.ChannelCreateCache.Get(id.String())
	case "channel_update":
		v, err = bot.ChannelUpdateCache.Get(id.String())
	case "channel_delete":
		v, err = bot.ChannelDeleteCache.Get(id.String())
	case "guild_update":
		v, err = bot.GuildUpdateCache.Get(id.String())
	case "guild_emojis_update":
		v, err = bot.GuildEmojisUpdateCache.Get(id.String())
	case "guild_role_create":
		v, err = bot.GuildRoleCreateCache.Get(id.String())
	case "guild_role_update":
		v, err = bot.GuildRoleUpdateCache.Get(id.String())
	case "guild_role_delete":
		v, err = bot.GuildRoleDeleteCache.Get(id.String())
	case "message_delete_bulk":
		v, err = bot.MessageDeleteBulkCache.Get(id.String())
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
func (bot *Bot) ResetCache(id discord.GuildID, channels ...discord.ChannelID) {
	bot.MessageDeleteCache.Remove(id.String())
	bot.MessageUpdateCache.Remove(id.String())
	bot.GuildMemberAddCache.Remove(id.String())
	bot.InviteCreateCache.Remove(id.String())
	bot.InviteDeleteCache.Remove(id.String())
	bot.GuildBanAddCache.Remove(id.String())
	bot.GuildBanRemoveCache.Remove(id.String())
	bot.GuildMemberRemoveCache.Remove(id.String())
	bot.GuildMemberUpdateCache.Remove(id.String())
	bot.GuildMemberNickUpdateCache.Remove(id.String())
	bot.ChannelCreateCache.Remove(id.String())
	bot.ChannelUpdateCache.Remove(id.String())
	bot.ChannelDeleteCache.Remove(id.String())
	bot.GuildUpdateCache.Remove(id.String())
	bot.GuildEmojisUpdateCache.Remove(id.String())
	bot.GuildRoleCreateCache.Remove(id.String())
	bot.GuildRoleUpdateCache.Remove(id.String())
	bot.GuildRoleDeleteCache.Remove(id.String())
	bot.MessageDeleteBulkCache.Remove(id.String())

	for _, ch := range channels {
		bot.MessageDeleteCache.Remove(ch.String())
		bot.MessageUpdateCache.Remove(ch.String())
	}
}

func (bot *Bot) getWebhook(guildID discord.GuildID, channelID discord.ChannelID, name string) (*discord.Webhook, error) {
	ws, err := bot.State(guildID).ChannelWebhooks(channelID)
	if err == nil {
		for _, w := range ws {
			if w.Name == name && (w.User.ID == bot.Bot.ID || !w.User.ID.IsValid()) {
				return &w, nil
			}
		}
	} else {
		return nil, err
	}

	w, err := bot.State(guildID).CreateWebhook(channelID, api.CreateWebhookData{
		Name: name,
	})
	return w, err
}

func (bot *Bot) webhookCache(t string, guildID discord.GuildID, ch discord.ChannelID) (*discord.Webhook, error) {
	// try getting the cached webhook
	var wh *discord.Webhook

	w, err := bot.GetWebhooks(t, guildID)
	if err != nil {
		wh, err = bot.getWebhook(guildID, ch, bot.Router.Bot.Username)
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

func (bot *Bot) getRedirect(guildID discord.GuildID, ch discord.ChannelID) (*discord.Webhook, error) {
	// try getting the cached webhook
	var wh *discord.Webhook

	v, err := bot.RedirectCache.Get(ch.String())
	if err == nil {
		return v.(*discord.Webhook), nil
	}

	wh, err = bot.getWebhook(guildID, ch, bot.Router.Bot.Username)
	if err != nil {
		return nil, err
	}

	bot.RedirectCache.Set(ch.String(), wh)

	return wh, nil
}

func (bot *Bot) webhooksUpdate(ev *gateway.WebhooksUpdateEvent) {
	bot.ResetCache(ev.GuildID, ev.ChannelID)
	bot.RedirectCache.Remove(ev.ChannelID.String())
}
