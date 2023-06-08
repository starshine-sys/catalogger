package events

import (
	"fmt"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/common"
	"github.com/starshine-sys/catalogger/events/handler"
)

func (bot *Bot) guildMemberUpdate(ev *gateway.GuildMemberUpdateEvent) (resp *handler.Response, err error) {
	m, err := bot.Member(ev.GuildID, ev.User.ID)
	if err != nil {
		common.Log.Errorf("Error getting member: %v", err)
		return
	}

	// update cache
	// copy member struct
	up := m
	up.RoleIDs = append([]discord.RoleID(nil), m.RoleIDs...)
	ev.UpdateMember(&up)

	ctx, cancel := getctx()
	defer cancel()

	if err := bot.MemberStore.SetMember(ctx, ev.GuildID, up); err != nil {
		common.Log.Errorf("Error updating member in cache: %v", err)
	}

	oldDisabledTime := m.CommunicationDisabledUntil.Time().Round(time.Minute)
	newDisabledTime := ev.CommunicationDisabledUntil.Time().Round(time.Minute)

	if !newDisabledTime.IsZero() && !newDisabledTime.Before(oldDisabledTime) && !newDisabledTime.Equal(oldDisabledTime) && !newDisabledTime.Before(time.Now().UTC().Add(-time.Hour)) {
		return bot.handleTimeout(ev)
	}

	if m.Nick != ev.Nick || m.User.Tag() != ev.User.Tag() || m.User.Avatar != ev.User.Avatar || m.User.DisplayName != ev.User.DisplayName {
		// username or nickname changed, so run that handler
		return bot.guildMemberNickUpdate(ev, m)
	}

	// check for added roles
	var addedRoles, removedRoles []discord.RoleID
	for _, oldRole := range m.RoleIDs {
		if !roleIn(ev.RoleIDs, oldRole) {
			removedRoles = append(removedRoles, oldRole)
		}
	}
	for _, newRole := range ev.RoleIDs {
		if !roleIn(m.RoleIDs, newRole) {
			addedRoles = append(addedRoles, newRole)
		}
	}

	if len(addedRoles) == 0 && len(removedRoles) == 0 {
		return
	}

	ch, err := bot.DB.Channels(ev.GuildID)
	if err != nil {
		return
	}

	bot.EventHandler.Call(&GuildKeyRoleUpdateEvent{
		GuildMemberUpdateEvent: ev,
		ChannelID:              ch[keys.GuildKeyRoleUpdate],
		AddedRoles:             addedRoles,
		RemovedRoles:           removedRoles,
	})

	if !ch[keys.GuildMemberUpdate].IsValid() {
		return
	}

	resp = &handler.Response{
		ChannelID: ch[keys.GuildMemberUpdate],
	}

	e := discord.Embed{
		Author: &discord.EmbedAuthor{
			Icon: m.User.AvatarURL(),
			Name: ev.User.Tag(),
		},
		Color:       bcr.ColourOrange,
		Title:       "Roles updated",
		Description: ev.User.Mention(),

		Footer: &discord.EmbedFooter{
			Text: fmt.Sprintf("User ID: %v", ev.User.ID),
		},
		Timestamp: discord.NowTimestamp(),
	}

	if len(addedRoles) > 0 {
		var s []string
		for _, r := range addedRoles {
			s = append(s, r.Mention())
		}
		v := strings.Join(s, ", ")
		if len(v) > 1000 {
			v = v[:1000] + "..."
		}

		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  "Added roles",
			Value: v,
		})
	}

	if len(removedRoles) > 0 {
		var s []string
		for _, r := range removedRoles {
			s = append(s, r.Mention())
		}
		v := strings.Join(s, ", ")
		if len(v) > 1000 {
			v = v[:1000] + "..."
		}

		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  "Removed roles",
			Value: v,
		})
	}

	resp.Embeds = append(resp.Embeds, e)
	return resp, err
}

func (bot *Bot) guildMemberNickUpdate(ev *gateway.GuildMemberUpdateEvent, m discord.Member) (resp *handler.Response, err error) {
	// Discord sends this as part of the normal guild member update event, so we register this event manually
	bot.DB.Stats.RegisterEvent("GuildMemberNickUpdateEvent")

	ch, err := bot.DB.Channels(ev.GuildID)
	if err != nil {
		return
	}

	if !ch[keys.GuildMemberNickUpdate].IsValid() {
		return
	}

	resp = &handler.Response{
		ChannelID: ch[keys.GuildMemberNickUpdate],
	}

	e := discord.Embed{
		Title: "Changed nickname",
		Author: &discord.EmbedAuthor{
			Icon: ev.User.AvatarURL(),
			Name: ev.User.Tag(),
		},
		Thumbnail: &discord.EmbedThumbnail{
			URL: ev.User.AvatarURL() + "?size=1024",
		},
		Color: bcr.ColourGreen,
		Footer: &discord.EmbedFooter{
			Text: fmt.Sprintf("User ID: %v", m.User.ID),
		},
		Timestamp: discord.NowTimestamp(),
	}

	var old, new string
	var nameChanged bool
	if m.User.Tag() != ev.User.Tag() {
		e.Title = "Changed username"
		old = m.User.Tag()
		new = ev.User.Tag()
		nameChanged = true
	} else if m.Nick != ev.Nick {
		e.Title = "Changed nickname"
		old = m.Nick
		new = ev.Nick
		nameChanged = true
	} else if m.User.DisplayName != ev.User.DisplayName {
		e.Title = "Changed display name"
		old = m.User.DisplayName
		new = ev.User.DisplayName
		nameChanged = true
	}

	if nameChanged {
		if old == "" {
			old = "<none>"
		}
		if new == "" {
			new = "<none>"
		}

		e.Description = fmt.Sprintf("**Before:** %v\n**After:** %v", old, new)
		resp.Embeds = append(resp.Embeds, e)
	}

	if m.User.Avatar != ev.User.Avatar {
		resp.Embeds = append(resp.Embeds, discord.Embed{
			Title: "Changed avatar",
			Author: &discord.EmbedAuthor{
				Icon: ev.User.AvatarURL(),
				Name: ev.User.Tag(),
			},
			Thumbnail: &discord.EmbedThumbnail{
				URL: ev.User.AvatarURL() + "?size=1024",
			},
			Description: fmt.Sprintf("**%v** changed their avatar.\n[Before](%v) (link will only work as long as the avatar is cached)\n[After](%v)", m.User.Tag(), m.User.AvatarURL()+"?size=1024", ev.User.AvatarURL()+"?size=1024"),
			Color:       bcr.ColourGreen,
			Footer: &discord.EmbedFooter{
				Text: fmt.Sprintf("User ID: %v", m.User.ID),
			},
			Timestamp: discord.NowTimestamp(),
		})
	}

	return resp, err
}

func roleIn(s []discord.RoleID, id discord.RoleID) (exists bool) {
	for _, r := range s {
		if id == r {
			return true
		}
	}
	return false
}
