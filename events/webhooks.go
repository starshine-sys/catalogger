package events

import (
	"context"
	"strings"

	"emperror.dev/errors"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/mediocregopher/radix/v4"
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
	GuildKeyRoleUpdate    string
	GuildMemberNickUpdate string
	GuildMemberRemove     string
	GuildMemberKick       string
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
	GuildKeyRoleUpdate:    "GUILD_KEY_ROLE_UPDATE",
	GuildMemberNickUpdate: "GUILD_MEMBER_NICK_UPDATE",
	GuildMemberRemove:     "GUILD_MEMBER_REMOVE",
	GuildMemberKick:       "GUILD_MEMBER_KICK",
	GuildBanAdd:           "GUILD_BAN_ADD",
	GuildBanRemove:        "GUILD_BAN_REMOVE",
	InviteCreate:          "INVITE_CREATE",
	InviteDelete:          "INVITE_DELETE",
	MessageUpdate:         "MESSAGE_UPDATE",
	MessageDelete:         "MESSAGE_DELETE",
	MessageDeleteBulk:     "MESSAGE_DELETE_BULK",
}

// 10 minutes
const webhookCacheExpiry = "600"

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
		err = bot.Redis.Do(context.Background(), radix.Cmd(nil, "DEL", key))
		return nil, errors.Append(ErrInvalid, err)
	}

	whID, err := discord.ParseSnowflake(w[0])
	if err != nil {
		err = bot.Redis.Do(context.Background(), radix.Cmd(nil, "DEL", key))
		return nil, errors.Append(ErrInvalid, err)
	}

	return &Webhook{
		ID:    discord.WebhookID(whID),
		Token: w[1],
	}, nil
}

func (bot *Bot) getWebhook(channelID discord.ChannelID, name string) (*discord.Webhook, error) {
	ws, err := bot.State(0).ChannelWebhooks(channelID)
	if err == nil {
		for _, w := range ws {
			if w.Name == name && (w.User.ID == bot.Router.Bot.ID || !w.User.ID.IsValid()) {
				return &w, nil
			}
		}
	} else {
		return nil, err
	}

	w, err := bot.State(0).CreateWebhook(channelID, api.CreateWebhookData{
		Name: name,
	})
	return w, err
}

func (bot *Bot) webhooksUpdate(ev *gateway.WebhooksUpdateEvent) {
	bot.ResetCache(ev.GuildID, ev.ChannelID)
}

func whKeyNew(id discord.ChannelID) string {
	return "wh:" + id.String()
}

// SetWebhook ...
func (bot *Bot) SetWebhook(channelID discord.ChannelID, w *Webhook) error {
	return bot.Redis.Do(context.Background(), radix.Cmd(nil, "SET", whKeyNew(channelID), w.ID.String()+":"+w.Token, "EX", webhookCacheExpiry))
}

// GetWebhook gets a cached webhook from redis
func (bot *Bot) GetWebhook(id discord.ChannelID) (*Webhook, error) {
	return bot.fetchCachedKey(whKeyNew(id))
}

// ResetCache ...
func (bot *Bot) ResetCache(id discord.GuildID, channels ...discord.ChannelID) {
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

func (bot *Bot) webhookCache(guildID discord.GuildID, ch discord.ChannelID) (*discord.Webhook, error) {
	// try getting the cached webhook
	var wh *discord.Webhook

	w, err := bot.GetWebhook(ch)
	if err != nil {
		bot.Sugar.Debugf("Couldn't find webhook for %v in cache, falling back to fetching webhook", ch)

		if err != ErrNotExists && err != ErrInvalid {
			bot.Sugar.Errorf("Error fetching webhook: %v", err)
		}

		wh, err = bot.getWebhook(ch, bot.Router.Bot.Username)
		if err != nil {
			return nil, err
		}

		err = bot.SetWebhook(ch, &Webhook{
			ID:    wh.ID,
			Token: wh.Token,
		})
		if err != nil {
			bot.Sugar.Errorf("Error setting webhook in redis: %v", err)
		}

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
