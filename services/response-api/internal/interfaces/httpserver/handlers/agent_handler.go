package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/interfaces/httpserver/responses"
)

// AgentHandler exposes HTTP entrypoints for the Agent Discovery API.
type AgentHandler struct {
	registry agent.Registry
	log      zerolog.Logger
}

// NewAgentHandler constructs the handler.
func NewAgentHandler(registry agent.Registry, log zerolog.Logger) *AgentHandler {
	return &AgentHandler{
		registry: registry,
		log:      log.With().Str("handler", "agent").Logger(),
	}
}

// List handles GET /v1/agents
// @Summary List available agents
// @Description Returns all available agent types with metadata
// @Tags Agents
// @Produce json
// @Success 200 {object} responses.AgentListResponse
// @Router /v1/agents [get]
func (h *AgentHandler) List(c *gin.Context) {
	agents := h.registry.GetAllAgents()
	response := responses.MapAgentListToResponse(agents)
	c.JSON(http.StatusOK, response)
}

// Get handles GET /v1/agents/:type
// @Summary Get agent details
// @Description Returns detailed information about a specific agent type
// @Tags Agents
// @Produce json
// @Param type path string true "Agent type (e.g., deep_research, slide_generator)"
// @Success 200 {object} responses.AgentDetailResponse
// @Failure 404 {object} map[string]string
// @Router /v1/agents/{type} [get]
func (h *AgentHandler) Get(c *gin.Context) {
	agentType := c.Param("type")

	detail := h.registry.GetAgent(agentType)
	if detail == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "agent_not_found",
			"message": "Agent type not found: " + agentType,
		})
		return
	}

	response := responses.MapAgentDetailToResponse(detail)
	c.JSON(http.StatusOK, response)
}

// GetSchema handles GET /v1/agents/:type/schema
// @Summary Get agent schema
// @Description Returns the input/output schemas for a specific agent type
// @Tags Agents
// @Produce json
// @Param type path string true "Agent type (e.g., deep_research, slide_generator)"
// @Success 200 {object} responses.AgentSchemaResponse
// @Failure 404 {object} map[string]string
// @Router /v1/agents/{type}/schema [get]
func (h *AgentHandler) GetSchema(c *gin.Context) {
	agentType := c.Param("type")

	detail := h.registry.GetAgent(agentType)
	if detail == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "agent_not_found",
			"message": "Agent type not found: " + agentType,
		})
		return
	}

	response := responses.AgentSchemaResponse{
		Type:        detail.Type,
		Name:        detail.Name,
		Description: detail.Description,
	}

	if detail.InputSchema != nil {
		response.InputSchema = detail.InputSchema
	}
	if detail.OutputSchema != nil {
		response.OutputSchema = detail.OutputSchema
	}
	if len(detail.Examples) > 0 {
		examples := make([]responses.AgentExampleResponse, 0, len(detail.Examples))
		for _, ex := range detail.Examples {
			examples = append(examples, responses.AgentExampleResponse{
				Input:       ex.Input,
				Description: ex.Description,
			})
		}
		response.Examples = examples
	}

	c.JSON(http.StatusOK, response)
}

// GetCapabilities handles GET /v1/agents/capabilities
// @Summary Get agent capabilities for LLM selection
// @Description Returns condensed agent capabilities optimized for LLM tool selection
// @Tags Agents
// @Produce json
// @Success 200 {object} responses.AgentCapabilitiesResponse
// @Router /v1/agents/capabilities [get]
func (h *AgentHandler) GetCapabilities(c *gin.Context) {
	capabilities := h.registry.GetAgentCapabilitiesForLLM()
	response := responses.MapAgentCapabilitiesToResponse(capabilities)
	c.JSON(http.StatusOK, response)
}
