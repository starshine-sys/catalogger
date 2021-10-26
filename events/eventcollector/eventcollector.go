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
