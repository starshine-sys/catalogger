package events

import (
	"fmt"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/api/webhook"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/db"
	"github.com/starshine-sys/pkgo"
)

func (bot *Bot) guildBanRemove(ev *gateway.GuildBanRemoveEvent) {
	ch, err := bot.DB.Channels(ev.GuildID)
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   "guild_ban_remove",
			GuildID: ev.GuildID,
		}, err)
	}

	if !ch["GUILD_BAN_REMOVE"].IsValid() {
		return
	}

	wh, err := bot.webhookCache("ban-remove", ev.GuildID, ch["GUILD_BAN_REMOVE"])
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   "guild_ban_remove",
			GuildID: ev.GuildID,
		}, err)
	}

	e := discord.Embed{
		Author: &discord.EmbedAuthor{
			Icon: ev.User.AvatarURL(),
			Name: ev.User.Username + "#" + ev.User.Discriminator,
		},

		Description: fmt.Sprintf("**%v#%v** (%v, %v) was unbanned", ev.User.Username, ev.User.Discriminator, ev.User.ID, ev.User.Mention()),

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
		Color:     bcr.ColourOrange,
	}

	// get ban reason/moderator
	// we need to sleep for this because discord can be slow
	time.Sleep(time.Second)
	logs, err := bot.State(ev.GuildID).AuditLog(ev.GuildID, api.AuditLogData{
		ActionType: discord.MemberBanRemove,
		Limit:      100,
	})
	if err == nil {
		for _, l := range logs.Entries {
			if discord.UserID(l.TargetID) == ev.User.ID && l.ID.Time().After(time.Now().Add(-10*time.Second)) {
				if l.Reason != "" {
					e.Fields[0].Value = l.Reason
				}

				mod, err := bot.State(ev.GuildID).User(l.UserID)
				if err != nil {
					e.Fields[1].Value = l.UserID.String()
					break
				}

				e.Fields[1].Value = fmt.Sprintf("%v#%v (%v)", mod.Username, mod.Discriminator, mod.ID)

				break
			}
		}
	}

	sys, err := pk.Account(pkgo.Snowflake(ev.User.ID))
	if err == nil {
		if sys.Name != "" {
			e.Fields = append(e.Fields, discord.EmbedField{
				Name:  "PluralKit system name",
				Value: sys.Name,
			})
		}

		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  "PluralKit system ID",
			Value: sys.ID,
		})

		banned, err := bot.DB.IsSystemBanned(ev.GuildID, sys.ID)
		if err != nil {
			bot.Sugar.Errorf("Error getting banned systems for %v: %v", ev.GuildID, err)
			bot.Sugar.Infof("Trying to unban %v anyway.", sys.ID)

			err = bot.DB.UnbanSystem(ev.GuildID, sys.ID)
			if err == nil {
				e.Fields = append(e.Fields, discord.EmbedField{
					Name:  "Linked system unbanned",
					Value: "The system linked to this account has been unbanned.",
				})
			}
		}

		if banned {
			err = bot.DB.UnbanSystem(ev.GuildID, sys.ID)
			if err == nil {
				e.Fields = append(e.Fields, discord.EmbedField{
					Name:  "Linked system unbanned",
					Value: "The system linked to this account has been unbanned.",
				})
			} else {
				e.Fields = append(e.Fields, discord.EmbedField{
					Name:  "Linked system not unbanned",
					Value: "There was an error trying to unban the linked system.\nYou will still be warned when an account linked to this system joins.",
				})
			}
		}
	}

	err = webhook.New(wh.ID, wh.Token).Execute(webhook.ExecuteData{
		AvatarURL: bot.Router.Bot.AvatarURL(),
		Embeds:    []discord.Embed{e},
	})
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   "guild_ban_remove",
			GuildID: ev.GuildID,
		}, err)
	}
}
