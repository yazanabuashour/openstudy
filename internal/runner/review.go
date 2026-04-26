package runner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yazanabuashour/openstudy/internal/localruntime"
	"github.com/yazanabuashour/openstudy/internal/study"
)

const (
	ReviewActionStartSession = "start_session"
	ReviewActionRecordAnswer = "record_answer"
	ReviewActionSummary      = "summary"
	ReviewActionFinish       = "finish_session"
)

type ReviewTaskRequest struct {
	Action          string             `json:"action"`
	Session         ReviewSessionInput `json:"session,omitempty"`
	SessionID       int64              `json:"session_id,omitempty"`
	CardID          int64              `json:"card_id,omitempty"`
	AnswerText      string             `json:"answer_text,omitempty"`
	Rating          string             `json:"rating,omitempty"`
	Grader          string             `json:"grader,omitempty"`
	EvidenceSummary string             `json:"evidence_summary,omitempty"`
	Now             string             `json:"now,omitempty"`
	AnsweredAt      string             `json:"answered_at,omitempty"`
}

type ReviewSessionInput struct {
	CardLimit        int `json:"card_limit,omitempty"`
	TimeLimitSeconds int `json:"time_limit_seconds,omitempty"`
}

type ReviewTaskResult struct {
	BaseResult
	Session    *ReviewSessionDTO       `json:"session,omitempty"`
	Attempt    *ReviewAttemptDTO       `json:"attempt,omitempty"`
	SummaryDTO *ReviewSummaryDTO       `json:"review_summary,omitempty"`
	Transition *SchedulerTransitionDTO `json:"transition,omitempty"`
	Cards      []CardDTO               `json:"cards,omitempty"`
}

func RunReviewTask(ctx context.Context, config Config, request ReviewTaskRequest) (ReviewTaskResult, error) {
	normalized, rejection := normalizeReviewTaskRequest(request)
	if rejection != "" {
		return rejectedReview(rejection), nil
	}
	if normalized.Action == ActionValidate {
		return ReviewTaskResult{BaseResult: validBase()}, nil
	}
	if normalized.Now != nil {
		config.Now = func() time.Time {
			return *normalized.Now
		}
	}

	runtime, err := localruntime.Open(ctx, localruntime.Config(config))
	if err != nil {
		return ReviewTaskResult{}, err
	}
	defer func() {
		_ = runtime.Close()
	}()

	switch normalized.Action {
	case ReviewActionStartSession:
		session, err := runtime.Service.StartReviewSession(ctx, study.StartReviewSessionInput{
			CardLimit:        normalized.CardLimit,
			TimeLimitSeconds: normalized.TimeLimitSeconds,
		})
		if err != nil {
			return rejectedReview(err.Error()), nil
		}
		cardLimit := defaultLimit
		if normalized.CardLimit != nil {
			cardLimit = *normalized.CardLimit
		}
		window, err := runtime.Service.ReviewWindow(ctx, cardLimit)
		if err != nil {
			return rejectedReview(err.Error()), nil
		}
		dto := toReviewSessionDTO(session)
		return ReviewTaskResult{
			BaseResult: BaseResult{Summary: fmt.Sprintf("started review session %d", session.ID)},
			Session:    &dto,
			Cards:      toCardsWithScheduleDTO(window.DueCards),
		}, nil
	case ReviewActionRecordAnswer:
		result, err := runtime.Service.RecordReview(ctx, study.RecordReviewInput{
			SessionID:       normalized.SessionID,
			CardID:          normalized.CardID,
			AnsweredAt:      normalized.AnsweredAt,
			AnswerText:      &normalized.AnswerText,
			Rating:          study.Rating(normalized.Rating),
			Grader:          study.Grader(normalized.Grader),
			EvidenceSummary: trimOptional(normalized.EvidenceSummary),
		})
		if err != nil {
			return rejectedReview(err.Error()), nil
		}
		transition, err := study.ExplainReviewAttempt(result.Attempt)
		if err != nil {
			return ReviewTaskResult{}, err
		}
		attemptDTO := toReviewAttemptDTO(result.Attempt)
		transitionDTO := toSchedulerTransitionDTO(transition)
		return ReviewTaskResult{
			BaseResult: BaseResult{Summary: fmt.Sprintf("recorded answer for card %d", result.Attempt.CardID)},
			Attempt:    &attemptDTO,
			Transition: &transitionDTO,
		}, nil
	case ReviewActionSummary:
		summary, err := runtime.Service.ReviewSessionSummary(ctx, normalized.SessionID)
		if err != nil {
			return rejectedReview(err.Error()), nil
		}
		dto := toReviewSummaryDTO(summary)
		return ReviewTaskResult{
			BaseResult: BaseResult{Summary: fmt.Sprintf("returned review session %d summary", normalized.SessionID)},
			SummaryDTO: &dto,
		}, nil
	case ReviewActionFinish:
		session, err := runtime.Service.FinishReviewSession(ctx, normalized.SessionID)
		if err != nil {
			return rejectedReview(err.Error()), nil
		}
		dto := toReviewSessionDTO(session)
		return ReviewTaskResult{
			BaseResult: BaseResult{Summary: fmt.Sprintf("finished review session %d", normalized.SessionID)},
			Session:    &dto,
		}, nil
	default:
		return ReviewTaskResult{}, fmt.Errorf("unsupported review task action %q", normalized.Action)
	}
}

type normalizedReviewTaskRequest struct {
	Action           string
	SessionID        int64
	CardID           int64
	AnswerText       string
	Rating           string
	Grader           string
	EvidenceSummary  string
	Now              *time.Time
	AnsweredAt       *time.Time
	CardLimit        *int
	TimeLimitSeconds *int
}

func normalizeReviewTaskRequest(request ReviewTaskRequest) (normalizedReviewTaskRequest, string) {
	action := strings.TrimSpace(request.Action)
	if action == "" {
		action = ActionValidate
	}
	now, rejection := optionalRFC3339(request.Now, "now")
	if rejection != "" {
		return normalizedReviewTaskRequest{}, rejection
	}
	answeredAt, rejection := optionalRFC3339(request.AnsweredAt, "answered_at")
	if rejection != "" {
		return normalizedReviewTaskRequest{}, rejection
	}
	normalized := normalizedReviewTaskRequest{
		Action:          action,
		SessionID:       request.SessionID,
		CardID:          request.CardID,
		AnswerText:      strings.TrimSpace(request.AnswerText),
		Rating:          strings.TrimSpace(request.Rating),
		Grader:          strings.TrimSpace(request.Grader),
		EvidenceSummary: strings.TrimSpace(request.EvidenceSummary),
		Now:             now,
		AnsweredAt:      answeredAt,
	}
	if request.Session.CardLimit < 0 {
		return normalizedReviewTaskRequest{}, "session.card_limit must be greater than or equal to 0"
	}
	if request.Session.CardLimit > maxLimit {
		return normalizedReviewTaskRequest{}, fmt.Sprintf("session.card_limit must be less than or equal to %d", maxLimit)
	}
	if request.Session.CardLimit > 0 {
		normalized.CardLimit = &request.Session.CardLimit
	}
	if request.Session.TimeLimitSeconds < 0 {
		return normalizedReviewTaskRequest{}, "session.time_limit_seconds must be greater than or equal to 0"
	}
	if request.Session.TimeLimitSeconds > 0 {
		normalized.TimeLimitSeconds = &request.Session.TimeLimitSeconds
	}

	switch action {
	case ActionValidate:
		return normalized, ""
	case ReviewActionStartSession:
		return normalized, ""
	case ReviewActionRecordAnswer:
		if rejection := requirePositiveID(normalized.SessionID, "session_id"); rejection != "" {
			return normalizedReviewTaskRequest{}, rejection
		}
		if rejection := requirePositiveID(normalized.CardID, "card_id"); rejection != "" {
			return normalizedReviewTaskRequest{}, rejection
		}
		if normalized.AnswerText == "" {
			return normalizedReviewTaskRequest{}, "answer_text is required"
		}
		if !validReviewRating(normalized.Rating) {
			return normalizedReviewTaskRequest{}, "rating must be again, hard, good, or easy"
		}
		if !validReviewGrader(normalized.Grader) {
			return normalizedReviewTaskRequest{}, "grader must be self or evidence"
		}
		return normalized, ""
	case ReviewActionSummary, ReviewActionFinish:
		if rejection := requirePositiveID(normalized.SessionID, "session_id"); rejection != "" {
			return normalizedReviewTaskRequest{}, rejection
		}
		return normalized, ""
	default:
		return normalizedReviewTaskRequest{}, fmt.Sprintf("unsupported review task action %q", action)
	}
}

func validReviewRating(value string) bool {
	switch value {
	case string(study.RatingAgain), string(study.RatingHard), string(study.RatingGood), string(study.RatingEasy):
		return true
	default:
		return false
	}
}

func validReviewGrader(value string) bool {
	switch value {
	case string(study.GraderSelf), string(study.GraderEvidence):
		return true
	default:
		return false
	}
}

func rejectedReview(reason string) ReviewTaskResult {
	return ReviewTaskResult{
		BaseResult: BaseResult{
			Rejected:        true,
			RejectionReason: reason,
			Summary:         reason,
		},
	}
}
