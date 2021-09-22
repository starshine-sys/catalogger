package events

import (
	"context"
	"strings"

	"emperror.dev/errors"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/mediocregopher/radix/v4"
	"github.com/starshine-sys/catalogger/db"
)

// Errors for the webhook cache
const (
	ErrNotExists = errors.Sentinel("webhook not found in cache")
	ErrInvalid   = errors.Sentinel("invalid cache value")
)

// Webhook ...
type Webhook struct {
	ID    discord.WebhookID
	Token string
}

var keys = struct {
	GuildUpdate           string
	GuildEmojisUpdate     string
	GuildRoleCreate       string
	GuildRoleUpdate       string
	GuildRoleDelete       string
	ChannelCreate         string
	ChannelUpdate         string
	ChannelDelete         string
	GuildMemberAdd        string
	GuildMemberUpdate     string
	GuildMemberNickUpdate string
	GuildMemberRemove     string
	GuildBanAdd           string
	GuildBanRemove        string
	InviteCreate          string
	InviteDelete          string
	MessageUpdate         string
	MessageDelete         string
	MessageDeleteBulk     string
}{
	GuildUpdate:           "GUILD_UPDATE",
	GuildEmojisUpdate:     "GUILD_EMOJIS_UPDATE",
	GuildRoleCreate:       "GUILD_ROLE_CREATE",
	GuildRoleUpdate:       "GUILD_ROLE_UPDATE",
	GuildRoleDelete:       "GUILD_ROLE_DELETE",
	ChannelCreate:         "CHANNEL_CREATE",
	ChannelUpdate:         "CHANNEL_UPDATE",
	ChannelDelete:         "CHANNEL_DELETE",
	GuildMemberAdd:        "GUILD_MEMBER_ADD",
	GuildMemberUpdate:     "GUILD_MEMBER_UPDATE",
	GuildMemberNickUpdate: "GUILD_MEMBER_NICK_UPDATE",
	GuildMemberRemove:     "GUILD_MEMBER_REMOVE",
	GuildBanAdd:           "GUILD_BAN_ADD",
	GuildBanRemove:        "GUILD_BAN_REMOVE",
	InviteCreate:          "INVITE_CREATE",
	InviteDelete:          "INVITE_DELETE",
	MessageUpdate:         "MESSAGE_UPDATE",
	MessageDelete:         "MESSAGE_DELETE",
	MessageDeleteBulk:     "MESSAGE_DELETE_BULK",
}

func whKey(key string, id discord.GuildID) string {
	return "wh:" + key + ":" + id.String()
}

func redirKey(channelID discord.ChannelID) string {
	return "wh:redir:" + channelID.String()
}

// 10 minutes
const webhookCacheExpiry = "600"

// SetWebhook ...
func (bot *Bot) SetWebhook(key string, guildID discord.GuildID, w *Webhook) error {
	return bot.Redis.Do(context.Background(), radix.Cmd(nil, "SET", whKey(key, guildID), w.ID.String()+":"+w.Token, "EX", webhookCacheExpiry))
}

func (bot *Bot) setRedirWebhook(chID discord.ChannelID, w *Webhook) error {
	return bot.Redis.Do(context.Background(), radix.Cmd(nil, "SET", redirKey(chID), w.ID.String()+":"+w.Token, "EX", webhookCacheExpiry))
}

// GetWebhook gets a cached webhook from redis
func (bot *Bot) GetWebhook(key string, id discord.GuildID) (*Webhook, error) {
	return bot.fetchCachedKey(whKey(key, id))
}

func (bot *Bot) fetchCachedKey(key string) (*Webhook, error) {
	var s string
	err := bot.Redis.Do(context.Background(), radix.Cmd(&s, "GET", key))
	if err != nil {
		return nil, err
	}
	if s == "" {
		return nil, ErrNotExists
	}

	w := strings.SplitN(s, ":", 2)
	if len(w) != 2 {
		bot.Redis.Do(context.Background(), radix.Cmd(nil, "DEL", key))
		return nil, ErrInvalid
	}

	whID, err := discord.ParseSnowflake(w[0])
	if err != nil {
		bot.Redis.Do(context.Background(), radix.Cmd(nil, "DEL", key))
		return nil, ErrInvalid
	}

	return &Webhook{
		ID:    discord.WebhookID(whID),
		Token: w[1],
	}, nil
}

// ResetCache ...
func (bot *Bot) ResetCache(id discord.GuildID, channels ...discord.ChannelID) {
	var keys []string

	for _, ev := range db.Events {
		keys = append(keys, whKey(ev, id))
	}

	for _, ch := range channels {
		keys = append(keys, redirKey(ch))
	}

	err := bot.Redis.Do(context.Background(), radix.Cmd(nil, "DEL", keys...))
	if err != nil {
		bot.Sugar.Errorf("Error resetting webhook cache: %v", err)
	}
}

func (bot *Bot) getWebhook(guildID discord.GuildID, channelID discord.ChannelID, name string) (*discord.Webhook, error) {
	ws, err := bot.State(guildID).ChannelWebhooks(channelID)
	if err == nil {
		for _, w := range ws {
			if w.Name == name && (w.User.ID == bot.Router.Bot.ID || !w.User.ID.IsValid()) {
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

	w, err := bot.GetWebhook(t, guildID)
	if err != nil {
		bot.Sugar.Debugf("Couldn't find webhook for %v in cache, falling back to fetching webhook", ch)

		if err != ErrNotExists && err != ErrInvalid {
			bot.Sugar.Errorf("Error fetching webhook: %v", err)
		}

		wh, err = bot.getWebhook(guildID, ch, bot.Router.Bot.Username)
		if err != nil {
			return nil, err
		}

		bot.SetWebhook(t, guildID, &Webhook{
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

	return wh, nil
}

func (bot *Bot) getRedirect(guildID discord.GuildID, ch discord.ChannelID) (*discord.Webhook, error) {
	// try getting the cached webhook
	w, err := bot.fetchCachedKey(redirKey(ch))
	if err == nil {
		return &discord.Webhook{
			ID:        w.ID,
			Token:     w.Token,
			ChannelID: ch,
		}, nil
	}

	bot.Sugar.Debugf("Couldn't find webhook for %v in cache, falling back to fetching webhook", ch)

	// else, create or fetch webhook
	wh, err := bot.getWebhook(guildID, ch, bot.Router.Bot.Username)
	if err != nil {
		return nil, err
	}

	err = bot.setRedirWebhook(ch, &Webhook{
		ID:    wh.ID,
		Token: wh.Token,
	})
	if err != nil {
		bot.Sugar.Errorf("Error setting redirect webhook for %v: %v", ch, err)
	}

	return wh, nil
}

func (bot *Bot) webhooksUpdate(ev *gateway.WebhooksUpdateEvent) {
	bot.ResetCache(ev.GuildID, ev.ChannelID)
}
