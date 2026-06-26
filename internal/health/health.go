package health

import (
	"strings"

	"disk-smi/internal/model"
)

const (
	SMARTFailed                  = "SMART_FAILED"
	CriticalWarningActive        = "CRITICAL_WARNING_ACTIVE"
	CriticalWarningReadOnly      = "CRITICAL_WARNING_READ_ONLY"
	AvailableSpareBelowThreshold = "AVAILABLE_SPARE_BELOW_THRESHOLD"
	EnduranceLow                 = "ENDURANCE_LOW"
	EnduranceRatedLimitReached   = "ENDURANCE_RATED_LIMIT_REACHED"
	MediaErrorsPresent           = "MEDIA_ERRORS_PRESENT"
	TemperatureWarning           = "TEMPERATURE_WARNING"
	TemperatureCritical          = "TEMPERATURE_CRITICAL"
	SelfTestFailed               = "SELF_TEST_FAILED"
	RequiredDataMissing          = "REQUIRED_DATA_MISSING"
	SmartDataUnavailable         = "SMART_DATA_UNAVAILABLE"
	PermissionRequired           = "PERMISSION_REQUIRED"
)

func Assess(metrics model.DriveMetrics) model.HealthAssessment {
	reasons := make([]string, 0)
	critical := false
	caution := false
	missingRequired := false

	if !metrics.SMARTPassed.Valid {
		missingRequired = true
		if metrics.SMARTPassed.Reason == model.MissingPermission {
			reasons = append(reasons, PermissionRequired)
		} else {
			reasons = append(reasons, SmartDataUnavailable)
		}
	} else if !metrics.SMARTPassed.Value {
		critical = true
		reasons = append(reasons, SMARTFailed)
	}

	if metrics.CriticalWarning.Valid && metrics.CriticalWarning.Value != 0 {
		critical = true
		reasons = append(reasons, CriticalWarningActive)
		if metrics.CriticalWarning.Value&(1<<3) != 0 {
			reasons = append(reasons, CriticalWarningReadOnly)
		}
	}

	if metrics.AvailableSparePercent.Valid && metrics.SpareThresholdPercent.Valid {
		if metrics.AvailableSparePercent.Value < metrics.SpareThresholdPercent.Value {
			critical = true
			reasons = append(reasons, AvailableSpareBelowThreshold)
		}
	} else {
		missingRequired = true
	}

	if metrics.EnduranceUsedPercent.Valid {
		if metrics.EnduranceUsedPercent.Value >= 100 {
			caution = true
			reasons = append(reasons, EnduranceRatedLimitReached)
		} else if metrics.EnduranceUsedPercent.Value >= 90 {
			caution = true
			reasons = append(reasons, EnduranceLow)
		}
	} else {
		missingRequired = true
	}

	if metrics.MediaErrors.Valid {
		if metrics.MediaErrors.Value.CmpInt64(0) > 0 {
			caution = true
			reasons = append(reasons, MediaErrorsPresent)
		}
	} else {
		missingRequired = true
	}

	if metrics.TemperatureCelsius.Valid {
		if metrics.CriticalTemperature.Valid && metrics.TemperatureCelsius.Value >= metrics.CriticalTemperature.Value {
			critical = true
			reasons = append(reasons, TemperatureCritical)
		} else if metrics.WarningTemperature.Valid && metrics.TemperatureCelsius.Value >= metrics.WarningTemperature.Value {
			caution = true
			reasons = append(reasons, TemperatureWarning)
		}
	}

	if metrics.LastSelfTestPassed.Valid && !metrics.LastSelfTestPassed.Value {
		if fatalSelfTest(metrics.LastSelfTestStatus) {
			critical = true
		} else {
			caution = true
		}
		reasons = append(reasons, SelfTestFailed)
	}

	if missingRequired {
		caution = true
		reasons = append(reasons, RequiredDataMissing)
	}

	status := model.StatusGood
	quality := model.DataQualityFull
	switch {
	case critical:
		status = model.StatusCritical
	case missingRequired && !hasCriticalEvidence(reasons):
		status = model.StatusUnknown
		quality = model.DataQualityPartial
	case caution:
		status = model.StatusCaution
		if missingRequired {
			quality = model.DataQualityPartial
		}
	}

	if missingRequired && quality == model.DataQualityFull {
		quality = model.DataQualityPartial
	}

	return model.HealthAssessment{
		OverallStatus: status,
		DataQuality:   quality,
		ReasonCodes:   reasons,
	}
}

func fatalSelfTest(status model.Optional[string]) bool {
	if !status.Valid {
		return false
	}
	value := strings.ToLower(status.Value)
	for _, token := range []string{"fatal", "read failure", "electrical failure", "servo failure"} {
		if strings.Contains(value, token) {
			return true
		}
	}
	return false
}

func hasCriticalEvidence(reasons []string) bool {
	for _, reason := range reasons {
		switch reason {
		case SMARTFailed, CriticalWarningActive, CriticalWarningReadOnly, AvailableSpareBelowThreshold, TemperatureCritical:
			return true
		}
	}
	return false
}
