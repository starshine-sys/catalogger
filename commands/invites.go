package commands

import (
	"fmt"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/starshine-sys/bcr"
)

func (bot *Bot) listInvites(ctx bcr.Contexter) (err error) {
	is, err := ctx.Session().GuildInvites(ctx.GetGuild().ID)
	if err != nil {
		bot.Sugar.Errorf("Error getting guild invites: %v", err)
		_, err = ctx.Sendf("Could not get this server's invites. Are you sure I have the **Manage Server** permission?")
		return
	}

	if len(is) == 0 {
		_, err = ctx.Send("This server has no invites.")
		return
	}

	var (
		invites = map[string]discord.Invite{}
		names   = map[string]string{}
	)

	for _, i := range is {
		i := i
		invites[i.Code] = i
		names[i.Code] = "Unnamed"
	}

	names, err = bot.DB.GetInvites(ctx.GetGuild().ID, names)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	var fields []discord.EmbedField

	for code, name := range names {
		fields = append(fields, discord.EmbedField{
			Name: code,
			Value: fmt.Sprintf(`%v
Uses: %v
Created by %v#%v`, name, invites[code].Uses, invites[code].Inviter.Username, invites[code].Inviter.Discriminator),
			Inline: true,
		})
	}

	_, _, err = ctx.ButtonPages(
		bcr.FieldPaginator("Invites", "", bcr.ColourPurple, fields, 4), 15*time.Minute,
	)
	return
}
