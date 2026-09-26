package content

import (
	"errors"
	"regexp"
	"slices"
	"strings"
)

type Answer struct {
	Text      string
	ChoiceIDs []string
}

func (question Question) Grade(answer Answer) (bool, error) {
	switch question.Grading.Strategy {
	case "exact-text":
		got := strings.TrimSpace(answer.Text)
		for _, accepted := range question.Grading.AcceptedAnswers {
			want := strings.TrimSpace(accepted)
			if question.Grading.CaseSensitive {
				if got == want {
					return true, nil
				}
				continue
			}
			if strings.EqualFold(got, want) {
				return true, nil
			}
		}
		return false, nil
	case "regex":
		pattern, err := regexp.Compile(question.Grading.Pattern)
		if err != nil {
			return false, err
		}
		return pattern.MatchString(strings.TrimSpace(answer.Text)), nil
	case "choice-ids":
		if hasDuplicate(answer.ChoiceIDs) {
			return false, nil
		}
		got := slices.Clone(answer.ChoiceIDs)
		want := slices.Clone(question.Grading.AcceptedChoiceIDs)
		slices.Sort(got)
		slices.Sort(want)
		return slices.Equal(got, want), nil
	case "ordered-ids":
		if hasDuplicate(answer.ChoiceIDs) {
			return false, nil
		}
		return slices.Equal(answer.ChoiceIDs, question.Grading.ExpectedOrder), nil
	default:
		return false, errors.New("unsupported grading strategy")
	}
}

func hasDuplicate(values []string) bool {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			return true
		}
		seen[value] = struct{}{}
	}
	return false
}
