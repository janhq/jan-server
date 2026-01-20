package steps

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

const plannerPrompt = `
You are the Planner for a slide-deck generation system.
Given BRIEF, ASSETS, DATA, and CONSTRAINTS, produce a slide plan only.

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
- Use at most 6 colors total: background, text, mutedText, border, primary, accent. No gradients.
- Indexing: plan.slides[i].index MUST equal i+1 exactly (1..TARGET SLIDE COUNT). No gaps. No duplicates.
- If the BRIEF asks for a table or timeline, you MUST set suggestedLayout to TABLE or TIMELINE.
- For TABLE or TIMELINE layouts, include intended column headers and row topics in keyPoints.
- Do not default to TITLE_AND_BULLETS when a structured layout is requested.

LAYOUT TYPES (use for suggestedLayout):
- TITLE: Title slide with centered title/subtitle
- SECTION_HEADER: Section divider
- TITLE_AND_BULLETS: Title with bullet points
- TITLE_TWO_COLUMNS: Two column layout
- TITLE_IMAGE: Title with image
- FULL_BLEED_IMAGE: Full-bleed image slide
- CHART: Data visualization
- TABLE: Tabular data
- QUOTE: Quote/testimonial
- CLOSING: End slide with call to action
- APPENDIX: Appendix/extra slide
- DASHBOARD_3KPI_2COL: Dashboard layout with 3 KPI slots and 2-column content
- CHART_AND_INSIGHTS: Chart with insight bullets
- TABLE_AND_CALLOUTS: Table with key takeaways callouts

Create a cohesive plan that flows logically from introduction to conclusion.
`

const templatePrompt = `
You are the Template Builder for a slide-deck generation system.
Given BRIEF, PLAN, THEME, and CONSTRAINTS, produce the template only.

OUTPUT FORMAT (STRICT):
- Return ONLY valid JSON that matches the provided schema.
- Do NOT wrap in markdown, code fences, or commentary.
- Do NOT include any extra keys outside the schema.

TEMPLATE REQUIREMENTS:
- version: "1.0"
- metadata: title, language (en), audience, purpose
- theme: canvas, grid, colors (palette + semantic), typography (families, scale, lineHeights), defaults
- masters: at least one master with id, name, background
- layouts: define layouts for each suggestedLayout used in the plan
- layout.id MUST equal the suggestedLayout enum (e.g., "TITLE", "TITLE_AND_BULLETS"). Do NOT prefix with "layout_".
- components: reusable elements (footer, header, etc.)
- export: format (pptx), fileName, includeSpeakerNotes
- Template MUST define header and footer components. Every layout except TITLE/SECTION_HEADER/CLOSING must include them.
- For each layout you create, you MUST define layouts[].slots[] with ids and grid positions.
- Do NOT embed per-slide geometry rules into the plan; geometry is resolved by slots.
`

const dataBankPrompt = `
You are the Data Bank Extractor for a slide-deck generation system.
Given BRIEF, ASSETS, and RESEARCH, extract concrete facts and chart-ready datasets.

OUTPUT FORMAT (STRICT):
- Return ONLY valid JSON that matches the provided schema.
- Do NOT wrap in markdown, code fences, or commentary.
- Do NOT include any extra keys outside the schema.

RULES:
- Facts must be atomic, sourced, and include a date when available.
- Datasets must be ready to use in charts (labels + numeric series).
- Use ONLY data that appears in the provided research context.
- If data is missing, leave the dataset list empty instead of inventing values.
`

func slideWriterPrompt(slideIndex int) string {
	return fmt.Sprintf(`
%s

You are the Slide Writer. Generate slide %d using the provided LOCKED_LAYOUT and plan entry.

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
- layoutId: MUST equal LOCKED_LAYOUT.layoutId from the user prompt
- slide.order MUST equal slideIndex; slide.id MUST be slide_<slideIndex>.
- useComponents MUST be an array (empty if unused).
- If LOCKED_LAYOUT.slotIds is non-empty, you MUST use slotId for each element and OMIT rect.
- slotId MUST be one of the provided LOCKED_LAYOUT.slotIds (or null when using rect).

ELEMENT TYPES:
- text: requires "text": {"content": "...", "style": {...}, "runs": [], "autoFit": "shrink"}
- image: requires "image": {"ref": "asset_id", "fit": "cover"}
- shape: requires "shape": {"kind": "rect|line|arrow|triangle|diamond", "style": {...}}  (use "rect" for containers; no ellipse/roundRect; cornerRadius:null; shadow:null)
- chart: requires "chart": {"chartType": "bar|line|pie", "datasetRef": "dataset_id"}
- table: requires "table": {"columns": [...], "rows": [[...], [...]], "style": {"headerTextStyle": {...}, "cellTextStyle": {...}}}

TABLE ELEMENT (USE WHEN REQUESTED):
- If the plan entry or BRIEF requests a table, you MUST create a type "table" element.
- Include a "table" object with columns and rows; do NOT fake tables with multiline text.
- Use short, meaningful cell text. Keep columns <= 5 when possible.

REQUIRED HEADER ELEMENT:
- Every content slide MUST include a header text element at the very top
- If slotId "header" exists in LOCKED_LAYOUT.slotIds, place the header text element in slotId="header" (preferred)
- Header content: Single short label only (max 80 characters). Do NOT add a subheader/summary line in the header band.
- Header style: fontSize: 11-12pt, color: muted/secondary color, align: left
- This provides context separate from the main title; the main title should start below the header band.
- Every content slide MUST also include a footer element (page number or brand line).
- If component IDs "header" and/or "footer" exist in AVAILABLE_COMPONENT_IDS, include them via useComponents.

TEXT LENGTH LIMITS:
- Header text: max 80 characters (REQUIRED for content slides)
- Slide titles: max 100 characters
- When placing multiple text elements horizontally, ensure >=20pt gap between them
- Use text.autoFit="shrink" for all text elements to prevent overflow

DATASETS FOR CHARTS:
- Prefer referencing an existing dataset from DATA BANK by setting chart.datasetRef to a DATA BANK dataset id.
- If you use a DATA BANK dataset id, keep requires.datasets empty (do NOT duplicate the dataset object).
- Only include datasets in requires.datasets when you introduce a dataset that is NOT already present in DATA BANK.
- DO NOT use string references like ["dataset_xyz"] inside requires.datasets - these will fail during rendering.
- When you include a dataset object in requires.datasets, it MUST be a COMPLETE dataset object with real data:
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
- Ensure labels and values arrays have the same length
- Use meaningful series names that match the chart purpose
- Add proper source attribution in sourceNote

IMPORTANT:
- requires.datasets must be an array of complete objects, NOT strings (when non-empty)
- The chart element's datasetRef must match either:
  - a DATA BANK dataset id, OR
  - a dataset id you include in requires.datasets
- If you add an image element, it MUST reference an asset id that exists in ASSETS AVAILABLE.
- Keep requires.assets empty unless you must introduce an asset not present in ASSETS AVAILABLE.

Create engaging, well-spaced content that fits the slide purpose.
Follow the plan entry's suggestedLayout exactly when provided.
List any required assets/datasets in the "requires" section.
`, sizeGuardPrompt, slideIndex, slideIndex)
}
