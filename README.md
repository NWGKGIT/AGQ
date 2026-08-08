# AGQ

**AGQ** is a local-first desktop app that monitors your model quotas for
[Antigravity](https://antigravity.google). It discovers the Antigravity
language server running on your machine, polls its local API for per-model
quota levels, and turns the history into a dashboard: remaining percentages
per model and provider, reset countdowns, consumption analytics, and inferred
login sessions — for every account you use.

Everything runs on your machine. AGQ never talks to any external service:
it probes loopback ports only, authenticates with the language server's own
CSRF token, and stores snapshots in a local SQLite database.

- **Desktop app** for Linux (AppImage) and Windows (Microsoft Store, MSIX)
- **Optional headless daemon** for Linux servers/tinkerers, exposing the same
  data as a local JSON API

## Highlights

- **Per-model quotas** — remaining percentage for every model (Gemini,
  Anthropic, OpenAI pools), colored by provider, with reset countdowns.
- **Multi-account** — every account the monitor has ever seen, with health
  states, live/idle status, and a per-account detail sheet.
- **Analytics** — remaining-quota trend per provider (7d/30d), most depleted
  model, next reset, and a sortable per-model consumption breakdown.
- **Smart about resets** — if a quota reset passes while you're logged out or
  the app is off, AGQ serves the pool as refilled (flagged "assumed") instead
  of showing a stale depleted number forever.
- **Login timeline** — inferred login/logout sessions from snapshot gaps.
- **Privacy switches** — email masking for screenshots; the local API is only
  exposed on a loopback port if you opt in.

## Install

### Linux

Download the latest `AGQ-x86_64.AppImage` from Releases, then:

```sh
chmod +x AGQ-x86_64.AppImage
./AGQ-x86_64.AppImage
```

### Windows

Install **AGQ** from the Microsoft Store (or sideload the `.msix` from
Releases).

## Build From Source

Requirements: Go 1.26+, Node 24+, and the [Wails v2 CLI](https://wails.io).
On Linux additionally GTK3 and webkit2gtk-4.1.

```sh
make desktop-build    # binary at desktop/build/bin/AGQ
make desktop-dev      # hot-reload development
make desktop-appimage # x86_64 AppImage release artifact
```

Run the test suites:

```sh
make test          # Go packages (daemon + monitor core)
make desktop-test  # desktop Go tests + frontend tests + typecheck/build
make docker-test   # the Go suite inside a container
```

If your Go build cache is not writable in a restricted environment:

```sh
GOCACHE=/tmp/agq-go-cache go test ./...
```

## Optional Headless Daemon (Linux)

The desktop app embeds the monitor; nothing else is required. For servers or
scripting, a standalone daemon serves the same JSON API on
`localhost:${AGQ_PORT:-7432}`:

```sh
make build     # compile agq-daemon
make install   # install binary + systemd user unit
make enable    # start on login
```

`make status`, `make logs`, `make disable`, and `make uninstall` manage the
service. A containerized build is also available (`make docker-build`); note
that process discovery needs the host PID namespace (`docker run --pid=host`).

## Configuration

| Variable | Default | Description |
| --- | --- | --- |
| `AGQ_PORT` | `7432` | Local HTTP API port (daemon, or desktop with "Expose API" on). |

Desktop settings (theme, email masking, API exposure) live in
`~/.agq/desktop.json` and are editable from the Settings page.

## Data

AGQ stores runtime data under `~/.agq`:

- `agq.db` — SQLite snapshot history (WAL mode, safe for concurrent readers)
- `agq.log` — JSON log file (headless daemon)

## Documentation

Extensive technical documentation lives in [`docs/`](docs/):

| Document | Contents |
| --- | --- |
| [architecture.md](docs/architecture.md) | Package map, runtime flow, desktop vs daemon modes |
| [detection.md](docs/detection.md) | How language servers are discovered on Linux and Windows |
| [data-and-logic.md](docs/data-and-logic.md) | Snapshot model, reset cycles, assumed-refill semantics, session inference |
| [api.md](docs/api.md) | Full JSON API reference |
| [frontend.md](docs/frontend.md) | Design system, component map, data-fetch cadence |
| [development.md](docs/development.md) | Building, testing, packaging, release checklist |

## Disclaimer

AGQ is an unofficial, local monitoring tool and is not affiliated with
Antigravity or any model provider.
