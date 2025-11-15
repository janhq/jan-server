package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management commands",
	Long:  `Manage Jan Server configuration files, validate, export, and inspect config values.`,
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration files",
	Long:  `Validate configuration files for syntax errors and required fields.`,
	RunE:  runConfigValidate,
}

var configExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export configuration in various formats",
	Long:  `Export configuration as environment variables, JSON, YAML, or docker-compose env file.`,
	RunE:  runConfigExport,
}

var configShowCmd = &cobra.Command{
	Use:   "show [service]",
	Short: "Show configuration values",
	Long:  `Display configuration values for a specific service or entire config.`,
	RunE:  runConfigShow,
}

var configK8sCmd = &cobra.Command{
	Use:   "k8s-values",
	Short: "Generate Kubernetes Helm values",
	Long:  `Generate Kubernetes Helm values file from configuration.`,
	RunE:  runConfigK8sValues,
}

func init() {
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configExportCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configK8sCmd)

	// validate flags
	configValidateCmd.Flags().StringP("file", "f", "config/defaults.yaml", "Config file to validate")
	configValidateCmd.Flags().String("schema", "", "Schema file to validate against")
	configValidateCmd.Flags().StringP("env", "e", "", "Environment to validate")

	// export flags
	configExportCmd.Flags().StringP("file", "f", "config/defaults.yaml", "Config file to export")
	configExportCmd.Flags().String("format", "env", "Output format: env, docker-env, json, yaml")
	configExportCmd.Flags().String("prefix", "", "Add prefix to exported variables")
	configExportCmd.Flags().StringP("output", "o", "", "Output file (default: stdout)")

	// show flags
	configShowCmd.Flags().StringP("file", "f", "config/defaults.yaml", "Config file to read")
	configShowCmd.Flags().String("path", "", "Config path to show (e.g., services.llm-api)")
	configShowCmd.Flags().String("format", "yaml", "Output format: yaml, json, value")

	// k8s-values flags
	configK8sCmd.Flags().StringP("env", "e", "development", "Environment (development, production, etc.)")
	configK8sCmd.Flags().StringP("output", "o", "", "Output file (default: stdout)")
	configK8sCmd.Flags().StringSlice("set", []string{}, "Override values (key=value)")
}

func runConfigValidate(cmd *cobra.Command, args []string) error {
	configFile, _ := cmd.Flags().GetString("file")
	schemaFile, _ := cmd.Flags().GetString("schema")
	env, _ := cmd.Flags().GetString("env")

	fmt.Printf("Validating configuration...\n")
	fmt.Printf("  Config: %s\n", configFile)
	if env != "" {
		fmt.Printf("  Environment: %s\n", env)
	}
	if schemaFile != "" {
		fmt.Printf("  Schema: %s\n", schemaFile)
	}

	// Load config file
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	// Parse YAML
	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("parse YAML: %w", err)
	}

	// If environment specified, merge environment overrides
	if env != "" {
		envFile := filepath.Join("config", env+".yaml")
		if _, err := os.Stat(envFile); err == nil {
			envData, err := os.ReadFile(envFile)
			if err != nil {
				return fmt.Errorf("read environment file: %w", err)
			}

			var envConfig map[string]interface{}
			if err := yaml.Unmarshal(envData, &envConfig); err != nil {
				return fmt.Errorf("parse environment YAML: %w", err)
			}

			// Merge configs
			mergeMaps(config, envConfig)
		}
	}

	// Basic validation
	errors := []string{}

	// Check required top-level keys
	requiredKeys := []string{"services"}
	for _, key := range requiredKeys {
		if _, ok := config[key]; !ok {
			errors = append(errors, fmt.Sprintf("missing required key: %s", key))
		}
	}

	if len(errors) > 0 {
		fmt.Println("\n Validation failed:")
		for _, err := range errors {
			fmt.Printf("  - %s\n", err)
		}
		return fmt.Errorf("validation failed with %d errors", len(errors))
	}

	fmt.Println("\n Configuration is valid")
	return nil
}

func runConfigExport(cmd *cobra.Command, args []string) error {
	configFile, _ := cmd.Flags().GetString("file")
	format, _ := cmd.Flags().GetString("format")
	prefix, _ := cmd.Flags().GetString("prefix")
	outputFile, _ := cmd.Flags().GetString("output")

	// Load config
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("parse YAML: %w", err)
	}

	// Generate output
	var output string
	switch format {
	case "env":
		output = exportAsEnv(config, prefix)
	case "docker-env":
		output = exportAsDockerEnv(config, prefix)
	case "json":
		jsonData, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal JSON: %w", err)
		}
		output = string(jsonData)
	case "yaml":
		yamlData, err := yaml.Marshal(config)
		if err != nil {
			return fmt.Errorf("marshal YAML: %w", err)
		}
		output = string(yamlData)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	// Write output
	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(output), 0644); err != nil {
			return fmt.Errorf("write output file: %w", err)
		}
		fmt.Printf(" Exported to %s\n", outputFile)
	} else {
		fmt.Print(output)
	}

	return nil
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	configFile, _ := cmd.Flags().GetString("file")
	path, _ := cmd.Flags().GetString("path")
	format, _ := cmd.Flags().GetString("format")

	// If service specified in args, use it as path
	if len(args) > 0 {
		if path == "" {
			path = "services." + args[0]
		}
	}

	// Load config
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("parse YAML: %w", err)
	}

	// Navigate to path if specified
	var value interface{} = config
	if path != "" {
		parts := strings.Split(path, ".")
		for _, part := range parts {
			if m, ok := value.(map[string]interface{}); ok {
				if v, exists := m[part]; exists {
					value = v
				} else {
					return fmt.Errorf("path not found: %s", path)
				}
			} else {
				return fmt.Errorf("cannot navigate path: %s", path)
			}
		}
	}

	// Format output
	switch format {
	case "yaml":
		yamlData, err := yaml.Marshal(value)
		if err != nil {
			return fmt.Errorf("marshal YAML: %w", err)
		}
		fmt.Print(string(yamlData))
	case "json":
		jsonData, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	case "value":
		fmt.Println(value)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	return nil
}

func runConfigK8sValues(cmd *cobra.Command, args []string) error {
	env, _ := cmd.Flags().GetString("env")
	outputFile, _ := cmd.Flags().GetString("output")
	_, _ = cmd.Flags().GetStringSlice("set") // TODO: use overrides

	fmt.Printf("Generating Kubernetes Helm values for environment: %s\n", env) // TODO: Implement K8s values generation
	output := fmt.Sprintf("# Generated Helm values for %s environment\n", env)
	output += "# This is a placeholder - integrate with pkg/config/k8s\n"

	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(output), 0644); err != nil {
			return fmt.Errorf("write output file: %w", err)
		}
		fmt.Printf(" Generated values file: %s\n", outputFile)
	} else {
		fmt.Print(output)
	}

	return nil
}

// Helper functions

func mergeMaps(dst, src map[string]interface{}) {
	for k, v := range src {
		if dstVal, ok := dst[k]; ok {
			if dstMap, dstOk := dstVal.(map[string]interface{}); dstOk {
				if srcMap, srcOk := v.(map[string]interface{}); srcOk {
					mergeMaps(dstMap, srcMap)
					continue
				}
			}
		}
		dst[k] = v
	}
}

func exportAsEnv(config map[string]interface{}, prefix string) string {
	var lines []string
	flatten("", config, prefix, &lines)

	var result strings.Builder
	for _, line := range lines {
		result.WriteString("export ")
		result.WriteString(line)
		result.WriteString("\n")
	}
	return result.String()
}

func exportAsDockerEnv(config map[string]interface{}, prefix string) string {
	var lines []string
	flatten("", config, prefix, &lines)

	var result strings.Builder
	for _, line := range lines {
		result.WriteString(line)
		result.WriteString("\n")
	}
	return result.String()
}

func flatten(prefix string, data interface{}, globalPrefix string, lines *[]string) {
	switch v := data.(type) {
	case map[string]interface{}:
		for key, val := range v {
			newPrefix := key
			if prefix != "" {
				newPrefix = prefix + "_" + strings.ToUpper(key)
			} else {
				newPrefix = strings.ToUpper(key)
			}
			flatten(newPrefix, val, globalPrefix, lines)
		}
	default:
		key := prefix
		if globalPrefix != "" {
			key = globalPrefix + "_" + prefix
		}
		*lines = append(*lines, fmt.Sprintf("%s=%v", key, v))
	}
}
