package channels

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/catalogger/v2/common"
	"github.com/starshine-sys/catalogger/v2/common/log"
)

func (bot *Bot) channelCreate(ev *gateway.ChannelCreateEvent) {
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		log.Debugf("setting channel %v in %v", ev.Channel.ID, ev.GuildID)
		err := bot.Cabinet.SetChannel(ctx, ev.GuildID, ev.Channel)
		if err != nil {
			log.Errorf("setting channel %v in %v: %v", ev.Channel.ID, ev.GuildID, err)
		}
	}()

	if !bot.ShouldLog() {
		return
	}

	e := discord.Embed{
		Title: "Channel created",
		Color: common.ColourGreen,
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
	case discord.GuildText, discord.GuildAnnouncement:
		e.Title = "Text channel created"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Category channels may be deleted eventually, so we use the category *name*, not a mention.
	if !ev.ParentID.IsValid() {
		e.Description = fmt.Sprintf("**Name:** %v\n**Category:** None", ev.Name)
	} else {
		cat, err := bot.Cabinet.Channel(ctx, ev.ParentID)
		if err == nil {
			e.Description = fmt.Sprintf("**Name:** %v\n**Category:** %v", ev.Name, cat.Name)
		}
	}

	// Iterate over each permission override and add it to the embed
	for _, p := range ev.Overwrites {
		f := discord.EmbedField{
			Name:  "Override for " + p.ID.String(),
			Value: "",
		}

		if p.Type == discord.OverwriteRole {
			r, err := bot.Cabinet.Role(ctx, ev.GuildID, discord.RoleID(p.ID))
			if err == nil {
				f.Name = "Role override for " + r.Name
			}
		} else if p.Type == discord.OverwriteMember {
			u, err := bot.GuildUser(ev.GuildID, discord.UserID(p.ID))
			if err == nil {
				f.Name = "Member override for " + u.Tag()
			}
		}

		if p.Allow != 0 {
			f.Value += fmt.Sprintf("✅ %v", strings.Join(common.PermStrings(p.Allow), ", "))
		}

		if p.Deny != 0 {
			f.Value += fmt.Sprintf("\n\n❌ %v", strings.Join(common.PermStrings(p.Deny), ", "))
		}

		e.Fields = append(e.Fields, f)
	}

	if len(e.Fields) > 24 {
		e.Fields = e.Fields[:24]
	}

	bot.Send(ev.GuildID, ev, SendData{
		Embeds: []discord.Embed{e},
	})
}
