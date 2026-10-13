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
	color  bool
	action Action
}

func NewDashboard(plan study.Plan, game gamification.Snapshot) Dashboard {
	return Dashboard{plan: plan, game: game, width: 80}
}

func (model Dashboard) Init() tea.Cmd { return nil }

func (model Dashboard) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		model.width, model.height = message.Width, message.Height
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

func (model Dashboard) View() tea.View { return tea.NewView(model.renderWithColor(model.color)) }

func (model Dashboard) ChosenAction() Action { return model.action }

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

func kindLabel(kind learning.SessionItemKind) string {
	switch kind {
	case learning.SessionReview:
		return "RÉVISION"
	case learning.SessionPractice:
		return "CONSOLIDATION"
	default:
		return "NOUVEAU"
	}
}

func masteryLabel(stage learning.MasteryStage) string {
	switch stage {
	case learning.StageUnseen:
		return "À découvrir"
	case learning.StageExposed:
		return "Découverte"
	case learning.StageRecall:
		return "À mémoriser"
	case learning.StageGuided:
		return "Avec aide"
	case learning.StageIndependent:
		return "Autonome"
	case learning.StageTransfer:
		return "Maîtrisé"
	default:
		return "Non évalué"
	}
}

func actionLabel(action Action) string {
	switch action.Kind {
	case ActionLesson:
		return "Cours"
	case ActionQuestion:
		return "Question"
	case ActionLab:
		return "Lab"
	default:
		return "Aucune activité"
	}
}

func (model Dashboard) render() string { return model.renderWithColor(false) }

func (model Dashboard) renderWithColor(colored bool) string {
	width := model.width
	if width < 36 {
		width = 36
	}

	var output strings.Builder
	output.WriteString(Paint(colored, ToneAccent, "LPIC DAILY  /  AUJOURD'HUI"))
	output.WriteByte('\n')
	stats := fmt.Sprintf("%d XP   ·   série %d j   ·   %d badge(s)", model.game.XP, model.game.CurrentStreakDays, len(model.game.UnlockedAchievements))
	output.WriteString(Paint(colored, ToneMuted, clip(stats, width)))
	output.WriteString("\n")
	output.WriteString(Paint(colored, ToneMuted, strings.Repeat("─", min(width, 64))))
	output.WriteByte('\n')

	if len(model.plan.Items) == 0 {
		output.WriteString("\n")
		output.WriteString(Paint(colored, ToneSuccess, "✓ Séance du jour terminée"))
		output.WriteString("\n")
		output.WriteString("Aucune révision n'est due pour le moment.\n")
		output.WriteString("Les prochaines révisions et activités pratiques seront proposées à leur échéance.\n")
		output.WriteString("\nq quitter\n")
		return output.String()
	}

	output.WriteString(Paint(colored, ToneTitle, fmt.Sprintf("\nÀ FAIRE  ·  %d activité(s)\n\n", len(model.plan.Items))))
	compact := model.height > 0 && model.height < 18
	maxRows := len(model.plan.Items)
	if model.height > 0 {
		reserved := 14
		if compact {
			reserved = 9
		}
		maxRows = min(maxRows, max(2, model.height-reserved))
	}
	start := 0
	if model.cursor >= maxRows {
		start = model.cursor - maxRows + 1
	}
	if start > 0 {
		output.WriteString(Paint(colored, ToneMuted, "  ↑ autres activités"))
		output.WriteByte('\n')
	}
	for index := start; index < min(len(model.plan.Items), start+maxRows); index++ {
		item := model.plan.Items[index]
		prefix := "  "
		if index == model.cursor {
			prefix = "▸ "
		}
		line := fmt.Sprintf("%s%2d  %-13s %s  %s", prefix, index+1, kindLabel(item.Kind), singleLine(item.ObjectiveID), singleLine(item.ConceptTitleFR))
		line = clip(line, width)
		if index == model.cursor {
			line = Paint(colored, ToneAccent, line)
		}
		output.WriteString(line)
		output.WriteByte('\n')
	}
	if start+maxRows < len(model.plan.Items) {
		output.WriteString(Paint(colored, ToneMuted, "  ↓ autres activités"))
		output.WriteByte('\n')
	}
	item := model.currentItem()
	if compact {
		if item != nil {
			output.WriteString("\n" + Paint(colored, ToneSuccess, "Entrée  →  "+actionLabel(defaultAction(item))) + "\n")
		}
		output.WriteString(Paint(colored, ToneMuted, clip("↑↓ naviguer   ·   Entrée lancer   ·   q quitter", width)))
		output.WriteByte('\n')
		return output.String()
	}
	if item != nil {
		output.WriteString("\n")
		output.WriteString(Paint(colored, ToneMuted, strings.Repeat("─", min(width, 64))))
		output.WriteString("\n")
		title := clip("SÉLECTION  ·  "+singleLine(item.ConceptTitleFR), width)
		output.WriteString(Paint(colored, ToneTitle, title))
		output.WriteByte('\n')
		output.WriteString(clip("Maîtrise: "+masteryLabel(item.MasteryStage), width))
		output.WriteByte('\n')
		if reason := singleLine(item.ReasonFR); reason != "" && (model.height == 0 || model.height > 15) {
			output.WriteString(Paint(colored, ToneMuted, clip("Pourquoi : "+reason, width)))
			output.WriteByte('\n')
		}
		recommended := actionLabel(defaultAction(item))
		output.WriteString(Paint(colored, ToneSuccess, "Entrée  →  "+recommended))
		output.WriteByte('\n')
		var alternatives []string
		if item.RecommendedLessonID != "" {
			alternatives = append(alternatives, "[l] cours")
		}
		if item.RecommendedQuestionID != "" {
			alternatives = append(alternatives, "[a] question")
		}
		if item.RecommendedLabID != "" {
			alternatives = append(alternatives, "[p] lab")
		}
		if len(alternatives) > 0 {
			output.WriteString(Paint(colored, ToneMuted, clip(strings.Join(alternatives, "  ·  "), width)))
			output.WriteByte('\n')
		}
	}
	output.WriteString("\n")
	output.WriteString(Paint(colored, ToneMuted, clip("↑↓ naviguer   ·   Entrée lancer   ·   q quitter", width)))
	output.WriteByte('\n')
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
