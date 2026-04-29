package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/yazanabuashour/openstudy/internal/study"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateCard(ctx context.Context, params study.CreateCardParams) (study.Card, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return study.Card{}, err
	}
	defer rollbackUnlessCommitted(tx)

	result, err := tx.ExecContext(ctx, `
INSERT INTO cards (status, front, back, created_at, updated_at)
VALUES (?, ?, ?, ?, ?)`,
		study.CardStatusActive,
		params.Front,
		params.Back,
		serializeInstant(params.Now),
		serializeInstant(params.Now),
	)
	if err != nil {
		return study.Card{}, wrapDatabaseError("create card", err)
	}
	cardID, err := result.LastInsertId()
	if err != nil {
		return study.Card{}, err
	}

	schedule := params.Schedule
	schedule.CardID = cardID
	if err := insertSchedule(ctx, tx, schedule); err != nil {
		return study.Card{}, err
	}
	if err := tx.Commit(); err != nil {
		return study.Card{}, err
	}

	card, err := r.GetCard(ctx, cardID)
	if err != nil {
		return study.Card{}, err
	}
	if card == nil {
		return study.Card{}, fmt.Errorf("created card %d not found", cardID)
	}
	return *card, nil
}

func (r *Repository) ListCards(ctx context.Context) ([]study.Card, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, status, front, back, created_at, updated_at, archived_at
FROM cards
ORDER BY id ASC`)
	if err != nil {
		return nil, wrapDatabaseError("list cards", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	cards := []study.Card{}
	for rows.Next() {
		card, err := scanCard(rows)
		if err != nil {
			return nil, err
		}
		cards = append(cards, card)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return cards, nil
}

func (r *Repository) GetCard(ctx context.Context, id int64) (*study.Card, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, status, front, back, created_at, updated_at, archived_at
FROM cards
WHERE id = ?`, id)
	card, err := scanCard(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &card, nil
}

func (r *Repository) ArchiveCard(ctx context.Context, params study.ArchiveCardParams) (study.Card, error) {
	result, err := r.db.ExecContext(ctx, `
UPDATE cards
SET status = ?, archived_at = ?, updated_at = ?
WHERE id = ?`,
		study.CardStatusArchived,
		serializeInstant(params.Now),
		serializeInstant(params.Now),
		params.ID,
	)
	if err != nil {
		return study.Card{}, wrapDatabaseError("archive card", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return study.Card{}, err
	}
	if changed == 0 {
		return study.Card{}, fmt.Errorf("card %d not found", params.ID)
	}
	card, err := r.GetCard(ctx, params.ID)
	if err != nil {
		return study.Card{}, err
	}
	if card == nil {
		return study.Card{}, fmt.Errorf("card %d not found", params.ID)
	}
	return *card, nil
}

func (r *Repository) AddSource(ctx context.Context, params study.AddSourceParams) (study.SourceReference, error) {
	result, err := r.db.ExecContext(ctx, `
INSERT INTO card_sources (card_id, source_system, source_key, source_anchor, label, created_at)
VALUES (?, ?, ?, ?, ?, ?)`,
		params.CardID,
		params.SourceSystem,
		params.SourceKey,
		nullableString(params.SourceAnchor),
		nullableString(params.Label),
		serializeInstant(params.Now),
	)
	if err != nil {
		return study.SourceReference{}, wrapDatabaseError("add source reference", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return study.SourceReference{}, err
	}
	return r.getSource(ctx, id)
}

func (r *Repository) ListSources(ctx context.Context, cardID int64) ([]study.SourceReference, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, card_id, source_system, source_key, source_anchor, label, created_at
FROM card_sources
WHERE card_id = ?
ORDER BY id ASC`, cardID)
	if err != nil {
		return nil, wrapDatabaseError("list source references", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	sources := []study.SourceReference{}
	for rows.Next() {
		source, err := scanSource(rows)
		if err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return sources, nil
}

func (r *Repository) CreateReviewSession(ctx context.Context, params study.CreateReviewSessionParams) (study.ReviewSession, error) {
	result, err := r.db.ExecContext(ctx, `
INSERT INTO review_sessions (started_at, status, card_limit, time_limit_seconds)
VALUES (?, ?, ?, ?)`,
		serializeInstant(params.StartedAt),
		study.SessionStatusActive,
		nullableInt(params.CardLimit),
		nullableInt(params.TimeLimitSeconds),
	)
	if err != nil {
		return study.ReviewSession{}, wrapDatabaseError("create review session", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return study.ReviewSession{}, err
	}
	return r.getReviewSession(ctx, id)
}

func (r *Repository) GetReviewSession(ctx context.Context, id int64) (*study.ReviewSession, error) {
	session, err := r.getReviewSession(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *Repository) FinishReviewSession(ctx context.Context, params study.FinishReviewSessionParams) (study.ReviewSession, error) {
	result, err := r.db.ExecContext(ctx, `
UPDATE review_sessions
SET status = ?, ended_at = ?
WHERE id = ?`,
		study.SessionStatusCompleted,
		serializeInstant(params.EndedAt),
		params.ID,
	)
	if err != nil {
		return study.ReviewSession{}, wrapDatabaseError("finish review session", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return study.ReviewSession{}, err
	}
	if changed == 0 {
		return study.ReviewSession{}, fmt.Errorf("review session %d not found", params.ID)
	}
	return r.getReviewSession(ctx, params.ID)
}

func (r *Repository) ReviewSessionSummary(ctx context.Context, id int64) (study.ReviewSessionSummary, error) {
	session, err := r.GetReviewSession(ctx, id)
	if err != nil {
		return study.ReviewSessionSummary{}, err
	}
	if session == nil {
		return study.ReviewSessionSummary{}, fmt.Errorf("review session %d not found", id)
	}

	rows, err := r.db.QueryContext(ctx, `
SELECT rating, COUNT(*)
FROM review_attempts
WHERE session_id = ?
GROUP BY rating`, id)
	if err != nil {
		return study.ReviewSessionSummary{}, wrapDatabaseError("summarize review session", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	summary := study.ReviewSessionSummary{
		Session:      *session,
		RatingCounts: map[study.Rating]int{},
	}
	for rows.Next() {
		var rating study.Rating
		var count int
		if err := rows.Scan(&rating, &count); err != nil {
			return study.ReviewSessionSummary{}, err
		}
		summary.RatingCounts[rating] = count
		summary.AttemptCount += count
	}
	if err := rows.Err(); err != nil {
		return study.ReviewSessionSummary{}, err
	}
	return summary, nil
}

func (r *Repository) RecordReviewAttempt(ctx context.Context, params study.RecordReviewAttemptParams) (study.ReviewAttempt, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return study.ReviewAttempt{}, err
	}
	defer rollbackUnlessCommitted(tx)

	beforeJSON, afterJSON, err := marshalScheduleSnapshots(params.ScheduleBefore, params.ScheduleAfter)
	if err != nil {
		return study.ReviewAttempt{}, err
	}
	if err := updateSchedule(ctx, tx, params.ScheduleAfter); err != nil {
		return study.ReviewAttempt{}, err
	}
	result, err := tx.ExecContext(ctx, `
INSERT INTO review_attempts (
  session_id, card_id, answered_at, answer_text, rating, grader, evidence_summary,
  schedule_before_json, schedule_after_json
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		params.SessionID,
		params.CardID,
		serializeInstant(params.AnsweredAt),
		nullableString(params.AnswerText),
		params.Rating,
		params.Grader,
		nullableString(params.EvidenceSummary),
		beforeJSON,
		afterJSON,
	)
	if err != nil {
		return study.ReviewAttempt{}, wrapDatabaseError("record review attempt", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return study.ReviewAttempt{}, err
	}
	if err := tx.Commit(); err != nil {
		return study.ReviewAttempt{}, err
	}
	return r.getReviewAttempt(ctx, id)
}

func (r *Repository) getSource(ctx context.Context, id int64) (study.SourceReference, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, card_id, source_system, source_key, source_anchor, label, created_at
FROM card_sources
WHERE id = ?`, id)
	return scanSource(row)
}

func (r *Repository) getReviewSession(ctx context.Context, id int64) (study.ReviewSession, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, started_at, ended_at, status, card_limit, time_limit_seconds
FROM review_sessions
WHERE id = ?`, id)
	return scanReviewSession(row)
}

func (r *Repository) getReviewAttempt(ctx context.Context, id int64) (study.ReviewAttempt, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, session_id, card_id, answered_at, answer_text, rating, grader, evidence_summary
FROM review_attempts
WHERE id = ?`, id)
	return scanReviewAttempt(row)
}

func marshalScheduleSnapshots(before, after study.CardSchedule) (string, string, error) {
	beforeJSON, err := json.Marshal(before)
	if err != nil {
		return "", "", err
	}
	afterJSON, err := json.Marshal(after)
	if err != nil {
		return "", "", err
	}
	return string(beforeJSON), string(afterJSON), nil
}
