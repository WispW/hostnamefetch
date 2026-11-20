package main

import (
	"strings"
	"testing"
)

func TestParseDefaults(t *testing.T) {
	cfg, act, err := parseArgs(nil)
	if err != nil {
		t.Fatalf("parseArgs(nil) error: %v", err)
	}
	if act != actionRun {
		t.Fatalf("action = %v, want actionRun", act)
	}

	if cfg.banner != "on" || cfg.layout != "auto" || cfg.valign != "center" {
		t.Errorf("unexpected banner/layout/valign: %q %q %q", cfg.banner, cfg.layout, cfg.valign)
	}
	if cfg.gap != 3 || cfg.bannerSpacing != 1 {
		t.Errorf("unexpected gap/spacing: %d %d", cfg.gap, cfg.bannerSpacing)
	}
	if cfg.blockRange != [2]int{0, 15} || cfg.blockWidth != 3 || cfg.blockHeight != 1 {
		t.Errorf("unexpected block defaults: %v %d %d", cfg.blockRange, cfg.blockWidth, cfg.blockHeight)
	}
	if cfg.separator != ":" || cfg.stdout {
		t.Errorf("unexpected separator/stdout: %q %v", cfg.separator, cfg.stdout)
	}
}

func TestParseActions(t *testing.T) {
	if _, act, err := parseArgs([]string{"--help"}); err != nil || act != actionHelp {
		t.Errorf("--help: act=%v err=%v", act, err)
	}
	if _, act, err := parseArgs([]string{"-v"}); err != nil || act != actionVersion {
		t.Errorf("-v: act=%v err=%v", act, err)
	}
}

func TestParseValues(t *testing.T) {
	cfg, _, err := parseArgs([]string{
		"--banner", "off",
		"--banner_text", "Test Machine",
		"--banner_colors", "12", "34",
		"--banner_chars", "ascii",
		"--banner_spacing", "0",
		"--layout", "left",
		"--valign", "top",
		"--gap", "0",
		"--color_text", "off",
		"--bold", "off",
		"--title_fqdn", "on",
		"--color_blocks", "off",
		"--block_range", "0", "7",
		"--block_width", "5",
		"--block_height", "2",
		"--separator", "=>",
		"--disable", "kernel", "local-ip",
		"--stdout",
	})
	if err != nil {
		t.Fatalf("parseArgs error: %v", err)
	}

	if cfg.banner != "off" || cfg.bannerText != "Test Machine" {
		t.Errorf("banner fields: %q %q", cfg.banner, cfg.bannerText)
	}
	if len(cfg.bannerColors) != 2 || cfg.bannerColors[0] != 12 || cfg.bannerColors[1] != 34 {
		t.Errorf("banner colors: %v", cfg.bannerColors)
	}
	if cfg.bannerChars != "ascii" || cfg.bannerSpacing != 0 {
		t.Errorf("chars/spacing: %q %d", cfg.bannerChars, cfg.bannerSpacing)
	}
	if cfg.layout != "left" || cfg.valign != "top" || cfg.gap != 0 {
		t.Errorf("layout fields: %q %q %d", cfg.layout, cfg.valign, cfg.gap)
	}
	if cfg.colorText != "off" || cfg.bold != "off" || cfg.titleFQDN != "on" || cfg.colorBlocks != "off" {
		t.Error("on/off fields not parsed")
	}
	if cfg.blockRange != [2]int{0, 7} || cfg.blockWidth != 5 || cfg.blockHeight != 2 {
		t.Error("block fields not parsed")
	}
	if cfg.separator != "=>" {
		t.Errorf("separator = %q", cfg.separator)
	}
	if len(cfg.disable) != 2 || cfg.disable[0] != "kernel" || cfg.disable[1] != "local_ip" {
		t.Errorf("disable = %v", cfg.disable)
	}
	if !cfg.stdout {
		t.Error("--stdout not applied")
	}
}

func TestParseErrors(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"--bad"}, "unknown option '--bad'"},
		{[]string{"--banner"}, "--banner requires a value"},
		{[]string{"--banner", "maybe"}, "expected 'on' or 'off'"},
		{[]string{"--banner_chars", "dots"}, "invalid --banner_chars"},
		{[]string{"--layout", "right"}, "invalid --layout"},
		{[]string{"--valign", "bottom"}, "invalid --valign"},
		{[]string{"--gap", "x"}, "expected a number"},
		{[]string{"--banner_colors"}, "needs at least one color"},
		{[]string{"--banner_colors", "300"}, "invalid color"},
		{[]string{"--disable"}, "needs at least one item"},
		{[]string{"--block_range", "9", "3"}, "start must be <= end"},
		{[]string{"--block_range", "0", "300"}, "between 0 and 255"},
		{[]string{"--block_width", "0"}, "must be greater than 0"},
		{[]string{"--block_height", "0"}, "must be greater than 0"},
		{[]string{"--banner_spacing", "2"}, "must be 0 or 1"},
	}

	for _, tc := range cases {
		_, _, err := parseArgs(tc.args)
		if err == nil {
			t.Errorf("parseArgs(%v) = nil error, want %q", tc.args, tc.want)
			continue
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("parseArgs(%v) error = %q, want it to contain %q", tc.args, err, tc.want)
		}
	}
}

func TestDisableAliases(t *testing.T) {
	cases := []struct {
		item string
		name string
	}{
		{"host", "hostname"},
		{"ip", "local_ip"},
		{"os", "distro"},
		{"machine", "model"},
	}

	for _, tc := range cases {
		if !isDisabled([]string{tc.item}, tc.name) {
			t.Errorf("alias %q should disable %q", tc.item, tc.name)
		}
		if isDisabled([]string{tc.item}, "kernel") {
			t.Errorf("alias %q should not disable kernel", tc.item)
		}
	}
}

func TestUnknownDisableItemIsIgnored(t *testing.T) {
	cfg, _, err := parseArgs([]string{"--disable", "kernel", "nonsense"})
	if err != nil {
		t.Fatalf("parseArgs error: %v", err)
	}
	if len(cfg.disable) != 1 || cfg.disable[0] != "kernel" {
		t.Errorf("disable = %v, want only kernel", cfg.disable)
	}
}
