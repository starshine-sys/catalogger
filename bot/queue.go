package bot

import (
	"reflect"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/api/webhook"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/catalogger/v2/common/log"
)

// Send either sends a slice of embeds immediately, or queues a single embed.
// `event` should either be the event received in the handler, or a string name.
func (bot *Bot) Send(wh *discord.Webhook, event any, embeds ...discord.Embed) {
	if len(embeds) == 0 {
		return
	}

	// get event name
	eventName, ok := event.(string)
	if !ok {
		eventName = bot.eventName(event)
	}

	// if the event should be queued to be sent in bulk, queue it and return
	if shouldQueue[eventName] && len(embeds) == 1 {
		bot.queue(wh, eventName, embeds[0])
		return
	}

	log.Debugf("Event for webhook %v should not be queued, sending embed", wh.ID)

	client := bot.webhookClient(wh)

	err := client.Execute(webhook.ExecuteData{
		AvatarURL: bot.user.AvatarURL(),
		Embeds:    embeds,
	})
	if err != nil {
		log.Errorf("executing webhook %v: %v", wh.ID, err)
	}
}

// shouldQueue is a map of all events that should be put into a webhook queue
var shouldQueue = map[string]bool{
	reflect.ValueOf(&gateway.GuildMemberUpdateEvent{}).Elem().Type().Name(): true,
	reflect.ValueOf(&gateway.MessageDeleteEvent{}).Elem().Type().Name():     true,
	reflect.ValueOf(&gateway.MessageUpdateEvent{}).Elem().Type().Name():     true,
	reflect.ValueOf(&gateway.GuildMemberAddEvent{}).Elem().Type().Name():    true,
	reflect.ValueOf(&gateway.GuildMemberRemoveEvent{}).Elem().Type().Name(): true,
	reflect.ValueOf(&gateway.GuildBanAddEvent{}).Elem().Type().Name():       true,
	reflect.ValueOf(&gateway.GuildBanRemoveEvent{}).Elem().Type().Name():    true,
	reflect.ValueOf(&gateway.ChannelCreateEvent{}).Elem().Type().Name():     true,
	reflect.ValueOf(&gateway.ChannelDeleteEvent{}).Elem().Type().Name():     true,
	reflect.ValueOf(&gateway.ChannelUpdateEvent{}).Elem().Type().Name():     true,
	reflect.ValueOf(&gateway.InviteCreateEvent{}).Elem().Type().Name():      true,
	reflect.ValueOf(&gateway.InviteDeleteEvent{}).Elem().Type().Name():      true,
}

// queue is a webhook embed queue.
type queue struct {
	mu    sync.Mutex
	queue []discord.Embed
	timer *time.Timer
}

// totalLength returns the total length of all embeds in this Queue.
func (q *queue) totalLength() (length int) {
	for _, e := range q.queue {
		length += e.Length()
	}
	return length
}

// newQueue returns a new Queue
func newQueue() *queue {
	return &queue{}
}

// eventName returns the event's type name
func (*Bot) eventName(i any) string {
	return reflect.ValueOf(i).Elem().Type().Name()
}

// queue queues an embed.
func (bot *Bot) queue(wh *discord.Webhook, event string, embed discord.Embed) {
	bot.queuesMu.Lock()
	q, ok := bot.queues[wh.ID]
	if !ok {
		log.Debugf("creating new embed queue for %v", wh.ID)
		q = newQueue()
		bot.queues[wh.ID] = q
	}
	bot.queuesMu.Unlock()

	client := bot.webhookClient(wh)

	log.Debugf("Adding embed to queue for %v", wh.ID)

	q.mu.Lock()
	defer q.mu.Unlock()
	if q.totalLength()+embed.Length() >= 6000 || len(q.queue) >= 5 {
		embeds := q.queue
		q.queue = nil

		if q.timer != nil {
			q.timer.Stop()
			q.timer = nil
		}

		if err := bot.queueInner(client, embeds); err != nil {
			log.Errorf("executing queue for %v: %v", wh.ID, err)
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
				log.Errorf("executing queue for %v: %v", wh.ID, err)
				return
			}
		})
	}
}

func (bot *Bot) queueInner(client *webhook.Client, embeds []discord.Embed) (err error) {
	log.Debugf("Executing webhook %v, with %v embed(s)", client.ID, len(embeds))

	_, err = client.ExecuteAndWait(webhook.ExecuteData{
		AvatarURL: bot.user.AvatarURL(),
		Embeds:    embeds,
		// won't ping anyway because it's all embeds, but can't hurt
		AllowedMentions: &api.AllowedMentions{
			Parse: []api.AllowedMentionType{},
		},
	})
	return
}
