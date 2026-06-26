package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"disk-smi/internal/discovery"
	"disk-smi/internal/health"
	"disk-smi/internal/jsonout"
	"disk-smi/internal/model"
	"disk-smi/internal/native"
	"disk-smi/internal/render"
	"disk-smi/internal/smartctl"
)

func RunSynthetic(opts render.Options) (string, error) {
	return render.RenderDrive(SyntheticSnapshot(), opts)
}

func Run(inputPath, target string, opts render.Options) (string, error) {
	output, _, err := RunWithDiagnostics(inputPath, target, opts)
	return output, err
}

func RunWithDiagnostics(inputPath, target string, opts render.Options) (string, Diagnostics, error) {
	return RunWithOptionsAndDiagnostics(inputPath, target, opts, SnapshotOptions{Detail: true})
}

func RunWithOptionsAndDiagnostics(inputPath, target string, opts render.Options, options SnapshotOptions) (string, Diagnostics, error) {
	snapshots, diagnostics, err := SnapshotsWithOptionsAndDiagnostics(inputPath, target, options)
	if err != nil {
		return "", diagnostics, err
	}
	output, err := render.RenderDrives(snapshots, opts)
	return output, diagnostics, err
}

func RunJSON(inputPath, target string, locale render.Locale, pretty bool, showSerial bool) (string, error) {
	output, _, err := RunJSONWithDiagnostics(inputPath, target, locale, pretty, showSerial)
	return output, err
}

func RunJSONWithDiagnostics(inputPath, target string, locale render.Locale, pretty bool, showSerial bool) (string, Diagnostics, error) {
	return RunJSONWithOptionsAndDiagnostics(inputPath, target, locale, pretty, showSerial, SnapshotOptions{})
}

func RunJSONWithOptionsAndDiagnostics(inputPath, target string, locale render.Locale, pretty bool, showSerial bool, options SnapshotOptions) (string, Diagnostics, error) {
	snapshots, diagnostics, err := SnapshotsWithOptionsAndDiagnostics(inputPath, target, options)
	if err != nil {
		return "", diagnostics, err
	}
	output, err := jsonout.Render(snapshots, string(locale), time.Now(), jsonout.Options{Pretty: pretty, ShowSerial: showSerial})
	return output, diagnostics, err
}

func Snapshot(inputPath, target string) (model.DriveSnapshot, error) {
	snapshots, err := Snapshots(inputPath, target)
	if err != nil {
		return model.DriveSnapshot{}, err
	}
	if len(snapshots) == 0 {
		return model.DriveSnapshot{}, coded(ExitNoSSD, fmt.Errorf("no SSD found"))
	}
	return snapshots[0], nil
}

func Snapshots(inputPath, target string) ([]model.DriveSnapshot, error) {
	snapshots, _, err := SnapshotsWithDiagnostics(inputPath, target)
	return snapshots, err
}

func SnapshotsWithDiagnostics(inputPath, target string) ([]model.DriveSnapshot, Diagnostics, error) {
	return SnapshotsWithOptionsAndDiagnostics(inputPath, target, SnapshotOptions{})
}

type SnapshotOptions struct {
	Detail             bool
	SmartctlPath       string
	DiskutilPath       string
	SystemProfilerPath string
	IORegPath          string
	Backend            Backend
}

type Backend string

const (
	BackendAuto     Backend = "auto"
	BackendSmartctl Backend = "smartctl"
	BackendNative   Backend = "native"
)

func SnapshotsWithOptionsAndDiagnostics(inputPath, target string, options SnapshotOptions) ([]model.DriveSnapshot, Diagnostics, error) {
	if inputPath == "" && target == "" {
		return discoverSnapshotsWithOptionsAndDiagnostics(context.Background(), options)
	}

	var snapshot model.DriveSnapshot
	diagnostics := Diagnostics{}
	if inputPath != "" {
		data, err := os.ReadFile(inputPath)
		if err != nil {
			return nil, diagnostics, err
		}
		snapshot, err = smartctl.Parse(data, target)
		if err != nil {
			return nil, diagnostics, err
		}
	} else {
		if backend(options) == BackendNative {
			snapshot, nativeDiagnostics, err := nativeSnapshotForTarget(context.Background(), target, options)
			diagnostics.Commands = append(diagnostics.Commands, nativeDiagnostics.Commands...)
			if err != nil {
				return nil, diagnostics, err
			}
			snapshot.Assessment = health.Assess(snapshot.Metrics)
			return []model.DriveSnapshot{snapshot}, diagnostics, nil
		}
		result, err := smartctl.Runner{Path: options.SmartctlPath, Detail: options.Detail}.Snapshot(context.Background(), target)
		diagnostics.Commands = append(diagnostics.Commands, smartctlDiagnostic(result.Result))
		if err != nil {
			if backend(options) == BackendAuto && canFallbackToNative(err) {
				snapshot, nativeDiagnostics, nativeErr := nativeSnapshotForTarget(context.Background(), target, options)
				diagnostics.Commands = append(diagnostics.Commands, nativeDiagnostics.Commands...)
				if nativeErr == nil {
					snapshot.Assessment = health.Assess(snapshot.Metrics)
					return []model.DriveSnapshot{snapshot}, diagnostics, nil
				}
			}
			return nil, diagnostics, err
		}
		snapshot = result.Snapshot
		if nativeAugmentSmartctlEnabled(options) {
			var nativeDiagnostics Diagnostics
			snapshot, nativeDiagnostics = augmentSmartctlSnapshotWithNative(context.Background(), snapshot, options)
			diagnostics.Commands = append(diagnostics.Commands, nativeDiagnostics.Commands...)
		}
	}
	snapshot.Assessment = health.Assess(snapshot.Metrics)
	return []model.DriveSnapshot{snapshot}, diagnostics, nil
}

func SyntheticSnapshot() model.DriveSnapshot {
	snapshot := model.SyntheticSnapshot()
	snapshot.Assessment = health.Assess(snapshot.Metrics)
	return snapshot
}

func discoverSnapshots(ctx context.Context) ([]model.DriveSnapshot, error) {
	snapshots, _, err := discoverSnapshotsWithDiagnostics(ctx)
	return snapshots, err
}

func discoverSnapshotsWithDiagnostics(ctx context.Context) ([]model.DriveSnapshot, Diagnostics, error) {
	return discoverSnapshotsWithOptionsAndDiagnostics(ctx, SnapshotOptions{})
}

func discoverSnapshotsWithOptionsAndDiagnostics(ctx context.Context, options SnapshotOptions) ([]model.DriveSnapshot, Diagnostics, error) {
	result, err := discovery.Runner{Path: options.DiskutilPath}.Discover(ctx)
	diagnostics := discoveryDiagnostics(result)
	if err != nil {
		return nil, diagnostics, err
	}
	if len(result.Candidates) == 0 {
		return nil, diagnostics, coded(ExitNoSSD, fmt.Errorf("no SSD found"))
	}

	snapshots := make([]model.DriveSnapshot, 0, len(result.Candidates))
	var failures []string
	var profileDevices []native.ProfileDevice
	var controllerInfos []native.ControllerInfo
	var storageStats []native.StorageStats
	profileLoaded := false
	controllerInfoLoaded := false
	storageStatsLoaded := false
	loadProfiles := func() []native.ProfileDevice {
		if profileLoaded {
			return profileDevices
		}
		profileLoaded = true
		profile, _ := native.Runner{Path: options.SystemProfilerPath}.ProfileNVMe(ctx)
		if len(profile.Result.Command) > 0 {
			diagnostics.Commands = append(diagnostics.Commands, nativeDiagnostic(profile.Result))
		}
		profileDevices = profile.Devices
		return profileDevices
	}
	loadControllerInfos := func() []native.ControllerInfo {
		if controllerInfoLoaded {
			return controllerInfos
		}
		controllerInfoLoaded = true
		info, _ := native.IORegRunner{Path: options.IORegPath}.ControllerInfos(ctx)
		if len(info.Result.Command) > 0 {
			diagnostics.Commands = append(diagnostics.Commands, nativeDiagnostic(info.Result))
		}
		controllerInfos = info.Devices
		return controllerInfos
	}
	loadStorageStats := func() []native.StorageStats {
		if storageStatsLoaded {
			return storageStats
		}
		storageStatsLoaded = true
		stats, _ := native.IORegRunner{Path: options.IORegPath}.StorageStats(ctx)
		if len(stats.Result.Command) > 0 {
			diagnostics.Commands = append(diagnostics.Commands, nativeDiagnostic(stats.Result))
		}
		storageStats = stats.Devices
		return storageStats
	}
	for _, candidate := range result.Candidates {
		if backend(options) == BackendNative {
			snapshot := native.Snapshot(candidate)
			snapshot = native.MergeProfile(snapshot, loadProfiles())
			snapshot = native.MergeControllerInfo(snapshot, loadControllerInfos())
			if rawNativeNVMeEnabled(options) {
				var rawDiagnostics []CommandDiagnostic
				snapshot, rawDiagnostics = mergeNativeNVMeRaw(ctx, snapshot)
				diagnostics.Commands = append(diagnostics.Commands, rawDiagnostics...)
			}
			snapshot = native.MergeStorageStats(snapshot, loadStorageStats())
			snapshot.Assessment = health.Assess(snapshot.Metrics)
			snapshots = append(snapshots, snapshot)
			continue
		}
		run, err := smartctl.Runner{Path: options.SmartctlPath, Detail: options.Detail}.Snapshot(ctx, candidate.DevicePath)
		diagnostics.Commands = append(diagnostics.Commands, smartctlDiagnostic(run.Result))
		if err != nil {
			if backend(options) == BackendAuto && canFallbackToNative(err) {
				snapshot := native.Snapshot(candidate)
				snapshot = native.MergeProfile(snapshot, loadProfiles())
				snapshot = native.MergeControllerInfo(snapshot, loadControllerInfos())
				if rawNativeNVMeEnabled(options) {
					var rawDiagnostics []CommandDiagnostic
					snapshot, rawDiagnostics = mergeNativeNVMeRaw(ctx, snapshot)
					diagnostics.Commands = append(diagnostics.Commands, rawDiagnostics...)
				}
				snapshot = native.MergeStorageStats(snapshot, loadStorageStats())
				snapshot.Assessment = health.Assess(snapshot.Metrics)
				snapshots = append(snapshots, snapshot)
				continue
			}
			failures = append(failures, fmt.Sprintf("%s: %v", candidate.BSDName, err))
			continue
		}
		snapshot := mergeDiscovery(run.Snapshot, candidate)
		if nativeAugmentSmartctlEnabled(options) {
			var nativeDiagnostics Diagnostics
			snapshot, nativeDiagnostics = augmentSmartctlSnapshotWithNative(ctx, snapshot, options)
			diagnostics.Commands = append(diagnostics.Commands, nativeDiagnostics.Commands...)
		}
		snapshot.Assessment = health.Assess(snapshot.Metrics)
		snapshots = append(snapshots, snapshot)
	}
	if len(snapshots) == 0 {
		if len(failures) > 0 {
			return nil, diagnostics, coded(ExitSMARTFailure, fmt.Errorf("SMART information unavailable: %s", strings.Join(failures, "; ")))
		}
		return nil, diagnostics, coded(ExitSMARTFailure, fmt.Errorf("SMART information unavailable"))
	}
	return snapshots, diagnostics, nil
}

func nativeSnapshotForTarget(ctx context.Context, target string, options SnapshotOptions) (model.DriveSnapshot, Diagnostics, error) {
	result, err := discovery.Runner{Path: options.DiskutilPath}.Inspect(ctx, target)
	diagnostics := Diagnostics{}
	if len(result.InfoResult.Command) > 0 {
		diagnostics.Commands = append(diagnostics.Commands, discoveryDiagnostic(result.InfoResult))
	}
	if err != nil {
		return model.DriveSnapshot{}, diagnostics, err
	}
	if !result.OK {
		return model.DriveSnapshot{}, diagnostics, coded(ExitNoSSD, fmt.Errorf("no SSD found for %s", target))
	}
	snapshot := native.Snapshot(result.Candidate)
	profile, _ := native.Runner{Path: options.SystemProfilerPath}.ProfileNVMe(ctx)
	if len(profile.Result.Command) > 0 {
		diagnostics.Commands = append(diagnostics.Commands, nativeDiagnostic(profile.Result))
	}
	snapshot = native.MergeProfile(snapshot, profile.Devices)
	info, _ := native.IORegRunner{Path: options.IORegPath}.ControllerInfos(ctx)
	if len(info.Result.Command) > 0 {
		diagnostics.Commands = append(diagnostics.Commands, nativeDiagnostic(info.Result))
	}
	snapshot = native.MergeControllerInfo(snapshot, info.Devices)
	if rawNativeNVMeEnabled(options) {
		var rawDiagnostics []CommandDiagnostic
		snapshot, rawDiagnostics = mergeNativeNVMeRaw(ctx, snapshot)
		diagnostics.Commands = append(diagnostics.Commands, rawDiagnostics...)
	}
	stats, _ := native.IORegRunner{Path: options.IORegPath}.StorageStats(ctx)
	if len(stats.Result.Command) > 0 {
		diagnostics.Commands = append(diagnostics.Commands, nativeDiagnostic(stats.Result))
	}
	snapshot = native.MergeStorageStats(snapshot, stats.Devices)
	return snapshot, diagnostics, nil
}

func augmentSmartctlSnapshotWithNative(ctx context.Context, snapshot model.DriveSnapshot, options SnapshotOptions) (model.DriveSnapshot, Diagnostics) {
	diagnostics := Diagnostics{}
	profile, _ := native.Runner{Path: options.SystemProfilerPath}.ProfileNVMe(ctx)
	if len(profile.Result.Command) > 0 {
		diagnostics.Commands = append(diagnostics.Commands, nativeDiagnostic(profile.Result))
	}
	snapshot = native.MergeProfile(snapshot, profile.Devices)
	info, _ := native.IORegRunner{Path: options.IORegPath}.ControllerInfos(ctx)
	if len(info.Result.Command) > 0 {
		diagnostics.Commands = append(diagnostics.Commands, nativeDiagnostic(info.Result))
	}
	snapshot = native.MergeControllerInfo(snapshot, info.Devices)
	if rawNativeNVMeEnabled(options) {
		var rawDiagnostics []CommandDiagnostic
		snapshot, rawDiagnostics = mergeNativeNVMeRaw(ctx, snapshot)
		diagnostics.Commands = append(diagnostics.Commands, rawDiagnostics...)
	}
	stats, _ := native.IORegRunner{Path: options.IORegPath}.StorageStats(ctx)
	if len(stats.Result.Command) > 0 {
		diagnostics.Commands = append(diagnostics.Commands, nativeDiagnostic(stats.Result))
	}
	snapshot = native.MergeStorageStats(snapshot, stats.Devices)
	return snapshot, diagnostics
}

func mergeNativeNVMeRaw(ctx context.Context, snapshot model.DriveSnapshot) (model.DriveSnapshot, []CommandDiagnostic) {
	if !strings.EqualFold(snapshot.Device.Protocol, "NVMe") {
		return snapshot, nil
	}
	var diagnostics []CommandDiagnostic
	var diagnostic CommandDiagnostic
	snapshot, diagnostic = mergeNativeNVMeIdentify(ctx, snapshot)
	diagnostics = append(diagnostics, diagnostic)
	snapshot, diagnostic = mergeNativeNVMeSMARTLog(ctx, snapshot)
	diagnostics = append(diagnostics, diagnostic)
	snapshot, diagnostic = mergeNativeNVMeOCPCloudSMARTLog(ctx, snapshot)
	diagnostics = append(diagnostics, diagnostic)
	snapshot, diagnostic = mergeNativeNVMeSelfTestLog(ctx, snapshot)
	diagnostics = append(diagnostics, diagnostic)
	snapshot, diagnostic = mergeNativeHIDTemperatureSensors(ctx, snapshot)
	diagnostics = append(diagnostics, diagnostic)
	return snapshot, diagnostics
}

func mergeNativeNVMeSMARTLog(ctx context.Context, snapshot model.DriveSnapshot) (model.DriveSnapshot, CommandDiagnostic) {
	diagnostic := nativeProbeDiagnostic("nvme-smart-log", snapshot.Device.DevicePath, nil)
	if !strings.EqualFold(snapshot.Device.Protocol, "NVMe") {
		return snapshot, diagnostic
	}
	log, err := native.ReadNVMeSMARTLog(ctx, snapshot.Device.DevicePath)
	if err != nil {
		return snapshot, nativeProbeDiagnostic("nvme-smart-log", snapshot.Device.DevicePath, err)
	}
	return native.MergeNVMeSMARTLog(snapshot, log), diagnostic
}

func mergeNativeNVMeOCPCloudSMARTLog(ctx context.Context, snapshot model.DriveSnapshot) (model.DriveSnapshot, CommandDiagnostic) {
	diagnostic := nativeProbeDiagnostic("nvme-ocp-cloud-smart-log", snapshot.Device.DevicePath, nil)
	if !strings.EqualFold(snapshot.Device.Protocol, "NVMe") {
		return snapshot, diagnostic
	}
	log, err := native.ReadNVMeOCPCloudSMARTLog(ctx, snapshot.Device.DevicePath)
	if err != nil {
		return snapshot, nativeProbeDiagnostic("nvme-ocp-cloud-smart-log", snapshot.Device.DevicePath, err)
	}
	return native.MergeNVMeOCPCloudSMARTLog(snapshot, log), diagnostic
}

func mergeNativeNVMeIdentify(ctx context.Context, snapshot model.DriveSnapshot) (model.DriveSnapshot, CommandDiagnostic) {
	diagnostic := nativeProbeDiagnostic("nvme-identify", snapshot.Device.DevicePath, nil)
	if !strings.EqualFold(snapshot.Device.Protocol, "NVMe") {
		return snapshot, diagnostic
	}
	identify, err := native.ReadNVMeIdentify(ctx, snapshot.Device.DevicePath)
	if err != nil {
		return snapshot, nativeProbeDiagnostic("nvme-identify", snapshot.Device.DevicePath, err)
	}
	return native.MergeNVMeIdentify(snapshot, identify), diagnostic
}

func mergeNativeNVMeSelfTestLog(ctx context.Context, snapshot model.DriveSnapshot) (model.DriveSnapshot, CommandDiagnostic) {
	diagnostic := nativeProbeDiagnostic("nvme-self-test-log", snapshot.Device.DevicePath, nil)
	if !strings.EqualFold(snapshot.Device.Protocol, "NVMe") {
		return snapshot, diagnostic
	}
	log, err := native.ReadNVMeSelfTestLog(ctx, snapshot.Device.DevicePath)
	if err != nil {
		return snapshot, nativeProbeDiagnostic("nvme-self-test-log", snapshot.Device.DevicePath, err)
	}
	return native.MergeNVMeSelfTestLog(snapshot, log), diagnostic
}

func mergeNativeHIDTemperatureSensors(ctx context.Context, snapshot model.DriveSnapshot) (model.DriveSnapshot, CommandDiagnostic) {
	diagnostic := nativeProbeDiagnostic("hid-temperature-sensors", snapshot.Device.DevicePath, nil)
	if !strings.EqualFold(snapshot.Device.Protocol, "NVMe") {
		return snapshot, diagnostic
	}
	sensors, err := native.ReadHIDTemperatureSensors(ctx)
	if err != nil {
		return snapshot, nativeProbeDiagnostic("hid-temperature-sensors", snapshot.Device.DevicePath, err)
	}
	return native.MergeHIDTemperatureSensors(snapshot, sensors), diagnostic
}

func rawNativeNVMeEnabled(options SnapshotOptions) bool {
	return options.DiskutilPath == "" && options.SystemProfilerPath == "" && options.IORegPath == ""
}

func nativeAugmentSmartctlEnabled(options SnapshotOptions) bool {
	return backend(options) == BackendAuto && options.SmartctlPath == "" && rawNativeNVMeEnabled(options)
}

func nativeProbeDiagnostic(operation string, target string, err error) CommandDiagnostic {
	diagnostic := CommandDiagnostic{
		Command: []string{"native", operation, target},
	}
	if err != nil {
		diagnostic.ExitCode = 1
		diagnostic.Stderr = err.Error()
	}
	return diagnostic
}

func backend(options SnapshotOptions) Backend {
	if options.Backend == "" {
		return BackendAuto
	}
	return options.Backend
}

func canFallbackToNative(err error) bool {
	code := ExitCode(err)
	if code == ExitMissingDependency || code == ExitPermission || errors.Is(err, os.ErrNotExist) {
		return true
	}
	var pathErr *os.PathError
	return errors.As(err, &pathErr) && os.IsNotExist(pathErr.Err)
}

func mergeDiscovery(snapshot model.DriveSnapshot, candidate discovery.Candidate) model.DriveSnapshot {
	if snapshot.Device.BSDName == "" {
		snapshot.Device.BSDName = candidate.BSDName
	}
	if snapshot.Device.DevicePath == "" {
		snapshot.Device.DevicePath = candidate.DevicePath
	}
	if snapshot.Device.Model == "" || snapshot.Device.Model == "UNKNOWN" {
		snapshot.Device.Model = candidate.Model
	}
	if snapshot.Device.CapacityByte.String() == "0" {
		snapshot.Device.CapacityByte = candidate.CapacityByte
	}
	if snapshot.Device.Protocol == "" && candidate.Protocol.Valid {
		snapshot.Device.Protocol = candidate.Protocol.Value
	}
	if !snapshot.Device.Transport.Valid {
		snapshot.Device.Transport = candidate.Transport
	}
	if !snapshot.Device.Location.Valid {
		snapshot.Device.Location = candidate.Location
	}
	return snapshot
}

func SyntheticSnapshots() []model.DriveSnapshot {
	return []model.DriveSnapshot{SyntheticSnapshot()}
}

type Diagnostics struct {
	Commands []CommandDiagnostic
}

type CommandDiagnostic struct {
	Command  []string
	Stderr   string
	ExitCode int
	TimedOut bool
}

func FormatDiagnostics(diagnostics Diagnostics) string {
	if len(diagnostics.Commands) == 0 {
		return ""
	}
	var lines []string
	for _, command := range diagnostics.Commands {
		if len(command.Command) == 0 {
			continue
		}
		lines = append(lines, "debug: command: "+shellQuote(command.Command))
		lines = append(lines, "debug: exit_code: "+strconv.Itoa(command.ExitCode))
		if command.TimedOut {
			lines = append(lines, "debug: timed_out: true")
		}
		if strings.TrimSpace(command.Stderr) != "" {
			lines = append(lines, "debug: stderr: "+strings.TrimSpace(command.Stderr))
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

func discoveryDiagnostics(result discovery.DiscoverResult) Diagnostics {
	diagnostics := Diagnostics{}
	if len(result.ListResult.Command) > 0 {
		diagnostics.Commands = append(diagnostics.Commands, discoveryDiagnostic(result.ListResult))
	}
	for _, item := range result.InfoResults {
		diagnostics.Commands = append(diagnostics.Commands, discoveryDiagnostic(item))
	}
	return diagnostics
}

func discoveryDiagnostic(result discovery.CommandResult) CommandDiagnostic {
	return CommandDiagnostic{
		Command:  result.Command,
		Stderr:   result.Stderr,
		ExitCode: result.ExitCode,
		TimedOut: result.TimedOut,
	}
}

func smartctlDiagnostic(result smartctl.CommandResult) CommandDiagnostic {
	return CommandDiagnostic{
		Command:  result.Command,
		Stderr:   result.Stderr,
		ExitCode: result.ExitCode,
		TimedOut: result.TimedOut,
	}
}

func nativeDiagnostic(result native.CommandResult) CommandDiagnostic {
	return CommandDiagnostic{
		Command:  result.Command,
		Stderr:   result.Stderr,
		ExitCode: result.ExitCode,
		TimedOut: result.TimedOut,
	}
}

func shellQuote(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "" {
			quoted = append(quoted, "''")
			continue
		}
		if strings.IndexFunc(arg, func(r rune) bool {
			return !(r == '/' || r == '.' || r == '-' || r == '_' || r == ':' || r == '=' || r == '+' || r == ',' || r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z')
		}) == -1 {
			quoted = append(quoted, arg)
			continue
		}
		quoted = append(quoted, "'"+strings.ReplaceAll(arg, "'", "'\\''")+"'")
	}
	return strings.Join(quoted, " ")
}
