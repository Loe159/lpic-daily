package sqlite

import (
	"context"
	"fmt"
	"time"
)

func (store *Store) NotificationSent(ctx context.Context, localDay string) (bool, error) {
	var count int
	if err := store.db.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM notification_delivery WHERE local_day = ?",
		localDay,
	).Scan(&count); err != nil {
		return false, fmt.Errorf("query notification delivery for %s: %w", localDay, err)
	}
	return count != 0, nil
}

func (store *Store) ClaimNotification(ctx context.Context, localDay string, at time.Time) (bool, error) {
	if localDay == "" || at.IsZero() {
		return false, fmt.Errorf("local day and timestamp are required")
	}
	result, err := store.db.ExecContext(
		ctx,
		`INSERT INTO notification_delivery (local_day, notified_at)
		 VALUES (?, ?)
		 ON CONFLICT(local_day) DO NOTHING`,
		localDay,
		at.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return false, fmt.Errorf("claim notification delivery for %s: %w", localDay, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("inspect notification claim for %s: %w", localDay, err)
	}
	return rows == 1, nil
}

func (store *Store) MarkNotificationSent(ctx context.Context, localDay string, at time.Time) error {
	if localDay == "" || at.IsZero() {
		return fmt.Errorf("local day and timestamp are required")
	}
	_, err := store.db.ExecContext(
		ctx,
		`INSERT INTO notification_delivery (local_day, notified_at)
		 VALUES (?, ?)
		 ON CONFLICT(local_day) DO UPDATE SET notified_at = excluded.notified_at`,
		localDay,
		at.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("mark notification delivery for %s: %w", localDay, err)
	}
	return nil
}
