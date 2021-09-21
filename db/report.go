package db

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/getsentry/sentry-go"
	"github.com/starshine-sys/bcr"
)

// ErrorContext is the context for an error
type ErrorContext struct {
	Event   string
	Command string

	UserID  discord.UserID
	GuildID discord.GuildID
}

// Report reports an error.
func (db *DB) Report(ctx ErrorContext, err error) *sentry.EventID {
	cs := ctx.Event
	if cs == "" {
		cs = ctx.Command
	}
	db.Sugar.Errorf("Error in %v: %v", err)

	if db.Hub == nil {
		return nil
	}

	hub := db.Hub.Clone()

	data := map[string]interface{}{}

	if ctx.Event != "" {
		data["event"] = ctx.Event
	}

	if ctx.Command != "" {
		data["command"] = ctx.Command
	}

	if ctx.GuildID.IsValid() {
		data["guild"] = ctx.GuildID
	}

	hub.ConfigureScope(func(scope *sentry.Scope) {
		if ctx.UserID.IsValid() {
			scope.SetUser(sentry.User{ID: ctx.UserID.String()})
			data["user"] = ctx.UserID
		}
	})

	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Data:      data,
		Level:     sentry.LevelError,
		Timestamp: time.Now().UTC(),
	}, nil)

	return hub.CaptureException(err)
}

// ReportCtx reports an error and sends the event ID to the context channel, if possible
func (db *DB) ReportCtx(ctx bcr.Contexter, e error) (err error) {
	var guildID discord.GuildID
	if ctx.GetGuild() != nil {
		guildID = ctx.GetGuild().ID
	}

	cmdName := ""
	if v, ok := ctx.(*bcr.Context); ok {
		cmdName = strings.Join(v.FullCommandPath, " ")
	} else if v, ok := ctx.(*bcr.SlashContext); ok {
		cmdName = v.CommandName
	}

	id := db.Report(ErrorContext{
		Command: cmdName,
		UserID:  ctx.User().ID,
		GuildID: guildID,
	}, e)

	return db.ReportEmbed(ctx, id)
}

// ReportEmbed ...
func (db *DB) ReportEmbed(ctx bcr.Contexter, id *sentry.EventID) (err error) {
	var s string
	var embeds []discord.Embed

	if id == nil {
		s = "Internal error occurred."
	} else {
		s = fmt.Sprintf("Error code: ``%v``", bcr.EscapeBackticks(string(*id)))
		embeds = append(embeds, discord.Embed{
			Title:       "Internal error occurred",
			Description: "An internal error has occurred. If this issue persists, please contact the bot developer with the error code above.",
			Color:       bcr.ColourRed,

			Footer: &discord.EmbedFooter{
				Text: string(*id),
			},
			Timestamp: discord.NowTimestamp(),
		})
		// oh look! spaghetti!
		if os.Getenv("SUPPORT_SERVER") != "" {
			embeds[0].Description = strings.NewReplacer("the bot developer", fmt.Sprintf("the bot developer in the [support server](%v)", os.Getenv("SUPPORT_SERVER"))).Replace(embeds[0].Description)
		}
	}

	return ctx.SendEphemeral(s, embeds...)
}
