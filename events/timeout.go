package events

import (
	"fmt"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/common"
	"github.com/starshine-sys/catalogger/events/handler"
)

func (bot *Bot) handleTimeout(ev *gateway.GuildMemberUpdateEvent) (resp *handler.Response, err error) {
	ch, err := bot.DB.Channels(ev.GuildID)
	if err != nil {
		return
	}

	if !ch[keys.GuildMemberKick].IsValid() {
		return
	}

	resp = &handler.Response{
		GuildID:   ev.GuildID,
		ChannelID: ch[keys.GuildMemberKick],
	}

	e := discord.Embed{
		Author: &discord.EmbedAuthor{
			Name: ev.User.Tag(),
			Icon: ev.User.AvatarURLWithType(discord.PNGImage),
		},
		Description: fmt.Sprintf("**%v** (%v, %v) was timed out", ev.User.Tag(), ev.User.ID, ev.User.Mention()),

		Fields: []discord.EmbedField{{
			Name:   "Until",
			Value:  fmt.Sprintf("<t:%v>\n%v", ev.CommunicationDisabledUntil.Time().Unix(), bcr.HumanizeTime(bcr.DurationPrecisionSeconds, ev.CommunicationDisabledUntil.Time().Add(time.Second))),
			Inline: false,
		}},

		Footer: &discord.EmbedFooter{
			Text: "User ID: " + ev.User.ID.String(),
		},
		Timestamp: discord.NowTimestamp(),
		Color:     bcr.ColourGold,
	}

	// wait a second to give the audit log time to catch up
	time.Sleep(time.Second)

	entries, err := bot.State(ev.GuildID).AuditLog(ev.GuildID, api.AuditLogData{
		ActionType: discord.MemberUpdate,
	})
	if err == nil {
		for _, entry := range entries.Entries {
			if time.Since(entry.ID.Time()) < 1*time.Minute &&
				entry.TargetID == discord.Snowflake(ev.User.ID) {

				reason := "None specified"
				if entry.Reason != "" {
					reason = entry.Reason
				}
				e.Fields = append(e.Fields, discord.EmbedField{
					Name:  "Reason",
					Value: reason,
				})

				mod, err := bot.User(entry.UserID)
				if err != nil {
					common.Log.Infof("Error fetching user: %v", err)
					e.Fields = append(e.Fields, discord.EmbedField{
						Name:  "Responsible moderator",
						Value: fmt.Sprintf("%v (%v)", entry.UserID.Mention(), entry.UserID),
					})
				} else {
					e.Fields = append(e.Fields, discord.EmbedField{
						Name:  "Responsible moderator",
						Value: fmt.Sprintf("%v (%v)", mod.Tag(), mod.ID),
					})
				}

				resp.Embeds = append(resp.Embeds, e)
				return resp, nil
			}
		}
	}

	e.Fields = append(e.Fields, discord.EmbedField{
		Name:  "Responsible moderator",
		Value: "Unknown",
	})

	resp.Embeds = append(resp.Embeds, e)
	return resp, nil
}
