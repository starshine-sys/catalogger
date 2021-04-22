package events

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/v2/api/webhook"
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/arikawa/v2/utils/sendpart"
	"github.com/starshine-sys/bcr"

	"git.sr.ht/~starshine-sys/logger/db"
)

func (bot *Bot) bulkMessageDelete(ev *gateway.MessageDeleteBulkEvent) {
	if !ev.GuildID.IsValid() {
		return
	}

	ch, err := bot.DB.Channels(ev.GuildID)
	if err != nil {
		bot.Sugar.Errorf("Error getting channels for %v: %v", ev.GuildID, err)
		return
	}

	if !ch["MESSAGE_DELETE_BULK"].IsValid() {
		return
	}

	// if the channels is blacklisted, return
	var blacklisted bool
	if bot.DB.Pool.QueryRow(context.Background(), "select exists(select id from guilds where $1 = any(ignored_channels) and id = $2)", ev.ChannelID, ev.GuildID).Scan(&blacklisted); blacklisted {
		return
	}

	wh, err := bot.webhookCache("message_delete_bulk", ev.GuildID, ch["MESSAGE_DELETE_BULK"])
	if err != nil {
		bot.Sugar.Errorf("Error getting webhook: %v", err)
		return
	}

	var msgs []*db.Message
	var found, notFound int
	users := map[discord.UserID]*discord.User{}

	for _, id := range ev.IDs {
		// first, try getting the ID of a normal message
		m, err := bot.DB.GetMessage(id)
		if err == nil && m.UserID != 0 {
			if u, ok := users[m.UserID]; ok {
				m.Username = u.Username + "#" + u.Discriminator
			} else {
				u, err := bot.State.User(m.UserID)
				if err == nil {
					m.Username = u.Username + "#" + u.Discriminator
					users[u.ID] = u
				} else {
					m.Username = "unknown#0000"
				}
			}

			msgs = append(msgs, m)
			found++
			continue
		}
		// else check if it's proxied
		m, err = bot.DB.GetProxied(id)
		if err == nil && m.UserID != 0 {
			if u, ok := users[m.UserID]; ok {
				m.Username = fmt.Sprintf("%v (%v#%v)", m.Username, u.Username, u.Discriminator)
			} else {
				u, err := bot.State.User(m.UserID)
				if err == nil {
					m.Username = fmt.Sprintf("%v (%v#%v)", m.Username, u.Username, u.Discriminator)
					users[u.ID] = u
				}
			}

			msgs = append(msgs, m)
			found++
			continue
		}
		// add a dummy message with the ID
		msgs = append(msgs, &db.Message{
			MsgID:     id,
			ChannelID: ev.ChannelID,
			ServerID:  ev.GuildID,
			Content:   "<message not in database>",
			Username:  "unknown#0000",
		})
		notFound++
	}

	// now sort the messages
	sort.Slice(msgs, func(i, j int) bool { return msgs[i].MsgID < msgs[j].MsgID })

	var buf string
	for _, m := range msgs {
		s := fmt.Sprintf(`[%v | %v] %v (%v)
--------------------------------------------
%v

--------------------------------------------
`,
			m.MsgID.Time().Format(time.ANSIC), m.MsgID, m.Username, m.UserID, m.Content,
		)
		if m.Member != "" {
			s = fmt.Sprintf(`[%v | %v] %v (%v)
PK system: %v / PK member: %v
--------------------------------------------
%v

--------------------------------------------
`,
				m.MsgID.Time().Format(time.ANSIC), m.MsgID, m.Username, m.UserID, m.System, m.Member, m.Content,
			)
		}

		buf += s
	}

	file := sendpart.File{
		Name:   fmt.Sprintf("bulk-delete-%v-%v.txt", ev.ChannelID, time.Now().UTC().Format("2006-01-02T15-04-05")),
		Reader: strings.NewReader(buf),
	}

	e := discord.Embed{
		Title:       "Bulk message deletion",
		Description: fmt.Sprintf("%v messages were deleted in %v.\n%v messages archived, %v messages not found.", len(ev.IDs), ev.ChannelID.Mention(), found, notFound),
		Color:       bcr.ColourRed,
		Timestamp:   discord.NowTimestamp(),
	}

	webhook.New(wh.ID, wh.Token).ExecuteAndWait(webhook.ExecuteData{
		AvatarURL: bot.Router.Bot.AvatarURL(),
		Embeds:    []discord.Embed{e},
		Files:     []sendpart.File{file},
	})
}