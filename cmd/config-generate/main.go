package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/janhq/jan-server/pkg/config/codegen"
)

func main() {
	var (
		outputDir  = flag.String("output", "config", "Output directory for generated files")
		schemaOnly = flag.Bool("schema-only", false, "Generate only JSON schemas")
		yamlOnly   = flag.Bool("yaml-only", false, "Generate only YAML defaults")
	)
	flag.Parse()

	log.Println("Starting configuration code generation...")

	// Determine what to generate
	generateSchema := !*yamlOnly
	generateYAML := !*schemaOnly

	// Generate JSON Schema
	if generateSchema {
		schemaDir := filepath.Join(*outputDir, "schema")
		log.Printf("Generating JSON Schema files in %s...", schemaDir)
		if err := codegen.GenerateJSONSchema(schemaDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating JSON schema: %v\n", err)
			os.Exit(1)
		}
	}

	// Generate YAML defaults
	if generateYAML {
		defaultsPath := filepath.Join(*outputDir, "defaults.yaml")
		log.Printf("Generating YAML defaults in %s...", defaultsPath)
		if err := codegen.GenerateDefaultsYAML(defaultsPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating YAML defaults: %v\n", err)
			os.Exit(1)
		}
	}

	log.Println("✓ Configuration generation complete!")
}
