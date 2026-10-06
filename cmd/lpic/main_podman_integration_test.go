//go:build integration

package main

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	lpicdaily "github.com/Loe159/lpic-daily"
	"github.com/Loe159/lpic-daily/internal/lab"
	"github.com/Loe159/lpic-daily/internal/runner"
	podmanrunner "github.com/Loe159/lpic-daily/internal/runner/podman"
)

func TestPhase1ImageProvidesInteractiveEditorsAndTerminfo(t *testing.T) {
	if os.Getenv("LPIC_DAILY_RUN_PODMAN_INTEGRATION") != "1" {
		t.Skip("set LPIC_DAILY_RUN_PODMAN_INTEGRATION=1 to run real rootless Podman integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	backend, err := podmanrunner.Open(ctx, "")
	if err != nil {
		t.Fatalf("Open(rootless Podman) error = %v", err)
	}
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}

	var authored lab.Lab
	for _, candidate := range labs {
		if candidate.Definition.ID == shellEnvironmentRepairID {
			authored = candidate
			break
		}
	}
	if authored.Definition.ID == "" {
		t.Fatal("shell-environment-repair lab not found")
	}

	session, err := lab.Start(ctx, authored, backend)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		if err := session.Close(cleanupCtx); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	}()

	result, err := backend.Exec(ctx, session.Instance, runner.ExecRequest{
		Argv: []string{
			"/usr/bin/bash",
			"-c",
			"command -v nano >/dev/null && command -v vi >/dev/null && command -v vim >/dev/null && test -r /usr/share/terminfo/x/xterm-256color && test -r /usr/share/terminfo/x/xterm-kitty",
		},
	})
	if err != nil {
		t.Fatalf("editor/terminfo probe error = %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("editor/terminfo probe exit = %d, want 0", result.ExitCode)
	}
}

func TestPrepareJobControlShellOnRootlessPodman(t *testing.T) {
	if os.Getenv("LPIC_DAILY_RUN_PODMAN_INTEGRATION") != "1" {
		t.Skip("set LPIC_DAILY_RUN_PODMAN_INTEGRATION=1 to run real rootless Podman integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	backend, err := podmanrunner.Open(ctx, "")
	if err != nil {
		t.Fatalf("Open(rootless Podman) error = %v", err)
	}
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}

	var authored lab.Lab
	for _, candidate := range labs {
		if candidate.Definition.ID == stuckWorkerID {
			authored = candidate
			break
		}
	}
	if authored.Definition.ID == "" {
		t.Fatal("stuck-worker lab not found")
	}

	session, err := lab.Start(ctx, authored, backend)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		if err := session.Close(cleanupCtx); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	}()

	if err := prepareJobControlShell(ctx, backend, session.Instance); err != nil {
		t.Fatalf("prepareJobControlShell() error = %v", err)
	}

	rc, err := backend.ReadFile(ctx, session.Instance, jobControlShellRCPath, 16<<10)
	if err != nil {
		t.Fatalf("ReadFile(rc) error = %v", err)
	}
	if !bytes.Contains(rc, []byte("__lpic_daily_job_control_log")) {
		t.Fatalf("rc file missing instrumentation: %q", rc)
	}

	events, err := backend.ReadFile(ctx, session.Instance, jobControlEventPath, 4096)
	if err != nil {
		t.Fatalf("ReadFile(events) error = %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("initial job-control event log = %q, want empty", events)
	}
}
