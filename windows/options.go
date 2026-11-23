package main

import (
	"fmt"
	"strconv"
	"strings"
)

// version is overridden at build time with
// -ldflags "-X main.version=<tag>".
var version = "2026.11.24"

type config struct {
	banner        string
	bannerText    string
	bannerColors  []int
	bannerChars   string
	bannerSpacing int
	layout        string
	valign        string
	gap           int

	colorText   string
	bold        string
	titleFQDN   string
	colorBlocks string
	blockRange  [2]int
	blockWidth  int
	blockHeight int
	separator   string

	disable []string
	stdout  bool
}

func defaultConfig() *config {
	return &config{
		banner:        "on",
		bannerChars:   "auto",
		bannerSpacing: 1,
		layout:        "auto",
		valign:        "center",
		gap:           3,
		colorText:     "on",
		bold:          "on",
		titleFQDN:     "off",
		colorBlocks:   "on",
		blockRange:    [2]int{0, 15},
		blockWidth:    3,
		blockHeight:   1,
		separator:     ":",
	}
}

func (c *config) colorsEnabled() bool {
	return c.colorText == "on"
}

type action int

const (
	actionRun action = iota
	actionHelp
	actionVersion
)

const usageText = `Usage: hostnamefetch [options]

A neofetch-style system information tool whose logo is a large banner
of your hostname.

Options:
  -h, --help                 Show this help and exit.
  -v, --version              Print the version and exit.

  --banner <on|off>          Show the hostname banner. Default: on.
  --banner_text <text>       Render <text> instead of the hostname.
  --banner_colors <c> [c2]   Banner/label ANSI colors (0-255).
                             Default: auto (derived from the hostname).
  --banner_chars <mode>      auto, blocks or ascii. Default: auto.
  --banner_spacing <0|1>     Space between banner glyphs. Default: 1.
  --layout <mode>            auto, left or top. Default: auto.
  --valign <mode>            top or center. Default: center.
  --gap <num>                Spaces between banner and info. Default: 3.

  --disable <item> [item...] Hide info items. Item names are the variable
                             names used internally: distro, model, kernel,
                             uptime, packages, shell, resolution, de, wm,
                             term, cpu, gpu, memory, local_ip, disk, cols,
                             title, underline, hostname, fqdn.

  --color_text <on|off>      Color the info text. Default: on.
  --bold <on|off>            Bold info text and banner. Default: on.
  --title_fqdn <on|off>      Use the FQDN in the title. Default: off.
  --color_blocks <on|off>    Show the color palette. Default: on.
  --block_range <a> <b>      Color block range. Default: 0 15.
  --block_width <num>        Color block width. Default: 3.
  --block_height <num>       Color block height. Default: 1.
  --separator <str>          Label separator. Default: ":".
  --stdout                   Print plain info only (no banner, no color).

Environment:
  NO_COLOR                   Disable colored output.
`

// parseArgs mirrors the bash get_args() contract, including its error
// messages and the "consume values until the next option" behavior.
func parseArgs(args []string) (*config, action, error) {
	cfg := defaultConfig()

	i := 0
	next := func(what string) (string, error) {
		if i+1 >= len(args) {
			return "", fmt.Errorf("%s requires a value", what)
		}
		i++
		return args[i], nil
	}

	for ; i < len(args); i++ {
		arg := args[i]

		switch arg {
		case "-h", "--help":
			return cfg, actionHelp, nil
		case "-v", "--version":
			return cfg, actionVersion, nil

		case "--banner":
			value, err := next(arg)
			if err != nil {
				return nil, actionRun, err
			}
			if err := checkOnOff(value); err != nil {
				return nil, actionRun, err
			}
			cfg.banner = value

		case "--banner_text":
			value, err := next(arg)
			if err != nil {
				return nil, actionRun, err
			}
			cfg.bannerText = value

		case "--banner_colors":
			cfg.bannerColors = nil
			for i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") && len(cfg.bannerColors) < 2 {
				i++
				n, err := parseColor(args[i])
				if err != nil {
					return nil, actionRun, err
				}
				cfg.bannerColors = append(cfg.bannerColors, n)
			}
			if len(cfg.bannerColors) == 0 {
				return nil, actionRun, fmt.Errorf("--banner_colors needs at least one color")
			}

		case "--banner_chars":
			value, err := next(arg)
			if err != nil {
				return nil, actionRun, err
			}
			switch value {
			case "auto", "blocks", "ascii":
				cfg.bannerChars = value
			default:
				return nil, actionRun, fmt.Errorf("invalid --banner_chars '%s' (auto, blocks, ascii)", value)
			}

		case "--banner_spacing":
			value, err := next(arg)
			if err != nil {
				return nil, actionRun, err
			}
			n, err := parseNumber(value)
			if err != nil {
				return nil, actionRun, err
			}
			cfg.bannerSpacing = n

		case "--layout":
			value, err := next(arg)
			if err != nil {
				return nil, actionRun, err
			}
			switch value {
			case "auto", "left", "top":
				cfg.layout = value
			default:
				return nil, actionRun, fmt.Errorf("invalid --layout '%s' (auto, left, top)", value)
			}

		case "--valign":
			value, err := next(arg)
			if err != nil {
				return nil, actionRun, err
			}
			switch value {
			case "top", "center":
				cfg.valign = value
			default:
				return nil, actionRun, fmt.Errorf("invalid --valign '%s' (top, center)", value)
			}

		case "--gap":
			value, err := next(arg)
			if err != nil {
				return nil, actionRun, err
			}
			n, err := parseNumber(value)
			if err != nil {
				return nil, actionRun, err
			}
			cfg.gap = n

		case "--disable":
			found := false
			for i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				item := normalizeName(args[i])
				if validDisableName(item) {
					cfg.disable = append(cfg.disable, item)
				} else {
					warn(fmt.Sprintf("unknown --disable item '%s' (ignored)", args[i]))
				}
				found = true
			}
			if !found {
				return nil, actionRun, fmt.Errorf("--disable needs at least one item")
			}

		case "--color_text":
			value, err := next(arg)
			if err != nil {
				return nil, actionRun, err
			}
			if err := checkOnOff(value); err != nil {
				return nil, actionRun, err
			}
			cfg.colorText = value

		case "--bold":
			value, err := next(arg)
			if err != nil {
				return nil, actionRun, err
			}
			if err := checkOnOff(value); err != nil {
				return nil, actionRun, err
			}
			cfg.bold = value

		case "--title_fqdn":
			value, err := next(arg)
			if err != nil {
				return nil, actionRun, err
			}
			if err := checkOnOff(value); err != nil {
				return nil, actionRun, err
			}
			cfg.titleFQDN = value

		case "--color_blocks":
			value, err := next(arg)
			if err != nil {
				return nil, actionRun, err
			}
			if err := checkOnOff(value); err != nil {
				return nil, actionRun, err
			}
			cfg.colorBlocks = value

		case "--block_range":
			start, err := next(arg)
			if err != nil {
				return nil, actionRun, err
			}
			end, err := next(arg)
			if err != nil {
				return nil, actionRun, err
			}
			a, err := parseNumber(start)
			if err != nil {
				return nil, actionRun, err
			}
			b, err := parseNumber(end)
			if err != nil {
				return nil, actionRun, err
			}
			cfg.blockRange = [2]int{a, b}

		case "--block_width":
			value, err := next(arg)
			if err != nil {
				return nil, actionRun, err
			}
			n, err := parseNumber(value)
			if err != nil {
				return nil, actionRun, err
			}
			cfg.blockWidth = n

		case "--block_height":
			value, err := next(arg)
			if err != nil {
				return nil, actionRun, err
			}
			n, err := parseNumber(value)
			if err != nil {
				return nil, actionRun, err
			}
			cfg.blockHeight = n

		case "--separator":
			value, err := next(arg)
			if err != nil {
				return nil, actionRun, err
			}
			cfg.separator = value

		case "--stdout":
			cfg.stdout = true

		default:
			return nil, actionRun, fmt.Errorf("unknown option '%s' (try --help)", arg)
		}
	}

	if cfg.blockWidth <= 0 {
		return nil, actionRun, fmt.Errorf("--block_width must be greater than 0")
	}
	if cfg.blockHeight <= 0 {
		return nil, actionRun, fmt.Errorf("--block_height must be greater than 0")
	}
	if cfg.blockRange[0] > cfg.blockRange[1] {
		return nil, actionRun, fmt.Errorf("--block_range start must be <= end")
	}
	if cfg.blockRange[0] > 255 || cfg.blockRange[1] > 255 {
		return nil, actionRun, fmt.Errorf("--block_range values must be between 0 and 255")
	}
	if cfg.bannerSpacing != 0 && cfg.bannerSpacing != 1 {
		return nil, actionRun, fmt.Errorf("--banner_spacing must be 0 or 1")
	}

	return cfg, actionRun, nil
}

func parseNumber(value string) (int, error) {
	n, err := strconv.Atoi(value)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("expected a number, got '%s'", value)
	}
	return n, nil
}

func parseColor(value string) (int, error) {
	n, err := strconv.Atoi(value)
	if err != nil || n < 0 || n > 255 {
		return 0, fmt.Errorf("invalid color '%s' (expected 0-255)", value)
	}
	return n, nil
}

func checkOnOff(value string) error {
	if value != "on" && value != "off" {
		return fmt.Errorf("expected 'on' or 'off', got '%s'", value)
	}
	return nil
}

func normalizeName(value string) string {
	value = strings.ToLower(value)
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	return value
}

func validDisableName(name string) bool {
	switch name {
	case "title", "underline", "hostname", "fqdn", "distro", "model", "kernel",
		"uptime", "packages", "shell", "resolution", "de", "wm", "term", "cpu",
		"gpu", "memory", "local_ip", "disk", "cols":
		return true
	case "host", "os", "machine", "ip":
		return true
	}
	return false
}
