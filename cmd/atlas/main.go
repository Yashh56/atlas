package main

import (
	"os"
	"path/filepath"

	"github.com/Yashh56/atlas/cmd/atlas/cmd"
	"github.com/joho/godotenv"
)

func findAndLoadEnv() {
	dir, err := os.Getwd()
	if err != nil {
		return
	}
	for {
		envPath := filepath.Join(dir, ".env")
		if _, err := os.Stat(envPath); err == nil {
			_ = godotenv.Load(envPath)
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
}

func main() {
	findAndLoadEnv()

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
