# disk-smi implementation status

This document tracks implementation progress against `docs/spec-v0.4.md`.

## Implemented

- Go CLI entrypoint: `cmd/disk-smi`
- English and Japanese panel rendering
- Unicode and ASCII borders
- Display-cell width padding, truncation, ANSI stripping, and golden width tests
- Localized missing-value reason rendering for unsupported, permission-required, error, and unknown values
- `--width`, `--ascii`, `-jp`, `--lang`, `--json`, `--json-pretty`, `--input`
- `--summary`, `--iec`, `--show-serial`, `--color`, `--no-color`
- `--width auto` using terminal width when available
- ASCII fallback for `TERM=dumb` and non-UTF locale environments
- `-l` / `--loop` basic refresh mode
- Loop deltas for read/write rate, IOPS, and temperature change
- Loop mode recomputes `--width auto` on each sample
- Loop mode wakes and redraws on SIGWINCH terminal resize events
- JSON output with English keys and string counters
- JSON detailed SMART sections for endurance, thermals, reliability, I/O, power, and null-value missing reasons
- Serial masking by default
- NVMe fixture parsing for major SMART fields
- NVMe Identify Controller temperature threshold parsing from smartctl JSON when `wctemp`/`cctemp` are present
- NVMe optional admin command parsing from smartctl JSON to mark unsupported self-test explicitly
- OCP `physical_media_units_written` parsing from JSON input, including 128-bit `hi`/`lo` counter shapes
- SATA best-effort fixture parsing for selected ATA SMART attributes
- SMART self-test history parsing for common NVMe/ATA smartctl JSON shapes
- macOS `diskutil list/info -plist` discovery foundation
- APFS virtual container filtering to avoid duplicate physical SSD display
- Best-effort external SSD discovery when enclosure plist omits `SolidState`
- Built-in macOS metadata backend selectable with `--backend native`
- Native backend SMART metrics from `diskutil info -plist` `SMARTDeviceSpecificKeysMayVaryNotGuaranteed`
- Native backend direct NVMe SMART/Health log reading through macOS IOKit
- Native backend OCP Cloud SMART Log Page `0xC0` parsing for physical media writes when the controller exposes it, guarded by the OCP SMART Cloud Attributes log GUID
- Native backend NVMe Identify parsing for model, serial, firmware, version, and temperature thresholds when the controller reports them
- Native backend NVMe self-test log parsing when Log Page 06h is exposed by the controller
- Native backend HID temperature sensor reading for Apple `NAND CH0 temp`
- Native backend NVMe augmentation from `system_profiler SPNVMeDataType -json`
- Native backend NVMe revision fallback from IORegistry controller properties
- Native backend fallback cumulative I/O processing time from `ioreg` `IOBlockStorageDriver` total read/write time when NVMe SMART reports zero or does not expose the counter
- Automatic fallback to the built-in backend when `smartctl` is unavailable
- Automatic native augmentation of successful `smartctl` snapshots when the built-in macOS backend provides additional fields
- Friendly missing-`smartctl` guidance with native backend and install paths
- Read-only `smartctl` runner with fixed args and timeout
- Detailed SMART collection through `smartctl -x -j` is enabled by default
- `--debug` command diagnostics for command, exit code, timeout, and stderr without stdout payload logging
- `--debug` native IOKit probe diagnostics for NVMe Identify, SMART log, OCP Cloud SMART log, self-test log, and HID temperature sensor attempts
- `--debug` decoded macOS `IOReturn` values for native NVMe probe failures, including system, sub, and code fields
- Reason-code based health assessment
- Golden outputs for en/ja unicode/ascii at width 100
- GitHub CI and release workflow templates
- CI build verification for macOS amd64 and arm64
- Release workflow Formula metadata asset with source archive SHA256 command
- Homebrew Formula template
- Homebrew Formula update helper for owner, version, and source SHA256
- Manual Homebrew tap update workflow using `HOMEBREW_TAP_TOKEN`
- Local release readiness script covering Go, Ruby, Formula, and darwin builds

## Partially Implemented

- Health rules: implemented for SMART failure, NVMe critical warning including read-only mode, spare threshold, endurance, media errors, self-test failure, permission-required SMART data, missing required data, and temperature thresholds.
- External SSD support: represented through fixtures, best-effort SMART parsing, and diskutil SSD-name heuristics; real bridge handling remains dependent on hardware passthrough.
- Multiple drives: render and JSON support multiple snapshots; discovery-to-smartctl flow exists but needs broader real-world validation.
- Loop mode: refreshes output, computes basic deltas, recomputes auto width, and wakes on SIGWINCH; broader long-running terminal behavior needs real-world validation.
- Native backend: works without `smartctl` and reports macOS-visible NVMe SMART pass/fail, critical warning, endurance, composite and NAND sensor temperature, lifetime host I/O, cumulative I/O processing time from storage statistics, power counters, error counters, firmware, serial, transport, and NVMe revision. It also attempts OCP Log Page `0xC0` physical media writes, Identify temperature thresholds, and device self-test history, but Apple internal SSDs may reject OCP/self-test log pages and return zero for Identify threshold fields. Media/NAND physical writes are not inferred from host writes because the spec forbids that substitution.

## Not Yet Implemented

- Additional real-hardware validation for uncommon external enclosure plist variants
- Actual tap publication run with a real tap target and token

## Verification Commands

```bash
scripts/check_release_ready.sh
```

Real disk integration is opt-in:

```bash
DISK_SMI_INTEGRATION=1 go test ./internal/integration
```
