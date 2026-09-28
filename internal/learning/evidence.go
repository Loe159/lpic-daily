package learning

import (
	"errors"
	"fmt"
	"time"
)

type ActivityKind string

const (
	ActivityLesson    ActivityKind = "lesson"
	ActivityQuestion  ActivityKind = "question"
	ActivityRep       ActivityKind = "rep"
	ActivityLab       ActivityKind = "lab"
	ActivityChallenge ActivityKind = "challenge"
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

type EvidenceEvent struct {
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
	PracticeContext  string
	AttemptIndex     int
}

func (event EvidenceEvent) Validate() error {
	var errs []error

	if event.EventID == "" {
		errs = append(errs, errors.New("event ID is required"))
	}
	if event.OccurredAt.IsZero() {
		errs = append(errs, errors.New("occurred_at is required"))
	}
	if event.ConceptID == "" {
		errs = append(errs, errors.New("concept ID is required"))
	}
	if len(event.ObjectiveIDs) == 0 {
		errs = append(errs, errors.New("at least one objective ID is required"))
	}
	if event.SourceItemID == "" {
		errs = append(errs, errors.New("source item ID is required"))
	}
	if !validActivity(event.ActivityKind) {
		errs = append(errs, fmt.Errorf("invalid activity kind %q", event.ActivityKind))
	}
	if !validEvidence(event.EvidenceKind) {
		errs = append(errs, fmt.Errorf("invalid evidence kind %q", event.EvidenceKind))
	}
	if !validResult(event.Result) {
		errs = append(errs, fmt.Errorf("invalid result %q", event.Result))
	}
	if event.HighestHintLevel < 0 || event.HighestHintLevel > 4 {
		errs = append(errs, fmt.Errorf("hint level %d outside 0..4", event.HighestHintLevel))
	}
	if event.SolutionRevealed && event.HighestHintLevel < 4 {
		errs = append(errs, errors.New("solution_revealed requires hint level 4"))
	}
	if event.HighestHintLevel == 4 && !event.SolutionRevealed {
		errs = append(errs, errors.New("hint level 4 must mark solution_revealed"))
	}
	if event.AttemptIndex < 1 {
		errs = append(errs, errors.New("attempt index must be >= 1"))
	}
	if event.Distribution == "" {
		errs = append(errs, errors.New("distribution is required"))
	}

	if event.ActivityKind == ActivityLesson && event.EvidenceKind != EvidenceExposure {
		errs = append(errs, errors.New("lesson activity may only record exposure evidence"))
	}
	if event.ActivityKind == ActivityQuestion &&
		event.EvidenceKind != EvidenceRecognition &&
		event.EvidenceKind != EvidenceRecall {
		errs = append(errs, errors.New("question activity may only record recognition or recall evidence"))
	}

	return errors.Join(errs...)
}

func EffectiveEvidenceKind(event EvidenceEvent) EvidenceKind {
	kind := event.EvidenceKind
	if event.Result != ResultPass {
		return EvidenceExposure
	}

	if event.SolutionRevealed || event.HighestHintLevel >= 2 {
		switch kind {
		case EvidenceIndependentPractice, EvidenceTransfer:
			return EvidenceGuidedPractice
		}
	}
	return kind
}

func validActivity(kind ActivityKind) bool {
	switch kind {
	case ActivityLesson, ActivityQuestion, ActivityRep, ActivityLab, ActivityChallenge:
		return true
	default:
		return false
	}
}

func validEvidence(kind EvidenceKind) bool {
	switch kind {
	case EvidenceExposure,
		EvidenceRecognition,
		EvidenceRecall,
		EvidenceGuidedPractice,
		EvidenceIndependentPractice,
		EvidenceTransfer:
		return true
	default:
		return false
	}
}

func validResult(result Result) bool {
	switch result {
	case ResultPass, ResultPartial, ResultFail:
		return true
	default:
		return false
	}
}
