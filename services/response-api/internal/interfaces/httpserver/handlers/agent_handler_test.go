package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"jan-server/services/response-api/internal/domain/agent"
)

func TestAgentHandler_List(t *testing.T) {
	gin.SetMode(gin.TestMode)
	registry := agent.NewRegistry()
	handler := NewAgentHandler(registry, zerolog.Nop())

	router := gin.New()
	router.GET("/v1/agents", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/v1/agents", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response struct {
		Agents []struct {
			Type        string   `json:"type"`
			Name        string   `json:"name"`
			Description string   `json:"description"`
			Keywords    []string `json:"keywords"`
			Enabled     bool     `json:"enabled"`
		} `json:"agents"`
		Total int `json:"total"`
	}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Should return default agents (deep_research, slide_creator, doc_generator, pdf_generator, spreadsheet_generator)
	assert.GreaterOrEqual(t, response.Total, 2)
	assert.GreaterOrEqual(t, len(response.Agents), 2)

	// Check that expected agents are present
	agentTypes := make(map[string]bool)
	for _, a := range response.Agents {
		agentTypes[a.Type] = true
		assert.NotEmpty(t, a.Name)
		assert.NotEmpty(t, a.Description)
		assert.NotEmpty(t, a.Keywords)
	}

	assert.True(t, agentTypes["deep_research"], "deep_research agent should be present")
	assert.True(t, agentTypes["slide_creator"], "slide_creator agent should be present")
}

func TestAgentHandler_Get(t *testing.T) {
	gin.SetMode(gin.TestMode)
	registry := agent.NewRegistry()
	handler := NewAgentHandler(registry, zerolog.Nop())

	router := gin.New()
	router.GET("/v1/agents/:type", handler.Get)

	tests := []struct {
		name           string
		agentType      string
		expectedStatus int
		checkFields    bool
	}{
		{
			name:           "get deep_research agent",
			agentType:      "deep_research",
			expectedStatus: http.StatusOK,
			checkFields:    true,
		},
		{
			name:           "get slide_creator agent",
			agentType:      "slide_creator",
			expectedStatus: http.StatusOK,
			checkFields:    true,
		},
		{
			name:           "get non-existent agent",
			agentType:      "non_existent",
			expectedStatus: http.StatusNotFound,
			checkFields:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/v1/agents/"+tt.agentType, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)

			if tt.checkFields {
				var response struct {
					Type         string      `json:"type"`
					Name         string      `json:"name"`
					Description  string      `json:"description"`
					Keywords     []string    `json:"keywords"`
					Capabilities []string    `json:"capabilities"`
					InputSchema  interface{} `json:"inputSchema"`
					Enabled      bool        `json:"enabled"`
				}
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, tt.agentType, response.Type)
				assert.NotEmpty(t, response.Name)
				assert.NotEmpty(t, response.Description)
				assert.NotEmpty(t, response.Keywords)
				assert.NotEmpty(t, response.Capabilities)
				assert.NotNil(t, response.InputSchema)
			}
		})
	}
}

func TestAgentHandler_GetSchema(t *testing.T) {
	gin.SetMode(gin.TestMode)
	registry := agent.NewRegistry()
	handler := NewAgentHandler(registry, zerolog.Nop())

	router := gin.New()
	router.GET("/v1/agents/:type/schema", handler.GetSchema)

	tests := []struct {
		name           string
		agentType      string
		expectedStatus int
	}{
		{
			name:           "get deep_research schema",
			agentType:      "deep_research",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "get slide_creator schema",
			agentType:      "slide_creator",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "get non-existent agent schema",
			agentType:      "non_existent",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/v1/agents/"+tt.agentType+"/schema", nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)

			if tt.expectedStatus == http.StatusOK {
				var response struct {
					Type        string `json:"type"`
					Name        string `json:"name"`
					Description string `json:"description"`
					InputSchema struct {
						Type       string                 `json:"type"`
						Properties map[string]interface{} `json:"properties"`
						Required   []string               `json:"required"`
					} `json:"inputSchema"`
				}
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, tt.agentType, response.Type)
				assert.NotEmpty(t, response.InputSchema.Type)
				assert.NotEmpty(t, response.InputSchema.Properties)
				assert.Contains(t, response.InputSchema.Required, "prompt")

				// Check that prompt property exists
				_, hasPrompt := response.InputSchema.Properties["prompt"]
				assert.True(t, hasPrompt, "InputSchema should have prompt property")
			}
		})
	}
}

func TestAgentHandler_GetCapabilities(t *testing.T) {
	gin.SetMode(gin.TestMode)
	registry := agent.NewRegistry()
	handler := NewAgentHandler(registry, zerolog.Nop())

	router := gin.New()
	router.GET("/v1/agents/capabilities", handler.GetCapabilities)

	req := httptest.NewRequest(http.MethodGet, "/v1/agents/capabilities", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response struct {
		SelectionPrompt string `json:"selection_prompt"`
		Agents          []struct {
			Type     string   `json:"type"`
			UseWhen  string   `json:"use_when"`
			Keywords []string `json:"keywords"`
		} `json:"agents"`
	}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotEmpty(t, response.SelectionPrompt)
	assert.GreaterOrEqual(t, len(response.Agents), 2)

	// Check that each agent has the required fields for LLM selection
	for _, agent := range response.Agents {
		assert.NotEmpty(t, agent.Type)
		assert.NotEmpty(t, agent.UseWhen)
		assert.NotEmpty(t, agent.Keywords)
	}

	// Check that expected agents are present
	agentTypes := make(map[string]bool)
	for _, a := range response.Agents {
		agentTypes[a.Type] = true
	}

	assert.True(t, agentTypes["deep_research"], "deep_research should be in capabilities")
	assert.True(t, agentTypes["slide_creator"], "slide_creator should be in capabilities")
}
