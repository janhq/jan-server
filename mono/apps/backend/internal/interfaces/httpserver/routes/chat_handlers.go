package routes

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"jan-server/mono/apps/backend/internal/domain/conversation"
	"jan-server/mono/apps/backend/internal/domain/model"
	"jan-server/mono/apps/backend/internal/infrastructure/config"
	"jan-server/mono/apps/backend/internal/infrastructure/database/repository"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ============================================
// Request/Response types (OpenAI-compatible)
// ============================================

type chatCompletionRequest struct {
	Model            string                   `json:"model" binding:"required"`
	Messages         []chatMessage            `json:"messages" binding:"required"`
	Temperature      *float64                 `json:"temperature"`
	TopP             *float64                 `json:"top_p"`
	N                *int                     `json:"n"`
	Stream           bool                     `json:"stream"`
	Stop             interface{}              `json:"stop"`
	MaxTokens        *int                     `json:"max_tokens"`
	PresencePenalty  *float64                 `json:"presence_penalty"`
	FrequencyPenalty *float64                 `json:"frequency_penalty"`
	LogitBias        map[string]float64       `json:"logit_bias"`
	User             string                   `json:"user"`
	Tools            []chatTool               `json:"tools"`
	ToolChoice       interface{}              `json:"tool_choice"`
	ResponseFormat   *responseFormat          `json:"response_format"`
	ConversationID   string                   `json:"conversation_id"` // Jan extension
}

type chatMessage struct {
	Role       string     `json:"role"`
	Content    interface{} `json:"content"` // string or array of content parts
	Name       *string    `json:"name,omitempty"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
	ToolCallID *string    `json:"tool_call_id,omitempty"`
}

type toolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function functionCall `json:"function"`
}

type functionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type chatTool struct {
	Type     string       `json:"type"`
	Function toolFunction `json:"function"`
}

type toolFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type responseFormat struct {
	Type string `json:"type"` // "text" or "json_object"
}

type chatCompletionResponse struct {
	ID                string              `json:"id"`
	Object            string              `json:"object"`
	Created           int64               `json:"created"`
	Model             string              `json:"model"`
	Choices           []chatChoice        `json:"choices"`
	Usage             *usageInfo          `json:"usage,omitempty"`
	SystemFingerprint string              `json:"system_fingerprint,omitempty"`
}

type chatChoice struct {
	Index        int          `json:"index"`
	Message      *chatMessage `json:"message,omitempty"`
	Delta        *chatMessage `json:"delta,omitempty"`
	FinishReason *string      `json:"finish_reason"`
	LogProbs     interface{}  `json:"logprobs"`
}

type usageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ============================================
// Chat Completions Handler
// ============================================

func chatCompletionsHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		var req chatCompletionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"message": err.Error(),
					"type":    "invalid_request_error",
				},
			})
			return
		}

		// Get model and provider
		modelRepo := repository.NewModelRepository(db)
		modelSvc := model.NewService(modelRepo)

		m, err := modelSvc.GetModelByID(c.Request.Context(), req.Model)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"message": "model not found: " + req.Model,
					"type":    "invalid_request_error",
				},
			})
			return
		}

		if !m.IsEnabled {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"message": "model is disabled: " + req.Model,
					"type":    "invalid_request_error",
				},
			})
			return
		}

		// Get provider
		provider, err := modelSvc.GetProviderByID(c.Request.Context(), m.ProviderID)
		if err != nil || !provider.IsEnabled {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"message": "provider not available",
					"type":    "invalid_request_error",
				},
			})
			return
		}

		// Save conversation and messages if conversation_id is provided
		if req.ConversationID != "" {
			convRepo := repository.NewConversationRepository(db)
			convSvc := conversation.NewService(convRepo)

			// Verify conversation exists and belongs to user
			_, err := convSvc.GetByID(c.Request.Context(), principal.ID, req.ConversationID)
			if err != nil {
				// Create conversation if it doesn't exist
				if req.ConversationID != "" {
					_, _ = convSvc.Create(c.Request.Context(), principal.ID, conversation.CreateConversationRequest{
						Title:   "New Conversation",
						ModelID: req.Model,
					})
				}
			}

			// Save user message
			for _, msg := range req.Messages {
				if msg.Role == "user" {
					content := ""
					switch v := msg.Content.(type) {
					case string:
						content = v
					}
					_ = convSvc.AddMessage(c.Request.Context(), principal.ID, req.ConversationID, &conversation.Message{
						Role:    msg.Role,
						Content: content,
						ModelID: &req.Model,
					})
				}
			}
		}

		// Forward request to provider
		if req.Stream {
			handleStreamingCompletion(c, cfg, provider, m, &req, principal.ID)
		} else {
			handleNonStreamingCompletion(c, cfg, provider, m, &req, principal.ID)
		}
	}
}

func handleNonStreamingCompletion(c *gin.Context, cfg *config.Config, provider *model.Provider, m *model.Model, req *chatCompletionRequest, userID string) {
	// Build request to provider
	providerReq := buildProviderRequest(provider, m, req)

	// Make request to provider
	client := &http.Client{Timeout: cfg.StreamTimeout}
	resp, err := client.Do(providerReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": gin.H{
				"message": "failed to connect to provider: " + err.Error(),
				"type":    "api_error",
			},
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.JSON(resp.StatusCode, json.RawMessage(body))
		return
	}

	// Parse and return response
	var completion chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&completion); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "failed to parse provider response",
				"type":    "api_error",
			},
		})
		return
	}

	c.JSON(http.StatusOK, completion)
}

func handleStreamingCompletion(c *gin.Context, cfg *config.Config, provider *model.Provider, m *model.Model, req *chatCompletionRequest, userID string) {
	// Build request to provider
	providerReq := buildProviderRequest(provider, m, req)

	// Make request to provider
	client := &http.Client{Timeout: cfg.StreamTimeout}
	resp, err := client.Do(providerReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": gin.H{
				"message": "failed to connect to provider: " + err.Error(),
				"type":    "api_error",
			},
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.JSON(resp.StatusCode, json.RawMessage(body))
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	// Stream response
	c.Stream(func(w io.Writer) bool {
		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err != io.EOF {
					fmt.Fprintf(w, "data: {\"error\": \"stream error\"}\n\n")
				}
				return false
			}

			// Forward the line
			w.Write(line)
			c.Writer.Flush()

			// Check for stream end
			if string(line) == "data: [DONE]\n" {
				return false
			}
		}
	})
}

func buildProviderRequest(provider *model.Provider, m *model.Model, req *chatCompletionRequest) *http.Request {
	// Build the request body
	body := map[string]interface{}{
		"model":    m.Name, // Use the model's name at the provider
		"messages": req.Messages,
		"stream":   req.Stream,
	}

	if req.Temperature != nil {
		body["temperature"] = *req.Temperature
	}
	if req.TopP != nil {
		body["top_p"] = *req.TopP
	}
	if req.MaxTokens != nil {
		body["max_tokens"] = *req.MaxTokens
	}
	if req.Stop != nil {
		body["stop"] = req.Stop
	}
	if len(req.Tools) > 0 {
		body["tools"] = req.Tools
	}
	if req.ToolChoice != nil {
		body["tool_choice"] = req.ToolChoice
	}
	if req.ResponseFormat != nil {
		body["response_format"] = req.ResponseFormat
	}

	jsonBody, _ := json.Marshal(body)

	// Build the HTTP request
	endpoint := provider.BaseURL
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1"
	}
	endpoint += "/chat/completions"

	httpReq, _ := http.NewRequest("POST", endpoint, bytes.NewReader(jsonBody))

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+provider.APIKey)

	return httpReq
}

// ============================================
// Response API Handlers
// ============================================

func createResponseHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Response API creates a multi-step response that can use tools
		// This is a simplified implementation
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		var req struct {
			Model        string        `json:"model" binding:"required"`
			Input        interface{}   `json:"input" binding:"required"`
			Instructions string        `json:"instructions"`
			Tools        []interface{} `json:"tools"`
			ToolChoice   interface{}   `json:"tool_choice"`
			MaxTokens    *int          `json:"max_tokens"`
			Temperature  *float64      `json:"temperature"`
			Metadata     map[string]any `json:"metadata"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Create response record
		responseID := uuid.New().String()
		now := time.Now()

		c.JSON(http.StatusCreated, gin.H{
			"id":           responseID,
			"object":       "response",
			"status":       "in_progress",
			"model":        req.Model,
			"instructions": req.Instructions,
			"created_at":   now.Unix(),
			"metadata":     req.Metadata,
		})
	}
}

func getResponseHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		id := c.Param("id")
		c.JSON(http.StatusOK, gin.H{
			"id":     id,
			"object": "response",
			"status": "completed",
		})
	}
}

func getResponseFullHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		id := c.Param("id")
		c.JSON(http.StatusOK, gin.H{
			"id":          id,
			"object":      "response",
			"status":      "completed",
			"output":      []interface{}{},
			"input_items": []interface{}{},
		})
	}
}

func deleteResponseHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"deleted": true})
	}
}

func cancelResponseHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "cancelled"})
	}
}

func retryResponseHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "retrying"})
	}
}

func getResponseInputItemsHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"input_items": []interface{}{},
		})
	}
}

// ============================================
// Plan Handlers (Response API extension)
// ============================================

func getPlanHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"plan":   nil,
			"status": "no_plan",
		})
	}
}

func getPlanProgressHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"progress": 0,
			"status":   "not_started",
		})
	}
}

func cancelPlanHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "cancelled"})
	}
}
