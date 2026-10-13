package checker_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Loe159/lpic-daily/internal/checker"
	"github.com/Loe159/lpic-daily/internal/runner"
)

type fakeProbe struct {
	files       map[string]runner.FileInfo
	contents    map[string][]byte
	processes   []runner.Process
	execResult  runner.ExecResult
	execCalls   int
	execErr     error
	lastRequest runner.ExecRequest
}

func (probe *fakeProbe) Exec(_ context.Context, _ runner.Instance, request runner.ExecRequest) (runner.ExecResult, error) {
	probe.execCalls++
	probe.lastRequest = request
	return probe.execResult, probe.execErr
}

func (probe *fakeProbe) Stat(_ context.Context, _ runner.Instance, path string) (runner.FileInfo, error) {
	info, exists := probe.files[path]
	if !exists {
		return runner.FileInfo{}, os.ErrNotExist
	}
	return info, nil
}

func (probe *fakeProbe) ReadFile(_ context.Context, _ runner.Instance, path string, max int64) ([]byte, error) {
	content, exists := probe.contents[path]
	if !exists {
		return nil, os.ErrNotExist
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

func TestStateChecksAcceptDifferentCommandHistoriesWithSameFinalState(t *testing.T) {
	histories := [][]string{
		{
			"chown root:project /srv/shared",
			"chmod 3770 /srv/shared",
		},
		{
			"chgrp project /srv/shared",
			"chmod u=rwx,g=rwx,o= /srv/shared",
			"chmod g+s,+t /srv/shared",
		},
	}

	checks := []checker.Check{
		checker.FileMode{CheckID: "mode", Path: "/srv/shared", Mode: 0o3770},
		checker.FileOwner{CheckID: "owner", Path: "/srv/shared", User: "root", Group: "project"},
	}

	for index, history := range histories {
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

		results, err := checker.EvaluateAll(
			context.Background(),
			probe,
			runner.Instance{ID: "fake"},
			checks,
		)
		if err != nil {
			t.Fatalf("history %d (%v): EvaluateAll() error = %v", index+1, history, err)
		}
		for _, result := range results {
			if !result.Pass {
				t.Fatalf("history %d (%v): check failed: %#v", index+1, history, result)
			}
		}
	}
}

type hostACPIProbe struct {
	*fakeProbe
	sent bool
	err  error
}

func (probe *hostACPIProbe) ACPIButtonObserved(_ context.Context, _ runner.Instance) (bool, error) {
	return probe.sent, probe.err
}

func TestACPIButtonRequiresIndependentHostEvidence(t *testing.T) {
	ctx := context.Background()
	check := checker.ACPIButtonSent{CheckID: "host-acpi"}
	instance := runner.Instance{ID: "owned-vm"}
	if got := check.Evaluate(ctx, &fakeProbe{}, instance); got.Pass || !errors.Is(got.Err, runner.ErrNotSupported) {
		t.Fatalf("guest-only probe incorrectly passed: %#v", got)
	}
	probe := &hostACPIProbe{fakeProbe: &fakeProbe{}}
	if got := check.Evaluate(ctx, probe, instance); got.Pass || got.Err != nil {
		t.Fatalf("no host event incorrectly passed: %#v", got)
	}
	probe.sent = true
	if got := check.Evaluate(ctx, probe, instance); !got.Pass || got.Err != nil {
		t.Fatalf("host event not accepted: %#v", got)
	}
	probe.err = errors.New("host unavailable")
	if got := check.Evaluate(ctx, probe, instance); got.Pass || !errors.Is(got.Err, probe.err) {
		t.Fatalf("host failure ignored: %#v", got)
	}
}

type fakeUEFIBaselineProbe struct {
	*fakeProbe
	expected string
	current  string
	err      error
}

func (probe *fakeUEFIBaselineProbe) CaptureUEFIBootOrder(context.Context, runner.Instance) error {
	return nil
}

func (probe *fakeUEFIBaselineProbe) ExpectedUEFIBootOrder(context.Context, runner.Instance) (string, error) {
	return probe.expected, probe.err
}

func (probe *fakeUEFIBaselineProbe) Exec(_ context.Context, _ runner.Instance, request runner.ExecRequest) (runner.ExecResult, error) {
	if request.Stdout != nil {
		_, _ = request.Stdout.Write([]byte(probe.current + "\n"))
	}
	return runner.ExecResult{ExitCode: 0}, nil
}

func TestUEFIBootOrderRequiresFullTrustedHostBaseline(t *testing.T) {
	ctx := context.Background()
	instance := runner.Instance{ID: "uefi-vm"}
	check := checker.UEFIBootOrderPreserved{CheckID: "firmware-order"}
	if got := check.Evaluate(ctx, &fakeProbe{}, instance); got.Pass || !errors.Is(got.Err, runner.ErrNotSupported) {
		t.Fatalf("guest-only UEFI evidence accepted: %#v", got)
	}
	probe := &fakeUEFIBaselineProbe{
		fakeProbe: &fakeProbe{},
		expected:  "0005,0004,0000,0001",
		current:   "0005,0004,0001,0000",
	}
	if got := check.Evaluate(ctx, probe, instance); got.Pass {
		t.Fatalf("reordered unrelated firmware entries passed: %#v", got)
	}
	probe.current = probe.expected
	if got := check.Evaluate(ctx, probe, instance); !got.Pass || got.Err != nil {
		t.Fatalf("correct complete boot order failed: %#v", got)
	}
	probe.err = errors.New("trusted baseline unavailable")
	if got := check.Evaluate(ctx, probe, instance); got.Pass || !errors.Is(got.Err, probe.err) {
		t.Fatalf("missing baseline treated as success: %#v", got)
	}
}

type blockedCommandProbe struct{ fakeProbe }

func (probe *blockedCommandProbe) Exec(ctx context.Context, _ runner.Instance, _ runner.ExecRequest) (runner.ExecResult, error) {
	<-ctx.Done()
	return runner.ExecResult{}, ctx.Err()
}

func TestCommandExitTimeoutReportsFailedCheckAndContinues(t *testing.T) {
	probe := &blockedCommandProbe{}
	results, err := checker.EvaluateAll(context.Background(), probe, runner.Instance{ID: "fake"}, []checker.Check{
		checker.CommandExit{CheckID: "blocked-shell", Argv: []string{"/usr/bin/bash", "-ic", "true"}, ExpectedExit: 0, TimeoutSeconds: 1},
		checker.ProcessState{CheckID: "still-checked", Match: "not-running", Present: false},
	})
	if err != nil {
		t.Fatalf("EvaluateAll() error = %v", err)
	}
	if len(results) != 2 || results[0].Pass || results[0].Err != nil ||
		!strings.Contains(results[0].Detail, "délai dépassé") {
		t.Fatalf("timeout result = %#v", results)
	}
	if !results[1].Pass || results[1].CheckID != "still-checked" {
		t.Fatalf("following check skipped: %#v", results)
	}
}

func TestCommandExitGuestTimeoutHasActionableDetail(t *testing.T) {
	probe := &fakeProbe{execResult: runner.ExecResult{ExitCode: 124}}
	check := checker.CommandExit{CheckID: "blocked-shell", Argv: []string{"/usr/bin/timeout", "6s", "/usr/bin/bash"}, ExpectedExit: 0}
	result := check.Evaluate(context.Background(), probe, runner.Instance{ID: "fake"})
	if result.Pass || result.Err != nil || !strings.Contains(result.Detail, ".bashrc") {
		t.Fatalf("timeout detail = %#v", result)
	}
}

func TestCommandBatchExecutesOnceAndPreservesIndividualResults(t *testing.T) {
	makeChecks := func() []checker.Check {
		checks := make([]checker.Check, 5)
		for i := range checks {
			checks[i] = checker.CommandExit{
				CheckID:        fmt.Sprintf("check-%02d", i+1),
				Argv:           []string{"/usr/bin/timeout", "3s", "/usr/bin/bash", "-c", "exit 0"},
				BatchID:        "group",
				FailureBit:     1 << i,
				TimeoutSeconds: 5,
			}
		}
		return checks
	}
	for _, tc := range []struct {
		exit   int
		failed []int
	}{
		{exit: 160},
		{exit: 165, failed: []int{0, 2}},
		{exit: 176, failed: []int{4}},
		{exit: 0, failed: []int{0, 1, 2, 3, 4}},
		{exit: 124, failed: []int{0, 1, 2, 3, 4}},
		{exit: 125, failed: []int{0, 1, 2, 3, 4}},
	} {
		t.Run(fmt.Sprint(tc.exit), func(t *testing.T) {
			probe := &fakeProbe{execResult: runner.ExecResult{ExitCode: tc.exit}}
			results, err := checker.EvaluateAll(context.Background(), probe, runner.Instance{ID: "fake"}, makeChecks())
			if err != nil {
				t.Fatalf("EvaluateAll error: %v", err)
			}
			if probe.execCalls != 1 || len(results) != 5 {
				t.Fatalf("calls=%d, results=%#v", probe.execCalls, results)
			}
			for i, result := range results {
				wantFail := slices.Contains(tc.failed, i)
				if result.Pass == wantFail {
					t.Errorf("check %d: got pass=%v, want fail=%v", i+1, result.Pass, wantFail)
				}
			}
		})
	}
}

func TestCommandBatchRejectsMismatchedCommandsWithoutRunning(t *testing.T) {
	probe := &fakeProbe{}
	group := []checker.Check{
		checker.CommandExit{CheckID: "a", Argv: []string{"true"}, BatchID: "group", FailureBit: 1, TimeoutSeconds: 5},
		checker.CommandExit{CheckID: "b", Argv: []string{"false"}, BatchID: "group", FailureBit: 2, TimeoutSeconds: 5},
	}
	_, err := checker.EvaluateAll(context.Background(), probe, runner.Instance{ID: "fake"}, group)
	if err == nil || probe.execCalls != 0 {
		t.Fatalf("invalid batch ran: calls=%d err=%v", probe.execCalls, err)
	}
}

// A check with no authored timeout must not run for the whole lab duration.
type deadlineObservingProbe struct {
	fakeProbe
	deadline time.Time
}

func (probe *deadlineObservingProbe) Exec(ctx context.Context, _ runner.Instance, _ runner.ExecRequest) (runner.ExecResult, error) {
	probe.deadline, _ = ctx.Deadline()
	return runner.ExecResult{ExitCode: 0}, nil
}
func TestCommandExitWithoutExplicitTimeoutHasBoundedDeadline(t *testing.T) {
	probe := &deadlineObservingProbe{}
	check := checker.CommandExit{CheckID: "default-deadline", Argv: []string{"/usr/bin/true"}}
	before := time.Now()
	result := check.Evaluate(context.Background(), probe, runner.Instance{ID: "fake"})
	if !result.Pass || probe.deadline.IsZero() {
		t.Fatalf("missing default deadline: %#v, deadline=%s", result, probe.deadline)
	}
	if remaining := probe.deadline.Sub(before); remaining < 14*time.Second || remaining > 16*time.Second {
		t.Fatalf("default timeout = %s, expected 15 seconds", remaining)
	}
}

func TestMissingLabOutputsAreFailedCriteriaNotFatalErrors(t *testing.T) {
	probe := &fakeProbe{files: map[string]runner.FileInfo{}, contents: map[string][]byte{}}
	checks := []checker.Check{
		checker.FileExists{CheckID: "exists", Path: "/run/lpic/missing"},
		checker.FileContentRegex{CheckID: "content", Path: "/run/lpic/missing", Pattern: "^ready$"},
		checker.FileMode{CheckID: "mode", Path: "/run/lpic/missing", Mode: 0o644},
		checker.FileOwner{CheckID: "owner", Path: "/run/lpic/missing", User: "root", Group: "root"},
	}
	results, err := checker.EvaluateAll(context.Background(), probe, runner.Instance{ID: "fake"}, checks)
	if err != nil {
		t.Fatalf("missing outputs must not abort check: %v", err)
	}
	for _, result := range results {
		if result.Pass || result.Err != nil || result.Detail == "" {
			t.Fatalf("missing output not reported as unmet criterion: %+v", result)
		}
	}
}

// A learner-controlled login profile may call 'exit 0' before the verifier
// is reached. A zero exit status without the randomized final receipt fails.
func TestBashCheckRequiresActualVerifierCompletion(t *testing.T) {
	probe := &fakeProbe{execResult: runner.ExecResult{ExitCode: 0}}
	check := checker.CommandExit{CheckID: "login", Argv: []string{"/usr/bin/bash", "-lc", "test -f /workspace/READY"}}
	got := check.Evaluate(context.Background(), probe, runner.Instance{ID:"fake"})
	if got.Pass || !strings.Contains(got.Detail, "avant") {
		t.Fatalf("zero exit without verifier receipt unexpectedly passed: %+v", got)
	}
	if len(probe.lastRequest.Argv) < 3 || !strings.Contains(probe.lastRequest.Argv[2], "LPIC_CHECK_COMPLETED_") {
		t.Fatalf("check script has no randomized completion receipt: %v", probe.lastRequest.Argv)
	}
}
