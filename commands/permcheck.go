package commands

import (
	"fmt"
	"strings"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/starshine-sys/bcr"
)

func (bot *Bot) permcheck(ctx *bcr.Context) (err error) {
	ch, err := bot.DB.Channels(ctx.Message.GuildID)
	if err != nil {
		_, err = ctx.Sendf("Error getting channels: %v\nPlease contact the developer.", err)
		return
	}

	var toCheck []discord.ChannelID
	for _, v := range ch {
		if v.IsValid() {
			toCheck = append(toCheck, v)
		}
	}

	fmt.Println(toCheck)

	missingPerms := map[string][]string{}

	e := discord.Embed{
		Title: "Permission check",
		Color: bcr.ColourPurple,
	}

	for _, ch := range toCheck {
		p, err := ctx.State.Permissions(ch, ctx.Bot.ID)
		if err != nil {
			_, err = ctx.Sendf("There was an error checking permissions for %v (ID: %v).", ch.Mention(), ch)
			return err
		}

		if !p.Has(discord.PermissionManageWebhooks) {
			missingPerms["webhooks"] = append(missingPerms["webhooks"], ch.Mention())
		}
		if !p.Has(discord.PermissionViewChannel) {
			missingPerms["view"] = append(missingPerms["view"], ch.Mention())
		}
	}

	if len(missingPerms["webhooks"]) == 0 && len(missingPerms["view"]) == 0 {
		e.Description = fmt.Sprintf(`No issues found! :)
If the bot still isn't logging, try `+"`%vclearcache`"+`.
If that doesn't work, contact the developer.`, ctx.Prefix)
		e.Color = bcr.ColourGreen
	} else {
		e.Color = bcr.ColourRed

		if len(missingPerms["webhooks"]) > 0 {
			e.Fields = append(e.Fields, discord.EmbedField{
				Name:  "Missing \"Manage Webhooks\"",
				Value: strings.Join(missingPerms["webhooks"], ", "),
			})
		}

		if len(missingPerms["view"]) > 0 {
			e.Fields = append(e.Fields, discord.EmbedField{
				Name:  "Missing \"View Channel\"",
				Value: strings.Join(missingPerms["view"], ", "),
			})
		}
	}

	_, err = ctx.Send("", &e)
	return
}
