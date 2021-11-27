package common

import (
	"os/exec"
	"strings"

	"github.com/joho/godotenv"
)

var Version = "[unknown]"

func init() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	InitLog()

	if Version != "[unknown]" {
		return
	}

	Log.Warnf("Git version is not set, falling back to command")

	git := exec.Command("git", "rev-parse", "--short", "HEAD")
	// ignoring errors *should* be fine? if there's no output we just fall back to "unknown"
	b, _ := git.Output()
	Version = strings.TrimSpace(string(b))
	if Version == "" {
		Version = "[unknown]"
	}
}
