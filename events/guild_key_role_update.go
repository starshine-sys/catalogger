package events

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/events/handler"
)

// GuildKeyRoleUpdateEvent ...
type GuildKeyRoleUpdateEvent struct {
	*gateway.GuildMemberUpdateEvent

	ChannelID    discord.ChannelID
	AddedRoles   []discord.RoleID
	RemovedRoles []discord.RoleID
}

func (bot *Bot) keyroleUpdate(ev *GuildKeyRoleUpdateEvent) (resp *handler.Response, err error) {
	if !ev.ChannelID.IsValid() {
		return nil, nil
	}

	resp = &handler.Response{
		ChannelID: ev.ChannelID,
	}

	var keyRoles []uint64
	err = bot.DB.QueryRow(context.Background(), "select key_roles from guilds where id = $1", ev.GuildID).Scan(&keyRoles)
	if err != nil {
		return nil, err
	}

	if len(keyRoles) == 0 {
		return nil, nil
	}

	var addedKeyRoles, removedKeyRoles []discord.RoleID
	for _, r := range ev.AddedRoles {
		for _, k := range keyRoles {
			if r == discord.RoleID(k) {
				addedKeyRoles = append(addedKeyRoles, r)
			}
		}
	}

	for _, r := range ev.RemovedRoles {
		for _, k := range keyRoles {
			if r == discord.RoleID(k) {
				removedKeyRoles = append(removedKeyRoles, r)
			}
		}
	}

	if len(addedKeyRoles) == 0 && len(removedKeyRoles) == 0 {
		return nil, nil
	}

	// register event in metrics
	go bot.DB.Stats.RegisterEvent("GuildKeyRoleUpdateEvent")

	e := discord.Embed{
		Title: "Key roles added or removed",
		Color: bcr.ColourOrange,

		Author: &discord.EmbedAuthor{
			Name: ev.User.Username + "#" + ev.User.Discriminator,
			Icon: ev.User.AvatarURL(),
		},

		Footer: &discord.EmbedFooter{
			Text: "User ID: " + ev.User.ID.String(),
		},

		Timestamp: discord.NowTimestamp(),
	}

	if len(addedKeyRoles) > 0 {
		buf := ""

		for _, r := range addedKeyRoles {
			buf += r.Mention() + ", "
		}

		name := "Added role"
		if len(addedKeyRoles) != 1 {
			name += "s"
		}

		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  name,
			Value: strings.TrimSuffix(strings.TrimSpace(buf), ","),
		})
	}

	if len(removedKeyRoles) > 0 {
		buf := ""

		for _, r := range removedKeyRoles {
			buf += r.Mention() + ", "
		}

		name := "Removed role"
		if len(removedKeyRoles) != 1 {
			name += "s"
		}

		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  name,
			Value: strings.TrimSuffix(strings.TrimSpace(buf), ","),
		})
	}

	// sleep for a bit for the audit log
	time.Sleep(time.Second)

	logs, err := bot.State(ev.GuildID).AuditLog(ev.GuildID, api.AuditLogData{
		ActionType: discord.MemberRoleUpdate,
		Limit:      100,
	})
	if err == nil {
		for _, l := range logs.Entries {
			if discord.UserID(l.TargetID) == ev.User.ID && l.ID.Time().After(time.Now().Add(-10*time.Second)) {
				mod, err := bot.User(l.UserID)
				if err == nil {
					e.Fields = append(e.Fields, discord.EmbedField{
						Name:  "Responsible moderator",
						Value: fmt.Sprintf("%v#%v (%v, %v)", mod.Username, mod.Discriminator, mod.Mention(), mod.ID),
					})
				} else {
					e.Fields = append(e.Fields, discord.EmbedField{
						Name:  "Responsible moderator",
						Value: fmt.Sprintf("%v (%v)", l.UserID.Mention(), l.UserID),
					})
				}
				break
			}
		}
	} else {
		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  "Responsible moderator",
			Value: "*Unknown*",
		})
	}

	resp.Embeds = append(resp.Embeds, e)
	return resp, err
}
