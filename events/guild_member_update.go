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

	if m.Nick != ev.Nick || m.User.Username+"#"+m.User.Discriminator != ev.User.Username+"#"+ev.User.Discriminator || m.User.Avatar != ev.User.Avatar {
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
		bot.DB.Report(db.ErrorContext{
			Event:   "guild_member_update",
			GuildID: ev.GuildID,
		}, err)
		return
	}

	if !ch["GUILD_MEMBER_UPDATE"].IsValid() {
		return
	}

	wh, err := bot.webhookCache("member-update", ev.GuildID, ch["GUILD_MEMBER_UPDATE"])
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   "guild_member_update",
			GuildID: ev.GuildID,
		}, err)
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

	err = webhook.FromAPI(wh.ID, wh.Token, bot.State(ev.GuildID).Client).Execute(webhook.ExecuteData{
		AvatarURL: bot.Router.Bot.AvatarURL(),
		Embeds:    []discord.Embed{e},
	})
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   "guild_member_update",
			GuildID: ev.GuildID,
		}, err)
		return
	}
}

func (bot *Bot) guildMemberNickUpdate(ev *gateway.GuildMemberUpdateEvent, m discord.Member) {
	ch, err := bot.DB.Channels(ev.GuildID)
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   "guild_member_nick_update",
			GuildID: ev.GuildID,
		}, err)
		return
	}

	if !ch["GUILD_MEMBER_NICK_UPDATE"].IsValid() {
		return
	}

	wh, err := bot.webhookCache("member-nick-update", ev.GuildID, ch["GUILD_MEMBER_NICK_UPDATE"])
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   "guild_member_nick_update",
			GuildID: ev.GuildID,
		}, err)
		return
	}

	e := discord.Embed{
		Title: "Changed nickname",
		Author: &discord.EmbedAuthor{
			Icon: ev.User.AvatarURL(),
			Name: ev.User.Username + "#" + ev.User.Discriminator,
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

	if m.User.Avatar != ev.User.Avatar {
		if oldNick == newNick {
			e.Title = ""
			e.Description = ""
		}

		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  "Avatar updated",
			Value: fmt.Sprintf("[Before](%v) (link will only work as long as the avatar is cached)\n[After](%v)", m.User.AvatarURL()+"?size=1024", ev.User.AvatarURL()+"?size=1024"),
		})
	}

	err = webhook.FromAPI(wh.ID, wh.Token, bot.State(ev.GuildID).Client).Execute(webhook.ExecuteData{
		AvatarURL: bot.Router.Bot.AvatarURL(),
		Embeds:    []discord.Embed{e},
	})
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   "guild_member_nick_update",
			GuildID: ev.GuildID,
		}, err)
		return
	}
}

func roleIn(s []discord.RoleID, id discord.RoleID) (exists bool) {
	for _, r := range s {
		if id == r {
			return true
		}
	}
	return false
}
