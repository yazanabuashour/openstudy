package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/yazanabuashour/openstudy/internal/study"
)

func TestRepositoryCardsSourcesDueAndArchive(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepository(t)
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)

	cardA, err := repo.CreateCard(ctx, study.CreateCardParams{
		Front: "What command lists ready work?",
		Back:  "bd ready",
		Now:   now,
		Schedule: study.CardSchedule{
			DueAt:     now.Add(-time.Hour),
			FSRSState: 0,
		},
	})
	if err != nil {
		t.Fatalf("create card A: %v", err)
	}
	cardB, err := repo.CreateCard(ctx, study.CreateCardParams{
		Front: "What file carries agent instructions?",
		Back:  "AGENTS.md",
		Now:   now,
		Schedule: study.CardSchedule{
			DueAt:     now.Add(-time.Minute),
			FSRSState: 0,
		},
	})
	if err != nil {
		t.Fatalf("create card B: %v", err)
	}
	_, err = repo.CreateCard(ctx, study.CreateCardParams{
		Front: "What command runs Go tests?",
		Back:  "mise exec -- go test ./...",
		Now:   now,
		Schedule: study.CardSchedule{
			DueAt:     now.Add(time.Hour),
			FSRSState: 0,
		},
	})
	if err != nil {
		t.Fatalf("create future card: %v", err)
	}

	label := "planning ADR"
	source, err := repo.AddSource(ctx, study.AddSourceParams{
		CardID:       cardA.ID,
		SourceSystem: "external-notes",
		SourceKey:    "note-123",
		Label:        &label,
		Now:          now,
	})
	if err != nil {
		t.Fatalf("add source: %v", err)
	}
	if source.SourceKey != "note-123" {
		t.Fatalf("source key = %q", source.SourceKey)
	}
	sources, err := repo.ListSources(ctx, cardA.ID)
	if err != nil {
		t.Fatalf("list sources: %v", err)
	}
	if len(sources) != 1 || sources[0].Label == nil || *sources[0].Label != label {
		t.Fatalf("sources = %#v", sources)
	}

	due, err := repo.ListDueCards(ctx, study.DueCardFilter{Now: now, Limit: 10})
	if err != nil {
		t.Fatalf("list due cards: %v", err)
	}
	if len(due) != 2 {
		t.Fatalf("due count = %d, want 2", len(due))
	}
	if due[0].ID != cardA.ID || due[1].ID != cardB.ID {
		t.Fatalf("due ordering ids = %d, %d", due[0].ID, due[1].ID)
	}

	archived, err := repo.ArchiveCard(ctx, study.ArchiveCardParams{ID: cardA.ID, Now: now.Add(time.Minute)})
	if err != nil {
		t.Fatalf("archive card: %v", err)
	}
	if archived.Status != study.CardStatusArchived || archived.ArchivedAt == nil {
		t.Fatalf("archived card = %#v", archived)
	}
	got, err := repo.GetCard(ctx, cardA.ID)
	if err != nil {
		t.Fatalf("get archived card: %v", err)
	}
	if got == nil || got.Status != study.CardStatusArchived {
		t.Fatalf("got archived card = %#v", got)
	}
	due, err = repo.ListDueCards(ctx, study.DueCardFilter{Now: now, Limit: 10})
	if err != nil {
		t.Fatalf("list due cards after archive: %v", err)
	}
	if len(due) != 1 || due[0].ID != cardB.ID {
		t.Fatalf("due after archive = %#v", due)
	}
}

func TestRepositoryListDueCardsDoesNotCompareSubsecondDueTimeEarly(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepository(t)
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)

	_, err := repo.CreateCard(ctx, study.CreateCardParams{
		Front: "When is this card due?",
		Back:  "Half a second later",
		Now:   now,
		Schedule: study.CardSchedule{
			DueAt:     now.Add(500 * time.Millisecond),
			FSRSState: 0,
		},
	})
	if err != nil {
		t.Fatalf("create card: %v", err)
	}

	due, err := repo.ListDueCards(ctx, study.DueCardFilter{Now: now, Limit: 10})
	if err != nil {
		t.Fatalf("list due cards: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("due count = %d, want 0", len(due))
	}
}

func TestServiceRecordReviewPersistsAttemptAndSchedule(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepository(t)
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	service := study.NewService(repo, study.WithClock(func() time.Time { return now }))

	card, err := service.CreateCard(ctx, study.CreateCardInput{
		Front: "What owns OpenStudy practice state?",
		Back:  "OpenStudy",
	})
	if err != nil {
		t.Fatalf("create card: %v", err)
	}
	session, err := service.StartReviewSession(ctx, study.StartReviewSessionInput{})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}

	answer := "OpenStudy owns the mutable review practice data."
	answeredAt := now.Add(5 * time.Minute)
	result, err := service.RecordReview(ctx, study.RecordReviewInput{
		SessionID:  session.ID,
		CardID:     card.ID,
		AnsweredAt: &answeredAt,
		AnswerText: &answer,
		Rating:     study.RatingGood,
		Grader:     study.GraderSelf,
	})
	if err != nil {
		t.Fatalf("record review: %v", err)
	}
	if result.Attempt.ID == 0 {
		t.Fatal("expected persisted attempt id")
	}
	if result.Transition.AttemptID != result.Attempt.ID || result.Transition.Rating != study.RatingGood {
		t.Fatalf("transition = %#v", result.Transition)
	}
	if result.Transition.Before.CardID != card.ID || result.Transition.After.CardID != card.ID {
		t.Fatalf("transition card ids = %d -> %d", result.Transition.Before.CardID, result.Transition.After.CardID)
	}
	if result.Transition.After.Reps != 1 {
		t.Fatalf("after reps = %d, want 1", result.Transition.After.Reps)
	}
	if result.Transition.After.LastReviewedAt == nil || !result.Transition.After.LastReviewedAt.Equal(answeredAt) {
		t.Fatalf("last reviewed at = %#v, want %s", result.Transition.After.LastReviewedAt, answeredAt)
	}
	if !result.Transition.After.DueAt.After(answeredAt) {
		t.Fatalf("due at = %s, want after %s", result.Transition.After.DueAt, answeredAt)
	}

	persisted, err := repo.GetCardSchedule(ctx, card.ID)
	if err != nil {
		t.Fatalf("get schedule: %v", err)
	}
	if persisted == nil || persisted.Reps != result.Transition.After.Reps || !persisted.DueAt.Equal(result.Transition.After.DueAt) {
		t.Fatalf("persisted schedule = %#v, result after = %#v", persisted, result.Transition.After)
	}

	summary, err := service.ReviewSessionSummary(ctx, session.ID)
	if err != nil {
		t.Fatalf("review session summary: %v", err)
	}
	if summary.AttemptCount != 1 || summary.RatingCounts[study.RatingGood] != 1 {
		t.Fatalf("summary = %#v", summary)
	}
}

func TestServiceReviewWindowAndValidation(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepository(t)
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	service := study.NewService(repo, study.WithClock(func() time.Time { return now }))

	card, err := service.CreateCard(ctx, study.CreateCardInput{
		Front: "Which tracker is used?",
		Back:  "bd",
	})
	if err != nil {
		t.Fatalf("create card: %v", err)
	}

	window, err := service.ReviewWindow(ctx, 5)
	if err != nil {
		t.Fatalf("review window: %v", err)
	}
	if len(window.DueCards) != 1 || window.DueCards[0].ID != card.ID {
		t.Fatalf("window = %#v", window)
	}

	if _, err := service.CreateCard(ctx, study.CreateCardInput{Front: " ", Back: "answer"}); err == nil {
		t.Fatal("expected missing front to fail")
	}
	if _, err := service.RecordReview(ctx, study.RecordReviewInput{
		SessionID: 1,
		CardID:    card.ID,
		Rating:    study.Rating("close"),
		Grader:    study.GraderSelf,
	}); err == nil {
		t.Fatal("expected unsupported rating to fail")
	}

	session, err := service.StartReviewSession(ctx, study.StartReviewSessionInput{})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	if _, err := service.FinishReviewSession(ctx, session.ID); err != nil {
		t.Fatalf("finish session: %v", err)
	}
	if _, err := service.RecordReview(ctx, study.RecordReviewInput{
		SessionID: session.ID,
		CardID:    card.ID,
		Rating:    study.RatingGood,
		Grader:    study.GraderSelf,
	}); err == nil {
		t.Fatal("expected completed session review to fail")
	}
}

func TestServiceListCardsWithSchedulesFilteringAndLimit(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepository(t)
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	service := study.NewService(repo, study.WithClock(func() time.Time { return now }))

	first, err := service.CreateCard(ctx, study.CreateCardInput{Front: "First active?", Back: "Yes"})
	if err != nil {
		t.Fatalf("create first card: %v", err)
	}
	second, err := service.CreateCard(ctx, study.CreateCardInput{Front: "Second active?", Back: "Yes"})
	if err != nil {
		t.Fatalf("create second card: %v", err)
	}
	archived, err := service.CreateCard(ctx, study.CreateCardInput{Front: "Archived?", Back: "Yes"})
	if err != nil {
		t.Fatalf("create archived card: %v", err)
	}
	if _, err := service.ArchiveCard(ctx, archived.ID); err != nil {
		t.Fatalf("archive card: %v", err)
	}

	active, err := service.ListCardsWithSchedules(ctx, study.ListCardsInput{
		Status: study.CardListStatusActive,
		Limit:  1,
	})
	if err != nil {
		t.Fatalf("list active cards: %v", err)
	}
	if len(active) != 1 || active[0].ID != first.ID || active[0].Schedule.CardID != first.ID {
		t.Fatalf("active cards = %#v", active)
	}

	all, err := service.ListCardsWithSchedules(ctx, study.ListCardsInput{
		Status: study.CardListStatusAll,
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("list all cards: %v", err)
	}
	if len(all) != 3 || all[0].ID != first.ID || all[1].ID != second.ID || all[2].ID != archived.ID {
		t.Fatalf("all cards = %#v", all)
	}
	if all[2].Status != study.CardStatusArchived || all[2].Schedule.CardID != archived.ID {
		t.Fatalf("archived card with schedule = %#v", all[2])
	}

	if _, err := service.ListCardsWithSchedules(ctx, study.ListCardsInput{Status: study.CardListStatus("missing"), Limit: 10}); err == nil {
		t.Fatal("expected unsupported status to fail")
	}
	if _, err := service.ListCardsWithSchedules(ctx, study.ListCardsInput{Status: study.CardListStatusActive}); err == nil {
		t.Fatal("expected missing limit to fail")
	}
}

func newTestRepository(t *testing.T) *Repository {
	t.Helper()
	ctx := context.Background()
	db, err := Open(filepath.Join(t.TempDir(), "data", "openstudy.sqlite"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	if err := ApplyMigrations(ctx, db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	return NewRepository(db)
}
