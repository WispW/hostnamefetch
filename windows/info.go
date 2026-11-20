package main

import (
	"strconv"
	"strings"
)

// systemInfo is the platform-neutral snapshot the renderer works with. It is
// filled by a collector: the Windows one on Windows, a fixture elsewhere.
type systemInfo struct {
	User     string
	Hostname string
	FQDN     string

	Distro   string
	Model    string
	Kernel   string
	Uptime   string
	Packages string
	Shell    string

	Resolution string
	DE         string
	WM         string
	Term       string

	CPU     string
	GPU     string
	Memory  string
	LocalIP string
	Disk    string
}

type row struct {
	text  string
	width int
}

func isDisabled(disable []string, name string) bool {
	for _, item := range disable {
		if item == name {
			return true
		}
		switch name {
		case "hostname":
			if item == "host" {
				return true
			}
		case "local_ip":
			if item == "ip" {
				return true
			}
		case "distro":
			if item == "os" {
				return true
			}
		case "model":
			if item == "machine" {
				return true
			}
		}
	}
	return false
}

// trimValue reproduces the bash trim(): drop CR, turn newlines into spaces,
// collapse runs of spaces and trim both ends.
func trimValue(value string) string {
	value = strings.ReplaceAll(value, "\r", "")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.Join(strings.Fields(value), " ")
	return value
}

func buildRows(cfg *config, info *systemInfo, th theme) []row {
	var rows []row

	user := info.User
	if user == "" {
		user = "user"
	}

	displayHost := info.Hostname
	if cfg.titleFQDN == "on" && info.FQDN != "" {
		displayHost = info.FQDN
	}

	titlePlain := user + "@" + displayHost

	if !isDisabled(cfg.disable, "title") {
		rows = append(rows, row{
			text:  th.title + th.bold + user + th.at + "@" + th.title + th.bold + displayHost + th.reset,
			width: len([]rune(titlePlain)),
		})
	}

	// The underline belongs to the title; hide it with the title.
	if !isDisabled(cfg.disable, "title") && !isDisabled(cfg.disable, "underline") {
		if n := len([]rune(titlePlain)); n > 0 {
			rows = append(rows, row{
				text:  th.underline + strings.Repeat(underlineChar, n) + th.reset,
				width: n,
			})
		}
	}

	add := func(label, key, value string) {
		if isDisabled(cfg.disable, key) {
			return
		}
		value = trimValue(value)
		if value == "" {
			return
		}
		rows = append(rows, row{
			text: th.subtitle + th.bold + label + th.reset +
				th.colon + cfg.separator + th.reset + " " +
				th.info + value + th.reset,
			width: len([]rune(label)) + len([]rune(cfg.separator)) + 1 + len([]rune(value)),
		})
	}

	add("Hostname", "hostname", info.Hostname)
	add("FQDN", "fqdn", info.FQDN)

	add("OS", "distro", info.Distro)
	add("Machine", "model", info.Model)
	add("Kernel", "kernel", info.Kernel)
	add("Uptime", "uptime", info.Uptime)
	add("Packages", "packages", info.Packages)
	add("Shell", "shell", info.Shell)

	add("Resolution", "resolution", info.Resolution)
	add("DE", "de", info.DE)
	add("WM", "wm", info.WM)
	add("Terminal", "term", info.Term)

	add("CPU", "cpu", info.CPU)
	add("GPU", "gpu", info.GPU)
	add("Memory", "memory", info.Memory)

	add("Local IP", "local_ip", info.LocalIP)
	add("Disk (/)", "disk", info.Disk)

	rows = append(rows, colorBlocks(cfg, th)...)

	return rows
}

// colorBlocks renders the 16-color palette shown at the end of the info column.
func colorBlocks(cfg *config, th theme) []row {
	if !cfg.colorsEnabled() || cfg.colorBlocks != "on" || isDisabled(cfg.disable, "cols") {
		return nil
	}

	var low, high []string

	for i := cfg.blockRange[0]; i <= cfg.blockRange[1]; i++ {
		spaces := strings.Repeat(" ", cfg.blockWidth)
		if i <= 7 {
			low = append(low, "\x1b[4"+strconv.Itoa(i)+"m"+spaces)
		} else {
			high = append(high, "\x1b[48;5;"+strconv.Itoa(i)+"m"+spaces)
		}
	}

	if len(low) == 0 && len(high) == 0 {
		return nil
	}

	var rows []row
	rows = append(rows, row{})

	lowText := strings.Join(low, "")
	highText := strings.Join(high, "")

	for i := 0; i < cfg.blockHeight; i++ {
		if lowText != "" {
			rows = append(rows, row{text: lowText + th.reset, width: len(low) * cfg.blockWidth})
		}
		if highText != "" {
			rows = append(rows, row{text: highText + th.reset, width: len(high) * cfg.blockWidth})
		}
	}

	return rows
}
