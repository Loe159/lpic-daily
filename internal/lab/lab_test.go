package lab_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	lpicdaily "github.com/Loe159/lpic-daily"
	"github.com/Loe159/lpic-daily/internal/lab"
	"github.com/Loe159/lpic-daily/internal/runner"
)

const (
	shellEnvironmentRepairID = "lpic1.103.1.shell-environment-repair"
	sharedDropboxID          = "lpic1.104.5.shared-dropbox"
	stuckWorkerID            = "lpic1.103.5.stuck-worker"
)

func loadBuiltinLab(t *testing.T, id string) lab.Lab {
	t.Helper()
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	for _, authored := range labs {
		if authored.Definition.ID == id {
			return authored
		}
	}
	t.Fatalf("built-in lab %s not found", id)
	return lab.Lab{}
}

func TestLoadBuiltinShellEnvironmentRepair(t *testing.T) {
	got := loadBuiltinLab(t, shellEnvironmentRepairID)

	if len(got.Definition.ConceptIDs) != 3 {
		t.Fatalf("concepts = %d, want 3", len(got.Definition.ConceptIDs))
	}
	if len(got.Hints) != 4 {
		t.Fatalf("hints = %d, want 4", len(got.Hints))
	}
	if !strings.Contains(got.SetupScript, "/opt/lpic/shadow/bin") || !strings.Contains(got.SetupScript, "REPORT_FILE=") {
		t.Fatal("shell environment setup script is incomplete")
	}

	checks, err := got.CompileChecks()
	if err != nil {
		t.Fatalf("CompileChecks() error = %v", err)
	}
	if len(checks) != 5 {
		t.Fatalf("checks = %d, want 5", len(checks))
	}
}

func TestLoadBuiltinSharedDropbox(t *testing.T) {
	got := loadBuiltinLab(t, sharedDropboxID)

	if len(got.Hints) != 4 {
		t.Fatalf("hints = %d, want 4", len(got.Hints))
	}
	for index, hint := range got.Hints {
		if hint.Level != index+1 {
			t.Fatalf("hint %d level = %d", index, hint.Level)
		}
	}
	if !strings.Contains(got.SetupScript, "getent group project") || !strings.Contains(got.SetupScript, "/srv/shared") {
		t.Fatal("setup script was not loaded")
	}
	if got.ReferenceSolutionRef == "" {
		t.Fatal("reference solution ref should remain traceable")
	}

	checks, err := got.CompileChecks()
	if err != nil {
		t.Fatalf("CompileChecks() error = %v", err)
	}
	if len(checks) != 4 {
		t.Fatalf("checks = %d, want 4", len(checks))
	}
}

func TestLoadBuiltinStuckWorker(t *testing.T) {
	got := loadBuiltinLab(t, stuckWorkerID)

	if len(got.Definition.ConceptIDs) != 5 {
		t.Fatalf("concepts = %d, want 5", len(got.Definition.ConceptIDs))
	}
	if len(got.Hints) != 4 {
		t.Fatalf("hints = %d, want 4", len(got.Hints))
	}
	if !strings.Contains(got.SetupScript, "stuck-worker") || !strings.Contains(got.SetupScript, "lpic-signal-probe") {
		t.Fatal("stuck-worker setup script is incomplete")
	}

	checks, err := got.CompileChecks()
	if err != nil {
		t.Fatalf("CompileChecks() error = %v", err)
	}
	if len(checks) != 7 {
		t.Fatalf("checks = %d, want 7", len(checks))
	}
	if !got.Definition.NeedsPersistentShell {
		t.Fatal("stuck-worker must require the persistent PTY shell")
	}
}

func TestSessionRunsSetupAndStateChecks(t *testing.T) {
	authored := loadBuiltinLab(t, sharedDropboxID)
	fake := &fakeRunner{}

	session, err := lab.Start(context.Background(), authored, fake)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if !fake.started {
		t.Fatal("runner was not started")
	}
	if !strings.Contains(strings.Join(fake.exec.Argv, " "), "/usr/bin/bash -eu -c") {
		t.Fatalf("setup argv = %v", fake.exec.Argv)
	}
	if !strings.Contains(fake.exec.Argv[len(fake.exec.Argv)-1], "getent group project") {
		t.Fatal("setup script was not passed into sandbox exec")
	}

	results, err := session.Evaluate(context.Background())
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	for _, result := range results {
		if !result.Pass {
			t.Fatalf("check failed: %#v", result)
		}
	}

	if err := session.Close(context.Background()); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if !fake.destroyed {
		t.Fatal("runner was not destroyed")
	}
}

func TestSessionResetRestartsAndReplaysSetup(t *testing.T) {
	authored := loadBuiltinLab(t, sharedDropboxID)
	fake := &fakeRunner{}

	session, err := lab.Start(context.Background(), authored, fake)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer session.Close(context.Background())

	if fake.startCalls != 1 || fake.execCalls != 1 {
		t.Fatalf("initial lifecycle start=%d exec=%d, want 1/1", fake.startCalls, fake.execCalls)
	}
	if err := session.Reset(context.Background()); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}
	if fake.resetCalls != 1 {
		t.Fatalf("reset calls = %d, want 1", fake.resetCalls)
	}
	if fake.startCalls != 2 || fake.execCalls != 2 {
		t.Fatalf("reset lifecycle start=%d exec=%d, want 2/2", fake.startCalls, fake.execCalls)
	}
	if !strings.Contains(fake.exec.Argv[len(fake.exec.Argv)-1], "getent group project") {
		t.Fatal("reset did not replay the lab setup")
	}
}

func TestVMSetupNoneSkipsGuestExec(t *testing.T) {
	authored := loadBuiltinLab(t, sharedDropboxID)
	authored.Definition.ID = "lpic1.104.1.test-vm"
	authored.Definition.Environment.Backend = "libvirt"
	authored.Definition.Environment.CapabilityProfile = "full-machine"
	authored.Definition.Environment.Machine = &lab.Machine{
		Firmware: "uefi",
		ExtraDisks: []lab.MachineDisk{{
			ID:     "data",
			SizeMB: 512,
		}},
	}
	authored.Definition.Setup = lab.Setup{ExecutionScope: "none"}
	authored.SetupScript = ""

	fake := &fakeRunner{failExec: true}
	session, err := lab.Start(context.Background(), authored, fake)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer session.Close(context.Background())

	if fake.execCalls != 0 {
		t.Fatalf("VM setup unexpectedly called Exec %d time(s)", fake.execCalls)
	}
	if fake.definition.Machine == nil || len(fake.definition.Machine.ExtraDisks) != 1 {
		t.Fatalf("VM definition = %#v", fake.definition)
	}
}

func TestDestructiveSetupCannotModifyHostSentinelThroughLabOrchestration(t *testing.T) {
	sentinel := filepath.Join(t.TempDir(), "host-sentinel")
	if err := os.WriteFile(sentinel, []byte("safe"), 0o600); err != nil {
		t.Fatalf("write sentinel: %v", err)
	}

	authored := loadBuiltinLab(t, sharedDropboxID)
	authored.SetupScript = fmt.Sprintf("printf 'pwned' > %q", sentinel)
	fake := &fakeRunner{}

	session, err := lab.Start(context.Background(), authored, fake)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer session.Close(context.Background())

	got, err := os.ReadFile(sentinel)
	if err != nil {
		t.Fatalf("read sentinel: %v", err)
	}
	if string(got) != "safe" {
		t.Fatalf("host sentinel changed to %q", got)
	}
	if len(fake.exec.Argv) != 4 || fake.exec.Argv[0] != "/usr/bin/bash" {
		t.Fatalf("setup was not delegated as structured sandbox argv: %v", fake.exec.Argv)
	}
	if fake.exec.Argv[3] != authored.SetupScript {
		t.Fatalf("setup body = %q, want %q", fake.exec.Argv[3], authored.SetupScript)
	}
}

type fakeRunner struct {
	definition      runner.Definition
	exec            runner.ExecRequest
	execCalls       int
	startCalls      int
	resetCalls      int
	failExec        bool
	started         bool
	destroyed       bool
	destroyCalls    int
	destroyFailures int
}

func (fake *fakeRunner) Prepare(_ context.Context, definition runner.Definition) (runner.Instance, error) {
	fake.definition = definition
	return runner.Instance{ID: "fake-lab"}, nil
}

func (fake *fakeRunner) Start(_ context.Context, _ runner.Instance) error {
	fake.started = true
	fake.startCalls++
	return nil
}

func (fake *fakeRunner) Exec(_ context.Context, _ runner.Instance, request runner.ExecRequest) (runner.ExecResult, error) {
	fake.execCalls++
	fake.exec = request
	if fake.failExec {
		return runner.ExecResult{}, errors.New("Exec must not be called")
	}
	return runner.ExecResult{ExitCode: 0}, nil
}

func (fake *fakeRunner) Stat(_ context.Context, _ runner.Instance, path string) (runner.FileInfo, error) {
	if path != "/srv/shared" {
		return runner.FileInfo{}, errors.New("not found")
	}
	return runner.FileInfo{
		Path:  path,
		Mode:  0o3770,
		UID:   0,
		GID:   2000,
		User:  "root",
		Group: "project",
		IsDir: true,
	}, nil
}

func (fake *fakeRunner) ReadFile(context.Context, runner.Instance, string, int64) ([]byte, error) {
	return nil, errors.New("not implemented in fake")
}

func (fake *fakeRunner) Processes(context.Context, runner.Instance) ([]runner.Process, error) {
	return nil, nil
}

func (fake *fakeRunner) Reset(context.Context, runner.Instance) error {
	fake.resetCalls++
	return nil
}

func (fake *fakeRunner) Destroy(context.Context, runner.Instance) error {
	fake.destroyCalls++
	if fake.destroyFailures > 0 {
		fake.destroyFailures--
		return errors.New("transient destroy failure")
	}
	fake.destroyed = true
	return nil
}

func (fake *fakeRunner) Close() error {
	return nil
}

func TestStartSurfacesCleanupFailure(t *testing.T) {
	authored := loadBuiltinLab(t, sharedDropboxID)
	fake := &fakeRunner{failExec: true, destroyFailures: 1}

	_, err := lab.Start(context.Background(), authored, fake)
	if err == nil || !strings.Contains(err.Error(), "cleanup failed start") {
		t.Fatalf("Start() error = %v, want surfaced cleanup failure", err)
	}
	if fake.destroyCalls != 1 {
		t.Fatalf("destroy calls = %d, want 1", fake.destroyCalls)
	}
}

func TestFailedResetCleanupRemainsRetryable(t *testing.T) {
	authored := loadBuiltinLab(t, sharedDropboxID)
	fake := &fakeRunner{}
	session, err := lab.Start(context.Background(), authored, fake)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	fake.failExec = true
	fake.destroyFailures = 1
	err = session.Reset(context.Background())
	if err == nil || !strings.Contains(err.Error(), "cleanup failed reset") {
		t.Fatalf("Reset() error = %v, want cleanup failure", err)
	}

	fake.failExec = false
	if err := session.Close(context.Background()); err != nil {
		t.Fatalf("Close() retry error = %v", err)
	}
	if fake.destroyCalls != 2 || !fake.destroyed {
		t.Fatalf("cleanup retry destroyCalls=%d destroyed=%v, want 2/true", fake.destroyCalls, fake.destroyed)
	}
}
