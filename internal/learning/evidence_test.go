package learning

import (
	"testing"
	"time"
)

func TestProjectDoesNotTreatRecognitionAsRecall(t *testing.T) {
	now := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	projection := Project("c1", []Event{{
		EventID: "e1", OccurredAt: now, ConceptID: "c1",
		EvidenceKind: EvidenceRecognition, Result: ResultPass,
	}})
	if projection.Stage != StageExposed {
		t.Fatalf("stage=%s, want exposed", projection.Stage)
	}
}

func TestProjectHintDowngradesIndependentEvidence(t *testing.T) {
	now := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	projection := Project("c1", []Event{{
		EventID: "e1", OccurredAt: now, ConceptID: "c1",
		EvidenceKind: EvidenceIndependentPractice, Result: ResultPass, HighestHintLevel: 1,
	}})
	if projection.Stage != StageGuided {
		t.Fatalf("stage=%s, want guided", projection.Stage)
	}
}

func TestConfirmedPracticalMasteryRequiresLaterTransfer(t *testing.T) {
	start := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	events := []Event{
		{EventID: "e1", OccurredAt: start, ConceptID: "c1", EvidenceKind: EvidenceIndependentPractice, Result: ResultPass},
		{EventID: "e2", OccurredAt: start.Add(48 * time.Hour), ConceptID: "c1", EvidenceKind: EvidenceTransfer, Result: ResultPass},
	}
	projection := Project("c1", events)
	if !projection.ConfirmedPracticalMastery(24 * time.Hour) {
		t.Fatal("expected practical mastery confirmation")
	}
	if projection.ConfirmedPracticalMastery(72 * time.Hour) {
		t.Fatal("confirmation should fail when minimum gap is not met")
	}
}

func TestSolutionRevealCannotProduceIndependentStage(t *testing.T) {
	now := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	projection := Project("c1", []Event{{
		EventID: "e1", OccurredAt: now, ConceptID: "c1",
		EvidenceKind: EvidenceTransfer, Result: ResultPass, SolutionRevealed: true, HighestHintLevel: 4,
	}})
	if projection.Stage != StageGuided {
		t.Fatalf("stage=%s, want guided", projection.Stage)
	}
}
