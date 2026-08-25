//go:build linux

package detector

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func platformProcesses() ([]processCandidate, error) {
	entries, err := filepath.Glob("/proc/*/cmdline")
	if err != nil {
		return nil, fmt.Errorf("glob /proc/*/cmdline: %w", err)
	}

	candidates := make([]processCandidate, 0)
	for _, path := range entries {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		pid, err := pidFromCmdlinePath(path)
		if err != nil {
			continue
		}
		ppid, start, ok := linuxProcessMetadata(pid)
		if !ok {
			continue
		}
		candidates = append(candidates, processCandidate{pid: pid, ppid: ppid, createTime: start, args: splitProcCmdline(data)})
	}
	return candidates, nil
}

func linuxProcessMetadata(pid int) (int, time.Time, bool) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0, time.Time{}, false
	}
	closeParen := strings.LastIndexByte(string(data), ')')
	if closeParen < 0 {
		return 0, time.Time{}, false
	}
	fields := strings.Fields(string(data)[closeParen+2:])
	if len(fields) < 20 {
		return 0, time.Time{}, false
	}
	ppid, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0, time.Time{}, false
	}
	ticks, err := strconv.ParseInt(fields[19], 10, 64)
	if err != nil {
		return 0, time.Time{}, false
	}
	b, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, time.Time{}, false
	}
	var boot int64
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "btime ") {
			boot, _ = strconv.ParseInt(strings.TrimSpace(strings.TrimPrefix(line, "btime ")), 10, 64)
			break
		}
	}
	if boot == 0 {
		return 0, time.Time{}, false
	}
	return ppid, time.Unix(boot, 0).Add(time.Duration(ticks * int64(time.Second) / 100)), true
}

func platformLoopbackPorts(pid int, _ []int) ([]int, error) {
	var ports []int
	for _, netFile := range []string{
		fmt.Sprintf("/proc/%d/net/tcp", pid),
		fmt.Sprintf("/proc/%d/net/tcp6", pid),
	} {
		ports = append(ports, parseNetTCPFile(netFile)...)
	}
	return validPorts(ports), nil
}

func splitProcCmdline(data []byte) []string {
	parts := strings.Split(string(data), "\x00")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

func pidFromCmdlinePath(path string) (int, error) {
	pidDir := filepath.Base(filepath.Dir(path))
	pid, err := strconv.Atoi(pidDir)
	if err != nil {
		return 0, fmt.Errorf("parse pid from %q: %w", path, err)
	}
	return pid, nil
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
