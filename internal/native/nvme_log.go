package native

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"disk-smi/internal/model"
)

const nvmeSMARTLogLength = 512

type NVMeSMARTLog struct {
	CriticalWarning         uint8
	TemperatureCelsius      int64
	AvailableSparePercent   uint8
	SpareThresholdPercent   uint8
	EnduranceUsedPercent    uint8
	DataUnitsRead           model.BigCounter
	DataUnitsWritten        model.BigCounter
	HostReadCommands        model.BigCounter
	HostWriteCommands       model.BigCounter
	ControllerBusyMinutes   model.BigCounter
	PowerCycles             model.BigCounter
	PowerOnHours            model.BigCounter
	UnsafeShutdowns         model.BigCounter
	MediaErrors             model.BigCounter
	ErrorLogEntries         model.BigCounter
	WarningTemperatureTime  uint32
	CriticalTemperatureTime uint32
	TemperatureSensors      []int64
}

func ParseNVMeSMARTLog(data []byte) (NVMeSMARTLog, error) {
	if len(data) < nvmeSMARTLogLength {
		return NVMeSMARTLog{}, fmt.Errorf("NVMe SMART log too short: %d bytes", len(data))
	}
	temperatureKelvin := binary.LittleEndian.Uint16(data[1:3])
	log := NVMeSMARTLog{
		CriticalWarning:         data[0],
		TemperatureCelsius:      int64(temperatureKelvin) - 273,
		AvailableSparePercent:   data[3],
		SpareThresholdPercent:   data[4],
		EnduranceUsedPercent:    data[5],
		DataUnitsRead:           le128Counter(data[32:48], 512000),
		DataUnitsWritten:        le128Counter(data[48:64], 512000),
		HostReadCommands:        le128Counter(data[64:80], 1),
		HostWriteCommands:       le128Counter(data[80:96], 1),
		ControllerBusyMinutes:   le128Counter(data[96:112], 1),
		PowerCycles:             le128Counter(data[112:128], 1),
		PowerOnHours:            le128Counter(data[128:144], 1),
		UnsafeShutdowns:         le128Counter(data[144:160], 1),
		MediaErrors:             le128Counter(data[160:176], 1),
		ErrorLogEntries:         le128Counter(data[176:192], 1),
		WarningTemperatureTime:  binary.LittleEndian.Uint32(data[192:196]),
		CriticalTemperatureTime: binary.LittleEndian.Uint32(data[196:200]),
	}
	for offset := 200; offset < 216; offset += 2 {
		kelvin := binary.LittleEndian.Uint16(data[offset : offset+2])
		if kelvin == 0 {
			continue
		}
		log.TemperatureSensors = append(log.TemperatureSensors, int64(kelvin)-273)
	}
	return log, nil
}

func MergeNVMeSMARTLog(snapshot model.DriveSnapshot, log NVMeSMARTLog) model.DriveSnapshot {
	snapshot.Metrics.CriticalWarning = model.Some(uint64(log.CriticalWarning))
	snapshot.Metrics.TemperatureCelsius = model.Some(log.TemperatureCelsius)
	snapshot.Metrics.AvailableSparePercent = model.Some(uint64(log.AvailableSparePercent))
	snapshot.Metrics.SpareThresholdPercent = model.Some(uint64(log.SpareThresholdPercent))
	snapshot.Metrics.EnduranceUsedPercent = model.Some(uint64(log.EnduranceUsedPercent))
	snapshot.Metrics.HostReadsBytes = model.Some(log.DataUnitsRead)
	snapshot.Metrics.HostWritesBytes = model.Some(log.DataUnitsWritten)
	snapshot.Metrics.ReadCommands = model.Some(log.HostReadCommands)
	snapshot.Metrics.WriteCommands = model.Some(log.HostWriteCommands)
	snapshot.Metrics.ControllerBusyMinutes = model.Some(log.ControllerBusyMinutes)
	snapshot.Metrics.PowerCycles = model.Some(log.PowerCycles)
	snapshot.Metrics.PowerOnHours = model.Some(log.PowerOnHours)
	snapshot.Metrics.UnsafeShutdowns = model.Some(log.UnsafeShutdowns)
	snapshot.Metrics.MediaErrors = model.Some(log.MediaErrors)
	snapshot.Metrics.ErrorLogEntries = model.Some(log.ErrorLogEntries)
	snapshot.Metrics.WarningTemperatureTime = model.Some(model.NewBigCounterString(fmt.Sprintf("%d", log.WarningTemperatureTime)))
	snapshot.Metrics.CriticalTemperatureTime = model.Some(model.NewBigCounterString(fmt.Sprintf("%d", log.CriticalTemperatureTime)))
	snapshot.Metrics.TemperatureSensors = make([]model.Optional[int64], 0, len(log.TemperatureSensors))
	for _, value := range log.TemperatureSensors {
		snapshot.Metrics.TemperatureSensors = append(snapshot.Metrics.TemperatureSensors, model.Some(value))
	}
	return snapshot
}

func le128Counter(data []byte, multiplier int64) model.BigCounter {
	value := new(big.Int)
	for index := len(data) - 1; index >= 0; index-- {
		value.Lsh(value, 8)
		value.Add(value, big.NewInt(int64(data[index])))
	}
	if multiplier != 1 {
		value.Mul(value, big.NewInt(multiplier))
	}
	return model.NewBigCounterString(value.String())
}
