package learning_test

import (
	"testing"
	"time"

	lpicdaily "github.com/Loe159/lpic-daily"
	"github.com/Loe159/lpic-daily/internal/curriculum"
	"github.com/Loe159/lpic-daily/internal/learning"
)

func loadBundle(t *testing.T) *curriculum.Bundle {
	t.Helper()
	bundle, err := curriculum.Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	return bundle
}

func TestSessionStartsWithShellFoundationWithinPhase1(t *testing.T) {
	bundle := loadBundle(t)
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)

	session, err := learning.BuildSession(learning.SessionInput{
		Now:                now,
		Bundle:             bundle,
		Projections:        map[string]learning.MasteryProjection{},
		ObjectiveReadiness: map[string]bool{},
		ScopeObjectives:    bundle.Phase1.SelectedObjectives,
		Policy:             learning.DefaultSessionPolicy(),
	})
	if err != nil {
		t.Fatalf("BuildSession() error = %v", err)
	}
	if len(session.Items) != 1 {
		t.Fatalf("items = %d, want 1 new concept", len(session.Items))
	}
	if session.Items[0].Kind != learning.SessionNew || session.Items[0].ObjectiveID != "103.1" {
		t.Fatalf("first item = %#v, want new 103.1 concept", session.Items[0])
	}
}

func TestSessionUnlocksProcessesAfterShellReadiness(t *testing.T) {
	bundle := loadBundle(t)
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)

	projections := map[string]learning.MasteryProjection{}
	for _, concept := range bundle.Concepts.Concepts {
		if concept.ObjectiveID == "103.1" {
			projections[concept.ID] = learning.MasteryProjection{
				ConceptID:           concept.ID,
				Stage:               learning.StageRecall,
				LastEvidenceAt:      now,
				LastStageEvidenceAt: now,
			}
		}
	}

	session, err := learning.BuildSession(learning.SessionInput{
		Now:                now,
		Bundle:             bundle,
		Projections:        projections,
		ObjectiveReadiness: map[string]bool{"103.1": true},
		ScopeObjectives:    []string{"103.5"},
		Policy:             learning.DefaultSessionPolicy(),
	})
	if err != nil {
		t.Fatalf("BuildSession() error = %v", err)
	}
	if len(session.Items) != 1 || session.Items[0].ObjectiveID != "103.5" || session.Items[0].Kind != learning.SessionNew {
		t.Fatalf("items = %#v, want one new 103.5 concept", session.Items)
	}
}

func TestOverdueReviewPrecedesNewMaterial(t *testing.T) {
	bundle := loadBundle(t)
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)

	var shellConcept string
	for _, concept := range bundle.Concepts.Concepts {
		if concept.ObjectiveID == "103.1" {
			shellConcept = concept.ID
			break
		}
	}

	projections := map[string]learning.MasteryProjection{
		shellConcept: {
			ConceptID:           shellConcept,
			Stage:               learning.StageRecall,
			LastEvidenceAt:      now.Add(-10 * 24 * time.Hour),
			LastStageEvidenceAt: now.Add(-10 * 24 * time.Hour),
		},
	}

	session, err := learning.BuildSession(learning.SessionInput{
		Now:                now,
		Bundle:             bundle,
		Projections:        projections,
		ObjectiveReadiness: map[string]bool{},
		ScopeObjectives:    []string{"103.1"},
		Policy:             learning.DefaultSessionPolicy(),
	})
	if err != nil {
		t.Fatalf("BuildSession() error = %v", err)
	}
	if len(session.Items) < 2 {
		t.Fatalf("items = %#v, want review + new concept", session.Items)
	}
	if session.Items[0].Kind != learning.SessionReview || session.Items[0].ConceptID != shellConcept {
		t.Fatalf("first item = %#v, want overdue review", session.Items[0])
	}
	if session.Items[1].Kind != learning.SessionNew {
		t.Fatalf("second item = %#v, want new concept", session.Items[1])
	}
}

func TestFailedReviewDoesNotPostponeNextReview(t *testing.T) {
	bundle := loadBundle(t)
	now := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)

	var shellConcept string
	for _, concept := range bundle.Concepts.Concepts {
		if concept.ObjectiveID == "103.1" {
			shellConcept = concept.ID
			break
		}
	}

	session, err := learning.BuildSession(learning.SessionInput{
		Now:    now,
		Bundle: bundle,
		Projections: map[string]learning.MasteryProjection{
			shellConcept: {
				ConceptID:           shellConcept,
				Stage:               learning.StageRecall,
				LastEvidenceAt:      now,
				LastStageEvidenceAt: now.Add(-4 * 24 * time.Hour),
			},
		},
		ObjectiveReadiness: map[string]bool{},
		ScopeObjectives:    []string{"103.1"},
		Policy:             learning.DefaultSessionPolicy(),
	})
	if err != nil {
		t.Fatalf("BuildSession() error = %v", err)
	}
	if len(session.Items) == 0 || session.Items[0].Kind != learning.SessionReview {
		t.Fatalf("items = %#v, want review to remain due after later failure", session.Items)
	}
}

func TestExposureRequiresImmediatePracticeBeforeNextConcept(t *testing.T) {
	bundle := loadBundle(t)
	now := time.Date(2026, 9, 27, 2, 0, 0, 0, time.UTC)
	conceptID := bundle.Phase1.ObjectiveConcepts["103.1"][0]

	session, err := learning.BuildSession(learning.SessionInput{
		Now:    now,
		Bundle: bundle,
		Projections: map[string]learning.MasteryProjection{
			conceptID: {
				ConceptID:          conceptID,
				Stage:              learning.StageExposed,
				SuccessfulExposure: 1,
				LastEvidenceAt:     now,
			},
		},
		ObjectiveReadiness: map[string]bool{},
		ScopeObjectives:    []string{"103.1"},
		Policy:             learning.DefaultSessionPolicy(),
	})
	if err != nil {
		t.Fatalf("BuildSession() error = %v", err)
	}
	if len(session.Items) != 1 || session.Items[0].Kind != learning.SessionPractice || session.Items[0].ConceptID != conceptID {
		t.Fatalf("items = %#v, want one immediate practice item", session.Items)
	}
}

func TestSuccessfulRecognitionRequiresPracticalWorkBeforeNextConcept(t *testing.T) {
	bundle := loadBundle(t)
	now := time.Date(2026, 9, 27, 2, 0, 0, 0, time.UTC)
	first := bundle.Phase1.ObjectiveConcepts["103.1"][0]

	session, err := learning.BuildSession(learning.SessionInput{
		Now:    now,
		Bundle: bundle,
		Projections: map[string]learning.MasteryProjection{
			first: {
				ConceptID:             first,
				Stage:                 learning.StageExposed,
				SuccessfulExposure:    1,
				SuccessfulRecognition: 1,
				LastEvidenceAt:        now,
			},
		},
		ObjectiveReadiness: map[string]bool{},
		ScopeObjectives:    []string{"103.1"},
		Policy:             learning.DefaultSessionPolicy(),
	})
	if err != nil {
		t.Fatalf("BuildSession() error = %v", err)
	}
	if len(session.Items) != 1 ||
		session.Items[0].Kind != learning.SessionPractice ||
		session.Items[0].ConceptID != first {
		t.Fatalf("items = %#v, want practical consolidation before any new concept", session.Items)
	}
}

func TestSuccessfulRecallRequiresPracticalWorkBeforeNextConcept(t *testing.T) {
	bundle := loadBundle(t)
	now := time.Date(2026, 10, 6, 10, 0, 0, 0, time.UTC)
	first := bundle.Phase1.ObjectiveConcepts["103.1"][0]

	session, err := learning.BuildSession(learning.SessionInput{
		Now:    now,
		Bundle: bundle,
		Projections: map[string]learning.MasteryProjection{
			first: {
				ConceptID:          first,
				Stage:              learning.StageRecall,
				SuccessfulExposure: 1,
				SuccessfulRecall:   1,
				LastEvidenceAt:     now,
			},
		},
		ObjectiveReadiness: map[string]bool{},
		ScopeObjectives:    []string{"103.1"},
		Policy:             learning.DefaultSessionPolicy(),
	})
	if err != nil {
		t.Fatalf("BuildSession() error = %v", err)
	}
	if len(session.Items) != 1 ||
		session.Items[0].Kind != learning.SessionPractice ||
		session.Items[0].ConceptID != first {
		t.Fatalf("items = %#v, want lab consolidation after recall", session.Items)
	}
}

func TestStartedObjectiveKeepsFocusUntilComplete(t *testing.T) {
	bundle := loadBundle(t)
	now := time.Date(2026, 10, 6, 13, 0, 0, 0, time.UTC)
	first := "lpic1.103.4.stdin-stdout-stderr-et-descripteurs"
	second := "lpic1.103.4.redirections-ecrasement-append"

	session, err := learning.BuildSession(learning.SessionInput{
		Now:    now,
		Bundle: bundle,
		Projections: map[string]learning.MasteryProjection{
			first: {
				ConceptID:           first,
				Stage:               learning.StageGuided,
				SuccessfulExposure:  1,
				SuccessfulRecall:    1,
				SuccessfulGuided:    1,
				LastEvidenceAt:      now,
				LastStageEvidenceAt: now,
			},
		},
		ObjectiveReadiness: map[string]bool{
			"103.1": true,
			"103.4": false,
			"103.5": false,
		},
		ScopeObjectives: []string{"103.4", "103.5"},
		Policy:          learning.DefaultSessionPolicy(),
	})
	if err != nil {
		t.Fatalf("BuildSession() error = %v", err)
	}
	if len(session.Items) != 1 {
		t.Fatalf("items = %#v, want one new concept in started objective", session.Items)
	}
	item := session.Items[0]
	if item.Kind != learning.SessionNew || item.ObjectiveID != "103.4" || item.ConceptID != second {
		t.Fatalf("item = %#v, want next 103.4 concept before switching objective", item)
	}
}

func TestPedagogicalOrderPrioritizesFilePermissionsBeforeProcesses(t *testing.T) {
	bundle := loadBundle(t)
	now := time.Date(2026, 9, 27, 3, 0, 0, 0, time.UTC)

	session, err := learning.BuildSession(learning.SessionInput{
		Now:                now,
		Bundle:             bundle,
		Projections:        map[string]learning.MasteryProjection{},
		ObjectiveReadiness: map[string]bool{"103.1": true},
		ScopeObjectives:    []string{"103.5", "104.5"},
		Policy:             learning.DefaultSessionPolicy(),
	})
	if err != nil {
		t.Fatalf("BuildSession() error = %v", err)
	}
	if len(session.Items) != 1 {
		t.Fatalf("items = %#v, want one new concept", session.Items)
	}
	if session.Items[0].ObjectiveID != "104.5" {
		t.Fatalf("selected objective = %s, want 104.5 before 103.5 in pedagogical order", session.Items[0].ObjectiveID)
	}

	onlyPermissions, err := learning.BuildSession(learning.SessionInput{
		Now:                now,
		Bundle:             bundle,
		Projections:        map[string]learning.MasteryProjection{},
		ObjectiveReadiness: map[string]bool{"103.1": true},
		ConceptReadiness:   map[string]bool{"lpic1.103.1.syntaxe-shell-et-sequences-de-commandes": true},
		ScopeObjectives:    []string{"104.5"},
		Policy:             learning.DefaultSessionPolicy(),
	})
	if err != nil {
		t.Fatalf("BuildSession(104.5) error = %v", err)
	}
	if len(onlyPermissions.Items) != 1 || onlyPermissions.Items[0].ObjectiveID != "104.5" {
		t.Fatalf("recommended prerequisite incorrectly blocked 104.5: %#v", onlyPermissions.Items)
	}
	if onlyPermissions.Items[0].ReasonCode != "concept-prerequisites-ready" {
		t.Fatalf("reason = %#v", onlyPermissions.Items[0])
	}
}

func TestPedagogicalOrderPrioritizesFoundationsOverExamWeight(t *testing.T) {
	bundle := loadBundle(t)
	now := time.Date(2026, 10, 6, 10, 0, 0, 0, time.UTC)

	session, err := learning.BuildSession(learning.SessionInput{
		Now:                now,
		Bundle:             bundle,
		Projections:        map[string]learning.MasteryProjection{},
		ObjectiveReadiness: map[string]bool{"103.1": true},
		ConceptReadiness: map[string]bool{
			"lpic1.103.3.creation-copie-deplacement-suppression": true,
			"lpic1.103.1.path-et-resolution-de-commande": true,
		},
		ScopeObjectives:    []string{"102.3", "103.8"},
		Policy:             learning.DefaultSessionPolicy(),
	})
	if err != nil {
		t.Fatalf("BuildSession() error = %v", err)
	}
	if len(session.Items) != 1 {
		t.Fatalf("items = %#v, want one new concept", session.Items)
	}
	if got := session.Items[0].ObjectiveID; got != "103.8" {
		t.Fatalf("selected objective = %s, want 103.8: authored pedagogy comes before exam-weight density", got)
	}
}

func TestExam101ConceptOrderStartsWithShellThenFiles(t *testing.T) {
	bundle := loadBundle(t)
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	ready := map[string]bool{}
	projections := map[string]learning.MasteryProjection{}
	seen := map[string]bool{}
	order := []string{}

	for i := 0; i < 162; i++ {
		session, err := learning.BuildSession(learning.SessionInput{
			Now: now, Bundle: bundle, Projections: projections,
			ConceptReadiness: ready, ObjectiveReadiness: map[string]bool{},
			ScopeObjectives: bundle.Phase3.SelectedObjectives,
			Policy: learning.DefaultSessionPolicy(),
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(session.Items) != 1 || session.Items[0].Kind != learning.SessionNew {
			t.Fatalf("step %d: expected one new concept, got %#v", i+1, session.Items)
		}
		id := session.Items[0].ConceptID
		if seen[id] {
			t.Fatalf("step %d: duplicate concept %s", i+1, id)
		}
		seen[id] = true
		order = append(order, id)
		ready[id] = true
		projections[id] = learning.MasteryProjection{
			ConceptID: id, Stage: learning.StageGuided,
			SuccessfulExposure: 1, SuccessfulRecall: 1, SuccessfulGuided: 1,
			LastEvidenceAt: now, LastStageEvidenceAt: now,
		}
	}
	if order[0] != "lpic1.103.1.syntaxe-shell-et-sequences-de-commandes" ||
		order[1] != "lpic1.103.3.creation-copie-deplacement-suppression" {
		t.Fatalf("initial concepts = %#v; expected shell then cp/mv/mkdir/touch", order[:2])
	}
	fileIndex, processIndex := -1, -1
	for i, id := range order {
		if id == "lpic1.103.3.creation-copie-deplacement-suppression" {
			fileIndex = i
		}
		if id == "lpic1.103.5.processus-avant-arriere-plan" {
			processIndex = i
		}
	}
	if fileIndex < 0 || processIndex <= fileIndex {
		t.Fatalf("file manipulation should precede processes (file=%d, process=%d)", fileIndex, processIndex)
	}
	if len(seen) != 162 {
		t.Fatalf("visited %d of 162 Exam-101 concepts", len(seen))
	}
}

func TestConceptPrerequisiteNeedsFullCourseQuizAndLabReadiness(t *testing.T) {
	bundle := loadBundle(t)
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	shell := "lpic1.103.1.syntaxe-shell-et-sequences-de-commandes"
	files := "lpic1.103.3.creation-copie-deplacement-suppression"
	base := learning.SessionInput{
		Now: now, Bundle: bundle,
		Projections: map[string]learning.MasteryProjection{
			shell: {ConceptID: shell, Stage: learning.StageGuided,
				SuccessfulExposure: 1, SuccessfulRecall: 1, SuccessfulGuided: 1},
		},
		ConceptReadiness: map[string]bool{}, // partial evidence: not all 3 activities passed
		ScopeObjectives: []string{"103.1", "103.3"},
		Policy: learning.DefaultSessionPolicy(),
	}
	before, err := learning.BuildSession(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(before.Items) > 0 && before.Items[0].ConceptID == files {
		t.Fatalf("files should stay locked without full prerequisite evidence: %#v", before.Items)
	}
	base.ConceptReadiness[shell] = true
	after, err := learning.BuildSession(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Items) != 1 || after.Items[0].ConceptID != files {
		t.Fatalf("expected foundational file manipulation as next step, got %#v", after.Items)
	}
}

func TestExistingAdvancedProgressDoesNotHideMissingFileFoundation(t *testing.T) {
	bundle := loadBundle(t)
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	shell := "lpic1.103.1.syntaxe-shell-et-sequences-de-commandes"
	process := "lpic1.103.5.processus-avant-arriere-plan"
	files := "lpic1.103.3.creation-copie-deplacement-suppression"
	session, err := learning.BuildSession(learning.SessionInput{
		Now: now, Bundle: bundle, ConceptReadiness: map[string]bool{shell: true},
		ObjectiveReadiness: map[string]bool{},
		Projections: map[string]learning.MasteryProjection{
			shell: {ConceptID: shell, Stage: learning.StageGuided, SuccessfulExposure: 1,
				SuccessfulRecall: 1, SuccessfulGuided: 1},
			process: {ConceptID: process, Stage: learning.StageGuided, SuccessfulExposure: 1,
				SuccessfulRecall: 1, SuccessfulGuided: 1},
		},
		ScopeObjectives: bundle.Phase3.SelectedObjectives,
		Policy: learning.DefaultSessionPolicy(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(session.Items) != 1 || session.Items[0].ConceptID != files {
		t.Fatalf("backfill should teach basic files before advancing processes: %#v", session.Items)
	}
}
