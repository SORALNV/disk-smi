package native

import (
	"testing"

	"disk-smi/internal/model"
)

func TestMergeHIDTemperatureSensorsKeepsNVMeSensors(t *testing.T) {
	snapshot := model.SyntheticSnapshot()
	snapshot.Metrics.TemperatureSensors = nil

	got := MergeHIDTemperatureSensors(snapshot, []HIDTemperatureSensor{
		{Name: "PMU tdev1", Celsius: 51.2},
		{Name: "NAND CH0 temp", Celsius: 42.6},
	})

	if len(got.Metrics.TemperatureSensors) != 1 {
		t.Fatalf("temperature sensor count = %d", len(got.Metrics.TemperatureSensors))
	}
	if !got.Metrics.TemperatureSensors[0].Valid || got.Metrics.TemperatureSensors[0].Value != 43 {
		t.Fatalf("temperature sensor = %#v", got.Metrics.TemperatureSensors[0])
	}
}

func TestMergeHIDTemperatureSensorsLeavesSnapshotWhenNoNVMeSensors(t *testing.T) {
	snapshot := model.SyntheticSnapshot()
	snapshot.Metrics.TemperatureSensors = nil

	got := MergeHIDTemperatureSensors(snapshot, []HIDTemperatureSensor{
		{Name: "PMU tdev1", Celsius: 51.2},
	})

	if len(got.Metrics.TemperatureSensors) != 0 {
		t.Fatalf("temperature sensors = %#v", got.Metrics.TemperatureSensors)
	}
}
