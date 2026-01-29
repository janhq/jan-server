package image

// ImageGenerationRequest represents an OpenAI-compatible image generation request.
// @Description OpenAI-compatible image generation request
type ImageGenerationRequest struct {
	// Model specifies the image generation model (e.g., "z-image", "flux-dev", "dall-e-3").
	// If omitted, defaults to the configured default model.
	Model string `json:"model,omitempty" example:"z-image"`

	// Prompt is the text description of the desired image. Required.
	Prompt string `json:"prompt" binding:"required" example:"A serene mountain landscape at sunset"`

	// N is the number of images to generate (1-10, default: 1).
	N int `json:"n,omitempty" example:"1"`

	// Size specifies the dimensions of the generated image.
	// Supported sizes: "256x256", "512x512", "1024x1024", "1024x1792", "1792x1024".
	// Default: "1024x1024".
	Size string `json:"size,omitempty" example:"1024x1024"`

	// Quality determines image quality. Valid values: "standard", "hd".
	// Default: "standard".
	Quality string `json:"quality,omitempty" example:"standard"`

	// Style influences the visual aesthetic. Valid values: "vivid", "natural".
	// Default: "natural".
	Style string `json:"style,omitempty" example:"natural"`

	// ResponseFormat determines output format. Valid values: "url", "b64_json".
	// Default: "url".
	ResponseFormat string `json:"response_format,omitempty" example:"url"`

	// User is an optional unique identifier representing the end-user for abuse monitoring.
	User string `json:"user,omitempty" example:"user-123"`

	// Jan-specific extensions:

	// ProviderID optionally overrides the default image provider selection.
	ProviderID string `json:"provider_id,omitempty" example:"prov_abc123"`

	// ConversationID optionally links this generation to a conversation.
	ConversationID string `json:"conversation_id,omitempty" example:"conv_abc123"`

	// Store controls whether to save the result to the conversation.
	// nil/true = store (default), false = don't store.
	Store *bool `json:"store,omitempty" example:"true"`

	// NumInferenceSteps is a provider-specific parameter (z-image/Flux).
	NumInferenceSteps int `json:"num_inference_steps,omitempty" example:"20"`

	// CfgScale is a provider-specific parameter (z-image/Flux) for guidance scale.
	CfgScale float64 `json:"cfg_scale,omitempty" example:"7.5"`
}
