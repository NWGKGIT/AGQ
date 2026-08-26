#!/usr/bin/env bash
set -euo pipefail
base="/tmp/agq-deb"; rm -rf "$base"
bash desktop/packaging/linux/stage-linux.sh "$VERSION" "$base"
mkdir -p "$base/DEBIAN"
mkdir -p /tmp/debian/debian
printf 'Source: agq\nPackage: agq\nArchitecture: amd64\nMaintainer: AGQ Contributors\nDescription: Local Antigravity quota monitor\n' > /tmp/debian/debian/control
shlibs="$(cd /tmp/debian && dpkg-shlibdeps -O -e"$base/usr/bin/agq")"
depends="${shlibs#shlibs:Depends=}"
printf 'Package: agq\nVersion: %s\nArchitecture: amd64\nMaintainer: AGQ Contributors\nDescription: Local Antigravity quota monitor\nDepends: %s\n' "$VERSION" "$depends" > "$base/DEBIAN/control"
cat > "$base/usr/share/doc/agq/copyright" <<'EOF'
Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/
Upstream-Name: AGQ
Source: https://github.com/NWGKGIT/AGQ
License: MIT
EOF
mkdir -p /artifacts
dpkg-deb --build "$base" "/artifacts/agq_${VERSION}_amd64.deb"
dpkg-deb --info "/artifacts/agq_${VERSION}_amd64.deb" >/dev/null
