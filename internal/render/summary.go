package render

import (
	"fmt"
	"strings"

	"disk-smi/internal/model"
)

func RenderSummary(snapshots []model.DriveSnapshot, opts Options) (string, error) {
	if opts.Width == 0 {
		opts.Width = 100
	}
	if opts.Width < 60 {
		opts.Width = 60
	}
	if opts.Locale == "" {
		opts.Locale = LocaleEnglish
	}
	text := labelsFor(opts.Locale)
	columns := summaryColumns(opts.Width)
	lines := []string{
		summaryRow([]string{"DEVICE", "MODEL", "STATUS", "ENDURANCE", "TEMP", "HOST WRITES"}, columns, true),
		summaryRule(opts.Width),
	}
	for _, snapshot := range snapshots {
		lines = append(lines, summaryRow([]string{
			snapshot.Device.BSDName,
			snapshot.Device.Model,
			status(snapshot.Assessment.OverallStatus, opts, text),
			enduranceRemaining(snapshot.Metrics.EnduranceUsedPercent, text),
			temperature(snapshot.Metrics.TemperatureCelsius, text),
			bytes(snapshot.Metrics.HostWritesBytes, opts, text),
		}, columns, false))
	}
	for index, line := range lines {
		if got := DisplayWidth(line); got != opts.Width {
			return "", fmt.Errorf("summary line %d has width %d, want %d: %q", index+1, got, opts.Width, line)
		}
	}
	return strings.Join(lines, "\n") + "\n", nil
}

func summaryColumns(width int) []int {
	if width < 60 {
		width = 60
	}
	available := width - 5
	device := 10
	status := 10
	endurance := 10
	temp := 8
	writes := 14
	modelWidth := available - device - status - endurance - temp - writes
	if modelWidth < 8 {
		modelWidth = 8
		writes = available - device - status - endurance - temp - modelWidth
	}
	return []int{device, modelWidth, status, endurance, temp, writes}
}

func summaryRow(values []string, widths []int, header bool) string {
	parts := make([]string, len(values))
	for i, value := range values {
		align := AlignLeft
		if i >= 2 {
			align = AlignRight
		}
		if header {
			align = AlignLeft
		}
		parts[i] = RenderCell(value, widths[i], align)
	}
	return strings.Join(parts, " ")
}

func summaryRule(width int) string {
	return strings.Repeat("-", width)
}
