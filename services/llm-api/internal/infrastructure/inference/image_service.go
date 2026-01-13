package inference

import (
	"context"

	domainmodel "jan-server/services/llm-api/internal/domain/model"
)

// ImageGenerateRequest represents an OpenAI-compatible image generation request.
type ImageGenerateRequest struct {
	// Model specifies the image generation model (e.g., "z-image", "flux-dev", "dall-e-3").
	Model string `json:"model,omitempty"`

	// Prompt is the text description of the desired image.
	Prompt string `json:"prompt"`

	// N is the number of images to generate (1-10, default: 1).
	N int `json:"n,omitempty"`

	// Size specifies the dimensions (e.g., "1024x1024", "1792x1024", "1024x1792").
	Size string `json:"size,omitempty"`

	// Quality determines image quality ("standard" or "hd").
	Quality string `json:"quality,omitempty"`

	// Style influences the visual aesthetic ("vivid" or "natural").
	Style string `json:"style,omitempty"`

	// ResponseFormat determines output format ("url" or "b64_json").
	ResponseFormat string `json:"response_format,omitempty"`

	// User is an optional unique identifier representing the end-user.
	User string `json:"user,omitempty"`
}

// ImageEditRequest represents an OpenAI-compatible image edit request.
type ImageEditRequest struct {
	Model          string  `json:"model,omitempty"`
	Prompt         string  `json:"prompt"`
	N              int     `json:"n,omitempty"`
	Size           string  `json:"size,omitempty"`
	ResponseFormat string  `json:"response_format,omitempty"`
	Strength       float64 `json:"strength,omitempty"`
	Steps          int     `json:"steps,omitempty"`
	Seed           int     `json:"seed,omitempty"`
	CfgScale       float64 `json:"cfg_scale,omitempty"`
	Sampler        string  `json:"sampler,omitempty"`
	Scheduler      string  `json:"scheduler,omitempty"`
	NegativePrompt string  `json:"negative_prompt,omitempty"`
	User           string  `json:"user,omitempty"`

	ImageData        []byte `json:"-"`
	ImageContentType string `json:"-"`
	MaskData         []byte `json:"-"`
	MaskContentType  string `json:"-"`
}

// ImageData represents a single generated image.
type ImageData struct {
	// URL is the generated image URL (when response_format="url").
	URL string `json:"url,omitempty"`

	// B64JSON is the base64-encoded image data (when response_format="b64_json").
	B64JSON string `json:"b64_json,omitempty"`

	// RevisedPrompt is the revised prompt used for generation (if applicable).
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

// ImageGenerateResponse represents an OpenAI-compatible image generation response.
type ImageGenerateResponse struct {
	// Created is the Unix timestamp of when the response was generated.
	Created int64 `json:"created"`

	// Data contains the generated images.
	Data []ImageData `json:"data"`
}

// ImageService defines the interface for image generation operations.
type ImageService interface {
	// Generate creates images based on the provided request and provider.
	Generate(ctx context.Context, provider *domainmodel.Provider, request *ImageGenerateRequest) (*ImageGenerateResponse, error)

	// Edit edits an image based on the provided request and provider.
	Edit(ctx context.Context, provider *domainmodel.Provider, request *ImageEditRequest) (*ImageGenerateResponse, error)

	// SupportsModel checks if the service supports the given model.
	SupportsModel(model string) bool

	// DefaultModel returns the default model for this service.
	DefaultModel() string

	// GetProviderKind returns the provider kind this service handles.
	GetProviderKind() domainmodel.ProviderKind
}
