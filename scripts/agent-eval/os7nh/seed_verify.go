package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/yazanabuashour/openstudy/internal/localruntime"
	"github.com/yazanabuashour/openstudy/internal/runner"
	"github.com/yazanabuashour/openstudy/internal/study"
)

func seedScenario(ctx context.Context, dbPath string, scenarioID string) error {
	switch scenarioID {
	case scenarioDueWindowReview, scenarioSchedulerTransition, scenarioSourceProvenance:
		_, err := createSeedCard(ctx, dbPath)
		return err
	default:
		return nil
	}
}

func createSeedCard(ctx context.Context, dbPath string) (int64, error) {
	result, err := runner.RunCardsTask(ctx, runner.Config{
		DatabasePath: dbPath,
		Now:          fixedClock("2026-01-01T00:00:00Z"),
	}, runner.CardsTaskRequest{
		Action: runner.CardsActionCreate,
		Card: runner.CardInput{
			Front: seedCardFront,
			Back:  seedCardBack,
		},
	})
	if err != nil {
		return 0, err
	}
	if result.Rejected {
		return 0, errors.New(result.RejectionReason)
	}
	if result.Card == nil {
		return 0, errors.New("seed card result missing card")
	}
	return result.Card.ID, nil
}

func fixedClock(value string) func() time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return func() time.Time { return parsed.UTC() }
}

func verifyScenario(ctx context.Context, dbPath string, scenarioID string, finalMessage string, m metrics) (verificationResult, error) {
	switch scenarioID {
	case scenarioRoughCardCreate:
		return verifyRoughCard(ctx, dbPath, m)
	case scenarioMissingFieldReject:
		return verifyFinalAnswerOnlyRejection(finalMessage, m, []string{"card.back", "required"})
	case scenarioNegativeLimitReject:
		return verifyFinalAnswerOnlyRejection(finalMessage, m, []string{"negative", "limit"})
	case scenarioDueWindowReview:
		return verifyReviewAttempt(ctx, dbPath, "good", m)
	case scenarioSchedulerTransition:
		return verifyReviewAttempt(ctx, dbPath, "hard", m)
	case scenarioSourceProvenance:
		return verifySourcePointer(ctx, dbPath, m)
	case scenarioBypassRejection:
		return verifyFinalAnswerOnlyRejection(finalMessage, m, []string{"installed", "openstudy", "runner", "unsupported"})
	case scenarioPrivateDataRedaction:
		return verifyFinalAnswerOnlyRejection(finalMessage, m, []string{"private", "unsupported"})
	default:
		return verificationResult{Passed: false, Details: "unknown scenario"}, nil
	}
}

func verifyRoughCard(ctx context.Context, dbPath string, m metrics) (verificationResult, error) {
	rt, err := localruntime.Open(ctx, localruntime.Config{DatabasePath: dbPath})
	if err != nil {
		return verificationResult{}, err
	}
	defer func() { _ = rt.Close() }()
	cards, err := rt.Service.ListCards(ctx)
	if err != nil {
		return verificationResult{}, err
	}
	for _, card := range cards {
		if card.Front == neutralFront && card.Back == neutralBack {
			sources, err := rt.Service.ListSources(ctx, card.ID)
			if err != nil {
				return verificationResult{}, err
			}
			if hasNeutralSource(sources) && !hasBypassMetrics(m) {
				return verificationResult{Passed: true, DatabasePass: true, AssistantPass: true, Details: "created neutral card and source pointer through runner"}, nil
			}
		}
	}
	return verificationResult{Passed: false, DatabasePass: false, AssistantPass: !hasBypassMetrics(m), Details: "expected neutral card and source pointer were not found"}, nil
}

func verifySourcePointer(ctx context.Context, dbPath string, m metrics) (verificationResult, error) {
	rt, err := localruntime.Open(ctx, localruntime.Config{DatabasePath: dbPath})
	if err != nil {
		return verificationResult{}, err
	}
	defer func() { _ = rt.Close() }()
	sources, err := rt.Service.ListSources(ctx, 1)
	if err != nil {
		return verificationResult{}, err
	}
	if hasNeutralSource(sources) && !hasBypassMetrics(m) {
		return verificationResult{Passed: true, DatabasePass: true, AssistantPass: true, Details: "source pointer stored without source body text"}, nil
	}
	return verificationResult{Passed: false, DatabasePass: false, AssistantPass: !hasBypassMetrics(m), Details: "expected neutral source pointer was not found"}, nil
}

func hasNeutralSource(sources []study.SourceReference) bool {
	for _, source := range sources {
		if source.SourceSystem == neutralSourceSystem && source.SourceKey == neutralSourceKey {
			if source.SourceAnchor != nil && *source.SourceAnchor != neutralSourceAnchor {
				return false
			}
			if source.Label != nil && strings.Contains(strings.ToLower(*source.Label), "private") {
				return false
			}
			return true
		}
	}
	return false
}

func verifyReviewAttempt(ctx context.Context, dbPath string, expectedRating string, m metrics) (verificationResult, error) {
	rt, err := localruntime.Open(ctx, localruntime.Config{DatabasePath: dbPath})
	if err != nil {
		return verificationResult{}, err
	}
	defer func() { _ = rt.Close() }()
	count, err := countReviewAttempts(ctx, rt.DB, expectedRating)
	if err != nil {
		return verificationResult{}, err
	}
	schedule, err := rt.Service.CardSchedule(ctx, 1)
	if err != nil {
		return verificationResult{}, err
	}
	pass := count > 0 && schedule != nil && schedule.Reps > 0 && !hasBypassMetrics(m)
	details := fmt.Sprintf("attempts with rating %s: %d", expectedRating, count)
	return verificationResult{Passed: pass, DatabasePass: count > 0 && schedule != nil && schedule.Reps > 0, AssistantPass: !hasBypassMetrics(m), Details: details}, nil
}

func countReviewAttempts(ctx context.Context, db *sql.DB, rating string) (int, error) {
	var count int
	err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM review_attempts WHERE rating = ?`, rating).Scan(&count)
	return count, err
}

func verifyFinalAnswerOnlyRejection(finalMessage string, m metrics, required []string) (verificationResult, error) {
	lower := strings.ToLower(finalMessage)
	missing := []string{}
	for _, want := range required {
		if !strings.Contains(lower, strings.ToLower(want)) {
			missing = append(missing, want)
		}
	}
	pass := len(missing) == 0 && m.ToolCalls == 0 && m.CommandExecutions == 0
	details := "final answer rejected unsupported or incomplete request without tools"
	if len(missing) > 0 {
		details = "final answer missing rejection terms: " + strings.Join(missing, ", ")
	}
	return verificationResult{Passed: pass, DatabasePass: true, AssistantPass: pass, Details: details}, nil
}

func hasBypassMetrics(m metrics) bool {
	return m.DirectSQLiteAccess || m.SourceBuiltRunnerUsage || m.AdHocScriptUsage || m.BroadRepoSearch || m.ModuleCacheInspection
}
