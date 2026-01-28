package modelhandler

import (
	"context"
	"sort"
	"strings"

	domainmodel "jan-server/services/llm-api/internal/domain/model"
)

type ModelHandler struct {
	provider             *domainmodel.ProviderService
	providerModelService *domainmodel.ProviderModelService
}

func NewModelHandler(
	provider *domainmodel.ProviderService,
	providerModelService *domainmodel.ProviderModelService,
) *ModelHandler {
	return &ModelHandler{
		provider:             provider,
		providerModelService: providerModelService,
	}
}

func (modelHandler *ModelHandler) BuildAccessibleProviderModels(ctx context.Context) (*domainmodel.AccessibleModels, error) {
	providers, err := modelHandler.provider.FindAllActiveProviders(ctx)
	if err != nil {
		return nil, err
	}

	providerIDs := make([]uint, 0, len(providers))
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		providerIDs = append(providerIDs, provider.ID)
	}

	providerModels, err := modelHandler.providerModelService.ListActiveProviderByIDs(ctx, providerIDs)
	if err != nil {
		return nil, err
	}

	result := &domainmodel.AccessibleModels{
		Providers:      providers,
		ProviderModels: providerModels,
	}
	return result, nil
}

type modelAggregate struct {
	response      domainmodel.ProviderModel
	providerKind  domainmodel.ProviderKind
	hasPricing    bool
	cheapestPrice domainmodel.MicroUSD
}

func (modelHandler *ModelHandler) MergeModels(
	providerModels []*domainmodel.ProviderModel,
	providerByID map[uint]*domainmodel.Provider,
) []*domainmodel.ProviderModel {
	aggregated := map[string]modelAggregate{}

	for _, pm := range providerModels {
		if pm == nil {
			continue
		}
		provider := providerByID[pm.ProviderID]
		if provider == nil {
			continue
		}
		cheapestPrice, hasPricing := lowestPricingAmount(pm.Pricing)
		incoming := modelAggregate{
			response:      *pm,
			providerKind:  provider.Kind,
			hasPricing:    hasPricing,
			cheapestPrice: cheapestPrice,
		}

		if existing, ok := aggregated[strings.ToLower(pm.ModelPublicID)]; ok {
			if !shouldReplaceModel(existing, incoming) {
				continue
			}
		}

		aggregated[pm.ModelPublicID] = incoming
	}

	candidates := make([]modelAggregate, 0, len(aggregated))
	for _, model := range aggregated {
		candidates = append(candidates, model)
	}

	sort.Slice(candidates, func(i, j int) bool {
		// Jan models first
		iIsJan := candidates[i].providerKind == domainmodel.ProviderJan
		jIsJan := candidates[j].providerKind == domainmodel.ProviderJan
		if iIsJan && !jIsJan {
			return true
		} else if !iIsJan && jIsJan {
			return false
		}

		// Then by category order number (lower = higher priority)
		iCatOrder := candidates[i].response.CategoryOrderNumber
		jCatOrder := candidates[j].response.CategoryOrderNumber
		if iCatOrder != jCatOrder {
			return iCatOrder < jCatOrder
		}

		// Then by model order number within category (lower = higher priority)
		iModelOrder := candidates[i].response.ModelOrderNumber
		jModelOrder := candidates[j].response.ModelOrderNumber
		if iModelOrder != jModelOrder {
			return iModelOrder < jModelOrder
		}

		// Finally by ID as tiebreaker
		return candidates[i].response.ID < candidates[j].response.ID
	})

	result := make([]*domainmodel.ProviderModel, len(candidates))
	for idx, candidate := range candidates {
		result[idx] = &candidate.response
	}

	return result
}

func lowestPricingAmount(pricing domainmodel.Pricing) (domainmodel.MicroUSD, bool) {
	if len(pricing.Lines) == 0 {
		return 0, false
	}
	lowest := pricing.Lines[0].Amount
	for _, line := range pricing.Lines[1:] {
		if line.Amount < lowest {
			lowest = line.Amount
		}
	}
	return lowest, true
}

func shouldReplaceModel(existing, incoming modelAggregate) bool {
	if incoming.hasPricing && existing.hasPricing {
		if incoming.cheapestPrice < existing.cheapestPrice {
			return true
		}
		if incoming.cheapestPrice > existing.cheapestPrice {
			return false
		}
	}

	if incoming.hasPricing && !existing.hasPricing {
		return true
	}
	if !incoming.hasPricing && existing.hasPricing {
		return false
	}

	incomingIsJan := incoming.providerKind == domainmodel.ProviderJan
	existingIsJan := existing.providerKind == domainmodel.ProviderJan
	if incomingIsJan && !existingIsJan {
		return true
	}
	if existingIsJan && !incomingIsJan {
		return false
	}

	return false
}

// GetFirstActiveModelID returns the first model ID using the same logic as /v1/models endpoint.
// This is used to set the default selected_model in user preferences.
func (modelHandler *ModelHandler) GetFirstActiveModelID(ctx context.Context) (string, error) {
	accessibleModels, err := modelHandler.BuildAccessibleProviderModels(ctx)
	if err != nil {
		return "", err
	}

	if accessibleModels == nil || len(accessibleModels.ProviderModels) == 0 || len(accessibleModels.Providers) == 0 {
		return "", nil
	}

	providerByID := make(map[uint]*domainmodel.Provider, len(accessibleModels.Providers))
	for _, provider := range accessibleModels.Providers {
		if provider == nil {
			continue
		}
		providerByID[provider.ID] = provider
	}

	mergedModels := modelHandler.MergeModels(accessibleModels.ProviderModels, providerByID)
	if len(mergedModels) == 0 {
		return "", nil
	}

	return mergedModels[0].ModelPublicID, nil
}
