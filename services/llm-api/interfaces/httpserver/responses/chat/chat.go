package chatresponses

// ChatCompletionResponse wraps the OpenAI response with conversation context
type ChatCompletionResponse struct {
	ID                string                   `json:"id"`
	Object            string                   `json:"object"`
	Created           int64                    `json:"created"`
	Model             string                   `json:"model"`
	Choices           []ChatCompletionChoice   `json:"choices"`
	Usage             *ChatCompletionUsage     `json:"usage,omitempty"`
	Conversation      *ChatConversationContext `json:"conversation,omitempty"`
	SystemFingerprint string                   `json:"system_fingerprint,omitempty"`
}

// ChatCompletionChoice represents a single completion choice
type ChatCompletionChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason,omitempty"`
	Logprobs     interface{} `json:"logprobs,omitempty"`
}

// ChatMessage represents a single chat message
type ChatMessage struct {
	Role             string      `json:"role"`
	Content          interface{} `json:"content"` // string or array of content parts
	Name             string      `json:"name,omitempty"`
	FunctionCall     interface{} `json:"function_call,omitempty"`
	ToolCalls        interface{} `json:"tool_calls,omitempty"`
	ToolCallID       string      `json:"tool_call_id,omitempty"`
	ReasoningContent string      `json:"reasoning_content,omitempty"`
}

// ChatCompletionUsage represents token usage statistics
type ChatCompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatConversationContext provides conversation information in response
type ChatConversationContext struct {
	ID    string  `json:"id"`
	Title *string `json:"title,omitempty"`
}

// ChatCompletionStreamChunk represents a chunk in streaming response
type ChatCompletionStreamChunk struct {
	ID           string                       `json:"id"`
	Object       string                       `json:"object"`
	Created      int64                        `json:"created"`
	Model        string                       `json:"model"`
	Choices      []ChatCompletionStreamChoice `json:"choices"`
	Conversation *ChatConversationContext     `json:"conversation,omitempty"`
}

// ChatCompletionStreamChoice represents a choice in streaming response
type ChatCompletionStreamChoice struct {
	Index        int              `json:"index"`
	Delta        ChatMessageDelta `json:"delta"`
	FinishReason *string          `json:"finish_reason,omitempty"`
}

// ChatMessageDelta represents incremental message content
type ChatMessageDelta struct {
	Role             string      `json:"role,omitempty"`
	Content          string      `json:"content,omitempty"`
	FunctionCall     interface{} `json:"function_call,omitempty"`
	ToolCalls        interface{} `json:"tool_calls,omitempty"`
	ReasoningContent string      `json:"reasoning_content,omitempty"`
}
