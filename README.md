# disk-smi

`disk-smi` is a macOS SSD status viewer for the terminal. It displays SMART-derived SSD health, endurance, temperature, power, and I/O counters in a fixed-width grid inspired by `nvidia-smi`.

The project follows [docs/spec-v0.4.md](docs/spec-v0.4.md) as its implementation reference.

## Status

This repository is under active implementation. Current capabilities include:

- English and Japanese panel rendering
- Unicode and ASCII borders
- Display-cell width handling for Japanese text and ANSI color
- `smartctl -j` NVMe fixture parsing
- Best-effort SATA SMART attribute parsing
- `diskutil list/info -plist` discovery foundation
- Native macOS NVMe SMART log reading without `smartctl`
- JSON output with stable English keys
- Masked serials by default, with `--show-serial` opt-in
- Summary view, IEC units, and basic loop mode

## Usage

```bash
disk-smi
disk-smi disk0
disk-smi /dev/disk0
disk-smi -jp
disk-smi --summary
disk-smi --json-pretty
disk-smi --backend native
disk-smi --input testdata/smartctl/nvme-good.json
```

Useful options:

```text
-jp                 Japanese full display using the native macOS backend
--lang ja-JP        Japanese display
--ascii             ASCII borders
--width N           Panel width in display cells
-l N, --loop N      Refresh every N seconds, minimum 2
--summary           Compact multi-drive table
--json              JSON output
--json-pretty       Pretty JSON output
--iec               Use KiB/MiB/GiB/TiB
--show-serial       Show full serial number
--color auto        Color only on TTY
--color always      Always emit ANSI color
--color never       Disable ANSI color
--no-color          Disable ANSI color
--backend auto      Use smartctl when available, native fallback otherwise
--backend native    Use built-in macOS metadata backend without smartctl
--backend smartctl  Require smartctl for detailed SMART data
--debug             Print command diagnostics to stderr
```

## Safety

`disk-smi` is read-only. It does not erase, repair, partition, mount, unmount, or write to disks. External commands are executed without a shell and with fixed argument arrays.

By default, `disk-smi` uses detailed SMART retrieval through `smartctl` when available and supplements it with the built-in macOS backend where that adds fields. If `smartctl` is unavailable or blocked by permissions, `auto` falls back to the native backend. On Apple Silicon Macs, `-jp` uses the built-in native macOS path directly. The native backend reads `diskutil` SMART dictionaries, direct IOKit NVMe SMART/Identify data, OCP Cloud SMART Log Page `0xC0` when exposed by the controller, HID temperature sensor events, `system_profiler` NVMe metadata, and IORegistry controller/storage statistics, so Apple internal NVMe SSDs can report SMART status, critical warning, endurance, composite and NAND sensor temperature, lifetime host I/O, cumulative I/O processing time, power-on hours, power cycles, unsafe shutdowns, error counters, firmware, serial, transport, and NVMe revision without installing `smartctl`. OCP-capable NVMe drives can also report physical media writes through Log Page `0xC0`, and JSON input can parse OCP `physical_media_units_written` counters. Fields the drive or macOS does not expose are omitted from the terminal panel and preserved in JSON `missing_reasons`.

With `--debug`, native IOKit probes such as `nvme-identify`, `nvme-smart-log`, `nvme-ocp-cloud-smart-log`, `nvme-self-test-log`, and `hid-temperature-sensors` are reported alongside external command diagnostics. This makes unsupported controller paths visible without logging raw SMART payloads. JSON output also includes detailed SMART fields and `missing_reasons` entries for null values.

## I/O Counters

`disk-smi` shows SSD-reported lifetime host I/O counters. `Host reads` and `Host writes` are cumulative SMART/NVMe values recorded by the SSD itself.

macOS Activity Monitor shows OS-side disk I/O counters, such as data read and written by the current macOS session or driver statistics window. Those values are useful for current system activity, but they are not the same as the SSD's lifetime SMART counters, so they can differ from `disk-smi`.

In short: `disk-smi` is for SSD health and lifetime counters; Activity Monitor is for current Mac activity.

The Homebrew Formula installs `smartmontools` automatically for external drives and controller paths that still need `smartctl`. Source builds can run `disk-smi --backend native` or install:

```bash
brew install smartmontools
```

The tool does not automatically invoke `sudo`. If SMART data requires elevated permission, run:

```bash
sudo disk-smi
```

## Development

Run the standard checks:

```bash
scripts/check_release_ready.sh
```

The script runs formatting, Go tests, race tests, vet, whitespace checks, Ruby/Formula checks, Formula helper tests, and macOS `amd64`/`arm64` builds.

CI and unit tests use fixtures and do not access real disks.

Fixture coverage includes NVMe endurance thresholds, SMART failure, critical warnings, media errors, large counters, malformed JSON, SATA best-effort parsing, USB unavailable data, and permission-denied stderr.

CI also cross-builds macOS `amd64` and `arm64` binaries.

## Release

Tagged releases build macOS `amd64` and `arm64` binaries through `.github/workflows/release.yml`.
The release also publishes `formula-update.txt` with the GitHub source archive SHA256 and a ready-to-run Formula update command.

The Homebrew formula template lives at `Formula/disk-smi.rb`. Before publishing a real tap, replace the placeholder owner, version URL, and SHA256.

Use the helper after computing the source archive SHA256:

```bash
ruby scripts/update_formula.rb --owner <owner> --version 1.0.0 --sha256 <sha256>
```

To update a tap through GitHub Actions, add a `HOMEBREW_TAP_TOKEN` secret with write access to the tap repository and run the `Update Homebrew Tap` workflow with the tap repository, version, and SHA256 from `formula-update.txt`.
