package eventcollector

import (
	"reflect"
	"time"

	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/getsentry/sentry-go"
	"github.com/starshine-sys/catalogger/events/handler"
)

// New ...
func New() *handler.SentryHandler {
	h := handler.NewSentry()

	h.AddHandler(messageCreate)
	h.AddHandler(messageDelete)
	h.AddHandler(messageDeleteBulk)
	h.AddHandler(guildBanAdd)
	h.AddHandler(guildBanRemove)
	h.AddHandler(guildMemberAdd)
	h.AddHandler(guildMemberRemove)
	h.AddHandler(guildMemberUpdate)
	h.AddHandler(inviteCreate)
	h.AddHandler(inviteDelete)
	h.AddHandler(channelCreate)
	h.AddHandler(channelDelete)
	h.AddHandler(channelUpdate)
	h.AddHandler(roleCreate)
	h.AddHandler(roleUpdate)
	h.AddHandler(roleDelete)

	return h
}

func messageCreate(hub *sentry.Hub, ev *gateway.MessageCreateEvent) {
	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Data: map[string]interface{}{
			"event":      reflect.ValueOf(ev).Elem().Type().Name(),
			"guild_id":   ev.GuildID,
			"channel_id": ev.ChannelID,
			"user_id":    ev.Author.ID,
			"is_webhook": ev.WebhookID.IsValid(),
		},
		Level:     sentry.LevelError,
		Timestamp: time.Now().UTC(),
	}, nil)
}

func messageDelete(hub *sentry.Hub, ev *gateway.MessageDeleteEvent) {
	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Data: map[string]interface{}{
			"event":      reflect.ValueOf(ev).Elem().Type().Name(),
			"guild_id":   ev.GuildID,
			"channel_id": ev.ChannelID,
		},
		Level:     sentry.LevelError,
		Timestamp: time.Now().UTC(),
	}, nil)
}

func messageDeleteBulk(hub *sentry.Hub, ev *gateway.MessageDeleteBulkEvent) {
	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Data: map[string]interface{}{
			"event":      reflect.ValueOf(ev).Elem().Type().Name(),
			"guild_id":   ev.GuildID,
			"channel_id": ev.ChannelID,
		},
		Level:     sentry.LevelError,
		Timestamp: time.Now().UTC(),
	}, nil)
}

func guildBanAdd(hub *sentry.Hub, ev *gateway.GuildBanAddEvent) {
	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Data: map[string]interface{}{
			"event":    reflect.ValueOf(ev).Elem().Type().Name(),
			"guild_id": ev.GuildID,
			"user_id":  ev.User.ID,
		},
		Level:     sentry.LevelError,
		Timestamp: time.Now().UTC(),
	}, nil)
}

func guildBanRemove(hub *sentry.Hub, ev *gateway.GuildBanRemoveEvent) {
	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Data: map[string]interface{}{
			"event":    reflect.ValueOf(ev).Elem().Type().Name(),
			"guild_id": ev.GuildID,
			"user_id":  ev.User.ID,
		},
		Level:     sentry.LevelError,
		Timestamp: time.Now().UTC(),
	}, nil)
}

func guildMemberAdd(hub *sentry.Hub, ev *gateway.GuildMemberAddEvent) {
	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Data: map[string]interface{}{
			"event":    reflect.ValueOf(ev).Elem().Type().Name(),
			"guild_id": ev.GuildID,
			"user_id":  ev.User.ID,
		},
		Level:     sentry.LevelError,
		Timestamp: time.Now().UTC(),
	}, nil)
}

func guildMemberRemove(hub *sentry.Hub, ev *gateway.GuildMemberRemoveEvent) {
	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Data: map[string]interface{}{
			"event":    reflect.ValueOf(ev).Elem().Type().Name(),
			"guild_id": ev.GuildID,
			"user_id":  ev.User.ID,
		},
		Level:     sentry.LevelError,
		Timestamp: time.Now().UTC(),
	}, nil)
}

func guildMemberUpdate(hub *sentry.Hub, ev *gateway.GuildMemberUpdateEvent) {
	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Data: map[string]interface{}{
			"event":    reflect.ValueOf(ev).Elem().Type().Name(),
			"guild_id": ev.GuildID,
			"user_id":  ev.User.ID,
		},
		Level:     sentry.LevelError,
		Timestamp: time.Now().UTC(),
	}, nil)
}

func inviteCreate(hub *sentry.Hub, ev *gateway.InviteCreateEvent) {
	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Data: map[string]interface{}{
			"event":    reflect.ValueOf(ev).Elem().Type().Name(),
			"guild_id": ev.GuildID,
		},
		Level:     sentry.LevelError,
		Timestamp: time.Now().UTC(),
	}, nil)
}

func inviteDelete(hub *sentry.Hub, ev *gateway.InviteDeleteEvent) {
	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Data: map[string]interface{}{
			"event":    reflect.ValueOf(ev).Elem().Type().Name(),
			"guild_id": ev.GuildID,
		},
		Level:     sentry.LevelError,
		Timestamp: time.Now().UTC(),
	}, nil)
}

func channelCreate(hub *sentry.Hub, ev *gateway.ChannelCreateEvent) {
	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Data: map[string]interface{}{
			"event":      reflect.ValueOf(ev).Elem().Type().Name(),
			"guild_id":   ev.GuildID,
			"channel_id": ev.ID,
		},
		Level:     sentry.LevelError,
		Timestamp: time.Now().UTC(),
	}, nil)
}

func channelUpdate(hub *sentry.Hub, ev *gateway.ChannelUpdateEvent) {
	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Data: map[string]interface{}{
			"event":      reflect.ValueOf(ev).Elem().Type().Name(),
			"guild_id":   ev.GuildID,
			"channel_id": ev.ID,
		},
		Level:     sentry.LevelError,
		Timestamp: time.Now().UTC(),
	}, nil)
}

func channelDelete(hub *sentry.Hub, ev *gateway.ChannelDeleteEvent) {
	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Data: map[string]interface{}{
			"event":      reflect.ValueOf(ev).Elem().Type().Name(),
			"guild_id":   ev.GuildID,
			"channel_id": ev.ID,
		},
		Level:     sentry.LevelError,
		Timestamp: time.Now().UTC(),
	}, nil)
}

func roleCreate(hub *sentry.Hub, ev *gateway.GuildRoleCreateEvent) {
	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Data: map[string]interface{}{
			"event":    reflect.ValueOf(ev).Elem().Type().Name(),
			"guild_id": ev.GuildID,
			"role_id":  ev.Role.ID,
		},
		Level:     sentry.LevelError,
		Timestamp: time.Now().UTC(),
	}, nil)
}

func roleUpdate(hub *sentry.Hub, ev *gateway.GuildRoleUpdateEvent) {
	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Data: map[string]interface{}{
			"event":    reflect.ValueOf(ev).Elem().Type().Name(),
			"guild_id": ev.GuildID,
			"role_id":  ev.Role.ID,
		},
		Level:     sentry.LevelError,
		Timestamp: time.Now().UTC(),
	}, nil)
}

func roleDelete(hub *sentry.Hub, ev *gateway.GuildRoleDeleteEvent) {
	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Data: map[string]interface{}{
			"event":    reflect.ValueOf(ev).Elem().Type().Name(),
			"guild_id": ev.GuildID,
			"role_id":  ev.RoleID,
		},
		Level:     sentry.LevelError,
		Timestamp: time.Now().UTC(),
	}, nil)
}
