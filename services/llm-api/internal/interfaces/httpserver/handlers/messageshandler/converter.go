package messageshandler

import (
	"encoding/json"
	"fmt"
	"strings"

	messagesrequests "jan-server/services/llm-api/internal/interfaces/httpserver/requests/messages"
	messagesresponses "jan-server/services/llm-api/internal/interfaces/httpserver/responses/messages"

	openai "github.com/sashabaranov/go-openai"
)

// ConvertAnthropicToOpenAI converts an Anthropic Messages request to OpenAI format
func ConvertAnthropicToOpenAI(req *messagesrequests.AnthropicMessagesRequest) openai.ChatCompletionRequest {
	messages := make([]openai.ChatCompletionMessage, 0, len(req.Messages)+1)

	// Add system message if present
	if !req.System.IsEmpty() {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: req.System.GetText(),
		})
	}

	// Convert each Anthropic message to OpenAI format
	for _, msg := range req.Messages {
		openaiMsg := convertAnthropicMessage(msg)
		messages = append(messages, openaiMsg...)
	}

	request := openai.ChatCompletionRequest{
		Model:     req.Model,
		Messages:  messages,
		MaxTokens: req.MaxTokens,
		Stream:    req.Stream,
	}

	// Optional parameters
	if req.Temperature != nil {
		request.Temperature = *req.Temperature
	}
	if req.TopP != nil {
		request.TopP = *req.TopP
	}
	if len(req.StopSequences) > 0 {
		request.Stop = req.StopSequences
	}

	// Convert tools
	if len(req.Tools) > 0 {
		request.Tools = convertAnthropicTools(req.Tools)
	}

	// Convert tool_choice
	if req.ToolChoice != nil {
		request.ToolChoice = convertAnthropicToolChoice(req.ToolChoice)
	}

	// Handle user metadata
	if req.Metadata != nil && req.Metadata.UserID != "" {
		request.User = req.Metadata.UserID
	}

	return request
}

// convertAnthropicMessage converts a single Anthropic message to OpenAI format
// Returns a slice because tool results may need to be split into multiple messages
func convertAnthropicMessage(msg messagesrequests.AnthropicMessage) []openai.ChatCompletionMessage {
	role := msg.Role
	if role == "assistant" {
		role = openai.ChatMessageRoleAssistant
	} else {
		role = openai.ChatMessageRoleUser
	}

	// Handle simple text content
	if msg.Content.IsText() && msg.Content.Text != "" {
		return []openai.ChatCompletionMessage{{
			Role:    role,
			Content: msg.Content.Text,
		}}
	}

	// Handle content blocks
	var messages []openai.ChatCompletionMessage
	var textParts []openai.ChatMessagePart
	var toolCalls []openai.ToolCall

	for _, block := range msg.Content.Blocks {
		switch block.Type {
		case "text":
			textParts = append(textParts, openai.ChatMessagePart{
				Type: openai.ChatMessagePartTypeText,
				Text: block.Text,
			})

		case "image":
			// Handle image content blocks
			if block.Source != nil {
				imageURL := convertAnthropicImage(block.Source)
				if imageURL != nil {
					textParts = append(textParts, openai.ChatMessagePart{
						Type:     openai.ChatMessagePartTypeImageURL,
						ImageURL: imageURL,
					})
				}
			}

		case "tool_use":
			// Assistant message with tool calls
			toolCalls = append(toolCalls, openai.ToolCall{
				ID:   block.ID,
				Type: openai.ToolTypeFunction,
				Function: openai.FunctionCall{
					Name:      block.Name,
					Arguments: string(block.Input),
				},
			})

		case "tool_result":
			// Tool result - create a separate tool message
			content := ""
			if block.Content != nil {
				content = block.Content.GetText()
			}
			messages = append(messages, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				Content:    content,
				ToolCallID: block.ToolUseID,
			})
		}
	}

	// Build the main message
	if len(textParts) > 0 || len(toolCalls) > 0 {
		mainMsg := openai.ChatCompletionMessage{
			Role: role,
		}

		if len(textParts) == 1 && textParts[0].Type == openai.ChatMessagePartTypeText {
			// Simple text content
			mainMsg.Content = textParts[0].Text
		} else if len(textParts) > 0 {
			// Multi-part content
			mainMsg.MultiContent = textParts
		}

		if len(toolCalls) > 0 {
			mainMsg.ToolCalls = toolCalls
		}

		// Insert main message before tool results
		messages = append([]openai.ChatCompletionMessage{mainMsg}, messages...)
	}

	return messages
}

// convertAnthropicImage converts an Anthropic image source to OpenAI format
func convertAnthropicImage(source *messagesrequests.AnthropicImageSource) *openai.ChatMessageImageURL {
	if source == nil {
		return nil
	}

	var url string
	switch source.Type {
	case "base64":
		// Convert base64 to data URL
		mediaType := source.MediaType
		if mediaType == "" {
			mediaType = "image/png"
		}
		url = fmt.Sprintf("data:%s;base64,%s", mediaType, source.Data)
	case "url":
		url = source.URL
	default:
		return nil
	}

	return &openai.ChatMessageImageURL{
		URL: url,
	}
}

// convertAnthropicTools converts Anthropic tools to OpenAI format
func convertAnthropicTools(tools []messagesrequests.AnthropicTool) []openai.Tool {
	result := make([]openai.Tool, 0, len(tools))
	for _, tool := range tools {
		// Parse input_schema to extract parameters
		var params map[string]interface{}
		if len(tool.InputSchema) > 0 {
			_ = json.Unmarshal(tool.InputSchema, &params)
		}

		// Build function definition
		def := openai.FunctionDefinition{
			Name:        tool.Name,
			Description: tool.Description,
		}

		// Set parameters - OpenAI expects the schema directly
		if params != nil {
			def.Parameters = params
		}

		result = append(result, openai.Tool{
			Type:     openai.ToolTypeFunction,
			Function: &def,
		})
	}
	return result
}

// convertAnthropicToolChoice converts Anthropic tool_choice to OpenAI format
func convertAnthropicToolChoice(choice *messagesrequests.AnthropicToolChoice) interface{} {
	if choice == nil {
		return nil
	}

	switch choice.Type {
	case "auto":
		return "auto"
	case "any":
		// "any" in Anthropic means the model must use a tool
		return "required"
	case "tool":
		// Specific tool
		return openai.ToolChoice{
			Type: openai.ToolTypeFunction,
			Function: openai.ToolFunction{
				Name: choice.Name,
			},
		}
	default:
		return "auto"
	}
}

// ConvertOpenAIToAnthropic converts an OpenAI response to Anthropic format
func ConvertOpenAIToAnthropic(resp *openai.ChatCompletionResponse, model string) *messagesresponses.AnthropicMessagesResponse {
	content := make([]messagesresponses.AnthropicResponseContentBlock, 0)

	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]

		// Add text content if present
		if choice.Message.Content != "" {
			content = append(content, messagesresponses.AnthropicResponseContentBlock{
				Type: "text",
				Text: choice.Message.Content,
			})
		}

		// Add tool calls as tool_use blocks
		for _, toolCall := range choice.Message.ToolCalls {
			content = append(content, messagesresponses.AnthropicResponseContentBlock{
				Type:  "tool_use",
				ID:    toolCall.ID,
				Name:  toolCall.Function.Name,
				Input: json.RawMessage(toolCall.Function.Arguments),
			})
		}
	}

	// Determine stop reason
	stopReason := convertOpenAIStopReason(resp)

	return &messagesresponses.AnthropicMessagesResponse{
		ID:         resp.ID,
		Type:       "message",
		Role:       "assistant",
		Content:    content,
		Model:      model,
		StopReason: stopReason,
		Usage: messagesresponses.AnthropicUsage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		},
	}
}

// convertOpenAIStopReason converts OpenAI finish_reason to Anthropic stop_reason
func convertOpenAIStopReason(resp *openai.ChatCompletionResponse) *string {
	if len(resp.Choices) == 0 {
		return nil
	}

	reason := string(resp.Choices[0].FinishReason)
	var stopReason string

	switch reason {
	case "stop":
		stopReason = "end_turn"
	case "length":
		stopReason = "max_tokens"
	case "tool_calls":
		stopReason = "tool_use"
	case "content_filter":
		stopReason = "end_turn"
	default:
		stopReason = "end_turn"
	}

	return &stopReason
}

// GenerateMessageID generates a unique message ID in Anthropic format
func GenerateMessageID() string {
	// Use a simple format similar to Anthropic's
	return fmt.Sprintf("msg_%s", generateRandomID(24))
}

// generateRandomID generates a random alphanumeric ID
func generateRandomID(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[i%len(charset)]
	}
	return string(b)
}

// ConvertOpenAIStreamChunkToAnthropic converts an OpenAI streaming chunk to Anthropic events
// Returns a list of events to emit
func ConvertOpenAIStreamChunkToAnthropic(
	data string,
	state *StreamingState,
) []interface{} {
	var events []interface{}

	// Parse the chunk
	var chunk struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Index int `json:"index"`
			Delta struct {
				Content          string            `json:"content"`
				ReasoningContent string            `json:"reasoning_content"`
				ToolCalls        []openai.ToolCall `json:"tool_calls"`
			} `json:"delta"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage *struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return events
	}

	// Update state with message info
	if chunk.ID != "" && state.MessageID == "" {
		state.MessageID = chunk.ID
	}
	if chunk.Model != "" && state.Model == "" {
		state.Model = chunk.Model
	}

	// Emit message_start if not yet emitted
	if !state.MessageStarted {
		state.MessageStarted = true
		inputTokens := 0
		if chunk.Usage != nil {
			inputTokens = chunk.Usage.PromptTokens
			state.InputTokens = inputTokens
		}
		events = append(events, messagesresponses.NewMessageStartEvent(
			state.MessageID,
			state.Model,
			inputTokens,
		))
	}

	// Process choices
	for _, choice := range chunk.Choices {
		// Handle reasoning/thinking content
		if choice.Delta.ReasoningContent != "" {
			if !state.ThinkingBlockStarted {
				state.ThinkingBlockStarted = true
				state.ThinkingBlockIndex = state.CurrentBlockIndex
				state.CurrentBlockIndex++
				events = append(events, &messagesresponses.ContentBlockStartEvent{
					Type:  "content_block_start",
					Index: state.ThinkingBlockIndex,
					ContentBlock: messagesresponses.AnthropicResponseContentBlock{
						Type:     "thinking",
						Thinking: "",
					},
				})
			}
			events = append(events, messagesresponses.NewThinkingDeltaEvent(
				state.ThinkingBlockIndex,
				choice.Delta.ReasoningContent,
			))
		}

		// Handle text content
		if choice.Delta.Content != "" {
			// Close thinking block if transitioning
			if state.ThinkingBlockStarted && !state.ThinkingBlockStopped {
				state.ThinkingBlockStopped = true
				events = append(events, messagesresponses.NewContentBlockStopEvent(state.ThinkingBlockIndex))
			}

			if !state.TextBlockStarted {
				state.TextBlockStarted = true
				state.TextBlockIndex = state.CurrentBlockIndex
				state.CurrentBlockIndex++
				events = append(events, messagesresponses.NewContentBlockStartEvent(
					state.TextBlockIndex,
					"text",
				))
			}
			events = append(events, messagesresponses.NewTextDeltaEvent(
				state.TextBlockIndex,
				choice.Delta.Content,
			))
		}

		// Handle tool calls
		for _, toolCall := range choice.Delta.ToolCalls {
			var toolIndex int
			if toolCall.Index != nil {
				toolIndex = *toolCall.Index
			}
			if toolIndex < 0 {
				toolIndex = 0
			}

			// Initialize tool call state if needed
			if _, exists := state.ToolCalls[toolIndex]; !exists {
				// Close text block if transitioning
				if state.TextBlockStarted && !state.TextBlockStopped {
					state.TextBlockStopped = true
					events = append(events, messagesresponses.NewContentBlockStopEvent(state.TextBlockIndex))
				}

				state.ToolCalls[toolIndex] = &ToolCallState{
					ID:         toolCall.ID,
					Name:       toolCall.Function.Name,
					Arguments:  "",
					BlockIndex: state.CurrentBlockIndex,
					Started:    false,
				}
				state.CurrentBlockIndex++
			}

			tc := state.ToolCalls[toolIndex]

			// Update tool call info
			if toolCall.ID != "" {
				tc.ID = toolCall.ID
			}
			if toolCall.Function.Name != "" {
				tc.Name = toolCall.Function.Name
			}

			// Emit content_block_start for tool_use if not yet started
			if !tc.Started && tc.ID != "" && tc.Name != "" {
				tc.Started = true
				events = append(events, messagesresponses.NewContentBlockStartToolUseEvent(
					tc.BlockIndex,
					tc.ID,
					tc.Name,
				))
			}

			// Stream tool arguments
			if toolCall.Function.Arguments != "" {
				tc.Arguments += toolCall.Function.Arguments
				events = append(events, messagesresponses.NewInputJSONDeltaEvent(
					tc.BlockIndex,
					toolCall.Function.Arguments,
				))
			}
		}

		// Handle finish reason
		if choice.FinishReason != "" {
			state.FinishReason = choice.FinishReason
		}
	}

	// Update usage
	if chunk.Usage != nil {
		state.OutputTokens = chunk.Usage.CompletionTokens
		if chunk.Usage.PromptTokens > 0 {
			state.InputTokens = chunk.Usage.PromptTokens
		}
	}

	return events
}

// FinalizeAnthropicStream generates final events to close the stream
func FinalizeAnthropicStream(state *StreamingState) []interface{} {
	var events []interface{}

	// Close any open thinking block
	if state.ThinkingBlockStarted && !state.ThinkingBlockStopped {
		events = append(events, messagesresponses.NewContentBlockStopEvent(state.ThinkingBlockIndex))
	}

	// Close any open text block
	if state.TextBlockStarted && !state.TextBlockStopped {
		events = append(events, messagesresponses.NewContentBlockStopEvent(state.TextBlockIndex))
	}

	// Close any open tool call blocks
	for _, tc := range state.ToolCalls {
		if tc.Started && !tc.Stopped {
			tc.Stopped = true
			events = append(events, messagesresponses.NewContentBlockStopEvent(tc.BlockIndex))
		}
	}

	// Emit message_delta with stop reason and usage
	stopReason := "end_turn"
	switch state.FinishReason {
	case "stop":
		stopReason = "end_turn"
	case "length":
		stopReason = "max_tokens"
	case "tool_calls":
		stopReason = "tool_use"
	}
	events = append(events, messagesresponses.NewMessageDeltaEvent(stopReason, state.OutputTokens))

	// Emit message_stop
	events = append(events, messagesresponses.NewMessageStopEvent())

	return events
}

// StreamingState tracks the state during Anthropic streaming
type StreamingState struct {
	MessageID            string
	Model                string
	MessageStarted       bool
	CurrentBlockIndex    int
	TextBlockIndex       int
	TextBlockStarted     bool
	TextBlockStopped     bool
	ThinkingBlockIndex   int
	ThinkingBlockStarted bool
	ThinkingBlockStopped bool
	ToolCalls            map[int]*ToolCallState
	FinishReason         string
	InputTokens          int
	OutputTokens         int
}

// ToolCallState tracks state for a single tool call
type ToolCallState struct {
	ID         string
	Name       string
	Arguments  string
	BlockIndex int
	Started    bool
	Stopped    bool
}

// NewStreamingState creates a new streaming state
func NewStreamingState(messageID, model string) *StreamingState {
	return &StreamingState{
		MessageID:         messageID,
		Model:             model,
		CurrentBlockIndex: 0,
		ToolCalls:         make(map[int]*ToolCallState),
	}
}

// HasThinkingEnabled checks if thinking is enabled in the request
func HasThinkingEnabled(req *messagesrequests.AnthropicMessagesRequest) bool {
	return req.Thinking != nil && strings.ToLower(req.Thinking.Type) == "enabled"
}
