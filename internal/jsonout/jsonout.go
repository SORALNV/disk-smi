package jsonout

import (
	"encoding/json"
	"strconv"
	"time"

	"disk-smi/internal/model"
)

type document struct {
	SchemaVersion int          `json:"schema_version"`
	GeneratedAt   string       `json:"generated_at"`
	DisplayLocale string       `json:"display_locale"`
	Drives        []driveEntry `json:"drives"`
}

type driveEntry struct {
	Device         deviceEntry       `json:"device"`
	Health         healthEntry       `json:"health"`
	Endurance      enduranceEntry    `json:"endurance"`
	Thermals       thermalsEntry     `json:"thermals"`
	Reliability    reliabilityEntry  `json:"reliability"`
	IO             ioEntry           `json:"io"`
	Power          powerEntry        `json:"power"`
	MissingReasons map[string]string `json:"missing_reasons,omitempty"`
}

type deviceEntry struct {
	Path         string  `json:"path"`
	Name         string  `json:"name"`
	Model        string  `json:"model"`
	CapacityByte string  `json:"capacity_bytes"`
	Protocol     string  `json:"protocol"`
	Firmware     *string `json:"firmware"`
	Transport    *string `json:"transport"`
	Location     *string `json:"location"`
	NVMeVersion  *string `json:"nvme_version"`
	SerialMasked *string `json:"serial_masked"`
	Serial       *string `json:"serial,omitempty"`
}

type healthEntry struct {
	OverallStatus string   `json:"overall_status"`
	DataQuality   string   `json:"data_quality"`
	ReasonCodes   []string `json:"reason_codes"`
}

type enduranceEntry struct {
	UsedPercent            *uint64 `json:"used_percent"`
	RemainingPercent       *uint64 `json:"remaining_percent"`
	AvailableSparePercent  *uint64 `json:"available_spare_percent"`
	SpareThresholdPercent  *uint64 `json:"spare_threshold_percent"`
	IsFailureProbability   bool    `json:"remaining_is_failure_probability"`
	ZeroMeansImmediateFail bool    `json:"zero_means_immediate_failure"`
}

type thermalsEntry struct {
	TemperatureCelsius      *int64   `json:"temperature_celsius"`
	WarningTemperature      *int64   `json:"warning_temperature_celsius"`
	CriticalTemperature     *int64   `json:"critical_temperature_celsius"`
	TemperatureSensors      []*int64 `json:"temperature_sensors_celsius"`
	WarningTemperatureTime  *string  `json:"warning_temperature_time_minutes"`
	CriticalTemperatureTime *string  `json:"critical_temperature_time_minutes"`
}

type reliabilityEntry struct {
	SmartPassed        *bool   `json:"smart_passed"`
	CriticalWarning    *uint64 `json:"critical_warning"`
	MediaErrors        *string `json:"media_errors"`
	ErrorLogEntries    *string `json:"error_log_entries"`
	LastSelfTestStatus *string `json:"last_self_test_status"`
	LastSelfTestPassed *bool   `json:"last_self_test_passed"`
}

type ioEntry struct {
	HostReadsBytes        *string `json:"host_reads_bytes"`
	HostWritesBytes       *string `json:"host_writes_bytes"`
	MediaWritesBytes      *string `json:"media_writes_bytes"`
	ReadCommands          *string `json:"read_commands"`
	WriteCommands         *string `json:"write_commands"`
	ControllerBusyMinutes *string `json:"controller_busy_minutes"`
}

type powerEntry struct {
	PowerOnHours    *string `json:"power_on_hours"`
	PowerCycles     *string `json:"power_cycles"`
	UnsafeShutdowns *string `json:"unsafe_shutdowns"`
}

type Options struct {
	Pretty     bool
	ShowSerial bool
}

func Render(snapshots []model.DriveSnapshot, locale string, generatedAt time.Time, opts Options) (string, error) {
	doc := document{
		SchemaVersion: 1,
		GeneratedAt:   generatedAt.Format(time.RFC3339),
		DisplayLocale: locale,
		Drives:        make([]driveEntry, 0, len(snapshots)),
	}
	for _, snapshot := range snapshots {
		doc.Drives = append(doc.Drives, drive(snapshot, opts))
	}

	var (
		data []byte
		err  error
	)
	if opts.Pretty {
		data, err = json.MarshalIndent(doc, "", "  ")
	} else {
		data, err = json.Marshal(doc)
	}
	if err != nil {
		return "", err
	}
	return string(data) + "\n", nil
}

func drive(snapshot model.DriveSnapshot, opts Options) driveEntry {
	missing := missingReasons(snapshot)
	return driveEntry{
		Device: deviceEntry{
			Path:         snapshot.Device.DevicePath,
			Name:         snapshot.Device.BSDName,
			Model:        snapshot.Device.Model,
			CapacityByte: snapshot.Device.CapacityByte.String(),
			Protocol:     snapshot.Device.Protocol,
			Firmware:     optionalString(snapshot.Device.Firmware),
			Transport:    optionalString(snapshot.Device.Transport),
			Location:     optionalString(snapshot.Device.Location),
			NVMeVersion:  optionalString(snapshot.Device.NVMeVersion),
			SerialMasked: optionalString(snapshot.Device.Serial),
			Serial:       rawSerial(snapshot, opts),
		},
		Health: healthEntry{
			OverallStatus: string(snapshot.Assessment.OverallStatus),
			DataQuality:   string(snapshot.Assessment.DataQuality),
			ReasonCodes:   snapshot.Assessment.ReasonCodes,
		},
		Endurance: enduranceEntry{
			UsedPercent:            optionalUint(snapshot.Metrics.EnduranceUsedPercent),
			RemainingPercent:       enduranceRemaining(snapshot.Metrics.EnduranceUsedPercent),
			AvailableSparePercent:  optionalUint(snapshot.Metrics.AvailableSparePercent),
			SpareThresholdPercent:  optionalUint(snapshot.Metrics.SpareThresholdPercent),
			IsFailureProbability:   false,
			ZeroMeansImmediateFail: false,
		},
		Thermals: thermalsEntry{
			TemperatureCelsius:      optionalInt(snapshot.Metrics.TemperatureCelsius),
			WarningTemperature:      optionalInt(snapshot.Metrics.WarningTemperature),
			CriticalTemperature:     optionalInt(snapshot.Metrics.CriticalTemperature),
			TemperatureSensors:      optionalIntSlice(snapshot.Metrics.TemperatureSensors),
			WarningTemperatureTime:  optionalCounter(snapshot.Metrics.WarningTemperatureTime),
			CriticalTemperatureTime: optionalCounter(snapshot.Metrics.CriticalTemperatureTime),
		},
		Reliability: reliabilityEntry{
			SmartPassed:        optionalBool(snapshot.Metrics.SMARTPassed),
			CriticalWarning:    optionalUint(snapshot.Metrics.CriticalWarning),
			MediaErrors:        optionalCounter(snapshot.Metrics.MediaErrors),
			ErrorLogEntries:    optionalCounter(snapshot.Metrics.ErrorLogEntries),
			LastSelfTestStatus: optionalString(snapshot.Metrics.LastSelfTestStatus),
			LastSelfTestPassed: optionalBool(snapshot.Metrics.LastSelfTestPassed),
		},
		IO: ioEntry{
			HostReadsBytes:        optionalCounter(snapshot.Metrics.HostReadsBytes),
			HostWritesBytes:       optionalCounter(snapshot.Metrics.HostWritesBytes),
			MediaWritesBytes:      optionalCounter(snapshot.Metrics.MediaWritesBytes),
			ReadCommands:          optionalCounter(snapshot.Metrics.ReadCommands),
			WriteCommands:         optionalCounter(snapshot.Metrics.WriteCommands),
			ControllerBusyMinutes: optionalCounter(snapshot.Metrics.ControllerBusyMinutes),
		},
		Power: powerEntry{
			PowerOnHours:    optionalCounter(snapshot.Metrics.PowerOnHours),
			PowerCycles:     optionalCounter(snapshot.Metrics.PowerCycles),
			UnsafeShutdowns: optionalCounter(snapshot.Metrics.UnsafeShutdowns),
		},
		MissingReasons: missing,
	}
}

func rawSerial(snapshot model.DriveSnapshot, opts Options) *string {
	if !opts.ShowSerial || !snapshot.Device.SerialRaw.Valid {
		return nil
	}
	return &snapshot.Device.SerialRaw.Value
}

func optionalString(value model.Optional[string]) *string {
	if !value.Valid {
		return nil
	}
	return &value.Value
}

func optionalUint(value model.Optional[uint64]) *uint64 {
	if !value.Valid {
		return nil
	}
	return &value.Value
}

func optionalInt(value model.Optional[int64]) *int64 {
	if !value.Valid {
		return nil
	}
	return &value.Value
}

func optionalBool(value model.Optional[bool]) *bool {
	if !value.Valid {
		return nil
	}
	return &value.Value
}

func optionalIntSlice(values []model.Optional[int64]) []*int64 {
	if len(values) == 0 {
		return nil
	}
	out := make([]*int64, 0, len(values))
	for _, value := range values {
		out = append(out, optionalInt(value))
	}
	return out
}

func optionalCounter(value model.Optional[model.BigCounter]) *string {
	if !value.Valid {
		return nil
	}
	out := value.Value.String()
	return &out
}

func enduranceRemaining(value model.Optional[uint64]) *uint64 {
	if !value.Valid {
		return nil
	}
	var remaining uint64
	if value.Value < 100 {
		remaining = 100 - value.Value
	}
	return &remaining
}

func missingReasons(snapshot model.DriveSnapshot) map[string]string {
	reasons := map[string]string{}
	addStringReason(reasons, "device.firmware", snapshot.Device.Firmware)
	addStringReason(reasons, "device.transport", snapshot.Device.Transport)
	addStringReason(reasons, "device.location", snapshot.Device.Location)
	addStringReason(reasons, "device.nvme_version", snapshot.Device.NVMeVersion)
	addBoolReason(reasons, "reliability.smart_passed", snapshot.Metrics.SMARTPassed)
	addUintReason(reasons, "reliability.critical_warning", snapshot.Metrics.CriticalWarning)
	addCounterReason(reasons, "reliability.media_errors", snapshot.Metrics.MediaErrors)
	addCounterReason(reasons, "reliability.error_log_entries", snapshot.Metrics.ErrorLogEntries)
	addStringReason(reasons, "reliability.last_self_test_status", snapshot.Metrics.LastSelfTestStatus)
	addBoolReason(reasons, "reliability.last_self_test_passed", snapshot.Metrics.LastSelfTestPassed)
	addUintReason(reasons, "endurance.used_percent", snapshot.Metrics.EnduranceUsedPercent)
	addUintReason(reasons, "endurance.available_spare_percent", snapshot.Metrics.AvailableSparePercent)
	addUintReason(reasons, "endurance.spare_threshold_percent", snapshot.Metrics.SpareThresholdPercent)
	addIntReason(reasons, "thermals.temperature_celsius", snapshot.Metrics.TemperatureCelsius)
	addIntReason(reasons, "thermals.warning_temperature_celsius", snapshot.Metrics.WarningTemperature)
	addIntReason(reasons, "thermals.critical_temperature_celsius", snapshot.Metrics.CriticalTemperature)
	if len(snapshot.Metrics.TemperatureSensors) == 0 {
		reasons["thermals.temperature_sensors_celsius"] = string(model.MissingUnsupported)
	}
	for index, value := range snapshot.Metrics.TemperatureSensors {
		addIntReason(reasons, "thermals.temperature_sensors_celsius."+strconv.Itoa(index), value)
	}
	addCounterReason(reasons, "thermals.warning_temperature_time_minutes", snapshot.Metrics.WarningTemperatureTime)
	addCounterReason(reasons, "thermals.critical_temperature_time_minutes", snapshot.Metrics.CriticalTemperatureTime)
	addCounterReason(reasons, "io.host_reads_bytes", snapshot.Metrics.HostReadsBytes)
	addCounterReason(reasons, "io.host_writes_bytes", snapshot.Metrics.HostWritesBytes)
	addCounterReason(reasons, "io.media_writes_bytes", snapshot.Metrics.MediaWritesBytes)
	addCounterReason(reasons, "io.read_commands", snapshot.Metrics.ReadCommands)
	addCounterReason(reasons, "io.write_commands", snapshot.Metrics.WriteCommands)
	addCounterReason(reasons, "io.controller_busy_minutes", snapshot.Metrics.ControllerBusyMinutes)
	addCounterReason(reasons, "power.power_on_hours", snapshot.Metrics.PowerOnHours)
	addCounterReason(reasons, "power.power_cycles", snapshot.Metrics.PowerCycles)
	addCounterReason(reasons, "power.unsafe_shutdowns", snapshot.Metrics.UnsafeShutdowns)
	if len(reasons) == 0 {
		return nil
	}
	return reasons
}

func addStringReason(reasons map[string]string, key string, value model.Optional[string]) {
	if !value.Valid {
		addReason(reasons, key, value.Reason)
	}
}

func addBoolReason(reasons map[string]string, key string, value model.Optional[bool]) {
	if !value.Valid {
		addReason(reasons, key, value.Reason)
	}
}

func addUintReason(reasons map[string]string, key string, value model.Optional[uint64]) {
	if !value.Valid {
		addReason(reasons, key, value.Reason)
	}
}

func addIntReason(reasons map[string]string, key string, value model.Optional[int64]) {
	if !value.Valid {
		addReason(reasons, key, value.Reason)
	}
}

func addCounterReason(reasons map[string]string, key string, value model.Optional[model.BigCounter]) {
	if !value.Valid {
		addReason(reasons, key, value.Reason)
	}
}

func addReason(reasons map[string]string, key string, reason model.MissingReason) {
	if reason == model.MissingNone {
		reason = model.MissingUnknown
	}
	reasons[key] = string(reason)
}
