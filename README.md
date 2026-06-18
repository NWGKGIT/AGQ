# AGQ Daemon

AGQ Daemon tracks Antigravity AI quota usage from the local language server and
stores snapshots in SQLite under `~/.agq/agq.db`.

The database keeps accounts, snapshots, and per-model quota rows. WAL mode is
enabled so dashboard readers can query while the daemon writes.

```sh
make build
make test
make run
```
