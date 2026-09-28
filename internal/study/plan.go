package study

import (
	"context"
	"errors"
	"fmt"
	"slices"
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
		input.Policy = learning.DefaultSessionPolicy()
	}

	phase1Concepts := make(map[string]struct{})
	for _, objectiveID := range input.Curriculum.Phase1.SelectedObjectives {
		for _, conceptID := range input.Curriculum.Phase1.ObjectiveConcepts[objectiveID] {
			phase1Concepts[conceptID] = struct{}{}
		}
	}

	projections := make(map[string]learning.MasteryProjection, len(phase1Concepts))
	evidenceByConcept := make(map[string][]learning.EvidenceEvent, len(phase1Concepts))
	for conceptID := range phase1Concepts {
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
		ScopeObjectives:    input.Curriculum.Phase1.SelectedObjectives,
		Policy:             input.Policy,
	})
	if err != nil {
		return Plan{}, fmt.Errorf("build learning session: %w", err)
	}

	conceptTitles := make(map[string]string, len(phase1Concepts))
	for _, concept := range input.Curriculum.Concepts.Concepts {
		if _, wanted := phase1Concepts[concept.ID]; wanted {
			conceptTitles[concept.ID] = concept.TitleFR
		}
	}

	lessons := make(map[string][]content.Lesson)
	for _, lesson := range input.Content.Lessons {
		for _, conceptID := range lesson.ConceptIDs {
			if _, wanted := phase1Concepts[conceptID]; wanted {
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
			if _, wanted := phase1Concepts[conceptID]; wanted {
				questions[conceptID] = append(questions[conceptID], question.ID)
			}
		}
	}

	labs := make(map[string][]string)
	for _, authored := range input.Labs {
		for _, conceptID := range authored.Definition.ConceptIDs {
			if _, wanted := phase1Concepts[conceptID]; wanted {
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
			return Plan{}, fmt.Errorf("scheduled unknown Phase-1 concept %s", scheduled.ConceptID)
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
		if len(item.QuestionIDs) != 0 {
			item.RecommendedQuestionID = item.QuestionIDs[0]
		}
		item.RecommendedLabID = recommendedLab(
			item.MasteryStage,
			item.LabIDs,
			evidenceByConcept[scheduled.ConceptID],
			input.Now,
		)

		plan.Items = append(plan.Items, item)
	}
	return plan, nil
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

func recommendedLab(
	stage learning.MasteryStage,
	labIDs []string,
	events []learning.EvidenceEvent,
	now time.Time,
) string {
	if len(labIDs) == 0 {
		return ""
	}
	candidates := slices.Clone(labIDs)
	slices.Sort(candidates)
	if stage < learning.StageIndependent {
		return candidates[0]
	}

	policy := learning.DefaultProjectionPolicy()
	used := make(map[string]bool)
	transferReady := false
	for _, event := range events {
		if event.Result != learning.ResultPass {
			continue
		}
		effective := learning.EffectiveEvidenceKind(event)
		if effective != learning.EvidenceIndependentPractice && effective != learning.EvidenceTransfer {
			continue
		}
		used[event.SourceItemID] = true
		if !event.OccurredAt.After(now) && now.Sub(event.OccurredAt) >= policy.MinTransferGap {
			transferReady = true
		}
	}
	if transferReady {
		for _, labID := range candidates {
			if !used[labID] {
				return labID
			}
		}
	}
	return candidates[0]
}
