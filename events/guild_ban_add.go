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
	"github.com/starshine-sys/pkgo"
)

func (bot *Bot) guildBanAdd(ev *gateway.GuildBanAddEvent) (resp *handler.Response, err error) {
	ch, err := bot.DB.Channels(ev.GuildID)
	if err != nil {
		return
	}

	if !ch[keys.GuildBanAdd].IsValid() {
		return
	}

	resp = &handler.Response{
		ChannelID: ch[keys.GuildBanAdd],
	}

	resp.Embeds = []discord.Embed{{
		Author: &discord.EmbedAuthor{
			Icon: ev.User.AvatarURL(),
			Name: ev.User.Username + "#" + ev.User.Discriminator,
		},

		Description: fmt.Sprintf("**%v#%v** (%v, %v) was banned", ev.User.Username, ev.User.Discriminator, ev.User.ID, ev.User.Mention()),

		Fields: []discord.EmbedField{
			{
				Name:  "Reason",
				Value: "None specified",
			},
			{
				Name:  "Responsible moderator",
				Value: "Unknown",
			},
		},

		Footer: &discord.EmbedFooter{
			Text: "User ID: " + ev.User.ID.String(),
		},
		Timestamp: discord.NowTimestamp(),
		Color:     bcr.ColourRed,
	}}

	// get ban reason/moderator
	// we need to sleep for this because discord can be slow
	time.Sleep(time.Second)
	logs, err := bot.State(ev.GuildID).AuditLog(ev.GuildID, api.AuditLogData{
		ActionType: discord.MemberBanAdd,
		Limit:      100,
	})
	if err == nil {
		for _, l := range logs.Entries {
			if discord.UserID(l.TargetID) == ev.User.ID && l.ID.Time().After(time.Now().Add(-10*time.Second)) {
				if l.Reason != "" {
					resp.Embeds[0].Fields[0].Value = l.Reason
				}

				mod, err := bot.User(l.UserID)
				if err != nil {
					resp.Embeds[0].Fields[1].Value = l.UserID.String()
					break
				}

				resp.Embeds[0].Fields[1].Value = fmt.Sprintf("%v#%v (%v)", mod.Username, mod.Discriminator, mod.ID)

				break
			}
		}
	}

	sys, err := pk.Account(pkgo.Snowflake(ev.User.ID))
	if err == nil {
		if sys.Name != "" {
			resp.Embeds[0].Fields = append(resp.Embeds[0].Fields, discord.EmbedField{
				Name:  "PluralKit system name",
				Value: sys.Name,
			})
		}

		resp.Embeds[0].Fields = append(resp.Embeds[0].Fields, discord.EmbedField{
			Name:  "PluralKit system ID",
			Value: sys.ID,
		})

		banned, err := bot.DB.IsSystemBanned(ev.GuildID, sys.ID)
		if err != nil {
			common.Log.Errorf("Error getting banned systems for %v: %v", ev.GuildID, err)
		}

		if banned {
			resp.Embeds[0].Fields = append(resp.Embeds[0].Fields, discord.EmbedField{
				Name:  "System banned",
				Value: "The system linked to this account has already been banned.",
			})
		} else {
			err = bot.DB.BanSystem(ev.GuildID, sys.ID)
			if err != nil {
				common.Log.Errorf("Erorr banning system: %v", err)
				resp.Embeds[0].Fields = append(resp.Embeds[0].Fields, discord.EmbedField{
					Name:  "System not banned",
					Value: "There was an error trying to ban the linked system.\nYou will **not** be warned when an account linked to this system joins.",
				})
			} else {
				resp.Embeds[0].Fields = append(resp.Embeds[0].Fields, discord.EmbedField{
					Name:  "System banned",
					Value: "The system linked to this account has been banned.\nYou will be warned when an account linked to this system joins.",
				})
			}
		}

	}

	return resp, nil
}
