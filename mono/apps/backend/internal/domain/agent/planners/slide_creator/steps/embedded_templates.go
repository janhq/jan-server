package steps

import (
	"io/fs"

	templateassets "jan-server/mono/apps/backend/assets/slide_creator/templates"
)

func embeddedTemplatesRoot() fs.FS {
	return templateassets.FS
}
