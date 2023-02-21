package bot

import (
	"fmt"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/starshine-sys/bcr/v2"
	"github.com/starshine-sys/catalogger/v2/common"
)

func (bot *Bot) ReportError(c bcr.HasContext, err error) error {
	ctx := c.Ctx()

	if bot.Config.Auth.Sentry == "" {
		embed := discord.Embed{
			Title: "Internal error occurred",
			Description: fmt.Sprintf("An internal error has occurred. "+
				"If this issue persists, please contact the developer "+
				"in the [support server](%v).", bot.Config.Info.SupportServer),
			Color:     common.ColourRed,
			Timestamp: discord.NowTimestamp(),
		}

		mErr := ctx.Reply("", embed)
		return mErr
	}

	hub := sentry.CurrentHub().Clone()
	hub.ConfigureScope(func(scope *sentry.Scope) {
		if ctx.User.ID.IsValid() {
			scope.SetUser(sentry.User{ID: ctx.User.ID.String()})
		}
	})

	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Data: map[string]any{
			"user": ctx.User.ID,
		},
		Level:     sentry.LevelError,
		Timestamp: time.Now().UTC(),
	}, nil)

	id := hub.CaptureException(err)
	if id == nil {
		uid := uuid.New().String()
		id = (*sentry.EventID)(&uid)
	}

	mErr := ctx.ReplyEphemeral(fmt.Sprintf("Error code: ``%v``", string(*id)),
		discord.Embed{
			Title: "Internal error occurred",
			Description: fmt.Sprintf("An internal error has occurred. "+
				"If this issue persists, please contact the developer "+
				"in the [support server](%v) with the error code above.", bot.Config.Info.SupportServer),
			Color:     common.ColourRed,
			Timestamp: discord.NowTimestamp(),
			Footer: &discord.EmbedFooter{
				Text: string(*id),
			},
		})
	return mErr
}
