package main

import (
	"strings"
	"testing"
	"time"

	"disk-smi/internal/app"
	"disk-smi/internal/model"
	"disk-smi/internal/render"
)

func TestParseLoopInterval(t *testing.T) {
	tests := []struct {
		name      string
		short     int
		long      int
		want      time.Duration
		wantError string
	}{
		{name: "disabled"},
		{name: "short", short: 5, want: 5 * time.Second},
		{name: "long", long: 10, want: 10 * time.Second},
		{name: "same", short: 5, long: 5, want: 5 * time.Second},
		{name: "too low", short: 1, wantError: "at least 2"},
		{name: "conflict", short: 5, long: 6, wantError: "different"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseLoopInterval(tt.short, tt.long)
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("error = %v, want containing %q", err, tt.wantError)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("interval = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestValidateColor(t *testing.T) {
	for _, value := range []string{"auto", "always", "never"} {
		if err := validateColor(value); err != nil {
			t.Fatalf("validateColor(%q): %v", value, err)
		}
	}
	if err := validateColor("bad"); err == nil {
		t.Fatal("validateColor accepted bad value")
	}
}

func TestParseBackend(t *testing.T) {
	for _, value := range []string{"auto", "native", "smartctl"} {
		if _, err := parseBackend(value); err != nil {
			t.Fatalf("parseBackend(%q): %v", value, err)
		}
	}
	if _, err := parseBackend("bad"); err == nil {
		t.Fatal("parseBackend accepted bad value")
	}
}

func TestResolveWidth(t *testing.T) {
	t.Setenv("DISK_SMI_TEST_TERMINAL_WIDTH", "132")
	got, err := resolveWidth("auto")
	if err != nil {
		t.Fatal(err)
	}
	if got != 132 {
		t.Fatalf("auto width = %d, want 132", got)
	}

	t.Setenv("DISK_SMI_TEST_TERMINAL_WIDTH", "")
	got, err = resolveWidth("auto")
	if err != nil {
		t.Fatal(err)
	}
	if got != 100 {
		t.Fatalf("fallback auto width = %d, want 100", got)
	}
}

func TestShouldFallbackASCII(t *testing.T) {
	t.Setenv("TERM", "dumb")
	t.Setenv("DISK_SMI_TEST_LANG", "en_US.UTF-8")
	if !shouldFallbackASCII() {
		t.Fatal("TERM=dumb did not force ASCII")
	}

	t.Setenv("TERM", "xterm-256color")
	t.Setenv("DISK_SMI_TEST_LANG", "C")
	if !shouldFallbackASCII() {
		t.Fatal("non-UTF locale did not force ASCII")
	}

	t.Setenv("DISK_SMI_TEST_LANG", "ja_JP.UTF-8")
	if shouldFallbackASCII() {
		t.Fatal("UTF-8 locale forced ASCII")
	}
}

func TestFormatDiagnosticsEmpty(t *testing.T) {
	if got := app.FormatDiagnostics(app.Diagnostics{}); got != "" {
		t.Fatalf("empty diagnostics = %q", got)
	}
}

func TestShouldStopLoop(t *testing.T) {
	t.Setenv("DISK_SMI_LOOP_COUNT", "2")
	if shouldStopLoop(1) {
		t.Fatal("loop stopped too early")
	}
	if !shouldStopLoop(2) {
		t.Fatal("loop did not stop at limit")
	}
}

func TestRunSnapshotLoopTwoIterations(t *testing.T) {
	t.Setenv("DISK_SMI_LOOP_COUNT", "2")
	calls := 0
	err := runSnapshotLoop(time.Nanosecond, render.Options{Width: 100, Locale: render.LocaleEnglish}, "100", func() ([]model.DriveSnapshot, error) {
		calls++
		snapshot := model.SyntheticSnapshot()
		if calls == 2 {
			snapshot.Metrics.HostReadsBytes = model.Some(model.NewBigCounterString("3500001000000"))
			snapshot.Metrics.HostWritesBytes = model.Some(model.NewBigCounterString("4200002000000"))
			snapshot.Metrics.ReadCommands = model.Some(model.NewBigCounter(82020334))
			snapshot.Metrics.WriteCommands = model.Some(model.NewBigCounter(61276004))
			snapshot.Metrics.TemperatureCelsius = model.Some(int64(39))
		}
		return []model.DriveSnapshot{snapshot}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

func TestRunSnapshotLoopAutoWidthRecomputes(t *testing.T) {
	t.Setenv("DISK_SMI_LOOP_COUNT", "2")
	t.Setenv("DISK_SMI_TEST_TERMINAL_WIDTH_SEQUENCE", "80,100")
	terminalWidthSequenceIndex = 0
	calls := 0
	err := runSnapshotLoop(time.Nanosecond, render.Options{Locale: render.LocaleEnglish, ASCII: true}, "auto", func() ([]model.DriveSnapshot, error) {
		calls++
		return []model.DriveSnapshot{model.SyntheticSnapshot()}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
	if terminalWidthSequenceIndex != 2 {
		t.Fatalf("terminal width calls = %d, want 2", terminalWidthSequenceIndex)
	}
}
