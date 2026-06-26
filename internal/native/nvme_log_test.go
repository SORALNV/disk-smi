package native

import (
	"encoding/binary"
	"testing"

	"disk-smi/internal/discovery"
	"disk-smi/internal/model"
)

func TestParseNVMeSMARTLog(t *testing.T) {
	data := make([]byte, nvmeSMARTLogLength)
	data[0] = 8
	binary.LittleEndian.PutUint16(data[1:3], 311)
	data[3] = 100
	data[4] = 99
	data[5] = 7
	putLE128(data[32:48], 11)
	putLE128(data[48:64], 13)
	putLE128(data[64:80], 17)
	putLE128(data[80:96], 19)
	putLE128(data[96:112], 23)
	putLE128(data[112:128], 29)
	putLE128(data[128:144], 31)
	putLE128(data[144:160], 37)
	putLE128(data[160:176], 41)
	putLE128(data[176:192], 43)
	binary.LittleEndian.PutUint32(data[192:196], 47)
	binary.LittleEndian.PutUint32(data[196:200], 53)
	binary.LittleEndian.PutUint16(data[200:202], 310)
	binary.LittleEndian.PutUint16(data[202:204], 315)

	log, err := ParseNVMeSMARTLog(data)
	if err != nil {
		t.Fatal(err)
	}
	if log.CriticalWarning != 8 || log.TemperatureCelsius != 38 {
		t.Fatalf("log = %#v", log)
	}
	if log.DataUnitsRead.String() != "5632000" {
		t.Fatalf("data units read bytes = %s", log.DataUnitsRead.String())
	}
	if log.HostWriteCommands.String() != "19" {
		t.Fatalf("host write commands = %s", log.HostWriteCommands.String())
	}
	if len(log.TemperatureSensors) != 2 || log.TemperatureSensors[1] != 42 {
		t.Fatalf("sensors = %#v", log.TemperatureSensors)
	}
}

func TestMergeNVMeSMARTLog(t *testing.T) {
	snapshot := Snapshot(discovery.Candidate{
		BSDName:      "disk0",
		DevicePath:   "/dev/disk0",
		Model:        "APPLE SSD AP1024Z",
		CapacityByte: model.NewBigCounterString("1000204886016"),
	})
	got := MergeNVMeSMARTLog(snapshot, NVMeSMARTLog{
		CriticalWarning:         0,
		TemperatureCelsius:      40,
		AvailableSparePercent:   100,
		SpareThresholdPercent:   99,
		EnduranceUsedPercent:    0,
		DataUnitsRead:           model.NewBigCounterString("7111818752000"),
		DataUnitsWritten:        model.NewBigCounterString("4189639680000"),
		HostReadCommands:        model.NewBigCounterString("293413990"),
		HostWriteCommands:       model.NewBigCounterString("207048659"),
		ControllerBusyMinutes:   model.NewBigCounter(0),
		PowerCycles:             model.NewBigCounter(115),
		PowerOnHours:            model.NewBigCounter(200),
		UnsafeShutdowns:         model.NewBigCounter(9),
		MediaErrors:             model.NewBigCounter(0),
		ErrorLogEntries:         model.NewBigCounter(0),
		WarningTemperatureTime:  0,
		CriticalTemperatureTime: 0,
		TemperatureSensors:      []int64{40, 42},
	})
	if !got.Metrics.CriticalWarning.Valid || got.Metrics.CriticalWarning.Value != 0 {
		t.Fatalf("critical warning = %#v", got.Metrics.CriticalWarning)
	}
	if !got.Metrics.WarningTemperatureTime.Valid || got.Metrics.WarningTemperatureTime.Value.String() != "0" {
		t.Fatalf("warning temp time = %#v", got.Metrics.WarningTemperatureTime)
	}
	if len(got.Metrics.TemperatureSensors) != 2 || got.Metrics.TemperatureSensors[1].Value != 42 {
		t.Fatalf("temperature sensors = %#v", got.Metrics.TemperatureSensors)
	}
}

func putLE128(data []byte, value uint64) {
	binary.LittleEndian.PutUint64(data[0:8], value)
	binary.LittleEndian.PutUint64(data[8:16], 0)
}
