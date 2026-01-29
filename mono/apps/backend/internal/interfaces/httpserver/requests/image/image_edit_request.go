package image

// ImageInput represents an image input for edit requests.
type ImageInput struct {
	// URL is a remote URL to the image.
	URL string `json:"url,omitempty" example:"https://example.com/image.png"`

	// B64JSON is base64-encoded image data (no data URL prefix).
	B64JSON string `json:"b64_json,omitempty"`
}

// ImageEditRequest represents an OpenAI-compatible image edit request.
type ImageEditRequest struct {
	// Model specifies the image edit model. Optional.
	Model string `json:"model,omitempty" example:"qwen-image-edit"`

	// Prompt is the text instruction describing the edit. Required.
	Prompt string `json:"prompt" binding:"required" example:"add golden sunglasses"`

	// Image is the input image to edit. Required.
	Image *ImageInput `json:"image" binding:"required"`

	// Mask is an optional mask for inpainting.
	Mask *ImageInput `json:"mask,omitempty"`

	// N is the number of images to generate (only 1 supported by most providers).
	N int `json:"n,omitempty" example:"1"`

	// Size specifies the output size ("original" or "WIDTHxHEIGHT").
	Size string `json:"size,omitempty" example:"original"`

	// ResponseFormat determines output format ("url" or "b64_json").
	ResponseFormat string `json:"response_format,omitempty" example:"b64_json"`

	// Strength controls edit intensity (0.0-1.0).
	Strength float64 `json:"strength,omitempty" example:"1"`

	// Steps controls sampling steps.
	Steps int `json:"steps,omitempty" example:"4"`

	// Seed sets random seed (-1 for random).
	Seed int `json:"seed,omitempty" example:"-1"`

	// CfgScale is classifier-free guidance scale.
	CfgScale float64 `json:"cfg_scale,omitempty" example:"1"`

	// Sampler selects the sampling algorithm.
	Sampler string `json:"sampler,omitempty" example:"euler"`

	// Scheduler selects the scheduler.
	Scheduler string `json:"scheduler,omitempty" example:"simple"`

	// NegativePrompt describes what to avoid.
	NegativePrompt string `json:"negative_prompt,omitempty" example:" "`

	// ProviderID optionally overrides the default image edit provider.
	ProviderID string `json:"provider_id,omitempty" example:"prov_abc123"`

	// ConversationID optionally links this edit to a conversation.
	ConversationID string `json:"conversation_id,omitempty" example:"conv_abc123"`

	// Store controls whether to save the result to the conversation.
	// nil/true = store (default), false = don't store.
	Store *bool `json:"store,omitempty" example:"true"`
}
