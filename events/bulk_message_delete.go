package events

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/utils/sendpart"
	"github.com/starshine-sys/bcr"

	"github.com/starshine-sys/catalogger/db"
	"github.com/starshine-sys/catalogger/events/handler"
)

func (bot *Bot) bulkMessageDelete(ev *gateway.MessageDeleteBulkEvent) (resp *handler.Response, err error) {
	s, _ := bot.Router.StateFromGuildID(ev.GuildID)

	if !ev.GuildID.IsValid() {
		return
	}

	channel, err := bot.State(ev.GuildID).Channel(ev.ChannelID)
	if err != nil {
		return nil, err
	}

	ch, err := bot.DB.Channels(ev.GuildID)
	if err != nil {
		return nil, err
	}

	if !ch[keys.MessageDeleteBulk].IsValid() {
		return
	}

	// if the channels is blacklisted, return
	channelID := ev.ChannelID
	if channel.Type == discord.GuildNewsThread || channel.Type == discord.GuildPrivateThread || channel.Type == discord.GuildPublicThread {
		channelID = channel.ParentID
	}
	var blacklisted bool
	if bot.DB.QueryRow(context.Background(), "select exists(select id from guilds where $1 = any(ignored_channels) and id = $2)", channelID, ev.GuildID).Scan(&blacklisted); blacklisted {
		return
	}

	resp = &handler.Response{
		ChannelID: ch[keys.MessageDeleteBulk],
	}

	redirects, err := bot.DB.Redirects(ev.GuildID)
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   keys.MessageDeleteBulk,
			GuildID: ev.GuildID,
		}, err)
		return
	}

	if redirects[channelID.String()].IsValid() {
		resp.ChannelID = redirects[channelID.String()]
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
				u, err := s.User(m.UserID)
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
		// else add a dummy message with the ID
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
		if m.Member != nil && m.System != nil {
			s = fmt.Sprintf(`[%v | %v] %v (%v)
PK system: %v / PK member: %v
--------------------------------------------
%v

--------------------------------------------
`,
				m.MsgID.Time().Format(time.ANSIC), m.MsgID, m.Username, m.UserID, *m.System, *m.Member, m.Content,
			)
		}

		buf += s
	}

	resp.Files = []sendpart.File{{
		Name:   fmt.Sprintf("bulk-delete-%v-%v.txt", ev.ChannelID, time.Now().UTC().Format("2006-01-02T15-04-05")),
		Reader: strings.NewReader(buf),
	}}

	resp.Embeds = []discord.Embed{{
		Title:       "Bulk message deletion",
		Description: fmt.Sprintf("%v messages were deleted in %v.\n%v messages archived, %v messages not found.", len(ev.IDs), ev.ChannelID.Mention(), found, notFound),
		Color:       bcr.ColourRed,
		Timestamp:   discord.NowTimestamp(),
	}}

	return resp, nil
}
