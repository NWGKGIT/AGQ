#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
desktop_dir="$(cd -- "${script_dir}/../.." && pwd)"
repo_dir="$(cd -- "${desktop_dir}/.." && pwd)"
manifest="${script_dir}/io.github.NWGKGIT.AGQ.yml"
output_dir="${desktop_dir}/build"
source_dir="${script_dir}/source"
builder_dir="${output_dir}/flatpak-builder"
flatpak_repo="${output_dir}/flatpak-repo"

for required_command in npm go flatpak flatpak-builder rsync; do
  if ! command -v "${required_command}" >/dev/null 2>&1; then
    echo "Missing required command: ${required_command}" >&2
    exit 1
  fi
done

rm -rf -- "${source_dir}" "${builder_dir}" "${flatpak_repo}"
mkdir -p -- "${source_dir}"
rsync -a \
  --exclude '.git' \
  --exclude 'desktop/build' \
  --exclude 'desktop/frontend/node_modules' \
  "${repo_dir}/" "${source_dir}/"

cd -- "${source_dir}/desktop/frontend"
npm ci
npm run build
rm -rf -- node_modules

cd -- "${source_dir}/desktop"
go mod vendor

cd -- "${script_dir}"
flatpak-builder --force-clean --repo="${flatpak_repo}" "${builder_dir}" "${manifest}"
rm -f -- "${output_dir}/AGQ.flatpak" "${output_dir}/AGQ.flatpak.sha256"
flatpak build-bundle "${flatpak_repo}" "${output_dir}/AGQ.flatpak" io.github.NWGKGIT.AGQ
sha256sum "${output_dir}/AGQ.flatpak" > "${output_dir}/AGQ.flatpak.sha256"
rm -rf -- "${source_dir}"

echo "Created ${output_dir}/AGQ.flatpak"
