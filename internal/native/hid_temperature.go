package native

import (
	"math"
	"strings"

	"disk-smi/internal/model"
)

type HIDTemperatureSensor struct {
	Name    string
	Celsius float64
}

func MergeHIDTemperatureSensors(snapshot model.DriveSnapshot, sensors []HIDTemperatureSensor) model.DriveSnapshot {
	values := make([]model.Optional[int64], 0, len(sensors))
	for _, sensor := range sensors {
		if !nvmeTemperatureSensorName(sensor.Name) {
			continue
		}
		if math.IsNaN(sensor.Celsius) || math.IsInf(sensor.Celsius, 0) || sensor.Celsius < -100 || sensor.Celsius > 200 {
			continue
		}
		values = append(values, model.Some(int64(math.Round(sensor.Celsius))))
	}
	if len(values) == 0 {
		return snapshot
	}
	snapshot.Metrics.TemperatureSensors = values
	return snapshot
}

func nvmeTemperatureSensorName(name string) bool {
	normalized := strings.ToLower(name)
	return strings.Contains(normalized, "nand") || strings.Contains(normalized, "nvme")
}
