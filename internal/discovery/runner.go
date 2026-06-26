package discovery

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"

	"disk-smi/internal/device"
)

const defaultDiskutilTimeout = 10 * time.Second

type Runner struct {
	Path    string
	Timeout time.Duration
}

type CommandResult struct {
	Command  []string
	Stdout   []byte
	Stderr   string
	ExitCode int
	TimedOut bool
}

type DiscoverResult struct {
	ListResult  CommandResult
	InfoResults []CommandResult
	Candidates  []Candidate
}

type InspectResult struct {
	InfoResult CommandResult
	Candidate  Candidate
	OK         bool
}

func (r Runner) Discover(ctx context.Context) (DiscoverResult, error) {
	path := r.Path
	if path == "" {
		path = "/usr/sbin/diskutil"
	}

	listResult, err := r.run(ctx, path, []string{"list", "-plist"})
	result := DiscoverResult{ListResult: listResult}
	if err != nil {
		return result, err
	}
	names, err := ParseList(listResult.Stdout)
	if err != nil {
		return result, err
	}

	for _, name := range names {
		disk, err := device.Parse(name)
		if err != nil {
			return result, err
		}
		infoResult, err := r.run(ctx, path, []string{"info", "-plist", disk.DevicePath})
		result.InfoResults = append(result.InfoResults, infoResult)
		if err != nil {
			return result, err
		}
		candidate, ok, err := ParseInfo(infoResult.Stdout)
		if err != nil {
			return result, err
		}
		if ok {
			result.Candidates = append(result.Candidates, candidate)
		}
	}
	return result, nil
}

func (r Runner) Inspect(ctx context.Context, target string) (InspectResult, error) {
	path := r.Path
	if path == "" {
		path = "/usr/sbin/diskutil"
	}
	disk, err := device.Parse(target)
	if err != nil {
		return InspectResult{}, err
	}
	infoResult, err := r.run(ctx, path, []string{"info", "-plist", disk.DevicePath})
	result := InspectResult{InfoResult: infoResult}
	if err != nil {
		return result, err
	}
	candidate, ok, err := ParseInfo(infoResult.Stdout)
	if err != nil {
		return result, err
	}
	result.Candidate = candidate
	result.OK = ok
	return result, nil
}

func (r Runner) run(ctx context.Context, path string, args []string) (CommandResult, error) {
	timeout := r.Timeout
	if timeout == 0 {
		timeout = defaultDiskutilTimeout
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, path, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	result := CommandResult{
		Command:  append([]string{path}, args...),
		Stdout:   stdout.Bytes(),
		Stderr:   stderr.String(),
		ExitCode: exitCode(err),
		TimedOut: errors.Is(runCtx.Err(), context.DeadlineExceeded),
	}
	if result.TimedOut {
		return result, fmt.Errorf("diskutil timed out after %s", timeout)
	}
	if err != nil {
		return result, fmt.Errorf("diskutil failed: %w", err)
	}
	return result, nil
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
