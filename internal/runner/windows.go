package runner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yazanabuashour/openstudy/internal/localruntime"
)

const (
	WindowsActionDueCards     = "due_cards"
	WindowsActionReviewWindow = "review_window"
)

type WindowsTaskRequest struct {
	Action string `json:"action"`
	Limit  int    `json:"limit,omitempty"`
	Now    string `json:"now,omitempty"`
}

type WindowsTaskResult struct {
	BaseResult
	Now   string    `json:"now,omitempty"`
	Cards []CardDTO `json:"cards,omitempty"`
}

func RunWindowsTask(ctx context.Context, config Config, request WindowsTaskRequest) (WindowsTaskResult, error) {
	normalized, rejection := normalizeWindowsTaskRequest(request)
	if rejection != "" {
		return rejectedWindows(rejection), nil
	}
	if normalized.Action == ActionValidate {
		return WindowsTaskResult{BaseResult: validBase()}, nil
	}
	if normalized.Now != nil {
		config.Now = func() time.Time {
			return *normalized.Now
		}
	}

	runtime, err := localruntime.Open(ctx, localruntime.Config(config))
	if err != nil {
		return WindowsTaskResult{}, err
	}
	defer func() {
		_ = runtime.Close()
	}()

	window, err := runtime.Service.ReviewWindow(ctx, normalized.Limit)
	if err != nil {
		return rejectedWindows(err.Error()), nil
	}
	summary := fmt.Sprintf("returned review window with %d due cards", len(window.DueCards))
	if normalized.Action == WindowsActionDueCards {
		summary = fmt.Sprintf("returned %d due cards", len(window.DueCards))
	}
	return WindowsTaskResult{
		BaseResult: BaseResult{Summary: summary},
		Now:        formatInstant(window.Now),
		Cards:      toCardsWithScheduleDTO(window.DueCards),
	}, nil
}

type normalizedWindowsTaskRequest struct {
	Action string
	Limit  int
	Now    *time.Time
}

func normalizeWindowsTaskRequest(request WindowsTaskRequest) (normalizedWindowsTaskRequest, string) {
	action := strings.TrimSpace(request.Action)
	if action == "" {
		action = ActionValidate
	}
	limit, rejection := normalizeLimit(request.Limit)
	if rejection != "" {
		return normalizedWindowsTaskRequest{}, rejection
	}
	now, rejection := optionalRFC3339(request.Now, "now")
	if rejection != "" {
		return normalizedWindowsTaskRequest{}, rejection
	}
	normalized := normalizedWindowsTaskRequest{
		Action: action,
		Limit:  limit,
		Now:    now,
	}
	switch action {
	case ActionValidate, WindowsActionDueCards, WindowsActionReviewWindow:
		return normalized, ""
	default:
		return normalizedWindowsTaskRequest{}, fmt.Sprintf("unsupported windows task action %q", action)
	}
}

func rejectedWindows(reason string) WindowsTaskResult {
	return WindowsTaskResult{
		BaseResult: BaseResult{
			Rejected:        true,
			RejectionReason: reason,
			Summary:         reason,
		},
	}
}
