package codegen

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/invopop/jsonschema"
	"github.com/janhq/jan-server/pkg/config"
)

// GenerateJSONSchema generates JSON Schema files from Go structs
func GenerateJSONSchema(outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	// Generate main config schema
	reflector := &jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            false,
		ExpandedStruct:            true,
	}

	schema := reflector.Reflect(&config.Config{})
	schema.Title = "Jan Server Configuration"
	schema.Description = "Complete configuration schema for Jan Server infrastructure and services"
	schema.Version = "1.0.0"

	// Write main schema
	mainSchemaPath := filepath.Join(outputDir, "config.schema.json")
	if err := writeSchemaFile(mainSchemaPath, schema); err != nil {
		return fmt.Errorf("write main schema: %w", err)
	}

	fmt.Printf("✓ Generated %s\n", mainSchemaPath)

	// Generate per-section schemas for better modularity
	sections := map[string]interface{}{
		"infrastructure": config.InfrastructureConfig{},
		"services":       config.ServicesConfig{},
		"inference":      config.InferenceConfig{},
		"monitoring":     config.MonitoringConfig{},
	}

	for name, typ := range sections {
		sectionSchema := reflector.Reflect(typ)
		sectionPath := filepath.Join(outputDir, fmt.Sprintf("%s.schema.json", name))
		if err := writeSchemaFile(sectionPath, sectionSchema); err != nil {
			return fmt.Errorf("write %s schema: %w", name, err)
		}
		fmt.Printf("✓ Generated %s\n", sectionPath)
	}

	return nil
}

func writeSchemaFile(path string, schema *jsonschema.Schema) error {
	data, err := schema.MarshalJSON()
	if err != nil {
		return fmt.Errorf("marshal schema: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}
