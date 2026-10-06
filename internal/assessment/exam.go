package assessment

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Loe159/lpic-daily/internal/content"
	"github.com/Loe159/lpic-daily/internal/curriculum"
)

// ExamQuestions builds a deterministic cumulative simulation whose number of
// questions per objective equals the official objective weight. Exam 101 has a
// total weight of 60, so this yields 60 mixed questions.
func ExamQuestions(
	curriculumBundle *curriculum.Bundle,
	contentBundle *content.Bundle,
	exam string,
) ([]content.Question, error) {
	if curriculumBundle == nil || contentBundle == nil {
		return nil, fmt.Errorf("curriculum and content bundles are required")
	}
	if exam != "101" {
		return nil, fmt.Errorf("unsupported exam %q", exam)
	}

	questionsByObjective := make(map[string][]content.Question)
	for _, question := range contentBundle.Questions {
		if question.Usage == "initial-assessment" || len(question.ObjectiveIDs) != 1 ||
			len(question.ConceptIDs) != 1 {
			continue
		}
		questionsByObjective[question.ObjectiveIDs[0]] = append(
			questionsByObjective[question.ObjectiveIDs[0]],
			question,
		)
	}

	var selected []content.Question
	for _, objective := range curriculumBundle.Objectives.Objectives {
		if !objective.Active || objective.Exam != exam {
			continue
		}
		candidates := questionsByObjective[objective.ID]
		slices.SortFunc(candidates, func(a, b content.Question) int {
			aPriority := examQuestionPriority(a)
			bPriority := examQuestionPriority(b)
			if aPriority != bPriority {
				return aPriority - bPriority
			}
			return strings.Compare(a.ID, b.ID)
		})
		if len(candidates) < objective.Weight {
			return nil, fmt.Errorf(
				"objective %s has %d eligible exam questions, need %d",
				objective.ID,
				len(candidates),
				objective.Weight,
			)
		}

		// Spread selections over the objective's concepts instead of taking a
		// cluster of questions from the first concept.
		seenConcept := make(map[string]bool)
		picked := 0
		for _, question := range candidates {
			conceptID := question.ConceptIDs[0]
			if seenConcept[conceptID] {
				continue
			}
			selected = append(selected, question)
			seenConcept[conceptID] = true
			picked++
			if picked == objective.Weight {
				break
			}
		}
		for _, question := range candidates {
			if picked == objective.Weight {
				break
			}
			if slices.ContainsFunc(selected, func(existing content.Question) bool {
				return existing.ID == question.ID
			}) {
				continue
			}
			selected = append(selected, question)
			picked++
		}
	}

	// Interleave objectives so the simulation is cumulative rather than a
	// sequence of topic blocks.
	slices.SortFunc(selected, func(a, b content.Question) int {
		aConcept := a.ConceptIDs[0]
		bConcept := b.ConceptIDs[0]
		aRank := stableExamRank(aConcept)
		bRank := stableExamRank(bConcept)
		if aRank != bRank {
			return aRank - bRank
		}
		return strings.Compare(a.ID, b.ID)
	})
	return selected, nil
}

func examQuestionPriority(question content.Question) int {
	switch {
	case question.EvidenceKindOnSuccess == "recall":
		return 0
	case strings.Contains(question.ID, ".q.autonomous-application"):
		return 1
	default:
		return 2
	}
}

func stableExamRank(value string) int {
	rank := 0
	for _, r := range value {
		rank = (rank*33 + int(r)) % 1009
	}
	return rank
}
