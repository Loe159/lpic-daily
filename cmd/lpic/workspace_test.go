package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	lpicdaily "github.com/Loe159/lpic-daily"
	"github.com/Loe159/lpic-daily/internal/lab"
)

func loadWorkspaceTestLab(t *testing.T) lab.Lab {
	t.Helper()
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatal(err)
	}
	authored, err := findLab(labs, "lpic1.103.5.stuck-worker")
	if err != nil {
		t.Fatal(err)
	}
	return authored
}

func TestWorkspaceSnapshotRoundTripAndAtomicReplacement(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "state.json")
	if err := writeWorkspaceSnapshot(path, workspaceSnapshot{LabID: "lab-1", Status: "ready", HintLevel: 2}); err != nil {
		t.Fatal(err)
	}
	state, err := readWorkspaceSnapshot(path)
	if err != nil || state.LabID != "lab-1" || state.HintLevel != 2 {
		t.Fatalf("state=%+v err=%v", state, err)
	}
	if err := writeWorkspaceSnapshot(path, workspaceSnapshot{
		LabID: "lab-1", Checked: true, Checks: []workspaceCheck{{Description: "worker sain", Passed: true}},
	}); err != nil {
		t.Fatal(err)
	}
	state, err = readWorkspaceSnapshot(path)
	if err != nil || !state.Checked || len(state.Checks) != 1 || !state.Checks[0].Passed {
		t.Fatalf("replacement state=%+v err=%v", state, err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("state file permissions: info=%v err=%v", info, err)
	}
}

func TestWorkspacePathIsPrivateAndDeterministic(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	root, err := workspaceRoot()
	if err != nil {
		t.Fatal(err)
	}
	authored := loadWorkspaceTestLab(t)
	name := workspaceName(authored.Definition.ID)
	path := filepath.Join(root, name+".json")
	if err := validateWorkspacePath(authored.Definition.ID, path); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"/tmp/other.json", filepath.Join(root, "fake.json"), filepath.Join(root, "..", "escape.json")} {
		if err := validateWorkspacePath(authored.Definition.ID, bad); err == nil {
			t.Errorf("accepted foreign workspace path %q", bad)
		}
	}
	if name != workspaceName(authored.Definition.ID) {
		t.Fatal("workspace ID changed")
	}
}

func TestWorkspacePanelRendersValidatedLayoutAndHints(t *testing.T) {
	authored := loadWorkspaceTestLab(t)
	snapshot := workspaceSnapshot{
		LabID: authored.Definition.ID,
		Status: "ready", HintLevel: 1, Checked: true,
		Checks: []workspaceCheck{
			{Description: "worker sain", Passed: true},
			{Description: "worker bloqué", Passed: false, Detail: "vérification incomplète"},
		},
	}
	rendered := renderWorkspacePanel(authored, snapshot, 50, 35, 0, 0)
	for _, required := range []string{
		"MISSION", "OBJECTIFS", "INDICES  1/4", "VÉRIFICATION",
		"1/2 critères OK", "worker sain", "worker bloqué",
		"vérification incomplète", "j/k : consigne",
	} {
		if !strings.Contains(rendered, required) {
			t.Errorf("missing %q in workspace panel", required)
		}
	}
	if strings.Contains(rendered, authored.Hints[3].ContentFR) {
		t.Fatal("unrevealed fourth hint was exposed")
	}
	if strings.Contains(rendered, "\x1b[?25l") == false {
		t.Fatal("missing terminal redraw escape")
	}
}

func TestWorkspaceRendererFitsAndSanitizes(t *testing.T) {
	wrapped := workspaceWrap("un deux trois quatre cinq", 12)
	if len(wrapped) < 2 {
		t.Fatalf("wrapped=%q", wrapped)
	}
	fitted := workspaceFit("Ceci est beaucoup trop long", 10)
	if len([]rune(fitted)) != 10 {
		t.Fatalf("fitted length=%d", len([]rune(fitted)))
	}
	fitted = workspaceFit("\x1b[31mALERTE\x1b[0m", 20)
	if strings.Contains(fitted, "\x1b") {
		t.Fatalf("untrusted ANSI control reached panel: %q", fitted)
	}
}

func TestTmuxWorkspaceFallsBackWithoutTTY(t *testing.T) {
	authored := loadWorkspaceTestLab(t)
	used, err := runTmuxLabWorkspace(t.Context(), authored, strings.NewReader(""), &strings.Builder{}, &strings.Builder{})
	if err != nil || used {
		t.Fatalf("used=%v err=%v, want a clean fallback", used, err)
	}
}
