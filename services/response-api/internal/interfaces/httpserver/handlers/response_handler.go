package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"

	"jan-server/services/response-api/internal/domain/llm"
	"jan-server/services/response-api/internal/domain/response"
	"jan-server/services/response-api/internal/domain/tool"
	"jan-server/services/response-api/internal/interfaces/httpserver/dto"
)

// ResponseHandler exposes HTTP entrypoints for the Responses API.
type ResponseHandler struct {
	service response.Service
	log     zerolog.Logger
}

// NewResponseHandler constructs the handler.
func NewResponseHandler(service response.Service, log zerolog.Logger) *ResponseHandler {
	return &ResponseHandler{
		service: service,
		log:     log.With().Str("handler", "response").Logger(),
	}
}

// Create handles POST /v1/responses
// @Summary Create a response
// @Description Creates a response and orchestrates MCP tool calls when required.
// @Tags Responses
// @Accept json
// @Produce json
// @Param request body dto.CreateResponseRequest true "Create request"
// @Success 200 {object} dto.ResponsePayload
// @Failure 400 {object} map[string]string
// @Router /v1/responses [post]
func (h *ResponseHandler) Create(c *gin.Context) {
	var req dto.CreateResponseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := req.User
	if userID == "" {
		userID = extractSubject(c)
		if userID == "" {
			userID = "guest"
		}
	}

	stream := req.Stream != nil && *req.Stream

	params := response.CreateParams{
		UserID:             userID,
		Model:              req.Model,
		Input:              req.Input,
		SystemPrompt:       req.SystemPrompt,
		Temperature:        req.Temperature,
		MaxTokens:          req.MaxTokens,
		Stream:             stream,
		ToolChoice:         mapToolChoice(req.ToolChoice),
		Tools:              mapTools(req.Tools),
		PreviousResponseID: req.PreviousResponseID,
		ConversationID:     req.Conversation,
		Metadata:           req.Metadata,
	}

	if stream {
		h.streamResponse(c, params)
		return
	}

	resp, err := h.service.Create(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.FromDomain(resp))
}

// Get handles GET /v1/responses/:id
// @Summary Get a response by ID
// @Tags Responses
// @Produce json
// @Param response_id path string true "Response ID"
// @Success 200 {object} dto.ResponsePayload
// @Failure 404 {object} map[string]string
// @Router /v1/responses/{response_id} [get]
func (h *ResponseHandler) Get(c *gin.Context) {
	id := c.Param("response_id")
	resp, err := h.service.GetByPublicID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, dto.FromDomain(resp))
}

// Cancel handles POST /v1/responses/:id/cancel
// @Summary Cancel a response
// @Tags Responses
// @Produce json
// @Param response_id path string true "Response ID"
// @Success 200 {object} dto.ResponsePayload
// @Router /v1/responses/{response_id}/cancel [post]
func (h *ResponseHandler) Cancel(c *gin.Context) {
	id := c.Param("response_id")
	resp, err := h.service.Cancel(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, dto.FromDomain(resp))
}

// Delete handles DELETE /v1/responses/:id
// @Summary Delete/Cancel a response
// @Tags Responses
// @Produce json
// @Param response_id path string true "Response ID"
// @Success 200 {object} dto.ResponsePayload
// @Router /v1/responses/{response_id} [delete]
func (h *ResponseHandler) Delete(c *gin.Context) {
	h.Cancel(c)
}

// ListInputItems handles GET /v1/responses/:id/input_items
// @Summary List conversation input items
// @Tags Responses
// @Produce json
// @Param response_id path string true "Response ID"
// @Success 200 {object} map[string]interface{}
// @Router /v1/responses/{response_id}/input_items [get]
func (h *ResponseHandler) ListInputItems(c *gin.Context) {
	id := c.Param("response_id")
	items, err := h.service.ListConversationItems(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
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

func mapTools(tools []dto.ToolDefinition) []llm.ToolDefinition {
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

func mapToolChoice(choice *dto.ToolChoice) *llm.ToolChoice {
	if choice == nil {
		return nil
	}
	return &llm.ToolChoice{
		Type: choice.Type,
		Function: struct {
			Name string `json:"name"`
		}{
			Name: choice.Function.Name,
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
	o.sendEvent("response.created", dto.FromDomain(resp))
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
	o.sendEvent("response.completed", dto.FromDomain(resp))
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
