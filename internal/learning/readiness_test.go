package learning_test

import (
	"testing"

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
	if len(shellConcepts) != 7 {
		t.Fatalf("103.1 concepts = %d, want 7", len(shellConcepts))
	}

	projections := map[string]learning.MasteryProjection{}
	for _, concept := range shellConcepts[:5] {
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
		t.Fatal("103.1 should be ready with 5/7 concepts at recall")
	}

	projections[shellConcepts[4].ID] = learning.MasteryProjection{
		ConceptID: shellConcepts[4].ID,
		Stage:     learning.StageExposed,
	}
	got, err = learning.ObjectiveReadiness(bundle, projections, learning.DefaultReadinessPolicy())
	if err != nil {
		t.Fatalf("ObjectiveReadiness() error = %v", err)
	}
	if got["103.1"] {
		t.Fatal("103.1 should not be ready with only 4/7 concepts at recall")
	}
}
