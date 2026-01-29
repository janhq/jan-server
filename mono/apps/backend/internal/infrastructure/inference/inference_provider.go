package inference

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"jan-server/mono/apps/backend/internal/config"
	domainmodel "jan-server/mono/apps/backend/internal/domain/model"
	"jan-server/mono/apps/backend/internal/infrastructure/router"
	"jan-server/mono/apps/backend/internal/utils/crypto"
	httpclients "jan-server/mono/apps/backend/internal/utils/httpclients"
	chatclient "jan-server/mono/apps/backend/internal/utils/httpclients/chat"
	"jan-server/mono/apps/backend/internal/utils/platformerrors"

	"resty.dev/v3"
)

type InferenceProvider struct {
	streamTimeout time.Duration
	router        domainmodel.EndpointRouter
}

func NewInferenceProvider(cfg *config.Config) *InferenceProvider {
	timeout := 300 * time.Second // default 5 minutes
	if cfg != nil && cfg.StreamTimeout > 0 {
		timeout = cfg.StreamTimeout
	}
	return &InferenceProvider{
		streamTimeout: timeout,
		router:        router.NewRoundRobinRouter(),
	}
}

func (ip *InferenceProvider) GetChatCompletionClient(ctx context.Context, provider *domainmodel.Provider) (*chatclient.ChatCompletionClient, error) {
	log.Debug().
		Str("provider_id", provider.PublicID).
		Str("provider_name", provider.DisplayName).
		Str("provider_kind", string(provider.Kind)).
		Msg("[DEBUG] GetChatCompletionClient: creating client for provider")

	client, selectedURL, err := ip.createRestyClient(ctx, provider)
	if err != nil {
		log.Error().
			Err(err).
			Str("provider_id", provider.PublicID).
			Str("provider_name", provider.DisplayName).
			Msg("[DEBUG] GetChatCompletionClient: failed to create resty client")
		return nil, err
	}

	clientName := provider.DisplayName
	log.Debug().
		Str("provider_name", clientName).
		Str("base_url", selectedURL).
		Msg("[DEBUG] GetChatCompletionClient: client created successfully")

	return chatclient.NewChatCompletionClient(client, clientName, selectedURL, chatclient.WithStreamTimeout(ip.streamTimeout)), nil
}

func (ip *InferenceProvider) GetChatModelClient(ctx context.Context, provider *domainmodel.Provider) (*chatclient.ChatModelClient, error) {
	client, selectedURL, err := ip.createRestyClient(ctx, provider)
	if err != nil {
		return nil, err
	}

	clientName := provider.DisplayName
	return chatclient.NewChatModelClient(client, clientName, selectedURL), nil
}

func (ip *InferenceProvider) ListModels(ctx context.Context, provider *domainmodel.Provider) ([]chatclient.Model, error) {
	modelClient, err := ip.GetChatModelClient(ctx, provider)
	if err != nil {
		return nil, err
	}

	resp, err := modelClient.ListModels(ctx)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

func (ip *InferenceProvider) createRestyClient(ctx context.Context, provider *domainmodel.Provider) (*resty.Client, string, error) {
	clientName := fmt.Sprintf("%sClient", provider.PublicID)

	requestID := ""
	if rid, ok := ctx.Value("request_id").(string); ok {
		requestID = rid
	}

	endpoints := provider.GetEndpoints()
	selectedURL, err := ip.router.NextEndpoint(provider.PublicID, endpoints)
	if err != nil {
		switch err {
		case domainmodel.ErrNoEndpoints:
			return nil, "", platformerrors.NewError(ctx, platformerrors.LayerInfrastructure, platformerrors.ErrorTypeValidation, "provider has no endpoints configured", err, "0af7cb2b-9c15-4ac9-9a32-8b7d50505145")
		case domainmodel.ErrNoHealthyEndpoints:
			log.Warn().
				Str("provider_id", provider.PublicID).
				Str("provider_name", provider.DisplayName).
				Str("request_id", requestID).
				Int("total_endpoints", len(endpoints)).
				Msg("all endpoints unhealthy, using fallback")
		default:
			log.Warn().
				Err(err).
				Str("provider_id", provider.PublicID).
				Str("provider_name", provider.DisplayName).
				Str("request_id", requestID).
				Msg("unexpected endpoint selection error")
		}
	}

	if selectedURL == "" {
		selectedURL = provider.BaseURL
		if selectedURL == "" {
			return nil, "", platformerrors.NewError(ctx, platformerrors.LayerInfrastructure, platformerrors.ErrorTypeValidation, "provider has no base URL configured", nil, "f646b2f0-b8f0-44c1-8b5b-9e1a45c03684")
		}
	}

	log.Debug().
		Str("provider_id", provider.PublicID).
		Str("provider_name", provider.DisplayName).
		Str("selected_url", selectedURL).
		Str("request_id", requestID).
		Int("total_endpoints", len(endpoints)).
		Bool("has_multiple", provider.HasMultipleEndpoints()).
		Msg("selected endpoint for request")

	client := httpclients.NewClient(clientName)
	client.SetBaseURL(selectedURL)

	// Set authorization header if API key exists
	if provider.EncryptedAPIKey != "" {
		apiKey, err := ip.decryptAPIKey(ctx, provider.EncryptedAPIKey)
		if err != nil {
			return nil, "", platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to decrypt API key")
		}
		if strings.TrimSpace(apiKey) != "" && strings.ToLower(apiKey) != "none" {
			switch provider.Kind {
			case domainmodel.ProviderAzureOpenAI:
				client.SetHeader("api-key", apiKey)
			case domainmodel.ProviderAnthropic:
				client.SetHeader("X-API-Key", apiKey)
				client.SetHeader("Anthropic-Version", "2023-06-01")
			case domainmodel.ProviderCohere:
				client.SetHeader("Authorization", fmt.Sprintf("Bearer %s", apiKey))
			default:
				client.SetHeader("Authorization", fmt.Sprintf("Bearer %s", apiKey))
			}
		}
	}

	return client, selectedURL, nil
}

func (ip *InferenceProvider) decryptAPIKey(ctx context.Context, encryptedAPIKey string) (string, error) {
	if encryptedAPIKey == "" {
		return "", nil
	}

	secret := strings.TrimSpace(config.GetGlobal().ModelProviderSecret)
	if secret == "" {
		return "", platformerrors.NewError(ctx, platformerrors.LayerInfrastructure, platformerrors.ErrorTypeInternal, "MODEL_PROVIDER_SECRET not configured", nil, "8f07ea41-1096-405b-ae2e-cde06564e5bc")
	}

	plainText, err := crypto.DecryptString(secret, encryptedAPIKey)
	if err != nil {
		return "", err
	}

	return plainText, nil
}
