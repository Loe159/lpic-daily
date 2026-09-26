package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/Loe159/lpic-daily/internal/learning"
)

func TestStoreMigratesAndRoundTripsEvidence(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "progress.db")
	s, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	event := learning.Event{
		EventID:          "evidence-1",
		OccurredAt:       time.Date(2026, 9, 26, 11, 0, 0, 123, time.UTC),
		ConceptID:        "lpic1.103.1.shell",
		ObjectiveIDs:     []string{"103.1"},
		SourceItemID:     "question-shell-1",
		ActivityKind:     learning.ActivityQuestion,
		EvidenceKind:     learning.EvidenceRecall,
		Result:           learning.ResultPass,
		HighestHintLevel: 0,
		Distribution:     "fedora",
		AttemptIndex:     1,
		Metadata:         map[string]any{"answer_ms": float64(1200)},
	}
	if err := s.AppendEvidence(ctx, event); err != nil {
		t.Fatal(err)
	}
	events, err := s.EvidenceForConcept(ctx, event.ConceptID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("events=%d, want 1", len(events))
	}
	got := events[0]
	if got.EventID != event.EventID || got.EvidenceKind != event.EvidenceKind || got.Distribution != "fedora" {
		t.Fatalf("round trip mismatch: %#v", got)
	}
	if got.Metadata["answer_ms"] != float64(1200) {
		t.Fatalf("metadata mismatch: %#v", got.Metadata)
	}
}

func TestEvidenceIsAppendOnlyAtDatabaseBoundary(t *testing.T) {
	ctx := context.Background()
	s, err := Open(ctx, filepath.Join(t.TempDir(), "progress.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	event := learning.Event{
		EventID: "evidence-1", OccurredAt: time.Now().UTC(), ConceptID: "c1",
		ObjectiveIDs: []string{"103.1"}, SourceItemID: "q1",
		ActivityKind: learning.ActivityQuestion, EvidenceKind: learning.EvidenceRecall,
		Result: learning.ResultPass, Distribution: "generic", AttemptIndex: 1,
	}
	if err := s.AppendEvidence(ctx, event); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, "UPDATE mastery_evidence SET result='fail' WHERE event_id=?", event.EventID); err == nil {
		t.Fatal("expected UPDATE to be rejected")
	}
	if _, err := s.db.ExecContext(ctx, "DELETE FROM mastery_evidence WHERE event_id=?", event.EventID); err == nil {
		t.Fatal("expected DELETE to be rejected")
	}
}

func TestGamificationDoesNotWriteMasteryEvidence(t *testing.T) {
	ctx := context.Background()
	s, err := Open(ctx, filepath.Join(t.TempDir(), "progress.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if err := s.AppendGamification(ctx, GamificationEvent{
		EventID: "xp-1", OccurredAt: time.Now().UTC(), Kind: "xp", Amount: 25,
		SourceItemID: "lesson-1",
	}); err != nil {
		t.Fatal(err)
	}
	var masteryCount int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM mastery_evidence").Scan(&masteryCount); err != nil {
		t.Fatal(err)
	}
	if masteryCount != 0 {
		t.Fatalf("mastery evidence count=%d, want 0", masteryCount)
	}
	var xpCount int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM gamification_events").Scan(&xpCount); err != nil {
		t.Fatal(err)
	}
	if xpCount != 1 {
		t.Fatalf("gamification event count=%d, want 1", xpCount)
	}
}

func TestMigrationsAreIdempotent(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "progress.db")
	first, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	second, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()

	var version int
	if err := second.db.QueryRowContext(ctx, "SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 1 {
		t.Fatalf("schema version=%d, want 1", version)
	}
}

func TestOpenRejectsDatabaseAheadOfBinary(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "future.db")
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, `CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, `INSERT INTO schema_migrations(version, applied_at) VALUES (999, 'future')`); err != nil {
		t.Fatal(err)
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err := Open(ctx, path); err == nil {
		t.Fatal("expected newer schema to fail closed")
	}
}
