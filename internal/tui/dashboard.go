package tui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"

	"github.com/Loe159/lpic-daily/internal/gamification"
	"github.com/Loe159/lpic-daily/internal/learning"
	"github.com/Loe159/lpic-daily/internal/study"
)

type ActionKind string

const (
	ActionNone     ActionKind = ""
	ActionLesson   ActionKind = "lesson"
	ActionQuestion ActionKind = "question"
	ActionLab      ActionKind = "lab"
)

type Action struct {
	Kind ActionKind
	ID   string
}

type Dashboard struct {
	plan   study.Plan
	game   gamification.Snapshot
	cursor int
	width  int
	height int
	action Action
}

func NewDashboard(plan study.Plan, game gamification.Snapshot) Dashboard {
	return Dashboard{
		plan:  plan,
		game:  game,
		width: 80,
	}
}

func (model Dashboard) Init() tea.Cmd {
	return nil
}

func (model Dashboard) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		model.width = message.Width
		model.height = message.Height
		return model, nil
	case tea.KeyPressMsg:
		next, quit := model.updateKey(message.String())
		if quit {
			return next, tea.Quit
		}
		return next, nil
	default:
		return model, nil
	}
}

func (model Dashboard) View() tea.View {
	return tea.NewView(model.render())
}

func (model Dashboard) ChosenAction() Action {
	return model.action
}

func (model Dashboard) updateKey(key string) (Dashboard, bool) {
	switch key {
	case "q", "ctrl+c", "esc":
		return model, true
	case "up", "k":
		if model.cursor > 0 {
			model.cursor--
		}
	case "down", "j":
		if model.cursor+1 < len(model.plan.Items) {
			model.cursor++
		}
	case "enter":
		model.action = defaultAction(model.currentItem())
		return model, model.action.Kind != ActionNone
	case "l":
		item := model.currentItem()
		if item != nil && item.RecommendedLessonID != "" {
			model.action = Action{Kind: ActionLesson, ID: item.RecommendedLessonID}
			return model, true
		}
	case "a":
		item := model.currentItem()
		if item != nil && item.RecommendedQuestionID != "" {
			model.action = Action{Kind: ActionQuestion, ID: item.RecommendedQuestionID}
			return model, true
		}
	case "p":
		item := model.currentItem()
		if item != nil && item.RecommendedLabID != "" {
			model.action = Action{Kind: ActionLab, ID: item.RecommendedLabID}
			return model, true
		}
	}
	return model, false
}

func (model Dashboard) currentItem() *study.Item {
	if model.cursor < 0 || model.cursor >= len(model.plan.Items) {
		return nil
	}
	return &model.plan.Items[model.cursor]
}

func defaultAction(item *study.Item) Action {
	if item == nil {
		return Action{}
	}
	if item.Kind == learning.SessionNew && item.RecommendedLessonID != "" {
		return Action{Kind: ActionLesson, ID: item.RecommendedLessonID}
	}
	if item.PreferLab && item.RecommendedLabID != "" {
		return Action{Kind: ActionLab, ID: item.RecommendedLabID}
	}
	if item.MasteryStage >= learning.StageGuided && item.RecommendedLabID != "" {
		return Action{Kind: ActionLab, ID: item.RecommendedLabID}
	}
	if item.RecommendedQuestionID != "" {
		return Action{Kind: ActionQuestion, ID: item.RecommendedQuestionID}
	}
	if item.RecommendedLabID != "" {
		return Action{Kind: ActionLab, ID: item.RecommendedLabID}
	}
	if item.RecommendedLessonID != "" {
		return Action{Kind: ActionLesson, ID: item.RecommendedLessonID}
	}
	return Action{}
}

func (model Dashboard) render() string {
	width := model.width
	if width < 40 {
		width = 40
	}

	var output strings.Builder
	output.WriteString("LPIC Daily · Aujourd'hui\n")
	output.WriteString(clip(
		fmt.Sprintf(
			"XP %d · série %d jour(s) · achievements %d",
			model.game.XP,
			model.game.CurrentStreakDays,
			len(model.game.UnlockedAchievements),
		),
		width,
	))
	output.WriteString("\n")
	output.WriteString(strings.Repeat("─", min(width, 72)))
	output.WriteString("\n")

	if len(model.plan.Items) == 0 {
		output.WriteString("\n✓ Séance du jour terminée\n")
		output.WriteString("Aucune révision n'est due pour le moment.\n")
		output.WriteString("LPIC Daily te proposera automatiquement les prochaines révisions et activités pratiques lorsqu'elles seront dues.\n")
		output.WriteString("\nq quitter\n")
		return output.String()
	}

	for index, item := range model.plan.Items {
		prefix := "  "
		if index == model.cursor {
			prefix = "> "
		}
		kind := "Nouveau"
		switch item.Kind {
		case learning.SessionReview:
			kind = "Révision"
		case learning.SessionPractice:
			kind = "Consolidation"
		}

		title := singleLine(item.ConceptTitleFR)
		line := fmt.Sprintf("%s%d. %s · %s · %s", prefix, index+1, kind, singleLine(item.ObjectiveID), title)
		output.WriteString(clip(line, width))
		output.WriteString("\n")

		if index == model.cursor {
			output.WriteString(clip(fmt.Sprintf("   Maîtrise: %s", singleLine(item.MasteryStage.String())), width))
			output.WriteString("\n")
			output.WriteString(clip("   "+singleLine(item.ReasonFR), width))
			output.WriteString("\n")
			if item.RecommendedLessonID != "" {
				output.WriteString(clip("   [l] Cours: "+singleLine(item.RecommendedLessonID), width))
				output.WriteString("\n")
			}
			if item.RecommendedQuestionID != "" {
				output.WriteString(clip("   [a] Question: "+singleLine(item.RecommendedQuestionID), width))
				output.WriteString("\n")
			}
			if item.RecommendedLabID != "" {
				output.WriteString(clip("   [p] Lab: "+singleLine(item.RecommendedLabID), width))
				output.WriteString("\n")
			}
		}
	}

	output.WriteString("\n↑/↓ ou j/k naviguer · Entrée lancer · l cours · a question · p pratique · q quitter\n")
	return output.String()
}

func SanitizeText(value string) string {
	var output strings.Builder
	output.Grow(len(value))
	for _, runeValue := range value {
		switch runeValue {
		case '\n', '\t':
			output.WriteRune(runeValue)
		default:
			if runeValue < 0x20 || (runeValue >= 0x7f && runeValue <= 0x9f) {
				continue
			}
			output.WriteRune(runeValue)
		}
	}
	return output.String()
}

func singleLine(value string) string {
	value = SanitizeText(value)
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\t", " ")
	return strings.Join(strings.Fields(value), " ")
}

func clip(value string, width int) string {
	value = SanitizeText(value)
	if width <= 1 || utf8.RuneCountInString(value) <= width {
		return value
	}
	runes := []rune(value)
	return string(runes[:width-1]) + "…"
}
