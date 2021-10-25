package events

import (
	"reflect"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/api/webhook"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/starshine-sys/catalogger/db"
	"github.com/starshine-sys/catalogger/events/handler"
)

// shouldQueue is a map of all events that should be put into a webhook queue
// doesn't need a mutex because it's never modified
var shouldQueue = map[string]bool{
	keys.GuildMemberUpdate:     true,
	keys.GuildMemberNickUpdate: true,
	keys.GuildKeyRoleUpdate:    true,
	keys.MessageDelete:         true,
	keys.MessageUpdate:         true,
	keys.GuildMemberAdd:        true,
	keys.GuildMemberRemove:     true,
	keys.GuildBanAdd:           true,
	keys.GuildBanRemove:        true,
	keys.InviteCreate:          true,
	keys.InviteDelete:          true,
}

// Send either sends a slice of embeds immediately, or queues a single embed
func (bot *Bot) Send(wh *discord.Webhook, event string, embeds ...discord.Embed) {
	if len(embeds) == 0 {
		return
	}

	if shouldQueue[event] && len(embeds) == 1 {
		bot.Queue(wh, event, embeds[0])
		return
	}

	bot.Sugar.Debugf("Event for webhook %v should not be queued, sending embed", wh.ID)

	client := bot.WebhookClient(wh)

	err := client.Execute(webhook.ExecuteData{
		AvatarURL: bot.Router.Bot.AvatarURL(),
		Embeds:    embeds,
	})
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   event,
			GuildID: wh.GuildID,
		}, err)
	}
}

// Queue is a webhook embed queue.
type Queue struct {
	mu      sync.Mutex
	queue   []discord.Embed
	timer   *time.Timer
	errFunc func(v ...interface{})
}

// NewQueue returns a new Queue
func NewQueue(f func(v ...interface{})) *Queue {
	if f == nil {
		f = func(v ...interface{}) { return }
	}

	return &Queue{
		errFunc: f,
	}
}

// WebhookClient gets a client for the given webhook.
func (bot *Bot) WebhookClient(wh *discord.Webhook) *webhook.Client {
	bot.WebhooksMu.Lock()
	defer bot.WebhooksMu.Unlock()

	client, ok := bot.WebhookClients[wh.ID]
	if !ok {
		bot.Sugar.Debugf("Creating new webhook client for %v", wh.ID)

		// always get the first state
		s, _ := bot.Router.StateFromGuildID(0)

		client = webhook.FromAPI(wh.ID, wh.Token, s.Client)
		bot.WebhookClients[wh.ID] = client
	}
	return client
}

// TotalLength returns the total length of all embeds in this Queue.
func (q *Queue) TotalLength() (length int) {
	for _, e := range q.queue {
		length += e.Length()
	}
	return length
}

// Queue queues an embed.
func (bot *Bot) Queue(wh *discord.Webhook, event string, embed discord.Embed) {
	bot.QueueMu.Lock()
	q, ok := bot.Queues[wh.ID]
	if !ok {
		bot.Sugar.Debugf("Creating new embed queue for %v", wh.ID)

		q = NewQueue(bot.Sugar.Error)
		bot.Queues[wh.ID] = q
	}
	bot.QueueMu.Unlock()

	client := bot.WebhookClient(wh)

	bot.Sugar.Debugf("Adding embed to queue for %v", wh.ID)

	q.mu.Lock()
	defer q.mu.Unlock()
	if q.TotalLength()+embed.Length() >= 6000 || len(q.queue) >= 5 {
		embeds := q.queue
		q.queue = nil

		if q.timer != nil {
			q.timer.Stop()
			q.timer = nil
		}

		if err := bot.queueInner(client, embeds); err != nil {
			bot.DB.Report(db.ErrorContext{
				Event:   event,
				GuildID: wh.GuildID,
			}, err)
			return
		}
	}

	q.queue = append(q.queue, embed)

	if q.timer == nil {
		q.timer = time.AfterFunc(5*time.Second, func() {
			q.mu.Lock()
			q.timer = nil
			embeds := q.queue
			q.queue = nil
			q.mu.Unlock()

			if len(embeds) == 0 {
				return
			}

			if err := bot.queueInner(client, embeds); err != nil {
				q.errFunc("Error executing queue:", err)
				bot.DB.Report(db.ErrorContext{
					Event:   event,
					GuildID: wh.GuildID,
				}, err)
				return
			}
		})
	}
}

func (bot *Bot) queueInner(client *webhook.Client, embeds []discord.Embed) (err error) {
	bot.Sugar.Debugf("Executing webhook %v, with %v embed(s)", client.ID, len(embeds))

	_, err = client.ExecuteAndWait(webhook.ExecuteData{
		AvatarURL: bot.Router.Bot.AvatarURL(),
		Embeds:    embeds,
		// won't ping anyway because it's all embeds, but can't hurt
		AllowedMentions: &api.AllowedMentions{
			Parse: []api.AllowedMentionType{},
		},
	})
	return
}

func (bot *Bot) handleResponse(ev reflect.Value, resp *handler.Response) {
	if !resp.ChannelID.IsValid() {
		return
	}

	evName := ev.Elem().Type().Name()

	wh, err := bot.webhookCacheNew(resp.GuildID, resp.ChannelID)
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   evName,
			GuildID: resp.GuildID,
		}, err)
		return
	}

	bot.Send(wh, evName, resp.Embeds...)
}
