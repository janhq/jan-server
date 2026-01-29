package search

import (
	"sort"

	"jan-server/mono/apps/backend/internal/domain/memory"
)

// RankedResult represents a search result with a combined score
type RankedResult struct {
	Item  interface{}
	Score float32
	Type  string // "user_memory", "project_fact", "episodic"
}

// Ranker handles result ranking and fusion
type Ranker struct {
	denseWeight   float32
	sparseWeight  float32
	lexicalWeight float32
}

// NewRanker creates a new ranker with default weights
func NewRanker() *Ranker {
	return &Ranker{
		denseWeight:   0.7,
		sparseWeight:  0.2,
		lexicalWeight: 0.1,
	}
}

// RankUserMemory ranks user memory items by weighted score
func (r *Ranker) RankUserMemory(items []memory.UserMemoryItem) []RankedResult {
	results := make([]RankedResult, len(items))

	for i, item := range items {
		// Score = similarity * (importance_score / 5.0)
		score := item.Similarity * (float32(item.Score) / 5.0)

		results[i] = RankedResult{
			Item:  item,
			Score: score,
			Type:  "user_memory",
		}
	}

	return results
}

// RankProjectFacts ranks project facts by weighted score
func (r *Ranker) RankProjectFacts(facts []memory.ProjectFact) []RankedResult {
	results := make([]RankedResult, len(facts))

	for i, fact := range facts {
		// Score = similarity * confidence
		score := fact.Similarity * fact.Confidence

		results[i] = RankedResult{
			Item:  fact,
			Score: score,
			Type:  "project_fact",
		}
	}

	return results
}

// RankEpisodicEvents ranks episodic events by weighted score
func (r *Ranker) RankEpisodicEvents(events []memory.EpisodicEvent) []RankedResult {
	results := make([]RankedResult, len(events))

	for i, event := range events {
		// Score = similarity * 0.8 (slightly lower weight for episodic)
		score := event.Similarity * 0.8

		results[i] = RankedResult{
			Item:  event,
			Score: score,
			Type:  "episodic",
		}
	}

	return results
}

// CombineAndRank combines results from multiple sources and ranks them
func (r *Ranker) CombineAndRank(
	userMemory []memory.UserMemoryItem,
	projectFacts []memory.ProjectFact,
	episodicEvents []memory.EpisodicEvent,
) []RankedResult {
	var allResults []RankedResult

	// Add user memory results
	allResults = append(allResults, r.RankUserMemory(userMemory)...)

	// Add project facts
	allResults = append(allResults, r.RankProjectFacts(projectFacts)...)

	// Add episodic events
	allResults = append(allResults, r.RankEpisodicEvents(episodicEvents)...)

	// Sort by score descending
	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].Score > allResults[j].Score
	})

	return allResults
}

// GetTopK returns the top K results
func (r *Ranker) GetTopK(results []RankedResult, k int) []RankedResult {
	if k > len(results) {
		k = len(results)
	}
	return results[:k]
}

// SeparateByType separates ranked results back into their original types
func (r *Ranker) SeparateByType(results []RankedResult) (
	userMemory []memory.UserMemoryItem,
	projectFacts []memory.ProjectFact,
	episodicEvents []memory.EpisodicEvent,
) {
	for _, result := range results {
		switch result.Type {
		case "user_memory":
			if item, ok := result.Item.(memory.UserMemoryItem); ok {
				userMemory = append(userMemory, item)
			}
		case "project_fact":
			if fact, ok := result.Item.(memory.ProjectFact); ok {
				projectFacts = append(projectFacts, fact)
			}
		case "episodic":
			if event, ok := result.Item.(memory.EpisodicEvent); ok {
				episodicEvents = append(episodicEvents, event)
			}
		}
	}

	return
}
