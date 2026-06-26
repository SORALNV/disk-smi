package jsonout

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"disk-smi/internal/health"
	"disk-smi/internal/model"
	"disk-smi/internal/smartctl"
)

func TestRenderJSON(t *testing.T) {
	snapshot := parseFixture(t, "nvme-good.json")
	snapshot.Assessment = health.Assess(snapshot.Metrics)

	rendered, err := Render([]model.DriveSnapshot{snapshot}, "ja-JP", fixedTime(), Options{Pretty: true})
	if err != nil {
		t.Fatal(err)
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(rendered), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["schema_version"].(float64) != 1 {
		t.Fatalf("schema_version = %#v", decoded["schema_version"])
	}
	if decoded["display_locale"].(string) != "ja-JP" {
		t.Fatalf("display_locale = %#v", decoded["display_locale"])
	}

	drive := decoded["drives"].([]any)[0].(map[string]any)
	device := drive["device"].(map[string]any)
	if device["serial_masked"].(string) != "****9K2A" {
		t.Fatalf("serial_masked = %#v", device["serial_masked"])
	}

	io := drive["io"].(map[string]any)
	if io["host_writes_bytes"].(string) != "4200000000000" {
		t.Fatalf("host_writes_bytes = %#v", io["host_writes_bytes"])
	}
	if io["media_writes_bytes"] != nil {
		t.Fatalf("media_writes_bytes = %#v, want nil", io["media_writes_bytes"])
	}

	endurance := drive["endurance"].(map[string]any)
	if endurance["remaining_is_failure_probability"].(bool) {
		t.Fatalf("remaining_is_failure_probability should be false")
	}
	if endurance["zero_means_immediate_failure"].(bool) {
		t.Fatalf("zero_means_immediate_failure should be false")
	}
	reliability := drive["reliability"].(map[string]any)
	if reliability["smart_passed"].(bool) != true {
		t.Fatalf("smart_passed = %#v", reliability["smart_passed"])
	}
	thermals := drive["thermals"].(map[string]any)
	if thermals["temperature_celsius"].(float64) != 36 {
		t.Fatalf("temperature_celsius = %#v", thermals["temperature_celsius"])
	}
	sensors := thermals["temperature_sensors_celsius"].([]any)
	if len(sensors) != 2 || sensors[0].(float64) != 36 || sensors[1].(float64) != 41 {
		t.Fatalf("temperature_sensors_celsius = %#v", sensors)
	}
	missing := drive["missing_reasons"].(map[string]any)
	if missing["io.media_writes_bytes"].(string) != "unsupported" {
		t.Fatalf("media write missing reason = %#v", missing["io.media_writes_bytes"])
	}
}

func TestRenderJSONMissingValuesAsNull(t *testing.T) {
	snapshot := parseFixture(t, "nvme-missing-endurance.json")
	snapshot.Assessment = health.Assess(snapshot.Metrics)

	rendered, err := Render([]model.DriveSnapshot{snapshot}, "en-US", fixedTime(), Options{})
	if err != nil {
		t.Fatal(err)
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(rendered), &decoded); err != nil {
		t.Fatal(err)
	}
	drive := decoded["drives"].([]any)[0].(map[string]any)
	endurance := drive["endurance"].(map[string]any)
	if endurance["used_percent"] != nil {
		t.Fatalf("used_percent = %#v, want nil", endurance["used_percent"])
	}
	if endurance["remaining_percent"] != nil {
		t.Fatalf("remaining_percent = %#v, want nil", endurance["remaining_percent"])
	}
	missing := drive["missing_reasons"].(map[string]any)
	if missing["endurance.used_percent"].(string) == "" {
		t.Fatalf("missing reason for endurance not rendered: %#v", missing)
	}
}

func TestRenderJSONMissingTemperatureSensorsAsNull(t *testing.T) {
	snapshot := model.SyntheticSnapshot()
	snapshot.Metrics.TemperatureSensors = nil
	snapshot.Assessment = health.Assess(snapshot.Metrics)

	rendered, err := Render([]model.DriveSnapshot{snapshot}, "en-US", fixedTime(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(rendered), &decoded); err != nil {
		t.Fatal(err)
	}
	drive := decoded["drives"].([]any)[0].(map[string]any)
	thermals := drive["thermals"].(map[string]any)
	if thermals["temperature_sensors_celsius"] != nil {
		t.Fatalf("temperature_sensors_celsius = %#v, want nil", thermals["temperature_sensors_celsius"])
	}
	missing := drive["missing_reasons"].(map[string]any)
	if missing["thermals.temperature_sensors_celsius"].(string) != "unsupported" {
		t.Fatalf("sensor missing reason = %#v", missing["thermals.temperature_sensors_celsius"])
	}
}

func TestRenderJSONMultipleDrives(t *testing.T) {
	first := parseFixture(t, "nvme-good.json")
	first.Assessment = health.Assess(first.Metrics)
	second := first
	second.Device.BSDName = "disk2"
	second.Device.DevicePath = "/dev/disk2"

	rendered, err := Render([]model.DriveSnapshot{first, second}, "en-US", fixedTime(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(rendered), &decoded); err != nil {
		t.Fatal(err)
	}
	drives := decoded["drives"].([]any)
	if len(drives) != 2 {
		t.Fatalf("drive count = %d, want 2", len(drives))
	}
}

func TestRenderJSONShowSerial(t *testing.T) {
	snapshot := parseFixture(t, "nvme-good.json")
	snapshot.Assessment = health.Assess(snapshot.Metrics)

	rendered, err := Render([]model.DriveSnapshot{snapshot}, "en-US", fixedTime(), Options{ShowSerial: true})
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(rendered), &decoded); err != nil {
		t.Fatal(err)
	}
	drive := decoded["drives"].([]any)[0].(map[string]any)
	device := drive["device"].(map[string]any)
	if device["serial"].(string) != "SYNTHETIC9K2A" {
		t.Fatalf("serial = %#v", device["serial"])
	}
}

func parseFixture(t *testing.T, name string) model.DriveSnapshot {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "smartctl", name))
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := smartctl.Parse(data, "")
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func fixedTime() time.Time {
	return time.Date(2026, 6, 25, 19, 42, 10, 0, time.FixedZone("JST", 9*60*60))
}
