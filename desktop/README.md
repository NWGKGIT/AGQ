# Antigravity Token Monitor Desktop

Wails + React desktop dashboard for [Antigravity Token Monitor](../README.md).
The desktop process starts the monitoring runtime itself and the React frontend
uses typed Wails bindings. Existing Linux users can continue to run the legacy
headless daemon on `localhost:${AGQ_PORT:-7432}` during migration.

## Features

- **Overview** — provider aggregate strip (Gemini / Anthropic / OpenAI),
  live/idle daemon status with poll cadence, account cards with per-model
  quota bars, and a per-account detail sheet: pool status, 7-day sparklines,
  inferred login history, recent snapshots.
- **Analytics** — headline stats, remaining-quota-over-time chart (7d/30d,
  avg/min), and a sortable per-account consumption breakdown.
- **Settings** — daemon port with connection test, light/dark/system theme,
  and email masking for screenshots (also togglable from the sidebar).

Settings and quota history currently remain in the compatible `~/.agq` data
directory; theme preference persists locally in the webview.

## Stack

- [Wails v2](https://wails.io) shell (Go backend, native webview window)
- React 19 + TypeScript + Vite
- Tailwind CSS v4 + shadcn/ui-style components
- Dark/light theme, persisted locally

## Development

```sh
wails dev -tags webkit2_41     # hot reload
wails build -tags webkit2_41   # production binary at build/bin/AntigravityTokenMonitor
```

Or from the repository root: `make desktop-dev` / `make desktop-build`.

The `webkit2_41` tag targets webkit2gtk-4.1, which current Linux distros ship.

## Layout

- `main.go`, `app.go` — Wails entry point and bound `App` struct
- `frontend/src/pages/` — one file per sidebar page
- `frontend/src/components/ui/` — shadcn-style primitives
- `frontend/wailsjs/` — generated bindings (do not edit)
