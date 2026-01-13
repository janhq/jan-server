package imagehandler

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"

	"jan-server/services/llm-api/internal/config"
	domainmodel "jan-server/services/llm-api/internal/domain/model"
	"jan-server/services/llm-api/internal/infrastructure/inference"
	"jan-server/services/llm-api/internal/infrastructure/mediaclient"
	"jan-server/services/llm-api/internal/infrastructure/observability"
	imagerequest "jan-server/services/llm-api/internal/interfaces/httpserver/requests/image"
	imageresponse "jan-server/services/llm-api/internal/interfaces/httpserver/responses/image"
	"jan-server/services/llm-api/internal/utils/platformerrors"
)

const (
	maxProviderRetries = 3
)

// ImageHandler handles image generation requests.
type ImageHandler struct {
	cfg             *config.Config
	providerService *domainmodel.ProviderService
	imageService    inference.ImageService
	mediaClient     *mediaclient.Client
	// NOTE: conversationService removed - conversation items are now managed by chat handler
}

// NewImageHandler creates a new ImageHandler instance.
func NewImageHandler(
	cfg *config.Config,
	providerService *domainmodel.ProviderService,
	imageService inference.ImageService,
	mediaClient *mediaclient.Client,
	_ interface{}, // conversationService - kept for backwards compatibility but unused
) *ImageHandler {
	return &ImageHandler{
		cfg:             cfg,
		providerService: providerService,
		imageService:    imageService,
		mediaClient:     mediaClient,
	}
}

// ImageGenerationResult wraps the response with additional context.
type ImageGenerationResult struct {
	Response *imageresponse.ImageGenerationResponse
}

// GenerateImage handles image generation requests.
func (h *ImageHandler) GenerateImage(
	ctx context.Context,
	reqCtx *gin.Context,
	userID uint,
	request imagerequest.ImageGenerationRequest,
) (*ImageGenerationResult, error) {
	// Start OpenTelemetry span
	ctx, span := observability.StartSpan(ctx, "llm-api", "ImageHandler.GenerateImage")
	defer span.End()

	startTime := time.Now()

	// Add span attributes
	observability.AddSpanAttributes(ctx,
		attribute.Int64("user_id", int64(userID)),
		attribute.String("model", request.Model),
		attribute.String("size", request.Size),
		attribute.Int("n", request.N),
	)

	log.Info().
		Uint("user_id", userID).
		Str("model", request.Model).
		Str("size", request.Size).
		Int("n", request.N).
		Str("prompt", truncatePrompt(request.Prompt, 100)).
		Msg("[ImageHandler] Processing image generation request")

	// Check if image generation is enabled
	if h.cfg != nil && !h.cfg.ImageGenerationEnabled {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain,
			platformerrors.ErrorTypeValidation,
			"image generation is not enabled",
			nil, "image-generation-disabled")
	}

	// Get active image provider
	provider, err := h.selectImageProvider(ctx, &request)
	if err != nil {
		log.Warn().Err(err).Msg("[ImageHandler] No active image provider found")
		return nil, err
	}

	log.Debug().
		Str("provider_id", provider.PublicID).
		Str("provider_name", provider.DisplayName).
		Str("provider_kind", string(provider.Kind)).
		Msg("[ImageHandler] Using image provider")

	// Build the service request
	serviceRequest := h.buildServiceRequest(&request)

	// Call the image service with retries
	serviceResponse, err := h.callWithRetry(ctx, provider, serviceRequest)
	if err != nil {
		observability.RecordError(ctx, err)
		return nil, err
	}

	// Convert to HTTP response (upload to media-api if needed)
	authHeader := reqCtx.GetHeader("Authorization")
	response := h.convertToHTTPResponse(ctx, serviceResponse, request.ResponseFormat, authHeader)

	// Calculate and add usage
	response.Usage = h.calculateUsage(len(request.Prompt), serviceResponse)

	// NOTE: Conversation storage is now handled by chat handler when tool calls complete
	// The storeInConversation logic has been removed to avoid duplicate items
	// See chat_handler.go for conversation item management

	duration := time.Since(startTime)
	log.Info().
		Uint("user_id", userID).
		Int("image_count", len(response.Data)).
		Dur("duration", duration).
		Msg("[ImageHandler] Image generation completed")

	observability.AddSpanAttributes(ctx,
		attribute.Int("image_count", len(response.Data)),
		attribute.Int64("duration_ms", duration.Milliseconds()),
	)

	return &ImageGenerationResult{
		Response: response,
	}, nil
}

// EditImage handles image edit requests.
func (h *ImageHandler) EditImage(
	ctx context.Context,
	reqCtx *gin.Context,
	userID uint,
	request imagerequest.ImageEditRequest,
) (*ImageGenerationResult, error) {
	ctx, span := observability.StartSpan(ctx, "llm-api", "ImageHandler.EditImage")
	defer span.End()

	startTime := time.Now()
	reqID := reqCtx.GetHeader("X-Request-ID")

	observability.AddSpanAttributes(ctx,
		attribute.Int64("user_id", int64(userID)),
		attribute.String("model", request.Model),
		attribute.String("size", request.Size),
		attribute.Int("n", request.N),
	)

	log.Info().
		Uint("user_id", userID).
		Str("model", request.Model).
		Str("size", request.Size).
		Int("n", request.N).
		Str("prompt", truncatePrompt(request.Prompt, 100)).
		Str("request_id", reqID).
		Msg("[ImageHandler] Processing image edit request")

	if h.cfg != nil && !h.cfg.ImageGenerationEnabled {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain,
			platformerrors.ErrorTypeValidation,
			"image generation is not enabled",
			nil, "image-generation-disabled")
	}

	provider, err := h.selectImageEditProvider(ctx, &request)
	if err != nil {
		log.Warn().Err(err).Msg("[ImageHandler] No default image edit provider found")
		return nil, err
	}

	log.Debug().
		Str("provider_id", provider.PublicID).
		Str("provider_name", provider.DisplayName).
		Str("provider_kind", string(provider.Kind)).
		Str("provider_base_url", provider.BaseURL).
		Str("request_id", reqID).
		Msg("[ImageHandler] Using image edit provider")

	if request.Image != nil {
		log.Debug().
			Str("image_url", request.Image.URL).
			Bool("image_has_b64", strings.TrimSpace(request.Image.B64JSON) != "").
			Str("request_id", reqID).
			Msg("[ImageHandler] Image edit input")
	}
	if request.Mask != nil {
		log.Debug().
			Str("mask_url", request.Mask.URL).
			Bool("mask_has_b64", strings.TrimSpace(request.Mask.B64JSON) != "").
			Str("request_id", reqID).
			Msg("[ImageHandler] Image edit mask input")
	}

	serviceRequest, err := h.buildEditServiceRequest(ctx, reqCtx, &request)
	if err != nil {
		return nil, err
	}

	log.Debug().
		Str("model", serviceRequest.Model).
		Str("size", serviceRequest.Size).
		Str("response_format", serviceRequest.ResponseFormat).
		Int("n", serviceRequest.N).
		Int("steps", serviceRequest.Steps).
		Float64("strength", serviceRequest.Strength).
		Int("seed", serviceRequest.Seed).
		Float64("cfg_scale", serviceRequest.CfgScale).
		Str("sampler", serviceRequest.Sampler).
		Str("scheduler", serviceRequest.Scheduler).
		Bool("has_image", len(serviceRequest.ImageData) > 0).
		Bool("has_mask", len(serviceRequest.MaskData) > 0).
		Str("request_id", reqID).
		Msg("[ImageHandler] Built image edit service request")

	serviceResponse, err := h.imageService.Edit(ctx, provider, serviceRequest)
	if err != nil {
		observability.RecordError(ctx, err)
		return nil, err
	}

	authHeader := reqCtx.GetHeader("Authorization")
	response := h.convertToHTTPResponse(ctx, serviceResponse, request.ResponseFormat, authHeader)
	response.Usage = h.calculateUsage(len(request.Prompt), serviceResponse)

	// NOTE: Conversation storage is now handled by chat handler when tool calls complete
	// The storeInConversationEdit logic has been removed to avoid duplicate items
	// See chat_handler.go for conversation item management

	duration := time.Since(startTime)
	log.Info().
		Uint("user_id", userID).
		Int("image_count", len(response.Data)).
		Dur("duration", duration).
		Msg("[ImageHandler] Image edit completed")

	observability.AddSpanAttributes(ctx,
		attribute.Int("image_count", len(response.Data)),
		attribute.Int64("duration_ms", duration.Milliseconds()),
	)

	return &ImageGenerationResult{
		Response: response,
	}, nil
}

func (h *ImageHandler) selectImageProvider(
	ctx context.Context,
	request *imagerequest.ImageGenerationRequest,
) (*domainmodel.Provider, error) {
	if request == nil {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain,
			platformerrors.ErrorTypeValidation,
			"image generation request is required", nil, "image-request-missing")
	}

	if strings.TrimSpace(request.ProviderID) != "" {
		provider, err := h.providerService.FindByPublicID(ctx, strings.TrimSpace(request.ProviderID))
		if err != nil {
			return nil, err
		}
		if provider == nil {
			return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain,
				platformerrors.ErrorTypeNotFound,
				"image provider not found", nil, "image-provider-not-found")
		}
		if !provider.Active || provider.Category != domainmodel.ProviderCategoryImage {
			return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain,
				platformerrors.ErrorTypeValidation,
				"image provider is not active or not an image provider", nil, "image-provider-invalid")
		}
		return provider, nil
	}

	return h.providerService.FindDefaultImageGenerateProvider(ctx)
}

// buildServiceRequest converts the HTTP request to a service request.
func (h *ImageHandler) buildServiceRequest(req *imagerequest.ImageGenerationRequest) *inference.ImageGenerateRequest {
	return &inference.ImageGenerateRequest{
		Model:          req.Model,
		Prompt:         req.Prompt,
		N:              req.N,
		Size:           req.Size,
		Quality:        req.Quality,
		Style:          req.Style,
		ResponseFormat: req.ResponseFormat,
		User:           req.User,
	}
}

func (h *ImageHandler) buildEditServiceRequest(
	ctx context.Context,
	reqCtx *gin.Context,
	req *imagerequest.ImageEditRequest,
) (*inference.ImageEditRequest, error) {
	if req == nil || req.Image == nil {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain,
			platformerrors.ErrorTypeValidation,
			"image is required", nil, "image-edit-validation-001")
	}

	imageBytes, imageContentType, err := h.resolveImageInput(ctx, reqCtx, req.Image)
	if err != nil {
		return nil, err
	}

	var maskBytes []byte
	var maskContentType string
	if req.Mask != nil {
		maskBytes, maskContentType, err = h.resolveImageInput(ctx, reqCtx, req.Mask)
		if err != nil {
			return nil, err
		}
	}

	responseFormat := req.ResponseFormat
	if responseFormat == "" {
		responseFormat = "b64_json"
	} else if responseFormat != "url" && responseFormat != "b64_json" {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain,
			platformerrors.ErrorTypeValidation,
			"response_format must be url or b64_json",
			nil, "image-edit-validation-004")
	}

	size := req.Size
	if size == "" {
		size = "original"
	}

	return &inference.ImageEditRequest{
		Model:            req.Model,
		Prompt:           req.Prompt,
		N:                req.N,
		Size:             size,
		ResponseFormat:   responseFormat,
		Strength:         req.Strength,
		Steps:            req.Steps,
		Seed:             req.Seed,
		CfgScale:         req.CfgScale,
		Sampler:          req.Sampler,
		Scheduler:        req.Scheduler,
		NegativePrompt:   req.NegativePrompt,
		ImageData:        imageBytes,
		ImageContentType: imageContentType,
		MaskData:         maskBytes,
		MaskContentType:  maskContentType,
	}, nil
}

// callWithRetry calls the image service with retry logic.
func (h *ImageHandler) callWithRetry(
	ctx context.Context,
	provider *domainmodel.Provider,
	request *inference.ImageGenerateRequest,
) (*inference.ImageGenerateResponse, error) {
	var lastErr error

	for attempt := 1; attempt <= maxProviderRetries; attempt++ {
		log.Debug().
			Int("attempt", attempt).
			Int("max_attempts", maxProviderRetries).
			Str("provider_id", provider.PublicID).
			Msg("[ImageHandler] Calling image provider")

		resp, err := h.imageService.Generate(ctx, provider, request)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		// Don't retry on client errors (4xx equivalent)
		if isClientError(err) {
			log.Warn().
				Err(err).
				Int("attempt", attempt).
				Msg("[ImageHandler] Client error, not retrying")
			return nil, err
		}

		// Log retry
		if attempt < maxProviderRetries {
			log.Warn().
				Err(err).
				Int("attempt", attempt).
				Int("max_attempts", maxProviderRetries).
				Msg("[ImageHandler] Provider call failed, retrying")
		}
	}

	log.Error().
		Err(lastErr).
		Int("attempts", maxProviderRetries).
		Str("provider_id", provider.PublicID).
		Msg("[ImageHandler] All retry attempts failed")

	return nil, platformerrors.NewError(ctx, platformerrors.LayerInfrastructure,
		platformerrors.ErrorTypeExternal,
		fmt.Sprintf("image provider failed after %d attempts: %v", maxProviderRetries, lastErr),
		lastErr, "image-provider-retry-exhausted")
}

func (h *ImageHandler) selectImageEditProvider(
	ctx context.Context,
	request *imagerequest.ImageEditRequest,
) (*domainmodel.Provider, error) {
	if request == nil {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain,
			platformerrors.ErrorTypeValidation,
			"image edit request is required", nil, "image-edit-request-missing")
	}

	if strings.TrimSpace(request.ProviderID) != "" {
		provider, err := h.providerService.FindByPublicID(ctx, strings.TrimSpace(request.ProviderID))
		if err != nil {
			return nil, err
		}
		if provider == nil {
			return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain,
				platformerrors.ErrorTypeNotFound,
				"image provider not found", nil, "image-provider-not-found")
		}
		if !provider.Active {
			return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain,
				platformerrors.ErrorTypeValidation,
				"image provider is not active", nil, "image-provider-inactive")
		}
		if supportsImageEdit(provider) {
			return provider, nil
		}
		log.Warn().
			Str("provider_id", provider.PublicID).
			Str("provider_name", provider.DisplayName).
			Str("provider_kind", string(provider.Kind)).
			Msg("[ImageHandler] Provider does not support image edits, falling back to default edit provider")
		return h.providerService.FindDefaultImageEditProvider(ctx)
	}

	return h.providerService.FindDefaultImageEditProvider(ctx)
}

func supportsImageEdit(provider *domainmodel.Provider) bool {
	if provider == nil {
		return false
	}
	if provider.Metadata != nil {
		if strings.TrimSpace(provider.Metadata[domainmodel.MetadataKeyImageEditPath]) != "" {
			return true
		}
	}
	if provider.Kind == domainmodel.ProviderZImage {
		return false
	}
	return provider.Category == domainmodel.ProviderCategoryImage
}

func (h *ImageHandler) resolveImageInput(
	ctx context.Context,
	reqCtx *gin.Context,
	input *imagerequest.ImageInput,
) ([]byte, string, error) {
	if input == nil {
		return nil, "", platformerrors.NewError(ctx, platformerrors.LayerDomain,
			platformerrors.ErrorTypeValidation,
			"image input is required", nil, "image-edit-validation-002")
	}

	if strings.TrimSpace(input.B64JSON) != "" {
		log.Debug().
			Msg("[ImageHandler] Resolving image input from base64")
		return decodeBase64Image(input.B64JSON)
	}

	if strings.TrimSpace(input.URL) != "" {
		log.Debug().
			Str("url", input.URL).
			Msg("[ImageHandler] Resolving image input from URL")
		return downloadImage(ctx, input.URL)
	}

	return nil, "", platformerrors.NewError(ctx, platformerrors.LayerDomain,
		platformerrors.ErrorTypeValidation,
		"image input must include url or b64_json", nil, "image-edit-validation-003")
}

func decodeBase64Image(raw string) ([]byte, string, error) {
	contentType := "image/png"
	data := raw
	if strings.HasPrefix(raw, "data:") {
		parts := strings.SplitN(raw, ",", 2)
		if len(parts) != 2 {
			return nil, "", fmt.Errorf("invalid data URL")
		}
		meta := parts[0]
		data = parts[1]
		if strings.Contains(meta, "image/") {
			metaParts := strings.SplitN(meta, ";", 2)
			if len(metaParts) > 0 && strings.HasPrefix(metaParts[0], "data:") {
				contentType = strings.TrimPrefix(metaParts[0], "data:")
			}
		}
	}

	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, "", fmt.Errorf("decode base64 image: %w", err)
	}
	return decoded, contentType, nil
}

func downloadImage(ctx context.Context, url string) ([]byte, string, error) {
	if strings.HasPrefix(url, "data:") {
		log.Debug().
			Msg("[ImageHandler] Downloading image from data URL")
		return decodeBase64Image(url)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("build image download request: %w", err)
	}

	log.Debug().
		Str("url", url).
		Msg("[ImageHandler] Downloading image from URL")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		log.Debug().
			Str("url", url).
			Int("status", resp.StatusCode).
			Msg("[ImageHandler] Image download failed")
		return nil, "", fmt.Errorf("download image failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/png"
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read image response: %w", err)
	}
	log.Debug().
		Str("url", url).
		Str("content_type", contentType).
		Int("bytes", len(body)).
		Msg("[ImageHandler] Image download completed")
	return body, contentType, nil
}

// convertToHTTPResponse converts the service response to an HTTP response.
// Uploads base64 images to media-api and returns direct URLs.
func (h *ImageHandler) convertToHTTPResponse(
	ctx context.Context,
	resp *inference.ImageGenerateResponse,
	responseFormat string,
	authHeader string,
) *imageresponse.ImageGenerationResponse {
	data := make([]imageresponse.ImageData, len(resp.Data))
	format := strings.TrimSpace(strings.ToLower(responseFormat))
	for i, item := range resp.Data {
		imgData := imageresponse.ImageData{
			RevisedPrompt: item.RevisedPrompt,
		}

		// If we have base64 data, upload to media-api and return direct URL
		if item.B64JSON != "" && format != "b64_json" && h.mediaClient != nil {
			mediaResp, err := h.mediaClient.UploadBase64Image(ctx, item.B64JSON, "image/png", authHeader)
			if err != nil {
				log.Warn().Err(err).Msg("[ImageHandler] Failed to upload to media-api, falling back to base64")
				imgData.B64JSON = item.B64JSON
			} else {
				imgData.URL = mediaResp.URL
			}
		} else if strings.TrimSpace(item.URL) != "" {
			// If provider returned a URL directly, use it
			imgData.URL = strings.TrimSpace(item.URL)
		} else if item.B64JSON != "" {
			// No media client, return base64 directly
			imgData.B64JSON = item.B64JSON
		}

		data[i] = imgData
	}

	return &imageresponse.ImageGenerationResponse{
		Created: resp.Created,
		Data:    data,
	}
}

// NOTE: storeInConversation, storeInConversationEdit, and getLastConversationItem
// have been removed. Conversation items are now managed by chat_handler.go
// when MCP tool calls complete. This avoids duplicate item creation.

// calculateUsage provides an estimated token usage for billing purposes.
// Image generation doesn't have true tokens - this maps bytes/params to pseudo-tokens.
func (h *ImageHandler) calculateUsage(promptLength int, resp *inference.ImageGenerateResponse) *imageresponse.ImageUsage {
	// Simple estimation: ~4 chars per token
	inputTokens := promptLength / 4
	if inputTokens < 1 {
		inputTokens = 1
	}

	// Estimate output tokens based on image count and size
	// A 1024x1024 image is roughly 1-4 MB, which we map to ~1000-4000 tokens
	outputTokens := len(resp.Data) * 1000 // Base estimate per image

	return &imageresponse.ImageUsage{
		TotalTokens:  inputTokens + outputTokens,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		InputTokensDetails: &imageresponse.InputTokensDetail{
			TextTokens:  inputTokens,
			ImageTokens: 0, // No input images for generation
		},
	}
}

// isClientError checks if an error is a client-side error (4xx equivalent).
func isClientError(err error) bool {
	if err == nil {
		return false
	}

	// Check for platform errors with validation type
	if platformErr, ok := err.(*platformerrors.PlatformError); ok {
		switch platformErr.Type {
		case platformerrors.ErrorTypeValidation,
			platformerrors.ErrorTypeNotFound,
			platformerrors.ErrorTypeUnauthorized,
			platformerrors.ErrorTypeForbidden,
			platformerrors.ErrorTypeConflict:
			return true
		}
	}

	return false
}

// truncatePrompt truncates a prompt for logging purposes.
func truncatePrompt(prompt string, maxLen int) string {
	if len(prompt) <= maxLen {
		return prompt
	}
	return prompt[:maxLen] + "..."
}
