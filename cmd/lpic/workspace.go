package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Loe159/lpic-daily/internal/lab"
	"github.com/Loe159/lpic-daily/internal/terminal"
	lpicui "github.com/Loe159/lpic-daily/internal/tui"
)

type labWorkspaceKey struct{}

type workspaceCheck struct {
	Description string `json:"description"`
	Passed      bool   `json:"passed"`
	Detail      string `json:"detail,omitempty"`
}

type workspaceSnapshot struct {
	LabID      string           `json:"lab_id"`
	Status     string           `json:"status"`
	HintLevel  int              `json:"hint_level"`
	Checks     []workspaceCheck `json:"checks,omitempty"`
	CheckText  string           `json:"check_text,omitempty"`
	Checked    bool             `json:"checked"`
	Passed     bool             `json:"passed"`
	UpdatedAt  time.Time        `json:"updated_at"`
}

type workspacePublisher struct {
	mu    sync.Mutex
	path  string
	state workspaceSnapshot
}

func (p *workspacePublisher) update(change func(*workspaceSnapshot)) error {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	change(&p.state)
	p.state.UpdatedAt = time.Now()
	return writeWorkspaceSnapshot(p.path, p.state)
}

func writeWorkspaceSnapshot(path string, state workspaceSnapshot) error {
	payload, err := json.Marshal(state)
	if err != nil {
		return err
	}
	if len(payload) > 1<<20 {
		return errors.New("lab workspace snapshot exceeds 1 MiB")
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".workspace-*")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(payload); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), path)
}

func readWorkspaceSnapshot(path string) (workspaceSnapshot, error) {
	file, err := os.Open(path)
	if err != nil {
		return workspaceSnapshot{}, err
	}
	defer file.Close()
	payload, err := io.ReadAll(io.LimitReader(file, (1<<20)+1))
	if err != nil {
		return workspaceSnapshot{}, err
	}
	if len(payload) > 1<<20 {
		return workspaceSnapshot{}, errors.New("oversized workspace snapshot")
	}
	var state workspaceSnapshot
	if err := json.Unmarshal(payload, &state); err != nil {
		return state, err
	}
	return state, nil
}

func workspaceRoot() (string, error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	root := filepath.Join(cache, "lpic-daily", "workspaces")
	if err := os.MkdirAll(root, 0o700); err != nil {
		return "", err
	}
	return root, nil
}

func workspaceName(labID string) string {
	digest := sha256.Sum256([]byte(labID))
	return "lpic-" + hex.EncodeToString(digest[:6])
}

func validateWorkspacePath(labID, path string) error {
	root, err := workspaceRoot()
	if err != nil {
		return err
	}
	expected := filepath.Join(root, workspaceName(labID)+".json")
	if !filepath.IsAbs(path) || filepath.Clean(path) != expected {
		return errors.New("workspace state path is outside LPIC Daily's private workspace directory")
	}
	return nil
}

func tmuxExec(ctx context.Context, socket string, stdin io.Reader, stdout, stderr io.Writer, args ...string) error {
	command := exec.CommandContext(ctx, "tmux", append([]string{"-S", socket}, args...)...)
	command.Stdin = stdin
	command.Stdout = stdout
	command.Stderr = stderr
	return command.Run()
}

func tmuxRun(ctx context.Context, socket string, args ...string) error {
	return tmuxExec(ctx, socket, nil, io.Discard, io.Discard, args...)
}

// The local tmux server only arranges panels. All learner commands still run
// through Podman's/libvirt's existing isolated Runner and PTY implementation.
func runTmuxLabWorkspace(ctx context.Context, authored lab.Lab, stdin io.Reader, stdout, stderr io.Writer) (bool, error) {
	input, inputOK := stdin.(*os.File)
	output, outputOK := stdout.(*os.File)
	if !inputOK || !outputOK || !terminal.IsTerminal(input) || !terminal.IsTerminal(output) {
		return false, nil
	}
	if os.Getenv("TMUX") != "" {
		// Do not nest a tmux client implicitly: existing tmux users retain the
		// historical lab mode until the outer-session integration is supported.
		return false, nil
	}
	if _, err := exec.LookPath("tmux"); err != nil {
		return false, nil
	}
	width, height, err := terminal.Size(output)
	if err != nil || width < 100 || height < 20 {
		return false, nil
	}
	root, err := workspaceRoot()
	if err != nil {
		return true, err
	}
	socket := filepath.Join(root, "tmux.sock")
	sessionName := workspaceName(authored.Definition.ID)
	statePath := filepath.Join(root, sessionName+".json")
	binary, err := os.Executable()
	if err != nil {
		return true, err
	}
	if override := strings.TrimSpace(os.Getenv("LPIC_DAILY_BINARY")); override != "" {
		binary, err = exec.LookPath(override)
		if err != nil {
			return true, fmt.Errorf("resolve LPIC_DAILY_BINARY: %w", err)
		}
	}
	// A running detached session must be reattached without restarting its
	// sandbox or replacing its snapshot.
	if err := tmuxRun(ctx, socket, "has-session", "-t", "="+sessionName); err != nil {
		_ = os.Remove(statePath)
		// tmux -S isolates these options from the user's personal sessions.
		if err := tmuxRun(ctx, socket, "new-session", "-d", "-s", sessionName, "-n", "lab",
			binary, "lab", "panel", authored.Definition.ID, statePath); err != nil {
			return true, fmt.Errorf("create LPIC tmux session: %w", err)
		}
		cleanupOnError := true
		defer func() {
			if cleanupOnError {
				_ = tmuxRun(context.Background(), socket, "kill-session", "-t", "="+sessionName)
			}
		}()
		// C-g belongs to LPIC's *host* tmux; Ctrl-B remains available to the
		// tmux commands learners practise inside the sandbox.
		if err := tmuxRun(ctx, socket, "set-option", "-g", "prefix", "C-g"); err != nil {
			return true, fmt.Errorf("configure LPIC tmux prefix: %w", err)
		}
		if err := tmuxRun(ctx, socket, "set-option", "-g", "status-left", " LPIC Daily "); err != nil {
			return true, err
		}
		if err := tmuxRun(ctx, socket, "set-option", "-g", "status-right",
			" C-g z : zoom | C-g d : détacher "); err != nil {
			return true, err
		}
		if err := tmuxRun(ctx, socket, "split-window", "-h", "-p", "60", "-t", sessionName+":0",
			binary, "lab", "run", authored.Definition.ID, "--workspace-state", statePath); err != nil {
			return true, fmt.Errorf("start sandbox pane: %w", err)
		}
		if err := tmuxRun(ctx, socket, "select-pane", "-t", sessionName+":0.1"); err != nil {
			return true, err
		}
		cleanupOnError = false
	}
	// The attach process owns the same TTY already used by the TUI. tmux
	// intentionally survives disconnection; exiting attach does not reset.
	if err := tmuxExec(ctx, socket, stdin, stdout, stderr, "attach-session", "-t", "="+sessionName); err != nil {
		return true, fmt.Errorf("attach LPIC tmux session: %w", err)
	}
	return true, nil
}

// renderWorkspacePanel is side-effect free and independently testable. All
// guest-controlled text is sanitised before rendering in this host TUI.
func renderWorkspacePanel(authored lab.Lab, snapshot workspaceSnapshot, width, height, topScroll, checkScroll int) string {
	if width < 24 {
		width = 24
	}
	if height < 12 {
		height = 12
	}
	inner := width - 3
	topHeight := height * 60 / 100
	if topHeight < 7 {
		topHeight = 7
	}
	bottomHeight := height - topHeight - 2
	top := []string{"MISSION  " + strings.Join(authored.Definition.ObjectiveIDs, ", ") + "  ·  " + authored.Definition.Environment.Backend, "", authored.Definition.TitleFR, ""}
	top = append(top, workspaceWrap(authored.Definition.BriefFR, inner)...)
	top = append(top, "", "OBJECTIFS")
	for i, criterion := range authored.Definition.SuccessCriteriaFR {
		top = append(top, workspaceWrap(fmt.Sprintf("%d. %s", i+1, criterion), inner)...)
	}
	top = append(top, "", fmt.Sprintf("INDICES  %d/%d", snapshot.HintLevel, len(authored.Hints)))
	if snapshot.HintLevel == 0 {
		top = append(top, "Aucun indice révélé. :hint dans le shell.")
	}
	for _, hint := range authored.Hints {
		if hint.Level <= snapshot.HintLevel {
			top = append(top, fmt.Sprintf("Indice %d", hint.Level))
			top = append(top, workspaceWrap(hint.ContentFR, inner)...)
		}
	}
	checks := []string{"VÉRIFICATION"}
	if !snapshot.Checked {
		checks = append(checks, "", "Pas encore de vérification.", "Saisis :check dans le shell.")
	} else {
		total, passed := len(snapshot.Checks), 0
		for _, check := range snapshot.Checks {
			if check.Passed {
				passed++
			}
		}
		checks = append(checks, fmt.Sprintf("%d/%d critères OK", passed, total), "")
		for _, check := range snapshot.Checks {
			mark := "✗ "
			if check.Passed {
				mark = "✓ "
			}
			checks = append(checks, workspaceWrap(mark+check.Description, inner)...)
			if !check.Passed && check.Detail != "" {
				checks = append(checks, workspaceWrap("    "+check.Detail, inner)...)
			}
		}
		if snapshot.Passed {
			checks = append(checks, "", "LAB RÉUSSI · progression enregistrée")
		}
		if snapshot.CheckText != "" && len(snapshot.Checks) == 0 {
			checks = append(checks, workspaceWrap(snapshot.CheckText, inner)...)
		}
	}
	if snapshot.Status == "starting" {
		checks = append(checks, "Initialisation de la sandbox…")
	}
	if snapshot.Status == "ended" {
		checks = append(checks, "", "Session terminée. q pour fermer.")
	}
	var b strings.Builder
	b.WriteString("\x1b[?25l\x1b[H\x1b[2J")
	renderWorkspaceSection(&b, top, width, topHeight, topScroll)
	b.WriteString(strings.Repeat("─", width-1) + "\r\n")
	renderWorkspaceSection(&b, checks, width, bottomHeight, checkScroll)
	b.WriteString("\x1b[7m" + workspaceFit(" j/k : consigne · J/K : checks · q : fermer le panneau ", width-1) + "\x1b[0m")
	return b.String()
}

func workspaceFit(text string, width int) string {
	text = lpicui.SanitizeText(strings.ReplaceAll(text, "\r", ""))
	runes := []rune(strings.TrimSpace(text))
	if len(runes) > width {
		if width > 1 {
			return string(runes[:width-1]) + "…"
		}
		return string(runes[:width])
	}
	return string(runes) + strings.Repeat(" ", width-len(runes))
}

func workspaceWrap(input string, width int) []string {
	if width < 12 {
		width = 12
	}
	var result []string
	for _, paragraph := range strings.Split(lpicui.SanitizeText(input), "\n") {
		if strings.TrimSpace(paragraph) == "" {
			result = append(result, "")
			continue
		}
		line := ""
		for _, word := range strings.Fields(paragraph) {
			if line != "" && len([]rune(line))+1+len([]rune(word)) > width {
				result = append(result, line)
				line = ""
			}
			if line != "" {
				line += " "
			}
			line += word
		}
		result = append(result, line)
	}
	return result
}

func renderWorkspaceSection(out *strings.Builder, lines []string, width, height, scroll int) {
	if scroll < 0 {
		scroll = 0
	}
	if scroll > len(lines)-height && len(lines) > height {
		scroll = len(lines) - height
	}
	for i := 0; i < height; i++ {
		value := ""
		if at := scroll + i; at >= 0 && at < len(lines) {
			value = lines[at]
		}
		out.WriteString(workspaceFit(value, width-1))
		out.WriteString("\r\n")
	}
}

func runWorkspacePanel(authored lab.Lab, statePath string, stdin io.Reader, stdout io.Writer) error {
	in, okIn := stdin.(*os.File)
	out, okOut := stdout.(*os.File)
	if !okIn || !okOut || !terminal.IsTerminal(in) || !terminal.IsTerminal(out) {
		return errors.New("lab panel requires a terminal")
	}
	state, err := terminal.MakeRaw(in)
	if err != nil {
		return err
	}
	defer terminal.Restore(in, state)
	defer fmt.Fprint(out, "\x1b[?25h\x1b[0m")
	keys := make(chan byte, 32)
	go func() {
		for {
			var key [1]byte
			if _, err := in.Read(key[:]); err != nil {
				return
			}
			select {
			case keys <- key[0]:
			default:
			}
		}
	}()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	topScroll, checkScroll := 0, 0
	lastFrame := ""
	for {
		select {
		case <-ctx.Done():
			return nil
		case key := <-keys:
			switch key {
			case 'q':
				return nil
			case 'j':
				topScroll++
			case 'k':
				if topScroll > 0 {
					topScroll--
				}
			case 'J':
				checkScroll++
			case 'K':
				if checkScroll > 0 {
					checkScroll--
				}
			}
		case <-ticker.C:
		}
		snapshot, err := readWorkspaceSnapshot(statePath)
		if err != nil {
			snapshot = workspaceSnapshot{LabID: authored.Definition.ID, Status: "starting"}
		}
		if snapshot.LabID != "" && snapshot.LabID != authored.Definition.ID {
			return errors.New("lab workspace state ID mismatch")
		}
		width, height, err := terminal.Size(out)
		if err != nil {
			return err
		}
		frame := renderWorkspacePanel(authored, snapshot, width, height, topScroll, checkScroll)
		if frame != lastFrame {
			if _, err := io.WriteString(out, frame); err != nil {
				return err
			}
			lastFrame = frame
		}
	}
}

// Used in tests and for manually inspecting the layout without a sandbox.
func workspaceSampleState(authored lab.Lab) workspaceSnapshot {
	return workspaceSnapshot{LabID: authored.Definition.ID, Status: "ready"}
}

func workspaceStatusText(state workspaceSnapshot) string {
	return state.Status + ":" + strconv.Itoa(state.HintLevel)
}
