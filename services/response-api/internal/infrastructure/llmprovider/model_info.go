package llmprovider

import (
	"context"
	"fmt"

	"jan-server/services/response-api/internal/domain/llm"
)

// ModelCatalogResponse mirrors the llm-api /v1/models/catalogs/{id} response.
type ModelCatalogResponse struct {
	ID            string `json:"id"`
	ContextLength *int   `json:"context_length,omitempty"`
}

// GetModelInfo fetches model metadata from llm-api.
func (c *Client) GetModelInfo(ctx context.Context, modelID string) (*llm.ModelInfo, error) {
	var resp ModelCatalogResponse

	request := c.httpClient.R().
		SetContext(ctx).
		SetResult(&resp)

	if token := llm.AuthTokenFromContext(ctx); token != "" {
		// Token already includes "Bearer " prefix from original Authorization header
		request.SetHeader("Authorization", token)
	}

	httpResp, err := request.Get(fmt.Sprintf("/v1/models/catalogs/%s", modelID))
	if err != nil {
		return nil, fmt.Errorf("fetch model info: %w", err)
	}

	if httpResp.IsError() {
		// Return nil info if model not found - caller can use defaults
		if httpResp.StatusCode() == 404 {
			return nil, nil
		}
		return nil, fmt.Errorf("llm api error: %s", httpResp.String())
	}

	return &llm.ModelInfo{
		ID:            resp.ID,
		ContextLength: resp.ContextLength,
	}, nil
}

// Ensure interface compliance.
var _ llm.ModelInfoProvider = (*Client)(nil)
