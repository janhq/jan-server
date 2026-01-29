package image

// ImageGenerationResponse represents an OpenAI-compatible image generation response.
// @Description OpenAI-compatible image generation response
type ImageGenerationResponse struct {
	// Created is the Unix timestamp of when the response was generated.
	Created int64 `json:"created" example:"1699000000"`

	// Data contains the generated images.
	Data []ImageData `json:"data"`

	// Usage contains token usage information for billing purposes.
	Usage *ImageUsage `json:"usage,omitempty"`
}

// ImageData represents a single generated image.
// @Description Single generated image data
type ImageData struct {
	// URL is the generated image URL.
	// Present when response_format="url".
	URL string `json:"url,omitempty" example:"https://media.jan.ai/images/example.png"`

	// B64JSON is the base64-encoded image data.
	// Present when response_format="b64_json".
	B64JSON string `json:"b64_json,omitempty"`

	// RevisedPrompt is the revised prompt used for generation, if the provider modified it.
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

// ImageUsage contains token usage information for billing purposes.
// Note: Image generation doesn't use traditional tokens; this is an approximation.
// @Description Token usage information for billing
type ImageUsage struct {
	// TotalTokens is the sum of input and output tokens.
	TotalTokens int `json:"total_tokens" example:"1500"`

	// InputTokens is the estimated tokens for the prompt.
	InputTokens int `json:"input_tokens" example:"100"`

	// OutputTokens is the estimated tokens for the generated images.
	OutputTokens int `json:"output_tokens" example:"1400"`

	// InputTokensDetails provides a breakdown of input token types.
	InputTokensDetails *InputTokensDetail `json:"input_tokens_details,omitempty"`
}

// InputTokensDetail provides a breakdown of input token types.
// @Description Breakdown of input token types
type InputTokensDetail struct {
	// TextTokens is the number of tokens from the text prompt.
	TextTokens int `json:"text_tokens" example:"100"`

	// ImageTokens is the number of tokens from input images (for edit/variation operations).
	ImageTokens int `json:"image_tokens" example:"0"`
}
