// Package llm provides LLM-related functionality.
package llm

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// CodeFixer provides LLM-based code fixing capabilities.
type CodeFixer struct {
	provider                 Provider
	model                    string
	disableCustomTemperature bool
}

// NewCodeFixer creates a new code fixer with the given LLM provider.
func NewCodeFixer(provider Provider, model string) *CodeFixer {
	if model == "" {
		model = "gpt-4o-mini" // Default model for code fixing
	}
	return &CodeFixer{
		provider: provider,
		model:    model,
	}
}

// SetDisableCustomTemperature controls whether explicit temperature values are sent.
func (cf *CodeFixer) SetDisableCustomTemperature(disable bool) {
	cf.disableCustomTemperature = disable
}

func (cf *CodeFixer) temperaturePtr(value float64) *float64 {
	if cf.disableCustomTemperature {
		return nil
	}
	return floatPtr(value)
}

// FixCode attempts to fix code that produced an error using the LLM.
func (cf *CodeFixer) FixCode(ctx context.Context, code string, errorMsg string, language string) (string, error) {
	if code == "" {
		return "", fmt.Errorf("empty code provided")
	}
	if language == "" {
		language = "python"
	}

	systemPrompt := fmt.Sprintf(`You are an expert %s programmer. Your task is to fix code that has produced an error.

Rules:
1. Analyze the error message and identify the root cause
2. Fix ONLY the issue causing the error - do not change working code
3. Return ONLY the fixed code, no explanations
4. Wrap your code in a single code block with the language identifier
5. Preserve the original code structure and logic
6. If the error is due to a missing import, add the import
7. If the error is a syntax error, fix the syntax
8. Do not add any comments explaining the fix
9. If the code writes files, use /home/user for output paths`, language)

	userPrompt := fmt.Sprintf(`The following %s code produced an error. Fix it.

**Original Code:**
%s%s
%s

**Error Message:**
%s

Provide the fixed code:`, language, "```"+language+"\n", code, "```", errorMsg)

	req := ChatCompletionRequest{
		Model: cf.model,
		Messages: []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: cf.temperaturePtr(0.2), // Low temperature for more deterministic fixes
		MaxTokens:   intPtr(40000),
		Stream:      false,
	}

	resp, err := cf.provider.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("LLM returned no choices")
	}

	fixedCode := extractCodeFromResponse(resp.Choices[0].Message.GetContentAsString(), language)
	if fixedCode == "" {
		return "", fmt.Errorf("could not extract fixed code from LLM response")
	}

	return fixedCode, nil
}

// Generate generates content based on a prompt using the LLM.
func (cf *CodeFixer) Generate(ctx context.Context, prompt string) (string, error) {
	return cf.GenerateWithModel(ctx, prompt, "")
}

// GenerateWithModel generates content based on a prompt using the specified model.
// If model is empty, uses the default model configured in CodeFixer.
func (cf *CodeFixer) GenerateWithModel(ctx context.Context, prompt string, model string) (string, error) {
	if prompt == "" {
		return "", fmt.Errorf("empty prompt provided")
	}

	useModel := cf.model
	if model != "" {
		useModel = model
	}

	req := ChatCompletionRequest{
		Model: useModel,
		Messages: []ChatMessage{
			{Role: "user", Content: prompt},
		},
		Temperature: cf.temperaturePtr(0.7), // Moderate temperature for balanced creativity
		MaxTokens:   intPtr(40000),
		Stream:      false,
	}

	resp, err := cf.provider.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("LLM returned no choices")
	}

	content := resp.Choices[0].Message.GetContentAsString()
	return content, nil
}

// GenerateWithModelWithMaxTokens generates content using a max_tokens override.
func (cf *CodeFixer) GenerateWithModelWithMaxTokens(ctx context.Context, prompt string, model string, maxTokens *int) (string, error) {
	if prompt == "" {
		return "", fmt.Errorf("empty prompt provided")
	}

	useModel := cf.model
	if model != "" {
		useModel = model
	}

	req := ChatCompletionRequest{
		Model: useModel,
		Messages: []ChatMessage{
			{Role: "user", Content: prompt},
		},
		Temperature: cf.temperaturePtr(0.7),
		MaxTokens:   resolveMaxTokens(40000, maxTokens),
		Stream:      false,
	}

	resp, err := cf.provider.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("LLM returned no choices")
	}

	return resp.Choices[0].Message.GetContentAsString(), nil
}

// GenerateWithModelWithTemperature generates content using the specified temperature.
func (cf *CodeFixer) GenerateWithModelWithTemperature(ctx context.Context, prompt string, model string, temperature float64) (string, error) {
	if prompt == "" {
		return "", fmt.Errorf("empty prompt provided")
	}

	useModel := cf.model
	if model != "" {
		useModel = model
	}

	req := ChatCompletionRequest{
		Model: useModel,
		Messages: []ChatMessage{
			{Role: "user", Content: prompt},
		},
		Temperature: cf.temperaturePtr(temperature),
		MaxTokens:   intPtr(40000),
		Stream:      false,
	}

	resp, err := cf.provider.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("LLM returned no choices")
	}

	content := resp.Choices[0].Message.GetContentAsString()
	return content, nil
}

// GenerateWithModelWithTemperatureAndMaxTokens generates content with temperature and max_tokens overrides.
func (cf *CodeFixer) GenerateWithModelWithTemperatureAndMaxTokens(ctx context.Context, prompt string, model string, temperature float64, maxTokens *int) (string, error) {
	if prompt == "" {
		return "", fmt.Errorf("empty prompt provided")
	}

	useModel := cf.model
	if model != "" {
		useModel = model
	}

	req := ChatCompletionRequest{
		Model: useModel,
		Messages: []ChatMessage{
			{Role: "user", Content: prompt},
		},
		Temperature: cf.temperaturePtr(temperature),
		MaxTokens:   resolveMaxTokens(40000, maxTokens),
		Stream:      false,
	}

	resp, err := cf.provider.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("LLM returned no choices")
	}

	return resp.Choices[0].Message.GetContentAsString(), nil
}

// extractCodeFromResponse extracts code from an LLM response that may contain markdown.
func extractCodeFromResponse(response string, language string) string {
	if response == "" {
		return ""
	}

	// Try to find code block with language identifier
	patterns := []string{
		fmt.Sprintf("```%s\n([\\s\\S]*?)```", language),
		"```\\w*\n([\\s\\S]*?)```",
		"```([\\s\\S]*?)```",
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(response)
		if len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}

	// If no code block found, assume the whole response is code
	// but only if it doesn't look like prose
	trimmed := strings.TrimSpace(response)
	if !strings.HasPrefix(trimmed, "I ") &&
		!strings.HasPrefix(trimmed, "The ") &&
		!strings.HasPrefix(trimmed, "Here") {
		return trimmed
	}

	return ""
}

// floatPtr returns a pointer to a float64.
func floatPtr(f float64) *float64 {
	return &f
}

// intPtr returns a pointer to an int.
func intPtr(i int) *int {
	return &i
}

// GenerateWithSystemPrompt generates content using a system prompt and user prompt.
// Uses low temperature for more deterministic output.
func (cf *CodeFixer) GenerateWithSystemPrompt(ctx context.Context, systemPrompt string, userPrompt string, model string) (string, error) {
	if userPrompt == "" {
		return "", fmt.Errorf("empty prompt provided")
	}

	useModel := cf.model
	if model != "" {
		useModel = model
	}

	messages := []ChatMessage{
		{Role: "user", Content: userPrompt},
	}
	if systemPrompt != "" {
		messages = []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		}
	}

	req := ChatCompletionRequest{
		Model:       useModel,
		Messages:    messages,
		Temperature: cf.temperaturePtr(0.2), // Low temperature for structured output
		MaxTokens:   intPtr(40000),
		Stream:      false,
	}

	resp, err := cf.provider.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("LLM returned no choices")
	}

	return resp.Choices[0].Message.GetContentAsString(), nil
}

// GenerateWithSystemPromptWithMaxTokens generates content using a max_tokens override.
func (cf *CodeFixer) GenerateWithSystemPromptWithMaxTokens(ctx context.Context, systemPrompt string, userPrompt string, model string, maxTokens *int) (string, error) {
	if userPrompt == "" {
		return "", fmt.Errorf("empty prompt provided")
	}

	useModel := cf.model
	if model != "" {
		useModel = model
	}

	messages := []ChatMessage{
		{Role: "user", Content: userPrompt},
	}
	if systemPrompt != "" {
		messages = []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		}
	}

	req := ChatCompletionRequest{
		Model:       useModel,
		Messages:    messages,
		Temperature: cf.temperaturePtr(0.2),
		MaxTokens:   resolveMaxTokens(40000, maxTokens),
		Stream:      false,
	}

	resp, err := cf.provider.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("LLM returned no choices")
	}

	return resp.Choices[0].Message.GetContentAsString(), nil
}

// GenerateWithSystemPromptWithTemperature generates content using a system prompt and temperature override.
func (cf *CodeFixer) GenerateWithSystemPromptWithTemperature(ctx context.Context, systemPrompt string, userPrompt string, model string, temperature float64) (string, error) {
	if userPrompt == "" {
		return "", fmt.Errorf("empty prompt provided")
	}

	useModel := cf.model
	if model != "" {
		useModel = model
	}

	messages := []ChatMessage{
		{Role: "user", Content: userPrompt},
	}
	if systemPrompt != "" {
		messages = []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		}
	}

	req := ChatCompletionRequest{
		Model:       useModel,
		Messages:    messages,
		Temperature: cf.temperaturePtr(temperature),
		MaxTokens:   intPtr(40000),
		Stream:      false,
	}

	resp, err := cf.provider.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("LLM returned no choices")
	}

	return resp.Choices[0].Message.GetContentAsString(), nil
}

// GenerateWithSystemPromptWithTemperatureAndMaxTokens generates content with temperature and max_tokens overrides.
func (cf *CodeFixer) GenerateWithSystemPromptWithTemperatureAndMaxTokens(ctx context.Context, systemPrompt string, userPrompt string, model string, temperature float64, maxTokens *int) (string, error) {
	if userPrompt == "" {
		return "", fmt.Errorf("empty prompt provided")
	}

	useModel := cf.model
	if model != "" {
		useModel = model
	}

	messages := []ChatMessage{
		{Role: "user", Content: userPrompt},
	}
	if systemPrompt != "" {
		messages = []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		}
	}

	req := ChatCompletionRequest{
		Model:       useModel,
		Messages:    messages,
		Temperature: cf.temperaturePtr(temperature),
		MaxTokens:   resolveMaxTokens(40000, maxTokens),
		Stream:      false,
	}

	resp, err := cf.provider.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("LLM returned no choices")
	}

	return resp.Choices[0].Message.GetContentAsString(), nil
}

// GenerateWithStructuredOutput generates content using response_format json_schema.
func (cf *CodeFixer) GenerateWithStructuredOutput(ctx context.Context, systemPrompt string, userPrompt string, model string, schema map[string]any) (string, error) {
	if userPrompt == "" {
		return "", fmt.Errorf("empty prompt provided")
	}
	if schema == nil {
		return "", fmt.Errorf("schema is required for structured output")
	}

	useModel := cf.model
	if model != "" {
		useModel = model
	}

	messages := []ChatMessage{
		{Role: "user", Content: userPrompt},
	}
	if systemPrompt != "" {
		messages = []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		}
	}

	req := ChatCompletionRequest{
		Model:       useModel,
		Messages:    messages,
		Temperature: cf.temperaturePtr(0.2),
		MaxTokens:   intPtr(40000),
		Stream:      false,
		ResponseFormat: &ResponseFormat{
			Type: "json_schema",
			JSONSchema: &JSONSchema{
				Name:   "output_schema",
				Schema: schema,
				Strict: true,
			},
		},
	}

	resp, err := cf.provider.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("LLM returned no choices")
	}

	return resp.Choices[0].Message.GetContentAsString(), nil
}

// GenerateWithStructuredOutputWithMaxTokens generates structured output with a max_tokens override.
func (cf *CodeFixer) GenerateWithStructuredOutputWithMaxTokens(ctx context.Context, systemPrompt string, userPrompt string, model string, schema map[string]any, maxTokens *int) (string, error) {
	if userPrompt == "" {
		return "", fmt.Errorf("empty prompt provided")
	}
	if schema == nil {
		return "", fmt.Errorf("schema is required for structured output")
	}

	useModel := cf.model
	if model != "" {
		useModel = model
	}

	messages := []ChatMessage{
		{Role: "user", Content: userPrompt},
	}
	if systemPrompt != "" {
		messages = []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		}
	}

	req := ChatCompletionRequest{
		Model:       useModel,
		Messages:    messages,
		Temperature: cf.temperaturePtr(0.2),
		MaxTokens:   resolveMaxTokens(40000, maxTokens),
		Stream:      false,
		ResponseFormat: &ResponseFormat{
			Type: "json_schema",
			JSONSchema: &JSONSchema{
				Name:   "output_schema",
				Schema: schema,
				Strict: true,
			},
		},
	}

	resp, err := cf.provider.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("LLM returned no choices")
	}

	return resp.Choices[0].Message.GetContentAsString(), nil
}

// GenerateWithStructuredOutputWithTemperature generates content using response_format json_schema and a temperature override.
func (cf *CodeFixer) GenerateWithStructuredOutputWithTemperature(ctx context.Context, systemPrompt string, userPrompt string, model string, schema map[string]any, temperature float64) (string, error) {
	if userPrompt == "" {
		return "", fmt.Errorf("empty prompt provided")
	}
	if schema == nil {
		return "", fmt.Errorf("schema is required for structured output")
	}

	useModel := cf.model
	if model != "" {
		useModel = model
	}

	messages := []ChatMessage{
		{Role: "user", Content: userPrompt},
	}
	if systemPrompt != "" {
		messages = []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		}
	}

	req := ChatCompletionRequest{
		Model:       useModel,
		Messages:    messages,
		Temperature: cf.temperaturePtr(temperature),
		MaxTokens:   intPtr(40000),
		Stream:      false,
		ResponseFormat: &ResponseFormat{
			Type: "json_schema",
			JSONSchema: &JSONSchema{
				Name:   "output_schema",
				Schema: schema,
				Strict: true,
			},
		},
	}

	resp, err := cf.provider.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("LLM returned no choices")
	}

	return resp.Choices[0].Message.GetContentAsString(), nil
}

// GenerateWithStructuredOutputWithTemperatureAndMaxTokens generates structured output with temperature and max_tokens overrides.
func (cf *CodeFixer) GenerateWithStructuredOutputWithTemperatureAndMaxTokens(ctx context.Context, systemPrompt string, userPrompt string, model string, schema map[string]any, temperature float64, maxTokens *int) (string, error) {
	if userPrompt == "" {
		return "", fmt.Errorf("empty prompt provided")
	}
	if schema == nil {
		return "", fmt.Errorf("schema is required for structured output")
	}

	useModel := cf.model
	if model != "" {
		useModel = model
	}

	messages := []ChatMessage{
		{Role: "user", Content: userPrompt},
	}
	if systemPrompt != "" {
		messages = []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		}
	}

	req := ChatCompletionRequest{
		Model:       useModel,
		Messages:    messages,
		Temperature: cf.temperaturePtr(temperature),
		MaxTokens:   resolveMaxTokens(40000, maxTokens),
		Stream:      false,
		ResponseFormat: &ResponseFormat{
			Type: "json_schema",
			JSONSchema: &JSONSchema{
				Name:   "output_schema",
				Schema: schema,
				Strict: true,
			},
		},
	}

	resp, err := cf.provider.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("LLM returned no choices")
	}

	return resp.Choices[0].Message.GetContentAsString(), nil
}

func resolveMaxTokens(defaultTokens int, override *int) *int {
	if override == nil || *override <= 0 {
		return intPtr(defaultTokens)
	}
	return override
}

// LLMResult contains the LLM response content and token usage.
type LLMResult struct {
	Content string
	Usage   *Usage
}

// GenerateWithStructuredOutputWithMaxTokensAndUsage generates structured output with max_tokens and returns usage.
func (cf *CodeFixer) GenerateWithStructuredOutputWithMaxTokensAndUsage(ctx context.Context, systemPrompt string, userPrompt string, model string, schema map[string]any, maxTokens *int) (*LLMResult, error) {
	if userPrompt == "" {
		return nil, fmt.Errorf("empty prompt provided")
	}
	if schema == nil {
		return nil, fmt.Errorf("schema is required for structured output")
	}

	useModel := cf.model
	if model != "" {
		useModel = model
	}

	messages := []ChatMessage{
		{Role: "user", Content: userPrompt},
	}
	if systemPrompt != "" {
		messages = []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		}
	}

	req := ChatCompletionRequest{
		Model:       useModel,
		Messages:    messages,
		Temperature: cf.temperaturePtr(0.2),
		MaxTokens:   resolveMaxTokens(40000, maxTokens),
		Stream:      false,
		ResponseFormat: &ResponseFormat{
			Type: "json_schema",
			JSONSchema: &JSONSchema{
				Name:   "output_schema",
				Schema: schema,
				Strict: true,
			},
		},
	}

	resp, err := cf.provider.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("LLM returned no choices")
	}

	return &LLMResult{
		Content: resp.Choices[0].Message.GetContentAsString(),
		Usage:   resp.Usage,
	}, nil
}

// GenerateWithStructuredOutputWithTemperatureAndMaxTokensAndUsage generates structured output with temperature, max_tokens and returns usage.
func (cf *CodeFixer) GenerateWithStructuredOutputWithTemperatureAndMaxTokensAndUsage(ctx context.Context, systemPrompt string, userPrompt string, model string, schema map[string]any, temperature float64, maxTokens *int) (*LLMResult, error) {
	if userPrompt == "" {
		return nil, fmt.Errorf("empty prompt provided")
	}
	if schema == nil {
		return nil, fmt.Errorf("schema is required for structured output")
	}

	useModel := cf.model
	if model != "" {
		useModel = model
	}

	messages := []ChatMessage{
		{Role: "user", Content: userPrompt},
	}
	if systemPrompt != "" {
		messages = []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		}
	}

	req := ChatCompletionRequest{
		Model:       useModel,
		Messages:    messages,
		Temperature: cf.temperaturePtr(temperature),
		MaxTokens:   resolveMaxTokens(40000, maxTokens),
		Stream:      false,
		ResponseFormat: &ResponseFormat{
			Type: "json_schema",
			JSONSchema: &JSONSchema{
				Name:   "output_schema",
				Schema: schema,
				Strict: true,
			},
		},
	}

	resp, err := cf.provider.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("LLM returned no choices")
	}

	return &LLMResult{
		Content: resp.Choices[0].Message.GetContentAsString(),
		Usage:   resp.Usage,
	}, nil
}
