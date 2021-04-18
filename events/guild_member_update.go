package events

import (
	"fmt"
	"strings"

	"github.com/diamondburned/arikawa/v2/api/webhook"
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/starshine-sys/bcr"
)

func (bot *Bot) guildMemberUpdate(ev *gateway.GuildMemberUpdateEvent) {
	bot.MembersMu.Lock()
	m, ok := bot.Members[memberCacheKey{
		GuildID: ev.GuildID,
		UserID:  ev.User.ID,
	}]
	if !ok {
		// something went wrong
		bot.MembersMu.Unlock()
		return
	}

	// update cache
	up := bot.Members[memberCacheKey{
		GuildID: ev.GuildID,
		UserID:  ev.User.ID,
	}]
	ev.Update(&up)
	bot.Members[memberCacheKey{
		GuildID: ev.GuildID,
		UserID:  ev.User.ID,
	}] = up
	bot.MembersMu.Unlock()

	if m.Nick != ev.Nick || m.User.Username+"#"+m.User.Discriminator != ev.User.Username+"#"+ev.User.Discriminator {
		// username or nickname changed, so run that handler
		bot.guildMemberNickUpdate(ev, m)
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
		bot.Sugar.Errorf("Error getting server channels: %v", err)
		return
	}
	if !ch["GUILD_MEMBER_UPDATE"].IsValid() {
		return
	}

	wh, err := bot.webhookCache("member-update", ev.GuildID, ch["GUILD_MEMBER_UPDATE"])
	if err != nil {
		bot.Sugar.Errorf("Error getting webhook: %v", err)
		return
	}

	e := discord.Embed{
		Author: &discord.EmbedAuthor{
			Icon: m.User.AvatarURL(),
			Name: ev.User.Username + "#" + ev.User.Discriminator,
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

	webhook.New(wh.ID, wh.Token).Execute(webhook.ExecuteData{
		AvatarURL: bot.Router.Bot.AvatarURL(),
		Embeds:    []discord.Embed{e},
	})
}

func (bot *Bot) guildMemberNickUpdate(ev *gateway.GuildMemberUpdateEvent, m discord.Member) {
	ch, err := bot.DB.Channels(ev.GuildID)
	if err != nil {
		bot.Sugar.Errorf("Error getting server channels: %v", err)
		return
	}
	if !ch["GUILD_MEMBER_NICK_UPDATE"].IsValid() {
		return
	}

	wh, err := bot.webhookCache("member-nick-update", ev.GuildID, ch["GUILD_MEMBER_NICK_UPDATE"])
	if err != nil {
		bot.Sugar.Errorf("Error getting webhook: %v", err)
		return
	}

	e := discord.Embed{
		Title: "Changed nickname",
		Author: &discord.EmbedAuthor{
			Icon: m.User.AvatarURL(),
			Name: ev.User.Username + "#" + ev.User.Discriminator,
		},
		Thumbnail: &discord.EmbedThumbnail{
			URL: m.User.AvatarURL(),
		},
		Color: bcr.ColourGreen,
		Footer: &discord.EmbedFooter{
			Text: fmt.Sprintf("User ID: %v", m.User.ID),
		},
		Timestamp: discord.NowTimestamp(),
	}

	if m.User.Username+"#"+m.User.Discriminator != ev.User.Username+"#"+ev.User.Discriminator {
		e.Title = "Changed username"
	}

	oldNick := m.Nick
	newNick := ev.Nick
	if oldNick == "" {
		oldNick = "<none>"
	}
	if newNick == "" {
		newNick = "<none>"
	}

	if oldNick == newNick {
		oldNick = m.User.Username + "#" + m.User.Discriminator
		newNick = ev.User.Username + "#" + ev.User.Discriminator
	}

	e.Description = fmt.Sprintf("**Before:** %v\n**After:** %v", oldNick, newNick)

	webhook.New(wh.ID, wh.Token).Execute(webhook.ExecuteData{
		AvatarURL: bot.Router.Bot.AvatarURL(),
		Embeds:    []discord.Embed{e},
	})
}

func roleIn(s []discord.RoleID, id discord.RoleID) (exists bool) {
	for _, r := range s {
		if id == r {
			return true
		}
	}
	return false
}
