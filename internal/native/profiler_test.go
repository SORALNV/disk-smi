package native

import (
	"os"
	"path/filepath"
	"testing"

	"disk-smi/internal/discovery"
	"disk-smi/internal/model"
)

func TestParseProfile(t *testing.T) {
	devices, err := ParseProfile(readFixture(t, "nvme.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(devices) != 1 {
		t.Fatalf("device count = %d, want 1", len(devices))
	}
	if devices[0].BSDName != "disk0" || devices[0].SMARTStatus != "Verified" {
		t.Fatalf("device = %#v", devices[0])
	}
}

func TestMergeProfile(t *testing.T) {
	snapshot := Snapshot(discovery.Candidate{
		BSDName:      "disk0",
		DevicePath:   "/dev/disk0",
		Model:        "APPLE SSD AP1024Z",
		CapacityByte: model.NewBigCounterString("1000204886016"),
	})
	devices, err := ParseProfile(readFixture(t, "nvme.json"))
	if err != nil {
		t.Fatal(err)
	}
	got := MergeProfile(snapshot, devices)
	if got.Device.Protocol != "NVMe" {
		t.Fatalf("protocol = %q", got.Device.Protocol)
	}
	if !got.Device.Firmware.Valid || got.Device.Firmware.Value != "2973.100" {
		t.Fatalf("firmware = %#v", got.Device.Firmware)
	}
	if !got.Device.Serial.Valid || got.Device.Serial.Value != "****9K2A" {
		t.Fatalf("masked serial = %#v", got.Device.Serial)
	}
	if !got.Metrics.SMARTPassed.Valid || !got.Metrics.SMARTPassed.Value {
		t.Fatalf("SMARTPassed = %#v", got.Metrics.SMARTPassed)
	}
}

func TestParseStorageStats(t *testing.T) {
	stats, err := ParseStorageStats(readIORegFixture(t, "block-storage-driver.plist"))
	if err != nil {
		t.Fatal(err)
	}
	if len(stats) != 1 {
		t.Fatalf("stats count = %d, want 1", len(stats))
	}
	if stats[0].BSDName != "disk0" || stats[0].BytesWritten != 2254375239680 {
		t.Fatalf("stats = %#v", stats[0])
	}
	if stats[0].TotalTimeReadNS != 51202470831156 || stats[0].TotalTimeWriteNS != 6728867279808 {
		t.Fatalf("total time stats = %#v", stats[0])
	}
}

func TestParseControllerInfos(t *testing.T) {
	infos, err := ParseControllerInfos(readIORegFixture(t, "nvme-controller.plist"))
	if err != nil {
		t.Fatal(err)
	}
	if len(infos) != 1 {
		t.Fatalf("info count = %d, want 1", len(infos))
	}
	info := infos[0]
	if info.BSDName != "disk0" || info.NVMeRevision != "1.10" || info.Firmware != "2973.100" {
		t.Fatalf("info = %#v", info)
	}
	if info.Transport != "Apple Fabric" || info.Location != "internal" {
		t.Fatalf("transport/location = %#v", info)
	}
}

func TestMergeStorageStats(t *testing.T) {
	snapshot := Snapshot(discovery.Candidate{
		BSDName:      "disk0",
		DevicePath:   "/dev/disk0",
		Model:        "APPLE SSD AP1024Z",
		CapacityByte: model.NewBigCounterString("1000204886016"),
	})
	stats, err := ParseStorageStats(readIORegFixture(t, "block-storage-driver.plist"))
	if err != nil {
		t.Fatal(err)
	}
	got := MergeStorageStats(snapshot, stats)
	if got.Metrics.HostWritesBytes.Valid {
		t.Fatalf("storage stats should not populate SMART host writes: %#v", got.Metrics.HostWritesBytes)
	}
	if got.Metrics.ReadCommands.Valid {
		t.Fatalf("storage stats should not populate SMART read commands: %#v", got.Metrics.ReadCommands)
	}
	if got.Metrics.MediaErrors.Valid {
		t.Fatalf("storage stats should not populate SMART media errors: %#v", got.Metrics.MediaErrors)
	}
	if !got.Metrics.ControllerBusyMinutes.Valid || got.Metrics.ControllerBusyMinutes.Value.String() != "965" {
		t.Fatalf("controller busy minutes = %#v", got.Metrics.ControllerBusyMinutes)
	}
	got.Metrics.ControllerBusyMinutes = model.Some(model.NewBigCounterString("12"))
	got = MergeStorageStats(got, stats)
	if got.Metrics.ControllerBusyMinutes.Value.String() != "12" {
		t.Fatalf("nonzero controller busy minutes was overwritten: %#v", got.Metrics.ControllerBusyMinutes)
	}
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "system_profiler", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func readIORegFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "ioreg", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
