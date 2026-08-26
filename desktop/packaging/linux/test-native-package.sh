#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repo_dir="$(cd -- "${script_dir}/../../.." && pwd)"
kind="${1:?package format required: deb, rpm, or arch}"
version="${VERSION:-1.0.0}"
version="${version#v}"
[[ "${version}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "Invalid VERSION: ${version}" >&2; exit 1; }

case "${kind}" in
  deb)
    image="debian:12@sha256:2f65600e1252c5649d2213e1d1ea4d74253d26514dc6530102a875e429245929"
    artifact="${repo_dir}/desktop/build/packages/agq_${version}_amd64.deb"
    install_command='apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends /tmp/agq.deb xvfb dbus-x11 desktop-file-utils procps && test "$(dpkg-query -W -f='"'"'${Version}'"'"' agq)" = "'"${version}"'"'
    uninstall_command='apt-get remove -y agq'
    ;;
  rpm)
    image="fedora:42@sha256:7c63468daf71fdc5bda3699cd483b169bb995b5137265d5ffe8f04e2ce87fbb8"
    artifact="${repo_dir}/desktop/build/packages/agq-${version}-1.x86_64.rpm"
    install_command='dnf install -y /tmp/agq.rpm xorg-x11-server-Xvfb dbus-x11 desktop-file-utils procps-ng && test "$(rpm -q --qf "%{VERSION}" agq)" = "'"${version}"'"'
    uninstall_command='dnf remove -y agq'
    ;;
  arch)
    image="archlinux:latest@sha256:c9dc8b5d1b06d8d50ace6d42b2c93fbb1e34c9e1332d1a2102936e497d3187ae"
    artifact="${repo_dir}/desktop/build/packages/agq-${version}-1-x86_64.pkg.tar.zst"
    install_command='pacman -Syu --noconfirm && pacman -S --noconfirm xorg-server-xvfb dbus desktop-file-utils procps-ng && pacman -U --noconfirm /tmp/agq.pkg.tar.zst && test "$(pacman -Q agq | cut -d" " -f2)" = "'"${version}"'-1"'
    uninstall_command='pacman -Rns --noconfirm agq'
    ;;
  *) echo "Unknown package format: ${kind}" >&2; exit 1 ;;
esac

test -f "${artifact}"
container="$(docker create --platform linux/amd64 "${image}" bash -lc "${install_command}; test -x /usr/bin/agq; test \"\$(stat -c %a /usr/bin/agq)\" = 755; grep -qx Exec=agq /usr/share/applications/agq.desktop; desktop-file-validate /usr/share/applications/agq.desktop; test -f /usr/share/icons/hicolor/512x512/apps/agq.png; test -f /usr/share/licenses/agq/LICENSE; ! ldd /usr/bin/agq | grep -q 'not found'; dbus-run-session -- bash -lc 'set +e; timeout 10s xvfb-run -a /usr/bin/agq; status=\$?; test \$status -eq 124'; ${uninstall_command}; test ! -e /usr/bin/agq")"
trap 'docker rm -f "${container:-}" >/dev/null 2>&1 || true' EXIT
case "${kind}" in
  deb) docker cp "${artifact}" "${container}:/tmp/agq.deb" ;;
  rpm) docker cp "${artifact}" "${container}:/tmp/agq.rpm" ;;
  arch) docker cp "${artifact}" "${container}:/tmp/agq.pkg.tar.zst" ;;
esac
docker start -a "${container}"
docker rm "${container}" >/dev/null
unset container
