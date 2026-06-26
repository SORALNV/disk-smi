package smartctl

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"disk-smi/internal/device"
	"disk-smi/internal/model"
)

const defaultTimeout = 15 * time.Second

type Runner struct {
	Path    string
	Detail  bool
	Timeout time.Duration
}

type CommandResult struct {
	Command  []string
	Stdout   []byte
	Stderr   string
	ExitCode int
	TimedOut bool
}

func (r Runner) Run(ctx context.Context, target string) (CommandResult, error) {
	disk, err := device.Parse(target)
	if err != nil {
		return CommandResult{}, err
	}

	path := r.Path
	if path == "" {
		path = "smartctl"
	}
	flag := "-a"
	if r.Detail {
		flag = "-x"
	}
	timeout := r.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}

	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	args := []string{flag, "-j", disk.DevicePath}
	cmd := exec.CommandContext(runCtx, path, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	result := CommandResult{
		Command:  append([]string{path}, args...),
		Stdout:   stdout.Bytes(),
		Stderr:   stderr.String(),
		ExitCode: exitCode(err),
		TimedOut: errors.Is(runCtx.Err(), context.DeadlineExceeded),
	}
	if result.TimedOut {
		return result, fmt.Errorf("smartctl timed out after %s", timeout)
	}
	if err != nil && len(result.Stdout) == 0 {
		if isPermissionError(result.Stderr) {
			return result, fmt.Errorf("smartctl permission denied: %s", strings.TrimSpace(result.Stderr))
		}
		return result, fmt.Errorf("smartctl failed: %w", err)
	}
	return result, nil
}

func (r Runner) Snapshot(ctx context.Context, target string) (ResultSnapshot, error) {
	result, err := r.Run(ctx, target)
	if err != nil {
		return ResultSnapshot{Result: result}, err
	}
	snapshot, parseErr := Parse(result.Stdout, target)
	return ResultSnapshot{Result: result, Snapshot: snapshot}, parseErr
}

type ResultSnapshot struct {
	Result   CommandResult
	Snapshot model.DriveSnapshot
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}

func isPermissionError(stderr string) bool {
	text := strings.ToLower(stderr)
	return strings.Contains(text, "permission") || strings.Contains(text, "must be root") || strings.Contains(text, "operation not permitted")
}
