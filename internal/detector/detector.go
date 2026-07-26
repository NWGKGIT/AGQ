package detector

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"agq-daemon/internal/domain"
)

const (
	// DefaultScanInterval is how often the detector rescans local processes.
	DefaultScanInterval = 15 * time.Second

	languageServerPrefix = "language_server_"
)

// PortProber discovers which loopback port serves GetUserStatus.
type PortProber interface {
	ProbeActivePort(ports []int, csrfToken string) (activePort int, scheme string)
}

type processCandidate struct {
	pid  int
	args []string
}

type processEnumerator func() ([]processCandidate, error)
type portEnumerator func(pid int, declaredPorts []int) ([]int, error)

// Scanner finds authenticated Antigravity language server processes.
type Scanner struct {
	Interval time.Duration
	Prober   PortProber
	Logger   *slog.Logger

	enumerateProcesses processEnumerator
	enumeratePorts     portEnumerator
}

// New creates a Scanner with production defaults for the current platform.
func New(prober PortProber, logger *slog.Logger) *Scanner {
	return &Scanner{
		Interval:           DefaultScanInterval,
		Prober:             prober,
		Logger:             logger,
		enumerateProcesses: platformProcesses,
		enumeratePorts:     platformLoopbackPorts,
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
	candidates, err := s.processes()()
	if err != nil {
		s.logger().Warn("detector: process enumeration failed", "err", err)
		return nil
	}

	results := make([]*domain.ProcessInfo, 0, len(candidates))
	for _, candidate := range candidates {
		if info, ok := s.tryCandidate(candidate); ok {
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

func (s *Scanner) tryCandidate(candidate processCandidate) (*domain.ProcessInfo, bool) {
	args := candidate.args
	if candidate.pid <= 0 || !isLanguageServer(args) {
		return nil, false
	}

	csrfToken := extractFlag(args, "--csrf_token")
	if csrfToken == "" {
		return nil, false
	}

	info := &domain.ProcessInfo{
		Pid:                candidate.pid,
		CsrfToken:          csrfToken,
		ExtensionCsrfToken: extractFlag(args, "--extension_server_csrf_token"),
		WorkspaceID:        extractFlag(args, "--workspace_id"),
		CloudEndpoint:      extractFlag(args, "--cloud_code_endpoint"),
		HttpsServerPort:    extractFlagInt(args, "--https_server_port"),
		LspPort:            extractFlagInt(args, "--lsp_port"),
		ExtensionPort:      extractFlagInt(args, "--extension_server_port"),
	}

	declaredPorts := validPorts([]int{info.HttpsServerPort, info.LspPort, info.ExtensionPort})
	ports, err := s.ports()(candidate.pid, declaredPorts)
	if err != nil {
		s.logger().Debug("detector: loopback port discovery failed", "pid", candidate.pid, "err", err)
	}
	ports = validPorts(ports)
	if len(ports) == 0 {
		s.logger().Debug("detector: no loopback listening ports", "pid", candidate.pid)
		return nil, false
	}

	if s.Prober == nil {
		s.logger().Warn("detector: no port prober configured")
		return nil, false
	}

	activePort, scheme := s.Prober.ProbeActivePort(ports, csrfToken)
	if activePort == 0 {
		s.logger().Debug("detector: no port responded to GetUserStatus", "pid", candidate.pid, "ports", ports)
		return nil, false
	}

	info.ActivePort = activePort
	info.Scheme = scheme

	s.logger().Info("detector: language server found",
		"pid", candidate.pid,
		"active_port", activePort,
		"scheme", scheme,
		"workspace_id", info.WorkspaceID)

	return info, true
}

func isLanguageServer(args []string) bool {
	return hasLanguageServerExecutable(args) &&
		hasFlag(args, "--enable_lsp") &&
		hasFlag(args, "--csrf_token")
}

func hasLanguageServerExecutable(args []string) bool {
	for _, arg := range args {
		name := arg
		if i := strings.LastIndexAny(name, `/\\`); i >= 0 {
			name = name[i+1:]
		}
		if strings.HasPrefix(strings.ToLower(name), languageServerPrefix) {
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

func validPorts(in []int) []int {
	seen := make(map[int]struct{}, len(in))
	out := make([]int, 0, len(in))
	for _, port := range in {
		if port < 1 || port > 65535 {
			continue
		}
		if _, ok := seen[port]; ok {
			continue
		}
		seen[port] = struct{}{}
		out = append(out, port)
	}
	return out
}

func (s *Scanner) processes() processEnumerator {
	if s != nil && s.enumerateProcesses != nil {
		return s.enumerateProcesses
	}
	return platformProcesses
}

func (s *Scanner) ports() portEnumerator {
	if s != nil && s.enumeratePorts != nil {
		return s.enumeratePorts
	}
	return platformLoopbackPorts
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

func unsupportedPlatformError(platform string) error {
	return fmt.Errorf("process detection is not supported on %s", platform)
}
