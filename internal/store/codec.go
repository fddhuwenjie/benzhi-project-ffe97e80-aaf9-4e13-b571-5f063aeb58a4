package store

import (
	"database/sql"
	"encoding/json"
	"time"
)

func encode(value any) (string, error) {
	data, err := json.Marshal(value)
	return string(data), err
}

func decode(raw string, target any) error { return json.Unmarshal([]byte(raw), target) }

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func parseTime(value string) (time.Time, error) { return time.Parse(time.RFC3339Nano, value) }

func parseNullableTime(value sql.NullString) (*time.Time, error) {
	if !value.Valid {
		return nil, nil
	}
	parsed, err := parseTime(value.String)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
