package common

import "github.com/diamondburned/arikawa/v3/discord"

// Perm is a single permission
type Perm struct {
	Permission discord.Permissions
	Name       string
}

// Permission constants that Arikawa is missing
const (
	PermissionViewServerInsights = 1 << 19
	PermissionModerateMembers    = 1 << 40
)

// All permissions
var (
	MajorPerms = []Perm{
		{discord.PermissionAdministrator, "Administrator"},
		{discord.PermissionManageGuild, "Manage Server"},
		{discord.PermissionManageWebhooks, "Manage Webhooks"},
		{discord.PermissionManageChannels, "Manage Channels"},

		{discord.PermissionBanMembers, "Ban Members"},
		{discord.PermissionKickMembers, "Kick Members"},

		{discord.PermissionManageRoles, "Manage Roles"},
		{discord.PermissionManageNicknames, "Manage Nicknames"},
		{discord.PermissionManageEmojisAndStickers, "Manage Emojis and Stickers"},
		{discord.PermissionManageThreads, "Manage Threads"},
		{discord.PermissionManageMessages, "Manage Messages"},
		{PermissionModerateMembers, "Moderate Members"},

		{discord.PermissionMentionEveryone, "Mention Everyone"},

		{discord.PermissionModerateMembers, "Timeout Members"},
		{discord.PermissionMuteMembers, "Voice Mute Members"},
		{discord.PermissionDeafenMembers, "Voice Deafen Members"},
		{discord.PermissionMoveMembers, "Voice Move Members"},
	}

	NotablePerms = []Perm{
		{discord.PermissionViewAuditLog, "View Audit Log"},
		{PermissionViewServerInsights, "View Server Insights"},

		{discord.PermissionPrioritySpeaker, "Priority Speaker"},
		{discord.PermissionSendTTSMessages, "Send TTS Messages"},

		{discord.PermissionCreateInstantInvite, "Create Invite"},
		{discord.PermissionCreatePublicThreads, "Create Public Threads"},
		{discord.PermissionCreatePrivateThreads, "Create Private Threads"},
	}

	MinorPerms = []Perm{
		{discord.PermissionStream, "Video"},
		{discord.PermissionUseVAD, "Use Voice Activity"},
		{discord.PermissionSpeak, "Speak"},
		{discord.PermissionConnect, "Connect"},
		{discord.PermissionRequestToSpeak, "Request to Speak"},

		{discord.PermissionAttachFiles, "Attach Files"},
		{discord.PermissionEmbedLinks, "Embed Links"},
		{discord.PermissionStartEmbeddedActivities, "Start Embedded Activities"},

		{discord.PermissionAddReactions, "Add Reactions"},
		{discord.PermissionSendMessages, "Send Messages"},
		{discord.PermissionSendMessagesInThreads, "Send Messages in Threads"},

		{discord.PermissionReadMessageHistory, "Read Message History"},
		{discord.PermissionViewChannel, "View Channel"},

		{discord.PermissionUseSlashCommands, "Use Slash Commands"},

		{discord.PermissionChangeNickname, "Change Nickname"},
		{discord.PermissionUseExternalEmojis, "Use External Emojis"},
	}

	AllPerms = append(MajorPerms, append(NotablePerms, MinorPerms...)...)
)

// PermStrings gives permission strings for all Discord permissions
func PermStrings(p discord.Permissions) []string {
	return PermStringsFor(AllPerms, p)
}

// PermStringsFor gives permission strings for the given Perm slice
func PermStringsFor(m []Perm, p discord.Permissions) []string {
	var out = make([]string, 0, 32)
	for _, perm := range m {
		if p&perm.Permission == perm.Permission {
			out = append(out, perm.Name)
		}
	}

	return out
}
