package schemas

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
