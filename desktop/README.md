# AGQ Desktop

The desktop application is a Wails v2 shell around the AGQ monitoring runtime. It embeds the React frontend and talks to the monitor in-process, so the headless daemon is not required.

## Stack

- Go and Wails v2.15.0
- React 19, TypeScript, and Vite
- Tailwind CSS v4 and Radix UI primitives
- TanStack Query and Recharts

## Development

From the repository root:

```sh
npm ci --prefix desktop/frontend
make desktop-dev
make desktop-test
make desktop-build
```

Linux builds use the `webkit2_41` tag for WebKitGTK 4.1. Generated Wails bindings live in `frontend/wailsjs/` and should not be edited manually.

Settings and quota history are stored under `~/.agq`.
