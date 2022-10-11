package common

import (
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
)

var Commands = []api.CreateCommandData{
	{
		Name:                     "config",
		Description:              "Configure Catalogger",
		DefaultMemberPermissions: discord.NewPermissions(discord.PermissionManageGuild),
		Options: discord.CommandOptions{
			&discord.SubcommandGroupOption{
				OptionName:  "event",
				Description: "Event settings",
				Subcommands: []*discord.SubcommandOption{
					{
						OptionName:  "set",
						Description: "Set an event to log to the given channel",
						Options: []discord.CommandOptionValue{
							&discord.StringOption{
								OptionName:  "event",
								Description: "The event to log",
								Required:    true,
								Choices:     events,
							},
							&discord.ChannelOption{
								OptionName:   "channel",
								Description:  "The channel to log to",
								Required:     true,
								ChannelTypes: []discord.ChannelType{discord.GuildText, discord.GuildNews},
							},
						},
					},
					{
						OptionName:  "disable",
						Description: "Disable logging an event",
						Options: []discord.CommandOptionValue{
							&discord.StringOption{
								OptionName:  "event",
								Description: "The event to stop logging",
								Required:    true,
								Choices:     events,
							},
						},
					},
				},
			},
		},
	},
}

var events = []discord.StringChoice{
	{Name: "Members joining", Value: "GUILD_MEMBER_ADD"},
	{Name: "Members leaving", Value: "GUILD_MEMBER_REMOVE"},
}
