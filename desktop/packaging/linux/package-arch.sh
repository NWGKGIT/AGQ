#!/usr/bin/env bash
set -euo pipefail
root="/tmp/agq-arch"; rm -rf "$root"; mkdir -p "$root"
bash desktop/packaging/linux/stage-linux.sh "$VERSION" "$root"
build_dir="/tmp/agq-makepkg"; rm -rf "$build_dir"; mkdir -p "$build_dir"
cat > "$build_dir/PKGBUILD" <<EOF
pkgname=agq
pkgver=${VERSION}
pkgrel=1
pkgdesc='Local Antigravity quota monitor'
arch=('x86_64')
license=('MIT')
depends=('gtk3' 'webkit2gtk-4.1')
package() { cp -a ${root}/* "\$pkgdir/"; }
EOF
chown -R builder:builder "$build_dir" "$root"
runuser -u builder -- bash -lc "cd '$build_dir' && makepkg --noconfirm --nodeps --nocheck"
mkdir -p /artifacts
install -m 0644 "$build_dir"/*.pkg.tar.zst /artifacts/
bsdtar -tf /artifacts/agq-"${VERSION}"-1-x86_64.pkg.tar.zst >/dev/null
