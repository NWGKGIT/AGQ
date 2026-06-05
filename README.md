# AGQ Daemon

AGQ Daemon is a local Go daemon for tracking Antigravity quota usage from the
desktop language server.

The first milestone is a small process monitor that can find an authenticated
language server and persist the latest quota snapshot locally.

## Development

```sh
make build
make test
```
