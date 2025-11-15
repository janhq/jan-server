// Package main demonstrates K8s values generation
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/janhq/jan-server/pkg/config"
	"github.com/janhq/jan-server/pkg/config/k8s"
	"gopkg.in/yaml.v3"
)

func main() {
	// Load configuration
	loader := config.NewConfigLoader("development", "../../config/defaults.yaml")
	cfg, err := loader.Load(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Create generator
	generator := k8s.NewValuesGenerator(cfg)

	// Generate for development
	fmt.Println("=== Generating Helm Values ===\n")

	// Write to file
	outputPath := "values-dev.yaml"
	if err := generator.GenerateToFile(outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Development values written to: %s\n", outputPath)

	// Generate production overrides
	prodPath := "values-prod.yaml"
	values, err := generator.GenerateWithOverrides("production")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating production values: %v\n", err)
		os.Exit(1)
	}

	data, _ := yaml.Marshal(values)
	header := fmt.Sprintf("# Helm values generated from configuration\n# Environment: production\n# Version: %s\n\n", cfg.Meta.Version)
	if err := os.WriteFile(prodPath, []byte(header+string(data)), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing production file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Production values written to: %s\n", prodPath)
	fmt.Println("\nDone!")
}
