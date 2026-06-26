package native

import (
	"fmt"

	"disk-smi/internal/model"
)

const nvmeSelfTestLogLength = 564

type NVMeSelfTestLog struct {
	CurrentOperation  uint8
	CurrentCompletion uint8
	Latest            model.Optional[NVMeSelfTestResult]
}

type NVMeSelfTestResult struct {
	Code         uint8
	Result       uint8
	Status       string
	Passed       bool
	PassedKnown  bool
	PowerOnHours model.BigCounter
}

func ParseNVMeSelfTestLog(data []byte) (NVMeSelfTestLog, error) {
	if len(data) < nvmeSelfTestLogLength {
		return NVMeSelfTestLog{}, fmt.Errorf("NVMe self-test log too short: %d bytes", len(data))
	}
	log := NVMeSelfTestLog{
		CurrentOperation:  data[0] & 0x0f,
		CurrentCompletion: data[1] & 0x7f,
		Latest:            model.None[NVMeSelfTestResult](model.MissingUnsupported),
	}
	for offset := 4; offset+28 <= nvmeSelfTestLogLength; offset += 28 {
		dsts := data[offset]
		result := dsts & 0x0f
		if result == 0x0f {
			continue
		}
		latest := NVMeSelfTestResult{
			Code:         dsts >> 4,
			Result:       result,
			Status:       selfTestResultStatus(result),
			PowerOnHours: le128Counter(data[offset+4:offset+12], 1),
		}
		if result == 0 {
			latest.Passed = true
			latest.PassedKnown = true
		} else {
			latest.Passed = false
			latest.PassedKnown = true
		}
		log.Latest = model.Some(latest)
		return log, nil
	}
	if log.CurrentOperation != 0 {
		status := fmt.Sprintf("%s in progress (%d%%)", selfTestOperation(log.CurrentOperation), log.CurrentCompletion)
		log.Latest = model.Some(NVMeSelfTestResult{
			Code:        log.CurrentOperation,
			Result:      0x0f,
			Status:      status,
			PassedKnown: false,
		})
	}
	return log, nil
}

func MergeNVMeSelfTestLog(snapshot model.DriveSnapshot, log NVMeSelfTestLog) model.DriveSnapshot {
	if !log.Latest.Valid {
		return snapshot
	}
	latest := log.Latest.Value
	if latest.Status != "" {
		snapshot.Metrics.LastSelfTestStatus = model.Some(latest.Status)
	}
	if latest.PassedKnown {
		snapshot.Metrics.LastSelfTestPassed = model.Some(latest.Passed)
	}
	return snapshot
}

func selfTestResultStatus(result uint8) string {
	switch result {
	case 0x0:
		return "Completed without error"
	case 0x1:
		return "Aborted by self-test command"
	case 0x2:
		return "Aborted by controller reset"
	case 0x3:
		return "Aborted by namespace removal"
	case 0x4:
		return "Aborted by format command"
	case 0x5:
		return "Completed: fatal or unknown error"
	case 0x6:
		return "Completed: failed segment unknown"
	case 0x7:
		return "Completed: failed segment known"
	case 0x8:
		return "Aborted for unknown reason"
	case 0x9:
		return "Aborted by sanitize operation"
	default:
		return fmt.Sprintf("Completed: vendor/reserved result 0x%x", result)
	}
}

func selfTestOperation(operation uint8) string {
	switch operation {
	case 0x1:
		return "Short self-test"
	case 0x2:
		return "Extended self-test"
	case 0xe:
		return "Vendor self-test"
	default:
		return fmt.Sprintf("Self-test 0x%x", operation)
	}
}
