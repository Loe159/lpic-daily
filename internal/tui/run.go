package tui

import (
	"context"
	"io"

	tea "charm.land/bubbletea/v2"

	"github.com/Loe159/lpic-daily/internal/activity"
	"github.com/Loe159/lpic-daily/internal/gamification"
	"github.com/Loe159/lpic-daily/internal/study"
)

func Run(
	ctx context.Context,
	plan study.Plan,
	game gamification.Snapshot,
	input io.Reader,
	output io.Writer,
	reports ...*activity.Report,
) (Action, error) {
	program := tea.NewProgram(
		NewDashboard(plan, game, reports...),
		tea.WithContext(ctx),
		tea.WithInput(input),
		tea.WithOutput(output),
	)
	finalModel, err := program.Run()
	if err != nil {
		return Action{}, err
	}
	dashboard, ok := finalModel.(Dashboard)
	if !ok {
		return Action{}, nil
	}
	return dashboard.ChosenAction(), nil
}
