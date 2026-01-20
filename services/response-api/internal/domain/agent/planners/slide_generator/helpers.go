package slide_generator

import (
	"strings"

	"jan-server/services/response-api/internal/domain/tool"
)

func firstTextContent(contents []tool.MCPContent) string {
	for _, content := range contents {
		if content.Type == "text" && content.Text != "" {
			return strings.TrimSpace(content.Text)
		}
	}
	return ""
}

func intPtr(value int) *int {
	return &value
}
