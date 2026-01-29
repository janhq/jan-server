package inference

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"resty.dev/v3"

	"jan-server/mono/apps/backend/internal/config"
	domainmodel "jan-server/mono/apps/backend/internal/domain/model"
	"jan-server/mono/apps/backend/internal/infrastructure/router"
	"jan-server/mono/apps/backend/internal/utils/crypto"
	httpclients "jan-server/mono/apps/backend/internal/utils/httpclients"
	"jan-server/mono/apps/backend/internal/utils/platformerrors"
)

// ZImageService implements ImageService for z-image (Flux) providers.
type ZImageService struct {
	cfg     *config.Config
	timeout time.Duration
	router  domainmodel.EndpointRouter
}

// NewZImageService creates a new ZImageService instance.
func NewZImageService(cfg *config.Config) *ZImageService {
	timeout := 120 * time.Second // default 2 minutes
	if cfg != nil && cfg.ImageGenerationTimeout > 0 {
		timeout = cfg.ImageGenerationTimeout
	}
	return &ZImageService{
		cfg:     cfg,
		timeout: timeout,
		router:  router.NewRoundRobinRouter(),
	}
}

// zImageRequest is the request format for the z-image provider.
type zImageRequest struct {
	Prompt            string  `json:"prompt"`
	Size              string  `json:"size,omitempty"`
	N                 int     `json:"n,omitempty"`
	NumInferenceSteps int     `json:"num_inference_steps,omitempty"`
	CfgScale          float64 `json:"cfg_scale,omitempty"`
	ResponseFormat    string  `json:"response_format,omitempty"`
	Model             string  `json:"model,omitempty"`
}

// zImageResponse is the response format from the z-image provider.
type zImageResponse struct {
	Created int64              `json:"created"`
	Data    []zImageDataItem   `json:"data"`
	Error   *zImageErrorDetail `json:"error,omitempty"`
}

type zImageDataItem struct {
	B64JSON       string `json:"b64_json,omitempty"`
	URL           string `json:"url,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

type zImageErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
}

// supportedModels lists models this service supports.
var supportedModels = map[string]bool{
	"z-image": true,
	"zimage":  true, // alias
}

// Generate implements ImageService.Generate.
func (s *ZImageService) Generate(ctx context.Context, provider *domainmodel.Provider, request *ImageGenerateRequest) (*ImageGenerateResponse, error) {
	log.Debug().
		Str("provider_id", provider.PublicID).
		Str("provider_name", provider.DisplayName).
		Str("prompt", truncatePrompt(request.Prompt, 50)).
		Str("size", request.Size).
		Int("n", request.N).
		Msg("[ZImageService] Generate called")

	client, selectedURL, err := s.createRestyClient(ctx, provider)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerInfrastructure, err, "failed to create z-image client")
	}

	// Build the provider request
	providerReq := s.buildProviderRequest(request)

	// Call the provider
	resp, err := s.callProvider(ctx, client, selectedURL, providerReq)
	if err != nil {
		return nil, err
	}

	// Convert to our response format
	return s.convertResponse(resp), nil
}

// Edit implements ImageService.Edit.
func (s *ZImageService) Edit(ctx context.Context, provider *domainmodel.Provider, request *ImageEditRequest) (*ImageGenerateResponse, error) {
	log.Debug().
		Str("provider_id", provider.PublicID).
		Str("provider_name", provider.DisplayName).
		Str("prompt", truncatePrompt(request.Prompt, 50)).
		Str("size", request.Size).
		Int("n", request.N).
		Msg("[ZImageService] Edit called")

	client, selectedURL, err := s.createRestyClient(ctx, provider)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerInfrastructure, err, "failed to create z-image client")
	}

	resp, err := s.callEditProvider(ctx, client, provider, selectedURL, request)
	if err != nil {
		return nil, err
	}

	return s.convertResponse(resp), nil
}

// SupportsModel implements ImageService.SupportsModel.
func (s *ZImageService) SupportsModel(model string) bool {
	normalized := strings.ToLower(strings.TrimSpace(model))
	return supportedModels[normalized]
}

// DefaultModel implements ImageService.DefaultModel.
func (s *ZImageService) DefaultModel() string {
	if s.cfg != nil && s.cfg.ImageDefaultModel != "" {
		return s.cfg.ImageDefaultModel
	}
	return "z-image"
}

// GetProviderKind implements ImageService.GetProviderKind.
func (s *ZImageService) GetProviderKind() domainmodel.ProviderKind {
	return domainmodel.ProviderZImage
}

// buildProviderRequest converts our request to the z-image provider format.
func (s *ZImageService) buildProviderRequest(req *ImageGenerateRequest) *zImageRequest {
	// Apply defaults
	n := req.N
	if n <= 0 {
		n = 1
	}
	if s.cfg != nil && n > s.cfg.ImageMaxN {
		n = s.cfg.ImageMaxN
	}

	size := req.Size
	if size == "" && s.cfg != nil {
		size = s.cfg.ImageDefaultSize
	}
	if size == "" {
		size = "1024x1024"
	}

	responseFormat := req.ResponseFormat
	if responseFormat == "" && s.cfg != nil {
		responseFormat = s.cfg.ImageDefaultResponseFormat
	}
	if responseFormat == "" {
		responseFormat = "url"
	}

	model := req.Model
	if model == "" {
		model = s.DefaultModel()
	}

	return &zImageRequest{
		Prompt:         req.Prompt,
		Size:           size,
		N:              n,
		ResponseFormat: responseFormat,
		Model:          model,
	}
}

func (s *ZImageService) callEditProvider(ctx context.Context, client *resty.Client, provider *domainmodel.Provider, baseURL string, req *ImageEditRequest) (*zImageResponse, error) {
	endpoint := resolveImageEditEndpoint(provider, baseURL)

	log.Debug().
		Str("endpoint", endpoint).
		Str("model", req.Model).
		Int("n", req.N).
		Msg("[ZImageService] Calling edit provider")

	form := client.R().
		SetContext(ctx).
		SetHeader("Accept", "application/json")

	imageName := "image.png"
	if req.ImageContentType == "image/jpeg" {
		imageName = "image.jpg"
	}
	form.SetMultipartField("image", imageName, req.ImageContentType, bytes.NewReader(req.ImageData))

	if len(req.MaskData) > 0 {
		maskName := "mask.png"
		if req.MaskContentType == "image/jpeg" {
			maskName = "mask.jpg"
		}
		form.SetMultipartField("mask", maskName, req.MaskContentType, bytes.NewReader(req.MaskData))
	}

	providerResponseFormat := req.ResponseFormat
	if providerResponseFormat == "" || providerResponseFormat == "url" {
		providerResponseFormat = "b64_json"
	}

	fields := map[string]string{
		"prompt": req.Prompt,
	}
	if req.Size != "" {
		fields["size"] = req.Size
	}
	if providerResponseFormat != "" {
		fields["response_format"] = providerResponseFormat
	}
	if req.Model != "" {
		fields["model"] = req.Model
	}
	if req.NegativePrompt != "" {
		fields["negative_prompt"] = req.NegativePrompt
	}
	if req.Sampler != "" {
		fields["sampler"] = req.Sampler
	}
	if req.Scheduler != "" {
		fields["scheduler"] = req.Scheduler
	}
	if req.N > 0 {
		fields["n"] = fmt.Sprintf("%d", req.N)
	}
	if req.Strength > 0 {
		fields["strength"] = fmt.Sprintf("%v", req.Strength)
	}
	if req.Steps > 0 {
		fields["steps"] = fmt.Sprintf("%d", req.Steps)
	}
	if req.Seed != 0 {
		fields["seed"] = fmt.Sprintf("%d", req.Seed)
	}
	if req.CfgScale > 0 {
		fields["cfg_scale"] = fmt.Sprintf("%v", req.CfgScale)
	}

	form.SetFormData(fields)

	resp, err := form.Post(endpoint)
	if err != nil {
		log.Error().Err(err).Str("endpoint", endpoint).Msg("[ZImageService] Edit provider call failed")
		return nil, platformerrors.NewError(ctx, platformerrors.LayerInfrastructure,
			platformerrors.ErrorTypeExternal,
			fmt.Sprintf("image edit provider call failed: %v", err),
			nil, "zimage-provider-error")
	}

	respBytes := resp.Bytes()
	if resp.StatusCode() >= 400 {
		var errResp zImageResponse
		if parseErr := json.Unmarshal(respBytes, &errResp); parseErr == nil && errResp.Error != nil {
			return nil, platformerrors.NewError(ctx, platformerrors.LayerInfrastructure,
				platformerrors.ErrorTypeExternal,
				fmt.Sprintf("image edit provider error: %s", errResp.Error.Message),
				nil, "zimage-provider-error")
		}
		return nil, platformerrors.NewError(ctx, platformerrors.LayerInfrastructure,
			platformerrors.ErrorTypeExternal,
			fmt.Sprintf("image edit provider returned status %d: %s", resp.StatusCode(), string(respBytes)),
			nil, "zimage-provider-http-error")
	}

	var result zImageResponse
	if err := json.Unmarshal(respBytes, &result); err != nil {
		log.Error().Err(err).Str("body", string(respBytes)).Msg("[ZImageService] Failed to parse edit provider response")
		return nil, platformerrors.NewError(ctx, platformerrors.LayerInfrastructure,
			platformerrors.ErrorTypeInternal,
			"failed to parse image edit provider response",
			err, "zimage-parse-error")
	}

	log.Debug().
		Int64("created", result.Created).
		Int("image_count", len(result.Data)).
		Msg("[ZImageService] Edit provider response received")

	return &result, nil
}

func resolveImageEditEndpoint(provider *domainmodel.Provider, baseURL string) string {
	trimmedBase := strings.TrimSuffix(baseURL, "/")
	if provider == nil || provider.Metadata == nil {
		return joinZImageEndpoint(trimmedBase, "/images/edits")
	}

	customPath := normalizeImageEditPath(provider.Metadata[domainmodel.MetadataKeyImageEditPath])
	if customPath == "" {
		return joinZImageEndpoint(trimmedBase, "/images/edits")
	}
	if strings.HasPrefix(customPath, "http://") || strings.HasPrefix(customPath, "https://") {
		return strings.TrimSuffix(customPath, "/")
	}

	customPath = "/" + strings.TrimPrefix(customPath, "/")
	return trimmedBase + customPath
}

func normalizeImageEditPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	lower := strings.ToLower(trimmed)
	if strings.HasSuffix(lower, "/v1/image/edits") && !strings.HasSuffix(lower, "/v1/images/edits") {
		return trimmed[:len(trimmed)-len("/v1/image/edits")] + "/v1/images/edits"
	}
	return trimmed
}

func joinZImageEndpoint(baseURL, path string) string {
	trimmedBase := strings.TrimSuffix(baseURL, "/")
	normalizedPath := "/" + strings.TrimPrefix(path, "/")
	if strings.HasSuffix(trimmedBase, "/v1") {
		return trimmedBase + normalizedPath
	}
	return trimmedBase + "/v1" + normalizedPath
}

// callProvider makes the HTTP call to the z-image provider.
func (s *ZImageService) callProvider(ctx context.Context, client *resty.Client, baseURL string, req *zImageRequest) (*zImageResponse, error) {
	endpoint := joinZImageEndpoint(baseURL, "/images/generations")

	log.Debug().
		Str("endpoint", endpoint).
		Str("model", req.Model).
		Int("n", req.N).
		Msg("[ZImageService] Calling provider")

	resp, err := client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(req).
		Post(endpoint)

	if err != nil {
		log.Error().Err(err).Str("endpoint", endpoint).Msg("[ZImageService] Provider call failed")
		return nil, platformerrors.NewError(ctx, platformerrors.LayerInfrastructure,
			platformerrors.ErrorTypeExternal,
			fmt.Sprintf("image provider call failed: %v", err),
			nil, "zimage-provider-error")
	}

	respBytes := resp.Bytes()

	// Check HTTP status
	if resp.StatusCode() >= 400 {
		var errResp zImageResponse
		if parseErr := json.Unmarshal(respBytes, &errResp); parseErr == nil && errResp.Error != nil {
			return nil, platformerrors.NewError(ctx, platformerrors.LayerInfrastructure,
				platformerrors.ErrorTypeExternal,
				fmt.Sprintf("image provider error: %s", errResp.Error.Message),
				nil, "zimage-provider-error")
		}
		return nil, platformerrors.NewError(ctx, platformerrors.LayerInfrastructure,
			platformerrors.ErrorTypeExternal,
			fmt.Sprintf("image provider returned status %d: %s", resp.StatusCode(), string(respBytes)),
			nil, "zimage-provider-http-error")
	}

	// Parse successful response
	var result zImageResponse
	if err := json.Unmarshal(respBytes, &result); err != nil {
		log.Error().Err(err).Str("body", string(respBytes)).Msg("[ZImageService] Failed to parse provider response")
		return nil, platformerrors.NewError(ctx, platformerrors.LayerInfrastructure,
			platformerrors.ErrorTypeInternal,
			"failed to parse image provider response",
			err, "zimage-parse-error")
	}

	log.Debug().
		Int64("created", result.Created).
		Int("image_count", len(result.Data)).
		Msg("[ZImageService] Provider response received")

	return &result, nil
}

// convertResponse converts the z-image provider response to our format.
func (s *ZImageService) convertResponse(resp *zImageResponse) *ImageGenerateResponse {
	data := make([]ImageData, len(resp.Data))
	for i, item := range resp.Data {
		data[i] = ImageData{
			URL:           item.URL,
			B64JSON:       item.B64JSON,
			RevisedPrompt: item.RevisedPrompt,
		}
	}

	created := resp.Created
	if created == 0 {
		created = time.Now().Unix()
	}

	return &ImageGenerateResponse{
		Created: created,
		Data:    data,
	}
}

// createRestyClient creates an HTTP client configured for the provider.
func (s *ZImageService) createRestyClient(ctx context.Context, provider *domainmodel.Provider) (*resty.Client, string, error) {
	endpoints := provider.GetEndpoints()
	selectedURL, err := s.router.NextEndpoint(provider.PublicID, endpoints)
	if err != nil {
		switch err {
		case domainmodel.ErrNoEndpoints:
			return nil, "", platformerrors.NewError(ctx, platformerrors.LayerInfrastructure,
				platformerrors.ErrorTypeValidation,
				"no endpoints configured for image provider",
				err, "no-endpoints")
		case domainmodel.ErrNoHealthyEndpoints:
			// Fall back to base URL if no healthy endpoints
			selectedURL = provider.BaseURL
			if selectedURL == "" {
				return nil, "", platformerrors.NewError(ctx, platformerrors.LayerInfrastructure,
					platformerrors.ErrorTypeExternal,
					"no healthy endpoints available for image provider",
					err, "no-healthy-endpoints")
			}
		default:
			return nil, "", platformerrors.NewError(ctx, platformerrors.LayerInfrastructure,
				platformerrors.ErrorTypeInternal,
				fmt.Sprintf("endpoint selection failed: %v", err),
				err, "endpoint-selection-error")
		}
	}

	clientName := fmt.Sprintf("zimage-%s", provider.PublicID)
	client := httpclients.NewClient(clientName)
	client.SetTimeout(s.timeout)
	client.SetRetryCount(0) // We handle retries at a higher level

	// Set API key if available
	if provider.EncryptedAPIKey != "" {
		secret := strings.TrimSpace(s.cfg.ModelProviderSecret)
		if secret != "" {
			decrypted, err := crypto.DecryptString(secret, provider.EncryptedAPIKey)
			if err != nil {
				log.Warn().Err(err).Str("provider_id", provider.PublicID).
					Msg("[ZImageService] Failed to decrypt API key")
			} else {
				client.SetHeader("Authorization", fmt.Sprintf("Bearer %s", decrypted))
			}
		}
	}

	// Set request ID for tracing
	if requestID, ok := ctx.Value("request_id").(string); ok && requestID != "" {
		client.SetHeader("X-Request-ID", requestID)
	}

	return client, selectedURL, nil
}

// truncatePrompt truncates a prompt for logging purposes.
func truncatePrompt(prompt string, maxLen int) string {
	if len(prompt) <= maxLen {
		return prompt
	}
	return prompt[:maxLen] + "..."
}

// Ensure ZImageService implements ImageService.
var _ ImageService = (*ZImageService)(nil)
