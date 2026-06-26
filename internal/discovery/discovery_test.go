package discovery

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseList(t *testing.T) {
	data := readFixture(t, "list.plist")
	got, err := ParseList(data)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"disk0", "disk3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseList = %#v, want %#v", got, want)
	}
}

func TestParseInfoSSDWholeDisk(t *testing.T) {
	candidate, ok, err := ParseInfo(readFixture(t, "info-disk0.plist"))
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("ParseInfo did not return candidate")
	}
	if candidate.BSDName != "disk0" || candidate.DevicePath != "/dev/disk0" {
		t.Fatalf("candidate identity = %#v", candidate)
	}
	if candidate.Model != "APPLE SSD AP1024Z" {
		t.Fatalf("model = %q", candidate.Model)
	}
	if !candidate.Location.Valid || candidate.Location.Value != "internal" {
		t.Fatalf("location = %#v", candidate.Location)
	}
	if candidate.SMART.Status != "Verified" {
		t.Fatalf("SMART status = %q", candidate.SMART.Status)
	}
	if candidate.SMART.Keys.Temperature == nil || *candidate.SMART.Keys.Temperature != 311 {
		t.Fatalf("SMART temperature = %#v", candidate.SMART.Keys.Temperature)
	}
}

func TestParseInfoSkipsNonSSD(t *testing.T) {
	_, ok, err := ParseInfo(readFixture(t, "info-disk3.plist"))
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("non-SSD candidate was returned")
	}
}

func TestParseInfoSkipsAPFSVirtualContainer(t *testing.T) {
	_, ok, err := ParseInfo(readFixture(t, "info-apfs-container.plist"))
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("APFS virtual container was returned")
	}
}

func TestParseInfoExternalSSDWithoutSolidStateFlag(t *testing.T) {
	candidate, ok, err := ParseInfo(readFixture(t, "info-usb-ssd-no-solidstate.plist"))
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("external SSD candidate was not returned")
	}
	if candidate.BSDName != "disk4" || candidate.Model != "USB SSD Media" {
		t.Fatalf("candidate = %#v", candidate)
	}
	if !candidate.Location.Valid || candidate.Location.Value != "external" {
		t.Fatalf("location = %#v", candidate.Location)
	}
	if candidate.SolidState {
		t.Fatalf("candidate should preserve raw SolidState=false for heuristic match")
	}
}

func TestParseInfoSkipsExternalHDDWithoutSSDSignals(t *testing.T) {
	_, ok, err := ParseInfo(readFixture(t, "info-usb-hdd-no-solidstate.plist"))
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("external HDD was returned")
	}
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "diskutil", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
