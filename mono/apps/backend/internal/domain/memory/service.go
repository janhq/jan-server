package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"jan-server/mono/apps/backend/internal/domain/embedding"
	"github.com/rs/zerolog/log"
)

// Service handles memory operations
type Service struct {
	repo            Repository
	embeddingClient embedding.Client
}

// NewService creates a new memory service
func NewService(repo Repository, embeddingClient embedding.Client) *Service {
	return &Service{
		repo:            repo,
		embeddingClient: embeddingClient,
	}
}

// Load retrieves relevant memories for a given query
func (s *Service) Load(ctx context.Context, req MemoryLoadRequest) (*MemoryLoadResponse, error) {
	// Set defaults
	if req.Options.MaxUserItems == 0 {
		req.Options.MaxUserItems = 20
	}
	if req.Options.MaxProjectItems == 0 {
		req.Options.MaxProjectItems = 20
	}
	if req.Options.MaxEpisodicItems == 0 {
		req.Options.MaxEpisodicItems = 20
	}
	if req.Options.MinSimilarity == 0 {
		req.Options.MinSimilarity = 0.5
	}

	// Embed the query
	queryEmbedding, err := s.embeddingClient.EmbedSingle(ctx, req.Query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	log.Debug().
		Str("user_id", req.UserID).
		Str("query", req.Query).
		Int("embedding_dim", len(queryEmbedding)).
		Msg("Query embedded successfully")

	// Search user memory
	userMemory, err := s.repo.SearchUserMemory(
		ctx,
		req.UserID,
		queryEmbedding,
		req.Options.MaxUserItems,
		req.Options.MinSimilarity,
	)
	if err != nil {
		return nil, fmt.Errorf("search user memory: %w", err)
	}

	if len(userMemory) == 0 {
		allUserMemory, err := s.repo.GetUserMemoryItems(ctx, req.UserID)
		if err == nil && len(allUserMemory) > 0 {
			if req.Options.MaxUserItems > 0 && len(allUserMemory) > req.Options.MaxUserItems {
				allUserMemory = allUserMemory[:req.Options.MaxUserItems]
			}
			userMemory = allUserMemory
		}
	}

	// Search project facts if project_id provided
	var projectFacts []ProjectFact
	if req.ProjectID != "" {
		projectFacts, err = s.repo.SearchProjectFacts(
			ctx,
			req.ProjectID,
			queryEmbedding,
			req.Options.MaxProjectItems,
			req.Options.MinSimilarity,
		)
		if err != nil {
			return nil, fmt.Errorf("search project facts: %w", err)
		}

		if len(projectFacts) == 0 {
			allFacts, err := s.repo.GetProjectFacts(ctx, req.ProjectID)
			if err == nil && len(allFacts) > 0 {
				if req.Options.MaxProjectItems > 0 && len(allFacts) > req.Options.MaxProjectItems {
					allFacts = allFacts[:req.Options.MaxProjectItems]
				}
				projectFacts = allFacts
			}
		}
	}

	// Search episodic events
	episodicEvents, err := s.repo.SearchEpisodicEvents(
		ctx,
		req.UserID,
		queryEmbedding,
		req.Options.MaxEpisodicItems,
		req.Options.MinSimilarity,
	)
	if err != nil {
		return nil, fmt.Errorf("search episodic events: %w", err)
	}

	if len(episodicEvents) == 0 {
		allEvents, err := s.repo.GetEpisodicEvents(ctx, req.UserID, req.Options.MaxEpisodicItems)
		if err == nil && len(allEvents) > 0 {
			episodicEvents = allEvents
		}
	}

	for i := range userMemory {
		if userMemory[i].Similarity == 0 {
			userMemory[i].Similarity = 1.0
		}
	}
	for i := range projectFacts {
		if projectFacts[i].Similarity == 0 {
			projectFacts[i].Similarity = projectFacts[i].Confidence
			if projectFacts[i].Similarity == 0 {
				projectFacts[i].Similarity = 1.0
			}
		}
	}
	for i := range episodicEvents {
		if episodicEvents[i].Similarity == 0 {
			episodicEvents[i].Similarity = 0.8
		}
	}

	log.Info().
		Int("user_memory_count", len(userMemory)).
		Int("project_facts_count", len(projectFacts)).
		Int("episodic_events_count", len(episodicEvents)).
		Msg("Memory search completed")

	if userMemory == nil {
		userMemory = []UserMemoryItem{}
	}
	if projectFacts == nil {
		projectFacts = []ProjectFact{}
	}
	if episodicEvents == nil {
		episodicEvents = []EpisodicEvent{}
	}

	return &MemoryLoadResponse{
		CoreMemory:     userMemory,
		SemanticMemory: projectFacts,
		EpisodicMemory: episodicEvents,
	}, nil
}

// Observe stores conversation and extracts memories
func (s *Service) Observe(ctx context.Context, req MemoryObserveRequest) error {
	// Store conversation items
	for _, msg := range req.Messages {
		msg.ConversationID = req.ConversationID
		if err := s.repo.CreateConversationItem(ctx, &msg); err != nil {
			log.Error().Err(err).Msg("Failed to store conversation item")
			// Continue processing even if storage fails
		}
	}

	// Extract memory actions from conversation
	memoryAction, err := s.extractMemoryActions(ctx, req)
	if err != nil {
		return fmt.Errorf("extract memory actions: %w", err)
	}

	// Process additions
	if err := s.processMemoryAdditions(ctx, req, memoryAction.Add); err != nil {
		return fmt.Errorf("process memory additions: %w", err)
	}

	// Process deletions
	for _, itemID := range memoryAction.Delete {
		// Try to delete from all tables (soft delete)
		s.repo.DeleteUserMemoryItem(ctx, itemID)
		s.repo.DeleteProjectFact(ctx, itemID)
		// Note: We don't delete episodic events as they're historical
	}

	log.Info().
		Int("user_memory_added", len(memoryAction.Add.UserMemory)).
		Int("project_facts_added", len(memoryAction.Add.ProjectMemory)).
		Int("episodic_added", len(memoryAction.Add.Episodic)).
		Int("deleted", len(memoryAction.Delete)).
		Msg("Memory observation completed")

	return nil
}

// extractMemoryActions analyzes conversation and determines what to remember
func (s *Service) extractMemoryActions(ctx context.Context, req MemoryObserveRequest) (*MemoryAction, error) {
	// For now, use a simple heuristic-based approach
	// In production, this would call an LLM to analyze the conversation

	action := &MemoryAction{
		Add: MemoryAddActions{
			UserMemory:    []UserMemoryItemInput{},
			ProjectMemory: []ProjectFactInput{},
			Episodic:      []EpisodicEventInput{},
		},
		Delete: []string{},
	}

	// Extract user preferences and context from messages
	for _, msg := range req.Messages {
		if msg.Role == "user" {
			// Simple pattern matching for demonstration
			content := strings.ToLower(msg.Content)

			// Detect preferences
			if strings.Contains(content, "i prefer") || strings.Contains(content, "i like") {
				action.Add.UserMemory = append(action.Add.UserMemory, UserMemoryItemInput{
					Scope:      "preference",
					Key:        "user_preference",
					Text:       msg.Content,
					Importance: "medium",
				})
			}

			// Detect project decisions
			if strings.Contains(content, "we should") || strings.Contains(content, "let's use") {
				action.Add.ProjectMemory = append(action.Add.ProjectMemory, ProjectFactInput{
					Kind:       "decision",
					Title:      "Project decision",
					Text:       msg.Content,
					Confidence: 0.8,
				})
			}
		}

		// Always create episodic event for the interaction
		action.Add.Episodic = append(action.Add.Episodic, EpisodicEventInput{
			Text: fmt.Sprintf("%s: %s", msg.Role, msg.Content),
			Kind: "interaction",
		})
	}

	return action, nil
}

// processMemoryAdditions processes and stores new memory items
func (s *Service) processMemoryAdditions(ctx context.Context, req MemoryObserveRequest, additions MemoryAddActions) error {
	// Collect all texts to embed
	var textsToEmbed []string
	var textTypes []string // Track what type each text is

	for _, item := range additions.UserMemory {
		textsToEmbed = append(textsToEmbed, item.Text)
		textTypes = append(textTypes, "user_memory")
	}

	for _, fact := range additions.ProjectMemory {
		textsToEmbed = append(textsToEmbed, fact.Text)
		textTypes = append(textTypes, "project_fact")
	}

	for _, event := range additions.Episodic {
		textsToEmbed = append(textsToEmbed, event.Text)
		textTypes = append(textTypes, "episodic")
	}

	if len(textsToEmbed) == 0 {
		return nil
	}

	// Batch embed all texts
	embeddings, err := s.embeddingClient.Embed(ctx, textsToEmbed)
	if err != nil {
		return fmt.Errorf("batch embed: %w", err)
	}

	log.Debug().
		Int("texts_embedded", len(textsToEmbed)).
		Msg("Batch embedding completed")

	// Process embeddings and store
	embeddingIndex := 0

	// Store user memory items
	for _, item := range additions.UserMemory {
		userItem := &UserMemoryItem{
			UserID:    req.UserID,
			Scope:     item.Scope,
			Key:       item.Key,
			Text:      item.Text,
			Score:     importanceToScore(item.Importance),
			Embedding: embeddings[embeddingIndex],
		}
		embeddingIndex++

		if _, err := s.repo.UpsertUserMemoryItem(ctx, userItem); err != nil {
			log.Error().Err(err).Msg("Failed to store user memory item")
		}
	}

	// Store project facts
	for _, fact := range additions.ProjectMemory {
		projectFact := &ProjectFact{
			ProjectID:            req.ProjectID,
			Kind:                 fact.Kind,
			Title:                fact.Title,
			Text:                 fact.Text,
			Confidence:           fact.Confidence,
			Embedding:            embeddings[embeddingIndex],
			SourceConversationID: req.ConversationID,
		}
		embeddingIndex++

		if _, err := s.repo.UpsertProjectFact(ctx, projectFact); err != nil {
			log.Error().Err(err).Msg("Failed to store project fact")
		}
	}

	// Store episodic events
	for _, event := range additions.Episodic {
		episodicEvent := &EpisodicEvent{
			UserID:         req.UserID,
			ProjectID:      req.ProjectID,
			ConversationID: req.ConversationID,
			Time:           req.Messages[len(req.Messages)-1].CreatedAt,
			Text:           event.Text,
			Kind:           event.Kind,
			Embedding:      embeddings[embeddingIndex],
		}
		embeddingIndex++

		if err := s.repo.CreateEpisodicEvent(ctx, episodicEvent); err != nil {
			log.Error().Err(err).Msg("Failed to store episodic event")
		}
	}

	return nil
}

// Helper function to convert importance string to score
func importanceToScore(importance string) int {
	switch strings.ToLower(importance) {
	case "critical":
		return 5
	case "high":
		return 4
	case "medium":
		return 3
	case "low":
		return 2
	default:
		return 3
	}
}

// RankResults combines and ranks results from different memory types
func (s *Service) RankResults(userMemory []UserMemoryItem, projectFacts []ProjectFact, episodic []EpisodicEvent) []interface{} {
	type rankedItem struct {
		item  interface{}
		score float32
	}

	var items []rankedItem

	// Add user memory with weighted score
	for _, item := range userMemory {
		score := item.Similarity * float32(item.Score) / 5.0
		items = append(items, rankedItem{item: item, score: score})
	}

	// Add project facts with weighted score
	for _, fact := range projectFacts {
		score := fact.Similarity * fact.Confidence
		items = append(items, rankedItem{item: fact, score: score})
	}

	// Add episodic events
	for _, event := range episodic {
		score := event.Similarity * 0.8 // Slightly lower weight for episodic
		items = append(items, rankedItem{item: event, score: score})
	}

	// Sort by score descending
	sort.Slice(items, func(i, j int) bool {
		return items[i].score > items[j].score
	})

	// Extract items
	result := make([]interface{}, len(items))
	for i, item := range items {
		result[i] = item.item
	}

	return result
}

// Helper to format memory for LLM context
func (s *Service) FormatMemoryForContext(resp *MemoryLoadResponse) string {
	var builder strings.Builder

	if len(resp.CoreMemory) > 0 {
		builder.WriteString("## Core Memory (User Preferences & Context)\n\n")
		for _, item := range resp.CoreMemory {
			builder.WriteString(fmt.Sprintf("- [%s] %s (importance: %d/5, similarity: %.2f)\n",
				item.Scope, item.Text, item.Score, item.Similarity))
		}
		builder.WriteString("\n")
	}

	if len(resp.SemanticMemory) > 0 {
		builder.WriteString("## Semantic Memory (Project Facts & Decisions)\n\n")
		for _, fact := range resp.SemanticMemory {
			builder.WriteString(fmt.Sprintf("- [%s] %s: %s (confidence: %.2f, similarity: %.2f)\n",
				fact.Kind, fact.Title, fact.Text, fact.Confidence, fact.Similarity))
		}
		builder.WriteString("\n")
	}

	if len(resp.EpisodicMemory) > 0 {
		builder.WriteString("## Episodic Memory (Recent Interactions)\n\n")
		for _, event := range resp.EpisodicMemory {
			builder.WriteString(fmt.Sprintf("- [%s] %s: %s (similarity: %.2f)\n",
				event.Time.Format("2006-01-02 15:04"), event.Kind, event.Text, event.Similarity))
		}
		builder.WriteString("\n")
	}

	return builder.String()
}

// GetMemoryStats returns statistics about stored memories
func (s *Service) GetMemoryStats(ctx context.Context, userID, projectID string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	userMemory, err := s.repo.GetUserMemoryItems(ctx, userID)
	if err == nil {
		stats["user_memory_count"] = len(userMemory)
	}

	if projectID != "" {
		projectFacts, err := s.repo.GetProjectFacts(ctx, projectID)
		if err == nil {
			stats["project_facts_count"] = len(projectFacts)
		}
	}

	episodic, err := s.repo.GetEpisodicEvents(ctx, userID, 100)
	if err == nil {
		stats["episodic_events_count"] = len(episodic)
	}

	return stats, nil
}

// ExportMemory exports all memory for a user (for data portability)
func (s *Service) ExportMemory(ctx context.Context, userID string) (string, error) {
	export := make(map[string]interface{})

	userMemory, err := s.repo.GetUserMemoryItems(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("get user memory: %w", err)
	}
	export["user_memory"] = userMemory

	episodic, err := s.repo.GetEpisodicEvents(ctx, userID, 1000)
	if err != nil {
		return "", fmt.Errorf("get episodic events: %w", err)
	}
	export["episodic_events"] = episodic

	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal export: %w", err)
	}

	return string(data), nil
}

// UpsertUserMemories upserts user memory items (for LLM tools)
func (s *Service) UpsertUserMemories(ctx context.Context, req UserMemoryUpsertRequest) ([]string, error) {
	ids := make([]string, 0, len(req.Items))

	// Collect all texts for batch embedding
	texts := make([]string, len(req.Items))
	for i, item := range req.Items {
		texts[i] = item.Text
	}

	// Batch embed all texts
	embeddings, err := s.embeddingClient.Embed(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("embed texts: %w", err)
	}

	log.Info().
		Str("user_id", req.UserID).
		Int("item_count", len(req.Items)).
		Msg("Upserting user memories")

	// Upsert each item
	for i, item := range req.Items {
		userItem := &UserMemoryItem{
			UserID:    req.UserID,
			Scope:     item.Scope,
			Key:       item.Key,
			Text:      item.Text,
			Score:     importanceToScore(item.Importance),
			Embedding: embeddings[i],
		}

		id, err := s.repo.UpsertUserMemoryItem(ctx, userItem)
		if err != nil {
			log.Error().Err(err).Str("text", item.Text).Msg("Failed to upsert user memory item")
			continue
		}

		ids = append(ids, id)
	}

	log.Info().
		Str("user_id", req.UserID).
		Int("upserted_count", len(ids)).
		Msg("User memories upserted successfully")

	return ids, nil
}

// UpsertProjectFacts upserts project facts (for LLM tools)
func (s *Service) UpsertProjectFacts(ctx context.Context, req ProjectFactUpsertRequest) ([]string, error) {
	ids := make([]string, 0, len(req.Facts))

	// Collect all texts for batch embedding
	texts := make([]string, len(req.Facts))
	for i, fact := range req.Facts {
		texts[i] = fact.Text
	}

	// Batch embed all texts
	embeddings, err := s.embeddingClient.Embed(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("embed texts: %w", err)
	}

	log.Info().
		Str("project_id", req.ProjectID).
		Int("fact_count", len(req.Facts)).
		Msg("Upserting project facts")

	// Upsert each fact
	for i, fact := range req.Facts {
		projectFact := &ProjectFact{
			ProjectID:  req.ProjectID,
			Kind:       fact.Kind,
			Title:      fact.Title,
			Text:       fact.Text,
			Confidence: fact.Confidence,
			Embedding:  embeddings[i],
		}

		id, err := s.repo.UpsertProjectFact(ctx, projectFact)
		if err != nil {
			log.Error().Err(err).Str("title", fact.Title).Msg("Failed to upsert project fact")
			continue
		}

		ids = append(ids, id)
	}

	log.Info().
		Str("project_id", req.ProjectID).
		Int("upserted_count", len(ids)).
		Msg("Project facts upserted successfully")

	return ids, nil
}

// DeleteMemories soft deletes memories by IDs (for LLM tools)
func (s *Service) DeleteMemories(ctx context.Context, req DeleteRequest) (int, error) {
	deletedCount := 0

	log.Info().
		Int("id_count", len(req.IDs)).
		Msg("Deleting memories")

	for _, id := range req.IDs {
		// Try deleting from user memory
		if err := s.repo.DeleteUserMemoryItem(ctx, id); err == nil {
			deletedCount++
			continue
		}

		// Try deleting from project facts
		if err := s.repo.DeleteProjectFact(ctx, id); err == nil {
			deletedCount++
			continue
		}

		// Try deleting from episodic events
		if err := s.repo.DeleteEpisodicEvent(ctx, id); err == nil {
			deletedCount++
			continue
		}

		log.Warn().Str("id", id).Msg("Memory ID not found in any table")
	}

	log.Info().
		Int("deleted_count", deletedCount).
		Int("requested_count", len(req.IDs)).
		Msg("Memories deleted")

	return deletedCount, nil
}
