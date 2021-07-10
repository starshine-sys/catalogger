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

func (bot *Bot) channelUpdate(ev *gateway.ChannelUpdateEvent) {
	bot.ChannelsMu.Lock()
	old, ok := bot.Channels[ev.ID]
	if !ok {
		bot.Sugar.Errorf("Couldn't find channel %v in the cache", ev.ID)
		bot.Channels[ev.ID] = ev.Channel
		bot.ChannelsMu.Unlock()
		return
	}
	bot.ChannelsMu.Unlock()

	ch, err := bot.DB.Channels(ev.GuildID)
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   "channel_update",
			GuildID: ev.GuildID,
		}, err)
	}
	if !ch["CHANNEL_UPDATE"].IsValid() {
		return
	}

	wh, err := bot.webhookCache("channel_update", ev.GuildID, ch["CHANNEL_UPDATE"])
	if err != nil {
		bot.Sugar.Errorf("Error getting webhook: %v", err)
		return
	}

	e := discord.Embed{
		Title:       "Channel updated",
		Color:       bcr.ColourBlue,
		Description: fmt.Sprintf("%v - #%v", ev.Mention(), ev.Name),

		Footer: &discord.EmbedFooter{
			Text: "ID: " + ev.ID.String(),
		},
		Timestamp: discord.NowTimestamp(),
	}

	switch ev.Type {
	case discord.GuildVoice:
		e.Title = "Voice channel updated"
	case discord.GuildCategory:
		e.Title = "Category channel updated"
	case discord.GuildText, discord.GuildNews:
		e.Title = "Text channel updated"
	}

	var changed bool

	if ev.CategoryID != old.CategoryID {
		f := discord.EmbedField{
			Name:  "Category updated",
			Value: "",
		}

		oldCat, err := bot.State(ev.GuildID).Channel(old.CategoryID)
		if err == nil {
			f.Value += fmt.Sprintf("**Before:** %v", oldCat.Name)
		}

		newCat, err := bot.State(ev.GuildID).Channel(ev.CategoryID)
		if err == nil {
			f.Value += fmt.Sprintf("\n**After:** %v", newCat.Name)
		}

		e.Fields = append(e.Fields, f)
		changed = true
	}

	if ev.Name != old.Name {
		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  "Name",
			Value: fmt.Sprintf("**Before:** %v\n**After:** %v", old.Name, ev.Name),
		})
		changed = true
	}

	if ev.Topic != old.Topic {
		old := old.Topic
		new := ev.Topic
		if old == "" {
			old = "None"
		}
		if new == "" {
			new = "None"
		}
		s := fmt.Sprintf("**Before:** %v\n\n**After:** %v", old, new)
		if len(s) > 1000 {
			s = s[:1000] + "..."
		}

		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  "Description",
			Value: s,
		})
		changed = true
	}

	var addedRoles, removedRoles, editedRoles []discord.Overwrite
	for _, oldRole := range old.Permissions {
		if !overwriteIn(ev.Permissions, oldRole) {
			removedRoles = append(removedRoles, oldRole)
			changed = true
		}
	}
	for _, newRole := range ev.Permissions {
		if !overwriteIn(old.Permissions, newRole) {
			addedRoles = append(addedRoles, newRole)
			changed = true
		}
	}
	for _, new := range ev.Permissions {
		for _, o := range old.Permissions {
			if new.ID == o.ID {
				if new.Allow != o.Allow || new.Deny != o.Deny {
					editedRoles = append(editedRoles, new)
					changed = true
				}
			}
		}
	}

	if len(removedRoles) != 0 {
		f := discord.EmbedField{
			Name: "Removed overrides",
		}

		for _, r := range removedRoles {
			if r.Type == discord.OverwriteRole {
				r, err := bot.State(ev.GuildID).Role(ev.GuildID, discord.RoleID(r.ID))
				if err == nil {
					f.Value += r.Name + ", "
				}
			} else if r.Type == discord.OverwriteMember {
				u, err := bot.State(ev.GuildID).User(discord.UserID(r.ID))
				if err == nil {
					f.Value += u.Username + "#" + u.Discriminator + ", "
				}
			} else {
				f.Value += r.ID.String() + ", "
			}
		}
	}

	for _, p := range append(addedRoles, editedRoles...) {
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

	if !changed {
		return
	}

	err = webhook.New(wh.ID, wh.Token).Execute(webhook.ExecuteData{
		AvatarURL: bot.Router.Bot.AvatarURL(),
		Embeds:    []discord.Embed{e},
	})
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   "channel_update",
			GuildID: ev.GuildID,
		}, err)
	}
}

func overwriteIn(s []discord.Overwrite, p discord.Overwrite) (exists bool) {
	for _, o := range s {
		if p.ID == o.ID {
			return true
		}
	}
	return false
}
