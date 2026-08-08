# Development & Release Guide

This document covers local development, testing, packaging, and the release process.

## Prerequisites

- **Go 1.26+**
- **Node 24+**
- **Wails v2 CLI** — install with `go install github.com/wailsio/wails/v2/cmd/wails@latest`
- **Linux:** GTK3, webkit2gtk-4.1 development headers
- **Windows:** MSVC or MinGW (for CGO); Visual Studio Build Tools recommended
- **macOS:** Xcode command-line tools (not fully supported yet)

## Project Layout

```
.
├── go.mod, go.sum             # Go dependencies
├── Makefile                    # Build targets
├── cmd/agq-daemon/             # Daemon entry point
├── internal/                   # Core packages
│   ├── domain/                 # Shared types
│   ├── detector/               # Process scanning
│   ├── languageserver/         # Language server client
│   ├── poller/                 # Polling orchestration
│   ├── store/                  # SQLite persistence
│   ├── state/                  # Runtime state
│   └── api/                    # HTTP API
├── monitor/                    # Monitor coordinator
├── desktop/                    # Wails frontend
│   ├── go.mod, go.sum         # Desktop-specific Go deps
│   ├── internal/
│   │   ├── apiclient/         # Typed API client
│   │   └── config/            # Settings persistence
│   ├── frontend/               # React SPA
│   │   ├── src/
│   │   ├── package.json        # Node deps + build scripts
│   │   ├── vite.config.ts      # Vite bundler config
│   │   ├── tsconfig.json
│   │   └── wailsjs/            # Auto-generated Wails bindings
│   ├── wails.json             # Wails build config
│   └── build/                 # Build output (binaries, AppImage)
├── docs/                       # Technical documentation
└── agq.service                 # systemd user unit
```

## Build Targets

All targets are defined in `Makefile`:

### Daemon (Headless)

```bash
make build          # Compile agq-daemon binary
make install        # Install binary + systemd unit to ~/.local
make enable         # Enable + start systemd user service
make disable        # Stop + disable service
make logs           # Tail systemd logs
make uninstall      # Remove binary + unit
make docker-build   # Build Docker image (requires --pid=host at runtime)
make docker-test    # Run Go tests inside container
```

### Desktop (Wails)

```bash
make desktop-dev       # Start dev server with hot reload (localhost:34115)
make desktop-build     # Compile binary (output: desktop/build/bin/AGQ)
make desktop-test      # Run all desktop tests (Go + frontend + typecheck)
make desktop-appimage  # Package x86_64 AppImage for release
```

### Testing

```bash
make test              # Run Go tests (daemon + monitor core)
make test-coverage     # Run tests with coverage report
GOCACHE=/tmp/... make test  # If build cache not writable
```

## Development Workflow

### Local Dev (Desktop)

```bash
make desktop-dev
```

Launches:
- Wails dev server on `localhost:34115` with hot reload
- Go monitor running with `AGQ_PORT=7432` (in-process)
- File watcher for both frontend and Go code

Edit files; changes reload automatically. Use browser dev tools (F12) for frontend debugging.

### Local Dev (Daemon)

```bash
make build
./agq-daemon
```

Runs the headless daemon. It listens on `localhost:7432` and logs to `~/.agq/agq.log`.

```bash
curl http://localhost:7432/api/health
```

### Adding a New API Endpoint

1. **Add handler in `internal/api/server.go`:**
   ```go
   func (s *Server) myEndpointHandler(w http.ResponseWriter, r *http.Request) {
     // ... logic
     s.writeJSON(w, http.StatusOK, result)
   }
   ```

2. **Register route in `monitor/runtime.go`:**
   ```go
   mux.HandleFunc("GET /api/my-endpoint", apiServer.myEndpointHandler)
   ```

3. **Add Wails binding in `desktop/internal/apiclient/client.go`:**
   ```go
   func (c *Client) MyEndpoint(ctx context.Context) (interface{}, error) {
     return c.Get(ctx, "/api/my-endpoint")
   }
   ```

4. **Auto-generate TypeScript types:**
   ```bash
   cd desktop
   wails generate
   ```

5. **Use in React:**
   ```typescript
   import { MyEndpoint } from '../../wailsjs/go/main/App'
   const { data } = useQuery({
     queryFn: MyEndpoint,
     // ...
   })
   ```

### Adding a New Database Query

1. **Add migration (if needed) in `internal/store/db.go`'s schema or `addColumnIfMissing()`**
2. **Add query method in `internal/store/db.go`:**
   ```go
   func (d *DB) MyQuery() ([]MyType, error) {
     rows, err := d.sql.Query(`SELECT ... WHERE ...`)
     // ... scan and return
   }
   ```
3. **Call from API handler or poller**
4. **Test with `make test`**

## Testing

### Go Tests

```bash
make test                  # All tests
make test-coverage        # Coverage report
go test ./internal/...     # Specific package
go test -run TestName ...  # Specific test
```

Tests are in `*_test.go` files alongside source. Key test suites:

- `internal/detector/detector_test.go` — process scanning, flag extraction
- `internal/languageserver/client_test.go` — response parsing
- `internal/poller/poller_test.go` — polling state machine
- `internal/store/db_test.go` — database queries

### Frontend Tests

```bash
cd desktop/frontend
npm test
```

### Type Checking

```bash
cd desktop/frontend
npm run typecheck
```

## Packaging & Release

### Desktop AppImage (Linux x86_64)

```bash
make desktop-appimage
```

Output: `desktop/build/bin/AGQ-x86_64.AppImage`

This is a portable, one-file executable for distribution. Users run:
```bash
chmod +x AGQ-x86_64.AppImage
./AGQ-x86_64.AppImage
```

### Windows MSIX

Build is performed on GitHub Actions (Windows runner). Local builds are untested.

### Release Checklist

Before pushing a release tag:

1. **Update version** in relevant places (if applicable):
   - `go.mod` (Go module version)
   - `desktop/wails.json` (app version)
   - Any hardcoded version strings

2. **Run full test suite:**
   ```bash
   make test
   make desktop-test
   make docker-test
   ```

3. **Manual smoke test (desktop):**
   ```bash
   make desktop-build
   ./desktop/build/bin/AGQ
   ```
   - Verify app launches
   - Check dashboard loads
   - Verify settings page works

4. **Build AppImage:**
   ```bash
   make desktop-appimage
   ```

5. **Verify AppImage:**
   ```bash
   ./desktop/build/bin/AGQ-x86_64.AppImage
   ```

6. **Create git tag and push:**
   ```bash
   git tag v1.2.3
   git push origin v1.2.3
   ```

7. **GitHub Actions will:**
   - Build MSIX for Windows (if configured)
   - Create Release with assets
   - Publish to Microsoft Store (if pipeline configured)

## CI/CD

GitHub Actions workflows in `.github/workflows/` (if present):
- **test.yml** — Run Go/desktop tests on every commit
- **build.yml** — Build AppImage/MSIX on release tags

## Database Migrations

AGQ uses a simple schema + incremental `ALTER TABLE` approach:

1. New columns are added via `addColumnIfMissing()` in `store/db.go`
2. Migrations run automatically on startup
3. No rollback support (treat as one-way)

To add a column:
```go
if err := db.addColumnIfMissing("model_quotas", "new_field", "TEXT"); err != nil {
  return nil, fmt.Errorf("migration: %w", err)
}
```

## Performance Profiling

### Go CPU Profile

```bash
go run -cpuprofile=cpu.prof ./cmd/agq-daemon &
# Let it run for ~30 seconds
kill %1
go tool pprof cpu.prof
```

### Memory Profile

```bash
go run -memprofile=mem.prof ./cmd/agq-daemon &
kill %1
go tool pprof mem.prof
```

## Debugging

### Enable Debug Logging

Set environment variable:
```bash
SLOG_LEVEL=debug make desktop-dev
```

### Attach Debugger

**Delve (Go debugger):**
```bash
dlv debug ./cmd/agq-daemon/
(dlv) break main.main
(dlv) continue
```

**Browser DevTools (Frontend):**
- Press F12 in Wails window
- Use Chrome DevTools to inspect/debug React

## Common Issues

**SQLite locked database:**
- Ensure only one process writes (desktop XOR daemon, not both)
- SQLite WAL mode allows concurrent readers

**Port already in use:**
```bash
lsof -i :7432  # Find process
kill -9 <pid>  # Kill it
```

**Wails binding issues:**
```bash
cd desktop
wails generate  # Regenerate TypeScript bindings
```

**Node modules corrupt:**
```bash
cd desktop/frontend
rm -rf node_modules package-lock.json
npm install
```
