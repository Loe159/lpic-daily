package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/Loe159/lpic-daily/internal/activity"
	"github.com/Loe159/lpic-daily/internal/gamification"
	"github.com/Loe159/lpic-daily/internal/study"
)

func TestActivityAvailableWhenStudyPlanEmpty(t *testing.T) {
	now := time.Date(2026, 10, 19, 14, 0, 0, 0, time.UTC)
	report := &activity.Report{
		Now: now,
		Nodes: map[string]*activity.Node{
			"all":      {Key: "all", Title: "LPIC-1", Total: 10, Covered: 4, Mastered: 2},
			"exam:101": {Key: "exam:101", Parent: "all", Title: "Examen 101", Total: 10, Covered: 4, Mastered: 2},
		},
		Children: map[string][]string{"all": {"exam:101"}},
		Days:     map[string]*activity.Day{},
	}
	model := NewDashboard(study.Plan{}, gamification.Snapshot{}, report)
	if !strings.Contains(model.render(), "Aucune activité prévue") {
		t.Fatalf("empty study dashboard changed: %q", model.render())
	}
	model, exit := model.updateKey("tab")
	if exit || !model.showActivity || !strings.Contains(model.render(), "CALENDRIER") {
		t.Fatalf("tab activity not opened: exit=%v output=%q", exit, model.render())
	}
	model, exit = model.updateKey("p")
	if exit || !strings.Contains(model.render(), "PROGRESSION") {
		t.Fatalf("progression not opened: exit=%v output=%q", exit, model.render())
	}
	model, exit = model.updateKey("enter")
	if exit || model.activityNode != "exam:101" {
		t.Fatalf("exam drilldown: key=%s exit=%v", model.activityNode, exit)
	}
	model, exit = model.updateKey("esc")
	if exit || model.activityNode != "all" {
		t.Fatalf("breadcrumb navigation: key=%s exit=%v", model.activityNode, exit)
	}
	model, exit = model.updateKey("tab")
	if exit || model.showActivity || !strings.Contains(model.render(), "Ma séance") {
		t.Fatalf("tab return: exit=%v output=%q", exit, model.render())
	}
}

func TestCalendarNeverSelectsFutureDate(t *testing.T) {
	now := time.Date(2026, 10, 19, 14, 0, 0, 0, time.UTC)
	report := &activity.Report{Now: now, Nodes: map[string]*activity.Node{"all": {Key: "all", Total: 1}}, Children: map[string][]string{}, Days: map[string]*activity.Day{}}
	model := NewDashboard(study.Plan{}, gamification.Snapshot{}, report)
	model, _ = model.updateKey("tab")
	model, _ = model.updateKey("right")
	if !model.activityDate.Equal(now) {
		t.Fatalf("selected future date: %v", model.activityDate)
	}
	model, _ = model.updateKey("enter")
	if !model.activityDayOpen {
		t.Fatal("enter did not show day details")
	}
	model, _ = model.updateKey("esc")
	if model.activityDayOpen {
		t.Fatal("escape did not return to calendar")
	}
}
