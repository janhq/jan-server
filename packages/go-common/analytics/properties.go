package analytics

// Standard property keys (matching frontend conventions)
const (
	// Context properties
	PropPlatform       = "platform"        // "web" | "desktop"
	PropEnvironment    = "environment"     // "dev" | "staging" | "prod"
	PropRequestID      = "request_id"      // X-Request-Id header
	PropUserStatus     = "user_status"     // "guest" | "authenticated"
	PropSessionID      = "session_id"      // Session identifier
	PropConversationID = "conversation_id" // Conversation identifier
	PropMessageID      = "message_id"      // Message identifier

	// Model/Mode properties
	PropModel        = "model"         // "v2-max", "v2-small", etc.
	PropMode         = "mode"          // "normal", "deep_research", etc.
	PropPreviousMode = "previous_mode" // Previous mode (for mode changes)
	PropProvider     = "provider"      // "openai", "anthropic", etc.
	PropIsStreaming  = "is_streaming"  // Whether streaming is enabled

	// Referrer properties
	PropReferrerSource = "referrer_source" // utm_source or referrer domain

	// Performance properties
	PropTTFRMs     = "ttfr_ms"     // Time to first response in milliseconds
	PropLatencyMs  = "latency_ms"  // Total latency in milliseconds
	PropDurationMs = "duration_ms" // Duration in milliseconds

	// Token properties
	PropTokensPrompt     = "tokens_prompt"     // Prompt tokens
	PropTokensCompletion = "tokens_completion" // Completion tokens
	PropTokensTotal      = "tokens_total"      // Total tokens

	// Status properties
	PropStatus       = "status"        // "success" | "stopped" | "error"
	PropErrorCode    = "error_code"    // Error code
	PropErrorMessage = "error_message" // Error message (will be sanitized)

	// Message properties
	PropMessageLength   = "message_length"   // Character count
	PropAttachmentCount = "attachment_count" // Number of attachments
	PropAttachmentTypes = "attachment_types" // Types of attachments
	PropHasAttachments  = "has_attachments"  // Whether message has attachments

	// Settings properties
	PropSource = "source" // Source of action

	// Tool properties
	PropToolName     = "tool_name"     // Tool name
	PropToolProvider = "tool_provider" // Tool provider

	// Plan properties
	PropAgentType  = "agent_type"  // Agent type
	PropPlanID     = "plan_id"     // Plan identifier
	PropStepAction = "step_action" // Step action
	PropProgress   = "progress"    // Progress percentage

	// Share properties
	PropShareType    = "share_type"    // "link" | "public"
	PropMessageCount = "message_count" // Number of messages in share

	// Auth properties
	PropAuthMethod = "method"      // "google" | "email" | "github"
	PropIsNewUser  = "is_new_user" // First login ever

	// Session properties
	PropSessionDuration = "session_duration_seconds" // Session duration in seconds

	// Project properties
	PropHasProject = "has_project" // Whether conversation has project
)

// ChatResponseParams contains parameters for chat_response event
type ChatResponseParams struct {
	Platform         string
	Environment      string
	RequestID        string
	UserStatus       string
	ConversationID   string
	MessageID        string
	Model            string
	Mode             string
	Provider         string
	IsStreaming      bool
	TTFRMs           int64
	LatencyMs        int64
	TokensPrompt     int
	TokensCompletion int
	TokensTotal      int
	Status           string // success | stopped | error
}

// ChatResponseProps builds properties for chat_response event
func ChatResponseProps(p ChatResponseParams) map[string]interface{} {
	props := map[string]interface{}{
		PropPlatform:         p.Platform,
		PropRequestID:        p.RequestID,
		PropUserStatus:       p.UserStatus,
		PropConversationID:   p.ConversationID,
		PropMessageID:        p.MessageID,
		PropModel:            p.Model,
		PropMode:             p.Mode,
		PropProvider:         p.Provider,
		PropIsStreaming:      p.IsStreaming,
		PropTTFRMs:           p.TTFRMs,
		PropLatencyMs:        p.LatencyMs,
		PropTokensPrompt:     p.TokensPrompt,
		PropTokensCompletion: p.TokensCompletion,
		PropTokensTotal:      p.TokensTotal,
		PropStatus:           p.Status,
	}

	if p.Environment != "" {
		props[PropEnvironment] = p.Environment
	}

	return props
}

// MessageSentParams contains parameters for message_sent event
type MessageSentParams struct {
	Platform        string
	ConversationID  string
	MessageID       string
	Model           string
	Mode            string
	UserStatus      string
	HasAttachments  bool
	AttachmentCount int
	AttachmentTypes []string
	MessageLength   int
}

// MessageSentProps builds properties for message_sent event
func MessageSentProps(p MessageSentParams) map[string]interface{} {
	return map[string]interface{}{
		PropPlatform:        p.Platform,
		PropConversationID:  p.ConversationID,
		PropMessageID:       p.MessageID,
		PropModel:           p.Model,
		PropMode:            p.Mode,
		PropUserStatus:      p.UserStatus,
		PropHasAttachments:  p.HasAttachments,
		PropAttachmentCount: p.AttachmentCount,
		PropAttachmentTypes: p.AttachmentTypes,
		PropMessageLength:   p.MessageLength,
	}
}

// ToolCalledParams contains parameters for tool_called event
type ToolCalledParams struct {
	Platform   string
	ToolName   string
	Provider   string
	DurationMs int64
	Status     string
	TokensUsed int
}

// ToolCalledProps builds properties for tool_called event
func ToolCalledProps(p ToolCalledParams) map[string]interface{} {
	return map[string]interface{}{
		PropPlatform:     p.Platform,
		PropToolName:     p.ToolName,
		PropToolProvider: p.Provider,
		PropDurationMs:   p.DurationMs,
		PropStatus:       p.Status,
		PropTokensTotal:  p.TokensUsed,
	}
}

// PlanEventParams contains parameters for plan events
type PlanEventParams struct {
	Platform   string
	PlanID     string
	AgentType  string
	DurationMs int64
	Status     string
	StepAction string
	Progress   float64
}

// PlanEventProps builds properties for plan events
func PlanEventProps(p PlanEventParams) map[string]interface{} {
	props := map[string]interface{}{
		PropPlatform:  p.Platform,
		PropPlanID:    p.PlanID,
		PropAgentType: p.AgentType,
		PropStatus:    p.Status,
	}

	if p.DurationMs > 0 {
		props[PropDurationMs] = p.DurationMs
	}
	if p.StepAction != "" {
		props[PropStepAction] = p.StepAction
	}
	if p.Progress > 0 {
		props[PropProgress] = p.Progress
	}

	return props
}

// AppOpenedParams contains parameters for app_opened event
type AppOpenedParams struct {
	Platform       string
	UserStatus     string
	ReferrerSource string
}

// AppOpenedProps builds properties for app_opened event
func AppOpenedProps(p AppOpenedParams) map[string]interface{} {
	props := map[string]interface{}{
		PropPlatform:   p.Platform,
		PropUserStatus: p.UserStatus,
	}

	if p.ReferrerSource != "" {
		props[PropReferrerSource] = p.ReferrerSource
	}

	return props
}

// ConversationCreatedParams contains parameters for conversation_created event
type ConversationCreatedParams struct {
	Platform       string
	ConversationID string
	Model          string
	Mode           string
	HasProject     bool
	UserStatus     string
}

// ConversationCreatedProps builds properties for conversation_created event
func ConversationCreatedProps(p ConversationCreatedParams) map[string]interface{} {
	return map[string]interface{}{
		PropPlatform:       p.Platform,
		PropConversationID: p.ConversationID,
		PropModel:          p.Model,
		PropMode:           p.Mode,
		PropHasProject:     p.HasProject,
		PropUserStatus:     p.UserStatus,
	}
}

// ShareCreatedParams contains parameters for share_created event
type ShareCreatedParams struct {
	Platform       string
	ConversationID string
	ShareType      string
	MessageCount   int
	UserStatus     string
}

// ShareCreatedProps builds properties for share_created event
func ShareCreatedProps(p ShareCreatedParams) map[string]interface{} {
	return map[string]interface{}{
		PropPlatform:       p.Platform,
		PropConversationID: p.ConversationID,
		PropShareType:      p.ShareType,
		PropMessageCount:   p.MessageCount,
		PropUserStatus:     p.UserStatus,
	}
}

// ModeSelectedParams contains parameters for mode_selected event
type ModeSelectedParams struct {
	Platform     string
	Mode         string
	PreviousMode string
	UserStatus   string
}

// ModeSelectedProps builds properties for mode_selected event
func ModeSelectedProps(p ModeSelectedParams) map[string]interface{} {
	props := map[string]interface{}{
		PropPlatform:   p.Platform,
		PropMode:       p.Mode,
		PropUserStatus: p.UserStatus,
	}

	if p.PreviousMode != "" {
		props[PropPreviousMode] = p.PreviousMode
	}

	return props
}

// UserLoggedInParams contains parameters for user_logged_in event
type UserLoggedInParams struct {
	Platform  string
	Method    string
	IsNewUser bool
}

// UserLoggedInProps builds properties for user_logged_in event
func UserLoggedInProps(p UserLoggedInParams) map[string]interface{} {
	return map[string]interface{}{
		PropPlatform:   p.Platform,
		PropAuthMethod: p.Method,
		PropIsNewUser:  p.IsNewUser,
	}
}

// ProviderErrorParams contains parameters for provider_error event
type ProviderErrorParams struct {
	Platform     string
	Provider     string
	Model        string
	ErrorCode    string
	ErrorMessage string
}

// ProviderErrorProps builds properties for provider_error event
func ProviderErrorProps(p ProviderErrorParams) map[string]interface{} {
	return map[string]interface{}{
		PropPlatform:     p.Platform,
		PropProvider:     p.Provider,
		PropModel:        p.Model,
		PropErrorCode:    p.ErrorCode,
		PropErrorMessage: p.ErrorMessage,
	}
}
