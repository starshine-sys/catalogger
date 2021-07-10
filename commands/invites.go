package commands

import (
	"fmt"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/starshine-sys/bcr"
)

func (bot *Bot) listInvites(ctx *bcr.Context) (err error) {
	is, err := ctx.State.GuildInvites(ctx.Message.GuildID)
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

	names, err = bot.DB.GetInvites(ctx.Message.GuildID, names)
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

	_, err = ctx.PagedEmbed(
		bcr.FieldPaginator("Invites", "A list of invites", bcr.ColourPurple, fields, 4), false,
	)
	return
}
