package commands

import (
	"fmt"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/common"
)

func (bot *Bot) listInvites(ctx bcr.Contexter) (err error) {
	is, err := ctx.Session().GuildInvites(ctx.GetGuild().ID)
	if err != nil {
		common.Log.Errorf("Error getting guild invites: %v", err)
		_, err = ctx.Sendf("Could not get this server's invites. Are you sure I have the **Manage Server** permission?")
		return
	}

	if len(is) == 0 {
		_, err = ctx.Send("This server has no invites.")
		return
	}

	invites := map[string]discord.Invite{}
	for _, i := range is {
		i := i
		invites[i.Code] = i
	}

	names, err := bot.DB.GetInvites(ctx.GetGuild().ID)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	var fields []discord.EmbedField

	for code, invite := range invites {
		fields = append(fields, discord.EmbedField{
			Name: code,
			Value: fmt.Sprintf(`%v
Uses: %v
Created by %v`, names.Name(code), invite.Uses, invite.Inviter.Tag()),
			Inline: true,
		})
	}

	_, _, err = ctx.ButtonPages(
		bcr.FieldPaginator("Invites", "", bcr.ColourPurple, fields, 4), 15*time.Minute,
	)
	return
}
