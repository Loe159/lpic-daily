package gamification_test

import (
	"context"
	"testing"
	"time"

	"github.com/Loe159/lpic-daily/internal/gamification"
)

type memoryStore struct {
	events []gamification.Event
}

func (store *memoryStore) AppendGamificationEvent(_ context.Context, event gamification.Event) error {
	store.events = append(store.events, event)
	return nil
}

func (store *memoryStore) GamificationEvents(context.Context) ([]gamification.Event, error) {
	return append([]gamification.Event(nil), store.events...), nil
}

func TestProjectStreakAndXP(t *testing.T) {
	location := time.FixedZone("CEST", 2*60*60)
	now := time.Date(2026, 9, 27, 10, 0, 0, 0, location)
	events := []gamification.Event{
		{EventID: "a", OccurredAt: now.AddDate(0, 0, -2), Type: gamification.EventLessonCompleted, Amount: 10},
		{EventID: "b", OccurredAt: now.AddDate(0, 0, -1), Type: gamification.EventQuestionPassed, Amount: 15},
		{EventID: "c", OccurredAt: now, Type: gamification.EventLabPassed, Amount: 50},
	}
	snapshot, err := gamification.Project(events, location, now)
	if err != nil {
		t.Fatalf("Project() error = %v", err)
	}
	if snapshot.XP != 75 || snapshot.CurrentStreakDays != 3 || snapshot.LabsPassed != 1 {
		t.Fatalf("snapshot = %#v", snapshot)
	}
}

func TestRecordActivityUnlocksAchievementsWithoutMasteryState(t *testing.T) {
	store := &memoryStore{}
	now := time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)

	snapshot, unlocked, err := gamification.RecordActivity(
		context.Background(),
		store,
		gamification.EventLessonCompleted,
		now,
		map[string]string{"source_item_id": "lesson-1"},
		time.UTC,
	)
	if err != nil {
		t.Fatalf("RecordActivity() error = %v", err)
	}
	if snapshot.ActivitiesCompleted != 1 {
		t.Fatalf("snapshot = %#v", snapshot)
	}
	if len(unlocked) != 1 || unlocked[0].ID != "first-steps" {
		t.Fatalf("unlocked = %#v", unlocked)
	}
	if snapshot.XP != 30 {
		t.Fatalf("XP = %d, want 30 (10 activity + 20 achievement)", snapshot.XP)
	}
	for _, achievement := range gamification.BuiltinAchievements {
		if achievement.AffectsMastery {
			t.Fatalf("achievement %s unexpectedly affects mastery", achievement.ID)
		}
	}
}
