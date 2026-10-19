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
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Loe159/lpic-daily/internal/lab"
	"github.com/Loe159/lpic-daily/internal/terminal"
	lpicui "github.com/Loe159/lpic-daily/internal/tui"
)

const tmuxLabSocket = "lpic-daily-ui"

type labPanelPathKey struct{}

type labPanelState struct {
	HintLevel int    `json:"hint_level"`
	Check     string `json:"check"`
	Finished  bool   `json:"finished"`
}

type labPanelReporter struct {
	mu    sync.Mutex
	path  string
	state labPanelState
}

func (reporter *labPanelReporter) persist() error {
	// The report never enters the sandbox; write a private, atomic snapshot.
	payload, err := json.Marshal(reporter.state)
	if err != nil {
		return err
	}
	if len(payload) > 64<<10 {
		return errors.New("lab panel report exceeds size limit")
	}
	temp, err := os.CreateTemp(filepath.Dir(reporter.path), ".panel-*")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if _, err := temp.Write(payload); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), reporter.path)
}

func (reporter *labPanelReporter) update(level int, check string) error {
	if reporter == nil {
		return nil
	}
	reporter.mu.Lock()
	defer reporter.mu.Unlock()
	if level > reporter.state.HintLevel {
		reporter.state.HintLevel = level
	}
	if check != "" {
		reporter.state.Check = lpicui.SanitizeText(check)
		if len(reporter.state.Check) > 24<<10 {
			reporter.state.Check = reporter.state.Check[:24<<10]
		}
	}
	return reporter.persist()
}

func (reporter *labPanelReporter) finish() {
	if reporter == nil {
		return
	}
	reporter.mu.Lock()
	defer reporter.mu.Unlock()
	reporter.state.Finished = true
	_ = reporter.persist()
}

// finishTmuxLabPanel also runs when the backend cannot start. The left pane
// must never stay stuck on "Aucun check exécuté" after a fatal launch error.
func finishTmuxLabPanel(path string, runErr error) error {
	state := labPanelState{}
	if payload, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(payload, &state)
	}
	if runErr != nil {
		state.Check = "ERREUR · " + lpicui.SanitizeText(runErr.Error())
	}
	state.Finished = true
	reporter := &labPanelReporter{path: path, state: state}
	return reporter.persist()
}

func tmuxLabName(id string) string {
	hash := sha256.Sum256([]byte(id))
	return "lpic-" + hex.EncodeToString(hash[:8])
}

func tmuxRun(ctx context.Context, args ...string) error {
	base := []string{"-L", tmuxLabSocket, "-f", "/dev/null"}
	cmd := exec.CommandContext(ctx, "tmux", append(base, args...)...)
	if result, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("tmux %s: %w (%s)", args[0], err, strings.TrimSpace(string(result)))
	}
	return nil
}

func tmuxAttach(ctx context.Context, session string, stdin io.Reader, stdout, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, "tmux", "-L", tmuxLabSocket, "-f", "/dev/null", "attach-session", "-t", session)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = stdin, stdout, stderr
	return cmd.Run()
}

// tryTmuxDashboardLab only wraps the host terminal. The right-hand pane still
// starts the original, isolated Podman/KVM lab and keeps its real PTY.
func tryTmuxDashboardLab(ctx context.Context, authored lab.Lab, stdin io.Reader, stdout, stderr io.Writer) (bool, error) {
	if os.Getenv("LPIC_DAILY_LAB_LAYOUT") == "legacy" || os.Getenv("TMUX") != "" {
		return false, nil
	}
	in, inOK := stdin.(*os.File)
	out, outOK := stdout.(*os.File)
	if !inOK || !outOK || !terminal.IsTerminal(in) || !terminal.IsTerminal(out) {
		return false, nil
	}
	if _, err := exec.LookPath("tmux"); err != nil {
		return false, nil
	}
	width, height, err := terminal.Size(out)
	if err != nil || width < 112 || height < 28 {
		return false, nil
	}
	binary, err := os.Executable()
	if err != nil {
		return false, nil
	}
	session := tmuxLabName(authored.Definition.ID)
	if err := tmuxRun(ctx, "has-session", "-t", "="+session); err == nil {
		return true, tmuxAttach(ctx, session, stdin, stdout, stderr)
	}

	dir, err := os.MkdirTemp("", "lpic-daily-tmux-")
	if err != nil {
		return false, nil
	}
	// A detached tmux session needs this file until it is reattached and closed.
	stateFile := filepath.Join(dir, "state.json")
	defer func() {
		if tmuxRun(context.Background(), "has-session", "-t", "="+session) != nil {
			_ = os.RemoveAll(dir)
		}
	}()
	panelArgs := []string{"new-session", "-d", "-s", session, "-n", "lab", "-x", strconv.Itoa(width), "-y", strconv.Itoa(height), "--",
		binary, "lab", "panel", authored.Definition.ID, stateFile}
	if err := tmuxRun(ctx, panelArgs...); err != nil {
		fmt.Fprintf(stderr, "Interface tmux indisponible, retour au mode classique : %v\n", err)
		return false, nil
	}
	terminate := func() {
		_ = tmuxRun(context.Background(), "kill-session", "-t", "="+session)
	}
	// A dedicated host-side tmux server leaves Ctrl+B to the guest's own
	// exercises. Ctrl+G is the interface prefix for resizing/detaching.
	for _, opt := range [][]string{
		{"set-option", "-t", session, "prefix", "C-g"},
		{"set-option", "-t", session, "status", "on"},
		{"set-option", "-t", session, "status-left", " LPIC Daily · Lab "},
		{"set-option", "-t", session, "status-right", " Ctrl+G d · détacher | Ctrl+G z · zoom "},
	} {
		if err := tmuxRun(ctx, opt...); err != nil {
			terminate()
			return false, nil
		}
	}
	rightArgs := []string{"split-window", "-h", "-l", "60%", "-t", session + ":0.0", "--",
		binary, "lab", "run", authored.Definition.ID, "--panel-state", stateFile}
	if err := tmuxRun(ctx, rightArgs...); err != nil {
		terminate()
		fmt.Fprintf(stderr, "Séparation tmux indisponible, retour au mode classique : %v\n", err)
		return false, nil
	}
	if err := tmuxRun(ctx, "select-pane", "-t", session+":0.1"); err != nil {
		terminate()
		return false, nil
	}
	return true, tmuxAttach(ctx, session, stdin, stdout, stderr)
}

// runTmuxLabPanel is a trusted host-only read-only pane. Guest PTY bytes must
// never be directly injected into this pane.
func runTmuxLabPanel(ctx context.Context, authored lab.Lab, stateFile string, stdin io.Reader, stdout io.Writer) error {
	file, ok := stdout.(*os.File)
	if !ok || !terminal.IsTerminal(file) {
		return errors.New("tmux lab panel requires an interactive terminal")
	}
	fmt.Fprint(stdout, "\x1b[?1049h\x1b[?25l")
	defer fmt.Fprint(stdout, "\x1b[?25h\x1b[?1049l")
	inputFile, ok := stdin.(*os.File)
	if !ok || !terminal.IsTerminal(inputFile) {
		return errors.New("tmux lab panel requires a real terminal on stdin")
	}
	state, err := terminal.MakeRaw(inputFile)
	if err != nil {
		return err
	}
	defer terminal.Restore(inputFile, state)
	pressed := make(chan byte, 16)
	go func() {
		reader := make([]byte, 1)
		for {
			n, err := stdin.Read(reader)
			if err != nil {
				return
			}
			if n > 0 {
				select {
				case pressed <- reader[0]:
				default:
				}
			}
		}
	}()
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	var prev string
	scroll := 0
	for {
		state := labPanelState{}
		if payload, err := os.ReadFile(stateFile); err == nil && len(payload) < 64<<10 {
			_ = json.Unmarshal(payload, &state)
		}
		width, height, err := terminal.Size(file)
		if err != nil {
			return err
		}
		frame := renderTmuxLabPanel(authored, state, width, height, scroll)
		if frame != prev {
			fmt.Fprint(stdout, "\x1b[H\x1b[2J", frame)
			prev = frame
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case key := <-pressed:
			switch key {
			case '\r', '\n':
				if state.Finished {
					return nil
				}
			case 'j', 'J':
				scroll++
			case 'k', 'K':
				if scroll > 0 {
					scroll--
				}
			case 'g':
				scroll = 0
			}
		case <-ticker.C:
		}
	}
}

func panelWrapped(text string, width int) []string {
	if width < 12 {
		width = 12
	}
	return strings.Split(labWrap(lpicui.SanitizeText(text), width), "\n")
}

func tmuxPanelColor(line string) string {
	if os.Getenv("NO_COLOR") != "" {
		return line
	}
	switch {
	case strings.HasPrefix(line, "MISSION"), strings.HasPrefix(line, "OBJECTIFS"),
		strings.HasPrefix(line, "INDICES"), strings.HasPrefix(line, "VÉRIFICATION"),
		strings.HasPrefix(line, "RÉSULTAT"):
		return "\x1b[1;36m" + line + "\x1b[0m"
	case strings.HasPrefix(line, "✗"), strings.HasPrefix(line, "À CORRIGER"),
		strings.HasPrefix(line, "ERREUR"):
		return "\x1b[1;31m" + line + "\x1b[0m"
	case strings.HasPrefix(line, "✓"), strings.HasPrefix(line, "VALIDÉ"):
		return "\x1b[1;32m" + line + "\x1b[0m"
	default:
		return line
	}
}

func renderTmuxLabPanel(authored lab.Lab, state labPanelState, width, height, scroll int) string {
	if width < 20 || height < 10 {
		return "Agrandis le terminal pour afficher le lab.\n"
	}
	inner := width - 4
	top := []string{
		"MISSION  " + authored.Definition.ObjectiveIDs[0],
		"",
	}
	top = append(top, panelWrapped(authored.Definition.TitleFR, inner)...)
	top = append(top, "")
	top = append(top, panelWrapped(authored.Definition.BriefFR, inner)...)
	top = append(top, "", "OBJECTIFS")
	for i, criterion := range authored.Definition.SuccessCriteriaFR {
		top = append(top, panelWrapped(fmt.Sprintf("%d. %s", i+1, criterion), inner)...)
	}
	top = append(top, "", fmt.Sprintf("INDICES · %d/%d", state.HintLevel, len(authored.Hints)))
	for _, hint := range authored.Hints {
		if hint.Level > state.HintLevel {
			break
		}
		top = append(top, panelWrapped(fmt.Sprintf("[%d] %s", hint.Level, hint.ContentFR), inner)...)
	}
	if state.HintLevel == 0 {
		top = append(top, "Aucun indice consulté. :hint dans le shell.")
	}
	bottom := []string{"VÉRIFICATION"}
	if state.Check == "" {
		bottom = append(bottom, "Aucun check exécuté.", "Tape :check dans le shell à droite.")
	} else {
		for _, line := range strings.Split(state.Check, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "RÉSULTAT") || strings.HasPrefix(line, "ERREUR") || strings.HasPrefix(line, "VALIDÉ") ||
				strings.HasPrefix(line, "À CORRIGER") || strings.HasPrefix(line, "✗") ||
				strings.HasPrefix(line, "✓") || strings.Contains(line, "Lab réussi") ||
				strings.HasPrefix(line, "Détail :") {
				bottom = append(bottom, panelWrapped(line, inner)...)
			}
		}
	}
	if state.Finished {
		bottom = append(bottom, "", "Session terminée. Entrée : retour au menu.")
	}
	// Reserve a stable bottom area for the check results, rather than pushing
	// the entire mission down after every verification.
	bottomMax := height / 3
	if bottomMax < 6 {
		bottomMax = 6
	}
	if len(bottom) > bottomMax {
		bottom = append(bottom[:bottomMax-1], "… autres critères non affichés")
	}
	topMax := height - len(bottom) - 2
	if topMax < 2 {
		topMax = 2
	}
	if len(top) > topMax {
		maxScroll := len(top) - topMax + 1
		if scroll > maxScroll {
			scroll = maxScroll
		}
		if scroll < 0 {
			scroll = 0
		}
		last := scroll + topMax - 1
		if last > len(top) {
			last = len(top)
		}
		visible := append([]string(nil), top[scroll:last]...)
		visible = append(visible, "↑↓ j/k : défiler · g : début")
		top = visible
	}
	var out strings.Builder
	for _, line := range top {
		fmt.Fprintf(&out, "  %s\n", tmuxPanelColor(line))
	}
	for i := len(top); i < topMax; i++ {
		out.WriteString("\n")
	}
	out.WriteString(strings.Repeat("─", width-1))
	out.WriteString("\n")
	for _, line := range bottom {
		fmt.Fprintf(&out, "  %s\n", tmuxPanelColor(line))
	}
	return out.String()
}
