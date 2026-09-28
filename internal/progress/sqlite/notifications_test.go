package sqlite

import (
	"context"
	"testing"
	"time"
)

func TestNotificationDeliveryRoundTrip(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	version, err := store.SchemaVersion(ctx)
	if err != nil {
		t.Fatalf("SchemaVersion() error = %v", err)
	}
	if version != 5 {
		t.Fatalf("schema version = %d, want 5", version)
	}

	const day = "2026-09-27"
	start := time.Date(2026, 9, 27, 8, 0, 0, 0, time.UTC)

	sent, err := store.NotificationSent(ctx, day)
	if err != nil {
		t.Fatalf("NotificationSent() error = %v", err)
	}
	if sent {
		t.Fatal("fresh day unexpectedly sent")
	}

	claimed, err := store.ClaimNotification(ctx, day, start)
	if err != nil {
		t.Fatalf("ClaimNotification() error = %v", err)
	}
	if !claimed {
		t.Fatal("fresh day was not claimed")
	}
	sent, err = store.NotificationSent(ctx, day)
	if err != nil {
		t.Fatalf("NotificationSent() after claim error = %v", err)
	}
	if sent {
		t.Fatal("claim was incorrectly treated as successful delivery")
	}

	claimedAgain, err := store.ClaimNotification(ctx, day, start.Add(time.Minute))
	if err != nil {
		t.Fatalf("second ClaimNotification() error = %v", err)
	}
	if claimedAgain {
		t.Fatal("live claim was acquired twice")
	}

	if err := store.ReleaseNotificationClaim(ctx, day); err != nil {
		t.Fatalf("ReleaseNotificationClaim() error = %v", err)
	}
	reclaimed, err := store.ClaimNotification(ctx, day, start.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("reclaim after release error = %v", err)
	}
	if !reclaimed {
		t.Fatal("released claim could not be acquired again")
	}

	if err := store.MarkNotificationSent(ctx, day, start.Add(3*time.Minute)); err != nil {
		t.Fatalf("MarkNotificationSent() error = %v", err)
	}
	sent, err = store.NotificationSent(ctx, day)
	if err != nil || !sent {
		t.Fatalf("NotificationSent() = %v, %v", sent, err)
	}
	claimedAfterSend, err := store.ClaimNotification(ctx, day, start.Add(24*time.Hour))
	if err != nil {
		t.Fatalf("ClaimNotification() after send error = %v", err)
	}
	if claimedAfterSend {
		t.Fatal("sent notification was claimed again")
	}
}

func TestNotificationClaimDoesNotExpire(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	const day = "2026-09-28"
	start := time.Date(2026, 9, 28, 8, 0, 0, 0, time.UTC)
	claimed, err := store.ClaimNotification(ctx, day, start)
	if err != nil || !claimed {
		t.Fatalf("first claim = %v, %v", claimed, err)
	}

	reclaimed, err := store.ClaimNotification(ctx, day, start.Add(12*time.Hour))
	if err != nil {
		t.Fatalf("second claim error = %v", err)
	}
	if reclaimed {
		t.Fatal("automatic notification claim expired and allowed a duplicate delivery")
	}
}
