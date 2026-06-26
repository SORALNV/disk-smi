package app

import (
	"errors"
	"os/exec"
	"strings"

	"disk-smi/internal/render"
)

const (
	ExitOK                = 0
	ExitCLIError          = 2
	ExitMissingDependency = 3
	ExitPermission        = 4
	ExitNoSSD             = 5
	ExitSMARTFailure      = 6
	ExitInternal          = 7
)

type Error struct {
	Code int
	Err  error
}

func (e *Error) Error() string {
	if e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *Error) Unwrap() error {
	return e.Err
}

func ExitCode(err error) int {
	if err == nil {
		return ExitOK
	}
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	var execErr *exec.Error
	if errors.As(err, &execErr) {
		return ExitMissingDependency
	}
	if strings.Contains(strings.ToLower(err.Error()), "permission") {
		return ExitPermission
	}
	return ExitInternal
}

func coded(code int, err error) error {
	if err == nil {
		return nil
	}
	return &Error{Code: code, Err: err}
}

func UserMessage(err error, locale render.Locale) string {
	if err == nil {
		return ""
	}
	code := ExitCode(err)
	if code == ExitPermission {
		if locale == render.LocaleJapanese {
			return "SMART情報の取得には管理者権限が必要です。\n\n次を実行してください:\n  sudo disk-smi"
		}
		return "SMART data requires elevated permission.\n\nRun:\n  sudo disk-smi"
	}
	if code == ExitMissingDependency {
		if strings.Contains(strings.ToLower(err.Error()), "smartctl") {
			if locale == render.LocaleJapanese {
				return "実SSDの詳細SMART情報を読むにはsmartctlが必要です。\n\n自前backendで起動:\n  disk-smi --backend native\n\n詳細SMARTを有効にする場合:\n  brew install smartmontools"
			}
			return "smartctl is required for detailed real SSD SMART data.\n\nRun with the built-in backend:\n  disk-smi --backend native\n\nFor detailed SMART data:\n  brew install smartmontools"
		}
		if locale == render.LocaleJapanese {
			return "必要な外部コマンドが見つかりません: " + err.Error()
		}
		return "Required external command is missing: " + err.Error()
	}
	return err.Error()
}
