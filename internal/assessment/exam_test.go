package assessment_test

import (
	"reflect"
	"testing"

	lpicdaily "github.com/Loe159/lpic-daily"
	"github.com/Loe159/lpic-daily/internal/assessment"
	"github.com/Loe159/lpic-daily/internal/content"
	"github.com/Loe159/lpic-daily/internal/curriculum"
)

func TestExam101QuestionsMatchOfficialObjectiveWeights(t *testing.T) {
	curriculumBundle, err := curriculum.Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("curriculum.Load() error = %v", err)
	}
	contentBundle, err := content.Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("content.Load() error = %v", err)
	}
	questions, err := assessment.ExamQuestions(curriculumBundle, contentBundle, "101")
	if err != nil {
		t.Fatalf("ExamQuestions() error = %v", err)
	}
	if len(questions) != 60 {
		t.Fatalf("Exam 101 questions = %d, want 60", len(questions))
	}

	counts := make(map[string]int)
	seenIDs := make(map[string]bool)
	application := 0
	recall := 0
	for _, question := range questions {
		if len(question.ObjectiveIDs) != 1 || len(question.ConceptIDs) != 1 {
			t.Fatalf("question %s is not concept/objective scoped", question.ID)
		}
		if question.Usage == "initial-assessment" {
			t.Fatalf("question %s leaked initial-assessment content into final simulation", question.ID)
		}
		if seenIDs[question.ID] {
			t.Fatalf("duplicate Exam 101 question %s", question.ID)
		}
		seenIDs[question.ID] = true
		counts[question.ObjectiveIDs[0]]++
		if question.EvidenceKindOnSuccess == "recall" {
			recall++
		}
		if len(question.ID) >= len(".q.autonomous-application") &&
			contains(question.ID, ".q.autonomous-application") {
			application++
		}
	}
	for _, objective := range curriculumBundle.Objectives.Objectives {
		if !objective.Active || objective.Exam != "101" {
			continue
		}
		if got := counts[objective.ID]; got != objective.Weight {
			t.Errorf("%s questions = %d, want official weight %d", objective.ID, got, objective.Weight)
		}
	}
	if recall == 0 || application == 0 {
		t.Fatalf("simulation mix recall=%d application=%d, want both", recall, application)
	}
}

func TestExam101QuestionSelectionIsDeterministic(t *testing.T) {
	curriculumBundle, err := curriculum.Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("curriculum.Load() error = %v", err)
	}
	contentBundle, err := content.Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("content.Load() error = %v", err)
	}
	first, err := assessment.ExamQuestions(curriculumBundle, contentBundle, "101")
	if err != nil {
		t.Fatalf("first ExamQuestions() error = %v", err)
	}
	second, err := assessment.ExamQuestions(curriculumBundle, contentBundle, "101")
	if err != nil {
		t.Fatalf("second ExamQuestions() error = %v", err)
	}
	firstIDs := make([]string, len(first))
	secondIDs := make([]string, len(second))
	for i := range first {
		firstIDs[i] = first[i].ID
		secondIDs[i] = second[i].ID
	}
	if !reflect.DeepEqual(firstIDs, secondIDs) {
		t.Fatalf("Exam 101 selection is not deterministic")
	}
}

func contains(value, needle string) bool {
	for i := 0; i+len(needle) <= len(value); i++ {
		if value[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
