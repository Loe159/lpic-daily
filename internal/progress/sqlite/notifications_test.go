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
	if version != 6 {
		t.Fatalf("schema version = %d, want 6", version)
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

	released, err := store.ReleaseNotificationClaim(ctx, day, start)
	if err != nil {
		t.Fatalf("ReleaseNotificationClaim() error = %v", err)
	}
	if !released {
		t.Fatal("owned claim was not released")
	}
	reclaimAt := start.Add(2 * time.Minute)
	reclaimed, err := store.ClaimNotification(ctx, day, reclaimAt)
	if err != nil {
		t.Fatalf("reclaim after release error = %v", err)
	}
	if !reclaimed {
		t.Fatal("released claim could not be acquired again")
	}

	if err := store.MarkNotificationSent(ctx, day, reclaimAt, start.Add(3*time.Minute)); err != nil {
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

func TestStaleNotificationOwnerCannotReleaseOrMarkReclaimedLease(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	const day = "2026-09-30"
	start := time.Date(2026, 9, 30, 8, 0, 0, 0, time.UTC)
	claimed, err := store.ClaimNotification(ctx, day, start)
	if err != nil || !claimed {
		t.Fatalf("first claim = %v, %v", claimed, err)
	}
	reclaimedAt := start.Add(notificationClaimTTL)
	reclaimed, err := store.ClaimNotification(ctx, day, reclaimedAt)
	if err != nil || !reclaimed {
		t.Fatalf("reclaim = %v, %v", reclaimed, err)
	}

	released, err := store.ReleaseNotificationClaim(ctx, day, start)
	if err != nil {
		t.Fatalf("stale ReleaseNotificationClaim() error = %v", err)
	}
	if released {
		t.Fatal("stale owner released the newer claim")
	}
	if err := store.MarkNotificationSent(ctx, day, start, start.Add(time.Hour)); err == nil {
		t.Fatal("stale owner marked the newer claim as sent")
	}

	refreshed, err := store.RefreshNotificationClaim(
		ctx,
		day,
		reclaimedAt,
		reclaimedAt.Add(time.Minute),
	)
	if err != nil || !refreshed {
		t.Fatalf("new owner lost its claim: refreshed=%v err=%v", refreshed, err)
	}
}

func TestNotificationClaimRefreshExtendsLease(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	const day = "2026-09-29"
	start := time.Date(2026, 9, 29, 8, 0, 0, 0, time.UTC)
	claimed, err := store.ClaimNotification(ctx, day, start)
	if err != nil || !claimed {
		t.Fatalf("first claim = %v, %v", claimed, err)
	}

	refreshedAt := start.Add(3 * time.Minute)
	refreshed, err := store.RefreshNotificationClaim(ctx, day, start, refreshedAt)
	if err != nil || !refreshed {
		t.Fatalf("refresh = %v, %v", refreshed, err)
	}

	staleOwnerRefresh, err := store.RefreshNotificationClaim(
		ctx,
		day,
		start,
		start.Add(4*time.Minute),
	)
	if err != nil {
		t.Fatalf("stale owner refresh error = %v", err)
	}
	if staleOwnerRefresh {
		t.Fatal("stale claim timestamp unexpectedly refreshed current lease")
	}

	reclaimed, err := store.ClaimNotification(ctx, day, start.Add(11*time.Minute))
	if err != nil {
		t.Fatalf("reclaim before refreshed lease expiry error = %v", err)
	}
	if reclaimed {
		t.Fatal("refreshed live claim was reclaimed too early")
	}

	reclaimed, err = store.ClaimNotification(
		ctx,
		day,
		refreshedAt.Add(notificationClaimTTL),
	)
	if err != nil {
		t.Fatalf("reclaim after refreshed lease expiry error = %v", err)
	}
	if !reclaimed {
		t.Fatal("expired refreshed claim was not reclaimable")
	}
}

func TestNotificationClaimExpiresAndCanBeReclaimed(t *testing.T) {
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

	freshClaim, err := store.ClaimNotification(ctx, day, start.Add(notificationClaimTTL-time.Minute))
	if err != nil {
		t.Fatalf("fresh claim retry error = %v", err)
	}
	if freshClaim {
		t.Fatal("live notification claim was reclaimed before its lease expired")
	}

	reclaimed, err := store.ClaimNotification(ctx, day, start.Add(notificationClaimTTL))
	if err != nil {
		t.Fatalf("stale claim retry error = %v", err)
	}
	if !reclaimed {
		t.Fatal("stale notification claim was not reclaimed")
	}
}
