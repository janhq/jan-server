package mcp

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// ToolTrackingContextKey is the context key for tool tracking data
type ToolTrackingContextKey struct{}

// ToolTrackingContext holds the tracking information from headers.
//
// Header requirements vary by use case:
//
// 1. TOOL TRACKING (callback to LLM-API to record results):
//   - Requires ALL: X-Conversation-ID, X-Tool-Call-ID, Authorization
//   - Must NOT be from response-api (X-JAN-SRC != RESPONSE)
//   - Used by direct client calls to record tool results in conversation history
//
// 2. SANDBOX TOOLS (workspace/conversation association):
//   - Requires ONLY: X-Conversation-ID, Authorization
//   - X-Tool-Call-ID is optional (only for logging)
//   - Used by response-api agents to manage sandbox workspaces
//   - Response-api sets X-JAN-SRC: RESPONSE to disable tracking
type ToolTrackingContext struct {
	ConversationID  string
	ToolCallID      string
	AuthToken       string
	Enabled         bool // True only when ALL headers present AND not from response-api
	FromResponseAPI bool // True if request comes from response-api (X-JAN-SRC: RESPONSE)
}

// ExtractToolTracking extracts tracking headers and injects into context.
// All headers are extracted regardless of completeness - individual tools
// decide which headers they require.
//
// Headers:
//   - X-Conversation-ID: The conversation ID (required for sandbox tools)
//   - X-Tool-Call-ID: The tool call ID from LLM (required for tracking callback only)
//   - Authorization: Bearer token for user authentication
//   - X-JAN-SRC: Set to "RESPONSE" by response-api to disable tracking
func ExtractToolTracking() gin.HandlerFunc {
	return func(reqCtx *gin.Context) {
		conversationID := reqCtx.GetHeader("X-Conversation-ID")
		toolCallID := reqCtx.GetHeader("X-Tool-Call-ID")
		authToken := reqCtx.GetHeader("Authorization")
		janSrc := reqCtx.GetHeader("X-JAN-SRC")

		// Check if request comes from response-api
		fromResponseAPI := janSrc == "RESPONSE"

		// Disable tracking for requests from response-api
		// Response-api handles its own conversation tracking internally
		enabled := conversationID != "" && toolCallID != "" && authToken != "" && !fromResponseAPI

		tracking := ToolTrackingContext{
			ConversationID:  conversationID,
			ToolCallID:      toolCallID,
			AuthToken:       authToken,
			Enabled:         enabled,
			FromResponseAPI: fromResponseAPI,
		}

		if tracking.Enabled {
			log.Debug().
				Str("conv_id", conversationID).
				Str("call_id", toolCallID).
				Bool("tracking_enabled", true).
				Msg("Tool tracking enabled for request")
		} else if fromResponseAPI {
			log.Debug().
				Bool("from_response_api", true).
				Msg("Tool tracking disabled for response-api request")
		}

		// Inject tracking context into request context
		ctx := context.WithValue(reqCtx.Request.Context(), ToolTrackingContextKey{}, tracking)
		reqCtx.Request = reqCtx.Request.WithContext(ctx)

		reqCtx.Next()
	}
}

// GetToolTracking retrieves tracking context from the request context
// Returns the tracking context and whether tracking is enabled
func GetToolTracking(ctx context.Context) (ToolTrackingContext, bool) {
	if val := ctx.Value(ToolTrackingContextKey{}); val != nil {
		if tracking, ok := val.(ToolTrackingContext); ok {
			return tracking, tracking.Enabled
		}
	}
	return ToolTrackingContext{}, false
}

// GetToolTrackingFromGin retrieves tracking context from a Gin context
func GetToolTrackingFromGin(reqCtx *gin.Context) (ToolTrackingContext, bool) {
	return GetToolTracking(reqCtx.Request.Context())
}

// GetConversationID extracts just the conversation_id from context.
// Unlike GetToolTracking, this doesn't require full tracking to be enabled.
// Use this for sandbox tools that only need X-Conversation-ID header.
func GetConversationID(ctx context.Context) string {
	if val := ctx.Value(ToolTrackingContextKey{}); val != nil {
		if tracking, ok := val.(ToolTrackingContext); ok {
			return tracking.ConversationID
		}
	}
	return ""
}
