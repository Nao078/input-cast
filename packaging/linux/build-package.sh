#!/usr/bin/env bash
set -euo pipefail

app_name="input-cast-client"
version="${VERSION:-0.1.0}"
release="${RELEASE:-1}"
package_types="${PACKAGE_TYPE:-rpm deb}"

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
package_root="${root_dir}/.build/package/${app_name}"
dist_dir="${root_dir}/dist"

command -v fpm >/dev/null 2>&1 || {
  echo "fpm is required. Install it with: gem install fpm" >&2
  exit 1
}

cd "${root_dir}"
rm -rf "${package_root}"
mkdir -p "${package_root}/usr/bin" \
  "${package_root}/usr/share/applications" \
  "${package_root}/usr/share/icons/hicolor/256x256/apps" \
  "${dist_dir}"

go build -o "${package_root}/usr/bin/${app_name}" ./cmd/input-cast-client
install -Dm644 packaging/linux/input-cast-client.desktop "${package_root}/usr/share/applications/input-cast-client.desktop"
install -Dm644 cmd/input-cast-client/assets/display-icon.png "${package_root}/usr/share/icons/hicolor/256x256/apps/input-cast-client.png"

for package_type in ${package_types}; do
  (
    cd "${package_root}"
    fpm -s dir -t "${package_type}" \
      -n "${app_name}" \
      -v "${version}" \
      --iteration "${release}" \
      --description "Input Cast Client" \
      --license "MIT" \
      --package "${dist_dir}/${app_name}-${version}-${release}.${package_type}" \
      usr/bin/input-cast-client \
      usr/share/applications/input-cast-client.desktop \
      usr/share/icons/hicolor/256x256/apps/input-cast-client.png
  )
done
