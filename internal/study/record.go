package study

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/Loe159/lpic-daily/internal/checker"
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
	results := make(map[string]learning.Result, len(authored.Definition.ConceptIDs))
	for _, conceptID := range authored.Definition.ConceptIDs {
		results[conceptID] = learning.ResultPass
	}
	return RecordLabConceptResults(ctx, store, authored, results, highestHintLevel, at)
}

func RecordLabAttempt(
	ctx context.Context,
	store EvidenceStore,
	authored lab.Lab,
	result learning.Result,
	highestHintLevel int,
	at time.Time,
) error {
	if result != learning.ResultPass && result != learning.ResultPartial && result != learning.ResultFail {
		return fmt.Errorf("invalid lab result %q", result)
	}
	results := make(map[string]learning.Result, len(authored.Definition.ConceptIDs))
	for _, conceptID := range authored.Definition.ConceptIDs {
		results[conceptID] = result
	}
	return RecordLabConceptResults(ctx, store, authored, results, highestHintLevel, at)
}

func RecordLabConceptResults(
	ctx context.Context,
	store EvidenceStore,
	authored lab.Lab,
	conceptResults map[string]learning.Result,
	highestHintLevel int,
	at time.Time,
) error {
	if store == nil {
		return fmt.Errorf("evidence store is required")
	}
	if highestHintLevel < 0 || highestHintLevel > 4 {
		return fmt.Errorf("hint level %d outside 0..4", highestHintLevel)
	}

	declared := make(map[string]struct{}, len(authored.Definition.ConceptIDs))
	for _, conceptID := range authored.Definition.ConceptIDs {
		declared[conceptID] = struct{}{}
		result, exists := conceptResults[conceptID]
		if !exists {
			return fmt.Errorf("missing lab result for concept %s", conceptID)
		}
		if result != learning.ResultPass && result != learning.ResultPartial && result != learning.ResultFail {
			return fmt.Errorf("invalid lab result %q for concept %s", result, conceptID)
		}
	}
	for conceptID := range conceptResults {
		if _, exists := declared[conceptID]; !exists {
			return fmt.Errorf("lab result references undeclared concept %s", conceptID)
		}
	}

	batch := make([]learning.EvidenceEvent, 0, len(authored.Definition.ConceptIDs))
	for _, conceptID := range authored.Definition.ConceptIDs {
		events, err := store.EvidenceForConcept(ctx, conceptID)
		if err != nil {
			return fmt.Errorf("load existing lab evidence for %s: %w", conceptID, err)
		}
		attempt := nextAttemptFromEvents(events, authored.Definition.ID)
		evidenceKind, err := practicalEvidenceKind(
			conceptID,
			events,
			authored.Definition.PracticeContext,
			highestHintLevel,
			at,
		)
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
			EvidenceKind: evidenceKind, Result: conceptResults[conceptID],
			HighestHintLevel: highestHintLevel, SolutionRevealed: highestHintLevel == 4,
			Distribution:    distributionOrGeneric(authored.Definition.Environment.Distribution),
			PracticeContext: authored.Definition.PracticeContext,
			AttemptIndex:    attempt,
		})
	}
	if err := store.AppendEvidenceBatch(ctx, batch); err != nil {
		return fmt.Errorf("record lab evidence: %w", err)
	}
	return nil
}

func LabConceptResults(authored lab.Lab, checkResults []checker.Result) (map[string]learning.Result, error) {
	if len(checkResults) != len(authored.Definition.Checks) {
		return nil, fmt.Errorf(
			"lab %s returned %d check results for %d authored checks",
			authored.Definition.ID,
			len(checkResults),
			len(authored.Definition.Checks),
		)
	}

	type tally struct {
		passed int
		failed int
	}
	declared := make(map[string]struct{}, len(authored.Definition.ConceptIDs))
	tallies := make(map[string]tally, len(authored.Definition.ConceptIDs))
	for _, conceptID := range authored.Definition.ConceptIDs {
		declared[conceptID] = struct{}{}
	}

	for index, result := range checkResults {
		for _, conceptID := range authored.Definition.Checks[index].ConceptIDs {
			if _, exists := declared[conceptID]; !exists {
				return nil, fmt.Errorf("check %d maps undeclared concept %s", index+1, conceptID)
			}
			current := tallies[conceptID]
			if result.Pass {
				current.passed++
			} else {
				current.failed++
			}
			tallies[conceptID] = current
		}
	}

	results := make(map[string]learning.Result, len(authored.Definition.ConceptIDs))
	for _, conceptID := range authored.Definition.ConceptIDs {
		current := tallies[conceptID]
		switch {
		case current.passed == 0 && current.failed == 0:
			return nil, fmt.Errorf("concept %s has no evaluated state check", conceptID)
		case current.failed == 0:
			results[conceptID] = learning.ResultPass
		case current.passed == 0:
			results[conceptID] = learning.ResultFail
		default:
			results[conceptID] = learning.ResultPartial
		}
	}
	return results, nil
}

func practicalEvidenceKind(
	conceptID string,
	events []learning.EvidenceEvent,
	practiceContext string,
	highestHintLevel int,
	at time.Time,
) (learning.EvidenceKind, error) {
	if highestHintLevel >= 2 {
		return learning.EvidenceIndependentPractice, nil
	}

	if practiceContext == "" {
		return "", fmt.Errorf("practice context is required for practical evidence")
	}

	policy := learning.DefaultProjectionPolicy()
	for _, event := range events {
		if err := event.Validate(); err != nil {
			return "", fmt.Errorf("validate existing practical evidence for %s: %w", conceptID, err)
		}
		if event.ConceptID != conceptID || event.Result != learning.ResultPass {
			continue
		}
		effective := learning.EffectiveEvidenceKind(event)
		if effective != learning.EvidenceIndependentPractice && effective != learning.EvidenceTransfer {
			continue
		}
		if at.Sub(event.OccurredAt) < policy.MinTransferGap {
			continue
		}
		if event.PracticeContext != "" && event.PracticeContext != practiceContext {
			return learning.EvidenceTransfer, nil
		}
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
