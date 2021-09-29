package commands

import (
	"fmt"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/db"
)

func (bot *Bot) resetChannel(ctx bcr.Contexter) (err error) {
	chs, err := bot.DB.Channels(ctx.GetGuild().ID)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	eventName := ctx.GetStringFlag("event")
	if v, ok := ctx.(*bcr.Context); ok {
		eventName = v.RawArgs
	}

	if _, ok := db.DefaultEventMap[eventName]; !ok {
		return ctx.SendEphemeral(fmt.Sprintf("\"%v\" isn't a valid event name.", eventName))
	}

	chs[eventName] = 0

	err = bot.DB.SetChannels(ctx.GetGuild().ID, chs)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	err = ctx.SendfX("Success! ``%v`` events will no longer be logged.", eventName)
	if err != nil {
		return
	}

	// won't respond 'cause the interaction has already been replied to, but hey, we only care about the cache stuff
	bot.Router.GetCommand("clearcache").SlashCommand(ctx)

	return
}

func (bot *Bot) setChannelSlash(ctx bcr.Contexter) (err error) {
	chs, err := bot.DB.Channels(ctx.GetGuild().ID)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	eventName := ctx.GetStringFlag("event")
	if _, ok := db.DefaultEventMap[eventName]; !ok {
		return ctx.SendEphemeral(fmt.Sprintf("\"%v\" isn't a valid event name.", eventName))
	}

	ch, err := ctx.GetChannelFlag("channel")
	if err != nil || ch.GuildID != ctx.GetGuild().ID || (ch.Type != discord.GuildNews && ch.Type != discord.GuildText) {
		return ctx.SendEphemeral(channelNotFoundError)
	}

	if uperm, _ := ctx.Session().Permissions(ch.ID, ctx.User().ID); !uperm.Has(discord.PermissionViewChannel) {
		return ctx.SendEphemeral(channelNotFoundError)
	}

	botPerm, _ := ctx.Session().Permissions(ch.ID, bot.Router.Bot.ID)
	if !botPerm.Has(discord.PermissionViewChannel) || !botPerm.Has(discord.PermissionManageWebhooks) {
		return ctx.SendEphemeral(
			fmt.Sprintf("%v does not have permission to view %v, or does not have permission to create webhooks there.", bot.Router.Bot.Username, ch.Mention()),
		)
	}

	chs[eventName] = ch.ID

	err = bot.DB.SetChannels(ctx.GetGuild().ID, chs)
	if err != nil {
		return bot.DB.ReportCtx(ctx, err)
	}

	err = ctx.SendfX("Success! ``%v`` events will now be logged to %v.", eventName, ch.Mention())
	if err != nil {
		return
	}

	// won't respond 'cause the interaction has already been replied to, but hey, we only care about the cache stuff
	bot.Router.GetCommand("clearcache").SlashCommand(ctx)

	return
}
