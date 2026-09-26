package learning

import (
	"errors"
	"fmt"
	"slices"
	"time"
)

type MasteryStage uint8

const (
	StageUnseen MasteryStage = iota
	StageExposed
	StageRecall
	StageGuided
	StageIndependent
	StageTransfer
)

func (stage MasteryStage) String() string {
	switch stage {
	case StageUnseen:
		return "unseen"
	case StageExposed:
		return "exposed"
	case StageRecall:
		return "recall"
	case StageGuided:
		return "guided"
	case StageIndependent:
		return "independent"
	case StageTransfer:
		return "transfer"
	default:
		return "unknown"
	}
}

type ProjectionPolicy struct {
	MinTransferGap time.Duration
}

func DefaultProjectionPolicy() ProjectionPolicy {
	return ProjectionPolicy{
		MinTransferGap: 24 * time.Hour,
	}
}

type MasteryProjection struct {
	ConceptID             string
	Stage                 MasteryStage
	SuccessfulExposure    int
	SuccessfulRecognition int
	SuccessfulRecall      int
	SuccessfulGuided      int
	SuccessfulIndependent int
	SuccessfulTransfer    int
	Failures              int
	Partials              int
	LastEvidenceAt        time.Time
	LastSuccessAt         time.Time
}

func ProjectMastery(conceptID string, events []EvidenceEvent, policy ProjectionPolicy) (MasteryProjection, error) {
	projection := MasteryProjection{ConceptID: conceptID, Stage: StageUnseen}
	if conceptID == "" {
		return projection, errors.New("concept ID is required")
	}
	if policy.MinTransferGap < 0 {
		return projection, errors.New("minimum transfer gap cannot be negative")
	}

	ordered := append([]EvidenceEvent(nil), events...)
	slices.SortStableFunc(ordered, func(a, b EvidenceEvent) int {
		switch {
		case a.OccurredAt.Before(b.OccurredAt):
			return -1
		case a.OccurredAt.After(b.OccurredAt):
			return 1
		case a.AttemptIndex < b.AttemptIndex:
			return -1
		case a.AttemptIndex > b.AttemptIndex:
			return 1
		default:
			return 0
		}
	})

	var independent []EvidenceEvent
	for _, event := range ordered {
		if err := event.Validate(); err != nil {
			return projection, fmt.Errorf("event %s: %w", event.EventID, err)
		}
		if event.ConceptID != conceptID {
			return projection, fmt.Errorf("event %s belongs to concept %s, expected %s", event.EventID, event.ConceptID, conceptID)
		}

		if event.OccurredAt.After(projection.LastEvidenceAt) {
			projection.LastEvidenceAt = event.OccurredAt
		}

		switch event.Result {
		case ResultFail:
			projection.Failures++
			continue
		case ResultPartial:
			projection.Partials++
			continue
		case ResultPass:
			if event.OccurredAt.After(projection.LastSuccessAt) {
				projection.LastSuccessAt = event.OccurredAt
			}
		}

		effective := EffectiveEvidenceKind(event)
		switch effective {
		case EvidenceExposure:
			projection.SuccessfulExposure++
			projection.Stage = maxStage(projection.Stage, StageExposed)
		case EvidenceRecognition:
			projection.SuccessfulRecognition++
			projection.Stage = maxStage(projection.Stage, StageExposed)
		case EvidenceRecall:
			projection.SuccessfulRecall++
			projection.Stage = maxStage(projection.Stage, StageRecall)
		case EvidenceGuidedPractice:
			projection.SuccessfulGuided++
			projection.Stage = maxStage(projection.Stage, StageGuided)
		case EvidenceIndependentPractice:
			projection.SuccessfulIndependent++
			projection.Stage = maxStage(projection.Stage, StageIndependent)
			independent = append(independent, event)
		case EvidenceTransfer:
			if qualifiesAsTransfer(event, independent, policy.MinTransferGap) {
				projection.SuccessfulTransfer++
				projection.Stage = maxStage(projection.Stage, StageTransfer)
			} else {
				projection.SuccessfulIndependent++
				projection.Stage = maxStage(projection.Stage, StageIndependent)
				independent = append(independent, event)
			}
		}
	}

	return projection, nil
}

func qualifiesAsTransfer(current EvidenceEvent, previous []EvidenceEvent, minGap time.Duration) bool {
	for _, event := range previous {
		if event.SourceItemID == current.SourceItemID {
			continue
		}
		if current.OccurredAt.Sub(event.OccurredAt) >= minGap {
			return true
		}
	}
	return false
}

func maxStage(a, b MasteryStage) MasteryStage {
	if b > a {
		return b
	}
	return a
}
