package learning_test

import (
	"testing"
	"time"

	"github.com/Loe159/lpic-daily/internal/learning"
)

const conceptID = "lpic1.103.5.processus-avant-arriere-plan"

func event(id string, at time.Time, activity learning.ActivityKind, evidence learning.EvidenceKind) learning.EvidenceEvent {
	return learning.EvidenceEvent{
		EventID:          id,
		OccurredAt:       at,
		ConceptID:        conceptID,
		ObjectiveIDs:     []string{"103.5"},
		SourceItemID:     "item-" + id,
		ActivityKind:     activity,
		EvidenceKind:     evidence,
		Result:           learning.ResultPass,
		HighestHintLevel: 0,
		SolutionRevealed: false,
		Distribution:     "fedora",
		AttemptIndex:     1,
	}
}

func TestPassiveAndRecognitionDoNotBecomeRecall(t *testing.T) {
	start := time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)
	events := []learning.EvidenceEvent{
		event("lesson", start, learning.ActivityLesson, learning.EvidenceExposure),
		event("mcq", start.Add(time.Minute), learning.ActivityQuestion, learning.EvidenceRecognition),
	}

	got, err := learning.ProjectMastery(conceptID, events, learning.DefaultProjectionPolicy())
	if err != nil {
		t.Fatalf("ProjectMastery() error = %v", err)
	}
	if got.Stage != learning.StageExposed {
		t.Fatalf("stage = %s, want exposed", got.Stage)
	}
}

func TestRecallPromotesRecallStage(t *testing.T) {
	at := time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)
	got, err := learning.ProjectMastery(
		conceptID,
		[]learning.EvidenceEvent{event("recall", at, learning.ActivityQuestion, learning.EvidenceRecall)},
		learning.DefaultProjectionPolicy(),
	)
	if err != nil {
		t.Fatalf("ProjectMastery() error = %v", err)
	}
	if got.Stage != learning.StageRecall {
		t.Fatalf("stage = %s, want recall", got.Stage)
	}
}

func TestStrongHintCapsIndependentAtGuided(t *testing.T) {
	at := time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)
	e := event("lab", at, learning.ActivityLab, learning.EvidenceIndependentPractice)
	e.HighestHintLevel = 3

	got, err := learning.ProjectMastery(conceptID, []learning.EvidenceEvent{e}, learning.DefaultProjectionPolicy())
	if err != nil {
		t.Fatalf("ProjectMastery() error = %v", err)
	}
	if got.Stage != learning.StageGuided {
		t.Fatalf("stage = %s, want guided", got.Stage)
	}
}

func TestSolutionRevealCannotCountAsIndependent(t *testing.T) {
	at := time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)
	e := event("lab", at, learning.ActivityLab, learning.EvidenceIndependentPractice)
	e.HighestHintLevel = 4
	e.SolutionRevealed = true

	got, err := learning.ProjectMastery(conceptID, []learning.EvidenceEvent{e}, learning.DefaultProjectionPolicy())
	if err != nil {
		t.Fatalf("ProjectMastery() error = %v", err)
	}
	if got.Stage != learning.StageGuided {
		t.Fatalf("stage = %s, want guided", got.Stage)
	}
}

func TestLaterIndependentConfirmationCanReachTransfer(t *testing.T) {
	start := time.Date(2026, 9, 24, 8, 0, 0, 0, time.UTC)
	independent := event("lab-a", start, learning.ActivityLab, learning.EvidenceIndependentPractice)
	confirmation := event("lab-a-confirmation", start.Add(25*time.Hour), learning.ActivityLab, learning.EvidenceTransfer)
	confirmation.SourceItemID = independent.SourceItemID
	confirmation.AttemptIndex = 2

	got, err := learning.ProjectMastery(
		conceptID,
		[]learning.EvidenceEvent{independent, confirmation},
		learning.DefaultProjectionPolicy(),
	)
	if err != nil {
		t.Fatalf("ProjectMastery() error = %v", err)
	}
	if got.Stage != learning.StageTransfer || got.SuccessfulTransfer != 1 {
		t.Fatalf("projection = %#v, want confirmed transfer", got)
	}
}

func TestTransferRequiresDifferentLaterContext(t *testing.T) {
	start := time.Date(2026, 9, 24, 8, 0, 0, 0, time.UTC)
	independent := event("lab-a", start, learning.ActivityLab, learning.EvidenceIndependentPractice)
	transfer := event("challenge-b", start.Add(25*time.Hour), learning.ActivityChallenge, learning.EvidenceTransfer)

	got, err := learning.ProjectMastery(
		conceptID,
		[]learning.EvidenceEvent{transfer, independent},
		learning.DefaultProjectionPolicy(),
	)
	if err != nil {
		t.Fatalf("ProjectMastery() error = %v", err)
	}
	if got.Stage != learning.StageTransfer {
		t.Fatalf("stage = %s, want transfer", got.Stage)
	}
	if got.SuccessfulTransfer != 1 {
		t.Fatalf("successful transfer = %d, want 1", got.SuccessfulTransfer)
	}
}

func TestTransferTooSoonStaysIndependent(t *testing.T) {
	start := time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)
	independent := event("lab-a", start, learning.ActivityLab, learning.EvidenceIndependentPractice)
	transfer := event("challenge-b", start.Add(2*time.Hour), learning.ActivityChallenge, learning.EvidenceTransfer)

	got, err := learning.ProjectMastery(
		conceptID,
		[]learning.EvidenceEvent{independent, transfer},
		learning.DefaultProjectionPolicy(),
	)
	if err != nil {
		t.Fatalf("ProjectMastery() error = %v", err)
	}
	if got.Stage != learning.StageIndependent {
		t.Fatalf("stage = %s, want independent", got.Stage)
	}
}

func TestLaterFailureDoesNotEraseDemonstratedStage(t *testing.T) {
	start := time.Date(2026, 9, 24, 8, 0, 0, 0, time.UTC)
	independent := event("lab-a", start, learning.ActivityLab, learning.EvidenceIndependentPractice)
	transfer := event("challenge-b", start.Add(25*time.Hour), learning.ActivityChallenge, learning.EvidenceTransfer)
	failure := event("later-fail", start.Add(48*time.Hour), learning.ActivityLab, learning.EvidenceIndependentPractice)
	failure.Result = learning.ResultFail

	got, err := learning.ProjectMastery(
		conceptID,
		[]learning.EvidenceEvent{independent, transfer, failure},
		learning.DefaultProjectionPolicy(),
	)
	if err != nil {
		t.Fatalf("ProjectMastery() error = %v", err)
	}
	if got.Stage != learning.StageTransfer {
		t.Fatalf("stage = %s, want transfer", got.Stage)
	}
	if got.Failures != 1 {
		t.Fatalf("failures = %d, want 1", got.Failures)
	}
	if !got.LastEvidenceAt.Equal(failure.OccurredAt) {
		t.Fatalf("last evidence = %s, want failure at %s", got.LastEvidenceAt, failure.OccurredAt)
	}
	if !got.LastStageEvidenceAt.Equal(transfer.OccurredAt) {
		t.Fatalf("stage anchor = %s, want transfer at %s", got.LastStageEvidenceAt, transfer.OccurredAt)
	}
}
