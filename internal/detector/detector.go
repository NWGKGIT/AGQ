package detector

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"agq-daemon/internal/domain"
)

const (
	DefaultScanInterval = 15 * time.Second

	languageServerMarker = "language_server_linux_x64"
)

type Scanner struct {
	Interval time.Duration
	Logger   *slog.Logger
}

func New(logger *slog.Logger) *Scanner {
	return &Scanner{Interval: DefaultScanInterval, Logger: logger}
}

func (s *Scanner) Run(ctx context.Context, out chan<- []*domain.ProcessInfo) {
	ticker := time.NewTicker(s.interval())
	defer ticker.Stop()

	sendUpdate(ctx, out, s.ScanAll())
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sendUpdate(ctx, out, s.ScanAll())
		}
	}
}

func (s *Scanner) ScanAll() []*domain.ProcessInfo {
	entries, err := filepath.Glob("/proc/*/cmdline")
	if err != nil {
		s.logger().Warn("detector: glob /proc/*/cmdline failed", "err", err)
		return nil
	}
	for _, path := range entries {
		if info, ok := s.tryParsePID(path); ok {
			return []*domain.ProcessInfo{info}
		}
	}
	return nil
}

func sendUpdate(ctx context.Context, out chan<- []*domain.ProcessInfo, infos []*domain.ProcessInfo) {
	select {
	case out <- infos:
	case <-ctx.Done():
	}
}

func (s *Scanner) tryParsePID(path string) (*domain.ProcessInfo, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	args := splitCmdline(data)
	if !isLanguageServer(args) {
		return nil, false
	}
	pid, err := pidFromCmdlinePath(path)
	if err != nil {
		return nil, false
	}
	httpsPort := extractFlagInt(args, "--https_server_port")
	if httpsPort == 0 {
		return nil, false
	}
	info := &domain.ProcessInfo{
		Pid:                pid,
		CsrfToken:          extractFlag(args, "--csrf_token"),
		ActivePort:         httpsPort,
		Scheme:             "https",
		ExtensionCsrfToken: extractFlag(args, "--extension_server_csrf_token"),
		WorkspaceID:        extractFlag(args, "--workspace_id"),
		CloudEndpoint:      extractFlag(args, "--cloud_code_endpoint"),
		HttpsServerPort:    httpsPort,
		LspPort:            extractFlagInt(args, "--lsp_port"),
		ExtensionPort:      extractFlagInt(args, "--extension_server_port"),
	}
	s.logger().Info("detector: language server found", "pid", pid, "https_server_port", httpsPort)
	return info, true
}

func splitCmdline(data []byte) []string {
	parts := strings.Split(string(data), "\x00")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

func isLanguageServer(args []string) bool {
	return containsArgFragment(args, languageServerMarker) &&
		hasFlag(args, "--enable_lsp") &&
		hasFlag(args, "--csrf_token")
}

func containsArgFragment(args []string, fragment string) bool {
	for _, arg := range args {
		if strings.Contains(arg, fragment) {
			return true
		}
	}
	return false
}

func hasFlag(args []string, flag string) bool {
	prefix := flag + "="
	for _, arg := range args {
		if arg == flag || strings.HasPrefix(arg, prefix) {
			return true
		}
	}
	return false
}

func pidFromCmdlinePath(path string) (int, error) {
	pidDir := filepath.Base(filepath.Dir(path))
	pid, err := strconv.Atoi(pidDir)
	if err != nil {
		return 0, fmt.Errorf("parse pid from %q: %w", path, err)
	}
	return pid, nil
}

func extractFlag(args []string, flag string) string {
	prefix := flag + "="
	for i, arg := range args {
		if strings.HasPrefix(arg, prefix) {
			return strings.TrimPrefix(arg, prefix)
		}
		if arg == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

func extractFlagInt(args []string, flag string) int {
	s := extractFlag(args, flag)
	if s == "" {
		return 0
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return v
}

func (s *Scanner) interval() time.Duration {
	if s == nil || s.Interval <= 0 {
		return DefaultScanInterval
	}
	return s.Interval
}

func (s *Scanner) logger() *slog.Logger {
	if s == nil || s.Logger == nil {
		return slog.Default()
	}
	return s.Logger
}
