package assessment_test

import (
	"context"
	"testing"
	"time"

	lpicdaily "github.com/Loe159/lpic-daily"
	"github.com/Loe159/lpic-daily/internal/assessment"
	"github.com/Loe159/lpic-daily/internal/content"
	"github.com/Loe159/lpic-daily/internal/curriculum"
	"github.com/Loe159/lpic-daily/internal/learning"
)

type evidenceMap map[string][]learning.EvidenceEvent

func (events evidenceMap) EvidenceForConcept(_ context.Context, conceptID string) ([]learning.EvidenceEvent, error) {
	return append([]learning.EvidenceEvent(nil), events[conceptID]...), nil
}

func TestInitialAssessmentQuestionsAreRecallOnlyAndCover1031(t *testing.T) {
	contentBundle, err := content.Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("content.Load() error = %v", err)
	}
	questions, err := assessment.Questions(contentBundle)
	if err != nil {
		t.Fatalf("Questions() error = %v", err)
	}
	if len(questions) != 7 {
		t.Fatalf("questions = %d, want 7", len(questions))
	}
	seen := map[string]bool{}
	for _, question := range questions {
		if question.EvidenceKindOnSuccess != "recall" {
			t.Fatalf("question %s evidence = %s, want recall", question.ID, question.EvidenceKindOnSuccess)
		}
		if len(question.ConceptIDs) != 1 {
			t.Fatalf("question %s concepts = %v", question.ID, question.ConceptIDs)
		}
		seen[question.ConceptIDs[0]] = true
	}
	if len(seen) != 7 {
		t.Fatalf("covered concepts = %d, want 7", len(seen))
	}
}

func TestRecallAssessmentCanSatisfyFoundationWithoutLessons(t *testing.T) {
	curriculumBundle, err := curriculum.Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("curriculum.Load() error = %v", err)
	}
	at := time.Date(2026, 9, 27, 4, 0, 0, 0, time.UTC)
	events := evidenceMap{}
	for index, conceptID := range curriculumBundle.Phase1.ObjectiveConcepts["103.1"] {
		if index >= 5 {
			break
		}
		events[conceptID] = []learning.EvidenceEvent{{
			EventID:          "assessment-" + conceptID,
			OccurredAt:       at,
			ConceptID:        conceptID,
			ObjectiveIDs:     []string{"103.1"},
			SourceItemID:     "assessment",
			ActivityKind:     learning.ActivityQuestion,
			EvidenceKind:     learning.EvidenceRecall,
			Result:           learning.ResultPass,
			HighestHintLevel: 0,
			SolutionRevealed: false,
			Distribution:     "generic",
			AttemptIndex:     1,
		}}
	}

	ready, err := assessment.FoundationReady(context.Background(), curriculumBundle, events)
	if err != nil {
		t.Fatalf("FoundationReady() error = %v", err)
	}
	if !ready {
		t.Fatal("foundation not ready after 5/7 recall concepts")
	}
	for _, conceptEvents := range events {
		for _, event := range conceptEvents {
			if event.ActivityKind == learning.ActivityLesson {
				t.Fatalf("assessment fabricated lesson evidence: %#v", event)
			}
		}
	}
}
