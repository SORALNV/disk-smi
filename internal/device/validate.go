package device

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var bsdDiskPattern = regexp.MustCompile(`^disk[0-9]+$`)

type Disk struct {
	BSDName    string
	DevicePath string
	RawPath    string
}

func Parse(value string) (Disk, error) {
	if value == "" {
		return Disk{}, fmt.Errorf("device is required")
	}
	if strings.ContainsAny(value, " \t\r\n;&|`$<>*?[]{}()!\\\"'") || strings.Contains(value, "..") {
		return Disk{}, fmt.Errorf("invalid device name %q", value)
	}

	switch {
	case bsdDiskPattern.MatchString(value):
		return Disk{BSDName: value, DevicePath: "/dev/" + value, RawPath: "/dev/r" + value}, nil
	case strings.HasPrefix(value, "/dev/disk"):
		name := filepath.Base(value)
		if !bsdDiskPattern.MatchString(name) {
			return Disk{}, fmt.Errorf("invalid device name %q", value)
		}
		return Disk{BSDName: name, DevicePath: "/dev/" + name, RawPath: "/dev/r" + name}, nil
	case strings.HasPrefix(value, "/dev/rdisk"):
		name := strings.TrimPrefix(filepath.Base(value), "r")
		if !bsdDiskPattern.MatchString(name) {
			return Disk{}, fmt.Errorf("invalid device name %q", value)
		}
		return Disk{BSDName: name, DevicePath: "/dev/" + name, RawPath: "/dev/r" + name}, nil
	default:
		return Disk{}, fmt.Errorf("invalid device name %q", value)
	}
}
