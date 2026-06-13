package detector

import "testing"

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
}
