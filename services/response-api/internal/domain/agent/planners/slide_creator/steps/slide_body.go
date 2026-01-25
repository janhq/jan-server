package steps

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"jan-server/services/response-api/internal/domain/agent"
)

func (e *SlideCreatorExecutor) generateSlideBody(ctx context.Context, input agent.ExecutionInput, plan DeckPlan, slide SlidePlan) (string, error) {
	planJSON, _ := json.Marshal(plan.Theme)
	slideJSON, _ := json.Marshal(slide)
	prompt := fmt.Sprintf(`You are generating the BODY HTML for a single slide.

Output: ONLY an HTML fragment for the body area (no <html>, <head>, <body>, <style>, <script>).

NON-NEGOTIABLE LAYOUT RULES:
- Root element must be <section> with:
  display:grid; grid-template-columns:repeat(12, 1fr); gap:32px; height:100%%;
- The body area must be fit-safe:
  - Every major container must have overflow:hidden.
  - Use max-height where needed so nothing can push outside the slide.
  - Clamp text (titles, subtitles, bullets) using -webkit-line-clamp.
- No overlap: do NOT use absolute positioning for text blocks.

COLOR + CONTRAST RULES (must follow):
- Use theme.text_color for all primary text at FULL opacity.
- "Muted" text must still be readable:
  - minimum opacity 0.85, never lighter than that.
- If you place text on top of an image, you MUST add a readable backdrop:
  - Add a rounded rectangle scrim behind text:
    - dark background: rgba(0,0,0,0.55) + white text
    - light background: rgba(255,255,255,0.75) + dark text
  - Include padding: 18-24px; border-radius: 18-22px.

TYPOGRAPHY RULES:
- Title block: max 2 lines.
- Subtitle: max 2 lines.
- Bullets: max 6 bullets; each bullet max 2 lines.
- Use line-height 1.15-1.35 for headings, 1.35-1.55 for body text.
- Prefer font sizes that fit 16:9 (no tiny text).

CONTENT RULES:
- Use image URLs from slide data and include alt text.
- Keep layout clean and readable at 16:9. Assume header/footer already exist.

Theme JSON:
%s

Slide JSON:
%s
`, string(planJSON), string(slideJSON))

	model := getModelFromContext(input)
	body, err := e.generateWithSystemPrompt(ctx, "", prompt, model, 0.3)
	if err != nil {
		return "", fmt.Errorf("slide body request failed: %w", err)
	}
	body = strings.TrimSpace(body)
	if err := validateBodyHTML(body); err == nil {
		return body, nil
	}

	fixPrompt := fmt.Sprintf(`Fix the following HTML fragment to satisfy ALL constraints.

Constraints:
- Output ONLY an HTML fragment (no <html>, <head>, <body>, <style>, <script>).
- Root element must be <section> with:
  display:grid; grid-template-columns:repeat(12, 1fr); gap:32px; height:100%%;
- NO overlap and NO overflow:
  - no absolute-positioned text blocks
  - overflow:hidden on major containers
  - clamp text with -webkit-line-clamp
- Contrast:
  - primary text uses theme.text_color at opacity 1.0
  - muted text opacity >= 0.85
  - any text on image must have a scrim backdrop with padding + rounded corners

BAD_HTML:
%s
`, body)

	fixed, err := e.generateWithSystemPrompt(ctx, "", fixPrompt, model, 0.0)
	if err != nil {
		return "", fmt.Errorf("invalid HTML from slide body (and repair failed): %w", err)
	}
	fixed = strings.TrimSpace(fixed)
	if err := validateBodyHTML(fixed); err != nil {
		return "", fmt.Errorf("invalid HTML after repair: %v", err)
	}
	return fixed, nil
}

func validateBodyHTML(body string) error {
	b := strings.ToLower(body)
	for _, bad := range []string{"<html", "<head", "<body", "<style", "<script"} {
		if strings.Contains(b, bad) {
			return fmt.Errorf("contains forbidden tag %s", bad)
		}
	}
	if !strings.Contains(strings.ToLower(strings.TrimSpace(body)), "<section") {
		return fmt.Errorf("missing root <section>")
	}
	return nil
}
