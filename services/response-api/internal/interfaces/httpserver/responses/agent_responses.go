package responses

import (
	"jan-server/services/response-api/internal/domain/agent"
)

// AgentListResponse represents the list of available agents.
type AgentListResponse struct {
	Agents []AgentResponse `json:"agents"`
	Total  int             `json:"total"`
}

// AgentResponse represents an agent in API responses.
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

// AgentDetailResponse represents full agent details.
type AgentDetailResponse struct {
	AgentResponse
	InputSchema  interface{}            `json:"inputSchema,omitempty"`
	OutputSchema interface{}            `json:"outputSchema,omitempty"`
	Examples     []AgentExampleResponse `json:"examples,omitempty"`
}

// AgentExampleResponse represents an agent usage example.
type AgentExampleResponse struct {
	Input       interface{} `json:"input"`
	Description string      `json:"description"`
}

// AgentSchemaResponse represents an agent's schema information.
type AgentSchemaResponse struct {
	Type         string      `json:"type"`
	Name         string      `json:"name"`
	Description  string      `json:"description"`
	InputSchema  interface{} `json:"inputSchema"`
	OutputSchema interface{} `json:"outputSchema,omitempty"`
	Examples     interface{} `json:"examples,omitempty"`
}

// AgentCapabilitiesResponse represents condensed agent capabilities for LLM selection.
type AgentCapabilitiesResponse struct {
	SelectionPrompt string                    `json:"selection_prompt"`
	Agents          []AgentCapabilityResponse `json:"agents"`
}

// AgentCapabilityResponse represents a single agent capability.
type AgentCapabilityResponse struct {
	Type     string   `json:"type"`
	UseWhen  string   `json:"use_when"`
	Keywords []string `json:"keywords"`
}

// MapAgentMetadataToResponse converts domain metadata to API response.
func MapAgentMetadataToResponse(m agent.AgentMetadata) AgentResponse {
	return AgentResponse{
		Type:              m.Type,
		Name:              m.Name,
		Description:       m.Description,
		Keywords:          m.Keywords,
		Capabilities:      m.Capabilities,
		OutputFormats:     m.OutputFormats,
		EstimatedDuration: m.EstimatedDuration,
		Enabled:           m.Enabled,
	}
}

// MapAgentDetailToResponse converts domain detail to API response.
func MapAgentDetailToResponse(d *agent.AgentDetail) *AgentDetailResponse {
	if d == nil {
		return nil
	}

	resp := &AgentDetailResponse{
		AgentResponse: MapAgentMetadataToResponse(d.AgentMetadata),
	}

	if d.InputSchema != nil {
		resp.InputSchema = d.InputSchema
	}

	if d.OutputSchema != nil {
		resp.OutputSchema = d.OutputSchema
	}

	if len(d.Examples) > 0 {
		examples := make([]AgentExampleResponse, 0, len(d.Examples))
		for _, ex := range d.Examples {
			examples = append(examples, AgentExampleResponse{
				Input:       ex.Input,
				Description: ex.Description,
			})
		}
		resp.Examples = examples
	}

	return resp
}

// MapAgentListToResponse converts a list of agent metadata to API response.
func MapAgentListToResponse(agents []agent.AgentMetadata) *AgentListResponse {
	responses := make([]AgentResponse, 0, len(agents))
	for _, a := range agents {
		responses = append(responses, MapAgentMetadataToResponse(a))
	}

	return &AgentListResponse{
		Agents: responses,
		Total:  len(responses),
	}
}

// MapAgentCapabilitiesToResponse converts capabilities to API response.
func MapAgentCapabilitiesToResponse(capabilities []agent.AgentCapability) *AgentCapabilitiesResponse {
	responses := make([]AgentCapabilityResponse, 0, len(capabilities))
	for _, c := range capabilities {
		responses = append(responses, AgentCapabilityResponse{
			Type:     c.Type,
			UseWhen:  c.UseWhen,
			Keywords: c.Keywords,
		})
	}

	return &AgentCapabilitiesResponse{
		SelectionPrompt: "Based on the user's request, select the most appropriate agent:",
		Agents:          responses,
	}
}
