package meta

import (
	"net/url"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/starshine-sys/bcr/v2"
	"github.com/starshine-sys/catalogger/v2/common"
	"github.com/starshine-sys/catalogger/v2/common/log"
)

func (bot *Bot) help(ctx *bcr.CommandContext) (err error) {
	embed := discord.Embed{
		Title:       "Catalogger help",
		Description: "Catalogger is a logging bot that integrates with PluralKit's message proxying.",
		Color:       common.ColourPurple,
		Fields: []discord.EmbedField{
			// TODO: the rest of the fucking help embed
			{
				Name:  "Developer",
				Value: "<@694563574386786314> / starshine#0001",
			},
			{
				Name:  "Source code",
				Value: "https://github.com/starshine-sys/catalogger",
			},
		},
	}

	// support server invite
	if bot.Config.Info.SupportServer != "" {
		embed.Fields = append(embed.Fields, discord.EmbedField{
			Name:  "Support server",
			Value: "Use this link to join the support server: " + bot.Config.Info.SupportServer,
		})
	}

	// extra help fields defined in configuration
	if len(bot.Config.Info.HelpFields) > 0 {
		embed.Fields = append(embed.Fields, bot.Config.Info.HelpFields...)
	}

	// dashboard links (including tos and privacy policy)
	components := discord.ActionRowComponent{}
	if bot.Config.Info.DashboardBase != "" {
		tosPath, err := url.JoinPath(bot.Config.Info.DashboardBase, "/page/tos")
		if err != nil {
			log.Errorf("joining tos path: %v", err)
			return bot.ReportError(ctx, err)
		}
		privacyPath, err := url.JoinPath(bot.Config.Info.DashboardBase, "/page/privacy")
		if err != nil {
			log.Errorf("joining privacy path: %v", err)
			return bot.ReportError(ctx, err)
		}

		components = append(components,
			&discord.ButtonComponent{
				Label: "Dashboard",
				Style: discord.LinkButtonStyle(bot.Config.Info.DashboardBase),
			},
			&discord.ButtonComponent{
				Label: "Privacy policy",
				Style: discord.LinkButtonStyle(tosPath),
			},
			&discord.ButtonComponent{
				Label: "Terms of service",
				Style: discord.LinkButtonStyle(privacyPath),
			})
	}

	return ctx.ReplyComplex(api.InteractionResponseData{
		Embeds:     &[]discord.Embed{embed},
		Components: &discord.ContainerComponents{&components},
		Flags:      discord.EphemeralMessage,
	})
}
