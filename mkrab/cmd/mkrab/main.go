package main

import (
	"log/slog"
	"os"

	"mkrab/internal/app"
)

func main() {
	if err := app.Execute(); err != nil {
		slog.Error("broker stopped", "error", err)
		os.Exit(1)
	}
}
