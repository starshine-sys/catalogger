package main

import (
	"fmt"
	"os"

	"github.com/starshine-sys/catalogger/v2/cmd"
)

func main() {
	if err := cmd.Run(); err != nil {
		fmt.Println("error in command:", err)
		os.Exit(1)
	}
}
