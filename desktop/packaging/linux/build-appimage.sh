#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
desktop_dir="$(cd -- "${script_dir}/../.." && pwd)"
appdir="${desktop_dir}/build/appimage/AGQ.AppDir"
output_dir="${desktop_dir}/build/appimage"
version="${VERSION:-1.0.0}"
version="${version#v}"
[[ "${version}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "Invalid VERSION: ${version}" >&2; exit 1; }
output_file="${output_dir}/AGQ-${version}-x86_64.AppImage"
linuxdeploy_command="${LINUXDEPLOY_COMMAND:-linuxdeploy-x86_64.AppImage}"
appimagetool_command="${APPIMAGETOOL_COMMAND:-appimagetool-x86_64.AppImage}"
linuxdeploy_url="${LINUXDEPLOY_URL:-https://github.com/linuxdeploy/linuxdeploy/releases/download/1-alpha-20251107-1/linuxdeploy-x86_64.AppImage}"
appimagetool_url="${APPIMAGETOOL_URL:-https://github.com/AppImage/appimagetool/releases/download/1.9.1/appimagetool-x86_64.AppImage}"
linuxdeploy_sha256="${LINUXDEPLOY_SHA256:-c20cd71e3a4e3b80c3483cef793cda3f4e990aca14014d23c544ca3ce1270b4d}"
appimagetool_sha256="${APPIMAGETOOL_SHA256:-ed4ce84f0d9caff66f50bcca6ff6f35aae54ce8135408b3fa33abfc3cb384eb0}"
if [[ "$(uname -m)" != "x86_64" ]]; then
  echo "The x86_64 AppImage must be built on an x86_64 Linux host." >&2
  exit 1
fi

native_ready=1
for command_name in pkg-config curl wails; do
  command -v "${command_name}" >/dev/null 2>&1 || native_ready=0
done
for pkg in gtk+-3.0 gio-unix-2.0 webkit2gtk-4.1 libsoup-3.0; do
  pkg-config --exists "${pkg}" 2>/dev/null || native_ready=0
done
if [[ "${native_ready}" == 0 && "${AGQ_APPIMAGE_CONTAINER:-}" != 1 ]]; then
  mkdir -p "${output_dir}"
  docker build -f "${script_dir}/docker/Dockerfile.appimage" -t agq-appimage "${desktop_dir}/.."
  container="$(docker create -e VERSION="${version}" agq-appimage)"
  trap 'docker rm -f "${container:-}" >/dev/null 2>&1 || true' EXIT
  docker start -a "${container}"
  docker cp "${container}:/artifacts/." "${output_dir}/"
  docker rm "${container}" >/dev/null
  exit 0
fi
if [[ "${native_ready}" == 0 ]]; then
  echo "Missing AppImage build prerequisites inside packaging container." >&2
  exit 1
fi

tools_dir="${desktop_dir}/build/appimage-tools"
mkdir -p -- "${tools_dir}"

download_tool() {
  local command_name="$1" url="$2"
  if command -v "${command_name}" >/dev/null 2>&1; then return; fi
  local target="${tools_dir}/${command_name}"
  expected_sha256="${linuxdeploy_sha256}"
  [[ "${command_name}" == "${appimagetool_command}" ]] && expected_sha256="${appimagetool_sha256}"
  if [[ -x "${target}" ]] && ! printf '%s  %s\n' "${expected_sha256}" "${target}" | sha256sum --check --status -; then
    rm -f -- "${target}"
  fi
  if [[ ! -x "${target}" ]]; then
    curl --fail --location --silent --show-error --retry 3 -o "${target}" "${url}"
    chmod +x "${target}"
  fi
  printf '%s  %s\n' "${expected_sha256}" "${target}" | sha256sum --check --status - || { echo "Checksum verification failed: ${target}" >&2; exit 1; }
  export PATH="${tools_dir}:${PATH}"
}

download_tool "${linuxdeploy_command}" "${linuxdeploy_url}"
download_tool "${appimagetool_command}" "${appimagetool_url}"

for required_command in wails "${linuxdeploy_command}" "${appimagetool_command}"; do
  if ! command -v "${required_command}" >/dev/null 2>&1; then
    echo "Missing required command: ${required_command}" >&2
    exit 1
  fi
done

cd -- "${desktop_dir}"
wails build -clean -trimpath -tags webkit2_41 -platform linux/amd64 -o AGQ

rm -rf -- "${appdir}"
mkdir -p -- \
  "${appdir}/usr/bin" \
  "${appdir}/usr/share/applications" \
  "${appdir}/usr/share/icons/hicolor/512x512/apps"
install -m 0755 "${desktop_dir}/build/bin/AGQ" "${appdir}/usr/bin/agq"
install -m 0755 "${script_dir}/AppRun" "${appdir}/AppRun"
install -m 0644 "${script_dir}/agq.desktop" "${appdir}/usr/share/applications/agq.desktop"
install -m 0644 "${script_dir}/agq.png" "${appdir}/usr/share/icons/hicolor/512x512/apps/agq.png"

"${linuxdeploy_command}" \
  --appdir "${appdir}" \
  --executable "${appdir}/usr/bin/agq" \
  --desktop-file "${appdir}/usr/share/applications/agq.desktop" \
  --icon-file "${appdir}/usr/share/icons/hicolor/512x512/apps/agq.png"

rm -f -- "${output_file}" "${output_file}.sha256"
env ARCH=x86_64 \
  APPIMAGE_EXTRACT_AND_RUN="${APPIMAGETOOL_APPIMAGE_EXTRACT_AND_RUN:-${APPIMAGE_EXTRACT_AND_RUN:-}}" \
  "${appimagetool_command}" "${appdir}" "${output_file}"
(cd "${output_dir}" && sha256sum "$(basename -- "${output_file}")" > "$(basename -- "${output_file}").sha256")

echo "Created ${output_file}"
