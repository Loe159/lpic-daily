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
		if strings.HasSuffix(lesson.ID, ".lesson.autonomous") {
			if !strings.Contains(lesson.BodyMarkdown, "## À retenir") {
				t.Errorf("%s generated focused lesson missing concise retention section", concept.ID)
			}
		} else if !strings.Contains(lesson.BodyMarkdown, standaloneSupplementHeading) {
			t.Errorf("%s authored focused lesson has no compact standalone supplement", concept.ID)
		}
		if strings.Contains(lesson.BodyMarkdown, "## Termes, fichiers et utilitaires à connaître pour") {
			t.Errorf("%s still uses the objective-wide term dump instead of focused Exam 101 pedagogy", concept.ID)
		}
		if strings.Contains(lesson.BodyMarkdown, "explique son rôle, donne un cas d'emploi") {
			t.Errorf("%s still contains a learner instruction where a worked example is required", concept.ID)
		}
		for _, filler := range []string{
			"Ce concept appartient à",
			"Ne mémorise pas seulement les noms",
			"Ramène cette pratique au concept",
			"Face à une question sur",
		} {
			if strings.Contains(lesson.BodyMarkdown, filler) {
				t.Errorf("%s still contains generic generated filler %q", concept.ID, filler)
			}
		}

		applicationID := concept.ID + ".q.autonomous-application"
		application, exists := questions[applicationID]
		if !exists {
			t.Errorf("%s has no applied Exam 101 scenario question", concept.ID)
		} else {
			if !strings.Contains(application.PromptFR, "Scénario opérationnel") {
				t.Errorf("%s is not framed as an operational scenario: %q", applicationID, application.PromptFR)
			}
			if previous, duplicate := seenPrompts[application.PromptFR]; duplicate {
				t.Errorf("%s application prompt duplicates %s", applicationID, previous)
			} else {
				seenPrompts[application.PromptFR] = applicationID
			}
			if len(application.Choices) != 4 {
				t.Errorf("%s has %d choices, want 4", applicationID, len(application.Choices))
			}
			seenLabels := make(map[string]struct{}, len(application.Choices))
			correctDescription := standaloneTermExplanation(concept.AnchorTerms[0], concept.ObjectiveID)
			for _, choice := range application.Choices {
				if strings.Contains(choice.LabelFR, correctDescription) {
					t.Errorf("%s leaks a term definition in choice %q", applicationID, choice.LabelFR)
				}
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


func TestGeneratedStandaloneLessonsStayConciseForEveryActiveConcept(t *testing.T) {
	bundle, err := Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	curriculumBundle, err := curriculum.Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("curriculum.Load() error = %v", err)
	}

	conceptByID := make(map[string]curriculum.Concept)
	for _, concept := range curriculumBundle.Concepts.Concepts {
		if concept.Active {
			conceptByID[concept.ID] = concept
		}
	}

	generated := 0
	for _, lesson := range bundle.Lessons {
		if lesson.Stage != "introduce" || len(lesson.ConceptIDs) != 1 ||
			!strings.HasSuffix(lesson.ID, ".lesson.autonomous") {
			continue
		}
		generated++
		concept := conceptByID[lesson.ConceptIDs[0]]
		if concept.ID == "" {
			t.Fatalf("%s references an unknown concept", lesson.ID)
		}
		if !strings.Contains(lesson.BodyMarkdown, "## À retenir") {
			t.Errorf("%s has no concise retention section", lesson.ID)
		}
		for _, anchor := range concept.AnchorTerms {
			if !strings.Contains(lesson.BodyMarkdown, "`"+anchor+"`") {
				t.Errorf("%s misses anchor %q", lesson.ID, anchor)
			}
			explanation := standaloneTermExplanation(anchor, concept.ObjectiveID)
			if !strings.Contains(lesson.BodyMarkdown, explanation) {
				t.Errorf("%s misses the specific explanation for %q", lesson.ID, anchor)
			}
		}
		for _, filler := range []string{
			"Ce concept appartient à",
			"## Modèle mental",
			"## Focus sur ce concept",
			"## Mise en pratique",
			"## Pièges et distinctions",
			"## Termes, fichiers et utilitaires à connaître pour",
			"## Ce que l'examen peut te demander de démontrer",
			"Avant de continuer, reformule",
			"## Vérifie-toi",
		} {
			if strings.Contains(lesson.BodyMarkdown, filler) {
				t.Errorf("%s contains verbose generated filler %q", lesson.ID, filler)
			}
		}
		nonEmptyLines := 0
		for _, line := range strings.Split(lesson.BodyMarkdown, "\n") {
			if strings.TrimSpace(line) != "" {
				nonEmptyLines++
			}
		}
		if maxLines := len(concept.AnchorTerms) + 7; nonEmptyLines > maxLines {
			t.Errorf("%s has %d non-empty lines, want at most %d for %d anchors",
				lesson.ID, nonEmptyLines, maxLines, len(concept.AnchorTerms))
		}
	}
	if generated != 280 {
		t.Fatalf("generated focused lessons = %d, want 280", generated)
	}
}
