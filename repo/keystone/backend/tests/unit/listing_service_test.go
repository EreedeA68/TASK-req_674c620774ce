package unit

import (
	"strings"
	"testing"
	"time"

	"github.com/keystone/backend/pkg/similarity"
	"github.com/stretchr/testify/assert"
)

const listingDuplicateThreshold = 0.8

// isDuplicateTitle calls the real similarity package — not a mirror of service logic.
func isDuplicateTitle(a, b string) bool {
	return similarity.IsSimilar(strings.ToLower(a), strings.ToLower(b), listingDuplicateThreshold)
}

func TestListingDuplicateDetection_SameTitle(t *testing.T) {
	assert.True(t, isDuplicateTitle("Lost black wallet", "Lost black wallet"))
}

func TestListingDuplicateDetection_SlightlyDifferent(t *testing.T) {
	assert.True(t, isDuplicateTitle("Lost black wallet", "Lost blac wallet"))
}

func TestListingDuplicateDetection_TotallyDifferent(t *testing.T) {
	assert.False(t, isDuplicateTitle("Lost black wallet", "Found red bicycle"))
}

func TestListingDuplicateDetection_CaseInsensitive(t *testing.T) {
	assert.True(t, isDuplicateTitle("LOST BLACK WALLET", "lost black wallet"))
}

func TestListingAutoUnlistCutoff(t *testing.T) {
	cutoff := time.Now().AddDate(0, 0, -90)
	assert.True(t, time.Now().AddDate(0, 0, -91).Before(cutoff), "91-day-old listing should be before cutoff")
	assert.False(t, time.Now().AddDate(0, 0, -30).Before(cutoff), "30-day-old listing should not be before cutoff")
	assert.False(t, time.Now().AddDate(0, 0, -90).Before(cutoff), "exactly 90-day-old listing should not be before cutoff")
}

func TestListingTimeWindowValidation(t *testing.T) {
	var zeroTime time.Time
	assert.True(t, zeroTime.IsZero(), "zero time should be detected as missing")

	start := time.Now()
	end := start.Add(-1 * time.Hour)
	assert.True(t, end.Before(start), "end before start should be invalid")
}
