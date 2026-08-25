# Process Detection

AGQ discovers local Antigravity language server processes by scanning the operating system's process table every 15 seconds. This document describes the detection pipeline and platform-specific implementations.

## Discovery Pipeline

1. **Enumerate processes** - scan all running processes for Antigravity language servers
2. **Validate identity** - check executable name and command-line flags
3. **Extract configuration** - parse CSRF token and port declarations from process args
4. **Discover loopback ports** - find listening ports from OS kernel or declared args
5. **Probe for active port** - test each port with a `GetUserStatus` request, return first success

## Process Identification

A process is identified as an Antigravity language server if it meets all three criteria:

- **Executable name** starts with `language_server_` (case-insensitive), e.g., `language_server_linux_x64`
- **Has flag** `--enable_lsp` anywhere in command-line arguments
- **Has flag** `--csrf_token=<TOKEN>` (token is extracted and required to be non-empty)

## Command-Line Flags

The detector extracts these flags from process arguments:

| Flag | Purpose | Required |
|------|---------|----------|
| `--csrf_token` | CSRF token for authenticating `GetUserStatus` requests | Yes |
| `--workspace_id` | Workspace identifier (informational) | No |
| `--cloud_code_endpoint` | Cloud Code API endpoint | No |
| `--https_server_port` | Declared HTTPS server port | No |
| `--lsp_port` | Declared LSP server port | No |
| `--extension_server_port` | Declared extension server port | No |

Flags may use `--name=value` or `--name value` syntax.

## Loopback Port Discovery

After extracting declared ports, AGQ queries the OS to find which ports the process is actually listening on.

### Linux

Reads `/proc/[pid]/net/tcp` and `/proc/[pid]/net/tcp6` (kernel's TCP connection table for the process's namespace). Parses the local address field to extract listening port numbers. Filters to loopback (127.0.0.1 and ::1).

### Windows

Calls `GetExtendedTcpTable()` Windows API with `TCP_TABLE_OWNER_PID_ALL` filter to enumerate all TCP connections with owner PID. Filters to the target process PID and loopback addresses. Returns listening port numbers.

### Port Collection Logic

1. Collects ports from declared args (`--https_server_port`, `--lsp_port`, `--extension_server_port`)
2. Collects ports from OS socket table
3. Deduplicates and validates (1-65535)
4. If OS table is empty, falls back to declared ports
5. Returns sorted list for probing

## Port Probing

Once a set of candidate ports is known, AGQ probes each one to find the active endpoint.

**Probe sequence (first-match wins):**
1. Iterate ports in order (newest-first, via PID sorting)
2. For each port, try HTTPS first, then HTTP
3. Send POST to `http(s)://127.0.0.1:[port]/exa.language_server_pb.LanguageServerService/GetUserStatus`
4. Include headers:
   - `Content-Type: application/json`
   - `X-Codeium-Csrf-Token: [csrfToken]`
   - `Connect-Protocol-Version: 1`
5. Request body: `{"metadata":{"ideName":"antigravity","extensionName":"antigravity","ideVersion":"1.0.0","locale":"en"}}`
6. Return port + scheme on 200 OK with valid email in response; retry next port on any failure

**Probe timeout:** 2 seconds per port.

### Account Switch Scenario

When a user switches accounts in Antigravity:
1. Old language server process remains running (not immediately killed)
2. New language server process spawns with new PID and CSRF token
3. At the next 15-second scan:
   - Both old and new PIDs are detected
   - Both are probed; PID sort prefers newer (higher) PIDs
   - Old process may still respond with old account email
   - Newer PID is tried first, so new account usually wins
   - But if old process responds faster or is earlier in port list, stale data may be served

**Limitation:** AGQ cannot distinguish "which process is current" without Antigravity exposing that information. Restarting Antigravity (not AGQ) kills stale processes and forces fresh account data.

## Freshness and Refresh Interval

The detector runs continuously on a 5-second ticker (`DefaultScanInterval`). It identifies the newest Antigravity root, then accepts only child service processes whose PPID ancestry reaches that root and whose creation time is no older than the root. Of those, only the newest process is authoritative. Processes with unreadable metadata are rejected conservatively. Active quota polling runs every 3 seconds by default; process changes still trigger an immediate poll and repeated failures retain the five-minute backoff.

When an account is changed inside an existing Antigravity process, its PID can
remain unchanged. AGQ therefore does not re-probe loopback sockets on every
5-second detector scan: that would cause socket churn and repeatedly
re-authenticate identical connections. Instead, the unchanged process is
polled on the 3-second ticker; once `GetUserStatus` returns the new email,
the previous account is removed from the confirmed active state automatically.
AGQ deliberately keeps exactly one current account: the email returned by the
newest valid Antigravity process. Older accounts remain as history. Restart
Antigravity after logging into a new account so the new session is unambiguous
and old background processes cannot be mistaken for the current login.

## Thread Safety

The scanner runs in a goroutine. The process enumeration functions are platform-specific and read-only (no state mutation). Results are sent over a channel to the poller.
