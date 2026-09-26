package learning

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Loe159/lpic-daily/internal/curriculum"
)

type SchedulerConfig struct {
	ScopeObjectiveIDs        []string
	PrerequisiteMinimumStage Stage
	ReviewLimit              int
	ReviewIntervals          map[Stage]time.Duration
}

func DefaultSchedulerConfig(scope []string) SchedulerConfig {
	return SchedulerConfig{
		ScopeObjectiveIDs:        append([]string(nil), scope...),
		PrerequisiteMinimumStage: StageRecall,
		ReviewLimit:              5,
		ReviewIntervals: map[Stage]time.Duration{
			StageExposed:     24 * time.Hour,
			StageRecall:      72 * time.Hour,
			StageGuided:      72 * time.Hour,
			StageIndependent: 7 * 24 * time.Hour,
			StageTransfer:    21 * 24 * time.Hour,
		},
	}
}

type Selection struct {
	ConceptID   string
	ObjectiveID string
	TitleFR     string
	Kind        string
	Score       int
	ReasonFR    string
	DueAt       time.Time
}

type SessionPlan struct {
	Reviews []Selection
	New     *Selection
}

type Scheduler struct {
	config SchedulerConfig
}

func NewScheduler(config SchedulerConfig) (*Scheduler, error) {
	if config.PrerequisiteMinimumStage < StageExposed || config.PrerequisiteMinimumStage > StageTransfer {
		return nil, fmt.Errorf("invalid prerequisite minimum stage: %s", config.PrerequisiteMinimumStage)
	}
	if config.ReviewLimit < 0 {
		return nil, fmt.Errorf("review limit cannot be negative")
	}
	if config.ReviewIntervals == nil {
		return nil, fmt.Errorf("review intervals are required")
	}
	return &Scheduler{config: config}, nil
}

func (s *Scheduler) Plan(catalog *curriculum.Catalog, allEvents []Event, now time.Time) (SessionPlan, error) {
	scope := s.config.ScopeObjectiveIDs
	if len(scope) == 0 {
		scope = catalog.ObjectiveIDs()
	}
	concepts, err := catalog.ScopeConcepts(scope)
	if err != nil {
		return SessionPlan{}, err
	}

	eventsByConcept := make(map[string][]Event)
	for _, event := range allEvents {
		eventsByConcept[event.ConceptID] = append(eventsByConcept[event.ConceptID], event)
	}
	projections := make(map[string]Projection, len(catalog.Concepts))
	for conceptID := range catalog.Concepts {
		projections[conceptID] = Project(conceptID, eventsByConcept[conceptID])
	}

	objectiveReady := make(map[string]bool, len(catalog.Objectives))
	for objectiveID := range catalog.Objectives {
		objectiveReady[objectiveID] = s.objectiveReady(catalog, objectiveID, projections)
	}

	var reviews []Selection
	var newCandidates []Selection
	for _, concept := range concepts {
		projection := projections[concept.ID]
		objective := catalog.Objectives[concept.ObjectiveID]
		if projection.Stage == StageUnseen {
			if !hardPrerequisitesReady(catalog.Graph[concept.ObjectiveID], objectiveReady) {
				continue
			}
			recommendedReady := recommendedReadyCount(catalog.Graph[concept.ObjectiveID], objectiveReady)
			score := 500 + objective.Weight*10 + recommendedReady*5 - concept.PedagogyOrder
			reason := fmt.Sprintf("nouveau concept prêt; objectif LPIC poids %d", objective.Weight)
			if len(catalog.Graph[concept.ObjectiveID].HardPrerequisites) > 0 {
				reason += "; prérequis durs satisfaits"
			}
			if recommendedReady > 0 {
				reason += fmt.Sprintf("; %d prérequis recommandé(s) satisfait(s)", recommendedReady)
			}
			newCandidates = append(newCandidates, Selection{
				ConceptID: concept.ID, ObjectiveID: concept.ObjectiveID, TitleFR: concept.TitleFR,
				Kind: "new", Score: score, ReasonFR: reason,
			})
			continue
		}

		interval, ok := s.config.ReviewIntervals[projection.Stage]
		if !ok {
			continue
		}
		dueAt := projection.LastSuccess.Add(interval)
		if projection.LastSuccess.IsZero() {
			dueAt = projection.LastAttempt.Add(interval)
		}
		if now.Before(dueAt) {
			continue
		}
		overdueHours := int(now.Sub(dueAt).Hours())
		if overdueHours > 240 {
			overdueHours = 240
		}
		score := 1000 + overdueHours + objective.Weight*10 - int(projection.Stage)*5
		reviews = append(reviews, Selection{
			ConceptID: concept.ID, ObjectiveID: concept.ObjectiveID, TitleFR: concept.TitleFR,
			Kind: "review", Score: score, DueAt: dueAt,
			ReasonFR: fmt.Sprintf("révision due; niveau %s; objectif LPIC poids %d", projection.Stage, objective.Weight),
		})
	}

	sortSelections(reviews)
	sortSelections(newCandidates)
	if len(reviews) > s.config.ReviewLimit {
		reviews = reviews[:s.config.ReviewLimit]
	}
	var nextNew *Selection
	if len(newCandidates) > 0 {
		candidate := newCandidates[0]
		nextNew = &candidate
	}
	return SessionPlan{Reviews: reviews, New: nextNew}, nil
}

func (s *Scheduler) objectiveReady(catalog *curriculum.Catalog, objectiveID string, projections map[string]Projection) bool {
	concepts := catalog.ConceptsByObjective[objectiveID]
	if len(concepts) == 0 {
		return false
	}
	for _, concept := range concepts {
		if projections[concept.ID].Stage < s.config.PrerequisiteMinimumStage {
			return false
		}
	}
	return true
}

func hardPrerequisitesReady(node curriculum.ObjectiveNode, ready map[string]bool) bool {
	for _, dep := range node.HardPrerequisites {
		if !ready[dep] {
			return false
		}
	}
	return true
}

func recommendedReadyCount(node curriculum.ObjectiveNode, ready map[string]bool) int {
	count := 0
	for _, dep := range node.RecommendedPrerequisites {
		if ready[dep] {
			count++
		}
	}
	return count
}

func sortSelections(items []Selection) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Score != items[j].Score {
			return items[i].Score > items[j].Score
		}
		if items[i].ObjectiveID != items[j].ObjectiveID {
			return strings.Compare(items[i].ObjectiveID, items[j].ObjectiveID) < 0
		}
		return strings.Compare(items[i].ConceptID, items[j].ConceptID) < 0
	})
}
