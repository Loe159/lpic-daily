package checker_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Loe159/lpic-daily/internal/checker"
	"github.com/Loe159/lpic-daily/internal/runner"
)

type fakeProbe struct {
	files       map[string]runner.FileInfo
	contents    map[string][]byte
	processes   []runner.Process
	execResult  runner.ExecResult
	execErr     error
	lastRequest runner.ExecRequest
}

func (probe *fakeProbe) Exec(_ context.Context, _ runner.Instance, request runner.ExecRequest) (runner.ExecResult, error) {
	probe.lastRequest = request
	return probe.execResult, probe.execErr
}

func (probe *fakeProbe) Stat(_ context.Context, _ runner.Instance, path string) (runner.FileInfo, error) {
	info, exists := probe.files[path]
	if !exists {
		return runner.FileInfo{}, errors.New("not found")
	}
	return info, nil
}

func (probe *fakeProbe) ReadFile(_ context.Context, _ runner.Instance, path string, max int64) ([]byte, error) {
	content, exists := probe.contents[path]
	if !exists {
		return nil, errors.New("not found")
	}
	if int64(len(content)) > max {
		return nil, errors.New("content exceeds limit")
	}
	return content, nil
}

func (probe *fakeProbe) Processes(_ context.Context, _ runner.Instance) ([]runner.Process, error) {
	return probe.processes, nil
}

func TestPermissionChecksValidateStateNotCommandHistory(t *testing.T) {
	probe := &fakeProbe{
		files: map[string]runner.FileInfo{
			"/srv/shared": {
				Path:  "/srv/shared",
				Mode:  0o3770,
				UID:   0,
				GID:   2000,
				User:  "root",
				Group: "project",
				IsDir: true,
			},
		},
	}
	instance := runner.Instance{ID: "fake"}

	checks := []checker.Check{
		checker.FileMode{CheckID: "mode", Path: "/srv/shared", Mode: 0o3770},
		checker.FileOwner{CheckID: "owner", Path: "/srv/shared", User: "root", Group: "project"},
	}
	results, err := checker.EvaluateAll(context.Background(), probe, instance, checks)
	if err != nil {
		t.Fatalf("EvaluateAll() error = %v", err)
	}
	for _, result := range results {
		if !result.Pass {
			t.Fatalf("check failed: %#v", result)
		}
	}
}

func TestOwnerCheckRejectsWrongNamedGroupEvenWithDiagnosticIDs(t *testing.T) {
	probe := &fakeProbe{
		files: map[string]runner.FileInfo{
			"/srv/shared": {
				Path:  "/srv/shared",
				UID:   0,
				GID:   2001,
				User:  "root",
				Group: "wrong-group",
			},
		},
	}
	check := checker.FileOwner{CheckID: "owner", Path: "/srv/shared", User: "root", Group: "project"}
	if result := check.Evaluate(context.Background(), probe, runner.Instance{ID: "fake"}); result.Pass {
		t.Fatalf("wrong named group unexpectedly passed: %#v", result)
	}
}

func TestProcessCheckerObservesFinalState(t *testing.T) {
	probe := &fakeProbe{
		processes: []runner.Process{
			{PID: 42, Command: "worker", Args: []string{"worker", "--safe"}},
		},
	}
	instance := runner.Instance{ID: "fake"}

	present := checker.ProcessState{CheckID: "worker-present", Match: "worker", Present: true}
	if result := present.Evaluate(context.Background(), probe, instance); !result.Pass {
		t.Fatalf("present check = %#v", result)
	}

	absent := checker.ProcessState{CheckID: "bad-worker-absent", Match: "--broken", Present: false}
	if result := absent.Evaluate(context.Background(), probe, instance); !result.Pass {
		t.Fatalf("absent check = %#v", result)
	}
}

func TestCommandExitChecksSandboxBehaviorWithStructuredArgv(t *testing.T) {
	probe := &fakeProbe{execResult: runner.ExecResult{ExitCode: 0}}
	check := checker.CommandExit{
		CheckID:      "login-environment",
		Argv:         []string{"/usr/bin/bash", "-lc", "command -v report-status >/dev/null"},
		ExpectedExit: 0,
		Env:          map[string]string{"CHECK_MODE": "1"},
	}

	result := check.Evaluate(context.Background(), probe, runner.Instance{ID: "fake"})
	if !result.Pass || result.Err != nil {
		t.Fatalf("command check = %#v", result)
	}
	if !slices.Equal(probe.lastRequest.Argv, check.Argv) {
		t.Fatalf("argv = %v, want %v", probe.lastRequest.Argv, check.Argv)
	}
	if probe.lastRequest.TTY || probe.lastRequest.Stdin != nil {
		t.Fatalf("command checker requested interactive execution: %#v", probe.lastRequest)
	}

	check.Env["CHECK_MODE"] = "mutated"
	if probe.lastRequest.Env["CHECK_MODE"] != "1" {
		t.Fatal("checker request environment aliased mutable content")
	}
}

func TestCommandExitReportsWrongExitWithoutExecutionError(t *testing.T) {
	probe := &fakeProbe{execResult: runner.ExecResult{ExitCode: 7}}
	check := checker.CommandExit{
		CheckID:      "expected-zero",
		Argv:         []string{"/usr/bin/false"},
		ExpectedExit: 0,
	}
	result := check.Evaluate(context.Background(), probe, runner.Instance{ID: "fake"})
	if result.Pass || result.Err != nil {
		t.Fatalf("command check = %#v, want ordinary failed check", result)
	}
}
