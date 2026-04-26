CREATE TABLE cards (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  status TEXT NOT NULL CHECK (status IN ('active', 'archived')),
  front TEXT NOT NULL CHECK (length(trim(front)) > 0),
  back TEXT NOT NULL CHECK (length(trim(back)) > 0),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  archived_at TEXT
);

CREATE INDEX idx_cards_status_id
  ON cards (status, id);

CREATE TABLE card_sources (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  card_id INTEGER NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
  source_system TEXT NOT NULL CHECK (length(trim(source_system)) > 0),
  source_key TEXT NOT NULL CHECK (length(trim(source_key)) > 0),
  source_anchor TEXT,
  label TEXT,
  created_at TEXT NOT NULL
);

CREATE INDEX idx_card_sources_card_id_id
  ON card_sources (card_id, id);

CREATE TABLE card_schedule (
  card_id INTEGER PRIMARY KEY REFERENCES cards(id) ON DELETE CASCADE,
  due_at TEXT NOT NULL,
  last_reviewed_at TEXT,
  reps INTEGER NOT NULL CHECK (reps >= 0),
  lapses INTEGER NOT NULL CHECK (lapses >= 0),
  stability REAL NOT NULL CHECK (stability >= 0),
  difficulty REAL NOT NULL CHECK (difficulty >= 0),
  scheduled_days INTEGER NOT NULL CHECK (scheduled_days >= 0),
  elapsed_days INTEGER NOT NULL CHECK (elapsed_days >= 0),
  fsrs_state INTEGER NOT NULL CHECK (fsrs_state >= 0)
);

CREATE INDEX idx_card_schedule_due_at_card_id
  ON card_schedule (due_at, card_id);

CREATE TABLE review_sessions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  started_at TEXT NOT NULL,
  ended_at TEXT,
  status TEXT NOT NULL CHECK (status IN ('active', 'completed')),
  card_limit INTEGER CHECK (card_limit IS NULL OR card_limit > 0),
  time_limit_seconds INTEGER CHECK (time_limit_seconds IS NULL OR time_limit_seconds > 0)
);

CREATE INDEX idx_review_sessions_status_id
  ON review_sessions (status, id);

CREATE TABLE review_attempts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id INTEGER NOT NULL REFERENCES review_sessions(id) ON DELETE RESTRICT,
  card_id INTEGER NOT NULL REFERENCES cards(id) ON DELETE RESTRICT,
  answered_at TEXT NOT NULL,
  answer_text TEXT,
  rating TEXT NOT NULL CHECK (rating IN ('again', 'hard', 'good', 'easy')),
  grader TEXT NOT NULL CHECK (grader IN ('self', 'evidence')),
  evidence_summary TEXT,
  schedule_before_json TEXT NOT NULL,
  schedule_after_json TEXT NOT NULL
);

CREATE INDEX idx_review_attempts_session_id_id
  ON review_attempts (session_id, id);

CREATE INDEX idx_review_attempts_card_id_answered_at
  ON review_attempts (card_id, answered_at);
