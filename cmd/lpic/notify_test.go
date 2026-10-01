package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type notificationCall struct {
	name string
	args []string
}

type notificationExecutor struct {
	output []byte
	runErr error
	runs   []notificationCall
	starts []notificationCall
}

func (executor *notificationExecutor) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	executor.runs = append(executor.runs, notificationCall{name: name, args: append([]string(nil), args...)})
	return executor.output, executor.runErr
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

func TestNotifyDoesNotCountPracticeAsNewConcept(t *testing.T) {
	t.Setenv("LPIC_DAILY_STATE_DIR", t.TempDir())
	if err := runWithIO(
		[]string{"learn", "lpic1.103.1.lesson.shell-sequences"},
		strings.NewReader("o\n"),
		&bytes.Buffer{},
		&bytes.Buffer{},
	); err != nil {
		t.Fatalf("learn error = %v", err)
	}

	executor := &notificationExecutor{}
	now := time.Date(2026, 9, 28, 9, 0, 0, 0, time.Local)
	if err := runNotifyWithExecutor(context.Background(), true, &bytes.Buffer{}, executor, now); err != nil {
		t.Fatalf("notify error = %v", err)
	}
	if len(executor.runs) != 1 {
		t.Fatalf("notify runs = %d, want 1", len(executor.runs))
	}
	args := strings.Join(executor.runs[0].args, " ")
	if !strings.Contains(args, "1 consolidation(s)") || !strings.Contains(args, "0 nouveau(x) concept(s)") {
		t.Fatalf("notification args = %q, practice was counted incorrectly", args)
	}
}

func TestNotifyFailureReleasesDailyClaim(t *testing.T) {
	t.Setenv("LPIC_DAILY_STATE_DIR", t.TempDir())
	now := time.Date(2026, 9, 28, 9, 0, 0, 0, time.Local)
	executor := &notificationExecutor{runErr: errors.New("notification service unavailable")}

	if err := runNotifyWithExecutor(context.Background(), false, &bytes.Buffer{}, executor, now); err == nil {
		t.Fatal("failed notify unexpectedly succeeded")
	}
	if len(executor.runs) != 1 {
		t.Fatalf("notify runs = %d, want 1", len(executor.runs))
	}

	executor.runErr = nil
	if err := runNotifyWithExecutor(context.Background(), false, &bytes.Buffer{}, executor, now.Add(time.Minute)); err != nil {
		t.Fatalf("retry notify error = %v", err)
	}
	if len(executor.runs) != 2 {
		t.Fatalf("notify retry runs = %d, want 2", len(executor.runs))
	}
}

func TestOrphanedNotificationClaimIsRecoveredAfterLeaseExpiry(t *testing.T) {
	t.Setenv("LPIC_DAILY_STATE_DIR", t.TempDir())
	ctx := context.Background()
	now := time.Date(2026, 9, 28, 9, 0, 0, 0, time.Local)

	store, err := openProgressStore(ctx)
	if err != nil {
		t.Fatalf("open progress store: %v", err)
	}
	claimed, err := store.ClaimNotification(ctx, now.Format("2006-01-02"), now)
	if err != nil || !claimed {
		_ = store.Close()
		t.Fatalf("preclaim = %v, %v", claimed, err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close progress store: %v", err)
	}

	executor := &notificationExecutor{}
	var stdout bytes.Buffer
	if err := runNotifyWithExecutor(ctx, false, &stdout, executor, now.Add(2*time.Hour)); err != nil {
		t.Fatalf("notify with orphaned claim error = %v", err)
	}
	if len(executor.runs) != 1 {
		t.Fatalf("stale orphaned claim did not recover notification delivery: %d runs", len(executor.runs))
	}
	if !strings.Contains(stdout.String(), "Notification quotidienne envoyée.") {
		t.Fatalf("unexpected recovered-claim output: %q", stdout.String())
	}
}
