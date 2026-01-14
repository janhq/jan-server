package responseapi_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AgentListResponse represents the response from GET /v1/agents
type AgentListResponse struct {
	Agents []AgentResponse `json:"agents"`
	Total  int             `json:"total"`
}

// AgentResponse represents an agent in API responses
type AgentResponse struct {
	Type              string   `json:"type"`
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	Keywords          []string `json:"keywords"`
	Capabilities      []string `json:"capabilities"`
	OutputFormats     []string `json:"output_formats"`
	EstimatedDuration string   `json:"estimated_duration"`
	Enabled           bool     `json:"enabled"`
}

// AgentDetailResponse represents full agent details
type AgentDetailResponse struct {
	AgentResponse
	InputSchema  interface{} `json:"inputSchema,omitempty"`
	OutputSchema interface{} `json:"outputSchema,omitempty"`
	Examples     interface{} `json:"examples,omitempty"`
}

// AgentSchemaResponse represents an agent's schema information
type AgentSchemaResponse struct {
	Type         string      `json:"type"`
	Name         string      `json:"name"`
	Description  string      `json:"description"`
	InputSchema  interface{} `json:"inputSchema"`
	OutputSchema interface{} `json:"outputSchema,omitempty"`
	Examples     interface{} `json:"examples,omitempty"`
}

// AgentCapabilitiesResponse represents condensed agent capabilities
type AgentCapabilitiesResponse struct {
	SelectionPrompt string                    `json:"selection_prompt"`
	Agents          []AgentCapabilityResponse `json:"agents"`
}

// AgentCapabilityResponse represents a single agent capability
type AgentCapabilityResponse struct {
	Type     string   `json:"type"`
	UseWhen  string   `json:"use_when"`
	Keywords []string `json:"keywords"`
}

func TestAgentDiscovery_ListAgents(t *testing.T) {
	skipIfNoAPI(t)

	resp, body := makeRequest(t, http.MethodGet, "/v1/agents", nil)
	assertStatus(t, resp, http.StatusOK, body)

	var response AgentListResponse
	err := json.Unmarshal(body, &response)
	require.NoError(t, err)

	// Should have at least 2 agents (deep_research, slide_generator)
	assert.GreaterOrEqual(t, response.Total, 2, "Should have at least 2 agents")
	assert.GreaterOrEqual(t, len(response.Agents), 2, "Should have at least 2 agents in list")

	// Check that expected agents are present
	agentTypes := make(map[string]bool)
	for _, agent := range response.Agents {
		agentTypes[agent.Type] = true

		// Validate required fields
		assert.NotEmpty(t, agent.Type, "Agent type should not be empty")
		assert.NotEmpty(t, agent.Name, "Agent name should not be empty")
		assert.NotEmpty(t, agent.Description, "Agent description should not be empty")
		assert.NotEmpty(t, agent.Keywords, "Agent keywords should not be empty")
		assert.NotEmpty(t, agent.Capabilities, "Agent capabilities should not be empty")
	}

	assert.True(t, agentTypes["deep_research"], "deep_research agent should be present")
	assert.True(t, agentTypes["slide_generator"], "slide_generator agent should be present")
}

func TestAgentDiscovery_GetAgent(t *testing.T) {
	skipIfNoAPI(t)

	tests := []struct {
		name           string
		agentType      string
		expectedStatus int
	}{
		{
			name:           "get deep_research agent",
			agentType:      "deep_research",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "get slide_generator agent",
			agentType:      "slide_generator",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "get non-existent agent",
			agentType:      "non_existent_agent",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, body := makeRequest(t, http.MethodGet, "/v1/agents/"+tt.agentType, nil)
			assertStatus(t, resp, tt.expectedStatus, body)

			if tt.expectedStatus == http.StatusOK {
				var response AgentDetailResponse
				err := json.Unmarshal(body, &response)
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

func TestAgentDiscovery_GetAgentSchema(t *testing.T) {
	skipIfNoAPI(t)

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
			name:           "get slide_generator schema",
			agentType:      "slide_generator",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, body := makeRequest(t, http.MethodGet, "/v1/agents/"+tt.agentType+"/schema", nil)
			assertStatus(t, resp, tt.expectedStatus, body)

			var response AgentSchemaResponse
			err := json.Unmarshal(body, &response)
			require.NoError(t, err)

			assert.Equal(t, tt.agentType, response.Type)
			assert.NotEmpty(t, response.Name)
			assert.NotEmpty(t, response.Description)
			assert.NotNil(t, response.InputSchema)

			// Validate InputSchema structure
			inputSchema, ok := response.InputSchema.(map[string]interface{})
			require.True(t, ok, "InputSchema should be an object")
			assert.Equal(t, "object", inputSchema["type"])

			properties, ok := inputSchema["properties"].(map[string]interface{})
			require.True(t, ok, "InputSchema should have properties")
			_, hasPrompt := properties["prompt"]
			assert.True(t, hasPrompt, "InputSchema should have prompt property")

			required, ok := inputSchema["required"].([]interface{})
			require.True(t, ok, "InputSchema should have required array")
			assert.Contains(t, required, "prompt", "prompt should be required")
		})
	}
}

func TestAgentDiscovery_GetCapabilities(t *testing.T) {
	skipIfNoAPI(t)

	resp, body := makeRequest(t, http.MethodGet, "/v1/agents/capabilities", nil)
	assertStatus(t, resp, http.StatusOK, body)

	var response AgentCapabilitiesResponse
	err := json.Unmarshal(body, &response)
	require.NoError(t, err)

	// Should have a selection prompt
	assert.NotEmpty(t, response.SelectionPrompt, "Should have a selection prompt")
	assert.Contains(t, response.SelectionPrompt, "select", "Selection prompt should mention selecting")

	// Should have at least 2 agents
	assert.GreaterOrEqual(t, len(response.Agents), 2, "Should have at least 2 agents")

	// Check that each agent has the required fields for LLM selection
	agentTypes := make(map[string]bool)
	for _, agent := range response.Agents {
		agentTypes[agent.Type] = true

		assert.NotEmpty(t, agent.Type, "Agent type should not be empty")
		assert.NotEmpty(t, agent.UseWhen, "Agent use_when should not be empty")
		assert.NotEmpty(t, agent.Keywords, "Agent keywords should not be empty")
	}

	assert.True(t, agentTypes["deep_research"], "deep_research should be in capabilities")
	assert.True(t, agentTypes["slide_generator"], "slide_generator should be in capabilities")
}

func TestAgentDiscovery_DeepResearchSchema(t *testing.T) {
	skipIfNoAPI(t)

	resp, body := makeRequest(t, http.MethodGet, "/v1/agents/deep_research/schema", nil)
	assertStatus(t, resp, http.StatusOK, body)

	var response AgentSchemaResponse
	err := json.Unmarshal(body, &response)
	require.NoError(t, err)

	inputSchema, ok := response.InputSchema.(map[string]interface{})
	require.True(t, ok)

	properties, ok := inputSchema["properties"].(map[string]interface{})
	require.True(t, ok)

	// Check deep_research specific properties
	expectedProps := []string{"prompt", "research_depth", "format", "max_sources", "include_citations"}
	for _, prop := range expectedProps {
		_, exists := properties[prop]
		assert.True(t, exists, "deep_research schema should have %s property", prop)
	}

	// Check research_depth enum
	researchDepth, ok := properties["research_depth"].(map[string]interface{})
	if ok {
		enum, hasEnum := researchDepth["enum"].([]interface{})
		if hasEnum {
			assert.Contains(t, enum, "minimal")
			assert.Contains(t, enum, "standard")
			assert.Contains(t, enum, "deep")
		}
	}
}

func TestAgentDiscovery_SlideGeneratorSchema(t *testing.T) {
	skipIfNoAPI(t)

	resp, body := makeRequest(t, http.MethodGet, "/v1/agents/slide_generator/schema", nil)
	assertStatus(t, resp, http.StatusOK, body)

	var response AgentSchemaResponse
	err := json.Unmarshal(body, &response)
	require.NoError(t, err)

	inputSchema, ok := response.InputSchema.(map[string]interface{})
	require.True(t, ok)

	properties, ok := inputSchema["properties"].(map[string]interface{})
	require.True(t, ok)

	// Check slide_generator specific properties
	expectedProps := []string{"prompt", "num_slides", "theme", "format", "research_depth"}
	for _, prop := range expectedProps {
		_, exists := properties[prop]
		assert.True(t, exists, "slide_generator schema should have %s property", prop)
	}

	// Check theme enum
	theme, ok := properties["theme"].(map[string]interface{})
	if ok {
		enum, hasEnum := theme["enum"].([]interface{})
		if hasEnum {
			assert.Contains(t, enum, "modern")
			assert.Contains(t, enum, "corporate")
			assert.Contains(t, enum, "minimal")
		}
	}

	// Check format enum
	format, ok := properties["format"].(map[string]interface{})
	if ok {
		enum, hasEnum := format["enum"].([]interface{})
		if hasEnum {
			assert.Contains(t, enum, "pptx")
			assert.Contains(t, enum, "pdf")
		}
	}
}
