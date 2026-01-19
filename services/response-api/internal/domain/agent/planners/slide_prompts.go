package planners

import "fmt"

const sizeGuardPrompt = `
COORDINATE SYSTEM:
- PowerPoint WIDE_16x9 slide size is 13.333in x 7.5in
- We author rects in points (pt). 72pt = 1in.
- Therefore the slide coordinate plane is 960pt x 540pt.
- IMPORTANT: Do NOT use 1920x1080 numeric coordinates when unit="pt".
- Safe margins: 36pt on all sides (0.5in)
- Usable area for ALL text and charts: x 36-924, y 36-504
- For every element (including shapes/images): x>=0, y>=0, (x+w)<=960, (y+h)<=540
- For text-heavy blocks: keep them inside the safe margins: x>=36, y>=36, (x+w)<=924, (y+h)<=504
- Full-bleed backgrounds/images MAY use x=0,y=0,w=960,h=540, but keep text inside safe margins
STYLE GUARDRAILS:
- Use only square-corner rectangles: shape.kind MUST be "rect" for rectangular containers (no roundRect/roundedRect/pill/ellipse)
- Do not use circles/ovals as decorative containers
- Avoid drop shadows; use flat fills with thin strokes
- For shapes: set shape.style.cornerRadius = null and shape.style.shadow = null (or omit shadow effects)
- No overlaps: ensure text boxes are tall enough for wrapping; keep >=16pt vertical padding between stacked blocks
- For bullets: use newline \n between items (do NOT use the '|' separator)
TEXT LENGTH CONSTRAINTS:
- Header text at very top of slide (y: 36-50): REQUIRED for content slides, max 80 characters to prevent overlap with date
- Position header text on LEFT side (x: 36-700) to avoid overlapping with date/metadata on RIGHT (x: 850+)
- Slide titles (y: 60-120): max 60 characters for single line, max 100 for wrapped titles
- Ensure adequate spacing (>=20pt horizontal gap) between adjacent text elements

`

const plannerAndTemplatePrompt = `
You are the Planner and Template Builder for a slide-deck generation system.
Given BRIEF, ASSETS, DATA, and CONSTRAINTS, produce BOTH a slide plan AND the template.

OUTPUT FORMAT (STRICT):
- Return ONLY valid JSON that matches the provided schema.
- Do NOT wrap in markdown, code fences, or commentary.
- Do NOT include any extra keys outside the schema.

DESIGN GUIDELINES:
- Canvas: WIDE_16x9, unit: pt. Coordinate plane: 960pt x 540pt.
- Safe margins: 36pt on all sides (0.5in)
- Usable area for ALL text and charts: x: 36-924, y: 36-504
- All colors must be hex format: #RRGGBB
- All rect coordinates use: x, y, w, h (not width/height)
- No rounded rectangles (avoid roundRect/roundedRect/pill); use clean square corners
- Avoid drop shadows; use flat shapes
- No circles/ovals as decorative containers
- Use only square-corner rectangles (shape.kind:"rect") for boxes; keep cornerRadius:null and shadow:null

LAYOUT TYPES (use for suggestedLayout):
- TITLE: Title slide with centered title/subtitle
- SECTION_HEADER: Section divider
- TITLE_AND_BULLETS: Title with bullet points
- TITLE_TWO_COLUMNS: Two column layout
- TITLE_IMAGE: Title with image
- CHART: Data visualization
- TABLE: Tabular data
- QUOTE: Quote/testimonial
- CLOSING: End slide with call to action

TEMPLATE REQUIREMENTS:
- version: "1.0"
- metadata: title, language (en), audience, purpose
- theme: canvas, grid, colors (palette + semantic), typography (families, scale, lineHeights), defaults
- masters: at least one master with id, name, background
- layouts: define layouts for each suggestedLayout used in the plan
- components: reusable elements (footer, header, etc.)
- export: format (pptx), fileName, includeSpeakerNotes

Create a cohesive plan that flows logically from introduction to conclusion.
`

func slideWriterPrompt(slideIndex int) string {
	return fmt.Sprintf(`
%s

You are the Slide Writer. Generate slide %d using the provided template and plan entry.

OUTPUT FORMAT (STRICT):
- Return ONLY valid JSON that matches the provided schema.
- Do NOT wrap in markdown, code fences, or commentary.
- Do NOT include any extra keys outside the schema.

DESIGN RULES:
- Canvas: WIDE_16x9, unit: pt. Coordinate plane: 960pt x 540pt.
- Safe margins: 36pt on all sides (usable: x:36-924, y:36-504)
- All rect use x, y, w, h coordinates
- Colors must be hex: #RRGGBB
- No rounded rectangles (avoid roundRect/roundedRect/pill); use kind:"rect"
- Avoid drop shadows; keep shapes flat
- For any rectangular box: use kind:"rect" and set shape.style.cornerRadius=null and shape.style.shadow=null
- For bullet lists: set text.style.bullet.enabled=true and separate items with newline \n (no | separators)
- For titles or any text that might wrap: set text.autoFit="shrink" so it won't overlap other blocks
- id: unique string like "slide_%d_title"
- layoutId: must match a layout.id from the template

ELEMENT TYPES:
- text: requires "text": {"content": "...", "style": {...}, "runs": [], "autoFit": "shrink"}
- image: requires "image": {"ref": "asset_id", "fit": "cover"}
- shape: requires "shape": {"kind": "rect|line|arrow|triangle|diamond", "style": {...}}  (use "rect" for containers; no ellipse/roundRect; cornerRadius:null; shadow:null)
- chart: requires "chart": {"chartType": "bar|line|pie", "datasetRef": "dataset_id"}

REQUIRED HEADER ELEMENT:
- Every content slide MUST include a header text element at the very top
- Header position: rect: {x: 36-50, y: 36-46, w: 650-700, h: 18-24}
- Header content: Single short label only (max 80 characters). Do NOT add a subheader/summary line in the header band.
- Header style: fontSize: 11-12pt, color: muted/secondary color, align: left
- This provides context separate from the main title; the main title should start below the header band.

TEXT LENGTH LIMITS:
- Header text: max 80 characters (REQUIRED for content slides)
- Slide titles: max 100 characters
- When placing multiple text elements horizontally, ensure >=20pt gap between them
- Use text.autoFit="shrink" for all text elements to prevent overflow

DATASET GENERATION (CRITICAL FOR CHARTS):
- If your slide includes a chart element, you MUST provide complete dataset definitions in requires.datasets
- DO NOT use string references like ["dataset_xyz"] - these will fail during rendering
- Instead, provide COMPLETE dataset objects with actual data from the research:
  {
    "id": "dataset_xyz",
    "kind": "series",
    "data": {
      "labels": ["Q2 2025", "Q3 2025"],  // Use ACTUAL labels from the BRIEF research data
      "series": [{"name": "GDP Growth (%%)", "values": [3.8, 4.3]}]  // Use ACTUAL numbers from research
    },
    "sourceNote": "Source: BEA 2025 GDP estimates"
  }
- Extract real numbers from the BRIEF section (research results, search output, etc.)
- For GDP data: look for percentages, growth rates, specific quarterly or annual values
- Ensure labels and values arrays have the same length
- Use meaningful series names that match the chart purpose
- Add proper source attribution in sourceNote

IMPORTANT:
- requires.datasets must be an array of complete objects, NOT strings
- Each dataset must have: id (string), kind ("series"), data (object with labels and series arrays)
- The chart element's datasetRef must match a dataset id you provide

Create engaging, well-spaced content that fits the slide purpose.
List any required assets/datasets in the "requires" section.
`, sizeGuardPrompt, slideIndex, slideIndex)
}

const deckValidatorPrompt = `
You are the Deck Validator. Review the presentation for semantic and content issues.

CHECK FOR:
- Narrative flow: Does the story progress logically?
- Duplicate content: Are there repeated points across slides?
- Visual balance: Is there variety in layouts?
- Completeness: Are there missing key points or gaps?
- Consistency: Is terminology and messaging consistent?
- Engagement: Will this hold the audience's attention?

DO NOT CHECK:
- JSON structure (handled by schema validation)
- Coordinate/margin violations (handled by schema)

For each issue found, provide:
- severity: "warn" for suggestions, "error" for must-fix
- slideIndex: 1-based index (0 for deck-wide issues)
- message: Clear description of the issue
- suggestedFix: Actionable recommendation
`
