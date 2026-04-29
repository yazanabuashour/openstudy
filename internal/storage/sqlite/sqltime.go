package sqlite

import (
	"database/sql"
	"time"
)

const fixedInstantLayout = "2006-01-02T15:04:05.000000000Z"

func serializeInstant(value time.Time) string {
	return value.UTC().Format(fixedInstantLayout)
}

func nullableInstant(value *time.Time) any {
	if value == nil {
		return nil
	}
	return serializeInstant(*value)
}

func parseInstant(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}

func parseNullableInstant(value sql.NullString) (*time.Time, error) {
	if !value.Valid {
		return nil, nil
	}
	parsed, err := parseInstant(value.String)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableStringPointer(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func nullableInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableIntPointer(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	converted := int(value.Int64)
	return &converted
}
