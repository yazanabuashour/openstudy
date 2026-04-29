package study

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	fsrs "github.com/open-spaced-repetition/go-fsrs/v4"
)

type Clock func() time.Time

type Option func(*Service)

type Service struct {
	repo Repository
	now  Clock
	fsrs *fsrs.FSRS
}

func WithClock(clock Clock) Option {
	return func(s *Service) {
		s.now = clock
	}
}

func NewService(repo Repository, opts ...Option) *Service {
	service := &Service{
		repo: repo,
		now: func() time.Time {
			return time.Now().UTC()
		},
		fsrs: fsrs.NewFSRS(fsrs.DefaultParam()),
	}
	for _, opt := range opts {
		opt(service)
	}
	return service
}

func (s *Service) CreateCard(ctx context.Context, input CreateCardInput) (Card, error) {
	front := strings.TrimSpace(input.Front)
	back := strings.TrimSpace(input.Back)
	if front == "" {
		return Card{}, errors.New("card front is required")
	}
	if back == "" {
		return Card{}, errors.New("card back is required")
	}

	now := s.now().UTC()
	schedule := scheduleFromFSRSCard(0, initialFSRSCard(now))
	return s.repo.CreateCard(ctx, CreateCardParams{
		Front:    front,
		Back:     back,
		Now:      now,
		Schedule: schedule,
	})
}

func (s *Service) ListCards(ctx context.Context) ([]Card, error) {
	return s.repo.ListCards(ctx)
}

func (s *Service) ListCardsWithSchedules(ctx context.Context, input ListCardsInput) ([]CardWithSchedule, error) {
	status := input.Status
	if status == "" {
		status = CardListStatusActive
	}
	if input.Limit <= 0 {
		return nil, errors.New("limit must be positive")
	}

	filter := ListCardsFilter{Limit: input.Limit}
	switch status {
	case CardListStatusActive:
		value := CardStatusActive
		filter.Status = &value
	case CardListStatusArchived:
		value := CardStatusArchived
		filter.Status = &value
	case CardListStatusAll:
	default:
		return nil, fmt.Errorf("unsupported card list status %q", status)
	}
	return s.repo.ListCardsWithSchedules(ctx, filter)
}

func (s *Service) GetCard(ctx context.Context, id int64) (*Card, error) {
	if id <= 0 {
		return nil, errors.New("card id is required")
	}
	return s.repo.GetCard(ctx, id)
}

func (s *Service) ArchiveCard(ctx context.Context, id int64) (Card, error) {
	if id <= 0 {
		return Card{}, errors.New("card id is required")
	}
	return s.repo.ArchiveCard(ctx, ArchiveCardParams{
		ID:  id,
		Now: s.now().UTC(),
	})
}

func (s *Service) AttachSource(ctx context.Context, input AttachSourceInput) (SourceReference, error) {
	if input.CardID <= 0 {
		return SourceReference{}, errors.New("card id is required")
	}
	if strings.TrimSpace(input.SourceSystem) == "" {
		return SourceReference{}, errors.New("source system is required")
	}
	if strings.TrimSpace(input.SourceKey) == "" {
		return SourceReference{}, errors.New("source key is required")
	}
	card, err := s.repo.GetCard(ctx, input.CardID)
	if err != nil {
		return SourceReference{}, err
	}
	if card == nil {
		return SourceReference{}, fmt.Errorf("card %d not found", input.CardID)
	}
	return s.repo.AddSource(ctx, AddSourceParams{
		CardID:       input.CardID,
		SourceSystem: strings.TrimSpace(input.SourceSystem),
		SourceKey:    strings.TrimSpace(input.SourceKey),
		SourceAnchor: trimOptional(input.SourceAnchor),
		Label:        trimOptional(input.Label),
		Now:          s.now().UTC(),
	})
}

func (s *Service) ListSources(ctx context.Context, cardID int64) ([]SourceReference, error) {
	if cardID <= 0 {
		return nil, errors.New("card id is required")
	}
	return s.repo.ListSources(ctx, cardID)
}

func (s *Service) StartReviewSession(ctx context.Context, input StartReviewSessionInput) (ReviewSession, error) {
	if input.CardLimit != nil && *input.CardLimit <= 0 {
		return ReviewSession{}, errors.New("card limit must be positive")
	}
	if input.TimeLimitSeconds != nil && *input.TimeLimitSeconds <= 0 {
		return ReviewSession{}, errors.New("time limit seconds must be positive")
	}
	return s.repo.CreateReviewSession(ctx, CreateReviewSessionParams{
		StartedAt:        s.now().UTC(),
		CardLimit:        input.CardLimit,
		TimeLimitSeconds: input.TimeLimitSeconds,
	})
}

func (s *Service) ReviewSessionSummary(ctx context.Context, id int64) (ReviewSessionSummary, error) {
	if id <= 0 {
		return ReviewSessionSummary{}, errors.New("session id is required")
	}
	return s.repo.ReviewSessionSummary(ctx, id)
}

func (s *Service) FinishReviewSession(ctx context.Context, id int64) (ReviewSession, error) {
	if id <= 0 {
		return ReviewSession{}, errors.New("session id is required")
	}
	return s.repo.FinishReviewSession(ctx, FinishReviewSessionParams{
		ID:      id,
		EndedAt: s.now().UTC(),
	})
}

func (s *Service) DueCards(ctx context.Context, limit int) ([]CardWithSchedule, error) {
	if limit <= 0 {
		return nil, errors.New("limit must be positive")
	}
	return s.repo.ListDueCards(ctx, DueCardFilter{
		Now:   s.now().UTC(),
		Limit: limit,
	})
}

func (s *Service) ReviewWindow(ctx context.Context, limit int) (ReviewWindow, error) {
	now := s.now().UTC()
	if limit <= 0 {
		return ReviewWindow{}, errors.New("limit must be positive")
	}
	cards, err := s.repo.ListDueCards(ctx, DueCardFilter{Now: now, Limit: limit})
	if err != nil {
		return ReviewWindow{}, err
	}
	return ReviewWindow{Now: now, DueCards: cards}, nil
}

func (s *Service) CardSchedule(ctx context.Context, cardID int64) (*CardSchedule, error) {
	if cardID <= 0 {
		return nil, errors.New("card id is required")
	}
	return s.repo.GetCardSchedule(ctx, cardID)
}

func (s *Service) RecordReview(ctx context.Context, input RecordReviewInput) (RecordReviewResult, error) {
	if input.SessionID <= 0 {
		return RecordReviewResult{}, errors.New("session id is required")
	}
	if input.CardID <= 0 {
		return RecordReviewResult{}, errors.New("card id is required")
	}
	if !validRating(input.Rating) {
		return RecordReviewResult{}, fmt.Errorf("unsupported rating %q", input.Rating)
	}
	if !validGrader(input.Grader) {
		return RecordReviewResult{}, fmt.Errorf("unsupported grader %q", input.Grader)
	}

	session, err := s.repo.GetReviewSession(ctx, input.SessionID)
	if err != nil {
		return RecordReviewResult{}, err
	}
	if session == nil {
		return RecordReviewResult{}, fmt.Errorf("review session %d not found", input.SessionID)
	}
	if session.Status != SessionStatusActive {
		return RecordReviewResult{}, fmt.Errorf("review session %d is not active", input.SessionID)
	}

	card, err := s.repo.GetCard(ctx, input.CardID)
	if err != nil {
		return RecordReviewResult{}, err
	}
	if card == nil {
		return RecordReviewResult{}, fmt.Errorf("card %d not found", input.CardID)
	}
	if card.Status == CardStatusArchived {
		return RecordReviewResult{}, fmt.Errorf("card %d is archived", input.CardID)
	}

	before, err := s.repo.GetCardSchedule(ctx, input.CardID)
	if err != nil {
		return RecordReviewResult{}, err
	}
	if before == nil {
		return RecordReviewResult{}, fmt.Errorf("card %d schedule not found", input.CardID)
	}

	answeredAt := s.now().UTC()
	if input.AnsweredAt != nil {
		answeredAt = input.AnsweredAt.UTC()
	}

	info := s.fsrs.Next(fsrsCardFromSchedule(*before), answeredAt, fsrsRating(input.Rating))
	after := scheduleFromFSRSCard(input.CardID, info.Card)

	attempt, err := s.repo.RecordReviewAttempt(ctx, RecordReviewAttemptParams{
		SessionID:       input.SessionID,
		CardID:          input.CardID,
		AnsweredAt:      answeredAt,
		AnswerText:      trimOptional(input.AnswerText),
		Rating:          input.Rating,
		Grader:          input.Grader,
		EvidenceSummary: trimOptional(input.EvidenceSummary),
		ScheduleBefore:  *before,
		ScheduleAfter:   after,
	})
	if err != nil {
		return RecordReviewResult{}, err
	}

	transition := SchedulerTransition{
		AttemptID: attempt.ID,
		CardID:    attempt.CardID,
		Rating:    attempt.Rating,
		Before:    *before,
		After:     after,
	}
	return RecordReviewResult{
		Attempt:    attempt,
		Transition: transition,
	}, nil
}

func initialFSRSCard(now time.Time) fsrs.Card {
	card := fsrs.NewCard()
	card.Due = now.UTC()
	return card
}

func scheduleFromFSRSCard(cardID int64, card fsrs.Card) CardSchedule {
	var lastReviewedAt *time.Time
	if !card.LastReview.IsZero() {
		value := card.LastReview.UTC()
		lastReviewedAt = &value
	}
	dueAt := card.Due.UTC()
	if dueAt.IsZero() {
		dueAt = time.Unix(0, 0).UTC()
	}
	return CardSchedule{
		CardID:         cardID,
		DueAt:          dueAt,
		LastReviewedAt: lastReviewedAt,
		Reps:           card.Reps,
		Lapses:         card.Lapses,
		Stability:      card.Stability,
		Difficulty:     card.Difficulty,
		ScheduledDays:  card.ScheduledDays,
		ElapsedDays:    card.ElapsedDays,
		FSRSState:      int(card.State),
	}
}

func fsrsCardFromSchedule(schedule CardSchedule) fsrs.Card {
	card := fsrs.NewCard()
	card.Due = schedule.DueAt.UTC()
	card.Stability = schedule.Stability
	card.Difficulty = schedule.Difficulty
	card.ElapsedDays = schedule.ElapsedDays
	card.ScheduledDays = schedule.ScheduledDays
	card.Reps = schedule.Reps
	card.Lapses = schedule.Lapses
	card.State = fsrs.State(schedule.FSRSState)
	if schedule.LastReviewedAt != nil {
		card.LastReview = schedule.LastReviewedAt.UTC()
	}
	return card
}

func fsrsRating(rating Rating) fsrs.Rating {
	switch rating {
	case RatingAgain:
		return fsrs.Again
	case RatingHard:
		return fsrs.Hard
	case RatingGood:
		return fsrs.Good
	case RatingEasy:
		return fsrs.Easy
	default:
		return fsrs.Good
	}
}

func validRating(rating Rating) bool {
	switch rating {
	case RatingAgain, RatingHard, RatingGood, RatingEasy:
		return true
	default:
		return false
	}
}

func validGrader(grader Grader) bool {
	switch grader {
	case GraderSelf, GraderEvidence:
		return true
	default:
		return false
	}
}

func trimOptional(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
