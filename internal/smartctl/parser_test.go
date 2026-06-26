package smartctl

import (
	"os"
	"path/filepath"
	"testing"

	"disk-smi/internal/health"
	"disk-smi/internal/model"
)

func TestParseNVMeGood(t *testing.T) {
	snapshot := parseFixture(t, "nvme-good.json")

	if snapshot.Device.DevicePath != "/dev/disk0" {
		t.Fatalf("device path = %q", snapshot.Device.DevicePath)
	}
	if snapshot.Device.Serial.Value != "****9K2A" {
		t.Fatalf("masked serial = %q", snapshot.Device.Serial.Value)
	}
	if snapshot.Metrics.HostWritesBytes.Value.String() != "4200000000000" {
		t.Fatalf("host writes bytes = %s", snapshot.Metrics.HostWritesBytes.Value.String())
	}
	if snapshot.Metrics.HostReadsBytes.Value.String() != "3500000256000" {
		t.Fatalf("host reads bytes = %s", snapshot.Metrics.HostReadsBytes.Value.String())
	}
	if len(snapshot.Metrics.TemperatureSensors) != 2 {
		t.Fatalf("temperature sensor count = %d", len(snapshot.Metrics.TemperatureSensors))
	}
}

func TestParseNVMeSmartctl75HostCommandAliases(t *testing.T) {
	snapshot := parseFixture(t, "nvme-smartctl75-apple.json")

	if !snapshot.Metrics.ReadCommands.Valid || snapshot.Metrics.ReadCommands.Value.String() != "294047374" {
		t.Fatalf("read commands = %#v", snapshot.Metrics.ReadCommands)
	}
	if !snapshot.Metrics.WriteCommands.Valid || snapshot.Metrics.WriteCommands.Value.String() != "207203122" {
		t.Fatalf("write commands = %#v", snapshot.Metrics.WriteCommands)
	}
	if !snapshot.Metrics.HostReadsBytes.Valid || snapshot.Metrics.HostReadsBytes.Value.String() != "7126110720000" {
		t.Fatalf("host reads bytes = %#v", snapshot.Metrics.HostReadsBytes)
	}
	if snapshot.Metrics.MediaWritesBytes.Valid || snapshot.Metrics.MediaWritesBytes.Reason != model.MissingUnsupported {
		t.Fatalf("media writes bytes = %#v, want unsupported missing value", snapshot.Metrics.MediaWritesBytes)
	}
	if snapshot.Metrics.LastSelfTestStatus.Valid || snapshot.Metrics.LastSelfTestStatus.Reason != model.MissingUnsupported {
		t.Fatalf("last self-test status = %#v, want unsupported missing value", snapshot.Metrics.LastSelfTestStatus)
	}
	if snapshot.Metrics.LastSelfTestPassed.Valid || snapshot.Metrics.LastSelfTestPassed.Reason != model.MissingUnsupported {
		t.Fatalf("last self-test passed = %#v, want unsupported missing value", snapshot.Metrics.LastSelfTestPassed)
	}
}

func TestParseNVMeIdentifyTemperatureThresholds(t *testing.T) {
	snapshot := parseFixture(t, "nvme-identify-temperature-thresholds.json")
	assessment := health.Assess(snapshot.Metrics)

	if !snapshot.Metrics.WarningTemperature.Valid || snapshot.Metrics.WarningTemperature.Value != 70 {
		t.Fatalf("warning temperature = %#v", snapshot.Metrics.WarningTemperature)
	}
	if !snapshot.Metrics.CriticalTemperature.Valid || snapshot.Metrics.CriticalTemperature.Value != 80 {
		t.Fatalf("critical temperature = %#v", snapshot.Metrics.CriticalTemperature)
	}
	if assessment.OverallStatus != model.StatusCaution || !contains(assessment.ReasonCodes, health.TemperatureWarning) {
		t.Fatalf("assessment = %#v, want temperature warning caution", assessment)
	}
}

func TestParseOCPPhysicalMediaUnitsWritten(t *testing.T) {
	snapshot := parseFixture(t, "nvme-ocp-physical-media-units.json")

	if !snapshot.Metrics.MediaWritesBytes.Valid {
		t.Fatalf("media writes bytes missing: %#v", snapshot.Metrics.MediaWritesBytes)
	}
	if got := snapshot.Metrics.MediaWritesBytes.Value.String(); got != "18446744073709551621" {
		t.Fatalf("media writes bytes = %s", got)
	}
}

func TestParsePhysicalMediaUnitsWrittenString(t *testing.T) {
	data := []byte(`{
		"device":{"name":"/dev/disk0","type":"nvme","protocol":"NVMe"},
		"model_name":"OCP NVME SSD",
		"smart_status":{"passed":true},
		"physical_media_units_written":"123456789012345678901234567890"
	}`)

	snapshot, err := Parse(data, "")
	if err != nil {
		t.Fatal(err)
	}
	if got := snapshot.Metrics.MediaWritesBytes.Value.String(); got != "123456789012345678901234567890" {
		t.Fatalf("media writes bytes = %s", got)
	}
}

func TestParseEndurance126AssessesAsCaution(t *testing.T) {
	snapshot := parseFixture(t, "nvme-endurance-126.json")
	assessment := health.Assess(snapshot.Metrics)

	if snapshot.Metrics.EnduranceUsedPercent.Value != 126 {
		t.Fatalf("endurance used = %d", snapshot.Metrics.EnduranceUsedPercent.Value)
	}
	if assessment.OverallStatus != model.StatusCaution {
		t.Fatalf("status = %s, want caution", assessment.OverallStatus)
	}
	if len(assessment.ReasonCodes) != 1 || assessment.ReasonCodes[0] != health.EnduranceRatedLimitReached {
		t.Fatalf("reason codes = %#v", assessment.ReasonCodes)
	}
}

func TestFixtureAssessments(t *testing.T) {
	tests := []struct {
		fixture string
		status  model.OverallStatus
		reason  string
	}{
		{fixture: "nvme-endurance-0.json", status: model.StatusGood},
		{fixture: "nvme-endurance-90.json", status: model.StatusCaution, reason: health.EnduranceLow},
		{fixture: "nvme-endurance-100.json", status: model.StatusCaution, reason: health.EnduranceRatedLimitReached},
		{fixture: "nvme-smart-failed.json", status: model.StatusCritical, reason: health.SMARTFailed},
		{fixture: "nvme-critical-warning.json", status: model.StatusCritical, reason: health.CriticalWarningActive},
		{fixture: "nvme-read-only-warning.json", status: model.StatusCritical, reason: health.CriticalWarningReadOnly},
		{fixture: "nvme-spare-below-threshold.json", status: model.StatusCritical, reason: health.AvailableSpareBelowThreshold},
		{fixture: "nvme-media-errors.json", status: model.StatusCaution, reason: health.MediaErrorsPresent},
		{fixture: "nvme-temperature-warning.json", status: model.StatusCaution, reason: health.TemperatureWarning},
		{fixture: "nvme-temperature-critical.json", status: model.StatusCritical, reason: health.TemperatureCritical},
		{fixture: "nvme-error-log-only.json", status: model.StatusGood},
		{fixture: "nvme-missing-temperature.json", status: model.StatusGood},
		{fixture: "nvme-self-test-failed.json", status: model.StatusCaution, reason: health.SelfTestFailed},
	}

	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			snapshot := parseFixture(t, tt.fixture)
			assessment := health.Assess(snapshot.Metrics)
			if assessment.OverallStatus != tt.status {
				t.Fatalf("status = %s, want %s; reasons=%#v", assessment.OverallStatus, tt.status, assessment.ReasonCodes)
			}
			if tt.reason != "" && !contains(assessment.ReasonCodes, tt.reason) {
				t.Fatalf("reasons = %#v, want %s", assessment.ReasonCodes, tt.reason)
			}
			if tt.fixture == "nvme-endurance-100.json" && snapshot.Metrics.EnduranceUsedPercent.Value != 100 {
				t.Fatalf("endurance used was not preserved")
			}
		})
	}
}

func TestParseSelfTestStatus(t *testing.T) {
	snapshot := parseFixture(t, "nvme-self-test-failed.json")

	if !snapshot.Metrics.LastSelfTestStatus.Valid {
		t.Fatalf("last self-test status missing")
	}
	if snapshot.Metrics.LastSelfTestStatus.Value != "Completed: failed" {
		t.Fatalf("last self-test status = %q", snapshot.Metrics.LastSelfTestStatus.Value)
	}
	if !snapshot.Metrics.LastSelfTestPassed.Valid || snapshot.Metrics.LastSelfTestPassed.Value {
		t.Fatalf("last self-test passed = %#v, want valid false", snapshot.Metrics.LastSelfTestPassed)
	}
}

func TestParseMissingEnduranceDoesNotBecomeZero(t *testing.T) {
	snapshot := parseFixture(t, "nvme-missing-endurance.json")

	if snapshot.Metrics.EnduranceUsedPercent.Valid {
		t.Fatalf("missing endurance parsed as valid: %#v", snapshot.Metrics.EnduranceUsedPercent)
	}
	assessment := health.Assess(snapshot.Metrics)
	if assessment.OverallStatus == model.StatusGood {
		t.Fatalf("missing endurance assessed as good")
	}
}

func TestParseUSBUnavailableIsUnknown(t *testing.T) {
	snapshot := parseFixture(t, "usb-unavailable.json")
	assessment := health.Assess(snapshot.Metrics)

	if snapshot.Metrics.SMARTPassed.Valid {
		t.Fatalf("SMART passed should be missing")
	}
	if assessment.OverallStatus != model.StatusUnknown {
		t.Fatalf("status = %s, want unknown; reasons=%#v", assessment.OverallStatus, assessment.ReasonCodes)
	}
	if !contains(assessment.ReasonCodes, health.SmartDataUnavailable) {
		t.Fatalf("reasons = %#v, want SMART unavailable", assessment.ReasonCodes)
	}
}

func TestParseSATAGoodBestEffort(t *testing.T) {
	snapshot := parseFixture(t, "sata-good.json")

	if snapshot.Device.Protocol != "ATA" {
		t.Fatalf("protocol = %q", snapshot.Device.Protocol)
	}
	if snapshot.Device.Transport.Value != "SATA" {
		t.Fatalf("transport = %#v", snapshot.Device.Transport)
	}
	if snapshot.Metrics.TemperatureCelsius.Value != 36 {
		t.Fatalf("temperature = %#v", snapshot.Metrics.TemperatureCelsius)
	}
	if snapshot.Metrics.PowerOnHours.Value.String() != "1500" {
		t.Fatalf("power on hours = %#v", snapshot.Metrics.PowerOnHours)
	}
	if snapshot.Metrics.PowerCycles.Value.String() != "428" {
		t.Fatalf("power cycles = %#v", snapshot.Metrics.PowerCycles)
	}
	if snapshot.Metrics.HostWritesBytes.Valid {
		t.Fatalf("SATA fixture should not synthesize host writes: %#v", snapshot.Metrics.HostWritesBytes)
	}
	assessment := health.Assess(snapshot.Metrics)
	if assessment.OverallStatus == model.StatusGood {
		t.Fatalf("SATA best-effort missing NVMe-required fields should not be GOOD")
	}
}

func TestParseLargeCounters(t *testing.T) {
	snapshot := parseFixture(t, "nvme-large-counters.json")

	if got := snapshot.Metrics.ReadCommands.Value.String(); got != "184467440737095516162" {
		t.Fatalf("read commands = %s", got)
	}
	if got := snapshot.Metrics.HostReadsBytes.Value.String(); got != "94447329657392904273920000" {
		t.Fatalf("host reads bytes = %s", got)
	}
}

func TestParseMalformed(t *testing.T) {
	data, err := os.ReadFile(fixturePath("nvme-malformed.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse(data, ""); err == nil {
		t.Fatalf("Parse malformed fixture succeeded")
	}
}

func parseFixture(t *testing.T, name string) model.DriveSnapshot {
	t.Helper()
	data, err := os.ReadFile(fixturePath(name))
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := Parse(data, "")
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func fixturePath(name string) string {
	return filepath.Join("..", "..", "testdata", "smartctl", name)
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
