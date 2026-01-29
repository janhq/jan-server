package sandbox

import "context"

// Context keys for passing user/conversation context through request context
type contextKey string

const (
	// UserIDContextKey is the context key for user ID
	UserIDContextKey contextKey = "sandbox_user_id"
	// ConversationIDContextKey is the context key for conversation ID
	ConversationIDContextKey contextKey = "sandbox_conversation_id"
)

// WithUserContext returns a new context with user and conversation IDs set
func WithUserContext(ctx context.Context, userID, conversationID string) context.Context {
	ctx = context.WithValue(ctx, UserIDContextKey, userID)
	ctx = context.WithValue(ctx, ConversationIDContextKey, conversationID)
	return ctx
}

// GetUserID extracts user ID from context
func GetUserID(ctx context.Context) string {
	if val := ctx.Value(UserIDContextKey); val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// GetConversationID extracts conversation ID from context
func GetConversationID(ctx context.Context) string {
	if val := ctx.Value(ConversationIDContextKey); val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// Provider defines the interface for sandbox backends (AIO or E2B)
// Only one provider can be active at a time, selected via SANDBOX_PROVIDER config.
type Provider interface {
	// IsEnabled returns whether this provider is configured and ready
	IsEnabled() bool

	// Name returns the provider name ("aio" or "e2b")
	Name() string

	// ShellExec executes a command in the sandbox shell
	ShellExec(ctx context.Context, command string) (*ShellResult, error)

	// FileRead reads file contents from the sandbox filesystem
	FileRead(ctx context.Context, path string) (string, error)

	// FileWrite writes content to a file in the sandbox filesystem
	FileWrite(ctx context.Context, path, content string) (*FileWriteResult, error)

	// FileList lists files and directories at the given path
	FileList(ctx context.Context, path string) ([]FileInfo, error)

	// CodeExecute executes code in the specified language (python, nodejs)
	CodeExecute(ctx context.Context, code, language string) (*CodeResult, error)

	// InstallPackages installs packages using the specified package manager
	// manager can be "pip" or "npm"
	InstallPackages(ctx context.Context, packages []string, manager string) (*InstallResult, error)

	// BrowserInfo returns browser automation URLs (AIO-specific)
	// Returns nil for providers that don't support browser automation
	BrowserInfo(ctx context.Context) (*BrowserInfo, error)

	// Screenshot takes a desktop screenshot (E2B-specific)
	// Returns base64-encoded PNG image
	Screenshot(ctx context.Context) (string, error)

	// Click performs a mouse click at the specified coordinates (E2B-specific)
	// button can be "left", "right", or "middle"
	Click(ctx context.Context, x, y int, button string) error

	// Type types text on the desktop (E2B-specific)
	Type(ctx context.Context, text string) error

	// MarkitdownConvert converts a URL or document to Markdown format
	MarkitdownConvert(ctx context.Context, url string) (string, error)
}

// ShellResult contains the result of a shell command execution
type ShellResult struct {
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr,omitempty"`
	ExitCode   int    `json:"exit_code"`
	DurationMs int64  `json:"duration_ms,omitempty"`
}

// FileWriteResult contains the result of a file write operation
type FileWriteResult struct {
	Success      bool   `json:"success"`
	Path         string `json:"path"`
	BytesWritten int    `json:"bytes_written"`
}

// FileInfo describes a file or directory in the sandbox filesystem
type FileInfo struct {
	Name  string `json:"name"`
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size,omitempty"`
}

// CodeResult contains the result of code execution
type CodeResult struct {
	Status     string `json:"status"` // "ok" or "error"
	Success    bool   `json:"success"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr,omitempty"`
	ExitCode   int    `json:"exit_code"`
	DurationMs int64  `json:"duration_ms,omitempty"`
}

// InstallResult contains the result of package installation
type InstallResult struct {
	Status     string   `json:"status"` // "success" or "error"
	Success    bool     `json:"success"`
	Packages   []string `json:"packages_installed,omitempty"`
	Output     string   `json:"output,omitempty"`
	ExitCode   int      `json:"exit_code"`
	DurationMs int64    `json:"duration_ms,omitempty"`
	Error      string   `json:"error,omitempty"`
}

// BrowserInfo contains browser automation URLs
type BrowserInfo struct {
	CdpURL string `json:"cdp_url"`
	VncURL string `json:"vnc_url"`
}

// ErrNotSupported is returned when a provider doesn't support an operation
type ErrNotSupported struct {
	Provider  string
	Operation string
}

func (e *ErrNotSupported) Error() string {
	return e.Provider + " provider does not support " + e.Operation
}

// NewNotSupportedError creates a new ErrNotSupported error
func NewNotSupportedError(provider, operation string) error {
	return &ErrNotSupported{Provider: provider, Operation: operation}
}
