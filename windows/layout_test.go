package main

import (
	"strings"
	"testing"
)

func testInfo() *systemInfo {
	return &systemInfo{
		User:     "qq341",
		Hostname: "Z3-PRO",
		Distro:   "Windows 10.0.19045 x86_64",
		Kernel:   "10.0.19045",
		Uptime:   "1 hour, 2 mins",
		Shell:    "cmd",
		Memory:   "100MiB / 200MiB (50%)",
	}
}

func testEnv() environment {
	return environment{isTTY: false, termWidth: 120}
}

func renderFor(t *testing.T, args []string, info *systemInfo, env environment) []string {
	t.Helper()

	cfg, _, err := parseArgs(args)
	if err != nil {
		t.Fatalf("parseArgs(%v): %v", args, err)
	}

	out, err := render(cfg, info, env)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	return strings.Split(strings.TrimSuffix(out, "\n"), "\n")
}

func TestRenderPlainTop(t *testing.T) {
	lines := renderFor(t, []string{
		"--color_text", "off",
		"--bold", "off",
		"--banner_chars", "ascii",
		"--layout", "top",
		"--banner_text", "TEST",
	}, testInfo(), testEnv())

	if len(lines) < 12 {
		t.Fatalf("expected banner + rows, got %d lines", len(lines))
	}

	wantBanner := []string{
		"##### #####  #### #####",
		"  #   #     #       #  ",
		"  #   ####   ###    #  ",
		"  #   #         #   #  ",
		"  #   ##### ####    #  ",
	}
	for i, want := range wantBanner {
		if lines[i] != strings.TrimRight(want, " ") {
			t.Errorf("banner row %d = %q, want %q", i, lines[i], strings.TrimRight(want, " "))
		}
	}

	if lines[5] != "" {
		t.Errorf("expected a blank line after the banner, got %q", lines[5])
	}

	wantRows := []string{
		"qq341@Z3-PRO",
		"------------",
		"Hostname: Z3-PRO",
		"OS: Windows 10.0.19045 x86_64",
		"Kernel: 10.0.19045",
		"Uptime: 1 hour, 2 mins",
		"Shell: cmd",
		"Memory: 100MiB / 200MiB (50%)",
	}
	for i, want := range wantRows {
		if lines[6+i] != want {
			t.Errorf("row %d = %q, want %q", i, lines[6+i], want)
		}
	}
}

func TestRenderLeftAlignment(t *testing.T) {
	lines := renderFor(t, []string{
		"--color_text", "off",
		"--bold", "off",
		"--banner_chars", "ascii",
		"--layout", "left",
		"--valign", "top",
		"--gap", "3",
		"--banner_text", "TEST",
	}, testInfo(), testEnv())

	const pad = 23 + 3

	if !strings.HasPrefix(lines[0], "##### ") {
		t.Fatalf("first line should be the banner, got %q", lines[0])
	}

	// Info rows share the banner lines, so every row must start at the same
	// visible column (banner width + gap).
	infoRows := []string{
		"qq341@Z3-PRO",
		"------------",
		"Hostname: Z3-PRO",
		"OS: Windows 10.0.19045 x86_64",
		"Kernel: 10.0.19045",
		"Uptime: 1 hour, 2 mins",
		"Shell: cmd",
		"Memory: 100MiB / 200MiB (50%)",
	}

	for i, want := range infoRows {
		line := lines[i]
		if len(line) <= pad {
			t.Fatalf("line %d too short for the info column: %q", i, line)
		}
		if line[pad:] != want {
			t.Errorf("line %d info column = %q, want %q", i, line[pad:], want)
		}
	}
}

func TestRenderStdoutDisablesBannerAndColor(t *testing.T) {
	lines := renderFor(t, []string{
		"--stdout",
		"--banner_text", "TEST",
	}, testInfo(), environment{isTTY: true, termWidth: 120, blocks: true})

	if strings.Contains(strings.Join(lines, "\n"), "\x1b[") {
		t.Fatal("--stdout must not emit escape sequences")
	}
	if strings.Contains(strings.Join(lines, "\n"), "#####") {
		t.Fatal("--stdout must not render the banner")
	}
	if lines[0] != "qq341@Z3-PRO" {
		t.Errorf("first row = %q", lines[0])
	}
}

func TestRenderDisableRemovesRow(t *testing.T) {
	lines := renderFor(t, []string{
		"--color_text", "off",
		"--bold", "off",
		"--stdout",
		"--disable", "kernel", "memory",
	}, testInfo(), testEnv())

	joined := strings.Join(lines, "\n")
	if strings.Contains(joined, "Kernel:") {
		t.Error("Kernel row should be hidden")
	}
	if strings.Contains(joined, "Memory:") {
		t.Error("Memory row should be hidden")
	}
	if !strings.Contains(joined, "Shell: cmd") {
		t.Error("Shell row should stay")
	}
}

func TestRenderSeparator(t *testing.T) {
	lines := renderFor(t, []string{
		"--color_text", "off",
		"--bold", "off",
		"--stdout",
		"--separator", "=>",
	}, testInfo(), testEnv())

	if lines[2] != "Hostname=> Z3-PRO" {
		t.Errorf("separator not applied: %q", lines[2])
	}
}

func TestRenderColorTitleAndReset(t *testing.T) {
	lines := renderFor(t, []string{
		"--banner_chars", "ascii",
		"--layout", "top",
		"--banner_text", "TEST",
	}, testInfo(), environment{isTTY: true, termWidth: 120, blocks: true})

	title := lines[6]
	if !strings.HasPrefix(title, "\x1b[0m") {
		t.Errorf("title should start with a reset, got %q", title)
	}
	if !strings.HasSuffix(title, "\x1b[0m") {
		t.Errorf("title should end with a reset, got %q", title)
	}
	if !strings.Contains(title, "qq341") || !strings.Contains(title, "Z3-PRO") {
		t.Errorf("title text missing: %q", title)
	}
}

func TestRenderNarrowTerminalFallsBackToTop(t *testing.T) {
	lines := renderFor(t, []string{
		"--color_text", "off",
		"--bold", "off",
		"--banner_chars", "ascii",
		"--layout", "left",
		"--banner_text", "LONG-HOSTNAME",
	}, testInfo(), environment{isTTY: false, termWidth: 30})

	if !strings.HasPrefix(lines[0], "##### ") && !strings.HasPrefix(lines[0], "#") {
		t.Fatalf("expected a top layout banner, got %q", lines[0])
	}
}
