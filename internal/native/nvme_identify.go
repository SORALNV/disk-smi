package native

import (
	"encoding/binary"
	"fmt"
	"strings"

	"disk-smi/internal/model"
)

const nvmeIdentifyLength = 4096

type NVMeIdentify struct {
	Model                      string
	Serial                     string
	Firmware                   string
	Version                    string
	WarningTemperatureCelsius  model.Optional[int64]
	CriticalTemperatureCelsius model.Optional[int64]
	SelfTestSupported          bool
}

func ParseNVMeIdentify(data []byte) (NVMeIdentify, error) {
	if len(data) < nvmeIdentifyLength {
		return NVMeIdentify{}, fmt.Errorf("NVMe identify data too short: %d bytes", len(data))
	}
	identify := NVMeIdentify{
		Model:             trimIdentifyString(data[24:64]),
		Serial:            trimIdentifyString(data[4:24]),
		Firmware:          trimIdentifyString(data[64:72]),
		Version:           nvmeVersionString(binary.LittleEndian.Uint32(data[80:84])),
		SelfTestSupported: binary.LittleEndian.Uint16(data[256:258])&(1<<4) != 0,
	}
	if value, ok := kelvinToCelsius(binary.LittleEndian.Uint16(data[266:268])); ok {
		identify.WarningTemperatureCelsius = model.Some(value)
	} else {
		identify.WarningTemperatureCelsius = model.None[int64](model.MissingUnsupported)
	}
	if value, ok := kelvinToCelsius(binary.LittleEndian.Uint16(data[268:270])); ok {
		identify.CriticalTemperatureCelsius = model.Some(value)
	} else {
		identify.CriticalTemperatureCelsius = model.None[int64](model.MissingUnsupported)
	}
	return identify, nil
}

func MergeNVMeIdentify(snapshot model.DriveSnapshot, identify NVMeIdentify) model.DriveSnapshot {
	if identify.Model != "" && (snapshot.Device.Model == "" || strings.EqualFold(snapshot.Device.Model, "UNKNOWN")) {
		snapshot.Device.Model = identify.Model
	}
	if identify.Serial != "" && !snapshot.Device.SerialRaw.Valid {
		snapshot.Device.SerialRaw = model.Some(identify.Serial)
		snapshot.Device.Serial = model.Some(maskSerial(identify.Serial))
	}
	if identify.Firmware != "" {
		snapshot.Device.Firmware = model.Some(identify.Firmware)
	}
	if identify.Version != "" {
		snapshot.Device.NVMeVersion = model.Some(identify.Version)
	}
	if identify.WarningTemperatureCelsius.Valid {
		snapshot.Metrics.WarningTemperature = identify.WarningTemperatureCelsius
	} else if !snapshot.Metrics.WarningTemperature.Valid {
		snapshot.Metrics.WarningTemperature = identify.WarningTemperatureCelsius
	}
	if identify.CriticalTemperatureCelsius.Valid {
		snapshot.Metrics.CriticalTemperature = identify.CriticalTemperatureCelsius
	} else if !snapshot.Metrics.CriticalTemperature.Valid {
		snapshot.Metrics.CriticalTemperature = identify.CriticalTemperatureCelsius
	}
	if !identify.SelfTestSupported && !snapshot.Metrics.LastSelfTestStatus.Valid {
		snapshot.Metrics.LastSelfTestStatus = model.None[string](model.MissingUnsupported)
		snapshot.Metrics.LastSelfTestPassed = model.None[bool](model.MissingUnsupported)
	}
	return snapshot
}

func trimIdentifyString(data []byte) string {
	return strings.TrimRight(strings.TrimSpace(string(data)), "\x00")
}

func nvmeVersionString(raw uint32) string {
	if raw == 0 {
		return ""
	}
	major := raw >> 16
	minor := (raw >> 8) & 0xff
	tertiary := raw & 0xff
	if tertiary == 0 {
		return fmt.Sprintf("%d.%d", major, minor)
	}
	return fmt.Sprintf("%d.%d.%d", major, minor, tertiary)
}

func kelvinToCelsius(kelvin uint16) (int64, bool) {
	if kelvin < 273 {
		return 0, false
	}
	return int64(kelvin) - 273, true
}
