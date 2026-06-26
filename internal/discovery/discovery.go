package discovery

import (
	"bytes"
	"fmt"
	"strings"

	"disk-smi/internal/model"
	"howett.net/plist"
)

type Candidate struct {
	BSDName      string
	DevicePath   string
	Model        string
	CapacityByte model.BigCounter
	Protocol     model.Optional[string]
	Transport    model.Optional[string]
	Location     model.Optional[string]
	Internal     bool
	SolidState   bool
	SMART        SMARTInfo
}

type SMARTInfo struct {
	Status string
	Keys   SMARTKeys
}

type SMARTKeys struct {
	AvailableSpare          *uint64 `plist:"AVAILABLE_SPARE"`
	AvailableSpareThreshold *uint64 `plist:"AVAILABLE_SPARE_THRESHOLD"`
	ControllerBusyTime0     *uint64 `plist:"CONTROLLER_BUSY_TIME_0"`
	ControllerBusyTime1     *uint64 `plist:"CONTROLLER_BUSY_TIME_1"`
	DataUnitsRead0          *uint64 `plist:"DATA_UNITS_READ_0"`
	DataUnitsRead1          *uint64 `plist:"DATA_UNITS_READ_1"`
	DataUnitsWritten0       *uint64 `plist:"DATA_UNITS_WRITTEN_0"`
	DataUnitsWritten1       *uint64 `plist:"DATA_UNITS_WRITTEN_1"`
	HostReadCommands0       *uint64 `plist:"HOST_READ_COMMANDS_0"`
	HostReadCommands1       *uint64 `plist:"HOST_READ_COMMANDS_1"`
	HostWriteCommands0      *uint64 `plist:"HOST_WRITE_COMMANDS_0"`
	HostWriteCommands1      *uint64 `plist:"HOST_WRITE_COMMANDS_1"`
	MediaErrors0            *uint64 `plist:"MEDIA_ERRORS_0"`
	MediaErrors1            *uint64 `plist:"MEDIA_ERRORS_1"`
	NumErrorLogEntries0     *uint64 `plist:"NUM_ERROR_INFO_LOG_ENTRIES_0"`
	NumErrorLogEntries1     *uint64 `plist:"NUM_ERROR_INFO_LOG_ENTRIES_1"`
	PercentageUsed          *uint64 `plist:"PERCENTAGE_USED"`
	PowerCycles0            *uint64 `plist:"POWER_CYCLES_0"`
	PowerCycles1            *uint64 `plist:"POWER_CYCLES_1"`
	PowerOnHours0           *uint64 `plist:"POWER_ON_HOURS_0"`
	PowerOnHours1           *uint64 `plist:"POWER_ON_HOURS_1"`
	Temperature             *uint64 `plist:"TEMPERATURE"`
	UnsafeShutdowns0        *uint64 `plist:"UNSAFE_SHUTDOWNS_0"`
	UnsafeShutdowns1        *uint64 `plist:"UNSAFE_SHUTDOWNS_1"`
}

type listDocument struct {
	WholeDisks            []string       `plist:"WholeDisks"`
	AllDisksAndPartitions []listDiskItem `plist:"AllDisksAndPartitions"`
}

type listDiskItem struct {
	DeviceIdentifier string `plist:"DeviceIdentifier"`
	WholeDisk        bool   `plist:"WholeDisk"`
	Size             uint64 `plist:"Size"`
	Content          string `plist:"Content"`
}

type infoDocument struct {
	DeviceIdentifier string `plist:"DeviceIdentifier"`
	DeviceNode       string `plist:"DeviceNode"`
	WholeDisk        bool   `plist:"WholeDisk"`
	Internal         bool   `plist:"Internal"`
	SolidState       bool   `plist:"SolidState"`
	MediaName        string `plist:"MediaName"`
	MediaType        string `plist:"MediaType"`
	IORegistryName   string `plist:"IORegistryEntryName"`
	Content          string `plist:"Content"`
	VirtualPhysical  string `plist:"VirtualOrPhysical"`
	APFSContainerRef string `plist:"APFSContainerReference"`
	APFSStores       []struct {
		DeviceIdentifier  string `plist:"DeviceIdentifier"`
		APFSPhysicalStore string `plist:"APFSPhysicalStore"`
	} `plist:"APFSPhysicalStores"`
	Protocol    string    `plist:"Protocol"`
	BusProtocol string    `plist:"BusProtocol"`
	TotalSize   uint64    `plist:"TotalSize"`
	DiskSize    uint64    `plist:"DiskSize"`
	SMARTStatus string    `plist:"SMARTStatus"`
	SMARTKeys   SMARTKeys `plist:"SMARTDeviceSpecificKeysMayVaryNotGuaranteed"`
}

func ParseList(data []byte) ([]string, error) {
	var doc listDocument
	if err := plist.NewDecoder(bytes.NewReader(data)).Decode(&doc); err != nil {
		return nil, fmt.Errorf("parse diskutil list plist: %w", err)
	}

	seen := make(map[string]bool)
	var disks []string
	for _, disk := range doc.WholeDisks {
		if disk != "" && !seen[disk] {
			seen[disk] = true
			disks = append(disks, disk)
		}
	}
	for _, item := range doc.AllDisksAndPartitions {
		if item.DeviceIdentifier == "" || seen[item.DeviceIdentifier] {
			continue
		}
		if item.WholeDisk {
			seen[item.DeviceIdentifier] = true
			disks = append(disks, item.DeviceIdentifier)
		}
	}
	return disks, nil
}

func ParseInfo(data []byte) (Candidate, bool, error) {
	var doc infoDocument
	if err := plist.NewDecoder(bytes.NewReader(data)).Decode(&doc); err != nil {
		return Candidate{}, false, fmt.Errorf("parse diskutil info plist: %w", err)
	}
	if doc.DeviceIdentifier == "" {
		return Candidate{}, false, fmt.Errorf("diskutil info missing DeviceIdentifier")
	}
	if !doc.WholeDisk || !isSSD(doc) || isVirtualContainer(doc) {
		return Candidate{}, false, nil
	}

	path := doc.DeviceNode
	if path == "" {
		path = "/dev/" + doc.DeviceIdentifier
	}
	size := doc.TotalSize
	if size == 0 {
		size = doc.DiskSize
	}

	return Candidate{
		BSDName:      doc.DeviceIdentifier,
		DevicePath:   path,
		Model:        fallbackModel(doc.MediaName, doc.IORegistryName, doc.MediaType),
		CapacityByte: model.NewBigCounterString(fmt.Sprintf("%d", size)),
		Protocol:     optional(doc.Protocol),
		Transport:    optional(doc.BusProtocol),
		Location:     location(doc.Internal),
		Internal:     doc.Internal,
		SolidState:   doc.SolidState,
		SMART:        SMARTInfo{Status: doc.SMARTStatus, Keys: doc.SMARTKeys},
	}, true, nil
}

func isSSD(doc infoDocument) bool {
	if doc.SolidState {
		return true
	}
	text := strings.ToLower(strings.Join([]string{
		doc.MediaName,
		doc.MediaType,
		doc.IORegistryName,
		doc.Protocol,
		doc.BusProtocol,
	}, " "))
	for _, token := range []string{"ssd", "solid state", "solid-state", "nvme"} {
		if strings.Contains(text, token) {
			return true
		}
	}
	return false
}

func isVirtualContainer(doc infoDocument) bool {
	if strings.EqualFold(doc.VirtualPhysical, "Virtual") {
		return true
	}
	if doc.APFSContainerRef != "" || len(doc.APFSStores) > 0 {
		return true
	}
	content := strings.ToLower(doc.Content)
	if strings.Contains(content, "apfs_container") {
		return true
	}
	return strings.Contains(strings.ToLower(doc.IORegistryName), "apfs")
}

func fallbackModel(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return "UNKNOWN"
}

func optional(value string) model.Optional[string] {
	if value == "" {
		return model.None[string](model.MissingUnavailable)
	}
	return model.Some(value)
}

func location(internal bool) model.Optional[string] {
	if internal {
		return model.Some("internal")
	}
	return model.Some("external")
}
