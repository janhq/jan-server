package mcp

import (
	"bytes"
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
	GitHubAPIURL = "https://api.github.com"
)

// GitHubMCP handles GitHub connector MCP tools.
type GitHubMCP struct {
	connectorClient *connectorapi.Client
	llmAPIURL       string
	enabled         bool
	httpClient      *http.Client
}

// NewGitHubMCP creates a new GitHub MCP handler.
func NewGitHubMCP(llmAPIURL string, enabled bool) *GitHubMCP {
	return &GitHubMCP{
		connectorClient: connectorapi.NewClient(llmAPIURL),
		llmAPIURL:       llmAPIURL,
		enabled:         enabled,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// RegisterTools registers all GitHub MCP tools.
func (g *GitHubMCP) RegisterTools(server *mcp.Server) {
	if !g.enabled {
		log.Warn().Msg("GitHub connector MCP tools disabled")
		return
	}

	// Read operations
	g.registerSearchRepositories(server)
	g.registerSearchIssues(server)
	g.registerGetFileContent(server)
	g.registerListPullRequests(server)
	g.registerListUserRepos(server)
	g.registerGetPullRequest(server)
	g.registerListBranches(server)

	// Write operations
	g.registerCreateBranch(server)
	g.registerCreateOrUpdateFile(server)
	g.registerDeleteFile(server)
	g.registerCreatePullRequest(server)
	g.registerMergePullRequest(server)
	g.registerAddPRReview(server)
	g.registerCreateIssue(server)
	g.registerAddComment(server)
	g.registerUpdateIssue(server)

	log.Info().Msg("Registered GitHub connector MCP tools (16 tools)")
}

// getAccessToken gets the GitHub access token for the user.
func (g *GitHubMCP) getAccessToken(ctx context.Context) (string, error) {
	authToken, ok := ctx.Value("auth_token").(string)
	if !ok || authToken == "" {
		return "", fmt.Errorf("authentication required")
	}

	tokenResp, err := g.connectorClient.GetAccessToken(ctx, authToken, connectorapi.ConnectorTypeGitHub)
	if err != nil {
		return "", fmt.Errorf("GitHub not connected: %w", err)
	}

	return tokenResp.AccessToken, nil
}

// githubRequest makes a request to the GitHub API.
func (g *GitHubMCP) githubRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	accessToken, err := g.getAccessToken(ctx)
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

	req, err := http.NewRequestWithContext(ctx, method, GitHubAPIURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

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
		return nil, fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// Read Operations

type SearchReposArgs struct {
	Query      string `json:"query"`
	MaxResults *int   `json:"max_results,omitempty"`
}

func (g *GitHubMCP) registerSearchRepositories(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "github_search_repositories",
		Description: "Search GitHub repositories. Requires GitHub connector to be connected.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input SearchReposArgs) (*mcp.CallToolResult, interface{}, error) {
		startTime := time.Now()

		if input.Query == "" {
			return nil, nil, fmt.Errorf("query is required")
		}

		maxResults := 10
		if input.MaxResults != nil && *input.MaxResults > 0 && *input.MaxResults <= 100 {
			maxResults = *input.MaxResults
		}

		path := fmt.Sprintf("/search/repositories?q=%s&per_page=%d", url.QueryEscape(input.Query), maxResults)
		respBody, err := g.githubRequest(ctx, http.MethodGet, path, nil)
		if err != nil {
			metrics.RecordToolCall("github_search_repositories", "github", "error", time.Since(startTime).Seconds())
			return nil, nil, err
		}

		metrics.RecordToolCall("github_search_repositories", "github", "success", time.Since(startTime).Seconds())
		return nil, json.RawMessage(respBody), nil
	})
}

type SearchIssuesArgs struct {
	Query      string `json:"query"`
	MaxResults *int   `json:"max_results,omitempty"`
}

func (g *GitHubMCP) registerSearchIssues(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "github_search_issues",
		Description: "Search GitHub issues and pull requests. Requires GitHub connector.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input SearchIssuesArgs) (*mcp.CallToolResult, interface{}, error) {
		startTime := time.Now()

		if input.Query == "" {
			return nil, nil, fmt.Errorf("query is required")
		}

		maxResults := 10
		if input.MaxResults != nil && *input.MaxResults > 0 && *input.MaxResults <= 100 {
			maxResults = *input.MaxResults
		}

		path := fmt.Sprintf("/search/issues?q=%s&per_page=%d", url.QueryEscape(input.Query), maxResults)
		respBody, err := g.githubRequest(ctx, http.MethodGet, path, nil)
		if err != nil {
			metrics.RecordToolCall("github_search_issues", "github", "error", time.Since(startTime).Seconds())
			return nil, nil, err
		}

		metrics.RecordToolCall("github_search_issues", "github", "success", time.Since(startTime).Seconds())
		return nil, json.RawMessage(respBody), nil
	})
}

type GetFileContentArgs struct {
	Owner  string  `json:"owner"`
	Repo   string  `json:"repo"`
	Path   string  `json:"path"`
	Branch *string `json:"branch,omitempty"`
}

func (g *GitHubMCP) registerGetFileContent(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "github_get_file_content",
		Description: "Get the content of a file from a GitHub repository. Requires GitHub connector.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetFileContentArgs) (*mcp.CallToolResult, interface{}, error) {
		startTime := time.Now()

		if input.Owner == "" || input.Repo == "" || input.Path == "" {
			return nil, nil, fmt.Errorf("owner, repo, and path are required")
		}

		path := fmt.Sprintf("/repos/%s/%s/contents/%s", input.Owner, input.Repo, input.Path)
		if input.Branch != nil && *input.Branch != "" {
			path += "?ref=" + url.QueryEscape(*input.Branch)
		}

		respBody, err := g.githubRequest(ctx, http.MethodGet, path, nil)
		if err != nil {
			metrics.RecordToolCall("github_get_file_content", "github", "error", time.Since(startTime).Seconds())
			return nil, nil, err
		}

		// Decode base64 content
		var fileResp struct {
			Content  string `json:"content"`
			Encoding string `json:"encoding"`
			Name     string `json:"name"`
			Path     string `json:"path"`
			SHA      string `json:"sha"`
			Size     int    `json:"size"`
		}
		if err := json.Unmarshal(respBody, &fileResp); err != nil {
			return nil, nil, fmt.Errorf("parse response: %w", err)
		}

		if fileResp.Encoding == "base64" {
			decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(fileResp.Content, "\n", ""))
			if err == nil {
				fileResp.Content = string(decoded)
			}
		}

		metrics.RecordToolCall("github_get_file_content", "github", "success", time.Since(startTime).Seconds())
		return nil, fileResp, nil
	})
}

type ListPullRequestsArgs struct {
	Owner      string  `json:"owner"`
	Repo       string  `json:"repo"`
	State      *string `json:"state,omitempty"` // open, closed, all
	MaxResults *int    `json:"max_results,omitempty"`
}

func (g *GitHubMCP) registerListPullRequests(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "github_list_pull_requests",
		Description: "List pull requests in a repository. Requires GitHub connector.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListPullRequestsArgs) (*mcp.CallToolResult, interface{}, error) {
		startTime := time.Now()

		if input.Owner == "" || input.Repo == "" {
			return nil, nil, fmt.Errorf("owner and repo are required")
		}

		state := "open"
		if input.State != nil && *input.State != "" {
			state = *input.State
		}
		maxResults := 30
		if input.MaxResults != nil && *input.MaxResults > 0 && *input.MaxResults <= 100 {
			maxResults = *input.MaxResults
		}

		path := fmt.Sprintf("/repos/%s/%s/pulls?state=%s&per_page=%d", input.Owner, input.Repo, state, maxResults)
		respBody, err := g.githubRequest(ctx, http.MethodGet, path, nil)
		if err != nil {
			metrics.RecordToolCall("github_list_pull_requests", "github", "error", time.Since(startTime).Seconds())
			return nil, nil, err
		}

		metrics.RecordToolCall("github_list_pull_requests", "github", "success", time.Since(startTime).Seconds())
		return nil, json.RawMessage(respBody), nil
	})
}

type ListUserReposArgs struct {
	MaxResults *int    `json:"max_results,omitempty"`
	Sort       *string `json:"sort,omitempty"` // created, updated, pushed, full_name
}

func (g *GitHubMCP) registerListUserRepos(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "github_list_user_repos",
		Description: "List repositories for the authenticated user. Requires GitHub connector.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListUserReposArgs) (*mcp.CallToolResult, interface{}, error) {
		startTime := time.Now()

		sort := "updated"
		if input.Sort != nil && *input.Sort != "" {
			sort = *input.Sort
		}
		maxResults := 30
		if input.MaxResults != nil && *input.MaxResults > 0 && *input.MaxResults <= 100 {
			maxResults = *input.MaxResults
		}

		path := fmt.Sprintf("/user/repos?sort=%s&per_page=%d", sort, maxResults)
		respBody, err := g.githubRequest(ctx, http.MethodGet, path, nil)
		if err != nil {
			metrics.RecordToolCall("github_list_user_repos", "github", "error", time.Since(startTime).Seconds())
			return nil, nil, err
		}

		metrics.RecordToolCall("github_list_user_repos", "github", "success", time.Since(startTime).Seconds())
		return nil, json.RawMessage(respBody), nil
	})
}

type GetPullRequestArgs struct {
	Owner      string `json:"owner"`
	Repo       string `json:"repo"`
	PullNumber int    `json:"pull_number"`
}

func (g *GitHubMCP) registerGetPullRequest(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "github_get_pull_request",
		Description: "Get details of a specific pull request. Requires GitHub connector.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetPullRequestArgs) (*mcp.CallToolResult, interface{}, error) {
		startTime := time.Now()

		if input.Owner == "" || input.Repo == "" || input.PullNumber == 0 {
			return nil, nil, fmt.Errorf("owner, repo, and pull_number are required")
		}

		path := fmt.Sprintf("/repos/%s/%s/pulls/%d", input.Owner, input.Repo, input.PullNumber)
		respBody, err := g.githubRequest(ctx, http.MethodGet, path, nil)
		if err != nil {
			metrics.RecordToolCall("github_get_pull_request", "github", "error", time.Since(startTime).Seconds())
			return nil, nil, err
		}

		metrics.RecordToolCall("github_get_pull_request", "github", "success", time.Since(startTime).Seconds())
		return nil, json.RawMessage(respBody), nil
	})
}

type ListBranchesArgs struct {
	Owner      string `json:"owner"`
	Repo       string `json:"repo"`
	MaxResults *int   `json:"max_results,omitempty"`
}

func (g *GitHubMCP) registerListBranches(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "github_list_branches",
		Description: "List branches in a repository. Requires GitHub connector.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListBranchesArgs) (*mcp.CallToolResult, interface{}, error) {
		startTime := time.Now()

		if input.Owner == "" || input.Repo == "" {
			return nil, nil, fmt.Errorf("owner and repo are required")
		}

		maxResults := 30
		if input.MaxResults != nil && *input.MaxResults > 0 && *input.MaxResults <= 100 {
			maxResults = *input.MaxResults
		}

		path := fmt.Sprintf("/repos/%s/%s/branches?per_page=%d", input.Owner, input.Repo, maxResults)
		respBody, err := g.githubRequest(ctx, http.MethodGet, path, nil)
		if err != nil {
			metrics.RecordToolCall("github_list_branches", "github", "error", time.Since(startTime).Seconds())
			return nil, nil, err
		}

		metrics.RecordToolCall("github_list_branches", "github", "success", time.Since(startTime).Seconds())
		return nil, json.RawMessage(respBody), nil
	})
}

// Write Operations

type CreateBranchArgs struct {
	Owner      string `json:"owner"`
	Repo       string `json:"repo"`
	BranchName string `json:"branch_name"`
	FromBranch string `json:"from_branch"` // Base branch (e.g., "main")
}

func (g *GitHubMCP) registerCreateBranch(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "github_create_branch",
		Description: "Create a new branch in a repository. Requires GitHub connector with write access.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input CreateBranchArgs) (*mcp.CallToolResult, interface{}, error) {
		startTime := time.Now()

		if input.Owner == "" || input.Repo == "" || input.BranchName == "" || input.FromBranch == "" {
			return nil, nil, fmt.Errorf("owner, repo, branch_name, and from_branch are required")
		}

		// Get the SHA of the base branch
		refPath := fmt.Sprintf("/repos/%s/%s/git/ref/heads/%s", input.Owner, input.Repo, input.FromBranch)
		refBody, err := g.githubRequest(ctx, http.MethodGet, refPath, nil)
		if err != nil {
			metrics.RecordToolCall("github_create_branch", "github", "error", time.Since(startTime).Seconds())
			return nil, nil, fmt.Errorf("get base branch: %w", err)
		}

		var refResp struct {
			Object struct {
				SHA string `json:"sha"`
			} `json:"object"`
		}
		if err := json.Unmarshal(refBody, &refResp); err != nil {
			return nil, nil, fmt.Errorf("parse ref response: %w", err)
		}

		// Create the new branch
		createPath := fmt.Sprintf("/repos/%s/%s/git/refs", input.Owner, input.Repo)
		createBody := map[string]string{
			"ref": "refs/heads/" + input.BranchName,
			"sha": refResp.Object.SHA,
		}

		respBody, err := g.githubRequest(ctx, http.MethodPost, createPath, createBody)
		if err != nil {
			metrics.RecordToolCall("github_create_branch", "github", "error", time.Since(startTime).Seconds())
			return nil, nil, err
		}

		log.Info().
			Str("owner", input.Owner).
			Str("repo", input.Repo).
			Str("branch", input.BranchName).
			Msg("[GitHub MCP] Created branch")

		metrics.RecordToolCall("github_create_branch", "github", "success", time.Since(startTime).Seconds())
		return nil, json.RawMessage(respBody), nil
	})
}

type CreateOrUpdateFileArgs struct {
	Owner         string  `json:"owner"`
	Repo          string  `json:"repo"`
	Path          string  `json:"path"`
	Content       string  `json:"content"`
	Message       string  `json:"message"`
	Branch        *string `json:"branch,omitempty"`
	SHA           *string `json:"sha,omitempty"` // Required for updates
	CommitterName *string `json:"committer_name,omitempty"`
	CommitterEmail *string `json:"committer_email,omitempty"`
}

func (g *GitHubMCP) registerCreateOrUpdateFile(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "github_create_or_update_file",
		Description: "Create or update a file in a repository. Provide SHA for updates. Requires GitHub connector with write access.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input CreateOrUpdateFileArgs) (*mcp.CallToolResult, interface{}, error) {
		startTime := time.Now()

		if input.Owner == "" || input.Repo == "" || input.Path == "" || input.Content == "" || input.Message == "" {
			return nil, nil, fmt.Errorf("owner, repo, path, content, and message are required")
		}

		apiPath := fmt.Sprintf("/repos/%s/%s/contents/%s", input.Owner, input.Repo, input.Path)

		body := map[string]interface{}{
			"message": input.Message,
			"content": base64.StdEncoding.EncodeToString([]byte(input.Content)),
		}

		if input.Branch != nil && *input.Branch != "" {
			body["branch"] = *input.Branch
		}
		if input.SHA != nil && *input.SHA != "" {
			body["sha"] = *input.SHA
		}
		if input.CommitterName != nil && input.CommitterEmail != nil {
			body["committer"] = map[string]string{
				"name":  *input.CommitterName,
				"email": *input.CommitterEmail,
			}
		}

		respBody, err := g.githubRequest(ctx, http.MethodPut, apiPath, body)
		if err != nil {
			metrics.RecordToolCall("github_create_or_update_file", "github", "error", time.Since(startTime).Seconds())
			return nil, nil, err
		}

		log.Info().
			Str("owner", input.Owner).
			Str("repo", input.Repo).
			Str("path", input.Path).
			Msg("[GitHub MCP] Created/updated file")

		metrics.RecordToolCall("github_create_or_update_file", "github", "success", time.Since(startTime).Seconds())
		return nil, json.RawMessage(respBody), nil
	})
}

type DeleteFileArgs struct {
	Owner   string  `json:"owner"`
	Repo    string  `json:"repo"`
	Path    string  `json:"path"`
	Message string  `json:"message"`
	SHA     string  `json:"sha"`
	Branch  *string `json:"branch,omitempty"`
}

func (g *GitHubMCP) registerDeleteFile(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "github_delete_file",
		Description: "Delete a file from a repository. Requires the file's SHA. Requires GitHub connector with write access.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input DeleteFileArgs) (*mcp.CallToolResult, interface{}, error) {
		startTime := time.Now()

		if input.Owner == "" || input.Repo == "" || input.Path == "" || input.Message == "" || input.SHA == "" {
			return nil, nil, fmt.Errorf("owner, repo, path, message, and sha are required")
		}

		apiPath := fmt.Sprintf("/repos/%s/%s/contents/%s", input.Owner, input.Repo, input.Path)

		body := map[string]interface{}{
			"message": input.Message,
			"sha":     input.SHA,
		}
		if input.Branch != nil && *input.Branch != "" {
			body["branch"] = *input.Branch
		}

		respBody, err := g.githubRequest(ctx, http.MethodDelete, apiPath, body)
		if err != nil {
			metrics.RecordToolCall("github_delete_file", "github", "error", time.Since(startTime).Seconds())
			return nil, nil, err
		}

		log.Info().
			Str("owner", input.Owner).
			Str("repo", input.Repo).
			Str("path", input.Path).
			Msg("[GitHub MCP] Deleted file")

		metrics.RecordToolCall("github_delete_file", "github", "success", time.Since(startTime).Seconds())
		return nil, json.RawMessage(respBody), nil
	})
}

type CreatePullRequestArgs struct {
	Owner string  `json:"owner"`
	Repo  string  `json:"repo"`
	Title string  `json:"title"`
	Body  *string `json:"body,omitempty"`
	Head  string  `json:"head"`  // Branch with changes
	Base  string  `json:"base"`  // Target branch (e.g., "main")
	Draft *bool   `json:"draft,omitempty"`
}

func (g *GitHubMCP) registerCreatePullRequest(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "github_create_pull_request",
		Description: "Create a pull request. Requires GitHub connector with write access.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input CreatePullRequestArgs) (*mcp.CallToolResult, interface{}, error) {
		startTime := time.Now()

		if input.Owner == "" || input.Repo == "" || input.Title == "" || input.Head == "" || input.Base == "" {
			return nil, nil, fmt.Errorf("owner, repo, title, head, and base are required")
		}

		apiPath := fmt.Sprintf("/repos/%s/%s/pulls", input.Owner, input.Repo)

		body := map[string]interface{}{
			"title": input.Title,
			"head":  input.Head,
			"base":  input.Base,
		}
		if input.Body != nil {
			body["body"] = *input.Body
		}
		if input.Draft != nil {
			body["draft"] = *input.Draft
		}

		respBody, err := g.githubRequest(ctx, http.MethodPost, apiPath, body)
		if err != nil {
			metrics.RecordToolCall("github_create_pull_request", "github", "error", time.Since(startTime).Seconds())
			return nil, nil, err
		}

		log.Info().
			Str("owner", input.Owner).
			Str("repo", input.Repo).
			Str("title", input.Title).
			Msg("[GitHub MCP] Created pull request")

		metrics.RecordToolCall("github_create_pull_request", "github", "success", time.Since(startTime).Seconds())
		return nil, json.RawMessage(respBody), nil
	})
}

type MergePullRequestArgs struct {
	Owner       string  `json:"owner"`
	Repo        string  `json:"repo"`
	PullNumber  int     `json:"pull_number"`
	CommitTitle *string `json:"commit_title,omitempty"`
	MergeMethod *string `json:"merge_method,omitempty"` // merge, squash, rebase
}

func (g *GitHubMCP) registerMergePullRequest(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "github_merge_pull_request",
		Description: "Merge a pull request. Requires GitHub connector with write access.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input MergePullRequestArgs) (*mcp.CallToolResult, interface{}, error) {
		startTime := time.Now()

		if input.Owner == "" || input.Repo == "" || input.PullNumber == 0 {
			return nil, nil, fmt.Errorf("owner, repo, and pull_number are required")
		}

		apiPath := fmt.Sprintf("/repos/%s/%s/pulls/%d/merge", input.Owner, input.Repo, input.PullNumber)

		body := map[string]interface{}{}
		if input.CommitTitle != nil {
			body["commit_title"] = *input.CommitTitle
		}
		if input.MergeMethod != nil {
			body["merge_method"] = *input.MergeMethod
		}

		respBody, err := g.githubRequest(ctx, http.MethodPut, apiPath, body)
		if err != nil {
			metrics.RecordToolCall("github_merge_pull_request", "github", "error", time.Since(startTime).Seconds())
			return nil, nil, err
		}

		log.Info().
			Str("owner", input.Owner).
			Str("repo", input.Repo).
			Int("pull_number", input.PullNumber).
			Msg("[GitHub MCP] Merged pull request")

		metrics.RecordToolCall("github_merge_pull_request", "github", "success", time.Since(startTime).Seconds())
		return nil, json.RawMessage(respBody), nil
	})
}

type AddPRReviewArgs struct {
	Owner      string  `json:"owner"`
	Repo       string  `json:"repo"`
	PullNumber int     `json:"pull_number"`
	Body       *string `json:"body,omitempty"`
	Event      string  `json:"event"` // APPROVE, REQUEST_CHANGES, COMMENT
}

func (g *GitHubMCP) registerAddPRReview(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "github_add_pr_review",
		Description: "Add a review to a pull request. Requires GitHub connector with write access.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input AddPRReviewArgs) (*mcp.CallToolResult, interface{}, error) {
		startTime := time.Now()

		if input.Owner == "" || input.Repo == "" || input.PullNumber == 0 || input.Event == "" {
			return nil, nil, fmt.Errorf("owner, repo, pull_number, and event are required")
		}

		apiPath := fmt.Sprintf("/repos/%s/%s/pulls/%d/reviews", input.Owner, input.Repo, input.PullNumber)

		body := map[string]interface{}{
			"event": input.Event,
		}
		if input.Body != nil {
			body["body"] = *input.Body
		}

		respBody, err := g.githubRequest(ctx, http.MethodPost, apiPath, body)
		if err != nil {
			metrics.RecordToolCall("github_add_pr_review", "github", "error", time.Since(startTime).Seconds())
			return nil, nil, err
		}

		log.Info().
			Str("owner", input.Owner).
			Str("repo", input.Repo).
			Int("pull_number", input.PullNumber).
			Str("event", input.Event).
			Msg("[GitHub MCP] Added PR review")

		metrics.RecordToolCall("github_add_pr_review", "github", "success", time.Since(startTime).Seconds())
		return nil, json.RawMessage(respBody), nil
	})
}

type CreateIssueArgs struct {
	Owner     string   `json:"owner"`
	Repo      string   `json:"repo"`
	Title     string   `json:"title"`
	Body      *string  `json:"body,omitempty"`
	Labels    []string `json:"labels,omitempty"`
	Assignees []string `json:"assignees,omitempty"`
}

func (g *GitHubMCP) registerCreateIssue(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "github_create_issue",
		Description: "Create a new issue in a repository. Requires GitHub connector with write access.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input CreateIssueArgs) (*mcp.CallToolResult, interface{}, error) {
		startTime := time.Now()

		if input.Owner == "" || input.Repo == "" || input.Title == "" {
			return nil, nil, fmt.Errorf("owner, repo, and title are required")
		}

		apiPath := fmt.Sprintf("/repos/%s/%s/issues", input.Owner, input.Repo)

		body := map[string]interface{}{
			"title": input.Title,
		}
		if input.Body != nil {
			body["body"] = *input.Body
		}
		if len(input.Labels) > 0 {
			body["labels"] = input.Labels
		}
		if len(input.Assignees) > 0 {
			body["assignees"] = input.Assignees
		}

		respBody, err := g.githubRequest(ctx, http.MethodPost, apiPath, body)
		if err != nil {
			metrics.RecordToolCall("github_create_issue", "github", "error", time.Since(startTime).Seconds())
			return nil, nil, err
		}

		log.Info().
			Str("owner", input.Owner).
			Str("repo", input.Repo).
			Str("title", input.Title).
			Msg("[GitHub MCP] Created issue")

		metrics.RecordToolCall("github_create_issue", "github", "success", time.Since(startTime).Seconds())
		return nil, json.RawMessage(respBody), nil
	})
}

type AddCommentArgs struct {
	Owner       string `json:"owner"`
	Repo        string `json:"repo"`
	IssueNumber int    `json:"issue_number"`
	Body        string `json:"body"`
}

func (g *GitHubMCP) registerAddComment(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "github_add_comment",
		Description: "Add a comment to an issue or pull request. Requires GitHub connector with write access.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input AddCommentArgs) (*mcp.CallToolResult, interface{}, error) {
		startTime := time.Now()

		if input.Owner == "" || input.Repo == "" || input.IssueNumber == 0 || input.Body == "" {
			return nil, nil, fmt.Errorf("owner, repo, issue_number, and body are required")
		}

		apiPath := fmt.Sprintf("/repos/%s/%s/issues/%d/comments", input.Owner, input.Repo, input.IssueNumber)

		body := map[string]string{
			"body": input.Body,
		}

		respBody, err := g.githubRequest(ctx, http.MethodPost, apiPath, body)
		if err != nil {
			metrics.RecordToolCall("github_add_comment", "github", "error", time.Since(startTime).Seconds())
			return nil, nil, err
		}

		log.Info().
			Str("owner", input.Owner).
			Str("repo", input.Repo).
			Int("issue_number", input.IssueNumber).
			Msg("[GitHub MCP] Added comment")

		metrics.RecordToolCall("github_add_comment", "github", "success", time.Since(startTime).Seconds())
		return nil, json.RawMessage(respBody), nil
	})
}

type UpdateIssueArgs struct {
	Owner       string   `json:"owner"`
	Repo        string   `json:"repo"`
	IssueNumber int      `json:"issue_number"`
	Title       *string  `json:"title,omitempty"`
	Body        *string  `json:"body,omitempty"`
	State       *string  `json:"state,omitempty"` // open, closed
	Labels      []string `json:"labels,omitempty"`
	Assignees   []string `json:"assignees,omitempty"`
}

func (g *GitHubMCP) registerUpdateIssue(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "github_update_issue",
		Description: "Update an existing issue. Requires GitHub connector with write access.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input UpdateIssueArgs) (*mcp.CallToolResult, interface{}, error) {
		startTime := time.Now()

		if input.Owner == "" || input.Repo == "" || input.IssueNumber == 0 {
			return nil, nil, fmt.Errorf("owner, repo, and issue_number are required")
		}

		apiPath := fmt.Sprintf("/repos/%s/%s/issues/%d", input.Owner, input.Repo, input.IssueNumber)

		body := map[string]interface{}{}
		if input.Title != nil {
			body["title"] = *input.Title
		}
		if input.Body != nil {
			body["body"] = *input.Body
		}
		if input.State != nil {
			body["state"] = *input.State
		}
		if len(input.Labels) > 0 {
			body["labels"] = input.Labels
		}
		if len(input.Assignees) > 0 {
			body["assignees"] = input.Assignees
		}

		respBody, err := g.githubRequest(ctx, http.MethodPatch, apiPath, body)
		if err != nil {
			metrics.RecordToolCall("github_update_issue", "github", "error", time.Since(startTime).Seconds())
			return nil, nil, err
		}

		log.Info().
			Str("owner", input.Owner).
			Str("repo", input.Repo).
			Int("issue_number", input.IssueNumber).
			Msg("[GitHub MCP] Updated issue")

		metrics.RecordToolCall("github_update_issue", "github", "success", time.Since(startTime).Seconds())
		return nil, json.RawMessage(respBody), nil
	})
}
