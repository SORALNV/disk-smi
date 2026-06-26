package native

import (
	"encoding/binary"
	"testing"

	"disk-smi/internal/model"
)

func TestParseNVMeOCPCloudSMARTLog(t *testing.T) {
	data := ocpLogFixture()
	binary.LittleEndian.PutUint64(data[0:8], 123456789)

	log, err := ParseNVMeOCPCloudSMARTLog(data)
	if err != nil {
		t.Fatal(err)
	}
	if log.LogPageVersion != 3 {
		t.Fatalf("log page version = %d", log.LogPageVersion)
	}
	if !log.PhysicalMediaUnitsWritten.Valid || log.PhysicalMediaUnitsWritten.Value.String() != "123456789" {
		t.Fatalf("physical media written = %#v", log.PhysicalMediaUnitsWritten)
	}
}

func TestMergeNVMeOCPCloudSMARTLog(t *testing.T) {
	snapshot := model.SyntheticSnapshot()
	snapshot.Metrics.MediaWritesBytes = model.None[model.BigCounter](model.MissingUnsupported)

	got := MergeNVMeOCPCloudSMARTLog(snapshot, NVMeOCPCloudSMARTLog{
		PhysicalMediaUnitsWritten: model.Some(model.NewBigCounterString("123456789")),
	})

	if !got.Metrics.MediaWritesBytes.Valid || got.Metrics.MediaWritesBytes.Value.String() != "123456789" {
		t.Fatalf("media writes = %#v", got.Metrics.MediaWritesBytes)
	}
}

func TestParseNVMeOCPCloudSMARTLogZeroMediaWritesIsUnsupported(t *testing.T) {
	data := ocpLogFixture()

	log, err := ParseNVMeOCPCloudSMARTLog(data)
	if err != nil {
		t.Fatal(err)
	}
	if log.PhysicalMediaUnitsWritten.Valid || log.PhysicalMediaUnitsWritten.Reason != model.MissingUnsupported {
		t.Fatalf("physical media written = %#v", log.PhysicalMediaUnitsWritten)
	}
}

func TestParseNVMeOCPCloudSMARTLogRejectsWrongGUID(t *testing.T) {
	data := make([]byte, nvmeOCPCloudSMARTLength)

	if _, err := ParseNVMeOCPCloudSMARTLog(data); err == nil {
		t.Fatalf("ParseNVMeOCPCloudSMARTLog accepted missing GUID")
	}
}

func ocpLogFixture() []byte {
	data := make([]byte, nvmeOCPCloudSMARTLength)
	binary.LittleEndian.PutUint16(data[494:496], 3)
	copy(data[496:512], nvmeOCPCloudSMARTGUID)
	return data
}
