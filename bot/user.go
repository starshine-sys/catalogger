package bot

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
)

func (bot *Bot) User(id discord.UserID) (*discord.User, error) {
	bot.userMu.Lock()
	defer bot.userMu.Unlock()

	u, ok := bot.userCache[id]
	if ok {
		return &u, nil
	}

	s, _ := bot.Router.StateFromGuildID(0)
	user, err := s.User(id)
	if err != nil {
		return nil, err
	}
	bot.userCache[id] = *user

	return user, nil
}

func (bot *Bot) SetUser(u discord.User) {
	bot.userMu.Lock()
	bot.userCache[u.ID] = u
	bot.userMu.Unlock()
}

func (bot *Bot) handleEventForCache(iface interface{}) {
	switch ev := iface.(type) {
	case *gateway.MessageCreateEvent:
		bot.SetUser(ev.Author)
	case *gateway.GuildMemberAddEvent:
		bot.SetUser(ev.User)
	case *gateway.GuildMemberRemoveEvent:
		bot.SetUser(ev.User)
	case *gateway.GuildMemberUpdateEvent:
		bot.SetUser(ev.User)
	}
}
