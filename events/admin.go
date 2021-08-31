package events

import (
	"fmt"
	"sort"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/dustin/go-humanize"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/db"
)

type guildInfo struct {
	ID      discord.GuildID
	OwnerID discord.UserID
	Name    string
	Users   int64
	Humans  int64
	Bots    int64
}

func (bot *Bot) adminInspectUsers(ctx *bcr.Context) (err error) {
	info := []guildInfo{}

	bot.GuildsMu.Lock()
	bot.MembersMu.Lock()
	for _, g := range bot.Guilds {
		ginfo := guildInfo{
			ID:      g.ID,
			OwnerID: g.OwnerID,
			Name:    g.Name,
		}

		for k, m := range bot.Members {
			if k.GuildID == g.ID {
				ginfo.Users++
				if m.User.Bot {
					ginfo.Bots++
				} else {
					ginfo.Humans++
				}
			}
		}

		info = append(info, ginfo)
	}
	bot.GuildsMu.Unlock()
	bot.MembersMu.Unlock()

	sort.Slice(info, func(i, j int) bool {
		return info[i].ID < info[j].ID
	})

	fields := []discord.EmbedField{}
	for _, g := range info {
		m, err := bot.DB.Channels(g.ID)
		if err != nil {
			return bot.DB.ReportCtx(ctx, err)
		}

		var events int
		for _, ch := range m {
			if ch.IsValid() {
				events++
			}
		}

		fields = append(fields, discord.EmbedField{
			Name:  fmt.Sprintf("%v (%v)", g.Name, g.ID),
			Value: fmt.Sprintf("**Owner:** %v (%v)\n**Users:** %v\n**Humans:** %v\n**Bots:** %v\n**%% bot:** %.2f\n**Enabled events:** %v/%v", g.OwnerID.Mention(), g.OwnerID, humanize.Comma(g.Users), humanize.Comma(g.Humans), humanize.Comma(g.Bots), (float64(g.Bots)/float64(g.Users))*100, events, len(db.DefaultEventMap)),
		})
	}

	_, _, err = ctx.ButtonPages(
		bcr.FieldPaginator(
			fmt.Sprintf("Guilds (%v)", len(info)),
			"", bcr.ColourPurple,
			fields, 5,
		), 15*time.Minute)
	return
}
