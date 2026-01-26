package steps

import (
	"io/fs"

	templateassets "jan-server/services/response-api/assets/slide_creator/templates"
)

func embeddedTemplatesRoot() fs.FS {
	return templateassets.FS
}
