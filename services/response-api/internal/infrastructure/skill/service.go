package skill

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	domain "jan-server/services/response-api/internal/domain/skill"
)

//go:embed templates/*.py.tmpl
var templatesFS embed.FS

type Service struct {
	templates map[domain.SkillType]*template.Template
}

// NewService creates a skill service with embedded templates.
func NewService() (*Service, error) {
	templates := make(map[domain.SkillType]*template.Template)
	mapping := map[domain.SkillType]string{
		domain.SkillTypeSlides:       "templates/slides.py.tmpl",
		domain.SkillTypeDocs:         "templates/docs.py.tmpl",
		domain.SkillTypePDFs:         "templates/pdfs.py.tmpl",
		domain.SkillTypeSpreadsheets: "templates/spreadsheets.py.tmpl",
	}

	for skillType, file := range mapping {
		content, err := templatesFS.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("read template %s: %w", file, err)
		}
		tmpl, err := template.New(string(skillType)).Parse(string(content))
		if err != nil {
			return nil, fmt.Errorf("parse template %s: %w", file, err)
		}
		templates[skillType] = tmpl
	}

	return &Service{templates: templates}, nil
}

// GenerateCode renders a python template for the requested skill.
func (s *Service) GenerateCode(ctx context.Context, req domain.GenerateCodeRequest) (string, error) {
	tmpl, ok := s.templates[req.SkillType]
	if !ok {
		return "", fmt.Errorf("unsupported skill type: %s", req.SkillType)
	}

	content := normalizeContent(req.Content)
	options := req.Options
	if options == nil {
		options = map[string]interface{}{}
	}

	contentJSON, err := json.Marshal(content)
	if err != nil {
		return "", fmt.Errorf("marshal content: %w", err)
	}
	optionsJSON, err := json.Marshal(options)
	if err != nil {
		return "", fmt.Errorf("marshal options: %w", err)
	}

	payload := map[string]interface{}{
		"ContentJSON": string(contentJSON),
		"OptionsJSON": string(optionsJSON),
		"OutputPath":  req.OutputPath,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, payload); err != nil {
		return "", fmt.Errorf("render template: %w", err)
	}

	return buf.String(), nil
}

func normalizeContent(content interface{}) interface{} {
	if content == nil {
		return map[string]interface{}{}
	}

	switch v := content.(type) {
	case json.RawMessage:
		parsed := parseJSONPayload(v, map[string]interface{}{"text": string(v)})
		return unwrapNestedContent(parsed)
	case []byte:
		parsed := parseJSONPayload(v, map[string]interface{}{"text": string(v)})
		return unwrapNestedContent(parsed)
	case string:
		if trimmed := strings.TrimSpace(v); trimmed != "" && (strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")) {
			parsed := parseJSONPayload([]byte(trimmed), map[string]interface{}{"text": v})
			return unwrapNestedContent(parsed)
		}
		return map[string]interface{}{"text": v}
	case map[string]interface{}:
		return unwrapNestedContent(v)
	default:
		return map[string]interface{}{"text": fmt.Sprintf("%v", v)}
	}
}

// unwrapNestedContent extracts the inner content from common wrapper keys.
// For example, {"presentation": {"slides": [...]}} -> {"slides": [...]}
func unwrapNestedContent(content interface{}) interface{} {
	m, ok := content.(map[string]interface{})
	if !ok {
		return content
	}

	// Common wrapper keys to unwrap
	wrapperKeys := []string{"presentation", "document", "spreadsheet", "data"}
	for _, key := range wrapperKeys {
		if inner, exists := m[key]; exists {
			if innerMap, ok := inner.(map[string]interface{}); ok {
				// Check if the inner content has the expected structure (slides, sheets, etc.)
				if _, hasSlides := innerMap["slides"]; hasSlides {
					return innerMap
				}
				if _, hasSheets := innerMap["sheets"]; hasSheets {
					return innerMap
				}
				if _, hasSections := innerMap["sections"]; hasSections {
					return innerMap
				}
				if _, hasPages := innerMap["pages"]; hasPages {
					return innerMap
				}
				// Return the inner map if it has substantial content
				if len(innerMap) > 0 {
					return innerMap
				}
			}
		}
	}

	return content
}

func parseJSONPayload(data []byte, fallback interface{}) interface{} {
	if len(data) == 0 {
		return fallback
	}
	var parsed interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return fallback
	}
	return parsed
}

var _ domain.Service = (*Service)(nil)
