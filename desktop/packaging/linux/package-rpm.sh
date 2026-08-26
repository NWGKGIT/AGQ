#!/usr/bin/env bash

set -euo pipefail

if [[ -z "${VERSION:-}" ]]; then
    echo "ERROR: VERSION is not set."
    echo "Example: VERSION=1.1.0 ./desktop/packaging/linux/build-native-packages.sh rpm"
    exit 1
fi

ROOT="/tmp/agq-rpm"
STAGING="/tmp/agq-rpm-stage"
ARTIFACTS="/artifacts"
PACKAGE_RELEASE="1"
PACKAGE_ARCH="x86_64"

rm -rf "$ROOT" "$STAGING" "$ARTIFACTS"

mkdir -p \
    "$ROOT/BUILD" \
    "$ROOT/RPMS" \
    "$ROOT/SOURCES" \
    "$ROOT/SPECS" \
    "$ROOT/SRPMS" \
    "$ARTIFACTS"

echo "==> Staging application..."
bash desktop/packaging/linux/stage-linux.sh \
    "$VERSION" \
    "$STAGING"

echo "==> Creating RPM spec..."

cat > "$ROOT/SPECS/agq.spec" <<EOF
Name:           agq
Version:        ${VERSION}
Release:        ${PACKAGE_RELEASE}
Summary:        Local Antigravity quota monitor
License:        MIT
URL:            https://github.com/NWGKGIT/AGQ
Packager:       AGQ Contributors
BuildArch:      ${PACKAGE_ARCH}

# stage-linux.sh already produces the final native binary. Do not create
# empty debuginfo/debugsource subpackages for this prebuilt payload.
%global debug_package %{nil}

Requires:       gtk3
Requires:       webkit2gtk4.1

%description
Local Antigravity quota monitor.

%install
install -d %{buildroot}
cp -a %{agq_stagedir}/. %{buildroot}/

%files
%{_bindir}/agq
%{_datadir}/applications/agq.desktop
%{_datadir}/icons/hicolor/512x512/apps/agq.png
%doc %{_docdir}/agq/LICENSE
%doc %{_docdir}/agq/version
%license %{_licensedir}/agq/LICENSE

%changelog
* Wed Aug 26 2026 AGQ Contributors <noreply@github.com> - ${VERSION}-${PACKAGE_RELEASE}
- Package AGQ for Fedora Linux.
EOF

echo "==> Preparing RPM build environment..."

chown -R builder:builder "$ROOT" "$STAGING" "$ARTIFACTS"

echo "==> Building RPM..."

runuser -u builder -- rpmbuild \
    --define "_topdir $ROOT" \
    --define "_builddir $ROOT/BUILD" \
    --define "_rpmdir $ARTIFACTS" \
    --define "agq_stagedir $STAGING" \
    -bb "$ROOT/SPECS/agq.spec"

echo "==> Locating RPM..."

rpm_name="agq-${VERSION}-${PACKAGE_RELEASE}.${PACKAGE_ARCH}.rpm"
mapfile -t rpm_files < <(find "$ARTIFACTS" -type f -name "$rpm_name" -print)

if (( ${#rpm_files[@]} != 1 )); then
    echo "ERROR: Expected exactly one $rpm_name, found ${#rpm_files[@]}." >&2
    exit 1
fi

rpm_file="${rpm_files[0]}"
output="$ARTIFACTS/$rpm_name"

if [[ "$rpm_file" != "$output" ]]; then
    mv "$rpm_file" "$output"
fi

echo "==> Checking RPM..."

rpm --checksig --nosignature "$output"

metadata="$(rpm -qp --queryformat '%{NAME} %{VERSION} %{RELEASE} %{ARCH}' "$output")"
expected_metadata="agq ${VERSION} ${PACKAGE_RELEASE} ${PACKAGE_ARCH}"
if [[ "$metadata" != "$expected_metadata" ]]; then
    echo "ERROR: Unexpected RPM metadata: $metadata" >&2
    exit 1
fi

expected_files=(
    /usr/bin/agq
    /usr/share/applications/agq.desktop
    /usr/share/doc/agq/LICENSE
    /usr/share/doc/agq/version
    /usr/share/icons/hicolor/512x512/apps/agq.png
    /usr/share/licenses/agq/LICENSE
)

mapfile -t packaged_files < <(rpm -qlp "$output")
for expected_file in "${expected_files[@]}"; do
    if ! printf '%s\n' "${packaged_files[@]}" | grep -Fxq -- "$expected_file"; then
        echo "ERROR: RPM payload is missing $expected_file" >&2
        exit 1
    fi
done

if ! rpm -qp --queryformat '[%{FILEMODES:perms} %{FILENAMES}\n]' "$output" \
    | grep -qx -- '-rwxr-xr-x /usr/bin/agq'; then
    echo "ERROR: /usr/bin/agq is not packaged with mode 0755." >&2
    exit 1
fi

echo
echo "RPM built successfully:"
echo "  $output"
