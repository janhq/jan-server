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
	provider Provider
	model    string
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
		Temperature: floatPtr(0.2), // Low temperature for more deterministic fixes
		MaxTokens:   intPtr(4096),
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
