package commands

import (
	"fmt"
	"math"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/starshine-sys/bcr"
)

func (bot *Bot) listInvites(ctx *bcr.Context) (err error) {
	is, err := bot.State.GuildInvites(ctx.Message.GuildID)
	if err != nil {
		bot.Sugar.Errorf("Error getting guild invites: %v", err)
		_, err = ctx.Sendf("Could not get this server's invites. Are you sure I have the **Manage Server** permission?")
		return
	}

	if len(is) == 0 {
		_, err = ctx.Send("This server has no invites.", nil)
		return
	}

	var (
		invites = map[string]discord.Invite{}
		names   = map[string]string{}
	)

	for _, i := range is {
		i := i
		invites[i.Code] = i
		names[i.Code] = "Unnamed"
	}

	names, err = bot.DB.GetInvites(ctx.Message.GuildID, names)
	if err != nil {
		bot.Sugar.Errorf("Error getting guild invite names: %v", err)
		_, err = ctx.Sendf("Internal error occurred.")
		return
	}

	var fields []discord.EmbedField

	for code, name := range names {
		fields = append(fields, discord.EmbedField{
			Name: code,
			Value: fmt.Sprintf(`%v
Uses: %v
Created by %v#%v`, name, invites[code].Uses, invites[code].Inviter.Username, invites[code].Inviter.Discriminator),
			Inline: true,
		})
	}

	_, err = ctx.PagedEmbed(
		FieldPaginator("Invites", bcr.ColourPurple, fields, 9), false,
	)
	return
}

// FieldPaginator paginates embed fields
func FieldPaginator(title string, colour discord.Color, fields []discord.EmbedField, perPage int) []discord.Embed {
	var (
		embeds []discord.Embed
		count  int

		pages = 1
		buf   = discord.Embed{
			Title: title,
			Color: colour,
			Footer: &discord.EmbedFooter{
				Text: fmt.Sprintf("Page 1/%v", math.Ceil(float64(float64(len(fields))/float64(perPage)))),
			},
		}
	)

	for _, field := range fields {
		if count >= perPage {
			embeds = append(embeds, buf)
			buf = discord.Embed{
				Title: title,
				Color: colour,
				Footer: &discord.EmbedFooter{
					Text: fmt.Sprintf("Page %v/%v", pages+1, math.Ceil(float64(float64(len(fields))/float64(perPage)))),
				},
			}
			count = 0
			pages++
		}
		buf.Fields = append(buf.Fields, field)
		count++
	}

	if len(buf.Fields) > 0 {
		embeds = append(embeds, buf)
	}

	return embeds
}
