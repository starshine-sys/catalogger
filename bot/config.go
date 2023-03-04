package bot

import (
	"os"

	"emperror.dev/errors"
	"github.com/BurntSushi/toml"
	"github.com/diamondburned/arikawa/v3/discord"
)

type Config struct {
	Auth      AuthConfig      `toml:"auth"`
	Bot       BotConfig       `toml:"bot"`
	Dashboard DashboardConfig `toml:"dashboard"`
	Info      InfoConfig      `toml:"info"`
}

type AuthConfig struct {
	Discord  string `toml:"discord"`
	Postgres string `toml:"postgres"`
	Redis    string `toml:"redis"`
	Sentry   string `toml:"sentry"`

	Influx AuthInfluxConfig `toml:"influx"`
}

type AuthInfluxConfig struct {
	URL          string `toml:"url"`
	Token        string `toml:"token"`
	Organization string `toml:"organization"`
	Database     string `toml:"database"`
}

type BotConfig struct {
	Owner           discord.UserID    `toml:"owner"`
	AESKey          string            `toml:"aes_key"`
	CommandsGuildID discord.GuildID   `toml:"commands_guild_id"`
	NoSyncCommands  bool              `toml:"no_sync_commands"`
	JoinLeaveLog    discord.ChannelID `toml:"join_leave_log"`
	// Ready event logs
	MetaLog discord.ChannelID `toml:"meta_log"`

	// TestMode disables all interaction with Discord that is not necessary for building a cache.
	// No logging or command responses are done in this mode, invites and members are still fetched.
	TestMode bool `toml:"test_mode"`

	// NoAutoMigrate specifies if migrations should be done automatically when the bot starts.
	// If this is set to true, migrations must be done manually by running the `./catalogger migrate` command.
	NoAutoMigrate bool `toml:"no_auto_migrate"`
}

type DashboardConfig struct {
	ClientID     string `toml:"client_id"`
	ClientSecret string `toml:"client_secret"`
	HTTPS        bool   `toml:"https"`
	Port         string `toml:"port"`
	Host         string `toml:"host"`

	AnnouncementChannel discord.ChannelID `toml:"announcement_channel"`
}

type InfoConfig struct {
	SupportServer string `toml:"support_server"`
	DashboardBase string `toml:"dashboard_base"`

	HelpFields []discord.EmbedField `toml:"help_fields"`
}

// ShouldLog returns true if test mode is not enabled.
func (bot *Bot) ShouldLog() bool {
	return !bot.Config.Bot.TestMode
}

func ReadConfig(path string) (c Config, err error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return c, errors.Wrap(err, "read config file")
	}

	err = toml.Unmarshal(b, &c)
	if err != nil {
		return c, errors.Wrap(err, "unmarshal config")
	}
	return c, nil
}
