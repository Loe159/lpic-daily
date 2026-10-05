package assessment

import (
	"context"
	"fmt"

	"github.com/Loe159/lpic-daily/internal/content"
	"github.com/Loe159/lpic-daily/internal/curriculum"
	"github.com/Loe159/lpic-daily/internal/learning"
)

var InitialPhase1QuestionIDs = []string{
	"lpic1.103.1.assess.shell-sequence",
	"lpic1.103.1.assess.export-variable",
	"lpic1.103.1.assess.path-resolution",
	"lpic1.103.1.assess.quoting",
	"lpic1.103.1.assess.history",
	"lpic1.103.1.assess.type",
	"lpic1.103.1.assess.uname-release",
	"lpic1.103.1.set-env-et-portee-des-variables.q.autonomous-recall",
	"lpic1.103.1.export-unset-et-processus-enfants.q.autonomous-recall",
	"lpic1.103.1.execution-de-commandes-hors-path.q.autonomous-recall",
	"lpic1.103.1.edition-et-persistance-de-l-historique.q.autonomous-recall",
	"lpic1.103.1.echo-et-expansion-shell.q.autonomous-recall",
}

type EvidenceReader interface {
	EvidenceForConcept(context.Context, string) ([]learning.EvidenceEvent, error)
}

func Questions(bundle *content.Bundle) ([]content.Question, error) {
	if bundle == nil {
		return nil, fmt.Errorf("content bundle is required")
	}
	result := make([]content.Question, 0, len(InitialPhase1QuestionIDs))
	for _, id := range InitialPhase1QuestionIDs {
		question, ok := bundle.QuestionByID(id)
		if !ok {
			return nil, fmt.Errorf("missing initial assessment question %s", id)
		}
		result = append(result, question)
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
