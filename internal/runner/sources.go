package runner

import (
	"context"
	"fmt"
	"strings"

	"github.com/yazanabuashour/openstudy/internal/study"
)

const (
	SourcesActionAttach = "attach_source"
	SourcesActionList   = "list_sources"
)

type SourcesTaskRequest struct {
	Action string      `json:"action"`
	CardID int64       `json:"card_id,omitempty"`
	Source SourceInput `json:"source,omitempty"`
}

type SourceInput struct {
	SourceSystem string `json:"source_system"`
	SourceKey    string `json:"source_key"`
	SourceAnchor string `json:"source_anchor,omitempty"`
	Label        string `json:"label,omitempty"`
}

type SourcesTaskResult struct {
	BaseResult
	Source  *SourceDTO  `json:"source,omitempty"`
	Sources []SourceDTO `json:"sources,omitempty"`
}

func RunSourcesTask(ctx context.Context, config Config, request SourcesTaskRequest) (SourcesTaskResult, error) {
	normalized, rejection := normalizeSourcesTaskRequest(request)
	if rejection != "" {
		return rejectedSources(rejection), nil
	}
	if normalized.Action == ActionValidate {
		return SourcesTaskResult{BaseResult: validBase()}, nil
	}

	return withStudyService(ctx, config, func(service *study.Service) (SourcesTaskResult, error) {
		switch normalized.Action {
		case SourcesActionAttach:
			source, err := service.AttachSource(ctx, study.AttachSourceInput{
				CardID:       normalized.CardID,
				SourceSystem: normalized.Source.SourceSystem,
				SourceKey:    normalized.Source.SourceKey,
				SourceAnchor: trimOptional(normalized.Source.SourceAnchor),
				Label:        trimOptional(normalized.Source.Label),
			})
			if err != nil {
				return rejectedSources(err.Error()), nil
			}
			dto := toSourceDTO(source)
			return SourcesTaskResult{
				BaseResult: BaseResult{Summary: fmt.Sprintf("attached source %d to card %d", source.ID, source.CardID)},
				Source:     &dto,
			}, nil
		case SourcesActionList:
			sources, err := service.ListSources(ctx, normalized.CardID)
			if err != nil {
				return rejectedSources(err.Error()), nil
			}
			return SourcesTaskResult{
				BaseResult: BaseResult{Summary: fmt.Sprintf("returned %d sources", len(sources))},
				Sources:    toSourcesDTO(sources),
			}, nil
		default:
			return SourcesTaskResult{}, fmt.Errorf("unsupported sources task action %q", normalized.Action)
		}
	})
}

type normalizedSourcesTaskRequest struct {
	Action string
	CardID int64
	Source SourceInput
}

func normalizeSourcesTaskRequest(request SourcesTaskRequest) (normalizedSourcesTaskRequest, string) {
	action := strings.TrimSpace(request.Action)
	if action == "" {
		action = ActionValidate
	}
	normalized := normalizedSourcesTaskRequest{
		Action: action,
		CardID: request.CardID,
		Source: SourceInput{
			SourceSystem: strings.TrimSpace(request.Source.SourceSystem),
			SourceKey:    strings.TrimSpace(request.Source.SourceKey),
			SourceAnchor: strings.TrimSpace(request.Source.SourceAnchor),
			Label:        strings.TrimSpace(request.Source.Label),
		},
	}
	switch action {
	case ActionValidate:
		return normalized, ""
	case SourcesActionAttach:
		if rejection := requirePositiveID(normalized.CardID, "card_id"); rejection != "" {
			return normalizedSourcesTaskRequest{}, rejection
		}
		if normalized.Source.SourceSystem == "" {
			return normalizedSourcesTaskRequest{}, "source.source_system is required"
		}
		if normalized.Source.SourceKey == "" {
			return normalizedSourcesTaskRequest{}, "source.source_key is required"
		}
		return normalized, ""
	case SourcesActionList:
		if rejection := requirePositiveID(normalized.CardID, "card_id"); rejection != "" {
			return normalizedSourcesTaskRequest{}, rejection
		}
		return normalized, ""
	default:
		return normalizedSourcesTaskRequest{}, fmt.Sprintf("unsupported sources task action %q", action)
	}
}

func rejectedSources(reason string) SourcesTaskResult {
	return SourcesTaskResult{BaseResult: rejectedBase(reason)}
}
