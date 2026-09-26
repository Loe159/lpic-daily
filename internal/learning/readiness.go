package learning

import (
	"fmt"

	"github.com/Loe159/lpic-daily/internal/curriculum"
)

type ReadinessPolicy struct {
	MinimumStage     MasteryStage
	RequiredFraction float64
}

func DefaultReadinessPolicy() ReadinessPolicy {
	return ReadinessPolicy{
		MinimumStage:     StageRecall,
		RequiredFraction: 0.70,
	}
}

// ObjectiveReadiness derives whether an objective provides enough foundation
// to unlock dependants. It is intentionally weaker than objective mastery.
func ObjectiveReadiness(
	bundle *curriculum.Bundle,
	projections map[string]MasteryProjection,
	policy ReadinessPolicy,
) (map[string]bool, error) {
	if policy.RequiredFraction <= 0 || policy.RequiredFraction > 1 {
		return nil, fmt.Errorf("required fraction must be in (0,1], got %f", policy.RequiredFraction)
	}

	total := make(map[string]int)
	ready := make(map[string]int)

	for _, concept := range bundle.Concepts.Concepts {
		if !concept.Active {
			continue
		}
		total[concept.ObjectiveID]++
		if projection, exists := projections[concept.ID]; exists && projection.Stage >= policy.MinimumStage {
			ready[concept.ObjectiveID]++
		}
	}

	result := make(map[string]bool, len(total))
	for objectiveID, count := range total {
		if count == 0 {
			continue
		}
		result[objectiveID] = float64(ready[objectiveID])/float64(count) >= policy.RequiredFraction
	}
	return result, nil
}
