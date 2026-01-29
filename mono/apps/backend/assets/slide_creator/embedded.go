package slidecreator

import _ "embed"

// ExportPPTXFullScript embeds export_pptx_full.js for PPTX export.
//
//go:embed export_pptx_full.js
var ExportPPTXFullScript string
