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
	if version != 3 {
		t.Fatalf("schema version = %d, want 3", version)
	}

	const day = "2026-09-27"
	sent, err := store.NotificationSent(ctx, day)
	if err != nil {
		t.Fatalf("NotificationSent() error = %v", err)
	}
	if sent {
		t.Fatal("fresh day unexpectedly sent")
	}

	claimed, err := store.ClaimNotification(ctx, day, time.Now())
	if err != nil {
		t.Fatalf("ClaimNotification() error = %v", err)
	}
	if !claimed {
		t.Fatal("fresh day was not claimed")
	}
	claimedAgain, err := store.ClaimNotification(ctx, day, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("second ClaimNotification() error = %v", err)
	}
	if claimedAgain {
		t.Fatal("same day was claimed twice")
	}
	sent, err = store.NotificationSent(ctx, day)
	if err != nil || !sent {
		t.Fatalf("NotificationSent() = %v, %v", sent, err)
	}
}
