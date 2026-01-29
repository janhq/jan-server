package action

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"jan-server/mono/apps/backend/internal/domain/memory"
	"github.com/rs/zerolog/log"
)

// Planner handles memory action planning using LLM
type Planner struct {
	scorer *Scorer
	llm    memory.LLMClient
	config PlannerConfig
}

// PlannerConfig holds configuration for the planner
type PlannerConfig struct {
	Model           string
	Temperature     float32
	MaxTokens       int
	UseHeuristics   bool // Fallback to heuristics if LLM fails
	IncludeContext  bool // Include existing memory in prompt
	DetectConflicts bool // Enable conflict detection
}

// LLMMemoryActionResponse represents the structured LLM response
type LLMMemoryActionResponse struct {
	Delete []string `json:"delete"`
	Add    struct {
		UserMemory    []memory.UserMemoryItemInput `json:"user_memory"`
		ProjectMemory []memory.ProjectFactInput    `json:"project_memory"`
		Episodic      []memory.EpisodicEventInput  `json:"episodic_memory"`
	} `json:"add"`
	Reasoning string `json:"reasoning,omitempty"`
}

// NewPlanner creates a new LLM-based memory action planner
func NewPlanner(llm memory.LLMClient, config PlannerConfig) *Planner {
	// Set defaults
	if config.Model == "" {
		config.Model = "gpt-4"
	}
	if config.Temperature == 0 {
		config.Temperature = 0.3
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 2000
	}

	return &Planner{
		scorer: NewScorer(),
		llm:    llm,
		config: config,
	}
}

// PlanActions analyzes conversation and determines memory actions using LLM
func (p *Planner) PlanActions(ctx context.Context, req memory.MemoryObserveRequest, existingMemory *ExistingMemoryContext) (*memory.MemoryAction, error) {
	// Try LLM-based planning first
	if p.llm != nil {
		action, err := p.planWithLLM(ctx, req, existingMemory)
		if err == nil {
			return action, nil
		}

		log.Warn().Err(err).Msg("LLM-based planning failed, falling back to heuristics")
	}

	// Fallback to heuristics if LLM fails or is disabled
	if p.config.UseHeuristics {
		return p.planWithHeuristics(ctx, req), nil
	}

	return nil, fmt.Errorf("LLM planning failed and heuristics disabled")
}

// planWithLLM uses LLM to analyze conversation and plan memory actions
func (p *Planner) planWithLLM(ctx context.Context, req memory.MemoryObserveRequest, existingMemory *ExistingMemoryContext) (*memory.MemoryAction, error) {
	// Build prompt
	prompt := p.buildMemoryActionPrompt(req, existingMemory)

	log.Debug().
		Int("message_count", len(req.Messages)).
		Bool("has_existing_memory", existingMemory != nil).
		Msg("Planning memory actions with LLM")

	// Call LLM
	response, err := p.llm.Complete(ctx, prompt, memory.LLMOptions{
		Model:          p.config.Model,
		Temperature:    p.config.Temperature,
		MaxTokens:      p.config.MaxTokens,
		ResponseFormat: "json",
	})
	if err != nil {
		return nil, fmt.Errorf("llm completion failed: %w", err)
	}

	// Parse JSON response
	var llmResp LLMMemoryActionResponse
	if err := json.Unmarshal([]byte(response), &llmResp); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	// Convert to MemoryAction
	action := &memory.MemoryAction{
		Add: memory.MemoryAddActions{
			UserMemory:    llmResp.Add.UserMemory,
			ProjectMemory: llmResp.Add.ProjectMemory,
			Episodic:      llmResp.Add.Episodic,
		},
		Delete: llmResp.Delete,
	}

	// Apply scoring enhancements
	p.enhanceScoring(action, req)

	// Detect conflicts if enabled
	if p.config.DetectConflicts && existingMemory != nil {
		p.detectAndResolveConflicts(action, existingMemory)
	}

	log.Info().
		Int("user_memory_add", len(action.Add.UserMemory)).
		Int("project_memory_add", len(action.Add.ProjectMemory)).
		Int("episodic_add", len(action.Add.Episodic)).
		Int("delete", len(action.Delete)).
		Str("reasoning", llmResp.Reasoning).
		Msg("Memory actions planned with LLM")

	return action, nil
}

// buildMemoryActionPrompt constructs the LLM prompt for memory action planning
func (p *Planner) buildMemoryActionPrompt(req memory.MemoryObserveRequest, existingMemory *ExistingMemoryContext) string {
	prompt := `You are a memory management system. Analyze the conversation and decide what information should be stored in long-term memory.

Your task:
1. Identify facts worth remembering (user preferences, project decisions, important context)
2. Detect contradictions with existing memory
3. Assign appropriate importance/confidence levels
4. Create episodic events for significant interactions

Rules:
- Only promote facts mentioned explicitly or repeatedly
- Mark contradictions for deletion
- Assign importance: low, medium, high, critical
- Assign confidence: 0.0-1.0 for project facts
- Be conservative - don't store trivial information

`

	// Add existing memory context if available
	if p.config.IncludeContext && existingMemory != nil {
		prompt += p.formatExistingMemory(existingMemory)
	}

	// Add conversation
	prompt += "Recent Conversation:\n"
	for _, msg := range req.Messages {
		prompt += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
	}

	// Add response format
	prompt += `
Return your analysis as JSON with this exact structure:
{
  "delete": ["memory_id_1", "memory_id_2"],
  "add": {
    "user_memory": [
      {
        "scope": "preference|profile|skill|other",
        "key": "descriptive_key",
        "text": "the fact to remember",
        "importance": "low|medium|high|critical"
      }
    ],
    "project_memory": [
      {
        "kind": "decision|assumption|risk|metric|fact",
        "title": "short title",
        "text": "detailed description",
        "confidence": 0.8
      }
    ],
    "episodic_memory": [
      {
        "kind": "tool_result|decision|incident|milestone",
        "text": "what happened"
      }
    ]
  },
  "reasoning": "brief explanation of your decisions"
}

Only include items that are truly worth remembering. Empty arrays are fine.
Ensure the response is valid JSON.`

	return prompt
}

// formatExistingMemory formats existing memory for the prompt
func (p *Planner) formatExistingMemory(existing *ExistingMemoryContext) string {
	var builder strings.Builder

	builder.WriteString("Existing Memory (check for contradictions):\n\n")

	if len(existing.UserMemory) > 0 {
		builder.WriteString("User Memory:\n")
		for _, item := range existing.UserMemory {
			builder.WriteString(fmt.Sprintf("- [%s] %s: %s (score: %d)\n",
				item.ID, item.Scope, item.Text, item.Score))
		}
		builder.WriteString("\n")
	}

	if len(existing.ProjectFacts) > 0 {
		builder.WriteString("Project Facts:\n")
		for _, fact := range existing.ProjectFacts {
			builder.WriteString(fmt.Sprintf("- [%s] %s: %s (confidence: %.2f)\n",
				fact.ID, fact.Kind, fact.Text, fact.Confidence))
		}
		builder.WriteString("\n")
	}

	return builder.String()
}

// enhanceScoring applies additional scoring logic based on conversation patterns
func (p *Planner) enhanceScoring(action *memory.MemoryAction, req memory.MemoryObserveRequest) {
	// Check for explicit "remember" commands
	hasExplicitRemember := false
	for _, msg := range req.Messages {
		if msg.Role == "user" {
			content := strings.ToLower(msg.Content)
			if strings.Contains(content, "remember") || strings.Contains(content, "don't forget") {
				hasExplicitRemember = true
				break
			}
		}
	}

	// Boost importance for explicit remember commands
	if hasExplicitRemember {
		for i := range action.Add.UserMemory {
			if action.Add.UserMemory[i].Importance == "medium" {
				action.Add.UserMemory[i].Importance = "high"
			} else if action.Add.UserMemory[i].Importance == "low" {
				action.Add.UserMemory[i].Importance = "medium"
			}
		}

		for i := range action.Add.ProjectMemory {
			newConfidence := action.Add.ProjectMemory[i].Confidence + 0.1
			if newConfidence > 1.0 {
				newConfidence = 1.0
			}
			action.Add.ProjectMemory[i].Confidence = newConfidence
		}
	}
}

// detectAndResolveConflicts detects contradictions and resolves them
func (p *Planner) detectAndResolveConflicts(action *memory.MemoryAction, existing *ExistingMemoryContext) {
	// Check new user memory against existing
	for _, newItem := range action.Add.UserMemory {
		for _, existingItem := range existing.UserMemory {
			// Same scope and key suggests potential conflict
			if newItem.Scope == existingItem.Scope && newItem.Key == existingItem.Key {
				// Check if texts are different (potential contradiction)
				if !strings.EqualFold(newItem.Text, existingItem.Text) {
					// Mark old item for deletion
					action.Delete = append(action.Delete, existingItem.ID)
					log.Info().
						Str("old_id", existingItem.ID).
						Str("old_text", existingItem.Text).
						Str("new_text", newItem.Text).
						Msg("Detected contradiction, marking old memory for deletion")
				}
			}
		}
	}

	// Check new project facts against existing
	for _, newFact := range action.Add.ProjectMemory {
		for _, existingFact := range existing.ProjectFacts {
			// Same kind and similar title suggests potential conflict
			if newFact.Kind == existingFact.Kind {
				// Simple similarity check (could be enhanced with embeddings)
				if strings.Contains(strings.ToLower(existingFact.Title), strings.ToLower(newFact.Title)) ||
					strings.Contains(strings.ToLower(newFact.Title), strings.ToLower(existingFact.Title)) {

					// If texts are different, it might be an update
					if !strings.EqualFold(newFact.Text, existingFact.Text) {
						// Reduce confidence of old fact instead of deleting
						action.Delete = append(action.Delete, existingFact.ID)
						log.Info().
							Str("old_id", existingFact.ID).
							Str("kind", existingFact.Kind).
							Msg("Detected potential update, marking old fact for deletion")
					}
				}
			}
		}
	}
}

// planWithHeuristics is the fallback heuristic-based planning
func (p *Planner) planWithHeuristics(ctx context.Context, req memory.MemoryObserveRequest) *memory.MemoryAction {
	action := &memory.MemoryAction{
		Add: memory.MemoryAddActions{
			UserMemory:    []memory.UserMemoryItemInput{},
			ProjectMemory: []memory.ProjectFactInput{},
			Episodic:      []memory.EpisodicEventInput{},
		},
		Delete: []string{},
	}

	// Analyze each message
	for _, msg := range req.Messages {
		if msg.Role == "user" {
			p.analyzeUserMessage(msg.Content, action, req.ProjectID)
		}

		// Always create episodic event for the interaction
		action.Add.Episodic = append(action.Add.Episodic, memory.EpisodicEventInput{
			Text: formatEpisodicText(msg.Role, msg.Content),
			Kind: "interaction",
		})
	}

	return action
}

// analyzeUserMessage analyzes a user message for memory extraction (heuristic)
func (p *Planner) analyzeUserMessage(content string, action *memory.MemoryAction, projectID string) {
	content = strings.TrimSpace(content)
	if content == "" {
		return
	}

	contentLower := strings.ToLower(content)

	// Detect user preferences
	if p.isPreference(contentLower) {
		importance := p.scorer.AnalyzeTextImportance(content)
		action.Add.UserMemory = append(action.Add.UserMemory, memory.UserMemoryItemInput{
			Scope:      "preference",
			Key:        "user_preference",
			Text:       content,
			Importance: importance,
		})
	}

	// Detect project decisions (only if project_id is set)
	if projectID != "" && p.isDecision(contentLower) {
		confidence := p.calculateConfidence(content)
		action.Add.ProjectMemory = append(action.Add.ProjectMemory, memory.ProjectFactInput{
			Kind:       "decision",
			Title:      extractTitle(content),
			Text:       content,
			Confidence: confidence,
		})
	}

	// Detect requirements
	if projectID != "" && p.isRequirement(contentLower) {
		confidence := p.calculateConfidence(content)
		action.Add.ProjectMemory = append(action.Add.ProjectMemory, memory.ProjectFactInput{
			Kind:       "assumption",
			Title:      extractTitle(content),
			Text:       content,
			Confidence: confidence,
		})
	}

	// Detect constraints
	if projectID != "" && p.isConstraint(contentLower) {
		confidence := p.calculateConfidence(content)
		action.Add.ProjectMemory = append(action.Add.ProjectMemory, memory.ProjectFactInput{
			Kind:       "risk",
			Title:      extractTitle(content),
			Text:       content,
			Confidence: confidence,
		})
	}
}

// Pattern detection functions (heuristic)

func (p *Planner) isPreference(text string) bool {
	patterns := []string{
		"i prefer", "i like", "i love", "i want",
		"i always", "i usually", "i typically",
		"my preference", "i'd rather",
	}

	for _, pattern := range patterns {
		if strings.Contains(text, pattern) {
			return true
		}
	}
	return false
}

func (p *Planner) isDecision(text string) bool {
	patterns := []string{
		"we should", "let's use", "we'll use", "we decided",
		"we're going to", "we will", "let's go with",
		"we chose", "we selected",
	}

	for _, pattern := range patterns {
		if strings.Contains(text, pattern) {
			return true
		}
	}
	return false
}

func (p *Planner) isRequirement(text string) bool {
	patterns := []string{
		"we need", "must have", "required", "requirement",
		"has to", "needs to", "should support",
	}

	for _, pattern := range patterns {
		if strings.Contains(text, pattern) {
			return true
		}
	}
	return false
}

func (p *Planner) isConstraint(text string) bool {
	patterns := []string{
		"can't", "cannot", "must not", "shouldn't",
		"limited to", "restricted", "constraint",
		"not allowed", "forbidden",
	}

	for _, pattern := range patterns {
		if strings.Contains(text, pattern) {
			return true
		}
	}
	return false
}

// Helper functions

func (p *Planner) calculateConfidence(text string) float32 {
	// Higher confidence for definitive statements
	textLower := strings.ToLower(text)

	if strings.Contains(textLower, "definitely") || strings.Contains(textLower, "certainly") {
		return 0.95
	}
	if strings.Contains(textLower, "probably") || strings.Contains(textLower, "likely") {
		return 0.75
	}
	if strings.Contains(textLower, "maybe") || strings.Contains(textLower, "might") {
		return 0.6
	}

	return 0.8 // Default confidence
}

func extractTitle(text string) string {
	// Extract first sentence or first 50 characters as title
	sentences := strings.Split(text, ".")
	if len(sentences) > 0 {
		title := strings.TrimSpace(sentences[0])
		if len(title) > 100 {
			title = title[:97] + "..."
		}
		return title
	}

	if len(text) > 100 {
		return text[:97] + "..."
	}
	return text
}

func formatEpisodicText(role, content string) string {
	// Truncate long content for episodic events
	if len(content) > 500 {
		content = content[:497] + "..."
	}
	return role + ": " + content
}

// ExistingMemoryContext holds existing memory for conflict detection
type ExistingMemoryContext struct {
	UserMemory   []memory.UserMemoryItem
	ProjectFacts []memory.ProjectFact
}
