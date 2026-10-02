package gamification

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

type Store interface {
	AppendGamificationEvent(context.Context, Event) error
	AppendGamificationEventIfAbsent(context.Context, Event) (bool, error)
	GamificationEvents(context.Context) ([]Event, error)
}

func RecordActivity(
	ctx context.Context,
	store Store,
	eventType EventType,
	at time.Time,
	metadata map[string]string,
	location *time.Location,
) (Snapshot, []Achievement, error) {
	if store == nil {
		return Snapshot{}, nil, fmt.Errorf("gamification store is required")
	}

	eventID, err := newEventID()
	if err != nil {
		return Snapshot{}, nil, err
	}
	event := Event{
		EventID:    eventID,
		OccurredAt: at,
		Type:       eventType,
		Amount:     XPFor(eventType),
		Metadata:   cloneMetadata(metadata),
	}
	if err := store.AppendGamificationEvent(ctx, event); err != nil {
		return Snapshot{}, nil, fmt.Errorf("append gamification event: %w", err)
	}

	events, err := store.GamificationEvents(ctx)
	if err != nil {
		return Snapshot{}, nil, fmt.Errorf("load gamification events: %w", err)
	}
	snapshot, err := Project(events, location, at)
	if err != nil {
		return Snapshot{}, nil, err
	}

	candidates := NewlyUnlocked(snapshot)
	newlyUnlocked := make([]Achievement, 0, len(candidates))
	for _, achievement := range candidates {
		unlock := Event{
			EventID:    "achievement-" + achievement.ID,
			OccurredAt: at,
			Type:       EventAchievementUnlocked,
			Amount:     achievement.XPReward,
			Metadata: map[string]string{
				"achievement_id": achievement.ID,
			},
		}
		inserted, err := store.AppendGamificationEventIfAbsent(ctx, unlock)
		if err != nil {
			return Snapshot{}, nil, fmt.Errorf("unlock achievement %s: %w", achievement.ID, err)
		}
		if inserted {
			newlyUnlocked = append(newlyUnlocked, achievement)
		}
	}

	if len(candidates) != 0 {
		events, err = store.GamificationEvents(ctx)
		if err != nil {
			return Snapshot{}, nil, err
		}
		snapshot, err = Project(events, location, at)
		if err != nil {
			return Snapshot{}, nil, err
		}
	}

	return snapshot, newlyUnlocked, nil
}

func newEventID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate gamification event id: %w", err)
	}
	return "game-" + hex.EncodeToString(value[:]), nil
}

func cloneMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return map[string]string{}
	}
	copy := make(map[string]string, len(metadata))
	for key, value := range metadata {
		copy[key] = value
	}
	return copy
}
