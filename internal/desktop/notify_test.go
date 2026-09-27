package desktop_test

import (
	"context"
	"errors"
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

func TestLaunchDailyUsesExplicitArgumentVector(t *testing.T) {
	t.Setenv("LPIC_DAILY_TERMINAL_LAUNCHER", "")
	executor := &fakeExecutor{}
	if err := desktop.LaunchDaily(context.Background(), executor); err != nil {
		t.Fatalf("LaunchDaily() error = %v", err)
	}
	want := call{name: "xdg-terminal-exec", args: []string{"--", "lpic", "tui"}}
	if len(executor.starts) != 1 || !reflect.DeepEqual(executor.starts[0], want) {
		t.Fatalf("starts = %#v, want %#v", executor.starts, want)
	}
}
