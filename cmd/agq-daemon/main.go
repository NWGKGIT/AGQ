package main

import (
	"log/slog"
	"os"
)

func main() {
	if err := run(); err != nil {
		slog.Error("agq-daemon stopped", "err", err)
		os.Exit(1)
	}
}

func run() error {
	slog.Info("agq-daemon starting")
	return nil
}
