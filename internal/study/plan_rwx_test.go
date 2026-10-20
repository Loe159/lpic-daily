package study_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Loe159/lpic-daily/internal/learning"
	"github.com/Loe159/lpic-daily/internal/study"
)

func TestRWXConceptGetsRealFocusedLabBeforeLater1045Lessons(t *testing.T) {
	curriculumBundle, contentBundle, labs := loadInputs(t)
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	rwx := "lpic1.104.5.permissions-rwx-fichier-dossier"
	focused := "lpic1.104.5.rwx-dispatch-access"
	composite := "lpic1.104.5.shared-dropbox"
	evidence := memoryEvidence{
		rwx: {
			{EventID: "rwx-lesson", OccurredAt: now, ConceptID: rwx,
				ObjectiveIDs: []string{"104.5"}, SourceItemID: "rwx-course",
				ActivityKind: learning.ActivityLesson, EvidenceKind: learning.EvidenceExposure,
				Result: learning.ResultPass, Distribution: "generic", AttemptIndex: 1},
			{EventID: "rwx-quiz", OccurredAt: now, ConceptID: rwx,
				ObjectiveIDs: []string{"104.5"}, SourceItemID: "rwx-quiz",
				ActivityKind: learning.ActivityQuestion, EvidenceKind: learning.EvidenceRecall,
				Result: learning.ResultPass, Distribution: "generic", AttemptIndex: 1},
		},
	}
	policy := learning.DefaultSessionPolicy()
	policy.MaxNewConcepts = 0
	build := func() study.Plan {
		t.Helper()
		plan, err := study.BuildPlan(context.Background(), study.PlanInput{
			Now: now, Curriculum: curriculumBundle, Content: contentBundle,
			Labs: labs, Evidence: evidence, Policy: policy,
		})
		if err != nil {
			t.Fatalf("BuildPlan(): %v", err)
		}
		return plan
	}
	plan := build()
	if len(plan.Items) != 1 || plan.Items[0].ConceptID != rwx {
		t.Fatalf("want rwx practice immediately after course+quiz: %#v", plan.Items)
	}
	item := plan.Items[0]
	if item.RecommendedLabID != focused {
		t.Fatalf("recommended lab = %q; want focused, not generated or composite", item.RecommendedLabID)
	}
	for _, labID := range item.LabIDs {
		if strings.Contains(labID, ".standalone-") || labID == composite {
			t.Fatalf("unready or generated lab was exposed for rwx: %s", labID)
		}
	}

	// The complete team-sharing scenario becomes available once its eight
	// distinct concepts have passed their corresponding courses and quizzes.
	teachLabPrerequisites(t, evidence, labs, composite, rwx, now)
	plan = build()
	if len(plan.Items) == 0 {
		t.Fatal("expected a continued rwx practice item")
	}
	found := false
	for _, item := range plan.Items {
		if item.ConceptID != rwx {
			continue
		}
		for _, labID := range item.LabIDs {
			found = found || labID == composite
			if strings.Contains(labID, ".standalone-") {
				t.Fatalf("generated lab leaked after composite unlocked: %s", labID)
			}
		}
	}
	if !found {
		t.Fatalf("team-sharing scenario still locked after all concepts studied: %#v", plan.Items)
	}
}
