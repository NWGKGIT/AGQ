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
	// DefaultScanInterval is how often the detector rescans /proc.
	DefaultScanInterval = 15 * time.Second

	languageServerMarker = "language_server_linux_x64"
)

// PortProber discovers which loopback port serves GetUserStatus.
type PortProber interface {
	ProbeActivePort(ports []int, csrfToken string) (activePort int, scheme string)
}

// Scanner finds authenticated Antigravity language server processes.
type Scanner struct {
	Interval time.Duration
	Prober   PortProber
	Logger   *slog.Logger
}

// New creates a Scanner with production defaults.
func New(prober PortProber, logger *slog.Logger) *Scanner {
	return &Scanner{
		Interval: DefaultScanInterval,
		Prober:   prober,
		Logger:   logger,
	}
}

// Run scans immediately, then on each interval, until ctx is cancelled.
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

// ScanAll returns one ProcessInfo for every detected authenticated language
// server instance.
func (s *Scanner) ScanAll() []*domain.ProcessInfo {
	entries, err := filepath.Glob("/proc/*/cmdline")
	if err != nil {
		s.logger().Warn("detector: glob /proc/*/cmdline failed", "err", err)
		return nil
	}

	var results []*domain.ProcessInfo
	for _, path := range entries {
		if info, ok := s.tryParsePID(path); ok {
			results = append(results, info)
		}
	}
	return results
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

	csrfToken := extractFlag(args, "--csrf_token")
	if csrfToken == "" {
		return nil, false
	}

	info := &domain.ProcessInfo{
		Pid:                pid,
		CsrfToken:          csrfToken,
		ExtensionCsrfToken: extractFlag(args, "--extension_server_csrf_token"),
		WorkspaceID:        extractFlag(args, "--workspace_id"),
		CloudEndpoint:      extractFlag(args, "--cloud_code_endpoint"),
		HttpsServerPort:    extractFlagInt(args, "--https_server_port"),
		LspPort:            extractFlagInt(args, "--lsp_port"),
		ExtensionPort:      extractFlagInt(args, "--extension_server_port"),
	}

	ports := getLoopbackListeningPorts(pid)
	if len(ports) == 0 {
		s.logger().Debug("detector: no loopback listening ports", "pid", pid)
		return nil, false
	}

	if s.Prober == nil {
		s.logger().Warn("detector: no port prober configured")
		return nil, false
	}

	activePort, scheme := s.Prober.ProbeActivePort(ports, csrfToken)
	if activePort == 0 {
		s.logger().Debug("detector: no port responded to GetUserStatus", "pid", pid, "ports", ports)
		return nil, false
	}

	info.ActivePort = activePort
	info.Scheme = scheme

	s.logger().Info("detector: language server found",
		"pid", pid,
		"active_port", activePort,
		"scheme", scheme,
		"workspace_id", info.WorkspaceID)

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

// getLoopbackListeningPorts reads /proc/{pid}/net/tcp and /proc/{pid}/net/tcp6
// and returns all ports in LISTEN state on 127.0.0.1 or ::1.
func getLoopbackListeningPorts(pid int) []int {
	var ports []int
	for _, netFile := range []string{
		fmt.Sprintf("/proc/%d/net/tcp", pid),
		fmt.Sprintf("/proc/%d/net/tcp6", pid),
	} {
		ports = append(ports, parseNetTCPFile(netFile)...)
	}
	return dedupInts(ports)
}

func parseNetTCPFile(path string) []int {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return parseNetTCP(data)
}

func parseNetTCP(data []byte) []int {
	var ports []int
	for i, line := range strings.Split(string(data), "\n") {
		if i == 0 {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 || fields[3] != "0A" {
			continue
		}

		addrParts := strings.Split(fields[1], ":")
		if len(addrParts) != 2 || !isLoopbackHex(addrParts[0]) {
			continue
		}

		port, err := strconv.ParseInt(addrParts[1], 16, 32)
		if err != nil {
			continue
		}
		ports = append(ports, int(port))
	}
	return ports
}

func isLoopbackHex(ipHex string) bool {
	switch len(ipHex) {
	case 8:
		return strings.EqualFold(ipHex, "0100007F")
	case 32:
		return strings.EqualFold(ipHex, "00000000000000000000000001000000")
	default:
		return false
	}
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

func dedupInts(in []int) []int {
	seen := make(map[int]struct{}, len(in))
	out := make([]int, 0, len(in))
	for _, v := range in {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
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
