#!/usr/bin/env bash
set -euo pipefail
script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repo_dir="$(cd -- "${script_dir}/../../.." && pwd)"
version="${VERSION:-1.0.0}"; version="${version#v}"
artifact="${repo_dir}/desktop/build/appimage/AGQ-${version}-x86_64.AppImage"
test -f "${artifact}"
container="$(docker create --platform linux/amd64 debian:12@sha256:2f65600e1252c5649d2213e1d1ea4d74253d26514dc6530102a875e429245929 bash -lc 'apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y xvfb dbus-x11 desktop-file-utils file procps fuse; chmod +x /tmp/AGQ.AppImage; cd /tmp; /tmp/AGQ.AppImage --appimage-extract >/dev/null; test -x squashfs-root/usr/bin/agq; grep -qx Exec=agq squashfs-root/usr/share/applications/agq.desktop; desktop-file-validate squashfs-root/usr/share/applications/agq.desktop; set +e; dbus-run-session -- timeout 10s xvfb-run -a env APPIMAGE_EXTRACT_AND_RUN=1 /tmp/AGQ.AppImage; status=$?; test $status -eq 124')"
trap 'docker rm -f "${container:-}" >/dev/null 2>&1 || true' EXIT
docker cp "${artifact}" "${container}:/tmp/AGQ.AppImage"
docker start -a "${container}"
docker rm "${container}" >/dev/null
unset container
