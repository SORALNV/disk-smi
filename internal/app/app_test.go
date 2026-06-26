package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"disk-smi/internal/model"
	"disk-smi/internal/render"
)

func TestRunWithInputFixture(t *testing.T) {
	output, err := Run(filepath.Join("..", "..", "testdata", "smartctl", "nvme-good.json"), "", render.Options{
		Width:  100,
		Locale: render.LocaleEnglish,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"APPLE SSD AP1024Z", "GOOD", "Endurance 70%", "Host writes  4.20 TB"} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func TestRunJSONWithInputFixture(t *testing.T) {
	output, err := RunJSON(filepath.Join("..", "..", "testdata", "smartctl", "nvme-endurance-126.json"), "", render.LocaleJapanese, true, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"display_locale": "ja-JP"`, `"overall_status": "caution"`, `"used_percent": 126`, `"remaining_percent": 0`} {
		if !strings.Contains(output, want) {
			t.Fatalf("JSON output missing %q:\n%s", want, output)
		}
	}
}

func TestRunRejectsUnsafeTarget(t *testing.T) {
	if _, err := Run("", "/dev/disk0;rm", render.Options{Width: 100, Locale: render.LocaleEnglish}); err == nil {
		t.Fatalf("unsafe target accepted")
	}
}

func TestSnapshotsWithOptionsUsesDetailedSmartctl(t *testing.T) {
	path := fakeSmartctl(t)
	_, diagnostics, err := SnapshotsWithOptionsAndDiagnostics("", "disk0", SnapshotOptions{
		Detail:       true,
		SmartctlPath: path,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics.Commands) != 1 {
		t.Fatalf("diagnostic command count = %d", len(diagnostics.Commands))
	}
	got := strings.Join(diagnostics.Commands[0].Command, " ")
	want := path + " -x -j /dev/disk0"
	if got != want {
		t.Fatalf("command = %q, want %q", got, want)
	}
}

func TestSnapshotsWithOptionsUsesStandardSmartctlByDefault(t *testing.T) {
	path := fakeSmartctl(t)
	_, diagnostics, err := SnapshotsWithOptionsAndDiagnostics("", "disk0", SnapshotOptions{
		SmartctlPath: path,
	})
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(diagnostics.Commands[0].Command, " ")
	want := path + " -a -j /dev/disk0"
	if got != want {
		t.Fatalf("command = %q, want %q", got, want)
	}
}

func TestSnapshotsWithNativeBackendDoesNotUseSmartctl(t *testing.T) {
	diskutilPath := fakeDiskutil(t)
	profilerPath := fakeSystemProfiler(t)
	ioregPath := fakeIOReg(t)
	snapshots, diagnostics, err := SnapshotsWithOptionsAndDiagnostics("", "disk0", SnapshotOptions{
		Backend:            BackendNative,
		SmartctlPath:       filepath.Join(t.TempDir(), "missing-smartctl"),
		DiskutilPath:       diskutilPath,
		SystemProfilerPath: profilerPath,
		IORegPath:          ioregPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshots) != 1 {
		t.Fatalf("snapshot count = %d, want 1", len(snapshots))
	}
	if snapshots[0].Device.Model != "APPLE SSD AP1024Z" {
		t.Fatalf("model = %q", snapshots[0].Device.Model)
	}
	if !snapshots[0].Metrics.SMARTPassed.Valid || !snapshots[0].Metrics.SMARTPassed.Value {
		t.Fatalf("native backend did not use system_profiler SMART status: %#v", snapshots[0].Metrics.SMARTPassed)
	}
	if !snapshots[0].Device.Firmware.Valid || snapshots[0].Device.Firmware.Value != "2973.100" {
		t.Fatalf("firmware = %#v", snapshots[0].Device.Firmware)
	}
	if !snapshots[0].Device.NVMeVersion.Valid || snapshots[0].Device.NVMeVersion.Value != "1.10" {
		t.Fatalf("nvme version = %#v", snapshots[0].Device.NVMeVersion)
	}
	if !snapshots[0].Device.Transport.Valid || snapshots[0].Device.Transport.Value != "Apple Fabric" {
		t.Fatalf("transport = %#v", snapshots[0].Device.Transport)
	}
	if snapshots[0].Assessment.DataQuality != model.DataQualityFull {
		t.Fatalf("data quality = %q, want full", snapshots[0].Assessment.DataQuality)
	}
	if snapshots[0].Metrics.EnduranceUsedPercent.Valid {
		if snapshots[0].Metrics.EnduranceUsedPercent.Value != 0 {
			t.Fatalf("endurance used = %#v", snapshots[0].Metrics.EnduranceUsedPercent)
		}
	} else {
		t.Fatalf("native backend did not read diskutil endurance data")
	}
	if !snapshots[0].Metrics.TemperatureCelsius.Valid || snapshots[0].Metrics.TemperatureCelsius.Value != 38 {
		t.Fatalf("temperature = %#v", snapshots[0].Metrics.TemperatureCelsius)
	}
	if !snapshots[0].Metrics.HostWritesBytes.Valid || snapshots[0].Metrics.HostWritesBytes.Value.String() != "4189639680000" {
		t.Fatalf("host writes = %#v", snapshots[0].Metrics.HostWritesBytes)
	}
	if !snapshots[0].Metrics.PowerOnHours.Valid || snapshots[0].Metrics.PowerOnHours.Value.String() != "200" {
		t.Fatalf("power on hours = %#v", snapshots[0].Metrics.PowerOnHours)
	}
	for _, command := range diagnostics.Commands {
		if len(command.Command) > 0 && strings.Contains(command.Command[0], "smartctl") {
			t.Fatalf("native backend executed smartctl: %#v", diagnostics.Commands)
		}
	}
}

func TestSnapshotsAutoFallsBackToNativeWhenSmartctlMissing(t *testing.T) {
	diskutilPath := fakeDiskutil(t)
	profilerPath := fakeSystemProfiler(t)
	ioregPath := fakeIOReg(t)
	snapshots, diagnostics, err := SnapshotsWithOptionsAndDiagnostics("", "disk0", SnapshotOptions{
		SmartctlPath:       filepath.Join(t.TempDir(), "missing-smartctl"),
		DiskutilPath:       diskutilPath,
		SystemProfilerPath: profilerPath,
		IORegPath:          ioregPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if snapshots[0].Device.BSDName != "disk0" {
		t.Fatalf("snapshot = %#v", snapshots[0].Device)
	}
	if snapshots[0].Assessment.OverallStatus != model.StatusGood {
		t.Fatalf("status = %q, want good", snapshots[0].Assessment.OverallStatus)
	}
	var sawSmartctl, sawDiskutil, sawProfiler, sawIOReg bool
	for _, command := range diagnostics.Commands {
		if len(command.Command) == 0 {
			continue
		}
		if strings.Contains(command.Command[0], "missing-smartctl") {
			sawSmartctl = true
		}
		if command.Command[0] == diskutilPath {
			sawDiskutil = true
		}
		if command.Command[0] == profilerPath {
			sawProfiler = true
		}
		if command.Command[0] == ioregPath {
			sawIOReg = true
		}
	}
	if !sawSmartctl || !sawDiskutil || !sawProfiler || !sawIOReg {
		t.Fatalf("diagnostics did not include smartctl, diskutil, system_profiler, and ioreg commands: %#v", diagnostics.Commands)
	}
}

func TestExitCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "nil", want: ExitOK},
		{name: "no ssd", err: coded(ExitNoSSD, fmt.Errorf("no SSD found")), want: ExitNoSSD},
		{name: "smart failure", err: coded(ExitSMARTFailure, fmt.Errorf("SMART information unavailable")), want: ExitSMARTFailure},
		{name: "missing dependency", err: fmt.Errorf("smartctl failed: %w", &exec.Error{Name: "smartctl", Err: exec.ErrNotFound}), want: ExitMissingDependency},
		{name: "permission", err: fmt.Errorf("permission denied"), want: ExitPermission},
		{name: "unknown", err: fmt.Errorf("other failure"), want: ExitInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExitCode(tt.err); got != tt.want {
				t.Fatalf("ExitCode(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}
}

func TestUserMessagePermission(t *testing.T) {
	err := fmt.Errorf("smartctl permission denied: Permission denied")
	if got := UserMessage(err, render.LocaleEnglish); !strings.Contains(got, "sudo disk-smi") {
		t.Fatalf("English permission message = %q", got)
	}
	if got := UserMessage(err, render.LocaleJapanese); !strings.Contains(got, "管理者権限") {
		t.Fatalf("Japanese permission message = %q", got)
	}
}

func TestUserMessageMissingSmartctl(t *testing.T) {
	err := fmt.Errorf("smartctl failed: %w", &exec.Error{Name: "smartctl", Err: exec.ErrNotFound})
	if got := UserMessage(err, render.LocaleEnglish); !strings.Contains(got, "disk-smi --backend native") {
		t.Fatalf("English missing smartctl message = %q", got)
	}
	if got := UserMessage(err, render.LocaleJapanese); !strings.Contains(got, "自前backend") {
		t.Fatalf("Japanese missing smartctl message = %q", got)
	}
}

func TestFormatDiagnostics(t *testing.T) {
	diagnostics := Diagnostics{
		Commands: []CommandDiagnostic{
			{
				Command:  []string{"smartctl", "-a", "-j", "/dev/disk0"},
				ExitCode: 4,
				Stderr:   "warning text\n",
			},
			{
				Command:  []string{"/usr/sbin/diskutil", "info", "-plist", "/dev/disk0"},
				ExitCode: 0,
			},
		},
	}

	got := FormatDiagnostics(diagnostics)
	for _, want := range []string{
		"debug: command: smartctl -a -j /dev/disk0",
		"debug: exit_code: 4",
		"debug: stderr: warning text",
		"debug: command: /usr/sbin/diskutil info -plist /dev/disk0",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("diagnostics missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "APPLE SSD") || strings.Contains(got, "SYNTHETIC9K2A") {
		t.Fatalf("diagnostics leaked SMART payload data:\n%s", got)
	}
}

func fakeSmartctl(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is Unix-only")
	}
	path := filepath.Join(t.TempDir(), "smartctl")
	script := `#!/bin/sh
cat <<'JSON'
{"device":{"name":"/dev/disk0","type":"nvme","protocol":"NVMe"},"model_name":"TEST SSD","nvme_total_capacity":1000,"smart_status":{"passed":true},"nvme_smart_health_information_log":{"critical_warning":0,"percentage_used":1,"available_spare":100,"available_spare_threshold":10,"media_errors":0}}
JSON
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func fakeDiskutil(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is Unix-only")
	}
	root := filepath.Join("..", "..", "testdata", "diskutil")
	path := filepath.Join(t.TempDir(), "diskutil")
	script := `#!/bin/sh
if [ "$1" = "list" ] && [ "$2" = "-plist" ]; then
  cat "` + filepath.Join(root, "list.plist") + `"
  exit 0
fi
if [ "$1" = "info" ] && [ "$2" = "-plist" ] && [ "$3" = "/dev/disk0" ]; then
  cat "` + filepath.Join(root, "info-disk0.plist") + `"
  exit 0
fi
if [ "$1" = "info" ] && [ "$2" = "-plist" ] && [ "$3" = "/dev/disk3" ]; then
  cat "` + filepath.Join(root, "info-disk3.plist") + `"
  exit 0
fi
echo "unexpected args: $@" >&2
exit 2
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func fakeSystemProfiler(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is Unix-only")
	}
	root := filepath.Join("..", "..", "testdata", "system_profiler")
	path := filepath.Join(t.TempDir(), "system_profiler")
	script := `#!/bin/sh
if [ "$1" = "SPNVMeDataType" ] && [ "$2" = "-json" ]; then
  cat "` + filepath.Join(root, "nvme.json") + `"
  exit 0
fi
echo "unexpected args: $@" >&2
exit 2
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func fakeIOReg(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is Unix-only")
	}
	root := filepath.Join("..", "..", "testdata", "ioreg")
	path := filepath.Join(t.TempDir(), "ioreg")
	script := `#!/bin/sh
if [ "$1" = "-r" ] && [ "$2" = "-c" ] && [ "$3" = "IOBlockStorageDriver" ] && [ "$4" = "-l" ] && [ "$5" = "-a" ]; then
  cat "` + filepath.Join(root, "block-storage-driver.plist") + `"
  exit 0
fi
if [ "$1" = "-r" ] && [ "$2" = "-c" ] && [ "$3" = "IONVMeController" ] && [ "$4" = "-l" ] && [ "$5" = "-a" ]; then
  cat "` + filepath.Join(root, "nvme-controller.plist") + `"
  exit 0
fi
echo "unexpected args: $@" >&2
exit 2
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}
