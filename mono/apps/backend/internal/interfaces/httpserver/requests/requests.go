package requests

// CreateResponseRequest represents a request to create a response
type CreateResponseRequest struct {
	User                 string                 `json:"user,omitempty"`
	Model                string                 `json:"model" binding:"required"`
	Input                interface{}            `json:"input" binding:"required"`
	SystemPrompt         *string                `json:"system_prompt,omitempty"`
	Temperature          *float64               `json:"temperature,omitempty"`
	MaxTokens            *int                   `json:"max_tokens,omitempty"`
	Stream               *bool                  `json:"stream,omitempty"`
	Background           *bool                  `json:"background,omitempty"`
	Store                *bool                  `json:"store,omitempty"`
	ToolChoice           *ToolChoice            `json:"tool_choice,omitempty"`
	Tools                []ToolDefinition       `json:"tools,omitempty"`
	PreviousResponseID   *string                `json:"previous_response_id,omitempty"`
	Conversation         *string                `json:"conversation,omitempty"`
	ParentConversationID *string                `json:"parent_conversation_id,omitempty"`
	Metadata             map[string]interface{} `json:"metadata,omitempty"`
}

// ToolDefinition represents a tool definition in a request
type ToolDefinition struct {
	Type     string              `json:"type"`
	Function ToolFunctionSchema  `json:"function"`
}

// ToolFunctionSchema represents the function schema for a tool
type ToolFunctionSchema struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// ToolChoice represents a tool choice specification
type ToolChoice struct {
	Type     string                 `json:"type,omitempty"`
	Tool     string                 `json:"tool,omitempty"`
	Function ToolChoiceFunction     `json:"function,omitempty"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

// ToolChoiceFunction represents function specification in tool choice
type ToolChoiceFunction struct {
	Name string `json:"name,omitempty"`
}
