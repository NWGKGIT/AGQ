package detector

import (
	"encoding/binary"
	"reflect"
	"testing"
)

func TestExtractFlagForms(t *testing.T) {
	args := []string{
		"server",
		"--csrf_token=abc",
		"--workspace_id",
		"workspace-1",
		"--bad_int",
		"oops",
		"--https_server_port",
		"45257",
	}

	if got := extractFlag(args, "--csrf_token"); got != "abc" {
		t.Fatalf("extractFlag equals form = %q, want abc", got)
	}
	if got := extractFlag(args, "--workspace_id"); got != "workspace-1" {
		t.Fatalf("extractFlag split form = %q, want workspace-1", got)
	}
	if got := extractFlagInt(args, "--https_server_port"); got != 45257 {
		t.Fatalf("extractFlagInt = %d, want 45257", got)
	}
	if got := extractFlagInt(args, "--bad_int"); got != 0 {
		t.Fatalf("extractFlagInt invalid = %d, want 0", got)
	}
}

func TestIsLanguageServerRequiresMarkerLSPAndCSRF(t *testing.T) {
	for _, executable := range []string{
		"/tmp/language_server_linux_x64",
		`C:\Program Files\Antigravity\language_server_windows_x64.exe`,
	} {
		args := []string{executable, "--enable_lsp", "--csrf_token=abc"}
		if !isLanguageServer(args) {
			t.Fatalf("isLanguageServer(%q) returned false, want true", executable)
		}
	}

	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "missing marker", args: []string{"server", "--enable_lsp", "--csrf_token=abc"}},
		{name: "missing lsp", args: []string{"/tmp/language_server_linux_x64", "--csrf_token=abc"}},
		{name: "missing csrf", args: []string{"/tmp/language_server_linux_x64", "--enable_lsp"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if isLanguageServer(tc.args) {
				t.Fatal("isLanguageServer returned true, want false")
			}
		})
	}
}

func TestValidPortsFiltersAndPreservesOrder(t *testing.T) {
	got := validPorts([]int{3, 0, 1, 3, 65536, 2, 1})
	want := []int{3, 1, 2}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("validPorts() = %#v, want %#v", got, want)
	}
}

func TestScannerUsesInjectedPlatformDiscovery(t *testing.T) {
	prober := &recordingProber{activePort: 45257, scheme: "https"}
	scanner := New(prober, nil)
	scanner.enumerateProcesses = func() ([]processCandidate, error) {
		return []processCandidate{{
			pid: 42,
			args: []string{
				`C:\Antigravity\language_server_windows_x64.exe`,
				"--enable_lsp",
				"--csrf_token=secret",
				"--workspace_id", "workspace-1",
				"--https_server_port=45257",
			},
		}}, nil
	}
	scanner.enumeratePorts = func(pid int, declared []int) ([]int, error) {
		if pid != 42 {
			t.Fatalf("port discovery pid = %d, want 42", pid)
		}
		if want := []int{45257}; !reflect.DeepEqual(declared, want) {
			t.Fatalf("declared ports = %#v, want %#v", declared, want)
		}
		return declared, nil
	}

	got := scanner.ScanAll()
	if len(got) != 1 {
		t.Fatalf("ScanAll() returned %d processes, want 1", len(got))
	}
	if got[0].Pid != 42 || got[0].WorkspaceID != "workspace-1" || got[0].ActivePort != 45257 || got[0].Scheme != "https" {
		t.Fatalf("ScanAll() process = %#v", got[0])
	}
	if want := []int{45257}; !reflect.DeepEqual(prober.ports, want) {
		t.Fatalf("probed ports = %#v, want %#v", prober.ports, want)
	}
}

func TestParseWindowsTCPTableFiltersPIDStateAndLoopback(t *testing.T) {
	table := make([]byte, 4+4*windowsTCPv4OwnerPIDRowSize)
	binary.LittleEndian.PutUint32(table[:4], 4)
	writeWindowsIPv4Row(table[4:], windowsTCPListen, [4]byte{127, 0, 0, 1}, 7432, 42)
	writeWindowsIPv4Row(table[4+windowsTCPv4OwnerPIDRowSize:], windowsTCPListen, [4]byte{0, 0, 0, 0}, 7433, 42)
	writeWindowsIPv4Row(table[4+2*windowsTCPv4OwnerPIDRowSize:], 5, [4]byte{127, 0, 0, 1}, 7434, 42)
	writeWindowsIPv4Row(table[4+3*windowsTCPv4OwnerPIDRowSize:], windowsTCPListen, [4]byte{127, 0, 0, 1}, 7435, 99)

	if got, want := parseWindowsTCPTable(table, windowsAFInet, 42), []int{7432}; !reflect.DeepEqual(got, want) {
		t.Fatalf("parseWindowsTCPTable(IPv4) = %#v, want %#v", got, want)
	}
}

func TestParseWindowsTCP6TableFiltersLoopback(t *testing.T) {
	table := make([]byte, 4+2*windowsTCPv6OwnerPIDRowSize)
	binary.LittleEndian.PutUint32(table[:4], 2)
	loopback := [16]byte{}
	loopback[15] = 1
	writeWindowsIPv6Row(table[4:], windowsTCPListen, loopback, 9001, 42)
	writeWindowsIPv6Row(table[4+windowsTCPv6OwnerPIDRowSize:], windowsTCPListen, [16]byte{0: 0xfe, 1: 0x80}, 9002, 42)

	if got, want := parseWindowsTCPTable(table, windowsAFInet6, 42), []int{9001}; !reflect.DeepEqual(got, want) {
		t.Fatalf("parseWindowsTCPTable(IPv6) = %#v, want %#v", got, want)
	}
}

func TestParseWindowsTCPTableTruncatedBuffer(t *testing.T) {
	table := make([]byte, 5)
	binary.LittleEndian.PutUint32(table[:4], 100)
	if got := parseWindowsTCPTable(table, windowsAFInet, 42); got != nil {
		t.Fatalf("parseWindowsTCPTable(truncated) = %#v, want nil", got)
	}
}

type recordingProber struct {
	ports      []int
	activePort int
	scheme     string
}

func (p *recordingProber) ProbeActivePort(ports []int, _ string) (int, string) {
	p.ports = append([]int(nil), ports...)
	return p.activePort, p.scheme
}

func writeWindowsIPv4Row(row []byte, state uint32, address [4]byte, port int, pid uint32) {
	binary.LittleEndian.PutUint32(row[0:4], state)
	copy(row[4:8], address[:])
	binary.BigEndian.PutUint16(row[8:10], uint16(port))
	binary.LittleEndian.PutUint32(row[20:24], pid)
}

func writeWindowsIPv6Row(row []byte, state uint32, address [16]byte, port int, pid uint32) {
	copy(row[:16], address[:])
	binary.BigEndian.PutUint16(row[20:22], uint16(port))
	binary.LittleEndian.PutUint32(row[48:52], state)
	binary.LittleEndian.PutUint32(row[52:56], pid)
}
