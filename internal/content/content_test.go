package content

import (
	"slices"
	"testing"

	lpicdaily "github.com/Loe159/lpic-daily"
	"github.com/Loe159/lpic-daily/internal/curriculum"
)

func TestBuiltinPhase1ContentCoversEveryConcept(t *testing.T) {
	bundle, err := Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(bundle.Lessons) != 25 {
		t.Fatalf("lessons = %d, want 25 (22 introductions + 3 deepen)", len(bundle.Lessons))
	}
	if len(bundle.Questions) != 29 {
		t.Fatalf("questions = %d, want 29 (22 daily + 7 initial assessment)", len(bundle.Questions))
	}

	curriculumBundle, err := curriculum.Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("curriculum.Load() error = %v", err)
	}

	phase1Concepts := make([]string, 0, 22)
	for _, objectiveID := range curriculumBundle.Phase1.SelectedObjectives {
		phase1Concepts = append(phase1Concepts, curriculumBundle.Phase1.ObjectiveConcepts[objectiveID]...)
	}
	slices.Sort(phase1Concepts)

	lessonCoverage := make(map[string]int)
	introductions := make(map[string]int)
	for _, lesson := range bundle.Lessons {
		for _, conceptID := range lesson.ConceptIDs {
			lessonCoverage[conceptID]++
			if lesson.Stage == "introduce" && len(lesson.ConceptIDs) == 1 {
				introductions[conceptID]++
			}
		}
	}

	questionCoverage := make(map[string]int)
	for _, question := range bundle.Questions {
		if len(question.ConceptIDs) != 1 {
			t.Fatalf("question %s maps %d concepts, want exactly 1 in Phase 1", question.ID, len(question.ConceptIDs))
		}
		questionCoverage[question.ConceptIDs[0]]++
	}

	for _, conceptID := range phase1Concepts {
		if lessonCoverage[conceptID] == 0 {
			t.Errorf("concept %s has no lesson", conceptID)
		}
		if introductions[conceptID] != 1 {
			t.Errorf("concept %s focused introduction coverage = %d, want exactly 1", conceptID, introductions[conceptID])
		}
		if questionCoverage[conceptID] < 1 {
			t.Errorf("concept %s has no question coverage", conceptID)
		}
	}
}

func TestGradeDeterministicStrategies(t *testing.T) {
	exact := Question{Grading: Grading{
		Strategy:        "exact-text",
		AcceptedAnswers: []string{"export"},
		CaseSensitive:   false,
	}}
	pass, err := exact.Grade(Answer{Text: " Export\n"})
	if err != nil || !pass {
		t.Fatalf("exact Grade() = %v, %v", pass, err)
	}

	choice := Question{Grading: Grading{
		Strategy:          "choice-ids",
		AcceptedChoiceIDs: []string{"a", "b"},
	}}
	pass, err = choice.Grade(Answer{ChoiceIDs: []string{"b", "a"}})
	if err != nil || !pass {
		t.Fatalf("choice Grade() = %v, %v", pass, err)
	}
	pass, err = choice.Grade(Answer{ChoiceIDs: []string{"a", "a", "b"}})
	if err != nil || pass {
		t.Fatalf("duplicate choice Grade() = %v, %v, want false", pass, err)
	}

	ordering := Question{Grading: Grading{
		Strategy:      "ordered-ids",
		ExpectedOrder: []string{"first", "second"},
	}}
	pass, err = ordering.Grade(Answer{ChoiceIDs: []string{"second", "first"}})
	if err != nil || pass {
		t.Fatalf("ordering Grade() = %v, %v, want false", pass, err)
	}
}
