package learning_test

import (
	"fmt"
	"slices"
	"testing"
	"time"

	lpicdaily "github.com/Loe159/lpic-daily"
	"github.com/Loe159/lpic-daily/internal/curriculum"
	"github.com/Loe159/lpic-daily/internal/learning"
)

func TestObjectiveReadinessUsesConceptFractionNotCompletion(t *testing.T) {
	bundle, err := curriculum.Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	var shellConcepts []curriculum.Concept
	for _, concept := range bundle.Concepts.Concepts {
		if concept.ObjectiveID == "103.1" {
			shellConcepts = append(shellConcepts, concept)
		}
	}
	if len(shellConcepts) != 12 {
		t.Fatalf("103.1 concepts = %d, want 12", len(shellConcepts))
	}

	projections := map[string]learning.MasteryProjection{}
	for _, concept := range shellConcepts[:9] {
		projections[concept.ID] = learning.MasteryProjection{
			ConceptID: concept.ID,
			Stage:     learning.StageRecall,
		}
	}

	got, err := learning.ObjectiveReadiness(bundle, projections, learning.DefaultReadinessPolicy())
	if err != nil {
		t.Fatalf("ObjectiveReadiness() error = %v", err)
	}
	if !got["103.1"] {
		t.Fatal("103.1 should be ready with 9/12 concepts at recall")
	}

	projections[shellConcepts[8].ID] = learning.MasteryProjection{
		ConceptID: shellConcepts[8].ID,
		Stage:     learning.StageExposed,
	}
	got, err = learning.ObjectiveReadiness(bundle, projections, learning.DefaultReadinessPolicy())
	if err != nil {
		t.Fatalf("ObjectiveReadiness() error = %v", err)
	}
	if got["103.1"] {
		t.Fatal("103.1 should not be ready with only 8/12 concepts at recall")
	}
}

func TestInitialAssessmentCanUnlockPrerequisiteWithoutLessonEvidence(t *testing.T) {
	bundle, err := curriculum.Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	var shellConcepts []curriculum.Concept
	for _, concept := range bundle.Concepts.Concepts {
		if concept.ObjectiveID == "103.1" {
			shellConcepts = append(shellConcepts, concept)
		}
	}
	if len(shellConcepts) != 12 {
		t.Fatalf("103.1 concepts = %d, want 12", len(shellConcepts))
	}

	at := time.Date(2026, 9, 27, 3, 0, 0, 0, time.UTC)
	projections := make(map[string]learning.MasteryProjection)
	for index, concept := range shellConcepts[:9] {
		event := learning.EvidenceEvent{
			EventID:          fmt.Sprintf("assessment-%d", index+1),
			OccurredAt:       at.Add(time.Duration(index) * time.Minute),
			ConceptID:        concept.ID,
			ObjectiveIDs:     []string{"103.1"},
			SourceItemID:     "initial-assessment.103.1",
			ActivityKind:     learning.ActivityQuestion,
			EvidenceKind:     learning.EvidenceRecall,
			Result:           learning.ResultPass,
			HighestHintLevel: 0,
			SolutionRevealed: false,
			Distribution:     "generic",
			AttemptIndex:     1,
		}
		projection, err := learning.ProjectMastery(
			concept.ID,
			[]learning.EvidenceEvent{event},
			learning.DefaultProjectionPolicy(),
		)
		if err != nil {
			t.Fatalf("ProjectMastery(%s) error = %v", concept.ID, err)
		}
		if projection.SuccessfulExposure != 0 {
			t.Fatalf("assessment fabricated lesson exposure for %s: %#v", concept.ID, projection)
		}
		projections[concept.ID] = projection
	}

	readiness, err := learning.ObjectiveReadiness(
		bundle,
		projections,
		learning.DefaultReadinessPolicy(),
	)
	if err != nil {
		t.Fatalf("ObjectiveReadiness() error = %v", err)
	}
	if !readiness["103.1"] {
		t.Fatal("103.1 should become ready from sufficient recall assessment evidence")
	}

	eligible := learning.EligibleObjectives(bundle.Prerequisites, readiness)
	if !slices.Contains(eligible, "103.5") {
		t.Fatalf("assessment-backed 103.1 readiness did not unlock 103.5: %v", eligible)
	}
}
