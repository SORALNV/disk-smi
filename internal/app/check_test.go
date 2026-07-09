package app

import (
	"path/filepath"
	"strings"
	"testing"

	"disk-smi/internal/render"
)

func fixture(name string) string {
	return filepath.Join("..", "..", "testdata", "smartctl", name)
}

func TestRunCheckGoodDrive(t *testing.T) {
	output, code, _, err := RunCheck(fixture("nvme-good.json"), "", render.LocaleEnglish, SnapshotOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d", code, ExitOK)
	}
	if !strings.HasPrefix(output, "GOOD disk0 APPLE SSD AP1024Z endurance=70% temp=36C") {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestRunCheckCautionDrive(t *testing.T) {
	output, code, _, err := RunCheck(fixture("nvme-endurance-90.json"), "", render.LocaleEnglish, SnapshotOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if code != CheckExitWarning {
		t.Fatalf("exit code = %d, want %d", code, CheckExitWarning)
	}
	if !strings.HasPrefix(output, "CAUTION ") {
		t.Fatalf("unexpected output: %q", output)
	}
	if !strings.Contains(output, "endurance=10%") {
		t.Fatalf("output missing endurance: %q", output)
	}
	if !strings.Contains(output, "reasons=") {
		t.Fatalf("output missing reason codes: %q", output)
	}
}

func TestRunCheckCriticalDrive(t *testing.T) {
	output, code, _, err := RunCheck(fixture("nvme-critical-warning.json"), "", render.LocaleEnglish, SnapshotOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if code != CheckExitCritical {
		t.Fatalf("exit code = %d, want %d", code, CheckExitCritical)
	}
	if !strings.HasPrefix(output, "CRITICAL ") {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestRunCheckMissingTemperatureRendersDash(t *testing.T) {
	output, _, _, err := RunCheck(fixture("nvme-missing-temperature.json"), "", render.LocaleEnglish, SnapshotOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "temp=-") {
		t.Fatalf("missing temperature was not rendered as -: %q", output)
	}
	if strings.Contains(output, "temp=0") {
		t.Fatalf("missing temperature must not be faked as zero: %q", output)
	}
}

func TestRunCheckJapaneseLocale(t *testing.T) {
	output, code, _, err := RunCheck(fixture("nvme-good.json"), "", render.LocaleJapanese, SnapshotOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d", code, ExitOK)
	}
	if !strings.HasPrefix(output, "正常 ") {
		t.Fatalf("unexpected Japanese output: %q", output)
	}
}

func TestRunCheckPropagatesRuntimeErrors(t *testing.T) {
	_, code, _, err := RunCheck(fixture("nvme-malformed.json"), "", render.LocaleEnglish, SnapshotOptions{})
	if err == nil {
		t.Fatal("expected error for malformed fixture")
	}
	if code == ExitOK || code == CheckExitWarning || code == CheckExitCritical {
		t.Fatalf("runtime error exit code %d collided with check status codes", code)
	}
}
