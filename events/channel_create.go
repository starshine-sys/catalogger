package events

import (
	"fmt"
	"strings"

	"github.com/diamondburned/arikawa/v3/api/webhook"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/db"
)

func (bot *Bot) channelCreate(ev *gateway.ChannelCreateEvent) {
	bot.ChannelsMu.Lock()
	bot.Channels[ev.ID] = ev.Channel
	bot.ChannelsMu.Unlock()

	ch, err := bot.DB.Channels(ev.GuildID)
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   "channel_create",
			GuildID: ev.GuildID,
		}, err)
		return
	}
	if !ch["CHANNEL_CREATE"].IsValid() {
		return
	}

	wh, err := bot.webhookCache("channel_create", ev.GuildID, ch["CHANNEL_CREATE"])
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   "channel_create",
			GuildID: ev.GuildID,
		}, err)
		return
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

	if !ev.CategoryID.IsValid() {
		e.Description = fmt.Sprintf("**Name:** %v\n**Category:** None", ev.Name)
	} else {
		cat, err := bot.State(ev.GuildID).Channel(ev.CategoryID)
		if err == nil {
			e.Description = fmt.Sprintf("**Name:** %v\n**Category:** %v", ev.Name, cat.Name)
		}
	}

	for _, p := range ev.Permissions {
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
			u, err := bot.State(ev.GuildID).User(discord.UserID(p.ID))
			if err == nil {
				f.Name = "Role override for " + u.Username + "#" + u.Discriminator
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

	err = webhook.New(wh.ID, wh.Token).Execute(webhook.ExecuteData{
		AvatarURL: bot.Router.Bot.AvatarURL(),
		Embeds:    []discord.Embed{e},
	})
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   "channel_create",
			GuildID: ev.GuildID,
		}, err)
		return
	}
}
