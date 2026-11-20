#!/usr/bin/env bash
# Build the hostnamefetch Debian package from a source tag.
#
# Usage:
#   packaging/apt/build-deb.sh [tag]
#
# The source tag is expected to be a CalVer tag such as v2026.11.21.
# debian/changelog on the current branch must match <upstream>-1.
# The resulting package is written to packaging/apt/dist/hostnamefetch.deb.

set -euo pipefail

root=$(cd -- "$(dirname -- "$0")/../.." && pwd)

die() {
    printf 'build-deb: %s\n' "$1" >&2
    exit 1
}

for cmd in git tar dpkg-buildpackage dpkg-parsechangelog dpkg-deb sha256sum; do
    command -v "$cmd" >/dev/null 2>&1 || die "required command not found: $cmd"
done

if [[ ${1:-} ]]; then
    tag=$1
else
    version=$(sed -n 's/^version="\(.*\)"$/\1/p' "$root/hostnamefetch" | head -n1)
    [[ $version ]] || die "cannot read version from hostnamefetch"
    tag="v$version"
fi

[[ $tag =~ ^v[0-9]{4}\.[0-9]{2}\.[0-9]{2}(\.[0-9]+)?$ ]] || \
    die "invalid CalVer tag: $tag"

upstream=${tag#v}
expected_version="${upstream}-1"
changelog_version=$(dpkg-parsechangelog -l "$root/debian/changelog" -SVersion)
[[ $changelog_version == "$expected_version" ]] || \
    die "debian/changelog has $changelog_version, expected $expected_version"

git -C "$root" rev-parse --verify --quiet "$tag^{commit}" >/dev/null || \
    die "tag does not exist: $tag"

workdir=$(mktemp -d)
trap 'rm -rf "$workdir"' EXIT

source_name="hostnamefetch-$upstream"
upstream_dir="$workdir/upstream"
source_dir="$workdir/source"

mkdir -p "$upstream_dir" "$source_dir"

git -C "$root" archive --format=tar --prefix="$source_name/" "$tag" | \
    tar -x -C "$upstream_dir"
rm -rf "$upstream_dir/$source_name/debian"

tar -C "$upstream_dir" -czf "$source_dir/hostnamefetch_${upstream}.orig.tar.gz" \
    "$source_name"

cp -a "$upstream_dir/$source_name" "$source_dir/"
cp -a "$root/debian" "$source_dir/$source_name/debian"

(
    cd "$source_dir/$source_name"
    dpkg-buildpackage -b -us -uc
)

built=$(find "$source_dir" -maxdepth 1 -type f -name '*.deb' -print -quit)
[[ $built ]] || die "dpkg-buildpackage did not produce a .deb"

output_dir=${OUTPUT_DIR:-"$root/packaging/apt/dist"}
mkdir -p "$output_dir"
install -m 0644 "$built" "$output_dir/hostnamefetch.deb"

actual_version=$(dpkg-deb -f "$output_dir/hostnamefetch.deb" Version)
[[ $actual_version == "$changelog_version" ]] || \
    die "built package has version $actual_version, expected $changelog_version"

printf 'package: %s\n' "$output_dir/hostnamefetch.deb"
printf 'version: %s\n' "$actual_version"
printf 'sha256: %s\n' "$(sha256sum "$output_dir/hostnamefetch.deb" | awk '{print $1}')"
