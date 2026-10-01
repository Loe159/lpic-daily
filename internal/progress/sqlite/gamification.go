package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Loe159/lpic-daily/internal/gamification"
)

func (store *Store) AppendGamificationEvent(ctx context.Context, event gamification.Event) error {
	if err := event.Validate(); err != nil {
		return fmt.Errorf("validate gamification event: %w", err)
	}
	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("encode gamification metadata: %w", err)
	}
	_, err = store.db.ExecContext(
		ctx,
		`INSERT INTO gamification_events (
			event_id,
			occurred_at,
			event_type,
			amount,
			metadata_json
		) VALUES (?, ?, ?, ?, ?)`,
		event.EventID,
		event.OccurredAt.UTC().Format(time.RFC3339Nano),
		string(event.Type),
		event.Amount,
		string(metadata),
	)
	if err != nil {
		return fmt.Errorf("append gamification event %s: %w", event.EventID, err)
	}
	return nil
}

func (store *Store) AppendGamificationEventIfAbsent(
	ctx context.Context,
	event gamification.Event,
) (bool, error) {
	if err := event.Validate(); err != nil {
		return false, fmt.Errorf("validate gamification event: %w", err)
	}
	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return false, fmt.Errorf("encode gamification metadata: %w", err)
	}
	result, err := store.db.ExecContext(
		ctx,
		`INSERT OR IGNORE INTO gamification_events (
			event_id,
			occurred_at,
			event_type,
			amount,
			metadata_json
		) VALUES (?, ?, ?, ?, ?)`,
		event.EventID,
		event.OccurredAt.UTC().Format(time.RFC3339Nano),
		string(event.Type),
		event.Amount,
		string(metadata),
	)
	if err != nil {
		return false, fmt.Errorf("append gamification event if absent %s: %w", event.EventID, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("inspect gamification insert %s: %w", event.EventID, err)
	}
	return rows == 1, nil
}

func (store *Store) GamificationEvents(ctx context.Context) ([]gamification.Event, error) {
	rows, err := store.db.QueryContext(
		ctx,
		`SELECT event_id, occurred_at, event_type, amount, metadata_json
		FROM gamification_events
		ORDER BY occurred_at, event_id`,
	)
	if err != nil {
		return nil, fmt.Errorf("query gamification events: %w", err)
	}
	defer rows.Close()

	var events []gamification.Event
	for rows.Next() {
		var (
			event        gamification.Event
			occurredAt   string
			eventType    string
			metadataJSON string
		)
		if err := rows.Scan(
			&event.EventID,
			&occurredAt,
			&eventType,
			&event.Amount,
			&metadataJSON,
		); err != nil {
			return nil, fmt.Errorf("scan gamification event: %w", err)
		}
		event.OccurredAt, err = time.Parse(time.RFC3339Nano, occurredAt)
		if err != nil {
			return nil, fmt.Errorf("parse gamification time %q: %w", occurredAt, err)
		}
		event.Type = gamification.EventType(eventType)
		if err := json.Unmarshal([]byte(metadataJSON), &event.Metadata); err != nil {
			return nil, fmt.Errorf("decode gamification metadata for %s: %w", event.EventID, err)
		}
		if err := event.Validate(); err != nil {
			return nil, fmt.Errorf("stored gamification event %s is invalid: %w", event.EventID, err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate gamification events: %w", err)
	}
	return events, nil
}
