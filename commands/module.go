package commands

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/spf13/pflag"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/bot"
	"github.com/starshine-sys/catalogger/db"
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
	})

	var choices []discord.StringChoice
	for _, ev := range db.Events {
		choices = append(choices, discord.StringChoice{
			Name:  db.EventDescs[ev],
			Value: ev,
		})
	}

	b.Router.AddCommand(&bcr.Command{
		Name:        "set-channel",
		Aliases:     []string{"setchannel"},
		Summary:     "Set the given event to log in a channel.",
		Description: "Set the given event(s) to log in the current channel.\nSeparate events with commas.\nUse `--clear` to disable the event.\nUse `events` for a list of valid events.",
		Usage:       "<events...>",
		Args:        bcr.MinArgs(1),
		Flags: func(fs *pflag.FlagSet) *pflag.FlagSet {
			fs.BoolP("clear", "c", false, "Disable the given event.")
			return fs
		},

		Permissions: discord.PermissionManageGuild,
		Command:     b.setChannel,

		SlashCommand: b.setChannelSlash,
		Options: &[]discord.CommandOption{
			&discord.ChannelOption{
				OptionName:   "channel",
				Description:  "The channel to log to.",
				Required:     true,
				ChannelTypes: []discord.ChannelType{discord.GuildNews, discord.GuildText},
			},
			&discord.StringOption{
				OptionName:  "event",
				Description: "The event to log.",
				Required:    true,
				Choices:     choices,
			},
		},
	})

	b.Router.AddCommand(&bcr.Command{
		Name:         "reset-event",
		Aliases:      []string{"reset-channel", "resetevent", "resetchannel"},
		Summary:      "Stop logging the given event.",
		Usage:        "<event>",
		Args:         bcr.MinArgs(1),
		SlashCommand: b.resetChannel,
		Options: &[]discord.CommandOption{&discord.StringOption{
			OptionName:  "event",
			Description: "The event to log.",
			Required:    true,
			Choices:     choices,
		}},
	})

	b.Router.AddCommand(&bcr.Command{
		Name:        "redirect",
		Summary:     "Show channels being redirected, or change where a channel is being redirected to.",
		Description: "Show channels being redirected, or change where a channel is being redirected to.\nUse `--clear` or `clear` to reset to the default log channel.",

		Usage: "[<source> <destination|--clear>]",

		Permissions: discord.PermissionManageGuild,
		Command:     b.redirect,
	})

	b.Router.AddGroup(&bcr.Group{
		Name:        "redirect",
		Description: "View or manage redirected logs.",
		Subcommands: []*bcr.Command{
			{
				Name:         "list",
				Summary:      "Show a list of all currently redirecting channels.",
				Permissions:  discord.PermissionManageGuild,
				SlashCommand: b.redirectList,
			},
			{
				Name:         "to",
				Summary:      "Redirect logs from a channel to another channel.",
				Permissions:  discord.PermissionManageGuild,
				SlashCommand: b.redirectTo,
				Options: &[]discord.CommandOption{
					&discord.ChannelOption{
						OptionName:   "from",
						Description:  "Text channel to redirect logs from.",
						Required:     true,
						ChannelTypes: []discord.ChannelType{discord.GuildNews, discord.GuildText},
					},
					&discord.ChannelOption{
						OptionName:   "to",
						Description:  "Text channel to redirect logs to.",
						Required:     true,
						ChannelTypes: []discord.ChannelType{discord.GuildNews, discord.GuildText},
					},
				},
			},
			{
				Name:         "remove",
				Summary:      "Reset a channel's logs, making them log to the default log channel again.",
				Permissions:  discord.PermissionManageGuild,
				SlashCommand: b.redirectRemove,
				Options: &[]discord.CommandOption{&discord.ChannelOption{
					OptionName:   "from",
					Description:  "Text channel to redirect logs from.",
					Required:     true,
					ChannelTypes: []discord.ChannelType{discord.GuildNews, discord.GuildText},
				}},
			},
		},
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
		Options: &[]discord.CommandOption{&discord.ChannelOption{
			OptionName:   "channel",
			Description:  "The channel to ignore.",
			Required:     true,
			ChannelTypes: []discord.ChannelType{discord.GuildNews, discord.GuildText, discord.GuildVoice, discord.GuildStageVoice},
		}},
	})

	b.Router.AddCommand(&bcr.Command{
		Name:    "clear-data",
		Aliases: []string{"cleardata"},
		Summary: "Clear ___all___ of this server's data.",

		Permissions:  discord.PermissionManageGuild,
		SlashCommand: b.clearData,
		Options:      &[]discord.CommandOption{},
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
		Name:         "invite",
		Summary:      "Get an invite link for the bot.",
		SlashCommand: b.invite,
	})

	helpGroup.Add(&bcr.Command{
		Name:         "dashboard",
		Summary:      "Get a link to the bot dashboard.",
		SlashCommand: b.dashboard,
	})

	helpGroup.Add(&bcr.Command{
		Name:         "events",
		Summary:      "Show all available events.",
		SlashCommand: b.events,
	})

	b.Router.AddCommand(&bcr.Command{
		Name:         "invite",
		Summary:      "Get an invite link for the bot.",
		SlashCommand: b.invite,
	})

	inv := b.Router.AddCommand(&bcr.Command{
		Name:    "invites",
		Summary: "List this server's invites.",

		Permissions:  discord.PermissionManageGuild,
		SlashCommand: b.listInvites,
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

	invGroup := &bcr.Group{
		Name:        "invites",
		Description: "List and manage this server's invites.",
	}

	invGroup.Add(&bcr.Command{
		Name:         "list",
		Summary:      "List this server's invites.",
		Permissions:  discord.PermissionManageGuild,
		SlashCommand: b.listInvites,
	})

	invGroup.Add(&bcr.Command{
		Name:         "name",
		Summary:      "Set or reset an invite's name.",
		Permissions:  discord.PermissionManageGuild,
		SlashCommand: b.renameInviteSlash,
		Options: &[]discord.CommandOption{
			&discord.StringOption{
				OptionName:  "code",
				Description: "The invite to name.",
				Required:    true,
			},
			&discord.StringOption{
				OptionName:  "name",
				Description: "The name to give to the invite. Leave empty to reset the invite's name.",
				Required:    false,
			},
		},
	})

	invGroup.Add(&bcr.Command{
		Name:         "create",
		Summary:      "Create a unique invite for a channel.",
		Permissions:  discord.PermissionManageGuild,
		SlashCommand: b.createInvite,
		Options: &[]discord.CommandOption{
			&discord.ChannelOption{
				OptionName:   "channel",
				ChannelTypes: []discord.ChannelType{discord.GuildNews, discord.GuildText},
				Description:  "The channel to create an invite in.",
				Required:     true,
			},
			&discord.StringOption{
				OptionName:  "name",
				Description: "What to name the new invite.",
				Required:    false,
			},
		},
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

	b.Router.AddGroup(&bcr.Group{
		Name:        "watchlist",
		Description: "Show or manage this server's user watchlist.",
		Subcommands: []*bcr.Command{
			{
				Name:         "show",
				Summary:      "Show the user watchlist.",
				Permissions:  discord.PermissionKickMembers,
				SlashCommand: b.watchlistSlash,
			},
			{
				Name:         "add",
				Summary:      "Add a user to the watchlist.",
				Permissions:  discord.PermissionKickMembers,
				SlashCommand: b.watchlistAddSlash,
				Options: &[]discord.CommandOption{
					&discord.UserOption{
						OptionName: "user",

						Description: "The user to add.",
						Required:    true,
					},
					&discord.StringOption{
						OptionName:  "reason",
						Description: "Why you're adding this user.",
						Required:    false,
					},
				},
			},
			{
				Name:         "remove",
				Summary:      "Remove a user from the watchlist.",
				Permissions:  discord.PermissionKickMembers,
				SlashCommand: b.watchlistRemoveSlash,
				Options: &[]discord.CommandOption{&discord.UserOption{
					OptionName:  "user",
					Description: "The user to remove.",
					Required:    true,
				}},
			},
		},
	})

	keyroles := b.Router.AddCommand(&bcr.Command{
		Name:         "keyrole",
		Aliases:      []string{"keyroles", "key-role", "key-roles"},
		Summary:      "List and manage this server's key roles.",
		SlashCommand: b.keyroleList,
		Permissions:  discord.PermissionManageGuild,
	})

	keyroles.AddSubcommand(&bcr.Command{
		Name:        "add",
		Summary:     "Add a key role.",
		Usage:       "<role>",
		Args:        bcr.MinArgs(1),
		Command:     b.keyroleAdd,
		Permissions: discord.PermissionManageGuild,
	})

	keyroles.AddSubcommand(&bcr.Command{
		Name:        "remove",
		Summary:     "Remove a key role.",
		Usage:       "<role>",
		Args:        bcr.MinArgs(1),
		Command:     b.keyroleRemove,
		Permissions: discord.PermissionManageGuild,
	})

	ignoredUsers := b.Router.AddCommand(&bcr.Command{
		Name:         "ignore-users",
		Aliases:      []string{"ignored-users", "ignore-user"},
		Summary:      "List and manage this server's ignored users.",
		SlashCommand: b.ignoreUsersList,
		Permissions:  discord.PermissionManageGuild,
	})

	ignoredUsers.AddSubcommand(&bcr.Command{
		Name:        "add",
		Aliases:     []string{"ignore"},
		Summary:     "Add a user to be ignored.",
		Usage:       "<user>",
		Args:        bcr.MinArgs(1),
		Command:     b.ignoreUsersAdd,
		Permissions: discord.PermissionManageGuild,
	})

	ignoredUsers.AddSubcommand(&bcr.Command{
		Name:        "remove",
		Aliases:     []string{"unignore"},
		Summary:     "Unignore a user.",
		Usage:       "<user>",
		Args:        bcr.MinArgs(1),
		Command:     b.ignoreUsersRemove,
		Permissions: discord.PermissionManageGuild,
	})

	b.Router.AddGroup(&bcr.Group{
		Name:        "keyroles",
		Description: "List and manage this server's key roles.",
		Subcommands: []*bcr.Command{
			{
				Name:         "list",
				Summary:      "List the current key roles.",
				SlashCommand: b.keyroleList,
				Permissions:  discord.PermissionManageGuild,
			},
			{
				Name:         "add",
				Summary:      "Add a key role.",
				SlashCommand: b.keyroleAddSlash,
				Permissions:  discord.PermissionManageGuild,
				Options: &[]discord.CommandOption{&discord.RoleOption{
					OptionName:  "role",
					Description: "The role to add.",
					Required:    true,
				}},
			},
			{
				Name:         "remove",
				Summary:      "Remove a key role.",
				SlashCommand: b.keyroleRemoveSlash,
				Permissions:  discord.PermissionManageGuild,
				Options: &[]discord.CommandOption{&discord.RoleOption{
					OptionName:  "role",
					Description: "The role to remove.",
					Required:    true,
				}},
			},
		},
	})

	b.Router.AddGroup(&bcr.Group{
		Name:        "ignore-users",
		Description: "Add and remove ignored users.",
		Subcommands: []*bcr.Command{
			{
				Name:         "list",
				Summary:      "List the currently ignored users.",
				SlashCommand: b.ignoreUsersList,
				Permissions:  discord.PermissionManageGuild,
			},
			{
				Name:         "add",
				Summary:      "Ignore a user.",
				SlashCommand: b.ignoreUsersAddSlash,
				Permissions:  discord.PermissionManageGuild,
				Options: &[]discord.CommandOption{&discord.UserOption{
					OptionName:  "user",
					Description: "The user to ignore.",
					Required:    true,
				}},
			},
			{
				Name:         "remove",
				Summary:      "Stop ignoring a user.",
				SlashCommand: b.ignoreUsersRemoveSlash,
				Permissions:  discord.PermissionManageGuild,
				Options: &[]discord.CommandOption{&discord.UserOption{
					OptionName:  "user",
					Description: "The user to stop ignoring.",
					Required:    true,
				}},
			},
		},
	})

	b.Router.AddGroup(helpGroup)
	b.Router.AddGroup(invGroup)
}
