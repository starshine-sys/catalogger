package config

import (
	"context"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/starshine-sys/bcr/v2"
	"github.com/starshine-sys/catalogger/v2/common"
	"github.com/starshine-sys/catalogger/v2/common/log"
)

var channelButtons = discord.ContainerComponents{
	&discord.ActionRowComponent{
		&discord.ButtonComponent{
			Label:    "Server changes",
			CustomID: "channel:GUILD_UPDATE",
			Style:    discord.PrimaryButtonStyle(),
		},
		&discord.ButtonComponent{
			Label:    "Emote changes",
			CustomID: "channel:GUILD_EMOJIS_UPDATE",
			Style:    discord.PrimaryButtonStyle(),
		},
		&discord.ButtonComponent{
			Label:    "New roles",
			CustomID: "channel:GUILD_ROLE_CREATE",
			Style:    discord.PrimaryButtonStyle(),
		},
		&discord.ButtonComponent{
			Label:    "Edited roles",
			CustomID: "channel:GUILD_ROLE_UPDATE",
			Style:    discord.PrimaryButtonStyle(),
		},
		&discord.ButtonComponent{
			Label:    "Deleted roles",
			CustomID: "channel:GUILD_ROLE_DELETE",
			Style:    discord.PrimaryButtonStyle(),
		},
	},
	&discord.ActionRowComponent{
		&discord.ButtonComponent{
			Label:    "New channels",
			CustomID: "channel:CHANNEL_CREATE",
			Style:    discord.PrimaryButtonStyle(),
		},
		&discord.ButtonComponent{
			Label:    "Edited channels",
			CustomID: "channel:CHANNEL_UPDATE",
			Style:    discord.PrimaryButtonStyle(),
		},
		&discord.ButtonComponent{
			Label:    "Deleted channels",
			CustomID: "channel:CHANNEL_DELETE",
			Style:    discord.PrimaryButtonStyle(),
		},
		&discord.ButtonComponent{
			Label:    "Member joins",
			CustomID: "channel:GUILD_MEMBER_ADD",
			Style:    discord.PrimaryButtonStyle(),
		},
		&discord.ButtonComponent{
			Label:    "Member leaves",
			CustomID: "channel:GUILD_MEMBER_REMOVE",
			Style:    discord.PrimaryButtonStyle(),
		},
	},
	&discord.ActionRowComponent{
		&discord.ButtonComponent{
			Label:    "Member role changes",
			CustomID: "channel:GUILD_MEMBER_UPDATE",
			Style:    discord.PrimaryButtonStyle(),
		},
		&discord.ButtonComponent{
			Label:    "Key role changes",
			CustomID: "channel:GUILD_KEY_ROLE_UPDATE",
			Style:    discord.PrimaryButtonStyle(),
		},
		&discord.ButtonComponent{
			Label:    "Member name changes",
			CustomID: "channel:GUILD_MEMBER_NICK_UPDATE",
			Style:    discord.PrimaryButtonStyle(),
		},
		&discord.ButtonComponent{
			Label:    "Avatar changes",
			CustomID: "channel:GUILD_MEMBER_AVATAR_UPDATE",
			Style:    discord.PrimaryButtonStyle(),
		},
		&discord.ButtonComponent{
			Label:    "Kicks",
			CustomID: "channel:GUILD_MEMBER_KICK",
			Style:    discord.PrimaryButtonStyle(),
		},
	},
	&discord.ActionRowComponent{
		&discord.ButtonComponent{
			Label:    "Bans",
			CustomID: "channel:GUILD_BAN_ADD",
			Style:    discord.PrimaryButtonStyle(),
		},
		&discord.ButtonComponent{
			Label:    "Unbans",
			CustomID: "channel:GUILD_BAN_REMOVE",
			Style:    discord.PrimaryButtonStyle(),
		},
		&discord.ButtonComponent{
			Label:    "New invites",
			CustomID: "channel:INVITE_CREATE",
			Style:    discord.PrimaryButtonStyle(),
		},
		&discord.ButtonComponent{
			Label:    "Deleted invites",
			CustomID: "channel:INVITE_DELETE",
			Style:    discord.PrimaryButtonStyle(),
		},
		&discord.ButtonComponent{
			Label:    "Edited messages",
			CustomID: "channel:MESSAGE_UPDATE",
			Style:    discord.PrimaryButtonStyle(),
		},
	},
	&discord.ActionRowComponent{
		&discord.ButtonComponent{
			Label:    "Deleted messages",
			CustomID: "channel:MESSAGE_DELETE",
			Style:    discord.PrimaryButtonStyle(),
		},
		&discord.ButtonComponent{
			Label:    "Bulk deleted messages",
			CustomID: "channel:MESSAGE_DELETE_BULK",
			Style:    discord.PrimaryButtonStyle(),
		},
		&discord.ButtonComponent{
			Label:    "Close",
			CustomID: "channel:close",
			Style:    discord.SecondaryButtonStyle(),
		},
	},
}

func (bot *Bot) channelsEntry(ctx *bcr.CommandContext) (err error) {
	logChannels, err := bot.DB.Channels(ctx.Event.GuildID)
	if err != nil {
		log.Errorf("getting log channels for guild %v: %v", ctx.Event.GuildID, err)
		return err
	}
	guildChannels, err := bot.Cabinet.Channels(context.Background(), ctx.Event.GuildID)
	if err != nil {
		log.Errorf("getting guild channels for guild %v: %v", ctx.Event.GuildID, err)
		return err
	}

	prettyChannelString := func(id discord.ChannelID) string {
		if !id.IsValid() {
			return "Not set"
		}

		var found bool
		for _, ch := range guildChannels {
			if id == ch.ID {
				found = true
				break
			}
		}

		if !found {
			return "*unknown channel " + id.String() + "*"
		}
		return id.Mention()
	}

	embed := discord.Embed{
		Title:       "Log channels for " + ctx.Guild.Name,
		Description: "Click one of the buttons below to change the channel for that event.",
		Color:       common.ColourPurple,
		Fields: []discord.EmbedField{
			{Name: "Server changes", Value: prettyChannelString(logChannels.Channels.GuildUpdate), Inline: true},
			{Name: "Emote changes", Value: prettyChannelString(logChannels.Channels.GuildEmojisUpdate), Inline: true},
			{Name: "New roles", Value: prettyChannelString(logChannels.Channels.GuildRoleCreate), Inline: true},
			{Name: "Edited roles", Value: prettyChannelString(logChannels.Channels.GuildRoleUpdate), Inline: true},
			{Name: "Deleted roles", Value: prettyChannelString(logChannels.Channels.GuildRoleDelete), Inline: true},

			{Name: "New channels", Value: prettyChannelString(logChannels.Channels.ChannelCreate), Inline: true},
			{Name: "Edited channels", Value: prettyChannelString(logChannels.Channels.ChannelUpdate), Inline: true},
			{Name: "Deleted channels", Value: prettyChannelString(logChannels.Channels.ChannelDelete), Inline: true},
			{Name: "Member joins", Value: prettyChannelString(logChannels.Channels.GuildMemberAdd), Inline: true},
			{Name: "Member leaves", Value: prettyChannelString(logChannels.Channels.GuildMemberRemove), Inline: true},

			{Name: "Member role changes", Value: prettyChannelString(logChannels.Channels.GuildMemberUpdate), Inline: true},
			{Name: "Key role changes", Value: prettyChannelString(logChannels.Channels.GuildKeyRoleUpdate), Inline: true},
			{Name: "Member name changes", Value: prettyChannelString(logChannels.Channels.GuildMemberNickUpdate), Inline: true},
			{Name: "Avatar changes", Value: prettyChannelString(logChannels.Channels.GuildMemberAvatarUpdate), Inline: true},
			{Name: "Kicks", Value: prettyChannelString(logChannels.Channels.GuildMemberKick), Inline: true},

			{Name: "Bans", Value: prettyChannelString(logChannels.Channels.GuildBanAdd), Inline: true},
			{Name: "Unbans", Value: prettyChannelString(logChannels.Channels.GuildBanRemove), Inline: true},
			{Name: "New invites", Value: prettyChannelString(logChannels.Channels.InviteCreate), Inline: true},
			{Name: "Deleted invites", Value: prettyChannelString(logChannels.Channels.InviteDelete), Inline: true},
			{Name: "Edited messages", Value: prettyChannelString(logChannels.Channels.MessageUpdate), Inline: true},

			{Name: "Deleted messages", Value: prettyChannelString(logChannels.Channels.MessageDelete), Inline: true},
			{Name: "Bulk deleted messages", Value: prettyChannelString(logChannels.Channels.MessageDeleteBulk), Inline: true},
		},
	}

	err = ctx.ReplyComplex(api.InteractionResponseData{
		Embeds:     &[]discord.Embed{embed},
		Components: &channelButtons,
	})
	if err != nil {
		log.Errorf("sending interaction response for %v: %v", ctx.Event.ID, err)
		return err
	}

	// TODO: the rest of the fucking owl
	return nil
}
