package schemas

// PlanAndTemplate is the combined output from planner + template builder.
type PlanAndTemplate struct {
	Plan     SlidePlan     `json:"plan"`
	Template SlideTemplate `json:"template"`
}

// SlidePlan contains the presentation plan.
type SlidePlan struct {
	DeckTitle             string      `json:"deckTitle"`
	Audience              string      `json:"audience"`
	Tone                  string      `json:"tone"`
	Purpose               string      `json:"purpose"`
	RecommendedSlideCount int         `json:"recommendedSlideCount"`
	Slides                []PlanEntry `json:"slides"`
}

// PlanEntry describes one planned slide.
type PlanEntry struct {
	Index           int      `json:"index"`
	Title           string   `json:"title"`
	Purpose         string   `json:"purpose"`
	KeyPoints       []string `json:"keyPoints"`
	SuggestedLayout string   `json:"suggestedLayout"`
	VisualIdeas     []string `json:"visualIdeas"`
}

// SlideTemplate contains the deck template structure.
type SlideTemplate struct {
	Version    string `json:"version"`
	Metadata   any    `json:"metadata"`
	Theme      any    `json:"theme"`
	Masters    any    `json:"masters"`
	Layouts    any    `json:"layouts"`
	Components any    `json:"components"`
	Export     any    `json:"export"`
}

// SlideGenResult is the output from individual slide generation.
type SlideGenResult struct {
	Slide    any               `json:"slide"`
	Requires SlideRequirements `json:"requires"`
}

// SlideRequirements lists assets and datasets needed.
type SlideRequirements struct {
	Assets   []any `json:"assets"`
	Datasets []any `json:"datasets"`
}

// IssuesReport contains semantic validation issues.
type IssuesReport struct {
	Issues []Issue `json:"issues"`
}

// Issue describes a single validation issue.
type Issue struct {
	Severity     string `json:"severity"`
	Message      string `json:"message"`
	SlideIndex   int    `json:"slideIndex"`
	SuggestedFix string `json:"suggestedFix"`
}
