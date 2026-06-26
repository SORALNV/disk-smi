package native

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"disk-smi/internal/model"
)

const defaultProfilerTimeout = 20 * time.Second

type Runner struct {
	Path    string
	Timeout time.Duration
}

type CommandResult struct {
	Command  []string
	Stdout   []byte
	Stderr   string
	ExitCode int
	TimedOut bool
}

type ProfileResult struct {
	Result  CommandResult
	Devices []ProfileDevice
}

type ProfileDevice struct {
	BSDName      string
	Model        string
	Firmware     string
	Serial       string
	CapacityByte uint64
	SMARTStatus  string
}

type profilerDocument struct {
	NVMe []profilerController `json:"SPNVMeDataType"`
}

type profilerController struct {
	Items []profilerDevice `json:"_items"`
}

type profilerDevice struct {
	Name         string `json:"_name"`
	BSDName      string `json:"bsd_name"`
	Model        string `json:"device_model"`
	Firmware     string `json:"device_revision"`
	Serial       string `json:"device_serial"`
	CapacityByte uint64 `json:"size_in_bytes"`
	SMARTStatus  string `json:"smart_status"`
}

func (r Runner) ProfileNVMe(ctx context.Context) (ProfileResult, error) {
	path := r.Path
	if path == "" {
		path = "/usr/sbin/system_profiler"
	}
	timeout := r.Timeout
	if timeout == 0 {
		timeout = defaultProfilerTimeout
	}

	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	args := []string{"SPNVMeDataType", "-json"}
	cmd := exec.CommandContext(runCtx, path, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	result := CommandResult{
		Command:  append([]string{path}, args...),
		Stdout:   stdout.Bytes(),
		Stderr:   stderr.String(),
		ExitCode: exitCode(err),
		TimedOut: errors.Is(runCtx.Err(), context.DeadlineExceeded),
	}
	profile := ProfileResult{Result: result}
	if result.TimedOut {
		return profile, fmt.Errorf("system_profiler timed out after %s", timeout)
	}
	if err != nil {
		return profile, fmt.Errorf("system_profiler failed: %w", err)
	}
	devices, err := ParseProfile(result.Stdout)
	if err != nil {
		return profile, err
	}
	profile.Devices = devices
	return profile, nil
}

func ParseProfile(data []byte) ([]ProfileDevice, error) {
	var doc profilerDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse system_profiler JSON: %w", err)
	}
	var devices []ProfileDevice
	for _, controller := range doc.NVMe {
		for _, item := range controller.Items {
			modelName := item.Model
			if modelName == "" {
				modelName = item.Name
			}
			devices = append(devices, ProfileDevice{
				BSDName:      filepath.Base(item.BSDName),
				Model:        modelName,
				Firmware:     item.Firmware,
				Serial:       item.Serial,
				CapacityByte: item.CapacityByte,
				SMARTStatus:  item.SMARTStatus,
			})
		}
	}
	return devices, nil
}

func MergeProfile(snapshot model.DriveSnapshot, devices []ProfileDevice) model.DriveSnapshot {
	device, ok := matchProfile(snapshot, devices)
	if !ok {
		return snapshot
	}
	if device.Model != "" {
		snapshot.Device.Model = device.Model
	}
	if device.Firmware != "" {
		snapshot.Device.Firmware = model.Some(device.Firmware)
	}
	if device.Serial != "" {
		snapshot.Device.SerialRaw = model.Some(device.Serial)
		snapshot.Device.Serial = model.Some(maskSerial(device.Serial))
	}
	if device.CapacityByte > 0 {
		snapshot.Device.CapacityByte = model.NewBigCounterString(new(big.Int).SetUint64(device.CapacityByte).String())
	}
	snapshot.Device.Protocol = "NVMe"
	if !snapshot.Device.Transport.Valid {
		snapshot.Device.Transport = model.Some("NVMe")
	}
	if device.SMARTStatus != "" {
		snapshot.Metrics.SMARTPassed = model.Some(smartStatusPassed(device.SMARTStatus))
	}
	return snapshot
}

func matchProfile(snapshot model.DriveSnapshot, devices []ProfileDevice) (ProfileDevice, bool) {
	for _, device := range devices {
		if device.BSDName != "" && device.BSDName == snapshot.Device.BSDName {
			return device, true
		}
	}
	for _, device := range devices {
		if device.Model != "" && strings.EqualFold(device.Model, snapshot.Device.Model) {
			return device, true
		}
	}
	return ProfileDevice{}, false
}

func smartStatusPassed(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	return normalized == "verified" || normalized == "passed" || normalized == "ok"
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

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}
