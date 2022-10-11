package main

import (
	"fmt"
	"os"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/starshine-sys/catalogger/v2/common"
)

func main() {
	if len(os.Args) == 0 {
		help()
		return
	}

	switch os.Args[len(os.Args)-1] {
	case "global":
		globalCommands()
		return

	default:
		if len(os.Args) < 3 {
			help()
			return
		}

		if os.Args[len(os.Args)-2] == "guild" {
			guildCommands(os.Args[len(os.Args)-1])
			return
		}

		help()
	}
}

func globalCommands() {
	appID, err := discord.ParseSnowflake(os.Getenv("APP_ID"))
	if err != nil {
		fmt.Println("Error parsing $APP_ID:", err)
		return
	}

	c := api.NewClient("Bot " + os.Getenv("TOKEN"))

	_, err = c.BulkOverwriteCommands(discord.AppID(appID), common.Commands)
	if err != nil {
		fmt.Println("Error overwriting commands:", err)
		return
	}

	fmt.Println("Wrote global commands!")
}

func guildCommands(id string) {
	guildID, err := discord.ParseSnowflake(id)
	if err != nil {
		fmt.Println("Error parsing guild id:", err)
		return
	}

	appID, err := discord.ParseSnowflake(os.Getenv("APP_ID"))
	if err != nil {
		fmt.Println("Error parsing $APP_ID:", err)
		return
	}

	c := api.NewClient("Bot " + os.Getenv("TOKEN"))

	_, err = c.BulkOverwriteGuildCommands(discord.AppID(appID), discord.GuildID(guildID), common.Commands)
	if err != nil {
		fmt.Println("Error overwriting commands:", err)
		return
	}

	fmt.Printf("Wrote guild commands in %v!\n", guildID)
}

func help() {
	fmt.Println("Update commands on Discord\n\nUsage:")
	fmt.Println("$TOKEN and $APP_ID environment variables must be set")
	fmt.Println("- global: update commands globally")
	fmt.Println("- guild <id>: update commands on a given guild ID")
}
