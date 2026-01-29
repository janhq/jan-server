package requestmodels

type ModelCatalogFilterParams struct {
	Status              *string `form:"status"`
	IsModerated         *bool   `form:"is_moderated"`
	Active              *bool   `form:"active"`
	Experimental        *bool   `form:"experimental"`
	RequiresFeatureFlag *string `form:"requires_feature_flag"`
	SupportsImages      *bool   `form:"supports_images"`
	SupportsEmbeddings  *bool   `form:"supports_embeddings"`
	SupportsReasoning   *bool   `form:"supports_reasoning"`
	SupportsAudio       *bool   `form:"supports_audio"`
	SupportsVideo       *bool   `form:"supports_video"`
	SupportsTools       *bool   `form:"supports_tools"`
	Family              *string `form:"family"`
}

type ProviderModelFilterParams struct {
	ProviderPublicID *string `form:"provider_id"`
	ModelKey         *string `form:"model_key"`
	Active           *bool   `form:"active"`
	SupportsImages   *bool   `form:"supports_images"`
	SearchText       *string `form:"search"`
}
