package main

import "strings"

// splitChunks breaks the banner text into chunks that fit maxChars glyphs.
// Like the bash implementation it prefers to break after - . _ + / so the
// banner stays readable.
func splitChunks(text string, maxChars int) []string {
	if maxChars < 1 {
		maxChars = 1
	}

	runes := []rune(text)
	var chunks []string

	for len(runes) > maxChars {
		head := runes[:maxChars]
		cut := maxChars

		// Prefer breaking after a separator so the banner stays readable.
		for i := len(head) - 1; i > 0; i-- {
			switch head[i] {
			case '-', '.', '_', '+', '/':
				cut = i + 1
			default:
				continue
			}
			break
		}

		chunks = append(chunks, string(runes[:cut]))
		runes = runes[cut:]
	}

	return append(chunks, string(runes))
}

type banner struct {
	lines   []string // trimmed, fill applied, no colors
	widths  []int    // visible width of each trimmed line
	colored []string // lines wrapped in the banner color
	width   int      // widest untrimmed line
}

// renderBanner turns the chunks into block rows. fill is the character that
// replaces "#" in the font (a full block or an ASCII "#").
func renderBanner(chunks []string, spacing int, fill, color, reset string) banner {
	var out banner

	for idx, chunk := range chunks {
		if idx > 0 {
			out.lines = append(out.lines, "")
			out.widths = append(out.widths, 0)
		}

		var rows [5]string
		chars := []rune(strings.ToUpper(chunk))

		for i, ch := range chars {
			g := glyphFor(ch)
			for row := 0; row < 5; row++ {
				rows[row] += g[row]
				if i < len(chars)-1 && spacing == 1 {
					rows[row] += " "
				}
			}
		}

		for row := 0; row < 5; row++ {
			line := strings.ReplaceAll(rows[row], "#", fill)

			// Width is measured before trimming, with the fill character
			// counting as one column.
			if width := len([]rune(line)); width > out.width {
				out.width = width
			}

			line = strings.TrimRight(line, " ")
			out.lines = append(out.lines, line)
			out.widths = append(out.widths, len([]rune(line)))
		}
	}

	for _, line := range out.lines {
		if line == "" {
			out.colored = append(out.colored, "")
			continue
		}
		out.colored = append(out.colored, color+line+reset)
	}

	return out
}
