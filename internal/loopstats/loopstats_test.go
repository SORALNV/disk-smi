package loopstats

import (
	"testing"
	"time"

	"disk-smi/internal/model"
)

func TestCompute(t *testing.T) {
	prev := model.SyntheticSnapshot()
	curr := model.SyntheticSnapshot()
	curr.Metrics.HostReadsBytes = model.Some(model.NewBigCounterString("3500001000000"))
	curr.Metrics.HostWritesBytes = model.Some(model.NewBigCounterString("4200002000000"))
	curr.Metrics.ReadCommands = model.Some(model.NewBigCounter(82020334))
	curr.Metrics.WriteCommands = model.Some(model.NewBigCounter(61276004))
	curr.Metrics.TemperatureCelsius = model.Some(int64(39))

	stats := Compute([]model.DriveSnapshot{prev}, []model.DriveSnapshot{curr}, 2*time.Second)["disk0"]
	if !stats.Valid {
		t.Fatalf("stats invalid: %#v", stats)
	}
	if got := stats.ReadRateBytesPerSec.Value.String(); got != "500000" {
		t.Fatalf("read rate = %s", got)
	}
	if got := stats.WriteRateBytesPerSec.Value.String(); got != "1000000" {
		t.Fatalf("write rate = %s", got)
	}
	if got := stats.ReadIOPS.Value.String(); got != "500" {
		t.Fatalf("read IOPS = %s", got)
	}
	if got := stats.WriteIOPS.Value.String(); got != "1000" {
		t.Fatalf("write IOPS = %s", got)
	}
	if got := stats.TemperatureChangeC.Value; got != 3 {
		t.Fatalf("temperature change = %d", got)
	}
}

func TestComputeReset(t *testing.T) {
	prev := model.SyntheticSnapshot()
	curr := model.SyntheticSnapshot()
	curr.Metrics.HostReadsBytes = model.Some(model.NewBigCounterString("1"))

	stats := Compute([]model.DriveSnapshot{prev}, []model.DriveSnapshot{curr}, 2*time.Second)["disk0"]
	if !stats.Reset {
		t.Fatalf("reset not detected: %#v", stats)
	}
	if stats.Valid {
		t.Fatalf("reset stats should not be valid")
	}
}
