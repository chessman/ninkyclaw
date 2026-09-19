package rating

import (
	"encoding/csv"
	"os"
	"sort"
	"strings"

	"ninkyclaw/pkg/model"
)

// Rating represents a qualitative rating for a concert.
type Rating string

const (
	Low      Rating = "Low"
	Medium   Rating = "Medium"
	High     Rating = "High"
	VeryHigh Rating = "Very High"
)

// String implements the fmt.Stringer interface.
func (r Rating) String() string {
	return string(r)
}

// parseRating converts a string into a Rating, returning Low if unrecognized.
func parseRating(s string) Rating {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "medium":
		return Medium
	case "high":
		return High
	case "very high", "veryhigh":
		return VeryHigh
	default:
		return Low
	}
}

// Rater defines a common interface for evaluating and rating concerts.
// Rate returns the rating plus the keywords that matched, sorted.
type Rater interface {
	Rate(c model.Concert) (Rating, []string, error)
}

// priorityMap maps rating enum to numeric priority for comparison.
var priorityMap = map[Rating]int{
	Low:      1,
	Medium:   2,
	High:     3,
	VeryHigh: 4,
}

// Priority returns the integer priority of a rating string. Unrecognized defaults to 1 (Low).
func Priority(s string) int {
	return priorityMap[parseRating(s)]
}

// KeywordRater rates concerts by matching keywords case-insensitively in descriptions.
type KeywordRater struct {
	keywords map[string]Rating
}

// NewKeywordRater creates a new KeywordRater with a set of keyword-to-rating rules.
func NewKeywordRater(rules map[string]Rating) *KeywordRater {
	return &KeywordRater{
		keywords: rules,
	}
}

// NewKeywordRaterFromCSVFile parses rules from a CSV file and returns a KeywordRater.
func NewKeywordRaterFromCSVFile(filePath string) (*KeywordRater, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	rules := make(map[string]Rating)
	for _, record := range records {
		if len(record) < 2 {
			continue
		}
		keyword := strings.TrimSpace(record[0])
		rateStr := strings.TrimSpace(record[1])
		if keyword == "" {
			continue
		}
		rules[keyword] = parseRating(rateStr)
	}

	return NewKeywordRater(rules), nil
}

// Rate evaluates the concert's description and extended description for keywords.
// Returns the highest rating matched (Low if none) and every keyword that
// matched, sorted so output is stable across runs.
func (kr *KeywordRater) Rate(c model.Concert) (Rating, []string, error) {
	descLower := strings.ToLower(c.Description)
	extDescLower := strings.ToLower(c.ExtendedDescription)

	highestRating := Low
	highestPriority := priorityMap[Low]
	var matched []string

	for kw, rate := range kr.keywords {
		kwLower := strings.ToLower(kw)
		if strings.Contains(descLower, kwLower) || strings.Contains(extDescLower, kwLower) {
			matched = append(matched, kw)
			prio := priorityMap[rate]
			if prio > highestPriority {
				highestPriority = prio
				highestRating = rate
			}
		}
	}

	sort.Strings(matched)

	return highestRating, matched, nil
}
