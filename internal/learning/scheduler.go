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

	var reviews []SessionItem
	for _, concept := range input.Bundle.Concepts.Concepts {
		if !concept.Active || !inScope(concept.ObjectiveID) {
			continue
		}
		projection, exists := input.Projections[concept.ID]
		if !exists || projection.Stage == StageUnseen || projection.LastEvidenceAt.IsZero() {
			continue
		}
		interval, exists := input.Policy.ReviewIntervals[projection.Stage]
		if !exists {
			continue
		}
		dueAt := projection.LastEvidenceAt.Add(interval)
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
		session.Items = append(session.Items, SessionItem{
			ConceptID:   concept.ID,
			ObjectiveID: concept.ObjectiveID,
			Kind:        SessionPractice,
			ReasonCode:  "practice-after-exposure",
			ReasonFR:    "Consolidation immédiate: le concept a été exposé mais aucune réponse correcte n'a encore confirmé sa compréhension.",
		})
		return session, nil
	}

	type newCandidate struct {
		concept               curriculum.Concept
		unmetRecommendedCount int
		unmetRecommended      []string
	}
	objectiveNodes := make(map[string]curriculum.ObjectiveNode, len(input.Bundle.Prerequisites.Nodes))
	for _, node := range input.Bundle.Prerequisites.Nodes {
		objectiveNodes[node.ObjectiveID] = node
	}

	var candidates []newCandidate
	for _, concept := range input.Bundle.Concepts.Concepts {
		if !concept.Active || !eligible[concept.ObjectiveID] {
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
		})
	}

	slices.SortFunc(candidates, func(a, b newCandidate) int {
		if a.unmetRecommendedCount != b.unmetRecommendedCount {
			return a.unmetRecommendedCount - b.unmetRecommendedCount
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
		reason := "Nouveau concept: ses hard prerequisites sont satisfaits et ses prerequisites recommandés sont prêts."
		reasonCode := "prerequisites-ready"
		if len(candidate.unmetRecommended) != 0 {
			reason = fmt.Sprintf(
				"Nouveau concept: ses hard prerequisites sont satisfaits. Prerequisite(s) recommandé(s) non prêt(s): %s; cela réduit seulement sa priorité et ne le bloque pas.",
				strings.Join(candidate.unmetRecommended, ", "),
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
	return projection.Stage == StageExposed &&
		projection.SuccessfulRecognition == 0 &&
		projection.SuccessfulRecall == 0 &&
		projection.SuccessfulGuided == 0 &&
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
