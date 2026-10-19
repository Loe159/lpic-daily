// Package activity builds read-only study statistics from historical mastery evidence.
package activity

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/Loe159/lpic-daily/internal/curriculum"
	"github.com/Loe159/lpic-daily/internal/learning"
)

type EvidenceReader interface {
	EvidenceForConcept(context.Context, string) ([]learning.EvidenceEvent, error)
}

type Node struct {
	Key, Parent, ID, Title                               string
	Total, Covered, CycleComplete, Mastered, Transferred int
	Stages                                               [6]int
	LastActivity                                         time.Time
}

func (n Node) CoveragePercent() int { return percent(n.Covered, n.Total) }
func (n Node) MasteryPercent() int  { return percent(n.Mastered, n.Total) }
func percent(n, total int) int {
	if total == 0 {
		return 0
	}
	return int(math.Round(float64(n) * 100 / float64(total)))
}

type Concept struct {
	ID, ObjectiveID, Title              string
	Stage                               learning.MasteryStage
	Covered, CycleComplete              bool
	LastActivity                        time.Time
	LessonDone, QuizDone, PracticalDone bool
}

type Entry struct {
	When                                time.Time
	Kind                                learning.ActivityKind
	SourceID, ObjectiveID, ConceptTitle string
	Result                              learning.Result
	Attempts                            int
}

type Day struct {
	Date       string
	Activities []Entry
}

func (day Day) Count() int { return len(day.Activities) }

type Report struct {
	Now                                        time.Time
	Nodes                                      map[string]*Node
	Children                                   map[string][]string
	Concepts                                   map[string]Concept
	Days                                       map[string]*Day
	TotalActivities, ActiveDays, CurrentStreak int
}

// Build uses the existing SQLite evidence interface: no migration and no new user data.
func Build(ctx context.Context, bundle *curriculum.Bundle, reader EvidenceReader, now time.Time, location *time.Location) (Report, error) {
	if bundle == nil || reader == nil || now.IsZero() {
		return Report{}, fmt.Errorf("activity: curriculum, progress store and date are required")
	}
	if location == nil {
		location = time.Local
	}
	r := Report{Now: now.In(location), Nodes: map[string]*Node{}, Children: map[string][]string{}, Concepts: map[string]Concept{}, Days: map[string]*Day{}}
	r.ensure("all", "", "LPIC-1", "LPIC-1")
	knownObjectives := map[string]curriculum.Objective{}
	for _, objective := range bundle.Objectives.Objectives {
		if !objective.Active {
			continue
		}
		knownObjectives[objective.ID] = objective
		examKey := "exam:" + objective.Exam
		topicKey := "topic:" + objective.Topic
		r.ensure(examKey, "all", objective.Exam, "Examen "+objective.Exam)
		r.ensure(topicKey, examKey, objective.Topic, topicTitle(objective.Topic))
		r.ensure("objective:"+objective.ID, topicKey, objective.ID, objective.TitleFR)
	}
	activities := map[string]*Entry{}
	for _, concept := range bundle.Concepts.Concepts {
		if !concept.Active {
			continue
		}
		obj, ok := knownObjectives[concept.ObjectiveID]
		if !ok {
			continue
		}
		evidence, err := reader.EvidenceForConcept(ctx, concept.ID)
		if err != nil {
			return Report{}, fmt.Errorf("activity: read concept %s: %w", concept.ID, err)
		}
		projection, err := learning.ProjectMastery(concept.ID, evidence, learning.DefaultProjectionPolicy())
		if err != nil {
			return Report{}, fmt.Errorf("activity: project %s: %w", concept.ID, err)
		}
		progress := Concept{ID: concept.ID, ObjectiveID: concept.ObjectiveID, Title: concept.TitleFR, Stage: projection.Stage, Covered: len(evidence) > 0, LastActivity: projection.LastEvidenceAt}
		for _, ev := range evidence {
			if ev.Result == learning.ResultPass {
				switch ev.ActivityKind {
				case learning.ActivityLesson:
					progress.LessonDone = true
				case learning.ActivityQuestion:
					progress.QuizDone = true
				case learning.ActivityLab, learning.ActivityRep, learning.ActivityChallenge:
					progress.PracticalDone = true
				}
			}
			day := ev.OccurredAt.In(location).Format("2006-01-02")
			// Legacy lab checks are not separate learning sessions. Count one lab
			// per source and local day, even if :check ran several times.
			key := string(ev.ActivityKind) + "|" + ev.SourceItemID + "|" + day
			if ev.ActivityKind == learning.ActivityQuestion || ev.ActivityKind == learning.ActivityRep {
				key += fmt.Sprintf("|%d|%s", ev.AttemptIndex, ev.OccurredAt.UTC().Format(time.RFC3339Nano))
			}
			entry := activities[key]
			if entry == nil {
				entry = &Entry{When: ev.OccurredAt, Kind: ev.ActivityKind, SourceID: ev.SourceItemID, ObjectiveID: concept.ObjectiveID, ConceptTitle: concept.TitleFR, Result: ev.Result, Attempts: 1}
				activities[key] = entry
			} else {
				if ev.OccurredAt.After(entry.When) {
					entry.When = ev.OccurredAt
				}
				if resultRank(ev.Result) > resultRank(entry.Result) {
					entry.Result = ev.Result
				}
			}
		}
		progress.CycleComplete = progress.LessonDone && progress.QuizDone && progress.PracticalDone
		r.Concepts[concept.ID] = progress
		conceptKey := "concept:" + concept.ID
		r.ensure(conceptKey, "objective:"+concept.ObjectiveID, concept.ID, concept.TitleFR)
		for _, key := range []string{"all", "exam:" + obj.Exam, "topic:" + obj.Topic, "objective:" + obj.ID, conceptKey} {
			node := r.Nodes[key]
			node.Total++
			if progress.Covered {
				node.Covered++
			}
			if progress.CycleComplete {
				node.CycleComplete++
			}
			if progress.Stage >= learning.StageIndependent {
				node.Mastered++
			}
			if progress.Stage >= learning.StageTransfer {
				node.Transferred++
			}
			node.Stages[progress.Stage]++
			if progress.LastActivity.After(node.LastActivity) {
				node.LastActivity = progress.LastActivity
			}
		}
	}
	for _, ev := range activities {
		date := ev.When.In(location).Format("2006-01-02")
		day := r.Days[date]
		if day == nil {
			day = &Day{Date: date}
			r.Days[date] = day
		}
		day.Activities = append(day.Activities, *ev)
		r.TotalActivities++
	}
	r.ActiveDays = len(r.Days)
	for _, day := range r.Days {
		sort.SliceStable(day.Activities, func(i, j int) bool {
			return day.Activities[i].When.Before(day.Activities[j].When)
		})
	}
	cursor := midnight(r.Now)
	for {
		if day := r.Days[cursor.Format("2006-01-02")]; day == nil || day.Count() == 0 {
			break
		}
		r.CurrentStreak++
		cursor = cursor.AddDate(0, 0, -1)
	}
	for key := range r.Children {
		sort.SliceStable(r.Children[key], func(i, j int) bool {
			return r.Nodes[r.Children[key][i]].ID < r.Nodes[r.Children[key][j]].ID
		})
	}
	return r, nil
}

func (r *Report) ensure(key, parent, id, title string) {
	if r.Nodes[key] != nil {
		return
	}
	r.Nodes[key] = &Node{Key: key, Parent: parent, ID: id, Title: title}
	if parent != "" {
		r.Children[parent] = append(r.Children[parent], key)
	}
}

func (r Report) DayAt(date time.Time) Day {
	key := date.In(r.Now.Location()).Format("2006-01-02")
	if day := r.Days[key]; day != nil {
		return *day
	}
	return Day{Date: key}
}

func midnight(date time.Time) time.Time {
	y, m, d := date.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, date.Location())
}

func resultRank(r learning.Result) int {
	switch r {
	case learning.ResultPass:
		return 3
	case learning.ResultPartial:
		return 2
	default:
		return 1
	}
}

func topicTitle(id string) string {
	titles := map[string]string{
		"101": "Architecture du système", "102": "Installation et paquets",
		"103": "Commandes GNU et Unix", "104": "Disques et systèmes de fichiers",
		"105": "Shells et scripts", "106": "Interfaces et bureaux",
		"107": "Administration", "108": "Services essentiels",
		"109": "Réseaux", "110": "Sécurité",
	}
	if t := titles[id]; t != "" {
		return id + " · " + t
	}
	return "Domaine " + strings.TrimSpace(id)
}
