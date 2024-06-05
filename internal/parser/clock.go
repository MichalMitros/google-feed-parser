package parser

import "time"

type systemClock struct{}

// Timestamp return current UTC timestamp.
func (c systemClock) Timestamp() int64 {
	return time.Now().UTC().UnixMilli()
}

// Now return current time.
func (c systemClock) Now() *time.Time {
	t := time.Now().UTC()
	return &t
}
