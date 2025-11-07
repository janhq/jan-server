package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

type SwaggerSpec struct {
	Swagger             string                   `json:"swagger"`
	Info                map[string]interface{}   `json:"info"`
	Host                string                   `json:"host,omitempty"`
	BasePath            string                   `json:"basePath,omitempty"`
	Schemes             []string                 `json:"schemes,omitempty"`
	Paths               map[string]interface{}   `json:"paths"`
	Definitions         map[string]interface{}   `json:"definitions,omitempty"`
	Tags                []map[string]interface{} `json:"tags,omitempty"`
	SecurityDefinitions map[string]interface{}   `json:"securityDefinitions,omitempty"`
}

func main() {
	llmAPIPath := flag.String("llm-api", "services/llm-api/docs/swagger/swagger.json", "Path to llm-api swagger.json")
	mcpToolsPath := flag.String("mcp-tools", "services/mcp-tools/docs/swagger/swagger.json", "Path to mcp-tools swagger.json")
	outputPath := flag.String("output", "services/llm-api/docs/swagger/swagger-combined.json", "Output path for combined swagger.json")
	flag.Parse()

	// Read llm-api swagger
	llmAPIData, err := ioutil.ReadFile(*llmAPIPath)
	if err != nil {
		fmt.Printf("Error reading llm-api swagger: %v\n", err)
		os.Exit(1)
	}

	var llmAPISpec SwaggerSpec
	if err := json.Unmarshal(llmAPIData, &llmAPISpec); err != nil {
		fmt.Printf("Error parsing llm-api swagger: %v\n", err)
		os.Exit(1)
	}

	// Read mcp-tools swagger
	mcpToolsData, err := ioutil.ReadFile(*mcpToolsPath)
	if err != nil {
		fmt.Printf("Warning: Could not read mcp-tools swagger: %v\n", err)
		fmt.Println("Creating combined spec with only llm-api...")
		// If mcp-tools doesn't exist, just output llm-api spec
		if err := writeOutput(*outputPath, llmAPIData); err != nil {
			fmt.Printf("Error writing output: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Combined swagger created at: %s (llm-api only)\n", *outputPath)
		return
	}

	var mcpToolsSpec SwaggerSpec
	if err := json.Unmarshal(mcpToolsData, &mcpToolsSpec); err != nil {
		fmt.Printf("Error parsing mcp-tools swagger: %v\n", err)
		os.Exit(1)
	}

	// Merge specs
	combined := llmAPISpec
	combined.Info["title"] = "Jan Server API (LLM API + MCP Tools)"
	combined.Info["description"] = "Unified API documentation for Jan Server including LLM API (OpenAI-compatible) and MCP Tools (search and scraping capabilities)"

	// Merge paths with /mcp prefix for mcp-tools
	if combined.Paths == nil {
		combined.Paths = make(map[string]interface{})
	}
	for path, methods := range mcpToolsSpec.Paths {
		// Add mcp- prefix to paths to avoid conflicts
		prefixedPath := "/mcp" + path
		combined.Paths[prefixedPath] = methods
	}

	// Merge definitions
	if combined.Definitions == nil {
		combined.Definitions = make(map[string]interface{})
	}
	for defName, def := range mcpToolsSpec.Definitions {
		// Prefix definition names to avoid conflicts
		prefixedName := "MCP_" + defName
		combined.Definitions[prefixedName] = def
	}

	// Merge tags
	mcpTag := map[string]interface{}{
		"name":        "MCP Tools",
		"description": "Model Context Protocol tools (running on port 8091)",
	}
	combined.Tags = append(combined.Tags, mcpTag)

	// Add mcp-tools tags with prefix
	for _, tag := range mcpToolsSpec.Tags {
		if tagName, ok := tag["name"].(string); ok {
			tag["name"] = "MCP: " + tagName
		}
		combined.Tags = append(combined.Tags, tag)
	}

	// Marshal combined spec
	combinedData, err := json.MarshalIndent(combined, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling combined swagger: %v\n", err)
		os.Exit(1)
	}

	// Write output
	if err := writeOutput(*outputPath, combinedData); err != nil {
		fmt.Printf("Error writing output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Combined swagger created at: %s\n", *outputPath)
	fmt.Printf("  - LLM API paths: %d\n", len(llmAPISpec.Paths))
	fmt.Printf("  - MCP Tools paths: %d (prefixed with /mcp)\n", len(mcpToolsSpec.Paths))
	fmt.Printf("  - Total paths: %d\n", len(combined.Paths))
}

func writeOutput(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return ioutil.WriteFile(path, data, 0644)
}
