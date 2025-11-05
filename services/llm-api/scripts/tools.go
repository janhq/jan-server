//go:build tools
// +build tools

// Package tools tracks tool dependencies for the project
package tools

import (
	_ "github.com/google/wire/cmd/wire"
	_ "github.com/swaggo/swag/cmd/swag"
	_ "gorm.io/gen"
)

//go:generate go install github.com/google/wire/cmd/wire
//go:generate go install github.com/swaggo/swag/cmd/swag
//go:generate go install gorm.io/gen/tools/gentool@latest
