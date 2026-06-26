package native

import (
	"encoding/binary"
	"testing"

	"disk-smi/internal/discovery"
	"disk-smi/internal/model"
)

func TestParseNVMeIdentify(t *testing.T) {
	data := make([]byte, nvmeIdentifyLength)
	copy(data[4:24], []byte("SERIAL123           "))
	copy(data[24:64], []byte("APPLE SSD AP1024Z                       "))
	copy(data[64:72], []byte("2973.100"))
	binary.LittleEndian.PutUint32(data[80:84], 0x00010400)
	binary.LittleEndian.PutUint16(data[256:258], 1<<4)
	binary.LittleEndian.PutUint16(data[266:268], 343)
	binary.LittleEndian.PutUint16(data[268:270], 353)

	identify, err := ParseNVMeIdentify(data)
	if err != nil {
		t.Fatal(err)
	}
	if identify.Model != "APPLE SSD AP1024Z" || identify.Serial != "SERIAL123" || identify.Firmware != "2973.100" {
		t.Fatalf("identity strings = %#v", identify)
	}
	if identify.Version != "1.4" {
		t.Fatalf("version = %q", identify.Version)
	}
	if !identify.WarningTemperatureCelsius.Valid || identify.WarningTemperatureCelsius.Value != 70 {
		t.Fatalf("warning temperature = %#v", identify.WarningTemperatureCelsius)
	}
	if !identify.CriticalTemperatureCelsius.Valid || identify.CriticalTemperatureCelsius.Value != 80 {
		t.Fatalf("critical temperature = %#v", identify.CriticalTemperatureCelsius)
	}
	if !identify.SelfTestSupported {
		t.Fatalf("self-test support = false")
	}
}

func TestMergeNVMeIdentify(t *testing.T) {
	snapshot := Snapshot(discovery.Candidate{
		BSDName:      "disk0",
		DevicePath:   "/dev/disk0",
		Model:        "UNKNOWN",
		CapacityByte: model.NewBigCounterString("1000204886016"),
	})
	got := MergeNVMeIdentify(snapshot, NVMeIdentify{
		Model:                      "APPLE SSD AP1024Z",
		Serial:                     "SERIAL123",
		Firmware:                   "2973.100",
		Version:                    "1.4",
		WarningTemperatureCelsius:  model.Some(int64(70)),
		CriticalTemperatureCelsius: model.Some(int64(80)),
		SelfTestSupported:          true,
	})
	if got.Device.Model != "APPLE SSD AP1024Z" || !got.Device.NVMeVersion.Valid || got.Device.NVMeVersion.Value != "1.4" {
		t.Fatalf("device = %#v", got.Device)
	}
	if !got.Metrics.WarningTemperature.Valid || got.Metrics.WarningTemperature.Value != 70 {
		t.Fatalf("warning temperature = %#v", got.Metrics.WarningTemperature)
	}
}

func TestMergeNVMeIdentifyPreservesUnsupportedThresholdReasons(t *testing.T) {
	snapshot := Snapshot(discovery.Candidate{
		BSDName:      "disk0",
		DevicePath:   "/dev/disk0",
		Model:        "APPLE SSD AP1024Z",
		CapacityByte: model.NewBigCounterString("1000204886016"),
	})
	got := MergeNVMeIdentify(snapshot, NVMeIdentify{
		WarningTemperatureCelsius:  model.None[int64](model.MissingUnsupported),
		CriticalTemperatureCelsius: model.None[int64](model.MissingUnsupported),
	})
	if got.Metrics.WarningTemperature.Valid || got.Metrics.WarningTemperature.Reason != model.MissingUnsupported {
		t.Fatalf("warning temperature = %#v", got.Metrics.WarningTemperature)
	}
	if got.Metrics.CriticalTemperature.Valid || got.Metrics.CriticalTemperature.Reason != model.MissingUnsupported {
		t.Fatalf("critical temperature = %#v", got.Metrics.CriticalTemperature)
	}
}
