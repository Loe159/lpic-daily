package tui

import (
	"fmt"
	"time"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"

	"github.com/Loe159/lpic-daily/internal/activity"
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
	activity *activity.Report
	showActivity bool
	activityView string
	activityDate time.Time
	activityDayOpen bool
	activityNode string
	activityCursor int
}

func NewDashboard(plan study.Plan, game gamification.Snapshot, reports ...*activity.Report) Dashboard {
	model := Dashboard{plan: plan, game: game, width: 80, activityNode: "all", activityView: "calendar"}
	if len(reports) > 0 && reports[0] != nil {
		model.activity = reports[0]
		model.activityDate = reports[0].Now
	}
	return model
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
	if model.activity != nil {
		if key == "tab" || key == "shift+tab" {
			model.showActivity = !model.showActivity
			return model, false
		}
		if model.showActivity {
			if key == "q" || key == "ctrl+c" { return model, true }
			return model.updateActivityKey(key)
		}
	}
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
	if model.showActivity { return model.renderActivity() }
	width := model.width
	if width < 40 {
		width = 40
	}
	if width > 90 {
		width = 90
	}
	var out strings.Builder
	out.WriteString("LPIC DAILY  /  Ma séance\n")
	out.WriteString(fmt.Sprintf("%d XP  ·  %d jours de série  ·  %d succès\n",
		model.game.XP, model.game.CurrentStreakDays, len(model.game.UnlockedAchievements)))
	out.WriteString(strings.Repeat("─", width))
	out.WriteString("\n")

	if len(model.plan.Items) == 0 {
		out.WriteString("\n✓ Tout est à jour. Aucune activité prévue.\n\n")
		if model.activity != nil { out.WriteString("[Tab] Activité   ") }
		out.WriteString("[q] Quitter\n")
		return out.String()
	}
	out.WriteString(fmt.Sprintf("\nAUJOURD'HUI  ·  %d activités\n\n", len(model.plan.Items)))
	for index, item := range model.plan.Items {
		prefix := "  "
		if index == model.cursor {
			prefix = "▸ "
		}
		kind := "Cours"
		switch item.Kind {
		case learning.SessionReview:
			kind = "Révision"
		case learning.SessionPractice:
			kind = "Pratique"
		}
		label := fmt.Sprintf("%s%d  %-9s %s  %s", prefix, index+1, kind,
			singleLine(item.ObjectiveID), singleLine(item.ConceptTitleFR))
		out.WriteString(clip(label, width))
		out.WriteString("\n")
	}
	item := model.currentItem()
	if item != nil {
		out.WriteString("\n")
		out.WriteString(strings.Repeat("─", width))
		out.WriteString("\nSÉLECTION  ·  ")
		out.WriteString(clip(singleLine(item.ConceptTitleFR), width-14))
		out.WriteString("\n")
		out.WriteString("Maîtrise : " + singleLine(item.MasteryStage.String()) + "\n")
		if reason := singleLine(item.ReasonFR); reason != "" {
			out.WriteString(clip("Pourquoi : "+reason, width))
			out.WriteString("\n")
		}
		var choices []string
		if item.RecommendedLessonID != "" {
			choices = append(choices, "[l] Cours")
		}
		if item.RecommendedQuestionID != "" {
			choices = append(choices, "[a] Quiz")
		}
		if item.RecommendedLabID != "" {
			choices = append(choices, "[p] Lab")
		}
		if len(choices) > 0 {
			out.WriteString("Activités : " + strings.Join(choices, "   ") + "\n")
		}
	}
	if model.activity != nil {
		out.WriteString("\n↑↓ Naviguer   Entrée Commencer   Tab Activité   q Quitter\n")
	} else {
		out.WriteString("\n↑↓ Naviguer   Entrée Commencer   q Quitter\n")
	}
	return out.String()
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
