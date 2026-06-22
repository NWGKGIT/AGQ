# AGQ Daemon

AGQ Daemon tracks Antigravity AI quota usage and exposes a local JSON API on
`localhost:${AGQ_PORT:-7432}`.

## API

- `GET /api/health`
- `GET /api/status`
- `GET /api/accounts`
- `GET /api/account/current`
- `GET /api/accounts/{email}/latest`
- `GET /api/accounts/{email}/snapshots`
- `GET /api/models/latest`

Data is stored in `~/.agq/agq.db` and logs are written to `~/.agq/agq.log`.
