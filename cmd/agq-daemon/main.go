package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"agq-daemon/internal/api"
	"agq-daemon/internal/detector"
	"agq-daemon/internal/domain"
	"agq-daemon/internal/languageserver"
	"agq-daemon/internal/poller"
	"agq-daemon/internal/state"
	"agq-daemon/internal/store"
)

func main() {
	if err := run(); err != nil {
		slog.Error("agq-daemon stopped", "err", err)
		os.Exit(1)
	}
}

func run() error {
	agqDir, err := ensureAGQDir()
	if err != nil {
		return err
	}

	logFile, logger, err := setupLogger(filepath.Join(agqDir, "agq.log"))
	if err != nil {
		return err
	}
	defer logFile.Close()
	slog.SetDefault(logger)

	db, err := store.Open(filepath.Join(agqDir, "agq.db"))
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	appState := state.New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	watchSignals(ctx, cancel, logger)

	infoCh := make(chan []*domain.ProcessInfo, 1)
	scanner := detector.New(logger)
	quotaClient := languageserver.NewClient(languageserver.RequestTimeout)
	quotaPoller := poller.New(db, appState, quotaClient, poller.Config{Logger: logger})

	go scanner.Run(ctx, infoCh)
	go quotaPoller.Run(ctx, infoCh)

	server := api.New(db, appState, api.WithLogger(logger))
	return server.Run(ctx, "localhost:"+envDefault("AGQ_PORT", "7432"))
}

func ensureAGQDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determine home directory: %w", err)
	}
	agqDir := filepath.Join(home, ".agq")
	if err := os.MkdirAll(agqDir, 0o755); err != nil {
		return "", fmt.Errorf("create %s: %w", agqDir, err)
	}
	return agqDir, nil
}

func setupLogger(logPath string) (*os.File, *slog.Logger, error) {
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, nil, fmt.Errorf("open log file %s: %w", logPath, err)
	}
	logger := slog.New(slog.NewJSONHandler(io.MultiWriter(os.Stderr, logFile), nil))
	return logFile, logger, nil
}

func watchSignals(ctx context.Context, cancel context.CancelFunc, logger *slog.Logger) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		select {
		case sig := <-sigs:
			logger.Info("agq-daemon shutting down", "signal", sig.String())
			cancel()
		case <-ctx.Done():
		}
		signal.Stop(sigs)
	}()
}

func envDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
