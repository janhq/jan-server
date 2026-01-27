package model

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"jan-server/services/llm-api/internal/domain/query"
)

type ProviderKind string

const (
	ProviderJan         ProviderKind = "jan"
	ProviderOpenAI      ProviderKind = "openai"
	ProviderOpenRouter  ProviderKind = "openrouter"
	ProviderAnthropic   ProviderKind = "anthropic"
	ProviderGoogle      ProviderKind = "google"
	ProviderMistral     ProviderKind = "mistral"
	ProviderGroq        ProviderKind = "groq"
	ProviderCohere      ProviderKind = "cohere"
	ProviderOllama      ProviderKind = "ollama"
	ProviderReplicate   ProviderKind = "replicate"
	ProviderAzureOpenAI ProviderKind = "azure_openai"
	ProviderAWSBedrock  ProviderKind = "aws_bedrock"
	ProviderPerplexity  ProviderKind = "perplexity"
	ProviderTogetherAI  ProviderKind = "togetherai"
	ProviderHuggingFace ProviderKind = "huggingface"
	ProviderVercelAI    ProviderKind = "vercel_ai"
	ProviderDeepInfra   ProviderKind = "deepinfra"
	ProviderCustom      ProviderKind = "custom"  // for any customer-provided API
	ProviderZImage      ProviderKind = "z-image" // Image generation provider
	ProviderOCR         ProviderKind = "ocr"     // Document OCR providers (Docling-compatible)
)

// ProviderCategory distinguishes between LLM and other provider types (e.g., image generation)
type ProviderCategory string

const (
	ProviderCategoryLLM     ProviderCategory = "llm"     // Default: Language model providers
	ProviderCategoryImage   ProviderCategory = "image"   // Image generation providers
	ProviderCategoryOCR     ProviderCategory = "ocr"     // Document OCR providers
	ProviderCategoryDocling ProviderCategory = "docling" // Legacy OCR category (Docling)
)

type Provider struct {
	ID                   uint              `json:"id"`
	PublicID             string            `json:"public_id"`
	DisplayName          string            `json:"display_name"`
	Kind                 ProviderKind      `json:"kind"`
	Category             ProviderCategory  `json:"category"`               // "llm", "image", or "ocr" (defaults to "llm")
	BaseURL              string            `json:"base_url"`               // e.g., https://api.openai.com/v1
	Endpoints            EndpointList      `json:"endpoints,omitempty"`    // Optional: multiple endpoints for round robin
	EncryptedAPIKey      string            `json:"-"`                      // encrypted at rest, decrypted in memory when needed
	APIKeyHint           *string           `json:"api_key_hint,omitempty"` // last4 or source name, not the secret
	IsModerated          bool              `json:"is_moderated"`           // whether provider enforces moderation upstream
	Active               bool              `json:"active"`
	DefaultImageGenerate bool              `json:"default_provider_image_generate"`
	DefaultImageEdit     bool              `json:"default_provider_image_edit"`
	Metadata             map[string]string `json:"metadata,omitempty"` // supports: image_input, file_attachment, description, etc.
	LastSyncedAt         *time.Time        `json:"last_synced_at,omitempty"`
	CreatedAt            time.Time         `json:"created_at"`
	UpdatedAt            time.Time         `json:"updated_at"`
}

// Metadata keys for provider capabilities
const (
	MetadataKeyImageInput       = "image_input"            // JSON string with ImageInputCapability
	MetadataKeyFileAttachment   = "file_attachment"        // JSON string with FileAttachmentCapability
	MetadataKeyDescription      = "description"            // Human-readable description
	MetadataKeyEnvironment      = "environment"            // e.g., "production", "staging", "local"
	MetadataKeyAutoEnableModels = "auto_enable_new_models" // "true" to auto-enable new models
	MetadataKeyToolSupport      = "tool_support"           // "true" if provider supports tools/tool_choice
	MetadataKeyImageEditPath    = "image_edit_path"        // optional path or full URL override for image edits
)

// ImageInputCapability describes how a provider supports image input
type ImageInputCapability struct {
	Supported bool   `json:"supported"`
	URL       bool   `json:"url"`    // Supports image URLs (https://)
	Base64    bool   `json:"base64"` // Supports base64-encoded images
	Schema    string `json:"schema"` // Description of the schema/format
}

// FileAttachmentCapability describes how a provider supports file attachments
type FileAttachmentCapability struct {
	Supported  bool   `json:"supported"`
	URL        bool   `json:"url"`         // Supports file URLs (https://)
	Base64     bool   `json:"base64"`      // Supports base64-encoded files
	FileUpload bool   `json:"file_upload"` // Supports file upload API (file_id references)
	Schema     string `json:"schema"`      // Description of the schema/format
}

// GetImageInputCapability parses and returns the image input capability from metadata
func (p *Provider) GetImageInputCapability() (*ImageInputCapability, error) {
	if p.Metadata == nil {
		return &ImageInputCapability{Supported: false}, nil
	}

	val, ok := p.Metadata[MetadataKeyImageInput]
	if !ok || val == "" {
		return &ImageInputCapability{Supported: false}, nil
	}

	// Handle simple boolean strings for backward compatibility
	if val == "true" || val == "1" {
		return &ImageInputCapability{Supported: true, URL: true, Base64: true}, nil
	}
	if val == "false" || val == "0" {
		return &ImageInputCapability{Supported: false}, nil
	}

	// Parse JSON structure
	var cap ImageInputCapability
	if err := json.Unmarshal([]byte(val), &cap); err != nil {
		// If parsing fails, treat as unsupported
		return &ImageInputCapability{Supported: false}, nil
	}

	return &cap, nil
}

// GetFileAttachmentCapability parses and returns the file attachment capability from metadata
func (p *Provider) GetFileAttachmentCapability() (*FileAttachmentCapability, error) {
	if p.Metadata == nil {
		return &FileAttachmentCapability{Supported: false}, nil
	}

	val, ok := p.Metadata[MetadataKeyFileAttachment]
	if !ok || val == "" {
		return &FileAttachmentCapability{Supported: false}, nil
	}

	// Handle simple boolean strings for backward compatibility
	if val == "true" || val == "1" {
		return &FileAttachmentCapability{Supported: true, URL: true, Base64: true, FileUpload: true}, nil
	}
	if val == "false" || val == "0" {
		return &FileAttachmentCapability{Supported: false}, nil
	}

	// Parse JSON structure
	var cap FileAttachmentCapability
	if err := json.Unmarshal([]byte(val), &cap); err != nil {
		// If parsing fails, treat as unsupported
		return &FileAttachmentCapability{Supported: false}, nil
	}

	return &cap, nil
}

// SupportsImageInput returns true if the provider supports image input
func (p *Provider) SupportsImageInput() bool {
	cap, _ := p.GetImageInputCapability()
	return cap != nil && cap.Supported
}

// SupportsFileAttachment returns true if the provider supports file attachments
func (p *Provider) SupportsFileAttachment() bool {
	cap, _ := p.GetFileAttachmentCapability()
	return cap != nil && cap.Supported
}

// GetDescription returns the provider description from metadata
func (p *Provider) GetDescription() string {
	if p.Metadata == nil {
		return ""
	}
	return p.Metadata[MetadataKeyDescription]
}

// GetEnvironment returns the environment from metadata (e.g., "production", "staging")
func (p *Provider) GetEnvironment() string {
	if p.Metadata == nil {
		return ""
	}
	return p.Metadata[MetadataKeyEnvironment]
}

// SupportsTools returns true if provider metadata indicates tool support.
func (p *Provider) SupportsTools() bool {
	if p == nil || p.Metadata == nil {
		return false
	}
	val := strings.TrimSpace(strings.ToLower(p.Metadata[MetadataKeyToolSupport]))
	switch val {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// GetEndpoints returns configured endpoints with backward-compat fallback to BaseURL.
// Always returns a non-empty list if BaseURL is set.
func (p *Provider) GetEndpoints() EndpointList {
	if len(p.Endpoints) > 0 {
		return p.Endpoints
	}
	if p.BaseURL != "" {
		return EndpointList{{URL: p.BaseURL, Weight: 1, Healthy: true}}
	}
	return nil
}

// SetEndpoints updates endpoints and keeps BaseURL in sync (first endpoint).
func (p *Provider) SetEndpoints(endpoints EndpointList) {
	p.Endpoints = endpoints
	if len(endpoints) > 0 {
		p.BaseURL = endpoints[0].URL
	}
}

// HasMultipleEndpoints reports whether provider has more than one configured endpoint.
func (p *Provider) HasMultipleEndpoints() bool {
	return len(p.Endpoints) > 1
}

// ProviderFilter defines optional conditions for querying providers.
type ProviderFilter struct {
	IDs              *[]uint
	PublicID         *string
	Kind             *ProviderKind
	Category         *ProviderCategory // NEW: filter by category ("llm" or "image")
	Active           *bool
	IsModerated      *bool
	LastSyncedAfter  *time.Time
	LastSyncedBefore *time.Time
	SearchText       *string
}

type AccessibleModels struct {
	Providers      []*Provider      `json:"providers"`
	ProviderModels []*ProviderModel `json:"provider_models"`
}

// ProviderRepository abstracts persistence for provider aggregate roots.
type ProviderRepository interface {
	Create(ctx context.Context, provider *Provider) error
	Update(ctx context.Context, provider *Provider) error
	DeleteByID(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*Provider, error)
	FindByPublicID(ctx context.Context, publicID string) (*Provider, error)
	FindByFilter(ctx context.Context, filter ProviderFilter, p *query.Pagination) ([]*Provider, error)
	Count(ctx context.Context, filter ProviderFilter) (int64, error)
	FindByIDs(ctx context.Context, ids []uint) ([]*Provider, error)
}
