package events

import (
	"sync"
	"time"

	"github.com/diamondburned/arikawa/v3/api/webhook"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/starshine-sys/catalogger/db"
)

// Queue is a webhook embed queue.
type Queue struct {
	mu    sync.Mutex
	queue []discord.Embed
	timer *time.Timer
}

// TotalLength returns the total length of all embeds in this Queue.
func (q *Queue) TotalLength() (length int) {
	for _, e := range q.queue {
		length += e.Length()
	}
	return length
}

// Queue queues an embed.
func (bot *Bot) Queue(wh *discord.Webhook, event string, client *webhook.Client, embed discord.Embed) {
	bot.QueueMu.Lock()
	q, ok := bot.Queues[wh.ChannelID]
	if !ok {
		q = &Queue{}
		bot.Queues[wh.ChannelID] = q
	}
	bot.QueueMu.Unlock()

	q.mu.Lock()
	defer q.mu.Unlock()
	if q.TotalLength()+embed.Length() >= 6000 || len(q.queue) > 5 {
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
	_, err = client.ExecuteAndWait(webhook.ExecuteData{
		AvatarURL: bot.Router.Bot.AvatarURL(),
		Embeds:    embeds,
	})
	return
}
