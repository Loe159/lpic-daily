package study

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/Loe159/lpic-daily/internal/content"
	"github.com/Loe159/lpic-daily/internal/curriculum"
	"github.com/Loe159/lpic-daily/internal/lab"
	"github.com/Loe159/lpic-daily/internal/learning"
)

type EvidenceReader interface {
	EvidenceForConcept(context.Context, string) ([]learning.EvidenceEvent, error)
}

type PlanInput struct {
	Now        time.Time
	Curriculum *curriculum.Bundle
	Content    *content.Bundle
	Labs       []lab.Lab
	Evidence   EvidenceReader
	Policy     learning.SessionPolicy
}

type Item struct {
	ConceptID             string
	ConceptTitleFR        string
	ObjectiveID           string
	Kind                  learning.SessionItemKind
	MasteryStage          learning.MasteryStage
	ReasonFR              string
	DueAt                 time.Time
	LessonIDs             []string
	QuestionIDs           []string
	LabIDs                []string
	RecommendedLessonID   string
	RecommendedQuestionID string
	RecommendedLabID      string
	PreferLab             bool
}

type Plan struct {
	GeneratedAt time.Time
	Items       []Item
}

func BuildPlan(ctx context.Context, input PlanInput) (Plan, error) {
	if input.Curriculum == nil {
		return Plan{}, errors.New("curriculum bundle is required")
	}
	if input.Content == nil {
		return Plan{}, errors.New("content bundle is required")
	}
	if input.Evidence == nil {
		return Plan{}, errors.New("evidence reader is required")
	}
	if input.Now.IsZero() {
		return Plan{}, errors.New("current time is required")
	}
	if input.Policy.ReviewIntervals == nil {
		defaults := learning.DefaultSessionPolicy()
		if input.Policy.MaxReviews == 0 && input.Policy.MaxNewConcepts == 0 {
			input.Policy = defaults
		} else {
			input.Policy.ReviewIntervals = defaults.ReviewIntervals
		}
	}

	scopeObjectives, scopeConcepts, err := schedulableScope(input.Curriculum, input.Content, input.Labs)
	if err != nil {
		return Plan{}, fmt.Errorf("derive schedulable curriculum scope: %w", err)
	}

	projections := make(map[string]learning.MasteryProjection, len(scopeConcepts))
	evidenceByConcept := make(map[string][]learning.EvidenceEvent, len(scopeConcepts))
	for conceptID := range scopeConcepts {
		events, err := input.Evidence.EvidenceForConcept(ctx, conceptID)
		if err != nil {
			return Plan{}, fmt.Errorf("load evidence for %s: %w", conceptID, err)
		}
		evidenceByConcept[conceptID] = slices.Clone(events)
		projection, err := learning.ProjectMastery(
			conceptID,
			events,
			learning.DefaultProjectionPolicy(),
		)
		if err != nil {
			return Plan{}, fmt.Errorf("project mastery for %s: %w", conceptID, err)
		}
		projections[conceptID] = projection
	}

	readiness, err := learning.ObjectiveReadiness(
		input.Curriculum,
		projections,
		learning.DefaultReadinessPolicy(),
	)
	if err != nil {
		return Plan{}, fmt.Errorf("derive objective readiness: %w", err)
	}

	session, err := learning.BuildSession(learning.SessionInput{
		Now:                input.Now,
		Bundle:             input.Curriculum,
		Projections:        projections,
		ObjectiveReadiness: readiness,
		ScopeObjectives:    scopeObjectives,
		Policy:             input.Policy,
	})
	if err != nil {
		return Plan{}, fmt.Errorf("build learning session: %w", err)
	}

	conceptTitles := make(map[string]string, len(scopeConcepts))
	for _, concept := range input.Curriculum.Concepts.Concepts {
		if _, wanted := scopeConcepts[concept.ID]; wanted {
			conceptTitles[concept.ID] = concept.TitleFR
		}
	}

	lessons := make(map[string][]content.Lesson)
	for _, lesson := range input.Content.Lessons {
		for _, conceptID := range lesson.ConceptIDs {
			if _, wanted := scopeConcepts[conceptID]; wanted {
				lessons[conceptID] = append(lessons[conceptID], lesson)
			}
		}
	}

	questions := make(map[string][]string)
	for _, question := range input.Content.Questions {
		if question.Usage == "initial-assessment" {
			continue
		}
		for _, conceptID := range question.ConceptIDs {
			if _, wanted := scopeConcepts[conceptID]; wanted {
				questions[conceptID] = append(questions[conceptID], question.ID)
			}
		}
	}

	labs := make(map[string][]string)
	labContexts := make(map[string]string, len(input.Labs))
	for _, authored := range input.Labs {
		labContexts[authored.Definition.ID] = authored.Definition.PracticeContext
		for _, conceptID := range authored.Definition.ConceptIDs {
			if _, wanted := scopeConcepts[conceptID]; wanted {
				labs[conceptID] = append(labs[conceptID], authored.Definition.ID)
			}
		}
	}

	plan := Plan{
		GeneratedAt: session.GeneratedAt,
		Items:       make([]Item, 0, len(session.Items)),
	}
	for _, scheduled := range session.Items {
		title, exists := conceptTitles[scheduled.ConceptID]
		if !exists {
			return Plan{}, fmt.Errorf("scheduled unknown study-scope concept %s", scheduled.ConceptID)
		}

		item := Item{
			ConceptID:      scheduled.ConceptID,
			ConceptTitleFR: title,
			ObjectiveID:    scheduled.ObjectiveID,
			Kind:           scheduled.Kind,
			MasteryStage:   projections[scheduled.ConceptID].Stage,
			ReasonFR:       scheduled.ReasonFR,
			DueAt:          scheduled.DueAt,
			QuestionIDs:    slices.Clone(questions[scheduled.ConceptID]),
			LabIDs:         slices.Clone(labs[scheduled.ConceptID]),
		}
		for _, lesson := range lessons[scheduled.ConceptID] {
			item.LessonIDs = append(item.LessonIDs, lesson.ID)
		}
		slices.Sort(item.LessonIDs)
		slices.Sort(item.QuestionIDs)
		slices.Sort(item.LabIDs)

		item.RecommendedLessonID = recommendedLesson(scheduled.Kind, lessons[scheduled.ConceptID])
		item.RecommendedQuestionID = recommendedQuestion(
			item.QuestionIDs,
			evidenceByConcept[scheduled.ConceptID],
		)
		item.RecommendedLabID = recommendedLab(
			item.MasteryStage,
			item.LabIDs,
			labContexts,
			evidenceByConcept[scheduled.ConceptID],
			input.Now,
		)
		projection := projections[scheduled.ConceptID]
		hasQuestionSuccess := projection.SuccessfulRecognition > 0 || projection.SuccessfulRecall > 0
		hasPracticalSuccess := projection.SuccessfulGuided > 0 ||
			projection.SuccessfulIndependent > 0 ||
			projection.SuccessfulTransfer > 0
		item.PreferLab = scheduled.Kind != learning.SessionNew &&
			(projection.Stage == learning.StageExposed || projection.Stage == learning.StageRecall) &&
			hasQuestionSuccess &&
			!hasPracticalSuccess &&
			item.RecommendedLabID != ""

		plan.Items = append(plan.Items, item)
	}
	return plan, nil
}

func schedulableScope(
	curriculumBundle *curriculum.Bundle,
	contentBundle *content.Bundle,
	labs []lab.Lab,
) ([]string, map[string]struct{}, error) {
	introductionCounts := make(map[string]int)
	for _, lesson := range contentBundle.Lessons {
		if lesson.Stage != "introduce" || len(lesson.ConceptIDs) != 1 {
			continue
		}
		introductionCounts[lesson.ConceptIDs[0]]++
	}
	questions := make(map[string]bool)
	for _, question := range contentBundle.Questions {
		if question.Usage == "initial-assessment" {
			continue
		}
		for _, conceptID := range question.ConceptIDs {
			questions[conceptID] = true
		}
	}
	practical := make(map[string]bool)
	for _, authored := range labs {
		for _, conceptID := range authored.Definition.ConceptIDs {
			practical[conceptID] = true
		}
	}

	conceptsByObjective := make(map[string][]string)
	for _, concept := range curriculumBundle.Concepts.Concepts {
		if concept.Active {
			conceptsByObjective[concept.ObjectiveID] = append(conceptsByObjective[concept.ObjectiveID], concept.ID)
		}
	}
	objectives := make([]string, 0, len(curriculumBundle.Objectives.Objectives))
	concepts := make(map[string]struct{})
	for _, objective := range curriculumBundle.Objectives.Objectives {
		if !objective.Active {
			continue
		}
		objectiveID := objective.ID
		conceptIDs := conceptsByObjective[objectiveID]
		if len(conceptIDs) == 0 {
			return nil, nil, fmt.Errorf("active objective %s has no concepts", objectiveID)
		}
		complete := true
		for _, conceptID := range conceptIDs {
			if introductionCounts[conceptID] != 1 || !questions[conceptID] || !practical[conceptID] {
				complete = false
				break
			}
		}
		if !complete {
			continue
		}
		objectives = append(objectives, objectiveID)
		for _, conceptID := range conceptIDs {
			concepts[conceptID] = struct{}{}
		}
	}
	if len(objectives) == 0 {
		return nil, nil, errors.New("no objectives have complete introduction, daily-question, and practical-lab coverage")
	}
	return objectives, concepts, nil
}

func recommendedLesson(kind learning.SessionItemKind, lessons []content.Lesson) string {
	if kind != learning.SessionNew {
		return ""
	}

	candidates := slices.Clone(lessons)
	slices.SortFunc(candidates, func(a, b content.Lesson) int {
		aFocused := a.Stage == "introduce" && len(a.ConceptIDs) == 1
		bFocused := b.Stage == "introduce" && len(b.ConceptIDs) == 1
		if aFocused != bFocused {
			if aFocused {
				return -1
			}
			return 1
		}
		switch {
		case a.ID < b.ID:
			return -1
		case a.ID > b.ID:
			return 1
		default:
			return 0
		}
	})
	if len(candidates) == 0 {
		return ""
	}
	return candidates[0].ID
}

func recommendedQuestion(
	questionIDs []string,
	events []learning.EvidenceEvent,
) string {
	if len(questionIDs) == 0 {
		return ""
	}

	candidates := slices.Clone(questionIDs)
	slices.Sort(candidates)
	attempts := make(map[string]int, len(candidates))
	known := make(map[string]struct{}, len(candidates))
	for _, questionID := range candidates {
		known[questionID] = struct{}{}
	}
	for _, event := range events {
		if event.ActivityKind != learning.ActivityQuestion {
			continue
		}
		if _, exists := known[event.SourceItemID]; !exists {
			continue
		}
		if event.AttemptIndex > attempts[event.SourceItemID] {
			attempts[event.SourceItemID] = event.AttemptIndex
		}
	}

	slices.SortFunc(candidates, func(a, b string) int {
		if attempts[a] != attempts[b] {
			return attempts[a] - attempts[b]
		}
		if questionPracticePriority(a) != questionPracticePriority(b) {
			return questionPracticePriority(a) - questionPracticePriority(b)
		}
		switch {
		case a < b:
			return -1
		case a > b:
			return 1
		default:
			return 0
		}
	})
	return candidates[0]
}

func questionPracticePriority(questionID string) int {
	switch {
	case strings.Contains(questionID, ".q.autonomous-recall"):
		return 0
	case strings.Contains(questionID, ".q.autonomous-application"):
		return 2
	case strings.Contains(questionID, ".q.autonomous-recognition"):
		return 3
	default:
		return 1
	}
}

func recommendedLab(
	stage learning.MasteryStage,
	labIDs []string,
	labContexts map[string]string,
	events []learning.EvidenceEvent,
	now time.Time,
) string {
	if len(labIDs) == 0 {
		return ""
	}
	candidates := slices.Clone(labIDs)
	slices.SortFunc(candidates, func(a, b string) int {
		aStandalone := strings.Contains(a, ".standalone-")
		bStandalone := strings.Contains(b, ".standalone-")
		if aStandalone != bStandalone {
			if aStandalone {
				return 1
			}
			return -1
		}
		return strings.Compare(a, b)
	})
	if stage < learning.StageIndependent {
		return candidates[0]
	}

	policy := learning.DefaultProjectionPolicy()
	usedContexts := make(map[string]bool)
	transferReady := false
	for _, event := range events {
		if event.Result != learning.ResultPass {
			continue
		}
		effective := learning.EffectiveEvidenceKind(event)
		if effective != learning.EvidenceIndependentPractice && effective != learning.EvidenceTransfer {
			continue
		}
		if event.PracticeContext == "" {
			continue
		}
		usedContexts[event.PracticeContext] = true
		if !event.OccurredAt.After(now) && now.Sub(event.OccurredAt) >= policy.MinTransferGap {
			transferReady = true
		}
	}
	if transferReady {
		for _, labID := range candidates {
			practiceContext := labContexts[labID]
			if practiceContext != "" && !usedContexts[practiceContext] {
				return labID
			}
		}
	}
	return candidates[0]
}
