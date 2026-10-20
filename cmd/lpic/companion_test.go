package main

import (
	"testing"
	"time"

	"github.com/Loe159/lpic-daily/internal/activity"
	"github.com/Loe159/lpic-daily/internal/gamification"
	"github.com/Loe159/lpic-daily/internal/learning"
	"github.com/Loe159/lpic-daily/internal/study"
)

func TestMakeCompanionStatus(t *testing.T) {
	now := time.Date(2026, time.October, 21, 14, 30, 0, 0, time.Local)
	plan := study.Plan{Items: []study.Item{
		{
			ConceptID: "test-1", ObjectiveID: "104.5", ConceptTitleFR: "Permissions",
			Kind: learning.SessionNew, RecommendedLessonID: "lesson-1",
		},
		{
			ConceptID: "test-2", ObjectiveID: "103.1", ConceptTitleFR: "Shell",
			Kind: learning.SessionReview, RecommendedQuestionID: "question-1",
		},
	}}
	report := activity.Report{
		Now: now,
		Nodes: map[string]*activity.Node{
			"all": {Total: 10, Covered: 6, Mastered: 3},
		},
		Days: map[string]*activity.Day{
			now.Format("2006-01-02"): {
				Date:       now.Format("2006-01-02"),
				Activities: []activity.Entry{{When: now}, {When: now}},
			},
		},
	}
	game := gamification.Snapshot{XP: 420, CurrentStreakDays: 5}
	status := makeCompanionStatus(plan, report, game, now)

	if status.SchemaVersion != companionSchemaVersion {
		t.Fatalf("schema version = %d", status.SchemaVersion)
	}
	if status.Session.Remaining != 2 || status.Session.Reviews != 1 || status.Session.New != 1 {
		t.Fatalf("unexpected session counters: %+v", status.Session)
	}
	if status.Session.Next == nil || status.Session.Next.ActionKind != "lesson" || status.Session.Next.ObjectiveID != "104.5" {
		t.Fatalf("unexpected next action: %+v", status.Session.Next)
	}
	if status.Progress.CoveragePercent != 60 || status.Progress.MasteryPercent != 30 {
		t.Fatalf("unexpected mastery counters: %+v", status.Progress)
	}
	if status.Activity.CompletedToday != 2 || status.Activity.StreakDays != 5 || status.Activity.XP != 420 {
		t.Fatalf("unexpected activity: %+v", status.Activity)
	}
	if len(status.Activity.Last14Days) != 14 || status.Activity.Last14Days[13].Count != 2 {
		t.Fatalf("unexpected heatmap: %+v", status.Activity.Last14Days)
	}
}

func TestMakeCompanionStatusEmpty(t *testing.T) {
	now := time.Date(2026, time.October, 21, 14, 30, 0, 0, time.Local)
	status := makeCompanionStatus(
		study.Plan{},
		activity.Report{Now: now, Days: map[string]*activity.Day{}},
		gamification.Snapshot{}, now,
	)
	if status.Session.Next != nil || status.Session.Remaining != 0 {
		t.Fatalf("empty plan must not expose a launchable activity: %+v", status.Session)
	}
	if len(status.Activity.Last14Days) != 14 {
		t.Fatalf("empty history must still provide 14 days")
	}
}
