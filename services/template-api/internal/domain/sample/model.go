package sample

// Sample represents a domain entity returned by the sample use case.
type Sample struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}
