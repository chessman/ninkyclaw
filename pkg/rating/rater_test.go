package rating

import (
	"slices"
	"testing"

	"ninkyclaw/pkg/model"
)

func TestKeywordRater(t *testing.T) {
	rules := map[string]Rating{
		"jazz":      Medium,
		"classical": High,
		"rock":      VeryHigh,
		"pop":       Low,
		"bach":      High,
	}
	rater := NewKeywordRater(rules)

	tests := []struct {
		name             string
		concert          model.Concert
		expectedRating   Rating
		expectedKeywords []string
	}{
		{
			name: "No matches at all",
			concert: model.Concert{
				Description:         "An ambient electronic music performance.",
				ExtendedDescription: "Featuring synthesisers.",
			},
			expectedRating:   Low,
			expectedKeywords: nil,
		},
		{
			name: "Keyword only in the title",
			concert: model.Concert{
				Title:       "6 Bachi motetti II - FLORIDANTE",
				Description: "",
			},
			expectedRating:   High,
			expectedKeywords: []string{"bach"},
		},
		{
			name: "Single low priority match",
			concert: model.Concert{
				Description: "A simple pop concert.",
			},
			expectedRating:   Low,
			expectedKeywords: []string{"pop"},
		},
		{
			name: "Case insensitive matching",
			concert: model.Concert{
				Description: "Great JAZZ tonight.",
			},
			expectedRating:   Medium,
			expectedKeywords: []string{"jazz"},
		},
		{
			name: "Multiple matches - should return highest rating",
			concert: model.Concert{
				Description:         "A classical piano concerto.",
				ExtendedDescription: "Followed by a small jazz improvisation session.",
			},
			expectedRating:   High, // classical is High, jazz is Medium, High is greater
			expectedKeywords: []string{"classical", "jazz"},
		},
		{
			name: "Match in extended description only",
			concert: model.Concert{
				Description:         "An evening of music.",
				ExtendedDescription: "Where we will play experimental rock tunes.",
			},
			expectedRating:   VeryHigh,
			expectedKeywords: []string{"rock"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, matched, err := rater.Rate(tt.concert)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if r != tt.expectedRating {
				t.Errorf("expected rating %q, got %q", tt.expectedRating, r)
			}
			if !slices.Equal(matched, tt.expectedKeywords) {
				t.Errorf("expected keywords %v, got %v", tt.expectedKeywords, matched)
			}
		})
	}
}
