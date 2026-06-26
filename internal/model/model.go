package model

import "math/big"

type Optional[T any] struct {
	Value  T
	Valid  bool
	Reason MissingReason
}

type MissingReason string

const (
	MissingNone        MissingReason = ""
	MissingUnavailable MissingReason = "unavailable"
	MissingUnsupported MissingReason = "unsupported"
	MissingPermission  MissingReason = "permission_required"
	MissingError       MissingReason = "error"
	MissingUnknown     MissingReason = "unknown"
)

func Some[T any](value T) Optional[T] {
	return Optional[T]{Value: value, Valid: true}
}

func None[T any](reason MissingReason) Optional[T] {
	return Optional[T]{Reason: reason}
}

type BigCounter struct {
	value *big.Int
}

func NewBigCounter(value int64) BigCounter {
	return BigCounter{value: big.NewInt(value)}
}

func NewBigCounterString(value string) BigCounter {
	parsed, ok := new(big.Int).SetString(value, 10)
	if !ok {
		parsed = new(big.Int)
	}
	return BigCounter{value: parsed}
}

func (c BigCounter) String() string {
	if c.value == nil {
		return "0"
	}
	return c.value.String()
}

func (c BigCounter) CmpInt64(value int64) int {
	if c.value == nil {
		return big.NewInt(0).Cmp(big.NewInt(value))
	}
	return c.value.Cmp(big.NewInt(value))
}

type OverallStatus string

const (
	StatusGood     OverallStatus = "good"
	StatusCaution  OverallStatus = "caution"
	StatusCritical OverallStatus = "critical"
	StatusUnknown  OverallStatus = "unknown"
)

type DataQuality string

const (
	DataQualityFull        DataQuality = "full"
	DataQualityPartial     DataQuality = "partial"
	DataQualityUnavailable DataQuality = "unavailable"
)

type DeviceInfo struct {
	DevicePath   string
	BSDName      string
	Model        string
	SerialRaw    Optional[string]
	Serial       Optional[string]
	Firmware     Optional[string]
	CapacityByte BigCounter
	Protocol     string
	Transport    Optional[string]
	Location     Optional[string]
	NVMeVersion  Optional[string]
}

type DriveMetrics struct {
	SMARTPassed             Optional[bool]
	CriticalWarning         Optional[uint64]
	TemperatureCelsius      Optional[int64]
	WarningTemperature      Optional[int64]
	CriticalTemperature     Optional[int64]
	TemperatureSensors      []Optional[int64]
	LastSelfTestStatus      Optional[string]
	LastSelfTestPassed      Optional[bool]
	EnduranceUsedPercent    Optional[uint64]
	AvailableSparePercent   Optional[uint64]
	SpareThresholdPercent   Optional[uint64]
	HostReadsBytes          Optional[BigCounter]
	HostWritesBytes         Optional[BigCounter]
	MediaWritesBytes        Optional[BigCounter]
	ReadCommands            Optional[BigCounter]
	WriteCommands           Optional[BigCounter]
	PowerOnHours            Optional[BigCounter]
	PowerCycles             Optional[BigCounter]
	UnsafeShutdowns         Optional[BigCounter]
	ControllerBusyMinutes   Optional[BigCounter]
	MediaErrors             Optional[BigCounter]
	ErrorLogEntries         Optional[BigCounter]
	WarningTemperatureTime  Optional[BigCounter]
	CriticalTemperatureTime Optional[BigCounter]
}

type HealthAssessment struct {
	OverallStatus OverallStatus
	DataQuality   DataQuality
	ReasonCodes   []string
}

type DriveSnapshot struct {
	Device     DeviceInfo
	Metrics    DriveMetrics
	Assessment HealthAssessment
}

type LoopStats struct {
	Valid                bool
	Reset                bool
	ReadRateBytesPerSec  Optional[BigCounter]
	WriteRateBytesPerSec Optional[BigCounter]
	ReadIOPS             Optional[BigCounter]
	WriteIOPS            Optional[BigCounter]
	TemperatureChangeC   Optional[int64]
}

func SyntheticSnapshot() DriveSnapshot {
	return DriveSnapshot{
		Device: DeviceInfo{
			DevicePath:   "/dev/disk0",
			BSDName:      "disk0",
			Model:        "APPLE SSD AP1024Z",
			SerialRaw:    Some("SYNTHETIC9K2A"),
			Serial:       Some("****9K2A"),
			Firmware:     Some("874.120.9"),
			CapacityByte: NewBigCounterString("1000204886016"),
			Protocol:     "NVMe",
			Transport:    Some("PCIe / NVMe"),
			Location:     Some("internal"),
			NVMeVersion:  Some("1.4"),
		},
		Metrics: DriveMetrics{
			SMARTPassed:             Some(true),
			CriticalWarning:         Some(uint64(0)),
			TemperatureCelsius:      Some(int64(36)),
			TemperatureSensors:      []Optional[int64]{Some(int64(36)), Some(int64(41))},
			LastSelfTestStatus:      Some("Completed without error"),
			LastSelfTestPassed:      Some(true),
			EnduranceUsedPercent:    Some(uint64(30)),
			AvailableSparePercent:   Some(uint64(100)),
			SpareThresholdPercent:   Some(uint64(10)),
			HostReadsBytes:          Some(NewBigCounterString("3500000000000")),
			HostWritesBytes:         Some(NewBigCounterString("4200000000000")),
			ReadCommands:            Some(NewBigCounter(82019334)),
			WriteCommands:           Some(NewBigCounter(61274004)),
			PowerOnHours:            Some(NewBigCounter(1500)),
			PowerCycles:             Some(NewBigCounter(428)),
			UnsafeShutdowns:         Some(NewBigCounter(7)),
			ControllerBusyMinutes:   Some(NewBigCounter(6720)),
			MediaErrors:             Some(NewBigCounter(0)),
			ErrorLogEntries:         Some(NewBigCounter(0)),
			WarningTemperatureTime:  Some(NewBigCounter(0)),
			CriticalTemperatureTime: Some(NewBigCounter(0)),
		},
		Assessment: HealthAssessment{
			OverallStatus: StatusGood,
			DataQuality:   DataQualityFull,
		},
	}
}
