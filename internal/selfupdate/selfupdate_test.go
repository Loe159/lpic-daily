package selfupdate

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeRunner struct {
	goPath string
	err    error
	args   []string
	env    []string
}

func (runner *fakeRunner) LookPath(file string) (string, error) {
	if file != "go" || runner.goPath == "" {
		return "", errors.New("not found")
	}
	return runner.goPath, nil
}

func (runner *fakeRunner) Run(
	_ context.Context,
	name string,
	args []string,
	env []string,
	_, _ io.Writer,
) error {
	runner.args = append([]string(nil), args...)
	runner.env = append([]string(nil), env...)
	if runner.err != nil {
		return runner.err
	}
	if name != runner.goPath {
		return errors.New("unexpected executable")
	}
	var goBin string
	for _, item := range env {
		if strings.HasPrefix(item, "GOBIN=") {
			goBin = strings.TrimPrefix(item, "GOBIN=")
		}
	}
	if goBin == "" {
		return errors.New("missing GOBIN")
	}
	if err := os.MkdirAll(goBin, 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(goBin, "lpic"), []byte("new-binary"), 0o755)
}

func TestUpdateBuildsMainAndAtomicallyInstallsUserBinary(t *testing.T) {
	home := t.TempDir()
	targetDir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(targetDir, 0o700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(targetDir, "lpic")
	if err := os.WriteFile(target, []byte("old-binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	runner := &fakeRunner{goPath: "/usr/bin/go"}
	var stdout bytes.Buffer
	if err := Update(context.Background(), Options{
		Runner:  runner,
		Stdout:  &stdout,
		Stderr:  &bytes.Buffer{},
		HomeDir: home,
	}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new-binary" {
		t.Fatalf("installed binary = %q", got)
	}
	if len(runner.args) != 2 || runner.args[0] != "install" || runner.args[1] != mainPackage {
		t.Fatalf("go args = %#v", runner.args)
	}
	for _, want := range []string{
		"GONOPROXY=github.com/Loe159/lpic-daily",
		"GONOSUMDB=github.com/Loe159/lpic-daily",
	} {
		found := false
		for _, item := range runner.env {
			if item == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("update env missing %q: %#v", want, runner.env)
		}
	}
	if !strings.Contains(stdout.String(), "mis à jour depuis main") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestUpdateFailsClearlyWhenGoIsUnavailable(t *testing.T) {
	err := Update(context.Background(), Options{
		Runner:  &fakeRunner{},
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
		HomeDir: t.TempDir(),
	})
	if err == nil || !strings.Contains(err.Error(), "Go est requis") {
		t.Fatalf("error = %v", err)
	}
}

func TestUpdateDoesNotTouchUserState(t *testing.T) {
	home := t.TempDir()
	state := filepath.Join(home, ".local", "share", "lpic-daily", "progress.db")
	if err := os.MkdirAll(filepath.Dir(state), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(state, []byte("progress"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := Update(context.Background(), Options{
		Runner:  &fakeRunner{goPath: "/usr/bin/go"},
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
		HomeDir: home,
	}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, err := os.ReadFile(state)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "progress" {
		t.Fatalf("state changed = %q", got)
	}
}
