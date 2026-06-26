//go:build darwin && cgo

package native

/*
#cgo LDFLAGS: -framework IOKit -framework CoreFoundation
#include <CoreFoundation/CoreFoundation.h>
#include <IOKit/hidsystem/IOHIDEventSystemClient.h>
#include <IOKit/hidsystem/IOHIDServiceClient.h>
#include <math.h>
#include <stdio.h>
#include <string.h>

typedef struct __IOHIDEvent *IOHIDEventRef;

#ifdef __LP64__
typedef double IOHIDFloat;
#else
typedef float IOHIDFloat;
#endif

IOHIDEventSystemClientRef IOHIDEventSystemClientCreate(CFAllocatorRef allocator);
int IOHIDEventSystemClientSetMatching(IOHIDEventSystemClientRef client, CFDictionaryRef match);
IOHIDEventRef IOHIDServiceClientCopyEvent(IOHIDServiceClientRef service, int64_t type, int32_t options, int64_t timestamp);
IOHIDFloat IOHIDEventGetFloatValue(IOHIDEventRef event, int32_t field);

#define disk_smi_hid_event_field_base(type) ((type) << 16)
#define disk_smi_hid_event_type_temperature 15
#define disk_smi_hid_temperature_usage_page 0xff00
#define disk_smi_hid_temperature_usage 5

typedef struct disk_smi_hid_temperature_sensor {
	char name[128];
	double celsius;
} disk_smi_hid_temperature_sensor;

static void disk_smi_hid_set_error(char *errbuf, int errlen, const char *message) {
	if (errbuf && errlen > 0) {
		snprintf(errbuf, errlen, "%s", message);
	}
}

static CFDictionaryRef disk_smi_hid_temperature_matching(void) {
	int page = disk_smi_hid_temperature_usage_page;
	int usage = disk_smi_hid_temperature_usage;
	CFStringRef keys[2] = { CFSTR("PrimaryUsagePage"), CFSTR("PrimaryUsage") };
	CFNumberRef values[2] = {
		CFNumberCreate(kCFAllocatorDefault, kCFNumberIntType, &page),
		CFNumberCreate(kCFAllocatorDefault, kCFNumberIntType, &usage),
	};
	if (!values[0] || !values[1]) {
		if (values[0]) CFRelease(values[0]);
		if (values[1]) CFRelease(values[1]);
		return NULL;
	}
	CFDictionaryRef matcher = CFDictionaryCreate(
		kCFAllocatorDefault,
		(const void **)keys,
		(const void **)values,
		2,
		&kCFTypeDictionaryKeyCallBacks,
		&kCFTypeDictionaryValueCallBacks);
	CFRelease(values[0]);
	CFRelease(values[1]);
	return matcher;
}

static int disk_smi_copy_hid_temperature_sensors(disk_smi_hid_temperature_sensor *out, int max, char *errbuf, int errlen) {
	if (!out || max <= 0) {
		disk_smi_hid_set_error(errbuf, errlen, "invalid argument");
		return -1;
	}
	CFDictionaryRef matcher = disk_smi_hid_temperature_matching();
	if (!matcher) {
		disk_smi_hid_set_error(errbuf, errlen, "HID temperature matcher allocation failed");
		return -1;
	}
	IOHIDEventSystemClientRef client = IOHIDEventSystemClientCreate(kCFAllocatorDefault);
	if (!client) {
		CFRelease(matcher);
		disk_smi_hid_set_error(errbuf, errlen, "IOHIDEventSystemClientCreate failed");
		return -1;
	}
	IOHIDEventSystemClientSetMatching(client, matcher);
	CFRelease(matcher);

	CFArrayRef services = IOHIDEventSystemClientCopyServices(client);
	if (!services) {
		CFRelease(client);
		disk_smi_hid_set_error(errbuf, errlen, "IOHIDEventSystemClientCopyServices failed");
		return -1;
	}

	int written = 0;
	CFIndex count = CFArrayGetCount(services);
	for (CFIndex i = 0; i < count && written < max; i++) {
		IOHIDServiceClientRef service = (IOHIDServiceClientRef)CFArrayGetValueAtIndex(services, i);
		if (!service) {
			continue;
		}
		char name[128] = {0};
		CFTypeRef product = IOHIDServiceClientCopyProperty(service, CFSTR("Product"));
		if (product && CFGetTypeID(product) == CFStringGetTypeID()) {
			CFStringGetCString((CFStringRef)product, name, sizeof(name), kCFStringEncodingUTF8);
		}
		if (product) {
			CFRelease(product);
		}
		if (name[0] == '\0') {
			snprintf(name, sizeof(name), "Unknown");
		}

		IOHIDEventRef event = IOHIDServiceClientCopyEvent(service, disk_smi_hid_event_type_temperature, 0, 0);
		if (!event) {
			continue;
		}
		double value = IOHIDEventGetFloatValue(event, disk_smi_hid_event_field_base(disk_smi_hid_event_type_temperature));
		CFRelease(event);
		if (isnan(value) || isinf(value)) {
			continue;
		}
		snprintf(out[written].name, sizeof(out[written].name), "%s", name);
		out[written].celsius = value;
		written++;
	}

	CFRelease(services);
	CFRelease(client);
	return written;
}
*/
import "C"

import (
	"context"
	"fmt"
	"unsafe"
)

const maxHIDTemperatureSensors = 64

func ReadHIDTemperatureSensors(ctx context.Context) ([]HIDTemperatureSensor, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var out [maxHIDTemperatureSensors]C.disk_smi_hid_temperature_sensor
	errbuf := (*C.char)(C.malloc(256))
	if errbuf == nil {
		return nil, fmt.Errorf("allocate HID temperature error buffer")
	}
	defer C.free(unsafe.Pointer(errbuf))
	*errbuf = 0

	count := int(C.disk_smi_copy_hid_temperature_sensors(&out[0], C.int(len(out)), errbuf, 256))
	if count < 0 {
		return nil, fmt.Errorf("native HID temperature sensors failed: %s", C.GoString(errbuf))
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	sensors := make([]HIDTemperatureSensor, 0, count)
	for index := 0; index < count; index++ {
		sensors = append(sensors, HIDTemperatureSensor{
			Name:    C.GoString(&out[index].name[0]),
			Celsius: float64(out[index].celsius),
		})
	}
	return sensors, nil
}
