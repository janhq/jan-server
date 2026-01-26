package inference

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"resty.dev/v3"

	"jan-server/services/llm-api/internal/config"
	domainmodel "jan-server/services/llm-api/internal/domain/model"
	"jan-server/services/llm-api/internal/infrastructure/router"
	"jan-server/services/llm-api/internal/utils/crypto"
	httpclients "jan-server/services/llm-api/internal/utils/httpclients"
	"jan-server/services/llm-api/internal/utils/platformerrors"
)

// DocumentOCRService handles document OCR processing via external providers.
type DocumentOCRService struct {
	cfg     *config.Config
	timeout time.Duration
	router  domainmodel.EndpointRouter
}

// NewDocumentOCRService creates a new DocumentOCRService instance.
func NewDocumentOCRService(cfg *config.Config) *DocumentOCRService {
	timeout := 120 * time.Second // default 2 minutes
	if cfg != nil && cfg.DocumentOCRTimeout > 0 {
		timeout = cfg.DocumentOCRTimeout
	}
	return &DocumentOCRService{
		cfg:     cfg,
		timeout: timeout,
		router:  router.NewRoundRobinRouter(),
	}
}

// DocumentOCRRequest is the request format for the OCR provider.
type DocumentOCRRequest struct {
	Model    string `json:"model,omitempty"`
	FileURL  string `json:"file_url,omitempty"`  // Presigned URL from media-api
	FileData string `json:"file_data,omitempty"` // Base64 encoded file data (alternative)
	MimeType string `json:"mime_type"`
	Filename string `json:"filename,omitempty"`
}

// DocumentOCRResponse is the response format from the OCR provider.
type DocumentOCRResponse struct {
	Text      string `json:"text"`
	PageCount int    `json:"page_count,omitempty"`
	WordCount int    `json:"word_count,omitempty"`
	Model     string `json:"model,omitempty"`
	Error     *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code,omitempty"`
	} `json:"error,omitempty"`
}

// Scan performs OCR on a document using the configured provider.
func (s *DocumentOCRService) Scan(ctx context.Context, provider *domainmodel.Provider, req *DocumentOCRRequest) (*DocumentOCRResponse, error) {
	log.Debug().
		Str("provider_id", provider.PublicID).
		Str("provider_name", provider.DisplayName).
		Str("mime_type", req.MimeType).
		Str("filename", req.Filename).
		Msg("[DocumentOCRService] Scan called")

	client, selectedURL, err := s.createRestyClient(ctx, provider)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerInfrastructure, err, "failed to create document OCR client")
	}

	// Set model if not specified
	if req.Model == "" {
		req.Model = s.DefaultModel()
	}

	// Call the provider
	resp, err := s.callProvider(ctx, client, selectedURL, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// ScanWithFileData scans a document using raw file data (base64 encoded).
func (s *DocumentOCRService) ScanWithFileData(ctx context.Context, provider *domainmodel.Provider, fileData []byte, mimeType, filename string) (*DocumentOCRResponse, error) {
	log.Debug().
		Str("provider_id", provider.PublicID).
		Str("provider_name", provider.DisplayName).
		Str("mime_type", mimeType).
		Str("filename", filename).
		Int("data_size", len(fileData)).
		Msg("[DocumentOCRService] ScanWithFileData called")

	client, selectedURL, err := s.createRestyClient(ctx, provider)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerInfrastructure, err, "failed to create document OCR client")
	}

	// Call the provider with multipart form data
	resp, err := s.callProviderWithFileData(ctx, client, selectedURL, fileData, mimeType, filename)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// DefaultModel returns the default OCR model.
func (s *DocumentOCRService) DefaultModel() string {
	if s.cfg != nil && s.cfg.DocumentOCRModel != "" {
		return s.cfg.DocumentOCRModel
	}
	return "docling-v1"
}

// IsEnabled returns whether document OCR is enabled.
func (s *DocumentOCRService) IsEnabled() bool {
	return s.cfg != nil && s.cfg.DocumentOCREnabled
}

// callProvider makes the HTTP call to the OCR provider with JSON body.
func (s *DocumentOCRService) callProvider(ctx context.Context, client *resty.Client, baseURL string, req *DocumentOCRRequest) (*DocumentOCRResponse, error) {
	endpoint := s.resolveEndpoint(baseURL)

	log.Debug().
		Str("endpoint", endpoint).
		Str("model", req.Model).
		Msg("[DocumentOCRService] Calling provider")

	resp, err := client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(req).
		Post(endpoint)

	if err != nil {
		log.Error().Err(err).Str("endpoint", endpoint).Msg("[DocumentOCRService] Provider call failed")
		return nil, platformerrors.NewError(ctx, platformerrors.LayerInfrastructure,
			platformerrors.ErrorTypeExternal,
			fmt.Sprintf("document OCR provider call failed: %v", err),
			nil, "doc-ocr-provider-error")
	}

	return s.parseResponse(ctx, resp)
}

// callProviderWithFileData makes the HTTP call to the OCR provider with multipart form data.
func (s *DocumentOCRService) callProviderWithFileData(ctx context.Context, client *resty.Client, baseURL string, fileData []byte, mimeType, filename string) (*DocumentOCRResponse, error) {
	endpoint := s.resolveEndpoint(baseURL)

	log.Debug().
		Str("endpoint", endpoint).
		Str("mime_type", mimeType).
		Str("filename", filename).
		Msg("[DocumentOCRService] Calling provider with file data")

	// Determine filename extension
	if filename == "" {
		filename = "document"
		switch mimeType {
		case "application/pdf":
			filename = "document.pdf"
		case "application/msword":
			filename = "document.doc"
		case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
			filename = "document.docx"
		case "text/plain":
			filename = "document.txt"
		case "text/markdown":
			filename = "document.md"
		}
	}

	resp, err := client.R().
		SetContext(ctx).
		SetHeader("Accept", "application/json").
		SetMultipartField("file", filename, mimeType, bytes.NewReader(fileData)).
		SetFormData(map[string]string{
			"model": s.DefaultModel(),
		}).
		Post(endpoint)

	if err != nil {
		log.Error().Err(err).Str("endpoint", endpoint).Msg("[DocumentOCRService] Provider call failed")
		return nil, platformerrors.NewError(ctx, platformerrors.LayerInfrastructure,
			platformerrors.ErrorTypeExternal,
			fmt.Sprintf("document OCR provider call failed: %v", err),
			nil, "doc-ocr-provider-error")
	}

	return s.parseResponse(ctx, resp)
}

// parseResponse parses the provider response.
func (s *DocumentOCRService) parseResponse(ctx context.Context, resp *resty.Response) (*DocumentOCRResponse, error) {
	respBytes := resp.Bytes()

	// Check HTTP status
	if resp.StatusCode() >= 400 {
		var errResp DocumentOCRResponse
		if parseErr := json.Unmarshal(respBytes, &errResp); parseErr == nil && errResp.Error != nil {
			return nil, platformerrors.NewError(ctx, platformerrors.LayerInfrastructure,
				platformerrors.ErrorTypeExternal,
				fmt.Sprintf("document OCR provider error: %s", errResp.Error.Message),
				nil, "doc-ocr-provider-error")
		}
		return nil, platformerrors.NewError(ctx, platformerrors.LayerInfrastructure,
			platformerrors.ErrorTypeExternal,
			fmt.Sprintf("document OCR provider returned status %d: %s", resp.StatusCode(), string(respBytes)),
			nil, "doc-ocr-provider-http-error")
	}

	// Parse successful response
	var result DocumentOCRResponse
	if err := json.Unmarshal(respBytes, &result); err != nil {
		log.Error().Err(err).Str("body", truncateStringOCR(string(respBytes), 500)).Msg("[DocumentOCRService] Failed to parse provider response")
		return nil, platformerrors.NewError(ctx, platformerrors.LayerInfrastructure,
			platformerrors.ErrorTypeInternal,
			"failed to parse document OCR provider response",
			err, "doc-ocr-parse-error")
	}

	// Calculate word count if not provided
	if result.WordCount == 0 && result.Text != "" {
		result.WordCount = len(strings.Fields(result.Text))
	}

	log.Debug().
		Int("text_length", len(result.Text)).
		Int("page_count", result.PageCount).
		Int("word_count", result.WordCount).
		Msg("[DocumentOCRService] Provider response received")

	return &result, nil
}

// resolveEndpoint resolves the OCR endpoint from the base URL.
func (s *DocumentOCRService) resolveEndpoint(baseURL string) string {
	trimmedBase := strings.TrimSuffix(baseURL, "/")
	// Default endpoint path for document OCR
	if strings.HasSuffix(trimmedBase, "/v1") {
		return trimmedBase + "/documents/scan"
	}
	return trimmedBase + "/v1/documents/scan"
}

// createRestyClient creates an HTTP client configured for the provider.
func (s *DocumentOCRService) createRestyClient(ctx context.Context, provider *domainmodel.Provider) (*resty.Client, string, error) {
	endpoints := provider.GetEndpoints()
	selectedURL, err := s.router.NextEndpoint(provider.PublicID, endpoints)
	if err != nil {
		switch err {
		case domainmodel.ErrNoEndpoints:
			return nil, "", platformerrors.NewError(ctx, platformerrors.LayerInfrastructure,
				platformerrors.ErrorTypeValidation,
				"no endpoints configured for document OCR provider",
				err, "no-endpoints")
		case domainmodel.ErrNoHealthyEndpoints:
			// Fall back to base URL if no healthy endpoints
			selectedURL = provider.BaseURL
			if selectedURL == "" {
				return nil, "", platformerrors.NewError(ctx, platformerrors.LayerInfrastructure,
					platformerrors.ErrorTypeExternal,
					"no healthy endpoints available for document OCR provider",
					err, "no-healthy-endpoints")
			}
		default:
			return nil, "", platformerrors.NewError(ctx, platformerrors.LayerInfrastructure,
				platformerrors.ErrorTypeInternal,
				fmt.Sprintf("endpoint selection failed: %v", err),
				err, "endpoint-selection-error")
		}
	}

	clientName := fmt.Sprintf("doc-ocr-%s", provider.PublicID)
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
					Msg("[DocumentOCRService] Failed to decrypt API key")
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

// FetchFileFromURL fetches a file from a URL (e.g., presigned media URL).
func (s *DocumentOCRService) FetchFileFromURL(ctx context.Context, fileURL string) ([]byte, string, error) {
	client := &http.Client{
		Timeout: s.timeout,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return nil, "", platformerrors.NewError(ctx, platformerrors.LayerInfrastructure,
			platformerrors.ErrorTypeInternal,
			"failed to create file fetch request",
			err, "file-fetch-request-error")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", platformerrors.NewError(ctx, platformerrors.LayerInfrastructure,
			platformerrors.ErrorTypeExternal,
			"failed to fetch file from URL",
			err, "file-fetch-error")
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, "", platformerrors.NewError(ctx, platformerrors.LayerInfrastructure,
			platformerrors.ErrorTypeExternal,
			fmt.Sprintf("file fetch returned status %d", resp.StatusCode),
			nil, "file-fetch-http-error")
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", platformerrors.NewError(ctx, platformerrors.LayerInfrastructure,
			platformerrors.ErrorTypeInternal,
			"failed to read file data",
			err, "file-read-error")
	}

	contentType := resp.Header.Get("Content-Type")
	return data, contentType, nil
}

// truncateString truncates a string for logging purposes (helper already exists but defining locally for safety).
func truncateStringOCR(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
