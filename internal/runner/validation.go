package runner

import (
	"fmt"
	"strings"
	"time"
)

const (
	defaultLimit = 10
	maxLimit     = 100
)

func normalizeLimit(limit int) (int, string) {
	if limit < 0 {
		return 0, "limit must be greater than or equal to 0"
	}
	if limit == 0 {
		return defaultLimit, ""
	}
	if limit > maxLimit {
		return 0, fmt.Sprintf("limit must be less than or equal to %d", maxLimit)
	}
	return limit, ""
}

func optionalRFC3339(value string, field string) (*time.Time, string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, ""
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return nil, field + " must be an RFC3339 timestamp"
	}
	parsed = parsed.UTC()
	return &parsed, ""
}

func requirePositiveID(id int64, field string) string {
	if id <= 0 {
		return field + " is required"
	}
	return ""
}

func trimOptional(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}
