package runner

import (
	"context"
	"fmt"
	"strings"

	"github.com/yazanabuashour/openstudy/internal/study"
)

const (
	CardsActionCreate   = "create_card"
	CardsActionList     = "list_cards"
	CardsActionGet      = "get_card"
	CardsActionArchive  = "archive_card"
	CardsStatusActive   = "active"
	CardsStatusArchived = "archived"
	CardsStatusAll      = "all"
)

type CardsTaskRequest struct {
	Action string    `json:"action"`
	Card   CardInput `json:"card,omitempty"`
	CardID int64     `json:"card_id,omitempty"`
	Status string    `json:"status,omitempty"`
	Limit  int       `json:"limit,omitempty"`
}

type CardInput struct {
	Front string `json:"front"`
	Back  string `json:"back"`
}

type CardsTaskResult struct {
	BaseResult
	Card  *CardDTO  `json:"card,omitempty"`
	Cards []CardDTO `json:"cards,omitempty"`
}

func RunCardsTask(ctx context.Context, config Config, request CardsTaskRequest) (CardsTaskResult, error) {
	normalized, rejection := normalizeCardsTaskRequest(request)
	if rejection != "" {
		return rejectedCards(rejection), nil
	}
	if normalized.Action == ActionValidate {
		return CardsTaskResult{BaseResult: validBase()}, nil
	}

	return withStudyService(ctx, config, func(service *study.Service) (CardsTaskResult, error) {
		switch normalized.Action {
		case CardsActionCreate:
			card, err := service.CreateCard(ctx, study.CreateCardInput{
				Front: normalized.Card.Front,
				Back:  normalized.Card.Back,
			})
			if err != nil {
				return rejectedCards(err.Error()), nil
			}
			schedule, err := service.CardSchedule(ctx, card.ID)
			if err != nil {
				return CardsTaskResult{}, err
			}
			dto := toCardDTO(card, schedule)
			return CardsTaskResult{
				BaseResult: BaseResult{Summary: fmt.Sprintf("created card %d", card.ID)},
				Card:       &dto,
			}, nil
		case CardsActionList:
			cards, err := service.ListCardsWithSchedules(ctx, study.ListCardsInput{
				Status: study.CardListStatus(normalized.Status),
				Limit:  normalized.Limit,
			})
			if err != nil {
				return CardsTaskResult{}, err
			}
			return CardsTaskResult{
				BaseResult: BaseResult{Summary: fmt.Sprintf("returned %d cards", len(cards))},
				Cards:      toCardsWithScheduleDTO(cards),
			}, nil
		case CardsActionGet:
			card, err := service.GetCard(ctx, normalized.CardID)
			if err != nil {
				return rejectedCards(err.Error()), nil
			}
			if card == nil {
				return rejectedCards(fmt.Sprintf("card %d not found", normalized.CardID)), nil
			}
			schedule, err := service.CardSchedule(ctx, card.ID)
			if err != nil {
				return CardsTaskResult{}, err
			}
			dto := toCardDTO(*card, schedule)
			return CardsTaskResult{
				BaseResult: BaseResult{Summary: fmt.Sprintf("returned card %d", card.ID)},
				Card:       &dto,
			}, nil
		case CardsActionArchive:
			card, err := service.ArchiveCard(ctx, normalized.CardID)
			if err != nil {
				return rejectedCards(err.Error()), nil
			}
			schedule, err := service.CardSchedule(ctx, card.ID)
			if err != nil {
				return CardsTaskResult{}, err
			}
			dto := toCardDTO(card, schedule)
			return CardsTaskResult{
				BaseResult: BaseResult{Summary: fmt.Sprintf("archived card %d", card.ID)},
				Card:       &dto,
			}, nil
		default:
			return CardsTaskResult{}, fmt.Errorf("unsupported cards task action %q", normalized.Action)
		}
	})
}

type normalizedCardsTaskRequest struct {
	Action string
	Card   CardInput
	CardID int64
	Status string
	Limit  int
}

func normalizeCardsTaskRequest(request CardsTaskRequest) (normalizedCardsTaskRequest, string) {
	action := strings.TrimSpace(request.Action)
	if action == "" {
		action = ActionValidate
	}
	normalized := normalizedCardsTaskRequest{
		Action: action,
		Card: CardInput{
			Front: strings.TrimSpace(request.Card.Front),
			Back:  strings.TrimSpace(request.Card.Back),
		},
		CardID: request.CardID,
		Status: strings.TrimSpace(request.Status),
		Limit:  request.Limit,
	}
	if normalized.Status == "" {
		normalized.Status = CardsStatusActive
	}
	if normalized.Limit < 0 {
		return normalizedCardsTaskRequest{}, "limit must be greater than or equal to 0"
	}
	switch action {
	case ActionValidate:
		return normalized, ""
	case CardsActionCreate:
		if normalized.Card.Front == "" {
			return normalizedCardsTaskRequest{}, "card.front is required"
		}
		if normalized.Card.Back == "" {
			return normalizedCardsTaskRequest{}, "card.back is required"
		}
		return normalized, ""
	case CardsActionList:
		if normalized.Status != CardsStatusActive && normalized.Status != CardsStatusArchived && normalized.Status != CardsStatusAll {
			return normalizedCardsTaskRequest{}, "status must be active, archived, or all"
		}
		if normalized.Limit == 0 {
			normalized.Limit = 50
		}
		if normalized.Limit > maxLimit {
			return normalizedCardsTaskRequest{}, fmt.Sprintf("limit must be less than or equal to %d", maxLimit)
		}
		return normalized, ""
	case CardsActionGet, CardsActionArchive:
		if rejection := requirePositiveID(normalized.CardID, "card_id"); rejection != "" {
			return normalizedCardsTaskRequest{}, rejection
		}
		return normalized, ""
	default:
		return normalizedCardsTaskRequest{}, fmt.Sprintf("unsupported cards task action %q", action)
	}
}

func rejectedCards(reason string) CardsTaskResult {
	return CardsTaskResult{BaseResult: rejectedBase(reason)}
}
