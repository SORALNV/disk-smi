//go:build darwin && cgo

package native

/*
#cgo LDFLAGS: -framework IOKit -framework CoreFoundation
#include <CoreFoundation/CoreFoundation.h>
#include <CoreFoundation/CFPlugInCOM.h>
#include <IOKit/IOCFPlugIn.h>
#include <IOKit/IOKitLib.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#define kIOPropertyNVMeSMARTCapableKey "NVMe SMART Capable"

typedef struct IONVMeSMARTInterface {
	IUNKNOWN_C_GUTS;
	UInt16 version;
	UInt16 revision;
	IOReturn (*SMARTReadData)(void *interface, void *NVMeSMARTData);
	IOReturn (*GetIdentifyData)(void *interface, void *NVMeIdentifyControllerStruct, unsigned int ns);
	UInt64 reserved0;
	UInt64 reserved1;
	IOReturn (*GetLogPage)(void *interface, void *data, unsigned int logPageId, unsigned int numDWords);
	UInt64 reserved2;
	UInt64 reserved3;
	UInt64 reserved4;
	UInt64 reserved5;
	UInt64 reserved6;
	UInt64 reserved7;
	UInt64 reserved8;
	UInt64 reserved9;
	UInt64 reserved10;
	UInt64 reserved11;
	UInt64 reserved12;
	UInt64 reserved13;
	UInt64 reserved14;
	UInt64 reserved15;
	UInt64 reserved16;
	UInt64 reserved17;
	UInt64 reserved18;
	UInt64 reserved19;
} IONVMeSMARTInterface;

static CFUUIDRef disk_smi_nvme_user_client_type_id(void) {
	return CFUUIDGetConstantUUIDWithBytes(NULL,
		0xAA, 0x0F, 0xA6, 0xF9, 0xC2, 0xD6, 0x45, 0x7F,
		0xB1, 0x0B, 0x59, 0xA1, 0x32, 0x53, 0x29, 0x2F);
}

static CFUUIDRef disk_smi_nvme_interface_id(void) {
	return CFUUIDGetConstantUUIDWithBytes(NULL,
		0xCC, 0xD1, 0xDB, 0x19, 0xFD, 0x9A, 0x4D, 0xAF,
		0xBF, 0x95, 0x12, 0x45, 0x4B, 0x23, 0x0A, 0xB6);
}

static void disk_smi_set_error(char *errbuf, int errlen, const char *message, IOReturn code) {
	if (errbuf && errlen > 0) {
		if (code) {
			snprintf(errbuf, errlen, "%s: 0x%x (system=0x%x, sub=0x%x, code=%d)",
				message,
				code,
				(code >> 26) & 0x3f,
				(code >> 14) & 0xfff,
				code & 0x3fff);
		} else {
			snprintf(errbuf, errlen, "%s", message);
		}
	}
}

static io_object_t disk_smi_find_nvme_smart_service(const char *bsdName, char *errbuf, int errlen) {
	CFMutableDictionaryRef matcher = IOBSDNameMatching(kIOMainPortDefault, 0, bsdName);
	if (!matcher) {
		disk_smi_set_error(errbuf, errlen, "IOBSDNameMatching failed", 0);
		return MACH_PORT_NULL;
	}
	io_object_t current = IOServiceGetMatchingService(kIOMainPortDefault, matcher);
	if (!current) {
		disk_smi_set_error(errbuf, errlen, "disk service not found", 0);
		return MACH_PORT_NULL;
	}
	while (current) {
		CFTypeRef capable = IORegistryEntryCreateCFProperty(
			current, CFSTR(kIOPropertyNVMeSMARTCapableKey), kCFAllocatorDefault, 0);
		if (capable) {
			CFRelease(capable);
			return current;
		}
		io_object_t parent = MACH_PORT_NULL;
		io_name_t plane;
		memset(plane, 0, sizeof(plane));
		strncpy(plane, kIOServicePlane, sizeof(plane) - 1);
		IOReturn err = IORegistryEntryGetParentEntry(current, plane, &parent);
		IOObjectRelease(current);
		if (err != kIOReturnSuccess || !parent) {
			disk_smi_set_error(errbuf, errlen, "NVMe SMART capable service not found", err);
			return MACH_PORT_NULL;
		}
		current = parent;
	}
	disk_smi_set_error(errbuf, errlen, "NVMe SMART capable service not found", 0);
	return MACH_PORT_NULL;
}

static int disk_smi_with_nvme_interface(const char *bsdName, IONVMeSMARTInterface ***smartOut, IOCFPlugInInterface ***pluginOut, io_object_t *serviceOut, char *errbuf, int errlen) {
	if (!bsdName || !smartOut || !pluginOut || !serviceOut) {
		disk_smi_set_error(errbuf, errlen, "invalid argument", 0);
		return -1;
	}
	io_object_t service = disk_smi_find_nvme_smart_service(bsdName, errbuf, errlen);
	if (!service) {
		return -1;
	}

	IOCFPlugInInterface **plugin = NULL;
	IONVMeSMARTInterface **smart = NULL;
	SInt32 score = 0;
	IOReturn err = IOCreatePlugInInterfaceForService(
		service,
		disk_smi_nvme_user_client_type_id(),
		kIOCFPlugInInterfaceID,
		&plugin,
		&score);
	if (err != kIOReturnSuccess || !plugin) {
		IOObjectRelease(service);
		disk_smi_set_error(errbuf, errlen, "IOCreatePlugInInterfaceForService failed", err);
		return -1;
	}

	HRESULT qerr = (*plugin)->QueryInterface(
		plugin,
		CFUUIDGetUUIDBytes(disk_smi_nvme_interface_id()),
		(LPVOID *)&smart);
	if (qerr || !smart || !*smart) {
		IODestroyPlugInInterface(plugin);
		IOObjectRelease(service);
		disk_smi_set_error(errbuf, errlen, "QueryInterface for IONVMeSMARTInterface failed", (IOReturn)qerr);
		return -1;
	}
	*smartOut = smart;
	*pluginOut = plugin;
	*serviceOut = service;
	return 0;
}

static void disk_smi_release_nvme_interface(IONVMeSMARTInterface **smart, IOCFPlugInInterface **plugin, io_object_t service) {
	if (smart && *smart) {
		(*smart)->Release(smart);
	}
	if (plugin) {
		IODestroyPlugInInterface(plugin);
	}
	if (service) {
		IOObjectRelease(service);
	}
}

static int disk_smi_read_nvme_log_page(const char *bsdName, unsigned int page, unsigned char *out, int outLen, char *errbuf, int errlen) {
	if (!bsdName || !out || outLen < 4 || (outLen % 4) != 0) {
		disk_smi_set_error(errbuf, errlen, "invalid argument", 0);
		return -1;
	}
	IOCFPlugInInterface **plugin = NULL;
	IONVMeSMARTInterface **smart = NULL;
	io_object_t service = MACH_PORT_NULL;
	if (disk_smi_with_nvme_interface(bsdName, &smart, &plugin, &service, errbuf, errlen) != 0) {
		return -1;
	}
	memset(out, 0, outLen);
	IOReturn err = (*smart)->GetLogPage(smart, out, page, outLen / 4 - 1);
	disk_smi_release_nvme_interface(smart, plugin, service);
	if (err != kIOReturnSuccess) {
		disk_smi_set_error(errbuf, errlen, "GetLogPage failed", err);
		return -1;
	}
	return 0;
}

static int disk_smi_read_nvme_smart_data(const char *bsdName, unsigned char *out, int outLen, char *errbuf, int errlen) {
	if (!bsdName || !out || outLen < 512) {
		disk_smi_set_error(errbuf, errlen, "invalid argument", 0);
		return -1;
	}
	IOCFPlugInInterface **plugin = NULL;
	IONVMeSMARTInterface **smart = NULL;
	io_object_t service = MACH_PORT_NULL;
	if (disk_smi_with_nvme_interface(bsdName, &smart, &plugin, &service, errbuf, errlen) != 0) {
		return -1;
	}
	memset(out, 0, outLen);
	IOReturn err = (*smart)->SMARTReadData(smart, out);
	disk_smi_release_nvme_interface(smart, plugin, service);
	if (err != kIOReturnSuccess) {
		disk_smi_set_error(errbuf, errlen, "SMARTReadData failed", err);
		return -1;
	}
	return 0;
}

static int disk_smi_read_nvme_identify(const char *bsdName, unsigned char *out, int outLen, char *errbuf, int errlen) {
	if (!bsdName || !out || outLen < 4096) {
		disk_smi_set_error(errbuf, errlen, "invalid argument", 0);
		return -1;
	}
	IOCFPlugInInterface **plugin = NULL;
	IONVMeSMARTInterface **smart = NULL;
	io_object_t service = MACH_PORT_NULL;
	if (disk_smi_with_nvme_interface(bsdName, &smart, &plugin, &service, errbuf, errlen) != 0) {
		return -1;
	}
	memset(out, 0, outLen);
	IOReturn err = (*smart)->GetIdentifyData(smart, out, 0);
	disk_smi_release_nvme_interface(smart, plugin, service);
	if (err != kIOReturnSuccess) {
		disk_smi_set_error(errbuf, errlen, "GetIdentifyData failed", err);
		return -1;
	}
	return 0;
}
*/
import "C"

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"unsafe"
)

func ReadNVMeSMARTLog(ctx context.Context, target string) (NVMeSMARTLog, error) {
	data, err := readNVMeLogPage(ctx, target, 0x02, nvmeSMARTLogLength)
	if err != nil {
		data, err = readNVMeSMARTData(ctx, target)
		if err != nil {
			return NVMeSMARTLog{}, err
		}
	}
	return ParseNVMeSMARTLog(data)
}

func readNVMeLogPage(ctx context.Context, target string, page uint, length int) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	bsdName := filepath.Base(target)
	if strings.HasPrefix(bsdName, "r") {
		bsdName = strings.TrimPrefix(bsdName, "r")
	}
	cName := C.CString(bsdName)
	defer C.free(unsafe.Pointer(cName))
	buffer := make([]byte, length)
	errbuf := make([]byte, 256)
	rc := C.disk_smi_read_nvme_log_page(
		cName,
		C.uint(page),
		(*C.uchar)(unsafe.Pointer(&buffer[0])),
		C.int(len(buffer)),
		(*C.char)(unsafe.Pointer(&errbuf[0])),
		C.int(len(errbuf)))
	if rc != 0 {
		message := C.GoString((*C.char)(unsafe.Pointer(&errbuf[0])))
		if message == "" {
			message = "unknown error"
		}
		return nil, fmt.Errorf("native NVMe log page 0x%02x failed: %s", page, message)
	}
	return buffer, nil
}

func ReadNVMeLogPageRaw(ctx context.Context, target string, page uint, length int) ([]byte, error) {
	return readNVMeLogPage(ctx, target, page, length)
}

func readNVMeSMARTData(ctx context.Context, target string) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	bsdName := filepath.Base(target)
	if strings.HasPrefix(bsdName, "r") {
		bsdName = strings.TrimPrefix(bsdName, "r")
	}
	cName := C.CString(bsdName)
	defer C.free(unsafe.Pointer(cName))
	buffer := make([]byte, nvmeSMARTLogLength)
	errbuf := make([]byte, 256)
	rc := C.disk_smi_read_nvme_smart_data(
		cName,
		(*C.uchar)(unsafe.Pointer(&buffer[0])),
		C.int(len(buffer)),
		(*C.char)(unsafe.Pointer(&errbuf[0])),
		C.int(len(errbuf)))
	if rc != 0 {
		message := C.GoString((*C.char)(unsafe.Pointer(&errbuf[0])))
		if message == "" {
			message = "unknown error"
		}
		return nil, fmt.Errorf("native NVMe SMARTReadData failed: %s", message)
	}
	return buffer, nil
}

func ReadNVMeIdentify(ctx context.Context, target string) (NVMeIdentify, error) {
	select {
	case <-ctx.Done():
		return NVMeIdentify{}, ctx.Err()
	default:
	}
	bsdName := filepath.Base(target)
	if strings.HasPrefix(bsdName, "r") {
		bsdName = strings.TrimPrefix(bsdName, "r")
	}
	cName := C.CString(bsdName)
	defer C.free(unsafe.Pointer(cName))
	buffer := make([]byte, nvmeIdentifyLength)
	errbuf := make([]byte, 256)
	rc := C.disk_smi_read_nvme_identify(
		cName,
		(*C.uchar)(unsafe.Pointer(&buffer[0])),
		C.int(len(buffer)),
		(*C.char)(unsafe.Pointer(&errbuf[0])),
		C.int(len(errbuf)))
	if rc != 0 {
		message := C.GoString((*C.char)(unsafe.Pointer(&errbuf[0])))
		if message == "" {
			message = "unknown error"
		}
		return NVMeIdentify{}, fmt.Errorf("native NVMe identify failed: %s", message)
	}
	return ParseNVMeIdentify(buffer)
}

func ReadNVMeSelfTestLog(ctx context.Context, target string) (NVMeSelfTestLog, error) {
	data, err := readNVMeLogPage(ctx, target, 0x06, nvmeSelfTestLogLength)
	if err != nil {
		return NVMeSelfTestLog{}, err
	}
	return ParseNVMeSelfTestLog(data)
}
