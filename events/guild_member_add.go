package events

import (
	"fmt"
	"strconv"
	"time"

	"github.com/diamondburned/arikawa/v2/api/webhook"
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/starshine-sys/bcr"
)

func (bot *Bot) guildMemberAdd(m *gateway.GuildMemberAddEvent) {
	bot.MembersMu.Lock()
	bot.Members[memberCacheKey{
		GuildID: m.GuildID,
		UserID:  m.User.ID,
	}] = m.Member
	bot.MembersMu.Unlock()

	ch, err := bot.DB.Channels(m.GuildID)
	if err != nil {
		bot.Sugar.Errorf("Error getting server channels: %v", err)
		return
	}
	if !ch["GUILD_MEMBER_ADD"].IsValid() {
		return
	}

	wh, err := bot.webhookCache("join", m.GuildID, ch["GUILD_MEMBER_ADD"])
	if err != nil {
		bot.Sugar.Errorf("Error getting webhook: %v", err)
		return
	}

	e := discord.Embed{
		Title: "Member joined",
		Thumbnail: &discord.EmbedThumbnail{
			URL: m.User.AvatarURL(),
		},

		Color:       bcr.ColourGreen,
		Description: fmt.Sprintf("%v\n%v#%v", m.Mention(), m.User.Username, m.User.Discriminator),

		Fields: []discord.EmbedField{
			{
				Name:   "Account age",
				Value:  bcr.HumanizeTime(bcr.DurationPrecisionMinutes, m.User.ID.Time()),
				Inline: true,
			},
		},

		Footer: &discord.EmbedFooter{
			Text: fmt.Sprintf("ID: %v", m.User.ID),
		},
		Timestamp: discord.NowTimestamp(),
	}

	g, err := bot.State.GuildWithCount(m.GuildID)
	if err == nil {
		e.Fields = append(e.Fields, discord.EmbedField{
			Name:   "Current member count",
			Value:  strconv.FormatUint(g.ApproximateMembers, 10),
			Inline: true,
		})
	}

	sys, err := pk.GetSystemByUserID(m.User.ID.String())
	if err == nil {
		e.Fields = append(e.Fields, discord.EmbedField{
			Name:   "​",
			Value:  "**PluralKit system information**",
			Inline: false,
		})

		if sys.Name != "" {
			e.Fields = append(e.Fields, discord.EmbedField{
				Name:   "Name",
				Value:  sys.Name,
				Inline: true,
			})
		}

		e.Fields = append(e.Fields, discord.EmbedField{
			Name:   "ID",
			Value:  sys.ID,
			Inline: true,
		})

		e.Fields = append(e.Fields, discord.EmbedField{
			Name:   "Created",
			Value:  bcr.HumanizeTime(bcr.DurationPrecisionMinutes, sys.Created),
			Inline: true,
		})
	}

	if !m.User.Bot {
		is, err := bot.State.GuildInvites(m.GuildID)
		if err == nil {
			bot.InviteMu.Lock()
			var (
				found bool
				inv   discord.Invite
			)

			for _, existing := range bot.Invites[m.GuildID] {
				for _, i := range is {
					if existing.Uses < i.Uses {
						found = true
						inv = i
						break
					}
				}
			}

			if !found {
				e.Fields = append(e.Fields, discord.EmbedField{
					Name:  "Invite used",
					Value: "Could not determine invite.",
				})
			} else {
				e.Fields = append(e.Fields, []discord.EmbedField{
					{
						Name:  "​",
						Value: "**Invite information**",
					},
					{
						Name:   "Code",
						Value:  inv.Code,
						Inline: true,
					},
					{
						Name:   "Uses",
						Value:  fmt.Sprint(inv.Uses),
						Inline: true,
					},
					{
						Name:   "Created at",
						Value:  inv.CreatedAt.Format(time.RFC1123),
						Inline: true,
					},
					{
						Name:   "Created by",
						Value:  fmt.Sprintf("%v - %v#%v", inv.Inviter.Mention(), inv.Inviter.Username, inv.Inviter.Discriminator),
						Inline: true,
					},
				}...)
			}
			bot.InviteMu.Unlock()
		}
	}

	webhook.New(wh.ID, wh.Token).Execute(webhook.ExecuteData{
		AvatarURL: bot.Router.Bot.AvatarURL(),
		Embeds:    []discord.Embed{e},
	})
}
