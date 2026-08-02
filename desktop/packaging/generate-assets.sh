#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
desktop_dir="$(cd -- "${script_dir}/.." && pwd)"
source_svg="${script_dir}/brand/app-mark.svg"
asset_tmp_dir="$(mktemp -d)"
trap 'rm -rf -- "${asset_tmp_dir}"' EXIT

for required_command in rsvg-convert magick; do
  if ! command -v "${required_command}" >/dev/null 2>&1; then
    echo "Missing required command: ${required_command}" >&2
    exit 1
  fi
done

render_png() {
  local size="$1"
  local destination="$2"
  mkdir -p -- "$(dirname -- "${destination}")"
  rsvg-convert --width "${size}" --height "${size}" "${source_svg}" --output "${destination}"
}

render_png 1024 "${desktop_dir}/build/appicon.png"
render_png 512 "${script_dir}/linux/agq.png"
render_png 50 "${script_dir}/windows/assets/StoreLogo.png"
render_png 44 "${script_dir}/windows/assets/Square44x44Logo.png"
render_png 150 "${script_dir}/windows/assets/Square150x150Logo.png"
render_png 132 "${asset_tmp_dir}/wide-mark.png"
magick -size 310x150 xc:'#172554' "${asset_tmp_dir}/wide-mark.png" -gravity center -compose over -composite "${script_dir}/windows/assets/Wide310x150Logo.png"

for icon_size in 16 24 32 48 64 128 256; do
  render_png "${icon_size}" "${asset_tmp_dir}/icon-${icon_size}.png"
done
magick \
  "${asset_tmp_dir}/icon-16.png" \
  "${asset_tmp_dir}/icon-24.png" \
  "${asset_tmp_dir}/icon-32.png" \
  "${asset_tmp_dir}/icon-48.png" \
  "${asset_tmp_dir}/icon-64.png" \
  "${asset_tmp_dir}/icon-128.png" \
  "${asset_tmp_dir}/icon-256.png" \
  "${desktop_dir}/build/windows/icon.ico"

echo "Generated AGQ packaging assets."
