package native

import (
	"os"
	"testing"
)

func TestManualReadNVMeIdentify(t *testing.T) {
	target := os.Getenv("DISK_SMI_MANUAL_NVME_TARGET")
	if target == "" {
		t.Skip("set DISK_SMI_MANUAL_NVME_TARGET to run")
	}
	identify, err := ReadNVMeIdentify(t.Context(), target)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("identify = %#v", identify)
}

func TestManualReadNVMeSelfTestLog(t *testing.T) {
	target := os.Getenv("DISK_SMI_MANUAL_NVME_TARGET")
	if target == "" {
		t.Skip("set DISK_SMI_MANUAL_NVME_TARGET to run")
	}
	log, err := ReadNVMeSelfTestLog(t.Context(), target)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("self-test = %#v", log)
}
