package sqlite

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Loe159/lpic-daily/internal/learning"
)

func testEvent(id string, at time.Time) learning.EvidenceEvent {
	return learning.EvidenceEvent{
		EventID:          id,
		OccurredAt:       at,
		ConceptID:        "lpic1.103.5.processus-avant-arriere-plan",
		ObjectiveIDs:     []string{"103.5"},
		SourceItemID:     "lab-stuck-worker",
		ActivityKind:     learning.ActivityLab,
		EvidenceKind:     learning.EvidenceIndependentPractice,
		Result:           learning.ResultPass,
		HighestHintLevel: 0,
		SolutionRevealed: false,
		Distribution:     "fedora",
		PracticeContext:  "process-incident",
		AttemptIndex:     1,
	}
}

func TestMigrationAndEvidenceRoundTrip(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "progress.sqlite")

	store, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	version, err := store.SchemaVersion(ctx)
	if err != nil {
		t.Fatalf("SchemaVersion() error = %v", err)
	}
	if version != 6 {
		t.Fatalf("schema version = %d, want 6", version)
	}

	at := time.Date(2026, 9, 26, 12, 0, 0, 123456789, time.UTC)
	want := testEvent("event-1", at)
	if err := store.AppendEvidence(ctx, want); err != nil {
		t.Fatalf("AppendEvidence() error = %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	store, err = Open(ctx, path)
	if err != nil {
		t.Fatalf("reopen error = %v", err)
	}
	defer store.Close()

	got, err := store.EvidenceForConcept(ctx, want.ConceptID)
	if err != nil {
		t.Fatalf("EvidenceForConcept() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("evidence rows = %d, want 1", len(got))
	}
	if got[0].EventID != want.EventID || !got[0].OccurredAt.Equal(want.OccurredAt) {
		t.Fatalf("round trip = %#v, want %#v", got[0], want)
	}
	if got[0].EvidenceKind != want.EvidenceKind || got[0].Result != want.Result {
		t.Fatalf("round trip kind/result = (%s, %s), want (%s, %s)", got[0].EvidenceKind, got[0].Result, want.EvidenceKind, want.Result)
	}
	if got[0].PracticeContext != want.PracticeContext {
		t.Fatalf("round trip practice context = %q, want %q", got[0].PracticeContext, want.PracticeContext)
	}
}

func TestOpenRejectsNewerSchemaVersion(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "future.sqlite")

	store, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("initial Open() error = %v", err)
	}
	if _, err := store.db.ExecContext(ctx, "PRAGMA user_version = 999"); err != nil {
		_ = store.Close()
		t.Fatalf("set future schema version: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close future database: %v", err)
	}

	store, err = Open(ctx, path)
	if err == nil {
		_ = store.Close()
		t.Fatal("Open() unexpectedly accepted a database from a newer schema")
	}
	if !strings.Contains(err.Error(), "newer than this binary supports") {
		t.Fatalf("Open() error = %v, want newer-schema refusal", err)
	}
}

func TestEvidenceIsAppendOnlyByEventID(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	event := testEvent("event-duplicate", time.Now().UTC())
	if err := store.AppendEvidence(ctx, event); err != nil {
		t.Fatalf("first AppendEvidence() error = %v", err)
	}
	if err := store.AppendEvidence(ctx, event); err == nil {
		t.Fatal("second AppendEvidence() unexpectedly succeeded")
	}
}

func TestEvidenceBatchRollsBackAtomically(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	at := time.Date(2026, 9, 28, 9, 0, 0, 0, time.UTC)
	first := testEvent("duplicate-event", at)
	second := testEvent("duplicate-event", at.Add(time.Second))
	second.ConceptID = "lpic1.104.5.permissions-rwx-fichier-dossier"
	second.ObjectiveIDs = []string{"104.5"}

	if err := store.AppendEvidenceBatch(ctx, []learning.EvidenceEvent{first, second}); err == nil {
		t.Fatal("AppendEvidenceBatch() unexpectedly succeeded with duplicate event IDs")
	}
	for _, conceptID := range []string{first.ConceptID, second.ConceptID} {
		events, err := store.EvidenceForConcept(ctx, conceptID)
		if err != nil {
			t.Fatalf("EvidenceForConcept(%s) error = %v", conceptID, err)
		}
		if len(events) != 0 {
			t.Fatalf("batch left partial evidence for %s: %#v", conceptID, events)
		}
	}
}

func TestLabDisclosureKeepsStrongestLevelUntilCleared(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	at := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	const labID = "lpic1.104.5.shared-dropbox"
	if err := store.RecordLabDisclosure(ctx, labID, 2, false, at); err != nil {
		t.Fatalf("RecordLabDisclosure(level 2) error = %v", err)
	}
	if err := store.RecordLabDisclosure(ctx, labID, 1, false, at.Add(time.Minute)); err != nil {
		t.Fatalf("RecordLabDisclosure(level 1) error = %v", err)
	}

	disclosure, err := store.LabDisclosure(ctx, labID)
	if err != nil {
		t.Fatalf("LabDisclosure() error = %v", err)
	}
	if disclosure.HighestHintLevel != 2 || disclosure.SolutionRevealed {
		t.Fatalf("disclosure = %#v, want level 2 without solution", disclosure)
	}

	if err := store.RecordLabDisclosure(ctx, labID, 4, true, at.Add(2*time.Minute)); err != nil {
		t.Fatalf("RecordLabDisclosure(level 4) error = %v", err)
	}
	disclosure, err = store.LabDisclosure(ctx, labID)
	if err != nil {
		t.Fatalf("LabDisclosure() error = %v", err)
	}
	if disclosure.HighestHintLevel != 4 || !disclosure.SolutionRevealed {
		t.Fatalf("disclosure = %#v, want level 4 solution reveal", disclosure)
	}

	if err := store.ClearLabDisclosure(ctx, labID); err != nil {
		t.Fatalf("ClearLabDisclosure() error = %v", err)
	}
	disclosure, err = store.LabDisclosure(ctx, labID)
	if err != nil {
		t.Fatalf("LabDisclosure() after clear error = %v", err)
	}
	if disclosure.HighestHintLevel != 0 || disclosure.SolutionRevealed {
		t.Fatalf("disclosure after clear = %#v, want empty", disclosure)
	}
}

func TestMigrationSixBackfillsKnownPhase1PracticeContexts(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "legacy-progress.sqlite")

	store, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	at := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	legacy := testEvent("legacy-transfer", at)
	legacy.SourceItemID = "lpic1.103.5.stuck-worker"
	legacy.PracticeContext = "process-incident"
	if err := store.AppendEvidence(ctx, legacy); err != nil {
		_ = store.Close()
		t.Fatalf("AppendEvidence() error = %v", err)
	}
	if _, err := store.db.ExecContext(ctx, "UPDATE mastery_evidence SET practice_context = '' WHERE event_id = ?", legacy.EventID); err != nil {
		_ = store.Close()
		t.Fatalf("clear legacy practice context: %v", err)
	}
	if _, err := store.db.ExecContext(ctx, "PRAGMA user_version = 5"); err != nil {
		_ = store.Close()
		t.Fatalf("set legacy schema version: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	store, err = Open(ctx, path)
	if err != nil {
		t.Fatalf("reopen with migration 006 error = %v", err)
	}
	defer store.Close()

	events, err := store.EvidenceForConcept(ctx, legacy.ConceptID)
	if err != nil {
		t.Fatalf("EvidenceForConcept() error = %v", err)
	}
	if len(events) != 1 || events[0].PracticeContext != "process-incident" {
		t.Fatalf("backfilled evidence = %#v, want process-incident context", events)
	}
}
