# Phase 1 Update Walkthrough

All requested updates for AGQ Daemon have been implemented and successfully verified.

## What was changed

### 1. Robust Port Discovery
- Removed reliance on the `--https_server_port` flag from `cmdline`.
- The `detector.go` now extracts the `pid` of processes running with `--enable_lsp`.
- For each authenticated language server, it reads `/proc/{pid}/net/tcp` and `/proc/{pid}/net/tcp6`.
- It identifies all `LISTEN` ports bound to loopback addresses (`127.0.0.1` and `::1`).
- It tests each port directly using a `GetUserStatus` probe (trying both HTTP and HTTPS). The first port that responds with a valid `email` is recorded as the `ActivePort`.

### 2. Precise Quota Storing
- Modified `poller.go` to directly parse and store what the language server API provides, with **zero inference** or internal logic for replenishment.
- Replaced the ad-hoc model structs with an exact 1:1 match of the language server's `quotaInfo`.

### 3. Pool Grouping
- Added a `pool_reset_time` column to the `model_quotas` table using an idempotent migration mechanism in `db.go`. 
- Models that share a `pool_reset_time` can now be logically grouped together as sharing a single pool by the frontend dashboard.

### 4. Support for Multiple Workspaces
- `detector.go` now discovers and reports **all** running language servers instead of just the first one it finds.
- `poller.go` polls all instances simultaneously every 60 seconds.
- Results are deduplicated by email before persisting to SQLite (last successful response for a given email per cycle wins).
- The `status` endpoint now correctly surfaces `emails: [...]` containing the emails of all currently active workspaces.

### 5. Staleness Metadata
- Extended API endpoints to calculate and return `staleness_seconds` alongside `captured_at` to tell the UI exactly how fresh the data is.

## Verification

The daemon was rebuilt and smoke tested against the live IDE instance.

**Results of the API outputs:**
- `ActivePort` was successfully discovered over HTTPS (`detector: language server found pid=18141 active_port=45257 scheme=https`).
- `GET /api/accounts` correctly outputs models with `pool_reset_time` and standard snake_case json keys.
- Staleness fields are present and functioning properly.
