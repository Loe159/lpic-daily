package activity

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Loe159/lpic-daily/internal/curriculum"
	"github.com/Loe159/lpic-daily/internal/learning"
)

type memoryEvidence map[string][]learning.EvidenceEvent

func (m memoryEvidence) EvidenceForConcept(_ context.Context, id string) ([]learning.EvidenceEvent, error) {
	return m[id], nil
}

func fixtureEvent(id, concept, source string, kind learning.ActivityKind, evidence learning.EvidenceKind, result learning.Result, when time.Time, attempt int) learning.EvidenceEvent {
	return learning.EvidenceEvent{
		EventID: id, ConceptID: concept, ObjectiveIDs: []string{"103.1"}, SourceItemID: source,
		ActivityKind: kind, EvidenceKind: evidence, Result: result, OccurredAt: when,
		Distribution: "fedora", PracticeContext: "scenario", AttemptIndex: attempt,
	}
}

func fixtureBundle() *curriculum.Bundle {
	return &curriculum.Bundle{
		Objectives: curriculum.ObjectivesFile{Objectives: []curriculum.Objective{
			{ID: "103.1", Exam: "101", Topic: "103", Active: true, TitleFR: "Shell"},
		}},
		Concepts: curriculum.ConceptsFile{Concepts: []curriculum.Concept{
			{ID: "shell", ObjectiveID: "103.1", TitleFR: "Variables", Active: true},
			{ID: "child", ObjectiveID: "103.1", TitleFR: "Enfants", Active: true},
			{ID: "new", ObjectiveID: "103.1", TitleFR: "Inconnu", Active: true},
		}},
	}
}

func TestBuildCountsPhysicalLabOnceAndSeparatesCoverageMastery(t *testing.T) {
	loc := time.FixedZone("Europe/Paris", 2*60*60)
	when := time.Date(2026, 10, 19, 11, 0, 0, 0, loc)
	evidence := memoryEvidence{
		"shell": {
			fixtureEvent("course", "shell", "course-1", learning.ActivityLesson, learning.EvidenceExposure, learning.ResultPass, when, 1),
			fixtureEvent("question", "shell", "question-1", learning.ActivityQuestion, learning.EvidenceRecall, learning.ResultPass, when.Add(time.Minute), 1),
			fixtureEvent("first-check", "shell", "scenario-1", learning.ActivityLab, learning.EvidenceIndependentPractice, learning.ResultFail, when.Add(2*time.Minute), 1),
			fixtureEvent("second-check", "shell", "scenario-1", learning.ActivityLab, learning.EvidenceIndependentPractice, learning.ResultPass, when.Add(4*time.Minute), 2),
		},
		"child": {
			fixtureEvent("same-lab", "child", "scenario-1", learning.ActivityLab, learning.EvidenceIndependentPractice, learning.ResultPass, when.Add(4*time.Minute), 1),
		},
	}
	r, err := Build(context.Background(), fixtureBundle(), evidence, when.Add(5*time.Minute), loc)
	if err != nil {
		t.Fatal(err)
	}
	root := r.Nodes["all"]
	if root.Total != 3 || root.Covered != 2 || root.Mastered != 2 || root.CycleComplete != 1 {
		t.Fatalf("unexpected totals: %+v", *root)
	}
	if root.CoveragePercent() != 67 || root.MasteryPercent() != 67 {
		t.Fatalf("coverage=%d mastery=%d", root.CoveragePercent(), root.MasteryPercent())
	}
	if got := r.DayAt(when).Count(); got != 3 {
		t.Fatalf("activity count=%d, want course+quiz+lab=3", got)
	}
	if r.TotalActivities != 3 || r.ActiveDays != 1 || r.CurrentStreak != 1 {
		t.Fatalf("counters=%+v", r)
	}
	if child := r.Nodes["concept:child"]; child.Total != 1 || child.Covered != 1 || child.Mastered != 1 || child.CycleComplete != 0 {
		t.Fatalf("child node=%+v", *child)
	}
	for _, key := range []string{"all", "exam:101", "topic:103", "objective:103.1", "concept:shell"} {
		if r.Nodes[key] == nil {
			t.Errorf("missing navigation node %s", key)
		}
	}
}

func TestBuildUsesLocalDayAndCountsFailedAttemptAsActivityNotMastery(t *testing.T) {
	loc := time.FixedZone("UTC+2", 2*60*60)
	utc := time.Date(2026, 10, 18, 22, 30, 0, 0, time.UTC)
	evidence := memoryEvidence{
		"shell": {fixtureEvent("failed", "shell", "lab-failed", learning.ActivityLab, learning.EvidenceIndependentPractice, learning.ResultFail, utc, 1)},
	}
	r, err := Build(context.Background(), fixtureBundle(), evidence, time.Date(2026, 10, 19, 12, 0, 0, 0, loc), loc)
	if err != nil {
		t.Fatal(err)
	}
	if r.DayAt(time.Date(2026, 10, 19, 12, 0, 0, 0, loc)).Count() != 1 {
		t.Fatal("UTC night activity should belong to local October 19")
	}
	if r.Nodes["all"].Mastered != 0 || r.Nodes["all"].Covered != 1 {
		t.Fatalf("unexpected mastery or coverage: %+v", r.Nodes["all"])
	}
}

type failingEvidence struct{}

func (failingEvidence) EvidenceForConcept(context.Context, string) ([]learning.EvidenceEvent, error) {
	return nil, errors.New("disk error")
}
func TestBuildPropagatesReadErrors(t *testing.T) {
	if _, err := Build(context.Background(), fixtureBundle(), failingEvidence{}, time.Now(), time.Local); err == nil {
		t.Fatal("expected storage read error")
	}
}
