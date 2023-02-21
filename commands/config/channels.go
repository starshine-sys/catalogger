package config

import (
	"context"
	"fmt"
	"strings"
	"time"

	"emperror.dev/errors"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
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
			Label:    "Members joining",
			CustomID: "channel:GUILD_MEMBER_ADD",
			Style:    discord.PrimaryButtonStyle(),
		},
		&discord.ButtonComponent{
			Label:    "Members leaving",
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
		return bot.ReportError(ctx, errors.Wrap(err, "getting log channels"))
	}
	guildChannels, err := bot.Cabinet.Channels(context.Background(), ctx.Event.GuildID)
	if err != nil {
		log.Errorf("getting guild channels for guild %v: %v", ctx.Event.GuildID, err)
		return bot.ReportError(ctx, errors.Wrap(err, "getting guild channels"))
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

	embed := func() discord.Embed {
		return discord.Embed{
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
				{Name: "Members joining", Value: prettyChannelString(logChannels.Channels.GuildMemberAdd), Inline: true},
				{Name: "Members leaving", Value: prettyChannelString(logChannels.Channels.GuildMemberRemove), Inline: true},

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
	}

	err = ctx.ReplyComplex(api.InteractionResponseData{
		Embeds:     &[]discord.Embed{embed()},
		Components: &channelButtons,
	})
	if err != nil {
		log.Errorf("sending interaction response for %v: %v", ctx.Event.ID, err)
		return bot.ReportError(ctx, errors.Wrap(err, "sending initial embed"))
	}

	// TODO: the rest of the fucking owl
	msg, err := ctx.Original()
	if err != nil {
		log.Errorf("getting original message for %v: %v", ctx.Event.ID, err)
		return bot.ReportError(ctx, errors.Wrap(err, "getting original message"))
	}

	clearComponents := func() error {
		_, err = ctx.State.EditInteractionResponse(discord.AppID(bot.Me().ID), ctx.InteractionToken, api.EditInteractionResponseData{
			Components: &discord.ContainerComponents{},
		})
		return err
	}

	// timeout after 10 minutes
	cctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	for {
		select {
		// if timeout, clear components and return
		case <-cctx.Done():
			err = clearComponents()
			if err != nil {
				log.Errorf("updating message for %v: %v", ctx.Event.ID, err)
				return bot.ReportError(ctx, errors.Wrap(err, "updating message"))
			}
			return

		default:
			// wait for a button event
			ev, ok := common.WaitFor(cctx, ctx.State, func(ev *gateway.InteractionCreateEvent) bool {
				if ev.Message == nil || ev.Message.ID != msg.ID {
					return false
				}

				data, ok := ev.Data.(*discord.ButtonInteraction)
				return ok && strings.HasPrefix(string(data.CustomID), "channel:")
			})
			if !ok {
				continue
			}

			data, ok := ev.Data.(*discord.ButtonInteraction)
			if !ok {
				continue
			}

			if data.CustomID == "channel:close" {
				err = clearComponents()
				if err != nil {
					log.Errorf("updating message for %v: %v", ctx.Event.ID, err)
					return bot.ReportError(ctx, errors.Wrap(err, "updating message"))
				}
				return
			}

			bctx, err := bot.Router.NewButtonContext(ev)
			if err != nil {
				log.Errorf("could not create button context for interaction %v: %v", ev.ID, err)
				continue
			}

			var hctx bcr.HasContext
			// yes, this has to be done manually, no easy tricks to reduce code here
			switch bctx.CustomID {
			case "channel:GUILD_UPDATE":
				hctx = bot.channelPage(bctx, "Server changes", &logChannels.Channels.GuildUpdate, prettyChannelString)
			case "channel:GUILD_EMOJIS_UPDATE":
				hctx = bot.channelPage(bctx, "Emote changes", &logChannels.Channels.GuildEmojisUpdate, prettyChannelString)
			case "channel:GUILD_ROLE_CREATE":
				hctx = bot.channelPage(bctx, "New roles", &logChannels.Channels.GuildRoleCreate, prettyChannelString)
			case "channel:GUILD_ROLE_UPDATE":
				hctx = bot.channelPage(bctx, "Edited roles", &logChannels.Channels.GuildRoleUpdate, prettyChannelString)
			case "channel:GUILD_ROLE_DELETE":
				hctx = bot.channelPage(bctx, "Deleted roles", &logChannels.Channels.GuildRoleDelete, prettyChannelString)
			case "channel:CHANNEL_CREATE":
				hctx = bot.channelPage(bctx, "New channels", &logChannels.Channels.ChannelCreate, prettyChannelString)
			case "channel:CHANNEL_UPDATE":
				hctx = bot.channelPage(bctx, "Edited channels", &logChannels.Channels.ChannelUpdate, prettyChannelString)
			case "channel:CHANNEL_DELETE":
				hctx = bot.channelPage(bctx, "Deleted channels", &logChannels.Channels.ChannelDelete, prettyChannelString)
			case "channel:GUILD_MEMBER_ADD":
				hctx = bot.channelPage(bctx, "Members joining", &logChannels.Channels.GuildMemberAdd, prettyChannelString)
			case "channel:GUILD_MEMBER_REMOVE":
				hctx = bot.channelPage(bctx, "Members leaving", &logChannels.Channels.GuildMemberRemove, prettyChannelString)
			case "channel:GUILD_MEMBER_UPDATE":
				hctx = bot.channelPage(bctx, "Member role changes", &logChannels.Channels.GuildMemberUpdate, prettyChannelString)
			case "channel:GUILD_KEY_ROLE_UPDATE":
				hctx = bot.channelPage(bctx, "Key role changes", &logChannels.Channels.GuildKeyRoleUpdate, prettyChannelString)
			case "channel:GUILD_MEMBER_NICK_UPDATE":
				hctx = bot.channelPage(bctx, "Member name changes", &logChannels.Channels.GuildMemberNickUpdate, prettyChannelString)
			case "channel:GUILD_MEMBER_AVATAR_UPDATE":
				hctx = bot.channelPage(bctx, "Avatar changes", &logChannels.Channels.GuildMemberAvatarUpdate, prettyChannelString)
			case "channel:GUILD_MEMBER_KICK":
				hctx = bot.channelPage(bctx, "Kicks", &logChannels.Channels.GuildMemberKick, prettyChannelString)
			case "channel:GUILD_BAN_ADD":
				hctx = bot.channelPage(bctx, "Bans", &logChannels.Channels.GuildBanAdd, prettyChannelString)
			case "channel:GUILD_BAN_REMOVE":
				hctx = bot.channelPage(bctx, "Unbans", &logChannels.Channels.GuildBanRemove, prettyChannelString)
			case "channel:INVITE_CREATE":
				hctx = bot.channelPage(bctx, "New invites", &logChannels.Channels.InviteCreate, prettyChannelString)
			case "channel:INVITE_DELETE":
				hctx = bot.channelPage(bctx, "Deleted invites", &logChannels.Channels.InviteDelete, prettyChannelString)
			case "channel:MESSAGE_UPDATE":
				hctx = bot.channelPage(bctx, "Edited messages", &logChannels.Channels.MessageUpdate, prettyChannelString)
			case "channel:MESSAGE_DELETE":
				hctx = bot.channelPage(bctx, "Deleted messages", &logChannels.Channels.MessageDelete, prettyChannelString)
			case "channel:MESSAGE_DELETE_BULK":
				hctx = bot.channelPage(bctx, "Bulk deleted messages", &logChannels.Channels.MessageDeleteBulk, prettyChannelString)
			}

			// the previous function *probably* updated something, but it's easier to just *always* update the db
			err = bot.DB.SetChannels(ev.GuildID, logChannels)
			if err != nil {
				log.Errorf("setting channels in guild %v: %v", ev.GuildID, err)
				return bot.ReportError(hctx, err)
			}

			nctx := hctx.Ctx()
			err = nctx.State.RespondInteraction(nctx.InteractionID, nctx.InteractionToken, api.InteractionResponse{
				Type: api.UpdateMessage,
				Data: &api.InteractionResponseData{
					Embeds:     &[]discord.Embed{embed()},
					Components: &channelButtons,
				},
			})
			if err != nil {
				log.Errorf("updating message for interaction %v: %v", nctx.InteractionID, err)
				return nil
			}
		}
	}
}

func (bot *Bot) channelPage(ctx *bcr.ButtonContext, subject string, channelID *discord.ChannelID, toString func(id discord.ChannelID) string) (returnCtx bcr.HasContext) {
	var desc string
	if !channelID.IsValid() {
		desc = "This event is not currently logged.\nTo start logging it somewhere, select a channel below."
	} else {
		desc = fmt.Sprintf(`This event is currently set to log to %v.
To change where it is logged, select a channel below.
To disable logging this event entirely, select "Stop logging" below.`, toString(*channelID))
	}

	err := ctx.State.RespondInteraction(ctx.InteractionID, ctx.InteractionToken, api.InteractionResponse{
		Type: api.UpdateMessage,
		Data: &api.InteractionResponseData{
			Embeds: &[]discord.Embed{{
				Title:       subject,
				Description: desc,
				Color:       common.ColourPurple,
			}},
			Components: &discord.ContainerComponents{
				&discord.ActionRowComponent{
					&discord.ChannelSelectComponent{
						CustomID:     "channel:select",
						ChannelTypes: []discord.ChannelType{discord.GuildText},
					},
				},
				&discord.ActionRowComponent{
					&discord.ButtonComponent{
						CustomID: "channel:reset",
						Label:    "Stop logging",
						Style:    discord.DangerButtonStyle(),
					},
					&discord.ButtonComponent{
						CustomID: "channel:cancel",
						Label:    "Return to menu",
						Style:    discord.SecondaryButtonStyle(),
					},
				},
			},
		},
	})
	if err != nil {
		log.Errorf("sending reply for %v: %v", ctx.InteractionID, err)
		return ctx
	}

	msg, err := ctx.Original()
	if err != nil {
		log.Errorf("getting original message for %v: %v", ctx.InteractionID, err)
		return ctx
	}

	cctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	ev, ok := common.WaitFor(cctx, ctx.State, func(ev *gateway.InteractionCreateEvent) bool {
		if ev.Message == nil || ev.Message.ID != msg.ID {
			return false
		}

		if _, ok := ev.Data.(*discord.ChannelSelectInteraction); !ok {
			if _, ok := ev.Data.(*discord.ButtonInteraction); !ok {
				return false
			}
		}

		return true
	})
	if !ok {
		return ctx
	}

	returnCtx, err = bot.Router.NewRootContext(ev)
	if err != nil {
		log.Errorf("creating context for interaction %v: %v", ev.ID, err)
		return ctx
	}

	switch data := ev.Data.(type) {
	case *discord.ChannelSelectInteraction:
		if len(data.Values) == 0 {
			return
		}
		*channelID = data.Values[0]
	case *discord.ButtonInteraction:
		if data.CustomID == "channel:reset" {
			*channelID = discord.NullChannelID
		}
	}

	return
}
