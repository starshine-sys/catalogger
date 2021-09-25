package events

import (
	"fmt"
	"strings"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/db"
)

func (bot *Bot) guildMemberRemove(ev *gateway.GuildMemberRemoveEvent) {
	ch, err := bot.DB.Channels(ev.GuildID)
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   keys.GuildMemberRemove,
			GuildID: ev.GuildID,
		}, err)
		return
	}

	if !ch[keys.GuildMemberRemove].IsValid() {
		return
	}

	wh, err := bot.webhookCache("leave", ev.GuildID, ch[keys.GuildMemberRemove])
	if err != nil {
		bot.DB.Report(db.ErrorContext{
			Event:   keys.GuildMemberRemove,
			GuildID: ev.GuildID,
		}, err)
		return
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

	bot.MembersMu.Lock()
	m, ok := bot.Members[memberCacheKey{
		GuildID: ev.GuildID,
		UserID:  ev.User.ID,
	}]
	if ok {
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
	bot.MembersMu.Unlock()

	bot.Send(wh, keys.GuildMemberRemove, e)
}
