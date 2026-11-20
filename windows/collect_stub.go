//go:build !windows

package main

import (
	"os"
	"strconv"
	"strings"
)

// collect is the non-Windows stand-in. It exists so the renderer can be built
// and unit-tested on Linux and macOS (CI runs the tests there); the real
// collector lives in collect_windows.go.
func collect() (environment, *systemInfo) {
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "unknown"
	}

	user := os.Getenv("USER")
	if user == "" {
		user = "user"
	}

	env := environment{
		isTTY:     isTerminal(os.Stdout),
		termWidth: termWidth(),
		noColor:   hasNoColor(),
		blocks:    true,
	}

	info := &systemInfo{
		User:     user,
		Hostname: hostname,
	}

	// Tests and the parity harness can load a fixture so the layout can be
	// compared with the bash implementation without a Windows machine.
	if path := os.Getenv("HOSTNAMEFETCH_FIXTURE"); path != "" {
		applyFixture(info, path)
	}

	return env, info
}

func applyFixture(info *systemInfo, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "user":
			info.User = value
		case "hostname":
			info.Hostname = value
		case "fqdn":
			info.FQDN = value
		case "distro":
			info.Distro = value
		case "model":
			info.Model = value
		case "kernel":
			info.Kernel = value
		case "uptime":
			info.Uptime = value
		case "packages":
			info.Packages = value
		case "shell":
			info.Shell = value
		case "resolution":
			info.Resolution = value
		case "de":
			info.DE = value
		case "wm":
			info.WM = value
		case "term":
			info.Term = value
		case "cpu":
			info.CPU = value
		case "gpu":
			info.GPU = value
		case "memory":
			info.Memory = value
		case "local_ip":
			info.LocalIP = value
		case "disk":
			info.Disk = value
		}
	}
}

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func termWidth() int {
	if value, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil && value > 0 {
		return value
	}
	return 80
}

func hasNoColor() bool {
	_, ok := os.LookupEnv("NO_COLOR")
	return ok
}
