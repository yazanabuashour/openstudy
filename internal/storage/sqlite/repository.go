package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/yazanabuashour/openstudy/internal/study"
)

const fixedInstantLayout = "2006-01-02T15:04:05.000000000Z"

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

func (r *Repository) GetCardSchedule(ctx context.Context, cardID int64) (*study.CardSchedule, error) {
	row := r.db.QueryRowContext(ctx, scheduleSelectSQL()+` WHERE card_id = ?`, cardID)
	schedule, err := scanSchedule(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &schedule, nil
}

func (r *Repository) ListDueCards(ctx context.Context, filter study.DueCardFilter) ([]study.CardWithSchedule, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT c.id, c.status, c.front, c.back, c.created_at, c.updated_at, c.archived_at,
       s.card_id, s.due_at, s.last_reviewed_at, s.reps, s.lapses, s.stability, s.difficulty,
       s.scheduled_days, s.elapsed_days, s.fsrs_state
FROM cards c
JOIN card_schedule s ON s.card_id = c.id
WHERE c.status = ? AND s.due_at <= ?
ORDER BY s.due_at ASC, c.id ASC
LIMIT ?`,
		study.CardStatusActive,
		serializeInstant(filter.Now),
		filter.Limit,
	)
	if err != nil {
		return nil, wrapDatabaseError("list due cards", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	cards := []study.CardWithSchedule{}
	for rows.Next() {
		card, schedule, err := scanCardWithSchedule(rows)
		if err != nil {
			return nil, err
		}
		cards = append(cards, study.CardWithSchedule{Card: card, Schedule: schedule})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return cards, nil
}

func (r *Repository) RecordReviewAttempt(ctx context.Context, params study.RecordReviewAttemptParams) (study.ReviewAttempt, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return study.ReviewAttempt{}, err
	}
	defer rollbackUnlessCommitted(tx)

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
		params.ScheduleBeforeJSON,
		params.ScheduleAfterJSON,
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

func insertSchedule(ctx context.Context, tx *sql.Tx, schedule study.CardSchedule) error {
	_, err := tx.ExecContext(ctx, `
INSERT INTO card_schedule (
  card_id, due_at, last_reviewed_at, reps, lapses, stability, difficulty,
  scheduled_days, elapsed_days, fsrs_state
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		schedule.CardID,
		serializeInstant(schedule.DueAt),
		nullableInstant(schedule.LastReviewedAt),
		schedule.Reps,
		schedule.Lapses,
		schedule.Stability,
		schedule.Difficulty,
		schedule.ScheduledDays,
		schedule.ElapsedDays,
		schedule.FSRSState,
	)
	if err != nil {
		return wrapDatabaseError("insert card schedule", err)
	}
	return nil
}

func updateSchedule(ctx context.Context, tx *sql.Tx, schedule study.CardSchedule) error {
	result, err := tx.ExecContext(ctx, `
UPDATE card_schedule
SET due_at = ?, last_reviewed_at = ?, reps = ?, lapses = ?, stability = ?, difficulty = ?,
    scheduled_days = ?, elapsed_days = ?, fsrs_state = ?
WHERE card_id = ?`,
		serializeInstant(schedule.DueAt),
		nullableInstant(schedule.LastReviewedAt),
		schedule.Reps,
		schedule.Lapses,
		schedule.Stability,
		schedule.Difficulty,
		schedule.ScheduledDays,
		schedule.ElapsedDays,
		schedule.FSRSState,
		schedule.CardID,
	)
	if err != nil {
		return wrapDatabaseError("update card schedule", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed == 0 {
		return fmt.Errorf("card %d schedule not found", schedule.CardID)
	}
	return nil
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
SELECT id, session_id, card_id, answered_at, answer_text, rating, grader, evidence_summary,
       schedule_before_json, schedule_after_json
FROM review_attempts
WHERE id = ?`, id)
	return scanReviewAttempt(row)
}

func scanCard(scanner interface {
	Scan(dest ...any) error
}) (study.Card, error) {
	var card study.Card
	var createdAt, updatedAt string
	var archivedAt sql.NullString
	if err := scanner.Scan(
		&card.ID,
		&card.Status,
		&card.Front,
		&card.Back,
		&createdAt,
		&updatedAt,
		&archivedAt,
	); err != nil {
		return study.Card{}, err
	}
	var err error
	card.CreatedAt, err = parseInstant(createdAt)
	if err != nil {
		return study.Card{}, err
	}
	card.UpdatedAt, err = parseInstant(updatedAt)
	if err != nil {
		return study.Card{}, err
	}
	card.ArchivedAt, err = parseNullableInstant(archivedAt)
	if err != nil {
		return study.Card{}, err
	}
	return card, nil
}

func scanSource(scanner interface {
	Scan(dest ...any) error
}) (study.SourceReference, error) {
	var source study.SourceReference
	var anchor, label sql.NullString
	var createdAt string
	if err := scanner.Scan(
		&source.ID,
		&source.CardID,
		&source.SourceSystem,
		&source.SourceKey,
		&anchor,
		&label,
		&createdAt,
	); err != nil {
		return study.SourceReference{}, err
	}
	var err error
	source.SourceAnchor = nullableStringPointer(anchor)
	source.Label = nullableStringPointer(label)
	source.CreatedAt, err = parseInstant(createdAt)
	if err != nil {
		return study.SourceReference{}, err
	}
	return source, nil
}

func scanReviewSession(scanner interface {
	Scan(dest ...any) error
}) (study.ReviewSession, error) {
	var session study.ReviewSession
	var startedAt string
	var endedAt sql.NullString
	var cardLimit, timeLimitSeconds sql.NullInt64
	if err := scanner.Scan(
		&session.ID,
		&startedAt,
		&endedAt,
		&session.Status,
		&cardLimit,
		&timeLimitSeconds,
	); err != nil {
		return study.ReviewSession{}, err
	}
	var err error
	session.StartedAt, err = parseInstant(startedAt)
	if err != nil {
		return study.ReviewSession{}, err
	}
	session.EndedAt, err = parseNullableInstant(endedAt)
	if err != nil {
		return study.ReviewSession{}, err
	}
	session.CardLimit = nullableIntPointer(cardLimit)
	session.TimeLimitSeconds = nullableIntPointer(timeLimitSeconds)
	return session, nil
}

func scanReviewAttempt(scanner interface {
	Scan(dest ...any) error
}) (study.ReviewAttempt, error) {
	var attempt study.ReviewAttempt
	var answeredAt string
	var answerText, evidenceSummary sql.NullString
	if err := scanner.Scan(
		&attempt.ID,
		&attempt.SessionID,
		&attempt.CardID,
		&answeredAt,
		&answerText,
		&attempt.Rating,
		&attempt.Grader,
		&evidenceSummary,
		&attempt.ScheduleBeforeJSON,
		&attempt.ScheduleAfterJSON,
	); err != nil {
		return study.ReviewAttempt{}, err
	}
	var err error
	attempt.AnsweredAt, err = parseInstant(answeredAt)
	if err != nil {
		return study.ReviewAttempt{}, err
	}
	attempt.AnswerText = nullableStringPointer(answerText)
	attempt.EvidenceSummary = nullableStringPointer(evidenceSummary)
	return attempt, nil
}

func scheduleSelectSQL() string {
	return `
SELECT card_id, due_at, last_reviewed_at, reps, lapses, stability, difficulty,
       scheduled_days, elapsed_days, fsrs_state
FROM card_schedule`
}

func scanSchedule(scanner interface {
	Scan(dest ...any) error
}) (study.CardSchedule, error) {
	var schedule study.CardSchedule
	var dueAt string
	var lastReviewedAt sql.NullString
	if err := scanner.Scan(
		&schedule.CardID,
		&dueAt,
		&lastReviewedAt,
		&schedule.Reps,
		&schedule.Lapses,
		&schedule.Stability,
		&schedule.Difficulty,
		&schedule.ScheduledDays,
		&schedule.ElapsedDays,
		&schedule.FSRSState,
	); err != nil {
		return study.CardSchedule{}, err
	}
	var err error
	schedule.DueAt, err = parseInstant(dueAt)
	if err != nil {
		return study.CardSchedule{}, err
	}
	schedule.LastReviewedAt, err = parseNullableInstant(lastReviewedAt)
	if err != nil {
		return study.CardSchedule{}, err
	}
	return schedule, nil
}

func scanCardWithSchedule(scanner interface {
	Scan(dest ...any) error
}) (study.Card, study.CardSchedule, error) {
	var card study.Card
	var schedule study.CardSchedule
	var createdAt, updatedAt, dueAt string
	var archivedAt, lastReviewedAt sql.NullString
	if err := scanner.Scan(
		&card.ID,
		&card.Status,
		&card.Front,
		&card.Back,
		&createdAt,
		&updatedAt,
		&archivedAt,
		&schedule.CardID,
		&dueAt,
		&lastReviewedAt,
		&schedule.Reps,
		&schedule.Lapses,
		&schedule.Stability,
		&schedule.Difficulty,
		&schedule.ScheduledDays,
		&schedule.ElapsedDays,
		&schedule.FSRSState,
	); err != nil {
		return study.Card{}, study.CardSchedule{}, err
	}
	var err error
	card.CreatedAt, err = parseInstant(createdAt)
	if err != nil {
		return study.Card{}, study.CardSchedule{}, err
	}
	card.UpdatedAt, err = parseInstant(updatedAt)
	if err != nil {
		return study.Card{}, study.CardSchedule{}, err
	}
	card.ArchivedAt, err = parseNullableInstant(archivedAt)
	if err != nil {
		return study.Card{}, study.CardSchedule{}, err
	}
	schedule.DueAt, err = parseInstant(dueAt)
	if err != nil {
		return study.Card{}, study.CardSchedule{}, err
	}
	schedule.LastReviewedAt, err = parseNullableInstant(lastReviewedAt)
	if err != nil {
		return study.Card{}, study.CardSchedule{}, err
	}
	return card, schedule, nil
}

func serializeInstant(value time.Time) string {
	return value.UTC().Format(fixedInstantLayout)
}

func nullableInstant(value *time.Time) any {
	if value == nil {
		return nil
	}
	return serializeInstant(*value)
}

func parseInstant(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}

func parseNullableInstant(value sql.NullString) (*time.Time, error) {
	if !value.Valid {
		return nil, nil
	}
	parsed, err := parseInstant(value.String)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableStringPointer(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func nullableInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableIntPointer(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	converted := int(value.Int64)
	return &converted
}

func rollbackUnlessCommitted(tx *sql.Tx) {
	_ = tx.Rollback()
}

func wrapDatabaseError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}
