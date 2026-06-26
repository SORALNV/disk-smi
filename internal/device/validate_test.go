package device

import "testing"

func TestParseValidDevices(t *testing.T) {
	tests := []struct {
		input string
		name  string
		path  string
		raw   string
	}{
		{input: "disk0", name: "disk0", path: "/dev/disk0", raw: "/dev/rdisk0"},
		{input: "/dev/disk1", name: "disk1", path: "/dev/disk1", raw: "/dev/rdisk1"},
		{input: "/dev/rdisk2", name: "disk2", path: "/dev/disk2", raw: "/dev/rdisk2"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := Parse(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if got.BSDName != tt.name || got.DevicePath != tt.path || got.RawPath != tt.raw {
				t.Fatalf("Parse(%q) = %#v", tt.input, got)
			}
		})
	}
}

func TestParseRejectsUnsafeDevices(t *testing.T) {
	tests := []string{
		"",
		"disk",
		"disk0s1",
		"/tmp/disk0",
		"/dev/disk0;rm",
		"/dev/disk0 extra",
		"../disk0",
		"/dev/disk0\n/dev/disk1",
		"/dev/rdiskx",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			if _, err := Parse(input); err == nil {
				t.Fatalf("Parse(%q) succeeded", input)
			}
		})
	}
}
