package commands

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/spf13/pflag"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/bot"
)

// Bot ...
type Bot struct {
	*bot.Bot
}

// Init ...
func Init(bot *bot.Bot) {
	b := &Bot{
		Bot: bot,
	}

	b.Router.AddCommand(&bcr.Command{
		Name:    "events",
		Summary: "Show all available events.",

		SlashCommand: b.events,
		Options:      &[]discord.CommandOption{},
	})

	b.Router.AddCommand(&bcr.Command{
		Name:        "setchannel",
		Summary:     "Set the given event(s) to log in the current channel.",
		Description: "Set the given event(s) to log in the current channel.\nSeparate events with commas.\nUse `--clear` to disable the event.\nUse `events` for a list of valid events.",
		Usage:       "<events...>",
		Args:        bcr.MinArgs(1),
		Flags: func(fs *pflag.FlagSet) *pflag.FlagSet {
			fs.BoolP("clear", "c", false, "Disable the given event.")
			return fs
		},

		Permissions: discord.PermissionManageGuild,
		Command:     b.setChannel,
	})

	b.Router.AddCommand(&bcr.Command{
		Name:        "redirect",
		Summary:     "Show channels being redirected, or change where a channel is being redirected to.",
		Description: "Show channels being redirected, or change where a channel is being redirected to.\nUse `--clear` or `clear` to reset to the default log channel.",

		Usage: "[<source> <destination|--clear>]",

		Permissions: discord.PermissionManageGuild,
		Command:     b.redirect,
	})

	b.Router.AddCommand(&bcr.Command{
		Name:    "channels",
		Summary: "Show all currently logging events.",

		Permissions:  discord.PermissionManageGuild,
		SlashCommand: b.channels,
		Options:      &[]discord.CommandOption{},
	})

	b.Router.AddCommand(&bcr.Command{
		Name:    "permcheck",
		Summary: "Check the bot's permissions.",

		Permissions:  discord.PermissionManageGuild,
		SlashCommand: b.permcheck,
		Options:      &[]discord.CommandOption{},
	})

	b.Router.AddCommand(&bcr.Command{
		Name:    "ignore-channel",
		Aliases: []string{"ignorechannel", "ignore"},
		Summary: "Ignore the given channel.",
		Usage:   "<channel>",
		Args:    bcr.MinArgs(1),

		Permissions:  discord.PermissionManageGuild,
		SlashCommand: b.ignore,
		Options: &[]discord.CommandOption{{
			Type:        discord.ChannelOption,
			Name:        "channel",
			Description: "The channel to ignore.",
			Required:    true,
		}},
	})

	b.Router.AddCommand(&bcr.Command{
		Name:    "clear-data",
		Aliases: []string{"cleardata"},
		Summary: "Clear ___all___ of this server's data.",

		Permissions:  discord.PermissionManageGuild,
		SlashCommand: b.clearData,
	})

	h := b.Router.AddCommand(&bcr.Command{
		Name:    "help",
		Summary: "Show information about the bot, or a specific command.",
		Usage:   "[command]",

		SlashCommand: b.help,
	})

	h.AddSubcommand(&bcr.Command{
		Name:    "permissions",
		Aliases: []string{"perms"},
		Summary: "Show a list of required permissions.",

		SlashCommand: b.perms,
	})

	h.AddSubcommand(&bcr.Command{
		Name:    "commands",
		Aliases: []string{"cmds"},
		Summary: "Show a list of all commands",

		SlashCommand: b.commands,
	})

	helpGroup := &bcr.Group{
		Name:        "help",
		Description: "Show information about the bot.",
	}

	helpGroup.Add(&bcr.Command{
		Name:         "info",
		Summary:      "Show information about the bot.",
		SlashCommand: b.help,
	})

	helpGroup.Add(&bcr.Command{
		Name:         "permissions",
		Summary:      "Show a list of required permissions.",
		SlashCommand: b.perms,
	})

	helpGroup.Add(&bcr.Command{
		Name:         "commands",
		Summary:      "Show a list of all commands",
		SlashCommand: b.commands,
	})

	helpGroup.Add(&bcr.Command{
		Name:         "invite",
		Summary:      "Get an invite link for the bot.",
		SlashCommand: b.invite,
	})

	helpGroup.Add(&bcr.Command{
		Name:         "dashboard",
		Summary:      "Get a link to the bot dashboard.",
		SlashCommand: b.dashboard,
	})

	b.Router.AddCommand(&bcr.Command{
		Name:         "invite",
		Summary:      "Get an invite link for the bot.",
		SlashCommand: b.invite,
	})

	inv := b.Router.AddCommand(&bcr.Command{
		Name:    "invites",
		Summary: "List this server's invites.",

		Permissions: discord.PermissionManageGuild,
		Command:     b.listInvites,
	})

	inv.AddSubcommand(&bcr.Command{
		Name:    "name",
		Aliases: []string{"rename"},
		Summary: "Set an invite's name.",
		Usage:   "<invite code> <new name>",
		Args:    bcr.MinArgs(2),

		Permissions: discord.PermissionManageGuild,
		Command:     b.renameInvite,
	})

	b.Router.AddCommand(&bcr.Command{
		Name:    "admin-stats",
		Aliases: []string{"adminstats"},
		Summary: "Per-server message stats.",

		Hidden:    true,
		OwnerOnly: true,
		Command:   b.adminStats,
	})

	b.Router.AddCommand(&bcr.Command{
		Name:         "dashboard",
		Summary:      "Get a link to the bot dashboard.",
		SlashCommand: b.dashboard,
	})

	wl := b.Router.AddCommand(&bcr.Command{
		Name:    "watchlist",
		Aliases: []string{"wl"},
		Summary: "Show or manage this server's user watchlist.",

		Permissions: discord.PermissionKickMembers,
		Command:     b.watchlist,
	})

	wl.AddSubcommand(&bcr.Command{
		Name:    "add",
		Summary: "Add a user to the watchlist.",
		Usage:   "<user> [reason]",
		Args:    bcr.MinArgs(1),

		Permissions: discord.PermissionKickMembers,
		Command:     b.watchlistAdd,
	})

	wl.AddSubcommand(&bcr.Command{
		Name:    "remove",
		Summary: "Remove a user from the watchlist.",
		Usage:   "<user>",
		Args:    bcr.MinArgs(1),

		Permissions: discord.PermissionKickMembers,
		Command:     b.watchlistRemove,
	})

	b.Router.AddGroup(helpGroup)
}
