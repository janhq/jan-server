package steps

// Theme defines deck theme colors and typography.
type Theme struct {
	PrimaryColor    string `json:"primary_color"`
	AccentColor     string `json:"accent_color"`
	BackgroundColor string `json:"background_color"`
	TextColor       string `json:"text_color"`
	FontFamily      string `json:"font_family"`
}

// SlideImage describes an image used in a slide.
type SlideImage struct {
	Src     string `json:"src"`
	Alt     string `json:"alt"`
	Caption string `json:"caption,omitempty"`
}

// TableData describes a table for a slide.
type TableData struct {
	Title   string   `json:"title,omitempty"`
	Columns []string `json:"columns,omitempty"`
	Rows    [][]any  `json:"rows,omitempty"`
	Notes   string   `json:"notes,omitempty"`
}

// ChartSeries defines a single chart series.
type ChartSeries struct {
	Name   string    `json:"name,omitempty"`
	Values []float64 `json:"values,omitempty"`
}

// ChartPosition specifies a chart's position for PPTX overlay.
type ChartPosition struct {
	X     float64 `json:"x,omitempty"`
	Y     float64 `json:"y,omitempty"`
	W     float64 `json:"w,omitempty"`
	H     float64 `json:"h,omitempty"`
	Units string  `json:"units,omitempty"` // "in" or "px"
}

// ChartData defines chart payload for a slide.
type ChartData struct {
	ID         int           `json:"id,omitempty"`
	Type       string        `json:"type,omitempty"` // bar|line|pie
	Title      string        `json:"title,omitempty"`
	Categories []string      `json:"categories,omitempty"`
	Series     []ChartSeries `json:"series,omitempty"`
	Units      string        `json:"units,omitempty"`
	Notes      string        `json:"notes,omitempty"`
	Position   ChartPosition `json:"position,omitempty"`
	Overlay    *bool         `json:"overlay,omitempty"`
}

// SlidePlan defines a single slide in the plan.
type SlidePlan struct {
	ID       int          `json:"id"`
	Layout   string       `json:"layout,omitempty"`
	Title    string       `json:"title"`
	Subtitle string       `json:"subtitle,omitempty"`
	Bullets  []string     `json:"bullets,omitempty"`
	Images   []SlideImage `json:"images,omitempty"`
	Table    *TableData   `json:"table,omitempty"`
	Chart    *ChartData   `json:"chart,omitempty"`
	Notes    string       `json:"notes,omitempty"`
}

// DeckPlan defines the full deck plan.
type DeckPlan struct {
	Title  string      `json:"title"`
	Theme  Theme       `json:"theme"`
	Slides []SlidePlan `json:"slides"`
}

// DeckTheme defines the deck title and shared theme.
type DeckTheme struct {
	Title string `json:"title"`
	Theme Theme  `json:"theme"`
}

// SlidePlanDraft captures a single slide plan with image guidance.
type SlidePlanDraft struct {
	SlideIndex    int       `json:"slide_index"`
	Slide         SlidePlan `json:"slide"`
	ImageQuery    string    `json:"image_query,omitempty"`
	ImageRequired *bool     `json:"image_required,omitempty"`
}

// TemplateCatalog describes available template packs.
type TemplateCatalog struct {
	Count     int            `json:"count"`
	Templates []TemplateMeta `json:"templates"`
}

// TemplateMeta describes a single template pack.
type TemplateMeta struct {
	ID          int            `json:"id"`
	Dir         string         `json:"dir"`
	Name        string         `json:"name"`
	Tone        string         `json:"tone"`
	Description string         `json:"description"`
	Variants    map[string]int `json:"variants"`
}

// TemplateChoice is a single template selection.
type TemplateChoice struct {
	ID     int    `json:"id"`
	Reason string `json:"reason"`
}

// TemplatePickResponse is the LLM response for template selection.
type TemplatePickResponse struct {
	Choices []TemplateChoice `json:"choices"`
}

// TemplateSelection stores the selected template and options.
type TemplateSelection struct {
	Type        string           `json:"type"`
	TemplateID  int              `json:"template_id"`
	TemplateDir string           `json:"template_dir"`
	Source      string           `json:"source"` // "embedded" or "filesystem"
	FallbackDir string           `json:"fallback_dir,omitempty"`
	Choices     []TemplateChoice `json:"choices,omitempty"`
	Selected    *TemplateChoice  `json:"selected,omitempty"`
}

// ChartsExport is written to charts.json.
type ChartsExport struct {
	Slides []ChartData `json:"slides"`
}

// ImagesExport is written to images.json.
type ImagesExport struct {
	Slides []SlideImages `json:"slides"`
}

// SlideImages groups images by slide.
type SlideImages struct {
	ID     int          `json:"id"`
	Images []SlideImage `json:"images"`
}
