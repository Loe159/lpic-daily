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

	curriculumBundle, err := curriculum.Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("curriculum.Load() error = %v", err)
	}

	phase1Concepts := make([]string, 0, 22)
	phase1Set := make(map[string]struct{}, 22)
	for _, objectiveID := range curriculumBundle.Phase1.SelectedObjectives {
		for _, conceptID := range curriculumBundle.Phase1.ObjectiveConcepts[objectiveID] {
			phase1Concepts = append(phase1Concepts, conceptID)
			phase1Set[conceptID] = struct{}{}
		}
	}
	slices.Sort(phase1Concepts)

	lessonCoverage := make(map[string]int)
	introductions := make(map[string]int)
	for _, lesson := range bundle.Lessons {
		for _, conceptID := range lesson.ConceptIDs {
			if _, phase1 := phase1Set[conceptID]; !phase1 {
				continue
			}
			lessonCoverage[conceptID]++
			if lesson.Stage == "introduce" && len(lesson.ConceptIDs) == 1 {
				introductions[conceptID]++
			}
		}
	}

	questionCoverage := make(map[string]int)
	for _, question := range bundle.Questions {
		if question.Usage == "initial-assessment" {
			continue
		}
		for _, conceptID := range question.ConceptIDs {
			if _, phase1 := phase1Set[conceptID]; phase1 {
				questionCoverage[conceptID]++
			}
		}
	}

	for _, conceptID := range phase1Concepts {
		if lessonCoverage[conceptID] == 0 {
			t.Errorf("concept %s has no lesson", conceptID)
		}
		if introductions[conceptID] != 1 {
			t.Errorf("concept %s focused introduction coverage = %d, want exactly 1", conceptID, introductions[conceptID])
		}
		if questionCoverage[conceptID] < 1 {
			t.Errorf("concept %s has no daily non-assessment question coverage", conceptID)
		}
	}
}

func TestQuestionTypeCannotOverstateMasteryEvidence(t *testing.T) {
	curriculumBundle, err := curriculum.Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("curriculum.Load() error = %v", err)
	}
	contentBundle, err := Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	knownObjectives := make(map[string]struct{})
	for _, objective := range curriculumBundle.Objectives.Objectives {
		if objective.Active {
			knownObjectives[objective.ID] = struct{}{}
		}
	}
	knownConcepts := make(map[string]curriculum.Concept)
	for _, concept := range curriculumBundle.Concepts.Concepts {
		if concept.Active {
			knownConcepts[concept.ID] = concept
		}
	}

	var multipleChoice Question
	var textQuestion Question
	for _, question := range contentBundle.Questions {
		if multipleChoice.ID == "" && question.Type == "multiple-choice" {
			multipleChoice = question
		}
		if textQuestion.ID == "" && (question.Type == "free-recall" || question.Type == "fill-in") {
			textQuestion = question
		}
	}
	if multipleChoice.ID == "" || textQuestion.ID == "" {
		t.Fatal("builtin content must contain both choice and text questions")
	}

	multipleChoice.EvidenceKindOnSuccess = "recall"
	if err := validateQuestion(multipleChoice, knownObjectives, knownConcepts); err == nil {
		t.Fatal("multiple-choice question incorrectly accepted recall evidence")
	}

	textQuestion.EvidenceKindOnSuccess = "recognition"
	if err := validateQuestion(textQuestion, knownObjectives, knownConcepts); err == nil {
		t.Fatal("text question incorrectly accepted recognition evidence")
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
