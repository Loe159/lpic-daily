package desktop_test

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/Loe159/lpic-daily/internal/desktop"
)

type call struct {
	name string
	args []string
}

type fakeExecutor struct {
	runOutput []byte
	runErr    error
	startErr  error
	runs      []call
	starts    []call
}

func (executor *fakeExecutor) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	executor.runs = append(executor.runs, call{name: name, args: append([]string(nil), args...)})
	return executor.runOutput, executor.runErr
}

func (executor *fakeExecutor) Start(_ context.Context, name string, args ...string) error {
	executor.starts = append(executor.starts, call{name: name, args: append([]string(nil), args...)})
	return executor.startErr
}

func TestSendDailyReturnsOpenAction(t *testing.T) {
	executor := &fakeExecutor{runOutput: []byte("open\n")}
	open, err := desktop.SendDaily(context.Background(), executor, desktop.Notification{
		Title: "LPIC Daily",
		Body:  "2 activités à faire",
	})
	if err != nil {
		t.Fatalf("SendDaily() error = %v", err)
	}
	if !open {
		t.Fatal("open = false, want true")
	}
	if len(executor.runs) != 1 || executor.runs[0].name != "notify-send" {
		t.Fatalf("runs = %#v", executor.runs)
	}
}

func TestSendDailyFailsClosedWhenNotifierFails(t *testing.T) {
	executor := &fakeExecutor{runErr: errors.New("missing")}
	if _, err := desktop.SendDaily(context.Background(), executor, desktop.Notification{
		Title: "LPIC Daily",
		Body:  "session",
	}); err == nil {
		t.Fatal("SendDaily() unexpectedly succeeded")
	}
}

func TestLaunchDailyUsesConfiguredLauncherAndBinary(t *testing.T) {
	t.Setenv("LPIC_DAILY_TERMINAL_LAUNCHER", "kitty -e")
	t.Setenv("LPIC_DAILY_BINARY", "/home/test/.local/bin/lpic")
	executor := &fakeExecutor{}
	if err := desktop.LaunchDaily(context.Background(), executor); err != nil {
		t.Fatalf("LaunchDaily() error = %v", err)
	}
	want := call{name: "kitty", args: []string{"-e", "/home/test/.local/bin/lpic", "tui"}}
	if len(executor.starts) != 1 || !reflect.DeepEqual(executor.starts[0], want) {
		t.Fatalf("starts = %#v, want %#v", executor.starts, want)
	}
}

func TestLaunchLabUsesConfiguredLauncherAndBinary(t *testing.T) {
	t.Setenv("LPIC_DAILY_TERMINAL_LAUNCHER", "kitty -e")
	t.Setenv("LPIC_DAILY_BINARY", "/home/test/.local/bin/lpic")
	executor := &fakeExecutor{}
	if err := desktop.LaunchLab(context.Background(), executor, "lpic1.103.1.shell-environment-repair"); err != nil {
		t.Fatalf("LaunchLab() error = %v", err)
	}
	want := call{
		name: "kitty",
		args: []string{
			"-e",
			"/home/test/.local/bin/lpic",
			"lab",
			"session",
			"lpic1.103.1.shell-environment-repair",
		},
	}
	if len(executor.starts) != 1 || !reflect.DeepEqual(executor.starts[0], want) {
		t.Fatalf("starts = %#v, want %#v", executor.starts, want)
	}
}

func TestLaunchLabRejectsEmptyID(t *testing.T) {
	executor := &fakeExecutor{}
	if err := desktop.LaunchLab(context.Background(), executor, "  "); err == nil {
		t.Fatal("LaunchLab() unexpectedly accepted an empty lab ID")
	}
	if len(executor.starts) != 0 {
		t.Fatalf("starts = %#v, want none", executor.starts)
	}
}

func TestDetectTerminalLauncherFallsBackToKitty(t *testing.T) {
	t.Setenv("LPIC_DAILY_TERMINAL_LAUNCHER", "")
	dir := t.TempDir()
	kitty := dir + "/kitty"
	if err := os.WriteFile(kitty, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write kitty stub: %v", err)
	}
	t.Setenv("PATH", dir)

	launcher, err := desktop.DetectTerminalLauncher()
	if err != nil {
		t.Fatalf("DetectTerminalLauncher() error = %v", err)
	}
	if launcher.Command != "kitty" || !reflect.DeepEqual(launcher.Prefix, []string{"--"}) {
		t.Fatalf("launcher = %#v", launcher)
	}
}
