package content

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	lpicdaily "github.com/Loe159/lpic-daily"
	"github.com/Loe159/lpic-daily/internal/curriculum"
)

func TestRecursiveJSONFilesIncludesNestedContent(t *testing.T) {
	fsys := fstest.MapFS{
		"content/lpic-1-v5/lessons/root.json":         {Data: []byte("{}")},
		"content/lpic-1-v5/lessons/topic/nested.json": {Data: []byte("{}")},
		"content/lpic-1-v5/lessons/topic/readme.txt":  {Data: []byte("ignore")},
	}
	got, err := recursiveJSONFiles(fsys, "content/lpic-1-v5/lessons")
	if err != nil {
		t.Fatalf("recursiveJSONFiles() error = %v", err)
	}
	want := []string{
		"content/lpic-1-v5/lessons/root.json",
		"content/lpic-1-v5/lessons/topic/nested.json",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("recursiveJSONFiles() = %#v, want %#v", got, want)
	}
}

func TestBuiltinPhase1ContentCoversEveryConcept(t *testing.T) {
	bundle, err := Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	curriculumBundle, err := curriculum.Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("curriculum.Load() error = %v", err)
	}

	phase1Concepts := make([]string, 0, 27)
	phase1Set := make(map[string]struct{}, 27)
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

func TestBuiltinStandaloneContentCoversEveryActiveConcept(t *testing.T) {
	bundle, err := Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	curriculumBundle, err := curriculum.Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("curriculum.Load() error = %v", err)
	}

	introductions := make(map[string]int)
	dailyQuestions := make(map[string]int)
	recallQuestions := make(map[string]int)
	for _, lesson := range bundle.Lessons {
		if lesson.Stage == "introduce" && len(lesson.ConceptIDs) == 1 {
			introductions[lesson.ConceptIDs[0]]++
		}
	}
	for _, question := range bundle.Questions {
		if question.Usage == "initial-assessment" {
			continue
		}
		for _, conceptID := range question.ConceptIDs {
			dailyQuestions[conceptID]++
			if question.EvidenceKindOnSuccess == "recall" {
				recallQuestions[conceptID]++
			}
		}
	}

	for _, concept := range curriculumBundle.Concepts.Concepts {
		if !concept.Active {
			continue
		}
		if introductions[concept.ID] != 1 {
			t.Errorf("concept %s focused introductions = %d, want exactly 1", concept.ID, introductions[concept.ID])
		}
		if dailyQuestions[concept.ID] < 2 {
			t.Errorf("concept %s daily questions = %d, want at least 2", concept.ID, dailyQuestions[concept.ID])
		}
		if recallQuestions[concept.ID] < 1 {
			t.Errorf("concept %s has no recall-capable daily question", concept.ID)
		}
	}
}

func TestEveryPedagogicalTermHasSpecificStandaloneExplanation(t *testing.T) {
	curriculumBundle, err := curriculum.Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("curriculum.Load() error = %v", err)
	}
	for _, objective := range curriculumBundle.Objectives.Objectives {
		if !objective.Active {
			continue
		}
		for _, term := range objective.TermsFilesUtilities {
			if _, ok := standaloneTermExplanations[term]; ok {
				continue
			}
			if _, ok := standalonePortServices[term]; ok && objective.ID == "109.1" {
				continue
			}
			t.Errorf("%s term %q has no specific standalone explanation", objective.ID, term)
		}
	}
}

func TestEveryPedagogicalTermAppearsInLearnerLessons(t *testing.T) {
	bundle, err := Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	curriculumBundle, err := curriculum.Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("curriculum.Load() error = %v", err)
	}

	bodiesByObjective := make(map[string]string)
	for _, lesson := range bundle.Lessons {
		for _, objectiveID := range lesson.ObjectiveIDs {
			bodiesByObjective[objectiveID] += "\n" + lesson.BodyMarkdown
		}
	}
	for _, objective := range curriculumBundle.Objectives.Objectives {
		if !objective.Active {
			continue
		}
		body := bodiesByObjective[objective.ID]
		for _, term := range objective.TermsFilesUtilities {
			if !strings.Contains(body, "`"+term+"`") {
				t.Errorf("%s term %q never appears in learner lesson content", objective.ID, term)
			}
		}
	}
}

func TestEveryConceptAnchorHasSpecificStandaloneExplanation(t *testing.T) {
	curriculumBundle, err := curriculum.Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("curriculum.Load() error = %v", err)
	}
	for _, concept := range curriculumBundle.Concepts.Concepts {
		if !concept.Active {
			continue
		}
		for _, anchor := range concept.AnchorTerms {
			if !hasSpecificStandaloneTermExplanation(anchor, concept.ObjectiveID) {
				t.Errorf("%s anchor %q has no specific standalone explanation", concept.ID, anchor)
			}
		}
	}
}

func TestEveryConceptAnchorAppearsInFocusedLesson(t *testing.T) {
	bundle, err := Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	curriculumBundle, err := curriculum.Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("curriculum.Load() error = %v", err)
	}

	focusedBodies := make(map[string]string)
	for _, lesson := range bundle.Lessons {
		if lesson.Stage == "introduce" && len(lesson.ConceptIDs) == 1 {
			focusedBodies[lesson.ConceptIDs[0]] = lesson.BodyMarkdown
		}
	}
	for _, concept := range curriculumBundle.Concepts.Concepts {
		if !concept.Active {
			continue
		}
		body := focusedBodies[concept.ID]
		for _, anchor := range concept.AnchorTerms {
			if !strings.Contains(body, "`"+anchor+"`") {
				t.Errorf("%s anchor %q missing from focused lesson", concept.ID, anchor)
			}
			explanation := standaloneTermExplanation(anchor, concept.ObjectiveID)
			if !strings.Contains(body, explanation) {
				t.Errorf("%s anchor %q explanation missing from focused lesson", concept.ID, anchor)
			}
		}
	}
}

func TestGeneratedRecallQuestionsUseConceptAnchor(t *testing.T) {
	bundle, err := Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	curriculumBundle, err := curriculum.Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("curriculum.Load() error = %v", err)
	}

	questions := make(map[string]Question)
	for _, question := range bundle.Questions {
		questions[question.ID] = question
	}
	for _, concept := range curriculumBundle.Concepts.Concepts {
		if !concept.Active {
			continue
		}
		for anchorIndex, anchor := range concept.AnchorTerms {
			questionID := concept.ID + ".q.autonomous-recall"
			if anchorIndex > 0 {
				questionID += fmt.Sprintf("-%02d", anchorIndex+1)
			}
			question, exists := questions[questionID]
			if !exists {
				t.Fatalf("missing generated recall question %s", questionID)
			}
			if len(question.Grading.AcceptedAnswers) != 1 ||
				question.Grading.AcceptedAnswers[0] != anchor {
				t.Errorf(
					"%s accepted answers = %v, want anchor %q",
					questionID,
					question.Grading.AcceptedAnswers,
					anchor,
				)
			}
		}
	}
}

func TestExam101GeneratedPedagogyIsFocusedAndApplied(t *testing.T) {
	bundle, err := Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	curriculumBundle, err := curriculum.Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("curriculum.Load() error = %v", err)
	}

	objectiveExam := make(map[string]string)
	for _, objective := range curriculumBundle.Objectives.Objectives {
		if objective.Active {
			objectiveExam[objective.ID] = objective.Exam
		}
	}
	focusedLessons := make(map[string]Lesson)
	questions := make(map[string]Question)
	for _, lesson := range bundle.Lessons {
		if lesson.Stage == "introduce" && len(lesson.ConceptIDs) == 1 {
			focusedLessons[lesson.ConceptIDs[0]] = lesson
		}
	}
	for _, question := range bundle.Questions {
		questions[question.ID] = question
	}

	seenPrompts := make(map[string]string)
	exam101Concepts := 0
	for _, concept := range curriculumBundle.Concepts.Concepts {
		if !concept.Active || objectiveExam[concept.ObjectiveID] != "101" {
			continue
		}
		exam101Concepts++

		lesson, exists := focusedLessons[concept.ID]
		if !exists {
			t.Fatalf("missing focused lesson for Exam 101 concept %s", concept.ID)
		}
		for _, heading := range []string{
			"## À comprendre précisément",
			"## Exemple travaillé / commandes",
			"## Raisonnement attendu",
			"## Auto-test",
		} {
			if !strings.Contains(lesson.BodyMarkdown, heading) {
				t.Errorf("%s focused lesson missing %q", concept.ID, heading)
			}
		}
		if strings.Contains(lesson.BodyMarkdown, "## Termes, fichiers et utilitaires à connaître pour") {
			t.Errorf("%s still uses the objective-wide term dump instead of focused Exam 101 pedagogy", concept.ID)
		}

		applicationID := concept.ID + ".q.autonomous-application"
		application, exists := questions[applicationID]
		if !exists {
			t.Errorf("%s has no applied Exam 101 scenario question", concept.ID)
		} else {
			if previous, duplicate := seenPrompts[application.PromptFR]; duplicate {
				t.Errorf("%s application prompt duplicates %s", applicationID, previous)
			} else {
				seenPrompts[application.PromptFR] = applicationID
			}
			if len(application.Choices) != 4 {
				t.Errorf("%s has %d choices, want 4", applicationID, len(application.Choices))
			}
			seenLabels := make(map[string]struct{}, len(application.Choices))
			for _, choice := range application.Choices {
				if strings.Contains(choice.LabelFR, "alternative-") {
					t.Errorf("%s contains synthetic distractor %q", applicationID, choice.LabelFR)
				}
				if _, duplicate := seenLabels[choice.LabelFR]; duplicate {
					t.Errorf("%s contains duplicate choice %q", applicationID, choice.LabelFR)
				}
				seenLabels[choice.LabelFR] = struct{}{}
			}
		}

		for anchorIndex, anchor := range concept.AnchorTerms {
			recallID := concept.ID + ".q.autonomous-recall"
			if anchorIndex > 0 {
				recallID += fmt.Sprintf("-%02d", anchorIndex+1)
			}
			recall := questions[recallID]
			wantCaseSensitive := recallAnswerCaseSensitive(anchor)
			if recall.Grading.CaseSensitive != wantCaseSensitive {
				t.Errorf("%s case-sensitive = %t, want %t for anchor %q", recallID, recall.Grading.CaseSensitive, wantCaseSensitive, anchor)
			}
			if previous, duplicate := seenPrompts[recall.PromptFR]; duplicate {
				t.Errorf("%s recall prompt duplicates %s", recallID, previous)
			} else {
				seenPrompts[recall.PromptFR] = recallID
			}
		}
	}
	if exam101Concepts != 162 {
		t.Fatalf("Exam 101 concept count = %d, want 162", exam101Concepts)
	}
}
