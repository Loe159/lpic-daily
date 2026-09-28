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
	LastStageEvidenceAt   time.Time
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
		demonstratedStage := StageUnseen
		switch effective {
		case EvidenceExposure:
			projection.SuccessfulExposure++
			demonstratedStage = StageExposed
		case EvidenceRecognition:
			projection.SuccessfulRecognition++
			demonstratedStage = StageExposed
		case EvidenceRecall:
			projection.SuccessfulRecall++
			demonstratedStage = StageRecall
		case EvidenceGuidedPractice:
			projection.SuccessfulGuided++
			demonstratedStage = StageGuided
		case EvidenceIndependentPractice:
			projection.SuccessfulIndependent++
			demonstratedStage = StageIndependent
			independent = append(independent, event)
		case EvidenceTransfer:
			if qualifiesAsTransfer(event, independent, policy.MinTransferGap) {
				projection.SuccessfulTransfer++
				demonstratedStage = StageTransfer
			} else {
				projection.SuccessfulIndependent++
				demonstratedStage = StageIndependent
				independent = append(independent, event)
			}
		}
		if demonstratedStage != StageUnseen {
			projection.Stage = maxStage(projection.Stage, demonstratedStage)
			// Reviews are anchored to evidence that actually supports the
			// current mastery stage. Failures and weaker later activities must
			// not postpone the next review.
			if demonstratedStage == projection.Stage && event.OccurredAt.After(projection.LastStageEvidenceAt) {
				projection.LastStageEvidenceAt = event.OccurredAt
			}
		}
	}

	return projection, nil
}

func qualifiesAsTransfer(current EvidenceEvent, previous []EvidenceEvent, minGap time.Duration) bool {
	for _, event := range previous {
		if current.OccurredAt.Sub(event.OccurredAt) < minGap {
			continue
		}
		// A different activity is genuine transfer. Repeating the same
		// practical activity as a later attempt is accepted as independent
		// confirmation, which is the Phase-1 path to full practical mastery.
		if event.SourceItemID != current.SourceItemID || current.AttemptIndex > event.AttemptIndex {
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
