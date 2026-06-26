package render

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"disk-smi/internal/model"
)

type Locale string

const (
	LocaleEnglish  Locale = "en-US"
	LocaleJapanese Locale = "ja-JP"
)

type Options struct {
	Width      int
	Locale     Locale
	ASCII      bool
	ShowSerial bool
	IEC        bool
	Color      bool
	Summary    bool
	LoopStats  map[string]model.LoopStats
}

type borderSet struct {
	topLeft, topRight, bottomLeft, bottomRight string
	horizontal, vertical, teeLeft, teeRight    string
	teeUp, teeDown                             string
	cross                                      string
}

var unicodeBorders = borderSet{
	topLeft: "┌", topRight: "┐", bottomLeft: "└", bottomRight: "┘",
	horizontal: "─", vertical: "│", teeLeft: "├", teeRight: "┤",
	teeUp: "┴", teeDown: "┬", cross: "┼",
}

var asciiBorders = borderSet{
	topLeft: "+", topRight: "+", bottomLeft: "+", bottomRight: "+",
	horizontal: "-", vertical: "|", teeLeft: "+", teeRight: "+",
	teeUp: "+", teeDown: "+", cross: "+",
}

type labels struct {
	overallStatus, temperature, powerOnTime, lifetimeIO string
	enduranceLine, hostWrites, hostReads                string
	healthEndurance, powerUsage                         string
	enduranceUsed, availableSpare, spareThreshold       string
	smartAssessment, criticalWarning                    string
	ssdPowerCycles, unsafeShutdowns, controllerBusyTime string
	readCommands, writeCommands                         string
	reliabilityThermals, deviceInformation              string
	mediaErrors, errorLogEntries                        string
	warningTempTime, criticalTempTime, tempSensors      string
	warningTemperature, criticalTemperature             string
	mediaWrites, healthReasons, lastSelfTest            string
	readRate, writeRate, readIOPS, writeIOPS            string
	temperatureChange, countersReset                    string
	model, firmware, nvmeVersion, transport, serial     string
	statusGood, smartPassed, none                       string
	internal                                            string
	missingUnsupported, missingPermission               string
	missingError, missingUnknown                        string
	note1, note2                                        string
}

func RenderDrive(snapshot model.DriveSnapshot, opts Options) (string, error) {
	out, err := RenderDrives([]model.DriveSnapshot{snapshot}, opts)
	if err != nil {
		return "", err
	}
	return out, nil
}

func RenderDrives(snapshots []model.DriveSnapshot, opts Options) (string, error) {
	if len(snapshots) == 0 {
		return "", fmt.Errorf("no drives to render")
	}
	if opts.Summary {
		return RenderSummary(snapshots, opts)
	}
	var rendered []string
	for _, snapshot := range snapshots {
		panel, err := renderOneDrive(snapshot, opts)
		if err != nil {
			return "", err
		}
		rendered = append(rendered, strings.TrimSuffix(panel, "\n"))
	}
	return strings.Join(rendered, "\n") + "\n", nil
}

func renderOneDrive(snapshot model.DriveSnapshot, opts Options) (string, error) {
	if opts.Width == 0 {
		opts.Width = 100
	}
	if opts.Width < 60 {
		return renderVertical(snapshot, opts)
	}
	if opts.Locale == "" {
		opts.Locale = LocaleEnglish
	}

	borders := unicodeBorders
	if opts.ASCII {
		borders = asciiBorders
	}
	labels := labelsFor(opts.Locale)
	lines := renderPanel(snapshot, opts, labels, borders)
	for index, line := range lines {
		if got := DisplayWidth(line); got != opts.Width {
			return "", fmt.Errorf("rendered line %d has width %d, want %d: %q", index+1, got, opts.Width, line)
		}
	}
	return strings.Join(lines, "\n") + "\n", nil
}

func renderPanel(snapshot model.DriveSnapshot, opts Options, text labels, borders borderSet) []string {
	width := opts.Width
	inner := width - 2
	topCols := topColumns(inner)
	left, right := bottomColumns(inner)
	lines := []string{
		borders.topLeft + strings.Repeat(borders.horizontal, inner) + borders.topRight,
		wrapSingle(snapshotHeader(snapshot, opts, text), inner, borders),
		separator(topCols, borders.teeDown, borders),
		topRow([]cell{
			{text.overallStatus, AlignCenter},
			{text.temperature, AlignCenter},
			{text.powerOnTime, AlignCenter},
			{text.lifetimeIO, AlignCenter},
		}, topCols, borders),
		topRow([]cell{{"", AlignCenter}, {"", AlignCenter}, {"", AlignCenter}, {"", AlignCenter}}, topCols, borders),
		topRow([]cell{
			{status(snapshot.Assessment.OverallStatus, opts, text), AlignCenter},
			{temperature(snapshot.Metrics.TemperatureCelsius, text), AlignCenter},
			{hours(snapshot.Metrics.PowerOnHours, text, false), AlignCenter},
			{text.hostWrites + "  " + bytes(snapshot.Metrics.HostWritesBytes, opts, text), AlignCenter},
		}, topCols, borders),
		topRow([]cell{
			{text.enduranceLine + " " + enduranceRemaining(snapshot.Metrics.EnduranceUsedPercent, text), AlignCenter},
			{"", AlignCenter},
			{hours(snapshot.Metrics.PowerOnHours, text, true), AlignCenter},
			{text.hostReads + "   " + bytes(snapshot.Metrics.HostReadsBytes, opts, text), AlignCenter},
		}, topCols, borders),
		topRow([]cell{{"", AlignCenter}, {"", AlignCenter}, {"", AlignCenter}, {"", AlignCenter}}, topCols, borders),
		separator(topCols, borders.teeUp, borders),
		titledSeparatorWithJunction(text.healthEndurance, text.powerUsage, left, right, borders.teeDown, borders),
		twoCol(text.enduranceUsed, percent(snapshot.Metrics.EnduranceUsedPercent, text), text.ssdPowerCycles, count(snapshot.Metrics.PowerCycles, optsSuffix(text, "回"), text), left, right, borders),
		twoCol(text.availableSpare, percent(snapshot.Metrics.AvailableSparePercent, text), text.unsafeShutdowns, count(snapshot.Metrics.UnsafeShutdowns, optsSuffix(text, "回"), text), left, right, borders),
		twoCol(text.spareThreshold, percent(snapshot.Metrics.SpareThresholdPercent, text), text.controllerBusyTime, controllerBusy(snapshot.Metrics.ControllerBusyMinutes, text), left, right, borders),
		twoCol(text.smartAssessment, smartPassed(snapshot.Metrics.SMARTPassed, text), text.readCommands, count(snapshot.Metrics.ReadCommands, "", text), left, right, borders),
		twoCol(text.criticalWarning, criticalWarning(snapshot.Metrics.CriticalWarning, text), text.writeCommands, count(snapshot.Metrics.WriteCommands, "", text), left, right, borders),
		titledSeparator(text.reliabilityThermals, text.deviceInformation, left, right, borders),
		twoCol(text.mediaErrors, count(snapshot.Metrics.MediaErrors, optsSuffix(text, "件"), text), text.model, snapshot.Device.Model, left, right, borders),
		twoCol(text.errorLogEntries, count(snapshot.Metrics.ErrorLogEntries, optsSuffix(text, "件"), text), text.firmware, optionalString(snapshot.Device.Firmware, text), left, right, borders),
		twoCol(text.warningTempTime, minutes(snapshot.Metrics.WarningTemperatureTime, text), text.nvmeVersion, optionalString(snapshot.Device.NVMeVersion, text), left, right, borders),
		twoCol(text.criticalTempTime, minutes(snapshot.Metrics.CriticalTemperatureTime, text), text.transport, optionalString(snapshot.Device.Transport, text), left, right, borders),
		twoCol(text.tempSensors, temperatureSensors(snapshot.Metrics.TemperatureSensors, text), text.serial, serial(snapshot, opts, text), left, right, borders),
	}
	lines = appendFullRow(lines, text.warningTemperature, temperature(snapshot.Metrics.WarningTemperature, text), optionalVisible(snapshot.Metrics.WarningTemperature), text.criticalTemperature, temperature(snapshot.Metrics.CriticalTemperature, text), optionalVisible(snapshot.Metrics.CriticalTemperature), left, right, borders)
	lines = appendFullRow(lines, text.mediaWrites, bytes(snapshot.Metrics.MediaWritesBytes, opts, text), optionalVisible(snapshot.Metrics.MediaWritesBytes), text.healthReasons, reasonCodes(snapshot.Assessment.ReasonCodes), len(snapshot.Assessment.ReasonCodes) > 0, left, right, borders)
	lines = appendFullRow(lines, text.lastSelfTest, selfTestStatus(snapshot.Metrics.LastSelfTestStatus, text), optionalVisible(snapshot.Metrics.LastSelfTestStatus), "", "", false, left, right, borders)
	if stats, ok := opts.LoopStats[snapshot.Device.BSDName]; ok {
		lines = append(lines, loopRows(stats, opts, text, left, right, borders)...)
	}
	lines = append(lines,
		endSeparator(left, right, borders),
		wrapSingle(text.note1, inner, borders),
		wrapSingle(text.note2, inner, borders),
		borders.bottomLeft+strings.Repeat(borders.horizontal, inner)+borders.bottomRight,
	)
	return lines
}

func appendFullRow(lines []string, leftLabel, leftValue string, leftVisible bool, rightLabel, rightValue string, rightVisible bool, leftWidth, rightWidth int, borders borderSet) []string {
	if !leftVisible && !rightVisible {
		return lines
	}
	if !leftVisible {
		leftLabel, leftValue = "", ""
	}
	if !rightVisible {
		rightLabel, rightValue = "", ""
	}
	return append(lines, twoCol(leftLabel, leftValue, rightLabel, rightValue, leftWidth, rightWidth, borders))
}

func optionalVisible[T any](value model.Optional[T]) bool {
	if value.Valid {
		return true
	}
	return value.Reason != model.MissingUnsupported && value.Reason != model.MissingNone
}

type cell struct {
	text      string
	alignment Alignment
}

func topColumns(inner int) []int {
	available := inner - 3
	c1 := available * 25 / 100
	c2 := available * 15 / 100
	c3 := available * 24 / 100
	c4 := available - c1 - c2 - c3
	return []int{c1, c2, c3, c4}
}

func bottomColumns(inner int) (int, int) {
	available := inner - 1
	left := available / 2
	return left, available - left
}

func separator(widths []int, junction string, borders borderSet) string {
	parts := make([]string, len(widths))
	for i, width := range widths {
		parts[i] = strings.Repeat(borders.horizontal, width)
	}
	return borders.teeLeft + strings.Join(parts, junction) + borders.teeRight
}

func titledSeparator(leftTitle, rightTitle string, leftWidth, rightWidth int, borders borderSet) string {
	return titledSeparatorWithJunction(leftTitle, rightTitle, leftWidth, rightWidth, borders.cross, borders)
}

func titledSeparatorWithJunction(leftTitle, rightTitle string, leftWidth, rightWidth int, junction string, borders borderSet) string {
	cells := horizontalCells(leftWidth+1+rightWidth, borders)
	setCell(cells, leftWidth, junction)
	placeTitle(cells, 0, leftWidth, leftTitle)
	placeTitle(cells, leftWidth+1, leftWidth+1+rightWidth, rightTitle)
	return borders.teeLeft + strings.Join(cells, "") + borders.teeRight
}

func endSeparator(leftWidth, rightWidth int, borders borderSet) string {
	cells := horizontalCells(leftWidth+1+rightWidth, borders)
	setCell(cells, leftWidth, borders.teeUp)
	return borders.teeLeft + strings.Join(cells, "") + borders.teeRight
}

func horizontalCells(width int, borders borderSet) []string {
	cells := make([]string, width)
	for i := range cells {
		cells[i] = borders.horizontal
	}
	return cells
}

func setCell(cells []string, position int, value string) {
	if position < 0 || position >= len(cells) {
		return
	}
	cells[position] = value
}

func placeTitle(cells []string, start, end int, title string) {
	title = " " + SanitizeTerminalText(title) + " "
	titleWidth := DisplayWidth(title)
	if titleWidth <= 0 || start >= end {
		return
	}
	if titleWidth > end-start {
		title = truncateDisplay(title, end-start)
		titleWidth = DisplayWidth(title)
	}
	position := start + 1
	if position+titleWidth > end {
		position = start
	}
	overlayDisplay(cells, position, title)
}

func overlayDisplay(cells []string, start int, text string) {
	position := start
	for _, r := range text {
		width := DisplayWidth(string(r))
		if width <= 0 {
			continue
		}
		if position >= len(cells) {
			return
		}
		cells[position] = string(r)
		for offset := 1; offset < width && position+offset < len(cells); offset++ {
			cells[position+offset] = ""
		}
		position += width
	}
}

func topRow(cells []cell, widths []int, borders borderSet) string {
	parts := make([]string, len(cells))
	for i, cell := range cells {
		parts[i] = RenderCell(cell.text, widths[i], cell.alignment)
	}
	return borders.vertical + strings.Join(parts, borders.vertical) + borders.vertical
}

func twoCol(leftLabel, leftValue, rightLabel, rightValue string, leftWidth, rightWidth int, borders borderSet) string {
	left := paddedKeyValue(leftLabel, leftValue, leftWidth)
	right := paddedKeyValue(rightLabel, rightValue, rightWidth)
	return borders.vertical + left + borders.vertical + right + borders.vertical
}

func loopRows(stats model.LoopStats, opts Options, text labels, leftWidth, rightWidth int, borders borderSet) []string {
	if stats.Reset {
		return []string{twoCol(text.countersReset, "yes", text.temperatureChange, temperatureDelta(stats.TemperatureChangeC, text), leftWidth, rightWidth, borders)}
	}
	if !stats.Valid {
		return nil
	}
	return []string{
		twoCol(text.readRate, rateBytes(stats.ReadRateBytesPerSec, opts), text.writeRate, rateBytes(stats.WriteRateBytesPerSec, opts), leftWidth, rightWidth, borders),
		twoCol(text.readIOPS, rateCount(stats.ReadIOPS, text), text.writeIOPS, rateCount(stats.WriteIOPS, text), leftWidth, rightWidth, borders),
		twoCol(text.temperatureChange, temperatureDelta(stats.TemperatureChangeC, text), "", "", leftWidth, rightWidth, borders),
	}
}

func paddedKeyValue(label, value string, width int) string {
	if width <= 2 {
		return keyValue(label, value, width)
	}
	return " " + keyValue(label, value, width-2) + " "
}

func keyValue(label, value string, width int) string {
	label = SanitizeTerminalText(label)
	value = SanitizeTerminalText(value)
	valueWidth := DisplayWidth(value)
	minGap := 1
	labelWidth := width - valueWidth - minGap
	if labelWidth < 1 {
		return RenderCell(label+" "+value, width, AlignLeft)
	}
	return RenderCell(label, labelWidth, AlignLeft) + strings.Repeat(" ", minGap) + RenderCell(value, valueWidth, AlignRight)
}

func wrapSingle(text string, width int, borders borderSet) string {
	return borders.vertical + " " + RenderCell(text, width-1, AlignLeft) + borders.vertical
}

func snapshotHeader(snapshot model.DriveSnapshot, opts Options, text labels) string {
	return fmt.Sprintf("%s / %s / %s / %s / %s",
		snapshot.Device.Model,
		formatCapacity(snapshot.Device.CapacityByte, opts),
		snapshot.Device.Protocol,
		location(snapshot.Device.Location, text),
		snapshot.Device.BSDName,
	)
}

func renderVertical(snapshot model.DriveSnapshot, opts Options) (string, error) {
	text := labelsFor(opts.Locale)
	lines := []string{
		"Model: " + snapshot.Device.Model,
		"Status: " + status(snapshot.Assessment.OverallStatus, opts, text),
		"Endurance: " + enduranceRemaining(snapshot.Metrics.EnduranceUsedPercent, text),
		"Temperature: " + temperature(snapshot.Metrics.TemperatureCelsius, text),
		"Host writes: " + bytes(snapshot.Metrics.HostWritesBytes, opts, text),
	}
	return strings.Join(lines, "\n") + "\n", nil
}

func labelsFor(locale Locale) labels {
	if locale == LocaleJapanese {
		return labels{
			overallStatus: "総合状態", temperature: "温度", powerOnTime: "通電時間", lifetimeIO: "累積ホストI/O",
			enduranceLine: "耐久残量", hostWrites: "総書き込み量", hostReads: "総読み込み量",
			healthEndurance: "健康・耐久", powerUsage: "電源・使用状況",
			enduranceUsed: "耐久使用率", availableSpare: "予備領域", spareThreshold: "予備領域閾値",
			smartAssessment: "SMART判定", criticalWarning: "重大警告",
			ssdPowerCycles: "SSD電源投入回数", unsafeShutdowns: "異常電源断回数", controllerBusyTime: "累積I/O処理時間",
			readCommands: "読み込みコマンド数", writeCommands: "書き込みコマンド数",
			reliabilityThermals: "信頼性・温度", deviceInformation: "デバイス情報",
			mediaErrors: "メディア・整合性エラー", errorLogEntries: "エラーログ件数",
			warningTempTime: "警告温度滞在時間", criticalTempTime: "危険温度滞在時間", tempSensors: "温度センサー",
			warningTemperature: "警告温度", criticalTemperature: "危険温度",
			mediaWrites: "NAND・媒体書き込み量", healthReasons: "判定理由", lastSelfTest: "最終自己診断",
			readRate: "読み込み速度", writeRate: "書き込み速度",
			readIOPS: "読み込みIOPS", writeIOPS: "書き込みIOPS",
			temperatureChange: "温度変化", countersReset: "カウンターリセット",
			model: "モデル", firmware: "ファームウェア", nvmeVersion: "NVMeバージョン", transport: "接続方式", serial: "シリアル",
			statusGood: "正常", smartPassed: "合格", none: "なし", internal: "内蔵",
			missingUnsupported: "非対応", missingPermission: "権限不足",
			missingError: "取得失敗", missingUnknown: "不明",
			note1: "注記: 耐久残量0%はSSDが報告する推定公称耐久を消費した目安です。",
			note2: "即時故障や使用不能を意味しません。",
		}
	}
	return labels{
		overallStatus: "OVERALL STATUS", temperature: "TEMPERATURE", powerOnTime: "POWER-ON TIME", lifetimeIO: "LIFETIME HOST I/O",
		enduranceLine: "Endurance", hostWrites: "Host writes", hostReads: "Host reads",
		healthEndurance: "HEALTH & ENDURANCE", powerUsage: "POWER & USAGE",
		enduranceUsed: "Endurance used", availableSpare: "Available spare", spareThreshold: "Spare threshold",
		smartAssessment: "SMART assessment", criticalWarning: "Critical warning",
		ssdPowerCycles: "SSD power cycles", unsafeShutdowns: "Unsafe shutdowns", controllerBusyTime: "Controller busy time",
		readCommands: "Read commands", writeCommands: "Write commands",
		reliabilityThermals: "RELIABILITY & THERMALS", deviceInformation: "DEVICE INFORMATION",
		mediaErrors: "Media/data errors", errorLogEntries: "Error log entries",
		warningTempTime: "Warning-temp time", criticalTempTime: "Critical-temp time", tempSensors: "Temperature sensors",
		warningTemperature: "Warning temperature", criticalTemperature: "Critical temperature",
		mediaWrites: "Media/NAND writes", healthReasons: "Reason codes", lastSelfTest: "Last self-test",
		readRate: "Read rate", writeRate: "Write rate",
		readIOPS: "Read IOPS", writeIOPS: "Write IOPS",
		temperatureChange: "Temperature change", countersReset: "Counter reset",
		model: "Model", firmware: "Firmware", nvmeVersion: "NVMe version", transport: "Transport", serial: "Serial",
		statusGood: "GOOD", smartPassed: "PASSED", none: "None", internal: "Internal",
		missingUnsupported: "UNSUPPORTED", missingPermission: "PERMISSION REQUIRED",
		missingError: "ERROR", missingUnknown: "UNKNOWN",
		note1: "NOTE: Endurance 0% means the device-reported rated endurance has been consumed.",
		note2: "It does not mean immediate SSD failure.",
	}
}

func status(value model.OverallStatus, opts Options, text labels) string {
	rendered := strings.ToUpper(string(value))
	if value == model.StatusGood {
		rendered = text.statusGood
	}
	if !opts.Color {
		return rendered
	}
	switch value {
	case model.StatusGood:
		return "\x1b[32m" + rendered + "\x1b[0m"
	case model.StatusCaution:
		return "\x1b[33m" + rendered + "\x1b[0m"
	case model.StatusCritical:
		return "\x1b[31m" + rendered + "\x1b[0m"
	default:
		return rendered
	}
}

func optionalString(value model.Optional[string], text labels) string {
	if !value.Valid {
		return missingReason(value.Reason, text)
	}
	return value.Value
}

func selfTestStatus(value model.Optional[string], text labels) string {
	return optionalString(value, text)
}

func location(value model.Optional[string], text labels) string {
	if !value.Valid {
		return missingReason(value.Reason, text)
	}
	if value.Value == "internal" {
		return text.internal
	}
	return value.Value
}

func percent(value model.Optional[uint64], text labels) string {
	if !value.Valid {
		return missingReason(value.Reason, text)
	}
	return fmt.Sprintf("%d%%", value.Value)
}

func enduranceRemaining(value model.Optional[uint64], text labels) string {
	if !value.Valid {
		return missingReason(value.Reason, text)
	}
	if value.Value >= 100 {
		return "0%"
	}
	return fmt.Sprintf("%d%%", 100-value.Value)
}

func temperature(value model.Optional[int64], text labels) string {
	if !value.Valid {
		return missingReason(value.Reason, text)
	}
	return fmt.Sprintf("%d°C", value.Value)
}

func temperatureSensors(values []model.Optional[int64], text labels) string {
	var parts []string
	var missing []string
	for _, value := range values {
		if value.Valid {
			parts = append(parts, strconv.FormatInt(value.Value, 10))
		} else if rendered := missingReason(value.Reason, text); rendered != "—" {
			missing = append(missing, rendered)
		}
	}
	switch {
	case len(parts) > 0:
		return strings.Join(parts, " / ") + "°C"
	case len(missing) > 0:
		return strings.Join(missing, " / ")
	default:
		return "—"
	}
}

func serial(snapshot model.DriveSnapshot, opts Options, text labels) string {
	if opts.ShowSerial && snapshot.Device.SerialRaw.Valid {
		return snapshot.Device.SerialRaw.Value
	}
	return optionalString(snapshot.Device.Serial, text)
}

func reasonCodes(values []string) string {
	if len(values) == 0 {
		return "—"
	}
	return strings.Join(values, ",")
}

func rateBytes(value model.Optional[model.BigCounter], opts Options) string {
	if !value.Valid {
		return missingReason(value.Reason, labelsFor(opts.Locale))
	}
	return formatBytes(value.Value.String(), opts) + "/s"
}

func rateCount(value model.Optional[model.BigCounter], text labels) string {
	if !value.Valid {
		return missingReason(value.Reason, text)
	}
	return comma(value.Value.String()) + "/s"
}

func temperatureDelta(value model.Optional[int64], text labels) string {
	if !value.Valid {
		return missingReason(value.Reason, text)
	}
	if value.Value > 0 {
		return fmt.Sprintf("+%d°C", value.Value)
	}
	return fmt.Sprintf("%d°C", value.Value)
}

func bytes(value model.Optional[model.BigCounter], opts Options, text labels) string {
	if !value.Valid {
		return missingReason(value.Reason, text)
	}
	return formatBytes(value.Value.String(), opts)
}

func formatCapacity(value model.BigCounter, opts Options) string {
	return formatBytes(value.String(), opts)
}

func formatBytes(raw string, opts Options) string {
	value, ok := new(big.Int).SetString(raw, 10)
	if !ok {
		return raw + " B"
	}
	if opts.IEC {
		return scaledBytes(value, []unitScale{
			{suffix: "TiB", scale: new(big.Int).Exp(big.NewInt(1024), big.NewInt(4), nil)},
			{suffix: "GiB", scale: new(big.Int).Exp(big.NewInt(1024), big.NewInt(3), nil)},
			{suffix: "MiB", scale: new(big.Int).Exp(big.NewInt(1024), big.NewInt(2), nil)},
			{suffix: "KiB", scale: big.NewInt(1024)},
		})
	}
	return scaledBytes(value, []unitScale{
		{suffix: "TB", scale: big.NewInt(1_000_000_000_000)},
		{suffix: "GB", scale: big.NewInt(1_000_000_000)},
		{suffix: "MB", scale: big.NewInt(1_000_000)},
		{suffix: "kB", scale: big.NewInt(1_000)},
	})
}

type unitScale struct {
	suffix string
	scale  *big.Int
}

func scaledBytes(value *big.Int, units []unitScale) string {
	for _, unit := range units {
		if value.Cmp(unit.scale) >= 0 {
			ratio := new(big.Rat).SetFrac(value, unit.scale)
			return ratio.FloatString(2) + " " + unit.suffix
		}
	}
	return value.String() + " B"
}

func count(value model.Optional[model.BigCounter], suffix string, text labels) string {
	if !value.Valid {
		return missingReason(value.Reason, text)
	}
	return comma(value.Value.String()) + suffix
}

func hours(value model.Optional[model.BigCounter], text labels, compact bool) string {
	if !value.Valid {
		return missingReason(value.Reason, text)
	}
	raw := value.Value.String()
	if raw != "1500" {
		return raw
	}
	if compact {
		if text.internal == "内蔵" {
			return "62日12時間"
		}
		return "62 d 12 h"
	}
	if text.internal == "内蔵" {
		return "1,500時間"
	}
	return "1,500 hours"
}

func controllerBusy(value model.Optional[model.BigCounter], text labels) string {
	if !value.Valid {
		return missingReason(value.Reason, text)
	}
	minutes, ok := new(big.Int).SetString(value.Value.String(), 10)
	if !ok || !minutes.IsInt64() {
		return value.Value.String() + " min"
	}
	total := minutes.Int64()
	if text.internal == "内蔵" {
		if total >= 60 {
			hours := total / 60
			remainder := total % 60
			if remainder == 0 {
				return fmt.Sprintf("%d時間", hours)
			}
			return fmt.Sprintf("%d時間%d分", hours, remainder)
		}
		return fmt.Sprintf("%d分", total)
	}
	if total >= 60 {
		hours := total / 60
		remainder := total % 60
		if remainder == 0 {
			return fmt.Sprintf("%d hours", hours)
		}
		return fmt.Sprintf("%d h %d min", hours, remainder)
	}
	return fmt.Sprintf("%d min", total)
}

func minutes(value model.Optional[model.BigCounter], text labels) string {
	if !value.Valid {
		return missingReason(value.Reason, text)
	}
	if text.internal == "内蔵" {
		return value.Value.String() + "分"
	}
	return value.Value.String() + " min"
}

func smartPassed(value model.Optional[bool], text labels) string {
	if !value.Valid {
		return missingReason(value.Reason, text)
	}
	if value.Value {
		return text.smartPassed
	}
	return "FAILED"
}

func criticalWarning(value model.Optional[uint64], text labels) string {
	if !value.Valid {
		return missingReason(value.Reason, text)
	}
	if value.Value == 0 {
		return text.none
	}
	return strconv.FormatUint(value.Value, 10)
}

func missingReason(reason model.MissingReason, text labels) string {
	switch reason {
	case model.MissingUnsupported:
		return text.missingUnsupported
	case model.MissingPermission:
		return text.missingPermission
	case model.MissingError:
		return text.missingError
	case model.MissingUnknown:
		return text.missingUnknown
	default:
		return "—"
	}
}

func optsSuffix(text labels, suffix string) string {
	if text.internal == "内蔵" {
		return suffix
	}
	return ""
}

func comma(value string) string {
	if len(value) <= 3 {
		return value
	}
	var out []byte
	prefix := len(value) % 3
	if prefix == 0 {
		prefix = 3
	}
	out = append(out, value[:prefix]...)
	for i := prefix; i < len(value); i += 3 {
		out = append(out, ',')
		out = append(out, value[i:i+3]...)
	}
	return string(out)
}
