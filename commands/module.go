package commands

import (
	"git.sr.ht/~starshine-sys/logger/db"
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/spf13/pflag"
	"github.com/starshine-sys/bcr"
	"go.uber.org/zap"
)

// Bot ...
type Bot struct {
	*bcr.Router

	DB    *db.DB
	Sugar *zap.SugaredLogger
}

// Init ...
func Init(r *bcr.Router, db *db.DB, s *zap.SugaredLogger) {
	b := &Bot{
		Router: r,
		DB:     db,
		Sugar:  s,
	}

	b.AddCommand(&bcr.Command{
		Name:    "events",
		Summary: "Show all available events.",

		Command: b.events,
	})

	b.AddCommand(&bcr.Command{
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

	b.AddCommand(&bcr.Command{
		Name:    "channels",
		Summary: "Show all currently logging events.",

		Permissions: discord.PermissionManageGuild,
		Command:     b.channels,
	})

	b.AddCommand(&bcr.Command{
		Name:    "permcheck",
		Summary: "Check the bot's permissions.",

		Permissions: discord.PermissionManageGuild,
		Command:     b.permcheck,
	})

	b.AddCommand(&bcr.Command{
		Name:    "ignore-channel",
		Aliases: []string{"ignorechannel"},
		Summary: "Ignore the current channel.",

		Permissions: discord.PermissionManageGuild,
		Command:     b.ignore,
	})

	b.AddCommand(&bcr.Command{
		Name:    "clear-data",
		Aliases: []string{"cleardata"},
		Summary: "**Clear all of this server's data.**",

		Permissions: discord.PermissionManageGuild,
		Command:     b.clearData,
	})

	h := b.AddCommand(&bcr.Command{
		Name:    "help",
		Summary: "Show information about the bot, or a specific command.",
		Usage:   "[command]",

		Command: b.help,
	})

	h.AddSubcommand(&bcr.Command{
		Name:    "permissions",
		Aliases: []string{"perms"},
		Summary: "Show a list of required permissions.",

		Command: b.perms,
	})

	b.AddCommand(&bcr.Command{
		Name:    "invite",
		Summary: "Get an invite link for the bot.",

		Command: b.invite,
	})

	inv := b.AddCommand(&bcr.Command{
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
}
