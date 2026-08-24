# Release packaging

This directory contains reproducible branding inputs and release scaffolding
for **AGQ**. Release outputs are written below
`desktop/build/` and are intentionally not committed.

The app is an unofficial, local monitor and is not affiliated with
Antigravity or any model provider. The original app mark uses blue/violet so
green, yellow, and red remain reserved for quota-health states in the UI.

## Brand assets

`brand/app-mark.svg` is the deterministic vector source. Regenerate the Wails,
MSIX, and AppImage raster assets on Linux with:

```sh
./packaging/generate-assets.sh
```

Generation requires `rsvg-convert` and ImageMagick. Commit the SVG and its
generated images together, and review the raster assets at their native sizes
before release.

## Microsoft Store (x64 MSIX)

Requirements:

- Windows 10/11 SDK (`makeappx.exe`, and `signtool.exe` for sideload signing)
- Go, Node.js, and Wails v2.13.0
- the exact package identity and publisher values assigned in Partner Center

From a Windows PowerShell prompt in `desktop/`:

```powershell
.\packaging\windows\build-msix.ps1 `
  -PackageIdentityName $env:WINDOWS_PACKAGE_IDENTITY_NAME `
  -Publisher $env:WINDOWS_PACKAGE_PUBLISHER `
  -PublisherDisplayName $env:WINDOWS_PUBLISHER_DISPLAY_NAME `
  -Version 1.0.0.0
```

The script builds with WebView2's browser recovery strategy, renders the
manifest, packages the x64 executable, and emits a SHA-256 file. It intentionally
has no checked-in publisher placeholder that could be mistaken for a Store
identity. Supply `-CertificatePath` (and, if required,
`-CertificatePassword`) only for a sideload test certificate whose subject
matches `-Publisher`. Partner Center handles signing for normal Store
submission.

Before submission:

1. Reserve the product name and copy the three identity values from Partner
   Center into protected repository variables.
2. Replace draft Store copy with final description, screenshots, age rating,
   privacy-policy URL, support URL, and data-use disclosures.
3. Run the Windows App Certification Kit against the packaged release.
4. Test install/update/uninstall as a standard user, WebView2 recovery, offline
   startup, high-DPI scaling, light/dark mode, and local process discovery.
5. Upload the generated MSIX through a protected release workflow. Never
   commit a PFX or its password.

The manifest requests only `runFullTrust`, which a Wails desktop process needs.
Add capabilities only when a shipped feature demonstrably requires them.

## Linux x86_64 AppImage

Install the Wails Linux build prerequisites, `linuxdeploy-x86_64.AppImage`, and
`appimagetool-x86_64.AppImage` on an x86_64 build host. Put both tools on
`PATH`, then run:

```sh
./packaging/linux/build-appimage.sh
```

Override tool command names with `LINUXDEPLOY_COMMAND` and
`APPIMAGETOOL_COMMAND`. The script builds the Wails executable, stages the
desktop entry/icon, lets linuxdeploy collect ELF dependencies, and emits an
AppImage plus SHA-256 file. It does not install systemd units or modify the
host system.

Validate the artifact on clean Ubuntu LTS and Arch installations under X11
and Wayland. WebKitGTK remains a compatibility-sensitive dependency; inspect
the AppDir produced by linuxdeploy and run smoke tests on the oldest supported
distribution before publishing. AppImage signing and zsync update metadata
are deferred until the project has a stable release repository URL and signing
key; do not invent either value in source control.

## Linux Flatpak

Flatpak is the preferred alternative on systems where AppImage's bundled
WebKitGTK libraries are unreliable. Install `flatpak-builder`, the GNOME 48
runtime and SDK, and the matching Go SDK extension, then run:

```sh
make desktop-flatpak
```

The script stages an offline build, compiles against the Flatpak runtime, and
emits `desktop/build/AGQ.flatpak` plus its SHA-256 file. Install it with:

```sh
flatpak install --user desktop/build/AGQ.flatpak
```

The sandbox persists only `~/.agq`, allows network access for local language
server discovery and the optional loopback API, and supports Wayland with an
X11 fallback.
