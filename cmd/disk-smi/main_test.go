package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"disk-smi/internal/app"
	"disk-smi/internal/jsonout"
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

func TestValidateCheckFlags(t *testing.T) {
	tests := []struct {
		name       string
		check      bool
		jsonOutput bool
		jsonPretty bool
		summary    bool
		loop       time.Duration
		wantError  string
	}{
		{name: "disabled, anything goes", check: false, jsonOutput: true, summary: true, loop: 5 * time.Second},
		{name: "check alone", check: true},
		{name: "check with json", check: true, jsonOutput: true, wantError: "--json"},
		{name: "check with json-pretty", check: true, jsonPretty: true, wantError: "--json"},
		{name: "check with summary", check: true, summary: true, wantError: "--summary"},
		{name: "check with loop", check: true, loop: 5 * time.Second, wantError: "--loop"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCheckFlags(tt.check, tt.jsonOutput, tt.jsonPretty, tt.summary, tt.loop)
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("error = %v, want containing %q", err, tt.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
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

func TestRunLoopEmitsNDJSONWithoutScreenClearing(t *testing.T) {
	t.Setenv("DISK_SMI_LOOP_COUNT", "3")

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writer
	defer func() { os.Stdout = originalStdout }()

	calls := 0
	loopErr := runLoop(time.Nanosecond, func() (string, error) {
		calls++
		snapshots := []model.DriveSnapshot{model.SyntheticSnapshot()}
		return jsonoutRender(t, snapshots)
	})
	writer.Close()
	os.Stdout = originalStdout

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		t.Fatal(err)
	}
	if loopErr != nil {
		t.Fatal(loopErr)
	}
	if calls != 3 {
		t.Fatalf("calls = %d, want 3", calls)
	}
	output := buf.String()
	if strings.Contains(output, "\x1b[") {
		t.Fatalf("NDJSON loop output must not contain ANSI escapes:\n%q", output)
	}
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 NDJSON lines, got %d:\n%q", len(lines), output)
	}
	for _, line := range lines {
		var decoded map[string]any
		if err := json.Unmarshal([]byte(line), &decoded); err != nil {
			t.Fatalf("line is not a single valid JSON document: %v\nline: %s", err, line)
		}
		if _, ok := decoded["generated_at"]; !ok {
			t.Fatalf("NDJSON record missing generated_at timestamp: %s", line)
		}
	}
}

func jsonoutRender(t *testing.T, snapshots []model.DriveSnapshot) (string, error) {
	t.Helper()
	return jsonout.Render(snapshots, string(render.LocaleEnglish), time.Now(), jsonout.Options{})
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
