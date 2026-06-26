package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"disk-smi/internal/health"
	"disk-smi/internal/model"
	"disk-smi/internal/smartctl"
)

func TestGoldenOutputs(t *testing.T) {
	snapshot := parseGoldenFixture(t)
	tests := []struct {
		name   string
		opts   Options
		golden string
	}{
		{name: "en unicode", opts: Options{Width: 100, Locale: LocaleEnglish}, golden: "en-100-unicode.txt"},
		{name: "en ascii", opts: Options{Width: 100, Locale: LocaleEnglish, ASCII: true}, golden: "en-100-ascii.txt"},
		{name: "ja unicode", opts: Options{Width: 100, Locale: LocaleJapanese}, golden: "ja-100-unicode.txt"},
		{name: "ja ascii", opts: Options{Width: 100, Locale: LocaleJapanese, ASCII: true}, golden: "ja-100-ascii.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RenderDrive(snapshot, tt.opts)
			if err != nil {
				t.Fatal(err)
			}
			want, err := os.ReadFile(filepath.Join("..", "..", "testdata", "golden", tt.golden))
			if err != nil {
				t.Fatal(err)
			}
			if got != string(want) {
				t.Fatalf("golden mismatch for %s", tt.golden)
			}
			for i, line := range strings.Split(strings.TrimSuffix(got, "\n"), "\n") {
				if width := DisplayWidth(line); width != 100 {
					t.Fatalf("line %d width = %d, want 100: %q", i+1, width, line)
				}
			}
		})
	}
}

func parseGoldenFixture(t *testing.T) model.DriveSnapshot {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "smartctl", "nvme-good.json"))
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := smartctl.Parse(data, "")
	if err != nil {
		t.Fatal(err)
	}
	snapshot.Assessment = health.Assess(snapshot.Metrics)
	return snapshot
}
