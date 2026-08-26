#!/usr/bin/env bash
set -euo pipefail
script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repo_dir="$(cd -- "${script_dir}/../../.." && pwd)"
version="${VERSION:-1.0.0}"
version="${version#v}"
[[ "${version}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "Invalid VERSION: ${version}" >&2; exit 1; }
out="${repo_dir}/desktop/build/packages"
mkdir -p "${out}"
if [[ $# -gt 0 ]]; then kinds=("$1"); else kinds=(deb rpm arch); fi
for kind in "${kinds[@]}"; do
  [[ "${kind}" =~ ^(deb|rpm|arch)$ ]] || { echo "Unknown package format: ${kind}" >&2; exit 1; }
  image="agq-package-${kind}"
  docker build --platform linux/amd64 -f "${script_dir}/docker/Dockerfile.${kind}" -t "${image}" "${repo_dir}"
  container="$(docker create --platform linux/amd64 -e VERSION="${version}" "${image}")"
  trap 'docker rm -f "${container:-}" >/dev/null 2>&1 || true' EXIT
  docker start -a "${container}"
  docker cp "${container}:/artifacts/." "${out}/"
  docker rm "${container}" >/dev/null
  unset container
  case "${kind}" in
    deb) artifact="${out}/agq_${version}_amd64.deb" ;;
    rpm) artifact="${out}/agq-${version}-1.x86_64.rpm" ;;
    arch) artifact="${out}/agq-${version}-1-x86_64.pkg.tar.zst" ;;
  esac
  test -f "${artifact}"
  (cd "${out}" && sha256sum "$(basename -- "${artifact}")" > "$(basename -- "${artifact}").sha256")
done
