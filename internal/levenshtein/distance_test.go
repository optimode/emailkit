package levenshtein_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/optimode/emailkit/internal/levenshtein"
)

func TestDistance(t *testing.T) {
	tests := []struct {
		s, t string
		want int
	}{
		{"", "", 0},
		{"a", "", 1},
		{"", "a", 1},
		{"gmail.com", "gmail.com", 0},
		{"gmial.com", "gmail.com", 2},   // two swaps
		{"gmal.com", "gmail.com", 1},    // one missing letter
		{"gmailll.com", "gmail.com", 2}, // two extra letters
		{"yahoo.com", "gmail.com", 5},   // completely different
	}
	for _, tt := range tests {
		t.Run(tt.s+"->"+tt.t, func(t *testing.T) {
			assert.Equal(t, tt.want, levenshtein.Distance(tt.s, tt.t))
		})
	}
}
