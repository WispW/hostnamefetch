#!/usr/bin/env bash
# Build the hostnamefetch Debian package.
#
# Usage:
#   packaging/apt/build-deb.sh [tag]
#   packaging/apt/build-deb.sh --worktree
#
# With a tag the source is taken from that tag (CalVer, e.g. v2026.11.21.1).
# With --worktree the current working tree is packaged instead, which lets CI
# validate a version bump before the tag exists. In both modes debian/changelog
# must match <upstream>-1.
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

worktree=off
tag=''

while (($# > 0)); do
    case $1 in
        -w|--worktree)
            worktree=on
            shift
        ;;
        *)
            [[ -z $tag ]] || die "unexpected argument: $1"
            tag=$1
            shift
        ;;
    esac
done

[[ $worktree == on && -n $tag ]] && die "--worktree cannot be combined with a tag"

if [[ $tag ]]; then
    [[ $tag =~ ^v[0-9]{4}\.[0-9]{2}\.[0-9]{2}(\.[0-9]+)?$ ]] || \
        die "invalid CalVer tag: $tag"
    upstream=${tag#v}
else
    upstream=$(sed -n 's/^version="\(.*\)"$/\1/p' "$root/hostnamefetch" | head -n1)
    [[ $upstream ]] || die "cannot read version from hostnamefetch"
    [[ $upstream =~ ^[0-9]{4}\.[0-9]{2}\.[0-9]{2}(\.[0-9]+)?$ ]] || \
        die "invalid version in hostnamefetch: $upstream"
fi

expected_version="${upstream}-1"
changelog_version=$(dpkg-parsechangelog -l "$root/debian/changelog" -SVersion)
[[ $changelog_version == "$expected_version" ]] || \
    die "debian/changelog has $changelog_version, expected $expected_version"

workdir=$(mktemp -d)
trap 'rm -rf "$workdir"' EXIT

source_name="hostnamefetch-$upstream"
upstream_dir="$workdir/upstream"
source_dir="$workdir/source"

mkdir -p "$upstream_dir" "$source_dir"

if [[ $worktree == on ]]; then
    mkdir -p "$upstream_dir/$source_name"
    tar -C "$root" -cf - \
        --exclude=./.git \
        --exclude=./.cache \
        --exclude=./packaging/apt/dist \
        --exclude=./windows/hostnamefetch.exe \
        . | tar -x -C "$upstream_dir/$source_name"
else
    git -C "$root" rev-parse --verify --quiet "$tag^{commit}" >/dev/null || \
        die "tag does not exist: $tag"

    git -C "$root" archive --format=tar --prefix="$source_name/" "$tag" | \
        tar -x -C "$upstream_dir"
fi

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
