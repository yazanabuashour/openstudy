package study

import (
	"context"
	"time"
)

type CardStatus string

const (
	CardStatusActive   CardStatus = "active"
	CardStatusArchived CardStatus = "archived"
)

type CardListStatus string

const (
	CardListStatusActive   CardListStatus = "active"
	CardListStatusArchived CardListStatus = "archived"
	CardListStatusAll      CardListStatus = "all"
)

type Rating string

const (
	RatingAgain Rating = "again"
	RatingHard  Rating = "hard"
	RatingGood  Rating = "good"
	RatingEasy  Rating = "easy"
)

type Grader string

const (
	GraderSelf     Grader = "self"
	GraderEvidence Grader = "evidence"
)

type SessionStatus string

const (
	SessionStatusActive    SessionStatus = "active"
	SessionStatusCompleted SessionStatus = "completed"
)

type Card struct {
	ID         int64
	Status     CardStatus
	Front      string
	Back       string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	ArchivedAt *time.Time
}

type SourceReference struct {
	ID           int64
	CardID       int64
	SourceSystem string
	SourceKey    string
	SourceAnchor *string
	Label        *string
	CreatedAt    time.Time
}

type CardSchedule struct {
	CardID         int64
	DueAt          time.Time
	LastReviewedAt *time.Time
	Reps           uint64
	Lapses         uint64
	Stability      float64
	Difficulty     float64
	ScheduledDays  uint64
	ElapsedDays    uint64
	FSRSState      int
}

type CardWithSchedule struct {
	Card
	Schedule CardSchedule
}

type ReviewSession struct {
	ID               int64
	StartedAt        time.Time
	EndedAt          *time.Time
	Status           SessionStatus
	CardLimit        *int
	TimeLimitSeconds *int
}

type ReviewAttempt struct {
	ID              int64
	SessionID       int64
	CardID          int64
	AnsweredAt      time.Time
	AnswerText      *string
	Rating          Rating
	Grader          Grader
	EvidenceSummary *string
}

type ReviewSessionSummary struct {
	Session      ReviewSession
	AttemptCount int
	RatingCounts map[Rating]int
}

type SchedulerTransition struct {
	AttemptID int64
	CardID    int64
	Rating    Rating
	Before    CardSchedule
	After     CardSchedule
}

type CreateCardInput struct {
	Front string
	Back  string
}

type ListCardsInput struct {
	Status CardListStatus
	Limit  int
}

type AttachSourceInput struct {
	CardID       int64
	SourceSystem string
	SourceKey    string
	SourceAnchor *string
	Label        *string
}

type StartReviewSessionInput struct {
	CardLimit        *int
	TimeLimitSeconds *int
}

type RecordReviewInput struct {
	SessionID       int64
	CardID          int64
	AnsweredAt      *time.Time
	AnswerText      *string
	Rating          Rating
	Grader          Grader
	EvidenceSummary *string
}

type RecordReviewResult struct {
	Attempt    ReviewAttempt
	Transition SchedulerTransition
}

type ListCardsFilter struct {
	Status *CardStatus
	Limit  int
}

type DueCardFilter struct {
	Now   time.Time
	Limit int
}

type ReviewWindow struct {
	Now      time.Time
	DueCards []CardWithSchedule
}

type Repository interface {
	CreateCard(ctx context.Context, params CreateCardParams) (Card, error)
	ListCards(ctx context.Context) ([]Card, error)
	ListCardsWithSchedules(ctx context.Context, filter ListCardsFilter) ([]CardWithSchedule, error)
	GetCard(ctx context.Context, id int64) (*Card, error)
	ArchiveCard(ctx context.Context, params ArchiveCardParams) (Card, error)
	AddSource(ctx context.Context, params AddSourceParams) (SourceReference, error)
	ListSources(ctx context.Context, cardID int64) ([]SourceReference, error)
	GetReviewSession(ctx context.Context, id int64) (*ReviewSession, error)
	CreateReviewSession(ctx context.Context, params CreateReviewSessionParams) (ReviewSession, error)
	FinishReviewSession(ctx context.Context, params FinishReviewSessionParams) (ReviewSession, error)
	ReviewSessionSummary(ctx context.Context, id int64) (ReviewSessionSummary, error)
	GetCardSchedule(ctx context.Context, cardID int64) (*CardSchedule, error)
	ListDueCards(ctx context.Context, filter DueCardFilter) ([]CardWithSchedule, error)
	RecordReviewAttempt(ctx context.Context, params RecordReviewAttemptParams) (ReviewAttempt, error)
}

type CreateCardParams struct {
	Front    string
	Back     string
	Now      time.Time
	Schedule CardSchedule
}

type ArchiveCardParams struct {
	ID  int64
	Now time.Time
}

type AddSourceParams struct {
	CardID       int64
	SourceSystem string
	SourceKey    string
	SourceAnchor *string
	Label        *string
	Now          time.Time
}

type CreateReviewSessionParams struct {
	StartedAt        time.Time
	CardLimit        *int
	TimeLimitSeconds *int
}

type FinishReviewSessionParams struct {
	ID      int64
	EndedAt time.Time
}

type RecordReviewAttemptParams struct {
	SessionID       int64
	CardID          int64
	AnsweredAt      time.Time
	AnswerText      *string
	Rating          Rating
	Grader          Grader
	EvidenceSummary *string
	ScheduleBefore  CardSchedule
	ScheduleAfter   CardSchedule
}
