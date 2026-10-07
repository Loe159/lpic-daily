package lab_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	lpicdaily "github.com/Loe159/lpic-daily"
	"github.com/Loe159/lpic-daily/internal/content"
	"github.com/Loe159/lpic-daily/internal/curriculum"
	"github.com/Loe159/lpic-daily/internal/lab"
	"github.com/Loe159/lpic-daily/internal/runner"
)

const (
	shellEnvironmentRepairID = "lpic1.103.1.shell-environment-repair"
	sharedDropboxID          = "lpic1.104.5.shared-dropbox"
	stuckWorkerID            = "lpic1.103.5.stuck-worker"
)

func loadBuiltinLab(t *testing.T, id string) lab.Lab {
	t.Helper()
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	for _, authored := range labs {
		if authored.Definition.ID == id {
			return authored
		}
	}
	t.Fatalf("built-in lab %s not found", id)
	return lab.Lab{}
}

func TestLoadBuiltinShellEnvironmentRepair(t *testing.T) {
	got := loadBuiltinLab(t, shellEnvironmentRepairID)

	if len(got.Definition.ConceptIDs) != 7 {
		t.Fatalf("concepts = %d, want 7", len(got.Definition.ConceptIDs))
	}
	if len(got.Hints) != 4 {
		t.Fatalf("hints = %d, want 4", len(got.Hints))
	}
	if !strings.Contains(got.SetupScript, "/opt/lpic/shadow/bin") || !strings.Contains(got.SetupScript, "REPORT_FILE=") {
		t.Fatal("shell environment setup script is incomplete")
	}

	checks, err := got.CompileChecks()
	if err != nil {
		t.Fatalf("CompileChecks() error = %v", err)
	}
	if len(checks) != 7 {
		t.Fatalf("checks = %d, want 7", len(checks))
	}
}

func TestLoadBuiltinSharedDropbox(t *testing.T) {
	got := loadBuiltinLab(t, sharedDropboxID)

	if len(got.Definition.ConceptIDs) != 8 || !strings.Contains(strings.Join(got.Definition.ConceptIDs, "\n"), "lpic1.104.5.suid") {
		t.Fatalf("shared-dropbox concepts = %v, want all 8 permission concepts", got.Definition.ConceptIDs)
	}
	if len(got.Hints) != 4 {
		t.Fatalf("hints = %d, want 4", len(got.Hints))
	}
	for index, hint := range got.Hints {
		if hint.Level != index+1 {
			t.Fatalf("hint %d level = %d", index, hint.Level)
		}
	}
	if !strings.Contains(got.SetupScript, "getent group project") || !strings.Contains(got.SetupScript, "/srv/shared") {
		t.Fatal("setup script was not loaded")
	}
	if got.ReferenceSolutionRef == "" {
		t.Fatal("reference solution ref should remain traceable")
	}

	checks, err := got.CompileChecks()
	if err != nil {
		t.Fatalf("CompileChecks() error = %v", err)
	}
	if len(checks) != 9 {
		t.Fatalf("checks = %d, want 9", len(checks))
	}
}

func TestLoadBuiltinStuckWorker(t *testing.T) {
	got := loadBuiltinLab(t, stuckWorkerID)

	if len(got.Definition.ConceptIDs) != 7 {
		t.Fatalf("concepts = %d, want 7", len(got.Definition.ConceptIDs))
	}
	if len(got.Hints) != 4 {
		t.Fatalf("hints = %d, want 4", len(got.Hints))
	}
	if !strings.Contains(got.SetupScript, "stuck-worker") || !strings.Contains(got.SetupScript, "lpic-signal-probe") {
		t.Fatal("stuck-worker setup script is incomplete")
	}

	checks, err := got.CompileChecks()
	if err != nil {
		t.Fatalf("CompileChecks() error = %v", err)
	}
	if len(checks) != 10 {
		t.Fatalf("checks = %d, want 10", len(checks))
	}
	if !got.Definition.NeedsPersistentShell {
		t.Fatal("stuck-worker must require the persistent PTY shell")
	}
}

func TestBuiltinLabsCompileRunnerDefinitionsAndChecksAtLoadTime(t *testing.T) {
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	for _, authored := range labs {
		if _, err := authored.RunnerDefinition(); err != nil {
			t.Fatalf("%s RunnerDefinition() error = %v", authored.Definition.ID, err)
		}
		if _, err := authored.CompileChecks(); err != nil {
			t.Fatalf("%s CompileChecks() error = %v", authored.Definition.ID, err)
		}
	}
}

func TestBuiltinLabsUseBackendSpecificCapabilityContracts(t *testing.T) {
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	for _, authored := range labs {
		switch authored.Definition.Environment.Backend {
		case "podman":
			if authored.Definition.Environment.CapabilityProfile == "full-machine" {
				t.Fatalf("%s uses VM capability profile on Podman", authored.Definition.ID)
			}
		case "libvirt":
			if authored.Definition.Environment.CapabilityProfile != "full-machine" {
				t.Fatalf("%s libvirt profile = %q, want full-machine", authored.Definition.ID, authored.Definition.Environment.CapabilityProfile)
			}
			if len(authored.Definition.Environment.WritableGuestPaths) != 0 {
				t.Fatalf("%s libvirt writable_guest_paths = %#v, want empty", authored.Definition.ID, authored.Definition.Environment.WritableGuestPaths)
			}
		}
	}
}

func TestBuiltinHintLaddersAreCompleteAndMatchEvidencePolicy(t *testing.T) {
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	for _, authored := range labs {
		if len(authored.Hints) != 4 {
			t.Fatalf("%s hints = %d, want 4", authored.Definition.ID, len(authored.Hints))
		}
		for index, hint := range authored.Hints {
			wantLevel := index + 1
			if hint.Level != wantLevel {
				t.Fatalf("%s hint[%d] level = %d, want %d", authored.Definition.ID, index, hint.Level, wantLevel)
			}
			switch hint.Level {
			case 1:
				if hint.EvidenceImpact != "none" && hint.EvidenceImpact != "minor" {
					t.Fatalf("%s level-1 impact = %q", authored.Definition.ID, hint.EvidenceImpact)
				}
			case 2, 3:
				if hint.EvidenceImpact != "material" {
					t.Fatalf("%s level-%d impact = %q, want material", authored.Definition.ID, hint.Level, hint.EvidenceImpact)
				}
			case 4:
				if hint.EvidenceImpact != "solution-revealed" {
					t.Fatalf("%s level-4 impact = %q, want solution-revealed", authored.Definition.ID, hint.EvidenceImpact)
				}
			}
		}
	}
}

func TestSessionRunsSetupAndStateChecks(t *testing.T) {
	authored := loadBuiltinLab(t, sharedDropboxID)
	fake := &fakeRunner{}

	session, err := lab.Start(context.Background(), authored, fake)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if !fake.started {
		t.Fatal("runner was not started")
	}
	if !strings.Contains(strings.Join(fake.exec.Argv, " "), "/usr/bin/bash -eu -c") {
		t.Fatalf("setup argv = %v", fake.exec.Argv)
	}
	if !strings.Contains(fake.exec.Argv[len(fake.exec.Argv)-1], "getent group project") {
		t.Fatal("setup script was not passed into sandbox exec")
	}

	results, err := session.Evaluate(context.Background())
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	for _, result := range results {
		if !result.Pass {
			t.Fatalf("check failed: %#v", result)
		}
	}

	if err := session.Close(context.Background()); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if !fake.destroyed {
		t.Fatal("runner was not destroyed")
	}
}

func TestSessionResetRestartsAndReplaysSetup(t *testing.T) {
	authored := loadBuiltinLab(t, sharedDropboxID)
	fake := &fakeRunner{}

	session, err := lab.Start(context.Background(), authored, fake)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer session.Close(context.Background())

	if fake.startCalls != 1 || fake.execCalls != 1 {
		t.Fatalf("initial lifecycle start=%d exec=%d, want 1/1", fake.startCalls, fake.execCalls)
	}
	if err := session.Reset(context.Background()); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}
	if fake.resetCalls != 1 {
		t.Fatalf("reset calls = %d, want 1", fake.resetCalls)
	}
	if fake.startCalls != 2 || fake.execCalls != 2 {
		t.Fatalf("reset lifecycle start=%d exec=%d, want 2/2", fake.startCalls, fake.execCalls)
	}
	if !strings.Contains(fake.exec.Argv[len(fake.exec.Argv)-1], "getent group project") {
		t.Fatal("reset did not replay the lab setup")
	}
}

func TestVMSetupNoneSkipsGuestExec(t *testing.T) {
	authored := loadBuiltinLab(t, sharedDropboxID)
	authored.Definition.ID = "lpic1.104.1.test-vm"
	authored.Definition.Environment.Backend = "libvirt"
	authored.Definition.Environment.CapabilityProfile = "full-machine"
	authored.Definition.Environment.Machine = &lab.Machine{
		Firmware: "uefi",
		ExtraDisks: []lab.MachineDisk{{
			ID:     "data",
			SizeMB: 512,
		}},
	}
	authored.Definition.Setup = lab.Setup{ExecutionScope: "none"}
	authored.SetupScript = ""

	fake := &fakeRunner{failExec: true}
	session, err := lab.Start(context.Background(), authored, fake)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer session.Close(context.Background())

	if fake.execCalls != 0 {
		t.Fatalf("VM setup unexpectedly called Exec %d time(s)", fake.execCalls)
	}
	if fake.definition.Machine == nil || len(fake.definition.Machine.ExtraDisks) != 1 {
		t.Fatalf("VM definition = %#v", fake.definition)
	}
}

func TestDestructiveSetupCannotModifyHostSentinelThroughLabOrchestration(t *testing.T) {
	sentinel := filepath.Join(t.TempDir(), "host-sentinel")
	if err := os.WriteFile(sentinel, []byte("safe"), 0o600); err != nil {
		t.Fatalf("write sentinel: %v", err)
	}

	authored := loadBuiltinLab(t, sharedDropboxID)
	authored.SetupScript = fmt.Sprintf("printf 'pwned' > %q", sentinel)
	fake := &fakeRunner{}

	session, err := lab.Start(context.Background(), authored, fake)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer session.Close(context.Background())

	got, err := os.ReadFile(sentinel)
	if err != nil {
		t.Fatalf("read sentinel: %v", err)
	}
	if string(got) != "safe" {
		t.Fatalf("host sentinel changed to %q", got)
	}
	if len(fake.exec.Argv) != 4 || fake.exec.Argv[0] != "/usr/bin/bash" {
		t.Fatalf("setup was not delegated as structured sandbox argv: %v", fake.exec.Argv)
	}
	if fake.exec.Argv[3] != authored.SetupScript {
		t.Fatalf("setup body = %q, want %q", fake.exec.Argv[3], authored.SetupScript)
	}
}

type fakeRunner struct {
	definition      runner.Definition
	exec            runner.ExecRequest
	execCalls       int
	startCalls      int
	resetCalls      int
	failExec        bool
	started         bool
	destroyed       bool
	destroyCalls    int
	destroyFailures int
}

func (fake *fakeRunner) Prepare(_ context.Context, definition runner.Definition) (runner.Instance, error) {
	fake.definition = definition
	return runner.Instance{ID: "fake-lab"}, nil
}

func (fake *fakeRunner) Start(_ context.Context, _ runner.Instance) error {
	fake.started = true
	fake.startCalls++
	return nil
}

func (fake *fakeRunner) Exec(_ context.Context, _ runner.Instance, request runner.ExecRequest) (runner.ExecResult, error) {
	fake.execCalls++
	fake.exec = request
	if fake.failExec {
		return runner.ExecResult{}, errors.New("Exec must not be called")
	}
	return runner.ExecResult{ExitCode: 0}, nil
}

func (fake *fakeRunner) Stat(_ context.Context, _ runner.Instance, path string) (runner.FileInfo, error) {
	switch path {
	case "/srv/shared":
		return runner.FileInfo{
			Path: path, Mode: 0o3770, UID: 0, GID: 2000,
			User: "root", Group: "project", IsDir: true,
		}, nil
	case "/srv/shared/team-note":
		return runner.FileInfo{
			Path: path, Mode: 0o0660, UID: 0, GID: 2000,
			User: "root", Group: "project", IsDir: false,
		}, nil
	case "/srv/shared/audit-helper":
		return runner.FileInfo{
			Path: path, Mode: 0o0755, UID: 0, GID: 0,
			User: "root", Group: "root", IsDir: false,
		}, nil
	default:
		return runner.FileInfo{}, errors.New("not found")
	}
}

func (fake *fakeRunner) ReadFile(context.Context, runner.Instance, string, int64) ([]byte, error) {
	return nil, errors.New("not implemented in fake")
}

func (fake *fakeRunner) Processes(context.Context, runner.Instance) ([]runner.Process, error) {
	return nil, nil
}

func (fake *fakeRunner) Reset(context.Context, runner.Instance) error {
	fake.resetCalls++
	return nil
}

func (fake *fakeRunner) Destroy(context.Context, runner.Instance) error {
	fake.destroyCalls++
	if fake.destroyFailures > 0 {
		fake.destroyFailures--
		return errors.New("transient destroy failure")
	}
	fake.destroyed = true
	return nil
}

func (fake *fakeRunner) Close() error {
	return nil
}

func TestStartSurfacesCleanupFailure(t *testing.T) {
	authored := loadBuiltinLab(t, sharedDropboxID)
	fake := &fakeRunner{failExec: true, destroyFailures: 1}

	_, err := lab.Start(context.Background(), authored, fake)
	if err == nil || !strings.Contains(err.Error(), "cleanup failed start") {
		t.Fatalf("Start() error = %v, want surfaced cleanup failure", err)
	}
	if fake.destroyCalls != 1 {
		t.Fatalf("destroy calls = %d, want 1", fake.destroyCalls)
	}
}

func TestFailedResetCleanupRemainsRetryable(t *testing.T) {
	authored := loadBuiltinLab(t, sharedDropboxID)
	fake := &fakeRunner{}
	session, err := lab.Start(context.Background(), authored, fake)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	fake.failExec = true
	fake.destroyFailures = 1
	err = session.Reset(context.Background())
	if err == nil || !strings.Contains(err.Error(), "cleanup failed reset") {
		t.Fatalf("Reset() error = %v, want cleanup failure", err)
	}

	fake.failExec = false
	if err := session.Close(context.Background()); err != nil {
		t.Fatalf("Close() retry error = %v", err)
	}
	if fake.destroyCalls != 2 || !fake.destroyed {
		t.Fatalf("cleanup retry destroyCalls=%d destroyed=%v, want 2/true", fake.destroyCalls, fake.destroyed)
	}
}

func TestBuiltinLabsProvideTwoPracticeContextsPerActiveConcept(t *testing.T) {
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	curriculumBundle, err := curriculum.Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("curriculum.Load() error = %v", err)
	}

	contexts := make(map[string]map[string]struct{})
	for _, authored := range labs {
		evidenced := make(map[string]struct{})
		for _, check := range authored.Definition.Checks {
			for _, conceptID := range check.ConceptIDs {
				evidenced[conceptID] = struct{}{}
			}
		}
		for conceptID := range evidenced {
			if contexts[conceptID] == nil {
				contexts[conceptID] = make(map[string]struct{})
			}
			contexts[conceptID][authored.Definition.PracticeContext] = struct{}{}
		}
	}

	for _, concept := range curriculumBundle.Concepts.Concepts {
		if !concept.Active {
			continue
		}
		if got := len(contexts[concept.ID]); got < 2 {
			t.Errorf("concept %s practical contexts = %d, want at least 2", concept.ID, got)
		}
	}
}

func TestGeneratedStandaloneLabsUseConceptAnchors(t *testing.T) {
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	curriculumBundle, err := curriculum.Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("curriculum.Load() error = %v", err)
	}

	conceptByID := make(map[string]curriculum.Concept)
	objectiveExam := make(map[string]string)
	for _, objective := range curriculumBundle.Objectives.Objectives {
		if objective.Active {
			objectiveExam[objective.ID] = objective.Exam
		}
	}
	for _, concept := range curriculumBundle.Concepts.Concepts {
		if concept.Active {
			conceptByID[concept.ID] = concept
		}
	}
	seen := make(map[string]int)
	commandEvidence := make(map[string]int)
	for _, authored := range labs {
		if !strings.Contains(authored.Definition.ID, ".standalone-") {
			continue
		}
		if len(authored.Definition.ConceptIDs) != 1 {
			t.Fatalf("%s generated fallback spans %d concepts, want exactly 1", authored.Definition.ID, len(authored.Definition.ConceptIDs))
		}
		historyPath, _ := lab.StandaloneCommandHistoryPath(authored.Definition)
		for _, conceptID := range authored.Definition.ConceptIDs {
			concept := conceptByID[conceptID]
			wantTerms := "TERMS=" + strings.Join(concept.AnchorTerms, ",")
			foundStructuredEvidence := false
			foundRuntimeCommand := false
			for _, check := range authored.Definition.Checks {
				if !slices.Contains(check.ConceptIDs, conceptID) {
					continue
				}
				if check.Path == historyPath {
					foundRuntimeCommand = true
					continue
				}
				if check.Type == "file-content-regex" &&
					strings.Contains(check.Pattern, regexp.QuoteMeta(wantTerms)) {
					foundStructuredEvidence = true
				}
			}
			if !foundStructuredEvidence {
				t.Errorf("%s has no structured evidence check for %s anchors %v", authored.Definition.ID, conceptID, concept.AnchorTerms)
			}
			for _, anchor := range concept.AnchorTerms {
				if !strings.Contains(authored.Definition.BriefFR, "`"+anchor+"`") {
					t.Errorf("%s brief misses %s anchor %q", authored.Definition.ID, conceptID, anchor)
				}
			}
			if foundRuntimeCommand {
				commandEvidence[conceptID]++
			}
			seen[conceptID]++
		}
	}
	for conceptID, concept := range conceptByID {
		if seen[conceptID] < 2 {
			t.Errorf("%s appears in %d generated standalone labs, want at least 2", conceptID, seen[conceptID])
		}
		if objectiveExam[concept.ObjectiveID] != "101" {
			continue
		}
		hasCommandAnchor := false
		for _, anchor := range concept.AnchorTerms {
			switch anchor {
			case "vi", "vim", "screen", "tmux", "shutdown", "init", "telinit",
				"grub-install", "grub-mkconfig", "dpkg-reconfigure":
				continue
			}
			usage := strings.TrimSpace(content.PedagogicalTermUsage(anchor))
			if usage == anchor || strings.HasPrefix(usage, anchor+" ") || strings.HasPrefix(usage, anchor+" ;") {
				hasCommandAnchor = true
				break
			}
		}
		if conceptID == "lpic1.103.1.set-env-et-portee-des-variables" {
			// This exercise compares shell-local and exported state inside one
			// persistent shell. The learner reflection is the evidence source;
			// a separate one-shot command would run in another shell.
			continue
		}
		if hasCommandAnchor && commandEvidence[conceptID] < 2 {
			t.Errorf("%s runtime command-evidence contexts = %d, want at least 2", conceptID, commandEvidence[conceptID])
		}
	}
}

func TestGeneratedStandaloneCommandEvidenceRequiresCommandPosition(t *testing.T) {
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("lab.LoadAll() error = %v", err)
	}
	for _, authored := range labs {
		if !strings.HasPrefix(authored.Definition.ID, "lpic1.103.5.") || !strings.Contains(authored.Definition.ID, ".standalone-") {
			continue
		}
		historyPath, ok := lab.StandaloneCommandHistoryPath(authored.Definition)
		if !ok {
			continue
		}
		for _, check := range authored.Definition.Checks {
			if check.Path != historyPath || !slices.Contains(check.ConceptIDs, "lpic1.103.5.inspection-arbres-ressources") {
				continue
			}
			re := regexp.MustCompile(check.Pattern)
			if re.MatchString("EXIT=0\tCOMMAND=echo ps") {
				t.Fatalf("%s accepts a mere textual mention of ps: %q", authored.Definition.ID, check.Pattern)
			}
			if !re.MatchString("EXIT=0\tCOMMAND=ps aux") {
				t.Fatalf("%s rejects actual ps execution: %q", authored.Definition.ID, check.Pattern)
			}
			return
		}
	}
	t.Fatal("no generated ps command-evidence check found")
}


func TestGeneratedStandaloneLabsStayConcreteAndConcise(t *testing.T) {
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	curriculumBundle, err := curriculum.Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("curriculum.Load() error = %v", err)
	}

	conceptByID := make(map[string]curriculum.Concept)
	for _, concept := range curriculumBundle.Concepts.Concepts {
		if concept.Active {
			conceptByID[concept.ID] = concept
		}
	}

	generated := 0
	for _, authored := range labs {
		if !strings.Contains(authored.Definition.ID, ".standalone-") {
			continue
		}
		generated++
		if len(authored.Definition.ConceptIDs) != 1 {
			t.Fatalf("%s concept count = %d, want 1", authored.Definition.ID, len(authored.Definition.ConceptIDs))
		}
		concept := conceptByID[authored.Definition.ConceptIDs[0]]
		brief := authored.Definition.BriefFR

		for _, forbidden := range []string{
			"Utilise ",
			" pour montrer concrètement ",
			"Reproduis ",
			"Fais au moins une manipulation",
			"CONCEPT=",
			"TERMS=",
			"COMMAND=",
			"OBSERVATION=",
			"EXPLANATION=",
			"lpic-daily-evidence",
			"## Contexte de diagnostic",
		} {
			if strings.Contains(brief, forbidden) {
				t.Errorf("%s contains generic/internal learner text %q", authored.Definition.ID, forbidden)
			}
		}
		if strings.Contains(brief, "**") {
			t.Errorf("%s still contains raw bold markdown in its brief", authored.Definition.ID)
		}
		if !strings.Contains(brief, "## À faire") || !strings.Contains(brief, "Quand tu as terminé, tape `:check`.") {
			t.Errorf("%s does not use the compact learner format", authored.Definition.ID)
		}
		for _, anchor := range concept.AnchorTerms {
			if !strings.Contains(brief, "`"+anchor+"`") {
				t.Errorf("%s brief misses anchor %q", authored.Definition.ID, anchor)
			}
		}

		specific := concept.ID == "lpic1.103.1.set-env-et-portee-des-variables"
		for _, anchor := range concept.AnchorTerms {
			if strings.Contains(brief, content.PedagogicalTermExplanation(anchor, concept.ObjectiveID)) {
				specific = true
				break
			}
		}
		if !specific {
			t.Errorf("%s brief contains no concept-specific explanation", authored.Definition.ID)
		}
		if words := len(strings.Fields(brief)); words > 130 {
			t.Errorf("%s brief = %d words, want <= 130", authored.Definition.ID, words)
		}
		if words := len(strings.Fields(authored.Definition.DebriefFR)); words > 80 {
			t.Errorf("%s debrief = %d words, want <= 80", authored.Definition.ID, words)
		}

		for _, hint := range authored.Hints {
			for _, forbidden := range []string{
				"Fais une petite manipulation",
				"Repars de la consigne",
				"montre concrètement",
				"CONCEPT=",
				"TERMS=",
			} {
				if strings.Contains(hint.ContentFR, forbidden) {
					t.Errorf("%s hint %d contains generic/internal text %q", authored.Definition.ID, hint.Level, forbidden)
				}
			}
			if words := len(strings.Fields(hint.ContentFR)); words > 55 {
				t.Errorf("%s hint %d = %d words, want <= 55", authored.Definition.ID, hint.Level, words)
			}
		}
	}
	if generated != 618 {
		t.Fatalf("generated standalone labs = %d, want 618", generated)
	}
}
