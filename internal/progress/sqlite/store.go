package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Loe159/lpic-daily/internal/learning"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Store struct {
	db *sql.DB
}

func Open(ctx context.Context, path string) (*Store, error) {
	if path == "" {
		return nil, errors.New("database path is required")
	}

	if path != ":memory:" && !strings.HasPrefix(path, "file:") {
		path = filepath.Clean(path)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}
	db.SetMaxOpenConns(1)

	store := &Store{db: db}
	if err := store.migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (store *Store) Close() error {
	return store.db.Close()
}

func (store *Store) AppendEvidence(ctx context.Context, event learning.EvidenceEvent) error {
	if err := event.Validate(); err != nil {
		return fmt.Errorf("validate evidence: %w", err)
	}

	objectiveIDs, err := json.Marshal(event.ObjectiveIDs)
	if err != nil {
		return fmt.Errorf("encode objective IDs: %w", err)
	}

	solutionRevealed := 0
	if event.SolutionRevealed {
		solutionRevealed = 1
	}

	_, err = store.db.ExecContext(
		ctx,
		`INSERT INTO mastery_evidence (
			event_id,
			occurred_at,
			concept_id,
			objective_ids_json,
			source_item_id,
			activity_kind,
			evidence_kind,
			result,
			highest_hint_level,
			solution_revealed,
			distribution,
			attempt_index
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.EventID,
		event.OccurredAt.UTC().Format(time.RFC3339Nano),
		event.ConceptID,
		string(objectiveIDs),
		event.SourceItemID,
		string(event.ActivityKind),
		string(event.EvidenceKind),
		string(event.Result),
		event.HighestHintLevel,
		solutionRevealed,
		event.Distribution,
		event.AttemptIndex,
	)
	if err != nil {
		return fmt.Errorf("append evidence %s: %w", event.EventID, err)
	}
	return nil
}

func (store *Store) EvidenceForConcept(ctx context.Context, conceptID string) ([]learning.EvidenceEvent, error) {
	if conceptID == "" {
		return nil, errors.New("concept ID is required")
	}

	rows, err := store.db.QueryContext(
		ctx,
		`SELECT
			event_id,
			occurred_at,
			concept_id,
			objective_ids_json,
			source_item_id,
			activity_kind,
			evidence_kind,
			result,
			highest_hint_level,
			solution_revealed,
			distribution,
			attempt_index
		FROM mastery_evidence
		WHERE concept_id = ?
		ORDER BY occurred_at, event_id`,
		conceptID,
	)
	if err != nil {
		return nil, fmt.Errorf("query evidence for %s: %w", conceptID, err)
	}
	defer rows.Close()

	var events []learning.EvidenceEvent
	for rows.Next() {
		var (
			event            learning.EvidenceEvent
			occurredAt       string
			objectiveIDsJSON string
			activity         string
			evidence         string
			result           string
			solutionRevealed int
		)

		if err := rows.Scan(
			&event.EventID,
			&occurredAt,
			&event.ConceptID,
			&objectiveIDsJSON,
			&event.SourceItemID,
			&activity,
			&evidence,
			&result,
			&event.HighestHintLevel,
			&solutionRevealed,
			&event.Distribution,
			&event.AttemptIndex,
		); err != nil {
			return nil, fmt.Errorf("scan evidence row: %w", err)
		}

		event.OccurredAt, err = time.Parse(time.RFC3339Nano, occurredAt)
		if err != nil {
			return nil, fmt.Errorf("parse evidence time %q: %w", occurredAt, err)
		}
		if err := json.Unmarshal([]byte(objectiveIDsJSON), &event.ObjectiveIDs); err != nil {
			return nil, fmt.Errorf("decode objective IDs for %s: %w", event.EventID, err)
		}
		event.ActivityKind = learning.ActivityKind(activity)
		event.EvidenceKind = learning.EvidenceKind(evidence)
		event.Result = learning.Result(result)
		event.SolutionRevealed = solutionRevealed == 1

		if err := event.Validate(); err != nil {
			return nil, fmt.Errorf("stored evidence %s is invalid: %w", event.EventID, err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate evidence rows: %w", err)
	}

	return events, nil
}

func (store *Store) SchemaVersion(ctx context.Context) (int, error) {
	var version int
	if err := store.db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return 0, fmt.Errorf("read sqlite user_version: %w", err)
	}
	return version, nil
}

func (store *Store) migrate(ctx context.Context) error {
	entries, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("read embedded migrations: %w", err)
	}

	type migration struct {
		version int
		name    string
		sql     string
	}
	var pending []migration

	current, err := store.SchemaVersion(ctx)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		prefix, _, ok := strings.Cut(entry.Name(), "_")
		if !ok {
			return fmt.Errorf("migration %s has no numeric prefix", entry.Name())
		}
		version, err := strconv.Atoi(prefix)
		if err != nil || version < 1 {
			return fmt.Errorf("migration %s has invalid version prefix", entry.Name())
		}
		if version <= current {
			continue
		}
		body, err := fs.ReadFile(migrations, "migrations/"+entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		pending = append(pending, migration{
			version: version,
			name:    entry.Name(),
			sql:     string(body),
		})
	}

	sort.Slice(pending, func(i, j int) bool {
		return pending[i].version < pending[j].version
	})

	expected := current + 1
	for _, migration := range pending {
		if migration.version != expected {
			return fmt.Errorf("migration sequence gap: expected %d, found %d (%s)", expected, migration.version, migration.name)
		}
		if err := store.applyMigration(ctx, migration.version, migration.name, migration.sql); err != nil {
			return err
		}
		expected++
	}
	return nil
}

func (store *Store) applyMigration(ctx context.Context, version int, name, body string) error {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", name, err)
	}

	if _, err := tx.ExecContext(ctx, body); err != nil {
		tx.Rollback()
		return fmt.Errorf("execute migration %s: %w", name, err)
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", version)); err != nil {
		tx.Rollback()
		return fmt.Errorf("set schema version for %s: %w", name, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %s: %w", name, err)
	}
	return nil
}
