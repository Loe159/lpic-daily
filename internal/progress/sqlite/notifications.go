package sqlite

import (
	"context"
	"database/sql"
	"errors"
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

	result, err := store.db.ExecContext(
		ctx,
		`INSERT INTO notification_delivery (local_day, notified_at, status)
		 VALUES (?, ?, 'claimed')
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
	if rows == 1 {
		return true, nil
	}

	var status, claimedAtText string
	err = store.db.QueryRowContext(
		ctx,
		"SELECT status, notified_at FROM notification_delivery WHERE local_day = ?",
		localDay,
	).Scan(&status, &claimedAtText)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect existing notification claim for %s: %w", localDay, err)
	}
	if status != "claimed" {
		return false, nil
	}
	claimedAt, err := time.Parse(time.RFC3339Nano, claimedAtText)
	if err != nil {
		return false, fmt.Errorf("parse notification claim timestamp for %s: %w", localDay, err)
	}
	if at.Before(claimedAt) || at.Sub(claimedAt) < notificationClaimTTL {
		return false, nil
	}

	result, err = store.db.ExecContext(
		ctx,
		`UPDATE notification_delivery
		 SET notified_at = ?
		 WHERE local_day = ? AND status = 'claimed' AND notified_at = ?`,
		at.UTC().Format(time.RFC3339Nano),
		localDay,
		claimedAtText,
	)
	if err != nil {
		return false, fmt.Errorf("reclaim stale notification delivery for %s: %w", localDay, err)
	}
	rows, err = result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("inspect reclaimed notification claim for %s: %w", localDay, err)
	}
	return rows == 1, nil
}

func (store *Store) RefreshNotificationClaim(
	ctx context.Context,
	localDay string,
	claimedAt time.Time,
	at time.Time,
) (bool, error) {
	if localDay == "" || claimedAt.IsZero() || at.IsZero() {
		return false, fmt.Errorf("local day and claim timestamps are required")
	}
	if at.Before(claimedAt) {
		return false, fmt.Errorf("notification claim refresh cannot move backwards in time")
	}

	result, err := store.db.ExecContext(
		ctx,
		`UPDATE notification_delivery
		 SET notified_at = ?
		 WHERE local_day = ? AND status = 'claimed' AND notified_at = ?`,
		at.UTC().Format(time.RFC3339Nano),
		localDay,
		claimedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return false, fmt.Errorf("refresh notification claim for %s: %w", localDay, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("inspect refreshed notification claim for %s: %w", localDay, err)
	}
	return rows == 1, nil
}

func (store *Store) ReleaseNotificationClaim(
	ctx context.Context,
	localDay string,
	claimedAt time.Time,
) (bool, error) {
	if localDay == "" || claimedAt.IsZero() {
		return false, fmt.Errorf("local day and claim timestamp are required")
	}
	result, err := store.db.ExecContext(
		ctx,
		`DELETE FROM notification_delivery
		 WHERE local_day = ? AND status = 'claimed' AND notified_at = ?`,
		localDay,
		claimedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return false, fmt.Errorf("release notification claim for %s: %w", localDay, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("inspect released notification claim for %s: %w", localDay, err)
	}
	return rows == 1, nil
}

func (store *Store) MarkNotificationSent(
	ctx context.Context,
	localDay string,
	claimedAt time.Time,
	at time.Time,
) error {
	if localDay == "" || claimedAt.IsZero() || at.IsZero() {
		return fmt.Errorf("local day and claim timestamps are required")
	}
	result, err := store.db.ExecContext(
		ctx,
		`UPDATE notification_delivery
		 SET notified_at = ?, status = 'sent'
		 WHERE local_day = ? AND status = 'claimed' AND notified_at = ?`,
		at.UTC().Format(time.RFC3339Nano),
		localDay,
		claimedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("mark notification delivery for %s: %w", localDay, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect marked notification delivery for %s: %w", localDay, err)
	}
	if rows != 1 {
		return fmt.Errorf("notification claim for %s was lost before delivery could be recorded", localDay)
	}
	return nil
}
