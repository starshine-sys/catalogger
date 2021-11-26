package events

import (
	"fmt"
	"strconv"
	"time"

	"emperror.dev/errors"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/jackc/pgx/v4"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/events/handler"
	"github.com/starshine-sys/pkgo"
)

func (bot *Bot) guildMemberAdd(m *gateway.GuildMemberAddEvent) (resp *handler.Response, err error) {
	ctx, cancel := getctx()
	defer cancel()

	err = bot.MemberStore.SetMember(ctx, m.GuildID, m.Member)
	if err != nil {
		bot.Sugar.Errorf("Error setting member %v in cache: %v", m.User.ID, err)
	}

	ch, err := bot.DB.Channels(m.GuildID)
	if err != nil {
		return
	}

	if !ch[keys.GuildMemberAdd].IsValid() {
		return
	}

	resp = &handler.Response{
		ChannelID: ch[keys.GuildMemberAdd],
		GuildID:   m.GuildID,
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
				Value:  fmt.Sprintf("<t:%v> (%v)", m.User.ID.Time().Unix(), bcr.HumanizeTime(bcr.DurationPrecisionMinutes, m.User.ID.Time())),
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

		tag := "(None)"
		if sys.Tag != "" {
			tag = sys.Tag
		}

		e.Fields = append(e.Fields, discord.EmbedField{
			Name:   "Tag",
			Value:  tag,
			Inline: true,
		})

		e.Fields = append(e.Fields, discord.EmbedField{
			Name:   "Created",
			Value:  fmt.Sprintf("<t:%v>\n%v", sys.Created.Unix(), bcr.HumanizeTime(bcr.DurationPrecisionMinutes, sys.Created)),
			Inline: false,
		})
	}

	if !m.User.Bot {
		is, err := bot.State(m.GuildID).GuildInvites(m.GuildID)
		if err == nil {
			allExisting, err := bot.MemberStore.Invites(ctx, m.GuildID)
			if err == nil {
				inv, found := checkInvites(allExisting, is)

				if !found {

					if g.VanityURLCode != "" {
						e.Fields = append(e.Fields, discord.EmbedField{
							Name:  "Invite used",
							Value: "Vanity invite (" + bcr.EscapeBackticks(g.VanityURLCode) + ")",
						})
					} else {
						e.Fields = append(e.Fields, discord.EmbedField{
							Name:  "Invite used",
							Value: "Could not determine invite.",
						})
					}

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
							Value:  fmt.Sprintf("<t:%v>", inv.CreatedAt.Time().Unix()),
							Inline: true,
						},
					}...)

					if inv.Inviter != nil {
						e.Fields = append(e.Fields, discord.EmbedField{
							Name:   "Created by",
							Value:  fmt.Sprintf("%v#%v %v", inv.Inviter.Username, inv.Inviter.Discriminator, inv.Inviter.Mention()),
							Inline: true,
						})
					} else {
						e.Fields = append(e.Fields, discord.EmbedField{
							Name:   "Created by",
							Value:  "Unknown",
							Inline: true,
						})
					}
				}

			} else {
				bot.Sugar.Errorf("Error fetching previous invites for %v: %v", m.GuildID, err)

				e.Fields = append(e.Fields, discord.EmbedField{
					Name:  "Invite used",
					Value: "Could not determine invite.",
				})
			}

			err = bot.MemberStore.SetInvites(ctx, m.GuildID, is)
			if err != nil {
				bot.Sugar.Errorf("Error updating invites for %v: %v", m.GuildID, err)
			}
		}
	}

	resp.Embeds = append(resp.Embeds, e)

	if m.User.CreatedAt().After(time.Now().UTC().Add(-168 * time.Hour)) {
		resp.Embeds = append(resp.Embeds, discord.Embed{
			Title:       "New account",
			Description: fmt.Sprintf("⚠️ This account was only created **%v** (<t:%v>)", bcr.HumanizeTime(bcr.DurationPrecisionSeconds, m.User.CreatedAt()), m.User.CreatedAt().Unix()),
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

			resp.Embeds = append(resp.Embeds, e)
		}
	}

	wl, err := bot.DB.UserWatchlist(m.GuildID, m.User.ID)
	if err != nil || wl == nil {
		if errors.Cause(err) == pgx.ErrNoRows {
			err = nil
		}
		return resp, err
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

	resp.Embeds = append(resp.Embeds, e)

	return resp, nil
}

func checkInvites(old, new []discord.Invite) (inv discord.Invite, found bool) {
	// check invites in both slices
	for _, o := range old {
		for _, n := range new {
			if o.Code == n.Code && o.Uses < n.Uses {
				return n, true
			}
		}
	}

	// check only new invites with 1 use
	for _, n := range new {
		if !invExists(old, n) && n.Uses == 1 {
			return n, true
		}
	}

	// check only old invites with 1 use less than max
	for _, o := range old {
		if !invExists(new, o) && o.MaxUses != 0 && o.MaxUses == o.Uses+1 {
			// this is an *old* invite so we should update the count before returning
			o.Uses = o.Uses + 1
			return o, true
		}
	}

	return inv, false
}

func invExists(invs []discord.Invite, i discord.Invite) bool {
	for _, o := range invs {
		if i.Code == o.Code {
			return true
		}
	}

	return false
}
