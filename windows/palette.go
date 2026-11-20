package main

import "strconv"

// bannerPalettes are used when --banner_colors is left on "auto". The pair is
// selected with a checksum of the banner text, so a given machine always gets
// the same colors. The order must match the bash implementation.
var bannerPalettes = [][2]int{
	{4, 6}, // blue / cyan
	{6, 4}, // cyan / blue
	{2, 6}, // green / cyan
	{5, 6}, // magenta / cyan
	{4, 5}, // blue / magenta
	{3, 4}, // yellow / blue
	{1, 6}, // red / cyan
	{6, 2}, // cyan / green
	{5, 4}, // magenta / blue
	{3, 6}, // yellow / cyan
	{2, 4}, // green / blue
	{1, 5}, // red / magenta
}

// posixCksum returns the same CRC value as POSIX `cksum`, which is what the
// bash implementation uses to pick a palette.
func posixCksum(data []byte) uint32 {
	crc := uint32(0)

	feed := func(b byte) {
		crc = (crc << 8) ^ crcTable[byte(crc>>24)^b]
	}

	for _, b := range data {
		feed(b)
	}

	// POSIX appends the length, least significant byte first.
	length := uint64(len(data))
	for length != 0 {
		feed(byte(length))
		length >>= 8
	}

	return ^crc
}

// crcTable is the MSB-first CRC-32 table used by POSIX cksum
// (polynomial 0x04C11DB7).
var crcTable = func() [256]uint32 {
	var table [256]uint32
	for i := 0; i < 256; i++ {
		c := uint32(i) << 24
		for j := 0; j < 8; j++ {
			if c&0x80000000 != 0 {
				c = (c << 1) ^ 0x04C11DB7
			} else {
				c <<= 1
			}
		}
		table[i] = c
	}
	return table
}()

// ansiColor renders an ANSI 256-color foreground code the same way the bash
// color() helper does, including the leading reset.
func ansiColor(n int, reset string) string {
	switch {
	case n >= 0 && n <= 6:
		return reset + "\x1b[3" + strconv.Itoa(n) + "m"
	case n == 7:
		return reset + "\x1b[37m"
	default:
		return "\x1b[38;5;" + strconv.Itoa(n) + "m"
	}
}

// theme holds the escape sequences used by the renderer. When colors are
// disabled every field is empty so output stays plain.
type theme struct {
	reset     string
	bold      string
	banner    string
	title     string
	at        string
	underline string
	subtitle  string
	colon     string
	info      string
}

func newTheme(cfg *config, hostnameUpper string) theme {
	if !cfg.colorsEnabled() {
		return theme{}
	}

	palette := resolvePalette(cfg, hostnameUpper)

	reset := "\x1b[0m"
	bold := ""
	if cfg.bold == "on" {
		bold = "\x1b[1m"
	}

	return theme{
		reset:     reset,
		bold:      bold,
		banner:    ansiColor(palette[0], reset) + bold,
		title:     ansiColor(palette[0], reset),
		at:        reset,
		underline: ansiColor(palette[1], reset),
		subtitle:  ansiColor(palette[1], reset),
		colon:     reset,
		info:      reset,
	}
}

func resolvePalette(cfg *config, hostnameUpper string) [2]int {
	if len(cfg.bannerColors) > 0 {
		first := cfg.bannerColors[0]
		second := first
		if len(cfg.bannerColors) > 1 {
			second = cfg.bannerColors[1]
		}
		return [2]int{first, second}
	}

	sum := posixCksum([]byte(hostnameUpper))
	return bannerPalettes[int(sum%uint32(len(bannerPalettes)))]
}
