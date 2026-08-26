#!/usr/bin/env bash
set -euo pipefail
script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
desktop_dir="$(cd -- "${script_dir}/../.." && pwd)"
version="${1:?version required}"
root="${2:?staging directory required}"
cd "${desktop_dir}"
wails build -clean -trimpath -tags webkit2_41 -platform linux/amd64 -o AGQ
rm -rf -- "${root}"
mkdir -p "${root}/usr/bin" "${root}/usr/share/applications" "${root}/usr/share/icons/hicolor/512x512/apps" "${root}/usr/share/doc/agq" "${root}/usr/share/licenses/agq"
install -m 0755 build/bin/AGQ "${root}/usr/bin/agq"
install -m 0644 "${script_dir}/agq.desktop" "${root}/usr/share/applications/agq.desktop"
install -m 0644 "${script_dir}/agq.png" "${root}/usr/share/icons/hicolor/512x512/apps/agq.png"
install -m 0644 "${desktop_dir}/../LICENSE" "${root}/usr/share/doc/agq/LICENSE"
install -m 0644 "${desktop_dir}/../LICENSE" "${root}/usr/share/licenses/agq/LICENSE"
printf '%s\n' "${version}" > "${root}/usr/share/doc/agq/version"
