package content

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/Loe159/lpic-daily/internal/curriculum"
)

const (
	lessonGlob   = "content/lpic-1-v5/lessons/*.json"
	questionGlob = "content/lpic-1-v5/questions/*.json"
)

var allowedLabels = map[string]struct{}{
	"lpic-required":   {},
	"lpic-legacy":     {},
	"modern-practice": {},
}

func Load(fsys fs.FS) (*Bundle, error) {
	curriculumBundle, err := curriculum.Load(fsys)
	if err != nil {
		return nil, fmt.Errorf("load curriculum for content validation: %w", err)
	}

	lessonPaths, err := fs.Glob(fsys, lessonGlob)
	if err != nil {
		return nil, fmt.Errorf("glob lessons: %w", err)
	}
	questionPaths, err := fs.Glob(fsys, questionGlob)
	if err != nil {
		return nil, fmt.Errorf("glob questions: %w", err)
	}
	slices.Sort(lessonPaths)
	slices.Sort(questionPaths)

	if len(lessonPaths) == 0 {
		return nil, errors.New("no built-in lessons found")
	}
	if len(questionPaths) == 0 {
		return nil, errors.New("no built-in questions found")
	}

	knownConcepts := make(map[string]curriculum.Concept, len(curriculumBundle.Concepts.Concepts))
	for _, concept := range curriculumBundle.Concepts.Concepts {
		if concept.Active {
			knownConcepts[concept.ID] = concept
		}
	}
	knownObjectives := make(map[string]struct{}, len(curriculumBundle.Objectives.Objectives))
	for _, objective := range curriculumBundle.Objectives.Objectives {
		if objective.Active {
			knownObjectives[objective.ID] = struct{}{}
		}
	}

	bundle := &Bundle{
		Lessons:   make([]Lesson, 0, len(lessonPaths)),
		Questions: make([]Question, 0, len(questionPaths)),
	}
	seenIDs := make(map[string]string, len(lessonPaths)+len(questionPaths))

	for _, name := range lessonPaths {
		var lesson Lesson
		if err := decodeStrictFile(fsys, name, &lesson); err != nil {
			return nil, err
		}
		if err := validateLesson(lesson, knownObjectives, knownConcepts); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		if previous, exists := seenIDs[lesson.ID]; exists {
			return nil, fmt.Errorf("%s: duplicate content id %s also used by %s", name, lesson.ID, previous)
		}
		seenIDs[lesson.ID] = name
		bundle.Lessons = append(bundle.Lessons, lesson)
	}

	for _, name := range questionPaths {
		var question Question
		if err := decodeStrictFile(fsys, name, &question); err != nil {
			return nil, err
		}
		if err := validateQuestion(question, knownObjectives, knownConcepts); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		if previous, exists := seenIDs[question.ID]; exists {
			return nil, fmt.Errorf("%s: duplicate content id %s also used by %s", name, question.ID, previous)
		}
		seenIDs[question.ID] = name
		bundle.Questions = append(bundle.Questions, question)
	}

	return bundle, nil
}

func validateLesson(
	lesson Lesson,
	knownObjectives map[string]struct{},
	knownConcepts map[string]curriculum.Concept,
) error {
	if lesson.SchemaVersion != "1.0.0" {
		return fmt.Errorf("unsupported schema version %q", lesson.SchemaVersion)
	}
	if strings.TrimSpace(lesson.ID) == "" || len(strings.TrimSpace(lesson.TitleFR)) < 3 {
		return errors.New("id and title_fr are required")
	}
	if lesson.EstimatedMinutes < 1 || lesson.EstimatedMinutes > 45 {
		return fmt.Errorf("estimated_minutes %d outside 1..45", lesson.EstimatedMinutes)
	}
	switch lesson.Stage {
	case "introduce", "deepen", "apply", "transfer", "exam-review":
	default:
		return fmt.Errorf("unsupported lesson stage %q", lesson.Stage)
	}
	if len(strings.TrimSpace(lesson.BodyMarkdown)) < 20 {
		return errors.New("body_markdown is too short")
	}
	if err := validateReferences(lesson.ObjectiveIDs, lesson.ConceptIDs, knownObjectives, knownConcepts); err != nil {
		return err
	}
	if err := validateLabels(lesson.Labels); err != nil {
		return err
	}
	for _, conceptID := range lesson.PrerequisiteConceptIDs {
		if _, exists := knownConcepts[conceptID]; !exists {
			return fmt.Errorf("unknown prerequisite concept %s", conceptID)
		}
	}
	return validateDistribution(lesson.Distribution)
}

func validateQuestion(
	question Question,
	knownObjectives map[string]struct{},
	knownConcepts map[string]curriculum.Concept,
) error {
	if question.SchemaVersion != "1.0.0" {
		return fmt.Errorf("unsupported schema version %q", question.SchemaVersion)
	}
	if strings.TrimSpace(question.ID) == "" || len(strings.TrimSpace(question.PromptFR)) < 5 {
		return errors.New("id and prompt_fr are required")
	}
	switch question.Type {
	case "free-recall", "fill-in", "multiple-choice", "ordering", "command-output":
	case "matching":
		return errors.New("matching questions are reserved by the schema but not implemented in the Phase-1 runtime")
	default:
		return fmt.Errorf("unsupported question type %q", question.Type)
	}
	if question.EvidenceKindOnSuccess != "recognition" && question.EvidenceKindOnSuccess != "recall" {
		return fmt.Errorf("unsupported evidence kind %q", question.EvidenceKindOnSuccess)
	}
	if err := validateReferences(question.ObjectiveIDs, question.ConceptIDs, knownObjectives, knownConcepts); err != nil {
		return err
	}
	if err := validateLabels(question.Labels); err != nil {
		return err
	}
	if err := validateDistribution(question.Distribution); err != nil {
		return err
	}

	choiceIDs := make(map[string]struct{}, len(question.Choices))
	for _, choice := range question.Choices {
		if strings.TrimSpace(choice.ID) == "" || strings.TrimSpace(choice.LabelFR) == "" {
			return errors.New("choice id and label_fr are required")
		}
		if _, exists := choiceIDs[choice.ID]; exists {
			return fmt.Errorf("duplicate choice id %s", choice.ID)
		}
		choiceIDs[choice.ID] = struct{}{}
	}

	switch question.Grading.Strategy {
	case "exact-text":
		if len(question.Grading.AcceptedAnswers) == 0 {
			return errors.New("exact-text grading requires accepted_answers")
		}
		if question.Grading.Pattern != "" || len(question.Grading.AcceptedChoiceIDs) != 0 || len(question.Grading.ExpectedOrder) != 0 {
			return errors.New("exact-text grading contains fields from another strategy")
		}
	case "regex":
		if question.Grading.Pattern == "" {
			return errors.New("regex grading requires pattern")
		}
		if _, err := regexp.Compile(question.Grading.Pattern); err != nil {
			return fmt.Errorf("invalid grading regex: %w", err)
		}
		if len(question.Grading.AcceptedAnswers) != 0 || len(question.Grading.AcceptedChoiceIDs) != 0 || len(question.Grading.ExpectedOrder) != 0 {
			return errors.New("regex grading contains fields from another strategy")
		}
	case "choice-ids":
		if question.Type != "multiple-choice" {
			return errors.New("choice-ids grading requires multiple-choice type")
		}
		if len(question.Choices) == 0 || len(question.Grading.AcceptedChoiceIDs) == 0 {
			return errors.New("choice-ids grading requires choices and accepted_choice_ids")
		}
		for _, id := range question.Grading.AcceptedChoiceIDs {
			if _, exists := choiceIDs[id]; !exists {
				return fmt.Errorf("accepted choice %s is not declared", id)
			}
		}
	case "ordered-ids":
		if question.Type != "ordering" {
			return errors.New("ordered-ids grading requires ordering type")
		}
		if len(question.Choices) < 2 || len(question.Grading.ExpectedOrder) < 2 {
			return errors.New("ordered-ids grading requires at least two choices")
		}
		if len(question.Grading.ExpectedOrder) != len(question.Choices) {
			return errors.New("expected_order must include every declared choice")
		}
		for _, id := range question.Grading.ExpectedOrder {
			if _, exists := choiceIDs[id]; !exists {
				return fmt.Errorf("ordered choice %s is not declared", id)
			}
		}
	default:
		return fmt.Errorf("unsupported grading strategy %q", question.Grading.Strategy)
	}

	switch question.Type {
	case "multiple-choice", "ordering":
		if len(question.Choices) == 0 {
			return fmt.Errorf("%s question requires choices", question.Type)
		}
	default:
		if len(question.Choices) != 0 {
			return fmt.Errorf("%s question must not declare choices", question.Type)
		}
		if question.Grading.Strategy != "exact-text" && question.Grading.Strategy != "regex" {
			return fmt.Errorf("%s question requires text/regex grading", question.Type)
		}
	}
	return nil
}

func validateReferences(
	objectiveIDs []string,
	conceptIDs []string,
	knownObjectives map[string]struct{},
	knownConcepts map[string]curriculum.Concept,
) error {
	if len(objectiveIDs) == 0 || len(conceptIDs) == 0 {
		return errors.New("objective_ids and concept_ids are required")
	}
	if duplicate := firstDuplicate(objectiveIDs); duplicate != "" {
		return fmt.Errorf("duplicate objective id %s", duplicate)
	}
	if duplicate := firstDuplicate(conceptIDs); duplicate != "" {
		return fmt.Errorf("duplicate concept id %s", duplicate)
	}

	objectiveSet := make(map[string]struct{}, len(objectiveIDs))
	for _, objectiveID := range objectiveIDs {
		if _, exists := knownObjectives[objectiveID]; !exists {
			return fmt.Errorf("unknown objective %s", objectiveID)
		}
		objectiveSet[objectiveID] = struct{}{}
	}
	for _, conceptID := range conceptIDs {
		concept, exists := knownConcepts[conceptID]
		if !exists {
			return fmt.Errorf("unknown concept %s", conceptID)
		}
		if _, exists := objectiveSet[concept.ObjectiveID]; !exists {
			return fmt.Errorf("concept %s belongs to objective %s which is not referenced", conceptID, concept.ObjectiveID)
		}
	}
	return nil
}

func validateLabels(labels []string) error {
	if len(labels) == 0 {
		return errors.New("at least one label is required")
	}
	if duplicate := firstDuplicate(labels); duplicate != "" {
		return fmt.Errorf("duplicate label %s", duplicate)
	}
	for _, label := range labels {
		if _, exists := allowedLabels[label]; !exists {
			return fmt.Errorf("unsupported label %q", label)
		}
	}
	return nil
}

func validateDistribution(distribution string) error {
	switch distribution {
	case "", "generic", "fedora", "debian", "opensuse":
		return nil
	default:
		return fmt.Errorf("unsupported distribution %q", distribution)
	}
}

func firstDuplicate(values []string) string {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			return value
		}
		seen[value] = struct{}{}
	}
	return ""
}

func decodeStrictFile(fsys fs.FS, name string, out any) error {
	file, err := fsys.Open(name)
	if err != nil {
		return fmt.Errorf("open %s: %w", name, err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return fmt.Errorf("decode %s: %w", name, err)
	}

	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("decode %s: trailing JSON value", name)
		}
		return fmt.Errorf("decode %s trailing data: %w", name, err)
	}
	return nil
}

func BaseName(id string) string {
	return path.Base(id)
}
