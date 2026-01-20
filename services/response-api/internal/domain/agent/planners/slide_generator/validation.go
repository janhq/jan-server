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
				// First pass: fix invalid slotIds by removing them or providing fallback rects
				filteredElements := make([]any, 0, len(elements))
				for _, elemAny := range elements {
					elem, ok := elemAny.(map[string]any)
					if !ok {
						filteredElements = append(filteredElements, elemAny)
						continue
					}

					slotID, hasSlotID := elem["slotId"].(string)
					if hasSlotID && slotID != "" {
						// Check if this slotId exists in the layout
						if _, slotExists := slotsForLayout[slotID]; !slotExists {
							// Invalid slotId - need to fix or remove
							slideID, _ := slide["id"].(string)
							log.Debug().
								Str("slide_id", slideID).
								Str("layout_id", layoutID).
								Str("invalid_slot_id", slotID).
								Msg("[slide_generator] fixing invalid slotId reference")

							// For non-content slides (TITLE, SECTION_HEADER, CLOSING),
							// remove header/footer elements entirely as they're not supposed to be there
							if !isContentSlide(layoutID) && (slotID == "header" || slotID == "footer") {
								log.Debug().
									Str("slide_id", slideID).
									Str("slot_id", slotID).
									Msg("[slide_generator] removing header/footer from non-content slide")
								continue // Skip this element
							}

							// For other cases, remove slotId and provide fallback rect
							delete(elem, "slotId")
							if _, hasRect := elem["rect"].(map[string]any); !hasRect {
								// Provide a sensible fallback rect based on element type
								fallbackRect := getFallbackRectForElement(elem, slotID, safeMargins)
								elem["rect"] = fallbackRect
								log.Debug().
									Str("slide_id", slideID).
									Str("removed_slot_id", slotID).
									Interface("fallback_rect", fallbackRect).
									Msg("[slide_generator] replaced invalid slotId with fallback rect")
							}
						}
					}

					filteredElements = append(filteredElements, elem)
				}
				slide["elements"] = filteredElements

				// Second pass: validate element types and ensure required sub-objects exist (P0 fix)
				validatedElements := make([]any, 0, len(filteredElements))
				for _, elemAny := range filteredElements {
					elem, ok := elemAny.(map[string]any)
					if !ok {
						continue
					}
					// Validate and sanitize element by type - drop invalid elements
					if !validateAndSanitizeElement(elem) {
						slideID, _ := slide["id"].(string)
						elemType, _ := elem["type"].(string)
						log.Debug().
							Str("slide_id", slideID).
							Str("element_type", elemType).
							Msg("[slide_generator] dropping invalid element missing required sub-object")
						continue
					}
					validatedElements = append(validatedElements, elem)
				}
				slide["elements"] = validatedElements

				// Third pass: normalize elements
				for _, elemAny := range validatedElements {
					elem, ok := elemAny.(map[string]any)
					if !ok {
						continue
					}
					if elemType, _ := elem["type"].(string); elemType == "table" {
						normalizeTableElement(elem)
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
					// P0 fix: Ensure every element ends with a usable rect (fallback if neither rect nor slotId)
					if _, hasRect := elem["rect"].(map[string]any); !hasRect {
						elemType, _ := elem["type"].(string)
						fallbackRect := getFallbackRectForElementType(elemType, safeMargins)
						elem["rect"] = fallbackRect
						slideID, _ := slide["id"].(string)
						log.Debug().
							Str("slide_id", slideID).
							Str("element_type", elemType).
							Interface("fallback_rect", fallbackRect).
							Msg("[slide_generator] added fallback rect for element missing positioning")
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
						// Ensure fill has a valid default (P0 fix: prevent python crash on nil fill)
						if fill, ok := style["fill"].(map[string]any); !ok || fill == nil {
							style["fill"] = map[string]any{"type": "solid", "color": "#FFFFFF"}
						}
						for _, k := range []string{"stroke", "cornerRadius", "shadow"} {
							if _, has := style[k]; !has {
								style[k] = nil
							}
						}
						style["cornerRadius"] = nil
						style["shadow"] = nil

						fixStyleProperties(style, allowedShapeStyleProps)
					}
				}

				// Fourth pass: auto-fix safe margin violations by clamping text elements
				for _, elemAny := range slide["elements"].([]any) {
					elem, ok := elemAny.(map[string]any)
					if !ok {
						continue
					}
					elemType, _ := elem["type"].(string)
					if elemType != "text" {
						continue // Safe margin check only applies to text elements
					}
					rect, ok := elem["rect"].(map[string]any)
					if !ok {
						continue
					}
					if clampRectToSafeMargins(rect, safeMargins) {
						slideID, _ := slide["id"].(string)
						elemID, _ := elem["id"].(string)
						log.Debug().
							Str("slide_id", slideID).
							Str("element_id", elemID).
							Interface("clamped_rect", rect).
							Msg("[slide_generator] clamped text element to safe margins")
					}
				}
			}
		}
	}

	return deck
}

// validateAndSanitizeElement checks that each element type has its required sub-object.
// Returns true if the element is valid (or was successfully auto-fixed), false if it should be dropped.
// P0 fix: Prevents python crashes from malformed elements like {"type":"chart"} without chart:{...}
func validateAndSanitizeElement(elem map[string]any) bool {
	elemType, ok := elem["type"].(string)
	if !ok || elemType == "" {
		return false // No type = invalid element
	}

	switch elemType {
	case "text":
		// Text must have text.content and text.style
		text, ok := elem["text"].(map[string]any)
		if !ok || text == nil {
			// Auto-fix: create minimal text object
			elem["text"] = map[string]any{
				"content": "",
				"runs":    []any{},
				"autoFit": "shrink",
				"style":   map[string]any{},
			}
		} else {
			if _, hasContent := text["content"].(string); !hasContent {
				text["content"] = ""
			}
			if _, hasStyle := text["style"].(map[string]any); !hasStyle {
				text["style"] = map[string]any{}
			}
		}
		return true

	case "shape":
		// Shape must have shape.kind and shape.style
		shape, ok := elem["shape"].(map[string]any)
		if !ok || shape == nil {
			// Auto-fix: create minimal shape object
			elem["shape"] = map[string]any{
				"kind": "rect",
				"style": map[string]any{
					"fill": map[string]any{"type": "solid", "color": "#FFFFFF"},
				},
			}
		} else {
			if _, hasKind := shape["kind"].(string); !hasKind {
				shape["kind"] = "rect"
			}
			if _, hasStyle := shape["style"].(map[string]any); !hasStyle {
				shape["style"] = map[string]any{
					"fill": map[string]any{"type": "solid", "color": "#FFFFFF"},
				}
			}
		}
		return true

	case "image":
		// Image must have image.ref
		image, ok := elem["image"].(map[string]any)
		if !ok || image == nil {
			return false // Cannot auto-fix missing image - drop element
		}
		if ref, hasRef := image["ref"].(string); !hasRef || ref == "" {
			return false // Image without ref is invalid
		}
		return true

	case "chart":
		// Chart must have chart.chartType and chart.datasetRef
		chart, ok := elem["chart"].(map[string]any)
		if !ok || chart == nil {
			return false // Cannot auto-fix missing chart - drop element
		}
		if chartType, has := chart["chartType"].(string); !has || chartType == "" {
			return false // Chart without chartType is invalid
		}
		if datasetRef, has := chart["datasetRef"].(string); !has || datasetRef == "" {
			return false // Chart without datasetRef is invalid
		}
		return true

	case "table":
		// Table must have table.columns and table.rows
		table, ok := elem["table"].(map[string]any)
		if !ok || table == nil {
			return false // Cannot auto-fix missing table - drop element
		}
		if cols, ok := table["columns"].([]any); !ok || len(cols) == 0 {
			return false // Table without columns is invalid
		}
		if rows, ok := table["rows"].([]any); !ok || len(rows) == 0 {
			return false // Table without rows is invalid
		}
		return true

	case "group":
		// Group should have children array
		if _, ok := elem["group"].(map[string]any); !ok {
			elem["group"] = map[string]any{"children": []any{}}
		} else if group := elem["group"].(map[string]any); group != nil {
			if _, hasChildren := group["children"].([]any); !hasChildren {
				group["children"] = []any{}
			}
		}
		return true

	default:
		// Unknown element types pass through (may be extension types)
		return true
	}
}

func normalizeTableElement(elem map[string]any) {
	table, _ := elem["table"].(map[string]any)
	if table == nil {
		return
	}

	if cols, ok := table["columns"].([]any); ok {
		for i, colAny := range cols {
			switch col := colAny.(type) {
			case string:
				continue
			case map[string]any:
				header := firstStringValue(col, "header", "title", "name", "label", "text")
				if header == "" {
					header = fmt.Sprintf("%v", col)
				}
				col["header"] = header
				cols[i] = col
			case nil:
				cols[i] = ""
			default:
				cols[i] = fmt.Sprintf("%v", col)
			}
		}
		table["columns"] = cols
	}

	if rows, ok := table["rows"].([]any); ok {
		for rIdx, rowAny := range rows {
			row, ok := rowAny.([]any)
			if !ok {
				continue
			}
			for cIdx, cellAny := range row {
				switch cell := cellAny.(type) {
				case string:
					continue
				case map[string]any:
					text := firstStringValue(cell, "text", "value", "label", "name")
					if text == "" {
						text = fmt.Sprintf("%v", cell)
					}
					row[cIdx] = text
				case nil:
					row[cIdx] = ""
				default:
					row[cIdx] = fmt.Sprintf("%v", cell)
				}
			}
			rows[rIdx] = row
		}
		table["rows"] = rows
	}
}

func firstStringValue(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if val, ok := values[key].(string); ok && strings.TrimSpace(val) != "" {
			return val
		}
	}
	return ""
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

// getFallbackRectForElement provides sensible fallback coordinates when an element
// references a slotId that doesn't exist in the layout. This prevents validation
// errors and ensures the element can still be rendered.
func getFallbackRectForElement(elem map[string]any, originalSlotID string, safeMargins map[string]float64) map[string]any {
	// Default fallback based on common slot patterns
	switch originalSlotID {
	case "header":
		return map[string]any{
			"x": safeMargins["left"],
			"y": safeMargins["top"],
			"w": 700.0,
			"h": 20.0,
		}
	case "footer":
		return map[string]any{
			"x": safeMargins["left"],
			"y": 540.0 - safeMargins["bottom"] - 18.0,
			"w": 960.0 - safeMargins["left"] - safeMargins["right"],
			"h": 18.0,
		}
	case "title":
		return map[string]any{
			"x": safeMargins["left"],
			"y": 72.0,
			"w": 888.0,
			"h": 48.0,
		}
	case "subtitle":
		return map[string]any{
			"x": safeMargins["left"],
			"y": 130.0,
			"w": 888.0,
			"h": 40.0,
		}
	case "body":
		return map[string]any{
			"x": safeMargins["left"],
			"y": 140.0,
			"w": 888.0,
			"h": 320.0,
		}
	case "left":
		return map[string]any{
			"x": safeMargins["left"],
			"y": 140.0,
			"w": 432.0,
			"h": 320.0,
		}
	case "right":
		return map[string]any{
			"x": 492.0,
			"y": 140.0,
			"w": 432.0,
			"h": 320.0,
		}
	case "table", "chart":
		return map[string]any{
			"x": safeMargins["left"],
			"y": 140.0,
			"w": 888.0,
			"h": 320.0,
		}
	case "image":
		// Check if it's a full-bleed image
		elemType, _ := elem["type"].(string)
		if elemType == "image" {
			return map[string]any{
				"x": 0.0,
				"y": 0.0,
				"w": 960.0,
				"h": 540.0,
			}
		}
		return map[string]any{
			"x": safeMargins["left"],
			"y": 140.0,
			"w": 432.0,
			"h": 320.0,
		}
	default:
		// Generic fallback: center the element
		return map[string]any{
			"x": safeMargins["left"],
			"y": 140.0,
			"w": 888.0,
			"h": 300.0,
		}
	}
}

// getFallbackRectForElementType provides fallback coordinates based on element type
// when an element has neither rect nor slotId. P0 fix to ensure all elements can be rendered.
func getFallbackRectForElementType(elemType string, safeMargins map[string]float64) map[string]any {
	switch elemType {
	case "text":
		return map[string]any{
			"x": safeMargins["left"],
			"y": 140.0,
			"w": 888.0,
			"h": 280.0,
		}
	case "image":
		return map[string]any{
			"x": safeMargins["left"],
			"y": 140.0,
			"w": 432.0,
			"h": 320.0,
		}
	case "chart":
		return map[string]any{
			"x": safeMargins["left"],
			"y": 140.0,
			"w": 600.0,
			"h": 320.0,
		}
	case "table":
		return map[string]any{
			"x": safeMargins["left"],
			"y": 140.0,
			"w": 888.0,
			"h": 320.0,
		}
	case "shape":
		return map[string]any{
			"x": safeMargins["left"],
			"y": 140.0,
			"w": 200.0,
			"h": 100.0,
		}
	default:
		return map[string]any{
			"x": safeMargins["left"],
			"y": 140.0,
			"w": 888.0,
			"h": 300.0,
		}
	}
}

// clampRectToSafeMargins adjusts a rect to fit within safe margins.
// Returns true if the rect was modified, false otherwise.
// This is an auto-fix for layout issues - not a schema error.
func clampRectToSafeMargins(rect map[string]any, safeMargins map[string]float64) bool {
	canvasW := 960.0
	canvasH := 540.0
	left := safeMargins["left"]
	top := safeMargins["top"]
	right := safeMargins["right"]
	bottom := safeMargins["bottom"]

	x := asFloat(rect["x"])
	y := asFloat(rect["y"])
	w := asFloat(rect["w"])
	h := asFloat(rect["h"])

	modified := false

	// Clamp x to be within left safe margin
	if x < left {
		x = left
		modified = true
	}

	// Clamp y to be within top safe margin
	if y < top {
		y = top
		modified = true
	}

	// Ensure element doesn't extend beyond right safe margin
	maxX := canvasW - right
	if x+w > maxX {
		// First try reducing width
		if x < maxX {
			w = maxX - x
			modified = true
		} else {
			// Element starts too far right - shift it left
			x = left
			w = maxX - left
			modified = true
		}
	}

	// Ensure element doesn't extend beyond bottom safe margin
	maxY := canvasH - bottom
	if y+h > maxY {
		// First try reducing height
		if y < maxY {
			h = maxY - y
			modified = true
		} else {
			// Element starts too low - shift it up
			y = top
			h = maxY - top
			modified = true
		}
	}

	// Ensure minimum dimensions
	if w < 10 {
		w = 10
		modified = true
	}
	if h < 10 {
		h = 10
		modified = true
	}

	if modified {
		rect["x"] = x
		rect["y"] = y
		rect["w"] = w
		rect["h"] = h
	}

	return modified
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
