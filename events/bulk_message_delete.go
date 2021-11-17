package events

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/utils/sendpart"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/dischtml"

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
			Content:   "*[message not in database]*",
			Username:  "unknown#0000",
		})
		notFound++
	}

	// now sort the messages
	sort.Slice(msgs, func(i, j int) bool { return msgs[i].MsgID < msgs[j].MsgID })

	html, err := bot.bulkHTML(ev.GuildID, ev.ChannelID, msgs)
	if err != nil {
		bot.Sugar.Errorf("Error creating HTML output: %v", err)
	} else {
		resp.Files = append(resp.Files, sendpart.File{
			Name:   fmt.Sprintf("bulk-delete-%v-%v.html", ev.ChannelID, time.Now().UTC().Format("2006-01-02T15-04-05")),
			Reader: strings.NewReader(html),
		})
	}

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

	resp.Files = append(resp.Files, sendpart.File{
		Name:   fmt.Sprintf("bulk-delete-%v-%v.txt", ev.ChannelID, time.Now().UTC().Format("2006-01-02T15-04-05")),
		Reader: strings.NewReader(buf),
	})

	resp.Embeds = []discord.Embed{{
		Title:       "Bulk message deletion",
		Description: fmt.Sprintf("%v messages were deleted in %v.\n%v messages archived, %v messages not found.", len(ev.IDs), ev.ChannelID.Mention(), found, notFound),
		Color:       bcr.ColourRed,
		Timestamp:   discord.NowTimestamp(),
	}}

	return resp, nil
}

func (bot *Bot) bulkHTML(guildID discord.GuildID, channelID discord.ChannelID, msgs []*db.Message) (string, error) {
	bot.GuildsMu.RLock()
	g, ok := bot.Guilds[guildID]
	if !ok {
		bot.GuildsMu.RUnlock()
		return "", errors.New("guild not found")
	}
	bot.GuildsMu.RUnlock()

	bot.ChannelsMu.RLock()
	ch, ok := bot.Channels[channelID]
	if !ok {
		bot.ChannelsMu.RUnlock()
		return "", errors.New("channel not found")
	}
	bot.ChannelsMu.RUnlock()

	chans, err := bot.State(g.ID).Channels(g.ID)
	if err != nil {
		return "", err
	}

	rls, err := bot.State(g.ID).Roles(g.ID)
	if err != nil {
		return "", err
	}

	members := bot.GuildMembers(g.ID)
	users := make([]discord.User, len(members))
	for i, m := range members {
		users[i] = m.User
	}

	c := dischtml.Converter{
		Guild:         g,
		Channels:      chans,
		Roles:         rls,
		Members:       members,
		Users:         users,
		ExtraUserInfo: make(map[discord.MessageID]string),
	}

	dm := make([]discord.Message, len(msgs))
	for i, m := range msgs {
		var u discord.User
		var found bool

		for _, user := range users {
			if user.ID == m.UserID {
				u = user
				found = true
				break
			}
		}

		if !found {
			u = discord.User{
				ID:            m.UserID,
				Username:      m.Username,
				Discriminator: "0000",
				Avatar:        "",
			}
		}

		dmsg := discord.Message{
			ID:        m.MsgID,
			ChannelID: m.ChannelID,
			GuildID:   m.ServerID,
			Content:   m.Content,
			Author:    u,
		}

		if m.Metadata != nil {
			dmsg.Embeds = m.Metadata.Embeds

			if len(dmsg.Embeds) > 0 && dmsg.Content == "None" {
				dmsg.Content = ""
			}

			if m.Metadata.UserID != nil {
				if found {
					us := fmt.Sprintf("(%s", u.Tag())
					if m.System != nil && m.Member != nil {
						us += fmt.Sprintf(", system: %v, member: %v", *m.System, *m.Member)
					}

					c.ExtraUserInfo[m.MsgID] = us + ")"
				} else if m.System != nil && m.Member != nil {
					c.ExtraUserInfo[m.MsgID] = fmt.Sprintf("(system: %s, member: %s)", *m.System, *m.Member)
				}

				dmsg.Author.ID = *m.Metadata.UserID
				dmsg.Author.Avatar = m.Metadata.Avatar
				dmsg.Author.Username = m.Metadata.Username
				dmsg.Author.Discriminator = "0000"
			}
		}

		dm[i] = dmsg
	}

	s, err := c.ConvertHTML(dm)
	if err != nil {
		return "", err
	}

	return dischtml.Wrap(g, ch, s, len(dm))
}

// GuildMembers returns all cached members of the given guild.
func (bot *Bot) GuildMembers(id discord.GuildID) (members []discord.Member) {
	bot.MembersMu.RLock()
	defer bot.MembersMu.RUnlock()

	for k, m := range bot.Members {
		if k.GuildID == id {
			members = append(members, m)
		}
	}

	return members
}
