package desktop

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Executor interface {
	Run(context.Context, string, ...string) ([]byte, error)
	Start(context.Context, string, ...string) error
}

type OSExecutor struct{}

func (OSExecutor) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, name, args...)
	return command.Output()
}

func (OSExecutor) Start(ctx context.Context, name string, args ...string) error {
	command := exec.CommandContext(ctx, name, args...)
	command.Stdin = nil
	command.Stdout = nil
	command.Stderr = nil
	if err := command.Start(); err != nil {
		return err
	}
	if err := command.Process.Release(); err != nil {
		return fmt.Errorf("release detached process: %w", err)
	}
	return nil
}

type Notification struct {
	Title string
	Body  string
}

func SendDaily(ctx context.Context, executor Executor, notification Notification) (bool, error) {
	if executor == nil {
		return false, errors.New("executor is required")
	}
	if strings.TrimSpace(notification.Title) == "" || strings.TrimSpace(notification.Body) == "" {
		return false, errors.New("notification title and body are required")
	}

	output, err := executor.Run(
		ctx,
		"notify-send",
		"--app-name=LPIC Daily",
		"--urgency=normal",
		"--action=open=Ouvrir",
		"--",
		notification.Title,
		notification.Body,
	)
	if err != nil {
		return false, fmt.Errorf("notify-send: %w", err)
	}
	return strings.TrimSpace(string(bytes.TrimSpace(output))) == "open", nil
}

func LaunchDaily(ctx context.Context, executor Executor) error {
	if executor == nil {
		return errors.New("executor is required")
	}

	if override := strings.TrimSpace(os.Getenv("LPIC_DAILY_TERMINAL_LAUNCHER")); override != "" {
		parts := strings.Fields(override)
		if len(parts) == 0 {
			return errors.New("LPIC_DAILY_TERMINAL_LAUNCHER is empty")
		}
		args := append(parts[1:], "lpic", "tui")
		if err := executor.Start(ctx, parts[0], args...); err != nil {
			return fmt.Errorf("launch configured terminal: %w", err)
		}
		return nil
	}

	if err := executor.Start(ctx, "xdg-terminal-exec", "--", "lpic", "tui"); err != nil {
		return fmt.Errorf("launch xdg-terminal-exec: %w", err)
	}
	return nil
}
