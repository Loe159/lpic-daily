package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type LabDisclosure struct {
	HighestHintLevel int
	SolutionRevealed bool
}

func (store *Store) LabDisclosure(ctx context.Context, labID string) (LabDisclosure, error) {
	if labID == "" {
		return LabDisclosure{}, errors.New("lab ID is required")
	}

	var level int
	var solutionRevealed int
	err := store.db.QueryRowContext(
		ctx,
		`SELECT highest_hint_level, solution_revealed
		 FROM lab_disclosure
		 WHERE lab_id = ?`,
		labID,
	).Scan(&level, &solutionRevealed)
	if errors.Is(err, sql.ErrNoRows) {
		return LabDisclosure{}, nil
	}
	if err != nil {
		return LabDisclosure{}, fmt.Errorf("query lab disclosure for %s: %w", labID, err)
	}

	return LabDisclosure{
		HighestHintLevel: level,
		SolutionRevealed: solutionRevealed == 1,
	}, nil
}

func (store *Store) RecordLabDisclosure(
	ctx context.Context,
	labID string,
	highestHintLevel int,
	solutionRevealed bool,
	at time.Time,
) error {
	if labID == "" {
		return errors.New("lab ID is required")
	}
	if highestHintLevel < 1 || highestHintLevel > 4 {
		return fmt.Errorf("hint level %d outside 1..4", highestHintLevel)
	}
	if solutionRevealed && highestHintLevel < 4 {
		return errors.New("solution reveal requires hint level 4")
	}
	if highestHintLevel == 4 && !solutionRevealed {
		return errors.New("hint level 4 must mark solution reveal")
	}
	if at.IsZero() {
		return errors.New("disclosure timestamp is required")
	}

	solution := 0
	if solutionRevealed {
		solution = 1
	}
	_, err := store.db.ExecContext(
		ctx,
		`INSERT INTO lab_disclosure (
			lab_id, highest_hint_level, solution_revealed, disclosed_at
		) VALUES (?, ?, ?, ?)
		ON CONFLICT(lab_id) DO UPDATE SET
			highest_hint_level = CASE
				WHEN excluded.highest_hint_level > lab_disclosure.highest_hint_level
				THEN excluded.highest_hint_level
				ELSE lab_disclosure.highest_hint_level
			END,
			solution_revealed = CASE
				WHEN excluded.solution_revealed > lab_disclosure.solution_revealed
				THEN excluded.solution_revealed
				ELSE lab_disclosure.solution_revealed
			END,
			disclosed_at = excluded.disclosed_at`,
		labID,
		highestHintLevel,
		solution,
		at.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("record lab disclosure for %s: %w", labID, err)
	}
	return nil
}

func (store *Store) ClearLabDisclosure(ctx context.Context, labID string) error {
	if labID == "" {
		return errors.New("lab ID is required")
	}
	if _, err := store.db.ExecContext(
		ctx,
		"DELETE FROM lab_disclosure WHERE lab_id = ?",
		labID,
	); err != nil {
		return fmt.Errorf("clear lab disclosure for %s: %w", labID, err)
	}
	return nil
}
