package domain

import "time"

// Model captures public metadata for a provider-backed model.
type Model struct {
	ID           string    `json:"id"`
	Provider     string    `json:"provider"`
	DisplayName  string    `json:"display_name"`
	Family       string    `json:"family"`
	Capabilities []string  `json:"capabilities"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
