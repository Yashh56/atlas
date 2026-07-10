package main

import (
	"os"

	"github.com/Yashh56/atlas/cmd/atlas/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
