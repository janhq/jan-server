package steps

func limitImageAssets(assets []map[string]any, max int) []map[string]any {
	if max <= 0 || len(assets) <= max {
		return assets
	}
	return assets[:max]
}
