# AGQ Daemon

AGQ Daemon tracks Antigravity AI quota usage from the local language server.

Current runtime flow:

1. Scan `/proc/*/cmdline` for authenticated Antigravity language servers.
2. Poll `GetUserStatus` on the detected loopback port.
3. Store quota snapshots for the active account.

Run locally with:

```sh
make run
```
