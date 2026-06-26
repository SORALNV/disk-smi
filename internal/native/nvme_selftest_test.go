package native

import (
	"encoding/binary"
	"testing"

	"disk-smi/internal/discovery"
	"disk-smi/internal/model"
)

func TestParseNVMeSelfTestLogLatestPassed(t *testing.T) {
	data := make([]byte, nvmeSelfTestLogLength)
	data[4] = 0x10
	binary.LittleEndian.PutUint64(data[8:16], 123)

	log, err := ParseNVMeSelfTestLog(data)
	if err != nil {
		t.Fatal(err)
	}
	if !log.Latest.Valid {
		t.Fatalf("latest = %#v", log.Latest)
	}
	latest := log.Latest.Value
	if latest.Code != 1 || latest.Result != 0 || latest.Status != "Completed without error" || !latest.Passed {
		t.Fatalf("latest = %#v", latest)
	}
	if latest.PowerOnHours.String() != "123" {
		t.Fatalf("poh = %s", latest.PowerOnHours.String())
	}
}

func TestParseNVMeSelfTestLogLatestFailed(t *testing.T) {
	data := make([]byte, nvmeSelfTestLogLength)
	data[4] = 0x27

	log, err := ParseNVMeSelfTestLog(data)
	if err != nil {
		t.Fatal(err)
	}
	if !log.Latest.Valid || log.Latest.Value.Passed {
		t.Fatalf("latest = %#v", log.Latest)
	}
	if log.Latest.Value.Status != "Completed: failed segment known" {
		t.Fatalf("status = %q", log.Latest.Value.Status)
	}
}

func TestParseNVMeSelfTestLogInProgress(t *testing.T) {
	data := make([]byte, nvmeSelfTestLogLength)
	data[0] = 1
	data[1] = 42
	for offset := 4; offset+28 <= nvmeSelfTestLogLength; offset += 28 {
		data[offset] = 0x0f
	}

	log, err := ParseNVMeSelfTestLog(data)
	if err != nil {
		t.Fatal(err)
	}
	if !log.Latest.Valid || log.Latest.Value.Status != "Short self-test in progress (42%)" {
		t.Fatalf("latest = %#v", log.Latest)
	}
	if log.Latest.Value.PassedKnown {
		t.Fatalf("in-progress result should not have pass/fail")
	}
}

func TestMergeNVMeSelfTestLog(t *testing.T) {
	snapshot := Snapshot(discovery.Candidate{
		BSDName:      "disk0",
		DevicePath:   "/dev/disk0",
		Model:        "APPLE SSD AP1024Z",
		CapacityByte: model.NewBigCounterString("1000204886016"),
	})
	got := MergeNVMeSelfTestLog(snapshot, NVMeSelfTestLog{
		Latest: model.Some(NVMeSelfTestResult{
			Status:      "Completed: failed segment known",
			Passed:      false,
			PassedKnown: true,
		}),
	})
	if !got.Metrics.LastSelfTestStatus.Valid || got.Metrics.LastSelfTestStatus.Value != "Completed: failed segment known" {
		t.Fatalf("status = %#v", got.Metrics.LastSelfTestStatus)
	}
	if !got.Metrics.LastSelfTestPassed.Valid || got.Metrics.LastSelfTestPassed.Value {
		t.Fatalf("passed = %#v", got.Metrics.LastSelfTestPassed)
	}
}
