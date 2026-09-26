package learning

import (
	"sort"
	"time"
)

type EvidenceKind string

const (
	EvidenceExposure            EvidenceKind = "exposure"
	EvidenceRecognition         EvidenceKind = "recognition"
	EvidenceRecall              EvidenceKind = "recall"
	EvidenceGuidedPractice      EvidenceKind = "guided-practice"
	EvidenceIndependentPractice EvidenceKind = "independent-practice"
	EvidenceTransfer            EvidenceKind = "transfer"
)

type Result string

const (
	ResultPass    Result = "pass"
	ResultPartial Result = "partial"
	ResultFail    Result = "fail"
)

type ActivityKind string

const (
	ActivityLesson    ActivityKind = "lesson"
	ActivityQuestion  ActivityKind = "question"
	ActivityRep       ActivityKind = "rep"
	ActivityLab       ActivityKind = "lab"
	ActivityChallenge ActivityKind = "challenge"
)

type Event struct {
	EventID          string
	OccurredAt       time.Time
	ConceptID        string
	ObjectiveIDs     []string
	SourceItemID     string
	ActivityKind     ActivityKind
	EvidenceKind     EvidenceKind
	Result           Result
	HighestHintLevel int
	SolutionRevealed bool
	Distribution     string
	AttemptIndex     int
}

type Stage int

const (
	StageUnseen Stage = iota
	StageExposed
	StageRecall
	StageGuided
	StageIndependent
	StageTransfer
)

func (s Stage) String() string {
	switch s {
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

type Projection struct {
	ConceptID               string
	Stage                   Stage
	Attempts                int
	Failures                int
	LastAttempt             time.Time
	LastSuccess             time.Time
	FirstIndependentSuccess time.Time
	LastIndependentSuccess  time.Time
	FirstTransferSuccess    time.Time
	LastTransferSuccess     time.Time
	SuccessfulDays          int
}

func Project(conceptID string, events []Event) Projection {
	projection := Projection{ConceptID: conceptID, Stage: StageUnseen}
	filtered := make([]Event, 0, len(events))
	for _, event := range events {
		if event.ConceptID == conceptID {
			filtered = append(filtered, event)
		}
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].OccurredAt.Before(filtered[j].OccurredAt) })

	days := make(map[string]struct{})
	for _, event := range filtered {
		projection.Attempts++
		if event.OccurredAt.After(projection.LastAttempt) {
			projection.LastAttempt = event.OccurredAt
		}
		if event.Result == ResultFail {
			projection.Failures++
			continue
		}
		if event.Result == ResultPartial {
			projection.Stage = maxStage(projection.Stage, StageExposed)
			continue
		}
		projection.LastSuccess = event.OccurredAt
		days[event.OccurredAt.UTC().Format("2006-01-02")] = struct{}{}

		stage := stageFromSuccessfulEvent(event)
		projection.Stage = maxStage(projection.Stage, stage)
		if stage >= StageIndependent {
			if projection.FirstIndependentSuccess.IsZero() {
				projection.FirstIndependentSuccess = event.OccurredAt
			}
			projection.LastIndependentSuccess = event.OccurredAt
		}
		if stage == StageTransfer {
			if projection.FirstTransferSuccess.IsZero() {
				projection.FirstTransferSuccess = event.OccurredAt
			}
			projection.LastTransferSuccess = event.OccurredAt
		}
	}
	projection.SuccessfulDays = len(days)
	return projection
}

func stageFromSuccessfulEvent(event Event) Stage {
	if event.SolutionRevealed {
		if event.EvidenceKind == EvidenceGuidedPractice || event.EvidenceKind == EvidenceIndependentPractice || event.EvidenceKind == EvidenceTransfer {
			return StageGuided
		}
		return StageExposed
	}
	if event.HighestHintLevel > 0 && (event.EvidenceKind == EvidenceIndependentPractice || event.EvidenceKind == EvidenceTransfer) {
		return StageGuided
	}
	switch event.EvidenceKind {
	case EvidenceExposure, EvidenceRecognition:
		return StageExposed
	case EvidenceRecall:
		return StageRecall
	case EvidenceGuidedPractice:
		return StageGuided
	case EvidenceIndependentPractice:
		return StageIndependent
	case EvidenceTransfer:
		return StageTransfer
	default:
		return StageUnseen
	}
}

func maxStage(a, b Stage) Stage {
	if b > a {
		return b
	}
	return a
}

func (p Projection) ConfirmedPracticalMastery(minGap time.Duration) bool {
	if p.FirstIndependentSuccess.IsZero() || p.LastTransferSuccess.IsZero() {
		return false
	}
	return !p.LastTransferSuccess.Before(p.FirstIndependentSuccess.Add(minGap))
}
