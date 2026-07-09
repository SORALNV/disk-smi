package app

import (
	"fmt"
	"strings"

	"disk-smi/internal/model"
	"disk-smi/internal/render"
)

// Exit codes for --check monitoring mode.
//
// The general disk-smi exit-code table (see errors.go / docs/spec-v0.4.md
// section 17) already reserves 0, 2, 3, 4, 5, 6, and 7 for CLI errors,
// missing dependencies, permission problems, and similar runtime failures.
// A conventional Nagios-style 0=OK/1=WARNING/2=CRITICAL scheme would collide
// with exit code 2, which already means "CLI argument error" for every other
// disk-smi invocation (including --check itself, e.g. an invalid flag
// combination). Reusing 2 for "a drive is critical" would make exit code 2
// ambiguous for monitoring scripts, so --check keeps 0 and 1 (OK / WARNING,
// which is free) but moves CRITICAL to 8, the first exit code not already
// claimed by the table. UNKNOWN drives are folded into the WARNING bucket:
// the spec requires that an inability to read SMART data must never be
// treated as healthy, but disk-smi does not have hard evidence of a fault
// either.
const (
	CheckExitWarning  = 1
	CheckExitCritical = 8
)

// RunCheck evaluates every requested drive and renders one compact status
// line per drive for cron/launchd/Nagios-style monitoring. It returns the
// rendered text, the process exit code to use (see the CheckExit* constants
// above), collected diagnostics, and an error if the drives could not be
// read at all.
func RunCheck(inputPath, target string, locale render.Locale, options SnapshotOptions) (string, int, Diagnostics, error) {
	snapshots, diagnostics, err := SnapshotsWithOptionsAndDiagnostics(inputPath, target, options)
	if err != nil {
		return "", ExitCode(err), diagnostics, err
	}
	lines := make([]string, 0, len(snapshots))
	code := ExitOK
	for _, snapshot := range snapshots {
		line, driveCode := checkLine(snapshot, locale)
		lines = append(lines, line)
		if driveCode > code {
			code = driveCode
		}
	}
	return strings.Join(lines, "\n") + "\n", code, diagnostics, nil
}

func checkLine(snapshot model.DriveSnapshot, locale render.Locale) (string, int) {
	statusWord, code := checkStatus(snapshot.Assessment.OverallStatus, locale)
	device := snapshot.Device.BSDName
	if device == "" {
		device = snapshot.Device.DevicePath
	}
	fields := []string{
		statusWord,
		device,
		snapshot.Device.Model,
		"endurance=" + checkEndurancePercent(snapshot.Metrics.EnduranceUsedPercent),
		"temp=" + checkTemperature(snapshot.Metrics.TemperatureCelsius),
	}
	if reasons := snapshot.Assessment.ReasonCodes; len(reasons) > 0 {
		fields = append(fields, "reasons="+strings.Join(reasons, ","))
	}
	return strings.Join(fields, " "), code
}

func checkStatus(status model.OverallStatus, locale render.Locale) (string, int) {
	if locale == render.LocaleJapanese {
		switch status {
		case model.StatusGood:
			return "正常", ExitOK
		case model.StatusCaution:
			return "注意", CheckExitWarning
		case model.StatusCritical:
			return "危険", CheckExitCritical
		default:
			return "不明", CheckExitWarning
		}
	}
	switch status {
	case model.StatusGood:
		return "GOOD", ExitOK
	case model.StatusCaution:
		return "CAUTION", CheckExitWarning
	case model.StatusCritical:
		return "CRITICAL", CheckExitCritical
	default:
		return "UNKNOWN", CheckExitWarning
	}
}

// checkEndurancePercent renders the remaining endurance percentage, matching
// the clamp(100-used, 0, 100) rule used elsewhere in the codebase. Missing
// values render as "-" rather than a fake 0%.
func checkEndurancePercent(value model.Optional[uint64]) string {
	if !value.Valid {
		return "-"
	}
	remaining := uint64(0)
	if value.Value < 100 {
		remaining = 100 - value.Value
	}
	return fmt.Sprintf("%d%%", remaining)
}

func checkTemperature(value model.Optional[int64]) string {
	if !value.Valid {
		return "-"
	}
	return fmt.Sprintf("%dC", value.Value)
}
