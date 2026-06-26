package native

import (
	"math/big"
	"strings"

	"disk-smi/internal/discovery"
	"disk-smi/internal/model"
)

func Snapshot(candidate discovery.Candidate) model.DriveSnapshot {
	protocol := "unknown"
	if candidate.Protocol.Valid && candidate.Protocol.Value != "" {
		protocol = candidate.Protocol.Value
	}
	snapshot := model.DriveSnapshot{
		Device: model.DeviceInfo{
			DevicePath:   candidate.DevicePath,
			BSDName:      candidate.BSDName,
			Model:        candidate.Model,
			SerialRaw:    model.None[string](model.MissingUnsupported),
			Serial:       model.None[string](model.MissingUnsupported),
			Firmware:     model.None[string](model.MissingUnsupported),
			CapacityByte: candidate.CapacityByte,
			Protocol:     protocol,
			Transport:    candidate.Transport,
			Location:     candidate.Location,
			NVMeVersion:  model.None[string](model.MissingUnsupported),
		},
		Metrics: unsupportedMetrics(),
	}
	return applyDiskutilSMART(snapshot, candidate.SMART)
}

func unsupportedMetrics() model.DriveMetrics {
	reason := model.MissingUnsupported
	return model.DriveMetrics{
		SMARTPassed:             model.None[bool](reason),
		CriticalWarning:         model.None[uint64](reason),
		TemperatureCelsius:      model.None[int64](reason),
		WarningTemperature:      model.None[int64](reason),
		CriticalTemperature:     model.None[int64](reason),
		TemperatureSensors:      []model.Optional[int64]{},
		LastSelfTestStatus:      model.None[string](reason),
		LastSelfTestPassed:      model.None[bool](reason),
		EnduranceUsedPercent:    model.None[uint64](reason),
		AvailableSparePercent:   model.None[uint64](reason),
		SpareThresholdPercent:   model.None[uint64](reason),
		HostReadsBytes:          model.None[model.BigCounter](reason),
		HostWritesBytes:         model.None[model.BigCounter](reason),
		MediaWritesBytes:        model.None[model.BigCounter](reason),
		ReadCommands:            model.None[model.BigCounter](reason),
		WriteCommands:           model.None[model.BigCounter](reason),
		PowerOnHours:            model.None[model.BigCounter](reason),
		PowerCycles:             model.None[model.BigCounter](reason),
		UnsafeShutdowns:         model.None[model.BigCounter](reason),
		ControllerBusyMinutes:   model.None[model.BigCounter](reason),
		MediaErrors:             model.None[model.BigCounter](reason),
		ErrorLogEntries:         model.None[model.BigCounter](reason),
		WarningTemperatureTime:  model.None[model.BigCounter](reason),
		CriticalTemperatureTime: model.None[model.BigCounter](reason),
	}
}

func applyDiskutilSMART(snapshot model.DriveSnapshot, smart discovery.SMARTInfo) model.DriveSnapshot {
	keys := smart.Keys
	if smart.Status != "" && !strings.EqualFold(smart.Status, "Not Supported") {
		snapshot.Metrics.SMARTPassed = model.Some(smartStatusPassed(smart.Status))
	}
	if keys.Temperature != nil && *keys.Temperature >= 273 {
		snapshot.Metrics.TemperatureCelsius = model.Some(int64(*keys.Temperature - 273))
	}
	if keys.PercentageUsed != nil {
		snapshot.Metrics.EnduranceUsedPercent = model.Some(*keys.PercentageUsed)
	}
	if keys.AvailableSpare != nil {
		snapshot.Metrics.AvailableSparePercent = model.Some(*keys.AvailableSpare)
	}
	if keys.AvailableSpareThreshold != nil {
		snapshot.Metrics.SpareThresholdPercent = model.Some(*keys.AvailableSpareThreshold)
	}
	if value, ok := combineDataUnits(keys.DataUnitsRead0, keys.DataUnitsRead1); ok {
		snapshot.Metrics.HostReadsBytes = model.Some(value)
	}
	if value, ok := combineDataUnits(keys.DataUnitsWritten0, keys.DataUnitsWritten1); ok {
		snapshot.Metrics.HostWritesBytes = model.Some(value)
	}
	if value, ok := combineCounter(keys.HostReadCommands0, keys.HostReadCommands1); ok {
		snapshot.Metrics.ReadCommands = model.Some(value)
	}
	if value, ok := combineCounter(keys.HostWriteCommands0, keys.HostWriteCommands1); ok {
		snapshot.Metrics.WriteCommands = model.Some(value)
	}
	if value, ok := combineCounter(keys.PowerOnHours0, keys.PowerOnHours1); ok {
		snapshot.Metrics.PowerOnHours = model.Some(value)
	}
	if value, ok := combineCounter(keys.PowerCycles0, keys.PowerCycles1); ok {
		snapshot.Metrics.PowerCycles = model.Some(value)
	}
	if value, ok := combineCounter(keys.UnsafeShutdowns0, keys.UnsafeShutdowns1); ok {
		snapshot.Metrics.UnsafeShutdowns = model.Some(value)
	}
	if value, ok := combineCounter(keys.ControllerBusyTime0, keys.ControllerBusyTime1); ok {
		snapshot.Metrics.ControllerBusyMinutes = model.Some(value)
	}
	if value, ok := combineCounter(keys.MediaErrors0, keys.MediaErrors1); ok {
		snapshot.Metrics.MediaErrors = model.Some(value)
	}
	if value, ok := combineCounter(keys.NumErrorLogEntries0, keys.NumErrorLogEntries1); ok {
		snapshot.Metrics.ErrorLogEntries = model.Some(value)
	}
	return snapshot
}

func combineDataUnits(low *uint64, high *uint64) (model.BigCounter, bool) {
	value, ok := combineCounterInt(low, high)
	if !ok {
		return model.BigCounter{}, false
	}
	value.Mul(value, big.NewInt(512000))
	return model.NewBigCounterString(value.String()), true
}

func combineCounter(low *uint64, high *uint64) (model.BigCounter, bool) {
	value, ok := combineCounterInt(low, high)
	if !ok {
		return model.BigCounter{}, false
	}
	return model.NewBigCounterString(value.String()), true
}

func combineCounterInt(low *uint64, high *uint64) (*big.Int, bool) {
	if low == nil && high == nil {
		return nil, false
	}
	value := new(big.Int)
	if high != nil {
		value.SetUint64(*high)
		value.Lsh(value, 64)
	}
	if low != nil {
		value.Add(value, new(big.Int).SetUint64(*low))
	}
	return value, true
}
