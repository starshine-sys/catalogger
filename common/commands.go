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
			&discord.SubcommandOption{
				OptionName:  "channels",
				Description: "Configure logging channels",
			},
		},
	},
}

var events = []discord.StringChoice{
	{Name: "Members joining", Value: "GUILD_MEMBER_ADD"},
	{Name: "Members leaving", Value: "GUILD_MEMBER_REMOVE"},
}
