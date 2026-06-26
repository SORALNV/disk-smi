package health

import (
	"reflect"
	"testing"

	"disk-smi/internal/model"
)

func baseMetrics() model.DriveMetrics {
	return model.DriveMetrics{
		SMARTPassed:           model.Some(true),
		CriticalWarning:       model.Some(uint64(0)),
		EnduranceUsedPercent:  model.Some(uint64(30)),
		AvailableSparePercent: model.Some(uint64(100)),
		SpareThresholdPercent: model.Some(uint64(10)),
		MediaErrors:           model.Some(model.NewBigCounter(0)),
	}
}

func TestAssess(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*model.DriveMetrics)
		status  model.OverallStatus
		quality model.DataQuality
		reasons []string
	}{
		{
			name:    "good",
			status:  model.StatusGood,
			quality: model.DataQualityFull,
		},
		{
			name: "endurance 90 is caution",
			mutate: func(metrics *model.DriveMetrics) {
				metrics.EnduranceUsedPercent = model.Some(uint64(90))
			},
			status:  model.StatusCaution,
			quality: model.DataQualityFull,
			reasons: []string{EnduranceLow},
		},
		{
			name: "endurance 100 is caution not critical",
			mutate: func(metrics *model.DriveMetrics) {
				metrics.EnduranceUsedPercent = model.Some(uint64(100))
			},
			status:  model.StatusCaution,
			quality: model.DataQualityFull,
			reasons: []string{EnduranceRatedLimitReached},
		},
		{
			name: "endurance 126 keeps caution semantics",
			mutate: func(metrics *model.DriveMetrics) {
				metrics.EnduranceUsedPercent = model.Some(uint64(126))
			},
			status:  model.StatusCaution,
			quality: model.DataQualityFull,
			reasons: []string{EnduranceRatedLimitReached},
		},
		{
			name: "smart failed is critical",
			mutate: func(metrics *model.DriveMetrics) {
				metrics.SMARTPassed = model.Some(false)
			},
			status:  model.StatusCritical,
			quality: model.DataQualityFull,
			reasons: []string{SMARTFailed},
		},
		{
			name: "critical warning is critical",
			mutate: func(metrics *model.DriveMetrics) {
				metrics.CriticalWarning = model.Some(uint64(1))
			},
			status:  model.StatusCritical,
			quality: model.DataQualityFull,
			reasons: []string{CriticalWarningActive},
		},
		{
			name: "read-only critical warning is specifically reported",
			mutate: func(metrics *model.DriveMetrics) {
				metrics.CriticalWarning = model.Some(uint64(8))
			},
			status:  model.StatusCritical,
			quality: model.DataQualityFull,
			reasons: []string{CriticalWarningActive, CriticalWarningReadOnly},
		},
		{
			name: "missing critical warning does not hide other good SMART data",
			mutate: func(metrics *model.DriveMetrics) {
				metrics.CriticalWarning = model.None[uint64](model.MissingUnsupported)
			},
			status:  model.StatusGood,
			quality: model.DataQualityFull,
		},
		{
			name: "available spare below threshold is critical",
			mutate: func(metrics *model.DriveMetrics) {
				metrics.AvailableSparePercent = model.Some(uint64(9))
			},
			status:  model.StatusCritical,
			quality: model.DataQualityFull,
			reasons: []string{AvailableSpareBelowThreshold},
		},
		{
			name: "media errors are caution",
			mutate: func(metrics *model.DriveMetrics) {
				metrics.MediaErrors = model.Some(model.NewBigCounter(1))
			},
			status:  model.StatusCaution,
			quality: model.DataQualityFull,
			reasons: []string{MediaErrorsPresent},
		},
		{
			name: "warning temperature is caution",
			mutate: func(metrics *model.DriveMetrics) {
				metrics.TemperatureCelsius = model.Some(int64(70))
				metrics.WarningTemperature = model.Some(int64(70))
				metrics.CriticalTemperature = model.Some(int64(80))
			},
			status:  model.StatusCaution,
			quality: model.DataQualityFull,
			reasons: []string{TemperatureWarning},
		},
		{
			name: "critical temperature is critical",
			mutate: func(metrics *model.DriveMetrics) {
				metrics.TemperatureCelsius = model.Some(int64(82))
				metrics.WarningTemperature = model.Some(int64(70))
				metrics.CriticalTemperature = model.Some(int64(80))
			},
			status:  model.StatusCritical,
			quality: model.DataQualityFull,
			reasons: []string{TemperatureCritical},
		},
		{
			name: "self-test failure is caution",
			mutate: func(metrics *model.DriveMetrics) {
				metrics.LastSelfTestStatus = model.Some("Completed: failed")
				metrics.LastSelfTestPassed = model.Some(false)
			},
			status:  model.StatusCaution,
			quality: model.DataQualityFull,
			reasons: []string{SelfTestFailed},
		},
		{
			name: "fatal self-test failure is critical",
			mutate: func(metrics *model.DriveMetrics) {
				metrics.LastSelfTestStatus = model.Some("Completed: read failure")
				metrics.LastSelfTestPassed = model.Some(false)
			},
			status:  model.StatusCritical,
			quality: model.DataQualityFull,
			reasons: []string{SelfTestFailed},
		},
		{
			name: "missing smart is unknown",
			mutate: func(metrics *model.DriveMetrics) {
				metrics.SMARTPassed = model.None[bool](model.MissingUnavailable)
			},
			status:  model.StatusUnknown,
			quality: model.DataQualityPartial,
			reasons: []string{SmartDataUnavailable, RequiredDataMissing},
		},
		{
			name: "permission missing smart is unknown with permission reason",
			mutate: func(metrics *model.DriveMetrics) {
				metrics.SMARTPassed = model.None[bool](model.MissingPermission)
			},
			status:  model.StatusUnknown,
			quality: model.DataQualityPartial,
			reasons: []string{PermissionRequired, RequiredDataMissing},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := baseMetrics()
			if tt.mutate != nil {
				tt.mutate(&metrics)
			}

			got := Assess(metrics)
			if got.OverallStatus != tt.status {
				t.Fatalf("status = %s, want %s", got.OverallStatus, tt.status)
			}
			if got.DataQuality != tt.quality {
				t.Fatalf("quality = %s, want %s", got.DataQuality, tt.quality)
			}
			if !sameReasons(got.ReasonCodes, tt.reasons) {
				t.Fatalf("reasons = %#v, want %#v", got.ReasonCodes, tt.reasons)
			}
		})
	}
}

func sameReasons(got, want []string) bool {
	if len(got) == 0 && len(want) == 0 {
		return true
	}
	return reflect.DeepEqual(got, want)
}
