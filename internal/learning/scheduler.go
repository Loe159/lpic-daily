package learning

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/Loe159/lpic-daily/internal/curriculum"
)

type SessionItemKind string

const (
	SessionReview   SessionItemKind = "review"
	SessionPractice SessionItemKind = "practice"
	SessionNew      SessionItemKind = "new"
)

type SessionItem struct {
	ConceptID   string
	ObjectiveID string
	Kind        SessionItemKind
	ReasonCode  string
	ReasonFR    string
	DueAt       time.Time
}

type Session struct {
	GeneratedAt time.Time
	Items       []SessionItem
}

type SessionPolicy struct {
	MaxReviews      int
	MaxNewConcepts  int
	ReviewIntervals map[MasteryStage]time.Duration
}

func DefaultSessionPolicy() SessionPolicy {
	return SessionPolicy{
		MaxReviews:     4,
		MaxNewConcepts: 1,
		ReviewIntervals: map[MasteryStage]time.Duration{
			StageExposed:     24 * time.Hour,
			StageRecall:      72 * time.Hour,
			StageGuided:      72 * time.Hour,
			StageIndependent: 7 * 24 * time.Hour,
			StageTransfer:    21 * 24 * time.Hour,
		},
	}
}

type SessionInput struct {
	Now                time.Time
	Bundle             *curriculum.Bundle
	Projections        map[string]MasteryProjection
	ObjectiveReadiness map[string]bool
	ScopeObjectives    []string
	Policy             SessionPolicy
}

func BuildSession(input SessionInput) (Session, error) {
	if input.Bundle == nil {
		return Session{}, fmt.Errorf("bundle is required")
	}
	if input.Now.IsZero() {
		return Session{}, fmt.Errorf("current time is required")
	}
	if input.Policy.MaxReviews < 0 || input.Policy.MaxNewConcepts < 0 {
		return Session{}, fmt.Errorf("session limits cannot be negative")
	}

	scope := make(map[string]bool)
	for _, objectiveID := range input.ScopeObjectives {
		scope[objectiveID] = true
	}
	inScope := func(objectiveID string) bool {
		return len(scope) == 0 || scope[objectiveID]
	}

	conceptCountByObjective := make(map[string]int)
	for _, concept := range input.Bundle.Concepts.Concepts {
		if concept.Active && inScope(concept.ObjectiveID) {
			conceptCountByObjective[concept.ObjectiveID]++
		}
	}
	objectivePriority := make(map[string]float64)
	for _, objective := range input.Bundle.Objectives.Objectives {
		count := conceptCountByObjective[objective.ID]
		if objective.Active && count > 0 {
			objectivePriority[objective.ID] = float64(objective.Weight) / float64(count)
		}
	}

	var reviews []SessionItem
	for _, concept := range input.Bundle.Concepts.Concepts {
		if !concept.Active || !inScope(concept.ObjectiveID) {
			continue
		}
		projection, exists := input.Projections[concept.ID]
		if !exists || projection.Stage == StageUnseen || projection.LastStageEvidenceAt.IsZero() {
			continue
		}
		interval, exists := input.Policy.ReviewIntervals[projection.Stage]
		if !exists {
			continue
		}
		dueAt := projection.LastStageEvidenceAt.Add(interval)
		if dueAt.After(input.Now) {
			continue
		}
		reviews = append(reviews, SessionItem{
			ConceptID:   concept.ID,
			ObjectiveID: concept.ObjectiveID,
			Kind:        SessionReview,
			ReasonCode:  "review-due",
			ReasonFR:    fmt.Sprintf("Révision due depuis %s après une preuve de niveau %s.", dueAt.Format("2006-01-02 15:04"), projection.Stage),
			DueAt:       dueAt,
		})
	}

	slices.SortFunc(reviews, func(a, b SessionItem) int {
		if a.DueAt.Before(b.DueAt) {
			return -1
		}
		if a.DueAt.After(b.DueAt) {
			return 1
		}
		if objectivePriority[a.ObjectiveID] != objectivePriority[b.ObjectiveID] {
			if objectivePriority[a.ObjectiveID] > objectivePriority[b.ObjectiveID] {
				return -1
			}
			return 1
		}
		return compareText(a.ConceptID, b.ConceptID)
	})

	session := Session{GeneratedAt: input.Now}
	for _, item := range reviews {
		if len(session.Items) >= input.Policy.MaxReviews {
			break
		}
		session.Items = append(session.Items, item)
	}

	eligible := make(map[string]bool)
	for _, objectiveID := range EligibleObjectives(input.Bundle.Prerequisites, input.ObjectiveReadiness) {
		if inScope(objectiveID) {
			eligible[objectiveID] = true
		}
	}

	for _, concept := range input.Bundle.Concepts.Concepts {
		if !concept.Active || !eligible[concept.ObjectiveID] {
			continue
		}
		projection, exists := input.Projections[concept.ID]
		if !exists || !needsImmediatePractice(projection) {
			continue
		}
		if containsConcept(session.Items, concept.ID) {
			continue
		}
		reason := "Consolidation immédiate: le concept a été exposé, mais aucune pratique réussie ne l'a encore consolidé."
		if projection.Stage == StageRecall {
			reason = "Consolidation immédiate: le rappel est réussi, mais une pratique réussie est requise avant le concept suivant."
		}
		session.Items = append(session.Items, SessionItem{
			ConceptID:   concept.ID,
			ObjectiveID: concept.ObjectiveID,
			Kind:        SessionPractice,
			ReasonCode:  "practice-before-advance",
			ReasonFR:    reason,
		})
		return session, nil
	}

	type newCandidate struct {
		concept               curriculum.Concept
		unmetRecommendedCount int
		unmetRecommended      []string
		objectivePriority     float64
	}
	objectiveNodes := make(map[string]curriculum.ObjectiveNode, len(input.Bundle.Prerequisites.Nodes))
	for _, node := range input.Bundle.Prerequisites.Nodes {
		objectiveNodes[node.ObjectiveID] = node
	}

	startedIncompleteObjectives := make(map[string]bool)
	for _, concept := range input.Bundle.Concepts.Concepts {
		if !concept.Active || !eligible[concept.ObjectiveID] || input.ObjectiveReadiness[concept.ObjectiveID] {
			continue
		}
		if projection, exists := input.Projections[concept.ID]; exists && projection.Stage != StageUnseen {
			startedIncompleteObjectives[concept.ObjectiveID] = true
		}
	}

	var candidates []newCandidate
	for _, concept := range input.Bundle.Concepts.Concepts {
		if !concept.Active || !eligible[concept.ObjectiveID] {
			continue
		}
		if len(startedIncompleteObjectives) > 0 && !startedIncompleteObjectives[concept.ObjectiveID] {
			continue
		}
		if projection, exists := input.Projections[concept.ID]; exists && projection.Stage != StageUnseen {
			continue
		}
		if containsConcept(session.Items, concept.ID) {
			continue
		}

		node := objectiveNodes[concept.ObjectiveID]
		var unmet []string
		for _, recommended := range node.RecommendedPrerequisites {
			if !input.ObjectiveReadiness[recommended] {
				unmet = append(unmet, recommended)
			}
		}
		candidates = append(candidates, newCandidate{
			concept:               concept,
			unmetRecommendedCount: len(unmet),
			unmetRecommended:      unmet,
			objectivePriority:     objectivePriority[concept.ObjectiveID],
		})
	}

	slices.SortFunc(candidates, func(a, b newCandidate) int {
		if a.unmetRecommendedCount != b.unmetRecommendedCount {
			return a.unmetRecommendedCount - b.unmetRecommendedCount
		}
		if a.objectivePriority != b.objectivePriority {
			if a.objectivePriority > b.objectivePriority {
				return -1
			}
			return 1
		}
		if a.concept.PedagogyOrder != b.concept.PedagogyOrder {
			return a.concept.PedagogyOrder - b.concept.PedagogyOrder
		}
		if compared := compareText(a.concept.ObjectiveID, b.concept.ObjectiveID); compared != 0 {
			return compared
		}
		return compareText(a.concept.ID, b.concept.ID)
	})

	for index, candidate := range candidates {
		if index >= input.Policy.MaxNewConcepts {
			break
		}
		reason := fmt.Sprintf("Nouveau concept: prérequis prêts; priorité d’étude %.3f (poids LPI de l’objectif réparti sur ses concepts internes).", candidate.objectivePriority)
		reasonCode := "prerequisites-ready"
		if len(candidate.unmetRecommended) != 0 {
			reason = fmt.Sprintf(
				"Nouveau concept: hard prerequisites satisfaits; prérequis recommandé(s) non prêt(s): %s. Priorité d’étude %.3f; les prérequis recommandés restent prioritaires sur le poids.",
				strings.Join(candidate.unmetRecommended, ", "),
				candidate.objectivePriority,
			)
			reasonCode = "hard-ready-recommended-pending"
		}
		session.Items = append(session.Items, SessionItem{
			ConceptID:   candidate.concept.ID,
			ObjectiveID: candidate.concept.ObjectiveID,
			Kind:        SessionNew,
			ReasonCode:  reasonCode,
			ReasonFR:    reason,
		})
	}

	return session, nil
}

func needsImmediatePractice(projection MasteryProjection) bool {
	if projection.Stage != StageExposed && projection.Stage != StageRecall {
		return false
	}
	return projection.SuccessfulGuided == 0 &&
		projection.SuccessfulIndependent == 0 &&
		projection.SuccessfulTransfer == 0
}

func containsConcept(items []SessionItem, conceptID string) bool {
	for _, item := range items {
		if item.ConceptID == conceptID {
			return true
		}
	}
	return false
}

func compareText(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}
