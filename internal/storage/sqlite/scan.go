package sqlite

import (
	"database/sql"

	"github.com/yazanabuashour/openstudy/internal/study"
)

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
