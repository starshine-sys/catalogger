package stats

import "regexp"

// Most of this file's regexes are taken from PluralKit's REST client:
// https://github.com/xSke/PluralKit/blob/d28e99ba43cd8002c893bf29b07007cff72c0360/Myriad/Rest/BaseRestClient.cs#L274-L321

var (
	versionRegexp = regexp.MustCompile(`/api/v\d+`)

	channelsRegexp             = regexp.MustCompile(`/channels/\d+`)
	messagesRegexp             = regexp.MustCompile(`/messages/\d+`)
	membersRegexp              = regexp.MustCompile(`/members/\d+`)
	webhooksExecRegexp         = regexp.MustCompile(`/webhooks/\d+/[^/]+`)
	webhooksRegexp             = regexp.MustCompile(`/webhooks/\d+`)
	usersRegexp                = regexp.MustCompile(`/users/\d+`)
	bansRegexp                 = regexp.MustCompile(`/bans/\d+`)
	rolesRegexp                = regexp.MustCompile(`/roles/\d+`)
	pinsRegexp                 = regexp.MustCompile(`/pins/\d+`)
	emojisRegexp               = regexp.MustCompile(`/emojis/\d+`)
	guildsRegexp               = regexp.MustCompile(`/guilds/\d+`)
	integrationsRegexp         = regexp.MustCompile(`/integrations/\d+`)
	permissionsRegexp          = regexp.MustCompile(`/permissions/\d+`)
	userReactionsRegexp        = regexp.MustCompile(`/reactions/[^{/]+/\d+`)
	reactionsRegexp            = regexp.MustCompile(`/reactions/[^{/]+`)
	invitesRegexp              = regexp.MustCompile(`/invites/[^{/]+`)
	interactionsResponseRegexp = regexp.MustCompile(`/interactions/\d+/[^{/]+`)
	interactionsRegexp         = regexp.MustCompile(`/interactions/\d+`)

	snowflakeRegexp = regexp.MustCompile(`\d{15,}`)
)

var (
	webhookToken     = regexp.MustCompile(`/webhooks/(\d+)/[^/]+`)
	interactionToken = regexp.MustCompile(`/interactions/(\d+)/[^{/]+`)
)

func NormalizePath(path string) string {
	path = channelsRegexp.ReplaceAllLiteralString(path, "/channels/{channel_id}")
	path = messagesRegexp.ReplaceAllLiteralString(path, "/messages/{message_id}")
	path = membersRegexp.ReplaceAllLiteralString(path, "/members/{user_id}")
	path = webhooksExecRegexp.ReplaceAllLiteralString(path, "/webhooks/{webhook_id}/{webhook_token}")
	path = webhooksRegexp.ReplaceAllLiteralString(path, "/webhooks/{webhook_id}")
	path = usersRegexp.ReplaceAllLiteralString(path, "/users/{user_id}")
	path = bansRegexp.ReplaceAllLiteralString(path, "/bans/{user_id}")
	path = rolesRegexp.ReplaceAllLiteralString(path, "/roles/{role_id}")
	path = pinsRegexp.ReplaceAllLiteralString(path, "/pins/{message_id}")
	path = emojisRegexp.ReplaceAllLiteralString(path, "/emojis/{emoji_id}")
	path = guildsRegexp.ReplaceAllLiteralString(path, "/guilds/{guild_id}")
	path = integrationsRegexp.ReplaceAllLiteralString(path, "/integrations/{integration_id}")
	path = permissionsRegexp.ReplaceAllLiteralString(path, "/permissions/{overwrite_id}")
	path = userReactionsRegexp.ReplaceAllLiteralString(path, "/reactions/{emoji}/{user_id}")
	path = reactionsRegexp.ReplaceAllLiteralString(path, "/reactions/{emoji}")
	path = invitesRegexp.ReplaceAllLiteralString(path, "/invites/{invite_code}")
	path = interactionsResponseRegexp.ReplaceAllLiteralString(path, "/interactions/{interaction_id}/{interaction_token}")
	path = interactionsRegexp.ReplaceAllLiteralString(path, "/interactions/{interaction_id}")

	path = snowflakeRegexp.ReplaceAllLiteralString(path, "{snowflake}")

	return path
}

func EndpointMetricsName(method, path string) string {
	path = versionRegexp.ReplaceAllLiteralString(path, "")

	return method + " " + NormalizePath(path)
}

func LoggingName(path string) string {
	path = webhookToken.ReplaceAllString(path, "/webhooks/$1/:token")
	path = interactionToken.ReplaceAllString(path, "/interactions/$1/:token")

	return path
}
