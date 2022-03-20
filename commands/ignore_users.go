package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/starshine-sys/bcr"
)

func (bot *Bot) ignoreUsersList(ctx bcr.Contexter) (err error) {
	var users []uint64
	err = bot.DB.Pool.QueryRow(context.Background(), "select ignored_users from guilds where id = $1", ctx.GetGuild().ID).Scan(&users)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	if len(users) == 0 {
		return ctx.SendX("There are no ignored users.")
	}

	s := make([]string, len(users))
	for i, r := range users {
		s[i] = fmt.Sprintf("<@!%v>\n", r)
	}

	_, _, err = ctx.ButtonPages(
		bcr.StringPaginator("Ignored users", bcr.ColourPurple, s, 20), 10*time.Minute,
	)
	return err
}

func (bot *Bot) ignoreUsersAddSlash(ctx bcr.Contexter) (err error) {
	var users []uint64
	err = bot.DB.Pool.QueryRow(context.Background(), "select ignored_users from guilds where id = $1", ctx.GetGuild().ID).Scan(&users)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	u, err := ctx.GetUserFlag("user")
	if err != nil {
		return ctx.SendX("Role not found.")
	}

	for _, id := range users {
		if discord.UserID(id) == u.ID {
			return ctx.SendX("", discord.Embed{
				Description: fmt.Sprintf("%v is already being ignored.", u.Mention()),
				Color:       bcr.ColourPurple,
			})
		}
	}

	_, err = bot.DB.Pool.Exec(context.Background(), "update guilds set ignored_users = array_append(ignored_users, $1) where id = $2", u.ID, ctx.GetGuild().ID)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	return ctx.SendX("", discord.Embed{
		Description: fmt.Sprintf("Messages sent by %v will now be ignored.", u.Mention()),
		Color:       bcr.ColourPurple,
	})
}

func (bot *Bot) ignoreUsersRemoveSlash(ctx bcr.Contexter) (err error) {
	var users []uint64
	err = bot.DB.Pool.QueryRow(context.Background(), "select ignored_users from guilds where id = $1", ctx.GetGuild().ID).Scan(&users)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	u, err := ctx.GetUserFlag("user")
	if err != nil {
		return ctx.SendX("Role not found.")
	}

	var isIgnored bool
	for _, id := range users {
		if discord.UserID(id) == u.ID {
			isIgnored = true
			break
		}
	}

	if !isIgnored {
		return ctx.SendX("", discord.Embed{
			Description: fmt.Sprintf("%v is not being ignored.", u.Mention()),
			Color:       bcr.ColourPurple,
		})
	}

	_, err = bot.DB.Pool.Exec(context.Background(), "update guilds set ignored_users = array_remove(ignored_users, $1) where id = $2", u.ID, ctx.GetGuild().ID)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	return ctx.SendX("", discord.Embed{
		Description: fmt.Sprintf("Messages sent by %v will no longer be ignored.", u.Mention()),
		Color:       bcr.ColourPurple,
	})
}

func (bot *Bot) ignoreUsersAdd(ctx *bcr.Context) (err error) {
	var users []uint64
	err = bot.DB.Pool.QueryRow(context.Background(), "select ignored_users from guilds where id = $1", ctx.Guild.ID).Scan(&users)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	u, err := ctx.ParseUser(ctx.RawArgs)
	if err != nil {
		_, err = ctx.Reply("No user with that name found.")
		return
	}

	for _, id := range users {
		if discord.UserID(id) == u.ID {
			_, err = ctx.Reply("%v is already being ignored.", u.Mention())
			return
		}
	}

	_, err = bot.DB.Pool.Exec(context.Background(), "update guilds set ignored_users = array_append(ignored_users, $1) where id = $2", u.ID, ctx.Message.GuildID)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	_, err = ctx.Reply("Messages sent by %v will now be ignored.", u.Mention())
	return
}

func (bot *Bot) ignoreUsersRemove(ctx *bcr.Context) (err error) {
	var users []uint64
	err = bot.DB.Pool.QueryRow(context.Background(), "select ignored_users from guilds where id = $1", ctx.Guild.ID).Scan(&users)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	u, err := ctx.ParseUser(ctx.RawArgs)
	if err != nil {
		_, err = ctx.Reply("No user with that ID found.")
		return
	}

	var isIgnored bool
	for _, id := range users {
		if discord.UserID(id) == u.ID {
			isIgnored = true
			break
		}
	}

	if !isIgnored {
		_, err = ctx.Reply("%v is not being ignored.", u.Mention())
		return
	}

	_, err = bot.DB.Pool.Exec(context.Background(), "update guilds set ignored_users = array_remove(ignored_users, $1) where id = $2", u.ID, ctx.Message.GuildID)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	_, err = ctx.Reply("Messages sent by %v will no longer be ignored.", u.Mention())
	return
}
