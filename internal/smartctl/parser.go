package smartctl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"path/filepath"
	"strings"

	"disk-smi/internal/model"
)

type document struct {
	Device struct {
		Name     string `json:"name"`
		Protocol string `json:"protocol"`
		Type     string `json:"type"`
	} `json:"device"`
	ModelName       string `json:"model_name"`
	SerialNumber    string `json:"serial_number"`
	FirmwareVersion string `json:"firmware_version"`
	TotalCapacity   uint64 `json:"nvme_total_capacity"`
	UserCapacity    struct {
		Bytes uint64 `json:"bytes"`
	} `json:"user_capacity"`
	NVMeVersion struct {
		String string `json:"string"`
	} `json:"nvme_version"`
	SmartStatus struct {
		Passed *bool `json:"passed"`
	} `json:"smart_status"`
	Temperature struct {
		Current *int64 `json:"current"`
	} `json:"temperature"`
	NVMeOptionalAdminCommands struct {
		SelfTest *bool `json:"self_test"`
	} `json:"nvme_optional_admin_commands"`
	NVMeLog                nvmeLog      `json:"nvme_smart_health_information_log"`
	NVMeIdentifyController nvmeIdentify `json:"nvme_identify_controller"`
	NVMeSelfTestLog        selfTestLog  `json:"nvme_self_test_log"`
	ATA                    ataInfo      `json:"ata_smart_attributes"`
	ATASelfTestLog         selfTestLog  `json:"ata_smart_self_test_log"`
}

type nvmeLog struct {
	CriticalWarning       *uint64  `json:"critical_warning"`
	Temperature           *int64   `json:"temperature"`
	WarningTemperature    *int64   `json:"warning_temperature"`
	CriticalTemperature   *int64   `json:"critical_temperature"`
	AvailableSpare        *uint64  `json:"available_spare"`
	AvailableSpareThresh  *uint64  `json:"available_spare_threshold"`
	PercentageUsed        *uint64  `json:"percentage_used"`
	DataUnitsRead         *decimal `json:"data_units_read"`
	DataUnitsWritten      *decimal `json:"data_units_written"`
	HostReadCommands      *decimal `json:"host_read_commands"`
	HostWriteCommands     *decimal `json:"host_write_commands"`
	HostReads             *decimal `json:"host_reads"`
	HostWrites            *decimal `json:"host_writes"`
	ControllerBusyTime    *decimal `json:"controller_busy_time"`
	PowerCycles           *decimal `json:"power_cycles"`
	PowerOnHours          *decimal `json:"power_on_hours"`
	UnsafeShutdowns       *decimal `json:"unsafe_shutdowns"`
	MediaErrors           *decimal `json:"media_errors"`
	NumErrLogEntries      *decimal `json:"num_err_log_entries"`
	WarningTempTime       *decimal `json:"warning_temp_time"`
	CriticalCompTime      *decimal `json:"critical_comp_time"`
	TemperatureSensors    []int64  `json:"temperature_sensors"`
	TemperatureSensorsAlt []int64  `json:"temperature_sensors_celsius"`
}

type nvmeIdentify struct {
	WarningCompositeTemperature  *int64 `json:"wctemp"`
	CriticalCompositeTemperature *int64 `json:"cctemp"`
	WarningTemperature           *int64 `json:"warning_temperature"`
	CriticalTemperature          *int64 `json:"critical_temperature"`
}

type ataInfo struct {
	Table []ataAttribute `json:"table"`
}

type ataAttribute struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Value int64  `json:"value"`
	Raw   struct {
		Value  *decimal `json:"value"`
		String string   `json:"string"`
	} `json:"raw"`
}

type selfTestLog struct {
	Standard struct {
		Table []selfTestEntry `json:"table"`
	} `json:"standard"`
	SelfTests []selfTestEntry `json:"self_tests"`
}

type selfTestEntry struct {
	Status selfTestStatus `json:"status"`
}

type selfTestStatus struct {
	String string
	Passed *bool
}

func (s *selfTestStatus) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		s.String = text
		return nil
	}
	var object struct {
		String string `json:"string"`
		Passed *bool  `json:"passed"`
	}
	if err := json.Unmarshal(data, &object); err != nil {
		return err
	}
	s.String = object.String
	s.Passed = object.Passed
	return nil
}

type decimal struct {
	value *big.Int
}

func (d *decimal) UnmarshalJSON(data []byte) error {
	text := strings.Trim(string(data), `"`)
	if text == "" || text == "null" {
		d.value = nil
		return nil
	}
	parsed, ok := new(big.Int).SetString(text, 10)
	if !ok {
		return fmt.Errorf("invalid decimal value %q", text)
	}
	d.value = parsed
	return nil
}

func Parse(data []byte, fallbackDevice string) (model.DriveSnapshot, error) {
	var doc document
	if err := json.Unmarshal(data, &doc); err != nil {
		return model.DriveSnapshot{}, fmt.Errorf("parse smartctl JSON: %w", err)
	}

	devicePath, bsdName := deviceNames(doc.Device.Name, fallbackDevice)
	capacity := doc.TotalCapacity
	if capacity == 0 {
		capacity = doc.UserCapacity.Bytes
	}
	protocol := doc.Device.Protocol
	if protocol == "" {
		protocol = strings.ToUpper(doc.Device.Type)
	}
	if protocol == "" {
		protocol = "NVMe"
	}

	snapshot := model.DriveSnapshot{
		Device: model.DeviceInfo{
			DevicePath:   devicePath,
			BSDName:      bsdName,
			Model:        fallbackString(doc.ModelName, "UNKNOWN"),
			SerialRaw:    optionalString(doc.SerialNumber, model.MissingUnavailable),
			Serial:       optionalString(maskSerial(doc.SerialNumber), model.MissingUnavailable),
			Firmware:     optionalString(doc.FirmwareVersion, model.MissingUnavailable),
			CapacityByte: model.NewBigCounterString(fmt.Sprintf("%d", capacity)),
			Protocol:     protocol,
			Transport:    model.Some("PCIe / NVMe"),
			Location:     model.None[string](model.MissingUnknown),
			NVMeVersion:  optionalString(doc.NVMeVersion.String, model.MissingUnavailable),
		},
		Metrics: model.DriveMetrics{
			SMARTPassed:             optionalBool(doc.SmartStatus.Passed, model.MissingUnavailable),
			CriticalWarning:         optionalUint(doc.NVMeLog.CriticalWarning, model.MissingUnavailable),
			TemperatureCelsius:      optionalInt(doc.NVMeLog.Temperature, model.MissingUnavailable),
			WarningTemperature:      optionalTemperature(firstInt(doc.NVMeLog.WarningTemperature, doc.NVMeIdentifyController.WarningTemperature, doc.NVMeIdentifyController.WarningCompositeTemperature), model.MissingUnavailable),
			CriticalTemperature:     optionalTemperature(firstInt(doc.NVMeLog.CriticalTemperature, doc.NVMeIdentifyController.CriticalTemperature, doc.NVMeIdentifyController.CriticalCompositeTemperature), model.MissingUnavailable),
			AvailableSparePercent:   optionalUint(doc.NVMeLog.AvailableSpare, model.MissingUnavailable),
			SpareThresholdPercent:   optionalUint(doc.NVMeLog.AvailableSpareThresh, model.MissingUnavailable),
			EnduranceUsedPercent:    optionalUint(doc.NVMeLog.PercentageUsed, model.MissingUnavailable),
			HostReadsBytes:          optionalDataUnits(doc.NVMeLog.DataUnitsRead),
			HostWritesBytes:         optionalDataUnits(doc.NVMeLog.DataUnitsWritten),
			MediaWritesBytes:        model.None[model.BigCounter](model.MissingUnsupported),
			ReadCommands:            optionalCounter(firstDecimal(doc.NVMeLog.HostReadCommands, doc.NVMeLog.HostReads)),
			WriteCommands:           optionalCounter(firstDecimal(doc.NVMeLog.HostWriteCommands, doc.NVMeLog.HostWrites)),
			ControllerBusyMinutes:   optionalCounter(doc.NVMeLog.ControllerBusyTime),
			PowerCycles:             optionalCounter(doc.NVMeLog.PowerCycles),
			PowerOnHours:            optionalCounter(doc.NVMeLog.PowerOnHours),
			UnsafeShutdowns:         optionalCounter(doc.NVMeLog.UnsafeShutdowns),
			MediaErrors:             optionalCounter(doc.NVMeLog.MediaErrors),
			ErrorLogEntries:         optionalCounter(doc.NVMeLog.NumErrLogEntries),
			WarningTemperatureTime:  optionalCounter(doc.NVMeLog.WarningTempTime),
			CriticalTemperatureTime: optionalCounter(doc.NVMeLog.CriticalCompTime),
			TemperatureSensors:      optionalSensors(doc.NVMeLog),
			LastSelfTestStatus:      model.None[string](model.MissingUnavailable),
			LastSelfTestPassed:      model.None[bool](model.MissingUnavailable),
		},
	}
	applyATA(&snapshot, doc)
	applySelfTest(&snapshot, doc)
	applyPhysicalMediaUnitsWritten(&snapshot, data)

	return snapshot, nil
}

func applyATA(snapshot *model.DriveSnapshot, doc document) {
	if !strings.EqualFold(snapshot.Device.Protocol, "ATA") && !strings.EqualFold(snapshot.Device.Protocol, "SATA") {
		return
	}
	snapshot.Device.Transport = model.Some("SATA")
	if doc.Temperature.Current != nil {
		snapshot.Metrics.TemperatureCelsius = model.Some(*doc.Temperature.Current)
	}
	for _, attr := range doc.ATA.Table {
		raw := ataRaw(attr)
		if raw == "" {
			continue
		}
		switch {
		case attr.ID == 9 || attrName(attr.Name, "Power_On_Hours"):
			snapshot.Metrics.PowerOnHours = model.Some(model.NewBigCounterString(raw))
		case attr.ID == 12 || attrName(attr.Name, "Power_Cycle_Count"):
			snapshot.Metrics.PowerCycles = model.Some(model.NewBigCounterString(raw))
		case attr.ID == 194 || attr.ID == 190 || attrName(attr.Name, "Temperature"):
			if value, ok := new(big.Int).SetString(raw, 10); ok {
				snapshot.Metrics.TemperatureCelsius = model.Some(value.Int64())
			}
		case attr.ID == 177 || attr.ID == 202 || attrName(attr.Name, "Percent_Lifetime"):
			if value, ok := new(big.Int).SetString(raw, 10); ok && value.Sign() >= 0 {
				snapshot.Metrics.EnduranceUsedPercent = model.Some(value.Uint64())
			}
		case attr.ID == 5 || attr.ID == 187 || attr.ID == 198 || attrName(attr.Name, "Reallocated") || attrName(attr.Name, "Uncorrectable"):
			snapshot.Metrics.MediaErrors = model.Some(model.NewBigCounterString(raw))
		}
	}
}

func ataRaw(attr ataAttribute) string {
	if attr.Raw.Value != nil && attr.Raw.Value.value != nil {
		return attr.Raw.Value.value.String()
	}
	fields := strings.Fields(attr.Raw.String)
	if len(fields) == 0 {
		return ""
	}
	if _, ok := new(big.Int).SetString(fields[0], 10); ok {
		return fields[0]
	}
	return ""
}

func attrName(name string, needle string) bool {
	return strings.Contains(strings.ToLower(name), strings.ToLower(needle))
}

func applySelfTest(snapshot *model.DriveSnapshot, doc document) {
	entry, ok := latestSelfTest(doc.NVMeSelfTestLog)
	if !ok {
		entry, ok = latestSelfTest(doc.ATASelfTestLog)
	}
	if !ok {
		if doc.NVMeOptionalAdminCommands.SelfTest != nil && !*doc.NVMeOptionalAdminCommands.SelfTest {
			snapshot.Metrics.LastSelfTestStatus = model.None[string](model.MissingUnsupported)
			snapshot.Metrics.LastSelfTestPassed = model.None[bool](model.MissingUnsupported)
		}
		return
	}
	if entry.Status.String != "" {
		snapshot.Metrics.LastSelfTestStatus = model.Some(entry.Status.String)
	}
	if entry.Status.Passed != nil {
		snapshot.Metrics.LastSelfTestPassed = model.Some(*entry.Status.Passed)
		return
	}
	if passed, known := selfTestPassed(entry.Status.String); known {
		snapshot.Metrics.LastSelfTestPassed = model.Some(passed)
	}
}

func applyPhysicalMediaUnitsWritten(snapshot *model.DriveSnapshot, data []byte) {
	if snapshot.Metrics.MediaWritesBytes.Valid {
		return
	}
	written, ok := physicalMediaUnitsWritten(data)
	if !ok || written.Sign() <= 0 {
		return
	}
	snapshot.Metrics.MediaWritesBytes = model.Some(model.NewBigCounterString(written.String()))
}

func physicalMediaUnitsWritten(data []byte) (*big.Int, bool) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var payload any
	if err := decoder.Decode(&payload); err != nil {
		return nil, false
	}
	return findPhysicalMediaUnitsWritten(payload)
}

func findPhysicalMediaUnitsWritten(value any) (*big.Int, bool) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if normalizedJSONKey(key) == "physical_media_units_written" {
				if parsed, ok := parseBigCounterJSONValue(child); ok {
					return parsed, true
				}
			}
		}
		for _, child := range typed {
			if parsed, ok := findPhysicalMediaUnitsWritten(child); ok {
				return parsed, true
			}
		}
	case []any:
		for _, child := range typed {
			if parsed, ok := findPhysicalMediaUnitsWritten(child); ok {
				return parsed, true
			}
		}
	}
	return nil, false
}

func parseBigCounterJSONValue(value any) (*big.Int, bool) {
	switch typed := value.(type) {
	case json.Number:
		return parseNonNegativeBigInt(typed.String())
	case string:
		return parseNonNegativeBigInt(typed)
	case map[string]any:
		if combined, ok := parseHiLoBigCounter(typed); ok {
			return combined, true
		}
		for _, key := range []string{"value", "raw", "bytes"} {
			if child, ok := typed[key]; ok {
				if parsed, ok := parseBigCounterJSONValue(child); ok {
					return parsed, true
				}
			}
		}
	}
	return nil, false
}

func parseHiLoBigCounter(value map[string]any) (*big.Int, bool) {
	hiValue, hiOK := value["hi"]
	loValue, loOK := value["lo"]
	if !hiOK || !loOK {
		return nil, false
	}
	hi, hiOK := parseBigCounterJSONValue(hiValue)
	lo, loOK := parseBigCounterJSONValue(loValue)
	if !hiOK || !loOK {
		return nil, false
	}
	combined := new(big.Int).Lsh(hi, 64)
	combined.Add(combined, lo)
	return combined, true
}

func parseNonNegativeBigInt(value string) (*big.Int, bool) {
	normalized := strings.TrimSpace(strings.ReplaceAll(value, ",", ""))
	if normalized == "" {
		return nil, false
	}
	parsed, ok := new(big.Int).SetString(normalized, 10)
	if !ok || parsed.Sign() < 0 {
		return nil, false
	}
	return parsed, true
}

func normalizedJSONKey(key string) string {
	normalized := strings.ToLower(strings.TrimSpace(key))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, " ", "_")
	return normalized
}

func latestSelfTest(log selfTestLog) (selfTestEntry, bool) {
	if len(log.SelfTests) > 0 {
		return log.SelfTests[0], true
	}
	if len(log.Standard.Table) > 0 {
		return log.Standard.Table[0], true
	}
	return selfTestEntry{}, false
}

func selfTestPassed(status string) (bool, bool) {
	normalized := strings.ToLower(strings.ReplaceAll(status, "_", " "))
	if strings.Contains(normalized, "without error") || strings.Contains(normalized, "passed") {
		return true, true
	}
	for _, token := range []string{"failed", "failure", "fatal", "read error", "read failure", "electrical error", "electrical failure", "servo error", "servo failure"} {
		if strings.Contains(normalized, token) {
			return false, true
		}
	}
	return false, false
}

func deviceNames(name, fallback string) (string, string) {
	if name == "" {
		name = fallback
	}
	if name == "" {
		name = "/dev/disk0"
	}
	base := filepath.Base(name)
	if strings.HasPrefix(base, "r") && strings.HasPrefix(name, "/dev/r") {
		base = strings.TrimPrefix(base, "r")
	}
	if strings.HasPrefix(name, "/dev/") {
		return name, base
	}
	return "/dev/" + base, base
}

func fallbackString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func optionalString(value string, reason model.MissingReason) model.Optional[string] {
	if value == "" {
		return model.None[string](reason)
	}
	return model.Some(value)
}

func optionalBool(value *bool, reason model.MissingReason) model.Optional[bool] {
	if value == nil {
		return model.None[bool](reason)
	}
	return model.Some(*value)
}

func optionalUint(value *uint64, reason model.MissingReason) model.Optional[uint64] {
	if value == nil {
		return model.None[uint64](reason)
	}
	return model.Some(*value)
}

func optionalInt(value *int64, reason model.MissingReason) model.Optional[int64] {
	if value == nil {
		return model.None[int64](reason)
	}
	return model.Some(*value)
}

func optionalTemperature(value *int64, reason model.MissingReason) model.Optional[int64] {
	if value == nil {
		return model.None[int64](reason)
	}
	if *value >= 273 {
		return model.Some(*value - 273)
	}
	return model.Some(*value)
}

func optionalCounter(value *decimal) model.Optional[model.BigCounter] {
	if value == nil || value.value == nil {
		return model.None[model.BigCounter](model.MissingUnavailable)
	}
	return model.Some(model.NewBigCounterString(value.value.String()))
}

func firstInt(values ...*int64) *int64 {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func firstDecimal(values ...*decimal) *decimal {
	for _, value := range values {
		if value != nil && value.value != nil {
			return value
		}
	}
	return nil
}

func optionalDataUnits(value *decimal) model.Optional[model.BigCounter] {
	if value == nil || value.value == nil {
		return model.None[model.BigCounter](model.MissingUnavailable)
	}
	units := new(big.Int).Set(value.value)
	units.Mul(units, big.NewInt(1000*512))
	return model.Some(model.NewBigCounterString(units.String()))
}

func optionalSensors(log nvmeLog) []model.Optional[int64] {
	values := log.TemperatureSensors
	if len(values) == 0 {
		values = log.TemperatureSensorsAlt
	}
	sensors := make([]model.Optional[int64], 0, len(values))
	for _, value := range values {
		sensors = append(sensors, model.Some(value))
	}
	return sensors
}

func maskSerial(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 4 {
		return strings.Repeat("*", len(value))
	}
	return "****" + value[len(value)-4:]
}
