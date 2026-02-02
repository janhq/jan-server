package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"jan-server/services/mcp-tools/internal/infrastructure/connectorapi"
	"jan-server/services/mcp-tools/internal/infrastructure/metrics"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog/log"
)

const (
	CalendarAPIURL = "https://www.googleapis.com/calendar/v3"
)

// CalendarMCP handles Google Calendar connector MCP tools.
type CalendarMCP struct {
	connectorClient *connectorapi.Client
	llmAPIURL       string
	enabled         bool
	httpClient      *http.Client
}

// NewCalendarMCP creates a new Calendar MCP handler.
func NewCalendarMCP(llmAPIURL string, enabled bool) *CalendarMCP {
	return &CalendarMCP{
		connectorClient: connectorapi.NewClient(llmAPIURL),
		llmAPIURL:       llmAPIURL,
		enabled:         enabled,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// RegisterTools registers all Calendar MCP tools.
func (c *CalendarMCP) RegisterTools(server *mcp.Server) {
	if !c.enabled {
		log.Warn().Msg("Google Calendar connector MCP tools disabled")
		return
	}

	// Read operations
	c.registerListEvents(server)
	c.registerSearchEvents(server)
	c.registerGetEvent(server)
	c.registerListCalendars(server)

	// Write operations
	c.registerCreateEvent(server)
	c.registerUpdateEvent(server)
	c.registerDeleteEvent(server)
	c.registerQuickAdd(server)

	log.Info().Msg("Registered Google Calendar connector MCP tools (8 tools)")
}

// getAccessToken gets the Calendar access token for the user.
func (c *CalendarMCP) getAccessToken(ctx context.Context) (string, error) {
	authToken, ok := ctx.Value("auth_token").(string)
	if !ok || authToken == "" {
		return "", fmt.Errorf("authentication required")
	}

	tokenResp, err := c.connectorClient.GetAccessToken(ctx, authToken, connectorapi.ConnectorTypeGoogleCalendar)
	if err != nil {
		return "", fmt.Errorf("Google Calendar not connected: %w", err)
	}

	return tokenResp.AccessToken, nil
}

// calendarRequest makes a request to the Calendar API.
func (c *CalendarMCP) calendarRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	accessToken, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, CalendarAPIURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Calendar API error %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// Read Operations

type ListEventsArgs struct {
	CalendarID *string `json:"calendar_id,omitempty"` // defaults to "primary"
	TimeMin    *string `json:"time_min,omitempty"`    // RFC3339 format
	TimeMax    *string `json:"time_max,omitempty"`    // RFC3339 format
	MaxResults *int    `json:"max_results,omitempty"`
}

func (c *CalendarMCP) registerListEvents(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "calendar_list_events",
		Description: "List upcoming events from Google Calendar. Requires Google Calendar connector.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListEventsArgs) (*mcp.CallToolResult, interface{}, error) {
		startTime := time.Now()

		calendarID := "primary"
		if input.CalendarID != nil && *input.CalendarID != "" {
			calendarID = *input.CalendarID
		}

		maxResults := 10
		if input.MaxResults != nil && *input.MaxResults > 0 && *input.MaxResults <= 50 {
			maxResults = *input.MaxResults
		}

		params := url.Values{
			"singleEvents": {"true"},
			"orderBy":      {"startTime"},
			"maxResults":   {fmt.Sprintf("%d", maxResults)},
		}

		if input.TimeMin != nil && *input.TimeMin != "" {
			params.Set("timeMin", *input.TimeMin)
		} else {
			// Default to now
			params.Set("timeMin", time.Now().Format(time.RFC3339))
		}

		if input.TimeMax != nil && *input.TimeMax != "" {
			params.Set("timeMax", *input.TimeMax)
		}

		path := fmt.Sprintf("/calendars/%s/events?%s", url.PathEscape(calendarID), params.Encode())
		respBody, err := c.calendarRequest(ctx, http.MethodGet, path, nil)
		if err != nil {
			metrics.RecordToolCall("calendar_list_events", "calendar", "error", time.Since(startTime).Seconds())
			return nil, nil, err
		}

		metrics.RecordToolCall("calendar_list_events", "calendar", "success", time.Since(startTime).Seconds())
		return nil, json.RawMessage(respBody), nil
	})
}

type SearchEventsArgs struct {
	Query      string  `json:"query"`
	CalendarID *string `json:"calendar_id,omitempty"`
	MaxResults *int    `json:"max_results,omitempty"`
}

func (c *CalendarMCP) registerSearchEvents(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "calendar_search_events",
		Description: "Search calendar events by keyword. Requires Google Calendar connector.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input SearchEventsArgs) (*mcp.CallToolResult, interface{}, error) {
		startTime := time.Now()

		if input.Query == "" {
			return nil, nil, fmt.Errorf("query is required")
		}

		calendarID := "primary"
		if input.CalendarID != nil && *input.CalendarID != "" {
			calendarID = *input.CalendarID
		}

		maxResults := 10
		if input.MaxResults != nil && *input.MaxResults > 0 && *input.MaxResults <= 50 {
			maxResults = *input.MaxResults
		}

		params := url.Values{
			"q":          {input.Query},
			"maxResults": {fmt.Sprintf("%d", maxResults)},
		}

		path := fmt.Sprintf("/calendars/%s/events?%s", url.PathEscape(calendarID), params.Encode())
		respBody, err := c.calendarRequest(ctx, http.MethodGet, path, nil)
		if err != nil {
			metrics.RecordToolCall("calendar_search_events", "calendar", "error", time.Since(startTime).Seconds())
			return nil, nil, err
		}

		metrics.RecordToolCall("calendar_search_events", "calendar", "success", time.Since(startTime).Seconds())
		return nil, json.RawMessage(respBody), nil
	})
}

type GetEventArgs struct {
	EventID    string  `json:"event_id"`
	CalendarID *string `json:"calendar_id,omitempty"`
}

func (c *CalendarMCP) registerGetEvent(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "calendar_get_event",
		Description: "Get details of a specific calendar event. Requires Google Calendar connector.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetEventArgs) (*mcp.CallToolResult, interface{}, error) {
		startTime := time.Now()

		if input.EventID == "" {
			return nil, nil, fmt.Errorf("event_id is required")
		}

		calendarID := "primary"
		if input.CalendarID != nil && *input.CalendarID != "" {
			calendarID = *input.CalendarID
		}

		path := fmt.Sprintf("/calendars/%s/events/%s", url.PathEscape(calendarID), url.PathEscape(input.EventID))
		respBody, err := c.calendarRequest(ctx, http.MethodGet, path, nil)
		if err != nil {
			metrics.RecordToolCall("calendar_get_event", "calendar", "error", time.Since(startTime).Seconds())
			return nil, nil, err
		}

		metrics.RecordToolCall("calendar_get_event", "calendar", "success", time.Since(startTime).Seconds())
		return nil, json.RawMessage(respBody), nil
	})
}

func (c *CalendarMCP) registerListCalendars(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "calendar_list_calendars",
		Description: "List all calendars accessible to the user. Requires Google Calendar connector.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input struct{}) (*mcp.CallToolResult, interface{}, error) {
		startTime := time.Now()

		path := "/users/me/calendarList"
		respBody, err := c.calendarRequest(ctx, http.MethodGet, path, nil)
		if err != nil {
			metrics.RecordToolCall("calendar_list_calendars", "calendar", "error", time.Since(startTime).Seconds())
			return nil, nil, err
		}

		metrics.RecordToolCall("calendar_list_calendars", "calendar", "success", time.Since(startTime).Seconds())
		return nil, json.RawMessage(respBody), nil
	})
}

// Write Operations

type CreateEventArgs struct {
	Summary     string   `json:"summary"`
	Description *string  `json:"description,omitempty"`
	Location    *string  `json:"location,omitempty"`
	StartTime   string   `json:"start_time"`   // RFC3339 format
	EndTime     string   `json:"end_time"`     // RFC3339 format
	TimeZone    *string  `json:"time_zone,omitempty"` // e.g., "America/Los_Angeles"
	Attendees   []string `json:"attendees,omitempty"` // Email addresses
	CalendarID  *string  `json:"calendar_id,omitempty"`
}

func (c *CalendarMCP) registerCreateEvent(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "calendar_create_event",
		Description: "Create a new calendar event. Times should be in RFC3339 format (e.g., '2025-01-28T10:00:00-08:00'). Requires Google Calendar connector with write access.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input CreateEventArgs) (*mcp.CallToolResult, interface{}, error) {
		startTime := time.Now()

		if input.Summary == "" || input.StartTime == "" || input.EndTime == "" {
			return nil, nil, fmt.Errorf("summary, start_time, and end_time are required")
		}

		calendarID := "primary"
		if input.CalendarID != nil && *input.CalendarID != "" {
			calendarID = *input.CalendarID
		}

		// Build event body
		event := map[string]interface{}{
			"summary": input.Summary,
			"start": map[string]string{
				"dateTime": input.StartTime,
			},
			"end": map[string]string{
				"dateTime": input.EndTime,
			},
		}

		if input.Description != nil {
			event["description"] = *input.Description
		}
		if input.Location != nil {
			event["location"] = *input.Location
		}
		if input.TimeZone != nil {
			event["start"].(map[string]string)["timeZone"] = *input.TimeZone
			event["end"].(map[string]string)["timeZone"] = *input.TimeZone
		}
		if len(input.Attendees) > 0 {
			attendees := make([]map[string]string, len(input.Attendees))
			for i, email := range input.Attendees {
				attendees[i] = map[string]string{"email": email}
			}
			event["attendees"] = attendees
		}

		path := fmt.Sprintf("/calendars/%s/events", url.PathEscape(calendarID))
		respBody, err := c.calendarRequest(ctx, http.MethodPost, path, event)
		if err != nil {
			metrics.RecordToolCall("calendar_create_event", "calendar", "error", time.Since(startTime).Seconds())
			return nil, nil, err
		}

		log.Info().
			Str("calendar_id", calendarID).
			Str("summary", input.Summary).
			Msg("[Calendar MCP] Created event")

		metrics.RecordToolCall("calendar_create_event", "calendar", "success", time.Since(startTime).Seconds())
		return nil, json.RawMessage(respBody), nil
	})
}

type UpdateEventArgs struct {
	EventID     string   `json:"event_id"`
	Summary     *string  `json:"summary,omitempty"`
	Description *string  `json:"description,omitempty"`
	Location    *string  `json:"location,omitempty"`
	StartTime   *string  `json:"start_time,omitempty"`
	EndTime     *string  `json:"end_time,omitempty"`
	TimeZone    *string  `json:"time_zone,omitempty"`
	Attendees   []string `json:"attendees,omitempty"`
	CalendarID  *string  `json:"calendar_id,omitempty"`
}

func (c *CalendarMCP) registerUpdateEvent(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "calendar_update_event",
		Description: "Update an existing calendar event. Only provided fields will be updated. Requires Google Calendar connector with write access.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input UpdateEventArgs) (*mcp.CallToolResult, interface{}, error) {
		startTime := time.Now()

		if input.EventID == "" {
			return nil, nil, fmt.Errorf("event_id is required")
		}

		calendarID := "primary"
		if input.CalendarID != nil && *input.CalendarID != "" {
			calendarID = *input.CalendarID
		}

		// Build update body (only include provided fields)
		event := make(map[string]interface{})

		if input.Summary != nil {
			event["summary"] = *input.Summary
		}
		if input.Description != nil {
			event["description"] = *input.Description
		}
		if input.Location != nil {
			event["location"] = *input.Location
		}
		if input.StartTime != nil {
			startMap := map[string]string{"dateTime": *input.StartTime}
			if input.TimeZone != nil {
				startMap["timeZone"] = *input.TimeZone
			}
			event["start"] = startMap
		}
		if input.EndTime != nil {
			endMap := map[string]string{"dateTime": *input.EndTime}
			if input.TimeZone != nil {
				endMap["timeZone"] = *input.TimeZone
			}
			event["end"] = endMap
		}
		if len(input.Attendees) > 0 {
			attendees := make([]map[string]string, len(input.Attendees))
			for i, email := range input.Attendees {
				attendees[i] = map[string]string{"email": email}
			}
			event["attendees"] = attendees
		}

		path := fmt.Sprintf("/calendars/%s/events/%s", url.PathEscape(calendarID), url.PathEscape(input.EventID))
		respBody, err := c.calendarRequest(ctx, http.MethodPatch, path, event)
		if err != nil {
			metrics.RecordToolCall("calendar_update_event", "calendar", "error", time.Since(startTime).Seconds())
			return nil, nil, err
		}

		log.Info().
			Str("calendar_id", calendarID).
			Str("event_id", input.EventID).
			Msg("[Calendar MCP] Updated event")

		metrics.RecordToolCall("calendar_update_event", "calendar", "success", time.Since(startTime).Seconds())
		return nil, json.RawMessage(respBody), nil
	})
}

type DeleteEventArgs struct {
	EventID    string  `json:"event_id"`
	CalendarID *string `json:"calendar_id,omitempty"`
}

func (c *CalendarMCP) registerDeleteEvent(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "calendar_delete_event",
		Description: "Delete a calendar event. Requires Google Calendar connector with write access.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input DeleteEventArgs) (*mcp.CallToolResult, interface{}, error) {
		startTime := time.Now()

		if input.EventID == "" {
			return nil, nil, fmt.Errorf("event_id is required")
		}

		calendarID := "primary"
		if input.CalendarID != nil && *input.CalendarID != "" {
			calendarID = *input.CalendarID
		}

		path := fmt.Sprintf("/calendars/%s/events/%s", url.PathEscape(calendarID), url.PathEscape(input.EventID))
		_, err := c.calendarRequest(ctx, http.MethodDelete, path, nil)
		if err != nil {
			metrics.RecordToolCall("calendar_delete_event", "calendar", "error", time.Since(startTime).Seconds())
			return nil, nil, err
		}

		log.Info().
			Str("calendar_id", calendarID).
			Str("event_id", input.EventID).
			Msg("[Calendar MCP] Deleted event")

		metrics.RecordToolCall("calendar_delete_event", "calendar", "success", time.Since(startTime).Seconds())
		return nil, map[string]interface{}{
			"deleted":    true,
			"event_id":   input.EventID,
			"calendar_id": calendarID,
		}, nil
	})
}

type QuickAddArgs struct {
	Text       string  `json:"text"` // e.g., "Meeting with John tomorrow 3pm"
	CalendarID *string `json:"calendar_id,omitempty"`
}

func (c *CalendarMCP) registerQuickAdd(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "calendar_quick_add",
		Description: "Create an event from natural language text (e.g., 'Meeting with John tomorrow at 3pm for 1 hour'). Requires Google Calendar connector with write access.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input QuickAddArgs) (*mcp.CallToolResult, interface{}, error) {
		startTime := time.Now()

		if input.Text == "" {
			return nil, nil, fmt.Errorf("text is required")
		}

		calendarID := "primary"
		if input.CalendarID != nil && *input.CalendarID != "" {
			calendarID = *input.CalendarID
		}

		path := fmt.Sprintf("/calendars/%s/events/quickAdd?text=%s",
			url.PathEscape(calendarID), url.QueryEscape(input.Text))
		respBody, err := c.calendarRequest(ctx, http.MethodPost, path, nil)
		if err != nil {
			metrics.RecordToolCall("calendar_quick_add", "calendar", "error", time.Since(startTime).Seconds())
			return nil, nil, err
		}

		log.Info().
			Str("calendar_id", calendarID).
			Str("text", input.Text).
			Msg("[Calendar MCP] Quick added event")

		metrics.RecordToolCall("calendar_quick_add", "calendar", "success", time.Since(startTime).Seconds())
		return nil, json.RawMessage(respBody), nil
	})
}
