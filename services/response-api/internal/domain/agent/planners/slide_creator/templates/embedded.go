package templates

import "embed"

// FS embeds the built-in HTML template catalog.
//
//go:embed index.json index.md */*
var FS embed.FS
