package main

import (
	"os"

	"github.com/Yashh56/atlas/cmd/atlas/cmd"
	"github.com/joho/godotenv"
)

func main() {
	// Best-effort load from .env in the current working directory.
	// We ignore the error because .env is optional.
	_ = godotenv.Load()

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
