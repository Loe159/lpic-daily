PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    applied_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS mastery_evidence (
    event_id TEXT PRIMARY KEY,
    occurred_at TEXT NOT NULL,
    concept_id TEXT NOT NULL,
    objective_ids_json TEXT NOT NULL,
    source_item_id TEXT NOT NULL,
    activity_kind TEXT NOT NULL CHECK (activity_kind IN ('lesson','question','rep','lab','challenge')),
    evidence_kind TEXT NOT NULL CHECK (evidence_kind IN ('exposure','recognition','recall','guided-practice','independent-practice','transfer')),
    result TEXT NOT NULL CHECK (result IN ('pass','partial','fail')),
    highest_hint_level INTEGER NOT NULL DEFAULT 0 CHECK (highest_hint_level BETWEEN 0 AND 4),
    solution_revealed INTEGER NOT NULL DEFAULT 0 CHECK (solution_revealed IN (0,1)),
    distribution TEXT NOT NULL CHECK (distribution IN ('generic','fedora','debian','opensuse')),
    attempt_index INTEGER NOT NULL CHECK (attempt_index >= 1),
    metadata_json TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_mastery_evidence_concept_time
    ON mastery_evidence(concept_id, occurred_at);

CREATE TRIGGER IF NOT EXISTS mastery_evidence_no_update
BEFORE UPDATE ON mastery_evidence
BEGIN
    SELECT RAISE(ABORT, 'mastery evidence is append-only');
END;

CREATE TRIGGER IF NOT EXISTS mastery_evidence_no_delete
BEFORE DELETE ON mastery_evidence
BEGIN
    SELECT RAISE(ABORT, 'mastery evidence is append-only');
END;

CREATE TABLE IF NOT EXISTS gamification_events (
    event_id TEXT PRIMARY KEY,
    occurred_at TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('xp','streak','achievement')),
    amount INTEGER NOT NULL DEFAULT 0,
    achievement_id TEXT,
    source_item_id TEXT,
    metadata_json TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_gamification_events_time
    ON gamification_events(occurred_at);

CREATE TRIGGER IF NOT EXISTS gamification_events_no_update
BEFORE UPDATE ON gamification_events
BEGIN
    SELECT RAISE(ABORT, 'gamification events are append-only');
END;

CREATE TRIGGER IF NOT EXISTS gamification_events_no_delete
BEFORE DELETE ON gamification_events
BEGIN
    SELECT RAISE(ABORT, 'gamification events are append-only');
END;

CREATE TABLE IF NOT EXISTS daily_notification_state (
    local_date TEXT PRIMARY KEY,
    notified_at TEXT NOT NULL,
    opened_at TEXT
);
