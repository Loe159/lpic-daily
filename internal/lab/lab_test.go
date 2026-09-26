package lab_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	lpicdaily "github.com/Loe159/lpic-daily"
	"github.com/Loe159/lpic-daily/internal/lab"
	"github.com/Loe159/lpic-daily/internal/runner"
)

func TestLoadBuiltinSharedDropbox(t *testing.T) {
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	if len(labs) != 1 {
		t.Fatalf("labs = %d, want 1", len(labs))
	}

	got := labs[0]
	if got.Definition.ID != "lpic1.104.5.shared-dropbox" {
		t.Fatalf("lab id = %q", got.Definition.ID)
	}
	if len(got.Hints) != 4 {
		t.Fatalf("hints = %d, want 4", len(got.Hints))
	}
	for index, hint := range got.Hints {
		if hint.Level != index+1 {
			t.Fatalf("hint %d level = %d", index, hint.Level)
		}
	}
	if !strings.Contains(got.SetupScript, "groupadd") {
		t.Fatal("setup script was not loaded")
	}
	if got.ReferenceSolutionRef == "" {
		t.Fatal("reference solution ref should remain traceable")
	}

	checks, err := got.CompileChecks()
	if err != nil {
		t.Fatalf("CompileChecks() error = %v", err)
	}
	if len(checks) != 3 {
		t.Fatalf("checks = %d, want 3", len(checks))
	}
}

func TestSessionRunsSetupAndStateChecks(t *testing.T) {
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	fake := &fakeRunner{}

	session, err := lab.Start(context.Background(), labs[0], fake)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if !fake.started {
		t.Fatal("runner was not started")
	}
	if !strings.Contains(strings.Join(fake.exec.Argv, " "), "/usr/bin/bash -eu -c") {
		t.Fatalf("setup argv = %v", fake.exec.Argv)
	}
	if !strings.Contains(fake.exec.Argv[len(fake.exec.Argv)-1], "useradd") {
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

type fakeRunner struct {
	definition runner.Definition
	exec       runner.ExecRequest
	started    bool
	destroyed  bool
}

func (fake *fakeRunner) Prepare(_ context.Context, definition runner.Definition) (runner.Instance, error) {
	fake.definition = definition
	return runner.Instance{ID: "fake-lab"}, nil
}

func (fake *fakeRunner) Start(_ context.Context, _ runner.Instance) error {
	fake.started = true
	return nil
}

func (fake *fakeRunner) Exec(_ context.Context, _ runner.Instance, request runner.ExecRequest) (runner.ExecResult, error) {
	fake.exec = request
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
	return nil
}

func (fake *fakeRunner) Destroy(context.Context, runner.Instance) error {
	fake.destroyed = true
	return nil
}

func (fake *fakeRunner) Close() error {
	return nil
}
