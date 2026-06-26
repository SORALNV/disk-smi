package loopstats

import (
	"math/big"
	"time"

	"disk-smi/internal/model"
)

func Compute(previous, current []model.DriveSnapshot, interval time.Duration) map[string]model.LoopStats {
	if len(previous) == 0 || len(current) == 0 || interval <= 0 {
		return nil
	}
	prevByName := make(map[string]model.DriveSnapshot, len(previous))
	for _, snapshot := range previous {
		prevByName[snapshot.Device.BSDName] = snapshot
	}

	out := make(map[string]model.LoopStats)
	for _, snapshot := range current {
		prev, ok := prevByName[snapshot.Device.BSDName]
		if !ok {
			continue
		}
		stats := computeOne(prev, snapshot, interval)
		if stats.Valid || stats.Reset {
			out[snapshot.Device.BSDName] = stats
		}
	}
	return out
}

func computeOne(previous, current model.DriveSnapshot, interval time.Duration) model.LoopStats {
	seconds := int64(interval / time.Second)
	if seconds <= 0 {
		seconds = 1
	}
	stats := model.LoopStats{}
	stats.ReadRateBytesPerSec, stats.Reset = rate(previous.Metrics.HostReadsBytes, current.Metrics.HostReadsBytes, seconds)
	writeRate, reset := rate(previous.Metrics.HostWritesBytes, current.Metrics.HostWritesBytes, seconds)
	stats.WriteRateBytesPerSec = writeRate
	stats.Reset = stats.Reset || reset
	stats.ReadIOPS, reset = rate(previous.Metrics.ReadCommands, current.Metrics.ReadCommands, seconds)
	stats.Reset = stats.Reset || reset
	stats.WriteIOPS, reset = rate(previous.Metrics.WriteCommands, current.Metrics.WriteCommands, seconds)
	stats.Reset = stats.Reset || reset
	if previous.Metrics.TemperatureCelsius.Valid && current.Metrics.TemperatureCelsius.Valid {
		stats.TemperatureChangeC = model.Some(current.Metrics.TemperatureCelsius.Value - previous.Metrics.TemperatureCelsius.Value)
		stats.Valid = true
	}
	if stats.ReadRateBytesPerSec.Valid || stats.WriteRateBytesPerSec.Valid || stats.ReadIOPS.Valid || stats.WriteIOPS.Valid {
		stats.Valid = true
	}
	if stats.Reset {
		stats.Valid = false
	}
	return stats
}

func rate(previous, current model.Optional[model.BigCounter], seconds int64) (model.Optional[model.BigCounter], bool) {
	if !previous.Valid || !current.Valid {
		return model.None[model.BigCounter](model.MissingUnavailable), false
	}
	prev, ok := new(big.Int).SetString(previous.Value.String(), 10)
	if !ok {
		return model.None[model.BigCounter](model.MissingUnknown), false
	}
	curr, ok := new(big.Int).SetString(current.Value.String(), 10)
	if !ok {
		return model.None[model.BigCounter](model.MissingUnknown), false
	}
	if curr.Cmp(prev) < 0 {
		return model.None[model.BigCounter](model.MissingUnknown), true
	}
	delta := new(big.Int).Sub(curr, prev)
	delta.Div(delta, big.NewInt(seconds))
	return model.Some(model.NewBigCounterString(delta.String())), false
}
