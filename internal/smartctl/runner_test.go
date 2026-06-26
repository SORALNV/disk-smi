package smartctl

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRunnerUsesFixedArgs(t *testing.T) {
	path := fakeSmartctl(t, `#!/bin/sh
cat <<'JSON'
{"device":{"name":"/dev/disk0","type":"nvme","protocol":"NVMe"},"model_name":"TEST SSD","nvme_total_capacity":1000,"smart_status":{"passed":true},"nvme_smart_health_information_log":{"critical_warning":0,"percentage_used":1,"available_spare":100,"available_spare_threshold":10,"media_errors":0}}
JSON
`)

	result, err := Runner{Path: path}.Run(context.Background(), "disk0")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{path, "-a", "-j", "/dev/disk0"}
	if !reflect.DeepEqual(result.Command, want) {
		t.Fatalf("command = %#v, want %#v", result.Command, want)
	}
	if result.ExitCode != 0 {
		t.Fatalf("exit code = %d, want 0", result.ExitCode)
	}
}

func TestRunnerAllowsNonZeroExitWithJSONStdout(t *testing.T) {
	path := fakeSmartctl(t, `#!/bin/sh
cat <<'JSON'
{"device":{"name":"/dev/disk0","type":"nvme","protocol":"NVMe"},"model_name":"TEST SSD","nvme_total_capacity":1000,"smart_status":{"passed":true},"nvme_smart_health_information_log":{"critical_warning":0,"percentage_used":1,"available_spare":100,"available_spare_threshold":10,"media_errors":0}}
JSON
echo "warning text" >&2
exit 4
`)

	got, err := Runner{Path: path}.Snapshot(context.Background(), "/dev/disk0")
	if err != nil {
		t.Fatal(err)
	}
	if got.Result.ExitCode != 4 {
		t.Fatalf("exit code = %d, want 4", got.Result.ExitCode)
	}
	if got.Result.Stderr != "warning text\n" {
		t.Fatalf("stderr = %q", got.Result.Stderr)
	}
	if got.Snapshot.Device.Model != "TEST SSD" {
		t.Fatalf("model = %q", got.Snapshot.Device.Model)
	}
}

func TestRunnerRejectsUnsafeDeviceBeforeExec(t *testing.T) {
	path := fakeSmartctl(t, `#!/bin/sh
echo should-not-run >&2
exit 99
`)

	if _, err := (Runner{Path: path}).Run(context.Background(), "/dev/disk0;rm"); err == nil {
		t.Fatalf("unsafe device was accepted")
	}
}

func TestRunnerPermissionDenied(t *testing.T) {
	message, err := os.ReadFile(fixturePath("permission-denied.txt"))
	if err != nil {
		t.Fatal(err)
	}
	path := fakeSmartctl(t, `#!/bin/sh
cat >&2 <<'ERR'
`+string(message)+`ERR
exit 2
`)

	_, err = (Runner{Path: path}).Run(context.Background(), "disk0")
	if err == nil {
		t.Fatalf("permission denied run succeeded")
	}
	if !strings.Contains(err.Error(), "permission") {
		t.Fatalf("error = %v", err)
	}
}

func TestRunnerTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is Unix-only")
	}
	path := fakeSmartctl(t, `#!/bin/sh
sleep 1
`)

	result, err := Runner{Path: path, Timeout: 10 * time.Millisecond}.Run(context.Background(), "disk0")
	if err == nil {
		t.Fatalf("timeout run succeeded")
	}
	if !result.TimedOut {
		t.Fatalf("TimedOut = false")
	}
}

func fakeSmartctl(t *testing.T, script string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is Unix-only")
	}
	path := filepath.Join(t.TempDir(), "smartctl")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}
