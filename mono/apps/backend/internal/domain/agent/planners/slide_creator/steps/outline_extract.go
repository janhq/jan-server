package steps

import (
	"regexp"
	"strings"
)

// OutlineSlideContent represents extracted structured content from an outline block.
type OutlineSlideContent struct {
	Index        int
	Title        string
	Bullets      []string
	TableData    *ExtractedTable
	ChartHint    *ExtractedChartHint
	ImageHint    string
	RawText      string
	HasDataNeeds bool
}

// ExtractedTable represents a table parsed from outline text.
type ExtractedTable struct {
	Title   string
	Headers []string
	Rows    [][]string
}

// ExtractedChartHint captures chart requirements from outline.
type ExtractedChartHint struct {
	Type       string // bar, line, pie
	Title      string
	Categories []string
	Values     []float64
	YAxisLabel string
	XAxisLabel string
}

// ExtractedDataPoint represents a single data point from inline data.
type ExtractedDataPoint struct {
	Label string
	Value float64
}

var (
	pipeTableRowRE    = regexp.MustCompile(`^\s*\|(.+)\|\s*$`)
	chartHintRE       = regexp.MustCompile(`(?i)\b(bar|line|pie)\s*(chart|graph)\b`)
	percentageRE      = regexp.MustCompile(`(\d+(?:\.\d+)?)\s*%`)
	numberValueRE     = regexp.MustCompile(`(\d+(?:,\d{3})*(?:\.\d+)?)\s*(?:M|B|K|T|million|billion|thousand|trillion)?`)
	inlineDataRE      = regexp.MustCompile(`(?i)([A-Za-z]+)\s*:\s*\$?([\d.,]+)\s*([MBKT]?)`)
	xAxisRE           = regexp.MustCompile(`(?i)x-?\s*axis\s*:\s*(.+)`)
	yAxisRE           = regexp.MustCompile(`(?i)y-?\s*axis\s*:\s*(.+)`)
	dataPointsLabelRE = regexp.MustCompile(`(?i)data\s*points?\s*[:\(]`)
)

// ExtractSlideContent parses an outline block and extracts structured content.
func ExtractSlideContent(block outlineBlock) OutlineSlideContent {
	content := OutlineSlideContent{
		Index:   block.Index,
		Title:   block.Title,
		RawText: outlineBlockText(block),
	}

	bullets := []string{}
	tableLines := []string{}
	inTable := false
	var chartDataPoints []ExtractedDataPoint
	var xAxisCategories []string
	var yAxisLabel string

	for _, line := range block.Lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if inTable && len(tableLines) > 0 {
				inTable = false
			}
			continue
		}

		// Check for pipe-delimited table row
		if pipeTableRowRE.MatchString(trimmed) {
			inTable = true
			tableLines = append(tableLines, trimmed)
			continue
		}

		// End table if we hit non-table content
		if inTable {
			inTable = false
		}

		// Check for X-axis categories
		if matches := xAxisRE.FindStringSubmatch(trimmed); matches != nil {
			xAxisCategories = parseCommaSeparatedValues(matches[1])
			continue
		}

		// Check for Y-axis label
		if matches := yAxisRE.FindStringSubmatch(trimmed); matches != nil {
			yAxisLabel = strings.TrimSpace(matches[1])
			continue
		}

		// Check for inline data series (e.g., "Jan: $1.7T | Feb: $1.8T")
		if dataPointsLabelRE.MatchString(trimmed) || strings.Contains(trimmed, ": $") || strings.Contains(trimmed, ": 1.") || strings.Contains(trimmed, ": 2.") {
			points := parseInlineDataSeries(trimmed)
			if len(points) > 0 {
				chartDataPoints = append(chartDataPoints, points...)
				continue
			}
		}

		// Check for bullet points
		if bullet := parseBulletLine(trimmed); bullet != "" {
			bullets = append(bullets, bullet)
			continue
		}

		// Check for chart hints
		if chartHintRE.MatchString(trimmed) && content.ChartHint == nil {
			content.ChartHint = parseChartHint(trimmed)
		}

		// Check for image hints
		if outlineImageMarker.MatchString(trimmed) && content.ImageHint == "" {
			content.ImageHint = extractImageHint(trimmed)
		}
	}

	content.Bullets = bullets

	// Parse table if found
	if len(tableLines) >= 2 {
		content.TableData = parseTableFromLines(tableLines)
	}

	// If we found chart data points, add them to the chart hint
	if len(chartDataPoints) > 0 {
		if content.ChartHint == nil {
			content.ChartHint = &ExtractedChartHint{Type: "line"}
		}
		content.ChartHint.Categories = make([]string, len(chartDataPoints))
		content.ChartHint.Values = make([]float64, len(chartDataPoints))
		for i, dp := range chartDataPoints {
			content.ChartHint.Categories[i] = dp.Label
			content.ChartHint.Values[i] = dp.Value
		}
		if len(xAxisCategories) > 0 {
			content.ChartHint.Categories = xAxisCategories
		}
		if yAxisLabel != "" {
			content.ChartHint.YAxisLabel = yAxisLabel
		}
	}

	// Determine if data bank is needed
	content.HasDataNeeds = content.TableData != nil || content.ChartHint != nil ||
		outlineBlockNeedsDataBank(block)

	return content
}

// parseBulletLine extracts bullet content from a line.
func parseBulletLine(line string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}

	// Common bullet prefixes
	prefixes := []string{"- ", "* ", "• ", "→ ", "> "}
	for _, prefix := range prefixes {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}

	// Numbered bullets: "1. ", "1) "
	if len(line) >= 3 {
		if line[0] >= '0' && line[0] <= '9' {
			for i := 1; i < len(line) && i < 4; i++ {
				if line[i] == '.' || line[i] == ')' {
					if i+1 < len(line) && line[i+1] == ' ' {
						return strings.TrimSpace(line[i+2:])
					}
				}
				if line[i] < '0' || line[i] > '9' {
					break
				}
			}
		}
	}

	return ""
}

// parseTableFromLines converts pipe-delimited lines to a table.
func parseTableFromLines(lines []string) *ExtractedTable {
	if len(lines) < 2 {
		return nil
	}

	table := &ExtractedTable{}
	for i, line := range lines {
		cells := parsePipeRow(line)
		if len(cells) == 0 {
			continue
		}

		// Skip separator rows (e.g., |---|---|)
		if isSeparatorRow(cells) {
			continue
		}

		if len(table.Headers) == 0 {
			table.Headers = cells
		} else {
			// Normalize row to match header count
			row := make([]string, len(table.Headers))
			for j := 0; j < len(table.Headers); j++ {
				if j < len(cells) {
					row[j] = cells[j]
				}
			}
			table.Rows = append(table.Rows, row)
		}

		// Limit rows to prevent oversized tables
		if len(table.Rows) >= 9 {
			break
		}

		_ = i
	}

	if len(table.Headers) == 0 || len(table.Rows) == 0 {
		return nil
	}

	// Limit columns
	if len(table.Headers) > 6 {
		table.Headers = table.Headers[:6]
		for i := range table.Rows {
			if len(table.Rows[i]) > 6 {
				table.Rows[i] = table.Rows[i][:6]
			}
		}
	}

	return table
}

// parsePipeRow splits a pipe-delimited row into cells.
func parsePipeRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.Trim(line, "|")

	parts := strings.Split(line, "|")
	cells := make([]string, 0, len(parts))
	for _, part := range parts {
		cell := strings.TrimSpace(part)
		if cell != "" {
			cells = append(cells, cell)
		}
	}
	return cells
}

// isSeparatorRow checks if all cells are separator patterns (e.g., ---, :--:).
func isSeparatorRow(cells []string) bool {
	for _, cell := range cells {
		cell = strings.TrimSpace(cell)
		cell = strings.Trim(cell, ":")
		if cell == "" {
			continue
		}
		for _, r := range cell {
			if r != '-' {
				return false
			}
		}
	}
	return true
}

// parseChartHint extracts chart type and title from a line.
func parseChartHint(line string) *ExtractedChartHint {
	hint := &ExtractedChartHint{}

	lower := strings.ToLower(line)
	if strings.Contains(lower, "bar") {
		hint.Type = "bar"
	} else if strings.Contains(lower, "line") {
		hint.Type = "line"
	} else if strings.Contains(lower, "pie") {
		hint.Type = "pie"
	} else {
		hint.Type = "bar" // default
	}

	// Extract title - text before "chart" or "graph"
	chartIdx := strings.Index(lower, "chart")
	if chartIdx < 0 {
		chartIdx = strings.Index(lower, "graph")
	}
	if chartIdx > 0 {
		title := strings.TrimSpace(line[:chartIdx])
		title = strings.TrimSuffix(title, "bar")
		title = strings.TrimSuffix(title, "line")
		title = strings.TrimSuffix(title, "pie")
		hint.Title = strings.TrimSpace(title)
	}

	return hint
}

// extractImageHint extracts image requirement description.
func extractImageHint(line string) string {
	// Find image-related keywords and extract context
	line = strings.TrimSpace(line)
	if len(line) > 100 {
		return line[:100]
	}
	return line
}

// FormatSlideContentForPrompt formats extracted content for LLM prompt.
func FormatSlideContentForPrompt(content OutlineSlideContent) string {
	var sb strings.Builder

	if content.Title != "" {
		sb.WriteString("Slide Title: ")
		sb.WriteString(content.Title)
		sb.WriteString("\n\n")
	}

	if len(content.Bullets) > 0 {
		sb.WriteString("Key Points:\n")
		for _, bullet := range content.Bullets {
			sb.WriteString("- ")
			sb.WriteString(bullet)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	if content.TableData != nil {
		sb.WriteString("Table Data:\n")
		sb.WriteString("Headers: ")
		sb.WriteString(strings.Join(content.TableData.Headers, " | "))
		sb.WriteString("\n")
		for _, row := range content.TableData.Rows {
			sb.WriteString("Row: ")
			sb.WriteString(strings.Join(row, " | "))
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	if content.ChartHint != nil {
		sb.WriteString("Chart Required: ")
		sb.WriteString(content.ChartHint.Type)
		if content.ChartHint.Title != "" {
			sb.WriteString(" - ")
			sb.WriteString(content.ChartHint.Title)
		}
		sb.WriteString("\n")
		if content.ChartHint.YAxisLabel != "" {
			sb.WriteString("Y-Axis: ")
			sb.WriteString(content.ChartHint.YAxisLabel)
			sb.WriteString("\n")
		}
		if len(content.ChartHint.Categories) > 0 && len(content.ChartHint.Values) > 0 {
			sb.WriteString("Data Points:\n")
			for i := 0; i < len(content.ChartHint.Categories) && i < len(content.ChartHint.Values); i++ {
				sb.WriteString("  ")
				sb.WriteString(content.ChartHint.Categories[i])
				sb.WriteString(": ")
				sb.WriteString(formatChartValue(content.ChartHint.Values[i]))
				sb.WriteString("\n")
			}
		}
		sb.WriteString("\n")
	}

	if content.ImageHint != "" {
		sb.WriteString("Image Note: ")
		sb.WriteString(content.ImageHint)
		sb.WriteString("\n\n")
	}

	if content.RawText != "" && sb.Len() < 500 {
		// Include raw text if structured extraction was minimal
		sb.WriteString("Additional Context:\n")
		rawText := content.RawText
		if len(rawText) > 1500 {
			rawText = rawText[:1500] + "..."
		}
		sb.WriteString(rawText)
	}

	return strings.TrimSpace(sb.String())
}

// ConvertExtractedTableToTableData converts an extracted table to the SlidePlan format.
func ConvertExtractedTableToTableData(ext *ExtractedTable) *TableData {
	if ext == nil || len(ext.Headers) == 0 || len(ext.Rows) == 0 {
		return nil
	}

	rows := make([][]any, 0, len(ext.Rows))
	for _, row := range ext.Rows {
		anyRow := make([]any, len(row))
		for i, cell := range row {
			anyRow[i] = cell
		}
		rows = append(rows, anyRow)
	}

	return &TableData{
		Title:   ext.Title,
		Columns: ext.Headers,
		Rows:    rows,
	}
}

// parseInlineDataSeries parses data series like "Jan: $1.7T | Feb: $1.8T | Mar: $1.9T"
func parseInlineDataSeries(line string) []ExtractedDataPoint {
	var points []ExtractedDataPoint

	// Split by | or ∣ (both pipe characters)
	parts := strings.FieldsFunc(line, func(r rune) bool {
		return r == '|' || r == '∣'
	})

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Match patterns like "Jan: $1.7T" or "Jan: 1.7T" or "Jan: 100,000"
		matches := inlineDataRE.FindStringSubmatch(part)
		if matches != nil && len(matches) >= 3 {
			label := strings.TrimSpace(matches[1])
			valueStr := strings.ReplaceAll(matches[2], ",", "")
			multiplier := 1.0

			if len(matches) >= 4 {
				switch strings.ToUpper(matches[3]) {
				case "T":
					multiplier = 1e12
				case "B":
					multiplier = 1e9
				case "M":
					multiplier = 1e6
				case "K":
					multiplier = 1e3
				}
			}

			value := parseFloatSimple(valueStr)
			if value > 0 {
				points = append(points, ExtractedDataPoint{
					Label: label,
					Value: value * multiplier,
				})
			}
		}
	}

	return points
}

// parseCommaSeparatedValues parses comma-separated values like "Jan, Feb, Mar, ..."
func parseCommaSeparatedValues(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

// parseFloatSimple parses a simple float from string
func parseFloatSimple(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	var val float64
	var decimal float64 = 0.1
	inDecimal := false

	for _, r := range s {
		if r >= '0' && r <= '9' {
			if inDecimal {
				val += float64(r-'0') * decimal
				decimal /= 10
			} else {
				val = val*10 + float64(r-'0')
			}
		} else if r == '.' {
			inDecimal = true
		}
	}

	return val
}

// formatChartValue formats a chart value for display
func formatChartValue(v float64) string {
	if v >= 1e12 {
		return formatFloat(v/1e12) + "T"
	}
	if v >= 1e9 {
		return formatFloat(v/1e9) + "B"
	}
	if v >= 1e6 {
		return formatFloat(v/1e6) + "M"
	}
	if v >= 1e3 {
		return formatFloat(v/1e3) + "K"
	}
	return formatFloat(v)
}

// formatFloat formats a float with up to 2 decimal places
func formatFloat(v float64) string {
	intPart := int(v)
	fracPart := int((v - float64(intPart)) * 100)
	if fracPart == 0 {
		return intToStr(intPart)
	}
	if fracPart%10 == 0 {
		return intToStr(intPart) + "." + intToStr(fracPart/10)
	}
	return intToStr(intPart) + "." + intToStr(fracPart)
}

// intToStr converts int to string
func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}
