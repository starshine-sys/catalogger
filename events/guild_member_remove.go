package events

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/common"
	"github.com/starshine-sys/catalogger/events/duration"
	"github.com/starshine-sys/catalogger/events/handler"
)

func (bot *Bot) guildMemberRemove(ev *gateway.GuildMemberRemoveEvent) (resp *handler.Response, err error) {
	ch, err := bot.DB.Channels(ev.GuildID)
	if err != nil {
		return
	}

	if !ch[keys.GuildMemberRemove].IsValid() {
		return
	}

	resp = &handler.Response{
		ChannelID: ch[keys.GuildMemberRemove],
		GuildID:   ev.GuildID,
	}

	e := discord.Embed{
		Title: "Member left",
		Author: &discord.EmbedAuthor{
			Icon: ev.User.AvatarURL(),
			Name: ev.User.Tag(),
		},

		Color:       bcr.ColourGold,
		Description: ev.User.Mention(),

		Footer: &discord.EmbedFooter{
			Text: fmt.Sprintf("ID: %v", ev.User.ID),
		},
		Timestamp: discord.NowTimestamp(),
	}

	ctx, cancel := getctx()
	defer cancel()

	m, err := bot.MemberStore.Member(ctx, ev.GuildID, ev.User.ID)
	if err == nil {
		e.Description = fmt.Sprintf("%v joined <t:%v>\n(%v)", e.Description, m.Joined.Time().Unix(), duration.FormatTime(m.Joined.Time()))

		if len(m.RoleIDs) > 0 {
			mentions := make([]string, 0, len(m.RoleIDs))

			rls, err := bot.State(ev.GuildID).Roles(ev.GuildID)
			if err == nil {
				userRoles := make([]discord.Role, 0, len(m.RoleIDs))

				for _, r := range rls {
					for _, id := range m.RoleIDs {
						if id == r.ID {
							userRoles = append(userRoles, r)
							break
						}
					}
				}

				sort.Sort(bcr.Roles(userRoles))
				for _, r := range userRoles {
					mentions = append(mentions, r.Mention())
				}

			} else {
				for _, r := range m.RoleIDs {
					mentions = append(mentions, r.Mention())
				}
			}

			var b strings.Builder
			for i, r := range mentions {
				if b.Len() > 900 {
					b.WriteString(fmt.Sprintf("\n(too many roles to list, showing %v/%v)", i, len(mentions)))
					break
				}
				b.WriteString(r)
				if i != len(mentions)-1 {
					b.WriteString(", ")
				}
			}

			if b.Len() != 0 {
				e.Fields = append(e.Fields, discord.EmbedField{
					Name:  "Roles",
					Value: b.String(),
				})
			}
		}
	}

	go func() {
		// wait a second to give the audit log time to catch up
		time.Sleep(time.Second)

		log, err := bot.State(ev.GuildID).AuditLog(ev.GuildID, api.AuditLogData{
			ActionType: discord.MemberKick,
		})
		if err == nil {
			for _, e := range log.Entries {
				if time.Since(e.ID.Time()) < 1*time.Minute &&
					e.TargetID == discord.Snowflake(m.User.ID) {

					mod, err := bot.User(e.UserID)
					if err != nil {
						common.Log.Infof("Error fetching user: %v", err)
					}

					bot.EventHandler.Call(&MemberKickEvent{
						ChannelID: ch[keys.GuildMemberKick],
						User:      ev.User,
						Entry:     e,
						Moderator: mod,
					})
				}
			}
		} else {
			common.Log.Errorf("Error fetching audit logs for %v: %v", ev.GuildID, err)
		}
	}()

	resp.Embeds = append(resp.Embeds, e)
	return resp, err
}
