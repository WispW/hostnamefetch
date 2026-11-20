#!/usr/bin/env bash
# AUR package updater for hostnamefetch.
#
# Usage:
#   packaging/aur/update.sh <tag> [--push]
#
#   <tag>    CalVer tag, e.g. v2026.11.21 or v2026.11.21.2
#   --push   Commit PKGBUILD/.SRCINFO to the AUR package repository.
#            SSH access to aur@aur.archlinux.org must already be configured.
#
# Environment:
#   REPO          GitHub repository. Default: WispW/hostnamefetch
#   AUR_PKG       AUR package name. Default: hostnamefetch
#   AUR_URL       AUR git remote. Default: ssh://aur@aur.archlinux.org/$AUR_PKG.git
#   AUR_GIT_NAME  Git author name for the AUR commit. Default: WispW
#   AUR_GIT_EMAIL Git author email for the AUR commit.
#                 Default: wispw@users.noreply.github.com

set -euo pipefail

repo=${REPO:-WispW/hostnamefetch}
aur_pkg=${AUR_PKG:-hostnamefetch}
aur_url=${AUR_URL:-ssh://aur@aur.archlinux.org/$aur_pkg.git}
script_dir=$(cd -- "$(dirname -- "$0")" && pwd)

usage() {
    printf '%s\n' \
        "Usage: $0 <tag> [--push]" \
        "" \
        "  <tag>    CalVer tag, e.g. v2026.11.21" \
        "  --push   push PKGBUILD/.SRCINFO to the AUR repository" >&2
    exit 1
}

die() {
    printf 'update.sh: %s\n' "$1" >&2
    exit 1
}

sha256() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | awk '{print $1}'
    else
        shasum -a 256 "$1" | awk '{print $1}'
    fi
}

tag=${1:-}
[[ $tag ]] || usage
[[ $tag =~ ^v[0-9]{4}\.[0-9]{2}\.[0-9]{2}(\.[0-9]+)?$ ]] || \
    die "invalid CalVer tag: $tag"
[[ ${2:-} == "--push" || ${2:-} == "" ]] || usage

pkgver=${tag#v}
workdir=$(mktemp -d)
trap 'rm -rf "$workdir"' EXIT

curl -fsSL "https://raw.githubusercontent.com/$repo/$tag/hostnamefetch" \
    -o "$workdir/hostnamefetch"
curl -fsSL "https://raw.githubusercontent.com/$repo/$tag/LICENSE" \
    -o "$workdir/LICENSE"

actual_version=$(bash "$workdir/hostnamefetch" --version 2>/dev/null | awk '{print $2}')
[[ $actual_version == "$pkgver" ]] || \
    die "tag $tag contains hostnamefetch $actual_version, expected $pkgver"

sha_script=$(sha256 "$workdir/hostnamefetch")
sha_license=$(sha256 "$workdir/LICENSE")

sed -e "s/@PKGVER@/$pkgver/g" \
    -e "s/@SHA256_SCRIPT@/$sha_script/g" \
    -e "s/@SHA256_LICENSE@/$sha_license/g" \
    "$script_dir/PKGBUILD.in" > "$script_dir/PKGBUILD"

# makepkg refuses to run as root, so use Docker for .SRCINFO generation
# on CI runners and other rootless environments without makepkg.
if command -v makepkg >/dev/null 2>&1 && ((EUID != 0)); then
    (cd "$script_dir" && makepkg --printsrcinfo > .SRCINFO)
elif command -v docker >/dev/null 2>&1; then
    docker run --rm \
        --user "$(id -u):$(id -g)" \
        --env HOME=/tmp \
        --volume "$script_dir:/src" \
        --workdir /src \
        archlinux:latest \
        bash -c 'makepkg --printsrcinfo' > "$script_dir/.SRCINFO"
else
    die "makepkg or docker is required to generate .SRCINFO"
fi

printf 'generated: %s\n' "$script_dir/PKGBUILD"
printf 'generated: %s\n' "$script_dir/.SRCINFO"
printf 'pkgver: %s\n' "$pkgver"
printf 'sha256: %s %s\n' "$sha_script" "$sha_license"

if [[ ${2:-} == "--push" ]]; then
    aur_repo="$workdir/aur"
    git clone "$aur_url" "$aur_repo"
    cp "$script_dir/PKGBUILD" "$script_dir/.SRCINFO" "$aur_repo/"
    cd "$aur_repo"
    git checkout -B master 2>/dev/null || git symbolic-ref HEAD refs/heads/master
    git config user.name "${AUR_GIT_NAME:-WispW}"
    git config user.email "${AUR_GIT_EMAIL:-wispw@users.noreply.github.com}"
    git add PKGBUILD .SRCINFO

    if git diff --cached --quiet; then
        printf '%s\n' "AUR package is already up to date"
    else
        git commit -m "$aur_pkg $pkgver-1"
        git push origin master
        printf 'pushed: https://aur.archlinux.org/packages/%s\n' "$aur_pkg"
    fi
fi
