// Package moderation provides a wrapper for the OpenAI Moderation API.
package moderation

// Service defines methods to operate on moderation API.
type Service interface {
	NewModerationBuilder() Builder
}

// Builder is a builder for the Moderation API.
type Builder interface {
	AddImage(url string) Builder
	AddText(text string) Builder
	Clear() Builder
	Execute() ([]*Result, error)
	SetMinConfidence(minPercent int) Builder
}

// Result is a single result of the Moderation API, includes the input,
// whether it was flagged and moderation categories with their scores.
type Result struct {
	Flagged                   bool                `json:"flagged"`
	Categories                map[string]bool     `json:"categories"`
	CategoryScores            map[string]float64  `json:"category_scores"`
	CategoryAppliedInputTypes map[string][]string `json:"category_applied_input_types"`

	// filled by us for convenience
	Input string `json:"-"`
}

// WithConfidence checks if the result passes the confidence threshold.
// If not, it sets the result as not flagged.
// Categories with low confidence are removed from the result.
func (r *Result) WithConfidence(minPercent int) {
	flaggedWithConfidence := false
	for category := range r.CategoryScores {
		if int(r.CategoryScores[category]*100) >= minPercent {
			flaggedWithConfidence = true
		} else {
			delete(r.CategoryScores, category)
			delete(r.Categories, category)
			delete(r.CategoryAppliedInputTypes, category)
		}
	}
	if !flaggedWithConfidence {
		r.Flagged = false
	}
}
