package entity

import "time"

// OptionalUTCTime returns a UTC copy of value, or nil when value is zero.
func OptionalUTCTime(value time.Time) *time.Time {
	return OptionalUTCTimePtr(&value)
}

// OptionalUTCTimePtr returns a UTC copy of value, or nil when value is nil or zero.
func OptionalUTCTimePtr(value *time.Time) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}

	utc := value.UTC()
	return &utc
}

// UTCTimeOrZero returns value normalized to UTC, or the zero time when value is nil.
func UTCTimeOrZero(value *time.Time) time.Time {
	if value == nil || value.IsZero() {
		return time.Time{}
	}

	return value.UTC()
}
