package study_test

import (
	"context"
	"testing"
	"time"

	lpicdaily "github.com/Loe159/lpic-daily"
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

func (store *recordingStore) AppendEvidence(_ context.Context, event learning.EvidenceEvent) error {
	store.events[event.ConceptID] = append(store.events[event.ConceptID], event)
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

func TestLaterUnhintedLabAttemptRecordsTransferConfirmation(t *testing.T) {
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
	if events[0].EvidenceKind != learning.EvidenceIndependentPractice {
		t.Fatalf("first evidence = %s, want independent-practice", events[0].EvidenceKind)
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
