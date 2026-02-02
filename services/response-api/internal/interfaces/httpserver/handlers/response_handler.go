package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"

	"jan-server/services/response-api/internal/domain/llm"
	"jan-server/services/response-api/internal/domain/response"
	"jan-server/services/response-api/internal/domain/tool"
	"jan-server/services/response-api/internal/infrastructure/apikey"
	"jan-server/services/response-api/internal/interfaces/httpserver/requests"
	"jan-server/services/response-api/internal/interfaces/httpserver/responses"
)

// ResponseHandler exposes HTTP entrypoints for the Responses API.
type ResponseHandler struct {
	service        response.Service
	apiKeyProvider apikey.Provider
	log            zerolog.Logger
}

// NewResponseHandler constructs the handler.
func NewResponseHandler(service response.Service, apiKeyProvider apikey.Provider, log zerolog.Logger) *ResponseHandler {
	return &ResponseHandler{
		service:        service,
		apiKeyProvider: apiKeyProvider,
		log:            log.With().Str("handler", "response").Logger(),
	}
}

// Create handles POST /v1/responses
// @Summary Create a response
// @Description Creates a response and orchestrates MCP tool calls when required. When `stream=true`, results are streamed as SSE events instead of a single JSON payload.
// @Tags Responses
// @Accept json
// @Produce json
// @Produce text/event-stream
// @Param request body requests.CreateResponseRequest true "Create request"
// @Success 200 {object} responses.ResponsePayload "JSON response when stream=false"
// @Success 200 {string} string "SSE event stream when stream=true"
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/responses [post]
func (h *ResponseHandler) Create(c *gin.Context) {
	var req requests.CreateResponseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	requestID := strings.TrimSpace(c.GetHeader("X-Request-Id"))

	userID := req.User
	if userID == "" {
		userID = extractSubject(c)
		if userID == "" {
			userID = "guest"
		}
	}

	stream := req.Stream != nil && *req.Stream
	background := req.Background != nil && *req.Background
	store := req.Store != nil && *req.Store

	// Validate background mode constraints
	if background && !store {
		c.JSON(http.StatusBadRequest, gin.H{"error": "background mode requires store=true"})
		return
	}

	// Extract bearer token from Authorization header for generating temporary API key
	// The Authorization header should contain "Bearer <JWT>" from Kong
	authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
	var bearerToken string
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		bearerToken = strings.TrimSpace(authHeader[7:]) // Extract JWT after "Bearer "
	} else if authHeader != "" && !strings.HasPrefix(authHeader, "sk_") {
		// If no "Bearer " prefix but has value and not an API key, use as-is
		bearerToken = authHeader
	}

	// Check for X-API-Key header (user-provided API key)
	userAPIKey := strings.TrimSpace(c.GetHeader("X-API-Key"))

	// Generate temporary API key for service-to-service calls
	// This ensures downstream services (mcp-tools, llm-api) only receive API key auth
	var apiKey string
	var tempKeyID string

	if userAPIKey != "" {
		// User provided their own API key, use it directly
		apiKey = userAPIKey
		h.log.Debug().Msg("Using user-provided API key")
	} else if h.apiKeyProvider != nil && bearerToken != "" {
		// Generate temporary API key using the user's JWT
		// For background tasks, use longer TTL since execution happens later
		ttl := time.Hour
		if background {
			ttl = 24 * time.Hour // Background tasks need longer-lived keys
			h.log.Debug().Msg("Creating long-lived API key for background task")
		}

		tempKey, err := h.apiKeyProvider.CreateTemporaryKey(c.Request.Context(), bearerToken, ttl)
		if err != nil {
			h.log.Error().Err(err).Msg("Failed to create temporary API key")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "failed to authenticate: " + err.Error()})
			return
		}
		apiKey = tempKey.Key
		tempKeyID = tempKey.ID
		h.log.Debug().
			Str("key_id", tempKey.ID).
			Bool("background", background).
			Msg("Created temporary API key for request")

		// Schedule cleanup of temporary key after request completes
		// IMPORTANT: Skip immediate cleanup for background tasks since execution happens later
		if !background {
			defer h.cleanupTempKey(context.Background(), bearerToken, tempKeyID)
		}
		// Note: Background task temp keys will be cleaned up by a separate cleanup job or TTL expiration
	} else {
		h.log.Error().
			Bool("has_auth_header", authHeader != "").
			Bool("has_bearer_token", bearerToken != "").
			Bool("has_api_key_provider", h.apiKeyProvider != nil).
			Msg("No valid authentication provided")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	var apiKeyPtr *string
	if apiKey != "" {
		apiKeyPtr = &apiKey
	}

	metadata := req.Metadata
	if req.ToolChoice != nil {
		if metadata == nil {
			metadata = map[string]interface{}{}
		}
		toolName := req.ToolChoice.Function.Name
		if toolName == "" {
			toolName = req.ToolChoice.Tool
		}
		if _, ok := metadata["agent_type"]; !ok && toolName == "generate_slide" {
			metadata["agent_type"] = "slide_creator"
		}
		if metadata["agent_type"] == "slide_creator" {
			if _, ok := metadata["research_depth"]; !ok {
				metadata["research_depth"] = "standard"
			}
			if _, ok := metadata["num_slides"]; !ok {
				metadata["num_slides"] = 10
			}
			if _, ok := metadata["theme"]; !ok {
				metadata["theme"] = "modern"
			}
			if _, ok := metadata["format"]; !ok {
				metadata["format"] = "pptx"
			}
			if _, ok := metadata["body_mode"]; !ok {
				metadata["body_mode"] = "template"
			}
		}
		if req.ToolChoice.Options != nil {
			if _, ok := metadata["options"]; !ok {
				metadata["options"] = req.ToolChoice.Options
			}
		}
	}

	params := response.CreateParams{
		UserID:               userID,
		Model:                req.Model,
		Input:                req.Input,
		RequestID:            requestID,
		SystemPrompt:         req.SystemPrompt,
		Temperature:          req.Temperature,
		MaxTokens:            req.MaxTokens,
		Stream:               stream,
		Background:           background,
		Store:                store,
		APIKey:               apiKeyPtr,
		ToolChoice:           mapToolChoice(req.ToolChoice),
		Tools:                mapTools(req.Tools),
		PreviousResponseID:   req.PreviousResponseID,
		ConversationID:       req.Conversation,
		ParentConversationID: req.ParentConversationID,
		Metadata:             metadata,
	}

	authCtx := llm.ContextWithAuthToken(c.Request.Context(), apiKey)
	c.Request = c.Request.WithContext(authCtx)

	if stream {
		h.streamResponse(c, params)
		return
	}

	resp, err := h.service.Create(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, responses.FromDomain(resp))
}

// Get handles GET /v1/responses/:id
// @Summary Get a response by ID
// @Tags Responses
// @Produce json
// @Param response_id path string true "Response ID"
// @Success 200 {object} responses.ResponsePayload
// @Failure 404 {object} map[string]string
// @Router /v1/responses/{response_id} [get]
func (h *ResponseHandler) Get(c *gin.Context) {
	id := c.Param("response_id")
	resp, err := h.service.GetByPublicID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, responses.FromDomain(resp))
}

// GetFull handles GET /v1/responses/:id/full
// @Summary Get a response with full plan details
// @Description Returns the response along with its execution plan, tasks, and steps in a single call
// @Tags Responses
// @Produce json
// @Param response_id path string true "Response ID"
// @Success 200 {object} responses.FullResponsePayload
// @Failure 404 {object} map[string]string
// @Router /v1/responses/{response_id}/full [get]
func (h *ResponseHandler) GetFull(c *gin.Context) {
	id := c.Param("response_id")
	rwp, err := h.service.GetWithPlanDetails(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, responses.FromDomainWithPlan(rwp, responses.MapPlanDetailFromInterface))
}

// Cancel handles POST /v1/responses/:id/cancel
// @Summary Cancel a response
// @Tags Responses
// @Produce json
// @Param response_id path string true "Response ID"
// @Success 200 {object} responses.ResponsePayload
// @Failure 400 {object} map[string]string
// @Router /v1/responses/{response_id}/cancel [post]
func (h *ResponseHandler) Cancel(c *gin.Context) {
	id := c.Param("response_id")
	resp, err := h.service.Cancel(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, responses.FromDomain(resp))
}

// Retry handles POST /v1/responses/:id/retry
// @Summary Retry the last failed plan step for a response
// @Tags Responses
// @Produce json
// @Param response_id path string true "Response ID"
// @Success 200 {object} responses.ResponsePayload
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/responses/{response_id}/retry [post]
func (h *ResponseHandler) Retry(c *gin.Context) {
	id := c.Param("response_id")
	resp, err := h.service.Retry(c.Request.Context(), id)
	if err != nil {
		h.log.Error().Err(err).Str("response_id", id).Msg("response retry failed")
		responses.HandleError(c, err, "failed to retry response")
		return
	}
	c.JSON(http.StatusOK, responses.FromDomain(resp))
}

// Delete handles DELETE /v1/responses/:id
// @Summary Delete/Cancel a response
// @Tags Responses
// @Produce json
// @Param response_id path string true "Response ID"
// @Success 200 {object} responses.ResponsePayload
// @Failure 400 {object} map[string]string
// @Router /v1/responses/{response_id} [delete]
func (h *ResponseHandler) Delete(c *gin.Context) {
	h.Cancel(c)
}

// ListInputItems handles GET /v1/responses/:id/input_items
// @Summary List conversation input items
// @Tags Responses
// @Produce json
// @Param response_id path string true "Response ID"
// @Success 200 {object} responses.ConversationItemsResponse
// @Failure 400 {object} map[string]string
// @Router /v1/responses/{response_id}/input_items [get]
func (h *ResponseHandler) ListInputItems(c *gin.Context) {
	id := c.Param("response_id")
	items, err := h.service.ListConversationItems(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Return items array directly, wrapped in data object for consistency
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *ResponseHandler) streamResponse(c *gin.Context, params response.CreateParams) {
	writer := c.Writer
	flusher, ok := writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	observer := newSSEObserver(writer, flusher, h.log)
	params.StreamObserver = observer

	resp, err := h.service.Create(c.Request.Context(), params)
	if err != nil {
		observer.SendError(err)
		c.Status(http.StatusInternalServerError)
		return
	}
	observer.SendCompleted(resp)
}

func extractSubject(c *gin.Context) string {
	tokenValue, exists := c.Get("auth_token")
	if !exists {
		return ""
	}
	token, ok := tokenValue.(*jwt.Token)
	if !ok {
		return ""
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		if sub, ok := claims["sub"].(string); ok {
			return sub
		}
	}
	return ""
}

func mapTools(tools []requests.ToolDefinition) []llm.ToolDefinition {
	if len(tools) == 0 {
		return nil
	}
	result := make([]llm.ToolDefinition, 0, len(tools))
	for _, t := range tools {
		result = append(result, llm.ToolDefinition{
			Type: t.Type,
			Function: llm.ToolFunctionSchema{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  t.Function.Parameters,
			},
		})
	}
	return result
}

func mapToolChoice(choice *requests.ToolChoice) *llm.ToolChoice {
	if choice == nil {
		return nil
	}
	toolName := choice.Function.Name
	if toolName == "" {
		toolName = choice.Tool
	}
	return &llm.ToolChoice{
		Type: choice.Type,
		Function: struct {
			Name string `json:"name"`
		}{
			Name: toolName,
		},
	}
}

type sseObserver struct {
	writer     http.ResponseWriter
	flusher    http.Flusher
	log        zerolog.Logger
	mu         sync.Mutex
	responseID string
}

func newSSEObserver(w http.ResponseWriter, flusher http.Flusher, log zerolog.Logger) *sseObserver {
	return &sseObserver{
		writer:  w,
		flusher: flusher,
		log:     log,
	}
}

func (o *sseObserver) OnResponseCreated(resp *response.Response) {
	o.responseID = resp.PublicID
	o.sendEvent("response.created", responses.FromDomain(resp))
}

func (o *sseObserver) OnDelta(delta llm.ChatCompletionDelta) {
	text := extractDeltaText(delta)
	if text == "" {
		return
	}
	payload := map[string]interface{}{
		"id":    o.responseID,
		"delta": text,
	}
	o.sendEvent("response.output_text.delta", payload)
}

func (o *sseObserver) OnToolCall(call tool.Call) {
	payload := map[string]interface{}{
		"id":   o.responseID,
		"call": call,
	}
	o.sendEvent("response.tool_call", payload)
}

func (o *sseObserver) OnToolResult(callID string, result *tool.Result) {
	payload := map[string]interface{}{
		"id":      o.responseID,
		"call_id": callID,
		"result":  result,
	}
	o.sendEvent("response.tool_result", payload)
}

func (o *sseObserver) SendCompleted(resp *response.Response) {
	o.sendEvent("response.completed", responses.FromDomain(resp))
}

func (o *sseObserver) SendError(err error) {
	o.sendEvent("response.error", map[string]string{
		"message": err.Error(),
	})
}

func (o *sseObserver) sendEvent(name string, payload interface{}) {
	o.mu.Lock()
	defer o.mu.Unlock()

	data, err := json.Marshal(payload)
	if err != nil {
		o.log.Error().Err(err).Msg("marshal SSE payload")
		return
	}

	fmt.Fprintf(o.writer, "event: %s\n", name)
	fmt.Fprintf(o.writer, "data: %s\n\n", data)
	o.flusher.Flush()
}

func extractDeltaText(delta llm.ChatCompletionDelta) string {
	for _, choice := range delta.Choices {
		if choice.Delta.Content == nil {
			continue
		}
		if text := normalizeContent(choice.Delta.Content); text != "" {
			return text
		}
	}
	return ""
}

func normalizeContent(content interface{}) string {
	switch v := content.(type) {
	case string:
		return v
	case []interface{}:
		builder := strings.Builder{}
		for _, item := range v {
			builder.WriteString(normalizeContent(item))
		}
		return builder.String()
	case map[string]interface{}:
		if text, ok := v["text"].(string); ok {
			return text
		}
	}
	return ""
}

var _ response.StreamObserver = (*sseObserver)(nil)

// cleanupTempKey deletes a temporary API key after request completion.
func (h *ResponseHandler) cleanupTempKey(ctx context.Context, bearerToken, keyID string) {
	if h.apiKeyProvider == nil || keyID == "" {
		return
	}

	// Use a short timeout for cleanup
	cleanupCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := h.apiKeyProvider.DeleteKey(cleanupCtx, bearerToken, keyID); err != nil {
		h.log.Warn().Err(err).Str("key_id", keyID).Msg("Failed to cleanup temporary API key")
	} else {
		h.log.Debug().Str("key_id", keyID).Msg("Cleaned up temporary API key")
	}
}
