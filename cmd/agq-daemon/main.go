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
	"time"

	"agq-daemon/internal/detector"
	"agq-daemon/internal/domain"
	"agq-daemon/internal/languageserver"
	"agq-daemon/internal/poller"
	"agq-daemon/internal/state"
)

type memoryStore struct{}

func (memoryStore) SaveSnapshot(domain.QuotaSnapshot) error { return nil }

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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	watchSignals(ctx, cancel, logger)

	infoCh := make(chan []*domain.ProcessInfo, 1)
	scanner := detector.New(logger)
	quotaClient := languageserver.NewClient(languageserver.RequestTimeout)
	quotaPoller := poller.New(memoryStore{}, state.New(), quotaClient, poller.Config{Logger: logger})

	go scanner.Run(ctx, infoCh)
	go quotaPoller.Run(ctx, infoCh)

	<-ctx.Done()
	return nil
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

var _ = time.Second
