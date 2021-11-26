package events

import (
	"fmt"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/bcr"
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
			Name: ev.User.Username + "#" + ev.User.Discriminator,
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
		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  "Joined",
			Value: fmt.Sprintf("<t:%v> (%v)", m.Joined.Time().Unix(), bcr.HumanizeTime(bcr.DurationPrecisionSeconds, m.Joined.Time())),
		})

		if len(m.RoleIDs) > 0 {
			var s []string
			for _, r := range m.RoleIDs {
				s = append(s, r.Mention())
			}

			v := strings.Join(s, ", ")
			if len(v) > 1000 {
				v = v[:1000] + "..."
			}

			e.Fields = append(e.Fields, discord.EmbedField{
				Name:  "Roles",
				Value: v,
			})
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

					mod, err := bot.State(ev.GuildID).User(e.UserID)
					if err != nil {
						bot.Sugar.Infof("Error fetching user: %v", err)
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
			bot.Sugar.Errorf("Error fetching audit logs for %v: %v", ev.GuildID, err)
		}
	}()

	resp.Embeds = append(resp.Embeds, e)
	return resp, err
}
