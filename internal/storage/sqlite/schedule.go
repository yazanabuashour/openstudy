package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/yazanabuashour/openstudy/internal/study"
)

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
