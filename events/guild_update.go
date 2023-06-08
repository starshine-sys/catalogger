package events

import (
	"fmt"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/common"
	"github.com/starshine-sys/catalogger/events/handler"
)

func (bot *Bot) guildUpdate(ev *gateway.GuildUpdateEvent) (resp *handler.Response, err error) {
	old, ok := bot.Guilds.Get(ev.ID)
	if !ok {
		common.Log.Errorf("Error getting info for guild %v", ev.ID)
		return
	}
	defer bot.Guilds.Set(ev.ID, ev.Guild)

	ch, err := bot.DB.Channels(ev.ID)
	if err != nil {
		return
	}

	if !ch[keys.GuildUpdate].IsValid() {
		return
	}

	resp = &handler.Response{
		ChannelID: ch[keys.GuildUpdate],
		GuildID:   ev.ID,
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
		newOwner, err := bot.User(ev.OwnerID)
		if err != nil {
			return nil, err
		}
		oldOwner, err := bot.User(old.OwnerID)
		if err != nil {
			return nil, err
		}

		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  "Ownership transferred",
			Value: fmt.Sprintf("**Before:** %v (%v)\n**After:** %v (%v)", oldOwner.Tag(), oldOwner.ID, newOwner.Tag(), newOwner.ID),
		})
		changed = true
	}

	if !changed {
		return
	}

	resp.Embeds = append(resp.Embeds, e)
	return
}
