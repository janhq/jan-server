package schemas

// DataBank contains extracted facts and datasets.
type DataBank struct {
	Facts    []Fact    `json:"facts"`
	Datasets []Dataset `json:"datasets"`
}

// Fact is a single sourced claim.
type Fact struct {
	Claim     string `json:"claim"`
	Value     string `json:"value"`
	Unit      string `json:"unit"`
	SourceURL string `json:"sourceUrl"`
	Date      string `json:"date"`
}

// Dataset is a chart-ready dataset.
type Dataset struct {
	ID         string      `json:"id"`
	Kind       string      `json:"kind"`
	Data       DatasetData `json:"data"`
	SourceNote string      `json:"sourceNote"`
}

// DatasetData holds labels and series values.
type DatasetData struct {
	Labels []string        `json:"labels"`
	Series []DatasetSeries `json:"series"`
}

// DatasetSeries is one series in a dataset.
type DatasetSeries struct {
	Name   string    `json:"name"`
	Values []float64 `json:"values"`
}
