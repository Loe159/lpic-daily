package content

import (
	"slices"
	"testing"

	lpicdaily "github.com/Loe159/lpic-daily"
)

func TestLoadBuiltin1031Content(t *testing.T) {
	bundle, err := Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(bundle.Lessons) != 1 {
		t.Fatalf("lessons = %d, want 1", len(bundle.Lessons))
	}
	if len(bundle.Questions) != 7 {
		t.Fatalf("questions = %d, want 7", len(bundle.Questions))
	}

	lesson := bundle.Lessons[0]
	if lesson.ID != "lpic1.103.1.lesson.command-line-foundations" {
		t.Fatalf("lesson id = %q", lesson.ID)
	}
	if len(lesson.ConceptIDs) != 7 {
		t.Fatalf("lesson concepts = %d, want 7", len(lesson.ConceptIDs))
	}

	questionConcepts := make([]string, 0, len(bundle.Questions))
	for _, question := range bundle.Questions {
		if len(question.ConceptIDs) != 1 {
			t.Fatalf("question %s maps %d concepts, want 1", question.ID, len(question.ConceptIDs))
		}
		questionConcepts = append(questionConcepts, question.ConceptIDs[0])
	}
	slices.Sort(questionConcepts)
	expected := slices.Clone(lesson.ConceptIDs)
	slices.Sort(expected)
	if !slices.Equal(questionConcepts, expected) {
		t.Fatalf("question concepts = %v, lesson concepts = %v", questionConcepts, expected)
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
