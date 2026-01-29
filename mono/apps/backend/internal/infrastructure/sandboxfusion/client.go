package sandboxfusion

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

type Client struct {
	baseURL    string
	httpClient *resty.Client
}

type RunCodeRequest struct {
	Code      string `json:"code"`
	Language  string `json:"language,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

type Artifact struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type RunResult struct {
	Status        string  `json:"status"`
	ExecutionTime float64 `json:"execution_time"`
	ReturnCode    int     `json:"return_code"`
	Stdout        string  `json:"stdout"`
	Stderr        string  `json:"stderr"`
}

type SandboxFusionAPIResponse struct {
	Status        string            `json:"status"`
	Message       string            `json:"message"`
	CompileResult interface{}       `json:"compile_result"`
	RunResult     *RunResult        `json:"run_result"`
	ExecutorPod   *string           `json:"executor_pod_name"`
	Files         map[string]string `json:"files"`
}

type RunCodeResponse struct {
	Stdout    string     `json:"stdout"`
	Stderr    string     `json:"stderr"`
	Duration  int        `json:"duration_ms"`
	SessionID string     `json:"session_id"`
	Artifacts []Artifact `json:"artifacts"`
	Error     string     `json:"error,omitempty"`
}

func NewClient(baseURL string) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		return nil
	}
	client := resty.New().
		SetBaseURL(baseURL).
		SetHeader("User-Agent", "Jan-MCP-SandboxFusion/1.0").
		SetTimeout(20 * time.Second)
	return &Client{
		baseURL:    baseURL,
		httpClient: client,
	}
}

func (c *Client) IsEnabled() bool {
	return c != nil && c.baseURL != ""
}

func (c *Client) RunCode(ctx context.Context, req RunCodeRequest) (*RunCodeResponse, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("sandboxfusion client is not configured")
	}
	var apiResp SandboxFusionAPIResponse
	httpResp, err := c.httpClient.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(req).
		SetResult(&apiResp).
		Post("/run_code")
	if err != nil {
		return nil, fmt.Errorf("sandboxfusion request failed: %w", err)
	}
	if httpResp.IsError() {
		return nil, fmt.Errorf("sandboxfusion error (%d): %s", httpResp.StatusCode(), httpResp.String())
	}

	// Map the API response to our expected format
	resp := &RunCodeResponse{
		SessionID: req.SessionID,
	}

	if apiResp.RunResult != nil {
		resp.Stdout = apiResp.RunResult.Stdout
		resp.Stderr = apiResp.RunResult.Stderr
		resp.Duration = int(apiResp.RunResult.ExecutionTime * 1000) // Convert to milliseconds
	}

	if apiResp.Status != "Success" {
		resp.Error = apiResp.Message
	}

	// Convert files map to artifacts
	if len(apiResp.Files) > 0 {
		resp.Artifacts = make([]Artifact, 0, len(apiResp.Files))
		for name, url := range apiResp.Files {
			resp.Artifacts = append(resp.Artifacts, Artifact{
				Name: name,
				URL:  url,
			})
		}
	}

	return resp, nil
}
