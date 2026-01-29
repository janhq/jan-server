package steps

import (
	"bufio"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

type outlineSlide struct {
	Title   string
	Bullets []string
}

var outlineSlideHeader = regexp.MustCompile(`(?i)^slide\s+(\d+)\s*:\s*(.+)$`)

func applyOutlineFallbacks(plan *DeckPlan, outline string) {
	if plan == nil {
		return
	}
	outline = strings.TrimSpace(outline)
	if outline == "" {
		return
	}
	outlineSlides := parseOutlineSlides(outline)
	if len(outlineSlides) == 0 {
		return
	}

	// Also parse outline blocks for structured content extraction
	outlineBlocks := parseOutlineBlocks(outline)
	blocksByIndex := make(map[int]outlineBlock)
	for _, block := range outlineBlocks {
		blocksByIndex[block.Index] = block
	}

	for i := range plan.Slides {
		slide := &plan.Slides[i]
		outlineSlide, ok := outlineSlides[slide.ID]
		if !ok {
			continue
		}

		if shouldReplaceTitle(slide.Title) && outlineSlide.Title != "" {
			slide.Title = trimToRunesNoEllipsis(outlineSlide.Title, 60)
		}

		// Extract structured content from outline block
		if block, hasBlock := blocksByIndex[slide.ID]; hasBlock {
			content := ExtractSlideContent(block)

			// Apply extracted table if slide doesn't have one
			if slide.Table == nil && content.TableData != nil {
				slide.Table = ConvertExtractedTableToTableData(content.TableData)
				if slide.Table != nil {
					slide.Layout = "table"
				}
			}

			// Apply extracted bullets if slide doesn't have content
			if !hasSlideContent(*slide) && len(content.Bullets) > 0 {
				slide.Bullets = clampBullets(content.Bullets, 6)
			}
		}

		if !hasSlideContent(*slide) && len(outlineSlide.Bullets) > 0 {
			slide.Bullets = clampBullets(outlineSlide.Bullets, 6)
		}

		if len(slide.Bullets) > 0 {
			slide.Bullets = ensureBulletCount(slide.Bullets, 3)
		} else if !hasSlideContent(*slide) {
			slide.Bullets = ensureBulletCount(defaultBulletsForTitle(slide.Title), 3)
		}
	}
}

func parseOutlineSlides(outline string) map[int]outlineSlide {
	parsed := map[int]outlineSlide{}
	scanner := bufio.NewScanner(strings.NewReader(outline))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	current := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if match := outlineSlideHeader.FindStringSubmatch(line); match != nil {
			idx, _ := strconv.Atoi(match[1])
			if idx <= 0 {
				continue
			}
			current = idx
			title := strings.TrimSpace(match[2])
			title = strings.Trim(title, "\"")
			entry := parsed[current]
			entry.Title = title
			parsed[current] = entry
			continue
		}
		if current == 0 {
			continue
		}
		if bullet := parseOutlineBullet(line); bullet != "" {
			entry := parsed[current]
			entry.Bullets = append(entry.Bullets, bullet)
			parsed[current] = entry
		}
	}

	return parsed
}

func parseOutlineBullet(line string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}
	switch {
	case strings.HasPrefix(line, "-"):
		line = strings.TrimSpace(strings.TrimPrefix(line, "-"))
	case strings.HasPrefix(line, "*"):
		line = strings.TrimSpace(strings.TrimPrefix(line, "*"))
	default:
		return ""
	}
	line = strings.Trim(line, "\"")
	return line
}

func shouldReplaceTitle(title string) bool {
	title = strings.TrimSpace(title)
	if title == "" || title == "..." {
		return true
	}
	lower := strings.ToLower(title)
	if strings.HasPrefix(lower, "slide ") {
		return true
	}
	if utf8.RuneCountInString(title) > 80 {
		return true
	}
	if strings.Contains(lower, "chars max") || strings.Contains(lower, "let me check") {
		return true
	}
	return false
}

func ensureBulletCount(bullets []string, min int) []string {
	if min <= 0 {
		return bullets
	}
	out := make([]string, 0, 6)
	for _, bullet := range bullets {
		if len(out) >= 6 {
			break
		}
		bullet = strings.TrimSpace(bullet)
		if bullet == "" {
			continue
		}
		out = append(out, trimToRunesNoEllipsis(bullet, 75))
	}

	fallbacks := []string{
		"Key signals and market drivers in 2025",
		"Evidence-backed trends and momentum shifts",
		"Implications for strategy and execution",
	}
	for _, fallback := range fallbacks {
		if len(out) >= min || len(out) >= 6 {
			break
		}
		out = append(out, fallback)
	}
	return out
}

func defaultBulletsForTitle(title string) []string {
	base := strings.TrimSpace(title)
	if base == "" {
		base = "the topic"
	}
	return []string{
		trimToRunesNoEllipsis("Scope and context for "+base, 75),
		"Key signals and market drivers in 2025",
		"Implications for strategy and execution",
	}
}
