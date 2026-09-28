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
	AppendEvidenceBatch(context.Context, []learning.EvidenceEvent) error
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
	batch := make([]learning.EvidenceEvent, 0, len(lesson.ConceptIDs))
	for _, conceptID := range lesson.ConceptIDs {
		attempt, err := nextAttempt(ctx, store, conceptID, lesson.ID)
		if err != nil {
			return err
		}
		eventID, err := newEventID()
		if err != nil {
			return err
		}
		batch = append(batch, learning.EvidenceEvent{
			EventID: eventID, OccurredAt: at, ConceptID: conceptID,
			ObjectiveIDs: append([]string(nil), lesson.ObjectiveIDs...),
			SourceItemID: lesson.ID, ActivityKind: learning.ActivityLesson,
			EvidenceKind: learning.EvidenceExposure, Result: learning.ResultPass,
			HighestHintLevel: 0, SolutionRevealed: false,
			Distribution: distributionOrGeneric(lesson.Distribution), AttemptIndex: attempt,
		})
	}
	if err := store.AppendEvidenceBatch(ctx, batch); err != nil {
		return fmt.Errorf("record lesson evidence: %w", err)
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
	batch := make([]learning.EvidenceEvent, 0, len(question.ConceptIDs))
	for _, conceptID := range question.ConceptIDs {
		attempt, err := nextAttempt(ctx, store, conceptID, question.ID)
		if err != nil {
			return err
		}
		eventID, err := newEventID()
		if err != nil {
			return err
		}
		batch = append(batch, learning.EvidenceEvent{
			EventID: eventID, OccurredAt: at, ConceptID: conceptID,
			ObjectiveIDs: append([]string(nil), question.ObjectiveIDs...),
			SourceItemID: question.ID, ActivityKind: learning.ActivityQuestion,
			EvidenceKind: evidenceKind, Result: result,
			HighestHintLevel: 0, SolutionRevealed: false,
			Distribution: distributionOrGeneric(question.Distribution), AttemptIndex: attempt,
		})
	}
	if err := store.AppendEvidenceBatch(ctx, batch); err != nil {
		return fmt.Errorf("record question evidence: %w", err)
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
	return RecordLabAttempt(ctx, store, authored, learning.ResultPass, highestHintLevel, at)
}

func RecordLabAttempt(
	ctx context.Context,
	store EvidenceStore,
	authored lab.Lab,
	result learning.Result,
	highestHintLevel int,
	at time.Time,
) error {
	if store == nil {
		return fmt.Errorf("evidence store is required")
	}
	if result != learning.ResultPass && result != learning.ResultPartial && result != learning.ResultFail {
		return fmt.Errorf("invalid lab result %q", result)
	}
	if highestHintLevel < 0 || highestHintLevel > 4 {
		return fmt.Errorf("hint level %d outside 0..4", highestHintLevel)
	}
	batch := make([]learning.EvidenceEvent, 0, len(authored.Definition.ConceptIDs))
	for _, conceptID := range authored.Definition.ConceptIDs {
		events, err := store.EvidenceForConcept(ctx, conceptID)
		if err != nil {
			return fmt.Errorf("load existing lab evidence for %s: %w", conceptID, err)
		}
		attempt := nextAttemptFromEvents(events, authored.Definition.ID)
		evidenceKind, err := practicalEvidenceKind(conceptID, events, highestHintLevel, at)
		if err != nil {
			return err
		}
		eventID, err := newEventID()
		if err != nil {
			return err
		}
		batch = append(batch, learning.EvidenceEvent{
			EventID: eventID, OccurredAt: at, ConceptID: conceptID,
			ObjectiveIDs: append([]string(nil), authored.Definition.ObjectiveIDs...),
			SourceItemID: authored.Definition.ID, ActivityKind: learning.ActivityLab,
			EvidenceKind: evidenceKind, Result: result,
			HighestHintLevel: highestHintLevel, SolutionRevealed: highestHintLevel == 4,
			Distribution: distributionOrGeneric(authored.Definition.Environment.Distribution),
			AttemptIndex: attempt,
		})
	}
	if err := store.AppendEvidenceBatch(ctx, batch); err != nil {
		return fmt.Errorf("record lab evidence: %w", err)
	}
	return nil
}

func practicalEvidenceKind(
	conceptID string,
	events []learning.EvidenceEvent,
	highestHintLevel int,
	at time.Time,
) (learning.EvidenceKind, error) {
	if highestHintLevel >= 2 {
		return learning.EvidenceIndependentPractice, nil
	}

	policy := learning.DefaultProjectionPolicy()
	projection, err := learning.ProjectMastery(conceptID, events, policy)
	if err != nil {
		return "", fmt.Errorf("project existing practical evidence for %s: %w", conceptID, err)
	}
	if projection.Stage >= learning.StageIndependent &&
		!projection.LastStageEvidenceAt.IsZero() &&
		!at.Before(projection.LastStageEvidenceAt.Add(policy.MinTransferGap)) {
		return learning.EvidenceTransfer, nil
	}
	return learning.EvidenceIndependentPractice, nil
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
	return nextAttemptFromEvents(events, sourceItemID), nil
}

func nextAttemptFromEvents(events []learning.EvidenceEvent, sourceItemID string) int {
	maxAttempt := 0
	for _, event := range events {
		if event.SourceItemID == sourceItemID && event.AttemptIndex > maxAttempt {
			maxAttempt = event.AttemptIndex
		}
	}
	return maxAttempt + 1
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
