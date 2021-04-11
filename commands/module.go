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
		Name:    "ping",
		Summary: "Show the bot's latency.",

		Command: b.ping,
	})

	b.AddCommand(&bcr.Command{
		Name:    "events",
		Summary: "Show all available events.",

		Command: b.events,
	})

	b.AddCommand(&bcr.Command{
		Name:        "setchannel",
		Summary:     "Set the current channel for a given event.",
		Description: "Set the current channel for a given event.\nUse `--clear` to disable the event.\nUse `events` for a list of valid events.",
		Usage:       "<event>",
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
		Name:    "help",
		Summary: "Show information about the bot, or a specific command.",
		Usage:   "[command]",

		Command: b.help,
	})

	b.AddCommand(&bcr.Command{
		Name:    "invite",
		Summary: "Get an invite link for the bot.",

		Command: b.invite,
	})
}
