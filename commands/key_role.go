package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/starshine-sys/bcr"
)

func (bot *Bot) keyroleList(ctx bcr.Contexter) (err error) {
	var keyRoles []uint64
	err = bot.DB.Pool.QueryRow(context.Background(), "select key_roles from guilds where id = $1", ctx.GetGuild().ID).Scan(&keyRoles)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	if len(keyRoles) == 0 {
		return ctx.SendX("There are no key roles set.")
	}

	s := make([]string, len(keyRoles))
	for i, r := range keyRoles {
		s[i] = fmt.Sprintf("<@&%v>\n", r)
	}

	_, _, err = ctx.ButtonPages(
		bcr.StringPaginator("Key roles", bcr.ColourPurple, s, 20), 10*time.Minute,
	)
	return err
}

func (bot *Bot) keyroleAddSlash(ctx bcr.Contexter) (err error) {
	var keyRoles []uint64
	err = bot.DB.Pool.QueryRow(context.Background(), "select key_roles from guilds where id = $1", ctx.GetGuild().ID).Scan(&keyRoles)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	r, err := ctx.GetRoleFlag("role")
	if err != nil {
		return ctx.SendX("Role not found.")
	}

	for _, kr := range keyRoles {
		if discord.RoleID(kr) == r.ID {
			return ctx.SendX("", discord.Embed{
				Description: fmt.Sprintf("%v is already a key role.", r.Mention()),
				Color:       bcr.ColourPurple,
			})
		}
	}

	_, err = bot.DB.Pool.Exec(context.Background(), "update guilds set key_roles = array_append(key_roles, $1) where id = $2", r.ID, ctx.GetGuild().ID)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	return ctx.SendX("", discord.Embed{
		Description: fmt.Sprintf("Added key role %v.", r.Mention()),
		Color:       bcr.ColourPurple,
	})
}

func (bot *Bot) keyroleRemoveSlash(ctx bcr.Contexter) (err error) {
	var keyRoles []uint64
	err = bot.DB.Pool.QueryRow(context.Background(), "select key_roles from guilds where id = $1", ctx.GetGuild().ID).Scan(&keyRoles)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	r, err := ctx.GetRoleFlag("role")
	if err != nil {
		return ctx.SendX("Role not found.")
	}

	var isKeyRole bool
	for _, kr := range keyRoles {
		if discord.RoleID(kr) == r.ID {
			isKeyRole = true
			break
		}
	}

	if !isKeyRole {
		return ctx.SendX("", discord.Embed{
			Description: fmt.Sprintf("%v is not a key role.", r.Mention()),
			Color:       bcr.ColourPurple,
		})
	}

	_, err = bot.DB.Pool.Exec(context.Background(), "update guilds set key_roles = array_remove(key_roles, $1) where id = $2", r.ID, ctx.GetGuild().ID)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	return ctx.SendX("", discord.Embed{
		Description: fmt.Sprintf("Removed key role %v.", r.Mention()),
		Color:       bcr.ColourPurple,
	})
}

func (bot *Bot) keyroleAdd(ctx *bcr.Context) (err error) {
	var keyRoles []uint64
	err = bot.DB.Pool.QueryRow(context.Background(), "select key_roles from guilds where id = $1", ctx.Guild.ID).Scan(&keyRoles)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	r, err := ctx.ParseRole(ctx.RawArgs)
	if err != nil {
		_, err = ctx.Reply("Role not found.")
		return
	}

	for _, kr := range keyRoles {
		if discord.RoleID(kr) == r.ID {
			_, err = ctx.Reply("%v is already a key role.", r.Mention())
			return
		}
	}

	_, err = bot.DB.Pool.Exec(context.Background(), "update guilds set key_roles = array_append(key_roles, $1) where id = $2", r.ID, ctx.Message.GuildID)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	_, err = ctx.Reply("Added key role %v.", r.Mention())
	return
}

func (bot *Bot) keyroleRemove(ctx *bcr.Context) (err error) {
	var keyRoles []uint64
	err = bot.DB.Pool.QueryRow(context.Background(), "select key_roles from guilds where id = $1", ctx.Guild.ID).Scan(&keyRoles)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	r, err := ctx.ParseRole(ctx.RawArgs)
	if err != nil {
		_, err = ctx.Reply("Role not found.")
		return
	}

	var isKeyRole bool
	for _, kr := range keyRoles {
		if discord.RoleID(kr) == r.ID {
			isKeyRole = true
			break
		}
	}

	if !isKeyRole {
		_, err = ctx.Reply("%v is not a key role.", r.Mention())
		return
	}

	_, err = bot.DB.Pool.Exec(context.Background(), "update guilds set key_roles = array_remove(key_roles, $1) where id = $2", r.ID, ctx.Message.GuildID)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	_, err = ctx.Reply("Removed key role %v.", r.Mention())
	return
}
