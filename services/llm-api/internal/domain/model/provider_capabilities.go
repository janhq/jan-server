package model

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"jan-server/services/llm-api/internal/infrastructure/logger"
)

// ProviderCapabilitiesDefaults holds the default capabilities for each provider kind
type ProviderCapabilitiesDefaults struct {
	ImageInput     ImageInputCapability     `json:"image_input"`
	FileAttachment FileAttachmentCapability `json:"file_attachment"`
}

var (
	defaultCapabilities     map[string]ProviderCapabilitiesDefaults
	defaultCapabilitiesMux  sync.RWMutex
	defaultCapabilitiesOnce sync.Once
)

// LoadDefaultCapabilities loads provider capabilities from providers_metadata_default.yml
func LoadDefaultCapabilities(configPath string) error {
	defaultCapabilitiesOnce.Do(func() {
		// Defer panic recovery in case logger or other dependencies aren't ready
		defer func() {
			if r := recover(); r != nil {
				// Silently use hardcoded defaults if anything fails during initialization
				defaultCapabilitiesMux.Lock()
				if defaultCapabilities == nil {
					defaultCapabilities = getHardcodedDefaults()
				}
				defaultCapabilitiesMux.Unlock()
			}
		}()

		// Try the provided path first
		yamlPath := configPath
		if yamlPath == "" {
			// Default to config/providers_metadata_default.yml
			yamlPath = filepath.Join("config", "providers_metadata_default.yml")
		}

		data, err := os.ReadFile(yamlPath)
		if err != nil {
			// Use hardcoded fallbacks if file not found
			defaultCapabilitiesMux.Lock()
			defaultCapabilities = getHardcodedDefaults()
			defaultCapabilitiesMux.Unlock()

			// Try to log warning
			log := logger.GetLogger()
			log.Warn().
				Str("path", yamlPath).
				Err(err).
				Msg("Could not load provider capabilities defaults, using hardcoded fallbacks")
			return
		}

		var defs map[string]ProviderCapabilitiesDefaults
		if err := json.Unmarshal(data, &defs); err != nil {
			// Use hardcoded fallbacks if parse fails
			defaultCapabilitiesMux.Lock()
			defaultCapabilities = getHardcodedDefaults()
			defaultCapabilitiesMux.Unlock()

			// Try to log warning
			log := logger.GetLogger()
			log.Warn().
				Str("path", yamlPath).
				Err(err).
				Msg("Could not parse provider capabilities defaults, using hardcoded fallbacks")
			return
		}

		defaultCapabilitiesMux.Lock()
		defaultCapabilities = defs
		defaultCapabilitiesMux.Unlock()

		// Log success
		log := logger.GetLogger()
		log.Info().
			Str("path", yamlPath).
			Int("providers", len(defs)).
			Msg("Loaded provider capabilities defaults")
	})

	return nil
}

// GetDefaultCapabilities returns the default capabilities for a provider kind
func GetDefaultCapabilities(kind ProviderKind) ProviderCapabilitiesDefaults {
	// Ensure defaults are loaded
	LoadDefaultCapabilities("")

	defaultCapabilitiesMux.RLock()
	defer defaultCapabilitiesMux.RUnlock()

	if caps, exists := defaultCapabilities[string(kind)]; exists {
		return caps
	}

	// Return empty/unsupported capabilities for unknown providers
	return ProviderCapabilitiesDefaults{
		ImageInput: ImageInputCapability{
			Supported: false,
		},
		FileAttachment: FileAttachmentCapability{
			Supported: false,
		},
	}
}

// getHardcodedDefaults returns hardcoded defaults as a fallback
func getHardcodedDefaults() map[string]ProviderCapabilitiesDefaults {
	return map[string]ProviderCapabilitiesDefaults{
		"openai": {
			ImageInput: ImageInputCapability{
				Supported: true,
				URL:       true,
				Base64:    true,
				Schema:    "messages[].content[].type='image_url'; image_url.url=https:// or data:image/...;base64,...",
			},
			FileAttachment: FileAttachmentCapability{
				Supported:  true,
				URL:        false,
				Base64:     false,
				FileUpload: true,
				Schema:     "messages[].content[].type='input_file'; file_id from Files API upload",
			},
		},
		"azure_openai": {
			ImageInput: ImageInputCapability{
				Supported: true,
				URL:       true,
				Base64:    true,
				Schema:    "identical to OpenAI vision; supports https:// and data:image/...;base64,...",
			},
			FileAttachment: FileAttachmentCapability{
				Supported:  true,
				URL:        false,
				Base64:     false,
				FileUpload: true,
				Schema:     "Files uploaded to Azure resource → reference with file_id",
			},
		},
		"google": {
			ImageInput: ImageInputCapability{
				Supported: true,
				URL:       false,
				Base64:    true,
				Schema:    "messages[].parts[].inline_data={mime_type,data} or file_data={file_uri,mime_type}",
			},
			FileAttachment: FileAttachmentCapability{
				Supported:  true,
				URL:        false,
				Base64:     true,
				FileUpload: true,
				Schema:     "Gemini: inline_data for small files; file_data.file_uri for uploaded files",
			},
		},
		"anthropic": {
			ImageInput: ImageInputCapability{
				Supported: true,
				URL:       true,
				Base64:    true,
				Schema:    "messages[].content[].type='image'; image_url=https:// or data:image/...;base64,... or file_id",
			},
			FileAttachment: FileAttachmentCapability{
				Supported:  true,
				URL:        true,
				Base64:     true,
				FileUpload: true,
				Schema:     "messages[].content[].type='input_file'; supports url, inline base64, or file_id",
			},
		},
		"ollama": {
			ImageInput: ImageInputCapability{
				Supported: true,
				URL:       true,
				Base64:    true,
				Schema:    "local path or data URI; multimodal models like llava accept both",
			},
			FileAttachment: FileAttachmentCapability{
				Supported:  true,
				URL:        true,
				Base64:     true,
				FileUpload: false,
				Schema:     "file={path or base64}; handled locally by Ollama server",
			},
		},
		"custom": {
			ImageInput: ImageInputCapability{
				Supported: false,
			},
			FileAttachment: FileAttachmentCapability{
				Supported: false,
			},
		},
		"ocr": {
			ImageInput: ImageInputCapability{
				Supported: false,
			},
			FileAttachment: FileAttachmentCapability{
				Supported:  true,
				URL:        true,
				Base64:     false,
				FileUpload: true,
				Schema:     "Docling OCR supports file upload (/v1/convert/file) or URL source (/v1/convert/source)",
			},
		},
	}
}
