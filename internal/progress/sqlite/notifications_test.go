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
	if version != 2 {
		t.Fatalf("schema version = %d, want 2", version)
	}

	const day = "2026-09-27"
	sent, err := store.NotificationSent(ctx, day)
	if err != nil {
		t.Fatalf("NotificationSent() error = %v", err)
	}
	if sent {
		t.Fatal("fresh day unexpectedly sent")
	}

	if err := store.MarkNotificationSent(ctx, day, time.Now()); err != nil {
		t.Fatalf("MarkNotificationSent() error = %v", err)
	}
	sent, err = store.NotificationSent(ctx, day)
	if err != nil || !sent {
		t.Fatalf("NotificationSent() = %v, %v", sent, err)
	}
}
