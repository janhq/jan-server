package slide_generator

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"jan-server/services/response-api/internal/domain/agent"

	"github.com/rs/zerolog/log"
)

func (e *SlideGeneratorExecutor) collectDataBankText(input agent.ExecutionInput) string {
	outputs := make([]json.RawMessage, 0, len(input.AccumulatedOutputs)+1)
	outputs = append(outputs, input.AccumulatedOutputs...)
	if len(input.PreviousOutput) > 0 {
		outputs = append(outputs, input.PreviousOutput)
	}
	for i := len(outputs) - 1; i >= 0; i-- {
		var payload map[string]any
		if err := json.Unmarshal(outputs[i], &payload); err != nil {
			continue
		}
		if payloadType, _ := payload["type"].(string); payloadType == "data_bank" {
			if content, ok := payload["content"].(string); ok && content != "" {
				log.Debug().Int("content_length", len(content)).Msg("[slide_generator] data bank text collected")
				return content
			}
			if data, ok := payload["data"]; ok {
				if raw, err := json.Marshal(data); err == nil {
					log.Debug().Int("content_length", len(raw)).Msg("[slide_generator] data bank text collected")
					return string(raw)
				}
			}
		}
	}
	return "[No data bank available]"
}

func (e *SlideGeneratorExecutor) collectDataBankDatasets(input agent.ExecutionInput) []any {
	outputs := make([]json.RawMessage, 0, len(input.AccumulatedOutputs)+1)
	outputs = append(outputs, input.AccumulatedOutputs...)
	if len(input.PreviousOutput) > 0 {
		outputs = append(outputs, input.PreviousOutput)
	}
	for i := len(outputs) - 1; i >= 0; i-- {
		var payload map[string]any
		if err := json.Unmarshal(outputs[i], &payload); err != nil {
			continue
		}
		if payloadType, _ := payload["type"].(string); payloadType == "data_bank" {
			if data, ok := payload["data"].(map[string]any); ok {
				if datasets, ok := data["datasets"].([]any); ok {
					log.Debug().Int("dataset_count", len(datasets)).Msg("[slide_generator] data bank datasets collected")
					return datasets
				}
			}
		}
	}
	return nil
}

func limitImageAssets(assets []map[string]any, max int) []map[string]any {
	if max <= 0 || len(assets) <= max {
		return assets
	}
	return assets[:max]
}

func (e *SlideGeneratorExecutor) collectImageAssets(input agent.ExecutionInput) []map[string]any {
	outputs := make([]json.RawMessage, 0, len(input.AccumulatedOutputs)+1)
	outputs = append(outputs, input.AccumulatedOutputs...)
	if len(input.PreviousOutput) > 0 {
		outputs = append(outputs, input.PreviousOutput)
	}

	assetsByID := map[string]map[string]any{}
	for _, output := range outputs {
		for _, asset := range extractImageAssetsFromOutput(output) {
			id, _ := asset["id"].(string)
			if id == "" {
				continue
			}
			if _, exists := assetsByID[id]; !exists {
				assetsByID[id] = asset
			}
		}
	}

	assets := make([]map[string]any, 0, len(assetsByID))
	for _, asset := range assetsByID {
		assets = append(assets, asset)
	}
	sort.Slice(assets, func(i, j int) bool {
		return fmt.Sprint(assets[i]["id"]) < fmt.Sprint(assets[j]["id"])
	})
	log.Debug().Int("assets", len(assets)).Msg("[slide_generator] image assets collected")
	return assets
}

func extractImageAssetsFromOutput(output json.RawMessage) []map[string]any {
	if len(output) == 0 {
		return nil
	}
	var parsed map[string]any
	if err := json.Unmarshal(output, &parsed); err != nil {
		return nil
	}

	results := []map[string]any{}
	if content, ok := parsed["content"].([]any); ok {
		for _, item := range content {
			if itemMap, ok := item.(map[string]any); ok {
				if text, ok := itemMap["text"].(string); ok && text != "" {
					var nested map[string]any
					if err := json.Unmarshal([]byte(text), &nested); err == nil {
						results = append(results, extractImageAssetsFromMap(nested)...)
					}
				}
			}
		}
	}

	results = append(results, extractImageAssetsFromMap(parsed)...)
	return results
}

func extractImageAssetsFromMap(data map[string]any) []map[string]any {
	results := []map[string]any{}
	for _, key := range []string{"images", "results", "items", "data"} {
		if arr, ok := data[key].([]any); ok {
			results = append(results, extractImageAssetsFromArray(arr)...)
		}
	}
	for _, value := range data {
		switch typed := value.(type) {
		case map[string]any:
			results = append(results, extractImageAssetsFromMap(typed)...)
		case []any:
			results = append(results, extractImageAssetsFromArray(typed)...)
		}
	}
	return results
}

func extractImageAssetsFromArray(arr []any) []map[string]any {
	results := []map[string]any{}
	for _, item := range arr {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if asset := assetFromImageResult(itemMap); asset != nil {
			results = append(results, asset)
		}
	}
	return results
}

func assetFromImageResult(item map[string]any) map[string]any {
	imageURL := firstString(item, "imageUrl", "image_url", "url", "link", "source_url")
	thumbURL := firstString(item, "thumbnailUrl", "thumbnail_url", "thumbnail", "thumb", "previewUrl", "preview_url")
	if imageURL == "" && thumbURL == "" {
		return nil
	}
	sourceURL := imageURL
	if sourceURL == "" {
		sourceURL = thumbURL
	}
	parsed, _ := url.Parse(sourceURL)
	host := ""
	if parsed != nil {
		host = parsed.Host
	}
	title := firstString(item, "title", "alt", "altText", "snippet", "description")
	altText := title
	if altText == "" {
		altText = host
	}
	license := firstString(item, "license")
	attribution := firstString(item, "attribution", "source")
	if attribution == "" {
		attribution = host
	}
	id := assetIDFromURL(sourceURL)
	asset := map[string]any{
		"id":   id,
		"kind": "image",
		"source": map[string]any{
			"type": "url",
			"url":  sourceURL,
		},
		"altText":     altText,
		"license":     license,
		"attribution": attribution,
	}
	if title != "" {
		asset["title"] = title
	}
	if imageURL != "" {
		asset["imageUrl"] = imageURL
	}
	if thumbURL != "" {
		asset["thumbnailUrl"] = thumbURL
	}
	return asset
}

func assetIDFromURL(urlStr string) string {
	hasher := sha1.New()
	hasher.Write([]byte(urlStr))
	return "img_" + hex.EncodeToString(hasher.Sum(nil))[:12]
}

func firstString(item map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := item[key].(string); ok && strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
