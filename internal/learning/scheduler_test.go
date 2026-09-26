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
				ConceptID:      concept.ID,
				Stage:          learning.StageRecall,
				LastEvidenceAt: now,
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
			ConceptID:      shellConcept,
			Stage:          learning.StageRecall,
			LastEvidenceAt: now.Add(-10 * 24 * time.Hour),
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
