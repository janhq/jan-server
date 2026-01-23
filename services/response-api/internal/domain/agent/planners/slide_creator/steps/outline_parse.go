package steps

import (
	"regexp"
	"strings"
)

type outlineBlock struct {
	Index int
	Title string
	Lines []string
}

var outlineDataMarker = regexp.MustCompile(`(?i)\b(table|chart|dataset)\b`)
var outlineImageMarker = regexp.MustCompile(`(?i)\b(image|visual|photo|illustration|diagram|icon|background)\b`)
var outlineURLMatcher = regexp.MustCompile(`https?://[^\s\])>]+`)

func parseOutlineBlocks(outline string) []outlineBlock {
	outline = strings.TrimSpace(outline)
	if outline == "" {
		return nil
	}
	lines := strings.Split(outline, "\n")
	blocks := []outlineBlock{}
	var current *outlineBlock
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if match := outlineSlideHeader.FindStringSubmatch(trimmed); match != nil {
			block := outlineBlock{
				Index: parseOutlineIndex(match[1]),
				Title: strings.TrimSpace(strings.Trim(match[2], "\"")),
				Lines: []string{},
			}
			if block.Index > 0 {
				blocks = append(blocks, block)
				current = &blocks[len(blocks)-1]
			} else {
				current = nil
			}
			continue
		}
		if current != nil {
			current.Lines = append(current.Lines, strings.TrimRight(line, " \t"))
		}
	}
	return blocks
}

func outlineBlockForSlide(outline string, slideIndex int) (outlineBlock, bool) {
	if slideIndex <= 0 {
		return outlineBlock{}, false
	}
	for _, block := range parseOutlineBlocks(outline) {
		if block.Index == slideIndex {
			return block, true
		}
	}
	return outlineBlock{}, false
}

func outlineBlockText(block outlineBlock) string {
	if len(block.Lines) == 0 {
		return ""
	}
	return strings.TrimSpace(strings.Join(block.Lines, "\n"))
}

func outlineNeedsDataBank(outline string) bool {
	outline = strings.TrimSpace(outline)
	if outline == "" {
		return false
	}
	return outlineDataMarker.MatchString(outline) || strings.Contains(outline, "|")
}

func outlineBlockNeedsDataBank(block outlineBlock) bool {
	if block.Title != "" && outlineDataMarker.MatchString(block.Title) {
		return true
	}
	for _, line := range block.Lines {
		if outlineDataMarker.MatchString(line) || strings.Contains(line, "|") {
			return true
		}
	}
	return false
}

func outlineBlockNeedsImage(block outlineBlock) bool {
	if block.Title != "" && outlineImageMarker.MatchString(block.Title) {
		return true
	}
	for _, line := range block.Lines {
		if outlineImageMarker.MatchString(line) {
			return true
		}
	}
	return false
}

func extractOutlineURLs(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	matches := outlineURLMatcher.FindAllString(text, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		trimmed := strings.TrimRight(match, ".,;:)")
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func fallbackDeckTitle(outline string, brief string) string {
	if outlineBlock, ok := outlineBlockForSlide(outline, 1); ok && outlineBlock.Title != "" {
		return trimToRunesNoEllipsis(outlineBlock.Title, 60)
	}
	return trimToRunesNoEllipsis(strings.TrimSpace(brief), 60)
}

func parseOutlineIndex(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	n := 0
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}
