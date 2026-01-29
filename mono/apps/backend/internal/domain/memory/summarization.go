package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

// SummarizerConfig holds configuration for conversation summarization
type SummarizerConfig struct {
	LLMEndpoint     string
	Model           string
	TriggerEveryN   int           // Summarize every N messages
	TriggerInterval time.Duration // Or every X duration
	MaxWindowSize   int           // Max messages to include in summary
	Temperature     float32
	MaxTokens       int
}

// Summarizer handles conversation summarization
type Summarizer struct {
	config SummarizerConfig
	llm    LLMClient
}

// SummarizationResult represents the structured output from summarization
type SummarizationResult struct {
	DialogueSummary string   `json:"dialogue_summary"`
	OpenTasks       []string `json:"open_tasks"`
	Entities        []string `json:"entities"`
	Decisions       []string `json:"decisions"`
}

// NewSummarizer creates a new conversation summarizer
func NewSummarizer(config SummarizerConfig, llm LLMClient) *Summarizer {
	// Set defaults
	if config.TriggerEveryN == 0 {
		config.TriggerEveryN = 10
	}
	if config.TriggerInterval == 0 {
		config.TriggerInterval = 5 * time.Minute
	}
	if config.MaxWindowSize == 0 {
		config.MaxWindowSize = 50
	}
	if config.Temperature == 0 {
		config.Temperature = 0.3
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 1000
	}
	if config.Model == "" {
		config.Model = "gpt-4"
	}

	return &Summarizer{
		config: config,
		llm:    llm,
	}
}

// ShouldSummarize determines if summarization should be triggered
func (s *Summarizer) ShouldSummarize(messageCount int, lastSummarizedAt time.Time) bool {
	// Trigger by message count
	if messageCount >= s.config.TriggerEveryN {
		return true
	}

	// Trigger by time interval
	if time.Since(lastSummarizedAt) >= s.config.TriggerInterval {
		return true
	}

	return false
}

// Summarize generates a summary of the conversation
func (s *Summarizer) Summarize(ctx context.Context, messages []ConversationItem, previousSummary *ConversationSummary) (*SummarizationResult, error) {
	// Build the prompt
	prompt := s.buildSummarizationPrompt(messages, previousSummary)

	log.Debug().
		Int("message_count", len(messages)).
		Bool("has_previous_summary", previousSummary != nil).
		Msg("Generating conversation summary")

	// Call LLM
	response, err := s.llm.Complete(ctx, prompt, LLMOptions{
		Model:          s.config.Model,
		Temperature:    s.config.Temperature,
		MaxTokens:      s.config.MaxTokens,
		ResponseFormat: "json",
	})
	if err != nil {
		return nil, fmt.Errorf("llm completion failed: %w", err)
	}

	// Parse JSON response
	var result SummarizationResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to parse summarization result: %w", err)
	}

	log.Info().
		Str("summary", result.DialogueSummary).
		Int("open_tasks", len(result.OpenTasks)).
		Int("entities", len(result.Entities)).
		Int("decisions", len(result.Decisions)).
		Msg("Conversation summarized successfully")

	return &result, nil
}

// buildSummarizationPrompt constructs the LLM prompt for summarization
func (s *Summarizer) buildSummarizationPrompt(messages []ConversationItem, previousSummary *ConversationSummary) string {
	prompt := `You are analyzing a conversation to extract key information. Your task is to:
1. Provide a concise 2-3 sentence summary of the conversation
2. List any open tasks or action items mentioned
3. Identify people, systems, services, or tools mentioned
4. Note any decisions or conclusions reached

Be precise and factual. Only include information explicitly mentioned in the conversation.

`

	// Add previous summary if exists
	if previousSummary != nil && previousSummary.DialogueSummary != "" {
		prompt += fmt.Sprintf(`Previous Summary:
%s

Previous Open Tasks:
%s

Previous Entities:
%s

Previous Decisions:
%s

`, previousSummary.DialogueSummary,
			formatList(previousSummary.OpenTasks),
			formatList(previousSummary.Entities),
			formatList(previousSummary.Decisions))
	}

	// Add conversation window
	prompt += "Recent Conversation:\n"
	for _, msg := range messages {
		prompt += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
	}

	prompt += `
Return your analysis as JSON with this exact structure:
{
  "dialogue_summary": "2-3 sentence summary of the conversation",
  "open_tasks": ["task 1", "task 2"],
  "entities": ["entity 1", "entity 2"],
  "decisions": ["decision 1", "decision 2"]
}

Ensure the response is valid JSON.`

	return prompt
}

// formatList formats a JSON array for display
func formatList(items []interface{}) string {
	if len(items) == 0 {
		return "(none)"
	}

	result := ""
	for i, item := range items {
		if str, ok := item.(string); ok {
			result += fmt.Sprintf("- %s\n", str)
		} else {
			// Handle JSON objects
			bytes, _ := json.Marshal(item)
			result += fmt.Sprintf("- %s\n", string(bytes))
		}
		if i >= 9 { // Limit to 10 items
			result += "- ...\n"
			break
		}
	}
	return result
}

// MergeSummaries merges new summary with previous one
func (s *Summarizer) MergeSummaries(previous *ConversationSummary, new *SummarizationResult) *ConversationSummary {
	if previous == nil {
		return &ConversationSummary{
			DialogueSummary: new.DialogueSummary,
			OpenTasks:       stringSliceToInterface(new.OpenTasks),
			Entities:        stringSliceToInterface(new.Entities),
			Decisions:       stringSliceToInterface(new.Decisions),
			UpdatedAt:       time.Now(),
		}
	}

	// Merge entities and decisions (deduplicate)
	entities := mergeUnique(interfaceToStringSlice(previous.Entities), new.Entities)
	decisions := mergeUnique(interfaceToStringSlice(previous.Decisions), new.Decisions)

	// For open tasks, replace with new ones (old tasks are assumed completed)
	openTasks := new.OpenTasks

	return &ConversationSummary{
		DialogueSummary: new.DialogueSummary,
		OpenTasks:       stringSliceToInterface(openTasks),
		Entities:        stringSliceToInterface(entities),
		Decisions:       stringSliceToInterface(decisions),
		UpdatedAt:       time.Now(),
	}
}

// Helper functions

func stringSliceToInterface(strs []string) []interface{} {
	result := make([]interface{}, len(strs))
	for i, s := range strs {
		result[i] = s
	}
	return result
}

func interfaceToStringSlice(items []interface{}) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		if str, ok := item.(string); ok {
			result = append(result, str)
		}
	}
	return result
}

func mergeUnique(existing, new []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)

	// Add existing items
	for _, item := range existing {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	// Add new items
	for _, item := range new {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}
