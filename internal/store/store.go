package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Loe159/lpic-daily/internal/learning"
	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

type GamificationEvent struct {
	EventID       string
	OccurredAt    time.Time
	Kind          string
	Amount        int
	AchievementID string
	SourceItemID  string
	Metadata      map[string]any
}

func Open(ctx context.Context, path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("database path is required")
	}
	if path != ":memory:" && !strings.HasPrefix(path, "file:") {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return nil, fmt.Errorf("create database directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// LPIC Daily is a local single-user application. Keeping a single SQLite
	// connection also guarantees connection-local PRAGMAs are consistently set.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	s := &Store{db: db}
	if err := s.configure(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := s.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func DefaultPath() (string, error) {
	if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" {
		return filepath.Join(dataHome, "lpic-daily", "progress.db"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".local", "share", "lpic-daily", "progress.db"), nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) configure(ctx context.Context) error {
	for _, statement := range []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA busy_timeout = 5000",
	} {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("configure sqlite (%s): %w", statement, err)
		}
	}
	return nil
}

func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
        version INTEGER PRIMARY KEY,
        applied_at TEXT NOT NULL
    )`); err != nil {
		return fmt.Errorf("ensure migration table: %w", err)
	}

	migrations, err := Migrations()
	if err != nil {
		return err
	}

	var current int
	if err := s.db.QueryRowContext(ctx, "SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&current); err != nil {
		return fmt.Errorf("read migration version: %w", err)
	}
	if current > len(migrations) {
		return fmt.Errorf("database schema version %d is newer than supported version %d", current, len(migrations))
	}

	for _, migration := range migrations {
		if migration.Version <= current {
			continue
		}
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %d: %w", migration.Version, err)
		}
		if _, err := tx.ExecContext(ctx, migration.SQL); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %d (%s): %w", migration.Version, migration.Name, err)
		}
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO schema_migrations(version, applied_at) VALUES (?, ?)",
			migration.Version, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %d: %w", migration.Version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", migration.Version, err)
		}
	}
	return nil
}

func (s *Store) AppendEvidence(ctx context.Context, event learning.Event) error {
	if err := validateEvidence(event); err != nil {
		return err
	}
	objectiveIDs, err := json.Marshal(event.ObjectiveIDs)
	if err != nil {
		return fmt.Errorf("encode objective IDs: %w", err)
	}
	metadata := event.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("encode evidence metadata: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO mastery_evidence(
        event_id, occurred_at, concept_id, objective_ids_json, source_item_id,
        activity_kind, evidence_kind, result, highest_hint_level,
        solution_revealed, distribution, attempt_index, metadata_json
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.EventID,
		event.OccurredAt.UTC().Format(time.RFC3339Nano),
		event.ConceptID,
		string(objectiveIDs),
		event.SourceItemID,
		string(event.ActivityKind),
		string(event.EvidenceKind),
		string(event.Result),
		event.HighestHintLevel,
		boolInt(event.SolutionRevealed),
		event.Distribution,
		event.AttemptIndex,
		string(metadataJSON),
	)
	if err != nil {
		return fmt.Errorf("append mastery evidence: %w", err)
	}
	return nil
}

func (s *Store) EvidenceForConcept(ctx context.Context, conceptID string) ([]learning.Event, error) {
	if strings.TrimSpace(conceptID) == "" {
		return nil, errors.New("concept ID is required")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT
        event_id, occurred_at, concept_id, objective_ids_json, source_item_id,
        activity_kind, evidence_kind, result, highest_hint_level,
        solution_revealed, distribution, attempt_index, metadata_json
        FROM mastery_evidence
        WHERE concept_id = ?
        ORDER BY occurred_at ASC, event_id ASC`, conceptID)
	if err != nil {
		return nil, fmt.Errorf("query mastery evidence: %w", err)
	}
	defer rows.Close()

	var events []learning.Event
	for rows.Next() {
		event, err := scanEvidence(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate mastery evidence: %w", err)
	}
	return events, nil
}

func (s *Store) AllEvidence(ctx context.Context) ([]learning.Event, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT
        event_id, occurred_at, concept_id, objective_ids_json, source_item_id,
        activity_kind, evidence_kind, result, highest_hint_level,
        solution_revealed, distribution, attempt_index, metadata_json
        FROM mastery_evidence
        ORDER BY occurred_at ASC, event_id ASC`)
	if err != nil {
		return nil, fmt.Errorf("query all mastery evidence: %w", err)
	}
	defer rows.Close()

	var events []learning.Event
	for rows.Next() {
		event, err := scanEvidence(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate mastery evidence: %w", err)
	}
	return events, nil
}

func (s *Store) AppendGamification(ctx context.Context, event GamificationEvent) error {
	if strings.TrimSpace(event.EventID) == "" {
		return errors.New("gamification event ID is required")
	}
	if event.OccurredAt.IsZero() {
		return errors.New("gamification event time is required")
	}
	switch event.Kind {
	case "xp", "streak", "achievement":
	default:
		return fmt.Errorf("invalid gamification kind %q", event.Kind)
	}
	metadata := event.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("encode gamification metadata: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO gamification_events(
        event_id, occurred_at, kind, amount, achievement_id, source_item_id, metadata_json
    ) VALUES (?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?)`,
		event.EventID,
		event.OccurredAt.UTC().Format(time.RFC3339Nano),
		event.Kind,
		event.Amount,
		event.AchievementID,
		event.SourceItemID,
		string(metadataJSON),
	)
	if err != nil {
		return fmt.Errorf("append gamification event: %w", err)
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanEvidence(scanner rowScanner) (learning.Event, error) {
	var (
		event            learning.Event
		occurredAt       string
		objectiveIDsJSON string
		activityKind     string
		evidenceKind     string
		result           string
		solutionRevealed int
		metadataJSON     string
	)
	if err := scanner.Scan(
		&event.EventID,
		&occurredAt,
		&event.ConceptID,
		&objectiveIDsJSON,
		&event.SourceItemID,
		&activityKind,
		&evidenceKind,
		&result,
		&event.HighestHintLevel,
		&solutionRevealed,
		&event.Distribution,
		&event.AttemptIndex,
		&metadataJSON,
	); err != nil {
		return learning.Event{}, fmt.Errorf("scan mastery evidence: %w", err)
	}
	parsedTime, err := time.Parse(time.RFC3339Nano, occurredAt)
	if err != nil {
		return learning.Event{}, fmt.Errorf("parse evidence time: %w", err)
	}
	event.OccurredAt = parsedTime
	event.ActivityKind = learning.ActivityKind(activityKind)
	event.EvidenceKind = learning.EvidenceKind(evidenceKind)
	event.Result = learning.Result(result)
	event.SolutionRevealed = solutionRevealed != 0
	if err := json.Unmarshal([]byte(objectiveIDsJSON), &event.ObjectiveIDs); err != nil {
		return learning.Event{}, fmt.Errorf("decode objective IDs: %w", err)
	}
	if metadataJSON != "" {
		if err := json.Unmarshal([]byte(metadataJSON), &event.Metadata); err != nil {
			return learning.Event{}, fmt.Errorf("decode evidence metadata: %w", err)
		}
	}
	return event, nil
}

func validateEvidence(event learning.Event) error {
	if strings.TrimSpace(event.EventID) == "" {
		return errors.New("evidence event ID is required")
	}
	if event.OccurredAt.IsZero() {
		return errors.New("evidence event time is required")
	}
	if strings.TrimSpace(event.ConceptID) == "" {
		return errors.New("evidence concept ID is required")
	}
	if len(event.ObjectiveIDs) == 0 {
		return errors.New("evidence objective IDs are required")
	}
	if strings.TrimSpace(event.SourceItemID) == "" {
		return errors.New("evidence source item ID is required")
	}
	switch event.ActivityKind {
	case learning.ActivityLesson, learning.ActivityQuestion, learning.ActivityRep, learning.ActivityLab, learning.ActivityChallenge:
	default:
		return fmt.Errorf("invalid activity kind %q", event.ActivityKind)
	}
	switch event.EvidenceKind {
	case learning.EvidenceExposure, learning.EvidenceRecognition, learning.EvidenceRecall, learning.EvidenceGuidedPractice, learning.EvidenceIndependentPractice, learning.EvidenceTransfer:
	default:
		return fmt.Errorf("invalid evidence kind %q", event.EvidenceKind)
	}
	switch event.Result {
	case learning.ResultPass, learning.ResultPartial, learning.ResultFail:
	default:
		return fmt.Errorf("invalid evidence result %q", event.Result)
	}
	if event.HighestHintLevel < 0 || event.HighestHintLevel > 4 {
		return errors.New("highest hint level must be between 0 and 4")
	}
	switch event.Distribution {
	case "generic", "fedora", "debian", "opensuse":
	default:
		return fmt.Errorf("invalid distribution %q", event.Distribution)
	}
	if event.AttemptIndex < 1 {
		return errors.New("attempt index must be >= 1")
	}
	return nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
