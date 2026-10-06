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

func restrictStudyInputs(
	contentBundle *content.Bundle,
	labs []lab.Lab,
	objectiveIDs ...string,
) (*content.Bundle, []lab.Lab) {
	allowed := make(map[string]bool, len(objectiveIDs))
	for _, objectiveID := range objectiveIDs {
		allowed[objectiveID] = true
	}
	hasAllowedObjective := func(ids []string) bool {
		for _, id := range ids {
			if allowed[id] {
				return true
			}
		}
		return false
	}

	filtered := *contentBundle
	filtered.Lessons = nil
	for _, lesson := range contentBundle.Lessons {
		if hasAllowedObjective(lesson.ObjectiveIDs) {
			filtered.Lessons = append(filtered.Lessons, lesson)
		}
	}
	filtered.Questions = nil
	for _, question := range contentBundle.Questions {
		if hasAllowedObjective(question.ObjectiveIDs) {
			filtered.Questions = append(filtered.Questions, question)
		}
	}

	filteredLabs := make([]lab.Lab, 0, len(labs))
	for _, authored := range labs {
		if hasAllowedObjective(authored.Definition.ObjectiveIDs) {
			filteredLabs = append(filteredLabs, authored)
		}
	}
	return &filtered, filteredLabs
}

func withoutObjectiveLabs(labs []lab.Lab, objectiveID string) []lab.Lab {
	filtered := make([]lab.Lab, 0, len(labs))
	for _, authored := range labs {
		contains := false
		for _, candidate := range authored.Definition.ObjectiveIDs {
			if candidate == objectiveID {
				contains = true
				break
			}
		}
		if !contains {
			filtered = append(filtered, authored)
		}
	}
	return filtered
}

func TestFreshPlanStartsWithFocused1031Introduction(t *testing.T) {
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
	if len(item.LessonIDs) < 2 || len(item.QuestionIDs) < 3 {
		t.Fatalf("resolved item = %#v, want authored lessons plus generated question coverage", item)
	}
	if item.RecommendedLessonID != "lpic1.103.1.lesson.shell-sequences" {
		t.Fatalf("recommended lesson = %q", item.RecommendedLessonID)
	}
	if item.RecommendedQuestionID != "lpic1.103.1.syntaxe-shell-et-sequences-de-commandes.q.autonomous-recall" {
		t.Fatalf("recommended question = %q", item.RecommendedQuestionID)
	}
	if len(item.LabIDs) < 4 || item.RecommendedLabID != "lpic1.103.1.shell-environment-repair" {
		t.Fatalf("first shell syntax concept lab recommendation = %#v", item)
	}
}

func TestComplete1034ObjectiveBecomesSchedulable(t *testing.T) {
	curriculumBundle, contentBundle, labs := loadInputs(t)
	contentBundle, labs = restrictStudyInputs(contentBundle, labs, "103.1", "103.4")
	now := time.Date(2026, 10, 1, 14, 0, 0, 0, time.UTC)
	evidence := memoryEvidence{}

	for _, conceptID := range curriculumBundle.Phase1.ObjectiveConcepts["103.1"] {
		evidence[conceptID] = []learning.EvidenceEvent{
			{
				EventID:      "ready-" + conceptID,
				OccurredAt:   now.Add(-time.Minute),
				ConceptID:    conceptID,
				ObjectiveIDs: []string{"103.1"},
				SourceItemID: "test-recall",
				ActivityKind: learning.ActivityQuestion,
				EvidenceKind: learning.EvidenceRecall,
				Result:       learning.ResultPass,
				Distribution: "generic",
				AttemptIndex: 1,
			},
			{
				EventID:         "practice-" + conceptID,
				OccurredAt:      now,
				ConceptID:       conceptID,
				ObjectiveIDs:    []string{"103.1"},
				SourceItemID:    "test-practice",
				ActivityKind:    learning.ActivityLab,
				EvidenceKind:    learning.EvidenceGuidedPractice,
				Result:          learning.ResultPass,
				Distribution:    "generic",
				PracticeContext: "test-practice",
				AttemptIndex:    1,
			},
		}
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
	if len(plan.Items) != 1 {
		t.Fatalf("items = %#v, want one new 103.4 item", plan.Items)
	}
	item := plan.Items[0]
	if item.Kind != learning.SessionNew || item.ObjectiveID != "103.4" {
		t.Fatalf("item = %#v, want new 103.4 concept after 103.1 readiness", item)
	}
	if item.RecommendedLessonID == "" || item.RecommendedQuestionID == "" || item.RecommendedLabID == "" {
		t.Fatalf("103.4 item is not runnable through course, quiz, and lab: %#v", item)
	}
}

func TestQuestionRecommendationPrefersRecallBeforeOtherQuestions(t *testing.T) {
	curriculumBundle, contentBundle, labs := loadInputs(t)
	contentBundle, labs = restrictStudyInputs(contentBundle, labs, "103.1", "103.4")
	now := time.Date(2026, 10, 1, 15, 0, 0, 0, time.UTC)
	evidence := memoryEvidence{}

	for _, conceptID := range curriculumBundle.Phase1.ObjectiveConcepts["103.1"] {
		evidence[conceptID] = []learning.EvidenceEvent{
			{
				EventID:      "ready-" + conceptID,
				OccurredAt:   now.Add(-time.Minute),
				ConceptID:    conceptID,
				ObjectiveIDs: []string{"103.1"},
				SourceItemID: "test-recall",
				ActivityKind: learning.ActivityQuestion,
				EvidenceKind: learning.EvidenceRecall,
				Result:       learning.ResultPass,
				Distribution: "generic",
				AttemptIndex: 1,
			},
			{
				EventID:         "practice-" + conceptID,
				OccurredAt:      now,
				ConceptID:       conceptID,
				ObjectiveIDs:    []string{"103.1"},
				SourceItemID:    "test-practice",
				ActivityKind:    learning.ActivityLab,
				EvidenceKind:    learning.EvidenceGuidedPractice,
				Result:          learning.ResultPass,
				Distribution:    "generic",
				PracticeContext: "test-practice",
				AttemptIndex:    1,
			},
		}
	}

	conceptID := "lpic1.103.4.stdin-stdout-stderr-et-descripteurs"
	evidence[conceptID] = []learning.EvidenceEvent{
		{
			EventID:      "lesson-1034",
			OccurredAt:   now,
			ConceptID:    conceptID,
			ObjectiveIDs: []string{"103.4"},
			SourceItemID: "lpic1.103.4.lesson.file-descriptors",
			ActivityKind: learning.ActivityLesson,
			EvidenceKind: learning.EvidenceExposure,
			Result:       learning.ResultPass,
			Distribution: "generic",
			AttemptIndex: 1,
		},
		{
			EventID:      "question-stderr",
			OccurredAt:   now,
			ConceptID:    conceptID,
			ObjectiveIDs: []string{"103.4"},
			SourceItemID: "lpic1.103.4.q.stderr-fd",
			ActivityKind: learning.ActivityQuestion,
			EvidenceKind: learning.EvidenceRecall,
			Result:       learning.ResultFail,
			Distribution: "generic",
			AttemptIndex: 1,
		},
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
	if len(plan.Items) != 1 || plan.Items[0].ConceptID != conceptID {
		t.Fatalf("items = %#v, want immediate practice for %s", plan.Items, conceptID)
	}
	want := conceptID + ".q.autonomous-recall"
	if got := plan.Items[0].RecommendedQuestionID; got != want {
		t.Fatalf("recommended question = %q, want recall-first %q", got, want)
	}
}

func TestObjectiveWithoutPracticalLabStaysOutOfScheduler(t *testing.T) {
	curriculumBundle, contentBundle, labs := loadInputs(t)
	contentBundle, labs = restrictStudyInputs(contentBundle, labs, "103.1", "103.4")
	labs = withoutObjectiveLabs(labs, "103.4")
	now := time.Date(2026, 10, 1, 16, 0, 0, 0, time.UTC)
	evidence := memoryEvidence{}
	for _, conceptID := range curriculumBundle.Phase1.ObjectiveConcepts["103.1"] {
		evidence[conceptID] = []learning.EvidenceEvent{{
			EventID: "ready-" + conceptID, OccurredAt: now, ConceptID: conceptID,
			ObjectiveIDs: []string{"103.1"}, SourceItemID: "test-recall",
			ActivityKind: learning.ActivityQuestion, EvidenceKind: learning.EvidenceRecall,
			Result: learning.ResultPass, Distribution: "generic", AttemptIndex: 1,
		}}
	}

	plan, err := study.BuildPlan(context.Background(), study.PlanInput{
		Now: now, Curriculum: curriculumBundle, Content: contentBundle, Labs: labs,
		Evidence: evidence, Policy: learning.DefaultSessionPolicy(),
	})
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	for _, item := range plan.Items {
		if item.ObjectiveID == "103.4" {
			t.Fatalf("103.4 scheduled without practical lab coverage: %#v", plan.Items)
		}
	}
}

func TestIncompleteFocusedIntroductionKeepsObjectiveOutOfScheduler(t *testing.T) {
	curriculumBundle, contentBundle, labs := loadInputs(t)
	contentBundle, labs = restrictStudyInputs(contentBundle, labs, "103.1", "103.4")
	now := time.Date(2026, 10, 1, 16, 0, 0, 0, time.UTC)
	evidence := memoryEvidence{}

	for _, conceptID := range curriculumBundle.Phase1.ObjectiveConcepts["103.1"] {
		evidence[conceptID] = []learning.EvidenceEvent{
			{
				EventID:      "ready-" + conceptID,
				OccurredAt:   now.Add(-time.Minute),
				ConceptID:    conceptID,
				ObjectiveIDs: []string{"103.1"},
				SourceItemID: "test-recall",
				ActivityKind: learning.ActivityQuestion,
				EvidenceKind: learning.EvidenceRecall,
				Result:       learning.ResultPass,
				Distribution: "generic",
				AttemptIndex: 1,
			},
			{
				EventID:         "practice-" + conceptID,
				OccurredAt:      now,
				ConceptID:       conceptID,
				ObjectiveIDs:    []string{"103.1"},
				SourceItemID:    "test-practice",
				ActivityKind:    learning.ActivityLab,
				EvidenceKind:    learning.EvidenceGuidedPractice,
				Result:          learning.ResultPass,
				Distribution:    "generic",
				PracticeContext: "test-practice",
				AttemptIndex:    1,
			},
		}
	}

	mutated := *contentBundle
	mutated.Lessons = append([]content.Lesson(nil), contentBundle.Lessons...)
	for index := range mutated.Lessons {
		if mutated.Lessons[index].ID == "lpic1.103.4.lesson.redirection-order" {
			mutated.Lessons[index].Stage = "deepen"
		}
	}

	plan, err := study.BuildPlan(context.Background(), study.PlanInput{
		Now:        now,
		Curriculum: curriculumBundle,
		Content:    &mutated,
		Labs:       labs,
		Evidence:   evidence,
		Policy:     learning.DefaultSessionPolicy(),
	})
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	for _, item := range plan.Items {
		if item.ObjectiveID == "103.4" {
			t.Fatalf("103.4 scheduled without focused introduction coverage: %#v", plan.Items)
		}
	}
}

func TestDuplicateFocusedIntroductionKeepsObjectiveOutOfScheduler(t *testing.T) {
	curriculumBundle, contentBundle, labs := loadInputs(t)
	contentBundle, labs = restrictStudyInputs(contentBundle, labs, "103.1", "103.4")
	now := time.Date(2026, 10, 1, 16, 30, 0, 0, time.UTC)
	evidence := memoryEvidence{}

	for _, conceptID := range curriculumBundle.Phase1.ObjectiveConcepts["103.1"] {
		evidence[conceptID] = []learning.EvidenceEvent{
			{
				EventID:      "ready-" + conceptID,
				OccurredAt:   now.Add(-time.Minute),
				ConceptID:    conceptID,
				ObjectiveIDs: []string{"103.1"},
				SourceItemID: "test-recall",
				ActivityKind: learning.ActivityQuestion,
				EvidenceKind: learning.EvidenceRecall,
				Result:       learning.ResultPass,
				Distribution: "generic",
				AttemptIndex: 1,
			},
			{
				EventID:         "practice-" + conceptID,
				OccurredAt:      now,
				ConceptID:       conceptID,
				ObjectiveIDs:    []string{"103.1"},
				SourceItemID:    "test-practice",
				ActivityKind:    learning.ActivityLab,
				EvidenceKind:    learning.EvidenceGuidedPractice,
				Result:          learning.ResultPass,
				Distribution:    "generic",
				PracticeContext: "test-practice",
				AttemptIndex:    1,
			},
		}
	}

	mutated := *contentBundle
	mutated.Lessons = append([]content.Lesson(nil), contentBundle.Lessons...)
	for _, lesson := range contentBundle.Lessons {
		if lesson.ID == "lpic1.103.4.lesson.redirection-order" {
			duplicate := lesson
			duplicate.ID = "lpic1.103.4.lesson.redirection-order-duplicate"
			mutated.Lessons = append(mutated.Lessons, duplicate)
			break
		}
	}

	plan, err := study.BuildPlan(context.Background(), study.PlanInput{
		Now:        now,
		Curriculum: curriculumBundle,
		Content:    &mutated,
		Labs:       labs,
		Evidence:   evidence,
		Policy:     learning.DefaultSessionPolicy(),
	})
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	for _, item := range plan.Items {
		if item.ObjectiveID == "103.4" {
			t.Fatalf("103.4 scheduled with duplicate focused introduction: %#v", plan.Items)
		}
	}
}

func TestPlanUsesStoredRecallToCreateDueReviewWithoutReplayingLesson(t *testing.T) {
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
	if plan.Items[0].RecommendedLessonID != "" {
		t.Fatalf("review unexpectedly recommends lesson %q", plan.Items[0].RecommendedLessonID)
	}
	wantQuestion := conceptID + ".q.autonomous-recall"
	if plan.Items[0].RecommendedQuestionID != wantQuestion {
		t.Fatalf("review question = %q, want least-attempted %q", plan.Items[0].RecommendedQuestionID, wantQuestion)
	}
}

func TestRecognitionOnlyConceptPrefersLabOnDueReview(t *testing.T) {
	curriculumBundle, contentBundle, labs := loadInputs(t)
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	conceptID := "lpic1.103.1.syntaxe-shell-et-sequences-de-commandes"
	evidence := memoryEvidence{
		conceptID: {
			{
				EventID:      "lesson",
				OccurredAt:   now.Add(-48 * time.Hour),
				ConceptID:    conceptID,
				ObjectiveIDs: []string{"103.1"},
				SourceItemID: "lpic1.103.1.lesson.shell-sequences",
				ActivityKind: learning.ActivityLesson,
				EvidenceKind: learning.EvidenceExposure,
				Result:       learning.ResultPass,
				Distribution: "generic",
				AttemptIndex: 1,
			},
			{
				EventID:      "recognition",
				OccurredAt:   now.Add(-47 * time.Hour),
				ConceptID:    conceptID,
				ObjectiveIDs: []string{"103.1"},
				SourceItemID: "lpic1.103.1.q.sequence-and",
				ActivityKind: learning.ActivityQuestion,
				EvidenceKind: learning.EvidenceRecognition,
				Result:       learning.ResultPass,
				Distribution: "generic",
				AttemptIndex: 1,
			},
		},
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
	for _, item := range plan.Items {
		if item.ConceptID != conceptID {
			continue
		}
		if item.Kind != learning.SessionReview || !item.PreferLab ||
			item.RecommendedLabID != "lpic1.103.1.shell-environment-repair" {
			t.Fatalf("recognized exposed review = %#v, want bespoke lab-preferred review", item)
		}
		return
	}
	t.Fatalf("plan = %#v, want due review for %s", plan.Items, conceptID)
}

func TestQuickPolicyLimitsReviews(t *testing.T) {
	curriculumBundle, contentBundle, labs := loadInputs(t)
	now := time.Date(2026, 9, 26, 20, 0, 0, 0, time.UTC)
	evidence := memoryEvidence{}

	for _, conceptID := range curriculumBundle.Phase1.ObjectiveConcepts["103.1"][:3] {
		evidence[conceptID] = []learning.EvidenceEvent{
			{
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
			},
			{
				EventID:         "practice-" + conceptID,
				OccurredAt:      now.Add(-9 * 24 * time.Hour),
				ConceptID:       conceptID,
				ObjectiveIDs:    []string{"103.1"},
				SourceItemID:    "test-practice",
				ActivityKind:    learning.ActivityLab,
				EvidenceKind:    learning.EvidenceGuidedPractice,
				Result:          learning.ResultPass,
				Distribution:    "generic",
				PracticeContext: "test-practice",
				AttemptIndex:    1,
			},
		}
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

func TestExposedConceptRecommendsQuestionNotLesson(t *testing.T) {
	curriculumBundle, contentBundle, labs := loadInputs(t)
	now := time.Date(2026, 9, 27, 2, 0, 0, 0, time.UTC)
	conceptID := "lpic1.103.1.syntaxe-shell-et-sequences-de-commandes"

	evidence := memoryEvidence{
		conceptID: {{
			EventID:          "lesson-1",
			OccurredAt:       now,
			ConceptID:        conceptID,
			ObjectiveIDs:     []string{"103.1"},
			SourceItemID:     "lpic1.103.1.lesson.shell-sequences",
			ActivityKind:     learning.ActivityLesson,
			EvidenceKind:     learning.EvidenceExposure,
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
	if len(plan.Items) != 1 || plan.Items[0].Kind != learning.SessionPractice {
		t.Fatalf("items = %#v, want immediate practice", plan.Items)
	}
	if plan.Items[0].RecommendedLessonID != "" {
		t.Fatalf("practice unexpectedly recommends lesson %q", plan.Items[0].RecommendedLessonID)
	}
	wantQuestion := conceptID + ".q.autonomous-recall"
	if plan.Items[0].RecommendedQuestionID != wantQuestion {
		t.Fatalf("practice question = %q, want %q", plan.Items[0].RecommendedQuestionID, wantQuestion)
	}
}

func TestIndependentConceptRecommendsUnusedTransferLab(t *testing.T) {
	curriculumBundle, contentBundle, labs := loadInputs(t)
	now := time.Date(2026, 9, 28, 16, 0, 0, 0, time.UTC)
	conceptID := "lpic1.103.1.path-et-resolution-de-commande"
	evidence := memoryEvidence{
		conceptID: {{
			EventID:         "independent-shell",
			OccurredAt:      now.Add(-8 * 24 * time.Hour),
			ConceptID:       conceptID,
			ObjectiveIDs:    []string{"103.1"},
			SourceItemID:    "lpic1.103.1.shell-environment-repair",
			ActivityKind:    learning.ActivityLab,
			EvidenceKind:    learning.EvidenceIndependentPractice,
			Result:          learning.ResultPass,
			Distribution:    "fedora",
			PracticeContext: "shell-env-repair",
			AttemptIndex:    1,
		}},
	}
	policy := learning.DefaultSessionPolicy()
	policy.MaxNewConcepts = 0
	plan, err := study.BuildPlan(context.Background(), study.PlanInput{
		Now: now, Curriculum: curriculumBundle, Content: contentBundle, Labs: labs,
		Evidence: evidence, Policy: policy,
	})
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	if len(plan.Items) != 1 || plan.Items[0].ConceptID != conceptID {
		t.Fatalf("items = %#v, want due independent review", plan.Items)
	}
	if got := plan.Items[0].RecommendedLabID; got != "lpic1.103.1.transfer-shell-handoff" {
		t.Fatalf("recommended lab = %q, want unused transfer context", got)
	}
}

func TestBuildPlanDefaultsIntervalsWithoutOverwritingCustomLimits(t *testing.T) {
	curriculumBundle, contentBundle, labs := loadInputs(t)
	now := time.Date(2026, 9, 28, 18, 0, 0, 0, time.UTC)
	evidence := memoryEvidence{}

	for _, conceptID := range curriculumBundle.Phase1.ObjectiveConcepts["103.1"][:3] {
		evidence[conceptID] = []learning.EvidenceEvent{
			{
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
			},
			{
				EventID:         "practice-" + conceptID,
				OccurredAt:      now.Add(-9 * 24 * time.Hour),
				ConceptID:       conceptID,
				ObjectiveIDs:    []string{"103.1"},
				SourceItemID:    "test-practice",
				ActivityKind:    learning.ActivityLab,
				EvidenceKind:    learning.EvidenceGuidedPractice,
				Result:          learning.ResultPass,
				Distribution:    "generic",
				PracticeContext: "test-practice",
				AttemptIndex:    1,
			},
		}
	}

	plan, err := study.BuildPlan(context.Background(), study.PlanInput{
		Now: now, Curriculum: curriculumBundle, Content: contentBundle, Labs: labs,
		Evidence: evidence,
		Policy: learning.SessionPolicy{
			MaxReviews:     1,
			MaxNewConcepts: 0,
		},
	})
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	if len(plan.Items) != 1 || plan.Items[0].Kind != learning.SessionReview {
		t.Fatalf("items = %#v, want exactly one review with default intervals", plan.Items)
	}
}

func TestIndependentConceptSkipsUnusedLabIDInAlreadyUsedPracticeContext(t *testing.T) {
	curriculumBundle, contentBundle, labs := loadInputs(t)
	now := time.Date(2026, 9, 28, 16, 0, 0, 0, time.UTC)
	conceptID := "lpic1.103.1.path-et-resolution-de-commande"

	var superficial lab.Lab
	for _, authored := range labs {
		if authored.Definition.ID == "lpic1.103.1.shell-environment-repair" {
			superficial = authored
			break
		}
	}
	if superficial.Definition.ID == "" {
		t.Fatal("shell-environment-repair lab not found")
	}
	superficial.Definition.ID = "lpic1.103.1.aaa-superficial-variant"
	labs = append(labs, superficial)

	evidence := memoryEvidence{
		conceptID: {{
			EventID:         "independent-shell",
			OccurredAt:      now.Add(-8 * 24 * time.Hour),
			ConceptID:       conceptID,
			ObjectiveIDs:    []string{"103.1"},
			SourceItemID:    "lpic1.103.1.shell-environment-repair",
			ActivityKind:    learning.ActivityLab,
			EvidenceKind:    learning.EvidenceIndependentPractice,
			Result:          learning.ResultPass,
			Distribution:    "fedora",
			PracticeContext: "shell-env-repair",
			AttemptIndex:    1,
		}},
	}
	policy := learning.DefaultSessionPolicy()
	policy.MaxNewConcepts = 0
	plan, err := study.BuildPlan(context.Background(), study.PlanInput{
		Now: now, Curriculum: curriculumBundle, Content: contentBundle, Labs: labs,
		Evidence: evidence, Policy: policy,
	})
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	if len(plan.Items) != 1 {
		t.Fatalf("items = %#v, want one due review", plan.Items)
	}
	if got := plan.Items[0].RecommendedLabID; got != "lpic1.103.1.transfer-shell-handoff" {
		t.Fatalf("recommended lab = %q, want materially different practice context", got)
	}
}

func TestCompletedExam101CanProgressIntoExam102(t *testing.T) {
	curriculumBundle, contentBundle, labs := loadInputs(t)
	now := time.Date(2026, 10, 5, 18, 0, 0, 0, time.UTC)
	evidence := memoryEvidence{}

	objectiveExam := make(map[string]string)
	for _, objective := range curriculumBundle.Objectives.Objectives {
		if objective.Active {
			objectiveExam[objective.ID] = objective.Exam
		}
	}
	for _, concept := range curriculumBundle.Concepts.Concepts {
		if !concept.Active || objectiveExam[concept.ObjectiveID] != "101" {
			continue
		}
		evidence[concept.ID] = []learning.EvidenceEvent{
			{
				EventID:      "exam101-ready-" + concept.ID,
				OccurredAt:   now.Add(-time.Minute),
				ConceptID:    concept.ID,
				ObjectiveIDs: []string{concept.ObjectiveID},
				SourceItemID: "exam101-ready",
				ActivityKind: learning.ActivityQuestion,
				EvidenceKind: learning.EvidenceRecall,
				Result:       learning.ResultPass,
				Distribution: "generic",
				AttemptIndex: 1,
			},
			{
				EventID:         "exam101-practice-" + concept.ID,
				OccurredAt:      now,
				ConceptID:       concept.ID,
				ObjectiveIDs:    []string{concept.ObjectiveID},
				SourceItemID:    "exam101-practice",
				ActivityKind:    learning.ActivityLab,
				EvidenceKind:    learning.EvidenceGuidedPractice,
				Result:          learning.ResultPass,
				Distribution:    "generic",
				PracticeContext: "exam101-complete",
				AttemptIndex:    1,
			},
		}
	}

	plan, err := study.BuildPlan(context.Background(), study.PlanInput{
		Now: now, Curriculum: curriculumBundle, Content: contentBundle, Labs: labs,
		Evidence: evidence, Policy: learning.DefaultSessionPolicy(),
	})
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	if len(plan.Items) == 0 {
		t.Fatal("plan is empty after completing Exam 101; want an Exam 102 concept")
	}
	if got := objectiveExam[plan.Items[0].ObjectiveID]; got != "102" {
		t.Fatalf("first new objective = %s (exam %s), want Exam 102", plan.Items[0].ObjectiveID, got)
	}
}

func TestRecallSuccessPrefersLabBeforeNextConcept(t *testing.T) {
	curriculumBundle, contentBundle, labs := loadInputs(t)
	now := time.Date(2026, 10, 6, 10, 0, 0, 0, time.UTC)
	conceptID := "lpic1.103.1.syntaxe-shell-et-sequences-de-commandes"
	evidence := memoryEvidence{
		conceptID: {
			{
				EventID:      "lesson",
				OccurredAt:   now.Add(-2 * time.Minute),
				ConceptID:    conceptID,
				ObjectiveIDs: []string{"103.1"},
				SourceItemID: "lpic1.103.1.lesson.shell-sequences",
				ActivityKind: learning.ActivityLesson,
				EvidenceKind: learning.EvidenceExposure,
				Result:       learning.ResultPass,
				Distribution: "generic",
				AttemptIndex: 1,
			},
			{
				EventID:      "recall",
				OccurredAt:   now.Add(-time.Minute),
				ConceptID:    conceptID,
				ObjectiveIDs: []string{"103.1"},
				SourceItemID: conceptID + ".q.autonomous-recall",
				ActivityKind: learning.ActivityQuestion,
				EvidenceKind: learning.EvidenceRecall,
				Result:       learning.ResultPass,
				Distribution: "generic",
				AttemptIndex: 1,
			},
		},
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
	if len(plan.Items) == 0 {
		t.Fatal("expected practical consolidation item")
	}
	item := plan.Items[0]
	if item.ConceptID != conceptID || item.Kind != learning.SessionPractice || !item.PreferLab || item.RecommendedLabID == "" {
		t.Fatalf("item = %#v, want lab-preferred consolidation after recall", item)
	}
}
