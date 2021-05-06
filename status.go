package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/arikawa/v2/state"
)

func statusLoop(s *state.State) {
	time.Sleep(5 * time.Second)
	for {
		status := fmt.Sprintf("%vhelp", strings.Split(os.Getenv("PREFIXES"), ",")[0])

		guilds, err := s.Guilds()
		if err == nil {
			status += fmt.Sprintf(" | in %v servers", len(guilds))
		}

		s.Gateway.UpdateStatus(gateway.UpdateStatusData{
			Activities: []discord.Activity{{
				Name: status,
			}},
		})

		time.Sleep(5 * time.Minute)
	}
}
