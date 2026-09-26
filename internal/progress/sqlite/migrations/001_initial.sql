CREATE TABLE IF NOT EXISTS mastery_evidence (
    event_id TEXT PRIMARY KEY,
    occurred_at TEXT NOT NULL,
    concept_id TEXT NOT NULL,
    objective_ids_json TEXT NOT NULL,
    source_item_id TEXT NOT NULL,
    activity_kind TEXT NOT NULL,
    evidence_kind TEXT NOT NULL,
    result TEXT NOT NULL,
    highest_hint_level INTEGER NOT NULL CHECK (highest_hint_level BETWEEN 0 AND 4),
    solution_revealed INTEGER NOT NULL CHECK (solution_revealed IN (0, 1)),
    distribution TEXT NOT NULL,
    attempt_index INTEGER NOT NULL CHECK (attempt_index >= 1)
);

CREATE INDEX IF NOT EXISTS idx_mastery_evidence_concept_time
    ON mastery_evidence(concept_id, occurred_at, event_id);

CREATE TABLE IF NOT EXISTS gamification_events (
    event_id TEXT PRIMARY KEY,
    occurred_at TEXT NOT NULL,
    event_type TEXT NOT NULL,
    amount INTEGER NOT NULL,
    metadata_json TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_gamification_events_time
    ON gamification_events(occurred_at, event_id);
