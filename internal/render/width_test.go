package render

import (
	"strings"
	"testing"
)

func TestDisplayWidth(t *testing.T) {
	tests := []struct {
		name string
		text string
		want int
	}{
		{name: "ascii", text: "A", want: 1},
		{name: "digit", text: "1", want: 1},
		{name: "japanese", text: "温度", want: 4},
		{name: "combining", text: "e\u0301", want: 1},
		{name: "ansi", text: "\x1b[32mGOOD\x1b[0m", want: 4},
		{name: "emoji", text: "😀", want: 2},
		{name: "unicode borders", text: "┌" + strings.Repeat("─", 98) + "┐", want: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DisplayWidth(tt.text); got != tt.want {
				t.Fatalf("DisplayWidth(%q) = %d, want %d", tt.text, got, tt.want)
			}
		})
	}
}

func TestRenderCellWidth(t *testing.T) {
	tests := []string{
		"APPLE SSD AP1024Z",
		"非常に長い日本語SSDモデル名",
		"é",
		"e\u0301",
		"温度",
		"36°C",
		"😀",
		"PCIe / NVMe",
		"\x1b[32mGOOD\x1b[0m",
		"改行付き\n不正文字列",
		"タブ付き\t文字列",
	}

	for _, text := range tests {
		got := RenderCell(text, 12, AlignCenter)
		if width := DisplayWidth(got); width != 12 {
			t.Fatalf("RenderCell(%q) width = %d, want 12; rendered %q", text, width, got)
		}
	}
}
