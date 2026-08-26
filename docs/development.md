# Development and Releases

## Requirements

- Go 1.26+
- Node.js 24+
- Wails v2.15.0
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

Linux packages (Docker is required; `VERSION` is required):

```sh
VERSION=1.0.0 make desktop-linux-packages
```

Build a single format with `make desktop-deb`, `make desktop-rpm`, or
`make desktop-arch`. `make desktop-appimage` builds the versioned fallback
AppImage and automatically uses an Ubuntu container when host GTK/WebKit
development packages are unavailable.

The packaging script requires `linuxdeploy-x86_64.AppImage` and `appimagetool-x86_64.AppImage` on `PATH` (or the command override variables documented in [`desktop/packaging/README.md`](../desktop/packaging/README.md)).

Windows MSIX packaging runs in GitHub Actions. It requires the protected repository variables `WINDOWS_PACKAGE_IDENTITY_NAME`, `WINDOWS_PACKAGE_PUBLISHER`, and `WINDOWS_PUBLISHER_DISPLAY_NAME`.

## Releases

Users should choose a version from the [GitHub Releases](https://github.com/NWGKGIT/AGQ/releases) page, then download the artifact for their platform:

| Platform                            | Recommended artifact                 |
| ----------------------------------- | ------------------------------------ |
| Debian or Ubuntu                    | `agq_<version>_amd64.deb`            |
| Fedora                              | `agq-<version>-1.x86_64.rpm`         |
| Arch Linux                          | `agq-<version>-1-x86_64.pkg.tar.zst` |
| Other Linux distributions           | `AGQ-<version>-x86_64.AppImage`      |
| Windows                             | `AGQ-<version>-windows-x64.zip`      |
| Microsoft Store or sideload testing | `AGQ-<version>.0-x64.msix`           |

The MSIX is included only when the repository has all three protected Windows
package identity variables configured. Every Linux package and AppImage also
includes a `.sha256` checksum file.

Maintainers can publish a release in either of these ways:

1. Push a semantic version tag, for example `git tag v1.0.0 && git push origin v1.0.0`.
2. Open **Actions**, select **Release packages**, choose **Run workflow**, and enter a semantic version such as `1.0.0`.

The workflow builds and tests the platform artifacts, verifies checksums, and
creates or updates the matching `v<version>` GitHub Release. A manual run uses
the selected workflow commit; it does not create a source commit or update the
default branch.

## Project Layout

`internal/` contains the monitor, detector, poller, persistence, and API packages. `desktop/` contains the Wails shell and React frontend. `docs/` contains technical notes and release assets.
