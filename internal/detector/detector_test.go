package detector

import (
	"reflect"
	"testing"
)

func TestParseNetTCPFiltersLoopbackListenPorts(t *testing.T) {
	data := []byte(`  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:1D08 00000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 123 1 0000000000000000 100 0 0 10 0
   1: 00000000:1D09 00000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 124 1 0000000000000000 100 0 0 10 0
   2: 0100007F:1D0A 00000000:0000 01 00000000:00000000 00:00000000 00000000  1000        0 125 1 0000000000000000 100 0 0 10 0
   3: 00000000000000000000000001000000:2329 00000000000000000000000000000000:0000 0A 00000000:00000000 00:00000000 00000000 1000 0 126 1 0000000000000000 100 0 0 10 0
`)

	got := parseNetTCP(data)
	want := []int{7432, 9001}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseNetTCP() = %#v, want %#v", got, want)
	}
}

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
	args := []string{
		"/tmp/language_server_linux_x64",
		"--enable_lsp",
		"--csrf_token=abc",
	}
	if !isLanguageServer(args) {
		t.Fatal("isLanguageServer returned false, want true")
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

func TestDedupIntsPreservesOrder(t *testing.T) {
	got := dedupInts([]int{3, 1, 3, 2, 1})
	want := []int{3, 1, 2}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("dedupInts() = %#v, want %#v", got, want)
	}
}
