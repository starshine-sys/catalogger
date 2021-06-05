package events

import (
	"fmt"

	"github.com/diamondburned/arikawa/v2/api/webhook"
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/starshine-sys/bcr"
)

func (bot *Bot) guildUpdate(ev *gateway.GuildUpdateEvent) {
	bot.GuildsMu.Lock()
	old, ok := bot.Guilds[ev.ID]
	if !ok {
		bot.Guilds[ev.ID] = ev.Guild
		bot.GuildsMu.Unlock()
		bot.Sugar.Errorf("Error getting info for guild %v", ev.ID)
		return
	}
	bot.Guilds[ev.ID] = ev.Guild
	bot.GuildsMu.Unlock()

	ch, err := bot.DB.Channels(ev.ID)
	if err != nil {
		bot.Sugar.Errorf("Error getting server channels: %v", err)
		return
	}
	if !ch["GUILD_UPDATE"].IsValid() {
		return
	}

	wh, err := bot.webhookCache("guild_update", ev.ID, ch["GUILD_UPDATE"])
	if err != nil {
		bot.Sugar.Errorf("Error getting webhook: %v", err)
		return
	}

	var changed bool

	e := discord.Embed{
		Title: "Server updated",
		Color: bcr.ColourBlue,

		Footer: &discord.EmbedFooter{
			Text: "ID: " + ev.ID.String(),
		},
		Timestamp: discord.NowTimestamp(),
	}

	if ev.Name != old.Name {
		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  "Name",
			Value: fmt.Sprintf("**Before:** %v\n**After:** %v", old.Name, ev.Name),
		})
		changed = true
	}

	if ev.Icon != old.Icon {
		e.Title = "Server icon updated"
		e.Image = &discord.EmbedImage{URL: ev.IconURL() + "?size=1024"}
		changed = true
	}

	if ev.OwnerID != old.OwnerID {
		newOwner, err := bot.State.User(ev.OwnerID)
		if err != nil {
			bot.Sugar.Errorf("Error getting new owner: %v", err)
			return
		}
		oldOwner, err := bot.State.User(old.OwnerID)
		if err != nil {
			bot.Sugar.Errorf("Error getting old owner: %v", err)
			return
		}

		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  "Ownership transferred",
			Value: fmt.Sprintf("**Before:** %v#%v (%v)\n**After:** %v#%v (%v)", oldOwner.Username, oldOwner.Discriminator, oldOwner.ID, newOwner.Username, newOwner.Discriminator, newOwner.ID),
		})
		changed = true
	}

	if !changed {
		return
	}

	webhook.New(wh.ID, wh.Token).Execute(webhook.ExecuteData{
		AvatarURL: bot.Router.Bot.AvatarURL(),
		Embeds:    []discord.Embed{e},
	})
}