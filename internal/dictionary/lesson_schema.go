package dictionary

var learningGuideSchema = map[string]any{
	"type": "object", "additionalProperties": false,
	"required": []string{"learning"},
	"properties": map[string]any{"learning": map[string]any{
		"type": "object", "additionalProperties": false,
		"required": []string{"meaning", "summary", "grammar_note", "formation_rules", "tables", "patterns", "examples"},
		"properties": map[string]any{
			"meaning": map[string]any{"type": "string"}, "summary": map[string]any{"type": "string"}, "grammar_note": map[string]any{"type": "string"},
			"formation_rules": map[string]any{"type": "array", "maxItems": 6, "items": map[string]any{"type": "string"}},
			"tables": map[string]any{"type": "array", "maxItems": 4, "items": map[string]any{
				"type": "object", "additionalProperties": false, "required": []string{"title", "note", "forms"},
				"properties": map[string]any{
					"title": map[string]any{"type": "string"}, "note": map[string]any{"type": "string"},
					"forms": map[string]any{"type": "array", "maxItems": 6, "items": map[string]any{
						"type": "object", "additionalProperties": false, "required": []string{"label", "value"},
						"properties": map[string]any{"label": map[string]any{"type": "string"}, "value": map[string]any{"type": "string"}},
					}},
				},
			}},
			"patterns": map[string]any{"type": "array", "maxItems": 5, "items": map[string]any{"type": "string"}},
			"examples": map[string]any{"type": "array", "minItems": 4, "maxItems": 6, "items": map[string]any{
				"type": "object", "additionalProperties": false, "required": []string{"italian", "english"},
				"properties": map[string]any{"italian": map[string]any{"type": "string"}, "english": map[string]any{"type": "string"}},
			}},
		},
	}},
}
