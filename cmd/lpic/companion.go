package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	lpicdaily "github.com/Loe159/lpic-daily"
	"github.com/Loe159/lpic-daily/internal/activity"
	"github.com/Loe159/lpic-daily/internal/bootstrap"
	"github.com/Loe159/lpic-daily/internal/content"
	"github.com/Loe159/lpic-daily/internal/curriculum"
	"github.com/Loe159/lpic-daily/internal/gamification"
	"github.com/Loe159/lpic-daily/internal/lab"
	"github.com/Loe159/lpic-daily/internal/learning"
	progresssqlite "github.com/Loe159/lpic-daily/internal/progress/sqlite"
	"github.com/Loe159/lpic-daily/internal/study"
	"github.com/Loe159/lpic-daily/internal/terminal"
	lpicui "github.com/Loe159/lpic-daily/internal/tui"
)

const companionSchemaVersion = 1

// CompanionStatus is a versioned, read-only local interface for desktop integrations.
// All scheduling and mastery calculations remain in LPIC Daily.
type CompanionStatus struct {
	SchemaVersion int                 `json:"schemaVersion"`
	GeneratedAt   time.Time           `json:"generatedAt"`
	Session       CompanionSession    `json:"session"`
	Progress      CompanionProgress   `json:"progress"`
	Activity      CompanionActivity   `json:"activity"`
}

type CompanionSession struct {
	Remaining int            `json:"remaining"`
	Reviews   int            `json:"reviews"`
	Practice  int            `json:"practice"`
	New       int            `json:"new"`
	Next      *CompanionNext `json:"next,omitempty"`
}

type CompanionNext struct {
	ConceptID   string         `json:"conceptId"`
	ObjectiveID string         `json:"objectiveId"`
	Title       string         `json:"title"`
	Kind        string         `json:"kind"`
	Action      lpicui.Action `json:"-"`
	ActionKind  string         `json:"actionKind"`
}

type CompanionProgress struct {
	TotalConcepts   int `json:"totalConcepts"`
	CoveredConcepts int `json:"coveredConcepts"`
	MasteredConcepts int `json:"masteredConcepts"`
	CoveragePercent int `json:"coveragePercent"`
	MasteryPercent  int `json:"masteryPercent"`
}

type CompanionDay struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type CompanionActivity struct {
	StreakDays     int            `json:"streakDays"`
	CompletedToday int            `json:"completedToday"`
	XP             int            `json:"xp"`
	Last14Days     []CompanionDay `json:"last14Days"`
}

func loadCompanionPlan(ctx context.Context, store *progresssqlite.Store, now time.Time) (study.Plan, *curriculum.Bundle, []lab.Lab, error) {
	curriculumBundle, err := curriculum.Load(lpicdaily.BuiltinFS)
	if err != nil {
		return study.Plan{}, nil, nil, fmt.Errorf("load curriculum: %w", err)
	}
	contentBundle, err := content.Load(lpicdaily.BuiltinFS)
	if err != nil {
		return study.Plan{}, nil, nil, fmt.Errorf("load content: %w", err)
	}
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		return study.Plan{}, nil, nil, fmt.Errorf("load labs: %w", err)
	}
	plan, err := study.BuildPlan(ctx, study.PlanInput{
		Now: now, Curriculum: curriculumBundle, Content: contentBundle,
		Labs: labs, Evidence: store, Policy: learning.DefaultSessionPolicy(),
	})
	if err != nil {
		return study.Plan{}, nil, nil, fmt.Errorf("build session: %w", err)
	}
	return plan, curriculumBundle, labs, nil
}

func makeCompanionStatus(plan study.Plan, report activity.Report, game gamification.Snapshot, now time.Time) CompanionStatus {
	result := CompanionStatus{
		SchemaVersion: companionSchemaVersion,
		GeneratedAt:   now,
		Session:       CompanionSession{Remaining: len(plan.Items)},
		Activity: CompanionActivity{
			StreakDays: game.CurrentStreakDays,
			CompletedToday: report.DayAt(now).Count(),
			XP: game.XP,
			Last14Days: make([]CompanionDay, 0, 14),
		},
	}
	for _, item := range plan.Items {
		switch item.Kind {
		case learning.SessionReview:
			result.Session.Reviews++
		case learning.SessionPractice:
			result.Session.Practice++
		case learning.SessionNew:
			result.Session.New++
		}
	}
	if len(plan.Items) > 0 {
		first := plan.Items[0]
		action := lpicui.DefaultAction(&first)
		result.Session.Next = &CompanionNext{
			ConceptID: first.ConceptID,
			ObjectiveID: first.ObjectiveID,
			Title: first.ConceptTitleFR,
			Kind: string(first.Kind),
			ActionKind: string(action.Kind),
		}
	}
	if all := report.Nodes["all"]; all != nil {
		result.Progress = CompanionProgress{
			TotalConcepts: all.Total,
			CoveredConcepts: all.Covered,
			MasteredConcepts: all.Mastered,
			CoveragePercent: all.CoveragePercent(),
			MasteryPercent: all.MasteryPercent(),
		}
	}
	for i := 13; i >= 0; i-- {
		day := now.AddDate(0, 0, -i)
		result.Activity.Last14Days = append(result.Activity.Last14Days, CompanionDay{
			Date: day.Format("2006-01-02"), Count: report.DayAt(day).Count(),
		})
	}
	return result
}

func runCompanionStatus(args []string, stdout io.Writer) error {
	if len(args) != 1 || args[0] != "--json" {
		return fmt.Errorf("usage: lpic status --json")
	}
	ctx := context.Background()
	now := time.Now()
	store, err := openProgressStore(ctx)
	if err != nil {
		return err
	}
	defer store.Close()

	plan, curriculumBundle, _, err := loadCompanionPlan(ctx, store, now)
	if err != nil {
		return err
	}
	game, err := loadGamificationSnapshot(ctx, store, now)
	if err != nil {
		return err
	}
	report, err := activity.Build(ctx, curriculumBundle, store, now, time.Local)
	if err != nil {
		return fmt.Errorf("build activity: %w", err)
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(makeCompanionStatus(plan, report, game, now))
}

// runCompanionLaunch recomputes the recommendation at click time, preventing a
// stale popup from bypassing prerequisites or launching an outdated lab.
func runCompanionLaunch(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) != 1 || args[0] != "--next" {
		return fmt.Errorf("usage: lpic launch --next")
	}
	inputFile, inOK := stdin.(*os.File)
	outputFile, outOK := stdout.(*os.File)
	if !inOK || !outOK || !terminal.IsTerminal(inputFile) || !terminal.IsTerminal(outputFile) {
		return fmt.Errorf("lpic launch --next requires an interactive terminal")
	}
	ctx := context.Background()
	if err := bootstrap.EnsureFirstRun(ctx, lpicdaily.BuiltinFS, bootstrap.Options{
		Stdin: stdin, Stdout: stdout, Stderr: stderr,
	}); err != nil {
		return fmt.Errorf("initial LPIC Daily setup: %w", err)
	}
	store, err := openProgressStore(ctx)
	if err != nil {
		return err
	}
	plan, _, labs, planErr := loadCompanionPlan(ctx, store, time.Now())
	closeErr := store.Close()
	if planErr != nil {
		return planErr
	}
	if closeErr != nil {
		return closeErr
	}
	if len(plan.Items) == 0 {
		fmt.Fprintln(stdout, "Aucune activité recommandée actuellement.")
		return nil
	}
	action := lpicui.DefaultAction(&plan.Items[0])
	switch action.Kind {
	case lpicui.ActionLesson:
		return runLearn([]string{action.ID}, stdin, stdout)
	case lpicui.ActionQuestion:
		return runQuestion([]string{action.ID}, stdin, stdout)
	case lpicui.ActionLab:
		authored, err := findLab(labs, action.ID)
		if err != nil {
			return err
		}
		return runDashboardLab(ctx, authored, stdin, stdout, stderr)
	default:
		return fmt.Errorf("aucune action disponible pour la prochaine activité")
	}
}
