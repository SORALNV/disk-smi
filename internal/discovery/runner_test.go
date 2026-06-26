package discovery

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestRunnerDiscover(t *testing.T) {
	path := fakeDiskutil(t)
	result, err := Runner{Path: path}.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Candidates) != 1 {
		t.Fatalf("candidate count = %d, want 1", len(result.Candidates))
	}
	if result.Candidates[0].BSDName != "disk0" {
		t.Fatalf("candidate = %#v", result.Candidates[0])
	}

	wantList := []string{path, "list", "-plist"}
	if !reflect.DeepEqual(result.ListResult.Command, wantList) {
		t.Fatalf("list command = %#v, want %#v", result.ListResult.Command, wantList)
	}
	wantInfo := []string{path, "info", "-plist", "/dev/disk0"}
	if !reflect.DeepEqual(result.InfoResults[0].Command, wantInfo) {
		t.Fatalf("info command = %#v, want %#v", result.InfoResults[0].Command, wantInfo)
	}
}

func TestRunnerInspect(t *testing.T) {
	path := fakeDiskutil(t)
	result, err := Runner{Path: path}.Inspect(context.Background(), "disk0")
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Fatal("inspect did not return candidate")
	}
	if result.Candidate.BSDName != "disk0" {
		t.Fatalf("candidate = %#v", result.Candidate)
	}
	want := []string{path, "info", "-plist", "/dev/disk0"}
	if !reflect.DeepEqual(result.InfoResult.Command, want) {
		t.Fatalf("info command = %#v, want %#v", result.InfoResult.Command, want)
	}
}

func fakeDiskutil(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is Unix-only")
	}
	root := filepath.Join("..", "..", "testdata", "diskutil")
	script := `#!/bin/sh
if [ "$1" = "list" ] && [ "$2" = "-plist" ]; then
  cat "` + filepath.Join(root, "list.plist") + `"
  exit 0
fi
if [ "$1" = "info" ] && [ "$2" = "-plist" ] && [ "$3" = "/dev/disk0" ]; then
  cat "` + filepath.Join(root, "info-disk0.plist") + `"
  exit 0
fi
if [ "$1" = "info" ] && [ "$2" = "-plist" ] && [ "$3" = "/dev/disk3" ]; then
  cat "` + filepath.Join(root, "info-disk3.plist") + `"
  exit 0
fi
echo "unexpected args: $@" >&2
exit 2
`
	path := filepath.Join(t.TempDir(), "diskutil")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}
