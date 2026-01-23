package steps

import (
	"io/fs"

	templateassets "jan-server/services/response-api/internal/domain/agent/planners/slide_creator/templates"
)

func embeddedTemplatesRoot() fs.FS {
	return templateassets.FS
}
