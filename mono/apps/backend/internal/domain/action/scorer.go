package action

import (
	"math"
	"strings"

	"jan-server/mono/apps/backend/internal/domain/memory"
)

// Scorer handles importance scoring for memory items
type Scorer struct{}

// NewScorer creates a new scorer
func NewScorer() *Scorer {
	return &Scorer{}
}

// ScoreImportance converts importance string to numeric score
func (s *Scorer) ScoreImportance(importance string) int {
	switch strings.ToLower(importance) {
	case "critical":
		return 5
	case "high":
		return 4
	case "medium":
		return 3
	case "low":
		return 2
	case "minimal":
		return 1
	default:
		return 3 // Default to medium
	}
}

// ScoreUserMemoryItem scores a user memory item based on various factors
func (s *Scorer) ScoreUserMemoryItem(item *memory.UserMemoryItemInput) int {
	score := s.ScoreImportance(item.Importance)

	// Adjust based on scope
	switch item.Scope {
	case "core":
		if score < 5 {
			score++ // Core facts are more important
		}
	case "preference":
		// Keep as is
	case "context":
		if score > 1 {
			score-- // Context is less permanent
		}
	}

	return score
}

// ScoreProjectFact scores a project fact based on confidence and kind
func (s *Scorer) ScoreProjectFact(fact *memory.ProjectFactInput) float32 {
	confidence := fact.Confidence

	// Adjust based on kind
	switch fact.Kind {
	case "decision":
		confidence = float32(math.Min(float64(confidence+0.1), 1.0)) // Decisions are more important
	case "requirement":
		confidence = float32(math.Min(float64(confidence+0.05), 1.0))
	case "constraint":
		// Keep as is
	case "context":
		confidence = float32(math.Max(float64(confidence-0.1), 0.0))
	}

	return confidence
}

// AnalyzeTextImportance analyzes text to determine importance
func (s *Scorer) AnalyzeTextImportance(text string) string {
	text = strings.ToLower(text)

	// Critical indicators
	criticalKeywords := []string{
		"must", "required", "critical", "essential", "mandatory",
		"always", "never", "security", "password", "api key",
	}
	for _, keyword := range criticalKeywords {
		if strings.Contains(text, keyword) {
			return "critical"
		}
	}

	// High importance indicators
	highKeywords := []string{
		"important", "should", "prefer", "recommend",
		"decision", "requirement", "constraint",
	}
	for _, keyword := range highKeywords {
		if strings.Contains(text, keyword) {
			return "high"
		}
	}

	// Low importance indicators
	lowKeywords := []string{
		"maybe", "might", "consider", "optional",
		"nice to have", "if possible",
	}
	for _, keyword := range lowKeywords {
		if strings.Contains(text, keyword) {
			return "low"
		}
	}

	// Default to medium
	return "medium"
}
