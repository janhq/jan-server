package requestmodels

type ModelCatalogFilterParams struct {
	Status      *string `form:"status"`
	IsModerated *bool   `form:"is_moderated"`
	Active      *bool   `form:"active"`
}

type ProviderModelFilterParams struct {
	ProviderPublicID *string `form:"provider_id"`
	ModelKey         *string `form:"model_key"`
	Active           *bool   `form:"active"`
	SupportsImages   *bool   `form:"supports_images"`
}
