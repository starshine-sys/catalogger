package events

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/api/webhook"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/common"
	"github.com/starshine-sys/catalogger/db"
	"github.com/starshine-sys/catalogger/events/handler"
)

// shouldQueue is a map of all events that should be put into a webhook queue
// doesn't need a mutex because it's never modified
// yes this is a mess of reflection, but at least that means we'll get a compiler error if any of these types change
var shouldQueue = map[string]bool{
	reflect.ValueOf(&gateway.GuildMemberUpdateEvent{}).Elem().Type().Name(): true,
	reflect.ValueOf(&GuildKeyRoleUpdateEvent{}).Elem().Type().Name():        true,
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
	reflect.ValueOf(&MemberKickEvent{}).Elem().Type().Name():                true,
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

	common.Log.Debugf("Event for webhook %v should not be queued, sending embed", wh.ID)

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
	mu    sync.Mutex
	queue []discord.Embed
	timer *time.Timer
}

// NewQueue returns a new Queue
func NewQueue() *Queue {
	return &Queue{}
}

// WebhookClient gets a client for the given webhook.
func (bot *Bot) WebhookClient(wh *discord.Webhook) *webhook.Client {
	bot.WebhooksMu.Lock()
	defer bot.WebhooksMu.Unlock()

	client, ok := bot.WebhookClients[wh.ID]
	if !ok {
		common.Log.Debugf("Creating new webhook client for %v", wh.ID)

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
		common.Log.Debugf("Creating new embed queue for %v", wh.ID)

		q = NewQueue()
		bot.Queues[wh.ID] = q
	}
	bot.QueueMu.Unlock()

	client := bot.WebhookClient(wh)

	common.Log.Debugf("Adding embed to queue for %v", wh.ID)

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
				common.Log.Error("Error executing queue:", err)
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
	common.Log.Debugf("Executing webhook %v, with %v embed(s)", client.ID, len(embeds))

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

	if len(resp.Embeds) == 0 {
		common.Log.Infof("Response for event %v was not nil, but has no embeds", evName)
		return
	}

	wh, err := bot.webhookCache(resp.GuildID, resp.ChannelID)
	if err != nil {
		switch v := err.(type) {
		case *httputil.HTTPError:
			common.Log.Infof("HTTP error sending log in %v: %v", resp.ChannelID, err)

			if v.Status == 403 {
				bot.sendUnauthorizedError(resp)
				return
			}
		default:
			bot.DB.Report(db.ErrorContext{
				Event:   evName,
				GuildID: resp.GuildID,
			}, err)
		}

		return
	}

	if len(resp.Files) != 0 {
		client := bot.WebhookClient(wh)

		_, err = client.ExecuteAndWait(webhook.ExecuteData{
			AvatarURL: bot.Router.Bot.AvatarURL(),
			Embeds:    resp.Embeds,
			Files:     resp.Files,
			// won't ping anyway because it's all embeds, but can't hurt
			AllowedMentions: &api.AllowedMentions{
				Parse: []api.AllowedMentionType{},
			},
		})
		if err != nil {
			bot.handleError(ev, err)
		}
		return
	}

	bot.Send(wh, evName, resp.Embeds...)
}

func (bot *Bot) shouldSendError(ch discord.ChannelID) bool {
	bot.UnauthorizedErrorsMu.Lock()
	defer bot.UnauthorizedErrorsMu.Unlock()

	if bot.UnauthorizedErrorsSent[ch].Before(time.Now().Add(-10 * time.Minute)) {
		bot.UnauthorizedErrorsSent[ch] = time.Now()

		return true
	}

	return false
}

func (bot *Bot) sendUnauthorizedError(resp *handler.Response) {
	if !bot.shouldSendError(resp.ChannelID) {
		return
	}

	_, err := bot.State(resp.GuildID).SendEmbeds(resp.ChannelID, discord.Embed{
		Color:       bcr.ColourRed,
		Description: fmt.Sprintf("%v does not have the **Manage Webhooks** permission in this channel, and thus cannot send log messages.\nCheck this server's permissions with `/permcheck`.", bot.Router.Bot.Username),
	})
	if err != nil {
		common.Log.Errorf("Error sending unauthorized message to %v: %v", resp.ChannelID, err)
	}
}
