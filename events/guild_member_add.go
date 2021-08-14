package events

import (
	"fmt"
	"strconv"
	"time"

	"emperror.dev/errors"
	"github.com/diamondburned/arikawa/v3/api/webhook"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/jackc/pgx/v4"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/db"
	"github.com/starshine-sys/pkgo"
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
		bot.DB.Report(db.ErrorContext{
			Event:   "guild_member_add",
			GuildID: m.GuildID,
		}, err)
		return
	}

	if !ch["GUILD_MEMBER_ADD"].IsValid() {
		return
	}

	wh, err := bot.webhookCache("join", m.GuildID, ch["GUILD_MEMBER_ADD"])
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   "guild_member_add",
			GuildID: m.GuildID,
		}, err)
		return
	}

	e := discord.Embed{
		Title: "Member joined",
		Thumbnail: &discord.EmbedThumbnail{
			URL: m.User.AvatarURL(),
		},

		Color:       bcr.ColourGreen,
		Description: fmt.Sprintf("%v#%v %v", m.User.Username, m.User.Discriminator, m.Mention()),

		Fields: []discord.EmbedField{
			{
				Name:   "Account created",
				Value:  bcr.HumanizeTime(bcr.DurationPrecisionMinutes, m.User.ID.Time()),
				Inline: true,
			},
		},

		Footer: &discord.EmbedFooter{
			Text: fmt.Sprintf("ID: %v", m.User.ID),
		},
		Timestamp: discord.NowTimestamp(),
	}

	g, err := bot.State(m.GuildID).GuildWithCount(m.GuildID)
	if err == nil {
		e.Fields = append(e.Fields, discord.EmbedField{
			Name:   "Current member count",
			Value:  strconv.FormatUint(g.ApproximateMembers, 10),
			Inline: true,
		})
	}

	sys, err := pk.Account(pkgo.Snowflake(m.User.ID))
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
		is, err := bot.State(m.GuildID).GuildInvites(m.GuildID)
		if err == nil {
			bot.InviteMu.Lock()
			var (
				found bool
				inv   discord.Invite
			)

			for _, existing := range bot.Invites[m.GuildID] {
				for _, i := range is {
					if existing.Code == i.Code && existing.Uses < i.Uses {
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
				name, err := bot.DB.GetInviteName(inv.Code)
				if err != nil {
					bot.Sugar.Errorf("Error getting invite name: %v", err)
				}

				e.Fields = append(e.Fields, []discord.EmbedField{
					{
						Name:  "​",
						Value: "**Invite information**",
					},
					{
						Name:   "Name",
						Value:  name,
						Inline: true,
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
						Value:  fmt.Sprintf("%v#%v %v", inv.Inviter.Username, inv.Inviter.Discriminator, inv.Inviter.Mention()),
						Inline: true,
					},
				}...)
			}

			bot.Invites[m.GuildID] = is

			bot.InviteMu.Unlock()
		}
	}

	embeds := []discord.Embed{e}

	if m.User.CreatedAt().After(time.Now().UTC().Add(-168 * time.Hour)) {
		embeds = append(embeds, discord.Embed{
			Title:       "New account",
			Description: fmt.Sprintf("⚠️ This account was created only **%v** (%v)", bcr.HumanizeTime(bcr.DurationPrecisionSeconds, m.User.CreatedAt()), m.User.CreatedAt().Format(time.RFC1123)),
			Color:       bcr.ColourOrange,
		})
	}

	if sys != nil && sys.ID != "" {
		if banned, _ := bot.DB.IsSystemBanned(m.GuildID, sys.ID); banned {
			e := discord.Embed{
				Title: "Banned system",

				Color: bcr.ColourRed,
				Footer: &discord.EmbedFooter{
					Text: "ID: " + sys.ID,
				},
				Timestamp: discord.NowTimestamp(),
			}

			if sys.Name != "" {
				e.Description = fmt.Sprintf("⚠️ The system associated with this account (**%v**) has been banned from the server.", sys.Name)
			} else {
				e.Description = "⚠️ The system associated with this account has been banned from the server."
			}

			embeds = append(embeds, e)
		}
	}

	// we create a client separately because we might need to send 2 messages
	client := webhook.New(wh.ID, wh.Token)

	err = client.Execute(webhook.ExecuteData{
		AvatarURL: bot.Router.Bot.AvatarURL(),
		Embeds:    embeds,
	})
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   "guild_member_add",
			GuildID: m.GuildID,
		}, err)
		return
	}

	wl, err := bot.DB.UserWatchlist(m.GuildID, m.User.ID)
	if err != nil || wl == nil {
		if errors.Cause(err) != pgx.ErrNoRows {
			bot.DB.Report(db.ErrorContext{
				Event:   "guild_member_add",
				GuildID: m.GuildID,
			}, err)
			return
		}
		return
	}

	e = discord.Embed{
		Title:       "⚠️ User on watchlist",
		Color:       bcr.ColourRed,
		Description: fmt.Sprintf("**%v#%v** is on this server's watchlist.", m.User.Username, m.User.Discriminator),
		Footer: &discord.EmbedFooter{
			Text: fmt.Sprintf("ID: %v | Added", m.User.ID),
		},
		Timestamp: discord.NewTimestamp(wl.Added),
	}

	field := discord.EmbedField{
		Name: "Reason",
	}

	for _, s := range wl.Reason {
		if len(field.Value) > 1000 {
			field.Value += "..."
			e.Fields = append(e.Fields, field)

			field = discord.EmbedField{
				Name:  "Reason (cont.)",
				Value: "...",
			}
		}
		field.Value += string(s)
	}

	if len(field.Value) > 0 {
		e.Fields = append(e.Fields, field)
	}

	mod, err := bot.State(m.GuildID).User(wl.Moderator)
	if err == nil {
		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  "Moderator",
			Value: fmt.Sprintf("%v#%v (%v)", mod.Username, mod.Discriminator, mod.Mention()),
		})
	} else {
		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  "Moderator",
			Value: wl.Moderator.Mention(),
		})
	}

	err = client.Execute(webhook.ExecuteData{
		AvatarURL: bot.Router.Bot.AvatarURL(),
		Embeds:    []discord.Embed{e},
	})
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   "guild_member_add",
			GuildID: m.GuildID,
		}, err)
		return
	}
}
