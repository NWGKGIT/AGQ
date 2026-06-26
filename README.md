# AGQ Daemon

AGQ Daemon tracks Antigravity AI quota usage for every authenticated local
language server process and exposes the latest data over a local JSON API.

The daemon now handles multiple running workspaces. Detector scans can return
more than one language server, the poller deduplicates successful snapshots by
email, and `GET /api/status` reports all active account emails.

Data is stored in `~/.agq/agq.db` and the API listens on
`localhost:${AGQ_PORT:-7432}`.
