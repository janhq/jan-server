package steps

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/status"

	"github.com/rs/zerolog/log"
)

type catalogSource struct {
	fs          fs.FS
	catalogPath string
	rootDir     string
}

func resolveCatalogSource(templateCatalog string) (catalogSource, error) {
	catalog := strings.TrimSpace(templateCatalog)
	if catalog == "" {
		return catalogSource{
			fs:          embeddedTemplatesRoot(),
			catalogPath: "index.json",
			rootDir:     "",
		}, nil
	}

	if filepath.IsAbs(catalog) {
		rootDir := filepath.Dir(catalog)
		return catalogSource{
			fs:          os.DirFS(rootDir),
			catalogPath: filepath.Base(catalog),
			rootDir:     rootDir,
		}, nil
	}

	rel := filepath.ToSlash(catalog)
	rel = strings.TrimPrefix(rel, "templates/")
	rel = strings.TrimPrefix(rel, "./")
	return catalogSource{
		fs:          embeddedTemplatesRoot(),
		catalogPath: rel,
		rootDir:     "",
	}, nil
}

func loadTemplateCatalog(src catalogSource) (TemplateCatalog, error) {
	raw, err := fs.ReadFile(src.fs, src.catalogPath)
	if err != nil {
		return TemplateCatalog{}, fmt.Errorf("read template catalog: %w", err)
	}
	var catalog TemplateCatalog
	if err := json.Unmarshal(raw, &catalog); err != nil {
		return TemplateCatalog{}, fmt.Errorf("parse template catalog: %w", err)
	}
	return catalog, nil
}

func templateDirForID(catalog TemplateCatalog, templateID int, rootDir string) string {
	dir := strconv.Itoa(templateID)
	for _, t := range catalog.Templates {
		if t.ID == templateID {
			candidate := strings.TrimSpace(t.Dir)
			if candidate != "" {
				dir = candidate
			}
			break
		}
	}
	if rootDir == "" {
		return path.Join(dir)
	}
	return filepath.Join(rootDir, dir)
}

func (e *SlideCreatorExecutor) executeSelectTemplates(ctx context.Context, params map[string]interface{}, input agent.ExecutionInput) (*agent.ExecutionResult, error) {
	config, _ := params["config"].(map[string]interface{})
	templateDir := strings.TrimSpace(stringValue(config, "template_dir"))
	templateCatalog := strings.TrimSpace(stringValue(config, "template_catalog"))
	tone := strings.TrimSpace(stringValue(config, "tone"))
	colorScheme := strings.TrimSpace(stringValue(config, "color_scheme"))
	style := strings.TrimSpace(stringValue(config, "style"))
	brief := strings.TrimSpace(stringValue(params, "brief"))
	templateID, _ := parseIntFromInterface(config["template_id"])

	selection := TemplateSelection{
		Type:        "template_selection",
		TemplateID:  templateID,
		TemplateDir: templateDir,
		Source:      "embedded",
	}

	if templateDir != "" {
		if filepath.IsAbs(templateDir) {
			selection.Source = "filesystem"
		}
		outputBytes, _ := json.Marshal(selection)
		return &agent.ExecutionResult{Status: status.StatusCompleted, Output: outputBytes}, nil
	}

	src, err := resolveCatalogSource(templateCatalog)
	if err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error:  &agent.ExecutionError{Code: "CATALOG_ERROR", Message: err.Error(), Severity: status.ErrorSeverityRetryable},
		}, nil
	}

	catalog, err := loadTemplateCatalog(src)
	if err != nil {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error:  &agent.ExecutionError{Code: "CATALOG_ERROR", Message: err.Error(), Severity: status.ErrorSeverityRetryable},
		}, nil
	}

	if templateID > 0 {
		selection.TemplateDir = templateDirForID(catalog, templateID, src.rootDir)
		if src.rootDir != "" {
			selection.Source = "filesystem"
		}
		outputBytes, _ := json.Marshal(selection)
		return &agent.ExecutionResult{Status: status.StatusCompleted, Output: outputBytes}, nil
	}

	model := getModelFromContext(input)
	choices, err := e.selectTemplateChoices(ctx, catalog, tone, style, colorScheme, brief, model)
	if err != nil || len(choices) == 0 {
		choices = fallbackTemplateChoices(catalog, tone, colorScheme, 3)
	}
	choices = filterTemplateChoicesByColorScheme(choices, catalog, colorScheme)
	if len(choices) == 0 {
		choices = fallbackTemplateChoices(catalog, "", colorScheme, 3)
		choices = filterTemplateChoicesByColorScheme(choices, catalog, colorScheme)
	}
	if len(choices) == 0 {
		return &agent.ExecutionResult{
			Status: status.StatusFailed,
			Error:  &agent.ExecutionError{Code: "NO_TEMPLATES", Message: "no templates available for selection", Severity: status.ErrorSeverityFatal},
		}, nil
	}

	selected := choices[0]
	selection.Choices = choices
	selection.Selected = &selected
	selection.TemplateID = selected.ID
	selection.TemplateDir = templateDirForID(catalog, selected.ID, src.rootDir)
	if src.rootDir != "" || filepath.IsAbs(selection.TemplateDir) {
		selection.Source = "filesystem"
	}

	outputBytes, _ := json.Marshal(selection)
	return &agent.ExecutionResult{Status: status.StatusCompleted, Output: outputBytes}, nil
}

func (e *SlideCreatorExecutor) selectTemplateChoices(ctx context.Context, catalog TemplateCatalog, tone string, style string, colorScheme string, brief string, model string) ([]TemplateChoice, error) {
	lines := make([]string, 0, len(catalog.Templates))
	for _, t := range catalog.Templates {
		line := fmt.Sprintf("%d | %s | %s | %s", t.ID, t.Name, t.Tone, t.Description)
		lines = append(lines, line)
	}
	prompt := fmt.Sprintf(`You are selecting the best 3 slide template packs.

Return ONLY JSON with this shape:
{
  "choices": [
    {"id": 1, "reason": "..."}
  ]
}

Rules:
- Choose exactly 3 template IDs from the catalog.
- Reasons must be short (<= 140 chars).
- Use tone and brief to match the best fit.
- Output JSON only.

User tone (optional):
%s

Style preference (optional):
%s

Color scheme (optional):
%s

If color scheme is bright/light/vibrant, avoid dark or monochrome templates.

Brief:
%s

Catalog:
%s
`, tone, style, colorScheme, brief, strings.Join(lines, "\n"))

	if e.llmProvider == nil {
		return nil, fmt.Errorf("LLM provider not configured")
	}

	result, err := e.generateWithStructuredOutput(ctx, "", prompt, model, TemplatePickSchema, 0.2)
	if err != nil {
		log.Warn().Err(err).Msg("[slide_creator] template selection failed")
		return nil, err
	}

	choices, err := parseTemplateChoices(result)
	if err != nil {
		return nil, err
	}
	choices = filterTemplateChoices(choices, catalog)
	choices = filterTemplateChoicesByColorScheme(choices, catalog, colorScheme)
	if len(choices) > 3 {
		choices = choices[:3]
	}
	return choices, nil
}

func parseTemplateChoices(content string) ([]TemplateChoice, error) {
	var resp TemplatePickResponse
	if err := json.Unmarshal([]byte(content), &resp); err == nil && len(resp.Choices) > 0 {
		return resp.Choices, nil
	}
	if extracted := extractFirstJSONObject(content); extracted != "" {
		if err := json.Unmarshal([]byte(extracted), &resp); err == nil && len(resp.Choices) > 0 {
			return resp.Choices, nil
		}
	}
	return nil, fmt.Errorf("could not parse template picker response")
}

func filterTemplateChoices(in []TemplateChoice, catalog TemplateCatalog) []TemplateChoice {
	seen := map[int]bool{}
	valid := map[int]bool{}
	for _, t := range catalog.Templates {
		valid[t.ID] = true
	}
	out := make([]TemplateChoice, 0, len(in))
	for _, c := range in {
		if c.ID <= 0 || !valid[c.ID] || seen[c.ID] {
			continue
		}
		seen[c.ID] = true
		out = append(out, c)
		if len(out) >= 3 {
			break
		}
	}
	return out
}

func fallbackTemplateChoices(catalog TemplateCatalog, tone string, colorScheme string, max int) []TemplateChoice {
	tone = strings.ToLower(strings.TrimSpace(tone))
	out := make([]TemplateChoice, 0, max)
	preferBright := isBrightColorScheme(colorScheme)
	if tone != "" {
		for _, t := range catalog.Templates {
			if strings.ToLower(strings.TrimSpace(t.Tone)) == tone {
				if preferBright && isDarkTemplateTone(t.Tone) {
					continue
				}
				out = append(out, TemplateChoice{ID: t.ID, Reason: "tone match"})
				if len(out) >= max {
					return out
				}
			}
		}
	}
	for _, t := range catalog.Templates {
		if preferBright && isDarkTemplateTone(t.Tone) {
			continue
		}
		out = append(out, TemplateChoice{ID: t.ID, Reason: "default"})
		if len(out) >= max {
			break
		}
	}
	return out
}

func filterTemplateChoicesByColorScheme(in []TemplateChoice, catalog TemplateCatalog, colorScheme string) []TemplateChoice {
	if !isBrightColorScheme(colorScheme) {
		return in
	}
	toneByID := map[int]string{}
	for _, t := range catalog.Templates {
		toneByID[t.ID] = t.Tone
	}
	out := make([]TemplateChoice, 0, len(in))
	for _, c := range in {
		if tone, ok := toneByID[c.ID]; ok && isDarkTemplateTone(tone) {
			continue
		}
		out = append(out, c)
	}
	return out
}

func isBrightColorScheme(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "bright", "light", "vibrant", "colorful", "colourful", "pastel":
		return true
	default:
		return false
	}
}

func isDarkTemplateTone(tone string) bool {
	tone = strings.ToLower(strings.TrimSpace(tone))
	return strings.Contains(tone, "dark") || strings.Contains(tone, "monochrome") || strings.Contains(tone, "noir")
}

func stringValue(values map[string]interface{}, key string) string {
	if values == nil {
		return ""
	}
	if value, ok := values[key].(string); ok {
		return value
	}
	return ""
}
