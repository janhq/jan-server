// Package agent defines interfaces and types for agent execution.
package agent

import (
	"encoding/json"
	"strings"
)

// AgentMetadata contains descriptive information about an agent for discovery.
type AgentMetadata struct {
	// Type is the unique identifier for this agent (e.g., "deep_research", "slide_creator")
	Type string `json:"type"`

	// Name is the human-readable name of the agent
	Name string `json:"name"`

	// Description explains what the agent does
	Description string `json:"description"`

	// Keywords are terms that might trigger this agent in user requests
	Keywords []string `json:"keywords"`

	// Capabilities lists what the agent can do
	Capabilities []string `json:"capabilities"`

	// OutputFormats lists supported output formats
	OutputFormats []string `json:"output_formats"`

	// EstimatedDuration is a human-readable estimate of execution time
	EstimatedDuration string `json:"estimated_duration"`

	// UseWhen describes when to use this agent (for LLM selection)
	UseWhen string `json:"use_when"`

	// Enabled indicates if the agent is currently available
	Enabled bool `json:"enabled"`
}

// AgentInputSchema describes the expected input for an agent.
type AgentInputSchema struct {
	Type       string                    `json:"type"`
	Properties map[string]SchemaProperty `json:"properties"`
	Required   []string                  `json:"required"`
}

// SchemaProperty describes a single property in the schema.
type SchemaProperty struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Default     any      `json:"default,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Minimum     *int     `json:"minimum,omitempty"`
	Maximum     *int     `json:"maximum,omitempty"`
}

// AgentOutputSchema describes the expected output from an agent.
type AgentOutputSchema struct {
	Type       string                    `json:"type"`
	Properties map[string]SchemaProperty `json:"properties"`
}

// AgentExample provides an example of agent usage.
type AgentExample struct {
	Input       json.RawMessage `json:"input"`
	Description string          `json:"description"`
}

// AgentDetail contains full information about an agent including schemas.
type AgentDetail struct {
	AgentMetadata
	InputSchema  *AgentInputSchema  `json:"inputSchema,omitempty"`
	OutputSchema *AgentOutputSchema `json:"outputSchema,omitempty"`
	Examples     []AgentExample     `json:"examples,omitempty"`
}

// AgentCapability is a condensed format for LLM agent selection.
type AgentCapability struct {
	Type     string   `json:"type"`
	UseWhen  string   `json:"use_when"`
	Keywords []string `json:"keywords"`
}

// SandboxCapabilities defines what sandbox features are available.
// This is static configuration, not dynamic tool detection.
type SandboxCapabilities struct {
	// Enabled is true if any sandbox provider is configured (AIO or E2B)
	Enabled bool `json:"enabled"`

	// Provider is "aio", "e2b", or "" if disabled
	Provider string `json:"provider"`

	// HasBrowser is true if sandbox has browser automation (E2B desktop)
	HasBrowser bool `json:"has_browser"`

	// HasCodeExecution is true if sandbox can execute code
	HasCodeExecution bool `json:"has_code_execution"`
}

// GetSandboxCapabilities returns sandbox capabilities based on provider config.
// This is deterministic - based on configuration, not runtime tool availability.
func GetSandboxCapabilities(sandboxProvider string) SandboxCapabilities {
	provider := strings.ToLower(strings.TrimSpace(sandboxProvider))

	switch provider {
	case "e2b":
		return SandboxCapabilities{
			Enabled:          true,
			Provider:         "e2b",
			HasBrowser:       true, // E2B desktop has browser
			HasCodeExecution: true,
		}
	case "aio":
		return SandboxCapabilities{
			Enabled:          true,
			Provider:         "aio",
			HasBrowser:       false, // AIO doesn't have MCP browser control yet
			HasCodeExecution: true,
		}
	default:
		return SandboxCapabilities{
			Enabled:          false,
			Provider:         "",
			HasBrowser:       false,
			HasCodeExecution: false,
		}
	}
}

// AgentCapabilityRequirements defines what an agent needs to function.
type AgentCapabilityRequirements struct {
	// RequiresSandbox - agent needs sandbox for core functionality
	RequiresSandbox bool `json:"requires_sandbox"`

	// RequiresBrowser - agent needs browser automation
	RequiresBrowser bool `json:"requires_browser"`

	// RequiresCodeExecution - agent needs code execution
	RequiresCodeExecution bool `json:"requires_code_execution"`
}

// GetAgentCapabilityRequirements returns capability requirements for an agent.
func GetAgentCapabilityRequirements(agentType string) AgentCapabilityRequirements {
	switch agentType {
	case "deep_research":
		return AgentCapabilityRequirements{
			RequiresSandbox:       false, // Can work with just search + scrape
			RequiresBrowser:       false,
			RequiresCodeExecution: false, // Code execution is optional enhancement
		}
	case "slide_creator":
		return AgentCapabilityRequirements{
			RequiresSandbox:       true, // Needs sandbox for PPTX export (runs Node.js + pptxgenjs)
			RequiresBrowser:       false,
			RequiresCodeExecution: true, // Runs code to generate PPTX
		}
	case "doc_generator":
		return AgentCapabilityRequirements{
			RequiresSandbox:       true,  // Needs sandbox for DOCX generation
			RequiresBrowser:       false,
			RequiresCodeExecution: true,
		}
	case "pdf_generator":
		return AgentCapabilityRequirements{
			RequiresSandbox:       true,  // Needs sandbox for PDF rendering
			RequiresBrowser:       false,
			RequiresCodeExecution: true,
		}
	case "spreadsheet_generator":
		return AgentCapabilityRequirements{
			RequiresSandbox:       true,  // Needs sandbox for XLSX generation
			RequiresBrowser:       false,
			RequiresCodeExecution: true,
		}
	default:
		return AgentCapabilityRequirements{}
	}
}

// CheckAgentEnabled checks if an agent can be enabled based on sandbox capabilities.
// This is deterministic - based on static config, not dynamic tool list.
func CheckAgentEnabled(agentType string, sandbox SandboxCapabilities) bool {
	reqs := GetAgentCapabilityRequirements(agentType)

	if reqs.RequiresSandbox && !sandbox.Enabled {
		return false
	}
	if reqs.RequiresBrowser && !sandbox.HasBrowser {
		return false
	}
	if reqs.RequiresCodeExecution && !sandbox.HasCodeExecution {
		return false
	}

	return true
}

// GetEnabledAgentTypes returns agent types that are enabled given sandbox config.
func GetEnabledAgentTypes(sandboxProvider string) []string {
	sandbox := GetSandboxCapabilities(sandboxProvider)
	allAgents := []string{
		"deep_research",
		"slide_creator",
		"doc_generator",
		"pdf_generator",
		"spreadsheet_generator",
	}

	enabled := make([]string, 0, len(allAgents))
	for _, agentType := range allAgents {
		if CheckAgentEnabled(agentType, sandbox) {
			enabled = append(enabled, agentType)
		}
	}
	return enabled
}

// DeepResearchMetadata returns metadata for the deep research agent.
func DeepResearchMetadata() AgentMetadata {
	return AgentMetadata{
		Type:        "deep_research",
		Name:        "Deep Research Agent",
		Description: "Performs comprehensive web research with source verification, cross-referencing, and structured reports. Ideal for in-depth analysis and investigation tasks.",
		Keywords: []string{
			"research", "analyze", "investigate", "study", "explore",
			"find information", "learn about", "deep dive", "comprehensive",
		},
		Capabilities: []string{
			"web_search", "scrape", "cross_reference", "citation",
			"report_generation", "source_verification", "data_synthesis",
		},
		OutputFormats:     []string{"markdown", "pdf", "json"},
		EstimatedDuration: "2-10 minutes",
		UseWhen:           "User wants to research, investigate, analyze, or learn about a topic in depth",
		Enabled:           true,
	}
}

// SlideCreatorMetadata returns metadata for the slide creator agent.
func SlideCreatorMetadata() AgentMetadata {
	return AgentMetadata{
		Type:        "slide_creator",
		Name:        "Slide Creator Agent",
		Description: "Builds HTML-based slide decks and exports them to editable PPTX with research-backed content.",
		Keywords: []string{
			"slides", "presentation", "powerpoint", "deck", "pitch",
			"pptx", "html slides", "editable slides",
		},
		Capabilities: []string{
			"research", "outline", "html_slide_generation", "template_selection",
			"pptx_export", "theme_customization",
		},
		OutputFormats:     []string{"pptx", "html"},
		EstimatedDuration: "3-15 minutes",
		UseWhen:           "User wants an editable PPTX generated from HTML slide layouts with template control",
		Enabled:           true,
	}
}

// DocGeneratorMetadata returns metadata for the document generator agent.
func DocGeneratorMetadata() AgentMetadata {
	return AgentMetadata{
		Type:        "doc_generator",
		Name:        "Document Generator Agent",
		Description: "Creates structured Word documents for reports, briefs, and structured write-ups.",
		Keywords: []string{
			"document", "docx", "report", "brief", "proposal",
			"word document", "write-up",
		},
		Capabilities: []string{
			"content_generation", "structured_sections", "tables", "export",
		},
		OutputFormats:     []string{"docx", "pdf"},
		EstimatedDuration: "1-5 minutes",
		UseWhen:           "User wants a structured document such as a report or brief",
		Enabled:           true,
	}
}

// PDFGeneratorMetadata returns metadata for the PDF generator agent.
func PDFGeneratorMetadata() AgentMetadata {
	return AgentMetadata{
		Type:        "pdf_generator",
		Name:        "PDF Generator Agent",
		Description: "Generates PDF documents with headings, tables, and structured sections.",
		Keywords: []string{
			"pdf", "export pdf", "report pdf", "document pdf",
		},
		Capabilities: []string{
			"content_generation", "tables", "pdf_export",
		},
		OutputFormats:     []string{"pdf"},
		EstimatedDuration: "1-5 minutes",
		UseWhen:           "User wants a PDF output",
		Enabled:           true,
	}
}

// SpreadsheetGeneratorMetadata returns metadata for the spreadsheet generator agent.
func SpreadsheetGeneratorMetadata() AgentMetadata {
	return AgentMetadata{
		Type:        "spreadsheet_generator",
		Name:        "Spreadsheet Generator Agent",
		Description: "Builds spreadsheets with multiple sheets, tables, and formulas.",
		Keywords: []string{
			"spreadsheet", "xlsx", "excel", "table", "worksheet",
		},
		Capabilities: []string{
			"table_generation", "formulas", "export",
		},
		OutputFormats:     []string{"xlsx"},
		EstimatedDuration: "1-5 minutes",
		UseWhen:           "User wants an Excel spreadsheet or structured tabular data",
		Enabled:           true,
	}
}

// DeepResearchInputSchema returns the input schema for deep research.
func DeepResearchInputSchema() *AgentInputSchema {
	return &AgentInputSchema{
		Type: "object",
		Properties: map[string]SchemaProperty{
			"prompt": {
				Type:        "string",
				Description: "The research question or topic to investigate",
			},
			"research_depth": {
				Type:        "string",
				Description: "How thorough the research should be",
				Enum:        []string{"minimal", "standard", "deep"},
				Default:     "standard",
			},
			"format": {
				Type:        "string",
				Description: "Output format for the research report",
				Enum:        []string{"markdown", "pdf", "json"},
				Default:     "markdown",
			},
			"max_sources": {
				Type:        "integer",
				Description: "Maximum number of sources to consult",
				Default:     10,
			},
			"include_citations": {
				Type:        "boolean",
				Description: "Whether to include source citations",
				Default:     true,
			},
		},
		Required: []string{"prompt"},
	}
}

// SlideCreatorInputSchema returns the input schema for slide creation.
func SlideCreatorInputSchema() *AgentInputSchema {
	min3 := 3
	max50 := 50
	return &AgentInputSchema{
		Type: "object",
		Properties: map[string]SchemaProperty{
			"prompt": {
				Type:        "string",
				Description: "Description of the presentation to create",
			},
			"num_slides": {
				Type:        "integer",
				Description: "Target number of slides",
				Default:     10,
				Minimum:     &min3,
				Maximum:     &max50,
			},
			"theme": {
				Type:        "string",
				Description: "Visual theme for the presentation",
				Enum:        []string{"modern", "corporate", "minimal", "creative", "dark"},
				Default:     "modern",
			},
			"format": {
				Type:        "string",
				Description: "Output format for the presentation",
				Enum:        []string{"pptx"},
				Default:     "pptx",
			},
			"research_depth": {
				Type:        "string",
				Description: "How much web research to perform",
				Enum:        []string{"minimal", "standard", "deep"},
				Default:     "standard",
			},
			"options_count": {
				Type:        "integer",
				Description: "Number of variations to generate for user selection",
				Default:     1,
			},
			"template_dir": {
				Type:        "string",
				Description: "Override template directory (absolute path for custom templates)",
			},
			"template_catalog": {
				Type:        "string",
				Description: "Template catalog JSON path (defaults to embedded catalog)",
			},
			"template_id": {
				Type:        "integer",
				Description: "Force a specific template ID",
			},
			"tone": {
				Type:        "string",
				Description: "Preferred tone for template selection",
			},
			"body_mode": {
				Type:        "string",
				Description: "Slide body rendering mode",
				Enum:        []string{"template", "llm"},
				Default:     "template",
			},
			"debug": {
				Type:        "boolean",
				Description: "Write debug outputs into the HTML bundle",
				Default:     false,
			},
		},
		Required: []string{"prompt"},
	}
}

// DocGeneratorInputSchema returns the input schema for document generation.
func DocGeneratorInputSchema() *AgentInputSchema {
	return &AgentInputSchema{
		Type: "object",
		Properties: map[string]SchemaProperty{
			"prompt": {
				Type:        "string",
				Description: "Description of the document to create",
			},
			"template": {
				Type:        "string",
				Description: "Document template style",
				Enum:        []string{"professional", "academic", "minimal"},
				Default:     "professional",
			},
			"format": {
				Type:        "string",
				Description: "Output format",
				Enum:        []string{"docx", "pdf"},
				Default:     "docx",
			},
		},
		Required: []string{"prompt"},
	}
}

// PDFGeneratorInputSchema returns the input schema for PDF generation.
func PDFGeneratorInputSchema() *AgentInputSchema {
	return &AgentInputSchema{
		Type: "object",
		Properties: map[string]SchemaProperty{
			"prompt": {
				Type:        "string",
				Description: "Description of the PDF to create",
			},
			"page_size": {
				Type:        "string",
				Description: "Page size for the PDF",
				Enum:        []string{"A4", "Letter", "Legal"},
				Default:     "A4",
			},
			"orientation": {
				Type:        "string",
				Description: "Page orientation",
				Enum:        []string{"portrait", "landscape"},
				Default:     "portrait",
			},
		},
		Required: []string{"prompt"},
	}
}

// SpreadsheetGeneratorInputSchema returns the input schema for spreadsheet generation.
func SpreadsheetGeneratorInputSchema() *AgentInputSchema {
	return &AgentInputSchema{
		Type: "object",
		Properties: map[string]SchemaProperty{
			"prompt": {
				Type:        "string",
				Description: "Description of the spreadsheet to create",
			},
			"include_charts": {
				Type:        "boolean",
				Description: "Whether to include charts where applicable",
				Default:     false,
			},
		},
		Required: []string{"prompt"},
	}
}

// DeepResearchOutputSchema returns the output schema for deep research.
func DeepResearchOutputSchema() *AgentOutputSchema {
	return &AgentOutputSchema{
		Type: "object",
		Properties: map[string]SchemaProperty{
			"artifact_id": {
				Type:        "string",
				Description: "ID of the generated research artifact",
			},
			"summary": {
				Type:        "string",
				Description: "Executive summary of the research findings",
			},
			"sources_count": {
				Type:        "integer",
				Description: "Number of sources consulted",
			},
			"download_url": {
				Type:        "string",
				Description: "URL to download the full report",
			},
		},
	}
}

// SlideCreatorOutputSchema returns the output schema for slide creation.
func SlideCreatorOutputSchema() *AgentOutputSchema {
	return &AgentOutputSchema{
		Type: "object",
		Properties: map[string]SchemaProperty{
			"pptx_artifact_id": {
				Type:        "string",
				Description: "ID of the generated PPTX artifact",
			},
			"html_artifact_id": {
				Type:        "string",
				Description: "ID of the generated HTML bundle artifact",
			},
			"images_artifact_id": {
				Type:        "string",
				Description: "ID of the slide images artifact (if available)",
			},
			"slides_count": {
				Type:        "integer",
				Description: "Number of slides generated",
			},
			"download_url": {
				Type:        "string",
				Description: "URL to download the presentation",
			},
			"preview_url": {
				Type:        "string",
				Description: "URL to preview the HTML slides",
			},
		},
	}
}

// DocGeneratorOutputSchema returns the output schema for document generation.
func DocGeneratorOutputSchema() *AgentOutputSchema {
	return &AgentOutputSchema{
		Type: "object",
		Properties: map[string]SchemaProperty{
			"artifact_id": {
				Type:        "string",
				Description: "ID of the generated document artifact",
			},
			"download_url": {
				Type:        "string",
				Description: "URL to download the document",
			},
		},
	}
}

// PDFGeneratorOutputSchema returns the output schema for PDF generation.
func PDFGeneratorOutputSchema() *AgentOutputSchema {
	return &AgentOutputSchema{
		Type: "object",
		Properties: map[string]SchemaProperty{
			"artifact_id": {
				Type:        "string",
				Description: "ID of the generated PDF artifact",
			},
			"download_url": {
				Type:        "string",
				Description: "URL to download the PDF",
			},
		},
	}
}

// SpreadsheetGeneratorOutputSchema returns the output schema for spreadsheet generation.
func SpreadsheetGeneratorOutputSchema() *AgentOutputSchema {
	return &AgentOutputSchema{
		Type: "object",
		Properties: map[string]SchemaProperty{
			"artifact_id": {
				Type:        "string",
				Description: "ID of the generated spreadsheet artifact",
			},
			"download_url": {
				Type:        "string",
				Description: "URL to download the spreadsheet",
			},
		},
	}
}

// DeepResearchExamples returns usage examples for deep research.
func DeepResearchExamples() []AgentExample {
	return []AgentExample{
		{
			Input:       json.RawMessage(`{"prompt": "Research the impact of AI on healthcare in 2026", "research_depth": "deep"}`),
			Description: "Comprehensive research on AI in healthcare with deep analysis",
		},
		{
			Input:       json.RawMessage(`{"prompt": "Analyze market trends for electric vehicles", "max_sources": 15}`),
			Description: "Market analysis with extended source coverage",
		},
	}
}

// SlideCreatorExamples returns usage examples for slide creation.
func SlideCreatorExamples() []AgentExample {
	return []AgentExample{
		{
			Input:       json.RawMessage(`{"prompt": "Create a pitch deck for a SaaS startup", "num_slides": 12, "theme": "modern", "body_mode": "template"}`),
			Description: "Creates a 12-slide PPTX using HTML templates for layout",
		},
		{
			Input:       json.RawMessage(`{"prompt": "Make slides about AI safety for executives", "template_id": 12, "tone": "corporate"}`),
			Description: "Uses a specific template pack and tone to guide selection",
		},
	}
}

// DocGeneratorExamples returns usage examples for document generation.
func DocGeneratorExamples() []AgentExample {
	return []AgentExample{
		{
			Input:       json.RawMessage(`{"prompt": "Write a project proposal for a new analytics platform", "format": "docx"}`),
			Description: "Creates a structured Word document proposal",
		},
	}
}

// PDFGeneratorExamples returns usage examples for PDF generation.
func PDFGeneratorExamples() []AgentExample {
	return []AgentExample{
		{
			Input:       json.RawMessage(`{"prompt": "Generate a quarterly report summary", "page_size": "Letter"}`),
			Description: "Creates a PDF summary report",
		},
	}
}

// SpreadsheetGeneratorExamples returns usage examples for spreadsheet generation.
func SpreadsheetGeneratorExamples() []AgentExample {
	return []AgentExample{
		{
			Input:       json.RawMessage(`{"prompt": "Create a budget tracker spreadsheet", "include_charts": false}`),
			Description: "Creates a budget tracker in XLSX format",
		},
	}
}

// GetAgentDetail returns the full detail for an agent type.
func GetAgentDetail(agentType string) *AgentDetail {
	switch agentType {
	case "deep_research":
		return &AgentDetail{
			AgentMetadata: DeepResearchMetadata(),
			InputSchema:   DeepResearchInputSchema(),
			OutputSchema:  DeepResearchOutputSchema(),
			Examples:      DeepResearchExamples(),
		}
	case "slide_creator":
		return &AgentDetail{
			AgentMetadata: SlideCreatorMetadata(),
			InputSchema:   SlideCreatorInputSchema(),
			OutputSchema:  SlideCreatorOutputSchema(),
			Examples:      SlideCreatorExamples(),
		}
	case "doc_generator":
		return &AgentDetail{
			AgentMetadata: DocGeneratorMetadata(),
			InputSchema:   DocGeneratorInputSchema(),
			OutputSchema:  DocGeneratorOutputSchema(),
			Examples:      DocGeneratorExamples(),
		}
	case "pdf_generator":
		return &AgentDetail{
			AgentMetadata: PDFGeneratorMetadata(),
			InputSchema:   PDFGeneratorInputSchema(),
			OutputSchema:  PDFGeneratorOutputSchema(),
			Examples:      PDFGeneratorExamples(),
		}
	case "spreadsheet_generator":
		return &AgentDetail{
			AgentMetadata: SpreadsheetGeneratorMetadata(),
			InputSchema:   SpreadsheetGeneratorInputSchema(),
			OutputSchema:  SpreadsheetGeneratorOutputSchema(),
			Examples:      SpreadsheetGeneratorExamples(),
		}
	default:
		return nil
	}
}

// GetAllAgentMetadata returns metadata for all registered agents.
func GetAllAgentMetadata() []AgentMetadata {
	return []AgentMetadata{
		DeepResearchMetadata(),
		SlideCreatorMetadata(),
		DocGeneratorMetadata(),
		PDFGeneratorMetadata(),
		SpreadsheetGeneratorMetadata(),
	}
}

// GetAgentCapabilities returns condensed capabilities for LLM selection.
func GetAgentCapabilities() []AgentCapability {
	metadata := GetAllAgentMetadata()
	capabilities := make([]AgentCapability, 0, len(metadata))

	for _, m := range metadata {
		if m.Enabled {
			capabilities = append(capabilities, AgentCapability{
				Type:     m.Type,
				UseWhen:  m.UseWhen,
				Keywords: m.Keywords,
			})
		}
	}

	return capabilities
}
