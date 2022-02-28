package events

import (
	"fmt"
	"time"

	"emperror.dev/errors"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/dustin/go-humanize"
	"github.com/jackc/pgx/v4"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/common"
	"github.com/starshine-sys/catalogger/events/duration"
	"github.com/starshine-sys/catalogger/events/handler"
	"github.com/starshine-sys/pkgo/v2"
)

func (bot *Bot) guildMemberAdd(m *gateway.GuildMemberAddEvent) (resp *handler.Response, err error) {
	ctx, cancel := getctx()
	defer cancel()

	err = bot.MemberStore.SetMember(ctx, m.GuildID, m.Member)
	if err != nil {
		common.Log.Errorf("Error setting member %v in cache: %v", m.User.ID, err)
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
		Author: &discord.EmbedAuthor{
			Name: m.User.Tag(),
			Icon: m.User.AvatarURL(),
		},
		Title:       "Member joined",
		Color:       bcr.ColourGreen,
		Description: m.Mention(),
		Footer: &discord.EmbedFooter{
			Text: fmt.Sprintf("ID: %v", m.User.ID),
		},
		Timestamp: discord.NowTimestamp(),
	}

	g, err := bot.State(m.GuildID).GuildWithCount(m.GuildID)
	if err == nil {
		e.Description += fmt.Sprintf(" %v to join", humanize.Ordinal(int(g.ApproximateMembers)))
	}

	e.Description += fmt.Sprintf("\ncreated <t:%v>\n(%v)", m.User.ID.Time().Unix(), duration.FormatTime(m.User.ID.Time()))

	sys, err := pk.Account(pkgo.Snowflake(m.User.ID))
	if err == nil {
		var (
			name = sys.Name
			tag  = sys.Tag
		)

		if name == "" {
			name = "*(none)*"
		}
		if tag == "" {
			name = "*(none)*"
		}

		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  "PluralKit system",
			Value: fmt.Sprintf("**ID:** %v\n**Name:** %v\n**Tag:** %v\n**Created:** <t:%v>", sys.ID, name, tag, sys.Created.Unix()),
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
							Value: "Vanity invite (" + bcr.AsCode(g.VanityURLCode) + ")",
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
						common.Log.Errorf("Error getting invite name: %v", err)
					}

					s := fmt.Sprintf("**Code:** %v\n**Name:** %v\n**Uses:** %v\n**Created at:** <t:%v>", inv.Code, name, inv.Uses, inv.CreatedAt.Time().Unix())

					if inv.Inviter != nil {
						s = fmt.Sprintf("%v\n**Created by:** %v %v", s, inv.Inviter.Tag(), inv.Inviter.Mention())
					} else {
						s = fmt.Sprintf("%v\n**Created by:** unknown", s)
					}

					e.Fields = append(e.Fields, discord.EmbedField{
						Name:  "Invite used",
						Value: s,
					})
				}

			} else {
				common.Log.Errorf("Error fetching previous invites for %v: %v", m.GuildID, err)

				e.Fields = append(e.Fields, discord.EmbedField{
					Name:  "Invite used",
					Value: "Could not determine invite.",
				})
			}

			err = bot.MemberStore.SetInvites(ctx, m.GuildID, is)
			if err != nil {
				common.Log.Errorf("Error updating invites for %v: %v", m.GuildID, err)
			}
		}
	}

	resp.Embeds = append(resp.Embeds, e)

	if m.User.CreatedAt().After(time.Now().UTC().Add(-168 * time.Hour)) {
		resp.Embeds = append(resp.Embeds, discord.Embed{
			Title:       "New account",
			Description: fmt.Sprintf("⚠️ Created **%v** (<t:%v>)", duration.FormatTime(m.User.CreatedAt()), m.User.CreatedAt().Unix()),
			Color:       bcr.ColourOrange,
		})
	}

	if sys.ID != "" {
		if banned, _ := bot.DB.IsSystemBanned(m.GuildID, sys.ID, sys.UUID); banned {
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

	mod, err := bot.User(wl.Moderator)
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
