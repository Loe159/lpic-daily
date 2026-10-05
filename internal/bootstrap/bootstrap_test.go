package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

type fakeRunner struct {
	paths       map[string]bool
	imageExists bool
	runs        []string
	interactive []string
}

func (runner *fakeRunner) LookPath(name string) (string, error) {
	if runner.paths[name] {
		return "/usr/bin/" + name, nil
	}
	return "", errors.New("missing")
}

func (runner *fakeRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	call := strings.Join(append([]string{name}, args...), " ")
	runner.runs = append(runner.runs, call)
	if name == "podman" && len(args) >= 2 && args[0] == "image" && args[1] == "exists" {
		if !runner.imageExists {
			return nil, errors.New("image missing")
		}
	}
	return nil, nil
}

func (runner *fakeRunner) Interactive(
	_ context.Context,
	_ io.Reader,
	_ io.Writer,
	_ io.Writer,
	name string,
	args ...string,
) error {
	call := strings.Join(append([]string{name}, args...), " ")
	runner.interactive = append(runner.interactive, call)
	if name == "podman" && len(args) > 0 && args[0] == "build" {
		runner.imageExists = true
	}
	return nil
}

func bootstrapAssets() fs.FS {
	return fstest.MapFS{
		notificationService: &fstest.MapFile{Data: []byte("[Service]\nExecStart=/usr/bin/env lpic notify\n")},
		notificationTimer:   &fstest.MapFile{Data: []byte("[Timer]\nOnCalendar=*-*-* 09:00:00\n")},
		desktopEntry:        &fstest.MapFile{Data: []byte("[Desktop Entry]\nExec=lpic tui\n")},
		"labs/images/fedora-phase1/Containerfile":          &fstest.MapFile{Data: []byte("FROM scratch\n")},
		"labs/images/fedora-phase1/report-status-approved": &fstest.MapFile{Data: []byte("#!/bin/sh\n")},
		"labs/images/fedora-phase1/report-status-shadow":   &fstest.MapFile{Data: []byte("#!/bin/sh\n")},
	}
}

func TestInstallConfiguresUserIntegrationAndBuildsMissingPodmanImage(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "data"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, "state"))
	t.Setenv("LPIC_DAILY_SKIP_SELF_INSTALL", "1")

	runner := &fakeRunner{
		paths: map[string]bool{
			"notify-send": true,
			"podman":      true,
			"systemctl":   true,
		},
	}
	var stdout, stderr bytes.Buffer
	if err := Install(context.Background(), bootstrapAssets(), Options{
		Runner: runner,
		Stdin:  strings.NewReader(""),
		Stdout: &stdout,
		Stderr: &stderr,
	}); err != nil {
		t.Fatalf("Install() error = %v; stderr=%q", err, stderr.String())
	}

	for _, path := range []string{
		filepath.Join(home, "config", "systemd", "user", "lpic-daily-notify.service"),
		filepath.Join(home, "config", "systemd", "user", "lpic-daily-notify.timer"),
		filepath.Join(home, "data", "applications", "lpic-daily.desktop"),
	} {
		if _, err := filepath.Glob(path); err != nil {
			t.Fatalf("glob %s: %v", path, err)
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected installed file %s: %v", path, err)
		}
	}
	if !runner.imageExists {
		t.Fatal("Podman image was not built")
	}
	if len(runner.interactive) != 2 ||
		!strings.Contains(runner.interactive[0], "podman pull") ||
		!strings.Contains(runner.interactive[1], "podman build --pull=never") {
		t.Fatalf("interactive calls = %#v", runner.interactive)
	}
}

func TestEnsureFirstRunUsesMarker(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "data"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, "state"))
	t.Setenv("LPIC_DAILY_SKIP_SELF_INSTALL", "1")

	runner := &fakeRunner{
		paths: map[string]bool{
			"notify-send": true,
			"podman":      true,
			"systemctl":   true,
		},
		imageExists: true,
	}
	opts := Options{
		Runner: runner,
		Stdin:  strings.NewReader(""),
		Stdout: io.Discard,
		Stderr: io.Discard,
	}
	if err := EnsureFirstRun(context.Background(), bootstrapAssets(), opts); err != nil {
		t.Fatalf("first EnsureFirstRun() error = %v", err)
	}
	runCount := len(runner.runs)
	if runCount == 0 {
		t.Fatal("first setup performed no commands")
	}
	if err := EnsureFirstRun(context.Background(), bootstrapAssets(), opts); err != nil {
		t.Fatalf("second EnsureFirstRun() error = %v", err)
	}
	if len(runner.runs) != runCount {
		t.Fatalf("second first-run setup repeated commands: before=%d after=%d", runCount, len(runner.runs))
	}
}
