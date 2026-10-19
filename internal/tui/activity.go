package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/Loe159/lpic-daily/internal/activity"
	"github.com/Loe159/lpic-daily/internal/learning"
)

// Activity navigation stays inside Bubble Tea and never starts a lab until an
// explicitly selected, already scheduled activity is confirmed.
func (model Dashboard) updateActivityKey(key string) (Dashboard, bool) {
	switch key {
	case "tab", "shift+tab":
		model.showActivity = false
		return model, false
	case "c":
		model.activityView = "calendar"
		model.activityDayOpen = false
		return model, false
	case "p":
		model.activityView = "progress"
		model.activityDayOpen = false
		return model, false
	case "esc", "backspace":
		if model.activityView == "calendar" && model.activityDayOpen {
			model.activityDayOpen = false
		} else if model.activityView == "progress" && model.activityNode != "all" {
			if node := model.activity.Nodes[model.activityNode]; node != nil && node.Parent != "" {
				model.activityNode = node.Parent
				model.activityCursor = 0
			}
		} else {
			model.showActivity = false
		}
		return model, false
	}
	if model.activityView == "progress" {
		children := model.activity.Children[model.activityNode]
		switch key {
		case "up", "k":
			if model.activityCursor > 0 {
				model.activityCursor--
			}
		case "down", "j":
			if model.activityCursor+1 < len(children) {
				model.activityCursor++
			}
		case "enter":
			if len(children) > 0 && model.activityCursor < len(children) {
				model.activityNode = children[model.activityCursor]
				model.activityCursor = 0
			} else if strings.HasPrefix(model.activityNode, "concept:") {
				id := strings.TrimPrefix(model.activityNode, "concept:")
				for i := range model.plan.Items {
					if model.plan.Items[i].ConceptID != id {
						continue
					}
					action := defaultAction(&model.plan.Items[i])
					if action.Kind != ActionNone {
						model.action = action
						return model, true
					}
				}
			}
		}
		return model, false
	}
	if model.activityDayOpen {
		switch key {
		case "left", "h":
			model.activityDate = model.activityDate.AddDate(0, 0, -1)
		case "right", "l":
			if model.activityDate.Before(model.activity.Now) {
				model.activityDate = model.activityDate.AddDate(0, 0, 1)
			}
		case "enter":
			model.activityDayOpen = false
		}
		return model, false
	}
	switch key {
	case "left", "h":
		model.activityDate = model.activityDate.AddDate(0, 0, -1)
	case "right", "l":
		if model.activityDate.Format("2006-01-02") < model.activity.Now.Format("2006-01-02") {
			model.activityDate = model.activityDate.AddDate(0, 0, 1)
		}
	case "up", "k":
		model.activityDate = model.activityDate.AddDate(0, 0, -7)
	case "down", "j":
		next := model.activityDate.AddDate(0, 0, 7)
		if next.After(model.activity.Now) {
			next = model.activity.Now
		}
		model.activityDate = next
	case "[":
		model.activityDate = model.activityDate.AddDate(0, 0, -91)
	case "]":
		next := model.activityDate.AddDate(0, 0, 91)
		if next.After(model.activity.Now) {
			next = model.activity.Now
		}
		model.activityDate = next
	case "enter":
		model.activityDayOpen = true
	}
	return model, false
}

func (model Dashboard) renderActivity() string {
	if model.activity == nil {
		return "Activité indisponible.\n"
	}
	if model.activityView == "progress" {
		return model.renderProgress()
	}
	if model.activityDayOpen {
		return model.renderActivityDay()
	}
	return model.renderActivityCalendar()
}

func (model Dashboard) renderActivityCalendar() string {
	var b strings.Builder
	report := model.activity
	root := report.Nodes["all"]
	if root == nil {
		return "Aucune donnée d'activité.\n"
	}
	fmt.Fprintln(&b, activityLabel("LPIC DAILY  /  ACTIVITÉ", "1;36"))
	fmt.Fprintln(&b, "Ma séance [Tab]   "+activityLabel("Activité", "1;37")+"   [p] Progression")
	fmt.Fprintln(&b, strings.Repeat("─", activityWidth(model.width)))
	fmt.Fprintf(&b, "Couverture %s   Maîtrise %s   Série %d j\n",
		activityLabel(fmt.Sprintf("%d%%", root.CoveragePercent()), "1;36"),
		activityLabel(fmt.Sprintf("%d%%", root.MasteryPercent()), "1;32"), report.CurrentStreak)
	fmt.Fprintf(&b, "Concepts : %d / %d abordés · %d autonomes · %d transférés\n",
		root.Covered, root.Total, root.Mastered, root.Transferred)
	fmt.Fprintf(&b, "Activités : %d   ·   Jours actifs : %d\n\n", report.TotalActivities, report.ActiveDays)
	fmt.Fprintln(&b, activityLabel("CALENDRIER  ·  une case = un jour", "1;37"))
	selected := report.DayAt(model.activityDate)
	fmt.Fprintf(&b, "Jour sélectionné : %s  ·  %d activité(s)\n", selected.Date, selected.Count())
	selectedWeekday := (int(model.activityDate.Weekday()) + 6) % 7
	weeks := (activityWidth(model.width) - 6) / 3
	if weeks > 13 {
		weeks = 13
	}
	if weeks < 1 {
		weeks = 1
	}
	start := model.activityDate.AddDate(0, 0, -selectedWeekday-(weeks-1)*7)
	fmt.Fprintf(&b, "             %s    →    %s\n", start.Format("02/01"), start.AddDate(0, 0, weeks*7-1).Format("02/01"))
	for dayIdx, label := range []string{"Lun", "Mar", "Mer", "Jeu", "Ven", "Sam", "Dim"} {
		fmt.Fprintf(&b, "%-5s ", label)
		for week := 0; week < weeks; week++ {
			date := start.AddDate(0, 0, week*7+dayIdx)
			count := report.DayAt(date).Count()
			mark := "·"
			color := "90"
			switch {
			case count >= 7:
				mark, color = "█", "1;92"
			case count >= 4:
				mark, color = "▓", "32"
			case count >= 2:
				mark, color = "▒", "32"
			case count == 1:
				mark, color = "░", "32"
			}
			if date.Format("2006-01-02") == model.activityDate.Format("2006-01-02") {
				b.WriteString(activityLabel("["+mark+"]", "7"))
			} else {
				b.WriteString(" " + activityLabel(mark, color) + " ")
			}
		}
		b.WriteString("\n")
	}
	fmt.Fprintln(&b, "Légende : · 0   ░ 1   ▒ 2-3   ▓ 4-6   █ 7+")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, activityLabel("PROGRESSION LPIC-1", "1;37"))
	for _, key := range report.Children["all"] {
		node := report.Nodes[key]
		if node == nil {
			continue
		}
		fmt.Fprintf(&b, "%s  Couverture %d%%  Maîtrise %d%%\n", node.Title, node.CoveragePercent(), node.MasteryPercent())
		fmt.Fprintln(&b, "   C "+activityBar(node.CoveragePercent(), 18, "36"))
		fmt.Fprintln(&b, "   M "+activityBar(node.MasteryPercent(), 18, "32"))
	}
	fmt.Fprintln(&b, "\n←→ Jour  ↑↓ Semaine  [ ] 13 sem.  Entrée Détails  p Domaines  Tab Séance  q Quitter")
	return b.String()
}

func (model Dashboard) renderActivityDay() string {
	var b strings.Builder
	day := model.activity.DayAt(model.activityDate)
	fmt.Fprintln(&b, activityLabel("LPIC DAILY  /  JOUR", "1;36"))
	fmt.Fprintln(&b, strings.Repeat("─", activityWidth(model.width)))
	fmt.Fprintf(&b, "%s  ·  %d activité(s)\n\n", day.Date, day.Count())
	courses, quizzes, labs := 0, 0, 0
	for _, item := range day.Activities {
		switch item.Kind {
		case learning.ActivityLesson:
			courses++
		case learning.ActivityQuestion:
			quizzes++
		default:
			labs++
		}
	}
	fmt.Fprintf(&b, "Cours %d   Quiz %d   Pratique %d\n\n", courses, quizzes, labs)
	if len(day.Activities) == 0 {
		fmt.Fprintln(&b, "Aucune activité enregistrée ce jour.")
	}
	maxLines := model.height - 10
	if maxLines < 4 {
		maxLines = 8
	}
	for i, entry := range day.Activities {
		if i >= maxLines {
			fmt.Fprintf(&b, "… %d activité(s) supplémentaire(s)\n", len(day.Activities)-i)
			break
		}
		label := activityKindName(entry.Kind)
		result := "réussi"
		if entry.Result == learning.ResultFail {
			result = "à revoir"
		}
		if entry.Result == learning.ResultPartial {
			result = "partiel"
		}
		text := fmt.Sprintf("%s  %-8s %-10s %s",
			entry.When.In(model.activity.Now.Location()).Format("15:04"), label, result, activitySource(entry))
		fmt.Fprintln(&b, clip(text, activityWidth(model.width)))
	}
	fmt.Fprintln(&b, "\n←→ Changer de jour  Esc Calendrier  p Progression  Tab Séance")
	return b.String()
}

func (model Dashboard) renderProgress() string {
	var b strings.Builder
	report := model.activity
	node := report.Nodes[model.activityNode]
	if node == nil {
		node = report.Nodes["all"]
	}
	if node == nil {
		return "Progression indisponible.\n"
	}
	fmt.Fprintln(&b, activityLabel("LPIC DAILY  /  PROGRESSION", "1;36"))
	fmt.Fprintln(&b, "Ma séance [Tab]   [c] Calendrier   "+activityLabel("Progression", "1;37"))
	fmt.Fprintln(&b, strings.Repeat("─", activityWidth(model.width)))
	fmt.Fprintln(&b, clip(node.Title, activityWidth(model.width)))
	fmt.Fprintf(&b, "Couverture %d/%d (%d%%)  ·  Maîtrise %d/%d (%d%%)\n",
		node.Covered, node.Total, node.CoveragePercent(), node.Mastered, node.Total, node.MasteryPercent())
	fmt.Fprintf(&b, "Cycle cours/quiz/lab %d/%d  ·  Transfert %d/%d\n",
		node.CycleComplete, node.Total, node.Transferred, node.Total)
	fmt.Fprintln(&b, "C "+activityBar(node.CoveragePercent(), 23, "36"))
	fmt.Fprintln(&b, "M "+activityBar(node.MasteryPercent(), 23, "32"))
	fmt.Fprintln(&b)
	children := report.Children[node.Key]
	if len(children) == 0 && strings.HasPrefix(node.Key, "concept:") {
		return b.String() + model.renderActivityConcept(strings.TrimPrefix(node.Key, "concept:"))
	}
	if len(children) == 0 {
		fmt.Fprintln(&b, "Aucun concept actif dans cette section.")
	}
	// Display a window around the focused item on a terminal with limited height.
	lines := model.height - 13
	if lines < 5 {
		lines = 8
	}
	from := 0
	if model.activityCursor >= lines {
		from = model.activityCursor - lines + 1
	}
	to := from + lines
	if to > len(children) {
		to = len(children)
	}
	for i := from; i < to; i++ {
		child := report.Nodes[children[i]]
		if child == nil {
			continue
		}
		prefix := "  "
		if model.activityCursor == i {
			prefix = activityLabel("▸ ", "1;36")
		}
		titleWidth := activityWidth(model.width) - 27
		if titleWidth < 12 {
			titleWidth = 12
		}
		fmt.Fprintf(&b, "%s%s\n", prefix, clip(child.ID+" · "+child.Title, titleWidth))
		fmt.Fprintf(&b, "    %3d%% vus  %3d%% autonomes  %d/%d\n",
			child.CoveragePercent(), child.MasteryPercent(), child.Mastered, child.Total)
	}
	fmt.Fprintln(&b, "\n↑↓ Sélectionner  Entrée Explorer  Esc Retour  c Calendrier  Tab Séance")
	return b.String()
}

func (model Dashboard) renderActivityConcept(id string) string {
	progress, ok := model.activity.Concepts[id]
	if !ok {
		return "\nConcept inconnu.\n"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Niveau : %s\n", activityStage(progress.Stage))
	fmt.Fprintf(&b, "Cours : %s    Quiz : %s    Lab : %s\n",
		activityDone(progress.LessonDone), activityDone(progress.QuizDone), activityDone(progress.PracticalDone))
	fmt.Fprintf(&b, "Cycle complet : %s\n", activityDone(progress.CycleComplete))
	if !progress.LastActivity.IsZero() {
		fmt.Fprintf(&b, "Dernière activité : %s\n", progress.LastActivity.In(model.activity.Now.Location()).Format("02/01/2006 15:04"))
	}
	for i := range model.plan.Items {
		if model.plan.Items[i].ConceptID == id && defaultAction(&model.plan.Items[i]).Kind != ActionNone {
			fmt.Fprintln(&b, "\nEntrée : lancer la prochaine activité proposée.")
			fmt.Fprintln(&b, "Esc : revenir aux concepts.  Tab : séance.")
			return b.String()
		}
	}
	fmt.Fprintln(&b, "\nAucune activité actuellement proposée pour ce concept.")
	fmt.Fprintln(&b, "Esc : revenir aux concepts.  Tab : séance.")
	return b.String()
}

func activityStage(stage learning.MasteryStage) string {
	switch stage {
	case learning.StageExposed:
		return "Découvert"
	case learning.StageRecall:
		return "Rappel démontré"
	case learning.StageGuided:
		return "Réussi avec aide"
	case learning.StageIndependent:
		return "Autonome"
	case learning.StageTransfer:
		return "Transfert confirmé"
	default:
		return "Non abordé"
	}
}
func activityDone(yes bool) string {
	if yes {
		return "✓"
	}
	return "—"
}
func activityKindName(kind learning.ActivityKind) string {
	switch kind {
	case learning.ActivityLesson:
		return "Cours"
	case learning.ActivityQuestion:
		return "Quiz"
	case learning.ActivityLab:
		return "Lab"
	default:
		return "Pratique"
	}
}
func activitySource(item activity.Entry) string {
	source := item.SourceID
	source = strings.TrimPrefix(source, "lpic1.")
	source = strings.ReplaceAll(source, "-", " ")
	if len(source) > 70 {
		source = clip(source, 70)
	}
	return source
}
func activityWidth(width int) int {
	if width < 40 {
		return 40
	}
	if width > 90 {
		return 90
	}
	return width
}
func activityLabel(text, style string) string {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return text
	}
	return "\x1b[" + style + "m" + text + "\x1b[0m"
}
func activityBar(value, width int, color string) string {
	filled := value * width / 100
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	return activityLabel(strings.Repeat("█", filled), color) + strings.Repeat("░", width-filled)
}
