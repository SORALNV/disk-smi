package native

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"reflect"
	"strings"
	"time"

	"disk-smi/internal/model"
	"howett.net/plist"
)

const defaultIORegTimeout = 10 * time.Second

type IORegRunner struct {
	Path    string
	Timeout time.Duration
}

type StorageStatsResult struct {
	Result  CommandResult
	Devices []StorageStats
}

type ControllerInfoResult struct {
	Result  CommandResult
	Devices []ControllerInfo
}

type StorageStats struct {
	BSDName          string
	BytesRead        uint64
	BytesWritten     uint64
	OperationsRead   uint64
	OperationsWrite  uint64
	ErrorsRead       uint64
	ErrorsWrite      uint64
	TotalTimeReadNS  uint64
	TotalTimeWriteNS uint64
}

type ControllerInfo struct {
	BSDName      string
	Model        string
	Serial       string
	Firmware     string
	Transport    string
	Location     string
	NVMeRevision string
}

func (r IORegRunner) StorageStats(ctx context.Context) (StorageStatsResult, error) {
	result, err := r.run(ctx, []string{"-r", "-c", "IOBlockStorageDriver", "-l", "-a"})
	statsResult := StorageStatsResult{Result: result}
	if err != nil {
		return statsResult, err
	}
	stats, err := ParseStorageStats(result.Stdout)
	if err != nil {
		return statsResult, err
	}
	statsResult.Devices = stats
	return statsResult, nil
}

func (r IORegRunner) ControllerInfos(ctx context.Context) (ControllerInfoResult, error) {
	result, err := r.run(ctx, []string{"-r", "-c", "IONVMeController", "-l", "-a"})
	infoResult := ControllerInfoResult{Result: result}
	if err != nil {
		return infoResult, err
	}
	infos, err := ParseControllerInfos(result.Stdout)
	if err != nil {
		return infoResult, err
	}
	infoResult.Devices = infos
	return infoResult, nil
}

func (r IORegRunner) run(ctx context.Context, args []string) (CommandResult, error) {
	path := r.Path
	if path == "" {
		path = "/usr/sbin/ioreg"
	}
	timeout := r.Timeout
	if timeout == 0 {
		timeout = defaultIORegTimeout
	}

	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

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
	if result.TimedOut {
		return result, fmt.Errorf("ioreg timed out after %s", timeout)
	}
	if err != nil {
		return result, fmt.Errorf("ioreg failed: %w", err)
	}
	return result, nil
}

func ParseStorageStats(data []byte) ([]StorageStats, error) {
	var nodes []map[string]any
	if err := plist.NewDecoder(bytes.NewReader(data)).Decode(&nodes); err != nil {
		return nil, fmt.Errorf("parse ioreg plist: %w", err)
	}
	stats := make([]StorageStats, 0, len(nodes))
	for _, node := range nodes {
		statMap := stringMap(node["Statistics"])
		if len(statMap) == 0 {
			continue
		}
		bsdName := wholeDiskName(node)
		if bsdName == "" {
			continue
		}
		stats = append(stats, StorageStats{
			BSDName:          bsdName,
			BytesRead:        uintStat(statMap, "Bytes (Read)"),
			BytesWritten:     uintStat(statMap, "Bytes (Write)"),
			OperationsRead:   uintStat(statMap, "Operations (Read)"),
			OperationsWrite:  uintStat(statMap, "Operations (Write)"),
			ErrorsRead:       uintStat(statMap, "Errors (Read)"),
			ErrorsWrite:      uintStat(statMap, "Errors (Write)"),
			TotalTimeReadNS:  uintStat(statMap, "Total Time (Read)"),
			TotalTimeWriteNS: uintStat(statMap, "Total Time (Write)"),
		})
	}
	return stats, nil
}

func ParseControllerInfos(data []byte) ([]ControllerInfo, error) {
	var nodes []map[string]any
	if err := plist.NewDecoder(bytes.NewReader(data)).Decode(&nodes); err != nil {
		return nil, fmt.Errorf("parse ioreg plist: %w", err)
	}
	infos := make([]ControllerInfo, 0, len(nodes))
	for _, node := range nodes {
		info := ControllerInfo{
			BSDName:      wholeDiskName(node),
			Model:        stringProperty(node, "Model Number"),
			Serial:       strings.TrimSpace(stringProperty(node, "Serial Number")),
			Firmware:     stringProperty(node, "Firmware Revision"),
			Transport:    stringProperty(node, "Physical Interconnect"),
			Location:     locationString(stringProperty(node, "Physical Interconnect Location")),
			NVMeRevision: stringProperty(node, "NVMe Revision Supported"),
		}
		if info.BSDName == "" {
			info.BSDName = wholeDiskNameFromController(node)
		}
		if info.BSDName == "" || (info.Model == "" && info.Serial == "" && info.Firmware == "" && info.NVMeRevision == "") {
			continue
		}
		infos = append(infos, info)
	}
	return infos, nil
}

func MergeStorageStats(snapshot model.DriveSnapshot, stats []StorageStats) model.DriveSnapshot {
	stat, ok := matchStorageStats(snapshot, stats)
	if !ok {
		return snapshot
	}
	if shouldUseStorageBusyTime(snapshot.Metrics.ControllerBusyMinutes) {
		if minutes := storageBusyMinutes(stat); minutes > 0 {
			snapshot.Metrics.ControllerBusyMinutes = model.Some(model.NewBigCounterString(fmt.Sprintf("%d", minutes)))
		}
	}
	return snapshot
}

func shouldUseStorageBusyTime(value model.Optional[model.BigCounter]) bool {
	return !value.Valid || value.Value.CmpInt64(0) == 0
}

func storageBusyMinutes(stat StorageStats) uint64 {
	totalNS := stat.TotalTimeReadNS + stat.TotalTimeWriteNS
	if totalNS == 0 {
		return 0
	}
	return totalNS / uint64(time.Minute)
}

func MergeControllerInfo(snapshot model.DriveSnapshot, infos []ControllerInfo) model.DriveSnapshot {
	info, ok := matchControllerInfo(snapshot, infos)
	if !ok {
		return snapshot
	}
	if info.Model != "" && (snapshot.Device.Model == "" || strings.EqualFold(snapshot.Device.Model, "UNKNOWN")) {
		snapshot.Device.Model = info.Model
	}
	if info.Serial != "" && !snapshot.Device.SerialRaw.Valid {
		snapshot.Device.SerialRaw = model.Some(info.Serial)
		snapshot.Device.Serial = model.Some(maskSerial(info.Serial))
	}
	if info.Firmware != "" && !snapshot.Device.Firmware.Valid {
		snapshot.Device.Firmware = model.Some(info.Firmware)
	}
	if info.Transport != "" && (!snapshot.Device.Transport.Valid || genericTransport(snapshot.Device.Transport.Value)) {
		snapshot.Device.Transport = model.Some(info.Transport)
	}
	if info.Location != "" && !snapshot.Device.Location.Valid {
		snapshot.Device.Location = model.Some(info.Location)
	}
	if info.NVMeRevision != "" && (!snapshot.Device.NVMeVersion.Valid || strings.HasPrefix(snapshot.Device.NVMeVersion.Value, "<")) {
		snapshot.Device.NVMeVersion = model.Some(info.NVMeRevision)
	}
	return snapshot
}

func matchStorageStats(snapshot model.DriveSnapshot, stats []StorageStats) (StorageStats, bool) {
	for _, stat := range stats {
		if stat.BSDName == snapshot.Device.BSDName {
			return stat, true
		}
	}
	return StorageStats{}, false
}

func matchControllerInfo(snapshot model.DriveSnapshot, infos []ControllerInfo) (ControllerInfo, bool) {
	for _, info := range infos {
		if info.BSDName != "" && info.BSDName == snapshot.Device.BSDName {
			return info, true
		}
		if info.Serial != "" && snapshot.Device.SerialRaw.Valid && info.Serial == snapshot.Device.SerialRaw.Value {
			return info, true
		}
		if info.Model != "" && info.Model == snapshot.Device.Model {
			return info, true
		}
	}
	return ControllerInfo{}, false
}

func wholeDiskName(node map[string]any) string {
	if name, _ := node["BSD Name"].(string); isWholeDiskName(name) {
		return name
	}
	children := anySlice(node["IORegistryEntryChildren"])
	for _, rawChild := range children {
		child := stringMap(rawChild)
		if name := wholeDiskName(child); name != "" {
			return name
		}
	}
	return ""
}

func wholeDiskNameFromController(node map[string]any) string {
	children := anySlice(node["IORegistryEntryChildren"])
	for _, rawChild := range children {
		child := stringMap(rawChild)
		if name := wholeDiskName(child); name != "" {
			return name
		}
	}
	return ""
}

func isWholeDiskName(name string) bool {
	if !strings.HasPrefix(name, "disk") {
		return false
	}
	return !strings.Contains(name[len("disk"):], "s")
}

func uintStat(stats map[string]any, key string) uint64 {
	switch value := stats[key].(type) {
	case uint64:
		return value
	case uint:
		return uint64(value)
	case uint32:
		return uint64(value)
	case int:
		if value >= 0 {
			return uint64(value)
		}
	case int64:
		if value >= 0 {
			return uint64(value)
		}
	case int32:
		if value >= 0 {
			return uint64(value)
		}
	}
	return 0
}

func stringProperty(node map[string]any, key string) string {
	value, _ := node[key].(string)
	return strings.TrimSpace(value)
}

func locationString(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "internal":
		return "internal"
	case "external":
		return "external"
	default:
		return value
	}
}

func genericTransport(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	return normalized == "" || normalized == "nvme" || normalized == "pcie" || normalized == "pcie / nvme"
}

func anySlice(value any) []any {
	rv := reflect.ValueOf(value)
	if !rv.IsValid() || rv.Kind() != reflect.Slice {
		return nil
	}
	out := make([]any, 0, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		out = append(out, rv.Index(i).Interface())
	}
	return out
}

func stringMap(value any) map[string]any {
	rv := reflect.ValueOf(value)
	if !rv.IsValid() || rv.Kind() != reflect.Map || rv.Type().Key().Kind() != reflect.String {
		return nil
	}
	out := make(map[string]any, rv.Len())
	iter := rv.MapRange()
	for iter.Next() {
		out[iter.Key().String()] = iter.Value().Interface()
	}
	return out
}
