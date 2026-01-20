package schemas

// MinifySchemaForLLM removes verbose, non-validating keys from a JSON schema.
//
// Motivation:
// - Many LLM providers count the JSON schema toward prompt/input tokens.
// - Large schemas increase latency/cost and can reduce adherence.
//
// This function is intentionally conservative: it only removes purely
// descriptive/annotative keys that do not affect JSON validation.
func MinifySchemaForLLM(schema any) {
	switch typed := schema.(type) {
	case map[string]any:
		// Pure annotations (do not affect validation)
		delete(typed, "description")
		delete(typed, "title")
		delete(typed, "examples")
		delete(typed, "$comment")
		delete(typed, "comment")
		delete(typed, "markdownDescription")
		delete(typed, "deprecated")

		for _, value := range typed {
			MinifySchemaForLLM(value)
		}
	case []any:
		for _, value := range typed {
			MinifySchemaForLLM(value)
		}
	}
}
