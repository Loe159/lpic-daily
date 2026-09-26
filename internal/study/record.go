package study

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/Loe159/lpic-daily/internal/content"
	"github.com/Loe159/lpic-daily/internal/lab"
	"github.com/Loe159/lpic-daily/internal/learning"
)

type EvidenceStore interface {
	EvidenceReader
	AppendEvidence(context.Context, learning.EvidenceEvent) error
}

func RecordLesson(
	ctx context.Context,
	store EvidenceStore,
	lesson content.Lesson,
	at time.Time,
) error {
	if store == nil {
		return fmt.Errorf("evidence store is required")
	}
	for _, conceptID := range lesson.ConceptIDs {
		attempt, err := nextAttempt(ctx, store, conceptID, lesson.ID)
		if err != nil {
			return err
		}
		eventID, err := newEventID()
		if err != nil {
			return err
		}
		event := learning.EvidenceEvent{
			EventID:          eventID,
			OccurredAt:       at,
			ConceptID:        conceptID,
			ObjectiveIDs:     append([]string(nil), lesson.ObjectiveIDs...),
			SourceItemID:     lesson.ID,
			ActivityKind:     learning.ActivityLesson,
			EvidenceKind:     learning.EvidenceExposure,
			Result:           learning.ResultPass,
			HighestHintLevel: 0,
			SolutionRevealed: false,
			Distribution:     distributionOrGeneric(lesson.Distribution),
			AttemptIndex:     attempt,
		}
		if err := store.AppendEvidence(ctx, event); err != nil {
			return fmt.Errorf("record lesson evidence for %s: %w", conceptID, err)
		}
	}
	return nil
}

func RecordQuestion(
	ctx context.Context,
	store EvidenceStore,
	question content.Question,
	pass bool,
	at time.Time,
) error {
	if store == nil {
		return fmt.Errorf("evidence store is required")
	}

	evidenceKind := learning.EvidenceKind(question.EvidenceKindOnSuccess)
	result := learning.ResultFail
	if pass {
		result = learning.ResultPass
	}

	for _, conceptID := range question.ConceptIDs {
		attempt, err := nextAttempt(ctx, store, conceptID, question.ID)
		if err != nil {
			return err
		}
		eventID, err := newEventID()
		if err != nil {
			return err
		}
		event := learning.EvidenceEvent{
			EventID:          eventID,
			OccurredAt:       at,
			ConceptID:        conceptID,
			ObjectiveIDs:     append([]string(nil), question.ObjectiveIDs...),
			SourceItemID:     question.ID,
			ActivityKind:     learning.ActivityQuestion,
			EvidenceKind:     evidenceKind,
			Result:           result,
			HighestHintLevel: 0,
			SolutionRevealed: false,
			Distribution:     distributionOrGeneric(question.Distribution),
			AttemptIndex:     attempt,
		}
		if err := store.AppendEvidence(ctx, event); err != nil {
			return fmt.Errorf("record question evidence for %s: %w", conceptID, err)
		}
	}
	return nil
}

func RecordLab(
	ctx context.Context,
	store EvidenceStore,
	authored lab.Lab,
	highestHintLevel int,
	at time.Time,
) error {
	if store == nil {
		return fmt.Errorf("evidence store is required")
	}
	if highestHintLevel < 0 || highestHintLevel > 4 {
		return fmt.Errorf("hint level %d outside 0..4", highestHintLevel)
	}

	for _, conceptID := range authored.Definition.ConceptIDs {
		attempt, err := nextAttempt(ctx, store, conceptID, authored.Definition.ID)
		if err != nil {
			return err
		}
		eventID, err := newEventID()
		if err != nil {
			return err
		}
		event := learning.EvidenceEvent{
			EventID:          eventID,
			OccurredAt:       at,
			ConceptID:        conceptID,
			ObjectiveIDs:     append([]string(nil), authored.Definition.ObjectiveIDs...),
			SourceItemID:     authored.Definition.ID,
			ActivityKind:     learning.ActivityLab,
			EvidenceKind:     learning.EvidenceIndependentPractice,
			Result:           learning.ResultPass,
			HighestHintLevel: highestHintLevel,
			SolutionRevealed: highestHintLevel == 4,
			Distribution:     distributionOrGeneric(authored.Definition.Environment.Distribution),
			AttemptIndex:     attempt,
		}
		if err := store.AppendEvidence(ctx, event); err != nil {
			return fmt.Errorf("record lab evidence for %s: %w", conceptID, err)
		}
	}
	return nil
}

func nextAttempt(
	ctx context.Context,
	store EvidenceReader,
	conceptID string,
	sourceItemID string,
) (int, error) {
	events, err := store.EvidenceForConcept(ctx, conceptID)
	if err != nil {
		return 0, fmt.Errorf("load existing attempts for %s: %w", conceptID, err)
	}
	maxAttempt := 0
	for _, event := range events {
		if event.SourceItemID == sourceItemID && event.AttemptIndex > maxAttempt {
			maxAttempt = event.AttemptIndex
		}
	}
	return maxAttempt + 1, nil
}

func newEventID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", fmt.Errorf("generate evidence event id: %w", err)
	}
	return "ev-" + hex.EncodeToString(bytes[:]), nil
}

func distributionOrGeneric(distribution string) string {
	if distribution == "" {
		return "generic"
	}
	return distribution
}
