CREATE TABLE IF NOT EXISTS lab_disclosure (
    lab_id TEXT PRIMARY KEY,
    highest_hint_level INTEGER NOT NULL CHECK (highest_hint_level BETWEEN 1 AND 4),
    solution_revealed INTEGER NOT NULL CHECK (solution_revealed IN (0, 1)),
    disclosed_at TEXT NOT NULL
);
