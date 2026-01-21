package steps

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// normalizeDeckPlan ensures slide count and layout safety.
func normalizeDeckPlan(plan DeckPlan, wantSlides int) DeckPlan {
	plan.Title = strings.TrimSpace(plan.Title)
	if plan.Title == "" {
		plan.Title = "Untitled Deck"
	}
	plan.Theme = normalizeTheme(plan.Theme)

	if wantSlides <= 0 {
		wantSlides = len(plan.Slides)
	}
	if len(plan.Slides) > wantSlides {
		plan.Slides = plan.Slides[:wantSlides]
	}
	for len(plan.Slides) < wantSlides {
		plan.Slides = append(plan.Slides, SlidePlan{ID: len(plan.Slides) + 1, Title: fmt.Sprintf("Slide %d", len(plan.Slides)+1)})
	}

	for i := range plan.Slides {
		plan.Slides[i].ID = i + 1
		plan.Slides[i].Title = strings.TrimSpace(plan.Slides[i].Title)
		if plan.Slides[i].Title == "" {
			plan.Slides[i].Title = fmt.Sprintf("Slide %d", i+1)
		}
		plan.Slides[i].Subtitle = strings.TrimSpace(plan.Slides[i].Subtitle)
		plan.Slides[i].Notes = strings.TrimSpace(plan.Slides[i].Notes)
		plan.Slides[i].Bullets = clampBullets(plan.Slides[i].Bullets, 10)

		if len(plan.Slides[i].Images) > 2 {
			plan.Slides[i].Images = plan.Slides[i].Images[:2]
		}
		for j := range plan.Slides[i].Images {
			plan.Slides[i].Images[j].Src = strings.TrimSpace(plan.Slides[i].Images[j].Src)
			plan.Slides[i].Images[j].Alt = strings.TrimSpace(plan.Slides[i].Images[j].Alt)
			plan.Slides[i].Images[j].Caption = strings.TrimSpace(plan.Slides[i].Images[j].Caption)
			if plan.Slides[i].Images[j].Alt == "" {
				plan.Slides[i].Images[j].Alt = plan.Slides[i].Title
			}
		}

		hasImg := len(plan.Slides[i].Images) > 0 && strings.TrimSpace(plan.Slides[i].Images[0].Src) != ""
		hasBul := len(plan.Slides[i].Bullets) > 0
		tableInfo := normalizeTable(plan.Slides[i].Table, 6, 9)
		hasTable := tableInfo.Has
		hasChart := plan.Slides[i].Chart != nil && len(plan.Slides[i].Chart.Series) > 0

		layout := strings.ToLower(strings.TrimSpace(plan.Slides[i].Layout))
		valid := map[string]bool{"split": true, "bullets": true, "hero": true, "title": true, "table": true, "chart": true}
		if !valid[layout] {
			layout = ""
		}
		if layout == "hero" && !hasImg {
			layout = ""
		}
		if layout == "split" && !(hasImg && hasBul) {
			layout = ""
		}
		if layout == "bullets" && !hasBul {
			layout = ""
		}
		if layout == "table" && !hasTable {
			layout = ""
		}
		if layout == "chart" && !hasChart {
			layout = ""
		}
		if layout == "" {
			layout = chooseLayout(plan.Slides[i])
		}
		plan.Slides[i].Layout = layout
	}

	return plan
}

func parseDeckPlan(content string) (DeckPlan, error) {
	var plan DeckPlan
	if err := json.Unmarshal([]byte(content), &plan); err == nil {
		return plan, nil
	}
	if extracted := extractFirstJSONObject(content); extracted != "" {
		if err := json.Unmarshal([]byte(extracted), &plan); err == nil {
			return plan, nil
		}
	}
	return DeckPlan{}, fmt.Errorf("could not parse deck plan JSON")
}

func extractFirstJSONObject(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)

	start := strings.Index(s, "{")
	if start < 0 {
		return ""
	}
	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
}

func chooseLayout(slide SlidePlan) string {
	hasImg := len(slide.Images) > 0 && strings.TrimSpace(slide.Images[0].Src) != ""
	hasBul := len(slide.Bullets) > 0
	hasTable := slide.Table != nil && len(slide.Table.Columns) > 0 && len(slide.Table.Rows) > 0
	hasChart := slide.Chart != nil && len(slide.Chart.Series) > 0
	if hasChart {
		return "chart"
	}
	if hasTable {
		return "table"
	}
	if hasImg && hasBul {
		return "split"
	}
	if hasImg {
		return "hero"
	}
	if hasBul {
		return "bullets"
	}
	return "title"
}

func clampBullets(in []string, max int) []string {
	out := make([]string, 0, len(in))
	for _, b := range in {
		b = strings.TrimSpace(b)
		b = strings.TrimPrefix(b, "-")
		b = strings.TrimSpace(b)
		if b == "" {
			continue
		}
		out = append(out, b)
		if len(out) >= max {
			break
		}
	}
	return out
}

// buildChartsExport writes charts.json for raster overlay or downstream use.
func buildChartsExport(plan DeckPlan) ChartsExport {
	out := ChartsExport{Slides: []ChartData{}}
	for _, slide := range plan.Slides {
		if slide.Chart == nil {
			continue
		}
		chart := *slide.Chart
		chart.ID = slide.ID
		chart.Position = normalizeChartPosition(chart.Position)
		chartCopy := chart
		chartCopy.Units = strings.TrimSpace(chartCopy.Units)
		out.Slides = append(out.Slides, chartCopy)
	}
	return out
}

func normalizeChartPosition(pos ChartPosition) ChartPosition {
	if pos.W == 0 {
		pos.W = 11.2
	}
	if pos.H == 0 {
		pos.H = 5.3
	}
	if pos.X == 0 {
		pos.X = 1.1
	}
	if pos.Y == 0 {
		pos.Y = 1.3
	}
	if strings.TrimSpace(pos.Units) == "" {
		pos.Units = "in"
	}
	return pos
}

var hexColorRE = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func isHexColor(s string) bool {
	return hexColorRE.MatchString(strings.TrimSpace(s))
}
