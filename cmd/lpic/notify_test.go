package main

import (
	"bytes"
	"context"
	"testing"
	"time"
)

type notificationCall struct {
	name string
	args []string
}

type notificationExecutor struct {
	output []byte
	runs   []notificationCall
	starts []notificationCall
}

func (executor *notificationExecutor) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	executor.runs = append(executor.runs, notificationCall{name: name, args: append([]string(nil), args...)})
	return executor.output, nil
}

func (executor *notificationExecutor) Start(_ context.Context, name string, args ...string) error {
	executor.starts = append(executor.starts, notificationCall{name: name, args: append([]string(nil), args...)})
	return nil
}

func TestNotifySendsOnlyOncePerLocalDay(t *testing.T) {
	t.Setenv("LPIC_DAILY_STATE_DIR", t.TempDir())
	now := time.Date(2026, 9, 27, 9, 0, 0, 0, time.Local)
	executor := &notificationExecutor{}

	var first bytes.Buffer
	if err := runNotifyWithExecutor(context.Background(), false, &first, executor, now); err != nil {
		t.Fatalf("first notify error = %v", err)
	}
	if len(executor.runs) != 1 {
		t.Fatalf("notify runs = %d, want 1", len(executor.runs))
	}

	var second bytes.Buffer
	if err := runNotifyWithExecutor(context.Background(), false, &second, executor, now.Add(time.Hour)); err != nil {
		t.Fatalf("second notify error = %v", err)
	}
	if len(executor.runs) != 1 {
		t.Fatalf("notify repeated on same day: %d runs", len(executor.runs))
	}
	if !bytes.Contains(second.Bytes(), []byte("déjà envoyée")) {
		t.Fatalf("second output = %q", second.String())
	}
}

func TestNotifyActionLaunchesTUI(t *testing.T) {
	t.Setenv("LPIC_DAILY_STATE_DIR", t.TempDir())
	t.Setenv("LPIC_DAILY_TERMINAL_LAUNCHER", "")
	executor := &notificationExecutor{output: []byte("open\n")}
	now := time.Date(2026, 9, 27, 9, 0, 0, 0, time.Local)

	if err := runNotifyWithExecutor(context.Background(), true, &bytes.Buffer{}, executor, now); err != nil {
		t.Fatalf("notify error = %v", err)
	}
	if len(executor.starts) != 1 || executor.starts[0].name != "xdg-terminal-exec" {
		t.Fatalf("starts = %#v", executor.starts)
	}
}
