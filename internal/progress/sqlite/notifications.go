package sqlite

import (
	"context"
	"fmt"
	"time"
)

const notificationClaimTTL = 10 * time.Minute

func (store *Store) NotificationSent(ctx context.Context, localDay string) (bool, error) {
	var count int
	if err := store.db.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM notification_delivery WHERE local_day = ? AND status = 'sent'",
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

	claimedAt := at.UTC()
	staleBefore := claimedAt.Add(-notificationClaimTTL)
	result, err := store.db.ExecContext(
		ctx,
		`INSERT INTO notification_delivery (local_day, notified_at, status)
		 VALUES (?, ?, 'claimed')
		 ON CONFLICT(local_day) DO UPDATE SET
		     notified_at = excluded.notified_at,
		     status = 'claimed'
		 WHERE notification_delivery.status = 'claimed'
		   AND notification_delivery.notified_at <= ?`,
		localDay,
		claimedAt.Format(time.RFC3339Nano),
		staleBefore.Format(time.RFC3339Nano),
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

func (store *Store) ReleaseNotificationClaim(ctx context.Context, localDay string) error {
	if localDay == "" {
		return fmt.Errorf("local day is required")
	}
	if _, err := store.db.ExecContext(
		ctx,
		"DELETE FROM notification_delivery WHERE local_day = ? AND status = 'claimed'",
		localDay,
	); err != nil {
		return fmt.Errorf("release notification claim for %s: %w", localDay, err)
	}
	return nil
}

func (store *Store) MarkNotificationSent(ctx context.Context, localDay string, at time.Time) error {
	if localDay == "" || at.IsZero() {
		return fmt.Errorf("local day and timestamp are required")
	}
	_, err := store.db.ExecContext(
		ctx,
		`INSERT INTO notification_delivery (local_day, notified_at, status)
		 VALUES (?, ?, 'sent')
		 ON CONFLICT(local_day) DO UPDATE SET
		     notified_at = excluded.notified_at,
		     status = 'sent'`,
		localDay,
		at.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("mark notification delivery for %s: %w", localDay, err)
	}
	return nil
}
