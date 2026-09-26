package learning

import (
	"testing"
	"time"

	"github.com/Loe159/lpic-daily/internal/curriculum"
)

func testCatalog() *curriculum.Catalog {
	c1031 := []curriculum.Concept{
		{ID: "lpic1.103.1.shell", ObjectiveID: "103.1", TitleFR: "shell", PedagogyOrder: 1, Active: true},
		{ID: "lpic1.103.1.path", ObjectiveID: "103.1", TitleFR: "PATH", PedagogyOrder: 2, Active: true},
	}
	c1035 := []curriculum.Concept{
		{ID: "lpic1.103.5.jobs", ObjectiveID: "103.5", TitleFR: "jobs", PedagogyOrder: 1, Active: true},
	}
	return &curriculum.Catalog{
		Certification:   "LPIC-1",
		SyllabusVersion: "5.0.0",
		Objectives: map[string]curriculum.Objective{
			"103.1": {ID: "103.1", Weight: 4, Active: true},
			"103.5": {ID: "103.5", Weight: 4, Active: true},
		},
		Graph: map[string]curriculum.ObjectiveNode{
			"103.1": {ObjectiveID: "103.1"},
			"103.5": {ObjectiveID: "103.5", HardPrerequisites: []string{"103.1"}},
		},
		Concepts: map[string]curriculum.Concept{
			c1031[0].ID: c1031[0],
			c1031[1].ID: c1031[1],
			c1035[0].ID: c1035[0],
		},
		ConceptsByObjective: map[string][]curriculum.Concept{
			"103.1": c1031,
			"103.5": c1035,
		},
		Phase1Slice: curriculum.Phase1Slice{SelectedObjectives: []string{"103.1", "103.5"}},
	}
}

func TestSchedulerStartsWithFoundation(t *testing.T) {
	scheduler, err := NewScheduler(DefaultSchedulerConfig([]string{"103.1", "103.5"}))
	if err != nil {
		t.Fatal(err)
	}
	plan, err := scheduler.Plan(testCatalog(), nil, time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if plan.New == nil || plan.New.ObjectiveID != "103.1" || plan.New.ConceptID != "lpic1.103.1.shell" {
		t.Fatalf("unexpected new selection: %#v", plan.New)
	}
}

func TestSchedulerUnlocksDependentObjectiveAfterRecall(t *testing.T) {
	now := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	events := []Event{
		{EventID: "e1", OccurredAt: now.Add(-time.Hour), ConceptID: "lpic1.103.1.shell", EvidenceKind: EvidenceRecall, Result: ResultPass},
		{EventID: "e2", OccurredAt: now.Add(-time.Hour), ConceptID: "lpic1.103.1.path", EvidenceKind: EvidenceRecall, Result: ResultPass},
	}
	config := DefaultSchedulerConfig([]string{"103.5"})
	scheduler, err := NewScheduler(config)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := scheduler.Plan(testCatalog(), events, now)
	if err != nil {
		t.Fatal(err)
	}
	if plan.New == nil || plan.New.ObjectiveID != "103.5" {
		t.Fatalf("dependent objective should be unlocked, got %#v", plan.New)
	}
}

func TestSchedulerPrioritizesDueReview(t *testing.T) {
	now := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	events := []Event{{
		EventID: "e1", OccurredAt: now.Add(-48 * time.Hour), ConceptID: "lpic1.103.1.shell",
		EvidenceKind: EvidenceExposure, Result: ResultPass,
	}}
	scheduler, err := NewScheduler(DefaultSchedulerConfig([]string{"103.1"}))
	if err != nil {
		t.Fatal(err)
	}
	plan, err := scheduler.Plan(testCatalog(), events, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Reviews) != 1 || plan.Reviews[0].ConceptID != "lpic1.103.1.shell" {
		t.Fatalf("expected due review, got %#v", plan.Reviews)
	}
}
