//go:build !darwin || !cgo

package native

import (
	"context"
	"fmt"
)

func ReadNVMeSMARTLog(ctx context.Context, target string) (NVMeSMARTLog, error) {
	return NVMeSMARTLog{}, fmt.Errorf("native NVMe SMART log is unavailable in this build")
}

func ReadNVMeIdentify(ctx context.Context, target string) (NVMeIdentify, error) {
	return NVMeIdentify{}, fmt.Errorf("native NVMe identify is unavailable in this build")
}

func ReadNVMeSelfTestLog(ctx context.Context, target string) (NVMeSelfTestLog, error) {
	return NVMeSelfTestLog{}, fmt.Errorf("native NVMe self-test log is unavailable in this build")
}

func ReadNVMeLogPageRaw(ctx context.Context, target string, page uint, length int) ([]byte, error) {
	return nil, fmt.Errorf("native NVMe log page is unavailable in this build")
}

func ReadHIDTemperatureSensors(ctx context.Context) ([]HIDTemperatureSensor, error) {
	return nil, fmt.Errorf("native HID temperature sensors are unavailable in this build")
}
