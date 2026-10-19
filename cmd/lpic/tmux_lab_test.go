package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	lpicdaily "github.com/Loe159/lpic-daily"
	"github.com/Loe159/lpic-daily/internal/lab"
)

func tmuxSampleLab(t *testing.T) lab.Lab {
	t.Helper()
	all, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatal(err)
	}
	got, err := findLab(all, "lpic1.103.5.stuck-worker")
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestTmuxSessionNameIsStableAndSafe(t *testing.T) {
	a := tmuxLabName("lpic1.103.5.stuck-worker")
	b := tmuxLabName("lpic1.103.5.stuck-worker")
	c := tmuxLabName("lpic1.103.4.bulk-argument-dispatch")
	if a != b || a == c || strings.ContainsAny(a, " ./;:$\n") {
		t.Fatalf("tmux session IDs not stable and isolated: %q %q %q", a, b, c)
	}
}

func TestTmuxPanelStoresHintsAndChecksWithoutGuestEscapeSequences(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "status.json")
	reporter := &labPanelReporter{path: path}
	if err := reporter.update(2, "RÉSULTAT  ·  1/3 OK\n\x1b[2J  ✗ Relevé incomplet"); err != nil {
		t.Fatal(err)
	}
	if err := reporter.update(1, ""); err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var state labPanelState
	if err := json.Unmarshal(payload, &state); err != nil {
		t.Fatal(err)
	}
	if state.HintLevel != 2 || !strings.Contains(state.Check, "Relevé incomplet") || strings.ContainsRune(state.Check, '\x1b') {
		t.Fatalf("unsafe or incorrect panel state: %#v", state)
	}
	if mode, err := os.Stat(path); err != nil || mode.Mode().Perm()&0o077 != 0 {
		t.Fatalf("panel state must remain private; stat=%v err=%v", mode, err)
	}
	reporter.finish()
	payload, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(payload, &state); err != nil || !state.Finished {
		t.Fatalf("finished session must be recorded, got=%+v err=%v", state, err)
	}
}

func TestTmuxLayoutBSeparatesMissionAndStableCheckResults(t *testing.T) {
	authored := tmuxSampleLab(t)
	state := labPanelState{
		HintLevel: 1,
		Check:     "RÉSULTAT  ·  2/11 OK\nÀ CORRIGER\n  ✗ Relevé incomplet\nVALIDÉ\n  ✓ Worker sain",
	}
	rendered := renderTmuxLabPanel(authored, state, 58, 40, 0)
	for _, want := range []string{"MISSION", "OBJECTIFS", "INDICES · 1/4", "VÉRIFICATION", "2/11 OK", "Relevé incomplet"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("layout B missing %q: %q", want, rendered)
		}
	}
	if strings.Contains(rendered, "Aucun check exécuté") || strings.Contains(rendered, "aucun indice") {
		t.Fatalf("unexpected empty-panel state: %q", rendered)
	}
	// A long scenario scrolls only the mission area; checks remain pinned.
	scrolled := renderTmuxLabPanel(authored, state, 40, 28, 500)
	if !strings.Contains(scrolled, "VÉRIFICATION") || !strings.Contains(scrolled, "Relevé incomplet") {
		t.Fatalf("checks should stay visible after scrolling: %q", scrolled)
	}
}

func TestTmuxLayoutBRetainsExistingDisclosedHints(t *testing.T) {
	authored := tmuxSampleLab(t)
	rendered := renderTmuxLabPanel(authored, labPanelState{HintLevel: 4}, 60, 50, 500)
	if !strings.Contains(rendered, "INDICES · 4/4") && !strings.Contains(rendered, "↑↓ j/k") {
		t.Fatalf("four pre-disclosed hints should remain available through scroll: %q", rendered)
	}
}

func TestTmuxPanelRejectsTinyTerminals(t *testing.T) {
	if got := renderTmuxLabPanel(tmuxSampleLab(t), labPanelState{}, 10, 6, 0); !strings.Contains(got, "Agrandis") {
		t.Fatalf("unexpected small-terminal result: %q", got)
	}
}
