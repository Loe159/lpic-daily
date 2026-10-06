package desktop

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

type TerminalLauncher struct {
	Command string
	Prefix  []string
	Source  string
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

func DetectTerminalLauncher() (TerminalLauncher, error) {
	if override := strings.TrimSpace(os.Getenv("LPIC_DAILY_TERMINAL_LAUNCHER")); override != "" {
		parts := strings.Fields(override)
		if len(parts) == 0 {
			return TerminalLauncher{}, errors.New("LPIC_DAILY_TERMINAL_LAUNCHER is empty")
		}
		return TerminalLauncher{
			Command: parts[0],
			Prefix:  append([]string(nil), parts[1:]...),
			Source:  "LPIC_DAILY_TERMINAL_LAUNCHER",
		}, nil
	}

	candidates := []TerminalLauncher{
		{Command: "xdg-terminal-exec", Prefix: []string{"--"}, Source: "auto"},
		{Command: "kitty", Prefix: []string{"--"}, Source: "auto"},
		{Command: "foot", Prefix: []string{"--"}, Source: "auto"},
		{Command: "wezterm", Prefix: []string{"start", "--"}, Source: "auto"},
		{Command: "gnome-terminal", Prefix: []string{"--"}, Source: "auto"},
		{Command: "konsole", Prefix: []string{"-e"}, Source: "auto"},
		{Command: "alacritty", Prefix: []string{"-e"}, Source: "auto"},
		{Command: "xterm", Prefix: []string{"-e"}, Source: "auto"},
	}
	for _, candidate := range candidates {
		if _, err := exec.LookPath(candidate.Command); err == nil {
			return candidate, nil
		}
	}
	return TerminalLauncher{}, errors.New(
		"no supported terminal launcher found (tried xdg-terminal-exec, kitty, foot, wezterm, gnome-terminal, konsole, alacritty and xterm)",
	)
}

func resolveBinary() (string, error) {
	binary := strings.TrimSpace(os.Getenv("LPIC_DAILY_BINARY"))
	if binary == "" {
		resolved, err := os.Executable()
		if err != nil {
			return "", fmt.Errorf("resolve LPIC Daily executable: %w", err)
		}
		binary = resolved
	}
	if !filepath.IsAbs(binary) && strings.ContainsRune(binary, filepath.Separator) {
		resolved, err := filepath.Abs(binary)
		if err != nil {
			return "", fmt.Errorf("resolve LPIC Daily executable path: %w", err)
		}
		binary = resolved
	}
	return binary, nil
}

func launchTerminal(ctx context.Context, executor Executor, commandArgs ...string) error {
	if executor == nil {
		return errors.New("executor is required")
	}
	launcher, err := DetectTerminalLauncher()
	if err != nil {
		return err
	}
	binary, err := resolveBinary()
	if err != nil {
		return err
	}
	args := append(append([]string(nil), launcher.Prefix...), binary)
	args = append(args, commandArgs...)
	if err := executor.Start(ctx, launcher.Command, args...); err != nil {
		return fmt.Errorf("launch %s terminal: %w", launcher.Command, err)
	}
	return nil
}

func LaunchDaily(ctx context.Context, executor Executor) error {
	return launchTerminal(ctx, executor, "tui")
}

func LaunchLab(ctx context.Context, executor Executor, labID string) error {
	labID = strings.TrimSpace(labID)
	if labID == "" {
		return errors.New("lab ID is required")
	}
	return launchTerminal(ctx, executor, "lab", "run", labID)
}
