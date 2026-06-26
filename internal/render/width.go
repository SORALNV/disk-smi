package render

import (
	"regexp"
	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/rivo/uniseg"
)

type Alignment int

const (
	AlignLeft Alignment = iota
	AlignRight
	AlignCenter
)

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)

func StripANSI(text string) string {
	return ansiPattern.ReplaceAllString(text, "")
}

func DisplayWidth(text string) int {
	return runewidth.StringWidth(StripANSI(text))
}

func SanitizeTerminalText(text string) string {
	text = strings.ReplaceAll(text, "\r", " ")
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\t", " ")
	return text
}

func PadLeft(text string, width int) string {
	text = truncateDisplay(SanitizeTerminalText(text), width)
	padding := width - DisplayWidth(text)
	if padding < 0 {
		padding = 0
	}
	return strings.Repeat(" ", padding) + text
}

func PadRight(text string, width int) string {
	text = truncateDisplay(SanitizeTerminalText(text), width)
	padding := width - DisplayWidth(text)
	if padding < 0 {
		padding = 0
	}
	return text + strings.Repeat(" ", padding)
}

func PadCenter(text string, width int) string {
	text = truncateDisplay(SanitizeTerminalText(text), width)
	padding := width - DisplayWidth(text)
	if padding < 0 {
		padding = 0
	}
	left := padding / 2
	right := padding - left
	return strings.Repeat(" ", left) + text + strings.Repeat(" ", right)
}

func RenderCell(text string, width int, alignment Alignment) string {
	switch alignment {
	case AlignRight:
		return PadLeft(text, width)
	case AlignCenter:
		return PadCenter(text, width)
	default:
		return PadRight(text, width)
	}
}

func truncateDisplay(text string, width int) string {
	if width <= 0 {
		return ""
	}
	if DisplayWidth(text) <= width {
		return text
	}
	if width <= 3 {
		return strings.Repeat(".", width)
	}

	limit := width - 3
	var builder strings.Builder
	current := 0
	graphemes := uniseg.NewGraphemes(text)
	for graphemes.Next() {
		cluster := graphemes.Str()
		clusterWidth := runewidth.StringWidth(StripANSI(cluster))
		if current+clusterWidth > limit {
			break
		}
		builder.WriteString(cluster)
		current += clusterWidth
	}
	return builder.String() + "..."
}
