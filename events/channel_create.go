package events

import (
	"fmt"
	"strings"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/events/handler"
)

func (bot *Bot) channelCreate(ev *gateway.ChannelCreateEvent) (resp *handler.Response, err error) {
	bot.ChannelsMu.Lock()
	bot.Channels[ev.ID] = ev.Channel
	bot.ChannelsMu.Unlock()

	ch, err := bot.DB.Channels(ev.GuildID)
	if err != nil {
		return
	}
	if !ch[keys.ChannelCreate].IsValid() {
		return
	}

	resp = &handler.Response{
		ChannelID: ch[keys.ChannelCreate],
		GuildID:   ev.GuildID,
	}

	e := discord.Embed{
		Title: "Channel created",
		Color: bcr.ColourGreen,

		Footer: &discord.EmbedFooter{
			Text: "ID: " + ev.ID.String(),
		},
		Timestamp: discord.NowTimestamp(),
	}

	switch ev.Type {
	case discord.GuildVoice:
		e.Title = "Voice channel created"
	case discord.GuildCategory:
		e.Title = "Category channel created"
	case discord.GuildText, discord.GuildNews:
		e.Title = "Text channel created"
	}

	if !ev.ParentID.IsValid() {
		e.Description = fmt.Sprintf("**Name:** %v\n**Category:** None", ev.Name)
	} else {
		cat, err := bot.State(ev.GuildID).Channel(ev.ParentID)
		if err == nil {
			e.Description = fmt.Sprintf("**Name:** %v\n**Category:** %v", ev.Name, cat.Name)
		}
	}

	for _, p := range ev.Overwrites {
		f := discord.EmbedField{
			Name:  "Override for " + p.ID.String(),
			Value: "",
		}

		if p.Type == discord.OverwriteRole {
			r, err := bot.State(ev.GuildID).Role(ev.GuildID, discord.RoleID(p.ID))
			if err == nil {
				f.Name = "Role override for " + r.Name
			}
		} else if p.Type == discord.OverwriteMember {
			u, err := bot.User(discord.UserID(p.ID))
			if err == nil {
				f.Name = "Member override for " + u.Username + "#" + u.Discriminator
			}
		}

		if p.Allow != 0 {
			f.Value += fmt.Sprintf("✅ %v", strings.Join(bcr.PermStrings(p.Allow), ", "))
		}

		if p.Deny != 0 {
			f.Value += fmt.Sprintf("\n\n❌ %v", strings.Join(bcr.PermStrings(p.Deny), ", "))
		}

		e.Fields = append(e.Fields, f)
	}

	if len(e.Fields) > 24 {
		e.Fields = e.Fields[:24]
	}

	resp.Embeds = append(resp.Embeds, e)
	return
}
