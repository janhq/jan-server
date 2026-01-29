package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"jan-server/mono/apps/backend/internal/utils/platformerrors"

	"resty.dev/v3"
)

type ChatModelClient struct {
	client  *resty.Client
	baseURL string
	name    string
}

type ModelsResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

type Model struct {
	ID            string         `json:"id"`
	Object        string         `json:"object"`
	OwnedBy       string         `json:"owned_by"`
	Created       int            `json:"created"`
	DisplayName   string         `json:"display_name"`
	Name          string         `json:"name"`
	CanonicalSlug string         `json:"canonical_slug"`
	Raw           map[string]any `json:"-"`
}

func (m *Model) UnmarshalJSON(data []byte) error {
	type Alias Model
	aux := Alias{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*m = Model(aux)
	m.Raw = raw
	if m.DisplayName == "" {
		if display, ok := raw["display_name"].(string); ok && display != "" {
			m.DisplayName = display
		} else if name, ok := raw["name"].(string); ok && name != "" {
			m.DisplayName = name
		} else {
			m.DisplayName = m.ID
		}
	}
	if m.Name == "" {
		if name, ok := raw["name"].(string); ok {
			m.Name = name
		}
	}
	if m.OwnedBy == "" {
		if ownedBy, ok := raw["owned_by"].(string); ok {
			m.OwnedBy = ownedBy
		}
	}
	if m.CanonicalSlug == "" {
		if slug, ok := raw["canonical_slug"].(string); ok {
			m.CanonicalSlug = slug
		}
	}
	if created, ok := raw["created"]; ok {
		if createdInt, castOK := created.(float64); castOK {
			m.Created = int(createdInt)
		} else if createdInt, castOK := created.(int); castOK {
			m.Created = createdInt
		}
	}
	return nil
}

func NewChatModelClient(client *resty.Client, name, baseURL string) *ChatModelClient {
	return &ChatModelClient{
		client:  client,
		baseURL: normalizeBaseURL(baseURL),
		name:    name,
	}
}

func (c *ChatModelClient) ListModels(ctx context.Context) (*ModelsResponse, error) {
	var respBody ModelsResponse
	resp, err := c.client.R().
		SetContext(ctx).
		SetResult(&respBody).
		Get(c.endpoint("/models"))
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, c.errorFromResponse(ctx, resp, "list models request failed")
	}
	return &respBody, nil
}

func (c *ChatModelClient) endpoint(path string) string {
	if path == "" {
		return c.baseURL
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if c.baseURL == "" {
		return path
	}
	if strings.HasPrefix(path, "/") {
		return c.baseURL + path
	}
	return c.baseURL + "/" + path
}

func (c *ChatModelClient) errorFromResponse(ctx context.Context, resp *resty.Response, message string) error {
	if resp == nil || resp.RawResponse == nil || resp.RawResponse.Body == nil {
		return platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeExternal, fmt.Sprintf("%s with status %d", message, statusCode(resp)), nil, "f4ea9b1a-e011-47f5-8704-4552e4901532")
	}
	defer resp.RawResponse.Body.Close()
	body, err := io.ReadAll(resp.RawResponse.Body)
	if err != nil {
		return platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeExternal, fmt.Sprintf("%s with status %d", message, statusCode(resp)), nil, "bb39f602-d488-4ed2-89ef-0c37b24ebe0e")
	}
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeExternal, fmt.Sprintf("%s with status %d", message, statusCode(resp)), nil, "0d526244-69a7-4d93-82f5-8bfbcb3dbf57")
	}
	return platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeExternal, fmt.Sprintf("%s with status %d: %s", message, statusCode(resp), trimmed), nil, "1d3cd5df-956e-46e7-80de-8dca838b91eb")
}
