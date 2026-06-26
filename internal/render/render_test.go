package render

import (
	"strings"
	"testing"

	"disk-smi/internal/model"
)

func TestRenderDriveLineWidths(t *testing.T) {
	tests := []struct {
		name   string
		locale Locale
		ascii  bool
		width  int
	}{
		{name: "en unicode 60", locale: LocaleEnglish, width: 60},
		{name: "en unicode 79", locale: LocaleEnglish, width: 79},
		{name: "en unicode 80", locale: LocaleEnglish, width: 80},
		{name: "en unicode 96", locale: LocaleEnglish, width: 96},
		{name: "en unicode 100", locale: LocaleEnglish, width: 100},
		{name: "en unicode 120", locale: LocaleEnglish, width: 120},
		{name: "en ascii 100", locale: LocaleEnglish, ascii: true, width: 100},
		{name: "ja unicode 100", locale: LocaleJapanese, width: 100},
		{name: "ja ascii 100", locale: LocaleJapanese, ascii: true, width: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rendered, err := RenderDrive(model.SyntheticSnapshot(), Options{
				Width:  tt.width,
				Locale: tt.locale,
				ASCII:  tt.ascii,
			})
			if err != nil {
				t.Fatal(err)
			}
			for i, line := range strings.Split(strings.TrimSuffix(rendered, "\n"), "\n") {
				if got := DisplayWidth(line); got != tt.width {
					t.Fatalf("line %d width = %d, want %d: %q", i+1, got, tt.width, line)
				}
			}
		})
	}
}

func TestRenderDriveJapaneseUsesJapaneseLabels(t *testing.T) {
	rendered, err := RenderDrive(model.SyntheticSnapshot(), Options{Width: 100, Locale: LocaleJapanese})
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"総合状態", "耐久残量 70%", "健康・耐久", "注記:"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("Japanese render missing %q:\n%s", want, rendered)
		}
	}
	if strings.Contains(rendered, "OVERALL STATUS") {
		t.Fatalf("Japanese render contains English label:\n%s", rendered)
	}
}

func TestRenderDriveASCIIBorders(t *testing.T) {
	rendered, err := RenderDrive(model.SyntheticSnapshot(), Options{Width: 100, Locale: LocaleEnglish, ASCII: true})
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(rendered, "┌┐└┘├┤┼│─") {
		t.Fatalf("ASCII render contains Unicode border characters:\n%s", rendered)
	}
	if !strings.HasPrefix(rendered, "+") {
		t.Fatalf("ASCII render should start with '+':\n%s", rendered)
	}
}

func TestRenderDriveShowSerial(t *testing.T) {
	rendered, err := RenderDrive(model.SyntheticSnapshot(), Options{Width: 100, Locale: LocaleEnglish, ShowSerial: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered, "SYNTHETIC9K2A") {
		t.Fatalf("full serial missing:\n%s", rendered)
	}
	if strings.Contains(rendered, "****9K2A") {
		t.Fatalf("masked serial still visible:\n%s", rendered)
	}
}

func TestRenderDriveColorMaintainsLineWidths(t *testing.T) {
	rendered, err := RenderDrive(model.SyntheticSnapshot(), Options{Width: 100, Locale: LocaleEnglish, Color: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered, "\x1b[32mGOOD\x1b[0m") {
		t.Fatalf("colored GOOD missing:\n%s", rendered)
	}
	for i, line := range strings.Split(strings.TrimSuffix(rendered, "\n"), "\n") {
		if got := DisplayWidth(line); got != 100 {
			t.Fatalf("line %d width = %d, want 100: %q", i+1, got, line)
		}
	}
}

func TestRenderDriveAddsAvailableDetailRows(t *testing.T) {
	snapshot := model.SyntheticSnapshot()
	snapshot.Assessment.ReasonCodes = []string{"ENDURANCE_LOW"}
	snapshot.Metrics.WarningTemperature = model.Some(int64(70))
	snapshot.Metrics.CriticalTemperature = model.Some(int64(80))
	snapshot.Metrics.MediaWritesBytes = model.Some(model.NewBigCounterString("4300000000000"))
	rendered, err := RenderDrive(snapshot, Options{Width: 100, Locale: LocaleEnglish})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Warning temperature", "Critical temperature", "Media/NAND writes", "ENDURANCE_LOW", "Last self-test", "Completed without error"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("full render missing %q:\n%s", want, rendered)
		}
	}
	for i, line := range strings.Split(strings.TrimSuffix(rendered, "\n"), "\n") {
		if got := DisplayWidth(line); got != 100 {
			t.Fatalf("line %d width = %d, want 100: %q", i+1, got, line)
		}
	}
}

func TestRenderDriveHidesUnsupportedDetailRows(t *testing.T) {
	snapshot := model.SyntheticSnapshot()
	snapshot.Metrics.WarningTemperature = model.None[int64](model.MissingUnsupported)
	snapshot.Metrics.CriticalTemperature = model.None[int64](model.MissingUnsupported)
	snapshot.Metrics.MediaWritesBytes = model.None[model.BigCounter](model.MissingUnsupported)
	snapshot.Metrics.LastSelfTestStatus = model.None[string](model.MissingUnsupported)

	rendered, err := RenderDrive(snapshot, Options{Width: 100, Locale: LocaleJapanese})
	if err != nil {
		t.Fatal(err)
	}
	for _, unwanted := range []string{"NAND・媒体書き込み量", "最終自己診断", "非対応"} {
		if strings.Contains(rendered, unwanted) {
			t.Fatalf("full render should hide unsupported %q:\n%s", unwanted, rendered)
		}
	}
	for i, line := range strings.Split(strings.TrimSuffix(rendered, "\n"), "\n") {
		if got := DisplayWidth(line); got != 100 {
			t.Fatalf("line %d width = %d, want 100: %q", i+1, got, line)
		}
	}
}

func TestRenderDriveMissingReasons(t *testing.T) {
	snapshot := model.SyntheticSnapshot()
	snapshot.Device.Firmware = model.None[string](model.MissingUnsupported)
	snapshot.Device.Transport = model.None[string](model.MissingUnknown)
	snapshot.Metrics.MediaWritesBytes = model.None[model.BigCounter](model.MissingPermission)

	rendered, err := RenderDrive(snapshot, Options{Width: 120, Locale: LocaleEnglish})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"UNSUPPORTED", "UNKNOWN", "PERMISSION REQUIRED"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("render missing %q:\n%s", want, rendered)
		}
	}

	rendered, err = RenderDrive(snapshot, Options{Width: 120, Locale: LocaleJapanese})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"非対応", "不明", "権限不足"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("Japanese render missing %q:\n%s", want, rendered)
		}
	}
}

func TestRenderDriveLoopStats(t *testing.T) {
	snapshot := model.SyntheticSnapshot()
	rendered, err := RenderDrive(snapshot, Options{
		Width:  100,
		Locale: LocaleEnglish,
		LoopStats: map[string]model.LoopStats{
			"disk0": {
				Valid:                true,
				ReadRateBytesPerSec:  model.Some(model.NewBigCounterString("500000")),
				WriteRateBytesPerSec: model.Some(model.NewBigCounterString("1000000")),
				ReadIOPS:             model.Some(model.NewBigCounter(500)),
				WriteIOPS:            model.Some(model.NewBigCounter(1000)),
				TemperatureChangeC:   model.Some(int64(3)),
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Read rate", "500.00 kB/s", "Write IOPS", "1,000/s", "+3°C"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("loop render missing %q:\n%s", want, rendered)
		}
	}
	for i, line := range strings.Split(strings.TrimSuffix(rendered, "\n"), "\n") {
		if got := DisplayWidth(line); got != 100 {
			t.Fatalf("line %d width = %d, want 100: %q", i+1, got, line)
		}
	}
}

func TestRenderDrivesMultiplePanels(t *testing.T) {
	first := model.SyntheticSnapshot()
	second := model.SyntheticSnapshot()
	second.Device.BSDName = "disk2"
	second.Device.DevicePath = "/dev/disk2"
	second.Device.Model = "SECOND SSD"

	rendered, err := RenderDrives([]model.DriveSnapshot{first, second}, Options{Width: 100, Locale: LocaleEnglish})
	if err != nil {
		t.Fatal(err)
	}
	if count := strings.Count(rendered, "┌"); count != 2 {
		t.Fatalf("panel count = %d, want 2:\n%s", count, rendered)
	}
	if !strings.Contains(rendered, "SECOND SSD") {
		t.Fatalf("second panel missing:\n%s", rendered)
	}
	for i, line := range strings.Split(strings.TrimSuffix(rendered, "\n"), "\n") {
		if got := DisplayWidth(line); got != 100 {
			t.Fatalf("line %d width = %d, want 100: %q", i+1, got, line)
		}
	}
}

func TestDecimalBytes(t *testing.T) {
	tests := map[string]string{
		"999":           "999 B",
		"1000":          "1.00 kB",
		"3500000256000": "3.50 TB",
		"4200000000000": "4.20 TB",
	}
	for input, want := range tests {
		if got := formatBytes(input, Options{}); got != want {
			t.Fatalf("formatBytes(%s) = %q, want %q", input, got, want)
		}
	}
}

func TestFormatBytesIEC(t *testing.T) {
	if got := formatBytes("1099511627776", Options{IEC: true}); got != "1.00 TiB" {
		t.Fatalf("IEC bytes = %q", got)
	}
}

func TestRenderSummary(t *testing.T) {
	first := model.SyntheticSnapshot()
	second := model.SyntheticSnapshot()
	second.Device.BSDName = "disk2"
	second.Device.Model = "SECOND SSD"

	rendered, err := RenderSummary([]model.DriveSnapshot{first, second}, Options{Width: 100, Locale: LocaleEnglish})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered, "HOST WRITES") || !strings.Contains(rendered, "SECOND SSD") {
		t.Fatalf("summary missing expected content:\n%s", rendered)
	}
	for i, line := range strings.Split(strings.TrimSuffix(rendered, "\n"), "\n") {
		if got := DisplayWidth(line); got != 100 {
			t.Fatalf("line %d width = %d, want 100: %q", i+1, got, line)
		}
	}
}
