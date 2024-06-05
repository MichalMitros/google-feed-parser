package parser

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUnitSystemClockTimestamp(t *testing.T) {
	assert.InDelta(
		t,
		time.Now().UTC().UnixMilli(),
		systemClock{}.Timestamp(),
		float64(50*time.Millisecond),
		"should return current timestamp",
	)
}

func TestUnitSystemClockNow(t *testing.T) {
	assert.InDelta(
		t,
		time.Now().UTC().UnixMilli(),
		systemClock{}.Now().UnixMilli(),
		float64(50*time.Millisecond),
		"should return current timestamp",
	)
}
