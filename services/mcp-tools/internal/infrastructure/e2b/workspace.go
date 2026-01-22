package e2b

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

// WorkspaceResolver resolves workspace paths for E2B sandboxes.
// It calls the e2b-api to get or create workspaces and provides
// path scoping to prevent traversal attacks.
type WorkspaceResolver struct {
	client *Client
}

// NewWorkspaceResolver creates a new workspace resolver
func NewWorkspaceResolver(client *Client) *WorkspaceResolver {
	return &WorkspaceResolver{client: client}
}

// ResolvedWorkspace contains the resolved workspace info
type ResolvedWorkspace struct {
	SandboxPublicID string // e.g., "sb_xxx"
	E2BSandboxID    string // The actual E2B sandbox ID for API calls
	WorkspacePath   string // e.g., "/workspace/conv_xxx/"
	UserID          string
	ConversationID  string
}

// ResolveWorkspace gets or creates a workspace for a user/conversation.
// This is idempotent and safe to call repeatedly.
func (r *WorkspaceResolver) ResolveWorkspace(ctx context.Context, userID, conversationID string) (*ResolvedWorkspace, error) {
	if r.client == nil || !r.client.IsEnabled() {
		return nil, fmt.Errorf("E2B client not enabled")
	}

	if userID == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	// Use default conversation if not provided
	if conversationID == "" {
		conversationID = "default"
	}

	log.Debug().
		Str("user_id", userID).
		Str("conversation_id", conversationID).
		Msg("Resolving workspace")

	// Call e2b-api to get or create workspace
	// This also ensures sandbox is started
	result, err := r.client.GetOrCreateWorkspace(ctx, userID, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve workspace: %w", err)
	}

	resolved := &ResolvedWorkspace{
		SandboxPublicID: result.Sandbox.PublicID,
		WorkspacePath:   result.Workspace.WorkspacePath,
		UserID:          userID,
		ConversationID:  conversationID,
	}

	// Extract E2B sandbox ID if available
	if result.Sandbox.E2BSandboxID != nil {
		resolved.E2BSandboxID = *result.Sandbox.E2BSandboxID
	}

	log.Debug().
		Str("sandbox_public_id", resolved.SandboxPublicID).
		Str("e2b_sandbox_id", resolved.E2BSandboxID).
		Str("workspace_path", resolved.WorkspacePath).
		Msg("Workspace resolved")

	return resolved, nil
}

// ScopePath scopes a relative path to the workspace directory.
// It prevents path traversal attacks by cleaning the path and
// ensuring it stays within the workspace.
func (r *WorkspaceResolver) ScopePath(workspacePath, relativePath string) (string, error) {
	// Clean the workspace path
	workspacePath = filepath.Clean(workspacePath)
	if !strings.HasPrefix(workspacePath, "/") {
		workspacePath = "/" + workspacePath
	}

	// Handle empty or current directory
	if relativePath == "" || relativePath == "." {
		return workspacePath, nil
	}

	// Clean the relative path to prevent traversal
	cleanRelative := filepath.Clean(relativePath)

	// Remove any leading slashes from relative path
	cleanRelative = strings.TrimPrefix(cleanRelative, "/")

	// Check for path traversal attempts
	if strings.Contains(cleanRelative, "..") {
		return "", fmt.Errorf("path traversal not allowed: %s", relativePath)
	}

	// Join paths
	scopedPath := filepath.Join(workspacePath, cleanRelative)

	// Verify the result is still under workspace
	if !strings.HasPrefix(scopedPath, workspacePath) {
		return "", fmt.Errorf("path escapes workspace: %s", relativePath)
	}

	return scopedPath, nil
}

// GetSandboxForUser gets the user's sandbox if it exists and is running
func (r *WorkspaceResolver) GetSandboxForUser(ctx context.Context, userID string) (*SandboxRecord, error) {
	if r.client == nil || !r.client.IsEnabled() {
		return nil, fmt.Errorf("E2B client not enabled")
	}

	return r.client.GetSandboxByUser(ctx, userID)
}

// IsSandboxRunning checks if a user has a running sandbox
func (r *WorkspaceResolver) IsSandboxRunning(ctx context.Context, userID string) (bool, string, error) {
	sandbox, err := r.GetSandboxForUser(ctx, userID)
	if err != nil {
		return false, "", err
	}

	if sandbox == nil {
		return false, "", nil
	}

	if sandbox.Status == "running" && sandbox.E2BSandboxID != nil {
		return true, sandbox.PublicID, nil
	}

	return false, sandbox.PublicID, nil
}
