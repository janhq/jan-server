package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"jan-server/services/mcp-tools/internal/infrastructure/connectorapi"
	"jan-server/services/mcp-tools/internal/infrastructure/metrics"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog/log"
)

const (
	GmailAPIURL = "https://gmail.googleapis.com/gmail/v1"
)

// GmailMCP handles Gmail connector MCP tools.
type GmailMCP struct {
	connectorClient *connectorapi.Client
	llmAPIURL       string
	enabled         bool
	httpClient      *http.Client
}

// NewGmailMCP creates a new Gmail MCP handler.
func NewGmailMCP(llmAPIURL string, enabled bool) *GmailMCP {
	return &GmailMCP{
		connectorClient: connectorapi.NewClient(llmAPIURL),
		llmAPIURL:       llmAPIURL,
		enabled:         enabled,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// RegisterTools registers all Gmail MCP tools.
func (g *GmailMCP) RegisterTools(server *mcp.Server) {
	if !g.enabled {
		log.Warn().Msg("Gmail connector MCP tools disabled")
		return
	}

	g.registerSearchEmails(server)
	g.registerGetEmail(server)
	g.registerListLabels(server)

	log.Info().Msg("Registered Gmail connector MCP tools (3 tools)")
}

// getAccessToken gets the Gmail access token for the user.
func (g *GmailMCP) getAccessToken(ctx context.Context) (string, error) {
	authToken, ok := ctx.Value("auth_token").(string)
	if !ok || authToken == "" {
		return "", fmt.Errorf("authentication required")
	}

	tokenResp, err := g.connectorClient.GetAccessToken(ctx, authToken, connectorapi.ConnectorTypeGmail)
	if err != nil {
		return "", fmt.Errorf("Gmail not connected: %w", err)
	}

	return tokenResp.AccessToken, nil
}

// gmailRequest makes a request to the Gmail API.
func (g *GmailMCP) gmailRequest(ctx context.Context, method, path string) ([]byte, error) {
	accessToken, err := g.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, GmailAPIURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Gmail API error %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

type SearchEmailsArgs struct {
	Query      string `json:"query"`
	MaxResults *int   `json:"max_results,omitempty"`
}

func (g *GmailMCP) registerSearchEmails(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "gmail_search_emails",
		Description: "Search emails using Gmail search syntax (e.g., 'from:boss@company.com', 'subject:meeting', 'is:unread'). Requires Gmail connector.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input SearchEmailsArgs) (*mcp.CallToolResult, interface{}, error) {
		startTime := time.Now()

		if input.Query == "" {
			return nil, nil, fmt.Errorf("query is required")
		}

		maxResults := 10
		if input.MaxResults != nil && *input.MaxResults > 0 && *input.MaxResults <= 50 {
			maxResults = *input.MaxResults
		}

		path := fmt.Sprintf("/users/me/messages?q=%s&maxResults=%d", url.QueryEscape(input.Query), maxResults)
		respBody, err := g.gmailRequest(ctx, http.MethodGet, path)
		if err != nil {
			metrics.RecordToolCall("gmail_search_emails", "gmail", "error", time.Since(startTime).Seconds())
			return nil, nil, err
		}

		// Parse message list
		var listResp struct {
			Messages []struct {
				ID       string `json:"id"`
				ThreadID string `json:"threadId"`
			} `json:"messages"`
			NextPageToken string `json:"nextPageToken"`
		}
		if err := json.Unmarshal(respBody, &listResp); err != nil {
			return nil, nil, fmt.Errorf("parse response: %w", err)
		}

		// Fetch details for each message (limited to avoid too many requests)
		var emails []map[string]interface{}
		limit := len(listResp.Messages)
		if limit > 5 {
			limit = 5 // Limit detailed fetches
		}

		for i := 0; i < limit; i++ {
			msgPath := fmt.Sprintf("/users/me/messages/%s?format=metadata&metadataHeaders=From&metadataHeaders=To&metadataHeaders=Subject&metadataHeaders=Date", listResp.Messages[i].ID)
			msgBody, err := g.gmailRequest(ctx, http.MethodGet, msgPath)
			if err != nil {
				continue
			}

			var msg map[string]interface{}
			if err := json.Unmarshal(msgBody, &msg); err != nil {
				continue
			}
			emails = append(emails, msg)
		}

		result := map[string]interface{}{
			"query":        input.Query,
			"total_found":  len(listResp.Messages),
			"emails":       emails,
			"has_more":     listResp.NextPageToken != "",
		}

		metrics.RecordToolCall("gmail_search_emails", "gmail", "success", time.Since(startTime).Seconds())
		return nil, result, nil
	})
}

type GetEmailArgs struct {
	MessageID string `json:"message_id"`
}

func (g *GmailMCP) registerGetEmail(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "gmail_get_email",
		Description: "Get the full content of an email by message ID. Requires Gmail connector.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetEmailArgs) (*mcp.CallToolResult, interface{}, error) {
		startTime := time.Now()

		if input.MessageID == "" {
			return nil, nil, fmt.Errorf("message_id is required")
		}

		path := fmt.Sprintf("/users/me/messages/%s?format=full", input.MessageID)
		respBody, err := g.gmailRequest(ctx, http.MethodGet, path)
		if err != nil {
			metrics.RecordToolCall("gmail_get_email", "gmail", "error", time.Since(startTime).Seconds())
			return nil, nil, err
		}

		// Parse and decode email content
		var msg struct {
			ID       string `json:"id"`
			ThreadID string `json:"threadId"`
			Snippet  string `json:"snippet"`
			Payload  struct {
				Headers []struct {
					Name  string `json:"name"`
					Value string `json:"value"`
				} `json:"headers"`
				MimeType string `json:"mimeType"`
				Body     struct {
					Data string `json:"data"`
				} `json:"body"`
				Parts []struct {
					MimeType string `json:"mimeType"`
					Body     struct {
						Data string `json:"data"`
					} `json:"body"`
				} `json:"parts"`
			} `json:"payload"`
		}
		if err := json.Unmarshal(respBody, &msg); err != nil {
			return nil, nil, fmt.Errorf("parse response: %w", err)
		}

		// Extract headers
		headers := make(map[string]string)
		for _, h := range msg.Payload.Headers {
			headers[h.Name] = h.Value
		}

		// Decode body content
		var bodyContent string
		if msg.Payload.Body.Data != "" {
			decoded, err := base64.URLEncoding.DecodeString(msg.Payload.Body.Data)
			if err == nil {
				bodyContent = string(decoded)
			}
		} else if len(msg.Payload.Parts) > 0 {
			for _, part := range msg.Payload.Parts {
				if strings.HasPrefix(part.MimeType, "text/plain") && part.Body.Data != "" {
					decoded, err := base64.URLEncoding.DecodeString(part.Body.Data)
					if err == nil {
						bodyContent = string(decoded)
						break
					}
				}
			}
		}

		result := map[string]interface{}{
			"id":       msg.ID,
			"thread_id": msg.ThreadID,
			"snippet":  msg.Snippet,
			"headers":  headers,
			"body":     bodyContent,
		}

		metrics.RecordToolCall("gmail_get_email", "gmail", "success", time.Since(startTime).Seconds())
		return nil, result, nil
	})
}

func (g *GmailMCP) registerListLabels(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "gmail_list_labels",
		Description: "List all Gmail labels/folders. Requires Gmail connector.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input struct{}) (*mcp.CallToolResult, interface{}, error) {
		startTime := time.Now()

		path := "/users/me/labels"
		respBody, err := g.gmailRequest(ctx, http.MethodGet, path)
		if err != nil {
			metrics.RecordToolCall("gmail_list_labels", "gmail", "error", time.Since(startTime).Seconds())
			return nil, nil, err
		}

		metrics.RecordToolCall("gmail_list_labels", "gmail", "success", time.Since(startTime).Seconds())
		return nil, json.RawMessage(respBody), nil
	})
}
