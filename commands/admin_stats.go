package commands

import (
	"context"
	"fmt"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/dustin/go-humanize"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/starshine-sys/bcr"
)

func (bot *Bot) adminStats(ctx *bcr.Context) (err error) {
	var stats []struct {
		ServerID discord.GuildID
		Total    int64
		Normal   int64
		Proxied  int64
	}

	err = pgxscan.Select(context.Background(), bot.DB.Pool, &stats, "select * from server_messages")
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	if len(stats) == 0 {
		_, err = ctx.Send("No servers.")
		return
	}

	var fields []discord.EmbedField
	for i, s := range stats {
		fields = append(fields, discord.EmbedField{
			Name:  fmt.Sprintf("%v. %v", i+1, s.ServerID),
			Value: fmt.Sprintf("%v total (%v normal/%v proxied)", humanize.Comma(s.Total), humanize.Comma(s.Normal), humanize.Comma(s.Proxied)),
		})
	}

	_, err = ctx.PagedEmbed(bcr.FieldPaginator("Server stats", "", bcr.ColourPurple, fields, 5), false)
	return
}
