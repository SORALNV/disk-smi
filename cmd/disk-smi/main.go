package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"disk-smi/internal/app"
	"disk-smi/internal/device"
	"disk-smi/internal/loopstats"
	"disk-smi/internal/model"
	"disk-smi/internal/render"
	"disk-smi/internal/version"
)

func main() {
	var (
		japanese    bool
		ascii       bool
		showVersion bool
		jsonOutput  bool
		jsonPretty  bool
		summary     bool
		noColor     bool
		iec         bool
		showSerial  bool
		debug       bool
		widthArg    string
		langArg     string
		colorArg    string
		backendArg  string
		inputPath   string
		loopShort   int
		loopLong    int
	)

	flag.BoolVar(&japanese, "jp", false, "display in Japanese")
	flag.BoolVar(&ascii, "ascii", false, "use ASCII borders")
	flag.BoolVar(&showVersion, "version", false, "print version")
	flag.BoolVar(&jsonOutput, "json", false, "output JSON")
	flag.BoolVar(&jsonPretty, "json-pretty", false, "output pretty JSON")
	flag.BoolVar(&summary, "summary", false, "show compact summary")
	flag.BoolVar(&noColor, "no-color", false, "disable color")
	flag.BoolVar(&iec, "iec", false, "display bytes as IEC units")
	flag.BoolVar(&showSerial, "show-serial", false, "show full serial number")
	flag.BoolVar(&debug, "debug", false, "print command diagnostics to stderr")
	flag.StringVar(&widthArg, "width", "100", "panel width in display cells")
	flag.StringVar(&langArg, "lang", "en-US", "display language")
	flag.StringVar(&colorArg, "color", "auto", "color mode: auto, always, never")
	flag.StringVar(&backendArg, "backend", "auto", "data backend: auto, native, smartctl")
	flag.StringVar(&inputPath, "input", "", "read smartctl JSON fixture")
	flag.IntVar(&loopShort, "l", 0, "refresh every N seconds")
	flag.IntVar(&loopLong, "loop", 0, "refresh every N seconds")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: disk-smi [options] [disk]\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	explicit := explicitFlags(flag.CommandLine)
	target := ""
	if flag.NArg() > 0 {
		target = flag.Arg(0)
	}
	if flag.NArg() > 1 {
		fmt.Fprintln(os.Stderr, "only one disk argument is supported")
		os.Exit(2)
	}
	if target != "" {
		if _, err := device.Parse(target); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}

	if showVersion {
		fmt.Println(version.String())
		return
	}

	if err := validateColor(colorArg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if japanese {
		if !explicit["backend"] {
			backendArg = string(app.BackendNative)
		}
	}
	backend, err := parseBackend(backendArg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	loopInterval, err := parseLoopInterval(loopShort, loopLong)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	width := 100
	if !(widthArg == "auto" && loopInterval > 0 && !jsonOutput && !jsonPretty) {
		width, err = resolveWidth(widthArg)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}

	locale := render.LocaleEnglish
	if japanese || langArg == "ja-JP" || langArg == "ja" {
		locale = render.LocaleJapanese
	}

	opts := render.Options{
		Width:      width,
		Locale:     locale,
		ASCII:      ascii || shouldFallbackASCII(),
		ShowSerial: showSerial,
		IEC:        iec,
		Color:      colorEnabled(colorArg, noColor),
		Summary:    summary,
	}

	renderOnce := func() (string, error) {
		var (
			output      string
			diagnostics app.Diagnostics
			err         error
		)
		snapshotOptions := app.SnapshotOptions{Detail: true, Backend: backend}
		if jsonOutput || jsonPretty {
			output, diagnostics, err = app.RunJSONWithOptionsAndDiagnostics(inputPath, target, locale, jsonPretty, showSerial, snapshotOptions)
		} else {
			output, diagnostics, err = app.RunWithOptionsAndDiagnostics(inputPath, target, opts, snapshotOptions)
		}
		printDiagnostics(debug, diagnostics)
		return output, err
	}
	if loopInterval > 0 {
		if jsonOutput || jsonPretty {
			err = runLoop(loopInterval, renderOnce)
		} else {
			err = runSnapshotLoop(loopInterval, opts, widthArg, func() ([]model.DriveSnapshot, error) {
				snapshots, diagnostics, err := app.SnapshotsWithOptionsAndDiagnostics(inputPath, target, app.SnapshotOptions{Detail: true, Backend: backend})
				printDiagnostics(debug, diagnostics)
				return snapshots, err
			})
		}
	} else {
		var output string
		output, err = renderOnce()
		if err == nil {
			fmt.Print(output)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, app.UserMessage(err, locale))
		os.Exit(app.ExitCode(err))
	}
}

func explicitFlags(flags *flag.FlagSet) map[string]bool {
	values := map[string]bool{}
	flags.Visit(func(f *flag.Flag) {
		values[f.Name] = true
	})
	return values
}

func printDiagnostics(enabled bool, diagnostics app.Diagnostics) {
	if !enabled {
		return
	}
	fmt.Fprint(os.Stderr, app.FormatDiagnostics(diagnostics))
}

func colorEnabled(mode string, noColor bool) bool {
	if noColor {
		return false
	}
	switch mode {
	case "always":
		return true
	case "never":
		return false
	case "auto":
		return isTerminal(os.Stdout)
	default:
		return false
	}
}

func validateColor(mode string) error {
	switch mode {
	case "auto", "always", "never":
		return nil
	default:
		return fmt.Errorf("invalid --color value: %s", mode)
	}
}

func parseBackend(value string) (app.Backend, error) {
	switch app.Backend(value) {
	case app.BackendAuto, app.BackendNative, app.BackendSmartctl:
		return app.Backend(value), nil
	default:
		return "", fmt.Errorf("invalid --backend value: %s", value)
	}
}

func parseWidth(value string) (int, error) {
	width, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid --width value: %s", value)
	}
	if width < 20 {
		return 0, fmt.Errorf("--width must be at least 20")
	}
	return width, nil
}

func resolveWidth(value string) (int, error) {
	if value != "auto" {
		return parseWidth(value)
	}
	width := terminalWidth()
	if width == 0 {
		return 100, nil
	}
	if width < 20 {
		return 20, nil
	}
	return width, nil
}

func terminalWidth() int {
	if forcedSequence := os.Getenv("DISK_SMI_TEST_TERMINAL_WIDTH_SEQUENCE"); forcedSequence != "" {
		return nextForcedTerminalWidth(forcedSequence)
	}
	if forced := os.Getenv("DISK_SMI_TEST_TERMINAL_WIDTH"); forced != "" {
		width, _ := strconv.Atoi(forced)
		return width
	}
	if !isTerminal(os.Stdout) {
		return 0
	}
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return 0
	}
	fields := strings.Fields(stdout.String())
	if len(fields) != 2 {
		return 0
	}
	width, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0
	}
	return width
}

var terminalWidthSequenceIndex int

func nextForcedTerminalWidth(sequence string) int {
	parts := strings.Split(sequence, ",")
	if len(parts) == 0 {
		return 0
	}
	index := terminalWidthSequenceIndex
	if index >= len(parts) {
		index = len(parts) - 1
	}
	terminalWidthSequenceIndex++
	width, _ := strconv.Atoi(strings.TrimSpace(parts[index]))
	return width
}

func shouldFallbackASCII() bool {
	if os.Getenv("TERM") == "dumb" {
		return true
	}
	locale := os.Getenv("DISK_SMI_TEST_LANG")
	if locale == "" {
		locale = os.Getenv("LC_ALL")
	}
	if locale == "" {
		locale = os.Getenv("LC_CTYPE")
	}
	if locale == "" {
		locale = os.Getenv("LANG")
	}
	if locale == "" {
		return false
	}
	upper := strings.ToUpper(locale)
	return !strings.Contains(upper, "UTF-8") && !strings.Contains(upper, "UTF8")
}

func parseLoopInterval(shortValue int, longValue int) (time.Duration, error) {
	if shortValue > 0 && longValue > 0 && shortValue != longValue {
		return 0, fmt.Errorf("-l and --loop specify different intervals")
	}
	seconds := shortValue
	if seconds == 0 {
		seconds = longValue
	}
	if seconds == 0 {
		return 0, nil
	}
	if seconds < 2 {
		return 0, fmt.Errorf("loop interval must be at least 2 seconds")
	}
	return time.Duration(seconds) * time.Second, nil
}

func runLoop(interval time.Duration, renderOnce func() (string, error)) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	resize, stopResize := resizeSignalChannel()
	defer stopResize()
	count := 0
	for {
		count++
		output, err := renderOnce()
		if err != nil {
			return err
		}
		if isTerminal(os.Stdout) {
			fmt.Print("\x1b[H\x1b[2J")
		}
		fmt.Print(output)
		if shouldStopLoop(count) {
			return nil
		}
		waitForNextLoopEvent(ticker.C, resize)
	}
}

func runSnapshotLoop(interval time.Duration, opts render.Options, widthArg string, snapshotOnce func() ([]model.DriveSnapshot, error)) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	resize, stopResize := resizeSignalChannel()
	defer stopResize()
	var previous []model.DriveSnapshot
	count := 0
	for {
		count++
		current, err := snapshotOnce()
		if err != nil {
			return err
		}
		renderOpts := opts
		if widthArg == "auto" {
			width, err := resolveWidth(widthArg)
			if err != nil {
				return err
			}
			renderOpts.Width = width
		}
		if len(previous) > 0 {
			renderOpts.LoopStats = loopstats.Compute(previous, current, interval)
		}
		output, err := render.RenderDrives(current, renderOpts)
		if err != nil {
			return err
		}
		if isTerminal(os.Stdout) {
			fmt.Print("\x1b[H\x1b[2J")
		}
		fmt.Print(output)
		previous = current
		if shouldStopLoop(count) {
			return nil
		}
		waitForNextLoopEvent(ticker.C, resize)
	}
}

func resizeSignalChannel() (chan os.Signal, func()) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	return ch, func() {
		signal.Stop(ch)
		close(ch)
	}
}

func waitForNextLoopEvent(ticker <-chan time.Time, resize <-chan os.Signal) {
	select {
	case <-ticker:
	case <-resize:
	}
}

func shouldStopLoop(count int) bool {
	if os.Getenv("DISK_SMI_LOOP_ONCE") == "1" {
		return true
	}
	limitText := os.Getenv("DISK_SMI_LOOP_COUNT")
	if limitText == "" {
		return false
	}
	limit, err := strconv.Atoi(limitText)
	return err == nil && limit > 0 && count >= limit
}

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	return err == nil && (info.Mode()&os.ModeCharDevice) != 0
}
