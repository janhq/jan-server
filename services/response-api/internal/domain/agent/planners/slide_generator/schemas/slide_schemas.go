package schemas

import "sort"

// PlanAndTemplateSchema is the JSON schema for the combined Planner + Template Builder response.
var PlanAndTemplateSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"plan": map[string]any{
			"type":        "object",
			"description": "The slide deck plan with content outline",
			"properties": map[string]any{
				"deckTitle": map[string]any{
					"type":        "string",
					"description": "The title of the presentation",
				},
				"audience": map[string]any{
					"type":        "string",
					"description": "Target audience description",
				},
				"tone": map[string]any{
					"type":        "string",
					"description": "Presentation tone (professional, casual, technical, etc.)",
				},
				"purpose": map[string]any{
					"type":        "string",
					"description": "Main purpose of the presentation",
				},
				"recommendedSlideCount": map[string]any{
					"type":        "integer",
					"description": "Recommended number of slides",
				},
				"slides": map[string]any{
					"type":        "array",
					"description": "Array of slide plan entries",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"index": map[string]any{
								"type":        "integer",
								"description": "1-based slide index",
							},
							"title": map[string]any{
								"type":        "string",
								"description": "Slide title",
							},
							"purpose": map[string]any{
								"type":        "string",
								"description": "Purpose of this slide",
							},
							"keyPoints": map[string]any{
								"type":        "array",
								"description": "Key points to cover",
								"items":       map[string]any{"type": "string"},
							},
							"suggestedLayout": map[string]any{
								"type":        "string",
								"description": "Suggested layout type",
								"enum":        []string{"TITLE", "SECTION_HEADER", "TITLE_AND_BULLETS", "TITLE_TWO_COLUMNS", "TITLE_IMAGE", "FULL_BLEED_IMAGE", "CHART", "TABLE", "QUOTE", "TIMELINE", "CLOSING", "APPENDIX", "DASHBOARD_3KPI_2COL", "CHART_AND_INSIGHTS", "TABLE_AND_CALLOUTS"},
							},
							"visualIdeas": map[string]any{
								"type":        "array",
								"description": "Visual element suggestions",
								"items":       map[string]any{"type": "string"},
							},
						},
						"required":             []string{"index", "title", "purpose", "keyPoints", "suggestedLayout", "visualIdeas"},
						"additionalProperties": false,
					},
				},
			},
			"required":             []string{"deckTitle", "audience", "tone", "purpose", "recommendedSlideCount", "slides"},
			"additionalProperties": false,
		},
		"template": map[string]any{
			"type":        "object",
			"description": "The DeckSpec template structure",
			"properties": map[string]any{
				"version": map[string]any{
					"type":        "string",
					"description": "Template version (e.g., '1.0')",
				},
				"metadata": map[string]any{
					"type":        "object",
					"description": "Presentation metadata",
					"properties": map[string]any{
						"title":    map[string]any{"type": "string"},
						"language": map[string]any{"type": "string"},
						"audience": map[string]any{"type": "string"},
						"purpose":  map[string]any{"type": "string"},
					},
					"required":             []string{"title", "language", "audience", "purpose"},
					"additionalProperties": false,
				},
				"theme": map[string]any{
					"type":        "object",
					"description": "Theme configuration including canvas, grid, colors, typography",
					"properties": map[string]any{
						"canvas": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"size": map[string]any{
									"type":        "string",
									"description": "Canvas size preset. This generator assumes a normal PowerPoint WIDE_16x9 slide.",
									"enum":        []string{"WIDE_16x9"},
								},
								"customSize": map[string]any{
									"type":        "object",
									"description": "Optional custom size (in inches) when size=CUSTOM. Use null for WIDE_16x9.",
									"properties": map[string]any{
										"width":  map[string]any{"type": "number"},
										"height": map[string]any{"type": "number"},
									},
									"required":             []string{"width", "height"},
									"additionalProperties": false,
								},
								"unit": map[string]any{
									"type":        "string",
									"description": "Coordinate unit for all rects. Use pt so WIDE_16x9 becomes 960pt x 540pt.",
									"enum":        []string{"pt"},
								},
								"safeMargins": map[string]any{
									"type": "object",
									"properties": map[string]any{
										"top":    map[string]any{"type": "number", "minimum": 0, "maximum": 200, "default": 36},
										"right":  map[string]any{"type": "number", "minimum": 0, "maximum": 200, "default": 36},
										"bottom": map[string]any{"type": "number", "minimum": 0, "maximum": 200, "default": 36},
										"left":   map[string]any{"type": "number", "minimum": 0, "maximum": 200, "default": 36},
									},
									"required":             []string{"top", "right", "bottom", "left"},
									"additionalProperties": false,
								},
							},
							"required":             []string{"size", "unit", "safeMargins"},
							"additionalProperties": false,
						},
						"grid": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"columns":  map[string]any{"type": "integer"},
								"gutter":   map[string]any{"type": "number"},
								"baseline": map[string]any{"type": "number"},
								"snap":     map[string]any{"type": "boolean"},
							},
							"required":             []string{"columns", "gutter", "baseline", "snap"},
							"additionalProperties": false,
						},
						"colors": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"palette": map[string]any{
									"type": "object",
									"properties": map[string]any{
										"primary":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
										"secondary": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
										"neutral":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
										"accent":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
									},
									"required":             []string{"primary", "secondary", "neutral", "accent"},
									"additionalProperties": false,
								},
								"semantic": map[string]any{
									"type": "object",
									"properties": map[string]any{
										"background": map[string]any{"type": "string"},
										"text":       map[string]any{"type": "string"},
										"mutedText":  map[string]any{"type": "string"},
										"border":     map[string]any{"type": "string"},
										"link":       map[string]any{"type": "string"},
										"success":    map[string]any{"type": "string"},
										"warning":    map[string]any{"type": "string"},
										"danger":     map[string]any{"type": "string"},
									},
									"required":             []string{"background", "text", "mutedText", "border", "link", "success", "warning", "danger"},
									"additionalProperties": false,
								},
							},
							"required":             []string{"palette", "semantic"},
							"additionalProperties": false,
						},
						"typography": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"families": map[string]any{
									"type": "object",
									"properties": map[string]any{
										"heading": map[string]any{"type": "string"},
										"body":    map[string]any{"type": "string"},
										"mono":    map[string]any{"type": "string"},
									},
									"required":             []string{"heading", "body", "mono"},
									"additionalProperties": false,
								},
								"scale": map[string]any{
									"type": "object",
									"properties": map[string]any{
										"h1":      map[string]any{"type": "number"},
										"h2":      map[string]any{"type": "number"},
										"h3":      map[string]any{"type": "number"},
										"body":    map[string]any{"type": "number"},
										"small":   map[string]any{"type": "number"},
										"caption": map[string]any{"type": "number"},
									},
									"required":             []string{"h1", "h2", "h3", "body", "small", "caption"},
									"additionalProperties": false,
								},
								"lineHeights": map[string]any{
									"type": "object",
									"properties": map[string]any{
										"tight":   map[string]any{"type": "number"},
										"normal":  map[string]any{"type": "number"},
										"relaxed": map[string]any{"type": "number"},
									},
									"required":             []string{"tight", "normal", "relaxed"},
									"additionalProperties": false,
								},
							},
							"required":             []string{"families", "scale", "lineHeights"},
							"additionalProperties": false,
						},
						"defaults": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"background": map[string]any{
									"type": "object",
									"properties": map[string]any{
										"type":  map[string]any{"type": "string"},
										"color": map[string]any{"type": "string"},
									},
									"required":             []string{"type", "color"},
									"additionalProperties": false,
								},
								"textStyle": map[string]any{
									"type": "object",
									"properties": map[string]any{
										"fontFamily": map[string]any{"type": "string"},
										"fontSize":   map[string]any{"type": "number"},
										"color":      map[string]any{"type": "string"},
										"align":      map[string]any{"type": "string"},
										"valign":     map[string]any{"type": "string"},
									},
									"required":             []string{"fontFamily", "fontSize", "color", "align", "valign"},
									"additionalProperties": false,
								},
								"shapeStyle": map[string]any{
									"type": "object",
									"properties": map[string]any{
										"fill": map[string]any{
											"type": "object",
											"properties": map[string]any{
												"type":  map[string]any{"type": "string"},
												"color": map[string]any{"type": "string"},
											},
											"required":             []string{"type", "color"},
											"additionalProperties": false,
										},
									},
									"required":             []string{"fill"},
									"additionalProperties": false,
								},
								"linkStyle": map[string]any{
									"type": "object",
									"properties": map[string]any{
										"color":     map[string]any{"type": "string"},
										"underline": map[string]any{"type": "boolean"},
									},
									"required":             []string{"color", "underline"},
									"additionalProperties": false,
								},
							},
							"required":             []string{"background", "textStyle", "shapeStyle", "linkStyle"},
							"additionalProperties": false,
						},
					},
					"required":             []string{"canvas", "grid", "colors", "typography", "defaults"},
					"additionalProperties": false,
				},
				"masters": map[string]any{
					"type":  "array",
					"items": map[string]any{"type": "object"},
				},
				"layouts": map[string]any{
					"type":  "array",
					"items": map[string]any{"type": "object"},
				},
				"components": map[string]any{
					"type":  "array",
					"items": map[string]any{"type": "object"},
				},
				"export": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"format":              map[string]any{"type": "string"},
						"fileName":            map[string]any{"type": "string"},
						"includeSpeakerNotes": map[string]any{"type": "boolean"},
					},
					"required":             []string{"format", "fileName", "includeSpeakerNotes"},
					"additionalProperties": false,
				},
			},
			"required":             []string{"version", "metadata", "theme", "masters", "layouts", "components", "export"},
			"additionalProperties": false,
		},
	},
	"required":             []string{"plan", "template"},
	"additionalProperties": false,
}

// SlidePlanSchema is the JSON schema for the plan-only response.
var SlidePlanSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"deckTitle": map[string]any{
			"type":        "string",
			"description": "The title of the presentation",
		},
		"audience": map[string]any{
			"type":        "string",
			"description": "Target audience description",
		},
		"tone": map[string]any{
			"type":        "string",
			"description": "Presentation tone (professional, casual, technical, etc.)",
		},
		"purpose": map[string]any{
			"type":        "string",
			"description": "Main purpose of the presentation",
		},
		"recommendedSlideCount": map[string]any{
			"type":        "integer",
			"description": "Recommended number of slides",
		},
		"slides": map[string]any{
			"type":        "array",
			"description": "Array of slide plan entries",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"index": map[string]any{
						"type":        "integer",
						"description": "1-based slide index",
					},
					"title": map[string]any{
						"type":        "string",
						"description": "Slide title",
					},
					"purpose": map[string]any{
						"type":        "string",
						"description": "Purpose of this slide",
					},
					"keyPoints": map[string]any{
						"type":        "array",
						"description": "Key points to cover",
						"items":       map[string]any{"type": "string"},
					},
					"suggestedLayout": map[string]any{
						"type":        "string",
						"description": "Suggested layout type",
						"enum":        []string{"TITLE", "SECTION_HEADER", "TITLE_AND_BULLETS", "TITLE_TWO_COLUMNS", "TITLE_IMAGE", "FULL_BLEED_IMAGE", "CHART", "TABLE", "QUOTE", "TIMELINE", "CLOSING", "APPENDIX", "DASHBOARD_3KPI_2COL", "CHART_AND_INSIGHTS", "TABLE_AND_CALLOUTS"},
					},
					"visualIdeas": map[string]any{
						"type":        "array",
						"description": "Visual element suggestions",
						"items":       map[string]any{"type": "string"},
					},
				},
				"required":             []string{"index", "title", "purpose", "keyPoints", "suggestedLayout", "visualIdeas"},
				"additionalProperties": false,
			},
		},
	},
	"required":             []string{"deckTitle", "audience", "tone", "purpose", "recommendedSlideCount", "slides"},
	"additionalProperties": false,
}

// SlideTemplateSchema is the JSON schema for the template-only response.
var SlideTemplateSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"version": map[string]any{
			"type":        "string",
			"description": "Template version (e.g., '1.0')",
		},
		"metadata": map[string]any{
			"type":        "object",
			"description": "Presentation metadata",
			"properties": map[string]any{
				"title":    map[string]any{"type": "string"},
				"language": map[string]any{"type": "string"},
				"audience": map[string]any{"type": "string"},
				"purpose":  map[string]any{"type": "string"},
			},
			"required":             []string{"title", "language", "audience", "purpose"},
			"additionalProperties": false,
		},
		"theme": map[string]any{
			"type":        "object",
			"description": "Theme configuration including canvas, grid, colors, typography",
			"properties": map[string]any{
				"canvas": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"size": map[string]any{
							"type":        "string",
							"description": "Canvas size preset. This generator assumes a normal PowerPoint WIDE_16x9 slide.",
							"enum":        []string{"WIDE_16x9"},
						},
						"customSize": map[string]any{
							"type":        "object",
							"description": "Optional custom size (in inches) when size=CUSTOM. Use null for WIDE_16x9.",
							"properties": map[string]any{
								"width":  map[string]any{"type": "number"},
								"height": map[string]any{"type": "number"},
							},
							"required":             []string{"width", "height"},
							"additionalProperties": false,
						},
						"unit": map[string]any{
							"type":        "string",
							"description": "Coordinate unit for all rects. Use pt so WIDE_16x9 becomes 960pt x 540pt.",
							"enum":        []string{"pt"},
						},
						"safeMargins": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"top":    map[string]any{"type": "number", "minimum": 0, "maximum": 200, "default": 36},
								"right":  map[string]any{"type": "number", "minimum": 0, "maximum": 200, "default": 36},
								"bottom": map[string]any{"type": "number", "minimum": 0, "maximum": 200, "default": 36},
								"left":   map[string]any{"type": "number", "minimum": 0, "maximum": 200, "default": 36},
							},
							"required":             []string{"top", "right", "bottom", "left"},
							"additionalProperties": false,
						},
					},
					"required":             []string{"size", "unit", "safeMargins"},
					"additionalProperties": false,
				},
				"grid": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"columns":  map[string]any{"type": "integer"},
						"gutter":   map[string]any{"type": "number"},
						"baseline": map[string]any{"type": "number"},
						"snap":     map[string]any{"type": "boolean"},
					},
					"required":             []string{"columns", "gutter", "baseline", "snap"},
					"additionalProperties": false,
				},
				"colors": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"palette": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"primary":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
								"secondary": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
								"neutral":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
								"accent":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
							},
							"required":             []string{"primary", "secondary", "neutral", "accent"},
							"additionalProperties": false,
						},
						"semantic": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"background": map[string]any{"type": "string"},
								"text":       map[string]any{"type": "string"},
								"mutedText":  map[string]any{"type": "string"},
								"border":     map[string]any{"type": "string"},
								"link":       map[string]any{"type": "string"},
								"success":    map[string]any{"type": "string"},
								"warning":    map[string]any{"type": "string"},
								"danger":     map[string]any{"type": "string"},
							},
							"required":             []string{"background", "text", "mutedText", "border", "link", "success", "warning", "danger"},
							"additionalProperties": false,
						},
					},
					"required":             []string{"palette", "semantic"},
					"additionalProperties": false,
				},
				"typography": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"families": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"heading": map[string]any{"type": "string"},
								"body":    map[string]any{"type": "string"},
								"mono":    map[string]any{"type": "string"},
							},
							"required":             []string{"heading", "body", "mono"},
							"additionalProperties": false,
						},
						"scale": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"h1":      map[string]any{"type": "number"},
								"h2":      map[string]any{"type": "number"},
								"h3":      map[string]any{"type": "number"},
								"body":    map[string]any{"type": "number"},
								"small":   map[string]any{"type": "number"},
								"caption": map[string]any{"type": "number"},
							},
							"required":             []string{"h1", "h2", "h3", "body", "small", "caption"},
							"additionalProperties": false,
						},
						"lineHeights": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"tight":   map[string]any{"type": "number"},
								"normal":  map[string]any{"type": "number"},
								"relaxed": map[string]any{"type": "number"},
							},
							"required":             []string{"tight", "normal", "relaxed"},
							"additionalProperties": false,
						},
					},
					"required":             []string{"families", "scale", "lineHeights"},
					"additionalProperties": false,
				},
				"defaults": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"background": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"type":  map[string]any{"type": "string"},
								"color": map[string]any{"type": "string"},
							},
							"required":             []string{"type", "color"},
							"additionalProperties": false,
						},
						"textStyle": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"fontFamily": map[string]any{"type": "string"},
								"fontSize":   map[string]any{"type": "number"},
								"color":      map[string]any{"type": "string"},
								"align":      map[string]any{"type": "string"},
								"valign":     map[string]any{"type": "string"},
							},
							"required":             []string{"fontFamily", "fontSize", "color", "align", "valign"},
							"additionalProperties": false,
						},
						"shapeStyle": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"fill": map[string]any{
									"type": "object",
									"properties": map[string]any{
										"type":  map[string]any{"type": "string"},
										"color": map[string]any{"type": "string"},
									},
									"required":             []string{"type", "color"},
									"additionalProperties": false,
								},
							},
							"required":             []string{"fill"},
							"additionalProperties": false,
						},
						"linkStyle": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"color":     map[string]any{"type": "string"},
								"underline": map[string]any{"type": "boolean"},
							},
							"required":             []string{"color", "underline"},
							"additionalProperties": false,
						},
					},
					"required":             []string{"background", "textStyle", "shapeStyle", "linkStyle"},
					"additionalProperties": false,
				},
			},
			"required":             []string{"canvas", "grid", "colors", "typography", "defaults"},
			"additionalProperties": false,
		},
		"masters": map[string]any{
			"type":  "array",
			"items": map[string]any{"type": "object"},
		},
		"layouts": map[string]any{
			"type":  "array",
			"items": map[string]any{"type": "object"},
		},
		"components": map[string]any{
			"type":  "array",
			"items": map[string]any{"type": "object"},
		},
		"export": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"format":              map[string]any{"type": "string"},
				"fileName":            map[string]any{"type": "string"},
				"includeSpeakerNotes": map[string]any{"type": "boolean"},
			},
			"required":             []string{"format", "fileName", "includeSpeakerNotes"},
			"additionalProperties": false,
		},
	},
	"required":             []string{"version", "metadata", "theme", "masters", "layouts", "components", "export"},
	"additionalProperties": false,
}

// SlideMetadataSchema is the JSON schema for slide metadata (id, order, title, layoutId, speakerNotes).
var SlideMetadataSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"id": map[string]any{
			"type":        "string",
			"description": "Unique slide identifier (e.g., 'slide_1')",
		},
		"order": map[string]any{
			"type":        "integer",
			"description": "1-based slide order",
		},
		"title": map[string]any{
			"type":        "string",
			"description": "Slide title",
		},
		"layoutId": map[string]any{
			"type":        "string",
			"description": "Reference to a layout ID from the template",
		},
		"speakerNotes": map[string]any{
			"type":        "string",
			"description": "Speaker notes (1-4 lines)",
		},
	},
	"required":             []string{"id", "order", "title", "layoutId", "speakerNotes"},
	"additionalProperties": false,
}

// TextElementSchema is the JSON schema for generating a text element.
var TextElementSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"element": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id": map[string]any{
					"type":        "string",
					"description": "Unique element identifier",
				},
				"type": map[string]any{
					"type":        "string",
					"description": "Must be 'text'",
					"enum":        []string{"text"},
				},
				"slotId": map[string]any{
					"type":        "string",
					"description": "Optional slot ID for layouts that define slots. Use slotId instead of rect when available.",
				},
				"rect": map[string]any{
					"type":        "object",
					"description": "Position and size (x, y, w, h) in pt. For WIDE_16x9: coordinate plane is 960pt x 540pt.",
					"properties": map[string]any{
						"x": map[string]any{"type": "number", "minimum": 0, "maximum": 960, "description": "X position"},
						"y": map[string]any{"type": "number", "minimum": 0, "maximum": 540, "description": "Y position"},
						"w": map[string]any{"type": "number", "exclusiveMinimum": 0, "maximum": 960, "description": "Width"},
						"h": map[string]any{"type": "number", "exclusiveMinimum": 0, "maximum": 540, "description": "Height"},
					},
					"required":             []string{"x", "y", "w", "h"},
					"additionalProperties": false,
				},
				"text": map[string]any{
					"type":        "object",
					"description": "Text content and style",
					"properties": map[string]any{
						"content": map[string]any{"type": "string"},
						"runs": map[string]any{
							"type":        "array",
							"description": "Optional rich text runs. Use an empty array when not needed.",
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"start": map[string]any{"type": "integer", "minimum": 0},
									"end":   map[string]any{"type": "integer", "minimum": 0},
									"style": map[string]any{
										"type":        "object",
										"description": "Run style. Same fields as text.style.",
										"properties": map[string]any{
											"fontFamily":    map[string]any{"type": "string"},
											"fontSize":      map[string]any{"type": "number", "minimum": 6},
											"bold":          map[string]any{"type": "boolean"},
											"italic":        map[string]any{"type": "boolean"},
											"underline":     map[string]any{"type": "boolean"},
											"color":         map[string]any{"type": "string"},
											"align":         map[string]any{"type": "string", "enum": []string{"left", "center", "right", "justify"}},
											"valign":        map[string]any{"type": "string", "enum": []string{"top", "middle", "bottom"}},
											"lineHeight":    map[string]any{"type": "number", "minimum": 0.8, "maximum": 2.5},
											"letterSpacing": map[string]any{"type": "number"},
											"bullet": map[string]any{
												"type": "object",
												"properties": map[string]any{
													"enabled": map[string]any{"type": "boolean"},
													"indent":  map[string]any{"type": "number", "minimum": 0},
													"hanging": map[string]any{"type": "number", "minimum": 0},
												},
												"required":             []string{"enabled"},
												"additionalProperties": false,
											},
										},
										"additionalProperties": false,
									},
								},
								"required":             []string{"start", "end", "style"},
								"additionalProperties": false,
							},
						},
						"autoFit": map[string]any{
							"type":        "string",
							"description": "Auto-fit behavior. Use 'shrink' to keep text inside its rect.",
							"enum":        []string{"shrink"},
						},
						"style": map[string]any{
							"type":        "object",
							"description": "Text styling - must match schema.json textStyle exactly",
							"properties": map[string]any{
								"fontFamily":    map[string]any{"type": "string"},
								"fontSize":      map[string]any{"type": "number", "minimum": 6},
								"bold":          map[string]any{"type": "boolean"},
								"italic":        map[string]any{"type": "boolean"},
								"underline":     map[string]any{"type": "boolean"},
								"color":         map[string]any{"type": "string", "description": "Hex color (#RRGGBB)"},
								"align":         map[string]any{"type": "string", "enum": []string{"left", "center", "right", "justify"}},
								"valign":        map[string]any{"type": "string", "enum": []string{"top", "middle", "bottom"}},
								"lineHeight":    map[string]any{"type": "number", "minimum": 0.8, "maximum": 2.5},
								"letterSpacing": map[string]any{"type": "number"},
								"bullet": map[string]any{
									"type":        "object",
									"description": "Bullets (indent/hanging in pt).",
									"properties": map[string]any{
										"enabled": map[string]any{"type": "boolean"},
										"indent":  map[string]any{"type": "number", "minimum": 0},
										"hanging": map[string]any{"type": "number", "minimum": 0},
									},
									"required":             []string{"enabled"},
									"additionalProperties": false,
								},
							},
							"additionalProperties": false,
						},
					},
					"required":             []string{"content", "runs", "autoFit", "style"},
					"additionalProperties": false,
				},
			},
			"required":             []string{"id", "type", "text"},
			"anyOf":                []any{map[string]any{"required": []string{"rect"}}, map[string]any{"required": []string{"slotId"}}},
			"additionalProperties": false,
		},
	},
	"required":             []string{"element"},
	"additionalProperties": false,
}

// ImageElementSchema is the JSON schema for generating an image element.
var ImageElementSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"element": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id": map[string]any{
					"type":        "string",
					"description": "Unique element identifier",
				},
				"type": map[string]any{
					"type":        "string",
					"description": "Must be 'image'",
					"enum":        []string{"image"},
				},
				"slotId": map[string]any{
					"type":        "string",
					"description": "Optional slot ID for layouts that define slots. Use slotId instead of rect when available.",
				},
				"rect": map[string]any{
					"type":        "object",
					"description": "Position and size (x, y, w, h) in pt. For WIDE_16x9: coordinate plane is 960pt x 540pt.",
					"properties": map[string]any{
						"x": map[string]any{"type": "number", "minimum": 0, "maximum": 960, "description": "X position"},
						"y": map[string]any{"type": "number", "minimum": 0, "maximum": 540, "description": "Y position"},
						"w": map[string]any{"type": "number", "exclusiveMinimum": 0, "maximum": 960, "description": "Width"},
						"h": map[string]any{"type": "number", "exclusiveMinimum": 0, "maximum": 540, "description": "Height"},
					},
					"required":             []string{"x", "y", "w", "h"},
					"additionalProperties": false,
				},
				"image": map[string]any{
					"type":        "object",
					"description": "Image reference",
					"properties": map[string]any{
						"ref":     map[string]any{"type": "string", "description": "Asset reference ID"},
						"fit":     map[string]any{"type": "string", "enum": []string{"cover", "contain", "stretch"}},
						"altText": map[string]any{"type": "string"},
					},
					"required":             []string{"ref"},
					"additionalProperties": false,
				},
			},
			"required":             []string{"id", "type", "image"},
			"anyOf":                []any{map[string]any{"required": []string{"rect"}}, map[string]any{"required": []string{"slotId"}}},
			"additionalProperties": false,
		},
		"asset": map[string]any{
			"type":        "object",
			"description": "Asset definition for this image",
			"properties": map[string]any{
				"id":   map[string]any{"type": "string"},
				"kind": map[string]any{"type": "string", "enum": []string{"image"}},
				"source": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"type":     map[string]any{"type": "string", "enum": []string{"url", "file", "base64"}},
						"url":      map[string]any{"type": "string"},
						"filePath": map[string]any{"type": "string"},
						"base64":   map[string]any{"type": "string"},
					},
					"required":             []string{"type"},
					"additionalProperties": false,
				},
				"altText":     map[string]any{"type": "string"},
				"license":     map[string]any{"type": "string"},
				"attribution": map[string]any{"type": "string"},
			},
			"required":             []string{"id", "kind", "source", "altText", "license", "attribution"},
			"additionalProperties": false,
		},
	},
	"required":             []string{"element", "asset"},
	"additionalProperties": false,
}

// ShapeElementSchema is the JSON schema for generating a shape element.
var ShapeElementSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"element": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id": map[string]any{
					"type":        "string",
					"description": "Unique element identifier",
				},
				"type": map[string]any{
					"type":        "string",
					"description": "Must be 'shape'",
					"enum":        []string{"shape"},
				},
				"slotId": map[string]any{
					"type":        "string",
					"description": "Optional slot ID for layouts that define slots. Use slotId instead of rect when available.",
				},
				"rect": map[string]any{
					"type":        "object",
					"description": "Position and size (x, y, w, h) in pt. For WIDE_16x9: coordinate plane is 960pt x 540pt.",
					"properties": map[string]any{
						"x": map[string]any{"type": "number", "minimum": 0, "maximum": 960, "description": "X position"},
						"y": map[string]any{"type": "number", "minimum": 0, "maximum": 540, "description": "Y position"},
						"w": map[string]any{"type": "number", "exclusiveMinimum": 0, "maximum": 960, "description": "Width"},
						"h": map[string]any{"type": "number", "exclusiveMinimum": 0, "maximum": 540, "description": "Height"},
					},
					"required":             []string{"x", "y", "w", "h"},
					"additionalProperties": false,
				},
				"shape": map[string]any{
					"type":        "object",
					"description": "Shape definition",
					"properties": map[string]any{
						"kind": map[string]any{
							"type":        "string",
							"description": "Supported shape kinds. Use 'rect' for containers; avoid rounded/oval shapes.",
							"enum":        []string{"rect", "line", "arrow", "triangle", "diamond"},
						},
						"style": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"fill": map[string]any{
									"type": "object",
									"properties": map[string]any{
										"type":  map[string]any{"type": "string", "enum": []string{"solid", "gradient", "none"}},
										"color": map[string]any{"type": "string", "description": "Hex color (#RRGGBB)"},
									},
									"required":             []string{"type", "color"},
									"additionalProperties": false,
								},
								"stroke": map[string]any{
									"type": "object",
									"properties": map[string]any{
										"color": map[string]any{"type": "string"},
										"width": map[string]any{"type": "number"},
									},
									"additionalProperties": false,
								},
								"cornerRadius": map[string]any{"type": "number"},
								"shadow":       map[string]any{"type": "object"},
							},
							"required":             []string{"fill"},
							"additionalProperties": false,
						},
					},
					"required":             []string{"kind", "style"},
					"additionalProperties": false,
				},
			},
			"required":             []string{"id", "type", "shape"},
			"anyOf":                []any{map[string]any{"required": []string{"rect"}}, map[string]any{"required": []string{"slotId"}}},
			"additionalProperties": false,
		},
	},
	"required":             []string{"element"},
	"additionalProperties": false,
}

// TableElementSchema is the JSON schema for generating a table element.
var TableElementSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"element": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id": map[string]any{
					"type":        "string",
					"description": "Unique element identifier",
				},
				"type": map[string]any{
					"type":        "string",
					"description": "Must be 'table'",
					"enum":        []string{"table"},
				},
				"slotId": map[string]any{
					"type":        "string",
					"description": "Optional slot ID for layouts that define slots. Use slotId instead of rect when available.",
				},
				"rect": map[string]any{
					"type":        "object",
					"description": "Position and size (x, y, w, h) in pt. For WIDE_16x9: coordinate plane is 960pt x 540pt.",
					"properties": map[string]any{
						"x": map[string]any{"type": "number", "minimum": 0, "maximum": 960, "description": "X position"},
						"y": map[string]any{"type": "number", "minimum": 0, "maximum": 540, "description": "Y position"},
						"w": map[string]any{"type": "number", "exclusiveMinimum": 0, "maximum": 960, "description": "Width"},
						"h": map[string]any{"type": "number", "exclusiveMinimum": 0, "maximum": 540, "description": "Height"},
					},
					"required":             []string{"x", "y", "w", "h"},
					"additionalProperties": false,
				},
				"table": map[string]any{
					"type":        "object",
					"description": "Table definition",
					"properties": map[string]any{
						"columns": map[string]any{
							"type":        "array",
							"description": "Table column headers",
							"items":       map[string]any{"type": "string"},
						},
						"rows": map[string]any{
							"type":        "array",
							"description": "Table rows",
							"items": map[string]any{
								"type":  "array",
								"items": map[string]any{"type": "string"},
							},
						},
						"style": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"headerTextStyle": map[string]any{"type": "object"},
								"cellTextStyle":   map[string]any{"type": "object"},
							},
							"additionalProperties": false,
						},
					},
					"required":             []string{"columns", "rows"},
					"additionalProperties": false,
				},
			},
			"required":             []string{"id", "type", "table"},
			"anyOf":                []any{map[string]any{"required": []string{"rect"}}, map[string]any{"required": []string{"slotId"}}},
			"additionalProperties": false,
		},
	},
	"required":             []string{"element"},
	"additionalProperties": false,
}

// ChartElementSchema is the JSON schema for generating a chart element.
var ChartElementSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"element": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id": map[string]any{
					"type":        "string",
					"description": "Unique element identifier",
				},
				"type": map[string]any{
					"type":        "string",
					"description": "Must be 'chart'",
					"enum":        []string{"chart"},
				},
				"slotId": map[string]any{
					"type":        "string",
					"description": "Optional slot ID for layouts that define slots. Use slotId instead of rect when available.",
				},
				"rect": map[string]any{
					"type":        "object",
					"description": "Position and size (x, y, w, h) in pt. For WIDE_16x9: coordinate plane is 960pt x 540pt.",
					"properties": map[string]any{
						"x": map[string]any{"type": "number", "minimum": 0, "maximum": 960, "description": "X position"},
						"y": map[string]any{"type": "number", "minimum": 0, "maximum": 540, "description": "Y position"},
						"w": map[string]any{"type": "number", "exclusiveMinimum": 0, "maximum": 960, "description": "Width"},
						"h": map[string]any{"type": "number", "exclusiveMinimum": 0, "maximum": 540, "description": "Height"},
					},
					"required":             []string{"x", "y", "w", "h"},
					"additionalProperties": false,
				},
				"chart": map[string]any{
					"type":        "object",
					"description": "Chart definition",
					"properties": map[string]any{
						"chartType":  map[string]any{"type": "string", "enum": []string{"bar", "line", "pie"}},
						"datasetRef": map[string]any{"type": "string", "description": "Reference to dataset ID"},
					},
					"required":             []string{"chartType", "datasetRef"},
					"additionalProperties": false,
				},
			},
			"required":             []string{"id", "type", "chart"},
			"anyOf":                []any{map[string]any{"required": []string{"rect"}}, map[string]any{"required": []string{"slotId"}}},
			"additionalProperties": false,
		},
		"dataset": map[string]any{
			"type":        "object",
			"description": "Complete dataset definition for this chart",
			"properties": map[string]any{
				"id": map[string]any{
					"type":        "string",
					"description": "Unique dataset identifier (e.g., 'dataset_gdp_2025')",
				},
				"kind": map[string]any{
					"type":        "string",
					"description": "Dataset type",
					"enum":        []string{"series"},
				},
				"data": map[string]any{
					"type":        "object",
					"description": "Dataset values",
					"properties": map[string]any{
						"labels": map[string]any{
							"type":        "array",
							"description": "Category labels (x-axis)",
							"items":       map[string]any{"type": "string"},
						},
						"series": map[string]any{
							"type":        "array",
							"description": "Data series",
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"name":   map[string]any{"type": "string"},
									"values": map[string]any{"type": "array", "items": map[string]any{"type": "number"}},
								},
								"required":             []string{"name", "values"},
								"additionalProperties": false,
							},
						},
					},
					"required":             []string{"labels", "series"},
					"additionalProperties": false,
				},
				"sourceNote": map[string]any{
					"type":        "string",
					"description": "Source attribution (optional)",
				},
			},
			"required":             []string{"id", "kind", "data"},
			"additionalProperties": false,
		},
	},
	"required":             []string{"element", "dataset"},
	"additionalProperties": false,
}

// GroupElementSchema is the JSON schema for generating a group element.
var GroupElementSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"element": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id": map[string]any{
					"type":        "string",
					"description": "Unique element identifier",
				},
				"type": map[string]any{
					"type":        "string",
					"description": "Must be 'group'",
					"enum":        []string{"group"},
				},
				"rect": map[string]any{
					"type":        "object",
					"description": "Position and size (x, y, w, h) in pt. For WIDE_16x9: coordinate plane is 960pt x 540pt.",
					"properties": map[string]any{
						"x": map[string]any{"type": "number", "minimum": 0, "maximum": 960, "description": "X position"},
						"y": map[string]any{"type": "number", "minimum": 0, "maximum": 540, "description": "Y position"},
						"w": map[string]any{"type": "number", "exclusiveMinimum": 0, "maximum": 960, "description": "Width"},
						"h": map[string]any{"type": "number", "exclusiveMinimum": 0, "maximum": 540, "description": "Height"},
					},
					"required":             []string{"x", "y", "w", "h"},
					"additionalProperties": false,
				},
				"group": map[string]any{
					"type":        "object",
					"description": "Group definition",
					"properties": map[string]any{
						"children": map[string]any{
							"type":        "array",
							"description": "Child elements with relative positioning (0..1). Each child must have a 'rel' field instead of 'rect'.",
							"items":       map[string]any{"type": "object"},
						},
					},
					"required":             []string{"children"},
					"additionalProperties": false,
				},
			},
			"required":             []string{"id", "type", "rect", "group"},
			"additionalProperties": false,
		},
	},
	"required":             []string{"element"},
	"additionalProperties": false,
}

// SlideRequirementsSchema is the JSON schema for collecting slide requirements (assets and datasets).
var SlideRequirementsSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"assets": map[string]any{
			"type":        "array",
			"description": "Asset objects required",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":   map[string]any{"type": "string"},
					"kind": map[string]any{"type": "string", "enum": []string{"image"}},
					"source": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"type":     map[string]any{"type": "string", "enum": []string{"url", "file", "base64"}},
							"url":      map[string]any{"type": "string"},
							"filePath": map[string]any{"type": "string"},
							"base64":   map[string]any{"type": "string"},
						},
						"required":             []string{"type"},
						"additionalProperties": false,
					},
					"altText":     map[string]any{"type": "string"},
					"license":     map[string]any{"type": "string"},
					"attribution": map[string]any{"type": "string"},
				},
				"required":             []string{"id", "kind", "source", "altText", "license", "attribution"},
				"additionalProperties": false,
			},
		},
		"datasets": map[string]any{
			"type":        "array",
			"description": "Complete dataset objects with id, kind, data, and optional sourceNote",
			"items": map[string]any{
				"type":        "object",
				"description": "Complete dataset definition",
				"properties": map[string]any{
					"id": map[string]any{
						"type":        "string",
						"description": "Unique dataset identifier (e.g., 'dataset_gdp_2025')",
					},
					"kind": map[string]any{
						"type":        "string",
						"description": "Dataset type",
						"enum":        []string{"series"},
					},
					"data": map[string]any{
						"type":        "object",
						"description": "Dataset values",
						"properties": map[string]any{
							"labels": map[string]any{
								"type":        "array",
								"description": "Category labels (x-axis)",
								"items":       map[string]any{"type": "string"},
							},
							"series": map[string]any{
								"type":        "array",
								"description": "Data series",
								"items": map[string]any{
									"type": "object",
									"properties": map[string]any{
										"name":   map[string]any{"type": "string"},
										"values": map[string]any{"type": "array", "items": map[string]any{"type": "number"}},
									},
									"required":             []string{"name", "values"},
									"additionalProperties": false,
								},
							},
						},
						"required":             []string{"labels", "series"},
						"additionalProperties": false,
					},
					"sourceNote": map[string]any{
						"type":        "string",
						"description": "Source attribution (optional)",
					},
				},
				"required":             []string{"id", "kind", "data"},
				"additionalProperties": false,
			},
		},
	},
	"required":             []string{"assets", "datasets"},
	"additionalProperties": false,
}

// SlideGenResultSchema is the JSON schema for individual slide generation (DEPRECATED - use element-specific schemas).
var SlideGenResultSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"slide": map[string]any{
			"type":        "object",
			"description": "The generated slide object",
			"properties": map[string]any{
				"id": map[string]any{
					"type":        "string",
					"description": "Unique slide identifier (e.g., 'slide_1')",
				},
				"order": map[string]any{
					"type":        "integer",
					"description": "1-based slide order",
				},
				"title": map[string]any{
					"type":        "string",
					"description": "Slide title",
				},
				"layoutId": map[string]any{
					"type":        "string",
					"description": "Reference to a layout ID from the template",
				},
				"speakerNotes": map[string]any{
					"type":        "string",
					"description": "Speaker notes (1-4 lines)",
				},
				"elements": map[string]any{
					"type":        "array",
					"description": "Slide elements (text, images, shapes, etc.)",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"id": map[string]any{
								"type":        "string",
								"description": "Unique element identifier",
							},
							"type": map[string]any{
								"type":        "string",
								"description": "Element type",
								"enum":        []string{"text", "image", "shape", "table", "chart", "group"},
							},
							"slotId": map[string]any{
								"type":        "string",
								"description": "Optional slot ID for layouts that define slots. Use slotId instead of rect when available.",
							},
							"rect": map[string]any{
								"type":        "object",
								"description": "Position and size (x, y, w, h) in pt. For WIDE_16x9: coordinate plane is 960pt x 540pt.",
								"properties": map[string]any{
									"x": map[string]any{"type": "number", "minimum": 0, "maximum": 960, "description": "X position"},
									"y": map[string]any{"type": "number", "minimum": 0, "maximum": 540, "description": "Y position"},
									"w": map[string]any{"type": "number", "exclusiveMinimum": 0, "maximum": 960, "description": "Width"},
									"h": map[string]any{"type": "number", "exclusiveMinimum": 0, "maximum": 540, "description": "Height"},
								},
								"required":             []string{"x", "y", "w", "h"},
								"additionalProperties": false,
							},
							"rel": map[string]any{
								"type":        "object",
								"description": "Relative placement (0..1) when inside a group. Only used for group children.",
								"properties": map[string]any{
									"x": map[string]any{"type": "number", "minimum": 0, "maximum": 1},
									"y": map[string]any{"type": "number", "minimum": 0, "maximum": 1},
									"w": map[string]any{"type": "number", "exclusiveMinimum": 0, "maximum": 1},
									"h": map[string]any{"type": "number", "exclusiveMinimum": 0, "maximum": 1},
								},
								"required":             []string{"x", "y", "w", "h"},
								"additionalProperties": false,
							},
							"text": map[string]any{
								"type":        "object",
								"description": "Text content and style (for type='text')",
								"properties": map[string]any{
									"content": map[string]any{"type": "string"},
									"runs": map[string]any{
										"type":        "array",
										"description": "Optional rich text runs. Use an empty array when not needed.",
										"items": map[string]any{
											"type": "object",
											"properties": map[string]any{
												"start": map[string]any{"type": "integer", "minimum": 0},
												"end":   map[string]any{"type": "integer", "minimum": 0},
												"style": map[string]any{
													"type":        "object",
													"description": "Run style. Same fields as text.style.",
													"properties": map[string]any{
														"fontFamily":    map[string]any{"type": "string"},
														"fontSize":      map[string]any{"type": "number", "minimum": 6},
														"bold":          map[string]any{"type": "boolean"},
														"italic":        map[string]any{"type": "boolean"},
														"underline":     map[string]any{"type": "boolean"},
														"color":         map[string]any{"type": "string"},
														"align":         map[string]any{"type": "string", "enum": []string{"left", "center", "right", "justify"}},
														"valign":        map[string]any{"type": "string", "enum": []string{"top", "middle", "bottom"}},
														"lineHeight":    map[string]any{"type": "number", "minimum": 0.8, "maximum": 2.5},
														"letterSpacing": map[string]any{"type": "number"},
														"bullet": map[string]any{
															"type": "object",
															"properties": map[string]any{
																"enabled": map[string]any{"type": "boolean"},
																"indent":  map[string]any{"type": "number", "minimum": 0},
																"hanging": map[string]any{"type": "number", "minimum": 0},
															},
															"required":             []string{"enabled"},
															"additionalProperties": false,
														},
													},
													"additionalProperties": false,
												},
											},
											"required":             []string{"start", "end", "style"},
											"additionalProperties": false,
										},
									},
									"autoFit": map[string]any{
										"type":        "string",
										"description": "Auto-fit behavior. Use 'shrink' to keep text inside its rect.",
										"enum":        []string{"shrink"},
									},
									"style": map[string]any{
										"type":        "object",
										"description": "Text styling - must match schema.json textStyle exactly",
										"properties": map[string]any{
											"fontFamily":    map[string]any{"type": "string"},
											"fontSize":      map[string]any{"type": "number", "minimum": 6},
											"bold":          map[string]any{"type": "boolean"},
											"italic":        map[string]any{"type": "boolean"},
											"underline":     map[string]any{"type": "boolean"},
											"color":         map[string]any{"type": "string", "description": "Hex color (#RRGGBB)"},
											"align":         map[string]any{"type": "string", "enum": []string{"left", "center", "right", "justify"}},
											"valign":        map[string]any{"type": "string", "enum": []string{"top", "middle", "bottom"}},
											"lineHeight":    map[string]any{"type": "number", "minimum": 0.8, "maximum": 2.5},
											"letterSpacing": map[string]any{"type": "number"},
											"bullet": map[string]any{
												"type":        "object",
												"description": "Bullets (indent/hanging in pt).",
												"properties": map[string]any{
													"enabled": map[string]any{"type": "boolean"},
													"indent":  map[string]any{"type": "number", "minimum": 0},
													"hanging": map[string]any{"type": "number", "minimum": 0},
												},
												"required":             []string{"enabled"},
												"additionalProperties": false,
											},
										},
										"additionalProperties": false,
									},
								},
								"required":             []string{"content", "runs", "autoFit", "style"},
								"additionalProperties": false,
							},
							"image": map[string]any{
								"type":        "object",
								"description": "Image reference (for type='image')",
								"properties": map[string]any{
									"ref":     map[string]any{"type": "string", "description": "Asset reference ID"},
									"fit":     map[string]any{"type": "string", "enum": []string{"cover", "contain", "stretch"}},
									"altText": map[string]any{"type": "string"},
								},
								"required":             []string{"ref"},
								"additionalProperties": false,
							},
							"shape": map[string]any{
								"type":        "object",
								"description": "Shape definition (for type='shape')",
								"properties": map[string]any{
									"kind": map[string]any{
										"type":        "string",
										"description": "Supported shape kinds. Use 'rect' for containers; avoid rounded/oval shapes.",
										"enum":        []string{"rect", "line", "arrow", "triangle", "diamond"},
									},
									"style": map[string]any{
										"type": "object",
										"properties": map[string]any{
											"fill": map[string]any{
												"type": "object",
												"properties": map[string]any{
													"type":  map[string]any{"type": "string", "enum": []string{"solid", "gradient", "none"}},
													"color": map[string]any{"type": "string", "description": "Hex color (#RRGGBB)"},
												},
												"required":             []string{"type", "color"},
												"additionalProperties": false,
											},
											"stroke": map[string]any{
												"type": "object",
												"properties": map[string]any{
													"color": map[string]any{"type": "string"},
													"width": map[string]any{"type": "number"},
												},
												"additionalProperties": false,
											},
											"cornerRadius": map[string]any{"type": "number"},
											"shadow":       map[string]any{"type": "object"},
										},
										"required":             []string{"fill"},
										"additionalProperties": false,
									},
								},
								"required":             []string{"kind", "style"},
								"additionalProperties": false,
							},
							"table": map[string]any{
								"type":        "object",
								"description": "Table definition (for type='table')",
								"properties": map[string]any{
									"columns": map[string]any{
										"type":        "array",
										"description": "Table column headers",
										"items":       map[string]any{"type": "string"},
									},
									"rows": map[string]any{
										"type":        "array",
										"description": "Table rows",
										"items": map[string]any{
											"type":  "array",
											"items": map[string]any{"type": "string"},
										},
									},
									"style": map[string]any{
										"type": "object",
										"properties": map[string]any{
											"headerTextStyle": map[string]any{"type": "object"},
											"cellTextStyle":   map[string]any{"type": "object"},
										},
										"additionalProperties": false,
									},
								},
								"required":             []string{"columns", "rows"},
								"additionalProperties": false,
							},
							"chart": map[string]any{
								"type":        "object",
								"description": "Chart definition (for type='chart')",
								"properties": map[string]any{
									"chartType":  map[string]any{"type": "string", "enum": []string{"bar", "line", "pie"}},
									"datasetRef": map[string]any{"type": "string", "description": "Reference to dataset ID"},
								},
								"required":             []string{"chartType", "datasetRef"},
								"additionalProperties": false,
							},
							"group": map[string]any{
								"type":        "object",
								"description": "Group definition (for type='group')",
								"properties": map[string]any{
									"children": map[string]any{
										"type":  "array",
										"items": map[string]any{"type": "object"},
									},
								},
								"required":             []string{"children"},
								"additionalProperties": true,
							},
						},
						"required":             []string{"id", "type"},
						"anyOf":                []any{map[string]any{"required": []string{"rect"}}, map[string]any{"required": []string{"slotId"}}},
						"additionalProperties": true,
					},
				},
				"useComponents": map[string]any{
					"type":        "array",
					"description": "Component IDs to apply from template.components",
					"items":       map[string]any{"type": "string"},
				},
			},
			"required":             []string{"id", "order", "title", "layoutId", "speakerNotes", "elements", "useComponents"},
			"additionalProperties": false,
		},
		"requires": map[string]any{
			"type":        "object",
			"description": "Assets and datasets required by this slide",
			"properties": map[string]any{
				"assets": map[string]any{
					"type":        "array",
					"description": "Asset objects required",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"id":   map[string]any{"type": "string"},
							"kind": map[string]any{"type": "string", "enum": []string{"image"}},
							"source": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"type":     map[string]any{"type": "string", "enum": []string{"url", "file", "base64"}},
									"url":      map[string]any{"type": "string"},
									"filePath": map[string]any{"type": "string"},
									"base64":   map[string]any{"type": "string"},
								},
								"required":             []string{"type"},
								"additionalProperties": false,
							},
							"altText":     map[string]any{"type": "string"},
							"license":     map[string]any{"type": "string"},
							"attribution": map[string]any{"type": "string"},
						},
						"required":             []string{"id", "kind", "source", "altText", "license", "attribution"},
						"additionalProperties": false,
					},
				},
				"datasets": map[string]any{
					"type":        "array",
					"description": "Complete dataset objects with id, kind, data, and optional sourceNote. DO NOT use string references.",
					"items": map[string]any{
						"type":        "object",
						"description": "Complete dataset definition",
						"properties": map[string]any{
							"id": map[string]any{
								"type":        "string",
								"description": "Unique dataset identifier (e.g., 'dataset_gdp_2025')",
							},
							"kind": map[string]any{
								"type":        "string",
								"description": "Dataset type",
								"enum":        []string{"series"},
							},
							"data": map[string]any{
								"type":        "object",
								"description": "Dataset values",
								"properties": map[string]any{
									"labels": map[string]any{
										"type":        "array",
										"description": "Category labels (x-axis)",
										"items":       map[string]any{"type": "string"},
									},
									"series": map[string]any{
										"type":        "array",
										"description": "Data series",
										"items": map[string]any{
											"type": "object",
											"properties": map[string]any{
												"name":   map[string]any{"type": "string"},
												"values": map[string]any{"type": "array", "items": map[string]any{"type": "number"}},
											},
											"required":             []string{"name", "values"},
											"additionalProperties": false,
										},
									},
								},
								"required":             []string{"labels", "series"},
								"additionalProperties": false,
							},
							"sourceNote": map[string]any{
								"type":        "string",
								"description": "Source attribution (optional)",
							},
						},
						"required":             []string{"id", "kind", "data"},
						"additionalProperties": false,
					},
				},
			},
			"required":             []string{"assets", "datasets"},
			"additionalProperties": false,
		},
	},
	"required":             []string{"slide", "requires"},
	"additionalProperties": false,
}

// IssuesReportSchema is the JSON schema for the deck validator response.
var IssuesReportSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"issues": map[string]any{
			"type":        "array",
			"description": "List of semantic issues found in the deck",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"severity": map[string]any{
						"type":        "string",
						"description": "Issue severity",
						"enum":        []string{"warn", "error"},
					},
					"message": map[string]any{
						"type":        "string",
						"description": "Description of the issue",
					},
					"slideIndex": map[string]any{
						"type":        "integer",
						"description": "1-based index of the affected slide (0 for deck-wide issues)",
					},
					"suggestedFix": map[string]any{
						"type":        "string",
						"description": "Suggested fix for the issue",
					},
				},
				"required":             []string{"severity", "message", "slideIndex", "suggestedFix"},
				"additionalProperties": false,
			},
		},
	},
	"required":             []string{"issues"},
	"additionalProperties": false,
}

// DataBankSchema is the JSON schema for extracted facts and datasets.
var DataBankSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"facts": map[string]any{
			"type":        "array",
			"description": "Atomic facts with sources",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"claim":     map[string]any{"type": "string"},
					"value":     map[string]any{"type": "string"},
					"unit":      map[string]any{"type": "string"},
					"sourceUrl": map[string]any{"type": "string"},
					"date":      map[string]any{"type": "string"},
				},
				"required":             []string{"claim", "value", "unit", "sourceUrl", "date"},
				"additionalProperties": false,
			},
		},
		"datasets": map[string]any{
			"type":        "array",
			"description": "Ready-to-use datasets for charts/tables",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":   map[string]any{"type": "string"},
					"kind": map[string]any{"type": "string", "enum": []string{"series"}},
					"data": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"labels": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
							"series": map[string]any{
								"type": "array",
								"items": map[string]any{
									"type": "object",
									"properties": map[string]any{
										"name":   map[string]any{"type": "string"},
										"values": map[string]any{"type": "array", "items": map[string]any{"type": "number"}},
									},
									"required":             []string{"name", "values"},
									"additionalProperties": false,
								},
							},
						},
						"required":             []string{"labels", "series"},
						"additionalProperties": false,
					},
					"sourceNote": map[string]any{"type": "string"},
				},
				"required":             []string{"id", "kind", "data", "sourceNote"},
				"additionalProperties": false,
			},
		},
	},
	"required":             []string{"facts", "datasets"},
	"additionalProperties": false,
}

// GetRecommendedTemperature returns the optimal temperature for a given schema.
// Lower temps (0.1-0.3) for deterministic structure, higher (0.6-0.8) for creative content.
func GetRecommendedTemperature(schema map[string]any) float64 {
	// Identify schema by checking unique properties
	if schema == nil {
		return 0.2 // default
	}

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		return 0.2 // default
	}

	// Check for "plan" and "template" - PlanAndTemplateSchema
	if _, hasPlan := props["plan"]; hasPlan {
		if _, hasTemplate := props["template"]; hasTemplate {
			return 0.5 // Balanced - needs both structure and creativity
		}
	}

	// Check for "deckTitle" - SlidePlanSchema
	if _, hasDeckTitle := props["deckTitle"]; hasDeckTitle {
		if _, hasSlides := props["slides"]; hasSlides {
			return 0.7 // High creativity for planning
		}
	}

	// Check for "version" and "metadata" - SlideTemplateSchema
	if _, hasVersion := props["version"]; hasVersion {
		if _, hasMetadata := props["metadata"]; hasMetadata {
			return 0.2 // Low - structural template
		}
	}

	// Check for "element" - Element schemas
	if elemProp, hasElement := props["element"]; hasElement {
		if elemMap, ok := elemProp.(map[string]any); ok {
			if elemProps, ok := elemMap["properties"].(map[string]any); ok {
				// Check element type
				if typeProp, hasType := elemProps["type"]; hasType {
					if typeMap, ok := typeProp.(map[string]any); ok {
						if enum, ok := typeMap["enum"].([]string); ok && len(enum) > 0 {
							switch enum[0] {
							case "text":
								return 0.7 // High creativity for text content
							case "image":
								return 0.5 // Medium - needs creative asset selection
							case "chart":
								return 0.3 // Low - data-driven
							case "table":
								return 0.3 // Low - data-driven
							case "shape":
								return 0.2 // Low - geometric/structural
							case "group":
								return 0.2 // Low - structural positioning
							}
						}
					}
				}
			}
		}
	}

	// Check for "id" and "order" - SlideMetadataSchema
	if _, hasID := props["id"]; hasID {
		if _, hasOrder := props["order"]; hasOrder {
			if _, hasLayoutId := props["layoutId"]; hasLayoutId {
				return 0.6 // Medium-high for titles and speaker notes
			}
		}
	}

	// Check for "slide" and "requires" - SlideGenResultSchema (deprecated)
	if _, hasSlide := props["slide"]; hasSlide {
		if _, hasRequires := props["requires"]; hasRequires {
			return 0.5 // Balanced
		}
	}

	// Check for "issues" - IssuesReportSchema
	if issuesProp, hasIssues := props["issues"]; hasIssues {
		if issuesMap, ok := issuesProp.(map[string]any); ok {
			if issuesMap["type"] == "array" {
				return 0.3 // Low - analytical
			}
		}
	}

	// Check for "facts" and "datasets" - DataBankSchema
	if _, hasFacts := props["facts"]; hasFacts {
		if _, hasDatasets := props["datasets"]; hasDatasets {
			return 0.1 // Very low - factual extraction
		}
	}

	// Check for "assets" and "datasets" - SlideRequirementsSchema
	if _, hasAssets := props["assets"]; hasAssets {
		if _, hasDatasets := props["datasets"]; hasDatasets {
			return 0.2 // Low - structural collection
		}
	}

	return 0.2 // default fallback
}

// NormalizeSchemaForStructuredOutput updates a schema to satisfy structured output requirements.
func NormalizeSchemaForStructuredOutput(schema any) {
	switch typed := schema.(type) {
	case map[string]any:
		delete(typed, "allOf")
		if isObjectSchema(typed) {
			props, _ := typed["properties"].(map[string]any)
			if props == nil {
				props = map[string]any{}
				typed["properties"] = props
			}

			requiredSet := parseRequiredSet(typed["required"])
			for key, propSchema := range props {
				if !requiredSet[key] {
					props[key] = allowNull(propSchema)
				}
			}

			requiredList := make([]string, 0, len(props))
			for key := range props {
				requiredList = append(requiredList, key)
			}
			sort.Strings(requiredList)
			typed["required"] = requiredList

			typed["additionalProperties"] = false
		}

		for _, value := range typed {
			NormalizeSchemaForStructuredOutput(value)
		}
	case []any:
		for _, value := range typed {
			NormalizeSchemaForStructuredOutput(value)
		}
	}
}

func isObjectSchema(schema map[string]any) bool {
	if schema == nil {
		return false
	}
	if t, ok := schema["type"].(string); ok {
		return t == "object"
	}
	if t, ok := schema["type"].([]any); ok {
		for _, item := range t {
			if s, ok := item.(string); ok && s == "object" {
				return true
			}
		}
	}
	return false
}

func parseRequiredSet(required any) map[string]bool {
	set := map[string]bool{}
	switch typed := required.(type) {
	case []string:
		for _, item := range typed {
			set[item] = true
		}
	case []any:
		for _, item := range typed {
			if s, ok := item.(string); ok {
				set[s] = true
			}
		}
	}
	return set
}

func allowNull(schema any) any {
	if allowsNull(schema) {
		return schema
	}
	return map[string]any{
		"anyOf": []any{
			schema,
			map[string]any{"type": "null"},
		},
	}
}

func allowsNull(schema any) bool {
	typed, ok := schema.(map[string]any)
	if !ok {
		return false
	}

	if t, ok := typed["type"].(string); ok && t == "null" {
		return true
	}
	if t, ok := typed["type"].([]any); ok {
		for _, item := range t {
			if s, ok := item.(string); ok && s == "null" {
				return true
			}
		}
	}
	if enumVals, ok := typed["enum"].([]any); ok {
		for _, item := range enumVals {
			if item == nil {
				return true
			}
		}
	}
	for _, key := range []string{"anyOf", "oneOf", "allOf"} {
		if list, ok := typed[key].([]any); ok {
			for _, sub := range list {
				if allowsNull(sub) {
					return true
				}
			}
		}
	}
	return false
}
