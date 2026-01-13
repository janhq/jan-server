package shareresponses

import (
	"time"

	"jan-server/services/llm-api/internal/domain/share"
)

// ShareResponse represents the response for a share
type ShareResponse struct {
	ID              string            `json:"id"`
	Object          string            `json:"object"`
	Slug            string            `json:"slug"`
	ShareURL        string            `json:"share_url,omitempty"`
	Title           *string           `json:"title,omitempty"`
	ItemID          *string           `json:"item_id,omitempty"`
	Visibility      string            `json:"visibility"`
	ViewCount       int               `json:"view_count"`
	RevokedAt       *int64            `json:"revoked_at,omitempty"`
	LastViewedAt    *int64            `json:"last_viewed_at,omitempty"`
	SnapshotVersion int               `json:"snapshot_version"`
	ShareOptions    *ShareOptionsResp `json:"share_options,omitempty"`
	CreatedAt       int64             `json:"created_at"`
	UpdatedAt       int64             `json:"updated_at"`
}

// ShareOptionsResp represents the share options in response
type ShareOptionsResp struct {
	IncludeImages          bool `json:"include_images"`
	IncludeContextMessages bool `json:"include_context_messages"`
}

// ShareListResponse represents a list of shares
type ShareListResponse struct {
	Object string          `json:"object"`
	Data   []ShareResponse `json:"data"`
}

// PublicShareResponse represents the public-facing share response (no auth)
type PublicShareResponse struct {
	Object    string        `json:"object"`
	Slug      string        `json:"slug"`
	Title     *string       `json:"title,omitempty"`
	CreatedAt int64         `json:"created_at"`
	Snapshot  *SnapshotResp `json:"snapshot"`
}

// SnapshotResp represents the snapshot in public response
type SnapshotResp struct {
	Title         string             `json:"title"`
	ModelName     *string            `json:"model_name,omitempty"`
	AssistantName *string            `json:"assistant_name,omitempty"`
	CreatedAt     int64              `json:"created_at"`
	Items         []SnapshotItemResp `json:"items"`
}

// SnapshotItemResp represents an item in the snapshot
type SnapshotItemResp struct {
	ID        string                `json:"id"`
	Type      string                `json:"type"`
	Role      string                `json:"role"`
	Content   []SnapshotContentResp `json:"content"`
	CallID    *string               `json:"call_id,omitempty"` // For tool role messages to match with tool_calls
	CreatedAt int64                 `json:"created_at"`
}

// SnapshotContentResp represents content in the snapshot
type SnapshotContentResp struct {
	Type        string           `json:"type"`
	Text        string           `json:"text,omitempty"`
	InputText   string           `json:"input_text,omitempty"`
	OutputText  string           `json:"output_text,omitempty"`
	ToolCallID  *string          `json:"tool_call_id,omitempty"`
	ToolCalls   []ToolCallResp   `json:"tool_calls,omitempty"`
	MCPCallData string           `json:"mcp_call,omitempty"` // Direct JSON string for mcp_call
	ToolResult  string           `json:"tool_result,omitempty"`
	Image       *ImageRefResp    `json:"image,omitempty"`
	FileRef     *FileRefResp     `json:"file_ref,omitempty"`
	Annotations []AnnotationResp `json:"annotations,omitempty"`
	Reasoning   string           `json:"reasoning_text,omitempty"`
}

// ToolCallResp represents a tool call in the response
type ToolCallResp struct {
	ID       string                `json:"id"`
	Type     string                `json:"type"`
	Function *ToolCallFunctionResp `json:"function,omitempty"`
}

// ToolCallFunctionResp represents the function details in a tool call
type ToolCallFunctionResp struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ImageRefResp represents an image reference
type ImageRefResp struct {
	URL    string `json:"url,omitempty"`
	FileID string `json:"file_id,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// FileRefResp represents a file reference
type FileRefResp struct {
	FileID   string  `json:"file_id,omitempty"`
	URL      *string `json:"url,omitempty"` // For data URLs or external image URLs
	MimeType *string `json:"mime_type,omitempty"`
	Name     *string `json:"name,omitempty"`
}

// AnnotationResp represents an annotation
type AnnotationResp struct {
	Type       string `json:"type"`
	Text       string `json:"text,omitempty"`
	StartIndex *int   `json:"start_index,omitempty"`
	EndIndex   *int   `json:"end_index,omitempty"`
	URL        string `json:"url,omitempty"`
	FileID     string `json:"file_id,omitempty"`
}

// ShareDeletedResponse represents the delete/revoke confirmation
type ShareDeletedResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Deleted bool   `json:"deleted"`
}

// NewShareResponse creates a ShareResponse from domain Share
func NewShareResponse(s *share.Share, baseURL string) *ShareResponse {
	resp := &ShareResponse{
		ID:              s.PublicID,
		Object:          "share",
		Slug:            s.Slug,
		ShareURL:        baseURL + "/v1/public/shares/" + s.Slug,
		Title:           s.Title,
		ItemID:          s.ItemPublicID,
		Visibility:      string(s.Visibility),
		ViewCount:       s.ViewCount,
		SnapshotVersion: s.SnapshotVersion,
		CreatedAt:       s.CreatedAt.Unix(),
		UpdatedAt:       s.UpdatedAt.Unix(),
	}

	if s.RevokedAt != nil {
		unix := s.RevokedAt.Unix()
		resp.RevokedAt = &unix
	}

	if s.LastViewedAt != nil {
		unix := s.LastViewedAt.Unix()
		resp.LastViewedAt = &unix
	}

	if s.ShareOptions != nil {
		resp.ShareOptions = &ShareOptionsResp{
			IncludeImages:          s.ShareOptions.IncludeImages,
			IncludeContextMessages: s.ShareOptions.IncludeContextMessages,
		}
	}

	return resp
}

// NewShareListResponse creates a ShareListResponse from domain shares
func NewShareListResponse(shares []*share.Share, baseURL string) *ShareListResponse {
	data := make([]ShareResponse, 0, len(shares))
	for _, s := range shares {
		data = append(data, *NewShareResponse(s, baseURL))
	}
	return &ShareListResponse{
		Object: "list",
		Data:   data,
	}
}

// NewPublicShareResponse creates a PublicShareResponse from domain Share
func NewPublicShareResponse(s *share.Share) *PublicShareResponse {
	resp := &PublicShareResponse{
		Object:    "public_share",
		Slug:      s.Slug,
		Title:     s.Title,
		CreatedAt: s.CreatedAt.Unix(),
	}

	if s.Snapshot != nil {
		resp.Snapshot = newSnapshotResp(s.Snapshot)
	}

	return resp
}

func newSnapshotResp(snapshot *share.Snapshot) *SnapshotResp {
	resp := &SnapshotResp{
		Title:         snapshot.Title,
		ModelName:     snapshot.ModelName,
		AssistantName: snapshot.AssistantName,
		CreatedAt:     snapshot.CreatedAt.Unix(),
		Items:         make([]SnapshotItemResp, 0, len(snapshot.Items)),
	}

	for _, item := range snapshot.Items {
		resp.Items = append(resp.Items, newSnapshotItemResp(item))
	}

	return resp
}

func newSnapshotItemResp(item share.SnapshotItem) SnapshotItemResp {
	resp := SnapshotItemResp{
		ID:        item.ID,
		Type:      item.Type,
		Role:      item.Role,
		Content:   make([]SnapshotContentResp, 0, len(item.Content)),
		CallID:    item.CallID, // For tool role messages to match with tool_calls
		CreatedAt: item.CreatedAt.Unix(),
	}

	for _, content := range item.Content {
		resp.Content = append(resp.Content, newSnapshotContentResp(content))
	}

	return resp
}

func newSnapshotContentResp(content share.SnapshotContent) SnapshotContentResp {
	resp := SnapshotContentResp{
		Type:        content.Type,
		Text:        content.Text,
		InputText:   content.InputText,
		OutputText:  content.OutputText,
		ToolCallID:  content.ToolCallID,
		ToolResult:  content.ToolResult,
		Reasoning:   content.Reasoning,
		MCPCallData: content.MCPCallData,
	}

	if len(content.ToolCalls) > 0 {
		resp.ToolCalls = make([]ToolCallResp, 0, len(content.ToolCalls))
		for _, tc := range content.ToolCalls {
			tcResp := ToolCallResp{
				ID:   tc.ID,
				Type: tc.Type,
			}
			if tc.Function != nil {
				tcResp.Function = &ToolCallFunctionResp{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				}
			}
			resp.ToolCalls = append(resp.ToolCalls, tcResp)
		}
	}

	if content.Image != nil {
		resp.Image = &ImageRefResp{
			URL:    content.Image.URL,
			FileID: content.Image.FileID,
			Detail: content.Image.Detail,
		}
	}

	if content.FileRef != nil {
		resp.FileRef = &FileRefResp{
			FileID:   content.FileRef.FileID,
			URL:      content.FileRef.URL,
			MimeType: content.FileRef.MimeType,
			Name:     content.FileRef.Name,
		}
	}

	if len(content.Annotations) > 0 {
		resp.Annotations = make([]AnnotationResp, 0, len(content.Annotations))
		for _, a := range content.Annotations {
			resp.Annotations = append(resp.Annotations, AnnotationResp{
				Type:       a.Type,
				Text:       a.Text,
				StartIndex: a.StartIdx,
				EndIndex:   a.EndIdx,
				URL:        a.URL,
				FileID:     a.FileID,
			})
		}
	}

	return resp
}

// NewShareDeletedResponse creates a delete confirmation response
func NewShareDeletedResponse(publicID string) *ShareDeletedResponse {
	return &ShareDeletedResponse{
		ID:      publicID,
		Object:  "share.deleted",
		Deleted: true,
	}
}

// Helper for time conversion
func timeToUnixPtr(t *time.Time) *int64 {
	if t == nil {
		return nil
	}
	unix := t.Unix()
	return &unix
}
