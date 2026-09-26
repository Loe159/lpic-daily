package study_test

import (
	"context"
	"testing"
	"time"

	lpicdaily "github.com/Loe159/lpic-daily"
	"github.com/Loe159/lpic-daily/internal/content"
	"github.com/Loe159/lpic-daily/internal/curriculum"
	"github.com/Loe159/lpic-daily/internal/lab"
	"github.com/Loe159/lpic-daily/internal/learning"
	"github.com/Loe159/lpic-daily/internal/study"
)

type memoryEvidence map[string][]learning.EvidenceEvent

func (memory memoryEvidence) EvidenceForConcept(_ context.Context, conceptID string) ([]learning.EvidenceEvent, error) {
	return append([]learning.EvidenceEvent(nil), memory[conceptID]...), nil
}

func loadInputs(t *testing.T) (*curriculum.Bundle, *content.Bundle, []lab.Lab) {
	t.Helper()
	curriculumBundle, err := curriculum.Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("curriculum.Load() error = %v", err)
	}
	contentBundle, err := content.Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("content.Load() error = %v", err)
	}
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("lab.LoadAll() error = %v", err)
	}
	return curriculumBundle, contentBundle, labs
}

func TestFreshPlanStartsWithFirst1031ConceptAndResolvesArtifacts(t *testing.T) {
	curriculumBundle, contentBundle, labs := loadInputs(t)
	now := time.Date(2026, 9, 26, 20, 0, 0, 0, time.UTC)

	plan, err := study.BuildPlan(context.Background(), study.PlanInput{
		Now:        now,
		Curriculum: curriculumBundle,
		Content:    contentBundle,
		Labs:       labs,
		Evidence:   memoryEvidence{},
		Policy:     learning.DefaultSessionPolicy(),
	})
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	if len(plan.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(plan.Items))
	}
	item := plan.Items[0]
	if item.Kind != learning.SessionNew || item.ObjectiveID != "103.1" {
		t.Fatalf("item = %#v, want new 103.1 concept", item)
	}
	if item.ConceptID != "lpic1.103.1.syntaxe-shell-et-sequences-de-commandes" {
		t.Fatalf("concept = %s", item.ConceptID)
	}
	if len(item.LessonIDs) != 1 || len(item.QuestionIDs) != 1 {
		t.Fatalf("resolved item = %#v, want lesson + question", item)
	}
	if len(item.LabIDs) != 0 {
		t.Fatalf("first shell syntax concept unexpectedly has lab coverage: %v", item.LabIDs)
	}
}

func TestPlanUsesStoredRecallToCreateDueReview(t *testing.T) {
	curriculumBundle, contentBundle, labs := loadInputs(t)
	now := time.Date(2026, 9, 26, 20, 0, 0, 0, time.UTC)
	conceptID := "lpic1.103.1.syntaxe-shell-et-sequences-de-commandes"

	evidence := memoryEvidence{
		conceptID: {{
			EventID:          "recall-1",
			OccurredAt:       now.Add(-4 * 24 * time.Hour),
			ConceptID:        conceptID,
			ObjectiveIDs:     []string{"103.1"},
			SourceItemID:     "lpic1.103.1.q.sequence-and",
			ActivityKind:     learning.ActivityQuestion,
			EvidenceKind:     learning.EvidenceRecall,
			Result:           learning.ResultPass,
			HighestHintLevel: 0,
			SolutionRevealed: false,
			Distribution:     "generic",
			AttemptIndex:     1,
		}},
	}

	plan, err := study.BuildPlan(context.Background(), study.PlanInput{
		Now:        now,
		Curriculum: curriculumBundle,
		Content:    contentBundle,
		Labs:       labs,
		Evidence:   evidence,
		Policy:     learning.DefaultSessionPolicy(),
	})
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	if len(plan.Items) < 2 {
		t.Fatalf("items = %#v, want review then new concept", plan.Items)
	}
	if plan.Items[0].Kind != learning.SessionReview || plan.Items[0].ConceptID != conceptID {
		t.Fatalf("first item = %#v, want due review", plan.Items[0])
	}
	if plan.Items[0].MasteryStage != learning.StageRecall {
		t.Fatalf("stage = %s, want recall", plan.Items[0].MasteryStage)
	}
}

func TestQuickPolicyLimitsReviews(t *testing.T) {
	curriculumBundle, contentBundle, labs := loadInputs(t)
	now := time.Date(2026, 9, 26, 20, 0, 0, 0, time.UTC)
	evidence := memoryEvidence{}

	for _, conceptID := range curriculumBundle.Phase1.ObjectiveConcepts["103.1"][:3] {
		evidence[conceptID] = []learning.EvidenceEvent{{
			EventID:          "event-" + conceptID,
			OccurredAt:       now.Add(-10 * 24 * time.Hour),
			ConceptID:        conceptID,
			ObjectiveIDs:     []string{"103.1"},
			SourceItemID:     "test-question",
			ActivityKind:     learning.ActivityQuestion,
			EvidenceKind:     learning.EvidenceRecall,
			Result:           learning.ResultPass,
			HighestHintLevel: 0,
			SolutionRevealed: false,
			Distribution:     "generic",
			AttemptIndex:     1,
		}}
	}

	policy := learning.DefaultSessionPolicy()
	policy.MaxReviews = 1
	policy.MaxNewConcepts = 0
	plan, err := study.BuildPlan(context.Background(), study.PlanInput{
		Now:        now,
		Curriculum: curriculumBundle,
		Content:    contentBundle,
		Labs:       labs,
		Evidence:   evidence,
		Policy:     policy,
	})
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	if len(plan.Items) != 1 || plan.Items[0].Kind != learning.SessionReview {
		t.Fatalf("items = %#v, want exactly one review", plan.Items)
	}
}
