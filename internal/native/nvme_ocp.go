package native

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math/big"

	"disk-smi/internal/model"
)

const (
	nvmeOCPCloudSMARTLogPage = 0xc0
	nvmeOCPCloudSMARTLength  = 512
)

var nvmeOCPCloudSMARTGUID = []byte{
	0xc5, 0xaf, 0x10, 0x28, 0xea, 0xbf, 0xf2, 0xa4,
	0x9c, 0x4f, 0x6f, 0x7c, 0xc9, 0x14, 0xd5, 0xaf,
}

type NVMeOCPCloudSMARTLog struct {
	PhysicalMediaUnitsWritten model.Optional[model.BigCounter]
	LogPageVersion            uint16
}

func ReadNVMeOCPCloudSMARTLog(ctx context.Context, target string) (NVMeOCPCloudSMARTLog, error) {
	data, err := ReadNVMeLogPageRaw(ctx, target, nvmeOCPCloudSMARTLogPage, nvmeOCPCloudSMARTLength)
	if err != nil {
		return NVMeOCPCloudSMARTLog{}, err
	}
	return ParseNVMeOCPCloudSMARTLog(data)
}

func ParseNVMeOCPCloudSMARTLog(data []byte) (NVMeOCPCloudSMARTLog, error) {
	if len(data) < nvmeOCPCloudSMARTLength {
		return NVMeOCPCloudSMARTLog{}, fmt.Errorf("NVMe OCP Cloud SMART log too short: %d bytes", len(data))
	}
	guid := data[496:512]
	if !bytes.Equal(guid, nvmeOCPCloudSMARTGUID) {
		return NVMeOCPCloudSMARTLog{}, fmt.Errorf("NVMe OCP Cloud SMART log GUID mismatch: got 0x%s", ocpGUIDString(guid))
	}
	written := le128RawCounter(data[0:16])
	log := NVMeOCPCloudSMARTLog{
		PhysicalMediaUnitsWritten: model.None[model.BigCounter](model.MissingUnsupported),
		LogPageVersion:            binary.LittleEndian.Uint16(data[494:496]),
	}
	if written.Sign() > 0 {
		log.PhysicalMediaUnitsWritten = model.Some(model.NewBigCounterString(written.String()))
	}
	return log, nil
}

func MergeNVMeOCPCloudSMARTLog(snapshot model.DriveSnapshot, log NVMeOCPCloudSMARTLog) model.DriveSnapshot {
	if log.PhysicalMediaUnitsWritten.Valid {
		snapshot.Metrics.MediaWritesBytes = log.PhysicalMediaUnitsWritten
	}
	return snapshot
}

func le128RawCounter(data []byte) *big.Int {
	value := new(big.Int)
	for index := len(data) - 1; index >= 0; index-- {
		value.Lsh(value, 8)
		value.Add(value, big.NewInt(int64(data[index])))
	}
	return value
}

func ocpGUIDString(guid []byte) string {
	if len(guid) != 16 {
		return fmt.Sprintf("%x", guid)
	}
	return fmt.Sprintf("%016x%016x",
		binary.LittleEndian.Uint64(guid[8:16]),
		binary.LittleEndian.Uint64(guid[0:8]))
}
