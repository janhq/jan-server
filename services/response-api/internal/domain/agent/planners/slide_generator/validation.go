package slide_generator

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/rs/zerolog/log"
)

func (e *SlideGeneratorExecutor) fixCommonSchemaIssues(deck map[string]any) map[string]any {
	allowedTextStyleProps := map[string]bool{
		"fontFamily": true, "fontSize": true, "bold": true, "italic": true,
		"underline": true, "color": true, "align": true, "valign": true,
		"lineHeight": true, "letterSpacing": true, "bullet": true,
	}

	allowedShapeStyleProps := map[string]bool{
		"fill": true, "stroke": true, "cornerRadius": true, "shadow": true,
	}

	layoutSlots := buildLayoutSlotMap(deck["layouts"])
	componentIDs := buildComponentIDSet(deck["components"])
	theme, _ := deck["theme"].(map[string]any)
	safeMargins := extractSafeMargins(theme)

	if slides, ok := deck["slides"].([]any); ok {
		for _, slideAny := range slides {
			slide, ok := slideAny.(map[string]any)
			if !ok {
				continue
			}
			ensureSlideUseComponents(slide)
			layoutID, _ := slide["layoutId"].(string)
			slotsForLayout := layoutSlots[layoutID]

			if isContentSlide(layoutID) {
				if componentIDs["header"] {
					appendComponentID(slide, "header")
				}
				if componentIDs["footer"] {
					appendComponentID(slide, "footer")
				}
				if !componentIDs["header"] && !hasSlotOrElement(slide, "header") {
					addHeaderElement(slide, slotsForLayout, theme, safeMargins)
				}
				if !componentIDs["footer"] && !hasSlotOrElement(slide, "footer") {
					addFooterElement(slide, slotsForLayout, theme, safeMargins)
				}
			}

			if elements, ok := slide["elements"].([]any); ok {
				for _, elemAny := range elements {
					elem, ok := elemAny.(map[string]any)
					if !ok {
						continue
					}
					if slotID, ok := elem["slotId"].(string); ok && slotID != "" {
						if _, hasRect := elem["rect"].(map[string]any); !hasRect {
							if slot := slotsForLayout[slotID]; slot != nil {
								if rect, ok := rectFromSlot(slot, theme); ok {
									elem["rect"] = rect
								}
							}
						}
					}
					if text, ok := elem["text"].(map[string]any); ok {
						if content, ok := text["content"].(string); ok {
							if strings.Contains(content, "|") && !strings.Contains(content, "\n") {
								text["content"] = strings.ReplaceAll(content, "|", "\n")
							}
						}
						if v, ok := text["autoFit"]; !ok || v == nil {
							text["autoFit"] = "shrink"
						}
						if style, ok := text["style"].(map[string]any); ok {
							for _, k := range []string{"fontFamily", "fontSize", "bold", "italic", "underline", "color", "align", "valign", "lineHeight", "letterSpacing", "bullet"} {
								if _, has := style[k]; !has {
									style[k] = nil
								}
							}
							if b, ok := style["bullet"].(map[string]any); ok {
								for _, k := range []string{"enabled", "indent", "hanging"} {
									if _, has := b[k]; !has {
										b[k] = nil
									}
								}
							}
							fixStyleProperties(style, allowedTextStyleProps)
						}
					}

					if shape, ok := elem["shape"].(map[string]any); ok {
						allowedKinds := map[string]bool{"rect": true, "line": true, "arrow": true, "triangle": true, "diamond": true}
						if kind, ok := shape["kind"].(string); ok {
							if !allowedKinds[kind] {
								shape["kind"] = "rect"
							}
						} else {
							shape["kind"] = "rect"
						}

						style, _ := shape["style"].(map[string]any)
						if style == nil {
							style = map[string]any{}
							shape["style"] = style
						}
						for _, k := range []string{"fill", "stroke", "cornerRadius", "shadow"} {
							if _, has := style[k]; !has {
								style[k] = nil
							}
						}
						style["cornerRadius"] = nil
						style["shadow"] = nil

						fixStyleProperties(style, allowedShapeStyleProps)
					}
				}
			}
		}
	}

	return deck
}

func fixStyleProperties(style map[string]any, allowed map[string]bool) {
	for key := range style {
		if !allowed[key] {
			delete(style, key)
		}
	}
}

func buildLayoutSlotMap(layouts any) map[string]map[string]map[string]any {
	result := map[string]map[string]map[string]any{}
	layoutSlice, ok := layouts.([]any)
	if !ok {
		return result
	}
	for _, layoutAny := range layoutSlice {
		layout, ok := layoutAny.(map[string]any)
		if !ok {
			continue
		}
		layoutID, _ := layout["id"].(string)
		if layoutID == "" {
			continue
		}
		slots := map[string]map[string]any{}
		if slotList, ok := layout["slots"].([]any); ok {
			for _, slotAny := range slotList {
				if slotMap, ok := slotAny.(map[string]any); ok {
					if slotID, ok := slotMap["id"].(string); ok && slotID != "" {
						slots[slotID] = slotMap
					}
				}
			}
		}
		result[layoutID] = slots
	}
	return result
}

func buildComponentIDSet(components any) map[string]bool {
	result := map[string]bool{}
	componentsSlice, ok := components.([]any)
	if !ok {
		return result
	}
	for _, compAny := range componentsSlice {
		comp, ok := compAny.(map[string]any)
		if !ok {
			continue
		}
		if id, ok := comp["id"].(string); ok && id != "" {
			result[id] = true
		}
	}
	return result
}

func appendComponentID(slide map[string]any, componentID string) {
	useComponents, _ := slide["useComponents"].([]any)
	if useComponents == nil {
		useComponents = []any{}
	}
	for _, existing := range useComponents {
		if existingID, ok := existing.(string); ok && existingID == componentID {
			return
		}
	}
	slide["useComponents"] = append(useComponents, componentID)
}

func isContentSlide(layoutID string) bool {
	switch strings.TrimSpace(layoutID) {
	case "TITLE", "SECTION_HEADER", "CLOSING":
		return false
	default:
		return true
	}
}

func hasSlotOrElement(slide map[string]any, target string) bool {
	elements, _ := slide["elements"].([]any)
	for _, elemAny := range elements {
		elem, ok := elemAny.(map[string]any)
		if !ok {
			continue
		}
		if slotID, ok := elem["slotId"].(string); ok && slotID == target {
			return true
		}
		if id, ok := elem["id"].(string); ok && strings.Contains(strings.ToLower(id), target) {
			return true
		}
	}
	return false
}

func addHeaderElement(slide map[string]any, slots map[string]map[string]any, theme map[string]any, safeMargins map[string]float64) {
	headerText := ""
	if title, ok := slide["title"].(string); ok {
		headerText = title
	}
	elem := map[string]any{
		"id":   fmt.Sprintf("%v_header", slide["id"]),
		"type": "text",
		"text": map[string]any{
			"content": headerText,
			"runs":    []any{},
			"autoFit": "shrink",
			"style": map[string]any{
				"fontSize": 12,
				"color":    themeMutedText(theme),
				"align":    "left",
			},
		},
	}
	if slot, ok := slots["header"]; ok {
		elem["slotId"] = "header"
		if rect, ok := rectFromSlot(slot, theme); ok {
			elem["rect"] = rect
		}
	} else {
		elem["rect"] = map[string]any{
			"x": safeMargins["left"],
			"y": safeMargins["top"],
			"w": 700.0,
			"h": 20.0,
		}
	}
	appendSlideElement(slide, elem)
}

func addFooterElement(slide map[string]any, slots map[string]map[string]any, theme map[string]any, safeMargins map[string]float64) {
	elem := map[string]any{
		"id":   fmt.Sprintf("%v_footer", slide["id"]),
		"type": "text",
		"text": map[string]any{
			"content": "{page}/{total_pages}",
			"runs":    []any{},
			"autoFit": "shrink",
			"style": map[string]any{
				"fontSize": 10,
				"color":    themeMutedText(theme),
				"align":    "right",
			},
		},
	}
	if slot, ok := slots["footer"]; ok {
		elem["slotId"] = "footer"
		if rect, ok := rectFromSlot(slot, theme); ok {
			elem["rect"] = rect
		}
	} else {
		elem["rect"] = map[string]any{
			"x": safeMargins["left"],
			"y": 540.0 - safeMargins["bottom"] - 18.0,
			"w": 960.0 - safeMargins["left"] - safeMargins["right"],
			"h": 18.0,
		}
	}
	appendSlideElement(slide, elem)
}

func appendSlideElement(slide map[string]any, elem map[string]any) {
	elements, _ := slide["elements"].([]any)
	slide["elements"] = append(elements, elem)
}

func themeMutedText(theme map[string]any) string {
	if theme == nil {
		return "#6B7280"
	}
	if colors, ok := theme["colors"].(map[string]any); ok {
		if semantic, ok := colors["semantic"].(map[string]any); ok {
			if muted, ok := semantic["mutedText"].(string); ok && muted != "" {
				return muted
			}
		}
	}
	return "#6B7280"
}

func extractSafeMargins(theme map[string]any) map[string]float64 {
	margins := map[string]float64{"top": 36, "right": 36, "bottom": 36, "left": 36}
	if theme == nil {
		return margins
	}
	canvas, _ := theme["canvas"].(map[string]any)
	if canvas == nil {
		return margins
	}
	safe, _ := canvas["safeMargins"].(map[string]any)
	if safe == nil {
		return margins
	}
	if v, ok := safe["top"].(float64); ok {
		margins["top"] = v
	}
	if v, ok := safe["right"].(float64); ok {
		margins["right"] = v
	}
	if v, ok := safe["bottom"].(float64); ok {
		margins["bottom"] = v
	}
	if v, ok := safe["left"].(float64); ok {
		margins["left"] = v
	}
	return margins
}

func rectFromSlot(slot map[string]any, theme map[string]any) (map[string]any, bool) {
	if slot == nil {
		return nil, false
	}
	if theme == nil {
		theme = map[string]any{}
	}
	if rect, ok := slot["rect"].(map[string]any); ok {
		return rect, true
	}
	gridSpec, _ := slot["grid"].(map[string]any)
	if gridSpec == nil {
		return nil, false
	}
	col := int(asFloat(gridSpec["col"]))
	span := int(asFloat(gridSpec["span"]))
	if col <= 0 || span <= 0 {
		return nil, false
	}

	themeGrid, _ := theme["grid"].(map[string]any)
	columns := int(asFloat(themeGrid["columns"]))
	if columns <= 0 {
		columns = 12
	}
	gutter := asFloat(themeGrid["gutter"])
	baseline := asFloat(themeGrid["baseline"])
	if baseline <= 0 {
		baseline = 8
	}
	snapOn := true
	if snapRaw, ok := themeGrid["snap"].(bool); ok {
		snapOn = snapRaw
	}

	margins := extractSafeMargins(theme)
	usableW := 960.0 - margins["left"] - margins["right"]
	colW := (usableW - gutter*float64(columns-1)) / float64(columns)
	x := margins["left"] + float64(col-1)*(colW+gutter)
	w := float64(span)*colW + float64(span-1)*gutter
	y := asFloat(slot["y"])
	h := asFloat(slot["h"])
	if snapOn {
		y = math.Round(y/baseline) * baseline
		h = math.Round(h/baseline) * baseline
	}
	return map[string]any{"x": x, "y": y, "w": w, "h": h}, true
}

func asFloat(val any) float64 {
	switch v := val.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		if f, err := v.Float64(); err == nil {
			return f
		}
	}
	return 0
}

func validateDeck(deck map[string]any, expectedSlides int) error {
	var issues []string
	slides, _ := deck["slides"].([]any)
	if expectedSlides > 0 && len(slides) != expectedSlides {
		issues = append(issues, fmt.Sprintf("expected %d slides, got %d", expectedSlides, len(slides)))
	}

	layoutIDs := templateLayoutIDs(deck["layouts"])
	componentIDs := buildComponentIDSet(deck["components"])
	theme, _ := deck["theme"].(map[string]any)
	safeMargins := extractSafeMargins(theme)
	layoutSlots := buildLayoutSlotMap(deck["layouts"])

	assetIDs := map[string]bool{}
	if assets, ok := deck["assets"].(map[string]any); ok {
		if images, ok := assets["images"].([]any); ok {
			for _, imgAny := range images {
				if img, ok := imgAny.(map[string]any); ok {
					if id, ok := img["id"].(string); ok && id != "" {
						assetIDs[id] = true
					}
				}
			}
		}
	}

	datasetIDs := map[string]bool{}
	if data, ok := deck["data"].(map[string]any); ok {
		if datasets, ok := data["datasets"].([]any); ok {
			for _, dsAny := range datasets {
				if ds, ok := dsAny.(map[string]any); ok {
					if id, ok := ds["id"].(string); ok && id != "" {
						datasetIDs[id] = true
					}
				}
			}
		}
	}

	orderSeen := map[int]bool{}
	idSeen := map[string]bool{}

	for idx, slideAny := range slides {
		slide, ok := slideAny.(map[string]any)
		if !ok {
			issues = append(issues, fmt.Sprintf("slide %d is not an object", idx+1))
			continue
		}
		order, _ := parseIntFromInterface(slide["order"])
		if order <= 0 {
			issues = append(issues, fmt.Sprintf("slide %d has invalid order", idx+1))
		} else {
			orderSeen[order] = true
		}
		if id, ok := slide["id"].(string); ok && id != "" {
			if idSeen[id] {
				issues = append(issues, fmt.Sprintf("duplicate slide id %s", id))
			}
			idSeen[id] = true
		} else {
			issues = append(issues, fmt.Sprintf("slide %d missing id", idx+1))
		}

		layoutID, _ := slide["layoutId"].(string)
		if layoutID == "" || !layoutIDs[layoutID] {
			issues = append(issues, fmt.Sprintf("slide %d has unknown layoutId %q", idx+1, layoutID))
		}

		if isContentSlide(layoutID) {
			useComponents, _ := slide["useComponents"].([]any)
			hasHeaderComponent := false
			hasFooterComponent := false
			for _, comp := range useComponents {
				if compID, ok := comp.(string); ok {
					if compID == "header" {
						hasHeaderComponent = true
					}
					if compID == "footer" {
						hasFooterComponent = true
					}
				}
			}
			if componentIDs["header"] && !hasHeaderComponent && !hasSlotOrElement(slide, "header") {
				issues = append(issues, fmt.Sprintf("slide %d missing header", idx+1))
			}
			if componentIDs["footer"] && !hasFooterComponent && !hasSlotOrElement(slide, "footer") {
				issues = append(issues, fmt.Sprintf("slide %d missing footer", idx+1))
			}
			if !componentIDs["header"] && !hasSlotOrElement(slide, "header") {
				issues = append(issues, fmt.Sprintf("slide %d missing header", idx+1))
			}
			if !componentIDs["footer"] && !hasSlotOrElement(slide, "footer") {
				issues = append(issues, fmt.Sprintf("slide %d missing footer", idx+1))
			}
		}

		elements, _ := slide["elements"].([]any)
		for _, elemAny := range elements {
			elem, ok := elemAny.(map[string]any)
			if !ok {
				continue
			}
			if slotID, ok := elem["slotId"].(string); ok && slotID != "" {
				if slotsForLayout, ok := layoutSlots[layoutID]; ok {
					if _, exists := slotsForLayout[slotID]; !exists {
						issues = append(issues, fmt.Sprintf("slide %d uses unknown slotId %q", idx+1, slotID))
					}
				}
			}
			if rect, ok := elem["rect"].(map[string]any); ok {
				x := asFloat(rect["x"])
				y := asFloat(rect["y"])
				w := asFloat(rect["w"])
				h := asFloat(rect["h"])
				if x < 0 || y < 0 || x+w > 960 || y+h > 540 {
					issues = append(issues, fmt.Sprintf("slide %d element %v out of bounds", idx+1, elem["id"]))
				}
				if elemType, _ := elem["type"].(string); elemType == "text" {
					if x < safeMargins["left"] || y < safeMargins["top"] || x+w > 960-safeMargins["right"] || y+h > 540-safeMargins["bottom"] {
						issues = append(issues, fmt.Sprintf("slide %d text element %v violates safe margins", idx+1, elem["id"]))
					}
				}
			}
			if elemType, _ := elem["type"].(string); elemType == "chart" {
				chart, _ := elem["chart"].(map[string]any)
				if chart != nil {
					if ref, ok := chart["datasetRef"].(string); ok && ref != "" && !datasetIDs[ref] {
						issues = append(issues, fmt.Sprintf("slide %d chart datasetRef %q missing", idx+1, ref))
					}
				}
			}
			if elemType, _ := elem["type"].(string); elemType == "image" {
				image, _ := elem["image"].(map[string]any)
				if image != nil {
					if ref, ok := image["ref"].(string); ok && ref != "" && !assetIDs[ref] {
						issues = append(issues, fmt.Sprintf("slide %d image ref %q missing", idx+1, ref))
					}
				}
			}
		}
	}

	for i := 1; i <= len(slides); i++ {
		if !orderSeen[i] {
			issues = append(issues, fmt.Sprintf("missing slide order %d", i))
		}
	}

	if len(issues) > 0 {
		log.Debug().
			Int("issue_count", len(issues)).
			Msg("[slide_generator] deck validation issues found")
		return fmt.Errorf("deck validation failed: %s", strings.Join(issues, "; "))
	}
	log.Debug().Msg("[slide_generator] deck validation passed")
	return nil
}

