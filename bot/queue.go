package bot

import (
	"reflect"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/api/webhook"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/utils/sendpart"
	"github.com/starshine-sys/catalogger/v2/common/log"
)

type SendData struct {
	// If ChanneLID is not valid, the channel ID is fetched by bot.Send.
	// This should only be set if the log channel ID is checked in the log handler.
	ChannelID discord.ChannelID

	Embeds []discord.Embed
	Files  []sendpart.File
}

// Send either sends a slice of embeds immediately, or queues a single embed.
// `event` should either be the event received in the handler, or a string name.
func (bot *Bot) Send(
	guildID discord.GuildID,
	event any,
	data SendData,
) {
	if len(data.Embeds) == 0 {
		return
	}

	// get event name
	eventName, ok := event.(string)
	if !ok {
		eventName = bot.eventName(event)
	}

	// get channel ID, if not set
	channelID := data.ChannelID
	if !channelID.IsValid() {
		channels, err := bot.DB.Channels(guildID)
		if err != nil {
			log.Errorf("getting channels for guild %v: %v", guildID, err)
		}

		channelID = channels.Channels.For(eventName)
		if !channelID.IsValid() {
			log.Debugf("event %v in guild %v has no valid channel", eventName, guildID)
			return
		}
	}

	// get webhook
	wh, err := bot.getWebhook(channelID)
	if err != nil {
		log.Errorf("getting webhook for channel %v: %v", channelID, err)
		return
	}

	// if the event should be queued to be sent in bulk, queue it and return
	if shouldQueue[eventName] && len(data.Embeds) == 1 && len(data.Files) == 0 {
		bot.queue(wh, eventName, data.Embeds[0])
		return
	}

	log.Debugf("Event for webhook %v should not be queued, sending embed", wh.ID)

	client := bot.webhookClient(wh)

	err = client.Execute(webhook.ExecuteData{
		AvatarURL: bot.user.AvatarURL(),
		Embeds:    data.Embeds,
		Files:     data.Files,
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
	// Internal events
	reflect.ValueOf(&gateway.ReadyEvent{}).Elem().Type().Name():       true,
	reflect.ValueOf(&gateway.GuildCreateEvent{}).Elem().Type().Name(): true,
	reflect.ValueOf(&gateway.GuildDeleteEvent{}).Elem().Type().Name(): true,
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
