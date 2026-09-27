package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/Loe159/lpic-daily/internal/gamification"
)

func TestGamificationRoundTripRemainsSeparate(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	event := gamification.Event{
		EventID:    "game-1",
		OccurredAt: time.Date(2026, 9, 27, 8, 0, 0, 0, time.UTC),
		Type:       gamification.EventLessonCompleted,
		Amount:     10,
		Metadata:   map[string]string{"source_item_id": "lesson-1"},
	}
	if err := store.AppendGamificationEvent(ctx, event); err != nil {
		t.Fatalf("AppendGamificationEvent() error = %v", err)
	}
	events, err := store.GamificationEvents(ctx)
	if err != nil {
		t.Fatalf("GamificationEvents() error = %v", err)
	}
	if len(events) != 1 || events[0].EventID != event.EventID {
		t.Fatalf("events = %#v", events)
	}
	evidence, err := store.EvidenceForConcept(ctx, "lpic1.103.1.syntaxe-shell-et-sequences-de-commandes")
	if err != nil {
		t.Fatalf("EvidenceForConcept() error = %v", err)
	}
	if len(evidence) != 0 {
		t.Fatalf("gamification leaked into mastery evidence: %#v", evidence)
	}
}
