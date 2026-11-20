#!/usr/bin/env bash
# shellcheck disable=SC1091,SC2034,SC2329
# SC1091: the sourced path is computed from $0.
# SC2034: test globals are consumed by the sourced functions.
# SC2329: system_profiler() is called indirectly by get_resolution/get_gpu.
# Read-only smoke tests for hostnamefetch.
#
# Usage: bash tests/smoke.sh
#
# These tests only read the repository and render data in memory.

root=$(cd -- "$(dirname -- "$0")/.." && pwd)

# shellcheck source=../hostnamefetch
source "$root/hostnamefetch"

failures=0

fail() {
    printf 'FAIL: %s\n' "$1"
    failures=$((failures + 1))
}

# ---------------------------------------------------------------------------
# Font: every glyph must be five rows of five columns.
# ---------------------------------------------------------------------------

for ch in "${!GLYPHS[@]}"; do
    IFS='|' read -r -a rows <<< "${GLYPHS[$ch]}"

    ((${#rows[@]} == 5)) || fail "glyph '$ch' has ${#rows[@]} rows"

    for row in "${rows[@]}"; do
        ((${#row} == 5)) || fail "glyph '$ch' row '$row' is ${#row} columns"
    done
done

# ---------------------------------------------------------------------------
# Renderer
# ---------------------------------------------------------------------------

c1=""
reset=""
banner_spacing=1
glyph_fill="#"

readarray -t chunks < <(make_banner_chunks 100 "ABC-123")
render_banner "${chunks[@]}"

((${#BANNER_LINES[@]} == 5)) || fail "banner should be 5 lines"
((BANNER_WIDTH == 41)) || fail "banner width should be 41, got $BANNER_WIDTH"

# ---------------------------------------------------------------------------
# Wrapping
# ---------------------------------------------------------------------------

readarray -t chunks < <(make_banner_chunks 4 "ABCDEFG")
((${#chunks[@]} == 2)) || fail "ABCDEFG should wrap into 2 chunks"
[[ ${chunks[0]} == "ABCD" ]] || fail "first chunk should be ABCD"

readarray -t chunks < <(make_banner_chunks 4 "AB-CD")
[[ ${chunks[0]} == "AB-" ]] || fail "prefer breaking after '-'"
[[ ${chunks[1]} == "CD" ]] || fail "second chunk should be CD"

readarray -t chunks < <(make_banner_chunks 1 "AB")
((${#chunks[@]} == 2)) || fail "max_chars=1 should wrap every character"

# ---------------------------------------------------------------------------
# The whole supported glyph set renders without falling back to '?'
# ---------------------------------------------------------------------------

text="ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._ ?@:/+=~"
readarray -t chunks < <(make_banner_chunks 200 "$text")
render_banner "${chunks[@]}"
((${#BANNER_LINES[@]} == 5)) || fail "full glyph set should render 5 lines"

# ---------------------------------------------------------------------------
# Long hostnames: chunks must stay within the limit and rebuild the original.
# ---------------------------------------------------------------------------

long_name="web-prod-us-east-1-node-0001-cluster-alpha-example-internal"

readarray -t chunks < <(make_banner_chunks 7 "$long_name")
joined=""
for chunk in "${chunks[@]}"; do
    joined+=$chunk
    ((${#chunk} <= 7)) || fail "long hostname chunk is wider than 7: '$chunk'"
done
[[ $joined == "$long_name" ]] || fail "long hostname chunks do not rebuild input"

# A single token without separators must still be hard-split.
readarray -t chunks < <(make_banner_chunks 7 "abcdefghijklmnopqrstuvwxyz0123")
((${#chunks[@]} > 1)) || fail "unbroken token should be split"

# Rendering many chunks must produce 5 lines per chunk plus separators.
readarray -t chunks < <(make_banner_chunks 7 "$long_name")
render_banner "${chunks[@]}"
expected_lines=$(( ${#chunks[@]} * 5 + ${#chunks[@]} - 1 ))
((${#BANNER_LINES[@]} == expected_lines)) || \
    fail "long banner should be $expected_lines lines"

# Every info row must start at the same column, even with a wrapped banner.
INFO_COLORED=()
for ((i = 0; i < 40; i++)); do
    printf -v row 'i%02d' "$i"
    INFO_COLORED+=("$row")
done

layout=left
banner=on
stdout=off
valign=center
gap=3

readarray -t out_lines < <(print_output)

aligned=$(printf '%s\n' "${out_lines[@]}" | \
    awk -v want="$((BANNER_WIDTH + gap + 1))" '
        /i[0-9][0-9]/ && index($0, "i") != want { bad++ }
        END { print bad + 0 }
    ')
((aligned == 0)) || fail "wrapped banner misaligned $aligned info row(s)"

# ---------------------------------------------------------------------------
# Vertical alignment
# ---------------------------------------------------------------------------

layout=left
banner=on
stdout=off
gap=0
BANNER_WIDTH=0
BANNER_WIDTHS=(0 0 0 0 0)
BANNER_COLORED=("b0" "b1" "b2" "b3" "b4")
INFO_COLORED=("i0" "i1" "i2" "i3" "i4" "i5" "i6" "i7" "i8")

valign=center
readarray -t out_lines < <(print_output)
[[ ${out_lines[0]} == "i0" ]] || fail "centered layout should start with info"
[[ ${out_lines[2]} == "b0i2" ]] || fail "banner should be centered (row 2)"

valign=top
readarray -t out_lines < <(print_output)
[[ ${out_lines[0]} == "b0i0" ]] || fail "top layout should start with the banner"

# ---------------------------------------------------------------------------
# Review regressions: color 7, macOS cache, memory fallback
# ---------------------------------------------------------------------------

reset=$'\e[0m'
[[ $(color 7) == $'\e[0m\e[37m' ]] || fail "color 7 should reset before white"
[[ $(color 4) == $'\e[0m\e[34m' ]] || fail "color 4 should reset before blue"

# macOS system_profiler must be called only once for resolution + GPU.
sp_calls=$(mktemp)
os=macOS
unset macos_display_info macos_display_info_loaded

system_profiler() {
    printf 'call\n' >> "$sp_calls"
    printf 'Resolution: 1920 x 1080\nChipset Model: Test GPU\n'
}

get_resolution
get_gpu

calls=$(wc -l < "$sp_calls")
((calls == 1)) || fail "macOS system_profiler should be called once, got $calls"

unset -f system_profiler
unset macos_display_info macos_display_info_loaded
rm -f "$sp_calls"

# Missing MemAvailable must fall back to MemFree + Buffers + Cached.
meminfo=$(mktemp)
cat > "$meminfo" <<'MEMINFO'
MemTotal:       1000000 kB
MemFree:         400000 kB
Buffers:         100000 kB
Cached:          200000 kB
MEMINFO

os=Linux
MEMINFO_FILE=$meminfo
memory=""
get_memory
rm -f "$meminfo"
unset MEMINFO_FILE

[[ $memory == "292MiB / 976MiB (30%)" ]] || \
    fail "memory fallback mismatch: '$memory'"

# macOS df -P reports 512-byte blocks; get_disk must use human-readable -h.
os=macOS
df() {
    printf '%s\n' 'Filesystem Size Used Avail Capacity Mounted on'
    printf '%s\n' '/dev/disk3s1s1 460Gi 9.7Gi 296Gi 4% /'
}

disk=""
get_disk
unset -f df
os=Linux

[[ $disk == "9.7Gi / 460Gi (4%)" ]] || \
    fail "macOS disk output mismatch: '$disk'"

# ---------------------------------------------------------------------------
# CLI smoke tests
# ---------------------------------------------------------------------------

hf_bash=${BASH:-bash}

"$hf_bash" "$root/hostnamefetch" --version >/dev/null || fail "--version failed"
"$hf_bash" "$root/hostnamefetch" --help >/dev/null || fail "--help failed"

"$hf_bash" "$root/hostnamefetch" --stdout | grep -q '^Hostname: ' || \
    fail "--stdout output is missing Hostname"

if "$hf_bash" "$root/hostnamefetch" --layout nope >/dev/null 2>&1; then
    fail "--layout nope should fail"
fi

if "$hf_bash" "$root/hostnamefetch" --banner_colors 300 >/dev/null 2>&1; then
    fail "--banner_colors 300 should fail"
fi

# --disable title must also suppress the underline.
out=$("$hf_bash" "$root/hostnamefetch" --stdout --disable title --disable cols)
[[ ${out%%$'\n'*} == Hostname:* ]] || \
    fail "--disable title should skip the underline"

# Unknown --disable items should warn, not fail.
err=$("$hf_bash" "$root/hostnamefetch" --stdout --disable typo --disable cols 2>&1 >/dev/null)
[[ $err == *"unknown --disable item 'typo'"* ]] || \
    fail "unknown --disable should print a warning"

# Explicit left layout should warn when it falls back to top.
err=$(COLUMNS=20 "$hf_bash" "$root/hostnamefetch" --layout left --disable cols 2>&1 >/dev/null)
[[ $err == *"falling back to top"* ]] || \
    fail "narrow --layout left should warn about the fallback"

# ---------------------------------------------------------------------------
# Long hostname integration (also runs under Git Bash on Windows)
# ---------------------------------------------------------------------------

out=$(COLUMNS=80 "$hf_bash" "$root/hostnamefetch" \
    --banner_text "$long_name" \
    --banner_chars ascii \
    --color_text off \
    --disable cols)

over=$(printf '%s\n' "$out" | \
    awk 'length($0) > 79 { n++ } END { print n + 0 }')
((over == 0)) || fail "top layout overflows 80 columns ($over line(s))"

out=$(COLUMNS=80 "$hf_bash" "$root/hostnamefetch" \
    --banner_text "$long_name" \
    --banner_chars ascii \
    --color_text off \
    --disable cols \
    --layout left)

over=$(printf '%s\n' "$out" | \
    awk 'length($0) > 79 { n++ } END { print n + 0 }')
((over == 0)) || fail "left layout overflows 80 columns ($over line(s))"

aligned=$(printf '%s\n' "$out" | \
    awk '
        /Hostname: / { h = index($0, "Hostname:") }
        /OS: / { o = index($0, "OS:") }
        END { print (h && h == o) ? h : 0 }
    ')
((aligned > 0)) || fail "left layout info column is not aligned"

# ---------------------------------------------------------------------------
# AUR packaging template/script sanity
# ---------------------------------------------------------------------------

grep -q '@PKGVER@' "$root/packaging/aur/PKGBUILD.in" || \
    fail "PKGBUILD.in is missing the @PKGVER@ placeholder"
grep -q '@SHA256_SCRIPT@' "$root/packaging/aur/PKGBUILD.in" || \
    fail "PKGBUILD.in is missing the @SHA256_SCRIPT@ placeholder"
grep -q '@SHA256_LICENSE@' "$root/packaging/aur/PKGBUILD.in" || \
    fail "PKGBUILD.in is missing the @SHA256_LICENSE@ placeholder"

if "$hf_bash" "$root/packaging/aur/update.sh" not-a-tag >/dev/null 2>&1; then
    fail "update.sh should reject an invalid tag"
fi

# ---------------------------------------------------------------------------

printf '%s\n' "----"

if ((failures > 0)); then
    printf '%s test(s) failed\n' "$failures"
    exit 1
fi

printf 'all smoke tests passed\n'
