package events

import (
	"fmt"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/events/handler"
)

type MemberKickEvent struct {
	ChannelID discord.ChannelID

	Entry     discord.AuditLogEntry
	User      discord.User
	Moderator *discord.User
}

func (bot *Bot) memberKick(ev *MemberKickEvent) (resp *handler.Response, err error) {
	resp = &handler.Response{
		ChannelID: ev.ChannelID,
	}

	e := discord.Embed{
		Title: "Member kicked",
		Author: &discord.EmbedAuthor{
			Icon: ev.User.AvatarURL(),
			Name: ev.User.Tag(),
		},
		Description: fmt.Sprintf("**%v** (%v, %v) was kicked", ev.User.Tag(), ev.User.ID, ev.User.Mention()),
		Timestamp:   discord.NowTimestamp(),
		Footer: &discord.EmbedFooter{
			Text: "User ID: " + ev.User.ID.String(),
		},
		Color: bcr.ColourRed,
	}

	reason := "None specified"
	if ev.Entry.Reason != "" {
		reason = ev.Entry.Reason
	}

	e.Fields = append(e.Fields, discord.EmbedField{
		Name:  "Reason",
		Value: reason,
	})

	if ev.Moderator != nil {
		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  "Responsible moderator",
			Value: fmt.Sprintf("%v (%v)", ev.Moderator.Tag(), ev.Moderator.ID),
		})
	} else {
		e.Fields = append(e.Fields, discord.EmbedField{
			Name:  "Responsible moderator",
			Value: fmt.Sprintf("%v (%v)", ev.Entry.UserID.Mention(), ev.Entry.UserID),
		})
	}

	resp.Embeds = append(resp.Embeds, e)

	return resp, nil
}
