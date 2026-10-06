package study_test

import (
	"context"
	"testing"
	"time"

	lpicdaily "github.com/Loe159/lpic-daily"
	"github.com/Loe159/lpic-daily/internal/checker"
	"github.com/Loe159/lpic-daily/internal/content"
	"github.com/Loe159/lpic-daily/internal/lab"
	"github.com/Loe159/lpic-daily/internal/learning"
	"github.com/Loe159/lpic-daily/internal/study"
)

type recordingStore struct {
	events map[string][]learning.EvidenceEvent
}

func newRecordingStore() *recordingStore {
	return &recordingStore{events: map[string][]learning.EvidenceEvent{}}
}

func (store *recordingStore) EvidenceForConcept(_ context.Context, conceptID string) ([]learning.EvidenceEvent, error) {
	return append([]learning.EvidenceEvent(nil), store.events[conceptID]...), nil
}

func (store *recordingStore) AppendEvidenceBatch(_ context.Context, events []learning.EvidenceEvent) error {
	for _, event := range events {
		store.events[event.ConceptID] = append(store.events[event.ConceptID], event)
	}
	return nil
}

func TestRecordLessonAndQuestionCreateAppendOnlyEvidence(t *testing.T) {
	bundle, err := content.Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("content.Load() error = %v", err)
	}
	lesson, ok := bundle.LessonByID("lpic1.103.1.lesson.shell-sequences")
	if !ok {
		t.Fatal("focused lesson not found")
	}
	question, ok := bundle.QuestionByID("lpic1.103.1.q.sequence-and")
	if !ok {
		t.Fatal("question not found")
	}

	store := newRecordingStore()
	at := time.Date(2026, 9, 26, 20, 0, 0, 0, time.UTC)
	if err := study.RecordLesson(context.Background(), store, lesson, at); err != nil {
		t.Fatalf("RecordLesson() error = %v", err)
	}
	if err := study.RecordQuestion(context.Background(), store, question, true, at.Add(time.Minute)); err != nil {
		t.Fatalf("RecordQuestion() error = %v", err)
	}

	conceptID := lesson.ConceptIDs[0]
	events := store.events[conceptID]
	if len(events) != 2 {
		t.Fatalf("events = %d, want 2", len(events))
	}
	if events[0].ActivityKind != learning.ActivityLesson || events[0].EvidenceKind != learning.EvidenceExposure {
		t.Fatalf("lesson event = %#v", events[0])
	}
	if events[1].ActivityKind != learning.ActivityQuestion || events[1].EvidenceKind != learning.EvidenceRecognition {
		t.Fatalf("question event = %#v", events[1])
	}
	if events[0].EventID == "" || events[1].EventID == "" || events[0].EventID == events[1].EventID {
		t.Fatalf("event IDs are not unique: %#v", events)
	}
}

func TestQuestionAttemptsIncrementPerSourceAndConcept(t *testing.T) {
	bundle, err := content.Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("content.Load() error = %v", err)
	}
	question, ok := bundle.QuestionByID("lpic1.103.1.q.export")
	if !ok {
		t.Fatal("question not found")
	}

	store := newRecordingStore()
	at := time.Date(2026, 9, 26, 20, 0, 0, 0, time.UTC)
	if err := study.RecordQuestion(context.Background(), store, question, false, at); err != nil {
		t.Fatalf("first RecordQuestion() error = %v", err)
	}
	if err := study.RecordQuestion(context.Background(), store, question, true, at.Add(time.Minute)); err != nil {
		t.Fatalf("second RecordQuestion() error = %v", err)
	}

	events := store.events[question.ConceptIDs[0]]
	if len(events) != 2 || events[0].AttemptIndex != 1 || events[1].AttemptIndex != 2 {
		t.Fatalf("attempts = %#v", events)
	}
	if events[0].Result != learning.ResultFail || events[1].Result != learning.ResultPass {
		t.Fatalf("results = %#v", events)
	}
}

func TestRepeatedUnhintedLabAttemptStaysIndependent(t *testing.T) {
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("lab.LoadAll() error = %v", err)
	}
	var shared lab.Lab
	for _, authored := range labs {
		if authored.Definition.ID == "lpic1.104.5.shared-dropbox" {
			shared = authored
			break
		}
	}
	if shared.Definition.ID == "" {
		t.Fatal("shared-dropbox lab not found")
	}

	store := newRecordingStore()
	start := time.Date(2026, 9, 24, 20, 0, 0, 0, time.UTC)
	if err := study.RecordLab(context.Background(), store, shared, 0, start); err != nil {
		t.Fatalf("first RecordLab() error = %v", err)
	}
	if err := study.RecordLab(context.Background(), store, shared, 0, start.Add(25*time.Hour)); err != nil {
		t.Fatalf("second RecordLab() error = %v", err)
	}

	conceptID := shared.Definition.ConceptIDs[0]
	events := store.events[conceptID]
	if len(events) != 2 {
		t.Fatalf("events = %d, want 2", len(events))
	}
	if events[0].EvidenceKind != learning.EvidenceIndependentPractice ||
		events[1].EvidenceKind != learning.EvidenceIndependentPractice {
		t.Fatalf("evidence = %#v, want repeated independent-practice", events)
	}
	projection, err := learning.ProjectMastery(conceptID, events, learning.DefaultProjectionPolicy())
	if err != nil {
		t.Fatalf("ProjectMastery() error = %v", err)
	}
	if projection.Stage != learning.StageIndependent {
		t.Fatalf("stage = %s, want independent", projection.Stage)
	}
}

func TestDifferentLabIDWithSamePracticeContextStaysIndependent(t *testing.T) {
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("lab.LoadAll() error = %v", err)
	}
	var shared lab.Lab
	for _, authored := range labs {
		if authored.Definition.ID == "lpic1.104.5.shared-dropbox" {
			shared = authored
			break
		}
	}
	if shared.Definition.ID == "" {
		t.Fatal("shared-dropbox lab not found")
	}

	sameContext := shared
	sameContext.Definition.ID = "lpic1.104.5.superficial-variant"

	store := newRecordingStore()
	start := time.Date(2026, 9, 24, 20, 0, 0, 0, time.UTC)
	if err := study.RecordLab(context.Background(), store, shared, 0, start); err != nil {
		t.Fatalf("first RecordLab() error = %v", err)
	}
	if err := study.RecordLab(context.Background(), store, sameContext, 0, start.Add(25*time.Hour)); err != nil {
		t.Fatalf("second RecordLab() error = %v", err)
	}

	events := store.events[shared.Definition.ConceptIDs[0]]
	if len(events) != 2 || events[1].EvidenceKind != learning.EvidenceIndependentPractice {
		t.Fatalf("evidence = %#v, want second independent-practice", events)
	}
}

func TestDifferentUnhintedLabContextRecordsTransfer(t *testing.T) {
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("lab.LoadAll() error = %v", err)
	}
	var shared lab.Lab
	for _, authored := range labs {
		if authored.Definition.ID == "lpic1.104.5.shared-dropbox" {
			shared = authored
			break
		}
	}
	if shared.Definition.ID == "" {
		t.Fatal("shared-dropbox lab not found")
	}

	transferContext := shared
	transferContext.Definition.ID = "lpic1.104.5.shared-dropbox-transfer"
	transferContext.Definition.PracticeContext = "team-share-audit-test"

	store := newRecordingStore()
	start := time.Date(2026, 9, 24, 20, 0, 0, 0, time.UTC)
	if err := study.RecordLab(context.Background(), store, shared, 0, start); err != nil {
		t.Fatalf("first RecordLab() error = %v", err)
	}
	if err := study.RecordLab(context.Background(), store, transferContext, 0, start.Add(25*time.Hour)); err != nil {
		t.Fatalf("transfer RecordLab() error = %v", err)
	}

	conceptID := shared.Definition.ConceptIDs[0]
	events := store.events[conceptID]
	if len(events) != 2 {
		t.Fatalf("events = %d, want 2", len(events))
	}
	if events[1].EvidenceKind != learning.EvidenceTransfer {
		t.Fatalf("second evidence = %s, want transfer", events[1].EvidenceKind)
	}
	projection, err := learning.ProjectMastery(conceptID, events, learning.DefaultProjectionPolicy())
	if err != nil {
		t.Fatalf("ProjectMastery() error = %v", err)
	}
	if projection.Stage != learning.StageTransfer {
		t.Fatalf("stage = %s, want transfer", projection.Stage)
	}
}

func TestLabHintLevelControlsEffectivePracticalEvidence(t *testing.T) {
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("lab.LoadAll() error = %v", err)
	}
	var shared lab.Lab
	for _, authored := range labs {
		if authored.Definition.ID == "lpic1.104.5.shared-dropbox" {
			shared = authored
			break
		}
	}
	if shared.Definition.ID == "" {
		t.Fatal("shared-dropbox lab not found")
	}

	store := newRecordingStore()
	at := time.Date(2026, 9, 26, 20, 0, 0, 0, time.UTC)
	if err := study.RecordLab(context.Background(), store, shared, 3, at); err != nil {
		t.Fatalf("RecordLab() error = %v", err)
	}

	event := store.events[shared.Definition.ConceptIDs[0]][0]
	if event.EvidenceKind != learning.EvidenceIndependentPractice || event.HighestHintLevel != 3 {
		t.Fatalf("stored event = %#v", event)
	}
	if got := learning.EffectiveEvidenceKind(event); got != learning.EvidenceGuidedPractice {
		t.Fatalf("effective evidence = %s, want guided-practice", got)
	}
}

func TestFailedLabAttemptIsStoredWithoutAdvancingMastery(t *testing.T) {
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("lab.LoadAll() error = %v", err)
	}
	var shared lab.Lab
	for _, authored := range labs {
		if authored.Definition.ID == "lpic1.104.5.shared-dropbox" {
			shared = authored
			break
		}
	}
	if shared.Definition.ID == "" {
		t.Fatal("shared-dropbox lab not found")
	}

	store := newRecordingStore()
	at := time.Date(2026, 9, 28, 12, 30, 0, 0, time.UTC)
	if err := study.RecordLabAttempt(
		context.Background(),
		store,
		shared,
		learning.ResultFail,
		0,
		at,
	); err != nil {
		t.Fatalf("RecordLabAttempt() error = %v", err)
	}

	conceptID := shared.Definition.ConceptIDs[0]
	events := store.events[conceptID]
	if len(events) != 1 || events[0].Result != learning.ResultFail {
		t.Fatalf("events = %#v, want one failed lab attempt", events)
	}
	projection, err := learning.ProjectMastery(conceptID, events, learning.DefaultProjectionPolicy())
	if err != nil {
		t.Fatalf("ProjectMastery() error = %v", err)
	}
	if projection.Stage != learning.StageUnseen || projection.Failures != 1 {
		t.Fatalf("projection = %#v, want unseen with one failure", projection)
	}
}

func TestLabConceptResultsPreserveMixedCheckOutcomes(t *testing.T) {
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("lab.LoadAll() error = %v", err)
	}
	var shared lab.Lab
	for _, authored := range labs {
		if authored.Definition.ID == "lpic1.104.5.shared-dropbox" {
			shared = authored
			break
		}
	}
	if shared.Definition.ID == "" {
		t.Fatal("shared-dropbox lab not found")
	}

	results := make([]checker.Result, len(shared.Definition.Checks))
	for index := range results {
		results[index] = checker.Result{CheckID: "check", Pass: true}
	}
	failedIndex := -1
	for index, check := range shared.Definition.Checks {
		if check.Type == "file-mode" && check.Path == "/srv/shared" {
			failedIndex = index
			break
		}
	}
	if failedIndex < 0 {
		t.Fatal("shared directory mode check not found")
	}
	results[failedIndex].Pass = false

	conceptResults, err := study.LabConceptResults(shared, results)
	if err != nil {
		t.Fatalf("LabConceptResults() error = %v", err)
	}
	if got := conceptResults["lpic1.104.5.permissions-rwx-fichier-dossier"]; got != learning.ResultPass {
		t.Fatalf("permissions result = %s, want pass", got)
	}
	if got := conceptResults["lpic1.104.5.sticky-bit"]; got != learning.ResultPartial {
		t.Fatalf("sticky result = %s, want partial", got)
	}
	if got := conceptResults["lpic1.104.5.proprietaire-groupe"]; got != learning.ResultPass {
		t.Fatalf("ownership result = %s, want pass", got)
	}

	store := newRecordingStore()
	at := time.Date(2026, 9, 28, 18, 0, 0, 0, time.UTC)
	if err := study.RecordLabConceptResults(
		context.Background(),
		store,
		shared,
		conceptResults,
		0,
		at,
	); err != nil {
		t.Fatalf("RecordLabConceptResults() error = %v", err)
	}
	permissionEvents := store.events["lpic1.104.5.permissions-rwx-fichier-dossier"]
	stickyEvents := store.events["lpic1.104.5.sticky-bit"]
	if len(permissionEvents) != 1 || permissionEvents[0].Result != learning.ResultPass {
		t.Fatalf("permission events = %#v, want pass", permissionEvents)
	}
	if len(stickyEvents) != 1 || stickyEvents[0].Result != learning.ResultPartial {
		t.Fatalf("sticky events = %#v, want partial", stickyEvents)
	}
}

func TestGeneratedExam101CommandEvidenceRemainsGuided(t *testing.T) {
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("lab.LoadAll() error = %v", err)
	}
	var diagnostic, transfer lab.Lab
	for _, authored := range labs {
		switch authored.Definition.ID {
		case "lpic1.103.2.flux-texte-ligne-octet.standalone-diagnostic":
			diagnostic = authored
		case "lpic1.103.2.flux-texte-ligne-octet.standalone-transfer":
			transfer = authored
		}
	}
	if diagnostic.Definition.ID == "" || transfer.Definition.ID == "" {
		t.Fatal("generated 103.2 standalone labs not found")
	}
	conceptID := "lpic1.103.2.flux-texte-ligne-octet"
	store := newRecordingStore()
	start := time.Date(2026, 10, 6, 8, 0, 0, 0, time.UTC)
	if err := study.RecordLab(context.Background(), store, diagnostic, 0, start); err != nil {
		t.Fatalf("diagnostic RecordLab() error = %v", err)
	}
	if err := study.RecordLab(context.Background(), store, transfer, 0, start.Add(25*time.Hour)); err != nil {
		t.Fatalf("transfer RecordLab() error = %v", err)
	}
	events := store.events[conceptID]
	if len(events) != 2 {
		t.Fatalf("events = %#v, want two generated practice events", events)
	}
	for index, event := range events {
		if event.EvidenceKind != learning.EvidenceGuidedPractice {
			t.Fatalf("event %d evidence = %s, want guided-practice", index+1, event.EvidenceKind)
		}
	}
}

func TestGeneratedExam101ConceptWithoutRuntimeCommandEvidenceStaysGuided(t *testing.T) {
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("lab.LoadAll() error = %v", err)
	}
	var diagnostic lab.Lab
	for _, authored := range labs {
		if authored.Definition.ID == "lpic1.101.2.bios-versus-uefi.standalone-diagnostic" {
			diagnostic = authored
			break
		}
	}
	if diagnostic.Definition.ID == "" {
		t.Fatal("generated 101.2 standalone diagnostic lab not found")
	}
	conceptID := "lpic1.101.2.bios-versus-uefi"
	store := newRecordingStore()
	if err := study.RecordLab(
		context.Background(),
		store,
		diagnostic,
		0,
		time.Date(2026, 10, 6, 8, 0, 0, 0, time.UTC),
	); err != nil {
		t.Fatalf("RecordLab() error = %v", err)
	}
	event := store.events[conceptID][0]
	if event.EvidenceKind != learning.EvidenceGuidedPractice {
		t.Fatalf("conceptual generated evidence = %s, want guided-practice", event.EvidenceKind)
	}
}
