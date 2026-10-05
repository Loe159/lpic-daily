package tui

import (
	"strings"
	"testing"

	"github.com/Loe159/lpic-daily/internal/gamification"
	"github.com/Loe159/lpic-daily/internal/learning"
	"github.com/Loe159/lpic-daily/internal/study"
)

func TestSanitizeTextRemovesTerminalControls(t *testing.T) {
	got := SanitizeText("safe\x1b[31mred\x1b[0m\rrewrite\x07\u009b31m")
	for _, forbidden := range []rune{'\x1b', '\r', '\x07', '\u009b'} {
		if strings.ContainsRune(got, forbidden) {
			t.Fatalf("sanitized text still contains control %U: %q", forbidden, got)
		}
	}
	if !strings.Contains(got, "safe") || !strings.Contains(got, "red") || !strings.Contains(got, "rewrite") {
		t.Fatalf("sanitized text lost printable content: %q", got)
	}
}

func TestDashboardRenderSanitizesPlanText(t *testing.T) {
	model := NewDashboard(study.Plan{Items: []study.Item{{
		ConceptID:             "concept",
		ConceptTitleFR:        "titre\x1b[2Jinjecté\r",
		ObjectiveID:           "103.1",
		Kind:                  learning.SessionNew,
		MasteryStage:          learning.StageUnseen,
		ReasonFR:              "raison\x07",
		RecommendedLessonID:   "lesson\x1b[31m",
		RecommendedQuestionID: "question",
	}}}, gamification.Snapshot{XP: 42, CurrentStreakDays: 2})

	rendered := model.render()
	if strings.ContainsRune(rendered, '\x1b') || strings.ContainsRune(rendered, '\r') || strings.ContainsRune(rendered, '\x07') {
		t.Fatalf("render contains control sequence: %q", rendered)
	}
	if !strings.Contains(rendered, "Maîtrise: unseen") {
		t.Fatalf("render lost mastery stage label: %q", rendered)
	}
}

func TestDashboardNavigationAndDefaultAction(t *testing.T) {
	model := NewDashboard(study.Plan{Items: []study.Item{
		{
			ConceptID:             "first",
			ConceptTitleFR:        "Premier",
			ObjectiveID:           "103.1",
			Kind:                  learning.SessionNew,
			RecommendedLessonID:   "lesson.first",
			RecommendedQuestionID: "question.first",
		},
		{
			ConceptID:             "second",
			ConceptTitleFR:        "Second",
			ObjectiveID:           "103.1",
			Kind:                  learning.SessionReview,
			RecommendedQuestionID: "question.second",
		},
	}}, gamification.Snapshot{})

	model, quit := model.updateKey("down")
	if quit || model.cursor != 1 {
		t.Fatalf("down => cursor=%d quit=%v", model.cursor, quit)
	}
	model, quit = model.updateKey("enter")
	if !quit {
		t.Fatal("enter did not request program exit for selected action")
	}
	if model.action.Kind != ActionQuestion || model.action.ID != "question.second" {
		t.Fatalf("action = %#v", model.action)
	}
}

func TestNewConceptDefaultsToFocusedLesson(t *testing.T) {
	model := NewDashboard(study.Plan{Items: []study.Item{{
		ConceptID:             "first",
		ConceptTitleFR:        "Premier",
		ObjectiveID:           "103.1",
		Kind:                  learning.SessionNew,
		RecommendedLessonID:   "lesson.first",
		RecommendedQuestionID: "question.first",
	}}}, gamification.Snapshot{})
	model, quit := model.updateKey("enter")
	if !quit || model.action != (Action{Kind: ActionLesson, ID: "lesson.first"}) {
		t.Fatalf("action = %#v quit=%v", model.action, quit)
	}
}

func TestRecognizedExposedReviewDefaultsToLab(t *testing.T) {
	model := NewDashboard(study.Plan{Items: []study.Item{{
		ConceptID:             "first",
		ConceptTitleFR:        "Premier",
		ObjectiveID:           "103.1",
		Kind:                  learning.SessionReview,
		MasteryStage:          learning.StageExposed,
		RecommendedQuestionID: "question.first",
		RecommendedLabID:      "lab.first",
		PreferLab:             true,
	}}}, gamification.Snapshot{})
	model, quit := model.updateKey("enter")
	if !quit || model.action != (Action{Kind: ActionLab, ID: "lab.first"}) {
		t.Fatalf("action = %#v quit=%v", model.action, quit)
	}
}

func TestGuidedReviewDefaultsToLab(t *testing.T) {
	model := NewDashboard(study.Plan{Items: []study.Item{{
		ConceptID:             "first",
		ConceptTitleFR:        "Premier",
		ObjectiveID:           "103.1",
		Kind:                  learning.SessionReview,
		MasteryStage:          learning.StageGuided,
		RecommendedQuestionID: "question.first",
		RecommendedLabID:      "lab.first",
	}}}, gamification.Snapshot{})
	model, quit := model.updateKey("enter")
	if !quit || model.action != (Action{Kind: ActionLab, ID: "lab.first"}) {
		t.Fatalf("action = %#v quit=%v", model.action, quit)
	}
}

func TestRecallReviewDefaultsToQuestion(t *testing.T) {
	model := NewDashboard(study.Plan{Items: []study.Item{{
		ConceptID:             "first",
		ConceptTitleFR:        "Premier",
		ObjectiveID:           "103.1",
		Kind:                  learning.SessionReview,
		MasteryStage:          learning.StageRecall,
		RecommendedQuestionID: "question.first",
		RecommendedLabID:      "lab.first",
	}}}, gamification.Snapshot{})
	model, quit := model.updateKey("enter")
	if !quit || model.action != (Action{Kind: ActionQuestion, ID: "question.first"}) {
		t.Fatalf("action = %#v quit=%v", model.action, quit)
	}
}

func TestPracticeDefaultsToQuestion(t *testing.T) {
	model := NewDashboard(study.Plan{Items: []study.Item{{
		ConceptID:             "first",
		ConceptTitleFR:        "Premier",
		ObjectiveID:           "103.1",
		Kind:                  learning.SessionPractice,
		RecommendedQuestionID: "question.first",
	}}}, gamification.Snapshot{})
	model, quit := model.updateKey("enter")
	if !quit || model.action != (Action{Kind: ActionQuestion, ID: "question.first"}) {
		t.Fatalf("action = %#v quit=%v", model.action, quit)
	}
	if !strings.Contains(model.render(), "Consolidation") {
		t.Fatalf("render = %q", model.render())
	}
}

func TestDashboardEmptyPlanShowsCompletedSession(t *testing.T) {
	model := NewDashboard(study.Plan{}, gamification.Snapshot{XP: 198, CurrentStreakDays: 1})

	rendered := model.render()
	for _, want := range []string{
		"✓ Séance du jour terminée",
		"Aucune révision n'est due pour le moment.",
		"prochaines révisions et activités pratiques",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("empty dashboard missing %q: %q", want, rendered)
		}
	}
	if strings.Contains(rendered, "Aucune activité due dans le périmètre") {
		t.Fatalf("empty dashboard still exposes implementation wording: %q", rendered)
	}
}
