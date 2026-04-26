package runner

import (
	"time"

	"github.com/yazanabuashour/openstudy/internal/localruntime"
	"github.com/yazanabuashour/openstudy/internal/study"
)

const (
	ActionValidate = "validate"
)

type Config = localruntime.Config

type BaseResult struct {
	Rejected        bool   `json:"rejected"`
	RejectionReason string `json:"rejection_reason,omitempty"`
	Summary         string `json:"summary"`
}

type CardDTO struct {
	ID         int64        `json:"id"`
	Status     string       `json:"status"`
	Front      string       `json:"front"`
	Back       string       `json:"back"`
	CreatedAt  string       `json:"created_at"`
	UpdatedAt  string       `json:"updated_at"`
	ArchivedAt *string      `json:"archived_at,omitempty"`
	Schedule   *ScheduleDTO `json:"schedule,omitempty"`
}

type ScheduleDTO struct {
	DueAt          string  `json:"due_at"`
	LastReviewedAt *string `json:"last_reviewed_at,omitempty"`
	Reps           uint64  `json:"reps"`
	Lapses         uint64  `json:"lapses"`
	Stability      float64 `json:"stability"`
	Difficulty     float64 `json:"difficulty"`
	ScheduledDays  uint64  `json:"scheduled_days"`
	ElapsedDays    uint64  `json:"elapsed_days"`
	FSRSState      int     `json:"fsrs_state"`
}

type SourceDTO struct {
	ID           int64   `json:"id"`
	CardID       int64   `json:"card_id"`
	SourceSystem string  `json:"source_system"`
	SourceKey    string  `json:"source_key"`
	SourceAnchor *string `json:"source_anchor,omitempty"`
	Label        *string `json:"label,omitempty"`
	CreatedAt    string  `json:"created_at"`
}

type ReviewSessionDTO struct {
	ID               int64   `json:"id"`
	StartedAt        string  `json:"started_at"`
	EndedAt          *string `json:"ended_at,omitempty"`
	Status           string  `json:"status"`
	CardLimit        *int    `json:"card_limit,omitempty"`
	TimeLimitSeconds *int    `json:"time_limit_seconds,omitempty"`
}

type ReviewAttemptDTO struct {
	ID              int64   `json:"id"`
	SessionID       int64   `json:"session_id"`
	CardID          int64   `json:"card_id"`
	AnsweredAt      string  `json:"answered_at"`
	AnswerText      *string `json:"answer_text,omitempty"`
	Rating          string  `json:"rating"`
	Grader          string  `json:"grader"`
	EvidenceSummary *string `json:"evidence_summary,omitempty"`
}

type ReviewSummaryDTO struct {
	Session      ReviewSessionDTO `json:"session"`
	AttemptCount int              `json:"attempt_count"`
	RatingCounts map[string]int   `json:"rating_counts"`
}

type SchedulerTransitionDTO struct {
	AttemptID int64       `json:"attempt_id"`
	CardID    int64       `json:"card_id"`
	Rating    string      `json:"rating"`
	Before    ScheduleDTO `json:"before"`
	After     ScheduleDTO `json:"after"`
}

func validBase() BaseResult {
	return BaseResult{Summary: "valid"}
}

func formatInstant(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func formatOptionalInstant(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := formatInstant(*value)
	return &formatted
}

func toCardDTO(card study.Card, schedule *study.CardSchedule) CardDTO {
	dto := CardDTO{
		ID:         card.ID,
		Status:     string(card.Status),
		Front:      card.Front,
		Back:       card.Back,
		CreatedAt:  formatInstant(card.CreatedAt),
		UpdatedAt:  formatInstant(card.UpdatedAt),
		ArchivedAt: formatOptionalInstant(card.ArchivedAt),
	}
	if schedule != nil {
		dto.Schedule = pointerToScheduleDTO(*schedule)
	}
	return dto
}

func toCardWithScheduleDTO(card study.CardWithSchedule) CardDTO {
	return toCardDTO(card.Card, &card.Schedule)
}

func toCardsWithScheduleDTO(cards []study.CardWithSchedule) []CardDTO {
	out := make([]CardDTO, 0, len(cards))
	for _, card := range cards {
		out = append(out, toCardWithScheduleDTO(card))
	}
	return out
}

func pointerToScheduleDTO(schedule study.CardSchedule) *ScheduleDTO {
	dto := toScheduleDTO(schedule)
	return &dto
}

func toScheduleDTO(schedule study.CardSchedule) ScheduleDTO {
	return ScheduleDTO{
		DueAt:          formatInstant(schedule.DueAt),
		LastReviewedAt: formatOptionalInstant(schedule.LastReviewedAt),
		Reps:           schedule.Reps,
		Lapses:         schedule.Lapses,
		Stability:      schedule.Stability,
		Difficulty:     schedule.Difficulty,
		ScheduledDays:  schedule.ScheduledDays,
		ElapsedDays:    schedule.ElapsedDays,
		FSRSState:      schedule.FSRSState,
	}
}

func toSourceDTO(source study.SourceReference) SourceDTO {
	return SourceDTO{
		ID:           source.ID,
		CardID:       source.CardID,
		SourceSystem: source.SourceSystem,
		SourceKey:    source.SourceKey,
		SourceAnchor: source.SourceAnchor,
		Label:        source.Label,
		CreatedAt:    formatInstant(source.CreatedAt),
	}
}

func toSourcesDTO(sources []study.SourceReference) []SourceDTO {
	out := make([]SourceDTO, 0, len(sources))
	for _, source := range sources {
		out = append(out, toSourceDTO(source))
	}
	return out
}

func toReviewSessionDTO(session study.ReviewSession) ReviewSessionDTO {
	return ReviewSessionDTO{
		ID:               session.ID,
		StartedAt:        formatInstant(session.StartedAt),
		EndedAt:          formatOptionalInstant(session.EndedAt),
		Status:           string(session.Status),
		CardLimit:        session.CardLimit,
		TimeLimitSeconds: session.TimeLimitSeconds,
	}
}

func toReviewAttemptDTO(attempt study.ReviewAttempt) ReviewAttemptDTO {
	return ReviewAttemptDTO{
		ID:              attempt.ID,
		SessionID:       attempt.SessionID,
		CardID:          attempt.CardID,
		AnsweredAt:      formatInstant(attempt.AnsweredAt),
		AnswerText:      attempt.AnswerText,
		Rating:          string(attempt.Rating),
		Grader:          string(attempt.Grader),
		EvidenceSummary: attempt.EvidenceSummary,
	}
}

func toReviewSummaryDTO(summary study.ReviewSessionSummary) ReviewSummaryDTO {
	counts := map[string]int{}
	for rating, count := range summary.RatingCounts {
		counts[string(rating)] = count
	}
	return ReviewSummaryDTO{
		Session:      toReviewSessionDTO(summary.Session),
		AttemptCount: summary.AttemptCount,
		RatingCounts: counts,
	}
}

func toSchedulerTransitionDTO(transition study.SchedulerTransition) SchedulerTransitionDTO {
	return SchedulerTransitionDTO{
		AttemptID: transition.AttemptID,
		CardID:    transition.CardID,
		Rating:    string(transition.Rating),
		Before:    toScheduleDTO(transition.Before),
		After:     toScheduleDTO(transition.After),
	}
}
