package commands

import (
	"fmt"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/common"
)

func (bot *Bot) createInvite(ctx bcr.Contexter) (err error) {
	ch, err := ctx.GetChannelFlag("channel")
	if err != nil || (ch.Type != discord.GuildNews && ch.Type != discord.GuildText) || ch.GuildID != ctx.GetGuild().ID {
		return ctx.SendEphemeral(channelNotFoundError)
	}

	perms, _ := ctx.Session().Permissions(ch.ID, ctx.User().ID)

	if !perms.Has(discord.PermissionViewChannel) {
		return ctx.SendEphemeral(channelNotFoundError)
	}

	if !perms.Has(discord.PermissionCreateInstantInvite) {
		return ctx.SendEphemeral("You don't have permission to create invites in that channel!")
	}

	if botPerms, _ := ctx.Session().Permissions(ch.ID, bot.Router.Bot.ID); !botPerms.Has(discord.PermissionViewChannel) || !botPerms.Has(discord.PermissionCreateInstantInvite) {
		return ctx.SendEphemeral(fmt.Sprintf("Either %v can't see %v, or does not have permission to create invites there.", bot.Router.Bot.Username, ch.Mention()))
	}

	inv, err := ctx.Session().CreateInvite(ch.ID, api.CreateInviteData{
		MaxAge:         option.NewUint(0),
		Temporary:      false,
		Unique:         true,
		AuditLogReason: api.AuditLogReason(fmt.Sprintf("%v (%v): create unique invite", ctx.User().Tag(), ctx.User().ID)),
	})
	if err != nil {
		common.Log.Errorf("Couldn't create invite in %v: %v", ch.ID, err)
		return ctx.SendEphemeral(fmt.Sprintf("Couldn't create an invite in %v. Are you sure %v has permission to create invites?", ch.Mention(), bot.Router.Bot.Username))
	}

	s := fmt.Sprintf("Success! Created invite https://discord.gg/%v for %v", inv.Code, ch.Mention())

	name := ctx.GetStringFlag("name")
	if name != "" {
		err = bot.DB.NameInvite(ctx.GetGuild().ID, inv.Code, name)
		if err != nil {
			return bot.DB.ReportCtx(ctx, err)
		}
		s += fmt.Sprintf(", with name **``%v``**.", bcr.EscapeBackticks(name))
	} else {
		s += "."
	}

	err = ctx.SendX(s)
	return
}
