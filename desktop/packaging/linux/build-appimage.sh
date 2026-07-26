#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
desktop_dir="$(cd -- "${script_dir}/../.." && pwd)"
appdir="${desktop_dir}/build/appimage/AntigravityTokenMonitor.AppDir"
output_dir="${desktop_dir}/build/appimage"
output_file="${output_dir}/AntigravityTokenMonitor-x86_64.AppImage"
linuxdeploy_command="${LINUXDEPLOY_COMMAND:-linuxdeploy-x86_64.AppImage}"
appimagetool_command="${APPIMAGETOOL_COMMAND:-appimagetool-x86_64.AppImage}"

for required_command in wails "${linuxdeploy_command}" "${appimagetool_command}"; do
  if ! command -v "${required_command}" >/dev/null 2>&1; then
    echo "Missing required command: ${required_command}" >&2
    exit 1
  fi
done

if [[ "$(uname -m)" != "x86_64" ]]; then
  echo "The x86_64 AppImage must be built on an x86_64 Linux host." >&2
  exit 1
fi

cd -- "${desktop_dir}"
wails build -clean -trimpath -tags webkit2_41 -platform linux/amd64 -o AntigravityTokenMonitor

rm -rf -- "${appdir}"
mkdir -p -- \
  "${appdir}/usr/bin" \
  "${appdir}/usr/share/applications" \
  "${appdir}/usr/share/icons/hicolor/512x512/apps"
install -m 0755 "${desktop_dir}/build/bin/AntigravityTokenMonitor" "${appdir}/usr/bin/AntigravityTokenMonitor"
install -m 0755 "${script_dir}/AppRun" "${appdir}/AppRun"
install -m 0644 "${script_dir}/AntigravityTokenMonitor.desktop" "${appdir}/usr/share/applications/AntigravityTokenMonitor.desktop"
install -m 0644 "${script_dir}/antigravity-token-monitor.png" "${appdir}/usr/share/icons/hicolor/512x512/apps/antigravity-token-monitor.png"

"${linuxdeploy_command}" \
  --appdir "${appdir}" \
  --executable "${appdir}/usr/bin/AntigravityTokenMonitor" \
  --desktop-file "${appdir}/usr/share/applications/AntigravityTokenMonitor.desktop" \
  --icon-file "${appdir}/usr/share/icons/hicolor/512x512/apps/antigravity-token-monitor.png"

rm -f -- "${output_file}" "${output_file}.sha256"
env ARCH=x86_64 "${appimagetool_command}" "${appdir}" "${output_file}"
sha256sum "${output_file}" > "${output_file}.sha256"

echo "Created ${output_file}"
