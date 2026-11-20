package main

import (
	"strings"
	"testing"
)

func TestPosixCksum(t *testing.T) {
	// Values produced by `printf '%s' <text> | cksum` on Linux.
	cases := []struct {
		text string
		want uint32
	}{
		{"", 4294967295},
		{"W", 2649123693},
		{"TEST", 1164413914},
		{"MAIN-PC", 1834687110},
		{"ALPHA", 2314623003},
		{"ZETA-9", 776109389},
		{"Z3-PRO", 281869633},
	}

	for _, tc := range cases {
		if got := posixCksum([]byte(tc.text)); got != tc.want {
			t.Errorf("posixCksum(%q) = %d, want %d", tc.text, got, tc.want)
		}
	}
}

func TestResolvePaletteAuto(t *testing.T) {
	cfg := defaultConfig()

	// MAIN-PC: 1834687110 % 12 = 6 -> red / cyan.
	if got := resolvePalette(cfg, "MAIN-PC"); got != [2]int{1, 6} {
		t.Errorf("resolvePalette(MAIN-PC) = %v, want [1 6]", got)
	}

	// W: 2649123693 % 12 = 9 -> yellow / cyan.
	if got := resolvePalette(cfg, "W"); got != [2]int{3, 6} {
		t.Errorf("resolvePalette(W) = %v, want [3 6]", got)
	}
}

func TestResolvePaletteExplicit(t *testing.T) {
	cfg := defaultConfig()
	cfg.bannerColors = []int{9}

	if got := resolvePalette(cfg, "ANYTHING"); got != [2]int{9, 9} {
		t.Errorf("single color should mirror itself, got %v", got)
	}

	cfg.bannerColors = []int{9, 12}
	if got := resolvePalette(cfg, "ANYTHING"); got != [2]int{9, 12} {
		t.Errorf("explicit pair not honored, got %v", got)
	}
}

func TestAnsiColor(t *testing.T) {
	cases := map[int]string{
		0:   "\x1b[0m\x1b[30m",
		6:   "\x1b[0m\x1b[36m",
		7:   "\x1b[0m\x1b[37m",
		8:   "\x1b[38;5;8m",
		196: "\x1b[38;5;196m",
	}

	for input, want := range cases {
		if got := ansiColor(input, "\x1b[0m"); got != want {
			t.Errorf("ansiColor(%d) = %q, want %q", input, got, want)
		}
	}
}

func TestSplitChunks(t *testing.T) {
	cases := []struct {
		text string
		max  int
		want []string
	}{
		{"TEST", 5, []string{"TEST"}},
		{"ABCDE", 2, []string{"AB", "CD", "E"}},
		{"MAIN-PC", 3, []string{"MAI", "N-", "PC"}},
		{"a-b-c-d", 4, []string{"a-b-", "c-d"}},
		{"very-long-name", 6, []string{"very-", "long-", "name"}},
		{"", 5, []string{""}},
		{"AB", 0, []string{"A", "B"}},
	}

	for _, tc := range cases {
		got := splitChunks(tc.text, tc.max)
		if strings.Join(got, "|") != strings.Join(tc.want, "|") {
			t.Errorf("splitChunks(%q, %d) = %v, want %v", tc.text, tc.max, got, tc.want)
		}
	}
}

func TestRenderBannerAscii(t *testing.T) {
	art := renderBanner([]string{"TEST"}, 1, "#", "", "")

	want := []string{
		"##### #####  #### #####",
		"  #   #     #       #  ",
		"  #   ####   ###    #  ",
		"  #   #         #   #  ",
		"  #   ##### ####    #  ",
	}

	if len(art.lines) != 5 {
		t.Fatalf("expected 5 rows, got %d", len(art.lines))
	}
	for i, line := range want {
		if art.lines[i] != strings.TrimRight(line, " ") {
			t.Errorf("row %d = %q, want %q", i, art.lines[i], strings.TrimRight(line, " "))
		}
	}

	if art.width != 23 {
		t.Errorf("banner width = %d, want 23", art.width)
	}
}

func TestGlyphFallback(t *testing.T) {
	if glyphFor('ß') != glyphFor('?') {
		t.Error("unknown runes should fall back to the ? glyph")
	}
}
