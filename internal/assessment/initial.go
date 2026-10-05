package assessment

import (
	"context"
	"fmt"
	"slices"

	"github.com/Loe159/lpic-daily/internal/content"
	"github.com/Loe159/lpic-daily/internal/curriculum"
	"github.com/Loe159/lpic-daily/internal/learning"
)

type EvidenceReader interface {
	EvidenceForConcept(context.Context, string) ([]learning.EvidenceEvent, error)
}

func Questions(
	curriculumBundle *curriculum.Bundle,
	contentBundle *content.Bundle,
) ([]content.Question, error) {
	if curriculumBundle == nil {
		return nil, fmt.Errorf("curriculum bundle is required")
	}
	if contentBundle == nil {
		return nil, fmt.Errorf("content bundle is required")
	}

	questionsByConcept := make(map[string][]content.Question)
	for _, question := range contentBundle.Questions {
		if question.EvidenceKindOnSuccess != "recall" || len(question.ConceptIDs) != 1 {
			continue
		}
		conceptID := question.ConceptIDs[0]
		questionsByConcept[conceptID] = append(questionsByConcept[conceptID], question)
	}
	for conceptID := range questionsByConcept {
		slices.SortFunc(questionsByConcept[conceptID], func(a, b content.Question) int {
			aAssessment := a.Usage == "initial-assessment"
			bAssessment := b.Usage == "initial-assessment"
			if aAssessment != bAssessment {
				if aAssessment {
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
	}

	conceptIDs := curriculumBundle.Phase1.ObjectiveConcepts["103.1"]
	result := make([]content.Question, 0, len(conceptIDs))
	for _, conceptID := range conceptIDs {
		candidates := questionsByConcept[conceptID]
		if len(candidates) == 0 {
			return nil, fmt.Errorf("missing recall-capable initial assessment question for concept %s", conceptID)
		}
		result = append(result, candidates[0])
	}
	return result, nil
}

func FoundationReady(
	ctx context.Context,
	bundle *curriculum.Bundle,
	evidence EvidenceReader,
) (bool, error) {
	if bundle == nil {
		return false, fmt.Errorf("curriculum bundle is required")
	}
	if evidence == nil {
		return false, fmt.Errorf("evidence reader is required")
	}

	projections := make(map[string]learning.MasteryProjection)
	for _, conceptID := range bundle.Phase1.ObjectiveConcepts["103.1"] {
		events, err := evidence.EvidenceForConcept(ctx, conceptID)
		if err != nil {
			return false, fmt.Errorf("load assessment evidence for %s: %w", conceptID, err)
		}
		projection, err := learning.ProjectMastery(
			conceptID,
			events,
			learning.DefaultProjectionPolicy(),
		)
		if err != nil {
			return false, fmt.Errorf("project assessment mastery for %s: %w", conceptID, err)
		}
		projections[conceptID] = projection
	}

	readiness, err := learning.ObjectiveReadiness(
		bundle,
		projections,
		learning.DefaultReadinessPolicy(),
	)
	if err != nil {
		return false, err
	}
	return readiness["103.1"], nil
}
