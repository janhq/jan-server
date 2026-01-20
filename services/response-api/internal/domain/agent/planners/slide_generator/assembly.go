package slide_generator

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"jan-server/services/response-api/internal/domain/agent"

	"github.com/rs/zerolog/log"
)

func (e *SlideGeneratorExecutor) assembleDeck(input agent.ExecutionInput) ([]byte, error) {
	log.Debug().Int("accumulated_outputs", len(input.AccumulatedOutputs)).Msg("[slide_generator] assembleDeck started")
	planAndTemplate := e.extractPlanAndTemplate(input)
	if planAndTemplate == nil {
		log.Error().Msg("[slide_generator] plan and template not found")
		return nil, fmt.Errorf("plan and template not found")
	}
	log.Debug().Int("plan_slides", len(planAndTemplate.Plan.Slides)).Msg("[slide_generator] extracted plan and template")

	slidesByIndex := map[int]any{}
	var allAssets []any
	var allDatasets []any
	lockedAssets := e.collectImageAssets(input)
	for _, asset := range lockedAssets {
		allAssets = append(allAssets, asset)
	}
	allDatasets = append(allDatasets, e.collectDataBankDatasets(input)...)

	for _, output := range input.AccumulatedOutputs {
		var parsed map[string]interface{}
		if err := json.Unmarshal(output, &parsed); err != nil {
			continue
		}
		if parsed["type"] == "slide_result" {
			if slideIndexRaw, ok := parsed["slide_index"].(float64); ok {
				if slide, ok := parsed["slide"]; ok {
					slidesByIndex[int(slideIndexRaw)] = slide
				}
				if requires, ok := parsed["requires"].(map[string]interface{}); ok {
					if assets, ok := requires["assets"].([]any); ok {
						allAssets = append(allAssets, assets...)
					}
					// Safely extract datasets - ignore invalid ones
					if datasets, ok := requires["datasets"].([]any); ok {
						for _, ds := range datasets {
							if ds != nil {
								allDatasets = append(allDatasets, ds)
							}
						}
					}
				}
			}
		}
	}

	expected := len(planAndTemplate.Plan.Slides)
	orderedSlides := make([]any, 0, expected)
	if expected > 0 {
		for i := 1; i <= expected; i++ {
			slide, ok := slidesByIndex[i]
			if !ok {
				return nil, fmt.Errorf("missing slide %d during assembly", i)
			}
			orderedSlides = append(orderedSlides, slide)
		}
	} else {
		indices := make([]int, 0, len(slidesByIndex))
		for idx := range slidesByIndex {
			indices = append(indices, idx)
		}
		sort.Ints(indices)
		for _, idx := range indices {
			orderedSlides = append(orderedSlides, slidesByIndex[idx])
		}
	}

	metadata := map[string]any{
		"title":    planAndTemplate.Plan.DeckTitle,
		"language": "en",
		"audience": planAndTemplate.Plan.Audience,
		"purpose":  planAndTemplate.Plan.Purpose,
		"tone":     planAndTemplate.Plan.Tone,
	}

	if tmplMeta, ok := planAndTemplate.Template.Metadata.(map[string]any); ok {
		if _, hasLang := tmplMeta["language"]; hasLang {
			for key, value := range tmplMeta {
				metadata[key] = value
			}
			metadata["title"] = planAndTemplate.Plan.DeckTitle
			if planAndTemplate.Plan.Audience != "" {
				metadata["audience"] = planAndTemplate.Plan.Audience
			}
			if planAndTemplate.Plan.Purpose != "" {
				metadata["purpose"] = planAndTemplate.Plan.Purpose
			}
			if planAndTemplate.Plan.Tone != "" {
				metadata["tone"] = planAndTemplate.Plan.Tone
			}
		}
	}

	assets, err := e.normalizeAssets(allAssets)
	if err != nil {
		return nil, err
	}
	data, err := e.normalizeDatasets(allDatasets)
	if err != nil {
		return nil, err
	}

	deck := map[string]any{
		"version":    planAndTemplate.Template.Version,
		"metadata":   metadata,
		"theme":      planAndTemplate.Template.Theme,
		"masters":    planAndTemplate.Template.Masters,
		"layouts":    planAndTemplate.Template.Layouts,
		"components": planAndTemplate.Template.Components,
		"slides":     orderedSlides,
		"assets":     assets,
		"data":       data,
		"validation": map[string]any{"rules": []any{}},
		"export":     planAndTemplate.Template.Export,
	}

	fixedDeck := e.fixCommonSchemaIssues(deck)
	if err := validateDeck(fixedDeck, expected); err != nil {
		return nil, err
	}
	deckJSON, err := json.Marshal(fixedDeck)
	if err != nil {
		log.Error().Err(err).Msg("[slide_generator] failed to marshal deck")
		return nil, err
	}
	log.Debug().
		Int("slides_count", len(orderedSlides)).
		Int("assets_count", len(allAssets)).
		Int("datasets_count", len(allDatasets)).
		Int("deck_json_size", len(deckJSON)).
		Msg("[slide_generator] assembleDeck completed")
	return deckJSON, nil
}

func (e *SlideGeneratorExecutor) normalizeAssets(allAssets []any) (map[string]any, error) {
	normalizedImages := []any{}
	seen := map[string]bool{}
	for _, asset := range allAssets {
		if asset == nil {
			continue
		}
		switch v := asset.(type) {
		case map[string]any:
			id, _ := v["id"].(string)
			if id == "" {
				id = fmt.Sprintf("asset_%d", len(normalizedImages)+1)
				v["id"] = id
			}
			if seen[id] {
				continue
			}
			seen[id] = true

			if _, ok := v["kind"].(string); !ok {
				v["kind"] = "image"
			}

			source, _ := v["source"].(map[string]any)
			if source == nil {
				source = map[string]any{}
			}

			if _, hasType := source["type"]; !hasType {
				if urlStr, ok := source["url"].(string); ok && urlStr != "" {
					source["type"] = "url"
				} else if filePath, ok := source["filePath"].(string); ok && filePath != "" {
					source["type"] = "file"
				} else if base64Str, ok := source["base64"].(string); ok && base64Str != "" {
					source["type"] = "base64"
				} else if urlStr, ok := v["url"].(string); ok && urlStr != "" {
					source["type"] = "url"
					source["url"] = urlStr
				} else if filePath, ok := v["filePath"].(string); ok && filePath != "" {
					source["type"] = "file"
					source["filePath"] = filePath
				} else if base64Str, ok := v["base64"].(string); ok && base64Str != "" {
					source["type"] = "base64"
					source["base64"] = base64Str
				}
			}

			sourceType, _ := source["type"].(string)
			switch sourceType {
			case "url":
				urlStr, ok := source["url"].(string)
				if !ok || urlStr == "" {
					return nil, fmt.Errorf("asset %s missing source.url", id)
				}
				// P1 fix: Validate URL scheme is http or https
				parsedURL, err := url.Parse(urlStr)
				if err != nil {
					return nil, fmt.Errorf("asset %s has invalid URL: %v", id, err)
				}
				scheme := strings.ToLower(parsedURL.Scheme)
				if scheme != "http" && scheme != "https" {
					return nil, fmt.Errorf("asset %s has unsupported URL scheme %q (must be http or https)", id, parsedURL.Scheme)
				}
			case "file":
				if pathStr, ok := source["filePath"].(string); !ok || pathStr == "" {
					return nil, fmt.Errorf("asset %s missing source.filePath", id)
				}
			case "base64":
				if base64Str, ok := source["base64"].(string); !ok || base64Str == "" {
					return nil, fmt.Errorf("asset %s missing source.base64", id)
				}
			default:
				return nil, fmt.Errorf("asset %s has unsupported source type %q", id, sourceType)
			}

			v["source"] = source
			if _, hasAltText := v["altText"]; !hasAltText {
				v["altText"] = id
			}
			if _, hasLicense := v["license"]; !hasLicense {
				v["license"] = nil
			}
			if _, hasAttribution := v["attribution"]; !hasAttribution {
				v["attribution"] = nil
			}
			normalizedImages = append(normalizedImages, v)
		case string:
			return nil, fmt.Errorf("asset %q must be an object, not a string", v)
		default:
			return nil, fmt.Errorf("unsupported asset type %T", asset)
		}
	}

	return map[string]any{"images": normalizedImages}, nil
}

func (e *SlideGeneratorExecutor) normalizeDatasets(allDatasets []any) (map[string]any, error) {
	normalizedDatasets := []any{}
	seen := map[string]bool{}
	for i, dataset := range allDatasets {
		if dataset == nil {
			return nil, fmt.Errorf("dataset %d is nil", i)
		}
		switch v := dataset.(type) {
		case string:
			return nil, fmt.Errorf("dataset %q must be an object, not a string", v)
		case map[string]any:
			id, _ := v["id"].(string)
			if id == "" {
				return nil, fmt.Errorf("dataset missing id at index %d", i)
			}
			if seen[id] {
				continue
			}
			seen[id] = true

			kind, _ := v["kind"].(string)
			if kind == "" {
				kind = "series"
				v["kind"] = "series"
			}
			if kind != "series" {
				return nil, fmt.Errorf("dataset %s has unsupported kind %q", id, kind)
			}

			dataMap, _ := v["data"].(map[string]any)
			if dataMap == nil {
				return nil, fmt.Errorf("dataset %s missing data", id)
			}

			labels, _ := dataMap["labels"].([]any)
			if len(labels) == 0 {
				return nil, fmt.Errorf("dataset %s missing labels", id)
			}
			series, _ := dataMap["series"].([]any)
			if len(series) == 0 {
				return nil, fmt.Errorf("dataset %s missing series", id)
			}
			for _, seriesAny := range series {
				seriesMap, ok := seriesAny.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("dataset %s has invalid series entry", id)
				}
				values, _ := seriesMap["values"].([]any)
				if len(values) != len(labels) {
					return nil, fmt.Errorf("dataset %s has mismatched labels/values length", id)
				}
			}
			v["data"] = dataMap
			if _, hasSourceNote := v["sourceNote"]; !hasSourceNote {
				v["sourceNote"] = nil
			}
			normalizedDatasets = append(normalizedDatasets, v)
		default:
			return nil, fmt.Errorf("invalid dataset type %T", dataset)
		}
	}

	return map[string]any{"datasets": normalizedDatasets}, nil
}
