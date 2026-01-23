package steps

// TemplatePickSchema is the JSON schema for template selection.
var TemplatePickSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"choices": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{
						"type": "integer",
					},
					"reason": map[string]any{
						"type": "string",
					},
				},
				"required":             []string{"id", "reason"},
				"additionalProperties": false,
			},
		},
	},
	"required":             []string{"choices"},
	"additionalProperties": false,
}

var slidePlanObjectSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"id":       map[string]any{"type": "integer"},
		"layout":   map[string]any{"type": "string"},
		"title":    map[string]any{"type": "string"},
		"subtitle": map[string]any{"type": "string"},
		"bullets": map[string]any{
			"type":  "array",
			"items": map[string]any{"type": "string"},
		},
		"images": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"src":     map[string]any{"type": "string"},
					"alt":     map[string]any{"type": "string"},
					"caption": map[string]any{"type": "string"},
				},
				"required":             []string{"src", "alt"},
				"additionalProperties": false,
			},
		},
		"table": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"title": map[string]any{"type": "string"},
				"columns": map[string]any{
					"type":  "array",
					"items": map[string]any{"type": "string"},
				},
				"rows": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": "string"},
					},
				},
				"notes": map[string]any{"type": "string"},
			},
			"additionalProperties": false,
		},
		"chart": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"type":  map[string]any{"type": "string"},
				"title": map[string]any{"type": "string"},
				"categories": map[string]any{
					"type":  "array",
					"items": map[string]any{"type": "string"},
				},
				"series": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"name": map[string]any{"type": "string"},
							"values": map[string]any{
								"type":  "array",
								"items": map[string]any{"type": "number"},
							},
						},
						"required":             []string{"values"},
						"additionalProperties": false,
					},
				},
				"notes": map[string]any{"type": "string"},
			},
			"additionalProperties": false,
		},
		"notes": map[string]any{"type": "string"},
	},
	"required":             []string{"id", "title"},
	"additionalProperties": false,
}

var deckThemeSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"title": map[string]any{"type": "string"},
		"theme": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"primary_color":    map[string]any{"type": "string"},
				"accent_color":     map[string]any{"type": "string"},
				"background_color": map[string]any{"type": "string"},
				"text_color":       map[string]any{"type": "string"},
				"font_family":      map[string]any{"type": "string"},
			},
			"required":             []string{"primary_color", "accent_color", "background_color", "text_color", "font_family"},
			"additionalProperties": false,
		},
	},
	"required":             []string{"title", "theme"},
	"additionalProperties": false,
}

// DeckPlanSchema is the JSON schema for HTML slide planning.
var DeckPlanSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"title": map[string]any{
			"type": "string",
		},
		"theme": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"primary_color":    map[string]any{"type": "string"},
				"accent_color":     map[string]any{"type": "string"},
				"background_color": map[string]any{"type": "string"},
				"text_color":       map[string]any{"type": "string"},
				"font_family":      map[string]any{"type": "string"},
			},
			"required":             []string{"primary_color", "accent_color", "background_color", "text_color", "font_family"},
			"additionalProperties": false,
		},
		"slides": map[string]any{
			"type":  "array",
			"items": slidePlanObjectSchema,
		},
	},
	"required":             []string{"title", "theme", "slides"},
	"additionalProperties": false,
}

// DeckThemeSchema is the JSON schema for deck theme selection.
var DeckThemeSchema = deckThemeSchema

// SlidePlanSchema is the JSON schema for a single slide plan.
var SlidePlanSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"slide":          slidePlanObjectSchema,
		"image_query":    map[string]any{"type": "string"},
		"image_required": map[string]any{"type": "boolean"},
	},
	"required":             []string{"slide"},
	"additionalProperties": false,
}
