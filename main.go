package main

import (
	"os"

	"github.com/allskar/llmux/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
