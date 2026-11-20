package main

import (
	"fmt"
	"strings"
)

const underlineChar = "-"

// environment captures the runtime facts the layout depends on. Tests set it
// explicitly so output can be compared with the bash implementation.
type environment struct {
	isTTY     bool
	termWidth int
	noColor   bool
	blocks    bool // the output charset can render the block character
}

// render implements the same pass as main() in the bash script: settle the
// layout, wrap the banner, then interleave banner and info rows.
func render(cfg *config, info *systemInfo, env environment) (string, error) {
	if cfg.stdout {
		cfg.colorText = "off"
		cfg.banner = "off"
		cfg.layout = "top"
	}

	// No escape sequences when stdout is not a terminal.
	if !env.isTTY {
		cfg.colorText = "off"
		if cfg.layout == "auto" {
			cfg.layout = "top"
		}
	}

	if env.noColor {
		cfg.colorText = "off"
	}

	if env.termWidth < 1 {
		env.termWidth = 80
	}

	termWidth := env.termWidth

	bannerRaw := cfg.bannerText
	if bannerRaw == "" {
		bannerRaw = info.Hostname
	}
	bannerRaw = trimValue(bannerRaw)
	if bannerRaw == "" {
		bannerRaw = "hostname"
	}
	bannerUpper := strings.ToUpper(bannerRaw)

	th := newTheme(cfg, bannerUpper)

	fill := "#"
	switch cfg.bannerChars {
	case "ascii":
		fill = "#"
	case "blocks":
		fill = "█"
	default:
		if env.blocks {
			fill = "█"
		}
	}

	rows := buildRows(cfg, info, th)

	contentWidth := 0
	for _, r := range rows {
		if r.width > contentWidth {
			contentWidth = r.width
		}
	}

	bannerFullWidth := 0
	if cfg.banner == "on" {
		if n := len([]rune(bannerUpper)); n > 0 {
			bannerFullWidth = n*5 + (n-1)*cfg.bannerSpacing
		}
	}

	switch {
	case cfg.banner == "off" || cfg.stdout:
		cfg.layout = "top"
	case cfg.layout == "auto":
		if bannerFullWidth+cfg.gap+contentWidth <= termWidth-1 {
			cfg.layout = "left"
		} else {
			cfg.layout = "top"
		}
	}

	maxChars := 1

	if cfg.layout == "left" {
		avail := termWidth - 1 - cfg.gap - contentWidth
		if avail < 5 {
			warn("terminal too narrow for --layout left; falling back to top layout")
			cfg.layout = "top"
		} else {
			maxChars = (avail + cfg.bannerSpacing) / (5 + cfg.bannerSpacing)
		}
	}

	if cfg.layout == "top" {
		avail := termWidth - 1
		maxChars = (avail + cfg.bannerSpacing) / (5 + cfg.bannerSpacing)
	}

	if maxChars < 1 {
		maxChars = 1
	}

	var art banner
	if cfg.banner == "on" {
		art = renderBanner(splitChunks(bannerUpper, maxChars), cfg.bannerSpacing, fill, th.banner, th.reset)
	}

	return compose(cfg, art, rows), nil
}

func compose(cfg *config, art banner, rows []row) string {
	var out []string

	if cfg.stdout || cfg.banner == "off" {
		for _, r := range rows {
			out = append(out, r.text)
		}
		return strings.Join(out, "\n") + "\n"
	}

	if cfg.layout == "left" {
		bannerCount := len(art.colored)
		infoCount := len(rows)

		bannerOffset := 0
		infoOffset := 0

		// Center the shorter block against the taller one.
		if cfg.valign == "center" {
			if infoCount > bannerCount {
				bannerOffset = (infoCount - bannerCount) / 2
			} else if bannerCount > infoCount {
				infoOffset = (bannerCount - infoCount) / 2
			}
		}

		total := bannerCount + bannerOffset
		if infoCount+infoOffset > total {
			total = infoCount + infoOffset
		}

		for i := 0; i < total; i++ {
			line := ""
			pad := art.width + cfg.gap

			if bi := i - bannerOffset; bi >= 0 && bi < bannerCount {
				line = art.colored[bi]
				pad = art.width + cfg.gap - art.widths[bi]
			}

			if ii := i - infoOffset; ii >= 0 && ii < infoCount {
				if pad > 0 {
					line += strings.Repeat(" ", pad)
				}
				line += rows[ii].text
			}

			out = append(out, line)
		}

		return strings.Join(out, "\n") + "\n"
	}

	for _, line := range art.colored {
		out = append(out, line)
	}
	if len(art.colored) > 0 {
		out = append(out, "")
	}
	for _, r := range rows {
		out = append(out, r.text)
	}

	return strings.Join(out, "\n") + "\n"
}

func warn(message string) {
	fmt.Fprintf(stderr, "hostnamefetch: warning: %s\n", message)
}
