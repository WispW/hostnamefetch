#!/usr/bin/env bash
# shellcheck disable=SC1091,SC2034
# SC1091: the sourced path is computed from $0.
# SC2034: test globals are consumed by the sourced functions.
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
# CLI smoke tests
# ---------------------------------------------------------------------------

bash "$root/hostnamefetch" --version >/dev/null || fail "--version failed"
bash "$root/hostnamefetch" --help >/dev/null || fail "--help failed"

bash "$root/hostnamefetch" --stdout | grep -q '^Hostname: ' || \
    fail "--stdout output is missing Hostname"

if bash "$root/hostnamefetch" --layout nope >/dev/null 2>&1; then
    fail "--layout nope should fail"
fi

if bash "$root/hostnamefetch" --banner_colors 300 >/dev/null 2>&1; then
    fail "--banner_colors 300 should fail"
fi

# ---------------------------------------------------------------------------
# Long hostname integration (also runs under Git Bash on Windows)
# ---------------------------------------------------------------------------

out=$(COLUMNS=80 bash "$root/hostnamefetch" \
    --banner_text "$long_name" \
    --banner_chars ascii \
    --color_text off \
    --disable cols)

over=$(printf '%s\n' "$out" | \
    awk 'length($0) > 79 { n++ } END { print n + 0 }')
((over == 0)) || fail "top layout overflows 80 columns ($over line(s))"

out=$(COLUMNS=80 bash "$root/hostnamefetch" \
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

printf '%s\n' "----"

if ((failures > 0)); then
    printf '%s test(s) failed\n' "$failures"
    exit 1
fi

printf 'all smoke tests passed\n'
