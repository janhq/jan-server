package steps

import "strings"

func limitImageAssets(assets []map[string]any, max int) []map[string]any {
	if max <= 0 || len(assets) <= max {
		return assets
	}
	return assets[:max]
}

func compactImageAssetsForPrompt(assets []map[string]any) []map[string]any {
	if len(assets) == 0 {
		return assets
	}
	out := make([]map[string]any, 0, len(assets))
	for _, asset := range assets {
		if asset == nil {
			continue
		}
		prompt := map[string]any{}
		if id, ok := asset["id"].(string); ok && strings.TrimSpace(id) != "" {
			prompt["id"] = id
		}
		if kind, ok := asset["kind"].(string); ok && strings.TrimSpace(kind) != "" {
			prompt["kind"] = kind
		}
		if url := promptImageURL(asset); url != "" {
			prompt["source"] = map[string]any{"type": "url", "url": url}
		}
		if title, ok := asset["title"].(string); ok && strings.TrimSpace(title) != "" {
			prompt["title"] = title
		}
		if alt, ok := asset["altText"].(string); ok && strings.TrimSpace(alt) != "" {
			prompt["altText"] = alt
		}
		if attribution, ok := asset["attribution"].(string); ok && strings.TrimSpace(attribution) != "" {
			prompt["attribution"] = attribution
		}
		if license, ok := asset["license"]; ok {
			prompt["license"] = license
		}
		if len(prompt) > 0 {
			out = append(out, prompt)
		}
	}
	return out
}

func promptImageURL(asset map[string]any) string {
	if asset == nil {
		return ""
	}
	if thumb, ok := asset["thumbnailUrl"].(string); ok && strings.TrimSpace(thumb) != "" {
		return thumb
	}
	if img, ok := asset["imageUrl"].(string); ok && strings.TrimSpace(img) != "" {
		return img
	}
	if source, ok := asset["source"].(map[string]any); ok {
		if url, ok := source["url"].(string); ok && strings.TrimSpace(url) != "" {
			return url
		}
	}
	return ""
}
