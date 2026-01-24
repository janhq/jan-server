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
	Values     []string
}

var (
	pipeTableRowRE = regexp.MustCompile(`^\s*\|(.+)\|\s*$`)
	chartHintRE    = regexp.MustCompile(`(?i)\b(bar|line|pie)\s*(chart|graph)\b`)
	percentageRE   = regexp.MustCompile(`(\d+(?:\.\d+)?)\s*%`)
	numberValueRE  = regexp.MustCompile(`(\d+(?:,\d{3})*(?:\.\d+)?)\s*(?:M|B|K|million|billion|thousand)?`)
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
		sb.WriteString("\n\n")
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
