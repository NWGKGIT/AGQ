# Development and Releases

## Requirements

- Go 1.26+
- Node.js 24+
- Wails v2.13.0
- Linux: GTK3 and WebKitGTK 4.1 development packages
- Windows: Go, WebView2, and a supported C compiler toolchain

## Development

```sh
make desktop-dev
```

The desktop app runs the monitor in-process. The optional daemon is separate:

```sh
make build
./agq-daemon
```

It serves the local API on `localhost:7432` by default. Set `AGQ_PORT` to change the port.

## Verification

```sh
make test
make desktop-test
make docker-test
```

`desktop-test` builds the frontend before testing the Go package because the Wails backend embeds `frontend/dist`.

## Packaging

Linux AppImage:

```sh
make desktop-appimage
```

The packaging script requires `linuxdeploy-x86_64.AppImage` and `appimagetool-x86_64.AppImage` on `PATH` (or the command override variables documented in [`desktop/packaging/README.md`](../desktop/packaging/README.md)).

Windows MSIX packaging runs in GitHub Actions. It requires the protected repository variables `WINDOWS_PACKAGE_IDENTITY_NAME`, `WINDOWS_PACKAGE_PUBLISHER`, and `WINDOWS_PUBLISHER_DISPLAY_NAME`.

## Release Process

1. Update `productVersion` in `desktop/wails.json`.
2. Run the verification commands above.
3. Create and push a semantic version tag, for example `v1.0.0`.
4. GitHub Actions builds the Linux AppImage and Windows MSIX artifacts.
5. Attach checksums and release notes to the GitHub release.

The release workflow does not publish to the Microsoft Store automatically. Store submission and signing remain manual Partner Center steps.

## Project Layout

`internal/` contains the monitor, detector, poller, persistence, and API packages. `desktop/` contains the Wails shell and React frontend. `docs/` contains technical notes and release assets.
