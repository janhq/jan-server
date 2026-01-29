package sliderenderer

import _ "embed"

// RenderDeckScript embeds render_deck.py for sandbox execution.
//
//go:embed render_deck.py
var RenderDeckScript string
