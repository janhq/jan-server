package response

import (
	"context"

	"jan-server/services/response-api/internal/domain/tool"
)

// Repository defines persistence operations for responses.
type Repository interface {
	Create(ctx context.Context, response *Response) error
	Update(ctx context.Context, response *Response) error
	FindByPublicID(ctx context.Context, publicID string) (*Response, error)
	MarkCancelled(ctx context.Context, response *Response) error
}

// ToolExecutionRepository persists tool execution metadata.
type ToolExecutionRepository interface {
	RecordExecutions(ctx context.Context, responseID uint, executions []tool.Execution) error
}
