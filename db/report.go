package db

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/v2/discord"
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
	db.Sugar.Error(err)

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
func (db *DB) ReportCtx(ctx *bcr.Context, e error) (err error) {
	var s string
	var embed *discord.Embed

	id := db.Report(ErrorContext{
		Command: ctx.Command,
		UserID:  ctx.Author.ID,
		GuildID: ctx.Message.GuildID,
	}, e)

	if id == nil {
		s = "Internal error occurred."
	} else {
		s = fmt.Sprintf("Error code: ``%v``", bcr.EscapeBackticks(string(*id)))
		embed = &discord.Embed{
			Title:       "Internal error occurred",
			Description: "An internal error has occurred. If this issue persists, please contact the bot developer with the error code above.",
			Color:       bcr.ColourRed,

			Footer: &discord.EmbedFooter{
				Text: string(*id),
			},
			Timestamp: discord.NowTimestamp(),
		}
		// oh look! spaghetti!
		if os.Getenv("SUPPORT_SERVER") != "" {
			embed.Description = strings.NewReplacer("the bot developer", fmt.Sprintf("the bot developer in the [support server](%v)", os.Getenv("SUPPORT_SERVER"))).Replace(embed.Description)
		}
	}

	_, err = ctx.Send(s, embed)
	return
}
